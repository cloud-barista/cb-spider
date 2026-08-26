// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud VPC Public IP (Floating IP) Handler
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"fmt"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"
	ips "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/extensions/floatingips"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/ports"
	portforward "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/extensions/layer3/portforwarding"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/extensions/layer3/staticnat"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/pagination"
)

type KTVpcPublicIPHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	NetworkClient  *ktvpcsdk.ServiceClient
	VMClient       *ktvpcsdk.ServiceClient
	ImageClient    *ktvpcsdk.ServiceClient
	VolumeClient   *ktvpcsdk.ServiceClient
}

func (h *KTVpcPublicIPHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, "ListIID", "floatingips.List()")
	start := call.Start()

	listOpts := ips.ListOpts{
		Page: 1, 
		Size: 2000, // Max page size, to list all data in a single page
	}	
	pager := ips.List(h.NetworkClient, listOpts)

	var iidList []*irs.IID
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		fipList, err := ips.ExtractFloatingIPs(page)
		if err != nil {
			return false, err
		}
		for _, fip := range fipList {
			iidList = append(iidList, &irs.IID{NameId: fip.PublicIpID, SystemId: fip.PublicIpID})
		}
		return true, nil
	})
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		loggingError(hiscallInfo, err)
		return nil, err
	}
	loggingInfo(hiscallInfo, start)

	return iidList, nil
}

func (h *KTVpcPublicIPHandler) CreatePublicIP(reqInfo irs.PublicIPInfo) (irs.PublicIPInfo, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, reqInfo.IId.NameId, "floatingips.Create()")
	start := call.Start()

	createOpts := ips.CreateOpts{}
	result, err := ips.Create(h.NetworkClient, createOpts).ExtractCreate()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		loggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	loggingInfo(hiscallInfo, start)

	// KT create only returns PublicIpID; get full info via list
	info, getErr := h.GetPublicIP(irs.IID{NameId: reqInfo.IId.NameId, SystemId: result.Data.PublicIpID})
	if getErr != nil {
		// Return minimal info if get fails
		return irs.PublicIPInfo{
			IId:             irs.IID{NameId: reqInfo.IId.NameId, SystemId: result.Data.PublicIpID},
			PublicIPAddress: "NA",
			Status:          irs.PublicIPAvailable,
			CreatedTime:     time.Time{},
		}, nil
	}
	info.IId.NameId = reqInfo.IId.NameId
	return info, nil
}

func (h *KTVpcPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, "All", "floatingips.List()")
	start := call.Start()

	listOpts := ips.ListOpts{
		Page: 1,
		Size: 2000, // Max page size, to list all data in a single page
	}
	pager := ips.List(h.NetworkClient, listOpts)

	var infoList []*irs.PublicIPInfo
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		fipList, err := ips.ExtractFloatingIPs(page)
		if err != nil {
			return false, err
		}
		for _, fip := range fipList {
			info := extractKTPublicIPInfo(&fip)
			infoList = append(infoList, &info)
		}
		return true, nil
	})
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		loggingError(hiscallInfo, err)
		return nil, err
	}
	loggingInfo(hiscallInfo, start)

	if infoList == nil {
		infoList = []*irs.PublicIPInfo{}
	}
	return infoList, nil
}

func (h *KTVpcPublicIPHandler) GetPublicIP(publicIPIID irs.IID) (irs.PublicIPInfo, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, publicIPIID.NameId, "floatingips.Get()")
	start := call.Start()

	var foundFip *ips.FloatingIP
	var err error

	if publicIPIID.SystemId != "" {
		result := ips.Get(h.NetworkClient, publicIPIID.SystemId)
		if result.Err != nil {
			err = result.Err
		} else {
			fip, extractErr := result.ExtractFloatingIP()
			if extractErr != nil {
				err = extractErr
			} else {
				foundFip = fip
			}
		}
	} else {
		// Search by listing
		listOpts := ips.ListOpts{
			Page: 1,
			Size: 2000, // Max page size, to list all data in a single page
		}
		pager := ips.List(h.NetworkClient, listOpts)
		pager.EachPage(func(page pagination.Page) (bool, error) {
			fipList, listErr := ips.ExtractFloatingIPs(page)
			if listErr != nil {
				return false, listErr
			}
			for i, fip := range fipList {
				if fip.PublicIpID == publicIPIID.NameId || fip.PublicIP == publicIPIID.NameId {
					foundFip = &fipList[i]
					return false, nil
				}
			}
			return true, nil
		})
		if foundFip == nil {
			err = fmt.Errorf("PublicIP not found: %s", publicIPIID.NameId)
		}
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		loggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	loggingInfo(hiscallInfo, start)

	info := extractKTPublicIPInfo(foundFip)
	if publicIPIID.NameId != "" {
		info.IId.NameId = publicIPIID.NameId
	}
	return info, nil
}

