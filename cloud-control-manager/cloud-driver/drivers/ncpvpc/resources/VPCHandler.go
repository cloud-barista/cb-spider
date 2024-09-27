// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP VPC Handler
//
// by ETRI, 2020.10.
// by ETRI, 2022.03. updated

package resources

import (
	"fmt"
	"errors"
	"time"
	"strings"

	// "github.com/davecgh/go-spew/spew"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vpc"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVpcVPCHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VPCClient      *vpc.APIClient
}

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP VPCHandler")
}

func (vpcHandler *NcpVpcVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateVPC()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcReqInfo.IId.NameId, "CreateVPC()")

	if strings.EqualFold(vpcReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid VPC Name!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

	// Check if the VPC Exists
	vpcInfoList, err := vpcHandler.ListVPC()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		// return irs.VPCInfo{}, err   // Caution!!
	}

	for _, vpcInfo := range vpcInfoList {
		if strings.EqualFold(vpcInfo.IId.NameId, vpcReqInfo.IId.NameId) {
			newErr := fmt.Errorf("VPC with the name [%s] exists already!!", vpcReqInfo.IId.NameId)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VPCInfo{}, newErr
		}
	}

	// Create New VPC
	var vpcInfo irs.VPCInfo
	vpcReqName := strings.ToLower(vpcReqInfo.IId.NameId)

	createVpcReq := vpc.CreateVpcRequest {
		RegionCode: 	&vpcHandler.RegionInfo.Region,
		Ipv4CidrBlock:  &vpcReqInfo.IPv4_CIDR,
		VpcName: 		&vpcReqName, // Allows only lowercase letters, numbers or special character "-". Start with an alphabet character.
	}

	callLogStart := call.Start()
	vpcResult, err := vpcHandler.VPCClient.V2Api.CreateVpc(&createVpcReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create requested VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr		
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(vpcResult.VpcList) < 1 {
		newErr := fmt.Errorf("Failed to Create any VPC. Neww VPC does Not Exist!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr		
	} else {
		cblogger.Infof("Succeeded in Creating the VPC!! : [%s]", vpcReqInfo.IId.NameId)
	}

	// $$$ Uses Default NetworkACL of the Created VPC $$$
	/*
	// Create NetworkACL
	netAclReqName := vpcReqName  // vpcReqName : length of 30 from CB-Spider Server
	// Caution!! 
	// # Only lower case letter(No upper case).
	// # Length constraints: Minimum length of 3. Maximum length of 30.

	createNetAclReq := vpc.CreateNetworkAclRequest {
		RegionCode: 		&vpcHandler.RegionInfo.Region,
		NetworkAclName:     &netAclReqName, // Allows only lowercase letters, numbers or special character "-". Start with an alphabet character.
		VpcNo: 				vpcResult.VpcList[0].VpcNo,
	}

	netAclResult, err := vpcHandler.VPCClient.V2Api.CreateNetworkAcl(&createNetAclReq)
	if err != nil {
		cblogger.Errorf("Failed to Create the NetworkACL : [%v]", err)
		LoggingError(callLogInfo, err)
		return irs.VPCInfo{}, err
	}

	if *netAclResult.TotalRows < 1 {
		cblogger.Error("Failed to Create any NetworkACL!!")
		return irs.VPCInfo{}, errors.New("Failed to Create any NetworkACL!!")
	} else {
		cblogger.Infof("Succeeded in Creating the NetworkACL!! : [%s]", netAclReqName)
	}

	// Create Subnet
	for _, subnetInfo := range vpcReqInfo.SubnetInfoList {
		_, err := vpcHandler.CreateSubnet(vpcResult.VpcList[0].VpcNo, netAclResult.NetworkAclList[0].NetworkAclNo, subnetInfo)
		if err != nil {
			newErr := fmt.Errorf("Failed to Create New Subnet : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VPCInfo{}, newErr
		}
	}
	*/
	
	newVpcIID := irs.IID{SystemId: *vpcResult.VpcList[0].VpcNo}

	cblogger.Infof("# Waitting while Creating New VPC and Default NetworkACL!!")
	vpcStatus, err := vpcHandler.WaitForCreateVPC(newVpcIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait for VPC Creation : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	cblogger.Infof("===> # Status of New VPC [%s] : [%s]", newVpcIID.SystemId, vpcStatus)

	// Add Requested Subnets
	for _, subnetReqInfo := range vpcReqInfo.SubnetInfoList {
		_, err := vpcHandler.AddSubnet(newVpcIID, subnetReqInfo) // Waitting time Included
		if err != nil {
			newErr := fmt.Errorf("Failed to Create New Subnet : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VPCInfo{}, newErr
		}
	}

	vpcInfo, getErr := vpcHandler.GetVPC(newVpcIID)
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Get VPC Info : [%v]", getErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	return vpcInfo, nil
}

func (vpcHandler *NcpVpcVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger.Info("NCP VPC cloud driver: called ListVPC()!!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, "ListVPC()", "ListVPC()")

	vpcListReq := vpc.GetVpcListRequest {
		RegionCode: 	&vpcHandler.RegionInfo.Region,
	}

	// cblogger.Infof("vpcListReq Ready!!")
	// spew.Dump(vpcListReq)
	callLogStart := call.Start()
	result, err := vpcHandler.VPCClient.V2Api.GetVpcList(&vpcListReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VPC List from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var vpcInfoList []*irs.VPCInfo
	if len(result.VpcList) < 1 {
		cblogger.Info("### VPC does Not Exist!!")
	} else {
		for _, vpc := range result.VpcList {
			vpcInfo, err := vpcHandler.MappingVpcInfo(vpc)
			if err != nil {
				newErr := fmt.Errorf("Failed to Map the VPC Info : [%v]", err)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return nil, newErr
			}
			vpcInfoList = append(vpcInfoList, vpcInfo)
		}
	}
	return vpcInfoList, nil
}

func (vpcHandler *NcpVpcVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetVPC()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcIID.SystemId, "GetVPC()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

	// Get VPC Info from NCP VPC
	ncpVpcInfo, err := vpcHandler.GetNcpVpcInfo(&vpcIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP VPC Info with the SystemId : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

	vpcInfo, err := vpcHandler.MappingVpcInfo(ncpVpcInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Map the VPC Info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	return *vpcInfo, nil
}

func (vpcHandler *NcpVpcVPCHandler) GetNcpVpcInfo(vpcId *string) (*vpc.Vpc, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetNcpVpcInfo()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, *vpcId, "GetNcpVpcInfo()")

	if strings.EqualFold(*vpcId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	vpcInfoReq := vpc.GetVpcDetailRequest {
		RegionCode: &vpcHandler.RegionInfo.Region,
		VpcNo: 		vpcId,
	}

	callLogStart := call.Start()
	result, err := vpcHandler.VPCClient.V2Api.GetVpcDetail(&vpcInfoReq)
	if err != nil {
		cblogger.Errorf("Failed to Find the VPC Info from NCP VPC : [%v]", err)
		LoggingError(callLogInfo, err)
		return nil, err
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.VpcList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any VPC Info with the ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	} else {
		cblogger.Infof("Succeeded in Getting the VPC Info from NCP VPC!!")
	}
	return result.VpcList[0], nil
}

func (vpcHandler *NcpVpcVPCHandler) GetSubnet(sunbnetIID irs.IID) (irs.SubnetInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetSubnet()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, sunbnetIID.SystemId, "GetSubnet()")

	if strings.EqualFold(sunbnetIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Subnet SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SubnetInfo{}, newErr
	}

	// Get Subnet Info from NCP VPC
	ncpSubnetInfo, err := vpcHandler.GetNcpSubnetInfo(&sunbnetIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Subnet Info with the SystemId : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SubnetInfo{}, newErr
	}	

	subnetInfo := vpcHandler.MappingSubnetInfo(ncpSubnetInfo)
	return *subnetInfo, nil
}

func (vpcHandler *NcpVpcVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called DeleteVPC()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcIID.SystemId, "DeleteVPC()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	// Check if the VPC exists
	vpcInfo, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find any VPC info. with the SystemId : [%s] : [%v]", vpcIID.SystemId, err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	cblogger.Infof("VPC NameId to Delete [%s]", vpcInfo.IId.NameId)
	
	// Get SubnetList to Delete
	subnetInfoList, err := vpcHandler.ListSubnet(&vpcIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get SubnetList with the SystemId : [%s] : [%v]", vpcIID.SystemId, err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	var lastSubnetNo string

	// Remove All Subnets belonging to the VPC
	for _, subnet := range subnetInfoList {
		// Remove the Subnet
		delReq := vpc.DeleteSubnetRequest {
			RegionCode: 	&vpcHandler.RegionInfo.Region,
			SubnetNo: 		&subnet.IId.SystemId,
		}

		callLogStart := call.Start()
		delResult, err := vpcHandler.VPCClient.V2Api.DeleteSubnet(&delReq)
		if err != nil {
			newErr := fmt.Errorf("Failed to Remove the Subnet with the SystemId : [%s] : [%v]", subnet.IId.SystemId, err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return false, newErr
		}
		LoggingInfo(callLogInfo, callLogStart)

		cblogger.Infof("Removed Subnet Name : [%s]", *delResult.SubnetList[0].SubnetName)

		lastSubnetNo = subnet.IId.SystemId
	}

	lastSubentIID := irs.IID{SystemId: lastSubnetNo}

	cblogger.Infof("# Waitting while Deleting All Subnets belonging to the VPC!!")
	subnetStatus, err := vpcHandler.WaitForDeleteSubnet(vpcIID.SystemId, lastSubentIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait for Subnet Deletion : [%v]", err)
		cblogger.Debug(newErr.Error()) // For Termination Completion of a Subnet
		LoggingError(callLogInfo, newErr)
		// return false, newErr
	}

	cblogger.Infof("===> # Status of Subnet [%s] : [%s]", lastSubentIID.SystemId, subnetStatus)

	// Delete the VPC
	delReq := vpc.DeleteVpcRequest {
		RegionCode: 	&vpcHandler.RegionInfo.Region,
		VpcNo: 			&vpcIID.SystemId,
	}

	callLogStart := call.Start()
	delResult, err := vpcHandler.VPCClient.V2Api.DeleteVpc(&delReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the VPC with the SystemId : [%s] : [%v]", vpcIID.SystemId, err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	cblogger.Infof("Succeeded in Deleting the VPC!! : [%s]", *delResult.VpcList[0].VpcName)
	return true, nil
}

// Only When any Subnet exists already(because of NetworkACLNo)
func (vpcHandler *NcpVpcVPCHandler) AddSubnet(vpcIID irs.IID, subnetReqInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called AddSubnet()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, subnetReqInfo.IId.NameId, "AddSubnet()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

	// Check if the VPC exists
	vpcInfo, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find any VPC info. with the SystemId : [%s]", vpcIID.SystemId)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

	cblogger.Infof("VPC NameId to Add the Subnet : [%s]", vpcInfo.IId.NameId)

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
			return irs.VPCInfo{}, newErr
		}
	}

	// Get the Default NetworkACL No. of the VPC
	netAclNo, getNoErr := vpcHandler.GetDefaultNetworkAclNo(vpcIID)
	if getNoErr != nil {
		newErr := fmt.Errorf("Failed to Get Network ACL No of the VPC : [%v]", getNoErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

	// Note : subnetUsageType : 'GEN' (general) | 'LOADB' (Load balancer only) | 'BM' (Bare metal only)
	subnetUsageType := "GEN"
	ncpSubnetInfo, err := vpcHandler.CreateSubnet(vpcIID, netAclNo, &subnetUsageType, subnetReqInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create the Subnet for General Purpose : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	cblogger.Infof("New Subnet SubnetNo : [%s]", *ncpSubnetInfo.SubnetNo)

	subnetStatus, err := vpcHandler.WaitForCreateSubnet(ncpSubnetInfo.SubnetNo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait for Creating the subnet : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	cblogger.Infof("# Subnet Status : [%s]", subnetStatus)

	vpcInfo, getErr := vpcHandler.GetVPC(irs.IID{SystemId: vpcIID.SystemId})
	if getErr != nil {
		cblogger.Error(getErr.Error())
		LoggingError(callLogInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	return vpcInfo, nil
}

func (vpcHandler *NcpVpcVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called RemoveSubnet()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, subnetIID.SystemId, "RemoveSubnet()")

	if strings.EqualFold(subnetIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Subnet SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	// Check if the Subnet Exists
	subnetInfoList, err := vpcHandler.ListSubnet(&vpcIID.SystemId)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		return false, err
	}

	var subnetName string

	for _, subnetInfo := range subnetInfoList {
		if strings.EqualFold(subnetInfo.IId.SystemId, subnetIID.SystemId) {
			subnetName = subnetInfo.IId.NameId
			break
		}		
	}

	if strings.EqualFold(subnetName, "") {
		return false, fmt.Errorf("Failed to Find the Subnet!! : [%s]", subnetIID.SystemId)
	}

	// Remove the Subnet
	delReq := vpc.DeleteSubnetRequest {
		RegionCode: 	&vpcHandler.RegionInfo.Region,
		SubnetNo: 		&subnetIID.SystemId,
	}

	callLogStart := call.Start()
	delResult, err := vpcHandler.VPCClient.V2Api.DeleteSubnet(&delReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Remove the Requested Subnet : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)
	cblogger.Infof("Removed Subnet Name : [%s]", *delResult.SubnetList[0].SubnetName)
	return true, nil
}

func (vpcHandler *NcpVpcVPCHandler) CreateSubnet(vpcIID irs.IID, netAclNo *string, subnetUsageType *string, subnetReqInfo irs.SubnetInfo) (*vpc.Subnet, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateSubnet()!")
	
	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, subnetReqInfo.IId.NameId, "CreateSubnet()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	if strings.EqualFold(subnetReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid Requested Subnet NameId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	// vpcInfo, err := vpcHandler.GetVPC(vpcIID) 	// Don't Need to Check if the VPC exists

	subnetReqName := strings.ToLower(subnetReqInfo.IId.NameId)
	subnetTypeCode := "PUBLIC" // 'PUBLIC' (for Internet Gateway) | 'PRIVATE'
	// subnetUsageType : 'GEN' (general) | 'LOADB' (Load balancer only) | 'BM' (Bare metal only)
	createSubnetReq := vpc.CreateSubnetRequest {
		RegionCode: 	&vpcHandler.RegionInfo.Region,
		SubnetTypeCode: &subnetTypeCode,
		UsageTypeCode: 	subnetUsageType,	
		NetworkAclNo: 	netAclNo,
		Subnet:			&subnetReqInfo.IPv4_CIDR,
		SubnetName:     &subnetReqName, // Allows only lowercase letters, numbers or special character "-". Start with an alphabet character.
		VpcNo: 			&vpcIID.SystemId,
		ZoneCode: 		&subnetReqInfo.Zone,
	}

	callLogStart := call.Start()
	subnet, err := vpcHandler.VPCClient.V2Api.CreateSubnet(&createSubnetReq)
	if err != nil {
		cblogger.Errorf("Failed to Create the Subnet : [%v]", err)
		LoggingError(callLogInfo, err)
		return nil, err
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(subnet.SubnetList) < 1 {
		newErr := fmt.Errorf("Failed to Create the Subnet. New Subnet does Not Exist!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	} else {
		cblogger.Infof("Succeeded in Creating the Subnet!! : [%s]", *subnet.SubnetList[0].SubnetName)
	}
	return subnet.SubnetList[0], nil
}

func (vpcHandler *NcpVpcVPCHandler) GetNcpSubnetInfo(sunbnetId *string) (*vpc.Subnet, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetNcpSubnetInfo()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, *sunbnetId, "GetNcpSubnetInfo()")

	if strings.EqualFold(*sunbnetId, "") {
		newErr := fmt.Errorf("Invalid Subnet ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	subnetInfoReq := vpc.GetSubnetDetailRequest {
		RegionCode: &vpcHandler.RegionInfo.Region,
		SubnetNo: 	sunbnetId,
	}

	callLogStart := call.Start()
	result, err := vpcHandler.VPCClient.V2Api.GetSubnetDetail(&subnetInfoReq)
	if err != nil {
		cblogger.Errorf("Failed to Get the Subnet Info from NCP VPC : [%v]", err)
		LoggingError(callLogInfo, err)
		return nil, err
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.SubnetList) < 1 {
		newErr := fmt.Errorf("Failed to Get any Subnet Info with the ID!!")
		cblogger.Debug(newErr.Error()) // For Termination Completion of a Subnet
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	} else {
		cblogger.Infof("Succeeded in Getting the Subnet Info from NCP VPC!!")
	}
	return result.SubnetList[0], nil
}

func (vpcHandler *NcpVpcVPCHandler) ListSubnet(vpcNo *string) ([]*irs.SubnetInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called ListSubnet()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, "ListSubnet()", "ListSubnet()")

	subnetListReq := vpc.GetSubnetListRequest {
		RegionCode: &vpcHandler.RegionInfo.Region,
		VpcNo: 		vpcNo,
	}

	cblogger.Infof("subnetListReq Ready!!")
	// spew.Dump(subnetListReq)

	result, err := vpcHandler.VPCClient.V2Api.GetSubnetList(&subnetListReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get subnetList : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	var subnetInfoList []*irs.SubnetInfo
	if len(result.SubnetList) < 1 {
		cblogger.Infof("### The VPC has No Subnet!!")
	} else {
		cblogger.Infof("Succeeded in Getting SubnetList!! : ")		
		for _, subnet := range result.SubnetList { // To Get Subnet info list
			subnetInfo := vpcHandler.MappingSubnetInfo(subnet)
			subnetInfoList = append(subnetInfoList, subnetInfo)
		}
	}
	return subnetInfoList, nil
}

func (vpcHandler *NcpVpcVPCHandler) GetDefaultNetworkAclNo(vpcIID irs.IID) (*string, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetDefaultNetworkAclNo()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcIID.SystemId, "GetDefaultNetworkAclNo()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	// # Caution!! : Infinite Loop
	// vpcInfo, err := vpcHandler.GetVPC(vpcIID)  	// Check if the VPC exists

	// Get the Default NetworkACL of the VPC
	getReq := vpc.GetNetworkAclListRequest {
		RegionCode: 	&vpcHandler.RegionInfo.Region,
		VpcNo: 			&vpcIID.SystemId,
	}

	callLogStart := call.Start()
	netAclResult, err := vpcHandler.VPCClient.V2Api.GetNetworkAclList(&getReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NetworkACL List!! : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var netACLNo *string
	if len(netAclResult.NetworkAclList) < 1 {
		cblogger.Info("# NetworkACL does Not Exist!!")
	} else {
		for _, netACL := range netAclResult.NetworkAclList {
			if strings.Contains(*netACL.NetworkAclName, "default-network-acl") {  // When Contains "default-network-acl" in NetworkAclName
				netACLNo = netACL.NetworkAclNo
				break
			}
		}
	}

	if strings.EqualFold(*netACLNo, "") {
		newErr := fmt.Errorf("Failed to Get the Default NetworkACL No of the VPC!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	return netACLNo, nil
}

func (vpcHandler *NcpVpcVPCHandler) MappingVpcInfo(vpc *vpc.Vpc) (*irs.VPCInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called MappingVpcInfo()!")

	// VPC info mapping
	vpcInfo := irs.VPCInfo{
		IId: irs.IID{
			NameId:   *vpc.VpcName,
			SystemId: *vpc.VpcNo,
		},
		IPv4_CIDR:  *vpc.Ipv4CidrBlock,
	}

	keyValueList := []irs.KeyValue{
		{Key: "RegionCode", Value: *vpc.RegionCode},
		{Key: "VpcStatus", Value: *vpc.VpcStatus.Code},
		{Key: "CreateDate", Value: *vpc.CreateDate},
	}
	vpcInfo.KeyValueList = keyValueList

	subnetList, err := vpcHandler.ListSubnet(vpc.VpcNo)
	if err != nil {
		cblogger.Errorf("Failed to Get subnet List : [%v]", err) // Caution!!
		// return nil   // Caution!!
	}

	var subnetInfoList []irs.SubnetInfo

	if len(subnetList) > 0 {
		for _, subnetInfo := range subnetList {    // Ref) var subnetList []*irs.SubnetInfo
			cblogger.Infof("# Subnet NameId : [%s]", subnetInfo.IId.NameId)
			subnetInfoList = append(subnetInfoList, *subnetInfo)
		}	
		vpcInfo.SubnetInfoList = subnetInfoList
	}

	// cblogger.Infof("vpcInfo.SubnetInfoList : ")
	// spew.Dump(vpcInfo.SubnetInfoList)

	/*
	// Get the Default NetworkACL of the VPC
	netAclNo, getNoErr := vpcHandler.GetDefaultNetworkAclNo(irs.IID{SystemId: *vpc.VpcNo})
	if getNoErr != nil {
		newErr := fmt.Errorf("Failed to Get Network ACL No of the VPC : [%v]", getNoErr)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	keyValue := irs.KeyValue{Key: "DefaultNetworkACLNo", Value: *netAclNo}
	vpcInfo.KeyValueList = append(vpcInfo.KeyValueList, keyValue)
	*/

	return &vpcInfo, nil
}

func (vpcHandler *NcpVpcVPCHandler) MappingSubnetInfo(subnet *vpc.Subnet) *irs.SubnetInfo {
	cblogger.Info("NCP VPC Cloud Driver: called MappingSubnetInfo()!")
	// spew.Dump(*subnet)

	// Subnet info mapping
	subnetInfo := irs.SubnetInfo {
		IId: irs.IID{
			NameId:   *subnet.SubnetName,
			SystemId: *subnet.SubnetNo,
		},
		Zone: 		  *subnet.ZoneCode,
		IPv4_CIDR:    *subnet.Subnet,
	}

	keyValueList := []irs.KeyValue{
		// {Key: "ZoneCode", Value: *subnet.ZoneCode},
		{Key: "SubnetStatus", Value: *subnet.SubnetStatus.Code},
		{Key: "SubnetType", Value: *subnet.SubnetType.Code},	
		{Key: "UsageType", Value: *subnet.UsageType.Code},
		{Key: "NetworkACLNo", Value: *subnet.NetworkAclNo},
		{Key: "CreateDate", Value: *subnet.CreateDate},
	}
	subnetInfo.KeyValueList = keyValueList
	return &subnetInfo
}

// Waiting for up to 600 seconds until VPC and Network ACL Creation processes are Finished.
func (vpcHandler *NcpVpcVPCHandler) WaitForCreateVPC(vpcIID irs.IID) (string, error) {
	cblogger.Info("======> As Subnet cannot be Created Immediately after VPC Creation Call, it waits until VPC and Network ACL Creation processes are Finished.")

	curRetryCnt := 0
	maxRetryCnt := 600

	for {
		ncpVpcInfo, getErr := vpcHandler.GetNcpVpcInfo(&vpcIID.SystemId)
		if getErr != nil {
			newErr := fmt.Errorf("Failed to Get VPC Info : [%v]", getErr)
			cblogger.Error(newErr.Error())
			return "", newErr
		} else {
			cblogger.Infof("Succeeded in Getting the VPC Info of [%s]", vpcIID.SystemId)
		}

		vpcStatus := *ncpVpcInfo.VpcStatus.Code
		cblogger.Infof("\n### VPC Status [%s] : ", vpcStatus)

		if strings.EqualFold(vpcStatus, "CREATING") {
			curRetryCnt++
			cblogger.Infof("The VPC and Network ACL are still [%s], so wait for a second more before Creating Subnet.", vpcStatus)
			time.Sleep(time.Second * 3)
			if curRetryCnt > maxRetryCnt {
				newErr := fmt.Errorf("Despite waiting for a long time(%d sec), the VPC status is '%s', so it is forcibly finishied.", maxRetryCnt, vpcStatus)
				return "", newErr
			}
		} else {
			cblogger.Infof("The VPC and Network ACL Creation processes are Finished.")

			// Wait More
			time.Sleep(time.Second * 5)
			return vpcStatus, nil
		}
	}
}

// Waiting for up to 600 seconds until VPC and Network ACL Creation processes are Finished.
func (vpcHandler *NcpVpcVPCHandler) WaitForDeleteSubnet(vpcNo string, subnetIID irs.IID) (string, error) {
	cblogger.Info("======> As VPC cannot be Deleted Immediately after Subnet Deletion Call, it waits until Subnet Deletion processes are Finished.")

	curRetryCnt := 0
	maxRetryCnt := 600

	subnetList, err := vpcHandler.ListSubnet(&vpcNo)
	if err != nil {
		cblogger.Errorf("Failed to Get subnet List : [%v]", err) // Caution!!
		// return nil   // Caution!!
	}

	if len(subnetList) > 0 {
		for {
			ncpSubnetInfo, getErr := vpcHandler.GetNcpSubnetInfo(&subnetIID.SystemId)
			if getErr != nil {
				newErr := fmt.Errorf("Failed to Get the Subnet Info : [%v]", getErr)
				cblogger.Debug(newErr.Error()) // For Termination Completion of a Subnet
				return "", newErr
			} else {
				cblogger.Infof("Succeeded in Getting the Subnet Info of [%s]", subnetIID.SystemId)
			}
	
			subnetStatus := *ncpSubnetInfo.SubnetStatus.Code
			cblogger.Infof("\n### Subnet Status [%s] : ", subnetStatus)

			if strings.EqualFold(subnetStatus, "TERMTING") || strings.EqualFold(subnetStatus, "CREATING"){
				curRetryCnt++
				cblogger.Infof("The Suntnet is still [%s], so wait for a second more before Deleting VPC.", subnetStatus)
				time.Sleep(time.Second * 3)
				if curRetryCnt > maxRetryCnt {
					newErr := fmt.Errorf("Despite waiting for a long time(%d sec), the Subnet status is '%s', so it is forcibly finishied.", maxRetryCnt, subnetStatus)
					return "", newErr
				}
			} else {
				cblogger.Infof("The Subnet Deletion processes are Finished.")

				// Wait More
				time.Sleep(time.Second * 5)

				return subnetStatus, nil
			}
		}
	} else {
		return "This VPC has No Subnet!!", nil
	}
}

// Waiting for up to 600 seconds until Subnet Creation processes are Finished.
func (vpcHandler *NcpVpcVPCHandler) WaitForCreateSubnet(subnetId *string) (string, error) {
	cblogger.Info("======> As Subnet cannot be Created Immediately after VPC Creation Call, it waits until VPC and Network ACL Creation processes are Finished.")

	curRetryCnt := 0
	maxRetryCnt := 600

	for {
		ncpSubnetInfo, getErr := vpcHandler.GetNcpSubnetInfo(subnetId)
		if getErr != nil {
			newErr := fmt.Errorf("Failed to Get the Subnet Info : [%v]", getErr)
			cblogger.Error(newErr.Error())
			return "", newErr
		} else {
			cblogger.Infof("Succeeded in Getting the Subnet Info of [%s]", *subnetId)
		}

		subnetStatus := *ncpSubnetInfo.SubnetStatus.Code
		cblogger.Infof("\n### Subnet Status [%s] : ", subnetStatus)

		if strings.EqualFold(subnetStatus, "CREATING") {
			curRetryCnt++
			cblogger.Infof("The Subnet is still [%s], so wait for a second more before Getting VPC Info.", subnetStatus)
			time.Sleep(time.Second * 3)
			if curRetryCnt > maxRetryCnt {
				newErr := fmt.Errorf("Despite waiting for a long time(%d sec), the Subnet status is '%s', so it is forcibly finishied.", maxRetryCnt, subnetStatus)
				return "", newErr
			}
		} else {
			cblogger.Infof("The Subnet Creation processes are Finished.")

			// Wait More
			time.Sleep(time.Second * 3)
			return subnetStatus, nil
		}
	}
}

func (vpcHandler *NcpVpcVPCHandler) getSubnetZone(vpcIID irs.IID, subnetIID irs.IID) (string, error) {
	cblogger.Info("NCP VPC cloud driver: called getSubnetZone()!!")

	if strings.EqualFold(vpcIID.SystemId, "") && strings.EqualFold(vpcIID.NameId, ""){
		newErr := fmt.Errorf("Invalid VPC Id!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	if strings.EqualFold(subnetIID.SystemId, "") && strings.EqualFold(subnetIID.NameId, ""){
		newErr := fmt.Errorf("Invalid Subnet Id!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	 // Get the VPC information
	 vpcInfo, err := vpcHandler.GetVPC(vpcIID)
	 if err != nil {
		newErr := fmt.Errorf("Failed to Get the VPC Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	 }
	//  cblogger.Info("\n\n### vpcInfo : ")
	//  spew.Dump(vpcInfo)
	//  cblogger.Info("\n")
 	 
	// Get the Zone info of the specified Subnet
	var subnetZone string
	for _, subnet := range vpcInfo.SubnetInfoList {
		if strings.EqualFold(subnet.IId.SystemId, subnetIID.SystemId) {
			subnetZone = subnet.Zone
			break
		}
	}
	if strings.EqualFold(subnetZone, "") {
		newErr := fmt.Errorf("Failed to Get the Zone info of the specified Subnet!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	return subnetZone, nil
}


func (vpcHandler *NcpVpcVPCHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}
