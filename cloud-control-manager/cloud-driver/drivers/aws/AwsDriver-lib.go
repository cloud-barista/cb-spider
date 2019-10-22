// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by powerkim@etri.re.kr, 2019.06.

package main

//package aws

import (
	"C"

	acon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws/connect"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"

	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	//icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect/AwsNewIfCloudConnect"
	//icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect/connect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)
import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

type AwsDriver struct {
}

func (AwsDriver) GetDriverVersion() string {
	return "TEST AWS DRIVER Version 1.0"
}

func (AwsDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VNetworkHandler = true
	drvCapabilityInfo.SecurityHandler = true
	drvCapabilityInfo.KeyPairHandler = true
	drvCapabilityInfo.VNicHandler = true
	drvCapabilityInfo.PublicIPHandler = true
	drvCapabilityInfo.VMHandler = true

	return drvCapabilityInfo
}

//func getVMClient(regionInfo idrv.RegionInfo) (*ec2.EC2, error) {
func getVMClient(connectionInfo idrv.ConnectionInfo) (*ec2.EC2, error) {

	// setup Region
	fmt.Println("AwsDriver : getVMClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")
	//fmt.Println("전달 받은 커넥션 정보")
	//spew.Dump(connectionInfo)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(connectionInfo.RegionInfo.Region),
		//Region:      aws.String("ap-northeast-2"),
		Credentials: credentials.NewStaticCredentials(connectionInfo.CredentialInfo.ClientId, connectionInfo.CredentialInfo.ClientSecret, "")},
	)
	if err != nil {
		fmt.Println("Could not create aws New Session", err)
		return nil, err
	}

	// Create EC2 service client
	svc := ec2.New(sess)
	if err != nil {
		fmt.Println("Could not create EC2 service client", err)
		return nil, err
	}

	return svc, nil
}

func (driver *AwsDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	//fmt.Println("ConnectCloud의 전달 받은 idrv.ConnectionInfo 정보")
	//spew.Dump(connectionInfo)

	// sample code, do not user like this^^
	//var iConn icon.CloudConnection
	vmClient, err := getVMClient(connectionInfo)
	//vmClient, err := getVMClient(connectionInfo.RegionInfo)
	if err != nil {
		return nil, err
	}

	//iConn = acon.AwsCloudConnection{}
	iConn := acon.AwsCloudConnection{
		Region:        connectionInfo.RegionInfo,
		VMClient:      vmClient,
		KeyPairClient: vmClient,

		VNetworkClient: vmClient,
		VNicClient:     vmClient,
		ImageClient:    vmClient,
		PublicIPClient: vmClient,
		SecurityClient: vmClient,
	}

	return &iConn, nil // return type: (icon.CloudConnection, error)
}

/*
func (AwsDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.
	// sample code, do not user like this^^
	var iConn icon.CloudConnection
	iConn = acon.AwsCloudConnection{}
	return iConn, nil // return type: (icon.CloudConnection, error)
}
*/
var CloudDriver AwsDriver
