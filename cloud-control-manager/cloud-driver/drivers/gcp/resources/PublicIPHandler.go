// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// GCP External IP Address Handler
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

type GCPPublicIPHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

// ListIID returns all regional External IP addresses.
func (h *GCPPublicIPHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, "ListIID", "addresses.list()")
	start := call.Start()

	projectID := h.Credential.ProjectID
	result, err := h.Client.Addresses.List(projectID, h.Region.Region).Context(h.Ctx).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	var iidList []*irs.IID
	for _, addr := range result.Items {
		iidList = append(iidList, &irs.IID{NameId: addr.Name, SystemId: addr.SelfLink})
	}
	return iidList, nil
}

// CreatePublicIP reserves a new static regional External IP address.
func (h *GCPPublicIPHandler) CreatePublicIP(reqInfo irs.PublicIPInfo) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, reqInfo.IId.NameId, "addresses.insert()")
	start := call.Start()

	projectID := h.Credential.ProjectID
	addr := &compute.Address{
		Name:        reqInfo.IId.NameId,
		AddressType: "EXTERNAL",
		NetworkTier: "PREMIUM",
		Description: "Managed by CB-Spider",
		Labels:      gcpPublicIPTagMap(reqInfo.TagList),
	}

	op, err := h.Client.Addresses.Insert(projectID, h.Region.Region, addr).Context(h.Ctx).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}

	if err := WaitForGCPRegionOperation(h.Client, h.Ctx, projectID, h.Region.Region, op.Name); err != nil {
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	return h.GetPublicIP(irs.IID{NameId: reqInfo.IId.NameId})
}

// ListPublicIP returns all regional External IP addresses.
func (h *GCPPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, "All", "addresses.list()")
	start := call.Start()

	projectID := h.Credential.ProjectID
	result, err := h.Client.Addresses.List(projectID, h.Region.Region).Context(h.Ctx).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	var infoList []*irs.PublicIPInfo
	for _, addr := range result.Items {
		info := extractGCPPublicIPInfo(addr)
		infoList = append(infoList, &info)
	}
	if infoList == nil {
		infoList = []*irs.PublicIPInfo{}
	}
	return infoList, nil
}

// GetPublicIP retrieves a single External IP address.
func (h *GCPPublicIPHandler) GetPublicIP(publicIPIID irs.IID) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "addresses.get()")
	start := call.Start()

	projectID := h.Credential.ProjectID
	resourceName := publicIPIID.NameId
	if publicIPIID.SystemId != "" && publicIPIID.NameId == "" {
		// selfLink: .../addresses/{name}
		parts := strings.Split(publicIPIID.SystemId, "/")
		resourceName = parts[len(parts)-1]
	}

	addr, err := h.Client.Addresses.Get(projectID, h.Region.Region, resourceName).Context(h.Ctx).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	info := extractGCPPublicIPInfo(addr)
	if publicIPIID.NameId != "" {
		info.IId.NameId = publicIPIID.NameId
	}
	return info, nil
}

// DeletePublicIP releases a static External IP address.
func (h *GCPPublicIPHandler) DeletePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "addresses.delete()")
	start := call.Start()

	projectID := h.Credential.ProjectID
	resourceName := publicIPIID.NameId
	if publicIPIID.SystemId != "" && resourceName == "" {
		parts := strings.Split(publicIPIID.SystemId, "/")
		resourceName = parts[len(parts)-1]
	}

	op, err := h.Client.Addresses.Delete(projectID, h.Region.Region, resourceName).Context(h.Ctx).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	if err := WaitForGCPRegionOperation(h.Client, h.Ctx, projectID, h.Region.Region, op.Name); err != nil {
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func extractGCPPublicIPInfo(addr *compute.Address) irs.PublicIPInfo {
	status := irs.PublicIPAvailable
	if addr.Status == "IN_USE" {
		status = irs.PublicIPAssociated
	}

	info := irs.PublicIPInfo{
		IId:             irs.IID{NameId: addr.Name, SystemId: addr.SelfLink},
		PublicIPAddress: addr.Address,
		Status:          status,
		CreatedTime:     gcpParseTime(addr.CreationTimestamp),
	}

	if len(addr.Users) > 0 {
		vmName := extractGCPVMNameFromUser(addr.Users[0])
		info.OwnedVM = irs.IID{NameId: vmName, SystemId: addr.Users[0]}
	}

	var tagList []irs.KeyValue
	for k, v := range addr.Labels {
		tagList = append(tagList, irs.KeyValue{Key: k, Value: v})
	}
	info.TagList = tagList

	info.KeyValueList = []irs.KeyValue{
		{Key: "Region", Value: extractGCPRegionFromSelfLink(addr.Region)},
		{Key: "AddressType", Value: addr.AddressType},
		{Key: "NetworkTier", Value: addr.NetworkTier},
		{Key: "Status", Value: addr.Status},
	}

	return info
}

func extractGCPVMNameFromUser(user string) string {
	// users[0]: .../instances/{vmName}
	parts := strings.Split(user, "/")
	for i, p := range parts {
		if p == "instances" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return "NA"
}

func extractGCPRegionFromSelfLink(regionLink string) string {
	parts := strings.Split(regionLink, "/")
	return parts[len(parts)-1]
}

func gcpParseTime(ts string) time.Time {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}
	}
	return t
}

