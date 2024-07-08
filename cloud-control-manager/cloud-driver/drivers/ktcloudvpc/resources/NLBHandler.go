// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI Team, 2024.02.

package resources

import (
	"fmt"
	"strings"
	"strconv"
	"time"
	// "github.com/davecgh/go-spew/spew"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	ktvpcsdk 	"github.com/cloud-barista/ktcloudvpc-sdk-go"
	ktvpclb 	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/loadbalancer/v1/loadbalancers"
	staticnat 	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/extensions/layer3/staticnat"
	rules       "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/extensions/fwaas_v2/rules"
	nim 		"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ktcloudvpc/resources/info_manager/nlb_info_manager"
)

type KTVpcNLBHandler struct {
	RegionInfo     idrv.RegionInfo
	VMClient       *ktvpcsdk.ServiceClient
	NetworkClient  *ktvpcsdk.ServiceClient
	NLBClient  	   *ktvpcsdk.ServiceClient
}

const (
	DefaultNLBOption 		string  = "roundrobin" // NLBOption : roundrobin / leastconnection / leastresponse / sourceiphash / 
	DefaultHealthCheckURL	string  = "abc.kt.com"
	NlbSubnetName			string  = "NLB-SUBNET" // Subnet for NLB
	DefaultVMGroupPort 		string  = "80"
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

	vpcHandler := KTVpcVPCHandler{
		RegionInfo: 	nlbHandler.RegionInfo,
		NetworkClient:  nlbHandler.NetworkClient, // Required!!
	}
	OsNetId, getError := vpcHandler.getOsNetworkIdWithTierName(NlbSubnetName)
	if getError != nil {
		newErr := fmt.Errorf("Failed to Get the OsNetwork ID of the 'NLB-Subnet' : [%v]", getError)
		cblogger.Error(newErr.Error())
		return irs.NLBInfo{}, newErr
	} else {
		cblogger.Infof("# OsNetwork ID of NLB-SUBNET : %s", OsNetId)
	}

	createOpts := ktvpclb.CreateOpts{
		Name:           	nlbReqInfo.IId.NameId,  			// Required
		ZoneID:         	nlbHandler.RegionInfo.Zone, 		// Required
		NlbOption: 			DefaultNLBOption,					// Required
		ServiceIP: 			"",									// Required. KT Cloud Virtual IP. $$$ In case of an empty value(""), it is newly created.
		ServicePort: 		nlbReqInfo.Listener.Port,			// Required
		ServiceType: 		nlbReqInfo.Listener.Protocol,		// Required
		HealthCheckType: 	nlbReqInfo.HealthChecker.Protocol,  // Required
		HealthCheckURL: 	DefaultHealthCheckURL,				// URL when the HealthCheckType (above) is 'http' or 'https'.
		NetworkID: 			OsNetId, 							// Required. Caution!!) Not Tier 'ID' but 'OsNetworkID' of the Tier!!
	}

	start := call.Start()
	resp, err := ktvpclb.Create(nlbHandler.NLBClient, createOpts).Extract() // Not 'NetworkClient'
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New NLB. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	loggingInfo(callLogInfo, start)
	cblogger.Infof("# New NLBId : %s", resp.Createnlbresponse.NLBId)

	cblogger.Info("\n### Creating New NLB!!")
	time.Sleep(time.Second * 10)

	ktNLB, err := nlbHandler.getKTCloudNlbWithName(nlbReqInfo.IId.NameId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	nlbIID := irs.IID{SystemId: strconv.Itoa(ktNLB.NlbID)}

	// cblogger.Info("\n\n### ktNLB : ")
	// spew.Dump(ktNLB)
	// cblogger.Info("\n")

	if countVMs(nlbReqInfo.VMGroup) > 0 {
		_, addErr := nlbHandler.AddVMs(nlbIID, nlbReqInfo.VMGroup.VMs)
		if addErr != nil {
			newErr := fmt.Errorf("Failed to Add the VMGroup VMs!! [%v]", addErr)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return irs.NLBInfo{}, newErr
		}
	}
	
	staticNatId, publicIp, natErr := nlbHandler.createStaticNatForNLB(ktNLB)
	if natErr != nil {
		newErr := fmt.Errorf("Failed to Add the VMGroup VMs!! [%v]", natErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	
	var keyValueList []irs.KeyValue
	keyValueList = append(keyValueList, irs.KeyValue{
		Key: 	"StaticNatID", 
		Value: 	staticNatId,
	})
	keyValueList = append(keyValueList, irs.KeyValue{
		Key: 	"PublicIP", 
		Value: 	publicIp,
	})

	// Register SecurityGroupInfo to DB
	providerName := "KTVPC"
	nlbDbInfo, regErr := nim.RegisterNlb(strconv.Itoa(ktNLB.NlbID), providerName, keyValueList)
	if regErr != nil {
		cblogger.Error(regErr)
		return irs.NLBInfo{}, regErr
	}
	cblogger.Infof(" === NLB Info to Register to DB : [%v]", nlbDbInfo)

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

	nlbInfo, getErr := nlbHandler.GetNLB(nlbIID)
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Get New NLB Info : [%v]", getErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	return nlbInfo, nil
}

func (nlbHandler *KTVpcNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", "ListNLB()", "ListNLB()")

	if strings.EqualFold(nlbHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Zone Info!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	
	listOpts := ktvpclb.ListOpts{
		ZoneID: nlbHandler.RegionInfo.Zone,
	}
	start := call.Start()
	firstPage, err := ktvpclb.List(nlbHandler.NLBClient, listOpts).FirstPage() // Not 'NetworkClient', Not 'AllPages()' 
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB List from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}	
	nlbList, err := ktvpclb.ExtractLoadBalancers(firstPage)
	if err != nil {
		newErr := fmt.Errorf("Failed to Extract NLB List : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	loggingInfo(callLogInfo, start)

	// cblogger.Info("\n\n# nlbList from KT Cloud VPC : ")
	// spew.Dump(nlbList)
	// cblogger.Info("\n")

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
	
	ktNLB, err := nlbHandler.getKTCloudNlb(nlbIID.SystemId)
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
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	ktNLB, err := nlbHandler.getKTCloudNlb(nlbIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}
	
	// Get NLB PublicIP Info from DB
	nlbDbInfo, getSGErr := nim.GetNlb(nlbIID.SystemId)
	if getSGErr != nil {
		cblogger.Debug(getSGErr)
		// return irs.VMInfo{}, getSGErr
	}

	var staticNatId string
	var publicIp string
	if countNlbKvList(*nlbDbInfo) > 0 {
		for _, kv := range nlbDbInfo.KeyValueInfoList {
			if kv.Key == "StaticNatID" {
				staticNatId = kv.Value				
			}
			if kv.Key == "PublicIP" {
				publicIp = kv.Value
			}
		}
	}

	vmHandler := KTVpcVMHandler{
		RegionInfo: 	nlbHandler.RegionInfo,
		VMClient:   	nlbHandler.VMClient,
		NetworkClient:  nlbHandler.NetworkClient, // Need!!
	}

	if !strings.EqualFold(publicIp, "") {
		// Delete FirewallRules
		_, dellFwErr := vmHandler.removeFirewallRule(publicIp, ktNLB.ServiceIP)
		if dellFwErr != nil {
			cblogger.Error(dellFwErr.Error())
			loggingError(callLogInfo, dellFwErr)
			return false, dellFwErr
		}

		// Delete Static NAT
		if !strings.EqualFold(staticNatId, "") {
			cblogger.Info("Deleting the Static NAT of the NLB!!")		
			err := staticnat.Delete(nlbHandler.NetworkClient, staticNatId).ExtractErr() // NetworkClient
			if err != nil {
				cblogger.Error(err.Error())
				return false, err
			}
			time.Sleep(time.Second * 3)
		}

		// Delete PublicIP
		_, dellIpErr := vmHandler.removePublicIP(publicIp)
		if dellIpErr != nil {
			cblogger.Error(dellIpErr.Error())
			loggingError(callLogInfo, dellIpErr)
			return false, dellIpErr
		}
	} else {
		cblogger.Info("The NLB doesn't have a Pulbic IP!! Waitting for Deletion!!")
	}

	// Delete the NLB info from DB
	_, unRegErr := nim.UnRegisterNlb(nlbIID.SystemId)
	if unRegErr != nil {
		cblogger.Debug(unRegErr.Error())
		loggingError(callLogInfo, unRegErr)
		// return irs.Failed, unRegErr
	}	

	deleteOpts := ktvpclb.DeleteOpts{
		NlbID: 	nlbIID.SystemId,
	}
	start := call.Start()
	delErr := ktvpclb.Delete(nlbHandler.NLBClient, deleteOpts).ExtractErr()
	if delErr != nil {
		newErr := fmt.Errorf("Failed to Delete the NLB : [%v]", delErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}
	loggingInfo(callLogInfo, start)

	return true, nil
}

//------ Frontend Control
func (nlbHandler *KTVpcNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {

	return irs.ListenerInfo{}, fmt.Errorf("Does not support yet!!")
}

//------ Backend Control
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

	type VMInfo struct {
		VmID 		string
		PrivateIP 	string
	}

	vmHandler := KTVpcVMHandler{
		RegionInfo: 	nlbHandler.RegionInfo,
		VMClient:   	nlbHandler.VMClient,
		NetworkClient:  nlbHandler.NetworkClient, // Need!!
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
				return irs.VMGroupInfo{}, newErr
			}
			vmInfoList = append(vmInfoList, vmInfo)

			time.Sleep(time.Second * 1)
			// To Prevent API timeout error
		}
	} else {
		newErr := fmt.Errorf("Failded to Find any VM NameId to Add to the NLB!!")
		cblogger.Error(newErr.Error())
		return irs.VMGroupInfo{}, newErr
	}

	for _, vm := range vmInfoList {
		addOpts := ktvpclb.AddServerOpts{
			NlbID:           	nlbIID.SystemId,  			// Required
			VMID:				vm.VmID,					// Required
			IPAddress: 			vm.PrivateIP,				// Required
			PublicPort: 		DefaultVMGroupPort,			// Required
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

		cblogger.Info("\n### Adding the VM to the NLB!!")
			// cblogger.Infof("# New NLBId : %s", resp.Createnlbresponse.NLBId)
		time.Sleep(time.Second * 5)

		// cblogger.Info("\n\n### resp : ")
		// spew.Dump(resp)
		// cblogger.Info("\n")
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

	type VMInfo struct {
		VmID 		string
		PrivateIP 	string
	}

	vmHandler := KTVpcVMHandler{
		RegionInfo: 	nlbHandler.RegionInfo,
		VMClient:   	nlbHandler.VMClient,
		NetworkClient:  nlbHandler.NetworkClient, // Need!!
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
			ServiceID:			serviceId, // Required. Not VMID
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

		cblogger.Info("\n### Removing the VM from the NLB!!")
		time.Sleep(time.Second * 5)
		
		// cblogger.Info("\n\n### resp : ")
		// spew.Dump(resp)
		// cblogger.Info("\n")
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

	listLbServerOpts := ktvpclb.ListOpts{
		NlbID: 	nlbIID.SystemId,  			// Required
	}
	start := call.Start()
	firstPage, err := ktvpclb.ListLbServer(nlbHandler.NLBClient, listLbServerOpts).FirstPage() // Not 'NetworkClient', Not 'AllPages()' 
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB List from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.HealthInfo{}, newErr
	}	
	nlbVmList, err := ktvpclb.ExtractLbServers(firstPage)
	if err != nil {
		newErr := fmt.Errorf("Failed to Extract NLB List : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.HealthInfo{}, newErr
	}
	loggingInfo(callLogInfo, start)

	// cblogger.Info("\n\n# nlb VM List from KT Cloud VPC : ")
	// spew.Dump(nlbVmList)
	// cblogger.Info("\n")

	time.Sleep(time.Second * 1) // Before 'return'
	// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"

	if len(nlbVmList) < 1 {
		cblogger.Info("# NLB VM does Not Exist!!")
		return irs.HealthInfo{}, nil // Not Return Error
	}

	var allVMs []irs.IID
	var healthVMs []irs.IID
	var unHealthVMs []irs.IID

	vmHandler := KTVpcVMHandler{
		RegionInfo: 	nlbHandler.RegionInfo,
		VMClient:   	nlbHandler.VMClient,
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
		Protocol: 	nlb.ServiceType,
		IP: 		nlb.ServiceIP,
		Port: 		nlb.ServicePort,
		// DNSName:	"NA",
		// CspID: 		"NA",
	}
	listenerKVList := []irs.KeyValue{
		// {Key: "NLB_DomainName", Value: *nlb.DomainName},
	}
	listenerInfo.KeyValueList = listenerKVList
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
		Protocol: 	nlb.HealthCheckType,
		Port:     	nlb.ServicePort,
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
	
	ktNLB, err := nlbHandler.getKTCloudNlb(nlbId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	listLbServerOpts := ktvpclb.ListOpts{
		NlbID: 	nlbId,  			// Required
	}
	start := call.Start()
	firstPage, err := ktvpclb.ListLbServer(nlbHandler.NLBClient, listLbServerOpts).FirstPage() // Not 'NetworkClient', Not 'AllPages()' 
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB List from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}	
	nlbVmList, err := ktvpclb.ExtractLbServers(firstPage)
	if err != nil {
		newErr := fmt.Errorf("Failed to Extract NLB List : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	loggingInfo(callLogInfo, start)

	// cblogger.Info("\n\n# nlb VM List from KT Cloud VPC : ")
	// spew.Dump(nlbVmList)
	// cblogger.Info("\n")
	
	vmIIds := []irs.IID{}
	keyValueList := []irs.KeyValue{}
	
	if len(nlbVmList) < 1 {
		cblogger.Debug("# NLB VM does Not Exist!!")
		return irs.VMGroupInfo{}, nil // Not Return Error
	}

	vmGroupInfo := irs.VMGroupInfo{
		Protocol: 	ktNLB.ServiceType, // Caution!!
		Port: 		nlbVmList[0].PublicPort, // In case, Any VM exists
		// CspID:    	"NA",
	}		

	vmHandler := KTVpcVMHandler{
		RegionInfo: nlbHandler.RegionInfo,
		VMClient:  	nlbHandler.VMClient,
	}
	for _, vm := range nlbVmList {
		vmName, err := vmHandler.getVmNameWithId(vm.VmID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the VM Name with the VM ID : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.VMGroupInfo{}, newErr
		}

		vmIIds = append(vmIIds, irs.IID{
			NameId:   vmName,
			SystemId: vm.VmID,
		})

		keyValueList = append(keyValueList, irs.KeyValue{
			Key: 	vmName + "_ServiceId",
			Value: 	strconv.Itoa(vm.ServiceID),		
		})

		time.Sleep(time.Second * 1)
		// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"
	}
	vmGroupInfo.VMs = &vmIIds
	vmGroupInfo.KeyValueList = keyValueList
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
	
	listLbServerOpts := ktvpclb.ListOpts{
		NlbID: 	nlbId,  			// Required
	}
	start := call.Start()
	firstPage, err := ktvpclb.ListLbServer(nlbHandler.NLBClient, listLbServerOpts).FirstPage() // Not 'NetworkClient', Not 'AllPages()' 
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB List from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}	
	nlbVmList, err := ktvpclb.ExtractLbServers(firstPage)
	if err != nil {
		newErr := fmt.Errorf("Failed to Extract NLB List : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}
	loggingInfo(callLogInfo, start)

	// cblogger.Info("\n\n# nlb VM List from KT Cloud VPC : ")
	// spew.Dump(nlbVmList)
	// cblogger.Info("\n")

	if len(nlbVmList) < 1 {		
		newErr := fmt.Errorf("Failed to Find Any VM on the NLB!!",)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	
	var serviceID string
	for _, vm := range nlbVmList {
		if strings.EqualFold(vm.VmID, vmId) {
		serviceID = strconv.Itoa(vm.ServiceID)
		break
		}
	}

	if strings.EqualFold(serviceID, "") {
		newErr := fmt.Errorf("Failed to Find the Service ID with the VM ID!!",)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	return serviceID, nil
}

func (nlbHandler *KTVpcNLBHandler) mappingNlbInfo(nlb *ktvpclb.LoadBalancer) (irs.NLBInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called mappingNlbInfo()")

	// cblogger.Info("\n\n### nlb : ")
	// spew.Dump(nlb)

	// vpcId, subnetId, publicIp, _, err := vmHandler.getNetIDsWithPrivateIP(vmInfo.PrivateIP)
	// if err != nil {
	// 	newErr := fmt.Errorf("Failed to Get PortForwarding Info. [%v]", err)
	// 	cblogger.Error(newErr.Error())
	// 	return irs.VMInfo{}, newErr
	// }
	// vmInfo.VpcIID.SystemId	  = vpcId
	// vmInfo.SubnetIID.SystemId = subnetId // Caution!!) Need modification. Not Tier 'ID' but 'OsNetworkID' to Create VM through REST API!!
	// vmInfo.PublicIP			  = publicIp


	vpcHandler := KTVpcVPCHandler{
		RegionInfo:    		nlbHandler.RegionInfo,
		NetworkClient:     	nlbHandler.NetworkClient,
	}

	vpcId, err := vpcHandler.getVPCIdWithOsNetworkID(nlb.NetworkID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get PortForwarding Info. [%v]", err)
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
			SystemId: vpcId,
		},
		Type:         "PUBLIC",
		Scope:        "REGION",
	}

	keyValueList := []irs.KeyValue{
		{Key: "NLB_Method", Value: nlb.NlbOption},
		{Key: "NLB_State", Value: nlb.State},
		{Key: "NLB_ServiceIP", Value: nlb.ServiceIP},
		{Key: "NLB_ServicePort", Value: nlb.ServicePort},
		{Key: "ZoneName", Value: nlb.ZoneName},
	}
	nlbInfo.KeyValueList = keyValueList

	if !strings.EqualFold(nlb.ServiceIP, "") {
		listenerInfo, err := nlbHandler.getListenerInfo(nlb)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Listener Info : [%v]", err.Error())
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		}
		nlbInfo.Listener = listenerInfo
	}

	// Get NLB Info from DB
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
			newErr := fmt.Errorf("Failed to Get HealthChecker Info. frome the NLB. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		}
		nlbInfo.HealthChecker = healthCheckerInfo
	}

	vmGroupInfo, err := nlbHandler.getVMGroupInfo(strconv.Itoa(nlb.NlbID))
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VM Group Info with the NLB ID : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.NLBInfo{}, newErr
	}
	nlbInfo.VMGroup = vmGroupInfo
	return nlbInfo, nil
}

func (nlbHandler *KTVpcNLBHandler) getKTCloudNlb(nlbId string) (*ktvpclb.LoadBalancer, error) {
	cblogger.Info("KT Cloud VPC Driver: called getKTCloudNlb()")
	InitLog()
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", nlbId, "getKTCloudNlb()")

	if strings.EqualFold(nlbId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	listOpts := ktvpclb.ListOpts{
		ZoneID: nlbHandler.RegionInfo.Zone,
		NlbID: 	nlbId,
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
		newErr := fmt.Errorf("Failed to Find the NLB info with the ID on the zone!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	// cblogger.Info("\n\n### result.Listnlbsresponse : ")
	// spew.Dump(result.Listnlbsresponse)

	return &nlbList[0], nil
}

func (nlbHandler *KTVpcNLBHandler) getKTCloudNlbWithName(nlbName string) (*ktvpclb.LoadBalancer, error) {
	cblogger.Info("KT Cloud VPC Driver: called getKTCloudNlbWithName()")
	InitLog()
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", nlbName, "getKTCloudNlbWithName()")

	if strings.EqualFold(nlbName, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	listOpts := ktvpclb.ListOpts{
		ZoneID: nlbHandler.RegionInfo.Zone,
		Name:  	nlbName,
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
	// cblogger.Info("\n\n### result.Listnlbsresponse : ")
	// spew.Dump(result.Listnlbsresponse)

	return &nlbList[0], nil
}

func countVMs(vmGroup irs.VMGroupInfo) int {
    if vmGroup.VMs == nil {
        return 0
    }
    return len(*vmGroup.VMs)
}

func (nlbHandler *KTVpcNLBHandler) createStaticNatForNLB(ktNLB *ktvpclb.LoadBalancer) (string, string, error) {
	cblogger.Info("KT Cloud VPC Driver: called createStaticNatForNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Zone, "NETWORKLOADBALANCE", ktNLB.Name, "createStaticNatForNLB()")

	// Static NAT allocation is required after new Public IP creation to be connected from the outside of the network.
	// # Create a Public IP
	var publicIP string
	var publicIPId string
	var creatErr error
	var ok bool

	cblogger.Info("\n### Creating New Public IP!!")
	vmHandler := KTVpcVMHandler{
		RegionInfo: 	nlbHandler.RegionInfo,
		VMClient:   	nlbHandler.VMClient,
		NetworkClient:  nlbHandler.NetworkClient, // Need!!
	}
	if ok, publicIP, publicIPId, creatErr = vmHandler.createPublicIP(); !ok {
		newErr := fmt.Errorf("Failed to Create a PublicIP : [%v]", creatErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", "", newErr
	}
	cblogger.Infof("# New PublicIP : [%s]\n", publicIP)
	time.Sleep(time.Second * 1)

	// ### Set Static NAT
	createNatOpts := &staticnat.CreateOpts{
		PrivateIpAddr: 	ktNLB.ServiceIP,
		SubnetID: 		ktNLB.NetworkID,
		PublicIpID: 	publicIPId,
	}
	natResult, err := staticnat.Create(nlbHandler.NetworkClient, createNatOpts).ExtractInfo()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Static NAT : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", "", newErr
	}
	cblogger.Info("\n### Creating the Static NAT to the NLB!!")
	time.Sleep(time.Second * 3)
	// cblogger.Info("\n\n### natResult")
	// spew.Dump(natResult)
	cblogger.Infof("\n# Static NAT ID : [%s]", natResult.ID)

	// ### Set FireWall Rules ("Inbound" FireWall Rules)
	vpcHandler := KTVpcVPCHandler{
		RegionInfo: 	nlbHandler.RegionInfo,
		NetworkClient:  nlbHandler.NetworkClient, // Required!!
	}
	vpcId, err := vpcHandler.getVPCIdWithOsNetworkID(ktNLB.NetworkID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VPC ID with teh OsNetwork ID. [%v]", err)
		cblogger.Error(newErr.Error())
		return "", "", newErr
	}

	externalNetId, getErr := vpcHandler.getExtSubnetId( irs.IID{SystemId: vpcId})
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Get the VPC Info : [%v]", getErr)
		cblogger.Error(newErr.Error())
		return "", "", newErr
	} else {
		cblogger.Infof("# ExternalNet ID : %s", externalNetId)
	}

	cblogger.Info("### Start to Create Firewall 'inbound' Rules!!")

	destCIDR, err := ipToCidr24(ktNLB.ServiceIP) // Output format ex) "172.25.1.0/24"
	if err != nil {
		cblogger.Errorf("Failed to Get Dest Net Band : [%v]", err)			
		return "", "", err
	} else {
		cblogger.Infof("Dest CIDR : %s", destCIDR)
	}

	tierId, getError := vpcHandler.getTierIdWithOsNetworkId(vpcId, ktNLB.NetworkID)
	if getError != nil {
		newErr := fmt.Errorf("Failed to Get the OsNetwork ID of the Tier : [%v]", getError)
		cblogger.Error(newErr.Error())
		return "", "", newErr
	} else {
		cblogger.Infof("# Tier ID : %s", tierId)
	}

	protocol := ktNLB.ServiceType

	var convertedProtocol rules.Protocol
	if strings.EqualFold(protocol, "tcp") {
		convertedProtocol = rules.ProtocolTCP
	} else if strings.EqualFold(protocol, "udp") {
		convertedProtocol = rules.ProtocolUDP
	} else if strings.EqualFold(protocol, "icmp") {
		convertedProtocol = rules.ProtocolICMP
	}

	inboundFWOpts := &rules.InboundCreateOpts{
		SourceNetID: 		externalNetId, 			// ExternalNet
		PortFordingID: 		natResult.ID,			// Caution!!
		DestIPAdds: 	    destCIDR,				// Destination network band (10.1.1.0/24, etc.)					
		StartPort: 		    ktNLB.ServicePort,
		EndPort:   			ktNLB.ServicePort,
		Protocol:           convertedProtocol,
		DestNetID:			tierId,					// Tier ID
		Action:             rules.ActionAllow, 		// "allow"
	}

	fwResult, err := rules.Create(vmHandler.NetworkClient, inboundFWOpts).ExtractJobInfo() // Not ~.Extract()
	if err != nil {
		cblogger.Errorf("Failed to Create the FireWall 'inbound' Rules : [%v]", err)
		return "", "", err
	}
	// cblogger.Infof("\n# fwResult.JopID : [%s]", fwResult.JopID)
	// cblogger.Info("\n")

	cblogger.Info("### Waiting for FireWall 'inbound' Rules to be Created(600sec) !!")

	// To prevent - json: cannot unmarshal string into Go struct field AsyncJobResult.nc_queryasyncjobresultresponse.result of type job.JobResult
	time.Sleep(time.Second * 3)

	jobWaitErr := vmHandler.waitForAsyncJob(fwResult.JopID, 600000000000)
	if jobWaitErr != nil {
		cblogger.Errorf("Failed to Wait the Job : [%v]", jobWaitErr)			
		return "", "", jobWaitErr
	}					

	return natResult.ID, publicIP, nil
}

func countNlbKvList(nlb nim.NlbInfo) int {
    if nlb.KeyValueInfoList == nil {
        return 0
    }
    return len(nlb.KeyValueInfoList)
}
