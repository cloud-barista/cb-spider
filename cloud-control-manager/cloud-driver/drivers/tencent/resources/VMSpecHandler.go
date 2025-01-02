package resources

import (
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type TencentVmSpecHandler struct {
	Region idrv.RegionInfo
	Client *cvm.Client
}

// @TODO : Region : zone id(Region이 아닌 zone id로 조회해야 함.)
func (vmSpecHandler *TencentVmSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	//cblogger.Infof("ListVMSpec(ZoneId:[%s])", Region)

	zoneId := vmSpecHandler.Region.Zone
	//zoneId := Region
	cblogger.Infof("Session Zone : [%s]", zoneId)
	if zoneId == "" {
		cblogger.Error("Connection information does not contain Zone information.")
		return nil, errors.New("Connection information does not contain Zone information.")
	}

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   vmSpecHandler.Region.Zone,
		ResourceType: call.VMSPEC,
		ResourceName: "ListVMSpec()",
		CloudOSAPI:   "DescribeInstanceTypeConfigs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewDescribeZoneInstanceConfigInfosRequest()
	// request := cvm.NewDescribeInstanceTypeConfigsRequest()
	request.Filters = []*cvm.Filter{
		&cvm.Filter{
			Name:   common.StringPtr("zone"),
			Values: common.StringPtrs([]string{zoneId}),
		},
	}
	callLogStart := call.Start()
	//DescribeInstanceTypes 로 바뀐것인가?
	response, err := vmSpecHandler.Client.DescribeZoneInstanceConfigInfos(request)
	// response, err := vmSpecHandler.Client.DescribeInstanceTypeConfigs(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return nil, err
	}

	//cblogger.Debug(response)
	//cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	var vmSpecInfoList []*irs.VMSpecInfo
	for _, curSpec := range response.Response.InstanceTypeQuotaSet {
		cblogger.Debugf("[%s] VM Spec Information Processing", *curSpec.InstanceType)

		vmSpecInfo := extractVmSpec(curSpec)
		vmSpecInfoList = append(vmSpecInfoList, &vmSpecInfo)
	}

	// for _, curSpec := range response.Response.InstanceTypeConfigSet {
	// 	cblogger.Debugf("[%s] VM Spec Information Processing", *curSpec.InstanceType)

	// 	vmSpecInfo := ExtractVMSpecInfo(curSpec)
	// 	vmSpecInfoList = append(vmSpecInfoList, &vmSpecInfo)
	// }

	cblogger.Debug(vmSpecInfoList)
	//cblogger.Debug(vmSpecInfoList)
	return vmSpecInfoList, nil
}

func (vmSpecHandler *TencentVmSpecHandler) GetVMSpec(name string) (irs.VMSpecInfo, error) {
	//cblogger.Infof("Start GetVMSpec(ZoneId:[%s], Name:[%s])", Region, Name)
	//name = "MA5.LARGE32"
	//name = "S2.SMALL1"
	name = "GN7.20XLARGE320"
	cblogger.Infof("Spec Name:[%s]", name)

	zoneId := vmSpecHandler.Region.Zone
	//zoneId := Region
	cblogger.Infof("Session Zone : [%s]", zoneId)
	if zoneId == "" {
		cblogger.Error("Connection information does not contain Zone information.")
		return irs.VMSpecInfo{}, errors.New("Connection information does not contain Zone information.")
	}

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   vmSpecHandler.Region.Zone,
		ResourceType: call.VMSPEC,
		ResourceName: name,
		CloudOSAPI:   "DescribeInstanceTypeConfigs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	//request := cvm.NewDescribeInstanceTypeConfigsRequest()
	request := cvm.NewDescribeZoneInstanceConfigInfosRequest()
	request.Filters = []*cvm.Filter{
		&cvm.Filter{
			Name:   common.StringPtr("zone"), //존으로 검색
			Values: common.StringPtrs([]string{zoneId}),
		},
		&cvm.Filter{
			Name:   common.StringPtr("instance-type"), //인스턴스 타입으로 검색
			Values: common.StringPtrs([]string{name}),
		},
	}
	callLogStart := call.Start()
	//response, err := vmSpecHandler.Client.DescribeInstanceTypeConfigs(request)
	response, err := vmSpecHandler.Client.DescribeZoneInstanceConfigInfos(request)

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return irs.VMSpecInfo{}, err
	}

	//cblogger.Debug(response)
	//cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	// if len(response.Response.InstanceTypeConfigSet) > 0 {
	// 	vmSpecInfo := ExtractVMSpecInfo(response.Response.InstanceTypeConfigSet[0])
	// 	cblogger.Debug(vmSpecInfo)
	// 	return vmSpecInfo, nil
	// } else {
	// 	return irs.VMSpecInfo{}, errors.New("No information found")
	// }
	if len(response.Response.InstanceTypeQuotaSet) > 0 { // 요금제만 다른 같은 값이 오므로 0번째 선택해도 무방
		vmSpecInfo := extractVmSpec(response.Response.InstanceTypeQuotaSet[0])
		cblogger.Debug(vmSpecInfo)
		return vmSpecInfo, nil
	} else {
		return irs.VMSpecInfo{}, errors.New("No information found")
	}
}

