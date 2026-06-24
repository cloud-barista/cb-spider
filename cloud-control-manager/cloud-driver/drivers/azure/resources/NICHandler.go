// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// Azure NIC Handler
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v8"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	NIC = "NIC"
)

type AzureNICHandler struct {
	Region    idrv.RegionInfo
	Ctx       context.Context
	NicClient *armnetwork.InterfacesClient
	VMClient  *armcompute.VirtualMachinesClient
}

// ListIID lists all NIC IIDs in the resource group.
func (nicHandler *AzureNICHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region, call.NIC, NIC, "ListIID()")
	start := call.Start()

	pager := nicHandler.NicClient.NewListPager(nicHandler.Region.Region, nil)
	var iidList []*irs.IID
	for pager.More() {
		page, err := pager.NextPage(nicHandler.Ctx)
		if err != nil {
			getErr := fmt.Errorf("failed to list NICs: %w", err)
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}
		for _, nic := range page.Value {
			if nic.Name == nil || nic.ID == nil {
				continue
			}
			iidList = append(iidList, &irs.IID{
				NameId:   *nic.Name,
				SystemId: *nic.ID,
			})
		}
	}
	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

// CreateNIC creates a new NIC with the given request info.
func (nicHandler *AzureNICHandler) CreateNIC(nicReqInfo irs.NICReqInfo) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region, call.NIC, nicReqInfo.IId.NameId, "CreateNIC()")
	start := call.Start()

	// Resolve Subnet SystemId
	subnetID := nicReqInfo.SubnetIID.SystemId
	if subnetID == "" {
		createErr := fmt.Errorf("failed to create NIC: SubnetIID.SystemId is required")
		LoggingError(hiscallInfo, createErr)
		return irs.NICInfo{}, createErr
	}

	privateIPAllocationMethod := armnetwork.IPAllocationMethodDynamic
	ipConfig := &armnetwork.InterfaceIPConfiguration{
		Name: toStrPtr("ipConfig1"),
		Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
			Subnet: &armnetwork.Subnet{
				ID: &subnetID,
			},
			PrivateIPAllocationMethod: &privateIPAllocationMethod,
		},
	}

	createOpts := armnetwork.Interface{
		Location: &nicHandler.Region.Region,
		Properties: &armnetwork.InterfacePropertiesFormat{
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{ipConfig},
		},
	}

	// Attach security group (Azure NIC supports one NSG; use first if multiple provided)
	if len(nicReqInfo.SecurityGroupIIDs) > 0 {
		sgID := nicReqInfo.SecurityGroupIIDs[0].SystemId
		if sgID == "" {
			createErr := fmt.Errorf("failed to create NIC: SecurityGroupIIDs[0].SystemId is required")
			LoggingError(hiscallInfo, createErr)
			return irs.NICInfo{}, createErr
		}
		createOpts.Properties.NetworkSecurityGroup = &armnetwork.SecurityGroup{
			ID: &sgID,
		}
	}

	// Apply tags
	if len(nicReqInfo.TagList) > 0 {
		tags := make(map[string]*string)
		for _, tag := range nicReqInfo.TagList {
			v := tag.Value
			tags[tag.Key] = &v
		}
		createOpts.Tags = tags
	}

	poller, err := nicHandler.NicClient.BeginCreateOrUpdate(
		nicHandler.Ctx,
		nicHandler.Region.Region,
		nicReqInfo.IId.NameId,
		createOpts,
		nil,
	)
	if err != nil {
		createErr := fmt.Errorf("failed to create NIC: %w", err)
		LoggingError(hiscallInfo, createErr)
		return irs.NICInfo{}, createErr
	}

	result, err := poller.PollUntilDone(nicHandler.Ctx, nil)
	if err != nil {
		createErr := fmt.Errorf("failed to create NIC (polling): %w", err)
		LoggingError(hiscallInfo, createErr)
		return irs.NICInfo{}, createErr
	}

	LoggingInfo(hiscallInfo, start)
	return extractAzureNICInfo(&result.Interface), nil
}

