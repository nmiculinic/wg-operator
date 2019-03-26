package server

import (
	"context"
	"fmt"
	wgv1alpha1 "github.com/KrakenSystems/wg-operator/pkg/apis/wg/v1alpha1"
	"github.com/KrakenSystems/wg-operator/pkg/wgctl"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_server")

// Add creates a new Client Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, setup *wgctl.WireguardSetup) error {
	r := &ReconcileClient{
		client:  mgr.GetClient(),
		scheme:  mgr.GetScheme(),
		wgSetup: setup,
	}

	// Create a new controller
	c, err := controller.New("server-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &wgv1alpha1.Client{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &wgv1alpha1.Server{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileClient{}

// ReconcileClient reconciles a Client object
type ReconcileClient struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client  client.Client
	scheme  *runtime.Scheme
	wgSetup *wgctl.WireguardSetup
}

func (r *ReconcileClient) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Client")
	ctx := context.Background()

	me := &wgv1alpha1.Server{}
	if err := r.client.Get(ctx, client.ObjectKey{Name: r.wgSetup.NodeName, Namespace: request.Namespace}, me); err != nil {
		log.Error(err, "cannot find myself as server", "Namespace", request.Namespace, "Name", r.wgSetup.NodeName)
		return reconcile.Result{}, err
	}

	clients := &wgv1alpha1.ClientList{}
	if err := r.client.List(ctx, &client.ListOptions{Namespace: request.Namespace}, clients); err != nil {
		log.Error(err, "cannot list all client", "Request.Namespace", request.Namespace)
		return reconcile.Result{}, err
	}

	servers := &wgv1alpha1.ServerList{}
	if err := r.client.List(ctx, &client.ListOptions{Namespace: request.Namespace}, servers); err != nil {
		log.Error(err, "cannot list all servers", "Request.Namespace", request.Namespace)
		return reconcile.Result{}, err
	}

	cfg, err := wgctl.CreateServerConfig(wgctl.ServerRequest{
		PrivateKey: "AAAA",
		Me:         *me,
		Clients:    *clients,
		Servers:    *servers,
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info(fmt.Sprintf("about to apply config:\n%s", cfg.String()), "interface", r.wgSetup.InterfaceName)
	if err := r.wgSetup.SetPrivateKey(cfg); err != nil {
		return reconcile.Result{}, err
	}

	if err := cfg.Sync(r.wgSetup.InterfaceName, logrus.WithField("mode", "server")); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
