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

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cdb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdb/v20170320"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type TencentRDBMSHandler struct {
	Region   idrv.RegionInfo
	Client   *cdb.Client
	VMClient *cvm.Client
}

// tencentDefaultAdminUser is the fixed admin username for Tencent Cloud Database (CDB).
// The CreateDBInstanceHour API always creates the instance with "root" as the admin user;
// there is no parameter to specify a custom admin username.
// Reference: https://www.tencentcloud.com/document/product/236/15865 (Password field description:
// "Sets the root account password.")
const tencentDefaultAdminUser = "root"

func (handler *TencentRDBMSHandler) GetMetaInfo(dbEngine string) (irs.RDBMSMetaInfo, error) {
	cblogger.Debug("Tencent CDB GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetMetaInfo", "DescribeCdbZoneConfig()")
	start := call.Start()
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return irs.RDBMSMetaInfo{}, err
	}

	supportedEngines, storageTypeOptions, storageSizeRange, err := handler.fetchCDBMetaOptions()
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSMetaInfo{}, fmt.Errorf("GetMetaInfo failed: %w", err)
	}

	metaInfo, err := irs.BuildRDBMSMetaInfo(requestedEngine, supportedEngines, storageTypeOptions, storageSizeRange, true, true, true, false, true)
	if err != nil {
		return irs.RDBMSMetaInfo{}, err
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return metaInfo, nil
}

func (handler *TencentRDBMSHandler) fetchCDBMetaOptions() (map[string][]string, map[string][]string, irs.StorageSizeRange, error) {
	if handler.Client == nil {
		return nil, nil, irs.StorageSizeRange{}, errors.New("CDB client is unavailable")
	}

	request := cdb.NewDescribeCdbZoneConfigRequest()
	response, err := handler.Client.DescribeCdbZoneConfig(request)
	if err != nil {
		return nil, nil, irs.StorageSizeRange{}, err
	}
	if response.Response == nil || response.Response.DataResult == nil {
		return nil, nil, irs.StorageSizeRange{}, errors.New("DescribeCdbZoneConfig returned empty response")
	}

	data := response.Response.DataResult
	configMap := make(map[int64]*cdb.CdbSellConfig)
	for _, cfg := range data.Configs {
		if cfg == nil || cfg.Id == nil {
			continue
		}
		if cfg.Status != nil && *cfg.Status != 0 {
			continue
		}
		configMap[*cfg.Id] = cfg
	}
	if len(configMap) == 0 {
		return nil, nil, irs.StorageSizeRange{}, errors.New("DescribeCdbZoneConfig returned no available CDB sell configs")
	}

	versionSet := make(map[string]struct{})
	storageTypeSet := make(map[string]struct{})
	selectedConfigIDs := make(map[int64]struct{})

	matchedZone := false
	for _, regionConf := range data.Regions {
		if regionConf == nil || regionConf.Region == nil {
			continue
		}
		if handler.Region.Region != "" && *regionConf.Region != handler.Region.Region {
			continue
		}

		for _, zoneConf := range regionConf.RegionConfig {
			if zoneConf == nil || zoneConf.Zone == nil {
				continue
			}
			if handler.Region.Zone != "" && *zoneConf.Zone != handler.Region.Zone {
				continue
			}
			if zoneConf.Status != nil && *zoneConf.Status != 1 {
				continue
			}

			matchedZone = true
			for _, sellType := range zoneConf.SellType {
				if sellType == nil {
					continue
				}
				for _, version := range sellType.EngineVersion {
					if version != nil && strings.TrimSpace(*version) != "" {
						versionSet[strings.TrimSpace(*version)] = struct{}{}
					}
				}
				for _, cfgID := range sellType.ConfigIds {
					if cfgID == nil {
						continue
					}
					_, ok := configMap[*cfgID]
					if !ok {
						continue
					}
					selectedConfigIDs[*cfgID] = struct{}{}
				}
			}

			for _, diskTypeConf := range zoneConf.DiskTypeConf {
				if diskTypeConf == nil {
					continue
				}
				for _, diskType := range diskTypeConf.DiskType {
					if diskType != nil && strings.TrimSpace(*diskType) != "" {
						storageTypeSet[strings.TrimSpace(*diskType)] = struct{}{}
					}
				}
			}
		}
	}

	if !matchedZone {
		return nil, nil, irs.StorageSizeRange{}, fmt.Errorf("DescribeCdbZoneConfig returned no online zone for region [%s], zone [%s]", handler.Region.Region, handler.Region.Zone)
	}
	if len(versionSet) == 0 {
		return nil, nil, irs.StorageSizeRange{}, fmt.Errorf("DescribeCdbZoneConfig returned no MySQL engine versions for region [%s], zone [%s]", handler.Region.Region, handler.Region.Zone)
	}
	if len(selectedConfigIDs) == 0 {
		return nil, nil, irs.StorageSizeRange{}, fmt.Errorf("DescribeCdbZoneConfig returned no available CDB config IDs for region [%s], zone [%s]", handler.Region.Region, handler.Region.Zone)
	}
	if len(storageTypeSet) == 0 {
		return nil, nil, irs.StorageSizeRange{}, fmt.Errorf("DescribeCdbZoneConfig returned no storage types for region [%s], zone [%s]", handler.Region.Region, handler.Region.Zone)
	}

	storageRange := irs.StorageSizeRange{}
	for cfgID := range selectedConfigIDs {
		cfg, ok := configMap[cfgID]
		if !ok || cfg.VolumeMin == nil || cfg.VolumeMax == nil {
			continue
		}
		if storageRange.Min == 0 || *cfg.VolumeMin < storageRange.Min {
			storageRange.Min = *cfg.VolumeMin
		}
		if *cfg.VolumeMax > storageRange.Max {
			storageRange.Max = *cfg.VolumeMax
		}
	}
	if storageRange.Min == 0 || storageRange.Max == 0 {
		return nil, nil, irs.StorageSizeRange{}, fmt.Errorf("DescribeCdbZoneConfig returned no storage size range for region [%s], zone [%s]", handler.Region.Region, handler.Region.Zone)
	}

	return map[string][]string{
			"mysql": sortedTencentVersionSet(versionSet),
		}, map[string][]string{
			"mysql": sortedTencentStringSet(storageTypeSet),
		}, storageRange, nil
}

