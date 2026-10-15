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
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// alibabaClassCatalogCommodityCodes lists every primary-instance CommodityCode this driver
// checks when building the Cpu/Memory catalog via ListClasses. The region being queried does
// NOT tell us whether the underlying Alibaba account is registered on the China site
// (bards/rds) or the International site (bards_intl/rds_intl) — an International-site account
// can still deploy into a China region (confirmed directly against this account's own RDS buy
// console session: it showed Region "China (Beijing)" being ordered through the
// alibabacloud.com International console domain). So rather than guessing the CommodityCode
// from the region string, all four are queried via ListClasses and merged.
func alibabaClassCatalogCommodityCodes() []string {
	return []string{"bards", "bards_intl", "rds", "rds_intl"}
}

// alibabaMemoryClassGBPattern extracts the leading numeric value from a MemoryClass string
// such as "16GB" or "16GB（通用型）".
var alibabaMemoryClassGBPattern = regexp.MustCompile(`^(\d+(?:\.\d+)?)`)

// alibabaParseMemoryClassGiB parses a ListClasses MemoryClass value and treats it as GiB, not
// the decimal GB its API reference literally states ("实例规格对应的内存容量。单位：GB" —
// https://help.aliyun.com/zh/rds/apsaradb-rds-for-mysql/api-rds-2014-08-15-listclasses-mysql).
// Cross-checked against VMSpec: ECS's DescribeInstanceTypes.MemorySize is explicitly documented
// as "Unit: GiB", and RDS class codes pair up with same-named ECS families 1:1 (e.g.
// rds.mysql.s2.large / ecs.s2.large, both 2 vCPU) — cloud VM memory is essentially always
// allocated in binary-aligned amounts (4096 MiB, not an arbitrary 4,000,000,000-byte figure),
// so despite the "GB" label this is treated the same as VMSpec's GiB convention for consistency.
func alibabaParseMemoryClassGiB(raw string) (int64, bool) {
	m := alibabaMemoryClassGBPattern.FindString(strings.TrimSpace(raw))
	if m == "" {
		return 0, false
	}
	gib, err := strconv.ParseFloat(m, 64)
	if err != nil {
		return 0, false
	}
	return int64(math.Round(gib)), true
}

// fetchAlibabaClassCatalog calls ListClasses, a flat catalog of every sellable RDS hardware
// class (Cpu/MemoryClass/MaxIOPS/MaxConnections/MaxIOMBPS/ReferencePrice/ClassGroup), keyed by
// ClassCode. Unlike DescribeClassDetails — which requires an exact CommodityCode+EngineVersion
// match per class and fails with InvalidClassCode.NotFound whenever that combination isn't the
// one the class happens to be registered under — ListClasses has neither an EngineVersion nor
// a ZoneId parameter, so a class appears here as long as it is sellable at all, independent of
// which zone/engine-version this connection happens to be scoped to. This replaces the former
// per-class, per-version DescribeClassDetails retry loop entirely.
func (handler *AlibabaRDBMSHandler) fetchAlibabaClassCatalog() (map[string]rds.ClassList, error) {
	catalog := map[string]rds.ClassList{}
	for _, commodityCode := range alibabaClassCatalogCommodityCodes() {
		req := rds.CreateListClassesRequest()
		req.CommodityCode = commodityCode
		req.OrderType = "BUY"
		req.QueryParams["RegionId"] = handler.Region.Region

		resp, err := handler.Client.ListClasses(req)
		if err != nil {
			cblogger.Debugf("ListClasses failed for CommodityCode '%s' (region %s): %v", commodityCode, handler.Region.Region, err)
			continue
		}
		for _, class := range resp.Items {
			if class.ClassCode == "" {
				continue
			}
			if _, ok := catalog[class.ClassCode]; !ok {
				catalog[class.ClassCode] = class
			}
		}
	}
	if len(catalog) == 0 {
		return nil, fmt.Errorf("ListClasses returned no classes for any of CommodityCode %v in region %s", alibabaClassCatalogCommodityCodes(), handler.Region.Region)
	}
	return catalog, nil
}

