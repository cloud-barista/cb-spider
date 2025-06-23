// Cloud Control Manager's Rest Runtime of CB-Spider.
// REST API implementation for S3Manager (minio-go based).
// by CB-Spider Team

package restruntime

import (
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	"github.com/labstack/echo/v4"
)

// ---------- dummy struct for Swagger documentation ----------

// --------------- for Swagger doc (minio.BucketInfo)
type S3BucketInfo struct {
	Name         string    `json:"Name"`
	BucketRegion string    `json:"BucketRegion,omitempty"`
	CreationDate time.Time `json:"CreationDate"`
}

// --------------- for Swagger doc (minio.ObjectInfo)
type S3ObjectInfo struct {
	ETag              string              `json:"ETag"`
	Key               string              `json:"Key"`
	LastModified      time.Time           `json:"LastModified"`
	Size              int64               `json:"Size"`
	ContentType       string              `json:"ContentType"`
	Expires           time.Time           `json:"Expires"`
	Metadata          map[string][]string `json:"Metadata"`
	UserMetadata      map[string]string   `json:"UserMetadata,omitempty"`
	UserTags          map[string]string   `json:"UserTags,omitempty"`
	UserTagCount      int                 `json:"UserTagCount"`
	Owner             *S3Owner            `json:"Owner,omitempty"`
	Grant             []S3Grant           `json:"Grant,omitempty"`
	StorageClass      string              `json:"StorageClass"`
	IsLatest          bool                `json:"IsLatest"`
	IsDeleteMarker    bool                `json:"IsDeleteMarker"`
	VersionID         string              `json:"VersionID"`
	ReplicationStatus string              `json:"ReplicationStatus"`
	ReplicationReady  bool                `json:"ReplicationReady"`
	Expiration        time.Time           `json:"Expiration"`
	ExpirationRuleID  string              `json:"ExpirationRuleID"`
	NumVersions       int                 `json:"NumVersions"`
	Restore           *S3RestoreInfo      `json:"Restore,omitempty"`
	ChecksumCRC32     string              `json:"ChecksumCRC32"`
	ChecksumCRC32C    string              `json:"ChecksumCRC32C"`
	ChecksumSHA1      string              `json:"ChecksumSHA1"`
	ChecksumSHA256    string              `json:"ChecksumSHA256"`
	ChecksumCRC64NVME string              `json:"ChecksumCRC64NVME"`
	ChecksumMode      string              `json:"ChecksumMode"`
}

type S3Owner struct {
	DisplayName string `json:"DisplayName"`
	ID          string `json:"ID"`
}
type S3Grant struct {
	Grantee    interface{} `json:"Grantee"`
	Permission string      `json:"Permission"`
}
type S3RestoreInfo struct {
	OngoingRestore bool `json:"OngoingRestore"` // Whether the object is currently being restored
	// When the restored copy of the archived object will be removed
	ExpiryTime time.Time `json:"ExpiryTime,omitempty"` // Optional, only if applicable
}

// --------------- for Swagger doc (minio.UploadInfo)
type S3UploadInfo struct {
	Bucket            string    `json:"Bucket"`
	Key               string    `json:"Key"`
	ETag              string    `json:"ETag"`
	Size              int64     `json:"Size"`
	LastModified      time.Time `json:"LastModified"`
	Location          string    `json:"Location"`
	VersionID         string    `json:"VersionID"`
	Expiration        time.Time `json:"Expiration"`
	ExpirationRuleID  string    `json:"ExpirationRuleID"`
	ChecksumCRC32     string    `json:"ChecksumCRC32"`
	ChecksumCRC32C    string    `json:"ChecksumCRC32C"`
	ChecksumSHA1      string    `json:"ChecksumSHA1"`
	ChecksumSHA256    string    `json:"ChecksumSHA256"`
	ChecksumCRC64NVME string    `json:"ChecksumCRC64NVME"`
	ChecksumMode      string    `json:"ChecksumMode"`
}

// --------------- for Swagger doc (minio.BooleanInfo)
type S3PresignedURL struct {
	PresignedURL string `json:"PresignedURL"`
}

// ---------- API Request structs ----------

type S3BucketCreateRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-s3-conn"`
	Name           string `json:"Name" validate:"required" example:"my-bucket-01"`
}

