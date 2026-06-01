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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/cloud-databases-go-sdk/clouddatabasesv5"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/globaltaggingv1"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
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
	ibmStorageUnitGB    = 1024
	ibmDefaultAdminUser = "admin"
	ibmRDBMSTimeoutSec  = 60 * 60
)

var ibmRDBMSHostFlavorIDs = map[string]bool{
	"multitenant":          true,
	"b3c.4x16.encrypted":   true,
	"b3c.8x32.encrypted":   true,
	"m3c.8x64.encrypted":   true,
	"b3c.16x64.encrypted":  true,
	"b3c.32x128.encrypted": true,
	"m3c.30x240.encrypted": true,
	"bx3d.4x20":            true,
	"bx3d.8x40":            true,
	"mx3d.8x64":            true,
}

type IbmRDBMSHandler struct {
	CredentialInfo     idrv.CredentialInfo
	Region             idrv.RegionInfo
	Ctx                context.Context
	ResourceController *resourcecontrollerv2.ResourceControllerV2
	TaggingService     *globaltaggingv1.GlobalTaggingV1
	CloudDBService     *clouddatabasesv5.CloudDatabasesV5
}

func (handler *IbmRDBMSHandler) GetMetaInfo(dbEngine string) (irs.RDBMSMetaInfo, error) {
	cblogger.Debug("IBM Cloud GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetMetaInfo", "ListDeployables()/GetDefaultScalingGroups()/GlobalCatalog Plan API")
	start := call.Start()
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return irs.RDBMSMetaInfo{}, err
	}
	supportedEngines, err := handler.fetchRDBMSVersions()
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}
	storageTypeOptions, err := handler.fetchRDBMSStorageTypeOptions(requestedEngine)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}
	storageSizeRange, err := handler.fetchRDBMSStorageSizeRange(requestedEngine)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}

	metaInfo, err := irs.BuildRDBMSMetaInfo(requestedEngine, supportedEngines, storageTypeOptions, storageSizeRange, true, true, true, true, true)
	if err != nil {
		return irs.RDBMSMetaInfo{}, err
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return metaInfo, nil
}

func (handler *IbmRDBMSHandler) fetchRDBMSVersions() (map[string][]string, error) {
	validEngines := map[string]bool{
		"mysql":      true,
		"postgresql": true,
	}
	supportedEngines := map[string][]string{}

	resp, _, err := handler.CloudDBService.ListDeployablesWithContext(handler.getContext(), handler.CloudDBService.NewListDeployablesOptions())
	if err != nil {
		return nil, fmt.Errorf("ListDeployables failed: %w", err)
	}
	if resp == nil || len(resp.Deployables) == 0 {
		return nil, errors.New("ListDeployables returned no deployable database metadata")
	}

	for _, deployable := range resp.Deployables {
		if deployable.Type == nil || !validEngines[*deployable.Type] {
			continue
		}
		var versions []string
		for _, version := range deployable.Versions {
			if version.Version == nil || version.Status == nil || *version.Status == clouddatabasesv5.DeployablesVersionsItemStatusDeprecatedConst {
				continue
			}
			versions = append(versions, *version.Version)
		}
		if len(versions) > 0 {
			supportedEngines[*deployable.Type] = versions
		}
	}

	return supportedEngines, nil
}

func (handler *IbmRDBMSHandler) fetchRDBMSStorageTypeOptions(dbEngine string) (map[string][]string, error) {
	serviceID, err := ibmRDBMSServiceID(dbEngine)
	if err != nil {
		return nil, err
	}
	body, err := getIbmPlanInfo(0, 100, serviceID)
	if err != nil {
		return nil, fmt.Errorf("IBM Global Catalog plan API failed for %s: %w", dbEngine, err)
	}

	var planInfo ResourceInfo
	if err := json.Unmarshal(body, &planInfo); err != nil {
		return nil, fmt.Errorf("failed to parse IBM Global Catalog plan response for %s: %w", dbEngine, err)
	}

	storageTypes := make([]string, 0, len(planInfo.Resources))
	seen := map[string]bool{}
	for _, plan := range planInfo.Resources {
		if !containsRegionInGeoTags(plan.GeoTags, handler.Region.Region) {
			continue
		}
		storageType := strings.TrimSpace(plan.Name)
		if storageType == "" {
			storageType = strings.TrimSpace(plan.Id)
		}
		storageType = strings.TrimPrefix(storageType, serviceID+"-")
		if storageType == "" || seen[storageType] {
			continue
		}
		seen[storageType] = true
		storageTypes = append(storageTypes, storageType)
	}
	if len(storageTypes) == 0 {
		return nil, fmt.Errorf("IBM Global Catalog returned no storage type plans for %s in region %s", dbEngine, handler.Region.Region)
	}
	sort.Strings(storageTypes)

	return map[string][]string{dbEngine: storageTypes}, nil
}

