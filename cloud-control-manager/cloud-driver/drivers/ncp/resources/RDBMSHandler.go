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
	"strconv"
	"strings"
	"time"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vmysql"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vpostgresql"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVpcRDBMSHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	MysqlClient    *vmysql.APIClient
	PostgresClient *vpostgresql.APIClient
	VServerClient  *vserver.APIClient
}

// GetMetaInfo returns metadata about NCP RDBMS capabilities by dynamically querying the CSP API.
func (handler *NcpVpcRDBMSHandler) GetMetaInfo(dbEngine string) (irs.RDBMSMetaInfo, error) {
	cblogger.Debug("NCP VPC RDBMSHandler GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()

	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return irs.RDBMSMetaInfo{}, err
	}

	var metaInfo irs.RDBMSMetaInfo
	switch requestedEngine {
	case "mysql":
		metaInfo, err = handler.getMysqlMetaInfo()
	case "postgresql":
		metaInfo, err = handler.getPostgresqlMetaInfo()
	default:
		return irs.RDBMSMetaInfo{}, fmt.Errorf("unsupported DBEngine '%s': NCP supports mysql and postgresql", requestedEngine)
	}
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return metaInfo, nil
}

// getMysqlMetaInfo fetches MySQL-specific metadata dynamically from the NCP CSP API.
// Only G3 generation products are supported.
func (handler *NcpVpcRDBMSHandler) getMysqlMetaInfo() (irs.RDBMSMetaInfo, error) {
	// Step 1: query image product list for supported versions.
	// Image products expose EngineVersionCode.
	imgResp, err := handler.MysqlClient.V2Api.GetCloudMysqlImageProductList(&vmysql.GetCloudMysqlImageProductListRequest{
		RegionCode: &handler.RegionInfo.Region,
	})
	if err != nil {
		return irs.RDBMSMetaInfo{}, fmt.Errorf("failed to query NCP MySQL image product list: %w", err)
	}
	if len(imgResp.ProductList) == 0 {
		return irs.RDBMSMetaInfo{}, errors.New("NCP MySQL image product list is empty")
	}

	// Filter G3 generation products only
	versionSeen := map[string]bool{}
	var versions []string
	var g3ImageProductCode string
	for _, p := range imgResp.ProductList {
		// Filter G3 generation only
		if p.GenerationCode == nil || *p.GenerationCode != "G3" {
			continue
		}
		if p.EngineVersionCode != nil && *p.EngineVersionCode != "" && !versionSeen[*p.EngineVersionCode] {
			versionSeen[*p.EngineVersionCode] = true
			versions = append(versions, *p.EngineVersionCode)
			if g3ImageProductCode == "" && p.ProductCode != nil {
				g3ImageProductCode = *p.ProductCode
			}
		}
	}
	if len(versions) == 0 {
		return irs.RDBMSMetaInfo{}, errors.New("no G3 generation MySQL versions found in NCP")
	}

	// Step 2: Query product list for available instance specs (CloudMysqlProductCode)
	// Use G3 image product code to query available server specs
	var instanceSpecs []string
	if g3ImageProductCode != "" {
		var err error
		instanceSpecs, err = handler.fetchMysqlProductSpecs(g3ImageProductCode)
		if err != nil {
			cblogger.Infof("Failed to fetch MySQL G3 product specs, using empty list: %v", err)
			instanceSpecs = []string{}
		}
	}

	// StorageTypeOptions: NCP G3 generation sets SSD automatically; not user-selectable.
	// StorageSizeRange: shown for reference only. NCP G3 starts at 10GB and auto-scales by 10GB increments up to 6000GB.
	return irs.RDBMSMetaInfo{
		DBEngine:                         "mysql",
		SupportedVersions:                versions,
		DBInstanceSpecOptions:            instanceSpecs,
		StorageTypeOptions:               []string{"NA"},
		StorageSizeRange:                 irs.StorageSizeRange{Min: 10, Max: 6000},
		SupportsHighAvailability:         true,
		SupportsBackup:                   true,
		SupportsPublicAccess:             false, // NCP does not expose a public domain assignment API; must be done manually via NCP Console
		SupportsDeletionProtection:       true,
		SupportsEncryption:               false, // 2024년 10월 21일부터 신규 서비스 암호화 미제공 (Rocky 8.10)
		SupportsStorageTypeSelection:     false,
		SupportsStorageSizeConfiguration: false,
		RequiresSubnet:                   true,
		RequiresSecurityGroup:            false,
	}, nil
}