type S3ObjectUploadRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-s3-conn"`
	BucketName     string `json:"BucketName" validate:"required" example:"my-bucket-01"`
	ObjectName     string `json:"ObjectName" validate:"required" example:"my-object.txt"`
	FilePath       string `json:"FilePath" validate:"required" example:"/tmp/data.txt"`
}

type S3ObjectDeleteRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-s3-conn"`
	BucketName     string `json:"BucketName" validate:"required" example:"my-bucket-01"`
	ObjectName     string `json:"ObjectName" validate:"required" example:"my-object.txt"`
}

// ---------- REST API Implementation ----------

// @Summary Create S3 Bucket
// @Description Create a new S3 bucket and register to CB-Spider infostore.
// @Tags [S3 Management]
// @Accept  json
// @Produce  json
// @Param S3BucketCreateRequest body restruntime.S3BucketCreateRequest true "Request body for creating an S3 bucket"
// @Success 200 {object} restruntime.S3BucketInfo
// @Failure 400 {object} restruntime.SimpleMsg
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/bucket [post]
func CreateS3Bucket(c echo.Context) error {
	var req S3BucketCreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	bucketInfo, err := cmrt.CreateS3Bucket(req.ConnectionName, req.Name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	// Swagger dummy struct로 변환해 반환 (실제 minio.BucketInfo에서 주요 필드만 사용)
	swaggerBucket := S3BucketInfo{
		Name:         bucketInfo.Name,
		CreationDate: bucketInfo.CreationDate,
	}
	return c.JSON(http.StatusOK, swaggerBucket)
}

// @Summary List S3 Buckets
// @Description List S3 buckets managed by CB-Spider (infostore).
// @Tags [S3 Management]
// @Produce  json
// @Param ConnectionName query string true "Connection Name"
// @Success 200 {array} restruntime.S3BucketInfo
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/bucket [get]
func ListS3Buckets(c echo.Context) error {
	conn := c.QueryParam("ConnectionName")
	result, err := cmrt.ListS3Buckets(conn)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	var swaggerList []S3BucketInfo
	for _, b := range result {
		swaggerList = append(swaggerList, S3BucketInfo{
			Name:         b.Name,
			CreationDate: b.CreationDate,
		})
	}
	return c.JSON(http.StatusOK, swaggerList)
}

// @Summary Get S3 Bucket
// @Description Get information of a specific S3 bucket
// @Tags [S3 Management]
// @Produce json
// @Param ConnectionName query string true "Connection Name"
// @Param Name path string true "Bucket Name"
// @Success 200 {object} restruntime.S3BucketInfo
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/bucket/{Name} [get]
func GetS3Bucket(c echo.Context) error {
	conn := c.QueryParam("ConnectionName")
	name := c.Param("Name")
	b, err := cmrt.GetS3Bucket(conn, name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	swaggerBucket := S3BucketInfo{
		Name:         b.Name,
		CreationDate: b.CreationDate,
		BucketRegion: b.BucketRegion,
	}
	return c.JSON(http.StatusOK, swaggerBucket)
}

// @Summary Delete S3 Bucket
// @Description Delete an S3 bucket (from S3 and infostore)
// @Tags [S3 Management]
// @Accept json
// @Produce json
// @Param ConnectionName query string true "Connection Name"
// @Param Name path string true "Bucket Name"
// @Success 200 {object} restruntime.BooleanInfo
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/bucket/{Name} [delete]
func DeleteS3Bucket(c echo.Context) error {
	conn := c.QueryParam("ConnectionName")
	name := c.Param("Name")
	result, err := cmrt.DeleteS3Bucket(conn, name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, BooleanInfo{Result: strconv.FormatBool(result)})
}

// @Summary List S3 Objects
// @Description List objects in an S3 bucket (managed bucket only)
// @Tags [S3 Management]
// @Produce json
// @Param ConnectionName query string true "Connection Name"
// @Param BucketName path string true "Bucket Name"
// @Param Prefix query string false "Prefix for filtering objects"
// @Success 200 {array} restruntime.S3ObjectInfo
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/bucket/{BucketName}/objectlist [get]
func ListS3Objects(c echo.Context) error {
	conn := c.QueryParam("ConnectionName")
	bucket := c.Param("BucketName")
	prefix := c.QueryParam("Prefix")
	result, err := cmrt.ListS3Objects(conn, bucket, prefix)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	var swaggerList []S3ObjectInfo
	for _, o := range result {
		swaggerList = append(swaggerList, S3ObjectInfo{
			Key:          o.Key,
			Size:         o.Size,
			LastModified: o.LastModified,
			ETag:         o.ETag,
		})
	}
	return c.JSON(http.StatusOK, swaggerList)
}

// @Summary Get S3 Object Metadata
// @Description Get metadata/stat of an object in S3
// @Tags [S3 Management]
// @Produce json
// @Param ConnectionName query string true "Connection Name"
// @Param BucketName path string true "Bucket Name"
// @Param ObjectName query string true "Object Name"
// @Success 200 {object} restruntime.S3ObjectInfo
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/bucket/{BucketName}/object [get]
func GetS3ObjectInfo(c echo.Context) error {
	conn := c.QueryParam("ConnectionName")
	bucket := c.Param("BucketName")
	obj := c.QueryParam("ObjectName")
	o, err := cmrt.GetS3ObjectInfo(conn, bucket, obj)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var owner *S3Owner
	if o.Owner.DisplayName != "" || o.Owner.ID != "" {
		owner = &S3Owner{
			DisplayName: o.Owner.DisplayName,
			ID:          o.Owner.ID,
		}
	}

	var grantList []S3Grant
	for _, g := range o.Grant {
		grantList = append(grantList, S3Grant{
			Grantee:    g.Grantee,
			Permission: g.Permission,
		})
	}

	var restore *S3RestoreInfo
	if o.Restore != nil {
		restore = &S3RestoreInfo{
			OngoingRestore: o.Restore.OngoingRestore,
			ExpiryTime:     o.Restore.ExpiryTime,
		}
	}

	// Convert UserMetadata and UserTags to maps
	um := map[string]string{}
	for k, v := range o.UserMetadata {
		um[k] = v
	}
	ut := map[string]string{}
	for k, v := range o.UserTags {
		ut[k] = v
	}

	s3Obj := S3ObjectInfo{
		ETag:              o.ETag,
		Key:               o.Key,
		LastModified:      o.LastModified,
		Size:              o.Size,
		ContentType:       o.ContentType,
		Expires:           o.Expires,
		Metadata:          map[string][]string(o.Metadata), // http.Header -> map[string][]string
		UserMetadata:      um,
		UserTags:          ut,
		UserTagCount:      o.UserTagCount,
		Owner:             owner,
		Grant:             grantList,
		StorageClass:      o.StorageClass,
		IsLatest:          o.IsLatest,
		IsDeleteMarker:    o.IsDeleteMarker,
		VersionID:         o.VersionID,
		ReplicationStatus: o.ReplicationStatus,
		ReplicationReady:  o.ReplicationReady,
		Expiration:        o.Expiration,
		ExpirationRuleID:  o.ExpirationRuleID,
		NumVersions:       o.NumVersions,
		Restore:           restore,
		ChecksumCRC32:     o.ChecksumCRC32,
		ChecksumCRC32C:    o.ChecksumCRC32C,
		ChecksumSHA1:      o.ChecksumSHA1,
		ChecksumSHA256:    o.ChecksumSHA256,
		ChecksumCRC64NVME: o.ChecksumCRC64NVME,
		ChecksumMode:      o.ChecksumMode,
	}

	return c.JSON(http.StatusOK, s3Obj)
}

// @Summary Upload S3 Object (from file path)
// @Description Upload a file to S3 bucket (managed bucket only)
// @Tags [S3 Management]
// @Accept json
// @Produce json
// @Param S3ObjectUploadRequest body restruntime.S3ObjectUploadRequest true "Upload info"
// @Success 200 {object} restruntime.S3UploadInfo
// @Failure 400 {object} restruntime.SimpleMsg
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/object [post]
func PutS3ObjectFromFile(c echo.Context) error {
	var req S3ObjectUploadRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	info, err := cmrt.PutS3ObjectFromFile(req.ConnectionName, req.BucketName, req.ObjectName, req.FilePath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, S3UploadInfo{
		Bucket:            info.Bucket,
		Key:               info.Key,
		ETag:              info.ETag,
		Size:              info.Size,
		LastModified:      info.LastModified,
		Location:          info.Location,
		VersionID:         info.VersionID,
		Expiration:        info.Expiration,
		ExpirationRuleID:  info.ExpirationRuleID,
		ChecksumCRC32:     info.ChecksumCRC32,
		ChecksumCRC32C:    info.ChecksumCRC32C,
		ChecksumSHA1:      info.ChecksumSHA1,
		ChecksumSHA256:    info.ChecksumSHA256,
		ChecksumCRC64NVME: info.ChecksumCRC64NVME,
		ChecksumMode:      info.ChecksumMode,
	})
}

// @Summary Delete S3 Object
// @Description Delete an object from S3 bucket (managed bucket only)
// @Tags [S3 Management]
// @Accept json
// @Produce json
// @Param S3ObjectDeleteRequest body restruntime.S3ObjectDeleteRequest true "Delete info"
// @Success 200 {object} restruntime.BooleanInfo
// @Failure 400 {object} restruntime.SimpleMsg
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/object [delete]
func DeleteS3Object(c echo.Context) error {
	var req S3ObjectDeleteRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	result, err := cmrt.DeleteS3Object(req.ConnectionName, req.BucketName, req.ObjectName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, BooleanInfo{Result: strconv.FormatBool(result)})
}

// @Summary Download S3 Object (Streaming)
// @Description Stream (download) an S3 object as a file (managed bucket only)
// @Tags [S3 Management]
// @Produce application/octet-stream
// @Param ConnectionName query string true "Connection Name"
// @Param BucketName path string true "Bucket Name"
// @Param ObjectName query string true "Object Name"
// @Success 200 {file} file
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/bucket/{BucketName}/object/download [get]
func DownloadS3Object(c echo.Context) error {
	conn := c.QueryParam("ConnectionName")
	bucket := c.Param("BucketName")
	obj := c.QueryParam("ObjectName")

	stream, err := cmrt.GetS3ObjectStream(conn, bucket, obj)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer stream.Close()

	filename := filepath.Base(obj) // obj = "/tmp/file.txt" -> "file.txt"
	c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	c.Response().Header().Set("Content-Type", "application/octet-stream")
	return c.Stream(http.StatusOK, "application/octet-stream", stream)
}

type S3PresignedURLRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-s3-conn"`
	BucketName     string `json:"BucketName" validate:"required" example:"my-bucket-01"`
	ObjectName     string `json:"ObjectName" validate:"required" example:"my-object.txt"`
	Method         string `json:"Method" validate:"required" example:"GET"` // "GET" or "PUT"
	ExpiresSeconds int    `json:"ExpiresSeconds" validate:"required" example:"3600"`
}

// @Summary Get S3 Presigned URL
// @Description Get a presigned URL for S3 object (GET/PUT)
// @Tags [S3 Management]
// @Accept json
// @Produce json
// @Param S3PresignedURLRequest body restruntime.S3PresignedURLRequest true "Presigned URL info"
// @Success 200 {object} restruntime.S3PresignedURL
// @Failure 400 {object} restruntime.SimpleMsg
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/object/presigned-url [post]
func GetS3PresignedURL(c echo.Context) error {
	var req S3PresignedURLRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	url, err := cmrt.GetS3PresignedURL(req.ConnectionName, req.BucketName, req.ObjectName, req.Method, int64(req.ExpiresSeconds))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, S3PresignedURL{PresignedURL: url})
}

