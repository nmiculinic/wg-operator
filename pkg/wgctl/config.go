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
AllowedIps = {{ .Spec.Address }}/32
Endpoint = {{ .Spec.Endpoint}}:{{ .Spec.ListenPort }}
{{- end }}
`


func CreateClientConfig(req ClientRequest) (string, error) {
	if err := req.Validate(); err != nil {
		return "", err
	}

	buff := &bytes.Buffer{}
	t := template.Must(template.New("client").Parse(clientTemplateSpec))
	if err := t.Execute(buff, req); err != nil {
		return "", err
	}
	return buff.String(), nil
}

