// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2023.11.

package resources

type CloudPriceData struct {
	Meta           Meta         `json:"meta"`
	CloudPriceList []CloudPrice `json:"cloudPriceList"`
}

type Meta struct {
	Version     string `json:"version"`
	Description string `json:"description"`
}

type CloudPrice struct {
	CloudName string      `json:"cloudName"`
	PriceList []PriceList `json:"priceList"`
}

type PriceList struct {
	ProductInfo ProductInfo `json:"productInfo"`
	PriceInfo   PriceInfo   `json:"priceInfo"`
}

type ProductInfo struct {
	ProductId  string `json:"productId"`
	RegionName string `json:"regionName"`

	//--------- Compute Instance
	InstanceType    string `json:"instanceType,omitempty"`
	Vcpu            string `json:"vcpu,omitempty"`
	Memory          string `json:"memory,omitempty"`
	Storage         string `json:"storage,omitempty"`
	Gpu             string `json:"gpu,omitempty"`
	GpuMemory       string `json:"gpuMemory,omitempty"`
	OperatingSystem string `json:"operatingSystem,omitempty"`
	PreInstalledSw  string `json:"preInstalledSw,omitempty"`
	//--------- Compute Instance

	//--------- Storage
	VolumeType          string `json:"volumeType,omitempty"`
	StorageMedia        string `json:"storageMedia,omitempty"`
	MaxVolumeSize       string `json:"maxVolumeSize,omitempty"`
	MaxIOPSVolume       string `json:"maxIopsvolume,omitempty"`
	MaxThroughputVolume string `json:"maxThroughputvolume,omitempty"`
	//--------- Storage

	Description    string      `json:"description"`
	CSPProductInfo interface{} `json:"cspProductInfo"`
}

type PriceInfo struct {
	PricingPolicies []PricingPolicies `json:"pricingPolicies"`
	CSPPriceInfo    interface{}       `json:"cspPriceInfo"`
}

type PricingPolicies struct {
	PricingId         string             `json:"pricingId"`
	PricingPolicy     string             `json:"pricingPolicy"`
	Unit              string             `json:"unit"`
	Currency          string             `json:"currency"`
	Price             string             `json:"price"`
	Description       string             `json:"description"`
	PricingPolicyInfo *PricingPolicyInfo `json:"pricingPolicyInfo,omitempty"`
}

type PricingPolicyInfo struct {
	LeaseContractLength string `json:"LeaseContractLength"`
	OfferingClass       string `json:"OfferingClass"`
	PurchaseOption      string `json:"PurchaseOption"`
}

type PriceInfoHandler interface {
	ListProductFamily(regionName string) ([]string, error)
	GetPriceInfo(productFamily string, regionName string, filterList []KeyValue) (string, error) // return string: json format
}
