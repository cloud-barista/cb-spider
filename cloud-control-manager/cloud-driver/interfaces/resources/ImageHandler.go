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
	IId     IID    `json:"IId" validate:"required" description:"The ID of the image."`                                          // {NameId, SystemId} // Deprecated
	GuestOS string `json:"GuestOS" validate:"required" example:"Ubuntu 18.04" description:"The operating system of the image."` // Windows7, Ubuntu etc. // Deprecated

	Name           string `json:"Name" validate:"required" example:"ami-00aa5a103ddf4509f" description:"The name of the image."`                                   // ami-00aa5a103ddf4509f
	OSArchitecture string `json:"OSArchitecture" validate:"required" example:"x86_64" description:"The architecture of the operating system of the image."`        // arm64, x86_64 etc.
	OSPlatform     string `json:"OSPlatform" validate:"required" example:"Linux/UNIX" description:"The platform of the operating system of the image."`            // Linux/UNIX, Windows, NA
	OSDistribution string `json:"OSDistribution" validate:"required" example:"Ubuntu 22.04~" description:"The distribution of the operating system of the image."` // Ubuntu 22.04~, CentOS 8 etc.
	OSDiskType     string `json:"OSDiskType" validate:"required" example:"gp3" description:"The type of the disk of the image."`                                   // gp3, etc.
	OSDiskSize     string `json:"OSDiskSize" validate:"required" example:"35" description:"The size of the disk of the image."`                                    // 35, etc., GB
	Status         string `json:"Status" validate:"required" example:"Available" description:"The status of the image, e.g., Available or Unavailable."`           // Available, Unavailable

	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty" description:"A list of key-value pairs associated with the image."`
}

type ImageHandler interface {
	CreateImage(imageReqInfo ImageReqInfo) (ImageInfo, error)
	ListImage() ([]*ImageInfo, error)
	GetImage(imageIID IID) (ImageInfo, error)
	CheckWindowsImage(imageIID IID) (bool, error)
	DeleteImage(imageIID IID) (bool, error)
}
