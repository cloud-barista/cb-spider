// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// IBM Cloud Floating IP Handler (VPC)
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type IbmPublicIPHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	VpcService *vpcv1.VpcV1
}

// Fix 4: ListIID with cursor-based pagination
func (h *IbmPublicIPHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, "ListIID", "ListFloatingIps()")
	start := call.Start()

	var iidList []*irs.IID
	var startToken *string
	for {
		options := &vpcv1.ListFloatingIpsOptions{}
		if startToken != nil {
			options.Start = startToken
		}
		result, _, err := h.VpcService.ListFloatingIpsWithContext(h.Ctx, options)
		if err != nil {
			hiscallInfo.ElapsedTime = call.Elapsed(start)
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		for _, fip := range result.FloatingIps {
			iidList = append(iidList, &irs.IID{NameId: *fip.Name, SystemId: *fip.ID})
		}
		next, err := result.GetNextStart()
		if err != nil || next == nil {
			break
		}
		startToken = next
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

func (h *IbmPublicIPHandler) CreatePublicIP(reqInfo irs.PublicIPInfo) (irs.PublicIPInfo, error) {
	// IBM VPC reserves names beginning with "ibm-" for provider-owned resources.
	if strings.HasPrefix(reqInfo.IId.NameId, "ibm-") {
		reqInfo.IId.NameId = "cb-" + reqInfo.IId.NameId[4:]
	}

	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, reqInfo.IId.NameId, "CreateFloatingIp()")
	start := call.Start()

	prototype := &vpcv1.FloatingIPPrototypeFloatingIPByZone{
		Name: &reqInfo.IId.NameId,
		Zone: &vpcv1.ZoneIdentityByName{Name: &h.Region.Zone},
	}

	options := &vpcv1.CreateFloatingIPOptions{
		FloatingIPPrototype: prototype,
	}

	fip, _, err := h.VpcService.CreateFloatingIPWithContext(h.Ctx, options)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	return extractIbmPublicIPInfo(fip), nil
}

// Fix 4: ListPublicIP with cursor-based pagination
func (h *IbmPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, "All", "ListFloatingIps()")
	start := call.Start()

	var infoList []*irs.PublicIPInfo
	var startToken *string
	for {
		options := &vpcv1.ListFloatingIpsOptions{}
		if startToken != nil {
			options.Start = startToken
		}
		result, _, err := h.VpcService.ListFloatingIpsWithContext(h.Ctx, options)
		if err != nil {
			hiscallInfo.ElapsedTime = call.Elapsed(start)
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		for _, fip := range result.FloatingIps {
			info := extractIbmPublicIPInfo(&fip)
			infoList = append(infoList, &info)
		}
		next, err := result.GetNextStart()
		if err != nil || next == nil {
			break
		}
		startToken = next
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	LoggingInfo(hiscallInfo, start)
	if infoList == nil {
		infoList = []*irs.PublicIPInfo{}
	}
	return infoList, nil
}

// Fix 4: GetPublicIP with pagination in name-search path
func (h *IbmPublicIPHandler) GetPublicIP(publicIPIID irs.IID) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "GetFloatingIp()")
	start := call.Start()

	var fip *vpcv1.FloatingIP
	var err error

	if publicIPIID.SystemId != "" {
		options := &vpcv1.GetFloatingIPOptions{ID: &publicIPIID.SystemId}
		fip, _, err = h.VpcService.GetFloatingIPWithContext(h.Ctx, options)
	} else {
		// Search by name with pagination
		var startToken *string
	searchLoop:
		for {
			listOptions := &vpcv1.ListFloatingIpsOptions{}
			if startToken != nil {
				listOptions.Start = startToken
			}
			result, _, listErr := h.VpcService.ListFloatingIpsWithContext(h.Ctx, listOptions)
			if listErr != nil {
				err = listErr
				break
			}
			for i, f := range result.FloatingIps {
				if *f.Name == publicIPIID.NameId {
					fip = &result.FloatingIps[i]
					break searchLoop
				}
			}
			next, nextErr := result.GetNextStart()
			if nextErr != nil || next == nil {
				break
			}
			startToken = next
		}
		if err == nil && fip == nil {
			err = fmt.Errorf("PublicIP not found: %s", publicIPIID.NameId)
		}
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	info := extractIbmPublicIPInfo(fip)
	if publicIPIID.NameId != "" {
		info.IId.NameId = publicIPIID.NameId
	}
	return info, nil
}

func (h *IbmPublicIPHandler) DeletePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "DeleteFloatingIp()")
	start := call.Start()

	systemId := publicIPIID.SystemId
	if systemId == "" {
		info, err := h.GetPublicIP(publicIPIID)
		if err != nil {
			return false, err
		}
		systemId = info.IId.SystemId
	}

	options := &vpcv1.DeleteFloatingIPOptions{ID: &systemId}
	_, err := h.VpcService.DeleteFloatingIPWithContext(h.Ctx, options)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

