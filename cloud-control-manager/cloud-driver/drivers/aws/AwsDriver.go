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
	"fmt"
	"os"

	acon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws/connect"
	profile "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws/profile"
	ars "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	ires "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"

	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	//icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect/AwsNewIfCloudConnect"
	//icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect/connect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/aws/aws-sdk-go/service/sts"
	cblogger "github.com/cloud-barista/cb-log"
)

var cblog *logrus.Logger

func init() {
	cblog = cblogger.GetLogger("CLOUD-BARISTA")
}

type AwsDriver struct {
}

func (AwsDriver) GetDriverVersion() string {
	return "AWS DRIVER Version 1.0"
}

func (AwsDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ZoneBasedControl = true

	drvCapabilityInfo.RegionZoneHandler = true
	drvCapabilityInfo.PriceInfoHandler = true
	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VMSpecHandler = true

	drvCapabilityInfo.VPCHandler = true
	drvCapabilityInfo.SecurityHandler = true
	drvCapabilityInfo.KeyPairHandler = true
	drvCapabilityInfo.VMHandler = true
	drvCapabilityInfo.DiskHandler = true
	drvCapabilityInfo.MyImageHandler = true
	drvCapabilityInfo.NLBHandler = true
	drvCapabilityInfo.ClusterHandler = true
	drvCapabilityInfo.FileSystemHandler = true

	drvCapabilityInfo.TagHandler = true
	drvCapabilityInfo.TagSupportResourceType = []ires.RSType{ires.VPC, ires.SUBNET, ires.SG, ires.KEY, ires.VM, ires.NLB, ires.DISK, ires.MYIMAGE, ires.CLUSTER, ires.FILESYSTEM}

	drvCapabilityInfo.VPC_CIDR = true

	return drvCapabilityInfo
}

// 공통 AWS 세션 생성 함수
func newAWSSession(connectionInfo idrv.ConnectionInfo, region string) (*session.Session, error) {
	// cblog.Info("****************************************************")
	// cblog.Info("ClientId : ", connectionInfo.CredentialInfo.ClientId)
	// cblog.Info("ClientSecret : ", connectionInfo.CredentialInfo.ClientSecret)
	// cblog.Info("StsToken : ", connectionInfo.CredentialInfo.StsToken)
	// cblog.Info("****************************************************")

	StsToken := ""
	if connectionInfo.CredentialInfo.StsToken == "Not set" || connectionInfo.CredentialInfo.StsToken == "" {
		cblog.Debug("======> StsToken is not set")
		connectionInfo.CredentialInfo.StsToken = "" // Ensure StsToken is empty if not set
		StsToken = ""
	} else {
		cblog.Debug("======> StsToken is set")
		StsToken = connectionInfo.CredentialInfo.StsToken
	}

	cblog.Debug("Received connection information")
	cblog.Debug("============================================================================================")
	// if connectionInfo.CredentialInfo.StsToken != "" {
	if StsToken != "" {
		cblog.Debugf("Using SessionToken(For iam-manager - STS) for AWS API calls : [%s]", StsToken)
	} else {
		cblog.Debugf("Using Normal AWS Session for AWS API calls [%s]", connectionInfo.CredentialInfo.ClientId)
	}
	cblog.Debug("============================================================================================")

	awsConfig := &aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(true),
		Region:                        aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			connectionInfo.CredentialInfo.ClientId,
			connectionInfo.CredentialInfo.ClientSecret,
			StsToken,
			//connectionInfo.CredentialInfo.StsToken, // TS SessionToekn field in AWS
		),
	}
	// return session.NewSession(awsConfig)

	// 기존의 getVMClient에 있던 NewCountingSession 로직을 살려 놓음.
	// API 호출 카운트를 위해 만들었던데 용도 파악 후 지금처럼 모든 Client에 적용하거나 필요 없으면 전부 제거해도 좋을 듯.
	// If the CALL_COUNT environment variable is set, use a counting session
	if os.Getenv("CALL_COUNT") != "" {
		// profile.NewCountingSession은 별도 구현체(테스트/계측용)라고 가정
		sess := profile.NewCountingSession(awsConfig)
		if sess == nil {
			return nil, fmt.Errorf("failed to create counting session")
		}
		// cblog.Infof("Using counting session for AWS API calls")
		return sess, nil
	} else {
		sess, err := session.NewSession(awsConfig)
		if err != nil {
			cblog.Error("Could not create AWS session", err)
			return nil, err
		}
		// cblog.Infof("Using regular session for AWS API calls")
		return sess, nil
	}
}

