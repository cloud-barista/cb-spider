// Cloud Control Manager's Rest Runtime of CB-Spider.
// Common Runtime for S3 Management
// by CB-Spider Team

package commonruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/cors"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"cloud.google.com/go/storage"
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
	rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"
	infostore "github.com/cloud-barista/cb-spider/info-store"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

type S3BucketIIDInfo struct {
	ConnectionName string `gorm:"primaryKey"`
	NameId         string `gorm:"primaryKey"`
	SystemId       string
	Region         string
}

func (S3BucketIIDInfo) TableName() string {
	return "s3bucket_iid_infos"
}

func init() {
	db, err := infostore.Open()
	if err != nil {
		cblog.Error(err)
		return
	}
	db.AutoMigrate(&S3BucketIIDInfo{})
	infostore.Close(db)
}

// getTencentAppId retrieves AppId from Tencent CAM API
func getTencentAppId(accessKey, secretKey string) (string, error) {
	credential := common.NewCredential(accessKey, secretKey)

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cam.tencentcloudapi.com"

	client, err := cam.NewClient(credential, "", cpf)
	if err != nil {
		return "", fmt.Errorf("failed to create CAM client: %v", err)
	}

	request := cam.NewGetUserAppIdRequest()

	response, err := client.GetUserAppId(request)
	if err != nil {
		return "", fmt.Errorf("failed to get AppId from CAM API: %v", err)
	}

	if response.Response == nil || response.Response.AppId == nil {
		return "", fmt.Errorf("AppId not found in CAM API response")
	}

	return fmt.Sprintf("%d", *response.Response.AppId), nil
}

// ============================================================================
// OpenStack/Ceph RGW Multipart Upload State Management
// ============================================================================

// getAccessKey is a helper to get access/secret key from multiple possible key names
func getAccessKey(kvl infostore.KVList, keys ...string) string {
	for _, key := range keys {
		for _, kv := range kvl {
			if kv.Key == key {
				return kv.Value
			}
		}
	}
	return ""
}

type S3ConnectionInfo struct {
	Endpoint       string
	AccessKey      string
	SecretKey      string
	UseSSL         bool
	RegionRequired bool
	Region         string
	ProviderName   string
	AppId          string // Tencent COS AppId
}

func GetS3ConnectionInfo(connectionName string) (*S3ConnectionInfo, error) {
	cccInfo, err := ccim.GetConnectionConfig(connectionName)
	if err != nil {
		return nil, err
	}

	regionInfo, err := rim.GetRegion(cccInfo.RegionName)
	if err != nil {
		return nil, err
	}
	regionID := ccm.KeyValueListGetValue(regionInfo.KeyValueInfoList, "Region")
	providerName := strings.ToUpper(regionInfo.ProviderName)

	crdInfo, err := cim.GetCredentialDecrypt(cccInfo.CredentialName)
	if err != nil {
		return nil, err
	}

	// Helper to get access/secret key from multiple possible key names
	getAccessKey := func(kvl infostore.KVList, keys ...string) string {
		for _, key := range keys {
			for _, kv := range kvl {
				if kv.Key == key {
					return kv.Value
				}
			}
		}
		return ""
	}

	// Helper to extract S3 endpoint from OpenStack IdentityEndpoint
	getOpenStackS3Endpoint := func() (string, error) {
		identityEndpoint := ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "IdentityEndpoint")
		if identityEndpoint == "" {
			return "", fmt.Errorf("IdentityEndpoint is required for OpenStack S3 connection")
		}

		parsedURL, err := url.Parse(identityEndpoint)
		if err != nil {
			return "", fmt.Errorf("failed to parse IdentityEndpoint URL: %v", err)
		}

		// Replace identity port with S3 port (8080)
		host := parsedURL.Host
		if parsedURL.Port() != "" {
			host = strings.Replace(host, ":"+parsedURL.Port(), ":8080", 1)
		} else {
			host = host + ":8080"
		}

		return host, nil
	}

	var accessKey, secretKey string
	var endpoint string
	var useSSL, regionRequired bool
	var appId string

	switch providerName {
	case "AWS":
		if accessKey == "" {
			accessKey = ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientId")
		}
		if secretKey == "" {
			secretKey = ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientSecret")
		}
		endpoint = fmt.Sprintf("s3.%s.amazonaws.com", regionID)
		useSSL = true
		regionRequired = true

	case "IBM":
		accessKey = getAccessKey(crdInfo.KeyValueInfoList, "S3AccessKey", "access_key_id")
		secretKey = getAccessKey(crdInfo.KeyValueInfoList, "S3SecretKey", "secret_access_key")
		endpoint = fmt.Sprintf("s3.%s.cloud-object-storage.appdomain.cloud", regionID)
		useSSL = true
		regionRequired = false

	case "OPENSTACK":
		accessKey = getAccessKey(crdInfo.KeyValueInfoList, "S3AccessKey", "access")
		secretKey = getAccessKey(crdInfo.KeyValueInfoList, "S3SecretKey", "secret")

		var err error
		endpoint, err = getOpenStackS3Endpoint()
		if err != nil {
			return nil, err
		}

		useSSL = false
		regionRequired = false

	case "KT":
		accessKey = getAccessKey(crdInfo.KeyValueInfoList, "S3AccessKey", "Access Key")
		secretKey = getAccessKey(crdInfo.KeyValueInfoList, "S3SecretKey", "Secret Key")
		endpoint = "obj-e-1.ktcloud.com"
		useSSL = true
		regionRequired = false

	case "GCP":
		accessKey = getAccessKey(crdInfo.KeyValueInfoList, "S3AccessKey", "Access Key")
		secretKey = getAccessKey(crdInfo.KeyValueInfoList, "S3SecretKey", "Secret")
		endpoint = "storage.googleapis.com"
		useSSL = true
		regionRequired = true

	case "ALIBABA":
		accessKey = ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientId")
		secretKey = ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientSecret")
		endpoint = fmt.Sprintf("oss-%s.aliyuncs.com", regionID)
		useSSL = true
		regionRequired = false

	case "TENCENT":
		accessKey = ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientId")
		secretKey = ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientSecret")

		// Dynamically retrieve AppId from Tencent CAM API
		var err error
		appId, err = getTencentAppId(accessKey, secretKey)
		if err != nil {
			cblog.Errorf("Failed to retrieve Tencent AppId: %v", err)
			return nil, fmt.Errorf("failed to retrieve Tencent AppId: %v", err)
		}
		cblog.Infof("Successfully retrieved Tencent AppId: %s", appId)

		endpoint = fmt.Sprintf("cos.%s.myqcloud.com", regionID)
		useSSL = true
		regionRequired = true

	case "NCP":
		accessKey = ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientId")
		secretKey = ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientSecret")
		endpoint = fmt.Sprintf("%s.object.ncloudstorage.com", regionID)
		useSSL = true
		regionRequired = false

	case "NHN":
		accessKey = getAccessKey(crdInfo.KeyValueInfoList, "S3AccessKey", "Access Key")
		secretKey = getAccessKey(crdInfo.KeyValueInfoList, "S3SecretKey", "Secret Key")
		endpoint = fmt.Sprintf("%s-api-object-storage.nhncloudservice.com", regionID)
		useSSL = true
		regionRequired = true

	default:
		return nil, fmt.Errorf("provider '%s' does not support Object Storage service or is not configured for S3 access", providerName)
	}

	// Check if we have empty credentials
	if accessKey == "" {
		return nil, fmt.Errorf("accessKey is empty for provider '%s'", providerName)
	}
	if secretKey == "" {
		return nil, fmt.Errorf("secretKey is empty for provider '%s'", providerName)
	}

	return &S3ConnectionInfo{
		Endpoint:       endpoint,
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		UseSSL:         useSSL,
		RegionRequired: regionRequired,
		Region:         regionID,
		ProviderName:   providerName,
		AppId:          appId,
	}, nil
}

func NewS3Client(connInfo *S3ConnectionInfo) (*minio.Client, error) {
	options := &minio.Options{
		Creds:  credentials.NewStaticV4(connInfo.AccessKey, connInfo.SecretKey, ""),
		Secure: connInfo.UseSSL,
	}

	if connInfo.RegionRequired {
		options.Region = connInfo.Region
	}

	// For Tencent, use virtual-hosted-style (DNS) for all operations except bucket creation
	// This is required by Tencent COS for bucket operations after creation
	if connInfo.ProviderName == "TENCENT" {
		options.BucketLookup = minio.BucketLookupDNS
		options.Region = connInfo.Region
	}

	return minio.New(connInfo.Endpoint, options)
}

// NewS3ClientForBucketCreation creates a client for bucket creation
// For Tencent, uses path-style (non virtual-hosted-style) to avoid DNS issues
func NewS3ClientForBucketCreation(connInfo *S3ConnectionInfo) (*minio.Client, error) {
	options := &minio.Options{
		Creds:  credentials.NewStaticV4(connInfo.AccessKey, connInfo.SecretKey, ""),
		Secure: connInfo.UseSSL,
	}

	if connInfo.RegionRequired {
		options.Region = connInfo.Region
	}

	// For Tencent, use path-style for bucket creation to avoid DNS resolution issues
	// and ensure region is set for proper authentication
	if connInfo.ProviderName == "TENCENT" {
		options.BucketLookup = minio.BucketLookupPath
		options.Region = connInfo.Region
	}

	return minio.New(connInfo.Endpoint, options)
}

func CreateS3Bucket(connectionName, bucketName string) (*minio.BucketInfo, error) {
	cblog.Info("call CreateS3Bucket()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	exist, err := infostore.HasByConditions(&S3BucketIIDInfo{}, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if exist {
		return nil, fmt.Errorf("S3 Bucket '%s' already exists in connection '%s'", bucketName, connectionName)
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cblog.Infof("CreateS3Bucket: Provider=%s, AppId='%s', BucketName=%s", connInfo.ProviderName, connInfo.AppId, bucketName)

	// For Tencent COS, AppId is required
	originalBucketName := bucketName
	if connInfo.ProviderName == "TENCENT" {
		if connInfo.AppId == "" {
			cblog.Error("Tencent COS AppId is empty!")
			return nil, fmt.Errorf("failed to retrieve Tencent AppId from CAM API")
		}
		if !strings.HasSuffix(bucketName, "-"+connInfo.AppId) {
			bucketName = bucketName + "-" + connInfo.AppId
			cblog.Infof("Tencent COS: Appending AppId to bucket name: %s -> %s", originalBucketName, bucketName)
		}
	}

	cblog.Infof("CreateS3Bucket: Final bucket name: %s", bucketName)

	// Use special client for bucket creation (Tencent uses path-style)
	client, err := NewS3ClientForBucketCreation(connInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("S3 Bucket '%s' already exists in connection '%s'", bucketName, connectionName)
	}
	if connInfo.RegionRequired {
		if connInfo.Region == "" {
			return nil, fmt.Errorf("Region is required for S3 connection %s", connectionName)
		} else {
			err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: connInfo.Region})
		}
	} else {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	}
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	iidInfo := S3BucketIIDInfo{
		ConnectionName: connectionName,
		NameId:         originalBucketName,
		SystemId:       bucketName,
		Region:         connInfo.Region,
	}
	err = infostore.Insert(&iidInfo)
	if err != nil {
		cblog.Error(err)
		_ = client.RemoveBucket(ctx, bucketName)
		return nil, err
	}
	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}
	for _, b := range buckets {
		if b.Name == bucketName {
			return &b, nil
		}
	}
	return nil, fmt.Errorf("S3 Bucket '%s' created, but info not found", bucketName)
}

