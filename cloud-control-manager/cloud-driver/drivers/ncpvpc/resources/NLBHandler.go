// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2022.10.

package resources

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	// "github.com/davecgh/go-spew/spew"

	ncloud "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	vlb "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vloadbalancer"
	vpc "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vpc"
	vserver "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"

	// cidr "github.com/apparentlymart/go-cidr/cidr"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVpcNLBHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *vserver.APIClient
	VPCClient      *vpc.APIClient
	VLBClient      *vlb.APIClient
}

const (
	// NCP VPC Cloud LB type code : 'APPLICATION' | 'NETWORK' | 'NETWORK_PROXY'
	NcpLbType						string = "NETWORK"

	// NCP VPC Cloud NLB network type code : 'PUBLIC' | 'PRIVATE' (Default: 'PUBLIC')
	NcpPublicNlBType   				string = "PUBLIC"
	NcpInternalNlBType 				string = "PRIVATE"

	// NCP LB performance(throughput) type code : 'SMALL' | 'MEDIUM' | 'LARGE' (Default: 'SMALL')
	// You can only select 'SMALL' if the LB type is 'NETWORK' and the LB network type is 'PRIVATE'.
	DefaultThroughputType 			string = "SMALL"

	// NCP VPC Cloud default value for Listener and Health Monitor
	DefaultConnectionLimit        	int32 = 60 // Min : 1, Max : 3600 sec(Dedicated LB : 1 ~ 480000). Default : 60 sec
	DefaultHealthCheckerInterval  	int32 = 30 // Min: 5, Max: 300 (seconds). Default: 30 seconds
	DefaultHealthCheckerThreshold 	int32 = 2  // Min: 2, Max: 10. Default: 2

	LbTypeSubnetDefaultCidr 		string = ".240/28"
	LbTypeSubnetDefaultName 		string = "ncpvpc-subnet-for-nlb"
)

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP VPC NLBHandler")
}