func sortedTencentStringSet(set map[string]struct{}) []string {
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func sortedTencentVersionSet(set map[string]struct{}) []string {
	values := sortedTencentStringSet(set)
	sort.Slice(values, func(i, j int) bool {
		return compareTencentVersionStrings(values[i], values[j]) < 0
	})
	return values
}

func compareTencentVersionStrings(left, right string) int {
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	maxLen := len(leftParts)
	if len(rightParts) > maxLen {
		maxLen = len(rightParts)
	}

	for i := 0; i < maxLen; i++ {
		leftNum := int64(0)
		rightNum := int64(0)
		if i < len(leftParts) {
			leftNum, _ = strconv.ParseInt(leftParts[i], 10, 64)
		}
		if i < len(rightParts) {
			rightNum, _ = strconv.ParseInt(rightParts[i], 10, 64)
		}
		if leftNum < rightNum {
			return -1
		}
		if leftNum > rightNum {
			return 1
		}
	}

	return strings.Compare(left, right)
}

func (handler *TencentRDBMSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListIID", "DescribeDBInstances()")
	start := call.Start()

	var iidList []*irs.IID
	var offset uint64 = 0
	var limit uint64 = 100

	for {
		request := cdb.NewDescribeDBInstancesRequest()
		request.Offset = &offset
		request.Limit = &limit

		response, err := handler.Client.DescribeDBInstances(request)
		if err != nil {
			hiscallInfo.ElapsedTime = call.Elapsed(start)
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}

		if response.Response == nil || response.Response.Items == nil {
			break
		}

		for _, inst := range response.Response.Items {
			name := ""
			sysId := ""
			if inst.InstanceName != nil {
				name = *inst.InstanceName
			}
			if inst.InstanceId != nil {
				sysId = *inst.InstanceId
			}
			iid := &irs.IID{
				NameId:   name,
				SystemId: sysId,
			}
			iidList = append(iidList, iid)
		}

		total := int64(0)
		if response.Response.TotalCount != nil {
			total = *response.Response.TotalCount
		}
		if int64(offset+limit) >= total {
			break
		}
		offset += limit
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return iidList, nil
}

func (handler *TencentRDBMSHandler) CreateRDBMS(rdbmsReqInfo irs.RDBMSInfo) (irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsReqInfo.IId.NameId, "CreateDBInstanceHour()")
	start := call.Start()

	// Validate required fields
	if rdbmsReqInfo.IId.NameId == "" {
		return irs.RDBMSInfo{}, errors.New("RDBMS NameId is required")
	}
	if rdbmsReqInfo.DBInstanceSpec == "" {
		return irs.RDBMSInfo{}, errors.New("DBInstanceSpec is required (Memory in MB)")
	}
	if rdbmsReqInfo.StorageSize == "" {
		return irs.RDBMSInfo{}, errors.New("StorageSize is required (in GB)")
	}
	if rdbmsReqInfo.MasterUserPassword == "" {
		return irs.RDBMSInfo{}, errors.New("MasterUserPassword is required")
	}
	if rdbmsReqInfo.MasterUserName != "" && rdbmsReqInfo.MasterUserName != tencentDefaultAdminUser {
		return irs.RDBMSInfo{}, fmt.Errorf("Tencent Cloud Database does not support custom MasterUserName: the admin user is always %q. Set MasterUserName to %q or leave it empty", tencentDefaultAdminUser, tencentDefaultAdminUser)
	}

	// DBInstanceSpec can be memory(MB) or Tencent VM spec name.
	memory, err := handler.resolveMemoryMBFromSpec(rdbmsReqInfo.DBInstanceSpec)
	if err != nil {
		return irs.RDBMSInfo{}, err
	}

	// Parse storage size (volume in GB)
	volume, err := strconv.ParseInt(rdbmsReqInfo.StorageSize, 10, 64)
	if err != nil {
		return irs.RDBMSInfo{}, fmt.Errorf("invalid StorageSize: %s", rdbmsReqInfo.StorageSize)
	}

	request := cdb.NewCreateDBInstanceHourRequest()
	request.Memory = &memory
	request.Volume = &volume
	request.GoodsNum = common.Int64Ptr(1)
	request.InstanceName = &rdbmsReqInfo.IId.NameId
	request.Password = &rdbmsReqInfo.MasterUserPassword

	// Engine version
	if rdbmsReqInfo.DBEngineVersion != "" {
		request.EngineVersion = &rdbmsReqInfo.DBEngineVersion
	}

	// VPC and Subnet
	if rdbmsReqInfo.VpcIID.SystemId != "" {
		request.UniqVpcId = &rdbmsReqInfo.VpcIID.SystemId
	}
	if len(rdbmsReqInfo.SubnetIIDs) > 0 && rdbmsReqInfo.SubnetIIDs[0].SystemId != "" {
		request.UniqSubnetId = &rdbmsReqInfo.SubnetIIDs[0].SystemId
	}

	// Zone
	if handler.Region.Zone != "" {
		request.Zone = &handler.Region.Zone
	}

	// Port
	if rdbmsReqInfo.Port != "" {
		port, portErr := strconv.ParseInt(rdbmsReqInfo.Port, 10, 64)
		if portErr == nil {
			request.Port = &port
		}
	}

	// HA - ProtectMode: 0=async, 1=semi-sync, 2=strong-sync
	if rdbmsReqInfo.HighAvailability {
		protectMode := int64(1) // semi-sync for HA
		request.ProtectMode = &protectMode
	}

	// Security Groups
	if len(rdbmsReqInfo.SecurityGroupIIDs) > 0 {
		sgIds := make([]*string, 0, len(rdbmsReqInfo.SecurityGroupIIDs))
		for _, sg := range rdbmsReqInfo.SecurityGroupIIDs {
			id := sg.SystemId
			sgIds = append(sgIds, &id)
		}
		request.SecurityGroup = sgIds
	}

	// Tags
	if len(rdbmsReqInfo.TagList) > 0 {
		tags := make([]*cdb.TagInfo, 0, len(rdbmsReqInfo.TagList))
		for _, tag := range rdbmsReqInfo.TagList {
			k := tag.Key
			v := tag.Value
			tags = append(tags, &cdb.TagInfo{
				TagKey:   &k,
				TagValue: []*string{&v},
			})
		}
		request.ResourceTags = tags
	}

	response, err := handler.Client.CreateDBInstanceHour(request)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err = handler.enrichUnsupportedSpecError(err, rdbmsReqInfo.DBEngineVersion)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	if response.Response == nil || len(response.Response.InstanceIds) == 0 {
		return irs.RDBMSInfo{}, errors.New("no instance ID returned from CreateDBInstanceHour")
	}

	instanceId := *response.Response.InstanceIds[0]

	// Wait for instance to be running (status=1)
	err = handler.waitForInstanceStatus(instanceId, 1, 600)
	if err != nil {
		cblogger.Warn("Instance created but wait for Running status failed: ", err)
	}

	// Enable public endpoint when requested.
	if rdbmsReqInfo.PublicAccess {
		openWanReq := cdb.NewOpenWanServiceRequest()
		openWanReq.InstanceId = &instanceId
		_, openWanErr := handler.Client.OpenWanService(openWanReq)
		if openWanErr != nil {
			return irs.RDBMSInfo{}, fmt.Errorf("failed to enable public access: %w", openWanErr)
		}

		// Wait until WAN is enabled to return a public endpoint.
		err = handler.waitForWanStatus(instanceId, 1, 300)
		if err != nil {
			return irs.RDBMSInfo{}, fmt.Errorf("public access requested but WAN was not enabled: %w", err)
		}
	}

	return handler.GetRDBMS(irs.IID{NameId: rdbmsReqInfo.IId.NameId, SystemId: instanceId})
}

func (handler *TencentRDBMSHandler) ListRDBMS() ([]*irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListRDBMS", "DescribeDBInstances()")
	start := call.Start()

	var rdbmsList []*irs.RDBMSInfo
	var offset uint64 = 0
	var limit uint64 = 100

	for {
		request := cdb.NewDescribeDBInstancesRequest()
		request.Offset = &offset
		request.Limit = &limit

		response, err := handler.Client.DescribeDBInstances(request)
		if err != nil {
			hiscallInfo.ElapsedTime = call.Elapsed(start)
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}

		if response.Response == nil || response.Response.Items == nil {
			break
		}

		for _, inst := range response.Response.Items {
			rdbmsInfo := handler.convertToRDBMSInfo(inst)
			rdbmsList = append(rdbmsList, &rdbmsInfo)
		}

		total := int64(0)
		if response.Response.TotalCount != nil {
			total = *response.Response.TotalCount
		}
		if int64(offset+limit) >= total {
			break
		}
		offset += limit
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return rdbmsList, nil
}

func (handler *TencentRDBMSHandler) GetRDBMS(rdbmsIID irs.IID) (irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsIID.NameId, "DescribeDBInstances()")
	start := call.Start()

	request := cdb.NewDescribeDBInstancesRequest()
	request.InstanceIds = []*string{&rdbmsIID.SystemId}

	response, err := handler.Client.DescribeDBInstances(request)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	if response.Response == nil || len(response.Response.Items) == 0 {
		return irs.RDBMSInfo{}, fmt.Errorf("DB instance not found: %s", rdbmsIID.SystemId)
	}

	return handler.convertToRDBMSInfo(response.Response.Items[0]), nil
}

func (handler *TencentRDBMSHandler) DeleteRDBMS(rdbmsIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsIID.NameId, "IsolateDBInstance()")
	start := call.Start()

	// Step 1: Isolate the instance
	isolateReq := cdb.NewIsolateDBInstanceRequest()
	isolateReq.InstanceId = &rdbmsIID.SystemId

	_, err := handler.Client.IsolateDBInstance(isolateReq)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	calllogger.Info(call.String(hiscallInfo))

	// Step 2: Offline the isolated instance (permanent delete)
	offlineReq := cdb.NewOfflineIsolatedInstancesRequest()
	offlineReq.InstanceIds = []*string{&rdbmsIID.SystemId}

	_, err = handler.Client.OfflineIsolatedInstances(offlineReq)
	if err != nil {
		cblogger.Warn("Isolate succeeded but Offline failed (instance will be in recycle bin): ", err)
		// Still return true since isolate succeeded
		return true, nil
	}

	return true, nil
}

