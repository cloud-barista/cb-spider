// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2026.08.

package commonruntime

import (
	"fmt"
	"os"
	"strings"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	infostore "github.com/cloud-barista/cb-spider/info-store"
)

// defaultPublicIPName derives the deterministic PublicIP name used by
// AssignVMDefaultPublicIP/UnassignVMDefaultPublicIP, so the pair can find
// each other's resource without any extra tracking.
func defaultPublicIPName(vmName string) string {
	return vmName + "-defaultpip"
}

// resolveDefaultNIC returns the VM's default (DeviceIndex==0) NIC's driver-level
// SystemId and its primary PrivateIP. NICs[0] is not trusted blindly (only Tencent
// sorts by DeviceIndex), so it scans for DeviceIndex==0 explicitly, falling back to
// VMInfo.NetworkInterface / VMInfo.PrivateIP when NICs is empty.
func resolveDefaultNIC(vmInfo cres.VMInfo) (nicSystemId string, privateIP string) {
	for _, nic := range vmInfo.NICs {
		if nic.DeviceIndex == 0 {
			privateIP = vmInfo.PrivateIP
			if len(nic.PrivateIPs) > 0 {
				privateIP = nic.PrivateIPs[0]
			}
			return nic.IId.SystemId, privateIP
		}
	}
	return vmInfo.NetworkInterface, vmInfo.PrivateIP
}

// buildAssociateArgs decides which (vmName, nicName, privateIP) shape to pass into
// AssociatePublicIP, matching the CSP-specific flows it already supports:
//   - NCP: Flow C - vmName only (VM-level association).
//   - GCP: Flow B - nicName = "{vmName}/nic0" (the default NIC is always nic0).
//   - KT: Flow A plus vmName - KT's driver resolves SecurityGroup/Tier info
//     via GetVM(vmIID) (KT Cloud VPC does not really run on Neutron, so its
//     `ports` API is not used), which needs vmIID to be populated.
//   - all others: Flow A - nicName = the default NIC's driver SystemId, plus privateIP.
func buildAssociateArgs(connectionName string, vmName string, nicSystemId string, privateIP string) (vmNameArg string, nicNameArg string, privateIPArg string, err error) {
	providerName, err := ccm.GetProviderNameByConnectionName(connectionName)
	if err != nil {
		return "", "", "", err
	}

	switch {
	case strings.EqualFold(providerName, "NCP"):
		return vmName, "", "", nil
	case strings.EqualFold(providerName, "GCP"):
		return "", fmt.Sprintf("%s/nic0", vmName), "", nil
	case strings.EqualFold(providerName, "KT"):
		return vmName, nicSystemId, privateIP, nil
	default:
		return "", nicSystemId, privateIP, nil
	}
}

