// Azure Blob Storage specific implementations for S3 operations.
// Uses Azure Go SDK (azblob) instead of S3/minio API.
// by CB-Spider Team

package commonruntime

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/streaming"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/cors"
	"github.com/rs/xid"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
	infostore "github.com/cloud-barista/cb-spider/info-store"
)

// ============================================================================
// Azure Blob Storage Client Helpers
// ============================================================================

// newAzureBlobClient creates an Azure Blob Storage client using SharedKeyCredential.
func newAzureBlobClient(connInfo *S3ConnectionInfo) (*azblob.Client, *azblob.SharedKeyCredential, error) {
	sharedKeyCred, err := azblob.NewSharedKeyCredential(connInfo.AccessKey, connInfo.SecretKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Azure SharedKeyCredential: %w", err)
	}

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", connInfo.AccessKey)
	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, sharedKeyCred, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Azure Blob client: %w", err)
	}

	return client, sharedKeyCred, nil
}

// AzureManagementInfo holds Azure management plane credentials.
type AzureManagementInfo struct {
	SubscriptionID     string
	TenantID           string
	ClientID           string
	ClientSecret       string
	StorageAccountName string
}

// getAzureManagementInfo retrieves Azure SP credentials and storage account info
// from the connection configuration for ARM (management plane) operations.
func getAzureManagementInfo(connectionName string) (*AzureManagementInfo, error) {
	cccInfo, err := ccim.GetConnectionConfig(connectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection config: %w", err)
	}

	crdInfo, err := cim.GetCredentialDecrypt(cccInfo.CredentialName)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}

	info := &AzureManagementInfo{
		ClientID:           ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientId"),
		ClientSecret:       ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "ClientSecret"),
		TenantID:           ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "TenantId"),
		SubscriptionID:     ccm.KeyValueListGetValue(crdInfo.KeyValueInfoList, "SubscriptionId"),
		StorageAccountName: getAccessKey(crdInfo.KeyValueInfoList, "S3AccessKey", "StorageAccountName"),
	}

	if info.ClientID == "" || info.ClientSecret == "" || info.TenantID == "" || info.SubscriptionID == "" {
		return nil, fmt.Errorf("Azure management credentials (ClientId, ClientSecret, TenantId, SubscriptionId) are required for this operation")
	}

	return info, nil
}

