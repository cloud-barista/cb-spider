// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// Azure Public IP Address Handler
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzurePublicIPHandler struct {
	Region         idrv.RegionInfo
	Ctx            context.Context
	PublicIPClient *armnetwork.PublicIPAddressesClient
	NicClient      *armnetwork.InterfacesClient
}

func (h *AzurePublicIPHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, "ListIID", "ListByResourceGroup()")
	start := call.Start()

	pager := h.PublicIPClient.NewListPager(h.Region.Region, nil)
	var iidList []*irs.IID
	for pager.More() {
		page, err := pager.NextPage(h.Ctx)
		if err != nil {
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		for _, pip := range page.Value {
			nameId := azurePublicIPNameId(pip)
			iidList = append(iidList, &irs.IID{NameId: nameId, SystemId: *pip.ID})
		}
	}
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

func (h *AzurePublicIPHandler) CreatePublicIP(reqInfo irs.PublicIPInfo) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, reqInfo.IId.NameId, "BeginCreateOrUpdate()")
	start := call.Start()

	tags := make(map[string]*string)
	tags["Name"] = &reqInfo.IId.NameId
	for _, kv := range reqInfo.TagList {
		v := kv.Value
		tags[kv.Key] = &v
	}

	sku := armnetwork.PublicIPAddressSKU{
		Name: toPtr(armnetwork.PublicIPAddressSKUNameStandard),
	}
	params := armnetwork.PublicIPAddress{
		Location: &h.Region.Region,
		SKU:      &sku,
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: toPtr(armnetwork.IPAllocationMethodStatic),
		},
		Tags: tags,
	}

	poller, err := h.PublicIPClient.BeginCreateOrUpdate(h.Ctx, h.Region.Region, reqInfo.IId.NameId, params, nil)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	resp, err := poller.PollUntilDone(h.Ctx, nil)
	if err != nil {
		cblogger.Error(err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	info := extractAzurePublicIPInfo(&resp.PublicIPAddress)
	h.resolveAzurePIPPrivateIP(&info)
	return info, nil
}

func (h *AzurePublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, "All", "ListByResourceGroup()")
	start := call.Start()

	pager := h.PublicIPClient.NewListPager(h.Region.Region, nil)
	var infoList []*irs.PublicIPInfo
	for pager.More() {
		page, err := pager.NextPage(h.Ctx)
		if err != nil {
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		for _, pip := range page.Value {
			info := extractAzurePublicIPInfo(pip)
			h.resolveAzurePIPPrivateIP(&info)
			infoList = append(infoList, &info)
		}
	}
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	LoggingInfo(hiscallInfo, start)

	if infoList == nil {
		infoList = []*irs.PublicIPInfo{}
	}
	return infoList, nil
}

func (h *AzurePublicIPHandler) GetPublicIP(publicIPIID irs.IID) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "Get()")
	start := call.Start()

	// Resolve resource name from SystemId if needed
	resourceName := publicIPIID.NameId
	if publicIPIID.SystemId != "" {
		parts := strings.Split(publicIPIID.SystemId, "/")
		resourceName = parts[len(parts)-1]
	}

	resp, err := h.PublicIPClient.Get(h.Ctx, h.Region.Region, resourceName, nil)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	info := extractAzurePublicIPInfo(&resp.PublicIPAddress)
	if publicIPIID.NameId != "" {
		info.IId.NameId = publicIPIID.NameId
	}
	h.resolveAzurePIPPrivateIP(&info)
	return info, nil
}

