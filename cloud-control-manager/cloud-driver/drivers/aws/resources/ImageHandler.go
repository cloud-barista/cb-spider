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

type AwsImageHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

func (imageHandler *AwsImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {

	return irs.ImageInfo{}, nil
}

func (imageHandler *AwsImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	return nil, nil
}

func (imageHandler *AwsImageHandler) GetImage(imageID string) (irs.ImageInfo, error) {
	return irs.ImageInfo{}, nil
}

func (imageHandler *AwsImageHandler) DeleteImage(imageID string) (bool, error) {
	return true, nil
}
