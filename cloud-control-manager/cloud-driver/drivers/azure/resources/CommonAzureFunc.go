package resources

import (
	"crypto/md5"
	"fmt"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const (
	CBResourceGroupName  = "CB-GROUP"
	CBVirutalNetworkName = "CB-VNet"
	CBVnetDefaultCidr    = "130.0.0.0/16"
	CBVMUser             = "cb-user"
	CBKeyPairPath        = "/cloud-control-manager/cloud-driver/driver-libs/.ssh-azure/"
)

// 서브넷 CIDR 생성 (CIDR C class 기준 생성)
func CreateSubnetCIDR(subnetList []*irs.VNetworkInfo) (*string, error) {

	// CIDR C class 최대값 찾기
	maxClassNum := 0
	for _, subnet := range subnetList {
		addressArr := strings.Split(subnet.AddressPrefix, ".")
		if curClassNum, err := strconv.Atoi(addressArr[2]); err != nil {
			return nil, err
		} else {
			if curClassNum > maxClassNum {
				maxClassNum = curClassNum
			}
		}
	}

	if len(subnetList) == 0 {
		maxClassNum = 0
	} else {
		maxClassNum = maxClassNum + 1
	}

	// 서브넷 CIDR 할당
	vNetIP := strings.Split(CBVnetDefaultCidr, "/")
	vNetIPClass := strings.Split(vNetIP[0], ".")
	subnetCIDR := fmt.Sprintf("%s.%s.%d.0/24", vNetIPClass[0], vNetIPClass[1], maxClassNum)
	return &subnetCIDR, nil
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
