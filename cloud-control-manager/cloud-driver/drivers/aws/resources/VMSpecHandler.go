package resources

//20211104 개선안 I에 의해 Region 파라메터 사용 로직 제거 - AWS는 Region이 아닌 Zone 기반으로 검색되며 Region은 로그용으로만 사용하고 있어서 세션 정보로 대체함.
import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// aws-sdk-go-v2 is used for DescribeInstanceTypes because NeuronInfo
	// (AWS Inferentia/Trainium) does not exist in the v1 SDK version (v1.39.4)
	// pinned for the rest of this driver. The JSON field names of v2
	// InstanceTypeInfo are the same as v1, so ListOrgVMSpec/GetOrgVMSpec keep
	// the same output format with the additional accelerator fields.
	credentialsv2 "github.com/aws/aws-sdk-go-v2/credentials"
	ec2v2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	typesv2 "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeInstanceTypes
type AwsVmSpecHandler struct {
	Region         idrv.RegionInfo
	Client         *ec2.EC2
	CredentialInfo idrv.CredentialInfo
}

// orgVMSpecList keeps the same JSON format as v1 ec2.DescribeInstanceTypesOutput.
type orgVMSpecList struct {
	InstanceTypes []typesv2.InstanceTypeInfo
	NextToken     *string
}

func (vmSpecHandler *AwsVmSpecHandler) getV2Client() *ec2v2.Client {
	// CB-Spider fills an unset StsToken with the literal string "Not set".
	// (see RemoveDefaultPublicIP in PublicIPHandler.go)
	stsToken := vmSpecHandler.CredentialInfo.StsToken
	if stsToken == "Not set" {
		stsToken = ""
	}
	return ec2v2.New(ec2v2.Options{
		Region: vmSpecHandler.Region.Region,
		Credentials: credentialsv2.NewStaticCredentialsProvider(
			vmSpecHandler.CredentialInfo.ClientId, vmSpecHandler.CredentialInfo.ClientSecret, stsToken),
	})
}

// describeInstanceTypes calls DescribeInstanceTypes(v2) for the given types (all types if empty).
func (vmSpecHandler *AwsVmSpecHandler) describeInstanceTypes(instanceTypes []string) ([]typesv2.InstanceTypeInfo, error) {
	input := &ec2v2.DescribeInstanceTypesInput{}
	for _, it := range instanceTypes {
		input.InstanceTypes = append(input.InstanceTypes, typesv2.InstanceType(it))
	}

	var result []typesv2.InstanceTypeInfo
	pageNum := 0
	paginator := ec2v2.NewDescribeInstanceTypesPaginator(vmSpecHandler.getV2Client(), input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		pageNum++
		cblogger.Infof("PageNum : [%d] / Count : [%d]", pageNum, len(page.InstanceTypes))
		result = append(result, page.InstanceTypes...)
	}
	return result, nil
}

func ExtractGpuInfo(gpuDeviceInfo typesv2.GpuDeviceInfo) irs.GpuInfo {
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
		gpuInfo.Count = strconv.FormatInt(int64(*gpuDeviceInfo.Count), 10)
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

	// Check GPU MemoryInfo and transform MiB to GB
	if gpuDeviceInfo.MemoryInfo != nil && gpuDeviceInfo.MemoryInfo.SizeInMiB != nil {
		// Convert MiB to GB
		gpuInfo.MemSizeGB = irs.ConvertMiBToGBInt64(int64(*gpuDeviceInfo.MemoryInfo.SizeInMiB))

		// Calculate TotalMemSizeGB
		if gpuDeviceInfo.Count != nil {
			totalMiB := int64(*gpuDeviceInfo.MemoryInfo.SizeInMiB) * int64(*gpuDeviceInfo.Count)
			gpuInfo.TotalMemSizeGB = irs.ConvertMiBToGBInt64(totalMiB)
		} else {
			gpuInfo.TotalMemSizeGB = "-1"
		}
	} else {
		gpuInfo.MemSizeGB = "-1" // Set number values to "-1" if nil
		gpuInfo.TotalMemSizeGB = "-1"
	}

	return gpuInfo
}

