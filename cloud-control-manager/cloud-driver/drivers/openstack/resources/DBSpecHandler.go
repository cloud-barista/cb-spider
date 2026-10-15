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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	computeflavors "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	dbflavors "github.com/gophercloud/gophercloud/v2/openstack/db/v1/flavors"
)

// fetchOpenStackTroveFlavors lists Trove (RDBMS) flavors, which carry RAM but not VCPUs.
func (handler *OpenStackRDBMSHandler) fetchOpenStackTroveFlavors() ([]dbflavors.Flavor, error) {
	if handler.DBClient == nil {
		return nil, errors.New("OpenStack Trove DB client is not initialized")
	}
	allPages, err := dbflavors.List(handler.DBClient).AllPages(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to list Trove flavors: %w", err)
	}
	flavorList, err := dbflavors.ExtractFlavors(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract Trove flavors: %w", err)
	}
	return flavorList, nil
}

// fetchOpenStackNovaFlavorsByName lists Nova compute flavors (VCPUs/RAM/Disk), keyed by
// name, for cross-referencing against Trove flavors — the same "Trove shares Nova flavors"
// assumption already used by resolveFlavorRef/resolveFlavorName for FlavorRef resolution.
func (handler *OpenStackRDBMSHandler) fetchOpenStackNovaFlavorsByName() (map[string]computeflavors.Flavor, error) {
	if handler.ComputeClient == nil {
		return nil, errors.New("OpenStack Nova compute client is not initialized")
	}
	allPages, err := computeflavors.ListDetail(handler.ComputeClient, computeflavors.ListOpts{}).AllPages(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to list Nova flavors: %w", err)
	}
	flavorList, err := computeflavors.ExtractFlavors(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract Nova flavors: %w", err)
	}
	byName := make(map[string]computeflavors.Flavor, len(flavorList))
	for _, f := range flavorList {
		if f.Name != "" {
			byName[f.Name] = f
		}
	}
	return byName, nil
}

// buildOpenStackDBSpecInfo converts one Trove flavor into DBSpecInfo,
// enriched with VCPUs/RAM from the matching Nova flavor (novaByName may be nil if the Nova
// cross-query failed). Nova's RAM field (gophercloud doc comment says "MB") is, by OpenStack
// convention, actually MiB, so it is used directly for MemSizeMiB with no numeric
// conversion. Trove's own Disk concept is intentionally not used here — Trove instances are
// backed by a separate Cinder volume sized via RDBMSInfo.StorageSize, unrelated to the
// flavor's (usually zero) local root disk.
func buildOpenStackDBSpecInfo(region, cbEngine string, troveFlavor dbflavors.Flavor, novaByName map[string]computeflavors.Flavor) *irs.DBSpecInfo {
	name := troveFlavor.Name
	if name == "" {
		name = troveFlavor.StrID
	}
	info := &irs.DBSpecInfo{
		Region:     region,
		DBEngine:   cbEngine,
		Name:       name,
		VCpu:       irs.VCpuInfo{Count: "-1", ClockGHz: "-1"},
		MemSizeMiB: "-1",
	}
	// StorageSizeRangeGB is intentionally left unset here (not "unknown due to an API gap"):
	// Trove instances are backed by a separate Cinder volume sized via RDBMSInfo.StorageSize
	// at creation time, not a range tied to this compute flavor. RDBMSMetaInfo.StorageSizeRangeGB
	// (the project's Cinder volume quota) is the relevant range instead.
	info.MarkStatic("StorageSizeRangeGB", "OpenStack Trove instances are backed by a separate Cinder volume sized via RDBMSInfo.StorageSize at creation time, not a range tied to the compute flavor; see RDBMSMetaInfo.StorageSizeRangeGB for the project's Cinder volume size range instead.")

	if novaFlavor, ok := novaByName[name]; ok {
		info.VCpu.Count = strconv.Itoa(novaFlavor.VCPUs)
		info.MemSizeMiB = strconv.Itoa(novaFlavor.RAM)
		return info
	}
	// Fall back to Trove's own RAM value (also OpenStack-convention MiB) when there is no
	// matching Nova flavor by name; VCPUs are unavailable from Trove alone.
	if troveFlavor.RAM > 0 {
		info.MemSizeMiB = strconv.Itoa(troveFlavor.RAM)
	} else {
		info.MarkStatic("MemSizeMiB", "Neither the matching Nova flavor nor the Trove flavor itself returned a RAM value")
	}
	info.MarkStatic("VCpu.Count", fmt.Sprintf("No Nova compute flavor named '%s' was found to cross-reference for VCPUs; Trove's own flavor API does not expose VCPUs", name))
	return info
}

