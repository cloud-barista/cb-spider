// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, August 2026.

package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/IBM/cloud-databases-go-sdk/clouddatabasesv5"
	"github.com/IBM/go-sdk-core/v5/core"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// fetchIbmPreferredVersion picks one supported, non-deprecated engine version to send as
// CreateCapability's Deployment.Version. IBM Cloud Databases' compute host flavors do not vary
// by DB engine version, so any valid version works — the preferred one is used when ListDeployables
// marks one, otherwise the first non-deprecated version.
func (handler *IbmRDBMSHandler) fetchIbmPreferredVersion(dbEngine string) (string, error) {
	resp, _, err := handler.CloudDBService.ListDeployablesWithContext(handler.getContext(), handler.CloudDBService.NewListDeployablesOptions())
	if err != nil {
		return "", fmt.Errorf("ListDeployables failed: %w", err)
	}
	if resp == nil {
		return "", errors.New("ListDeployables returned no deployable database metadata")
	}
	for _, deployable := range resp.Deployables {
		if deployable.Type == nil || *deployable.Type != dbEngine {
			continue
		}
		fallback := ""
		for _, version := range deployable.Versions {
			if version.Version == nil || version.Status == nil || *version.Status == clouddatabasesv5.DeployablesVersionsItemStatusDeprecatedConst {
				continue
			}
			if fallback == "" {
				fallback = *version.Version
			}
			if version.IsPreferred != nil && *version.IsPreferred {
				return *version.Version, nil
			}
		}
		if fallback != "" {
			return fallback, nil
		}
	}
	return "", fmt.Errorf("ListDeployables returned no supported version for engine '%s'", dbEngine)
}

// fetchIbmFlavors calls CreateCapability(capability_id="flavors"), which needs no existing
// deployment and returns the real per-host-flavor catalog (Cpu/Memory/HostingSize). This
// replaces the driver's former use of GetDefaultScalingGroups' HostFlavor parameter, which per
// its own SDK documentation only recognizes the literal value "multitenant" ("When a
// host_flavor of 'multitenant' is included with the request, IBM Cloud Database's new shared
// compute default groups will be returned") — passing any other flavor ID there is not a
// supported per-flavor lookup and silently returns the same generic default group every time,
// which is why every entry previously showed near-identical Memory and Cpu=0.
func (handler *IbmRDBMSHandler) fetchIbmFlavors(dbEngine string) ([]clouddatabasesv5.FlavorsCapabilityItem, error) {
	version, err := handler.fetchIbmPreferredVersion(dbEngine)
	if err != nil {
		return nil, err
	}
	deployment := &clouddatabasesv5.CreateCapabilityRequestDeployment{
		Type:     core.StringPtr(dbEngine),
		Version:  core.StringPtr(version),
		Platform: core.StringPtr("classic"),
		Location: core.StringPtr(handler.Region.Region),
	}
	options := handler.CloudDBService.NewCreateCapabilityOptions(clouddatabasesv5.CreateCapabilityOptionsCapabilityIDFlavorsConst)
	options.SetDeployment(deployment)

	resp, _, err := handler.CloudDBService.CreateCapabilityWithContext(handler.getContext(), options)
	if err != nil {
		return nil, fmt.Errorf("CreateCapability(flavors) failed for %s/%s in %s: %w", dbEngine, version, handler.Region.Region, err)
	}
	if resp == nil || resp.Capability == nil || len(resp.Capability.Flavors) == 0 {
		return nil, fmt.Errorf("CreateCapability(flavors) returned no flavors for %s/%s in %s", dbEngine, version, handler.Region.Region)
	}
	return resp.Capability.Flavors, nil
}

// fetchIbmHostFlavorSet returns the set of currently orderable host flavor IDs for dbEngine
// (via fetchIbmFlavors, which already includes "multitenant" alongside the dedicated flavors).
// Falls back to the static ibmRDBMSHostFlavorIDs catalog if the live call fails, so a transient
// API error degrades CreateRDBMS validation back to the old fixed list instead of blocking
// instance creation outright.
func (handler *IbmRDBMSHandler) fetchIbmHostFlavorSet(dbEngine string) map[string]bool {
	flavors, err := handler.fetchIbmFlavors(dbEngine)
	if err != nil {
		cblogger.Warnf("fetchIbmFlavors failed for %s, falling back to static host flavor list: %v", dbEngine, err)
		fallback := make(map[string]bool, len(ibmRDBMSHostFlavorIDs))
		for id := range ibmRDBMSHostFlavorIDs {
			fallback[id] = true
		}
		return fallback
	}
	set := make(map[string]bool, len(flavors))
	for _, flavor := range flavors {
		if flavor.ID != nil && *flavor.ID != "" {
			set[*flavor.ID] = true
		}
	}
	return set
}

