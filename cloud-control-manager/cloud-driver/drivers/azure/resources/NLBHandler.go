package resources

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

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
	VPCClient                    *armnetwork.VirtualNetworksClient
	VMClient                     *armcompute.VirtualMachinesClient
	ScaleSetVMsClient            *armcompute.VirtualMachineScaleSetVMsClient
	DiskClient                   *armcompute.DisksClient
	SubnetClient                 *armnetwork.SubnetsClient
	IPConfigClient               *armnetwork.InterfaceIPConfigurationsClient
	NLBLoadBalancingRulesClient  *armnetwork.LoadBalancerLoadBalancingRulesClient
	MetricClient                 *azquery.MetricsClient
	SecurityGroupClient          *armnetwork.SecurityGroupsClient
	SecurityRuleClient           *armnetwork.SecurityRulesClient
}

type BackendAddressesIPRefType string

type NLBType string
type NLBScope string

const (
	FrontEndIPConfigPrefix            = "frontEndIp"
	LoadBalancingRulesPrefix          = "lbrule"
	ProbeNamePrefix                   = "probe"
	BackEndAddressPoolPrefix          = "backend"
	NLBPublicType            NLBType  = "PUBLIC"
	NLBInternalType          NLBType  = "INTERNAL"
	NLBGlobalType            NLBScope = "GLOBAL"
	NLBRegionType            NLBScope = "REGION"
)

func convertToLoadBalancerBackendAddressStruct(backEndAddressPoolName string, vpcId string, privateIP string) *armnetwork.LoadBalancerBackendAddress {
	return &armnetwork.LoadBalancerBackendAddress{
		Properties: &armnetwork.LoadBalancerBackendAddressPropertiesFormat{
			VirtualNetwork: &armnetwork.SubResource{
				ID: &vpcId,
			},
			IPAddress: &privateIP,
		},
		Name: toStrPtr(backEndAddressPoolName + privateIP),
	}
}

