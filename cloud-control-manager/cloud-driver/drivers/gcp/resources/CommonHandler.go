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
	"time"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	compute "google.golang.org/api/compute/v1"
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

func GetImageInfo(client *compute.Service, projectId string, imageName string) (*compute.Image, error) {
	imageResp, err := client.Images.Get(projectId, imageName).Do()
	if err != nil {
		cblogger.Error(err)
		return &compute.Image{}, err
	}

	return imageResp, nil
}