func gcpPublicIPTagMap(tagList []irs.KeyValue) map[string]string {
	m := make(map[string]string)
	for _, kv := range tagList {
		m[strings.ToLower(kv.Key)] = kv.Value
	}
	return m
}

// WaitForGCPRegionOperation waits for a regional GCP operation to complete.
func WaitForGCPRegionOperation(client *compute.Service, ctx context.Context, project, region, opName string) error {
	for {
		op, err := client.RegionOperations.Get(project, region, opName).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("failed to get operation %s: %w", opName, err)
		}
		if op.Status == "DONE" {
			if op.Error != nil && len(op.Error.Errors) > 0 {
				return fmt.Errorf("operation %s failed: %s", opName, op.Error.Errors[0].Message)
			}
			return nil
		}
		time.Sleep(2 * time.Second)
	}
}

// AssociatePublicIP attaches a static External IP to a VM instance via accessConfig.
func (h *GCPPublicIPHandler) AssociatePublicIP(publicIPIID irs.IID, vmIID irs.IID, nicIID irs.IID, privateIP string) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "addAccessConfig()")
	start := call.Start()

	projectID := h.Credential.ProjectID
	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	// GCP: add accessConfig to the instance's network interface
	accessConfig := &compute.AccessConfig{
		Type:  "ONE_TO_ONE_NAT",
		Name:  "External NAT",
		NatIP: info.PublicIPAddress,
	}
	op, err := h.Client.Instances.AddAccessConfig(projectID, h.Region.Zone, vmIID.NameId, "nic0", accessConfig).Context(h.Ctx).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	if err := WaitForGCPZoneOperation(h.Client, h.Ctx, projectID, h.Region.Zone, op.Name); err != nil {
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	return h.GetPublicIP(publicIPIID)
}

// DisassociatePublicIP removes the accessConfig (external IP) from a VM.
func (h *GCPPublicIPHandler) DisassociatePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "deleteAccessConfig()")
	start := call.Start()

	projectID := h.Credential.ProjectID
	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return false, err
	}
	if info.Status != irs.PublicIPAssociated || info.OwnedVM.NameId == "" {
		return false, fmt.Errorf("PublicIP %s is not associated", publicIPIID.NameId)
	}
	vmName := info.OwnedVM.NameId

	op, err := h.Client.Instances.DeleteAccessConfig(projectID, h.Region.Zone, vmName, "External NAT", "nic0").Context(h.Ctx).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	if err := WaitForGCPZoneOperation(h.Client, h.Ctx, projectID, h.Region.Zone, op.Name); err != nil {
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

// WaitForGCPZoneOperation waits for a zone-level GCP operation to complete.
func WaitForGCPZoneOperation(client *compute.Service, ctx context.Context, project, zone, opName string) error {
	for {
		op, err := client.ZoneOperations.Get(project, zone, opName).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("failed to get zone operation %s: %w", opName, err)
		}
		if op.Status == "DONE" {
			if op.Error != nil && len(op.Error.Errors) > 0 {
				return fmt.Errorf("zone operation %s failed: %s", opName, op.Error.Errors[0].Message)
			}
			return nil
		}
		time.Sleep(2 * time.Second)
	}
}
