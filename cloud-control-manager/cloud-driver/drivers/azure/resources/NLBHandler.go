package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"strconv"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzureNLBHandler struct {
	CredentialInfo               idrv.CredentialInfo
	Region                       idrv.RegionInfo
	Ctx                          context.Context
	NLBClient                    *armnetwork.LoadBalancersClient
	NLBBackendAddressPoolsClient *armnetwork.LoadBalancerBackendAddressPoolsClient
	VNicClient                   *armnetwork.InterfacesClient
	PublicIPClient               *armnetwork.PublicIPAddressesClient
	VMClient                     *armcompute.VirtualMachinesClient
	SubnetClient                 *armnetwork.SubnetsClient
	IPConfigClient               *armnetwork.InterfaceIPConfigurationsClient
	NLBLoadBalancingRulesClient  *armnetwork.LoadBalancerLoadBalancingRulesClient
	MetricClient                 *azquery.MetricsClient
}

type BackendAddressesIPRefType string

type NLBType string
type NLBScope string

const (
	FrontEndIPConfigPrefix                                       = "frontEndIp"
	LoadBalancingRulesPrefix                                     = "lbrule"
	ProbeNamePrefix                                              = "probe"
	BackEndAddressPoolPrefix                                     = "backend"
	BackendAddressesIPAddressRef       BackendAddressesIPRefType = "IPADDRESS"
	BackendAddressesIPConfigurationRef BackendAddressesIPRefType = "IPCONFIGURATION"
	NLBPublicType                      NLBType                   = "PUBLIC"
	NLBInternalType                    NLBType                   = "INTERNAL"
	NLBGlobalType                      NLBScope                  = "GLOBAL"
	NLBRegionType                      NLBScope                  = "REGION"
)

// ------ NLB Management
func (nlbHandler *AzureNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (createNLB irs.NLBInfo, createError error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbReqInfo.IId.NameId, "CreateNLB()")
	start := call.Start()
	err := checkValidationNLB(nlbReqInfo)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError)
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}
	exist, err := nlbHandler.existNLB(nlbReqInfo.IId)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError)
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}
	if exist {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = already exist NLB %s", nlbReqInfo.IId.NameId))
		cblogger.Error(createError)
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}

	publicIp, err := nlbHandler.createPublicIP(nlbReqInfo.IId.NameId)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError)
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}

	defer func() {
		if createError != nil {
			_, cleanerErr := nlbHandler.NLBCleaner(nlbReqInfo.IId)
			if cleanerErr != nil {
				createError = errors.New(fmt.Sprintf("%s and Failed to rollback err = %s", createError.Error(), cleanerErr.Error()))
			}
			cblogger.Error(createError)
			LoggingError(hiscallInfo, createError)
		}
	}()

	// Create NLB PublicIP (NLB EndPoint)
	var frontendIPConfigurations []*armnetwork.FrontendIPConfiguration
	frontendIPConfiguration := getAzureFrontendIPConfiguration(&publicIp)
	frontendIPConfigurations = append(frontendIPConfigurations, frontendIPConfiguration)

	// Create healthCheckProbe (BackendPool VM HealthCheck)
	var probes []*armnetwork.Probe
	healthCheckProbe, err := getAzureProbeByCBHealthChecker(nlbReqInfo.HealthChecker)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError)
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}
	probes = append(probes, healthCheckProbe)

	var backendAddressPools []*armnetwork.BackendAddressPool
	var loadBalancingRules []*armnetwork.LoadBalancingRule
	backEndAddressPoolName := generateRandName(BackEndAddressPoolPrefix)
	// Create BackendAddressPools (Front => Backend)
	backendAddressPools = append(backendAddressPools, &armnetwork.BackendAddressPool{Name: &backEndAddressPoolName})

	// Create Related ID for Create loadBalancingRules (Front => Backend)
	nlbId := GetNetworksResourceIdByName(nlbHandler.CredentialInfo, nlbHandler.Region, AzureLoadBalancers, nlbReqInfo.IId.NameId)
	frontEndIPConfigId := fmt.Sprintf("%s/frontendIPConfigurations/%s", nlbId, *frontendIPConfiguration.Name)
	backEndAddressPoolId := fmt.Sprintf("%s/backendAddressPools/%s", nlbId, backEndAddressPoolName)
	if len(*nlbReqInfo.VMGroup.VMs) == 0 {
		backEndAddressPoolId = ""
	}
	probeId := fmt.Sprintf("%s/probes/%s", nlbId, *healthCheckProbe.Name)

	// Create loadBalancingRules (Front => Backend)
	var loadBalancingRule *armnetwork.LoadBalancingRule
	loadBalancingRule, err = getAzureLoadBalancingRuleByCBListenerInfo(nlbReqInfo.Listener, nlbReqInfo.VMGroup, frontEndIPConfigId, backEndAddressPoolId, probeId)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError)
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}

	loadBalancingRules = append(loadBalancingRules, loadBalancingRule)

	options := armnetwork.LoadBalancer{
		Location: toStrPtr(nlbHandler.Region.Region),
		SKU: &armnetwork.LoadBalancerSKU{
			Name: (*armnetwork.LoadBalancerSKUName)(toStrPtr(string(armnetwork.LoadBalancerSKUNameStandard))),
			Tier: (*armnetwork.LoadBalancerSKUTier)(toStrPtr(string(armnetwork.LoadBalancerSKUTierRegional))),
		},
		Properties: &armnetwork.LoadBalancerPropertiesFormat{
			// TODO: Deliver multiple FrontendIPConfigurations, BackendAddressPools, Probes, loadBalancingRules in the future
			FrontendIPConfigurations: frontendIPConfigurations,
			BackendAddressPools:      backendAddressPools,
			Probes:                   probes,
			LoadBalancingRules:       loadBalancingRules,
		},
		Tags: map[string]*string{
			"createdAt": toStrPtr(strconv.FormatInt(time.Now().Unix(), 10)),
		},
	}

	if nlbReqInfo.TagList != nil {
		for _, tag := range nlbReqInfo.TagList {
			options.Tags[tag.Key] = &tag.Value
		}
	}

	poller, err := nlbHandler.NLBClient.BeginCreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.Region, nlbReqInfo.IId.NameId, options, nil)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError)
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}
	_, err = poller.PollUntilDone(nlbHandler.Ctx, nil)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError)
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}

	if len(*nlbReqInfo.VMGroup.VMs) > 0 {
		// Update BackEndPool
		privateIPs := make([]string, len(*nlbReqInfo.VMGroup.VMs))
		for i, vmIId := range *nlbReqInfo.VMGroup.VMs {
			convertedIID, err := ConvertVMIID(vmIId, nlbHandler.CredentialInfo, nlbHandler.Region)
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to Get VM. err = %s", err))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return irs.NLBInfo{}, getErr
			}
			//vm, err := GetRawVM(vmIId, nlbHandler.Region.Region, nlbHandler.VMClient, nlbHandler.Ctx)
			vm, err := GetRawVM(convertedIID, nlbHandler.Region.Region, nlbHandler.VMClient, nlbHandler.Ctx)
			if err != nil {
				createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
				cblogger.Error(createError)
				LoggingError(hiscallInfo, createError)
				return irs.NLBInfo{}, createError
			}
			ip, _ := nlbHandler.getVPCNameSubnetNameAndPrivateIPByVM(vm)
			privateIPs[i] = ip
		}

		vpcId := GetNetworksResourceIdByName(nlbHandler.CredentialInfo, nlbHandler.Region, AzureVirtualNetworks, nlbReqInfo.VpcIID.NameId)
		// subnetId := fmt.Sprintf("%s/subnets/%s", vpcId, subnetName)

		resp, err := nlbHandler.NLBBackendAddressPoolsClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, nlbReqInfo.IId.NameId, backEndAddressPoolName, nil)
		if err != nil {
			createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
			cblogger.Error(createError)
			LoggingError(hiscallInfo, createError)
			return irs.NLBInfo{}, createError
		}
		LoadBalancerBackendAddresses := resp.BackendAddressPool.Properties.LoadBalancerBackendAddresses

		for _, ip := range privateIPs {
			LoadBalancerBackendAddress, err := nlbHandler.getLoadBalancerBackendAddress(backEndAddressPoolName, vpcId, ip)
			if err != nil {
				createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
				cblogger.Error(createError)
				LoggingError(hiscallInfo, createError)
				return irs.NLBInfo{}, createError
			}
			LoadBalancerBackendAddresses = append(LoadBalancerBackendAddresses, LoadBalancerBackendAddress)
		}

		resp.BackendAddressPool.Properties.LoadBalancerBackendAddresses = LoadBalancerBackendAddresses

		poller, err := nlbHandler.NLBBackendAddressPoolsClient.BeginCreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.Region, nlbReqInfo.IId.NameId, backEndAddressPoolName, resp.BackendAddressPool, nil)
		if err != nil {
			createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
			cblogger.Error(createError)
			LoggingError(hiscallInfo, createError)
			return irs.NLBInfo{}, createError
		}
		_, err = poller.PollUntilDone(nlbHandler.Ctx, nil)
		if err != nil {
			createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
			cblogger.Error(createError)
			LoggingError(hiscallInfo, createError)
			return irs.NLBInfo{}, createError
		}
	}

	rawNLB, err := nlbHandler.getRawNLB(nlbReqInfo.IId)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError)
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}
	info, err := nlbHandler.setterNLB(rawNLB)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError)
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}
	LoggingInfo(hiscallInfo, start)
	return *info, nil
}
func (nlbHandler *AzureNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", "NLB", "ListNLB()")
	start := call.Start()

	var nlbList []*armnetwork.LoadBalancer

	pager := nlbHandler.NLBClient.NewListPager(nlbHandler.Region.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(nlbHandler.Ctx)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List Cluster. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}

		for _, nlb := range page.Value {
			nlbList = append(nlbList, nlb)
		}
	}

	nlbInfoList := make([]*irs.NLBInfo, len(nlbList))
	var err error

	for i, rawNLB := range nlbList {
		nlbInfoList[i], err = nlbHandler.setterNLB(rawNLB)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List NLB. err = %s", err.Error()))
			cblogger.Error(getErr)
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}
	}
	LoggingInfo(hiscallInfo, start)
	return nlbInfoList, nil
}
func (nlbHandler *AzureNLBHandler) GetNLB(nlbIID irs.IID) (irs.NLBInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "GetNLB()")
	start := call.Start()
	rawNLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get NLB. err = %s", err.Error()))
		cblogger.Error(getErr)
		LoggingError(hiscallInfo, getErr)
		return irs.NLBInfo{}, getErr
	}

	info, err := nlbHandler.setterNLB(rawNLB)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get NLB. err = %s", err.Error()))
		cblogger.Error(getErr)
		LoggingError(hiscallInfo, getErr)
		return irs.NLBInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return *info, nil
}
func (nlbHandler *AzureNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "DeleteNLB()")
	start := call.Start()
	deleteResult, err := nlbHandler.NLBCleaner(nlbIID)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)
	return deleteResult, nil
}