// getAzureResourceGroup finds the resource group for a given storage account
// by listing all storage accounts in the subscription.
func getAzureResourceGroup(mgmtInfo *AzureManagementInfo) (string, error) {
	cred, err := azidentity.NewClientSecretCredential(mgmtInfo.TenantID, mgmtInfo.ClientID, mgmtInfo.ClientSecret, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create Azure credential: %w", err)
	}

	accountsClient, err := armstorage.NewAccountsClient(mgmtInfo.SubscriptionID, cred, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create storage accounts client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pager := accountsClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to list storage accounts: %w", err)
		}
		for _, account := range page.Value {
			if account.Name != nil && *account.Name == mgmtInfo.StorageAccountName {
				// Parse resource group from ID:
				// /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}
				if account.ID != nil {
					parts := strings.Split(*account.ID, "/")
					for i, part := range parts {
						if strings.EqualFold(part, "resourceGroups") && i+1 < len(parts) {
							return parts[i+1], nil
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("storage account '%s' not found in subscription '%s'", mgmtInfo.StorageAccountName, mgmtInfo.SubscriptionID)
}

// azureBlockID generates a base64-encoded block ID from upload ID and part number.
// All block IDs for a blob must be the same length (before base64 encoding).
func azureBlockID(uploadID string, partNumber int) string {
	// Use 8 chars from uploadID + 6-digit part number = 14 chars fixed length
	clean := strings.ReplaceAll(uploadID, "-", "")
	raw := fmt.Sprintf("%.8s%06d", clean, partNumber)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// stripAzureETagQuotes removes surrounding quotes from Azure ETags for consistency.
func stripAzureETagQuotes(etag string) string {
	return strings.Trim(etag, "\"")
}

// ============================================================================
// Bucket (Container) Operations
// ============================================================================

func createAzureBucket(connectionName, bucketName string, connInfo *S3ConnectionInfo) (*minio.BucketInfo, error) {
	cblog.Infof("createAzureBucket: Creating container '%s' in Azure Storage Account '%s'", bucketName, connInfo.AccessKey)

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = client.CreateContainer(ctx, bucketName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure container '%s': %w", bucketName, err)
	}

	cblog.Infof("Successfully created Azure container '%s'", bucketName)

	return &minio.BucketInfo{
		Name:         bucketName,
		CreationDate: time.Now(),
	}, nil
}

func listAzureBuckets(connInfo *S3ConnectionInfo, iidInfoList []*S3BucketIIDInfo) ([]*minio.BucketInfo, error) {
	cblog.Info("listAzureBuckets: Listing Azure containers")

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build a map of SystemId -> NameId for quick lookup
	systemToName := make(map[string]string)
	for _, iid := range iidInfoList {
		systemToName[iid.SystemId] = iid.NameId
	}

	// List all containers from Azure
	containerMap := make(map[string]time.Time)
	pager := client.NewListContainersPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list Azure containers: %w", err)
		}
		for _, item := range page.ContainerItems {
			if item.Name != nil {
				lastModified := time.Time{}
				if item.Properties != nil && item.Properties.LastModified != nil {
					lastModified = *item.Properties.LastModified
				}
				containerMap[*item.Name] = lastModified
			}
		}
	}

	var out []*minio.BucketInfo
	for _, iid := range iidInfoList {
		if lastModified, found := containerMap[iid.SystemId]; found {
			out = append(out, &minio.BucketInfo{
				Name:         iid.NameId,
				CreationDate: lastModified,
			})
		} else {
			cblog.Warnf("Container '%s' (SystemId: %s) exists in metadata but not found in Azure", iid.NameId, iid.SystemId)
			out = append(out, &minio.BucketInfo{
				Name:         iid.NameId,
				CreationDate: time.Time{},
			})
		}
	}

	return out, nil
}

func listAzureBucketsWithIID(connInfo *S3ConnectionInfo, iidInfoList []*S3BucketIIDInfo) ([]*S3BucketWithIID, error) {
	cblog.Info("listAzureBucketsWithIID: Listing Azure containers with IID")

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	containerMap := make(map[string]time.Time)
	pager := client.NewListContainersPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list Azure containers: %w", err)
		}
		for _, item := range page.ContainerItems {
			if item.Name != nil {
				lastModified := time.Time{}
				if item.Properties != nil && item.Properties.LastModified != nil {
					lastModified = *item.Properties.LastModified
				}
				containerMap[*item.Name] = lastModified
			}
		}
	}

	var out []*S3BucketWithIID
	for _, iid := range iidInfoList {
		if lastModified, found := containerMap[iid.SystemId]; found {
			out = append(out, &S3BucketWithIID{
				NameId:       iid.NameId,
				SystemId:     iid.SystemId,
				CreationDate: lastModified,
			})
		} else {
			cblog.Warnf("Container '%s' (SystemId: %s) exists in metadata but not found in Azure", iid.NameId, iid.SystemId)
			out = append(out, &S3BucketWithIID{
				NameId:       iid.NameId,
				SystemId:     iid.SystemId,
				CreationDate: time.Time{},
			})
		}
	}

	return out, nil
}

func getAzureBucket(connInfo *S3ConnectionInfo, iidInfo *S3BucketIIDInfo) (*minio.BucketInfo, error) {
	cblog.Infof("getAzureBucket: Getting container '%s' from Azure", iidInfo.SystemId)

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get container properties
	containerClient := client.ServiceClient().NewContainerClient(iidInfo.SystemId)
	props, err := containerClient.GetProperties(ctx, nil)
	if err != nil {
		cblog.Warnf("Container '%s' not found in Azure: %v", iidInfo.SystemId, err)
		return &minio.BucketInfo{
			Name:         iidInfo.NameId,
			CreationDate: time.Time{},
		}, nil
	}

	creationDate := time.Time{}
	if props.LastModified != nil {
		creationDate = *props.LastModified
	}

	return &minio.BucketInfo{
		Name:         iidInfo.NameId,
		CreationDate: creationDate,
	}, nil
}

func deleteAzureBucket(connInfo *S3ConnectionInfo, systemId string) error {
	cblog.Infof("deleteAzureBucket: Deleting container '%s' from Azure", systemId)

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = client.DeleteContainer(ctx, systemId, nil)
	if err != nil {
		return fmt.Errorf("failed to delete Azure container '%s': %w", systemId, err)
	}

	cblog.Infof("Successfully deleted Azure container '%s'", systemId)
	return nil
}

// ============================================================================
// Object (Blob) Operations
// ============================================================================

func listAzureObjects(connInfo *S3ConnectionInfo, bucketName, prefix string) ([]minio.ObjectInfo, error) {
	cblog.Infof("listAzureObjects: Listing blobs in container '%s' with prefix '%s'", bucketName, prefix)

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	opts := &azblob.ListBlobsFlatOptions{}
	if prefix != "" {
		opts.Prefix = to.Ptr(prefix)
	}

	var out []minio.ObjectInfo
	pager := client.NewListBlobsFlatPager(bucketName, opts)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list blobs in Azure container '%s': %w", bucketName, err)
		}
		for _, item := range page.Segment.BlobItems {
			objInfo := azureBlobItemToObjectInfo(item)
			out = append(out, objInfo)
		}
	}

	cblog.Infof("Found %d blobs in Azure container '%s'", len(out), bucketName)
	return out, nil
}