// fetchMysqlProductSpecs queries NCP GetCloudMysqlProductList API to retrieve available product codes (instance specs)
// The imageProductCode parameter is already filtered to G3, so no additional generation filtering is needed.
func (handler *NcpVpcRDBMSHandler) fetchMysqlProductSpecs(imageProductCode string) ([]string, error) {
	prodResp, err := handler.MysqlClient.V2Api.GetCloudMysqlProductList(&vmysql.GetCloudMysqlProductListRequest{
		RegionCode:                 &handler.RegionInfo.Region,
		CloudMysqlImageProductCode: &imageProductCode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query NCP MySQL product list: %w", err)
	}
	if len(prodResp.ProductList) == 0 {
		return []string{}, nil
	}

	specSeen := map[string]bool{}
	var specs []string
	for _, p := range prodResp.ProductList {
		if p.ProductCode != nil && *p.ProductCode != "" && !specSeen[*p.ProductCode] {
			specSeen[*p.ProductCode] = true
			specs = append(specs, *p.ProductCode)
		}
	}
	return specs, nil
}

// getPostgresqlMetaInfo fetches PostgreSQL-specific metadata dynamically from the NCP CSP API.
func (handler *NcpVpcRDBMSHandler) getPostgresqlMetaInfo() (irs.RDBMSMetaInfo, error) {
	// Step 1: query image product list for supported versions.
	// Image products expose EngineVersionCode.
	imgResp, err := handler.PostgresClient.V2Api.GetCloudPostgresqlImageProductList(&vpostgresql.GetCloudPostgresqlImageProductListRequest{
		RegionCode: &handler.RegionInfo.Region,
	})
	if err != nil {
		return irs.RDBMSMetaInfo{}, fmt.Errorf("failed to query NCP PostgreSQL image product list: %w", err)
	}
	if len(imgResp.ProductList) == 0 {
		return irs.RDBMSMetaInfo{}, errors.New("NCP PostgreSQL image product list is empty")
	}

	versionSeen := map[string]bool{}
	var versions []string
	for _, p := range imgResp.ProductList {
		if p.EngineVersionCode != nil && *p.EngineVersionCode != "" && !versionSeen[*p.EngineVersionCode] {
			versionSeen[*p.EngineVersionCode] = true
			versions = append(versions, *p.EngineVersionCode)
		}
	}
	if len(versions) == 0 {
		return irs.RDBMSMetaInfo{}, errors.New("no engine version info found in NCP PostgreSQL image products")
	}

	// Step 2: Query product list for available instance specs (CloudPostgresqlProductCode)
	// Use the first image product code to query available server specs
	var imageProductCode string
	if len(imgResp.ProductList) > 0 && imgResp.ProductList[0].ProductCode != nil {
		imageProductCode = *imgResp.ProductList[0].ProductCode
	}

	var instanceSpecs []string
	if imageProductCode != "" {
		var err error
		instanceSpecs, err = handler.fetchPostgresqlProductSpecs(imageProductCode)
		if err != nil {
			cblogger.Infof("Failed to fetch PostgreSQL product specs, using empty list: %v", err)
			instanceSpecs = []string{}
		}
	}

	// StorageTypeOptions: NCP G3 generation sets SSD automatically; not user-selectable.
	// StorageSizeRange: shown for reference only. NCP G3 starts at 10GB and auto-scales by 10GB increments up to 6000GB.
	return irs.RDBMSMetaInfo{
		DBEngine:                         "postgresql",
		SupportedVersions:                versions,
		DBInstanceSpecOptions:            instanceSpecs,
		StorageTypeOptions:               []string{"NA"},
		StorageSizeRange:                 irs.StorageSizeRange{Min: 10, Max: 6000},
		SupportsHighAvailability:         true,
		SupportsBackup:                   true,
		SupportsPublicAccess:             false,
		SupportsDeletionProtection:       false,
		SupportsEncryption:               true,
		SupportsStorageTypeSelection:     false,
		SupportsStorageSizeConfiguration: false,
		RequiresSubnet:                   true,
		RequiresSecurityGroup:            false,
	}, nil
}

// fetchPostgresqlProductSpecs queries NCP GetCloudPostgresqlProductList API to retrieve available product codes (instance specs)
func (handler *NcpVpcRDBMSHandler) fetchPostgresqlProductSpecs(imageProductCode string) ([]string, error) {
	prodResp, err := handler.PostgresClient.V2Api.GetCloudPostgresqlProductList(&vpostgresql.GetCloudPostgresqlProductListRequest{
		RegionCode:                      &handler.RegionInfo.Region,
		CloudPostgresqlImageProductCode: &imageProductCode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query NCP PostgreSQL product list: %w", err)
	}
	if len(prodResp.ProductList) == 0 {
		return []string{}, nil
	}

	specSeen := map[string]bool{}
	var specs []string
	for _, p := range prodResp.ProductList {
		if p.ProductCode != nil && *p.ProductCode != "" && !specSeen[*p.ProductCode] {
			specSeen[*p.ProductCode] = true
			specs = append(specs, *p.ProductCode)
		}
	}
	return specs, nil
}

// ListIID returns a list of all RDBMS instance IIDs (MySQL + PostgreSQL).
func (handler *NcpVpcRDBMSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "ListIID", "GetCloudMysqlInstanceList()+GetCloudPostgresqlInstanceList()")
	start := call.Start()

	var iidList []*irs.IID

	// List MySQL instances
	mysqlResp, err := handler.MysqlClient.V2Api.GetCloudMysqlInstanceList(&vmysql.GetCloudMysqlInstanceListRequest{
		RegionCode: &handler.RegionInfo.Region,
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, fmt.Errorf("failed to list MySQL instances: %w", err)
	}
	for _, inst := range mysqlResp.CloudMysqlInstanceList {
		iidList = append(iidList, &irs.IID{
			NameId:   derefStr(inst.CloudMysqlServiceName),
			SystemId: derefStr(inst.CloudMysqlInstanceNo),
		})
	}

	// List PostgreSQL instances
	pgResp, err := handler.PostgresClient.V2Api.GetCloudPostgresqlInstanceList(&vpostgresql.GetCloudPostgresqlInstanceListRequest{
		RegionCode: &handler.RegionInfo.Region,
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, fmt.Errorf("failed to list PostgreSQL instances: %w", err)
	}
	for _, inst := range pgResp.CloudPostgresqlInstanceList {
		iidList = append(iidList, &irs.IID{
			NameId:   derefStr(inst.CloudPostgresqlServiceName),
			SystemId: derefStr(inst.CloudPostgresqlInstanceNo),
		})
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

// CreateRDBMS creates a new NCP managed database instance.
func (handler *NcpVpcRDBMSHandler) CreateRDBMS(rdbmsReqInfo irs.RDBMSInfo) (irs.RDBMSInfo, error) {
	engineType := strings.ToLower(rdbmsReqInfo.DBEngine)

	hiscallInfo := GetCallLogScheme(handler.RegionInfo.Region, call.RDBMS, rdbmsReqInfo.IId.NameId, "CreateRDBMS()")
	start := call.Start()

	// Validate required fields
	if rdbmsReqInfo.IId.NameId == "" {
		return irs.RDBMSInfo{}, errors.New("RDBMS instance name is required")
	}
	if rdbmsReqInfo.VpcIID.SystemId == "" {
		return irs.RDBMSInfo{}, errors.New("VpcIID.SystemId (VpcNo) is required")
	}
	if len(rdbmsReqInfo.SubnetIIDs) == 0 || rdbmsReqInfo.SubnetIIDs[0].SystemId == "" {
		return irs.RDBMSInfo{}, errors.New("at least one SubnetIID.SystemId (SubnetNo) is required")
	}
	if rdbmsReqInfo.MasterUserName == "" {
		return irs.RDBMSInfo{}, errors.New("MasterUserName is required")
	}
	if rdbmsReqInfo.MasterUserPassword == "" {
		return irs.RDBMSInfo{}, errors.New("MasterUserPassword is required")
	}

	switch engineType {
	case "mysql":
		return handler.createMysqlInstance(rdbmsReqInfo, hiscallInfo, start)
	case "postgresql":
		return handler.createPostgresqlInstance(rdbmsReqInfo, hiscallInfo, start)
	default:
		return irs.RDBMSInfo{}, fmt.Errorf("unsupported DBEngine '%s': NCP supports mysql and postgresql", rdbmsReqInfo.DBEngine)
	}
}

func (handler *NcpVpcRDBMSHandler) createMysqlInstance(reqInfo irs.RDBMSInfo, hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) (irs.RDBMSInfo, error) {
	isHa := reqInfo.HighAvailability
	isBackup := true
	hostIp := "%"
	dbName := "mydb" // NCP requires initial database name

	// Find G3 image product code for the requested engine version
	imageProductCode, err := handler.findMysqlG3ImageProductCode(reqInfo.DBEngineVersion)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, fmt.Errorf("failed to find G3 image product for MySQL version %s: %w", reqInfo.DBEngineVersion, err)
	}

	// CloudMysqlServerNamePrefix: max 15 chars, must not end with '-'
	serverNamePrefix := reqInfo.IId.NameId
	if len(serverNamePrefix) > 15 {
		serverNamePrefix = serverNamePrefix[:15]
	}
	serverNamePrefix = strings.TrimRight(serverNamePrefix, "-")

	createReq := &vmysql.CreateCloudMysqlInstanceRequest{
		RegionCode:                 &handler.RegionInfo.Region,
		VpcNo:                      &reqInfo.VpcIID.SystemId,
		SubnetNo:                   &reqInfo.SubnetIIDs[0].SystemId,
		CloudMysqlServiceName:      &reqInfo.IId.NameId,
		CloudMysqlServerNamePrefix: &serverNamePrefix,
		CloudMysqlUserName:         &reqInfo.MasterUserName,
		CloudMysqlUserPassword:     &reqInfo.MasterUserPassword,
		HostIp:                     &hostIp,
		CloudMysqlDatabaseName:     &dbName,
		IsHa:                       &isHa,
		IsBackup:                   &isBackup,
		CloudMysqlImageProductCode: &imageProductCode,
	}

	// DBInstanceSpec is required: it specifies CPU, memory, and base storage configuration
	if reqInfo.DBInstanceSpec == "" {
		return irs.RDBMSInfo{}, errors.New("DBInstanceSpec is required for NCP MySQL instance creation. Use GetMetaInfo to get available options")
	}
	createReq.CloudMysqlProductCode = &reqInfo.DBInstanceSpec

	if reqInfo.DBEngineVersion != "" {
		createReq.EngineVersionCode = &reqInfo.DBEngineVersion
	}

	// StorageSize: NCP G3 always starts with 10GB and automatically scales up by 10GB as data increases (up to 6000GB).
	// Storage size cannot be specified at creation time.
	if reqInfo.StorageSize != "" {
		return irs.RDBMSInfo{}, fmt.Errorf("StorageSize is not configurable for NCP: NCP G3 starts at 10GB and auto-scales by 10GB increments up to 6000GB. See SupportsStorageSizeConfiguration in GetMetaInfo. Requested: %s GB", reqInfo.StorageSize)
	}

	if reqInfo.StorageType != "" {
		return irs.RDBMSInfo{}, errors.New("StorageType is not supported for NCP: storage type is set automatically by the CSP. See SupportsStorageTypeSelection in GetMetaInfo")
	}
	// Encryption is not configurable at creation via Spider (NCP uses default encryption settings)
	if reqInfo.DeletionProtection {
		createReq.IsDeleteProtection = &reqInfo.DeletionProtection
	}
	backupPeriod := reqInfo.BackupRetentionDays
	if backupPeriod <= 0 {
		backupPeriod = 1 // NCP default
	}
	period := int32(backupPeriod)
	createReq.BackupFileRetentionPeriod = &period
	// BackupTime is not configurable at creation via Spider (NCP auto-assigns)

	resp, err := handler.MysqlClient.V2Api.CreateCloudMysqlInstance(createReq)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, fmt.Errorf("failed to create MySQL instance: %w", err)
	}

	LoggingInfo(hiscallInfo, start)

	if len(resp.CloudMysqlInstanceList) == 0 {
		return irs.RDBMSInfo{}, errors.New("MySQL instance created but no instance returned in response")
	}

	instanceNo := derefStr(resp.CloudMysqlInstanceList[0].CloudMysqlInstanceNo)
	instForInfo := resp.CloudMysqlInstanceList[0]

	if reqInfo.PublicAccess {
		port := "3306"
		// NCP only populates AccessControlGroupNoList once the instance reaches "running" state.
		cblogger.Infof("NCP MySQL: waiting for instance %s to reach running state before configuring ACG...", instanceNo)
		runningInst, waitErr := handler.waitForMysqlRunningAndGet(instanceNo, 40*time.Minute)
		if waitErr != nil {
			cblogger.Warnf("NCP MySQL: error waiting for running state: %v; ACG not configured", waitErr)
		} else {
			instForInfo = runningInst
			if len(runningInst.AccessControlGroupNoList) > 0 {
				acgNo := derefStr(runningInst.AccessControlGroupNoList[0])
				if err := handler.addPublicACGInboundRule(reqInfo.VpcIID.SystemId, acgNo, port); err != nil {
					cblogger.Warnf("NCP MySQL: failed to open ACG %s for public access: %v", acgNo, err)
				} else {
					cblogger.Infof("NCP MySQL: opened ACG %s (TCP 0.0.0.0/0 -> %s) for public access", acgNo, port)
				}
			} else {
				cblogger.Warnf("NCP MySQL: instance running but ACG list empty for instance %s; add inbound rule 0.0.0.0/0->%s manually", instanceNo, port)
			}
		}
	}

	info := handler.convertMysqlInstanceToRDBMSInfo(instForInfo)
	// NCP does not expose master username via API; populate from request
	info.MasterUserName = reqInfo.MasterUserName
	return info, nil
}

func (handler *NcpVpcRDBMSHandler) createPostgresqlInstance(reqInfo irs.RDBMSInfo, hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) (irs.RDBMSInfo, error) {
	isHa := reqInfo.HighAvailability
	isBackup := true
	clientCidr := "0.0.0.0/0"
	dbName := "mydb" // NCP requires initial database name

	// Find G3 image product code for the requested engine version
	imageProductCode, err := handler.findPostgresqlG3ImageProductCode(reqInfo.DBEngineVersion)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, fmt.Errorf("failed to find G3 image product for PostgreSQL version %s: %w", reqInfo.DBEngineVersion, err)
	}

	createReq := &vpostgresql.CreateCloudPostgresqlInstanceRequest{
		RegionCode:                      &handler.RegionInfo.Region,
		VpcNo:                           &reqInfo.VpcIID.SystemId,
		SubnetNo:                        &reqInfo.SubnetIIDs[0].SystemId,
		CloudPostgresqlServiceName:      &reqInfo.IId.NameId,
		CloudPostgresqlServerNamePrefix: &reqInfo.IId.NameId,
		CloudPostgresqlUserName:         &reqInfo.MasterUserName,
		CloudPostgresqlUserPassword:     &reqInfo.MasterUserPassword,
		ClientCidr:                      &clientCidr,
		CloudPostgresqlDatabaseName:     &dbName,
		IsHa:                            &isHa,
		IsBackup:                        &isBackup,
		CloudPostgresqlImageProductCode: &imageProductCode,
	}

	// DBInstanceSpec is required: it specifies CPU, memory, and base storage configuration
	if reqInfo.DBInstanceSpec == "" {
		return irs.RDBMSInfo{}, errors.New("DBInstanceSpec is required for NCP PostgreSQL instance creation. Use GetMetaInfo to get available options")
	}
	createReq.CloudPostgresqlProductCode = &reqInfo.DBInstanceSpec

	if reqInfo.DBEngineVersion != "" {
		createReq.EngineVersionCode = &reqInfo.DBEngineVersion
	}

	// StorageSize: NCP G3 always starts with 10GB and automatically scales up by 10GB as data increases (up to 6000GB).
	// Storage size cannot be specified at creation time.
	if reqInfo.StorageSize != "" {
		return irs.RDBMSInfo{}, fmt.Errorf("StorageSize is not configurable for NCP: NCP G3 starts at 10GB and auto-scales by 10GB increments up to 6000GB. See SupportsStorageSizeConfiguration in GetMetaInfo. Requested: %s GB", reqInfo.StorageSize)
	}

	if reqInfo.StorageType != "" {
		return irs.RDBMSInfo{}, errors.New("StorageType is not supported for NCP: storage type is set automatically by the CSP. See SupportsStorageTypeSelection in GetMetaInfo")
	}
	// Encryption is not configurable at creation via Spider (NCP uses default encryption settings)
	backupPeriod := reqInfo.BackupRetentionDays
	if backupPeriod <= 0 {
		backupPeriod = 1 // NCP default
	}
	period := int32(backupPeriod)
	createReq.BackupFileRetentionPeriod = &period
	// BackupTime is not configurable at creation via Spider (NCP auto-assigns)

	resp, err := handler.PostgresClient.V2Api.CreateCloudPostgresqlInstance(createReq)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, fmt.Errorf("failed to create PostgreSQL instance: %w", err)
	}

	LoggingInfo(hiscallInfo, start)

	if len(resp.CloudPostgresqlInstanceList) == 0 {
		return irs.RDBMSInfo{}, errors.New("PostgreSQL instance created but no instance returned in response")
	}

	instanceNo := derefStr(resp.CloudPostgresqlInstanceList[0].CloudPostgresqlInstanceNo)
	instForInfo := resp.CloudPostgresqlInstanceList[0]

	if reqInfo.PublicAccess {
		port := "5432"
		cblogger.Infof("NCP PostgreSQL: waiting for instance %s to reach running state before configuring ACG...", instanceNo)
		runningInst, waitErr := handler.waitForPostgresqlRunningAndGet(instanceNo, 40*time.Minute)
		if waitErr != nil {
			cblogger.Warnf("NCP PostgreSQL: error waiting for running state: %v; ACG not configured", waitErr)
		} else {
			instForInfo = runningInst
			if len(runningInst.AccessControlGroupNoList) > 0 {
				acgNo := derefStr(runningInst.AccessControlGroupNoList[0])
				if err := handler.addPublicACGInboundRule(reqInfo.VpcIID.SystemId, acgNo, port); err != nil {
					cblogger.Warnf("NCP PostgreSQL: failed to open ACG %s for public access: %v", acgNo, err)
				} else {
					cblogger.Infof("NCP PostgreSQL: opened ACG %s (TCP 0.0.0.0/0 -> %s) for public access", acgNo, port)
				}
			} else {
				cblogger.Warnf("NCP PostgreSQL: instance running but ACG list empty for instance %s; add inbound rule 0.0.0.0/0->%s manually", instanceNo, port)
			}
		}
	}

	info := handler.convertPostgresqlInstanceToRDBMSInfo(instForInfo)
	// NCP does not expose master username via API; populate from request
	info.MasterUserName = reqInfo.MasterUserName
	return info, nil
}

// ListRDBMS returns a list of all RDBMS instances (MySQL + PostgreSQL).
func (handler *NcpVpcRDBMSHandler) ListRDBMS() ([]*irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "ListRDBMS", "GetCloudMysqlInstanceList()+GetCloudPostgresqlInstanceList()")
	start := call.Start()

	var rdbmsList []*irs.RDBMSInfo

	// List MySQL instances
	mysqlResp, err := handler.MysqlClient.V2Api.GetCloudMysqlInstanceList(&vmysql.GetCloudMysqlInstanceListRequest{
		RegionCode: &handler.RegionInfo.Region,
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, fmt.Errorf("failed to list MySQL instances: %w", err)
	}
	for _, inst := range mysqlResp.CloudMysqlInstanceList {
		info := handler.convertMysqlInstanceToRDBMSInfo(inst)
		rdbmsList = append(rdbmsList, &info)
	}

	// List PostgreSQL instances
	pgResp, err := handler.PostgresClient.V2Api.GetCloudPostgresqlInstanceList(&vpostgresql.GetCloudPostgresqlInstanceListRequest{
		RegionCode: &handler.RegionInfo.Region,
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, fmt.Errorf("failed to list PostgreSQL instances: %w", err)
	}
	for _, inst := range pgResp.CloudPostgresqlInstanceList {
		info := handler.convertPostgresqlInstanceToRDBMSInfo(inst)
		rdbmsList = append(rdbmsList, &info)
	}

	LoggingInfo(hiscallInfo, start)
	return rdbmsList, nil
}

// GetRDBMS retrieves a specific RDBMS instance by IID.
func (handler *NcpVpcRDBMSHandler) GetRDBMS(rdbmsIID irs.IID) (irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.RegionInfo.Region, call.RDBMS, rdbmsIID.NameId, "GetRDBMS()")
	start := call.Start()

	// Try MySQL first
	if rdbmsIID.SystemId != "" {
		mysqlResp, err := handler.MysqlClient.V2Api.GetCloudMysqlInstanceDetail(&vmysql.GetCloudMysqlInstanceDetailRequest{
			RegionCode:           &handler.RegionInfo.Region,
			CloudMysqlInstanceNo: &rdbmsIID.SystemId,
		})
		if err == nil && len(mysqlResp.CloudMysqlInstanceList) > 0 {
			LoggingInfo(hiscallInfo, start)
			return handler.convertMysqlInstanceToRDBMSInfo(mysqlResp.CloudMysqlInstanceList[0]), nil
		}

		// Try PostgreSQL
		pgResp, err := handler.PostgresClient.V2Api.GetCloudPostgresqlInstanceDetail(&vpostgresql.GetCloudPostgresqlInstanceDetailRequest{
			RegionCode:                &handler.RegionInfo.Region,
			CloudPostgresqlInstanceNo: &rdbmsIID.SystemId,
		})
		if err == nil && len(pgResp.CloudPostgresqlInstanceList) > 0 {
			LoggingInfo(hiscallInfo, start)
			return handler.convertPostgresqlInstanceToRDBMSInfo(pgResp.CloudPostgresqlInstanceList[0]), nil
		}

		notFoundErr := fmt.Errorf("RDBMS instance with SystemId '%s' not found", rdbmsIID.SystemId)
		LoggingError(hiscallInfo, notFoundErr)
		return irs.RDBMSInfo{}, notFoundErr
	}

	// Search by NameId
	if rdbmsIID.NameId != "" {
		// Search MySQL
		mysqlResp, err := handler.MysqlClient.V2Api.GetCloudMysqlInstanceList(&vmysql.GetCloudMysqlInstanceListRequest{
			RegionCode:            &handler.RegionInfo.Region,
			CloudMysqlServiceName: &rdbmsIID.NameId,
		})
		if err == nil && len(mysqlResp.CloudMysqlInstanceList) > 0 {
			LoggingInfo(hiscallInfo, start)
			return handler.convertMysqlInstanceToRDBMSInfo(mysqlResp.CloudMysqlInstanceList[0]), nil
		}

		// Search PostgreSQL
		pgResp, err := handler.PostgresClient.V2Api.GetCloudPostgresqlInstanceList(&vpostgresql.GetCloudPostgresqlInstanceListRequest{
			RegionCode:                 &handler.RegionInfo.Region,
			CloudPostgresqlServiceName: &rdbmsIID.NameId,
		})
		if err == nil && len(pgResp.CloudPostgresqlInstanceList) > 0 {
			LoggingInfo(hiscallInfo, start)
			return handler.convertPostgresqlInstanceToRDBMSInfo(pgResp.CloudPostgresqlInstanceList[0]), nil
		}
	}

	notFoundErr := fmt.Errorf("RDBMS instance '%s/%s' not found", rdbmsIID.NameId, rdbmsIID.SystemId)
	LoggingError(hiscallInfo, notFoundErr)
	return irs.RDBMSInfo{}, notFoundErr
}

// DeleteRDBMS deletes an NCP managed database instance.
func (handler *NcpVpcRDBMSHandler) DeleteRDBMS(rdbmsIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(handler.RegionInfo.Region, call.RDBMS, rdbmsIID.NameId, "DeleteRDBMS()")
	start := call.Start()

	// Need to find the instance first to determine engine type
	info, err := handler.GetRDBMS(rdbmsIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, fmt.Errorf("failed to find RDBMS instance for deletion: %w", err)
	}

	engineType := strings.ToLower(info.DBEngine)
	systemID := info.IId.SystemId

	switch {
	case strings.Contains(engineType, "mysql"):
		_, err = handler.MysqlClient.V2Api.DeleteCloudMysqlInstance(&vmysql.DeleteCloudMysqlInstanceRequest{
			RegionCode:           &handler.RegionInfo.Region,
			CloudMysqlInstanceNo: &systemID,
		})
	case strings.Contains(engineType, "postgresql"):
		_, err = handler.PostgresClient.V2Api.DeleteCloudPostgresqlInstance(&vpostgresql.DeleteCloudPostgresqlInstanceRequest{
			RegionCode:                &handler.RegionInfo.Region,
			CloudPostgresqlInstanceNo: &systemID,
		})
	default:
		return false, fmt.Errorf("unknown engine type '%s' for instance '%s'", engineType, systemID)
	}

	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, fmt.Errorf("failed to delete RDBMS instance '%s': %w", systemID, err)
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// ---- Helper functions ----

// extractStorageSizeFromProductCode parses storage size (GB) from NCP product code.
// Product code format: SVR.VDBAS.AMD.STAND.C002.M008.NET.SSD.B050.G003
// .BXXX. pattern indicates storage size (e.g., B050 = 50GB, B100 = 100GB)
// Returns 0 if pattern not found.
func extractStorageSizeFromProductCode(productCode string) int {
	parts := strings.Split(productCode, ".")
	for _, part := range parts {
		if len(part) > 1 && part[0] == 'B' {
			// Extract numeric portion (e.g., "B050" -> "050" -> 50)
			sizeStr := part[1:]
			size, err := strconv.Atoi(sizeStr)
			if err == nil && size > 0 {
				return size
			}
		}
	}
	return 0
}

// calculateStorageRangeFromSpecs calculates min/max storage size from product code list.
// If no valid storage sizes found, returns (0, 0) to indicate "not applicable".
func calculateStorageRangeFromSpecs(specs []string) (int, int) {
	if len(specs) == 0 {
		return 0, 0
	}

	minSize := 0
	maxSize := 0
	for _, spec := range specs {
		size := extractStorageSizeFromProductCode(spec)
		if size > 0 {
			if minSize == 0 || size < minSize {
				minSize = size
			}
			if size > maxSize {
				maxSize = size
			}
		}
	}

	// If no storage sizes found, return 0,0
	if minSize == 0 && maxSize == 0 {
		return 0, 0
	}
	return minSize, maxSize
}

// findMysqlG3ImageProductCode finds the G3 image product code for the specified MySQL engine version.
func (handler *NcpVpcRDBMSHandler) findMysqlG3ImageProductCode(engineVersion string) (string, error) {
	imgResp, err := handler.MysqlClient.V2Api.GetCloudMysqlImageProductList(&vmysql.GetCloudMysqlImageProductListRequest{
		RegionCode: &handler.RegionInfo.Region,
	})
	if err != nil {
		return "", fmt.Errorf("failed to query NCP MySQL image product list: %w", err)
	}
	if len(imgResp.ProductList) == 0 {
		return "", errors.New("NCP MySQL image product list is empty")
	}

	// Find G3 image product with matching engine version
	for _, p := range imgResp.ProductList {
		// Filter G3 generation only
		if p.GenerationCode == nil || *p.GenerationCode != "G3" {
			continue
		}
		// Match engine version if specified
		if engineVersion != "" {
			if p.EngineVersionCode != nil && *p.EngineVersionCode == engineVersion {
				if p.ProductCode != nil && *p.ProductCode != "" {
					return *p.ProductCode, nil
				}
			}
		} else {
			// If no version specified, return first G3 image product
			if p.ProductCode != nil && *p.ProductCode != "" {
				return *p.ProductCode, nil
			}
		}
	}

	if engineVersion != "" {
		return "", fmt.Errorf("no G3 image product found for MySQL version %s", engineVersion)
	}
	return "", errors.New("no G3 image product found for MySQL")
}

// findPostgresqlG3ImageProductCode finds the G3 image product code for the specified PostgreSQL engine version.
func (handler *NcpVpcRDBMSHandler) findPostgresqlG3ImageProductCode(engineVersion string) (string, error) {
	imgResp, err := handler.PostgresClient.V2Api.GetCloudPostgresqlImageProductList(&vpostgresql.GetCloudPostgresqlImageProductListRequest{
		RegionCode: &handler.RegionInfo.Region,
	})
	if err != nil {
		return "", fmt.Errorf("failed to query NCP PostgreSQL image product list: %w", err)
	}
	if len(imgResp.ProductList) == 0 {
		return "", errors.New("NCP PostgreSQL image product list is empty")
	}

	// Find G3 image product with matching engine version
	for _, p := range imgResp.ProductList {
		// PostgreSQL is G3 only, but check anyway
		if p.GenerationCode != nil && *p.GenerationCode != "G3" {
			continue
		}
		// Match engine version if specified
		if engineVersion != "" {
			if p.EngineVersionCode != nil && *p.EngineVersionCode == engineVersion {
				if p.ProductCode != nil && *p.ProductCode != "" {
					return *p.ProductCode, nil
				}
			}
		} else {
			// If no version specified, return first G3 image product
			if p.ProductCode != nil && *p.ProductCode != "" {
				return *p.ProductCode, nil
			}
		}
	}

	if engineVersion != "" {
		return "", fmt.Errorf("no G3 image product found for PostgreSQL version %s", engineVersion)
	}
	return "", errors.New("no G3 image product found for PostgreSQL")
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func derefInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

func derefInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

// waitForMysqlRunningAndGet polls until the MySQL instance reaches "running" status and returns it.
// Empty API responses (NCP temporarily not queryable during server replacement) are treated as
// in-progress and retried, not as errors.
func (handler *NcpVpcRDBMSHandler) waitForMysqlRunningAndGet(instanceNo string, timeout time.Duration) (*vmysql.CloudMysqlInstance, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := handler.MysqlClient.V2Api.GetCloudMysqlInstanceDetail(&vmysql.GetCloudMysqlInstanceDetailRequest{
			RegionCode:           &handler.RegionInfo.Region,
			CloudMysqlInstanceNo: &instanceNo,
		})
		if err == nil && len(resp.CloudMysqlInstanceList) > 0 {
			inst := resp.CloudMysqlInstanceList[0]
			status := derefStr(inst.CloudMysqlInstanceStatusName)
			cblogger.Infof("NCP MySQL instance %s status: %s", instanceNo, status)
			switch strings.ToLower(status) {
			case "running":
				return inst, nil
			case "deleting", "error":
				return nil, fmt.Errorf("instance %s entered terminal state '%s'", instanceNo, status)
			}
		} else {
			cblogger.Infof("NCP MySQL instance %s temporarily not queryable — retrying", instanceNo)
		}
		time.Sleep(30 * time.Second)
	}
	return nil, fmt.Errorf("timed out waiting for MySQL instance %s to reach running state", instanceNo)
}

// waitForPostgresqlRunningAndGet polls until the PostgreSQL instance reaches "running" status and returns it.
func (handler *NcpVpcRDBMSHandler) waitForPostgresqlRunningAndGet(instanceNo string, timeout time.Duration) (*vpostgresql.CloudPostgresqlInstance, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := handler.PostgresClient.V2Api.GetCloudPostgresqlInstanceDetail(&vpostgresql.GetCloudPostgresqlInstanceDetailRequest{
			RegionCode:                &handler.RegionInfo.Region,
			CloudPostgresqlInstanceNo: &instanceNo,
		})
		if err == nil && len(resp.CloudPostgresqlInstanceList) > 0 {
			inst := resp.CloudPostgresqlInstanceList[0]
			status := derefStr(inst.CloudPostgresqlInstanceStatusName)
			cblogger.Infof("NCP PostgreSQL instance %s status: %s", instanceNo, status)
			switch strings.ToLower(status) {
			case "running":
				return inst, nil
			case "deleting", "error":
				return nil, fmt.Errorf("instance %s entered terminal state '%s'", instanceNo, status)
			}
		} else {
			cblogger.Infof("NCP PostgreSQL instance %s temporarily not queryable — retrying", instanceNo)
		}
		time.Sleep(30 * time.Second)
	}
	return nil, fmt.Errorf("timed out waiting for PostgreSQL instance %s to reach running state", instanceNo)
}

// waitForMysqlRunning polls until the MySQL instance reaches "running" status or times out.
func (handler *NcpVpcRDBMSHandler) waitForMysqlRunning(instanceNo string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := handler.MysqlClient.V2Api.GetCloudMysqlInstanceDetail(&vmysql.GetCloudMysqlInstanceDetailRequest{
			RegionCode:           &handler.RegionInfo.Region,
			CloudMysqlInstanceNo: &instanceNo,
		})
		if err == nil && len(resp.CloudMysqlInstanceList) > 0 {
			status := derefStr(resp.CloudMysqlInstanceList[0].CloudMysqlInstanceStatusName)
			cblogger.Infof("NCP MySQL instance %s status: %s", instanceNo, status)
			switch strings.ToLower(status) {
			case "running":
				return nil
			case "deleting", "error":
				return fmt.Errorf("instance %s entered terminal state '%s'", instanceNo, status)
			}
		}
		time.Sleep(30 * time.Second)
	}
	return fmt.Errorf("timed out waiting for MySQL instance %s to reach running state", instanceNo)
}

