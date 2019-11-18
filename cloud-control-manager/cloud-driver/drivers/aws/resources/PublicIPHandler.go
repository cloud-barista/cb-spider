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
	"errors"
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

//2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨.
// VM 생성 시 PublicIP의 AllocationId를 전달 받는 방식으로 1차 확정되어서 이 곳에서는 관련 로직을 제거 함.
// VMHandler.go에서 할당및 회수 함
func (publicIpHandler *AwsPublicIPHandler) CreatePublicIP(publicIPReqInfo irs.PublicIPReqInfo) (irs.PublicIPInfo, error) {
	cblogger.Info("Start : ", publicIPReqInfo)

	cblogger.Infof("중복 생성 방지를 위해 기존에 생성된 PublicIp[%s]가 있는지 조회 함.", publicIPReqInfo.Name)
	publicInfo, errPublicInfo := publicIpHandler.GetPublicIP(publicIPReqInfo.Name)
	if errPublicInfo == nil {
		cblogger.Errorf("이미 요청한 PublicIp[%s]가 존재하기 때문에 생성하지 않고 기존 정보와 함께 에러를 리턴함.", publicIPReqInfo.Name)
		cblogger.Info(publicInfo)
		return publicInfo, errors.New("InvalidPublicIp.Duplicate: The PublicIp '" + publicIPReqInfo.Name + "' already exists.")
	}

	//var publicIPInfo irs.PublicIPInfo

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

	// Tag에 Name 설정
	cblogger.Info("Name 설정 ", publicIPReqInfo.Name)
	_, errtag := publicIpHandler.Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{allocRes.AllocationId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(publicIPReqInfo.Name),
			},
		},
	})
	if errtag != nil {
		cblogger.Errorf("Public IP에 [%s] Name Tag 설정 실패 : ", publicIPReqInfo.Name)
		cblogger.Error(errtag)
		return irs.PublicIPInfo{}, errors.New("PublicIP [" + *allocRes.PublicIp + "]에 [" + publicIPReqInfo.Name + "] Name Tag 설정 실패")
	}

	//publicIPInfo, errExtract := publicIpHandler.GetPublicIP(*allocRes.AllocationId)
	publicIPInfo, errExtract := publicIpHandler.GetPublicIP(publicIPReqInfo.Name)
	if errExtract != nil {
		cblogger.Errorf("Public IP 정보 추출 실패 ", errExtract)
		return irs.PublicIPInfo{}, err
	}

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
			publicIPInfo := extractPublicIpDescribeInfo(addr)
			if len(publicIPInfo.Name) < 1 {
				cblogger.Errorf("PublicIp [%s]의 Name Tag가 없으므로 결과에서 제외함", publicIPInfo.PublicIP)
				continue
			}

			publicIpList = append(publicIpList, &publicIPInfo)
		}
	}

	return publicIpList, nil
}

