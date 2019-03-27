package wgctl

import (
	"github.com/nmiculinic/wg-quick-go"
	"io/ioutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type WireguardSetup struct {
	NodeName       string
	InterfaceName  string
	PrivateKeyFile string
}

func (n *WireguardSetup) SetPrivateKey(cfg *wgctl.Config) error {
	f, err := ioutil.ReadFile(n.PrivateKeyFile)
	if err != nil {
		return err
	}
	key, err := wgctl.ParseKey(string(f))
	if err != nil {
		return err
	}
	cfg.PrivateKey = &key
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
