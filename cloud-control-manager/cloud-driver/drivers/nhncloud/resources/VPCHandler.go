// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, Innogrid, 2021.12.
// by ETRI 2022.03. updated
// by ETRI 2023.11. updated
// by ETRI 2024.02. updated (New REST API Applied)

package resources

import (
	"strings"
	"fmt"
	// "sync"
	"time"
	// "github.com/davecgh/go-spew/spew"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/extensions/external"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/vpcs"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/subnets"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NhnCloudVPCHandler struct {
	CredentialInfo  idrv.CredentialInfo
	RegionInfo 		idrv.RegionInfo
	NetworkClient   *nhnsdk.ServiceClient
}

type NetworkWithExt struct {
	vpcs.VPC
	external.NetworkExternalExt
}

func (vpcHandler *NhnCloudVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger.Info("NHN Cloud Cloud Driver: called CreateVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcReqInfo.IId.NameId, "CreateVPC()")

	if strings.EqualFold(vpcReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid VPC NameId!!")
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	if strings.EqualFold(vpcReqInfo.IPv4_CIDR, "") {
		newErr := fmt.Errorf("Invalid IPv4_CIDR!!")
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}
	
	if strings.EqualFold(vpcHandler.CredentialInfo.TenantId, "") {
		newErr := fmt.Errorf("Invalid Tenant ID!!")
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	createOpts := vpcs.CreateOpts{
		Name:       vpcReqInfo.IId.NameId,
		CIDRv4:		vpcReqInfo.IPv4_CIDR,
		TenantID: 	vpcHandler.CredentialInfo.TenantId, // Need to Specify!!
	}
	start := call.Start()
	vpcResult, err := vpcs.Create(vpcHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New VPC. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	// Because there are functions that use 'NameId', Input NameId too
	newVpcIID := irs.IID{NameId: vpcResult.Name, SystemId: vpcResult.ID}

	// Wait for New VPC info to be inquired
	curStatus, err := vpcHandler.waitForVpcCreation(newVpcIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait to Get VPC Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	cblogger.Infof("==> # Status of VPC [%s] : [%s]", newVpcIID.NameId, curStatus)

	vpcInfo, err := vpcHandler.GetVPC(irs.IID{SystemId: vpcResult.ID})
	if err != nil {
		cblogger.Errorf("Failed to Find any VPC Info with the SystemId. : [%s], %v", vpcResult.ID, err)
		LoggingError(callLogInfo, err)
		return irs.VPCInfo{}, err
	} else {
		vpcInfo.IId.NameId = vpcReqInfo.IId.NameId  // Caution!! For IID2 NameID validation check for VPC
	}

	// Create Subnet
	var subnetList []irs.SubnetInfo
	for _, subnet := range vpcReqInfo.SubnetInfoList {
		cblogger.Infof("# Subnet NameId to Create : [%s]", subnet.IId.NameId)
		newSubnet, err := vpcHandler.createSubnet(vpcResult.ID, subnet) // Caution!! For IID2 NameID validation check for Subnet
		if err != nil {
			cblogger.Errorf("Failed to Create NHN Cloud Sunbnet : [%v]", err)
			LoggingError(callLogInfo, err)
			return irs.VPCInfo{}, err
		}
		subnetList = append(subnetList, newSubnet)
	}

	vpcInfo.SubnetInfoList = subnetList
	return vpcInfo, err
}

func (vpcHandler *NhnCloudVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	cblogger.Info("NHN Cloud Cloud Driver: called GetVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcIID.SystemId, "GetVPC()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

	start := call.Start()
	var vpc vpcs.VPC
	err := vpcs.Get(vpcHandler.NetworkClient, vpcIID.SystemId).ExtractInto(&vpc)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VPC with the SystemId : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	vpcInfo, err := vpcHandler.mappingVpcInfo(vpc)
	if err != nil {
		newErr := fmt.Errorf("Failed to Map the VPC Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}
	return *vpcInfo, nil
}

func (vpcHandler *NhnCloudVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger.Info("NHN Cloud Cloud Driver: called ListVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, "ListVPC()", "ListVPC()")

	if strings.EqualFold(vpcHandler.CredentialInfo.TenantId, "") {
		newErr := fmt.Errorf("Invalid Tenant ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	start := call.Start()
	listOpts := vpcs.ListOpts{
		TenantID: vpcHandler.CredentialInfo.TenantId,
		RouterExternal: false,
	}
	allPages, err := vpcs.List(vpcHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		cblogger.Errorf("Failed to Get VPC Pages. %v", err)
		LoggingError(callLogInfo, err)
		return nil, err
	}
	LoggingInfo(callLogInfo, start)

	vpcList, err := vpcs.ExtractVPCs(allPages)
	if err != nil {
		cblogger.Errorf("Failed to Get VPC list from NHN Cloud. %v", err)
		LoggingError(callLogInfo, err)
		return nil, err
	}

	var vpcInfoList []*irs.VPCInfo
	if len(vpcList) > 0 {
		for _, vpc := range vpcList {
			vpcInfo, err := vpcHandler.mappingVpcSubnetInfo(vpc) // Caution!!
			if err != nil {
				newErr := fmt.Errorf("Failed to Map the VPC Info : [%v]", err)
				cblogger.Error(newErr.Error())
				return nil, newErr
			}
			vpcInfoList = append(vpcInfoList, vpcInfo)
		}
	}
	return vpcInfoList, nil
}

func (vpcHandler *NhnCloudVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Cloud Driver: called DeleteVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcIID.SystemId, "DeleteVPC()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	err := vpcs.Delete(vpcHandler.NetworkClient, vpcIID.SystemId).ExtractErr()
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the VPC with the SystemID. : [%s] : [%v]", vpcIID.SystemId, err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, err)
		return false, newErr
	} else {
		cblogger.Infof("Succeeded in Deleting the VPC : " + vpcIID.SystemId)
	}

	return true, nil
}

func (vpcHandler *NhnCloudVPCHandler) createSubnet(vpcId string, subnetReqInfo irs.SubnetInfo) (irs.SubnetInfo, error) {
	cblogger.Info("NHN Cloud cloud driver: called createSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, subnetReqInfo.IId.NameId, "createSubnet()")

	if strings.EqualFold(vpcId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SubnetInfo{}, newErr
	}

	if strings.EqualFold(subnetReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid Subnet NameId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SubnetInfo{}, newErr
	}

	createOpts := subnets.CreateOpts{
		VpcID: 	vpcId,
		CIDR: 	subnetReqInfo.IPv4_CIDR,
		Name: 	subnetReqInfo.IId.NameId,
	}
	start := call.Start()
	subnet, err := subnets.Create(vpcHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Createt New Subnet!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.SubnetInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	subnetInfo := vpcHandler.mappingSubnetInfo(*subnet)
	return *subnetInfo, nil
}

func (vpcHandler *NhnCloudVPCHandler) getSubnet(subnetIId irs.IID) (irs.SubnetInfo, error) {
	cblogger.Info("NHN Cloud cloud driver: called getSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, subnetIId.SystemId, "getSubnet()")

	if strings.EqualFold(subnetIId.SystemId, "") {
		newErr := fmt.Errorf("Invalid Subnet SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SubnetInfo{}, newErr
	}

	subnet, err := subnets.Get(vpcHandler.NetworkClient, subnetIId.SystemId).Extract()
	if err != nil {
		cblogger.Errorf("Failed to Get Subnet with SystemId [%s] : %v", subnetIId.SystemId, err)
		LoggingError(callLogInfo, err)
		return irs.SubnetInfo{}, nil
	}
	subnetInfo := vpcHandler.mappingSubnetInfo(*subnet)
	return *subnetInfo, nil
}

func (vpcHandler *NhnCloudVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	cblogger.Info("NHN Cloud cloud driver: called AddSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcIID.SystemId, "AddSubnet()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC System ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

	if strings.EqualFold(subnetInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid Subnet NameId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

	createOpts := subnets.CreateOpts{
		VpcID: 	vpcIID.SystemId,
		CIDR: 	subnetInfo.IPv4_CIDR,
		Name: 	subnetInfo.IId.NameId,
	}
	start := call.Start()
	_, err := subnets.Create(vpcHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Createt New Subnet!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)


	vpcInfo, err := vpcHandler.GetVPC(irs.IID{SystemId: vpcIID.SystemId})
	if err != nil {
		cblogger.Errorf("Failed to Get the VPC Info with the SystemId. : [%s], %v", vpcIID.SystemId, err)
		LoggingError(callLogInfo, err)
		return irs.VPCInfo{}, err
	}
	return vpcInfo, nil
}

func (vpcHandler *NhnCloudVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud cloud driver: called RemoveSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, subnetIID.SystemId, "RemoveSubnet()")

	if strings.EqualFold(subnetIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Subnet SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	err := subnets.Delete(vpcHandler.NetworkClient, subnetIID.SystemId).ExtractErr()
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Subnet with the SystemID. : [%s] : [%v]", subnetIID.SystemId, err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, err)
		return false, newErr
	} else {
		cblogger.Infof("Succeeded in Deleting the Subnet : " + subnetIID.SystemId)
	}

	return true, nil
}

func (vpcHandler *NhnCloudVPCHandler) mappingVpcInfo(vpc vpcs.VPC) (*irs.VPCInfo, error) {
	cblogger.Info("NHN Cloud cloud driver: called mappingVpcInfo()!!")
	// cblogger.Info("\n\n### vpc : ")
	// spew.Dump(vpc)
	// cblogger.Info("\n")

	vpcInfo := irs.VPCInfo {
		IId: irs.IID{
			NameId:   vpc.Name,
			SystemId: vpc.ID,
		},
	}
	vpcInfo.IPv4_CIDR = vpc.CIDRv4
	
	// Get Subnet info list.
	var subnetInfoList []irs.SubnetInfo
	if len(vpc.Subnets) > 0 {
		for _, subnet := range vpc.Subnets {  // Because of vpcs.Subnet type (Not subnets.Subnet type), need to getSubnet()
			subnetInfo, err := vpcHandler.getSubnet(irs.IID{SystemId: subnet.ID})
			if err != nil {
				newErr := fmt.Errorf("Failed to Get Subnet info with the subnetId [%s]. [%v]", subnet.ID, err)
				cblogger.Error(newErr.Error())
				return nil, newErr
			}
			subnetInfoList = append(subnetInfoList, subnetInfo)
		}
	}	
	vpcInfo.SubnetInfoList = subnetInfoList

	var RouterExternal string
	if vpc.RouterExternal {
		RouterExternal = "Yes"
	} else if !vpc.RouterExternal {
		RouterExternal = "No"
	}

	keyValueList := []irs.KeyValue{
		{Key: "Status", Value: vpc.State},
		{Key: "RouterExternal", Value: RouterExternal},
		{Key: "CreatedTime", Value: vpc.CreateTime},
	}
	vpcInfo.KeyValueList = keyValueList

	return &vpcInfo, nil
}

func (vpcHandler *NhnCloudVPCHandler) mappingVpcSubnetInfo(vpc vpcs.VPC) (*irs.VPCInfo, error) {
	cblogger.Info("NHN Cloud cloud driver: called mappingVpcSubnetInfo()!!")

	vpcInfo := irs.VPCInfo {
		IId: irs.IID{
			NameId:   vpc.Name,
			SystemId: vpc.ID,
		},
	}
	vpcInfo.IPv4_CIDR = vpc.CIDRv4

	// ### Since New NHN Cloud VPC API 'GET ~/v2.0/vpcs' for Getting VPC List does Not return Subnet List
	listOpts := subnets.ListOpts{
		VPCID: vpc.ID,
	}
	allPages, err := subnets.List(vpcHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Subnet Pages from NHN Cloud!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	subnetList, err := subnets.ExtractSubnets(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Subnet List from NHN Cloud!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Get Subnet info list.
	var subnetInfoList []irs.SubnetInfo
	if len(subnetList) > 0 {
		for _, subnet := range subnetList {
			subnetInfo := vpcHandler.mappingSubnetInfo(subnet)
			subnetInfoList = append(subnetInfoList, *subnetInfo)
		}
	}	
	vpcInfo.SubnetInfoList = subnetInfoList

	var RouterExternal string
	if vpc.RouterExternal {
		RouterExternal = "Yes"
	} else if !vpc.RouterExternal {
		RouterExternal = "No"
	}

	// Shoud Add Create time
	keyValueList := []irs.KeyValue{
		{Key: "Status", Value: vpc.State},
		{Key: "RouterExternal", Value: RouterExternal},
	}
	vpcInfo.KeyValueList = keyValueList

	return &vpcInfo, nil
}

func (vpcHandler *NhnCloudVPCHandler) mappingSubnetInfo(subnet subnets.Subnet) *irs.SubnetInfo { // subnets.Subnets
	cblogger.Info("NHN Cloud cloud driver: called mappingSubnetInfo()!!")
	// spew.Dump(subnet)
	
	subnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId:   subnet.Name,
			SystemId: subnet.ID,
		},
		IPv4_CIDR: subnet.CIDR,
	}

	var RouterExternal string
	if subnet.RouterExternal {
		RouterExternal = "Yes"
	} else if !subnet.RouterExternal {
		RouterExternal = "No"
	}

	keyValueList := []irs.KeyValue{
		{Key: "VPCId", Value: subnet.VPCID},
		{Key: "RouterExternal", Value: RouterExternal},
		{Key: "CreatedTime", Value: subnet.CreateTime},		
	}
	subnetInfo.KeyValueList = keyValueList
	return &subnetInfo
}

// Waiting for up to 500 seconds during Disk creation until Disk info. can be get
func (vpcHandler *NhnCloudVPCHandler) waitForVpcCreation(vpcIID irs.IID) (string, error) {
	cblogger.Info("===> Since VPC info. cannot be retrieved immediately after VPC creation, it waits until running.")

	curRetryCnt := 0
	maxRetryCnt := 500
	for {
		curStatus, err := vpcHandler.getVpcStatus(vpcIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the VPC Status of [%s] : [%v] ", vpcIID.NameId, err)
			cblogger.Error(newErr.Error())
			return "Failed. ", newErr
		} else {
			cblogger.Infof("Succeeded in Getting the VPC Status of [%s] : [%s]", vpcIID.NameId, curStatus)
		}

		cblogger.Infof("===> VPC Status : [%s]", curStatus)

		switch curStatus {
		case "creating":
			curRetryCnt++
			cblogger.Infof("The Disk is still 'Creating', so wait for a second more before inquiring the Disk info.")
			time.Sleep(time.Second * 2)
			if curRetryCnt > maxRetryCnt {
				newErr := fmt.Errorf("Despite waiting for a long time(%d sec), the VPC status is %s, so it is forcibly finished.", maxRetryCnt, curStatus)
				cblogger.Error(newErr.Error())
				return "Failed. ", newErr
			}
		case "available":
			cblogger.Infof("===> ### The VPC 'Creation' is Finished, stopping the waiting.")
			return curStatus, nil
		default:
			cblogger.Infof("===> ### The VPC 'Creation' is Faild, stopping the waiting.")
			return curStatus, nil
			//break
		}
	}
}

func (vpcHandler *NhnCloudVPCHandler) getVpcStatus(vpcIID irs.IID) (string, error) {
	cblogger.Info("NHN Cloud Driver: called getVPCStatus()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	vpcResult, err := vpcs.Get(vpcHandler.NetworkClient, vpcIID.SystemId).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NHN VPC Info!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	cblogger.Infof("# VPC Creation Status of NHN Cloud : [%s]", vpcResult.State)
	return vpcResult.State, nil
}

func (vpcHandler *NhnCloudVPCHandler) getNnnVPC(vpcId string) (vpcs.VPC, error) {
	cblogger.Info("NHN Cloud Cloud Driver: called getNnnVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcId, "getNnnVPC()")

	if strings.EqualFold(vpcId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return vpcs.VPC{}, newErr
	}

	start := call.Start()
	var vpc vpcs.VPC
	err := vpcs.Get(vpcHandler.NetworkClient, vpcId).ExtractInto(&vpc)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VPC Info from NHN Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return vpcs.VPC{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	return vpc, nil
}

// # Check whether the Routing Table (of the VPC) is connected to an Internet Gateway
func (vpcHandler *NhnCloudVPCHandler) isConnectedToGateway(vpcId string) (bool, error) {
	cblogger.Info("NHN Cloud Cloud Driver: called isConnectedToGateway()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcId, "isConnectedToGateway()")

	if strings.EqualFold(vpcId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	vpcInfo, err := vpcHandler.getNnnVPC(vpcId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VPC Info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	hasInternetGateway := false
	if !strings.EqualFold(vpcInfo.RoutingTables[0].GatewayID, "") {
		hasInternetGateway = true
	}
	return hasInternetGateway, nil
}
