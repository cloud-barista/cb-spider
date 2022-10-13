package disk

import (
	"errors"
	"fmt"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var cblogger *logrus.Logger

func init() {
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type DiskReqInfo struct {
	Name      string `json:"name,omitempty" required:"true"`
	ID        string `json:"volumeId,omitempty"`
	ClusterId string `json:"clusterId,omitempty" required:"true"`
	Size      int    `json:"size,omitempty" required:"true"`
	Mode      string `json:"mode,omitempty"`
}

type DiskInfo struct {
	ID        string
	Name      string `updateAble:"Name"`
	State     string
	Size      int `updateAble:"Size"`
	CreatedAt string

	// miscellaneous properties
	TemplateId  string `responseType:"KeyValue"`
	ClusterId   string `responseType:"KeyValue"`
	PoolId      string `responseType:"KeyValue"`
	Bootable    string `responseType:"KeyValue"`
	Iops        int    `responseType:"KeyValue"`
	Throughput  int    `responseType:"KeyValue"`
	Creator     string `responseType:"KeyValue"`
	Description string `responseType:"KeyValue" updateAble:"Description"`
	Dev         string `responseType:"KeyValue"`
}

func (diskInfo *DiskInfo) GetKeyValues() []irs.KeyValue {
	var keyValues []irs.KeyValue

	valueOfThis := reflect.ValueOf(*diskInfo)
	typeOfThis := reflect.TypeOf(*diskInfo)

	for idx := 0; idx < valueOfThis.NumField(); idx++ {
		field := typeOfThis.Field(idx)
		if field.Tag.Get("responseType") == "KeyValue" {
			strValue, strOk := valueOfThis.Field(idx).Interface().(string)
			if strOk {
				keyValues = append(keyValues, irs.KeyValue{Key: field.Name, Value: strValue})
			}
			intValue, intOk := valueOfThis.Field(idx).Interface().(int)
			if intOk {
				keyValues = append(keyValues, irs.KeyValue{Key: field.Name, Value: strconv.Itoa(intValue)})
			}
		}
	}

	return keyValues
}

func (diskInfo *DiskInfo) ToIRSDisk(restClient *client.RestClient) irs.DiskInfo {
	// 볼륨이 장착된 VM 이름 획득
	ownerVm, _ := GetOwnerVm(restClient, diskInfo.ID, &client.RequestOpts{
		MoreHeaders: restClient.AuthenticatedHeaders(),
	})

	var createdTime time.Time
	if diskInfo.CreatedAt != "" {
		timeArr := strings.Split(diskInfo.CreatedAt, " ")
		timeFormatStr := fmt.Sprintf("%sT%sZ", timeArr[0], timeArr[1])
		if createTime, err := time.Parse(time.RFC3339, timeFormatStr); err == nil {
			createdTime = createTime
		}
	}

	return irs.DiskInfo{
		IId:          irs.IID{NameId: diskInfo.Name, SystemId: diskInfo.ID},
		DiskType:     "",
		DiskSize:     strconv.Itoa(diskInfo.Size),
		Status:       getDiskStatus(diskInfo.State),
		CreatedTime:  createdTime,
		KeyValueList: diskInfo.GetKeyValues(),
		OwnerVM:      ownerVm.IId,
	}
}

func (diskInfo *DiskInfo) ToUpdateDiskReqInfo() *DiskReqInfo {
	diskReqInfo := DiskReqInfo{}

	valueOfThis := reflect.ValueOf(*diskInfo)
	typeOfThis := reflect.TypeOf(*diskInfo)
	valueOfReqInfo := reflect.ValueOf(&diskReqInfo).Elem()
	typeOfReqInfo := reflect.TypeOf(diskReqInfo)

	for idxThisField := 0; idxThisField < valueOfThis.NumField(); idxThisField++ {
		mappedFieldName := typeOfThis.Field(idxThisField).Tag.Get("updateAble")
		_, exist := typeOfReqInfo.FieldByName(mappedFieldName)
		if exist {
			reqInfoFieldValue := valueOfReqInfo.FieldByName(mappedFieldName)
			if reqInfoFieldValue.CanSet() {
				strValue, strOk := valueOfThis.Field(idxThisField).Interface().(string)
				if strOk {
					reqInfoFieldValue.SetString(strValue)
				}
				intValue, intOk := valueOfThis.Field(idxThisField).Interface().(int)
				if intOk {
					reqInfoFieldValue.SetInt(int64(intValue))
				}
			}
		}
	}

	if diskReqInfo == (DiskReqInfo{}) {
		return nil
	}

	return &diskReqInfo
}

func getDiskStatus(diskStatus string) irs.DiskStatus {
	var resultStatus string
	switch strings.ToLower(diskStatus) {
	case "creating":
		resultStatus = "Creating"
	case "deleting":
		resultStatus = "Deleting"
	case "available":
		resultStatus = "Available"
	case "attaching", "detaching", "in_use", "converting", "extending":
		resultStatus = "Attached"
	case "failed":
		resultStatus = "Failed"
	default:
		resultStatus = "Failed"
	}

	return irs.DiskStatus(resultStatus)
}

func List(restClient *client.RestClient, requestOpts *client.RequestOpts) (*[]DiskInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "volumes")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var diskList []DiskInfo
	if err := result.ExtractInto(&diskList); err != nil {
		return nil, err
	}

	return &diskList, nil
}

func Get(restClient *client.RestClient, id string, requestOpts *client.RequestOpts) (*DiskInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "volumes")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var diskList []DiskInfo
	if err := result.ExtractInto(&diskList); err != nil {
		return nil, err
	}
	for _, disk := range diskList {
		if disk.ID == id {
			return &disk, nil
		}
	}

	return nil, errors.New("cannot find disk")
}

func Create(restClient *client.RestClient, requestOpts *client.RequestOpts) (*DiskInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "volumes")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Post(requestURL, nil, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var disk DiskInfo
	if err := result.ExtractInto(&disk); err != nil {
		return nil, err
	}

	return &disk, nil
}

func Update(restClient *client.RestClient, id string, requestOpts *client.RequestOpts) (*DiskInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "volumes", id, "update")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Post(requestURL, nil, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var disk DiskInfo
	if err := result.ExtractInto(&disk); err != nil {
		return nil, err
	}

	return &disk, nil
}

func Delete(restClient *client.RestClient, id string, requestOpts *client.RequestOpts) error {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "volumes", id)
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Delete(requestURL, requestOpts); result.Err != nil {
		return result.Err
	}

	return nil
}

func GetOwnerVm(restClient *client.RestClient, id string, requestOpts *client.RequestOpts) (irs.VMInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "volumes", id, "servers")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return irs.VMInfo{}, result.Err
	}

	var ownerVmList []struct {
		VmId   string
		VmName string
	}
	if err := result.ExtractInto(&ownerVmList); err != nil || len(ownerVmList) == 0 {
		return irs.VMInfo{}, err
	}

	return irs.VMInfo{IId: irs.IID{NameId: ownerVmList[0].VmName, SystemId: ownerVmList[0].VmId}}, nil
}
