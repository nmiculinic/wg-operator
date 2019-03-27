package wgctl

import (
	"github.com/KrakenSystems/wg-operator/pkg/apis/wg/v1alpha1"
	"github.com/mdlayher/wireguardctrl/wgtypes"
	"github.com/nmiculinic/wg-quick-go"
	"net"
	"strings"
)

func commonPeerConfig(common v1alpha1.CommonSpec) (wgtypes.PeerConfig, error) {
	srvKey, err := wgctl.ParseKey(common.PublicKey)
	peer := wgtypes.PeerConfig{
		ReplaceAllowedIPs: true,
		PublicKey:         srvKey,
		AllowedIPs:        make([]net.IPNet, 0, 1+len(common.ExtraAllowedIPs)),
	}

	_, c, err := net.ParseCIDR(common.Address + "/32")
	if err != nil {
		_, c, err = net.ParseCIDR(common.Address)
		if err != nil {
			return wgtypes.PeerConfig{}, err
		}
	}
	peer.AllowedIPs = append(peer.AllowedIPs, *c)
	for _, cidr := range common.ExtraAllowedIPs {
		_, c, err := net.ParseCIDR(cidr)
		if err != nil {
			return wgtypes.PeerConfig{}, err
		}
		peer.AllowedIPs = append(peer.AllowedIPs, *c)
	}

	return peer, nil
}

func ServerPeers(servers []v1alpha1.Server, skipServer string) ([]wgtypes.PeerConfig, error) {
	peers := make([]wgtypes.PeerConfig, 0, len(servers))
	for _, srv := range servers {
		if srv.Name == skipServer {
			continue
		}
		peer, err := commonPeerConfig(srv.Spec.CommonSpec)
		if err != nil {
			return nil, err
		}
		ep, err := net.ResolveUDPAddr("", srv.Spec.Endpoint)
		if err != nil {
			return nil, err
		}
		peer.Endpoint = ep
		peers = append(peers, peer)
	}
	return peers, nil
}

func ClientPeers(servers []v1alpha1.Client, skipClient string) ([]wgtypes.PeerConfig, error) {
	peers := make([]wgtypes.PeerConfig, 0, len(servers))
	for _, srv := range servers {
		if srv.Name == skipClient {
			continue
		}
		peer, err := commonPeerConfig(srv.Spec.CommonSpec)
		if err != nil {
			return nil, err
		}
		peers = append(peers, peer)
	}
	return peers, nil
}

type ClientRequest struct {
	PrivateKey string
	Client     v1alpha1.Client
	Servers    v1alpha1.ServerList
}

func CreateClientConfig(req ClientRequest) (*wgctl.Config, error) {
	key, err := wgctl.ParseKey(req.PrivateKey)
	if err != nil {
		return nil, err
	}

	peers, err := ServerPeers(req.Servers.Items, "")
	if err != nil {
		return nil, err
	}

	var ip *net.IPNet
	if strings.Contains(req.Client.Spec.Address, "/") {
		_, ip, err = net.ParseCIDR(req.Client.Spec.Address)
		if err != nil {
			return nil, err
		}
	} else {
		_, ip, err = net.ParseCIDR(req.Client.Spec.Address + "/32")
		if err != nil {
			return nil, err
		}
	}
	cfg := wgctl.Config{
		Address: []*net.IPNet{ip},
		Config: wgtypes.Config{
			PrivateKey:   &key,
			ReplacePeers: true,
			Peers:        peers,
		},
	}

	return &cfg, nil
}

type ServerRequest struct {
	PrivateKey string
	Me         v1alpha1.Server
	Clients    v1alpha1.ClientList
	Servers    v1alpha1.ServerList
}

func CreateServerConfig(req ServerRequest) (*wgctl.Config, error) {
	key, err := wgctl.ParseKey(req.PrivateKey)
	if err != nil {
		return nil, err
	}

	ep, err := net.ResolveUDPAddr("", req.Me.Spec.Endpoint)
	if err != nil {
		return nil, err
	}

	serverPeers, err := ServerPeers(req.Servers.Items, req.Me.Name)
	if err != nil {
		return nil, err
	}

	clientPeers, err := ClientPeers(req.Clients.Items, "")
	if err != nil {
		return nil, err
	}

	var ip *net.IPNet
	if strings.Contains(req.Me.Spec.Address, "/") {
		_, ip, err = net.ParseCIDR(req.Me.Spec.Address)
		if err != nil {
			return nil, err
		}
	} else {
		_, ip, err = net.ParseCIDR(req.Me.Spec.Address + "/32")
		if err != nil {
			return nil, err
		}
	}

	cfg := wgctl.Config{
		Address: []*net.IPNet{ip},
		Config: wgtypes.Config{
			PrivateKey:   &key,
			ReplacePeers: true,
			Peers:        append(serverPeers, clientPeers...),
			ListenPort:   &ep.Port,
		},
	}
	return &cfg, nil
}
