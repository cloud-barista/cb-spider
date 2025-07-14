// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// program by ysjeon@mz.co.kr, 2019.07.
// modify by devunet@mz.co.kr, 2019.11.

package resources

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	filestore "cloud.google.com/go/filestore/apiv1"
	filestorepb "cloud.google.com/go/filestore/apiv1/filestorepb"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	compute "google.golang.org/api/compute/v1"
	container "google.golang.org/api/container/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

const (
	CBVMUser = "cscservice"
	//CBKeyPairPath = "/cloud-control-manager/cloud-driver/driver-libs/.ssh-gcp/"
	// by powerkim, 2019.10.30
	CBKeyPairPath     = "/meta_db/.ssh-gcp/"
	CBKeyPairProvider = "GCP"
)

const CBDefaultVNetName string = "cb-vnet"   // CB Default Virtual Network Name
const CBDefaultSubnetName string = "cb-vnet" // CB Default Subnet Name
const KEY_VALUE_CONVERT_DEBUG_INFO bool = false

const OperationGlobal = 1
const OperationRegion = 2
const OperationZone = 3

type GcpCBNetworkInfo struct {
	VpcName   string
	VpcId     string
	CidrBlock string
	IsDefault bool
	State     string

	SubnetName string
	SubnetId   string
}

var once sync.Once
var cblogger *logrus.Logger
var calllogger *logrus.Logger

func InitLog() {
	once.Do(func() {
		// cblog is a global variable.
		cblogger = cblog.GetLogger("CB-SPIDER")
		calllogger = call.GetLogger("HISCALL")
	})
}

func LoggingError(hiscallInfo call.CLOUDLOGSCHEMA, err error) {
	hiscallInfo.ErrorMSG = err.Error()
	calllogger.Error(call.String(hiscallInfo))
}

func LoggingInfo(hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) {
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
}

func GetCallLogScheme(region idrv.RegionInfo, resourceType call.RES_TYPE, resourceName string, apiName string) call.CLOUDLOGSCHEMA {
	cblogger.Info(fmt.Sprintf("Call %s %s", call.GCP, apiName))
	return call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   region.Region,
		ResourceType: resourceType,
		ResourceName: resourceName,
		CloudOSAPI:   apiName,
	}
}

// VPC
func GetCBDefaultVNetName() string {
	return CBDefaultVNetName
}

// Subnet
func GetCBDefaultSubnetName() string {
	return CBDefaultSubnetName
}

// KeyValue gen func
func GetKeyValueList(i map[string]interface{}) []irs.KeyValue {
	var keyValueList []irs.KeyValue
	for k, v := range i {
		//cblogger.Infof("K:[%s]====>", k)
		_, ok := v.(string)
		if !ok {
			cblogger.Errorf("The value for key [%s] cannot be converted.", k)
			continue
		}
		//if strings.EqualFold(k, "users") {
		//	continue
		//}
		//cblogger.Infof("====>", v)
		keyValueList = append(keyValueList, irs.KeyValue{k, v.(string)})
		cblogger.Info("getKeyValueList : ", keyValueList)
	}

	return keyValueList
}

// Cloud Object를 CB-KeyValue 형식으로 변환이 필요할 경우 이용
func ConvertKeyValueList(v interface{}) ([]irs.KeyValue, error) {
	//cblogger.Debug(v)
	var keyValueList []irs.KeyValue
	var i map[string]interface{}

	jsonBytes, errJson := json.Marshal(v)
	if errJson != nil {
		cblogger.Error("KeyValue conversion failed")
		cblogger.Error(errJson)
		return nil, errJson
	}

	json.Unmarshal(jsonBytes, &i)

	for k, v := range i {
		if KEY_VALUE_CONVERT_DEBUG_INFO {
			cblogger.Debugf("K:[%s]====>", k)
		}
		/*
			cblogger.Infof("v:[%s]====>", reflect.ValueOf(v))

			vv := reflect.ValueOf(v)
			cblogger.Infof("value ====>[%s]", vv.String())
			s := fmt.Sprint(v)
			cblogger.Infof("value2 ====>[%s]", s)
		*/
		//value := fmt.Sprint(v)
		value, errString := ConvertToString(v)
		if errString != nil {
			//cblogger.Debugf("Key[%s]의 값은 변환 불가 - [%s]", k, errString) //요구에 의해서 Error에서 Warn으로 낮춤
			continue
		}
		keyValueList = append(keyValueList, irs.KeyValue{k, value})

		/*
			_, ok := v.(string)
			if !ok {
				cblogger.Errorf("Key[%s] value cannot be converted", k)
				continue
			}
			keyValueList = append(keyValueList, irs.KeyValue{k, v.(string)})
		*/
	}
	cblogger.Debug("getKeyValueList : ", keyValueList)
	//keyValueList = append(keyValueList, irs.KeyValue{"test", typeToString([]float32{3.14, 1.53, 2.0000000000000})})

	return keyValueList, nil
}

// CB-KeyValue 등을 위해 String 타입으로 변환
func ConvertToString(value interface{}) (string, error) {
	if value == nil {
		if KEY_VALUE_CONVERT_DEBUG_INFO {
			cblogger.Debugf("Nil Value")
		}
		return "", errors.New("Nil. Value")
	}

	var result string
	t := reflect.ValueOf(value)
	if KEY_VALUE_CONVERT_DEBUG_INFO {
		cblogger.Debug("==>ValueOf : ", t)
	}

	switch value.(type) {
	case float32:
		result = strconv.FormatFloat(t.Float(), 'f', -1, 32) // f, fmt, prec, bitSize
	case float64:
		result = strconv.FormatFloat(t.Float(), 'f', -1, 64) // f, fmt, prec, bitSize
		//strconv.FormatFloat(instanceTypeInfo.MemorySize, 'f', 0, 64)

	default:
		if KEY_VALUE_CONVERT_DEBUG_INFO {
			cblogger.Debug("--> default type:", reflect.ValueOf(value).Type())
		}
		result = fmt.Sprint(value)
	}

	return result, nil
}

// KeyPair 해시 생성 함수
func CreateHashString(credentialInfo idrv.CredentialInfo) (string, error) {
	keyString := credentialInfo.ClientId + credentialInfo.ClientSecret + credentialInfo.TenantId + credentialInfo.SubscriptionId
	hasher := md5.New()
	_, err := io.WriteString(hasher, keyString)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// Public KeyPair 정보 가져오기
func GetPublicKey(credentialInfo idrv.CredentialInfo, keyPairName string) (string, error) {
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	hashString, err := CreateHashString(credentialInfo)
	if err != nil {
		return "", err
	}

	publicKeyPath := keyPairPath + hashString + "--" + keyPairName + ".pub"
	publicKeyBytes, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		return "", err
	}
	return string(publicKeyBytes), nil
}

