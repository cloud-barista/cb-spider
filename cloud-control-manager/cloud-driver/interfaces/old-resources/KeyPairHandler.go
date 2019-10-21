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
	//2차 인터페이스
	Name string

	// @todo - 삭제예정(1차 인터페이스 잔여 필드)
	Id string
}

type KeyPairInfo struct {
	//2차 인터페이스
	Name        string
	Fingerprint string // 추가 - AWS, OpenStack
	PublicKey   string
	PrivateKey  string
	VMUserID    string

	KeyValueList []KeyValue

	// @todo - 삭제예정(1차 인터페이스 잔여 필드)
	Id          string
	KeyMaterial string // 추가 - AWS(PEM파일-RSA PRIVATE KEY)
}

type KeyPairHandler interface {
	CreateKey(keyPairReqInfo KeyPairReqInfo) (KeyPairInfo, error)
	ListKey() ([]*KeyPairInfo, error)
	GetKey(keyName string) (KeyPairInfo, error) // AWS는 keyPairName
	DeleteKey(keyName string) (bool, error)     // AWS는 keyPairName
}
