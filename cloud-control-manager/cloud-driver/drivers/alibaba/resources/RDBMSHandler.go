// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, April 2026.

package resources

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaRDBMSHandler struct {
	Region idrv.RegionInfo
	Client *rds.Client
}

type alibabaRDBMSEngine struct {
	cbName  string
	aliName string
}

type alibabaRDBMSStorageLookup struct {
	zoneID        string
	engineVersion string
	category      string
	storageType   string
}

func (handler *AlibabaRDBMSHandler) GetMetaInfo(dbEngine string) (irs.RDBMSMetaInfo, error) {
	cblogger.Debug("Alibaba RDS GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return irs.RDBMSMetaInfo{}, err
	}
	supportedEngines, instanceSpecOptions, storageTypeOptions, storageSizeRange, err := handler.fetchRDBMSMetaOptions(requestedEngine)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}

	metaInfo, err := irs.BuildRDBMSMetaInfo(requestedEngine, supportedEngines, instanceSpecOptions, storageTypeOptions, storageSizeRange, true, true, true, true, true, "7-730", true, false, true, true)
	if err != nil {
		return irs.RDBMSMetaInfo{}, err
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return metaInfo, nil
}

func (handler *AlibabaRDBMSHandler) fetchRDBMSMetaOptions(dbEngine string) (map[string][]string, map[string][]string, map[string][]string, irs.StorageSizeRange, error) {
	allEngineNames := []alibabaRDBMSEngine{
		{cbName: "mysql", aliName: "MySQL"},
		{cbName: "mariadb", aliName: "MariaDB"},
		{cbName: "postgresql", aliName: "PostgreSQL"},
	}
	engineNames := make([]alibabaRDBMSEngine, 0, 1)
	for _, eng := range allEngineNames {
		if eng.cbName == dbEngine {
			engineNames = append(engineNames, eng)
			break
		}
	}
	if len(engineNames) == 0 {
		return nil, nil, nil, irs.StorageSizeRange{}, fmt.Errorf("DBEngine '%s' is not supported by Alibaba RDS", dbEngine)
	}
	engineByAlibabaName := map[string]alibabaRDBMSEngine{}
	for _, eng := range engineNames {
		engineByAlibabaName[strings.ToLower(eng.aliName)] = eng
	}

	versionSets := map[string]map[string]bool{}
	storageTypeSets := map[string]map[string]bool{}
	storageLookups := map[string][]alibabaRDBMSStorageLookup{}
	for _, eng := range engineNames {
		zonesReq := rds.CreateDescribeAvailableZonesRequest()
		zonesReq.Engine = eng.aliName
		zonesReq.InstanceChargeType = "Postpaid"
		if handler.Region.Zone != "" {
			zonesReq.ZoneId = handler.Region.Zone
		}
		zonesResp, err := handler.Client.DescribeAvailableZones(zonesReq)
		if err != nil {
			return nil, nil, nil, irs.StorageSizeRange{}, fmt.Errorf("DescribeAvailableZones failed for engine %s: %w", eng.aliName, err)
		}
		selectedZoneID := handler.Region.Zone
		if selectedZoneID == "" && len(zonesResp.AvailableZones) > 0 {
			selectedZoneID = zonesResp.AvailableZones[0].ZoneId
		}
		if selectedZoneID == "" {
			return nil, nil, nil, irs.StorageSizeRange{}, fmt.Errorf("DescribeAvailableZones returned no zones for %s", eng.aliName)
		}

		for _, zone := range zonesResp.AvailableZones {
			if zone.ZoneId != selectedZoneID {
				continue
			}
			for _, supportedEngine := range zone.SupportedEngines {
				if _, ok := engineByAlibabaName[strings.ToLower(supportedEngine.Engine)]; !ok || !strings.EqualFold(supportedEngine.Engine, eng.aliName) {
					continue
				}
				if versionSets[eng.cbName] == nil {
					versionSets[eng.cbName] = map[string]bool{}
				}
				if storageTypeSets[eng.cbName] == nil {
					storageTypeSets[eng.cbName] = map[string]bool{}
				}
				for _, engineVersion := range supportedEngine.SupportedEngineVersions {
					if engineVersion.Version == "" {
						continue
					}
					versionSets[eng.cbName][engineVersion.Version] = true
					for _, category := range engineVersion.SupportedCategorys {
						for _, storageType := range category.SupportedStorageTypes {
							if storageType.StorageType != "" {
								storageTypeSets[eng.cbName][storageType.StorageType] = true
								storageLookups[eng.aliName] = append(storageLookups[eng.aliName], alibabaRDBMSStorageLookup{
									zoneID:        zone.ZoneId,
									engineVersion: engineVersion.Version,
									category:      category.Category,
									storageType:   storageType.StorageType,
								})
							}
						}
					}
				}
			}
		}
	}

	supportedEngines := map[string][]string{}
	storageTypeOptions := map[string][]string{}
	for _, eng := range engineNames {
		versions := alibabaSortedSet(versionSets[eng.cbName])
		if len(versions) == 0 {
			return nil, nil, nil, irs.StorageSizeRange{}, fmt.Errorf("DescribeAvailableZones returned no engine versions for %s", eng.aliName)
		}
		supportedEngines[eng.cbName] = versions

		storageTypes := alibabaSortedSet(storageTypeSets[eng.cbName])
		if len(storageTypes) == 0 {
			return nil, nil, nil, irs.StorageSizeRange{}, fmt.Errorf("DescribeAvailableZones returned no storage types for %s", eng.aliName)
		}
		storageTypeOptions[eng.cbName] = storageTypes
	}

	instanceSpecOptions, storageSizeRange, err := handler.fetchRDBMSInstanceOptions(engineNames, storageLookups)
	if err != nil {
		return nil, nil, nil, irs.StorageSizeRange{}, err
	}

	return supportedEngines, instanceSpecOptions, storageTypeOptions, storageSizeRange, nil
}