// Fix 1 & 2: extractIbmPublicIPInfo - OwnedNIC, OwnedPrivateIP, status based on Target presence
func extractIbmPublicIPInfo(fip *vpcv1.FloatingIP) irs.PublicIPInfo {
	if fip == nil {
		return irs.PublicIPInfo{}
	}

	// Fix 2: Status determined by Target presence, not status field.
	// "pending"/"deleting" are lifecycle states; association is indicated by Target != nil.
	status := irs.PublicIPAvailable
	if fip.Target != nil {
		status = irs.PublicIPAssociated
	}

	createdTime := time.Time{}
	if fip.CreatedAt != nil {
		createdTime = time.Time(*fip.CreatedAt)
	}

	info := irs.PublicIPInfo{
		IId:             irs.IID{NameId: *fip.Name, SystemId: *fip.ID},
		PublicIPAddress: *fip.Address,
		Status:          status,
		CreatedTime:     createdTime,
	}

	// Fix 1: Extract OwnedNIC and OwnedPrivateIP from Target.
	// IBM SDK may deserialize the target as the base *FloatingIPTarget struct (not a concrete subtype).
	// All concrete types (NetworkInterfaceReference, VirtualNetworkInterfaceReference) share the same
	// fields (ID, Name, PrimaryIP) in the base struct, so we handle *FloatingIPTarget directly.
	if fip.Target != nil {
		switch t := fip.Target.(type) {
		case *vpcv1.FloatingIPTargetNetworkInterfaceReference:
			info.OwnedNIC = irs.IID{NameId: ibmDerefStr(t.Name), SystemId: ibmDerefStr(t.ID)}
			if t.PrimaryIP != nil && t.PrimaryIP.Address != nil {
				info.OwnedPrivateIP = *t.PrimaryIP.Address
			}
		case *vpcv1.FloatingIPTargetVirtualNetworkInterfaceReference:
			info.OwnedNIC = irs.IID{NameId: ibmDerefStr(t.Name), SystemId: ibmDerefStr(t.ID)}
			if t.PrimaryIP != nil && t.PrimaryIP.Address != nil {
				info.OwnedPrivateIP = *t.PrimaryIP.Address
			}
		case *vpcv1.FloatingIPTargetPublicGatewayReference:
			info.OwnedVM = irs.IID{NameId: ibmDerefStr(t.Name), SystemId: ibmDerefStr(t.ID)}
		case *vpcv1.FloatingIPTarget:
			// Base struct returned when SDK cannot determine concrete subtype.
			// ResourceType: "network_interface" (classic NIC), "virtual_network_interface" (VNI), "public_gateway"
			resType := ibmDerefStr(t.ResourceType)
			if resType == "network_interface" || resType == "virtual_network_interface" || resType == "" {
				info.OwnedNIC = irs.IID{NameId: ibmDerefStr(t.Name), SystemId: ibmDerefStr(t.ID)}
				if t.PrimaryIP != nil && t.PrimaryIP.Address != nil {
					info.OwnedPrivateIP = *t.PrimaryIP.Address
				}
			} else if resType == "public_gateway" {
				info.OwnedVM = irs.IID{NameId: ibmDerefStr(t.Name), SystemId: ibmDerefStr(t.ID)}
			}
		}
	}

	zoneName := "NA"
	if fip.Zone != nil {
		zoneName = *fip.Zone.Name
	}

	ibmStatus := "NA"
	if fip.Status != nil {
		ibmStatus = *fip.Status
	}

	info.KeyValueList = []irs.KeyValue{
		{Key: "Zone", Value: zoneName},
		{Key: "CRN", Value: ibmDerefStr(fip.CRN)},
		{Key: "Href", Value: ibmDerefStr(fip.Href)},
		{Key: "IBMStatus", Value: ibmStatus},
	}

	return info
}

