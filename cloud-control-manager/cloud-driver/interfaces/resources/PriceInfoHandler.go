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

// CloudPriceData represents the structure of cloud pricing data.
type CloudPriceData struct {
	Meta           Meta         `json:"meta" validate:"required" description:"Metadata information about the price data"`
	CloudPriceList []CloudPrice `json:"cloudPriceList" validate:"required" description:"List of cloud prices"`
}

// Meta contains metadata information about the price data.
type Meta struct {
	Version     string `json:"version" validate:"required" example:"1.0"`        // Version of the pricing data
	Description string `json:"description,omitempty" example:"Cloud price data"` // Description of the pricing data
}

// CloudPrice represents the pricing information for a specific cloud provider.
type CloudPrice struct {
	CloudName string  `json:"cloudName" validate:"required" example:"AWS"`                // Name of the cloud provider
	PriceList []Price `json:"priceList" validate:"required" description:"List of prices"` // List of prices for different services/products
}

// Price represents the price information for a specific product.
type Price struct {
	ProductInfo ProductInfo `json:"productInfo" validate:"required" description:"Information about the product"` // Information about the product
	PriceInfo   PriceInfo   `json:"priceInfo" validate:"required" description:"Pricing details of the product"`  // Pricing details of the product
}

// ProductInfo represents the product details.
type ProductInfo struct {
	ProductId  string `json:"productId" validate:"required" example:"prod-123"`   // ID of the product
	RegionName string `json:"regionName" validate:"required" example:"us-east-1"` // Name of the region
	ZoneName   string `json:"zoneName,omitempty" example:"us-east-1a"`            // Name of the zone

	//--------- Compute Instance Info
	InstanceType    string `json:"instanceType,omitempty" example:"t2.micro"` // Type of the instance
	Vcpu            string `json:"vcpu,omitempty" example:"2"`                // Number of vCPUs
	Memory          string `json:"memory,omitempty" example:"4096"`           // Amount of memory in MB
	Storage         string `json:"storage,omitempty" example:"100"`           // Root-disk storage size in GB
	Gpu             string `json:"gpu,omitempty" example:"1"`                 // Number of GPUs
	GpuMemory       string `json:"gpuMemory,omitempty" example:"8192"`        // GPU memory size in MB
	OperatingSystem string `json:"operatingSystem,omitempty" example:"Linux"` // Operating system type
	PreInstalledSw  string `json:"preInstalledSw,omitempty" example:"None"`   // Pre-installed software

	//--------- Storage Info  // Data-Disk(AWS:EBS)
	VolumeType          string `json:"volumeType,omitempty" example:"gp2"`          // Type of volume
	StorageMedia        string `json:"storageMedia,omitempty" example:"SSD"`        // Storage media type
	MaxVolumeSize       string `json:"maxVolumeSize,omitempty" example:"16384"`     // Maximum volume size in GB
	MaxIOPSVolume       string `json:"maxIopsvolume,omitempty" example:"3000"`      // Maximum IOPS for the volume
	MaxThroughputVolume string `json:"maxThroughputvolume,omitempty" example:"250"` // Maximum throughput for the volume in MB/s

	Description    string      `json:"description,omitempty" example:"General purpose instance"`                 // Description of the product
	CSPProductInfo interface{} `json:"cspProductInfo" validate:"required" description:"Additional product info"` // Additional product information specific to CSP
}

// PriceInfo represents the pricing details for a product.
type PriceInfo struct {
	PricingPolicies []PricingPolicies `json:"pricingPolicies" validate:"required" description:"List of pricing policies"` // List of pricing policies
	CSPPriceInfo    interface{}       `json:"cspPriceInfo" validate:"required" description:"Additional price info"`       // Additional price information specific to CSP
}

// PricingPolicies represents a single pricing policy.
type PricingPolicies struct {
	PricingId         string             `json:"pricingId" validate:"required" example:"price-123"`           // ID of the pricing policy
	PricingPolicy     string             `json:"pricingPolicy" validate:"required" example:"On-Demand"`       // Name of the pricing policy
	Unit              string             `json:"unit" validate:"required" example:"hour"`                     // Unit of the pricing (e.g., per hour)
	Currency          string             `json:"currency" validate:"required" example:"USD"`                  // Currency of the pricing
	Price             string             `json:"price" validate:"required" example:"0.02"`                    // Price in the specified currency per unit
	Description       string             `json:"description,omitempty" example:"Pricing for t2.micro"`        // Description of the pricing policy
	PricingPolicyInfo *PricingPolicyInfo `json:"pricingPolicyInfo" description:"Detailed info about pricing"` // Detail information about the pricing policy
}

// PricingPolicyInfo represents additional details about a pricing policy.
type PricingPolicyInfo struct {
	LeaseContractLength string `json:"LeaseContractLength,omitempty" example:"1 year"` // Length of the lease contract
	OfferingClass       string `json:"OfferingClass,omitempty" example:"standard"`     // Offering class (e.g., standard, convertible)
	PurchaseOption      string `json:"PurchaseOption,omitempty" example:"No Upfront"`  // Purchase option (e.g., no upfront, partial upfront)
}

type PriceInfoHandler interface {
	ListProductFamily(regionName string) ([]string, error)
	GetPriceInfo(productFamily string, regionName string, filterList []KeyValue) (string, error) // return string: json format
}