// fetchIbmDeploymentDefaults calls GetDefaultScalingGroups without setting HostFlavor (the
// correct, generic default group query — the same usage RDBMSHandler.go's
// fetchRDBMSStorageSizeRange already makes) to read the member group's disk range and default
// member count. Unlike CPU/Memory, these are not flavor-specific in IBM Cloud Databases, so
// this is computed once per ListDBSpec call and reused for every flavor.
func (handler *IbmRDBMSHandler) fetchIbmDeploymentDefaults(dbEngine string) (irs.StorageSizeRange, int64, error) {
	resp, _, err := handler.CloudDBService.GetDefaultScalingGroupsWithContext(handler.getContext(), handler.CloudDBService.NewGetDefaultScalingGroupsOptions(dbEngine))
	if err != nil {
		return irs.StorageSizeRange{}, 0, fmt.Errorf("GetDefaultScalingGroups failed for %s: %w", dbEngine, err)
	}
	if resp == nil || len(resp.Groups) == 0 {
		return irs.StorageSizeRange{}, 0, fmt.Errorf("GetDefaultScalingGroups returned no scaling groups for %s", dbEngine)
	}

	var storageRange irs.StorageSizeRange
	var memberCount int64
	for _, group := range resp.Groups {
		if group.ID == nil || *group.ID != clouddatabasesv5.GroupIDMemberConst {
			continue
		}
		if group.Disk != nil && group.Disk.MinimumMb != nil && group.Disk.MaximumMb != nil {
			storageRange = irs.StorageSizeRange{
				Min: irs.GiBToGB(*group.Disk.MinimumMb / ibmStorageUnitGB),
				Max: irs.GiBToGB(*group.Disk.MaximumMb / ibmStorageUnitGB),
			}
		}
		if group.Members != nil && group.Members.AllocationCount != nil {
			memberCount = *group.Members.AllocationCount
		}
		break
	}
	return storageRange, memberCount, nil
}

// fetchIbmMultitenantCpuMemory calls GetDefaultScalingGroups WITH HostFlavor set to
// "multitenant" — the one case where that parameter is actually documented to do something
// (see fetchIbmFlavors) — to get the shared-compute default group's real CPU/Memory. The
// "multitenant" entry from CreateCapability(flavors) itself carries no fixed CPU/Memory (its
// capacity is expressed through this scaling group instead, not a per-flavor hardware spec).
func (handler *IbmRDBMSHandler) fetchIbmMultitenantCpuMemory(dbEngine string) (cpuCount int64, memMiB int64, err error) {
	options := handler.CloudDBService.NewGetDefaultScalingGroupsOptions(dbEngine)
	options.SetHostFlavor(clouddatabasesv5.GetDefaultScalingGroupsOptionsHostFlavorMultitenantConst)
	resp, _, err := handler.CloudDBService.GetDefaultScalingGroupsWithContext(handler.getContext(), options)
	if err != nil {
		return 0, 0, fmt.Errorf("GetDefaultScalingGroups(host_flavor=multitenant) failed for %s: %w", dbEngine, err)
	}
	if resp == nil || len(resp.Groups) == 0 {
		return 0, 0, fmt.Errorf("GetDefaultScalingGroups(host_flavor=multitenant) returned no scaling groups for %s", dbEngine)
	}
	for _, group := range resp.Groups {
		if group.ID == nil || *group.ID != clouddatabasesv5.GroupIDMemberConst {
			continue
		}
		if group.CPU != nil && group.CPU.AllocationCount != nil {
			cpuCount = *group.CPU.AllocationCount
		}
		if group.Memory != nil && group.Memory.AllocationMb != nil {
			memMiB = *group.Memory.AllocationMb
		}
		break
	}
	return cpuCount, memMiB, nil
}

// buildIbmDBSpecInfo converts one FlavorsCapabilityItem into DBSpecInfo.
// CPU.AllocationCount is a plain core count. Memory.AllocationMb is treated as MiB with no
// conversion, the same "Mb name, actually MiB" convention already applied to
// GetDefaultScalingGroups' Group.Memory.AllocationMb elsewhere in this driver.
func buildIbmDBSpecInfo(region, cbEngine string, flavor clouddatabasesv5.FlavorsCapabilityItem, storageRange irs.StorageSizeRange, memberCount int64) *irs.DBSpecInfo {
	name := "-1"
	if flavor.ID != nil && *flavor.ID != "" {
		name = *flavor.ID
	}
	info := &irs.DBSpecInfo{
		Region:     region,
		DBEngine:   cbEngine,
		Name:       name,
		VCpu:       irs.VCpuInfo{Count: "-1", ClockGHz: "-1"},
		MemSizeMiB: "-1",
	}
	if name == "-1" {
		info.MarkStatic("Name", "CreateCapability(flavors) returned an entry with no id")
	}

	if flavor.CPU != nil && flavor.CPU.AllocationCount != nil {
		info.VCpu.Count = strconv.FormatInt(*flavor.CPU.AllocationCount, 10)
	} else {
		info.MarkStatic("VCpu.Count", "CreateCapability(flavors) did not return CPU.AllocationCount for this flavor")
	}
	if flavor.Memory != nil && flavor.Memory.AllocationMb != nil {
		info.MemSizeMiB = strconv.FormatInt(*flavor.Memory.AllocationMb, 10)
	} else {
		info.MarkStatic("MemSizeMiB", "CreateCapability(flavors) did not return Memory.AllocationMb for this flavor")
	}
	if storageRange.Min > 0 || storageRange.Max > 0 {
		info.StorageSizeRangeGB = storageRange
	} else {
		info.MarkStatic("StorageSizeRangeGB", "GetDefaultScalingGroups did not return a member disk range for this engine")
	}

	if flavor.Name != nil && *flavor.Name != "" && *flavor.Name != name {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "FlavorName", Value: *flavor.Name})
	}
	if flavor.HostingSize != nil && *flavor.HostingSize != "" {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "HostingSize", Value: *flavor.HostingSize})
	}
	if memberCount > 0 {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "DefaultMemberCount", Value: strconv.FormatInt(memberCount, 10)})
	}
	return info
}

