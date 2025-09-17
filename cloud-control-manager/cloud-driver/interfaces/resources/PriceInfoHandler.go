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

// CloudPrice represents the pricing information for a specific cloud provider.
type CloudPrice struct {
	Meta       Meta   `json:"Meta" validate:"required" description:"Metadata information about the price data"`
	CloudName  string `json:"CloudName" validate:"required" example:"AWS"`        // Name of the cloud provider
	RegionName string `json:"RegionName" validate:"required" example:"us-east-1"` // Name of the region

	PriceList []Price `json:"PriceList" validate:"required" description:"List of prices"` // List of prices for different services/products
}

// Meta contains metadata information about the price data.
type Meta struct {
	Version     string `json:"Version" validate:"required" example:"1.0"`        // Version of the pricing data
	Description string `json:"Description,omitempty" example:"Cloud price data"` // Description of the pricing data
}

// Price represents the price information for a specific product.
type Price struct {
	ZoneName    string      `json:"ZoneName,omitempty" example:"us-east-1a"`                                     // Name of the zone
	ProductInfo ProductInfo `json:"ProductInfo" validate:"required" description:"Information about the product"` // Information about the product
	PriceInfo   PriceInfo   `json:"PriceInfo" validate:"required" description:"Pricing details of the product"`  // Pricing details of the product
}

// ProductInfo represents the product details.
type ProductInfo struct {
	ProductId      string      `json:"ProductId" validate:"required" example:"prod-123"`                                      // ID of the product
	VMSpecName     string      `json:"VMSpecName,omitempty" validate:"omitempty" example:"t2.micro"`                          // Name of the VM spec (used in simple mode)
	VMSpecInfo     *VMSpecInfo `json:"VMSpecInfo,omitempty" validate:"omitempty" description:"Information about the VM spec"` // Information about the VM spec (used in detailed mode, default mode)
	Description    string      `json:"Description,omitempty" example:"General purpose instance"`                              // Description of the product
	CSPProductInfo interface{} `json:"CSPProductInfo" validate:"required" description:"Additional product info"`              // Additional product information specific to CSP
}

// PriceInfo represents the pricing details for a product.
type PriceInfo struct {
	OnDemand     OnDemand    `json:"OnDemand" validate:"required" description:"Ondemand pricing details"`  // Ondemand pricing details
	CSPPriceInfo interface{} `json:"CSPPriceInfo" validate:"required" description:"Additional price info"` // Additional price information specific to CSP
}

// OnDemand represents the OnDemand pricing details.
type OnDemand struct {
	PricingId   string `json:"PricingId" validate:"required" example:"price-123"`    // ID of the pricing policy
	Unit        string `json:"Unit" validate:"required" example:"Hour"`              // Unit of the pricing (e.g., per hour)
	Currency    string `json:"Currency" validate:"required" example:"USD"`           // Currency of the pricing
	Price       string `json:"Price" validate:"required" example:"0.02"`             // Price in the specified currency per unit
	Description string `json:"Description,omitempty" example:"Pricing for t2.micro"` // Description of the pricing policy
}

type PriceInfoHandler interface {
	ListProductFamily(regionName string) ([]string, error)
	GetPriceInfo(productFamily string, regionName string, filterList []KeyValue) (string, error) // return string: json format
}
