package v1alpha1

import (
	"net"

	"github.com/mdlayher/wireguardctrl/wgtypes"
	"github.com/nmiculinic/wg-quick-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ServerSpec defines the desired state of Server
// +k8s:openapi-gen=true
type ServerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	CommonSpec `json:",inline"`
	Endpoint   string `json:"endpoint"`
}

var _ VPNNode = (*Server)(nil)

func (*Server) isNode() {}

func (server *Server) ToPeerConfig() (wgtypes.PeerConfig, error) {
	peer, err := server.Spec.CommonSpec.toPeerConfig()
	if err != nil {
		return wgtypes.PeerConfig{}, err
	}
	peer.Endpoint, err = net.ResolveUDPAddr("", server.Spec.Endpoint)
	if err != nil {
		return wgtypes.PeerConfig{}, err
	}
	return peer, nil
}

func (server *Server) ToInterfaceConfig(privateKeyFile string) (*wgquick.Config, error) {
	cfg, err := server.Spec.CommonSpec.toInterfaceConfig(privateKeyFile)
	if err != nil {
		return nil, err
	}
	ep, err := net.ResolveUDPAddr("", server.Spec.Endpoint)
	if err != nil {
		return nil, err
	}
	cfg.ListenPort = &ep.Port
	return cfg, nil
}

// ServerStatus defines the observed state of Server
// +k8s:openapi-gen=true
type ServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Server is the Schema for the servers API
// +k8s:openapi-gen=true
type Server struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServerSpec   `json:"spec,omitempty"`
	Status ServerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServerList contains a list of Server
type ServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Server `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Server{}, &ServerList{})
}