// func getVMClient(regionInfo idrv.RegionInfo) (*ec2.EC2, error) {
func getVMClient(connectionInfo idrv.ConnectionInfo) (*ec2.EC2, error) {
	// // setup Region
	// awsConfig := &aws.Config{
	// 	CredentialsChainVerboseErrors: aws.Bool(true),
	// 	Region:                        aws.String(connectionInfo.RegionInfo.Region),
	// 	Credentials: credentials.NewStaticCredentials(
	// 		connectionInfo.CredentialInfo.ClientId,
	// 		connectionInfo.CredentialInfo.ClientSecret,
	// 		connectionInfo.CredentialInfo.AuthToken, // The AuthToken field is a SessionToken value based on STS in AWS. (for iam-manager) // @todo
	// 		//"",
	// 	),
	// }

	// var sess *session.Session

	// // If the CALL_COUNT environment variable is set, use a counting session
	// if os.Getenv("CALL_COUNT") != "" {
	// 	sess = profile.NewCountingSession(awsConfig)
	// 	if sess == nil {
	// 		return nil, fmt.Errorf("failed to create counting session")
	// 	}
	// 	cblog.Infof("Using counting session for AWS API calls")
	// } else {
	// 	var err error
	// 	sess, err = session.NewSession(awsConfig)
	// 	if err != nil {
	// 		cblog.Error("Could not create AWS session", err)
	// 		return nil, err
	// 	}
	// 	cblog.Infof("Using regular session for AWS API calls")
	// }

	sess, err := newAWSSession(connectionInfo, connectionInfo.RegionInfo.Region)
	if err != nil {
		cblog.Error("Could not create AWS session", err)
		return nil, err
	}
	// EC2 서비스 클라이언트 생성
	svc := ec2.New(sess)

	return svc, nil
}

// 로드밸런서 처리를 위한 ELB 클라이언트 획득
func getNLBClient(connectionInfo idrv.ConnectionInfo) (*elbv2.ELBV2, error) {
	//func getNLBClient(connectionInfo idrv.ConnectionInfo) (*elb.ELB, error) {

	// setup Region
	//cblog.Info("AwsDriver : getVMClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")
	//cblog.Info("AwsDriver : getVMClient() - Zone : [" + connectionInfo.RegionInfo.Zone + "]")
	//cblog.Info("전달 받은 커넥션 정보")

	sess, err := newAWSSession(connectionInfo, connectionInfo.RegionInfo.Region)
	if err != nil {
		cblog.Error("Could not create AWS session", err)
		return nil, err
	}

	// //svc := elb.New(sess)
	// // Create ELBv2 service client
	// svc := elbv2.New(sess)
	// if err != nil {
	// 	cblog.Error("Could not create ELBv2 service client", err)
	// 	return nil, err
	// }

	// return svc, nil
	return elbv2.New(sess), nil
}

// EKS 처리를 위한 EKS 클라이언트 획득
func getEKSClient(connectionInfo idrv.ConnectionInfo) (*eks.EKS, error) {

	// setup Region
	//cblog.Info("AwsDriver : getEKSClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")
	//cblog.Info("AwsDriver : getEKSClient() - Zone : [" + connectionInfo.RegionInfo.Zone + "]")
	//cblog.Info("전달 받은 커넥션 정보")

	// sess, err := session.NewSession(&aws.Config{
	// 	Region: aws.String(connectionInfo.RegionInfo.Region),
	// 	//Region:      aws.String("ap-northeast-2"),
	// 	Credentials: credentials.NewStaticCredentials(connectionInfo.CredentialInfo.ClientId, connectionInfo.CredentialInfo.ClientSecret, "")},
	// )
	// if err != nil {
	// 	cblog.Error("Could not create aws New Session", err)
	// 	return nil, err
	// }

	// svc := eks.New(sess)
	// //if err != nil {
	// //	cblog.Error("Could not create eks service client", err)
	// //	return nil, err
	// //}

	// return svc, nil
	sess, err := newAWSSession(connectionInfo, connectionInfo.RegionInfo.Region)
	if err != nil {
		cblog.Error("Could not create AWS session", err)
		return nil, err
	}
	return eks.New(sess), nil
}

