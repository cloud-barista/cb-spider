// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI 2022.08.
// Updated by ETRI, 2025.02.
// Updated by ETRI, 2025.07.

package resources

import (
	// "errors"
	"fmt"
	"strings"
	"time"
	// "github.com/davecgh/go-spew/spew"

	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"
	external "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/extensions/external"
	networks "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/networks"
	subnets "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/subnets"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/pagination"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type KTVpcVPCHandler struct {
	RegionInfo    idrv.RegionInfo
	NetworkClient *ktvpcsdk.ServiceClient
}

type NetworkWithExt struct {
	networks.VPC
	external.NetworkExternalExt
}

// KT Cloud (D platform) VPC Open API doc. : https://cloud.kt.com/docs/open-api-guide/d/computing/networking
// KT Cloud (D platform) Tier Open API doc. : https://cloud.kt.com/docs/open-api-guide/d/computing/tier

func (vpcHandler *KTVpcVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcReqInfo.IId.NameId, "CreateVPC()")

	if strings.EqualFold(vpcReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid VPC NameId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

	// Get VPC list
	// Note) KT Cloud (D platform) supports only one VPC that has been created.
	start := call.Start()
	vpcList, err := vpcHandler.listVPC()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VPC list!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}
	loggingInfo(callLogInfo, start)

	if len(vpcList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any VPC Info.")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	if strings.EqualFold(vpcList[0].VpcID, "") {
		cblogger.Error("Failed to Create the Required VPC!!")
		return irs.VPCInfo{}, nil
	}

	// Create the Requested Subnets
	for _, subnetReqInfo := range vpcReqInfo.SubnetInfoList {
		_, err := vpcHandler.createSubnet(&subnetReqInfo)
		if err != nil {
			newErr := fmt.Errorf("Failed to Create New Subnet : [%v]", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return irs.VPCInfo{}, newErr
		}
	}

	vpcInfo, getErr := vpcHandler.GetVPC(irs.IID{SystemId: vpcList[0].VpcID})
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Get the VPC Info : [%v]", getErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	} else {
		// Specify the VPC name
		vpcInfo.IId.NameId = vpcReqInfo.IId.NameId // Caution!! For IID2 NameID validation check for VPC
	}
	return vpcInfo, nil
}

func (vpcHandler *KTVpcVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcIID.SystemId, "GetVPC()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

    vpc, err := networks.Get(vpcHandler.NetworkClient, vpcIID.SystemId).ExtractVPC()
    if err != nil {
		newErr := fmt.Errorf("Failed to Get the VPC info.")
		cblogger.Error(newErr.Error())
        return irs.VPCInfo{}, newErr
    }

	vpcInfo, mapErr := vpcHandler.mappingVpcInfo(vpc)
	if mapErr != nil {
		newErr := fmt.Errorf("Failed to Map the VPC Info : [%v]", mapErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	return *vpcInfo, nil
}

func (vpcHandler *KTVpcVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, "ListVPC()", "ListVPC()")

	// ### If enter a different number to ListOpts, the value will not be retrieved correctly.
    listOpts := networks.ListOpts{
        Page: 1,
        Size: 20,    
	}
	start := call.Start()
    pager := networks.List(vpcHandler.NetworkClient, listOpts)
    loggingInfo(callLogInfo, start)

	var vpcInfoList []*irs.VPCInfo
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
        vpcs, err := networks.ExtractVPCs(page)
        if err != nil {
			newErr := fmt.Errorf("Failed to Extract VPC list : [%v]", err)
			cblogger.Error(newErr.Error())
		    return false, newErr
		}

		if len(vpcs) < 1 {
			newErr := fmt.Errorf("Failed to Get Any VPC Info.")
			cblogger.Infof("No VPC found : %v", newErr)
			return false, newErr
		}
    
        for _, vpc := range vpcs {
			vpcInfo, err := vpcHandler.mappingVpcInfo(&vpc)
			if err != nil {
				newErr := fmt.Errorf("Failed to Map the VPC Info : [%v]", err)
				cblogger.Error(newErr.Error())
				loggingError(callLogInfo, newErr)
				return false, newErr
			}
			vpcInfoList = append(vpcInfoList, vpcInfo)
		}
 		return true, nil
	})
    if err != nil {
        if err != nil {
			newErr := fmt.Errorf("Failed to Get VPC list : [%v]", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
		    return nil, newErr
		}
    }    
 	return vpcInfoList, nil
}

