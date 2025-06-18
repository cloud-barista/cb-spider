// Cloud Control Manager's Rest Runtime of CB-Spider.
// REST API implementation for S3Handler (minio-go 기반, Swagger dummy struct 적용)
// by CB-Spider Team

package restruntime

import (
	"net/http"
	"strconv"
	"time"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	"github.com/labstack/echo/v4"
)

// ---------- Swagger 문서화용 dummy struct ----------

// S3BucketInfo Swagger doc용 (minio.BucketInfo와 호환)
type S3BucketInfo struct {
	Name         string    `json:"Name"`
	CreationDate time.Time `json:"CreationDate"`
}

// S3ObjectInfo Swagger doc용 (minio.ObjectInfo와 호환)
type S3ObjectInfo struct {
	Key          string    `json:"Key"`
	Size         int64     `json:"Size"`
	LastModified time.Time `json:"LastModified"`
	ETag         string    `json:"ETag"`
}

// S3UploadInfo Swagger doc용 (minio.UploadInfo와 호환)
type S3UploadInfo struct {
	Bucket string `json:"Bucket"`
	Key    string `json:"Key"`
	Size   int64  `json:"Size"`
	ETag   string `json:"ETag"`
}

type S3PresignedURL struct {
	PresignedURL string `json:"PresignedURL"`
}

// ---------- API 요청 구조체 ----------

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

type S3PresignedURLRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-s3-conn"`
	BucketName     string `json:"BucketName" validate:"required" example:"my-bucket-01"`
	ObjectName     string `json:"ObjectName" validate:"required" example:"my-object.txt"`
	Method         string `json:"Method" validate:"required" example:"GET"` // "GET" or "PUT"
	ExpiresSeconds int    `json:"ExpiresSeconds" validate:"required" example:"3600"`
}

// ---------- REST API 구현 ----------

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
// @Router /s3/bucket/{BucketName}/object [get]
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
// @Param ObjectName path string true "Object Name"
// @Success 200 {object} restruntime.S3ObjectInfo
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/bucket/{BucketName}/object/{ObjectName} [get]
func GetS3ObjectInfo(c echo.Context) error {
	conn := c.QueryParam("ConnectionName")
	bucket := c.Param("BucketName")
	obj := c.Param("ObjectName")
	o, err := cmrt.GetS3ObjectInfo(conn, bucket, obj)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, S3ObjectInfo{
		Key:          o.Key,
		Size:         o.Size,
		LastModified: o.LastModified,
		ETag:         o.ETag,
	})
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
		Bucket: info.Bucket,
		Key:    info.Key,
		Size:   info.Size,
		ETag:   info.ETag,
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
// @Param ObjectName path string true "Object Name"
// @Success 200 {file} file
// @Failure 500 {object} restruntime.SimpleMsg
// @Router /s3/bucket/{BucketName}/object/{ObjectName}/download [get]
func DownloadS3Object(c echo.Context) error {
	conn := c.QueryParam("ConnectionName")
	bucket := c.Param("BucketName")
	obj := c.Param("ObjectName")

	stream, err := cmrt.GetS3ObjectStream(conn, bucket, obj)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer stream.Close()

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+obj+"\"")
	c.Response().Header().Set("Content-Type", "application/octet-stream")
	return c.Stream(http.StatusOK, "application/octet-stream", stream)
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
