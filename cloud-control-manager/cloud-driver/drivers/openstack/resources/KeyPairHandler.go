package resources

import (
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/rackspace/gophercloud/pagination"
)

type OpenStackKeyPairHandler struct {
	Client *gophercloud.ServiceClient
}

func setterKeypair(keypair keypairs.KeyPair) *irs.KeyPairInfo {
	keypairInfo := &irs.KeyPairInfo{
		Name:        keypair.Name,
		Fingerprint: keypair.Fingerprint,
		PublicKey:   keypair.PublicKey,
		PrivateKey:  keypair.PrivateKey,
		VMUserID:    keypair.UserID,
	}
	return keypairInfo
}

func (keyPairHandler *OpenStackKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	create0pts := keypairs.CreateOpts{
		Name:      keyPairReqInfo.Name,
		PublicKey: "",
	}
	keyPair, err := keypairs.Create(keyPairHandler.Client, create0pts).Extract()
	if err != nil {
		return irs.KeyPairInfo{}, err
	}

	// 생성된 KeyPair 정보 리턴
	keyPairInfo := setterKeypair(*keyPair)
	return *keyPairInfo, nil
}

func (keyPairHandler *OpenStackKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	var keyPairList []*irs.KeyPairInfo

	pager := keypairs.List(keyPairHandler.Client)
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		// Get KeyPair
		list, err := keypairs.ExtractKeyPairs(page)
		if err != nil {
			return false, err
		}
		// Add to List
		for _, k := range list {
			keyPairInfo := setterKeypair(k)
			keyPairList = append(keyPairList, keyPairInfo)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return keyPairList, nil
}

func (keyPairHandler *OpenStackKeyPairHandler) GetKey(keyName string) (irs.KeyPairInfo, error) {
	keyPair, err := keypairs.Get(keyPairHandler.Client, keyName).Extract()
	if err != nil {
		return irs.KeyPairInfo{}, err
	}

	keyPairInfo := setterKeypair(*keyPair)
	return *keyPairInfo, nil
}

func (keyPairHandler *OpenStackKeyPairHandler) DeleteKey(keyName string) (bool, error) {
	err := keypairs.Delete(keyPairHandler.Client, keyName).ExtractErr()
	if err != nil {
		return false, err
	}
	return true, nil
}
