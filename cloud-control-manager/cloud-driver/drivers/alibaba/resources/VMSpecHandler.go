package resources

//20211104 개선안 I에 의해 Region 파라메터 대신 세션의 Region 정보로 대체함.
import (
	"errors"
	"strconv"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaVmSpecHandler struct {
	Region idrv.RegionInfo
	Client *ecs.Client
}

// 인스턴스 스펙 정보를 추출함
func ExtractVMSpecInfo(Region string, instanceTypeInfo ecs.InstanceType) irs.VMSpecInfo {
	//@TODO : 2020-03-26 Ali클라우드 API 구조가 바뀐 것 같아서 임시로 변경해 놓음.
	//윈도우즈에서는 ecs.InstanceType를 인식하지만 Mac과 신규 API에서는 ecs.InstanceType를 못찾고 ecs.InstanceTypeInDescribeInstanceTypes를 이용함.
	//func ExtractVMSpecInfo(Region string, instanceTypeInfo ecs.InstanceTypeInDescribeInstanceTypes) irs.VMSpecInfo {
	//@todo : 2020-04-20 ecs.InstanceTypeInDescribeInstanceTypes을 인식 못해서 다시 ecs.InstanceType을 사용함.
	//func ExtractVMSpecInfo(Region string, instanceTypeInfo ecs.InstanceType) irs.VMSpecInfo {
	//ecs.InstanceType
	cblogger.Infof("ExtractVMSpecInfo : Region:[%s] / SpecName:[%s]", Region, instanceTypeInfo.InstanceTypeFamily)
	//cblogger.Debug(instanceTypeInfo)

	vCpuInfo := irs.VCpuInfo{
		Count:    "-1",
		ClockGHz: "-1",
	}
	// gpuInfoList := []irs.GpuInfo{
	// 	{
	// 		Count: strconv.Itoa(instanceTypeInfo.GPUAmount),
	// 		Model: instanceTypeInfo.GPUSpec,
	// 	},
	// }
	gpuInfoList := []irs.GpuInfo{}

	// 기본 값 설정
	gpuInfo := irs.GpuInfo{
		Count:          "-1",
		Model:          "NA",
		Mfr:            "NA",
		MemSizeGB:      "-1",
		TotalMemSizeGB: "-1",
	}

	if instanceTypeInfo.GPUAmount != 0 {
		gpuInfo.Count = strconv.Itoa(instanceTypeInfo.GPUAmount)
		if instanceTypeInfo.GPUSpec != "" {
			gpuInfo.Model = strings.ToUpper(instanceTypeInfo.GPUSpec)
			gpu := strings.Split(instanceTypeInfo.GPUSpec, " ") // "Nvidia Tesla P4"
			if len(gpu) > 0 {
				gpuInfo.Mfr = strings.ToUpper(gpu[0])
				cblogger.Infof("Manufacturer Information Extraction: Original[%s] / Extracted[%s]", instanceTypeInfo.GPUSpec, gpuInfo.Mfr)
			}
		}

		if instanceTypeInfo.GPUMemorySize != 0 {
			gpuInfo.MemSizeGB = strconv.Itoa(int(instanceTypeInfo.GPUMemorySize))
			gpuInfo.TotalMemSizeGB = strconv.Itoa(int(instanceTypeInfo.GPUMemorySize) * instanceTypeInfo.GPUAmount)
		}

		gpuInfoList = append(gpuInfoList, gpuInfo)
	}

	// if !reflect.ValueOf(&instanceTypeInfo.GPUSpec).IsNil() {
	// 	gpu := strings.Split(instanceTypeInfo.GPUSpec, " ") //"Nvidia Tesla P4"
	// 	cblogger.Infof("Manufacturer Information Extraction: Original[%s] / Extracted[%s]", instanceTypeInfo.GPUSpec, gpu[0])
	// 	gpuInfoList[0].Mfr = gpu[0]
	// }

	//결과에 리전 정보는 없기 때문에 조회한 리전 정보를 전달 받아서 처리함.
	vmSpecInfo := irs.VMSpecInfo{
		Region: Region,
	}

	//VCPU 정보 처리 - Count
	//if !reflect.ValueOf(&instanceTypeInfo.CpuCoreCount).IsNil() {
	vCpuInfo.Count = strconv.Itoa(instanceTypeInfo.CpuCoreCount)
	//}

	vmSpecInfo.VCpu = vCpuInfo

	vmSpecInfo.Gpu = gpuInfoList

	//if !reflect.ValueOf(&instanceTypeInfo.InstanceTypeId).IsNil() {
	vmSpecInfo.Name = instanceTypeInfo.InstanceTypeId
	//}

	//if !reflect.ValueOf(&instanceTypeInfo.MemorySize).IsNil() {
	//vmSpecInfo.Mem = strconv.FormatFloat(instanceTypeInfo.MemorySize, 'f', 0, 64)
	vmSpecInfo.MemSizeMiB = strconv.FormatFloat(instanceTypeInfo.MemorySize*1024, 'f', 0, 64) // GiB(real)->MiB로 변환
	//}

	// LocalStorageCapacity: GiB -> GB (1GiB = 1.073741824GB)
	if instanceTypeInfo.LocalStorageCapacity > 0 {
		vmSpecInfo.DiskSizeGB = irs.ConvertGiBToGBInt64(instanceTypeInfo.LocalStorageCapacity)
	} else {
		vmSpecInfo.DiskSizeGB = "-1"
	}

	vmSpecInfo.KeyValueList = irs.StructToKeyValueList(instanceTypeInfo)

	return vmSpecInfo
}

func (vmSpecHandler *AlibabaVmSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	Region := vmSpecHandler.Region.Region
	cblogger.Infof("Start ListVMSpec(Session Region:[%s])", Region)
	var vMSpecInfoList []*irs.VMSpecInfo

	request := ecs.CreateDescribeInstanceTypesRequest()
	request.Scheme = "https"
	request.RegionId = Region

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   Region,
		ResourceType: call.VMSPEC,
		ResourceName: "ListVMSpec()",
		CloudOSAPI:   "DescribeInstanceTypes()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	resp, err := vmSpecHandler.Client.DescribeInstanceTypes(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Debug(resp)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to get ListVMSpec - %v", err)
		return vMSpecInfoList, err
	}
	callogger.Info(call.String(callLogInfo))

	//cblogger.Debug(resp)
	cblogger.Info("Number of Retrieved Instance Types : ", len(resp.InstanceTypes.InstanceType))
	for _, curInstance := range resp.InstanceTypes.InstanceType {
		cblogger.Infof("[%s] VM Retrieve Specification Information", curInstance.InstanceTypeFamily)
		vMSpecInfo := ExtractVMSpecInfo(Region, curInstance)
		vMSpecInfoList = append(vMSpecInfoList, &vMSpecInfo)
	}
	//cblogger.Debug(vMSpecInfoList)
	return vMSpecInfoList, nil
}