// ListNIC lists all NICs in the resource group.
func (nicHandler *AzureNICHandler) ListNIC() ([]*irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region, call.NIC, NIC, "ListNIC()")
	start := call.Start()

	pager := nicHandler.NicClient.NewListPager(nicHandler.Region.Region, nil)
	var nicList []*irs.NICInfo
	for pager.More() {
		page, err := pager.NextPage(nicHandler.Ctx)
		if err != nil {
			listErr := fmt.Errorf("failed to list NICs: %w", err)
			LoggingError(hiscallInfo, listErr)
			return nil, listErr
		}
		for _, nic := range page.Value {
			if nic.Name == nil {
				continue
			}
			// Re-fetch with expand to get PublicIPAddress.IPAddress in each IP config.
			expandStr := "ipConfigurations/publicIPAddress"
			getResp, err := nicHandler.NicClient.Get(nicHandler.Ctx, nicHandler.Region.Region, *nic.Name,
				&armnetwork.InterfacesClientGetOptions{Expand: &expandStr})
			if err == nil {
				info := extractAzureNICInfo(&getResp.Interface)
				nicList = append(nicList, &info)
			} else {
				info := extractAzureNICInfo(nic)
				nicList = append(nicList, &info)
			}
		}
	}
	LoggingInfo(hiscallInfo, start)
	return nicList, nil
}

// GetNIC retrieves a NIC by its IID.
func (nicHandler *AzureNICHandler) GetNIC(nicIID irs.IID) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region, call.NIC, nicIID.NameId, "GetNIC()")
	start := call.Start()

	nicName := nicIID.NameId
	if nicName == "" && nicIID.SystemId != "" {
		// Extract resource name from SystemId (last segment)
		parts := strings.Split(nicIID.SystemId, "/")
		nicName = parts[len(parts)-1]
	}

	expandOpt2 := "ipConfigurations/publicIPAddress"
	resp, err := nicHandler.NicClient.Get(nicHandler.Ctx, nicHandler.Region.Region, nicName, &armnetwork.InterfacesClientGetOptions{Expand: &expandOpt2})
	if err != nil {
		getErr := fmt.Errorf("failed to get NIC %s: %w", nicName, err)
		LoggingError(hiscallInfo, getErr)
		return irs.NICInfo{}, getErr
	}

	LoggingInfo(hiscallInfo, start)
	return extractAzureNICInfo(&resp.Interface), nil
}

// DeleteNIC deletes a NIC by its IID.
func (nicHandler *AzureNICHandler) DeleteNIC(nicIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region, call.NIC, nicIID.NameId, "DeleteNIC()")
	start := call.Start()

	nicName := nicIID.NameId
	if nicName == "" && nicIID.SystemId != "" {
		parts := strings.Split(nicIID.SystemId, "/")
		nicName = parts[len(parts)-1]
	}

	poller, err := nicHandler.NicClient.BeginDelete(nicHandler.Ctx, nicHandler.Region.Region, nicName, nil)
	if err != nil {
		delErr := fmt.Errorf("failed to delete NIC %s: %w", nicName, err)
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}

	_, err = poller.PollUntilDone(nicHandler.Ctx, nil)
	if err != nil {
		delErr := fmt.Errorf("failed to delete NIC %s (polling): %w", nicName, err)
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// AttachNIC attaches a NIC to a VM via Azure VM NetworkProfile update.
// Azure requires: deallocate VM → add NIC → start VM.
func (nicHandler *AzureNICHandler) AttachNIC(nicIID irs.IID, vmIID irs.IID) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region, call.NIC, nicIID.NameId, "AttachNIC()")
	start := call.Start()

	// Resolve NIC name
	nicName := nicIID.NameId
	if nicName == "" && nicIID.SystemId != "" {
		parts := strings.Split(nicIID.SystemId, "/")
		nicName = parts[len(parts)-1]
	}
	// Get NIC ARM ID
	nicResp, err := nicHandler.NicClient.Get(nicHandler.Ctx, nicHandler.Region.Region, nicName, nil)
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("AttachNIC: failed to get NIC %s: %w", nicName, err)
	}
	nicARMID := nicResp.ID

	// Resolve VM name
	vmName := vmIID.NameId
	if vmName == "" && vmIID.SystemId != "" {
		parts := strings.Split(vmIID.SystemId, "/")
		vmName = parts[len(parts)-1]
	}

	// 1) Deallocate VM
	deallocPoller, err := nicHandler.VMClient.BeginDeallocate(nicHandler.Ctx, nicHandler.Region.Region, vmName, nil)
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("AttachNIC: failed to start VM deallocation: %w", err)
	}
	if _, err = deallocPoller.PollUntilDone(nicHandler.Ctx, nil); err != nil {
		return irs.NICInfo{}, fmt.Errorf("AttachNIC: VM deallocation failed: %w", err)
	}

	// 2) Get current VM config
	vmResp, err := nicHandler.VMClient.Get(nicHandler.Ctx, nicHandler.Region.Region, vmName, nil)
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("AttachNIC: failed to get VM %s: %w", vmName, err)
	}
	vm := vmResp.VirtualMachine

	// 3) Append NIC to NetworkProfile
	// Azure requires Primary property set on all NetworkInterfaceReferences.
	// First NIC = Primary:true, all others = Primary:false.
	if vm.Properties == nil {
		vm.Properties = &armcompute.VirtualMachineProperties{}
	}
	if vm.Properties.NetworkProfile == nil {
		vm.Properties.NetworkProfile = &armcompute.NetworkProfile{}
	}
	isPrimary := len(vm.Properties.NetworkProfile.NetworkInterfaces) == 0
	// Ensure existing NICs have Primary properly set (first one = primary)
	for i, ref := range vm.Properties.NetworkProfile.NetworkInterfaces {
		if ref.Properties == nil {
			ref.Properties = &armcompute.NetworkInterfaceReferenceProperties{}
		}
		primary := (i == 0)
		ref.Properties.Primary = &primary
	}
	falseVal := false
	if isPrimary { falseVal = true } // only NIC → primary
	vm.Properties.NetworkProfile.NetworkInterfaces = append(
		vm.Properties.NetworkProfile.NetworkInterfaces,
		&armcompute.NetworkInterfaceReference{
			ID: nicARMID,
			Properties: &armcompute.NetworkInterfaceReferenceProperties{Primary: &falseVal},
		},
	)

	// 4) Update VM
	updatePoller, err := nicHandler.VMClient.BeginCreateOrUpdate(nicHandler.Ctx, nicHandler.Region.Region, vmName, vm, nil)
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("AttachNIC: failed to update VM NetworkProfile: %w", err)
	}
	if _, err = updatePoller.PollUntilDone(nicHandler.Ctx, nil); err != nil {
		return irs.NICInfo{}, fmt.Errorf("AttachNIC: VM update failed: %w", err)
	}

	// 5) Start VM
	startPoller, err := nicHandler.VMClient.BeginStart(nicHandler.Ctx, nicHandler.Region.Region, vmName, nil)
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("AttachNIC: failed to start VM: %w", err)
	}
	if _, err = startPoller.PollUntilDone(nicHandler.Ctx, nil); err != nil {
		return irs.NICInfo{}, fmt.Errorf("AttachNIC: VM start failed: %w", err)
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	LoggingInfo(hiscallInfo, start)
	return nicHandler.GetNIC(nicIID)
}

