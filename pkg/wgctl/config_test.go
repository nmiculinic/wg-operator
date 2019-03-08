package wgctl

import (
	"github.com/KrakenSystems/wg-operator/pkg/apis/wg/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

type clientTestCase struct {
	req ClientRequest
	targetConfig string
	targetError error
}

var (
	cl1 = v1alpha1.Client{
	Spec: v1alpha1.ClientSpec{
		CommonSpec: v1alpha1.CommonSpec{
			PublicKey: "pub-key-cl1",
			Address: "10.100.0.1",
			ListenPort: 11,
		},
	},
	}

	cl2 = v1alpha1.Client{
		Spec: v1alpha1.ClientSpec{
			CommonSpec: v1alpha1.CommonSpec{
				PublicKey: "pub-key-cl2",
				Address: "10.100.0.2",
				ListenPort: 12,
			},
		},
	}
	server1 = v1alpha1.Server{
		Spec:v1alpha1.ServerSpec{
			CommonSpec:v1alpha1.CommonSpec{
				PublicKey: "pub-key-server1",
				Address: "10.100.1.1",
				ListenPort: 555,
			},
			Endpoint:"35.12.23.34",
		},
	}

	server2 = v1alpha1.Server{
		Spec:v1alpha1.ServerSpec{
			CommonSpec:v1alpha1.CommonSpec{
				PublicKey: "pub-key-server2",
				Address: "10.100.2.1",
				ListenPort: 123,
			},
			Endpoint:"55.12.23.34",
		},
	}
	server3 = v1alpha1.Server{
		Spec:v1alpha1.ServerSpec{
			CommonSpec:v1alpha1.CommonSpec{
				PublicKey: "pub-key-server3",
				Address: "10.100.2.1",
				ListenPort: 123,
				ExtraAllowedIPs: []string{"10.100.3.0/16"},
			},
			Endpoint:"55.12.23.34",
		},
	}

	clNoIP = v1alpha1.Client{
		Spec: v1alpha1.ClientSpec{
			CommonSpec: v1alpha1.CommonSpec{
				PublicKey: "pub-key-cl1NoIP",
				Address: "",
				ListenPort: 12,
			},
		},
	}
)

func TestClient(t *testing.T) {
	tbl := map[string]clientTestCase{
		"no-server": {
			req: ClientRequest{
				PrivateKey: "PRIVATE_KEY",
				Servers:    v1alpha1.ServerList{},
				Client:     cl1,
			},
			targetConfig:
			`[Interface]
Address = 10.100.0.1
PrivateKey = PRIVATE_KEY
ListenPort = 11
`,
		},

			"single-server": {
				req: ClientRequest{
					PrivateKey: "PRIVATE_KEY",
					Servers:v1alpha1.ServerList{Items:[]v1alpha1.Server{server1}},
					Client: cl1,
				},
				targetConfig:
				`[Interface]
Address = 10.100.0.1
PrivateKey = PRIVATE_KEY
ListenPort = 11

[Peer]
PublicKey = pub-key-server1
AllowedIps = 10.100.1.1/32
Endpoint = 35.12.23.34:555
`,
		},
		"two-servers": {
			req: ClientRequest{
				PrivateKey: "PRIVATE_KEY",
				Servers:v1alpha1.ServerList{Items:[]v1alpha1.Server{server1, server2}},
				Client: cl1,
			},
			targetConfig:
			`[Interface]
Address = 10.100.0.1
PrivateKey = PRIVATE_KEY
ListenPort = 11

[Peer]
PublicKey = pub-key-server1
AllowedIps = 10.100.1.1/32
Endpoint = 35.12.23.34:555

[Peer]
PublicKey = pub-key-server2
AllowedIps = 10.100.2.1/32
Endpoint = 55.12.23.34:123
`,
		},
		"server-extraIPs": {
			req: ClientRequest{
				PrivateKey: "PRIVATE_KEY",
				Servers:v1alpha1.ServerList{Items:[]v1alpha1.Server{server3}},
				Client: cl1,
			},
			targetConfig:
			`[Interface]
Address = 10.100.0.1
PrivateKey = PRIVATE_KEY
ListenPort = 11

[Peer]
PublicKey = pub-key-server3
AllowedIps = 10.100.2.1/32, 10.100.3.0/16
Endpoint = 55.12.23.34:123
`,
		},

	}
	for name , testCase := range tbl {
		t.Run(name, func(t *testing.T) {
			result, err := CreateClientConfig(testCase.req)
			if err != nil {
				t.Log("err", err.Error())
				assert.Equal(t, testCase.targetError.Error(), err.Error())
			}

			if testCase.targetError == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			assert.Equal(t, testCase.targetConfig, result)
		})
	}
}
