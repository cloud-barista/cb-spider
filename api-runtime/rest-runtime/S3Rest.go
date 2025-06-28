// Cloud Control Manager's Rest Runtime of CB-Spider.
// REST API implementation for S3Manager (minio-go based).
// by CB-Spider Team

package restruntime

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
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

// ---------- Common functions ----------

func getConnectionName(c echo.Context) (string, bool) {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256") {
		accessKey, err := extractAccessKey(authHeader)
		if err == nil && accessKey != "" {
			cblog.Debugf("S3 API request detected with AccessKey: %s", accessKey)
			return accessKey, true
		}
	}

	conn := c.QueryParam("ConnectionName")
	if conn != "" {
		cblog.Debugf("CB-Spider API request with ConnectionName: %s", conn)
		return conn, false
	}

	cblog.Debug("No connection name found in request")
	return "", false
}

func extractAccessKey(authHeader string) (string, error) {
	const prefix = "AWS4-HMAC-SHA256 "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", errors.New("invalid Authorization header prefix")
	}

	parts := strings.Split(authHeader[len(prefix):], ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "Credential=") {
			credValue := strings.TrimPrefix(part, "Credential=")
			segments := strings.Split(credValue, "/")
			if len(segments) < 1 {
				return "", errors.New("invalid Credential format")
			}
			return segments[0], nil
		}
	}
	return "", errors.New("Credential field not found")
}

// S3 Error Response
type S3Error struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource"`
	RequestId string   `xml:"RequestId"`
}

func returnS3Error(c echo.Context, statusCode int, errorCode string, message string, resource string) error {
	requestId := fmt.Sprintf("%d", time.Now().Unix())
	c.Response().Header().Set("x-amz-request-id", requestId)
	c.Response().Header().Set("x-amz-id-2", requestId)

	s3Error := S3Error{
		Code:      errorCode,
		Message:   message,
		Resource:  resource,
		RequestId: requestId,
	}

	xmlData, err := xml.Marshal(s3Error)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}

	fullXML := append([]byte(xml.Header), xmlData...)
	return c.Blob(statusCode, "application/xml", fullXML)
}

func addS3Headers(c echo.Context) {
	requestId := fmt.Sprintf("%d", time.Now().Unix())
	c.Response().Header().Set("x-amz-request-id", requestId)
	c.Response().Header().Set("x-amz-id-2", requestId)
}

// ---------- XML Response Structures ----------

type ListAllMyBucketsResult struct {
	XMLName xml.Name `xml:"ListAllMyBucketsResult"`
	Xmlns   string   `xml:"xmlns,attr"`
	Owner   Owner    `xml:"Owner"`
	Buckets Buckets  `xml:"Buckets"`
}

type Owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type Buckets struct {
	Bucket []Bucket `xml:"Bucket"`
}

type Bucket struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

type ListBucketResult struct {
	XMLName     xml.Name      `xml:"ListBucketResult"`
	Xmlns       string        `xml:"xmlns,attr"`
	Name        string        `xml:"Name"`
	Prefix      string        `xml:"Prefix"`
	Marker      string        `xml:"Marker"`
	MaxKeys     int           `xml:"MaxKeys"`
	IsTruncated bool          `xml:"IsTruncated"`
	Contents    []S3ObjectXML `xml:"Contents"`
}

type S3ObjectXML struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
	Owner        *Owner `xml:"Owner,omitempty"`
}

type CreateBucketConfiguration struct {
	XMLName            xml.Name `xml:"CreateBucketConfiguration"`
	LocationConstraint string   `xml:"LocationConstraint"`
}

