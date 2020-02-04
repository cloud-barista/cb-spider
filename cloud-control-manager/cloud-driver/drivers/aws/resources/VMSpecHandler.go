package resources

import (
	"errors"
	"reflect"
	"strconv"

	//sdk2 "github.com/aws/aws-sdk-go-v2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

//https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeInstanceTypes
type AwsVmSpecHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

func ExtractGpuInfo(gpuDeviceInfo *ec2.GpuDeviceInfo) irs.GpuInfo {
	cblogger.Info(gpuDeviceInfo)
	//cblogger.Info("================")
	spew.Dump(gpuDeviceInfo)

	gpuInfo := irs.GpuInfo{
		Count: strconv.FormatInt(*gpuDeviceInfo.Count, 10),
		Mfr:   *gpuDeviceInfo.Manufacturer,
		Model: *gpuDeviceInfo.Name,
		Mem:   strconv.FormatInt(*gpuDeviceInfo.MemoryInfo.SizeInMiB, 10),
	}

	return gpuInfo
}

//인스턴스 스펙 정보를 추출함
func ExtractVMSpecInfo(Region string, instanceTypeInfo *ec2.InstanceTypeInfo) irs.VMSpecInfo {
	cblogger.Infof("ExtractVMSpecInfo : Region:[%s] / SpecName:[%s]", Region, *instanceTypeInfo.InstanceType)
	spew.Dump(instanceTypeInfo)

	vCpuInfo := irs.VCpuInfo{}
	gpuInfoList := []irs.GpuInfo{}

	//리전 정보는 없기 때문에 조회한 리전 정보를 전달 받아서 처리함.
	vmSpecInfo := irs.VMSpecInfo{
		Region: Region,
	}

	//VCPU 정보 처리 - Count
	if !reflect.ValueOf(instanceTypeInfo.VCpuInfo.DefaultVCpus).IsNil() {
		vCpuInfo.Count = strconv.FormatInt(*instanceTypeInfo.VCpuInfo.DefaultVCpus, 10)
	}

	//VCPU 정보 처리 - Clock
	if !reflect.ValueOf(instanceTypeInfo.ProcessorInfo.SustainedClockSpeedInGhz).IsNil() {
		vCpuInfo.Clock = strconv.FormatFloat(*instanceTypeInfo.ProcessorInfo.SustainedClockSpeedInGhz, 'f', 1, 64)
	}
	vmSpecInfo.VCpu = vCpuInfo

	//GPU 정보가 있는 인스터스는 GPU 처리
	if !reflect.ValueOf(instanceTypeInfo.GpuInfo).IsNil() {
		for _, curGpu := range instanceTypeInfo.GpuInfo.Gpus {
			cblogger.Infof("[%s] Gpu 스펙 정보 조회", *curGpu.Name)
			gpuInfo := ExtractGpuInfo(curGpu)
			gpuInfoList = append(gpuInfoList, gpuInfo)
		}
		spew.Dump(gpuInfoList)
	}
	vmSpecInfo.Gpu = gpuInfoList

	if !reflect.ValueOf(instanceTypeInfo.InstanceType).IsNil() {
		vmSpecInfo.Name = *instanceTypeInfo.InstanceType
	}

	if !reflect.ValueOf(instanceTypeInfo.MemoryInfo.SizeInMiB).IsNil() {
		vmSpecInfo.Mem = strconv.FormatInt(*instanceTypeInfo.MemoryInfo.SizeInMiB, 10)
	}


	keyValueList := []irs.KeyValue{
		{Key: "Domain", Value: *allocRes.Domain},
		{Key: "PublicIpv4Pool", Value: *allocRes.PublicIpv4Pool},
		{Key: "AllocationId", Value: *allocRes.AllocationId},
	}

	return vmSpecInfo
}

func (vmSpecHandler *AwsVmSpecHandler) ListVMSpec(Region string) ([]*irs.VMSpecInfo, error) {

	cblogger.Infof("Start ListVMSpec(Region:[%s])", Region)

	var vMSpecInfoList []*irs.VMSpecInfo
	/*
		//https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeInstanceTypes.html
		//    LocationType LocationType `type:"string" enum:"true"`
		input := &ec2.DescribeInstanceTypeOfferingsInput{
			//[]*string
			LocationType: aws.String("region"),
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("location"),
					Values: aws.StringSlice([]string{"ap-northeast-2"}),
				},
			},
			//MaxResults: aws.Int64(5),
		}

		// Example sending a request using the DescribeInstanceTypesRequest method.
		//req, resp := vmSpecHandler.Client.DescribeInstanceTypesRequest(nil)
		req, resp := vmSpecHandler.Client.DescribeInstanceTypeOfferingsRequest(input)
		err := req.Send()
		if err != nil { // resp is now filled
			cblogger.Errorf("Unable to get ListVMSpec - %v", err)
			return vMSpecInfoList, err
		}
	*/

	input := &ec2.DescribeInstanceTypesInput{
		//MaxResults: aws.Int64(5),
	}

	req, resp := vmSpecHandler.Client.DescribeInstanceTypesRequest(input)
	err := req.Send()
	if err != nil { // resp is now filled
		cblogger.Errorf("Unable to get ListVMSpec - %v", err)
		return vMSpecInfoList, err
	}

	cblogger.Info(resp)
	//fmt.Println(resp)

	//ExtractVMSpecInfo(Region, resp.InstanceTypes[0])
	//var vMSpecInfoList []*irs.VMSpecInfo
	for _, curInstance := range resp.InstanceTypes {
		cblogger.Infof("[%s] VM 스펙 정보 조회", *curInstance.InstanceType)
		vMSpecInfo := ExtractVMSpecInfo(Region, curInstance)
		vMSpecInfoList = append(vMSpecInfoList, &vMSpecInfo)
	}
	spew.Dump(vMSpecInfoList)

	return vMSpecInfoList, nil
}

func (vmSpecHandler *AwsVmSpecHandler) GetVMSpec(Region string, Name string) (irs.VMSpecInfo, error) {
	cblogger.Infof("Start GetVMSpec(Region:[%s], Name:[%s])", Region, Name)

	//https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeInstanceTypes.html
	input := &ec2.DescribeInstanceTypesInput{
		//[]*string
		InstanceTypes: []*string{
			aws.String(Name),
		},
	}

	// Example sending a request using the DescribeInstanceTypesRequest method.
	//req, resp := vmSpecHandler.Client.DescribeInstanceTypesRequest(nil)
	req, resp := vmSpecHandler.Client.DescribeInstanceTypesRequest(input)
	err := req.Send()
	if err != nil { // resp is now filled
		cblogger.Errorf("Unable to get GetVMSpec - %v", err)
		return irs.VMSpecInfo{}, err
	}

	cblogger.Info(resp)
	//fmt.Println(resp)
	if len(resp.InstanceTypes) < 1 {
		return irs.VMSpecInfo{}, errors.New(Name + "에 해당하는 Spec 정보를 찾을 수 없습니다.")
	}

	vMSpecInfo := ExtractVMSpecInfo(Region, resp.InstanceTypes[0])

	//KeyValue 목록 처리
	keyValueList, errKeyValue := ConvertKeyValueList(resp.InstanceTypes[0])
	if errKeyValue != nil {
		return irs.VMSpecInfo{}, errKeyValue
	}
	vMSpecInfo.KeyValueList = keyValueList

	return vMSpecInfo, nil
}

// AWS의 정보 그대로를 가공 없이 JSON으로 리턴 함.
func (vmSpecHandler *AwsVmSpecHandler) ListOrgVMSpec(Region string) (string, error) {
	cblogger.Infof("Start ListOrgVMSpec(Region:[%s])", Region)

	input := &ec2.DescribeInstanceTypesInput{
		//MaxResults: aws.Int64(5),
	}

	req, resp := vmSpecHandler.Client.DescribeInstanceTypesRequest(input)
	err := req.Send()
	if err != nil { // resp is now filled
		cblogger.Errorf("Unable to get ListOrgVMSpec - %v", err)
		return "", err
	}

	cblogger.Info(resp)
	//fmt.Println(resp)

	jsonString, errJson := ConvertJsonString(resp.InstanceTypes[0])
	if errJson != nil {
		cblogger.Error(errJson)
	}
	return jsonString, errJson
}

// AWS의 정보 그대로를 가공 없이 JSON으로 리턴 함.
func (vmSpecHandler *AwsVmSpecHandler) GetOrgVMSpec(Region string, Name string) (string, error) {
	cblogger.Infof("Start GetOrgVMSpec(Region:[%s], Name:[%s])", Region, Name)

	input := &ec2.DescribeInstanceTypesInput{
		InstanceTypes: []*string{
			aws.String(Name),
		},
	}

	req, resp := vmSpecHandler.Client.DescribeInstanceTypesRequest(input)
	err := req.Send()
	if err != nil { // resp is now filled
		cblogger.Errorf("Unable to get GetVMSpec - %v", err)
		return "", err
	}

	cblogger.Info(resp)
	//fmt.Println(resp)
	if len(resp.InstanceTypes) < 1 {
		return "", errors.New(Name + "에 해당하는 Spec 정보를 찾을 수 없습니다.")
	}

	jsonString, errJson := ConvertJsonString(resp.InstanceTypes[0])
	if errJson != nil {
		cblogger.Error(errJson)
	}
	return jsonString, errJson
}