// waitForPostgresqlRunning polls until the PostgreSQL instance reaches "running" status or times out.
func (handler *NcpVpcRDBMSHandler) waitForPostgresqlRunning(instanceNo string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := handler.PostgresClient.V2Api.GetCloudPostgresqlInstanceDetail(&vpostgresql.GetCloudPostgresqlInstanceDetailRequest{
			RegionCode:                &handler.RegionInfo.Region,
			CloudPostgresqlInstanceNo: &instanceNo,
		})
		if err == nil && len(resp.CloudPostgresqlInstanceList) > 0 {
			status := derefStr(resp.CloudPostgresqlInstanceList[0].CloudPostgresqlInstanceStatusName)
			cblogger.Infof("NCP PostgreSQL instance %s status: %s", instanceNo, status)
			switch strings.ToLower(status) {
			case "running":
				return nil
			case "deleting", "error":
				return fmt.Errorf("instance %s entered terminal state '%s'", instanceNo, status)
			}
		}
		time.Sleep(30 * time.Second)
	}
	return fmt.Errorf("timed out waiting for PostgreSQL instance %s to reach running state", instanceNo)
}

// addPublicACGInboundRule adds a TCP 0.0.0.0/0 inbound rule for the given port to the
// specified ACG, enabling external client connections when PublicAccess is requested.
func (handler *NcpVpcRDBMSHandler) addPublicACGInboundRule(vpcNo, acgNo, port string) error {
	if handler.VServerClient == nil {
		return fmt.Errorf("VServerClient not initialized; cannot configure ACG")
	}
	protocol := "TCP"
	ipBlock := "0.0.0.0/0"
	_, err := handler.VServerClient.V2Api.AddAccessControlGroupInboundRule(
		&vserver.AddAccessControlGroupInboundRuleRequest{
			RegionCode:           &handler.RegionInfo.Region,
			VpcNo:                &vpcNo,
			AccessControlGroupNo: &acgNo,
			AccessControlGroupRuleList: []*vserver.AddAccessControlGroupRuleParameter{
				{
					ProtocolTypeCode: &protocol,
					IpBlock:          &ipBlock,
					PortRange:        &port,
				},
			},
		},
	)
	return err
}

