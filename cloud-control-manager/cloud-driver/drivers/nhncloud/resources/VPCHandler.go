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
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"strings"

	// "sync"
	"time"
	// "github.com/davecgh/go-spew/spew"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/extensions/external"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/vpcs"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/vpcsubnets"

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

func (vpcHandler *NhnCloudVPCHandler) getRawVPC(vpcIID irs.IID) (*vpcs.VPC, error) {
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
	cblogger.Info("NHN Cloud Cloud Driver: called GetVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcIID.SystemId, "GetVPC()")

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
	cblogger.Info("NHN Cloud Cloud Driver: called ListVPC()!")
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
	cblogger.Info("NHN Cloud Cloud Driver: called DeleteVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcIID.SystemId, "DeleteVPC()")

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
	cblogger.Info("NHN Cloud cloud driver: called getSubnet()!!")
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
	cblogger.Info("NHN Cloud cloud driver: called AddSubnet()!!")
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
	cblogger.Info("NHN Cloud cloud driver: called RemoveSubnet()!!")
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
	cblogger.Info("NHN Cloud cloud driver: called mappingVpcInfo()!!")
	cblogger.Info("\n\n### vpc : ")
	spew.Dump(vpc)
	cblogger.Info("\n")

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
	cblogger.Info("NHN Cloud cloud driver: called mappingVpcSubnetInfo()!!")

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
	cblogger.Info("NHN Cloud cloud driver: called mappingSubnetInfo()!!")
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
	cblogger.Info("NHN Cloud Cloud Driver: called isConnectedToGateway()!")
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
