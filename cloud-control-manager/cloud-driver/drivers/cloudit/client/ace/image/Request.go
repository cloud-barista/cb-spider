package image

import (
	cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/sirupsen/logrus"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type ImageReqInfo struct {
	Name         string `json:"name" required:"true"`
	VolumeId     string `json:"volumeId" required:"true"`   // 정지된 서버 볼륨을 기준으로 이미지 템플릿 생성
	SnapshotId   string `json:"snapshotId" required:"true"` // 서버 스냅샷을 기준으로 이미지 템플릿 생성
	Ownership    string `json:"ownership" required:"true"`  // TENANT, PRIVATE
	Format       string `json:"format" required:"true"`     // raw, vdi, vmdk, vpc, qcow2
	SourceType   string `json:"sourceType" required:"true"` // server, snapshot
	TemplateType string `json:"templateType" required:"true"`
	Size         int    `json:"size" required:"false"`
	PoolId       string `json:"poolId" required:"false"`
	Protection   int    `json:"protection" required:"false"`
}

type ImageInfo struct {
	ID            string
	TenantID      string
	ClusterID     string
	ClusterName   string
	Size          int
	RealSize      int
	RefCount      int
	Name          string
	CreatedAt     string
	Ownership     string // 테넌트 소유, 개인 소유
	TemplateType  string
	State         string
	Protection    int
	OS            string
	Arch          string // Architecture
	Format        string
	Enabled       int
	Description   string
	PoolID        string
	PoolName      string
	SnapshotID    string
	Creator       string
	VolumeID      string
	Url           string
	Pause         int
	SourceType    string
	MinKvmVersion int
}

func List(restClient *client.RestClient, requestOpts *client.RequestOpts) (*[]ImageInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "templates")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var image []ImageInfo
	if err := result.ExtractInto(&image); err != nil {
		return nil, err
	}
	return &image, nil
}

func Get(restClient *client.RestClient, templateId string, requestOpts *client.RequestOpts) (*ImageInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "templates", templateId)
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var image ImageInfo
	if err := result.ExtractInto(&image); err != nil {
		return nil, err
	}
	return &image, nil
}

func Create(restClient *client.RestClient, requestOpts *client.RequestOpts) (*ImageInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "templates")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Post(requestURL, nil, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var image ImageInfo
	if err := result.ExtractInto(&image); err != nil {
		return nil, err
	}
	return &image, nil
}

func Delete(restClient *client.RestClient, templateId string, requestOpts *client.RequestOpts) error {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "templates", templateId)
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Delete(requestURL, requestOpts); result.Err != nil {
		return result.Err
	}
	return nil
}
