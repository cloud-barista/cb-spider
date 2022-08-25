package resources

import (
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/listeners"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/monitors"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/providers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type OpenStackNLBHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VMClient       *gophercloud.ServiceClient
	NetworkClient  *gophercloud.ServiceClient
	NLBClient      *gophercloud.ServiceClient
}

type NLBType string
type NLBScope string

const (
	NLBPublicType   NLBType  = "PUBLIC"
	NLBInternalType NLBType  = "INTERNAL"
	NLBGlobalType   NLBScope = "GLOBAL"
	NLBRegionType   NLBScope = "REGION"
)

//------ NLB Management
func (nlbHandler *OpenStackNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (createNLB irs.NLBInfo, createError error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region.Region, "NETWORKLOADBALANCE", nlbReqInfo.IId.NameId, "CreateNLB()")
	start := call.Start()
	// Check LoadBalancer Service
	err := nlbHandler.checkNLBClient()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}
	nlb, err := nlbHandler.createLoadBalancer(nlbReqInfo)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}
	defer func() {
		if createError != nil {
			_, cleanerErr := nlbHandler.CleanerNLB(irs.IID{
				SystemId: nlb.ID,
			})
			if cleanerErr != nil {
				createError = errors.New(fmt.Sprintf("%s and Failed to rollback err = %s", createError.Error(), cleanerErr.Error()))
			}
		}
	}()
	// No - BatchCreate!
	createPool, err := nlbHandler.createPool(nlbReqInfo, nlb.ID)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError.Error())
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}
	_, err = nlbHandler.createPoolHealthCheck(nlbReqInfo, createPool.ID)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError.Error())
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}
	_, err = nlbHandler.attachPoolMembers(*nlbReqInfo.VMGroup.VMs, nlbReqInfo.VMGroup.Port, createPool.ID)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError.Error())
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}
	_, err = nlbHandler.createListener(nlbReqInfo, nlb.ID, createPool.ID)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError.Error())
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}
	_, err = nlbHandler.AssociatePublicIP(nlb.VipPortID)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError.Error())
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}
	rawnlb, err := nlbHandler.getRawNLB(nlbReqInfo.IId)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError.Error())
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}
	info, err := nlbHandler.setterNLB(*rawnlb)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError.Error())
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}
func (nlbHandler *OpenStackNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region.Region, "NETWORKLOADBALANCE", "NLB", "ListNLB()")
	start := call.Start()
	// Check LoadBalancer Service
	err := nlbHandler.checkNLBClient()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List NLB. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	nlbList, err := nlbHandler.getRawNLBList()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List NLB. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	infoList := make([]*irs.NLBInfo, len(nlbList))
	for i, rawnlb := range nlbList {
		info, err := nlbHandler.setterNLB(rawnlb)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List NLB. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}
		infoList[i] = &info
	}
	LoggingInfo(hiscallInfo, start)
	return infoList, nil
}
func (nlbHandler *OpenStackNLBHandler) GetNLB(nlbIID irs.IID) (irs.NLBInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "GetNLB()")
	start := call.Start()
	// Check LoadBalancer Service
	err := nlbHandler.checkNLBClient()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get NLB. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.NLBInfo{}, getErr
	}

	rawnlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get NLB. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.NLBInfo{}, getErr
	}
	info, err := nlbHandler.setterNLB(*rawnlb)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get NLB. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.NLBInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}
func (nlbHandler *OpenStackNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "DeleteNLB()")
	start := call.Start()

	// Check LoadBalancer Service
	err := nlbHandler.checkNLBClient()
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete NLB. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}

	_, err = nlbHandler.CleanerNLB(nlbIID)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete NLB. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