func azureBlobItemToObjectInfo(item *container.BlobItem) minio.ObjectInfo {
	info := minio.ObjectInfo{}
	if item.Name != nil {
		info.Key = *item.Name
	}
	if item.Properties != nil {
		if item.Properties.ContentLength != nil {
			info.Size = *item.Properties.ContentLength
		}
		if item.Properties.LastModified != nil {
			info.LastModified = *item.Properties.LastModified
		}
		if item.Properties.ETag != nil {
			info.ETag = stripAzureETagQuotes(string(*item.Properties.ETag))
		}
		if item.Properties.ContentType != nil {
			info.ContentType = *item.Properties.ContentType
		}
	}
	if item.VersionID != nil {
		info.VersionID = *item.VersionID
	}
	if item.IsCurrentVersion != nil {
		info.IsLatest = *item.IsCurrentVersion
	}
	return info
}

func getAzureObjectInfo(connInfo *S3ConnectionInfo, bucketName, objectName string) (*minio.ObjectInfo, error) {
	cblog.Infof("getAzureObjectInfo: Getting blob '%s' from container '%s'", objectName, bucketName)

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	blobClient := client.ServiceClient().NewContainerClient(bucketName).NewBlobClient(objectName)
	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob properties for '%s/%s': %w", bucketName, objectName, err)
	}

	info := &minio.ObjectInfo{
		Key: objectName,
	}
	if props.ContentLength != nil {
		info.Size = *props.ContentLength
	}
	if props.LastModified != nil {
		info.LastModified = *props.LastModified
	}
	if props.ETag != nil {
		info.ETag = stripAzureETagQuotes(string(*props.ETag))
	}
	if props.ContentType != nil {
		info.ContentType = *props.ContentType
	}
	if props.VersionID != nil {
		info.VersionID = *props.VersionID
	}

	return info, nil
}

func getAzureObjectInfoWithVersion(connInfo *S3ConnectionInfo, bucketName, objectName, versionId string) (*minio.ObjectInfo, error) {
	cblog.Infof("getAzureObjectInfoWithVersion: Getting blob '%s' version '%s' from container '%s'", objectName, versionId, bucketName)

	_, sharedKeyCred, err := newAzureBlobClient(connInfo)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a versioned blob client by appending versionid to URL
	blobURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?versionid=%s",
		connInfo.AccessKey, bucketName, url.PathEscape(objectName), url.QueryEscape(versionId))
	blobClient, err := blob.NewClientWithSharedKeyCredential(blobURL, sharedKeyCred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create versioned blob client: %w", err)
	}

	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get versioned blob properties for '%s/%s' (version: %s): %w", bucketName, objectName, versionId, err)
	}

	info := &minio.ObjectInfo{
		Key:       objectName,
		VersionID: versionId,
	}
	if props.ContentLength != nil {
		info.Size = *props.ContentLength
	}
	if props.LastModified != nil {
		info.LastModified = *props.LastModified
	}
	if props.ETag != nil {
		info.ETag = stripAzureETagQuotes(string(*props.ETag))
	}
	if props.ContentType != nil {
		info.ContentType = *props.ContentType
	}

	return info, nil
}

func deleteAzureObject(connInfo *S3ConnectionInfo, bucketName, objectName string) error {
	cblog.Infof("deleteAzureObject: Deleting blob '%s' from container '%s'", objectName, bucketName)

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = client.DeleteBlob(ctx, bucketName, objectName, nil)
	if err != nil {
		return fmt.Errorf("failed to delete blob '%s/%s': %w", bucketName, objectName, err)
	}

	cblog.Infof("Successfully deleted blob '%s/%s'", bucketName, objectName)
	return nil
}

func deleteAzureObjectVersion(connInfo *S3ConnectionInfo, bucketName, objectName, versionID string) (bool, error) {
	cblog.Infof("deleteAzureObjectVersion: Deleting blob '%s' version '%s' from container '%s'", objectName, versionID, bucketName)

	_, sharedKeyCred, err := newAzureBlobClient(connInfo)
	if err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	blobURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?versionid=%s",
		connInfo.AccessKey, bucketName, url.PathEscape(objectName), url.QueryEscape(versionID))
	blobClient, err := blob.NewClientWithSharedKeyCredential(blobURL, sharedKeyCred, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create versioned blob client: %w", err)
	}

	_, err = blobClient.Delete(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("failed to delete blob version '%s/%s' (version: %s): %w", bucketName, objectName, versionID, err)
	}

	cblog.Infof("Successfully deleted blob version '%s/%s' (version: %s)", bucketName, objectName, versionID)
	return true, nil
}

func deleteMultipleAzureObjects(connInfo *S3ConnectionInfo, bucketName string, objectNames []string) ([]DeleteResult, error) {
	cblog.Infof("deleteMultipleAzureObjects: Deleting %d blobs from container '%s'", len(objectNames), bucketName)

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	var results []DeleteResult

	for _, objectName := range objectNames {
		_, err := client.DeleteBlob(ctx, bucketName, objectName, nil)
		if err != nil {
			results = append(results, DeleteResult{
				Key:     objectName,
				Success: false,
				Error:   err.Error(),
			})
		} else {
			results = append(results, DeleteResult{
				Key:     objectName,
				Success: true,
			})
		}
	}

	return results, nil
}