func (h *AzurePublicIPHandler) DeletePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "BeginDelete()")
	start := call.Start()

	resourceName := publicIPIID.NameId
	if publicIPIID.SystemId != "" {
		parts := strings.Split(publicIPIID.SystemId, "/")
		resourceName = parts[len(parts)-1]
	}

	poller, err := h.PublicIPClient.BeginDelete(h.Ctx, h.Region.Region, resourceName, nil)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	_, err = poller.PollUntilDone(h.Ctx, nil)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func extractAzurePublicIPInfo(pip *armnetwork.PublicIPAddress) irs.PublicIPInfo {
	if pip == nil {
		return irs.PublicIPInfo{}
	}
	info := irs.PublicIPInfo{
		IId: irs.IID{
			NameId:   derefStr(pip.Name),
			SystemId: derefStr(pip.ID),
		},
		PublicIPAddress: "NA",
		Status:          irs.PublicIPAvailable,
		CreatedTime:     time.Time{},
	}

	// Name tag override
	if pip.Tags != nil {
		if v, ok := pip.Tags["Name"]; ok && v != nil {
			info.IId.NameId = *v
		}
	}

	if pip.Properties != nil {
		if pip.Properties.IPAddress != nil {
			info.PublicIPAddress = *pip.Properties.IPAddress
		}
		if pip.Properties.IPConfiguration != nil {
			info.Status = irs.PublicIPAssociated
			if pip.Properties.IPConfiguration.ID != nil {
				ipConfigID := *pip.Properties.IPConfiguration.ID
				// ipConfigID: /subscriptions/.../networkInterfaces/{nic}/ipConfigurations/{cfg}
				nicName := extractAzureVMNameFromIPConfig(ipConfigID)
				// Build NIC ARM ID (strip /ipConfigurations/{cfg} suffix)
				nicSystemID := extractAzureNICIDFromIPConfig(ipConfigID)
				info.OwnedNIC = irs.IID{NameId: nicName, SystemId: nicSystemID}
				// Also keep OwnedVM pointing at the NIC for DisassociatePublicIP fallback
				info.OwnedVM = irs.IID{NameId: nicName, SystemId: ipConfigID}
			}
		}
	}

	var tagList []irs.KeyValue
	for k, v := range pip.Tags {
		if k != "Name" && v != nil {
			tagList = append(tagList, irs.KeyValue{Key: k, Value: *v})
		}
	}
	info.TagList = tagList

	kvList := []irs.KeyValue{
		{Key: "Location", Value: derefStr(pip.Location)},
	}
	if pip.SKU != nil && pip.SKU.Name != nil {
		kvList = append(kvList, irs.KeyValue{Key: "SKU", Value: string(*pip.SKU.Name)})
	}
	info.KeyValueList = kvList

	return info
}

func azurePublicIPNameId(pip *armnetwork.PublicIPAddress) string {
	if pip.Tags != nil {
		if v, ok := pip.Tags["Name"]; ok && v != nil {
			return *v
		}
	}
	return derefStr(pip.Name)
}

// resolveAzurePIPPrivateIP fetches the NIC to find the PrivateIPAddress of the IP config
// that this PublicIP is attached to, and populates info.OwnedPrivateIP.
// OwnedVM.SystemId holds the full IPConfig ARM ID:
//   /subscriptions/.../networkInterfaces/{nic}/ipConfigurations/{cfg}
func (h *AzurePublicIPHandler) resolveAzurePIPPrivateIP(info *irs.PublicIPInfo) {
	if info.Status != irs.PublicIPAssociated || info.OwnedVM.SystemId == "" {
		return
	}
	ipConfigID := info.OwnedVM.SystemId

	// Extract NIC name and IPConfig name from the ARM path
	parts := strings.Split(ipConfigID, "/")
	nicName := ""
	ipConfigName := ""
	for i, p := range parts {
		if strings.EqualFold(p, "networkInterfaces") && i+1 < len(parts) {
			nicName = parts[i+1]
		}
		if strings.EqualFold(p, "ipConfigurations") && i+1 < len(parts) {
			ipConfigName = parts[i+1]
		}
	}
	if nicName == "" {
		return
	}

	// Azure NIC GET always returns IPConfigurations with PrivateIPAddress — no expand needed.
	resp, err := h.NicClient.Get(h.Ctx, h.Region.Region, nicName, nil)
	if err != nil {
		return
	}
	if resp.Properties == nil {
		return
	}

	for _, cfg := range resp.Properties.IPConfigurations {
		if cfg == nil || cfg.Properties == nil || cfg.Name == nil {
			continue
		}
		if ipConfigName == "" || strings.EqualFold(*cfg.Name, ipConfigName) {
			if cfg.Properties.PrivateIPAddress != nil {
				info.OwnedPrivateIP = *cfg.Properties.PrivateIPAddress
				return
			}
		}
	}
}