func (handler *IbmRDBMSHandler) fetchRDBMSStorageSizeRange(dbEngine string) (irs.StorageSizeRange, error) {
	resp, _, err := handler.CloudDBService.GetDefaultScalingGroupsWithContext(handler.getContext(), handler.CloudDBService.NewGetDefaultScalingGroupsOptions(dbEngine))
	if err != nil {
		return irs.StorageSizeRange{}, fmt.Errorf("GetDefaultScalingGroups failed for %s: %w", dbEngine, err)
	}
	if resp == nil || len(resp.Groups) == 0 {
		return irs.StorageSizeRange{}, fmt.Errorf("GetDefaultScalingGroups returned no scaling groups for %s", dbEngine)
	}

	var minMB, maxMB int64
	for _, group := range resp.Groups {
		if group.ID == nil || *group.ID != clouddatabasesv5.GroupIDMemberConst || group.Disk == nil {
			continue
		}
		if group.Disk.MinimumMb != nil {
			minMB = *group.Disk.MinimumMb
		}
		if group.Disk.MaximumMb != nil {
			maxMB = *group.Disk.MaximumMb
		}
		break
	}
	if minMB <= 0 || maxMB <= 0 {
		return irs.StorageSizeRange{}, fmt.Errorf("GetDefaultScalingGroups returned invalid member disk range for %s: minMB=%d, maxMB=%d", dbEngine, minMB, maxMB)
	}

	return irs.StorageSizeRange{
		Min: minMB / ibmStorageUnitGB,
		Max: maxMB / ibmStorageUnitGB,
	}, nil
}

func ibmRDBMSServiceID(dbEngine string) (string, error) {
	switch dbEngine {
	case "mysql":
		return ibmServiceIDMySQL, nil
	case "postgresql":
		return ibmServiceIDPG, nil
	default:
		return "", fmt.Errorf("unsupported IBM RDBMS engine: %s", dbEngine)
	}
}

