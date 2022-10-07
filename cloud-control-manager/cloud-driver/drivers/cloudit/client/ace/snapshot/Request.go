package snapshot

import (
	"errors"
	"fmt"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/server"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type SnapshotReqInfo struct {
	Name     string `json:"name,omitempty"`
	VolumeId string `json:"volumeId,omitempty"`
}

type SnapshotInfo struct {
	Name       string
	Id         string
	ServerName string
	State      string
	CreatedAt  string

	// miscellaneous properties
	Bootable   string
	Creator    string
	TemplateId string
}

func ToIRSMyImage(restClient *client.RestClient, associatedSnapshots *[]SnapshotInfo) (irs.MyImageInfo, error) {
	var result irs.MyImageInfo

	isAvailable := true
	result.Status = irs.MyImageAvailable
	result.SourceVM.SystemId = ""
	for _, snapshot := range *associatedSnapshots {
		if snapshot.Bootable == "yes" {
			result.IId.NameId = strings.Split(snapshot.Name, "-dev-")[0]
			result.IId.SystemId = snapshot.Id
			result.SourceVM.NameId = snapshot.ServerName
			if snapshot.CreatedAt != "" {
				timeArr := strings.Split(snapshot.CreatedAt, " ")
				timeFormatStr := fmt.Sprintf("%sT%sZ", timeArr[0], timeArr[1])
				if createTime, err := time.Parse(time.RFC3339, timeFormatStr); err == nil {
					result.CreatedTime = createTime
				}
			}
		}
		if isAvailable && getSnapshotStatus(snapshot.State) == irs.MyImageUnavailable {
			isAvailable = false
			result.Status = irs.MyImageUnavailable
		}
	}

	requestOpts := client.RequestOpts{
		MoreHeaders: restClient.AuthenticatedHeaders(),
	}
	serverList, err := server.List(restClient, &requestOpts)
	if err != nil {
		return irs.MyImageInfo{}, err
	}
	for _, server := range *serverList {
		if server.Name == result.SourceVM.NameId {
			result.SourceVM.SystemId = server.ID
			break
		}
	}

	if result.SourceVM.NameId == "" {
		result.SourceVM.NameId = "Deleted"
	}
	if result.SourceVM.SystemId == "" {
		result.SourceVM.SystemId = "Deleted"
	}

	return result, nil
}

func List(restClient *client.RestClient, requestOpts *client.RequestOpts) (*[]SnapshotInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "snapshots")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var snapshotList []SnapshotInfo
	if err := result.ExtractInto(&snapshotList); err != nil {
		return nil, err
	}

	return &snapshotList, nil
}

func Get(restClient *client.RestClient, snapshotId string, requestOpts *client.RequestOpts) (SnapshotInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "snapshots")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return SnapshotInfo{}, result.Err
	}

	var snapshotList []SnapshotInfo
	if err := result.ExtractInto(&snapshotList); err != nil {
		return SnapshotInfo{}, err
	}

	for _, snapshot := range snapshotList {
		if snapshot.Id == snapshotId {
			return snapshot, nil
		}
	}

	return SnapshotInfo{}, errors.New("Snapshot not found")
}

//func GetSnapshotsByMyImage(restClient *client.RestClient, myImageNameId string, requestOpts *client.RequestOpts) (SnapshotInfo, error) {
//
//}

func CreateSnapshot(restClient *client.RestClient, requestOpts *client.RequestOpts) (SnapshotInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "snapshots")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Post(requestURL, nil, &result.Body, requestOpts); result.Err != nil {
		return SnapshotInfo{}, result.Err
	}

	var snapshot SnapshotInfo
	if err := result.ExtractInto(&snapshot); err != nil {
		return SnapshotInfo{}, err
	}

	return snapshot, nil
}

func DeleteSnapshot(restClient *client.RestClient, snapshotId string, requestOpts *client.RequestOpts) (bool, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "snapshots", snapshotId)
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Delete(requestURL, requestOpts); result.Err != nil {
		return false, result.Err
	}

	return true, nil
}

func CreateVolumeBySnapshot(restClient *client.RestClient, snapshotId string, requestOpts *client.RequestOpts) (bool, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "snapshots", snapshotId, "volume")
	cblogger.Info(requestURL)

	var result client.Result
	restClient.Post(requestURL, nil, &result.Body, requestOpts)

	return true, nil
}

func getSnapshotStatus(snapshotStatus string) irs.MyImageStatus {
	var resultStatus string
	switch strings.ToLower(snapshotStatus) {
	case "completed":
		resultStatus = "Available"
	case "creating", "deleting", "converting", "failed":
		resultStatus = "Unavailable"
	default:
		resultStatus = "Unavailable"
	}

	return irs.MyImageStatus(resultStatus)
}
