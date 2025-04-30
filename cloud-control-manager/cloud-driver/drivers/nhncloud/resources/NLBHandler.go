// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI Team, 2022.07.
// by ETRI Team, 2024.04.

package resources

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	// "github.com/davecgh/go-spew/spew"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/loadbalancer/v2/listeners"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/loadbalancer/v2/loadbalancers"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/vpcsubnets"

	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/servers"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/loadbalancer/v2/monitors"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/loadbalancer/v2/pools"

	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/extensions/layer3/floatingips"
)

type NhnCloudNLBHandler struct {
	RegionInfo    idrv.RegionInfo
	VMClient      *nhnsdk.ServiceClient
	NetworkClient *nhnsdk.ServiceClient
}

const (
	PublicType          string = "shared"
	InternalType        string = "dedicated"
	DefaultWeight       int    = 1
	DefaultAdminStateUp bool   = true

	// NHN Cloud default value for Listener and Health Monitor
	DefaultConnectionLimit        int = 2000 // NHN Cloud Listener ConnectionLimit range : 1 ~ 60000 (Dedicated LB : 1 ~ 480000)
	DefaultKeepAliveTimeout       int = 300  // NHN Cloud Listener KeepAliveTimeout range : 0 ~ 3600
	DefaultHealthCheckerInterval  int = 30
	DefaultHealthCheckerTimeout   int = 5
	DefaultHealthCheckerThreshold int = 2
)

func (nlbHandler *NhnCloudNLBHandler) getRawNLB(iid irs.IID) (*loadbalancers.LoadBalancer, error) {
	if iid.SystemId != "" {
		return loadbalancers.Get(nlbHandler.NetworkClient, iid.SystemId).Extract()
	} else {
		listOpts := loadbalancers.ListOpts{
			Name: iid.NameId,
		}
		rawListAllPage, err := loadbalancers.List(nlbHandler.NetworkClient, listOpts).AllPages()
		if err != nil {
			return nil, err
		}
		list, err := loadbalancers.ExtractLoadBalancers(rawListAllPage)
		if err != nil {
			return nil, err
		}
		if len(list) == 1 {
			return &list[0], err
		}
		return nil, errors.New(fmt.Sprintf("NLB not found : %s", iid.NameId))
	}
}

