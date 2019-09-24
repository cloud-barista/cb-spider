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
	Name string
	Id   string
	// @todo
}

type KeyPairInfo struct {
	Name string // AWS
	Id   string
	// @todo
	Fingerprint string // 추가 - AWS, OpenStack
	KeyMaterial string // 추가 - AWS(PEM파일-RSA PRIVATE KEY)
}

type KeyPairHandler interface {
	CreateKey(keyPairReqInfo KeyPairReqInfo) (KeyPairInfo, error)
	ListKey() ([]*KeyPairInfo, error)
	GetKey(keyPairID string) (KeyPairInfo, error) // AWS는 keyPairName
	DeleteKey(keyPairID string) (bool, error)     // AWS는 keyPairName
}