// ===== Helper Functions =====

func (handler *TencentRDBMSHandler) convertToRDBMSInfo(inst *cdb.InstanceInfo) irs.RDBMSInfo {
	rdbmsInfo := irs.RDBMSInfo{}

	if inst.InstanceName != nil {
		rdbmsInfo.IId.NameId = *inst.InstanceName
	}
	if inst.InstanceId != nil {
		rdbmsInfo.IId.SystemId = *inst.InstanceId
	}

	rdbmsInfo.DBEngine = "mysql" // Tencent CDB is MySQL only
	if inst.EngineVersion != nil {
		rdbmsInfo.DBEngineVersion = *inst.EngineVersion
	}

	// Instance spec (memory)
	if inst.Memory != nil {
		rdbmsInfo.DBInstanceSpec = strconv.FormatInt(*inst.Memory, 10)
	}

	// Storage
	if inst.Volume != nil {
		rdbmsInfo.StorageSize = strconv.FormatInt(*inst.Volume, 10)
	}
	rdbmsInfo.StorageType = "NA" // Not directly exposed in InstanceInfo

	// Endpoint: prefer public WAN endpoint when public access is enabled.
	if inst.WanStatus != nil && *inst.WanStatus == 1 && inst.WanDomain != nil && *inst.WanDomain != "" {
		rdbmsInfo.Endpoint = *inst.WanDomain
		if inst.WanPort != nil {
			rdbmsInfo.Endpoint += ":" + strconv.FormatInt(*inst.WanPort, 10)
			rdbmsInfo.Port = strconv.FormatInt(*inst.WanPort, 10)
		}
	} else if inst.Vip != nil {
		rdbmsInfo.Endpoint = *inst.Vip
		if inst.Vport != nil {
			rdbmsInfo.Endpoint += ":" + strconv.FormatInt(*inst.Vport, 10)
			rdbmsInfo.Port = strconv.FormatInt(*inst.Vport, 10)
		}
	}

	// VPC
	if inst.UniqVpcId != nil {
		rdbmsInfo.VpcIID = irs.IID{SystemId: *inst.UniqVpcId}
	}

	// Subnet
	if inst.UniqSubnetId != nil {
		rdbmsInfo.SubnetIIDs = []irs.IID{{SystemId: *inst.UniqSubnetId}}
	}

	// Status
	if inst.Status != nil {
		rdbmsInfo.Status = convertTencentStatusToRDBMSStatus(*inst.Status)
	}

	// HA (ProtectMode)
	if inst.ProtectMode != nil {
		rdbmsInfo.HighAvailability = (*inst.ProtectMode > 0)
	}

	// Master username
	rdbmsInfo.MasterUserName = tencentDefaultAdminUser // Tencent CDB always uses "root"

	// Created time
	if inst.CreateTime != nil {
		t, err := time.Parse("2006-01-02 15:04:05", *inst.CreateTime)
		if err == nil {
			rdbmsInfo.CreatedTime = t
		}
	}

	// WanStatus (public access)
	if inst.WanStatus != nil {
		rdbmsInfo.PublicAccess = (*inst.WanStatus == 1)
	}

	rdbmsInfo.DatabaseName = "NA"
	rdbmsInfo.BackupTime = "NA"
	rdbmsInfo.ReplicationType = "NA"
	rdbmsInfo.DeletionProtection = false
	rdbmsInfo.Encryption = false

	// KeyValueList
	rdbmsInfo.KeyValueList = irs.StructToKeyValueList(inst)

	return rdbmsInfo
}

