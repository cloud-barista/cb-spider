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

type ImageReqInfo struct {
	IId   IID 	// {NameId, SystemId}
	// @todo
}

type ImageInfo struct {
	IId   IID 	// {NameId, SystemId}
	GuestOS string // Windows7, Ubuntu etc.
	Status string  // available, unavailable

	KeyValueList []KeyValue 
}

type ImageHandler interface {
	CreateImage(imageReqInfo ImageReqInfo) (ImageInfo, error)
	ListImage() ([]*ImageInfo, error)
	GetImage(imageIID IID) (ImageInfo, error)
	CheckWindowsImage(imageIID IID) (bool, error)
	DeleteImage(imageIID IID) (bool, error)
}