func (nlbHandler *AzureNLBHandler) getVMPrivateIP(vpcID string, vmIID irs.IID) (privateIP string, err error) {
	vmHandler := AzureVMHandler{
		CredentialInfo:    nlbHandler.CredentialInfo,
		Region:            nlbHandler.Region,
		Ctx:               nlbHandler.Ctx,
		Client:            nlbHandler.VMClient,
		ScaleSetVMsClient: nlbHandler.ScaleSetVMsClient,
		SubnetClient:      nlbHandler.SubnetClient,
		NicClient:         nlbHandler.VNicClient,
		PublicIPClient:    nlbHandler.PublicIPClient,
		DiskClient:        nlbHandler.DiskClient,
	}

	vm, err := vmHandler.GetVM(vmIID)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to get VM. err = %s", err))
		cblogger.Error(err)
		return "", err
	}

	var nlbVPC *armnetwork.VirtualNetwork
	pager := nlbHandler.VPCClient.NewListPager(nlbHandler.Region.Region, nil)
	for pager.More() {
		page, err := pager.NextPage(nlbHandler.Ctx)
		if err != nil {
			return "", errors.New(fmt.Sprintf("Failed to get VPC list. err = %s", err))
		}

		for _, vpc := range page.Value {
			if *vpc.ID == vpcID {
				nlbVPC = vpc
				break
			}
		}
	}

	if nlbVPC == nil {
		return "", errors.New("failed to get NLB VPC")
	}

	if vm.VpcIID.SystemId != *nlbVPC.ID {
		return "", errors.New("VM does not belong to VPC")
	}

	return vm.PrivateIP, nil
}

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
	if nlbReqInfo.VMGroup.VMs == nil || len(*nlbReqInfo.VMGroup.VMs) == 0 {
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
			"createdAt": toStrPtr(strconv.FormatInt(time.Now().UTC().Unix(), 10)),
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

	// VPC validation is only needed when VMs are present (to get private IPs)
	var nlbVPC *armnetwork.VirtualNetwork
	if nlbReqInfo.VMGroup.VMs != nil && len(*nlbReqInfo.VMGroup.VMs) > 0 {
		pager := nlbHandler.VPCClient.NewListPager(nlbHandler.Region.Region, nil)
		for pager.More() {
			page, err := pager.NextPage(nlbHandler.Ctx)
			if err != nil {
				return irs.NLBInfo{}, errors.New(fmt.Sprintf("Failed to get VPC list. err = %s", err))
			}

			for _, vpc := range page.Value {
				if *vpc.Name == nlbReqInfo.VpcIID.NameId || *vpc.ID == nlbReqInfo.VpcIID.SystemId {
					nlbVPC = vpc
					break
				}
			}
		}

		if nlbVPC == nil {
			return irs.NLBInfo{}, errors.New("failed to get NLB VPC")
		}

		if nlbReqInfo.VpcIID.NameId != "" && nlbReqInfo.VpcIID.NameId != *nlbVPC.Name {
			return irs.NLBInfo{}, errors.New("found NLB VPC NameId is not matched")
		}
		if nlbReqInfo.VpcIID.SystemId != "" && nlbReqInfo.VpcIID.SystemId != *nlbVPC.ID {
			return irs.NLBInfo{}, errors.New("found NLB VPC SystemId is not matched")
		}
	}

	if nlbReqInfo.VMGroup.VMs != nil && len(*nlbReqInfo.VMGroup.VMs) > 0 && nlbVPC != nil {
		// Update BackEndPool
		var privateIPs []string
		for _, vmIId := range *nlbReqInfo.VMGroup.VMs {
			convertedIID, err := ConvertVMIID(vmIId, nlbHandler.CredentialInfo, nlbHandler.Region)
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to Get VM. err = %s", err))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return irs.NLBInfo{}, getErr
			}

			privateIP, err := nlbHandler.getVMPrivateIP(*nlbVPC.ID, convertedIID)
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return irs.NLBInfo{}, getErr
			}
			privateIPs = append(privateIPs, privateIP)
		}

		vpcId := GetNetworksResourceIdByName(nlbHandler.CredentialInfo, nlbHandler.Region, AzureVirtualNetworks, nlbReqInfo.VpcIID.NameId)

		resp, err := nlbHandler.NLBBackendAddressPoolsClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, nlbReqInfo.IId.NameId, backEndAddressPoolName, nil)
		if err != nil {
			createError = errors.New(fmt.Sprintf("Failed to Create NLB. err = %s", err.Error()))
			cblogger.Error(createError)
			LoggingError(hiscallInfo, createError)
			return irs.NLBInfo{}, createError
		}

		for _, ip := range privateIPs {
			resp.BackendAddressPool.Properties.LoadBalancerBackendAddresses =
				append(resp.BackendAddressPool.Properties.LoadBalancerBackendAddresses,
					convertToLoadBalancerBackendAddressStruct(backEndAddressPoolName, vpcId, ip))
		}

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

		// Add AzureLoadBalancer Health Probe rule to NSG for each VM
		err = nlbHandler.addHealthProbeRuleToVMsNSG(nlbReqInfo.IId, *nlbReqInfo.VMGroup.VMs, nlbReqInfo.HealthChecker)
		if err != nil {
			cblogger.Warnf("Failed to add AzureLoadBalancer rule to NSG (non-fatal): %s", err.Error())
			// Non-fatal error: continue NLB creation
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
			getErr := errors.New(fmt.Sprintf("Failed to List NLB. err = %s", err))
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

	// Get NLB info before deletion to retrieve VMs and HealthChecker info
	nlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err == nil && nlbInfo.VMGroup.VMs != nil && len(*nlbInfo.VMGroup.VMs) > 0 {
		// Remove AzureLoadBalancer rules from NSGs for this NLB
		err = nlbHandler.removeHealthProbeRuleFromVMsNSG(nlbIID, *nlbInfo.VMGroup.VMs, nlbInfo.HealthChecker)
		if err != nil {
			cblogger.Warnf("Failed to remove AzureLoadBalancer rule from NSG (non-fatal): %s", err.Error())
			// Non-fatal error: continue NLB deletion
		}
	}

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

		// Get VPC ID: if backend pool has existing VMs, use their VPC; otherwise get from first VM to add
		var vpcID string
		if cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses != nil && len(cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses) > 0 {
			vpcID = *cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses[0].Properties.VirtualNetwork.ID
		} else {
			// No existing VMs in NLB, get VPC from the first VM to add
			firstVMIID := (*vmIIDs)[0]
			vpcID, err = nlbHandler.getVPCIDFromVM(firstVMIID)
			if err != nil {
				addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
				cblogger.Error(addErr.Error())
				LoggingError(hiscallInfo, addErr)
				return irs.VMGroupInfo{}, addErr
			}
		}

		nlbCurrentVMIIds, err := nlbHandler.getVMIIDsByLoadBalancerBackendAddresses(vpcID, cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses)
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
		var privateIPs []string
		for _, vmIID := range *vmIIDs {
			convertedIID, err := ConvertVMIID(vmIID, nlbHandler.CredentialInfo, nlbHandler.Region)
			if err != nil {
				addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
				cblogger.Error(addErr.Error())
				LoggingError(hiscallInfo, addErr)
				return irs.VMGroupInfo{}, addErr
			}

			ip, err := nlbHandler.getVMPrivateIP(vpcID, convertedIID)
			if err != nil {
				addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
				cblogger.Error(addErr.Error())
				LoggingError(hiscallInfo, addErr)
				return irs.VMGroupInfo{}, addErr
			}
			privateIPs = append(privateIPs, ip)
		}

		resp, err := nlbHandler.NLBBackendAddressPoolsClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, nlbIID.NameId, backendPoolName, nil)
		if err != nil {
			addErr := errors.New(fmt.Sprintf("Failed to AddVMs NLB. err = %s", err.Error()))
			cblogger.Error(addErr.Error())
			LoggingError(hiscallInfo, addErr)
			return irs.VMGroupInfo{}, addErr
		}

		for _, ip := range privateIPs {
			resp.BackendAddressPool.Properties.LoadBalancerBackendAddresses =
				append(resp.BackendAddressPool.Properties.LoadBalancerBackendAddresses,
					convertToLoadBalancerBackendAddressStruct(backendPoolName, vpcID, ip))
		}

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

		// If NLB was created without VMs, LoadBalancingRule may not have BackendAddressPool reference
		// Update LoadBalancingRule to reference BackendAddressPool
		if len(nlb.Properties.LoadBalancingRules) > 0 {
			loadBalancingRule := nlb.Properties.LoadBalancingRules[0]
			if loadBalancingRule.Properties.BackendAddressPool == nil || loadBalancingRule.Properties.BackendAddressPool.ID == nil || *loadBalancingRule.Properties.BackendAddressPool.ID == "" {
				cblogger.Infof("LoadBalancingRule has no BackendAddressPool reference. Updating...")

				// Update NLB to connect LoadBalancingRule with BackendAddressPool
				nlbId := GetNetworksResourceIdByName(nlbHandler.CredentialInfo, nlbHandler.Region, AzureLoadBalancers, nlbIID.NameId)
				backEndAddressPoolId := fmt.Sprintf("%s/backendAddressPools/%s", nlbId, backendPoolName)

				loadBalancingRule.Properties.BackendAddressPool = &armnetwork.SubResource{
					ID: &backEndAddressPoolId,
				}

				nlbPoller, err := nlbHandler.NLBClient.BeginCreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.Region, nlbIID.NameId, *nlb, nil)
				if err != nil {
					cblogger.Warnf("Failed to update LoadBalancingRule with BackendAddressPool (non-fatal): %s", err.Error())
				} else {
					_, err = nlbPoller.PollUntilDone(nlbHandler.Ctx, nil)
					if err != nil {
						cblogger.Warnf("Failed to update LoadBalancingRule with BackendAddressPool (non-fatal): %s", err.Error())
					} else {
						cblogger.Infof("Successfully updated LoadBalancingRule to reference BackendAddressPool")
					}
				}
			}
		}

		// Get NLB HealthChecker info
		nlbInfo, err := nlbHandler.GetNLB(nlbIID)
		if err != nil {
			cblogger.Warnf("Failed to get NLB HealthChecker info: %s", err.Error())
		} else {
			cblogger.Infof("Adding AzureLoadBalancer Health Probe rules to NSG for %d VMs (Protocol: %s, Port: %s)",
				len(*vmIIDs), nlbInfo.HealthChecker.Protocol, nlbInfo.HealthChecker.Port)
			// Add AzureLoadBalancer Health Probe rule to NSG for newly added VMs
			err = nlbHandler.addHealthProbeRuleToVMsNSG(nlbIID, *vmIIDs, nlbInfo.HealthChecker)
			if err != nil {
				cblogger.Warnf("Failed to add AzureLoadBalancer rule to NSG (non-fatal): %s", err.Error())
				// Non-fatal error: continue AddVMs operation
			} else {
				cblogger.Infof("Successfully completed adding AzureLoadBalancer rules to NSGs for NLB %s", nlbIID.NameId)
			}
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
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "RemoveVMs()")
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
		vpcID := *cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses[0].Properties.VirtualNetwork.ID

		nlbCurrentVMIIds, err := nlbHandler.getVMIIDsByLoadBalancerBackendAddresses(vpcID, cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses)
		if err != nil {
			removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
			cblogger.Error(removeErr.Error())
			LoggingError(hiscallInfo, removeErr)
			return false, removeErr
		}

		for _, removeVmIId := range *vmIIDs {
			found := false
			for _, currentVMIId := range nlbCurrentVMIIds {
				if strings.EqualFold(currentVMIId.NameId, removeVmIId.NameId) {
					found = true
					break
				}
			}

			if !found {
				removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = can't remove not exist vm (" + removeVmIId.NameId + ")"))
				cblogger.Error(removeErr.Error())
				LoggingError(hiscallInfo, removeErr)
				return false, removeErr
			}
		}

		backendPoolName := *cbOnlyOneBackendPool.Name

		var nlbUpdateVMIIds []irs.IID
		for _, currentVMIId := range nlbCurrentVMIIds {
			found := false
			for _, removeVmIId := range *vmIIDs {
				if strings.EqualFold(currentVMIId.NameId, removeVmIId.NameId) {
					found = true
					break
				}
			}

			if found {
				continue
			}

			nlbUpdateVMIIds = append(nlbUpdateVMIIds, currentVMIId)
		}

		var updateVMPrivateIPs []string
		for _, vmIId := range nlbUpdateVMIIds {
			convertedIID, err := ConvertVMIID(vmIId, nlbHandler.CredentialInfo, nlbHandler.Region)
			if err != nil {
				removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
				cblogger.Error(removeErr.Error())
				LoggingError(hiscallInfo, removeErr)
				return false, removeErr
			}

			privateIP, err := nlbHandler.getVMPrivateIP(vpcID, convertedIID)
			if err != nil {
				removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
				cblogger.Error(removeErr.Error())
				LoggingError(hiscallInfo, removeErr)
				return false, removeErr
			}

			updateVMPrivateIPs = append(updateVMPrivateIPs, privateIP)
		}

		resp, err := nlbHandler.NLBBackendAddressPoolsClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, nlbIID.NameId, backendPoolName, nil)
		if err != nil {
			removeErr := errors.New(fmt.Sprintf("Failed to RemoveVMs NLB. err = %s", err.Error()))
			cblogger.Error(removeErr.Error())
			LoggingError(hiscallInfo, removeErr)
			return false, removeErr
		}

		var updateLoadBalancerBackendAddresses []*armnetwork.LoadBalancerBackendAddress
		for _, updateIP := range updateVMPrivateIPs {
			updateLoadBalancerBackendAddresses =
				append(updateLoadBalancerBackendAddresses,
					convertToLoadBalancerBackendAddressStruct(backendPoolName, vpcID, updateIP))
		}

		resp.BackendAddressPool.Properties.LoadBalancerBackendAddresses = updateLoadBalancerBackendAddresses

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

		// Remove AzureLoadBalancer rules from NSGs for removed VMs
		nlbInfo, err := nlbHandler.GetNLB(nlbIID)
		if err == nil {
			err = nlbHandler.removeHealthProbeRuleFromVMsNSG(nlbIID, *vmIIDs, nlbInfo.HealthChecker)
			if err != nil {
				cblogger.Warnf("Failed to remove AzureLoadBalancer rule from NSG (non-fatal): %s", err.Error())
				// Non-fatal error: continue RemoveVMs operation
			}
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
	// Use 3 minutes window for more stable results
	startTime := endTime.Add(time.Duration(-3) * time.Minute)
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
		cblogger.Warnf("Failed to get Azure Health Probe metric: %s (VM will be marked as unhealthy)", err.Error())
		return false, nil
	}

	if len(resp.Value) < 1 {
		cblogger.Warnf("No metric data available for backend IP %s (may be initializing)", ip)
		return false, nil
	}
	values := resp.Value
	if values[0] == nil {
		cblogger.Warnf("Metric value is nil for backend IP %s", ip)
		return false, nil
	}
	if len((*(values[0])).TimeSeries) < 1 {
		cblogger.Warnf("No time series data for backend IP %s", ip)
		return false, nil
	}
	TimeSeries := (*(values[0])).TimeSeries
	if TimeSeries[0] == nil {
		cblogger.Warnf("First time series is nil for backend IP %s", ip)
		return false, nil
	}
	timeSeriesData := (*(TimeSeries[0])).Data
	if len(timeSeriesData) < 1 {
		cblogger.Warnf("No data points in time series for backend IP %s", ip)
		return false, nil
	}
	data := timeSeriesData[len(timeSeriesData)-1]

	// Check if Average is nil
	if data.Average == nil {
		cblogger.Warnf("Average metric is nil for backend IP %s", ip)
		return false, nil
	}

	avg := int(*data.Average)
	cblogger.Infof("Backend IP %s DipAvailability: %d%%", ip, avg)

	// Consider healthy if availability is >= 80% (more lenient than 100%)
	if avg >= 80 {
		return true, nil
	}
	return false, nil
}