func (handler *AlibabaRDBMSHandler) fetchRDBMSInstanceOptions(engineNames []alibabaRDBMSEngine, storageLookups map[string][]alibabaRDBMSStorageLookup) (map[string][]string, irs.StorageSizeRange, error) {
	instanceSpecOptions := map[string][]string{}
	var minStorage int64
	var maxStorage int64
	latestVersionByEngine := map[string]string{}
	for _, eng := range engineNames {
		for _, lookup := range storageLookups[eng.aliName] {
			if latestVersionByEngine[eng.aliName] == "" || alibabaCompareVersionStrings(lookup.engineVersion, latestVersionByEngine[eng.aliName]) > 0 {
				latestVersionByEngine[eng.aliName] = lookup.engineVersion
			}
		}
	}

	for _, eng := range engineNames {
		instanceSpecSet := map[string]bool{}
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
				return nil, irs.StorageSizeRange{}, fmt.Errorf("DescribeAvailableClasses failed for engine %s version %s category %s storage type %s zone %s: %w", eng.aliName, lookup.engineVersion, lookup.category, lookup.storageType, lookup.zoneID, err)
			}

			for _, class := range classesResp.DBInstanceClasses {
				if class.DBInstanceClass != "" {
					instanceSpecSet[class.DBInstanceClass] = true
				}
				classMin, classMax := alibabaStorageRangeValues(class)
				if classMin > 0 && (minStorage == 0 || classMin < minStorage) {
					minStorage = classMin
				}
				if classMax > maxStorage {
					maxStorage = classMax
				}
			}
		}

		instanceSpecs := alibabaSortedSet(instanceSpecSet)
		if len(instanceSpecs) > 0 {
			instanceSpecOptions[eng.cbName] = instanceSpecs
		}
	}

	if minStorage == 0 || maxStorage == 0 {
		return nil, irs.StorageSizeRange{}, errors.New("DescribeAvailableClasses returned no storage size range")
	}

	return instanceSpecOptions, irs.StorageSizeRange{Min: minStorage, Max: maxStorage}, nil
}

func (handler *AlibabaRDBMSHandler) describeAvailableClasses(request *rds.DescribeAvailableClassesRequest) (*rds.DescribeAvailableClassesResponse, error) {
	type describeAvailableClassesResult struct {
		response *rds.DescribeAvailableClassesResponse
		err      error
	}
	resultChan := make(chan describeAvailableClassesResult, 1)
	go func() {
		response, err := handler.Client.DescribeAvailableClasses(request)
		resultChan <- describeAvailableClassesResult{response: response, err: err}
	}()

	timer := time.NewTimer(20 * time.Second)
	defer timer.Stop()

	select {
	case result := <-resultChan:
		if result.err != nil {
			return nil, result.err
		}
		if result.response == nil {
			return nil, errors.New("DescribeAvailableClasses returned no response")
		}
		return result.response, nil
	case <-timer.C:
		return nil, errors.New("DescribeAvailableClasses timed out")
	}
}

