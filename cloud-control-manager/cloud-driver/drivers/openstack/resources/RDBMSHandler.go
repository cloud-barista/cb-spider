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
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/db/v1/datastores"
	"github.com/gophercloud/gophercloud/v2/openstack/db/v1/instances"
	"github.com/gophercloud/gophercloud/v2/pagination"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type OpenStackRDBMSHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	DBClient       *gophercloud.ServiceClient
}

// GetMetaInfo returns metadata about OpenStack Trove capabilities.
func (handler *OpenStackRDBMSHandler) GetMetaInfo() (irs.RDBMSMetaInfo, error) {
	cblogger.Debug("OpenStack Trove GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(handler.CredentialInfo.IdentityEndpoint, call.RDBMS, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()

	supportedEngines := make(map[string][]string)

	// Query available datastores dynamically from Trove
	allPages, err := datastores.List(handler.DBClient).AllPages(context.TODO())
	if err != nil {
		cblogger.Warn(fmt.Sprintf("Failed to list datastores from Trove, using static defaults: %v", err))
		// Fallback to common Trove defaults
		supportedEngines["mysql"] = []string{"5.7", "8.0"}
		supportedEngines["mariadb"] = []string{"10.4", "10.5", "10.6"}
		supportedEngines["postgresql"] = []string{"13", "14", "15", "16"}
	} else {
		dsList, err := datastores.ExtractDatastores(allPages)
		if err != nil {
			cblogger.Warn(fmt.Sprintf("Failed to extract datastores: %v", err))
		} else {
			for _, ds := range dsList {
				dsName := strings.ToLower(ds.Name)
				// Filter to supported RDBMS types
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

		SupportsHighAvailability:   false, // Trove replication is separate from HA; varies by deployment
		SupportsBackup:             true,  // Trove supports backup API
		SupportsPublicAccess:       false, // Trove instances are typically private network only
		SupportsDeletionProtection: false, // Trove does not provide deletion protection
		SupportsEncryption:         false, // Trove does not expose storage encryption option

		StorageSizeRange: irs.StorageSizeRange{
			Min: 1,
			Max: 300,
		},
	}

	LoggingInfo(hiscallInfo, start)
	return metaInfo, nil
}

// ListIID returns a list of RDBMS instance IIDs.
func (handler *OpenStackRDBMSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(handler.CredentialInfo.IdentityEndpoint, call.RDBMS, "ListIID", "instances.List()")
	start := call.Start()

	var iidList []*irs.IID

	err := instances.List(handler.DBClient).EachPage(context.TODO(), func(_ context.Context, page pagination.Page) (bool, error) {
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
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, fmt.Errorf("failed to list RDBMS IIDs: %w", err)
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

// CreateRDBMS creates a new Trove database instance.
func (handler *OpenStackRDBMSHandler) CreateRDBMS(rdbmsReqInfo irs.RDBMSInfo) (irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.CredentialInfo.IdentityEndpoint, call.RDBMS, rdbmsReqInfo.IId.NameId, "instances.Create()")
	start := call.Start()

	// Validate required fields
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

	// Map engine name to Trove datastore type
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

	if rdbmsReqInfo.StorageType != "" {
		createOpts.VolumeType = rdbmsReqInfo.StorageType
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

	// Set availability zone if provided in Region
	if handler.Region.Zone != "" {
		createOpts.AvailabilityZone = handler.Region.Zone
	}

	result := instances.Create(context.TODO(), handler.DBClient, createOpts)
	createdInstance, err := result.Extract()
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, fmt.Errorf("failed to create RDBMS instance: %w", err)
	}

	LoggingInfo(hiscallInfo, start)

	rdbmsInfo := handler.convertInstanceToRDBMSInfo(createdInstance)
	return rdbmsInfo, nil
}

// ListRDBMS returns a list of all RDBMS instances.
func (handler *OpenStackRDBMSHandler) ListRDBMS() ([]*irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.CredentialInfo.IdentityEndpoint, call.RDBMS, "ListRDBMS", "instances.List()")
	start := call.Start()

	var rdbmsList []*irs.RDBMSInfo

	err := instances.List(handler.DBClient).EachPage(context.TODO(), func(_ context.Context, page pagination.Page) (bool, error) {
		instanceList, err := instances.ExtractInstances(page)
		if err != nil {
			return false, err
		}
		for _, inst := range instanceList {
			info := handler.convertInstanceToRDBMSInfo(&inst)
			rdbmsList = append(rdbmsList, &info)
		}
		return true, nil
	})

	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, fmt.Errorf("failed to list RDBMS instances: %w", err)
	}

	LoggingInfo(hiscallInfo, start)
	return rdbmsList, nil
}

// GetRDBMS retrieves a specific RDBMS instance by IID.
func (handler *OpenStackRDBMSHandler) GetRDBMS(rdbmsIID irs.IID) (irs.RDBMSInfo, error) {
	instanceID := rdbmsIID.SystemId
	if instanceID == "" {
		instanceID = rdbmsIID.NameId
	}

	hiscallInfo := GetCallLogScheme(handler.CredentialInfo.IdentityEndpoint, call.RDBMS, instanceID, "instances.Get()")
	start := call.Start()

	// If only NameId is provided, find by name
	if rdbmsIID.SystemId == "" {
		foundID, err := handler.findInstanceIDByName(rdbmsIID.NameId)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.RDBMSInfo{}, err
		}
		instanceID = foundID
	}

	result := instances.Get(context.TODO(), handler.DBClient, instanceID)
	inst, err := result.Extract()
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, fmt.Errorf("failed to get RDBMS instance '%s': %w", instanceID, err)
	}

	LoggingInfo(hiscallInfo, start)

	rdbmsInfo := handler.convertInstanceToRDBMSInfo(inst)
	return rdbmsInfo, nil
}

// DeleteRDBMS deletes a Trove database instance.
func (handler *OpenStackRDBMSHandler) DeleteRDBMS(rdbmsIID irs.IID) (bool, error) {
	instanceID := rdbmsIID.SystemId
	if instanceID == "" {
		instanceID = rdbmsIID.NameId
	}

	hiscallInfo := GetCallLogScheme(handler.CredentialInfo.IdentityEndpoint, call.RDBMS, instanceID, "instances.Delete()")
	start := call.Start()

	// If only NameId is provided, find by name
	if rdbmsIID.SystemId == "" {
		foundID, err := handler.findInstanceIDByName(rdbmsIID.NameId)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
		instanceID = foundID
	}

	result := instances.Delete(context.TODO(), handler.DBClient, instanceID)
	if result.Err != nil {
		cblogger.Error(result.Err)
		LoggingError(hiscallInfo, result.Err)
		return false, fmt.Errorf("failed to delete RDBMS instance '%s': %w", instanceID, result.Err)
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// ---- Helper functions ----

// findInstanceIDByName finds a Trove instance's system ID by its name.
func (handler *OpenStackRDBMSHandler) findInstanceIDByName(name string) (string, error) {
	var foundID string

	err := instances.List(handler.DBClient).EachPage(context.TODO(), func(_ context.Context, page pagination.Page) (bool, error) {
		instanceList, err := instances.ExtractInstances(page)
		if err != nil {
			return false, err
		}
		for _, inst := range instanceList {
			if inst.Name == name {
				foundID = inst.ID
				return false, nil // stop iteration
			}
		}
		return true, nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to find RDBMS instance by name '%s': %w", name, err)
	}
	if foundID == "" {
		return "", fmt.Errorf("RDBMS instance with name '%s' not found", name)
	}
	return foundID, nil
}

// convertInstanceToRDBMSInfo converts a Trove Instance to RDBMSInfo.
func (handler *OpenStackRDBMSHandler) convertInstanceToRDBMSInfo(inst *instances.Instance) irs.RDBMSInfo {
	// Extract IP address
	endpoint := "NA"
	if inst.Hostname != "" {
		endpoint = inst.Hostname
	} else if len(inst.IP) > 0 {
		endpoint = inst.IP[0]
	} else if len(inst.Addresses) > 0 {
		endpoint = inst.Addresses[0].Address
	}

	// Determine default port based on engine type
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

		MasterUserName: "NA", // Trove uses root user managed via EnableRootUser API
		DatabaseName:   "NA", // Not exposed in instance info

		HighAvailability: false, // Trove HA is deployment-specific
		ReplicationType:  "NA",

		BackupRetentionDays: 0,
		BackupTime:          "NA",

		PublicAccess:       false,
		Encryption:         false, // Trove does not expose encryption status
		DeletionProtection: false, // Trove does not support deletion protection

		Status:      convertTroveStatusToRDBMSStatus(inst.Status),
		CreatedTime: inst.Created,

		KeyValueList: []irs.KeyValue{
			{Key: "Hostname", Value: inst.Hostname},
			{Key: "VolumeUsed", Value: fmt.Sprintf("%.2f", inst.Volume.Used)},
		},
	}
}

// convertTroveStatusToRDBMSStatus maps Trove status strings to RDBMSStatus.
func convertTroveStatusToRDBMSStatus(status string) irs.RDBMSStatus {
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
