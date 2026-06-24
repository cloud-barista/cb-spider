// Cloud Driver Interface of CB-Spider.
// GCP NIC Handler - GCP does not support standalone NIC lifecycle.
// NICs are created with VM instances and cannot be added/removed after creation.
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	compute "google.golang.org/api/compute/v1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type GCPNICHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

// GCP does not support standalone NIC creation.
func (h *GCPNICHandler) ListIID() ([]*irs.IID, error) {
	return nil, fmt.Errorf("GCP NIC Handler: ListIID not supported — GCP NICs are created with VM instances and cannot be managed independently")
}
func (h *GCPNICHandler) CreateNIC(req irs.NICReqInfo) (irs.NICInfo, error) {
	return irs.NICInfo{}, fmt.Errorf("GCP NIC Handler: CreateNIC not supported — GCP NICs are defined at VM creation time and cannot be added afterward")
}
func (h *GCPNICHandler) DeleteNIC(iid irs.IID) (bool, error) {
	return false, fmt.Errorf("GCP NIC Handler: DeleteNIC not supported — GCP NICs are removed only by deleting the VM")
}
func (h *GCPNICHandler) AttachNIC(nicIID irs.IID, vmIID irs.IID) (irs.NICInfo, error) {
	return irs.NICInfo{}, fmt.Errorf("GCP NIC Handler: AttachNIC not supported — GCP does not support adding NICs to existing VMs")
}
func (h *GCPNICHandler) DetachNIC(nicIID irs.IID) (bool, error) {
	return false, fmt.Errorf("GCP NIC Handler: DetachNIC not supported — GCP does not support removing NICs from existing VMs")
}

// ListNIC returns NICs from all VM instances in the zone (read-only view).
func (h *GCPNICHandler) ListNIC() ([]*irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, "All", "instances.list()")
	start := call.Start()
	projectID := h.Credential.ProjectID
	result, err := h.Client.Instances.List(projectID, h.Region.Zone).Context(h.Ctx).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return nil, err }
	LoggingInfo(hiscallInfo, start)
	var list []*irs.NICInfo
	for _, inst := range result.Items {
		for i, ni := range inst.NetworkInterfaces {
			info := extractGCPNICInfo(inst.Name, i, ni)
			list = append(list, &info)
		}
	}
	if list == nil { list = []*irs.NICInfo{} }
	return list, nil
}

// GetNIC returns a NIC from a VM. nicIID.NameId format: "{vmName}/nic{index}" e.g. "my-vm/nic0"
func (h *GCPNICHandler) GetNIC(iid irs.IID) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, iid.NameId, "instances.get()")
	start := call.Start()
	// Parse "vmName/nicN"
	parts := strings.SplitN(iid.NameId, "/", 2)
	if len(parts) != 2 { return irs.NICInfo{}, fmt.Errorf("GCP NIC GetNIC: NameId must be '{vmName}/nic{index}', got: %s", iid.NameId) }
	vmName, nicName := parts[0], parts[1]
	nicIdx := 0
	fmt.Sscanf(nicName, "nic%d", &nicIdx)
	projectID := h.Credential.ProjectID
	inst, err := h.Client.Instances.Get(projectID, h.Region.Zone, vmName).Context(h.Ctx).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return irs.NICInfo{}, err }
	if nicIdx >= len(inst.NetworkInterfaces) { return irs.NICInfo{}, fmt.Errorf("NIC index %d not found on VM %s", nicIdx, vmName) }
	LoggingInfo(hiscallInfo, start)
	return extractGCPNICInfo(vmName, nicIdx, inst.NetworkInterfaces[nicIdx]), nil
}

func extractGCPNICInfo(vmName string, idx int, ni *compute.NetworkInterface) irs.NICInfo {
	subnetParts := strings.Split(ni.Subnetwork, "/")
	vpcParts := strings.Split(ni.Network, "/")
	status := irs.NICAttached
	info := irs.NICInfo{
		IId:         irs.IID{NameId: fmt.Sprintf("%s/nic%d", vmName, idx), SystemId: ni.Name},
		SubnetIID:   irs.IID{SystemId: subnetParts[len(subnetParts)-1]},
		VpcIID:      irs.IID{SystemId: vpcParts[len(vpcParts)-1]},
		PrivateIP:   ni.NetworkIP,
		DeviceIndex: idx,
		Status:      status,
		CreatedTime: time.Time{},
	}
	info.OwnerVM = irs.IID{NameId: vmName, SystemId: vmName}
	if len(ni.AccessConfigs) > 0 && ni.AccessConfigs[0].NatIP != "" { info.PublicIP = ni.AccessConfigs[0].NatIP }
	info.KeyValueList = []irs.KeyValue{
		{Key: "NICName", Value: ni.Name},
		{Key: "Network", Value: ni.Network},
	}
	return info
}

// AddPrivateIP is not yet implemented for GCP.
// GCP supports alias IP ranges on network interfaces via instances.updateNetworkInterface,
// but alias IPs differ semantically from secondary private IPs in other CSPs
// (they are routed ranges, not directly assigned addresses).
// Use alias IP ranges via UpdateNetworkInterface — not yet implemented.
func (h *GCPNICHandler) AddPrivateIP(nicIID irs.IID, privateIP string) (irs.NICInfo, error) {
	return irs.NICInfo{}, fmt.Errorf("GCP AddPrivateIP: use alias IP ranges via instances.updateNetworkInterface — not yet implemented")
}

// RemovePrivateIP is not yet implemented for GCP.
// GCP primary NIC IPs cannot be changed on running instances.
// Secondary/alias IP ranges require instances.updateNetworkInterface — not yet implemented.
func (h *GCPNICHandler) RemovePrivateIP(nicIID irs.IID, privateIP string) (bool, error) {
	return false, fmt.Errorf("GCP RemovePrivateIP: use alias IP ranges via instances.updateNetworkInterface — not yet implemented")
}

// GetNICOSConfigScript returns an empty string for GCP.
// GCP NICs are configured at instance creation time; hot-attach is not supported,
// so no post-attach OS configuration script is required.
func (h *GCPNICHandler) GetNICOSConfigScript(nicIID irs.IID) (string, error) {
	return "", nil
}