func hasInstanceGroup(client *compute.Service, credential idrv.CredentialInfo, region idrv.RegionInfo, instanceGroup string) (bool, error) {
	projectID := credential.ProjectID
	zone := region.Zone
	// Attempt to get the instance group to verify if it exists
	instanceGroupGet, err := client.InstanceGroups.Get(projectID, zone, instanceGroup).Do()
	if err != nil {
		// Check if the error is a "not found" error
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}

	return instanceGroupGet != nil, nil
}

// InstanceGroup의 인스턴스 목록 return
func GetInstancesOfInstanceGroup(client *compute.Service, credential idrv.CredentialInfo, region idrv.RegionInfo, instanceGroup string) ([]string, error) {
	projectID := credential.ProjectID
	zone := region.Zone

	rb := &compute.InstanceGroupsListInstancesRequest{
		// TODO: Add desired fields of the request body.
	}

	instanceGroupsListInstances, err := client.InstanceGroups.ListInstances(projectID, zone, instanceGroup, rb).Do()
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	//cblogger.Info("instanceGroupsListInstances : ", instanceGroupsListInstances)
	var instanceList []string
	for _, instance := range instanceGroupsListInstances.Items {
		instanceUrl := instance.Instance
		urlArr := strings.Split(instanceUrl, "/")
		instanceName := urlArr[len(urlArr)-1]
		instanceList = append(instanceList, instanceName)
	}
	//cblogger.Info("instanceList : ", instanceList)

	return instanceList, nil
}

// Instance 정보조회
func GetInstance(client *compute.Service, credential idrv.CredentialInfo, region idrv.RegionInfo, instance string) (*compute.Instance, error) {
	projectID := credential.ProjectID
	zone := region.Zone

	instanceInfo, err := client.Instances.Get(projectID, zone, instance).Do()
	if err != nil {
		return nil, err
	}

	return instanceInfo, nil
}

// Operation 이 완료 될 때까지 기다림.
func WaitUntilComplete(client *compute.Service, project string, region string, resourceId string, isGlobalAction bool) error {
	before_time := time.Now()
	max_time := 300 //최대 300초간 체크

	var opSatus *compute.Operation
	var err error

	for {
		if isGlobalAction {
			opSatus, err = client.GlobalOperations.Get(project, resourceId).Do()
		} else {
			opSatus, err = client.RegionOperations.Get(project, region, resourceId).Do()
		}
		if err != nil {
			cblogger.Errorf("WaitUntilComplete / [%s]", err)
			return err
		}
		cblogger.Infof("==> Status: Progress: [%d] / [%s]", opSatus.Progress, opSatus.Status)

		//PENDING, RUNNING, or DONE.
		if (opSatus.Status == "RUNNING" || opSatus.Status == "DONE") && opSatus.Progress >= 100 {
			//if opSatus.Status == "RUNNING" || opSatus.Status == "DONE" {
			//if opSatus.Status == "DONE" {
			cblogger.Info("Exiting Wait.", resourceId, ":", opSatus.Status)
			return nil
		}

		time.Sleep(time.Second * 1)
		after_time := time.Now()
		diff := after_time.Sub(before_time)
		if int(diff.Seconds()) > max_time {
			cblogger.Errorf("Forcing termination of Wait because the status of resource [%s] has not completed within [%d] seconds.", max_time, resourceId)
			return errors.New("Forcing termination of Wait due to the request operation not completing for a long time.")
		}
	}

	return nil
}

func WaitOperationComplete(client *compute.Service, project string, region string, zone string, resourceId string, operationType int) error {
	before_time := time.Now()
	max_time := 300 //최대 300초간 체크

	var opSatus *compute.Operation
	var err error

	for {
		switch operationType {
		case OperationGlobal:
			opSatus, err = client.GlobalOperations.Get(project, resourceId).Do()
		case OperationRegion:
			opSatus, err = client.RegionOperations.Get(project, region, resourceId).Do()
		case OperationZone:
			opSatus, err = client.ZoneOperations.Get(project, zone, resourceId).Do()
		}
		if err != nil {
			cblogger.Errorf("WaitUntilOperationComplete / [%s]", err)
			return err
		}
		cblogger.Infof("==> Status: Progress: [%d] / [%s]", opSatus.Progress, opSatus.Status)

		//PENDING, RUNNING, or DONE.
		if (opSatus.Status == "RUNNING" || opSatus.Status == "DONE") && opSatus.Progress >= 100 {
			//if opSatus.Status == "RUNNING" || opSatus.Status == "DONE" {
			//if opSatus.Status == "DONE" {
			cblogger.Info("Exiting Wait.", resourceId, ":", opSatus.Status)
			return nil
		}

		time.Sleep(time.Second * 1)
		after_time := time.Now()
		diff := after_time.Sub(before_time)
		if int(diff.Seconds()) > max_time {
			cblogger.Errorf("Forcing termination of Wait because the status of resource [%s] has not completed within [%d] seconds.", max_time, resourceId)
			return errors.New("Forcing termination of Wait due to the request operation not completing for a long time.")
		}
	}

	return nil
}

// Get 공통으로 사용
func GetDiskInfo(client *compute.Service, credential idrv.CredentialInfo, region idrv.RegionInfo, diskName string) (*compute.Disk, error) {
	projectID := credential.ProjectID
	zone := region.Zone
	targetZone := region.TargetZone

	// 대상 zone이 다른경우 targetZone을 사용
	if targetZone != "" {
		zone = targetZone
	}
	diskResp, err := client.Disks.Get(projectID, zone, diskName).Do()
	if err != nil {
		cblogger.Error(err)
		return &compute.Disk{}, err
	}

	return diskResp, nil
}

func GetMachineImageInfo(client *compute.Service, projectId string, imageName string) (*compute.MachineImage, error) {
	cblogger.Infof("projectId : [%s] / imageName : [%s]", projectId, imageName)
	imageResp, err := client.MachineImages.Get(projectId, imageName).Do()
	if err != nil {
		cblogger.Error(err)
		return &compute.MachineImage{}, err
	}
	if imageResp == nil {
		return nil, errors.New("Not Found : [" + imageName + "] Image information not found")
	}
	// cblogger.Infof("result ", imageResp)
	// cblogger.Debug(imageResp)
	return imageResp, nil
}

// IID 에서 systemID로 image 조회.  : systemID가 URL로 되어있어 필요한 값들을 추출하여 사용. projectId, imageName
func GetPublicImageInfo(client *compute.Service, imageIID irs.IID) (*compute.Image, error) {
	projectId := ""
	imageName := ""

	arrLink := strings.Split(imageIID.SystemId, "/")
	if len(arrLink) > 0 {
		imageName = arrLink[len(arrLink)-1]
		for pos, item := range arrLink {
			if strings.EqualFold(item, "projects") {
				projectId = arrLink[pos+1]
				break
			}
		}
	}
	cblogger.Infof("projectId : [%s] / imageName : [%s]", projectId, imageName)
	if projectId == "" {
		return nil, errors.New("ProjectId information not found in URL.")
	}

	image, err := client.Images.Get(projectId, imageName).Do()
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	return image, nil

}

