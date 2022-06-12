package resources

import (
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/server"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/dna/adaptiveip"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/dna/loadbalancer"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type NLBType string
type NLBScope string
type NLBHealthType string

const (
	NLBPublicType   NLBType       = "PUBLIC"
	NLBInternalType NLBType       = "INTERNAL"
	NLBGlobalType   NLBScope      = "GLOBAL"
	NLBRegionType   NLBScope      = "REGION"
	NLBOpenHealth   NLBHealthType = "OPEN"
	//NLBClosedHealth   NLBHealthType = "CLOSED"
)

type ClouditNLBHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

//------ NLB Management
func (nlbHandler *ClouditNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (irs.NLBInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "NETWORKLOADBALANCE", nlbReqInfo.IId.NameId, "CreateNLB()")
	start := call.Start()
	// baseSet
	nlbCreateInfo := loadbalancer.LoadBalancerReqInfo{
		Name:      nlbReqInfo.IId.NameId,
		Scheduler: string(loadbalancer.LoadBalancerAlgorithmRoundRobin),
		MaxConn:   99999,
		StatsPort: 65535,
		Type:      string(loadbalancer.LoadBalancerExternalType),
	}
	if strings.EqualFold(nlbReqInfo.Type, string(loadbalancer.LoadBalancerInternalType)) {
		nlbCreateInfo.Type = string(loadbalancer.LoadBalancerInternalType)
	}
	// createIP
	ipInfo, err := nlbHandler.creatablePublicIP()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}
	nlbCreateInfo.IP = ipInfo.IP

	// HealthChecker
	switch strings.ToUpper(nlbReqInfo.HealthChecker.Protocol) {
	case "HTTP":
		nlbCreateInfo.MonitorType = strings.ToLower(nlbReqInfo.HealthChecker.Protocol)
		nlbCreateInfo.HttpUrl = "/"
	case "TCP":
		nlbCreateInfo.MonitorType = strings.ToLower(nlbReqInfo.HealthChecker.Protocol)
	default:
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = cloudit HealthChecker provides only TCP protocols"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}
	_, err = healthCheckPolicyValidation(nlbReqInfo.HealthChecker)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}

	nlbCreateInfo.UnhealthyThreshold = nlbReqInfo.HealthChecker.Threshold
	nlbCreateInfo.HealthyThreshold = nlbReqInfo.HealthChecker.Threshold
	nlbCreateInfo.IntervalTime = nlbReqInfo.HealthChecker.Interval
	nlbCreateInfo.ResponseTime = nlbReqInfo.HealthChecker.Timeout

	if !strings.EqualFold(strings.ToUpper(nlbReqInfo.VMGroup.Protocol), strings.ToUpper(nlbReqInfo.Listener.Protocol)) {
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = ListenerProtocol and vmGroupProtocol in cloudit must be the same"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}

	//VMGroup
	switch strings.ToUpper(nlbReqInfo.VMGroup.Protocol) {
	// case "HTTP", "TCP", "HTTPS":
	case "TCP":
		nlbCreateInfo.Protocol = strings.ToLower(nlbReqInfo.VMGroup.Protocol)
	default:
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = cloudit VMGroup provides only TCP protocols"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}
	vmGroupPortInt, err := strconv.Atoi(nlbReqInfo.VMGroup.Port)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = invalid vmGroup Port"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}
	if vmGroupPortInt < 1 || vmGroupPortInt > 65535 {
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = cloudit vmGroup Port provides an port of between 1 and 65535"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}

	rawVMList, err := nlbHandler.getRawVmList()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = %s",err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}
	memberReqList := make([]loadbalancer.LoadBalancerReqMember, len(*nlbReqInfo.VMGroup.VMs))
	for i, vmIId := range *nlbReqInfo.VMGroup.VMs {
		existCheck := false
		for _, rawVM := range rawVMList {
			if strings.EqualFold(vmIId.NameId, rawVM.Name) {
				existCheck = true
				memberReqList[i] = loadbalancer.LoadBalancerReqMember{
					MemberIp:   rawVM.PrivateIp,
					MemberPort: strconv.Itoa(vmGroupPortInt),
				}
			}
		}
		if !existCheck {
			createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = cloudit vmGroup VM : %s Not Exist", vmIId.NameId))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.NLBInfo{}, createErr
		}
	}

	nlbCreateInfo.Members = memberReqList

	// Listener
	listenerPortInt, err := strconv.Atoi(nlbReqInfo.Listener.Port)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = invalid vmGroup Port"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}
	if listenerPortInt < 1 || listenerPortInt > 65534 {
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = cloudit Listener Port provides an port of between 1 and 65534, port 65535 NLB Status Page Port"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}
	nlbCreateInfo.Port = listenerPortInt

	createRawNLB, err := nlbHandler.createNLB(nlbCreateInfo)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}
	rawNLB, err := nlbHandler.waitNLBCompleted(createRawNLB.Id)
	if err != nil {
		nlbHandler.deleteNLB(nlbReqInfo.IId)
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}

	rawNLB, err = nlbHandler.setDescriptionNLB(rawNLB.Id, fmt.Sprintf("vmgroupport:%s", nlbReqInfo.VMGroup.Port))
	if err != nil {
		nlbHandler.deleteNLB(nlbReqInfo.IId)
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}

	info, err := nlbHandler.setterNLB(rawNLB)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NLBInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}
func (nlbHandler *ClouditNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "NETWORKLOADBALANCE", "NLB", "ListNLB()")
	start := call.Start()
	nlbList, err := nlbHandler.getRawNLBList()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List NLB. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	nlbInfoList := make([]*irs.NLBInfo, len(nlbList))
	for i, nlb := range nlbList {
		info, err := nlbHandler.setterNLB(nlb)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List NLB. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}
		nlbInfoList[i] = &info
	}
	LoggingInfo(hiscallInfo, start)
	return nlbInfoList, nil
}
func (nlbHandler *ClouditNLBHandler) GetNLB(nlbIID irs.IID) (irs.NLBInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "NETWORKLOADBALANCE", nlbIID.NameId, "DeleteNLB()")
	start := call.Start()
	nlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get NLB. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.NLBInfo{}, getErr
	}
	info, err := nlbHandler.setterNLB(nlb)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get NLB. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.NLBInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}