func (h *KTVpcPublicIPHandler) DeletePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, publicIPIID.NameId, "floatingips.Delete()")
	start := call.Start()

	systemId := publicIPIID.SystemId
	if systemId == "" {
		info, err := h.GetPublicIP(publicIPIID)
		if err != nil {
			return false, err
		}
		systemId = info.IId.SystemId
	}

	result := ips.Delete(h.NetworkClient, systemId)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if result.Err != nil {
		cblogger.Error(result.Err)
		loggingError(hiscallInfo, result.Err)
		return false, result.Err
	}
	loggingInfo(hiscallInfo, start)

	return true, nil
}

func extractKTPublicIPInfo(fip *ips.FloatingIP) irs.PublicIPInfo {
	status := irs.PublicIPAvailable
	ownedPrivateIP := ""
	if len(fip.StaticNats) > 0 {
		status = irs.PublicIPAssociated
	}
	// PortForwardings is embedded directly in the floating-IP response (no
	// extra API call needed) and, unlike StaticNats, carries the mapped
	// private IP - this is the mechanism VM creation uses for
	// AssignPublicIP=true (see StartVM), so most real associations resolve
	// here rather than through the StaticNAT list.
	if len(fip.PortForwardings) > 0 {
		status = irs.PublicIPAssociated
		ownedPrivateIP = fip.PortForwardings[0].MappedIP
	}

	info := irs.PublicIPInfo{
		IId:             irs.IID{NameId: fip.PublicIpID, SystemId: fip.PublicIpID},
		PublicIPAddress: fip.PublicIP,
		Status:          status,
		OwnedPrivateIP:  ownedPrivateIP,
		CreatedTime:     time.Time{},
	}

	info.KeyValueList = []irs.KeyValue{
		{Key: "PublicIpID", Value: fip.PublicIpID},
		{Key: "VpcID", Value: fip.VpcID},
		{Key: "ZoneID", Value: fip.ZoneID},
		{Key: "Type", Value: fip.Type},
	}

	return info
}

// resolveKTPublicIPOwner fills OwnedPrivateIP from the StaticNAT list, for
// the legacy case where extractKTPublicIPInfo's embedded PortForwardings
// data didn't already resolve it (StaticNatInfo carries no mapped-IP field
// of its own, so that case needs this separate lookup).
func (h *KTVpcPublicIPHandler) resolveKTPublicIPOwner(info *irs.PublicIPInfo) {
	if info.Status != irs.PublicIPAssociated || info.OwnedPrivateIP != "" {
		return
	}
	allPages, err := staticnat.List(h.NetworkClient, staticnat.ListOpts{}).AllPages()
	if err != nil {
		return
	}
	nats, err := staticnat.ExtractStaticNats(allPages)
	if err != nil {
		return
	}
	for _, nat := range nats {
		if nat.PublicIpID == info.IId.SystemId {
			info.OwnedPrivateIP = nat.MappedIP
			break
		}
	}
}

