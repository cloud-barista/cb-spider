package commonruntime

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

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
	Endpoint  string
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
}

func GetS3ConnectionInfo(connectionName string) (*S3ConnectionInfo, error) {
	// 실제 환경에서는 info-store에서 추출
	switch connectionName {
	case "aws":
		return &S3ConnectionInfo{
			Endpoint:  "s3.amazonaws.com",
			AccessKey: "YOUR_AWS_ACCESS_KEY",
			SecretKey: "YOUR_AWS_SECRET_KEY",
			Region:    "us-east-1",
			UseSSL:    true,
		}, nil
	default:
		return nil, fmt.Errorf("No S3 connection info found for: %s", connectionName)
	}
}

func NewS3Client(connInfo *S3ConnectionInfo) (*minio.Client, error) {
	return minio.New(connInfo.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(connInfo.AccessKey, connInfo.SecretKey, ""),
		Secure: connInfo.UseSSL,
		Region: connInfo.Region,
	})
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
	err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: connInfo.Region})
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

func PutS3ObjectFromFile(connectionName, bucketName, objectName, filePath string) (*minio.UploadInfo, error) {
	cblog.Info("call PutS3ObjectFromFile()")
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
	info, err := client.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{})
	if err != nil {
		return nil, err
	}
	return &info, nil
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
	return obj, nil // io.ReadCloser
}

func GetS3PresignedURL(connectionName, bucketName, objectName, method string, expiresSeconds int64) (string, error) {
	cblog.Info("call GetS3PresignedURL()")
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
	expires := time.Duration(expiresSeconds) * time.Second

	switch method {
	case "GET":
		u, err := client.PresignedGetObject(ctx, bucketName, objectName, expires, nil)
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
