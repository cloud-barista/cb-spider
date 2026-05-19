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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsRDBMSHandler struct {
	Region     idrv.RegionInfo
	Client     *rds.RDS
	EC2Client  *ec2.EC2
	TagHandler *AwsTagHandler
}

// GetMetaInfo returns metadata about AWS RDS capabilities.
func (handler *AwsRDBMSHandler) GetMetaInfo() (irs.RDBMSMetaInfo, error) {
	cblogger.Debug("AWS RDS GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()

	metaInfo := irs.RDBMSMetaInfo{
		SupportedEngines: map[string][]string{
			"mysql":      {"5.7", "8.0", "8.4"},
			"mariadb":    {"10.4", "10.5", "10.6", "10.11"},
			"postgresql": {"13", "14", "15", "16", "17"},
		},
		SupportsHighAvailability:   true,
		SupportsBackup:             true,
		SupportsPublicAccess:       true,
		SupportsDeletionProtection: true,
		SupportsEncryption:         true,
		StorageTypeOptions: map[string][]string{
			"mysql":      {"gp2", "gp3", "io1", "io2"},
			"mariadb":    {"gp2", "gp3", "io1", "io2"},
			"postgresql": {"gp2", "gp3", "io1", "io2"},
		},
		StorageSizeRange: irs.StorageSizeRange{
			Min: 20,
			Max: 65536,
		},
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return metaInfo, nil
}

// ListIID returns a list of RDBMS IIDs.
func (handler *AwsRDBMSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListIID", "DescribeDBInstances()")
	start := call.Start()

	input := &rds.DescribeDBInstancesInput{}
	var iidList []*irs.IID

	err := handler.Client.DescribeDBInstancesPages(input, func(page *rds.DescribeDBInstancesOutput, lastPage bool) bool {
		for _, dbInstance := range page.DBInstances {
			iid := &irs.IID{
				NameId:   aws.StringValue(dbInstance.DBInstanceIdentifier),
				SystemId: aws.StringValue(dbInstance.DBInstanceIdentifier),
			}
			iidList = append(iidList, iid)
		}
		return true
	})

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	calllogger.Info(call.String(hiscallInfo))

	return iidList, nil
}

// CreateRDBMS creates a new RDS instance.
func (handler *AwsRDBMSHandler) CreateRDBMS(rdbmsReqInfo irs.RDBMSInfo) (irs.RDBMSInfo, error) {
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

	storageSize, err := strconv.ParseInt(rdbmsReqInfo.StorageSize, 10, 64)
	if err != nil {
		return irs.RDBMSInfo{}, fmt.Errorf("invalid StorageSize: %s", rdbmsReqInfo.StorageSize)
	}

	// Create DB Subnet Group if VPC and Subnets are provided
	subnetGroupName := ""
	if rdbmsReqInfo.VpcIID.SystemId != "" && len(rdbmsReqInfo.SubnetIIDs) > 0 {
		subnetGroupName = "cb-spider-" + rdbmsReqInfo.IId.NameId
		err := handler.createDBSubnetGroup(subnetGroupName, rdbmsReqInfo.SubnetIIDs)
		if err != nil {
			return irs.RDBMSInfo{}, fmt.Errorf("failed to create DB subnet group: %w", err)
		}
	}

	// Build CreateDBInstance input
	input := &rds.CreateDBInstanceInput{
		DBInstanceIdentifier: aws.String(rdbmsReqInfo.IId.NameId),
		DBInstanceClass:      aws.String(rdbmsReqInfo.DBInstanceSpec),
		Engine:               aws.String(rdbmsReqInfo.DBEngine),
		EngineVersion:        aws.String(rdbmsReqInfo.DBEngineVersion),
		MasterUsername:       aws.String(rdbmsReqInfo.MasterUserName),
		MasterUserPassword:   aws.String(rdbmsReqInfo.MasterUserPassword),
		AllocatedStorage:     aws.Int64(storageSize),
	}

	// DB Subnet Group
	if subnetGroupName != "" {
		input.DBSubnetGroupName = aws.String(subnetGroupName)
	}

	// Security Groups
	if len(rdbmsReqInfo.SecurityGroupIIDs) > 0 {
		var sgIDs []*string
		for _, sg := range rdbmsReqInfo.SecurityGroupIIDs {
			sgIDs = append(sgIDs, aws.String(sg.SystemId))
		}
		input.VpcSecurityGroupIds = sgIDs
	}

	// Storage Type (Advanced - default: gp2)
	if rdbmsReqInfo.StorageType != "" {
		input.StorageType = aws.String(rdbmsReqInfo.StorageType)
	}

	// Port
	if rdbmsReqInfo.Port != "" {
		port, err := strconv.ParseInt(rdbmsReqInfo.Port, 10, 64)
		if err == nil {
			input.Port = aws.Int64(port)
		}
	}

	// Database Name
	if rdbmsReqInfo.DatabaseName != "" {
		input.DBName = aws.String(rdbmsReqInfo.DatabaseName)
	}

	// High Availability (Multi-AZ)
	input.MultiAZ = aws.Bool(rdbmsReqInfo.HighAvailability)

	// Backup
	if rdbmsReqInfo.BackupRetentionDays > 0 {
		input.BackupRetentionPeriod = aws.Int64(int64(rdbmsReqInfo.BackupRetentionDays))
	}
	if rdbmsReqInfo.BackupTime != "" {
		// AWS expects "HH:MM-HH:MM" format for preferred backup window
		input.PreferredBackupWindow = aws.String(rdbmsReqInfo.BackupTime)
	}

	// Public Access
	input.PubliclyAccessible = aws.Bool(rdbmsReqInfo.PublicAccess)

	// Encryption
	input.StorageEncrypted = aws.Bool(rdbmsReqInfo.Encryption)

	// Deletion Protection
	input.DeletionProtection = aws.Bool(rdbmsReqInfo.DeletionProtection)

	// Tags
	var rdsTags []*rds.Tag
	for _, tag := range rdbmsReqInfo.TagList {
		rdsTags = append(rdsTags, &rds.Tag{
			Key:   aws.String(tag.Key),
			Value: aws.String(tag.Value),
		})
	}
	// Add Name tag
	rdsTags = append(rdsTags, &rds.Tag{
		Key:   aws.String("Name"),
		Value: aws.String(rdbmsReqInfo.IId.NameId),
	})
	input.Tags = rdsTags

	// Create the RDS instance
	result, err := handler.Client.CreateDBInstance(input)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		// Clean up subnet group on failure
		if subnetGroupName != "" {
			handler.deleteDBSubnetGroup(subnetGroupName)
		}
		return irs.RDBMSInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	rdbmsInfo := handler.convertDBInstanceToRDBMSInfo(result.DBInstance)
	return rdbmsInfo, nil
}

// ListRDBMS returns a list of all RDBMS instances.
func (handler *AwsRDBMSHandler) ListRDBMS() ([]*irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListRDBMS", "DescribeDBInstances()")
	start := call.Start()

	input := &rds.DescribeDBInstancesInput{}
	var rdbmsList []*irs.RDBMSInfo

	err := handler.Client.DescribeDBInstancesPages(input, func(page *rds.DescribeDBInstancesOutput, lastPage bool) bool {
		for _, dbInstance := range page.DBInstances {
			rdbmsInfo := handler.convertDBInstanceToRDBMSInfo(dbInstance)
			rdbmsList = append(rdbmsList, &rdbmsInfo)
		}
		return true
	})

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	calllogger.Info(call.String(hiscallInfo))

	return rdbmsList, nil
}

// GetRDBMS returns the details of a specific RDBMS instance.
func (handler *AwsRDBMSHandler) GetRDBMS(rdbmsIID irs.IID) (irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsIID.NameId, "DescribeDBInstances()")
	start := call.Start()

	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(rdbmsIID.SystemId),
	}

	result, err := handler.Client.DescribeDBInstances(input)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	if len(result.DBInstances) == 0 {
		return irs.RDBMSInfo{}, fmt.Errorf("RDBMS instance not found: %s", rdbmsIID.SystemId)
	}

	rdbmsInfo := handler.convertDBInstanceToRDBMSInfo(result.DBInstances[0])
	return rdbmsInfo, nil
}

