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
	"strings"

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
	zone := vmSpecHandler.Region.Zone

	resp, err := vmSpecHandler.Client.MachineTypes.List(projectID, zone).Do()

	if err != nil {
		cblogger.Error(err)
		return []*irs.VMSpecInfo{}, err
	}
	var vmSpecInfo []*irs.VMSpecInfo
	for _, i := range resp.Items {
		info := irs.VMSpecInfo{
			Region: zone,
			Name:   i.Name,
			VCpu: irs.VCpuInfo{
				Count: string(i.GuestCpus),
			},
			Mem: string(i.MemoryMb),
		}
		vmSpecInfo = append(vmSpecInfo, &info)
	}
	return vmSpecInfo, nil
}

func (vmSpecHandler *GCPVMSpecHandler) GetVMSpec(Region string, Name string) (irs.VMSpecInfo, error) {
	// default info
	projectID := vmSpecHandler.Credential.ProjectID
	zone := vmSpecHandler.Region.Zone

	info, err := vmSpecHandler.Client.MachineTypes.Get(projectID, zone, Name).Do()

	if err != nil {
		cblogger.Error(err)
		return irs.VMSpecInfo{}, err
	}

	vmSpecInfo := irs.VMSpecInfo{
		Region: Region,
		Name:   Name,
		VCpu: irs.VCpuInfo{
			Count: string(info.GuestCpus),
			Clock: "",
		},
		Mem: string(info.MemoryMb),
		Gpu: []irs.GpuInfo{
			{
				Count: "",
				Mfr:   "",
				Model: "",
				Mem:   "",
			},
		},
	}

	return vmSpecInfo, nil
}

func (vmSpecHandler *GCPVMSpecHandler) ListOrgVMSpec(Region string) (string, error) {
	// projectID := vmSpecHandler.Credential.ProjectID
	// zone := vmSpecHandler.Region.Zone

	return "", nil
}

func (vmSpecHandler *GCPVMSpecHandler) GetOrgVMSpec(Region string, Name string) (string, error) {
	// projectID := vmSpecHandler.Credential.ProjectID
	// zone := vmSpecHandler.Region.Zone

	return "", nil
}

// gcp 같은경우 n1 타입만 그래픽 카드가 추가 되며
// 1. n1타입인지 확인하는 로직 필요
// 2. 해당 카드에 관련된 정보를 조회하는 로직필요.
// 3. 해당 리스트를 조회하고 해당 GPU를 선택하는 로직

func CheckMachineType(Name string) bool {
	prefix := "n1"

	if ok := strings.HasPrefix(prefix, Name); ok {
		return ok
	}

	return false

}