func ListS3Buckets(connectionName string) ([]*minio.BucketInfo, error) {
	cblog.Info("call ListS3Buckets()")

	var iidInfoList []*S3BucketIIDInfo
	err := infostore.ListByCondition(&iidInfoList, "connection_name", connectionName)
	if err != nil {
		return nil, err
	}

	// Return empty list if no metadata exists
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList := []*minio.BucketInfo{}
		return infoList, nil
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return nil, err
	}
	client, err := NewS3Client(connInfo)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	allBuckets, err := client.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}

	var out []*minio.BucketInfo
	for _, iid := range iidInfoList {
		found := false
		for _, b := range allBuckets {
			if b.Name == iid.SystemId {
				// Create a copy and replace SystemId with NameId for user display
				bucketInfo := b
				bucketInfo.Name = iid.NameId
				out = append(out, &bucketInfo)
				found = true
				break
			}
		}
		// If not found in CSP, return metadata-only info (like NLB pattern)
		if !found {
			cblog.Warnf("Bucket '%s' (SystemId: %s) exists in metadata but not found in CSP", iid.NameId, iid.SystemId)
			bucketInfo := minio.BucketInfo{
				Name:         iid.NameId,
				CreationDate: time.Time{}, // Zero value for metadata-only bucket
			}
			out = append(out, &bucketInfo)
		}
	}
	return out, nil
}

func GetS3Bucket(connectionName, bucketName string) (*minio.BucketInfo, error) {
	cblog.Info("call GetS3Bucket()")
	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return nil, err
	}
	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return nil, err
	}
	client, err := NewS3Client(connInfo)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}
	for _, b := range buckets {
		if b.Name == iidInfo.SystemId {
			// Return NameId instead of SystemId for user display
			bucketInfo := b
			bucketInfo.Name = iidInfo.NameId
			return &bucketInfo, nil
		}
	}
	// If not found in CSP, return metadata-only info (like NLB pattern)
	cblog.Warnf("Bucket '%s' (SystemId: %s) exists in metadata but not found in CSP", iidInfo.NameId, iidInfo.SystemId)
	bucketInfo := minio.BucketInfo{
		Name:         iidInfo.NameId,
		CreationDate: time.Time{}, // Zero value for metadata-only bucket
	}
	return &bucketInfo, nil
}

func GetS3BucketRegionInfo(connectionName, bucketName string) (string, error) {
	cblog.Info("call GetS3BucketRegionInfo()")
	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return "", err
	}
	return iidInfo.Region, nil
}

func DeleteS3Bucket(connectionName, bucketName string, force string) (bool, error) {
	cblog.Info("call DeleteS3Bucket()")

	// (1) get IID for the bucket
	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	// (2) delete Resource from CSP
	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	ctx := context.Background()
	result := false

	// Use SystemId (actual bucket name with AppId for Tencent)
	err = client.RemoveBucket(ctx, iidInfo.SystemId)
	if err != nil {
		cblog.Error(err)
		if checkNotFoundError(err) {
			// if not found in CSP, require explicit force parameter
			if force != "true" {
				cblog.Errorf("Bucket %s not found in CSP. Use force=true parameter to delete metadata only.", bucketName)
				return false, fmt.Errorf("bucket not found in CSP (metadata exists). Use force=true to delete metadata only")
			}
			cblog.Infof("Bucket %s not found in CSP, proceeding with force delete (metadata only)", bucketName)
		} else if force != "true" {
			return false, err
		}
	} else {
		result = true
	}

	if force != "true" {
		if result == false {
			return result, nil
		}
	}

	// (3) delete IID from metadata
	_, err = infostore.DeleteByConditions(&S3BucketIIDInfo{}, "connection_name", iidInfo.ConnectionName, "name_id", bucketName)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}

	return true, nil
}

func ListS3Objects(connectionName, bucketName, prefix string) ([]minio.ObjectInfo, error) {
	cblog.Info("call ListS3Objects()")
	cblog.Infof("Parameters - Connection: %s, Bucket: %s, Prefix: '%s'", connectionName, bucketName, prefix)

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		cblog.Errorf("Failed to get bucket info for %s: %v", bucketName, err)
		return nil, err
	}
	cblog.Infof("Found bucket info - SystemId: %s, Region: %s", iidInfo.SystemId, iidInfo.Region)

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		cblog.Errorf("Failed to get connection info: %v", err)
		return nil, err
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		cblog.Errorf("Failed to create S3 client: %v", err)
		return nil, err
	}

	// Set timeout for listing objects to prevent indefinite hangs
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true, // Get all objects, including in subdirectories
	}

	cblog.Infof("Listing objects with options - Prefix: '%s', Recursive: %t", opts.Prefix, opts.Recursive)

	var out []minio.ObjectInfo
	objectCount := 0

	// Use SystemId (actual bucket name with AppId for Tencent)
	for obj := range client.ListObjects(ctx, iidInfo.SystemId, opts) {
		if obj.Err != nil {
			cblog.Errorf("Error listing object: %v", obj.Err)
			// Check for context timeout
			if ctx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("listing objects timed out after 60s (provider: %s may have network issues)", connInfo.ProviderName)
			}
			continue
		}

		objectCount++
		if objectCount <= 10 { // Log first 10 objects for debugging
			cblog.Infof("Object %d: Key=%s, Size=%d, LastModified=%s, ETag=%s",
				objectCount, obj.Key, obj.Size, obj.LastModified, obj.ETag)
		}

		out = append(out, obj)
	}

	cblog.Infof("Total objects found: %d", len(out))

	if len(out) > 10 {
		cblog.Infof("... and %d more objects (showing first 10 in logs)", len(out)-10)
	}

	// Log some statistics
	var totalSize int64
	var folderCount int
	var fileCount int

	for _, obj := range out {
		totalSize += obj.Size
		if strings.HasSuffix(obj.Key, "/") {
			folderCount++
		} else {
			fileCount++
		}
	}

	cblog.Infof("Summary - Files: %d, Folders: %d, Total size: %d bytes", fileCount, folderCount, totalSize)

	return out, nil
}

func GetS3ObjectInfo(connectionName, bucketName, objectName string) (*minio.ObjectInfo, error) {
	cblog.Info("call GetS3ObjectInfo()")
	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return nil, err
	}
	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return nil, err
	}
	client, err := NewS3Client(connInfo)
	if err != nil {
		return nil, err
	}
	// Set timeout for StatObject
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stat, err := client.StatObject(ctx, iidInfo.SystemId, objectName, minio.StatObjectOptions{})
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("getting object info timed out after 30s (provider: %s may have network issues)", connInfo.ProviderName)
		}
		return nil, err
	}
	return &stat, nil
}

func GetS3ObjectInfoWithVersion(connectionName, bucketName, objectName, versionId string) (*minio.ObjectInfo, error) {
	cblog.Info("call GetS3ObjectInfoWithVersion()")
	cblog.Infof("Parameters - Connection: %s, Bucket: %s, Object: %s, Version: %s",
		connectionName, bucketName, objectName, versionId)

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		cblog.Errorf("Failed to get bucket info: %v", err)
		return nil, err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		cblog.Errorf("Failed to get connection info: %v", err)
		return nil, err
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		cblog.Errorf("Failed to create S3 client: %v", err)
		return nil, err
	}

	// Set timeout for StatObject with version
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Special handling for null version ID
	if versionId == "null" {
		cblog.Infof("Handling null version ID")

		// List all versions to find the null version
		opts := minio.ListObjectsOptions{
			Prefix:       objectName,
			Recursive:    false,
			WithVersions: true,
		}

		var nullVersionExists bool
		var actualVersionId string

		for obj := range client.ListObjects(ctx, iidInfo.SystemId, opts) {
			if obj.Err != nil {
				cblog.Errorf("Error listing objects: %v", obj.Err)
				if ctx.Err() == context.DeadlineExceeded {
					return nil, fmt.Errorf("listing object versions timed out after 30s (provider: %s may have network issues)", connInfo.ProviderName)
				}
				continue
			}

			// Look for the exact object with null version ID that has content
			if obj.Key == objectName && !obj.IsDeleteMarker && obj.Size > 0 {
				if obj.VersionID == "" || obj.VersionID == "null" {
					nullVersionExists = true
					actualVersionId = obj.VersionID
					break
				}
			}
		}

		if !nullVersionExists {
			cblog.Errorf("Could not find null version object with content")
			return nil, fmt.Errorf("null version object not found or has no content")
		}

		// Use the actual version ID we found
		statOpts := minio.StatObjectOptions{VersionID: actualVersionId}
		stat, err := client.StatObject(ctx, iidInfo.SystemId, objectName, statOpts)
		if err == nil {
			cblog.Infof("Successfully got null version object info")
			return &stat, nil
		}
		cblog.Errorf("Failed to get null version object info: %v", err)

		// Try alternative methods if the direct approach fails
		methods := []string{"", "null"}
		for _, versionID := range methods {
			if versionID != actualVersionId {
				statOpts := minio.StatObjectOptions{VersionID: versionID}
				stat, err := client.StatObject(ctx, iidInfo.SystemId, objectName, statOpts)
				if err == nil {
					cblog.Infof("Successfully got object info using alternative method")
					return &stat, nil
				}
			}
		}

		// Try without version ID as last resort
		statOpts2 := minio.StatObjectOptions{}
		stat, err = client.StatObject(ctx, iidInfo.SystemId, objectName, statOpts2)
		if err == nil {
			cblog.Infof("Successfully got object info without version ID")
			return &stat, nil
		}

		return nil, fmt.Errorf("all methods failed to get null version object info")
	}

	// Handle normal version IDs
	opts := minio.StatObjectOptions{}
	if versionId != "" && versionId != "undefined" {
		opts.VersionID = versionId
		cblog.Infof("Using version ID: %s", versionId)
	} else {
		cblog.Infof("No version ID specified, getting latest version")
	}

	stat, err := client.StatObject(ctx, iidInfo.SystemId, objectName, opts)
	if err != nil {
		cblog.Errorf("Failed to stat object: %v", err)
		return nil, err
	}

	cblog.Infof("Successfully got object info for %s (version: %s)", objectName, versionId)
	return &stat, nil
}

func DeleteS3Object(connectionName, bucketName, objectName string) (bool, error) {
	cblog.Info("call DeleteS3Object()")
	cblog.Infof("Parameters - Connection: %s, Bucket: %s, Object: %s", connectionName, bucketName, objectName)

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return false, err
	}
	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return false, err
	}
	client, err := NewS3Client(connInfo)
	if err != nil {
		return false, err
	}
	ctx := context.Background()
	err = client.RemoveObject(ctx, iidInfo.SystemId, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		cblog.Errorf("Failed to delete object: %v", err)
		return false, err
	}
	cblog.Infof("Successfully deleted object - Bucket: %s, Object: %s", bucketName, objectName)
	return true, nil
}

