package ibmcloudvpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/IBM/go-sdk-core/v5/core"
	vpc0230 "github.com/IBM/vpc-go-sdk/0.23.0/vpcv1"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibmcloud-vpc/connect"
	ibms "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibmcloud-vpc/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	"time"
)

type IbmCloudDriver struct{}

const (
	cspTimeout time.Duration = 6000
)

func (IbmCloudDriver) GetDriverVersion() string {
	return "IBM DRIVER Version 1.0"
}
func (IbmCloudDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VPCHandler = true
	drvCapabilityInfo.SecurityHandler = true
	drvCapabilityInfo.KeyPairHandler = true
	drvCapabilityInfo.VNicHandler = false
	drvCapabilityInfo.PublicIPHandler = false
	drvCapabilityInfo.VMHandler = true
	drvCapabilityInfo.VMSpecHandler = true
	drvCapabilityInfo.NLBHandler = true

	return drvCapabilityInfo
}

func (driver *IbmCloudDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	ibms.InitLog()
	err := checkConnectionInfo(connectionInfo)
	if err != nil {
		return nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	// Region & Zone Check
	initVpcService, err := vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
		Authenticator: &core.IamAuthenticator{
			ApiKey: connectionInfo.CredentialInfo.ApiKey,
		},
	})
	if err != nil {
		return nil, err
	}
	var endPoint string
	getRegionOptions := &vpcv1.GetRegionOptions{}
	getRegionOptions.SetName(connectionInfo.RegionInfo.Region)
	region, details, err := initVpcService.GetRegionWithContext(ctx, getRegionOptions)
	if err != nil {
		fmt.Println(details)
		return nil, err
	} else {
		getZoneOptions := &vpcv1.GetRegionZoneOptions{}
		getZoneOptions.SetRegionName(*region.Name)
		getZoneOptions.SetName(connectionInfo.RegionInfo.Zone)
		_, _, err := initVpcService.GetRegionZoneWithContext(ctx, getZoneOptions)
		if err != nil {
			return nil, err
		}
		endPoint = *region.Endpoint + "/v1"
	}
	vpcService, err := vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
		Authenticator: &core.IamAuthenticator{
			ApiKey: connectionInfo.CredentialInfo.ApiKey,
		},
		URL: endPoint,
	})
	vpcService0230, err := vpc0230.NewVpcV1(&vpc0230.VpcV1Options{
		Authenticator: &core.IamAuthenticator{
			ApiKey: connectionInfo.CredentialInfo.ApiKey,
		},
		URL: endPoint,
	})
	if err != nil {
		return nil, err
	}
	iConn := connect.IbmCloudConnection{
		CredentialInfo: connectionInfo.CredentialInfo,
		Region:         connectionInfo.RegionInfo,
		VpcService:     vpcService,
		VpcService0230: vpcService0230,
		Ctx:            ctx,
	}
	return &iConn, nil
}

func checkConnectionInfo(connectionInfo idrv.ConnectionInfo) error {
	if connectionInfo.CredentialInfo.ApiKey == "" {
		return errors.New("not exist ApiKey")
	}
	if connectionInfo.RegionInfo.Region == "" {
		return errors.New("not exist Region")
	}
	if connectionInfo.RegionInfo.Zone == "" {
		return errors.New("not exist Zone")
	}
	return nil
}