// 인스턴스 스펙 정보를 추출함
func ExtractVMSpecInfo(Region string, instanceTypeInfo typesv2.InstanceTypeInfo) irs.VMSpecInfo {
	cblogger.Debugf("ExtractVMSpecInfo : Region:[%s] / SpecName:[%s]", Region, instanceTypeInfo.InstanceType)

	vCpuInfo := irs.VCpuInfo{}
	gpuInfoList := []irs.GpuInfo{}

	//리전 정보는 없기 때문에 조회한 리전 정보를 전달 받아서 처리함.
	vmSpecInfo := irs.VMSpecInfo{
		Region: Region,
	}

	//Check Disk Info (Root volume information is only provided in AMI information)
	vmSpecInfo.DiskSizeGB = "-1"

	// Check VCPU - Count
	if instanceTypeInfo.VCpuInfo != nil && instanceTypeInfo.VCpuInfo.DefaultVCpus != nil {
		vCpuInfo.Count = strconv.FormatInt(int64(*instanceTypeInfo.VCpuInfo.DefaultVCpus), 10)
	} else {
		vCpuInfo.Count = "-1"
	}

	// Check VCPU - Clock
	if instanceTypeInfo.ProcessorInfo != nil && instanceTypeInfo.ProcessorInfo.SustainedClockSpeedInGhz != nil {
		vCpuInfo.ClockGHz = strconv.FormatFloat(*instanceTypeInfo.ProcessorInfo.SustainedClockSpeedInGhz, 'f', 1, 64)
	} else {
		vCpuInfo.ClockGHz = "-1"
	}
	vmSpecInfo.VCpu = vCpuInfo

	//GPU 정보가 있는 인스터스는 GPU 처리
	if instanceTypeInfo.GpuInfo != nil {
		for _, curGpu := range instanceTypeInfo.GpuInfo.Gpus {
			cblogger.Debugf("[%s] Gpu Retrieve Specification Information", aws.StringValue(curGpu.Name))
			gpuInfo := ExtractGpuInfo(curGpu)
			gpuInfoList = append(gpuInfoList, gpuInfo)
		}
	}
	vmSpecInfo.Gpu = gpuInfoList
	vmSpecInfo.Accelerator = ExtractAcceleratorInfo(instanceTypeInfo)

	if instanceTypeInfo.InstanceType != "" {
		vmSpecInfo.Name = string(instanceTypeInfo.InstanceType)
	} else {
		vmSpecInfo.Name = "NA"
	}

	if instanceTypeInfo.MemoryInfo != nil && instanceTypeInfo.MemoryInfo.SizeInMiB != nil {
		vmSpecInfo.MemSizeMiB = strconv.FormatInt(*instanceTypeInfo.MemoryInfo.SizeInMiB, 10)
	} else {
		vmSpecInfo.MemSizeMiB = "-1"
	}

	vmSpecInfo.KeyValueList = irs.StructToKeyValueList(instanceTypeInfo)

	return vmSpecInfo
}

// ExtractAcceleratorInfo builds the AI accelerator list from the CSP API fields:
//   - GpuInfo                  => GPU (NVIDIA, AMD, ...)
//   - NeuronInfo               => NPU (AWS Inferentia, Trainium)
//   - InferenceAcceleratorInfo => NPU (Qualcomm AI100, ...); skipped when already reported in NeuronInfo
func ExtractAcceleratorInfo(instanceTypeInfo typesv2.InstanceTypeInfo) []irs.AcceleratorInfo {
	accList := []irs.AcceleratorInfo{}

	if instanceTypeInfo.GpuInfo != nil {
		for _, gpu := range instanceTypeInfo.GpuInfo.Gpus {
			var memMiB *int32
			if gpu.MemoryInfo != nil {
				memMiB = gpu.MemoryInfo.SizeInMiB
			}
			accList = append(accList, newAcceleratorInfo(irs.AcceleratorGPU, gpu.Manufacturer, gpu.Name, gpu.Count, memMiB))
		}
	}

	neuronNames := map[string]bool{}
	if instanceTypeInfo.NeuronInfo != nil {
		for _, dev := range instanceTypeInfo.NeuronInfo.NeuronDevices {
			var memMiB *int32
			if dev.MemoryInfo != nil {
				memMiB = dev.MemoryInfo.SizeInMiB
			}
			neuronNames[aws.StringValue(dev.Name)] = true
			accList = append(accList, newAcceleratorInfo(irs.AcceleratorNPU, aws.String("AWS"), dev.Name, dev.Count, memMiB))
		}
	}

	if instanceTypeInfo.InferenceAcceleratorInfo != nil {
		for _, acc := range instanceTypeInfo.InferenceAcceleratorInfo.Accelerators {
			if neuronNames[aws.StringValue(acc.Name)] {
				continue
			}
			var memMiB *int32
			if acc.MemoryInfo != nil {
				memMiB = acc.MemoryInfo.SizeInMiB
			}
			accList = append(accList, newAcceleratorInfo(irs.AcceleratorNPU, acc.Manufacturer, acc.Name, acc.Count, memMiB))
		}
	}

	return accList
}

