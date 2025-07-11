// Cloud Control Manager's Rest Runtime of CB-Spider.
// Common Runtime for S3 Management
// by CB-Spider Team

package commonruntime

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/cors"
	"github.com/minio/minio-go/v7/pkg/credentials"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
	rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"
	infostore "github.com/cloud-barista/cb-spider/info-store"
)

type S3BucketIIDInfo struct {
	ConnectionName string `gorm:"primaryKey"`
	NameId         string `gorm:"primaryKey"`
	SystemId       string
	Region         string
	CreatedAt      time.Time
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

type S3ConnectionInfo struct {
	Endpoint       string
	AccessKey      string
	SecretKey      string
	UseSSL         bool
	RegionRequired bool
	Region         string
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

	crdInfo, err := cim.GetCredentialDecrypt(cccInfo.CredentialName)
	if err != nil {
		return nil, err
	}

	endpoint := ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "S3Endpoint")
	endpoint = strings.Replace(endpoint, "{region}", regionID, -1)
	endpoint = strings.Replace(endpoint, "{REGION}", regionID, -1)
	endpoint = strings.Replace(endpoint, "{Region}", regionID, -1)

	return &S3ConnectionInfo{
		Endpoint:       endpoint,
		AccessKey:      ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "S3AccessKey"),
		SecretKey:      ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "S3SecretKey"),
		UseSSL:         ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "S3UseSSL") == "true",
		RegionRequired: ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "S3RegionRequired") == "true",
		Region:         regionID,
	}, nil
}

func NewS3Client(connInfo *S3ConnectionInfo) (*minio.Client, error) {
	if connInfo.RegionRequired {
		return minio.New(connInfo.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(connInfo.AccessKey, connInfo.SecretKey, ""),
			Secure: connInfo.UseSSL,
			Region: connInfo.Region,
		})
	} else {
		return minio.New(connInfo.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(connInfo.AccessKey, connInfo.SecretKey, ""),
			Secure: connInfo.UseSSL,
		})
	}
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
		return nil, fmt.Errorf("S3 Bucket %s already exists", bucketName)
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	client, err := NewS3Client(connInfo)
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
		return nil, fmt.Errorf("S3 Bucket %s already exists in S3", bucketName)
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
		NameId:         bucketName,
		SystemId:       bucketName,
		Region:         connInfo.Region,
		CreatedAt:      time.Now(),
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
	return nil, fmt.Errorf("Bucket %s created, but info not found", bucketName)
}

func ListS3Buckets(connectionName string) ([]*minio.BucketInfo, error) {
	cblog.Info("call ListS3Buckets()")

	var iidInfoList []*S3BucketIIDInfo
	err := infostore.ListByCondition(&iidInfoList, "connection_name", connectionName)
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
	allBuckets, err := client.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}

	var out []*minio.BucketInfo
	for _, iid := range iidInfoList {
		for _, b := range allBuckets {
			if b.Name == iid.NameId {
				out = append(out, &b)
				break
			}
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
		if b.Name == iidInfo.NameId {
			return &b, nil
		}
	}
	return nil, fmt.Errorf("Bucket %s not found", bucketName)
}

func DeleteS3Bucket(connectionName, bucketName string) (bool, error) {
	cblog.Info("call DeleteS3Bucket()")
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
	err = client.RemoveBucket(ctx, bucketName)
	if err != nil {
		return false, err
	}
	_, err = infostore.DeleteByConditions(&S3BucketIIDInfo{}, "connection_name", iidInfo.ConnectionName, "name_id", bucketName)
	if err != nil {
		return false, err
	}
	return true, nil
}

func ListS3Objects(connectionName, bucketName, prefix string) ([]minio.ObjectInfo, error) {
	cblog.Info("call ListS3Objects()")
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
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}
	var out []minio.ObjectInfo
	for obj := range client.ListObjects(ctx, bucketName, opts) {
		if obj.Err != nil {
			continue
		}
		out = append(out, obj)
	}
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
	ctx := context.Background()
	stat, err := client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}
	return &stat, nil
}

