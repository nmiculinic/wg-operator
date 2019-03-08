package wgctl

import (
	"bytes"
	"fmt"
	"github.com/KrakenSystems/wg-operator/pkg/apis/wg/v1alpha1"
	"net"
	"text/template"
)

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

const clientTemplateSpec =
`[Interface]
Address = {{ .Client.Spec.Address }}
PrivateKey = {{ .PrivateKey }}
ListenPort = {{ .Client.Spec.ListenPort }}
{{- range .Servers.Items }}

[Peer]
PublicKey = {{ .Spec.PublicKey }}
AllowedIps = {{ .Spec.Address }}/32 {{- range .Spec.ExtraAllowedIPs }}, {{ . }}{{- end }}
Endpoint = {{ .Spec.Endpoint}}:{{ .Spec.ListenPort }}
{{- end }}
`
var clientTemplate = template.Must(template.New("client").Parse(clientTemplateSpec))


func CreateClientConfig(req ClientRequest) (string, error) {
	if err := req.Validate(); err != nil {
		return "", err
	}

	buff := &bytes.Buffer{}
	if err := clientTemplate.Execute(buff, req); err != nil {
		return "", err
	}
	return buff.String(), nil
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