type vmIP struct {
	VMIID irs.IID
	IP    string
}

func (nlbHandler *AzureNLBHandler) getVMIPs(nlbIId irs.IID) ([]vmIP, error) {
	rawNLB, err := nlbHandler.getRawNLB(nlbIId)
	if err != nil {
		return nil, err
	}

	info, err := nlbHandler.setterNLB(rawNLB)
	if err != nil {
		return nil, err
	}

	var vmIPs []vmIP

	// Check if VMs is nil or empty
	if info.VMGroup.VMs == nil || len(*info.VMGroup.VMs) == 0 {
		return vmIPs, nil
	}

	for _, vmIID := range *info.VMGroup.VMs {
		resp, err := nlbHandler.VMClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, vmIID.NameId, nil)
		if err != nil {
			return nil, err
		}
		ip, err := nlbHandler.getPrivateIPByRawVm(&resp.VirtualMachine)
		if err != nil {
			return nil, err
		}

		vmIPs = append(vmIPs, vmIP{
			IP:    ip,
			VMIID: vmIID,
		})
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
			nlbInfo.CreatedTime = time.Unix(timeInt64, 0).UTC()
		}
	}

	// Get VPC information from backend addresses or frontend IP configuration
	var vpcID string
	if len(nlb.Properties.BackendAddressPools) > 0 {
		// TODO: Deliver multiple backendPools in the future
		cbOnlyOneBackendPool := nlb.Properties.BackendAddressPools[0]
		// VPC information can be retrieved from backend addresses if they exist
		if len(cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses) > 0 {
			vpcID = *cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses[0].Properties.VirtualNetwork.ID
		}
	}

	// If no backend addresses (no VMs), get VPC info from subnet in frontend IP config
	if vpcID == "" && len(nlb.Properties.FrontendIPConfigurations) > 0 {
		frontendIPConfig := nlb.Properties.FrontendIPConfigurations[0]
		if frontendIPConfig.Properties.Subnet != nil && frontendIPConfig.Properties.Subnet.ID != nil {
			// Subnet ID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{vnet}/subnets/{subnet}
			// Extract VNet ID by removing the subnet part
			subnetID := *frontendIPConfig.Properties.Subnet.ID
			re := regexp.MustCompile(`(.+/virtualNetworks/[^/]+)`)
			if match := re.FindStringSubmatch(subnetID); len(match) > 1 {
				vpcID = match[1]
			}
		}
	}

	if vpcID != "" {
		nlbInfo.VpcIID = irs.IID{
			NameId:   GetResourceNameById(vpcID),
			SystemId: vpcID,
		}
	}

	vmGroup, listenerInfo, healthCheckerInfo, err := nlbHandler.getLoadBalancingRuleInfoByNLB(nlb)
	if err != nil {
		return nil, err
	}
	nlbInfo.VMGroup = *vmGroup
	nlbInfo.HealthChecker = *healthCheckerInfo
	nlbInfo.Listener = *listenerInfo

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

	nlbInfo.KeyValueList = irs.StructToKeyValueList(nlb)

	return &nlbInfo, nil
}