func alibabaStorageRangeValues(class rds.DBInstanceClass) (int64, int64) {
	if class.DBInstanceStorageRange.MinValue > 0 || class.DBInstanceStorageRange.MaxValue > 0 {
		return int64(class.DBInstanceStorageRange.MinValue), int64(class.DBInstanceStorageRange.MaxValue)
	}

	replacer := strings.NewReplacer("~", "-", ",", "-", " ", "")
	parts := strings.Split(replacer.Replace(class.StorageRange), "-")
	if len(parts) < 2 {
		return 0, 0
	}
	minValue, minErr := strconv.ParseInt(parts[0], 10, 64)
	maxValue, maxErr := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if minErr != nil || maxErr != nil {
		return 0, 0
	}
	return minValue, maxValue
}

func alibabaSortedSet(set map[string]bool) []string {
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		return alibabaCompareVersionStrings(values[i], values[j]) < 0
	})
	return values
}

func alibabaCompareVersionStrings(leftVersion, rightVersion string) int {
	leftParts := strings.Split(leftVersion, ".")
	rightParts := strings.Split(rightVersion, ".")
	maxLen := len(leftParts)
	if len(rightParts) > maxLen {
		maxLen = len(rightParts)
	}

	for index := 0; index < maxLen; index++ {
		leftValue, rightValue := 0, 0
		if index < len(leftParts) {
			leftValue, _ = strconv.Atoi(leftParts[index])
		}
		if index < len(rightParts) {
			rightValue, _ = strconv.Atoi(rightParts[index])
		}
		if leftValue < rightValue {
			return -1
		}
		if leftValue > rightValue {
			return 1
		}
	}
	return strings.Compare(leftVersion, rightVersion)
}