// Note) KT Cloud (D platform) supports only one VPC that has been created.
func (vpcHandler *KTVpcVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called DeleteVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcIID.SystemId, "DeleteVPC()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	// Check whether the VPC exists.
	vpcInfo, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		cblogger.Errorf("Failed to Find the VPC with the SystemID. : [%s] : [%v]", vpcIID.SystemId, err)
		loggingError(callLogInfo, err)
		return false, err
	}

	// Delete the Subnets belonged in the VPC
	for _, subnetInfo := range vpcInfo.SubnetInfoList {
		if !strings.EqualFold(subnetInfo.IId.NameId, "Private") && !strings.EqualFold(subnetInfo.IId.NameId, "DMZ") && !strings.EqualFold(subnetInfo.IId.NameId, "external") && !strings.EqualFold(subnetInfo.IId.NameId, "NLB-SUBNET") {
			_, err := vpcHandler.RemoveSubnet(irs.IID{SystemId: vpcIID.SystemId}, irs.IID{SystemId: subnetInfo.IId.SystemId})
			if (err != nil) && !strings.Contains(err.Error(), ":true") { // Cauton!! : Abnormal Error when removing a subnet on D1 Platform
				newErr := fmt.Errorf("Failed to Delete the Subnet : [%v]", err)
				cblogger.Error(newErr.Error())
				loggingError(callLogInfo, newErr)
				return false, newErr
			}
		}
	}

	result, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		cblogger.Errorf("Failed to Find the VPC with the SystemID. : [%s] : [%v]", vpcIID.SystemId, err)
		loggingError(callLogInfo, err)
		return false, err
	} else {
		cblogger.Infof("Succeeded in Deleting the VPC : " + result.IId.SystemId)
	}

	return true, nil
}

