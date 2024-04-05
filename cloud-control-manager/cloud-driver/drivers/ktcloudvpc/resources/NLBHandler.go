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

	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"
	ktvpclb "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/loadbalancer/v1/loadbalancers"
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
		newErr := fmt.Errorf("Failed to Get the OsNetwork ID of the Subnet : [%v]", getError)
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

	cblogger.Info("\n### Creating New NLB Now!!")
	time.Sleep(time.Second * 15)

	ktNLB, err := nlbHandler.getKTCloudNlbWithName(nlbReqInfo.IId.NameId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud NLB info!! [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}
	nlbIID := irs.IID{SystemId: strconv.Itoa(ktNLB.NlbID)}	

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

	deleteOpts := ktvpclb.DeleteOpts{
		NlbID: 	nlbIID.SystemId,
	}

	start := call.Start()
	delErr := ktvpclb.Delete(nlbHandler.NLBClient, deleteOpts).ExtractErr()
	if delErr != nil {
		newErr := fmt.Errorf("Failed to Delete the KeyPair : [%v]", delErr)
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

	return irs.VMGroupInfo{}, fmt.Errorf("Does not support yet!!")
}

func (nlbHandler *KTVpcNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {

	return false, fmt.Errorf("Does not support yet!!")
}

func (nlbHandler *KTVpcNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	cblogger.Info("KT Cloud Driver: called GetVMGroupHealthInfo()")

	return irs.HealthInfo{}, nil
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
		// DNSName:	"N/A",
		// CspID: 		"N/A",
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
		// CspID: 		"N/A",
	}
	return healthCheckerInfo, nil
}

func (nlbHandler *KTVpcNLBHandler) mappingNlbInfo(nlb *ktvpclb.LoadBalancer) (irs.NLBInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called mappingNlbInfo()")
	// cblogger.Info("\n\n### nlb : ")
	// spew.Dump(nlb)

	// // # Get ImageInfo frome the Disk Volume
	// diskHandler := KTVpcDiskHandler{
	// 	RegionInfo:    vmHandler.RegionInfo,
	// 	VMClient:      vmHandler.VMClient,
	// 	VolumeClient:  vmHandler.VolumeClient,
	// }


	// vpcId, subnetId, publicIp, _, err := vmHandler.getNetIDsWithPrivateIP(vmInfo.PrivateIP)
	// if err != nil {
	// 	newErr := fmt.Errorf("Failed to Get PortForwarding Info. [%v]", err)
	// 	cblogger.Error(newErr.Error())
	// 	return irs.VMInfo{}, newErr
	// }
	// vmInfo.VpcIID.SystemId	  = vpcId
	// vmInfo.SubnetIID.SystemId = subnetId // Caution!!) Need modification. Not Tier 'ID' but 'OsNetworkID' to Create VM through REST API!!
	// vmInfo.PublicIP			  = publicIp

	nlbInfo := irs.NLBInfo{
		IId: irs.IID{
			NameId:   nlb.Name,
			SystemId: strconv.Itoa(nlb.NlbID),
		},
		// VpcIID: irs.IID{
		// 	NameId:   "N/A", // Cauton!!) 'NameId: "N/A"' makes an Error on CB-Spider
		// 	SystemId: "N/A",
		// },
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

	if !strings.EqualFold(nlb.HealthCheckType, "") {
		healthCheckerInfo, err := nlbHandler.getHealthCheckerInfo(nlb)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get HealthChecker Info. frome the NLB. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			return irs.NLBInfo{}, newErr
		}
		nlbInfo.HealthChecker = healthCheckerInfo
	}

	// vmGroupInfo, err := nlbHandler.getVMGroupInfo(strconv.Itoa(nlb.NLBId))
	// if err != nil {
	// 	newErr := fmt.Errorf("Failed to Get VM Group Info with the NLB ID : [%v]", err)
	// 	cblogger.Error(newErr.Error())
	// 	return irs.NLBInfo{}, newErr
	// }
	// nlbInfo.VMGroup = vmGroupInfo
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
		newErr := fmt.Errorf("Failed to Find the NLB info with the Name on the zone!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	// cblogger.Info("\n\n### result.Listnlbsresponse : ")
	// spew.Dump(result.Listnlbsresponse)

	return &nlbList[0], nil
}