func ibmDerefStr(s *string) string {
	if s == nil {
		return "NA"
	}
	return *s
}

// Fix 3: AssociatePublicIP - uses nicIID when provided; falls back to primary NIC
func (h *IbmPublicIPHandler) AssociatePublicIP(publicIPIID irs.IID, vmIID irs.IID, nicIID irs.IID, privateIP string) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "AddInstanceNetworkInterfaceFloatingIp()")
	start := call.Start()

	// Get floating IP system ID
	pipInfo, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}
	floatingIPId := pipInfo.IId.SystemId

	// Fix 3: If nicIID is provided, use UpdateFloatingIP to bind directly to that NIC/VNI.
	if nicIID.SystemId != "" || nicIID.NameId != "" {
		targetNICId := nicIID.SystemId
		if targetNICId == "" {
			targetNICId = nicIID.NameId
		}
		patch := &vpcv1.FloatingIPPatch{
			Target: &vpcv1.FloatingIPTargetPatch{
				ID: &targetNICId,
			},
		}
		patchMap, patchErr := patch.AsPatch()
		if patchErr != nil {
			return irs.PublicIPInfo{}, fmt.Errorf("failed to build FloatingIP patch: %v", patchErr)
		}
		updateOpts := &vpcv1.UpdateFloatingIPOptions{
			ID:              &floatingIPId,
			FloatingIPPatch: patchMap,
		}
		fip, _, updateErr := h.VpcService.UpdateFloatingIPWithContext(h.Ctx, updateOpts)
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		if updateErr != nil {
			cblogger.Error(updateErr)
			LoggingError(hiscallInfo, updateErr)
			return irs.PublicIPInfo{}, updateErr
		}
		LoggingInfo(hiscallInfo, start)
		return extractIbmPublicIPInfo(fip), nil
	}

	// Bind to the VM's primary VNI via VNI model (UpdateFloatingIP).
	// Classic AddInstanceNetworkInterfaceFloatingIP is NOT used because all new IBM VPC
	// instances are created with the VNI (network attachment) model.
	instanceId := vmIID.SystemId
	if instanceId == "" {
		var startToken *string
	instanceLoop:
		for {
			listOpts := &vpcv1.ListInstancesOptions{}
			if startToken != nil {
				listOpts.Start = startToken
			}
			result, _, listErr := h.VpcService.ListInstancesWithContext(h.Ctx, listOpts)
			if listErr != nil {
				return irs.PublicIPInfo{}, listErr
			}
			for _, inst := range result.Instances {
				if inst.Name != nil && *inst.Name == vmIID.NameId {
					instanceId = *inst.ID
					break instanceLoop
				}
			}
			next, nextErr := result.GetNextStart()
			if nextErr != nil || next == nil {
				break
			}
			startToken = next
		}
	}
	if instanceId == "" {
		return irs.PublicIPInfo{}, fmt.Errorf("VM %s not found", vmIID.NameId)
	}

	// Get the primary VNI ID from the primary network attachment.
	attachments, _, attErr := h.VpcService.ListInstanceNetworkAttachmentsWithContext(h.Ctx, &vpcv1.ListInstanceNetworkAttachmentsOptions{
		InstanceID: &instanceId,
	})
	if attErr != nil || attachments == nil || len(attachments.NetworkAttachments) == 0 {
		return irs.PublicIPInfo{}, fmt.Errorf("no network attachments found for VM %s", instanceId)
	}
	// Primary attachment is the one with Type=="primary"; fall back to first if not tagged.
	primaryVNIId := ""
	for _, att := range attachments.NetworkAttachments {
		if att.VirtualNetworkInterface == nil || att.VirtualNetworkInterface.ID == nil {
			continue
		}
		if att.Type != nil && *att.Type == "primary" {
			primaryVNIId = *att.VirtualNetworkInterface.ID
			break
		}
		if primaryVNIId == "" {
			primaryVNIId = *att.VirtualNetworkInterface.ID
		}
	}
	if primaryVNIId == "" {
		return irs.PublicIPInfo{}, fmt.Errorf("primary VNI not found for VM %s", instanceId)
	}

	patch := &vpcv1.FloatingIPPatch{
		Target: &vpcv1.FloatingIPTargetPatch{
			ID: &primaryVNIId,
		},
	}
	patchMap, patchErr := patch.AsPatch()
	if patchErr != nil {
		return irs.PublicIPInfo{}, fmt.Errorf("failed to build FloatingIP patch: %v", patchErr)
	}
	updateOpts := &vpcv1.UpdateFloatingIPOptions{
		ID:              &floatingIPId,
		FloatingIPPatch: patchMap,
	}
	fip, _, err := h.VpcService.UpdateFloatingIPWithContext(h.Ctx, updateOpts)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	return extractIbmPublicIPInfo(fip), nil
}

