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

func (handler *AlibabaRDBMSHandler) GetMetaInfo() (irs.RDBMSMetaInfo, error) {
	cblogger.Debug("Alibaba RDS GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()

	metaInfo := irs.RDBMSMetaInfo{
		SupportedEngines: map[string][]string{
			"mysql":      {"5.7", "8.0"},
			"mariadb":    {"10.3"},
			"postgresql": {"10.0", "11.0", "12.0", "13.0", "14.0", "15.0", "16.0"},
		},
		SupportsHighAvailability:   true,
		SupportsBackup:             true,
		SupportsPublicAccess:       true,
		SupportsDeletionProtection: true,
		SupportsEncryption:         true,
		StorageTypeOptions: map[string][]string{
			"mysql":      {"cloud_ssd", "cloud_essd", "cloud_essd2", "cloud_essd3", "local_ssd"},
			"mariadb":    {"cloud_ssd", "cloud_essd"},
			"postgresql": {"cloud_ssd", "cloud_essd", "cloud_essd2", "cloud_essd3", "local_ssd"},
		},
		StorageSizeRange: irs.StorageSizeRange{
			Min: 20,
			Max: 32768,
		},
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return metaInfo, nil
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

	// Port
	if rdbmsReqInfo.Port != "" {
		request.Port = rdbmsReqInfo.Port
	}

	// HA (Category)
	if rdbmsReqInfo.HighAvailability {
		request.Category = "HighAvailability"
	} else {
		request.Category = "Basic"
	}

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

	// Create master user account
	if rdbmsReqInfo.MasterUserName != "" {
		acctReq := rds.CreateCreateAccountRequest()
		acctReq.DBInstanceId = dbInstanceId
		acctReq.AccountName = rdbmsReqInfo.MasterUserName
		acctReq.AccountPassword = rdbmsReqInfo.MasterUserPassword
		acctReq.AccountType = "Super"
		_, acctErr := handler.Client.CreateAccount(acctReq)
		if acctErr != nil {
			cblogger.Errorf("Failed to create master account [%s] for [%s]: %v", rdbmsReqInfo.MasterUserName, dbInstanceId, acctErr)
			return irs.RDBMSInfo{}, fmt.Errorf("failed to create master account: %w", acctErr)
		}
		cblogger.Infof("Master account [%s] created for [%s]", rdbmsReqInfo.MasterUserName, dbInstanceId)
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
				rdbmsInfo.Endpoint = netInfo.IPAddress
				rdbmsInfo.Port = netInfo.Port
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
	rdbmsInfo.Port = attr.Port
	rdbmsInfo.Endpoint = attr.ConnectionString
	rdbmsInfo.MasterUserName = "" // Retrieved via DescribeAccounts in getDBInstanceAttribute
	rdbmsInfo.DatabaseName = ""

	// VPC
	rdbmsInfo.VpcIID = irs.IID{SystemId: attr.VpcId}

	// Subnet (VSwitch)
	if attr.VSwitchId != "" {
		rdbmsInfo.SubnetIIDs = []irs.IID{{SystemId: attr.VSwitchId}}
	}

	// Status
	rdbmsInfo.Status = convertAlibabaStatusToRDBMSStatus(attr.DBInstanceStatus)

	// HA
	if attr.Category == "HighAvailability" || attr.Category == "AlwaysOn" || attr.Category == "Finance" {
		rdbmsInfo.HighAvailability = true
	}

	// Deletion protection
	rdbmsInfo.DeletionProtection = attr.DeletionProtection

	// Created time
	if attr.CreationTime != "" {
		t, err := time.Parse("2006-01-02T15:04:05Z", attr.CreationTime)
		if err == nil {
			rdbmsInfo.CreatedTime = t
		}
	}

	// Backup/Encryption are not directly exposed in attribute - return NA/defaults
	rdbmsInfo.BackupRetentionDays = 0 // Can be retrieved through DescribeBackupPolicy, but not included in attribute
	rdbmsInfo.BackupTime = "NA"
	rdbmsInfo.Encryption = false // Requires separate DescribeDBInstanceSSL/encryption API
	rdbmsInfo.ReplicationType = "NA"

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
	rdbmsInfo.Port = "NA"
	rdbmsInfo.Endpoint = db.ConnectionString
	rdbmsInfo.MasterUserName = "NA"
	rdbmsInfo.DatabaseName = "NA"
	rdbmsInfo.StorageSize = "NA"
	rdbmsInfo.BackupTime = "NA"
	rdbmsInfo.ReplicationType = "NA"

	rdbmsInfo.VpcIID = irs.IID{SystemId: db.VpcId}
	if db.VSwitchId != "" {
		rdbmsInfo.SubnetIIDs = []irs.IID{{SystemId: db.VSwitchId}}
	}

	rdbmsInfo.Status = convertAlibabaStatusToRDBMSStatus(db.DBInstanceStatus)

	if db.Category == "HighAvailability" || db.Category == "AlwaysOn" || db.Category == "Finance" {
		rdbmsInfo.HighAvailability = true
	}

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
