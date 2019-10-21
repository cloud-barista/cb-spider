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

//package image

type ImageReqInfo struct {
	//2차 인터페이스
	Name string
	Id   string
	// @todo
}

type ImageInfo struct {
	//2차 인터페이스
	Id      string
	Name    string
	GuestOS string // Windows7, Ubuntu etc.
	Status  string // available, unavailable

	KeyValueList []KeyValue
}

type ImageHandler interface {
	CreateImage(imageReqInfo ImageReqInfo) (ImageInfo, error)
	ListImage() ([]*ImageInfo, error)
	GetImage(imageID string) (ImageInfo, error)
	DeleteImage(imageID string) (bool, error)
}