// ------ Frontend Control
// ------ Backend Control
func (nlbHandler *AzureNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "ChangeListener()")
	start := call.Start()

	protocol, err := convertListenerInfoProtocolsToInboundRuleProtocol(listener.Protocol)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	frontendPort, err := strconv.Atoi(listener.Port)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}

	nlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}

	if len(nlb.Properties.LoadBalancingRules) < 1 {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = not Exist Listener"))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}

	nlb.Properties.LoadBalancingRules[0].Properties.Protocol = &protocol
	frontendPortInt32 := int32(frontendPort)
	nlb.Properties.LoadBalancingRules[0].Properties.FrontendPort = &frontendPortInt32

	poller, err := nlbHandler.NLBClient.BeginCreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.Region, *nlb.Name, *nlb, nil)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	_, err = poller.PollUntilDone(nlbHandler.Ctx, nil)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	rawNLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	info, err := nlbHandler.setterNLB(rawNLB)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	LoggingInfo(hiscallInfo, start)
	return info.Listener, nil
}
func (nlbHandler *AzureNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "ChangeVMGroupInfo()")
	start := call.Start()
	nlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	// Rule Change
	backendPort, err := strconv.Atoi(vmGroup.Port)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	if len(nlb.Properties.LoadBalancingRules) < 1 {
		return irs.VMGroupInfo{}, errors.New("not Exist Listener")
	}

	backendPortInt32 := int32(backendPort)
	nlb.Properties.LoadBalancingRules[0].Properties.BackendPort = &backendPortInt32

	poller, err := nlbHandler.NLBClient.BeginCreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.Region, *nlb.Name, *nlb, nil)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	_, err = poller.PollUntilDone(nlbHandler.Ctx, nil)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}

	nlb, err = nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	info, err := nlbHandler.setterNLB(nlb)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	LoggingInfo(hiscallInfo, start)
	return info.VMGroup, nil
}
func (nlbHandler *AzureNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "AddVMs()")
	start := call.Start()
	nlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
		cblogger.Error(addErr.Error())
		LoggingError(hiscallInfo, addErr)
		return irs.VMGroupInfo{}, addErr
	}

	if len(nlb.Properties.BackendAddressPools) > 0 && len(*vmIIDs) > 0 {
		backendPools := nlb.Properties.BackendAddressPools
		cbOnlyOneBackendPool := backendPools[0]

		nlbCurrentVMIIds, err := nlbHandler.getVMIIDsByLoadBalancerBackendAddresses(cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses)
		existCheck := false
		for _, currentVMIId := range nlbCurrentVMIIds {
			for _, addVmIId := range *vmIIDs {
				if strings.EqualFold(currentVMIId.NameId, addVmIId.NameId) {
					existCheck = true
					break
				}
			}
			if existCheck {
				break
			}
		}

		if existCheck {
			addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = can't add already exist vm"))
			cblogger.Error(addErr.Error())
			LoggingError(hiscallInfo, addErr)
			return irs.VMGroupInfo{}, addErr
		}

		backendPoolName := *cbOnlyOneBackendPool.Name
		privateIPs := make([]string, len(*vmIIDs))
		for i, vmIId := range *vmIIDs {
			vm, err := GetRawVM(vmIId, nlbHandler.Region.Region, nlbHandler.VMClient, nlbHandler.Ctx)
			if err != nil {
				cblogger.Error(err.Error())
				LoggingError(hiscallInfo, err)
				return irs.VMGroupInfo{}, err
			}
			ip, err := nlbHandler.getVPCNameSubnetNameAndPrivateIPByVM(vm)
			if err != nil {
				addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
				cblogger.Error(addErr.Error())
				LoggingError(hiscallInfo, addErr)
				return irs.VMGroupInfo{}, addErr
			}
			privateIPs[i] = ip
		}
		// 적어도 하나의 VM 존재시
		vpcId := ""
		if cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses != nil && len(cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses) > 0 {
			vpcIId, err := nlbHandler.getVPCIIDByLoadBalancerBackendAddresses(cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses)
			if err != nil {
				addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
				cblogger.Error(addErr.Error())
				LoggingError(hiscallInfo, addErr)
				return irs.VMGroupInfo{}, addErr
			}
			vpcId = vpcIId.SystemId
		} else {
			for _, vmIId := range *vmIIDs {
				vm, err := GetRawVM(vmIId, nlbHandler.Region.Region, nlbHandler.VMClient, nlbHandler.Ctx)
				if err != nil {
					addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
					cblogger.Error(addErr.Error())
					LoggingError(hiscallInfo, addErr)
					return irs.VMGroupInfo{}, addErr
				}
				vpcIID, err := nlbHandler.getVPCIIDByVM(vm)
				if err != nil {
					addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
					cblogger.Error(addErr.Error())
					LoggingError(hiscallInfo, addErr)
					return irs.VMGroupInfo{}, addErr
				}
				if vpcId == "" {
					vpcId = vpcIID.SystemId
				} else if !strings.EqualFold(vpcId, vpcIID.SystemId) {
					addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = vms in the service group must belong to the same vpc"))
					cblogger.Error(addErr.Error())
					LoggingError(hiscallInfo, addErr)
					return irs.VMGroupInfo{}, addErr
				}
			}
		}

		resp, err := nlbHandler.NLBBackendAddressPoolsClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, nlbIID.NameId, backendPoolName, nil)
		if err != nil {
			addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
			cblogger.Error(addErr.Error())
			LoggingError(hiscallInfo, addErr)
			return irs.VMGroupInfo{}, addErr
		}
		LoadBalancerBackendAddresses := resp.BackendAddressPool.Properties.LoadBalancerBackendAddresses

		for _, ip := range privateIPs {
			LoadBalancerBackendAddress, err := nlbHandler.getLoadBalancerBackendAddress(backendPoolName, vpcId, ip)
			if err != nil {
				addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
				cblogger.Error(addErr.Error())
				LoggingError(hiscallInfo, addErr)
				return irs.VMGroupInfo{}, addErr
			}
			LoadBalancerBackendAddresses = append(LoadBalancerBackendAddresses, LoadBalancerBackendAddress)
		}

		resp.BackendAddressPool.Properties.LoadBalancerBackendAddresses = LoadBalancerBackendAddresses

		poller, err := nlbHandler.NLBBackendAddressPoolsClient.BeginCreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.Region, nlbIID.NameId, backendPoolName, resp.BackendAddressPool, nil)
		if err != nil {
			addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
			cblogger.Error(addErr.Error())
			LoggingError(hiscallInfo, addErr)
			return irs.VMGroupInfo{}, addErr
		}
		_, err = poller.PollUntilDone(nlbHandler.Ctx, nil)
		if err != nil {
			addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
			cblogger.Error(addErr.Error())
			LoggingError(hiscallInfo, addErr)
			return irs.VMGroupInfo{}, addErr
		}
	}
	nlb, err = nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
		cblogger.Error(addErr.Error())
		LoggingError(hiscallInfo, addErr)
		return irs.VMGroupInfo{}, addErr
	}
	info, err := nlbHandler.setterNLB(nlb)
	if err != nil {
		addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
		cblogger.Error(addErr.Error())
		LoggingError(hiscallInfo, addErr)
		return irs.VMGroupInfo{}, addErr
	}
	LoggingInfo(hiscallInfo, start)
	return info.VMGroup, nil
}
func (nlbHandler *AzureNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "AddVMs()")
	start := call.Start()
	nlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
		cblogger.Error(removeErr.Error())
		LoggingError(hiscallInfo, removeErr)
		return false, removeErr
	}

	if len(nlb.Properties.BackendAddressPools) > 0 && len(*vmIIDs) > 0 {
		backendPools := nlb.Properties.BackendAddressPools
		cbOnlyOneBackendPool := backendPools[0]

		nlbCurrentVMIIds, err := nlbHandler.getVMIIDsByLoadBalancerBackendAddresses(cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses)

		for _, removeVmIId := range *vmIIDs {
			existCheck := false
			for _, currentVMIId := range nlbCurrentVMIIds {
				if strings.EqualFold(currentVMIId.NameId, removeVmIId.NameId) {
					existCheck = true
				}
			}
			if !existCheck {
				removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = can't remove not exist vm"))
				cblogger.Error(removeErr.Error())
				LoggingError(hiscallInfo, removeErr)
				return false, removeErr
			}
		}

		backendPoolName := *cbOnlyOneBackendPool.Name

		removPrivateIPs := make([]string, len(*vmIIDs))
		for i, vmIId := range *vmIIDs {
			vm, err := GetRawVM(vmIId, nlbHandler.Region.Region, nlbHandler.VMClient, nlbHandler.Ctx)
			if err != nil {
				cblogger.Error(err.Error())
				LoggingError(hiscallInfo, err)
				return false, err
			}
			ip, err := nlbHandler.getVPCNameSubnetNameAndPrivateIPByVM(vm)
			if err != nil {
				removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
				cblogger.Error(removeErr.Error())
				LoggingError(hiscallInfo, removeErr)
				return false, removeErr
			}
			removPrivateIPs[i] = ip
		}

		resp, err := nlbHandler.NLBBackendAddressPoolsClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, nlbIID.NameId, backendPoolName, nil)
		if err != nil {
			removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
			cblogger.Error(removeErr.Error())
			LoggingError(hiscallInfo, removeErr)
			return false, removeErr
		}

		newLoadBalancerBackendAddresses := make([]*armnetwork.LoadBalancerBackendAddress, 0)
		currentVMIIds, err := nlbHandler.getVMIIDsByLoadBalancerBackendAddresses(resp.BackendAddressPool.Properties.LoadBalancerBackendAddresses)
		if err != nil {
			removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
			cblogger.Error(removeErr.Error())
			LoggingError(hiscallInfo, removeErr)
			return false, removeErr
		}
		currentVMPrivateIPs := make([]string, len(currentVMIIds))

		for i, vmIId := range currentVMIIds {
			vm, err := GetRawVM(vmIId, nlbHandler.Region.Region, nlbHandler.VMClient, nlbHandler.Ctx)
			if err != nil {
				removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
				cblogger.Error(removeErr.Error())
				LoggingError(hiscallInfo, removeErr)
				return false, removeErr
			}
			ip, err := nlbHandler.getVPCNameSubnetNameAndPrivateIPByVM(vm)
			if err != nil {
				removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
				cblogger.Error(removeErr.Error())
				LoggingError(hiscallInfo, removeErr)
				return false, removeErr
			}
			currentVMPrivateIPs[i] = ip
		}

		vpcIId, err := nlbHandler.getVPCIIDByLoadBalancerBackendAddresses(cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses)
		if err != nil {
			removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
			cblogger.Error(removeErr.Error())
			LoggingError(hiscallInfo, removeErr)
			return false, removeErr
		}
		for _, currentIP := range currentVMPrivateIPs {
			chk := false
			addIPSet := ""
			for _, removeIP := range removPrivateIPs {
				if strings.EqualFold(removeIP, currentIP) {
					chk = true
					break
				}
				addIPSet = currentIP
			}
			if !chk {
				LoadBalancerBackendAddress, err := nlbHandler.getLoadBalancerBackendAddress(backendPoolName, vpcIId.SystemId, addIPSet)
				if err != nil {
					removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
					cblogger.Error(removeErr.Error())
					LoggingError(hiscallInfo, removeErr)
					return false, removeErr
				}
				newLoadBalancerBackendAddresses = append(newLoadBalancerBackendAddresses, LoadBalancerBackendAddress)
			}
		}
		resp.BackendAddressPool.Properties.LoadBalancerBackendAddresses = newLoadBalancerBackendAddresses

		poller, err := nlbHandler.NLBBackendAddressPoolsClient.BeginCreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.Region, nlbIID.NameId, backendPoolName, resp.BackendAddressPool, nil)
		if err != nil {
			removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
			cblogger.Error(removeErr.Error())
			LoggingError(hiscallInfo, removeErr)
			return false, removeErr
		}
		_, err = poller.PollUntilDone(nlbHandler.Ctx, nil)
		if err != nil {
			removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
			cblogger.Error(removeErr.Error())
			LoggingError(hiscallInfo, removeErr)
			return false, removeErr
		}
		LoggingInfo(hiscallInfo, start)
		return true, nil
	}
	removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = no exist vm to remove"))
	cblogger.Error(removeErr.Error())
	LoggingError(hiscallInfo, removeErr)
	return false, removeErr
}
func (nlbHandler *AzureNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "GetVMGroupHealthInfo()")
	start := call.Start()
	rawNLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to GetVMGroupHealthInfo NLB. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.HealthInfo{}, getErr
	}
	nlbId := *rawNLB.ID
	vmIPs, err := nlbHandler.getVMIPs(nlbIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to GetVMGroupHealthInfo NLB. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.HealthInfo{}, getErr
	}
	allVMIIds := make([]irs.IID, len(vmIPs))
	healthVMIIds := make([]irs.IID, 0)
	unhealthVMIIds := make([]irs.IID, 0)
	for i, vmip := range vmIPs {
		allVMIIds[i] = vmip.VMIID
		status, err := nlbHandler.getProbeMetricStatus(nlbId, vmip.IP)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to GetVMGroupHealthInfo NLB. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return irs.HealthInfo{}, getErr
		}
		if status {
			healthVMIIds = append(healthVMIIds, vmip.VMIID)
		} else {
			unhealthVMIIds = append(unhealthVMIIds, vmip.VMIID)
		}
	}
	LoggingInfo(hiscallInfo, start)
	return irs.HealthInfo{
		AllVMs:       &allVMIIds,
		HealthyVMs:   &healthVMIIds,
		UnHealthyVMs: &unhealthVMIIds,
	}, nil
}
func (nlbHandler *AzureNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "ChangeHealthCheckerInfo()")
	start := call.Start()
	err := checkValidationNLBHealthCheck(healthChecker)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	nlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.HealthCheckerInfo{}, changeErr
	}
	if len(nlb.Properties.Probes) > 0 {
		protocol := armnetwork.ProbeProtocolHTTP
		switch strings.ToUpper(healthChecker.Protocol) {
		case "HTTP", "HTTPS", "TCP":
			protocol = armnetwork.ProbeProtocol(strings.Title(strings.ToLower(healthChecker.Protocol)))
		default:
			changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = invalid HealthCheckerInfo Protocol"))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.HealthCheckerInfo{}, changeErr
		}
		port, err := strconv.Atoi(healthChecker.Port)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = invalid HealthCheckerInfo Port"))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.HealthCheckerInfo{}, changeErr
		}
		if healthChecker.Interval < 5 {
			return irs.HealthCheckerInfo{}, errors.New("invalid HealthCheckerInfo Interval, interval must be greater than 5")
		}
		if healthChecker.Threshold < 1 {
			return irs.HealthCheckerInfo{}, errors.New("invalid HealthCheckerInfo Threshold, Threshold  must be greater than 1")
		}
		if healthChecker.Interval*healthChecker.Threshold > 2147483647 {
			return irs.HealthCheckerInfo{}, errors.New("invalid HealthCheckerInfo Interval * Threshold must be between 5 and 2147483647 ")
		}

		portIn32 := int32(port)
		intervalInSecondsInt32 := int32(healthChecker.Interval)
		thresholdInt32 := int32(healthChecker.Threshold)

		nlb.Properties.Probes[0].Properties.Protocol = &protocol
		nlb.Properties.Probes[0].Properties.Port = &portIn32
		nlb.Properties.Probes[0].Properties.IntervalInSeconds = &intervalInSecondsInt32
		nlb.Properties.Probes[0].Properties.NumberOfProbes = &thresholdInt32
		if protocol == armnetwork.ProbeProtocolHTTP || protocol == armnetwork.ProbeProtocolHTTPS {
			path := "/"
			nlb.Properties.Probes[0].Properties.RequestPath = &path
		} else {
			nlb.Properties.Probes[0].Properties.RequestPath = nil
		}
		poller, err := nlbHandler.NLBClient.BeginCreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.Region, *nlb.Name, *nlb, nil)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.HealthCheckerInfo{}, changeErr
		}
		_, err = poller.PollUntilDone(nlbHandler.Ctx, nil)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.HealthCheckerInfo{}, changeErr
		}
		nlb, err = nlbHandler.getRawNLB(nlbIID)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.HealthCheckerInfo{}, changeErr
		}
		info, err := nlbHandler.setterNLB(nlb)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.HealthCheckerInfo{}, changeErr
		}
		LoggingInfo(hiscallInfo, start)
		return info.HealthChecker, nil
	}
	changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = no exist Probe to Change"))
	cblogger.Error(changeErr.Error())
	LoggingError(hiscallInfo, changeErr)
	return irs.HealthCheckerInfo{}, changeErr
}