func (handler *OpenStackRDBMSHandler) ListDBSpec(dbEngine string) ([]*irs.DBSpecInfo, error) {
	cblogger.Debug("OpenStack Trove ListDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.CredentialInfo.IdentityEndpoint, call.RDBMS, "ListDBSpec", "Trove flavors.List()+Nova flavors.ListDetail()")
	start := call.Start()

	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return nil, err
	}
	supportedEngines, err := handler.fetchSupportedEngines()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	if _, ok := supportedEngines[requestedEngine]; !ok {
		err := fmt.Errorf("DBEngine '%s' is not available in this OpenStack deployment's Trove datastore configuration", requestedEngine)
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	troveFlavors, err := handler.fetchOpenStackTroveFlavors()
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	novaByName, novaErr := handler.fetchOpenStackNovaFlavorsByName()
	if novaErr != nil {
		cblogger.Warnf("OpenStack DBSpec: Nova flavor cross-query failed, VCpu will be marked unavailable: %v", novaErr)
		novaByName = nil
	}

	sort.Slice(troveFlavors, func(i, j int) bool { return troveFlavors[i].Name < troveFlavors[j].Name })

	infoList := make([]*irs.DBSpecInfo, 0, len(troveFlavors))
	for _, f := range troveFlavors {
		infoList = append(infoList, buildOpenStackDBSpecInfo(handler.Region.Region, requestedEngine, f, novaByName))
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
	return infoList, nil
}

func (handler *OpenStackRDBMSHandler) GetDBSpec(dbEngine string, name string) (irs.DBSpecInfo, error) {
	cblogger.Debug("OpenStack Trove GetDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.CredentialInfo.IdentityEndpoint, call.RDBMS, "GetDBSpec", "Trove flavors.List()+Nova flavors.ListDetail()")
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

func (handler *OpenStackRDBMSHandler) ListOrgDBSpec(dbEngine string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	supportedEngines, err := handler.fetchSupportedEngines()
	if err != nil {
		return "", err
	}
	if _, ok := supportedEngines[requestedEngine]; !ok {
		return "", fmt.Errorf("DBEngine '%s' is not available in this OpenStack deployment", requestedEngine)
	}
	troveFlavors, err := handler.fetchOpenStackTroveFlavors()
	if err != nil {
		return "", err
	}
	novaByName, _ := handler.fetchOpenStackNovaFlavorsByName()

	type rawEntry struct {
		Trove dbflavors.Flavor       `json:"trove"`
		Nova  *computeflavors.Flavor `json:"nova,omitempty"`
	}
	raw := make([]rawEntry, 0, len(troveFlavors))
	for _, f := range troveFlavors {
		entry := rawEntry{Trove: f}
		if nova, ok := novaByName[f.Name]; ok {
			entry.Nova = &nova
		}
		raw = append(raw, entry)
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return "", fmt.Errorf("failed to marshal raw flavor list: %w", err)
	}
	return string(b), nil
}

func (handler *OpenStackRDBMSHandler) GetOrgDBSpec(dbEngine string, name string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	supportedEngines, err := handler.fetchSupportedEngines()
	if err != nil {
		return "", err
	}
	if _, ok := supportedEngines[requestedEngine]; !ok {
		return "", fmt.Errorf("DBEngine '%s' is not available in this OpenStack deployment", requestedEngine)
	}
	troveFlavors, err := handler.fetchOpenStackTroveFlavors()
	if err != nil {
		return "", err
	}
	novaByName, _ := handler.fetchOpenStackNovaFlavorsByName()
	for _, f := range troveFlavors {
		fname := f.Name
		if fname == "" {
			fname = f.StrID
		}
		if fname == name {
			result := map[string]interface{}{"trove": f}
			if nova, ok := novaByName[fname]; ok {
				result["nova"] = nova
			}
			b, err := json.Marshal(result)
			if err != nil {
				return "", fmt.Errorf("failed to marshal raw flavor: %w", err)
			}
			return string(b), nil
		}
	}
	return "", fmt.Errorf("DBSpec '%s' not found", name)
}
