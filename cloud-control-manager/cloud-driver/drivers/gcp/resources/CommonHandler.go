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
	"fmt"
	"io"
	"io/ioutil"
	"os"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	CBVMUser = "cscservice"
	//CBKeyPairPath = "/cloud-control-manager/cloud-driver/driver-libs/.ssh-gcp/"
	// by powerkim, 2019.10.30
	CBKeyPairPath = "/cloud-driver-libs/.ssh-gcp/"
)

const CBDefaultVNetName string = "cb-vnet"   // CB Default Virtual Network Name
const CBDefaultSubnetName string = "cb-vnet" // CB Default Subnet Name

type GcpCBNetworkInfo struct {
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