// DeleteRDBMS deletes an RDBMS instance.
func (handler *AwsRDBMSHandler) DeleteRDBMS(rdbmsIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsIID.NameId, "DeleteDBInstance()")
	start := call.Start()

	// First, get the instance to find the subnet group name
	descInput := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(rdbmsIID.SystemId),
	}
	descResult, err := handler.Client.DescribeDBInstances(descInput)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	var subnetGroupName string
	if len(descResult.DBInstances) > 0 && descResult.DBInstances[0].DBSubnetGroup != nil {
		subnetGroupName = aws.StringValue(descResult.DBInstances[0].DBSubnetGroup.DBSubnetGroupName)
	}

	// Disable deletion protection if enabled
	if len(descResult.DBInstances) > 0 && aws.BoolValue(descResult.DBInstances[0].DeletionProtection) {
		modInput := &rds.ModifyDBInstanceInput{
			DBInstanceIdentifier: aws.String(rdbmsIID.SystemId),
			DeletionProtection:   aws.Bool(false),
		}
		_, err := handler.Client.ModifyDBInstance(modInput)
		if err != nil {
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return false, fmt.Errorf("failed to disable deletion protection: %w", err)
		}
	}

	// Delete the DB instance
	input := &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier:   aws.String(rdbmsIID.SystemId),
		SkipFinalSnapshot:      aws.Bool(true),
		DeleteAutomatedBackups: aws.Bool(true),
	}

	_, err = handler.Client.DeleteDBInstance(input)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			cblogger.Error(aerr.Error())
		}
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	calllogger.Info(call.String(hiscallInfo))

	// Wait for deletion then clean up subnet group
	if subnetGroupName != "" && strings.HasPrefix(subnetGroupName, "cb-spider-") {
		cblogger.Infof("Waiting for DB instance deletion to clean up subnet group: %s", subnetGroupName)
		err := handler.waitForDBInstanceDeleted(rdbmsIID.SystemId)
		if err != nil {
			cblogger.Warnf("Failed to wait for DB instance deletion: %v", err)
		} else {
			handler.deleteDBSubnetGroup(subnetGroupName)
		}
	}

	return true, nil
}

