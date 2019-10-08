// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by powerkim@etri.re.kr, 2019.06.

//VPC 처리
package resources

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type AwsVNetworkHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

func (vNetworkHandler *AwsVNetworkHandler) ListVNetwork() ([]*irs.VNetworkInfo, error) {
	cblogger.Debug("Start")
	result, err := vNetworkHandler.Client.DescribeVpcs(&ec2.DescribeVpcsInput{})
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
		return nil, err
	}

	var vNetworkInfoList []*irs.VNetworkInfo
	for _, curVpc := range result.Vpcs {
		cblogger.Info("[%s] VPC 정보 조회", *curVpc.VpcId)
		vNetworkInfo := ExtractDescribeInfo(curVpc)
		vNetworkInfoList = append(vNetworkInfoList, &vNetworkInfo)
	}

	spew.Dump(vNetworkInfoList)
	return vNetworkInfoList, nil
}

func (vNetworkHandler *AwsVNetworkHandler) CreateVNetwork(vNetworkReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {
	cblogger.Info(vNetworkReqInfo)
	input := &ec2.CreateVpcInput{
		CidrBlock: aws.String(vNetworkReqInfo.CidrBlock),
	}

	spew.Dump(input)
	result, err := vNetworkHandler.Client.CreateVpc(input)
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
		return irs.VNetworkInfo{}, err
	}

	cblogger.Info(result)
	spew.Dump(result)
	//vNetworkInfo := irs.VNetworkInfo{}
	vNetworkInfo := ExtractDescribeInfo(result.Vpc)

	//VPC Name 태깅
	tagInput := &ec2.CreateTagsInput{
		Resources: []*string{
			aws.String(*result.Vpc.VpcId),
		},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(vNetworkReqInfo.Name),
			},
		},
	}
	spew.Dump(tagInput)

	_, errTag := vNetworkHandler.Client.CreateTags(tagInput)
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
		return vNetworkInfo, errTag
	}

	vNetworkInfo.Name = vNetworkReqInfo.Name

	return vNetworkInfo, nil
}

func (vNetworkHandler *AwsVNetworkHandler) GetVNetwork(vNetworkID string) (irs.VNetworkInfo, error) {
	cblogger.Info("vNetworkID : [%s]", vNetworkID)

	input := &ec2.DescribeVpcsInput{
		VpcIds: []*string{
			aws.String(vNetworkID),
		},
	}

	result, err := vNetworkHandler.Client.DescribeVpcs(input)
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
		return irs.VNetworkInfo{}, err
	}

	cblogger.Info(result)
	//spew.Dump(result)

	vNetworkInfo := ExtractDescribeInfo(result.Vpcs[0])
	return vNetworkInfo, nil
}

//VPC 정보를 추출함
func ExtractDescribeInfo(vpcInfo *ec2.Vpc) irs.VNetworkInfo {
	vNetworkInfo := irs.VNetworkInfo{
		Id:        *vpcInfo.VpcId,
		CidrBlock: *vpcInfo.CidrBlock,
		IsDefault: *vpcInfo.IsDefault,
		State:     *vpcInfo.State,
	}

	//Name은 Tag의 "Name" 속성에만 저장됨
	cblogger.Debug("Name Tag 찾기")
	for _, t := range vpcInfo.Tags {
		if *t.Key == "Name" {
			vNetworkInfo.Name = *t.Value
			cblogger.Debug("VPC Name : ", vNetworkInfo.Name)
			break
		}
	}

	return vNetworkInfo
}

func (vNetworkHandler *AwsVNetworkHandler) DeleteVNetwork(vNetworkID string) (bool, error) {
	cblogger.Info("vNetworkID : [%s]", vNetworkID)
	input := &ec2.DeleteVpcInput{
		VpcId: aws.String(vNetworkID),
	}

	result, err := vNetworkHandler.Client.DeleteVpc(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Error(err.Error())
		}
		return false, err
	}

	cblogger.Info(result)
	spew.Dump(result)
	return true, nil
}
