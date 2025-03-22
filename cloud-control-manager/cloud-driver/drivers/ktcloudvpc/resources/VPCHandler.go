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

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type KTVpcVPCHandler struct {
	RegionInfo    idrv.RegionInfo
	NetworkClient *ktvpcsdk.ServiceClient
}

type NetworkWithExt struct {
	networks.Network
	external.NetworkExternalExt
}

func (vpcHandler *KTVpcVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcReqInfo.IId.NameId, "CreateVPC()")

	if strings.EqualFold(vpcReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid VPC NameId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

	// KT Cloud (D1) VPC API doc. : https://cloud.kt.com/docs/open-api-guide/d/computing/networking
	// KT Cloud (D1) Tier API doc. : https://cloud.kt.com/docs/open-api-guide/d/computing/tier

	start := call.Start()
	listOpts := networks.ListOpts{}
	firstPage, err := networks.List(vpcHandler.NetworkClient, listOpts).FirstPage() // Caution!! : First Page Only
	if err != nil {
		cblogger.Errorf("Failed to Create KT Cloud VPC : [%v]", err)
		loggingError(callLogInfo, err)
		return irs.VPCInfo{}, err
	}
	loggingInfo(callLogInfo, start)

	vpcList, err := networks.ExtractVPCs(firstPage)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VPC Network list. [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	if len(vpcList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any VPC Network Info.")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	if strings.EqualFold(vpcList[0].ID, "") {
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

	vpcInfo, getErr := vpcHandler.GetVPC(irs.IID{SystemId: vpcList[0].ID})
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Get the VPC Info : [%v]", getErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	} else {
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

	start := call.Start()
	listOpts := networks.ListOpts{}
	firstPage, err := networks.List(vpcHandler.NetworkClient, listOpts).FirstPage() // Caution!! : First Page Only
	if err != nil {
		cblogger.Errorf("Failed to Get VPC Network info from KT Cloud VPC : [%v]", err)
		loggingError(callLogInfo, err)
		return irs.VPCInfo{}, err
	}
	loggingInfo(callLogInfo, start)

	vpcList, err := networks.ExtractVPCs(firstPage)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VPC Network list. [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}
	if len(vpcList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any VPC Network Info.")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VPCInfo{}, newErr
	}

	var vpcInfo *irs.VPCInfo
	for _, vpc := range vpcList {
		if strings.EqualFold(vpcIID.SystemId, vpc.ID) {
			var mapErr error
			vpcInfo, mapErr = vpcHandler.mappingVpcInfo(&vpc)
			if mapErr != nil {
				newErr := fmt.Errorf("Failed to Map the VPC Info : [%v]", mapErr)
				cblogger.Error(newErr.Error())
				loggingError(callLogInfo, newErr)
				return irs.VPCInfo{}, newErr
			}
			break
		}
	}
	return *vpcInfo, nil
}

func (vpcHandler *KTVpcVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListVPC()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, "ListVPC()", "ListVPC()")

	// KT Cloud (D1) VPC API guide : https://cloud.kt.com/docs/open-api-guide/d/computing/networking
	// KT Cloud (D1) Tier API guide : https://cloud.kt.com/docs/open-api-guide/d/computing/tier

	listOpts := networks.ListOpts{}
	start := call.Start()
	firstPage, err := networks.List(vpcHandler.NetworkClient, listOpts).FirstPage() // Caution!! : First Page Only
	if err != nil {
		cblogger.Errorf("Failed to Get VPC Network info from KT Cloud VPC : [%v]", err)
		loggingError(callLogInfo, err)
		return nil, err
	}
	loggingInfo(callLogInfo, start)

	vpcList, err := networks.ExtractVPCs(firstPage)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VPC Network list. [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	if len(vpcList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any VPC Network Info.")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	var vpcInfoList []*irs.VPCInfo
	for _, vpc := range vpcList {
		vpcInfo, err := vpcHandler.mappingVpcInfo(&vpc)
		if err != nil {
			newErr := fmt.Errorf("Failed to Map the VPC Info : [%v]", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return nil, newErr
		}
		vpcInfoList = append(vpcInfoList, vpcInfo)
	}
	return vpcInfoList, nil
}

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

	// Since KT Cloud VPC(D Platform) supports Single VPC
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

func (vpcHandler *KTVpcVPCHandler) GetSubnet(subnetIID irs.IID) (irs.SubnetInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetSubnet()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, subnetIID.SystemId, "GetSubnet()")

	if strings.EqualFold(subnetIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Subnet SystemId!!")
		cblogger.Error(newErr.Error())
		return irs.SubnetInfo{}, newErr
	}

	start := call.Start()
	subnet, err := subnets.Get(vpcHandler.NetworkClient, subnetIID.SystemId).Extract()
	if err != nil {
		cblogger.Errorf("Failed to Get Subnet with SystemId [%s] : %v", subnetIID.SystemId, err)
		loggingError(callLogInfo, err)
		return irs.SubnetInfo{}, nil
	}
	loggingInfo(callLogInfo, start)

	subnetInfo := vpcHandler.mappingSubnetInfo(*subnet)
	return *subnetInfo, nil
}

func (vpcHandler *KTVpcVPCHandler) AddSubnet(vpcIID irs.IID, subnetReqInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	cblogger.Info("KT Cloud VPC driver: called AddSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, subnetReqInfo.IId.NameId, "AddSubnet()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	if subnetReqInfo.IId.NameId == "" {
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

func (vpcHandler *KTVpcVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC driver: called RemoveSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, subnetIID.SystemId, "RemoveSubnet()")

	if strings.EqualFold(subnetIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Subnet SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	err := subnets.Delete(vpcHandler.NetworkClient, subnetIID.SystemId).ExtractErr()
	if err != nil {
		newErr := fmt.Errorf("Failed to Remove the Subnet from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}
	cblogger.Info("\n### Waiting for Deleting the Subnet!!")
	time.Sleep(time.Second * 5)

	return true, nil
}

func (vpcHandler *KTVpcVPCHandler) mappingVpcInfo(nvpc *networks.Network) (*irs.VPCInfo, error) {
	cblogger.Info("KT Cloud VPC driver: called mappingVpcInfo()!!")
	// cblogger.Info("\n### KTCloud VPC")
	// spew.Dump(nvpc)

	if strings.EqualFold(nvpc.ID, "") {
		newErr := fmt.Errorf("Invalid VPC Info!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Mapping VPC info.
	vpcInfo := irs.VPCInfo{
		IId: irs.IID{
			NameId:   	nvpc.Name,
			SystemId: 	nvpc.ID,
		},
		IPv4_CIDR: 		"172.25.0.0/12", // VPC CIDR of KT Cloud D1 Platform Default VPC
		KeyValueList:   irs.StructToKeyValueList(nvpc),
	}

	// Get Subnet info list.
	var subnetInfoList []irs.SubnetInfo
	for _, subnet := range nvpc.Subnets {
		if !strings.EqualFold(subnet.Name, "Private_Sub") && !strings.EqualFold(subnet.Name, "DMZ_Sub") && !strings.EqualFold(subnet.Name, "external") {
			// # When apply filtering

			// cblogger.Infof("# Subnet Name : [%s]", subnet.Name)
			// if strings.EqualFold(subnet.Name, "NLB-SUBNET_Sub"){  // Note) '_Sub' is automatically appended to the original subnet name

			subnetInfo := vpcHandler.mappingSubnetInfo(subnet)
			subnetInfoList = append(subnetInfoList, *subnetInfo)
		}
	}
	vpcInfo.SubnetInfoList = subnetInfoList

	return &vpcInfo, nil
}

func (vpcHandler *KTVpcVPCHandler) mappingSubnetInfo(subnet subnets.Subnet) *irs.SubnetInfo {
	cblogger.Info("KT Cloud VPC driver: called mappingSubnetInfo()!!")

	// To remove "_Sub" in the subnet name (Note. "_Sub" is appended to a subnet name in the KT Cloud)
	// Removing "_Sub" for CB-Spdier IID manager
	subnetName := strings.Split(subnet.Name, "_Sub")
	newName := subnetName[0]

	subnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId:   	newName,
			SystemId: 	subnet.OsNetworkID, // Caution!! Not 'subnet.ID(Tier ID)' but 'subnet.OsNetworkID' to Create VM!!
		},
		Zone:      		subnet.ZoneID,
		IPv4_CIDR: 		subnet.CIDR,
		KeyValueList:   irs.StructToKeyValueList(subnet),
	}

	keyValue := irs.KeyValue{}
	if !strings.EqualFold(subnet.ID, "") {
		keyValue = irs.KeyValue{Key: "TierUUID", Value: subnet.ID} // 'Tier ID' on KT Cloud D platform Consol
	}
	subnetInfo.KeyValueList = append(subnetInfo.KeyValueList, keyValue)
	return &subnetInfo
}

// Create New Subnet (Tire) and Return OS Network ID
func (vpcHandler *KTVpcVPCHandler) createSubnet(subnetReqInfo *irs.SubnetInfo) (string, error) {
	cblogger.Info("KT Cloud VPC driver: called createSubnet()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, subnetReqInfo.IId.NameId, "createSubnet()")

	if subnetReqInfo.IId.NameId == "" {
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

	detailTierInfo := subnets.DetailInfo{
		CIDR:      subnetReqInfo.IPv4_CIDR,
		StartIP:   vmStartIP, // For VM
		EndIP:     vmEndIP,
		LBStartIP: lbStartIP, // For NLB
		LBEndIP:   lbEndIP,
		BMStartIP: bmStartIP, // For BareMetal Machine
		BMEndIP:   bmEndIP,
		Gateway:   gatewayIP,
	}

	// Create Subnet
	createOpts := subnets.CreateOpts{
		Name:       subnetReqInfo.IId.NameId,   // Required
		Zone:       vpcHandler.RegionInfo.Zone, // Required
		Type:       "tier",                     // Required
		UserCustom: "y",                        // Required
		Detail:     detailTierInfo,
	}
	// cblogger.Info("\n### Subnet createOpts : ")
	// spew.Dump(createOpts)
	// cblogger.Info("\n")

	cblogger.Info("\n### Adding New Subnet Now!!")
	start := call.Start()
	result, err := subnets.Create(vpcHandler.NetworkClient, createOpts).ExtractCreateInfo()
	if err != nil {
		if !strings.Contains(err.Error(), ":true") {
			newErr := fmt.Errorf("Failed to Create the Subnet : [%v]", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return "", newErr
		}
	} else if strings.EqualFold(result.NetworkID, "") {
		newErr := fmt.Errorf("Failed to Create the Subnet!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	} else {
		cblogger.Info("\n### Waiting for Creating the Subnet!!")
		time.Sleep(time.Second * 20)
	}
	loggingInfo(callLogInfo, start)

	return result.NetworkID, nil
}

func (vpcHandler *KTVpcVPCHandler) getKtCloudVpc(vpcId string) (*networks.Network, error) {
	cblogger.Info("KT Cloud VPC Driver: called getKtCloudVpc()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcId, "getKtCloudVpc()")

	if strings.EqualFold(vpcId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	listOpts := networks.ListOpts{}
	start := call.Start()
	firstPage, err := networks.List(vpcHandler.NetworkClient, listOpts).FirstPage() // Caution!! : First Page Only
	if err != nil {
		cblogger.Errorf("Failed to Get VPC Network info from KT Cloud VPC : [%v]", err)
		loggingError(callLogInfo, err)
		return nil, err
	}
	loggingInfo(callLogInfo, start)

	vpcList, err := networks.ExtractVPCs(firstPage)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VPC Network list. [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	if len(vpcList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any VPC Network Info.")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	var ktVpc *networks.Network
	for _, vpc := range vpcList {
		if strings.EqualFold(vpcId, vpc.ID) {
			ktVpc = &vpc
			break
		}
	}
	return ktVpc, nil
}

func (vpcHandler *KTVpcVPCHandler) getExtSubnetId(vpcIID irs.IID) (string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getExtSubnetId()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcIID.SystemId, "getExtSubnetId()")

	if strings.EqualFold(vpcIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VPC SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}

	ktVpc, err := vpcHandler.getKtCloudVpc(vpcIID.SystemId)
	if err != nil {
		cblogger.Errorf("Failed to Get the VPC Info from KT Cloud. : [%v]", err)
		loggingError(callLogInfo, err)
		return "", err
	}

	var extSubnetId string
	for _, subnet := range ktVpc.Subnets {
		if strings.EqualFold(subnet.Name, "external") {
			extSubnetId = subnet.ID
			break
		}
	}

	if strings.EqualFold(extSubnetId, "") {
		newErr := fmt.Errorf("Failed to Find the External Subnet ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	return extSubnetId, nil
}

func (vpcHandler *KTVpcVPCHandler) getOsNetworkIdWithTierId(vpcId string, tierId string) (string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getOsNetworkIdWithTierId()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, tierId, "getOsNetworkIdWithTierId()")

	if strings.EqualFold(vpcId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}

	if strings.EqualFold(tierId, "") {
		newErr := fmt.Errorf("Invalid Subnet(Tier) ID!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}

	ktVpc, err := vpcHandler.getKtCloudVpc(vpcId)
	if err != nil {
		cblogger.Errorf("Failed to Get the VPC Info from KT Cloud. [%v]", err)
		loggingError(callLogInfo, err)
		return "", err
	}

	var osNetworkId string
	for _, subnet := range ktVpc.Subnets {
		if strings.EqualFold(subnet.ID, tierId) {
			osNetworkId = subnet.OsNetworkID
			break
		}
	}

	if strings.EqualFold(osNetworkId, "") {
		newErr := fmt.Errorf("Failed to Find the OsNetwork ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	return osNetworkId, nil
}

func (vpcHandler *KTVpcVPCHandler) getTierIdWithOsNetworkId(vpcId string, osNetworkId string) (string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getTierIdWithOsNetworkId()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, osNetworkId, "getTierIdWithOsNetworkId()")

	if strings.EqualFold(vpcId, "") {
		newErr := fmt.Errorf("Invalid VPC ID!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}

	if strings.EqualFold(osNetworkId, "") {
		newErr := fmt.Errorf("Invalid Subnet(Tier) ID!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}

	ktVpc, err := vpcHandler.getKtCloudVpc(vpcId)
	if err != nil {
		cblogger.Errorf("Failed to Get the VPC Info from KT Cloud. [%v]", err)
		loggingError(callLogInfo, err)
		return "", err
	}

	var tierId string
	for _, subnet := range ktVpc.Subnets {
		if strings.EqualFold(subnet.OsNetworkID, osNetworkId) {
			tierId = subnet.ID
			break
		}
	}

	if strings.EqualFold(tierId, "") {
		newErr := fmt.Errorf("Failed to Find the Tier UUID ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	return tierId, nil
}

func (vpcHandler *KTVpcVPCHandler) getOsNetworkIdWithTierName(tierName string) (string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getOsNetworkIdWithTierName()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, tierName, "getOsNetworkIdWithTierName()")

	if strings.EqualFold(tierName, "") {
		newErr := fmt.Errorf("Invalid Subnet(Tier) Name!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}
	cblogger.Infof("# Subnet(Tier) Name to Find OsNetwork ID : %s", tierName)

	start := call.Start()
	listOpts := networks.ListOpts{}
	firstPage, err := networks.List(vpcHandler.NetworkClient, listOpts).FirstPage() // Caution!! : First Page Only
	if err != nil {
		cblogger.Errorf("Failed to Get VPC Network info from KT Cloud VPC : [%v]", err)
		loggingError(callLogInfo, err)
		return "", err
	}
	loggingInfo(callLogInfo, start)

	vpcList, err := networks.ExtractVPCs(firstPage)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VPC Network list. [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}
	if len(vpcList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any VPC Network Info.")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}

	var osNetworkId string
	for _, vpc := range vpcList {
		for _, subnet := range vpc.Subnets {
			// To remove "_Sub" in the subnet name (Note. "_Sub" is appended to a subnet name in the KT Cloud)
			subnetName := strings.Split(subnet.Name, "_Sub") // Caution!!

			if strings.EqualFold(subnetName[0], tierName) {
				osNetworkId = subnet.OsNetworkID
				break
			}
		}
	}

	if strings.EqualFold(osNetworkId, "") {
		newErr := fmt.Errorf("Failed to Find the OsNetwork ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	return osNetworkId, nil
}

func (vpcHandler *KTVpcVPCHandler) getVPCIdWithOsNetworkID(osNetworkId string) (string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getVPCIdWithOsNetworkID()!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, osNetworkId, "getVPCIdWithOsNetworkID()")

	if strings.EqualFold(osNetworkId, "") {
		newErr := fmt.Errorf("Invalid OSNetworkID!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}

	start := call.Start()
	listOpts := networks.ListOpts{}
	firstPage, err := networks.List(vpcHandler.NetworkClient, listOpts).FirstPage() // Caution!! : First Page Only
	if err != nil {
		cblogger.Errorf("Failed to Get VPC Network info from KT Cloud VPC : [%v]", err)
		loggingError(callLogInfo, err)
		return "", err
	}
	loggingInfo(callLogInfo, start)

	vpcList, err := networks.ExtractVPCs(firstPage)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VPC Network list. [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}
	if len(vpcList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any VPC Network Info.")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}

	var vpcId string
	for _, vpc := range vpcList {
		for _, subnet := range vpc.Subnets {
			if strings.EqualFold(subnet.OsNetworkID, osNetworkId) {
				vpcId = subnet.VpcID
				break
			}
		}
	}

	if strings.EqualFold(vpcId, "") {
		newErr := fmt.Errorf("Failed to Find the VPC ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	return vpcId, nil
}

// func (vpcHandler *KTVpcVPCHandler) getVpcIdWithName(vpcName string) (string, error) {
// 	cblogger.Info("KT Cloud VPC Driver: called getVpcIdWithName()!")
// 	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, vpcName, "getVpcIdWithName()")

// 	if strings.EqualFold(vpcName, "") {
// 		newErr := fmt.Errorf("Invalid VPC ID!!")
// 		cblogger.Error(newErr.Error())
// 		loggingError(callLogInfo, newErr)
// 		return "", newErr
// 	}

// 	start := call.Start()
// 	listOpts := networks.ListOpts{}
// 	firstPage, err := networks.List(vpcHandler.NetworkClient, listOpts).FirstPage() // Caution!! : First Page Only
// 	if err != nil {
// 		cblogger.Errorf("Failed to Get VPC Network info from KT Cloud VPC : [%v]", err)
// 		loggingError(callLogInfo, err)
// 		return "", err
// 	}
// 	loggingInfo(callLogInfo, start)

// 	vpcList, err := networks.ExtractVPCs(firstPage)
// 	if err != nil {
// 		newErr := fmt.Errorf("Failed to Get KT Cloud VPC Network list. [%v]", err)
// 		cblogger.Error(newErr.Error())
// 		loggingError(callLogInfo, newErr)
// 		return "", newErr
// 	}
// 	if len(vpcList) < 1 {
// 		newErr := fmt.Errorf("Failed to Get Any VPC Network Info.")
// 		cblogger.Error(newErr.Error())
// 		loggingError(callLogInfo, newErr)
// 		return "", newErr
// 	}

// 	var vpcId string
// 	for _, vpc := range vpcList {
// 		if strings.EqualFold(vpc.Name, vpcName) {
// 			vpcId = vpc.ID
// 			break
// 		}
// 	}

// 	if strings.EqualFold(vpcId, "") {
// 		newErr := fmt.Errorf("Failed to Find the VPC ID!!")
// 		cblogger.Error(newErr.Error())
// 		return "", newErr
// 	}

// 	return vpcId, nil
// }

func (vpcHandler *KTVpcVPCHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("KT Cloud VPC driver: called ListIID()!!")
	callLogInfo := getCallLogScheme(vpcHandler.RegionInfo.Zone, call.VPCSUBNET, "ListIID()", "ListIID()")

	start := call.Start()
    listOpts := networks.ListOpts{}
    firstPage, err := networks.List(vpcHandler.NetworkClient, listOpts).FirstPage() // Caution!! : First Page Only
    if err != nil {
        newErr := fmt.Errorf("Failed to Get VPC Network info from KT Cloud VPC : [%v]", err)
        cblogger.Error(newErr.Error())
        return nil, newErr
    }
	loggingInfo(callLogInfo, start)

    vpcList, err := networks.ExtractVPCs(firstPage)
    if err != nil {
        newErr := fmt.Errorf("Failed to Get KT Cloud VPC Network list. [%v]", err)
        cblogger.Error(newErr.Error())
        return nil, newErr
    }
    if len(vpcList) < 1 {
        cblogger.Info("### There is No VPC!!")
        return nil, nil
    }

    var iidList []*irs.IID
    for _, vpc := range vpcList {
        iid := &irs.IID{
            NameId:   vpc.Name,
            SystemId: vpc.ID,
        }
        iidList = append(iidList, iid)
    }
    return iidList, nil
}