func DeleteS3Object(connectionName, bucketName, objectName string) (bool, error) {
	cblog.Info("call DeleteS3Object()")
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
	err = client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
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
	obj, err := client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
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
		bucketName,
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

	client, err := NewS3Client(connInfo)
	if err != nil {
		return "", err
	}

	ctx := context.Background()

	core := minio.Core{Client: client}
	uploadID, err := core.NewMultipartUpload(ctx, bucketName, objectName, minio.PutObjectOptions{})
	if err != nil {
		cblog.Error("Failed to initiate multipart upload:", err)
		return "", err
	}

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

	client, err := NewS3Client(connInfo)
	if err != nil {
		return "", err
	}

	ctx := context.Background()

	core := minio.Core{Client: client}
	part, err := core.PutObjectPart(ctx, bucketName, objectName, uploadID, partNumber, reader, size, minio.PutObjectPartOptions{})
	if err != nil {
		cblog.Error("Failed to upload part:", err)
		return "", err
	}

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

	client, err := NewS3Client(connInfo)
	if err != nil {
		return "", "", err
	}

	ctx := context.Background()
	var completeParts []minio.CompletePart
	for _, part := range parts {
		completeParts = append(completeParts, minio.CompletePart{
			PartNumber: part.PartNumber,
			ETag:       part.ETag,
		})
	}

	core := minio.Core{Client: client}
	uploadInfo, err := core.CompleteMultipartUpload(ctx, bucketName, objectName, uploadID, completeParts, minio.PutObjectOptions{})
	if err != nil {
		cblog.Error("Failed to complete multipart upload:", err)
		return "", "", err
	}

	location := fmt.Sprintf("/%s/%s", bucketName, objectName)
	return location, uploadInfo.ETag, nil
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
	for err := range client.RemoveObjects(ctx, bucketName, objectsCh, minio.RemoveObjectsOptions{}) {
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
	cblog.Info("call GetS3PresignedURL()")
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

	switch method {
	case "GET":
		var params url.Values
		if responseContentDisposition != "" {
			params = url.Values{}
			params.Set("response-content-disposition", responseContentDisposition)
		}
		u, err := client.PresignedGetObject(ctx, bucketName, objectName, expires, params)
		if err != nil {
			return "", err
		}
		return u.String(), nil

	case "PUT":
		u, err := client.PresignedPutObject(ctx, bucketName, objectName, expires)
		if err != nil {
			return "", err
		}
		return u.String(), nil

	default:
		return "", fmt.Errorf("Unsupported method: %s", method)
	}
}

func SetS3BucketACL(connectionName string, bucketName string, acl string) (string, error) {
	cblog.Info("call SetS3BucketACL()")
	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
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

	var policy string
	switch acl {
	case "public-read":
		policy = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":["s3:GetObject"],"Resource":["arn:aws:s3:::` + bucketName + `/*"]}]}`
	case "private":
		policy = `{"Version":"2012-10-17","Statement":[]}`
	default:
		return "", fmt.Errorf("unsupported ACL: %s", acl)
	}
	err = client.SetBucketPolicy(ctx, bucketName, policy)
	if err != nil {
		return "", err
	}
	appliedPolicy, err := client.GetBucketPolicy(ctx, bucketName)
	if err != nil {
		return "", err
	}
	return appliedPolicy, nil
}

func GetS3BucketACL(connectionName string, bucketName string) (string, error) {
	cblog.Info("call GetS3BucketACL()")
	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
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
	policy, err := client.GetBucketPolicy(ctx, bucketName)
	if err != nil {
		return "", err
	}
	return policy, nil
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
	client, err := NewS3Client(connInfo)
	if err != nil {
		return false, err
	}
	ctx := context.Background()
	opts := minio.BucketVersioningConfiguration{
		Status: "Enabled",
	}

	err = client.SetBucketVersioning(ctx, bucketName, opts)
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
	client, err := NewS3Client(connInfo)
	if err != nil {
		return false, err
	}
	ctx := context.Background()
	opts := minio.BucketVersioningConfiguration{
		Status: "Suspended",
	}
	err = client.SetBucketVersioning(ctx, bucketName, opts)
	if err != nil {
		return false, err
	}
	return true, nil
}

func ListS3ObjectVersions(connectionName string, bucketName string, prefix string) ([]minio.ObjectInfo, error) {
	cblog.Info("call ListS3ObjectVersions()")
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
	opts := minio.ListObjectsOptions{
		Prefix:       prefix,
		Recursive:    true,
		WithVersions: true,
	}

	var out []minio.ObjectInfo
	for obj := range client.ListObjects(ctx, bucketName, opts) {
		if obj.Err != nil {
			continue
		}
		out = append(out, obj)
	}
	return out, nil
}

func SetS3BucketCORS(connectionName string, bucketName string, allowedOrigins []string, allowedMethods []string, allowedHeaders []string, exposeHeaders []string, maxAgeSeconds int) (bool, error) {
	cblog.Info("call SetS3BucketCORS()")

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
	err = core.SetBucketCors(ctx, bucketName, corsConfig)
	if err != nil {
		cblog.Error("Failed to set bucket CORS:", err)
		return false, err
	}

	cblog.Infof("Successfully set CORS for bucket %s", bucketName)
	return true, nil
}

func GetS3BucketCORS(connectionName string, bucketName string) (*cors.Config, error) {
	cblog.Info("call GetS3BucketCORS()")

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

	core := minio.Core{Client: client}
	corsConfig, err := core.GetBucketCors(ctx, bucketName)
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

	core := minio.Core{Client: client}
	emptyCORS := &cors.Config{
		CORSRules: []cors.Rule{},
	}
	err = core.SetBucketCors(ctx, bucketName, emptyCORS)
	if err != nil {
		cblog.Error("Failed to delete bucket CORS:", err)
		return false, err
	}

	cblog.Infof("Successfully deleted CORS for bucket %s", bucketName)
	return true, nil
}
