package wgctl

import (
	"fmt"
	"github.com/mdlayher/wireguardctrl"
	"github.com/vishvananda/netlink"
	"io/ioutil"
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
	if err := exec.Command("ip", "link", "set", n.InterfaceName, "up").Run(); err != nil {
		log.Error(err, "cannot up link", "iface", n.InterfaceName)
		return err
	}
	log.Info("set device up", "iface", n.InterfaceName)

	addrs, err := netlink.AddrList(link, syscall.AF_INET)
	if err != nil {
		log.Error(err, "cannot read link address")
		return err
	}

	presentAddresses := make(map[string]int, 0)
	for _, addr := range addrs {
		presentAddresses[addr.IP.String()] = 1
	}

	for _, addr := range []string{cfg.Address.String()} {
		_, present := presentAddresses[addr]
		presentAddresses[cfg.Address.String()] = 2
		if present {
			log.Info("address present", "addr", addr, "iface", link.Attrs().Name)
			continue
		}

		cmd := exec.Command("ip", "addr", "add", "dev", n.InterfaceName, addr+"/32")
		if stdoutStderr, err := cmd.CombinedOutput(); err != nil {
			log.Error(err, "cannot delete addr", "iface", n.InterfaceName, "args", fmt.Sprint(cmd.Args), "output", string(stdoutStderr))
			return err
		}
		log.Info("address added", "addr", addr, "iface", link.Attrs().Name)
	}

	for addr, p := range presentAddresses {
		if p < 2 {
			log.Info("extra manual address found", "iface", n.InterfaceName, "addr", addr)
			cmd := exec.Command("ip", "addr", "del", "dev", n.InterfaceName, addr)
			if stdoutStderr, err := cmd.CombinedOutput(); err != nil {
				log.Error(err, "cannot delete addr", "iface", n.InterfaceName, "args", fmt.Sprint(cmd.Args), "output", string(stdoutStderr))
				return err
			}
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
			cmd := exec.Command("ip", "route", "add", rt.String(), "dev", n.InterfaceName)
			if stdoutStderr, err := cmd.CombinedOutput(); err != nil {
				log.Error(err, "cannot setup route", "iface", n.InterfaceName, "args", fmt.Sprint(cmd.Args), "output", string(stdoutStderr))
				return err
			}
			log.Info("route added", "iface", n.InterfaceName, "route", rt.String())
		}
	}

	// Clean extra routes
	for rt, p := range presentRoutes {
		if p < 2 {
			log.Info("extra manual route found", "iface", n.InterfaceName, "route", rt)
			cmd := exec.Command("ip", "route", "del", rt, "dev", n.InterfaceName)
			if stdoutStderr, err := cmd.CombinedOutput(); err != nil {
				log.Error(err, "cannot delete route", "iface", n.InterfaceName, "args", fmt.Sprint(cmd.Args), "output", string(stdoutStderr))
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
