// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by powerkim@etri.re.kr, 2019.06.

package resources

type KeyPairReqInfo struct {
        Name     string
}

type KeyPairInfo struct {
     Name        string
     Fingerprint string
     PublicKey   string
     PrivateKey  string
     VMUserID      string

     KeyValueList []KeyValue 
}

type KeyPairHandler interface {
	CreateKey(keyPairReqInfo KeyPairReqInfo) (KeyPairInfo, error)
	ListKey() ([]*KeyPairInfo, error)
	GetKey(keyName string) (KeyPairInfo, error) // AWS는 keyPairName
	DeleteKey(keyName string) (bool, error)     // AWS는 keyPairName
}
