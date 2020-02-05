package resources

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

//https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeInstanceTypes
type AlibabaVmSpecHandler struct {
	Region idrv.RegionInfo
	Client *ecs.Client
}

func (vmSpecHandler *AlibabaVmSpecHandler) ListVMSpec(Region string) ([]*irs.VMSpecInfo, error) {
	cblogger.Infof("Start ListVMSpec(Region:[%s])", Region)
	var vMSpecInfoList []*irs.VMSpecInfo

	return vMSpecInfoList, nil
}

func (vmSpecHandler *AlibabaVmSpecHandler) GetVMSpec(Region string, Name string) (irs.VMSpecInfo, error) {
	cblogger.Infof("Start GetVMSpec(Region:[%s], Name:[%s])", Region, Name)

	return irs.VMSpecInfo{}, nil
}

// AWS의 정보 그대로를 가공 없이 JSON으로 리턴 함.
func (vmSpecHandler *AlibabaVmSpecHandler) ListOrgVMSpec(Region string) (string, error) {
	cblogger.Infof("Start ListOrgVMSpec(Region:[%s])", Region)

	return "", nil
}

// AWS의 정보 그대로를 가공 없이 JSON으로 리턴 함.
func (vmSpecHandler *AlibabaVmSpecHandler) GetOrgVMSpec(Region string, Name string) (string, error) {
	cblogger.Infof("Start GetOrgVMSpec(Region:[%s], Name:[%s])", Region, Name)
	return "", nil
}