func (handler *IbmRDBMSHandler) getContext() context.Context {
	if handler.Ctx != nil {
		return handler.Ctx
	}
	return context.Background()
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
	if err := validateIBMCreateRequest(rdbmsReqInfo); err != nil {
		return irs.RDBMSInfo{}, err
	}

	resourcePlanID, storageType, err := handler.resolveRDBMSResourcePlanID(rdbmsReqInfo.DBEngine, rdbmsReqInfo.StorageType)
	if err != nil {
		return irs.RDBMSInfo{}, err
	}
	if err := handler.validateRDBMSStorageSizeRange(rdbmsReqInfo.DBEngine, rdbmsReqInfo.StorageSize); err != nil {
		return irs.RDBMSInfo{}, err
	}
	resourceGroupID, err := handler.getDefaultResourceGroupId()
	if err != nil {
		return irs.RDBMSInfo{}, fmt.Errorf("failed to get IBM default resource group: %w", err)
	}

	// Build parameters
	params := map[string]interface{}{}
	if rdbmsReqInfo.DBEngineVersion != "" {
		params["version"] = rdbmsReqInfo.DBEngineVersion
	}
	// Note: admin_password is NOT passed as a provisioning parameter because IBM Cloud
	// Databases does not reliably apply it during provisioning. The password is set
	// explicitly via UpdateUser after the instance is ready (see setAdminPassword below).
	if rdbmsReqInfo.HighAvailability {
		params["members"] = 3 // IBM HA uses 3 members
	}
	if rdbmsReqInfo.PublicAccess {
		params["service-endpoints"] = "public"
	}
	if rdbmsReqInfo.StorageSize != "" {
		storageSizeGB, err := strconv.ParseInt(rdbmsReqInfo.StorageSize, 10, 64)
		if err != nil {
			return irs.RDBMSInfo{}, fmt.Errorf("IBM StorageSize must be an integer GB value: %w", err)
		}
		memberCount, err := handler.getInitialMemberCount(rdbmsReqInfo.DBEngine, rdbmsReqInfo.DBInstanceSpec)
		if err != nil {
			return irs.RDBMSInfo{}, err
		}
		params["members_disk_allocation_mb"] = storageSizeGB * ibmStorageUnitGB * memberCount
	}
	if isIBMHostFlavor(rdbmsReqInfo.DBInstanceSpec) {
		params["members_host_flavor"] = rdbmsReqInfo.DBInstanceSpec
	}

	// Target region
	target := handler.Region.Region

	createOpts := &resourcecontrollerv2.CreateResourceInstanceOptions{
		Name:           &rdbmsReqInfo.IId.NameId,
		Target:         &target,
		ResourceGroup:  &resourceGroupID,
		ResourcePlanID: &resourcePlanID,
		Parameters:     params,
	}
	if rdbmsReqInfo.DeletionProtection {
		createOpts.EntityLock = core.BoolPtr(true)
	}

	// Tags
	tags := make([]string, 0, len(rdbmsReqInfo.TagList))
	if len(rdbmsReqInfo.TagList) > 0 {
		for _, tag := range rdbmsReqInfo.TagList {
			tags = append(tags, tag.Key+":"+tag.Value)
		}
		createOpts.Tags = tags
	}

	result, _, err := handler.ResourceController.CreateResourceInstanceWithContext(handler.getContext(), createOpts)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	// Wait for active state
	if result == nil {
		return irs.RDBMSInfo{}, errors.New("CreateResourceInstance returned empty result")
	}
	resourceInstanceID := ""
	if result.GUID != nil {
		resourceInstanceID = *result.GUID
	}
	if resourceInstanceID == "" {
		return irs.RDBMSInfo{}, errors.New("CreateResourceInstance returned no GUID for Resource Controller polling")
	}
	deploymentID := ""
	if result.CRN != nil {
		deploymentID = *result.CRN
	} else if result.ID != nil {
		deploymentID = *result.ID
	}
	if deploymentID == "" {
		return irs.RDBMSInfo{}, errors.New("CreateResourceInstance returned no CRN for Cloud Databases API calls")
	}
	err = handler.waitForResourceInstanceState(resourceInstanceID, "active", ibmRDBMSTimeoutSec)
	if err != nil {
		return irs.RDBMSInfo{}, fmt.Errorf("created IBM RDBMS instance but failed to wait for active state: %w", err)
	}
	if err := handler.waitForCloudDBDeploymentReady(deploymentID, ibmRDBMSTimeoutSec); err != nil {
		return irs.RDBMSInfo{}, fmt.Errorf("created IBM RDBMS instance but Cloud Databases API is not ready: %w", err)
	}
	if rdbmsReqInfo.MasterUserPassword != "" {
		if err := handler.setAdminPassword(deploymentID, rdbmsReqInfo.MasterUserPassword); err != nil {
			return irs.RDBMSInfo{}, fmt.Errorf("IBM RDBMS instance created but failed to set admin password: %w", err)
		}
	}

	info := handler.convertResourceInstanceToRDBMSInfo(result)
	info.DBEngineVersion = rdbmsReqInfo.DBEngineVersion
	info.StorageType = storageType
	info.StorageSize = rdbmsReqInfo.StorageSize
	info.DBInstanceSpec = rdbmsReqInfo.DBInstanceSpec
	info.MasterUserName = rdbmsReqInfo.MasterUserName
	info.HighAvailability = rdbmsReqInfo.HighAvailability
	info.PublicAccess = rdbmsReqInfo.PublicAccess
	info.DeletionProtection = rdbmsReqInfo.DeletionProtection
	info.TagList = rdbmsReqInfo.TagList

	return info, nil
}