//------ Frontend Control
func (nlbHandler *OpenStackNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "ChangeListener()")
	start := call.Start()
	// Check LoadBalancer Service
	err := nlbHandler.checkNLBClient()
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	rawNLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}

	if len(rawNLB.Listeners) < 1 {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = not Exist Listener"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}

	cbOnlyOneListener := rawNLB.Listeners[0]
	listenerId := cbOnlyOneListener.ID
	oldListener, err := nlbHandler.getRawListenerById(listenerId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	// Valid Check
	portInt, err := strconv.Atoi(listener.Port)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = invalid HealthChecker Port"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	if portInt < 1 || portInt > 65535 {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = invalid HealthChecker Port"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	if listener.Protocol != "" && !strings.EqualFold(strings.ToUpper(listener.Protocol), strings.ToUpper(oldListener.Protocol)) {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = changing the protocol of the Listener is not supported"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	// New Listener Create  // Empty DefaultPoolID
	newOpt := listeners.CreateOpts{
		Protocol:       listeners.Protocol(strings.ToUpper(oldListener.Protocol)),
		ProtocolPort:   portInt,
		Name:           oldListener.Name,
		LoadbalancerID: rawNLB.ID,
		AdminStateUp:   gophercloud.Enabled,
	}
	newListener, err := listeners.Create(nlbHandler.NLBClient, newOpt).Extract()
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	_, err = nlbHandler.waitingNLBListenerActive(newListener.ID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	// Delete OldListener
	err = listeners.Delete(nlbHandler.NLBClient, oldListener.ID).Err
	if err != nil {
		listeners.Delete(nlbHandler.NLBClient, newListener.ID)
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	_, err = nlbHandler.waitingNLBListenerDeleted(oldListener.ID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}

	// changer newListener DefaultPoolID
	updateOpts := listeners.UpdateOpts{
		DefaultPoolID: &oldListener.DefaultPoolID,
	}

	update, err := listeners.Update(nlbHandler.NLBClient, newListener.ID, updateOpts).Extract()
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	_, err = nlbHandler.waitingNLBListenerActive(update.ID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	rawNLB, err = nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	info, err := nlbHandler.getListenerInfo(*rawNLB)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}

//------ Backend Control
func (nlbHandler *OpenStackNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "ChangeVMGroupInfo()")
	start := call.Start()
	// Check LoadBalancer Service
	err := nlbHandler.checkNLBClient()
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	rawNLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	if len(rawNLB.Pools) < 1 {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = not Exist Listener"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	cbOnlyOnePool := rawNLB.Pools[0]
	poolId := cbOnlyOnePool.ID

	oldPool, err := nlbHandler.getRawPoolById(poolId)

	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	portInt, err := strconv.Atoi(vmGroup.Port)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	if portInt < 1 || portInt > 65535 {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = invalid VMGroup Port"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	if vmGroup.Protocol != "" && !strings.EqualFold(strings.ToUpper(vmGroup.Protocol), strings.ToUpper(oldPool.Protocol)) {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = changing the protocol of the VMGroup is not supported"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	oldVMGroup, err := nlbHandler.getVMGroup(*rawNLB)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	// not update
	if strings.EqualFold(vmGroup.Port, oldVMGroup.Port) {
		return oldVMGroup, nil
	}

	oldVMIIDs := *oldVMGroup.VMs

	for _, oldMember := range oldPool.Members {
		_, err = nlbHandler.detachPoolMemberByMemberID(oldMember.ID, poolId)
		if err != nil {
			return irs.VMGroupInfo{}, nil
		}
	}

	_, err = nlbHandler.attachPoolMembers(oldVMIIDs, vmGroup.Port, poolId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	vmGroupPortSticker := generateVMGroupPortDescription(portInt)

	_, err = nlbHandler.setPoolDescription(vmGroupPortSticker, poolId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	rawNLB, err = nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	newVMGroup, err := nlbHandler.getVMGroup(*rawNLB)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	LoggingInfo(hiscallInfo, start)
	return newVMGroup, nil
}
func (nlbHandler *OpenStackNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "AddVMs()")
	start := call.Start()
	// Check LoadBalancer Service
	err := nlbHandler.checkNLBClient()
	if err != nil {
		addErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = %s", err.Error()))
		cblogger.Error(addErr.Error())
		LoggingError(hiscallInfo, addErr)
		return irs.VMGroupInfo{}, addErr
	}

	rawnlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		addErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = %s", err.Error()))
		cblogger.Error(addErr.Error())
		LoggingError(hiscallInfo, addErr)
		return irs.VMGroupInfo{}, addErr
	}
	vmGroup, err := nlbHandler.getVMGroup(*rawnlb)
	if err != nil {
		addErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = %s", err.Error()))
		cblogger.Error(addErr.Error())
		LoggingError(hiscallInfo, addErr)
		return irs.VMGroupInfo{}, addErr
	}

	currentvmIIDs := *vmGroup.VMs
	for _, currentVMIID := range currentvmIIDs {
		for _, addvm := range *vmIIDs {
			if strings.EqualFold(currentVMIID.NameId, addvm.NameId) {
				return irs.VMGroupInfo{}, errors.New(fmt.Sprintf("already Exist vm %s", addvm.NameId))
			}
		}
	}
	_, err = nlbHandler.attachPoolMembers(*vmIIDs, vmGroup.Port, vmGroup.CspID)
	if err != nil {
		addErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = %s", err.Error()))
		cblogger.Error(addErr.Error())
		LoggingError(hiscallInfo, addErr)
		return irs.VMGroupInfo{}, addErr
	}
	updatednlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		addErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = %s", err.Error()))
		cblogger.Error(addErr.Error())
		LoggingError(hiscallInfo, addErr)
		return irs.VMGroupInfo{}, addErr
	}
	updatedVMGroup, err := nlbHandler.getVMGroup(*updatednlb)
	if err != nil {
		addErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = %s", err.Error()))
		cblogger.Error(addErr.Error())
		LoggingError(hiscallInfo, addErr)
		return irs.VMGroupInfo{}, addErr
	}
	LoggingInfo(hiscallInfo, start)
	return updatedVMGroup, nil
}
func (nlbHandler *OpenStackNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "AddVMs()")
	start := call.Start()
	// Check LoadBalancer Service
	err := nlbHandler.checkNLBClient()
	if err != nil {
		removeErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = %s", err.Error()))
		cblogger.Error(removeErr.Error())
		LoggingError(hiscallInfo, removeErr)
		return false, removeErr
	}
	rawnlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		removeErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = %s", err.Error()))
		cblogger.Error(removeErr.Error())
		LoggingError(hiscallInfo, removeErr)
		return false, removeErr
	}
	vmGroup, err := nlbHandler.getVMGroup(*rawnlb)
	if err != nil {
		removeErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = %s", err.Error()))
		cblogger.Error(removeErr.Error())
		LoggingError(hiscallInfo, removeErr)
		return false, removeErr
	}

	currentvmIIDs := *vmGroup.VMs

	for _, removeVM := range *vmIIDs {
		exist := false
		for _, currentVMIID := range currentvmIIDs {
			if strings.EqualFold(currentVMIID.NameId, removeVM.NameId) {
				exist = true
			}
		}
		if !exist {
			removeErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = not found vm %s ", removeVM.NameId))
			cblogger.Error(removeErr.Error())
			LoggingError(hiscallInfo, removeErr)
			return false, removeErr
		}
	}
	for _, removeVM := range *vmIIDs {
		_, err = nlbHandler.detachPoolMember(removeVM, vmGroup.CspID)
		if err != nil {
			removeErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = %s", err.Error()))
			cblogger.Error(removeErr.Error())
			LoggingError(hiscallInfo, removeErr)
			return false, removeErr
		}
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
func (nlbHandler *OpenStackNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "ChangeHealthCheckerInfo()")
	start := call.Start()
	// Check LoadBalancer Service
	err := nlbHandler.checkNLBClient()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to GetVMGroupHealthInfo NLB. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.HealthInfo{}, createErr
	}
	rawnlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to GetVMGroupHealthInfo. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.HealthInfo{}, getErr
	}
	if len(rawnlb.Pools) < 1 {
		getErr := errors.New(fmt.Sprintf("Failed to GetVMGroupHealthInfo. err = not Exist Pool"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.HealthInfo{}, getErr
	}
	cbOnlyOnePool := rawnlb.Pools[0]
	poolId := cbOnlyOnePool.ID
	rawPool, err := nlbHandler.getRawPoolById(poolId)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to GetVMGroupHealthInfo. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.HealthInfo{}, getErr
	}

	members, err := nlbHandler.getRawPoolMembersById(rawPool.ID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to GetVMGroupHealthInfo. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.HealthInfo{}, getErr
	}
	allVMs := make([]irs.IID, len(*members))
	healthVMs := make([]irs.IID, 0)
	unHealthVMs := make([]irs.IID, 0)
	for i, member := range *members {
		vm, err := nlbHandler.getRawVMByName(member.Name)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to GetVMGroupHealthInfo. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return irs.HealthInfo{}, getErr
		}
		vmIID := irs.IID{
			NameId:   vm.Name,
			SystemId: vm.ID,
		}
		allVMs[i] = vmIID
		if strings.EqualFold(strings.ToUpper(member.OperatingStatus), "ONLINE") {
			healthVMs = append(healthVMs, vmIID)
		} else if strings.EqualFold(strings.ToUpper(member.OperatingStatus), "DRAINING") {
			unHealthVMs = append(unHealthVMs, vmIID)
		} else {
			// OFFLINE DEGRADED ERROR NO_MONITOR 일 경우, HealthCheck 결과가 멤버에 제대로 갱신되지 않는 octavia 이슈일 수 있음
			getErr := errors.New(fmt.Sprintf("Failed to GetVMGroupHealthInfo. err = Unable to determine operating status of member. This openstack may not update the OperatingStatus"))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return irs.HealthInfo{}, getErr
		}
	}
	LoggingInfo(hiscallInfo, start)
	return irs.HealthInfo{
		AllVMs:       &allVMs,
		HealthyVMs:   &healthVMs,
		UnHealthyVMs: &unHealthVMs,
	}, nil
}
func (nlbHandler *OpenStackNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "ChangeHealthCheckerInfo()")
	start := call.Start()
	// Check LoadBalancer Service
	err := nlbHandler.checkNLBClient()
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	rawNLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	if len(rawNLB.Pools) < 1 {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = not Exist Pool"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}

	change, err := nlbHandler.checkPoolHealthCheckChange(healthChecker, *rawNLB)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	if !change {
		healthInfo, err := nlbHandler.getHealthCheckerInfo(*rawNLB)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.HealthCheckerInfo{}, changeErr
		}
		LoggingInfo(hiscallInfo, start)
		return healthInfo, nil
	}
	healthType, err := checkPoolHealthCheck(healthChecker)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	cbOnlyOnePool := rawNLB.Pools[0]
	poolId := cbOnlyOnePool.ID
	rawPool, err := nlbHandler.getRawPoolById(poolId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	monitor, err := nlbHandler.getRawPoolMonitorById(rawPool.MonitorID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	err = monitors.Delete(nlbHandler.NLBClient, monitor.ID).ExtractErr()
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	_, err = nlbHandler.waitingNLBHealthDeleted(monitor.ID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	_, err = nlbHandler.waitingNLBPoolActive(poolId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	monitorCreatOpts := monitors.CreateOpts{
		PoolID:     poolId,
		Type:       healthType,
		Name:       nlbIID.NameId,
		Delay:      healthChecker.Interval,
		MaxRetries: healthChecker.Threshold,
		Timeout:    healthChecker.Timeout,
	}
	createMonitor, err := monitors.Create(nlbHandler.NLBClient, &monitorCreatOpts).Extract()
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	_, err = nlbHandler.waitingNLBPoolHealthCheckActive(createMonitor.ID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	_, err = nlbHandler.waitingNLBPoolActive(poolId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	rawNLB, err = nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	healthInfo, err := nlbHandler.getHealthCheckerInfo(*rawNLB)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	LoggingInfo(hiscallInfo, start)
	return healthInfo, nil
}

func (nlbHandler *OpenStackNLBHandler) setterNLB(rawNLB loadbalancers.LoadBalancer) (irs.NLBInfo, error) {
	nlbInfo := irs.NLBInfo{
		IId: irs.IID{
			NameId:   rawNLB.Name,
			SystemId: rawNLB.ID,
		},
		Scope:       string(NLBRegionType),
		Type:        string(NLBPublicType),
		CreatedTime: rawNLB.CreatedAt,
	}
	vpcIId, err := nlbHandler.getVPCIID(rawNLB)
	if err == nil {
		nlbInfo.VpcIID = vpcIId
	}
	vmGroup, err := nlbHandler.getVMGroup(rawNLB)
	if err == nil {
		nlbInfo.VMGroup = vmGroup
	}
	listenInfo, err := nlbHandler.getListenerInfo(rawNLB)
	if err == nil {
		nlbInfo.Listener = listenInfo
	}
	healthInfo, err := nlbHandler.getHealthCheckerInfo(rawNLB)
	if err == nil {
		nlbInfo.HealthChecker = healthInfo
	}

	return nlbInfo, nil
}

func (nlbHandler *OpenStackNLBHandler) getVPCIID(nlb loadbalancers.LoadBalancer) (irs.IID, error) {
	network, err := networks.Get(nlbHandler.NetworkClient, nlb.VipNetworkID).Extract()
	if err != nil {
		return irs.IID{}, err
	}

	return irs.IID{
		NameId:   network.Name,
		SystemId: network.ID,
	}, nil
}

func (nlbHandler *OpenStackNLBHandler) getVMGroup(nlb loadbalancers.LoadBalancer) (irs.VMGroupInfo, error) {
	if len(nlb.Pools) < 1 {
		return irs.VMGroupInfo{}, errors.New("not Exist Pool")
	}
	cbOnlyOnePool := nlb.Pools[0]
	poolId := cbOnlyOnePool.ID
	rawPool, err := nlbHandler.getRawPoolById(poolId)
	if err != nil {
		return irs.VMGroupInfo{}, err
	}

	members, err := nlbHandler.getRawPoolMembersById(rawPool.ID)
	if err != nil {
		return irs.VMGroupInfo{}, err
	}

	vmIIds := make([]irs.IID, 0)
	for _, member := range *members {
		// 생성시 member의 이름을 vm의 이름과 동일하게 설정.
		vm, err := nlbHandler.getRawVMByName(member.Name)
		if err == nil {
			vmIIds = append(vmIIds, irs.IID{
				NameId:   vm.Name,
				SystemId: vm.ID,
			})
		}
	}

	info := irs.VMGroupInfo{
		Protocol: rawPool.Protocol,
		VMs:      &vmIIds,
		CspID:    rawPool.ID,
	}
	portInt, err := nlbHandler.getVMGroupPortByPool(*rawPool)
	if err == nil {
		info.Port = strconv.Itoa(portInt)
	}

	return info, nil
}

func (nlbHandler *OpenStackNLBHandler) getRawVMByName(name string) (*servers.Server, error) {
	opts := servers.ListOpts{
		Name: name,
	}
	pager, err := servers.List(nlbHandler.VMClient, opts).AllPages()
	if err != nil {
		return nil, err
	}
	list, err := servers.ExtractServers(pager)
	if len(list) != 1 {
		return nil, errors.New("not Exist Server")
	}
	return &list[0], nil
}

func (nlbHandler *OpenStackNLBHandler) getRawVMByIP(ip string) (*servers.Server, error) {
	opts := servers.ListOpts{
		IP: ip,
	}
	pager, err := servers.List(nlbHandler.VMClient, opts).AllPages()
	if err != nil {
		return nil, err
	}
	list, err := servers.ExtractServers(pager)
	if len(list) != 1 {
		return nil, errors.New("not Exist Server")
	}
	return &list[0], nil
}

func (nlbHandler *OpenStackNLBHandler) getListenerInfo(nlb loadbalancers.LoadBalancer) (irs.ListenerInfo, error) {
	if len(nlb.Listeners) < 1 {
		return irs.ListenerInfo{}, errors.New("not Exist Listener")
	}
	ip, err := nlbHandler.getNLBRawPublicIP(irs.IID{
		SystemId: nlb.ID,
	})
	if err != nil {
		return irs.ListenerInfo{}, err
	}
	cbOnlyOneListener := nlb.Listeners[0]
	listenerId := cbOnlyOneListener.ID
	rawListener, err := nlbHandler.getRawListenerById(listenerId)
	if err != nil {
		return irs.ListenerInfo{}, err
	}
	info := irs.ListenerInfo{
		Protocol: rawListener.Protocol,
		Port:     strconv.Itoa(rawListener.ProtocolPort),
		IP:       ip.FloatingIP,
		CspID:    rawListener.ID,
	}
	return info, nil
}

func (nlbHandler *OpenStackNLBHandler) getHealthCheckerInfo(nlb loadbalancers.LoadBalancer) (irs.HealthCheckerInfo, error) {
	if len(nlb.Pools) < 1 {
		return irs.HealthCheckerInfo{}, errors.New("not Exist Pool")
	}
	cbOnlyOnePool := nlb.Pools[0]
	poolId := cbOnlyOnePool.ID
	rawPool, err := nlbHandler.getRawPoolById(poolId)
	if err != nil {
		return irs.HealthCheckerInfo{}, err
	}
	monitor, err := nlbHandler.getRawPoolMonitorById(rawPool.MonitorID)
	if err != nil {
		return irs.HealthCheckerInfo{}, err
	}
	info := irs.HealthCheckerInfo{
		Protocol:  monitor.Type,
		Interval:  monitor.Delay,
		Timeout:   monitor.Timeout,
		Threshold: monitor.MaxRetries,
		CspID:     monitor.ID,
	}
	portInt, err := nlbHandler.getVMGroupPortByPool(*rawPool)
	if err == nil {
		info.Port = strconv.Itoa(portInt)
	}
	return info, nil
}
func (nlbHandler *OpenStackNLBHandler) getRawPoolMembersById(poolId string) (*[]pools.Member, error) {
	poolMemberOption := pools.ListMembersOpts{}
	pages, err := pools.ListMembers(nlbHandler.NLBClient, poolId, &poolMemberOption).AllPages()
	if err != nil {
		return nil, err
	}
	list, err := pools.ExtractMembers(pages)
	if err != nil {
		return nil, err
	}
	return &list, nil
}

func (nlbHandler *OpenStackNLBHandler) getRawPoolMonitorById(monitorId string) (*monitors.Monitor, error) {
	monitorOption := monitors.ListOpts{
		ID: monitorId,
	}
	page, err := monitors.List(nlbHandler.NLBClient, monitorOption).AllPages()
	if err != nil {
		return nil, err
	}
	list, err := monitors.ExtractMonitors(page)
	if len(list) != 1 {
		return nil, errors.New("not Exist Listener")
	}
	return &list[0], nil
}

func (nlbHandler *OpenStackNLBHandler) getRawPoolMemberById(poolId string, memberId string) (*pools.Member, error) {
	memberOpts := pools.ListMembersOpts{
		ID: memberId,
	}
	page, err := pools.ListMembers(nlbHandler.NLBClient, poolId, memberOpts).AllPages()
	if err != nil {
		return nil, err
	}
	list, err := pools.ExtractMembers(page)
	if len(list) != 1 {
		return nil, errors.New("not Exist Listener")
	}
	return &list[0], nil
}

func (nlbHandler *OpenStackNLBHandler) getRawPoolById(poolId string) (*pools.Pool, error) {
	poolOption := pools.ListOpts{
		ID: poolId,
	}
	page, err := pools.List(nlbHandler.NLBClient, poolOption).AllPages()
	if err != nil {
		return nil, err
	}
	list, err := pools.ExtractPools(page)
	if len(list) != 1 {
		return nil, errors.New("not Exist Listener")
	}
	return &list[0], nil
}
func (nlbHandler *OpenStackNLBHandler) getRawListenerById(listenerId string) (*listeners.Listener, error) {
	listenerOptions := listeners.ListOpts{
		ID: listenerId,
	}
	page, err := listeners.List(nlbHandler.NLBClient, &listenerOptions).AllPages()
	if err != nil {
		return nil, err
	}
	list, err := listeners.ExtractListeners(page)
	if len(list) != 1 {
		return nil, errors.New("not Exist Listener")
	}
	return &list[0], nil
}

func (nlbHandler *OpenStackNLBHandler) getRawNLB(iid irs.IID) (*loadbalancers.LoadBalancer, error) {
	if iid.SystemId != "" {
		return loadbalancers.Get(nlbHandler.NLBClient, iid.SystemId).Extract()
	} else {
		listOpts := loadbalancers.ListOpts{
			ProjectID: nlbHandler.CredentialInfo.ProjectID,
			Name:      iid.NameId,
		}
		rawListAllPage, err := loadbalancers.List(nlbHandler.NLBClient, listOpts).AllPages()
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
		return nil, errors.New(fmt.Sprintf("not Exist NLB : %s", iid.NameId))
	}
}

func (nlbHandler *OpenStackNLBHandler) getRawNLBList() ([]loadbalancers.LoadBalancer, error) {
	listOpts := loadbalancers.ListOpts{
		ProjectID: nlbHandler.CredentialInfo.ProjectID,
	}
	rawListAllPage, err := loadbalancers.List(nlbHandler.NLBClient, listOpts).AllPages()
	if err != nil {
		return nil, err
	}
	list, err := loadbalancers.ExtractLoadBalancers(rawListAllPage)
	if err != nil {
		return nil, err
	}
	return list, nil
}
func (nlbHandler *OpenStackNLBHandler) getRawNLBProvider() (string, error) {
	list, err := nlbHandler.getRawNLBProviderList()
	if err != nil {
		return "", err
	}
	for _, provider := range list {
		if provider.Name == "amphora" {
			return provider.Name, nil
		}
	}
	return "", errors.New("no Exist Openstack LoadBalancer Provider amphora")
}

func (nlbHandler *OpenStackNLBHandler) getMemberCreateOpt(vmIIds []irs.IID, port int) ([]pools.CreateMemberOpts, error) {
	poolMemberOptions := make([]pools.CreateMemberOpts, len(vmIIds))
	for i, vmIId := range vmIIds {
		vm, err := nlbHandler.getRawVMByName(vmIId.NameId)
		if err != nil {
			return nil, err
		}
		IP := ""
		for _, addrArray := range vm.Addresses {
			breakCheck := false
			for _, addr := range addrArray.([]interface{}) {
				addrMap := addr.(map[string]interface{})
				if addrMap["OS-EXT-IPS:type"] == "fixed" {
					IP = addrMap["addr"].(string)
					breakCheck = true
					break
				}
			}
			if breakCheck {
				break
			}
		}
		if IP == "" {
			return nil, errors.New("not found VM-IP")
		}
		subnetID := ""
		portResource, _ := GetPortByDeviceID(nlbHandler.NetworkClient, vm.ID)
		if portResource != nil {
			// Subnet 정보 설정
			if len(portResource.FixedIPs) > 0 {
				ipInfo := portResource.FixedIPs[0]
				subnetID = ipInfo.SubnetID
			}
		}
		if IP == "" {
			return nil, errors.New("not found subnetID")
		}
		// PoolMemberName := vm.Name
		poolMemberOptions[i] = pools.CreateMemberOpts{
			Name:         vm.Name,
			Address:      IP,
			ProtocolPort: port,
			SubnetID:     subnetID,
			Weight:       gophercloud.IntToPointer(1),
		}
	}
	return poolMemberOptions, nil
}

//getListenerCreateOpt
func (nlbHandler *OpenStackNLBHandler) getListenerCreateOpt(nlbReqInfo irs.NLBInfo, nlbID string, poolId string) (listeners.CreateOpts, error) {
	listenerProtocol, err := checkListenerProtocol(nlbReqInfo.Listener.Protocol)
	if err != nil {
		return listeners.CreateOpts{}, err
	}
	portInt, err := strconv.Atoi(nlbReqInfo.Listener.Port)
	if err != nil {
		return listeners.CreateOpts{}, errors.New("invalid Listener Port")
	}
	if portInt < 1 || portInt > 65535 {
		return listeners.CreateOpts{}, errors.New("invalid Listener Port")
	}
	return listeners.CreateOpts{
		Protocol:       listenerProtocol,
		DefaultPoolID:  poolId,
		ProtocolPort:   portInt,
		Name:           nlbReqInfo.IId.NameId,
		LoadbalancerID: nlbID,
		AdminStateUp:   gophercloud.Enabled,
	}, nil
}

func (nlbHandler *OpenStackNLBHandler) getPoolCreateOpt(nlbReqInfo irs.NLBInfo, nlbId string) (pools.CreateOpts, error) {
	poolProtocol, err := checkvmGroupProtocol(nlbReqInfo.VMGroup.Protocol)
	if err != nil {
		return pools.CreateOpts{}, err
	}
	portInt, err := strconv.Atoi(nlbReqInfo.VMGroup.Port)
	if err != nil {
		return pools.CreateOpts{}, errors.New("invalid vmGroup Port")
	}
	if portInt < 1 || portInt > 65535 {
		return pools.CreateOpts{}, errors.New("invalid vmGroup Port")
	}

	// TODO : GET에서 해당 Description 이용
	vmGroupPortSticker := generateVMGroupPortDescription(portInt)
	return pools.CreateOpts{
		LoadbalancerID: nlbId,
		Name:           nlbReqInfo.IId.NameId,
		LBMethod:       pools.LBMethodRoundRobin,
		Protocol:       poolProtocol,
		AdminStateUp:   gophercloud.Enabled,
		Description:    vmGroupPortSticker,
	}, nil
}
func (nlbHandler *OpenStackNLBHandler) getRawNLBProviderList() ([]providers.Provider, error) {
	listOpts := providers.ListOpts{}
	rawListAllPage, err := providers.List(nlbHandler.NLBClient, listOpts).AllPages()
	if err != nil {
		return nil, err
	}
	list, err := providers.ExtractProviders(rawListAllPage)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (nlbHandler *OpenStackNLBHandler) getFirstSubnetAndNetworkId(vpcName string) (subnet string, network string, err error) {
	rawVPC, err := nlbHandler.getRawVPCByName(vpcName)
	if err != nil {
		return "", "", err
	}
	if len(rawVPC.Subnets) > 0 {
		return rawVPC.Subnets[0], rawVPC.ID, nil
	}
	return "", "", errors.New("not found subnet")
}

func (nlbHandler *OpenStackNLBHandler) getRawVPCByName(vpcName string) (*NetworkWithExt, error) {
	listOpts := external.ListOptsExt{
		ListOptsBuilder: networks.ListOpts{
			Name: vpcName,
		},
	}
	page, err := networks.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return nil, err
	}

	var vpcList []NetworkWithExt
	err = networks.ExtractNetworksInto(page, &vpcList)
	if err != nil {
		return nil, err
	}

	for _, vpc := range vpcList {
		if vpc.Name == vpcName {
			return &vpc, nil
		}
	}
	return nil, errors.New("not found vpc")
}

func generateVMGroupPortDescription(port int) string {
	return fmt.Sprintf("vmgroupport:%d,", port)
}

func getVMGroupPortByDescription(des string) (int, error) {
	if reg, err := regexp.Compile("vmgroupport:[0-9]+"); err == nil {
		if stArr := reg.FindAllString(des, -1); len(stArr) > 0 {
			if portReg2, err := regexp.Compile("[0-9]+"); err == nil {
				portStrArr := portReg2.FindAllString(stArr[0], -1)
				if len(portStrArr) > 0 {
					portInt, err := strconv.Atoi(portStrArr[0])
					if err == nil {
						return portInt, nil
					}
				}
			}
		}
	}
	return 0, errors.New("unable to get port for vmGroup from Description")
}

func (nlbHandler *OpenStackNLBHandler) getNLBRawPublicIP(nlbIId irs.IID) (floatingips.FloatingIP, error) {
	rawnlb, err := nlbHandler.getRawNLB(nlbIId)
	if err != nil {
		return floatingips.FloatingIP{}, err
	}
	externVPCID, err := GetPublicVPCInfo(nlbHandler.NetworkClient, "ID")
	if err != nil {
		return floatingips.FloatingIP{}, err
	}
	listOPt := floatingips.ListOpts{
		FloatingNetworkID: externVPCID,
		PortID:            rawnlb.VipPortID,
	}
	pager, err := floatingips.List(nlbHandler.NetworkClient, listOPt).AllPages()
	if err != nil {
		return floatingips.FloatingIP{}, err
	}
	all, err := floatingips.ExtractFloatingIPs(pager)
	if err != nil {
		return floatingips.FloatingIP{}, err
	}
	if len(all) > 0 {
		if strings.EqualFold(all[0].PortID, rawnlb.VipPortID) {
			return all[0], nil
		}
	}
	return floatingips.FloatingIP{}, errors.New("not found floatingIP")
}

func (nlbHandler *OpenStackNLBHandler) AssociatePublicIP(nlbPortId string) (bool, error) {
	// PublicIP 생성
	externVPCID, err := GetPublicVPCInfo(nlbHandler.NetworkClient, "ID")
	if err != nil {
		return false, err
	}
	createOpts := floatingips.CreateOpts{
		FloatingNetworkID: externVPCID,
		PortID:            nlbPortId,
	}
	_, err = floatingips.Create(nlbHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		return false, err
	}
	return true, nil
}
func (nlbHandler *OpenStackNLBHandler) checkDeletable(nlbIID irs.IID) (bool, error) {
	rawnlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		return false, err
	}
	if rawnlb.ProvisioningStatus == "PENDING_CREATE" {
		return false, errors.New("cannot delete ProvisioningStatus when it is PENDING_CREATE")
	}
	return true, nil
}

func (nlbHandler *OpenStackNLBHandler) waitingNLBHealthDeleted(monitorId string) (bool, error) {
	listOpts := monitors.ListOpts{
		ID: monitorId,
	}
	page, err := monitors.List(nlbHandler.NLBClient, listOpts).AllPages()
	if err != nil {
		return false, err
	}
	list, err := monitors.ExtractMonitors(page)
	if err != nil {
		return false, err
	}
	if len(list) == 0 {
		return true, nil
	}
	curRetryCnt := 0
	maxRetryCnt := 240
	for {
		curRetryCnt++
		page, err = monitors.List(nlbHandler.NLBClient, listOpts).AllPages()
		if err != nil {
			return false, err
		}
		list, err = monitors.ExtractMonitors(page)
		if err != nil {
			return false, err
		}
		if len(list) == 0 {
			return true, nil
		}
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			return false, errors.New(fmt.Sprintf("failed to delete NLB Listener, exceeded maximum retry count %d", maxRetryCnt))
		}
	}
}

func (nlbHandler *OpenStackNLBHandler) waitingNLBListenerDeleted(listenerId string) (bool, error) {
	listOpts := listeners.ListOpts{
		ID: listenerId,
	}
	page, err := listeners.List(nlbHandler.NLBClient, listOpts).AllPages()
	if err != nil {
		return false, err
	}
	list, err := listeners.ExtractListeners(page)
	if err != nil {
		return false, err
	}
	if len(list) == 0 {
		return true, nil
	}
	curRetryCnt := 0
	maxRetryCnt := 240
	for {
		curRetryCnt++
		page, err = listeners.List(nlbHandler.NLBClient, listOpts).AllPages()
		if err != nil {
			return false, err
		}
		list, err = listeners.ExtractListeners(page)
		if err != nil {
			return false, err
		}
		if len(list) == 0 {
			return true, nil
		}
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			return false, errors.New(fmt.Sprintf("failed to delete NLB Listener, exceeded maximum retry count %d", maxRetryCnt))
		}
	}
}

func (nlbHandler *OpenStackNLBHandler) waitingNLBPoolHealthCheckActive(healthCheckId string) (bool, error) {
	health, err := nlbHandler.getRawPoolMonitorById(healthCheckId)
	if err != nil {
		return false, err
	}
	curRetryCnt := 0
	maxRetryCnt := 240
	for {
		curRetryCnt++
		health, err = nlbHandler.getRawPoolMonitorById(healthCheckId)
		if err == nil {
			if health.ProvisioningStatus == "ACTIVE" {
				return true, nil
			}
			if health.ProvisioningStatus == "ERROR" {
				return false, errors.New(fmt.Sprintf("failed to create NLB Pool, ProvisioningStatus : ERROR"))
			}
		}
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			return false, errors.New(fmt.Sprintf("failed to create NLB Pool, exceeded maximum retry count %d", maxRetryCnt))
		}
	}
}

func (nlbHandler *OpenStackNLBHandler) waitingNLBPoolMemberActive(poolId string, memberId string) (bool, error) {
	member, err := nlbHandler.getRawPoolMemberById(poolId, memberId)
	if err != nil {
		return false, err
	}
	curRetryCnt := 0
	maxRetryCnt := 240
	for {
		curRetryCnt++
		member, err = nlbHandler.getRawPoolMemberById(poolId, memberId)
		if err == nil {
			if member.ProvisioningStatus == "ACTIVE" {
				return true, nil
			}
			if member.ProvisioningStatus == "ERROR" {
				return false, errors.New(fmt.Sprintf("failed to create NLB Pool, ProvisioningStatus : ERROR"))
			}
		}
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			return false, errors.New(fmt.Sprintf("failed to create NLB Pool, exceeded maximum retry count %d", maxRetryCnt))
		}
	}
}

func (nlbHandler *OpenStackNLBHandler) waitingNLBPoolActive(poolId string) (bool, error) {
	pool, err := nlbHandler.getRawPoolById(poolId)
	if err != nil {
		return false, err
	}
	curRetryCnt := 0
	maxRetryCnt := 240
	for {
		curRetryCnt++
		pool, err = nlbHandler.getRawPoolById(poolId)
		if err == nil {
			if strings.ToUpper(pool.ProvisioningStatus) == "ACTIVE" {
				return true, nil
			}
			if strings.ToUpper(pool.ProvisioningStatus) == "ERROR" {
				return false, errors.New(fmt.Sprintf("failed to create NLB Pool, ProvisioningStatus : ERROR"))
			}
		}
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			return false, errors.New(fmt.Sprintf("failed to create NLB Pool, exceeded maximum retry count %d", maxRetryCnt))
		}
	}
}

func (nlbHandler *OpenStackNLBHandler) waitingNLBListenerActive(listenerId string) (bool, error) {
	listener, err := nlbHandler.getRawListenerById(listenerId)
	if err != nil {
		return false, err
	}
	curRetryCnt := 0
	maxRetryCnt := 240
	for {
		curRetryCnt++
		listener, err = nlbHandler.getRawListenerById(listenerId)
		if err == nil {
			if listener.ProvisioningStatus == "ACTIVE" {
				return true, nil
			}
			if listener.ProvisioningStatus == "ERROR" {
				return false, errors.New(fmt.Sprintf("failed to create NLB Listener, ProvisioningStatus : ERROR"))
			}
		}
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			return false, errors.New(fmt.Sprintf("failed to create NLB Listener, exceeded maximum retry count %d", maxRetryCnt))
		}
	}
}