// AssociatePublicIP associates a Public IP to a VM's private IP via KT Cloud
// PortForwarding + Firewall rules - the exact mechanism VM creation uses for
// AssignPublicIP=true (see StartVM/createPortForwardingFirewallRules), so a
// PublicIP attached here behaves identically to one assigned at VM-creation
// time. KT Cloud VPC also offers a separate StaticNAT primitive (a 1:1 IP
// mapping outside of SecurityGroup-derived rules), but VM creation never
// uses it, and creating one here would leave this PublicIP in a state
// inconsistent with every VM-creation-time PublicIP - so this deliberately
// does not call staticnat.Create. (DisassociatePublicIP/RemoveDefaultPublicIP
// still clean up a StaticNAT if one happens to exist, for PublicIPs
// associated before this change.)
func (h *KTVpcPublicIPHandler) AssociatePublicIP(publicIPIID irs.IID, vmIID irs.IID, nicIID irs.IID, privateIP string) (irs.PublicIPInfo, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, publicIPIID.NameId, "createPortForwardingFirewallRules()")
	start := call.Start()

	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	if privateIP == "" {
		// Auto-resolve private IP from VM if vmIID is provided
		if vmIID.SystemId != "" || vmIID.NameId != "" {
			resolved, resolveErr := h.resolveVMPrivateIP(vmIID)
			if resolveErr != nil {
				return irs.PublicIPInfo{}, fmt.Errorf("AssociatePublicIP: privateIP not provided and failed to resolve from VM [%s]: %w", vmIID.NameId, resolveErr)
			}
			privateIP = resolved
		} else {
			return irs.PublicIPInfo{}, fmt.Errorf("AssociatePublicIP: privateIP is required (or provide vmIID to auto-resolve)")
		}
	}

	if fwErr := h.createSecurityRules(vmIID, nicIID, privateIP, info.PublicIPAddress, info.IId.SystemId); fwErr != nil {
		newErr := fmt.Errorf("AssociatePublicIP: failed to set up PortForwarding/Firewall rules: %w", fwErr)
		cblogger.Error(newErr)
		return irs.PublicIPInfo{}, newErr
	}
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	loggingInfo(hiscallInfo, start)

	result, getErr := h.GetPublicIP(irs.IID{NameId: publicIPIID.NameId, SystemId: info.IId.SystemId})
	if getErr != nil {
		return irs.PublicIPInfo{}, getErr
	}
	h.resolveKTPublicIPOwner(&result)
	return result, nil
}

