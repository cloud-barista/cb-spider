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
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// awsInstanceClassDetail aggregates OrderableDBInstanceOption fields for one DBInstanceClass
// across all (EngineVersion, AZ, StorageType) combinations returned for the engine.
type awsInstanceClassDetail struct {
	minStorageGiB int64
	maxStorageGiB int64
}

// awsRDSClassToEC2Type converts an RDS instance class (e.g. "db.t3.medium") to the EC2
// instance type it is named after (e.g. "t3.medium"). RDS instance classes are documented
// by AWS to mirror EC2 instance types 1:1 in vCPU/Memory.
func awsRDSClassToEC2Type(dbInstanceClass string) string {
	return strings.TrimPrefix(dbInstanceClass, "db.")
}

// fetchAwsInstanceClassDetails calls DescribeOrderableDBInstanceOptions for the given engine
// (across all versions, since instance-class availability is queried here, not per-version
// meta capability) and aggregates min/max storage (GiB, AWS's native unit) per DBInstanceClass.
func (handler *AwsRDBMSHandler) fetchAwsInstanceClassDetails(awsEngineName string) (map[string]*awsInstanceClassDetail, error) {
	// mariadb alone has been observed to exceed a 45s budget here (many more
	// (EngineVersion, AZ, StorageType) combinations to page through than mysql/postgres),
	// causing "context deadline exceeded" before pagination finished; 120s gives real calls
	// enough headroom while still failing fast on a genuinely broken/unreachable API.
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	details := map[string]*awsInstanceClassDetail{}
	input := &rds.DescribeOrderableDBInstanceOptionsInput{
		Engine:     aws.String(awsEngineName),
		MaxRecords: aws.Int64(100),
	}
	err := handler.Client.DescribeOrderableDBInstanceOptionsPagesWithContext(ctx, input, func(page *rds.DescribeOrderableDBInstanceOptionsOutput, lastPage bool) bool {
		for _, option := range page.OrderableDBInstanceOptions {
			if option.DBInstanceClass == nil || *option.DBInstanceClass == "" {
				continue
			}
			d, ok := details[*option.DBInstanceClass]
			if !ok {
				d = &awsInstanceClassDetail{}
				details[*option.DBInstanceClass] = d
			}
			if option.MinStorageSize != nil && *option.MinStorageSize > 0 && (d.minStorageGiB == 0 || *option.MinStorageSize < d.minStorageGiB) {
				d.minStorageGiB = *option.MinStorageSize
			}
			if option.MaxStorageSize != nil && *option.MaxStorageSize > d.maxStorageGiB {
				d.maxStorageGiB = *option.MaxStorageSize
			}
		}
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeOrderableDBInstanceOptions failed for engine %s: %w", awsEngineName, err)
	}
	if len(details) == 0 {
		return nil, fmt.Errorf("DescribeOrderableDBInstanceOptions returned no instance classes for engine %s", awsEngineName)
	}
	return details, nil
}

// fetchAwsEC2InstanceTypeSpecs batch-queries EC2 DescribeInstanceTypes for the given EC2
// instance type names and returns vCPU/Memory per type. EC2 DescribeInstanceTypes fails the
// ENTIRE call if any requested type name is unknown, so on error this returns the error to
// the caller, which falls back to marking VCpu/MemSizeMiB as unavailable rather than failing
// the whole spec listing.
func (handler *AwsRDBMSHandler) fetchAwsEC2InstanceTypeSpecs(ec2TypeNames []string) (map[string]*ec2.InstanceTypeInfo, error) {
	if handler.EC2Client == nil {
		return nil, fmt.Errorf("EC2 client is not available")
	}
	result := map[string]*ec2.InstanceTypeInfo{}
	const batchSize = 100
	for i := 0; i < len(ec2TypeNames); i += batchSize {
		end := i + batchSize
		if end > len(ec2TypeNames) {
			end = len(ec2TypeNames)
		}
		batch := ec2TypeNames[i:end]
		types := make([]*string, 0, len(batch))
		for _, t := range batch {
			types = append(types, aws.String(t))
		}
		out, err := handler.EC2Client.DescribeInstanceTypes(&ec2.DescribeInstanceTypesInput{InstanceTypes: types})
		if err != nil {
			return nil, fmt.Errorf("EC2 DescribeInstanceTypes failed: %w", err)
		}
		for _, info := range out.InstanceTypes {
			if info.InstanceType != nil {
				result[*info.InstanceType] = info
			}
		}
	}
	return result, nil
}

// buildAwsDBSpecInfo assembles a DBSpecInfo for one DBInstanceClass, using
// EC2 type specs when available (ec2Specs may be nil if the EC2 cross-query failed).
func (handler *AwsRDBMSHandler) buildAwsDBSpecInfo(cbEngine, className string, detail *awsInstanceClassDetail, ec2Specs map[string]*ec2.InstanceTypeInfo) *irs.DBSpecInfo {
	info := &irs.DBSpecInfo{
		Region:     handler.Region.Region,
		DBEngine:   cbEngine,
		Name:       className,
		VCpu:       irs.VCpuInfo{Count: "-1", ClockGHz: "-1"},
		MemSizeMiB: "-1",
	}
	if detail != nil {
		info.StorageSizeRangeGB = irs.StorageSizeRange{
			Min: irs.GiBToGB(detail.minStorageGiB),
			Max: irs.GiBToGB(detail.maxStorageGiB),
		}
	}

	ec2Type := awsRDSClassToEC2Type(className)
	if ec2Specs != nil {
		if spec, ok := ec2Specs[ec2Type]; ok && spec != nil {
			if spec.VCpuInfo != nil && spec.VCpuInfo.DefaultVCpus != nil {
				info.VCpu.Count = strconv.FormatInt(*spec.VCpuInfo.DefaultVCpus, 10)
			}
			if spec.ProcessorInfo != nil && spec.ProcessorInfo.SustainedClockSpeedInGhz != nil {
				info.VCpu.ClockGHz = strconv.FormatFloat(*spec.ProcessorInfo.SustainedClockSpeedInGhz, 'f', 1, 64)
			}
			if spec.MemoryInfo != nil && spec.MemoryInfo.SizeInMiB != nil {
				info.MemSizeMiB = strconv.FormatInt(*spec.MemoryInfo.SizeInMiB, 10)
			}
			info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "EC2InstanceType", Value: ec2Type})
			return info
		}
	}
	info.MarkStatic("VCpu.Count", fmt.Sprintf("EC2 DescribeInstanceTypes lookup for '%s' (derived from RDS class '%s') did not return a result; AWS RDS API itself does not expose vCPU.", ec2Type, className))
	info.MarkStatic("MemSizeMiB", fmt.Sprintf("EC2 DescribeInstanceTypes lookup for '%s' (derived from RDS class '%s') did not return a result; AWS RDS API itself does not expose Memory.", ec2Type, className))
	return info
}