// getMysqlMasterUserName fetches the DB user list for a MySQL instance and returns
// the first user with DDL authority (the master/owner user created at instance creation).
// Returns "" on any error so callers can fall back to their own stored value.
func (handler *NcpVpcRDBMSHandler) getMysqlMasterUserName(instanceNo string) string {
	pageNo := int32(0)
	pageSize := int32(100)
	resp, err := handler.MysqlClient.V2Api.GetCloudMysqlUserList(&vmysql.GetCloudMysqlUserListRequest{
		RegionCode:           &handler.RegionInfo.Region,
		CloudMysqlInstanceNo: &instanceNo,
		PageNo:               &pageNo,
		PageSize:             &pageSize,
	})
	if err != nil || resp == nil {
		cblogger.Infof("NCP MySQL: could not fetch user list for instance %s: %v", instanceNo, err)
		return ""
	}
	for _, u := range resp.CloudMysqlUserList {
		if u.Authority != nil && strings.ToUpper(*u.Authority) == "DDL" {
			return derefStr(u.UserName)
		}
	}
	return ""
}

// getPostgresqlMasterUserName fetches the DB user list for a PostgreSQL instance and
// returns the first non-replication-role user (the master user created at instance creation).
// Returns "" on any error so callers can fall back to their own stored value.
func (handler *NcpVpcRDBMSHandler) getPostgresqlMasterUserName(instanceNo string) string {
	resp, err := handler.PostgresClient.V2Api.GetCloudPostgresqlUserList(&vpostgresql.GetCloudPostgresqlUserListRequest{
		RegionCode:                &handler.RegionInfo.Region,
		CloudPostgresqlInstanceNo: &instanceNo,
	})
	if err != nil || resp == nil {
		cblogger.Infof("NCP PostgreSQL: could not fetch user list for instance %s: %v", instanceNo, err)
		return ""
	}
	for _, u := range resp.CloudPostgresqlUserList {
		if u.IsReplicationRole != nil && !*u.IsReplicationRole {
			return derefStr(u.UserName)
		}
	}
	// fallback: return first user regardless
	if len(resp.CloudPostgresqlUserList) > 0 {
		return derefStr(resp.CloudPostgresqlUserList[0].UserName)
	}
	return ""
}