func convertTencentStatusToRDBMSStatus(status int64) irs.RDBMSStatus {
	switch status {
	case 0:
		return irs.RDBMSCreating
	case 1:
		return irs.RDBMSAvailable
	case 4:
		return irs.RDBMSDeleting
	case 5:
		return irs.RDBMSStopped
	default:
		return irs.RDBMSError
	}
}

func (handler *TencentRDBMSHandler) waitForInstanceStatus(instanceId string, targetStatus int64, timeoutSec int) error {
	for elapsed := 0; elapsed < timeoutSec; elapsed += 15 {
		request := cdb.NewDescribeDBInstancesRequest()
		request.InstanceIds = []*string{&instanceId}

		response, err := handler.Client.DescribeDBInstances(request)
		if err != nil {
			return err
		}

		if response.Response != nil && len(response.Response.Items) > 0 {
			if response.Response.Items[0].Status != nil && *response.Response.Items[0].Status == targetStatus {
				return nil
			}
		}

		time.Sleep(15 * time.Second)
	}
	return fmt.Errorf("timeout waiting for instance %s to reach status %d", instanceId, targetStatus)
}

func (handler *TencentRDBMSHandler) waitForWanStatus(instanceId string, targetWanStatus int64, timeoutSec int) error {
	for elapsed := 0; elapsed < timeoutSec; elapsed += 10 {
		request := cdb.NewDescribeDBInstancesRequest()
		request.InstanceIds = []*string{&instanceId}

		response, err := handler.Client.DescribeDBInstances(request)
		if err != nil {
			return err
		}

		if response.Response != nil && len(response.Response.Items) > 0 {
			inst := response.Response.Items[0]
			if inst.WanStatus != nil && *inst.WanStatus == targetWanStatus {
				return nil
			}
		}

		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("timeout waiting for instance %s to reach WAN status %d", instanceId, targetWanStatus)
}

func (handler *TencentRDBMSHandler) resolveMemoryMBFromSpec(spec string) (int64, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return 0, errors.New("DBInstanceSpec is empty")
	}

	memoryMB, err := strconv.ParseInt(spec, 10, 64)
	if err == nil {
		return memoryMB, nil
	}

	if handler.VMClient == nil {
		return 0, fmt.Errorf("DBInstanceSpec [%s] is not numeric and VM client is unavailable; provide memory in MB or a valid Tencent VM spec", spec)
	}
	if handler.Region.Zone == "" {
		return 0, fmt.Errorf("DBInstanceSpec [%s] looks like VM spec, but connection Zone is empty; Tencent VM spec resolution requires Zone", spec)
	}

	request := cvm.NewDescribeZoneInstanceConfigInfosRequest()
	request.Filters = []*cvm.Filter{
		{
			Name:   common.StringPtr("zone"),
			Values: common.StringPtrs([]string{handler.Region.Zone}),
		},
		{
			Name:   common.StringPtr("instance-type"),
			Values: common.StringPtrs([]string{spec}),
		},
	}

	response, queryErr := handler.VMClient.DescribeZoneInstanceConfigInfos(request)
	if queryErr != nil {
		return 0, fmt.Errorf("failed to resolve DBInstanceSpec [%s] from Tencent VM spec: %w", spec, queryErr)
	}
	if response.Response == nil || len(response.Response.InstanceTypeQuotaSet) == 0 {
		availableSpecs, _ := handler.listAvailableVMSpecNames(20)
		if len(availableSpecs) > 0 {
			return 0, fmt.Errorf("unsupported Tencent VM spec [%s] in zone [%s]; available VM specs (sample): %s", spec, handler.Region.Zone, strings.Join(availableSpecs, ", "))
		}
		return 0, fmt.Errorf("unsupported Tencent VM spec [%s] in zone [%s]", spec, handler.Region.Zone)
	}

	item := response.Response.InstanceTypeQuotaSet[0]
	if item == nil || item.Memory == nil {
		return 0, fmt.Errorf("Tencent VM spec [%s] has no memory information", spec)
	}

	// Tencent CVM spec memory is reported in GiB, but CDB Memory expects decimal MB-like units (e.g., 2000/4000).
	return (*item.Memory) * 1000, nil
}

