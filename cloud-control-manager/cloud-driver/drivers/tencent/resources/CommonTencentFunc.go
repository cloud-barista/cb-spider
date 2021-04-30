// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2019.06.

package resources

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const CBDefaultVNetName string = "CB-VNet"          // CB Default Virtual Network Name
const CBDefaultSubnetName string = "CB-VNet-Subnet" // CB Default Subnet Name
const CBDefaultCidrBlock string = "192.168.0.0/16"  // CB Default CidrBlock

type TencentCBNetworkInfo struct {
	VpcName   string
	VpcId     string
	CidrBlock string
	IsDefault bool
	State     string

	SubnetName string
	SubnetId   string
}

const CUSTOM_ERR_CODE_TOOMANY string = "600"  //awserr.New("600", "n개 이상의 xxxx 정보가 존재합니다.", nil)
const CUSTOM_ERR_CODE_NOTFOUND string = "404" //awserr.New("404", "XXX 정보가 존재하지 않습니다.", nil)

//VPC
func GetCBDefaultVNetName() string {
	return CBDefaultVNetName
}

//Subnet
func GetCBDefaultSubnetName() string {
	return CBDefaultSubnetName
}

func GetCBDefaultCidrBlock() string {
	return CBDefaultCidrBlock
}

//Name Tag 설정
func SetNameTag(Client *ec2.EC2, Id string, value string) bool {
	// Tag에 Name 설정
	cblogger.Infof("Name Tage 설정 - ResourceId : [%s]  Value : [%s] ", Id, value)
	_, errtag := Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{&Id},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(value),
			},
		},
	})
	if errtag != nil {
		cblogger.Error("Name Tag 설정 실패 : ")
		cblogger.Error(errtag)
		return false
	}

	return true
}

func JSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

//Cloud Object를 JSON String 타입으로 변환
func ConvertJsonStringNoEscape(v interface{}) (string, error) {
	//jsonBytes, errJson := json.Marshal(v)

	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	errJson := encoder.Encode(v)
	if errJson != nil {
		cblogger.Error("JSON 변환 실패")
		cblogger.Error(errJson)
		return "", errJson
	}

	//fmt.Println("After marshal", string(buffer.Bytes()))
	//spew.Dump(string(buffer.Bytes()))
	//spew.Dump("\"TEST")

	jsonString := string(buffer.Bytes())
	//jsonString = strings.Replace(jsonString, "\n", "", -1)
	jsonString = strings.Replace(jsonString, "\"", "", -1)

	return jsonString, nil
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
		cblogger.Debugf("Nil Value")
		return "", errors.New("Nil. Value")
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
	//spew.Dump(v)
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
			//cblogger.Errorf("Key[%s]의 값은 변환 불가 - [%s]", k, errString)
			cblogger.Debugf("Key[%s]의 값은 변환 불가 - [%s]", k, errString) //요구에 의해서 Error에서 Warn으로 낮춤
			continue
		}
		keyValueList = append(keyValueList, irs.KeyValue{k, value})

		/*
			_, ok := v.(string)
			if !ok {
				cblogger.Errorf("Key[%s]의 값은 변환 불가", k)
				continue
			}
			keyValueList = append(keyValueList, irs.KeyValue{k, v.(string)})
		*/
	}
	cblogger.Debug("getKeyValueList : ", keyValueList)
	//keyValueList = append(keyValueList, irs.KeyValue{"test", typeToString([]float32{3.14, 1.53, 2.0000000000000})})

	return keyValueList, nil
}
