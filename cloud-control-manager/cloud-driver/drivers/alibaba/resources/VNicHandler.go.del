// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by zephy@mz.co.kr, 2019.09.

package resources

//@TODO : Default VPC & Default Subnet 처리해야 함.
import (
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

//https://amzn.to/2L0lfQS
type AlibabaVNicHandler struct {
	Region idrv.RegionInfo
	Client *ecs.Client
}

//@TODO : 퍼블릭IP(EIP)는 이 곳이 아닌 VM생성 시 처리함. 이곳에서 처리해야 하면 구현해야 함.
func (vNicHandler *AlibabaVNicHandler) CreateVNic(vNicReqInfo irs.VNicReqInfo) (irs.VNicInfo, error) {

	return irs.VNicInfo{}, nil

	/*

		request := ecs.CreateCreateNetworkInterfaceRequest()
		request.Scheme = "https"

		// getVSWitch() GetVNetwork(vNetworkID string) (VNetworkInfo, error)
		alibabaVNetworkInfo, err := vNetworkHandler.GetVNetwork(vNicReqInfo.VNetName)
		if err != nil {
			cblogger.Error(err.Error())
			return irs.VNicInfo{}, err
		}

		//기존 정보가 존재하면...
		if alibabaVNetworkInfo.Id != "" {
			return irs.VNicInfo{}, nil
		} else {
			//vNetworkInfo := irs.VNetworkInfo{}
			vNetworkInfo := ExtractSubnetDescribeInfo(result.VSwitches[0])
			request.VSwitchId = vNetworkInfo.Id // "vsw-t4nrtfnolaw76jxxffcqu"
		}

		request.SecurityGroupId = vNicReqInfo.SecurityGroupIds[0] // "sg-t4naprq9s3l738l28y0f"
		// request.Tag = &[]ecs.CreateNetworkInterfaceTag{
		// 	{
		// 		Key: "cbName",
		// 		Value: "cbVal",
		// 	},
		// }
		// request.ResourceGroupId = "rg"
		// request.PrimaryIpAddress = "172.16.1.109"
		request.NetworkInterfaceName = vNicReqInfo.Name // "cb-eni-zep04"
		request.Description = vNicReqInfo.Name          // "cb eni zep04"

		// input := &ec2.CreateNetworkInterfaceInput{
		// 	Description: aws.String(vNicReqInfo.Name),
		// 	//PrivateIpAddress: aws.String("10.0.2.17"),
		// 	SubnetId: aws.String("subnet-0a25f65671fa64155"),
		// 	Groups:   aws.StringSlice(vNicReqInfo.SecurityGroupIds),
		// }

		/*===========
			//보안그룹 처리
			securityGroupIds := []*string{}
			for _, id := range vNicReqInfo.SecurityGroupIds {
				securityGroupIds = append(securityGroupIds, aws.String(id))
			}
			input.Groups = securityGroupIds
		==========/

		cblogger.Info(request)
		//spew.Dump(request)
		result, err := vNicHandler.Client.CreateNetworkInterface(request)
		//spew.Dump(result)
		cblogger.Info(result)
		if err != nil {
			if aerr, ok := err.(errors.Error); ok {
				switch aerr.Code() {
				default:
					cblogger.Error(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				cblogger.Error(err.Error())
			}
			return irs.VNicInfo{}, err
		}

		// 획득된 NetworkInterfaceId를 통해 조회
		vNicInfo, _ = vNicHandler.GetVNic(result.NetworkInterfaceId)
		return vNicInfo, nil
	*/
}

func (vNicHandler *AlibabaVNicHandler) ListVNic() ([]*irs.VNicInfo, error) {
	cblogger.Info("Start")
	return nil, nil

	/*

		request := ecs.CreateDescribeNetworkInterfacesRequest()
		request.Scheme = "https"

		alibabaVpcInfo, err := vNetworkHandler.GetVpc(GetCBDefaultVNetName())
		if err != nil {
			if aerr, ok := err.(errors.Error); ok {
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

		//기존 정보가 존재하면...
		if alibabaVpcInfo.Id != "" {
			request.VpcId = &[]string{alibabaVpcInfo.Id}
		} else {
			return alibabaVpcInfo.Id, nil
		}

		// request.NetworkInterfaceId = &[]string{"eni-t4naprq9s3l738l557xn", "eni-t4nc2pm81zn80cx9h1gn"}

		// input := &ec2.DescribeNetworkInterfacesInput{
		// 	NetworkInterfaceIds: []*string{
		// 		nil,
		// 	},
		// 	Filters: []*ec2.Filter{
		// 		{
		// 			Name:   aws.String("vpc-id"),
		// 			Values: aws.StringSlice([]string{"vpc-027696b302162edeb"}),
		// 		},
		// 	},
		// }

		result, err := vNicHandler.Client.DescribeNetworkInterfaces(request)
		if err != nil {
			if aerr, ok := err.(errors.Error); ok {
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

		var vNicInfoList []*irs.VNicInfo
		for _, cur := range result.NetworkInterfaceSet {
			cblogger.Infof("[%s] vNic 정보 처리", *cur.NetworkInterfaceId)
			vNicInfo := ExtractVNicDescribeInfo(cur)
			vNicInfoList = append(vNicInfoList, &vNicInfo)
		}

		return vNicInfoList, nil
	*/
}

//VNic 정보를 추출함
func ExtractVNicDescribeInfo(netIf *ecs.NetworkInterfaceSet) irs.VNicInfo {
	spew.Dump(netIf)
	return irs.VNicInfo{}
	/*
		vNicInfo := irs.VNicInfo{
			Id:     *netIf.NetworkInterfaceId,
			Status: *netIf.Status,
		}

		keyValueList := []irs.KeyValue{
			{Key: "Type", Value: *netIf.Type}, // Alibaba
			{Key: "VpcId", Value: *netIf.VpcId},
			{Key: "VSwitchId", Value: *netIf.VSwitchId},

			{Key: "ZoneId", Value: *netIf.ZoneId},
			{Key: "NetworkInterfaceName", Value: *netIf.NetworkInterfaceName},
			{Key: "InstanceId", Value: *netIf.InstanceId},
			{Key: "CreationTime", Value: *netIf.CreationTime},
			{Key: "ResourceGroupId", Value: *netIf.ResourceGroupId},
			{Key: "ServiceID", Value: *netIf.ServiceID},
			{Key: "ServiceManaged", Value: *netIf.ServiceManaged},

			// {Key: "AssociatedPublicIp", Value: *netIf.AssociatedPublicIp},
			// {Key: "PrivateIpSets", Value: *netIf.PrivateIpSets},

			// {Key: "OwnerId", Value: *netIf.OwnerId},
			{Key: "PrivateIpAddress", Value: *netIf.PrivateIpAddress},
			// {Key: "InterfaceType", Value: *netIf.InterfaceType},
			// {Key: "AvailabilityZone", Value: *netIf.AvailabilityZone},
		}

		if !reflect.ValueOf(netIf.MacAddress).IsNil() {
			vNicInfo.MacAdress = *netIf.MacAddress
		}

		// 할당된 VM 정보 조회
		vNicInfo.OwnedVMID = *netIf.InstanceId
		vNicInfo.PublicIP = *netIf.AssociatedPublicIp

		// if !reflect.ValueOf(netIf.Attachment).IsNil() {
		// 	//인스턴스에 할당된 경우
		// 	if !reflect.ValueOf(netIf.Attachment.InstanceId).IsNil() {
		// 		vNicInfo.OwnedVMID = *netIf.Attachment.InstanceId
		// 		keyValueList = append(keyValueList, irs.KeyValue{Key: "InstanceOwnerId", Value: *netIf.Attachment.InstanceOwnerId})

		// 		keyValueList = append(keyValueList, irs.KeyValue{Key: "AttachTime", Value: netIf.Attachment.AttachTime.String()})
		// 	}
		// }

		//보안그룹
		if !reflect.ValueOf(netIf.SecurityGroupIds).IsNil() {
			for _, t := range netIf.SecurityGroupIds {
				vNicInfo.SecurityGroupIds = append(vNicInfo.SecurityGroupIds, *t.SecurityGroupId)
			}
		}

		//Name은 Tag의 "Name" 속성에만 저장됨
		// cblogger.Debug("Name Tag 찾기")
		// for _, t := range netIf.TagSet {
		// 	if *t.Key == "Name" {
		// 		vNicInfo.Name = *t.Value
		// 		cblogger.Debug("vNic 명칭 : ", vNicInfo.Name)
		// 		break
		// 	}
		// }

		// if !reflect.ValueOf(netIf.Association).IsNil() {
		// 	vNicInfo.PublicIP = *netIf.Association.PublicIp

		// 	//keyValueList = append(keyValueList, irs.KeyValue{Key: "AllocationId", Value: *netIf.Association.AllocationId})
		// 	//keyValueList = append(keyValueList, irs.KeyValue{Key: "AssociationId", Value: *netIf.Association.AssociationId})
		// 	keyValueList = append(keyValueList, irs.KeyValue{Key: "IpOwnerId", Value: *netIf.Association.IpOwnerId})
		// }

		// 일부 이미지들은 아래 정보가 없어서 예외 처리 함.
		if !reflect.ValueOf(netIf.Description).IsNil() {
			keyValueList = append(keyValueList, irs.KeyValue{Key: "Description", Value: *netIf.Description})
		}

		vNicInfo.KeyValueList = keyValueList

		return vNicInfo
	*/
}

func (vNicHandler *AlibabaVNicHandler) GetVNic(vNicID string) (irs.VNicInfo, error) {
	cblogger.Info("vNicID : ", vNicID)
	return irs.VNicInfo{}, nil

	/*

		request := ecs.CreateDescribeNetworkInterfacesRequest()
		request.Scheme = "https"

		request.NetworkInterfaceId = &[]string{vNicID}

		// input := &ec2.DescribeNetworkInterfacesInput{
		// 	NetworkInterfaceIds: []*string{
		// 		aws.String(vNicID),
		// 	},
		// }

		result, err := vNicHandler.Client.DescribeNetworkInterfaces(request)
		if err != nil {
			if aerr, ok := err.(erros.Error); ok {
				switch aerr.Code() {
				default:
					cblogger.Error(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				cblogger.Error(err.Error())
			}
			return irs.VNicInfo{}, err
		}

		vNicInfo := ExtractVNicDescribeInfo(result.NetworkInterfaceSets.NetworkInterfaceSet[0])
		return vNicInfo, nil
	*/
}

func (vNicHandler *AlibabaVNicHandler) DeleteVNic(vNicID string) (bool, error) {
	cblogger.Info("vNicID : ", vNicID)
	return false, nil

	/*
		request := ecs.CreateDeleteNetworkInterfaceRequest()
		request.Scheme = "https"

		request.NetworkInterfaceId = vNicID // "NI"

		// input := &ec2.DeleteNetworkInterfaceInput{
		// 	NetworkInterfaceId: aws.String(vNicID),
		// }

		_, err := vNicHandler.Client.DeleteNetworkInterface(request)
		if err != nil {
			if aerr, ok := err.(errors.Error); ok {
				switch aerr.Code() {
				default:
					cblogger.Error(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				cblogger.Error(err.Error())
			}
			return false, err
		}

		return true, nil
	*/
}