func (handler *TencentRDBMSHandler) listAvailableVMSpecNames(limit int) ([]string, error) {
	if handler.VMClient == nil {
		return nil, errors.New("VM client is unavailable")
	}
	if handler.Region.Zone == "" {
		return nil, errors.New("connection Zone is empty")
	}

	request := cvm.NewDescribeZoneInstanceConfigInfosRequest()
	request.Filters = []*cvm.Filter{
		{
			Name:   common.StringPtr("zone"),
			Values: common.StringPtrs([]string{handler.Region.Zone}),
		},
	}

	response, err := handler.VMClient.DescribeZoneInstanceConfigInfos(request)
	if err != nil {
		return nil, err
	}
	if response.Response == nil || len(response.Response.InstanceTypeQuotaSet) == 0 {
		return nil, nil
	}

	specSet := make(map[string]struct{})
	for _, item := range response.Response.InstanceTypeQuotaSet {
		if item != nil && item.InstanceType != nil && *item.InstanceType != "" {
			specSet[*item.InstanceType] = struct{}{}
		}
	}

	specs := make([]string, 0, len(specSet))
	for spec := range specSet {
		specs = append(specs, spec)
	}
	sort.Strings(specs)
	if limit > 0 && len(specs) > limit {
		return specs[:limit], nil
	}

	return specs, nil
}

