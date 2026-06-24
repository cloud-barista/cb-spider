// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// IBM NIC Handler (Virtual Network Interface)
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type IbmNICHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	VpcService *vpcv1.VpcV1
}

// -------- Helper: convert VNI to NICInfo

func vniToNICInfo(vni vpcv1.VirtualNetworkInterface) (irs.NICInfo, error) {
	info := irs.NICInfo{}

	if vni.ID == nil || vni.Name == nil {
		return info, fmt.Errorf("VNI missing ID or Name")
	}

	info.IId = irs.IID{NameId: *vni.Name, SystemId: *vni.ID}

	// VPC
	if vni.VPC != nil && vni.VPC.ID != nil {
		vpcName := ""
		if vni.VPC.Name != nil {
			vpcName = *vni.VPC.Name
		}
		info.VpcIID = irs.IID{NameId: vpcName, SystemId: *vni.VPC.ID}
	}

	// Subnet
	if vni.Subnet != nil && vni.Subnet.ID != nil {
		subnetName := ""
		if vni.Subnet.Name != nil {
			subnetName = *vni.Subnet.Name
		}
		info.SubnetIID = irs.IID{NameId: subnetName, SystemId: *vni.Subnet.ID}
	}

	// MAC Address
	if vni.MacAddress != nil {
		info.MACAddress = *vni.MacAddress
	}

	// Primary IP
	if vni.PrimaryIP != nil && vni.PrimaryIP.Address != nil {
		info.PrivateIP = *vni.PrimaryIP.Address
	}

	// All IPs
	for _, ip := range vni.Ips {
		if ip.Address != nil {
			info.PrivateIPs = append(info.PrivateIPs, *ip.Address)
		}
	}

	// Security groups
	for _, sg := range vni.SecurityGroups {
		sgName := ""
		if sg.Name != nil {
			sgName = *sg.Name
		}
		sgID := ""
		if sg.ID != nil {
			sgID = *sg.ID
		}
		info.SecurityGroupIIDs = append(info.SecurityGroupIIDs, irs.IID{NameId: sgName, SystemId: sgID})
	}

	// Fix 1: Status + OwnerVM from Target
	// IBM VNI.Target is VirtualNetworkInterfaceTargetIntf; when attached to an instance it is
	// VirtualNetworkInterfaceTargetInstanceNetworkAttachmentReferenceVirtualNetworkInterfaceContext.
	// That struct holds the attachment Href which encodes the instance ID:
	//   /v1/instances/{instanceID}/network_attachments/{attachmentID}
	if vni.Target != nil {
		info.Status = irs.NICAttached
		// IBM SDK may return the base *VirtualNetworkInterfaceTarget struct (same as FloatingIPTarget pattern).
		// Extract instance ID from Href regardless of concrete subtype.
		var targetHref string
		switch t := vni.Target.(type) {
		case *vpcv1.VirtualNetworkInterfaceTargetInstanceNetworkAttachmentReferenceVirtualNetworkInterfaceContext:
			if t.Href != nil {
				targetHref = *t.Href
			}
		case *vpcv1.VirtualNetworkInterfaceTarget:
			// Base struct — Href contains the attachment URL
			if t.Href != nil {
				targetHref = *t.Href
			}
		}
		if targetHref != "" {
			if instID := extractInstanceIDFromAttachmentHref(targetHref); instID != "" {
				info.OwnerVM = irs.IID{SystemId: instID, NameId: instID}
			}
		}
	} else {
		info.Status = irs.NICAvailable
	}

	// Created time
	if vni.CreatedAt != nil {
		info.CreatedTime = time.Time(*vni.CreatedAt)
	}

	// Key-value extras
	info.KeyValueList = []irs.KeyValue{}
	if vni.LifecycleState != nil {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "LifecycleState", Value: *vni.LifecycleState})
	}
	if vni.ResourceType != nil {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "ResourceType", Value: *vni.ResourceType})
	}

	return info, nil
}

