package resources

//20211104 개선안 I에 의해 Region 파라메터 사용 로직 제거 - AWS는 Region이 아닌 Zone 기반으로 검색되며 Region은 로그용으로만 사용하고 있어서 세션 정보로 대체함.
import (
	"errors"
	"reflect"
	"strconv"
	"strings"

	//sdk2 "github.com/aws/aws-sdk-go-v2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeInstanceTypes
type AwsVmSpecHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

func ExtractGpuInfo(gpuDeviceInfo *ec2.GpuDeviceInfo) irs.GpuInfo {
	cblogger.Debug(gpuDeviceInfo)
	//cblogger.Info("================")
	//cblogger.Debug(gpuDeviceInfo)

	// gpuInfo := irs.GpuInfo{
	// 	Count: strconv.FormatInt(*gpuDeviceInfo.Count, 10),
	// 	Mfr:   strings.ToUpper(*gpuDeviceInfo.Manufacturer),
	// 	Model: strings.ToUpper(*gpuDeviceInfo.Name),
	// 	Mem:   strconv.FormatInt(*gpuDeviceInfo.MemoryInfo.SizeInMiB, 10),
	// }

	// GPU Struct
	gpuInfo := irs.GpuInfo{}

	// Check Count
	if gpuDeviceInfo.Count != nil {
		gpuInfo.Count = strconv.FormatInt(*gpuDeviceInfo.Count, 10)
	} else {
		gpuInfo.Count = "-1" // Set number values to "-1" if nil
	}

	// Check Manufacturer
	if gpuDeviceInfo.Manufacturer != nil {
		gpuInfo.Mfr = strings.ToUpper(*gpuDeviceInfo.Manufacturer)
	} else {
		gpuInfo.Mfr = "NA" // Set string values to "NA" if nil
	}

	// Check Model
	if gpuDeviceInfo.Name != nil {
		gpuInfo.Model = strings.ToUpper(*gpuDeviceInfo.Name)
	} else {
		gpuInfo.Model = "NA" // Set string values to "NA" if nil
	}

	// Check MemoryInfo
	if gpuDeviceInfo.MemoryInfo != nil && gpuDeviceInfo.MemoryInfo.SizeInMiB != nil {
		gpuInfo.Mem = strconv.FormatInt(*gpuDeviceInfo.MemoryInfo.SizeInMiB, 10)
	} else {
		gpuInfo.Mem = "-1" // Set number values to "-1" if nil
	}

	return gpuInfo
}

// 인스턴스 스펙 정보를 추출함
func ExtractVMSpecInfo(Region string, instanceTypeInfo *ec2.InstanceTypeInfo) irs.VMSpecInfo {
	cblogger.Debugf("ExtractVMSpecInfo : Region:[%s] / SpecName:[%s]", Region, *instanceTypeInfo.InstanceType)
	//cblogger.Debug(instanceTypeInfo)

	//fmt.Println("=======[EC2 응답] ==========")
	//spew.Dump(instanceTypeInfo)
	//fmt.Println("===================")

	vCpuInfo := irs.VCpuInfo{}
	gpuInfoList := []irs.GpuInfo{}

	//리전 정보는 없기 때문에 조회한 리전 정보를 전달 받아서 처리함.
	vmSpecInfo := irs.VMSpecInfo{
		Region: Region,
	}

	//Check Disk Info (Root volume information is only provided in AMI information)
	vmSpecInfo.Disk = "-1"

	// Check VCPU - Count
	if !reflect.ValueOf(instanceTypeInfo.VCpuInfo.DefaultVCpus).IsNil() {
		vCpuInfo.Count = strconv.FormatInt(*instanceTypeInfo.VCpuInfo.DefaultVCpus, 10)
	} else {
		vCpuInfo.Count = "-1"
	}

	// Check VCPU - Clock
	if !reflect.ValueOf(instanceTypeInfo.ProcessorInfo.SustainedClockSpeedInGhz).IsNil() {
		vCpuInfo.Clock = strconv.FormatFloat(*instanceTypeInfo.ProcessorInfo.SustainedClockSpeedInGhz, 'f', 1, 64)
	} else {
		vCpuInfo.Clock = "-1"
	}
	vmSpecInfo.VCpu = vCpuInfo

	//GPU 정보가 있는 인스터스는 GPU 처리
	if !reflect.ValueOf(instanceTypeInfo.GpuInfo).IsNil() {
		for _, curGpu := range instanceTypeInfo.GpuInfo.Gpus {
			cblogger.Debugf("[%s] Gpu Retrieve Specification Information", *curGpu.Name)
			gpuInfo := ExtractGpuInfo(curGpu)
			gpuInfoList = append(gpuInfoList, gpuInfo)
		}
		//cblogger.Debug(gpuInfoList)
	}
	vmSpecInfo.Gpu = gpuInfoList

	if !reflect.ValueOf(instanceTypeInfo.InstanceType).IsNil() {
		vmSpecInfo.Name = *instanceTypeInfo.InstanceType
	} else {
		vmSpecInfo.Name = "NA"
	}

	if !reflect.ValueOf(instanceTypeInfo.MemoryInfo.SizeInMiB).IsNil() {
		vmSpecInfo.Mem = strconv.FormatInt(*instanceTypeInfo.MemoryInfo.SizeInMiB, 10)
	} else {
		vmSpecInfo.Mem = "-1"
	}

	//KeyValue 목록 처리
	keyValueList, errKeyValue := ConvertKeyValueList(instanceTypeInfo)
	if errKeyValue != nil {
		cblogger.Error(errKeyValue)
	}
	/*
		if errKeyValue != nil {
			return irs.VMSpecInfo{}, errKeyValue
		}
	*/
	vmSpecInfo.KeyValueList = keyValueList

	return vmSpecInfo
}

// 해당 Zone의 스펙 ID 목록을 조회함.
func (vmSpecHandler *AwsVmSpecHandler) ListVMSpecAZ(ZoneName string) (map[string]string, error) {
	cblogger.Infof("Start ListVMSpecAZ(ZoneName:[%s])", ZoneName)
	if ZoneName == "" {
		cblogger.Error("Connection information does not contain Zone information.")
		return nil, errors.New("Connection information does not contain Zone information.")
	}

	var mapVmSpecIds map[string]string
	mapVmSpecIds = make(map[string]string)

	//https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeInstanceTypeOfferings.html
	input := &ec2.DescribeInstanceTypeOfferingsInput{
		//[]*string
		LocationType: aws.String("availability-zone"),
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("location"),
				Values: aws.StringSlice([]string{ZoneName}),
			},
		},
		MaxResults: aws.Int64(1000), //5~1000
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmSpecHandler.Region.Zone,
		ResourceType: call.VMSPEC,
		ResourceName: "",
		CloudOSAPI:   "ListVMSpecAZ()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	pageNum := 0
	totCnt := 0
	err := vmSpecHandler.Client.DescribeInstanceTypeOfferingsPages(input,
		func(page *ec2.DescribeInstanceTypeOfferingsOutput, lastPage bool) bool {
			pageNum++
			//fmt.Println(page)
			cblogger.Infof("PageNum : [%d] / Count : [%d] / lastPage : [%v]", pageNum, len(page.InstanceTypeOfferings), lastPage)
			//totCnt = totCnt + len(page.InstanceTypeOfferings)

			for _, specInfo := range page.InstanceTypeOfferings {
				totCnt++
				//cblogger.Infof("===> [%s]", *specInfo.InstanceType)
				mapVmSpecIds[*specInfo.InstanceType] = ""
			}
			return !lastPage
		})
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil { // resp is now filled
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Infof("===> Total Check AZ Spec Count : [%d]", totCnt)
	//cblogger.Debug(mapVmSpecIds)

	return mapVmSpecIds, nil
}

func (vmSpecHandler *AwsVmSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	cblogger.Infof("Start ListVMSpec(Region:[%s] / Zone:[%s])", vmSpecHandler.Region.Region, vmSpecHandler.Region.Zone)

	zoneId := vmSpecHandler.Region.Zone
	cblogger.Infof("Request Zone : [%s]", zoneId)
	if zoneId == "" {
		cblogger.Error("Connection information does not contain Zone information.")
		return nil, errors.New("Connection information does not contain Zone information.")
	}

	mapVmSpecIds, errListVMSpecAZ := vmSpecHandler.ListVMSpecAZ(zoneId)
	if errListVMSpecAZ != nil {
		cblogger.Error(errListVMSpecAZ)
		return nil, errListVMSpecAZ
	}

	var vMSpecInfoList []*irs.VMSpecInfo
	input := &ec2.DescribeInstanceTypesInput{
		//MaxResults: aws.Int64(5),
	}

	/*
		req, resp := vmSpecHandler.Client.DescribeInstanceTypesRequest(input)
		err := req.Send()
		if err != nil { // resp is now filled
			cblogger.Errorf("Unable to get ListVMSpec - %v", err)
			return vMSpecInfoList, err
		}
	*/

	//cblogger.Info(resp)
	//fmt.Println(resp)

	//ExtractVMSpecInfo(Region, resp.InstanceTypes[0])
	//var vMSpecInfoList []*irs.VMSpecInfo
	/*
		for _, curInstance := range resp.InstanceTypes {
			cblogger.Infof("[%s] VM 스펙 정보 조회", *curInstance.InstanceType)

			_, exists := mapVmSpecIds[*curInstance.InstanceType]
			if !exists {
				cblogger.Infof("[%s] 스펙은 [%s] Zone에서 사용할 수 없습니다.", *curInstance.InstanceType, zoneId)
				continue
			}

			//vMSpecInfo := ExtractVMSpecInfo(Region, curInstance)
			//vMSpecInfoList = append(vMSpecInfoList, &vMSpecInfo)
		}
	*/

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmSpecHandler.Region.Zone,
		ResourceType: call.VMSPEC,
		ResourceName: "",
		CloudOSAPI:   "ListVMSpec()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	pageNum := 0
	totCnt := 0
	err := vmSpecHandler.Client.DescribeInstanceTypesPages(input,
		func(page *ec2.DescribeInstanceTypesOutput, lastPage bool) bool {
			pageNum++
			//fmt.Println(page)
			cblogger.Infof("PageNum : [%d] / Count : [%d] / isLastPage : [%v]", pageNum, len(page.InstanceTypes), lastPage)
			//totCnt = totCnt + len(page.InstanceTypes)

			for _, curInstance := range page.InstanceTypes {
				totCnt++
				_, exists := mapVmSpecIds[*curInstance.InstanceType]
				if !exists {
					cblogger.Debugf("The [%s] spec is not supported in the [%s] Zone.", *curInstance.InstanceType, zoneId)
					continue
				}

				vMSpecInfo := ExtractVMSpecInfo(vmSpecHandler.Region.Region, curInstance)
				vMSpecInfoList = append(vMSpecInfoList, &vMSpecInfo)
			}

			return !lastPage
		})

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil { // resp is now filled
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return vMSpecInfoList, err
	}
	callogger.Info(call.String(callLogInfo))
	//cblogger.Debug(vMSpecInfoList)

	//cblogger.Infof("===> Total Check Spec Count : [%d]", totCnt)
	cblogger.Infof("==>In the [%s] AZ, [%d] specs are available from the [%s] region, and [%d] of them can be utilized.", zoneId, vmSpecHandler.Region.Region, totCnt, len(vMSpecInfoList))

	return vMSpecInfoList, nil
}

func (vmSpecHandler *AwsVmSpecHandler) GetVMSpec(Name string) (irs.VMSpecInfo, error) {
	cblogger.Infof("Start GetVMSpec(Region:[%s], Name:[%s])", vmSpecHandler.Region.Region, Name)

	//https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeInstanceTypes.html
	input := &ec2.DescribeInstanceTypesInput{
		//[]*string
		InstanceTypes: []*string{
			aws.String(Name),
		},
	}

	//svc := ec2.New(&sess)
	//svc := ec2.New(&vmSpecHandler.Client, aws.NewConfig().WithRegion("us-west-2"))
	//req, resp := svc.DescribeInstanceTypesRequest(input)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmSpecHandler.Region.Zone,
		ResourceType: call.VMSPEC,
		ResourceName: Name,
		CloudOSAPI:   "DescribeInstanceTypesRequest()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	// Example sending a request using the DescribeInstanceTypesRequest method.
	req, resp := vmSpecHandler.Client.DescribeInstanceTypesRequest(input)
	err := req.Send()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil { // resp is now filled
		cblogger.Errorf("Unable to get GetVMSpec - %v", err)
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		return irs.VMSpecInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	//cblogger.Info(resp)
	//fmt.Println(resp)
	if len(resp.InstanceTypes) < 1 {
		return irs.VMSpecInfo{}, errors.New("Spec not found for " + Name)
	}

	vMSpecInfo := ExtractVMSpecInfo(vmSpecHandler.Region.Region, resp.InstanceTypes[0])

	/*
		//KeyValue 목록 처리
		keyValueList, errKeyValue := ConvertKeyValueList(resp.InstanceTypes[0])
		if errKeyValue != nil {
			return irs.VMSpecInfo{}, errKeyValue
		}
		vMSpecInfo.KeyValueList = keyValueList
	*/

	return vMSpecInfo, nil
}

// AWS의 정보 그대로를 가공 없이 JSON으로 리턴 함.
func (vmSpecHandler *AwsVmSpecHandler) ListOrgVMSpec() (string, error) {
	cblogger.Infof("Start ListOrgVMSpec(Region:[%s])", vmSpecHandler.Region.Region)

	zoneId := vmSpecHandler.Region.Zone
	cblogger.Infof("Zone : %s", zoneId)
	if zoneId == "" {
		cblogger.Error("Connection information does not contain Zone information.")
		return "", errors.New("Connection information does not contain Zone information.")
	}

	mapVmSpecIds, errListVMSpecAZ := vmSpecHandler.ListVMSpecAZ(zoneId)
	if errListVMSpecAZ != nil {
		cblogger.Error(errListVMSpecAZ)
		return "", errListVMSpecAZ
	}

	input := &ec2.DescribeInstanceTypesInput{
		//MaxResults: aws.Int64(5),
	}

	/*
		req, resp := vmSpecHandler.Client.DescribeInstanceTypesRequest(input)
		err := req.Send()
		if err != nil { // resp is now filled
			cblogger.Errorf("Unable to get ListOrgVMSpec - %v", err)
			return "", err
		}
	*/

	//cblogger.Info(resp)
	//fmt.Println(resp)

	//var resp *ec2.DescribeInstanceTypesOutput

	/*
		resp := *ec2.DescribeInstanceTypesOutput{
			InstanceTypes: &[]ec2.InstanceTypeInfo{{}},
		}
	*/

	resp := new(ec2.DescribeInstanceTypesOutput)

	pageNum := 0
	totCnt := 0
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmSpecHandler.Region.Zone,
		ResourceType: call.VMSPEC,
		ResourceName: "",
		CloudOSAPI:   "ListOrgVMSpec()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	err := vmSpecHandler.Client.DescribeInstanceTypesPages(input,
		func(page *ec2.DescribeInstanceTypesOutput, lastPage bool) bool {
			pageNum++
			//fmt.Println(page)
			cblogger.Infof("PageNum : [%d] / Count : [%d] / isLastPage : [%v]", pageNum, len(page.InstanceTypes), lastPage)
			//totCnt = totCnt + len(page.InstanceTypes)

			for _, curInstance := range page.InstanceTypes {
				totCnt++

				_, exists := mapVmSpecIds[*curInstance.InstanceType]
				if !exists {
					cblogger.Debugf("The [%s] spec is not supported in the [%s] Zone.", *curInstance.InstanceType, zoneId)
					continue
				}

				//vMSpecInfo := ExtractVMSpecInfo(Region, curInstance)
				//vMSpecInfoList = append(vMSpecInfoList, &vMSpecInfo)
				resp.InstanceTypes = append(resp.InstanceTypes, curInstance)
			}

			return !lastPage
		})

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil { // resp is now filled
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return "", err
	}
	callogger.Info(call.String(callLogInfo))
	//cblogger.Debug(vMSpecInfoList)

	cblogger.Infof("==>In the [%s] AZ, [%d] specs are available from the [%s] region, and [%d] of them can be utilized.", zoneId, vmSpecHandler.Region.Region, totCnt, len(resp.InstanceTypes))

	//jsonString, errJson := ConvertJsonString(resp.InstanceTypes[0])
	jsonString, errJson := ConvertJsonString(resp)
	if errJson != nil {
		cblogger.Error(errJson)
	}
	return jsonString, errJson
}

// AWS의 정보 그대로를 가공 없이 JSON으로 리턴 함.
func (vmSpecHandler *AwsVmSpecHandler) ListOrgVMSpecOld() (string, error) {
	cblogger.Infof("Start ListOrgVMSpec(Region:[%s])", vmSpecHandler.Region.Region)

	input := &ec2.DescribeInstanceTypesInput{
		//MaxResults: aws.Int64(5),
	}

	req, resp := vmSpecHandler.Client.DescribeInstanceTypesRequest(input)
	err := req.Send()
	if err != nil { // resp is now filled
		cblogger.Errorf("Unable to get ListOrgVMSpec - %v", err)
		return "", err
	}

	//cblogger.Info(resp)
	//fmt.Println(resp)

	//00, errJson := ConvertJsonString(resp.InstanceTypes[0])
	jsonString, errJson := ConvertJsonString(resp)
	if errJson != nil {
		cblogger.Error(errJson)
	}
	return jsonString, errJson
}

// AWS의 정보 그대로를 가공 없이 JSON으로 리턴 함.
func (vmSpecHandler *AwsVmSpecHandler) GetOrgVMSpec(Name string) (string, error) {
	cblogger.Infof("Start GetOrgVMSpec(Region:[%s], Name:[%s])", vmSpecHandler.Region.Region, Name)

	input := &ec2.DescribeInstanceTypesInput{
		InstanceTypes: []*string{
			aws.String(Name),
		},
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmSpecHandler.Region.Zone,
		ResourceType: call.VMSPEC,
		ResourceName: Name,
		CloudOSAPI:   "DescribeInstanceTypesRequest()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	req, resp := vmSpecHandler.Client.DescribeInstanceTypesRequest(input)
	err := req.Send()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil { // resp is now filled
		cblogger.Errorf("Unable to get GetVMSpec - %v", err)
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		return "", err
	}
	callogger.Info(call.String(callLogInfo))

	//cblogger.Info(resp)
	//fmt.Println(resp)
	if len(resp.InstanceTypes) < 1 {
		return "", errors.New("Spec not found for " + Name)
	}

	jsonString, errJson := ConvertJsonString(resp.InstanceTypes[0])
	//jsonString, errJson := ConvertJsonStringNoEscape(resp.InstanceTypes[0])

	if errJson != nil {
		cblogger.Error(errJson)
	}
	return jsonString, errJson
}
