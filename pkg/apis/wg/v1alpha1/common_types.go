package v1alpha1

import (
	"io/ioutil"
	"net"
	"strings"

	"github.com/mdlayher/wireguardctrl/wgtypes"
	"github.com/nmiculinic/wg-quick-go"
)

type VPNNode interface {
	ToPeerConfig() (wgtypes.PeerConfig, error)
	ToInterfaceConfig(privateKeyFile string) (*wgquick.Config, error)
	isNode()
}

type CommonSpec struct {
	PublicKey       string   `json:"publicKey"`
	Address         string   `json:"address"`
	ExtraAllowedIPs []string `json:"extraAllowedIPs"`
}

func (common *CommonSpec) toPeerConfig() (wgtypes.PeerConfig, error) {
	srvKey, err := wgquick.ParseKey(common.PublicKey)
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

func (common *CommonSpec) toInterfaceConfig(privateKeyFile string) (*wgquick.Config, error) {
	pkey, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, err
	}
	key, err := wgquick.ParseKey(string(pkey))
	if err != nil {
		return nil, err
	}

	var ip *net.IPNet
	if strings.Contains(common.Address, "/") {
		_, ip, err = net.ParseCIDR(common.Address)
		if err != nil {
			return nil, err
		}
	} else {
		_, ip, err = net.ParseCIDR(common.Address + "/32")
		if err != nil {
			return nil, err
		}
	}
	cfg := wgquick.Config{
		Address: []*net.IPNet{ip},
		Config: wgtypes.Config{
			PrivateKey:   &key,
			ReplacePeers: true,
		},
	}
	return &cfg, nil
}