// IID 에서 systemID로 image 조회.
func FindImageByID(client *compute.Service, imageIID irs.IID) (*compute.Image, error) {
	reqImageName := imageIID.SystemId

	//https://cloud.google.com/compute/docs/images?hl=ko
	arrImageProjectList := []string{
		//"ubuntu-os-cloud",

		"gce-uefi-images", // 보안 VM을 지원하는 이미지

		//보안 VM을 지원하지 않는 이미지들
		"centos-cloud",
		"cos-cloud",
		"coreos-cloud",
		"debian-cloud",
		"rhel-cloud",
		"rhel-sap-cloud",
		"suse-cloud",
		"suse-sap-cloud",
		"ubuntu-os-cloud",
		"windows-cloud",
		"windows-sql-cloud",
	}

	cnt := 0
	nextPageToken := ""
	var req *compute.ImagesListCall
	var res *compute.ImageList
	var err error
	for _, projectId := range arrImageProjectList {
		req = client.Images.List(projectId)
		//req.Filter("name=" + reqImageName)
		//req.Filter("SelfLink=" + reqImageName)

		res, err = req.Do()
		if err != nil {
			cblogger.Errorf("[%s] Failed to retrieve the list of project-owned images", projectId)
			cblogger.Error(err)
			return nil, err
		}

		nextPageToken = res.NextPageToken
		cblogger.Info("NestPageToken : ", nextPageToken)

		for {
			cblogger.Debug("Loop?")
			for _, item := range res.Items {
				cnt++
				if strings.EqualFold(reqImageName, item.SelfLink) {
					cblogger.Debug("found Image : ", item)
					return item, nil
				}
				cblogger.Debug("cnt : ", item)
			}
		}
	}
	return nil, errors.New("Not Found : [" + reqImageName + "] Image information not found")

}

// container 의 operation
func WaitContainerOperationComplete(client *container.Service, project string, region string, zone string, resourceId string, operationType int) error {
	before_time := time.Now()
	max_time := 300 //최대 300초간 체크

	var opSatus *container.Operation
	var err error

	operationName := "projects/" + project + "/locations/" + zone + "/operations/" + resourceId
	for {
		opSatus, err = client.Projects.Locations.Operations.Get(operationName).Do()
		if err != nil {
			cblogger.Errorf("WaitUntilOperationComplete / [%s]", err)
			return err
		}
		cblogger.Infof("==> Status: Progress: [%d] / [%s]", opSatus.Progress, opSatus.Status)

		//PENDING, RUNNING, or DONE.

		// STATUS_UNSPECIFIED 	Not set.
		// PENDING 	The operation has been created.
		// RUNNING 	The operation is currently running.
		// DONE 	The operation is done, either cancelled or completed.
		// ABORTING 	The operation is aborting.
		if opSatus.Status == "DONE" {
			cblogger.Info("Exiting Wait.", resourceId, ":", opSatus.Status)
			return nil
		}

		time.Sleep(time.Second * 1)
		after_time := time.Now()
		diff := after_time.Sub(before_time)
		if int(diff.Seconds()) > max_time {
			cblogger.Errorf("Forcing termination of Wait because the status of resource [%s] has not completed within [%d] seconds.", max_time, resourceId)
			return errors.New("Forcing termination of Wait due to the request operation not completing for a long time.")
		}
	}

	return nil
}

// 30초동안 Fail 이 떨어지지 않으면 성공
func WaitContainerOperationFail(client *container.Service, project string, region string, zone string, resourceId string, operationType int) error {
	before_time := time.Now()
	max_time := 30

	var opSatus *container.Operation
	var err error

	operationName := "projects/" + project + "/locations/" + zone + "/operations/" + resourceId
	for {
		opSatus, err = client.Projects.Locations.Operations.Get(operationName).Do()
		if err != nil {
			cblogger.Infof("WaitContainerOperationFail / [%s]", err)
			return err
		}
		cblogger.Debug(opSatus)

		if opSatus.Progress != nil && len(opSatus.Progress.Metrics) > 0 && opSatus.Progress.Metrics[0] != nil {
			cblogger.Infof("==> Status: Progress: [%d] / [%s]", opSatus.Progress.Metrics[0].IntValue, opSatus.Status)
		}

		//PENDING, RUNNING, or DONE.

		// STATUS_UNSPECIFIED 	Not set.
		// PENDING 	The operation has been created.
		// RUNNING 	The operation is currently running.
		// DONE 	The operation is done, either cancelled or completed.
		// ABORTING 	The operation is aborting.
		if opSatus.Status == "ABORTING" {
			cblogger.Info("Exiting Wait.", resourceId, ":", opSatus.Status)
			return nil
		}

		time.Sleep(time.Second * 5)
		after_time := time.Now()
		diff := after_time.Sub(before_time)
		if int(diff.Seconds()) > max_time {
			cblogger.Infof("Forcing termination of Wait because the status of resource [%s] has not failed within [%d] seconds.", resourceId, max_time)
			return nil
		}
	}

	return nil
}

// 20분
func WaitContainerOperationDone(client *container.Service, project string, region string, zone string, resourceId string, operationType int, maxTime int) error {
	before_time := time.Now()

	var opSatus *container.Operation
	var err error

	operationName := "projects/" + project + "/locations/" + zone + "/operations/" + resourceId
	for {
		opSatus, err = client.Projects.Locations.Operations.Get(operationName).Do()
		if err != nil {
			cblogger.Errorf("WaitContainerOperationDone / [%s]", err)
			return err
		}
		cblogger.Debug(opSatus)
		cblogger.Infof("==> Status: Progress: [%d] / [%s]", opSatus.Progress, opSatus.Status)

		//PENDING, RUNNING, or DONE.

		// STATUS_UNSPECIFIED 	Not set.
		// PENDING 	The operation has been created.
		// RUNNING 	The operation is currently running.
		// DONE 	The operation is done, either cancelled or completed.
		// ABORTING 	The operation is aborting.
		if opSatus.Status == "DONE" {
			cblogger.Info("Exiting Wait.", resourceId, ":", opSatus.Status)
			return nil
		}

		time.Sleep(time.Second * 5)
		after_time := time.Now()
		diff := after_time.Sub(before_time)
		if int(diff.Seconds()) > maxTime {
			cblogger.Errorf("Forcing termination of Wait because the status of resource [%s] has not completed within [%d] seconds.", maxTime, resourceId)
			return nil
		}
	}

	return nil
}