//2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨.
func extractPublicIpDescribeInfo(allocRes *ec2.Address) irs.PublicIPInfo {
	var publicIPInfo irs.PublicIPInfo

	keyValueList := []irs.KeyValue{
		{Key: "Domain", Value: *allocRes.Domain},
		{Key: "PublicIpv4Pool", Value: *allocRes.PublicIpv4Pool},
		{Key: "AllocationId", Value: *allocRes.AllocationId},
	}

	spew.Dump(allocRes)
	publicIPInfo.PublicIP = *allocRes.PublicIp
	publicIPInfo.Id = *allocRes.AllocationId //2019-11-16 Name 기반으로 로직 변경
	//publicIPInfo.Domain = *allocRes.Domain
	//publicIPInfo.PublicIpv4Pool = *allocRes.PublicIpv4Pool
	//publicIPInfo.AllocationId = *allocRes.AllocationId

	if !reflect.ValueOf(allocRes.InstanceId).IsNil() {
		//publicIPInfo.InstanceId = *allocRes.InstanceId // AWS:연결된 VM
		publicIPInfo.OwnedVMID = *allocRes.InstanceId // AWS:연결된 VM
	}

	if !reflect.ValueOf(allocRes.AssociationId).IsNil() {
		//publicIPInfo.AssociationId = *allocRes.AssociationId // AWS:연결ID
		keyValueList = append(keyValueList, irs.KeyValue{Key: "AssociationId", Value: *allocRes.AssociationId}) // AWS:연결ID
	}

	if !reflect.ValueOf(allocRes.NetworkInterfaceId).IsNil() {
		//publicIPInfo.NetworkInterfaceId = *allocRes.NetworkInterfaceId // AWS:연결된 Nic
		//publicIPInfo.NetworkInterfaceOwnerId = *allocRes.NetworkInterfaceOwnerId
		//publicIPInfo.PrivateIpAddress = *allocRes.PrivateIpAddress

		keyValueList = append(keyValueList, irs.KeyValue{Key: "NetworkInterfaceId", Value: *allocRes.NetworkInterfaceId}) // AWS:연결된 Nic
		keyValueList = append(keyValueList, irs.KeyValue{Key: "NetworkInterfaceOwnerId", Value: *allocRes.NetworkInterfaceOwnerId})
		keyValueList = append(keyValueList, irs.KeyValue{Key: "PrivateIpAddress", Value: *allocRes.PrivateIpAddress})
	}

	//publicIPInfo.Name = *allocRes.AllocationId //2019-10-21 Name 필드에 ID 값을 리턴하도록 변경 (만약을 위해 Key&Value 목록에 Name 정보 리턴)
	//Name 태그 설정
	for _, t := range allocRes.Tags {
		if *t.Key == "Name" {
			//publicIPInfo.Name = *t.Value
			//cblogger.Debug("명칭 : ", publicIPInfo.Name)
			keyValueList = append(keyValueList, irs.KeyValue{Key: "Name", Value: *t.Value})
			publicIPInfo.Name = *t.Value //2019-11-16 NameId 기반으로 로직 변경
			break
		}
	}

	publicIPInfo.KeyValueList = keyValueList
	return publicIPInfo
}

//@TODO : 2차 정책에 의해 IP에서 할당ID 기반으로 변경함.
//2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨.
func (publicIpHandler *AwsPublicIPHandler) GetPublicIP(publicIPNameId string) (irs.PublicIPInfo, error) {
	cblogger.Infof("publicIPNameId : [%s]", publicIPNameId)
	var publicIPInfo irs.PublicIPInfo

	// Make the API request to EC2 filtering for the addresses in the account's VPC.
	result, err := publicIpHandler.Client.DescribeAddresses(&ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			{
				//Name: aws.String("public-ip"),
				//Name: aws.String("allocation-id"),
				//Values: aws.StringSlice([]string{"vpc"}),
				Name: aws.String("tag:Name"), // subnet-id
				Values: []*string{
					aws.String(publicIPNameId),
				},
			},
		},
	})
	if err != nil {
		cblogger.Error(err)
		return irs.PublicIPInfo{}, err
	}

	// Printout the IP addresses if there are any.
	if len(result.Addresses) == 0 {
		//cblogger.Errorf("Not found Elastic IP Information - Request allocation-id : [%s]", publicIPNameId)
		cblogger.Errorf("Not found Elastic IP Information - Request name : [%s]", publicIPNameId)
		return irs.PublicIPInfo{}, errors.New("PublicIP NotFound : " + publicIPNameId)
	} else {
		cblogger.Info("Elastic IPs")
		for _, addr := range result.Addresses {
			publicIPInfo = extractPublicIpDescribeInfo(addr)
		}
	}

	return publicIPInfo, nil
}

//2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨.
// Public IP를 완전히 제거 함.(AWS Pool로 되돌려 보냄)
func (publicIpHandler *AwsPublicIPHandler) DeletePublicIP(publicIPNameId string) (bool, error) {
	//cblogger.Infof("allocationId : [%s]", allocationId)
	cblogger.Infof("publicIPNameId : [%s]", publicIPNameId)

	publicInfo, errPublicInfo := publicIpHandler.GetPublicIP(publicIPNameId)
	if errPublicInfo != nil {
		return false, errPublicInfo
	}
	cblogger.Info(publicInfo)

	input := &ec2.ReleaseAddressInput{
		//AllocationId: aws.String(allocationId), //eipalloc-64d5890a - VPC에서 삭제
		AllocationId: aws.String(publicInfo.Id),
	}
	cblogger.Info(input)

	result, err := publicIpHandler.Client.ReleaseAddress(input)
	cblogger.Info(result)
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