// DeleteS3ObjectDeleteMarker deletes a delete marker (null version ID case)
func DeleteS3ObjectDeleteMarker(connectionName, bucketName, objectName string) (bool, error) {
	cblog.Info("call DeleteS3ObjectDeleteMarker()")
	cblog.Infof("Parameters - Connection: %s, Bucket: %s, Object: %s", connectionName, bucketName, objectName)

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		cblog.Errorf("Failed to get bucket info: %v", err)
		return false, err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		cblog.Errorf("Failed to get connection info: %v", err)
		return false, err
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		cblog.Errorf("Failed to create S3 client: %v", err)
		return false, err
	}

	ctx := context.Background()

	// Declare variables that will be reused
	var stillExists bool
	var verifyErr error

	// Method 1: Try to delete using empty version ID (for latest delete marker)
	cblog.Infof("Attempting to delete DELETE MARKER using empty version ID")
	err = client.RemoveObject(ctx, iidInfo.SystemId, objectName, minio.RemoveObjectOptions{
		VersionID: "",
	})

	if err == nil {
		cblog.Infof("RemoveObject call succeeded, verifying DELETE MARKER is actually deleted")

		// Verify the delete marker is actually gone
		stillExists, verifyErr := verifyDeleteMarkerRemoved(client, ctx, iidInfo.SystemId, objectName)
		if verifyErr != nil {
			cblog.Warnf("Failed to verify DELETE MARKER deletion: %v", verifyErr)
		} else if stillExists {
			cblog.Warnf("DELETE MARKER still exists after deletion attempt")
		} else {
			cblog.Infof("DELETE MARKER successfully removed and verified")
			return true, nil
		}
	}

	cblog.Warnf("Failed to delete with empty version ID: %v", err)

	// Method 2: Try to find the actual delete marker and delete it
	cblog.Infof("Attempting to find and delete DELETE MARKER by listing versions")

	opts := minio.ListObjectsOptions{
		Prefix:       objectName,
		Recursive:    false,
		WithVersions: true,
	}

	for obj := range client.ListObjects(ctx, iidInfo.SystemId, opts) {
		if obj.Err != nil {
			cblog.Errorf("Error listing objects: %v", obj.Err)
			continue
		}

		// Look for the exact object that is a delete marker
		if obj.Key == objectName && obj.IsDeleteMarker {
			cblog.Infof("Found DELETE MARKER: Key=%s, VersionID=%s, IsLatest=%t",
				obj.Key, obj.VersionID, obj.IsLatest)

			// Try to delete using the actual version ID if available
			if obj.VersionID != "" {
				cblog.Infof("Attempting to delete DELETE MARKER using version ID: %s", obj.VersionID)
				err = client.RemoveObject(ctx, iidInfo.SystemId, objectName, minio.RemoveObjectOptions{
					VersionID: obj.VersionID,
				})

				if err == nil {
					cblog.Infof("Successfully deleted DELETE MARKER using version ID: %s", obj.VersionID)

					// Verify deletion
					stillExists, verifyErr := verifyDeleteMarkerRemoved(client, ctx, iidInfo.SystemId, objectName)
					if verifyErr != nil {
						cblog.Warnf("Failed to verify DELETE MARKER deletion: %v", verifyErr)
					} else if !stillExists {
						return true, nil
					}
				}

				cblog.Warnf("Failed to delete DELETE MARKER with version ID %s: %v", obj.VersionID, err)
			}

			// Try deleting without version ID for this specific delete marker
			cblog.Infof("Attempting to delete DELETE MARKER without version ID")
			err = client.RemoveObject(ctx, iidInfo.SystemId, objectName, minio.RemoveObjectOptions{})

			if err == nil {
				cblog.Infof("Successfully deleted DELETE MARKER without version ID")

				// Verify deletion
				stillExists, verifyErr = verifyDeleteMarkerRemoved(client, ctx, iidInfo.SystemId, objectName)
				if verifyErr != nil {
					cblog.Warnf("Failed to verify DELETE MARKER deletion: %v", verifyErr)
				} else if !stillExists {
					return true, nil
				}
			}

			cblog.Warnf("Failed to delete DELETE MARKER without version ID: %v", err)

			// Try using minio Core API for low-level access
			cblog.Infof("Attempting to delete DELETE MARKER using Core API")
			core := minio.Core{Client: client}
			err = core.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})

			if err == nil {
				cblog.Infof("Successfully deleted DELETE MARKER using Core API")

				// Verify deletion
				stillExists, verifyErr = verifyDeleteMarkerRemoved(client, ctx, iidInfo.SystemId, objectName)
				if verifyErr != nil {
					cblog.Warnf("Failed to verify DELETE MARKER deletion: %v", verifyErr)
				} else if !stillExists {
					return true, nil
				}
			}

			cblog.Warnf("Failed to delete DELETE MARKER using Core API: %v", err)
		}
	}

	// Method 3: Alternative approach - try to restore object by creating a new version
	// This effectively removes the delete marker by making it non-latest
	cblog.Infof("Attempting to remove DELETE MARKER by creating a minimal new version")

	// Create a very small temporary object to overwrite the delete marker
	tempContent := strings.NewReader("temp")
	putInfo, err := client.PutObject(ctx, iidInfo.SystemId, objectName, tempContent, 4, minio.PutObjectOptions{
		ContentType: "text/plain",
	})

	if err != nil {
		cblog.Errorf("Failed to create temporary object to overwrite DELETE MARKER: %v", err)
		return false, fmt.Errorf("all deletion methods failed for DELETE MARKER")
	}

	cblog.Infof("Created temporary object to overwrite DELETE MARKER, ETag: %s", putInfo.ETag)

	// Now the delete marker should no longer be the latest version
	// Verify this worked
	stillExists, verifyErr = verifyDeleteMarkerRemoved(client, ctx, iidInfo.SystemId, objectName)
	if verifyErr != nil {
		cblog.Warnf("Failed to verify DELETE MARKER removal after creating new version: %v", verifyErr)
	} else if !stillExists {
		cblog.Infof("DELETE MARKER successfully removed by creating new version")

		// Optionally, delete the temporary object we just created
		cblog.Infof("Deleting temporary object created to remove DELETE MARKER")
		err = client.RemoveObject(ctx, iidInfo.SystemId, objectName, minio.RemoveObjectOptions{})
		if err != nil {
			cblog.Warnf("Failed to delete temporary object: %v", err)
			// This is not a critical error, just log it
		}

		return true, nil
	}

	cblog.Errorf("DELETE MARKER still exists even after creating new version")

	// Clean up the temporary object since our approach didn't work
	cblog.Infof("Cleaning up temporary object")
	err = client.RemoveObject(ctx, iidInfo.SystemId, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		cblog.Warnf("Failed to clean up temporary object: %v", err)
	}

	cblog.Infof("Successfully removed DELETE MARKER by creating and deleting new version")

	// Final verification
	stillExists, verifyErr = verifyDeleteMarkerRemoved(client, ctx, iidInfo.SystemId, objectName)
	if verifyErr != nil {
		cblog.Warnf("Failed to verify final DELETE MARKER deletion: %v", verifyErr)
	} else if stillExists {
		cblog.Errorf("DELETE MARKER still exists after all deletion attempts")
		return false, fmt.Errorf("DELETE MARKER still exists after all deletion attempts")
	}

	return true, nil
}

// Helper function to verify if delete marker is actually removed
func verifyDeleteMarkerRemoved(client *minio.Client, ctx context.Context, bucketName, objectName string) (bool, error) {
	cblog.Infof("Verifying DELETE MARKER removal for %s", objectName)

	opts := minio.ListObjectsOptions{
		Prefix:       objectName,
		Recursive:    false,
		WithVersions: true,
	}

	for obj := range client.ListObjects(ctx, bucketName, opts) {
		if obj.Err != nil {
			return false, obj.Err
		}

		if obj.Key == objectName && obj.IsDeleteMarker {
			cblog.Warnf("DELETE MARKER still exists: Key=%s, VersionID=%s", obj.Key, obj.VersionID)
			return true, nil // Still exists
		}
	}

	cblog.Infof("DELETE MARKER no longer exists for %s", objectName)
	return false, nil // Successfully removed
}

func GetS3ObjectStream(connectionName, bucketName, objectName string) (io.ReadCloser, error) {
	cblog.Info("call GetS3ObjectStream()")
	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return nil, err
	}
	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return nil, err
	}
	client, err := NewS3Client(connInfo)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	obj, err := client.GetObject(ctx, iidInfo.SystemId, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func GetS3ObjectStreamWithVersion(connectionName, bucketName, objectName, versionId string) (io.ReadCloser, error) {
	cblog.Info("call GetS3ObjectStreamWithVersion()")
	cblog.Infof("Parameters - Connection: %s, Bucket: %s, Object: %s, Version: %s",
		connectionName, bucketName, objectName, versionId)

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		cblog.Errorf("Failed to get bucket info: %v", err)
		return nil, err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		cblog.Errorf("Failed to get connection info: %v", err)
		return nil, err
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		cblog.Errorf("Failed to create S3 client: %v", err)
		return nil, err
	}

	ctx := context.Background()

	// Special handling for null version ID
	if versionId == "null" {
		cblog.Infof("Handling null version ID")

		// List all versions to find the null version
		opts := minio.ListObjectsOptions{
			Prefix:       objectName,
			Recursive:    false,
			WithVersions: true,
		}

		var nullVersionExists bool
		var actualVersionId string

		for obj := range client.ListObjects(ctx, iidInfo.SystemId, opts) {
			if obj.Err != nil {
				cblog.Errorf("Error listing objects: %v", obj.Err)
				continue
			}

			// Look for the exact object with null version ID that has content
			if obj.Key == objectName && !obj.IsDeleteMarker && obj.Size > 0 {
				if obj.VersionID == "" || obj.VersionID == "null" {
					nullVersionExists = true
					actualVersionId = obj.VersionID
					cblog.Infof("Found null version object with size: %d", obj.Size)
					break
				}
			}
		}

		if !nullVersionExists {
			cblog.Errorf("Could not find null version object with content")
			return nil, fmt.Errorf("null version object not found or has no content")
		}

		// Use the actual version ID we found
		getOpts := minio.GetObjectOptions{VersionID: actualVersionId}
		obj, err := client.GetObject(ctx, iidInfo.SystemId, objectName, getOpts)
		if err == nil {
			cblog.Infof("Successfully got null version object")
			return obj, nil
		}
		cblog.Errorf("Failed to get null version object: %v", err)

		// Try alternative methods if the direct approach fails
		methods := []struct {
			name      string
			versionID string
		}{
			{"empty string", ""},
			{"null string", "null"},
		}

		for _, method := range methods {
			if method.versionID != actualVersionId {
				getOpts := minio.GetObjectOptions{VersionID: method.versionID}
				obj, err := client.GetObject(ctx, iidInfo.SystemId, objectName, getOpts)
				if err == nil {
					cblog.Infof("Successfully got object using alternative method")
					return obj, nil
				}
			}
		}

		// Try without version ID as last resort
		getOpts2 := minio.GetObjectOptions{}
		obj, err = client.GetObject(ctx, iidInfo.SystemId, objectName, getOpts2)
		if err == nil {
			cblog.Infof("Successfully got object without version ID")
			return obj, nil
		}

		return nil, fmt.Errorf("all methods failed to get null version object")
	}

	// Handle normal version IDs
	opts := minio.GetObjectOptions{}
	if versionId != "" && versionId != "undefined" {
		opts.VersionID = versionId
		cblog.Infof("Using version ID: %s", versionId)
	} else {
		cblog.Infof("No version ID specified, getting latest version")
	}

	obj, err := client.GetObject(ctx, iidInfo.SystemId, objectName, opts)
	if err != nil {
		cblog.Errorf("Failed to get object: %v", err)
		return nil, err
	}

	cblog.Infof("Successfully got object stream for %s (version: %s)", objectName, versionId)
	return obj, nil
}

func PutS3ObjectFromReader(connectionName string, bucketName string, objectName string, reader io.Reader, objectSize int64) (minio.UploadInfo, error) {
	cblog.Info("call PutS3ObjectFromReader()")

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return minio.UploadInfo{}, err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return minio.UploadInfo{}, err
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		return minio.UploadInfo{}, err
	}

	ctx := context.Background()
	contentType := "application/octet-stream"

	info, err := client.PutObject(
		ctx,
		iidInfo.SystemId,
		objectName,
		reader,
		objectSize,
		minio.PutObjectOptions{ContentType: contentType},
	)

	if err != nil {
		cblog.Error("Failed to upload object from reader:", err)
		return minio.UploadInfo{}, err
	}

	cblog.Infof("Successfully uploaded %s of size %d to bucket %s", objectName, info.Size, bucketName)

	uploadInfo := minio.UploadInfo{
		Bucket:       info.Bucket,
		Key:          info.Key,
		ETag:         info.ETag,
		Size:         info.Size,
		LastModified: info.LastModified,
		Location:     info.Location,
		VersionID:    info.VersionID,
	}

	return uploadInfo, nil
}

type CompletePart struct {
	PartNumber int
	ETag       string
}

type DeleteResult struct {
	Key     string
	Success bool
	Error   string
}

func InitiateMultipartUpload(connectionName string, bucketName string, objectName string) (string, error) {
	cblog.Info("call InitiateMultipartUpload()")

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return "", err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return "", err
	}

	// Check if provider supports multipart upload
	if connInfo.ProviderName == "OPENSTACK" {
		return "", fmt.Errorf("multipart upload is not supported by %s:%s", connectionName, connInfo.ProviderName)
	}

	cblog.Infof("Initiating multipart upload - Provider: %s, Bucket: %s, Object: %s",
		connInfo.ProviderName, iidInfo.SystemId, objectName)

	// Use standard S3 API for all providers
	client, err := NewS3Client(connInfo)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	core := minio.Core{Client: client}
	uploadID, err := core.NewMultipartUpload(ctx, iidInfo.SystemId, objectName, minio.PutObjectOptions{})
	if err != nil {
		cblog.Errorf("Failed to initiate multipart upload for provider %s: %v", connInfo.ProviderName, err)
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("multipart upload initiation timed out after 30s (provider: %s may not support this operation)", connInfo.ProviderName)
		}
		return "", err
	}

	cblog.Infof("Successfully initiated multipart upload - UploadID: %s", uploadID)
	return uploadID, nil
}

