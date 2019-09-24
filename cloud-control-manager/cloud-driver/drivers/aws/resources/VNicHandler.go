// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by powerkim@etri.re.kr, 2019.06.

package resources

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-driver/interfaces/resources"
)

type AwsVNicHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

func (vNicHandler *AwsVNicHandler) CreateVNic(vNicReqInfo irs.VNicReqInfo) (irs.VNicInfo, error) {
	return irs.VNicInfo{}, nil
}

func (vNicHandler *AwsVNicHandler) ListVNic() ([]*irs.VNicInfo, error) {
	return nil, nil
}

func (vNicHandler *AwsVNicHandler) GetVNic(vNicID string) (irs.VNicInfo, error) {
	return irs.VNicInfo{}, nil
}

func (vNicHandler *AwsVNicHandler) DeleteVNic(vNicID string) (bool, error) {
	return true, nil
}