// resolveSecurityContext resolves the TierNetworkId and SecurityGroup
// SystemIDs needed to (re)create PortForwarding/Firewall rules, via
// GetVM() + getNetworkIdWithTierId() - the same calls StartVM uses to build
// these values from the original create request. KT Cloud VPC does not
// really run on Neutron - the openstack-shaped `ports` API
// (ports.Get/ports.List) is a thin compatibility shim that has been
// observed to return persistent Internal Server Errors, so it is not used
// here at all.
//
// GetVM() itself pulls in some unrelated lookups (image/disk info) and one
// of those has been observed to hang indefinitely on a slow/misbehaving KT
// API response. To avoid blocking this request forever, the call is bounded
// with a timeout - the goroutine is abandoned (not cancelled) if it fires,
// trading a leaked goroutine on the rare hang for a request that still
// returns an error instead of hanging.
func (h *KTVpcPublicIPHandler) resolveSecurityContext(vmIID irs.IID, nicIID irs.IID) (tierNetworkId string, sgSystemIDs []string, err error) {
	if vmIID.SystemId == "" && vmIID.NameId == "" {
		return "", nil, fmt.Errorf("resolveSecurityContext: vmIID is required to resolve Tier/SecurityGroup info via GetVM()")
	}

	// GetVM() (via mappingVMInfo) uses ImageClient/VolumeClient too (e.g. for
	// disk/image lookups) - a handler built with only these three fields
	// leaves those nil and crashes the whole process with a SIGSEGV the
	// moment it touches one, since that happens inside a goroutine with no
	// recover(). Always construct this from every client this handler has.
	vmHandler := &KTVpcVMHandler{
		CredentialInfo: h.CredentialInfo,
		RegionInfo:     h.RegionInfo,
		NetworkClient:  h.NetworkClient,
		VMClient:       h.VMClient,
		ImageClient:    h.ImageClient,
		VolumeClient:   h.VolumeClient,
	}

	type getVMResult struct {
		vmInfo irs.VMInfo
		err    error
	}
	resultCh := make(chan getVMResult, 1)
	go func() {
		// This goroutine has no caller to propagate a panic to, so an
		// unrecovered one (e.g. a future nil-pointer bug inside GetVM) would
		// crash the entire CB-Spider process, not just this request. Recover
		// and report it as an error instead.
		defer func() {
			if r := recover(); r != nil {
				resultCh <- getVMResult{irs.VMInfo{}, fmt.Errorf("panic in GetVM: %v", r)}
			}
		}()
		vmInfo, getErr := vmHandler.GetVM(vmIID)
		resultCh <- getVMResult{vmInfo, getErr}
	}()

	var vmInfo irs.VMInfo
	select {
	case r := <-resultCh:
		if r.err != nil {
			return "", nil, fmt.Errorf("resolveSecurityContext: GetVM failed for VM [%s]: %w", vmIID.NameId, r.err)
		}
		vmInfo = r.vmInfo
	case <-time.After(90 * time.Second):
		return "", nil, fmt.Errorf("resolveSecurityContext: GetVM timed out after 90s for VM [%s]", vmIID.NameId)
	}

	if vmInfo.SubnetIID.SystemId == "" {
		return "", nil, fmt.Errorf("resolveSecurityContext: VM [%s] has no resolved Subnet(Tier) ID", vmIID.NameId)
	}

	vpcHandler := KTVpcVPCHandler{RegionInfo: h.RegionInfo, NetworkClient: h.NetworkClient}
	tierNetId, tierErr := vpcHandler.getNetworkIdWithTierId(vmInfo.SubnetIID.SystemId)
	if tierErr != nil {
		return "", nil, fmt.Errorf("resolveSecurityContext: getNetworkIdWithTierId failed for VM [%s]: %w", vmIID.NameId, tierErr)
	}

	for _, sg := range vmInfo.SecurityGroupIIds {
		sgSystemIDs = append(sgSystemIDs, sg.SystemId)
	}
	return *tierNetId, sgSystemIDs, nil
}

// createSecurityRules builds and creates the PortForwarding/Firewall rules
// that enforce the VM's SecurityGroups against the given PublicIP - the
// only mechanism KT Cloud VPC provides for SecurityGroup enforcement.
func (h *KTVpcPublicIPHandler) createSecurityRules(vmIID irs.IID, nicIID irs.IID, privateIP string, publicIP string, publicIPId string) error {
	tierNetworkId, sgSystemIDs, err := h.resolveSecurityContext(vmIID, nicIID)
	if err != nil {
		return err
	}

	ruleSet := &SecurityRuleSet{
		TierNetworkId:          tierNetworkId,
		SecurityGroupSystemIDs: sgSystemIDs,
		PrivateIP:              privateIP,
		PublicIP:               publicIP,
		PublicIPId:             publicIPId,
	}

	vmHandler := &KTVpcVMHandler{RegionInfo: h.RegionInfo, NetworkClient: h.NetworkClient, VMClient: h.VMClient}
	if ok, err := vmHandler.createPortForwardingFirewallRules(ruleSet); !ok {
		return err
	}
	return nil
}

