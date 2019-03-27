package v1alpha1

import (
	"github.com/mdlayher/wireguardctrl/wgtypes"
	"github.com/nmiculinic/wg-quick-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClientSpec defines the desired state of Client
// +k8s:openapi-gen=true
type ClientSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	CommonSpec `json:",inline"`
}

var _ VPNNode = (*ClientSpec)(nil)

func (*ClientSpec) isNode() {}

func (client *ClientSpec) ToPeerConfig() (wgtypes.PeerConfig, error) {
	return client.CommonSpec.toPeerConfig()
}

func (client *ClientSpec) ToInterfaceConfig(privateKeyFile string) (*wgquick.Config, error) {
	return client.CommonSpec.toInterfaceConfig(privateKeyFile)
}

// ClientStatus defines the observed state of Client
// +k8s:openapi-gen=true
type ClientStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Client is the Schema for the clients API
// +k8s:openapi-gen=true
type Client struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClientSpec   `json:"spec,omitempty"`
	Status ClientStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClientList contains a list of Client
type ClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Client `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Client{}, &ClientList{})
}