func (vmSpecHandler *AlibabaVmSpecHandler) GetVMSpec(Name string) (irs.VMSpecInfo, error) {
	Region := vmSpecHandler.Region.Region
	cblogger.Infof("Start GetVMSpec(Session Region:[%s], Name:[%s])", Region, Name)

	request := ecs.CreateDescribeInstanceTypesRequest()
	request.Scheme = "https"
	request.RegionId = Region
	//request.InstanceTypeFamily = Name

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   Region,
		ResourceType: call.VMSPEC,
		ResourceName: "Region:" + Region + "/ Name:" + Name,
		CloudOSAPI:   "DescribeInstanceTypes()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	resp, err := vmSpecHandler.Client.DescribeInstanceTypes(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Debug(resp)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to get GetVMSpec - %v", err)
		return irs.VMSpecInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Info("Number of Retrieved Instance Types : ", len(resp.InstanceTypes.InstanceType))
	//	cblogger.Debug(resp)

	if len(resp.InstanceTypes.InstanceType) < 1 {
		return irs.VMSpecInfo{}, errors.New("Notfound: '" + Name + "'Cannot find Spec information for the specified item.")
	}

	var vMSpecInfo irs.VMSpecInfo
	//인스턴스 타입으로 필터가 안되기 때문에 직접 처리함.
	//속도를 고려하면 조회 대상을 전체로 설정하지 말고 InstanceTypeFamily을 이용해서 패밀리 그룹을 제한할 수는 있음.
	for _, curInstance := range resp.InstanceTypes.InstanceType {
		cblogger.Debugf("[%s]", curInstance.InstanceTypeId)
		if Name == curInstance.InstanceTypeId {
			cblogger.Debugf("===> [%s]", curInstance.InstanceTypeId)
			cblogger.Infof("[%s] VM Retrieve Specification Information", curInstance.InstanceTypeId)
			vMSpecInfo = ExtractVMSpecInfo(Region, curInstance)
			break
		}
	}

	return vMSpecInfo, nil
}

