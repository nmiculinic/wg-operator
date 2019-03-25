package v1alpha1

type CommonSpec struct {
	PublicKey       string   `json:"publicKey"`
	Address         string   `json:"address"`
	ExtraAllowedIPs []string `json:"extraAllowedIPs"`
}