func (nlbHandler *OpenStackNLBHandler) waitingNLBActive(nlbIID irs.IID) (bool, error) {
	rawnlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		return false, err
	}
	curRetryCnt := 0
	maxRetryCnt := 240
	for {
		curRetryCnt++
		rawnlb, err = nlbHandler.getRawNLB(nlbIID)
		if err == nil {
			if rawnlb.ProvisioningStatus == "ACTIVE" {
				return true, nil
			}
			if rawnlb.ProvisioningStatus == "ERROR" {
				return false, errors.New(fmt.Sprintf("failed to create NLB, ProvisioningStatus : ERROR"))
			}
		}
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			return false, errors.New(fmt.Sprintf("failed to create NLB, exceeded maximum retry count %d", maxRetryCnt))
		}
	}
}

func (nlbHandler *OpenStackNLBHandler) ExistEqualName(name string) (bool, error) {
	if name == "" {
		return false, errors.New("invalid Name")
	}
	list, err := nlbHandler.getRawNLBList()
	if err != nil {
		return false, err
	}
	for _, nlb := range list {
		if strings.EqualFold(nlb.Name, name) {
			return true, nil
		}
	}
	return false, nil
}

func (nlbHandler *OpenStackNLBHandler) CleanerNLB(nlbIID irs.IID) (bool, error) {
	rawnlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		return false, err
	}
	_, err = nlbHandler.checkDeletable(nlbIID)
	if err != nil {
		return false, err
	}
	floatingIP, err := nlbHandler.getNLBRawPublicIP(nlbIID)
	if err == nil {
		ipDelErr := floatingips.Delete(nlbHandler.NetworkClient, floatingIP.ID).ExtractErr()
		if ipDelErr != nil {
			return false, ipDelErr
		}
	}
	err = loadbalancers.Delete(nlbHandler.NLBClient, rawnlb.ID, loadbalancers.DeleteOpts{Cascade: true}).ExtractErr()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (nlbHandler *OpenStackNLBHandler) setPoolDescription(des string, poolId string) (*pools.Pool, error) {
	updateOpts := pools.UpdateOpts{
		Description: &des,
	}
	pool, err := pools.Update(nlbHandler.NLBClient, poolId, updateOpts).Extract()
	if err != nil {
		return nil, err
	}
	_, err = nlbHandler.waitingNLBPoolActive(pool.ID)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func (nlbHandler *OpenStackNLBHandler) createPool(nlbReqInfo irs.NLBInfo, nlbId string) (*pools.Pool, error) {
	poolCreateOpts, err := nlbHandler.getPoolCreateOpt(nlbReqInfo, nlbId)
	if err != nil {
		return nil, err
	}
	pool, err := pools.Create(nlbHandler.NLBClient, poolCreateOpts).Extract()
	if err != nil {
		return nil, err
	}
	_, err = nlbHandler.waitingNLBPoolActive(pool.ID)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func (nlbHandler *OpenStackNLBHandler) detachPoolMemberByMemberID(memberId string, poolId string) (bool, error) {
	err := pools.DeleteMember(nlbHandler.NLBClient, poolId, memberId).ExtractErr()
	if err != nil {
		return false, err
	}
	_, err = nlbHandler.waitingNLBPoolActive(poolId)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (nlbHandler *OpenStackNLBHandler) detachPoolMember(removeIID irs.IID, poolId string) (bool, error) {
	members, err := nlbHandler.getRawPoolMembersById(poolId)
	if err != nil {
		return false, err
	}
	for _, member := range *members {
		vm, err := nlbHandler.getRawVMByName(member.Name)
		if err != nil {
			return false, err
		}
		if strings.EqualFold(removeIID.NameId, vm.Name) {
			err = pools.DeleteMember(nlbHandler.NLBClient, poolId, member.ID).ExtractErr()
			if err != nil {
				return false, err
			}
		}
	}
	_, err = nlbHandler.waitingNLBPoolActive(poolId)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (nlbHandler *OpenStackNLBHandler) attachPoolMembers(vmIIds []irs.IID, portStr string, poolId string) ([]pools.Member, error) {
	portInt, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, errors.New("invalid Member Port")
	}
	if portInt < 1 || portInt > 65535 {
		return nil, errors.New("invalid Member Port")
	}
	memberOpts, err := nlbHandler.getMemberCreateOpt(vmIIds, portInt)
	if err != nil {
		return nil, err
	}
	createdMembers := make([]pools.Member, len(memberOpts))
	for i, memberOpt := range memberOpts {
		mem, err := pools.CreateMember(nlbHandler.NLBClient, poolId, memberOpt).Extract()
		if err != nil {
			return nil, errors.New(fmt.Sprintf("failed Create Member err = %s", err.Error()))
		}
		_, err = nlbHandler.waitingNLBPoolMemberActive(poolId, mem.ID)
		if err != nil {
			return nil, err
		}
		createdMembers[i] = *mem
	}
	_, err = nlbHandler.waitingNLBPoolActive(poolId)
	if err != nil {
		return nil, err
	}
	return createdMembers, nil
}

func (nlbHandler *OpenStackNLBHandler) checkPoolHealthCheckChange(healthCheckerInfo irs.HealthCheckerInfo, nlb loadbalancers.LoadBalancer) (bool, error) {
	if len(nlb.Pools) < 1 {
		return false, errors.New("not Exist Pool")
	}
	cbOnlyOnePool := nlb.Pools[0]
	poolId := cbOnlyOnePool.ID
	rawPool, err := nlbHandler.getRawPoolById(poolId)
	if err != nil {
		return false, err
	}

	healthType, err := checkPoolHealthCheck(healthCheckerInfo)
	if err != nil {
		return false, err
	}

	if healthCheckerInfo.Port != "" {
		newPort, err := strconv.Atoi(healthCheckerInfo.Port)
		if err != nil {
			return false, errors.New("changing the port of the healthChecker is not supported")
		}
		portInt, err := nlbHandler.getVMGroupPortByPool(*rawPool)
		if err != nil {
			return false, err
		}
		if newPort != portInt {
			return false, errors.New("changing the port of the healthChecker is not supported")
		}
	}

	oldHealthCheckerInfo, err := nlbHandler.getHealthCheckerInfo(nlb)
	if err != nil {
		return false, err
	}
	if reflect.DeepEqual(irs.HealthCheckerInfo{
		Protocol:  oldHealthCheckerInfo.Protocol,
		Interval:  oldHealthCheckerInfo.Interval,
		Timeout:   oldHealthCheckerInfo.Timeout,
		Threshold: oldHealthCheckerInfo.Threshold,
	}, irs.HealthCheckerInfo{
		Protocol:  healthType,
		Interval:  healthCheckerInfo.Interval,
		Timeout:   healthCheckerInfo.Timeout,
		Threshold: healthCheckerInfo.Threshold,
	}) {
		return false, nil
	}
	return true, nil
}

func checkPoolHealthCheckProtocol(healthCheckerInfo irs.HealthCheckerInfo) (string, error) {
	switch strings.ToUpper(healthCheckerInfo.Protocol) {
	case "PING", "TCP", "HTTP", "HTTPS":
		return strings.ToUpper(healthCheckerInfo.Protocol), nil
	default:
		return "", errors.New(fmt.Sprintf("invalid HealthChecker Protocol"))
	}
}

func checkPoolHealthCheck(healthCheckerInfo irs.HealthCheckerInfo) (string, error) {
	healthType, err := checkPoolHealthCheckProtocol(healthCheckerInfo)
	if err != nil {
		return "", err
	}
	if healthCheckerInfo.Threshold > 10 || healthCheckerInfo.Threshold < 1 {
		return "", errors.New(fmt.Sprintf("invalid HealthChecker Threshold, interval must be between 1 and 10"))
	}
	if healthCheckerInfo.Timeout < 0 {
		return "", errors.New(fmt.Sprintf("invalid HealthChecker Timeout, Timeout must be a number greater than or equal to 0"))
	}
	if healthCheckerInfo.Interval < healthCheckerInfo.Timeout {
		return "", errors.New(fmt.Sprintf("invalid HealthChecker Interval, interval must be greater than or equal to the timeout"))
	}
	return healthType, nil
}

func (nlbHandler *OpenStackNLBHandler) createPoolHealthCheck(nlbReqInfo irs.NLBInfo, poolId string) (monitors.Monitor, error) {
	healthType, err := checkPoolHealthCheck(nlbReqInfo.HealthChecker)
	if err != nil {
		return monitors.Monitor{}, err
	}

	monitorCreatOpts := monitors.CreateOpts{
		PoolID:     poolId,
		Type:       healthType,
		Name:       nlbReqInfo.IId.NameId,
		Delay:      nlbReqInfo.HealthChecker.Interval,
		MaxRetries: nlbReqInfo.HealthChecker.Threshold,
		Timeout:    nlbReqInfo.HealthChecker.Timeout,
	}
	createMonitor, err := monitors.Create(nlbHandler.NLBClient, &monitorCreatOpts).Extract()
	if err != nil {
		return monitors.Monitor{}, errors.New(fmt.Sprintf("failed Create HealthChecker err = %s", err.Error()))
	}
	_, err = nlbHandler.waitingNLBPoolHealthCheckActive(createMonitor.ID)
	if err != nil {
		return monitors.Monitor{}, err
	}
	_, err = nlbHandler.waitingNLBPoolActive(poolId)
	if err != nil {
		return monitors.Monitor{}, err
	}
	return *createMonitor, nil
}
func (nlbHandler *OpenStackNLBHandler) createListener(nlbReqInfo irs.NLBInfo, nlbId string, poolId string) (listeners.Listener, error) {
	listenerCreateOpts, err := nlbHandler.getListenerCreateOpt(nlbReqInfo, nlbId, poolId)
	if err != nil {
		return listeners.Listener{}, err
	}
	listener, err := listeners.Create(nlbHandler.NLBClient, listenerCreateOpts).Extract()
	if err != nil {
		return listeners.Listener{}, errors.New(fmt.Sprintf("failed Create Listener err = %s", err.Error()))
	}
	_, err = nlbHandler.waitingNLBListenerActive(listener.ID)
	if err != nil {
		return listeners.Listener{}, err
	}
	return *listener, nil
}

func (nlbHandler *OpenStackNLBHandler) createLoadBalancer(nlbReqInfo irs.NLBInfo) (loadbalancers.LoadBalancer, error) {
	exist, err := nlbHandler.ExistEqualName(nlbReqInfo.IId.NameId)
	if err != nil {
		return loadbalancers.LoadBalancer{}, err
	}
	if exist {
		return loadbalancers.LoadBalancer{}, errors.New(fmt.Sprintf("already exist nlb Name %s", nlbReqInfo.IId.NameId))
	}
	providerName, err := nlbHandler.getRawNLBProvider()
	if err != nil {
		return loadbalancers.LoadBalancer{}, err
	}
	subnetId, networkId, err := nlbHandler.getFirstSubnetAndNetworkId(nlbReqInfo.VpcIID.NameId)
	if err != nil {
		return loadbalancers.LoadBalancer{}, err
	}
	createOpts := loadbalancers.CreateOpts{
		Name:         nlbReqInfo.IId.NameId,
		AdminStateUp: gophercloud.Enabled,
		VipSubnetID:  subnetId,
		VipNetworkID: networkId,
		Provider:     providerName,
	}
	loadbalancer, err := loadbalancers.Create(nlbHandler.NLBClient, createOpts).Extract()
	if err != nil {
		return loadbalancers.LoadBalancer{}, errors.New(fmt.Sprintf("failed Create LoadBalancer err = %s", err.Error()))
	}
	_, err = nlbHandler.waitingNLBActive(irs.IID{
		SystemId: loadbalancer.ID,
	})
	if err != nil {
		return loadbalancers.LoadBalancer{}, err
	}
	return *loadbalancer, nil
}

func (nlbHandler *OpenStackNLBHandler) getVMGroupPortByPool(pool pools.Pool) (int, error) {
	portInt, err := getVMGroupPortByDescription(pool.Description)
	if err != nil {
		members, err := nlbHandler.getRawPoolMembersById(pool.ID)
		if err == nil && len(*members) > 0 {
			tempMembers := *members
			portInt = tempMembers[0].ProtocolPort
			return portInt, nil
		}
		return 0, errors.New("unable to get port for vmGroup")
	}
	return portInt, nil
}

func (nlbHandler *OpenStackNLBHandler) checkNLBClient() error {
	// require openstack octavia
	if nlbHandler.NLBClient == nil {
		return errors.New("this Openstack cannot provide LoadBalancer. Please check if LoadBalancer is installed")
	}
	return nil
}

func checkListenerProtocol(protocol string) (listeners.Protocol, error) {
	if strings.EqualFold(strings.ToUpper(protocol), string(listeners.ProtocolTCP)) {
		return listeners.ProtocolTCP, nil
	}
	return "", errors.New("invalid Listener Protocols, openstack Listener provides only TCP protocols")
}

func checkvmGroupProtocol(protocol string) (pools.Protocol, error) {
	if strings.EqualFold(strings.ToUpper(protocol), string(pools.ProtocolTCP)) {
		return pools.ProtocolTCP, nil
	}
	return "", errors.New("invalid vmGroup Protocols, openstack vmGroup provides only TCP protocols")
}