func extractAzureVMNameFromIPConfig(ipConfigID string) string {
	// /subscriptions/.../resourceGroups/.../providers/Microsoft.Network/networkInterfaces/{nic}/ipConfigurations/{config}
	parts := strings.Split(ipConfigID, "/")
	for i, p := range parts {
		if strings.EqualFold(p, "networkInterfaces") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return "NA"
}

// extractAzureNICIDFromIPConfig returns the NIC ARM ID by stripping /ipConfigurations/{cfg} from the ipConfigID.
func extractAzureNICIDFromIPConfig(ipConfigID string) string {
	parts := strings.Split(ipConfigID, "/")
	for i, p := range parts {
		if strings.EqualFold(p, "ipConfigurations") && i >= 1 {
			return strings.Join(parts[:i], "/")
		}
	}
	return ipConfigID
}

func toPtr[T any](v T) *T { return &v }

func derefStr(s *string) string {
	if s == nil {
		return "NA"
	}
	return *s
}

// AssociatePublicIP associates a Public IP with a VM's NIC IP configuration.
func (h *AzurePublicIPHandler) AssociatePublicIP(publicIPIID irs.IID, vmIID irs.IID, nicIID irs.IID, privateIP string) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "AssociatePublicIP()")
	start := call.Start()

	// Resolve resource name
	resourceName := publicIPIID.NameId
	if publicIPIID.SystemId != "" {
		parts := strings.Split(publicIPIID.SystemId, "/")
		resourceName = parts[len(parts)-1]
	}

	// Get PublicIP resource to obtain its ID
	pipResp, err := h.PublicIPClient.Get(h.Ctx, h.Region.Region, resourceName, nil)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	pipID := pipResp.ID

	// Resolve the Azure NIC resource name:
	// 1. From nicIID.SystemId (ARM path like /.../networkInterfaces/<name>) — most reliable
	// 2. From nicIID.NameId (Spider name — may differ from Azure internal name)
	// 3. Fallback: vmIID.NameId + "-nic" (legacy path when no nicIID provided)
	vmNicName := ""
	if nicIID.SystemId != "" {
		parts := strings.Split(nicIID.SystemId, "/")
		vmNicName = parts[len(parts)-1]
	} else if nicIID.NameId != "" {
		vmNicName = nicIID.NameId
	} else {
		vmNicName = vmIID.NameId + "-nic"
	}

	nic, err := h.NicClient.Get(h.Ctx, h.Region.Region, vmNicName, nil)
	if err != nil {
		return irs.PublicIPInfo{}, fmt.Errorf("failed to get NIC for VM %s: %w", vmIID.NameId, err)
	}

	if nic.Properties == nil || len(nic.Properties.IPConfigurations) == 0 {
		return irs.PublicIPInfo{}, fmt.Errorf("VM %s has no IP configurations", vmIID.NameId)
	}

	// Associate with specific privateIP if given; otherwise use the first IP config
	targetCfgIdx := 0
	if privateIP != "" {
		for i, cfg := range nic.Properties.IPConfigurations {
			if cfg.Properties != nil && cfg.Properties.PrivateIPAddress != nil && *cfg.Properties.PrivateIPAddress == privateIP {
				targetCfgIdx = i
				break
			}
		}
	}
	nic.Properties.IPConfigurations[targetCfgIdx].Properties.PublicIPAddress = &armnetwork.PublicIPAddress{ID: pipID}

	poller, err := h.NicClient.BeginCreateOrUpdate(h.Ctx, h.Region.Region, vmNicName, nic.Interface, nil)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	if _, err = poller.PollUntilDone(h.Ctx, nil); err != nil {
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	return h.GetPublicIP(publicIPIID)
}

// DisassociatePublicIP removes the Public IP from a VM's NIC IP configuration.
func (h *AzurePublicIPHandler) DisassociatePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "DisassociatePublicIP()")
	start := call.Start()

	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return false, err
	}
	if info.Status != irs.PublicIPAssociated || info.OwnedVM.NameId == "" {
		return false, fmt.Errorf("PublicIP %s is not associated with any VM", publicIPIID.NameId)
	}

	// OwnedVM.SystemId = full IPConfig ARM path: .../networkInterfaces/{nic}/ipConfigurations/{cfg}
	// OwnedVM.NameId  = Azure NIC resource name
	vmNicName := info.OwnedVM.NameId
	if vmNicName == "" && info.OwnedNIC.NameId != "" {
		vmNicName = info.OwnedNIC.NameId
	}
	nic, err := h.NicClient.Get(h.Ctx, h.Region.Region, vmNicName, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get NIC: %w", err)
	}

	// Extract the specific ipConfig name from the ARM path so we remove PublicIP from the right config.
	targetCfgName := ""
	if info.OwnedVM.SystemId != "" {
		parts := strings.Split(info.OwnedVM.SystemId, "/")
		for i, p := range parts {
			if strings.EqualFold(p, "ipConfigurations") && i+1 < len(parts) {
				targetCfgName = parts[i+1]
			}
		}
	}

	removed := false
	if nic.Properties != nil {
		for _, cfg := range nic.Properties.IPConfigurations {
			if cfg == nil || cfg.Properties == nil || cfg.Name == nil {
				continue
			}
			if targetCfgName == "" || strings.EqualFold(*cfg.Name, targetCfgName) {
				cfg.Properties.PublicIPAddress = nil
				removed = true
				break
			}
		}
	}
	if !removed {
		return false, fmt.Errorf("IPConfig %q not found on NIC %s", targetCfgName, vmNicName)
	}

	poller, err := h.NicClient.BeginCreateOrUpdate(h.Ctx, h.Region.Region, vmNicName, nic.Interface, nil)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	if _, err = poller.PollUntilDone(h.Ctx, nil); err != nil {
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}