// fetchAlibabaStorageRangesByClass calls DescribeAvailableClasses across the given
// (zone, engineVersion, category, storageType) lookup combinations — the same combinations
// fetchRDBMSInstanceOptions() iterates for GetMetaInfo() — but instead of collapsing every
// class's DBInstanceStorageRange into one engine-wide min/max, it keeps a per-class-name
// StorageSizeRange (merged as min-of-mins/max-of-maxes across the lookups the class name
// appears in, since the same class can be offered under more than one zone/storageType).
// This is what fills DBSpecInfo.StorageSizeRangeGB, which ListClasses does not provide
// (it has no storage-range fields either).
func (handler *AlibabaRDBMSHandler) fetchAlibabaStorageRangesByClass(engineNames []alibabaRDBMSEngine, storageLookups map[string][]alibabaRDBMSStorageLookup) (map[string]irs.StorageSizeRange, string, error) {
	latestVersionByEngine := map[string]string{}
	for _, eng := range engineNames {
		for _, lookup := range storageLookups[eng.aliName] {
			if latestVersionByEngine[eng.aliName] == "" || alibabaCompareVersionStrings(lookup.engineVersion, latestVersionByEngine[eng.aliName]) > 0 {
				latestVersionByEngine[eng.aliName] = lookup.engineVersion
			}
		}
	}
	var latestVersion string
	if len(engineNames) > 0 {
		latestVersion = latestVersionByEngine[engineNames[0].aliName]
	}

	ranges := map[string]irs.StorageSizeRange{}
	for _, eng := range engineNames {
		seenLookup := map[string]bool{}
		for _, lookup := range storageLookups[eng.aliName] {
			if lookup.engineVersion != latestVersionByEngine[eng.aliName] {
				continue
			}
			lookupKey := lookup.zoneID + ":" + lookup.engineVersion + ":" + lookup.category + ":" + lookup.storageType
			if seenLookup[lookupKey] {
				continue
			}
			seenLookup[lookupKey] = true

			classesReq := rds.CreateDescribeAvailableClassesRequest()
			classesReq.Engine = eng.aliName
			classesReq.EngineVersion = lookup.engineVersion
			classesReq.DBInstanceStorageType = lookup.storageType
			classesReq.InstanceChargeType = "Postpaid"
			classesReq.ZoneId = lookup.zoneID
			classesReq.Category = lookup.category

			classesResp, err := handler.describeAvailableClasses(classesReq)
			if err != nil {
				return nil, "", fmt.Errorf("DescribeAvailableClasses failed for engine %s version %s category %s storage type %s zone %s: %w", eng.aliName, lookup.engineVersion, lookup.category, lookup.storageType, lookup.zoneID, err)
			}

			for _, class := range classesResp.DBInstanceClasses {
				if class.DBInstanceClass == "" {
					continue
				}
				classMin, classMax := alibabaStorageRangeValues(class)
				if classMin <= 0 && classMax <= 0 {
					continue
				}
				existing, ok := ranges[class.DBInstanceClass]
				if !ok {
					ranges[class.DBInstanceClass] = irs.StorageSizeRange{Min: classMin, Max: classMax}
					continue
				}
				if classMin > 0 && (existing.Min == 0 || classMin < existing.Min) {
					existing.Min = classMin
				}
				if classMax > existing.Max {
					existing.Max = classMax
				}
				ranges[class.DBInstanceClass] = existing
			}
		}
	}
	return ranges, latestVersion, nil
}

// buildAlibabaDBSpecInfo looks up one class code in the ListClasses catalog (Cpu is a
// plain core count; MemoryClass is GB per Alibaba's own API reference, so it is converted to
// MiB) and merges in the per-class storage range obtained separately from
// DescribeAvailableClasses (see fetchAlibabaStorageRangesByClass) — ListClasses itself has no
// storage fields.
func buildAlibabaDBSpecInfo(cbEngine, className string, catalog map[string]rds.ClassList, storageRange irs.StorageSizeRange) *irs.DBSpecInfo {
	info := &irs.DBSpecInfo{
		Region:     "",
		DBEngine:   cbEngine,
		Name:       className,
		VCpu:       irs.VCpuInfo{Count: "-1", ClockGHz: "-1"},
		MemSizeMiB: "-1",
	}

	if storageRange.Min > 0 || storageRange.Max > 0 {
		// No unit conversion applied: DescribeAvailableClasses' DBInstanceStorageRange/
		// StorageRange is widely documented as decimal GB, but the vendored SDK (auto
		// generated) carries no unit comment to confirm this independently.
		info.StorageSizeRangeGB = storageRange
	} else {
		info.MarkStatic("StorageSizeRangeGB", fmt.Sprintf("DescribeAvailableClasses did not return a storage range for class '%s' in the connection's current zone/engine-version", className))
	}

	class, ok := catalog[className]
	if !ok {
		note := fmt.Sprintf("ListClasses (CommodityCode bards/bards_intl/rds/rds_intl) does not list class '%s' at all; likely a retired/legacy hardware generation no longer sold", className)
		info.MarkStatic("VCpu.Count", note)
		info.MarkStatic("MemSizeMiB", note)
		return info
	}

	if class.Cpu != "" {
		info.VCpu.Count = class.Cpu
	} else {
		info.MarkStatic("VCpu.Count", "ListClasses returned an empty Cpu value for this class")
	}

	if gib, ok := alibabaParseMemoryClassGiB(class.MemoryClass); ok {
		info.MemSizeMiB = irs.ConvertGiBToMiBInt64(gib)
	} else {
		info.MarkStatic("MemSizeMiB", "ListClasses returned an empty or unparsable MemoryClass value for this class")
	}

	if class.MemoryClass != "" {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "RawMemoryClass", Value: class.MemoryClass})
	}
	if class.ClassGroup != "" {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "ClassGroup", Value: class.ClassGroup})
	}
	if class.MaxIOPS != "" {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "MaxIOPS", Value: class.MaxIOPS})
	}
	if class.MaxConnections != "" {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "MaxConnections", Value: class.MaxConnections})
	}
	if class.MaxIOMBPS != "" {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "MaxIOMBPS", Value: class.MaxIOMBPS})
	}
	if class.InstructionSetArch != "" {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "InstructionSetArch", Value: class.InstructionSetArch})
	}
	if class.ReferencePrice != "" {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "ReferencePrice", Value: class.ReferencePrice})
	}
	if class.EncryptedMemory != "" && class.EncryptedMemory != "0" {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "EncryptedMemoryGB", Value: class.EncryptedMemory})
	}
	return info
}