func deleteMultipleAzureObjectVersions(connInfo *S3ConnectionInfo, bucketName string, objects []ObjectVersionToDelete) ([]DeleteResult, error) {
	cblog.Infof("deleteMultipleAzureObjectVersions: Deleting %d blob versions from container '%s'", len(objects), bucketName)

	var results []DeleteResult

	for _, obj := range objects {
		var err error
		if obj.VersionID == "" || obj.VersionID == "null" {
			err = deleteAzureObject(connInfo, bucketName, obj.Key)
		} else {
			_, err = deleteAzureObjectVersion(connInfo, bucketName, obj.Key, obj.VersionID)
		}

		if err != nil {
			results = append(results, DeleteResult{
				Key:     obj.Key,
				Success: false,
				Error:   err.Error(),
			})
		} else {
			results = append(results, DeleteResult{
				Key:     obj.Key,
				Success: true,
			})
		}
	}

	return results, nil
}

// ============================================================================
// Stream Operations (Upload/Download)
// ============================================================================

func getAzureObjectStream(connInfo *S3ConnectionInfo, bucketName, objectName string) (io.ReadCloser, error) {
	cblog.Infof("getAzureObjectStream: Downloading blob '%s' from container '%s'", objectName, bucketName)

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := client.DownloadStream(ctx, bucketName, objectName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download blob '%s/%s': %w", bucketName, objectName, err)
	}

	return resp.Body, nil
}

func getAzureObjectStreamWithVersion(connInfo *S3ConnectionInfo, bucketName, objectName, versionId string) (io.ReadCloser, error) {
	cblog.Infof("getAzureObjectStreamWithVersion: Downloading blob '%s' version '%s' from container '%s'", objectName, versionId, bucketName)

	_, sharedKeyCred, err := newAzureBlobClient(connInfo)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	blobURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?versionid=%s",
		connInfo.AccessKey, bucketName, url.PathEscape(objectName), url.QueryEscape(versionId))
	blobClient, err := blob.NewClientWithSharedKeyCredential(blobURL, sharedKeyCred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create versioned blob client: %w", err)
	}

	resp, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download versioned blob '%s/%s' (version: %s): %w", bucketName, objectName, versionId, err)
	}

	return resp.Body, nil
}

func putAzureObject(connInfo *S3ConnectionInfo, bucketName, objectName string, reader io.Reader, objectSize int64) (minio.UploadInfo, error) {
	cblog.Infof("putAzureObject: Uploading blob '%s' to container '%s' (size: %d)", objectName, bucketName, objectSize)

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return minio.UploadInfo{}, err
	}

	ctx := context.Background()
	resp, err := client.UploadStream(ctx, bucketName, objectName, reader, nil)
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to upload blob '%s/%s': %w", bucketName, objectName, err)
	}

	info := minio.UploadInfo{
		Bucket:   bucketName,
		Key:      objectName,
		Size:     objectSize,
		Location: fmt.Sprintf("/%s/%s", bucketName, objectName),
	}
	if resp.ETag != nil {
		info.ETag = stripAzureETagQuotes(string(*resp.ETag))
	}
	if resp.LastModified != nil {
		info.LastModified = *resp.LastModified
	}
	if resp.VersionID != nil {
		info.VersionID = *resp.VersionID
	}

	cblog.Infof("Successfully uploaded blob '%s/%s', ETag: %s", bucketName, objectName, info.ETag)
	return info, nil
}

func getAzureBucketTotalSize(connInfo *S3ConnectionInfo, bucketName string) (int64, int64, error) {
	cblog.Infof("getAzureBucketTotalSize: Calculating total size of container '%s'", bucketName)

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return 0, 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var totalSize int64
	var totalCount int64

	pager := client.NewListBlobsFlatPager(bucketName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to list blobs for size calculation: %w", err)
		}
		for _, item := range page.Segment.BlobItems {
			totalCount++
			if item.Properties != nil && item.Properties.ContentLength != nil {
				totalSize += *item.Properties.ContentLength
			}
		}
	}

	cblog.Infof("Container '%s': %d objects, total size: %d bytes", bucketName, totalCount, totalSize)
	return totalSize, totalCount, nil
}

// ============================================================================
// Block Blob (Multipart Upload) Operations
// ============================================================================

func initiateAzureMultipartUpload(connInfo *S3ConnectionInfo, bucketName, objectName string) (string, error) {
	cblog.Infof("initiateAzureMultipartUpload: Initiating block blob upload for '%s/%s'", bucketName, objectName)

	// Azure doesn't require explicit initiation of block blob uploads.
	// Generate a unique upload ID for API compatibility.
	uploadID := xid.New().String()

	cblog.Infof("Generated Azure multipart upload ID: %s (used for block ID generation)", uploadID)
	return uploadID, nil
}

