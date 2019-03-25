package wgctl

import (
	"github.com/mdlayher/wireguardctrl"
	"github.com/vishvananda/netlink"
	"io/ioutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"syscall"
)

var log = logf.Log.WithName("wgctl")

type WireguardSetup struct {
	NodeName       string
	InterfaceName  string
	PrivateKeyFile string
	Client         *wireguardctrl.Client
}

func (n *WireguardSetup) SetPrivateKey(cfg *Config) error {
	f, err := ioutil.ReadFile(n.PrivateKeyFile)
	if err != nil {
		return err
	}
	key, err := parseKey(string(f))
	if err != nil {
		return err
	}
	cfg.PrivateKey = &key
	return nil
}

func (n *WireguardSetup) SyncConfigToMachine(cfg *Config) error {
	if err := n.Client.ConfigureDevice(n.InterfaceName, cfg.Config); err != nil {
		return err
	}
	dev, err := n.Client.Device(n.InterfaceName)
	if err != nil {
		return err
	}

	link, err := netlink.LinkByName(dev.Name)
	if err != nil {
		return err
	}

	addrs, err := netlink.AddrList(link, syscall.AF_INET)
	if err != nil {
		log.Error(err, "cannot read link address")
		return err
	}
	for _, addr := range addrs {
		log.Info("found address", "addr", addr, "iface", link.Attrs().Name)
	}
	addr, err := netlink.ParseAddr(cfg.Address.String())
	if err := netlink.AddrAdd(link, addr); err != nil {
		log.Error(err, "cannot setup link address")
		return err
	}

	routes, err := netlink.RouteList(link, syscall.AF_INET)
	if err != nil {
		log.Error(err, "cannot read existing routes")
		return err
	}
	for _, r := range routes {
		log.Info("found address", "route", r, "iface", link.Attrs().Name)
	}

	for _, peer := range cfg.Peers {
		for _, rt := range peer.AllowedIPs {
			route := netlink.Route{LinkIndex: link.Attrs().Index, Dst: &rt, Src: cfg.Address}
			if err := netlink.RouteAdd(&route); err != nil {
				log.Error(err, "cannot setup route", "peer", peer.PublicKey, "IPNet", rt)
				return err
			}
		}
	}
	log.Info("Successfully setup device", "interface", n.InterfaceName, "type", dev.Type)
	return nil
}

func (n *WireguardSetup) Create(ev event.CreateEvent) bool {
	return ev.Meta.GetName() == n.NodeName
}

func (n *WireguardSetup) Delete(ev event.DeleteEvent) bool {
	return ev.Meta.GetName() == n.NodeName
}

func (n *WireguardSetup) Update(ev event.UpdateEvent) bool {
	return ev.MetaOld.GetName() == n.NodeName
}

func (n *WireguardSetup) Generic(ev event.GenericEvent) bool {
	return ev.Meta.GetName() == n.NodeName
}