func getVNICNames(nlbVPC *armnetwork.VirtualNetwork) []*string {
	var VNICNames []*string

	re := regexp.MustCompile(`networkInterfaces/(.+?)/ipConfigurations`)

	subnets := nlbVPC.Properties.Subnets
	for _, subnet := range subnets {
		ipConfigs := subnet.Properties.IPConfigurations
		for _, ipConfig := range ipConfigs {
			if ipConfig.ID != nil {
				match := re.FindStringSubmatch(*ipConfig.ID)
				if len(match) > 1 {
					VNICNames = append(VNICNames, &match[1])
				}
			}
		}
	}

	return VNICNames
}

func (nlbHandler *AzureNLBHandler) getVMIIDsByLoadBalancerBackendAddresses(vpcID string, address []*armnetwork.LoadBalancerBackendAddress) ([]irs.IID, error) {
	vmIIds := make([]irs.IID, 0)

	if len(address) < 1 {
		return vmIIds, nil
	}

	var nlbVPC *armnetwork.VirtualNetwork
	pager := nlbHandler.VPCClient.NewListPager(nlbHandler.Region.Region, nil)
	for pager.More() {
		page, err := pager.NextPage(nlbHandler.Ctx)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("failed to list VPC list: %s", err.Error()))
		}

		for _, vpc := range page.Value {
			if *vpc.ID == vpcID {
				nlbVPC = vpc
				break
			}
		}
	}

	if nlbVPC == nil {
		return nil, errors.New("failed to get NLB VPC")
	}

	vmHandler := AzureVMHandler{
		CredentialInfo:    nlbHandler.CredentialInfo,
		Region:            nlbHandler.Region,
		Ctx:               nlbHandler.Ctx,
		Client:            nlbHandler.VMClient,
		ScaleSetVMsClient: nlbHandler.ScaleSetVMsClient,
		SubnetClient:      nlbHandler.SubnetClient,
		NicClient:         nlbHandler.VNicClient,
		PublicIPClient:    nlbHandler.PublicIPClient,
		DiskClient:        nlbHandler.DiskClient,
	}

	vmList, err := vmHandler.ListVM()
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to get VMs. err = %s", err))
		cblogger.Error(err)
		return nil, err
	}

	var ips []*string
	for _, addr := range address {
		if addr.Properties.IPAddress != nil {
			ips = append(ips, addr.Properties.IPAddress)
		}
	}

	vNICNames := getVNICNames(nlbVPC)
	for _, vm := range vmList {
		vmFound := false

		for _, vNICName := range vNICNames {
			if strings.ToLower(*vNICName) == strings.ToLower(vm.NetworkInterface) {
				for _, ip := range ips {
					if vm.PrivateIP == *ip {
						vmIIds = append(vmIIds, vm.IId)
						vmFound = true
						break
					}
				}
			}

			if vmFound {
				continue
			}
		}
	}

	return vmIIds, err
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

