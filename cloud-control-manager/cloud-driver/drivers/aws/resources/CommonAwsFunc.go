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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

const CBDefaultVNetName string = "CB-VNet"          // CB Default Virtual Network Name
const CBDefaultSubnetName string = "CB-VNet-Subnet" // CB Default Subnet Name
const CBDefaultCidrBlock string = "192.168.0.0/16"  // CB Default CidrBlock

type AwsCBNetworkInfo struct {
	VpcName   string
	VpcId     string
	CidrBlock string
	IsDefault bool
	State     string

	SubnetName string
	SubnetId   string
}

const CUSTOM_ERR_CODE_TOOMANY string = "600"  //awserr.New("600", "n개 이상의 xxxx 정보가 존재합니다.", nil)
const CUSTOM_ERR_CODE_NOTFOUND string = "404" //awserr.New("404", "XXX 정보가 존재하지 않습니다.", nil)

//VPC
func GetCBDefaultVNetName() string {
	return CBDefaultVNetName
}

//Subnet
func GetCBDefaultSubnetName() string {
	return CBDefaultSubnetName
}

func GetCBDefaultCidrBlock() string {
	return CBDefaultCidrBlock
}

//이 함수는 VPC & Subnet이 존재하는 곳에서만 사용됨.
//VPC & Subnet이 존재하는 경우 정보를 리턴하고 없는 경우 Default VPC & Subnet을 생성 후 정보를 리턴 함.
func (vNetworkHandler *AwsVNetworkHandler) GetAutoCBNetworkInfo() (AwsCBNetworkInfo, error) {
	var awsCBNetworkInfo AwsCBNetworkInfo

	subNetId := vNetworkHandler.GetMcloudBaristaDefaultSubnetId()
	if subNetId == "" {
		//내부에서 VPC를 자동으로 생성후 Subnet을 생성함.
		_, err := vNetworkHandler.CreateVNetwork(irs.VNetworkReqInfo{})
		if err != nil {
			cblogger.Error("Default VNetwork(VPC & Subnet) 자동 생성 실패")
			cblogger.Error(err)
			return AwsCBNetworkInfo{}, err
		}
	}

	//VPC & Subnet을 생성했으므로 예외처리 없이 조회만 처리함.
	awsVpcInfo, _ := vNetworkHandler.GetVpc(GetCBDefaultVNetName())
	spew.Dump(awsVpcInfo)
	awsCBNetworkInfo.VpcId = awsVpcInfo.Id
	awsCBNetworkInfo.VpcName = awsVpcInfo.Name

	awsSubnetInfo, _ := vNetworkHandler.GetVNetwork("")
	spew.Dump(awsSubnetInfo)
	awsCBNetworkInfo.SubnetId = awsSubnetInfo.Id
	awsCBNetworkInfo.SubnetName = awsSubnetInfo.Name

	spew.Dump(awsCBNetworkInfo)

	return awsCBNetworkInfo, nil
}

func (vNetworkHandler *AwsVNetworkHandler) GetMcloudBaristaDefaultVpcId() string {
	awsVpcInfo, err := vNetworkHandler.GetVpc(GetCBDefaultVNetName())
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			cblogger.Error(err.Error())
		}
		return ""
	}

	//기존 정보가 존재하면...
	if awsVpcInfo.Id != "" {
		return awsVpcInfo.Id
	} else {
		return ""
	}
}

func (vNetworkHandler *AwsVNetworkHandler) GetMcloudBaristaDefaultSubnetId() string {
	awsSubnetInfo, err := vNetworkHandler.GetVNetwork("")
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			cblogger.Error(err.Error())
		}
		return ""
	}

	//기존 정보가 존재하면...
	if awsSubnetInfo.Id != "" {
		return awsSubnetInfo.Id
	} else {
		return ""
	}
}