// DisassociatePublicIP removes the PortForwarding/Firewall binding for a
// PublicIP - the mechanism AssociatePublicIP now uses (see its comment),
// matching VM-creation-time behavior.
func (h *KTVpcPublicIPHandler) DisassociatePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, publicIPIID.NameId, "removePortForwardingRules()")
	start := call.Start()

	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return false, err
	}
	if info.Status != irs.PublicIPAssociated {
		return false, fmt.Errorf("DisassociatePublicIP: PublicIP %s is not associated", publicIPIID.NameId)
	}

	// A firewall-deletion failure is non-fatal (warn and continue, mirrors
	// TerminateVM's cleanup order/risk tolerance), but a port-forwarding
	// failure blocks the disassociation.
	vmHandler := &KTVpcVMHandler{RegionInfo: h.RegionInfo, NetworkClient: h.NetworkClient, VMClient: h.VMClient}
	if _, fwErr := vmHandler.removeFirewallRules(info.PublicIPAddress); fwErr != nil {
		cblogger.Warnf("DisassociatePublicIP: failed to remove firewall rules for PublicIP %s (continuing): %v", info.PublicIPAddress, fwErr)
	}
	if _, pfErr := vmHandler.removePortForwardingRules(info.PublicIPAddress); pfErr != nil {
		newErr := fmt.Errorf("DisassociatePublicIP: failed to remove port-forwarding rules for PublicIP %s: %w", info.PublicIPAddress, pfErr)
		cblogger.Error(newErr)
		return false, newErr
	}

	// Best-effort: also clear a StaticNAT binding if one happens to exist
	// (e.g. a PublicIP associated before this change, or one bound manually
	// via the KT console/older code). Not required for success - KT Cloud's
	// StaticNAT list endpoint has been observed to be flaky for extended
	// periods, and that shouldn't fail a disassociation that already
	// succeeded via PortForwarding.
	h.removeStaticNATIfAny(info.IId.SystemId)

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	loggingInfo(hiscallInfo, start)
	return true, nil
}

// removeStaticNATIfAny is a best-effort cleanup for the legacy StaticNAT
// binding mechanism (see AssociatePublicIP's comment) - failures are logged,
// not returned, since callers should not fail an otherwise-successful
// disassociation/removal over this.
func (h *KTVpcPublicIPHandler) removeStaticNATIfAny(publicIPSystemId string) {
	allPages, err := staticnat.List(h.NetworkClient, staticnat.ListOpts{}).AllPages()
	if err != nil {
		cblogger.Warnf("removeStaticNATIfAny: failed to list StaticNATs for PublicIP %s (skipping): %v", publicIPSystemId, err)
		return
	}
	nats, err := staticnat.ExtractStaticNats(allPages)
	if err != nil {
		cblogger.Warnf("removeStaticNATIfAny: failed to extract StaticNATs for PublicIP %s (skipping): %v", publicIPSystemId, err)
		return
	}
	for _, nat := range nats {
		if nat.PublicIpID == publicIPSystemId {
			if delErr := staticnat.Delete(h.NetworkClient, nat.StaticNatID).ExtractErr(); delErr != nil {
				cblogger.Warnf("removeStaticNATIfAny: failed to delete StaticNAT %s for PublicIP %s: %v", nat.StaticNatID, publicIPSystemId, delErr)
			}
			return
		}
	}
}

// resolveVMPrivateIP finds the first private IP of a VM via its ports.
func (h *KTVpcPublicIPHandler) resolveVMPrivateIP(vmIID irs.IID) (string, error) {
	deviceID := vmIID.SystemId
	if deviceID == "" {
		deviceID = vmIID.NameId
	}
	allPages, err := ports.List(h.NetworkClient, ports.ListOpts{DeviceID: deviceID}).AllPages()
	if err != nil {
		return "", fmt.Errorf("failed to list ports for VM [%s]: %w", deviceID, err)
	}
	portList, err := ports.ExtractPorts(allPages)
	if err != nil {
		return "", fmt.Errorf("failed to extract ports for VM [%s]: %w", deviceID, err)
	}
	for _, p := range portList {
		if len(p.FixedIPs) > 0 && p.FixedIPs[0].IPAddress != "" {
			return p.FixedIPs[0].IPAddress, nil
		}
	}
	return "", fmt.Errorf("no private IP found for VM [%s]", deviceID)
}