func (handler *AlibabaRDBMSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListIID", "DescribeDBInstances()")
	start := call.Start()

	request := rds.CreateDescribeDBInstancesRequest()
	request.RegionId = handler.Region.Region

	var iidList []*irs.IID
	pageNumber := 1
	for {
		request.PageNumber = requests.NewInteger(pageNumber)
		request.PageSize = requests.NewInteger(100)

		response, err := handler.Client.DescribeDBInstances(request)
		if err != nil {
			hiscallInfo.ElapsedTime = call.Elapsed(start)
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}

		for _, db := range response.Items.DBInstance {
			iid := &irs.IID{
				NameId:   db.DBInstanceDescription,
				SystemId: db.DBInstanceId,
			}
			if iid.NameId == "" {
				iid.NameId = db.DBInstanceId
			}
			iidList = append(iidList, iid)
		}

		if len(response.Items.DBInstance) < 100 {
			break
		}
		pageNumber++
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return iidList, nil
}

func (handler *AlibabaRDBMSHandler) CreateRDBMS(rdbmsReqInfo irs.RDBMSInfo) (irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsReqInfo.IId.NameId, "CreateDBInstance()")
	start := call.Start()

	// Validate required fields
	if rdbmsReqInfo.IId.NameId == "" {
		return irs.RDBMSInfo{}, errors.New("RDBMS NameId is required")
	}
	if rdbmsReqInfo.DBEngine == "" {
		return irs.RDBMSInfo{}, errors.New("DBEngine is required")
	}
	if rdbmsReqInfo.DBEngineVersion == "" {
		return irs.RDBMSInfo{}, errors.New("DBEngineVersion is required")
	}
	if rdbmsReqInfo.DBInstanceSpec == "" {
		return irs.RDBMSInfo{}, errors.New("DBInstanceSpec is required")
	}
	if rdbmsReqInfo.MasterUserName == "" {
		return irs.RDBMSInfo{}, errors.New("MasterUserName is required")
	}
	if rdbmsReqInfo.MasterUserPassword == "" {
		return irs.RDBMSInfo{}, errors.New("MasterUserPassword is required")
	}
	if rdbmsReqInfo.StorageSize == "" {
		return irs.RDBMSInfo{}, errors.New("StorageSize is required")
	}
	// Alibaba RDS requires VPC and Subnet (VSwitch) for instance creation
	if rdbmsReqInfo.VpcIID.SystemId == "" {
		return irs.RDBMSInfo{}, errors.New("VPC (VpcIID.SystemId) is required for Alibaba RDS")
	}
	if len(rdbmsReqInfo.SubnetIIDs) == 0 || rdbmsReqInfo.SubnetIIDs[0].SystemId == "" {
		return irs.RDBMSInfo{}, errors.New("Subnet (SubnetIIDs[0].SystemId / VSwitch) is required for Alibaba RDS")
	}

	request := rds.CreateCreateDBInstanceRequest()
	request.RegionId = handler.Region.Region
	request.Engine = strings.ToUpper(rdbmsReqInfo.DBEngine[:1]) + strings.ToLower(rdbmsReqInfo.DBEngine[1:])
	// Alibaba expects: MySQL, MariaDB, PostgreSQL
	switch strings.ToLower(rdbmsReqInfo.DBEngine) {
	case "mysql":
		request.Engine = "MySQL"
	case "mariadb":
		request.Engine = "MariaDB"
	case "postgresql":
		request.Engine = "PostgreSQL"
	default:
		return irs.RDBMSInfo{}, fmt.Errorf("unsupported engine: %s", rdbmsReqInfo.DBEngine)
	}

	request.EngineVersion = rdbmsReqInfo.DBEngineVersion
	request.DBInstanceClass = rdbmsReqInfo.DBInstanceSpec
	request.DBInstanceStorage = requests.Integer(rdbmsReqInfo.StorageSize)
	request.DBInstanceDescription = rdbmsReqInfo.IId.NameId
	request.SecurityIPList = "0.0.0.0/0" // Default, user can modify later
	request.PayType = "Postpaid"
	request.InstanceNetworkType = "VPC"
	request.DBInstanceNetType = "Intranet"

	// Storage type
	if rdbmsReqInfo.StorageType != "" {
		request.DBInstanceStorageType = rdbmsReqInfo.StorageType
	} else {
		request.DBInstanceStorageType = "cloud_essd"
	}

	// VPC and Subnet (VSwitch)
	if rdbmsReqInfo.VpcIID.SystemId != "" {
		request.VPCId = rdbmsReqInfo.VpcIID.SystemId
	}
	if len(rdbmsReqInfo.SubnetIIDs) > 0 && rdbmsReqInfo.SubnetIIDs[0].SystemId != "" {
		request.VSwitchId = rdbmsReqInfo.SubnetIIDs[0].SystemId
	}

	// Zone
	if handler.Region.Zone != "" {
		request.ZoneId = handler.Region.Zone
	}

	// HA (Category)
	if rdbmsReqInfo.HighAvailability {
		request.Category = "HighAvailability"
	} else {
		request.Category = "Basic"
	}

	// Backup Configuration
	// Note: Alibaba RDS CreateDBInstance API does not support backup settings.
	// BackupRetentionDays and BackupTime must be configured via ModifyBackupPolicy API after instance creation.
	// TODO: Implement post-creation backup configuration using ModifyBackupPolicy API
	// if rdbmsReqInfo.BackupRetentionDays > 0 || rdbmsReqInfo.BackupTime != "" {
	//     cblogger.Infof("[Alibaba RDBMS] Backup settings (RetentionDays=%d, Time=%s) will use CSP defaults. "+
	//         "Post-creation configuration via ModifyBackupPolicy is not yet implemented.",
	//         rdbmsReqInfo.BackupRetentionDays, rdbmsReqInfo.BackupTime)
	// }

	// Tags
	if len(rdbmsReqInfo.TagList) > 0 {
		tags := make([]rds.CreateDBInstanceTag, 0, len(rdbmsReqInfo.TagList))
		for _, tag := range rdbmsReqInfo.TagList {
			tags = append(tags, rds.CreateDBInstanceTag{
				Key:   tag.Key,
				Value: tag.Value,
			})
		}
		request.Tag = &tags
	}

	response, err := handler.Client.CreateDBInstance(request)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	// Wait for the instance to be running
	dbInstanceId := response.DBInstanceId
	err = handler.waitForDBInstanceStatus(dbInstanceId, "Running", 600)
	if err != nil {
		cblogger.Warn("Instance created but wait for Running state failed: ", err)
	}

	// Allocate public endpoint if PublicAccess is requested
	if rdbmsReqInfo.PublicAccess {
		allocReq := rds.CreateAllocateInstancePublicConnectionRequest()
		allocReq.DBInstanceId = dbInstanceId
		allocReq.ConnectionStringPrefix = dbInstanceId + "-pub"
		allocReq.Port = "3306"
		if strings.ToLower(rdbmsReqInfo.DBEngine) == "postgresql" {
			allocReq.Port = "5432"
		}
		_, allocErr := handler.Client.AllocateInstancePublicConnection(allocReq)
		if allocErr != nil {
			cblogger.Errorf("Failed to allocate public connection for [%s]: %v", dbInstanceId, allocErr)
			return irs.RDBMSInfo{}, fmt.Errorf("failed to allocate public connection: %w", allocErr)
		}
		cblogger.Infof("Public connection allocated for [%s]", dbInstanceId)

		// Wait for Running state again after public connection allocation
		err = handler.waitForDBInstanceStatus(dbInstanceId, "Running", 120)
		if err != nil {
			cblogger.Warnf("Wait for Running after public connection allocation: %v", err)
		}
	}

	// Create master user account with retry for IncorrectDBInstanceState.
	// The instance may still be initializing after public connection allocation.
	if rdbmsReqInfo.MasterUserName != "" {
		const acctRetryInterval = 15
		const acctRetryTimeout = 180
		acctAttempts := 0
		for {
			acctReq := rds.CreateCreateAccountRequest()
			acctReq.DBInstanceId = dbInstanceId
			acctReq.AccountName = rdbmsReqInfo.MasterUserName
			acctReq.AccountPassword = rdbmsReqInfo.MasterUserPassword
			acctReq.AccountType = "Super"
			_, acctErr := handler.Client.CreateAccount(acctReq)
			if acctErr == nil {
				cblogger.Infof("Master account [%s] created for [%s]", rdbmsReqInfo.MasterUserName, dbInstanceId)
				break
			}
			if strings.Contains(acctErr.Error(), "IncorrectDBInstanceState") {
				elapsed := acctAttempts * acctRetryInterval
				if elapsed >= acctRetryTimeout {
					return irs.RDBMSInfo{}, fmt.Errorf("failed to create master account after %ds: %w", acctRetryTimeout, acctErr)
				}
				cblogger.Warnf("[Alibaba] IncorrectDBInstanceState on account create attempt %d – retrying in %ds: %v",
					acctAttempts+1, acctRetryInterval, acctErr)
				time.Sleep(time.Duration(acctRetryInterval) * time.Second)
				acctAttempts++
				continue
			}
			return irs.RDBMSInfo{}, fmt.Errorf("failed to create master account: %w", acctErr)
		}
	}

	// Set deletion protection if requested
	if rdbmsReqInfo.DeletionProtection {
		modReq := rds.CreateModifyDBInstanceDeletionProtectionRequest()
		modReq.DBInstanceId = dbInstanceId
		modReq.DeletionProtection = "true"
		_, modErr := handler.Client.ModifyDBInstanceDeletionProtection(modReq)
		if modErr != nil {
			cblogger.Warn("Failed to set deletion protection: ", modErr)
		}
	}

	// Get the created instance info
	return handler.GetRDBMS(irs.IID{SystemId: dbInstanceId})
}

