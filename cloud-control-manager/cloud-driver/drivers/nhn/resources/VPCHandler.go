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
// by ETRI 2025.08. updated (New REST API for Internet Gateway Mgmt. Applied)

package resources

import (
	"errors"
	"fmt"
	"strings"

	// "sync"
	"time"
	// "github.com/davecgh/go-spew/spew"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	nhnstack "github.com/cloud-barista/nhncloud-sdk-go/openstack"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/extensions/external"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/vpcs"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/vpcsubnets"
	"github.com/cloud-barista/nhncloud-sdk-go/pagination"

	igw "github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/extensions/layer3/internetgateways"
	rt "github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/extensions/layer3/routers"
	rtable "github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/extensions/layer3/routingtables"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NhnCloudVPCHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	NetworkClient  *nhnsdk.ServiceClient
}

type NetworkWithExt struct {
	vpcs.VPC
	external.NetworkExternalExt
}

// WaitCondition defines what condition to wait for
type WaitCondition struct {
	TargetState         igw.InternetGatewayState
	TargetMigrateStatus igw.MigrateStatus
	AvoidErrorState     bool
}

func (vpcHandler *NhnCloudVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger.Info("NHN Cloud Driver: called CreateVPC()!")
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
		Name:     vpcReqInfo.IId.NameId,
		CIDRv4:   vpcReqInfo.IPv4_CIDR,
		TenantID: vpcHandler.CredentialInfo.TenantId, // Need to Specify!!
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

	// Create Internet Gateway and Attache to Default Routing Table of the Created VPC
	var internetGW string
	internetGW, getErr := vpcHandler.setInternetGateway(vpcResult.ID)
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Create Internet Gateway for the VPC : [%v]", getErr)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}
	cblogger.Infof("# New Internet Gateway ID : [%s]", internetGW)

	vpcInfo, err := vpcHandler.GetVPC(irs.IID{SystemId: vpcResult.ID})
	if err != nil {
		cblogger.Errorf("Failed to Find any VPC Info with the SystemId. : [%s], %v", vpcResult.ID, err)
		LoggingError(callLogInfo, err)
		return irs.VPCInfo{}, err
	} else {
		vpcInfo.IId.NameId = vpcReqInfo.IId.NameId // Caution!! For IID2 NameID validation check for VPC
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
	cblogger.Info("NHN Cloud Driver: called GetVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcIID.SystemId, "GetVPC()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	start := call.Start()
	vpc, err := vpcHandler.getRawVPC(vpcIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	vpcInfo, err := vpcHandler.mappingVpcInfo(*vpc)
	if err != nil {
		newErr := fmt.Errorf("Failed to Map the VPC Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}
	return *vpcInfo, nil
}

func (vpcHandler *NhnCloudVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ListVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, "ListVPC()", "ListVPC()")

	if strings.EqualFold(vpcHandler.CredentialInfo.TenantId, "") {
		newErr := fmt.Errorf("Invalid Tenant ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	start := call.Start()
	listOpts := vpcs.ListOpts{
		TenantID:       vpcHandler.CredentialInfo.TenantId,
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
	cblogger.Info("NHN Cloud Driver: called DeleteVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcIID.SystemId, "DeleteVPC()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC System ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	// Detache Internet Gateway from the Default Routing Table of the VPC and Delete it
	_, removeErr := vpcHandler.removeInternetGateway(vpcIID.SystemId)
	if removeErr != nil {
		newErr := fmt.Errorf("Failed to Remove the Internet Gateway for the VPC : [%v]", removeErr)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	vpc, err := vpcHandler.getRawVPC(vpcIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	var subnets []vpcs.Subnet

	// Remove duplicated subnets (NHN provide duplicated subnets)
	for _, subnet := range vpc.Subnets {
		var skip bool

		for _, sb := range subnets {
			if sb.ID == subnet.ID {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		subnets = append(subnets, subnet)
	}

	for _, subnet := range subnets {
		if _, err := vpcHandler.RemoveSubnet(irs.IID{SystemId: vpc.ID}, irs.IID{SystemId: subnet.ID}); err != nil {
			newErr := fmt.Errorf("Failed to Remove the Subnet with the SystemID. : [%s] : [%v]", subnet.ID, err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, err)
			return false, newErr
		}
		time.Sleep(time.Second * 2)
	}

	err = vpcs.Delete(vpcHandler.NetworkClient, vpc.ID).ExtractErr()
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the VPC with the SystemID. : [%s] : [%v]", vpc.ID, err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, err)
		return false, newErr
	} else {
		cblogger.Infof("Succeeded in Deleting the VPC : " + vpc.ID)
	}

	return true, nil
}

func (vpcHandler *NhnCloudVPCHandler) createSubnet(vpcId string, subnetReqInfo irs.SubnetInfo) (irs.SubnetInfo, error) {
	cblogger.Info("NHN Cloud driver: called createSubnet()!!")
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

	createOpts := vpcsubnets.CreateOpts{
		VpcID: vpcId,
		CIDR:  subnetReqInfo.IPv4_CIDR,
		Name:  subnetReqInfo.IId.NameId,
	}
	start := call.Start()
	vpcsubnet, err := vpcsubnets.Create(vpcHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Createt New Subnet!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.SubnetInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	subnetInfo := vpcHandler.mappingSubnetInfo(*vpcsubnet)
	return *subnetInfo, nil
}

func (vpcHandler *NhnCloudVPCHandler) getSubnet(subnetIId irs.IID) (irs.SubnetInfo, error) {
	cblogger.Info("NHN Cloud driver: called getSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, subnetIId.SystemId, "getSubnet()")

	if strings.EqualFold(subnetIId.SystemId, "") {
		newErr := fmt.Errorf("Invalid Subnet SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SubnetInfo{}, newErr
	}

	vpcsubnet, err := vpcsubnets.Get(vpcHandler.NetworkClient, subnetIId.SystemId).Extract()
	if err != nil {
		cblogger.Errorf("Failed to Get Subnet with SystemId [%s] : %v", subnetIId.SystemId, err)
		LoggingError(callLogInfo, err)
		return irs.SubnetInfo{}, nil
	}
	subnetInfo := vpcHandler.mappingSubnetInfo(*vpcsubnet)
	return *subnetInfo, nil
}

func (vpcHandler *NhnCloudVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	cblogger.Info("NHN Cloud driver: called AddSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcIID.SystemId, "AddSubnet()")

	vpc, err := vpcHandler.getRawVPC(vpcIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VPC : [%v]", err)
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

	createOpts := vpcsubnets.CreateOpts{
		VpcID: vpc.ID,
		CIDR:  subnetInfo.IPv4_CIDR,
		Name:  subnetInfo.IId.NameId,
	}
	start := call.Start()
	_, err = vpcsubnets.Create(vpcHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Createt New Subnet!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	vpcInfo, err := vpcHandler.GetVPC(irs.IID{SystemId: vpc.ID})
	if err != nil {
		cblogger.Errorf("Failed to Get the VPC Info with the SystemId. : [%s], %v", vpc.ID, err)
		LoggingError(callLogInfo, err)
		return irs.VPCInfo{}, err
	}
	return vpcInfo, nil
}

func (vpcHandler *NhnCloudVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud driver: called RemoveSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, subnetIID.SystemId, "RemoveSubnet()")

	if vpcIID.SystemId == "" && vpcIID.NameId == "" {
		return false, errors.New("invalid vpcIID")
	}

	if subnetIID.SystemId == "" && subnetIID.NameId == "" {
		return false, errors.New("invalid subnetIID")
	}

	if vpcIID.SystemId == "" || subnetIID.SystemId == "" {
		vpc, err := vpcHandler.getRawVPC(vpcIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the VPC : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return false, newErr
		}
		vpcIID.SystemId = vpc.ID

		if subnetIID.SystemId == "" {
			var subnets []vpcs.Subnet

			// Remove duplicated subnets (NHN provide duplicated subnets)
			for _, subnet := range vpc.Subnets {
				var skip bool
				for _, sb := range subnets {
					if sb.ID == subnet.ID {
						skip = true
						break
					}
				}
				if skip {
					continue
				}
				subnets = append(subnets, subnet)
			}

			for _, subnet := range subnets {
				if subnet.Name == subnetIID.NameId {
					subnetIID.SystemId = subnet.ID
					break
				}
			}
		}
	}

	err := vpcsubnets.Delete(vpcHandler.NetworkClient, subnetIID.SystemId).ExtractErr()
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
	cblogger.Info("NHN Cloud driver: called mappingVpcInfo()!!")
	// cblogger.Info("\n\n### vpc : ")
	// spew.Dump(vpc)
	// cblogger.Info("")

	vpcInfo := irs.VPCInfo{
		IId: irs.IID{
			NameId:   vpc.Name,
			SystemId: vpc.ID,
		},
	}
	vpcInfo.IPv4_CIDR = vpc.CIDRv4

	var subnets []vpcs.Subnet
	// Remove duplicated subnets (NHN provide duplicated subnets)
	for _, subnet := range vpc.Subnets {
		var skip bool
		for _, sb := range subnets {
			if sb.ID == subnet.ID {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		subnets = append(subnets, subnet)
	}

	// Get Subnet info list.
	var subnetInfoList []irs.SubnetInfo
	for _, subnet := range subnets { // Because of vpcs.Subnet type (Not subnets.Subnet type), need to getSubnet()
		var skip bool

		for _, subnetInfo := range subnetInfoList {
			if subnetInfo.IId.SystemId == subnet.ID {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		subnetInfo, err := vpcHandler.getSubnet(irs.IID{SystemId: subnet.ID})
		if err != nil {
			newErr := fmt.Errorf("Failed to Get Subnet info with the subnetId [%s]. [%v]", subnet.ID, err)
			cblogger.Error(newErr.Error())
			return nil, newErr
		}
		subnetInfoList = append(subnetInfoList, subnetInfo)
	}
	vpcInfo.SubnetInfoList = subnetInfoList

	var RouterExternal string
	if vpc.RouterExternal {
		RouterExternal = "Yes"
	} else if !vpc.RouterExternal {
		RouterExternal = "No"
	}

	vpcInfo.KeyValueList = irs.StructToKeyValueList(vpc)

	keyValueList := []irs.KeyValue{
		{Key: "Status", Value: vpc.State},
		{Key: "RouterExternal", Value: RouterExternal},
		{Key: "CreatedTime", Value: vpc.CreateTime},
	}
	vpcInfo.KeyValueList = keyValueList

	return &vpcInfo, nil
}

func (vpcHandler *NhnCloudVPCHandler) mappingVpcSubnetInfo(vpc vpcs.VPC) (*irs.VPCInfo, error) {
	cblogger.Info("NHN Cloud driver: called mappingVpcSubnetInfo()!!")

	vpcInfo := irs.VPCInfo{
		IId: irs.IID{
			NameId:   vpc.Name,
			SystemId: vpc.ID,
		},
	}
	vpcInfo.IPv4_CIDR = vpc.CIDRv4

	// ### Since New NHN Cloud VPC API 'GET ~/v2.0/vpcs' for Getting VPC List does Not return Subnet List
	listOpts := vpcsubnets.ListOpts{
		VPCID: vpc.ID,
	}
	allPages, err := vpcsubnets.List(vpcHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Subnet Pages from NHN Cloud!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	vpcsubnetList, err := vpcsubnets.ExtractVpcsubnets(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Subnet List from NHN Cloud!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Get Subnet info list.
	var subnetInfoList []irs.SubnetInfo
	if len(vpcsubnetList) > 0 {
		for _, vpcsubnet := range vpcsubnetList {
			subnetInfo := vpcHandler.mappingSubnetInfo(vpcsubnet)
			subnetInfoList = append(subnetInfoList, *subnetInfo)
		}
	}
	vpcInfo.SubnetInfoList = subnetInfoList

	//var RouterExternal string
	//if vpc.RouterExternal {
	//	RouterExternal = "Yes"
	//} else if !vpc.RouterExternal {
	//	RouterExternal = "No"
	//}

	// Shoud Add Create time
	//keyValueList := []irs.KeyValue{
	//	{Key: "Status", Value: vpc.State},
	//	{Key: "RouterExternal", Value: RouterExternal},
	//}
	//vpcInfo.KeyValueList = keyValueList
	//
	vpcInfo.KeyValueList = irs.StructToKeyValueList(vpc)

	return &vpcInfo, nil
}

func (vpcHandler *NhnCloudVPCHandler) mappingSubnetInfo(subnet vpcsubnets.Vpcsubnet) *irs.SubnetInfo { // subnets.Subnets
	cblogger.Info("NHN Cloud driver: called mappingSubnetInfo()!!")
	// spew.Dump(subnet)

	subnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId:   subnet.Name,
			SystemId: subnet.ID,
		},
		IPv4_CIDR: subnet.CIDR,
	}

	subnetInfo.KeyValueList = irs.StructToKeyValueList(subnet)

	//var RouterExternal string
	//if subnet.RouterExternal {
	//	RouterExternal = "Yes"
	//} else if !subnet.RouterExternal {
	//	RouterExternal = "No"
	//}

	//keyValueList := []irs.KeyValue{
	//	{Key: "VPCId", Value: subnet.VPCID},
	//	{Key: "RouterExternal", Value: RouterExternal},
	//	{Key: "CreatedTime", Value: subnet.CreateTime},
	//}
	//subnetInfo.KeyValueList = keyValueList

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

// # Check whether the Routing Table (of the VPC) is connected to an Internet Gateway
func (vpcHandler *NhnCloudVPCHandler) isConnectedToGateway(vpcId string) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called isConnectedToGateway()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcId, "isConnectedToGateway()")

	if strings.EqualFold(vpcId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	vpc, err := vpcHandler.getRawVPC(irs.IID{SystemId: vpcId})
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	hasInternetGateway := false
	if !strings.EqualFold(vpc.RoutingTables[0].GatewayID, "") {
		hasInternetGateway = true
	}
	return hasInternetGateway, nil
}

func (vpcHandler *NhnCloudVPCHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, "vpdId", "ListIID()")

	start := call.Start()

	var iidList []*irs.IID

	listOpts := vpcs.ListOpts{}

	allPages, err := vpcs.List(vpcHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VPC information from NhnCloud!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return make([]*irs.IID, 0), newErr
	}

	allVpcs, err := vpcs.ExtractVPCs(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VPC List from NhnCloud!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return make([]*irs.IID, 0), newErr
	}

	for _, vpc := range allVpcs {
		var iid irs.IID
		iid.SystemId = vpc.ID
		iid.NameId = vpc.Name

		iidList = append(iidList, &iid)
	}

	LoggingInfo(callLogInfo, start)

	return iidList, nil
}

func (vpcHandler *NhnCloudVPCHandler) getRawVPC(vpcIID irs.IID) (*vpcs.VPC, error) {
	cblogger.Info("NHN Cloud Driver: called getRawVPC()!")

	if vpcIID.SystemId == "" && vpcIID.NameId == "" {
		return nil, errors.New("invalid IID")
	}

	var vpc vpcs.VPC

	if vpcIID.SystemId == "" {
		listOpts := external.ListOptsExt{
			ListOptsBuilder: vpcs.ListOpts{
				Name: vpcIID.NameId,
			},
		}
		page, err := vpcs.List(vpcHandler.NetworkClient, listOpts).AllPages()
		if err != nil {
			return nil, err
		}

		vpcList, err := vpcs.ExtractVPCs(page)
		if err != nil {
			return nil, err
		}

		var vpcFound bool
		for _, v := range vpcList {
			if v.Name == vpcIID.NameId {
				vpc = v
				vpcFound = true
				break
			}
		}

		if !vpcFound {
			return nil, errors.New("not found vpc")
		}
	} else {
		err := vpcs.Get(vpcHandler.NetworkClient, vpcIID.SystemId).ExtractInto(&vpc)
		if err != nil {
			return nil, err
		}
	}

	subnetListOpts := vpcsubnets.ListOpts{
		VPCID: vpc.ID,
	}
	page, err := vpcsubnets.List(vpcHandler.NetworkClient, subnetListOpts).AllPages()
	if err != nil {
		return nil, err
	}
	subnetList, err := vpcsubnets.ExtractVpcsubnets(page)
	if err != nil {
		return nil, err
	}

	for _, subnet := range subnetList {
		var routes []vpcs.Route

		for _, route := range subnet.Routes {
			routes = append(routes, vpcs.Route{
				SubnetID:  route.SubnetID,
				TenantID:  route.TenantID,
				Mask:      route.Mask,
				Gateway:   route.Gateway,
				GatewayID: route.Gateway,
				CIDR:      route.CIDR,
				ID:        route.ID,
			})
		}

		vpc.Subnets = append(vpc.Subnets, vpcs.Subnet{
			Name: subnet.Name,
			ID:   subnet.ID,
		})
	}

	return &vpc, nil
}

func (vpcHandler *NhnCloudVPCHandler) listRouter() error {
	cblogger.Info("NHN Cloud Driver: called isConnectedToGateway()!")

	listOpts := rt.ListOpts{}
	allPages, err := rt.List(vpcHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		panic(err)

	}

	allRouters, err := rt.ExtractRouters(allPages)
	if err != nil {
		panic(err)

	}

	for _, router := range allRouters {
		cblogger.Infof("")
		cblogger.Infof("%+v", router)
		cblogger.Infof("")
	}

	return err
}

func (vpcHandler *NhnCloudVPCHandler) listInternetGateways() {

	// Create list options (empty for all gateways)
	listOpts := igw.ListOpts{}

	// Perform the list operation
	pager := igw.List(vpcHandler.NetworkClient, listOpts)

	// Extract all pages
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		gateways, err := igw.ExtractInternetGateways(page)
		if err != nil {
			return false, err
		}

		for _, gateway := range gateways {
			cblogger.Infof("ID: %s, Name: %s, State: %s",
				gateway.ID, gateway.Name, gateway.State)
			cblogger.Infof("  External Network: %s", gateway.ExternalNetworkID)
			if gateway.RoutingTableID != nil {
				cblogger.Infof("  Routing Table: %s", *gateway.RoutingTableID)
			} else {
				cblogger.Infof("  Routing Table: Not connected")
			}
			cblogger.Infof("  Created: %s", gateway.CreateTime.Format(time.RFC3339))
			cblogger.Infof("  Migrate Status: %s", gateway.MigrateStatus)
			cblogger.Info()
		}

		return true, nil
	})

	if err != nil {
		cblogger.Infof("Error listing gateways: %v", err)
	}
}

// hasInternetGatewayByRoutingTable checks if there are any Internet Gateways for a specific routing table
func (vpcHandler *NhnCloudVPCHandler) hasInternetGatewayByRoutingTable(routingtableId string) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called hasInternetGatewayByRoutingTable()!")

	listOpts := igw.ListOpts{
		RoutingTableID: routingtableId,
	}

	pager := igw.List(vpcHandler.NetworkClient, listOpts)
	page, err := pager.AllPages()
	if err != nil {
		return false, fmt.Errorf("Ｆailed to Get list of internet gateways by routingtableId: %w", err)
	}

	gateways, err := igw.ExtractInternetGateways(page)
	if err != nil {
		return false, fmt.Errorf("Ｆailed to Ｅxtract Ｉnternet Ｇateways: %w", err)
	}
	return len(gateways) > 0, nil
}

func (vpcHandler *NhnCloudVPCHandler) getExternalNetId() (string, error) {
	cblogger.Info("NHN Cloud Driver: called getExternalNetId()!")

	listOpts := external.ListOptsExt{
		ListOptsBuilder: vpcs.ListOpts{
			RouterExternal: true,
		},
	}
	page, err := vpcs.List(vpcHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return "", err
	}

	vpcList, err := vpcs.ExtractVPCs(page)
	if err != nil {
		return "", err
	}

	var vpcId string
	for _, v := range vpcList {
		if v.RouterExternal {
			vpcId = v.ID
			break
		}
	}

	if strings.EqualFold(vpcId, "") {
		return "", errors.New("Faild to Get External Net ID")
	}

	return vpcId, nil
}

// Returns New Internet Gatgway ID
func (vpcHandler *NhnCloudVPCHandler) setInternetGateway(vpcId string) (string, error) {
	cblogger.Info("NHN Cloud Driver: called setInternetGateway()!")

	if strings.EqualFold(vpcId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	vpcIID := irs.IID{SystemId: vpcId}
	newVpc, getErr := vpcHandler.getRawVPC(vpcIID)
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Get the VPC : [%v]", getErr)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	var defaultRoutingTableId string
	if len(newVpc.RoutingTables) > 0 {
		for _, routingTable := range newVpc.RoutingTables {
			if routingTable.DefaultTable {
				defaultRoutingTableId = routingTable.ID
				cblogger.Infof("### Default RoutingTable : [%s]", defaultRoutingTableId)
				break
			}
		}
	} else {
		newErr := fmt.Errorf("### No Default Routinng Rable on the VPC. [%s]", vpcId)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	hasIG, err := vpcHandler.hasInternetGatewayByRoutingTable(defaultRoutingTableId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Check if the VPC has Internet Gateway : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	var newIGW string
	if !hasIG {
		cblogger.Infof("# No Internet Gateway attached to the routing table for the VPC : %s", vpcId)

		extNetId, err := vpcHandler.getExternalNetId()
		if err != nil {
			newErr := fmt.Errorf("Failed to Get External Network ID : [%v]", err)
			cblogger.Error(newErr.Error())
			return "", newErr
		}
		cblogger.Infof("# External Network ID [%s]", extNetId)

		iGW, createErr := vpcHandler.createInternetGateway("GW_for_Default_RT", extNetId)
		if createErr != nil {
			newErr := fmt.Errorf("Failed to Check if the VPC has Internet Gateway : [%v]", createErr)
			cblogger.Error(newErr.Error())
			return "", newErr
		}

		attachErr := vpcHandler.attachInternetGateway(defaultRoutingTableId, iGW.ID)
		if attachErr != nil {
			newErr := fmt.Errorf("Failed to Attache the Internet Gateway to the Routing Table : [%v]", attachErr)
			cblogger.Error(newErr.Error())
			return "", newErr
		}
		newIGW = iGW.ID
	}
	return newIGW, nil
}

// createInternetGateway creates a new Internet Gateway and waits for it to be ready
func (vpcHandler *NhnCloudVPCHandler) createInternetGateway(name string, externalNetworkId string) (*igw.InternetGateway, error) {
	cblogger.Info("NHN Cloud Driver: called createInternetGateway()!")

	if strings.EqualFold(name, "") {
		newErr := fmt.Errorf("Invalid InternetGateway Name!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if strings.EqualFold(externalNetworkId, "") {
		newErr := fmt.Errorf("Invalid External Network ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	createOpts := igw.CreateOpts{
		Name:              name,
		ExternalNetworkID: externalNetworkId,
	}
	cblogger.Infof("🔄 Creating Internet Gateway '%s'...", name)

	result := igw.Create(vpcHandler.NetworkClient, createOpts)
	gateway, err := result.Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to create Internet Gateway: %w", err)
	}
	cblogger.Infof("✅ Internet Gateway created:")
	cblogger.Infof("  ID: %s", gateway.ID)
	cblogger.Infof("  Name: %s", gateway.Name)
	// cblogger.Infof("  Initial State: %s", gateway.State)
	// cblogger.Infof("  External Network ID: %s", gateway.ExternalNetworkID)
	// cblogger.Infof("  Created At: %s", gateway.CreateTime.Format(time.RFC3339))

	// Wait for the gateway to be in a stable state based on current status
	cblogger.Infof("🔄 Waiting for Internet Gateway to reach stable state...")
	finalGateway, err := vpcHandler.waitForStableState(gateway)
	if err != nil {
		return gateway, fmt.Errorf("gateway created but failed to reach stable state: %w", err)
	}

	cblogger.Infof("✅ Internet Gateway is now ready!!")
	// cblogger.Infof("  Final State: %s", finalGateway.State)
	// cblogger.Infof("  Migrate Status: %s", finalGateway.MigrateStatus)

	return finalGateway, nil
}

// waitForStableState waits for the gateway to reach a stable state based on its current status
func (vpcHandler *NhnCloudVPCHandler) waitForStableState(gateway *igw.InternetGateway) (*igw.InternetGateway, error) {
	cblogger.Info("NHN Cloud Driver: called waitForStableState()!")

	if gateway == nil {
		newErr := fmt.Errorf("Invalid InternetGateway value!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Default timeout and check interval
	timeout := 10 * time.Minute
	checkInterval := 3 * time.Second

	startTime := time.Now()
	gatewayID := gateway.ID
	currentState := gateway.State
	currentMigrateStatus := gateway.MigrateStatus

	cblogger.Infof("  Current state: %s, migrate status: %s", currentState, currentMigrateStatus)

	// Determine what state to wait for based on current status
	var targetCondition WaitCondition
	var waitDescription string

	switch {
	case currentState == string(igw.StateMigrating):
		// If migrating, wait for migration to complete and reach available or unavailable
		targetCondition = WaitCondition{
			TargetState:         igw.StateUnavailable, // Most common post-migration state
			TargetMigrateStatus: igw.MigrateStatusNone,
			AvoidErrorState:     true,
		}
		waitDescription = "migration to complete"

	case currentMigrateStatus != string(igw.MigrateStatusNone):
		// If any migration process is ongoing, wait for it to complete
		targetCondition = WaitCondition{
			TargetState:         igw.InternetGatewayState(currentState), // Keep current state
			TargetMigrateStatus: igw.MigrateStatusNone,
			AvoidErrorState:     true,
		}
		waitDescription = fmt.Sprintf("migration process (%s) to complete", currentMigrateStatus)

	case currentState == string(igw.StateUnavailable):
		// Already in stable unavailable state, just verify it stays stable
		return vpcHandler.verifyStableState(gatewayID, 30*time.Second, checkInterval)

	case currentState == string(igw.StateAvailable):
		// Already in stable available state, just verify it stays stable
		return vpcHandler.verifyStableState(gatewayID, 30*time.Second, checkInterval)

	case currentState == string(igw.StateError):
		// Already in error state, return immediately
		errorMsg := "unknown error"
		if gateway.MigrateError != nil {
			errorMsg = *gateway.MigrateError
		}
		return nil, fmt.Errorf("gateway is in error state: %s", errorMsg)

	default:
		// For any other state, wait for unavailable (most common stable state for new gateways)
		targetCondition = WaitCondition{
			TargetState:         igw.StateUnavailable,
			TargetMigrateStatus: igw.MigrateStatusNone,
			AvoidErrorState:     true,
		}
		waitDescription = "stable unavailable state"
	}

	cblogger.Infof("  Waiting for %s (timeout: %v)...", waitDescription, timeout)

	// Wait for the determined condition
	for {
		elapsed := time.Since(startTime)

		// Check timeout
		if elapsed > timeout {
			return nil, fmt.Errorf("timeout after %v waiting for %s", elapsed.Round(time.Second), waitDescription)
		}

		// Get current gateway status
		result := igw.Get(vpcHandler.NetworkClient, gatewayID)
		currentGateway, err := result.Extract()
		if err != nil {
			return nil, fmt.Errorf("failed to get gateway status: %w", err)
		}

		// Log status if changed
		if currentGateway.State != currentState || currentGateway.MigrateStatus != currentMigrateStatus {
			cblogger.Infof("    [%v] State: %s → %s, Migrate: %s → %s",
				elapsed.Round(time.Second),
				currentState, currentGateway.State,
				currentMigrateStatus, currentGateway.MigrateStatus)
			currentState = currentGateway.State
			currentMigrateStatus = currentGateway.MigrateStatus
		}

		// Check for error state if we should avoid it
		if targetCondition.AvoidErrorState && currentGateway.State == string(igw.StateError) {
			errorMsg := "unknown error"
			if currentGateway.MigrateError != nil {
				errorMsg = *currentGateway.MigrateError
			}
			return nil, fmt.Errorf("gateway entered error state: %s", errorMsg)
		}

		// Check if target condition is met
		stateMatches := currentGateway.State == string(targetCondition.TargetState)
		migrateMatches := currentGateway.MigrateStatus == string(targetCondition.TargetMigrateStatus)

		if stateMatches && migrateMatches {
			cblogger.Infof("    ✅ Reached stable state after %v", elapsed.Round(time.Second))
			return currentGateway, nil
		}

		// Special handling for migration status changes
		if strings.Contains(currentGateway.MigrateStatus, "error") {
			return nil, fmt.Errorf("migration failed with status: %s", currentGateway.MigrateStatus)
		}

		// Wait before next check
		time.Sleep(checkInterval)
	}
}

// verifyStableState verifies that a gateway remains in its current stable state
func (vpcHandler *NhnCloudVPCHandler) verifyStableState(gatewayId string, verifyDuration, checkInterval time.Duration) (*igw.InternetGateway, error) {
	cblogger.Info("NHN Cloud Driver: called verifyStableState()!")

	if strings.EqualFold(gatewayId, "") {
		newErr := fmt.Errorf("Invalid InternetGateway ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if verifyDuration == 0 {
		newErr := fmt.Errorf("Invalid Verify Duration Value!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if checkInterval == 0 {
		newErr := fmt.Errorf("Invalid Check Interval Value!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	cblogger.Infof("  Verifying gateway remains in stable state for %v...", verifyDuration)

	startTime := time.Now()
	var lastGateway *igw.InternetGateway

	for time.Since(startTime) < verifyDuration {
		result := igw.Get(vpcHandler.NetworkClient, gatewayId)
		gateway, err := result.Extract()
		if err != nil {
			return nil, fmt.Errorf("failed to verify gateway state: %w", err)
		}

		if lastGateway != nil {
			// Check if state changed unexpectedly
			if gateway.State != lastGateway.State {
				return nil, fmt.Errorf("gateway state changed unexpectedly from %s to %s during verification",
					lastGateway.State, gateway.State)
			}

			// Check if migration status changed unexpectedly
			if gateway.MigrateStatus != lastGateway.MigrateStatus {
				cblogger.Infof("    ⚠️  Migrate status changed during verification: %s → %s",
					lastGateway.MigrateStatus, gateway.MigrateStatus)
			}
		}

		lastGateway = gateway
		time.Sleep(checkInterval)
	}

	cblogger.Infof("    ✅ Gateway remained stable for %v", verifyDuration)
	return lastGateway, nil
}

func (vpcHandler *NhnCloudVPCHandler) attachInternetGateway(routingTableId string, gatewayId string) error {
	cblogger.Info("NHN Cloud Driver: called attachInternetGateway()!")

	if strings.EqualFold(routingTableId, "") {
		newErr := fmt.Errorf("Invalid RoutingTable ID!!")
		cblogger.Error(newErr.Error())
		return newErr
	}

	if strings.EqualFold(gatewayId, "") {
		newErr := fmt.Errorf("Invalid Gateway ID!!")
		cblogger.Error(newErr.Error())
		return newErr
	}

	// First, verify the routing table exists
	cblogger.Infof("Verifying routing table %s exists...", routingTableId)
	getResult := rtable.Get(vpcHandler.NetworkClient, routingTableId)
	if getResult.Err != nil {
		newErr := fmt.Errorf("Routing table not found: %v", getResult.Err)
		cblogger.Error(newErr.Error())
		return newErr
	}
	originalRT, err := getResult.Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to extract routing table: %v", err)
		cblogger.Error(newErr.Error())
		return newErr
	}
	cblogger.Infof("Found routing table: %s (Default: %t, Distributed: %t)", originalRT.Name, originalRT.DefaultTable, originalRT.Distributed)

	// Check if gateway is already attached
	if originalRT.GatewayID != "" {
		cblogger.Infof("Warning: Gateway %s is already attached to this routing table", originalRT.GatewayID)
		if originalRT.GatewayID == gatewayId {
			cblogger.Info("Gateway is already the same one we want to attach. Skipping...")
			return nil
		}
		cblogger.Info("Proceeding to replace the existing gateway...")
	}

	// Create attach gateway options
	attachOpts := rtable.AttachGatewayOpts{
		GatewayID: gatewayId,
	}

	// Attach the gateway
	cblogger.Infof("Attaching gateway %s to routing table %s", gatewayId, routingTableId)
	result := rtable.AttachGateway(vpcHandler.NetworkClient, routingTableId, attachOpts)
	if result.Err != nil {
		// Handle different types of errors
		if nhnstack.ResponseCodeIs(result.Err, 404) {
			newErr := fmt.Errorf("Routing table or gateway not found: %v", result.Err)
			cblogger.Error(newErr.Error())
			return newErr
		} else if nhnstack.ResponseCodeIs(result.Err, 400) {
			newErr := fmt.Errorf("Bad request - check gateway ID and routing table compatibility: %v", result.Err)
			cblogger.Error(newErr.Error())
			return newErr
		} else if nhnstack.ResponseCodeIs(result.Err, 409) {
			newErr := fmt.Errorf("Conflict - gateway may already be in use: %v", result.Err)
			cblogger.Error(newErr.Error())
			return newErr
		} else {
			newErr := fmt.Errorf("Failed to attach gateway: %v", result.Err)
			cblogger.Error(newErr.Error())
			return newErr
		}
	}

	// Extract and validate the result
	updatedRT, err := result.Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to extract updated routing table: %v", err)
		cblogger.Error(newErr.Error())
		return newErr
	}

	// Verify the attachment was successful
	if updatedRT.GatewayID != gatewayId {
		newErr := fmt.Errorf("Gateway attachment failed - expected %s, got %s", gatewayId, updatedRT.GatewayID)
		cblogger.Error(newErr.Error())
		return newErr
	}
	cblogger.Info("✓ Gateway attached successfully!")

	// Display the updated routes (if any)
	if len(updatedRT.Routes) > 0 {
		cblogger.Info("  Updated routes:")
		for _, route := range updatedRT.Routes {
			if route.GatewayID == gatewayId {
				cblogger.Infof("    - %s -> %s (via gateway %s)",
					route.CIDR, route.Gateway, route.GatewayID)
			} else {
				cblogger.Infof("    - %s -> %s", route.CIDR, route.Gateway)
			}
		}
	}

	return nil
}

func (vpcHandler *NhnCloudVPCHandler) removeInternetGateway(vpcId string) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called removeInternetGateway()!")

	if strings.EqualFold(vpcId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	vpcIID := irs.IID{SystemId: vpcId}
	vpc, getErr := vpcHandler.getRawVPC(vpcIID)
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Get the VPC : [%v]", getErr)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	var defaultRoutingTableId string
	if len(vpc.RoutingTables) > 0 {
		for _, routingTable := range vpc.RoutingTables {
			if routingTable.DefaultTable {
				defaultRoutingTableId = routingTable.ID
				cblogger.Infof("### Default RoutingTable : [%s]", defaultRoutingTableId)
				break
			}
		}
	} else {
		newErr := fmt.Errorf("### No Default Routinng Rable on the VPC. [%s]", vpcId)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	dettachErr := vpcHandler.detachGateway(defaultRoutingTableId)
	if dettachErr != nil {
		newErr := fmt.Errorf("Failed to Get the VPC : [%v]", dettachErr)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	var iGWId string
	if len(vpc.RoutingTables) > 0 {
		for _, routingTable := range vpc.RoutingTables {
			if routingTable.DefaultTable {
				iGWId = routingTable.GatewayID
				break
			}
		}
	} else {
		newErr := fmt.Errorf("### No Getway ID according to the Routinng Rable on the VPC. [%s]", vpcId)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	if strings.EqualFold(iGWId, "") {
		newErr := fmt.Errorf("Failed to Get the Internet Gateway ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	deleteErr := vpcHandler.deleteInternetGateway(iGWId)
	if deleteErr != nil {
		newErr := fmt.Errorf("Failed to Get the VPC : [%v]", deleteErr)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	return true, nil
}

func (vpcHandler *NhnCloudVPCHandler) detachGateway(routingTableId string) error {
	cblogger.Info("NHN Cloud Driver: called detachGateway()!")

	if strings.EqualFold(routingTableId, "") {
		newErr := fmt.Errorf("Invalid RoutingTable ID!!")
		cblogger.Error(newErr.Error())
		return newErr
	}

	// First, verify the routing table exists and has a gateway attached
	cblogger.Infof("Verifying routing table %s exists and has gateway attached...", routingTableId)
	getResult := rtable.Get(vpcHandler.NetworkClient, routingTableId)
	if getResult.Err != nil {
		newErr := fmt.Errorf("Routing table not found: %v", getResult.Err)
		cblogger.Error(newErr.Error())
		return newErr
	}

	originalRT, err := getResult.Extract()
	if err != nil {
		cblogger.Errorf("Failed to extract routing table: %v", err)
	}

	cblogger.Infof("Found routing table: %s (Default: %t, Distributed: %t)",
		originalRT.Name, originalRT.DefaultTable, originalRT.Distributed)

	// Check if a gateway is currently attached
	if originalRT.GatewayID == "" {
		cblogger.Info("Warning: No gateway currently attached to this routing table")
		cblogger.Info("Detach operation may not be necessary, but proceeding anyway...")
	} else {
		cblogger.Infof("Current gateway: %s (%s)", originalRT.GatewayName, originalRT.GatewayID)
		cblogger.Info("Proceeding with detach operation...")
	}

	// Store original gateway info for potential rollback
	originalGatewayID := originalRT.GatewayID
	originalGatewayName := originalRT.GatewayName

	// Detach the gateway
	cblogger.Infof("Detaching gateway from routing table %s...", routingTableId)
	result := rtable.DetachGateway(vpcHandler.NetworkClient, routingTableId)
	if result.Err != nil {
		// Handle different types of errors
		if nhnstack.ResponseCodeIs(result.Err, 404) {
			newErr := fmt.Errorf("Routing table not found: %v", result.Err)
			cblogger.Error(newErr.Error())
			return newErr
		} else if nhnstack.ResponseCodeIs(result.Err, 400) {
			newErr := fmt.Errorf("Bad request - routing table may not have a gateway attached: %v", result.Err)
			cblogger.Error(newErr.Error())
			return newErr
		} else if nhnstack.ResponseCodeIs(result.Err, 409) {
			newErr := fmt.Errorf("Conflict - gateway may be in use by other resources: %v", result.Err)
			cblogger.Error(newErr.Error())
			return newErr
		} else if nhnstack.ResponseCodeIs(result.Err, 422) {
			newErr := fmt.Errorf("Unprocessable entity - operation not allowed on this routing table: %v", result.Err)
			cblogger.Error(newErr.Error())
			return newErr
		} else {
			newErr := fmt.Errorf("Failed to detach gateway: %v", result.Err)
			cblogger.Error(newErr.Error())
			return newErr
		}
	}

	// Extract and validate the result
	updatedRT, err := result.Extract()
	if err != nil {
		cblogger.Errorf("Failed to extract updated routing table: %v", err)
	}

	// Verify the detachment was successful
	if updatedRT.GatewayID != "" {
		cblogger.Debugf("Warning: Gateway detachment may not be complete - Gateway ID still present: %s", updatedRT.GatewayID)
	}

	cblogger.Info("✓ Gateway detached successfully!")
	// cblogger.Infof("  Routing Table: %s", updatedRT.Name)
	// cblogger.Infof("  Gateway ID: %s (should be empty)", updatedRT.GatewayID)
	// cblogger.Infof("  Gateway Name: %s (should be empty)", updatedRT.GatewayName)

	// Display the updated routes (gateway routes should be removed)
	if len(updatedRT.Routes) > 0 {
		cblogger.Info("  Remaining routes after gateway detach:")
		for _, route := range updatedRT.Routes {
			if route.GatewayID != "" {
				cblogger.Infof("    ⚠ Route still references gateway: %s -> %s (Gateway: %s)",
					route.CIDR, route.Gateway, route.GatewayID)
			} else {
				cblogger.Infof("    - %s -> %s", route.CIDR, route.Gateway)
			}
		}
	} else {
		cblogger.Info("  No routes remaining after gateway detach")
	}

	// Store detach info for potential rollback demonstration
	cblogger.Infof("Original gateway info stored for potential rollback:")
	cblogger.Infof("  Detached Gateway ID: %s", originalGatewayID)
	cblogger.Infof("  Detached Gateway Name: %s", originalGatewayName)

	return nil
}

// DeleteGatewayRequest represents a request to delete an Internet Gateway
type DeleteGatewayRequest struct {
	GatewayID         string
	WaitForCompletion bool
	Timeout           time.Duration
	CheckInterval     time.Duration
	ForceDelete       bool
}

// DeleteGatewayResult represents the result of deleting an Internet Gateway
type DeleteGatewayResult struct {
	Success     bool
	Error       error
	TimeElapsed time.Duration
	WasAttached bool
}

// delete with validation and waiting
func (vpcHandler *NhnCloudVPCHandler) deleteInternetGateway(gatewayId string) error {
	cblogger.Info("NHN Cloud Driver: called deleteInternetGateway()!")

	if strings.EqualFold(gatewayId, "") {
		newErr := fmt.Errorf("Invalid Internet Gateway ID!!")
		cblogger.Error(newErr.Error())
		return newErr
	}

	request := DeleteGatewayRequest{
		GatewayID:         gatewayId,
		WaitForCompletion: true,
		Timeout:           10 * time.Minute,
		CheckInterval:     3 * time.Second,
		ForceDelete:       false,
	}

	result := vpcHandler.deleteGatewayWithValidation(request)
	if result.Success {
		cblogger.Infof("✅ Gateway deletion completed:")
		cblogger.Infof("  Time elapsed: %v", result.TimeElapsed)
		if result.WasAttached {
			cblogger.Infof("  Note: Gateway was detached from routing table before deletion")
		}
	} else {
		newErr := fmt.Errorf("❌ Gateway deletion failed: %v", result.Error)
		cblogger.Error(newErr.Error())
		return newErr
	}

	return nil
}

// deleteGatewayWithValidation performs deletion with validation and optional waiting
func (vpcHandler *NhnCloudVPCHandler) deleteGatewayWithValidation(request DeleteGatewayRequest) DeleteGatewayResult {
	cblogger.Info("NHN Cloud Driver: called deleteGatewayWithValidation()!")

	startTime := time.Now()

	// Step 1: Check if gateway exists
	cblogger.Infof("🔍 Validating gateway %s exists...", request.GatewayID)

	getResult := igw.Get(vpcHandler.NetworkClient, request.GatewayID)
	gateway, err := getResult.Extract()
	if err != nil {
		if nhnstack.ResponseCodeIs(err, 404) {
			return DeleteGatewayResult{
				Success:     true, // Gateway doesn't exist, consider it successfully "deleted"
				Error:       nil,
				TimeElapsed: time.Since(startTime),
				WasAttached: false,
			}
		}
		return DeleteGatewayResult{
			Success:     false,
			Error:       fmt.Errorf("failed to validate gateway existence: %w", err),
			TimeElapsed: time.Since(startTime),
			WasAttached: false,
		}
	}
	cblogger.Infof("  Gateway found - State: %s", gateway.State)

	// Step 2: Check if gateway is attached to routing table
	wasAttached := gateway.RoutingTableID != nil
	if wasAttached && !request.ForceDelete {
		return DeleteGatewayResult{
			Success:     false,
			Error:       fmt.Errorf("gateway is attached to routing table %s - use force delete or detach first", *gateway.RoutingTableID),
			TimeElapsed: time.Since(startTime),
			WasAttached: true,
		}
	}

	if wasAttached {
		cblogger.Infof("  ⚠️  Gateway is attached to routing table: %s", *gateway.RoutingTableID)
	}

	// Step 3: Check gateway state
	if gateway.State == string(igw.StateError) {
		cblogger.Infof("  ⚠️  Gateway is in error state, proceeding with deletion")
	} else if gateway.State == string(igw.StateMigrating) {
		cblogger.Infof("  ⚠️  Gateway is migrating, deletion may fail")
	}

	// Step 4: Perform deletion
	cblogger.Infof("🔄 Proceeding with deletion...")

	deleteResult := igw.Delete(vpcHandler.NetworkClient, request.GatewayID)
	if deleteResult.Err != nil {
		return DeleteGatewayResult{
			Success:     false,
			Error:       fmt.Errorf("delete operation failed: %w", deleteResult.Err),
			TimeElapsed: time.Since(startTime),
			WasAttached: wasAttached,
		}
	}

	// Step 5: Wait for completion if requested
	if request.WaitForCompletion {
		cblogger.Infof("🔄 Waiting for deletion to complete...")

		err = vpcHandler.waitForGatewayDeletion(request.GatewayID, request.Timeout, request.CheckInterval)
		if err != nil {
			return DeleteGatewayResult{
				Success:     false,
				Error:       fmt.Errorf("deletion initiated but wait failed: %w", err),
				TimeElapsed: time.Since(startTime),
				WasAttached: wasAttached,
			}
		}
	}

	return DeleteGatewayResult{
		Success:     true,
		Error:       nil,
		TimeElapsed: time.Since(startTime),
		WasAttached: wasAttached,
	}
}

// waitForGatewayDeletion waits for a gateway to be completely deleted
func (vpcHandler *NhnCloudVPCHandler) waitForGatewayDeletion(gatewayId string, timeout, checkInterval time.Duration) error {

	if strings.EqualFold(gatewayId, "") {
		newErr := fmt.Errorf("Invalid InternetGateway ID!!")
		cblogger.Error(newErr.Error())
		return newErr
	}

	if timeout == 0 {
		newErr := fmt.Errorf("Invalid Timeout Value!!")
		cblogger.Error(newErr.Error())
		return newErr
	}

	if checkInterval == 0 {
		newErr := fmt.Errorf("Invalid Check Interval Value!!")
		cblogger.Error(newErr.Error())
		return newErr
	}

	startTime := time.Now()
	cblogger.Infof("  ⏳ Waiting for gateway deletion to complete...")

	for {
		elapsed := time.Since(startTime)

		// Check timeout
		if elapsed > timeout {
			return fmt.Errorf("timeout after %v waiting for gateway deletion", elapsed.Round(time.Second))
		}

		// Try to get the gateway
		getResult := igw.Get(vpcHandler.NetworkClient, gatewayId)
		_, err := getResult.Extract()

		if err != nil {
			// If 404, gateway is deleted
			if nhnstack.ResponseCodeIs(err, 404) {
				cblogger.Infof("  ✅ Gateway deletion verified after %v", elapsed.Round(time.Second))
				return nil
			} else if strings.Contains(err.Error(), "Not found internetgateway") {
				cblogger.Infof("  ✅ Gateway deletion verified after %v", elapsed.Round(time.Second))
				return nil
			}

			// Other errors might indicate temporary issues
			cblogger.Infof("  [%v] Error checking deletion status: %v", elapsed.Round(time.Second), err)
		} else {
			// Gateway still exists
			cblogger.Infof("  [%v] Gateway still exists, continuing to wait...", elapsed.Round(time.Second))
		}

		time.Sleep(checkInterval)
	}
}