func (handler *TencentRDBMSHandler) enrichUnsupportedSpecError(err error, engineVersion string) error {
	errMsg := strings.ToLower(err.Error())
	if !strings.Contains(errMsg, "invalidparameter") && !strings.Contains(errMsg, "spec") && !strings.Contains(errMsg, "规格") {
		return err
	}

	specHints, memoryHints, hintErr := handler.listSupportedCDBSpecs(engineVersion, 20)
	if hintErr != nil {
		return fmt.Errorf("%w; failed to query supported CDB specs: %v", err, hintErr)
	}

	if len(specHints) > 0 {
		return fmt.Errorf("%w; supported CDB specs (sample): %s", err, strings.Join(specHints, ", "))
	}
	if len(memoryHints) > 0 {
		return fmt.Errorf("%w; supported memory sizes (MB): %s", err, strings.Join(memoryHints, ", "))
	}

	return err
}

func (handler *TencentRDBMSHandler) listSupportedCDBSpecs(engineVersion string, limit int) ([]string, []string, error) {
	request := cdb.NewDescribeCdbZoneConfigRequest()
	response, err := handler.Client.DescribeCdbZoneConfig(request)
	if err != nil {
		return nil, nil, err
	}
	if response.Response == nil || response.Response.DataResult == nil {
		return nil, nil, nil
	}

	configMap := make(map[int64]*cdb.CdbSellConfig)
	for _, cfg := range response.Response.DataResult.Configs {
		if cfg == nil || cfg.Id == nil {
			continue
		}
		if cfg.Status != nil && *cfg.Status != 0 {
			continue
		}
		configMap[*cfg.Id] = cfg
	}

	selectedIDs := make(map[int64]struct{})
	for _, regionConf := range response.Response.DataResult.Regions {
		if regionConf == nil || regionConf.Region == nil {
			continue
		}
		if handler.Region.Region != "" && *regionConf.Region != handler.Region.Region {
			continue
		}

		for _, zoneConf := range regionConf.RegionConfig {
			if zoneConf == nil || zoneConf.Zone == nil {
				continue
			}
			if handler.Region.Zone != "" && *zoneConf.Zone != handler.Region.Zone {
				continue
			}
			if zoneConf.Status != nil && *zoneConf.Status != 1 {
				continue
			}

			for _, sellType := range zoneConf.SellType {
				if sellType == nil {
					continue
				}
				if !isEngineVersionMatched(engineVersion, sellType.EngineVersion) {
					continue
				}
				for _, cfgID := range sellType.ConfigIds {
					if cfgID != nil {
						selectedIDs[*cfgID] = struct{}{}
					}
				}
			}
		}
	}

	if len(selectedIDs) == 0 {
		for cfgID := range configMap {
			selectedIDs[cfgID] = struct{}{}
		}
	}

	type specItem struct {
		memory int64
		spec   string
	}
	items := []specItem{}
	memorySet := make(map[int64]struct{})
	for cfgID := range selectedIDs {
		cfg, ok := configMap[cfgID]
		if !ok || cfg.Memory == nil {
			continue
		}
		memory := *cfg.Memory
		cpu := int64(-1)
		if cfg.Cpu != nil {
			cpu = *cfg.Cpu
		}
		deviceType := "NA"
		if cfg.DeviceType != nil {
			deviceType = *cfg.DeviceType
		}
		items = append(items, specItem{
			memory: memory,
			spec:   fmt.Sprintf("id=%d,memory=%dMB,cpu=%d,device=%s", cfgID, memory, cpu, deviceType),
		})
		memorySet[memory] = struct{}{}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].memory == items[j].memory {
			return items[i].spec < items[j].spec
		}
		return items[i].memory < items[j].memory
	})

	memoryList := make([]int64, 0, len(memorySet))
	for memory := range memorySet {
		memoryList = append(memoryList, memory)
	}
	sort.Slice(memoryList, func(i, j int) bool {
		return memoryList[i] < memoryList[j]
	})

	memoryHints := make([]string, 0, len(memoryList))
	for _, memory := range memoryList {
		memoryHints = append(memoryHints, strconv.FormatInt(memory, 10))
	}

	specHints := make([]string, 0, len(items))
	for _, item := range items {
		specHints = append(specHints, item.spec)
	}

	if limit > 0 {
		if len(specHints) > limit {
			specHints = specHints[:limit]
		}
		if len(memoryHints) > limit {
			memoryHints = memoryHints[:limit]
		}
	}

	return specHints, memoryHints, nil
}