// 리전 목록 조회
func ListRegion(client *compute.Service, projectId string) (*compute.RegionList, error) {

	if projectId == "" {
		return nil, errors.New("ProjectId not found.")
	}

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   "",
		ResourceType: call.REGIONZONE,
		ResourceName: "",
		CloudOSAPI:   "List()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	resp, err := client.Regions.List(projectId).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return nil, err
	}
	return resp, nil
}

// region 조회
// GCP에서 region은 regionName과 regionUri로 구분 됨. regionName으로 찾는 function임.
func GetRegion(client *compute.Service, projectId string, regionName string) (*compute.Region, error) {

	if projectId == "" {
		return nil, errors.New("ProjectId not found.")
	}

	if regionName == "" {
		return nil, errors.New("Region Name not found.")
	}

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   regionName,
		ResourceType: call.REGIONZONE,
		ResourceName: "",
		CloudOSAPI:   "Get()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	resp, err := client.Regions.Get(projectId, regionName).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return nil, err
	}
	return resp, nil

}

// region에 해당하는 zone 목록 조회
// filter조건으로 사용하는 region조건은 regionUrl로 넘겨야 함.
// filter조건 자체가 string이며 regionUrl에 특수문자가 있고 따옴표로 감싸야만 결과가 정상적으로 나옴
// region="xxx/xxx/xxx" 형태로 보내야하며
// ` ` 로 감싸야 함.
// filter := "region=https://www.googleapis.com/compute/v1/projects/xxx/regions/us-east1" -> error return.
// filter := `region="https://www.googleapis.com/compute/v1/projects/xxx/regions/us-east1"` -> 조회결과 옴
// filter := `region="us-east1"`// -> 조회결과 없음
func GetZoneListByRegion(client *compute.Service, projectId string, regionUrl string) (*compute.ZoneList, error) {

	if projectId == "" {
		return nil, errors.New("Project information not found")
	}
	if regionUrl == "" {
		return nil, errors.New("Region information not found")
	}

	filter := `region="` + regionUrl + `"`

	resp, err := client.Zones.List(projectId).Filter(filter).Do()

	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	// cblogger.Debug(resp)
	return resp, nil
}

func GetZonesByRegion(client *compute.Service, projectId string, region string) (*compute.ZoneList, error) {

	// region에 대한 selflink를 만들어서 GetZoneListByRegion 호출
	if projectId == "" {
		return nil, errors.New("Project information not found")
	}
	if region == "" {
		return nil, errors.New("Region information not found")
	}

	regionUrl := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/regions/%s", projectId, region)
	resp, err := GetZoneListByRegion(client, projectId, regionUrl)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	return resp, nil
}

// Available or Unavailable 로 return
// Status of the zone, either UP or DOWN. (지원하지 않는 경우 NotSupported)
func GetZoneStatus(status string) irs.ZoneStatus {
	if status == "UP" {
		return irs.ZoneAvailable
	} else {
		return irs.ZoneUnavailable
	}
}

/*
### container operation ###
(*container.Operation)(0xc0003d6a00)({
 ClusterConditions: ([]*container.StatusCondition) <nil>,
 Detail: (string) "",
 EndTime: (string) "",
 Error: (*container.Status)(<nil>),
 Location: (string) "",
 Name: (string) (len=32) "operation-1670486783109-b7af5968",
 NodepoolConditions: ([]*container.StatusCondition) <nil>,
 OperationType: (string) (len=14) "CREATE_CLUSTER",
 Progress: (*container.OperationProgress)(<nil>),
 SelfLink: (string) (len=125) "https://container.googleapis.com/v1/projects/244703045150/zones/asia-northeast3-a/operations/operation-1670486783109-b7af5968",
 StartTime: (string) (len=30) "2022-12-08T08:06:23.109241982Z",
 Status: (string) (len=7) "RUNNING",
 StatusMessage: (string) "",
 TargetLink: (string) (len=97) "https://container.googleapis.com/v1/projects/244703045150/zones/asia-northeast3-a/clusters/pmks08",
 Zone: (string) (len=17) "asia-northeast3-a",
 ServerResponse: (googleapi.ServerResponse) {
  HTTPStatusCode: (int) 200,
  Header: (http.Header) (len=9) {
   (string) (len=12) "Content-Type": ([]string) (len=1 cap=1) {
    (string) (len=31) "application/json; charset=UTF-8"
   },
   (string) (len=4) "Vary": ([]string) (len=3 cap=4) {
    (string) (len=6) "Origin",
    (string) (len=8) "X-Origin",
    (string) (len=7) "Referer"
   },
   (string) (len=4) "Date": ([]string) (len=1 cap=1) {
    (string) (len=29) "Thu, 08 Dec 2022 08:06:23 GMT"
   },
   (string) (len=15) "X-Frame-Options": ([]string) (len=1 cap=1) {
    (string) (len=10) "SAMEORIGIN"
   },
   (string) (len=22) "X-Content-Type-Options": ([]string) (len=1 cap=1) {
    (string) (len=7) "nosniff"
   },
   (string) (len=7) "Alt-Svc": ([]string) (len=1 cap=1) {
    (string) (len=162) "h3=\":443\"; ma=2592000,h3-29=\":443\"; ma=2592000,h3-Q050=\":443\"; ma=2592000,h3-Q046=\":443\"; ma=2592000,h3-Q043=\":443\"; ma=2592000,quic=\":443\"; ma=2592000; v=\"46,43\""
   },
   (string) (len=6) "Server": ([]string) (len=1 cap=1) {
    (string) (len=3) "ESF"
   },
   (string) (len=13) "Cache-Control": ([]string) (len=1 cap=1) {
    (string) (len=7) "private"
   },
   (string) (len=16) "X-Xss-Protection": ([]string) (len=1 cap=1) {
    (string) (len=1) "0"
   }
  }
 },
 ForceSendFields: ([]string) <nil>,
 NullFields: ([]string) <nil>
})
*/

// GCP Filestore API Common Functions

func ListFilestoreInstancesByRegionAndZones(filestoreClient *filestore.CloudFilestoreManagerClient, computeClient *compute.Service, credential idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) ([]*filestorepb.Instance, error) {
	var instances []*filestorepb.Instance
	regionParent := FormatParentPath(credential.ProjectID, region.Region)
	regionInstances, err := ListFilestoreInstances(filestoreClient, ctx, regionParent)
	if err != nil {
		cblogger.Error("error while listing file system instances")
		return nil, err
	}
	cblogger.Debug("regionInstances", regionInstances)
	instances = append(instances, regionInstances...)

	// 해당 connection에 대한 region
	zonesByRegion, err := GetZonesByRegion(computeClient, credential.ProjectID, region.Region)
	if err != nil {
		cblogger.Error("error while listing zones by region")
		return nil, err
	}
	cblogger.Debug("zonesByRegion ", zonesByRegion)

	for _, zone := range zonesByRegion.Items {
		//cblogger.Debug("zone", zone)
		zoneParent := FormatParentPath(credential.ProjectID, zone.Name)
		cblogger.Debug("zoneParent ", zoneParent)
		zoneInstances, err := ListFilestoreInstances(filestoreClient, ctx, zoneParent)
		if err != nil {
			cblogger.Error("failed to listing file system instances")
			continue
		}
		instances = append(instances, zoneInstances...)
	}
	return instances, nil
}

