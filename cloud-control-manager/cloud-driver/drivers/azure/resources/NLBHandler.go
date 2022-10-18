package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strconv"
	"strings"
	"time"
)

type AzureNLBHandler struct {
	CredentialInfo               idrv.CredentialInfo
	Region                       idrv.RegionInfo
	Ctx                          context.Context
	NLBClient                    *network.LoadBalancersClient
	NLBBackendAddressPoolsClient *network.LoadBalancerBackendAddressPoolsClient
	VNicClient                   *network.InterfacesClient
	PublicIPClient               *network.PublicIPAddressesClient
	VMClient                     *compute.VirtualMachinesClient
	SubnetClient                 *network.SubnetsClient
	IPConfigClient               *network.InterfaceIPConfigurationsClient
	NLBLoadBalancingRulesClient  *network.LoadBalancerLoadBalancingRulesClient
	MetricClient                 *insights.MetricsClient
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
		}
	}()

	// Create NLB PublicIP (NLB EndPoint)
	var frontendIPConfigurations []network.FrontendIPConfiguration
	frontendIPConfiguration, err := getAzureFrontendIPConfiguration(&publicIp)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError)
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}
	frontendIPConfigurations = append(frontendIPConfigurations, frontendIPConfiguration)

	// Create healthCheckProbe (BackendPool VM HealthCheck)
	var Probes []network.Probe
	healthCheckProbe, err := getAzureProbeByCBHealthChecker(nlbReqInfo.HealthChecker)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError)
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}
	Probes = append(Probes, healthCheckProbe)

	var BackendAddressPools []network.BackendAddressPool
	var loadBalancingRules []network.LoadBalancingRule
	backEndAddressPoolName := generateRandName(BackEndAddressPoolPrefix)
	// Create BackendAddressPools (Front => Backend)
	BackendAddressPools = append(BackendAddressPools, network.BackendAddressPool{Name: to.StringPtr(backEndAddressPoolName)})

	// Create Related ID for Create loadBalancingRules (Front => Backend)
	nlbId := GetNetworksResourceIdByName(nlbHandler.CredentialInfo, nlbHandler.Region, AzureLoadBalancers, nlbReqInfo.IId.NameId)
	frontEndIPConfigId := fmt.Sprintf("%s/frontendIPConfigurations/%s", nlbId, *frontendIPConfiguration.Name)
	backEndAddressPoolId := fmt.Sprintf("%s/backendAddressPools/%s", nlbId, backEndAddressPoolName)
	if len(*nlbReqInfo.VMGroup.VMs) == 0 {
		backEndAddressPoolId = ""
	}
	probeId := fmt.Sprintf("%s/probes/%s", nlbId, *healthCheckProbe.Name)

	// Create loadBalancingRules (Front => Backend)
	var loadBalancingRule network.LoadBalancingRule
	loadBalancingRule, err = getAzureLoadBalancingRuleByCBListenerInfo(nlbReqInfo.Listener, nlbReqInfo.VMGroup, frontEndIPConfigId, backEndAddressPoolId, probeId)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError)
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}

	nowTime := strconv.FormatInt(time.Now().Unix(), 10)
	loadBalancingRules = append(loadBalancingRules, loadBalancingRule)
	options := network.LoadBalancer{
		Location: to.StringPtr(nlbHandler.Region.Region),
		Sku: &network.LoadBalancerSku{
			Name: network.LoadBalancerSkuNameStandard,
			Tier: network.LoadBalancerSkuTierRegional,
		},
		LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
			// TODO: Deliver multiple FrontendIPConfigurations, BackendAddressPools, Probes, loadBalancingRules in the future
			FrontendIPConfigurations: &frontendIPConfigurations,
			BackendAddressPools:      &BackendAddressPools,
			Probes:                   &Probes,
			LoadBalancingRules:       &loadBalancingRules,
		},
		Tags: map[string]*string{
			"createdAt": to.StringPtr(nowTime),
		},
	}
	future, err := nlbHandler.NLBClient.CreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, nlbReqInfo.IId.NameId, options)
	if err != nil {
		createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
		cblogger.Error(createError)
		LoggingError(hiscallInfo, createError)
		return irs.NLBInfo{}, createError
	}
	err = future.WaitForCompletionRef(nlbHandler.Ctx, nlbHandler.NLBClient.Client)
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
			//vm, err := GetRawVM(vmIId, nlbHandler.Region.ResourceGroup, nlbHandler.VMClient, nlbHandler.Ctx)
			vm, err := GetRawVM(convertedIID, nlbHandler.Region.ResourceGroup, nlbHandler.VMClient, nlbHandler.Ctx)
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

		pool, err := nlbHandler.NLBBackendAddressPoolsClient.Get(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, nlbReqInfo.IId.NameId, backEndAddressPoolName)
		if err != nil {
			createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
			cblogger.Error(createError)
			LoggingError(hiscallInfo, createError)
			return irs.NLBInfo{}, createError
		}
		LoadBalancerBackendAddresses := *pool.LoadBalancerBackendAddresses

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

		pool.LoadBalancerBackendAddresses = &LoadBalancerBackendAddresses

		backendAddressPoolFuture, err := nlbHandler.NLBBackendAddressPoolsClient.CreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, nlbReqInfo.IId.NameId, backEndAddressPoolName, pool)
		if err != nil {
			createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
			cblogger.Error(createError)
			LoggingError(hiscallInfo, createError)
			return irs.NLBInfo{}, createError
		}
		err = backendAddressPoolFuture.WaitForCompletionRef(nlbHandler.Ctx, nlbHandler.NLBBackendAddressPoolsClient.Client)
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
	info, err := nlbHandler.setterNLB(*rawNLB)
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

	rawNLBList, err := nlbHandler.NLBClient.List(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List NLB. err = %s", err.Error()))
		cblogger.Error(getErr)
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	nlbInfoList := make([]*irs.NLBInfo, len(rawNLBList.Values()))

	for i, rawNLB := range rawNLBList.Values() {
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

	info, err := nlbHandler.setterNLB(*rawNLB)
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

	loadBalancingRules := *nlb.LoadBalancingRules
	if len(loadBalancingRules) < 1 {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = not Exist Listener"))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}

	cbOnlyOneLoadBalancingRule := &loadBalancingRules[0]
	cbOnlyOneLoadBalancingRule.Protocol = protocol
	cbOnlyOneLoadBalancingRule.FrontendPort = to.Int32Ptr(int32(frontendPort))
	nlb.LoadBalancingRules = &loadBalancingRules

	future, err := nlbHandler.NLBClient.CreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, *nlb.Name, *nlb)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeListener NLB. err = %s", err.Error()))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.ListenerInfo{}, changeErr
	}
	err = future.WaitForCompletionRef(nlbHandler.Ctx, nlbHandler.NLBClient.Client)
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
	info, err := nlbHandler.setterNLB(*rawNLB)
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
	loadBalancingRules := *nlb.LoadBalancingRules
	if len(loadBalancingRules) < 1 {
		return irs.VMGroupInfo{}, errors.New("not Exist Listener")
	}

	cbOnlyOneLoadBalancingRule := &loadBalancingRules[0]
	cbOnlyOneLoadBalancingRule.BackendPort = to.Int32Ptr(int32(backendPort))
	nlb.LoadBalancingRules = &loadBalancingRules

	future, err := nlbHandler.NLBClient.CreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, *nlb.Name, *nlb)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeVMGroupInfo NLB. err = %s", err.Error()))
		cblogger.Error(changeErr)
		LoggingError(hiscallInfo, changeErr)
		return irs.VMGroupInfo{}, changeErr
	}
	err = future.WaitForCompletionRef(nlbHandler.Ctx, nlbHandler.NLBClient.Client)
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
	info, err := nlbHandler.setterNLB(*nlb)
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

	if len(*nlb.BackendAddressPools) > 0 && len(*vmIIDs) > 0 {
		backendPools := *nlb.BackendAddressPools
		cbOnlyOneBackendPool := backendPools[0]

		nlbCurrentVMIIds, err := nlbHandler.getVMIIDsByLoadBalancerBackendAddresses(*cbOnlyOneBackendPool.LoadBalancerBackendAddresses)
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
			vm, err := GetRawVM(vmIId, nlbHandler.Region.ResourceGroup, nlbHandler.VMClient, nlbHandler.Ctx)
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
		if cbOnlyOneBackendPool.LoadBalancerBackendAddresses != nil && len(*cbOnlyOneBackendPool.LoadBalancerBackendAddresses) > 0 {
			vpcIId, err := nlbHandler.getVPCIIDByLoadBalancerBackendAddresses(*cbOnlyOneBackendPool.LoadBalancerBackendAddresses)
			if err != nil {
				addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
				cblogger.Error(addErr.Error())
				LoggingError(hiscallInfo, addErr)
				return irs.VMGroupInfo{}, addErr
			}
			vpcId = vpcIId.SystemId
		} else {
			for _, vmIId := range *vmIIDs {
				vm, err := GetRawVM(vmIId, nlbHandler.Region.ResourceGroup, nlbHandler.VMClient, nlbHandler.Ctx)
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

		pool, err := nlbHandler.NLBBackendAddressPoolsClient.Get(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, nlbIID.NameId, backendPoolName)
		if err != nil {
			addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
			cblogger.Error(addErr.Error())
			LoggingError(hiscallInfo, addErr)
			return irs.VMGroupInfo{}, addErr
		}
		LoadBalancerBackendAddresses := *pool.LoadBalancerBackendAddresses

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

		pool.LoadBalancerBackendAddresses = &LoadBalancerBackendAddresses

		backendAddressPoolFuture, err := nlbHandler.NLBBackendAddressPoolsClient.CreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, nlbIID.NameId, backendPoolName, pool)
		if err != nil {
			addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
			cblogger.Error(addErr.Error())
			LoggingError(hiscallInfo, addErr)
			return irs.VMGroupInfo{}, addErr
		}
		err = backendAddressPoolFuture.WaitForCompletionRef(nlbHandler.Ctx, nlbHandler.NLBBackendAddressPoolsClient.Client)
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
	info, err := nlbHandler.setterNLB(*nlb)
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

	if len(*nlb.BackendAddressPools) > 0 && len(*vmIIDs) > 0 {
		backendPools := *nlb.BackendAddressPools
		cbOnlyOneBackendPool := backendPools[0]

		nlbCurrentVMIIds, err := nlbHandler.getVMIIDsByLoadBalancerBackendAddresses(*cbOnlyOneBackendPool.LoadBalancerBackendAddresses)

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
			vm, err := GetRawVM(vmIId, nlbHandler.Region.ResourceGroup, nlbHandler.VMClient, nlbHandler.Ctx)
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

		pool, err := nlbHandler.NLBBackendAddressPoolsClient.Get(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, nlbIID.NameId, backendPoolName)
		if err != nil {
			removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
			cblogger.Error(removeErr.Error())
			LoggingError(hiscallInfo, removeErr)
			return false, removeErr
		}

		newLoadBalancerBackendAddresses := make([]network.LoadBalancerBackendAddress, 0)
		currentVMIIds, err := nlbHandler.getVMIIDsByLoadBalancerBackendAddresses(*pool.LoadBalancerBackendAddresses)
		if err != nil {
			removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
			cblogger.Error(removeErr.Error())
			LoggingError(hiscallInfo, removeErr)
			return false, removeErr
		}
		currentVMPrivateIPs := make([]string, len(currentVMIIds))

		for i, vmIId := range currentVMIIds {
			vm, err := GetRawVM(vmIId, nlbHandler.Region.ResourceGroup, nlbHandler.VMClient, nlbHandler.Ctx)
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

		vpcIId, err := nlbHandler.getVPCIIDByLoadBalancerBackendAddresses(*cbOnlyOneBackendPool.LoadBalancerBackendAddresses)
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
		pool.LoadBalancerBackendAddresses = &newLoadBalancerBackendAddresses

		backendAddressPoolFuture, err := nlbHandler.NLBBackendAddressPoolsClient.CreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, nlbIID.NameId, backendPoolName, pool)
		if err != nil {
			removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
			cblogger.Error(removeErr.Error())
			LoggingError(hiscallInfo, removeErr)
			return false, removeErr
		}
		err = backendAddressPoolFuture.WaitForCompletionRef(nlbHandler.Ctx, nlbHandler.NLBBackendAddressPoolsClient.Client)
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
	currentProbes := *nlb.Probes
	if len(currentProbes) > 0 {
		protocol := network.ProbeProtocolHTTP
		switch strings.ToUpper(healthChecker.Protocol) {
		case "HTTP", "HTTPS", "TCP":
			protocol = network.ProbeProtocol(strings.Title(strings.ToLower(healthChecker.Protocol)))
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
		currentProbes[0].Protocol = protocol
		currentProbes[0].Port = to.Int32Ptr(int32(port))
		currentProbes[0].IntervalInSeconds = to.Int32Ptr(int32(healthChecker.Interval))
		currentProbes[0].NumberOfProbes = to.Int32Ptr(int32(healthChecker.Threshold))
		if protocol == network.ProbeProtocolHTTP || protocol == network.ProbeProtocolHTTPS {
			currentProbes[0].RequestPath = to.StringPtr("/")
		} else {
			currentProbes[0].RequestPath = nil
		}
		nlb.Probes = &currentProbes
		future, err := nlbHandler.NLBClient.CreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, *nlb.Name, *nlb)
		if err != nil {
			changeErr := errors.New(fmt.Sprintf("Failed to ChangeHealthCheckerInfo NLB. err = %s", err.Error()))
			cblogger.Error(changeErr.Error())
			LoggingError(hiscallInfo, changeErr)
			return irs.HealthCheckerInfo{}, changeErr
		}
		err = future.WaitForCompletionRef(nlbHandler.Ctx, nlbHandler.NLBClient.Client)
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
		info, err := nlbHandler.setterNLB(*nlb)
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
	endTime := time.Now().UTC()
	startTime := endTime.Add(time.Duration(-1) * time.Minute)
	timespan := fmt.Sprintf("%s/%s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	metrics := make([]string, 0)
	metrics = append(metrics, "DipAvailability")
	filter := fmt.Sprintf("BackendIPAddress eq '%s'", ip)
	resp, err := nlbHandler.MetricClient.List(context.Background(), nlbId, timespan, nil, strings.Join(metrics, ","), "average", nil, "", filter, insights.Data, "")
	if err != nil {
		return false, err
	}
	if resp.Value == nil || len(*resp.Value) < 1 {
		return false, nil
	}
	values := *resp.Value
	if values[0].Timeseries == nil || len(*values[0].Timeseries) < 1 {
		return false, nil
	}
	Timeseries := *values[0].Timeseries
	if Timeseries[0].Data == nil || len(*Timeseries[0].Data) < 1 {
		return false, nil
	}
	data := *Timeseries[len(*Timeseries[0].Data)-1].Data
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
	info, err := nlbHandler.setterNLB(*rawnlb)
	if err != nil {
		return nil, err
	}
	vmIPs := make([]vmIP, len(*info.VMGroup.VMs))
	for i, vmiid := range *info.VMGroup.VMs {
		rawvm, err := nlbHandler.VMClient.Get(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, vmiid.NameId, "")
		if err != nil {
			return nil, err
		}
		ip, err := nlbHandler.getPrivateIPByRawVm(rawvm)
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
func (nlbHandler *AzureNLBHandler) getPrivateIPByRawVm(rawVm compute.VirtualMachine) (string, error) {
	niList := *rawVm.NetworkProfile.NetworkInterfaces
	var VNicId string
	for _, ni := range niList {
		if *ni.Primary && ni.ID != nil {
			VNicId = *ni.ID
		}
	}
	rawVnic, err := nlbHandler.VNicClient.Get(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, GetResourceNameById(VNicId), "")
	if err != nil {
		return "", errors.New("not found IP")
	}
	for _, iPConfiguration := range *rawVnic.IPConfigurations {
		if *iPConfiguration.Primary {
			return *iPConfiguration.PrivateIPAddress, nil
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
		ips, err := nlbHandler.PublicIPClient.List(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup)
		if err != nil {
			return false, errors.New("nlb does not exist, but you have failed a publicIP query related to nlb")
		}
		for _, pip := range ips.Values() {
			// nlb not exist, exist PublicIP related to nlb
			if strings.EqualFold(*pip.Name, nlbIID.NameId) && pip.Tags["createdBy"] != nil && strings.EqualFold(*pip.Tags["createdBy"], nlbIID.NameId) {
				publicIPFeature, err := nlbHandler.PublicIPClient.Delete(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, nlbIID.NameId)
				if err != nil {
					return false, errors.New(fmt.Sprintf("not exist NLB, but failed To Delete PublicIP related to nlb err : %s", err.Error()))
				}
				err = publicIPFeature.WaitForCompletionRef(nlbHandler.Ctx, nlbHandler.PublicIPClient.Client)
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
	if nlb.FrontendIPConfigurations != nil && len(*nlb.FrontendIPConfigurations) > 0 {
		FrontendIPConfigurations := *nlb.FrontendIPConfigurations
		cbOnlyOneNLBFrontendIPConfigurations := FrontendIPConfigurations[0]
		if cbOnlyOneNLBFrontendIPConfigurations.PublicIPAddress != nil {
			publicIPName = GetResourceNameById(*cbOnlyOneNLBFrontendIPConfigurations.PublicIPAddress.ID)
		}
	}

	// Delete
	feature, err := nlbHandler.NLBClient.Delete(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, nlbIID.NameId)
	if err != nil {
		return false, err
	}
	err = feature.WaitForCompletionRef(nlbHandler.Ctx, nlbHandler.NLBClient.Client)
	if err != nil {
		return false, err
	}
	if publicIPName != "" {
		publicIPFeature, err := nlbHandler.PublicIPClient.Delete(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, publicIPName)
		if err != nil {
			return false, errors.New(fmt.Sprintf("Failed To Delete PublicIP err : %s", err.Error()))
		}
		err = publicIPFeature.WaitForCompletionRef(nlbHandler.Ctx, nlbHandler.PublicIPClient.Client)
		if err != nil {
			return false, errors.New(fmt.Sprintf("Failed To Delete PublicIP err : %s", err.Error()))
		}
	}
	return true, nil
}
func (nlbHandler *AzureNLBHandler) existNLB(nlbIID irs.IID) (bool, error) {
	// exist Check
	rawNLBList, err := nlbHandler.NLBClient.List(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup)
	if err != nil {
		return false, err
	}
	for _, nlb := range rawNLBList.Values() {
		if strings.EqualFold(*nlb.Name, nlbIID.NameId) {
			return true, nil
		}
	}
	return false, nil
}
func (nlbHandler *AzureNLBHandler) setterNLB(nlb network.LoadBalancer) (*irs.NLBInfo, error) {
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

	if len(*nlb.BackendAddressPools) > 0 {
		backendPools := *nlb.BackendAddressPools
		// TODO: Deliver multiple backendPools in the future
		cbOnlyOneBackendPool := backendPools[0]
		vpcIId, err := nlbHandler.getVPCIIDByLoadBalancerBackendAddresses(*cbOnlyOneBackendPool.LoadBalancerBackendAddresses)
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
	if nlb.Sku.Tier == network.LoadBalancerSkuTierRegional {
		nlbInfo.Scope = string(NLBRegionType)
	} else {
		nlbInfo.Scope = string(NLBGlobalType)
	}
	return &nlbInfo, nil
}
func (nlbHandler *AzureNLBHandler) getVPCIIDByLoadBalancerBackendAddresses(address []network.LoadBalancerBackendAddress) (irs.IID, error) {
	for _, addr := range address {
		if addr.VirtualNetwork != nil {
			vpcName := GetResourceNameById(*addr.VirtualNetwork.ID)
			return irs.IID{NameId: vpcName, SystemId: *addr.VirtualNetwork.ID}, nil
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
func (nlbHandler *AzureNLBHandler) getVMIIDsByLoadBalancerBackendAddresses(address []network.LoadBalancerBackendAddress) ([]irs.IID, error) {
	vmIIds := make([]irs.IID, 0)
	if len(address) < 1 {
		return vmIIds, nil
	}
	refType, err := checkLoadBalancerBackendAddressesIPRefType(address)
	if err != nil {
		return nil, err
	}
	if refType == BackendAddressesIPAddressRef {
		allVMS, err := nlbHandler.VMClient.List(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup)
		if err != nil {
			return nil, err
		}
		ips, err := getIpsByLoadBalancerBackendAddresses(address)
		if err != nil {
			return nil, err
		}
		for _, vm := range allVMS.Values() {
			breakCheck := false
			niList := *vm.NetworkProfile.NetworkInterfaces
			var VNicId string
			for _, ni := range niList {
				if *ni.Primary && ni.ID != nil {
					VNicId = *ni.ID
				}
			}
			rawVnic, err := nlbHandler.VNicClient.Get(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, GetResourceNameById(VNicId), "")
			if err != nil {
				return nil, errors.New("not found VMIIDs")
			}
			for _, iPConfiguration := range *rawVnic.IPConfigurations {
				if *iPConfiguration.Primary {
					// PrivateIP 정보 설정
					for _, ip := range ips {
						if strings.EqualFold(ip, *iPConfiguration.PrivateIPAddress) {
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
func (nlbHandler *AzureNLBHandler) getRawNLB(nlbIId irs.IID) (*network.LoadBalancer, error) {
	if nlbIId.NameId == "" {
		return nil, errors.New("invalid IID")
	}

	nlb, err := nlbHandler.NLBClient.Get(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, nlbIId.NameId, "")
	return &nlb, err
}
func (nlbHandler *AzureNLBHandler) getRawNic(nicIID irs.IID) (*network.Interface, error) {
	if nicIID.NameId == "" {
		return nil, errors.New("invalid IID")
	}
	nic, err := nlbHandler.VNicClient.Get(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, nicIID.NameId, "")
	return &nic, err
}
func (nlbHandler *AzureNLBHandler) getVPCIIdByNic(nic network.Interface) (irs.IID, error) {
	vpcName := ""
	for _, ipConfig := range *nic.IPConfigurations {
		if *ipConfig.Primary {
			subnetId := *ipConfig.Subnet.ID
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
			NameId:   GetResourceNameById(*nic.VirtualMachine.ID),
			SystemId: *nic.VirtualMachine.ID,
		}
	}
	return ids, nil
}
func checkLoadBalancerBackendAddressesIPRefType(address []network.LoadBalancerBackendAddress) (BackendAddressesIPRefType, error) {
	for _, addr := range address {
		if addr.IPAddress != nil {
			return BackendAddressesIPAddressRef, nil
		}
		if addr.NetworkInterfaceIPConfiguration != nil {
			return BackendAddressesIPConfigurationRef, nil
		}
	}
	return "", errors.New("BackendAddressesIPRefType cannot be estimated")
}
func getIpsByLoadBalancerBackendAddresses(address []network.LoadBalancerBackendAddress) ([]string, error) {
	ips := make([]string, 0, len(address))
	for _, addr := range address {
		if addr.IPAddress != nil {
			ips = append(ips, *addr.IPAddress)
		}
	}
	return ips, nil
}
func getNicNameByLoadBalancerBackendAddresses(address []network.LoadBalancerBackendAddress) ([]string, error) {
	names := make([]string, len(address))
	for i, addr := range address {
		if addr.NetworkInterfaceIPConfiguration != nil {
			names[i] = GetResourceNameById(*addr.NetworkInterfaceIPConfiguration.ID)
		} else {
			return nil, errors.New("invalid LoadBalancerBackendAddresses")
		}
	}
	return names, nil
}
func (nlbHandler *AzureNLBHandler) getLoadBalancingRuleInfoByNLB(nlb network.LoadBalancer) (irs.VMGroupInfo, irs.ListenerInfo, irs.HealthCheckerInfo, error) {
	LoadBalancingRules := *nlb.LoadBalancingRules
	Probes := *nlb.Probes
	if len(LoadBalancingRules) <= 0 {
		return irs.VMGroupInfo{}, irs.ListenerInfo{}, irs.HealthCheckerInfo{}, errors.New("invalid LoadBalancer")
	}
	frontendIP, err := nlbHandler.getFrontendIPByNLB(nlb)
	if err != nil {
		return irs.VMGroupInfo{}, irs.ListenerInfo{}, irs.HealthCheckerInfo{}, errors.New("invalid LoadBalancer")
	}
	cbOnlyOneLoadBalancingRule := LoadBalancingRules[0]
	VMGroup := irs.VMGroupInfo{
		Protocol: strings.ToUpper(string(cbOnlyOneLoadBalancingRule.Protocol)),
		Port:     strconv.Itoa(int(*cbOnlyOneLoadBalancingRule.BackendPort)),
		// TODO: ?
		CspID: "",
	}
	listenerInfo := irs.ListenerInfo{
		Protocol: strings.ToUpper(string(cbOnlyOneLoadBalancingRule.Protocol)),
		Port:     strconv.Itoa(int(*cbOnlyOneLoadBalancingRule.FrontendPort)),
		IP:       frontendIP,
		// TODO: ?
		CspID: "",
	}
	if nlb.BackendAddressPools == nil || len(*nlb.BackendAddressPools) < 1 {
		return irs.VMGroupInfo{}, irs.ListenerInfo{}, irs.HealthCheckerInfo{}, errors.New("invalid LoadBalancer")
	}
	backendPools := *nlb.BackendAddressPools

	cbOnlyOneBackendPool := backendPools[0]

	vmIId, err := nlbHandler.getVMIIDsByLoadBalancerBackendAddresses(*cbOnlyOneBackendPool.LoadBalancerBackendAddresses)
	if err == nil {
		VMGroup.VMs = &vmIId
	}
	healthCheckerInfo := irs.HealthCheckerInfo{}

	probeId := *cbOnlyOneLoadBalancingRule.Probe.ID
	for _, probe := range Probes {
		if *probe.ID == probeId {
			// Azure not support
			healthCheckerInfo.Timeout = -1
			healthCheckerInfo.Threshold = int(*probe.NumberOfProbes)
			healthCheckerInfo.Interval = int(*probe.IntervalInSeconds)
			healthCheckerInfo.Port = strconv.Itoa(int(*probe.Port))
			healthCheckerInfo.Protocol = strings.ToUpper(string(probe.Protocol))
			break
		}
	}
	return VMGroup, listenerInfo, healthCheckerInfo, nil
}
func (nlbHandler *AzureNLBHandler) getFrontendIPByNLB(nlb network.LoadBalancer) (string, error) {
	FrontendIPConfigurations := *nlb.FrontendIPConfigurations
	if len(FrontendIPConfigurations) <= 0 {
		return "", errors.New("invalid LoadBalancer")
	}
	cbOnlyOneFrontendIPConfigurations := FrontendIPConfigurations[0]
	if cbOnlyOneFrontendIPConfigurations.PublicIPAddress != nil {
		publicName := GetResourceNameById(*cbOnlyOneFrontendIPConfigurations.PublicIPAddress.ID)
		rawPublicIP, err := nlbHandler.getRawPublicIP(irs.IID{NameId: publicName})
		if err == nil {
			return *rawPublicIP.IPAddress, nil
		}
	} else if cbOnlyOneFrontendIPConfigurations.PrivateIPAddress != nil {
		return *cbOnlyOneFrontendIPConfigurations.PrivateIPAddress, nil
	}
	return "", errors.New("invalid LoadBalancer")
}
func getNLBTypeByNLB(nlb network.LoadBalancer) (NLBType, error) {
	FrontendIPConfigurations := *nlb.FrontendIPConfigurations
	if len(FrontendIPConfigurations) <= 0 {
		return "", errors.New("invalid LoadBalancer")
	}
	cbOnlyOneFrontendIPConfigurations := FrontendIPConfigurations[0]
	if cbOnlyOneFrontendIPConfigurations.PublicIPAddress != nil {
		return NLBPublicType, nil
	} else if cbOnlyOneFrontendIPConfigurations.PrivateIPAddress != nil {
		return NLBInternalType, nil
	}
	return "", errors.New("invalid LoadBalancer")
}
func (nlbHandler *AzureNLBHandler) getRawPublicIP(publicIPIId irs.IID) (*network.PublicIPAddress, error) {
	if publicIPIId.NameId == "" {
		return nil, errors.New("invalid IID")
	}

	pIP, err := nlbHandler.PublicIPClient.Get(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, publicIPIId.NameId, "")
	if err != nil {
		return nil, err
	}
	return &pIP, err
}
func getAzureLoadBalancingRuleByCBListenerInfo(listenerInfo irs.ListenerInfo, serviceGroupInfo irs.VMGroupInfo, frontEndIPConfigId string, backEndAddressPoolId string, probeId string) (network.LoadBalancingRule, error) {
	protocol := network.TransportProtocolTCP
	switch strings.ToUpper(listenerInfo.Protocol) {
	case "TCP", "ALL", "UDP":
		protocol = network.TransportProtocol(strings.Title(strings.ToLower(listenerInfo.Protocol)))
	default:
		return network.LoadBalancingRule{}, errors.New("invalid listenerInfo Protocol")
	}
	backendPort, err := strconv.Atoi(serviceGroupInfo.Port)
	if err != nil {
		return network.LoadBalancingRule{}, errors.New("invalid serviceGroupInfo Protocol")
	}
	frontendPort, err := strconv.Atoi(listenerInfo.Port)
	if err != nil {
		return network.LoadBalancingRule{}, errors.New("invalid listenerInfo Protocol")
	}
	loadBalancingRule := network.LoadBalancingRule{
		Name: to.StringPtr(generateRandName(LoadBalancingRulesPrefix)),
		LoadBalancingRulePropertiesFormat: &network.LoadBalancingRulePropertiesFormat{
			Protocol:             protocol,
			FrontendPort:         to.Int32Ptr(int32(frontendPort)),
			BackendPort:          to.Int32Ptr(int32(backendPort)),
			IdleTimeoutInMinutes: to.Int32Ptr(4),
			EnableFloatingIP:     to.BoolPtr(false),
			LoadDistribution:     network.LoadDistributionDefault,
			FrontendIPConfiguration: &network.SubResource{
				ID: to.StringPtr(frontEndIPConfigId),
			},
			BackendAddressPool: &network.SubResource{
				ID: to.StringPtr(backEndAddressPoolId),
			},
			Probe: &network.SubResource{
				ID: to.StringPtr(probeId),
			},
		},
	}
	if backEndAddressPoolId == "" {
		loadBalancingRule.LoadBalancingRulePropertiesFormat.BackendAddressPool = nil
	}
	return loadBalancingRule, nil
}
func getAzureProbeByCBHealthChecker(healthChecker irs.HealthCheckerInfo) (network.Probe, error) {
	protocol := network.ProbeProtocolHTTP
	switch strings.ToUpper(healthChecker.Protocol) {
	case "HTTP", "HTTPS", "TCP":
		protocol = network.ProbeProtocol(strings.Title(strings.ToLower(healthChecker.Protocol)))
	default:
		return network.Probe{}, errors.New("invalid HealthCheckerInfo Protocol")
	}
	port, err := strconv.Atoi(healthChecker.Port)
	if err != nil {
		return network.Probe{}, errors.New("invalid HealthCheckerInfo Protocol")
	}

	if healthChecker.Interval < 5 {
		return network.Probe{}, errors.New("invalid HealthCheckerInfo Interval, interval must be greater than 5")
	}
	if healthChecker.Threshold < 1 {
		return network.Probe{}, errors.New("invalid HealthCheckerInfo Threshold, Threshold  must be greater than 1")
	}
	if healthChecker.Interval*healthChecker.Threshold > 2147483647 {
		return network.Probe{}, errors.New("invalid HealthCheckerInfo Interval * Threshold must be between 5 and 2147483647 ")
	}
	probe := network.Probe{
		Name: to.StringPtr(generateRandName(ProbeNamePrefix)),
		ProbePropertiesFormat: &network.ProbePropertiesFormat{
			Protocol:          protocol,
			Port:              to.Int32Ptr(int32(port)),
			IntervalInSeconds: to.Int32Ptr(int32(healthChecker.Interval)),
			NumberOfProbes:    to.Int32Ptr(int32(healthChecker.Threshold)),
		},
	}
	if protocol == network.ProbeProtocolHTTP || protocol == network.ProbeProtocolHTTPS {
		probe.ProbePropertiesFormat.RequestPath = to.StringPtr("/")
	}
	return probe, nil
}
func getAzureFrontendIPConfiguration(publicIp *network.PublicIPAddress) (network.FrontendIPConfiguration, error) {
	return network.FrontendIPConfiguration{
		Name: to.StringPtr(generateRandName(FrontEndIPConfigPrefix)),
		FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
			PrivateIPAllocationMethod: network.IPAllocationMethodDynamic,
			PublicIPAddress:           publicIp,
		},
	}, nil
}
func (nlbHandler *AzureNLBHandler) createPublicIP(nlbName string) (network.PublicIPAddress, error) {
	// PublicIP 이름 생성
	publicIPName := nlbName

	createOpts := network.PublicIPAddress{
		Name: to.StringPtr(publicIPName),
		Sku: &network.PublicIPAddressSku{
			Name: network.PublicIPAddressSkuNameStandard,
		},
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAddressVersion:   network.IPVersion("IPv4"),
			PublicIPAllocationMethod: network.IPAllocationMethodStatic,
			IdleTimeoutInMinutes:     to.Int32Ptr(4),
		},
		Location: to.StringPtr(nlbHandler.Region.Region),
		Tags: map[string]*string{
			"createdBy": to.StringPtr(nlbName),
		},
	}

	future, err := nlbHandler.PublicIPClient.CreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, publicIPName, createOpts)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create PublicIP, error=%s", err))
		return network.PublicIPAddress{}, createErr
	}
	err = future.WaitForCompletionRef(nlbHandler.Ctx, nlbHandler.PublicIPClient.Client)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create PublicIP, error=%s", err))
		return network.PublicIPAddress{}, createErr
	}

	// 생성된 PublicIP 정보 리턴
	publicIP, err := nlbHandler.PublicIPClient.Get(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, publicIPName, "")
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create PublicIP, error=%s", err))
		return network.PublicIPAddress{}, createErr
	}
	return publicIP, nil
}
func (nlbHandler *AzureNLBHandler) getVPCNameSubnetNameAndPrivateIPByVM(server compute.VirtualMachine) (privateIP string, err error) {
	niList := *server.NetworkProfile.NetworkInterfaces
	var VNicId string
	for _, ni := range niList {
		if ni.ID != nil {
			VNicId = *ni.ID
			break
		}
	}

	nicIdArr := strings.Split(VNicId, "/")
	nicName := nicIdArr[len(nicIdArr)-1]
	vNic, err := nlbHandler.VNicClient.Get(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, nicName, "")
	if err != nil {
		return "", err
	}
	for _, ip := range *vNic.IPConfigurations {
		if *ip.Primary {
			privateIP := *ip.PrivateIPAddress
			return privateIP, nil
		}
	}
	return "", errors.New("not found subnet")
}
func (nlbHandler *AzureNLBHandler) getVPCIIDByVM(server compute.VirtualMachine) (irs.IID, error) {
	niList := *server.NetworkProfile.NetworkInterfaces
	var VNicId string
	for _, ni := range niList {
		if ni.ID != nil {
			VNicId = *ni.ID
			break
		}
	}

	nicIdArr := strings.Split(VNicId, "/")
	nicName := nicIdArr[len(nicIdArr)-1]
	vNic, err := nlbHandler.VNicClient.Get(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, nicName, "")
	if err != nil {
		return irs.IID{}, err
	}
	for _, ip := range *vNic.IPConfigurations {
		if *ip.Primary {
			subnetIdArr := strings.Split(*ip.InterfaceIPConfigurationPropertiesFormat.Subnet.ID, "/")
			vpcId := strings.Join(subnetIdArr[:len(subnetIdArr)-2], "/")
			return irs.IID{
				NameId:   GetResourceNameById(vpcId),
				SystemId: vpcId,
			}, nil
		}
	}
	return irs.IID{}, errors.New("not found subnet")
}
func (nlbHandler *AzureNLBHandler) getLoadBalancerBackendAddress(backEndAddressPoolName string, vpcId string, privateIP string) (network.LoadBalancerBackendAddress, error) {
	return network.LoadBalancerBackendAddress{
		LoadBalancerBackendAddressPropertiesFormat: &network.LoadBalancerBackendAddressPropertiesFormat{
			VirtualNetwork: &network.SubResource{
				ID: to.StringPtr(vpcId),
			},
			IPAddress: to.StringPtr(privateIP),
		},
		Name: to.StringPtr(backEndAddressPoolName + privateIP),
	}, nil
}
func (nlbHandler *AzureNLBHandler) getNLBType(nlb network.LoadBalancer) (string, error) {
	FrontendIPConfigurations := *nlb.FrontendIPConfigurations
	if len(FrontendIPConfigurations) <= 0 {
		return "", errors.New("invalid LoadBalancer")
	}
	cbOnlyOneFrontendIPConfigurations := FrontendIPConfigurations[0]
	if cbOnlyOneFrontendIPConfigurations.PublicIPAddress.ID == nil {
		return "INTERNAL", nil
	} else {
		return "PUBLIC", nil
	}
}
func (nlbHandler *AzureNLBHandler) getPrivateIPBYNicName(nicName string) (string, error) {
	vNic, err := nlbHandler.VNicClient.Get(nlbHandler.Ctx, nlbHandler.Region.ResourceGroup, nicName, "")
	if err != nil {
		return "", err
	}
	for _, ip := range *vNic.IPConfigurations {
		if *ip.Primary {
			privateIP := *ip.PrivateIPAddress
			return privateIP, nil
		}
	}
	return "", errors.New("Not exist IP")
}
func convertListenerInfoProtocolsToInboundRuleProtocol(protocol string) (network.TransportProtocol, error) {
	switch strings.ToUpper(protocol) {
	case "TCP":
		return network.TransportProtocolTCP, nil
	case "UDP":
		return network.TransportProtocolUDP, nil
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