func UploadPart(connectionName string, bucketName string, objectName string, uploadID string, partNumber int, reader io.Reader, size int64) (string, error) {
	cblog.Info("call UploadPart()")

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return "", err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return "", err
	}

	// Check if provider supports multipart upload
	if connInfo.ProviderName == "OPENSTACK" {
		return "", fmt.Errorf("multipart upload is not supported by %s:%s", connectionName, connInfo.ProviderName)
	}

	cblog.Infof("Uploading part %d - Provider: %s, Bucket: %s, Object: %s, UploadID: %s, Size: %d",
		partNumber, connInfo.ProviderName, iidInfo.SystemId, objectName, uploadID, size)

	// Use standard S3 API for all providers
	client, err := NewS3Client(connInfo)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	core := minio.Core{Client: client}
	part, err := core.PutObjectPart(ctx, iidInfo.SystemId, objectName, uploadID, partNumber, reader, size, minio.PutObjectPartOptions{})
	if err != nil {
		cblog.Errorf("Failed to upload part %d for provider %s: %v", partNumber, connInfo.ProviderName, err)
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("part upload timed out after 60s (provider: %s may have network issues)", connInfo.ProviderName)
		}
		return "", err
	}

	cblog.Infof("Successfully uploaded part %d - ETag: %s", partNumber, part.ETag)
	return part.ETag, nil
}

func CompleteMultipartUpload(connectionName string, bucketName string, objectName string, uploadID string, parts []CompletePart) (string, string, error) {
	cblog.Info("call CompleteMultipartUpload()")

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return "", "", err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return "", "", err
	}

	// Check if provider supports multipart upload
	if connInfo.ProviderName == "OPENSTACK" {
		return "", "", fmt.Errorf("multipart upload is not supported by %s:%s", connectionName, connInfo.ProviderName)
	}

	cblog.Infof("Completing multipart upload - Provider: %s, Bucket: %s, Object: %s, UploadID: %s, Parts: %d",
		connInfo.ProviderName, iidInfo.SystemId, objectName, uploadID, len(parts))

	// Use standard S3 API for all providers
	client, err := NewS3Client(connInfo)
	if err != nil {
		return "", "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var completeParts []minio.CompletePart
	for _, part := range parts {
		completeParts = append(completeParts, minio.CompletePart{
			PartNumber: part.PartNumber,
			ETag:       part.ETag,
		})
	}

	core := minio.Core{Client: client}
	uploadInfo, err := core.CompleteMultipartUpload(ctx, iidInfo.SystemId, objectName, uploadID, completeParts, minio.PutObjectOptions{})
	if err != nil {
		cblog.Errorf("Failed to complete multipart upload for provider %s: %v", connInfo.ProviderName, err)
		if ctx.Err() == context.DeadlineExceeded {
			return "", "", fmt.Errorf("multipart upload completion timed out after 60s (provider: %s may not support this operation)", connInfo.ProviderName)
		}
		return "", "", err
	}

	location := fmt.Sprintf("/%s/%s", iidInfo.NameId, objectName)
	cblog.Infof("Successfully completed multipart upload - Location: %s, ETag: %s", location, uploadInfo.ETag)
	return location, uploadInfo.ETag, nil
}

func AbortMultipartUpload(connectionName string, bucketName string, objectName string, uploadID string) error {
	cblog.Info("call AbortMultipartUpload()")

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return err
	}

	// Check if provider supports multipart upload
	if connInfo.ProviderName == "OPENSTACK" {
		return fmt.Errorf("multipart upload is not supported by %s:%s", connectionName, connInfo.ProviderName)
	}

	cblog.Infof("Aborting multipart upload - Provider: %s, Bucket: %s, Object: %s, UploadID: %s",
		connInfo.ProviderName, iidInfo.SystemId, objectName, uploadID)

	// Use standard S3 API for all providers
	client, err := NewS3Client(connInfo)
	if err != nil {
		return err
	}

	// Set timeout for aborting multipart upload
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	core := minio.Core{Client: client}
	err = core.AbortMultipartUpload(ctx, iidInfo.SystemId, objectName, uploadID)
	if err != nil {
		cblog.Errorf("Failed to abort multipart upload for provider %s: %v", connInfo.ProviderName, err)
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("multipart upload abort timed out after 30s (provider: %s may not support this operation)", connInfo.ProviderName)
		}
		return err
	}

	cblog.Infof("Successfully aborted multipart upload - UploadID: %s", uploadID)
	return nil
}

type ListPartsResult struct {
	Bucket               string     `xml:"Bucket"`
	Key                  string     `xml:"Key"`
	UploadID             string     `xml:"UploadId"`
	PartNumberMarker     int        `xml:"PartNumberMarker"`
	NextPartNumberMarker int        `xml:"NextPartNumberMarker"`
	MaxParts             int        `xml:"MaxParts"`
	IsTruncated          bool       `xml:"IsTruncated"`
	Parts                []PartInfo `xml:"Part"`
	Initiator            Initiator  `xml:"Initiator"`
	Owner                Owner      `xml:"Owner"`
	StorageClass         string     `xml:"StorageClass"`
}

type PartInfo struct {
	PartNumber   int       `xml:"PartNumber"`
	LastModified time.Time `xml:"LastModified"`
	ETag         string    `xml:"ETag"`
	Size         int64     `xml:"Size"`
}

type Initiator struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type Owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

func ListParts(connectionName string, bucketName string, objectName string, uploadID string, partNumberMarker int, maxParts int) (*ListPartsResult, error) {
	cblog.Info("call ListParts()")

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return nil, err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return nil, err
	}

	if maxParts == 0 {
		maxParts = 1000 // Default max parts
	}

	cblog.Infof("Listing parts - Provider: %s, Bucket: %s, Object: %s, UploadID: %s",
		connInfo.ProviderName, iidInfo.SystemId, objectName, uploadID)

	// Use standard S3 API for all providers
	client, err := NewS3Client(connInfo)
	if err != nil {
		return nil, err
	}

	// Set timeout for listing parts
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	core := minio.Core{Client: client}

	result, err := core.ListObjectParts(ctx, iidInfo.SystemId, objectName, uploadID, partNumberMarker, maxParts)
	if err != nil {
		cblog.Errorf("Failed to list parts for provider %s: %v", connInfo.ProviderName, err)
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("listing parts timed out after 30s (provider: %s may not support this operation)", connInfo.ProviderName)
		}
		return nil, err
	}

	cblog.Infof("Successfully listed parts - Found: %d parts", len(result.ObjectParts))

	listResult := &ListPartsResult{
		Bucket:               iidInfo.NameId,
		Key:                  result.Key,
		UploadID:             result.UploadID,
		PartNumberMarker:     result.PartNumberMarker,
		NextPartNumberMarker: result.NextPartNumberMarker,
		MaxParts:             result.MaxParts,
		IsTruncated:          result.IsTruncated,
		StorageClass:         result.StorageClass,
		Initiator: Initiator{
			ID:          result.Initiator.ID,
			DisplayName: result.Initiator.DisplayName,
		},
		Owner: Owner{
			ID:          result.Owner.ID,
			DisplayName: result.Owner.DisplayName,
		},
	}

	for _, part := range result.ObjectParts {
		listResult.Parts = append(listResult.Parts, PartInfo{
			PartNumber:   part.PartNumber,
			LastModified: part.LastModified,
			ETag:         part.ETag,
			Size:         part.Size,
		})
	}

	return listResult, nil
}

type ListMultipartUploadsResult struct {
	Bucket             string                `xml:"Bucket"`
	KeyMarker          string                `xml:"KeyMarker"`
	UploadIDMarker     string                `xml:"UploadIdMarker"`
	NextKeyMarker      string                `xml:"NextKeyMarker"`
	NextUploadIDMarker string                `xml:"NextUploadIdMarker"`
	MaxUploads         int                   `xml:"MaxUploads"`
	IsTruncated        bool                  `xml:"IsTruncated"`
	Uploads            []MultipartUploadInfo `xml:"Upload"`
	Prefix             string                `xml:"Prefix"`
	Delimiter          string                `xml:"Delimiter"`
}

type MultipartUploadInfo struct {
	Key          string    `xml:"Key"`
	UploadID     string    `xml:"UploadId"`
	Initiated    time.Time `xml:"Initiated"`
	StorageClass string    `xml:"StorageClass"`
	Initiator    Initiator `xml:"Initiator"`
	Owner        Owner     `xml:"Owner"`
}

func ListMultipartUploads(connectionName string, bucketName string, prefix string, keyMarker string, uploadIDMarker string, delimiter string, maxUploads int) (*ListMultipartUploadsResult, error) {
	cblog.Info("call ListMultipartUploads()")

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return nil, err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return nil, err
	}

	// Check if provider supports multipart upload
	if connInfo.ProviderName == "OPENSTACK" {
		return nil, fmt.Errorf("multipart upload is not supported by %s:%s", connectionName, connInfo.ProviderName)
	}

	if maxUploads == 0 {
		maxUploads = 1000 // Default max uploads
	}

	cblog.Infof("Listing multipart uploads - Provider: %s, Bucket: %s, Prefix: %s, MaxUploads: %d",
		connInfo.ProviderName, iidInfo.SystemId, prefix, maxUploads)

	// Use standard S3 API for all providers
	client, err := NewS3Client(connInfo)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	core := minio.Core{Client: client}

	result, err := core.ListMultipartUploads(ctx, iidInfo.SystemId, prefix, keyMarker, uploadIDMarker, delimiter, maxUploads)
	if err != nil {
		cblog.Errorf("Failed to list multipart uploads for provider %s: %v", connInfo.ProviderName, err)
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("listing multipart uploads timed out after 30s (provider: %s may not support this operation)", connInfo.ProviderName)
		}
		return nil, err
	}

	cblog.Infof("Successfully listed multipart uploads - Found: %d, IsTruncated: %v", len(result.Uploads), result.IsTruncated)

	listResult := &ListMultipartUploadsResult{
		Bucket:             iidInfo.NameId,
		KeyMarker:          result.KeyMarker,
		UploadIDMarker:     result.UploadIDMarker,
		NextKeyMarker:      result.NextKeyMarker,
		NextUploadIDMarker: result.NextUploadIDMarker,
		MaxUploads:         int(result.MaxUploads),
		IsTruncated:        result.IsTruncated,
		Prefix:             result.Prefix,
		Delimiter:          result.Delimiter,
	}

	for _, upload := range result.Uploads {
		listResult.Uploads = append(listResult.Uploads, MultipartUploadInfo{
			Key:          upload.Key,
			UploadID:     upload.UploadID,
			Initiated:    upload.Initiated,
			StorageClass: upload.StorageClass,
			Initiator: Initiator{
				ID:          upload.Initiator.ID,
				DisplayName: upload.Initiator.DisplayName,
			},
			Owner: Owner{
				ID:          upload.Owner.ID,
				DisplayName: upload.Owner.DisplayName,
			},
		})
	}

	return listResult, nil
}

func DeleteMultipleObjects(connectionName string, bucketName string, objectNames []string) ([]DeleteResult, error) {
	cblog.Info("call DeleteMultipleObjects()")

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return nil, err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return nil, err
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	objectsCh := make(chan minio.ObjectInfo)

	go func() {
		defer close(objectsCh)
		for _, objectName := range objectNames {
			objectsCh <- minio.ObjectInfo{
				Key: objectName,
			}
		}
	}()

	results := []DeleteResult{}
	for err := range client.RemoveObjects(ctx, iidInfo.SystemId, objectsCh, minio.RemoveObjectsOptions{}) {
		result := DeleteResult{
			Key:     err.ObjectName,
			Success: false,
			Error:   err.Err.Error(),
		}
		results = append(results, result)
	}

	for _, objectName := range objectNames {
		found := false
		for _, result := range results {
			if result.Key == objectName {
				found = true
				break
			}
		}
		if !found {
			results = append(results, DeleteResult{
				Key:     objectName,
				Success: true,
			})
		}
	}

	return results, nil
}

