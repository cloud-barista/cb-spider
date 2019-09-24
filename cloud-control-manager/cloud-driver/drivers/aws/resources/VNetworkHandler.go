// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by powerkim@etri.re.kr, 2019.06.

//VPC 처리
package resources

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-driver/interfaces/resources"
)

type AwsVNetworkHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

func (vNetworkHandler *AwsVNetworkHandler) ListVNetwork() ([]*irs.VNetworkInfo, error) {
	cblogger.Debug("Start")
	return nil, nil
}

func (vNetworkHandler *AwsVNetworkHandler) CreateVNetwork(vNetworkReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {
	cblogger.Info(vNetworkReqInfo)
	return irs.VNetworkInfo{}, nil
}

func (vNetworkHandler *AwsVNetworkHandler) GetVNetwork(vNetworkID string) (irs.VNetworkInfo, error) {
	cblogger.Info("vNetworkID : [%s]", vNetworkID)
	//result, err := vNetworkHandler.Client.DescribeKeyPairs(input)
	return irs.VNetworkInfo{}, nil
}

func (vNetworkHandler *AwsVNetworkHandler) DeleteVNetwork(vNetworkID string) (bool, error) {
	cblogger.Info("vNetworkID : [%s]", vNetworkID)
	return true, nil
}
