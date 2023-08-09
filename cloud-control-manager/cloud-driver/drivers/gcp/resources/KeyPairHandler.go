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

	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	//중복 체크
	keyValue, err := keypair.GetKey(CBKeyPairProvider, hashString, keyPairName)
	cblogger.Debug(keyValue)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist!") {

		} else {
			cblogger.Error(err)
			return irs.KeyPairInfo{}, err
		}
	} else {
		errMsg := fmt.Sprintf("KeyPair with name %s already exist", keyPairName)
		createErr := errors.New(errMsg)
		cblogger.Error(err)
		return irs.KeyPairInfo{}, createErr
	}

	/*
		//20211209 공통 모듈 기반으로 변경(공개키는 저장되지 않음)
		privateKeyBytes, publicKeyBytes, err := keypair.GenKeyPair()
		publicKeyString := string(publicKeyBytes)
		publicKeyString = strings.TrimSpace(publicKeyString) + " " + "cb-user"
		// projectId 대신에 cb-user 고정
		cblog.Info("publicKeyString : ", publicKeyString)
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
	*/

	privateKeyBytes, publicKeyBytes, err := keypair.GenKeyPair()
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	err = keypair.AddKey(CBKeyPairProvider, hashString, keyPairName, string(privateKeyBytes))
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	publicKeyString := string(publicKeyBytes)
	publicKeyString = strings.TrimSpace(publicKeyString) + " " + "cb-user"

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
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error("Fail CreateHashString")
		cblogger.Error(err)
		return nil, err
	}

	var keyPairInfoList []*irs.KeyPairInfo
	keyValueList, err := keypair.ListKey(CBKeyPairProvider, hashString)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	for _, keyValue := range keyValueList {
		keypairInfo, err := keyPairHandler.GetKey(irs.IID{SystemId: keyValue.Key})
		if err != nil {
			cblogger.Error("Fail GetKey")
			cblogger.Error(err)
			return nil, err
		}
		keyPairInfoList = append(keyPairInfoList, &keypairInfo)
	}

	return keyPairInfoList, nil
}

func (keyPairHandler *GCPKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	cblogger.Infof("keyPairName : [%s]", keyIID.SystemId)
	keyPairName := strings.ToLower(keyIID.SystemId)
	cblogger.Infof("keyPairName 소문자로 치환 : [%s]", keyPairName)

	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error("Fail CreateHashString")
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	keyValue, err := keypair.GetKey(CBKeyPairProvider, hashString, keyPairName)
	cblogger.Debug(keyValue)
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	keypairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   keyPairName,
			SystemId: keyPairName,
		},
		PublicKey:  "",
		PrivateKey: keyValue.Value,
	}
	return keypairInfo, nil
}

func (keyPairHandler *GCPKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	cblogger.Infof("keyPairName : [%s]", keyIID.SystemId)
	keyPairName := strings.ToLower(keyIID.SystemId)
	cblogger.Infof("keyPairName 소문자로 치환 : [%s]", keyPairName)

	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error("Fail CreateHashString")
		cblogger.Error(err)
		return false, err
	}

	//키 페어 존재 여부 체크
	if _, err := keyPairHandler.GetKey(keyIID); err != nil {
		cblogger.Error(err)
		return false, errors.New("Not Found : [" + keyIID.SystemId + "] KeyPair Not Found.")
	}

	err = keypair.DelKey(CBKeyPairProvider, hashString, keyPairName)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	return true, nil
}
