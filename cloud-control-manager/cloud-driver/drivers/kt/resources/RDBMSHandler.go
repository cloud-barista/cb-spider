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

	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/db/v1/datastores"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/db/v1/instances"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/pagination"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type KTVpcRDBMSHandler struct {
	RegionInfo idrv.RegionInfo
	DBClient   *ktvpcsdk.ServiceClient
}

// GetMetaInfo returns metadata about KT Cloud Trove capabilities.
func (handler *KTVpcRDBMSHandler) GetMetaInfo() (irs.RDBMSMetaInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetMetaInfo()")
	callLogInfo := getCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()

	supportedEngines := make(map[string][]string)

	// Query available datastores dynamically from Trove
	allPages, err := datastores.List(handler.DBClient).AllPages()
	if err != nil {
		cblogger.Warn(fmt.Sprintf("Failed to list datastores from KT Trove, using defaults: %v", err))
		supportedEngines["mysql"] = []string{"5.7", "8.0"}
	} else {
		dsList, err := datastores.ExtractDatastores(allPages)
		if err != nil {
			cblogger.Warn(fmt.Sprintf("Failed to extract datastores: %v", err))
		} else {
			for _, ds := range dsList {
				dsName := strings.ToLower(ds.Name)
				if dsName == "mysql" || dsName == "mariadb" || dsName == "postgresql" {
					var versions []string
					for _, v := range ds.Versions {
						versions = append(versions, v.Name)
					}
					supportedEngines[dsName] = versions
				}
			}
		}
	}

	metaInfo := irs.RDBMSMetaInfo{
		SupportedEngines: supportedEngines,

		SupportsHighAvailability:   false, // KT Trove HA is deployment-specific
		SupportsBackup:             true,
		SupportsPublicAccess:       false,
		SupportsDeletionProtection: false,
		SupportsEncryption:         false,

		StorageSizeRange: irs.StorageSizeRange{
			Min: 1,
			Max: 300,
		},
	}

	loggingInfo(callLogInfo, start)
	return metaInfo, nil
}

