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

type VMSpecInfo struct {
	Region string
	Name   string
	VCpu   VCpuInfo
	Mem    string // MB
	Gpu    []GpuInfo

	KeyValueList []KeyValue
}

type VCpuInfo struct {
	Count string
	Clock string // GHz
}

type GpuInfo struct {
	Count string
	Mfr   string
	Model string
	Mem   string // MB
}

type VMSpecHandler interface {

	ListVMSpec() ([]*VMSpecInfo, error)
	GetVMSpec(Name string) (VMSpecInfo, error)

	ListOrgVMSpec() (string, error)             // return string: json format
	GetOrgVMSpec(Name string) (string, error) // return string: json format
}