// ListFilestoreInstances lists all filestore instances in a given parent location
func ListFilestoreInstances(filestoreClient *filestore.CloudFilestoreManagerClient, ctx context.Context, parent string) ([]*filestorepb.Instance, error) {
	var instances []*filestorepb.Instance
	//projects/csta-349809/locations/asia-northeast3/instances
	cblogger.Debug("ListFilestoreInstances parent : ", parent)
	req := &filestorepb.ListInstancesRequest{
		Parent: parent,
	}

	it := filestoreClient.ListInstances(ctx, req)
	for {
		instance, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate filestore instances: %v", err)
		}
		instances = append(instances, instance)
	}

	return instances, nil
}

// GetFilestoreInstance gets a specific filestore instance by name : parent 구조로 된 이름이어야 함.
func GetFilestoreInstance(filestoreClient *filestore.CloudFilestoreManagerClient, ctx context.Context, name string) (*filestorepb.Instance, error) {
	cblogger.Debug("GetFilestoreInstance name: ", name)
	req := &filestorepb.GetInstanceRequest{
		Name: name,
	}

	instance, err := filestoreClient.GetInstance(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get filestore instance %s: %v", name, err)
	}

	return instance, nil
}

// CreateFilestoreInstance creates a new filestore instance
func CreateFilestoreInstance(filestoreClient *filestore.CloudFilestoreManagerClient, ctx context.Context, req *filestorepb.CreateInstanceRequest) (*filestorepb.Instance, error) {

	op, err := filestoreClient.CreateInstance(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create filestore instance: %v", err)
	}

	// Wait for operation to complete
	result, err := op.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for filestore instance creation: %v", err)
	}

	return result, nil
}

// DeleteFilestoreInstance deletes a filestore instance
func DeleteFilestoreInstance(filestoreClient *filestore.CloudFilestoreManagerClient, ctx context.Context, name string) error {
	req := &filestorepb.DeleteInstanceRequest{
		Name: name,
	}

	// name
	// name := "projects/csta-349809/locations/asia-northeast1-b/instances/testfs01"

	op, err := filestoreClient.DeleteInstance(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete filestore instance %s: %v", name, err)
	}

	// Wait for operation to complete
	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for filestore instance deletion: %v", err)
	}

	return nil
}

// FormatParentPath formats the parent path for filestore API calls
// filestore에서 parent는 zone임. In Filestore, locations map to Google Cloud zones, for example us-west1-b.
func FormatParentPath(projectID, zone string) string {
	return fmt.Sprintf("projects/%s/locations/%s", projectID, zone)
}

func FormatFilestoreInstanceName(projectID, zone, instanceName string) string {
	return fmt.Sprintf("projects/%s/locations/%s/instances/%s", projectID, zone, instanceName)
}

// "projects/%s/locations/%s/instances/%s" 이 형태에서 마지막 부분을 추출
func ExtractFilestoreName(instanceName string) string {
	parentName := strings.Split(instanceName, "/")
	return parentName[len(parentName)-1]
}

// func extractFileSystemInfo(fs *filestorepb.Instance, vNetInfo *irs.VPCInfo) *irs.FileSystemInfo {
// 	cblogger.Debug("extractFileSystemInfo")
// 	fsInstanceName := fs.Name

// 	parentFormat := regexp.MustCompile(`^projects/[^/]+/locations/([^/]+)/instances/([^/]+)$`)
// 	matches := parentFormat.FindStringSubmatch(fsInstanceName)

// 	if len(matches) != 3 {
// 		cblogger.Error("유효하지 않은 인스턴스 이름 형식: %s", fsInstanceName)
// 		return nil
// 	}

// 	location := matches[1]
// 	fsName := matches[2]

// 	fsType := irs.FileSystemType(fs.Tier)
// 	fsCapacity := fs.FileShares[0].CapacityGb
// 	fsLabels := fs.Labels
// 	fsNetwork := fs.Networks[0]
// 	fsEtag := fs.Etag
// 	fsStatus := irs.FileSystemStatus(fs.State)

// 	cblogger.Debug("extractFileSystemInfo")
// 	cblogger.Debug("fsName: ", fsName)
// 	cblogger.Debug("fsType: ", fsType)
// 	cblogger.Debug("fsCapacity: ", fsCapacity)
// 	cblogger.Debug("fsLabels: ", fsLabels)
// 	cblogger.Debug("fsNetwork: ", fsNetwork)
// 	cblogger.Debug("fsEtag: ", fsEtag)

// 	// Tier는 hdd/ssd/ .
// 	// Instance_TIER_UNSPECIFIED Instance_Tier = 0
// 	// // STANDARD tier. BASIC_HDD is the preferred term for this tier.
// 	// Instance_STANDARD Instance_Tier = 1
// 	// // PREMIUM tier. BASIC_SSD is the preferred term for this tier.
// 	// Instance_PREMIUM Instance_Tier = 2
// 	// // BASIC instances offer a maximum capacity of 63.9 TB.
// 	// // BASIC_HDD is an alias for STANDARD Tier, offering economical
// 	// // performance backed by HDD.
// 	// Instance_BASIC_HDD Instance_Tier = 3
// 	// // BASIC instances offer a maximum capacity of 63.9 TB.
// 	// // BASIC_SSD is an alias for PREMIUM Tier, and offers improved
// 	// // performance backed by SSD.
// 	// Instance_BASIC_SSD Instance_Tier = 4
// 	// // HIGH_SCALE instances offer expanded capacity and performance scaling
// 	// // capabilities.
// 	// Instance_HIGH_SCALE_SSD Instance_Tier = 5
// 	// // ENTERPRISE instances offer the features and availability needed for
// 	// // mission-critical workloads.
// 	// Instance_ENTERPRISE Instance_Tier = 6
// 	// // ZONAL instances offer expanded capacity and performance scaling
// 	// // capabilities.
// 	// Instance_ZONAL Instance_Tier = 7
// 	// // REGIONAL instances offer the features and availability needed for
// 	// // mission-critical workloads.
// 	// Instance_REGIONAL Instance_Tier = 8

// 	fsRegion := ""
// 	fsZone := ""

// 	switch fs.Tier {
// 	case filestorepb.Instance_ZONAL:
// 		fsType = irs.ZoneType
// 		fsRegion = location[:len(location)-2]
// 		fsZone = location
// 	case filestorepb.Instance_REGIONAL:
// 		fsType = irs.RegionType
// 		fsRegion = location
// 		fsZone = ""
// 	}

