package wgctl

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type WireguardSetup struct {
	NodeName       string
	InterfaceName  string
	PrivateKeyFile string
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