// Iam 처리를 위한 iam 클라이언트 획득
func getIamClient(connectionInfo idrv.ConnectionInfo) (*iam.IAM, error) {
	// setup Region
	//cblog.Info("AwsDriver : getIamClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")
	//cblog.Info("AwsDriver : getIamClient() - Zone : [" + connectionInfo.RegionInfo.Zone + "]")
	//cblog.Info("전달 받은 커넥션 정보")

	// sess, err := session.NewSession(&aws.Config{
	// 	Region: aws.String(connectionInfo.RegionInfo.Region),
	// 	//Region:      aws.String("ap-northeast-2"),
	// 	Credentials: credentials.NewStaticCredentials(connectionInfo.CredentialInfo.ClientId, connectionInfo.CredentialInfo.ClientSecret, "")},
	// )
	// if err != nil {
	// 	cblog.Error("Could not create aws New Session", err)
	// 	return nil, err
	// }

	// svc := iam.New(sess)
	// //if err != nil {
	// //	cblog.Error("Could not create iam service client", err)
	// //	return nil, err
	// //}

	// return svc, nil
	sess, err := newAWSSession(connectionInfo, connectionInfo.RegionInfo.Region)
	if err != nil {
		cblog.Error("Could not create AWS session", err)
		return nil, err
	}
	return iam.New(sess), nil
}

// Get STS client for STS processing
func getStsClient(connectionInfo idrv.ConnectionInfo) (*sts.STS, error) {
	sess, err := newAWSSession(connectionInfo, connectionInfo.RegionInfo.Region)
	if err != nil {
		cblog.Error("Could not create AWS session", err)
		return nil, err
	}
	return sts.New(sess), nil
}

// AutoScaling 처리를 위한 autoScaling 클라이언트 획득
func getAutoScalingClient(connectionInfo idrv.ConnectionInfo) (*autoscaling.AutoScaling, error) {

	// setup Region
	//cblog.Info("AwsDriver : getAutoScalingClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")
	//cblog.Info("AwsDriver : getAutoScalingClient() - Zone : [" + connectionInfo.RegionInfo.Zone + "]")
	//cblog.Info("전달 받은 커넥션 정보")

	// sess, err := session.NewSession(&aws.Config{
	// 	Region:      aws.String(connectionInfo.RegionInfo.Region),
	// 	Credentials: credentials.NewStaticCredentials(connectionInfo.CredentialInfo.ClientId, connectionInfo.CredentialInfo.ClientSecret, "")},
	// )
	// if err != nil {
	// 	cblog.Error("Could not create aws New Session", err)
	// 	return nil, err
	// }

	// //svc := elb.New(sess)
	// // Create ELBv2 service client
	// svc := autoscaling.New(sess)
	// //if err != nil {
	// //	cblog.Error("Could not create autoscaling service client", err)
	// //	return nil, err
	// //}

	// return svc, nil

	sess, err := newAWSSession(connectionInfo, connectionInfo.RegionInfo.Region)
	if err != nil {
		cblog.Error("Could not create AWS session", err)
		return nil, err
	}
	return autoscaling.New(sess), nil
}

// EFS 처리를 위한 EFS 클라이언트 획득
func getEFSClient(connectionInfo idrv.ConnectionInfo) (*efs.EFS, error) {
	sess, err := newAWSSession(connectionInfo, connectionInfo.RegionInfo.Region)
	if err != nil {
		cblog.Error("Could not create AWS session", err)
		return nil, err
	}
	return efs.New(sess), nil
}