// ===== Helper Functions =====

func (handler *AwsRDBMSHandler) createDBSubnetGroup(groupName string, subnetIIDs []irs.IID) error {
	var subnetIDs []*string
	for _, subnet := range subnetIIDs {
		subnetIDs = append(subnetIDs, aws.String(subnet.SystemId))
	}

	input := &rds.CreateDBSubnetGroupInput{
		DBSubnetGroupName:        aws.String(groupName),
		DBSubnetGroupDescription: aws.String("CB-Spider RDBMS subnet group for " + groupName),
		SubnetIds:                subnetIDs,
	}

	_, err := handler.Client.CreateDBSubnetGroup(input)
	return err
}

func (handler *AwsRDBMSHandler) deleteDBSubnetGroup(groupName string) {
	input := &rds.DeleteDBSubnetGroupInput{
		DBSubnetGroupName: aws.String(groupName),
	}
	_, err := handler.Client.DeleteDBSubnetGroup(input)
	if err != nil {
		cblogger.Warnf("Failed to delete DB subnet group %s: %v", groupName, err)
	}
}

func (handler *AwsRDBMSHandler) waitForDBInstanceDeleted(dbInstanceId string) error {
	maxWait := 30 * time.Minute
	pollInterval := 30 * time.Second
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		input := &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(dbInstanceId),
		}
		_, err := handler.Client.DescribeDBInstances(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == rds.ErrCodeDBInstanceNotFoundFault {
					return nil // Instance deleted
				}
			}
			return err
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("timeout waiting for DB instance %s to be deleted", dbInstanceId)
}

