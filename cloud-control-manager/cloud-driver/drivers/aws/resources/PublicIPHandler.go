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
	"fmt"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type AwsPublicIPHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

//@TODO : EC2에 Public를 할당하는 Associate함수 필요 함.
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
			cblogger.Info("*", fmtAddress(addr))
			spew.Dump(addr)
		}
	}

	return publicIpList, nil
}

func fmtAddress(addr *ec2.Address) string {
	out := fmt.Sprintf("IP: %s,  allocation id: %s",
		aws.StringValue(addr.PublicIp), aws.StringValue(addr.AllocationId))
	if addr.InstanceId != nil {
		out += fmt.Sprintf(", instance-id: %s", *addr.InstanceId)
	}
	return out
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
		for _, allocRes := range result.Addresses {
			cblogger.Info("*", fmtAddress(allocRes))
			spew.Dump(allocRes)
			publicIPInfo.Domain = *allocRes.Domain
			publicIPInfo.PublicIp = *allocRes.PublicIp
			publicIPInfo.PublicIpv4Pool = *allocRes.PublicIpv4Pool
			publicIPInfo.AllocationId = *allocRes.AllocationId

			if !reflect.ValueOf(allocRes.AssociationId).IsNil() {
				publicIPInfo.AssociationId = *allocRes.AssociationId           // AWS:연결ID
				publicIPInfo.InstanceId = *allocRes.InstanceId                 // AWS:연결된 VM
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

		}
	}

	return publicIPInfo, nil
}

func (publicIpHandler *AwsPublicIPHandler) DeletePublicIP(publicIPID string) (bool, error) {
	cblogger.Infof("publicIPID : [%s]", publicIPID)
	input := &ec2.ReleaseAddressInput{
		//AllocationId: aws.String("eipalloc-64d5890a"),
		PublicIp: aws.String(publicIPID),
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

	fmt.Println(result)
	return true, nil
}
