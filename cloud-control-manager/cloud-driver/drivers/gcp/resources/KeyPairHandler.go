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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	keypair "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type GCPKeyPairHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
}

func (keyPairHandler *GCPKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	keyPairName := strings.ToLower(keyPairReqInfo.IId.NameId)
	cblogger.Infof("keyPairName [%s] --> [%s]", keyPairReqInfo.IId.NameId, keyPairName)

	//projectId := keyPairHandler.CredentialInfo.ProjectID
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	//키페어 생성 시 폴더가 존재하지 않으면 생성 함.
	_, errChkDir := os.Stat(keyPairPath)
	if os.IsNotExist(errChkDir) {
		cblogger.Errorf("[%s] Path가 존재하지 않아서 생성합니다.", keyPairPath)

		errDir := os.MkdirAll(keyPairPath, 0755)
		//errDir := os.MkdirAll(keyPairPath, os.ModePerm) // os.ModePerm : 0777	//os.ModeDir
		if errDir != nil {
			//log.Fatal(err)
			cblogger.Errorf("[%s] Path가 생성 실패", keyPairPath)
			cblogger.Error(errDir)
			return irs.KeyPairInfo{}, errDir
		}
	}

	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	savePrivateFileTo := keyPairPath + hashString + "--" + keyPairName
	savePublicFileTo := keyPairPath + hashString + "--" + keyPairName + ".pub"

	// Check KeyPair Exists
	if _, err := os.Stat(savePrivateFileTo); err == nil {
		errMsg := fmt.Sprintf("KeyPair with name %s already exist", keyPairName)
		createErr := errors.New(errMsg)
		cblogger.Error(err)
		return irs.KeyPairInfo{}, createErr
	}

	privateKeyBytes, publicKeyBytes, err := keypair.GenKeyPair()
	publicKeyString := string(publicKeyBytes)
	// projectId 대신에 cb-user 고정
	publicKeyString = strings.TrimSpace(publicKeyString) + " " + "cb-user"
	fmt.Println("publicKeyString : ", publicKeyString)
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	// 파일에 private Key를 쓴다
	err = keypair.SaveKey(privateKeyBytes, savePrivateFileTo)
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	// 파일에 public Key를 쓴다
	err = keypair.SaveKey([]byte(publicKeyString), savePublicFileTo)
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	keyPairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   keyPairName,
			SystemId: keyPairName,
		},
		PublicKey:  publicKeyString,
		PrivateKey: string(privateKeyBytes),
	}
	return keyPairInfo, nil
}

func (keyPairHandler *GCPKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error("Fail CreateHashString")
		cblogger.Error(err)
		return nil, err
	}

	var keyPairInfoList []*irs.KeyPairInfo

	files, err := ioutil.ReadDir(keyPairPath)
	if err != nil {
		//cblogger.Error("Fail ReadDir(keyPairPath)")
		//cblogger.Error(err)
		//return nil, err

		//키페어 폴더가 없는 경우 생성된 키가 없는 것으로 변경
		return nil, nil
	}

	for _, f := range files {
		if strings.Contains(f.Name(), ".pub") {
			continue
		}
		if strings.Contains(f.Name(), hashString) {
			fileNameArr := strings.Split(f.Name(), "--")
			keypairInfo, err := keyPairHandler.GetKey(irs.IID{SystemId: fileNameArr[1]})
			if err != nil {
				cblogger.Error("Fail GetKey")
				cblogger.Error(err)
				return nil, err
			}
			keyPairInfoList = append(keyPairInfoList, &keypairInfo)
		}
	}

	return keyPairInfoList, nil
}

func (keyPairHandler *GCPKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	cblogger.Infof("keyPairName : [%s]", keyIID.SystemId)
	keyPairName := strings.ToLower(keyIID.SystemId)
	cblogger.Infof("keyPairName 소문자로 치환 : [%s]", keyPairName)

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error("Fail CreateHashString")
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	privateKeyPath := keyPairPath + hashString + "--" + keyPairName
	publicKeyPath := keyPairPath + hashString + "--" + keyPairName + ".pub"

	//키 페어 존재 여부 체크
	if _, err := os.Stat(privateKeyPath); err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, errors.New("Not Found : [" + keyIID.SystemId + "] KeyPair Not Found.")
	}

	// Private Key, Public Key 파일 정보 가져오기
	privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}
	publicKeyBytes, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	keypairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   keyPairName,
			SystemId: keyPairName,
		},
		PublicKey:  string(publicKeyBytes),
		PrivateKey: string(privateKeyBytes),
	}
	return keypairInfo, nil
}

func (keyPairHandler *GCPKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	cblogger.Infof("keyPairName : [%s]", keyIID.SystemId)
	keyPairName := strings.ToLower(keyIID.SystemId)
	cblogger.Infof("keyPairName 소문자로 치환 : [%s]", keyPairName)

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error("Fail CreateHashString")
		cblogger.Error(err)
		return false, err
	}

	privateKeyPath := keyPairPath + hashString + "--" + keyPairName
	publicKeyPath := keyPairPath + hashString + "--" + keyPairName + ".pub"

	//키 페어 존재 여부 체크
	if _, err := os.Stat(privateKeyPath); err != nil {
		cblogger.Error(err)
		return false, errors.New("Not Found : [" + keyIID.SystemId + "] KeyPair Not Found.")
	}

	// Private Key, Public Key 삭제
	err = os.Remove(privateKeyPath)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}
	err = os.Remove(publicKeyPath)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	return true, nil
}
