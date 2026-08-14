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

import "math"

// DBSpecInfo represents one orderable Database instance spec/class/flavor for a
// specific DB engine (mysql | mariadb | postgresql). It complements RDBMSMetaInfo's
// DBSpecOptions ([]string, names only) with vCPU/Memory/StorageSize details,
// mirroring VMSpecInfo's field/unit conventions (VCpuInfo, MemSizeMiB, KeyValueList).
// @description Database Instance Spec Information for CSP-specific DBSpec values
type DBSpecInfo struct {
	Region   string `json:"Region" example:"us-east-1"`  // Region where this spec is orderable
	DBEngine string `json:"DBEngine" example:"mysql"`    // mysql | mariadb | postgresql — engine this spec applies to
	Name     string `json:"Name" example:"db.t3.medium"` // CSP original spec/class/flavor name

	VCpu       VCpuInfo `json:"VCpu"`                      // CPU details. Count is "-1" when the CSP does not expose it and no objective cross-service basis exists to derive it
	MemSizeMiB string   `json:"MemSizeMiB" example:"4096"` // Memory size in MiB. "-1" when the CSP's native unit cannot be objectively confirmed (see DataSourceNotes)

	// StorageSizeRangeGB is the valid StorageSize range (GB) the caller may request for
	// THIS spec when creating a database instance — NOT a disk built into the spec.
	// {-1,-1} means "no per-spec constraint known" (falls back to RDBMSMetaInfo.StorageSizeRangeGB).
	StorageSizeRangeGB StorageSizeRange `json:"StorageSizeRangeGB,omitempty"`

	// DataSource records, per field name, whether that field's value was obtained live
	// from the CSP API ("API") or is a fixed/unconvertible value ("Static") for this
	// response. A field with no entry here is "API". Same convention as RDBMSMetaInfo.
	DataSource      map[string]RDBMSDataSource `json:"DataSource,omitempty"`
	DataSourceNotes map[string]string          `json:"DataSourceNotes,omitempty"`

	KeyValueList []KeyValue `json:"KeyValueList,omitempty"` // CSP-specific extras (e.g. raw source value/unit, IOPS, architecture, reference price)
}

// MarkStatic records that the given field is a fixed/unconvertible value for this
// response rather than a live CSP API result, with an optional explanation.
// Same convention as RDBMSMetaInfo.MarkStatic.
func (m *DBSpecInfo) MarkStatic(field string, note string) {
	if m.DataSource == nil {
		m.DataSource = map[string]RDBMSDataSource{}
	}
	m.DataSource[field] = DataSourceStatic
	if note != "" {
		if m.DataSourceNotes == nil {
			m.DataSourceNotes = map[string]string{}
		}
		m.DataSourceNotes[field] = note
	}
}

// HasNoSpecData reports whether this entry carries neither vCPU count nor memory size
// (both still "-1"), meaning it offers nothing a caller could use to choose it over another
// spec. CSP drivers should exclude such entries from ListDBSpec results — see
// FilterDBSpecsWithNoData — while still returning them as-is from GetDBSpec/
// ListOrgDBSpec/GetOrgDBSpec, since those are explicit per-name lookups where
// silently hiding the (honest, still informative) failure reason would be confusing.
func (m *DBSpecInfo) HasNoSpecData() bool {
	return m.VCpu.Count == "-1" && m.MemSizeMiB == "-1"
}

// FilterDBSpecsWithNoData drops entries with no usable vCPU/memory data (see
// HasNoSpecData) from a ListDBSpec result. Called explicitly by each CSP driver's
// ListDBSpec, not injected centrally, so each driver keeps direct control over what it
// returns.
func FilterDBSpecsWithNoData(list []*DBSpecInfo) []*DBSpecInfo {
	filtered := make([]*DBSpecInfo, 0, len(list))
	for _, info := range list {
		if info.HasNoSpecData() {
			continue
		}
		filtered = append(filtered, info)
	}
	return filtered
}

// BytesToMiB converts a byte count to the nearest whole MiB (2^20 bytes), rounding
// to the nearest integer. Use for CSP fields objectively documented as raw bytes
// (e.g. GCP Tier.RAM, NCP MemorySize).
func BytesToMiB(bytes int64) int64 {
	if bytes <= 0 {
		return bytes
	}
	return int64(math.Round(float64(bytes) / (1024 * 1024)))
}

// BytesToGB converts a byte count to the nearest whole decimal GB (10^9 bytes), rounding
// to the nearest integer. Use for CSP fields objectively documented as raw bytes
// (e.g. GCP DiskQuota, NCP BaseBlockStorageSize/DataStorageSize).
func BytesToGB(bytes int64) int64 {
	if bytes <= 0 {
		return bytes
	}
	return int64(math.Round(float64(bytes) / 1000000000))
}

// -------- DBSpec Handler API
type DBSpecHandler interface {
	ListDBSpec(dbEngine string) ([]*DBSpecInfo, error)
	GetDBSpec(dbEngine string, Name string) (DBSpecInfo, error)

	ListOrgDBSpec(dbEngine string) (string, error)             // return string: json format
	GetOrgDBSpec(dbEngine string, Name string) (string, error) // return string: json format
}