// S3 Advanced Features Implementation

func GetS3PresignedURL(connectionName string, bucketName string, objectName string, method string, expiresSeconds int64, responseContentDisposition string) (string, error) {
	cblog.Info("call GetS3PresignedURL() - CSP S3 URL mode")
	var iidInfo S3BucketIIDInfo
	if err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName); err != nil {
		return "", err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return "", err
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	expires := time.Duration(expiresSeconds) * time.Second

	// CSP S3 PreSigned URL 반환
	switch method {
	case "GET":
		var params url.Values
		if responseContentDisposition != "" {
			params = url.Values{}
			params.Set("response-content-disposition", responseContentDisposition)
		}
		u, err := client.PresignedGetObject(ctx, iidInfo.SystemId, objectName, expires, params)
		if err != nil {
			return "", err
		}
		cblog.Infof("CSP S3 PreSigned GET URL: %s", u.String())
		return u.String(), nil

	case "PUT":
		u, err := client.PresignedPutObject(ctx, iidInfo.SystemId, objectName, expires)
		if err != nil {
			return "", err
		}
		cblog.Infof("CSP S3 PreSigned PUT URL: %s", u.String())
		return u.String(), nil

	default:
		return "", fmt.Errorf("Unsupported method: %s", method)
	}
}

func EnableVersioning(connectionName string, bucketName string) (bool, error) {
	cblog.Info("call EnableVersioning()")
	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return false, err
	}
	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return false, err
	}

	// Check if provider supports versioning
	if connInfo.ProviderName == "OPENSTACK" {
		return false, fmt.Errorf("bucket versioning is not supported by %s:%s", connectionName, connInfo.ProviderName)
	}
	if connInfo.ProviderName == "NHN" {
		return false, fmt.Errorf("bucket versioning is not supported by %s:%s (NHN Cloud Object Storage does not support versioning feature)", connectionName, connInfo.ProviderName)
	}
	if connInfo.ProviderName == "NCP" || connInfo.ProviderName == "NCPVPC" {
		return false, fmt.Errorf("bucket versioning is not supported by %s:%s (Naver Cloud Platform Object Storage does not support versioning feature)", connectionName, connInfo.ProviderName)
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		return false, err
	}
	ctx := context.Background()
	opts := minio.BucketVersioningConfiguration{
		Status: "Enabled",
	}

	err = client.SetBucketVersioning(ctx, iidInfo.SystemId, opts)
	if err != nil {
		return false, err
	}
	return true, nil
}

func SuspendVersioning(connectionName string, bucketName string) (bool, error) {
	cblog.Info("call SuspendVersioning()")
	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return false, err
	}
	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return false, err
	}

	// Check if provider supports versioning
	if connInfo.ProviderName == "OPENSTACK" {
		return false, fmt.Errorf("bucket versioning is not supported by %s:%s", connectionName, connInfo.ProviderName)
	}
	if connInfo.ProviderName == "NHN" {
		return false, fmt.Errorf("bucket versioning is not supported by %s:%s (NHN Cloud Object Storage does not support versioning feature)", connectionName, connInfo.ProviderName)
	}
	if connInfo.ProviderName == "NCP" || connInfo.ProviderName == "NCPVPC" {
		return false, fmt.Errorf("bucket versioning is not supported by %s:%s (Naver Cloud Platform Object Storage does not support versioning feature)", connectionName, connInfo.ProviderName)
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		return false, err
	}
	ctx := context.Background()
	opts := minio.BucketVersioningConfiguration{
		Status: "Suspended",
	}
	err = client.SetBucketVersioning(ctx, iidInfo.SystemId, opts)
	if err != nil {
		return false, err
	}
	return true, nil
}

func GetVersioning(connectionName string, bucketName string) (string, error) {
	cblog.Info("call GetVersioning()")
	cblog.Infof("Getting versioning for bucket: %s, connection: %s", bucketName, connectionName)

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		cblog.Errorf("Failed to get bucket IID info: %v", err)
		return "", err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		cblog.Errorf("Failed to get connection info: %v", err)
		return "", err
	}

	// Check if provider supports versioning
	if connInfo.ProviderName == "OPENSTACK" {
		return "", fmt.Errorf("bucket versioning is not supported by %s:%s", connectionName, connInfo.ProviderName)
	}
	if connInfo.ProviderName == "NHN" {
		return "", fmt.Errorf("bucket versioning is not supported by %s:%s (NHN Cloud Object Storage does not support versioning feature)", connectionName, connInfo.ProviderName)
	}
	if connInfo.ProviderName == "NCP" || connInfo.ProviderName == "NCPVPC" {
		return "", fmt.Errorf("bucket versioning is not supported by %s:%s (Naver Cloud Platform Object Storage does not support versioning feature)", connectionName, connInfo.ProviderName)
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		cblog.Errorf("Failed to create S3 client: %v", err)
		return "", err
	}

	ctx := context.Background()
	cblog.Infof("Calling GetBucketVersioning for bucket: %s", bucketName)

	versioningConfig, err := client.GetBucketVersioning(ctx, iidInfo.SystemId)
	if err != nil {
		cblog.Errorf("GetBucketVersioning failed: %v", err)
		cblog.Infof("Returning default status 'Suspended' due to error")
		return "Suspended", nil
	}

	cblog.Infof("GetBucketVersioning success - Status: %s", versioningConfig.Status)

	if versioningConfig.Status == "" {
		cblog.Infof("Empty status returned, defaulting to 'Suspended'")
		return "Suspended", nil
	}

	return versioningConfig.Status, nil
}

func ListS3ObjectVersions(connectionName string, bucketName string, prefix string) ([]minio.ObjectInfo, error) {
	cblog.Info("call ListS3ObjectVersions()")

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return nil, err
	}

	// Check if provider supports versioning
	if connInfo.ProviderName == "OPENSTACK" {
		return nil, fmt.Errorf("bucket versioning is not supported by %s:%s", connectionName, connInfo.ProviderName)
	}
	if connInfo.ProviderName == "NHN" {
		return nil, fmt.Errorf("bucket versioning is not supported by %s:%s (NHN Cloud Object Storage does not support versioning feature)", connectionName, connInfo.ProviderName)
	}
	if connInfo.ProviderName == "NCP" || connInfo.ProviderName == "NCPVPC" {
		return nil, fmt.Errorf("bucket versioning is not supported by %s:%s (Naver Cloud Platform Object Storage does not support versioning feature)", connectionName, connInfo.ProviderName)
	}

	// Use GCP Storage SDK for GCP
	if connInfo.ProviderName == "GCP" {
		return listGCPObjectVersions(connectionName, bucketName, prefix)
	}

	var iidInfo S3BucketIIDInfo
	err = infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return nil, err
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		return nil, err
	}

	// Set timeout for listing object versions
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cblog.Infof("Listing object versions - Provider: %s, Bucket: %s, Prefix: %s",
		connInfo.ProviderName, iidInfo.SystemId, prefix)

	opts := minio.ListObjectsOptions{
		Prefix:       prefix,
		Recursive:    true,
		WithVersions: true,
	}

	var out []minio.ObjectInfo
	for obj := range client.ListObjects(ctx, iidInfo.SystemId, opts) {
		if obj.Err != nil {
			cblog.Warnf("Error listing object version: %v", obj.Err)
			// Check for context timeout
			if ctx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("listing object versions timed out after 30s (provider: %s may not support this operation)", connInfo.ProviderName)
			}
			continue
		}
		out = append(out, obj)
	}

	cblog.Infof("Successfully listed object versions - Found: %d versions", len(out))
	return out, nil
}

// listSwiftObjectVersions lists object versions using Swift native versioning
// listGCPObjectVersions lists object versions using GCP Storage SDK
func listGCPObjectVersions(connectionName string, bucketName string, prefix string) ([]minio.ObjectInfo, error) {
	cblog.Info("call listGCPObjectVersions() - using GCP Storage SDK")

	// Get connection config to extract credential info
	cccInfo, err := ccim.GetConnectionConfig(connectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection config: %w", err)
	}

	crdInfo, err := cim.GetCredentialDecrypt(cccInfo.CredentialName)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}

	// Build GCP credentials JSON
	clientEmail := ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientEmail")
	privateKey := ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "PrivateKey")

	if clientEmail == "" || privateKey == "" {
		return nil, fmt.Errorf("GCP credentials (ClientEmail, PrivateKey) not found")
	}

	credentialsJSON := map[string]string{
		"type":         "service_account",
		"private_key":  privateKey,
		"client_email": clientEmail,
		"token_uri":    "https://oauth2.googleapis.com/token",
	}

	credBytes, err := json.Marshal(credentialsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	ctx := context.Background()

	// Create GCP storage client with credentials
	storageClient, err := storage.NewClient(ctx, option.WithCredentialsJSON(credBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP storage client: %w", err)
	}
	defer storageClient.Close()

	// Get bucket IID info
	var iidInfo S3BucketIIDInfo
	err = infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket info: %w", err)
	}

	bucket := storageClient.Bucket(iidInfo.SystemId)

	// List all object versions (including non-current versions)
	query := &storage.Query{
		Prefix:   prefix,
		Versions: true, // Include all versions
	}

	it := bucket.Objects(ctx, query)

	var versions []minio.ObjectInfo
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			cblog.Errorf("Error iterating object versions: %v", err)
			return nil, fmt.Errorf("failed to iterate object versions: %w", err)
		}

		// Convert GCP ObjectAttrs to minio.ObjectInfo
		objInfo := minio.ObjectInfo{
			Key:          attrs.Name,
			Size:         attrs.Size,
			LastModified: attrs.Updated,
			ETag:         attrs.Etag,
			VersionID:    fmt.Sprintf("%d", attrs.Generation),      // GCP uses Generation as version
			IsLatest:     attrs.Generation == attrs.Metageneration, // Approximate
		}

		// Check if this is a delete marker (in GCP, deleted objects have Deleted time set)
		if !attrs.Deleted.IsZero() {
			objInfo.IsDeleteMarker = true
		}

		versions = append(versions, objInfo)
	}

	cblog.Infof("Found %d object versions in GCP bucket %s", len(versions), bucketName)
	return versions, nil
}

// setGCPBucketCORS sets CORS configuration using GCP Storage SDK
func setGCPBucketCORS(connectionName string, bucketName string, allowedOrigins []string, allowedMethods []string, allowedHeaders []string, exposeHeaders []string, maxAgeSeconds int) (bool, error) {
	cblog.Info("call setGCPBucketCORS() - using GCP Storage SDK")

	// Get connection config to extract credential info
	cccInfo, err := ccim.GetConnectionConfig(connectionName)
	if err != nil {
		return false, fmt.Errorf("failed to get connection config: %w", err)
	}

	crdInfo, err := cim.GetCredentialDecrypt(cccInfo.CredentialName)
	if err != nil {
		return false, fmt.Errorf("failed to get credential: %w", err)
	}

	// Build GCP credentials JSON
	clientEmail := ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientEmail")
	privateKey := ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "PrivateKey")

	if clientEmail == "" || privateKey == "" {
		return false, fmt.Errorf("GCP credentials (ClientEmail, PrivateKey) not found")
	}

	credentialsJSON := map[string]string{
		"type":         "service_account",
		"private_key":  privateKey,
		"client_email": clientEmail,
		"token_uri":    "https://oauth2.googleapis.com/token",
	}

	credBytes, err := json.Marshal(credentialsJSON)
	if err != nil {
		return false, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	ctx := context.Background()

	// Create GCP storage client with credentials
	storageClient, err := storage.NewClient(ctx, option.WithCredentialsJSON(credBytes))
	if err != nil {
		return false, fmt.Errorf("failed to create GCP storage client: %w", err)
	}
	defer storageClient.Close()

	// Get bucket IID info
	var iidInfo S3BucketIIDInfo
	err = infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return false, fmt.Errorf("failed to get bucket info: %w", err)
	}

	bucket := storageClient.Bucket(iidInfo.SystemId)

	// Convert to GCP CORS format
	gcpCORS := []storage.CORS{
		{
			MaxAge:          time.Duration(maxAgeSeconds) * time.Second,
			Methods:         allowedMethods,
			Origins:         allowedOrigins,
			ResponseHeaders: exposeHeaders,
		},
	}

	// Update bucket CORS configuration
	bucketAttrsToUpdate := storage.BucketAttrsToUpdate{
		CORS: gcpCORS,
	}

	_, err = bucket.Update(ctx, bucketAttrsToUpdate)
	if err != nil {
		cblog.Errorf("Failed to set GCP bucket CORS: %v", err)
		return false, fmt.Errorf("failed to set GCP bucket CORS: %w", err)
	}

	cblog.Infof("Successfully set CORS for GCP bucket %s using GCP Storage SDK", bucketName)
	return true, nil
}

