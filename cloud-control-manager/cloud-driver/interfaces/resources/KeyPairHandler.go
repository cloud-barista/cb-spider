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
	IId IID // {NameId, SystemId}

	TagList []KeyValue
}

// KeyPairInfo represents information about a KeyPair.
type KeyPairInfo struct {
	IId         IID    `json:"IId" validate:"required"`                                                                              // {NameId, SystemId}
	Fingerprint string `json:"Fingerprint,omitempty" validate:"omitempty" example:"3b:16:bf:1b:13:4b:b3:58:97:dc:72:19:45:bb:2c:8f"` // Unique identifier for the public key
	PublicKey   string `json:"PublicKey,omitempty" validate:"omitempty" example:"ssh-rsa AAAAB3..."`                                 // Public part of the KeyPair
	PrivateKey  string `json:"PrivateKey,omitempty" validate:"omitempty" example:"-----BEGIN PRIVATE KEY-----..."`                   // Private part of the KeyPair
	VMUserID    string `json:"VMUserID,omitempty" validate:"omitempty" example:"cb-user"`                                            // cb-user or Administrator

	TagList      []KeyValue `json:"TagList,omitempty" validate:"omitempty"`      // List of tags associated with this KeyPair
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"` // Additional metadata as key-value pairs
}

type KeyPairHandler interface {
	CreateKey(keyPairReqInfo KeyPairReqInfo) (KeyPairInfo, error)
	ListKey() ([]*KeyPairInfo, error)
	GetKey(keyIID IID) (KeyPairInfo, error)
	DeleteKey(keyIID IID) (bool, error)
	ListIID() ([]*IID, error)
}