func (handler *AlibabaRDBMSHandler) ListDBSpec(dbEngine string) ([]*irs.DBSpecInfo, error) {
	cblogger.Debug("Alibaba RDS ListDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListDBSpec", "DescribeAvailableClasses()+ListClasses()")
	start := call.Start()

	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return nil, err
	}

	engineNames, _, _, storageLookups, err := handler.fetchRDBMSZoneDiscovery(requestedEngine)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	storageRanges, _, err := handler.fetchAlibabaStorageRangesByClass(engineNames, storageLookups)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	catalog, err := handler.fetchAlibabaClassCatalog()
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	classNames := make([]string, 0, len(storageRanges))
	for className := range storageRanges {
		classNames = append(classNames, className)
	}
	sort.Strings(classNames)

	infoList := make([]*irs.DBSpecInfo, 0, len(classNames))
	for _, className := range classNames {
		info := buildAlibabaDBSpecInfo(requestedEngine, className, catalog, storageRanges[className])
		info.Region = handler.Region.Region
		infoList = append(infoList, info)
	}
	infoList = irs.FilterDBSpecsWithNoData(infoList)

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
	return infoList, nil
}

func (handler *AlibabaRDBMSHandler) GetDBSpec(dbEngine string, name string) (irs.DBSpecInfo, error) {
	cblogger.Debug("Alibaba RDS GetDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetDBSpec", "DescribeAvailableClasses()+ListClasses()")
	start := call.Start()

	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return irs.DBSpecInfo{}, err
	}

	engineNames, _, _, storageLookups, err := handler.fetchRDBMSZoneDiscovery(requestedEngine)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return irs.DBSpecInfo{}, err
	}
	storageRanges, _, err := handler.fetchAlibabaStorageRangesByClass(engineNames, storageLookups)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return irs.DBSpecInfo{}, err
	}
	catalog, err := handler.fetchAlibabaClassCatalog()
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return irs.DBSpecInfo{}, err
	}

	info := buildAlibabaDBSpecInfo(requestedEngine, name, catalog, storageRanges[name])
	info.Region = handler.Region.Region
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
	return *info, nil
}

func (handler *AlibabaRDBMSHandler) ListOrgDBSpec(dbEngine string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	engineNames, _, _, storageLookups, err := handler.fetchRDBMSZoneDiscovery(requestedEngine)
	if err != nil {
		return "", err
	}
	storageRanges, _, err := handler.fetchAlibabaStorageRangesByClass(engineNames, storageLookups)
	if err != nil {
		return "", err
	}
	catalog, err := handler.fetchAlibabaClassCatalog()
	if err != nil {
		return "", err
	}

	classNames := make([]string, 0, len(storageRanges))
	for className := range storageRanges {
		classNames = append(classNames, className)
	}
	sort.Strings(classNames)

	type rawEntry struct {
		ClassDetails *rds.ClassList       `json:"classDetails,omitempty"`
		StorageRange irs.StorageSizeRange `json:"storageRangeGB"`
	}
	results := make([]rawEntry, 0, len(classNames))
	for _, className := range classNames {
		entry := rawEntry{StorageRange: storageRanges[className]}
		if class, ok := catalog[className]; ok {
			entry.ClassDetails = &class
		}
		results = append(results, entry)
	}
	b, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("failed to marshal raw ListClasses list: %w", err)
	}
	return string(b), nil
}

func (handler *AlibabaRDBMSHandler) GetOrgDBSpec(dbEngine string, name string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	engineNames, _, _, storageLookups, err := handler.fetchRDBMSZoneDiscovery(requestedEngine)
	if err != nil {
		return "", err
	}
	storageRanges, _, err := handler.fetchAlibabaStorageRangesByClass(engineNames, storageLookups)
	if err != nil {
		return "", err
	}
	catalog, err := handler.fetchAlibabaClassCatalog()
	if err != nil {
		return "", err
	}

	class, ok := catalog[name]
	if !ok {
		return "", fmt.Errorf("ListClasses does not list class '%s'", name)
	}
	result := map[string]interface{}{
		"classDetails":   class,
		"storageRangeGB": storageRanges[name],
	}
	b, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal raw ListClasses entry: %w", err)
	}
	return string(b), nil
}