// 	// vpc 조회
// 	// subnetList
// 	subnetList := []irs.IID{}
// 	for _, subnet := range vNetInfo.SubnetInfoList {
// 		subnetList = append(subnetList, subnet.IId)
// 	}

// 	fsNfsVersion := ""

// 	switch fs.Protocol {
// 	case filestorepb.Instance_NFS_V3:
// 		fsNfsVersion = "3.0"
// 	case filestorepb.Instance_NFS_V4_1:
// 		fsNfsVersion = "4.1"
// 	}

// 	performanceInfo := map[string]string{}
// 	pc := fs.PerformanceConfig
// 	if pc.GetIopsPerTb() != nil {
// 		performanceInfo["MaxIopsPerTiB"] = strconv.FormatInt(pc.GetIopsPerTb().GetMaxIopsPerTb(), 10)
// 	}
// 	if pc.GetFixedIops() != nil {
// 		performanceInfo["MaxIops"] = strconv.FormatInt(pc.GetFixedIops().GetMaxIops(), 10)
// 	}

// 	pl := fs.PerformanceLimits
// 	if pl != nil {
// 		performanceInfo["Limit_MaxIops"] = strconv.FormatInt(pl.GetMaxIops(), 10)
// 		performanceInfo["Limit_MaxReadIops"] = strconv.FormatInt(pl.GetMaxReadIops(), 10)
// 		performanceInfo["Limit_MaxWriteIops"] = strconv.FormatInt(pl.GetMaxWriteIops(), 10)
// 		performanceInfo["Limit_MaxReadThroughputBytePerSecond"] = strconv.FormatInt(pl.GetMaxReadThroughputBps(), 10)
// 		performanceInfo["Limit_MaxWriteThroughputBytePerSecond"] = strconv.FormatInt(pl.GetMaxWriteThroughputBps(), 10)
// 	}

// 	// Performance options, e.g., {"Tier": "STANDARD"}, {"ThroughputMode": "provisioned", "Throughput": "128"}; CSP default if omitted

// 	//
// 	// UsedSizeGB      int64             `json:"UsedSizeGB" validate:"required" example:"256"` // Current used size in GB.
// 	// MountTargetList []MountTargetInfo `json:"MountTargetList,omitempty"`

// 	// CreatedTime  time.Time  `json:"CreatedTime" validate:"required"`

// 	tagList := []irs.KeyValue{}
// 	for key, value := range fsLabels {
// 		tagList = append(tagList, irs.KeyValue{Key: key, Value: value})
// 	}

// 	fsInfo := irs.FileSystemInfo{
// 		IId: irs.IID{
// 			SystemId: fsName,
// 		},
// 		Region:           fsRegion,
// 		Zone:             fsZone,
// 		VpcIID:           vNetInfo.IId,
// 		AccessSubnetList: subnetList,
// 		Encryption:       true, // kmsKeyName을 이용하여 암호화 처리
// 		// 	BackupSchedule : irs.FileSystemBackupInfo{
// 		// 		FileSystemIID : fsName,
// 		//	}
// 		FileSystemType: fsType,
// 		NFSVersion:     fsNfsVersion,
// 		CapacityGB:     fsCapacity,

// 		Status:  fsStatus,
// 		TagList: tagList,
// 	}
// 	return &fsInfo
// }

// 	cblogger.Debug("extractFileSystemInfo")
// 	fsInstanceName := fs.Name

// 	parentFormat := regexp.MustCompile(`^projects/[^/]+/locations/([^/]+)/instances/([^/]+)$`)
// 	matches := parentFormat.FindStringSubmatch(fsInstanceName)

// 	if len(matches) != 3 {
// 		cblogger.Error("유효하지 않은 인스턴스 이름 형식: %s", fsInstanceName)
// 		return nil
// 	}

// 	location := matches[1]
// 	fsName := matches[2]

// 	fsType := irs.FileSystemType(fs.Tier)
// 	fsCapacity := fs.FileShares[0].CapacityGb
// 	fsLabels := fs.Labels
// 	fsNetwork := fs.Networks[0]
// 	fsEtag := fs.Etag
// 	fsStatus := irs.FileSystemStatus(fs.State)

// 	cblogger.Debug("extractFileSystemInfo")
// 	cblogger.Debug("fsName: ", fsName)
// 	cblogger.Debug("fsType: ", fsType)
// 	cblogger.Debug("fsCapacity: ", fsCapacity)
// 	cblogger.Debug("fsLabels: ", fsLabels)
// 	cblogger.Debug("fsNetwork: ", fsNetwork)
// 	cblogger.Debug("fsEtag: ", fsEtag)

// 	// Tier는 hdd/ssd/ .
// 	// Instance_TIER_UNSPECIFIED Instance_Tier = 0
// 	// // STANDARD tier. BASIC_HDD is the preferred term for this tier.
// 	// Instance_STANDARD Instance_Tier = 1
// 	// // PREMIUM tier. BASIC_SSD is the preferred term for this tier.
// 	// Instance_PREMIUM Instance_Tier = 2
// 	// // BASIC instances offer a maximum capacity of 63.9 TB.
// 	// // BASIC_HDD is an alias for STANDARD Tier, offering economical
// 	// // performance backed by HDD.
// 	// Instance_BASIC_HDD Instance_Tier = 3
// 	// // BASIC instances offer a maximum capacity of 63.9 TB.
// 	// // BASIC_SSD is an alias for PREMIUM Tier, and offers improved
// 	// // performance backed by SSD.
// 	// Instance_BASIC_SSD Instance_Tier = 4
// 	// // HIGH_SCALE instances offer expanded capacity and performance scaling
// 	// // capabilities.
// 	// Instance_HIGH_SCALE_SSD Instance_Tier = 5
// 	// // ENTERPRISE instances offer the features and availability needed for
// 	// // mission-critical workloads.
// 	// Instance_ENTERPRISE Instance_Tier = 6
// 	// // ZONAL instances offer expanded capacity and performance scaling
// 	// // capabilities.
// 	// Instance_ZONAL Instance_Tier = 7
// 	// // REGIONAL instances offer the features and availability needed for
// 	// // mission-critical workloads.
// 	// Instance_REGIONAL Instance_Tier = 8

// 	fsRegion := ""
// 	fsZone := ""

// 	switch fs.Tier {
// 	case filestorepb.Instance_ZONAL:
// 		fsType = irs.ZoneType
// 		fsRegion = location[:len(location)-2]
// 		fsZone = location
// 	case filestorepb.Instance_REGIONAL:
// 		fsType = irs.RegionType
// 		fsRegion = location
// 		fsZone = ""
// 	}

// 	// vpc 조회
// 	// subnetList
// 	subnetList := []irs.IID{}
// 	for _, subnet := range vNetInfo.SubnetInfoList {
// 		subnetList = append(subnetList, subnet.IId)
// 	}