// DisassociatePublicIP disassociates a floating IP from its target VNI using the VNI model.
// The floating IP's target (OwnedNIC) is already populated by GetPublicIP/extractIbmPublicIPInfo
// for both classic NIC and VNI targets. We use UpdateFloatingIP with target=nil to detach,
// avoiding the need to search all instances and NICs.
func (h *IbmPublicIPHandler) DisassociatePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "UpdateFloatingIp(disassociate)")
	start := call.Start()

	pipInfo, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return false, err
	}
	if pipInfo.Status != irs.PublicIPAssociated {
		return false, fmt.Errorf("PublicIP %s is not associated", publicIPIID.NameId)
	}
	floatingIPId := pipInfo.IId.SystemId

	// Disassociate via UpdateFloatingIP with target set to nil.
	// This works for both VNI-model targets (virtual_network_interface) and classic NIC targets
	// (network_interface). The OwnedNIC field is populated by extractIbmPublicIPInfo when the
	// floating IP is associated; we do not need it here because IBM accepts target=nil to detach.
	disassociatePatch := map[string]interface{}{
		"target": nil,
	}
	updateOpts := &vpcv1.UpdateFloatingIPOptions{
		ID:              &floatingIPId,
		FloatingIPPatch: disassociatePatch,
	}
	_, _, updateErr := h.VpcService.UpdateFloatingIPWithContext(h.Ctx, updateOpts)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if updateErr != nil {
		cblogger.Error(updateErr)
		LoggingError(hiscallInfo, updateErr)
		return false, updateErr
	}
	LoggingInfo(hiscallInfo, start)

	// Poll until floating IP returns to Available state (target == nil).
	waitErr := h.waitForFloatingIPAvailable(floatingIPId, 30*time.Second)
	if waitErr != nil {
		cblogger.Warn("DisassociatePublicIP: poll timeout waiting for Available state:", waitErr)
	}
	return true, nil
}

// waitForFloatingIPAvailable polls until the floating IP has no target (Available) or timeout.
func (h *IbmPublicIPHandler) waitForFloatingIPAvailable(floatingIPId string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		opts := &vpcv1.GetFloatingIPOptions{ID: &floatingIPId}
		fip, _, err := h.VpcService.GetFloatingIPWithContext(h.Ctx, opts)
		if err != nil {
			return err
		}
		if fip.Target == nil {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for floating IP %s to become Available", floatingIPId)
}