func newAcceleratorInfo(accType irs.AcceleratorType, mfr, model *string, count, memMiB *int32) irs.AcceleratorInfo {
	accInfo := irs.AcceleratorInfo{
		Type:           accType,
		Mfr:            "NA",
		Model:          "NA",
		Count:          "-1",
		MemSizeGB:      "-1",
		TotalMemSizeGB: "-1",
	}
	if mfr != nil {
		accInfo.Mfr = *mfr
	}
	if model != nil {
		accInfo.Model = *model
	}
	if count != nil {
		accInfo.Count = strconv.FormatInt(int64(*count), 10)
	}
	if memMiB != nil {
		accInfo.MemSizeGB = irs.ConvertMiBToGBInt64(int64(*memMiB))
		if count != nil {
			accInfo.TotalMemSizeGB = irs.ConvertMiBToGBInt64(int64(*memMiB) * int64(*count))
		}
	}
	return accInfo
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

	instanceTypes, err := vmSpecHandler.describeInstanceTypes(nil)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	var vMSpecInfoList []*irs.VMSpecInfo
	for _, curInstance := range instanceTypes {
		if _, exists := mapVmSpecIds[string(curInstance.InstanceType)]; !exists {
			cblogger.Debugf("The [%s] spec is not supported in the [%s] Zone.", curInstance.InstanceType, zoneId)
			continue
		}
		vMSpecInfo := ExtractVMSpecInfo(vmSpecHandler.Region.Region, curInstance)
		vMSpecInfoList = append(vMSpecInfoList, &vMSpecInfo)
	}

	cblogger.Infof("==>In the [%s] AZ, [%d] specs are available from the [%s] region, and [%d] of them can be utilized.", zoneId, len(instanceTypes), vmSpecHandler.Region.Region, len(vMSpecInfoList))

	return vMSpecInfoList, nil
}

func (vmSpecHandler *AwsVmSpecHandler) GetVMSpec(Name string) (irs.VMSpecInfo, error) {
	cblogger.Infof("Start GetVMSpec(Region:[%s], Name:[%s])", vmSpecHandler.Region.Region, Name)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmSpecHandler.Region.Zone,
		ResourceType: call.VMSPEC,
		ResourceName: Name,
		CloudOSAPI:   "DescribeInstanceTypes()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	//https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeInstanceTypes.html
	instanceTypes, err := vmSpecHandler.describeInstanceTypes([]string{Name})
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		cblogger.Errorf("Unable to get GetVMSpec - %v", err)
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		return irs.VMSpecInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	if len(instanceTypes) < 1 {
		return irs.VMSpecInfo{}, errors.New("Spec not found for " + Name)
	}

	return ExtractVMSpecInfo(vmSpecHandler.Region.Region, instanceTypes[0]), nil
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

	instanceTypes, err := vmSpecHandler.describeInstanceTypes(nil)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return "", err
	}
	callogger.Info(call.String(callLogInfo))

	resp := orgVMSpecList{}
	for _, curInstance := range instanceTypes {
		if _, exists := mapVmSpecIds[string(curInstance.InstanceType)]; !exists {
			cblogger.Debugf("The [%s] spec is not supported in the [%s] Zone.", curInstance.InstanceType, zoneId)
			continue
		}
		resp.InstanceTypes = append(resp.InstanceTypes, curInstance)
	}

	cblogger.Infof("==>In the [%s] AZ, [%d] specs are available from the [%s] region, and [%d] of them can be utilized.", zoneId, len(instanceTypes), vmSpecHandler.Region.Region, len(resp.InstanceTypes))

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

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmSpecHandler.Region.Zone,
		ResourceType: call.VMSPEC,
		ResourceName: Name,
		CloudOSAPI:   "DescribeInstanceTypes()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	instanceTypes, err := vmSpecHandler.describeInstanceTypes([]string{Name})
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		cblogger.Errorf("Unable to get GetVMSpec - %v", err)
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		return "", err
	}
	callogger.Info(call.String(callLogInfo))

	if len(instanceTypes) < 1 {
		return "", errors.New("Spec not found for " + Name)
	}

	jsonString, errJson := ConvertJsonString(instanceTypes[0])
	if errJson != nil {
		cblogger.Error(errJson)
	}
	return jsonString, errJson
}