// ListIID returns a list of RDBMS instance IIDs.
func (handler *KTVpcRDBMSHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListIID()")
	callLogInfo := getCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "ListIID", "instances.List()")
	start := call.Start()

	var iidList []*irs.IID

	err := instances.List(handler.DBClient).EachPage(func(page pagination.Page) (bool, error) {
		instanceList, err := instances.ExtractInstances(page)
		if err != nil {
			return false, err
		}
		for _, inst := range instanceList {
			iid := &irs.IID{
				NameId:   inst.Name,
				SystemId: inst.ID,
			}
			iidList = append(iidList, iid)
		}
		return true, nil
	})

	if err != nil {
		newErr := fmt.Errorf("Failed to list RDBMS IIDs. [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	loggingInfo(callLogInfo, start)
	return iidList, nil
}

// CreateRDBMS creates a new KT Cloud Trove database instance.
func (handler *KTVpcRDBMSHandler) CreateRDBMS(rdbmsReqInfo irs.RDBMSInfo) (irs.RDBMSInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateRDBMS()")
	callLogInfo := getCallLogScheme(handler.RegionInfo.Region, call.RDBMS, rdbmsReqInfo.IId.NameId, "instances.Create()")
	start := call.Start()

	if rdbmsReqInfo.IId.NameId == "" {
		return irs.RDBMSInfo{}, errors.New("RDBMS instance name is required")
	}
	if rdbmsReqInfo.DBEngine == "" {
		return irs.RDBMSInfo{}, errors.New("DBEngine is required")
	}
	if rdbmsReqInfo.DBEngineVersion == "" {
		return irs.RDBMSInfo{}, errors.New("DBEngineVersion is required")
	}
	if rdbmsReqInfo.DBInstanceSpec == "" {
		return irs.RDBMSInfo{}, errors.New("DBInstanceSpec (FlavorRef) is required")
	}
	if rdbmsReqInfo.StorageSize == "" {
		return irs.RDBMSInfo{}, errors.New("StorageSize is required")
	}

	storageSize, err := strconv.Atoi(rdbmsReqInfo.StorageSize)
	if err != nil {
		return irs.RDBMSInfo{}, fmt.Errorf("invalid StorageSize '%s': %w", rdbmsReqInfo.StorageSize, err)
	}

	engineType := strings.ToLower(rdbmsReqInfo.DBEngine)

	createOpts := instances.CreateOpts{
		Name:      rdbmsReqInfo.IId.NameId,
		FlavorRef: rdbmsReqInfo.DBInstanceSpec,
		Size:      storageSize,
		Datastore: &instances.DatastoreOpts{
			Type:    engineType,
			Version: rdbmsReqInfo.DBEngineVersion,
		},
	}

	// Set network if SubnetIIDs provided
	if len(rdbmsReqInfo.SubnetIIDs) > 0 {
		var networks []instances.NetworkOpts
		for _, subnetIID := range rdbmsReqInfo.SubnetIIDs {
			netID := subnetIID.SystemId
			if netID == "" {
				netID = subnetIID.NameId
			}
			networks = append(networks, instances.NetworkOpts{
				UUID: netID,
			})
		}
		createOpts.Networks = networks
	}

	result := instances.Create(handler.DBClient, createOpts)
	createdInstance, err := result.Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to create RDBMS instance. [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.RDBMSInfo{}, newErr
	}

	loggingInfo(callLogInfo, start)
	return convertKtInstanceToRDBMSInfo(createdInstance), nil
}

// ListRDBMS returns a list of all RDBMS instances.
func (handler *KTVpcRDBMSHandler) ListRDBMS() ([]*irs.RDBMSInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListRDBMS()")
	callLogInfo := getCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "ListRDBMS", "instances.List()")
	start := call.Start()

	var rdbmsList []*irs.RDBMSInfo

	err := instances.List(handler.DBClient).EachPage(func(page pagination.Page) (bool, error) {
		instanceList, err := instances.ExtractInstances(page)
		if err != nil {
			return false, err
		}
		for _, inst := range instanceList {
			info := convertKtInstanceToRDBMSInfo(&inst)
			rdbmsList = append(rdbmsList, &info)
		}
		return true, nil
	})

	if err != nil {
		newErr := fmt.Errorf("Failed to list RDBMS instances. [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	loggingInfo(callLogInfo, start)
	return rdbmsList, nil
}

// GetRDBMS retrieves a specific RDBMS instance by IID.
func (handler *KTVpcRDBMSHandler) GetRDBMS(rdbmsIID irs.IID) (irs.RDBMSInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetRDBMS()")

	instanceID := rdbmsIID.SystemId
	if instanceID == "" {
		instanceID = rdbmsIID.NameId
	}
	callLogInfo := getCallLogScheme(handler.RegionInfo.Region, call.RDBMS, instanceID, "instances.Get()")
	start := call.Start()

	// If only NameId is provided, find by name
	if rdbmsIID.SystemId == "" {
		foundID, err := handler.findInstanceIDByName(rdbmsIID.NameId)
		if err != nil {
			loggingError(callLogInfo, err)
			return irs.RDBMSInfo{}, err
		}
		instanceID = foundID
	}

	result := instances.Get(handler.DBClient, instanceID)
	inst, err := result.Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to get RDBMS instance '%s'. [%v]", instanceID, err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.RDBMSInfo{}, newErr
	}

	loggingInfo(callLogInfo, start)
	return convertKtInstanceToRDBMSInfo(inst), nil
}

// DeleteRDBMS deletes a Trove database instance.
func (handler *KTVpcRDBMSHandler) DeleteRDBMS(rdbmsIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called DeleteRDBMS()")

	instanceID := rdbmsIID.SystemId
	if instanceID == "" {
		instanceID = rdbmsIID.NameId
	}
	callLogInfo := getCallLogScheme(handler.RegionInfo.Region, call.RDBMS, instanceID, "instances.Delete()")
	start := call.Start()

	// If only NameId is provided, find by name
	if rdbmsIID.SystemId == "" {
		foundID, err := handler.findInstanceIDByName(rdbmsIID.NameId)
		if err != nil {
			loggingError(callLogInfo, err)
			return false, err
		}
		instanceID = foundID
	}

	result := instances.Delete(handler.DBClient, instanceID)
	if result.Err != nil {
		newErr := fmt.Errorf("Failed to delete RDBMS instance '%s'. [%v]", instanceID, result.Err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	loggingInfo(callLogInfo, start)
	return true, nil
}

// ---- Helper functions ----

func (handler *KTVpcRDBMSHandler) findInstanceIDByName(name string) (string, error) {
	var foundID string

	err := instances.List(handler.DBClient).EachPage(func(page pagination.Page) (bool, error) {
		instanceList, err := instances.ExtractInstances(page)
		if err != nil {
			return false, err
		}
		for _, inst := range instanceList {
			if inst.Name == name {
				foundID = inst.ID
				return false, nil
			}
		}
		return true, nil
	})

	if err != nil {
		return "", fmt.Errorf("Failed to find RDBMS instance by name '%s'. [%v]", name, err)
	}
	if foundID == "" {
		return "", fmt.Errorf("RDBMS instance with name '%s' not found", name)
	}
	return foundID, nil
}

func convertKtInstanceToRDBMSInfo(inst *instances.Instance) irs.RDBMSInfo {
	endpoint := "NA"
	if inst.Hostname != "" {
		endpoint = inst.Hostname
	} else if len(inst.IP) > 0 {
		endpoint = inst.IP[0]
	} else if len(inst.Addresses) > 0 {
		endpoint = inst.Addresses[0].Address
	}

	port := "NA"
	engineType := strings.ToLower(inst.Datastore.Type)
	switch {
	case strings.Contains(engineType, "mysql") || strings.Contains(engineType, "mariadb"):
		port = "3306"
	case strings.Contains(engineType, "postgresql"):
		port = "5432"
	}

	return irs.RDBMSInfo{
		IId: irs.IID{
			NameId:   inst.Name,
			SystemId: inst.ID,
		},
		VpcIID: irs.IID{NameId: "NA", SystemId: "NA"},

		DBEngine:        inst.Datastore.Type,
		DBEngineVersion: inst.Datastore.Version,
		DBInstanceSpec:  inst.Flavor.ID,
		DBInstanceType:  "Primary",

		StorageType: "NA",
		StorageSize: strconv.Itoa(inst.Volume.Size),

		Port:     port,
		Endpoint: endpoint,

		MasterUserName: "NA",
		DatabaseName:   "NA",

		HighAvailability: false,
		ReplicationType:  "NA",

		BackupRetentionDays: 0,
		BackupTime:          "NA",

		PublicAccess:       false,
		Encryption:         false,
		DeletionProtection: false,

		Status:      convertKtTroveStatusToRDBMSStatus(inst.Status),
		CreatedTime: inst.Created,

		KeyValueList: []irs.KeyValue{
			{Key: "Hostname", Value: inst.Hostname},
			{Key: "VolumeUsed", Value: fmt.Sprintf("%.2f", inst.Volume.Used)},
		},
	}
}

func convertKtTroveStatusToRDBMSStatus(status string) irs.RDBMSStatus {
	switch strings.ToUpper(status) {
	case "BUILD", "BACKUP", "RESTART_REQUIRED":
		return irs.RDBMSCreating
	case "ACTIVE":
		return irs.RDBMSAvailable
	case "SHUTDOWN":
		return irs.RDBMSStopped
	case "DELETED":
		return irs.RDBMSDeleting
	case "ERROR", "FAILED":
		return irs.RDBMSError
	default:
		return irs.RDBMSError
	}
}