func (nlbHandler *AzureNLBHandler) getProbeMetricStatus(nlbId string, ip string) (bool, error) {
	aggregation := azquery.AggregationTypeAverage
	filter := fmt.Sprintf("BackendIPAddress eq '%s'", ip)
	metrics := make([]string, 0)
	metrics = append(metrics, "DipAvailability")
	metricNames := strings.Join(metrics, ",")
	endTime := time.Now().UTC()
	startTime := endTime.Add(time.Duration(-1) * time.Minute)
	resultType := azquery.ResultTypeData
	timespan := azquery.TimeInterval(fmt.Sprintf("%s/%s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339)))

	resp, err := nlbHandler.MetricClient.QueryResource(context.Background(), nlbId, &azquery.MetricsClientQueryResourceOptions{
		Aggregation:     []*azquery.AggregationType{&aggregation},
		Filter:          &filter,
		Interval:        nil,
		MetricNames:     &metricNames,
		MetricNamespace: nil,
		OrderBy:         nil,
		ResultType:      &resultType,
		Timespan:        &timespan,
		Top:             nil,
	})
	if err != nil {
		return false, err
	}

	if len(resp.Value) < 1 {
		return false, nil
	}
	values := resp.Value
	if values[0] == nil {
		return false, nil
	}
	if len((*(values[0])).TimeSeries) < 1 {
		return false, nil
	}
	TimeSeries := (*(values[0])).TimeSeries
	if len((*(TimeSeries[0])).Data) < 1 {
		return false, nil
	}
	data := TimeSeries[len((*(TimeSeries[0])).Data)-1].Data
	avg := int(*data[0].Average)
	if avg == 100 {
		return true, nil
	}
	return false, nil
}

type vmIP struct {
	VMIID irs.IID
	IP    string
}

func (nlbHandler *AzureNLBHandler) getVMIPs(nlbIId irs.IID) ([]vmIP, error) {
	rawnlb, err := nlbHandler.getRawNLB(nlbIId)
	if err != nil {
		return nil, err
	}
	info, err := nlbHandler.setterNLB(rawnlb)
	if err != nil {
		return nil, err
	}
	vmIPs := make([]vmIP, len(*info.VMGroup.VMs))
	for i, vmiid := range *info.VMGroup.VMs {
		resp, err := nlbHandler.VMClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, vmiid.NameId, nil)
		if err != nil {
			return nil, err
		}
		ip, err := nlbHandler.getPrivateIPByRawVm(&resp.VirtualMachine)
		if err != nil {
			return nil, err
		}
		vmIPs[i] = vmIP{
			IP:    ip,
			VMIID: vmiid,
		}
	}
	return vmIPs, nil
}
func (nlbHandler *AzureNLBHandler) getPrivateIPByRawVm(rawVm *armcompute.VirtualMachine) (string, error) {
	niList := rawVm.Properties.NetworkProfile.NetworkInterfaces
	var VNicId string
	for _, ni := range niList {
		if *ni.Properties.Primary && ni.ID != nil {
			VNicId = *ni.ID
		}
	}
	resp, err := nlbHandler.VNicClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, GetResourceNameById(VNicId), nil)
	if err != nil {
		return "", errors.New("not found IP")
	}
	for _, iPConfiguration := range resp.Interface.Properties.IPConfigurations {
		if *iPConfiguration.Properties.Primary {
			return *iPConfiguration.Properties.PrivateIPAddress, nil
		}
	}
	return "", errors.New("not found IP")
}
func (nlbHandler *AzureNLBHandler) NLBCleaner(nlbIID irs.IID) (bool, error) {
	// exist Check
	exist, err := nlbHandler.existNLB(nlbIID)
	if err != nil {
		return false, err
	}
	if !exist {
		// nlb not exist, check PublicIP related to nlb
		var publicIPList []*armnetwork.PublicIPAddress

		pager := nlbHandler.PublicIPClient.NewListPager(nlbHandler.Region.Region, nil)

		for pager.More() {
			page, err := pager.NextPage(nlbHandler.Ctx)
			if err != nil {
				return false, errors.New("nlb does not exist, but you have failed a publicIP query related to nlb")
			}

			for _, publicIP := range page.Value {
				publicIPList = append(publicIPList, publicIP)
			}
		}

		for _, pip := range publicIPList {
			// nlb not exist, exist PublicIP related to nlb
			if strings.EqualFold(*pip.Name, nlbIID.NameId) && pip.Tags["createdBy"] != nil && strings.EqualFold(*pip.Tags["createdBy"], nlbIID.NameId) {
				poller, err := nlbHandler.PublicIPClient.BeginDelete(nlbHandler.Ctx, nlbHandler.Region.Region, nlbIID.NameId, nil)
				if err != nil {
					return false, errors.New(fmt.Sprintf("not exist NLB, but failed To Delete PublicIP related to nlb err : %s", err.Error()))
				}
				_, err = poller.PollUntilDone(nlbHandler.Ctx, nil)
				if err != nil {
					return false, errors.New(fmt.Sprintf("not exist NLB, but failed To Delete PublicIP related to nlb err : %s", err.Error()))
				}
				return true, nil
			}
		}
		return true, nil
	}
	nlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		return false, err
	}
	publicIPName := ""
	if len(nlb.Properties.FrontendIPConfigurations) > 0 {
		FrontendIPConfigurations := nlb.Properties.FrontendIPConfigurations
		cbOnlyOneNLBFrontendIPConfigurations := FrontendIPConfigurations[0]
		if cbOnlyOneNLBFrontendIPConfigurations.Properties.PublicIPAddress != nil {
			publicIPName = GetResourceNameById(*cbOnlyOneNLBFrontendIPConfigurations.Properties.PublicIPAddress.ID)
		}
	}

	// Delete
	poller, err := nlbHandler.NLBClient.BeginDelete(nlbHandler.Ctx, nlbHandler.Region.Region, nlbIID.NameId, nil)
	if err != nil {
		return false, err
	}
	_, err = poller.PollUntilDone(nlbHandler.Ctx, nil)
	if err != nil {
		return false, err
	}
	if publicIPName != "" {
		poller, err := nlbHandler.PublicIPClient.BeginDelete(nlbHandler.Ctx, nlbHandler.Region.Region, publicIPName, nil)
		if err != nil {
			return false, errors.New(fmt.Sprintf("Failed To Delete PublicIP err : %s", err.Error()))
		}
		_, err = poller.PollUntilDone(nlbHandler.Ctx, nil)
		if err != nil {
			return false, errors.New(fmt.Sprintf("Failed To Delete PublicIP err : %s", err.Error()))
		}
	}
	return true, nil
}
func (nlbHandler *AzureNLBHandler) existNLB(nlbIID irs.IID) (bool, error) {
	// exist Check
	var nlbList []*armnetwork.LoadBalancer

	pager := nlbHandler.NLBClient.NewListPager(nlbHandler.Region.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(nlbHandler.Ctx)
		if err != nil {
			return false, err
		}

		for _, nlb := range page.Value {
			nlbList = append(nlbList, nlb)
		}
	}

	for _, nlb := range nlbList {
		if strings.EqualFold(*nlb.Name, nlbIID.NameId) {
			return true, nil
		}
	}
	return false, nil
}
func (nlbHandler *AzureNLBHandler) setterNLB(nlb *armnetwork.LoadBalancer) (*irs.NLBInfo, error) {
	nlbInfo := irs.NLBInfo{
		IId: irs.IID{
			NameId:   *nlb.Name,
			SystemId: *nlb.ID,
		},
	}

	if nlb.Tags["createdAt"] != nil {
		createAt := *nlb.Tags["createdAt"]
		timeInt64, err := strconv.ParseInt(createAt, 10, 64)
		if err == nil {
			nlbInfo.CreatedTime = time.Unix(timeInt64, 0)
		}
	}

	if len(nlb.Properties.BackendAddressPools) > 0 {
		// TODO: Deliver multiple backendPools in the future
		cbOnlyOneBackendPool := nlb.Properties.BackendAddressPools[0]
		vpcIId, err := nlbHandler.getVPCIIDByLoadBalancerBackendAddresses(cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses)
		if err == nil {
			nlbInfo.VpcIID = vpcIId
		}
	}

	vmGroup, listenerInfo, healthCheckerInfo, err := nlbHandler.getLoadBalancingRuleInfoByNLB(nlb)
	if err == nil {
		nlbInfo.VMGroup = vmGroup
		nlbInfo.HealthChecker = healthCheckerInfo
		nlbInfo.Listener = listenerInfo
	}

	nlbType, err := getNLBTypeByNLB(nlb)
	if err == nil {
		nlbInfo.Type = string(nlbType)
	}
	if *nlb.SKU.Tier == armnetwork.LoadBalancerSKUTierRegional {
		nlbInfo.Scope = string(NLBRegionType)
	} else {
		nlbInfo.Scope = string(NLBGlobalType)
	}

	if nlb.Tags != nil {
		nlbInfo.TagList = setTagList(nlb.Tags)
	}

	return &nlbInfo, nil
}
func (nlbHandler *AzureNLBHandler) getVPCIIDByLoadBalancerBackendAddresses(address []*armnetwork.LoadBalancerBackendAddress) (irs.IID, error) {
	for _, addr := range address {
		if addr.Properties.VirtualNetwork != nil {
			vpcName := GetResourceNameById(*addr.Properties.VirtualNetwork.ID)
			return irs.IID{NameId: vpcName, SystemId: *addr.Properties.VirtualNetwork.ID}, nil
		} else {
			nicNames, err := getNicNameByLoadBalancerBackendAddresses(address)
			if err != nil {
				return irs.IID{}, err
			}
			if len(nicNames) > 0 {
				nic, err := nlbHandler.getRawNic(irs.IID{NameId: nicNames[0]})
				if err != nil {
					return irs.IID{}, err
				}
				vpcIId, err := nlbHandler.getVPCIIdByNic(*nic)
				if err != nil {
					return irs.IID{}, err
				}
				return vpcIId, nil
			}
		}
	}
	return irs.IID{}, errors.New("not found vpc")
}
func (nlbHandler *AzureNLBHandler) getVMIIDsByLoadBalancerBackendAddresses(address []*armnetwork.LoadBalancerBackendAddress) ([]irs.IID, error) {
	vmIIds := make([]irs.IID, 0)
	if len(address) < 1 {
		return vmIIds, nil
	}
	refType, err := checkLoadBalancerBackendAddressesIPRefType(address)
	if err != nil {
		return nil, err
	}
	if refType == BackendAddressesIPAddressRef {
		var vmList []*armcompute.VirtualMachine

		pager := nlbHandler.VMClient.NewListPager(nlbHandler.Region.Region, nil)

		for pager.More() {
			page, err := pager.NextPage(nlbHandler.Ctx)
			if err != nil {
				return nil, err
			}

			for _, vm := range page.Value {
				vmList = append(vmList, vm)
			}
		}

		ips, err := getIpsByLoadBalancerBackendAddresses(address)
		if err != nil {
			return nil, err
		}
		for _, vm := range vmList {
			breakCheck := false
			niList := vm.Properties.NetworkProfile.NetworkInterfaces
			var VNicId string
			for _, ni := range niList {
				if *ni.Properties.Primary && ni.ID != nil {
					VNicId = *ni.ID
				}
			}
			resp, err := nlbHandler.VNicClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, GetResourceNameById(VNicId), nil)
			if err != nil {
				return nil, errors.New("not found VMIIDs")
			}
			for _, iPConfiguration := range resp.Interface.Properties.IPConfigurations {
				if *iPConfiguration.Properties.Primary {
					// PrivateIP 정보 설정
					for _, ip := range ips {
						if strings.EqualFold(ip, *iPConfiguration.Properties.PrivateIPAddress) {
							vmIIds = append(vmIIds, irs.IID{SystemId: *vm.ID, NameId: *vm.Name})
							breakCheck = true
							break
						}
					}
				}
				if breakCheck {
					break
				}
			}
		}
		return vmIIds, err
	} else {
		nicNames, err := getNicNameByLoadBalancerBackendAddresses(address)
		if err != nil {
			return nil, err
		}
		vmIIds, err := nlbHandler.getVMIIDsByNicNames(nicNames)
		if err != nil {
			return nil, err
		}
		return vmIIds, nil
	}
}
func (nlbHandler *AzureNLBHandler) getRawNLB(nlbIId irs.IID) (*armnetwork.LoadBalancer, error) {
	if nlbIId.NameId == "" {
		return nil, errors.New("invalid IID")
	}

	resp, err := nlbHandler.NLBClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, nlbIId.NameId, nil)
	if err != nil {
		return nil, err
	}

	return &resp.LoadBalancer, err
}
func (nlbHandler *AzureNLBHandler) getRawNic(nicIID irs.IID) (*armnetwork.Interface, error) {
	if nicIID.NameId == "" {
		return nil, errors.New("invalid IID")
	}

	resp, err := nlbHandler.VNicClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, nicIID.NameId, nil)
	if err != nil {
		return nil, err
	}

	return &resp.Interface, err
}
func (nlbHandler *AzureNLBHandler) getVPCIIdByNic(nic armnetwork.Interface) (irs.IID, error) {
	vpcName := ""
	for _, ipConfig := range nic.Properties.IPConfigurations {
		if *ipConfig.Properties.Primary {
			subnetId := *ipConfig.Properties.Subnet.ID
			idSplits := strings.Split(subnetId, "/")
			for i, str := range idSplits {
				if strings.EqualFold(str, "virtualNetworks") && len(idSplits)-1 > i {
					vpcName = idSplits[i+1]
				}
			}
			if vpcName != "" {
				return irs.IID{NameId: vpcName, SystemId: GetNetworksResourceIdByName(nlbHandler.CredentialInfo, nlbHandler.Region, AzureVirtualNetworks, vpcName)}, nil
			}
		}
	}
	return irs.IID{}, errors.New("invalid Nic")
}
func (nlbHandler *AzureNLBHandler) getVMIIDsByNicNames(nicNames []string) ([]irs.IID, error) {
	ids := make([]irs.IID, len(nicNames))
	for i, name := range nicNames {
		nic, err := nlbHandler.getRawNic(irs.IID{NameId: name})
		if err != nil {
			return nil, err
		}
		ids[i] = irs.IID{
			NameId:   GetResourceNameById(*nic.Properties.VirtualMachine.ID),
			SystemId: *nic.Properties.VirtualMachine.ID,
		}
	}
	return ids, nil
}
func checkLoadBalancerBackendAddressesIPRefType(address []*armnetwork.LoadBalancerBackendAddress) (BackendAddressesIPRefType, error) {
	for _, addr := range address {
		if addr.Properties.IPAddress != nil {
			return BackendAddressesIPAddressRef, nil
		}
		if addr.Properties.NetworkInterfaceIPConfiguration != nil {
			return BackendAddressesIPConfigurationRef, nil
		}
	}
	return "", errors.New("BackendAddressesIPRefType cannot be estimated")
}
func getIpsByLoadBalancerBackendAddresses(address []*armnetwork.LoadBalancerBackendAddress) ([]string, error) {
	ips := make([]string, 0, len(address))
	for _, addr := range address {
		if addr.Properties.IPAddress != nil {
			ips = append(ips, *addr.Properties.IPAddress)
		}
	}
	return ips, nil
}
func getNicNameByLoadBalancerBackendAddresses(address []*armnetwork.LoadBalancerBackendAddress) ([]string, error) {
	names := make([]string, len(address))
	for i, addr := range address {
		if addr.Properties.NetworkInterfaceIPConfiguration != nil {
			names[i] = GetResourceNameById(*addr.Properties.NetworkInterfaceIPConfiguration.ID)
		} else {
			return nil, errors.New("invalid LoadBalancerBackendAddresses")
		}
	}
	return names, nil
}
func (nlbHandler *AzureNLBHandler) getLoadBalancingRuleInfoByNLB(nlb *armnetwork.LoadBalancer) (irs.VMGroupInfo, irs.ListenerInfo, irs.HealthCheckerInfo, error) {
	LoadBalancingRules := nlb.Properties.LoadBalancingRules
	Probes := nlb.Properties.Probes
	if len(LoadBalancingRules) <= 0 {
		return irs.VMGroupInfo{}, irs.ListenerInfo{}, irs.HealthCheckerInfo{}, errors.New("invalid LoadBalancer")
	}
	frontendIP, err := nlbHandler.getFrontendIPByNLB(nlb)
	if err != nil {
		return irs.VMGroupInfo{}, irs.ListenerInfo{}, irs.HealthCheckerInfo{}, errors.New("invalid LoadBalancer")
	}
	cbOnlyOneLoadBalancingRule := LoadBalancingRules[0]
	VMGroup := irs.VMGroupInfo{
		Protocol: strings.ToUpper(string(*cbOnlyOneLoadBalancingRule.Properties.Protocol)),
		Port:     strconv.Itoa(int(*cbOnlyOneLoadBalancingRule.Properties.BackendPort)),
		// TODO: ?
		CspID: "",
	}
	listenerInfo := irs.ListenerInfo{
		Protocol: strings.ToUpper(string(*cbOnlyOneLoadBalancingRule.Properties.Protocol)),
		Port:     strconv.Itoa(int(*cbOnlyOneLoadBalancingRule.Properties.FrontendPort)),
		IP:       frontendIP,
		// TODO: ?
		CspID: "",
	}
	if len(nlb.Properties.BackendAddressPools) < 1 {
		return irs.VMGroupInfo{}, irs.ListenerInfo{}, irs.HealthCheckerInfo{}, errors.New("invalid LoadBalancer")
	}
	backendPools := nlb.Properties.BackendAddressPools

	cbOnlyOneBackendPool := backendPools[0]

	vmIId, err := nlbHandler.getVMIIDsByLoadBalancerBackendAddresses(cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses)
	if err == nil {
		VMGroup.VMs = &vmIId
	}
	healthCheckerInfo := irs.HealthCheckerInfo{}

	probeId := *cbOnlyOneLoadBalancingRule.Properties.Probe.ID
	for _, probe := range Probes {
		if *probe.ID == probeId {
			// Azure not support
			healthCheckerInfo.Timeout = -1
			healthCheckerInfo.Threshold = int(*probe.Properties.NumberOfProbes)
			healthCheckerInfo.Interval = int(*probe.Properties.IntervalInSeconds)
			healthCheckerInfo.Port = strconv.Itoa(int(*probe.Properties.Port))
			healthCheckerInfo.Protocol = strings.ToUpper(string(*probe.Properties.Protocol))
			break
		}
	}
	return VMGroup, listenerInfo, healthCheckerInfo, nil
}
func (nlbHandler *AzureNLBHandler) getFrontendIPByNLB(nlb *armnetwork.LoadBalancer) (string, error) {
	FrontendIPConfigurations := nlb.Properties.FrontendIPConfigurations
	if len(FrontendIPConfigurations) <= 0 {
		return "", errors.New("invalid LoadBalancer")
	}
	cbOnlyOneFrontendIPConfigurations := FrontendIPConfigurations[0]
	if cbOnlyOneFrontendIPConfigurations.Properties.PublicIPAddress != nil {
		publicName := GetResourceNameById(*cbOnlyOneFrontendIPConfigurations.Properties.PublicIPAddress.ID)
		rawPublicIP, err := nlbHandler.getRawPublicIP(irs.IID{NameId: publicName})
		if err == nil {
			return *rawPublicIP.Properties.IPAddress, nil
		}
	} else if cbOnlyOneFrontendIPConfigurations.Properties.PrivateIPAddress != nil {
		return *cbOnlyOneFrontendIPConfigurations.Properties.PrivateIPAddress, nil
	}
	return "", errors.New("invalid LoadBalancer")
}
func getNLBTypeByNLB(nlb *armnetwork.LoadBalancer) (NLBType, error) {
	FrontendIPConfigurations := nlb.Properties.FrontendIPConfigurations
	if len(FrontendIPConfigurations) <= 0 {
		return "", errors.New("invalid LoadBalancer")
	}
	cbOnlyOneFrontendIPConfigurations := FrontendIPConfigurations[0]
	if cbOnlyOneFrontendIPConfigurations.Properties.PublicIPAddress != nil {
		return NLBPublicType, nil
	} else if cbOnlyOneFrontendIPConfigurations.Properties.PrivateIPAddress != nil {
		return NLBInternalType, nil
	}
	return "", errors.New("invalid LoadBalancer")
}
func (nlbHandler *AzureNLBHandler) getRawPublicIP(publicIPIId irs.IID) (*armnetwork.PublicIPAddress, error) {
	if publicIPIId.NameId == "" {
		return nil, errors.New("invalid IID")
	}

	resp, err := nlbHandler.PublicIPClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, publicIPIId.NameId, nil)
	if err != nil {
		return nil, err
	}
	return &resp.PublicIPAddress, err
}
func getAzureLoadBalancingRuleByCBListenerInfo(listenerInfo irs.ListenerInfo, serviceGroupInfo irs.VMGroupInfo, frontEndIPConfigId string, backEndAddressPoolId string, probeId string) (*armnetwork.LoadBalancingRule, error) {
	protocol := armnetwork.TransportProtocolTCP
	switch strings.ToUpper(listenerInfo.Protocol) {
	case "TCP", "ALL", "UDP":
		protocol = armnetwork.TransportProtocol(strings.Title(strings.ToLower(listenerInfo.Protocol)))
	default:
		return nil, errors.New("invalid listenerInfo Protocol")
	}
	backendPort, err := strconv.Atoi(serviceGroupInfo.Port)
	if err != nil {
		return nil, errors.New("invalid serviceGroupInfo Protocol")
	}
	frontendPort, err := strconv.Atoi(listenerInfo.Port)
	if err != nil {
		return nil, errors.New("invalid listenerInfo Protocol")
	}

	loadBalancingRule := &armnetwork.LoadBalancingRule{
		Name: toStrPtr(generateRandName(LoadBalancingRulesPrefix)),
		Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
			Protocol:             &protocol,
			FrontendPort:         toInt32Ptr(frontendPort),
			BackendPort:          toInt32Ptr(backendPort),
			IdleTimeoutInMinutes: toInt32Ptr(4),
			EnableFloatingIP:     toBoolPtr(false),
			LoadDistribution:     (*armnetwork.LoadDistribution)(toStrPtr(string(armnetwork.LoadDistributionDefault))),
			FrontendIPConfiguration: &armnetwork.SubResource{
				ID: &frontEndIPConfigId,
			},
			BackendAddressPool: &armnetwork.SubResource{
				ID: &backEndAddressPoolId,
			},
			Probe: &armnetwork.SubResource{
				ID: &probeId,
			},
		},
	}
	if backEndAddressPoolId == "" {
		loadBalancingRule.Properties.BackendAddressPool = nil
	}
	return loadBalancingRule, nil
}
func getAzureProbeByCBHealthChecker(healthChecker irs.HealthCheckerInfo) (*armnetwork.Probe, error) {
	protocol := armnetwork.ProbeProtocolHTTP
	switch strings.ToUpper(healthChecker.Protocol) {
	case "HTTP", "HTTPS", "TCP":
		protocol = armnetwork.ProbeProtocol(strings.Title(strings.ToLower(healthChecker.Protocol)))
	default:
		return nil, errors.New("invalid HealthCheckerInfo Protocol")
	}
	port, err := strconv.Atoi(healthChecker.Port)
	if err != nil {
		return nil, errors.New("invalid HealthCheckerInfo Protocol")
	}

	if healthChecker.Interval < 5 {
		return nil, errors.New("invalid HealthCheckerInfo Interval, interval must be greater than 5")
	}
	if healthChecker.Threshold < 1 {
		return nil, errors.New("invalid HealthCheckerInfo Threshold, Threshold  must be greater than 1")
	}
	if healthChecker.Interval*healthChecker.Threshold > 2147483647 {
		return nil, errors.New("invalid HealthCheckerInfo Interval * Threshold must be between 5 and 2147483647 ")
	}

	probe := &armnetwork.Probe{
		Name: toStrPtr(generateRandName(ProbeNamePrefix)),
		Properties: &armnetwork.ProbePropertiesFormat{
			Protocol:          (*armnetwork.ProbeProtocol)(toStrPtr(string(protocol))),
			Port:              toInt32Ptr(port),
			IntervalInSeconds: toInt32Ptr(healthChecker.Interval),
			NumberOfProbes:    toInt32Ptr(healthChecker.Threshold),
		},
	}
	if protocol == armnetwork.ProbeProtocolHTTP || protocol == armnetwork.ProbeProtocolHTTPS {
		path := "/"
		probe.Properties.RequestPath = &path
	}
	return probe, nil
}
func getAzureFrontendIPConfiguration(publicIp *armnetwork.PublicIPAddress) *armnetwork.FrontendIPConfiguration {
	return &armnetwork.FrontendIPConfiguration{
		Name: toStrPtr(generateRandName(FrontEndIPConfigPrefix)),
		Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
			PrivateIPAllocationMethod: (*armnetwork.IPAllocationMethod)(toStrPtr(string(armnetwork.IPAllocationMethodDynamic))),
			PublicIPAddress:           publicIp,
		},
	}
}
func (nlbHandler *AzureNLBHandler) createPublicIP(nlbName string) (armnetwork.PublicIPAddress, error) {
	// PublicIP 이름 생성
	publicIPName := nlbName

	createOpts := armnetwork.PublicIPAddress{
		Name: &publicIPName,
		SKU: &armnetwork.PublicIPAddressSKU{
			Name: (*armnetwork.PublicIPAddressSKUName)(toStrPtr(string(armnetwork.PublicIPAddressSKUNameStandard))),
		},
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAddressVersion:   (*armnetwork.IPVersion)(toStrPtr(string(armnetwork.IPVersionIPv4))),
			PublicIPAllocationMethod: (*armnetwork.IPAllocationMethod)(toStrPtr(string(armnetwork.IPAllocationMethodStatic))),
			IdleTimeoutInMinutes:     toInt32Ptr(4),
		},
		Location: &nlbHandler.Region.Region,
		Tags: map[string]*string{
			"createdBy": &nlbName,
		},
	}

	poller, err := nlbHandler.PublicIPClient.BeginCreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.Region, publicIPName, createOpts, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create PublicIP, error=%s", err))
		return armnetwork.PublicIPAddress{}, createErr
	}
	_, err = poller.PollUntilDone(nlbHandler.Ctx, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create PublicIP, error=%s", err))
		return armnetwork.PublicIPAddress{}, createErr
	}

	// 생성된 PublicIP 정보 리턴
	resp, err := nlbHandler.PublicIPClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, publicIPName, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create PublicIP, error=%s", err))
		return armnetwork.PublicIPAddress{}, createErr
	}
	return resp.PublicIPAddress, nil
}
func (nlbHandler *AzureNLBHandler) getVPCNameSubnetNameAndPrivateIPByVM(server armcompute.VirtualMachine) (privateIP string, err error) {
	niList := server.Properties.NetworkProfile.NetworkInterfaces
	var VNicId string
	for _, ni := range niList {
		if ni.ID != nil {
			VNicId = *ni.ID
			break
		}
	}

	nicIdArr := strings.Split(VNicId, "/")
	nicName := nicIdArr[len(nicIdArr)-1]
	resp, err := nlbHandler.VNicClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, nicName, nil)
	if err != nil {
		return "", err
	}
	for _, ip := range resp.Interface.Properties.IPConfigurations {
		if *ip.Properties.Primary {
			privateIP := *ip.Properties.PrivateIPAddress
			return privateIP, nil
		}
	}
	return "", errors.New("not found subnet")
}
func (nlbHandler *AzureNLBHandler) getVPCIIDByVM(server armcompute.VirtualMachine) (irs.IID, error) {
	niList := server.Properties.NetworkProfile.NetworkInterfaces
	var VNicId string
	for _, ni := range niList {
		if ni.ID != nil {
			VNicId = *ni.ID
			break
		}
	}

	nicIdArr := strings.Split(VNicId, "/")
	nicName := nicIdArr[len(nicIdArr)-1]
	resp, err := nlbHandler.VNicClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, nicName, nil)
	if err != nil {
		return irs.IID{}, err
	}
	for _, ip := range resp.Interface.Properties.IPConfigurations {
		if *ip.Properties.Primary {
			subnetIdArr := strings.Split(*ip.Properties.Subnet.ID, "/")
			vpcId := strings.Join(subnetIdArr[:len(subnetIdArr)-2], "/")
			return irs.IID{
				NameId:   GetResourceNameById(vpcId),
				SystemId: vpcId,
			}, nil
		}
	}
	return irs.IID{}, errors.New("not found subnet")
}
func (nlbHandler *AzureNLBHandler) getLoadBalancerBackendAddress(backEndAddressPoolName string, vpcId string, privateIP string) (*armnetwork.LoadBalancerBackendAddress, error) {
	return &armnetwork.LoadBalancerBackendAddress{
		Properties: &armnetwork.LoadBalancerBackendAddressPropertiesFormat{
			VirtualNetwork: &armnetwork.SubResource{
				ID: &vpcId,
			},
			IPAddress: &privateIP,
		},
		Name: toStrPtr(backEndAddressPoolName + privateIP),
	}, nil
}
func (nlbHandler *AzureNLBHandler) getNLBType(nlb armnetwork.LoadBalancer) (string, error) {
	FrontendIPConfigurations := nlb.Properties.FrontendIPConfigurations
	if len(FrontendIPConfigurations) <= 0 {
		return "", errors.New("invalid LoadBalancer")
	}
	cbOnlyOneFrontendIPConfigurations := FrontendIPConfigurations[0]
	if cbOnlyOneFrontendIPConfigurations.Properties.PublicIPAddress.ID == nil {
		return "INTERNAL", nil
	} else {
		return "PUBLIC", nil
	}
}
func (nlbHandler *AzureNLBHandler) getPrivateIPBYNicName(nicName string) (string, error) {
	resp, err := nlbHandler.VNicClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, nicName, nil)
	if err != nil {
		return "", err
	}
	for _, ip := range resp.Interface.Properties.IPConfigurations {
		if *ip.Properties.Primary {
			privateIP := *ip.Properties.PrivateIPAddress
			return privateIP, nil
		}
	}
	return "", errors.New("Not exist IP")
}
func convertListenerInfoProtocolsToInboundRuleProtocol(protocol string) (armnetwork.TransportProtocol, error) {
	switch strings.ToUpper(protocol) {
	case "TCP":
		return armnetwork.TransportProtocolTCP, nil
	case "UDP":
		return armnetwork.TransportProtocolUDP, nil
	}
	return "", errors.New("invalid Protocols")
}

func checkValidationNLB(nlbReqInfo irs.NLBInfo) error {
	err := checkValidationNLBHealthCheck(nlbReqInfo.HealthChecker)
	return err
}

func checkValidationNLBHealthCheck(healthCheckerInfo irs.HealthCheckerInfo) error {
	// Not -1
	if healthCheckerInfo.Timeout != -1 {
		return errors.New(fmt.Sprintf("Azure NLB does not support timeout."))
	}
	if healthCheckerInfo.Interval < 5 {
		return errors.New("invalid HealthCheckerInfo Interval, interval must be greater than 5")
	}
	if healthCheckerInfo.Threshold < 1 {
		return errors.New("invalid HealthCheckerInfo Threshold, Threshold  must be greater than 1")
	}
	if healthCheckerInfo.Interval*healthCheckerInfo.Threshold > 2147483647 {
		return errors.New("invalid HealthCheckerInfo Interval * Threshold must be between 5 and 2147483647 ")
	}
	return nil
}