// RemoveDefaultPublicIP removes whatever PublicIP is currently bound to the
// VM's private IP, discovered live (regardless of whether it was ever
// tracked as a separate CB-Spider PublicIP resource). It looks up the
// binding via PortForwarding rules first - the mechanism VM creation
// actually uses for AssignPublicIP=true (see StartVM) - and falls back to
// StaticNAT (the older mechanism AssociatePublicIP used before it was
// aligned to match VM creation; see its comment) if nothing is found there.
// KT Cloud VPC enforces Security Groups ONLY via per-PublicIP
// PortForwarding/Firewall rules, so removing the PublicIP also destroys the
// VM's network-level Security Group enforcement - this is unavoidable on
// this CSP and is logged loudly rather than silently. Works on a running VM.
func (h *KTVpcPublicIPHandler) RemoveDefaultPublicIP(vmIID irs.IID) (bool, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, vmIID.NameId, "RemoveDefaultPublicIP()")
	start := call.Start()

	privateIP, err := h.resolveVMPrivateIP(vmIID)
	if err != nil {
		cblogger.Error(err)
		loggingError(hiscallInfo, err)
		return false, err
	}

	publicIP, publicIPId, err := h.findBoundPublicIPByPrivateIP(privateIP)
	if err != nil {
		err := fmt.Errorf("RemoveDefaultPublicIP: %w", err)
		cblogger.Error(err)
		loggingError(hiscallInfo, err)
		return false, err
	}

	cblogger.Warnf("RemoveDefaultPublicIP: removing PublicIP %s from VM %s - KT Cloud VPC enforces "+
		"Security Groups only via per-PublicIP Firewall/PortForwarding rules, so this VM's "+
		"SecurityGroups will NO LONGER be enforced at the network level after this operation.",
		publicIP, vmIID.NameId)

	// Best-effort: clean up firewall/port-forwarding rules (and any legacy
	// StaticNAT binding) first, so they aren't left orphaned once the
	// PublicIP itself is gone.
	cleanupVMHandler := &KTVpcVMHandler{RegionInfo: h.RegionInfo, NetworkClient: h.NetworkClient}
	if _, fwErr := cleanupVMHandler.removeFirewallRules(publicIP); fwErr != nil {
		cblogger.Warnf("RemoveDefaultPublicIP: failed to remove firewall rules for PublicIP %s: %v", publicIP, fwErr)
	}
	if _, pfErr := cleanupVMHandler.removePortForwardingRules(publicIP); pfErr != nil {
		cblogger.Warnf("RemoveDefaultPublicIP: failed to remove port-forwarding rules for PublicIP %s: %v", publicIP, pfErr)
	}
	h.removeStaticNATIfAny(publicIPId)

	if _, err := h.DeletePublicIP(irs.IID{SystemId: publicIPId}); err != nil {
		cblogger.Error(err)
		loggingError(hiscallInfo, err)
		return false, err
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	loggingInfo(hiscallInfo, start)

	return true, nil
}

// findBoundPublicIPByPrivateIP finds the PublicIP address+ID currently bound
// to privateIP, checking PortForwarding rules first (the mechanism VM
// creation uses) and falling back to StaticNAT (the older mechanism - see
// AssociatePublicIP's comment) if nothing is found there.
func (h *KTVpcPublicIPHandler) findBoundPublicIPByPrivateIP(privateIP string) (publicIP string, publicIPId string, err error) {
	pfAllPages, pfErr := portforward.List(h.NetworkClient, portforward.ListOpts{Page: 1, Size: 2000}).AllPages()
	if pfErr == nil {
		if pfs, extractErr := portforward.ExtractPFs(pfAllPages); extractErr == nil {
			for _, pf := range pfs {
				if pf.MappedIP == privateIP {
					return pf.PublicIP, pf.PublicIPID, nil
				}
			}
		}
	}

	allPages, snErr := staticnat.List(h.NetworkClient, staticnat.ListOpts{}).AllPages()
	if snErr != nil {
		return "", "", fmt.Errorf("no PublicIP found for private IP %s (no matching PortForwarding rule; StaticNAT lookup also failed: %w)", privateIP, snErr)
	}
	nats, extractErr := staticnat.ExtractStaticNats(allPages)
	if extractErr != nil {
		return "", "", fmt.Errorf("no PublicIP found for private IP %s (no matching PortForwarding rule; StaticNAT extract also failed: %w)", privateIP, extractErr)
	}
	for _, nat := range nats {
		if nat.MappedIP == privateIP {
			return nat.PublicIP, nat.PublicIpID, nil
		}
	}
	return "", "", fmt.Errorf("no PublicIP found bound to private IP %s (checked both PortForwarding rules and StaticNATs)", privateIP)
}