// AssignVMDefaultPublicIP creates a new PublicIP and attaches it to the VM's
// default NIC (nic0) - the same effect as creating the VM with AssignPublicIP=true.
// Fails (including the existing address) if the VM already has a default PublicIP.
func AssignVMDefaultPublicIP(connectionName string, vmName string) (*cres.PublicIPInfo, error) {
	cblog.Info("call AssignVMDefaultPublicIP()")

	vmInfo, err := GetVM(connectionName, VM, vmName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if vmInfo.PublicIP != "" {
		err := fmt.Errorf("VM '%s' already has a default PublicIP: %s", vmName, vmInfo.PublicIP)
		cblog.Error(err)
		return nil, err
	}

	nicSystemId, privateIP := resolveDefaultNIC(*vmInfo)

	vmNameArg, nicNameArg, privateIPArg, err := buildAssociateArgs(connectionName, vmName, nicSystemId, privateIP)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	publicIPName := defaultPublicIPName(vmName)

	if _, err := CreatePublicIP(connectionName, PUBLICIP, cres.PublicIPInfo{IId: cres.IID{NameId: publicIPName}}, ""); err != nil {
		cblog.Error(err)
		return nil, err
	}

	info, err := AssociatePublicIP(connectionName, publicIPName, vmNameArg, nicNameArg, privateIPArg)
	if err != nil {
		cblog.Error(err)
		// rollback the just-created PublicIP so it isn't left orphaned/unassociated
		if _, delErr := DeletePublicIP(connectionName, PUBLICIP, publicIPName, ""); delErr != nil {
			cblog.Error(delErr)
		}
		return nil, err
	}

	return info, nil
}

// UnassignVMDefaultPublicIP disassociates and DELETES the PublicIP currently
// attached to the VM's default NIC. Fails if the VM has no default PublicIP.
//
// The VM's current PublicIP can be in one of three states, and each is
// handled differently to avoid ever silently deleting a resource the user
// owns:
//  1. Assigned via AssignVMDefaultPublicIP (deterministic name) -> disassociate + delete.
//  2. Attached via AttachVMPublicIP, i.e. a PublicIP the user created via the
//     PublicIP Manager and cb-spider tracks under some OTHER name -> refuse
//     to delete; tell the caller to use DetachVMPublicIP with that name instead.
//  3. Assigned natively by the CSP at VM-creation time (AssignPublicIP=true,
//     the default) and never tracked as a CB-Spider PublicIP resource at all
//     -> remove it directly via the driver's own native mechanism.
func UnassignVMDefaultPublicIP(connectionName string, vmName string) (bool, error) {
	cblog.Info("call UnassignVMDefaultPublicIP()")

	vmInfo, err := GetVM(connectionName, VM, vmName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	if vmInfo.PublicIP == "" {
		err := fmt.Errorf("VM '%s' has no default PublicIP assigned", vmName)
		cblog.Error(err)
		return false, err
	}

	// (1) Assigned via AssignVMDefaultPublicIP - deterministic name.
	publicIPName := defaultPublicIPName(vmName)
	if _, err := DisassociatePublicIP(connectionName, publicIPName); err == nil {
		return deletePublicIPWithRetry(connectionName, publicIPName)
	}

	// (2) Tracked by cb-spider under some other name - it's the user's own
	// resource (attached via AttachVMPublicIP); refuse to delete it here.
	if trackedName := findTrackedPublicIPNameForVM(connectionName, vmName); trackedName != "" {
		err := fmt.Errorf(
			"VM '%s''s PublicIP is a user-managed PublicIP resource '%s'. "+
				"To detach it without deleting it, use DetachVMPublicIP (DELETE /vm/%s/publicip/%s) instead",
			vmName, trackedName, vmName, trackedName)
		cblog.Error(err)
		return false, err
	}

	// (3) Not tracked by cb-spider at all - a CSP-native auto-assigned PublicIP.
	return removeNativeDefaultPublicIP(connectionName, vmName)
}

// deletePublicIPWithRetry deletes a just-disassociated PublicIP, retrying on
// failure for a short while. Several CSPs (observed on Tencent -
// "OperationDenied.MutexTaskRunning" - and Alibaba - "SDK.ServerError") leave
// the address in a transitional state for a few seconds after Disassociate
// returns success, so an immediate Delete can be rejected even though the
// disassociation itself succeeded.
func deletePublicIPWithRetry(connectionName string, publicIPName string) (bool, error) {
	waiter := NewWaiter(3, 15)
	for {
		ok, err := DeletePublicIP(connectionName, PUBLICIP, publicIPName, "")
		if err == nil {
			return ok, nil
		}
		if !waiter.Wait() {
			return false, err
		}
		cblog.Warnf("DeletePublicIP retrying after transient error: %v", err)
	}
}

// findTrackedPublicIPNameForVM scans cb-spider's own PublicIP records (not a
// live CSP query) for one whose OwnedVM matches vmName, and returns its
// Spider name - or "" if none is found.
func findTrackedPublicIPNameForVM(connectionName string, vmName string) string {
	pipList, err := ListPublicIP(connectionName, PUBLICIP)
	if err != nil {
		cblog.Error(err)
		return ""
	}
	for _, pip := range pipList {
		if pip != nil && pip.OwnedVM.NameId == vmName {
			return pip.IId.NameId
		}
	}
	return ""
}

// removeNativeDefaultPublicIP resolves the VM's driver-level IID and calls
// handler.RemoveDefaultPublicIP(vmDriverIID) directly - the same
// connection/IID resolution pattern GetVM/DeleteVM use in VMManager.go - to
// remove a PublicIP that was never registered as a CB-Spider PublicIP
// resource (i.e. it has no Spider name to resolve through the usual
// Associate/Disassociate/Delete path).
func removeNativeDefaultPublicIP(connectionName string, vmName string) (bool, error) {
	var iidInfo VMIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*VMIIDInfo
		if err := getAuthIIDInfoList(connectionName, &iidInfoList); err != nil {
			cblog.Error(err)
			return false, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, vmName)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		iidInfo = *castedIIDInfo.(*VMIIDInfo)
	} else {
		if err := infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, vmName); err != nil {
			cblog.Error(err)
			return false, err
		}
	}

	cldConn, err := ccm.GetZoneLevelCloudConnection(connectionName, iidInfo.ZoneId)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	vmDriverIID := getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
	return handler.RemoveDefaultPublicIP(vmDriverIID)
}

// AttachVMPublicIP attaches an existing, user-created PublicIP (via PublicIP
// Manager) to the VM's default NIC. Fails (including the existing address) if
// the VM already has a default PublicIP.
func AttachVMPublicIP(connectionName string, vmName string, publicIPName string) (*cres.PublicIPInfo, error) {
	cblog.Info("call AttachVMPublicIP()")

	vmInfo, err := GetVM(connectionName, VM, vmName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if vmInfo.PublicIP != "" {
		err := fmt.Errorf("VM '%s' already has a default PublicIP: %s", vmName, vmInfo.PublicIP)
		cblog.Error(err)
		return nil, err
	}

	nicSystemId, privateIP := resolveDefaultNIC(*vmInfo)

	vmNameArg, nicNameArg, privateIPArg, err := buildAssociateArgs(connectionName, vmName, nicSystemId, privateIP)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return AssociatePublicIP(connectionName, publicIPName, vmNameArg, nicNameArg, privateIPArg)
}

// DetachVMPublicIP detaches a user-supplied PublicIP from the VM's default NIC
// WITHOUT deleting the PublicIP resource (it belongs to the user, not to this
// pairing). Fails if the VM has no default PublicIP assigned.
func DetachVMPublicIP(connectionName string, vmName string, publicIPName string) (bool, error) {
	cblog.Info("call DetachVMPublicIP()")

	vmInfo, err := GetVM(connectionName, VM, vmName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	if vmInfo.PublicIP == "" {
		err := fmt.Errorf("VM '%s' has no default PublicIP assigned", vmName)
		cblog.Error(err)
		return false, err
	}

	return DisassociatePublicIP(connectionName, publicIPName)
}