func (driver *AwsDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	// Initialize Logger
	ars.InitLog()

	//cblog.Info("ConnectCloud의 전달 받은 idrv.ConnectionInfo 정보")

	// sample code, do not user like this^^
	//var iConn icon.CloudConnection
	vmClient, err := getVMClient(connectionInfo)
	nlbClient, err := getNLBClient(connectionInfo)
	eksClient, err := getEKSClient(connectionInfo)
	iamClient, err := getIamClient(connectionInfo)
	stsClient, err := getStsClient(connectionInfo)
	pricingClient, err := getPricingClient(connectionInfo)
	autoScalingClient, err := getAutoScalingClient(connectionInfo)
	efsClient, err := getEFSClient(connectionInfo)
	//vmClient, err := getVMClient(connectionInfo.RegionInfo)
	if err != nil {
		return nil, err
	}
	costExplorerClient, err := getCostExplorerClient(connectionInfo)

	cloudwatchClient, err := getCloudWatchClient(connectionInfo)

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
		StsClient:         stsClient,
		AutoScalingClient: autoScalingClient,

		RegionZoneClient: vmClient,
		PriceInfoClient:  pricingClient,

		// Connection for AnyCall
		AnyCallClient: vmClient,

		TagClient:          vmClient,
		CostExplorerClient: costExplorerClient,
		CloudWatchClient:   cloudwatchClient,
		FileSystemClient:   efsClient,
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
	// match := false
	// for _, str := range pricingEndpointRegion {
	// 	if str == connectionInfo.RegionInfo.Region {
	// 		match = true
	// 		break
	// 	}
	// }

	// var targetRegion string
	// if match {
	// 	targetRegion = connectionInfo.RegionInfo.Region
	// } else {
	// 	targetRegion = "us-east-1"
	// }

	// sess := session.Must(session.NewSession())
	// // Create a Pricing client with additional configuration
	// svc := pricing.New(sess, &aws.Config{
	// 	// Region: aws.String(connectionInfo.RegionInfo.Region),
	// 	Region: aws.String(targetRegion),
	// 	//Region:      aws.String("ap-northeast-2"),
	// 	Credentials: credentials.NewStaticCredentials(connectionInfo.CredentialInfo.ClientId, connectionInfo.CredentialInfo.ClientSecret, "")},
	// )

	// return svc, nil
	targetRegion := "us-east-1"
	for _, str := range pricingEndpointRegion {
		if str == connectionInfo.RegionInfo.Region {
			targetRegion = connectionInfo.RegionInfo.Region
			break
		}
	}
	sess, err := newAWSSession(connectionInfo, targetRegion)
	if err != nil {
		cblog.Error("Could not create AWS session", err)
		return nil, err
	}
	return pricing.New(sess), nil
}

func getCostExplorerClient(connectionInfo idrv.ConnectionInfo) (*costexplorer.CostExplorer, error) {
	// sess := session.Must(session.NewSession())
	// svc := costexplorer.New(sess, &aws.Config{
	// 	Region:      aws.String(connectionInfo.RegionInfo.Region),
	// 	Credentials: credentials.NewStaticCredentials(connectionInfo.CredentialInfo.ClientId, connectionInfo.CredentialInfo.ClientSecret, "")},
	// )

	// return svc, nil
	sess, err := newAWSSession(connectionInfo, connectionInfo.RegionInfo.Region)
	if err != nil {
		cblog.Error("Could not create AWS session", err)
		return nil, err
	}
	return costexplorer.New(sess), nil
}

func getCloudWatchClient(connectionInfo idrv.ConnectionInfo) (*cloudwatch.CloudWatch, error) {

	// sess, err := session.NewSession(&aws.Config{
	// 	Region: aws.String(connectionInfo.RegionInfo.Region),
	// 	Credentials: credentials.NewStaticCredentials(
	// 		connectionInfo.CredentialInfo.ClientId,
	// 		connectionInfo.CredentialInfo.ClientSecret,
	// 		"",
	// 	),
	// })

	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create session: %v", err)
	// }

	// svc := cloudwatch.New(sess)

	// return svc, nil
	sess, err := newAWSSession(connectionInfo, connectionInfo.RegionInfo.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	return cloudwatch.New(sess), nil
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