func uploadAzurePart(connInfo *S3ConnectionInfo, bucketName, objectName, uploadID string, partNumber int, reader io.Reader, size int64) (string, error) {
	cblog.Infof("uploadAzurePart: Staging block %d for '%s/%s' (upload: %s, size: %d)", partNumber, bucketName, objectName, uploadID, size)

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Read the part data into memory for ReadSeekCloser requirement
	data, err := io.ReadAll(io.LimitReader(reader, size))
	if err != nil {
		return "", fmt.Errorf("failed to read part data: %w", err)
	}

	blockID := azureBlockID(uploadID, partNumber)
	body := streaming.NopCloser(bytes.NewReader(data))

	bbClient := client.ServiceClient().NewContainerClient(bucketName).NewBlockBlobClient(objectName)
	_, err = bbClient.StageBlock(ctx, blockID, body, nil)
	if err != nil {
		return "", fmt.Errorf("failed to stage block %d for '%s/%s': %w", partNumber, bucketName, objectName, err)
	}

	// Azure StageBlock doesn't return an ETag per block, so we compute an MD5-like identifier
	// Use the block ID as the ETag equivalent for API compatibility
	etag := blockID

	cblog.Infof("Successfully staged block %d (blockID: %s)", partNumber, blockID)
	return etag, nil
}

func completeAzureMultipartUpload(connInfo *S3ConnectionInfo, bucketName, objectName, uploadID string, parts []CompletePart) (string, string, error) {
	cblog.Infof("completeAzureMultipartUpload: Committing %d blocks for '%s/%s'", len(parts), bucketName, objectName)

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return "", "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Reconstruct block IDs from parts
	blockIDs := make([]string, len(parts))
	for i, part := range parts {
		blockIDs[i] = azureBlockID(uploadID, part.PartNumber)
	}

	bbClient := client.ServiceClient().NewContainerClient(bucketName).NewBlockBlobClient(objectName)
	resp, err := bbClient.CommitBlockList(ctx, blockIDs, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to commit block list for '%s/%s': %w", bucketName, objectName, err)
	}

	etag := ""
	if resp.ETag != nil {
		etag = stripAzureETagQuotes(string(*resp.ETag))
	}

	location := fmt.Sprintf("/%s/%s", bucketName, objectName)
	cblog.Infof("Successfully committed block list for '%s/%s', ETag: %s", bucketName, objectName, etag)
	return location, etag, nil
}

func abortAzureMultipartUpload(connInfo *S3ConnectionInfo, bucketName, objectName, uploadID string) error {
	cblog.Infof("abortAzureMultipartUpload: Aborting block blob upload for '%s/%s' (upload: %s)", bucketName, objectName, uploadID)

	// Azure automatically cleans up uncommitted blocks after 7 days.
	// There's no explicit abort operation.
	// We can try to delete the blob if it exists with uncommitted blocks, but this is best-effort.
	cblog.Infof("Azure does not require explicit abort - uncommitted blocks expire automatically after 7 days")
	return nil
}

func listAzureParts(connInfo *S3ConnectionInfo, bucketName, objectName, uploadID string, partNumberMarker, maxParts int) (*ListPartsResult, error) {
	cblog.Infof("listAzureParts: Listing blocks for '%s/%s'", bucketName, objectName)

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bbClient := client.ServiceClient().NewContainerClient(bucketName).NewBlockBlobClient(objectName)
	resp, err := bbClient.GetBlockList(ctx, blockblob.BlockListTypeUncommitted, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get block list for '%s/%s': %w", bucketName, objectName, err)
	}

	result := &ListPartsResult{
		Bucket:   bucketName,
		Key:      objectName,
		UploadID: uploadID,
	}

	// Filter blocks by upload ID prefix
	uploadPrefix := strings.ReplaceAll(uploadID, "-", "")
	if len(uploadPrefix) > 8 {
		uploadPrefix = uploadPrefix[:8]
	}

	if resp.UncommittedBlocks != nil {
		for _, block := range resp.UncommittedBlocks {
			if block.Name == nil {
				continue
			}

			// Decode block ID to check upload prefix
			decoded, err := base64.StdEncoding.DecodeString(*block.Name)
			if err != nil {
				continue
			}
			decodedStr := string(decoded)
			if !strings.HasPrefix(decodedStr, uploadPrefix) {
				continue
			}

			// Extract part number from block ID
			partNumStr := decodedStr[8:]
			partNum := 0
			fmt.Sscanf(partNumStr, "%d", &partNum)

			blockSize := int64(0)
			if block.Size != nil {
				blockSize = *block.Size
			}

			result.Parts = append(result.Parts, PartInfo{
				PartNumber: partNum,
				ETag:       *block.Name,
				Size:       blockSize,
			})
		}
	}

	return result, nil
}

// ============================================================================
// Versioning Operations
// ============================================================================

