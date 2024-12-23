// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// program by ysjeon@mz.co.kr, 2020.01.

package resources

import (
	"context"
	_ "errors"
	"strconv"
	"strings"

	compute "google.golang.org/api/compute/v1"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type GCPVMSpecHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

func (vmSpecHandler *GCPVMSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {

	projectID := vmSpecHandler.Credential.ProjectID
	zone := vmSpecHandler.Region.Zone

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vmSpecHandler.Region.Zone,
		ResourceType: call.VMSPEC,
		ResourceName: "",
		CloudOSAPI:   "List()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	resp, err := vmSpecHandler.Client.MachineTypes.List(projectID, zone).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return []*irs.VMSpecInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))
	var vmSpecInfo []*irs.VMSpecInfo
	for _, i := range resp.Items {

		gpuInfoList := []irs.GpuInfo{}
		if i.Accelerators != nil {
			gpuInfoList = acceleratorsToGPUInfoList(i.Accelerators)

		}
		info := irs.VMSpecInfo{
			Region: zone,
			Name:   i.Name,
			VCpu: irs.VCpuInfo{
				Count: strconv.FormatInt(i.GuestCpus, 10),
			},
			Mem:  strconv.FormatInt(i.MemoryMb, 10),
			Disk: "-1",
			Gpu:  gpuInfoList,
		}

		info.KeyValueList, err = ConvertKeyValueList(info)
		if err != nil {
			info.KeyValueList = nil
			cblogger.Error(err)
		}

		vmSpecInfo = append(vmSpecInfo, &info)
	}
	return vmSpecInfo, nil
}

func (vmSpecHandler *GCPVMSpecHandler) GetVMSpec(Name string) (irs.VMSpecInfo, error) {
	// default info
	projectID := vmSpecHandler.Credential.ProjectID
	zone := vmSpecHandler.Region.Zone

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vmSpecHandler.Region.Zone,
		ResourceType: call.VMSPEC,
		ResourceName: Name,
		CloudOSAPI:   "Get()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	info, err := vmSpecHandler.Client.MachineTypes.Get(projectID, zone, Name).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.VMSpecInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	gpuInfoList := []irs.GpuInfo{}
	if info.Accelerators != nil {
		gpuInfoList = acceleratorsToGPUInfoList(info.Accelerators)

	}

	vmSpecInfo := irs.VMSpecInfo{
		Region: vmSpecHandler.Region.Region,
		Name:   Name,
		VCpu: irs.VCpuInfo{
			Count: strconv.FormatInt(info.GuestCpus, 10),
			Clock: "",
		},
		Mem:  strconv.FormatInt(info.MemoryMb, 10),
		Disk: "-1",
		Gpu:  gpuInfoList,
	}

	vmSpecInfo.KeyValueList, err = ConvertKeyValueList(vmSpecInfo)
	if err != nil {
		vmSpecInfo.KeyValueList = nil
		cblogger.Error(err)
	}
	return vmSpecInfo, nil
}

func (vmSpecHandler *GCPVMSpecHandler) ListOrgVMSpec() (string, error) {
	projectID := vmSpecHandler.Credential.ProjectID
	zone := vmSpecHandler.Region.Zone

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vmSpecHandler.Region.Zone,
		ResourceType: call.VMSPEC,
		ResourceName: "",
		CloudOSAPI:   "List()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	resp, err := vmSpecHandler.Client.MachineTypes.List(projectID, zone).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return "", err
	}
	callogger.Info(call.String(callLogInfo))
	j, _ := resp.MarshalJSON()

	return string(j), err
}

func (vmSpecHandler *GCPVMSpecHandler) GetOrgVMSpec(Name string) (string, error) {
	projectID := vmSpecHandler.Credential.ProjectID
	zone := vmSpecHandler.Region.Zone

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vmSpecHandler.Region.Zone,
		ResourceType: call.VMSPEC,
		ResourceName: Name,
		CloudOSAPI:   "Get()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	info, err := vmSpecHandler.Client.MachineTypes.Get(projectID, zone, Name).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return "", err
	}
	callogger.Info(call.String(callLogInfo))
	j, _ := info.MarshalJSON()

	return string(j), err
}