type VersioningConfiguration struct {
	XMLName xml.Name `xml:"VersioningConfiguration"`
	Status  string   `xml:"Status"`
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
	conn, isS3Api := getConnectionName(c)

	if isS3Api {
		bucketName := c.Param("Name")
		if bucketName == "" {
			return returnS3Error(c, http.StatusBadRequest, "InvalidBucketName", "Bucket name is required", "/")
		}

		var region string = "us-east-1"
		if c.Request().ContentLength > 0 {
			var config CreateBucketConfiguration
			if err := xml.NewDecoder(c.Request().Body).Decode(&config); err == nil {
				if config.LocationConstraint != "" {
					region = config.LocationConstraint
				}
			}
		}

		cblog.Infof("Creating S3 bucket: %s in region: %s", bucketName, region)

		_, err := cmrt.CreateS3Bucket(conn, bucketName)
		if err != nil {
			cblog.Errorf("Failed to create bucket %s: %v", bucketName, err)

			errorCode := "InternalError"
			statusCode := http.StatusInternalServerError

			if strings.Contains(err.Error(), "already exists") {
				errorCode = "BucketAlreadyExists"
				statusCode = http.StatusConflict
			} else if strings.Contains(err.Error(), "already owned") {
				errorCode = "BucketAlreadyOwnedByYou"
				statusCode = http.StatusConflict
			}

			return returnS3Error(c, statusCode, errorCode, err.Error(), "/"+bucketName)
		}

		addS3Headers(c)
		c.Response().Header().Set("Location", "/"+bucketName)
		return c.NoContent(http.StatusOK)
	}

	var req S3BucketCreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	bucketInfo, err := cmrt.CreateS3Bucket(req.ConnectionName, req.Name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
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
	conn, isS3Api := getConnectionName(c)

	cblog.Infof("ListS3Buckets called - isS3Api: %v, conn: %s", isS3Api, conn)

	result, err := cmrt.ListS3Buckets(conn)
	if err != nil {
		if isS3Api {
			return returnS3Error(c, http.StatusInternalServerError, "InternalError", err.Error(), "/")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if isS3Api {
		var bucketElems []Bucket
		for _, b := range result {
			bucketElems = append(bucketElems, Bucket{
				Name:         b.Name,
				CreationDate: b.CreationDate.UTC().Format(time.RFC3339),
			})
		}

		resp := ListAllMyBucketsResult{
			Xmlns: "http://s3.amazonaws.com/doc/2006-03-01/",
			Owner: Owner{
				ID:          conn,
				DisplayName: conn,
			},
			Buckets: Buckets{Bucket: bucketElems},
		}

		// XML을 수동으로 생성하여 정확한 형식 보장
		var buf bytes.Buffer
		buf.WriteString(xml.Header)
		enc := xml.NewEncoder(&buf)
		enc.Indent("", "  ")

		if err := enc.Encode(resp); err != nil {
			return returnS3Error(c, http.StatusInternalServerError, "InternalError", err.Error(), "/")
		}

		xmlContent := buf.Bytes()
		cblog.Debugf("Generated XML response: %s", string(xmlContent))

		addS3Headers(c)
		c.Response().Header().Set("Content-Type", "application/xml")
		c.Response().Header().Set("Content-Length", strconv.Itoa(len(xmlContent)))

		return c.Blob(http.StatusOK, "application/xml", xmlContent)
	}

	var swaggerBuckets []S3BucketInfo
	for _, b := range result {
		swaggerBuckets = append(swaggerBuckets, S3BucketInfo{
			Name:         b.Name,
			CreationDate: b.CreationDate,
		})
	}
	return c.JSON(http.StatusOK, swaggerBuckets)
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
	conn, isS3Api := getConnectionName(c)
	name := c.Param("Name")
	name = strings.TrimSuffix(name, "/")

	if isS3Api && c.Request().Method == "GET" {
		if c.QueryParam("location") != "" {
			return GetBucketLocation(c)
		}

		if c.QueryParam("acl") == "" &&
			c.QueryParam("versioning") == "" &&
			c.QueryParam("policy") == "" &&
			c.QueryParam("lifecycle") == "" &&
			c.QueryParam("cors") == "" {
			cblog.Infof("Redirecting to ListS3Objects for bucket: %s", name)
			c.SetParamNames("Name")
			c.SetParamValues(name)
			return ListS3Objects(c)
		}
	}

	if isS3Api && c.Request().Method == "HEAD" {
		_, err := cmrt.GetS3Bucket(conn, name)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return c.NoContent(http.StatusNotFound)
			}
			return c.NoContent(http.StatusForbidden)
		}
		addS3Headers(c)
		return c.NoContent(http.StatusOK)
	}

	b, err := cmrt.GetS3Bucket(conn, name)
	if err != nil {
		if isS3Api {
			errorCode := "NoSuchBucket"
			if strings.Contains(err.Error(), "not found") {
				return returnS3Error(c, http.StatusNotFound, errorCode, err.Error(), "/"+name)
			}
			return returnS3Error(c, http.StatusInternalServerError, "InternalError", err.Error(), "/"+name)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	swaggerBucket := S3BucketInfo{
		Name:         b.Name,
		CreationDate: b.CreationDate,
		BucketRegion: b.BucketRegion,
	}
	return c.JSON(http.StatusOK, swaggerBucket)
}

// GetBucketLocation returns the location (region) of a bucket
func GetBucketLocation(c echo.Context) error {
	conn, _ := getConnectionName(c)
	bucketName := c.Param("Name")
	bucketName = strings.TrimSuffix(bucketName, "/")

	bucketInfo, err := cmrt.GetS3Bucket(conn, bucketName)
	region := ""
	if err == nil && bucketInfo.BucketRegion != "" {
		region = bucketInfo.BucketRegion
	}

	type LocationConstraint struct {
		XMLName            xml.Name `xml:"LocationConstraint"`
		Xmlns              string   `xml:"xmlns,attr"`
		LocationConstraint string   `xml:",chardata"`
	}

	resp := LocationConstraint{
		Xmlns:              "http://s3.amazonaws.com/doc/2006-03-01/",
		LocationConstraint: region,
	}

	addS3Headers(c)

	xmlData, err := xml.Marshal(resp)
	if err != nil {
		return returnS3Error(c, http.StatusInternalServerError, "InternalError", err.Error(), "/"+bucketName)
	}

	fullXML := append([]byte(xml.Header), xmlData...)
	return c.Blob(http.StatusOK, "application/xml", fullXML)
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
	conn, isS3Api := getConnectionName(c)
	name := c.Param("Name")

	result, err := cmrt.DeleteS3Bucket(conn, name)
	if err != nil {
		if isS3Api {
			errorCode := "InternalError"
			if strings.Contains(err.Error(), "not empty") {
				errorCode = "BucketNotEmpty"
			} else if strings.Contains(err.Error(), "not found") {
				errorCode = "NoSuchBucket"
			}
			return returnS3Error(c, http.StatusConflict, errorCode, err.Error(), "/"+name)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if isS3Api {
		addS3Headers(c)
		return c.NoContent(http.StatusNoContent)
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
	cblog.Infof("ListS3Objects called - Path: %s, Method: %s", c.Path(), c.Request().Method)

	conn, isS3Api := getConnectionName(c)
	var bucket string
	var prefix string
	var delimiter string

	if isS3Api {
		bucket = c.Param("Name")
		if bucket == "" {
			bucket = c.Param("BucketName")
		}
		bucket = strings.TrimSuffix(bucket, "/")

		prefix = c.QueryParam("prefix")
		if prefix == "" {
			prefix = c.QueryParam("Prefix")
		}

		delimiter = c.QueryParam("delimiter")
		if delimiter == "" {
			delimiter = c.QueryParam("Delimiter")
		}

		cblog.Infof("S3 API - Bucket: %s, Prefix: %s, Delimiter: %s", bucket, prefix, delimiter)
	} else {
		bucket = c.Param("BucketName")
		prefix = c.QueryParam("Prefix")
		cblog.Infof("CB-Spider API - Bucket: %s, Prefix: %s", bucket, prefix)
	}

	if bucket == "" {
		if isS3Api {
			return returnS3Error(c, http.StatusBadRequest, "InvalidBucketName", "Bucket name is required", "/")
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Bucket name is required")
	}

	result, err := cmrt.ListS3Objects(conn, bucket, prefix)
	if err != nil {
		cblog.Errorf("Failed to list objects in bucket %s: %v", bucket, err)
		if isS3Api {
			errorCode := "NoSuchBucket"
			statusCode := http.StatusNotFound
			if !strings.Contains(err.Error(), "not found") {
				errorCode = "InternalError"
				statusCode = http.StatusInternalServerError
			}
			return returnS3Error(c, statusCode, errorCode, err.Error(), "/"+bucket)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	cblog.Infof("Found %d objects in bucket %s", len(result), bucket)

	if isS3Api {
		if delimiter == "/" {
			type CommonPrefix struct {
				Prefix string `xml:"Prefix"`
			}

			type ListBucketResultWithPrefix struct {
				XMLName        xml.Name       `xml:"ListBucketResult"`
				Xmlns          string         `xml:"xmlns,attr"`
				Name           string         `xml:"Name"`
				Prefix         string         `xml:"Prefix"`
				Delimiter      string         `xml:"Delimiter"`
				Marker         string         `xml:"Marker"`
				MaxKeys        int            `xml:"MaxKeys"`
				IsTruncated    bool           `xml:"IsTruncated"`
				Contents       []S3ObjectXML  `xml:"Contents"`
				CommonPrefixes []CommonPrefix `xml:"CommonPrefixes"`
			}

			var contents []S3ObjectXML
			commonPrefixMap := make(map[string]bool)

			for _, obj := range result {
				if prefix != "" && !strings.HasPrefix(obj.Key, prefix) {
					continue
				}

				relativeKey := obj.Key
				if prefix != "" {
					relativeKey = strings.TrimPrefix(obj.Key, prefix)
				}

				if idx := strings.Index(relativeKey, delimiter); idx > 0 {
					subPrefix := prefix + relativeKey[:idx+1]
					commonPrefixMap[subPrefix] = true
				} else if relativeKey != "" {
					if !(strings.HasSuffix(obj.Key, "/") && obj.Key == prefix) {
						contents = append(contents, S3ObjectXML{
							Key:          obj.Key,
							LastModified: obj.LastModified.UTC().Format(time.RFC3339),
							ETag:         strings.Trim(obj.ETag, "\""),
							Size:         obj.Size,
							StorageClass: "STANDARD",
						})
					}
				}
			}

			var commonPrefixes []CommonPrefix
			for prefix := range commonPrefixMap {
				commonPrefixes = append(commonPrefixes, CommonPrefix{Prefix: prefix})
			}

			resp := ListBucketResultWithPrefix{
				Xmlns:          "http://s3.amazonaws.com/doc/2006-03-01/",
				Name:           bucket,
				Prefix:         prefix,
				Delimiter:      delimiter,
				Marker:         "",
				MaxKeys:        1000,
				IsTruncated:    false,
				Contents:       contents,
				CommonPrefixes: commonPrefixes,
			}

			addS3Headers(c)

			xmlData, err := xml.Marshal(resp)
			if err != nil {
				return returnS3Error(c, http.StatusInternalServerError, "InternalError", err.Error(), "/"+bucket)
			}

			fullXML := append([]byte(xml.Header), xmlData...)
			cblog.Debugf("Returning XML with %d objects and %d common prefixes", len(contents), len(commonPrefixes))
			return c.Blob(http.StatusOK, "application/xml", fullXML)
		}

		// delimiter가 없으면 기존 방식대로 처리
		var contents []S3ObjectXML
		for _, o := range result {
			contents = append(contents, S3ObjectXML{
				Key:          o.Key,
				LastModified: o.LastModified.UTC().Format(time.RFC3339),
				ETag:         strings.Trim(o.ETag, "\""),
				Size:         o.Size,
				StorageClass: "STANDARD",
			})
		}

		resp := ListBucketResult{
			Xmlns:       "http://s3.amazonaws.com/doc/2006-03-01/",
			Name:        bucket,
			Prefix:      prefix,
			Marker:      "",
			MaxKeys:     1000,
			IsTruncated: false,
			Contents:    contents,
		}

		addS3Headers(c)
		cblog.Debugf("Returning XML response with %d objects", len(contents))

		xmlData, err := xml.Marshal(resp)
		if err != nil {
			return returnS3Error(c, http.StatusInternalServerError, "InternalError", err.Error(), "/"+bucket)
		}

		fullXML := append([]byte(xml.Header), xmlData...)
		return c.Blob(http.StatusOK, "application/xml", fullXML)
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
	conn, isS3Api := getConnectionName(c)
	bucket := c.Param("BucketName")
	var obj string

	if isS3Api {
		obj = c.Param("ObjectKey+")
	} else {
		obj = c.QueryParam("ObjectName")
	}

	o, err := cmrt.GetS3ObjectInfo(conn, bucket, obj)
	if err != nil {
		if isS3Api {
			errorCode := "NoSuchKey"
			statusCode := http.StatusNotFound
			if strings.Contains(err.Error(), "bucket") {
				errorCode = "NoSuchBucket"
			}
			return returnS3Error(c, statusCode, errorCode, err.Error(), "/"+bucket+"/"+obj)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if isS3Api && c.Request().Method == "HEAD" {
		addS3Headers(c)
		c.Response().Header().Set("Content-Type", o.ContentType)
		c.Response().Header().Set("Content-Length", strconv.FormatInt(o.Size, 10))
		c.Response().Header().Set("Last-Modified", o.LastModified.UTC().Format(http.TimeFormat))
		c.Response().Header().Set("ETag", o.ETag)
		if o.VersionID != "" {
			c.Response().Header().Set("x-amz-version-id", o.VersionID)
		}
		return c.NoContent(http.StatusOK)
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
		Metadata:          map[string][]string(o.Metadata),
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
	if c.QueryParam("uploadId") != "" && c.QueryParam("partNumber") != "" {
		return UploadPart(c)
	}

	conn, isS3Api := getConnectionName(c)

	if isS3Api {
		bucket := c.Param("BucketName")
		objKey := c.Param("ObjectKey+")

		// S3 Browser는 폴더 생성 시 Content-Length: 0으로 요청을 보냄
		// 폴더인지 확인 (Content-Length가 0이고 슬래시로 끝나지 않는 경우)
		if c.Request().ContentLength == 0 && !strings.HasSuffix(objKey, "/") {
			// User-Agent로 S3 Browser 확인
			userAgent := c.Request().Header.Get("User-Agent")
			if strings.Contains(userAgent, "S3 Browser") {
				// S3 Browser의 폴더 생성 요청인 경우 키 이름에 슬래시 추가
				objKey = objKey + "/"
				cblog.Infof("S3 Browser folder creation detected, adding trailing slash: %s", objKey)
			}
		}

		body := c.Request().Body
		defer body.Close()

		info, err := cmrt.PutS3ObjectFromReader(conn, bucket, objKey, body, c.Request().ContentLength)
		if err != nil {
			errorCode := "InternalError"
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "bucket") {
				errorCode = "NoSuchBucket"
				statusCode = http.StatusNotFound
			}
			return returnS3Error(c, statusCode, errorCode, err.Error(), "/"+bucket+"/"+objKey)
		}

		addS3Headers(c)
		c.Response().Header().Set("ETag", info.ETag)
		if info.VersionID != "" {
			c.Response().Header().Set("x-amz-version-id", info.VersionID)
		}
		return c.NoContent(http.StatusOK)
	}

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
	conn, isS3Api := getConnectionName(c)

	if isS3Api {
		bucket := c.Param("BucketName")
		objKey := c.Param("ObjectKey+")

		// 로깅 추가
		cblog.Infof("DeleteS3Object called - bucket: %s, objKey: %s", bucket, objKey)

		// S3 Browser의 폴더 삭제 요청 처리
		userAgent := c.Request().Header.Get("User-Agent")
		if strings.Contains(userAgent, "S3 Browser") && !strings.HasSuffix(objKey, "/") {
			// 먼저 슬래시가 붙은 버전이 있는지 확인
			objKeyWithSlash := objKey + "/"
			_, err := cmrt.GetS3ObjectInfo(conn, bucket, objKeyWithSlash)
			if err == nil {
				// 폴더가 존재하면 슬래시를 추가
				objKey = objKeyWithSlash
				cblog.Infof("S3 Browser folder deletion detected, adding trailing slash: %s", objKey)
			} else {
				cblog.Debugf("No folder found with slash, proceeding with original key: %s", objKey)
			}
		}

		_, err := cmrt.DeleteS3Object(conn, bucket, objKey)
		if err != nil {
			errorCode := "NoSuchKey"
			statusCode := http.StatusNotFound
			if strings.Contains(err.Error(), "bucket") {
				errorCode = "NoSuchBucket"
			}
			return returnS3Error(c, statusCode, errorCode, err.Error(), "/"+bucket+"/"+objKey)
		}

		addS3Headers(c)
		return c.NoContent(http.StatusNoContent)
	}

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
	conn, isS3Api := getConnectionName(c)
	bucket := c.Param("BucketName")
	var obj string

	if isS3Api {
		objKey := c.Param("ObjectKey+")
		obj = objKey
	} else {
		obj = c.QueryParam("ObjectName")
	}

	stream, err := cmrt.GetS3ObjectStream(conn, bucket, obj)
	if err != nil {
		if isS3Api {
			errorCode := "NoSuchKey"
			statusCode := http.StatusNotFound
			if strings.Contains(err.Error(), "bucket") {
				errorCode = "NoSuchBucket"
			}
			return returnS3Error(c, statusCode, errorCode, err.Error(), "/"+bucket+"/"+obj)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer stream.Close()

	if isS3Api {
		addS3Headers(c)
	} else {
		filename := filepath.Base(obj)
		c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	}

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
	conn, isS3Api := getConnectionName(c)

	if isS3Api {
		bucket := c.Param("Name")
		if c.QueryParam("acl") == "" {
			return returnS3Error(c, http.StatusBadRequest, "MissingParameter", "ACL parameter required", "/"+bucket)
		}

		aclHeader := c.Request().Header.Get("x-amz-acl")
		if aclHeader == "" {
			aclHeader = "private"
		}

		_, err := cmrt.SetS3BucketACL(conn, bucket, aclHeader)
		if err != nil {
			errorCode := "InternalError"
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				errorCode = "NoSuchBucket"
				statusCode = http.StatusNotFound
			}
			return returnS3Error(c, statusCode, errorCode, err.Error(), "/"+bucket)
		}

		addS3Headers(c)
		return c.NoContent(http.StatusOK)
	}

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
	conn, isS3Api := getConnectionName(c)

	if isS3Api {
		bucket := c.Param("Name")
		if c.QueryParam("versioning") == "" {
			return returnS3Error(c, http.StatusBadRequest, "MissingParameter", "versioning parameter required", "/"+bucket)
		}

		var config VersioningConfiguration
		if err := xml.NewDecoder(c.Request().Body).Decode(&config); err != nil {
			return returnS3Error(c, http.StatusBadRequest, "MalformedXML", err.Error(), "/"+bucket)
		}

		var err error
		if config.Status == "Enabled" {
			_, err = cmrt.EnableVersioning(conn, bucket)
		} else if config.Status == "Suspended" {
			_, err = cmrt.SuspendVersioning(conn, bucket)
		} else {
			return returnS3Error(c, http.StatusBadRequest, "InvalidArgument", "Invalid versioning status", "/"+bucket)
		}

		if err != nil {
			errorCode := "InternalError"
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				errorCode = "NoSuchBucket"
				statusCode = http.StatusNotFound
			}
			return returnS3Error(c, statusCode, errorCode, err.Error(), "/"+bucket)
		}

		addS3Headers(c)
		return c.NoContent(http.StatusOK)
	}

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
	conn, isS3Api := getConnectionName(c)

	var bucket, prefix string
	if isS3Api {
		bucket = c.Param("BucketName")
		if c.QueryParam("versions") == "" {
			return returnS3Error(c, http.StatusBadRequest, "MissingParameter", "versions parameter required", "/"+bucket)
		}
		prefix = c.QueryParam("prefix")
	} else {
		var req S3ObjectVersionsRequest
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		bucket = req.BucketName
		prefix = req.Prefix
	}

	result, err := cmrt.ListS3ObjectVersions(conn, bucket, prefix)
	if err != nil {
		if isS3Api {
			errorCode := "NoSuchBucket"
			statusCode := http.StatusNotFound
			if !strings.Contains(err.Error(), "not found") {
				errorCode = "InternalError"
				statusCode = http.StatusInternalServerError
			}
			return returnS3Error(c, statusCode, errorCode, err.Error(), "/"+bucket)
		}
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

// HandleS3BucketPost handles various POST operations on S3 bucket
func HandleS3BucketPost(c echo.Context) error {
	// 쿼리 파라미터 확인
	if c.QueryParam("uploads") != "" {
		return InitiateMultipartUpload(c)
	}
	if c.QueryParam("uploadId") != "" {
		return CompleteMultipartUpload(c)
	}
	if c.QueryParam("delete") != "" {
		return DeleteMultipleObjects(c)
	}

	// Content-Type 확인하여 bulk delete 요청 감지
	contentType := c.Request().Header.Get("Content-Type")
	if strings.Contains(contentType, "application/xml") || c.Request().Header.Get("Content-MD5") != "" {
		// S3 Browser의 bulk delete 요청
		cblog.Info("Bulk delete request detected")
		return DeleteMultipleObjects(c)
	}

	// multipart/form-data인 경우 PostObject 처리
	if strings.Contains(contentType, "multipart/form-data") {
		return PostObject(c)
	}

	// 기본적으로 DeleteMultipleObjects 시도
	return DeleteMultipleObjects(c)
}

// InitiateMultipartUpload initiates a multipart upload
func InitiateMultipartUpload(c echo.Context) error {
	conn, isS3Api := getConnectionName(c)
	bucket := c.Param("Name")
	if bucket == "" {
		bucket = c.Param("BucketName")
	}
	key := c.QueryParam("key")

	if key == "" {
		if isS3Api {
			return returnS3Error(c, http.StatusBadRequest, "MissingParameter", "key parameter is required", "/"+bucket)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "key parameter is required")
	}

	uploadID, err := cmrt.InitiateMultipartUpload(conn, bucket, key)
	if err != nil {
		if isS3Api {
			errorCode := "InternalError"
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				errorCode = "NoSuchBucket"
				statusCode = http.StatusNotFound
			}
			return returnS3Error(c, statusCode, errorCode, err.Error(), "/"+bucket+"/"+key)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if isS3Api {
		type InitiateMultipartUploadResult struct {
			XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
			Xmlns    string   `xml:"xmlns,attr"`
			Bucket   string   `xml:"Bucket"`
			Key      string   `xml:"Key"`
			UploadId string   `xml:"UploadId"`
		}

		resp := InitiateMultipartUploadResult{
			Xmlns:    "http://s3.amazonaws.com/doc/2006-03-01/",
			Bucket:   bucket,
			Key:      key,
			UploadId: uploadID,
		}

		addS3Headers(c)

		xmlData, err := xml.Marshal(resp)
		if err != nil {
			return returnS3Error(c, http.StatusInternalServerError, "InternalError", err.Error(), "/"+bucket+"/"+key)
		}

		fullXML := append([]byte(xml.Header), xmlData...)
		return c.Blob(http.StatusOK, "application/xml", fullXML)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"UploadId": uploadID,
		"Bucket":   bucket,
		"Key":      key,
	})
}

// CompleteMultipartUpload completes a multipart upload
func CompleteMultipartUpload(c echo.Context) error {
	conn, isS3Api := getConnectionName(c)
	bucket := c.Param("Name")
	if bucket == "" {
		bucket = c.Param("BucketName")
	}
	key := c.Param("ObjectKey+")
	uploadID := c.QueryParam("uploadId")

	if uploadID == "" {
		if isS3Api {
			return returnS3Error(c, http.StatusBadRequest, "MissingParameter", "uploadId parameter is required", "/"+bucket+"/"+key)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "uploadId parameter is required")
	}

	type Part struct {
		PartNumber int    `xml:"PartNumber"`
		ETag       string `xml:"ETag"`
	}

	type CompleteMultipartUploadRequest struct {
		XMLName xml.Name `xml:"CompleteMultipartUpload"`
		Parts   []Part   `xml:"Part"`
	}

	var req CompleteMultipartUploadRequest
	if err := xml.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		if isS3Api {
			return returnS3Error(c, http.StatusBadRequest, "MalformedXML", err.Error(), "/"+bucket+"/"+key)
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var parts []cmrt.CompletePart
	for _, p := range req.Parts {
		parts = append(parts, cmrt.CompletePart{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
		})
	}

	location, etag, err := cmrt.CompleteMultipartUpload(conn, bucket, key, uploadID, parts)
	if err != nil {
		if isS3Api {
			errorCode := "InternalError"
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				errorCode = "NoSuchUpload"
				statusCode = http.StatusNotFound
			}
			return returnS3Error(c, statusCode, errorCode, err.Error(), "/"+bucket+"/"+key)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if isS3Api {
		type CompleteMultipartUploadResult struct {
			XMLName  xml.Name `xml:"CompleteMultipartUploadResult"`
			Xmlns    string   `xml:"xmlns,attr"`
			Location string   `xml:"Location"`
			Bucket   string   `xml:"Bucket"`
			Key      string   `xml:"Key"`
			ETag     string   `xml:"ETag"`
		}

		resp := CompleteMultipartUploadResult{
			Xmlns:    "http://s3.amazonaws.com/doc/2006-03-01/",
			Location: location,
			Bucket:   bucket,
			Key:      key,
			ETag:     etag,
		}

		addS3Headers(c)

		xmlData, err := xml.Marshal(resp)
		if err != nil {
			return returnS3Error(c, http.StatusInternalServerError, "InternalError", err.Error(), "/"+bucket+"/"+key)
		}

		fullXML := append([]byte(xml.Header), xmlData...)
		return c.Blob(http.StatusOK, "application/xml", fullXML)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"Location": location,
		"Bucket":   bucket,
		"Key":      key,
		"ETag":     etag,
	})
}

// DeleteMultipleObjects deletes multiple objects from S3
func DeleteMultipleObjects(c echo.Context) error {
	conn, isS3Api := getConnectionName(c)
	bucket := c.Param("Name")
	if bucket == "" {
		bucket = c.Param("BucketName")
	}

	cblog.Infof("DeleteMultipleObjects called - bucket: %s", bucket)

	type Object struct {
		Key       string `xml:"Key"`
		VersionId string `xml:"VersionId,omitempty"`
	}

	type Delete struct {
		XMLName xml.Name `xml:"Delete"`
		Objects []Object `xml:"Object"`
		Quiet   bool     `xml:"Quiet"`
	}

	var req Delete
	if err := xml.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		cblog.Errorf("Failed to decode delete request: %v", err)
		if isS3Api {
			return returnS3Error(c, http.StatusBadRequest, "MalformedXML", err.Error(), "/"+bucket)
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	cblog.Infof("Deleting %d objects from bucket %s", len(req.Objects), bucket)

	var keys []string
	for _, obj := range req.Objects {
		keys = append(keys, obj.Key)
		cblog.Debugf("Object to delete: %s", obj.Key)
	}

	results, err := cmrt.DeleteMultipleObjects(conn, bucket, keys)
	if err != nil {
		cblog.Errorf("Failed to delete multiple objects: %v", err)
		if isS3Api {
			return returnS3Error(c, http.StatusInternalServerError, "InternalError", err.Error(), "/"+bucket)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if isS3Api {
		type Deleted struct {
			Key string `xml:"Key"`
		}

		type Error struct {
			Key     string `xml:"Key"`
			Code    string `xml:"Code"`
			Message string `xml:"Message"`
		}

		type DeleteResult struct {
			XMLName xml.Name  `xml:"DeleteResult"`
			Xmlns   string    `xml:"xmlns,attr"`
			Deleted []Deleted `xml:"Deleted"`
			Error   []Error   `xml:"Error"`
		}

		resp := DeleteResult{
			Xmlns: "http://s3.amazonaws.com/doc/2006-03-01/",
		}

		for _, result := range results {
			if result.Success {
				resp.Deleted = append(resp.Deleted, Deleted{Key: result.Key})
			} else {
				resp.Error = append(resp.Error, Error{
					Key:     result.Key,
					Code:    "InternalError",
					Message: result.Error,
				})
			}
		}

		addS3Headers(c)

		xmlData, err := xml.Marshal(resp)
		if err != nil {
			return returnS3Error(c, http.StatusInternalServerError, "InternalError", err.Error(), "/"+bucket)
		}

		fullXML := append([]byte(xml.Header), xmlData...)
		cblog.Debugf("Returning delete result with %d deleted and %d errors", len(resp.Deleted), len(resp.Error))
		return c.Blob(http.StatusOK, "application/xml", fullXML)
	}

	return c.JSON(http.StatusOK, results)
}

// PostObject handles browser-based file upload using HTML form
func PostObject(c echo.Context) error {
	conn, isS3Api := getConnectionName(c)
	bucket := c.Param("Name")
	if bucket == "" {
		bucket = c.Param("BucketName")
	}

	form, err := c.MultipartForm()
	if err != nil {
		if isS3Api {
			return returnS3Error(c, http.StatusBadRequest, "MalformedPOSTRequest", err.Error(), "/"+bucket)
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	key := form.Value["key"][0]
	if key == "" {
		if isS3Api {
			return returnS3Error(c, http.StatusBadRequest, "MissingFields", "key is required", "/"+bucket)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "key is required")
	}

	files := form.File["file"]
	if len(files) == 0 {
		if isS3Api {
			return returnS3Error(c, http.StatusBadRequest, "MissingFields", "file is required", "/"+bucket)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "file is required")
	}

	file, err := files[0].Open()
	if err != nil {
		if isS3Api {
			return returnS3Error(c, http.StatusInternalServerError, "InternalError", err.Error(), "/"+bucket+"/"+key)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer file.Close()

	info, err := cmrt.PutS3ObjectFromReader(conn, bucket, key, file, files[0].Size)
	if err != nil {
		if isS3Api {
			errorCode := "InternalError"
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "bucket") {
				errorCode = "NoSuchBucket"
				statusCode = http.StatusNotFound
			}
			return returnS3Error(c, statusCode, errorCode, err.Error(), "/"+bucket+"/"+key)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	successRedirect := form.Value["success_action_redirect"]
	if len(successRedirect) > 0 && successRedirect[0] != "" {
		return c.Redirect(http.StatusSeeOther, successRedirect[0])
	}

	if isS3Api {
		addS3Headers(c)
		return c.NoContent(http.StatusNoContent)
	}

	return c.JSON(http.StatusOK, S3UploadInfo{
		Bucket:       info.Bucket,
		Key:          info.Key,
		ETag:         info.ETag,
		Size:         info.Size,
		LastModified: info.LastModified,
		Location:     info.Location,
		VersionID:    info.VersionID,
	})
}

// UploadPart uploads a part in a multipart upload
func UploadPart(c echo.Context) error {
	conn, isS3Api := getConnectionName(c)
	bucket := c.Param("BucketName")
	key := c.Param("ObjectKey+")
	uploadID := c.QueryParam("uploadId")
	partNumberStr := c.QueryParam("partNumber")

	if uploadID == "" || partNumberStr == "" {
		if isS3Api {
			return returnS3Error(c, http.StatusBadRequest, "MissingParameter", "uploadId and partNumber are required", "/"+bucket+"/"+key)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "uploadId and partNumber are required")
	}

	partNumber, err := strconv.Atoi(partNumberStr)
	if err != nil {
		if isS3Api {
			return returnS3Error(c, http.StatusBadRequest, "InvalidArgument", "invalid partNumber", "/"+bucket+"/"+key)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "invalid partNumber")
	}

	body := c.Request().Body
	defer body.Close()

	etag, err := cmrt.UploadPart(conn, bucket, key, uploadID, partNumber, body, c.Request().ContentLength)
	if err != nil {
		if isS3Api {
			errorCode := "InternalError"
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				errorCode = "NoSuchUpload"
				statusCode = http.StatusNotFound
			}
			return returnS3Error(c, statusCode, errorCode, err.Error(), "/"+bucket+"/"+key)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if isS3Api {
		addS3Headers(c)
		c.Response().Header().Set("ETag", etag)
		return c.NoContent(http.StatusOK)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"ETag":       etag,
		"PartNumber": partNumberStr,
	})
}