func (nlbHandler *AzureNLBHandler) getLoadBalancingRuleInfoByNLB(nlb *armnetwork.LoadBalancer) (*irs.VMGroupInfo, *irs.ListenerInfo, *irs.HealthCheckerInfo, error) {
	LoadBalancingRules := nlb.Properties.LoadBalancingRules
	if len(LoadBalancingRules) <= 0 {
		return nil, nil, nil, errors.New("invalid LoadBalancer")
	}

	frontendIP, err := nlbHandler.getFrontendIPByNLB(nlb)
	if err != nil {
		return nil, nil, nil, errors.New("invalid LoadBalancer")
	}

	cbOnlyOneLoadBalancingRule := LoadBalancingRules[0]
	VMGroup := &irs.VMGroupInfo{
		Protocol: strings.ToUpper(string(*cbOnlyOneLoadBalancingRule.Properties.Protocol)),
		Port:     strconv.Itoa(int(*cbOnlyOneLoadBalancingRule.Properties.BackendPort)),
		// TODO: ?
		CspID: "",
	}

	VMGroup.KeyValueList = irs.StructToKeyValueList(cbOnlyOneLoadBalancingRule.Properties)

	listenerInfo := &irs.ListenerInfo{
		Protocol: strings.ToUpper(string(*cbOnlyOneLoadBalancingRule.Properties.Protocol)),
		Port:     strconv.Itoa(int(*cbOnlyOneLoadBalancingRule.Properties.FrontendPort)),
		IP:       frontendIP,
		// TODO: ?
		CspID: "",
	}

	listenerInfo.KeyValueList = irs.StructToKeyValueList(cbOnlyOneLoadBalancingRule.Properties)

	if len(nlb.Properties.BackendAddressPools) < 1 {
		return nil, nil, nil, errors.New("invalid LoadBalancer")
	}

	backendPools := nlb.Properties.BackendAddressPools
	cbOnlyOneBackendPool := backendPools[0]

	// VMs are optional - if no backend addresses, use empty list
	vmIIds := make([]irs.IID, 0)
	if len(cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses) > 0 {
		vpcID := *cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses[0].Properties.VirtualNetwork.ID
		var err error
		vmIIds, err = nlbHandler.getVMIIDsByLoadBalancerBackendAddresses(vpcID, cbOnlyOneBackendPool.Properties.LoadBalancerBackendAddresses)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	VMGroup.VMs = &vmIIds
	healthCheckerInfo := &irs.HealthCheckerInfo{}

	probeId := *cbOnlyOneLoadBalancingRule.Properties.Probe.ID
	for _, probe := range nlb.Properties.Probes {
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

func getAzureLoadBalancingRuleByCBListenerInfo(listenerInfo irs.ListenerInfo, vmGroupInfo irs.VMGroupInfo, frontEndIPConfigId string, backEndAddressPoolId string, probeId string) (*armnetwork.LoadBalancingRule, error) {
	var protocol armnetwork.TransportProtocol

	switch strings.ToUpper(listenerInfo.Protocol) {
	case "TCP":
		protocol = armnetwork.TransportProtocolTCP
	case "UDP":
		protocol = armnetwork.TransportProtocolUDP
	case "ALL":
		protocol = armnetwork.TransportProtocolAll
	default:
		return nil, errors.New("invalid listenerInfo protocol")
	}

	backendPort, err := strconv.Atoi(vmGroupInfo.Port)
	if err != nil {
		return nil, errors.New("invalid vmGroupInfo port")
	}
	frontendPort, err := strconv.Atoi(listenerInfo.Port)
	if err != nil {
		return nil, errors.New("invalid listenerInfo port")
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
	var protocol armnetwork.ProbeProtocol

	switch strings.ToUpper(healthChecker.Protocol) {
	case "HTTP":
		protocol = armnetwork.ProbeProtocolHTTP
	case "HTTPS":
		protocol = armnetwork.ProbeProtocolHTTPS
	case "TCP":
		protocol = armnetwork.ProbeProtocolTCP
	default:
		return nil, errors.New("invalid HealthCheckerInfo protocol")
	}

	port, err := strconv.Atoi(healthChecker.Port)
	if err != nil {
		return nil, errors.New("invalid HealthCheckerInfo port")
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

	resp, err := nlbHandler.PublicIPClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, publicIPName, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create PublicIP, error=%s", err))
		return armnetwork.PublicIPAddress{}, createErr
	}

	return resp.PublicIPAddress, nil
}

func (nlbHandler *AzureNLBHandler) getVPCIIDByVM(vmIID irs.IID) (*irs.IID, error) {
	vmHandler := AzureVMHandler{
		CredentialInfo:    nlbHandler.CredentialInfo,
		Region:            nlbHandler.Region,
		Ctx:               nlbHandler.Ctx,
		Client:            nlbHandler.VMClient,
		ScaleSetVMsClient: nlbHandler.ScaleSetVMsClient,
		SubnetClient:      nlbHandler.SubnetClient,
		NicClient:         nlbHandler.VNicClient,
		PublicIPClient:    nlbHandler.PublicIPClient,
	}

	vm, err := vmHandler.GetVM(vmIID)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to get VPC, error=%s", err))
	}

	return &vm.VpcIID, nil
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
	// Skip validation if Interval is -1 (default)
	if healthCheckerInfo.Interval != -1 && healthCheckerInfo.Interval < 5 {
		return errors.New("invalid HealthCheckerInfo Interval, interval must be greater than 5")
	}
	// Skip validation if Threshold is -1 (default)
	if healthCheckerInfo.Threshold != -1 && healthCheckerInfo.Threshold < 1 {
		return errors.New("invalid HealthCheckerInfo Threshold, Threshold  must be greater than 1")
	}
	// Skip validation if either is -1 (default)
	if healthCheckerInfo.Interval != -1 && healthCheckerInfo.Threshold != -1 && healthCheckerInfo.Interval*healthCheckerInfo.Threshold > 2147483647 {
		return errors.New("invalid HealthCheckerInfo Interval * Threshold must be between 5 and 2147483647 ")
	}
	return nil
}

// getVPCIDFromVM retrieves VPC ID from a VM's network interface
func (nlbHandler *AzureNLBHandler) getVPCIDFromVM(vmIID irs.IID) (string, error) {
	convertedIID, err := ConvertVMIID(vmIID, nlbHandler.CredentialInfo, nlbHandler.Region)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to convert VM IID. err = %s", err))
	}

	vm, err := nlbHandler.VMClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, convertedIID.NameId, nil)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to get VM. err = %s", err))
	}

	if vm.Properties.NetworkProfile == nil || len(vm.Properties.NetworkProfile.NetworkInterfaces) == 0 {
		return "", errors.New("VM has no network interfaces")
	}

	nicID := *vm.Properties.NetworkProfile.NetworkInterfaces[0].ID
	nicIDArr := strings.Split(nicID, "/")
	nicName := nicIDArr[len(nicIDArr)-1]

	nic, err := nlbHandler.VNicClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, nicName, nil)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to get NIC. err = %s", err))
	}

	if nic.Properties.IPConfigurations == nil || len(nic.Properties.IPConfigurations) == 0 {
		return "", errors.New("NIC has no IP configurations")
	}

	if nic.Properties.IPConfigurations[0].Properties.Subnet == nil {
		return "", errors.New("NIC IP configuration has no subnet")
	}

	subnetID := *nic.Properties.IPConfigurations[0].Properties.Subnet.ID
	// Subnet ID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{vnet}/subnets/{subnet}
	// Extract VPC ID by removing the last "/subnets/{subnet}" part
	lastSlashIndex := strings.LastIndex(subnetID, "/subnets/")
	if lastSlashIndex == -1 {
		return "", errors.New("Invalid subnet ID format")
	}

	return subnetID[:lastSlashIndex], nil
}

