package resources

import (
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/keypairs"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	KeyPair = "KEYPAIR"
)

type OpenStackKeyPairHandler struct {
	Client *gophercloud.ServiceClient
}

func setterKeypair(keypair keypairs.KeyPair) *irs.KeyPairInfo {
	keypairInfo := &irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   keypair.Name,
			SystemId: keypair.Name,
		},
		Fingerprint: keypair.Fingerprint,
		PublicKey:   keypair.PublicKey,
		PrivateKey:  keypair.PrivateKey,
	}
	return keypairInfo
}

func (keyPairHandler *OpenStackKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(keyPairHandler.Client.IdentityEndpoint, call.VMKEYPAIR, keyPairReqInfo.IId.NameId, "CreateKey()")

	create0pts := keypairs.CreateOpts{
		Name:      keyPairReqInfo.IId.NameId,
		PublicKey: "",
	}

	start := call.Start()
	keyPair, err := keypairs.Create(keyPairHandler.Client, create0pts).Extract()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	// 생성된 KeyPair 정보 리턴
	keyPairInfo := setterKeypair(*keyPair)
	return *keyPairInfo, nil
}

func (keyPairHandler *OpenStackKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(keyPairHandler.Client.IdentityEndpoint, call.VMKEYPAIR, KeyPair, "ListKey()")

	// 키페어 목록 조회
	start := call.Start()
	pager, err := keypairs.List(keyPairHandler.Client).AllPages()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	keypair, err := keypairs.ExtractKeyPairs(pager)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	// 키페어 목록 정보 매핑
	keyPairList := make([]*irs.KeyPairInfo, len(keypair))
	for i, k := range keypair {
		keyPairList[i] = setterKeypair(k)
	}
	return keyPairList, nil
}

func (keyPairHandler *OpenStackKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(keyPairHandler.Client.IdentityEndpoint, call.VMKEYPAIR, keyIID.NameId, "GetKey()")

	start := call.Start()
	keyPair, err := keypairs.Get(keyPairHandler.Client, keyIID.NameId).Extract()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	keyPairInfo := setterKeypair(*keyPair)
	return *keyPairInfo, nil
}

func (keyPairHandler *OpenStackKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(keyPairHandler.Client.IdentityEndpoint, call.VMKEYPAIR, keyIID.NameId, "DeleteKey()")

	start := call.Start()
	err := keypairs.Delete(keyPairHandler.Client, keyIID.NameId).ExtractErr()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
