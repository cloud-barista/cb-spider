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
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/limits"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/quotasets"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/db/v1/databases"
	"github.com/gophercloud/gophercloud/v2/openstack/db/v1/datastores"
	"github.com/gophercloud/gophercloud/v2/openstack/db/v1/instances"
	"github.com/gophercloud/gophercloud/v2/openstack/db/v1/users"
	"github.com/gophercloud/gophercloud/v2/pagination"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type OpenStackRDBMSHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	DBClient       *gophercloud.ServiceClient
	VolumeClient   *gophercloud.ServiceClient // for Cinder volume type and quota metadata
	ComputeClient  *gophercloud.ServiceClient // for flavor name→UUID resolution
	NetworkClient  *gophercloud.ServiceClient // for floating IP management
}

// GetMetaInfo returns metadata about OpenStack Trove capabilities.
func (handler *OpenStackRDBMSHandler) GetMetaInfo(dbEngine string) (irs.RDBMSMetaInfo, error) {
	cblogger.Debug("OpenStack Trove GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(handler.CredentialInfo.IdentityEndpoint, call.RDBMS, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return irs.RDBMSMetaInfo{}, err
	}

	supportedEngines, err := handler.fetchSupportedEngines()
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}

	storageTypeOptions, err := handler.fetchStorageTypeOptions(supportedEngines)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}

	storageSizeRange, err := handler.fetchStorageSizeRange()
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}

	metaInfo, err := irs.BuildRDBMSMetaInfo(requestedEngine, supportedEngines, storageTypeOptions, storageSizeRange, false, true, true, false, false)
	if err != nil {
		return irs.RDBMSMetaInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return metaInfo, nil
}

func (handler *OpenStackRDBMSHandler) fetchSupportedEngines() (map[string][]string, error) {
	if handler.DBClient == nil {
		return nil, errors.New("OpenStack Trove DB client is not initialized")
	}

	allPages, err := datastores.List(handler.DBClient).AllPages(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to list Trove datastores: %w", err)
	}

	dsList, err := datastores.ExtractDatastores(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract Trove datastores: %w", err)
	}

	supportedEngines := make(map[string][]string)
	for _, ds := range dsList {
		dsName := strings.ToLower(ds.Name)
		if dsName != "mysql" && dsName != "mariadb" && dsName != "postgresql" {
			continue
		}

		versions := make([]string, 0, len(ds.Versions))
		for _, v := range ds.Versions {
			if v.Name != "" {
				versions = append(versions, v.Name)
			}
		}
		if len(versions) == 0 {
			return nil, fmt.Errorf("Trove datastore %s returned no versions", dsName)
		}
		sort.Strings(versions)
		supportedEngines[dsName] = versions
	}

	if len(supportedEngines) == 0 {
		return nil, errors.New("Trove returned no supported RDBMS datastores")
	}

	return supportedEngines, nil
}

func (handler *OpenStackRDBMSHandler) fetchStorageTypeOptions(supportedEngines map[string][]string) (map[string][]string, error) {
	if handler.VolumeClient == nil {
		return nil, errors.New("OpenStack Cinder volume client is not initialized")
	}

	allPages, err := volumetypes.List(handler.VolumeClient, volumetypes.ListOpts{}).AllPages(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to list Cinder volume types: %w", err)
	}

	volumeTypeList, err := volumetypes.ExtractVolumeTypes(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract Cinder volume types: %w", err)
	}

	storageTypes := make([]string, 0, len(volumeTypeList))
	for _, volumeType := range volumeTypeList {
		if volumeType.Name != "" {
			storageTypes = append(storageTypes, volumeType.Name)
		}
	}
	if len(storageTypes) == 0 {
		return nil, errors.New("Cinder returned no volume types")
	}
	sort.Strings(storageTypes)

	storageTypeOptions := make(map[string][]string, len(supportedEngines))
	for engine := range supportedEngines {
		engineStorageTypes := make([]string, len(storageTypes))
		copy(engineStorageTypes, storageTypes)
		storageTypeOptions[engine] = engineStorageTypes
	}

	return storageTypeOptions, nil
}