// 	fsNfsVersion := ""

// 	switch fs.Protocol {
// 	case filestorepb.Instance_NFS_V3:
// 		fsNfsVersion = "3.0"
// 	case filestorepb.Instance_NFS_V4_1:
// 		fsNfsVersion = "4.1"
// 	}

// 	performanceInfo := map[string]string{}
// 	pc := fs.PerformanceConfig
// 	if pc.GetIopsPerTb() != nil {
// 		performanceInfo["MaxIopsPerTiB"] = strconv.FormatInt(pc.GetIopsPerTb().GetMaxIopsPerTb(), 10)
// 	}
// 	if pc.GetFixedIops() != nil {
// 		performanceInfo["MaxIops"] = strconv.FormatInt(pc.GetFixedIops().GetMaxIops(), 10)
// 	}

// 	pl := fs.PerformanceLimits
// 	if pl != nil {
// 		performanceInfo["Limit_MaxIops"] = strconv.FormatInt(pl.GetMaxIops(), 10)
// 		performanceInfo["Limit_MaxReadIops"] = strconv.FormatInt(pl.GetMaxReadIops(), 10)
// 		performanceInfo["Limit_MaxWriteIops"] = strconv.FormatInt(pl.GetMaxWriteIops(), 10)
// 		performanceInfo["Limit_MaxReadThroughputBytePerSecond"] = strconv.FormatInt(pl.GetMaxReadThroughputBps(), 10)
// 		performanceInfo["Limit_MaxWriteThroughputBytePerSecond"] = strconv.FormatInt(pl.GetMaxWriteThroughputBps(), 10)
// 	}

// 	// Performance options, e.g., {"Tier": "STANDARD"}, {"ThroughputMode": "provisioned", "Throughput": "128"}; CSP default if omitted

// 	//
// 	// UsedSizeGB      int64             `json:"UsedSizeGB" validate:"required" example:"256"` // Current used size in GB.
// 	// MountTargetList []MountTargetInfo `json:"MountTargetList,omitempty"`

// 	// CreatedTime  time.Time  `json:"CreatedTime" validate:"required"`

// 	tagList := []irs.KeyValue{}
// 	for key, value := range fsLabels {
// 		tagList = append(tagList, irs.KeyValue{Key: key, Value: value})
// 	}

// 	fsInfo := irs.FileSystemInfo{
// 		IId: irs.IID{
// 			SystemId: fsName,
// 		},
// 		Region:           fsRegion,
// 		Zone:             fsZone,
// 		VpcIID:           vNetInfo.IId,
// 		AccessSubnetList: subnetList,
// 		Encryption:       true, // kmsKeyName을 이용하여 암호화 처리
// 		// 	BackupSchedule : irs.FileSystemBackupInfo{
// 		// 		FileSystemIID : fsName,
// 		//	}
// 		FileSystemType: fsType,
// 		NFSVersion:     fsNfsVersion,
// 		CapacityGB:     fsCapacity,

// 		Status:  fsStatus,
// 		TagList: tagList,
// 	}
// 	return &fsInfo
// }

func GetVpcInfo(client *compute.Service, credential idrv.CredentialInfo, region idrv.RegionInfo, vpcIID irs.IID) (irs.VPCInfo, error) {
	infoVPC, err := client.Networks.Get(credential.ProjectID, vpcIID.SystemId).Do()
	if err != nil {
		cblogger.Error(err)
		return irs.VPCInfo{}, err
	}

	subnetInfoList := []irs.SubnetInfo{}
	if infoVPC.Subnetworks != nil {
		for _, item := range infoVPC.Subnetworks {
			str := strings.Split(item, "/")
			subnet := str[len(str)-1]
			infoSubnet, err := GetSubnetListInfo(client, credential, region, subnet)
			if err != nil {
				cblogger.Error(err)
				return irs.VPCInfo{}, err
			}
			subnetInfoList = append(subnetInfoList, mappingSubnet(infoSubnet))
		}

	}

	vpcInfo := irs.VPCInfo{
		IId: irs.IID{
			NameId: infoVPC.Name,
			//SystemId: strconv.FormatUint(infoVPC.Id, 10),
			SystemId: infoVPC.Name,
		},
		IPv4_CIDR:      "GCP VPC does not support IPv4_CIDR",
		SubnetInfoList: subnetInfoList,
	}
	// 2025-03-13 StructToKeyValueList 사용으로 변경
	vpcInfo.KeyValueList = irs.StructToKeyValueList(infoVPC)

	return vpcInfo, nil
}

func GetSubnetListInfo(client *compute.Service, credential idrv.CredentialInfo, region idrv.RegionInfo, subnet string) (*compute.Subnetwork, error) {
	infoSubnet, err := client.Subnetworks.Get(credential.ProjectID, region.Region, subnet).Do()
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	return infoSubnet, nil
}

