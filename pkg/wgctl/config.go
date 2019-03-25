package wgctl

import (
	"bytes"
	"github.com/Masterminds/sprig"
	"encoding/base64"
	"fmt"
	"github.com/KrakenSystems/wg-operator/pkg/apis/wg/v1alpha1"
	"github.com/mdlayher/wireguardctrl/wgtypes"
	"net"
	"text/template"
)

type Config struct {
	wgtypes.Config
	Address net.IP
}

// Address = {{ .Client.Spec.Address }}
const wgtypeTemplateSpec =
	`[Interface]
Address = {{ .Address }}
PrivateKey = {{ .PrivateKey | wgKey }}
{{- if .ListenPort }}ListenPort = {{ .ListenPort }}{{ end }}
{{- range .Peers }}

[Peer]
PublicKey = {{ .PublicKey | wgKey }}
AllowedIps = {{ range $i, $el := .AllowedIPs }}{{if $i}}, {{ end }}{{ $el }}{{ end }}
Endpoint = {{ .Endpoint }}
{{- end }}
`

func serializeKey(key *wgtypes.Key) string {
	return base64.StdEncoding.EncodeToString(key[:])
}

var cfgTemplate = template.Must(
	template.
		New("wg-cfg").
		Funcs(sprig.HermeticTxtFuncMap()).
		Funcs(template.FuncMap(map[string]interface{}{"wgKey": serializeKey})).
		Parse(wgtypeTemplateSpec))


func SerializeConfig(cfg *Config) (string, error) {
	buff := &bytes.Buffer{}
	if err := cfgTemplate.Execute(buff, cfg); err != nil {
		return "", err
	}
	return buff.String(), nil
}

type ClientRequest struct {
	PrivateKey string
	Client v1alpha1.Client
	Servers v1alpha1.ServerList
}

func (c *ClientRequest) Validate() error {
	if net.ParseIP(c.Client.Spec.Address) == nil {
		return fmt.Errorf("invalid client ip")
	}
	for _, srv := range c.Servers.Items {
		for _, addr := range srv.Spec.ExtraAllowedIPs {
			if _, _, err := net.ParseCIDR(addr); err != nil {
				return fmt.Errorf("cannot parse CIDR %s for server %s", addr, srv.Name)
			}
		}
	}
	return nil
}



func CreateClientConfig(req ClientRequest) (*Config, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	pkeySlice, err := base64.StdEncoding.DecodeString(req.PrivateKey)
	if err != nil {
		return nil, err
	}

	var pkey wgtypes.Key
	copy(pkey[:], pkeySlice[:])

	cfg := Config{
		Address: net.ParseIP(req.Client.Spec.Address),
		Config: wgtypes.Config{
			PrivateKey: &pkey,
			ReplacePeers:true,
			Peers:make([]wgtypes.PeerConfig, len(req.Servers.Items)),
		},
	}
	if cfg.Address == nil {
		return nil, fmt.Errorf("cannot parse IP %v", req.Client.Spec.Address)
	}

	for i, srv := range req.Servers.Items {
		pkeySlice, err := base64.StdEncoding.DecodeString(srv.Spec.PublicKey)
		if err != nil {
			return nil, err
		}
		var srvKey wgtypes.Key
		copy(srvKey[:], pkeySlice[:])

		ep, err := net.ResolveUDPAddr("", srv.Spec.Endpoint)
		if err != nil {
			return nil, err
		}

		cfg.Peers[i] = wgtypes.PeerConfig{
			ReplaceAllowedIPs:true,
			PublicKey: srvKey,
			Endpoint: ep,
			AllowedIPs:make([]net.IPNet, 0, 1 + len(srv.Spec.ExtraAllowedIPs)),
		}

		_, c, err := net.ParseCIDR(srv.Spec.Address + "/32")
		if err != nil {
			return nil, err
		}
		cfg.Peers[i].AllowedIPs = append(cfg.Peers[i].AllowedIPs, *c)
		for _, cidr := range srv.Spec.ExtraAllowedIPs {
			_, c, err := net.ParseCIDR(cidr)
			if err != nil {
				return nil, err
			}
			cfg.Peers[i].AllowedIPs = append(cfg.Peers[i].AllowedIPs, *c)
		}

	}
	return &cfg, nil
}

type ServerRequest struct {
	PrivateKey string
	Me v1alpha1.Server
	Clients v1alpha1.ClientList
	Servers v1alpha1.ServerList
}

func (c *ServerRequest) Validate() error {
	if net.ParseIP(c.Me.Spec.Address) == nil {
		return fmt.Errorf("invalid client ip")
	}
	for _, srv := range c.Servers.Items {
		for _, addr := range srv.Spec.ExtraAllowedIPs {
			if _, _, err := net.ParseCIDR(addr); err != nil {
				return fmt.Errorf("cannot parse CIDR %s for server %s", addr, srv.Name)
			}
		}
		if srv.Spec.Address == c.Me.Spec.Address {
			return fmt.Errorf("me: [%s] has same address as server %s -> %s", c.Me.Name, srv.Name, srv.Spec.Address)
		}
	}

	for _, client := range c.Clients.Items {
		for _, addr := range client.Spec.ExtraAllowedIPs {
			if _, _, err := net.ParseCIDR(addr); err != nil {
				return fmt.Errorf("cannot parse CIDR %s for client %s", addr, client.Name)
			}
		}
	}
	return nil
}

const serverTemplateSpec =
	`[Interface]
Address = {{ .Me.Spec.Address }}
PrivateKey = {{ .PrivateKey }}
ListenPort = {{ .Me.Spec.ListenPort }}
{{- range .Servers.Items }}

[Peer]
PublicKey = {{ .Spec.PublicKey }}
AllowedIps = {{ .Spec.Address }}/32 {{- range .Spec.ExtraAllowedIPs }}, {{ . }}{{- end }}
Endpoint = {{ .Spec.Endpoint}}:{{ .Spec.ListenPort }}
{{- end }}
{{- range .Clients.Items }}

[Peer]
PublicKey = {{ .Spec.PublicKey }}
AllowedIps = {{ .Spec.Address }}/32 {{- range .Spec.ExtraAllowedIPs }}, {{ . }}{{- end }}
{{- end }}
`
var serverTemplate = template.Must(template.New("server").Parse(serverTemplateSpec))

func CreateServerConfig(req ServerRequest) (string, error) {
	if err := req.Validate(); err != nil {
		return "", err
	}

	buff := &bytes.Buffer{}
	if err := serverTemplate.Execute(buff, req); err != nil {
		return "", err
	}
	return buff.String(), nil
}