func enableAzureVersioning(connectionName string, connInfo *S3ConnectionInfo) (bool, error) {
	cblog.Info("enableAzureVersioning: Enabling blob versioning on Azure storage account")

	mgmtInfo, err := getAzureManagementInfo(connectionName)
	if err != nil {
		return false, fmt.Errorf("failed to get Azure management info: %w", err)
	}

	resourceGroup, err := getAzureResourceGroup(mgmtInfo)
	if err != nil {
		return false, fmt.Errorf("failed to find resource group: %w", err)
	}

	cred, err := azidentity.NewClientSecretCredential(mgmtInfo.TenantID, mgmtInfo.ClientID, mgmtInfo.ClientSecret, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	blobSvcClient, err := armstorage.NewBlobServicesClient(mgmtInfo.SubscriptionID, cred, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create blob services client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	properties := armstorage.BlobServiceProperties{
		BlobServiceProperties: &armstorage.BlobServicePropertiesProperties{
			IsVersioningEnabled: to.Ptr(true),
		},
	}

	_, err = blobSvcClient.SetServiceProperties(ctx, resourceGroup, mgmtInfo.StorageAccountName, properties, nil)
	if err != nil {
		return false, fmt.Errorf("failed to enable versioning on Azure storage account '%s': %w", mgmtInfo.StorageAccountName, err)
	}

	cblog.Infof("Successfully enabled versioning on Azure storage account '%s' (NOTE: affects all containers in this storage account)", mgmtInfo.StorageAccountName)
	return true, nil
}

func suspendAzureVersioning(connectionName string, connInfo *S3ConnectionInfo) (bool, error) {
	cblog.Info("suspendAzureVersioning: Disabling blob versioning on Azure storage account")

	mgmtInfo, err := getAzureManagementInfo(connectionName)
	if err != nil {
		return false, fmt.Errorf("failed to get Azure management info: %w", err)
	}

	resourceGroup, err := getAzureResourceGroup(mgmtInfo)
	if err != nil {
		return false, fmt.Errorf("failed to find resource group: %w", err)
	}

	cred, err := azidentity.NewClientSecretCredential(mgmtInfo.TenantID, mgmtInfo.ClientID, mgmtInfo.ClientSecret, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	blobSvcClient, err := armstorage.NewBlobServicesClient(mgmtInfo.SubscriptionID, cred, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create blob services client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	properties := armstorage.BlobServiceProperties{
		BlobServiceProperties: &armstorage.BlobServicePropertiesProperties{
			IsVersioningEnabled: to.Ptr(false),
		},
	}

	_, err = blobSvcClient.SetServiceProperties(ctx, resourceGroup, mgmtInfo.StorageAccountName, properties, nil)
	if err != nil {
		return false, fmt.Errorf("failed to suspend versioning on Azure storage account '%s': %w", mgmtInfo.StorageAccountName, err)
	}

	cblog.Infof("Successfully suspended versioning on Azure storage account '%s' (NOTE: affects all containers in this storage account)", mgmtInfo.StorageAccountName)
	return true, nil
}

func getAzureVersioning(connectionName string, connInfo *S3ConnectionInfo) (string, error) {
	cblog.Info("getAzureVersioning: Getting blob versioning status from Azure storage account")

	mgmtInfo, err := getAzureManagementInfo(connectionName)
	if err != nil {
		return "", fmt.Errorf("failed to get Azure management info: %w", err)
	}

	resourceGroup, err := getAzureResourceGroup(mgmtInfo)
	if err != nil {
		return "", fmt.Errorf("failed to find resource group: %w", err)
	}

	cred, err := azidentity.NewClientSecretCredential(mgmtInfo.TenantID, mgmtInfo.ClientID, mgmtInfo.ClientSecret, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create Azure credential: %w", err)
	}

	blobSvcClient, err := armstorage.NewBlobServicesClient(mgmtInfo.SubscriptionID, cred, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create blob services client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := blobSvcClient.GetServiceProperties(ctx, resourceGroup, mgmtInfo.StorageAccountName, nil)
	if err != nil {
		return "Suspended", nil
	}

	if resp.BlobServiceProperties.BlobServiceProperties != nil &&
		resp.BlobServiceProperties.BlobServiceProperties.IsVersioningEnabled != nil &&
		*resp.BlobServiceProperties.BlobServiceProperties.IsVersioningEnabled {
		return "Enabled", nil
	}

	return "Suspended", nil
}

func listAzureObjectVersions(connInfo *S3ConnectionInfo, bucketName, prefix string) ([]minio.ObjectInfo, error) {
	cblog.Infof("listAzureObjectVersions: Listing blob versions in container '%s' with prefix '%s'", bucketName, prefix)

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	opts := &azblob.ListBlobsFlatOptions{
		Include: azblob.ListBlobsInclude{
			Versions: true,
		},
	}
	if prefix != "" {
		opts.Prefix = to.Ptr(prefix)
	}

	var out []minio.ObjectInfo
	pager := client.NewListBlobsFlatPager(bucketName, opts)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list blob versions in Azure container '%s': %w", bucketName, err)
		}
		for _, item := range page.Segment.BlobItems {
			objInfo := azureBlobItemToObjectInfo(item)
			out = append(out, objInfo)
		}
	}

	cblog.Infof("Found %d blob versions in Azure container '%s'", len(out), bucketName)
	return out, nil
}

// ============================================================================
// CORS Operations
// ============================================================================

func setAzureBucketCORS(connInfo *S3ConnectionInfo, allowedOrigins []string, allowedMethods []string, allowedHeaders []string, exposeHeaders []string, maxAgeSeconds int) (bool, error) {
	cblog.Info("setAzureBucketCORS: Setting CORS on Azure storage account")

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Azure CORS rules use comma-separated strings
	corsRule := &service.CORSRule{
		AllowedOrigins:  to.Ptr(strings.Join(allowedOrigins, ",")),
		AllowedMethods:  to.Ptr(strings.Join(allowedMethods, ",")),
		AllowedHeaders:  to.Ptr(strings.Join(allowedHeaders, ",")),
		ExposedHeaders:  to.Ptr(strings.Join(exposeHeaders, ",")),
		MaxAgeInSeconds: to.Ptr(int32(maxAgeSeconds)),
	}

	svcClient := client.ServiceClient()

	// Get current properties to preserve other settings
	currentProps, err := svcClient.GetProperties(ctx, nil)
	if err != nil {
		cblog.Warnf("Failed to get current service properties, setting CORS only: %v", err)
	}

	setOpts := &service.SetPropertiesOptions{
		CORS: []*service.CORSRule{corsRule},
	}

	// Preserve other settings if available
	if currentProps.Logging != nil {
		setOpts.Logging = currentProps.Logging
	}
	if currentProps.HourMetrics != nil {
		setOpts.HourMetrics = currentProps.HourMetrics
	}
	if currentProps.MinuteMetrics != nil {
		setOpts.MinuteMetrics = currentProps.MinuteMetrics
	}

	_, err = svcClient.SetProperties(ctx, setOpts)
	if err != nil {
		return false, fmt.Errorf("failed to set CORS on Azure storage account: %w", err)
	}

	cblog.Infof("Successfully set CORS on Azure storage account '%s' (NOTE: affects all containers in this storage account)", connInfo.AccessKey)
	return true, nil
}

func getAzureBucketCORS(connInfo *S3ConnectionInfo) (*cors.Config, error) {
	cblog.Info("getAzureBucketCORS: Getting CORS from Azure storage account")

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	svcClient := client.ServiceClient()
	props, err := svcClient.GetProperties(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get service properties: %w", err)
	}

	if props.CORS == nil || len(props.CORS) == 0 {
		return nil, fmt.Errorf("CORS configuration not found for Azure storage account '%s'", connInfo.AccessKey)
	}

	// Convert Azure CORS rules to minio cors.Config
	corsConfig := &cors.Config{}
	for _, azureCORS := range props.CORS {
		rule := cors.Rule{}
		if azureCORS.AllowedOrigins != nil {
			rule.AllowedOrigin = strings.Split(*azureCORS.AllowedOrigins, ",")
		}
		if azureCORS.AllowedMethods != nil {
			rule.AllowedMethod = strings.Split(*azureCORS.AllowedMethods, ",")
		}
		if azureCORS.AllowedHeaders != nil {
			rule.AllowedHeader = strings.Split(*azureCORS.AllowedHeaders, ",")
		}
		if azureCORS.ExposedHeaders != nil {
			rule.ExposeHeader = strings.Split(*azureCORS.ExposedHeaders, ",")
		}
		if azureCORS.MaxAgeInSeconds != nil {
			rule.MaxAgeSeconds = int(*azureCORS.MaxAgeInSeconds)
		}
		corsConfig.CORSRules = append(corsConfig.CORSRules, rule)
	}

	return corsConfig, nil
}

func deleteAzureBucketCORS(connInfo *S3ConnectionInfo) (bool, error) {
	cblog.Info("deleteAzureBucketCORS: Deleting CORS from Azure storage account")

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	svcClient := client.ServiceClient()

	// Get current properties to preserve other settings
	currentProps, err := svcClient.GetProperties(ctx, nil)
	if err != nil {
		cblog.Warnf("Failed to get current service properties: %v", err)
	}

	setOpts := &service.SetPropertiesOptions{
		CORS: []*service.CORSRule{}, // Empty list to clear CORS
	}

	// Preserve other settings
	if currentProps.Logging != nil {
		setOpts.Logging = currentProps.Logging
	}
	if currentProps.HourMetrics != nil {
		setOpts.HourMetrics = currentProps.HourMetrics
	}
	if currentProps.MinuteMetrics != nil {
		setOpts.MinuteMetrics = currentProps.MinuteMetrics
	}

	_, err = svcClient.SetProperties(ctx, setOpts)
	if err != nil {
		return false, fmt.Errorf("failed to delete CORS from Azure storage account: %w", err)
	}

	cblog.Infof("Successfully deleted CORS from Azure storage account '%s'", connInfo.AccessKey)
	return true, nil
}

// ============================================================================
// Presigned URL (SAS) Operations
// ============================================================================

func getAzurePresignedURL(connInfo *S3ConnectionInfo, bucketName, objectName, method string, expiresSeconds int64, responseContentDisposition string) (string, error) {
	cblog.Infof("getAzurePresignedURL: Generating SAS URL for '%s/%s' (method: %s, expires: %ds)", bucketName, objectName, method, expiresSeconds)

	_, sharedKeyCred, err := newAzureBlobClient(connInfo)
	if err != nil {
		return "", err
	}

	expiry := time.Now().UTC().Add(time.Duration(expiresSeconds) * time.Second)

	var permissions sas.BlobPermissions
	switch method {
	case "GET":
		permissions = sas.BlobPermissions{Read: true}
	case "PUT":
		permissions = sas.BlobPermissions{Write: true, Create: true}
	default:
		return "", fmt.Errorf("unsupported method for SAS URL: %s", method)
	}

	// Build SAS query parameters
	sasValues := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     time.Now().UTC().Add(-5 * time.Minute),
		ExpiryTime:    expiry,
		ContainerName: bucketName,
		BlobName:      objectName,
		Permissions:   to.Ptr(permissions).String(),
	}

	if responseContentDisposition != "" {
		sasValues.ContentDisposition = responseContentDisposition
	}

	sasQueryParams, err := sasValues.SignWithSharedKey(sharedKeyCred)
	if err != nil {
		return "", fmt.Errorf("failed to generate SAS token: %w", err)
	}

	sasURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?%s",
		connInfo.AccessKey, bucketName, url.PathEscape(objectName), sasQueryParams.Encode())

	cblog.Infof("Generated SAS URL for '%s/%s'", bucketName, objectName)
	return sasURL, nil
}

// ============================================================================
// Force Operations
// ============================================================================

func forceEmptyAzureBucket(connectionName, bucketName string) (bool, error) {
	cblog.Infof("forceEmptyAzureBucket: Force emptying Azure container '%s'", bucketName)

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return false, err
	}

	var iidInfo S3BucketIIDInfo
	err = infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return false, fmt.Errorf("failed to get bucket info: %w", err)
	}

	client, _, err := newAzureBlobClient(connInfo)
	if err != nil {
		return false, err
	}

	ctx := context.Background()

	// First, try to delete all versioned blobs if versioning is enabled
	versionedOpts := &azblob.ListBlobsFlatOptions{
		Include: azblob.ListBlobsInclude{
			Versions: true,
		},
	}

	pager := client.NewListBlobsFlatPager(iidInfo.SystemId, versionedOpts)
	deletedCount := 0
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			cblog.Warnf("Failed to list versioned blobs, falling back to non-versioned: %v", err)
			break
		}
		for _, item := range page.Segment.BlobItems {
			if item.Name == nil {
				continue
			}
			if item.VersionID != nil && *item.VersionID != "" {
				_, err := deleteAzureObjectVersion(connInfo, iidInfo.SystemId, *item.Name, *item.VersionID)
				if err != nil {
					cblog.Warnf("Failed to delete blob version '%s' (version: %s): %v", *item.Name, *item.VersionID, err)
				} else {
					deletedCount++
				}
			} else {
				_, err := client.DeleteBlob(ctx, iidInfo.SystemId, *item.Name, nil)
				if err != nil {
					cblog.Warnf("Failed to delete blob '%s': %v", *item.Name, err)
				} else {
					deletedCount++
				}
			}
		}
	}

	// Also delete non-versioned blobs
	pager = client.NewListBlobsFlatPager(iidInfo.SystemId, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			break
		}
		for _, item := range page.Segment.BlobItems {
			if item.Name == nil {
				continue
			}
			_, err := client.DeleteBlob(ctx, iidInfo.SystemId, *item.Name, nil)
			if err != nil {
				cblog.Warnf("Failed to delete blob '%s': %v", *item.Name, err)
			} else {
				deletedCount++
			}
		}
	}

	cblog.Infof("Force emptied Azure container '%s': deleted %d items", bucketName, deletedCount)
	return true, nil
}

