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
	clouditdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit"
	dockerdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/docker"
	gcpdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/gcp"
	mockdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock"
	openstackdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack"
	tencentdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent"
	ibmvpcdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibmcloud-vpc"

	// ncpdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncp" // NCP
	// ncpvpcdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncpvpc" // NCP-VPC

	"github.com/cloud-barista/cb-store/config"
	"github.com/sirupsen/logrus"

	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"

	"fmt"
)

var cblog *logrus.Logger

func init() {
	cblog = config.Cblogger
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
	case "CLOUDIT":
		cloudDriver = new(clouditdrv.ClouditDriver)
	case "DOCKER":
		cloudDriver = new(dockerdrv.DockerDriver)
	case "TENCENT":
		cloudDriver = new(tencentdrv.TencentDriver)
	// case "NCP": // NCP
	//  cloudDriver = new(ncpdrv.NcpDriver) // NCP
	// case "NCPVPC": // NCP-VPC
	//  cloudDriver = new(ncpvpcdrv.NcpVpcDriver) // NCP-VPC
	case "MOCK":
		cloudDriver = new(mockdrv.MockDriver)
	case "IBM":
		cloudDriver = new(ibmvpcdrv.IbmCloudDriver)

	default:
		errmsg := cldDrvInfo.ProviderName + " is not supported static Cloud Driver!!"
		return cloudDriver, fmt.Errorf(errmsg)
	}

	return cloudDriver, nil
}