func validateIBMCreateRequest(rdbmsReqInfo irs.RDBMSInfo) error {
	if len(rdbmsReqInfo.SubnetIIDs) > 0 {
		return errors.New("IBM Cloud Databases provisioning does not support SubnetNames/SubnetIIDs; service endpoints are selected instead")
	}
	if len(rdbmsReqInfo.SecurityGroupIIDs) > 0 {
		return errors.New("IBM Cloud Databases provisioning does not support SecurityGroupNames/SecurityGroupIIDs; use IBM Cloud Databases allowlist APIs after creation")
	}
	if rdbmsReqInfo.DatabaseName != "" && !isIBMDefaultValue(rdbmsReqInfo.DatabaseName) && rdbmsReqInfo.DatabaseName != "ibmclouddb" {
		return errors.New("IBM Cloud Databases API does not support creating an initial DatabaseName during provisioning")
	}
	if rdbmsReqInfo.BackupRetentionDays > 0 {
		return errors.New("IBM Cloud Databases API does not support setting BackupRetentionDays during provisioning")
	}
	if rdbmsReqInfo.BackupTime != "" && !isIBMDefaultValue(rdbmsReqInfo.BackupTime) {
		return errors.New("IBM Cloud Databases API does not support setting BackupTime during provisioning")
	}
	if rdbmsReqInfo.Port != "" {
		expectedPort := ""
		switch rdbmsReqInfo.DBEngine {
		case "mysql":
			expectedPort = "3306"
		case "postgresql":
			expectedPort = "5432"
		}
		if expectedPort != "" && rdbmsReqInfo.Port != expectedPort {
			return fmt.Errorf("IBM Cloud Databases does not support custom Port during provisioning; %s uses service-managed port %s", rdbmsReqInfo.DBEngine, expectedPort)
		}
	}
	if rdbmsReqInfo.DBInstanceSpec != "" && !isIBMDefaultValue(rdbmsReqInfo.DBInstanceSpec) && !isIBMHostFlavor(rdbmsReqInfo.DBInstanceSpec) {
		return fmt.Errorf("IBM DBInstanceSpec must be an IBM host_flavor id, got %s", rdbmsReqInfo.DBInstanceSpec)
	}
	if rdbmsReqInfo.StorageSize != "" {
		if _, err := strconv.ParseInt(rdbmsReqInfo.StorageSize, 10, 64); err != nil {
			return fmt.Errorf("IBM StorageSize must be an integer GB value: %w", err)
		}
	}
	if rdbmsReqInfo.MasterUserName != "" && rdbmsReqInfo.MasterUserName != ibmDefaultAdminUser {
		return fmt.Errorf("IBM Cloud Databases does not support custom MasterUserName: the admin user is always %q. Set MasterUserName to %q or leave it empty", ibmDefaultAdminUser, ibmDefaultAdminUser)
	}
	if rdbmsReqInfo.MasterUserPassword != "" {
		if err := validateIBMAdminPassword(rdbmsReqInfo.MasterUserPassword); err != nil {
			return fmt.Errorf("IBM Cloud Databases MasterUserPassword validation failed: %w", err)
		}
	}
	return nil
}

// validateIBMAdminPassword validates MasterUserPassword against IBM Cloud Databases
// password requirements for the database (admin) user type, as stated in the
// IBM Cloud Console "Change database admin password" UI:
//   - At least 15 characters long
//   - Must contain at least one number and one letter
//   - Allowed characters: uppercase letters, lowercase letters, numbers, - (hyphen), _ (underscore)
//   - Must not begin with a special character (_ or -)
//
// Max length of 72 is derived from the IBM Terraform Provider (ibm_database resource).
// Reference: https://github.com/IBM-Cloud/terraform-provider-ibm/blob/master/ibm/service/database/resource_ibm_database.go
func validateIBMAdminPassword(password string) error {
	const minLen = 15
	const maxLen = 72

	if len(password) < minLen || len(password) > maxLen {
		return fmt.Errorf("must be between %d and %d characters long (got %d)", minLen, maxLen, len(password))
	}

	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !allowedChars.MatchString(password) {
		return fmt.Errorf("must contain only letters (a-z, A-Z), digits (0-9), underscore (_), or hyphen (-)")
	}

	if password[0] == '_' || password[0] == '-' {
		return fmt.Errorf("must not begin with a special character (_ or -)")
	}

	hasLetter := regexp.MustCompile(`[a-zA-Z]`)
	if !hasLetter.MatchString(password) {
		return fmt.Errorf("must contain at least one letter")
	}

	hasDigit := regexp.MustCompile(`[0-9]`)
	if !hasDigit.MatchString(password) {
		return fmt.Errorf("must contain at least one digit")
	}

	return nil
}

func (handler *IbmRDBMSHandler) resolveRDBMSResourcePlanID(dbEngine string, requestedStorageType string) (string, string, error) {
	serviceID, err := ibmRDBMSServiceID(dbEngine)
	if err != nil {
		return "", "", err
	}
	storageType := strings.TrimSpace(requestedStorageType)
	if storageType == "" || isIBMDefaultValue(storageType) {
		storageType = "standard"
	}
	storageOptions, err := handler.fetchRDBMSStorageTypeOptions(dbEngine)
	if err != nil {
		return "", "", err
	}
	for _, option := range storageOptions[dbEngine] {
		if option == storageType {
			return serviceID + "-" + storageType, storageType, nil
		}
	}
	return "", "", fmt.Errorf("IBM storage type %s is not available for %s in region %s", storageType, dbEngine, handler.Region.Region)
}