func (handler *AlibabaRDBMSHandler) ListRDBMS() ([]*irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListRDBMS", "DescribeDBInstances()")
	start := call.Start()

	request := rds.CreateDescribeDBInstancesRequest()
	request.RegionId = handler.Region.Region

	var rdbmsList []*irs.RDBMSInfo
	pageNumber := 1
	for {
		request.PageNumber = requests.NewInteger(pageNumber)
		request.PageSize = requests.NewInteger(100)

		response, err := handler.Client.DescribeDBInstances(request)
		if err != nil {
			hiscallInfo.ElapsedTime = call.Elapsed(start)
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}

		for _, db := range response.Items.DBInstance {
			// Get detailed attribute for each instance
			attrInfo, err := handler.getDBInstanceAttribute(db.DBInstanceId)
			if err != nil {
				cblogger.Warn("Failed to get attribute for ", db.DBInstanceId, ": ", err)
				// Use basic info from list
				rdbmsInfo := handler.convertListItemToRDBMSInfo(&db)
				rdbmsList = append(rdbmsList, &rdbmsInfo)
				continue
			}
			rdbmsList = append(rdbmsList, &attrInfo)
		}

		if len(response.Items.DBInstance) < 100 {
			break
		}
		pageNumber++
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return rdbmsList, nil
}

func (handler *AlibabaRDBMSHandler) GetRDBMS(rdbmsIID irs.IID) (irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsIID.NameId, "DescribeDBInstanceAttribute()")
	start := call.Start()

	rdbmsInfo, err := handler.getDBInstanceAttribute(rdbmsIID.SystemId)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	return rdbmsInfo, nil
}

