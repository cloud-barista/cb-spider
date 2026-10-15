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

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cdb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdb/v20170320"
)

// fetchTencentSellConfigsForZone re-derives, from DescribeCdbZoneConfig, the CdbSellConfig
// entries actually selectable in the connection's configured region/zone — the same
// zone-matching logic as fetchCDBMetaOptions, but returning the configs themselves
// (deduplicated by Memory, since Tencent's DBSpec identity IS the Memory value)
// instead of collapsing them into aggregate name/storage-range lists.
func (handler *TencentRDBMSHandler) fetchTencentSellConfigsForZone() ([]*cdb.CdbSellConfig, error) {
	if handler.Client == nil {
		return nil, errors.New("CDB client is unavailable")
	}

	request := cdb.NewDescribeCdbZoneConfigRequest()
	response, err := handler.Client.DescribeCdbZoneConfig(request)
	if err != nil {
		return nil, err
	}
	if response.Response == nil || response.Response.DataResult == nil {
		return nil, errors.New("DescribeCdbZoneConfig returned empty response")
	}

	data := response.Response.DataResult
	configMap := make(map[int64]*cdb.CdbSellConfig)
	for _, cfg := range data.Configs {
		if cfg == nil || cfg.Id == nil {
			continue
		}
		if cfg.Status != nil && *cfg.Status != 0 {
			continue
		}
		configMap[*cfg.Id] = cfg
	}
	if len(configMap) == 0 {
		return nil, errors.New("DescribeCdbZoneConfig returned no available CDB sell configs")
	}

	matchedZone := false
	selectedConfigIDs := make(map[int64]struct{})
	for _, regionConf := range data.Regions {
		if regionConf == nil || regionConf.Region == nil {
			continue
		}
		if handler.Region.Region != "" && *regionConf.Region != handler.Region.Region {
			continue
		}
		for _, zoneConf := range regionConf.RegionConfig {
			if zoneConf == nil || zoneConf.Zone == nil {
				continue
			}
			if handler.Region.Zone != "" && *zoneConf.Zone != handler.Region.Zone {
				continue
			}
			if zoneConf.Status != nil && *zoneConf.Status != 1 {
				continue
			}
			matchedZone = true
			for _, sellType := range zoneConf.SellType {
				if sellType == nil {
					continue
				}
				for _, cfgID := range sellType.ConfigIds {
					if cfgID != nil {
						selectedConfigIDs[*cfgID] = struct{}{}
					}
				}
			}
		}
	}
	if !matchedZone {
		return nil, fmt.Errorf("DescribeCdbZoneConfig returned no online zone for region [%s], zone [%s]", handler.Region.Region, handler.Region.Zone)
	}
	if len(selectedConfigIDs) == 0 {
		return nil, fmt.Errorf("DescribeCdbZoneConfig returned no available CDB config IDs for region [%s], zone [%s]", handler.Region.Region, handler.Region.Zone)
	}

	seenMemory := map[int64]bool{}
	var configs []*cdb.CdbSellConfig
	for cfgID := range selectedConfigIDs {
		cfg, ok := configMap[cfgID]
		if !ok || cfg.Memory == nil || *cfg.Memory <= 0 {
			continue
		}
		if seenMemory[*cfg.Memory] {
			continue
		}
		seenMemory[*cfg.Memory] = true
		configs = append(configs, cfg)
	}
	if len(configs) == 0 {
		return nil, fmt.Errorf("DescribeCdbZoneConfig returned no usable sell configs for region [%s], zone [%s]", handler.Region.Region, handler.Region.Zone)
	}
	return configs, nil
}