func (handler *OpenStackRDBMSHandler) fetchStorageSizeRange() (irs.StorageSizeRange, error) {
	if handler.VolumeClient == nil {
		return irs.StorageSizeRange{}, errors.New("OpenStack Cinder volume client is not initialized")
	}
	if handler.CredentialInfo.ProjectID == "" {
		return irs.StorageSizeRange{}, errors.New("OpenStack project ID is required to query Cinder quota")
	}

	quotaSet, err := quotasets.Get(context.TODO(), handler.VolumeClient, handler.CredentialInfo.ProjectID).Extract()
	if err != nil {
		return irs.StorageSizeRange{}, fmt.Errorf("failed to get Cinder quota set: %w", err)
	}

	cinderLimits, err := limits.Get(context.TODO(), handler.VolumeClient).Extract()
	if err != nil {
		return irs.StorageSizeRange{}, fmt.Errorf("failed to get Cinder limits: %w", err)
	}

	maxSize := quotaSet.PerVolumeGigabytes
	if maxSize <= 0 {
		maxSize = quotaSet.Gigabytes
	}
	if cinderLimits.Absolute.MaxTotalVolumeGigabytes > 0 && (maxSize <= 0 || cinderLimits.Absolute.MaxTotalVolumeGigabytes < maxSize) {
		maxSize = cinderLimits.Absolute.MaxTotalVolumeGigabytes
	}
	if quotaSet.PerVolumeGigabytes == -1 && quotaSet.Gigabytes == -1 && cinderLimits.Absolute.MaxTotalVolumeGigabytes == -1 {
		return irs.StorageSizeRange{Min: 1, Max: -1}, nil
	}
	if maxSize <= 0 {
		return irs.StorageSizeRange{}, fmt.Errorf("Cinder returned invalid storage size limit: perVolume=%d, gigabytes=%d, maxTotal=%d", quotaSet.PerVolumeGigabytes, quotaSet.Gigabytes, cinderLimits.Absolute.MaxTotalVolumeGigabytes)
	}

	return irs.StorageSizeRange{Min: 1, Max: int64(maxSize)}, nil
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

// troveCreateOpts extends instances.CreateOpts with Trove access settings
// (is_public, allowed_cidrs) which are not yet supported by gophercloud.
type troveCreateOpts struct {
	instances.CreateOpts
	IsPublic     bool
	AllowedCIDRs []string
}

func (opts troveCreateOpts) ToInstanceCreateMap() (map[string]any, error) {
	m, err := opts.CreateOpts.ToInstanceCreateMap()
	if err != nil {
		return nil, err
	}
	if opts.IsPublic {
		access := map[string]any{"is_public": true}
		if len(opts.AllowedCIDRs) > 0 {
			access["allowed_cidrs"] = opts.AllowedCIDRs
		}
		m["instance"].(map[string]any)["access"] = access
	}
	return m, nil
}

// CreateRDBMS creates a new Trove database instance.
// DBInstanceSpec accepts either a Nova flavor UUID or a flavor name (e.g.
// "m1.small"). In DevStack, Trove uses the same Nova flavor catalog so the
// names and UUIDs are identical to the VM spec list.
// After the instance reaches ACTIVE status, EnableRootUser is called and, when
// MasterUserName/MasterUserPassword are provided for a non-root user, that user
// is created via the Trove users API.
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
		return irs.RDBMSInfo{}, errors.New("DBInstanceSpec (flavor name or UUID) is required")
	}
	if rdbmsReqInfo.StorageSize == "" {
		return irs.RDBMSInfo{}, errors.New("StorageSize is required")
	}

	storageSize, err := strconv.Atoi(rdbmsReqInfo.StorageSize)
	if err != nil {
		return irs.RDBMSInfo{}, fmt.Errorf("invalid StorageSize '%s': %w", rdbmsReqInfo.StorageSize, err)
	}

	// Resolve flavor name → UUID (Trove shares Nova flavors in DevStack)
	flavorRef, err := handler.resolveFlavorRef(rdbmsReqInfo.DBInstanceSpec)
	if err != nil {
		return irs.RDBMSInfo{}, err
	}

	// Map engine name to Trove datastore type
	engineType := strings.ToLower(rdbmsReqInfo.DBEngine)

	createOpts := troveCreateOpts{
		CreateOpts: instances.CreateOpts{
			Name:      rdbmsReqInfo.IId.NameId,
			FlavorRef: flavorRef,
			Size:      storageSize,
			Datastore: &instances.DatastoreOpts{
				Type:    engineType,
				Version: rdbmsReqInfo.DBEngineVersion,
			},
		},
		IsPublic: rdbmsReqInfo.PublicAccess,
	}

	if rdbmsReqInfo.StorageType != "" {
		createOpts.VolumeType = rdbmsReqInfo.StorageType
	}

	// Network: Trove needs a neutron *network* UUID (= VPC SystemId), not subnet IDs.
	if rdbmsReqInfo.VpcIID.SystemId != "" {
		createOpts.Networks = []instances.NetworkOpts{
			{UUID: rdbmsReqInfo.VpcIID.SystemId},
		}
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

	instanceID := createdInstance.ID
	cblogger.Infof("Trove instance %s created – waiting for ACTIVE status…", instanceID)

	// Wait up to 10 minutes for the instance to become ACTIVE
	if waitErr := handler.waitForTroveActive(instanceID, 600); waitErr != nil {
		cblogger.Error(waitErr)
		LoggingError(hiscallInfo, waitErr)
		return irs.RDBMSInfo{}, fmt.Errorf("RDBMS instance %s did not reach ACTIVE: %w", instanceID, waitErr)
	}

	// Enable root and create the requested DB user
	effectiveUser, rootPwd, userErr := handler.enableAndConfigureUser(
		instanceID,
		rdbmsReqInfo.MasterUserName,
		rdbmsReqInfo.MasterUserPassword,
	)
	if userErr != nil {
		cblogger.Error(userErr)
		LoggingError(hiscallInfo, userErr)
		return irs.RDBMSInfo{}, userErr
	}

	LoggingInfo(hiscallInfo, start)

	// Re-fetch to get the latest status/endpoint after ACTIVE
	freshInst, err := instances.Get(context.TODO(), handler.DBClient, instanceID).Extract()
	if err != nil {
		cblogger.Warnf("failed to re-fetch instance %s after creation: %v", instanceID, err)
		freshInst = createdInstance
	}

	rdbmsInfo, err := handler.convertInstanceToRDBMSInfo(freshInst)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, err
	}
	rdbmsInfo.MasterUserName = effectiveUser

	// Grant global admin privileges to the non-root master user so it can
	// create/drop databases (Trove's user API only grants db-specific access).
	if effectiveUser != "root" {
		if grantErr := grantAdminPrivileges(rdbmsInfo.Endpoint, rdbmsInfo.Port, rootPwd, effectiveUser); grantErr != nil {
			cblogger.Warnf("CreateRDBMS: could not grant admin privileges to '%s': %v", effectiveUser, grantErr)
		} else {
			cblogger.Infof("CreateRDBMS: granted ALL PRIVILEGES ON *.* to '%s'", effectiveUser)
		}
	}

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
			info, err := handler.convertInstanceToRDBMSInfo(&inst)
			if err != nil {
				return false, err
			}
			info.MasterUserName = handler.queryMasterUserName(inst.ID)
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

	rdbmsInfo, err := handler.convertInstanceToRDBMSInfo(inst)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, err
	}
	rdbmsInfo.MasterUserName = handler.queryMasterUserName(instanceID)
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

	// Trove automatically releases the floating IP for is_public instances on deletion.
	result := instances.Delete(context.TODO(), handler.DBClient, instanceID)
	if result.Err != nil {
		cblogger.Error(result.Err)
		LoggingError(hiscallInfo, result.Err)
		return false, fmt.Errorf("failed to delete RDBMS instance '%s': %w", instanceID, result.Err)
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// resolveFlavorRef resolves a VM spec name (e.g. "m1.small") or UUID to the
// flavor UUID accepted by Trove's FlavorRef. In DevStack, Trove shares the
// same Nova flavor catalog, so the IDs are identical to VM spec IDs.
func (handler *OpenStackRDBMSHandler) resolveFlavorRef(spec string) (string, error) {
	if handler.ComputeClient == nil {
		return "", fmt.Errorf("resolveFlavorRef: ComputeClient is not available; cannot resolve flavor name '%s' to UUID", spec)
	}

	// Look up by name via Nova flavors (same catalog used by Trove in DevStack)
	allPages, err := flavors.ListDetail(handler.ComputeClient, flavors.ListOpts{}).AllPages(context.TODO())
	if err != nil {
		return "", fmt.Errorf("failed to list flavors for spec resolution: %w", err)
	}
	flavorList, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		return "", fmt.Errorf("failed to extract flavors: %w", err)
	}
	for _, f := range flavorList {
		if f.Name == spec {
			return f.ID, nil
		}
	}
	return "", fmt.Errorf("flavor '%s' not found; available: %s",
		spec, joinFlavorNames(flavorList))
}

