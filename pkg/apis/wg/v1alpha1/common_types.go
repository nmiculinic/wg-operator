package v1alpha1

import (
	"fmt"
	"io/ioutil"
	"net"
	"strings"

	"github.com/mdlayher/wireguardctrl/wgtypes"
	"github.com/nmiculinic/wg-quick-go"
)

type VPNNode interface {
	ToPeerConfig() (wgtypes.PeerConfig, error)
	ToInterfaceConfig(privateKeyFile string) (*wgquick.Config, error)
	NodeName() string
	isNode()
}

type CommonSpec struct {
	PublicKey string   `json:"publicKey"`
	Addresses []string `json:"addresses"`
	DNS       []string `json:"dns,omitempty"`
	// Each Address/32 is appended to allowedIPs
	AllowedIPs []string `json:"allowedIPs"`
	PreUp      string   `json:"preUp,omitempty"`
	PostUp     string   `json:"postUp,omitempty"`
	PreDown    string   `json:"preDown,omitempty"`
	PostDown   string   `json:"postDown,omitempty"`
	MTU        int      `json:"mtu,omitempty"`
	Table      int      `json:"table,omitempty"`
}

func parseAddress(addr string) (*net.IPNet, error) {
	if strings.Contains(addr, "/") {
		ip, cidr, err := net.ParseCIDR(addr)
		if err != nil {
			return nil, err
		}
		return &net.IPNet{IP: ip, Mask: cidr.Mask}, nil
	} else {
		return parseAddress(addr + "/32")
	}
}

func (common *CommonSpec) toPeerConfig() (wgtypes.PeerConfig, error) {
	srvKey, err := wgquick.ParseKey(common.PublicKey)
	if err != nil {
		return wgtypes.PeerConfig{}, nil
	}
	peer := wgtypes.PeerConfig{
		ReplaceAllowedIPs: true,
		PublicKey:         srvKey,
		AllowedIPs:        make([]net.IPNet, 0, len(common.AllowedIPs)),
	}

	for _, cidr := range common.AllowedIPs {
		c, err := parseAddress(cidr)
		if err != nil {
			return wgtypes.PeerConfig{}, err
		}
		peer.AllowedIPs = append(peer.AllowedIPs, net.IPNet{IP: c.IP.Mask(c.Mask), Mask: c.Mask})
	}
	return peer, nil
}

func (common *CommonSpec) toInterfaceConfig(privateKeyFile string) (*wgquick.Config, error) {
	pkey, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, err
	}
	key, err := wgquick.ParseKey(string(pkey))
	if err != nil {
		return nil, err
	}

	var addrs []net.IPNet
	for _, addr := range common.Addresses {
		a, err := parseAddress(addr)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %s: %v", addr, err)
		}
		addrs = append(addrs, *a)
	}

	cfg := wgquick.Config{
		Address: addrs,
		Config: wgtypes.Config{
			PrivateKey:   &key,
			ReplacePeers: true,
		},
		PreUp:    common.PreUp,
		PostUp:   common.PostUp,
		PreDown:  common.PreDown,
		PostDown: common.PostDown,
		MTU:      common.MTU,
		Table:    common.Table,
	}
	return &cfg, nil
}