func (nlbHandler *ClouditNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "NETWORKLOADBALANCE", nlbIID.NameId, "DeleteNLB()")
	start := call.Start()
	result, err := nlbHandler.deleteNLB(nlbIID)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete NLB. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)
	return result, err
}

//------ Frontend Control
func (nlbHandler *ClouditNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "NETWORKLOADBALANCE", nlbIID.NameId, "ChangeListener()")
	start := call.Start()
	LoggingInfo(hiscallInfo, start)
	cblogger.Info("CLOUDIT_CANNOT_CHANGE_LISTENERINFO", start)
	return irs.ListenerInfo{}, errors.New("CLOUDIT_CANNOT_CHANGE_LISTENERINFO")
}

//------ Backend Control
func (nlbHandler *ClouditNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "NETWORKLOADBALANCE", nlbIID.NameId, "ChangeVMGroupInfo()")
	start := call.Start()
	rawNLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	err = nlbHandler.checkValidChangeVMGroupInfo(rawNLB, vmGroup)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	equalCheck, err := nlbHandler.checkEqualVMGroupInfo(rawNLB, vmGroup)

	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	nlbInfo, err := nlbHandler.setterNLB(rawNLB)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo. err = invalid vmGroup Port"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	if equalCheck {
		LoggingInfo(hiscallInfo, start)
		return nlbInfo.VMGroup, nil
	}

	oldVmIIds := *nlbInfo.VMGroup.VMs

	removeMembers := make([]string, len(rawNLB.Members))
	for i, mem := range rawNLB.Members {
		removeMembers[i] = mem.Id
	}

	_, err = nlbHandler.deleteMembers(rawNLB.Id, removeMembers)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	newPort := vmGroup.Port
	rawNLB, err = nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	addMembers := make([]loadbalancer.AddMemberInfo, len(oldVmIIds))
	for i, vm := range oldVmIIds {
		for _, member := range rawNLB.Members {
			if strings.EqualFold(member.ServerName, vm.NameId) {
				return irs.VMGroupInfo{}, errors.New("can't add already exist vm")
			}
		}
		rawVm, err := nlbHandler.getRawVmByName(vm.NameId)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.VMGroupInfo{}, changeErr
		}
		addMembers[i] = loadbalancer.AddMemberInfo{
			Network:    rawVm.SubnetAddr,
			MemberIp:   rawVm.PrivateIp,
			MemberPort: newPort,
			HostName:   rawVm.HostName,
		}
	}

	_, err = nlbHandler.addMembers(rawNLB.Id, addMembers)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	descriptionString := fmt.Sprintf("vmgroupport:%s", vmGroup.Port)
	rawNLB, err = nlbHandler.setDescriptionNLB(rawNLB.Id, descriptionString)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo. err = invalid vmGroup Port"))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	rawNLB, err = nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	info, err := nlbHandler.setterNLB(rawNLB)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	LoggingInfo(hiscallInfo, start)
	return info.VMGroup, nil
}
func (nlbHandler *ClouditNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "NETWORKLOADBALANCE", nlbIID.NameId, "AddVMs()")
	start := call.Start()
	rawNLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		addErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = %s", err.Error()))
		cblogger.Error(addErr.Error())
		LoggingError(hiscallInfo, addErr)
		return irs.VMGroupInfo{}, addErr
	}
	port, err := nlbHandler.getVMGroupPort(rawNLB.Id)
	if err != nil {
		addErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = %s", err.Error()))
		cblogger.Error(addErr.Error())
		LoggingError(hiscallInfo, addErr)
		return irs.VMGroupInfo{}, addErr
	}

	addMembers := make([]loadbalancer.AddMemberInfo, len(*vmIIDs))
	for i, vm := range *vmIIDs {
		for _, member := range rawNLB.Members {
			if strings.EqualFold(member.ServerName, vm.NameId) {

				return irs.VMGroupInfo{}, errors.New("can't add already exist vm")
			}
		}
		rawVm, err := nlbHandler.getRawVmByName(vm.NameId)
		if err != nil {
			addErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = %s", err.Error()))
			cblogger.Error(addErr.Error())
			LoggingError(hiscallInfo, addErr)
			return irs.VMGroupInfo{}, addErr
		}
		addMembers[i] = loadbalancer.AddMemberInfo{
			Network:    rawVm.SubnetAddr,
			MemberIp:   rawVm.PrivateIp,
			MemberPort: port,
			HostName:   rawVm.HostName,
		}
	}
	updatedNLB, err := nlbHandler.addMembers(rawNLB.Id, addMembers)
	if err != nil {
		addErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = %s", err.Error()))
		cblogger.Error(addErr.Error())
		LoggingError(hiscallInfo, addErr)
		return irs.VMGroupInfo{}, addErr
	}
	info, err := nlbHandler.setterNLB(updatedNLB)
	if err != nil {
		addErr := errors.New(fmt.Sprintf("Failed to AddVMs. err = %s", err.Error()))
		cblogger.Error(addErr.Error())
		LoggingError(hiscallInfo, addErr)
		return irs.VMGroupInfo{}, addErr
	}
	LoggingInfo(hiscallInfo, start)
	return info.VMGroup, nil
}
func (nlbHandler *ClouditNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "NETWORKLOADBALANCE", nlbIID.NameId, "RemoveVMs()")
	start := call.Start()
	rawNLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to RemoveVMs. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	removeMembers := make([]string, len(*vmIIDs))

	for i, vm := range *vmIIDs {
		existCheck := false
		for _, member := range rawNLB.Members {
			if strings.EqualFold(member.ServerName, vm.NameId) {
				existCheck = true
				removeMembers[i] = member.Id
			}
		}
		if !existCheck {
			getErr := errors.New(fmt.Sprintf("Failed to RemoveVMs. err = not exist vm : %s", vm.NameId))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return false, getErr
		}
	}
	result, err := nlbHandler.deleteMembers(rawNLB.Id, removeMembers)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to RemoveVMs. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return result, nil
}
func (nlbHandler *ClouditNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "NETWORKLOADBALANCE", nlbIID.NameId, "GetVMGroupHealthInfo()")
	start := call.Start()
	rawNLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to GetVMGroupHealthInfo. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.HealthInfo{}, getErr
	}
	allVmIIds := make([]irs.IID, len(rawNLB.Members))
	healthVmIIds := make([]irs.IID, 0)
	unHealthVmIIds := make([]irs.IID, 0)
	for i, member := range rawNLB.Members {
		rawVm, err := nlbHandler.getRawVmByName(member.ServerName)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to GetVMGroupHealthInfo. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return irs.HealthInfo{}, getErr
		}
		vmIID := irs.IID{
			NameId:   member.ServerName,
			SystemId: rawVm.ID,
		}
		allVmIIds[i] = vmIID
		if strings.EqualFold(member.HealthState, string(NLBOpenHealth)) {
			healthVmIIds = append(healthVmIIds, vmIID)
		} else {
			unHealthVmIIds = append(unHealthVmIIds, vmIID)
		}
	}
	LoggingInfo(hiscallInfo, start)
	return irs.HealthInfo{
		AllVMs:       &allVmIIds,
		UnHealthyVMs: &unHealthVmIIds,
		HealthyVMs:   &healthVmIIds,
	}, nil
}
func (nlbHandler *ClouditNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "NETWORKLOADBALANCE", nlbIID.NameId, "ChangeHealthCheckerInfo()")
	start := call.Start()
	rawNLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}

	err = nlbHandler.checkValidChangeHealthCheckerInfo(rawNLB, healthChecker)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}

	equalCheck, err := nlbHandler.checkEqualHealthCheckerInfo(rawNLB, healthChecker)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	// not update
	if equalCheck {
		info, err := nlbHandler.setterNLB(rawNLB)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.HealthCheckerInfo{}, changeErr
		}
		LoggingInfo(hiscallInfo, start)
		return info.HealthChecker, nil
	}

	nlbUpdateInfo := loadbalancer.LoadBalancerHealthCheckerUpdateInfo{
		UnhealthyThreshold: healthChecker.Threshold,
		HealthyThreshold:   healthChecker.Threshold,
		IntervalTime:       healthChecker.Interval,
		ResponseTime:       healthChecker.Timeout,
	}

	_, err = nlbHandler.updateHealthCheckerPolicy(rawNLB.Id, nlbUpdateInfo)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	updatedNLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	info, err := nlbHandler.setterNLB(updatedNLB)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}

	LoggingInfo(hiscallInfo, start)
	return info.HealthChecker, err
}

