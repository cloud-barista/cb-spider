// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2020.01.

package resources

// VMSpecInfo represents the detailed information of a VM specification.
type VMSpecInfo struct {
	Region string    `json:"Region" validate:"required" example:"us-east-1"` // Region where the VM spec is available
	Name   string    `json:"Name" validate:"required" example:"t2.micro"`    // Name of the VM spec
	VCpu   VCpuInfo  `json:"VCpu" validate:"required"`                       // CPU details of the VM spec
	Mem    string    `json:"Mem" validate:"required" example:"1024"`         // Memory size in MB
	Disk   string    `json:"Disk" validate:"required" example:"8"`           // Disk size in GB, NA when not applicable
	Gpu    []GpuInfo `json:"Gpu,omitempty" validate:"omitempty"`             // GPU details if available

	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"` // Additional key-value pairs for the VM spec
}

// VCpuInfo represents the CPU details of a VM specification.
type VCpuInfo struct {
	Count string `json:"Count" validate:"required" example:"2"`              // Number of CPU cores, NA when not applicable
	Clock string `json:"Clock,omitempty" validate:"omitempty" example:"2.5"` // Clock speed in GHz, NA when not applicable
}

// GpuInfo represents the GPU details of a VM specification.
type GpuInfo struct {
	Count string `json:"Count" validate:"required" example:"1"`                    // Number of GPUs, NA when not applicable
	Mfr   string `json:"Mfr,omitempty" validate:"omitempty" example:"NVIDIA"`      // Manufacturer of the GPU, NA when not applicable
	Model string `json:"Model,omitempty" validate:"omitempty" example:"Tesla K80"` // Model of the GPU, NA when not applicable
	Mem   string `json:"Mem,omitempty" validate:"omitempty" example:"8192"`        // Memory size of the GPU in MB, NA when not applicable
}

type VMSpecHandler interface {
	ListVMSpec() ([]*VMSpecInfo, error)
	GetVMSpec(Name string) (VMSpecInfo, error)

	ListOrgVMSpec() (string, error)           // return string: json format
	GetOrgVMSpec(Name string) (string, error) // return string: json format
}