func (handler *OpenStackRDBMSHandler) resolveFlavorName(flavorID string) (string, error) {
	if flavorID == "" {
		return "", errors.New("flavor ID is empty")
	}
	if handler.ComputeClient == nil {
		return "", fmt.Errorf("resolveFlavorName: ComputeClient is not available; cannot resolve flavor ID '%s' to name", flavorID)
	}

	allPages, err := flavors.ListDetail(handler.ComputeClient, flavors.ListOpts{}).AllPages(context.TODO())
	if err != nil {
		return "", fmt.Errorf("failed to list flavors for flavor name resolution: %w", err)
	}
	flavorList, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		return "", fmt.Errorf("failed to extract flavors: %w", err)
	}
	for _, f := range flavorList {
		if f.ID == flavorID {
			return f.Name, nil
		}
	}
	return "", fmt.Errorf("flavor ID '%s' not found; available: %s", flavorID, joinFlavorNames(flavorList))
}

// joinFlavorNames returns a comma-separated list of flavor names for error messages.
func joinFlavorNames(fl []flavors.Flavor) string {
	names := make([]string, 0, len(fl))
	for _, f := range fl {
		names = append(names, f.Name)
	}
	return strings.Join(names, ", ")
}

// waitForTroveActive polls the Trove instance status every 10 s until it
// reaches ACTIVE (or ERROR / timeout).
func (handler *OpenStackRDBMSHandler) waitForTroveActive(instanceID string, timeoutSec int) error {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		inst, err := instances.Get(context.TODO(), handler.DBClient, instanceID).Extract()
		if err != nil {
			return fmt.Errorf("waitForTroveActive: get instance failed: %w", err)
		}
		switch strings.ToUpper(inst.Status) {
		case "ACTIVE":
			return nil
		case "ERROR", "FAILED":
			return fmt.Errorf("Trove instance %s entered %s status", instanceID, inst.Status)
		}
		cblogger.Infof("Trove instance %s status=%s – waiting…", instanceID, inst.Status)
		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("waitForTroveActive: timed out after %d seconds for instance %s", timeoutSec, instanceID)
}