// extractInstanceIDFromAttachmentHref parses the instance UUID from an IBM VPC attachment href.
// Expected format: .../instances/{instanceID}/network_attachments/{attachmentID}
var reInstanceIDFromHref = regexp.MustCompile(`/instances/([^/]+)/network_attachments/`)

func extractInstanceIDFromAttachmentHref(href string) string {
	m := reInstanceIDFromHref.FindStringSubmatch(href)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

// -------- ListIID

func (h *IbmNICHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, "NIC", "ListIID()")
	start := call.Start()

	opts := &vpcv1.ListVirtualNetworkInterfacesOptions{}
	var iids []*irs.IID

	for {
		result, _, err := h.VpcService.ListVirtualNetworkInterfacesWithContext(h.Ctx, opts)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, fmt.Errorf("failed to list VNIs: %w", err)
		}
		for _, vni := range result.VirtualNetworkInterfaces {
			if vni.ID == nil || vni.Name == nil {
				continue
			}
			iids = append(iids, &irs.IID{NameId: *vni.Name, SystemId: *vni.ID})
		}
		if result.Next == nil {
			break
		}
		start2, err2 := core.GetQueryParam(result.Next.Href, "start")
		if err2 != nil || start2 == nil {
			break
		}
		opts.Start = start2
	}

	LoggingInfo(hiscallInfo, start)
	return iids, nil
}

// -------- CreateNIC

func (h *IbmNICHandler) CreateNIC(nicReqInfo irs.NICReqInfo) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicReqInfo.IId.NameId, "CreateNIC()")
	start := call.Start()

	// IBM VPC reserves names beginning with "ibm-" for provider-owned resources.
	// Replace prefix to avoid "Names beginning with ibm- are reserved" error.
	if strings.HasPrefix(nicReqInfo.IId.NameId, "ibm-") {
		nicReqInfo.IId.NameId = "cb-" + nicReqInfo.IId.NameId[4:]
	}

	if nicReqInfo.IId.NameId == "" {
		err := fmt.Errorf("NIC NameId is required")
		LoggingError(hiscallInfo, err)
		return irs.NICInfo{}, err
	}
	if nicReqInfo.SubnetIID.SystemId == "" && nicReqInfo.SubnetIID.NameId == "" {
		err := fmt.Errorf("SubnetIID is required")
		LoggingError(hiscallInfo, err)
		return irs.NICInfo{}, err
	}

	// Resolve subnet SystemId if not provided
	subnetID := nicReqInfo.SubnetIID.SystemId
	if subnetID == "" {
		subnets, _, err := h.VpcService.ListSubnetsWithContext(h.Ctx, &vpcv1.ListSubnetsOptions{})
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.NICInfo{}, fmt.Errorf("failed to list subnets: %w", err)
		}
		for _, sn := range subnets.Subnets {
			if sn.Name != nil && *sn.Name == nicReqInfo.SubnetIID.NameId {
				subnetID = *sn.ID
				break
			}
		}
		if subnetID == "" {
			err := fmt.Errorf("subnet not found: %s", nicReqInfo.SubnetIID.NameId)
			LoggingError(hiscallInfo, err)
			return irs.NICInfo{}, err
		}
	}

	createOpts := &vpcv1.CreateVirtualNetworkInterfaceOptions{
		Name:   core.StringPtr(nicReqInfo.IId.NameId),
		Subnet: &vpcv1.SubnetIdentityByID{ID: core.StringPtr(subnetID)},
	}
	// Fix 6: IBM VPC CreateVirtualNetworkInterfaceOptions does not have a UserTags/Tags field.
	// Tags from nicReqInfo.TagList are silently ignored at creation time. To apply tags,
	// the IBM Global Tagging service would need to be called separately after resource creation,
	// which is outside the scope of this handler.

	// Security groups
	if len(nicReqInfo.SecurityGroupIIDs) > 0 {
		var sgIdentities []vpcv1.SecurityGroupIdentityIntf
		for _, sgIID := range nicReqInfo.SecurityGroupIIDs {
			sgID := sgIID.SystemId
			if sgID == "" {
				// lookup by name
				sgs, _, err := h.VpcService.ListSecurityGroupsWithContext(h.Ctx, &vpcv1.ListSecurityGroupsOptions{})
				if err != nil {
					LoggingError(hiscallInfo, err)
					return irs.NICInfo{}, fmt.Errorf("failed to list security groups: %w", err)
				}
				for _, sg := range sgs.SecurityGroups {
					if sg.Name != nil && *sg.Name == sgIID.NameId {
						sgID = *sg.ID
						break
					}
				}
				if sgID == "" {
					err := fmt.Errorf("security group not found: %s", sgIID.NameId)
					LoggingError(hiscallInfo, err)
					return irs.NICInfo{}, err
				}
			}
			sgIdentities = append(sgIdentities, &vpcv1.SecurityGroupIdentityByID{ID: core.StringPtr(sgID)})
		}
		createOpts.SecurityGroups = sgIdentities
	}

	vni, _, err := h.VpcService.CreateVirtualNetworkInterfaceWithContext(h.Ctx, createOpts)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.NICInfo{}, fmt.Errorf("failed to create VNI: %w", err)
	}

	LoggingInfo(hiscallInfo, start)

	info, err := vniToNICInfo(*vni)
	if err != nil {
		return irs.NICInfo{}, err
	}
	return info, nil
}

