// Cloud Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

const (
	// default Resource GROUP Name
	CBResourceGroupName = "CB-GROUP"
	// default VPC Name
	CBVirutalNetworkName = "CB-VNet"
	// default CIDR Block
	CBVnetDefaultCidr = "130.0.0.0/16"

	// default Subnet Name
	CBSubnetName = "CB-VNet-Sub"

	// default Bandwidth is 5 Mbit/s
	CBBandwidth = "5"
	// default InstanceChargeType
	CBInstanceChargeType = "PostPaid"
	// default InternetChargeType
	CBInternetChargeType = "PayByTraffic"

	// default Tag Name
	CBMetaDefaultTagName = "cbCate"
	// default Tag Value
	CBMetaDefaultTagValue = "cbAlibaba"

	CBPageOn = true
	// page number for control pages
	CBPageNumber = 1
	// page size for control pages
	CBPageSize = 10
)

type AlibabaCBNetworkInfo struct {
	VpcName   string
	VpcId     string
	CidrBlock string
	IsDefault bool
	State     string

	SubnetName string
	SubnetId   string
}

func GetCBResourceGroupName() string {
	return CBResourceGroupName
}

//VPC
func GetCBVirutalNetworkName() string {
	return CBVirutalNetworkName
}

//Subnet
func GetCBSubnetName() string {
	return CBSubnetName
}

func GetCBVnetDefaultCidr() string {
	return CBVnetDefaultCidr
}

// func GetCBDefaultVNetName() string {
// 	return CBVirutalNetworkName
// }

// func GetCBDefaultSubnetName() string {
// 	return CBSubnetName
// }

// func GetCBDefaultCidrBlock() string {
// 	return CBVnetDefaultCidr
// }

//이 함수는 VPC & Subnet이 존재하는 곳에서만 사용됨.
//VPC & Subnet이 존재하는 경우 정보를 리턴하고 없는 경우 Default VPC & Subnet을 생성 후 정보를 리턴 함.
func (vNetworkHandler *AlibabaVNetworkHandler) GetAutoCBNetworkInfo() (AlibabaCBNetworkInfo, error) {
	var alibabaCBNetworkInfo AlibabaCBNetworkInfo

	subNetId := vNetworkHandler.GetMcloudBaristaDefaultSubnetId()
	if subNetId == "" {
		//내부에서 VPC를 자동으로 생성후 Subnet을 생성함.
		_, err := vNetworkHandler.CreateVNetwork(irs.VNetworkReqInfo{})
		if err != nil {
			cblogger.Error("Default VNetwork(VPC & Subnet) 자동 생성 실패")
			cblogger.Error(err)
			return AlibabaCBNetworkInfo{}, err
		}
	}

	//VPC & Subnet을 생성했으므로 예외처리 없이 조회만 처리함.
	alibabaVpcInfo, _ := vNetworkHandler.GetVpc(GetCBVirutalNetworkName())
	spew.Dump(alibabaVpcInfo)
	alibabaCBNetworkInfo.VpcId = alibabaVpcInfo.Id
	alibabaCBNetworkInfo.VpcName = alibabaVpcInfo.Name

	alibabaSubnetInfo, _ := vNetworkHandler.GetVNetwork("")
	spew.Dump(alibabaSubnetInfo)
	alibabaCBNetworkInfo.SubnetId = alibabaSubnetInfo.Id
	alibabaCBNetworkInfo.SubnetName = alibabaSubnetInfo.Name

	spew.Dump(alibabaCBNetworkInfo)

	return alibabaCBNetworkInfo, nil
}

func (vNetworkHandler *AlibabaVNetworkHandler) GetMcloudBaristaDefaultVpcId() string {
	alibabaVpcInfo, err := vNetworkHandler.GetVpc(GetCBVirutalNetworkName())
	if err != nil {
		// if aerr, ok := err.(errors.Error); ok {
		// 	switch aerr.Code() {
		// 	default:
		// 		cblogger.Error(aerr.Error())
		// 	}
		// } else {
		// 	cblogger.Error(err.Error())
		// }
		cblogger.Error(err.Error())
		return ""
	}

	//기존 정보가 존재하면...
	if alibabaVpcInfo.Id != "" {
		return alibabaVpcInfo.Id
	} else {
		return ""
	}
}

func (vNetworkHandler *AlibabaVNetworkHandler) GetMcloudBaristaDefaultSubnetId() string {
	alibabaSubnetInfo, err := vNetworkHandler.GetVNetwork("")
	if err != nil {
		// if aerr, ok := err.(errors.Error); ok {
		// 	switch aerr.Code() {
		// 	default:
		// 		cblogger.Error(aerr.Error())
		// 	}
		// } else {
		// 	cblogger.Error(err.Error())
		// }
		cblogger.Error(err.Error())
		return ""
	}

	//기존 정보가 존재하면...
	if alibabaSubnetInfo.Id != "" {
		return alibabaSubnetInfo.Id
	} else {
		return ""
	}
}

