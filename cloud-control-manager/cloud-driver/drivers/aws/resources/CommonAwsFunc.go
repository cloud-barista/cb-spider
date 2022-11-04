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

const KEY_VALUE_CONVERT_DEBUG_INFO bool = false     // JSON 및 Key Value객체 Convert시(ConvertToString, ConvertKeyValueList) Debug 로그 정보 출력 여부(Debug 모드로 개발 할 때 불필요한 정보를 줄이기 위해 추가)
const CBDefaultVNetName string = "CB-VNet"          // CB Default Virtual Network Name
const CBDefaultSubnetName string = "CB-VNet-Subnet" // CB Default Subnet Name
const CBDefaultCidrBlock string = "192.168.0.0/16"  // CB Default CidrBlock
//const CBKeyPairPath string = "/meta_db/.ssh-aws/" // 이슈 #480에 의한 로컬 키 관리 제거

//const CBCloudInitFilePath string = "/cloud-driver-libs/.cloud-init-aws/cloud-init"
const CBCloudInitWindowsFilePath string = "/cloud-driver-libs/.cloud-init-aws/cloud-init-windows" //Windows용 사용자 비번 설정을 위한 탬플릿
const CBCloudInitFilePath string = "/cloud-driver-libs/.cloud-init-common/cloud-init"
const CBDefaultVmUserName string = "cb-user" // default VM User Name

const CUSTOM_ERR_CODE_TOOMANY string = "600"            //awserr.New("600", "n개 이상의 xxxx 정보가 존재합니다.", nil)
const CUSTOM_ERR_CODE_BAD_REQUEST string = "400"        //awserr.New("400", "요청 정보가 잘 못 되었습니다.", nil)
const CUSTOM_ERR_CODE_NOTFOUND string = "404"           //awserr.New("404", "XXX 정보가 존재하지 않습니다.", nil)
const CUSTOM_ERR_CODE_METHOD_NOT_ALLOWED string = "405" //awserr.New("405", "지원되지 않는 기능입니다.", nil)
const CUSTOM_ERR_CODE_NOT_IMPLEMENTED string = "501"    //awserr.New("501", "기능이 구현되어 있지 않습니다.", nil)

type AwsCBNetworkInfo struct {
	VpcName   string
	VpcId     string
	CidrBlock string
	IsDefault bool
	State     string

	SubnetName string
	SubnetId   string
}

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

/*
//이 함수는 VPC & Subnet이 존재하는 곳에서만 사용됨.
//VPC & Subnet이 존재하는 경우 정보를 리턴하고 없는 경우 Default VPC & Subnet을 생성 후 정보를 리턴 함.
func (VPCHandler *AwsVPCHandler) GetAutoCBNetworkInfo() (AwsCBNetworkInfo, error) {
	return AwsCBNetworkInfo{}, errors.New("인터페이스 변경해야 함!!!!")
		var awsCBNetworkInfo AwsCBNetworkInfo

		subNetId := VPCHandler.GetMcloudBaristaDefaultSubnetId()
		if subNetId == "" {
			//내부에서 VPC를 자동으로 생성후 Subnet을 생성함.
			_, err := VPCHandler.CreateVNetwork(irs.VNetworkReqInfo{})
			if err != nil {
				cblogger.Error("Default VNetwork(VPC & Subnet) 자동 생성 실패")
				cblogger.Error(err)
				return AwsCBNetworkInfo{}, err
			}
		}

		//VPC & Subnet을 생성했으므로 예외처리 없이 조회만 처리함.
		awsVpcInfo, _ := VPCHandler.GetVpc(GetCBDefaultVNetName())
		spew.Dump(awsVpcInfo)
		awsCBNetworkInfo.VpcId = awsVpcInfo.Id
		awsCBNetworkInfo.VpcName = awsVpcInfo.Name

		awsSubnetInfo, _ := VPCHandler.GetVNetwork(irs.IID{})
		spew.Dump(awsSubnetInfo)
		//awsCBNetworkInfo.SubnetId = awsSubnetInfo.Id
		//awsCBNetworkInfo.SubnetName = awsSubnetInfo.Name
		awsCBNetworkInfo.SubnetId = awsSubnetInfo.IId.SystemId
		awsCBNetworkInfo.SubnetName = awsSubnetInfo.IId.NameId

		spew.Dump(awsCBNetworkInfo)

		return awsCBNetworkInfo, nil
}
*/

