package resources

import (
	"errors"
	"strings"

	keypair "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type OracleKeyPairHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
}

func (handler *OracleKeyPairHandler) CreateKey(req irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	if req.IId.NameId == "" {
		return irs.KeyPairInfo{}, errors.New("invalid keypair name")
	}
	privateKey, publicKey, err := keypair.GenKeyPair()
	if err != nil {
		return irs.KeyPairInfo{}, err
	}
	hash, err := handler.keyHash()
	if err != nil {
		return irs.KeyPairInfo{}, err
	}
	if err := keypair.AddKey(oracleProviderName, hash, req.IId.NameId, string(privateKey)); err != nil {
		return irs.KeyPairInfo{}, err
	}
	return irs.KeyPairInfo{IId: irs.IID{NameId: req.IId.NameId, SystemId: req.IId.NameId}, PrivateKey: string(privateKey), PublicKey: strings.TrimSpace(string(publicKey)), VMUserID: defaultVMUserID}, nil
}

func (handler *OracleKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	hash, err := handler.keyHash()
	if err != nil {
		return nil, err
	}
	keys, err := keypair.ListKey(oracleProviderName, hash)
	if err != nil {
		return nil, err
	}
	infos := make([]*irs.KeyPairInfo, 0, len(keys))
	for _, key := range keys {
		publicKey, _ := keypair.MakePublicKeyFromPrivateKey(key.Value)
		infos = append(infos, &irs.KeyPairInfo{IId: irs.IID{NameId: key.Key, SystemId: key.Key}, PrivateKey: key.Value, PublicKey: publicKey, VMUserID: defaultVMUserID})
	}
	return infos, nil
}

func (handler *OracleKeyPairHandler) GetKey(iid irs.IID) (irs.KeyPairInfo, error) {
	hash, err := handler.keyHash()
	if err != nil {
		return irs.KeyPairInfo{}, err
	}
	key, err := keypair.GetKey(oracleProviderName, hash, iid.NameId)
	if err != nil {
		return irs.KeyPairInfo{}, err
	}
	publicKey, _ := keypair.MakePublicKeyFromPrivateKey(key.Value)
	return irs.KeyPairInfo{IId: irs.IID{NameId: key.Key, SystemId: key.Key}, PrivateKey: key.Value, PublicKey: publicKey, VMUserID: defaultVMUserID}, nil
}

func (handler *OracleKeyPairHandler) DeleteKey(iid irs.IID) (bool, error) {
	hash, err := handler.keyHash()
	if err != nil {
		return false, err
	}
	return true, keypair.DelKey(oracleProviderName, hash, iid.NameId)
}

func (handler *OracleKeyPairHandler) ListIID() ([]*irs.IID, error) {
	infos, err := handler.ListKey()
	if err != nil {
		return nil, err
	}
	iids := make([]*irs.IID, 0, len(infos))
	for _, info := range infos {
		iids = append(iids, &info.IId)
	}
	return iids, nil
}

func (handler *OracleKeyPairHandler) keyHash() (string, error) {
	return keypair.GenHash([]string{handler.CredentialInfo.TenantId, handler.CredentialInfo.ClientId, handler.CredentialInfo.ProjectID, handler.Region.Region})
}