type S3BucketACLRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required"`
	BucketName     string `json:"BucketName" validate:"required"`
	ACL            string `json:"ACL" validate:"required" example:"private|public-read"`
}

type S3ObjectACLRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required"`
	BucketName     string `json:"BucketName" validate:"required"`
	ObjectName     string `json:"ObjectName" validate:"required"`
	ACL            string `json:"ACL" validate:"required" example:"private|public-read"`
}

type S3BucketACLSetResponse struct {
	Policy string `json:"Policy"`
}

type S3BucketACLInfo struct {
	Policy string `json:"Policy"`
}

// @Summary Set S3 Bucket ACL
// @Description Set the ACL for a specific S3 bucket and return the applied policy
// @Tags [S3 Management]
// @Accept json
// @Produce json
// @Param S3BucketACLRequest body restruntime.S3BucketACLRequest true "ACL info"
// @Success 200 {object} restruntime.S3BucketACLSetResponse
// @Failure 400 {object} restruntime.SimpleMsg
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/bucket/acl [post]
func SetS3BucketACL(c echo.Context) error {
	var req S3BucketACLRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	policy, err := cmrt.SetS3BucketACL(req.ConnectionName, req.BucketName, req.ACL)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, S3BucketACLSetResponse{
		Policy: policy,
	})
}

