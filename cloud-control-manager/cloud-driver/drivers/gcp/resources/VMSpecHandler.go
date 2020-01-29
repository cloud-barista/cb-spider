// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// program by ysjeon@mz.co.kr, 2019.07.
// modify by devunet@mz.co.kr, 2019.11.

package resources

import (
	"context"
	_ "errors"

	compute "google.golang.org/api/compute/v1"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type GCPVMSpecHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

func (vmSpecHandler *GCPVMSpecHandler) ListVMSpec(Region string) ([]*irs.VMSpecInfo, error) {
	
	projectID := vmSpecHandler.Credential.ProjectID
	zone := 

	return nil, nil
}

func (vmSpecHandler *GCPVMSpecHandler) GetVMSpec(Region string, Name string) (irs.VMSpecInfo, error) {
	// default info
	projectID := vmSpecHandler.Credential.ProjectID
	zone := vmSpecHandler.Region.Zone

	info, err := vmSpecHandler.Client.MachineTypes.Get(projectID, zone, Name).Do()

	if err != nil {
		cblogger.Error(err)
		return irs.VMSpecInfo{},err
	}

	vmSpecInfo := irs.VMSpecInfo{
		Region : Region,
		Name : Name,
		VCpu : irs.VCpuInfo{
			Count :string(info.GuestCpus),
			Clock : "",
		},
		Mem : string(info.MemoryMb),
		Gpu : irs.GpuInfo{
			{
				Count :"",
				Mfr : "",
				Model : "",
				Mem : "",
			},
		},
	}
	
	return vmSpecInfo, nil
}

func (vmSpecHandler *GCPVMSpecHandler) ListOrgVMSpec(Region string) (string, error) {
	return nil, nil
}

func (vmSpecHandler *GCPVMSpecHandler) GetOrgVMSpec(Region string, Name string) (string, error) {
	return nil, nil
}