func (handler *IbmRDBMSHandler) ListDBSpec(dbEngine string) ([]*irs.DBSpecInfo, error) {
	cblogger.Debug("IBM Cloud ListDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListDBSpec", "CreateCapability(flavors)+GetDefaultScalingGroups()")
	start := call.Start()

	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return nil, err
	}
	if _, err := ibmRDBMSServiceID(requestedEngine); err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	flavors, err := handler.fetchIbmFlavors(requestedEngine)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	storageRange, memberCount, err := handler.fetchIbmDeploymentDefaults(requestedEngine)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	sort.Slice(flavors, func(i, j int) bool {
		iID, jID := "", ""
		if flavors[i].ID != nil {
			iID = *flavors[i].ID
		}
		if flavors[j].ID != nil {
			jID = *flavors[j].ID
		}
		return iID < jID
	})

	infoList := make([]*irs.DBSpecInfo, 0, len(flavors))
	for _, flavor := range flavors {
		info := buildIbmDBSpecInfo(handler.Region.Region, requestedEngine, flavor, storageRange, memberCount)
		if flavor.ID != nil && *flavor.ID == clouddatabasesv5.GetDefaultScalingGroupsOptionsHostFlavorMultitenantConst {
			if mtCPU, mtMemMiB, mtErr := handler.fetchIbmMultitenantCpuMemory(requestedEngine); mtErr == nil {
				if mtCPU > 0 {
					info.VCpu.Count = strconv.FormatInt(mtCPU, 10)
				} else {
					// Confirmed via two independent calls (CreateCapability(flavors) and this
					// group query) that IBM reports 0 here, not a missing/nil field: the
					// multitenant (shared compute) model has no fixed vCPU allocation, so "0"
					// is left as "-1" rather than shown as a literal, misleading CPU count.
					info.MarkStatic("VCpu.Count", "IBM's multitenant (shared compute) model does not report a fixed vCPU count; CPU is shared/variable rather than dedicated")
				}
				if mtMemMiB > 0 {
					info.MemSizeMiB = strconv.FormatInt(mtMemMiB, 10)
				}
			} else {
				note := fmt.Sprintf("GetDefaultScalingGroups(host_flavor=multitenant) failed: %v", mtErr)
				info.MarkStatic("VCpu.Count", note)
				info.MarkStatic("MemSizeMiB", note)
			}
		}
		infoList = append(infoList, info)
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
	return infoList, nil
}

func (handler *IbmRDBMSHandler) GetDBSpec(dbEngine string, name string) (irs.DBSpecInfo, error) {
	cblogger.Debug("IBM Cloud GetDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetDBSpec", "CreateCapability(flavors)+GetDefaultScalingGroups()")
	start := call.Start()

	infoList, err := handler.ListDBSpec(dbEngine)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DBSpecInfo{}, err
	}
	for _, info := range infoList {
		if info.Name == name {
			hiscallInfo.ElapsedTime = call.Elapsed(start)
			calllogger.Info(call.String(hiscallInfo))
			return *info, nil
		}
	}
	err = fmt.Errorf("DBSpec '%s' not found for engine '%s'", name, dbEngine)
	LoggingError(hiscallInfo, err)
	return irs.DBSpecInfo{}, err
}

func (handler *IbmRDBMSHandler) ListOrgDBSpec(dbEngine string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	if _, err := ibmRDBMSServiceID(requestedEngine); err != nil {
		return "", err
	}
	flavors, err := handler.fetchIbmFlavors(requestedEngine)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(flavors)
	if err != nil {
		return "", fmt.Errorf("failed to marshal raw flavors capability list: %w", err)
	}
	return string(b), nil
}

func (handler *IbmRDBMSHandler) GetOrgDBSpec(dbEngine string, name string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	if _, err := ibmRDBMSServiceID(requestedEngine); err != nil {
		return "", err
	}
	flavors, err := handler.fetchIbmFlavors(requestedEngine)
	if err != nil {
		return "", err
	}
	for _, flavor := range flavors {
		if flavor.ID != nil && *flavor.ID == name {
			b, err := json.Marshal(flavor)
			if err != nil {
				return "", fmt.Errorf("failed to marshal raw flavor: %w", err)
			}
			return string(b), nil
		}
	}
	return "", fmt.Errorf("DBSpec '%s' not found", name)
}
