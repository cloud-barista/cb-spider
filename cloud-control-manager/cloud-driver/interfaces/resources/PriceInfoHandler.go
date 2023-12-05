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
	InstanceType    string `json:"instanceType"`
	Vcpu            string `json:"vcpu"`
	Memory          string `json:"memory"`
	Storage         string `json:"storage"`
	Gpu             string `json:"gpu"`
	GpuMemory       string `json:"gpuMemory"`
	OperatingSystem string `json:"operatingSystem"`
	PreInstalledSw  string `json:"preInstalledSw"`
	//--------- Compute Instance

	//--------- Storage
	VolumeType          string `json:"volumeType"`
	StorageMedia        string `json:"storageMedia"`
	MaxVolumeSize       string `json:"maxVolumeSize"`
	MaxIOPSVolume       string `json:"maxIopsvolume"`
	MaxThroughputVolume string `json:"maxThroughputvolume"`
	//--------- Storage

	Description    string `json:"description"`
	CSPProductInfo string `json:"cspProductInfo"`
}

type PriceInfo struct {
	PricingPolicies []PricingPolicies `json:"pricingPolicies"`
	CSPPriceInfo    string            `json:"cspPriceInfo"`
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
