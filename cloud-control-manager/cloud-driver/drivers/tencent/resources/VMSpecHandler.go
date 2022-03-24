package resources

import (
	"errors"
	"reflect"
	"strconv"

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

//@TODO : Region : zone id(Region이 아닌 zone id로 조회해야 함.)
func (vmSpecHandler *TencentVmSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	//cblogger.Infof("ListVMSpec(ZoneId:[%s])", Region)

	zoneId := vmSpecHandler.Region.Zone
	//zoneId := Region
	cblogger.Infof("Session Zone : [%s]", zoneId)
	if zoneId == "" {
		cblogger.Error("Connection 정보에 Zone 정보가 없습니다.")
		return nil, errors.New("Connection 정보에 Zone 정보가 없습니다.")
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
		return nil, err
	}

	//spew.Dump(response)
	//cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	var vmSpecInfoList []*irs.VMSpecInfo
	for _, curSpec := range response.Response.InstanceTypeConfigSet {
		cblogger.Debugf("[%s] VM Spec 정보 처리", *curSpec.InstanceType)
		vmSpecInfo := ExtractVMSpecInfo(curSpec)
		vmSpecInfoList = append(vmSpecInfoList, &vmSpecInfo)
	}

	cblogger.Debug(vmSpecInfoList)
	//spew.Dump(vmSpecInfoList)
	return vmSpecInfoList, nil
}

func (vmSpecHandler *TencentVmSpecHandler) GetVMSpec(Name string) (irs.VMSpecInfo, error) {
	//cblogger.Infof("Start GetVMSpec(ZoneId:[%s], Name:[%s])", Region, Name)
	cblogger.Infof("Spec Name:[%s]", Name)

	zoneId := vmSpecHandler.Region.Zone
	//zoneId := Region
	cblogger.Infof("Session Zone : [%s]", zoneId)
	if zoneId == "" {
		cblogger.Error("Connection 정보에 Zone 정보가 없습니다.")
		return irs.VMSpecInfo{}, errors.New("Connection 정보에 Zone 정보가 없습니다.")
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
			Name:   common.StringPtr("zone"), //존으로 검색
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
		return irs.VMSpecInfo{}, err
	}

	//spew.Dump(response)
	//cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	if len(response.Response.InstanceTypeConfigSet) > 0 {
		vmSpecInfo := ExtractVMSpecInfo(response.Response.InstanceTypeConfigSet[0])
		cblogger.Debug(vmSpecInfo)
		return vmSpecInfo, nil
	} else {
		return irs.VMSpecInfo{}, errors.New("정보를 찾을 수 없습니다")
	}
}

func (vmSpecHandler *TencentVmSpecHandler) ListOrgVMSpec() (string, error) {
	//cblogger.Infof("ListOrgVMSpec(ZoneId:[%s])", Region)

	zoneId := vmSpecHandler.Region.Zone
	//zoneId := Region
	cblogger.Infof("Session Zone : [%s]", zoneId)
	if zoneId == "" {
		cblogger.Error("Connection 정보에 Zone 정보가 없습니다.")
		return "", errors.New("Connection 정보에 Zone 정보가 없습니다.")
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

	//spew.Dump(response)
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
		cblogger.Error("Connection 정보에 Zone 정보가 없습니다.")
		return "", errors.New("Connection 정보에 Zone 정보가 없습니다.")
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

	//spew.Dump(response)
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
		return "", errors.New("정보를 찾을 수 없습니다")
	}
}

//인스턴스 스펙 정보를 추출함
func ExtractVMSpecInfo(instanceTypeInfo *cvm.InstanceTypeConfig) irs.VMSpecInfo {
	cblogger.Debugf("ExtractVMSpecInfo : SpecName:[%s]", *instanceTypeInfo.InstanceType)
	//spew.Dump(instanceTypeInfo)

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
		cblogger.Errorf("[%]의 KeyValue 추출 실패", *instanceTypeInfo.InstanceType)
		cblogger.Error(errKeyValue)
	}
	vmSpecInfo.KeyValueList = keyValueList

	return vmSpecInfo
}
