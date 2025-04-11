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
	Meta           Meta         `json:"Meta" validate:"required" description:"Metadata information about the price data"`
	CloudPriceList []CloudPrice `json:"CloudPriceList" validate:"required" description:"List of cloud prices"`
}

// Meta contains metadata information about the price data.
type Meta struct {
	Version     string `json:"Version" validate:"required" example:"1.0"`        // Version of the pricing data
	Description string `json:"Description,omitempty" example:"Cloud price data"` // Description of the pricing data
}

// CloudPrice represents the pricing information for a specific cloud provider.
type CloudPrice struct {
	CloudName string  `json:"CloudName" validate:"required" example:"AWS"`                // Name of the cloud provider
	PriceList []Price `json:"PriceList" validate:"required" description:"List of prices"` // List of prices for different services/products
}

// Price represents the price information for a specific product.
type Price struct {
	ProductInfo ProductInfo `json:"ProductInfo" validate:"required" description:"Information about the product"` // Information about the product
	PriceInfo   PriceInfo   `json:"PriceInfo" validate:"required" description:"Pricing details of the product"`  // Pricing details of the product
}

// ProductInfo represents the product details.
type ProductInfo struct {
	ProductId  string `json:"ProductId" validate:"required" example:"prod-123"`   // ID of the product
	RegionName string `json:"RegionName" validate:"required" example:"us-east-1"` // Name of the region
	ZoneName   string `json:"ZoneName,omitempty" example:"us-east-1a"`            // Name of the zone

	//--------- Compute Instance Info
	VMSpecInfo VMSpecInfo `json:"VMSpecInfo" validate:"required" description:"Information about the VM spec"` // Information about the VM spec

	OSDistribution string `json:"OSDistribution,omitempty" example:"Linux"` // Operating system distribution
	PreInstalledSw string `json:"PreInstalledSw,omitempty" example:"None"`  // Pre-installed software

	//--------- Storage Info  // Data-Disk(AWS:EBS)
	VolumeType          string `json:"VolumeType,omitempty" example:"gp2"`          // Type of volume
	StorageMedia        string `json:"StorageMedia,omitempty" example:"SSD"`        // Storage media type
	MaxVolumeSize       string `json:"MaxVolumeSize,omitempty" example:"16384"`     // Maximum volume size in GB
	MaxIOPSVolume       string `json:"MaxIopsvolume,omitempty" example:"3000"`      // Maximum IOPS for the volume
	MaxThroughputVolume string `json:"MaxThroughputvolume,omitempty" example:"250"` // Maximum throughput for the volume in MB/s

	Description    string      `json:"Description,omitempty" example:"General purpose instance"`                 // Description of the product
	CSPProductInfo interface{} `json:"CSPProductInfo" validate:"required" description:"Additional product info"` // Additional product information specific to CSP
}

// PriceInfo represents the pricing details for a product.
type PriceInfo struct {
	PricingPolicies []PricingPolicies `json:"PricingPolicies" validate:"required" description:"List of pricing policies"` // List of pricing policies
	CSPPriceInfo    interface{}       `json:"CSPPriceInfo" validate:"required" description:"Additional price info"`       // Additional price information specific to CSP
}

// PricingPolicies represents a single pricing policy.
type PricingPolicies struct {
	PricingId         string             `json:"PricingId" validate:"required" example:"price-123"`           // ID of the pricing policy
	PricingPolicy     string             `json:"PricingPolicy" validate:"required" example:"On-Demand"`       // Name of the pricing policy
	Unit              string             `json:"Unit" validate:"required" example:"hour"`                     // Unit of the pricing (e.g., per hour)
	Currency          string             `json:"Currency" validate:"required" example:"USD"`                  // Currency of the pricing
	Price             string             `json:"Price" validate:"required" example:"0.02"`                    // Price in the specified currency per unit
	Description       string             `json:"Description,omitempty" example:"Pricing for t2.micro"`        // Description of the pricing policy
	PricingPolicyInfo *PricingPolicyInfo `json:"PricingPolicyInfo" description:"Detailed info about pricing"` // Detail information about the pricing policy
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
