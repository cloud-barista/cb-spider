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

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vmysql"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// fetchNcpMysqlProducts finds the G3-generation MySQL image product (same selection logic
// as getMysqlMetaInfo/findMysqlG3ImageProductCode) and returns its full CloudDbProduct list
// (ProductCode, CpuCount, MemorySize, BaseBlockStorageSize), not just the ProductCode names
// that fetchMysqlProductSpecs extracts.
func (handler *NcpVpcRDBMSHandler) fetchNcpMysqlProducts() ([]*vmysql.CloudDbProduct, error) {
	imgResp, err := handler.MysqlClient.V2Api.GetCloudMysqlImageProductList(&vmysql.GetCloudMysqlImageProductListRequest{
		RegionCode: &handler.RegionInfo.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query NCP MySQL image product list: %w", err)
	}
	if len(imgResp.ProductList) == 0 {
		return nil, errors.New("NCP MySQL image product list is empty")
	}

	var g3ImageProductCode string
	for _, p := range imgResp.ProductList {
		if p.GenerationCode != nil && *p.GenerationCode == "G3" && p.ProductCode != nil && *p.ProductCode != "" {
			g3ImageProductCode = *p.ProductCode
			break
		}
	}
	if g3ImageProductCode == "" {
		return nil, errors.New("no G3 generation MySQL image product found in NCP")
	}

	prodResp, err := handler.MysqlClient.V2Api.GetCloudMysqlProductList(&vmysql.GetCloudMysqlProductListRequest{
		RegionCode:                 &handler.RegionInfo.Region,
		CloudMysqlImageProductCode: &g3ImageProductCode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query NCP MySQL product list: %w", err)
	}
	if len(prodResp.ProductList) == 0 {
		return nil, errors.New("NCP MySQL product list is empty")
	}
	return prodResp.ProductList, nil
}

// buildNcpDBSpecInfo converts one CloudDbProduct into DBSpecInfo.
// CpuCount is a plain core count (no unit issue). MemorySize/BaseBlockStorageSize are raw
// bytes (same field name/type as CloudMysqlServerInstance.MemorySize, which the existing
// GetRDBMS() code already treats as raw bytes — see KeyValueList "MemorySizeGB" there),
// so they are objectively converted with BytesToMiB/BytesToGB rather than passed through.
func ncpBuildDBSpecInfo(region string, p *vmysql.CloudDbProduct) *irs.DBSpecInfo {
	info := &irs.DBSpecInfo{
		Region:     region,
		DBEngine:   "mysql",
		Name:       derefStr(p.ProductCode),
		VCpu:       irs.VCpuInfo{Count: "-1", ClockGHz: "-1"},
		MemSizeMiB: "-1",
	}
	if p.CpuCount != nil {
		info.VCpu.Count = strconv.Itoa(int(*p.CpuCount))
	} else {
		info.MarkStatic("VCpu.Count", "GetCloudMysqlProductList did not return CpuCount for this product")
	}
	if p.MemorySize != nil {
		info.MemSizeMiB = strconv.FormatInt(irs.BytesToMiB(*p.MemorySize), 10)
	} else {
		info.MarkStatic("MemSizeMiB", "GetCloudMysqlProductList did not return MemorySize for this product")
	}
	// StorageSizeRangeGB is intentionally left unset: GetCloudMysqlProductList does not return
	// a min/max range for this product (BaseBlockStorageSize, when present, is a single
	// starting size that auto-scales in increments, not a range) — see
	// RDBMSMetaInfo.StorageSizeRangeGB's static 10-6000GB approximation instead.
	info.MarkStatic("StorageSizeRangeGB", "GetCloudMysqlProductList does not return a min/max storage range for this product; NCP G3 storage starts at a base size (see KeyValueList.BaseBlockStorageSizeGB when available) and auto-scales in increments rather than a fixed range. See RDBMSMetaInfo.StorageSizeRangeGB for the broader static approximation instead.")
	if p.BaseBlockStorageSize != nil && *p.BaseBlockStorageSize > 0 {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "BaseBlockStorageSizeGB", Value: strconv.FormatInt(irs.BytesToGB(*p.BaseBlockStorageSize), 10)})
	}
	if p.ProductType != nil && p.ProductType.CodeName != nil {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "ProductType", Value: *p.ProductType.CodeName})
	}
	return info
}

func (handler *NcpVpcRDBMSHandler) ListDBSpec(dbEngine string) ([]*irs.DBSpecInfo, error) {
	cblogger.Debug("NCP VPC RDBMSHandler ListDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "ListDBSpec", "GetCloudMysqlProductList()")
	start := call.Start()

	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return nil, err
	}
	if requestedEngine != "mysql" {
		err := fmt.Errorf("NCP DBSpecHandler only supports mysql (postgresql/mariadb not implemented)")
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	products, err := handler.fetchNcpMysqlProducts()
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	sort.Slice(products, func(i, j int) bool { return derefStr(products[i].ProductCode) < derefStr(products[j].ProductCode) })

	infoList := make([]*irs.DBSpecInfo, 0, len(products))
	for _, p := range products {
		infoList = append(infoList, ncpBuildDBSpecInfo(handler.RegionInfo.Region, p))
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
	return infoList, nil
}

func (handler *NcpVpcRDBMSHandler) GetDBSpec(dbEngine string, name string) (irs.DBSpecInfo, error) {
	cblogger.Debug("NCP VPC RDBMSHandler GetDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "GetDBSpec", "GetCloudMysqlProductList()")
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

func (handler *NcpVpcRDBMSHandler) ListOrgDBSpec(dbEngine string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	if requestedEngine != "mysql" {
		return "", fmt.Errorf("NCP DBSpecHandler only supports mysql")
	}
	products, err := handler.fetchNcpMysqlProducts()
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(products)
	if err != nil {
		return "", fmt.Errorf("failed to marshal raw CloudDbProduct list: %w", err)
	}
	return string(b), nil
}

func (handler *NcpVpcRDBMSHandler) GetOrgDBSpec(dbEngine string, name string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	if requestedEngine != "mysql" {
		return "", fmt.Errorf("NCP DBSpecHandler only supports mysql")
	}
	products, err := handler.fetchNcpMysqlProducts()
	if err != nil {
		return "", err
	}
	for _, p := range products {
		if derefStr(p.ProductCode) == name {
			b, err := json.Marshal(p)
			if err != nil {
				return "", fmt.Errorf("failed to marshal raw CloudDbProduct: %w", err)
			}
			return string(b), nil
		}
	}
	return "", fmt.Errorf("DBSpec '%s' not found", name)
}
