package wgctl

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/KrakenSystems/wg-operator/pkg/apis/wg/v1alpha1"
	"github.com/mdlayher/wireguardctrl/wgtypes"
	"net"
	"strings"
	"text/template"
)

type Config struct {
	wgtypes.Config
	Address *net.IPNet
}

func (cfg *Config) String() string {
	b, err := cfg.MarshalText()
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (cfg *Config) MarshalText() (text []byte, err error) {
	buff := &bytes.Buffer{}
	if err := cfgTemplate.Execute(buff, cfg); err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

const wgtypeTemplateSpec = `[Interface]
Address = {{ .Address }}
PrivateKey = {{ .PrivateKey | wgKey }}
{{- if .ListenPort }}{{ "\n" }}ListenPort = {{ .ListenPort }}{{ end }}
{{- range .Peers }}

[Peer]
PublicKey = {{ .PublicKey | wgKey }}
AllowedIps = {{ range $i, $el := .AllowedIPs }}{{if $i}}, {{ end }}{{ $el }}{{ end }}
{{- if .Endpoint }}{{ "\n" }}Endpoint = {{ .Endpoint }}{{ end }}

{{- end }}
`

func serializeKey(key *wgtypes.Key) string {
	return base64.StdEncoding.EncodeToString(key[:])
}

func parseKey(key string) (wgtypes.Key, error) {
	var pkey wgtypes.Key
	pkeySlice, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return pkey, err
	}
	copy(pkey[:], pkeySlice[:])
	return pkey, nil
}

var cfgTemplate = template.Must(
	template.
		New("wg-cfg").
		Funcs(template.FuncMap(map[string]interface{}{"wgKey": serializeKey})).
		Parse(wgtypeTemplateSpec))

func commonPeerConfig(common v1alpha1.CommonSpec) (wgtypes.PeerConfig, error) {
	srvKey, err := parseKey(common.PublicKey)

	peer := wgtypes.PeerConfig{
		ReplaceAllowedIPs: true,
		PublicKey:         srvKey,
		AllowedIPs:        make([]net.IPNet, 0, 1+len(common.ExtraAllowedIPs)),
	}

	_, c, err := net.ParseCIDR(common.Address + "/32")
	if err != nil {
		return wgtypes.PeerConfig{}, err
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

func CreateClientConfig(req ClientRequest) (*Config, error) {
	key, err := parseKey(req.PrivateKey)
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
	cfg := Config{
		Address: ip,
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

func CreateServerConfig(req ServerRequest) (*Config, error) {
	key, err := parseKey(req.PrivateKey)
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

	cfg := Config{
		Address: ip,
		Config: wgtypes.Config{
			PrivateKey:   &key,
			ReplacePeers: true,
			Peers:        append(serverPeers, clientPeers...),
			ListenPort:   &ep.Port,
		},
	}
	if cfg.Address == nil {
		return nil, fmt.Errorf("cannot parse IP %v", req.Me.Spec.Address)
	}
	return &cfg, nil
}
