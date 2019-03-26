package wgctl

import (
	"github.com/KrakenSystems/wg-operator/pkg/apis/wg/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type clientTestCase struct {
	req          ClientRequest
	targetConfig string
	targetError  error
}

var (
	cl1 = v1alpha1.Client{
		ObjectMeta: v1.ObjectMeta{Name: "client-cl1"},
		Spec: v1alpha1.ClientSpec{
			CommonSpec: v1alpha1.CommonSpec{
				PublicKey: "rh3vMGPqe6UhQwly7kZKYAvG4tJa7+j5lOPICXI/1kI=",
				Address:   "10.100.0.1",
			},
		},
	}

	cl2 = v1alpha1.Client{
		ObjectMeta: v1.ObjectMeta{Name: "client-cl2"},
		Spec: v1alpha1.ClientSpec{
			CommonSpec: v1alpha1.CommonSpec{
				PublicKey: "VssQj+7Qxa4e8Ar9i2lr9hs1U6SagOf3kTLP+Mj4HVA=",
				Address:   "10.100.0.2",
			},
		},
	}
	server1 = v1alpha1.Server{
		ObjectMeta: v1.ObjectMeta{Name: "server1"},
		Spec: v1alpha1.ServerSpec{
			CommonSpec: v1alpha1.CommonSpec{
				PublicKey: "qlgnbDFeqmA/qbxbtol4mYB0Eq/rDNfJA7Wg97mJ2Vs=",
				Address:   "10.100.1.1",
			},
			Endpoint: "35.12.23.34:555",
		},
	}

	server2 = v1alpha1.Server{
		ObjectMeta: v1.ObjectMeta{Name: "server2"},
		Spec: v1alpha1.ServerSpec{
			CommonSpec: v1alpha1.CommonSpec{
				PublicKey: "W+4V89h8brHTGM2/iqNItkuRWuCVYfBrl4lKGjW/zCg=",
				Address:   "10.100.2.1",
			},
			Endpoint: "55.12.23.34:123",
		},
	}
	server3 = v1alpha1.Server{
		ObjectMeta: v1.ObjectMeta{Name: "server3"},
		Spec: v1alpha1.ServerSpec{
			CommonSpec: v1alpha1.CommonSpec{
				PublicKey:       "NsLCXWIocmA3c9hbktxigwLdqA3x56QTVV9v9R/Wym4=",
				Address:         "10.100.2.1",
				ExtraAllowedIPs: []string{"10.100.3.0/16"},
			},
			Endpoint: "55.12.23.34:123",
		},
	}

	clNoIP = v1alpha1.Client{
		ObjectMeta: v1.ObjectMeta{Name: "client-cl-no-ip"},
		Spec: v1alpha1.ClientSpec{
			CommonSpec: v1alpha1.CommonSpec{
				PublicKey: "RZbjqfNqvnl14BwxPEvjeFMZje2TUfg6POL93Okr3DQ=",
				Address:   "",
			},
		},
	}
)