// buildTencentDBSpecInfo converts one CdbSellConfig into DBSpecInfo.
// Cpu/Memory/VolumeMin/VolumeMax/Iops are all directly usable: Memory is Tencent's own
// DBSpec identity string (see resolveMemoryMBFromSpec — it is GiB*1000, not true
// MiB, so it is converted here), and VolumeMin/Max are documented decimal GB (see
// fetchCDBMetaOptions comment) so they are passed through unconverted.
func tencentBuildDBSpecInfo(region, cbEngine string, cfg *cdb.CdbSellConfig) *irs.DBSpecInfo {
	name := strconv.FormatInt(*cfg.Memory, 10)
	info := &irs.DBSpecInfo{
		Region:     region,
		DBEngine:   cbEngine,
		Name:       name,
		VCpu:       irs.VCpuInfo{Count: "-1", ClockGHz: "-1"},
		MemSizeMiB: "-1",
	}
	if cfg.Cpu != nil {
		info.VCpu.Count = strconv.FormatInt(*cfg.Cpu, 10)
	} else {
		info.MarkStatic("VCpu.Count", "DescribeCdbZoneConfig did not return Cpu for this sell config")
	}
	// Tencent CDB's Memory field is GiB*1000 (a CDB-specific unit, not true decimal MB nor
	// binary MiB — see resolveMemoryMBFromSpec's comment). Convert to real MiB: (Memory/1000)*1024.
	info.MemSizeMiB = strconv.FormatInt((*cfg.Memory/1000)*1024, 10)
	info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "RawMemoryValue(GiBx1000)", Value: name})
	if cfg.VolumeMin != nil && cfg.VolumeMax != nil {
		info.StorageSizeRangeGB = irs.StorageSizeRange{Min: *cfg.VolumeMin, Max: *cfg.VolumeMax}
	}
	if cfg.Iops != nil {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "Iops", Value: strconv.FormatInt(*cfg.Iops, 10)})
	}
	// DescribeCdbZoneConfig has no human-readable class-name field (unlike AWS/Alibaba's
	// DBInstanceClass); Name stays the raw Memory value on purpose because
	// resolveMemoryMBFromSpec (used by CreateRDBMS) accepts that exact numeric string directly
	// as the DBSpec parameter. DeviceType/EngineType are surfaced here instead so the
	// Misc column still conveys what AWS/Alibaba's class-name string would.
	if cfg.DeviceType != nil && *cfg.DeviceType != "" {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "DeviceType", Value: *cfg.DeviceType})
	}
	if cfg.EngineType != nil && *cfg.EngineType != "" {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "EngineType", Value: *cfg.EngineType})
	}
	if cfg.Info != nil && *cfg.Info != "" {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "Info", Value: *cfg.Info})
	}
	return info
}

func (handler *TencentRDBMSHandler) ListDBSpec(dbEngine string) ([]*irs.DBSpecInfo, error) {
	cblogger.Debug("Tencent CDB ListDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListDBSpec", "DescribeCdbZoneConfig()")
	start := call.Start()

	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return nil, err
	}
	if requestedEngine != "mysql" {
		err := fmt.Errorf("Tencent CDB RDBMSHandler only supports mysql; mariadb is not supported (CDB API rejects it)")
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	configs, err := handler.fetchTencentSellConfigsForZone()
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	sort.Slice(configs, func(i, j int) bool { return *configs[i].Memory < *configs[j].Memory })

	infoList := make([]*irs.DBSpecInfo, 0, len(configs))
	for _, cfg := range configs {
		infoList = append(infoList, tencentBuildDBSpecInfo(handler.Region.Region, requestedEngine, cfg))
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
	return infoList, nil
}

func (handler *TencentRDBMSHandler) GetDBSpec(dbEngine string, name string) (irs.DBSpecInfo, error) {
	cblogger.Debug("Tencent CDB GetDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetDBSpec", "DescribeCdbZoneConfig()")
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

func (handler *TencentRDBMSHandler) ListOrgDBSpec(dbEngine string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	if requestedEngine != "mysql" {
		return "", fmt.Errorf("Tencent CDB RDBMSHandler only supports mysql")
	}
	configs, err := handler.fetchTencentSellConfigsForZone()
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(configs)
	if err != nil {
		return "", fmt.Errorf("failed to marshal raw CdbSellConfig list: %w", err)
	}
	return string(b), nil
}

func (handler *TencentRDBMSHandler) GetOrgDBSpec(dbEngine string, name string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	if requestedEngine != "mysql" {
		return "", fmt.Errorf("Tencent CDB RDBMSHandler only supports mysql")
	}
	configs, err := handler.fetchTencentSellConfigsForZone()
	if err != nil {
		return "", err
	}
	for _, cfg := range configs {
		if strconv.FormatInt(*cfg.Memory, 10) == name {
			b, err := json.Marshal(cfg)
			if err != nil {
				return "", fmt.Errorf("failed to marshal raw CdbSellConfig: %w", err)
			}
			return string(b), nil
		}
	}
	return "", fmt.Errorf("DBSpec '%s' not found", name)
}
