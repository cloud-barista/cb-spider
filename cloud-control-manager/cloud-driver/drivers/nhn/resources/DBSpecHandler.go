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
	"fmt"
	"sort"
	"strconv"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// listRDSDBFlavorDetailsWithEndpoint calls the same /v3.0/db-flavors endpoint as
// listRDSDBFlavorsWithEndpoint, but returns the full nhnRDSDBFlavorInfo (including Vcpus/Ram)
// instead of collapsing it to a []string of names.
func (handler *NhnCloudRDBMSHandler) listRDSDBFlavorDetailsWithEndpoint(ctx context.Context, endpointFn func() (string, error)) ([]nhnRDSDBFlavorInfo, error) {
	var result nhnRDSFlavorListResponse
	if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-flavors", &result); err != nil {
		return nil, err
	}
	if err := checkRDSResponseHeader(result.Header); err != nil {
		return nil, err
	}
	return result.DBFlavors, nil
}

func nhnFormatVcpus(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// buildNhnDBSpecInfo converts one nhnRDSDBFlavorInfo into DBSpecInfo.
// Both Vcpus and Ram come from the same /v3.0/db-flavors response CB-Spider already calls
// for GetMetaInfo()'s DBSpecOptions; Ram is used directly as MiB (OpenStack Trove
// convention — NHN Cloud RDS is Trove-based; confirmed against published flavor names like
// "m2.c1m2" = 1 vCPU / 2GiB matching ram=2048).
func nhnBuildDBSpecInfo(region, cbEngine string, flavor nhnRDSDBFlavorInfo) *irs.DBSpecInfo {
	info := &irs.DBSpecInfo{
		Region:     region,
		DBEngine:   cbEngine,
		Name:       flavor.DBFlavorName,
		VCpu:       irs.VCpuInfo{Count: "-1", ClockGHz: "-1"},
		MemSizeMiB: "-1",
	}
	if flavor.Vcpus > 0 {
		info.VCpu.Count = nhnFormatVcpus(flavor.Vcpus)
	} else {
		info.MarkStatic("VCpu.Count", "/v3.0/db-flavors did not return a positive vcpus value for this flavor")
	}
	if flavor.Ram > 0 {
		info.MemSizeMiB = strconv.FormatInt(flavor.Ram, 10)
	} else {
		info.MarkStatic("MemSizeMiB", "/v3.0/db-flavors did not return a positive ram value for this flavor")
	}
	info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "DBFlavorId", Value: flavor.DBFlavorId})
	return info
}

func (handler *NhnCloudRDBMSHandler) dbSpecEndpointFn(requestedEngine string) func() (string, error) {
	if requestedEngine == "mariadb" {
		return handler.rdsMariaDBEndpoint
	}
	return handler.rdsEndpoint
}

func (handler *NhnCloudRDBMSHandler) ListDBSpec(dbEngine string) ([]*irs.DBSpecInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ListDBSpec()")
	callLogInfo := getCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "ListDBSpec", "GET /v3.0/db-flavors")
	start := call.Start()

	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		LoggingError(callLogInfo, err)
		return nil, err
	}
	if requestedEngine != "mysql" && requestedEngine != "mariadb" {
		err := fmt.Errorf("NHN Cloud RDS supports mysql and mariadb engines, requested: %s", requestedEngine)
		LoggingError(callLogInfo, err)
		return nil, err
	}
	if err := handler.checkRDSCredentials(); err != nil {
		LoggingError(callLogInfo, err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	flavors, err := handler.listRDSDBFlavorDetailsWithEndpoint(ctx, handler.dbSpecEndpointFn(requestedEngine))
	if err != nil {
		newErr := fmt.Errorf("failed to list DB flavors from NHN Cloud RDS API: %w", err)
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	sort.Slice(flavors, func(i, j int) bool { return flavors[i].DBFlavorName < flavors[j].DBFlavorName })

	infoList := make([]*irs.DBSpecInfo, 0, len(flavors))
	for _, f := range flavors {
		infoList = append(infoList, nhnBuildDBSpecInfo(handler.RegionInfo.Region, requestedEngine, f))
	}

	LoggingInfo(callLogInfo, start)
	return infoList, nil
}

func (handler *NhnCloudRDBMSHandler) GetDBSpec(dbEngine string, name string) (irs.DBSpecInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetDBSpec()")
	callLogInfo := getCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "GetDBSpec", "GET /v3.0/db-flavors")
	start := call.Start()

	infoList, err := handler.ListDBSpec(dbEngine)
	if err != nil {
		LoggingError(callLogInfo, err)
		return irs.DBSpecInfo{}, err
	}
	for _, info := range infoList {
		if info.Name == name {
			LoggingInfo(callLogInfo, start)
			return *info, nil
		}
	}
	err = fmt.Errorf("DBSpec '%s' not found for engine '%s'", name, dbEngine)
	LoggingError(callLogInfo, err)
	return irs.DBSpecInfo{}, err
}

func (handler *NhnCloudRDBMSHandler) ListOrgDBSpec(dbEngine string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	if requestedEngine != "mysql" && requestedEngine != "mariadb" {
		return "", fmt.Errorf("NHN Cloud RDS supports mysql and mariadb engines, requested: %s", requestedEngine)
	}
	if err := handler.checkRDSCredentials(); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	flavors, err := handler.listRDSDBFlavorDetailsWithEndpoint(ctx, handler.dbSpecEndpointFn(requestedEngine))
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(flavors)
	if err != nil {
		return "", fmt.Errorf("failed to marshal raw db-flavors list: %w", err)
	}
	return string(b), nil
}

func (handler *NhnCloudRDBMSHandler) GetOrgDBSpec(dbEngine string, name string) (string, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", err
	}
	if requestedEngine != "mysql" && requestedEngine != "mariadb" {
		return "", fmt.Errorf("NHN Cloud RDS supports mysql and mariadb engines, requested: %s", requestedEngine)
	}
	if err := handler.checkRDSCredentials(); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	flavors, err := handler.listRDSDBFlavorDetailsWithEndpoint(ctx, handler.dbSpecEndpointFn(requestedEngine))
	if err != nil {
		return "", err
	}
	for _, f := range flavors {
		if f.DBFlavorName == name {
			b, err := json.Marshal(f)
			if err != nil {
				return "", fmt.Errorf("failed to marshal raw db-flavor: %w", err)
			}
			return string(b), nil
		}
	}
	return "", fmt.Errorf("DBSpec '%s' not found", name)
}