func isEngineVersionMatched(target string, candidates []*string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return true
	}
	for _, version := range candidates {
		if version != nil && strings.TrimSpace(*version) == target {
			return true
		}
	}
	return false
}

// ─── rdbmsDatabaseManager interface implementation ───────────────────────────

// CreateDatabase creates a database in a Tencent Cloud CDB instance.
func (handler *TencentRDBMSHandler) CreateDatabase(rdbmsSystemId, dbEngine, dbName string) error {
	charSet := "utf8mb4"
	req := cdb.NewCreateDatabaseRequest()
	req.InstanceId = common.StringPtr(rdbmsSystemId)
	req.DBName = common.StringPtr(dbName)
	req.CharacterSetName = common.StringPtr(charSet)
	if _, err := handler.Client.CreateDatabase(req); err != nil {
		return fmt.Errorf("Tencent CreateDatabase: %w", err)
	}
	return nil
}

// ListDatabases lists all databases in a Tencent Cloud CDB instance.
func (handler *TencentRDBMSHandler) ListDatabases(rdbmsSystemId, dbEngine string) ([]string, error) {
	req := cdb.NewDescribeDatabasesRequest()
	req.InstanceId = common.StringPtr(rdbmsSystemId)
	resp, err := handler.Client.DescribeDatabases(req)
	if err != nil {
		return nil, fmt.Errorf("Tencent ListDatabases: %w", err)
	}
	var names []string
	for _, item := range resp.Response.Items {
		if item != nil {
			names = append(names, *item)
		}
	}
	return names, nil
}

// DeleteDatabase deletes a database from a Tencent Cloud CDB instance.
func (handler *TencentRDBMSHandler) DeleteDatabase(rdbmsSystemId, dbEngine, dbName string) error {
	req := cdb.NewDeleteDatabaseRequest()
	req.InstanceId = common.StringPtr(rdbmsSystemId)
	req.DBName = common.StringPtr(dbName)
	if _, err := handler.Client.DeleteDatabase(req); err != nil {
		return fmt.Errorf("Tencent DeleteDatabase: %w", err)
	}
	return nil
}