// getGCPBucketCORS retrieves CORS configuration using GCP Storage SDK
func getGCPBucketCORS(connectionName string, bucketName string) (*cors.Config, error) {
	cblog.Info("call getGCPBucketCORS() - using GCP Storage SDK")

	// Get connection config to extract credential info
	cccInfo, err := ccim.GetConnectionConfig(connectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection config: %w", err)
	}

	crdInfo, err := cim.GetCredentialDecrypt(cccInfo.CredentialName)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}

	// Build GCP credentials JSON
	clientEmail := ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientEmail")
	privateKey := ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "PrivateKey")

	if clientEmail == "" || privateKey == "" {
		return nil, fmt.Errorf("GCP credentials (ClientEmail, PrivateKey) not found")
	}

	credentialsJSON := map[string]string{
		"type":         "service_account",
		"private_key":  privateKey,
		"client_email": clientEmail,
		"token_uri":    "https://oauth2.googleapis.com/token",
	}

	credBytes, err := json.Marshal(credentialsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	ctx := context.Background()

	// Create GCP storage client with credentials
	storageClient, err := storage.NewClient(ctx, option.WithCredentialsJSON(credBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP storage client: %w", err)
	}
	defer storageClient.Close()

	// Get bucket IID info
	var iidInfo S3BucketIIDInfo
	err = infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket info: %w", err)
	}

	bucket := storageClient.Bucket(iidInfo.SystemId)
	attrs, err := bucket.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket attributes: %w", err)
	}

	if len(attrs.CORS) == 0 {
		return nil, fmt.Errorf("CORS configuration not found for bucket %s", bucketName)
	}

	// Convert GCP CORS to minio CORS format for consistent response
	corsConfig := &cors.Config{
		CORSRules: make([]cors.Rule, 0, len(attrs.CORS)),
	}

	for _, gcpCORS := range attrs.CORS {
		rule := cors.Rule{
			AllowedOrigin: gcpCORS.Origins,
			AllowedMethod: gcpCORS.Methods,
			AllowedHeader: []string{"*"}, // GCP doesn't expose this in attrs
			ExposeHeader:  gcpCORS.ResponseHeaders,
			MaxAgeSeconds: int(gcpCORS.MaxAge.Seconds()),
		}
		corsConfig.CORSRules = append(corsConfig.CORSRules, rule)
	}

	cblog.Infof("Successfully retrieved CORS for GCP bucket %s", bucketName)
	return corsConfig, nil
}

// deleteGCPBucketCORS deletes CORS configuration using GCP Storage SDK
func deleteGCPBucketCORS(connectionName string, bucketName string) (bool, error) {
	cblog.Info("call deleteGCPBucketCORS() - using GCP Storage SDK")

	// Get connection config to extract credential info
	cccInfo, err := ccim.GetConnectionConfig(connectionName)
	if err != nil {
		return false, fmt.Errorf("failed to get connection config: %w", err)
	}

	crdInfo, err := cim.GetCredentialDecrypt(cccInfo.CredentialName)
	if err != nil {
		return false, fmt.Errorf("failed to get credential: %w", err)
	}

	// Build GCP credentials JSON
	clientEmail := ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientEmail")
	privateKey := ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "PrivateKey")

	if clientEmail == "" || privateKey == "" {
		return false, fmt.Errorf("GCP credentials (ClientEmail, PrivateKey) not found")
	}

	credentialsJSON := map[string]string{
		"type":         "service_account",
		"private_key":  privateKey,
		"client_email": clientEmail,
		"token_uri":    "https://oauth2.googleapis.com/token",
	}

	credBytes, err := json.Marshal(credentialsJSON)
	if err != nil {
		return false, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	ctx := context.Background()

	// Create GCP storage client with credentials
	storageClient, err := storage.NewClient(ctx, option.WithCredentialsJSON(credBytes))
	if err != nil {
		return false, fmt.Errorf("failed to create GCP storage client: %w", err)
	}
	defer storageClient.Close()

	// Get bucket IID info
	var iidInfo S3BucketIIDInfo
	err = infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return false, fmt.Errorf("failed to get bucket info: %w", err)
	}

	bucket := storageClient.Bucket(iidInfo.SystemId)

	// Set CORS to empty slice to delete
	bucketAttrsToUpdate := storage.BucketAttrsToUpdate{
		CORS: []storage.CORS{},
	}

	_, err = bucket.Update(ctx, bucketAttrsToUpdate)
	if err != nil {
		cblog.Errorf("Failed to delete GCP bucket CORS: %v", err)
		return false, fmt.Errorf("failed to delete GCP bucket CORS: %w", err)
	}

	cblog.Infof("Successfully deleted CORS for GCP bucket %s", bucketName)
	return true, nil
}

func SetS3BucketCORS(connectionName string, bucketName string, allowedOrigins []string, allowedMethods []string, allowedHeaders []string, exposeHeaders []string, maxAgeSeconds int) (bool, error) {
	cblog.Info("call SetS3BucketCORS()")

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return false, err
	}

	// Check if provider supports CORS
	if connInfo.ProviderName == "NHN" || connInfo.ProviderName == "NCP" || connInfo.ProviderName == "NCPVPC" {
		return false, fmt.Errorf("CORS configuration is not supported by %s:%s", connectionName, connInfo.ProviderName)
	}

	// Use GCP Storage SDK for GCP
	if connInfo.ProviderName == "GCP" {
		return setGCPBucketCORS(connectionName, bucketName, allowedOrigins, allowedMethods, allowedHeaders, exposeHeaders, maxAgeSeconds)
	}

	var iidInfo S3BucketIIDInfo
	err = infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return false, err
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		return false, err
	}

	ctx := context.Background()

	corsConfig := &cors.Config{
		CORSRules: []cors.Rule{
			{
				AllowedOrigin: allowedOrigins,
				AllowedMethod: allowedMethods,
				AllowedHeader: allowedHeaders,
				ExposeHeader:  exposeHeaders,
				MaxAgeSeconds: maxAgeSeconds,
			},
		},
	}

	core := minio.Core{Client: client}
	err = core.SetBucketCors(ctx, iidInfo.SystemId, corsConfig)
	if err != nil {
		cblog.Error("Failed to set bucket CORS:", err)
		return false, err
	}

	cblog.Infof("Successfully set CORS for bucket %s", bucketName)

	// Wait and verify CORS configuration propagation
	maxRetries := 40 // 40 retries * 3 seconds = 120 seconds (2 minutes) max
	retryInterval := 3 * time.Second

	// Require 3 consecutive successful verifications (10 for IBM due to slower propagation)
	requiredSuccesses := 3 // Default: 3 consecutive successes
	if connInfo.ProviderName == "IBM" {
		requiredSuccesses = 10 // IBM requires more verifications due to slower CORS propagation
	}

	consecutiveSuccesses := 0

	for i := 0; i < maxRetries; i++ {
		time.Sleep(retryInterval)

		// Try to get CORS configuration
		corsConfig, verifyErr := core.GetBucketCors(ctx, iidInfo.SystemId)
		if verifyErr == nil && corsConfig != nil && len(corsConfig.CORSRules) > 0 {
			// CORS configuration exists and has rules
			consecutiveSuccesses++
			if consecutiveSuccesses >= requiredSuccesses {
				// CORS configuration successfully verified with required consecutive successes
				cblog.Infof("CORS configuration verified for bucket %s after %d retries (%d consecutive successes)", bucketName, i+1, consecutiveSuccesses)
				return true, nil
			}
		} else {
			// Reset counter on failure or if no CORS rules found
			if consecutiveSuccesses > 0 {
				cblog.Debugf("CORS verification failed (retry %d): error=%v, hasRules=%v - resetting counter from %d to 0",
					i+1, verifyErr != nil, corsConfig != nil && len(corsConfig.CORSRules) > 0, consecutiveSuccesses)
			}
			consecutiveSuccesses = 0
		}

		// Only log warning on final retry
		if i == maxRetries-1 {
			cblog.Errorf("CORS configuration not propagated for bucket %s after %d seconds (needed %d consecutive successes, got %d)", bucketName, (i+1)*3, requiredSuccesses, consecutiveSuccesses)
		}
	}

	// Verification timed out - CORS configuration not properly propagated
	return false, fmt.Errorf("CORS configuration not verified for bucket %s after %d seconds (needed %d consecutive successes, got %d)", bucketName, maxRetries*3, requiredSuccesses, consecutiveSuccesses)
}

func GetS3BucketCORS(connectionName string, bucketName string) (*cors.Config, error) {
	cblog.Info("call GetS3BucketCORS()")

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return nil, err
	}

	// Check if provider supports CORS
	if connInfo.ProviderName == "NHN" || connInfo.ProviderName == "NCP" || connInfo.ProviderName == "NCPVPC" {
		return nil, fmt.Errorf("CORS configuration is not supported by %s:%s", connectionName, connInfo.ProviderName)
	}

	// Use GCP Storage SDK for GCP
	if connInfo.ProviderName == "GCP" {
		return getGCPBucketCORS(connectionName, bucketName)
	}

	var iidInfo S3BucketIIDInfo
	err = infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return nil, err
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	core := minio.Core{Client: client}
	corsConfig, err := core.GetBucketCors(ctx, iidInfo.SystemId)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchCORSConfiguration") {
			return nil, fmt.Errorf("CORS configuration not found for bucket %s", bucketName)
		}
		cblog.Error("Failed to get bucket CORS:", err)
		return nil, err
	}

	return corsConfig, nil
}

func DeleteS3BucketCORS(connectionName string, bucketName string) (bool, error) {
	cblog.Info("call DeleteS3BucketCORS()")

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return false, err
	}

	// Check if provider supports CORS
	if connInfo.ProviderName == "NHN" || connInfo.ProviderName == "NCP" || connInfo.ProviderName == "NCPVPC" {
		return false, fmt.Errorf("CORS configuration is not supported by %s:%s", connectionName, connInfo.ProviderName)
	}

	// Use GCP Storage SDK for GCP
	if connInfo.ProviderName == "GCP" {
		return deleteGCPBucketCORS(connectionName, bucketName)
	}

	var iidInfo S3BucketIIDInfo
	err = infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return false, err
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		return false, err
	}

	ctx := context.Background()
	core := minio.Core{Client: client}

	err = core.SetBucketCors(ctx, iidInfo.SystemId, nil)
	if err != nil {
		cblog.Errorf("Failed to delete bucket CORS: %v", err)
		return false, err
	}

	cblog.Infof("Successfully deleted CORS for bucket %s", bucketName)

	// Wait and verify CORS deletion propagation
	maxRetries := 40 // 40 retries * 3 seconds = 120 seconds (2 minutes) max
	retryInterval := 3 * time.Second
	requiredSuccesses := 3 // Require 3 consecutive verifications of NoSuchCORSConfiguration
	if connInfo.ProviderName == "IBM" {
		requiredSuccesses = 10 // IBM requires more verifications due to slower CORS propagation
	}
	consecutiveSuccesses := 0

	for i := 0; i < maxRetries; i++ {
		time.Sleep(retryInterval)

		// Try to get CORS configuration - should fail with NoSuchCORSConfiguration
		corsResult, verifyErr := core.GetBucketCors(ctx, iidInfo.SystemId)

		// Check if CORS is deleted: either error contains NoSuchCORSConfiguration or result is nil/empty
		isDeleted := false
		if verifyErr != nil {
			errStr := verifyErr.Error()
			// Check for various NoSuchCORSConfiguration error formats
			if strings.Contains(errStr, "NoSuchCORSConfiguration") ||
				strings.Contains(errStr, "NoSuchCors") ||
				strings.Contains(errStr, "does not exist") {
				isDeleted = true
			}
			if i == 0 {
				cblog.Debugf("CORS deletion check error: %v", verifyErr)
			}
		} else if corsResult == nil || len(corsResult.CORSRules) == 0 {
			// No error but also no CORS rules = deleted
			isDeleted = true
		}

		if isDeleted {
			// CORS successfully deleted
			consecutiveSuccesses++
			if consecutiveSuccesses >= requiredSuccesses {
				cblog.Infof("CORS deletion verified for bucket %s after %d retries (%d seconds, %d consecutive successes)",
					bucketName, i+1, (i+1)*3, consecutiveSuccesses)
				return true, nil
			}
		} else {
			// CORS still exists or unexpected error
			if consecutiveSuccesses > 0 {
				cblog.Debugf("CORS deletion verification failed (retry %d): CORS still exists - resetting counter from %d to 0",
					i+1, consecutiveSuccesses)
			}
			consecutiveSuccesses = 0
		}

		if i == maxRetries-1 {
			cblog.Errorf("CORS deletion not verified for bucket %s after %d seconds (needed %d consecutive successes, got %d)",
				bucketName, (i+1)*3, requiredSuccesses, consecutiveSuccesses)
		}
	}

	// Verification timed out
	cblog.Errorf("CORS deletion not verified for bucket %s after %d seconds (needed %d consecutive successes, got %d)",
		bucketName, maxRetries*3, requiredSuccesses, consecutiveSuccesses)
	return false, fmt.Errorf("CORS deletion not verified for bucket %s after %d seconds (needed %d consecutive successes, got %d)",
		bucketName, maxRetries*3, requiredSuccesses, consecutiveSuccesses)
}