func (handler *IbmRDBMSHandler) validateRDBMSStorageSizeRange(dbEngine string, storageSize string) error {
	if storageSize == "" {
		return nil
	}
	storageSizeGB, err := strconv.ParseInt(storageSize, 10, 64)
	if err != nil {
		return fmt.Errorf("IBM StorageSize must be an integer GB value: %w", err)
	}
	storageRange, err := handler.fetchRDBMSStorageSizeRange(dbEngine)
	if err != nil {
		return err
	}
	if storageSizeGB < storageRange.Min || storageSizeGB > storageRange.Max {
		return fmt.Errorf("IBM StorageSize %dGB is outside supported range for %s in %s: %dGB-%dGB", storageSizeGB, dbEngine, handler.Region.Region, storageRange.Min, storageRange.Max)
	}
	return nil
}

func (handler *IbmRDBMSHandler) getInitialMemberCount(dbEngine string, hostFlavor string) (int64, error) {
	group, err := handler.getDefaultMemberScalingGroup(dbEngine, hostFlavor)
	if err != nil {
		return 0, err
	}
	if group.Members != nil && group.Members.MinimumCount != nil && *group.Members.MinimumCount > 0 {
		return *group.Members.MinimumCount, nil
	}
	return 1, nil
}

func (handler *IbmRDBMSHandler) getDefaultMemberScalingGroup(dbEngine string, hostFlavor string) (*clouddatabasesv5.Group, error) {
	options := handler.CloudDBService.NewGetDefaultScalingGroupsOptions(dbEngine)
	if isIBMHostFlavor(hostFlavor) {
		options.SetHostFlavor(hostFlavor)
	}
	resp, _, err := handler.CloudDBService.GetDefaultScalingGroupsWithContext(handler.getContext(), options)
	if err != nil {
		return nil, fmt.Errorf("GetDefaultScalingGroups failed for %s: %w", dbEngine, err)
	}
	if resp == nil || len(resp.Groups) == 0 {
		return nil, fmt.Errorf("GetDefaultScalingGroups returned no scaling groups for %s", dbEngine)
	}
	for _, group := range resp.Groups {
		if group.ID != nil && *group.ID == clouddatabasesv5.GroupIDMemberConst {
			return &group, nil
		}
	}
	return nil, fmt.Errorf("GetDefaultScalingGroups returned no member scaling group for %s", dbEngine)
}

// setAdminPassword sets the admin (database) user password via the IBM Cloud Databases
// UpdateUser API after the instance is provisioned and ready. IBM does not reliably
// apply admin_password passed as a provisioning parameter; the explicit UpdateUser call
// (matching the IBM Terraform Provider approach) is required to actually set the password.
func (handler *IbmRDBMSHandler) setAdminPassword(deploymentID, password string) error {
	deployment, _, err := handler.CloudDBService.GetDeploymentInfoWithContext(
		handler.getContext(),
		handler.CloudDBService.NewGetDeploymentInfoOptions(deploymentID),
	)
	if err != nil || deployment == nil || deployment.Deployment == nil {
		return fmt.Errorf("failed to get deployment info for setting admin password: %w", err)
	}

	adminUser := deployment.Deployment.AdminUsernames["database"]
	if adminUser == "" {
		return fmt.Errorf("IBM Cloud Databases deployment has no database admin user in AdminUsernames")
	}

	updateUserOptions := &clouddatabasesv5.UpdateUserOptions{
		ID:       core.StringPtr(deploymentID),
		UserType: core.StringPtr("database"),
		Username: core.StringPtr(adminUser),
		User:     &clouddatabasesv5.UserUpdatePasswordSetting{Password: core.StringPtr(password)},
	}

	resp, _, err := handler.CloudDBService.UpdateUserWithContext(handler.getContext(), updateUserOptions)
	if err != nil {
		return fmt.Errorf("failed to set admin password via UpdateUser: %w", err)
	}
	if resp == nil || resp.Task == nil {
		return nil
	}
	return handler.waitForCloudDBTask(resp.Task, ibmRDBMSTimeoutSec)
}

