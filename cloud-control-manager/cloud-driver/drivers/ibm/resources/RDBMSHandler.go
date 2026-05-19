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
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IBM/cloud-databases-go-sdk/clouddatabasesv5"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// IBM Cloud Databases service plan IDs (from IBM Global Catalog)
const (
	ibmMySQLPlanID      = "databases-for-mysql-standard"
	ibmPostgreSQLPlanID = "databases-for-postgresql-standard"
	ibmServiceIDMySQL   = "databases-for-mysql"
	ibmServiceIDPG      = "databases-for-postgresql"
)

type IbmRDBMSHandler struct {
	CredentialInfo     idrv.CredentialInfo
	Region             idrv.RegionInfo
	Ctx                context.Context
	ResourceController *resourcecontrollerv2.ResourceControllerV2
	CloudDBService     *clouddatabasesv5.CloudDatabasesV5
}

func (handler *IbmRDBMSHandler) GetMetaInfo() (irs.RDBMSMetaInfo, error) {
	cblogger.Debug("IBM Cloud GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()

	metaInfo := irs.RDBMSMetaInfo{
		SupportedEngines: map[string][]string{
			"mysql":      {"5.7", "8.0"},
			"postgresql": {"12", "13", "14", "15", "16"},
		},
		SupportsHighAvailability:   true,
		SupportsBackup:             true,
		SupportsPublicAccess:       true,
		SupportsDeletionProtection: true, // IBM uses resource locking
		SupportsEncryption:         true, // Always encrypted at rest
		StorageTypeOptions: map[string][]string{
			"mysql":      {"standard"},
			"postgresql": {"standard"},
		},
		StorageSizeRange: irs.StorageSizeRange{
			Min: 5,
			Max: 4096,
		},
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return metaInfo, nil
}

func (handler *IbmRDBMSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListIID", "ListResourceInstances()")
	start := call.Start()

	var iidList []*irs.IID

	// List MySQL instances
	mysqlIIDs, err := handler.listResourceInstancesByType(ibmServiceIDMySQL)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	iidList = append(iidList, mysqlIIDs...)

	// List PostgreSQL instances
	pgIIDs, err := handler.listResourceInstancesByType(ibmServiceIDPG)
	if err != nil {
		cblogger.Warn("Failed to list PostgreSQL instances: ", err)
	} else {
		iidList = append(iidList, pgIIDs...)
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return iidList, nil
}

func (handler *IbmRDBMSHandler) CreateRDBMS(rdbmsReqInfo irs.RDBMSInfo) (irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsReqInfo.IId.NameId, "CreateResourceInstance()")
	start := call.Start()

	// Validate required fields
	if rdbmsReqInfo.IId.NameId == "" {
		return irs.RDBMSInfo{}, errors.New("RDBMS NameId is required")
	}
	if rdbmsReqInfo.DBEngine == "" {
		return irs.RDBMSInfo{}, errors.New("DBEngine is required (mysql or postgresql)")
	}

	// Determine resource plan ID based on engine
	var resourcePlanID string
	switch rdbmsReqInfo.DBEngine {
	case "mysql":
		resourcePlanID = ibmMySQLPlanID
	case "postgresql":
		resourcePlanID = ibmPostgreSQLPlanID
	default:
		return irs.RDBMSInfo{}, fmt.Errorf("unsupported engine: %s (supported: mysql, postgresql)", rdbmsReqInfo.DBEngine)
	}

	// Build parameters
	params := map[string]interface{}{}
	if rdbmsReqInfo.DBEngineVersion != "" {
		params["version"] = rdbmsReqInfo.DBEngineVersion
	}
	if rdbmsReqInfo.MasterUserPassword != "" {
		params["admin_password"] = rdbmsReqInfo.MasterUserPassword
	}
	if rdbmsReqInfo.HighAvailability {
		params["members"] = 3 // IBM HA uses 3 members
	}

	// Target region
	target := handler.Region.Region

	createOpts := &resourcecontrollerv2.CreateResourceInstanceOptions{
		Name:           &rdbmsReqInfo.IId.NameId,
		Target:         &target,
		ResourcePlanID: &resourcePlanID,
		Parameters:     params,
	}

	// Tags
	if len(rdbmsReqInfo.TagList) > 0 {
		tags := make([]string, 0, len(rdbmsReqInfo.TagList))
		for _, tag := range rdbmsReqInfo.TagList {
			tags = append(tags, tag.Key+":"+tag.Value)
		}
		createOpts.Tags = tags
	}

	result, _, err := handler.ResourceController.CreateResourceInstance(createOpts)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	// Wait for active state
	instanceId := ""
	if result.GUID != nil {
		instanceId = *result.GUID
	}
	if instanceId != "" {
		err = handler.waitForResourceInstanceState(instanceId, "active", 1200)
		if err != nil {
			cblogger.Warn("Instance created but wait for active state failed: ", err)
		}
	}

	return handler.convertResourceInstanceToRDBMSInfo(result), nil
}

func (handler *IbmRDBMSHandler) ListRDBMS() ([]*irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListRDBMS", "ListResourceInstances()")
	start := call.Start()

	var rdbmsList []*irs.RDBMSInfo

	// List MySQL instances
	mysqlInstances, err := handler.listResourceInstances(ibmServiceIDMySQL)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	for _, inst := range mysqlInstances {
		info := handler.convertResourceInstanceToRDBMSInfo(&inst)
		rdbmsList = append(rdbmsList, &info)
	}

	// List PostgreSQL instances
	pgInstances, err := handler.listResourceInstances(ibmServiceIDPG)
	if err != nil {
		cblogger.Warn("Failed to list PostgreSQL instances: ", err)
	} else {
		for _, inst := range pgInstances {
			info := handler.convertResourceInstanceToRDBMSInfo(&inst)
			rdbmsList = append(rdbmsList, &info)
		}
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return rdbmsList, nil
}

func (handler *IbmRDBMSHandler) GetRDBMS(rdbmsIID irs.IID) (irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsIID.NameId, "GetResourceInstance()")
	start := call.Start()

	getOpts := &resourcecontrollerv2.GetResourceInstanceOptions{
		ID: &rdbmsIID.SystemId,
	}

	result, _, err := handler.ResourceController.GetResourceInstance(getOpts)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	return handler.convertResourceInstanceToRDBMSInfo(result), nil
}

func (handler *IbmRDBMSHandler) DeleteRDBMS(rdbmsIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsIID.NameId, "DeleteResourceInstance()")
	start := call.Start()

	deleteOpts := &resourcecontrollerv2.DeleteResourceInstanceOptions{
		ID: &rdbmsIID.SystemId,
	}

	_, err := handler.ResourceController.DeleteResourceInstance(deleteOpts)
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

func (handler *IbmRDBMSHandler) listResourceInstancesByType(serviceID string) ([]*irs.IID, error) {
	instances, err := handler.listResourceInstances(serviceID)
	if err != nil {
		return nil, err
	}

	var iidList []*irs.IID
	for _, inst := range instances {
		name := ""
		sysId := ""
		if inst.Name != nil {
			name = *inst.Name
		}
		if inst.GUID != nil {
			sysId = *inst.GUID
		}
		iidList = append(iidList, &irs.IID{NameId: name, SystemId: sysId})
	}
	return iidList, nil
}

func (handler *IbmRDBMSHandler) listResourceInstances(serviceID string) ([]resourcecontrollerv2.ResourceInstance, error) {
	var allInstances []resourcecontrollerv2.ResourceInstance

	listOpts := &resourcecontrollerv2.ListResourceInstancesOptions{
		ResourceID: &serviceID,
	}

	result, _, err := handler.ResourceController.ListResourceInstances(listOpts)
	if err != nil {
		return nil, err
	}

	if result != nil && result.Resources != nil {
		allInstances = append(allInstances, result.Resources...)
	}

	return allInstances, nil
}

func (handler *IbmRDBMSHandler) convertResourceInstanceToRDBMSInfo(inst *resourcecontrollerv2.ResourceInstance) irs.RDBMSInfo {
	rdbmsInfo := irs.RDBMSInfo{}

	if inst.Name != nil {
		rdbmsInfo.IId.NameId = *inst.Name
	}
	if inst.GUID != nil {
		rdbmsInfo.IId.SystemId = *inst.GUID
	}

	// Determine engine from resource plan ID
	if inst.ResourcePlanID != nil {
		switch *inst.ResourcePlanID {
		case ibmMySQLPlanID:
			rdbmsInfo.DBEngine = "mysql"
			rdbmsInfo.Port = "3306"
		case ibmPostgreSQLPlanID:
			rdbmsInfo.DBEngine = "postgresql"
			rdbmsInfo.Port = "5432"
		}
	}

	// State
	if inst.State != nil {
		rdbmsInfo.Status = convertIBMStateToRDBMSStatus(*inst.State)
	}

	// Created time
	if inst.CreatedAt != nil {
		rdbmsInfo.CreatedTime = time.Time(*inst.CreatedAt)
	}

	// Parameters (version, etc.)
	if inst.Parameters != nil {
		if ver, ok := inst.Parameters["version"]; ok {
			if verStr, ok := ver.(string); ok {
				rdbmsInfo.DBEngineVersion = verStr
			}
		}
	}

	// IBM Cloud Databases are always encrypted at rest
	rdbmsInfo.Encryption = true
	rdbmsInfo.MasterUserName = "admin" // IBM default
	rdbmsInfo.DatabaseName = "NA"
	rdbmsInfo.StorageSize = "NA"
	rdbmsInfo.StorageType = "standard"
	rdbmsInfo.DBInstanceSpec = "NA"
	rdbmsInfo.BackupTime = "NA"
	rdbmsInfo.ReplicationType = "NA"
	rdbmsInfo.Endpoint = "NA"
	rdbmsInfo.VpcIID = irs.IID{SystemId: "NA"}
	rdbmsInfo.DeletionProtection = false
	rdbmsInfo.PublicAccess = false

	// KeyValueList
	rdbmsInfo.KeyValueList = irs.StructToKeyValueList(inst)

	return rdbmsInfo
}

func convertIBMStateToRDBMSStatus(state string) irs.RDBMSStatus {
	switch state {
	case "active":
		return irs.RDBMSAvailable
	case "provisioning":
		return irs.RDBMSCreating
	case "inactive", "removed":
		return irs.RDBMSStopped
	case "failed":
		return irs.RDBMSError
	default:
		return irs.RDBMSError
	}
}

func (handler *IbmRDBMSHandler) waitForResourceInstanceState(instanceId string, targetState string, timeoutSec int) error {
	for elapsed := 0; elapsed < timeoutSec; elapsed += 30 {
		getOpts := &resourcecontrollerv2.GetResourceInstanceOptions{
			ID: &instanceId,
		}

		result, _, err := handler.ResourceController.GetResourceInstance(getOpts)
		if err != nil {
			return err
		}

		if result != nil && result.State != nil && *result.State == targetState {
			return nil
		}

		time.Sleep(30 * time.Second)
	}
	return fmt.Errorf("timeout waiting for instance %s to reach state %s", instanceId, targetState)
}