/*

func (VPCHandler *AwsVPCHandler) GetMcloudBaristaDefaultVpcId() string {
	return ""
		awsVpcInfo, err := VPCHandler.GetVpc(GetCBDefaultVNetName())
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				default:
					cblogger.Error(aerr.Error())
				}
			} else {
				cblogger.Error(err.Error())
			}
			return ""
		}

		//기존 정보가 존재하면...
		if awsVpcInfo.Id != "" {
			return awsVpcInfo.Id
		} else {
			return ""
		}
}
*/

/*

//@TODO : awsSubnetInfo.IId.SystemId를 리턴해야 하는지 NameId를 리턴해야 하는지 체크해야 함. -> 생성된 정보가 있는지만 체크 하므로 상관 없음.
func (VPCHandler *AwsVPCHandler) GetMcloudBaristaDefaultSubnetId() string {
	awsSubnetInfo, err := VPCHandler.GetVNetwork(irs.IID{})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			cblogger.Error(err.Error())
		}
		return ""
	}

	//기존 정보가 존재하면...
	//if awsSubnetInfo.Id != "" {
	//	return awsSubnetInfo.Id
	if awsSubnetInfo.IId.SystemId != "" {
		return awsSubnetInfo.IId.SystemId
	} else {
		return ""
	}
}

//@TODO : ListVNetwork()에서 호출되는 경우도 있기 때문에 필요하면 VPC조회와 생성을 별도의 Func으로 분리해야함.(일단은 큰 문제는 없어서 놔둠)
//CB Default Virtual Network가 존재하지 않으면 생성하며, 존재하는 경우 Vpc ID를 리턴 함.
func (VPCHandler *AwsVPCHandler) FindOrCreateMcloudBaristaDefaultVPC(vNetworkReqInfo irs.VNetworkReqInfo) (string, error) {
	cblogger.Info(vNetworkReqInfo)

	awsVpcInfo, err := VPCHandler.GetVpc(GetCBDefaultVNetName())
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return "", err
	}

	//기존 정보가 존재하면...
	if awsVpcInfo.Id != "" {
		return awsVpcInfo.Id, nil
	} else {
		//@TODO : Subnet과 VPC모두 CSP별 고정된 값으로 드라이버가 내부적으로 자동으로 생성하도록 CB규약이 바뀌어서 서브넷 정보 기반의 로직은 모두 잠시 죽여 놓음 - 리스트 요청시에도 내부적으로 자동 생성하도록 변경 중
		/ *
			cblogger.Infof("기본 VPC[%s]가 없어서 Subnet 요청 정보를 기반으로 /16 범위의 VPC를 생성합니다.", GetCBDefaultVNetName())
			cblogger.Info("Subnet CIDR 요청 정보 : ", vNetworkReqInfo.CidrBlock)
			if vNetworkReqInfo.CidrBlock == "" {
				//VPC가 없는 최초 상태에서 List()에서 호출되었을 수 있기 때문에 에러 처리는 하지 않고 nil을 전달함.
				cblogger.Infof("요청 정보에 CIDR 정보가 없어서 Default VPC[%s]를 생성하지 않음", GetCBDefaultVNetName())
				return "", nil
			}

			reqCidr := strings.Split(vNetworkReqInfo.CidrBlock, ".")
			//cblogger.Info("CIDR 추출 정보 : ", reqCidr[0])
			VpcCidrBlock := reqCidr[0] + "." + reqCidr[1] + ".0.0/16"
			cblogger.Info("신규 VPC에 사용할 CIDR 정보 : ", VpcCidrBlock)
		* /

		cblogger.Infof("기본 VPC[%s]가 없어서 CIDR[%s] 범위의 VPC를 자동으로 생성합니다.", GetCBDefaultVNetName(), GetCBDefaultCidrBlock())
		awsVpcReqInfo := AwsVpcReqInfo{
			Name: GetCBDefaultVNetName(),
			//CidrBlock: VpcCidrBlock,
			CidrBlock: GetCBDefaultCidrBlock(),
		}

		result, errVpc := VPCHandler.CreateVpc(awsVpcReqInfo)
		if errVpc != nil {
			cblogger.Error(errVpc)
			return "", errVpc
		}
		cblogger.Infof("CB Default VPC[%s] 생성 완료 - CIDR : [%s]", GetCBDefaultVNetName(), result.CidrBlock)
		cblogger.Info(result)
		spew.Dump(result)

		return result.Id, nil
	}
}

//자동으로 생성된 VPC & Subnet을 삭제해도 되는가?
//명시적으로 Subnet 삭제의 호출이 없기 때문에 시큐리티 그룹이나 vNic이 삭제되는 시점에 호출됨.
func (VPCHandler *AwsVPCHandler) IsAvailableAutoCBNet() bool {
	return false
}
*/

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

// array에 주어진 string이 있는지 체크
func ContainString(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