//@TODO : ListVNetwork()에서 호출되는 경우도 있기 때문에 필요하면 VPC조회와 생성을 별도의 Func으로 분리해야함.(일단은 큰 문제는 없어서 놔둠)
//CB Default Virtual Network가 존재하지 않으면 생성하며, 존재하는 경우 Vpc ID를 리턴 함.
func (vNetworkHandler *AlibabaVNetworkHandler) FindOrCreateMcloudBaristaDefaultVPC(vNetworkReqInfo irs.VNetworkReqInfo) (string, error) {
	cblogger.Info(vNetworkReqInfo)

	alibabaVpcInfo, err := vNetworkHandler.GetVpc(GetCBVirutalNetworkName())
	if err != nil {
		// if aerr, ok := err.(errors.Error); ok {
		// 	switch aerr.Code() {
		// 	default:
		// 		cblogger.Error(aerr.Error())
		// 	}
		// } else {
		// 	// Print the error, cast err to awserr.Error to get the Code and
		// 	// Message from an error.
		// 	cblogger.Error(err.Error())
		// }
		cblogger.Error(err.Error())
		return "", err
	}

	//기존 정보가 존재하면...
	if alibabaVpcInfo.Id != "" {
		return alibabaVpcInfo.Id, nil
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

		cblogger.Infof("기본 VPC[%s]가 없어서 CIDR[%s] 범위의 VPC를 자동으로 생성합니다.", GetCBVirutalNetworkName(), GetCBVnetDefaultCidr())
		alibabaVpcReqInfo := AlibabaVpcReqInfo{
			Name: GetCBVirutalNetworkName(),
			//CidrBlock: VpcCidrBlock,
			CidrBlock: GetCBVnetDefaultCidr(),
		}

		result, errVpc := vNetworkHandler.CreateVpc(alibabaVpcReqInfo)
		if errVpc != nil {
			cblogger.Error(errVpc)
			return "", errVpc
		}
		cblogger.Infof("CB Default VPC[%s] 생성 완료 - CIDR : [%s]", GetCBVirutalNetworkName(), result.CidrBlock)
		cblogger.Info(result)
		spew.Dump(result)

		return result.Id, nil
	}
}

//자동으로 생성된 VPC & Subnet을 삭제해도 되는가?
//명시적으로 Subnet 삭제의 호출이 없기 때문에 시큐리티 그룹이나 vNic이 삭제되는 시점에 호출됨.
func (vNetworkHandler *AlibabaVNetworkHandler) IsAvailableAutoCBNet() bool {
	return false
}

func SetNameTag(Client *ecs.Client, resourceId string, resourceType string, value string) bool {
	// Tag에 Name 설정
	cblogger.Infof("Name Tage 설정 - ResourceId : [%s]  Value : [%s] ", resourceId, value)

	request := ecs.CreateAddTagsRequest()
	request.Scheme = "https"

	request.ResourceType = resourceType // "disk", "instance", "image", "securitygroup", "snapshot"
	request.ResourceId = resourceId     // "i-t4n4qtfwa4w5aavx588v"
	request.Tag = &[]ecs.AddTagsTag{
		{
			Key:   "Name",
			Value: value, // "cbVal",
		},
		{
			Key:   "cbCate",
			Value: "cbAlibaba",
		},
		{
			Key:   "cbName",
			Value: value, // "cbVal",
		},
		// Resources: []*string{&Id},
	}
	_, errtag := Client.AddTags(request)
	if errtag != nil {
		cblogger.Error("Name Tag 설정 실패 : ")
		cblogger.Error(errtag)
		return false
	}

	return true
}

// 서브넷 CIDR 생성 (CIDR C class 기준 생성)
func CreateSubnetCIDR(subnetList []*irs.VNetworkInfo) (*string, error) {

	// CIDR C class 최대값 찾기
	maxClassNum := 0
	for _, subnet := range subnetList {
		addressArr := strings.Split(subnet.AddressPrefix, ".")
		if curClassNum, err := strconv.Atoi(addressArr[2]); err != nil {
			return nil, err
		} else {
			if curClassNum > maxClassNum {
				maxClassNum = curClassNum
			}
		}
	}

	if len(subnetList) == 0 {
		maxClassNum = 0
	} else {
		maxClassNum = maxClassNum + 1
	}

	// 서브넷 CIDR 할당
	vNetIP := strings.Split(CBVnetDefaultCidr, "/")
	vNetIPClass := strings.Split(vNetIP[0], ".")
	subnetCIDR := fmt.Sprintf("%s.%s.%d.0/24", vNetIPClass[0], vNetIPClass[1], maxClassNum)
	return &subnetCIDR, nil
}

// AssociationId 대신 PublicIP로도 가능 함.
func AssociatePublicIP(client *ecs.Client, allocationId string, instanceId string) (bool, error) {
	cblogger.Infof("ECS에 퍼블릭 IP할당 - AllocationId : [%s], InstanceId : [%s]", allocationId, instanceId)

	// ECS에 할당.
	// Associate the new Elastic IP address with an existing ECS instance.
	request := ecs.CreateAssociateEipAddressRequest()
	request.Scheme = "https"

	request.InstanceId = instanceId
	request.AllocationId = allocationId

	assocRes, err := client.AssociateEipAddress(request)
	spew.Dump(assocRes)
	cblogger.Infof("[%s] ECS에 EIP(AllocationId : [%s]) 할당 완료", instanceId, allocationId)
	// cblogger.Infof("[%s] ECS에 EIP(AllocationId : [%s]) 할당 완료 - AssociationId Id : [%s]", instanceId, allocationId, *assocRes.AssociationId)

	if err != nil {
		cblogger.Errorf("Unable to associate IP address with %s, %v", instanceId, err)
		// if aerr, ok := err.(awserr.Error); ok {
		// 	switch aerr.Code() {
		// 	default:
		// 		cblogger.Errorf(aerr.Error())
		// 	}
		// } else {
		// 	// Print the error, cast err to awserr.Error to get the Code and
		// 	// Message from an error.
		// 	cblogger.Errorf(err.Error())
		// }
		return false, err
	}

	cblogger.Info(assocRes)
	return true, nil
}