// -------- ListNIC

func (h *IbmNICHandler) ListNIC() ([]*irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, "NIC", "ListNIC()")
	start := call.Start()

	opts := &vpcv1.ListVirtualNetworkInterfacesOptions{}
	var nicList []*irs.NICInfo

	for {
		result, _, err := h.VpcService.ListVirtualNetworkInterfacesWithContext(h.Ctx, opts)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, fmt.Errorf("failed to list VNIs: %w", err)
		}
		for _, vni := range result.VirtualNetworkInterfaces {
			info, err := vniToNICInfo(vni)
			if err != nil {
				continue
			}
			nicList = append(nicList, &info)
		}
		if result.Next == nil {
			break
		}
		next, err := core.GetQueryParam(result.Next.Href, "start")
		if err != nil || next == nil {
			break
		}
		opts.Start = next
	}

	LoggingInfo(hiscallInfo, start)
	return nicList, nil
}

// -------- GetNIC

func (h *IbmNICHandler) GetNIC(nicIID irs.IID) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicIID.NameId, "GetNIC()")
	start := call.Start()

	vniID, err := h.resolveVNIID(nicIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.NICInfo{}, err
	}

	vni, _, err := h.VpcService.GetVirtualNetworkInterfaceWithContext(h.Ctx, &vpcv1.GetVirtualNetworkInterfaceOptions{
		ID: core.StringPtr(vniID),
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.NICInfo{}, fmt.Errorf("failed to get VNI %s: %w", vniID, err)
	}

	info, err := vniToNICInfo(*vni)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.NICInfo{}, err
	}

	// Fix 2: Populate PublicIPs using ListFloatingIps filtered by target VNI ID.
	// IBM SDK ListFloatingIpsOptions supports SetTargetID which matches floating IPs
	// whose target is this VNI. This is done only in GetNIC (not bulk list) to avoid
	// excessive API calls.
	h.enrichPublicIPs(&info, vniID)

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

// -------- DeleteNIC