func (handler *NcpVpcRDBMSHandler) convertMysqlInstanceToRDBMSInfo(inst *vmysql.CloudMysqlInstance) irs.RDBMSInfo {
	info := irs.RDBMSInfo{
		IId: irs.IID{
			NameId:   derefStr(inst.CloudMysqlServiceName),
			SystemId: derefStr(inst.CloudMysqlInstanceNo),
		},

		DBEngine:        "mysql",
		DBEngineVersion: derefStr(inst.EngineVersion),

		HighAvailability: derefBool(inst.IsHa),
		Encryption:       false,

		BackupRetentionDays: int(derefInt32(inst.BackupFileRetentionPeriod)),
		BackupTime:          derefStr(inst.BackupTime),

		MasterUserName:     handler.getMysqlMasterUserName(derefStr(inst.CloudMysqlInstanceNo)),
		DeletionProtection: false,
		PublicAccess:       false,

		Status: convertNcpMysqlStatusToRDBMSStatus(inst.CloudMysqlInstanceStatusName),
	}

	if inst.CreateDate != nil {
		t, err := time.Parse("2006-01-02T15:04:05+0900", *inst.CreateDate)
		if err == nil {
			info.CreatedTime = t
		}
	}

	// Extract details from first server instance
	if len(inst.CloudMysqlServerInstanceList) > 0 {
		serverInst := inst.CloudMysqlServerInstanceList[0]
		info.VpcIID = irs.IID{NameId: "NA", SystemId: derefStr(serverInst.VpcNo)}
		info.DBInstanceSpec = derefStr(serverInst.CloudMysqlProductCode)

		// Set endpoint with port
		var domain string
		if serverInst.PublicDomain != nil && *serverInst.PublicDomain != "" {
			domain = *serverInst.PublicDomain
			info.PublicAccess = true
		} else {
			domain = derefStr(serverInst.PrivateDomain)
			info.PublicAccess = false
		}

		// Append port to endpoint
		port := derefInt32(inst.CloudMysqlPort)
		if port > 0 {
			info.Endpoint = fmt.Sprintf("%s:%d", domain, port)
		} else {
			info.Endpoint = domain
		}

		info.Encryption = derefBool(serverInst.IsStorageEncryption)

		// Storage size and type: use values from NCP API directly
		storageSizeBytes := derefInt64(serverInst.DataStorageSize)
		if storageSizeBytes > 0 {
			info.StorageSize = strconv.FormatInt(storageSizeBytes/(1024*1024*1024), 10)
		} else {
			info.StorageSize = "NA"
		}

		if serverInst.DataStorageType != nil && serverInst.DataStorageType.CodeName != nil {
			info.StorageType = *serverInst.DataStorageType.CodeName
		} else {
			// G3 generation always uses SSD
			if derefStr(inst.GenerationCode) == "G3" {
				info.StorageType = "SSD"
			} else {
				info.StorageType = "NA"
			}
		}

		if len(inst.CloudMysqlServerInstanceList) > 0 {
			var subnetIIDs []irs.IID
			for _, si := range inst.CloudMysqlServerInstanceList {
				subnetIIDs = append(subnetIIDs, irs.IID{NameId: "NA", SystemId: derefStr(si.SubnetNo)})
			}
			info.SubnetIIDs = subnetIIDs
		}

		if serverInst.CloudMysqlServerRole != nil && serverInst.CloudMysqlServerRole.CodeName != nil {
			info.DBInstanceType = *serverInst.CloudMysqlServerRole.CodeName
		}
	}

	// KeyValueList: NCP-specific fields not covered by standard RDBMSInfo
	kvList := []irs.KeyValue{
		{Key: "GenerationCode", Value: derefStr(inst.GenerationCode)},
		{Key: "ImageProductCode", Value: derefStr(inst.CloudMysqlImageProductCode)},
		{Key: "IsMultiZone", Value: strconv.FormatBool(derefBool(inst.IsMultiZone))},
	}
	if inst.License != nil && inst.License.CodeName != nil {
		kvList = append(kvList, irs.KeyValue{Key: "License", Value: *inst.License.CodeName})
	}
	if len(inst.AccessControlGroupNoList) > 0 {
		acgNos := []string{}
		for _, v := range inst.AccessControlGroupNoList {
			if v != nil {
				acgNos = append(acgNos, *v)
			}
		}
		kvList = append(kvList, irs.KeyValue{Key: "AccessControlGroupNos", Value: strings.Join(acgNos, ",")})
	}
	if len(inst.CloudMysqlConfigList) > 0 {
		configs := []string{}
		for _, v := range inst.CloudMysqlConfigList {
			if v != nil {
				configs = append(configs, *v)
			}
		}
		kvList = append(kvList, irs.KeyValue{Key: "MySQLConfigs", Value: strings.Join(configs, ";")})
	}
	if len(inst.CloudMysqlServerInstanceList) > 0 {
		s := inst.CloudMysqlServerInstanceList[0]
		kvList = append(kvList,
			irs.KeyValue{Key: "ServerName", Value: derefStr(s.CloudMysqlServerName)},
			irs.KeyValue{Key: "ServerInstanceNo", Value: derefStr(s.CloudMysqlServerInstanceNo)},
			irs.KeyValue{Key: "ZoneCode", Value: derefStr(s.ZoneCode)},
			irs.KeyValue{Key: "PrivateIp", Value: derefStr(s.PrivateIp)},
			irs.KeyValue{Key: "CpuCount", Value: strconv.Itoa(int(derefInt32(s.CpuCount)))},
			irs.KeyValue{Key: "MemorySizeGB", Value: strconv.FormatInt(derefInt64(s.MemorySize)/(1024*1024*1024), 10)},
			irs.KeyValue{Key: "UsedStorageSizeGB", Value: strconv.FormatInt(derefInt64(s.UsedDataStorageSize)/(1024*1024*1024), 10)},
			irs.KeyValue{Key: "Uptime", Value: derefStr(s.Uptime)},
		)
	}
	info.KeyValueList = kvList

	return info
}

