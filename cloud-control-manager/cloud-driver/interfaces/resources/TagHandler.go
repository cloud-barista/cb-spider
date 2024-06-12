// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2024.06.

package resources

type TagInfo struct {
	ResType RSType // "image" | "vpc", "subnet", etc.,.
	ResIId  IID    // {NameId, SystemId}

	TagList      []KeyValue
	KeyValueList []KeyValue // reserved for optinal usage
}

type TagHandler interface {
	// A resource level (e.g., VPC, VM, etc.,.)
	AddTag(resTyp string, resIID IID, tag KeyValue) (TagInfo, error)
	ListTag(resTyp string, resIID IID) ([]*TagInfo, error)
	GetTag(resTyp string, resIID IID, key string) (TagInfo, error)
	RemoveTag(resType string, resIID IID, key string) (bool, error)

	// Find tags by tag key or value
	// resType: ALL | VPC, SUBNET, etc.,.
	// keyword: The keyword to search for in the tag key or value.
	// if you want to find all tags, set keyword to "" or "*".
	FindTag(resType RSType, keyword string) ([]*TagInfo, error)
}
