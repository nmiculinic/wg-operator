package node

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	wgv1alpha1 "github.com/KrakenSystems/wg-operator/pkg/apis/wg/v1alpha1"
	"github.com/mdlayher/wireguardctrl/wgtypes"
	"github.com/nmiculinic/wg-quick-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type Mode int

const (
	Unset  Mode = iota
	Server      = iota
	Client      = iota
)

type NodeControllerConfig struct {
	NodeName       string
	Interface      string
	PrivateKeyFile string
	Namespace      string
	RouteMetric    int
	RouteProto     int
	RouteTable     int
	Mode           Mode
	SplitServers   bool
	DryRun         bool
	SyncConfigPath string
	SyncConfig     bool
}

func (n *NodeControllerConfig) Create(ev event.CreateEvent) bool {
	return ev.Meta.GetName() == n.NodeName
}

func (n *NodeControllerConfig) Delete(ev event.DeleteEvent) bool {
	return ev.Meta.GetName() == n.NodeName
}

func (n *NodeControllerConfig) Update(ev event.UpdateEvent) bool {
	return ev.MetaOld.GetName() == n.NodeName
}

func (n *NodeControllerConfig) Generic(ev event.GenericEvent) bool {
	return ev.Meta.GetName() == n.NodeName
}

type nodeController struct {
	NodeControllerConfig
	client client.Client
	scheme *runtime.Scheme
	update chan bool
	dirty  bool
}

var _ manager.Runnable = (*nodeController)(nil)
var _ reconcile.Reconciler = (*nodeController)(nil)

func (r *nodeController) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logrus.WithField("name", request.Name).WithField("namespace", request.Namespace).Infoln("update triggered")
	r.update <- true
	return reconcile.Result{}, nil
}

func (ctl *nodeController) Start(done <-chan struct{}) error {
	log := logrus.WithContext(context.Background())
	// retry on error
	// TODO: Added exponential backoff
	// TODO: extract these times to constants
	sync := func() {
		err := ctl.refresh()
		switch err {
		case nil:
			ctl.dirty = false
			log.Infoln("successfully synced config")
		default:
			ctl.dirty = true
			log.WithError(err).Errorln("error during syncing")
		}
	}

	t := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-t.C:
			if ctl.dirty {
				sync()
			}
		case <-done:
			return nil
		case <-ctl.update:
			// coalesce update interrupts
			coalesce := time.After(200 * time.Millisecond)
		outer:
			for {
				select {
				case <-ctl.update:
				case <-coalesce:
					break outer
				}
			}
			sync()
		}
	}
}

func (r *nodeController) fetchMyself(ctx context.Context) (wgv1alpha1.VPNNode, error) {
	switch r.Mode {
	case Server:
		srvme := &wgv1alpha1.Server{}
		if err := r.client.Get(ctx, client.ObjectKey{Name: r.NodeName, Namespace: r.Namespace}, srvme); err != nil {
			return nil, errors.New("cannot find myself -- server")
		}
		return srvme, nil
	case Client:
		clientMe := &wgv1alpha1.Client{}
		if err := r.client.Get(ctx, client.ObjectKey{Name: r.NodeName, Namespace: r.Namespace}, clientMe); err != nil {
			return nil, errors.New("cannot find myself -- client")
		}
		return clientMe, nil
	default:
		return nil, errors.New("invalid mode type!")
	}
}

func (r *nodeController) syncConfig(ctx context.Context, cfg *wgquick.Config, iface string, log logrus.FieldLogger) error {
	log = log.WithField("iface", iface)
	pub := cfg.PrivateKey.PublicKey()
	log.Info("read private key", "public key", base64.StdEncoding.EncodeToString(pub[:]))

	// set dummy key and log the config with fake private key
	privKey := cfg.PrivateKey
	dummyKey := wgtypes.Key([32]byte{0})
	cfg.PrivateKey = &dummyKey
	log.Infof("about to apply config:\n%s", cfg.String())
	cfg.PrivateKey = privKey
	if r.DryRun {
		log.Info("Dry run, not applying config!")
		return nil
	}
	if err := wgquick.Sync(cfg, iface, log); err != nil {
		return err
	}
	if r.SyncConfig {
		m, err := cfg.MarshalText()
		if err != nil {
			return fmt.Errorf("cannot marshal config: %v", err)
		}
		pp := path.Join(r.SyncConfigPath, iface+".conf")
		if err := ioutil.WriteFile(pp, m, 0600); err != nil {
			return fmt.Errorf("cannot write config to %s: %v", pp, err)
		}
		log.Infoln("Synced config to disk")
	}
	return nil
}

func (r *nodeController) refresh() error {
	ctx := context.Background()
	log := logrus.WithField("iface", r.Interface)

	me, err := r.fetchMyself(ctx)
	if err != nil {
		return err
	}

	cfg, err := me.ToInterfaceConfig(r.PrivateKeyFile)
	if err != nil {
		return fmt.Errorf("cannot create interface config: %v", err)
	}
	cfg.Table = r.RouteTable
	cfg.RouteProtocol = r.RouteProto
	cfg.RouteMetric = r.RouteMetric

	servers := &wgv1alpha1.ServerList{}
	if err := r.client.List(ctx, &client.ListOptions{Namespace: r.Namespace}, servers); err != nil {
		log.Error(err, "cannot list all servers")
		return err
	}

	for _, srv := range servers.Items {
		if srv.Name == me.NodeName() {
			continue
		}
		peer, err := srv.ToPeerConfig()
		if err != nil {
			return fmt.Errorf("cannot generate peer config for server %s: %v", srv.Name, err)
		}
		cfg.Peers = append(cfg.Peers, peer)

		if r.SplitServers {
			c := cfg
			oldPeers := c.Peers
			c.Peers = []wgtypes.PeerConfig{peer}
			c.ListenPort = nil
			if err := r.syncConfig(ctx, c, r.Interface+"-"+srv.Name, log); err != nil {
				return fmt.Errorf("cannot sync server %s: %v", srv.Name, err)
			}
			c.Peers = oldPeers
		}
	}

	if r.Mode == Server {
		clients := &wgv1alpha1.ClientList{}
		if err := r.client.List(ctx, &client.ListOptions{Namespace: r.Namespace}, clients); err != nil {
			log.Error(err, "cannot list all client")
		}
		for _, cl := range clients.Items {
			if cl.Name == (me).(*wgv1alpha1.Server).Name {
				continue
			}
			peer, err := cl.ToPeerConfig()
			if err != nil {
				return fmt.Errorf("cannot generate peer config for client %s: %v", cl.Name, err)
			}
			cfg.Peers = append(cfg.Peers, peer)
		}
	}

	return r.syncConfig(ctx, cfg, r.Interface, log)
}

// Add creates a new Client Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, config NodeControllerConfig) error {
	r := &nodeController{
		client:               mgr.GetClient(),
		scheme:               mgr.GetScheme(),
		update:               make(chan bool, 100),
		NodeControllerConfig: config,
	}

	if err := mgr.Add(r); err != nil {
		return err
	}

	// Create a new controller
	c, err := controller.New("server-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	switch config.Mode {
	case Client:
		err = c.Watch(&source.Kind{Type: &wgv1alpha1.Client{}}, &handler.EnqueueRequestForObject{}, &config)
	case Server:
		err = c.Watch(&source.Kind{Type: &wgv1alpha1.Client{}}, &handler.EnqueueRequestForObject{})
	default:
		return fmt.Errorf("unknown mode %d", config.Mode)
	}

	err = c.Watch(&source.Kind{Type: &wgv1alpha1.Server{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}
