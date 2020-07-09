// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2020.04.
// by CB-Spider Team, 2019.06.

package resources

type KeyPairReqInfo struct {
	IId   IID       // {NameId, SystemId}
}

type KeyPairInfo struct {
	IId   IID       // {NameId, SystemId}
	Fingerprint string
	PublicKey   string
	PrivateKey  string
	VMUserID      string

	KeyValueList []KeyValue 
}

type KeyPairHandler interface {
	CreateKey(keyPairReqInfo KeyPairReqInfo) (KeyPairInfo, error)
	ListKey() ([]*KeyPairInfo, error)
	GetKey(keyIID IID) (KeyPairInfo, error) 
	DeleteKey(keyIID IID) (bool, error)     
}
