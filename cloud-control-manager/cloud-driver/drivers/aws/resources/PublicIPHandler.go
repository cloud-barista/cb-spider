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
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type AwsPublicIPHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

//@TODO : 공통 I/F에 함수 추가해야 함. - EC2에 Public를 할당하는 AssociatePublicIP함수를 별도로 구현이 필요 함. - 현재는 테스트를 위해 생성과 동시에 할당하도록 해 놨음.
func (publicIpHandler *AwsPublicIPHandler) CreatePublicIP(publicIPReqInfo irs.PublicIPReqInfo) (irs.PublicIPInfo, error) {
	cblogger.Info("Start : ", publicIPReqInfo)

	var publicIPInfo irs.PublicIPInfo

	//@TODO: 대체해야 함.
	instanceID := publicIPReqInfo.Id

	// Attempt to allocate the Elastic IP address.
	allocRes, err := publicIpHandler.Client.AllocateAddress(&ec2.AllocateAddressInput{
		Domain: aws.String("vpc"), // 적용 범위 : VPC
	})

	if err != nil {
		cblogger.Errorf("Unable to allocate IP address, %v", err)
		return irs.PublicIPInfo{}, err
	}

	spew.Dump(allocRes)
	cblogger.Infof("EIP 생성 성공 - Public IP : [%s], Allocation Id : [%s]", *allocRes.PublicIp, *allocRes.AllocationId)
	publicIPInfo.Domain = *allocRes.Domain
	publicIPInfo.PublicIp = *allocRes.PublicIp
	publicIPInfo.PublicIpv4Pool = *allocRes.PublicIpv4Pool
	publicIPInfo.AllocationId = *allocRes.AllocationId

	cblogger.Infof("[%s] EC2에 [%s] IP 할당 시작", instanceID, *allocRes.PublicIp)
	// EC2에 할당.
	// Associate the new Elastic IP address with an existing EC2 instance.
	assocRes, err := publicIpHandler.Client.AssociateAddress(&ec2.AssociateAddressInput{
		AllocationId: allocRes.AllocationId,
		InstanceId:   aws.String(instanceID),
	})
	if err != nil {
		cblogger.Errorf("Unable to associate IP address with %s, %v", instanceID, err)
		return irs.PublicIPInfo{}, err
	}
	spew.Dump(assocRes)
	cblogger.Infof("[%s] EC2에 [%s] IP 할당 완료 - Allocation Id : [%s]", instanceID, *allocRes.PublicIp, *assocRes.AssociationId)

	return publicIPInfo, nil
}

func (publicIpHandler *AwsPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	cblogger.Info("Start~")
	var publicIpList []*irs.PublicIPInfo

	// Make the API request to EC2 filtering for the addresses in the
	// account's VPC.
	result, err := publicIpHandler.Client.DescribeAddresses(&ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("domain"),
				Values: aws.StringSlice([]string{"vpc"}),
			},
		},
	})
	if err != nil {
		cblogger.Errorf("Unable to elastic IP address, %v", err)
		return nil, err
	}

	// Printout the IP addresses if there are any.
	if len(result.Addresses) == 0 {
		cblogger.Infof("No elastic IPs for %s region\n", *publicIpHandler.Client.Config.Region)
	} else {
		cblogger.Info("Elastic IPs")
		for _, addr := range result.Addresses {
			publicIPInfo := fmtAddress(addr)
			publicIpList = append(publicIpList, &publicIPInfo)
		}
	}

	return publicIpList, nil
}

func fmtAddress(allocRes *ec2.Address) irs.PublicIPInfo {
	var publicIPInfo irs.PublicIPInfo

	spew.Dump(allocRes)
	publicIPInfo.Domain = *allocRes.Domain
	publicIPInfo.PublicIp = *allocRes.PublicIp
	publicIPInfo.PublicIpv4Pool = *allocRes.PublicIpv4Pool
	publicIPInfo.AllocationId = *allocRes.AllocationId

	if !reflect.ValueOf(allocRes.InstanceId).IsNil() {
		publicIPInfo.InstanceId = *allocRes.InstanceId // AWS:연결된 VM
	}

	if !reflect.ValueOf(allocRes.AssociationId).IsNil() {
		publicIPInfo.AssociationId = *allocRes.AssociationId // AWS:연결ID
	}

	if !reflect.ValueOf(allocRes.NetworkInterfaceId).IsNil() {
		publicIPInfo.NetworkInterfaceId = *allocRes.NetworkInterfaceId // AWS:연결된 Nic
		publicIPInfo.NetworkInterfaceOwnerId = *allocRes.NetworkInterfaceOwnerId
		publicIPInfo.PrivateIpAddress = *allocRes.PrivateIpAddress
	}

	for _, t := range allocRes.Tags {
		if *t.Key == "Name" {
			publicIPInfo.Name = *t.Value
			cblogger.Debug("명칭 : ", publicIPInfo.Name)
			break
		}
	}

	return publicIPInfo
}

func (publicIpHandler *AwsPublicIPHandler) GetPublicIP(publicIPID string) (irs.PublicIPInfo, error) {
	cblogger.Infof("publicIPID : [%s]", publicIPID)

	var publicIPInfo irs.PublicIPInfo

	// Make the API request to EC2 filtering for the addresses in the account's VPC.
	result, err := publicIpHandler.Client.DescribeAddresses(&ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("public-ip"),
				//Values: aws.StringSlice([]string{"vpc"}),
				Values: []*string{
					aws.String(publicIPID),
				},
			},
		},
	})
	if err != nil {
		cblogger.Errorf("Unable to elastic IP address, %v", err)
		return irs.PublicIPInfo{}, err
	}

	// Printout the IP addresses if there are any.
	if len(result.Addresses) == 0 {
		cblogger.Infof("No elastic IPs for %s region\n", *publicIpHandler.Client.Config.Region)
	} else {
		cblogger.Info("Elastic IPs")
		for _, addr := range result.Addresses {
			publicIPInfo = fmtAddress(addr)
		}
	}

	return publicIPInfo, nil
}

// Public IP를 완전히 제거 함.(AWS Pool로 되돌려 보냄)
func (publicIpHandler *AwsPublicIPHandler) DeletePublicIP(allocationId string) (bool, error) {
	cblogger.Infof("allocationId : [%s]", allocationId)
	input := &ec2.ReleaseAddressInput{
		AllocationId: aws.String(allocationId), //eipalloc-64d5890a - VPC에서 삭제
	}

	result, err := publicIpHandler.Client.ReleaseAddress(input)
	if err != nil {
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

	cblogger.Info(result)
	return true, nil
}

//@TODO : 공통 I/F에 함수 추가해야 함. - EC2 인스턴스와의 연결만 해제하는 DisassociatePublicIP
// publicIP 대신 AssociationId로도 가능 함.
func (publicIpHandler *AwsPublicIPHandler) DisassociatePublicIP(publicIP string) (bool, error) {
	cblogger.Infof("publicIP : [%s]", publicIP)
	input := &ec2.DisassociateAddressInput{
		// AssociationId: aws.String("eipassoc-2bebb745"),
		PublicIp: aws.String(publicIP),
	}

	result, err := publicIpHandler.Client.DisassociateAddress(input)
	if err != nil {
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

	cblogger.Info(result)
	return true, nil
}