// enableAndConfigureUser enables the root user via Trove and, when a non-root
// MasterUserName is given, creates that user with the supplied password.
// Returns (effectiveUserName, rootPassword, error). rootPassword is non-empty
// so callers can use it to grant global privileges via a direct MySQL connection.
func (handler *OpenStackRDBMSHandler) enableAndConfigureUser(instanceID, masterUserName, masterUserPassword string) (string, string, error) {
	// Always enable root so callers can log in via root if needed.
	rootResult := instances.EnableRootUser(context.TODO(), handler.DBClient, instanceID)
	rootUser, err := rootResult.Extract()
	if err != nil {
		return "", "", fmt.Errorf("EnableRootUser for instance %s failed: %w", instanceID, err)
	}
	rootPassword := rootUser.Password
	cblogger.Infof("EnableRootUser succeeded for instance %s", instanceID)

	// Create the requested non-root user with the supplied password.
	if masterUserName != "" && strings.ToLower(masterUserName) != "root" && masterUserPassword != "" {
		createOpts := users.BatchCreateOpts{
			users.CreateOpts{
				Name:     masterUserName,
				Password: masterUserPassword,
			},
		}
		if err := users.Create(context.TODO(), handler.DBClient, instanceID, createOpts).ExtractErr(); err != nil {
			return "", "", fmt.Errorf("failed to create DB user '%s' on instance %s: %w", masterUserName, instanceID, err)
		}
		cblogger.Infof("DB user '%s' created on instance %s", masterUserName, instanceID)
		return masterUserName, rootPassword, nil
	}
	return "root", rootPassword, nil
}

