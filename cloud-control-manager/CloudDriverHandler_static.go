//go:build !dyna
// +build !dyna

// Cloud Driver Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.12.

package clouddriverhandler

import (
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"

	alibabadrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/alibaba"
	awsdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws"
	azuredrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure"
	gcpdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/gcp"

	ibmvpcdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibmcloud-vpc"
	mockdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock"
	openstackdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack"
	tencentdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent"

	ktdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ktclassic"
	ktvpcdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ktcloudvpc"
	ncpdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncp"
	nhndrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/nhn"

	cblogger "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"

	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"

	"fmt"
)

var cblog *logrus.Logger

func init() {
	cblog = cblogger.GetLogger("CLOUD-BARISTA")
}

func getCloudDriver(cldDrvInfo dim.CloudDriverInfo) (idrv.CloudDriver, error) {
	cblog.Info("CloudDriverHandler: called getStaticCloudDriver() - " + cldDrvInfo.DriverName)

	var cloudDriver idrv.CloudDriver

	// select driver
	switch cldDrvInfo.ProviderName {
	case "AWS":
		cloudDriver = new(awsdrv.AwsDriver)
	case "AZURE":
		cloudDriver = new(azuredrv.AzureDriver)
	case "GCP":
		cloudDriver = new(gcpdrv.GCPDriver)
	case "ALIBABA":
		cloudDriver = new(alibabadrv.AlibabaDriver)
	case "OPENSTACK":
		cloudDriver = new(openstackdrv.OpenStackDriver)
	case "TENCENT":
		cloudDriver = new(tencentdrv.TencentDriver)
	case "IBM":
		cloudDriver = new(ibmvpcdrv.IbmCloudDriver)
	case "NCP":
		cloudDriver = new(ncpdrv.NcpVpcDriver)
	case "NHN":
		cloudDriver = new(nhndrv.NhnCloudDriver)
	case "KTCLASSIC":
		cloudDriver = new(ktdrv.KtCloudDriver)
	case "KTCLOUDVPC":
		cloudDriver = new(ktvpcdrv.KTCloudVpcDriver)
	case "MOCK":
		cloudDriver = new(mockdrv.MockDriver)

	default:
		errmsg := cldDrvInfo.ProviderName + " is not supported static Cloud Driver!!"
		return cloudDriver, fmt.Errorf(errmsg)
	}

	return cloudDriver, nil
}
