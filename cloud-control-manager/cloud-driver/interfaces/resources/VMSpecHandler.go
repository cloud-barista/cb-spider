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
	Mem    string
	Gpu    []GpuInfo

	KeyValueList []KeyValue
}

type VCpuInfo struct {
<<<<<<< HEAD
	Count string //  오타로 보여 수정 Conut => Count
	Clock string // GHz
=======
	Count	         string
	Clock	         string // GHz
>>>>>>> 5ad3f3eae178e16d742f545b6b77e4e8227ae2af
}

type GpuInfo struct {
	Count string //  오타로 보여 수정 Conut => Count
	Mfr   string
	Model string
	Mem   string
}

type VMSpecHandler interface {

	// Region: AWS=Region, GCP=Zone, Azure=Location
	ListVMSpec(Region string) ([]*VMSpecInfo, error)
	GetVMSpec(Region string, Name string) (VMSpecInfo, error)

<<<<<<< HEAD
	ListOrgVMSpec(Region string) (string, error)             // return string: json format
=======
	ListOrgVMSpec(Region string) (string error) // return string: json format
>>>>>>> 5ad3f3eae178e16d742f545b6b77e4e8227ae2af
	GetOrgVMSpec(Region string, Name string) (string, error) // return string: json format
}
