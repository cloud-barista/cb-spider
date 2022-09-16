package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type NLBType string
type NLBScope string
type NLBVMStatus string

const (
	NLBPublicType      NLBType     = "PUBLIC"
	NLBInternalType    NLBType     = "INTERNAL"
	NLBGlobalType      NLBScope    = "GLOBAL"
	NLBRegionType      NLBScope    = "REGION"
	NLBVMStatusFault   NLBVMStatus = "faulted"
	NLBVMStatusOK      NLBVMStatus = "ok"
	NLBVMStatusUnknown NLBVMStatus = "unknown"
)

type IbmNLBHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VpcService     *vpcv1.VpcV1
	Ctx            context.Context
}

// ------ NLB Management
func (nlbHandler *IbmNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (irs.NLBInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbReqInfo.IId.NameId, "CreateNLB()")
	start := call.Start()
	rawNLB, err := nlbHandler.createNLB(nlbReqInfo)
	if err != nil {
		nlbHandler.cleanerNLB(nlbReqInfo.IId)
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}
	info, err := nlbHandler.setterNLB(rawNLB)
	if err != nil {
		nlbHandler.cleanerNLB(nlbReqInfo.IId)
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)
	return *info, nil
}
func (nlbHandler *IbmNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", "NLB", "ListNLB()")
	start := call.Start()

	nlbList, err := nlbHandler.getRawNLBList()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List NLB. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	nlbInfoList := make([]*irs.NLBInfo, len(*nlbList))

	var wait sync.WaitGroup
	wait.Add(len(*nlbList))
	var errList []string

	for i, nlb := range *nlbList {
		go func() {
			defer wait.Done()
			info, getErr := nlbHandler.setterNLB(nlb)
			if getErr != nil {
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				errList = append(errList, getErr.Error())
			}
			nlbInfoList[i] = info
		}()
	}
	wait.Wait()
	LoggingInfo(hiscallInfo, start)

	if len(errList) > 0 {
		errList = append([]string{"Failed to List NLB. err = "}, errList...)
		return nil, errors.New(strings.Join(errList, "\n\t"))
	}

	return nlbInfoList, nil
}
func (nlbHandler *IbmNLBHandler) GetNLB(nlbIID irs.IID) (irs.NLBInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "GetNLB()")
	start := call.Start()
	rawNLB, err := nlbHandler.getRawNLBByName(nlbIID.NameId)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get NLB. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.NLBInfo{}, getErr
	}
	info, err := nlbHandler.setterNLB(rawNLB)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get NLB. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.NLBInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return *info, err
}
func (nlbHandler *IbmNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "DeleteNLB()")
	start := call.Start()

	_, err := nlbHandler.cleanerNLB(nlbIID)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete NLB. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// ------ Frontend Control