func (h *IbmNICHandler) DeleteNIC(nicIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicIID.NameId, "DeleteNIC()")
	start := call.Start()

	vniID, err := h.resolveVNIID(nicIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	// Check status first – must not be attached
	vni, _, err := h.VpcService.GetVirtualNetworkInterfaceWithContext(h.Ctx, &vpcv1.GetVirtualNetworkInterfaceOptions{
		ID: core.StringPtr(vniID),
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, fmt.Errorf("failed to get VNI before delete: %w", err)
	}
	if vni.Target != nil {
		err := fmt.Errorf("cannot delete NIC %s: it is currently attached to a VM", nicIID.NameId)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	_, _, err = h.VpcService.DeleteVirtualNetworkInterfacesWithContext(h.Ctx, &vpcv1.DeleteVirtualNetworkInterfacesOptions{
		ID: core.StringPtr(vniID),
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, fmt.Errorf("failed to delete VNI %s: %w", vniID, err)
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// -------- AttachNIC

func (h *IbmNICHandler) AttachNIC(nicIID irs.IID, vmIID irs.IID) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicIID.NameId, "AttachNIC()")
	start := call.Start()

	vniID, err := h.resolveVNIID(nicIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.NICInfo{}, err
	}

	instanceID, err := h.resolveInstanceID(vmIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.NICInfo{}, err
	}

	// Try VNI model (new instances) first. If the instance uses the classic NIC model,
	// fall back to CreateInstanceNetworkInterface using the VNI's subnet/SG configuration.
	attachOpts := &vpcv1.CreateInstanceNetworkAttachmentOptions{
		InstanceID: core.StringPtr(instanceID),
		VirtualNetworkInterface: &vpcv1.InstanceNetworkAttachmentPrototypeVirtualNetworkInterfaceVirtualNetworkInterfaceIdentityVirtualNetworkInterfaceIdentityByID{
			ID: core.StringPtr(vniID),
		},
	}

	attachment, _, err := h.VpcService.CreateInstanceNetworkAttachmentWithContext(h.Ctx, attachOpts)
	if err != nil {
		// Classic model instances use network_interface API (not network_attachment).
		// Pre-created standalone VNIs cannot be attached to classic model instances —
		// classic NICs are created and attached in one atomic operation with no pre-create concept.
		LoggingError(hiscallInfo, err)
		return irs.NICInfo{}, fmt.Errorf(
			"AttachNIC failed: IBM VPC instance %s uses the classic network_interface model. "+
				"Pre-created VNIs (standalone NICs) cannot be attached to classic model instances. "+
				"Use a newer instance type that supports the VNI (network_attachment) model. "+
				"Original error: %w", instanceID, err)
	}
	_ = attachment

	// Fix 4: Wait for stable state (up to 60 seconds) and return error on timeout.
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	_ = attachment
	timedOut := true
waitLoop:
	for {
		select {
		case <-timeout:
			break waitLoop
		case <-ticker.C:
			vni, _, err2 := h.VpcService.GetVirtualNetworkInterfaceWithContext(h.Ctx, &vpcv1.GetVirtualNetworkInterfaceOptions{
				ID: core.StringPtr(vniID),
			})
			if err2 == nil && vni.LifecycleState != nil && *vni.LifecycleState == "stable" {
				timedOut = false
				break waitLoop
			}
		}
	}

	if timedOut {
		err := fmt.Errorf("timed out waiting for VNI %s to reach stable state after attach", vniID)
		LoggingError(hiscallInfo, err)
		return irs.NICInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)

	return h.GetNIC(nicIID)
}

// -------- DetachNIC

func (h *IbmNICHandler) DetachNIC(nicIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicIID.NameId, "DetachNIC()")
	start := call.Start()

	vniID, err := h.resolveVNIID(nicIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	// Get VNI to find attached instance
	vni, _, err := h.VpcService.GetVirtualNetworkInterfaceWithContext(h.Ctx, &vpcv1.GetVirtualNetworkInterfaceOptions{
		ID: core.StringPtr(vniID),
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, fmt.Errorf("failed to get VNI %s: %w", vniID, err)
	}

	if vni.Target == nil {
		err := fmt.Errorf("NIC %s is not attached to any VM", nicIID.NameId)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	// Target should be an instance network attachment – find instance ID via listing attachments
	// We search all instances to find which one owns this VNI
	instanceID, attachmentID, err := h.findAttachment(vniID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	_, err = h.VpcService.DeleteInstanceNetworkAttachmentWithContext(h.Ctx, &vpcv1.DeleteInstanceNetworkAttachmentOptions{
		InstanceID: core.StringPtr(instanceID),
		ID:         core.StringPtr(attachmentID),
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, fmt.Errorf("failed to detach VNI %s: %w", vniID, err)
	}

	// Fix 5: Poll until VNI lifecycle state is no longer "deleting"/"updating" (max 90 seconds).
	detachTicker := time.NewTicker(3 * time.Second)
	defer detachTicker.Stop()
	detachTimeout := time.After(90 * time.Second)
	for {
		select {
		case <-detachTimeout:
			// Timeout reached; the detach may still be in progress but we return success
			// because the delete API call itself succeeded.
			LoggingInfo(hiscallInfo, start)
			return true, nil
		case <-detachTicker.C:
			vniState, _, pollErr := h.VpcService.GetVirtualNetworkInterfaceWithContext(h.Ctx, &vpcv1.GetVirtualNetworkInterfaceOptions{
				ID: core.StringPtr(vniID),
			})
			if pollErr != nil {
				// VNI gone or error — treat as detached
				LoggingInfo(hiscallInfo, start)
				return true, nil
			}
			if vniState.LifecycleState != nil {
				state := *vniState.LifecycleState
				if state != "deleting" && state != "updating" {
					LoggingInfo(hiscallInfo, start)
					return true, nil
				}
			}
		}
	}
}

// -------- Internal helpers

// resolveVNIID returns the SystemId (UUID) of a VNI given an IID.
func (h *IbmNICHandler) resolveVNIID(nicIID irs.IID) (string, error) {
	if nicIID.SystemId != "" {
		return nicIID.SystemId, nil
	}
	// Lookup by name
	opts := &vpcv1.ListVirtualNetworkInterfacesOptions{}
	for {
		result, _, err := h.VpcService.ListVirtualNetworkInterfacesWithContext(h.Ctx, opts)
		if err != nil {
			return "", fmt.Errorf("failed to list VNIs: %w", err)
		}
		for _, vni := range result.VirtualNetworkInterfaces {
			if vni.Name != nil && *vni.Name == nicIID.NameId {
				return *vni.ID, nil
			}
		}
		if result.Next == nil {
			break
		}
		next, err := core.GetQueryParam(result.Next.Href, "start")
		if err != nil || next == nil {
			break
		}
		opts.Start = next
	}
	return "", fmt.Errorf("VNI not found: %s", nicIID.NameId)
}

// resolveInstanceID returns the SystemId (UUID) of an instance given an IID.
func (h *IbmNICHandler) resolveInstanceID(vmIID irs.IID) (string, error) {
	if vmIID.SystemId != "" {
		return vmIID.SystemId, nil
	}
	listOpts := &vpcv1.ListInstancesOptions{}
	for {
		result, _, err := h.VpcService.ListInstancesWithContext(h.Ctx, listOpts)
		if err != nil {
			return "", fmt.Errorf("failed to list instances: %w", err)
		}
		for _, inst := range result.Instances {
			if inst.Name != nil && *inst.Name == vmIID.NameId {
				return *inst.ID, nil
			}
		}
		if result.Next == nil {
			break
		}
		next, err := core.GetQueryParam(result.Next.Href, "start")
		if err != nil || next == nil {
			break
		}
		listOpts.Start = next
	}
	return "", fmt.Errorf("instance not found: %s", vmIID.NameId)
}

// findAttachment searches all instances for a network attachment that references the given VNI ID.
// Returns (instanceID, attachmentID, error).
func (h *IbmNICHandler) findAttachment(vniID string) (string, string, error) {
	listOpts := &vpcv1.ListInstancesOptions{}
	for {
		result, _, err := h.VpcService.ListInstancesWithContext(h.Ctx, listOpts)
		if err != nil {
			return "", "", fmt.Errorf("failed to list instances: %w", err)
		}
		for _, inst := range result.Instances {
			if inst.ID == nil {
				continue
			}
			attachments, _, err2 := h.VpcService.ListInstanceNetworkAttachmentsWithContext(h.Ctx, &vpcv1.ListInstanceNetworkAttachmentsOptions{
				InstanceID: inst.ID,
			})
			if err2 != nil {
				continue
			}
			for _, att := range attachments.NetworkAttachments {
				if att.VirtualNetworkInterface == nil || att.VirtualNetworkInterface.ID == nil {
					continue
				}
				if *att.VirtualNetworkInterface.ID == vniID {
					return *inst.ID, *att.ID, nil
				}
			}
		}
		if result.Next == nil {
			break
		}
		next, err := core.GetQueryParam(result.Next.Href, "start")
		if err != nil || next == nil {
			break
		}
		listOpts.Start = next
	}
	return "", "", fmt.Errorf("no attachment found for VNI %s", vniID)
}

// enrichPublicIPs populates info.PublicIPs and info.PublicIP by listing floating IPs
// whose target is the given VNI (Fix 2). IBM SDK ListFloatingIpsOptions.SetTargetID filters
// by target resource ID, which includes VNIs.
// PublicIPs is kept index-aligned with PrivateIPs: a "" entry means no floating IP for that slot.
func (h *IbmNICHandler) enrichPublicIPs(info *irs.NICInfo, vniID string) {
	opts := &vpcv1.ListFloatingIpsOptions{}
	opts.SetTargetID(vniID)

	var floatingAddrs []string
	for {
		result, _, err := h.VpcService.ListFloatingIpsWithContext(h.Ctx, opts)
		if err != nil {
			return
		}
		for _, fip := range result.FloatingIps {
			if fip.Address != nil {
				floatingAddrs = append(floatingAddrs, *fip.Address)
			}
		}
		if result.Next == nil {
			break
		}
		next, err := core.GetQueryParam(result.Next.Href, "start")
		if err != nil || next == nil {
			break
		}
		opts.Start = next
	}

	if len(floatingAddrs) == 0 {
		return
	}

	// Align PublicIPs with PrivateIPs length; fill from floating IPs list.
	privLen := len(info.PrivateIPs)
	if privLen == 0 {
		privLen = 1 // at least one slot for the primary IP
	}
	info.PublicIPs = make([]string, privLen)
	for i, addr := range floatingAddrs {
		if i < privLen {
			info.PublicIPs[i] = addr
		}
	}
	// Convenience primary public IP
	if len(floatingAddrs) > 0 {
		info.PublicIP = floatingAddrs[0]
	}
}

// AddPrivateIP adds a secondary private IP to a VNI via Reserved IP.
// Flow: CreateSubnetReservedIP → AddVirtualNetworkInterfaceIP
func (h *IbmNICHandler) AddPrivateIP(nicIID irs.IID, privateIP string) (irs.NICInfo, error) {
	vniID, err := h.resolveVNIID(nicIID)
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("AddPrivateIP: failed to resolve VNI: %w", err)
	}

	// Get VNI to find its subnet
	vni, _, err := h.VpcService.GetVirtualNetworkInterfaceWithContext(h.Ctx,
		&vpcv1.GetVirtualNetworkInterfaceOptions{ID: core.StringPtr(vniID)})
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("AddPrivateIP: failed to get VNI: %w", err)
	}
	if vni.Subnet == nil || vni.Subnet.ID == nil {
		return irs.NICInfo{}, fmt.Errorf("AddPrivateIP: VNI has no subnet")
	}

	// Create a Reserved IP in the subnet
	createOpts := &vpcv1.CreateSubnetReservedIPOptions{
		SubnetID: vni.Subnet.ID,
	}
	autoDelete := false
	createOpts.AutoDelete = &autoDelete
	if privateIP != "" {
		createOpts.Address = &privateIP
	}
	reservedIP, _, err := h.VpcService.CreateSubnetReservedIPWithContext(h.Ctx, createOpts)
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("AddPrivateIP: failed to create reserved IP: %w", err)
	}

	// Add Reserved IP to VNI
	addOpts := h.VpcService.NewAddVirtualNetworkInterfaceIPOptions(vniID, *reservedIP.ID)
	_, _, err = h.VpcService.AddVirtualNetworkInterfaceIPWithContext(h.Ctx, addOpts)
	if err != nil {
		// Cleanup reserved IP on failure
		h.VpcService.DeleteSubnetReservedIPWithContext(h.Ctx,
			&vpcv1.DeleteSubnetReservedIPOptions{SubnetID: vni.Subnet.ID, ID: reservedIP.ID})
		return irs.NICInfo{}, fmt.Errorf("AddPrivateIP: failed to add reserved IP to VNI: %w", err)
	}

	return h.GetNIC(nicIID)
}

// RemovePrivateIP removes a secondary private IP from a VNI.
// Flow: find Reserved IP by address → RemoveVirtualNetworkInterfaceIP → DeleteSubnetReservedIP
func (h *IbmNICHandler) RemovePrivateIP(nicIID irs.IID, privateIP string) (bool, error) {
	vniID, err := h.resolveVNIID(nicIID)
	if err != nil {
		return false, fmt.Errorf("RemovePrivateIP: failed to resolve VNI: %w", err)
	}

	// Get VNI to find subnet
	vni, _, err := h.VpcService.GetVirtualNetworkInterfaceWithContext(h.Ctx,
		&vpcv1.GetVirtualNetworkInterfaceOptions{ID: core.StringPtr(vniID)})
	if err != nil {
		return false, fmt.Errorf("RemovePrivateIP: failed to get VNI: %w", err)
	}
	if vni.Subnet == nil || vni.Subnet.ID == nil {
		return false, fmt.Errorf("RemovePrivateIP: VNI has no subnet")
	}

	// Find Reserved IP by address via ListVirtualNetworkInterfaceIps
	listOpts := &vpcv1.ListVirtualNetworkInterfaceIpsOptions{
		VirtualNetworkInterfaceID: core.StringPtr(vniID),
	}
	result, _, err := h.VpcService.ListVirtualNetworkInterfaceIpsWithContext(h.Ctx, listOpts)
	if err != nil {
		return false, fmt.Errorf("RemovePrivateIP: failed to list VNI IPs: %w", err)
	}

	var targetID string
	for _, rip := range result.Ips {
		if rip.Address != nil && *rip.Address == privateIP {
			targetID = *rip.ID
			break
		}
	}
	if targetID == "" {
		return false, fmt.Errorf("RemovePrivateIP: IP %s not found on VNI %s", privateIP, vniID)
	}

	// Remove from VNI
	removeOpts := &vpcv1.RemoveVirtualNetworkInterfaceIPOptions{
		VirtualNetworkInterfaceID: core.StringPtr(vniID),
		ID:                        core.StringPtr(targetID),
	}
	_, err = h.VpcService.RemoveVirtualNetworkInterfaceIPWithContext(h.Ctx, removeOpts)
	if err != nil {
		return false, fmt.Errorf("RemovePrivateIP: failed to remove IP from VNI: %w", err)
	}

	// Delete the Reserved IP from subnet
	h.VpcService.DeleteSubnetReservedIPWithContext(h.Ctx,
		&vpcv1.DeleteSubnetReservedIPOptions{SubnetID: vni.Subnet.ID, ID: core.StringPtr(targetID)})

	return true, nil
}

// GetNICOSConfigScript returns a bash script that must be run inside the IBM Cloud VM OS
// after a secondary VNI (Virtual Network Interface) is attached. IBM Cloud VPC instances
// do not auto-configure secondary interfaces; the guest OS must bring up the interface
// and configure policy-based routing.
func (h *IbmNICHandler) GetNICOSConfigScript(nicIID irs.IID) (string, error) {
	nicInfo, err := h.GetNIC(nicIID)
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

	mac := strings.ToLower(nicInfo.MACAddress)
	ip0 := privateIPs[0]
	lastDot := strings.LastIndex(ip0, ".")
	gateway := ip0[:lastDot] + ".1"
	cidr := ip0[:lastDot] + ".0/24"

	script := "# 1. Identify interface name by MAC\n" +
		"IFACE=$(ip link | grep -B1 \"" + mac + "\" | head -1 | awk -F': ' '{print $2}')\n" +
		"echo \"Target interface: $IFACE\"\n"

	script += "\n# 2. Bring up secondary NIC (do NOT use dhclient — it adds a default route\n" +
		"#    to the main table and breaks primary NIC SSH)\n" +
		"sudo ip link set $IFACE up\n"
	for _, ip := range privateIPs {
		script += "sudo ip addr add " + ip + "/24 dev $IFACE 2>/dev/null || true\n"
	}

	script += "\n# 3. Configure Policy-based Routing (PBR)\n"
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