// Note : Cloud-Barista supports only this case => [ LB : Listener : VMGroup : Health Checker = 1 : 1 : 1 : 1 ]
// ------ NLB Management
func (nlbHandler *NcpVpcNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (createNLB irs.NLBInfo, newErr error) {
	cblogger.Info("NPC VPC Cloud Driver: called CreateNLB()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbReqInfo.IId.NameId, "CreateNLB()")

	if strings.EqualFold(nlbReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid NLB NameId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	// Note!! : NCP VPC LB type code : 'APPLICATION' | 'NETWORK' | 'NETWORK_PROXY'
	lbType := NcpLbType

	// Note!! : Only valid if NcpLbType is Not a 'NETWORK' type.
	// Min : 1, Max : 3600 sec(Dedicated LB : 1 ~ 480000). Default : 60 sec
	timeOut := int32(nlbReqInfo.HealthChecker.Timeout)
	if timeOut == -1 {
		timeOut = DefaultConnectionLimit
	}
	if timeOut < 1 || timeOut > 3600 {
		return irs.NLBInfo{}, fmt.Errorf("Invalid Timeout value. Must be a number between 1 and 3600.") // According to the NCP VPC API document.
	}

	// Note!! : NCP VPC LB network type code : 'PUBLIC' | 'PRIVATE' (Default: 'PUBLIC')
	var lbNetType string
	if strings.EqualFold(nlbReqInfo.Type, "PUBLIC") || strings.EqualFold(nlbReqInfo.Type, "default") || strings.EqualFold(nlbReqInfo.Type, "") {
		lbNetType = NcpPublicNlBType
	} else if strings.EqualFold(nlbReqInfo.Type, "INTERNAL") {
		lbNetType = NcpInternalNlBType
	}

	// LB performance(throughput) type code : 'SMALL' | 'MEDIUM' | 'LARGE' (Default: 'SMALL')
	// You can only select 'SMALL' if the LB type is 'NETWORK' and the LB network type is 'PRIVATE'.
	throughputType := DefaultThroughputType

	ncpVPCInfo, err := nlbHandler.GetNcpVpcInfoWithName(nlbReqInfo.VpcIID.NameId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NPC VPC Info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	cblogger.Infof("\n### ncpVPCInfo.VpcNo : [%s]", *ncpVPCInfo.VpcNo)

	// Caution!! : ### Need a Subnet for 'LB Only'('LB Type' Subnet)
	lbTypeSubnetId, err := nlbHandler.GetSubnetIdForNlbOnly(*ncpVPCInfo.VpcNo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the SubnetId of LB Type subnet : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	cblogger.Infof("\n### ncpVPCInfo.Ipv4CidrBlock : [%s]", *ncpVPCInfo.Ipv4CidrBlock)
	// VPC IP address ranges : /16~/28 in private IP range (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
	cidrBlock := strings.Split(*ncpVPCInfo.Ipv4CidrBlock, ".")
	cidrForNlbSubnet := cidrBlock[0] + "." + cidrBlock[1] + "." + cidrBlock[2] + LbTypeSubnetDefaultCidr
	// Ex) In case, VpcCIDR : "10.0.0.0/16",  
	// Subnet for VM : "10.0.0.0/28"
	// LB Type Subnet : "10.0.0.240/28"

	// ### In case, there is No Subnet for 'LB Only'('LB Type' subnet), Create the 'LB Type' subnet.
	if strings.EqualFold(lbTypeSubnetId, "") {
		cblogger.Info("\n# There is No Subnet for 'LB Only'('LB Type' Subnet), so it will be Created.")
		// NCP VPC Subnet Name Max length : 30
		// LbTypeSubnetDefaultName : "ncpvpc-subnet-for-nlb" => length : 21
		lbTypeSubnetName := LbTypeSubnetDefaultName + "-" + randSeq(8)
		cblogger.Infof("\n### Subnet Name for LB Type subnet : [%s]", lbTypeSubnetName)

		subnetReqInfo := irs.SubnetInfo{
			IId: irs.IID{
				NameId: lbTypeSubnetName,
			},
			IPv4_CIDR: cidrForNlbSubnet,
		}
		// Note : Create a 'LOADB' type of subnet ('LB Type' subnet for LB Only)
		ncpNlbSubnetInfo, err := nlbHandler.CreatNcpSubnetForNlbOnly(irs.IID{SystemId: *ncpVPCInfo.VpcNo}, subnetReqInfo) // Waitting time Included
		if err != nil {
			newErr := fmt.Errorf("Failed to Create the 'LB Type' Subnet : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.NLBInfo{}, newErr
		}
		cblogger.Infof("'LB type' Subnet ID : [%s]", *ncpNlbSubnetInfo.SubnetNo)
		lbTypeSubnetId = *ncpNlbSubnetInfo.SubnetNo
	}

	// To Get Subnet No list
	subnetNoList := []*string{ncloud.String(lbTypeSubnetId)}
	// cblogger.Infof("\n### ID list of 'LB Type' Subnet : ")
	// spew.Dump(subnetNoList)

	// Note!! : SubnetNoList[] : Range constraints: Minimum range of 1. Maximum range of 2.
	if len(subnetNoList) < 1 || len(subnetNoList) > 2 {
		newErr := fmt.Errorf("SubnetNoList range constraints : Min. range of 1. Max. range of 2.")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	lbReq := vlb.CreateLoadBalancerInstanceRequest{
		RegionCode:                  &nlbHandler.RegionInfo.Region,
		IdleTimeout:                 &timeOut,
		LoadBalancerNetworkTypeCode: &lbNetType,
		LoadBalancerTypeCode:        &lbType, 			// *** Required (Not Optional)
		LoadBalancerName:            &nlbReqInfo.IId.NameId,
		ThroughputTypeCode:          &throughputType,
		VpcNo:                       ncpVPCInfo.VpcNo, 	// *** Required (Not Optional)
		SubnetNoList:                subnetNoList,     	// *** Required (Not Optional)
	}

	// ### LoadBalancerSubnetList > PublicIpInstanceNo
	// Caution!! : About Public IP instance number
	// It's only valid when loadBalancerNetworkTypeCode is 'PUBLIC'.
	// It can only be used in the 'SGN'(Singapore) and 'JPN'(Japan)region.
	// Default: A new public IP is created and assigned.

	// vserver.~
	// createPublicIpInstance()
	// disassociatePublicIpFromServerInstance()

	callLogStart := call.Start()
	result, err := nlbHandler.VLBClient.V2Api.CreateLoadBalancerInstance(&lbReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New NLB : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if *result.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Create New NLB. NLB does Not Exist!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	} else {
		cblogger.Info("Succeeded in Creating New NLB.")
	}

	newNlbIID := irs.IID{SystemId: *result.LoadBalancerInstanceList[0].LoadBalancerInstanceNo}
	_, err = nlbHandler.WaitToGetNlbInfo(newNlbIID) // Wait until 'provisioningStatus' is "Running"
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait For Creating the NLB. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	ncpVMGroupInfo, err := nlbHandler.CreateVMGroup(*ncpVPCInfo.VpcNo, nlbReqInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create the VMGroup. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	cblogger.Infof("# VMGroupNo : [%s]", *ncpVMGroupInfo.TargetGroupNo)

	cblogger.Info("\n\n#### Waiting for Provisioning the New VMGroup!!")
	time.Sleep(20 * time.Second)

	ncpListenerInfo, err := nlbHandler.CreateListener(newNlbIID.SystemId, nlbReqInfo, *ncpVMGroupInfo.TargetGroupNo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create the Listener. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	cblogger.Infof("# LoadBalancerListenerNo : [%s]", *ncpListenerInfo.LoadBalancerListenerNo)

	cblogger.Info("\n\n#### Waiting for Changing the NLB Settings!!")
	_, err = nlbHandler.WaitToGetNlbInfo(newNlbIID) // Wait until 'provisioningStatus' is "Changing" -> "Running"
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait For Changing the NLB. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	nlbInfo, err := nlbHandler.GetNLB(newNlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Created NLB Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	return nlbInfo, nil
}

func (nlbHandler *NcpVpcNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {
	cblogger.Info("NPC VPC Cloud Driver: called ListNLB()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", "ListNLB()", "ListNLB()")

	lbReq := vlb.GetLoadBalancerInstanceListRequest{
		RegionCode: &nlbHandler.RegionInfo.Region, // CAUTION!! : Searching NLB Info by RegionCode (Not RegionNo)
	}

	callLogStart := call.Start()
	result, err := nlbHandler.VLBClient.V2Api.GetLoadBalancerInstanceList(&lbReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find NLB list from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var nlbInfoList []*irs.NLBInfo
	if *result.TotalRows < 1 {
		cblogger.Info("# NLB does Not Exist!!")
	} else {
		for _, nlb := range result.LoadBalancerInstanceList {
			nlbInfo, err := nlbHandler.MappingNlbInfo(*nlb)
			if err != nil {
				newErr := fmt.Errorf("Failed to Map NLB lnfo. : [%v]", err)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return nil, newErr
			}
			nlbInfoList = append(nlbInfoList, &nlbInfo)
		}
	}

	return nlbInfoList, nil
}

func (nlbHandler *NcpVpcNLBHandler) GetNLB(nlbIID irs.IID) (irs.NLBInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetNLB()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "GetNLB()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return irs.NLBInfo{}, newErr
	}

	ncpNlbInfo, err := nlbHandler.GetNcpNlbInfo(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NLB info from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	nlbInfo, err := nlbHandler.MappingNlbInfo(*ncpNlbInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Map the NLB Info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	return nlbInfo, nil
}

func (nlbHandler *NcpVpcNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called DeleteNLB()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "DeleteNLB()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	lbNoList := []*string{ncloud.String(nlbIID.SystemId)}

	lbReq := vlb.DeleteLoadBalancerInstancesRequest{
		RegionCode:                 &nlbHandler.RegionInfo.Region,
		LoadBalancerInstanceNoList: lbNoList,
		// ReturnPublicIpTogether: // It can only be used in the SGN(Singapore) and JPN(Japan) region. Default: 'true'
	}

	callLogStart := call.Start()
	result, err := nlbHandler.VLBClient.V2Api.DeleteLoadBalancerInstances(&lbReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the NLB : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if !strings.EqualFold(*result.ReturnMessage, "success") {
		newErr := fmt.Errorf("Failed to Delete the NLB!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	} else {
		cblogger.Info("Succeeded in Deleting the NLB.")
	}

	newNlbIID := irs.IID{SystemId: *result.LoadBalancerInstanceList[0].LoadBalancerInstanceNo}
	_, err = nlbHandler.WaitForDelNlb(newNlbIID) // Wait until 'provisioningStatus' is "Terminated"
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait For Deleting the NLB. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		// return false, newErr // Catuton!! : Incase the status is 'Terminated', fail to get NLB info.
	}

	// Cleanup the rest resources(VMGroup, LB Type subnet) of the NLB
	_, cleanErr := nlbHandler.CleanUpNLB(result.LoadBalancerInstanceList[0].VpcNo)
	if cleanErr != nil {
		newErr := fmt.Errorf("Failed to Cleanup the rest resources of the NLB with the VPC ID. [%v]", cleanErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	return true, nil
}

func (nlbHandler *NcpVpcNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called AddVMs()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "AddVMs()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	nlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	vmHandler := NcpVpcVMHandler{
		RegionInfo: nlbHandler.RegionInfo,
		VMClient:   nlbHandler.VMClient,
	}

	var newVmIdList []*string
	if len(*vmIIDs) > 0 {
		for _, vmIID := range *vmIIDs {
			vmId, err := vmHandler.GetVmIdByName(vmIID.NameId)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the VM ID with the VM Name : [%v]", err)
				cblogger.Error(newErr.Error())
				return irs.VMGroupInfo{}, newErr
			}
			newVmIdList = append(newVmIdList, ncloud.String(vmId)) // ncloud.String func(v string) *string
		}
	} else {
		newErr := fmt.Errorf("Failded to Find any VM NameId to Add to the VMGroup!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	addReq := vlb.AddTargetRequest{
		RegionCode:    &nlbHandler.RegionInfo.Region,
		TargetGroupNo: &nlbInfo.VMGroup.CspID,
		TargetNoList:  newVmIdList,
	}
	
	callLogStart := call.Start()
	result, err := nlbHandler.VLBClient.V2Api.AddTarget(&addReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Add the VM to the Target Group : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if *result.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Add Any VM to the Target Group!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	} else {
		cblogger.Info("Succeeded in Adding New VM to the Target Group.")
	}

	cblogger.Info("\n\n#### Waiting for Changing the NLB Settings!!")
	_, err = nlbHandler.WaitToGetNlbInfo(nlbIID) // Wait until 'provisioningStatus' is "Changing" -> "Running"
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait For Changing the NLB. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	newVMGroupNlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	return newVMGroupNlbInfo.VMGroup, nil
}

func (nlbHandler *NcpVpcNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called RemoveVMs()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "RemoveVMs()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	nlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	// if len(*nlbInfo.VMGroup.VMs) < 1 {
	// 	// newErr := fmt.Errorf("The NLB does Not have any VM Member!!")
	// 	// cblogger.Error(newErr.Error())
	// 	// LoggingError(callLogInfo, newErr)
	// 	// return false, newErr
	// } else {
	// }

	vmHandler := NcpVpcVMHandler{
		RegionInfo: nlbHandler.RegionInfo,
		VMClient:   nlbHandler.VMClient,
	}

	var vmIdList []*string
	if len(*vmIIDs) > 0 {
		for _, vmIID := range *vmIIDs {
			vmId, err := vmHandler.GetVmIdByName(vmIID.NameId)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the VM ID with the VM Name : [%v]", err)
				cblogger.Error(newErr.Error())
				return false, newErr
			}
			vmIdList = append(vmIdList, ncloud.String(vmId)) // ncloud.String func(v string) *string
		}
	} else {
		newErr := fmt.Errorf("Failed to Find any VM NameId to Remove from the VMGroup!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	removeReq := vlb.RemoveTargetRequest{
		RegionCode:    &nlbHandler.RegionInfo.Region,
		TargetGroupNo: &nlbInfo.VMGroup.CspID,
		TargetNoList:  vmIdList,
	}

	callLogStart := call.Start()
	result, err := nlbHandler.VLBClient.V2Api.RemoveTarget(&removeReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Remove the VM frome the VMGroup : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if !strings.EqualFold(*result.ReturnMessage, "success") {
		newErr := fmt.Errorf("Failed to Remove the VM from the VMGroup!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	} else {
		cblogger.Info("Succeeded in Removing the VM from the VMGroup!!")
	}

	cblogger.Info("\n\n#### Waiting for Changing the NLB Settings!!")
	_, err = nlbHandler.WaitToGetNlbInfo(nlbIID) // Wait until 'provisioningStatus' is "Changing" -> "Running"
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait For Changing the NLB. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	return true, nil

}

func (nlbHandler *NcpVpcNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetVMGroupHealthInfo()")
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "GetVMGroupHealthInfo()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthInfo{}, newErr 
	}

	nlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB info!! [%v]", err)		
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthInfo{}, newErr
	}

	vmMemberList, err := nlbHandler.GetNcpTargetVMList(nlbInfo.VMGroup.CspID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VM Member list. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		return irs.HealthInfo{}, newErr
	}

	var vmGroupHealthInfo irs.HealthInfo
	if len(vmMemberList) > 0 {
		var allVMs []irs.IID
		var healthVMs []irs.IID
		var unHealthVMs []irs.IID
	
		vmHandler := NcpVpcVMHandler{
			RegionInfo: nlbHandler.RegionInfo,
			VMClient:   nlbHandler.VMClient,
		}

		for _, member := range vmMemberList {
			vm, err := vmHandler.GetNcpVMInfo(*member.TargetNo)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the NCP VM Info with Target No. [%v]", err.Error())
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.HealthInfo{}, newErr
			}
	
			allVMs = append(allVMs, irs.IID{NameId: *vm.ServerName, SystemId: *vm.ServerInstanceNo})  // Caution : Not 'VM Member ID' but 'VM System ID'
	
			// HealthCheckStatus : UP(Health UP), DOWN(Health DOWN), UNUSED(Health UNUSED)
			if strings.EqualFold(*member.HealthCheckStatus.Code, "UP") {
				cblogger.Infof("\n### [%s] is Healthy VM.", "")
				healthVMs = append(healthVMs, irs.IID{NameId: *vm.ServerName, SystemId: *vm.ServerInstanceNo})
			} else {
				cblogger.Infof("\n### [%s] is Unhealthy VM.", "")
				unHealthVMs = append(unHealthVMs, irs.IID{NameId: *vm.ServerName, SystemId: *vm.ServerInstanceNo})  // In case of "INACTIVE", ...
			}
		}
	
		vmGroupHealthInfo = irs.HealthInfo{
			AllVMs:       &allVMs,
			HealthyVMs:   &healthVMs,
			UnHealthyVMs: &unHealthVMs,
		}
		return vmGroupHealthInfo, nil
	} else {
		return irs.HealthInfo{}, nil
	}
}

func (nlbHandler *NcpVpcNLBHandler) CreateVMGroup(vpcId string, nlbReqInfo irs.NLBInfo) (*vlb.TargetGroup, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateVMGroup()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", "CreateVMGroup()", "CreateVMGroup()")

	// REST API and Resource Constraints Ref :
	// https://api.ncloud-docs.com/docs/en/networking-vloadbalancer-targetgroup-createtargetgroup

	if strings.EqualFold(vpcId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	vmGroupPort, err := strconv.Atoi(nlbReqInfo.VMGroup.Port)
	if err != nil {
		newErr := fmt.Errorf("Invalid VMGroup Port. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	if vmGroupPort < 1 || vmGroupPort > 65535 {
		newErr := fmt.Errorf("Invalid VMGroup Port.(Must be between 1 and 65535)")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	int32vmGroupPort := int32(vmGroupPort)

	// # The available protocol types for VM Group of NCP LB
	// Network Load Balancer : TCP / UDP
	// Network Proxy Load Balancer : PROXY_TCP
	// Application Load Balancer : HTTP / HTTPS
	// # Caution!! : The UDP protocol can only be used in the SGN(Singapore) and JPN(Japan) region.
	vmGroupProtocol := strings.ToUpper(nlbReqInfo.VMGroup.Protocol)
	switch vmGroupProtocol {
	case "TCP", "UDP":
		cblogger.Infof("\n# VMGroup Protocol : [%s]", vmGroupProtocol)
	default:
		newErr := fmt.Errorf("Invalid VMGroup Protocol Type. NCP VPC 'Network' Type LB VMGroup supports only TCP or UDP protocol!!")  // According to the NCP VPC API document
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr 
	}

	healthCheckPort, err := strconv.Atoi(nlbReqInfo.HealthChecker.Port)
	if err != nil {
		newErr := fmt.Errorf("Invalid HealthChecker Port. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	if healthCheckPort < 1 || healthCheckPort > 65535 {
		newErr := fmt.Errorf("Invalid HealthChecker Port.(Must be between 1 and 65535)")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	int32healthCheckPort := int32(healthCheckPort)

	// # The selectable health check protocol type is limited, depending on the VM Group protocol type.
	// TCP / PROXY_TCP : TCP
	// HTTP / HTTPS : HTTP / HTTPS
	healthCheckProtocol := strings.ToUpper(nlbReqInfo.HealthChecker.Protocol)
	if strings.EqualFold(vmGroupProtocol, "TCP") || strings.EqualFold(vmGroupProtocol, "PROXY_TCP") {
		if strings.EqualFold(healthCheckProtocol, "TCP") {
			cblogger.Infof("\n# HealthChecker Protocol : [%s]", healthCheckProtocol)
		} else {
			newErr := fmt.Errorf("Invalid HealthChecker Protocol Type!! (Must be 'TCP', when VMGroup Protocol is 'TCP' or 'PROXY_TCP' for NCP VPC Cloud.)")
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return nil, newErr
		}
	}
	if strings.EqualFold(vmGroupProtocol, "HTTP") || strings.EqualFold(vmGroupProtocol, "HTTPS") {
		if strings.EqualFold(healthCheckProtocol, "HTTP") || strings.EqualFold(healthCheckProtocol, "HTTPS") {
			cblogger.Infof("\n# HealthChecker Protocol : [%s]", healthCheckProtocol)
		} else {
			newErr := fmt.Errorf("Invalid HealthChecker Protocol Type!! (Must be 'HTTP' or 'HTTPS', when VMGroup Protocol is 'HTTP' or 'HTTPS' for NCP VPC Cloud.)")
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return nil, newErr
		}
	}

	// Note!! : Range constraints: Min: 5, Max: 300 (seconds), Default: 30 seconds
	healthCheckCycle := int32(nlbReqInfo.HealthChecker.Interval)
	if healthCheckCycle == -1 {
		healthCheckCycle = DefaultHealthCheckerInterval
	}
	if healthCheckCycle < 5 || healthCheckCycle > 300 {
		return nil, fmt.Errorf("Invalid HealthChecker Interval value. Must be a number between 5 and 300 seconds.") // According to the NCP VPC API document.
	}

	// Note!! : Range constraints: Min: 2, Max: 10, Default: 2
	healthCheckThreshold := int32(nlbReqInfo.HealthChecker.Threshold)
	if healthCheckThreshold == -1 {
		healthCheckThreshold = DefaultHealthCheckerThreshold
	}
	if healthCheckThreshold < 2 || healthCheckThreshold > 10 {
		return nil, fmt.Errorf("Invalid HealthChecker Threshold value. Must be a number between 2 and 10.") // According to the NCP VPC API document.
	}

	// To get TargetNoList
	vmHandler := NcpVpcVMHandler{
		RegionInfo: nlbHandler.RegionInfo,
		VMClient:   nlbHandler.VMClient,
	}

	var targetNoList []*string
	if len(*nlbReqInfo.VMGroup.VMs) > 0 {
		for _, vmIID := range *nlbReqInfo.VMGroup.VMs {
			vmId, err := vmHandler.GetVmIdByName(vmIID.NameId)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the VM ID with the VM Name : [%v]", err)
				cblogger.Error(newErr.Error())
				return nil, newErr
			}
			targetNoList = append(targetNoList, ncloud.String(vmId)) // ncloud.String func(v string) *string
		}
	}

	// Note : TargetGroupProtocolTypeCode
	// Network Load Balancer : 'TCP' / 'UDP'
	// Network Proxy Load Balancer : 'PROXY_TCP'
	// Application Load Balancer : 'HTTP' / 'HTTPS'
	// ### The 'UDP' protocol can only be used in the SGN(Singapore) and JPN(Japan) region.
	targetGroupReq := vlb.CreateTargetGroupRequest{
		RegionCode:                  &nlbHandler.RegionInfo.Region,
		TargetGroupPort:             &int32vmGroupPort,             // Caution!! : Range constraints: Min: 1, Max: 65534
		TargetGroupProtocolTypeCode: &vmGroupProtocol,              // *** Required (Not Optional)
		HealthCheckCycle:            &healthCheckCycle,             // Caution!! : Range constraints: Min: 5, Max: 300 (seconds), Default: 30 seconds
		HealthCheckPort:             &int32healthCheckPort,
		HealthCheckProtocolTypeCode: &healthCheckProtocol,  		// *** Required (Not Optional)
		HealthCheckUpThreshold:      &healthCheckThreshold, 		// Caution!! : Range constraints: Min: 2, Max: 10, Default: 2
		TargetNoList:                targetNoList,
		VpcNo:                       &vpcId, 						// *** Required (Not Optional)
	}

	callLogStart := call.Start()
	result, err := nlbHandler.VLBClient.V2Api.CreateTargetGroup(&targetGroupReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New TargetGroup : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if *result.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Create New TargetGroup. TargetGroup does Not Exist!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Creating New TargetGroup.")
	}

	return result.TargetGroupList[0], nil
}

func (nlbHandler *NcpVpcNLBHandler) DeleteVMGroup(vmGroupId string) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called DeleteNLB()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", vmGroupId, "DeleteNcpVMGroup()")

	if strings.EqualFold(vmGroupId, "") {
		newErr := fmt.Errorf("Invalid VMGroup ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	vmGroupNoList := []*string{ncloud.String(vmGroupId)}
	vmGroupReq := vlb.DeleteTargetGroupsRequest{
		RegionCode:        &nlbHandler.RegionInfo.Region,
		TargetGroupNoList: vmGroupNoList,
	}

	callLogStart := call.Start()
	result, err := nlbHandler.VLBClient.V2Api.DeleteTargetGroups(&vmGroupReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the VMGroup : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if !strings.EqualFold(*result.ReturnMessage, "success") {
		newErr := fmt.Errorf("Failed to Delete the VMGroup!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	} else {
		cblogger.Info("Succeeded in Deleting the VMGroup.")
	}

	return true, nil
}

func (nlbHandler *NcpVpcNLBHandler) CreateListener(nlbId string, nlbReqInfo irs.NLBInfo, vmGroupNo string) (*vlb.LoadBalancerListener, error) {
	cblogger.Info("NHN Cloud Driver: called CreateListener()")
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbReqInfo.IId.NameId, "CreateListener()")

	if strings.EqualFold(nlbId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	portNum, err := strconv.Atoi(nlbReqInfo.Listener.Port)
	if err != nil {
		newErr := fmt.Errorf("Invalid Listener Port. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	if portNum < 1 || portNum > 65535 {
		newErr := fmt.Errorf("Invalid Listener Port.(Must be between 1 and 65535)")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	listenerProtocol := strings.ToUpper(nlbReqInfo.Listener.Protocol)
	switch listenerProtocol {
	case "TCP", "UDP":
		cblogger.Infof("\n# Listener Protocol : [%s]", listenerProtocol)
	default:
		newErr := fmt.Errorf("Invalid Listener Protocol Type. NCP VPC 'Network' Type LB Listener supports only TCP or UDP protocol!!")  // According to the NCP VPC API document
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr 
	}

	int32PortNum := int32(portNum)
	listenerReq := vlb.CreateLoadBalancerListenerRequest{
		RegionCode:             &nlbHandler.RegionInfo.Region,
		LoadBalancerInstanceNo: &nlbId,                        // *** Required (Not Optional)
		Port:                   &int32PortNum,                 // *** Required (Not Optional)
		ProtocolTypeCode:       &listenerProtocol,             // *** Required (Not Optional)
		TargetGroupNo:          &vmGroupNo,                    // *** Required (Not Optional)
	}

	callLogStart := call.Start()
	result, err := nlbHandler.VLBClient.V2Api.CreateLoadBalancerListener(&listenerReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New Listener : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if *result.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Create New Listener. Listener does Not Exist!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Creating New Listener.")
	}

	return result.LoadBalancerListenerList[0], nil
}

func (nlbHandler *NcpVpcNLBHandler) GetListenerInfo(listenerId string, loadBalancerId string) (*irs.ListenerInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetListenerInfo()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", listenerId, "GetListenerInfo()")

	if strings.EqualFold(listenerId, "") {
		newErr := fmt.Errorf("Invalid Listener ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	listenerReq := vlb.GetLoadBalancerListenerListRequest{
		RegionCode:             &nlbHandler.RegionInfo.Region, // *** Required (Not Optional)
		LoadBalancerInstanceNo: &loadBalancerId,               // *** Required (Not Optional)
	}

	callLogStart := call.Start()
	result, err := nlbHandler.VLBClient.V2Api.GetLoadBalancerListenerList(&listenerReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find Listener list from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	// cblogger.Info("\n### GetLoadBalancerListenerList result")
	// spew.Dump(result)

	var listenerInfo *irs.ListenerInfo
	if *result.TotalRows < 1 {
		cblogger.Info("### Listener does Not Exist!!")
	} else {
		cblogger.Info("Succeeded in Getting Listener list from NCP VPC.")
		for _, listener := range result.LoadBalancerListenerList {
			if strings.EqualFold(*listener.LoadBalancerListenerNo, listenerId) {
				ncpListenerInfo, err := nlbHandler.MappingListenerInfo(*listener)
				if err != nil {
					newErr := fmt.Errorf("Failed to Map NLB Listener lnfo. : [%v]", err)
					cblogger.Error(newErr.Error())
					LoggingError(callLogInfo, newErr)
					return nil, newErr
				}
				listenerInfo = &ncpListenerInfo
				break
			}
		}
	}
	return listenerInfo, nil
}

func (nlbHandler *NcpVpcNLBHandler) GetVMGroupInfo(nlb vlb.LoadBalancerInstance) (irs.VMGroupInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetVMGroupInfo()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", *nlb.LoadBalancerInstanceNo, "GetVMGroupInfo()")

	if strings.EqualFold(*nlb.VpcNo, "") {
		newErr := fmt.Errorf("Invalid LoadBalancer VPC No.!!")
		cblogger.Error(newErr.Error())
		return irs.VMGroupInfo{}, newErr
	}

	// Note : Cloud-Barista supports only this case => [ LB : Listener : VM Group : Health Checker = 1 : 1 : 1 : 1 ]
	ncpTargetGroupList, err := nlbHandler.GetNcpTargetGroupListWithVpcId(*nlb.VpcNo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP TargetGroup List with the VPC ID. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	if len(ncpTargetGroupList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any NCP TargetGroup. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	// cblogger.Info("\n### ncpTargetGroupList")
	// spew.Dump(ncpTargetGroupList)

	vmGroupInfo := irs.VMGroupInfo{
		Protocol: *ncpTargetGroupList[0].TargetGroupProtocolType.Code,
		Port:     strconv.FormatInt(int64(*ncpTargetGroupList[0].TargetGroupPort), 10),
		CspID:    *ncpTargetGroupList[0].TargetGroupNo,
	}

	targetVmList, err := nlbHandler.GetNcpTargetVMList(*ncpTargetGroupList[0].TargetGroupNo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP VPC Target Members. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	// cblogger.Info("\n### targetList")
	// spew.Dump(targetList)

	if len(targetVmList) > 0 {
		vmHandler := NcpVpcVMHandler{
			RegionInfo: nlbHandler.RegionInfo,
			VMClient:   nlbHandler.VMClient,
		}

		var vmIIds []irs.IID
		for _, member := range targetVmList {
			vm, err := vmHandler.GetNcpVMInfo(*member.TargetNo)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the NCP VM Info with Target No. [%v]", err.Error())
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMGroupInfo{}, newErr
			}

			vmIIds = append(vmIIds, irs.IID{
				NameId:   *vm.ServerName,
				SystemId: *vm.ServerInstanceNo,
			})
		}
		vmGroupInfo.VMs = &vmIIds
	} else {
		cblogger.Info("The VMGroup does Not have any VM Member!!")
	}

	keyValueList := []irs.KeyValue{
		{Key: "AlgorithmType", Value: *ncpTargetGroupList[0].AlgorithmType.CodeName},
		{Key: "TargetType", Value: *ncpTargetGroupList[0].TargetType.CodeName},
	}
	vmGroupInfo.KeyValueList = keyValueList

	return vmGroupInfo, nil
}

func (nlbHandler *NcpVpcNLBHandler) GetHealthCheckerInfo(nlb vlb.LoadBalancerInstance) (irs.HealthCheckerInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetHealthCheckerInfo()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", *nlb.LoadBalancerInstanceNo, "GetHealthCheckerInfo()")

	if strings.EqualFold(*nlb.VpcNo, "") {
		newErr := fmt.Errorf("Invalid LoadBalancer VPC No.!!")
		cblogger.Error(newErr.Error())
		return irs.HealthCheckerInfo{}, newErr
	}

	// Note : Cloud-Barista supports only this case => [ LB : Listener : VM Group : Health Checker = 1 : 1 : 1 : 1 ]
	ncpTargetGroupList, err := nlbHandler.GetNcpTargetGroupListWithVpcId(*nlb.VpcNo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP TargetGroup List with the VPC ID. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthCheckerInfo{}, newErr
	}
	if len(ncpTargetGroupList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any NCP TargetGroup. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthCheckerInfo{}, newErr
	}

	// cblogger.Info("\n### ncpTargetGroupList")
	// spew.Dump(ncpTargetGroupList)

	healthCheckerInfo := irs.HealthCheckerInfo{
		Protocol: *ncpTargetGroupList[0].HealthCheckProtocolType.Code,
		Port:     strconv.FormatInt(int64(*ncpTargetGroupList[0].HealthCheckPort), 10),
		Interval: int(*ncpTargetGroupList[0].HealthCheckCycle),
		// Timeout: int,
		Threshold: int(*ncpTargetGroupList[0].HealthCheckUpThreshold),
		CspID:    *ncpTargetGroupList[0].TargetGroupNo,
	}
	return healthCheckerInfo, nil
}

func (nlbHandler *NcpVpcNLBHandler) GetNcpTargetGroupListWithVpcId(vpcId string) ([]*vlb.TargetGroup, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetNcpTargetGroupListWithVpcId()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", vpcId, "GetNcpTargetGroupListWithVpcId()")

	if strings.EqualFold(vpcId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	targetGroupReq := vlb.GetTargetGroupListRequest{
		RegionCode: &nlbHandler.RegionInfo.Region,
		VpcNo:      &vpcId,
	}

	callLogStart := call.Start()
	result, err := nlbHandler.VLBClient.V2Api.GetTargetGroupList(&targetGroupReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find TargetGroup List from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if *result.TotalRows < 1 {
		cblogger.Info("### TargetGroup does Not Exist!!")
	}

	return result.TargetGroupList, nil
}

// Get VM Members of the NLB with the targetGroupId
func (nlbHandler *NcpVpcNLBHandler) GetNcpTargetVMList(targetGroupId string) ([]*vlb.Target, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetNcpTargetVMList()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", targetGroupId, "GetNcpTargetVMList()")

	if strings.EqualFold(targetGroupId, "") {
		newErr := fmt.Errorf("Invalid TargetGroup ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	targetReq := vlb.GetTargetListRequest{
		RegionCode:    &nlbHandler.RegionInfo.Region,
		TargetGroupNo: &targetGroupId,
	}

	callLogStart := call.Start()
	result, err := nlbHandler.VLBClient.V2Api.GetTargetList(&targetReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find Target List from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if *result.TotalRows < 1 {
		cblogger.Info("### The VMGroup does Not have any VM Member!!")
		return nil, nil // Caution!!
	}

	return result.TargetList, nil
}

func (nlbHandler *NcpVpcNLBHandler) GetNcpVpcInfoWithName(vpcName string) (*vpc.Vpc, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetNPCVpcInfoWithName()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", vpcName, "GetNPCVpcInfoWithName()")

	if strings.EqualFold(vpcName, "") {
		newErr := fmt.Errorf("Invalid VPC Name!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	vpcListReq := vpc.GetVpcListRequest{
		RegionCode: &nlbHandler.RegionInfo.Region,
	}

	callLogStart := call.Start()
	result, err := nlbHandler.VPCClient.V2Api.GetVpcList(&vpcListReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VPC List from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if *result.TotalRows < 1 {
		cblogger.Info("### VPC does Not Exist!!")
	} else {
		for _, vpc := range result.VpcList {
			if strings.EqualFold(*vpc.VpcName, vpcName) {
				return vpc, nil
			}
		}
	}

	return nil, fmt.Errorf("Failed to Find VPC Info with the name.")
}

// Get SubnetId for 'LB Only' subnet('LB Type' subnet)
func (nlbHandler *NcpVpcNLBHandler) GetSubnetIdForNlbOnly(vpcId string) (string, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetSubnetIdForNLB()")

	vpcHandler := NcpVpcVPCHandler{
		RegionInfo: nlbHandler.RegionInfo,
		VPCClient:  nlbHandler.VPCClient,
	}

	subnetInfoList, err := vpcHandler.ListSubnet(&vpcId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get SubnetList with the VPC No. : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	if len(subnetInfoList) < 1 {
		newErr := fmt.Errorf("### The VPC has No Subnet!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	var subnetId string
	for _, subnetInfo := range subnetInfoList {
		// Use Key/Value info of the subnetInfo
		for _, keyInfo := range subnetInfo.KeyValueList {
			if strings.EqualFold(keyInfo.Key, "UsageType") {
				if strings.EqualFold(keyInfo.Value, "LOADB") {
					subnetId = subnetInfo.IId.SystemId
					break
				}
			}
		}
	}
	return subnetId, nil
}

func (nlbHandler *NcpVpcNLBHandler) WaitToGetNlbInfo(nlbIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called WaitToGetNlbInfo()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	curRetryCnt := 0
	maxRetryCnt := 1000
	for {
		curRetryCnt++
		nlbStatus, err := nlbHandler.GetNcpNlbStatus(nlbIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the NLB Provisioning Status : [%v]", err)
			cblogger.Error(newErr.Error())
			return false, newErr
		} else if strings.EqualFold(nlbStatus, "Running") {
			return true, nil
		}
		time.Sleep(5 * time.Second)
		if curRetryCnt > maxRetryCnt {
			return false, fmt.Errorf("Failed to Create the NLB. Exceeded maximum retry count %d", maxRetryCnt)
		}
	}
}

func (nlbHandler *NcpVpcNLBHandler) WaitForDelNlb(nlbIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called WaitForDelNlb()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	curRetryCnt := 0
	maxRetryCnt := 500
	for {
		curRetryCnt++
		nlbStatus, err := nlbHandler.GetNcpNlbStatus(nlbIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the NLB Provisioning Status : [%v]", err)
			cblogger.Error(newErr.Error())
			return false, newErr
		} else if !strings.EqualFold(nlbStatus, "Running") && !strings.EqualFold(nlbStatus, "Terminating") {
			return true, nil
		}
		time.Sleep(3 * time.Second)
		if curRetryCnt > maxRetryCnt {
			return false, fmt.Errorf("Failed to Del the NLB. Exceeded maximum retry count %d", maxRetryCnt)
		}
	}
}

// NCP VPC LoadBalancerInstanceStatusName : Creating, Running, Changing, Terminating, Terminated, Repairing
func (nlbHandler *NcpVpcNLBHandler) GetNcpNlbStatus(nlbIID irs.IID) (string, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetNcpNlbStatus()")
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "GetNcpNlbStatus()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	ncpNlbInfo, err := nlbHandler.GetNcpNlbInfo(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NCP VPC NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	cblogger.Infof("\n### NLB Status : [%s]", *ncpNlbInfo.LoadBalancerInstanceStatusName)
	return *ncpNlbInfo.LoadBalancerInstanceStatusName, nil
}

func (nlbHandler *NcpVpcNLBHandler) GetNcpNlbInfo(nlbIID irs.IID) (*vlb.LoadBalancerInstance, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetNcpNlbInfo()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "GetNcpNlbInfo()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	lbReq := vlb.GetLoadBalancerInstanceDetailRequest{
		RegionCode:             &nlbHandler.RegionInfo.Region, // CAUTION!! : Searching NLB Info by RegionCode (Not RegionNo)
		LoadBalancerInstanceNo: &nlbIID.SystemId,
	}

	callLogStart := call.Start()
	result, err := nlbHandler.VLBClient.V2Api.GetLoadBalancerInstanceDetail(&lbReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NLB Info from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if *result.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Get Any NLB Info with the ID from NCP VPC!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Getting the NLB Info.")
	}

	return result.LoadBalancerInstanceList[0], nil
}

func (nlbHandler *NcpVpcNLBHandler) GetNcpNlbListWithVpcId(vpcId *string) ([]*vlb.LoadBalancerInstance, error) {
	cblogger.Info("NPC VPC Cloud Driver: called GetNcpNlbListWithVpcId()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", "GetNcpNlbListWithVpcId()", "GetNcpNlbListWithVpcId()")

	if strings.EqualFold(*vpcId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	lbReq := vlb.GetLoadBalancerInstanceListRequest{
		RegionCode: &nlbHandler.RegionInfo.Region,
		VpcNo:		vpcId,
	}

	callLogStart := call.Start()
	result, err := nlbHandler.VLBClient.V2Api.GetLoadBalancerInstanceList(&lbReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find NLB list from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	return result.LoadBalancerInstanceList, nil
}

// Only When any Subnet exists already(because of NetworkACLNo)
// Creat a Subnet for 'LB Only'('LB Type' Subnet)
func (nlbHandler *NcpVpcNLBHandler) CreatNcpSubnetForNlbOnly(vpcIID irs.IID, subnetReqInfo irs.SubnetInfo) (*vpc.Subnet, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreatNcpSubnetForNlb()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Zone, call.VPCSUBNET, subnetReqInfo.IId.NameId, "CreatNcpSubnetForNlb()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	vpcHandler := NcpVpcVPCHandler{
		RegionInfo: nlbHandler.RegionInfo,
		VPCClient:  nlbHandler.VPCClient,
	}

	// Check if the SubnetName Exists
	subnetInfoList, err := vpcHandler.ListSubnet(&vpcIID.SystemId)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		// return irs.VPCInfo{}, err    // Caution!!
	}

	for _, subnet := range subnetInfoList {
		if strings.EqualFold(subnet.IId.NameId, subnetReqInfo.IId.NameId) {
			newErr := fmt.Errorf("Subnet with the name [%s] exists already!!", subnetReqInfo.IId.NameId)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return nil, newErr
		}
	}

	// Get the Default NetworkACL No. of the VPC
	netAclNo, getNoErr := vpcHandler.GetDefaultNetworkAclNo(vpcIID)
	if getNoErr != nil {
		newErr := fmt.Errorf("Failed to Get Network ACL No of the VPC : [%v]", getNoErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	// Note : subnetUsageType : 'GEN' (general) | 'LOADB' (Load balancer only) | 'BM' (Bare metal only)
	subnetUsageType := "LOADB"
	ncpSubnetInfo, err := vpcHandler.CreateSubnet(vpcIID, netAclNo, &subnetUsageType, subnetReqInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create the Subnet for NLB Only : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	cblogger.Infof("New Subnet Name : [%s]", *ncpSubnetInfo.SubnetName)

	subnetStatus, err := vpcHandler.WaitForCreateSubnet(ncpSubnetInfo.SubnetNo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait for Creating the subnet : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	cblogger.Infof("\n### Subnet Status : [%s]", subnetStatus)

	return ncpSubnetInfo, nil
}


// Clean up the VMGroup and LB Type subnet
func (nlbHandler *NcpVpcNLBHandler) CleanUpNLB(vpcId *string) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CleanUpNLB()")
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", *vpcId, "CleanUpNLB()")

	if strings.EqualFold(*vpcId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	ncpNlbList, err := nlbHandler.GetNcpNlbListWithVpcId(vpcId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP NLB List with the VPC ID. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	nlbCount := len(ncpNlbList)
	cblogger.Infof("\n# NLB Count : [%d]", nlbCount)

	// Note : Cloud-Barista supports only this case => [ LB : Listener : VM Group : Health Checker = 1 : 1 : 1 : 1 ]
	ncpTargetGroupList, err := nlbHandler.GetNcpTargetGroupListWithVpcId(*vpcId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP VMGroup List with the VPC ID. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	if len(ncpTargetGroupList) < 1 {
		cblogger.Info("# VMGroup does Not Exist!!")
	} else {
		cblogger.Infof("\n\n# VMGroup No to Delete : [%s]", *ncpTargetGroupList[0].TargetGroupNo)
		delResult, err := nlbHandler.DeleteVMGroup(*ncpTargetGroupList[0].TargetGroupNo)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get NCP VMGroup List with the VPC ID. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return false, newErr
		}
		if !delResult {
			newErr := fmt.Errorf("Failed to Delete the VMGroup!!")
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return false, newErr
		}
	}

	// #### Note!!
	// If the deleted LB was the last one to delete, Need to delete the 'LB Type' Subnet.
	if nlbCount == 0 {
		lbTypeSubnetId, err := nlbHandler.GetSubnetIdForNlbOnly(*vpcId)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the SubnetId of LB Type subnet : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return false, newErr
		}
	
		vpcHandler := NcpVpcVPCHandler{
			RegionInfo: nlbHandler.RegionInfo,
			VPCClient:  nlbHandler.VPCClient,
		}
	
		result, removeErr := vpcHandler.RemoveSubnet(irs.IID{SystemId: *vpcId}, irs.IID{SystemId: lbTypeSubnetId})
		if removeErr != nil {
			newErr := fmt.Errorf("Failed to Remove the LB Type Subnet : [%v]", removeErr)
			cblogger.Error(newErr.Error())
			return false, newErr
		}
		if result {
			cblogger.Info("Succeeded in Removing the LB Type subnet.")
			return true, nil
		}
	}
	
	return true, nil
}

// NCP LB resource Def. : https://api.ncloud-docs.com/docs/en/common-vapidatatype-loadbalancerinstance
func (nlbHandler *NcpVpcNLBHandler) MappingNlbInfo(nlb vlb.LoadBalancerInstance) (irs.NLBInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called MappingNlbInfo()")

	if strings.EqualFold(*nlb.LoadBalancerInstanceNo, "") {
		newErr := fmt.Errorf("Invalid LoadBalancer Instance Info!!")
		cblogger.Error(newErr.Error())
		return irs.NLBInfo{}, newErr
	}

	// cblogger.Info("\n### NCP NlbInfo")
	// spew.Dump(nlb)

	var nlbType string
	if strings.EqualFold(*nlb.LoadBalancerNetworkType.CodeName, "PUBLIC") {
		nlbType = "PUBLIC"
	} else if strings.EqualFold(*nlb.LoadBalancerNetworkType.CodeName, "PRIVATE") {
		nlbType = "INTERNAL"
	}

	convertedTime, err := convertTimeFormat(*nlb.CreateDate)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert the Time Format!!")
		cblogger.Error(newErr.Error())
		return irs.NLBInfo{}, newErr
	}

	nlbInfo := irs.NLBInfo{
		IId: irs.IID{
			NameId:   *nlb.LoadBalancerName,
			SystemId: *nlb.LoadBalancerInstanceNo,
		},
		VpcIID: irs.IID{
			SystemId: *nlb.VpcNo,
		},
		Type:  nlbType,
		Scope: "REGION",
		CreatedTime: convertedTime,
	}

	keyValueList := []irs.KeyValue{
		{Key: "RegionCode", Value: *nlb.RegionCode},
		{Key: "NLB1stSubnetZone", Value: *nlb.LoadBalancerSubnetList[0].ZoneCode},
		{Key: "NLB_Status", Value: *nlb.LoadBalancerInstanceStatusName},
	}

	if len(nlb.LoadBalancerListenerNoList) > 0 {
		// Caution : If Get Listener info during Changing settings of a NLB., makes an Error.
		if strings.EqualFold(*nlb.LoadBalancerInstanceStatusName, "Changing") {
			cblogger.Info("### The NLB is being Changed Settings Now. Try again after finishing the changing processes.")
		} else {
			// Note : It is assumed that there is only one listener in the LB.
			listenerInfo, err := nlbHandler.GetListenerInfo(*nlb.LoadBalancerListenerNoList[0], *nlb.LoadBalancerInstanceNo)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the Listener Info : [%v]", err.Error())
				cblogger.Error(newErr.Error())
				return irs.NLBInfo{}, newErr
			}
			listenerInfo.IP = *nlb.LoadBalancerIpList[0] // Note : LoadBalancer IP is created already during LB creation.
			nlbInfo.Listener = *listenerInfo

			listenerKeyValue := irs.KeyValue{Key: "ListenerId", Value: *nlb.LoadBalancerListenerNoList[0]}
			keyValueList = append(keyValueList, listenerKeyValue)

			vmGroupInfo, err := nlbHandler.GetVMGroupInfo(nlb)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get VMGroup Info from the NLB. [%v]", err.Error())
				cblogger.Error(newErr.Error())
				// return irs.NLBInfo{}, newErr
			}
			nlbInfo.VMGroup = vmGroupInfo

			healthCheckerInfo, err := nlbHandler.GetHealthCheckerInfo(nlb)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get HealthChecker Info. frome the NLB. [%v]", err.Error())
				cblogger.Error(newErr.Error())
				return irs.NLBInfo{}, newErr
			}
			nlbInfo.HealthChecker = healthCheckerInfo

			monitorKeyValue := irs.KeyValue{Key: "HealthCheckerId", Value: nlbInfo.HealthChecker.CspID}
			keyValueList = append(keyValueList, monitorKeyValue)
		}
	}

	nlbInfo.KeyValueList = keyValueList
	return nlbInfo, nil
}

func (nlbHandler *NcpVpcNLBHandler) MappingListenerInfo(listener vlb.LoadBalancerListener) (irs.ListenerInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called MappingListenerInfo()")

	if strings.EqualFold(*listener.LoadBalancerListenerNo, "") {
		newErr := fmt.Errorf("Invalid LoadBalancer Listener Info!!")
		cblogger.Error(newErr.Error())
		return irs.ListenerInfo{}, newErr
	}

	if *listener.Port < 1 || *listener.Port > 65535 {
		newErr := fmt.Errorf("Invalid Listener Port.(Must be between 1 and 65535)")
		cblogger.Error(newErr.Error())
		return irs.ListenerInfo{}, newErr
	}

	listenerInfo := irs.ListenerInfo{
		Protocol: *listener.ProtocolType.Code,
		Port:     strconv.FormatInt(int64(*listener.Port), 10),
		CspID:    *listener.LoadBalancerListenerNo,
	}

	keyValueList := []irs.KeyValue{
		{Key: "UseHttp2", Value: strconv.FormatBool(*listener.UseHttp2)},
	}
	listenerInfo.KeyValueList = keyValueList

	return listenerInfo, nil
}

// Note!! : Will be decided later if we would support bellow methoeds or not.
// ------ Frontend Control
func (nlbHandler *NcpVpcNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {

	return irs.ListenerInfo{}, fmt.Errorf("Does not support yet!!")
}

// ------ Backend Control
func (nlbHandler *NcpVpcNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {

	return irs.VMGroupInfo{}, fmt.Errorf("Does not support yet!!")
}

func (nlbHandler *NcpVpcNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {

	return irs.HealthCheckerInfo{}, fmt.Errorf("Does not support yet!!")
}