// The Order to Create NHN NLB : NLB (w/ Subnet ID) -> Listener (w/ NLB ID) -> Pool (w/ Listener ID) -> HealthMonitor (w/ Pool ID)  -> VM Members (w/ Pool ID), NLB Public IP (w/ NLB VIP_Port ID)
func (nlbHandler *NhnCloudNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (irs.NLBInfo, error) {
	cblogger.Info("NHN Cloud Driver: called CreateNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbReqInfo.IId.NameId, "CreateNLB()")

	vpcHandler := NhnCloudVPCHandler{
		NetworkClient: nlbHandler.NetworkClient,
	}
	vpc, err := vpcHandler.getRawVPC(nlbReqInfo.VpcIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VPC. : [%s]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	var subnetID string
	if len(vpc.Subnets) > 0 {
		subnetID = vpc.Subnets[0].ID
	} else {
		newErr := fmt.Errorf("Failed to Get FirstSubnetId with VPC Name. : [%s]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	var nlbType string
	if strings.EqualFold(nlbReqInfo.Type, "") {
		nlbType = PublicType
	} else if strings.EqualFold(nlbReqInfo.Type, "PUBLIC") {
		nlbType = PublicType
	} else if strings.EqualFold(nlbReqInfo.Type, "INTERNAL") {
		nlbType = InternalType
	} else {
		newErr := fmt.Errorf("Invalid NLB Type required!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	if strings.EqualFold(nlbReqInfo.Scope, "GLOBAL") {
		newErr := fmt.Errorf("NHN Cloud NLB does Not support 'GLOBAL' Scope!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	callLogStart := call.Start()
	createOpts := loadbalancers.CreateOpts{
		Name:             nlbReqInfo.IId.NameId,
		Description:      "CB-NLB : " + nlbReqInfo.IId.NameId,
		VipSubnetID:      subnetID,
		AdminStateUp:     *nhnsdk.Enabled,
		LoadBalancerType: nlbType,
	}
	newNlb, err := loadbalancers.Create(nlbHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create NHN Cloud NLB : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	// Wait for provisioning to complete.('provisioningStatus' is "ACTIVE")
	_, err = nlbHandler.waitToGetNLBInfo(irs.IID{SystemId: newNlb.ID})
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait for Provisioning to Complete. : [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	// cblogger.Info("\n\n# New NLB on NHN Cloud : ")
	// spew.Dump(newNlb)

	newNlbIID := irs.IID{SystemId: newNlb.ID}

	newListener, err := nlbHandler.createListener(newNlb.ID, nlbReqInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Listener. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)

		// Clean up the Created New NLB in case of Creating Failure
		_, err := nlbHandler.cleanUpNLB(newNlbIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Clean up the NLB. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		} else {
			cblogger.Info("\n\n# Succeeded in Deleting the NLB.")
		}

		return irs.NLBInfo{}, newErr
	}
	// cblogger.Info("\n\n# New Listener : ")
	// spew.Dump(newListener)

	cblogger.Info("\n\n#### Waiting for Provisioning the New Listener!!")
	time.Sleep(25 * time.Second)

	newPool, err := nlbHandler.createPool(newListener.ID, nlbReqInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Pool. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)

		// Clean up the Created New NLB in case of Creating Failure
		_, err := nlbHandler.cleanUpNLB(newNlbIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Clean up the NLB. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		} else {
			cblogger.Info("\n\n# Succeeded in Deleting the NLB.")
		}

		return irs.NLBInfo{}, newErr
	}
	// cblogger.Info("\n\n# New Pool : ")
	// spew.Dump(newPool)

	cblogger.Info("\n\n#### Waiting for Provisioning the New Pool!!")
	time.Sleep(25 * time.Second)

	// newHealthMonitor, createErr := nlbHandler.createHealthMonitor(newPool.ID, nlbReqInfo)
	_, createErr := nlbHandler.createHealthMonitor(newPool.ID, nlbReqInfo)
	if createErr != nil {
		newErr := fmt.Errorf("Failed to Create HealthMonitor. [%v]", createErr.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)

		// Clean up the Created New NLB in case of Creating Failure
		_, err := nlbHandler.cleanUpNLB(newNlbIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Clean up the NLB. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		} else {
			cblogger.Info("\n\n# Succeeded in Deleting the NLB.")
		}

		return irs.NLBInfo{}, newErr
	}
	// cblogger.Info("\n\n# New Health Monitor : ")
	// spew.Dump(newHealthMonitor)

	cblogger.Info("\n\n#### Waiting for Provisioning the New Health Monitor!!")
	time.Sleep(25 * time.Second)

	// newMembers, createError := nlbHandler.createVMMembers(newPool.ID, nlbReqInfo.VMGroup)
	_, createError := nlbHandler.createVMMembers(newPool.ID, nlbReqInfo.VMGroup)
	if createError != nil {
		newErr := fmt.Errorf("Failed to Create NLB Pool Members. [%v]", createError.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)

		// Clean up the Created New NLB in case of Creating Failure
		_, err := nlbHandler.cleanUpNLB(newNlbIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Clean up the NLB. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		} else {
			cblogger.Info("\n\n# Succeeded in Deleting the NLB.")
		}

		return irs.NLBInfo{}, newErr
	}
	// cblogger.Info("\n\n# New Members : ")
	// spew.Dump(newMembers)

	newFloatingIp, err := nlbHandler.createPublicIP(newNlb.VipPortID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create PublicIP for the NLB. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	if !strings.EqualFold(newFloatingIp, "") {
		cblogger.Infof("\n\n# Succeeded in Creating New PublicIP for the NLB. : [%s]", newFloatingIp)
	}

	nlbInfo, err := nlbHandler.GetNLB(newNlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Created NLB Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	return nlbInfo, nil
}

func (nlbHandler *NhnCloudNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ListNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", "ListNLB()", "ListNLB()")

	callLogStart := call.Start()
	listOpts := loadbalancers.ListOpts{}
	allPages, err := loadbalancers.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN NLB list. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	nlbList, err := loadbalancers.ExtractLoadBalancers(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN NLB list. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	// cblogger.Info("\n\n# nlbList from NHNCLOUD : ")
	// spew.Dump(nlbList)

	var nlbInfoList []*irs.NLBInfo
	for _, nlb := range nlbList {
		nlbInfo, err := nlbHandler.mappingNlbInfo(nlb)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get NLB Info. : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return nil, newErr
		}
		nlbInfoList = append(nlbInfoList, &nlbInfo)
	}
	return nlbInfoList, nil
}

func (nlbHandler *NhnCloudNLBHandler) GetNLB(nlbIID irs.IID) (irs.NLBInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "GetNLB()")

	callLogStart := call.Start()
	nlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	nlbInfo, err := nlbHandler.mappingNlbInfo(*nlb)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB Info from NHN Cloud. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	return nlbInfo, nil
}

func (nlbHandler *NhnCloudNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called DeleteNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "DeleteNLB()")

	nlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	_, err = nlbHandler.deletePublicIP(nlb.ID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the PublicIP of the NLB. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	callLogStart := call.Start()
	delOpts := loadbalancers.DeleteOpts{Cascade: true} // Note : 'Cascade' will delete all children of the LB (Listeners, Monitors, etc).
	delErr := loadbalancers.Delete(nlbHandler.NetworkClient, nlb.ID, delOpts).ExtractErr()
	if delErr != nil {
		newErr := fmt.Errorf("Failed to Delete the NLB. : [%v]", delErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	return true, nil
}

func (nlbHandler *NhnCloudNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ChangeListener()")

	return irs.ListenerInfo{}, fmt.Errorf("NHN Cloud does not support ChangeListener() yet!!")
}

func (nlbHandler *NhnCloudNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ChangeVMGroupInfo()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "ChangeVMGroupInfo()")

	nlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	var oldVMGroupInfo irs.VMGroupInfo
	if !strings.EqualFold(nlbInfo.VMGroup.Protocol, "") {
		oldVMGroupInfo = nlbInfo.VMGroup
	} else {
		newErr := fmt.Errorf("VMGroup is not Available in the NLB Info!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	// In case, New VMGroup Protocol and Port number are the Same as the Old one.
	if strings.EqualFold(oldVMGroupInfo.Protocol, vmGroup.Protocol) && strings.EqualFold(oldVMGroupInfo.Port, vmGroup.Port) {
		return oldVMGroupInfo, nil
	}

	if len(*vmGroup.VMs) < 1 { // In case, the 'vmGroup' parameter does not contain VM IID value.
		vmGroup.VMs = oldVMGroupInfo.VMs
	}

	// newMembers, createErr := nlbHandler.createVMMembers(nlbInfo.VMGroup.CspID, vmGroup)
	_, createErr := nlbHandler.createVMMembers(nlbInfo.VMGroup.CspID, vmGroup)
	if createErr != nil {
		newErr := fmt.Errorf("Failed to Create NLB Pool Members. [%v]", createErr.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	// cblogger.Info("\n\n# New Members : ")
	// spew.Dump(newMembers)

	newVMGroupNlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	return newVMGroupNlbInfo.VMGroup, nil
}

func (nlbHandler *NhnCloudNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	cblogger.Info("NHN Cloud Driver: called AddVMs()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "AddVMs()")

	if len(*vmIIDs) < 1 {
		newErr := fmt.Errorf("Failded to Find any VM to Add to the VMGroup!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	nlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	var newVMsInfo irs.VMGroupInfo
	if !strings.EqualFold(nlbInfo.VMGroup.Protocol, "") {
		newVMsInfo.Protocol = nlbInfo.VMGroup.Protocol
	} else {
		newErr := fmt.Errorf("VMGroup Protocol is not Available in the NLB Info!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	if !strings.EqualFold(nlbInfo.VMGroup.Port, "") {
		newVMsInfo.Port = nlbInfo.VMGroup.Port
	} else {
		newErr := fmt.Errorf("VMGroup Port is not Available in the NLB Info!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	newVMsInfo.VMs = vmIIDs
	// newMembers, createErr := nlbHandler.createVMMembers(nlbInfo.VMGroup.CspID, newVMsInfo)
	_, createErr := nlbHandler.createVMMembers(nlbInfo.VMGroup.CspID, newVMsInfo)
	if createErr != nil {
		newErr := fmt.Errorf("Failed to Create NLB Pool Members. [%v]", createErr.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	// cblogger.Info("\n\n# New VM Members : ")
	// spew.Dump(newMembers)

	newVMGroupNlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	return newVMGroupNlbInfo.VMGroup, nil
}

func (nlbHandler *NhnCloudNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called RemoveVMs()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", "RemoveVMs()", "RemoveVMs()")

	if len(*vmIIDs) < 1 {
		newErr := fmt.Errorf("Failed to Find any VM to Remove!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	nlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	// Note : Cloud-Barista supports only this case => [ LB : Listener : Pool : Health Checker = 1 : 1 : 1 : 1 ]
	nhnPoolList, err := nlbHandler.getNhnPoolListWithListenerId(nlbInfo.Listener.CspID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Pool list with the Listener ID. [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	if len(nhnPoolList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any NHN Pool. [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	vmMembers := nlbInfo.VMGroup.VMs

	for _, member := range *vmMembers {
		for _, vmToDel := range *vmIIDs {
			if strings.EqualFold(member.NameId, vmToDel.NameId) {
				cblogger.Infof("\n\n#### Deleting VM [%s] from the VMGroup : ", vmToDel.NameId)
				_, err = nlbHandler.deleteVMMember(nhnPoolList[0].ID, vmToDel)
				if err != nil {
					newErr := fmt.Errorf("Failed to Delete the VM Member. [%v]", err)
					cblogger.Error(newErr.Error())
					LoggingError(callLogInfo, newErr)
					return false, newErr
				}
			}
		}
	}

	return true, nil
}

func (nlbHandler *NhnCloudNLBHandler) createListener(nlbId string, nlbReqInfo irs.NLBInfo) (listeners.Listener, error) {
	cblogger.Info("NHN Cloud Driver: called createListener()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbReqInfo.IId.NameId, "createListener()")

	if strings.EqualFold(nlbId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return listeners.Listener{}, newErr
	}

	listenerProtocol, err := getListenerProtocol(nlbReqInfo.Listener.Protocol)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Listener Protocol : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return listeners.Listener{}, newErr
	}

	portNum, err := strconv.Atoi(nlbReqInfo.Listener.Port)
	if err != nil {
		newErr := fmt.Errorf("Invalid Listener Port. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return listeners.Listener{}, newErr
	}
	if portNum < 1 || portNum > 65535 {
		newErr := fmt.Errorf("Invalid Listener Port.(Must be between 1 and 65535)")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return listeners.Listener{}, newErr
	}

	callLogStart := call.Start()
	createOpts := listeners.CreateOpts{
		Protocol:         listenerProtocol,
		Description:      nlbReqInfo.IId.NameId,
		Name:             nlbReqInfo.IId.NameId,
		LoadbalancerID:   nlbId,
		AdminStateUp:     *nhnsdk.Enabled,
		ConnLimit:        DefaultConnectionLimit,  // NHN Cloud Listener ConnectionLimit range : 1 ~ 60000 (Dedicated LB : 1 ~ 480000)
		KeepAliveTimeout: DefaultKeepAliveTimeout, // NHN Cloud Listener KeepAliveTimeout range : 0 ~ 3600
		ProtocolPort:     portNum,
	}
	listener, err := listeners.Create(nlbHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Listener. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return listeners.Listener{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)
	return *listener, nil
}

func (nlbHandler *NhnCloudNLBHandler) createPool(listenerId string, nlbReqInfo irs.NLBInfo) (*pools.Pool, error) {
	cblogger.Info("NHN Cloud Driver: called createPool()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", "createPool()", "createPool()")

	if strings.EqualFold(listenerId, "") {
		newErr := fmt.Errorf("Invalid Listener ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	poolProtocol, err := getPoolProtocol(nlbReqInfo.VMGroup.Protocol)
	if err != nil {
		newErr := fmt.Errorf("Invalid Pool Protocol!! : [%v]", err.Error())
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	portNum, err := strconv.Atoi(nlbReqInfo.VMGroup.Port)
	if err != nil {
		newErr := fmt.Errorf("Invalid vmGroup Port. [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	if portNum < 1 || portNum > 65535 {
		newErr := fmt.Errorf("Invalid vmGroup Port.(Must be between 1 and 65535)")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	callLogStart := call.Start()
	// # Note : NHN Cloud LBMethods on GoSDK(nhncloud-sdk-go)
	// LBMethodRoundRobin       LBMethod = "ROUND_ROBIN"
	// LBMethodLeastConnections LBMethod = "LEAST_CONNECTIONS"
	// LBMethodSourceIp         LBMethod = "SOURCE_IP"
	createOpts := pools.CreateOpts{
		ListenerID:  listenerId,               // required:"true"
		LBMethod:    pools.LBMethodRoundRobin, // required:"true"
		Protocol:    poolProtocol,             // required:"true". # Protocol of VM Member
		Description: nlbReqInfo.IId.NameId,
		MemberPort:  portNum,
	}
	newPool, err := pools.Create(nlbHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create NHN Cloud Pool. [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)
	return newPool, nil
}

func (nlbHandler *NhnCloudNLBHandler) createHealthMonitor(poolId string, nlbReqInfo irs.NLBInfo) (monitors.Monitor, error) {
	cblogger.Info("NHN Cloud Driver: called createHealthMonitor()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", "createHealthMonitor()", "createHealthMonitor()")

	if strings.EqualFold(poolId, "") {
		newErr := fmt.Errorf("Invalid Pool ID!!")
		cblogger.Error(newErr.Error())
		return monitors.Monitor{}, newErr
	}

	switch strings.ToUpper(nlbReqInfo.HealthChecker.Protocol) {
	case "TCP", "HTTP", "HTTPS":
		cblogger.Infof("\n# HealthChecker.Protocol : [%s]", strings.ToUpper(nlbReqInfo.HealthChecker.Protocol))
	default:
		return monitors.Monitor{}, fmt.Errorf("Invalid Health Monitor Type. (Must be 'TCP', 'HTTP' or 'HTTPS' for NHN Cloud.)") // According to the NHN Cloud API document
	}

	healthCheckerInterval := nlbReqInfo.HealthChecker.Interval
	healthCheckerTimeout := nlbReqInfo.HealthChecker.Timeout
	healthCheckerThreshold := nlbReqInfo.HealthChecker.Threshold

	if healthCheckerInterval == -1 {
		healthCheckerInterval = DefaultHealthCheckerInterval // 30 seconds
	}
	if healthCheckerTimeout == -1 {
		healthCheckerTimeout = DefaultHealthCheckerTimeout // 5 seconds
	}
	if healthCheckerThreshold == -1 {
		healthCheckerThreshold = DefaultHealthCheckerThreshold // 2 times
	}

	if healthCheckerInterval > 5000 || healthCheckerInterval < 1 {
		return monitors.Monitor{}, fmt.Errorf("Invalid HealthChecker Interval value. Must be a number between 1 and 5000") // According to the NHN Cloud LB console
	}
	// Ref) Interval : Status check interval

	if healthCheckerTimeout < 1 {
		return monitors.Monitor{}, fmt.Errorf("Invalid HealthChecker Timeout value. Must be a number greater than zero.")
	}
	if healthCheckerInterval < healthCheckerTimeout {
		return monitors.Monitor{}, fmt.Errorf("Invalid HealthChecker Timeout value. Must be less than the 'Interval' value.")
	}
	// Ref) Timeout : Maximum number of seconds for a Monitor to wait for a ping reply before it times out. The value must be less than the Interval(Delay) value.

	if healthCheckerThreshold > 10 || healthCheckerThreshold < 1 {
		return monitors.Monitor{}, fmt.Errorf("Invalid HealthChecker Threshold value. Must be a number between 1 and 10") // According to the NHN Cloud LB console
	}
	// Ref) Threshold (MaxRetries) : Number of permissible ping failures before changing the member's status to INACTIVE. Must be a number between 1 and 10.

	portNum, err := strconv.Atoi(nlbReqInfo.HealthChecker.Port)
	if err != nil {
		newErr := fmt.Errorf("Invalid HealthChecker Port. [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return monitors.Monitor{}, newErr
	}
	if portNum < 1 || portNum > 65535 {
		newErr := fmt.Errorf("Invalid HealthChecker Port.(Must be between 1 and 65535)")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return monitors.Monitor{}, newErr
	}

	callLogStart := call.Start()
	// Note : NHN Cloud HealthChecker Protocols : TCP, HTTP or HTTPS
	creatOpts := monitors.CreateOpts{
		PoolID:          poolId, // required:"true"
		HealthCheckPort: portNum,
		Delay:           healthCheckerInterval,                              // required:"true". Must be between 1 and 5000.
		MaxRetries:      healthCheckerThreshold,                             // required:"true"  Must be between 1 and 10.
		Timeout:         healthCheckerTimeout,                               // required:"true". Must be between 1 and 5000. Must be smaller than Interval time.
		Type:            strings.ToUpper(nlbReqInfo.HealthChecker.Protocol), // required:"true"
	}
	newMonitor, err := monitors.Create(nlbHandler.NetworkClient, &creatOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create HealthChecker. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return monitors.Monitor{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)
	return *newMonitor, nil
}

func (nlbHandler *NhnCloudNLBHandler) createVMMembers(poolId string, vmGroupInfo irs.VMGroupInfo) ([]pools.Member, error) {
	cblogger.Info("NHN Cloud Driver: called createVMMembers()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", poolId, "createVMMembers()")

	if strings.EqualFold(poolId, "") {
		newErr := fmt.Errorf("Invalid Pool ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	if len(*vmGroupInfo.VMs) < 1 {
		newErr := fmt.Errorf("Failed to Find any VM to Create the VMGroup!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	portNum, err := strconv.Atoi(vmGroupInfo.Port)
	if err != nil {
		newErr := fmt.Errorf("Invalid VMGroup Port Number. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	if portNum < 1 || portNum > 65535 {
		newErr := fmt.Errorf("Invalid VMGroup Port Number.(Must be between 1 and 65535)")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	var poolMembers []pools.Member
	for _, vmIId := range *vmGroupInfo.VMs {
		cblogger.Infof("\n\n#### Adding VM [%s] as a VMGroup Member : ", vmIId.NameId)

		privateIp, subnetId, err := nlbHandler.getNetInfoWithVMName(vmIId.NameId)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get Private IP and Subnet ID.")
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return nil, newErr
		}

		callLogStart := call.Start()
		creatOpts := pools.CreateMemberOpts{
			Weight:       DefaultWeight,
			AdminStateUp: DefaultAdminStateUp,
			SubnetID:     subnetId,
			Address:      privateIp,
			ProtocolPort: portNum,
		}
		createResult, err := pools.CreateMember(nlbHandler.NetworkClient, poolId, creatOpts).Extract()
		if err != nil {
			newErr := fmt.Errorf("Failed to Create NLB Member with the Pool ID. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return nil, newErr
		}
		LoggingInfo(callLogInfo, callLogStart)

		poolMembers = append(poolMembers, *createResult)

		cblogger.Info("\n\n#### Waiting for Provisioning the Pool Member!!")
		time.Sleep(25 * time.Second)

		// _, err = nlbHandler.WaitToGetVMMemberInfo(poolId, *&createResult.ID)   // Wait until 'provisioningStatus' is "ACTIVE"
		// if err != nil {
		// 	return nil, err
		// }
	}
	return poolMembers, nil
}

func (nlbHandler *NhnCloudNLBHandler) createPublicIP(nlbVipPortId string) (string, error) {
	cblogger.Info("NHN Cloud Driver: called createPublicIP()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbVipPortId, "createPublicIP()")

	if strings.EqualFold(nlbVipPortId, "") {
		newErr := fmt.Errorf("Invalid Vip Port ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	externVPCID, err := getPublicVPCInfo(nlbHandler.NetworkClient, "ID")
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VPC ID. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", err
	}

	callLogStart := call.Start()
	createOpts := floatingips.CreateOpts{
		FloatingNetworkID: externVPCID,
		PortID:            nlbVipPortId,
	}
	createResult, err := floatingips.Create(nlbHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create PublicIP for the New NLB. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}
	LoggingInfo(callLogInfo, callLogStart)
	// spew.Dump(createResult)

	newFloatingIp := createResult.FloatingIP
	return newFloatingIp, nil
}

func (nlbHandler *NhnCloudNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetVMGroupHealthInfo()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "GetVMGroupHealthInfo()")

	nlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthInfo{}, newErr
	}

	vmMemberList, err := nlbHandler.getNhnVMMembersWithPoolId(nlbInfo.VMGroup.CspID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN VM Member list. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		return irs.HealthInfo{}, newErr
	}

	var allVMs []irs.IID
	var healthVMs []irs.IID
	var unHealthVMs []irs.IID

	callLogStart := call.Start()
	for _, member := range *vmMemberList {
		vm, err := nlbHandler.getNhnVMWithPrivateIp(member.Address)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get NHN VM with the Private IP address. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			return irs.HealthInfo{}, newErr
		}

		allVMs = append(allVMs, irs.IID{NameId: vm.Name, SystemId: vm.ID}) // Caution : Not 'VM Member ID' but 'VM System ID'

		if strings.EqualFold(member.OperatingStatus, "ACTIVE") {
			cblogger.Infof("\n### [%s] is Healthy VM.", vm.Name)
			healthVMs = append(healthVMs, irs.IID{NameId: vm.Name, SystemId: vm.ID})
		} else {
			cblogger.Infof("\n### [%s] is Unhealthy VM.", vm.Name)
			unHealthVMs = append(unHealthVMs, irs.IID{NameId: vm.Name, SystemId: vm.ID}) // In case of "INACTIVE", ...
		}
	}
	LoggingInfo(callLogInfo, callLogStart)

	vmGroupHealthInfo := irs.HealthInfo{
		AllVMs:       &allVMs,
		HealthyVMs:   &healthVMs,
		UnHealthyVMs: &unHealthVMs,
	}
	return vmGroupHealthInfo, nil
}

func (nlbHandler *NhnCloudNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ChangeHealthCheckerInfo()")

	return irs.HealthCheckerInfo{}, fmt.Errorf("NHN Cloud does not support ChangeHealthCheckerInfo() yet!!")
}

func (nlbHandler *NhnCloudNLBHandler) getListenerInfo(listenerId string) (irs.ListenerInfo, error) {
	cblogger.Info("NHN Cloud Driver: called getListenerInfo()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", listenerId, "getListenerInfo()")

	if strings.EqualFold(listenerId, "") {
		newErr := fmt.Errorf("Invalid Listener ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.ListenerInfo{}, newErr
	}

	callLogStart := call.Start()
	listenerOptions := listeners.ListOpts{
		ID: listenerId,
	}
	allPages, err := listeners.List(nlbHandler.NetworkClient, &listenerOptions).AllPages()
	if err != nil {
		return irs.ListenerInfo{}, err
	}

	nhnListenerList, err := listeners.ExtractListeners(allPages)
	if len(nhnListenerList) < 1 {
		newErr := fmt.Errorf("Failed to Get Listener with the ID [%s] : [%v]", listenerId, err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.ListenerInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	listenerInfo, err := nlbHandler.mappingListenerInfo(nhnListenerList[0])
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Listener Info from NHN Listener. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.ListenerInfo{}, newErr
	}
	return listenerInfo, nil
}

// Waiting for Provisioning to Complete.
func (nlbHandler *NhnCloudNLBHandler) waitToGetNLBInfo(nlbIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called waitToGetNLBInfo()")

	curRetryCnt := 0
	maxRetryCnt := 240
	for {
		curRetryCnt++
		provisioningStatus, err := nlbHandler.getNlbProvisioningStatus(nlbIID)
		if err == nil {
			if strings.EqualFold(provisioningStatus, "ACTIVE") {
				return true, nil
			}
			if strings.EqualFold(provisioningStatus, "ERROR") {
				return false, fmt.Errorf("Failed to Create NLB. ProvisioningStatus : ERROR")
			}
		}
		time.Sleep(3 * time.Second)
		if curRetryCnt > maxRetryCnt {
			return false, fmt.Errorf("Failed to Create NLB. Exceeded maximum retry count %d", maxRetryCnt)
		}
	}
}

func (nlbHandler *NhnCloudNLBHandler) getNlbProvisioningStatus(nlbIID irs.IID) (string, error) {
	cblogger.Info("NHN Cloud Driver: called getNlbProvisioningStatus()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "getNlbProvisioningStatus()")

	nlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	var status string
	// Use Key/Value info of the nlbInfo.
	for _, keyInfo := range nlbInfo.KeyValueList {
		if strings.EqualFold(keyInfo.Key, "NLB_ProvisioningStatus") {
			status = keyInfo.Value
			break
		}
	}
	return status, nil
}

func (nlbHandler *NhnCloudNLBHandler) getNlbVipPortId(nlbSystemId string) (string, error) {
	cblogger.Info("NHN Cloud Driver: called getNlbVipPortId()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbSystemId, "getNlbVipPortId()")

	if strings.EqualFold(nlbSystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	callLogStart := call.Start()
	listOpts := loadbalancers.ListOpts{
		ID: nlbSystemId,
	}
	allPages, err := loadbalancers.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud NLB Pages. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	nlbList, err := loadbalancers.ExtractLoadBalancers(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud NLB list. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var vipPortId string
	if len(nlbList) > 0 {
		vipPortId = nlbList[0].VipPortID
	} else {
		newErr := fmt.Errorf("Failed to Get Any NHN Cloud NLB Info. with the NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}
	cblogger.Info("\n# VipPortId : " + vipPortId)
	return vipPortId, nil
}

func (nlbHandler *NhnCloudNLBHandler) getNlbPrivateIp(nlbSystemID string) (string, error) {
	cblogger.Info("NHN Cloud Driver: called getNlbPrivateIp()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbSystemID, "getNlbPrivateIp()")

	if strings.EqualFold(nlbSystemID, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	callLogStart := call.Start()
	listOpts := loadbalancers.ListOpts{
		ID: nlbSystemID,
	}
	allPages, err := loadbalancers.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud NLB Pages. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	nlbList, err := loadbalancers.ExtractLoadBalancers(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud NLB list. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var privateIp string
	if len(nlbList) > 0 {
		privateIp = nlbList[0].VipAddress
	} else {
		newErr := fmt.Errorf("Failed to Get Any NHN Cloud NLB Info. with the NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}
	cblogger.Info("\n# NLB Private IP : " + privateIp)
	return privateIp, nil
}

// Caution : 'vmMemberId' is not VM ID.
func (nlbHandler *NhnCloudNLBHandler) getVMMemberOperatingStatus(poolId string, vmMemberId string) (string, error) {
	cblogger.Info("NHN Cloud Driver: called getVMMemberOperatingStatus()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", vmMemberId, "getVMMemberOperatingStatus()")

	if strings.EqualFold(poolId, "") {
		newErr := fmt.Errorf("Invalid Pool ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}
	if strings.EqualFold(vmMemberId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	callLogStart := call.Start()
	nhnMember, err := nlbHandler.getNhnVMMember(poolId, vmMemberId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN VM Member!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	return *&nhnMember.OperatingStatus, nil
}

// Note : NHN Cloud Listener Protocols : TCP, HTTP, HTTPS or TERMINATED_HTTPS
func getListenerProtocol(protocol string) (listeners.Protocol, error) {
	cblogger.Info("NHN Cloud Driver: called getListenerProtocol()")

	if strings.EqualFold(protocol, string(listeners.ProtocolTCP)) {
		return listeners.ProtocolTCP, nil
	} else if strings.EqualFold(protocol, string(listeners.ProtocolHTTP)) {
		return listeners.ProtocolHTTP, nil
	} else if strings.EqualFold(protocol, string(listeners.ProtocolHTTPS)) {
		return listeners.ProtocolHTTPS, nil
	} else if strings.EqualFold(protocol, string(listeners.ProtocolTerminatedHTTPS)) {
		return listeners.ProtocolTerminatedHTTPS, nil
	}

	newErr := fmt.Errorf("NHN Listener supports only TCP, HTTP, HTTPS or TERMINATED_HTTPS protocol!!") // According to the NHN Cloud API document
	cblogger.Error(newErr.Error())
	return "", newErr
}

func getPoolProtocol(protocol string) (pools.Protocol, error) {
	cblogger.Info("NHN Cloud Driver: called getPoolProtocol()")

	if strings.EqualFold(protocol, string(pools.ProtocolTCP)) {
		return pools.ProtocolTCP, nil
	} else if strings.EqualFold(protocol, string(pools.ProtocolHTTP)) {
		return pools.ProtocolHTTP, nil
	} else if strings.EqualFold(protocol, string(pools.ProtocolHTTPS)) {
		return pools.ProtocolHTTPS, nil
	}

	newErr := fmt.Errorf("NHN Pool supports only TCP, HTTP or HTTPS protocol!!")
	cblogger.Error(newErr.Error())
	return "", newErr
}

func (nlbHandler *NhnCloudNLBHandler) getHealthMonitorInfo(healthMonitorId string) (irs.HealthCheckerInfo, error) {
	cblogger.Info("NHN Cloud Driver: called getHealthMonitorInfo()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", healthMonitorId, "getHealthMonitorInfo()")

	if strings.EqualFold(healthMonitorId, "") {
		newErr := fmt.Errorf("Invalid Health Monitor ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthCheckerInfo{}, newErr
	}

	callLogStart := call.Start()
	listOpts := monitors.ListOpts{
		ID: healthMonitorId,
	}
	allPages, err := monitors.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN HealthChecker with the ID. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthCheckerInfo{}, newErr
	}

	nhnMonitorlist, err := monitors.ExtractMonitors(allPages)
	if len(nhnMonitorlist) < 1 {
		newErr := fmt.Errorf("Failed to Get Any NHN HealthChecker with the ID. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthCheckerInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	healthCheckerInfo := nlbHandler.mappingMonitorInfo(nhnMonitorlist[0])

	return healthCheckerInfo, nil
}

func (nlbHandler *NhnCloudNLBHandler) getHealthMonitorListWithPoolId(poolId string) ([]irs.HealthCheckerInfo, error) {
	cblogger.Info("NHN Cloud Driver: called getHealthMonitorListWithPoolId()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", poolId, "getHealthMonitorListWithPoolId()")

	if strings.EqualFold(poolId, "") {
		newErr := fmt.Errorf("Invalid Pool ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	callLogStart := call.Start()
	listOpts := monitors.ListOpts{}
	allPages, err := monitors.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN HealthChecker list. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	nhnMonitorlist, err := monitors.ExtractMonitors(allPages)
	if len(nhnMonitorlist) < 1 {
		newErr := fmt.Errorf("Failed to Get Any NHN HealthChecker. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var healthCheckerInfoList []irs.HealthCheckerInfo

	for _, nhnMonitor := range nhnMonitorlist {
		for _, pool := range nhnMonitor.Pools {
			if pool.ID == poolId {
				healthCheckerInfo := nlbHandler.mappingMonitorInfo(nhnMonitor)
				healthCheckerInfoList = append(healthCheckerInfoList, healthCheckerInfo)
			}
		}
	}

	return healthCheckerInfoList, nil
}

func (nlbHandler *NhnCloudNLBHandler) getNhnPoolListWithListenerId(listenerId string) ([]pools.Pool, error) {
	cblogger.Info("NHN Cloud Driver: called getNhnPoolListWithListenerId()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", listenerId, "getNhnPoolListWithListenerId()")

	if strings.EqualFold(listenerId, "") {
		newErr := fmt.Errorf("Invalid Listener ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	listOpts := pools.ListOpts{}
	callLogStart := call.Start()
	allPages, err := pools.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Pool list. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	poolList, err := pools.ExtractPools(allPages)
	if len(poolList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any NHN Pool. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var nhnPoolList []pools.Pool

	for _, pool := range poolList {
		for _, listener := range pool.Listeners {
			if listener.ID == listenerId {
				nhnPoolList = append(nhnPoolList, pool)
			}
		}
	}

	return nhnPoolList, nil
}

func (nlbHandler *NhnCloudNLBHandler) getHealthMonitorInfoWithListenerId(listenerId string) (irs.HealthCheckerInfo, error) {
	cblogger.Info("NHN Cloud Driver: called getHealthMonitorInfoWithListenerId()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", listenerId, "getHealthMonitorInfoWithListenerId()")

	if strings.EqualFold(listenerId, "") {
		newErr := fmt.Errorf("Invalid Listener ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthCheckerInfo{}, newErr
	}

	nhnPoolList, err := nlbHandler.getNhnPoolListWithListenerId(listenerId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Pool list with the Listener ID. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthCheckerInfo{}, newErr
	}
	if len(nhnPoolList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any NHN Pool. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthCheckerInfo{}, newErr
	}

	// Note : Cloud-Barista supports only this case => [ LB : Listener : Pool : Health Checker = 1 : 1 : 1 : 1 ]
	monitorList, err := nlbHandler.getHealthMonitorListWithPoolId(nhnPoolList[0].ID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Health Monitor list with the ID. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthCheckerInfo{}, newErr
	}
	if len(monitorList) < 1 {
		newErr := fmt.Errorf("Failed to Get Health Monitor list. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthCheckerInfo{}, newErr
	}

	// Note : Cloud-Barista supports only this case => [ LB : Listener : Pool : Health Checker = 1 : 1 : 1 : 1 ]
	return monitorList[0], nil
}

func (nlbHandler *NhnCloudNLBHandler) getNetInfoWithVMName(vmName string) (string, string, error) {
	cblogger.Info("NHN Cloud Driver: called getNetInfoWithVMName()")

	if strings.EqualFold(vmName, "") {
		newErr := fmt.Errorf("Invalid VM Name!!")
		cblogger.Error(newErr.Error())
		return "", "", newErr
	}

	listOpts := servers.ListOpts{
		Limit: 200,
	}
	allPages, err := servers.List(nlbHandler.VMClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get HNH Server list. [%v]", err.Error())
		cblogger.Error(err.Error())
		return "", "", newErr
	}

	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get HNH Server list. [%v]", err.Error())
		cblogger.Error(err.Error())
		return "", "", newErr
	}

	var ipAddress string
	var subnetId string
	for _, server := range serverList {
		if strings.EqualFold(server.Name, vmName) {
			for _, subnet := range server.Addresses {
				for _, addr := range subnet.([]interface{}) {
					addrMap := addr.(map[string]interface{})
					if addrMap["OS-EXT-IPS:type"] == "fixed" { // In case of fixed IP (Private IP Address)
						ipAddress = addrMap["addr"].(string)
					}
				}
			}

			// Get Subnet, Network Interface Info
			port, err := getPortWithDeviceId(nlbHandler.NetworkClient, server.ID)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get HNH Port Info. [%v]", err.Error())
				cblogger.Error(err.Error())
				return "", "", newErr
			}
			if port != nil {
				if len(port.FixedIPs) > 0 {
					subnetId = port.FixedIPs[0].SubnetID
				}
			}
		}
	}

	return ipAddress, subnetId, nil
}

func (nlbHandler *NhnCloudNLBHandler) deleteVMMember(poolId string, vmIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called deleteVMMember()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", poolId, "deleteVMMember()")

	if strings.EqualFold(poolId, "") {
		newErr := fmt.Errorf("Invalid Pool ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	if strings.EqualFold(vmIID.NameId, "") {
		newErr := fmt.Errorf("Invalid VM NameId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	vmMemberList, err := nlbHandler.getNhnVMMembersWithPoolId(poolId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN VM Member list. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	for _, member := range *vmMemberList {
		vm, err := nlbHandler.getNhnVMWithPrivateIp(member.Address)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get NHN VM with the Private IP address. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return false, newErr
		}

		if strings.EqualFold(vmIID.NameId, vm.Name) {
			callLogStart := call.Start()
			err = pools.DeleteMember(nlbHandler.NetworkClient, poolId, member.ID).ExtractErr()
			if err != nil {
				newErr := fmt.Errorf("Failed to Delete the NHN VM Member. : [%v]", err.Error())
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return false, newErr
			}
			LoggingInfo(callLogInfo, callLogStart)
		}

	}

	cblogger.Info("\n\n#### Waiting for Deleting the Pool Member!!")
	time.Sleep(10 * time.Second)

	return true, nil
}

func (nlbHandler *NhnCloudNLBHandler) getVMGroupInfo(nlb loadbalancers.LoadBalancer) (irs.VMGroupInfo, error) {
	cblogger.Info("NHN Cloud Driver: called getVMGroupInfo()")

	if strings.EqualFold(nlb.Listeners[0].ID, "") {
		newErr := fmt.Errorf("Invalid Listener ID")
		cblogger.Error(newErr.Error())
		return irs.VMGroupInfo{}, newErr
	}
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlb.Listeners[0].ID, "getVMGroupInfo()")

	// Note : Cloud-Barista supports only this case => [ LB : Listener : Pool : Health Checker = 1 : 1 : 1 : 1 ]
	nhnPoolList, err := nlbHandler.getNhnPoolListWithListenerId(nlb.Listeners[0].ID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Pool list with the Listener ID. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	if len(nhnPoolList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any NHN Pool. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	vmGroupInfo := irs.VMGroupInfo{
		Protocol: nhnPoolList[0].Protocol,                 // Note : Protocol of the 'NHN Pool'
		Port:     strconv.Itoa(nhnPoolList[0].MemberPort), // Member's port for receiving. Deliver traffic to this port. Not Exits on API Manual.
		CspID:    nhnPoolList[0].ID,                       // Note : ID of the 'NHN Pool'
	}

	vmMemberList, err := nlbHandler.getNhnVMMembersWithPoolId(nhnPoolList[0].ID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN VM Members. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	var vmIIds []irs.IID
	for _, member := range *vmMemberList {
		vm, err := nlbHandler.getNhnVMWithPrivateIp(member.Address)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get NHN VM with the Private IP address. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMGroupInfo{}, newErr
		}

		vmIIds = append(vmIIds, irs.IID{
			NameId:   vm.Name,
			SystemId: vm.ID, // Caution : Not 'VM Member' ID
		})
	}
	vmGroupInfo.VMs = &vmIIds
	vmGroupInfo.KeyValueList = irs.StructToKeyValueList(nlb)
	return vmGroupInfo, nil
}

func (nlbHandler *NhnCloudNLBHandler) getNhnVMMembersWithPoolId(poolId string) (*[]pools.Member, error) {
	cblogger.Info("NHN Cloud Driver: called getNhnVMMembersWithPoolId()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", poolId, "getNhnVMMembersWithPoolId()")

	if strings.EqualFold(poolId, "") {
		newErr := fmt.Errorf("Invalid Pool ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	callLogStart := call.Start()
	lostOpts := pools.ListMembersOpts{}
	allPages, err := pools.ListMembers(nlbHandler.NetworkClient, poolId, &lostOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN VM Member pages. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	vmMemberList, err := pools.ExtractMembers(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN VM Member list. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	return &vmMemberList, nil
}

func (nlbHandler *NhnCloudNLBHandler) getNhnVMMember(poolId string, memberId string) (*pools.Member, error) {
	cblogger.Info("NHN Cloud Driver: called getNhnVMMember()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", memberId, "getNhnVMMember()")

	if strings.EqualFold(poolId, "") {
		newErr := fmt.Errorf("Invalid Pool ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	if strings.EqualFold(memberId, "") {
		newErr := fmt.Errorf("Invalid VM Member ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	callLogStart := call.Start()
	lostOpts := pools.ListMembersOpts{
		ID: memberId,
	}
	allPages, err := pools.ListMembers(nlbHandler.NetworkClient, poolId, &lostOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN VM Member pages. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	vmMemberList, err := pools.ExtractMembers(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN VM Member list. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	return &vmMemberList[0], nil
}

func (nlbHandler *NhnCloudNLBHandler) getNhnVMWithPrivateIp(privateIp string) (*servers.Server, error) {
	cblogger.Info("NHN Cloud Driver: called getNhnVMWithPrivateIp()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", privateIp, "getNhnVMWithPrivateIp()")

	if strings.EqualFold(privateIp, "") {
		newErr := fmt.Errorf("Invalid Private IP!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	callLogStart := call.Start()
	listOpts := servers.ListOpts{
		Limit: 200,
	}
	allPages, err := servers.List(nlbHandler.VMClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN VM Pages. [%v]", err.Error())
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get HNH VM list. [%v]", err.Error())
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var nhnVM servers.Server
	for _, server := range serverList {
		for _, subnet := range server.Addresses {
			for _, addr := range subnet.([]interface{}) {
				addrMap := addr.(map[string]interface{})
				if addrMap["OS-EXT-IPS:type"] == "fixed" { // In case of fixed IP (Private IP Address)
					if strings.EqualFold(addrMap["addr"].(string), privateIp) {
						nhnVM = server
					}
				}
			}
		}
	}

	return &nhnVM, nil
}

func (nlbHandler *NhnCloudNLBHandler) mappingNlbInfo(nhnNLB loadbalancers.LoadBalancer) (irs.NLBInfo, error) {
	cblogger.Info("NHN Cloud Driver: called mappingNlbInfo()")

	vpcId, err := nlbHandler.getVPCIdWithVpcsubnetId(nhnNLB.VipSubnetID)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.NLBInfo{}, err
	}

	var nlbType string
	if strings.EqualFold(nhnNLB.LoadBalancerType, PublicType) {
		nlbType = "PUBLIC"
	} else if strings.EqualFold(nhnNLB.LoadBalancerType, InternalType) {
		nlbType = "INTERNAL"
	}

	nlbInfo := irs.NLBInfo{
		IId: irs.IID{
			NameId:   nhnNLB.Name,
			SystemId: nhnNLB.ID,
		},
		VpcIID: irs.IID{
			SystemId: vpcId,
		},
		Type:  nlbType,
		Scope: "REGION",
	}

	//keyValueList := []irs.KeyValue{
	//	{Key: "NLB_ProvisioningStatus", Value: nhnNLB.ProvisioningStatus},
	//	{Key: "NLB_OperatingStatus", Value: nhnNLB.OperatingStatus},
	//	{Key: "Provider", Value: nhnNLB.Provider},
	//	{Key: "NLB_PrivateIp", Value: nhnNLB.VipAddress},
	//	{Key: "SubnetId", Value: nhnNLB.VipSubnetID},
	//	{Key: "VipPortId", Value: nhnNLB.VipPortID},
	//}

	if len(nhnNLB.Listeners) > 0 {
		publicIp, err := nlbHandler.getNlbPublicIP(nhnNLB.ID)
		if err != nil {
			cblogger.Error(err.Error())
			return irs.NLBInfo{}, err
		}

		listenerInfo, err := nlbHandler.getListenerInfo(nhnNLB.Listeners[0].ID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get Listener with the ID. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		}
		listenerInfo.IP = publicIp
		nlbInfo.Listener = listenerInfo

		//listenerKeyValue := irs.KeyValue{Key: "ListenerId", Value: nhnNLB.Listeners[0].ID}
		//keyValueList = append(keyValueList, listenerKeyValue)

		monitorInfo, err := nlbHandler.getHealthMonitorInfoWithListenerId(nhnNLB.Listeners[0].ID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get Listener with the ID. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		}
		nlbInfo.HealthChecker = monitorInfo

		//monitorKeyValue := irs.KeyValue{Key: "HealthCheckerId", Value: nlbInfo.HealthChecker.CspID}
		//keyValueList = append(keyValueList, monitorKeyValue)

		vmGroupInfo, err := nlbHandler.getVMGroupInfo(nhnNLB)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get VM Info with the NHH NLB info. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		}

		nlbInfo.VMGroup = vmGroupInfo

		//poolKeyValue := irs.KeyValue{Key: "PoolId", Value: nlbInfo.VMGroup.CspID} // Note : VMGroup.CspID => PoolId
		//keyValueList = append(keyValueList, poolKeyValue)
	}

	//nlbInfo.KeyValueList = keyValueList
	nlbInfo.KeyValueList = irs.StructToKeyValueList(nlbInfo)

	//nlbInfo.KeyValueList = append(nlbInfo.KeyValueList,
	//	irs.KeyValue{Key: "NLB_ProvisioningStatus", Value: nhnNLB.ProvisioningStatus},
	//	irs.KeyValue{Key: "NLB_OperatingStatus", Value: nhnNLB.OperatingStatus},
	//	irs.KeyValue{Key: "NLB_PrivateIp", Value: nhnNLB.VipAddress},
	//	irs.KeyValue{Key: "Provider", Value: nhnNLB.Provider},
	//	irs.KeyValue{Key: "SubnetId", Value: nhnNLB.VipSubnetID},
	//	irs.KeyValue{Key: "VipPortId", Value: nhnNLB.VipPortID},
	//)

	//if nlbInfo.Listener.CspID != "" {
	//	nlbInfo.KeyValueList = append(nlbInfo.KeyValueList,
	//		irs.KeyValue{Key: "ListenerId", Value: nlbInfo.Listener.CspID})
	//}
	//
	//if nlbInfo.HealthChecker.CspID != "" {
	//	nlbInfo.KeyValueList = append(nlbInfo.KeyValueList,
	//		irs.KeyValue{Key: "HealthCheckerId", Value: nlbInfo.HealthChecker.CspID})
	//}
	//
	//if nlbInfo.VMGroup.CspID != "" {
	//	nlbInfo.KeyValueList = append(nlbInfo.KeyValueList,
	//		irs.KeyValue{Key: "PoolId", Value: nlbInfo.VMGroup.CspID})
	//}

	return nlbInfo, nil
}

func (nlbHandler *NhnCloudNLBHandler) mappingListenerInfo(nhnListener listeners.Listener) (irs.ListenerInfo, error) {
	cblogger.Info("NHN Cloud Driver: called mappingListenerInfo()")

	listenerProtocol, err := getListenerProtocol(string(nhnListener.Protocol))
	if err != nil {
		newErr := fmt.Errorf("Invalid Listener Protocol!!")
		cblogger.Error(newErr.Error())
		return irs.ListenerInfo{}, newErr
	}

	listenerInfo := irs.ListenerInfo{
		Protocol: string(listenerProtocol),
		Port:     strconv.Itoa(nhnListener.ProtocolPort),
		CspID:    nhnListener.ID,
	}
	//
	//keyValueList := []irs.KeyValue{
	//	{Key: "AdminStateUp", Value: strconv.FormatBool(nhnListener.AdminStateUp)},
	//	{Key: "ConnectionLimit", Value: strconv.Itoa(nhnListener.ConnLimit)},
	//	{Key: "KeepaliveTimeout(Sec)", Value: strconv.Itoa(nhnListener.KeepaliveTimeout)},
	//}
	//listenerInfo.KeyValueList = keyValueList

	listenerInfo.KeyValueList = irs.StructToKeyValueList(nhnListener)

	return listenerInfo, nil
}

func (nlbHandler *NhnCloudNLBHandler) mappingMonitorInfo(nhnMonitor monitors.Monitor) irs.HealthCheckerInfo {
	cblogger.Info("NHN Cloud Driver: called mappingMonitorInfo()")

	healthCheckerInfo := irs.HealthCheckerInfo{
		Protocol:  nhnMonitor.Type,
		Port:      strconv.Itoa(nhnMonitor.HealthCheckPort),
		Interval:  nhnMonitor.Delay,
		Threshold: nhnMonitor.MaxRetries,
		Timeout:   nhnMonitor.Timeout,
		CspID:     nhnMonitor.ID,
	}

	healthCheckerInfo.KeyValueList = irs.StructToKeyValueList(healthCheckerInfo)

	//keyValueList := []irs.KeyValue{
	//	{Key: "AdminStateUp", Value: strconv.FormatBool(nhnMonitor.AdminStateUp)},
	//	{Key: "PoolId", Value: nhnMonitor.Pools[0].ID},
	//}
	//healthCheckerInfo.KeyValueList = keyValueList
	if len(nhnMonitor.Pools) > 0 {
		healthCheckerInfo.KeyValueList = append(healthCheckerInfo.KeyValueList, irs.KeyValue{Key: "PoolId", Value: nhnMonitor.Pools[0].ID})
	}
	healthCheckerInfo.KeyValueList = append(healthCheckerInfo.KeyValueList, irs.KeyValue{Key: "AdminStateUp", Value: strconv.FormatBool(nhnMonitor.AdminStateUp)})

	return healthCheckerInfo
}

func (nlbHandler *NhnCloudNLBHandler) getNlbPublicIP(nlbSystemID string) (string, error) {
	cblogger.Info("NHN Cloud Driver: called getNlbPublicIP()")

	privateIp, err := nlbHandler.getNlbPrivateIp(nlbSystemID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB VPC ID and VIP Port ID!! [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	// To Get Floating IP(Public IP) Address Info.
	listOpts := floatingips.ListOpts{
		FixedIP: privateIp,
	}
	allPages, err := floatingips.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get FloatingIP Pages!! [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	ipList, err := floatingips.ExtractFloatingIPs(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get FloatingIP List of the NLB!! [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	var floatingIp string
	if len(ipList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any FloatingIP Info of the NLB!! [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	} else {
		floatingIp = ipList[0].FloatingIP
	}
	cblogger.Info("\n# NLB Floating IP : " + floatingIp)

	return floatingIp, nil
}

func (nlbHandler *NhnCloudNLBHandler) deletePublicIP(nlbSystemId string) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called deletePublicIP()")

	vipPortId, err := nlbHandler.getNlbVipPortId(nlbSystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB VPC ID and VIP Port ID!! [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	// To Get Floating IP(Public IP) Address Info.
	listOpts := floatingips.ListOpts{
		PortID: vipPortId,
	}
	allPages, err := floatingips.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get FloatingIP Pages!! [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	ipList, err := floatingips.ExtractFloatingIPs(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get FloatingIP List!! [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	var floatingIpId string
	if len(ipList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any FloatingIP Info!! [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	} else {
		floatingIpId = ipList[0].ID
	}

	delErr := floatingips.Delete(nlbHandler.NetworkClient, floatingIpId).ExtractErr()
	if delErr != nil {
		newErr := fmt.Errorf("Failed to Delete the FloatingIP of the NLB!! [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	} else {
		cblogger.Info("\n# Succeeded in Deleting the FloatingIP of the NLB.")
	}

	return true, nil
}

// Clean up the Created New NLB in case of failure
func (nlbHandler *NhnCloudNLBHandler) cleanUpNLB(nlbIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called cleanUpNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "cleanUpNLB()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	cblogger.Info("\n\n#### Waiting for Provisioning to Delete the NLB!!")
	time.Sleep(20 * time.Second)

	callLogStart := call.Start()
	delOpts := loadbalancers.DeleteOpts{Cascade: true} // Note : 'Cascade' will delete all children of the LB (Listeners, Monitors, etc).
	delErr := loadbalancers.Delete(nlbHandler.NetworkClient, nlbIID.SystemId, delOpts).ExtractErr()
	if delErr != nil {
		newErr := fmt.Errorf("Failed to Delete the NLB. : [%v]", delErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	return true, nil
}

func (nlbHandler *NhnCloudNLBHandler) getVPCIdWithVpcsubnetId(vpcsubnetId string) (string, error) {
	cblogger.Info("NHN Cloud Driver: called getVPCIdWithVpcsubnetId()")

	if strings.EqualFold(vpcsubnetId, "") {
		newErr := fmt.Errorf("Invalid Vpcsubnet ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	vpcsubnet, err := vpcsubnets.Get(nlbHandler.NetworkClient, vpcsubnetId).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud VPC Subnet Info with the Vpcsubnet ID [%s] : %v", vpcsubnetId, err.Error())
		cblogger.Error(newErr.Error())
		return "", nil
	}
	VPCId := vpcsubnet.VPCID

	return VPCId, nil
}

func (NLBHandler *NhnCloudNLBHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	callLogInfo := getCallLogScheme(NLBHandler.RegionInfo.Zone, call.NLB, "nlbId", "ListIID()")

	start := call.Start()

	var iidList []*irs.IID

	listOpts := loadbalancers.ListOpts{}

	allPages, err := loadbalancers.List(NLBHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get disk information from NhnCloud!! : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return make([]*irs.IID, 0), newErr
	}

	allNlbs, err := loadbalancers.ExtractLoadBalancers(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get disk List from NhnCloud!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return make([]*irs.IID, 0), newErr
	}

	for _, nlb := range allNlbs {
		var iid irs.IID
		iid.SystemId = nlb.ID
		iid.NameId = nlb.Name

		iidList = append(iidList, &iid)
	}

	LoggingInfo(callLogInfo, start)

	return iidList, nil

}
