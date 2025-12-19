// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI Team, 2024.02.
// Updated by ETRI Team, 2025.11.

package resources

import (
	// "errors"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	// "github.com/davecgh/go-spew/spew"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	nim "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/kt/resources/info_manager/nlb_info_manager"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"
	ktvpclb "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/loadbalancer/v1/loadbalancers"
	rules "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/extensions/fwaas_v2/rules"
	nat "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/extensions/layer3/staticnat"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/pagination"
)

type KTVpcNLBHandler struct {
	RegionInfo    idrv.RegionInfo
	VMClient      *ktvpcsdk.ServiceClient
	NetworkClient *ktvpcsdk.ServiceClient
	NLBClient     *ktvpcsdk.ServiceClient
}

const (
	DefaultNLBOption      string = "roundrobin" // NLBOption : roundrobin / leastconnection / leastresponse / sourceiphash /
	DefaultHealthCheckURL string = "abc.kt.com"
)

func (nlbHandler *KTVpcNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (createNLB irs.NLBInfo, newErr error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", nlbReqInfo.IId.NameId, "CreateNLB()")

	if strings.EqualFold(nlbReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid NLB NameId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	if strings.EqualFold(nlbReqInfo.VpcIID.NameId, "") {
		newErr := fmt.Errorf("Invalid VPC NameId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	if !strings.EqualFold(nlbReqInfo.Listener.Protocol, nlbReqInfo.VMGroup.Protocol) {
		newErr := fmt.Errorf("Listener Protocol and VMGroup Protocol should be the Same for KT Cloud NLB!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	if !strings.EqualFold(nlbReqInfo.Listener.Port, nlbReqInfo.VMGroup.Port) {
		newErr := fmt.Errorf("Listener Port and VMGroup Prot should be the Same for this KT Cloud connection driver!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	ctx := context.Background()

	// Check if NLB with the same name already exists
	exists, checkErr := nlbHandler.checkNLBExists(ctx, nlbReqInfo.IId.NameId)
	if checkErr != nil {
		newErr := fmt.Errorf("failed to check NLB existence: %w", checkErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	if exists {
		newErr := fmt.Errorf("NLB with name [%s] already exists in zone [%s]", nlbReqInfo.IId.NameId, nlbHandler.RegionInfo.Zone)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	/*
		vpcHandler := KTVpcVPCHandler{
			RegionInfo:    nlbHandler.RegionInfo,
			NetworkClient: nlbHandler.NetworkClient, // Required!!
		}
		tierId, getError := vpcHandler.getTierIdWithTierName(NlbSubnetName)
		if getError != nil {
			newErr := fmt.Errorf("Failed to Get the Tier ID of the 'NLB-Subnet' : [%v]", getError)
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		} else {
			cblogger.Infof("# Tier ID of NLB-SUBNET : %s", *tierId)
		}
	*/

	// Get subnet where VMGroup VMs belong
	vmGroupSubnetId, err := nlbHandler.getSubnetOfVMGroup(ctx, nlbReqInfo.VMGroup)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VMGroup Subnet ID : %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	cblogger.Infof("VMGroup Subnet ID: %s", vmGroupSubnetId)

	cblogger.Info("### Start to Create New NLB!!")
	createOpts := ktvpclb.CreateOpts{
		Name:            nlbReqInfo.IId.NameId,             // Required
		ZoneID:          nlbHandler.RegionInfo.Zone,        // Required
		NlbOption:       DefaultNLBOption,                  // Required
		ServiceIP:       "",                                // Required. KT Cloud Virtual IP. $$$ In case of an empty value(""), it is newly created.
		ServicePort:     nlbReqInfo.Listener.Port,          // Required
		ServiceType:     nlbReqInfo.Listener.Protocol,      // Required
		HealthCheckType: nlbReqInfo.HealthChecker.Protocol, // Required
		HealthCheckURL:  DefaultHealthCheckURL,             // URL when the HealthCheckType (above) is 'http' or 'https'.
		NetworkID:       vmGroupSubnetId,                   // Required. Caution!!) Not Tier 'ID' but 'NetworkID' of the Tier!!
	}
	start := call.Start()
	resp, err := ktvpclb.Create(nlbHandler.NLBClient, createOpts).ExtractCreate() // Not 'NetworkClient'
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New NLB : [%w]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	loggingInfo(callLogInfo, start)

	nlbIdStr := strconv.Itoa(resp.Createnlbresponse.NLBId)
	cblogger.Infof("# ID of the NLB being created : %s", nlbIdStr)
	cblogger.Infof("# Full API Response: %+v", resp)

	// Check if API returned an error (check ErrorCode, ErrorText, or DisplayText)
	if resp.Createnlbresponse.ErrorCode != "" || resp.Createnlbresponse.ErrorText != "" ||
		(resp.Createnlbresponse.DisplayText != "" && resp.Createnlbresponse.NLBId == 0) {
		// DisplayText often contains error messages like "internal server error"
		newErr := fmt.Errorf("KT Cloud NLB creation failed - ErrorCode: %s, ErrorText: %s, DisplayText: %s",
			resp.Createnlbresponse.ErrorCode,
			resp.Createnlbresponse.ErrorText,
			resp.Createnlbresponse.DisplayText)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	// Check if NLB creation was successful
	if resp.Createnlbresponse.NLBId == 0 {
		// NLB ID 0 usually indicates creation failure
		cblogger.Warn("Received NLB ID 0 from KT Cloud (no explicit error in response).")

		// Check if VMs were provided
		hasVMs := nlbReqInfo.VMGroup.VMs != nil && len(*nlbReqInfo.VMGroup.VMs) > 0
		if !hasVMs {
			newErr := fmt.Errorf("KT Cloud NLB creation returned ID 0. KT Cloud may require at least one VM to create NLB. Please add VMs to the NLB request and try again")
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return irs.NLBInfo{}, newErr
		}

		// If VMs were provided but still got ID 0, wait and try to find by name
		cblogger.Warn("VMs were provided but received ID 0. Waiting and searching by name...")
		time.Sleep(time.Second * 15)

		foundNLB, findErr := nlbHandler.getKtNlbWithName(nlbReqInfo.IId.NameId)
		if findErr != nil {
			newErr := fmt.Errorf("NLB creation failed. Could not find NLB with name %s: %w", nlbReqInfo.IId.NameId, findErr)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return irs.NLBInfo{}, newErr
		}

		nlbIdStr = strconv.Itoa(foundNLB.NlbID)
		cblogger.Infof("Found NLB by name with ID: %s", nlbIdStr)
	} else {
		// # Note) KtNlbInfo.State is info. related to VMGroup HealthInfo. (not related to the operational status.)
		cblogger.Info("### Creating New NLB!!")
		time.Sleep(time.Second * 10)
	}

	// Create VMGroup only if VMs are provided
	if nlbReqInfo.VMGroup.VMs != nil && len(*nlbReqInfo.VMGroup.VMs) > 0 {
		cblogger.Info("### Start to Create VMGroup!!")
		createErr := nlbHandler.createVMGroup(nlbIdStr, nlbReqInfo)
		if createErr != nil {
			newErr := fmt.Errorf("Failed to Create the VMGroup : [%w]", createErr)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return irs.NLBInfo{}, newErr
		}
	} else {
		cblogger.Info("### No VMs provided - skipping VMGroup creation")
	}

	ktNLB, getNLBErr := nlbHandler.getKtNlbInfo(nlbIdStr)
	if getNLBErr != nil {
		newErr := fmt.Errorf("Failed to get KT Cloud NLB info : %w", getNLBErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	cblogger.Info("### Start to Create Public IP and StaticNAT for NLB!!")
	natErr := nlbHandler.createStaticNatForNLB(ktNLB)
	if natErr != nil {
		newErr := fmt.Errorf("Failed to Create StaticNAT : [%w]", natErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	// createTagOpts := ktvpclb.CreateTagOpts{
	// 	NlbID:           	strconv.Itoa(ktNLB.NlbID),  			// Required
	// 	Tag:         		publicIp, 								// Required
	// }
	// _, tagErr := ktvpclb.CreateTag(nlbHandler.NLBClient, createTagOpts).Extract() // Not 'NetworkClient'
	// if tagErr != nil {
	// 	newErr := fmt.Errorf("Failed to Create the Tag : [%v]", tagErr.Error())
	// 	cblogger.Error(newErr.Error())
	// 	loggingError(callLogInfo, newErr)
	// 	return irs.NLBInfo{}, newErr
	// }
	// loggingInfo(callLogInfo, start)

	// cblogger.Info("\n### Creating New Tag!!")
	// time.Sleep(time.Second * 3)

	nlbIID := irs.IID{SystemId: nlbIdStr}
	nlbInfo, getErr := nlbHandler.GetNLB(nlbIID)
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Get New NLB Info : [%v]", getErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	return nlbInfo, nil
}

func (nlbHandler *KTVpcNLBHandler) createVMGroup(nlbId string, nlbReqInfo irs.NLBInfo) error {
	cblogger.Info("KT Cloud Driver: called createVMGroup()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbId, "createVMGroup()")

	if strings.EqualFold(nlbId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return newErr
	}

	// KT Cloud NLB can be created without VMs
	if nlbReqInfo.VMGroup.VMs == nil || len(*nlbReqInfo.VMGroup.VMs) < 1 {
		cblogger.Info("No VMs to add to VMGroup - skipping VMGroup creation")
		return nil
	}

	vmGroupPort, err := strconv.Atoi(nlbReqInfo.VMGroup.Port)
	if err != nil {
		newErr := fmt.Errorf("Invalid VMGroup Port. : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}
	if vmGroupPort < 1 || vmGroupPort > 65535 {
		newErr := fmt.Errorf("Invalid VMGroup Port.(Must be between 1 and 65535)")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}

	type VMInfo struct {
		VmID      string
		PrivateIP string
	}

	vmHandler := KTVpcVMHandler{
		RegionInfo:    nlbHandler.RegionInfo,
		VMClient:      nlbHandler.VMClient,
		NetworkClient: nlbHandler.NetworkClient, // Need!!
	}

	var vmInfo VMInfo
	var vmInfoList []VMInfo
	if len(*nlbReqInfo.VMGroup.VMs) > 0 {
		for _, vmIID := range *nlbReqInfo.VMGroup.VMs {
			var err error
			vmInfo.VmID, vmInfo.PrivateIP, err = vmHandler.getVmIdAndPrivateIPWithName(vmIID.NameId)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the VM ID with the VM Name : [%w]", err)
				cblogger.Error(newErr.Error())
				return newErr
			}
			vmInfoList = append(vmInfoList, vmInfo)

			time.Sleep(time.Second * 1)
			// To Prevent API timeout error
		}
	} else {
		newErr := fmt.Errorf("Failded to Find any VM NameId to Add to the NLB!!")
		cblogger.Error(newErr.Error())
		return newErr
	}

	for _, vm := range vmInfoList {
		addOpts := ktvpclb.AddServerOpts{
			NlbID:      nlbId,                   // Required
			VMID:       vm.VmID,                 // Required
			IPAddress:  vm.PrivateIP,            // Required
			PublicPort: nlbReqInfo.VMGroup.Port, // Required
		}
		start := call.Start()
		_, err := ktvpclb.AddServer(nlbHandler.NLBClient, addOpts).Extract() // Not 'NetworkClient'
		if err != nil {
			newErr := fmt.Errorf("Failed to Add VM to NLB. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return newErr
		}
		loggingInfo(callLogInfo, start)

		cblogger.Info("### Adding the VM to the NLB!!")
		time.Sleep(time.Second * 5)
	}

	return nil
}

func (nlbHandler *KTVpcNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", "ListNLB()", "ListNLB()")

	if strings.EqualFold(nlbHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Zone Info!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	var nlbList []ktvpclb.LoadBalancer
	var nlbCount int

	listOpts := ktvpclb.ListOpts{
		ZoneID: nlbHandler.RegionInfo.Zone,
	}
	// Paginate through results
	err := ktvpclb.List(nlbHandler.NLBClient, listOpts).EachPage(func(page pagination.Page) (bool, error) {
		loadBalancers, err := ktvpclb.ExtractLoadBalancers(page)
		if err != nil {
			newErr := fmt.Errorf("Failed to Extract load balancers : %w", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return false, newErr
		}

		nlbList = append(nlbList, loadBalancers...)
		nlbCount += len(loadBalancers)

		// Continue pagination
		return true, nil
	})

	if err != nil {
		newErr := fmt.Errorf("Pagination failed: %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	cblogger.Infof("=== Total: %d Load Balancers Found ===", nlbCount)

	time.Sleep(time.Second * 1) // Before 'return'
	// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"

	if len(nlbList) < 1 {
		cblogger.Info("# KT Cloud NLB does Not Exist!!")
		return []*irs.NLBInfo{}, nil // Not Return Error
	}

	var nlbInfoList []*irs.NLBInfo
	for _, nlb := range nlbList {
		nlbInfo, err := nlbHandler.mappingNlbInfo(&nlb)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get NLB Info : [%v]", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return nil, newErr
		}
		nlbInfoList = append(nlbInfoList, &nlbInfo)
	}
	return nlbInfoList, nil
}

func (nlbHandler *KTVpcNLBHandler) GetNLB(nlbIID irs.IID) (irs.NLBInfo, error) {
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", nlbIID.SystemId, "GetNLB()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return irs.NLBInfo{}, newErr
	}

	ktNLB, err := nlbHandler.getKtNlbInfo(nlbIID.SystemId)
	if err != nil {

		newErr := fmt.Errorf("Failed to Get KT Cloud NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	var nlbInfo irs.NLBInfo
	nlbInfo, err = nlbHandler.mappingNlbInfo(ktNLB)
	if err != nil {
		newErr := fmt.Errorf("Failed to Map NLB Info with the NLB : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	return nlbInfo, nil
}

func (nlbHandler *KTVpcNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called DeleteNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "DeleteNLB()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID : empty string!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	/*
		ktNLB, err := nlbHandler.getKtNlbInfo(nlbIID.SystemId)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get KT Cloud NLB info!! [%v]", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return false, newErr
		}
	*/

	// Delete FirewallRules
	dellVmErr := nlbHandler.deleteAllVMs(nlbIID)
	if dellVmErr != nil {
		cblogger.Error(dellVmErr.Error())
		loggingError(callLogInfo, dellVmErr)
		return false, dellVmErr
	}

	// Get StaticNatID and PublicIP Info from DB
	nlbDbInfo, getSGErr := nim.GetNlb(nlbIID.SystemId)
	if getSGErr != nil {
		cblogger.Debug(getSGErr)
		// return irs.VMInfo{}, getSGErr
	}

	var staticNatId string
	var publicIp string
	if nlbDbInfo != nil && countNlbKvList(*nlbDbInfo) > 0 {
		for _, kv := range nlbDbInfo.KeyValueInfoList {
			if kv.Key == "StaticNatID" {
				staticNatId = kv.Value
			}
			if kv.Key == "PublicIP" {
				publicIp = kv.Value
			}
		}
	}
	cblogger.Infof("# Retrieved from DB - StaticNatID : %s, PublicIP : %s", staticNatId, publicIp)

	// If Static NAT ID not in DB, try to find by Public IP
	if strings.EqualFold(staticNatId, "") && !strings.EqualFold(publicIp, "") {
		cblogger.Warn("StaticNAT ID not in DB, searching by Public IP")
		foundId, findErr := nlbHandler.getStaticNatIdByPublicIP(publicIp)
		if findErr != nil {
			cblogger.Warnf("Failed to Find StaticNAT by Public IP: %v", findErr)
		} else {
			staticNatId = foundId
			cblogger.Infof("Found StaticNAT ID: %s", staticNatId)
		}
	}

	vmHandler := KTVpcVMHandler{
		RegionInfo:    nlbHandler.RegionInfo,
		VMClient:      nlbHandler.VMClient,
		NetworkClient: nlbHandler.NetworkClient, // Need!!
	}

	// Caution) Before deleting the StaticNAT, must first delete the firewall settings.
	// Delete FirewallRules, Static NAT and Public IP
	if !strings.EqualFold(publicIp, "") {

		// Delete FirewallRules
		_, dellFwErr := vmHandler.removeFirewallRules(publicIp)
		if dellFwErr != nil {
			cblogger.Error(dellFwErr.Error())
			loggingError(callLogInfo, dellFwErr)
			return false, dellFwErr
		}

		// Delete StaticNAT
		if !strings.EqualFold(staticNatId, "") {
			cblogger.Info("### Deleting the StaticNAT of the NLB!!")
			err := nat.Delete(nlbHandler.NetworkClient, staticNatId).ExtractErr() // NetworkClient
			if err != nil {
				newErr := fmt.Errorf("Failed to Delete StaticNAT : [%v]", err)
				cblogger.Error(newErr.Error())
				loggingError(callLogInfo, newErr)
				return false, newErr
			}
			time.Sleep(time.Second * 4)
			cblogger.Info("StaticNAT deleted Successfully")
		}

		// Delete PublicIP
		cblogger.Info("### Deleting the Public IP of the NLB!!")
		_, dellIpErr := vmHandler.removePublicIP(publicIp)
		if dellIpErr != nil {
			newErr := fmt.Errorf("Failed to Delete Public IP : [%v]", dellIpErr)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return false, newErr
		}
		cblogger.Info("Public IP deleted successfully")
	} else {
		cblogger.Info("The NLB doesn't have any Pulbic IP!! Skipping IP deletion")
	}

	// Delete the NLB info from DB
	_, unRegErr := nim.UnRegisterNlb(nlbIID.SystemId)
	if unRegErr != nil {
		cblogger.Debug(unRegErr.Error())
		loggingError(callLogInfo, unRegErr)
		// return irs.Failed, unRegErr
	}

	// Delete the NLB
	deleteOpts := ktvpclb.DeleteOpts{
		NlbID: nlbIID.SystemId,
	}
	cblogger.Info("### Deleting the NLB!!")
	start := call.Start()
	delErr := ktvpclb.Delete(nlbHandler.NLBClient, deleteOpts).ExtractErr()
	if delErr != nil {
		newErr := fmt.Errorf("Failed to Delete the NLB : [%v]", delErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}
	loggingInfo(callLogInfo, start)
	cblogger.Info("NLB deleted Successfully")

	return true, nil
}

// ------ Frontend Control
func (nlbHandler *KTVpcNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {

	return irs.ListenerInfo{}, fmt.Errorf("Does not support yet!!")
}

// ------ Backend Control
func (nlbHandler *KTVpcNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {

	return irs.VMGroupInfo{}, fmt.Errorf("Does not support yet!!")
}

func (nlbHandler *KTVpcNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	cblogger.Info("KT Cloud Driver: called AddVMs()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "AddVMs()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return irs.VMGroupInfo{}, newErr
	}
	if len(*vmIIDs) < 1 {
		newErr := fmt.Errorf("Failded to Find any VM to Add to the VMGroup!!")
		cblogger.Error(newErr.Error())
		return irs.VMGroupInfo{}, newErr
	}

	// Get NLB information to check its subnet
	ktNLB, err := nlbHandler.getKtNlbInfo(nlbIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB info: %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	nlbSubnetId := ktNLB.NetworkID
	cblogger.Infof("NLB is in subnet (tier): %s", nlbSubnetId)

	type VMInfo struct {
		VmID      string
		PrivateIP string
		SubnetID  string
		VMName    string
	}

	vmHandler := KTVpcVMHandler{
		RegionInfo:    nlbHandler.RegionInfo,
		VMClient:      nlbHandler.VMClient,
		NetworkClient: nlbHandler.NetworkClient, // Need!!
	}

	vpcHandler := KTVpcVPCHandler{
		RegionInfo:    nlbHandler.RegionInfo,
		NetworkClient: nlbHandler.NetworkClient,
	}

	var vmInfo VMInfo
	var vmInfoList []VMInfo
	if len(*vmIIDs) > 0 {
		for _, vmIID := range *vmIIDs {
			var err error
			vmInfo.VmID, vmInfo.PrivateIP, err = vmHandler.getVmIdAndPrivateIPWithName(vmIID.NameId)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the VM ID with the VM Name : [%v]", err)
				cblogger.Error(newErr.Error())
				loggingError(callLogInfo, newErr)
				return irs.VMGroupInfo{}, newErr
			}
			vmInfo.VMName = vmIID.NameId

			// Get VM's subnet to validate it matches NLB's subnet
			ktVM, getErr := vmHandler.getKtVMInfo(vmInfo.VmID)
			if getErr != nil {
				newErr := fmt.Errorf("Failed to Get KT VM info for %s: %w", vmIID.NameId, getErr)
				cblogger.Error(newErr.Error())
				loggingError(callLogInfo, newErr)
				return irs.VMGroupInfo{}, newErr
			}

			// Extract subnet (tier) name from VM's network addresses
			var subnetName string
			for key := range ktVM.Addresses {
				subnetName = key
				break
			}

			if strings.EqualFold(subnetName, "") {
				newErr := fmt.Errorf("VM [%s] does not have valid subnet information", vmIID.NameId)
				cblogger.Error(newErr.Error())
				loggingError(callLogInfo, newErr)
				return irs.VMGroupInfo{}, newErr
			}

			// Get tier ID from tier name
			tierId, getNetErr := vpcHandler.getTierIdWithTierName(subnetName)
			if getNetErr != nil {
				newErr := fmt.Errorf("Failed to Get tier ID with name [%s]: %w", subnetName, getNetErr)
				cblogger.Error(newErr.Error())
				loggingError(callLogInfo, newErr)
				return irs.VMGroupInfo{}, newErr
			}
			vmInfo.SubnetID = *tierId

			// Validate VM is in the same subnet as NLB
			if vmInfo.SubnetID != nlbSubnetId {
				newErr := fmt.Errorf("KT Cloud NLB requires all VMs to be in the same subnet as the NLB. NLB is in subnet '%s', but VM '%s' is in subnet '%s'. Please ensure the VM is in the same subnet as the NLB",
					nlbSubnetId, vmIID.NameId, vmInfo.SubnetID)
				cblogger.Error(newErr.Error())
				loggingError(callLogInfo, newErr)
				return irs.VMGroupInfo{}, newErr
			}

			cblogger.Infof("VM %s (ID: %s) is in subnet %s - matches NLB subnet", vmIID.NameId, vmInfo.VmID, vmInfo.SubnetID)
			vmInfoList = append(vmInfoList, vmInfo)

			time.Sleep(time.Second * 1)
			// To Prevent API timeout error
		}
	} else {
		newErr := fmt.Errorf("Failded to Find any VM NameId to Add to the NLB!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	vmGroupInfo, err := nlbHandler.getVMGroupInfo(nlbIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VM Group Info with the NLB ID : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMGroupInfo{}, newErr
	}
	vmGroupPort := vmGroupInfo.Port

	for _, vm := range vmInfoList {
		addOpts := ktvpclb.AddServerOpts{
			NlbID:      nlbIID.SystemId, // Required
			VMID:       vm.VmID,         // Required
			IPAddress:  vm.PrivateIP,    // Required
			PublicPort: vmGroupPort,     // Required
		}
		start := call.Start()
		_, err := ktvpclb.AddServer(nlbHandler.NLBClient, addOpts).Extract() // Not 'NetworkClient'
		if err != nil {
			newErr := fmt.Errorf("Failed to Add VM to NLB. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return irs.VMGroupInfo{}, newErr
		}
		loggingInfo(callLogInfo, start)

		cblogger.Info("### Adding the VM to the NLB!!")
		// cblogger.Infof("# New NLBId : %s", resp.Createnlbresponse.NLBId)
		time.Sleep(time.Second * 5)
	}

	nlbInfo, getErr := nlbHandler.GetNLB(nlbIID)
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Get New NLB Info : [%v]", getErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	return nlbInfo.VMGroup, nil
}

func (nlbHandler *KTVpcNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called RemoveVMs()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", nlbIID.SystemId, "RemoveVMs()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	if len(*vmIIDs) < 1 {
		newErr := fmt.Errorf("Failded to Find any VM to Add to the VMGroup!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	// Check current VM count in the NLB
	_, currentVMCount, err := nlbHandler.getNLBVMList(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to get current VM count of NLB: %v", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	// KT Cloud requires at least one VM in the NLB at all times
	if currentVMCount-len(*vmIIDs) < 1 {
		newErr := fmt.Errorf("Cannot remove all VMs from NLB. KT Cloud NLB requires at least one VM. Current VM count: %d, Requested removal: %d. Please ensure at least one VM remains in the NLB", currentVMCount, len(*vmIIDs))
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	type VMInfo struct {
		VmID      string
		PrivateIP string
	}

	vmHandler := KTVpcVMHandler{
		RegionInfo:    nlbHandler.RegionInfo,
		VMClient:      nlbHandler.VMClient,
		NetworkClient: nlbHandler.NetworkClient, // Need!!
	}

	var vmInfo VMInfo
	var vmInfoList []VMInfo
	if len(*vmIIDs) > 0 {
		for _, vmIID := range *vmIIDs {
			var err error
			vmInfo.VmID, vmInfo.PrivateIP, err = vmHandler.getVmIdAndPrivateIPWithName(vmIID.NameId)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the VM ID with the VM Name : [%v]", err)
				cblogger.Error(newErr.Error())
				return false, newErr
			}
			vmInfoList = append(vmInfoList, vmInfo)

			time.Sleep(time.Second * 1)
			// To Prevent API timeout error
		}
	} else {
		newErr := fmt.Errorf("Failded to Find any VM NameId to Add to the NLB!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	for _, vm := range vmInfoList {
		serviceId, err := nlbHandler.getNlbServiceIdWithVMId(nlbIID.SystemId, vm.VmID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Service ID of the VM on the NLB. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return false, newErr
		}

		removeOpts := ktvpclb.RemoveServerOpts{
			ServiceID: serviceId, // Required. Not VMID
		}
		start := call.Start()
		_, rmErr := ktvpclb.RemoveServer(nlbHandler.NLBClient, removeOpts).Extract() // Not 'NetworkClient'
		if rmErr != nil {
			newErr := fmt.Errorf("Failed to Add VM to NLB. [%v]", rmErr.Error())
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return false, newErr
		}
		loggingInfo(callLogInfo, start)
		// cblogger.Infof("# New NLBId : %s", resp.Createnlbresponse.NLBId)

		cblogger.Info("### Removing the VM from the NLB!!")
		time.Sleep(time.Second * 5)
	}

	return true, nil
}

func (nlbHandler *KTVpcNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	cblogger.Info("KT Cloud Driver: called GetVMGroupHealthInfo()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", nlbIID.SystemId, "GetVMGroupHealthInfo()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return irs.HealthInfo{}, newErr
	}

	nlbVmList, vmCount, err := nlbHandler.getNLBVMList(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB VM list: %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.HealthInfo{}, newErr
	}
	cblogger.Infof("Found %d VM(s) in NLB [%s]", vmCount, nlbIID.SystemId)

	if len(nlbVmList) < 1 {
		cblogger.Info("VM does not exist on the NLB")
		return irs.HealthInfo{}, nil // Not Return Error
	}

	var allVMs []irs.IID
	var healthVMs []irs.IID
	var unHealthVMs []irs.IID

	vmHandler := KTVpcVMHandler{
		RegionInfo: nlbHandler.RegionInfo,
		VMClient:   nlbHandler.VMClient,
	}

	for _, vm := range nlbVmList {
		vmName, err := vmHandler.getVmNameWithId(vm.VmID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the VM Name with the VM ID : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.HealthInfo{}, newErr
		}

		allVMs = append(allVMs, irs.IID{NameId: vmName, SystemId: vm.VmID})

		if strings.EqualFold(vm.VmState, "UP") {
			// cblogger.Infof("\n### [%s] is Healthy VM.", vm.Name)
			healthVMs = append(healthVMs, irs.IID{NameId: vmName, SystemId: vm.VmID})
		} else {
			// cblogger.Infof("\n### [%s] is Unhealthy VM.", vm.Name)
			unHealthVMs = append(unHealthVMs, irs.IID{NameId: vmName, SystemId: vm.VmID}) // In case of "DOWN", ...
		}
	}

	vmGroupHealthInfo := irs.HealthInfo{
		AllVMs:       &allVMs,
		HealthyVMs:   &healthVMs,
		UnHealthyVMs: &unHealthVMs,
	}
	return vmGroupHealthInfo, nil
}

func (nlbHandler *KTVpcNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {
	return irs.HealthCheckerInfo{}, fmt.Errorf("KT Cloud does not support ChangeHealthCheckerInfo() yet!!")
}

func (nlbHandler *KTVpcNLBHandler) getListenerInfo(nlb *ktvpclb.LoadBalancer) (irs.ListenerInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called getListenerInfo()")

	nlbId := strconv.Itoa(nlb.NlbID)
	InitLog()
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbId, "getListenerInfo()")

	if strings.EqualFold(nlbId, "") {
		newErr := fmt.Errorf("Invalid Load-Balancer ID. The LB does Not Exit!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.ListenerInfo{}, newErr
	}

	listenerInfo := irs.ListenerInfo{
		Protocol: nlb.ServiceType,
		IP:       nlb.ServiceIP,
		Port:     nlb.ServicePort,
		// DNSName:		"NA",
		// CspID: 		"NA",
		// KeyValueList:   irs.StructToKeyValueList(nlb),
	}
	return listenerInfo, nil
}

func (nlbHandler *KTVpcNLBHandler) getHealthCheckerInfo(nlb *ktvpclb.LoadBalancer) (irs.HealthCheckerInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called getHealthCheckerInfo()")

	nlbId := strconv.Itoa(nlb.NlbID)
	InitLog()
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbId, "getHealthCheckerInfo()")

	if strings.EqualFold(nlbId, "") {
		newErr := fmt.Errorf("Invalid Load-Balancer ID. The LB does Not Exit!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.HealthCheckerInfo{}, newErr
	}

	healthCheckerInfo := irs.HealthCheckerInfo{
		Protocol: nlb.HealthCheckType,
		Port:     nlb.ServicePort,
		// CspID: 		"NA",
	}
	return healthCheckerInfo, nil
}

func (nlbHandler *KTVpcNLBHandler) getVMGroupInfo(nlbId string) (irs.VMGroupInfo, error) {
	cblogger.Info("KT Cloud Driver: called getVMGroupInfo()")
	InitLog()
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbId, "getVMGroupInfo()")

	if strings.EqualFold(nlbId, "") {
		newErr := fmt.Errorf("Invalid NLB ID")
		cblogger.Error(newErr.Error())
		return irs.VMGroupInfo{}, newErr
	}

	ktNLB, getErr := nlbHandler.getKtNlbInfo(nlbId)
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud NLB info!! [%w]", getErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	nlbIID := irs.IID{SystemId: nlbId}
	nlbVmList, vmCount, err := nlbHandler.getNLBVMList(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB VM list: %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	cblogger.Infof("Found %d VM(s) in NLB [%s]", vmCount, nlbId)

	if len(nlbVmList) < 1 {
		cblogger.Info("VM does not exist on the NLB")
		// Return VMGroupInfo with basic protocol and port info from NLB, even when no VMs exist
		vmGroupInfo := irs.VMGroupInfo{
			Protocol: ktNLB.ServiceType,
			Port:     ktNLB.ServicePort,
			VMs:      &[]irs.IID{}, // Empty VM list
		}
		return vmGroupInfo, nil
	}

	vmGroupInfo := irs.VMGroupInfo{
		Protocol: ktNLB.ServiceType,       // Caution!!
		Port:     nlbVmList[0].PublicPort, // In case, Any VM exists
		// CspID:    	"NA",
		KeyValueList: irs.StructToKeyValueList(nlbVmList[0]),
	}

	vmIIds := []irs.IID{}
	vmHandler := KTVpcVMHandler{
		RegionInfo: nlbHandler.RegionInfo,
		VMClient:   nlbHandler.VMClient,
	}
	for _, vm := range nlbVmList {
		vmName, err := vmHandler.getVmNameWithId(vm.VmID)
		if err != nil {
			newErr := fmt.Errorf("failed to get VM name with ID [%s]: %w", vm.VmID, err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return irs.VMGroupInfo{}, newErr
		}

		vmIIds = append(vmIIds, irs.IID{
			NameId:   vmName,
			SystemId: vm.VmID,
		})

		keyValue := irs.KeyValue{
			Key:   vmName + "_ServiceId",
			Value: strconv.Itoa(vm.ServiceID),
		}
		vmGroupInfo.KeyValueList = append(vmGroupInfo.KeyValueList, keyValue)

		time.Sleep(time.Second * 1)
		// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"
	}
	vmGroupInfo.VMs = &vmIIds
	return vmGroupInfo, nil
}

func (nlbHandler *KTVpcNLBHandler) getNlbServiceIdWithVMId(nlbId string, vmId string) (string, error) {
	cblogger.Info("KT Cloud Driver: called getNlbServiceIdWithVMId()")
	InitLog()
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbId, "getNlbServiceIdWithVMId()")

	if strings.EqualFold(nlbId, "") {
		newErr := fmt.Errorf("Invalid NLB ID")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	if strings.EqualFold(vmId, "") {
		newErr := fmt.Errorf("Invalid VM ID")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	nlbIID := irs.IID{SystemId: nlbId}
	nlbVmList, vmCount, err := nlbHandler.getNLBVMList(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB VM list: %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}
	cblogger.Infof("Found %d VM(s) in NLB [%s]", vmCount, nlbId)

	if len(nlbVmList) < 1 {
		cblogger.Info("VM does not exist on the NLB")
		return "", nil // Not Return Error
	}

	var serviceID string
	for _, vm := range nlbVmList {
		if strings.EqualFold(vm.VmID, vmId) {
			serviceID = strconv.Itoa(vm.ServiceID)
			break
		}
	}

	if strings.EqualFold(serviceID, "") {
		newErr := fmt.Errorf("Failed to Find the Service ID with the VM ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	return serviceID, nil
}

func (nlbHandler *KTVpcNLBHandler) mappingNlbInfo(nlb *ktvpclb.LoadBalancer) (irs.NLBInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called mappingNlbInfo()")

	vpcHandler := KTVpcVPCHandler{
		RegionInfo:    nlbHandler.RegionInfo,
		NetworkClient: nlbHandler.NetworkClient,
	}

	vpcId, err := vpcHandler.getVPCIdWithTierId(nlb.NetworkID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VPC ID with the Tier ID : [%w]", err)
		cblogger.Error(newErr.Error())
		return irs.NLBInfo{}, newErr
	}

	nlbInfo := irs.NLBInfo{
		IId: irs.IID{
			NameId:   nlb.Name,
			SystemId: strconv.Itoa(nlb.NlbID),
		},
		VpcIID: irs.IID{
			// NameId:   "N/A", // Cauton!!) 'NameId: "N/A"' makes an Error on CB-Spider
			SystemId: *vpcId,
		},
		Type:         "PUBLIC",
		Scope:        "REGION",
		KeyValueList: irs.StructToKeyValueList(nlb),
	}

	if !strings.EqualFold(nlb.ServiceIP, "") {
		listenerInfo, err := nlbHandler.getListenerInfo(nlb)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get Listener Info : [%w]", err)
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		}
		nlbInfo.Listener = listenerInfo
	}

	// Get NLB info from DB
	nlbDbInfo, getSGErr := nim.GetNlb(strconv.Itoa(nlb.NlbID))
	if getSGErr != nil {
		cblogger.Debug(getSGErr)
		// return irs.VMInfo{}, getSGErr
	}

	var publicIp string
	if countNlbKvList(*nlbDbInfo) > 0 {
		for _, kv := range nlbDbInfo.KeyValueInfoList {
			if kv.Key == "PublicIP" {
				publicIp = kv.Value
				break
			}
		}
	}
	nlbInfo.Listener.IP = publicIp

	if !strings.EqualFold(nlb.HealthCheckType, "") {
		healthCheckerInfo, err := nlbHandler.getHealthCheckerInfo(nlb)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get HealthChecker info frome the NLB. [%w]", err)
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		}
		nlbInfo.HealthChecker = healthCheckerInfo
	}

	vmGroupInfo, err := nlbHandler.getVMGroupInfo(strconv.Itoa(nlb.NlbID))
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VMGroup Info with the NLB ID : [%w]", err)
		cblogger.Error(newErr.Error())
		return irs.NLBInfo{}, newErr
	}
	nlbInfo.VMGroup = vmGroupInfo
	return nlbInfo, nil
}

func (nlbHandler *KTVpcNLBHandler) getKtNlbInfo(nlbId string) (*ktvpclb.LoadBalancer, error) {
	cblogger.Info("KT Cloud VPC Driver: called getKtNlbInfo()")
	InitLog()
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", nlbId, "getKtNlbInfo()")

	if strings.EqualFold(nlbId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	var nlbList []ktvpclb.LoadBalancer
	listOpts := ktvpclb.ListOpts{
		ZoneID: nlbHandler.RegionInfo.Zone,
		NlbID:  nlbId,
	}
	// Paginate through results
	err := ktvpclb.List(nlbHandler.NLBClient, listOpts).EachPage(func(page pagination.Page) (bool, error) {
		loadBalancers, err := ktvpclb.ExtractLoadBalancers(page)
		if err != nil {
			return false, fmt.Errorf("Failed to Extract load balancers : %w", err)
		}

		nlbList = append(nlbList, loadBalancers...)

		// Continue pagination
		return true, nil
	})

	if err != nil {
		return nil, fmt.Errorf("Pagination failed: %w", err)
	}

	time.Sleep(time.Second * 1) // Before 'return'
	// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"

	if len(nlbList) < 1 {
		newErr := fmt.Errorf("Failed to Find the NLB with the ID on the zone!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	return &nlbList[0], nil
}

func (nlbHandler *KTVpcNLBHandler) getKtNlbWithName(nlbName string) (*ktvpclb.LoadBalancer, error) {
	cblogger.Info("KT Cloud VPC Driver: called getKtNlbWithName()")
	InitLog()
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", nlbName, "getKtNlbWithName()")

	if strings.EqualFold(nlbName, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	listOpts := ktvpclb.ListOpts{
		ZoneID: nlbHandler.RegionInfo.Zone,
		Name:   nlbName,
	}
	start := call.Start()
	firstPage, err := ktvpclb.List(nlbHandler.NLBClient, listOpts).FirstPage() // Not '~.AllPages()'
	if err != nil {
		cblogger.Errorf("Failed to Get KT Cloud NLB Page : [%v]", err)
		return nil, err
	}
	nlbList, err := ktvpclb.ExtractLoadBalancers(firstPage)
	if err != nil {
		cblogger.Errorf("Failed to Get KT Cloud NLB list : [%v]", err)
		return nil, err
	}
	loggingInfo(callLogInfo, start)

	time.Sleep(time.Second * 1) // Before 'return'
	// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"

	if len(nlbList) < 1 {
		newErr := fmt.Errorf("Failed to Find the NLB with the Name on the zone!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	return &nlbList[0], nil
}

func countVMs(vmGroup irs.VMGroupInfo) int {
	if vmGroup.VMs == nil {
		return 0
	}
	return len(*vmGroup.VMs)
}

// Note) StaticNAT allocation is required after new Public IP creation to be connected from the outside of the network.
func (nlbHandler *KTVpcNLBHandler) createStaticNatForNLB(ktNLB *ktvpclb.LoadBalancer) error {
	cblogger.Info("KT Cloud VPC Driver: called createStaticNatForNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", ktNLB.Name, "createStaticNatForNLB()")

	if ktNLB == nil {
		newErr := fmt.Errorf("invalid NLB info: nil pointer")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}

	if strings.EqualFold(ktNLB.ServiceIP, "") {
		newErr := fmt.Errorf("invalid NLB service IP: empty string")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}

	vmHandler := KTVpcVMHandler{
		RegionInfo:    nlbHandler.RegionInfo,
		VMClient:      nlbHandler.VMClient,
		NetworkClient: nlbHandler.NetworkClient, // Need!!
	}
	// Create Public IP
	cblogger.Info("Creating New Public IP for NLB")
	ok, publicIPId, creatErr := vmHandler.createPublicIP()
	if !ok {
		newErr := fmt.Errorf("Failed to create a public IP: %w", creatErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}
	cblogger.Infof("# Created PublicIP ID : %s", publicIPId)
	time.Sleep(time.Second * 1)

	publicIP, err := vmHandler.findPublicIPByID(publicIPId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find PublicIP by ID : %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}
	cblogger.Infof("# Public IP address : %s", publicIP)

	// Create StaticNAT
	createNatOpts := &nat.CreateOpts{
		PublicIpID:    publicIPId,
		PrivateIpAddr: ktNLB.ServiceIP,
	}
	cblogger.Info("### Creating the StaticNAT for the NLB!!")
	natResult, err := nat.Create(nlbHandler.NetworkClient, createNatOpts).ExtractCreate()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create StaticNAT : [%w]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}
	cblogger.Info("StaticNAT creation initiated")
	time.Sleep(time.Second * 3)

	// Extract Static NAT ID from response
	var staticNatId string
	if natResult != nil && natResult.Data.StaticNatID != "" {
		staticNatId = natResult.Data.StaticNatID
		cblogger.Infof("StaticNAT ID from response: %s", staticNatId)
	} else {
		cblogger.Warn("StaticNAT ID not in response, querying by public IP")
		time.Sleep(time.Second * 5)

		queryId, queryErr := nlbHandler.getStaticNatIdByPublicIP(publicIP)
		if queryErr != nil {
			newErr := fmt.Errorf("Failed to Get StaticNAT ID by public IP: %w", queryErr)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return newErr
		}
		staticNatId = queryId
	}

	if strings.EqualFold(staticNatId, "") {
		newErr := fmt.Errorf("StaticNAT ID is empty after creation")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}

	cblogger.Infof("StaticNAT created successfully - ID: %s, PublicIP: %s", staticNatId, publicIP)

	var keyValueList []irs.KeyValue
	keyValueList = append(keyValueList, irs.KeyValue{
		Key:   "StaticNatID",
		Value: staticNatId,
	})
	keyValueList = append(keyValueList, irs.KeyValue{
		Key:   "PublicIP",
		Value: publicIP,
	})

	// Register NLB info to DB
	providerName := "KTVPC"
	nlbDbInfo, regErr := nim.RegisterNlb(strconv.Itoa(ktNLB.NlbID), providerName, keyValueList)
	if regErr != nil {
		cblogger.Error(regErr)
		return regErr
	}
	cblogger.Infof("# NLB Info to Register to DB : [%v]", nlbDbInfo)

	// Optional: Create firewall rules for NLB
	if err := nlbHandler.createFirewallRulesForNLB(ktNLB, publicIP, staticNatId); err != nil {
		newErr := fmt.Errorf("failed to create firewall rules for NLB: %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}

	return nil
}

// getStaticNatIdByPublicIP retrieves StaticNAT ID using Public IP address
func (nlbHandler *KTVpcNLBHandler) getStaticNatIdByPublicIP(publicIP string) (string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getStaticNatIdByPublicIP()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", publicIP, "getStaticNatIdByPublicIP()")

	if strings.EqualFold(publicIP, "") {
		newErr := fmt.Errorf("invalid public IP: empty string")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}

	start := call.Start()
	staticNatList, err := nlbHandler.listKTStaticNat()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Static NAT List : %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}
	loggingInfo(callLogInfo, start)

	if len(staticNatList) == 0 {
		newErr := fmt.Errorf("No static NAT found in the zone")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}

	// Find Static NAT matching the public IP
	for _, staticNat := range staticNatList {
		if strings.EqualFold(staticNat.PublicIP, publicIP) {
			cblogger.Infof("Found static NAT ID: %s for public IP: %s", staticNat.StaticNatID, publicIP)
			return staticNat.StaticNatID, nil
		}
	}

	// No matching Static NAT found
	newErr := fmt.Errorf("No static NAT found for public IP [%s]", publicIP)
	cblogger.Error(newErr.Error())
	loggingError(callLogInfo, newErr)
	return "", newErr
}

func countNlbKvList(nlb nim.NlbInfo) int {
	if nlb.KeyValueInfoList == nil {
		return 0
	}
	return len(nlb.KeyValueInfoList)
}

func (nlbHandler *KTVpcNLBHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("KT Cloud VPC driver: called ListIID()!!")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", "ListIID()", "ListIID()")

	listOpts := ktvpclb.ListOpts{
		ZoneID: nlbHandler.RegionInfo.Zone,
	}
	start := call.Start()
	firstPage, err := ktvpclb.List(nlbHandler.NLBClient, listOpts).FirstPage() // Not 'NetworkClient', Not 'AllPages()'
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB List from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	loggingInfo(callLogInfo, start)

	nlbList, err := ktvpclb.ExtractLoadBalancers(firstPage)
	if err != nil {
		newErr := fmt.Errorf("Failed to Extract NLB List : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if len(nlbList) < 1 {
		cblogger.Info("### There is No NLB!!")
		return nil, nil
	}

	var iidList []*irs.IID
	for _, nlb := range nlbList {
		iid := &irs.IID{
			NameId:   nlb.Name,
			SystemId: strconv.Itoa(nlb.NlbID),
		}
		iidList = append(iidList, iid)
	}
	return iidList, nil
}

// getSubnetOfVMGroup finds the subnet (tier) where VMs in the VMGroup belong
// Returns the subnet ID after validating all VMs are in the same subnet
// KT Cloud NLB requires at least one VM and all VMs must be in the same subnet
func (nlbHandler *KTVpcNLBHandler) getSubnetOfVMGroup(ctx context.Context, vmGroup irs.VMGroupInfo) (string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getSubnetOfVMGroup()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", "VMGroup", "getSubnetOfVMGroup()")

	// KT Cloud NLB requires at least one VM because NLB operates at subnet level
	if vmGroup.VMs == nil || len(*vmGroup.VMs) == 0 {
		newErr := fmt.Errorf("KT Cloud NLB requires at least one VM. NLB operates at subnet level and must be created with VMs in the target subnet. Please provide at least one VM in the VMGroup")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		newErr := fmt.Errorf("Context cancelled before getting VM subnet: %w", ctx.Err())
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	default:
	}

	vmHandler := KTVpcVMHandler{
		RegionInfo:    nlbHandler.RegionInfo,
		VMClient:      nlbHandler.VMClient,
		NetworkClient: nlbHandler.NetworkClient,
	}

	vpcHandler := KTVpcVPCHandler{
		RegionInfo:    nlbHandler.RegionInfo,
		NetworkClient: nlbHandler.NetworkClient,
	}

	// Validate all VMs and collect their subnet IDs
	vms := *vmGroup.VMs
	var subnetIds []string
	var vmNames []string

	cblogger.Infof("Validating %d VMs for subnet consistency", len(vms))

	for _, vmIID := range vms {
		if strings.EqualFold(vmIID.NameId, "") && strings.EqualFold(vmIID.SystemId, "") {
			newErr := fmt.Errorf("Invalid VM IID: both NameId and SystemId are empty")
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return "", newErr
		}

		var vmId string
		if !strings.EqualFold(vmIID.SystemId, "") {
			vmId = vmIID.SystemId
		} else {
			// Get VM ID from VM name
			foundVmId, _, getIdErr := vmHandler.getVmIdAndPrivateIPWithName(vmIID.NameId)
			if getIdErr != nil {
				newErr := fmt.Errorf("Failed to Get VM ID with name [%s]: %w", vmIID.NameId, getIdErr)
				cblogger.Error(newErr.Error())
				loggingError(callLogInfo, newErr)
				return "", newErr
			}
			vmId = foundVmId
		}

		// Get KT Cloud VM using getKTVM
		ktVM, getErr := vmHandler.getKtVMInfo(vmId)
		if getErr != nil {
			newErr := fmt.Errorf("Failed to Get KT VM info for %s: %w", vmIID.NameId, getErr)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return "", newErr
		}

		// Extract subnet (tier) name from VM's network addresses
		var subnetName string
		for key := range ktVM.Addresses {
			subnetName = key
			break
		}

		if strings.EqualFold(subnetName, "") {
			newErr := fmt.Errorf("VM [%s] does not have valid subnet information", vmIID.NameId)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return "", newErr
		}

		// Get tier ID from tier name
		tierId, getNetErr := vpcHandler.getTierIdWithTierName(subnetName)
		if getNetErr != nil {
			newErr := fmt.Errorf("Failed to Get tier ID with name [%s]: %w", subnetName, getNetErr)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return "", newErr
		}

		subnetIds = append(subnetIds, *tierId)
		vmNames = append(vmNames, vmIID.NameId)

		cblogger.Infof("VM %s (ID: %s) is in subnet %s (tier name: %s)", vmIID.NameId, vmId, *tierId, subnetName)
	}

	// Validate all VMs are in the same subnet
	if len(subnetIds) == 0 {
		newErr := fmt.Errorf("No valid VMs found in VMGroup")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}

	firstSubnetId := subnetIds[0]
	for i := 1; i < len(subnetIds); i++ {
		if subnetIds[i] != firstSubnetId {
			newErr := fmt.Errorf("KT Cloud NLB requires all VMs to be in the same subnet. VM '%s' is in subnet '%s', but VM '%s' is in subnet '%s'. Please ensure all VMs in the VMGroup are in the same subnet",
				vmNames[0], firstSubnetId, vmNames[i], subnetIds[i])
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return "", newErr
		}
	}

	cblogger.Infof("All %d VMs validated - all are in subnet %s", len(vms), firstSubnetId)
	return firstSubnetId, nil
}

// createFirewallRulesForNLB creates inbound and outbound firewall rules for NLB
// Supports TCP, UDP, and ICMP protocols with proper error handling and logging
func (nlbHandler *KTVpcNLBHandler) createFirewallRulesForNLB(ktNLB *ktvpclb.LoadBalancer, publicIP, staticNatId string) error {
	cblogger.Info("KT Cloud VPC Driver: called createFirewallRulesForNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", strconv.Itoa(ktNLB.NlbID), "createFirewallRulesForNLB()")

	if ktNLB == nil {
		newErr := fmt.Errorf("Invalid NLB info")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}

	if strings.EqualFold(publicIP, "") {
		newErr := fmt.Errorf("Invalid public IP")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}

	if strings.EqualFold(staticNatId, "") {
		newErr := fmt.Errorf("Invalid static NAT ID")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}

	vpcHandler := KTVpcVPCHandler{
		RegionInfo:    nlbHandler.RegionInfo,
		NetworkClient: nlbHandler.NetworkClient,
	}
	// Get external network ID
	extNetId, err := vpcHandler.getNetworkID("external")
	if err != nil {
		newErr := fmt.Errorf("Failed to Get External network ID: %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}
	cblogger.Infof("External network ID: %s", *extNetId)

	// Get tier network ID
	networkId, err := vpcHandler.getNetworkIdWithTierId(ktNLB.NetworkID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Tier Network ID: %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}
	cblogger.Infof("Tier network ID: %s", *networkId)

	// Convert service type to protocol
	protocol := ktNLB.ServiceType
	var protocols []string

	if strings.EqualFold(protocol, "tcp") {
		protocols = []string{"TCP"}
	} else if strings.EqualFold(protocol, "udp") {
		protocols = []string{"UDP"}
	} else if strings.EqualFold(protocol, "icmp") {
		protocols = []string{"ICMP"}
	} else {
		newErr := fmt.Errorf("Unsupported protocol: %s", protocol)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}

	// Create firewall rules for each protocol
	for _, curProtocol := range protocols {
		cblogger.Infof("Creating firewall rules for protocol: %s", curProtocol)

		// Create inbound firewall rule
		if err := nlbHandler.createStaticNatInboundFirewallRule(ktNLB, publicIP, staticNatId, curProtocol, *extNetId, *networkId); err != nil {
			newErr := fmt.Errorf("Failed to create inbound firewall rule for %s: %w", curProtocol, err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return newErr
		}

		// Create outbound firewall rule
		if err := nlbHandler.createStaticNatOutboundFirewallRule(ktNLB, curProtocol, *extNetId, *networkId); err != nil {
			newErr := fmt.Errorf("Failed to create outbound firewall rule for %s: %w", curProtocol, err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return newErr
		}
	}

	return nil
}

// ### createInboundFirewallRule creates an inbound firewall rule for the specified protocol 'fot StaticNAT'
func (nlbHandler *KTVpcNLBHandler) createStaticNatInboundFirewallRule(ktNLB *ktvpclb.LoadBalancer, publicIP, staticNatId, protocol, extNetId, networkId string) error {
	cblogger.Info("KT Cloud VPC Driver: called createInboundFirewallRule()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", strconv.Itoa(ktNLB.NlbID), "createInboundFirewallRule()")

	cblogger.Infof("Start to create firewall inbound rule for protocol: [%s]", protocol)

	// Prepare destination CIDR
	destCIDR, err := ipToCidr32(publicIP)
	if err != nil {
		newErr := fmt.Errorf("Failed to convert public IP to CIDR: %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}
	cblogger.Infof("destination CIDR: %s", destCIDR)

	srcIPAdds := "0.0.0.0/0"

	// Create firewall rule options
	var inboundFWOpts *rules.CreateOpts

	if strings.EqualFold(protocol, "icmp") {
		// ICMP does not have port forwarding
		comment := "Allow inbound - " + protocol
		inboundFWOpts = &rules.CreateOpts{
			Action:      true, // accept
			Protocol:    protocol,
			SrcNetwork:  []string{extNetId},
			StaticNatId: staticNatId, // Caution!!
			SrcAddress:  []string{srcIPAdds},
			DstAddress:  []string{destCIDR},
			Comment:     comment,
			SrcNat:      false,
		}
	} else {
		// TCP and UDP with port forwarding
		comment := "Allow inbound - " + protocol + " - " + ktNLB.ServicePort + " to " + ktNLB.ServicePort
		inboundFWOpts = &rules.CreateOpts{
			Action:      true, // accept
			Protocol:    protocol,
			StartPort:   ktNLB.ServicePort,
			EndPort:     ktNLB.ServicePort,
			SrcNetwork:  []string{extNetId},
			StaticNatId: staticNatId, // Caution!!
			SrcAddress:  []string{srcIPAdds},
			DstAddress:  []string{destCIDR},
			Comment:     comment,
			SrcNat:      false,
		}
	}

	vmHandler := KTVpcVMHandler{
		RegionInfo:    nlbHandler.RegionInfo,
		VMClient:      nlbHandler.VMClient,
		NetworkClient: nlbHandler.NetworkClient,
	}

	// Create firewall rule
	start := call.Start()
	fwResult := rules.Create(vmHandler.NetworkClient, inboundFWOpts)
	if fwResult.Err != nil {
		newErr := fmt.Errorf("Failed to Create Firewall Inbound rule: %w", fwResult.Err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}
	loggingInfo(callLogInfo, start)

	jobId, err := rules.ExtractJobID(fwResult)
	if err != nil {
		cblogger.Infof("Failed to extract job ID: %v", err)
	} else {
		cblogger.Infof("Firewall inbound rule job ID: %s", jobId)
	}

	cblogger.Info("Waiting for firewall inbound rule to be created (600 sec)")
	time.Sleep(time.Second * 2)

	cblogger.Infof("Successfully created firewall inbound rule for protocol: %s", protocol)
	return nil
}

// createOutboundFirewallRule creates an outbound firewall rule for the specified protocol
func (nlbHandler *KTVpcNLBHandler) createStaticNatOutboundFirewallRule(ktNLB *ktvpclb.LoadBalancer, protocol, extNetId, networkId string) error {
	cblogger.Info("KT Cloud VPC Driver: called createOutboundFirewallRule()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", strconv.Itoa(ktNLB.NlbID), "createOutboundFirewallRule()")

	cblogger.Infof("Start to create firewall outbound rule for protocol: [%s]", protocol)

	// Prepare source CIDR
	srcCIDR, err := ipToCidr32(ktNLB.ServiceIP)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert service IP to CIDR: %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}
	cblogger.Infof("Source CIDR: %s", srcCIDR)

	destIPAdds := "0.0.0.0/0"

	// Create firewall rule options
	var outboundFWOpts *rules.CreateOpts

	if strings.EqualFold(protocol, "icmp") {
		// ICMP does not use ports
		comment := "Allow outbound - " + protocol
		outboundFWOpts = &rules.CreateOpts{
			Action:     true, // accept
			Protocol:   protocol,
			SrcNetwork: []string{networkId},
			DstNetwork: []string{extNetId},
			SrcAddress: []string{srcCIDR},
			DstAddress: []string{destIPAdds},
			Comment:    comment,
			SrcNat:     true,
		}
	} else {
		// TCP and UDP with ports
		comment := "Allow outbound - " + protocol + " - " + ktNLB.ServicePort + " to " + ktNLB.ServicePort
		outboundFWOpts = &rules.CreateOpts{
			Action:     true, // accept
			Protocol:   protocol,
			StartPort:  ktNLB.ServicePort,
			EndPort:    ktNLB.ServicePort,
			SrcNetwork: []string{networkId},
			DstNetwork: []string{extNetId},
			SrcAddress: []string{srcCIDR},
			DstAddress: []string{destIPAdds},
			Comment:    comment,
			SrcNat:     true,
		}
	}

	vmHandler := KTVpcVMHandler{
		RegionInfo:    nlbHandler.RegionInfo,
		VMClient:      nlbHandler.VMClient,
		NetworkClient: nlbHandler.NetworkClient,
	}

	// Create firewall rule
	start := call.Start()
	fwResult := rules.Create(vmHandler.NetworkClient, outboundFWOpts)
	if fwResult.Err != nil {
		newErr := fmt.Errorf("Failed to create firewall outbound rule: %w", fwResult.Err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return newErr
	}
	loggingInfo(callLogInfo, start)

	jobId, err := rules.ExtractJobID(fwResult)
	if err != nil {
		cblogger.Infof("Failed to extract job ID: %v", err)
	} else {
		cblogger.Infof("Firewall outbound rule job ID: %s", jobId)
	}

	cblogger.Info("Waiting for firewall outbound rule to be created (600 sec)")
	time.Sleep(time.Second * 2)

	cblogger.Infof("Successfully created firewall outbound rule for protocol: %s", protocol)
	return nil
}

// checkNLBExists checks if an NLB with the given name already exists in the VPC
func (nlbHandler *KTVpcNLBHandler) checkNLBExists(ctx context.Context, nlbName string) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called checkNLBExists()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", nlbName, "checkNLBExists()")

	if strings.EqualFold(nlbName, "") {
		newErr := fmt.Errorf("Invalid NLB name: empty string")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		newErr := fmt.Errorf("Context cancelled before checking NLB existence: %w", ctx.Err())
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	default:
	}

	listOpts := ktvpclb.ListOpts{
		Name:   nlbName,
		ZoneID: nlbHandler.RegionInfo.Zone,
	}

	var nlbExists bool

	start := call.Start()
	err := ktvpclb.List(nlbHandler.NLBClient, listOpts).EachPage(func(page pagination.Page) (bool, error) {
		// Check context during pagination
		select {
		case <-ctx.Done():
			return false, fmt.Errorf("Context cancelled during NLB existence check: %w", ctx.Err())
		default:
		}

		loadBalancers, err := ktvpclb.ExtractLoadBalancers(page)
		if err != nil {
			return false, fmt.Errorf("Failed to extract load balancers: %w", err)
		}

		if len(loadBalancers) > 0 {
			nlbExists = true
			cblogger.Infof("NLB with name [%s] already exists (ID: %d)", nlbName, loadBalancers[0].NlbID)
			return false, nil // Stop pagination
		}

		// Continue searching
		return true, nil
	})
	loggingInfo(callLogInfo, start)

	if err != nil {
		newErr := fmt.Errorf("Failed to check NLB existence: %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	if !nlbExists {
		cblogger.Infof("No NLB found with the name [%s]", nlbName)
	}

	return nlbExists, nil
}

func (nlbHandler *KTVpcNLBHandler) listKTStaticNat() ([]*nat.StaticNAT, error) {
	cblogger.Info("KT Cloud VPC Driver: called listKTStaticNat()!")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, call.VPCSUBNET, "listKTStaticNat()", "listKTStaticNat()")

	// ### If enter a different number to ListOpts, the value will not be retrieved correctly.
	listOpts := nat.ListOpts{
		Page: 1,
		Size: 20,
	}
	start := call.Start()
	pager := nat.List(nlbHandler.NetworkClient, listOpts) // NetworkClient
	loggingInfo(callLogInfo, start)

	var staticNatList []*nat.StaticNAT
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		natlist, err := nat.ExtractStaticNats(page)
		if err != nil {
			newErr := fmt.Errorf("Failed to Extract StaticNAT List : [%v]", err)
			cblogger.Error(newErr.Error())
			return false, newErr
		}
		if len(natlist) < 1 {
			newErr := fmt.Errorf("Failed to Find Any StaticNAT on this page!!")
			cblogger.Debug(newErr.Error())
			return false, newErr
		}

		for _, staticNat := range natlist {
			staticNatList = append(staticNatList, &staticNat)
		}

		return true, nil
	})
	if err != nil {
		newErr := fmt.Errorf("Failed to Get StaticNAT List : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	cblogger.Infof("Found %d static NAT(s) in the zone", len(staticNatList))
	return staticNatList, nil
}

// Delete all VMs with detailed error handling
func (nlbHandler *KTVpcNLBHandler) deleteAllVMs(nlbIID irs.IID) error {
	cblogger.Info("KT Cloud VPC Driver: called deleteAllVMs()")

	// Get initial VM count
	nlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err != nil {
		fmt.Printf("Failed to get NLB info: %v\n", err)
		return err
	}

	initialCount := countVMs(nlbInfo.VMGroup)
	fmt.Printf("Initial VM count: %d\n", initialCount)

	if initialCount == 0 {
		fmt.Println("No VMs to remove")
		return nil
	}

	// Delete all VMs
	fmt.Println("Starting VM removal process...")
	removedCount, err := nlbHandler.deleteAllVMsFromNLB(nlbIID)

	// Handle results
	if err != nil {
		fmt.Printf("Error occurred during VM removal: %v\n", err)
		fmt.Printf("Successfully removed: %d/%d VMs\n", removedCount, initialCount)

		// Verify remaining VMs
		updatedInfo, getErr := nlbHandler.GetNLB(nlbIID)
		if getErr != nil {
			fmt.Printf("Failed to verify remaining VMs: %v\n", getErr)
			return getErr
		}

		remainingCount := countVMs(updatedInfo.VMGroup)
		fmt.Printf("Remaining VMs: %d\n", remainingCount)

		if remainingCount > 0 {
			fmt.Println("Some VMs remain in NLB:")
			for _, vm := range *updatedInfo.VMGroup.VMs {
				fmt.Printf("  - VM: %s (ID: %s)\n", vm.NameId, vm.SystemId)
			}
		}
	} else {
		fmt.Printf("Successfully removed all %d VM(s) from NLB\n", removedCount)

		// Verify NLB is empty
		verifyInfo, getErr := nlbHandler.GetNLB(nlbIID)
		if getErr != nil {
			fmt.Printf("Failed to verify NLB status: %v\n", getErr)
			return getErr
		}

		finalCount := countVMs(verifyInfo.VMGroup)
		if finalCount == 0 {
			fmt.Println("✓ Verified: NLB is now empty")
		} else {
			fmt.Printf("⚠ Warning: NLB still has %d VM(s)\n", finalCount)
		}
	}

	return nil
}

// deleteAllVMsFromNLB removes all VMs from the specified NLB
// Returns the number of VMs removed and any error encountered
func (nlbHandler *KTVpcNLBHandler) deleteAllVMsFromNLB(nlbIID irs.IID) (int, error) {
	cblogger.Info("KT Cloud VPC Driver: called deleteAllVMsFromNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", nlbIID.SystemId, "deleteAllVMsFromNLB()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID: empty string")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return 0, newErr
	}

	nlbVmList, vmCount, err := nlbHandler.getNLBVMList(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB VM list: %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return 0, newErr
	}

	cblogger.Infof("Found %d VM(s) in NLB [%s]", vmCount, nlbIID.SystemId)

	if len(nlbVmList) < 1 {
		cblogger.Info("VM does not exist on the NLB")
		return 0, nil // Not Return Error
	}

	// Remove each VM from the NLB
	removedCount := 0
	var removeErrors []string

	for _, vm := range nlbVmList {
		cblogger.Infof("Removing VM [%s] (ServiceID: %d) from NLB", vm.VmID, vm.ServiceID)

		removeOpts := ktvpclb.RemoveServerOpts{
			ServiceID: strconv.Itoa(vm.ServiceID),
		}

		start := call.Start()
		_, rmErr := ktvpclb.RemoveServer(nlbHandler.NLBClient, removeOpts).Extract()
		if rmErr != nil {
			errMsg := fmt.Sprintf("Failed to Remove VM [%s] (ServiceID: %d): %v", vm.VmID, vm.ServiceID, rmErr)
			cblogger.Error(errMsg)
			removeErrors = append(removeErrors, errMsg)
			loggingError(callLogInfo, rmErr)
			continue
		}
		loggingInfo(callLogInfo, start)

		cblogger.Infof("Successfully removed VM [%s] from NLB", vm.VmID)
		removedCount++

		// Wait between removals to prevent rate limiting
		time.Sleep(time.Second * 3)
	}

	// Check if any errors occurred
	if len(removeErrors) > 0 {
		newErr := fmt.Errorf("Removed %d/%d VMs, errors: %s", removedCount, vmCount, strings.Join(removeErrors, "; "))
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return removedCount, newErr
	}

	cblogger.Infof("Successfully removed all %d VM(s) from NLB [%s]", removedCount, nlbIID.SystemId)
	return removedCount, nil
}

// getNLBVMList retrieves all VMs attached to the specified NLB
// Returns a list of LbServer structs and the total count
func (nlbHandler *KTVpcNLBHandler) getNLBVMList(nlbIID irs.IID) ([]ktvpclb.LbServer, int, error) {
	cblogger.Info("KT Cloud VPC Driver: called getNLBVMList()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", nlbIID.SystemId, "getNLBVMList()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID: empty string")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, 0, newErr
	}

	// Verify NLB exists
	_, getErr := nlbHandler.getKtNlbInfo(nlbIID.SystemId)
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud NLB info: %w", getErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, 0, newErr
	}

	// Get all VMs currently in the NLB
	var nlbVmList []ktvpclb.LbServer
	var vmCount int

	listLbServerOpts := ktvpclb.ListOpts{
		NlbID: nlbIID.SystemId, // Required
	}

	// Paginate through all servers in the NLB
	start := call.Start()
	err := ktvpclb.ListLbServer(nlbHandler.NLBClient, listLbServerOpts).EachPage(func(page pagination.Page) (bool, error) {
		servers, err := ktvpclb.ExtractLbServers(page)
		if err != nil {
			newErr := fmt.Errorf("Failed to extract servers: %w", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return false, newErr
		}

		nlbVmList = append(nlbVmList, servers...)
		vmCount += len(servers)

		// Continue pagination
		return true, nil
	})
	loggingInfo(callLogInfo, start)

	if err != nil {
		newErr := fmt.Errorf("Pagination failed: %w", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, 0, newErr
	}

	cblogger.Infof("Found %d VM(s) in NLB [%s]", vmCount, nlbIID.SystemId)

	time.Sleep(time.Second * 1)
	// To prevent the error: "Unable to execute API command listTags due to ratelimit timeout"

	return nlbVmList, vmCount, nil
}
