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
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/davecgh/go-spew/spew"
)

type AwsCBNetworkInfo struct {
	VpcName   string
	VpcId     string
	CidrBlock string
	IsDefault bool
	State     string

	SubnetName string
	SubnetId   string
}

const CBDefaultVNetName string = "CB-VNet"          // CB Default Virtual Network Name
const CBDefaultSubnetName string = "CB-VNet-Subnet" // CB Default Subnet Name
const CBDefaultCidrBlock string = "192.168.0.0/16"  // CB Default CidrBlock

func GetCBDefaultVNetName() string {
	return CBDefaultVNetName
}

func GetCBDefaultSubnetName() string {
	return CBDefaultSubnetName
}

func GetCBDefaultCidrBlock() string {
	return CBDefaultCidrBlock
}

//VPC & Subnet이 존재하는 경우 정보를 리턴하고 없는 경우 Default VPC & Subnet을 생성 후 정보를 리턴 함.
func GetCreateAutoCBNetworkInfo(client *ec2.EC2) (AwsCBNetworkInfo, error) {
	var awsCBNetworkInfo AwsCBNetworkInfo
	awsVpcInfo, err := GetVpc(client)
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
		return AwsCBNetworkInfo{}, err
	}

	//기존 정보가 존재하면...
	if awsVpcInfo.Id != "" {
		//return awsVpcInfo.Id, nil
		awsCBNetworkInfo.VpcId = awsVpcInfo.Id
		awsCBNetworkInfo.VpcName = awsVpcInfo.Name
		awsCBNetworkInfo.CidrBlock = awsVpcInfo.CidrBlock
	} else {
		cblogger.Infof("기본 VPC[%s]가 없어서 CIDR[%s] 범위의 VPC를 자동으로 생성합니다.", GetCBDefaultVNetName(), GetCBDefaultCidrBlock())
		awsVpcReqInfo := AwsVpcReqInfo{
			Name:      GetCBDefaultVNetName(),
			CidrBlock: GetCBDefaultCidrBlock(),
		}

		result, errVpc := CreateVpc(client, awsVpcReqInfo)
		if errVpc != nil {
			cblogger.Error(errVpc)
			return AwsCBNetworkInfo{}, errVpc
		}
		cblogger.Infof("CB Default VPC[%s] 생성 완료 - CIDR : [%s]", GetCBDefaultVNetName(), result.CidrBlock)
		cblogger.Info(result)
		spew.Dump(result)

		awsCBNetworkInfo.VpcId = awsVpcInfo.Id
		awsCBNetworkInfo.VpcName = awsVpcInfo.Name
		awsCBNetworkInfo.CidrBlock = result.CidrBlock

		//return result.Id, nil
	}

	return awsCBNetworkInfo, nil
}

func GetVpc(client *ec2.EC2) (AwsVpcInfo, error) {
	cblogger.Info("VPC Name : ", GetCBDefaultVNetName())

	input := &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("tag:Name"), // vpc-id , dhcp-options-id
				Values: []*string{
					aws.String(GetCBDefaultVNetName()),
				},
			},
		},
	}

	result, err := client.DescribeVpcs(input)
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
		return AwsVpcInfo{}, err
	}

	cblogger.Info(result)
	//spew.Dump(result)

	if !reflect.ValueOf(result.Vpcs).IsNil() {
		awsVpcInfo := ExtractVpcDescribeInfo(result.Vpcs[0])
		return awsVpcInfo, nil
	} else {
		return AwsVpcInfo{}, nil
	}
}

// FindOrCreateMcloudBaristaDefaultVPC()에서 호출됨. - 이 곳은 나중을 위해 전달 받은 정보는 이용함
// 기본 VPC 생성이 필요하면 FindOrCreateMcloudBaristaDefaultVPC()를 호출할 것
func CreateVpc(client *ec2.EC2, awsVpcReqInfo AwsVpcReqInfo) (AwsVpcInfo, error) {
	cblogger.Info(awsVpcReqInfo)

	input := &ec2.CreateVpcInput{
		CidrBlock: aws.String(awsVpcReqInfo.CidrBlock),
	}

	spew.Dump(input)
	result, err := client.CreateVpc(input)
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
		return AwsVpcInfo{}, err
	}

	cblogger.Info(result)
	spew.Dump(result)
	awsVpcInfo := ExtractVpcDescribeInfo(result.Vpc)

	//VPC Name 태깅
	tagInput := &ec2.CreateTagsInput{
		Resources: []*string{
			aws.String(*result.Vpc.VpcId),
		},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(awsVpcReqInfo.Name),
			},
		},
	}
	spew.Dump(tagInput)

	_, errTag := client.CreateTags(tagInput)
	if errTag != nil {
		//@TODO : Name 태깅 실패시 생성된 VPC를 삭제할지 Name 태깅을 하라고 전달할지 결정해야 함. - 일단, 바깥에서 처리 가능하도록 생성된 VPC 정보는 전달 함.
		if aerr, ok := errTag.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(errTag.Error())
		}
		return awsVpcInfo, errTag
	}

	awsVpcInfo.Name = awsVpcReqInfo.Name

	return awsVpcInfo, nil
}

//
// 현재는 VM 생성 시에만 사용하므로 일단 VMHandler에 구현해 놨음.
//
/*
// AssociationId 대신 PublicIP로도 가능 함.
func AssociatePublicIP(client *ec2.EC2, allocationId string, instanceId string) (bool, error) {
	cblogger.Infof("EC2에 퍼블릭 IP할당 - AllocationId : [%s], InstanceId : [%s]", allocationId, instanceId)

	// EC2에 할당.
	// Associate the new Elastic IP address with an existing EC2 instance.
	assocRes, err := client.AssociateAddress(&ec2.AssociateAddressInput{
		AllocationId: aws.String(allocationId),
		InstanceId:   aws.String(instanceId),
	})

	spew.Dump(assocRes)
	cblogger.Infof("[%s] EC2에 EIP(AllocationId : [%s]) 할당 완료 - AssociationId Id : [%s]", instanceId, allocationId, *assocRes.AssociationId)

	if err != nil {
		cblogger.Errorf("Unable to associate IP address with %s, %v", instanceId, err)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Errorf(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Errorf(err.Error())
		}
		return false, err
	}

	cblogger.Info(assocRes)
	return true, nil
}
*/
