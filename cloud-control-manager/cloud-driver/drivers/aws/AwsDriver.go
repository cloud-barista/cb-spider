// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by CB-Spider Team, 2019.06.
// https://docs.aws.amazon.com/sdk-for-go/api/service/elbv2

package aws

import (
	acon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws/connect"
	ars "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"

	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	//icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect/AwsNewIfCloudConnect"
	//icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect/connect"

	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/pricing"
)

type AwsDriver struct {
}

func (AwsDriver) GetDriverVersion() string {
	return "AWS DRIVER Version 1.0"
}

func (AwsDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
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
	drvCapabilityInfo.RegionZoneHandler = true
	drvCapabilityInfo.PriceInfoHandler = true

	return drvCapabilityInfo
}

// func getVMClient(regionInfo idrv.RegionInfo) (*ec2.EC2, error) {
func getVMClient(connectionInfo idrv.ConnectionInfo) (*ec2.EC2, error) {

	// setup Region
	// fmt.Println("AwsDriver : getVMClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")
	// fmt.Println("AwsDriver : getVMClient() - Zone : [" + connectionInfo.RegionInfo.Zone + "]")
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

// 로드밸런서 처리를 위한 ELB 클라이언트 획득
func getNLBClient(connectionInfo idrv.ConnectionInfo) (*elbv2.ELBV2, error) {
	//func getNLBClient(connectionInfo idrv.ConnectionInfo) (*elb.ELB, error) {

	// setup Region
	// fmt.Println("AwsDriver : getVMClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")
	// fmt.Println("AwsDriver : getVMClient() - Zone : [" + connectionInfo.RegionInfo.Zone + "]")
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

	//svc := elb.New(sess)
	// Create ELBv2 service client
	svc := elbv2.New(sess)
	if err != nil {
		fmt.Println("Could not create ELBv2 service client", err)
		return nil, err
	}

	return svc, nil
}

// EKS 처리를 위한 EKS 클라이언트 획득
func getEKSClient(connectionInfo idrv.ConnectionInfo) (*eks.EKS, error) {

	// setup Region
	// fmt.Println("AwsDriver : getEKSClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")
	// fmt.Println("AwsDriver : getEKSClient() - Zone : [" + connectionInfo.RegionInfo.Zone + "]")
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

	svc := eks.New(sess)
	if err != nil {
		fmt.Println("Could not create eks service client", err)
		return nil, err
	}

	return svc, nil
}

// Iam 처리를 위한 iam 클라이언트 획득
func getIamClient(connectionInfo idrv.ConnectionInfo) (*iam.IAM, error) {
	// setup Region
	// fmt.Println("AwsDriver : getIamClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")
	// fmt.Println("AwsDriver : getIamClient() - Zone : [" + connectionInfo.RegionInfo.Zone + "]")
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

	svc := iam.New(sess)
	if err != nil {
		fmt.Println("Could not create iam service client", err)
		return nil, err
	}

	return svc, nil
}

// AutoScaling 처리를 위한 autoScaling 클라이언트 획득
func getAutoScalingClient(connectionInfo idrv.ConnectionInfo) (*autoscaling.AutoScaling, error) {

	// setup Region
	// fmt.Println("AwsDriver : getAutoScalingClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")
	// fmt.Println("AwsDriver : getAutoScalingClient() - Zone : [" + connectionInfo.RegionInfo.Zone + "]")
	//fmt.Println("전달 받은 커넥션 정보")
	//spew.Dump(connectionInfo)

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(connectionInfo.RegionInfo.Region),
		Credentials: credentials.NewStaticCredentials(connectionInfo.CredentialInfo.ClientId, connectionInfo.CredentialInfo.ClientSecret, "")},
	)
	if err != nil {
		fmt.Println("Could not create aws New Session", err)
		return nil, err
	}

	//svc := elb.New(sess)
	// Create ELBv2 service client
	svc := autoscaling.New(sess)
	if err != nil {
		fmt.Println("Could not create autoscaling service client", err)
		return nil, err
	}

	return svc, nil
}

func (driver *AwsDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	// Initialize Logger
	ars.InitLog()

	//fmt.Println("ConnectCloud의 전달 받은 idrv.ConnectionInfo 정보")
	//spew.Dump(connectionInfo)

	// sample code, do not user like this^^
	//var iConn icon.CloudConnection
	vmClient, err := getVMClient(connectionInfo)
	nlbClient, err := getNLBClient(connectionInfo)
	eksClient, err := getEKSClient(connectionInfo)
	iamClient, err := getIamClient(connectionInfo)
	pricingClient, err := getPricingClient(connectionInfo)
	autoScalingClient, err := getAutoScalingClient(connectionInfo)
	//vmClient, err := getVMClient(connectionInfo.RegionInfo)
	if err != nil {
		return nil, err
	}

	//iConn = acon.AwsCloudConnection{}
	iConn := acon.AwsCloudConnection{
		CredentialInfo: connectionInfo.CredentialInfo,
		Region:         connectionInfo.RegionInfo,
		VMClient:       vmClient,
		KeyPairClient:  vmClient,

		VNetworkClient: vmClient,
		//VNicClient:     vmClient,
		ImageClient: vmClient,
		//PublicIPClient: vmClient,
		SecurityClient: vmClient,
		VmSpecClient:   vmClient,
		NLBClient:      nlbClient,
		DiskClient:     vmClient,
		MyImageClient:  vmClient,

		EKSClient:         eksClient,
		IamClient:         iamClient,
		AutoScalingClient: autoScalingClient,

		RegionZoneClient: vmClient,
		PriceInfoClient:  pricingClient,

		// Connection for AnyCall
		AnyCallClient: vmClient,
	}

	return &iConn, nil // return type: (icon.CloudConnection, error)
}

// Pricing data를 위한 Pricing 클라이언트 획득
func getPricingClient(connectionInfo idrv.ConnectionInfo) (*pricing.Pricing, error) {

	// "us-east-1", "eu-central-1", "ap-south-1" 3개 리전의 엔드포인트만 지원
	// AWS 리전은 Price List Query API의 API 엔드포인트입니다.
	// 엔드포인트는 제품 또는 서비스 속성과 관련이 없습니다.
	// https://docs.aws.amazon.com/ko_kr/awsaccountbilling/latest/aboutv2/using-price-list-query-api.html#price-list-query-api-endpoints

	pricingEndpointRegion := []string{"us-east-1", "eu-central-1", "ap-south-1"}
	match := false
	for _, str := range pricingEndpointRegion {
		if str == connectionInfo.RegionInfo.Region {
			match = true
			break
		}
	}

	var targetRegion string
	if match {
		targetRegion = connectionInfo.RegionInfo.Region
	} else {
		targetRegion = "us-east-1"
	}

	sess := session.Must(session.NewSession())
	// Create a Pricing client with additional configuration
	svc := pricing.New(sess, &aws.Config{
		// Region: aws.String(connectionInfo.RegionInfo.Region),
		Region: aws.String(targetRegion),
		//Region:      aws.String("ap-northeast-2"),
		Credentials: credentials.NewStaticCredentials(connectionInfo.CredentialInfo.ClientId, connectionInfo.CredentialInfo.ClientSecret, "")},
	)

	return svc, nil
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
