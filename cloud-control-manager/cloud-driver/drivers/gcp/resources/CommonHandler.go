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
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	compute "google.golang.org/api/compute/v1"
	container "google.golang.org/api/container/v1"
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

func GetKeyValueList(i map[string]interface{}) []irs.KeyValue {
	var keyValueList []irs.KeyValue
	for k, v := range i {
		//cblogger.Infof("K:[%s]====>", k)
		_, ok := v.(string)
		if !ok {
			cblogger.Errorf("Key[%s]의 값은 변환 불가", k)
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
			cblogger.Infof("WaitUntilComplete / [%s]", err)
			return err
		}
		cblogger.Infof("==> 상태 : 진행율 : [%d] / [%s]", opSatus.Progress, opSatus.Status)

		//PENDING, RUNNING, or DONE.
		if (opSatus.Status == "RUNNING" || opSatus.Status == "DONE") && opSatus.Progress >= 100 {
			//if opSatus.Status == "RUNNING" || opSatus.Status == "DONE" {
			//if opSatus.Status == "DONE" {
			cblogger.Info("Wait을 종료합니다.", resourceId, ":", opSatus.Status)
			return nil
		}

		time.Sleep(time.Second * 1)
		after_time := time.Now()
		diff := after_time.Sub(before_time)
		if int(diff.Seconds()) > max_time {
			cblogger.Errorf("[%d]초 동안 리소스[%s]의 상태가 완료되지 않아서 Wait을 강제로 종료함.", max_time, resourceId)
			return errors.New("장시간 요청 작업이 완료되지 않아서 Wait을 강제로 종료함.)")
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
			cblogger.Infof("WaitUntilOperationComplete / [%s]", err)
			return err
		}
		cblogger.Infof("==> 상태 : 진행율 : [%d] / [%s]", opSatus.Progress, opSatus.Status)

		//PENDING, RUNNING, or DONE.
		if (opSatus.Status == "RUNNING" || opSatus.Status == "DONE") && opSatus.Progress >= 100 {
			//if opSatus.Status == "RUNNING" || opSatus.Status == "DONE" {
			//if opSatus.Status == "DONE" {
			cblogger.Info("Wait을 종료합니다.", resourceId, ":", opSatus.Status)
			return nil
		}

		time.Sleep(time.Second * 1)
		after_time := time.Now()
		diff := after_time.Sub(before_time)
		if int(diff.Seconds()) > max_time {
			cblogger.Errorf("[%d]초 동안 리소스[%s]의 상태가 완료되지 않아서 Wait을 강제로 종료함.", max_time, resourceId)
			return errors.New("장시간 요청 작업이 완료되지 않아서 Wait을 강제로 종료함.)")
		}
	}

	return nil
}

// Get 공통으로 사용
func GetDiskInfo(client *compute.Service, credential idrv.CredentialInfo, region idrv.RegionInfo, diskName string) (*compute.Disk, error) {
	projectID := credential.ProjectID
	zone := region.Zone

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
	// spew.Dump(imageResp)
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
			cblogger.Errorf("[%s] 프로젝트 소유의 이미지 목록 조회 실패!", projectId)
			cblogger.Error(err)
			return nil, err
		}

		nextPageToken = res.NextPageToken
		cblogger.Info("NestPageToken : ", nextPageToken)

		for {
			cblogger.Info("Loop?")
			for _, item := range res.Items {
				cnt++
				if strings.EqualFold(reqImageName, item.SelfLink) {
					cblogger.Info("found Image : ", item)
					return item, nil
				}
				cblogger.Info("cnt : ", item)
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
			cblogger.Infof("WaitUntilOperationComplete / [%s]", err)
			return err
		}
		cblogger.Infof("==> 상태 : 진행율 : [%d] / [%s]", opSatus.Progress, opSatus.Status)

		//PENDING, RUNNING, or DONE.

		// STATUS_UNSPECIFIED 	Not set.
		// PENDING 	The operation has been created.
		// RUNNING 	The operation is currently running.
		// DONE 	The operation is done, either cancelled or completed.
		// ABORTING 	The operation is aborting.
		if opSatus.Status == "DONE" {
			cblogger.Info("Wait을 종료합니다.", resourceId, ":", opSatus.Status)
			return nil
		}

		time.Sleep(time.Second * 1)
		after_time := time.Now()
		diff := after_time.Sub(before_time)
		if int(diff.Seconds()) > max_time {
			cblogger.Errorf("[%d]초 동안 리소스[%s]의 상태가 완료되지 않아서 Wait을 강제로 종료함.", max_time, resourceId)
			return errors.New("장시간 요청 작업이 완료되지 않아서 Wait을 강제로 종료함.)")
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
		spew.Dump(opSatus)
		cblogger.Infof("==> 상태 : 진행율 : [%d] / [%s]", opSatus.Progress, opSatus.Status)

		//PENDING, RUNNING, or DONE.

		// STATUS_UNSPECIFIED 	Not set.
		// PENDING 	The operation has been created.
		// RUNNING 	The operation is currently running.
		// DONE 	The operation is done, either cancelled or completed.
		// ABORTING 	The operation is aborting.
		if opSatus.Status == "ABORTING" {
			cblogger.Info("Wait을 종료합니다.", resourceId, ":", opSatus.Status)
			return nil
		}

		time.Sleep(time.Second * 5)
		after_time := time.Now()
		diff := after_time.Sub(before_time)
		if int(diff.Seconds()) > max_time {
			cblogger.Errorf("[%d]초 동안 리소스[%s]의 상태가 완료되지 않아서 Wait을 강제로 종료함.", max_time, resourceId)
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
			cblogger.Infof("WaitContainerOperationDone / [%s]", err)
			return err
		}
		spew.Dump(opSatus)
		cblogger.Infof("==> 상태 : 진행율 : [%d] / [%s]", opSatus.Progress, opSatus.Status)

		//PENDING, RUNNING, or DONE.

		// STATUS_UNSPECIFIED 	Not set.
		// PENDING 	The operation has been created.
		// RUNNING 	The operation is currently running.
		// DONE 	The operation is done, either cancelled or completed.
		// ABORTING 	The operation is aborting.
		if opSatus.Status == "DONE" {
			cblogger.Info("Wait을 종료합니다.", resourceId, ":", opSatus.Status)
			return nil
		}

		time.Sleep(time.Second * 5)
		after_time := time.Now()
		diff := after_time.Sub(before_time)
		if int(diff.Seconds()) > maxTime {
			cblogger.Errorf("[%d]초 동안 리소스[%s]의 상태가 완료되지 않아서 Wait을 강제로 종료함.", maxTime, resourceId)
			return nil
		}
	}

	return nil
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