//@TODO : ListVNetwork()에서 호출되는 경우도 있기 때문에 필요하면 VPC조회와 생성을 별도의 Func으로 분리해야함.(일단은 큰 문제는 없어서 놔둠)
//CB Default Virtual Network가 존재하지 않으면 생성하며, 존재하는 경우 Vpc ID를 리턴 함.
func (vNetworkHandler *AwsVNetworkHandler) FindOrCreateMcloudBaristaDefaultVPC(vNetworkReqInfo irs.VNetworkReqInfo) (string, error) {
	cblogger.Info(vNetworkReqInfo)

	awsVpcInfo, err := vNetworkHandler.GetVpc(GetCBDefaultVNetName())
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return "", err
	}

	//기존 정보가 존재하면...
	if awsVpcInfo.Id != "" {
		return awsVpcInfo.Id, nil
	} else {
		//@TODO : Subnet과 VPC모두 CSP별 고정된 값으로 드라이버가 내부적으로 자동으로 생성하도록 CB규약이 바뀌어서 서브넷 정보 기반의 로직은 모두 잠시 죽여 놓음 - 리스트 요청시에도 내부적으로 자동 생성하도록 변경 중
		/*
			cblogger.Infof("기본 VPC[%s]가 없어서 Subnet 요청 정보를 기반으로 /16 범위의 VPC를 생성합니다.", GetCBDefaultVNetName())
			cblogger.Info("Subnet CIDR 요청 정보 : ", vNetworkReqInfo.CidrBlock)
			if vNetworkReqInfo.CidrBlock == "" {
				//VPC가 없는 최초 상태에서 List()에서 호출되었을 수 있기 때문에 에러 처리는 하지 않고 nil을 전달함.
				cblogger.Infof("요청 정보에 CIDR 정보가 없어서 Default VPC[%s]를 생성하지 않음", GetCBDefaultVNetName())
				return "", nil
			}

			reqCidr := strings.Split(vNetworkReqInfo.CidrBlock, ".")
			//cblogger.Info("CIDR 추출 정보 : ", reqCidr[0])
			VpcCidrBlock := reqCidr[0] + "." + reqCidr[1] + ".0.0/16"
			cblogger.Info("신규 VPC에 사용할 CIDR 정보 : ", VpcCidrBlock)
		*/

		cblogger.Infof("기본 VPC[%s]가 없어서 CIDR[%s] 범위의 VPC를 자동으로 생성합니다.", GetCBDefaultVNetName(), GetCBDefaultCidrBlock())
		awsVpcReqInfo := AwsVpcReqInfo{
			Name: GetCBDefaultVNetName(),
			//CidrBlock: VpcCidrBlock,
			CidrBlock: GetCBDefaultCidrBlock(),
		}

		result, errVpc := vNetworkHandler.CreateVpc(awsVpcReqInfo)
		if errVpc != nil {
			cblogger.Error(errVpc)
			return "", errVpc
		}
		cblogger.Infof("CB Default VPC[%s] 생성 완료 - CIDR : [%s]", GetCBDefaultVNetName(), result.CidrBlock)
		cblogger.Info(result)
		spew.Dump(result)

		return result.Id, nil
	}
}

//자동으로 생성된 VPC & Subnet을 삭제해도 되는가?
//명시적으로 Subnet 삭제의 호출이 없기 때문에 시큐리티 그룹이나 vNic이 삭제되는 시점에 호출됨.
func (vNetworkHandler *AwsVNetworkHandler) IsAvailableAutoCBNet() bool {
	return false
}

func SetNameTag(Client *ec2.EC2, Id string, value string) bool {
	// Tag에 Name 설정
	cblogger.Infof("Name Tage 설정 - ResourceId : [%s]  Value : [%s] ", Id, value)
	_, errtag := Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{&Id},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(value),
			},
		},
	})
	if errtag != nil {
		cblogger.Error("Name Tag 설정 실패 : ")
		cblogger.Error(errtag)
		return false
	}

	return true
}