func (handler *AwsRDBMSHandler) convertDBInstanceToRDBMSInfo(dbInstance *rds.DBInstance) irs.RDBMSInfo {
	rdbmsInfo := irs.RDBMSInfo{}

	dbId := aws.StringValue(dbInstance.DBInstanceIdentifier)
	rdbmsInfo.IId = irs.IID{
		NameId:   dbId,
		SystemId: dbId,
	}

	// VPC
	if dbInstance.DBSubnetGroup != nil {
		rdbmsInfo.VpcIID = irs.IID{
			SystemId: aws.StringValue(dbInstance.DBSubnetGroup.VpcId),
		}
		// Subnets
		for _, subnet := range dbInstance.DBSubnetGroup.Subnets {
			rdbmsInfo.SubnetIIDs = append(rdbmsInfo.SubnetIIDs, irs.IID{
				SystemId: aws.StringValue(subnet.SubnetIdentifier),
			})
		}
	}

	// DB Engine
	rdbmsInfo.DBEngine = aws.StringValue(dbInstance.Engine)
	rdbmsInfo.DBEngineVersion = aws.StringValue(dbInstance.EngineVersion)

	// Instance Spec
	rdbmsInfo.DBInstanceSpec = aws.StringValue(dbInstance.DBInstanceClass)
	if aws.BoolValue(dbInstance.MultiAZ) {
		rdbmsInfo.DBInstanceType = "Multi-AZ"
	} else {
		rdbmsInfo.DBInstanceType = "Primary"
	}

	// Storage
	rdbmsInfo.StorageType = aws.StringValue(dbInstance.StorageType)
	if dbInstance.AllocatedStorage != nil {
		rdbmsInfo.StorageSize = strconv.FormatInt(aws.Int64Value(dbInstance.AllocatedStorage), 10)
	}

	// Security Groups
	for _, sg := range dbInstance.VpcSecurityGroups {
		rdbmsInfo.SecurityGroupIIDs = append(rdbmsInfo.SecurityGroupIIDs, irs.IID{
			SystemId: aws.StringValue(sg.VpcSecurityGroupId),
		})
	}

	// Port
	if dbInstance.Endpoint != nil && dbInstance.Endpoint.Port != nil {
		rdbmsInfo.Port = strconv.FormatInt(aws.Int64Value(dbInstance.Endpoint.Port), 10)
	} else if dbInstance.DbInstancePort != nil {
		rdbmsInfo.Port = strconv.FormatInt(aws.Int64Value(dbInstance.DbInstancePort), 10)
	}

	// Authentication
	rdbmsInfo.MasterUserName = aws.StringValue(dbInstance.MasterUsername)
	// MasterUserPassword is never returned by AWS API

	// Database
	rdbmsInfo.DatabaseName = aws.StringValue(dbInstance.DBName)

	// High Availability
	rdbmsInfo.HighAvailability = aws.BoolValue(dbInstance.MultiAZ)
	if dbInstance.StatusInfos != nil {
		for _, info := range dbInstance.StatusInfos {
			if aws.StringValue(info.StatusType) == "read replication" {
				rdbmsInfo.ReplicationType = "async"
			}
		}
	}

	// Backup
	if dbInstance.BackupRetentionPeriod != nil {
		rdbmsInfo.BackupRetentionDays = int(aws.Int64Value(dbInstance.BackupRetentionPeriod))
	}
	rdbmsInfo.BackupTime = aws.StringValue(dbInstance.PreferredBackupWindow)

	// Access
	rdbmsInfo.PublicAccess = aws.BoolValue(dbInstance.PubliclyAccessible)
	if dbInstance.Endpoint != nil {
		rdbmsInfo.Endpoint = fmt.Sprintf("%s:%d",
			aws.StringValue(dbInstance.Endpoint.Address),
			aws.Int64Value(dbInstance.Endpoint.Port))
	}

	// Encryption
	rdbmsInfo.Encryption = aws.BoolValue(dbInstance.StorageEncrypted)

	// Protection
	rdbmsInfo.DeletionProtection = aws.BoolValue(dbInstance.DeletionProtection)

	// Status
	rdbmsInfo.Status = convertRDSStatusToRDBMSStatus(aws.StringValue(dbInstance.DBInstanceStatus))

	// Created Time
	if dbInstance.InstanceCreateTime != nil {
		rdbmsInfo.CreatedTime = *dbInstance.InstanceCreateTime
	}

	// KeyValueList - capture CSP-specific data
	rdbmsInfo.KeyValueList = irs.StructToKeyValueList(dbInstance)

	return rdbmsInfo
}

func convertRDSStatusToRDBMSStatus(rdsStatus string) irs.RDBMSStatus {
	switch strings.ToLower(rdsStatus) {
	case "creating", "backing-up", "modifying", "upgrading", "configuring-enhanced-monitoring",
		"configuring-iam-database-auth", "configuring-log-exports", "converting-to-vpc",
		"moving-to-vpc", "rebooting", "renaming", "resetting-master-credentials",
		"starting", "storage-optimization":
		return irs.RDBMSCreating
	case "available":
		return irs.RDBMSAvailable
	case "deleting":
		return irs.RDBMSDeleting
	case "stopped", "stopping", "storage-full":
		return irs.RDBMSStopped
	case "failed", "inaccessible-encryption-credentials", "incompatible-network",
		"incompatible-option-group", "incompatible-parameters", "incompatible-restore",
		"restore-error":
		return irs.RDBMSError
	default:
		return irs.RDBMSError
	}
}