func (handler *AlibabaRDBMSHandler) DeleteRDBMS(rdbmsIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsIID.NameId, "DeleteDBInstance()")
	start := call.Start()

	// Check and disable deletion protection first
	attrInfo, err := handler.getDBInstanceAttribute(rdbmsIID.SystemId)
	if err == nil && attrInfo.DeletionProtection {
		modReq := rds.CreateModifyDBInstanceDeletionProtectionRequest()
		modReq.DBInstanceId = rdbmsIID.SystemId
		modReq.DeletionProtection = "false"
		_, modErr := handler.Client.ModifyDBInstanceDeletionProtection(modReq)
		if modErr != nil {
			cblogger.Warn("Failed to disable deletion protection: ", modErr)
		}
	}

	request := rds.CreateDeleteDBInstanceRequest()
	request.DBInstanceId = rdbmsIID.SystemId

	_, err = handler.Client.DeleteDBInstance(request)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	calllogger.Info(call.String(hiscallInfo))

	return true, nil
}

// ===== Helper Functions =====

func (handler *AlibabaRDBMSHandler) getDBInstanceAttribute(dbInstanceId string) (irs.RDBMSInfo, error) {
	request := rds.CreateDescribeDBInstanceAttributeRequest()
	request.DBInstanceId = dbInstanceId

	response, err := handler.Client.DescribeDBInstanceAttribute(request)
	if err != nil {
		return irs.RDBMSInfo{}, err
	}

	if len(response.Items.DBInstanceAttribute) == 0 {
		return irs.RDBMSInfo{}, fmt.Errorf("DB instance not found: %s", dbInstanceId)
	}

	attr := response.Items.DBInstanceAttribute[0]
	rdbmsInfo := handler.convertAttributeToRDBMSInfo(&attr)

	// Retrieve public endpoint via DescribeDBInstanceNetInfo
	netReq := rds.CreateDescribeDBInstanceNetInfoRequest()
	netReq.DBInstanceId = dbInstanceId
	netResp, netErr := handler.Client.DescribeDBInstanceNetInfo(netReq)
	if netErr == nil && netResp != nil {
		for _, netInfo := range netResp.DBInstanceNetInfos.DBInstanceNetInfo {
			if netInfo.IPType == "Public" {
				// Add port to endpoint
				if netInfo.Port != "" {
					rdbmsInfo.Endpoint = fmt.Sprintf("%s:%s", netInfo.IPAddress, netInfo.Port)
				} else {
					rdbmsInfo.Endpoint = netInfo.IPAddress
				}
				rdbmsInfo.PublicAccess = true
				break
			}
		}
	}

	// Retrieve master username via DescribeAccounts
	acctReq := rds.CreateDescribeAccountsRequest()
	acctReq.DBInstanceId = dbInstanceId
	acctResp, acctErr := handler.Client.DescribeAccounts(acctReq)
	if acctErr == nil && acctResp != nil {
		for _, acct := range acctResp.Accounts.DBInstanceAccount {
			if acct.AccountName != "" {
				rdbmsInfo.MasterUserName = acct.AccountName
				break
			}
		}
	}

	// Retrieve backup policy via DescribeBackupPolicy
	backupReq := rds.CreateDescribeBackupPolicyRequest()
	backupReq.DBInstanceId = dbInstanceId
	backupResp, backupErr := handler.Client.DescribeBackupPolicy(backupReq)
	if backupErr == nil && backupResp != nil {
		rdbmsInfo.BackupRetentionDays = backupResp.BackupRetentionPeriod
		rdbmsInfo.BackupTime = backupResp.PreferredBackupTime
	} else {
		cblogger.Debug("Failed to retrieve backup policy: ", backupErr)
	}

	return rdbmsInfo, nil
}