// grantAdminPrivileges connects to MySQL as root and grants ALL PRIVILEGES ON *.*
// to the given user so they can create databases and manage the instance.
// Failures are non-fatal – logged as warnings only.
func grantAdminPrivileges(endpoint, port, rootPassword, userName string) error {
	if endpoint == "" || endpoint == "NA" || rootPassword == "" || userName == "" {
		return nil
	}
	if port == "" || port == "NA" {
		port = "3306"
	}
	// DSN: root:<pass>@tcp(<host>:<port>)/?timeout=10s
	dsn := fmt.Sprintf("root:%s@tcp(%s:%s)/?timeout=10s", rootPassword, endpoint, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("grantAdminPrivileges: sql.Open failed: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	grantSQL := fmt.Sprintf("GRANT ALL PRIVILEGES ON *.* TO '%s'@'%%' WITH GRANT OPTION", userName)
	if _, err := db.ExecContext(ctx, grantSQL); err != nil {
		return fmt.Errorf("grantAdminPrivileges: GRANT failed: %w", err)
	}
	if _, err := db.ExecContext(ctx, "FLUSH PRIVILEGES"); err != nil {
		return fmt.Errorf("grantAdminPrivileges: FLUSH PRIVILEGES failed: %w", err)
	}
	return nil
}

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
func (handler *OpenStackRDBMSHandler) convertInstanceToRDBMSInfo(inst *instances.Instance) (irs.RDBMSInfo, error) {
	flavorName, err := handler.resolveFlavorName(inst.Flavor.ID)
	if err != nil {
		return irs.RDBMSInfo{}, err
	}

	// Extract endpoint: prefer public-type address (is_public floating IP), then any address.
	endpoint := "NA"
	publicAccess := false
	for _, addr := range inst.Addresses {
		if strings.ToLower(addr.Type) == "public" {
			endpoint = addr.Address
			publicAccess = true
			break
		}
	}
	if endpoint == "NA" {
		if len(inst.Addresses) > 0 {
			endpoint = inst.Addresses[0].Address
		} else if inst.Hostname != "" {
			endpoint = inst.Hostname
		} else if len(inst.IP) > 0 {
			endpoint = inst.IP[0]
		}
	}

	// Determine default port based on engine type.
	// Trove API does not expose the actual port in the instance response,
	// so standard defaults are used as a best-effort value.
	engineType := strings.ToLower(inst.Datastore.Type)
	port := "NA"
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
		DBInstanceSpec:  flavorName,
		DBInstanceType:  "Primary",

		StorageType: "NA",
		StorageSize: strconv.Itoa(inst.Volume.Size),

		Port:     port,
		Endpoint: endpoint,

		MasterUserName: "NA", // populated by caller via queryMasterUserName()
		DatabaseName:   "NA", // Not exposed in instance info

		HighAvailability: false, // Trove HA is deployment-specific
		ReplicationType:  "NA",

		BackupRetentionDays: 0,
		BackupTime:          "NA",

		PublicAccess:       publicAccess,
		Encryption:         false, // Trove does not expose encryption status
		DeletionProtection: false, // Trove does not support deletion protection

		Status:      convertTroveStatusToRDBMSStatus(inst.Status),
		CreatedTime: inst.Created,

		KeyValueList: []irs.KeyValue{
			{Key: "Hostname", Value: inst.Hostname},
			{Key: "VolumeUsed", Value: fmt.Sprintf("%.2f", inst.Volume.Used)},
		},
	}, nil
}

// queryMasterUserName queries the Trove users API for the given instance and
// returns the first non-root user name found, or "root" if none exist.
func (handler *OpenStackRDBMSHandler) queryMasterUserName(instanceID string) string {
	var found string
	err := users.List(handler.DBClient, instanceID).EachPage(context.TODO(), func(_ context.Context, page pagination.Page) (bool, error) {
		userList, err := users.ExtractUsers(page)
		if err != nil {
			return false, err
		}
		for _, u := range userList {
			if strings.ToLower(u.Name) != "root" {
				found = u.Name
				return false, nil // stop iteration
			}
		}
		return true, nil
	})
	if err != nil {
		cblogger.Warnf("queryMasterUserName for instance %s: %v", instanceID, err)
	}
	if found == "" {
		return "root"
	}
	return found
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

// ─── rdbmsDatabaseManager interface implementation ───────────────────────────

// CreateDatabase creates a database in an OpenStack Trove instance.
func (handler *OpenStackRDBMSHandler) CreateDatabase(rdbmsSystemId, dbEngine, dbName string) error {
	ctx := context.Background()
	opts := databases.BatchCreateOpts{databases.CreateOpts{Name: dbName}}
	if err := databases.Create(ctx, handler.DBClient, rdbmsSystemId, opts).ExtractErr(); err != nil {
		return fmt.Errorf("OpenStack CreateDatabase: %w", err)
	}
	return nil
}

// ListDatabases lists all user-created databases in an OpenStack Trove instance.
func (handler *OpenStackRDBMSHandler) ListDatabases(rdbmsSystemId, dbEngine string) ([]string, error) {
	allPages, err := databases.List(handler.DBClient, rdbmsSystemId).AllPages(context.Background())
	if err != nil {
		return nil, fmt.Errorf("OpenStack ListDatabases: %w", err)
	}
	dbList, err := databases.ExtractDBs(allPages)
	if err != nil {
		return nil, fmt.Errorf("OpenStack ListDatabases extract: %w", err)
	}
	var names []string
	for _, db := range dbList {
		names = append(names, db.Name)
	}
	return names, nil
}

// DeleteDatabase deletes a database from an OpenStack Trove instance.
func (handler *OpenStackRDBMSHandler) DeleteDatabase(rdbmsSystemId, dbEngine, dbName string) error {
	ctx := context.Background()
	if err := databases.Delete(ctx, handler.DBClient, rdbmsSystemId, dbName).ExtractErr(); err != nil {
		return fmt.Errorf("OpenStack DeleteDatabase: %w", err)
	}
	return nil
}