func (handler *NcpVpcRDBMSHandler) convertPostgresqlInstanceToRDBMSInfo(inst *vpostgresql.CloudPostgresqlInstance) irs.RDBMSInfo {
	info := irs.RDBMSInfo{
		IId: irs.IID{
			NameId:   derefStr(inst.CloudPostgresqlServiceName),
			SystemId: derefStr(inst.CloudPostgresqlInstanceNo),
		},

		DBEngine:        "postgresql",
		DBEngineVersion: derefStr(inst.EngineVersion),

		HighAvailability: derefBool(inst.IsHa),
		Encryption:       false,

		BackupRetentionDays: int(derefInt32(inst.BackupFileRetentionPeriod)),
		BackupTime:          derefStr(inst.BackupTime),

		MasterUserName:     handler.getPostgresqlMasterUserName(derefStr(inst.CloudPostgresqlInstanceNo)),
		DeletionProtection: false, // PostgreSQL does not support deletion protection
		PublicAccess:       false,

		Status: convertNcpPostgresqlStatusToRDBMSStatus(inst.CloudPostgresqlInstanceStatusName),
	}

	if inst.CreateDate != nil {
		t, err := time.Parse("2006-01-02T15:04:05+0900", *inst.CreateDate)
		if err == nil {
			info.CreatedTime = t
		}
	}

	// Extract details from first server instance
	if len(inst.CloudPostgresqlServerInstanceList) > 0 {
		serverInst := inst.CloudPostgresqlServerInstanceList[0]
		info.VpcIID = irs.IID{NameId: "NA", SystemId: derefStr(serverInst.VpcNo)}
		info.DBInstanceSpec = derefStr(serverInst.CloudPostgresqlProductCode)

		// Set endpoint with port
		var domain string
		if serverInst.PublicDomain != nil && *serverInst.PublicDomain != "" {
			domain = *serverInst.PublicDomain
			info.PublicAccess = true
		} else {
			domain = derefStr(serverInst.PrivateDomain)
			info.PublicAccess = false
		}

		// Append port to endpoint
		port := derefInt32(inst.CloudPostgresqlPort)
		if port > 0 {
			info.Endpoint = fmt.Sprintf("%s:%d", domain, port)
		} else {
			info.Endpoint = domain
		}

		info.Encryption = derefBool(serverInst.IsStorageEncryption)

		// Storage size and type: use values from NCP API directly
		storageSizeBytes := derefInt64(serverInst.DataStorageSize)
		if storageSizeBytes > 0 {
			info.StorageSize = strconv.FormatInt(storageSizeBytes/(1024*1024*1024), 10)
		} else {
			info.StorageSize = "NA"
		}

		if serverInst.DataStorageType != nil && serverInst.DataStorageType.CodeName != nil {
			info.StorageType = *serverInst.DataStorageType.CodeName
		} else {
			// G3 generation always uses SSD
			if derefStr(inst.GenerationCode) == "G3" {
				info.StorageType = "SSD"
			} else {
				info.StorageType = "NA"
			}
		}

		if len(inst.CloudPostgresqlServerInstanceList) > 0 {
			var subnetIIDs []irs.IID
			for _, si := range inst.CloudPostgresqlServerInstanceList {
				subnetIIDs = append(subnetIIDs, irs.IID{NameId: "NA", SystemId: derefStr(si.SubnetNo)})
			}
			info.SubnetIIDs = subnetIIDs
		}

		if serverInst.CloudPostgresqlServerRole != nil && serverInst.CloudPostgresqlServerRole.CodeName != nil {
			info.DBInstanceType = *serverInst.CloudPostgresqlServerRole.CodeName
		}
	}

	// KeyValueList: NCP-specific fields not covered by standard RDBMSInfo
	kvList := []irs.KeyValue{
		{Key: "GenerationCode", Value: derefStr(inst.GenerationCode)},
		{Key: "ImageProductCode", Value: derefStr(inst.CloudPostgresqlImageProductCode)},
		{Key: "IsMultiZone", Value: strconv.FormatBool(derefBool(inst.IsMultiZone))},
		{Key: "License", Value: derefStr(inst.License)},
	}
	if len(inst.AccessControlGroupNoList) > 0 {
		acgNos := []string{}
		for _, v := range inst.AccessControlGroupNoList {
			if v != nil {
				acgNos = append(acgNos, *v)
			}
		}
		kvList = append(kvList, irs.KeyValue{Key: "AccessControlGroupNos", Value: strings.Join(acgNos, ",")})
	}
	if len(inst.CloudPostgresqlConfigList) > 0 {
		configs := []string{}
		for _, v := range inst.CloudPostgresqlConfigList {
			if v != nil {
				configs = append(configs, *v)
			}
		}
		kvList = append(kvList, irs.KeyValue{Key: "PostgreSQLConfigs", Value: strings.Join(configs, ";")})
	}
	if len(inst.CloudPostgresqlServerInstanceList) > 0 {
		s := inst.CloudPostgresqlServerInstanceList[0]
		kvList = append(kvList,
			irs.KeyValue{Key: "ServerName", Value: derefStr(s.CloudPostgresqlServerName)},
			irs.KeyValue{Key: "ServerInstanceNo", Value: derefStr(s.CloudPostgresqlServerInstanceNo)},
			irs.KeyValue{Key: "ZoneCode", Value: derefStr(s.ZoneCode)},
			irs.KeyValue{Key: "PrivateIp", Value: derefStr(s.PrivateIp)},
			irs.KeyValue{Key: "CpuCount", Value: strconv.Itoa(int(derefInt32(s.CpuCount)))},
			irs.KeyValue{Key: "MemorySizeGB", Value: strconv.FormatInt(derefInt64(s.MemorySize)/(1024*1024*1024), 10)},
			irs.KeyValue{Key: "UsedStorageSizeGB", Value: strconv.FormatInt(derefInt64(s.UsedDataStorageSize)/(1024*1024*1024), 10)},
			irs.KeyValue{Key: "Uptime", Value: derefStr(s.Uptime)},
		)
	}
	info.KeyValueList = kvList

	return info
}

