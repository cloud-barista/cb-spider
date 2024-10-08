// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2023.08.

package resources

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	// "github.com/davecgh/go-spew/spew"

	ncloud "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	lb "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/loadbalancer"
	server "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/server"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpNLBHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *server.APIClient
	LBClient       *lb.APIClient
}

const (
	// NCP Classic LB Algorithm type codes : RR (ROUND ROBIN), LC (LEAST_CONNECTION), SIPHS (Source IP Hash)
	DefaultLBAlgorithmType string = "RR" // ROUND ROBIN

	// You can select whether to create a load balancer with public/private IP
	// NCP Classic Cloud NLB network type code : PBLIP(Public IP LB), PRVT(Private IP LB). default : PBLIP
	NcpPublicNlBType   string = "PBLIP"
	NcpInternalNlBType string = "PRVT"

	// 'L7HealthCheckPath' required if ProtocolTypeCode value is 'HTTP' or 'HTTPS' for NCP Classic NLB.
	DefaulthealthCheckPath string = "/index.html"
)

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP Clssic NLBHandler")
}

// Note : Cloud-Barista supports only this case => [ LB : Listener : VM Group : Health Checker = 1 : 1 : 1 : 1 ]
// NCP Classic NLB Supported regions: Korea, US West, Hong Kong, Singapore, Japan, Germany
// ### Caution!! : Listener, VM Group and Healthchecker all use the same protocol type in NCP Classic NLB.(The Protocol specified by 'ProtocolTypeCode' when created).
func (nlbHandler *NcpNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (createNLB irs.NLBInfo, newErr error) {
	cblogger.Info("NPC Classic Cloud Driver: called CreateNLB()")

	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbReqInfo.IId.NameId, "CreateNLB()")

	if strings.EqualFold(nlbReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid NLB NameId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	// ### ProtocolTypeCode : Enter the protocol identification code in the load balancer RULE.
	// The following codes can be entered for the protocol identification code: HTTP, HTTPS, TCP, SSL
	if strings.EqualFold(nlbReqInfo.Listener.Protocol, "HTTP") || strings.EqualFold(nlbReqInfo.Listener.Protocol, "HTTPS") || strings.EqualFold(nlbReqInfo.Listener.Protocol, "TCP") || strings.EqualFold(nlbReqInfo.Listener.Protocol, "SSL") {
		cblogger.Info("# It's Supporting Listener Protocol in NCP Classic!!")
	} else {
		return irs.NLBInfo{}, fmt.Errorf("Invalid Listener Protocol. Must be 'HTTP', 'HTTPS', 'TCP' or 'SSL' for NCP Classic NLB.") // According to the NCP Classic API document.
	}

	if !strings.EqualFold(nlbReqInfo.Listener.Protocol, nlbReqInfo.VMGroup.Protocol) {
		return irs.NLBInfo{}, fmt.Errorf("NLB can be created only when Listener.Protocol and VMGroup.Protocol are of the Same Protocol type in case of NCP Classic.")
	}

	if !strings.EqualFold(nlbReqInfo.VMGroup.Protocol, nlbReqInfo.HealthChecker.Protocol) {
		return irs.NLBInfo{}, fmt.Errorf("NLB can be created only when VMGroup.Protocol and HealthChecker.Protocol are of the Same Protocol type in case of NCP Classic.")
	}

	if !strings.EqualFold(nlbReqInfo.VMGroup.Port, nlbReqInfo.HealthChecker.Port) {
		return irs.NLBInfo{}, fmt.Errorf("NLB can be created only when VMGroup.Port and HealthChecker.Port are of the Same Number in case of NCP Classic.")
	}

	// NCP Classic Cloud NLB network type code : PBLIP(Public IP LB), PRVT(Private IP LB). default : PBLIP
	var lbNetType string
	if strings.EqualFold(nlbReqInfo.Type, "PUBLIC") || strings.EqualFold(nlbReqInfo.Type, "default") || strings.EqualFold(nlbReqInfo.Type, "") {
		lbNetType = NcpPublicNlBType
	} else if strings.EqualFold(nlbReqInfo.Type, "INTERNAL") {
		lbNetType = NcpInternalNlBType
	}

	vmHandler := NcpVMHandler{
		RegionInfo: nlbHandler.RegionInfo,
		VMClient:   nlbHandler.VMClient,
	}
	regionNo, err := vmHandler.GetRegionNo(nlbHandler.RegionInfo.Region)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Region No of the Region Code: [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	zoneNo, err := vmHandler.GetZoneNo(nlbHandler.RegionInfo.Region, nlbHandler.RegionInfo.Zone)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Zone No of the Zone Code : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	zoneNoList := []*string{zoneNo}

	// LB Port num. : Min : 1, Max : 65534
	lbPort, err := strconv.ParseInt(nlbReqInfo.Listener.Port, 10, 32) // Caution : Covert String to Int32
	if err != nil {
		panic(err)
	}
	int32lbPort := int32(lbPort)
	if int32lbPort < 1 || int32lbPort > 65534 {
		newErr := fmt.Errorf("Invalid LB Port Number.(Must be between 1 and 65534)")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	// VM(Server) Port num. : Min : 1, Max : 65534
	vmPort, err := strconv.ParseInt(nlbReqInfo.VMGroup.Port, 10, 32) // Caution : Covert String to Int32
	if err != nil {
		panic(err)
	}
	int32vmPort := int32(vmPort)
	if int32vmPort < 1 || int32vmPort > 65534 {
		newErr := fmt.Errorf("Invalid Target VM Port Number.(Must be between 1 and 65534)")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	// 'L7HealthCheckPath' required if ProtocolTypeCode value is 'HTTP' or 'HTTPS' for NCP Classic NLB.
	var healthCheckPath string
	if strings.EqualFold(nlbReqInfo.Listener.Protocol, "HTTP") || strings.EqualFold(nlbReqInfo.Listener.Protocol, "HTTPS") {
		healthCheckPath = DefaulthealthCheckPath
	}

	// ### ProtocolTypeCode : Enter the protocol identification code in the load balancer RULE.
	// The following codes can be entered for the protocol identification code: HTTP, HTTPS, TCP, SSL
	ruleParameter := []*lb.LoadBalancerRuleParameter{
		{
			ProtocolTypeCode:  ncloud.String(nlbReqInfo.Listener.Protocol), // *** Required (Not Optional)
			LoadBalancerPort:  &int32lbPort,                                // *** Required (Not Optional)
			ServerPort:        &int32vmPort,                                // *** Required (Not Optional)
			L7HealthCheckPath: ncloud.String(healthCheckPath),              // *** Required In case the ProtocolTypeCode is HTTP or HTTPS.

			// ProxyProtocolUseYn: 	ncloud.String(doesUseProxyProtocol),
			// StickySessionUseYn: 	ncloud.String(doesUseStickySession),
			// Http2UseYn:			ncloud.String(doesUseHttp2),

			// Can only be set when the ProtocloTypeCode value is HTTPS. (Options : "HTTP" or "HTTPS". Default : HTTP)
			// ServerProtocolTypeCode: nlbReqInfo.VMGroup.Protocol, // Does Not support yet through NCP API SDK.
		},
	}

	var vmNoList []*string // Caution : var. type
	if len(*nlbReqInfo.VMGroup.VMs) > 0 {
		var vmIds []*string
		for _, IId := range *nlbReqInfo.VMGroup.VMs {
			vmId, err := vmHandler.GetVmIdByName(IId.NameId)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the NCP VM ID with VM Name. [%v]", err.Error())
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.NLBInfo{}, newErr
			}
			vmIds = append(vmIds, &vmId)
		}
		vmNoList = vmIds
	} else {
		cblogger.Info("The VMGroup does Not have any VM Member!!")
	}

	// NCP Classic LB Algorithm type codes : RR (ROUND ROBIN), LC (LEAST_CONNECTION), SIPHS (Source IP Hash)
	lbReq := lb.CreateLoadBalancerInstanceRequest{
		LoadBalancerName:              ncloud.String(nlbReqInfo.IId.NameId),
		LoadBalancerAlgorithmTypeCode: ncloud.String(DefaultLBAlgorithmType),
		LoadBalancerRuleList:          ruleParameter, // *** Required (Not Optional)
		ServerInstanceNoList:          vmNoList,
		NetworkUsageTypeCode:          ncloud.String(lbNetType),
		RegionNo:                      regionNo,   // Caution!! : RegionNo (Not RegionCode)
		ZoneNoList:                    zoneNoList, // Caution!! : ZoneNoList (Not ZoneCodeList)
	}

	callLogStart := call.Start()
	result, err := nlbHandler.LBClient.V2Api.CreateLoadBalancerInstance(&lbReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New NLB : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.LoadBalancerInstanceList) < 1 {
		newErr := fmt.Errorf("Failed to Create New NLB. NLB does Not Exist!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	} else {
		cblogger.Info("Succeeded in Creating New NLB.")
	}

	newNlbIID := irs.IID{SystemId: *result.LoadBalancerInstanceList[0].LoadBalancerInstanceNo}
	_, err = nlbHandler.WaitToGetNlbInfo(newNlbIID) // Wait until the NLB Status is "USED"
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait For Creating the NLB. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
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

func (nlbHandler *NcpNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {
	cblogger.Info("NPC Classic Cloud Driver: called ListNLB()")

	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", "ListNLB()", "ListNLB()")

	vmHandler := NcpVMHandler{
		RegionInfo: nlbHandler.RegionInfo,
		VMClient:   nlbHandler.VMClient,
	}
	regionNo, err := vmHandler.GetRegionNo(nlbHandler.RegionInfo.Region)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Region No of the Region Code: [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	zoneNo, err := vmHandler.GetZoneNo(nlbHandler.RegionInfo.Region, nlbHandler.RegionInfo.Zone)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Zone No of the Zone Code : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	lbReq := lb.GetLoadBalancerInstanceListRequest{
		RegionNo: regionNo, // Caution!! : RegionNo (Not RegionCode)
		ZoneNo:   zoneNo,   // Caution!! : ZoneNo (Not ZoneCode)
	}
	callLogStart := call.Start()
	result, err := nlbHandler.LBClient.V2Api.GetLoadBalancerInstanceList(&lbReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find NLB list from NCP Classic : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var nlbInfoList []*irs.NLBInfo
	if len(result.LoadBalancerInstanceList) < 1 {
		cblogger.Info("# NLB does Not Exist!!")
	} else {
		for _, nlb := range result.LoadBalancerInstanceList {
			nlbInfo, err := nlbHandler.MappingNlbInfo(nlb)
			if err != nil {
				newErr := fmt.Errorf("Failed to Map NLB lnfo. : [%v]", err)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return nil, newErr
			}
			nlbInfoList = append(nlbInfoList, &nlbInfo)
		}
	}
	return nlbInfoList, nil
}

func (nlbHandler *NcpNLBHandler) GetNLB(nlbIID irs.IID) (irs.NLBInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called GetNLB()")

	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "GetNLB()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return irs.NLBInfo{}, newErr
	}

	ncpNlbInfo, err := nlbHandler.GetNcpNlbInfo(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NLB info from NCP Classic : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	nlbInfo, err := nlbHandler.MappingNlbInfo(ncpNlbInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Map the NLB Info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	return nlbInfo, nil
}

func (nlbHandler *NcpNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	cblogger.Info("NCP Classic Cloud Driver: called DeleteNLB()")

	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "DeleteNLB()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	lbNoList := []*string{ncloud.String(nlbIID.SystemId)}
	lbReq := lb.DeleteLoadBalancerInstancesRequest{
		LoadBalancerInstanceNoList: lbNoList,
	}
	callLogStart := call.Start()
	result, err := nlbHandler.LBClient.V2Api.DeleteLoadBalancerInstances(&lbReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the NLB : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if !strings.EqualFold(*result.ReturnMessage, "success") {
		newErr := fmt.Errorf("Failed to Delete the NLB!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	} else {
		cblogger.Info("Succeeded in Deleting the NLB.")
	}

	newNlbIID := irs.IID{SystemId: *result.LoadBalancerInstanceList[0].LoadBalancerInstanceNo}
	_, err = nlbHandler.WaitForDelNlb(newNlbIID) // Wait until 'provisioningStatus' is "Terminated"
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait For Deleting the NLB. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		// return false, newErr // Catuton!! : Incase the status is 'Terminated', fail to get NLB info.
	}

	return true, nil
}

func (nlbHandler *NcpNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called AddVMs()")

	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "AddVMs()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	vmHandler := NcpVMHandler{
		RegionInfo: nlbHandler.RegionInfo,
		VMClient:   nlbHandler.VMClient,
	}
	var newVmIdList []*string
	if len(*vmIIDs) > 0 {
		for _, vmIID := range *vmIIDs {
			vmId, err := vmHandler.GetVmIdByName(vmIID.NameId)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the VM ID with the VM Name : [%v]", err)
				cblogger.Error(newErr.Error())
				return irs.VMGroupInfo{}, newErr
			}
			newVmIdList = append(newVmIdList, ncloud.String(vmId))
		}
	} else {
		newErr := fmt.Errorf("Failded to Find any VM NameId to Add to the VMGroup!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	addVMReq := lb.AddServerInstancesToLoadBalancerRequest{
		LoadBalancerInstanceNo: ncloud.String(nlbIID.SystemId),
		ServerInstanceNoList:   newVmIdList,
	}
	callLogStart := call.Start()
	result, err := nlbHandler.LBClient.V2Api.AddServerInstancesToLoadBalancer(&addVMReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Add the VM to the Target Group : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.LoadBalancerInstanceList[0].LoadBalancedServerInstanceList) < 1 {
		newErr := fmt.Errorf("Failed to Add Any VM to the LoadBalancer!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	} else {
		cblogger.Info("Succeeded in Adding New VM to the LoadBalancer.")
	}

	cblogger.Info("\n\n#### Waiting for Changing the NLB Settings!!")
	_, err = nlbHandler.WaitToGetNlbInfo(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait For Changing the NLB. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
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

func (nlbHandler *NcpNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	cblogger.Info("NCP Classic Cloud Driver: called RemoveVMs()")

	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "RemoveVMs()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	vmHandler := NcpVMHandler{
		RegionInfo: nlbHandler.RegionInfo,
		VMClient:   nlbHandler.VMClient,
	}
	var vmIdList []*string
	if len(*vmIIDs) > 0 {
		for _, vmIID := range *vmIIDs {
			vmId, err := vmHandler.GetVmIdByName(vmIID.NameId)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the VM ID with the VM Name : [%v]", err)
				cblogger.Error(newErr.Error())
				return false, newErr
			}
			vmIdList = append(vmIdList, ncloud.String(vmId)) // ncloud.String func(v string) *string
		}
	} else {
		newErr := fmt.Errorf("Failed to Find any VM NameId to Remove from the LoadBalancer!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	removeVMReq := lb.DeleteServerInstancesFromLoadBalancerRequest{
		LoadBalancerInstanceNo: ncloud.String(nlbIID.SystemId),
		ServerInstanceNoList:   vmIdList,
	}
	callLogStart := call.Start()
	result, err := nlbHandler.LBClient.V2Api.DeleteServerInstancesFromLoadBalancer(&removeVMReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Remove the VM frome the VMGroup : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if !strings.EqualFold(*result.ReturnMessage, "success") {
		newErr := fmt.Errorf("Failed to Remove the VM from the LoadBalancer!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	} else {
		cblogger.Info("Succeeded in Removing the VM from the LoadBalancer!!")
	}

	cblogger.Info("\n\n#### Waiting for Changing the NLB Settings!!")
	_, err = nlbHandler.WaitToGetNlbInfo(nlbIID) // Wait until 'provisioningStatus' is "Changing" -> "Running"
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait For Changing the NLB. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	return true, nil
}

func (nlbHandler *NcpNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called GetVMGroupHealthInfo()")
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "GetVMGroupHealthInfo()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthInfo{}, newErr
	}

	ncpNlbInfo, err := nlbHandler.GetNcpNlbInfo(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NCP Classic NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthInfo{}, newErr
	}
	cblogger.Infof("\n### NLB Status : [%s]", *ncpNlbInfo.LoadBalancerInstanceStatus.Code) // Ex) "USED" means "Running"

	var vmGroupHealthInfo irs.HealthInfo
	if len(ncpNlbInfo.LoadBalancedServerInstanceList) > 0 {
		var allVMs []irs.IID
		var healthVMs []irs.IID
		var unHealthVMs []irs.IID

		for _, member := range ncpNlbInfo.LoadBalancedServerInstanceList {
			allVMs = append(allVMs, irs.IID{NameId: *member.ServerInstance.ServerName, SystemId: *member.ServerInstance.ServerInstanceNo}) // Caution : Not 'VM Member ID' but 'VM System ID'

			// Note : Server Status (Is Not NLB Status) : True (Healthy), False (Unhealthy)
			if *member.ServerHealthCheckStatusList[0].ServerStatus {
				cblogger.Infof("\n### [%s] is a Healthy VM.", *member.ServerInstance.ServerName)
				healthVMs = append(healthVMs, irs.IID{NameId: *member.ServerInstance.ServerName, SystemId: *member.ServerInstance.ServerInstanceNo})
			} else if !*member.ServerHealthCheckStatusList[0].ServerStatus {
				cblogger.Infof("\n### [%s] is an Unhealthy VM.", *member.ServerInstance.ServerName)
				unHealthVMs = append(unHealthVMs, irs.IID{NameId: *member.ServerInstance.ServerName, SystemId: *member.ServerInstance.ServerInstanceNo})
			}
		}

		vmGroupHealthInfo = irs.HealthInfo{
			AllVMs:       &allVMs,
			HealthyVMs:   &healthVMs,
			UnHealthyVMs: &unHealthVMs,
		}
		return vmGroupHealthInfo, nil
	} else {
		return irs.HealthInfo{}, nil
	}
}

func (nlbHandler *NcpNLBHandler) GetListenerInfo(nlb lb.LoadBalancerInstance) (irs.ListenerInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called GetListenerInfo()")

	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", *nlb.LoadBalancerInstanceNo, "GetListenerInfo()")

	if strings.EqualFold(*nlb.LoadBalancerInstanceNo, "") {
		newErr := fmt.Errorf("Invalid LoadBalancerInstance ID. The LB does Not Exit!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.ListenerInfo{}, newErr
	}

	listenerInfo := irs.ListenerInfo{
		Protocol: *nlb.LoadBalancerRuleList[0].ProtocolType.Code,
		Port:     strconv.FormatInt(int64(*nlb.LoadBalancerRuleList[0].LoadBalancerPort), 10),
		DNSName:  *nlb.DomainName,
	}

	virtualIPs := strings.Split(*nlb.VirtualIp, ",")

	if len(virtualIPs) >= 2 {
		cblogger.Infof("First part: %s", virtualIPs[0])
		listenerInfo.IP = virtualIPs[0]
	} else {
		cblogger.Info("nlb.VirtualIp does not contain a comma.")
		listenerInfo.IP = *nlb.VirtualIp
	}

	listenerKVList := []irs.KeyValue{
		// {Key: "NLB_DomainName", Value: *nlb.DomainName},
	}
	listenerInfo.KeyValueList = listenerKVList

	return listenerInfo, nil
}

func (nlbHandler *NcpNLBHandler) GetVMGroupInfo(nlb lb.LoadBalancerInstance) (irs.VMGroupInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called GetVMGroupInfo()")

	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", *nlb.LoadBalancerInstanceNo, "GetVMGroupInfo()")

	if strings.EqualFold(*nlb.LoadBalancerInstanceNo, "") {
		newErr := fmt.Errorf("Invalid LoadBalancer No.!!")
		cblogger.Error(newErr.Error())
		return irs.VMGroupInfo{}, newErr
	}

	// Note : Cloud-Barista supports only this case => [ LB : Listener : VM Group : Health Checker = 1 : 1 : 1 : 1 ]
	vmGroupInfo := irs.VMGroupInfo{
		Protocol: *nlb.LoadBalancerRuleList[0].ProtocolType.Code,
		Port:     strconv.FormatInt(int64(*nlb.LoadBalancerRuleList[0].ServerPort), 10),
	}

	if len(nlb.LoadBalancedServerInstanceList) > 0 {
		vmHandler := NcpVMHandler{
			RegionInfo: nlbHandler.RegionInfo,
			VMClient:   nlbHandler.VMClient,
		}
		var vmIIds []irs.IID
		for _, member := range nlb.LoadBalancedServerInstanceList {
			vm, err := vmHandler.GetNcpVMInfo(*member.ServerInstance.ServerInstanceNo)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the NCP VM Info with ServerInstance No. [%v]", err.Error())
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMGroupInfo{}, newErr
			}
			vmIIds = append(vmIIds, irs.IID{
				NameId:   *vm.ServerName,
				SystemId: *vm.ServerInstanceNo,
			})
		}
		vmGroupInfo.VMs = &vmIIds
	} else {
		cblogger.Info("The VMGroup does Not have any VM Member!!")
	}
	return vmGroupInfo, nil
}

func (nlbHandler *NcpNLBHandler) GetHealthCheckerInfo(nlb lb.LoadBalancerInstance) (irs.HealthCheckerInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called GetHealthCheckerInfo()")

	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", *nlb.LoadBalancerInstanceNo, "GetHealthCheckerInfo()")

	if len(nlb.LoadBalancedServerInstanceList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any NCP VM List.")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.HealthCheckerInfo{}, newErr
	}

	// ### In case of NCP Classic Cloud,
	// When a load balancer is created, a health check is performed with the designated server port, and servers that fail the health check are excluded from the load balancing target.
	// In the case of the HTTP service, if you enter the content path in the L7 Health Check field, the normal operation of the content is checked, and servers that fail the health check are excluded from load balancing. Input (example) /somedir/index.html
	healthCheckerInfo := irs.HealthCheckerInfo{
		Protocol: *nlb.LoadBalancedServerInstanceList[0].ServerHealthCheckStatusList[0].ProtocolType.Code,
		Port:     strconv.FormatInt(int64(*nlb.LoadBalancedServerInstanceList[0].ServerHealthCheckStatusList[0].ServerPort), 10), // Note!! : ServerPort
		// Interval: int(*ncpTargetGroupList[0].HealthCheckCycle),
		Timeout: int(*nlb.ConnectionTimeout),
		// Threshold: int(*ncpTargetGroupList[0].HealthCheckUpThreshold),
		// CspID:    *ncpTargetGroupList[0].TargetGroupNo,
	}
	keyValueList := []irs.KeyValue{
		{Key: "L7HealthCheckPath", Value: *nlb.LoadBalancedServerInstanceList[0].ServerHealthCheckStatusList[0].L7HealthCheckPath},
	}
	healthCheckerInfo.KeyValueList = keyValueList
	return healthCheckerInfo, nil
}

func (nlbHandler *NcpNLBHandler) WaitToGetNlbInfo(nlbIID irs.IID) (bool, error) {
	cblogger.Info("NCP Classic Cloud Driver: called WaitToGetNlbInfo()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	curRetryCnt := 0
	maxRetryCnt := 1000
	for {
		curRetryCnt++
		nlbStatus, err := nlbHandler.GetNcpNlbStatus(nlbIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the NLB Provisioning Status : [%v]", err)
			cblogger.Error(newErr.Error())
			return false, newErr
		} else if strings.EqualFold(nlbStatus, "Running") { // Caution!!
			return true, nil
		} else {
			cblogger.Infof("\n### NLB Status : [%s]", nlbStatus)
		}
		time.Sleep(5 * time.Second)
		if curRetryCnt > maxRetryCnt {
			return false, fmt.Errorf("Failed to Create the NLB. Exceeded maximum retry count %d", maxRetryCnt)
		}
	}
}

func (nlbHandler *NcpNLBHandler) WaitForDelNlb(nlbIID irs.IID) (bool, error) {
	cblogger.Info("NCP Classic Cloud Driver: called WaitForDelNlb()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	curRetryCnt := 0
	maxRetryCnt := 500
	for {
		curRetryCnt++
		nlbStatus, err := nlbHandler.GetNcpNlbStatus(nlbIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the NLB Status : [%v]", err)
			cblogger.Error(newErr.Error())
			return false, newErr
		} else if !strings.EqualFold(nlbStatus, "Running") && !strings.EqualFold(nlbStatus, "Terminating") {
			return true, nil
		} else {
			cblogger.Infof("\n### NLB Status : [%s]", nlbStatus)
		}
		time.Sleep(3 * time.Second)
		if curRetryCnt > maxRetryCnt {
			return false, fmt.Errorf("Failed to Del the NLB. Exceeded maximum retry count %d", maxRetryCnt)
		}
	}
}

func (nlbHandler *NcpNLBHandler) GetNcpNlbStatus(nlbIID irs.IID) (string, error) {
	cblogger.Info("NCP Classic Cloud Driver: called GetNcpNlbStatus()")
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "GetNcpNlbStatus()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	ncpNlbInfo, err := nlbHandler.GetNcpNlbInfo(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NCP Classic NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	if strings.EqualFold(*ncpNlbInfo.LoadBalancerInstanceStatus.Code, "USED") {
		return "Running", nil
	} else if strings.EqualFold(*ncpNlbInfo.LoadBalancerInstanceStatus.Code, "INIT") {
		return "Creating", nil
	} else {
		return *ncpNlbInfo.LoadBalancerInstanceStatus.Code, nil
	}
}

func (nlbHandler *NcpNLBHandler) GetNcpNlbInfo(nlbIID irs.IID) (*lb.LoadBalancerInstance, error) {
	cblogger.Info("NCP Classic Cloud Driver: called GetNcpNlbInfo()")

	InitLog()
	callLogInfo := GetCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "GetNcpNlbInfo()")

	if strings.EqualFold(nlbIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid NLB ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	vmHandler := NcpVMHandler{
		RegionInfo: nlbHandler.RegionInfo,
		VMClient:   nlbHandler.VMClient,
	}
	regionNo, err := vmHandler.GetRegionNo(nlbHandler.RegionInfo.Region)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Region No of the Region Code: [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	zoneNo, err := vmHandler.GetZoneNo(nlbHandler.RegionInfo.Region, nlbHandler.RegionInfo.Zone)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Zone No of the Zone Code : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	lbInstanceNoList := []*string{ncloud.String(nlbIID.SystemId)}
	lbReq := lb.GetLoadBalancerInstanceListRequest{
		RegionNo:                   regionNo, // Caution!! : Not RegionCode
		ZoneNo:                     zoneNo,   // Caution!! : Not ZoneCode
		LoadBalancerInstanceNoList: lbInstanceNoList,
	}
	callLogStart := call.Start()
	result, err := nlbHandler.LBClient.V2Api.GetLoadBalancerInstanceList(&lbReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NLB Info from NCP Classic : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.LoadBalancerInstanceList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any NLB Info with the ID from NCP Classic!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Getting the NLB Info.")
	}
	return result.LoadBalancerInstanceList[0], nil
}

// NCP Classic LB resource Def. : https://api.ncloud-docs.com/docs/networking-loadbalancing-createloadbalancerinstance
func (nlbHandler *NcpNLBHandler) MappingNlbInfo(nlb *lb.LoadBalancerInstance) (irs.NLBInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called MappingNlbInfo()")

	if strings.EqualFold(*nlb.LoadBalancerInstanceNo, "") {
		newErr := fmt.Errorf("Invalid LoadBalancer Instance Info!!")
		cblogger.Error(newErr.Error())
		return irs.NLBInfo{}, newErr
	}

	// cblogger.Info("\n### NCP NlbInfo")
	// spew.Dump(nlb)

	// You can select whether to create a load balancer with public/private IP
	// PBLIP(Public IP LB), PRVT(Private IP LB). default : PBLIP
	var nlbType string
	if strings.EqualFold(*nlb.NetworkUsageType.Code, "PBLIP") {
		nlbType = "PUBLIC"
	} else if strings.EqualFold(*nlb.NetworkUsageType.Code, "PRVT") {
		nlbType = "INTERNAL"
	}

	convertedTime, err := convertTimeFormat(*nlb.CreateDate)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert the Time Format!!")
		cblogger.Error(newErr.Error())
		return irs.NLBInfo{}, newErr
	}

	nlbInfo := irs.NLBInfo{
		IId: irs.IID{
			NameId:   *nlb.LoadBalancerName,
			SystemId: *nlb.LoadBalancerInstanceNo,
		},
		// VpcIID: irs.IID{
		// 	SystemId: *nlb.VpcNo,
		// },
		Type:        nlbType,
		Scope:       "REGION",
		CreatedTime: convertedTime,
	}

	var nlbStatus string
	if strings.EqualFold(*nlb.LoadBalancerInstanceStatus.Code, "USED") {
		nlbStatus = "Running"
	} else {
		nlbStatus = *nlb.LoadBalancerInstanceStatus.Code
	}

	keyValueList := []irs.KeyValue{
		{Key: "Region", Value: *nlb.Region.RegionCode},
		{Key: "NLB_Status", Value: nlbStatus},
		{Key: "LoadBalancerAlgorithmType", Value: *nlb.LoadBalancerAlgorithmType.CodeName},
	}
	nlbInfo.KeyValueList = keyValueList

	// 	// Caution : If Get Listener info during Changing settings of a NLB., makes an Error.
	// 	if strings.EqualFold(*nlb.LoadBalancerInstanceStatusName, "Changing") {
	// 		cblogger.Info("### The NLB is being Changed Settings Now. Try again after finishing the changing processes.")
	// 	} else {
	// 		// Note : It is assumed that there is only one listener in the LB.
	//  }
	listenerInfo, err := nlbHandler.GetListenerInfo(*nlb)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Listener Info : [%v]", err.Error())
		cblogger.Error(newErr.Error())
		return irs.NLBInfo{}, newErr
	}
	nlbInfo.Listener = listenerInfo

	if len(nlb.LoadBalancedServerInstanceList) > 0 {
		vmGroupInfo, err := nlbHandler.GetVMGroupInfo(*nlb)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get VMGroup Info from the NLB. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			// return irs.NLBInfo{}, newErr
		}
		nlbInfo.VMGroup = vmGroupInfo

		// ### In case of NCP Classic Cloud,
		// When a load balancer is created, a health check is performed with the designated server port, and servers that fail the health check are excluded from the load balancing target.
		// In the case of the HTTP service, if you enter the content path in the L7 Health Check field, the normal operation of the content is checked, and servers that fail the health check are excluded from load balancing. Input (example) /somedir/index.html
		healthCheckerInfo, err := nlbHandler.GetHealthCheckerInfo(*nlb)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get HealthChecker Info. frome the NLB. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		}
		nlbInfo.HealthChecker = healthCheckerInfo
	}

	return nlbInfo, nil
}

// Note!! : Will be decided later if we would support bellow methoeds or not.
// ------ Frontend Control
func (nlbHandler *NcpNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {

	return irs.ListenerInfo{}, fmt.Errorf("Does not support yet!!")
}

// ------ Backend Control
func (nlbHandler *NcpNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {

	return irs.VMGroupInfo{}, fmt.Errorf("Does not support yet!!")
}

func (nlbHandler *NcpNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {

	return irs.HealthCheckerInfo{}, fmt.Errorf("Does not support yet!!")
}

func (NLBHandler *NcpNLBHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}
