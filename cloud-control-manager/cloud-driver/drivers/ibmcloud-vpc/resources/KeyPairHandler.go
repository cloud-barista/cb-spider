package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	keypair "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"io/ioutil"
	"net/url"
	"os"
)

type IbmKeyPairHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VpcService     *vpcv1.VpcV1
	Ctx            context.Context
}

func (keyPairHandler *IbmKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	hiscallInfo := GetCallLogScheme(keyPairHandler.Region, call.VMKEYPAIR, keyPairReqInfo.IId.NameId, "CreateKey()")

	//IID확인
	err := checkValidKeyReqInfo(keyPairReqInfo)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	//존재여부 확인
	exist, err := keyPairHandler.existKey(keyPairReqInfo.IId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}

	if exist {
		err = errors.New(fmt.Sprintf("already exist VPC %s", keyPairReqInfo.IId.NameId))
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	start := call.Start()
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	if err := checkKeyPairFolder(keyPairPath); err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	savePrivateFileTo := keyPairPath + hashString + "--" + keyPairReqInfo.IId.NameId
	savePublicFileTo := keyPairPath + hashString + "--" + keyPairReqInfo.IId.NameId + ".pub"

	if _, err := os.Stat(savePrivateFileTo); err == nil {
		errMsg := fmt.Sprintf("KeyPair with name %s already exist", keyPairReqInfo.IId.NameId)
		createErr := errors.New(errMsg)

		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}
	privateKey, publicKey, err := keypair.GenKeyPair()

	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	err = keypair.SaveKey(privateKey, savePrivateFileTo)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	err = keypair.SaveKey(publicKey, savePublicFileTo)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}

	options := &vpcv1.CreateKeyOptions{}
	options.SetName(keyPairReqInfo.IId.NameId)
	options.SetPublicKey(string(publicKey))
	key, _, err := keyPairHandler.VpcService.CreateKeyWithContext(keyPairHandler.Ctx, options)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}

	createKeypairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   *key.Name,
			SystemId: *key.ID,
		},
		Fingerprint: *key.Fingerprint,
		PublicKey:   *key.PublicKey,
		PrivateKey:  string(privateKey),
	}
	LoggingInfo(hiscallInfo, start)
	return createKeypairInfo, nil
}

func (keyPairHandler *IbmKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	hiscallInfo := GetCallLogScheme(keyPairHandler.Region, call.VMKEYPAIR, "VMKEYPAIR", "ListKey()")
	start := call.Start()
	listKeysOptions := &vpcv1.ListKeysOptions{}
	keys, _, err := keyPairHandler.VpcService.ListKeysWithContext(keyPairHandler.Ctx, listKeysOptions)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	var ListKeys []*irs.KeyPairInfo
	for {
		for _, key := range keys.Keys {
			keyInfo, err := setKeyInfo(key, keyPairHandler.CredentialInfo)
			if err != nil {
				LoggingError(hiscallInfo, err)
				continue
			}
			ListKeys = append(ListKeys, &keyInfo)
		}
		nextstr, _ := getKeyNextHref(keys.Next)
		if nextstr != "" {
			listKeysOptions := &vpcv1.ListKeysOptions{
				Start: core.StringPtr(nextstr),
			}
			keys, _, err = keyPairHandler.VpcService.ListKeysWithContext(keyPairHandler.Ctx, listKeysOptions)
			if err != nil {
				LoggingError(hiscallInfo, err)
				return nil, err
				//break
			}
		} else {
			break
		}
	}
	LoggingInfo(hiscallInfo, start)
	return ListKeys, nil
}

func (keyPairHandler *IbmKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	hiscallInfo := GetCallLogScheme(keyPairHandler.Region, call.VMKEYPAIR, keyIID.NameId, "GetKey()")
	start := call.Start()
	key, err := getRawKey(keyIID, keyPairHandler.VpcService, keyPairHandler.Ctx)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	keyInfo, err := setKeyInfo(key, keyPairHandler.CredentialInfo)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)
	return keyInfo, nil
}

func (keyPairHandler *IbmKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(keyPairHandler.Region, call.VMKEYPAIR, keyIID.NameId, "DeleteKey()")
	start := call.Start()
	key, err := getRawKey(keyIID, keyPairHandler.VpcService, keyPairHandler.Ctx)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	deleteKeyOptions := &vpcv1.DeleteKeyOptions{}
	deleteKeyOptions.SetID(*key.ID)
	response, err := keyPairHandler.VpcService.DeleteKeyWithContext(keyPairHandler.Ctx, deleteKeyOptions)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	if response.StatusCode == 204 {
		keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
		err = checkKeyPairFolder(keyPairPath)
		// 폴더 없음 err != nil => local delete 필요 없음
		if err != nil {
			LoggingInfo(hiscallInfo, start)
			return true, nil
		}
		hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}

		privateKeyPath := keyPairPath + hashString + "--" + *key.Name
		publicKeyPath := keyPairPath + hashString + "--" + *key.Name + ".pub"

		err = os.Remove(privateKeyPath)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
		err = os.Remove(publicKeyPath)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
		LoggingInfo(hiscallInfo, start)
		return true, nil
	} else {
		LoggingError(hiscallInfo, err)
		return false, err
	}
}

