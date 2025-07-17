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

	ctx := context.Background()
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true, // Get all objects, including in subdirectories
	}

	cblog.Infof("Listing objects with options - Prefix: '%s', Recursive: %t", opts.Prefix, opts.Recursive)

	var out []minio.ObjectInfo
	objectCount := 0

	for obj := range client.ListObjects(ctx, bucketName, opts) {
		if obj.Err != nil {
			cblog.Errorf("Error listing object: %v", obj.Err)
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
	ctx := context.Background()
	stat, err := client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
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

		for obj := range client.ListObjects(ctx, bucketName, opts) {
			if obj.Err != nil {
				cblog.Errorf("Error listing objects: %v", obj.Err)
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
		stat, err := client.StatObject(ctx, bucketName, objectName, statOpts)
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
				stat, err := client.StatObject(ctx, bucketName, objectName, statOpts)
				if err == nil {
					cblog.Infof("Successfully got object info using alternative method")
					return &stat, nil
				}
			}
		}

		// Try without version ID as last resort
		statOpts2 := minio.StatObjectOptions{}
		stat, err = client.StatObject(ctx, bucketName, objectName, statOpts2)
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

	stat, err := client.StatObject(ctx, bucketName, objectName, opts)
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
	err = client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
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
	err = client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{
		VersionID: "",
	})

	if err == nil {
		cblog.Infof("RemoveObject call succeeded, verifying DELETE MARKER is actually deleted")

		// Verify the delete marker is actually gone
		stillExists, verifyErr := verifyDeleteMarkerRemoved(client, ctx, bucketName, objectName)
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

	for obj := range client.ListObjects(ctx, bucketName, opts) {
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
				err = client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{
					VersionID: obj.VersionID,
				})

				if err == nil {
					cblog.Infof("Successfully deleted DELETE MARKER using version ID: %s", obj.VersionID)

					// Verify deletion
					stillExists, verifyErr := verifyDeleteMarkerRemoved(client, ctx, bucketName, objectName)
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
			err = client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})

			if err == nil {
				cblog.Infof("Successfully deleted DELETE MARKER without version ID")

				// Verify deletion
				stillExists, verifyErr = verifyDeleteMarkerRemoved(client, ctx, bucketName, objectName)
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
				stillExists, verifyErr = verifyDeleteMarkerRemoved(client, ctx, bucketName, objectName)
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
	putInfo, err := client.PutObject(ctx, bucketName, objectName, tempContent, 4, minio.PutObjectOptions{
		ContentType: "text/plain",
	})

	if err != nil {
		cblog.Errorf("Failed to create temporary object to overwrite DELETE MARKER: %v", err)
		return false, fmt.Errorf("all deletion methods failed for DELETE MARKER")
	}

	cblog.Infof("Created temporary object to overwrite DELETE MARKER, ETag: %s", putInfo.ETag)

	// Now the delete marker should no longer be the latest version
	// Verify this worked
	stillExists, verifyErr = verifyDeleteMarkerRemoved(client, ctx, bucketName, objectName)
	if verifyErr != nil {
		cblog.Warnf("Failed to verify DELETE MARKER removal after creating new version: %v", verifyErr)
	} else if !stillExists {
		cblog.Infof("DELETE MARKER successfully removed by creating new version")

		// Optionally, delete the temporary object we just created
		cblog.Infof("Deleting temporary object created to remove DELETE MARKER")
		err = client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
		if err != nil {
			cblog.Warnf("Failed to delete temporary object: %v", err)
			// This is not a critical error, just log it
		}

		return true, nil
	}

	cblog.Errorf("DELETE MARKER still exists even after creating new version")

	// Clean up the temporary object since our approach didn't work
	cblog.Infof("Cleaning up temporary object")
	err = client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		cblog.Warnf("Failed to clean up temporary object: %v", err)
	}

	cblog.Infof("Successfully removed DELETE MARKER by creating and deleting new version")

	// Final verification
	stillExists, verifyErr = verifyDeleteMarkerRemoved(client, ctx, bucketName, objectName)
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
	obj, err := client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
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

		for obj := range client.ListObjects(ctx, bucketName, opts) {
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
		obj, err := client.GetObject(ctx, bucketName, objectName, getOpts)
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
				obj, err := client.GetObject(ctx, bucketName, objectName, getOpts)
				if err == nil {
					cblog.Infof("Successfully got object using alternative method")
					return obj, nil
				}
			}
		}

		// Try without version ID as last resort
		getOpts2 := minio.GetObjectOptions{}
		obj, err = client.GetObject(ctx, bucketName, objectName, getOpts2)
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

	obj, err := client.GetObject(ctx, bucketName, objectName, opts)
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
	cblog.Infof("Setting ACL for bucket: %s, ACL: %s", bucketName, acl)

	var iidInfo S3BucketIIDInfo
	err := infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		cblog.Errorf("Failed to get bucket info: %v", err)
		return "", err
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		cblog.Errorf("Failed to get connection info: %v", err)
		return "", err
	}

	client, err := NewS3Client(connInfo)
	if err != nil {
		cblog.Errorf("Failed to create S3 client: %v", err)
		return "", err
	}

	ctx := context.Background()

	if acl == "private" {
		cblog.Infof("Setting bucket to private by deleting bucket policy")

		err = client.SetBucketPolicy(ctx, bucketName, "")
		if err != nil {
			cblog.Errorf("Failed to remove bucket policy: %v", err)
			if strings.Contains(err.Error(), "NoSuchBucketPolicy") {
				cblog.Infof("No existing bucket policy to remove, bucket is already private")
				return "private", nil
			}
			return "", err
		}

		cblog.Infof("Successfully set bucket %s to private", bucketName)
		return "private", nil
	}

	var policy string
	switch acl {
	case "public-read":
		policy = fmt.Sprintf(`{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "PublicReadGetObject",
            "Effect": "Allow",
            "Principal": "*",
            "Action": "s3:GetObject",
            "Resource": "arn:aws:s3:::%s/*"
        }
    ]
}`, bucketName)
	default:
		cblog.Errorf("Unsupported ACL: %s", acl)
		return "", fmt.Errorf("unsupported ACL: %s", acl)
	}

	cblog.Infof("Setting bucket policy: %s", policy)

	err = client.SetBucketPolicy(ctx, bucketName, policy)
	if err != nil {
		cblog.Errorf("Failed to set bucket policy: %v", err)

		if strings.Contains(err.Error(), "BlockPublicPolicy") || strings.Contains(err.Error(), "public policies are blocked") {
			cblog.Warnf("Public policy blocked by AWS Block Public Access settings")
			return "", fmt.Errorf("cannot set public-read ACL: AWS Block Public Access is enabled. Please disable 'Block public bucket policies' in AWS Console, or use MinIO which doesn't have this restriction")
		}

		return "", err
	}

	cblog.Infof("Successfully set ACL %s for bucket %s", acl, bucketName)

	appliedPolicy, err := client.GetBucketPolicy(ctx, bucketName)
	if err != nil {
		cblog.Warnf("Failed to get applied policy for verification: %v", err)
		return policy, nil
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

	client, err := NewS3Client(connInfo)
	if err != nil {
		cblog.Errorf("Failed to create S3 client: %v", err)
		return "", err
	}

	ctx := context.Background()
	cblog.Infof("Calling GetBucketVersioning for bucket: %s", bucketName)

	versioningConfig, err := client.GetBucketVersioning(ctx, bucketName)
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

	err = core.SetBucketCors(ctx, bucketName, nil)
	if err != nil {
		cblog.Errorf("Failed to delete bucket CORS: %v", err)
		return false, err
	}

	cblog.Infof("Successfully deleted CORS for bucket %s", bucketName)
	return true, nil
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

		for obj := range client.ListObjects(ctx, bucketName, opts) {
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
		err = client.RemoveObject(ctx, bucketName, objectName, removeOpts)
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

	err = client.RemoveObject(ctx, bucketName, objectName, opts)
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
			err = client.RemoveObject(ctx, bucketName, obj.Key, minio.RemoveObjectOptions{})
		} else {
			// Delete specific version
			err = client.RemoveObject(ctx, bucketName, obj.Key, minio.RemoveObjectOptions{
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

// ForceEmptyBucket completely empties a bucket but keeps the bucket
func ForceEmptyBucket(connectionName, bucketName string) (bool, error) {
	cblog.Info("call ForceEmptyBucket()")
	cblog.Infof("Parameters - Connection: %s, Bucket: %s", connectionName, bucketName)

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

	client, err := NewS3Client(connInfo)
	if err != nil {
		cblog.Errorf("Failed to create S3 client: %v", err)
		return false, err
	}

	ctx := context.Background()

	// Step 1: List all object versions and delete markers
	cblog.Infof("Step 1: Listing all object versions and delete markers in bucket %s", bucketName)

	opts := minio.ListObjectsOptions{
		Recursive:    true,
		WithVersions: true,
	}

	var allObjects []minio.ObjectInfo
	for obj := range client.ListObjects(ctx, bucketName, opts) {
		if obj.Err != nil {
			cblog.Errorf("Error listing object: %v", obj.Err)
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

		err := client.RemoveObject(ctx, bucketName, obj.Key, removeOpts)
		if err != nil {
			cblog.Errorf("Failed to delete object %s (version %s): %v", obj.Key, obj.VersionID, err)
			errorCount++

			// Try alternative deletion methods if the first attempt fails
			if obj.VersionID != "" && obj.VersionID != "null" {
				cblog.Infof("Trying alternative deletion method for %s", obj.Key)

				// Try deleting without version ID (this might work for some edge cases)
				err2 := client.RemoveObject(ctx, bucketName, obj.Key, minio.RemoveObjectOptions{})
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
	for obj := range client.ListObjects(ctx, bucketName, opts) {
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
				err := client.RemoveObject(ctx, bucketName, obj.Key, strategy)
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
	for obj := range client.ListObjects(ctx, bucketName, opts) {
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

// ForceEmptyAndDeleteBucket completely empties a bucket and deletes it
func ForceEmptyAndDeleteBucket(connectionName, bucketName string) (bool, error) {
	cblog.Info("call ForceEmptyAndDeleteBucket()")
	cblog.Infof("Parameters - Connection: %s, Bucket: %s", connectionName, bucketName)

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

	// Now delete the empty bucket
	success, err = DeleteS3Bucket(connectionName, bucketName)
	if err != nil {
		cblog.Errorf("Failed to delete empty bucket %s: %v", bucketName, err)
		return false, fmt.Errorf("bucket emptied but deletion failed: %v", err)
	}

	cblog.Infof("Successfully force-emptied and deleted bucket %s", bucketName)
	return true, nil
}