func (handler *AlibabaRDBMSHandler) convertAttributeToRDBMSInfo(attr *rds.DBInstanceAttribute) irs.RDBMSInfo {
	rdbmsInfo := irs.RDBMSInfo{}

	rdbmsInfo.IId = irs.IID{
		NameId:   attr.DBInstanceDescription,
		SystemId: attr.DBInstanceId,
	}
	if rdbmsInfo.IId.NameId == "" {
		rdbmsInfo.IId.NameId = attr.DBInstanceId
	}

	rdbmsInfo.DBEngine = strings.ToLower(attr.Engine)
	rdbmsInfo.DBEngineVersion = attr.EngineVersion
	rdbmsInfo.DBInstanceSpec = attr.DBInstanceClass
	rdbmsInfo.StorageType = attr.DBInstanceStorageType
	rdbmsInfo.StorageSize = strconv.Itoa(attr.DBInstanceStorage)
	rdbmsInfo.Endpoint = attr.ConnectionString
	rdbmsInfo.MasterUserName = "" // Retrieved via DescribeAccounts in getDBInstanceAttribute

	// VPC
	rdbmsInfo.VpcIID = irs.IID{SystemId: attr.VpcId}

	// Subnet (VSwitch)
	if attr.VSwitchId != "" {
		rdbmsInfo.SubnetIIDs = []irs.IID{{SystemId: attr.VSwitchId}}
	}

	// Status
	rdbmsInfo.Status = convertAlibabaStatusToRDBMSStatus(attr.DBInstanceStatus)

	// HA and DBInstanceType (Category)
	if attr.Category == "HighAvailability" || attr.Category == "AlwaysOn" || attr.Category == "Finance" {
		rdbmsInfo.HighAvailability = true
	}
	rdbmsInfo.DBInstanceType = attr.Category

	// Deletion protection
	rdbmsInfo.DeletionProtection = attr.DeletionProtection

	// Created time
	if attr.CreationTime != "" {
		t, err := time.Parse("2006-01-02T15:04:05Z", attr.CreationTime)
		if err == nil {
			rdbmsInfo.CreatedTime = t
		}
	}

	// Backup/Encryption are not directly exposed in attribute - requires separate API calls
	// BackupRetentionDays and BackupTime are retrieved via DescribeBackupPolicy in getDBInstanceAttribute
	rdbmsInfo.BackupRetentionDays = 0
	rdbmsInfo.BackupTime = "NA"
	rdbmsInfo.Encryption = false // Requires separate DescribeDBInstanceSSL/encryption API

	// PublicAccess is determined by DescribeDBInstanceNetInfo in getDBInstanceAttribute
	rdbmsInfo.PublicAccess = false

	// KeyValueList
	rdbmsInfo.KeyValueList = irs.StructToKeyValueList(attr)

	return rdbmsInfo
}

func (handler *AlibabaRDBMSHandler) convertListItemToRDBMSInfo(db *rds.DBInstance) irs.RDBMSInfo {
	rdbmsInfo := irs.RDBMSInfo{}

	rdbmsInfo.IId = irs.IID{
		NameId:   db.DBInstanceDescription,
		SystemId: db.DBInstanceId,
	}
	if rdbmsInfo.IId.NameId == "" {
		rdbmsInfo.IId.NameId = db.DBInstanceId
	}

	rdbmsInfo.DBEngine = strings.ToLower(db.Engine)
	rdbmsInfo.DBEngineVersion = db.EngineVersion
	rdbmsInfo.DBInstanceSpec = db.DBInstanceClass
	rdbmsInfo.StorageType = db.DBInstanceStorageType
	rdbmsInfo.Endpoint = db.ConnectionString
	rdbmsInfo.MasterUserName = "NA"
	rdbmsInfo.StorageSize = "NA"
	rdbmsInfo.BackupTime = "NA"

	rdbmsInfo.VpcIID = irs.IID{SystemId: db.VpcId}
	if db.VSwitchId != "" {
		rdbmsInfo.SubnetIIDs = []irs.IID{{SystemId: db.VSwitchId}}
	}

	rdbmsInfo.Status = convertAlibabaStatusToRDBMSStatus(db.DBInstanceStatus)

	if db.Category == "HighAvailability" || db.Category == "AlwaysOn" || db.Category == "Finance" {
		rdbmsInfo.HighAvailability = true
	}
	rdbmsInfo.DBInstanceType = db.Category

	if db.CreateTime != "" {
		t, err := time.Parse("2006-01-02T15:04:05Z", db.CreateTime)
		if err == nil {
			rdbmsInfo.CreatedTime = t
		}
	}

	return rdbmsInfo
}