// DeleteS3ObjectVersion deletes a specific version of an object
func DeleteS3ObjectVersion(connectionName, bucketName, objectName, versionID string) (bool, error) {
	cblog.Info("call DeleteS3ObjectVersion()")
	cblog.Infof("Parameters - Connection: %s, Bucket: %s, Object: %s, Version: %s",
		connectionName, bucketName, objectName, versionID)

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		cblog.Errorf("Failed to get bucket info for %s: %v", bucketName, err)
		return false, err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		cblog.Errorf("Failed to get connection info: %v", err)
		return false, err
	}

	// Check if provider supports versioning
	if connInfo.ProviderName == "OPENSTACK" {
		return false, fmt.Errorf("bucket versioning is not supported by %s:%s", connectionName, connInfo.ProviderName)
	}
	if connInfo.ProviderName == "NHN" {
		return false, fmt.Errorf("bucket versioning is not supported by %s:%s (NHN Cloud Object Storage does not support versioning feature)", connectionName, connInfo.ProviderName)
	}
	if connInfo.ProviderName == "NCP" || connInfo.ProviderName == "NCPVPC" {
		return false, fmt.Errorf("bucket versioning is not supported by %s:%s (Naver Cloud Platform Object Storage does not support versioning feature)", connectionName, connInfo.ProviderName)
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		cblog.Errorf("Failed to create S3 client: %v", err)
		return false, err
	}

	ctx := context.Background()

	// Special handling for null version ID
	if versionID == "null" {
		cblog.Infof("Handling null version ID deletion")

		// List all versions to find the actual null version
		opts := minio.ListObjectsOptions{
			Prefix:       objectName,
			Recursive:    false,
			WithVersions: true,
		}

		var actualVersionId string
		var found bool

		for obj := range client.ListObjects(ctx, iidInfo.SystemId, opts) {
			if obj.Err != nil {
				cblog.Errorf("Error listing objects: %v", obj.Err)
				continue
			}

			if obj.Key == objectName && (obj.VersionID == "" || obj.VersionID == "null") {
				actualVersionId = obj.VersionID
				found = true
				break
			}
		}

		if !found {
			cblog.Errorf("Could not find null version object")
			return false, fmt.Errorf("null version object not found")
		}

		// Use the actual version ID we found
		removeOpts := minio.RemoveObjectOptions{VersionID: actualVersionId}
		err = client.RemoveObject(ctx, iidInfo.SystemId, objectName, removeOpts)
		if err != nil {
			cblog.Errorf("Failed to delete null version object: %v", err)
			return false, err
		}

		cblog.Infof("Successfully deleted null version object")
		return true, nil
	}

	// Handle normal version IDs
	opts := minio.RemoveObjectOptions{}
	if versionID != "" && versionID != "undefined" {
		opts.VersionID = versionID
		cblog.Infof("Using version ID for deletion: %s", versionID)
	} else {
		cblog.Infof("No version ID specified for deletion")
	}

	err = client.RemoveObject(ctx, iidInfo.SystemId, objectName, opts)
	if err != nil {
		cblog.Errorf("Failed to delete object version: %v", err)
		return false, err
	}

	cblog.Infof("Successfully deleted object version - Bucket: %s, Object: %s, Version: %s", bucketName, objectName, versionID)
	return true, nil
}

// DeleteMultipleObjectVersions deletes multiple object versions
func DeleteMultipleObjectVersions(connectionName, bucketName string, objects []ObjectVersionToDelete) ([]DeleteResult, error) {
	cblog.Info("call DeleteMultipleObjectVersions()")
	cblog.Infof("Parameters - Connection: %s, Bucket: %s, Objects: %d", connectionName, bucketName, len(objects))

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		cblog.Errorf("Failed to get bucket info for %s: %v", bucketName, err)
		return nil, err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		cblog.Errorf("Failed to get connection info: %v", err)
		return nil, err
	}

	// Check if provider supports versioning
	if connInfo.ProviderName == "OPENSTACK" {
		return nil, fmt.Errorf("bucket versioning is not supported by %s:%s", connectionName, connInfo.ProviderName)
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		cblog.Errorf("Failed to create S3 client: %v", err)
		return nil, err
	}

	ctx := context.Background()

	// For version-specific deletes, we need to use individual calls
	// as minio-go doesn't support bulk versioned deletes directly
	var results []DeleteResult

	for _, obj := range objects {
		cblog.Infof("Deleting object version: %s (version: %s)", obj.Key, obj.VersionID)

		if obj.VersionID == "" || obj.VersionID == "null" {
			// Delete current version (no version ID)
			err = client.RemoveObject(ctx, iidInfo.SystemId, obj.Key, minio.RemoveObjectOptions{})
		} else {
			// Delete specific version
			err = client.RemoveObject(ctx, iidInfo.SystemId, obj.Key, minio.RemoveObjectOptions{
				VersionID: obj.VersionID,
			})
		}

		if err != nil {
			cblog.Errorf("Failed to delete object %s (version %s): %v", obj.Key, obj.VersionID, err)
			results = append(results, DeleteResult{
				Key:     obj.Key,
				Success: false,
				Error:   err.Error(),
			})
		} else {
			cblog.Infof("Successfully deleted object %s (version %s)", obj.Key, obj.VersionID)
			results = append(results, DeleteResult{
				Key:     obj.Key,
				Success: true,
			})
		}
	}

	cblog.Infof("Deletion completed: %d total objects processed", len(results))
	return results, nil
}

// Helper struct for versioned delete operations
type ObjectVersionToDelete struct {
	Key       string
	VersionID string
}

// forceEmptyGCPBucket empties a GCP bucket using GCP Storage SDK
func forceEmptyGCPBucket(connectionName, bucketName string) (bool, error) {
	cblog.Info("call forceEmptyGCPBucket() - using GCP Storage SDK")

	// Get connection config to extract credential info
	cccInfo, err := ccim.GetConnectionConfig(connectionName)
	if err != nil {
		return false, fmt.Errorf("failed to get connection config: %w", err)
	}

	crdInfo, err := cim.GetCredentialDecrypt(cccInfo.CredentialName)
	if err != nil {
		return false, fmt.Errorf("failed to get credential: %w", err)
	}

	// Build GCP credentials JSON
	clientEmail := ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientEmail")
	privateKey := ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "PrivateKey")

	if clientEmail == "" || privateKey == "" {
		return false, fmt.Errorf("GCP credentials (ClientEmail, PrivateKey) not found")
	}

	credentialsJSON := map[string]string{
		"type":         "service_account",
		"private_key":  privateKey,
		"client_email": clientEmail,
		"token_uri":    "https://oauth2.googleapis.com/token",
	}

	credBytes, err := json.Marshal(credentialsJSON)
	if err != nil {
		return false, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	ctx := context.Background()

	// Create GCP storage client with credentials
	storageClient, err := storage.NewClient(ctx, option.WithCredentialsJSON(credBytes))
	if err != nil {
		return false, fmt.Errorf("failed to create GCP storage client: %w", err)
	}
	defer storageClient.Close()

	// Get bucket IID info
	var iidInfo S3BucketIIDInfo
	err = infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return false, fmt.Errorf("failed to get bucket info: %w", err)
	}

	bucket := storageClient.Bucket(iidInfo.SystemId)

	// List and delete all objects including versions
	cblog.Infof("Listing all objects in GCP bucket %s", bucketName)

	// GCP Storage SDK iterator - more reliable than minio for GCP
	query := &storage.Query{
		Versions: true, // Include all versions
	}

	it := bucket.Objects(ctx, query)
	var objectsToDelete []string
	var deletedCount int
	var errorCount int

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			cblog.Errorf("Error iterating objects: %v", err)
			return false, fmt.Errorf("failed to iterate objects: %w", err)
		}

		objectsToDelete = append(objectsToDelete, attrs.Name)
		cblog.Debugf("Found object: %s (generation: %d)", attrs.Name, attrs.Generation)
	}

	cblog.Infof("Found %d objects to delete from GCP bucket", len(objectsToDelete))

	if len(objectsToDelete) == 0 {
		cblog.Infof("GCP bucket %s is already empty", bucketName)
		return true, nil
	}

	// Delete all object versions using GCP SDK
	for i, objName := range objectsToDelete {
		cblog.Infof("Deleting object %d/%d: %s", i+1, len(objectsToDelete), objName)

		// Delete all versions of this object
		// GCP requires deleting each generation separately
		it2 := bucket.Objects(ctx, &storage.Query{
			Prefix:   objName,
			Versions: true,
		})

		for {
			attrs, err := it2.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				cblog.Errorf("Error getting object attrs: %v", err)
				break
			}

			if attrs.Name != objName {
				continue
			}

			// Delete specific generation
			obj := bucket.Object(attrs.Name).Generation(attrs.Generation)
			err = obj.Delete(ctx)
			if err != nil {
				cblog.Errorf("Failed to delete %s (gen: %d): %v", attrs.Name, attrs.Generation, err)
				errorCount++
			} else {
				cblog.Debugf("Deleted %s (gen: %d)", attrs.Name, attrs.Generation)
				deletedCount++
			}
		}

		// Small delay to avoid rate limiting
		if i%20 == 0 && i > 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	cblog.Infof("GCP bucket emptying complete: %d deleted, %d errors", deletedCount, errorCount)

	// Verify bucket is empty
	it3 := bucket.Objects(ctx, &storage.Query{Versions: true})
	remainingCount := 0
	for {
		_, err := it3.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			cblog.Errorf("Error verifying bucket emptiness: %v", err)
			break
		}
		remainingCount++
	}

	if remainingCount > 0 {
		cblog.Errorf("GCP bucket still contains %d objects after cleanup", remainingCount)
		return false, fmt.Errorf("failed to completely empty bucket: %d objects remain", remainingCount)
	}

	cblog.Infof("Successfully emptied GCP bucket %s", bucketName)
	return true, nil
}

