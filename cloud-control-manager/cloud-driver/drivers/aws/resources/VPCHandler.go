// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by devunet@mz.co.kr

package resources

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsVPCHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

type AwsVpcReqInfo struct {
	Name      string
	CidrBlock string // AWS
}

type AwsVpcInfo struct {
	Name      string
	Id        string
	CidrBlock string // AWS
	IsDefault bool   // AWS
	State     string // AWS
}

func (VPCHandler *AwsVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	return irs.VPCInfo{}, nil
}

func (VPCHandler *AwsVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	return nil, nil
}

func (VPCHandler *AwsVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	return irs.VPCInfo{}, nil
}

func (VPCHandler *AwsVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	return false, nil
}