// @Summary Get S3 Bucket ACL (Policy)
// @Description Get the current ACL(policy) for an S3 bucket
// @Tags [S3 Management]
// @Produce json
// @Param ConnectionName query string true "Connection Name"
// @Param BucketName query string true "Bucket Name"
// @Success 200 {object} restruntime.S3BucketACLInfo
// @Failure 400 {object} restruntime.SimpleMsg
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/bucket/acl [get]
func GetS3BucketACL(c echo.Context) error {
	conn := c.QueryParam("ConnectionName")
	bucket := c.QueryParam("BucketName")
	if conn == "" || bucket == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "ConnectionName and BucketName are required")
	}
	policy, err := cmrt.GetS3BucketACL(conn, bucket)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, S3BucketACLInfo{Policy: policy})
}

type S3BucketVersioningRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required"`
	BucketName     string `json:"BucketName" validate:"required"`
}

type S3ObjectVersionsRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required"`
	BucketName     string `json:"BucketName" validate:"required"`
	Prefix         string `json:"Prefix"`
}

// @Summary Enable S3 Bucket Versioning
// @Description Enable versioning for an S3 bucket
// @Tags [S3 Management]
// @Accept json
// @Produce json
// @Param S3BucketVersioningRequest body restruntime.S3BucketVersioningRequest true "Versioning info"
// @Success 200 {object} restruntime.BooleanInfo
// @Failure 400 {object} restruntime.SimpleMsg
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/bucket/versioning/enable [post]
func EnableVersioning(c echo.Context) error {
	var req S3BucketVersioningRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	result, err := cmrt.EnableVersioning(req.ConnectionName, req.BucketName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, BooleanInfo{Result: strconv.FormatBool(result)})
}