func (handler *AwsRDBMSHandler) listDBSpecDetails(dbEngine string) (string, map[string]*awsInstanceClassDetail, map[string]*ec2.InstanceTypeInfo, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return "", nil, nil, err
	}
	allEngineNames := []awsRDBMSEngine{
		{"mysql", "mysql"},
		{"mariadb", "mariadb"},
		{"postgresql", "postgres"},
	}
	var awsEngineName string
	for _, eng := range allEngineNames {
		if eng.cbName == requestedEngine {
			awsEngineName = eng.awsName
			break
		}
	}
	if awsEngineName == "" {
		return "", nil, nil, fmt.Errorf("DBEngine '%s' is not supported by AWS RDS", requestedEngine)
	}

	details, err := handler.fetchAwsInstanceClassDetails(awsEngineName)
	if err != nil {
		return "", nil, nil, err
	}

	ec2TypeNames := make([]string, 0, len(details))
	for className := range details {
		ec2TypeNames = append(ec2TypeNames, awsRDSClassToEC2Type(className))
	}
	ec2Specs, ec2Err := handler.fetchAwsEC2InstanceTypeSpecs(ec2TypeNames)
	if ec2Err != nil {
		cblogger.Warnf("AWS DBSpec: EC2 DescribeInstanceTypes cross-query failed, VCpu/MemSizeMiB will be marked unavailable: %v", ec2Err)
		ec2Specs = nil
	}
	return requestedEngine, details, ec2Specs, nil
}

