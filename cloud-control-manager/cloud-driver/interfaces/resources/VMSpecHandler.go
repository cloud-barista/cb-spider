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
	Region           string
	Name             string
	VCpu             VCpuInfo
	Mem	         string
	Gpu              []GpuInfo

	KeyValueList     []KeyValue
}

type VCpuInfo struct {
	Conut	         string
	Clock	         string // GHz
}

type GpuInfo struct {
	Conut	         string
	Mfr	         string
	Model	         string
	Mem	         string
}

type VMSpecHandler interface {

	// Region: AWS=Region, GCP=Zone, Azure=Location	
	ListVMSpec(Region string) ([]*VMSpecInfo, error)
	GetVVMSpec(Region string, Name string) (VMSpecInfo, error)

	ListOrgVMSpec(Region string) (string error) // return string: json format
	GetOrgVVMSpec(Region string, Name string) (string, error) // return string: json format
}