// Alibaba Cloud의 정보 그대로를 가공 없이 JSON으로 리턴 함.
func (vmSpecHandler *AlibabaVmSpecHandler) ListOrgVMSpec() (string, error) {
	Region := vmSpecHandler.Region.Region
	cblogger.Infof("Start ListOrgVMSpec(Session Region:[%s])", Region)

	request := ecs.CreateDescribeInstanceTypesRequest()
	request.Scheme = "https"
	request.RegionId = Region

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   Region,
		ResourceType: call.VMSPEC,
		ResourceName: "ListOrgVMSpec()",
		CloudOSAPI:   "DescribeInstanceTypes()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	resp, err := vmSpecHandler.Client.DescribeInstanceTypes(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Debug(resp)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to get ListOrgVMSpec - %v", err)
		return "", err
	}
	callogger.Info(call.String(callLogInfo))

	//jsonString, errJson := ConvertJsonString(resp.InstanceTypes.InstanceType)
	jsonString, errJson := ConvertJsonString(resp.InstanceTypes)
	if errJson != nil {
		cblogger.Error(errJson)
	}
	return jsonString, errJson
}

func (vmSpecHandler *AlibabaVmSpecHandler) GetOrgVMSpec(Name string) (string, error) {
	Region := vmSpecHandler.Region.Region
	cblogger.Infof("Start GetOrgVMSpec(Session Region:[%s], Name:[%s])", Region, Name)
	request := ecs.CreateDescribeInstanceTypesRequest()
	request.Scheme = "https"
	request.RegionId = Region

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   Region,
		ResourceType: call.VMSPEC,
		ResourceName: "Region:" + Region + "/ Name:" + Name,
		CloudOSAPI:   "DescribeInstanceTypes()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	resp, err := vmSpecHandler.Client.DescribeInstanceTypes(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Debug(resp)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to get GetVMSpec - %v", err)
		return "", err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Info("Number of Retrieved Instance Types : ", len(resp.InstanceTypes.InstanceType))
	//	cblogger.Debug(resp)

	if len(resp.InstanceTypes.InstanceType) < 1 {
		return "", errors.New(Name + "Cannot find Spec information for the specified item.")
	}

	var jsonString string
	var errJson error
	//인스턴스 타입으로 필터가 안되기 때문에 직접 처리함.
	//속도를 고려하면 조회 대상을 전체로 설정하지 말고 InstanceTypeFamily을 이용해서 패밀리 그룹을 제한할 수는 있음.
	for _, curInstance := range resp.InstanceTypes.InstanceType {
		cblogger.Debugf("[%s]", curInstance.InstanceTypeId)
		if Name == curInstance.InstanceTypeId {
			cblogger.Debugf("===> [%s]", curInstance.InstanceTypeId)
			cblogger.Infof("[%s] VM Retrieve Specification Information", curInstance.InstanceTypeId)

			jsonString, errJson = ConvertJsonString(curInstance)
			if errJson != nil {
				cblogger.Error(errJson)
				return "", errJson
			}

			break
		}
	}

	return jsonString, nil
}
