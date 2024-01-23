// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI Team, 2024.01.

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
	
	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"
)

type KtCloudNLBHandler struct {
	RegionInfo     	idrv.RegionInfo
	Client       	*ktsdk.KtCloudClient
	NLBClient  		*ktsdk.KtCloudClient
}

const (
	DefaultNLBOption 		string  = "roundrobin" // NLBOption : roundrobin / leastconnection / leastresponse / sourceiphash / 
	DefaultHealthCheckURL	string  = "abc.kt.com"
)

func (nlbHandler *KtCloudNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (irs.NLBInfo, error) {
	cblogger.Info("KT Cloud Driver: called CreateNLB()")
	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbReqInfo.IId.NameId, "CreateNLB()")

	if strings.EqualFold(nlbReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid NLB NameId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	if strings.EqualFold(nlbReqInfo.VpcIID.NameId, "") {
		newErr := fmt.Errorf("Invalid VPC NameId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	if !strings.EqualFold(nlbReqInfo.Listener.Protocol, nlbReqInfo.VMGroup.Protocol) {
		newErr := fmt.Errorf("Listener Protocol and VMGroup Protocol should be the Same for KT Cloud NLB!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	if !strings.EqualFold(nlbReqInfo.Listener.Port, nlbReqInfo.VMGroup.Port) {
		newErr := fmt.Errorf("Listener Port and VMGroup Prot should be the Same for this KT Cloud connection driver!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	lbReq := ktsdk.CreateNLBReqInfo{
		Name:           	nlbReqInfo.IId.NameId,  			// Required
		ZoneId:         	nlbHandler.RegionInfo.Zone, 		// Required
		NLBOption: 			DefaultNLBOption,					// Required
		ServiceIP: 			"",									// Required. KT Cloud Virtual IP. $$$ In case of an empty value(""), it is newly created.
		ServicePort: 		nlbReqInfo.Listener.Port,			// Required
		ServiceType: 		nlbReqInfo.Listener.Protocol,		// Required
		HealthCheckType: 	nlbReqInfo.HealthChecker.Protocol,  // Required
		HealthCheckURL: 	DefaultHealthCheckURL,				// URL when the HealthCheckType (above) is 'http' or 'https'.
	}
	start := call.Start()
	nlbResp, err := nlbHandler.NLBClient.CreateNLB(lbReq) // Not 'Client'
	if (err != nil) ||(nlbResp.Createnlbresponse.ErrorText != "") { // Note!! : Apply 'ErrorText'
		newErr := fmt.Errorf("Failed to Create New NLB. [%v]. [%v]", err, nlbResp.Createnlbresponse.ErrorText)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)
	cblogger.Infof("# New NLBId : %s", nlbResp.Createnlbresponse.NLBId)

	cblogger.Info("\n### New NLB is Creating Now!!")
	time.Sleep(time.Second * 7)

	newNlbIID := irs.IID{SystemId: nlbResp.Createnlbresponse.NLBId}
	nlbInfo, err := nlbHandler.GetNLB(newNlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the New NLB Info. [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)	
		return irs.NLBInfo{}, newErr
	}
	return nlbInfo, nil
}

func (nlbHandler *KtCloudNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {
	cblogger.Info("KT Cloud Driver: called ListNLB()")
	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", "ListNLB()", "ListNLB()")

	lbReq := ktsdk.ListNLBsReqInfo{
		ZoneId: nlbHandler.RegionInfo.Zone,
	}
	start := call.Start()
	nlbResp, err := nlbHandler.NLBClient.ListNLBs(lbReq) // Not 'Client'
	if err != nil {
		cblogger.Error("Failed to Get KT Cloud NLB list : [%v]", err)
		return []*irs.NLBInfo{}, err
	}
	LoggingInfo(callLogInfo, start)
	// spew.Dump(result)

	time.Sleep(time.Second * 1) // Before 'return'
	// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"

	if len(nlbResp.Listnlbsresponse.NLB) < 1 {
		cblogger.Info("# KT Cloud NLB does Not Exist!!")
		return []*irs.NLBInfo{}, nil // Not Return Error
	}
	// cblogger.Info("\n\n### nlbResp.Listnlbsresponse : ")
	// spew.Dump(nlbResp.Listnlbsresponse)

	var nlbInfoList []*irs.NLBInfo
    for _, nlb := range nlbResp.Listnlbsresponse.NLB {
		nlbInfo, err := nlbHandler.MappingNlbInfo(&nlb)
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

func (nlbHandler *KtCloudNLBHandler) GetNLB(nlbIID irs.IID) (irs.NLBInfo, error) {
	cblogger.Info("KT Cloud Driver: called GetNLB()")
	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "GetNLB()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return irs.NLBInfo{}, newErr
	}
	
	ktNLB, err := nlbHandler.GetKTCloudNLB(nlbIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	var nlbInfo irs.NLBInfo
	nlbInfo, err = nlbHandler.MappingNlbInfo(ktNLB)
	if err != nil {
		newErr := fmt.Errorf("Failed to Map NLB Info with the NLB : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	return nlbInfo, nil
}

func (nlbHandler *KtCloudNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called DeleteNLB()")
	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "DeleteNLB()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	// Note!!) Should check 'EstablishedConn'(Client Connections) value before deletion?
	start := call.Start()
	nlbResp, err := nlbHandler.NLBClient.DeleteNLB(nlbIID.SystemId) // Not 'Client'
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the NLB!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, start)
	// cblogger.Info("\n\n### delResult : ")
	// spew.Dump(nlbResp)

	if !nlbResp.Deletenlbresponse.Success {
		newErr := fmt.Errorf("Failed to Delete the NLB!! : [%s] ", nlbResp.Deletenlbresponse.Displaytext)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	} else {
		cblogger.Infof("# Result : %s", nlbResp.Deletenlbresponse.Displaytext)		
	}

	return true, nil
}

func (nlbHandler *KtCloudNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {
	cblogger.Info("KT Cloud Driver: called ChangeListener()")

	return irs.ListenerInfo{}, fmt.Errorf("KT Cloud does not support ChangeListener() yet!!")
}

func (nlbHandler *KtCloudNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {
	cblogger.Info("KT Cloud Driver: called ChangeVMGroupInfo()")

	return irs.VMGroupInfo{}, fmt.Errorf("KT Cloud does not support ChangeVMGroupInfo() yet!!")
}

func (nlbHandler *KtCloudNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	cblogger.Info("KT Cloud Driver: called AddVMs()")
	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "AddVMs()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	if len(*vmIIDs) < 1 {
		newErr := fmt.Errorf("Failded to Find any requested VM to Add to the NLB!!")
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
	cblogger.Infof("# NLB Service Port : %s", nlbInfo.Listener.Port)

	vmHandler := KtCloudVMHandler{
		RegionInfo: nlbHandler.RegionInfo,
		Client:   	nlbHandler.Client,
	}
	var vmIdList []string
	if len(*vmIIDs) > 0 {
		for _, vmIID := range *vmIIDs {
			vmId, err := vmHandler.GetVmIdWithName(vmIID.NameId)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the VM ID with the VM Name : [%v]", err)
				cblogger.Error(newErr.Error())
				return irs.VMGroupInfo{}, newErr
			}
			vmIdList = append(vmIdList, vmId)

			time.Sleep(time.Second * 1)
			// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"
		}
	} else {
		newErr := fmt.Errorf("Failded to Find any VM NameId to Add to the NLB!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	for _, vmId := range vmIdList {
		publicIP, err := vmHandler.GetIPFromPortForwardingRules(vmId)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get Public IP Info : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.VMGroupInfo{}, newErr
		}
		cblogger.Infof("# VM Public IP : %s", publicIP)

		time.Sleep(time.Second * 1)
		// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"

		addVmReq := ktsdk.AddNLBVMReqInfo{
			NLBId:         	nlbIID.SystemId,		// Required
			VMId:			vmId,					// Required
			IpAddress: 		publicIP,				// Required. 'Public IP' of the VM
			PublicPort: 	nlbInfo.Listener.Port,	// Required. The same as the Listener Port (Service Port)
		}
		start := call.Start()
		addVMResp, err := nlbHandler.NLBClient.AddNLBVM(addVmReq)
		if err != nil {
			newErr := fmt.Errorf("Failed to Attach the Disk Volume. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMGroupInfo{}, newErr
		}
		LoggingInfo(callLogInfo, start)
		// cblogger.Info("\n\n### AddVMResp : ")
		// spew.Dump(addVMResp)
		// cblogger.Info("\n")
		cblogger.Infof("# ServiceId of the VM : %s : %d ", vmId, addVMResp.Addnlbvmresponse.ServiceId)

		time.Sleep(time.Second * 1)
		// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"
	}
	
	newVMGroupNlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	return newVMGroupNlbInfo.VMGroup, nil
}

func (nlbHandler *KtCloudNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called RemoveVMs()")
	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", "RemoveVMs()", "RemoveVMs()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	if len(*vmIIDs) < 1 {
		newErr := fmt.Errorf("Failed to Find any VM to Remove!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	vmHandler := KtCloudVMHandler{
		RegionInfo: nlbHandler.RegionInfo,
		Client:   	nlbHandler.Client,
	}
	var vmIdList []string
	if len(*vmIIDs) > 0 {
		for _, vmIID := range *vmIIDs {
			vmId, err := vmHandler.GetVmIdWithName(vmIID.NameId)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the VM ID with the VM Name : [%v]", err)
				cblogger.Error(newErr.Error())
				return false, newErr
			}
			vmIdList = append(vmIdList, vmId)

			time.Sleep(time.Second * 1)
			// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"
		}
	} else {
		newErr := fmt.Errorf("Failded to Find any VM NameId to Add to the NLB!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	for _, vmId := range vmIdList {
		serviceId, err := nlbHandler.GetServiceIdWithVMId(nlbIID.SystemId, vmId)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get Service ID of the NLB VM!! [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return false, newErr
		}
		cblogger.Infof("# ServiceId : %d of VM : %s", serviceId, vmId)

		start := call.Start()
		removeResp, err := nlbHandler.NLBClient.RemoveNLBVM(strconv.Itoa(serviceId))
		if err != nil {
			newErr := fmt.Errorf("Failed to Attach the Disk Volume. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return false, newErr
		}
		LoggingInfo(callLogInfo, start)
		// cblogger.Info("\n\n### RemoveResp : ")
		// spew.Dump(removeResp)
		// cblogger.Info("\n")		

		time.Sleep(time.Second * 1) // Before 'return'
		// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"

		if !removeResp.Removenlbvmresponse.Success {
			newErr := fmt.Errorf("Failed to Remove the NLB VM!! : [%s] ", removeResp.Removenlbvmresponse.Displaytext)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return false, newErr
		} else {
			cblogger.Infof("# Result : %s", removeResp.Removenlbvmresponse.Displaytext)		
		}
	}

	return true, nil
}

func (nlbHandler *KtCloudNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	cblogger.Info("KT Cloud Driver: called GetVMGroupHealthInfo()")
	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "GetVMGroupHealthInfo()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthInfo{}, newErr
	}

	// Get KT Cloud NLB VM list
	nlbResp, err := nlbHandler.NLBClient.ListNLBVMs(nlbIID.SystemId) // Not 'Client'
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB VM list : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.HealthInfo{}, newErr
	}
	
	time.Sleep(time.Second * 1) // Before 'return'
	// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"

	if len(nlbResp.Listnlbvmsresponse.NLBVM) < 1 {
		newErr := fmt.Errorf("Failed to Find Any VM from NLB VM list!!")
		cblogger.Error(newErr.Error())
		return irs.HealthInfo{}, newErr
	}
	// cblogger.Info("\n\n### nlbResp.Listnlbsresponse : ")
	// spew.Dump(nlbResp.Listnlbvmsresponse)

	vmHandler := KtCloudVMHandler{
		RegionInfo: nlbHandler.RegionInfo,
		Client:   	nlbHandler.Client,
	}

	var allVMs []irs.IID
	var healthVMs []irs.IID
	var unHealthVMs []irs.IID

	for _, vm := range nlbResp.Listnlbvmsresponse.NLBVM {
		vmName, err := vmHandler.GetVmNameWithId(vm.VMId)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the VM Name with the VM ID : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.HealthInfo{}, newErr
		}
		
		allVMs = append(allVMs, irs.IID{NameId: vmName, SystemId: vm.VMId})

		if strings.EqualFold(vm.State, "UP") {
			cblogger.Infof("\n### [%s] is Healthy VM.", vmName)
			healthVMs = append(healthVMs, irs.IID{NameId: vmName, SystemId: vm.VMId})
		} else {
			cblogger.Infof("\n### [%s] is Unhealthy VM.", vmName)
			unHealthVMs = append(unHealthVMs, irs.IID{NameId: vmName, SystemId: vm.VMId})  // In case of "DOWN"
		}

		time.Sleep(time.Second * 1)
		// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"
	}

	vmGroupHealthInfo := irs.HealthInfo{
		AllVMs:       &allVMs,
		HealthyVMs:   &healthVMs,
		UnHealthyVMs: &unHealthVMs,
	}
	return vmGroupHealthInfo, nil
}

func (nlbHandler *KtCloudNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {
	cblogger.Info("KT Cloud Driver: called ChangeHealthCheckerInfo()")

	return irs.HealthCheckerInfo{}, fmt.Errorf("KT Cloud does not support ChangeHealthCheckerInfo() yet!!")
}

func (nlbHandler *KtCloudNLBHandler) GetListenerInfo(nlb *ktsdk.NLB) (irs.ListenerInfo, error) {
	cblogger.Info("KT Cloud Driver: called GetListenerInfo()")
	nlbId := strconv.Itoa(nlb.NLBId)
	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbId, "GetListenerInfo()")

	if strings.EqualFold(nlbId, "") {
		newErr := fmt.Errorf("Invalid Load-Balancer ID. The LB does Not Exit!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.ListenerInfo{}, newErr
	}
	
	listenerInfo := irs.ListenerInfo{
		Protocol: 	nlb.ServiceType,
		IP: 		nlb.ServiceIP,
		Port: 		nlb.ServicePort,
		DNSName:	"N/A",
		CspID: 		"N/A",
	}
	listenerKVList := []irs.KeyValue{
		// {Key: "NLB_DomainName", Value: *nlb.DomainName},
	}
	listenerInfo.KeyValueList = listenerKVList
	return listenerInfo, nil
}

func (nlbHandler *KtCloudNLBHandler) GetHealthCheckerInfo(nlb *ktsdk.NLB) (irs.HealthCheckerInfo, error) {
	cblogger.Info("KT Cloud Driver: called GetHealthCheckerInfo()")
	nlbId := strconv.Itoa(nlb.NLBId)
	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbId, "GetHealthCheckerInfo()")

	if strings.EqualFold(nlbId, "") {
		newErr := fmt.Errorf("Invalid Load-Balancer ID. The LB does Not Exit!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthCheckerInfo{}, newErr
	}

	healthCheckerInfo := irs.HealthCheckerInfo{
		Protocol: 	nlb.HealthCheckType,
		Port:     	nlb.ServicePort,
		CspID: 		"N/A",
	}
	return healthCheckerInfo, nil
}

func (nlbHandler *KtCloudNLBHandler) GetVMGroupInfo(nlbId string) (irs.VMGroupInfo, error) {
	cblogger.Info("KT Cloud Driver: called GetVMGroupInfo()")
	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbId, "GetVMGroupInfo()")

	if strings.EqualFold(nlbId, "") {
		newErr := fmt.Errorf("Invalid NLB ID")
		cblogger.Error(newErr.Error())
		return irs.VMGroupInfo{}, newErr
	}
	
	ktNLB, err := nlbHandler.GetKTCloudNLB(nlbId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	serviceProtocol := ktNLB.ServiceType

	// Get KT Cloud NLB VM list
	start := call.Start()
	nlbResp, err := nlbHandler.NLBClient.ListNLBVMs(nlbId) // Not 'Client'
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB VM list : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMGroupInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)
	// spew.Dump(nlbResp)

	time.Sleep(time.Second * 1) // Before 'return'
	// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"

	if len(nlbResp.Listnlbvmsresponse.NLBVM) < 1 {
		cblogger.Info("# NLB VM does Not Exist Yet!!")
		return irs.VMGroupInfo{}, nil // Not Return Error
	}
	// cblogger.Info("\n\n### nlbResp.Listnlbsresponse : ")
	// spew.Dump(nlbResp.Listnlbvmsresponse)
	
	vmGroupInfo := irs.VMGroupInfo{
		Protocol: 	serviceProtocol, // Caution!!
		Port: 		nlbResp.Listnlbvmsresponse.NLBVM[0].PublicPort,
		CspID:    	"N/A",
	}

	vmHandler := KtCloudVMHandler{
		RegionInfo: nlbHandler.RegionInfo,
		Client:   	nlbHandler.Client,
	}

	vmIIds := []irs.IID{}
	keyValueList := []irs.KeyValue{}
	for _, vm := range nlbResp.Listnlbvmsresponse.NLBVM {
		vmName, err := vmHandler.GetVmNameWithId(vm.VMId)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the VM Name with the VM ID : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.VMGroupInfo{}, newErr
		}

		vmIIds = append(vmIIds, irs.IID{
			NameId:   vmName,
			SystemId: vm.VMId,
		})

		keyValueList = append(keyValueList, irs.KeyValue{
			Key: 	vmName + "_ServiceId",
			Value: 	strconv.Itoa(vm.ServiceId),				
		})

		time.Sleep(time.Second * 1)
		// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"
	}
	vmGroupInfo.VMs = &vmIIds
	vmGroupInfo.KeyValueList = keyValueList
	return vmGroupInfo, nil
}

func (nlbHandler *KtCloudNLBHandler) GetServiceIdWithVMId(nlbId string, vmId string) (int, error) {
	cblogger.Info("KT Cloud cloud driver: called GetServiceIdWithVMId()!")

	if strings.EqualFold(nlbId, "") {
		newErr := fmt.Errorf("Invalid NLB ID")
		cblogger.Error(newErr.Error())
		return 0, newErr
	}

	if strings.EqualFold(vmId, "") {
		newErr := fmt.Errorf("Invalid VM ID")
		cblogger.Error(newErr.Error())
		return 0, newErr
	}

	// Get KT Cloud NLB VM list
	nlbResp, err := nlbHandler.NLBClient.ListNLBVMs(nlbId) // Not 'Client'
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB VM list : [%v]", err)
		cblogger.Error(newErr.Error())
		return 0, newErr
	}

	time.Sleep(time.Second * 1) // Before 'return'
	// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"

	if len(nlbResp.Listnlbvmsresponse.NLBVM) < 1 {
		newErr := fmt.Errorf("Failed to Find Any VM from NLB VM list!!")
		cblogger.Error(newErr.Error())
		return 0, newErr
	}
	// cblogger.Info("\n\n### nlbResp.Listnlbsresponse : ")
	// spew.Dump(nlbResp.Listnlbvmsresponse)

	var serviceId int // Not 'string'
	for _, vm := range nlbResp.Listnlbvmsresponse.NLBVM {
		if strings.EqualFold(vm.VMId, vmId) {
			serviceId = vm.ServiceId
			break
		}
	}

	if serviceId == 0 {
		newErr := fmt.Errorf("Failed to Find the ServiceId with the NLB VMId %s", vmId)
		cblogger.Error(newErr.Error())
		return 0, newErr
	} else {
	return serviceId, nil
	}
}

func (nlbHandler *KtCloudNLBHandler) MappingNlbInfo(nlb *ktsdk.NLB) (irs.NLBInfo, error) {
	cblogger.Info("KT Cloud Driver: called MappingNlbInfo()")
	// cblogger.Info("\n\n### nlb : ")
	// spew.Dump(nlb)	

	nlbInfo := irs.NLBInfo{
		IId: irs.IID{
			NameId:   nlb.Name,
			SystemId: strconv.Itoa(nlb.NLBId),
		},
		VpcIID: irs.IID{
			NameId:   "N/A",
			SystemId: "N/A",
		},
		Type:         "PUBLIC",
		Scope:        "REGION",
	}

	keyValueList := []irs.KeyValue{
		{Key: "NLB_Method", Value: nlb.NLBOption},
		{Key: "NLB_State", Value: nlb.State},
		{Key: "NLB_ServiceIP", Value: nlb.ServiceIP},
		{Key: "NLB_ServicePort", Value: nlb.ServicePort},
		{Key: "ZoneName", Value: nlb.ZoneName},
	}
	nlbInfo.KeyValueList = keyValueList

	if !strings.EqualFold(nlb.ServiceIP, "") {
		listenerInfo, err := nlbHandler.GetListenerInfo(nlb)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Listener Info : [%v]", err.Error())
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		}
		nlbInfo.Listener = listenerInfo
	}

	if !strings.EqualFold(nlb.HealthCheckType, "") {
		healthCheckerInfo, err := nlbHandler.GetHealthCheckerInfo(nlb)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get HealthChecker Info. frome the NLB. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		}
		nlbInfo.HealthChecker = healthCheckerInfo
	}

	vmGroupInfo, err := nlbHandler.GetVMGroupInfo(strconv.Itoa(nlb.NLBId))
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VM Group Info with the NLB ID : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.NLBInfo{}, newErr
	}
	nlbInfo.VMGroup = vmGroupInfo
	return nlbInfo, nil
}

func (nlbHandler *KtCloudNLBHandler) GetKTCloudNLB(nlbId string) (*ktsdk.NLB, error) {
	cblogger.Info("KT Cloud Driver: called GetKTCloudNLB()")
	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbId, "GetKTCloudNLB()")

	if strings.EqualFold(nlbId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	lbReq := ktsdk.ListNLBsReqInfo{
		ZoneId: nlbHandler.RegionInfo.Zone,
		NLBId: 	nlbId,
	}
	start := call.Start()
	nlbResp, err := nlbHandler.NLBClient.ListNLBs(lbReq) // Not 'Client'
	if err != nil {
		cblogger.Error("Failed to Get KT Cloud NLB list : [%v]", err)
		return nil, err
	}
	LoggingInfo(callLogInfo, start)
	// cblogger.Info("\n# nlbResp : ")
	// spew.Dump(nlbResp)

	time.Sleep(time.Second * 1) // Before 'return' 
	// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"

	if len(nlbResp.Listnlbsresponse.NLB) < 1 {
		newErr := fmt.Errorf("Failed to Find the NLB info with the ID on the zone!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	// cblogger.Info("\n\n### result.Listnlbsresponse : ")
	// spew.Dump(result.Listnlbsresponse)

	return &nlbResp.Listnlbsresponse.NLB[0], nil
}
