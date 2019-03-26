package wgctl

import (
	"github.com/mdlayher/wireguardctrl"
	"github.com/vishvananda/netlink"
	"io/ioutil"
	"net"
	"os/exec"
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

func (n *WireguardSetup) SyncConfigToMachine(cfg *Config) (retErr error) {
	link, err := netlink.LinkByName(n.InterfaceName)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); !ok {
			log.Error(err, "cannot read link, probably doesn't exist")
			return err
		}
		log.Info("link not found, creating")
		if err := exec.Command("ip", "link", "add", "dev", n.InterfaceName, "type", "wireguard").Run(); err != nil {
			log.Error(err, "cannot create link", "iface", n.InterfaceName)
			return err
		}

		link, err = netlink.LinkByName(n.InterfaceName)
		if err != nil {
			log.Error(err, "cannot read link")
			return err
		}
	}
	log.Info("link", "type", link.Type(), "attrs", link.Attrs())
	if err := netlink.LinkSetUp(link); err != nil {
		log.Error(err, "cannot set link up", "type", link.Type(), "attrs", link.Attrs())
		return err
	}
	log.Info("set device up", "iface", n.InterfaceName)

	if err := n.Client.ConfigureDevice(n.InterfaceName, cfg.Config); err != nil {
		log.Error(err, "cannot configure device", "iface", n.InterfaceName)
		return err
	}
	dev, err := n.Client.Device(n.InterfaceName)
	if err != nil {
		log.Error(err, "cannot get device", "iface", n.InterfaceName)
		return err
	}
	log.Info("found wg device", "iface", n.InterfaceName, "type", dev.Type)

	if err := n.syncAddress(link, cfg); err != nil {
		log.Error(err, "cannot sync addresses")
		return err
	}

	if err := n.syncRoutes(link, cfg); err != nil {
		log.Error(err, "cannot sync routes")
		return err
	}

	log.Info("Successfully setup device", "interface", n.InterfaceName, "type", dev.Type)
	return nil
}

func (n *WireguardSetup) syncAddress(link netlink.Link, cfg *Config) error {
	addrs, err := netlink.AddrList(link, syscall.AF_INET)
	if err != nil {
		log.Error(err, "cannot read link address")
		return err
	}

	presentAddresses := make(map[string]int, 0)
	for _, addr := range addrs {
		presentAddresses[addr.IPNet.String()] = 1
	}

	for _, addr := range []*net.IPNet{cfg.Address} {
		_, present := presentAddresses[addr.String()]
		presentAddresses[addr.String()] = 2
		if present {
			log.Info("address present", "addr", addr, "iface", link.Attrs().Name)
			continue
		}

		if err := netlink.AddrAdd(link, &netlink.Addr{
			IPNet: addr,
		}); err != nil {
			log.Error(err, "cannot add addr", "iface", n.InterfaceName)
			return err
		}
		log.Info("address added", "addr", addr, "iface", link.Attrs().Name)
	}

	for addr, p := range presentAddresses {
		if p < 2 {
			nlAddr, err := netlink.ParseAddr(addr)
			if err != nil {
				log.Error(err, "cannot parse del addr", "iface", n.InterfaceName, "addr", addr)
				return err
			}
			if err := netlink.AddrAdd(link, nlAddr); err != nil {
				log.Error(err, "cannot delete addr", "iface", n.InterfaceName, "addr", addr)
				return err
			}
			log.Info("address deleted", "addr", addr, "iface", link.Attrs().Name)
		}
	}
	return nil
}

func (n *WireguardSetup) syncRoutes(link netlink.Link, cfg *Config) error {
	routes, err := netlink.RouteList(link, syscall.AF_INET)
	if err != nil {
		log.Error(err, "cannot read existing routes")
		return err
	}

	presentRoutes := make(map[string]int, 0)
	for _, r := range routes {
		presentRoutes[r.Dst.String()] = 1
	}

	for _, peer := range cfg.Peers {
		for _, rt := range peer.AllowedIPs {
			_, present := presentRoutes[rt.String()]
			presentRoutes[rt.String()] = 2
			if present {
				log.Info("route present", "iface", n.InterfaceName, "route", rt.String())
				continue
			}
			if err := netlink.RouteAdd(&netlink.Route{
				LinkIndex: link.Attrs().Index,
				Dst: &rt,
			}); err != nil {
				log.Error(err, "cannot setup route", "iface", n.InterfaceName, "route", rt.String())
				return err
			}
			log.Info("route added", "iface", n.InterfaceName, "route", rt.String())
		}
	}

	// Clean extra routes
	for rtStr, p := range presentRoutes {
		_, rt, err := net.ParseCIDR(rtStr)
		if err != nil {
			log.Info("cannot parse route", "iface", n.InterfaceName, "route", rtStr)
			return err
		}
		if p < 2 {
			log.Info("extra manual route found", "iface", n.InterfaceName, "route", rt.String())
			if err := netlink.RouteDel(&netlink.Route{
				LinkIndex: link.Attrs().Index,
				Dst: rt,
			}); err != nil {
				log.Error(err, "cannot setup route", "iface", n.InterfaceName, "route", rt.String())
				return err
			}
			log.Info("route deleted", "iface", n.InterfaceName, "route", rt)
		}
	}
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
