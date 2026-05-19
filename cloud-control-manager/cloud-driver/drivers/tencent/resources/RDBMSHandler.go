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
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cdb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdb/v20170320"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

type TencentRDBMSHandler struct {
	Region idrv.RegionInfo
	Client *cdb.Client
}

func (handler *TencentRDBMSHandler) GetMetaInfo() (irs.RDBMSMetaInfo, error) {
	cblogger.Debug("Tencent CDB GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()

	metaInfo := irs.RDBMSMetaInfo{
		SupportedEngines: map[string][]string{
			"mysql": {"5.5", "5.6", "5.7", "8.0"},
		},
		SupportsHighAvailability:   true,
		SupportsBackup:             true,
		SupportsPublicAccess:       true,
		SupportsDeletionProtection: false, // Tencent CDB uses isolate/offline pattern, not deletion protection
		SupportsEncryption:         true,
		StorageTypeOptions: map[string][]string{
			"mysql": {"CLOUD_SSD", "CLOUD_HSSD"},
		},
		StorageSizeRange: irs.StorageSizeRange{
			Min: 25,
			Max: 16000,
		},
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return metaInfo, nil
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

	// Parse Memory (DBInstanceSpec is memory in MB for Tencent CDB)
	memory, err := strconv.ParseInt(rdbmsReqInfo.DBInstanceSpec, 10, 64)
	if err != nil {
		return irs.RDBMSInfo{}, fmt.Errorf("invalid DBInstanceSpec (Memory in MB): %s", rdbmsReqInfo.DBInstanceSpec)
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

	// Endpoint
	if inst.Vip != nil {
		rdbmsInfo.Endpoint = *inst.Vip
		if inst.Vport != nil {
			rdbmsInfo.Endpoint += ":" + strconv.FormatInt(*inst.Vport, 10)
		}
	}
	if inst.Vport != nil {
		rdbmsInfo.Port = strconv.FormatInt(*inst.Vport, 10)
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
	rdbmsInfo.MasterUserName = "root" // Tencent CDB default

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