// addHealthProbeRuleToVMsNSG adds AzureLoadBalancer service tag rule to NSG of VMs for Health Probe
func (nlbHandler *AzureNLBHandler) addHealthProbeRuleToVMsNSG(nlbIID irs.IID, vmIIDs []irs.IID, healthChecker irs.HealthCheckerInfo) error {
	if len(vmIIDs) == 0 {
		return nil
	}

	// Get unique NSGs from all VMs
	nsgMap := make(map[string]bool)
	for _, vmIID := range vmIIDs {
		nsgIIDs, err := nlbHandler.getNSGsFromVM(vmIID)
		if err != nil {
			cblogger.Warnf("Failed to get NSG from VM %s: %s", vmIID.NameId, err.Error())
			continue
		}
		for _, nsgIID := range nsgIIDs {
			nsgMap[nsgIID.SystemId] = true
		}
	}

	if len(nsgMap) == 0 {
		return errors.New("no NSGs found for the VMs")
	}

	// Add AzureLoadBalancer rule to each unique NSG
	for nsgID := range nsgMap {
		nsgName := GetResourceNameById(nsgID)
		err := nlbHandler.addAzureLoadBalancerRuleToNSG(nlbIID, nsgName, healthChecker)
		if err != nil {
			cblogger.Warnf("Failed to add AzureLoadBalancer rule to NSG %s: %s", nsgName, err.Error())
			// Continue to next NSG even if one fails
		} else {
			cblogger.Infof("Successfully added AzureLoadBalancer rule to NSG %s for NLB %s (port %s)", nsgName, nlbIID.NameId, healthChecker.Port)
		}
	}

	return nil
}

