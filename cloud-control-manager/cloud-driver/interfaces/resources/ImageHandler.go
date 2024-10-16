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
	IId IID // {NameId, SystemId}
	// @todo
}

// ImageInfo represents the information of an Image.
type ImageInfo struct {
	IId     IID    `json:"IId" validate:"required" description:"The ID of the image."`                                                            // {NameId, SystemId}
	GuestOS string `json:"GuestOS" validate:"required" example:"Ubuntu 18.04" description:"The operating system of the image."`                   // Windows7, Ubuntu etc.
	Status  string `json:"Status" validate:"required" example:"available" description:"The status of the image, e.g., available or unavailable."` // available, unavailable

	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty" description:"A list of key-value pairs associated with the image."`
}

type ImageHandler interface {
	CreateImage(imageReqInfo ImageReqInfo) (ImageInfo, error)
	ListImage() ([]*ImageInfo, error)
	GetImage(imageIID IID) (ImageInfo, error)
	CheckWindowsImage(imageIID IID) (bool, error)
	DeleteImage(imageIID IID) (bool, error)
}
