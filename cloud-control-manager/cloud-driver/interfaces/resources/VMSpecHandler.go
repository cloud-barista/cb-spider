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
	Region     string    `json:"Region" validate:"required" example:"us-east-1"` // Region where the VM spec is available
	Name       string    `json:"Name" validate:"required" example:"t2.micro"`    // Name of the VM spec
	VCpu       VCpuInfo  `json:"VCpu" validate:"required"`                       // CPU details of the VM spec
	MemSizeMiB string    `json:"MemSizeMiB" validate:"required" example:"1024"`  // Memory size in MiB
	DiskSizeGB string    `json:"DiskSizeGB" validate:"required" example:"8"`     // Disk size in GB, "-1" when not applicable
	Gpu        []GpuInfo `json:"Gpu,omitempty" validate:"omitempty"`             // GPU details if available

	Accelerator []AcceleratorInfo `json:"Accelerator,omitempty" validate:"omitempty"` // AI accelerator(GPU/TPU/NPU) details if available

	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"` // Additional key-value pairs for the VM spec
}

// VCpuInfo represents the CPU details of a VM specification.
type VCpuInfo struct {
	Count    string `json:"Count" validate:"required" example:"2"`                 // Number of VCpu, "-1" when not applicable
	ClockGHz string `json:"ClockGHz,omitempty" validate:"omitempty" example:"2.5"` // Clock speed in GHz, "-1" when not applicable
}

// GpuInfo represents the GPU details of a VM specification.
type GpuInfo struct {
	Count          string `json:"Count" validate:"required" example:"2"`                      // Number of GPUs, "-1" when not applicable
	Mfr            string `json:"Mfr,omitempty" validate:"omitempty" example:"NVIDIA"`        // Manufacturer of the GPU, NA when not applicable
	Model          string `json:"Model,omitempty" validate:"omitempty" example:"Tesla K80"`   // Model of the GPU, NA when not applicable
	MemSizeGB      string `json:"MemSizeGB,omitempty" validate:"omitempty" example:"12"`      // Memory size of the GPU in GB, "-1" when not applicable
	TotalMemSizeGB string `json:"TotalMemSizeGB,omitempty" validate:"omitempty" example:"24"` // Total Memory size of the GPU in GB, "-1" when not applicable
}

// AcceleratorType is the kind of AI accelerator attached to a VM specification.
type AcceleratorType string

const (
	AcceleratorGPU AcceleratorType = "GPU" // General-purpose GPU (NVIDIA, AMD, ...)
	AcceleratorTPU AcceleratorType = "TPU" // Google Tensor Processing Unit
	AcceleratorNPU AcceleratorType = "NPU" // Neural processing ASIC (AWS Inferentia/Trainium, Qualcomm AI 100, ...)
)

// AcceleratorInfo represents the AI accelerator(GPU/TPU/NPU) details of a VM specification.
// Unlike GpuInfo, Count and memory sizes describe the devices as seen by the VM.
type AcceleratorInfo struct {
	Type           AcceleratorType `json:"Type" validate:"required" example:"GPU"`                     // GPU, TPU, NPU
	Mfr            string          `json:"Mfr,omitempty" validate:"omitempty" example:"NVIDIA"`        // Manufacturer of the accelerator, NA when not applicable
	Model          string          `json:"Model,omitempty" validate:"omitempty" example:"Tesla K80"`   // Model of the accelerator, NA when not applicable
	Count          string          `json:"Count" validate:"required" example:"2"`                      // Number of accelerator devices attached to the VM, "-1" when not applicable
	Fraction       string          `json:"Fraction,omitempty" validate:"omitempty" example:""`         // Share of one physical accelerator per device for partial(fractional) GPUs, empty for whole devices
	MemSizeGB      string          `json:"MemSizeGB,omitempty" validate:"omitempty" example:"12"`      // Memory size per device available to the VM in GB, "-1" when not applicable
	TotalMemSizeGB string          `json:"TotalMemSizeGB,omitempty" validate:"omitempty" example:"24"` // Total memory size of all devices available to the VM in GB, "-1" when not applicable
}

type VMSpecHandler interface {
	ListVMSpec() ([]*VMSpecInfo, error)
	GetVMSpec(Name string) (VMSpecInfo, error)

	ListOrgVMSpec() (string, error)           // return string: json format
	GetOrgVMSpec(Name string) (string, error) // return string: json format
}