// @Summary Suspend S3 Bucket Versioning
// @Description Suspend versioning for an S3 bucket
// @Tags [S3 Management]
// @Accept json
// @Produce json
// @Param S3BucketVersioningRequest body restruntime.S3BucketVersioningRequest true "Versioning info"
// @Success 200 {object} restruntime.BooleanInfo
// @Failure 400 {object} restruntime.SimpleMsg
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/bucket/versioning/suspend [post]
func SuspendVersioning(c echo.Context) error {
	var req S3BucketVersioningRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	result, err := cmrt.SuspendVersioning(req.ConnectionName, req.BucketName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, BooleanInfo{Result: strconv.FormatBool(result)})
}

// @Summary List S3 Object Versions
// @Description List all versions of objects in a bucket (versioning enabled)
// @Tags [S3 Management]
// @Accept json
// @Produce json
// @Param S3ObjectVersionsRequest body restruntime.S3ObjectVersionsRequest true "Versions info"
// @Success 200 {array} restruntime.S3ObjectInfo
// @Failure 400 {object} restruntime.SimpleMsg
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/bucket/object/versions [post]
func ListS3ObjectVersions(c echo.Context) error {
	var req S3ObjectVersionsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	result, err := cmrt.ListS3ObjectVersions(req.ConnectionName, req.BucketName, req.Prefix)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	var swaggerList []S3ObjectInfo
	for _, o := range result {
		swaggerList = append(swaggerList, S3ObjectInfo{
			Key:            o.Key,
			Size:           o.Size,
			LastModified:   o.LastModified,
			ETag:           o.ETag,
			VersionID:      o.VersionID,
			IsLatest:       o.IsLatest,
			IsDeleteMarker: o.IsDeleteMarker,
		})
	}
	return c.JSON(http.StatusOK, swaggerList)
}
