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
	"errors"
	"strings"
	"fmt"
	// "sync"
	"time"
	"github.com/davecgh/go-spew/spew"

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
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Region, call.VPCSUBNET, vpcReqInfo.IId.NameId, "CreateVPC()")

	if strings.EqualFold(vpcReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid VPC NameId!!")
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}
	
	if strings.EqualFold(vpcHandler.CredentialInfo.TenantId, "") {
		newErr := fmt.Errorf("Invalid Tenant ID!!")
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	createOpts := vpcs.CreateOpts{
		VPC: vpcs.NewVPC {
			Name:       vpcReqInfo.IId.NameId,
			CIDRv4:		vpcReqInfo.IPv4_CIDR,
			TenantID: 	vpcHandler.CredentialInfo.TenantId, // Caution!!
		},
	}
	cblogger.Info("\n\n### createOpts : ")
	spew.Dump(createOpts)
	cblogger.Info("\n")

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

	// Wait for created Disk info to be inquired
	curStatus, err := vpcHandler.waitForVpcCreation(newVpcIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait to Get VPC Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	cblogger.Infof("==> Disk Status of [%s] : [%s]", newVpcIID.NameId, curStatus)

	/*

	// Get VPC list
	vpcList, err := vpcHandler.ListVPC()
	if err != nil {
		cblogger.Errorf("Failed to Get VPC list : %v", err)
		LoggingError(callLogInfo, err)
		return irs.VPCInfo{}, err
	}

	// Search VPC SystemId by NameID in the VPC list
	var vpcSystemId string
	for _, curVPC := range vpcList {
		if strings.EqualFold(curVPC.IId.NameId, "Default Network") {
			vpcSystemId = curVPC.IId.SystemId
			cblogger.Infof("# SystemId of the VPC : [%s]", vpcSystemId)
			break
		}
	}

	// When the "Default Network" VPC is not found
	if strings.EqualFold(vpcSystemId, "") {
		cblogger.Error("Failed to Find the 'Default Network' VPC on your NHN Cloud project!!")
		return irs.VPCInfo{}, nil
	}
	*/

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
		newSubnet, err := vpcHandler.createSubnet(subnet) // Caution!! For IID2 NameID validation check for Subnet
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
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Region, call.VPCSUBNET, vpcIID.SystemId, "GetVPC()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

	start := call.Start()
	var vpc vpcs.VPC
	// var vpc NetworkWithExt
	err := vpcs.Get(vpcHandler.NetworkClient, vpcIID.SystemId).ExtractInto(&vpc)
	// err := vpcs.Get(vpcHandler.NetworkClient, vpcIID.SystemId).ExtractInto(&vpc) 
	if err != nil {
		cblogger.Errorf("Failed to Get VPC with the SystemId : [%s]. %v", vpcIID.SystemId, err)
		LoggingError(callLogInfo, err)
		return irs.VPCInfo{}, err
	}
	LoggingInfo(callLogInfo, start)


	// cblogger.Info("\n\n### vpc : ")
	// spew.Dump(vpc)
	// cblogger.Info("\n")


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
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Region, call.VPCSUBNET, "ListVPC()", "ListVPC()")

	if strings.EqualFold(vpcHandler.CredentialInfo.TenantId, "") {
		newErr := fmt.Errorf("Invalid Tenant ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	start := call.Start()
	listOpts := vpcs.ListOpts{
		TenantID: vpcHandler.CredentialInfo.TenantId,
	}
	// listOpts := external.ListOptsExt{
	// 	ListOptsBuilder: vpcs.ListOpts{},
	// }
	allPages, err := vpcs.List(vpcHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		cblogger.Errorf("Failed to Get VPC Pages. %v", err)
		LoggingError(callLogInfo, err)
		return nil, err
	}
	LoggingInfo(callLogInfo, start)

	// To Get VPC info list
	// var vpcList []vpcs.VPC
	// var vpcList []NetworkWithExt
	vpcList, err := vpcs.ExtractVPCs(allPages)
	// err = vpcs.ExtractVPCsInto(allPages, &vpcList)
	if err != nil {
		cblogger.Errorf("Failed to Get VPC list from NHN Cloud. %v", err)
		LoggingError(callLogInfo, err)
		return nil, err
	}
	
	// cblogger.Info("\n\n### vpcList : ")
	// spew.Dump(vpcList)
	// cblogger.Info("\n")

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
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Region, call.VPCSUBNET, vpcIID.SystemId, "DeleteVPC()")

	//To check whether the VPC exists.
	cblogger.Infof("vpcIID.SystemId to Delete : [%s]", vpcIID.SystemId)

	vpcInfo, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		cblogger.Errorf("Failed to Find the VPC with the SystemID. : [%s] : [%v]", vpcIID.SystemId, err)
		LoggingError(callLogInfo, err)
		return false, err
	} else {
		cblogger.Infof("Succeeded in Deleting the VPC : " + vpcInfo.IId.SystemId)
	}

	return true, nil
}