func forceEmptyAndDeleteAzureBucket(connectionName, bucketName string) (bool, error) {
	cblog.Infof("forceEmptyAndDeleteAzureBucket: Force emptying and deleting Azure container '%s'", bucketName)

	// First empty the bucket
	success, err := forceEmptyAzureBucket(connectionName, bucketName)
	if err != nil {
		return false, fmt.Errorf("failed to empty Azure container: %w", err)
	}
	if !success {
		return false, fmt.Errorf("failed to empty Azure container '%s'", bucketName)
	}

	connInfo, err := GetS3ConnectionInfo(connectionName)
	if err != nil {
		return false, err
	}

	var iidInfo S3BucketIIDInfo
	err = infostore.GetByConditions(&iidInfo, "connection_name", connectionName, "name_id", bucketName)
	if err != nil {
		return false, fmt.Errorf("failed to get bucket info: %w", err)
	}

	// Delete the container
	err = deleteAzureBucket(connInfo, iidInfo.SystemId)
	if err != nil {
		return false, fmt.Errorf("container emptied but deletion failed: %w", err)
	}

	// Remove from database
	db, err := infostore.Open()
	if err != nil {
		return false, fmt.Errorf("container deleted but failed to update database: %w", err)
	}
	defer infostore.Close(db)

	err = db.Delete(&iidInfo).Error
	if err != nil {
		return false, fmt.Errorf("container deleted but failed to update database: %w", err)
	}

	cblog.Infof("Successfully force-deleted Azure container '%s'", bucketName)
	return true, nil
}