func extractFileSystemInfo(
	computeClient *compute.Service,
	filestoreClient *filestore.CloudFilestoreManagerClient,
	monitoringClient *monitoring.MetricClient,
	credential idrv.CredentialInfo,
	region idrv.RegionInfo,
	fileStoreInstances []*filestorepb.Instance) ([]*irs.FileSystemInfo, error) {

	fsInfoList := []*irs.FileSystemInfo{}
	vpcInfoMap := make(map[string]irs.VPCInfo)
	for _, instance := range fileStoreInstances {

		// fileStore 정보 설정
		fsInstanceName := instance.Name
		parentFormat := regexp.MustCompile(`^projects/[^/]+/locations/([^/]+)/instances/([^/]+)$`)
		matches := parentFormat.FindStringSubmatch(fsInstanceName)

		if len(matches) != 3 {
			cblogger.Error("invalid instance name format: %s", fsInstanceName)
			//return nil, errors.New("유효하지 않은 인스턴스 이름 형식: " + fsInstanceName)
			continue
		}

		location := matches[1]
		fsName := matches[2] // fileStoreName := ExtractFilestoreName(instance.Name) 이것도 가능한데 위치정보랑 추출한 것을 그대로 사용

		fsType := irs.FileSystemType("")
		if instance.Tier == filestorepb.Instance_ZONAL {
			fsType = irs.ZoneType
		} else if instance.Tier == filestorepb.Instance_REGIONAL {
			fsType = irs.RegionType
		}

		fsCapacity := instance.FileShares[0].CapacityGb
		fsLabels := instance.Labels
		//fsNetwork := instance.Networks[0]
		fsVPCNetworkName := instance.Networks[0].Network
		fsFileStoreIP := instance.Networks[0].IpAddresses[0]
		fsReservedIpRange := instance.Networks[0].ReservedIpRange

		// Convert enum values to strings for joining
		var modeStrings []string
		for _, mode := range instance.Networks[0].Modes {
			modeStrings = append(modeStrings, mode.String())
		}
		fsIPStackTypes := strings.Join(modeStrings, ",")

		//fsEtag := instance.Etag
		//fsStatus := irs.FileSystemStatus(instance.State)
		fsStatus := irs.FileSystemStatus("")

		if instance.State == filestorepb.Instance_CREATING {
			fsStatus = irs.FileSystemCreating
		} else if instance.State == filestorepb.Instance_READY {
			fsStatus = irs.FileSystemAvailable
		} else if instance.State == filestorepb.Instance_DELETING {
			fsStatus = irs.FileSystemDeleting
		} else if instance.State == filestorepb.Instance_ERROR {
			fsStatus = irs.FileSystemError
		}

		cblogger.Debug("extractFileSystemInfo")
		cblogger.Debug("fsName: ", fsName)
		cblogger.Debug("fsType: ", fsType)
		cblogger.Debug("fsCapacity: ", fsCapacity)
		cblogger.Debug("fsLabels: ", fsLabels)
		cblogger.Debug("fsVPCNetworkName: ", fsVPCNetworkName)
		cblogger.Debug("fsFileStoreIP: ", fsFileStoreIP)
		cblogger.Debug("fsReservedIpRange: ", fsReservedIpRange)
		cblogger.Debug("fsStatus: ", fsStatus)

		fsRegion := ""
		fsZone := ""

		switch instance.Tier { // Tier에 따라서 location이 region인지 zone인지 구별 가능
		case filestorepb.Instance_ZONAL:
			fsType = irs.ZoneType
			fsRegion = location[:len(location)-2]
			fsZone = location
		case filestorepb.Instance_REGIONAL:
			fsType = irs.RegionType
			fsRegion = location
			fsZone = ""
		}

		cblogger.Debug("fsRegion: ", fsRegion)
		cblogger.Debug("fsZone: ", fsZone)

		// vpc 조회
		vpcSystemId := fsVPCNetworkName // 현재버전에서 1개의 vpc만 지원 됨. subnet은 지정 불가
		if _, ok := vpcInfoMap[vpcSystemId]; !ok {
			vpcIID := irs.IID{SystemId: vpcSystemId}
			vpcInfo, errVnet := GetVpcInfo(computeClient, credential, region, vpcIID)
			if errVnet != nil {
				cblogger.Error(errVnet)
				//return nil, errVnet
				continue
			}
			vpcInfoMap[vpcSystemId] = vpcInfo
		}
		currentVpcInfo := vpcInfoMap[vpcSystemId]
		// subnetList
		subnetList := []irs.IID{}
		for _, subnet := range currentVpcInfo.SubnetInfoList {
			subnetList = append(subnetList, subnet.IId)
		}

		// cblogger.Debug("VpcInfo: ", currentVpcInfo)
		// cblogger.Debug("subnetList: ", subnetList)

		fsNfsVersion := ""
		switch instance.Protocol {
		case filestorepb.Instance_NFS_V3:
			fsNfsVersion = string(GCPNFSVersion(NFS_V3))
		case filestorepb.Instance_NFS_V4_1:
			fsNfsVersion = string(GCPNFSVersion(NFS_V4_1))
		}
		cblogger.Debug("fsNfsVersion: ", fsNfsVersion)

		performanceInfo := map[string]string{}
		pc := instance.PerformanceConfig
		if pc.GetIopsPerTb() != nil {
			performanceInfo["MaxIopsPerTiB"] = strconv.FormatInt(pc.GetIopsPerTb().GetMaxIopsPerTb(), 10)
		}
		if pc.GetFixedIops() != nil {
			performanceInfo["MaxIops"] = strconv.FormatInt(pc.GetFixedIops().GetMaxIops(), 10)
		}

		pl := instance.PerformanceLimits
		if pl != nil {
			performanceInfo["Limit_MaxIops"] = strconv.FormatInt(pl.GetMaxIops(), 10)
			performanceInfo["Limit_MaxReadIops"] = strconv.FormatInt(pl.GetMaxReadIops(), 10)
			performanceInfo["Limit_MaxWriteIops"] = strconv.FormatInt(pl.GetMaxWriteIops(), 10)
			performanceInfo["Limit_MaxReadThroughputBytePerSecond"] = strconv.FormatInt(pl.GetMaxReadThroughputBps(), 10)
			performanceInfo["Limit_MaxWriteThroughputBytePerSecond"] = strconv.FormatInt(pl.GetMaxWriteThroughputBps(), 10)
		}
		cblogger.Debug("performanceInfo: ", performanceInfo)

		// 	//
		// 	// UsedSizeGB      int64             `json:"UsedSizeGB" validate:"required" example:"256"` // Current used size in GB.
		// 	// MountTargetList []MountTargetInfo `json:"MountTargetList,omitempty"`

		// 	// CreatedTime  time.Time  `json:"CreatedTime" validate:"required"`

		endpoint := fsFileStoreIP + ":/" + fsName
		mountTargetKeyList := []irs.KeyValue{
			{Key: "ipRanges", Value: fsReservedIpRange},
			{Key: "ipStackTypes", Value: fsIPStackTypes},
			{Key: "ipAddresses", Value: fsFileStoreIP},
		}

		//subnetList
		mountTarget := irs.MountTargetInfo{
			// subnet은 매핑불가. vpc의 subnet과 다름. SecurityGroup도 지정 불가
			//SecurityGroups:      ,
			Endpoint:            endpoint, // instance 접속 ip
			MountCommandExample: "sudo mount -o hard,intr,rw,vers=" + fsNfsVersion + ",tcp " + endpoint + " /mnt/" + fsName,
			KeyValueList:        mountTargetKeyList,
		}

		tagList := []irs.KeyValue{}
		for key, value := range fsLabels {
			tagList = append(tagList, irs.KeyValue{Key: key, Value: value})
		}
		cblogger.Debug("tagList: ", tagList)

		// fileShares
		if len(instance.FileShares) > 0 {
			// mount 정보 설정 : 모니터링 api 에서 가져오기
		}

		fsInfo := irs.FileSystemInfo{}
		fsInfo.IId = irs.IID{SystemId: fsName}
		fsInfo.Region = fsRegion
		fsInfo.Zone = fsZone
		fsInfo.VpcIID = currentVpcInfo.IId
		fsInfo.AccessSubnetList = subnetList
		fsInfo.Encryption = true
		fsInfo.FileSystemType = fsType
		fsInfo.NFSVersion = fsNfsVersion
		fsInfo.CapacityGB = fsCapacity
		fsInfo.Status = fsStatus
		fsInfo.MountTargetList = []irs.MountTargetInfo{mountTarget}
		fsInfo.TagList = tagList

		fsInfoList = append(fsInfoList, &fsInfo)

	} // end of for
	return fsInfoList, nil
}
