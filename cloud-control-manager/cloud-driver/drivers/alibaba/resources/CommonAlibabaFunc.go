// Cloud Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by devunet@mz.co.kr, 2019.09.

package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

const (
	// default Resource GROUP Name
	CBResourceGroupName = "CB-GROUP"
	// default VPC Name
	CBVirutalNetworkName = "CB-VNet"
	// default CIDR Block
	CBVnetDefaultCidr = "130.0.0.0/16"
	// default VM User Name
	CBDefaultVmUserName = "cb-user"

	// default Subnet Name
	CBSubnetName = "CB-VNet-Sub"

	// default Bandwidth is 5 Mbit/s
	CBBandwidth = "5"
	// default InstanceChargeType
	CBInstanceChargeType = "PostPaid"
	// default InternetChargeType
	CBInternetChargeType = "PayByTraffic"

	// default Tag Name
	CBMetaDefaultTagName = "cbCate"
	// default Tag Value
	CBMetaDefaultTagValue = "cbAlibaba"

	CBPageOn = true
	// page number for control pages
	CBPageNumber = 1 // 페이지 시작은 1부터 시작되기 때문에 삭제 예정

	// page size for control pages
	CBPageSize = 100 //오브젝트(예: 이미지 / 키페어 / ...) 마다 지정 개수가 달라서 삭제 예정

	CBKeyPairPath = "/meta_db/.ssh-aliyun/"
	//CBCloudInitFilePath = "/cloud-driver-libs/.cloud-init-aliyun/cloud-init"
	CBCloudInitFilePath = "/cloud-driver-libs/.cloud-init-common/cloud-init"
)

type AlibabaCBNetworkInfo struct {
	VpcName   string
	VpcId     string
	CidrBlock string
	IsDefault bool
	State     string

	SubnetName string
	SubnetId   string
}

func GetCBResourceGroupName() string {
	return CBResourceGroupName
}

//VPC
func GetCBVirutalNetworkName() string {
	return CBVirutalNetworkName
}

//Subnet
func GetCBSubnetName() string {
	return CBSubnetName
}

func GetCBVnetDefaultCidr() string {
	return CBVnetDefaultCidr
}

func SetNameTag(Client *ecs.Client, resourceId string, resourceType string, value string) bool {
	// Tag에 Name 설정
	cblogger.Infof("Name Tage 설정 - ResourceId : [%s]  Value : [%s] ", resourceId, value)

	request := ecs.CreateAddTagsRequest()
	request.Scheme = "https"

	request.ResourceType = resourceType // "disk", "instance", "image", "securitygroup", "snapshot"
	request.ResourceId = resourceId     // "i-t4n4qtfwa4w5aavx588v"
	request.Tag = &[]ecs.AddTagsTag{
		{
			Key:   "Name",
			Value: value, // "cbVal",
		},
		{
			Key:   "cbCate",
			Value: "cbAlibaba",
		},
		{
			Key:   "cbName",
			Value: value, // "cbVal",
		},
		// Resources: []*string{&Id},
	}
	_, errtag := Client.AddTags(request)
	if errtag != nil {
		cblogger.Error("Name Tag 설정 실패 : ")
		cblogger.Error(errtag)
		return false
	}

	return true
}

//Cloud Object를 JSON String 타입으로 변환
func ConvertJsonString(v interface{}) (string, error) {
	jsonBytes, errJson := json.Marshal(v)
	if errJson != nil {
		cblogger.Error("JSON 변환 실패")
		cblogger.Error(errJson)
		return "", errJson
	}

	jsonString := string(jsonBytes)

	return jsonString, nil
}

//CB-KeyValue 등을 위해 String 타입으로 변환
func ConvertToString(value interface{}) (string, error) {
	if value == nil {
		cblogger.Error("Nil Value")
		return "", errors.New("NIL Value")
	}

	var result string
	t := reflect.ValueOf(value)
	cblogger.Debug("==>ValueOf : ", t)

	switch value.(type) {
	case float32:
		result = strconv.FormatFloat(t.Float(), 'f', -1, 32) // f, fmt, prec, bitSize
	case float64:
		result = strconv.FormatFloat(t.Float(), 'f', -1, 64) // f, fmt, prec, bitSize
		//strconv.FormatFloat(instanceTypeInfo.MemorySize, 'f', 0, 64)

	default:
		cblogger.Debug("--> default type:", reflect.ValueOf(value).Type())
		result = fmt.Sprint(value)
	}

	return result, nil
}

//Cloud Object를 CB-KeyValue 형식으로 변환이 필요할 경우 이용
func ConvertKeyValueList(v interface{}) ([]irs.KeyValue, error) {
	spew.Dump(v)

	var keyValueList []irs.KeyValue
	var i map[string]interface{}

	jsonBytes, errJson := json.Marshal(v)
	if errJson != nil {
		cblogger.Error("KeyValue 변환 실패")
		cblogger.Error(errJson)
		return nil, errJson
	}

	json.Unmarshal(jsonBytes, &i)

	for k, v := range i {
		cblogger.Debugf("K:[%s]====>", k)
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
			cblogger.Errorf("Key[%s]의 값은 변환 불가 - [%s]", k, errString)
			continue
		}
		keyValueList = append(keyValueList, irs.KeyValue{k, value})
	}
	cblogger.Debug("getKeyValueList : ", keyValueList)
	//keyValueList = append(keyValueList, irs.KeyValue{"test", typeToString([]float32{3.14, 1.53, 2.0000000000000})})

	return keyValueList, nil
}

// array에 주어진 string이 있는지 체크
func ContainString(s []string, str string) bool {
	for _, v := range s {
		cblogger.Info(v + " : " + str)
		cblogger.Info(v == str)
		if v == str {
			return true
		}
	}
	return false
}

/**
json 형태로 출력
*/
func printToJson(class interface{}) {
	e, err := json.Marshal(class)
	if err != nil {
		cblogger.Info(err)
	}
	cblogger.Info(string(e))
}