// DetachNIC detaches a NIC from its VM via Azure VM NetworkProfile update.
// Azure requires: deallocate VM → remove NIC → start VM.
func (nicHandler *AzureNICHandler) DetachNIC(nicIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region, call.NIC, nicIID.NameId, "DetachNIC()")
	start := call.Start()

	// Resolve NIC name and get ARM ID
	nicName := nicIID.NameId
	if nicName == "" && nicIID.SystemId != "" {
		parts := strings.Split(nicIID.SystemId, "/")
		nicName = parts[len(parts)-1]
	}
	nicResp, err := nicHandler.NicClient.Get(nicHandler.Ctx, nicHandler.Region.Region, nicName, nil)
	if err != nil {
		return false, fmt.Errorf("DetachNIC: failed to get NIC %s: %w", nicName, err)
	}
	nicARMID := *nicResp.ID

	// Find attached VM via NIC's VirtualMachine reference
	if nicResp.Properties == nil || nicResp.Properties.VirtualMachine == nil || nicResp.Properties.VirtualMachine.ID == nil {
		return false, fmt.Errorf("DetachNIC: NIC %s is not attached to any VM", nicName)
	}
	vmParts := strings.Split(*nicResp.Properties.VirtualMachine.ID, "/")
	vmName := vmParts[len(vmParts)-1]

	// 1) Deallocate VM
	deallocPoller, err := nicHandler.VMClient.BeginDeallocate(nicHandler.Ctx, nicHandler.Region.Region, vmName, nil)
	if err != nil {
		return false, fmt.Errorf("DetachNIC: failed to start VM deallocation: %w", err)
	}
	if _, err = deallocPoller.PollUntilDone(nicHandler.Ctx, nil); err != nil {
		return false, fmt.Errorf("DetachNIC: VM deallocation failed: %w", err)
	}

	// 2) Get current VM config
	vmResp, err := nicHandler.VMClient.Get(nicHandler.Ctx, nicHandler.Region.Region, vmName, nil)
	if err != nil {
		return false, fmt.Errorf("DetachNIC: failed to get VM %s: %w", vmName, err)
	}
	vm := vmResp.VirtualMachine

	// 3) Remove this NIC from NetworkProfile (keep primary NIC)
	if vm.Properties == nil || vm.Properties.NetworkProfile == nil {
		return false, fmt.Errorf("DetachNIC: VM %s has no NetworkProfile", vmName)
	}
	var remaining []*armcompute.NetworkInterfaceReference
	for _, ref := range vm.Properties.NetworkProfile.NetworkInterfaces {
		if ref.ID != nil && !strings.EqualFold(*ref.ID, nicARMID) {
			remaining = append(remaining, ref)
		}
	}
	if len(remaining) == len(vm.Properties.NetworkProfile.NetworkInterfaces) {
		return false, fmt.Errorf("DetachNIC: NIC %s not found in VM %s NetworkProfile", nicName, vmName)
	}
	vm.Properties.NetworkProfile.NetworkInterfaces = remaining

	// 4) Update VM
	updatePoller, err := nicHandler.VMClient.BeginCreateOrUpdate(nicHandler.Ctx, nicHandler.Region.Region, vmName, vm, nil)
	if err != nil {
		return false, fmt.Errorf("DetachNIC: failed to update VM NetworkProfile: %w", err)
	}
	if _, err = updatePoller.PollUntilDone(nicHandler.Ctx, nil); err != nil {
		return false, fmt.Errorf("DetachNIC: VM update failed: %w", err)
	}

	// 5) Start VM
	startPoller, err := nicHandler.VMClient.BeginStart(nicHandler.Ctx, nicHandler.Region.Region, vmName, nil)
	if err != nil {
		return false, fmt.Errorf("DetachNIC: failed to start VM: %w", err)
	}
	if _, err = startPoller.PollUntilDone(nicHandler.Ctx, nil); err != nil {
		return false, fmt.Errorf("DetachNIC: VM start failed: %w", err)
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// extractAzureNICInfo converts an *armnetwork.Interface to irs.NICInfo.
func extractAzureNICInfo(nic *armnetwork.Interface) irs.NICInfo {
	info := irs.NICInfo{}

	if nic == nil {
		return info
	}

	if nic.Name != nil {
		info.IId.NameId = *nic.Name
	}
	if nic.ID != nil {
		info.IId.SystemId = *nic.ID
	}

	// Default status
	info.Status = irs.NICAvailable

	if nic.Properties != nil {
		// Provisioning state / status
		if nic.Properties.ProvisioningState != nil {
			state := string(*nic.Properties.ProvisioningState)
			switch strings.ToLower(state) {
			case "deleting":
				info.Status = irs.NICDeleting
			case "failed":
				info.Status = irs.NICError
			default:
				info.Status = irs.NICAvailable
			}
		}

		// If attached to a VM, mark as attached
		if nic.Properties.VirtualMachine != nil && nic.Properties.VirtualMachine.ID != nil {
			info.Status = irs.NICAttached
			vmID := *nic.Properties.VirtualMachine.ID
			parts := strings.Split(vmID, "/")
			vmName := parts[len(parts)-1]
			info.OwnerVM = irs.IID{NameId: vmName, SystemId: vmID}
		}

		// MAC address
		if nic.Properties.MacAddress != nil {
			info.MACAddress = *nic.Properties.MacAddress
		}

		// NSG (security groups)
		if nic.Properties.NetworkSecurityGroup != nil && nic.Properties.NetworkSecurityGroup.ID != nil {
			nsgID := *nic.Properties.NetworkSecurityGroup.ID
			parts := strings.Split(nsgID, "/")
			nsgName := parts[len(parts)-1]
			info.SecurityGroupIIDs = []irs.IID{{NameId: nsgName, SystemId: nsgID}}
		}

		// IP configurations — build index-aligned PrivateIPs[] / PublicIPs[] parallel arrays.
		for i, ipCfg := range nic.Properties.IPConfigurations {
			if ipCfg == nil || ipCfg.Properties == nil {
				continue
			}
			props := ipCfg.Properties

			// Private IPs
			privateIP := ""
			if props.PrivateIPAddress != nil {
				privateIP = *props.PrivateIPAddress
				info.PrivateIPs = append(info.PrivateIPs, privateIP)
				if i == 0 {
					info.PrivateIP = privateIP
				}
			}

			// Public IP (index-aligned with PrivateIPs)
			pubIP := ""
			if props.PublicIPAddress != nil && props.PublicIPAddress.Properties != nil &&
				props.PublicIPAddress.Properties.IPAddress != nil {
				pubIP = *props.PublicIPAddress.Properties.IPAddress
			}
			info.PublicIPs = append(info.PublicIPs, pubIP)
			if i == 0 && pubIP != "" {
				info.PublicIP = pubIP
			}

			// Subnet / VPC (first config only)
			if i == 0 && props.Subnet != nil && props.Subnet.ID != nil {
				subnetID := *props.Subnet.ID
				// /subscriptions/.../resourceGroups/.../providers/Microsoft.Network/virtualNetworks/<vnet>/subnets/<subnet>
				parts := strings.Split(subnetID, "/")
				subnetName := parts[len(parts)-1]
				info.SubnetIID = irs.IID{NameId: subnetName, SystemId: subnetID}

				// VPC is everything up to and excluding /subnets/<subnet>
				if len(parts) >= 3 {
					vpcParts := parts[:len(parts)-2]
					vpcName := vpcParts[len(vpcParts)-1]
					info.VpcIID = irs.IID{NameId: vpcName, SystemId: strings.Join(vpcParts, "/")}
				}
			}
		}
	}

	// CreatedTime: Azure NIC does not expose a creation timestamp; use zero value.
	info.CreatedTime = time.Time{}

	// Tags
	if nic.Tags != nil {
		for k, v := range nic.Tags {
			val := ""
			if v != nil {
				val = *v
			}
			info.TagList = append(info.TagList, irs.KeyValue{Key: k, Value: val})
		}
	}

	return info
}

// AddPrivateIP adds a secondary private IP to an Azure NIC by appending a new IPConfiguration.
func (h *AzureNICHandler) AddPrivateIP(nicIID irs.IID, privateIP string) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicIID.NameId, "AddPrivateIP(BeginCreateOrUpdate)")
	start := call.Start()

	nicName := nicIID.NameId
	if nicName == "" && nicIID.SystemId != "" {
		parts := strings.Split(nicIID.SystemId, "/")
		nicName = parts[len(parts)-1]
	}

	resp, err := h.NicClient.Get(h.Ctx, h.Region.Region, nicName, nil)
	if err != nil {
		getErr := fmt.Errorf("AddPrivateIP: failed to get NIC %s: %w", nicName, err)
		LoggingError(hiscallInfo, getErr)
		return irs.NICInfo{}, getErr
	}
	nic := resp.Interface

	if nic.Properties == nil {
		return irs.NICInfo{}, fmt.Errorf("AddPrivateIP: NIC %s has no properties", nicName)
	}

	// Build new IP configuration name
	newConfigName := fmt.Sprintf("ipConfig-%d", len(nic.Properties.IPConfigurations)+1)

	// Determine subnet from primary config
	var subnetID *string
	for _, cfg := range nic.Properties.IPConfigurations {
		if cfg.Properties != nil && cfg.Properties.Subnet != nil && cfg.Properties.Subnet.ID != nil {
			subnetID = cfg.Properties.Subnet.ID
			break
		}
	}
	if subnetID == nil {
		return irs.NICInfo{}, fmt.Errorf("AddPrivateIP: could not determine subnet for NIC %s", nicName)
	}

	dynMethod := armnetwork.IPAllocationMethodDynamic
	staticMethod := armnetwork.IPAllocationMethodStatic
	newCfg := &armnetwork.InterfaceIPConfiguration{
		Name: &newConfigName,
		Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
			Subnet:                    &armnetwork.Subnet{ID: subnetID},
			PrivateIPAllocationMethod: &dynMethod,
		},
	}
	if privateIP != "" {
		newCfg.Properties.PrivateIPAddress = &privateIP
		newCfg.Properties.PrivateIPAllocationMethod = &staticMethod
	}

	nic.Properties.IPConfigurations = append(nic.Properties.IPConfigurations, newCfg)

	poller, err := h.NicClient.BeginCreateOrUpdate(h.Ctx, h.Region.Region, nicName, nic, nil)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		updateErr := fmt.Errorf("AddPrivateIP: failed to update NIC %s: %w", nicName, err)
		LoggingError(hiscallInfo, updateErr)
		return irs.NICInfo{}, updateErr
	}
	result, err := poller.PollUntilDone(h.Ctx, nil)
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("AddPrivateIP: polling failed for NIC %s: %w", nicName, err)
	}
	LoggingInfo(hiscallInfo, start)
	return extractAzureNICInfo(&result.Interface), nil
}