// type GpuInfo struct {
//     Count string `json:"Count" validate:"required" example:"1"`                    // Number of GPUs, "-1" when not applicable
//     Mfr   string `json:"Mfr,omitempty" validate:"omitempty" example:"NVIDIA"`      // Manufacturer of the GPU, NA when not applicable
//     Model string `json:"Model,omitempty" validate:"omitempty" example:"Tesla K80"` // Model of the GPU, NA when not applicable
//     Mem   string `json:"Mem,omitempty" validate:"omitempty" example:"8192"`        // Memory size of the GPU in MB, "-1" when not applicable
// }

// accerators 목록을 GPU목록으로 변경
// 이름에서 발견된 규칙
//
//	0번째는 제조사
//	마지막에 gb면 용량
//	가운데는 모델
//		"name": "nvidia-tesla-a100",
//		"name": "nvidia-h100-80gb",
//		"name": "nvidia-h100-mega-80gb",
//		"name": "nvidia-l4"
//		"name": "nvidia-l4-vws"
func acceleratorsToGPUInfoList(accerators []*compute.MachineTypeAccelerators) []irs.GpuInfo {
	gpuInfoList := []irs.GpuInfo{}
	for _, accelerator := range accerators {
		gpuInfo := irs.GpuInfo{}

		accrType := strings.Split(accelerator.GuestAcceleratorType, "-")
		if len(accrType) >= 3 {
			// 첫 번째 요소를 Mfr에 할당
			gpuInfo.Mfr = strings.ToUpper(accrType[0])

			// 마지막 요소를 확인
			lastElement := accrType[len(accrType)-1]
			if strings.HasSuffix(lastElement, "gb") {
				// "gb"를 제거하고 숫자만 추출
				numStr := strings.TrimSuffix(lastElement, "gb")
				if num, err := strconv.Atoi(numStr); err == nil {
					gpuInfo.Mem = strconv.Itoa(num * 1024) // GB를 MB로 변환 후 숫자만 string으로 저장
				}
				// 첫 번째와 마지막 요소를 제외한 나머지를 Model에 할당
				if len(accrType) > 2 {
					gpuInfo.Model = strings.ToUpper(strings.Join(accrType[1:len(accrType)-1], " "))
				}
			} else {
				// 마지막 요소가 "gb"로 끝나지 않는 경우
				gpuInfo.Mem = "" // Mem은 빈 문자열로 설정
				if len(accrType) > 1 {
					// 첫 번째 요소를 제외한 나머지를 Model에 할당
					gpuInfo.Model = strings.ToUpper(strings.Join(accrType[1:], " "))
				}
			}
		}
		gpuInfo.Count = strconv.FormatInt(accelerator.GuestAcceleratorCount, 10)

		gpuInfoList = append(gpuInfoList, gpuInfo)
	}
	return gpuInfoList
}

// GPU 정보조회 미구현
// accelerator 정보를 조회하여 set하려했으나, 해당정보안에 gpuInfo에 넣을 값이 없어서 미구현.
// func getAcceleratorType(client *compute.Service, project string, region string, zone string, acceleratorNames []string)(irs.GpuInfo, error){
// 	// logger for HisCall
// 	callogger := call.GetLogger("HISCALL")
// 	callLogInfo := call.CLOUDLOGSCHEMA{
// 		CloudOS:      call.GCP,
// 		RegionZone:   zone,
// 		ResourceType: call.VMSPEC,
// 		ResourceName: "VMSpec",
// 		CloudOSAPI:   "AcceleratorTypes.Get()",
// 		ElapsedTime:  "",
// 		ErrorMSG:     "",
// 	}
// 	callLogStart := call.Start()
//
// 	for _, i := range acceleratorNames {
// 	info, err := client.AcceleratorTypes.Get(project, zone, AcceleratorName).Do()
// 	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
//
// 	if err != nil {
// 		callLogInfo.ErrorMSG = err.Error()
// 		callogger.Info(call.String(callLogInfo))
// 		cblogger.Error(err)
// 		return irs.GpuInfo{}, err
// 	}
// 	callogger.Info(call.String(callLogInfo))
//
// 	gpuInfo := irs.GpuInfo{}
//
// }

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