func (handler *AwsRDBMSHandler) ListDBSpec(dbEngine string) ([]*irs.DBSpecInfo, error) {
	cblogger.Debug("AWS RDS ListDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListDBSpec", "DescribeOrderableDBInstanceOptions()+EC2.DescribeInstanceTypes()")
	start := call.Start()

	requestedEngine, details, ec2Specs, err := handler.listDBSpecDetails(dbEngine)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	classNames := make([]string, 0, len(details))
	for className := range details {
		classNames = append(classNames, className)
	}
	sort.Strings(classNames)

	infoList := make([]*irs.DBSpecInfo, 0, len(classNames))
	for _, className := range classNames {
		infoList = append(infoList, handler.buildAwsDBSpecInfo(requestedEngine, className, details[className], ec2Specs))
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
	return infoList, nil
}

func (handler *AwsRDBMSHandler) GetDBSpec(dbEngine string, name string) (irs.DBSpecInfo, error) {
	cblogger.Debug("AWS RDS GetDBSpec() called")
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetDBSpec", "DescribeOrderableDBInstanceOptions()+EC2.DescribeInstanceTypes()")
	start := call.Start()

	requestedEngine, details, ec2Specs, err := handler.listDBSpecDetails(dbEngine)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return irs.DBSpecInfo{}, err
	}
	detail, ok := details[name]
	if !ok {
		err := fmt.Errorf("DBSpec '%s' not found for engine '%s'", name, requestedEngine)
		LoggingError(hiscallInfo, err)
		return irs.DBSpecInfo{}, err
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
	return *handler.buildAwsDBSpecInfo(requestedEngine, name, detail, ec2Specs), nil
}

func (handler *AwsRDBMSHandler) fetchAwsOrderableOptionsRaw(dbEngine string, dbInstanceClass string) ([]*rds.OrderableDBInstanceOption, error) {
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return nil, err
	}
	allEngineNames := []awsRDBMSEngine{
		{"mysql", "mysql"},
		{"mariadb", "mariadb"},
		{"postgresql", "postgres"},
	}
	var awsEngineName string
	for _, eng := range allEngineNames {
		if eng.cbName == requestedEngine {
			awsEngineName = eng.awsName
			break
		}
	}
	if awsEngineName == "" {
		return nil, fmt.Errorf("DBEngine '%s' is not supported by AWS RDS", requestedEngine)
	}

	// Same pagination workload as fetchAwsInstanceClassDetails above; see its comment for
	// why 120s (not the previous 45s, which mariadb was observed to exceed).
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	input := &rds.DescribeOrderableDBInstanceOptionsInput{
		Engine:     aws.String(awsEngineName),
		MaxRecords: aws.Int64(100),
	}
	if dbInstanceClass != "" {
		input.DBInstanceClass = aws.String(dbInstanceClass)
	}
	var options []*rds.OrderableDBInstanceOption
	err = handler.Client.DescribeOrderableDBInstanceOptionsPagesWithContext(ctx, input, func(page *rds.DescribeOrderableDBInstanceOptionsOutput, lastPage bool) bool {
		options = append(options, page.OrderableDBInstanceOptions...)
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeOrderableDBInstanceOptions failed: %w", err)
	}
	if dbInstanceClass != "" && len(options) == 0 {
		return nil, fmt.Errorf("DBSpec '%s' not found for engine '%s'", dbInstanceClass, requestedEngine)
	}
	return options, nil
}

func (handler *AwsRDBMSHandler) ListOrgDBSpec(dbEngine string) (string, error) {
	options, err := handler.fetchAwsOrderableOptionsRaw(dbEngine, "")
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(options)
	if err != nil {
		return "", fmt.Errorf("failed to marshal raw OrderableDBInstanceOption list: %w", err)
	}
	return string(b), nil
}

func (handler *AwsRDBMSHandler) GetOrgDBSpec(dbEngine string, name string) (string, error) {
	options, err := handler.fetchAwsOrderableOptionsRaw(dbEngine, name)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(options)
	if err != nil {
		return "", fmt.Errorf("failed to marshal raw OrderableDBInstanceOption: %w", err)
	}
	return string(b), nil
}