func (nlbHandler *IbmNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "ChangeListener()")
	start := call.Start()

	rawNLB, err := nlbHandler.getRawNLBByName(nlbIID.NameId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Change Listener. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	nlbId := *rawNLB.ID
	if rawNLB.Listeners == nil || len(rawNLB.Listeners) < 1 {
		changeErr := errors.New(fmt.Sprintf("Failed to Change Listener. err = listener does not exist within that NLB"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	ListenerOption := vpcv1.GetLoadBalancerListenerOptions{}
	ListenerOption.SetLoadBalancerID(*rawNLB.ID)
	ListenerOption.SetID(*rawNLB.Listeners[0].ID)
	rawListener, _, err := nlbHandler.VpcService.GetLoadBalancerListenerWithContext(nlbHandler.Ctx, &ListenerOption)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Change Listener. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	if listener.Protocol != "" && !strings.EqualFold(strings.ToUpper(listener.Protocol), strings.ToUpper(*rawListener.Protocol)) {
		changeErr := errors.New(fmt.Sprintf("Failed to Change Listener. err = changing the protocol of the Listener is not supported."))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	oldListenerPort := int(*rawListener.Port)
	listenerPort, err := strconv.Atoi(listener.Port)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Change Listener. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	if oldListenerPort != listenerPort {
		if listenerPort > 65535 || listenerPort < 1 {
			return irs.ListenerInfo{}, errors.New("ibm-NLB Listener provides an port of between 1 and 65535")
		}

		listenerPort64 := int64(listenerPort)

		updateMaps := make(map[string]interface{})
		updateMaps["port"] = core.Int64Ptr(listenerPort64)

		updateOption := vpcv1.UpdateLoadBalancerListenerOptions{}
		updateOption.SetID(*rawListener.ID)
		updateOption.SetLoadBalancerID(nlbId)
		updateOption.SetLoadBalancerListenerPatch(updateMaps)

		_, _, err = nlbHandler.VpcService.UpdateLoadBalancerListenerWithContext(nlbHandler.Ctx, &updateOption)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to Change Listener. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.ListenerInfo{}, changeErr
		}

	}
	updatedRawNLB, err := nlbHandler.checkUpdatableNLB(nlbId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Change Listener. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}

	info, err := nlbHandler.setterNLB(updatedRawNLB)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Change Listener. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	LoggingInfo(hiscallInfo, start)
	return info.Listener, err
}

// ------ Backend Control
func (nlbHandler *IbmNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "ChangeVMGroupInfo()")
	start := call.Start()
	rawNLB, err := nlbHandler.getRawNLBByName(nlbIID.NameId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Change VMGroupInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	if rawNLB.Pools == nil || len(rawNLB.Pools) < 1 {
		changeErr := errors.New(fmt.Sprintf("Failed to Change VMGroupInfo. err = VMGroup does not exist within that NLB"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	updatePoolId := *rawNLB.Pools[0].ID
	updateNLBId := *rawNLB.ID

	poolOption := vpcv1.GetLoadBalancerPoolOptions{}
	poolOption.SetLoadBalancerID(updateNLBId)
	poolOption.SetID(updatePoolId)

	rawPool, _, err := nlbHandler.VpcService.GetLoadBalancerPoolWithContext(nlbHandler.Ctx, &poolOption)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Change VMGroupInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	vmGroupPort, err := nlbHandler.getPoolPort(updateNLBId, updatePoolId)

	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Change VMGroupInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	if vmGroup.Protocol != "" && !strings.EqualFold(strings.ToUpper(vmGroup.Protocol), strings.ToUpper(*rawPool.Protocol)) {
		changeErr := errors.New(fmt.Sprintf("Failed to Change Listener. err = changing the protocol of the Listener is not supported."))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	// 변경시만 업데이트 하도록
	if !strings.EqualFold(vmGroup.Port, strconv.Itoa(vmGroupPort)) {
		memberPort, err := strconv.Atoi(vmGroup.Port)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to Change VMGroupInfo. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.VMGroupInfo{}, changeErr
		}
		if memberPort > 65535 || memberPort < 1 {
			changeErr := errors.New(fmt.Sprintf("Failed to Change VMGroupInfo. err = ibm-NLB vmGroup Port provides an port of between 1 and 65535"))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.VMGroupInfo{}, changeErr
		}

		updateMaps := make(map[string]interface{})
		updateMaps["name"] = generatePoolName(vmGroup.Port)

		updatePoolOption := vpcv1.UpdateLoadBalancerPoolOptions{}
		updatePoolOption.SetID(updatePoolId)
		updatePoolOption.SetLoadBalancerID(updateNLBId)
		updatePoolOption.SetLoadBalancerPoolPatch(updateMaps)

		_, err = nlbHandler.checkUpdatableNLB(updateNLBId)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to Change VMGroupInfo. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.VMGroupInfo{}, changeErr
		}

		_, _, err = nlbHandler.VpcService.UpdateLoadBalancerPoolWithContext(nlbHandler.Ctx, &updatePoolOption)

		_, err = nlbHandler.checkUpdatableNLB(updateNLBId)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to Change VMGroupInfo. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.VMGroupInfo{}, changeErr
		}

		poolMemberListOptions := vpcv1.ListLoadBalancerPoolMembersOptions{}
		poolMemberListOptions.SetLoadBalancerID(updateNLBId)
		poolMemberListOptions.SetPoolID(updatePoolId)

		poolAllMembers, _, err := nlbHandler.VpcService.ListLoadBalancerPoolMembersWithContext(nlbHandler.Ctx, &poolMemberListOptions)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to Change VMGroupInfo. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.VMGroupInfo{}, changeErr
		}
		vmIIDs, err := nlbHandler.getAllVMIIDsByMembers(poolAllMembers.Members)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to Change VMGroupInfo. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.VMGroupInfo{}, changeErr
		}
		vmGroup.VMs = &vmIIDs

		memberArrayOption, err := nlbHandler.convertCBVMGroupToIbmPoolMember(vmGroup)
		updateMemberOption := vpcv1.ReplaceLoadBalancerPoolMembersOptions{}
		updateMemberOption.SetPoolID(updatePoolId)
		updateMemberOption.SetLoadBalancerID(updateNLBId)
		updateMemberOption.SetMembers(memberArrayOption)

		_, _, err = nlbHandler.VpcService.ReplaceLoadBalancerPoolMembersWithContext(nlbHandler.Ctx, &updateMemberOption)

		_, err = nlbHandler.checkUpdatableNLB(updateNLBId)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to Change VMGroupInfo. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.VMGroupInfo{}, changeErr
		}
	}
	rawNLB, err = nlbHandler.getRawNLBByName(nlbIID.NameId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Change VMGroupInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	info, err := nlbHandler.setterNLB(rawNLB)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Change VMGroupInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	LoggingInfo(hiscallInfo, start)
	return info.VMGroup, nil
}
func (nlbHandler *IbmNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "AddVMs()")
	start := call.Start()
	rawNLB, err := nlbHandler.getRawNLBByName(nlbIID.NameId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Add VMs. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	if rawNLB.Pools == nil || len(rawNLB.Pools) < 1 {
		changeErr := errors.New(fmt.Sprintf("Failed to Add VMs. err = vmGroup does not exist within that NLB"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	updatePoolId := *rawNLB.Pools[0].ID
	updateNLBId := *rawNLB.ID

	// Current vmGroup
	vmGroup, err := nlbHandler.getVMGroup(rawNLB)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Add VMs. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	currentVMIIDs := *vmGroup.VMs
	addedVm := *vmIIDs
	for _, addVMIId := range addedVm {
		for _, currentIId := range currentVMIIDs {
			if strings.EqualFold(addVMIId.NameId, currentIId.NameId) {
				return irs.VMGroupInfo{}, errors.New("can't add already exist vm")
			}
		}
	}

	addedVm = append(currentVMIIDs, addedVm...)
	vmGroup.VMs = &addedVm

	memberArrayOption, err := nlbHandler.convertCBVMGroupToIbmPoolMember(vmGroup)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Add VMs. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	updateMemberOption := vpcv1.ReplaceLoadBalancerPoolMembersOptions{}
	updateMemberOption.SetPoolID(updatePoolId)
	updateMemberOption.SetLoadBalancerID(updateNLBId)
	updateMemberOption.SetMembers(memberArrayOption)

	_, _, err = nlbHandler.VpcService.ReplaceLoadBalancerPoolMembersWithContext(nlbHandler.Ctx, &updateMemberOption)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Add VMs. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	updatedNLB, err := nlbHandler.checkUpdatableNLB(updateNLBId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Add VMs. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	info, err := nlbHandler.setterNLB(updatedNLB)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Add VMs. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	LoggingInfo(hiscallInfo, start)
	return info.VMGroup, nil
}
func (nlbHandler *IbmNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "RemoveVMs()")
	start := call.Start()
	rawNLB, err := nlbHandler.getRawNLBByName(nlbIID.NameId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Remove VMs. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return false, changeErr
	}
	if rawNLB.Pools == nil || len(rawNLB.Pools) < 1 {
		changeErr := errors.New(fmt.Sprintf("Failed to Remove VMs. err = vmGroup does not exist within that NLB"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return false, changeErr
	}

	updatePoolId := *rawNLB.Pools[0].ID
	updateNLBId := *rawNLB.ID

	// Current vmGroup
	vmGroup, err := nlbHandler.getVMGroup(rawNLB)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Remove VMs. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return false, changeErr
	}

	currentVMIIDs := *vmGroup.VMs

	removedIIDs := *vmIIDs
	for _, removedVMIId := range removedIIDs {
		existCheck := false
		for _, currentIId := range currentVMIIDs {
			if strings.EqualFold(removedVMIId.NameId, currentIId.NameId) {
				existCheck = true
				break
			}
		}
		if !existCheck {
			changeErr := errors.New(fmt.Sprintf("Failed to Remove VMs. err = can't remove a vm that does not exist"))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return false, changeErr
		}

	}
	var updatevmIId []irs.IID
	for _, currentIId := range currentVMIIDs {
		removeCheck := false
		for _, removedVMIId := range removedIIDs {
			if strings.EqualFold(removedVMIId.NameId, currentIId.NameId) {
				removeCheck = true
				break
			}
		}
		if !removeCheck {
			updatevmIId = append(updatevmIId, currentIId)
		}
	}

	vmGroup.VMs = &updatevmIId

	updateMemberArrayOption, err := nlbHandler.convertCBVMGroupToIbmPoolMember(vmGroup)

	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Remove VMs. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return false, changeErr
	}
	updateMemberOption := vpcv1.ReplaceLoadBalancerPoolMembersOptions{}
	updateMemberOption.SetPoolID(updatePoolId)
	updateMemberOption.SetLoadBalancerID(updateNLBId)
	updateMemberOption.SetMembers(updateMemberArrayOption)

	_, _, err = nlbHandler.VpcService.ReplaceLoadBalancerPoolMembersWithContext(nlbHandler.Ctx, &updateMemberOption)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Remove VMs. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return false, changeErr
	}
	_, err = nlbHandler.checkUpdatableNLB(updateNLBId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Remove VMs. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return false, changeErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
func (nlbHandler *IbmNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "GetVMGroupHealthInfo()")
	start := call.Start()

	rawNLB, err := nlbHandler.getRawNLBByName(nlbIID.NameId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Get VMGroupHealthInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthInfo{}, changeErr
	}
	if rawNLB.Pools == nil || len(rawNLB.Pools) < 1 {
		changeErr := errors.New(fmt.Sprintf("Failed to Get VMGroupHealthInfo. err = VMGroup does not exist within that NLB"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthInfo{}, changeErr
	}

	updatePoolId := *rawNLB.Pools[0].ID
	updateNLBId := *rawNLB.ID

	poolOption := vpcv1.GetLoadBalancerPoolOptions{}
	poolOption.SetLoadBalancerID(updateNLBId)
	poolOption.SetID(updatePoolId)

	rawPool, _, err := nlbHandler.VpcService.GetLoadBalancerPoolWithContext(nlbHandler.Ctx, &poolOption)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Get VMGroupHealthInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthInfo{}, changeErr
	}
	if rawPool.Members == nil || len(rawPool.Members) < 1 {
		// not exist
		return irs.HealthInfo{}, nil
	}

	poolMemberListOptions := vpcv1.ListLoadBalancerPoolMembersOptions{}
	poolMemberListOptions.SetLoadBalancerID(updateNLBId)
	poolMemberListOptions.SetPoolID(updatePoolId)

	members, _, err := nlbHandler.VpcService.ListLoadBalancerPoolMembersWithContext(nlbHandler.Ctx, &poolMemberListOptions)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Get VMGroupHealthInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthInfo{}, changeErr
	}
	info, err := nlbHandler.getHealthInfoByMembers(members.Members)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Get VMGroupHealthInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthInfo{}, changeErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}
func (nlbHandler *IbmNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "ChangeHealthCheckerInfo()")
	start := call.Start()
	rawNLB, err := nlbHandler.getRawNLBByName(nlbIID.NameId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Change HealthCheckerInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	if rawNLB.Pools == nil || len(rawNLB.Pools) < 1 {
		changeErr := errors.New(fmt.Sprintf("Failed to Change HealthCheckerInfo. err = VMGroup does not exist within that NLB"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	updatePoolId := *rawNLB.Pools[0].ID
	updateNLBId := *rawNLB.ID

	poolOption := vpcv1.GetLoadBalancerPoolOptions{}
	poolOption.SetLoadBalancerID(updateNLBId)
	poolOption.SetID(updatePoolId)

	rawPool, _, err := nlbHandler.VpcService.GetLoadBalancerPoolWithContext(nlbHandler.Ctx, &poolOption)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Change HealthCheckerInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}

	currentHealthCheckerInfo, err := convertHealthMonitorToHealthCheckerInfo(rawPool.HealthMonitor)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Change HealthCheckerInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	if reflect.DeepEqual(healthChecker, currentHealthCheckerInfo) {
		rawNLB, err := nlbHandler.getRawNLBByName(nlbIID.NameId)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to Change HealthCheckerInfo. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.HealthCheckerInfo{}, changeErr
		}
		info, err := nlbHandler.setterNLB(rawNLB)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to Change HealthCheckerInfo. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.HealthCheckerInfo{}, changeErr
		}
		// not Update
		return info.HealthChecker, nil
	} else {
		poolHealthOption, err := convertCBHealthToIbmHealth(healthChecker)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to Change HealthCheckerInfo. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.HealthCheckerInfo{}, changeErr
		}
		updateMaps := make(map[string]interface{})
		updateMaps["health_monitor"] = &poolHealthOption

		updatePoolOption := vpcv1.UpdateLoadBalancerPoolOptions{}
		updatePoolOption.SetID(updatePoolId)
		updatePoolOption.SetLoadBalancerID(updateNLBId)
		updatePoolOption.SetLoadBalancerPoolPatch(updateMaps)

		_, err = nlbHandler.checkUpdatableNLB(updateNLBId)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to Change HealthCheckerInfo. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.HealthCheckerInfo{}, changeErr
		}
		_, _, err = nlbHandler.VpcService.UpdateLoadBalancerPoolWithContext(nlbHandler.Ctx, &updatePoolOption)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to Change HealthCheckerInfo. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.HealthCheckerInfo{}, changeErr
		}
		_, err = nlbHandler.checkUpdatableNLB(updateNLBId)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to Change HealthCheckerInfo. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.HealthCheckerInfo{}, changeErr
		}
	}
	rawNLB, err = nlbHandler.getRawNLBByName(nlbIID.NameId)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Change HealthCheckerInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	info, err := nlbHandler.setterNLB(rawNLB)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to Change HealthCheckerInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	LoggingInfo(hiscallInfo, start)
	return info.HealthChecker, nil
}

func checkVmGroupHealth(health string) bool {
	if health == "" {
		return false
	}
	if NLBVMStatus(health) == NLBVMStatusOK {
		return true
	}
	return false
}

func (nlbHandler *IbmNLBHandler) setterNLB(nlb vpcv1.LoadBalancer) (*irs.NLBInfo, error) {
	nlbInfo := irs.NLBInfo{
		IId: irs.IID{
			NameId:   *nlb.Name,
			SystemId: *nlb.ID,
		},
		Scope: string(NLBRegionType),
		Type:  string(NLBPublicType),
	}

	if !*nlb.IsPublic {
		nlbInfo.Type = string(NLBInternalType)
	}

	var apiCallFunctions []func(rawNlb vpcv1.LoadBalancer, irsNlb *irs.NLBInfo) error
	// define and register get VPC IID
	apiCallFunctions = append(apiCallFunctions, func(nlbInner vpcv1.LoadBalancer, nlbInfoInner *irs.NLBInfo) error {
		vpcIId, err := nlbHandler.getVPCIID(nlbInner)
		if err != nil {
			return err
		}
		nlbInfoInner.VpcIID = vpcIId
		return nil
	})
	// define and register get VM Group
	apiCallFunctions = append(apiCallFunctions, func(nlbInner vpcv1.LoadBalancer, nlbInfoInner *irs.NLBInfo) error {
		vmGroup, err := nlbHandler.getVMGroup(nlb)
		if err != nil {
			return err
		}
		nlbInfo.VMGroup = vmGroup
		return nil
	})
	// define and register get Listener
	apiCallFunctions = append(apiCallFunctions, func(nlbInner vpcv1.LoadBalancer, nlbInfoInner *irs.NLBInfo) error {
		listenerInfo, err := nlbHandler.getListenerInfo(nlb)
		if err != nil {
			return err
		}
		nlbInfo.Listener = listenerInfo
		return nil
	})
	// define and register get HealthChecker
	apiCallFunctions = append(apiCallFunctions, func(nlbInner vpcv1.LoadBalancer, nlbInfoInner *irs.NLBInfo) error {
		healthCheckerInfo, err := nlbHandler.getHealthCheckerInfo(nlb)
		if err != nil {
			return err
		}
		nlbInfo.HealthChecker = healthCheckerInfo
		return nil
	})

	// prepare api call
	var wait sync.WaitGroup
	wait.Add(len(apiCallFunctions))
	var errList []string

	// define api call done behaviour
	callAndWaitGroup := func(call func(rawNlb vpcv1.LoadBalancer, irsNlb *irs.NLBInfo) error) {
		defer wait.Done()
		if err := call(nlb, &nlbInfo); err != nil {
			errList = append(errList, err.Error())
		}
	}

	// asynchronously call registered apis
	for _, apiCallFunction := range apiCallFunctions {
		go callAndWaitGroup(apiCallFunction)
	}

	// wait for all registered api call is done
	wait.Wait()

	// return all errors if occurs
	if len(errList) > 0 {
		return &irs.NLBInfo{}, errors.New(strings.Join(errList, "\n\t"))
	}

	nlbInfo.CreatedTime = time.Time(*nlb.CreatedAt)

	return &nlbInfo, nil
}

func (nlbHandler *IbmNLBHandler) getRawNLBById(nlbID string) (vpcv1.LoadBalancer, error) {
	if nlbID == "" {
		return vpcv1.LoadBalancer{}, errors.New("id cannot be an empty value")
	}
	options := &vpcv1.GetLoadBalancerOptions{}
	options.SetID(nlbID)
	lb, _, err := nlbHandler.VpcService.GetLoadBalancer(options)
	if err != nil {
		return vpcv1.LoadBalancer{}, err
	}
	return *lb, nil
}
func (nlbHandler *IbmNLBHandler) getListenerInfo(nlb vpcv1.LoadBalancer) (irs.ListenerInfo, error) {
	listenerInfo := irs.ListenerInfo{}
	if len(nlb.PublicIps) > 0 {
		listenerInfo.IP = *nlb.PublicIps[0].Address
	}
	if len(nlb.Listeners) > 0 {
		listeners := nlb.Listeners
		listenerOptions := vpcv1.GetLoadBalancerListenerOptions{}
		listenerOptions.SetID(*listeners[0].ID)
		listenerOptions.SetLoadBalancerID(*nlb.ID)
		listener, _, err := nlbHandler.VpcService.GetLoadBalancerListenerWithContext(nlbHandler.Ctx, &listenerOptions)
		if err != nil {
			return irs.ListenerInfo{}, err
		}
		listenerInfo.CspID = *listener.ID
		listenerInfo.Protocol = strings.ToUpper(*listener.Protocol)
		listenerInfo.Port = strconv.Itoa(int(*listener.Port))
	}
	return listenerInfo, nil
}
func (nlbHandler *IbmNLBHandler) getVPCIID(nlb vpcv1.LoadBalancer) (irs.IID, error) {
	if len(nlb.Subnets) < 1 {
		return irs.IID{}, errors.New("not found Subnets")
	}
	cbOnlyOneSubnet := nlb.Subnets[0]

	getSubnetOptions := vpcv1.GetSubnetOptions{}
	getSubnetOptions.SetID(*cbOnlyOneSubnet.ID)
	// TODO : VPC 정보 GET
	rawSubnet, _, err := nlbHandler.VpcService.GetSubnetWithContext(nlbHandler.Ctx, &getSubnetOptions)
	if err != nil {
		return irs.IID{}, errors.New("not found Subnets")
	}
	if rawSubnet.VPC != nil {
		return irs.IID{
			NameId:   *rawSubnet.VPC.Name,
			SystemId: *rawSubnet.VPC.ID,
		}, nil
	}
	return irs.IID{}, errors.New("not found Subnets")
}

func (nlbHandler *IbmNLBHandler) getHealthCheckerInfo(nlb vpcv1.LoadBalancer) (irs.HealthCheckerInfo, error) {
	healthCheckerInfo := irs.HealthCheckerInfo{}
	if len(nlb.Pools) > 0 {
		backendPools := nlb.Pools
		getPoolOption := vpcv1.GetLoadBalancerPoolOptions{}
		getPoolOption.SetID(*backendPools[0].ID)
		getPoolOption.SetLoadBalancerID(*nlb.ID)
		cbOnlyOneBackendPool, _, err := nlbHandler.VpcService.GetLoadBalancerPoolWithContext(nlbHandler.Ctx, &getPoolOption)
		if err != nil {
			return irs.HealthCheckerInfo{}, err
		}
		healthCheckerInfo, err = convertHealthMonitorToHealthCheckerInfo(cbOnlyOneBackendPool.HealthMonitor)
		if err != nil {
			return irs.HealthCheckerInfo{}, err
		}
	} else {
		return healthCheckerInfo, errors.New("VMGroup does not exist within that NLB")
	}
	return healthCheckerInfo, nil
}

func convertHealthMonitorToHealthCheckerInfo(rawHealthMonitor *vpcv1.LoadBalancerPoolHealthMonitor) (irs.HealthCheckerInfo, error) {
	healthCheckerInfo := irs.HealthCheckerInfo{}
	if rawHealthMonitor != nil {
		port := "-1"
		if rawHealthMonitor.Port != nil {
			port = strconv.Itoa(int(*rawHealthMonitor.Port))
		}
		healthCheckerInfo.Port = port
		healthCheckerInfo.Protocol = strings.ToUpper(*rawHealthMonitor.Type)
		healthCheckerInfo.Interval = int(*rawHealthMonitor.Delay)
		healthCheckerInfo.Threshold = int(*rawHealthMonitor.MaxRetries)
		healthCheckerInfo.Timeout = int(*rawHealthMonitor.Timeout)
		return healthCheckerInfo, nil
	}
	return healthCheckerInfo, errors.New("not exist HealthMonitor")
}

func getCreateListenerOptions(nlbReqInfo irs.NLBInfo, poolName *string) ([]vpcv1.LoadBalancerListenerPrototypeLoadBalancerContext, error) {
	if poolName == nil || *poolName == "" {
		return nil, errors.New("invalid PoolName")
	}
	listenerDefaultPoolName := *poolName
	listenerPort, err := strconv.Atoi(nlbReqInfo.Listener.Port)
	if err != nil {
		return nil, err
	}
	if listenerPort > 65535 || listenerPort < 1 {
		return nil, errors.New("ibm-NLB Listener provides an port of between 1 and 65535")
	}

	listenerPort64 := int64(listenerPort)

	listenerProtocol := ""
	switch strings.ToLower(nlbReqInfo.Listener.Protocol) {
	case "tcp", "udp":
		listenerProtocol = strings.ToLower(nlbReqInfo.Listener.Protocol)
	default:
		return nil, errors.New("ibm-NLB Listener provides only TCP and UDP protocols")
	}

	listener := vpcv1.LoadBalancerListenerPrototypeLoadBalancerContext{
		Port:     &listenerPort64,
		Protocol: &listenerProtocol,
		DefaultPool: &vpcv1.LoadBalancerPoolIdentityByName{
			Name: &listenerDefaultPoolName,
		},
	}

	var listenerArray = []vpcv1.LoadBalancerListenerPrototypeLoadBalancerContext{
		listener,
	}
	return listenerArray, nil
}

func (nlbHandler *IbmNLBHandler) getCreatePoolOptions(nlbReqInfo irs.NLBInfo) ([]vpcv1.LoadBalancerPoolPrototype, error) {
	// memberOption := vpcv1.LoadBalancerPoolMemberTargetPrototype{}
	var poolArray []vpcv1.LoadBalancerPoolPrototype

	memberArrayOption, err := nlbHandler.convertCBVMGroupToIbmPoolMember(nlbReqInfo.VMGroup)
	if err != nil {
		return nil, err
	}

	poolHealthOption, err := convertCBHealthToIbmHealth(nlbReqInfo.HealthChecker)
	if err != nil {
		return nil, err
	}

	poolAlgorithm := "round_robin"
	poolProtocol := ""
	switch strings.ToLower(nlbReqInfo.VMGroup.Protocol) {
	case "tcp", "udp":
		poolProtocol = strings.ToLower(nlbReqInfo.VMGroup.Protocol)
	default:
		return nil, errors.New("ibm-NLB VMGroup provides only TCP and UDP protocols")
	}

	poolName := generatePoolName(nlbReqInfo.VMGroup.Port)

	poolOption := vpcv1.LoadBalancerPoolPrototype{
		Algorithm:     core.StringPtr(poolAlgorithm),
		Protocol:      core.StringPtr(poolProtocol),
		Name:          core.StringPtr(poolName),
		HealthMonitor: &poolHealthOption,
		Members:       memberArrayOption,
	}
	poolArray = append(poolArray, poolOption)
	return poolArray, nil
}

func (nlbHandler *IbmNLBHandler) convertCBVMGroupToIbmPoolMember(vmGroup irs.VMGroupInfo) ([]vpcv1.LoadBalancerPoolMemberPrototype, error) {
	vms := *vmGroup.VMs
	memberPort, err := strconv.Atoi(vmGroup.Port)
	if err != nil {
		return nil, err
	}
	if memberPort > 65535 || memberPort < 1 {
		return nil, errors.New("ibm-NLB vmGroup Port provides an port of between 1 and 65535")
	}
	memberArrayOption := make([]vpcv1.LoadBalancerPoolMemberPrototype, len(vms))
	memberPort64 := int64(memberPort)
	var allVMS *[]vpcv1.Instance
	for i, vmIID := range vms {
		vmSystemId := vmIID.SystemId
		if vmSystemId == "" {
			if allVMS == nil {
				allVMS, err = nlbHandler.getRawVMList()
				if err != nil {
					return nil, err
				}
			}
			for _, rawVM := range *allVMS {
				if strings.EqualFold(*rawVM.Name, vmIID.NameId) {
					vmSystemId = *rawVM.ID
				}
			}
			if vmSystemId == "" {
				return nil, errors.New(fmt.Sprintf("not found VM %s", vmIID.NameId))
			}
		}
		memberArrayOption[i] = vpcv1.LoadBalancerPoolMemberPrototype{
			Port: core.Int64Ptr(memberPort64),
			Target: &vpcv1.LoadBalancerPoolMemberTargetPrototype{
				ID: core.StringPtr(vmSystemId),
			},
		}
	}
	return memberArrayOption, nil
}

func convertCBHealthToIbmHealth(healthChecker irs.HealthCheckerInfo) (vpcv1.LoadBalancerPoolHealthMonitorPrototype, error) {
	HealthProtocol := ""
	switch strings.ToLower(healthChecker.Protocol) {
	case "tcp", "http":
		HealthProtocol = strings.ToLower(healthChecker.Protocol)
	default:
		return vpcv1.LoadBalancerPoolHealthMonitorPrototype{}, errors.New("ibm-NLB healthCheck provides only HTTP and TCP protocols")
	}

	HealthDelay := int64(healthChecker.Interval)
	if HealthDelay > 60 || HealthDelay < 2 {
		return vpcv1.LoadBalancerPoolHealthMonitorPrototype{}, errors.New("ibm-NLB healthCheck provides an interval of between 2 and 60 seconds")
	}
	HealthMaxRetries := int64(healthChecker.Threshold)
	if HealthMaxRetries > 10 || HealthMaxRetries < 1 {
		return vpcv1.LoadBalancerPoolHealthMonitorPrototype{}, errors.New("ibm-NLB healthCheck provides an Threshold of between 1 and 10")
	}
	HealthTimeOut := int64(healthChecker.Timeout)
	if HealthTimeOut > 59 || HealthTimeOut < 1 {
		return vpcv1.LoadBalancerPoolHealthMonitorPrototype{}, errors.New("ibm-NLB healthCheck provides an Timeout of between 1 and 59 seconds")
	}
	HealthPort, err := strconv.Atoi(healthChecker.Port)
	if err != nil {
		return vpcv1.LoadBalancerPoolHealthMonitorPrototype{}, err
	}
	if HealthPort > 65535 || HealthPort < 1 {
		return vpcv1.LoadBalancerPoolHealthMonitorPrototype{}, errors.New("ibm-NLB healthCheck provides an port of between 1 and 65535")
	}

	HealthPort64 := int64(HealthPort)
	opts := vpcv1.LoadBalancerPoolHealthMonitorPrototype{
		Type:       core.StringPtr(HealthProtocol),
		Delay:      core.Int64Ptr(HealthDelay),
		MaxRetries: core.Int64Ptr(HealthMaxRetries),
		Timeout:    core.Int64Ptr(HealthTimeOut),
		Port:       core.Int64Ptr(HealthPort64),
	}
	if HealthProtocol == "http" {
		opts.URLPath = core.StringPtr("/")
	}
	return opts, nil
}

func (nlbHandler *IbmNLBHandler) getMatchVMIIdAndFirstSubnetId(vmIIDs *[]irs.IID) ([]irs.IID, string, error) {
	// nameId => systemID, NameId
	if vmIIDs == nil || len(*vmIIDs) < 1 {
		return nil, "", errors.New("vmIIDs is empty")
	}
	allVMS, err := nlbHandler.getRawVMList()
	if err != nil {
		return nil, "", err
	}
	subnetId := ""
	vms := *vmIIDs
	for i, vmIID := range vms {
		errCheck := true
		for _, rawVM := range *allVMS {
			if strings.EqualFold(vmIID.NameId, *rawVM.Name) {
				if subnetId == "" {
					subnetId = *rawVM.PrimaryNetworkInterface.Subnet.ID
				}
				vms[i].SystemId = *rawVM.ID
				errCheck = false
				break
			}
		}
		if errCheck {
			return nil, "", errors.New("not found vm")
		}
	}
	return vms, subnetId, nil
}

func (nlbHandler *IbmNLBHandler) createNLB(nlbReqInfo irs.NLBInfo) (vpcv1.LoadBalancer, error) {
	exist, err := nlbHandler.existNLBByName(nlbReqInfo.IId.NameId)
	if err != nil {
		return vpcv1.LoadBalancer{}, err
	}
	if exist {
		return vpcv1.LoadBalancer{}, errors.New(fmt.Sprintf("already exist NLB : %s", nlbReqInfo.IId.NameId))
	}

	vms, subnetId, err := nlbHandler.getMatchVMIIdAndFirstSubnetId(nlbReqInfo.VMGroup.VMs)

	if err != nil {
		return vpcv1.LoadBalancer{}, err
	}

	nlbReqInfo.VMGroup.VMs = &vms

	// Set - BaseOption
	createNLBOptions := vpcv1.CreateLoadBalancerOptions{}
	createNLBOptions.SetIsPublic(true)
	createNLBOptions.SetName(nlbReqInfo.IId.NameId)
	LoadBalancerProfileName := "network-fixed"
	createNLBOptions.SetProfile(&vpcv1.LoadBalancerProfileIdentity{
		Name: &LoadBalancerProfileName,
	})

	// Set - subnet
	var subnetArray = []vpcv1.SubnetIdentityIntf{
		&vpcv1.SubnetIdentity{
			ID: &subnetId,
		},
	}

	poolArray, err := nlbHandler.getCreatePoolOptions(nlbReqInfo)
	if err != nil {
		return vpcv1.LoadBalancer{}, err
	}
	poolName := poolArray[0].Name

	listenerArray, err := getCreateListenerOptions(nlbReqInfo, poolName)
	if err != nil {
		return vpcv1.LoadBalancer{}, err
	}

	createNLBOptions.SetSubnets(subnetArray)
	createNLBOptions.SetPools(poolArray)
	createNLBOptions.SetListeners(listenerArray)

	nlb, _, err := nlbHandler.VpcService.CreateLoadBalancerWithContext(nlbHandler.Ctx, &createNLBOptions)
	if err != nil {
		return vpcv1.LoadBalancer{}, err
	}
	_, err = nlbHandler.checkUpdatableNLB(*nlb.ID)
	if err != nil {
		return vpcv1.LoadBalancer{}, err
	}
	return *nlb, nil
}

func (nlbHandler *IbmNLBHandler) getVMGroup(nlb vpcv1.LoadBalancer) (irs.VMGroupInfo, error) {
	vmGroup := irs.VMGroupInfo{}
	if len(nlb.Pools) > 0 {
		backendPools := nlb.Pools
		getPoolOption := vpcv1.GetLoadBalancerPoolOptions{}
		getPoolOption.SetID(*backendPools[0].ID)
		getPoolOption.SetLoadBalancerID(*nlb.ID)
		cbOnlyOneBackendPool, _, err := nlbHandler.VpcService.GetLoadBalancerPoolWithContext(nlbHandler.Ctx, &getPoolOption)
		if err != nil {
			return irs.VMGroupInfo{}, err
		}
		vmGroup.CspID = *cbOnlyOneBackendPool.ID
		vmGroup.Protocol = strings.ToUpper(*cbOnlyOneBackendPool.Protocol)

		poolMemberListOptions := vpcv1.ListLoadBalancerPoolMembersOptions{}
		poolMemberListOptions.SetLoadBalancerID(*nlb.ID)
		poolMemberListOptions.SetPoolID(*cbOnlyOneBackendPool.ID)

		poolAllMembers, _, err := nlbHandler.VpcService.ListLoadBalancerPoolMembersWithContext(nlbHandler.Ctx, &poolMemberListOptions)
		if err != nil {
			return irs.VMGroupInfo{}, err
		}
		vmIIDs, err := nlbHandler.getAllVMIIDsByMembers(poolAllMembers.Members)
		if err != nil {
			return irs.VMGroupInfo{}, err
		}
		vmGroup.VMs = &vmIIDs
		portInt, err := nlbHandler.getPoolPort(*nlb.ID, *cbOnlyOneBackendPool.ID)
		if err != nil {
			return irs.VMGroupInfo{}, err
		}
		vmGroup.Port = strconv.Itoa(portInt)
	}
	return vmGroup, nil
}

func getMemberTarget(member vpcv1.LoadBalancerPoolMember) (vpcv1.LoadBalancerPoolMemberTarget, error) {
	target := member.Target

	targetJsonBytes, err := json.Marshal(target)
	if err != nil {
		return vpcv1.LoadBalancerPoolMemberTarget{}, err
	}
	var targetMember vpcv1.LoadBalancerPoolMemberTarget
	err = json.Unmarshal(targetJsonBytes, &targetMember)
	if err != nil {
		return vpcv1.LoadBalancerPoolMemberTarget{}, err
	}
	return targetMember, nil
}

func (nlbHandler *IbmNLBHandler) existNLBByName(nlbName string) (bool, error) {
	if nlbName == "" {
		return false, errors.New("name cannot be an empty value")
	}
	allNLBList, err := nlbHandler.getRawNLBList()
	if err != nil {
		return false, err
	}
	for _, rawNLB := range *allNLBList {
		if strings.EqualFold(nlbName, *rawNLB.Name) {
			return true, nil
		}
	}
	return false, nil
}

func (nlbHandler *IbmNLBHandler) getRawNLBByName(nlbName string) (vpcv1.LoadBalancer, error) {
	if nlbName == "" {
		return vpcv1.LoadBalancer{}, errors.New("name cannot be an empty value")
	}
	allNLBList, err := nlbHandler.getRawNLBList()
	if err != nil {
		return vpcv1.LoadBalancer{}, err
	}
	for _, rawNLB := range *allNLBList {
		if strings.EqualFold(nlbName, *rawNLB.Name) {
			return rawNLB, nil
		}
	}
	return vpcv1.LoadBalancer{}, errors.New(fmt.Sprintf("not found NLB : %s", nlbName))
}

func (nlbHandler *IbmNLBHandler) getRawVMList() (*[]vpcv1.Instance, error) {
	options := &vpcv1.ListInstancesOptions{}
	var list []vpcv1.Instance
	res, _, err := nlbHandler.VpcService.ListInstancesWithContext(nlbHandler.Ctx, options)
	if err != nil {
		return nil, err
	}

	for {
		if len(res.Instances) > 0 {
			list = append(list, res.Instances...)
		} else {
			break
		}
		nextStr := ""
		if res.Next != nil && res.Next.Href != nil {
			nextStr, _ = getNextHref(*res.Next.Href)
		}
		if nextStr != "" {
			options2 := &vpcv1.ListInstancesOptions{
				Start: core.StringPtr(nextStr),
			}
			res, _, err = nlbHandler.VpcService.ListInstancesWithContext(nlbHandler.Ctx, options2)
			if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}
	return &list, nil
}

func (nlbHandler *IbmNLBHandler) checkUpdatableNLB(nlbId string) (vpcv1.LoadBalancer, error) {
	if nlbId == "" {
		return vpcv1.LoadBalancer{}, errors.New("")
	}
	updatableCheckNLBOptions := vpcv1.GetLoadBalancerOptions{}
	updatableCheckNLBOptions.SetID(nlbId)
	var updatableCheckNLB *vpcv1.LoadBalancer
	var err error

	curRetryCnt := 0
	maxRetryCnt := 240
	for {
		updatableCheckNLB, _, err = nlbHandler.VpcService.GetLoadBalancerWithContext(nlbHandler.Ctx, &updatableCheckNLBOptions)
		if err == nil && *updatableCheckNLB.ProvisioningStatus == "active" {
			return *updatableCheckNLB, nil
		}
		time.Sleep(2 * time.Second)
		curRetryCnt++
		if curRetryCnt > maxRetryCnt {
			return vpcv1.LoadBalancer{}, errors.New(fmt.Sprintf("Failed to Update NLB VMGroup. err = exceeded maximum retry count %d", maxRetryCnt))
		}
	}
}

func (nlbHandler *IbmNLBHandler) getRawNLBList() (*[]vpcv1.LoadBalancer, error) {
	options := &vpcv1.ListLoadBalancersOptions{}
	var list []vpcv1.LoadBalancer
	res, _, err := nlbHandler.VpcService.ListLoadBalancersWithContext(nlbHandler.Ctx, options)
	if err != nil {
		return nil, err
	}

	for {
		if len(res.LoadBalancers) > 0 {
			list = append(list, res.LoadBalancers...)
		} else {
			break
		}
		nextStr := ""
		if res.Next != nil && res.Next.Href != nil {
			nextStr, _ = getNextHref(*res.Next.Href)
		}
		if nextStr != "" {
			listNLBOptions2 := &vpcv1.ListLoadBalancersOptions{
				Start: core.StringPtr(nextStr),
			}
			res, _, err = nlbHandler.VpcService.ListLoadBalancersWithContext(nlbHandler.Ctx, listNLBOptions2)
			if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}
	return &list, nil
}

func (nlbHandler *IbmNLBHandler) cleanerNLB(nlbIID irs.IID) (bool, error) {
	// Exist?
	exist, err := nlbHandler.existNLBByName(nlbIID.NameId)
	if err != nil {
		return false, err
	}
	if !exist {
		return false, errors.New("not found nlb")
	}
	rawNLB, err := nlbHandler.getRawNLBByName(nlbIID.NameId)
	if err != nil {
		return false, err
	}
	nlbId := *rawNLB.ID
	nlbDeleteOption := vpcv1.DeleteLoadBalancerOptions{}
	nlbDeleteOption.SetID(*rawNLB.ID)
	_, err = nlbHandler.VpcService.DeleteLoadBalancerWithContext(nlbHandler.Ctx, &nlbDeleteOption)
	if err != nil {
		return false, err
	}
	_, err = nlbHandler.waitDeletedNLB(nlbId)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (nlbHandler *IbmNLBHandler) waitDeletedNLB(nlbId string) (bool, error) {
	curRetryCnt := 0
	maxRetryCnt := 240
	for {
		curRetryCnt++
		list, err := nlbHandler.getRawNLBList()
		if err != nil {
			return false, err
		}
		exist := false
		for _, rawNLb := range *list {
			if *rawNLb.ID == nlbId {
				exist = true
			}
		}
		if !exist {
			return true, nil
		}
		time.Sleep(2 * time.Second)
		if curRetryCnt > maxRetryCnt {
			return false, errors.New(fmt.Sprintf("failed to create NLB, exceeded maximum retry count %d", maxRetryCnt))
		}
	}
}

func (nlbHandler *IbmNLBHandler) getHealthInfoByMembers(members []vpcv1.LoadBalancerPoolMember) (irs.HealthInfo, error) {
	var allVMS *[]vpcv1.Instance
	healthyVMs := make([]irs.IID, 0)
	unHealthyVMs := make([]irs.IID, 0)
	vmIIDs := make([]irs.IID, len(members))
	for i, member := range members {
		memberTarget, err := getMemberTarget(member)
		if err != nil {
			return irs.HealthInfo{}, err
		}
		// targetMember.ID = instanceID
		if memberTarget.ID != nil {
			instanceOptions := &vpcv1.GetInstanceOptions{}
			instanceOptions.SetID(*memberTarget.ID)
			rawVM, _, err := nlbHandler.VpcService.GetInstance(instanceOptions)
			if err != nil {
				return irs.HealthInfo{}, err
			}
			vmIIDs[i] = irs.IID{
				NameId:   *rawVM.Name,
				SystemId: *rawVM.ID,
			}
		} else if memberTarget.Address != nil {
			if allVMS == nil {
				allVMS, err = nlbHandler.getRawVMList()
				if err != nil {
					return irs.HealthInfo{}, err
				}
			}
			for _, rawVM := range *allVMS {
				if strings.EqualFold(*rawVM.PrimaryNetworkInterface.PrimaryIpv4Address, *memberTarget.Address) {
					vmIIDs[i] = irs.IID{
						NameId:   *rawVM.Name,
						SystemId: *rawVM.ID,
					}
				}
			}
		}
		if checkVmGroupHealth(*member.Health) {
			healthyVMs = append(healthyVMs, vmIIDs[i])
		} else {
			unHealthyVMs = append(unHealthyVMs, vmIIDs[i])
		}
	}
	return irs.HealthInfo{
		AllVMs:       &vmIIDs,
		HealthyVMs:   &healthyVMs,
		UnHealthyVMs: &unHealthyVMs,
	}, nil
}

func (nlbHandler *IbmNLBHandler) getPoolPort(nlbId string, poolId string) (int, error) {
	poolOption := vpcv1.GetLoadBalancerPoolOptions{}
	poolOption.SetLoadBalancerID(nlbId)
	poolOption.SetID(poolId)

	rawPool, _, err := nlbHandler.VpcService.GetLoadBalancerPoolWithContext(nlbHandler.Ctx, &poolOption)
	if err != nil {
		return 0, err
	}
	vmGroupPort, err := getPoolPortByPoolName(*rawPool.Name)
	if err != nil {
		poolMemberListOptions := vpcv1.ListLoadBalancerPoolMembersOptions{}
		poolMemberListOptions.SetLoadBalancerID(nlbId)
		poolMemberListOptions.SetPoolID(poolId)
		poolAllMembers, _, err := nlbHandler.VpcService.ListLoadBalancerPoolMembersWithContext(nlbHandler.Ctx, &poolMemberListOptions)
		if err != nil {
			return 0, err
		}
		if len(poolAllMembers.Members) > 0 {
			vmGroupPort = int(*poolAllMembers.Members[0].Port)
		} else {
			return 0, err
		}
	}
	return vmGroupPort, nil
}

func (nlbHandler *IbmNLBHandler) getAllVMIIDsByMembers(members []vpcv1.LoadBalancerPoolMember) ([]irs.IID, error) {
	var allVMS *[]vpcv1.Instance
	vmIIDs := make([]irs.IID, len(members))
	for i, member := range members {
		memberTarget, err := getMemberTarget(member)
		if err != nil {
			return nil, err
		}
		// targetMember.ID = instanceID
		if memberTarget.ID != nil {
			instanceOptions := &vpcv1.GetInstanceOptions{}
			instanceOptions.SetID(*memberTarget.ID)
			rawVM, _, err := nlbHandler.VpcService.GetInstance(instanceOptions)
			if err != nil {
				return nil, err
			}
			vmIIDs[i] = irs.IID{
				NameId:   *rawVM.Name,
				SystemId: *rawVM.ID,
			}
		} else if memberTarget.Address != nil {
			if allVMS == nil {
				allVMS, err = nlbHandler.getRawVMList()
				if err != nil {
					return nil, err
				}
			}
			for _, rawVM := range *allVMS {
				if strings.EqualFold(*rawVM.PrimaryNetworkInterface.PrimaryIpv4Address, *memberTarget.Address) {
					vmIIDs[i] = irs.IID{
						NameId:   *rawVM.Name,
						SystemId: *rawVM.ID,
					}
				}
			}
		}
	}
	return vmIIDs, nil
}

func getPoolPortByPoolName(poolName string) (int, error) {
	splits := strings.Split(poolName, "-")
	if len(splits) != 3 {
		return 0, errors.New("unable to get port information for vmGroup")
	}
	return strconv.Atoi(splits[1])
}

func generatePoolName(port string) string {
	prefix := fmt.Sprintf("backend-%s", port)
	return generateRandName(prefix)
}

func getNextHref(str string) (string, error) {
	href := str
	u, err := url.Parse(href)
	if err != nil {
		return "", err
	}
	paramMap, _ := url.ParseQuery(u.RawQuery)
	if paramMap != nil {
		safe := paramMap["start"]
		if safe != nil && len(safe) > 0 {
			return safe[0], nil
		}
	}
	return "", errors.New("NOT NEXT")
}