func convertAlibabaStatusToRDBMSStatus(status string) irs.RDBMSStatus {
	switch status {
	case "Creating", "Restoring", "Importing":
		return irs.RDBMSCreating
	case "Running":
		return irs.RDBMSAvailable
	case "Deleting":
		return irs.RDBMSDeleting
	case "DBInstanceClassChanging", "GuardDBInstanceCreating", "ReplicaCreating",
		"Rebooting", "Transing", "TransingToOthers", "EngineVersionUpgrading",
		"MinorVersionUpgrading", "NET_CREATING", "NET_DELETING",
		"Switching", "DISK_EXPANDING":
		return irs.RDBMSAvailable
	case "Locked", "LockMode":
		return irs.RDBMSStopped
	default:
		return irs.RDBMSError
	}
}

func (handler *AlibabaRDBMSHandler) waitForDBInstanceStatus(dbInstanceId string, targetStatus string, timeoutSec int) error {
	for elapsed := 0; elapsed < timeoutSec; elapsed += 15 {
		request := rds.CreateDescribeDBInstanceAttributeRequest()
		request.DBInstanceId = dbInstanceId

		response, err := handler.Client.DescribeDBInstanceAttribute(request)
		if err != nil {
			return err
		}

		if len(response.Items.DBInstanceAttribute) > 0 {
			if response.Items.DBInstanceAttribute[0].DBInstanceStatus == targetStatus {
				return nil
			}
		}

		time.Sleep(15 * time.Second)
	}
	return fmt.Errorf("timeout waiting for instance %s to reach status %s", dbInstanceId, targetStatus)
}

// ─── rdbmsDatabaseManager interface implementation ───────────────────────────

// CreateDatabase creates a database in an Alibaba Cloud RDS instance.
func (handler *AlibabaRDBMSHandler) CreateDatabase(rdbmsSystemId, dbEngine, dbName string) error {
	req := rds.CreateCreateDatabaseRequest()
	req.DBInstanceId = rdbmsSystemId
	req.DBName = dbName
	req.CharacterSetName = "utf8mb4"
	if _, err := handler.Client.CreateDatabase(req); err != nil {
		return fmt.Errorf("Alibaba CreateDatabase: %w", err)
	}
	return nil
}

// ListDatabases lists all databases in an Alibaba Cloud RDS instance.
func (handler *AlibabaRDBMSHandler) ListDatabases(rdbmsSystemId, dbEngine string) ([]string, error) {
	req := rds.CreateDescribeDatabasesRequest()
	req.DBInstanceId = rdbmsSystemId
	resp, err := handler.Client.DescribeDatabases(req)
	if err != nil {
		return nil, fmt.Errorf("Alibaba ListDatabases: %w", err)
	}
	var names []string
	for _, db := range resp.Databases.Database {
		names = append(names, db.DBName)
	}
	return names, nil
}

// DeleteDatabase deletes a database from an Alibaba Cloud RDS instance.
func (handler *AlibabaRDBMSHandler) DeleteDatabase(rdbmsSystemId, dbEngine, dbName string) error {
	req := rds.CreateDeleteDatabaseRequest()
	req.DBInstanceId = rdbmsSystemId
	req.DBName = dbName
	if _, err := handler.Client.DeleteDatabase(req); err != nil {
		return fmt.Errorf("Alibaba DeleteDatabase: %w", err)
	}
	return nil
}