func convertNcpMysqlStatusToRDBMSStatus(statusName *string) irs.RDBMSStatus {
	if statusName == nil {
		return irs.RDBMSError
	}
	switch strings.ToLower(*statusName) {
	case "creating", "setup", "settingup", "setting up", "configuring":
		return irs.RDBMSCreating
	case "running":
		return irs.RDBMSAvailable
	case "deleting":
		return irs.RDBMSDeleting
	case "stopped", "shutting down":
		return irs.RDBMSStopped
	default:
		return irs.RDBMSError
	}
}

func convertNcpPostgresqlStatusToRDBMSStatus(statusName *string) irs.RDBMSStatus {
	if statusName == nil {
		return irs.RDBMSError
	}
	switch strings.ToLower(*statusName) {
	case "creating", "setup", "settingup", "setting up", "configuring":
		return irs.RDBMSCreating
	case "running":
		return irs.RDBMSAvailable
	case "deleting":
		return irs.RDBMSDeleting
	case "stopped", "shutting down":
		return irs.RDBMSStopped
	default:
		return irs.RDBMSError
	}
}

// -------- RDBMSDatabaseManager implementation --------
// NcpVpcRDBMSHandler implements the optional irs.RDBMSDatabaseManager interface,
// enabling cloud-native database CRUD via the NCP API so that callers do not need
// elevated SQL privileges (NCP Cloud MySQL/PostgreSQL restrict CREATE DATABASE via SQL).