func (vpcHandler *KTVpcVPCHandler) AddSubnet(vpcIID irs.IID, subnetReqInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	cblogger.Info("KT Cloud VPC driver: called AddSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, subnetReqInfo.IId.NameId, "AddSubnet()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	if strings.EqualFold(subnetReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid Sunbet NameId!!")
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	_, err := vpcHandler.createSubnet(&subnetReqInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Add New Subnet : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

	vpcInfo, err := vpcHandler.GetVPC(irs.IID{SystemId: vpcIID.SystemId})
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VPC Info!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	return vpcInfo, nil
}

// Note) The basic tiers “DMZ” and “Private” cannot be deleted.
func (vpcHandler *KTVpcVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC driver: called RemoveSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, subnetIID.SystemId, "RemoveSubnet()")

	if strings.EqualFold(subnetIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Subnet SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}
	cblogger.Infof("\n# Subnet Id(TierId) to Remove : %s", subnetIID.SystemId)
	
	networkId, err := vpcHandler.getNetworkIdWithTierId(subnetIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Network ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	cblogger.Infof("\n# Subnet(Tier) NetworkId to Remove : %s", *networkId)

	// ### Need NetworkId, not TierId.
	delErr := subnets.Delete(vpcHandler.NetworkClient, *networkId).ExtractErr()
	if err != nil {
		newErr := fmt.Errorf("Failed to Remove the Subnet : [%v]", delErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}
	cblogger.Info("\n### Waiting for Deleting the Subnet!! Subnet NetworkId: %s", *networkId)
	vpcHandler.waitForSubnetDeletion(*networkId)
	if err != nil {
		newErr := fmt.Errorf("Failed to wait for the subnet creation: %v", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	return true, nil
}

func (vpcHandler *KTVpcVPCHandler) mappingVpcInfo(nvpc *networks.VPC) (*irs.VPCInfo, error) {
	cblogger.Info("KT Cloud VPC driver: called mappingVpcInfo()!!")
	// cblogger.Info("\n### KTCloud VPC")
	// spew.Dump(nvpc)

	if strings.EqualFold(nvpc.VpcID, "") {
		newErr := fmt.Errorf("Invalid VPC Info!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Mapping VPC info.
	vpcInfo := irs.VPCInfo{
		IId: irs.IID{
			NameId:   	nvpc.Name,
			SystemId: 	nvpc.VpcID,
		},
		IPv4_CIDR: 		"172.25.0.0/12", // VPC CIDR of KT Cloud D Platform default VPC
		KeyValueList:   irs.StructToKeyValueList(nvpc),
	}

	// Get Subnet list
	subnets, err := vpcHandler.listSubnet()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Subnet list!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Get Subnet info list
	var subnetInfoList []irs.SubnetInfo
	for _, subnet := range subnets {
		// if !strings.EqualFold(subnet.RefName, "Private") && !strings.EqualFold(subnet.RefName, "DMZ") && !strings.EqualFold(subnet.RefName, "external") {
		// $$$ When apply filtering
			subnetInfo := vpcHandler.mappingSubnetInfo(*subnet)
			subnetInfoList = append(subnetInfoList, *subnetInfo)
		// }
	}
	vpcInfo.SubnetInfoList = subnetInfoList
	return &vpcInfo, nil
}

func (vpcHandler *KTVpcVPCHandler) mappingSubnetInfo(subnet subnets.Subnet) *irs.SubnetInfo {
	cblogger.Info("KT Cloud VPC driver: called mappingSubnetInfo()!!")

	subnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId:   	subnet.RefName,
			SystemId: 	subnet.RefID, // Caution!! Not 'subnet.NetworkID(Tier UUID)' but 'Tier ID based on OpenStack Neutron' to Create VM!!
		},
		Zone:      		subnet.ZoneID,
		IPv4_CIDR: 		subnet.CIDR,
		KeyValueList:   irs.StructToKeyValueList(subnet),
	}

	keyValue := irs.KeyValue{}
	if !strings.EqualFold(subnet.NetworkID, "") {
		keyValue = irs.KeyValue{Key: "TierNetworkID", Value: subnet.NetworkID} // 'Tier UUID' on KT Cloud D platform Consol
	}
	subnetInfo.KeyValueList = append(subnetInfo.KeyValueList, keyValue)
	return &subnetInfo
}

// Create New Subnet (Tire) and Return Tier 'NetworkID'
func (vpcHandler *KTVpcVPCHandler) createSubnet(subnetReqInfo *irs.SubnetInfo) (string, error) {
	cblogger.Info("KT Cloud VPC driver: called createSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, subnetReqInfo.IId.NameId, "createSubnet()")

	if strings.EqualFold(subnetReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid Sunbet NameId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}

	// KT Cloud D1 platform API guide - Tier : https://cloud.kt.com/docs/open-api-guide/d/computing/tier
	cidrBlock := strings.Split(subnetReqInfo.IPv4_CIDR, ".")
	vmStartIP := cidrBlock[0] + "." + cidrBlock[1] + "." + cidrBlock[2] + "." + "6"
	vmEndIP := cidrBlock[0] + "." + cidrBlock[1] + "." + cidrBlock[2] + "." + "180"
	lbStartIP := cidrBlock[0] + "." + cidrBlock[1] + "." + cidrBlock[2] + "." + "181"
	lbEndIP := cidrBlock[0] + "." + cidrBlock[1] + "." + cidrBlock[2] + "." + "199"
	bmStartIP := cidrBlock[0] + "." + cidrBlock[1] + "." + cidrBlock[2] + "." + "201"
	bmEndIP := cidrBlock[0] + "." + cidrBlock[1] + "." + cidrBlock[2] + "." + "250"
	gatewayIP := cidrBlock[0] + "." + cidrBlock[1] + "." + cidrBlock[2] + "." + "1"

	detailTierInfo := subnets.SubnetDetail{
		CIDR:      	subnetReqInfo.IPv4_CIDR,
		StartIP:   	vmStartIP, // For VM
		EndIP:     	vmEndIP,
		LBStartIP: 	lbStartIP, // For NLB
		LBEndIP:   	lbEndIP,
		BMStartIP: 	bmStartIP, // For BareMetal Machine
		BMEndIP:   	bmEndIP,
		GatewayIP:  gatewayIP,
	}

	// Create Subnet (No Zone info)
	createOpts := subnets.CreateOpts{
		Name:       subnetReqInfo.IId.NameId,   // Required
		Type:       "tier",                     // Required
		IsCustom: 	true,                       // Required		
		Detail:     detailTierInfo,
	}
	// cblogger.Info("\n### Subnet createOpts : ")
	// spew.Dump(createOpts)
	// cblogger.Info("\n")

	cblogger.Info("\n### Adding New Subnet Now!!")
	start := call.Start()
	result, err := subnets.Create(vpcHandler.NetworkClient, createOpts).ExtractCreate()
	if err != nil {
		if !strings.Contains(err.Error(), ":true") {
			newErr := fmt.Errorf("Failed to create Subnet: %v", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return "", newErr
		}
	} else if strings.EqualFold(result.Data.NetworkID, "") {
		newErr := fmt.Errorf("Failed to create Subnet")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	} else {
		cblogger.Info("\n### Waiting for Creating the Subnet!! Subnet NetworkId: %s", result.Data.NetworkID)
		vpcHandler.waitForSubnetActive(result.Data.NetworkID)
		if err != nil {
			newErr := fmt.Errorf("Failed to wait for the subnet creation: %v", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return "", newErr
		}
	}
	loggingInfo(callLogInfo, start)

	return result.Data.NetworkID, nil
}

func (vpcHandler *KTVpcVPCHandler) getKtCloudVpc(vpcId string) (*networks.VPC, error) {
	cblogger.Info("KT Cloud VPC Driver: called getKtCloudVpc()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcId, "getKtCloudVpc()")

	if strings.EqualFold(vpcId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	start := call.Start()
    vpc, err := networks.Get(vpcHandler.NetworkClient, vpcId).ExtractVPC()
    if err != nil {
		newErr := fmt.Errorf("Failed to Get the VPC info.")
		cblogger.Error(newErr.Error())
        return nil, newErr
    }
	loggingInfo(callLogInfo, start)

	return vpc, nil
}

// getSubnet retrieves info of a specific subnet by its 'NetworkId'.
func (vpcHandler *KTVpcVPCHandler) getSubnet(networkId string) (*subnets.Subnet, error) {
	cblogger.Info("KT Cloud VPC Driver: called getSubnet()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, networkId, "getSubnet()")

	if strings.EqualFold(networkId, "") {
		newErr := fmt.Errorf("Invalid Subnet NetworkId!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	start := call.Start()
	subnet, err := subnets.Get(vpcHandler.NetworkClient, networkId).ExtractSubnet()
	if err != nil {
		cblogger.Errorf("Failed to Get the Subnet info with NetworkId [%s] : %v", networkId, err)
		loggingError(callLogInfo, err)
		return nil, nil
	}
	loggingInfo(callLogInfo, start)

	return subnet, nil
}

func (vpcHandler *KTVpcVPCHandler) listVPC() ([]*networks.VPC, error) {
	cblogger.Info("KT Cloud VPC Driver: called listVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, "listVPC()", "listVPC()")

	// ### If enter a different number to ListOpts, the value will not be retrieved correctly.
    listOpts := networks.ListOpts{
        Page: 1,
        Size: 20,    
	}
	start := call.Start()
    pager := networks.List(vpcHandler.NetworkClient, listOpts)
    loggingInfo(callLogInfo, start)

	var vpcAdrsList []*networks.VPC
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
        vpcList, err := networks.ExtractVPCs(page)
        if err != nil {
			newErr := fmt.Errorf("Failed to Extract VPC list : [%v]", err)
			cblogger.Error(newErr.Error())
		    return false, newErr
		}
		if len(vpcList) < 1 {
			newErr := fmt.Errorf("Failed to Get Any VPC Info.")
			cblogger.Infof("No VPC found : %v", newErr)
			return false, newErr
		}
		for _, vpc := range vpcList {
			vpcAdrsList = append(vpcAdrsList, &vpc)
		}

 		return true, nil
	})
    if err != nil {
        if err != nil {
			newErr := fmt.Errorf("Failed to Get VPC list : [%v]", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
		    return nil, newErr
		}
    }

 	return vpcAdrsList, nil
}

func (vpcHandler *KTVpcVPCHandler) listSubnet() ([]*subnets.Subnet, error) {
	cblogger.Info("KT Cloud VPC Driver: called listSubnet()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, "listSubnet()", "listSubnet()")

	// ### If enter a different number to ListOpts, the value will not be retrieved correctly.
    listOpts := subnets.ListOpts{
        Page: 1,
        Size: 20,
	}
	start := call.Start()
    pager := subnets.List(vpcHandler.NetworkClient, listOpts)
    loggingInfo(callLogInfo, start)
	
	var subnetAdrsList []*subnets.Subnet
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
        subnetlist, err := subnets.ExtractSubnets(page)
        if err != nil {
			newErr := fmt.Errorf("Failed to Extract Subnet list : [%v]", err)
			cblogger.Error(newErr.Error())
		    return false, newErr
		}
		if len(subnetlist) < 1 {
			newErr := fmt.Errorf("Failed to Get Any Subnet Info.")
			cblogger.Infof("No Subent found : %v", newErr)
			return false, newErr
		}
		for _, subnet := range subnetlist {
			subnetAdrsList = append(subnetAdrsList, &subnet)
		}
    
 		return true, nil
	})
    if err != nil {
        if err != nil {
			newErr := fmt.Errorf("Failed to Get Subnet list : [%v]", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
		    return nil, newErr
		}
    } 
 	return subnetAdrsList, nil
}

func (vpcHandler *KTVpcVPCHandler) getExtSubnetId() (*string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getExtSubnetId()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, "getExtSubnetId()", "getExtSubnetId()")

	// Get Subnet list
	start := call.Start()
	subnets, err := vpcHandler.listSubnet()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Subnet list!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	loggingInfo(callLogInfo, start)

	var extSubnetId string
	for _, subnet := range subnets {
		if strings.EqualFold(subnet.RefName, "external") {
			extSubnetId = subnet.NetworkID
			break
		}
	}

	if strings.EqualFold(extSubnetId, "") {
		newErr := fmt.Errorf("Failed to Find the External Subnet ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	return &extSubnetId, nil
}

func (vpcHandler *KTVpcVPCHandler) getTierIdWithNetworkId(networkId string) (*string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getTierIdWithNetworkId()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, networkId, "getTierIdWithNetworkId()")

	if strings.EqualFold(networkId, "") {
		newErr := fmt.Errorf("Invalid Subnet(Tier) Network ID!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	// ### If enter a different number to ListOpts, the value will not be retrieved correctly.
	listOpts := subnets.ListOpts{
		Page: 1,
		Size: 20,
		NetworkID: networkId, // Tier NetworkId
	}
	start := call.Start()
	pager := subnets.List(vpcHandler.NetworkClient, listOpts)
	loggingInfo(callLogInfo, start)
	
	var subnetAdrsList []*subnets.Subnet
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		subnetlist, err := subnets.ExtractSubnets(page)
		if err != nil {
			newErr := fmt.Errorf("Failed to Extract Subnet list : [%v]", err)
			cblogger.Error(newErr.Error())
			return false, newErr
		}
		if len(subnetlist) < 1 {
			newErr := fmt.Errorf("Failed to Get Any Subnet Info.")
			cblogger.Infof("No Subent found : %v", newErr)
			return false, newErr
		}
		for _, subnet := range subnetlist {
			subnetAdrsList = append(subnetAdrsList, &subnet)
		}
	
		return true, nil
	})
	if err != nil {
		if err != nil {
			newErr := fmt.Errorf("Failed to Get Subnet list : [%v]", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return nil, newErr
		}
	} 

	// Caution!!
	if len(subnetAdrsList) == 0 || strings.EqualFold(subnetAdrsList[0].RefID, "") {
		newErr := fmt.Errorf("No TierId found with the NetworkId")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	return &subnetAdrsList[0].RefID, nil
}

func (vpcHandler *KTVpcVPCHandler) getNetworkIdWithTierId(tierId string) (*string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getNetworkIdWithTierId()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, tierId, "getNetworkIdWithTierId()")

	if strings.EqualFold(tierId, "") {
		newErr := fmt.Errorf("Invalid Subnet(Tier) ID!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	cblogger.Infof("# Subnet(Tier) ID to Find Network ID : %s", tierId)

	// ### If enter a different number to ListOpts, the value will not be retrieved correctly.
	listOpts := subnets.ListOpts{
		Page: 1,
		Size: 20,
		RefID: tierId, // Tier Id
	}
	start := call.Start()
	pager := subnets.List(vpcHandler.NetworkClient, listOpts)
	loggingInfo(callLogInfo, start)
	
	var subnetAdrsList []*subnets.Subnet
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		subnetlist, err := subnets.ExtractSubnets(page)
		if err != nil {
			newErr := fmt.Errorf("Failed to Extract Subnet list : [%v]", err)
			cblogger.Error(newErr.Error())
			return false, newErr
		}
		if len(subnetlist) < 1 {
			newErr := fmt.Errorf("Failed to Get Any Subnet Info.")
			cblogger.Infof("No Subent found : %v", newErr)
			return false, newErr
		}
		for _, subnet := range subnetlist {
			subnetAdrsList = append(subnetAdrsList, &subnet)
		}
	
		return true, nil
	})
	if err != nil {
		if err != nil {
			newErr := fmt.Errorf("Failed to Get Subnet list : [%v]", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return nil, newErr
		}
	}

	// Caution!!
	if len(subnetAdrsList) == 0 || strings.EqualFold(subnetAdrsList[0].NetworkID, "") {
		newErr := fmt.Errorf("No NetworkId found with the TierId")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	return &subnetAdrsList[0].NetworkID, nil
}


func (vpcHandler *KTVpcVPCHandler) getTierIdWithTierName(tierName string) (*string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getTierIdWithTierName()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, tierName, "getTierIdWithTierName()")

	if strings.EqualFold(tierName, "") {
		newErr := fmt.Errorf("Invalid Subnet(Tier) Name!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	// Get Subnet list
	start := call.Start()
	subnets, err := vpcHandler.listSubnet()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Subnet list!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	loggingInfo(callLogInfo, start)

	var tierId string
	for _, subnet := range subnets {
		if strings.EqualFold(subnet.RefName, tierName) {
			tierId = subnet.RefID
			break
		}
	}
	if strings.EqualFold(tierId, "") {
		newErr := fmt.Errorf("Failed to Find the Subnet(Tier) ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	return &tierId, nil
}

func (vpcHandler *KTVpcVPCHandler) getVPCIdWithTierId(tierId string) (*string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getVPCIdWithTierId()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, tierId, "getVPCIdWithTierId()")

	if strings.EqualFold(tierId, "") {
		newErr := fmt.Errorf("Invalid Subnet(Tier) ID!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	cblogger.Infof("# Subnet(Tier) ID to Find Network ID : %s", tierId)

	// ### If enter a different number to ListOpts, the value will not be retrieved correctly.
	listOpts := subnets.ListOpts{
		Page: 1,
		Size: 20,
		RefID: tierId, // Tier Id
	}
	start := call.Start()
	pager := subnets.List(vpcHandler.NetworkClient, listOpts)
	loggingInfo(callLogInfo, start)
	
	var subnetAdrsList []*subnets.Subnet
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		subnetlist, err := subnets.ExtractSubnets(page)
		if err != nil {
			newErr := fmt.Errorf("Failed to Extract Subnet list : [%v]", err)
			cblogger.Error(newErr.Error())
			return false, newErr
		}
		if len(subnetlist) < 1 {
			newErr := fmt.Errorf("Failed to Get Any Subnet Info.")
			cblogger.Infof("No Subent found : %v", newErr)
			return false, newErr
		}
		for _, subnet := range subnetlist {
			subnetAdrsList = append(subnetAdrsList, &subnet)
		}
	
		return true, nil
	})
	if err != nil {
		if err != nil {
			newErr := fmt.Errorf("Failed to Get Subnet list : [%v]", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return nil, newErr
		}
	}

	// Caution!!
	if len(subnetAdrsList) == 0 || strings.EqualFold(subnetAdrsList[0].VpcID, "") {
		newErr := fmt.Errorf("No NetworkId found with the TierId")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	return &subnetAdrsList[0].VpcID, nil
}

func (vpcHandler *KTVpcVPCHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("KT Cloud VPC driver: called ListIID()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, "ListIID()", "ListIID()")

	// ### If enter a different number to ListOpts, the value will not be retrieved correctly.
    listOpts := networks.ListOpts{
        Page: 1,
        Size: 20,    
	}
	start := call.Start()
    pager := networks.List(vpcHandler.NetworkClient, listOpts)
    loggingInfo(callLogInfo, start)

    var iidList []*irs.IID
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
        vpcs, err := networks.ExtractVPCs(page)
        if err != nil {
			newErr := fmt.Errorf("Failed to Extract VPC list : [%v]", err)
			cblogger.Error(newErr.Error())
		    return false, newErr
		}

		if len(vpcs) < 1 {
			newErr := fmt.Errorf("Failed to Get Any VPC Info.")
			cblogger.Infof("No VPC found : %v", newErr)
			return false, newErr
		}
    
        for _, vpc := range vpcs {
			iid := &irs.IID{
				NameId:   vpc.Name,
				SystemId: vpc.VpcID,
			}
			iidList = append(iidList, iid)
		}
 	return true, nil
	})
    if err != nil {
        if err != nil {
			newErr := fmt.Errorf("Failed to Get VPC list : [%v]", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
		    return nil, newErr
		}
    }    
    return iidList, nil
}

// getSubnetStatus retrieves the status of a specific subnet by its 'NetworkId'.
func (vpcHandler *KTVpcVPCHandler) getSubnetStatus(networkId string) (string, error) {

	if strings.EqualFold(networkId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

    result := subnets.Get(vpcHandler.NetworkClient, networkId)
    if result.Err != nil {
        cblogger.Errorf("Failed to Get the subnet info with NetworkId [%s]: %v", networkId, result.Err)
        return "", result.Err
    }    
    subnet, err := result.ExtractSubnet()
    if err != nil {
        cblogger.Errorf("Failed to Extract subnet info: %v", err)
        return "", err
    }    
    if subnet == nil {
        return "UNKNOWN", fmt.Errorf("Subnet with ID [%s] not found", networkId)
    }
    return subnet.Status, nil
}

// waitForSubnetStatus waits for a subnet to reach a specific status.
// This function polls the subnet status at regular intervals until the desired status is reached
// or the maximum number of attempts is exceeded.
func (vpcHandler *KTVpcVPCHandler) waitForSubnetStatus(networkId string, desiredStatus string, maxAttempts int, delaySeconds int) error {
	cblogger.Info("KT Cloud VPC driver: called waitForSubnetStatus()!!")

    cblogger.Infof("\n# Waiting for subnet [%s] to reach status [%s]", networkId, desiredStatus)    
    for attempt := 1; attempt <= maxAttempts; attempt++ {
        status, err := vpcHandler.getSubnetStatus(networkId)
        if err != nil {
            cblogger.Errorf("Error checking subnet status (attempt %d/%d): %v", attempt, maxAttempts, err)
            // Keep trying even if there are errors
            time.Sleep(time.Duration(delaySeconds) * time.Second)
            continue
        }        
        cblogger.Infof("\n# Subnet [%s] status: %s (attempt %d/%d)", networkId, status, attempt, maxAttempts)
        
        if status == desiredStatus {
            cblogger.Infof("\n# Subnet [%s] reached desired status [%s]", networkId, desiredStatus)
            return nil
        }        
        if status == "ERROR" || status == "FAILED" {
            return fmt.Errorf("subnet reached error state: %s", status)
        }
        
        time.Sleep(time.Duration(delaySeconds) * time.Second)
    }
    
    return fmt.Errorf("maximum number of attempts (%d) exceeded waiting for subnet [%s] to reach status [%s]", maxAttempts, networkId, desiredStatus)
}

// isSubnetActive checks if a subnet is in ACTIVE status.
func (vpcHandler *KTVpcVPCHandler) isSubnetActive(networkId string) (bool, error) {
    status, err := vpcHandler.getSubnetStatus(networkId)
    if err != nil {
        return false, err
    }
    return status == "ACTIVE", nil
}

// waitForSubnetActive waits for a subnet to reach ACTIVE status. (20 attempts, 3-second intervals)
func (vpcHandler *KTVpcVPCHandler) waitForSubnetActive(networkId string) error {
	cblogger.Info("KT Cloud VPC driver: called waitForSubnetActive()!!")

    return vpcHandler.waitForSubnetStatus(networkId, "ACTIVE", 20, 3)
}

// isSubnetDeleted checks if a subnet has been successfully deleted.
// It returns true if the subnet can't be found (404 error).
func (vpcHandler *KTVpcVPCHandler) isSubnetDeleted(networkId string) (bool, error) {	
    result := subnets.Get(vpcHandler.NetworkClient, networkId)
    if result.Err != nil {
        // 404 error means that the subnet has been deleted.
        if _, ok := result.Err.(ktvpcsdk.ErrDefault404); ok {
            return true, nil
        }
        return false, result.Err
    }

	return false, nil
}

// waitForSubnetDeletion waits for a subnet to be deleted.
func (vpcHandler *KTVpcVPCHandler) waitForSubnetDeletion(networkId string) error {
	cblogger.Info("KT Cloud VPC driver: called waitForSubnetDeletion()!!")

    maxAttempts := 3
    delaySeconds := 3
    
    for attempt := 1; attempt <= maxAttempts; attempt++ {
        isSubnetActive, err := vpcHandler.isSubnetActive(networkId)
        if err != nil {
            cblogger.Errorf("Error checking subnet deletion (attempt %d/%d): %v", attempt, maxAttempts, err)
            time.Sleep(time.Duration(delaySeconds) * time.Second)
            continue
        }
        if !isSubnetActive {
            cblogger.Infof("Subnet [%s] is Not Active", networkId)
            return nil
        }
        
        cblogger.Infof("\nSubnet [%s] is still being deleted (attempt %d/%d)", networkId, attempt, maxAttempts)
        time.Sleep(time.Duration(delaySeconds) * time.Second)
    }
    
//     for attempt := 1; attempt <= maxAttempts; attempt++ {
//         deleted, err := vpcHandler.isSubnetDeleted(networkId)
//         if err != nil {
//             cblogger.Errorf("Error checking subnet deletion (attempt %d/%d): %v", attempt, maxAttempts, err)
//             time.Sleep(time.Duration(delaySeconds) * time.Second)
//             continue
//         }        
//         if deleted {
//             cblogger.Infof("Subnet [%s] has been deleted", networkId)
//             return nil
//         }
		
//         cblogger.Infof("\nSubnet [%s] is still being deleted (attempt %d/%d)", networkId, attempt, maxAttempts)
//         time.Sleep(time.Duration(delaySeconds) * time.Second)
//     }

    return fmt.Errorf("maximum number of attempts (%d) exceeded waiting for subnet [%s] to be deleted", maxAttempts, networkId)
}