func (handler *IbmRDBMSHandler) waitForCloudDBDeploymentReady(instanceID string, timeoutSec int) error {
	for elapsed := 0; elapsed < timeoutSec; elapsed += 30 {
		deployment, _, err := handler.CloudDBService.GetDeploymentInfoWithContext(handler.getContext(), handler.CloudDBService.NewGetDeploymentInfoOptions(instanceID))
		if err != nil || deployment == nil || deployment.Deployment == nil {
			time.Sleep(30 * time.Second)
			continue
		}

		groups, _, err := handler.CloudDBService.ListDeploymentScalingGroupsWithContext(handler.getContext(), handler.CloudDBService.NewListDeploymentScalingGroupsOptions(instanceID))
		if err != nil || groups == nil || len(groups.Groups) == 0 {
			time.Sleep(30 * time.Second)
			continue
		}
		return nil
	}
	return fmt.Errorf("timeout waiting for IBM Cloud Databases deployment %s to become ready", instanceID)
}

func (handler *IbmRDBMSHandler) waitForCloudDBTask(task *clouddatabasesv5.Task, timeoutSec int) error {
	if task == nil || task.ID == nil || *task.ID == "" {
		return errors.New("IBM Cloud Databases task response did not include task id")
	}
	for elapsed := 0; elapsed < timeoutSec; elapsed += 30 {
		result, response, err := handler.CloudDBService.GetTaskWithContext(handler.getContext(), handler.CloudDBService.NewGetTaskOptions(*task.ID))
		if err != nil {
			return err
		}
		if response != nil && response.StatusCode == http.StatusSeeOther {
			return nil
		}
		if result == nil || result.Task == nil {
			return nil
		}
		if result.Task.Status != nil {
			switch *result.Task.Status {
			case clouddatabasesv5.TaskStatusCompletedConst:
				return nil
			case clouddatabasesv5.TaskStatusFailedConst, clouddatabasesv5.TaskStatusExpiredConst:
				return fmt.Errorf("task %s ended with status %s", *task.ID, *result.Task.Status)
			}
		}
		time.Sleep(30 * time.Second)
	}
	return fmt.Errorf("timeout waiting for IBM Cloud Databases task %s", *task.ID)
}

func isIBMDefaultValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "na", "n/a", "default", "standard":
		return true
	default:
		return false
	}
}