// RemovePrivateIP removes a secondary private IP from an Azure NIC by deleting its IPConfiguration.
func (h *AzureNICHandler) RemovePrivateIP(nicIID irs.IID, privateIP string) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicIID.NameId, "RemovePrivateIP(BeginCreateOrUpdate)")
	start := call.Start()

	nicName := nicIID.NameId
	if nicName == "" && nicIID.SystemId != "" {
		parts := strings.Split(nicIID.SystemId, "/")
		nicName = parts[len(parts)-1]
	}

	resp, err := h.NicClient.Get(h.Ctx, h.Region.Region, nicName, nil)
	if err != nil {
		getErr := fmt.Errorf("RemovePrivateIP: failed to get NIC %s: %w", nicName, err)
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	nic := resp.Interface

	if nic.Properties == nil {
		return false, fmt.Errorf("RemovePrivateIP: NIC %s has no properties", nicName)
	}

	// Find and remove the IP configuration that holds privateIP
	found := false
	var updatedConfigs []*armnetwork.InterfaceIPConfiguration
	for _, cfg := range nic.Properties.IPConfigurations {
		if cfg.Properties != nil && cfg.Properties.PrivateIPAddress != nil && *cfg.Properties.PrivateIPAddress == privateIP {
			// Skip primary IP configuration
			if cfg.Properties.Primary != nil && *cfg.Properties.Primary {
				return false, fmt.Errorf("RemovePrivateIP: cannot remove primary IP %s from NIC %s", privateIP, nicName)
			}
			found = true
			continue
		}
		updatedConfigs = append(updatedConfigs, cfg)
	}
	if !found {
		return false, fmt.Errorf("RemovePrivateIP: private IP %s not found on NIC %s", privateIP, nicName)
	}

	nic.Properties.IPConfigurations = updatedConfigs

	poller, err := h.NicClient.BeginCreateOrUpdate(h.Ctx, h.Region.Region, nicName, nic, nil)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		updateErr := fmt.Errorf("RemovePrivateIP: failed to update NIC %s: %w", nicName, err)
		LoggingError(hiscallInfo, updateErr)
		return false, updateErr
	}
	if _, err = poller.PollUntilDone(h.Ctx, nil); err != nil {
		return false, fmt.Errorf("RemovePrivateIP: polling failed for NIC %s: %w", nicName, err)
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// GetNICOSConfigScript returns a bash script that must be run inside the Azure VM OS
// after an additional NIC is attached. Azure does not configure secondary NICs automatically;
// the guest OS must configure policy-based routing so that replies from the secondary NIC's
// public IP are sent back through the correct interface.
func (nicHandler *AzureNICHandler) GetNICOSConfigScript(nicIID irs.IID) (string, error) {
	nicInfo, err := nicHandler.GetNIC(nicIID)
	if err != nil {
		return "", fmt.Errorf("GetNICOSConfigScript: failed to get NIC info: %w", err)
	}

	privateIPs := nicInfo.PrivateIPs
	if len(privateIPs) == 0 && nicInfo.PrivateIP != "" {
		privateIPs = []string{nicInfo.PrivateIP}
	}
	if len(privateIPs) == 0 {
		return "", fmt.Errorf("GetNICOSConfigScript: NIC has no private IPs")
	}

	mac := strings.ToLower(strings.ReplaceAll(nicInfo.MACAddress, "-", ":"))
	ip0 := privateIPs[0]
	lastDot := strings.LastIndex(ip0, ".")
	gateway := ip0[:lastDot] + ".1"
	cidr := ip0[:lastDot] + ".0/24"

	script := "# 1. Identify interface name by MAC\n" +
		"IFACE=$(ip link | grep -B1 \"" + mac + "\" | head -1 | awk -F': ' '{print $2}')\n" +
		"echo \"Target interface: $IFACE\"\n"

	script += "\n# 2. Configure Policy-based Routing (PBR)\n"
	script += "sudo ip route flush table 101 2>/dev/null || true\n"
	for _, ip := range privateIPs {
		script += "sudo ip rule del from " + ip + " lookup 101 2>/dev/null || true\n"
	}
	script += "sudo ip route add " + cidr + " dev $IFACE src " + ip0 + " table 101\n"
	script += "sudo ip route add default via " + gateway + " dev $IFACE table 101\n"
	for _, ip := range privateIPs {
		script += "sudo ip rule add from " + ip + " lookup 101\n"
	}

	return script, nil
}
