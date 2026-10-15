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
	"fmt"
	"regexp"
	"sort"
	"strconv"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

var (
	// db-custom-<vCPU>-<memoryMB>, or db-<family>-custom-<vCPU>-<memoryMB> for newer families
	// (e.g. db-n2-custom-4-16384, db-e2-custom-2-8192).
	gcpCustomTierPattern = regexp.MustCompile(`^db-(?:[a-z0-9]+-)?custom-(\d+)-\d+$`)
	// db-<family>-<type>-<vCPU>, covering every predefined machine family Cloud SQL has ever
	// offered (not just n1): e.g. db-n1-standard-4, db-n2-highmem-8, db-c4a-highmem-2,
	// db-e2-highcpu-16, db-t2a-standard-1, db-m1-ultramem-40. The trailing numeric segment is
	// the vCPU count in all of these; only the shared-core tiers below have no such suffix.
	gcpPredefinedTierPattern = regexp.MustCompile(`^db-[a-z0-9]+-[a-z]+-(\d+)$`)
)

// gcpVCPUFromTierName derives vCPU count from the Cloud SQL tier name convention, since
// Tiers.List() does not return a separate vCPU field. Shared-core tiers (db-f1-micro,
// db-g1-small, db-e2-micro/small/medium) have no whole-core count and are not matched.
func gcpVCPUFromTierName(tier string) (string, bool) {
	if m := gcpCustomTierPattern.FindStringSubmatch(tier); m != nil {
		return m[1], true
	}
	if m := gcpPredefinedTierPattern.FindStringSubmatch(tier); m != nil {
		return m[1], true
	}
	return "", false
}

// buildGCPDBSpecInfo converts one Cloud SQL Tier into DBSpecInfo.
// Tier.RAM/DiskQuota are documented by the SDK as raw bytes (no unit ambiguity), so
// BytesToMiB/BytesToGB are applied directly rather than guessed.
func buildGCPDBSpecInfo(region, cbEngine string, tier *sqladmin.Tier) *irs.DBSpecInfo {
	info := &irs.DBSpecInfo{
		Region:     region,
		DBEngine:   cbEngine,
		Name:       tier.Tier,
		VCpu:       irs.VCpuInfo{Count: "-1", ClockGHz: "-1"},
		MemSizeMiB: "-1",
	}
	if vcpu, ok := gcpVCPUFromTierName(tier.Tier); ok {
		info.VCpu.Count = vcpu
	} else {
		info.MarkStatic("VCpu.Count", "Cloud SQL Tiers.List() has no vCPU field; this tier name does not match a known db-custom-N-M / db-n1-standard-N / db-n1-highmem-N naming convention (e.g. it may be a shared-core tier like db-f1-micro/db-g1-small with a fractional vCPU)")
	}
	if tier.RAM > 0 {
		info.MemSizeMiB = strconv.FormatInt(irs.BytesToMiB(tier.RAM), 10)
	} else {
		info.MarkStatic("MemSizeMiB", "Cloud SQL Tiers.List() did not return a positive RAM value for this tier")
	}
	if tier.DiskQuota > 0 {
		info.StorageSizeRangeGB = irs.StorageSizeRange{Min: -1, Max: irs.BytesToGB(tier.DiskQuota)}
		info.MarkStatic("StorageSizeRangeGB.Min", "Cloud SQL Tiers.List() only returns a maximum disk quota per tier, not a minimum; RDBMSMetaInfo.StorageSizeRangeGB.Min (static 10GB) applies instead")
	}
	return info
}

func (handler *GCPRDBMSHandler) fetchGCPTiers() ([]*sqladmin.Tier, error) {
	projectID := handler.getProjectId()
	if projectID == "" {
		return nil, fmt.Errorf("GCP project ID is empty")
	}
	resp, err := handler.Client.Tiers.List(projectID).Do()
	if err != nil {
		return nil, fmt.Errorf("Cloud SQL Tiers.List failed: %w", err)
	}
	var tiers []*sqladmin.Tier
	for _, tier := range resp.Items {
		if tier == nil || tier.Tier == "" {
			continue
		}
		if !cloudSQLTierSupportsRegion(tier, handler.Region.Region) {
			continue
		}
		tiers = append(tiers, tier)
	}
	if len(tiers) == 0 {
		return nil, fmt.Errorf("Cloud SQL Tiers.List returned no tiers for region %s", handler.Region.Region)
	}
	return tiers, nil
}

func (handler *GCPRDBMSHandler) ListDBSpec(dbEngine string) ([]*irs.DBSpecInfo, error) {
	cblogger.Debug("GCP Cloud SQL ListDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListDBSpec", "Tiers.List()")
	start := call.Start()

	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return nil, err
	}
	if requestedEngine == "mariadb" {
		err := fmt.Errorf("GCP Cloud SQL does not support mariadb")
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	tiers, err := handler.fetchGCPTiers()
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	sort.Slice(tiers, func(i, j int) bool { return tiers[i].Tier < tiers[j].Tier })

	infoList := make([]*irs.DBSpecInfo, 0, len(tiers))
	for _, tier := range tiers {
		infoList = append(infoList, buildGCPDBSpecInfo(handler.Region.Region, requestedEngine, tier))
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
	return infoList, nil
}

func (handler *GCPRDBMSHandler) GetDBSpec(dbEngine string, name string) (irs.DBSpecInfo, error) {
	cblogger.Debug("GCP Cloud SQL GetDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetDBSpec", "Tiers.List()")
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

func (handler *GCPRDBMSHandler) ListOrgDBSpec(dbEngine string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	if requestedEngine == "mariadb" {
		return "", fmt.Errorf("GCP Cloud SQL does not support mariadb")
	}
	tiers, err := handler.fetchGCPTiers()
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(tiers)
	if err != nil {
		return "", fmt.Errorf("failed to marshal raw Tier list: %w", err)
	}
	return string(b), nil
}

func (handler *GCPRDBMSHandler) GetOrgDBSpec(dbEngine string, name string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	if requestedEngine == "mariadb" {
		return "", fmt.Errorf("GCP Cloud SQL does not support mariadb")
	}
	tiers, err := handler.fetchGCPTiers()
	if err != nil {
		return "", err
	}
	for _, tier := range tiers {
		if tier.Tier == name {
			b, err := json.Marshal(tier)
			if err != nil {
				return "", fmt.Errorf("failed to marshal raw Tier: %w", err)
			}
			return string(b), nil
		}
	}
	return "", fmt.Errorf("DBSpec '%s' not found", name)
}