func isIBMHostFlavor(value string) bool {
	return ibmRDBMSHostFlavorIDs[strings.TrimSpace(value)]
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
		if err := handler.enrichRDBMSInfoFromCloudDB(&info, &inst); err != nil {
			hiscallInfo.ElapsedTime = call.Elapsed(start)
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		rdbmsList = append(rdbmsList, &info)
	}

	// List PostgreSQL instances
	pgInstances, err := handler.listResourceInstances(ibmServiceIDPG)
	if err != nil {
		cblogger.Warn("Failed to list PostgreSQL instances: ", err)
	} else {
		for _, inst := range pgInstances {
			info := handler.convertResourceInstanceToRDBMSInfo(&inst)
			if err := handler.enrichRDBMSInfoFromCloudDB(&info, &inst); err != nil {
				hiscallInfo.ElapsedTime = call.Elapsed(start)
				LoggingError(hiscallInfo, err)
				return nil, err
			}
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

	info := handler.convertResourceInstanceToRDBMSInfo(result)
	if err := handler.enrichRDBMSInfoFromCloudDB(&info, result); err != nil {
		return irs.RDBMSInfo{}, err
	}
	return info, nil
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
		resourcePlanID := *inst.ResourcePlanID
		switch {
		case strings.HasPrefix(resourcePlanID, ibmServiceIDMySQL+"-"):
			rdbmsInfo.DBEngine = "mysql"
			rdbmsInfo.Port = "3306"
			rdbmsInfo.StorageType = strings.TrimPrefix(resourcePlanID, ibmServiceIDMySQL+"-")
		case strings.HasPrefix(resourcePlanID, ibmServiceIDPG+"-"):
			rdbmsInfo.DBEngine = "postgresql"
			rdbmsInfo.Port = "5432"
			rdbmsInfo.StorageType = strings.TrimPrefix(resourcePlanID, ibmServiceIDPG+"-")
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
	rdbmsInfo.MasterUserName = ibmDefaultAdminUser // IBM default
	rdbmsInfo.DatabaseName = "NA"
	rdbmsInfo.StorageSize = "NA"
	if rdbmsInfo.StorageType == "" {
		rdbmsInfo.StorageType = "standard"
	}
	rdbmsInfo.DBInstanceSpec = "NA"
	rdbmsInfo.BackupTime = "NA"
	rdbmsInfo.ReplicationType = "NA"
	rdbmsInfo.Endpoint = "NA"
	rdbmsInfo.VpcIID = irs.IID{SystemId: "NA"}
	if inst.Locked != nil {
		rdbmsInfo.DeletionProtection = *inst.Locked
	}
	rdbmsInfo.PublicAccess = false

	// KeyValueList
	rdbmsInfo.KeyValueList = irs.StructToKeyValueList(inst)

	return rdbmsInfo
}

func (handler *IbmRDBMSHandler) enrichRDBMSInfoFromCloudDB(info *irs.RDBMSInfo, inst *resourcecontrollerv2.ResourceInstance) error {
	if info == nil || inst == nil || info.Status != irs.RDBMSAvailable {
		return nil
	}

	deploymentID := ibmDeploymentIDFromResourceInstance(inst)
	if deploymentID == "" {
		return errors.New("IBM Resource Controller instance did not include CRN for Cloud Databases API calls")
	}

	deployment, _, err := handler.CloudDBService.GetDeploymentInfoWithContext(handler.getContext(), handler.CloudDBService.NewGetDeploymentInfoOptions(deploymentID))
	if err != nil {
		return fmt.Errorf("failed to get IBM Cloud Databases deployment info for %s: %w", info.IId.NameId, err)
	}
	if deployment != nil && deployment.Deployment != nil {
		if deployment.Deployment.Version != nil {
			info.DBEngineVersion = *deployment.Deployment.Version
		}
		if deployment.Deployment.AdminUsernames != nil && deployment.Deployment.AdminUsernames["database"] != "" {
			info.MasterUserName = deployment.Deployment.AdminUsernames["database"]
		}
		if deployment.Deployment.EnablePublicEndpoints != nil {
			info.PublicAccess = *deployment.Deployment.EnablePublicEndpoints
		}
	}
	tagList, err := handler.getTaggedRDBMSInfo(deploymentID)
	if err != nil {
		return err
	}
	info.TagList = tagList

	groups, _, err := handler.CloudDBService.ListDeploymentScalingGroupsWithContext(handler.getContext(), handler.CloudDBService.NewListDeploymentScalingGroupsOptions(deploymentID))
	if err != nil {
		return fmt.Errorf("failed to get IBM Cloud Databases scaling groups for %s: %w", info.IId.NameId, err)
	}
	if groups != nil {
		for _, group := range groups.Groups {
			if group.ID == nil || *group.ID != clouddatabasesv5.GroupIDMemberConst {
				continue
			}
			memberCount := int64(1)
			if group.Members != nil && group.Members.AllocationCount != nil && *group.Members.AllocationCount > 0 {
				memberCount = *group.Members.AllocationCount
			}
			if group.Disk != nil && group.Disk.AllocationMb != nil {
				info.StorageSize = strconv.FormatInt((*group.Disk.AllocationMb/memberCount)/ibmStorageUnitGB, 10)
			}
			if group.HostFlavor != nil && group.HostFlavor.ID != nil && *group.HostFlavor.ID != "" {
				info.DBInstanceSpec = *group.HostFlavor.ID
			}
			break
		}
	}

	endpointType := clouddatabasesv5.GetConnectionOptionsEndpointTypePrivateConst
	if info.PublicAccess {
		endpointType = clouddatabasesv5.GetConnectionOptionsEndpointTypePublicConst
	}
	connectionUser := info.MasterUserName
	if connectionUser == "" || connectionUser == "NA" {
		connectionUser = ibmDefaultAdminUser
	}
	connection, _, err := handler.CloudDBService.GetConnectionWithContext(handler.getContext(), handler.CloudDBService.NewGetConnectionOptions(deploymentID, "database", connectionUser, endpointType))
	if err != nil {
		return fmt.Errorf("failed to get IBM Cloud Databases %s connection for %s: %w", endpointType, info.IId.NameId, err)
	}
	if err := applyIBMConnectionInfo(info, connection); err != nil {
		return err
	}

	return nil
}

func ibmDeploymentIDFromResourceInstance(inst *resourcecontrollerv2.ResourceInstance) string {
	if inst == nil {
		return ""
	}
	if inst.CRN != nil && *inst.CRN != "" {
		return *inst.CRN
	}
	if inst.ID != nil && *inst.ID != "" {
		return *inst.ID
	}
	return ""
}

func (handler *IbmRDBMSHandler) getTaggedRDBMSInfo(deploymentID string) ([]irs.KeyValue, error) {
	if handler.TaggingService == nil || deploymentID == "" {
		return nil, errors.New("IBM Global Tagging service is not configured for RDBMS tag lookup")
	}
	tags, _, err := handler.TaggingService.ListTagsWithContext(handler.getContext(), &globaltaggingv1.ListTagsOptions{
		TagType:    core.StringPtr(globaltaggingv1.ListTagsOptionsTagTypeUserConst),
		Providers:  []string{globaltaggingv1.ListTagsOptionsProvidersGhostConst},
		AttachedTo: core.StringPtr(deploymentID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list IBM RDBMS resource tags: %w", err)
	}
	if tags == nil {
		return nil, nil
	}

	var tagList []irs.KeyValue
	for _, tag := range tags.Items {
		if tag.Name == nil {
			continue
		}
		tagKey, tagValue, _ := strings.Cut(*tag.Name, ":")
		if tagKey != "" {
			tagList = append(tagList, irs.KeyValue{Key: tagKey, Value: tagValue})
		}
	}
	return tagList, nil
}

func applyIBMConnectionInfo(info *irs.RDBMSInfo, response *clouddatabasesv5.GetConnectionResponse) error {
	if info == nil || response == nil || response.Connection == nil {
		return errors.New("IBM Cloud Databases connection response did not include connection information")
	}
	connection, ok := response.Connection.(*clouddatabasesv5.Connection)
	if !ok || connection == nil {
		return errors.New("IBM Cloud Databases connection response had unexpected connection type")
	}

	switch info.DBEngine {
	case "mysql":
		if connection.Mysql == nil {
			return errors.New("IBM Cloud Databases MySQL connection response did not include mysql endpoint")
		}
		if err := applyIBMConnectionHosts(info, connection.Mysql.Hosts); err != nil {
			return err
		}
		if connection.Mysql.Database != nil && *connection.Mysql.Database != "" {
			info.DatabaseName = *connection.Mysql.Database
		}
	case "postgresql":
		if connection.Postgres == nil {
			return errors.New("IBM Cloud Databases PostgreSQL connection response did not include postgres endpoint")
		}
		if err := applyIBMConnectionHosts(info, connection.Postgres.Hosts); err != nil {
			return err
		}
		if connection.Postgres.Database != nil && *connection.Postgres.Database != "" {
			info.DatabaseName = *connection.Postgres.Database
		}
	default:
		return fmt.Errorf("IBM Cloud Databases connection response is not supported for DBEngine %s", info.DBEngine)
	}

	return nil
}

func applyIBMConnectionHosts(info *irs.RDBMSInfo, hosts []clouddatabasesv5.ConnectionHost) error {
	if len(hosts) == 0 || hosts[0].Hostname == nil || *hosts[0].Hostname == "" {
		return errors.New("IBM Cloud Databases connection response did not include endpoint hostname")
	}
	if hosts[0].Port == nil || *hosts[0].Port <= 0 {
		return errors.New("IBM Cloud Databases connection response did not include endpoint port")
	}
	info.Port = strconv.FormatInt(*hosts[0].Port, 10)
	info.Endpoint = fmt.Sprintf("%s:%d", *hosts[0].Hostname, *hosts[0].Port)
	return nil
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

func (handler *IbmRDBMSHandler) getDefaultResourceGroupId() (string, error) {
	if defaultResourceGroupId != "" {
		return defaultResourceGroupId, nil
	}

	resourceManagerService, err := resourcemanagerv2.NewResourceManagerV2UsingExternalConfig(&resourcemanagerv2.ResourceManagerV2Options{
		Authenticator: &core.IamAuthenticator{ApiKey: handler.CredentialInfo.ApiKey},
	})
	if err != nil {
		return "", err
	}

	resourceGroups, _, err := resourceManagerService.ListResourceGroupsWithContext(handler.getContext(), &resourcemanagerv2.ListResourceGroupsOptions{Default: core.BoolPtr(true)})
	if err != nil {
		return "", err
	}
	for _, resourceGroup := range resourceGroups.Resources {
		if resourceGroup.Name != nil && resourceGroup.ID != nil && strings.EqualFold(*resourceGroup.Name, DefaultResourceGroup) {
			defaultResourceGroupId = *resourceGroup.ID
			return defaultResourceGroupId, nil
		}
	}

	return "", errors.New("failed to get default resource group")
}
