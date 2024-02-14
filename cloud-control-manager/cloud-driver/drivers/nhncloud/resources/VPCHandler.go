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

package resources

import (
	"errors"
	"strings"
	"fmt"
	"sync"
	// "github.com/davecgh/go-spew/spew"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/extensions/external"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/networks"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/subnets"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NhnCloudVPCHandler struct {
	RegionInfo 		idrv.RegionInfo
	NetworkClient   *nhnsdk.ServiceClient
}

type NetworkWithExt struct {
	networks.Network
	external.NetworkExternalExt
}

func (vpcHandler *NhnCloudVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger.Info("NHN Cloud Cloud Driver: called CreateVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Region, call.VPCSUBNET, vpcReqInfo.IId.NameId, "CreateVPC()")

	if strings.EqualFold(vpcReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid VPC NameId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

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

	vpcInfo, err := vpcHandler.GetVPC(irs.IID{SystemId: vpcSystemId})
	if err != nil {
		cblogger.Errorf("Failed to Find any VPC Info with the SystemId. : [%s], %v", vpcSystemId, err)
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
	var vpc NetworkWithExt
	err := networks.Get(vpcHandler.NetworkClient, vpcIID.SystemId).ExtractInto(&vpc)
	if err != nil {
		cblogger.Errorf("Failed to Get VPC with the SystemId : [%s]. %v", vpcIID.SystemId, err)
		LoggingError(callLogInfo, err)
		return irs.VPCInfo{}, err
	}
	LoggingInfo(callLogInfo, start)

	vpcInfo := vpcHandler.mappingVpcInfo(vpc)
	return *vpcInfo, nil
}

func (vpcHandler *NhnCloudVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger.Info("NHN Cloud Cloud Driver: called ListVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Region, call.VPCSUBNET, "ListVPC()", "ListVPC()")

	start := call.Start()
	listOpts := external.ListOptsExt{
		ListOptsBuilder: networks.ListOpts{},
	}
	allPages, err := networks.List(vpcHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		cblogger.Errorf("Failed to Get Network list. %v", err)
		LoggingError(callLogInfo, err)
		return nil, err
	}
	LoggingInfo(callLogInfo, start)

	// To Get VPC info list
	var vpcList []NetworkWithExt
	err = networks.ExtractNetworksInto(allPages, &vpcList)
	if err != nil {
		cblogger.Errorf("Failed to Get VPC list. %v", err)
		LoggingError(callLogInfo, err)
		return nil, err
	}

	var vpcInfoList []*irs.VPCInfo
	for _, vpc := range vpcList {
		vpcInfo := vpcHandler.mappingVpcInfo(vpc)
		vpcInfoList = append(vpcInfoList, vpcInfo)
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

	listOpts := external.ListOptsExt{
		ListOptsBuilder: networks.ListOpts{},
	}
	allPages, err := networks.List(vpcHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		cblogger.Errorf("Failed to Get Network list. %v", err)
		LoggingError(callLogInfo, err)
		return irs.SubnetInfo{}, err
	}

	// To Get VPC info list
	var vpcList []NetworkWithExt
	err = networks.ExtractNetworksInto(allPages, &vpcList)
	if err != nil {
		cblogger.Errorf("Failed to Get VPC list. %v", err)
		LoggingError(callLogInfo, err)
		return irs.SubnetInfo{}, err
	}

	var newSubnetInfo irs.SubnetInfo

	// To Get Subnet info with the originalNameId
	for _, vpc := range vpcList {
		for _, subnetId := range vpc.Subnets {
			subnetInfo, err := vpcHandler.getSubnet(irs.IID{SystemId: subnetId})
			if err != nil {
				cblogger.Errorf("Failed to Get Subnet with Id [%s] : %s", subnetId, err)
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

func (vpcHandler *NhnCloudVPCHandler) mappingVpcInfo(nvpc NetworkWithExt) *irs.VPCInfo {
	// Mapping VPC info.
	vpcInfo := irs.VPCInfo {
		IId: irs.IID{
			NameId:   nvpc.Name,
			SystemId: nvpc.ID,
		},
	}
	// vpcInfo.IPv4_CIDR = "N/A"

	var External string
	if nvpc.External {
		External = "Yes"
	} else if !nvpc.External {
		External = "No"
	}

	keyValueList := []irs.KeyValue{
		{Key: "Status", Value: nvpc.Status},
		{Key: "External_Network", Value: External},
	}
	vpcInfo.KeyValueList = keyValueList

	// Get Subnet info list.
	var subnetInfoList []irs.SubnetInfo
	var wait sync.WaitGroup
	for _, subnetId := range nvpc.Subnets {
		wait.Add(1)
		go func(subnetId string) {
			defer wait.Done()
			subnetInfo, err := vpcHandler.getSubnet(irs.IID{SystemId: subnetId})
			if err != nil {
				cblogger.Errorf("Failed to Get Subnet info with the subnetId [%s]. [%v]", subnetId, err)
				// continue
			}
			subnetInfoList = append(subnetInfoList, subnetInfo)
		}(subnetId)
	}
	wait.Wait()

	vpcInfo.SubnetInfoList = subnetInfoList
	return &vpcInfo
}

func (vpcHandler *NhnCloudVPCHandler) mappingSubnetInfo(subnet subnets.Subnet) *irs.SubnetInfo {
	// spew.Dump(subnet)
	subnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId:   subnet.Name,
			SystemId: subnet.ID,
		},
		IPv4_CIDR: subnet.CIDR,
	}

	keyValueList := []irs.KeyValue{
		{Key: "VPCId", Value: subnet.NetworkID},
	}
	subnetInfo.KeyValueList = keyValueList
	return &subnetInfo
}