func (vmSpecHandler *TencentVmSpecHandler) ListOrgVMSpec() (string, error) {
	//cblogger.Infof("ListOrgVMSpec(ZoneId:[%s])", Region)

	zoneId := vmSpecHandler.Region.Zone
	//zoneId := Region
	cblogger.Infof("Session Zone : [%s]", zoneId)
	if zoneId == "" {
		cblogger.Error("Connection information does not contain Zone information.")
		return "", errors.New("Connection information does not contain Zone information.")
	}

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   vmSpecHandler.Region.Zone,
		ResourceType: call.VMSPEC,
		ResourceName: "ListOrgVMSpec()",
		CloudOSAPI:   "DescribeInstanceTypeConfigs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewDescribeInstanceTypeConfigsRequest()
	request.Filters = []*cvm.Filter{
		&cvm.Filter{
			Name:   common.StringPtr("zone"),
			Values: common.StringPtrs([]string{zoneId}),
		},
	}
	callLogStart := call.Start()
	response, err := vmSpecHandler.Client.DescribeInstanceTypeConfigs(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return "", err
	}

	//cblogger.Debug(response)
	// cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	jsonString, errJson := ConvertJsonString(response.Response.InstanceTypeConfigSet)
	if errJson != nil {
		cblogger.Error(errJson)
		return "", errJson
	}
	cblogger.Debug(jsonString)
	return jsonString, errJson
}

func (vmSpecHandler *TencentVmSpecHandler) GetOrgVMSpec(Name string) (string, error) {
	cblogger.Infof("Spec Name:[%s]", Name)
	//cblogger.Infof("Start GetOrgVMSpec(ZoneId:[%s], Name:[%s])", Region, Name)

	zoneId := vmSpecHandler.Region.Zone
	//zoneId := Region
	cblogger.Infof("Session Zone : [%s]", zoneId)
	if zoneId == "" {
		cblogger.Error("Connection information does not contain Zone information.")
		return "", errors.New("Connection information does not contain Zone information.")
	}

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   vmSpecHandler.Region.Zone,
		ResourceType: call.VMSPEC,
		ResourceName: Name,
		CloudOSAPI:   "DescribeInstanceTypeConfigs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewDescribeInstanceTypeConfigsRequest()
	request.Filters = []*cvm.Filter{
		&cvm.Filter{
			Name:   common.StringPtr("zone"),
			Values: common.StringPtrs([]string{zoneId}),
		},
		&cvm.Filter{
			Name:   common.StringPtr("instance-type"), //인스턴스 타입으로 검색
			Values: common.StringPtrs([]string{Name}),
		},
	}
	callLogStart := call.Start()
	response, err := vmSpecHandler.Client.DescribeInstanceTypeConfigs(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return "", err
	}

	//cblogger.Debug(response)
	//cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	if len(response.Response.InstanceTypeConfigSet) > 0 {
		jsonString, errJson := ConvertJsonString(response.Response.InstanceTypeConfigSet[0])
		if errJson != nil {
			cblogger.Error(errJson)
			return "", errJson
		}
		cblogger.Debug(jsonString)
		return jsonString, errJson
	} else {

		return "", errors.New("Unable to find information")

	}
}