// CreateDatabase creates a new database inside an NCP MySQL or PostgreSQL instance.
func (handler *NcpVpcRDBMSHandler) CreateDatabase(rdbmsSystemId, dbEngine, dbName string) error {
	cblogger.Infof("NCP RDBMSDatabaseManager: CreateDatabase instanceNo=%s engine=%s db=%s", rdbmsSystemId, dbEngine, dbName)
	engine := strings.ToLower(dbEngine)
	switch {
	case strings.Contains(engine, "mysql"):
		_, err := handler.MysqlClient.V2Api.AddCloudMysqlDatabaseList(&vmysql.AddCloudMysqlDatabaseListRequest{
			RegionCode:                 &handler.RegionInfo.Region,
			CloudMysqlInstanceNo:       &rdbmsSystemId,
			CloudMysqlDatabaseNameList: []*string{&dbName},
		})
		if err != nil {
			return fmt.Errorf("NCP AddCloudMysqlDatabaseList: %w", err)
		}
		return nil
	case strings.Contains(engine, "postgres"):
		// PostgreSQL database creation requires an owner (the master user).
		owner := handler.getPostgresqlMasterUserName(rdbmsSystemId)
		if owner == "" {
			owner = "postgres" // fallback
		}
		_, err := handler.PostgresClient.V2Api.AddCloudPostgresqlDatabaseList(&vpostgresql.AddCloudPostgresqlDatabaseListRequest{
			RegionCode:                &handler.RegionInfo.Region,
			CloudPostgresqlInstanceNo: &rdbmsSystemId,
			CloudPostgresqlDatabaseList: []*vpostgresql.CloudPostgresqlDatabaseParameter{
				{Name: &dbName, Owner: &owner},
			},
		})
		if err != nil {
			return fmt.Errorf("NCP AddCloudPostgresqlDatabaseList: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("NCP RDBMSDatabaseManager: unsupported DB engine: %s", dbEngine)
	}
}

// ListDatabases returns all database names in an NCP MySQL or PostgreSQL instance.
func (handler *NcpVpcRDBMSHandler) ListDatabases(rdbmsSystemId, dbEngine string) ([]string, error) {
	cblogger.Infof("NCP RDBMSDatabaseManager: ListDatabases instanceNo=%s engine=%s", rdbmsSystemId, dbEngine)
	engine := strings.ToLower(dbEngine)
	switch {
	case strings.Contains(engine, "mysql"):
		pageNo := int32(0)
		pageSize := int32(100)
		resp, err := handler.MysqlClient.V2Api.GetCloudMysqlDatabaseList(&vmysql.GetCloudMysqlDatabaseListRequest{
			RegionCode:           &handler.RegionInfo.Region,
			CloudMysqlInstanceNo: &rdbmsSystemId,
			PageNo:               &pageNo,
			PageSize:             &pageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("NCP GetCloudMysqlDatabaseList: %w", err)
		}
		var names []string
		for _, db := range resp.CloudMysqlDatabaseList {
			if db.DatabaseName != nil {
				names = append(names, *db.DatabaseName)
			}
		}
		return names, nil
	case strings.Contains(engine, "postgres"):
		resp, err := handler.PostgresClient.V2Api.GetCloudPostgresqlDatabaseList(&vpostgresql.GetCloudPostgresqlDatabaseListRequest{
			RegionCode:                &handler.RegionInfo.Region,
			CloudPostgresqlInstanceNo: &rdbmsSystemId,
		})
		if err != nil {
			return nil, fmt.Errorf("NCP GetCloudPostgresqlDatabaseList: %w", err)
		}
		var names []string
		for _, db := range resp.CloudPostgresqlDatabaseList {
			if db.DatabaseName != nil {
				names = append(names, *db.DatabaseName)
			}
		}
		return names, nil
	default:
		return nil, fmt.Errorf("NCP RDBMSDatabaseManager: unsupported DB engine: %s", dbEngine)
	}
}

// DeleteDatabase drops a database from an NCP MySQL or PostgreSQL instance.
func (handler *NcpVpcRDBMSHandler) DeleteDatabase(rdbmsSystemId, dbEngine, dbName string) error {
	cblogger.Infof("NCP RDBMSDatabaseManager: DeleteDatabase instanceNo=%s engine=%s db=%s", rdbmsSystemId, dbEngine, dbName)
	engine := strings.ToLower(dbEngine)
	switch {
	case strings.Contains(engine, "mysql"):
		_, err := handler.MysqlClient.V2Api.DeleteCloudMysqlDatabaseList(&vmysql.DeleteCloudMysqlDatabaseListRequest{
			RegionCode:                 &handler.RegionInfo.Region,
			CloudMysqlInstanceNo:       &rdbmsSystemId,
			CloudMysqlDatabaseNameList: []*string{&dbName},
		})
		if err != nil {
			return fmt.Errorf("NCP DeleteCloudMysqlDatabaseList: %w", err)
		}
		return nil
	case strings.Contains(engine, "postgres"):
		_, err := handler.PostgresClient.V2Api.DeleteCloudPostgresqlDatabaseList(&vpostgresql.DeleteCloudPostgresqlDatabaseListRequest{
			RegionCode:                &handler.RegionInfo.Region,
			CloudPostgresqlInstanceNo: &rdbmsSystemId,
			CloudPostgresqlDatabaseList: []*vpostgresql.CloudPostgresqlDatabaseKeyParameter{
				{Name: &dbName},
			},
		})
		if err != nil {
			return fmt.Errorf("NCP DeleteCloudPostgresqlDatabaseList: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("NCP RDBMSDatabaseManager: unsupported DB engine: %s", dbEngine)
	}
}