func TestClient(t *testing.T) {
	tbl := map[string]clientTestCase{
		"no-server": {
			req: ClientRequest{
				PrivateKey: "QBNloaEPjZd/nafQcH55kdYqnQ6YB6gX35l//QGra2E=",
				Servers:    v1alpha1.ServerList{},
				Client:     cl1,
			},
			targetConfig: `[Interface]
Address = 10.100.0.1/32
PrivateKey = QBNloaEPjZd/nafQcH55kdYqnQ6YB6gX35l//QGra2E=
`,
		},

		"single-server": {
			req: ClientRequest{
				PrivateKey: "QBNloaEPjZd/nafQcH55kdYqnQ6YB6gX35l//QGra2E=",
				Servers:    v1alpha1.ServerList{Items: []v1alpha1.Server{server1}},
				Client:     cl1,
			},
			targetConfig: `[Interface]
Address = 10.100.0.1/32
PrivateKey = QBNloaEPjZd/nafQcH55kdYqnQ6YB6gX35l//QGra2E=

[Peer]
PublicKey = qlgnbDFeqmA/qbxbtol4mYB0Eq/rDNfJA7Wg97mJ2Vs=
AllowedIps = 10.100.1.1/32
Endpoint = 35.12.23.34:555
`,
		},
		"two-servers": {
			req: ClientRequest{
				PrivateKey: "QBNloaEPjZd/nafQcH55kdYqnQ6YB6gX35l//QGra2E=",
				Servers:    v1alpha1.ServerList{Items: []v1alpha1.Server{server1, server2}},
				Client:     cl1,
			},
			targetConfig: `[Interface]
Address = 10.100.0.1/32
PrivateKey = QBNloaEPjZd/nafQcH55kdYqnQ6YB6gX35l//QGra2E=

[Peer]
PublicKey = qlgnbDFeqmA/qbxbtol4mYB0Eq/rDNfJA7Wg97mJ2Vs=
AllowedIps = 10.100.1.1/32
Endpoint = 35.12.23.34:555

[Peer]
PublicKey = W+4V89h8brHTGM2/iqNItkuRWuCVYfBrl4lKGjW/zCg=
AllowedIps = 10.100.2.1/32
Endpoint = 55.12.23.34:123
`,
		},
		"server-extraIPs": {
			req: ClientRequest{
				PrivateKey: "QBNloaEPjZd/nafQcH55kdYqnQ6YB6gX35l//QGra2E=",
				Servers:    v1alpha1.ServerList{Items: []v1alpha1.Server{server3}},
				Client:     cl1,
			},
			targetConfig: `[Interface]
Address = 10.100.0.1/32
PrivateKey = QBNloaEPjZd/nafQcH55kdYqnQ6YB6gX35l//QGra2E=

[Peer]
PublicKey = NsLCXWIocmA3c9hbktxigwLdqA3x56QTVV9v9R/Wym4=
AllowedIps = 10.100.2.1/32, 10.100.0.0/16
Endpoint = 55.12.23.34:123
`,
		},
	}
	for name, testCase := range tbl {
		t.Run(name, func(t *testing.T) {
			result, err := CreateClientConfig(testCase.req)
			t.Log(result)
			if err != nil {
				t.Log("err", err.Error())
				if testCase.targetError != nil {
					assert.Equal(t, testCase.targetError.Error(), err.Error())
				} else {
					t.Fail()
				}
			}

			if testCase.targetError == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			cfg, err := result.MarshalText()
			t.Logf("Serialized config:\n%v", string(cfg))
			assert.NoError(t, err)
			assert.Equal(t, testCase.targetConfig, string(cfg))
		})
	}
}

type serverTestCase struct {
	req          ServerRequest
	targetConfig string
	targetError  error
}

func TestServer(t *testing.T) {
	tbl := map[string]serverTestCase{
		"no-server": {
			req: ServerRequest{
				Me:         server3,
				PrivateKey: "QBNloaEPjZd/nafQcH55kdYqnQ6YB6gX35l//QGra2E=",
				Servers:    v1alpha1.ServerList{Items: []v1alpha1.Server{server3}},
				Clients:    v1alpha1.ClientList{},
			},
			targetConfig: `[Interface]
Address = 10.100.2.1/32
PrivateKey = QBNloaEPjZd/nafQcH55kdYqnQ6YB6gX35l//QGra2E=
ListenPort = 123
`,
			targetError: nil,
		},
		"2server-2clients": {
			req: ServerRequest{
				Me:         server3,
				PrivateKey: "QBNloaEPjZd/nafQcH55kdYqnQ6YB6gX35l//QGra2E=",
				Servers:    v1alpha1.ServerList{Items: []v1alpha1.Server{server1}},
				Clients:    v1alpha1.ClientList{Items: []v1alpha1.Client{cl1, cl2}},
			},
			targetConfig: `[Interface]
Address = 10.100.2.1/32
PrivateKey = QBNloaEPjZd/nafQcH55kdYqnQ6YB6gX35l//QGra2E=
ListenPort = 123

[Peer]
PublicKey = qlgnbDFeqmA/qbxbtol4mYB0Eq/rDNfJA7Wg97mJ2Vs=
AllowedIps = 10.100.1.1/32
Endpoint = 35.12.23.34:555

[Peer]
PublicKey = rh3vMGPqe6UhQwly7kZKYAvG4tJa7+j5lOPICXI/1kI=
AllowedIps = 10.100.0.1/32

[Peer]
PublicKey = VssQj+7Qxa4e8Ar9i2lr9hs1U6SagOf3kTLP+Mj4HVA=
AllowedIps = 10.100.0.2/32
`,
		},
	}
	for name, testCase := range tbl {
		t.Run(name, func(t *testing.T) {
			result, err := CreateServerConfig(testCase.req)
			t.Log(result)
			if err != nil {
				t.Log("err", err.Error())
				if testCase.targetError != nil {
					assert.Equal(t, testCase.targetError.Error(), err.Error())
				} else {
					t.Fail()
				}
			}

			if testCase.targetError == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			cfg, err := result.MarshalText()
			t.Logf("Serialized config:\n%v", string(cfg))
			assert.NoError(t, err)
			assert.Equal(t, testCase.targetConfig, string(cfg))
		})
	}
}