func extractVmSpec(instanceTypeInfo *cvm.InstanceTypeQuotaItem) irs.VMSpecInfo {
	vCpuInfo := irs.VCpuInfo{}
	// gpuInfoList := []irs.GpuInfo{}

	//기본 정보
	vmSpecInfo := irs.VMSpecInfo{
		Name:   *instanceTypeInfo.InstanceType,
		Region: *instanceTypeInfo.Zone,
	}

	//Memory 정보 처리
	if !reflect.ValueOf(instanceTypeInfo.Memory).IsNil() {
		vmSpecInfo.Mem = strconv.FormatInt(*instanceTypeInfo.Memory*1024, 10) // GB->MB로 변환
	}

	//VCPU 정보 처리 - Count
	if !reflect.ValueOf(instanceTypeInfo.Cpu).IsNil() {
		vCpuInfo.Count = strconv.FormatInt(*instanceTypeInfo.Cpu, 10)

		// Clock 추출
		//"Frequency": "-/3.1GHz", "Frequency": "2.5GHz/-", "Frequency": "2.5GHz/3.1GHz",

		parts := strings.Split(*instanceTypeInfo.Frequency, "/")
		var targetClock string

		if parts[0] != "-" {
			targetClock = parts[0] // 기본 주파수 우선
		} else if len(parts) > 1 && parts[1] != "-" {
			targetClock = parts[1] // 최대 터보 주파수 사용
		} else {
			targetClock = "-1"
		}

		re := regexp.MustCompile(`[-+]?([0-9]*\.[0-9]+|[0-9]+)`)
		match := re.FindString(targetClock)

		if match != "" {
			targetClock = match
		}
		vCpuInfo.Clock = targetClock

	}
	vmSpecInfo.VCpu = vCpuInfo

	cblogger.Info("instanceTypeInfo.Gpu ", *instanceTypeInfo.Gpu)
	gpuInfoList := []irs.GpuInfo{}
	//GPU 정보가 있는 인스터스는 GPU 처리
	val := reflect.ValueOf(*instanceTypeInfo.Gpu)
	//val int64 :: int64 ::: 4
	cblogger.Info("instanceTypeInfo.Gpu val ", val.Kind(), " :: ", reflect.Int64, " ::: ", val.Int())
	//if !reflect.ValueOf(*instanceTypeInfo.Gpu).IsNil() {
	if val.Kind() == reflect.Int64 && val.Int() > 0 {
		//vmSpecInfo.Gpu = []irs.GpuInfo{irs.GpuInfo{Count: strconv.FormatInt(*instanceTypeInfo.GPU, 10)}}
		//GPUInfo

		// 기본 값 설정
		gpuInfo := irs.GpuInfo{
			Count: "-1",
			Model: "NA",
			Mfr:   "NA",
			Mem:   "0",
		}
		gpuInfo.Count = strconv.FormatInt(*instanceTypeInfo.Gpu, 10)

		gpuInfoList = append(gpuInfoList, gpuInfo)
	} else {
		cblogger.Info("val.Kind() == reflect.Int64", val.Kind() == reflect.Int64)
		cblogger.Info("val.Int() > 0", val.Int() > 0)
	}
	vmSpecInfo.Gpu = gpuInfoList

	// Disk   string    `json:"Disk" validate:"required" example:"8"`           // Disk size in GB, "-1" when not applicable
	vmSpecInfo.Disk = "-1"

	//KeyValue 목록 처리
	keyValueList, errKeyValue := ConvertKeyValueList(instanceTypeInfo)
	if errKeyValue != nil {

		cblogger.Error(errKeyValue)

	}
	vmSpecInfo.KeyValueList = keyValueList

	return vmSpecInfo
}

// 인스턴스 스펙 정보를 추출함
func ExtractVMSpecInfo(instanceTypeInfo *cvm.InstanceTypeConfig) irs.VMSpecInfo {
	cblogger.Debugf("ExtractVMSpecInfo : SpecName:[%s]", *instanceTypeInfo.InstanceType)
	//cblogger.Debug(instanceTypeInfo)

	vCpuInfo := irs.VCpuInfo{}
	// gpuInfoList := []irs.GpuInfo{}

	//기본 정보
	vmSpecInfo := irs.VMSpecInfo{
		Name:   *instanceTypeInfo.InstanceType,
		Region: *instanceTypeInfo.Zone,
	}

	//Memory 정보 처리
	if !reflect.ValueOf(instanceTypeInfo.Memory).IsNil() {
		vmSpecInfo.Mem = strconv.FormatInt(*instanceTypeInfo.Memory*1024, 10) // GB->MB로 변환
	}

	//VCPU 정보 처리 - Count
	if !reflect.ValueOf(instanceTypeInfo.CPU).IsNil() {
		vCpuInfo.Count = strconv.FormatInt(*instanceTypeInfo.CPU, 10)
	}
	vmSpecInfo.VCpu = vCpuInfo

	//GPU 정보가 있는 인스터스는 GPU 처리
	if !reflect.ValueOf(instanceTypeInfo.GPU).IsNil() {
		vCpuInfo.Count = strconv.FormatInt(*instanceTypeInfo.GPU, 10)
		vmSpecInfo.Gpu = []irs.GpuInfo{irs.GpuInfo{Count: strconv.FormatInt(*instanceTypeInfo.GPU, 10)}}
	}

	//KeyValue 목록 처리
	keyValueList, errKeyValue := ConvertKeyValueList(instanceTypeInfo)
	if errKeyValue != nil {

		cblogger.Error(errKeyValue)

	}
	vmSpecInfo.KeyValueList = keyValueList

	return vmSpecInfo
}