// getNSGsFromVM retrieves NSG IIDs associated with a VM
func (nlbHandler *AzureNLBHandler) getNSGsFromVM(vmIID irs.IID) ([]irs.IID, error) {
	convertedIID, err := ConvertVMIID(vmIID, nlbHandler.CredentialInfo, nlbHandler.Region)
	if err != nil {
		return nil, err
	}

	vm, err := nlbHandler.VMClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, convertedIID.NameId, nil)
	if err != nil {
		return nil, err
	}

	var nsgIIDs []irs.IID
	if vm.Properties.NetworkProfile != nil {
		for _, nicRef := range vm.Properties.NetworkProfile.NetworkInterfaces {
			nicID := *nicRef.ID
			nicName := GetResourceNameById(nicID)

			nic, err := nlbHandler.VNicClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, nicName, nil)
			if err != nil {
				continue
			}

			// Check NIC-level NSG
			if nic.Properties.NetworkSecurityGroup != nil {
				nsgIIDs = append(nsgIIDs, irs.IID{
					NameId:   GetResourceNameById(*nic.Properties.NetworkSecurityGroup.ID),
					SystemId: *nic.Properties.NetworkSecurityGroup.ID,
				})
			}

			// Check Subnet-level NSG
			if nic.Properties.IPConfigurations != nil && len(nic.Properties.IPConfigurations) > 0 {
				if nic.Properties.IPConfigurations[0].Properties.Subnet != nil {
					subnetID := *nic.Properties.IPConfigurations[0].Properties.Subnet.ID
					subnetIDArr := strings.Split(subnetID, "/")
					if len(subnetIDArr) >= 11 {
						vnetName := subnetIDArr[8]
						subnetName := subnetIDArr[10]

						subnet, err := nlbHandler.SubnetClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, vnetName, subnetName, nil)
						if err == nil && subnet.Properties.NetworkSecurityGroup != nil {
							nsgIIDs = append(nsgIIDs, irs.IID{
								NameId:   GetResourceNameById(*subnet.Properties.NetworkSecurityGroup.ID),
								SystemId: *subnet.Properties.NetworkSecurityGroup.ID,
							})
						}
					}
				}
			}
		}
	}

	return nsgIIDs, nil
}

// addAzureLoadBalancerRuleToNSG adds a rule to allow AzureLoadBalancer service tag
func (nlbHandler *AzureNLBHandler) addAzureLoadBalancerRuleToNSG(nlbIID irs.IID, nsgName string, healthChecker irs.HealthCheckerInfo) error {
	// Get existing NSG
	nsg, err := nlbHandler.SecurityGroupClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, nsgName, nil)
	if err != nil {
		return err
	}

	// Check if AzureLoadBalancer rule already exists for this NLB and port
	ruleName := fmt.Sprintf("AllowAzureLoadBalancer-%s-%s-%s", nlbIID.NameId, healthChecker.Protocol, healthChecker.Port)
	for _, rule := range nsg.Properties.SecurityRules {
		if rule.Name != nil && *rule.Name == ruleName {
			cblogger.Infof("AzureLoadBalancer rule already exists in NSG %s", nsgName)
			return nil
		}
	}

	// Find available priority (start from 4000 to avoid conflicts with user rules)
	priority := int32(4000)
	priorityMap := make(map[int32]bool)
	for _, rule := range nsg.Properties.SecurityRules {
		if rule.Properties.Priority != nil {
			priorityMap[*rule.Properties.Priority] = true
		}
	}
	for priorityMap[priority] && priority < 4096 {
		priority++
	}
	if priority >= 4096 {
		return errors.New("no available priority for NSG rule (4000-4095 range full)")
	}

	// Create security rule for AzureLoadBalancer service tag
	protocol := armnetwork.SecurityRuleProtocolTCP
	if strings.EqualFold(healthChecker.Protocol, "UDP") {
		protocol = armnetwork.SecurityRuleProtocolUDP
	}

	securityHandler := &AzureSecurityHandler{
		Region:     nlbHandler.Region,
		Ctx:        nlbHandler.Ctx,
		Client:     nlbHandler.SecurityGroupClient,
		RuleClient: nlbHandler.SecurityRuleClient,
	}

	rule := armnetwork.SecurityRule{
		Name: &ruleName,
		Properties: &armnetwork.SecurityRulePropertiesFormat{
			Protocol:                 &protocol,
			SourcePortRange:          toStrPtr("*"),
			DestinationPortRange:     &healthChecker.Port,
			SourceAddressPrefix:      toStrPtr("AzureLoadBalancer"),
			DestinationAddressPrefix: toStrPtr("*"),
			Access:                   toArmNetworkAccess(armnetwork.SecurityRuleAccessAllow),
			Priority:                 &priority,
			Direction:                toArmNetworkDirection(armnetwork.SecurityRuleDirectionInbound),
			Description:              toStrPtr("Allow Azure Load Balancer Health Probe"),
		},
	}

	poller, err := securityHandler.RuleClient.BeginCreateOrUpdate(nlbHandler.Ctx, nlbHandler.Region.Region, nsgName, ruleName, rule, nil)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(nlbHandler.Ctx, nil)
	if err != nil {
		return err
	}

	return nil
}