func (nlbHandler *ClouditNLBHandler) checkValidChangeVMGroupInfo(rawNLB loadbalancer.LoadBalancerInfo, vmGroup irs.VMGroupInfo) error {
	currentVMGroup, err := nlbHandler.getVMGroup(rawNLB)
	if err != nil {
		return err
	}
	if !(vmGroup.Protocol == "" || strings.EqualFold(strings.ToUpper(vmGroup.Protocol), strings.ToUpper(currentVMGroup.Protocol))) {
		return errors.New(fmt.Sprintf("CLOUDIT_CANNOT_CHANGE_VMGROUPINFO_PROTOCOL"))
	}
	vmGroupPortInt, err := strconv.Atoi(vmGroup.Port)
	if err != nil {
		return errors.New(fmt.Sprintf("invalid vmGroup Port"))
	}
	if vmGroupPortInt < 1 || vmGroupPortInt > 65535 {
		return errors.New(fmt.Sprintf("cloudit vmGroup Port provides an port of between 1 and 65535"))
	}
	return nil
}
func (nlbHandler *ClouditNLBHandler) checkEqualVMGroupInfo(rawNLB loadbalancer.LoadBalancerInfo, vmGroup irs.VMGroupInfo) (bool, error) {
	currentVMGroup, err := nlbHandler.getVMGroup(rawNLB)
	if err != nil {
		return false, err
	}
	if !strings.EqualFold(strings.ToUpper(currentVMGroup.Port), strings.ToUpper(vmGroup.Port)) {
		return false, nil
	}
	return true, nil
}
func (nlbHandler *ClouditNLBHandler) checkValidChangeHealthCheckerInfo(rawNLB loadbalancer.LoadBalancerInfo, healthChecker irs.HealthCheckerInfo) error {
	currentHealthCheckerInfo, err := nlbHandler.getHealthCheckerInfo(rawNLB)
	if err != nil {
		return err
	}
	if !(healthChecker.Protocol == "" || strings.EqualFold(strings.ToUpper(healthChecker.Protocol), strings.ToUpper(currentHealthCheckerInfo.Protocol))) {
		return errors.New(fmt.Sprintf("CLOUDIT_CANNOT_CHANGE_HEALTHCHECKKERINFO_PROTOCOL"))
	}
	if !(healthChecker.Port == "" || strings.EqualFold(healthChecker.Port, currentHealthCheckerInfo.Port)) {
		return errors.New(fmt.Sprintf("CLOUDIT_CANNOT_CHANGE_HEALTHCHECKKERINFO_PORT"))
	}
	_, err = healthCheckPolicyValidation(healthChecker)
	if err != nil {
		return err
	}
	return nil
}
func (nlbHandler *ClouditNLBHandler) checkEqualHealthCheckerInfo(rawNLB loadbalancer.LoadBalancerInfo, healthChecker irs.HealthCheckerInfo) (bool, error) {
	currentHealthCheckerInfo, err := nlbHandler.getHealthCheckerInfo(rawNLB)
	if err != nil {
		return false, err
	}
	currentHealthCheckerInfo.CspID = ""
	currentHealthCheckerInfo.KeyValueList = nil

	healthChecker.Protocol = strings.ToUpper(healthChecker.Protocol)
	healthChecker.CspID = ""
	healthChecker.KeyValueList = nil

	// Protocol
	if healthChecker.Protocol == "" {
		currentHealthCheckerInfo.Protocol = ""
	}

	if healthChecker.Port == "" {
		currentHealthCheckerInfo.Port = ""
	}

	if reflect.DeepEqual(healthChecker, currentHealthCheckerInfo) {
		return true, nil
	}
	return false, nil
}
func (nlbHandler *ClouditNLBHandler) waitNLBCompleted(nlbId string) (loadbalancer.LoadBalancerInfo, error) {
	nlbHandler.Client.TokenID = nlbHandler.CredentialInfo.AuthToken
	authHeader := nlbHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		nlb, err := loadbalancer.GetSimple(nlbHandler.Client, &requestOpts, nlbId)
		if err == nil && nlb.State == "COMPLETED" {
			members, err := loadbalancer.MemberList(nlbHandler.Client, &requestOpts, nlbId)
			if err != nil {
				return loadbalancer.LoadBalancerInfo{}, err
			}
			nlb.Members = *members
			return *nlb, nil
		}
		time.Sleep(1 * time.Second)
		curRetryCnt++
		if curRetryCnt > maxRetryCnt {
			return loadbalancer.LoadBalancerInfo{}, errors.New(fmt.Sprintf("Failed to Create NLB err = exceeded maximum retry count %d", maxRetryCnt))
		}
	}
}
func (nlbHandler *ClouditNLBHandler) setterNLB(rawNLB loadbalancer.LoadBalancerInfo) (irs.NLBInfo, error) {
	nlbInfo := irs.NLBInfo{
		IId: irs.IID{
			NameId:   rawNLB.Name,
			SystemId: rawNLB.Id,
		},
		Scope: string(NLBRegionType),
		Type:  string(NLBPublicType),
	}

	VPCHandler := ClouditVPCHandler{
		Client:         nlbHandler.Client,
		CredentialInfo: nlbHandler.CredentialInfo,
	}
	defaultVPC, err := VPCHandler.GetDefaultVPC()
	if err == nil {
		nlbInfo.VpcIID = defaultVPC.IId
	}

	createdTime, err := time.Parse("2006-01-02 15:04:05", rawNLB.CreatedAt)
	if err != nil {
		return irs.NLBInfo{}, err
	}
	nlbInfo.CreatedTime = createdTime

	if strings.EqualFold(rawNLB.Type, string(loadbalancer.LoadBalancerInternalType)) {
		nlbInfo.Type = string(NLBInternalType)
	}
	vmGroup, err := nlbHandler.getVMGroup(rawNLB)
	if err != nil {
		return irs.NLBInfo{}, err
	}
	nlbInfo.VMGroup = vmGroup

	listenInfo, err := nlbHandler.getListenerInfo(rawNLB)
	if err != nil {
		return irs.NLBInfo{}, err
	}
	nlbInfo.Listener = listenInfo

	healthCheckerInfo, err := nlbHandler.getHealthCheckerInfo(rawNLB)
	if err != nil {
		return irs.NLBInfo{}, err
	}
	nlbInfo.HealthChecker = healthCheckerInfo

	return nlbInfo, nil
}
func (nlbHandler *ClouditNLBHandler) getHealthCheckerInfo(rawNLB loadbalancer.LoadBalancerInfo) (irs.HealthCheckerInfo, error) {
	port, err := nlbHandler.getVMGroupPort(rawNLB.Id)
	if err != nil {
		return irs.HealthCheckerInfo{}, err
	}
	return irs.HealthCheckerInfo{
		Protocol:  strings.ToUpper(rawNLB.MonitorType),
		Timeout:   rawNLB.ResponseTime,
		Interval:  rawNLB.IntervalTime,
		Threshold: rawNLB.UnhealthyThreshold,
		Port:      port,
	}, nil
}
func (nlbHandler *ClouditNLBHandler) getListenerInfo(rawNLB loadbalancer.LoadBalancerInfo) (irs.ListenerInfo, error) {
	return irs.ListenerInfo{
		Protocol: strings.ToUpper(rawNLB.Protocol),
		Port:     strconv.Itoa(rawNLB.Port),
		IP:       rawNLB.Ip,
	}, nil
}
func (nlbHandler *ClouditNLBHandler) getVMGroupPort(nlbId string) (string, error) {
	nlbHandler.Client.TokenID = nlbHandler.CredentialInfo.AuthToken
	authHeader := nlbHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	rawNLB, err := loadbalancer.Get(nlbHandler.Client, &requestOpts, nlbId)
	if err != nil {
		return "", errors.New(fmt.Sprintf("unable to get port information for vmGroup err = %s", err.Error()))
	}
	if len(rawNLB.Members) == 0 {
		if rawNLB.Description != "" {
			return getVMGroupPortByDescription(rawNLB.Description)
		}
		return "", errors.New("unable to get port information for vmGroup")
	}
	firstMember := rawNLB.Members[0]
	if firstMember.MemberPort == "" {
		return "", errors.New("unable to get port information for vmGroup")
	}
	// desPort, _ := getVMGroupPortByDescription(rawNLB.Description)
	//if desPort == "" || !strings.EqualFold(desPort, firstMember.MemberPort) {
	//	nlbHandler.setDescriptionNLB(rawNLB.Id,fmt.Sprintf("vmgroupport:%s",firstMember.MemberPort))
	//}
	return firstMember.MemberPort, nil
}
func (nlbHandler *ClouditNLBHandler) getVMGroup(rawNLB loadbalancer.LoadBalancerInfo) (irs.VMGroupInfo, error) {
	vmGroup := irs.VMGroupInfo{}
	vmGroup.Protocol = strings.ToUpper(rawNLB.Protocol)

	nlbHandler.Client.TokenID = nlbHandler.CredentialInfo.AuthToken
	authHeader := nlbHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	serverList, err := server.List(nlbHandler.Client, &requestOpts)
	if err != nil {
		return irs.VMGroupInfo{}, err
	}
	vmIIds := make([]irs.IID, len(rawNLB.Members))
	for i, member := range rawNLB.Members {
		for _, vm := range *serverList {
			if strings.EqualFold(vm.Name, member.ServerName) {
				vmIIds[i] = irs.IID{NameId: vm.Name, SystemId: vm.ID}
			}
		}
	}
	vmGroup.VMs = &vmIIds
	port, err := nlbHandler.getVMGroupPort(rawNLB.Id)
	if err != nil {
		return irs.VMGroupInfo{}, err
	}
	vmGroup.Port = port
	return vmGroup, nil
}
func (nlbHandler *ClouditNLBHandler) getRawNLBList() ([]loadbalancer.LoadBalancerInfo, error) {
	nlbHandler.Client.TokenID = nlbHandler.CredentialInfo.AuthToken
	authHeader := nlbHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	nlbList, err := loadbalancer.List(nlbHandler.Client, &requestOpts)
	if err != nil {
		return nil, err
	}
	return *nlbList, nil
}
func (nlbHandler *ClouditNLBHandler) getRawNLB(IId irs.IID) (loadbalancer.LoadBalancerInfo, error) {
	if IId.NameId == "" && IId.SystemId == "" {
		return loadbalancer.LoadBalancerInfo{}, errors.New("invalid IID")
	}
	if IId.NameId != "" {
		nlbHandler.Client.TokenID = nlbHandler.CredentialInfo.AuthToken
		authHeader := nlbHandler.Client.AuthenticatedHeaders()

		requestOpts := client.RequestOpts{
			MoreHeaders: authHeader,
		}
		nlb, err := loadbalancer.GetByName(nlbHandler.Client, &requestOpts, IId.NameId)
		if err != nil {
			return loadbalancer.LoadBalancerInfo{}, err
		}
		return *nlb, err
	}
	nlbHandler.Client.TokenID = nlbHandler.CredentialInfo.AuthToken
	authHeader := nlbHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	nlb, err := loadbalancer.Get(nlbHandler.Client, &requestOpts, IId.SystemId)
	if err != nil {
		return loadbalancer.LoadBalancerInfo{}, err
	}
	return *nlb, err
}
func (nlbHandler *ClouditNLBHandler) creatablePublicIP() (adaptiveip.IPInfo, error) {
	nlbHandler.Client.TokenID = nlbHandler.CredentialInfo.AuthToken
	authHeader := nlbHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	if availableIPList, err := adaptiveip.ListAvailableIP(nlbHandler.Client, &requestOpts); err != nil {
		return adaptiveip.IPInfo{}, err
	} else {
		if len(*availableIPList) == 0 {
			return adaptiveip.IPInfo{}, errors.New(fmt.Sprintf("There is no PublicIPs to allocate"))
		} else {
			return (*availableIPList)[0], nil
		}
	}
}
func (nlbHandler *ClouditNLBHandler) getRawVmList() ([]server.ServerInfo, error) {
	nlbHandler.Client.TokenID = nlbHandler.CredentialInfo.AuthToken
	authHeader := nlbHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	vmList, err := server.List(nlbHandler.Client, &requestOpts)
	if err != nil {
		return nil, err
	}
	return *vmList, nil
}
func (nlbHandler *ClouditNLBHandler) createNLB(nlbReqInfo loadbalancer.LoadBalancerReqInfo) (loadbalancer.LoadBalancerInfo, error) {
	nlbHandler.Client.TokenID = nlbHandler.CredentialInfo.AuthToken
	authHeader := nlbHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
		JSONBody:    nlbReqInfo,
	}
	rawnlb, err := loadbalancer.Create(nlbHandler.Client, &requestOpts)
	if err != nil {
		return loadbalancer.LoadBalancerInfo{}, err
	}
	return *rawnlb, nil
}
func (nlbHandler *ClouditNLBHandler) deleteNLB(nlbIId irs.IID) (bool, error) {
	if nlbIId.SystemId == "" && nlbIId.NameId == "" {
		return false, errors.New("invalid IID")
	}
	nlbList, err := nlbHandler.getRawNLBList()
	if err != nil {
		return false, err
	}
	deleteNLBName := ""
	deleteNLBId := ""
	exitCheck := false
	for _, rawNLB := range nlbList {
		if strings.EqualFold(rawNLB.Id, nlbIId.SystemId) || strings.EqualFold(rawNLB.Name, nlbIId.NameId) {
			deleteNLBId = rawNLB.Id
			deleteNLBName = rawNLB.Name
			exitCheck = true
		}
	}
	if !exitCheck {
		return false, errors.New(fmt.Sprintf("not exist nlb : %s", deleteNLBName))
	}
	nlbHandler.Client.TokenID = nlbHandler.CredentialInfo.AuthToken
	authHeader := nlbHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	result, err := loadbalancer.Delete(nlbHandler.Client, &requestOpts, deleteNLBId)
	if err != nil {
		return false, err
	}
	return result, nil
}
func (nlbHandler *ClouditNLBHandler) getRawVmByName(vmName string) (server.ServerInfo, error) {
	if vmName == "" {
		return server.ServerInfo{}, errors.New("invalid IID")
	}
	nlbHandler.Client.TokenID = nlbHandler.CredentialInfo.AuthToken
	authHeader := nlbHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	vmList, err := server.List(nlbHandler.Client, &requestOpts)
	if err != nil {
		return server.ServerInfo{}, err
	}
	for _, vm := range *vmList {
		if strings.EqualFold(vm.Name, vmName) {
			return vm, nil
		}
	}
	return server.ServerInfo{}, errors.New(fmt.Sprintf("not found vm : %s", vmName))
}
func getVMGroupPortByDescription(des string) (string, error) {
	if reg, err := regexp.Compile("vmgroupport:[0-9]+"); err == nil {
		if stArr := reg.FindAllString(des, -1); len(stArr) > 0 {
			if portReg2, err := regexp.Compile("[0-9]+"); err == nil {
				portStrArr := portReg2.FindAllString(stArr[0], -1)
				if len(portStrArr) > 0 {
					return portStrArr[0], nil
				}
			}
		}
	}
	return "", errors.New("unable to get port for vmGroup from Description")
}
func (nlbHandler *ClouditNLBHandler) setDescriptionNLB(nlbId string, des string) (loadbalancer.LoadBalancerInfo, error) {

	nlbHandler.Client.TokenID = nlbHandler.CredentialInfo.AuthToken
	authHeader := nlbHandler.Client.AuthenticatedHeaders()

	updateRequestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
		JSONBody: loadbalancer.LoadBalancerUpdateInfo{
			Description: des,
		},
	}
	getNLBRequestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	expectDes := des
	_, err := loadbalancer.Update(nlbHandler.Client, &updateRequestOpts, nlbId)
	if err != nil {
		return loadbalancer.LoadBalancerInfo{}, err
	}
	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		rawNLB, err := loadbalancer.Get(nlbHandler.Client, &getNLBRequestOpts, nlbId)
		if err == nil &&
			rawNLB.Description == expectDes {
			return *rawNLB, nil
		}
		time.Sleep(1 * time.Second)
		curRetryCnt++
		if curRetryCnt > maxRetryCnt {
			return loadbalancer.LoadBalancerInfo{}, errors.New(fmt.Sprintf("Failed to Clean NLB Member err = exceeded maximum retry count %d", maxRetryCnt))
		}
	}
}
func (nlbHandler *ClouditNLBHandler) deleteMembers(nlbId string, memberIds []string) (bool, error) {
	nlbHandler.Client.TokenID = nlbHandler.CredentialInfo.AuthToken
	authHeader := nlbHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	prevRawNLB, err := loadbalancer.Get(nlbHandler.Client, &requestOpts, nlbId)
	if err != nil {
		return false, err
	}
	expectMemberNum := len(prevRawNLB.Members) - len(memberIds)
	if expectMemberNum < 0 {
		return false, errors.New("cannot delete more members than the current member")
	}
	for _, id := range memberIds {
		err := loadbalancer.DeleteMember(nlbHandler.Client, &requestOpts, nlbId, id)
		if err != nil {
			return false, err
		}
	}
	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		list, err := loadbalancer.MemberList(nlbHandler.Client, &requestOpts, nlbId)
		if err == nil && len(*list) == expectMemberNum {
			return true, nil
		}
		time.Sleep(1 * time.Second)
		curRetryCnt++
		if curRetryCnt > maxRetryCnt {
			return false, errors.New(fmt.Sprintf("Failed to Clean NLB Member err = exceeded maximum retry count %d", maxRetryCnt))
		}
	}
}
func (nlbHandler *ClouditNLBHandler) addMembers(nlbId string, members []loadbalancer.AddMemberInfo) (loadbalancer.LoadBalancerInfo, error) {
	nlbHandler.Client.TokenID = nlbHandler.CredentialInfo.AuthToken
	authHeader := nlbHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	prevRawNLB, err := loadbalancer.Get(nlbHandler.Client, &requestOpts, nlbId)
	if err != nil {
		return loadbalancer.LoadBalancerInfo{}, err
	}

	expectMemberNum := len(prevRawNLB.Members) + len(members)

	for _, mem := range members {
		addOpts := requestOpts
		addOpts.JSONBody = mem
		_, err := loadbalancer.AddMember(nlbHandler.Client, &addOpts, nlbId)
		if err != nil {
			return loadbalancer.LoadBalancerInfo{}, err
		}
	}
	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		list, err := loadbalancer.MemberList(nlbHandler.Client, &requestOpts, nlbId)
		if err == nil && len(*list) == expectMemberNum {
			rawNLB, err := loadbalancer.Get(nlbHandler.Client, &requestOpts, nlbId)
			if err != nil {
				return loadbalancer.LoadBalancerInfo{}, err
			}
			return *rawNLB, nil
		}
		time.Sleep(1 * time.Second)
		curRetryCnt++
		if curRetryCnt > maxRetryCnt {
			return loadbalancer.LoadBalancerInfo{}, errors.New(fmt.Sprintf("Failed to Clean NLB Member err = exceeded maximum retry count %d", maxRetryCnt))
		}
	}
}
func (nlbHandler *ClouditNLBHandler) updateHealthCheckerPolicy(nlbId string, nlbUpdateInfo loadbalancer.LoadBalancerHealthCheckerUpdateInfo) (bool, error) {
	nlbHandler.Client.TokenID = nlbHandler.CredentialInfo.AuthToken
	authHeader := nlbHandler.Client.AuthenticatedHeaders()

	updateRequestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
		JSONBody:    nlbUpdateInfo,
	}
	getNLBRequestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	expectUnhealthyThreshold := nlbUpdateInfo.UnhealthyThreshold
	expectHealthyThreshold := nlbUpdateInfo.HealthyThreshold
	expectIntervalTime := nlbUpdateInfo.IntervalTime
	expectResponseTime := nlbUpdateInfo.ResponseTime

	_, err := loadbalancer.UpdateHealthChecker(nlbHandler.Client, &updateRequestOpts, nlbId)
	if err != nil {
		return false, err
	}
	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		rawNLB, err := loadbalancer.Get(nlbHandler.Client, &getNLBRequestOpts, nlbId)
		if err == nil &&
			rawNLB.UnhealthyThreshold == expectUnhealthyThreshold &&
			rawNLB.HealthyThreshold == expectHealthyThreshold &&
			rawNLB.IntervalTime == expectIntervalTime &&
			rawNLB.ResponseTime == expectResponseTime {
			return true, nil
		}
		time.Sleep(1 * time.Second)
		curRetryCnt++
		if curRetryCnt > maxRetryCnt {
			return false, errors.New(fmt.Sprintf("Failed to Clean NLB Member err = exceeded maximum retry count %d", maxRetryCnt))
		}
	}
}
func healthCheckPolicyValidation(info irs.HealthCheckerInfo) (bool, error) {
	if !(info.Threshold >= 2 && info.Threshold <= 10) {
		return false, errors.New("invalid HealthCheckerInfo Threshold, err : Threshold must be between 2 and 10")
	}
	if !(info.Timeout >= 2 && info.Timeout <= 60) {
		return false, errors.New("invalid HealthCheckerInfo Timeout, err : Timeout must be between 2 and 60")
	}
	if !(info.Interval >= 5 && info.Interval <= 300) {
		return false, errors.New("invalid HealthCheckerInfo Interval, err : Interval must be between 5 and 300")
	}
	if info.Interval < info.Timeout {
		return false, errors.New("invalid HealthCheckerInfo Interval, Timeout err : Interval must be equal to or greater than Timeout")
	}
	return true, nil
}