func (vpcHandler *NhnCloudVPCHandler) createSubnet(subnetReqInfo irs.SubnetInfo) (irs.SubnetInfo, error) {
	cblogger.Info("NHN Cloud cloud driver: called createSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Region, call.VPCSUBNET, subnetReqInfo.IId.NameId, "createSubnet()")

	if subnetReqInfo.IId.NameId == "" {
		newErr := fmt.Errorf("Invalid Subnet NameId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SubnetInfo{}, newErr
	}

	listOpts := vpcs.ListOpts{}
	// listOpts := external.ListOptsExt{
	// 	ListOptsBuilder: vpcs.ListOpts{},
	// }
	allPages, err := vpcs.List(vpcHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		cblogger.Errorf("Failed to Get Network list. %v", err)
		LoggingError(callLogInfo, err)
		return irs.SubnetInfo{}, err
	}

	// To Get VPC info list
	var vpcList []NetworkWithExt
	err = vpcs.ExtractVPCsInto(allPages, &vpcList)
	if err != nil {
		cblogger.Errorf("Failed to Get VPC list. %v", err)
		LoggingError(callLogInfo, err)
		return irs.SubnetInfo{}, err
	}

	var newSubnetInfo irs.SubnetInfo

	// To Get Subnet info with the Subent ID
	for _, vpc := range vpcList {
		for _, subnet := range vpc.Subnets {
			subnetInfo, err := vpcHandler.getSubnet(irs.IID{SystemId: subnet.ID})
			if err != nil {
				cblogger.Errorf("Failed to Get Subnet with Id [%s] : %s", subnet.ID, err)
				continue
			}
			if strings.EqualFold(subnetInfo.IId.NameId, "Default Network") {
				cblogger.Infof("# Found Default Subnet on NHN Cloud : [%s]", subnetInfo.IId.NameId)
				newSubnetInfo = subnetInfo
				newSubnetInfo.IId.NameId = subnetReqInfo.IId.NameId //Caution!! For IID2 NameID validation check
				break
			}
		}
	}

	// When the Subnet is not found
	if newSubnetInfo.IId.SystemId == "" {
		newErr := fmt.Errorf("Failed to Find the 'Default Network' Subnet on your NHN Cloud project!!")
		LoggingError(callLogInfo, newErr)
		return irs.SubnetInfo{}, newErr
	}
	return newSubnetInfo, err
}

func (vpcHandler *NhnCloudVPCHandler) getSubnet(subnetIId irs.IID) (irs.SubnetInfo, error) {
	cblogger.Info("NHN Cloud cloud driver: called getSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Region, call.VPCSUBNET, subnetIId.SystemId, "getSubnet()")

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

	return irs.VPCInfo{}, errors.New("Does not support AddSubnet() yet!!")
}

func (vpcHandler *NhnCloudVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud cloud driver: called RemoveSubnet()!!")

	return true, errors.New("Does not support RemoveSubnet() yet!!")
}

func (vpcHandler *NhnCloudVPCHandler) DeleteSubnet(subnetIId irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud cloud driver: called DeleteSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Region, call.VPCSUBNET, subnetIId.SystemId, "DeleteSubnet()")

	//To check whether the Subnet exists.
	cblogger.Infof("subnetIId.SystemId to Delete : [%s]", subnetIId.SystemId)

	subnetInfo, err := vpcHandler.getSubnet(subnetIId)
	if err != nil {
		cblogger.Errorf("Failed to Find the Subnet with the SystemID. : [%s] : %v", subnetIId.SystemId, err)
		LoggingError(callLogInfo, err)
		return false, err
	} else {
		cblogger.Infof("Succeeded in Deleting the Subnet : [%s]", subnetInfo.IId.SystemId)
	}
	return true, nil
}

func (vpcHandler *NhnCloudVPCHandler) mappingVpcInfo(vpc vpcs.VPC) (*irs.VPCInfo, error) {
	cblogger.Info("NHN Cloud cloud driver: called mappingVpcInfo()!!")

	// Mapping VPC info.
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
		for _, subnet := range vpc.Subnets {  // Caution!!) vpc.Subnets (Not subnets.Subnets)
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
	}
	vpcInfo.KeyValueList = keyValueList

	return &vpcInfo, nil
}

func (vpcHandler *NhnCloudVPCHandler) mappingVpcSubnetInfo(vpc vpcs.VPC) (*irs.VPCInfo, error) {
	cblogger.Info("NHN Cloud cloud driver: called mappingVpcSubnetInfo()!!")

	// Mapping VPC info.
	vpcInfo := irs.VPCInfo {
		IId: irs.IID{
			NameId:   vpc.Name,
			SystemId: vpc.ID,
		},
	}
	vpcInfo.IPv4_CIDR = vpc.CIDRv4

	// ### Since New API 'GET ~/v2.0/vpcs' for Getting VPC List does Not return Subnet List
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
		// {Key: "CreateTime", Value: subnet.CreateTime},
		
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