// Helper function to convert to armnetwork.SecurityRuleAccess pointer
func toArmNetworkAccess(access armnetwork.SecurityRuleAccess) *armnetwork.SecurityRuleAccess {
	return &access
}

// Helper function to convert to armnetwork.SecurityRuleDirection pointer
func toArmNetworkDirection(direction armnetwork.SecurityRuleDirection) *armnetwork.SecurityRuleDirection {
	return &direction
}

// removeHealthProbeRuleFromVMsNSG removes AzureLoadBalancer service tag rule from NSG of VMs
func (nlbHandler *AzureNLBHandler) removeHealthProbeRuleFromVMsNSG(nlbIID irs.IID, vmIIDs []irs.IID, healthChecker irs.HealthCheckerInfo) error {
	if len(vmIIDs) == 0 {
		return nil
	}

	// Get unique NSGs from all VMs
	nsgMap := make(map[string]bool)
	for _, vmIID := range vmIIDs {
		nsgIIDs, err := nlbHandler.getNSGsFromVM(vmIID)
		if err != nil {
			cblogger.Warnf("Failed to get NSG from VM %s: %s", vmIID.NameId, err.Error())
			continue
		}
		for _, nsgIID := range nsgIIDs {
			nsgMap[nsgIID.SystemId] = true
		}
	}

	if len(nsgMap) == 0 {
		cblogger.Warnf("No NSGs found for the VMs (VMs may not have NSG or already deleted)")
		return nil
	}

	// Remove AzureLoadBalancer rule from each unique NSG
	for nsgID := range nsgMap {
		nsgName := GetResourceNameById(nsgID)
		err := nlbHandler.removeAzureLoadBalancerRuleFromNSG(nlbIID, nsgName, healthChecker)
		if err != nil {
			cblogger.Warnf("Failed to remove AzureLoadBalancer rule from NSG %s: %s", nsgName, err.Error())
			// Continue to next NSG even if one fails
		} else {
			cblogger.Infof("Successfully removed AzureLoadBalancer rule from NSG %s for NLB %s (port %s)", nsgName, nlbIID.NameId, healthChecker.Port)
		}
	}

	return nil
}

// removeAzureLoadBalancerRuleFromNSG removes the AzureLoadBalancer rule for specific NLB
func (nlbHandler *AzureNLBHandler) removeAzureLoadBalancerRuleFromNSG(nlbIID irs.IID, nsgName string, healthChecker irs.HealthCheckerInfo) error {
	// Get existing NSG
	nsg, err := nlbHandler.SecurityGroupClient.Get(nlbHandler.Ctx, nlbHandler.Region.Region, nsgName, nil)
	if err != nil {
		return err
	}

	// Check if AzureLoadBalancer rule exists for this NLB
	ruleName := fmt.Sprintf("AllowAzureLoadBalancer-%s-%s-%s", nlbIID.NameId, healthChecker.Protocol, healthChecker.Port)
	ruleExists := false
	for _, rule := range nsg.Properties.SecurityRules {
		if rule.Name != nil && *rule.Name == ruleName {
			ruleExists = true
			break
		}
	}

	if !ruleExists {
		cblogger.Infof("AzureLoadBalancer rule does not exist in NSG %s (already removed or never added)", nsgName)
		return nil
	}

	// Delete the rule
	securityHandler := &AzureSecurityHandler{
		Region:     nlbHandler.Region,
		Ctx:        nlbHandler.Ctx,
		Client:     nlbHandler.SecurityGroupClient,
		RuleClient: nlbHandler.SecurityRuleClient,
	}

	poller, err := securityHandler.RuleClient.BeginDelete(nlbHandler.Ctx, nlbHandler.Region.Region, nsgName, ruleName, nil)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(nlbHandler.Ctx, nil)
	if err != nil {
		return err
	}

	return nil
}

func (nlbHandler *AzureNLBHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(nlbHandler.Region, "NETWORKLOADBALANCE", "NLB", "ListIID()")
	start := call.Start()

	var iidList []*irs.IID

	pager := nlbHandler.NLBClient.NewListPager(nlbHandler.Region.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(nlbHandler.Ctx)
		if err != nil {
			err = errors.New(fmt.Sprintf("Failed to List NLB. err = %s", err))
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
			return make([]*irs.IID, 0), err
		}

		for _, nlb := range page.Value {
			var iid irs.IID

			if nlb.ID != nil {
				iid.SystemId = *nlb.ID
			}
			if nlb.Name != nil {
				iid.NameId = *nlb.Name
			}

			iidList = append(iidList, &iid)
		}
	}

	LoggingInfo(hiscallInfo, start)

	return iidList, nil
}