// ForceEmptyBucket completely empties a bucket but keeps the bucket
func ForceEmptyBucket(connectionName, bucketName string) (bool, error) {
	cblog.Info("call ForceEmptyBucket()")
	cblog.Infof("Parameters - Connection: %s, Bucket: %s", connectionName, bucketName)

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return false, err
	}

	// Use GCP Storage SDK for GCP
	if connInfo.ProviderName == "GCP" {
		return forceEmptyGCPBucket(connectionName, bucketName)
	}

	var iidInfo S3BucketIIDInfo
	err = infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		cblog.Errorf("Failed to get bucket info for %s: %v", bucketName, err)
		return false, err
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		cblog.Errorf("Failed to create S3 client: %v", err)
		return false, err
	}

	// Use a longer timeout for force empty operations (5 minutes total)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Step 0: Abort all incomplete multipart uploads
	cblog.Infof("Step 0: Aborting incomplete multipart uploads in bucket %s", bucketName)

	core := &minio.Core{Client: client}
	multipartUploads, err := core.ListMultipartUploads(ctx, iidInfo.SystemId, "", "", "", "", 1000)
	if err != nil {
		cblog.Warnf("Failed to list multipart uploads (may be normal if none exist): %v", err)
		// Check for timeout
		if ctx.Err() == context.DeadlineExceeded {
			return false, fmt.Errorf("force empty operation timed out while listing multipart uploads (provider: %s may not support this operation)", connInfo.ProviderName)
		}
	} else if len(multipartUploads.Uploads) > 0 {
		cblog.Infof("Found %d incomplete multipart uploads to abort", len(multipartUploads.Uploads))
		for _, upload := range multipartUploads.Uploads {
			cblog.Infof("Aborting multipart upload: Key=%s, UploadID=%s", upload.Key, upload.UploadID)
			err := core.AbortMultipartUpload(ctx, iidInfo.SystemId, upload.Key, upload.UploadID)
			if err != nil {
				cblog.Errorf("Failed to abort multipart upload %s (ID: %s): %v", upload.Key, upload.UploadID, err)
			} else {
				cblog.Infof("Successfully aborted multipart upload: %s", upload.Key)
			}
		}
	} else {
		cblog.Infof("No incomplete multipart uploads found")
	}

	// Step 1: List all object versions and delete markers
	cblog.Infof("Step 1: Listing all object versions and delete markers in bucket %s", bucketName)

	opts := minio.ListObjectsOptions{
		Recursive:    true,
		WithVersions: true,
	}

	var allObjects []minio.ObjectInfo
	for obj := range client.ListObjects(ctx, iidInfo.SystemId, opts) {
		if obj.Err != nil {
			cblog.Errorf("Error listing object: %v", obj.Err)
			// Check for timeout
			if ctx.Err() == context.DeadlineExceeded {
				return false, fmt.Errorf("force empty operation timed out while listing objects (provider: %s may have network issues)", connInfo.ProviderName)
			}
			continue
		}
		allObjects = append(allObjects, obj)
	}

	cblog.Infof("Found %d total items (objects and delete markers) to delete", len(allObjects))

	if len(allObjects) == 0 {
		cblog.Infof("Bucket %s is already empty", bucketName)
		return true, nil
	}

	// Step 2: Delete each version individually
	deletedCount := 0
	errorCount := 0

	for i, obj := range allObjects {
		cblog.Infof("Deleting item %d/%d: Key=%s, VersionID=%s, IsDeleteMarker=%t",
			i+1, len(allObjects), obj.Key, obj.VersionID, obj.IsDeleteMarker)

		// Create remove options with version ID
		removeOpts := minio.RemoveObjectOptions{}
		if obj.VersionID != "" && obj.VersionID != "null" {
			removeOpts.VersionID = obj.VersionID
		}

		err := client.RemoveObject(ctx, iidInfo.SystemId, obj.Key, removeOpts)
		if err != nil {
			cblog.Errorf("Failed to delete object %s (version %s): %v", obj.Key, obj.VersionID, err)
			errorCount++

			// Try alternative deletion methods if the first attempt fails
			if obj.VersionID != "" && obj.VersionID != "null" {
				cblog.Infof("Trying alternative deletion method for %s", obj.Key)

				// Try deleting without version ID (this might work for some edge cases)
				err2 := client.RemoveObject(ctx, iidInfo.SystemId, obj.Key, minio.RemoveObjectOptions{})
				if err2 != nil {
					cblog.Errorf("Alternative deletion also failed for %s: %v", obj.Key, err2)
				} else {
					cblog.Infof("Alternative deletion succeeded for %s", obj.Key)
					deletedCount++
				}
			}
		} else {
			cblog.Infof("Successfully deleted %s (version %s)", obj.Key, obj.VersionID)
			deletedCount++
		}

		// Add a small delay to avoid overwhelming the API
		if i%10 == 0 && i > 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	cblog.Infof("Deletion summary: %d successful, %d errors", deletedCount, errorCount)

	// Step 3: Verify bucket is empty
	cblog.Infof("Step 3: Verifying bucket is empty")

	// Check if bucket is now empty
	remainingObjects := []minio.ObjectInfo{}
	for obj := range client.ListObjects(ctx, iidInfo.SystemId, opts) {
		if obj.Err != nil {
			cblog.Errorf("Error checking remaining objects: %v", obj.Err)
			continue
		}
		remainingObjects = append(remainingObjects, obj)
	}

	if len(remainingObjects) > 0 {
		cblog.Warnf("Bucket still contains %d objects after cleanup:", len(remainingObjects))
		for i, obj := range remainingObjects {
			if i < 5 { // Log first 5 remaining objects
				cblog.Warnf("  Remaining: Key=%s, VersionID=%s, IsDeleteMarker=%t",
					obj.Key, obj.VersionID, obj.IsDeleteMarker)
			}
		}

		// Try one more round of cleanup for remaining objects
		cblog.Infof("Attempting final cleanup round for %d remaining objects", len(remainingObjects))
		for _, obj := range remainingObjects {
			// Try multiple deletion strategies
			strategies := []minio.RemoveObjectOptions{
				{VersionID: obj.VersionID}, // With version ID
				{},                         // Without version ID
				{ForceDelete: true},        // Force delete if supported
			}

			deleted := false
			for j, strategy := range strategies {
				err := client.RemoveObject(ctx, iidInfo.SystemId, obj.Key, strategy)
				if err == nil {
					cblog.Infof("Final cleanup succeeded for %s using strategy %d", obj.Key, j+1)
					deleted = true
					break
				} else {
					cblog.Debugf("Strategy %d failed for %s: %v", j+1, obj.Key, err)
				}
			}

			if !deleted {
				cblog.Errorf("All strategies failed for %s", obj.Key)
			}
		}
	}

	// Final verification
	finalCheck := []minio.ObjectInfo{}
	for obj := range client.ListObjects(ctx, iidInfo.SystemId, opts) {
		if obj.Err != nil {
			continue
		}
		finalCheck = append(finalCheck, obj)
	}

	if len(finalCheck) > 0 {
		cblog.Errorf("Bucket still not empty after all cleanup attempts: %d objects remain", len(finalCheck))
		return false, fmt.Errorf("failed to completely empty bucket: %d objects remain", len(finalCheck))
	}

	cblog.Infof("Successfully force-emptied bucket %s (bucket preserved)", bucketName)
	return true, nil
}

// forceEmptyAndDeleteGCPBucket empties and deletes a GCP bucket using GCP Storage SDK
func forceEmptyAndDeleteGCPBucket(connectionName, bucketName string) (bool, error) {
	cblog.Info("call forceEmptyAndDeleteGCPBucket() - using GCP Storage SDK")

	// First empty the bucket
	success, err := forceEmptyGCPBucket(connectionName, bucketName)
	if err != nil {
		cblog.Errorf("Failed to empty GCP bucket %s: %v", bucketName, err)
		return false, err
	}

	if !success {
		return false, fmt.Errorf("failed to empty GCP bucket")
	}

	cblog.Infof("GCP bucket %s emptied, now deleting bucket", bucketName)

	// Get connection config
	cccInfo, err := ccim.GetConnectionConfig(connectionName)
	if err != nil {
		return false, fmt.Errorf("failed to get connection config: %w", err)
	}

	crdInfo, err := cim.GetCredentialDecrypt(cccInfo.CredentialName)
	if err != nil {
		return false, fmt.Errorf("failed to get credential: %w", err)
	}

	// Build GCP credentials JSON
	clientEmail := ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientEmail")
	privateKey := ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "PrivateKey")

	if clientEmail == "" || privateKey == "" {
		return false, fmt.Errorf("GCP credentials not found")
	}

	credentialsJSON := map[string]string{
		"type":         "service_account",
		"private_key":  privateKey,
		"client_email": clientEmail,
		"token_uri":    "https://oauth2.googleapis.com/token",
	}

	credBytes, err := json.Marshal(credentialsJSON)
	if err != nil {
		return false, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	ctx := context.Background()

	// Create GCP storage client
	storageClient, err := storage.NewClient(ctx, option.WithCredentialsJSON(credBytes))
	if err != nil {
		return false, fmt.Errorf("failed to create GCP storage client: %w", err)
	}
	defer storageClient.Close()

	// Get bucket IID info
	var iidInfo S3BucketIIDInfo
	err = infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return false, fmt.Errorf("failed to get bucket info: %w", err)
	}

	bucket := storageClient.Bucket(iidInfo.SystemId)

	// Delete the bucket - GCP will verify it's empty
	cblog.Infof("Deleting GCP bucket %s", iidInfo.SystemId)
	err = bucket.Delete(ctx)
	if err != nil {
		cblog.Errorf("Failed to delete GCP bucket: %v", err)
		return false, fmt.Errorf("failed to delete GCP bucket: %w", err)
	}

	// Remove from database
	db, err := infostore.Open()
	if err != nil {
		cblog.Errorf("Failed to open database: %v", err)
		return false, fmt.Errorf("bucket deleted but failed to update database: %w", err)
	}
	defer infostore.Close(db)

	err = db.Delete(&iidInfo).Error
	if err != nil {
		cblog.Errorf("Failed to delete bucket info from database: %v", err)
		return false, fmt.Errorf("bucket deleted but failed to update database: %w", err)
	}

	cblog.Infof("Successfully force-deleted GCP bucket %s", bucketName)
	return true, nil
}

// ForceEmptyAndDeleteBucket completely empties a bucket and deletes it
func ForceEmptyAndDeleteBucket(connectionName, bucketName string) (bool, error) {
	cblog.Info("call ForceEmptyAndDeleteBucket()")
	cblog.Infof("Parameters - Connection: %s, Bucket: %s", connectionName, bucketName)

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return false, err
	}

	// Use GCP Storage SDK for GCP
	if connInfo.ProviderName == "GCP" {
		return forceEmptyAndDeleteGCPBucket(connectionName, bucketName)
	}

	// First, empty the bucket
	success, err := ForceEmptyBucket(connectionName, bucketName)
	if err != nil {
		cblog.Errorf("Failed to empty bucket %s: %v", bucketName, err)
		return false, err
	}

	if !success {
		cblog.Errorf("Failed to empty bucket %s", bucketName)
		return false, fmt.Errorf("failed to empty bucket")
	}

	cblog.Infof("Bucket %s emptied successfully, now deleting bucket", bucketName)

	// Add a small delay for eventual consistency
	// Some CSPs (like GCP) may need time to propagate the empty state
	cblog.Infof("Waiting 2 seconds for eventual consistency before deleting bucket")
	time.Sleep(2 * time.Second)

	// Try to delete the bucket with retries for eventual consistency
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		cblog.Infof("Attempt %d/%d to delete bucket %s", attempt, maxRetries, bucketName)

		success, err = DeleteS3Bucket(connectionName, bucketName, "true")
		if err == nil && success {
			cblog.Infof("Successfully force-emptied and deleted bucket %s on attempt %d", bucketName, attempt)
			return true, nil
		}

		lastErr = err
		cblog.Warnf("Attempt %d failed to delete bucket %s: %v", attempt, bucketName, err)

		// If this is not the last attempt, wait before retrying
		if attempt < maxRetries {
			waitTime := time.Duration(attempt) * 2 * time.Second
			cblog.Infof("Waiting %v before retry %d", waitTime, attempt+1)
			time.Sleep(waitTime)
		}
	}

	cblog.Errorf("Failed to delete bucket %s after %d attempts: %v", bucketName, maxRetries, lastErr)
	return false, fmt.Errorf("bucket emptied but deletion failed after %d attempts: %v", maxRetries, lastErr)
}

func CountS3BucketsByConnection(connectionName string) (int64, error) {
	var info S3BucketIIDInfo
	count, err := infostore.CountNameIDsByConnection(&info, connectionName)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}