func (keyPairHandler *IbmKeyPairHandler) existKey(keyIID irs.IID) (bool, error) {
	if keyIID.NameId == "" {
		return false, errors.New("inValid Name")
	} else {
		listKeysOptions := &vpcv1.ListKeysOptions{}
		keys, _, err := keyPairHandler.VpcService.ListKeysWithContext(keyPairHandler.Ctx, listKeysOptions)
		if err != nil {
			return false, err
		}
		for {
			for _, key := range keys.Keys {
				if *key.Name == keyIID.NameId {
					return true, nil
				}
			}
			nextstr, _ := getKeyNextHref(keys.Next)
			if nextstr != "" {
				listKeysOptions := &vpcv1.ListKeysOptions{
					Start: core.StringPtr(nextstr),
				}
				keys, _, err = keyPairHandler.VpcService.ListKeysWithContext(keyPairHandler.Ctx, listKeysOptions)
				if err != nil {
					return false, errors.New("failed Get KeyList")
				}
			} else {
				break
			}
		}
		return false, nil
	}
}

func setKeyInfo(key vpcv1.Key, credentialInfo idrv.CredentialInfo) (irs.KeyPairInfo, error) {
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	if err := checkKeyPairFolder(keyPairPath); err != nil {
		// LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	hashString, err := CreateHashString(credentialInfo)
	if err != nil {
		//LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	privateKeyPath := keyPairPath + hashString + "--" + *key.Name

	privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)

	if err != nil {
		return irs.KeyPairInfo{}, err
	}
	keypairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   *key.Name,
			SystemId: *key.ID,
		},
		Fingerprint: *key.Fingerprint,
		PublicKey:   *key.PublicKey,
		PrivateKey:  string(privateKeyBytes),
	}
	return keypairInfo, nil
}

func checkValidKeyReqInfo(keyReqInfo irs.KeyPairReqInfo) error {
	if keyReqInfo.IId.NameId == "" {
		return errors.New("invalid VPCReqInfo NameId")
	}
	return nil
}

func getRawKey(keyIID irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) (vpcv1.Key, error) {
	if keyIID.SystemId == "" {
		if keyIID.NameId == "" {
			err := errors.New("invalid IId")
			return vpcv1.Key{}, err
		}
		listKeysOptions := &vpcv1.ListKeysOptions{}
		keys, _, err := vpcService.ListKeysWithContext(ctx, listKeysOptions)
		if err != nil {
			return vpcv1.Key{}, err
		}
		for {
			for _, key := range keys.Keys {
				if *key.Name == keyIID.NameId {
					return key, nil
				}
			}
			nextstr, _ := getKeyNextHref(keys.Next)
			if nextstr != "" {
				listKeysOptions := &vpcv1.ListKeysOptions{
					Start: core.StringPtr(nextstr),
				}
				keys, _, err = vpcService.ListKeysWithContext(ctx, listKeysOptions)
				if err != nil {
					// LoggingError(hiscallInfo, err)
					return vpcv1.Key{}, err
					//break
				}
			} else {
				break
			}
		}
		err = errors.New(fmt.Sprintf("not found Key %s", keyIID.NameId))
		return vpcv1.Key{}, err
	} else {
		options := &vpcv1.GetKeyOptions{
			ID: core.StringPtr(keyIID.SystemId),
		}
		key, _, err := vpcService.GetKeyWithContext(ctx, options)
		if err != nil {
			return vpcv1.Key{}, err
		}
		return *key, nil
	}
}

func getKeyNextHref(next *vpcv1.KeyCollectionNext) (string, error) {
	if next != nil {
		href := *next.Href
		u, err := url.Parse(href)
		if err != nil {
			return "", err
		}
		paramMap, _ := url.ParseQuery(u.RawQuery)
		if paramMap != nil {
			safe := paramMap["start"]
			if safe != nil && len(safe) > 0 {
				return safe[0], nil
			}
		}
	}
	return "", errors.New("NOT NEXT")
}

func checkKeyPairFolder(folderPath string) error {
	// Check KeyPair Folder Exists
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		if err := os.MkdirAll(folderPath, 0700); err != nil {
			return err
		}
	}
	return nil
}
