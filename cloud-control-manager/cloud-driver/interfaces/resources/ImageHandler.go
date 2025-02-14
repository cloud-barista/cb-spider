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

type OSArchitecture string

const (
	ARM32          OSArchitecture = "arm32"
	ARM64          OSArchitecture = "arm64"
	ARM64_MAC      OSArchitecture = "arm64_mac"
	X86_32         OSArchitecture = "x86_32"
	X86_64         OSArchitecture = "x86_64"
	X86_32_MAC     OSArchitecture = "x86_32_mac"
	X86_64_MAC     OSArchitecture = "x86_64_mac"
	ArchitectureNA OSArchitecture = "NA"
)

type OSPlatform string

const (
	Linux_UNIX OSPlatform = "Linux/UNIX"
	Windows    OSPlatform = "Windows"
	PlatformNA OSPlatform = "NA"
)

type ImageStatus string

const (
	ImageAvailable   ImageStatus = "Available"
	ImageUnavailable ImageStatus = "Unavailable"
	ImageNA          ImageStatus = "NA"
)

// ImageInfo represents the information of an Image.
type ImageInfo struct {
	IId IID `json:"IId" validate:"required" description:"The ID of the image."` // {NameId, SystemId} // Deprecated

	Name           string         `json:"Name" validate:"required" example:"ami-00aa5a103ddf4509f" description:"The name of the image."`                                   // ami-00aa5a103ddf4509f
	OSArchitecture OSArchitecture `json:"OSArchitecture" validate:"required" example:"x86_64" description:"The architecture of the operating system of the image."`        // arm64, x86_64 etc.
	OSPlatform     OSPlatform     `json:"OSPlatform" validate:"required" example:"Linux/UNIX" description:"The platform of the operating system of the image."`            // Linux/UNIX, Windows, NA
	OSDistribution string         `json:"OSDistribution" validate:"required" example:"Ubuntu 22.04~" description:"The distribution of the operating system of the image."` // Ubuntu 22.04~, CentOS 8 etc.
	OSDiskType     string         `json:"OSDiskType" validate:"required" example:"HDD" description:"The type of the root disk of for the VM being created."`               // ebs, HDD, etc.
	OSDiskSizeInGB string         `json:"OSDiskSizeInGB" validate:"required" example:"50" description:"The (minimum) root disk size in GB for the VM being created."`      // 10, 50, 100 etc.

	ImageStatus ImageStatus `json:"ImageStatus" validate:"required" example:"Available" description:"The status of the image, e.g., Available or Unavailable."` // Available, Unavailable

	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty" description:"A list of key-value pairs associated with the image."`
}

type ImageHandler interface {
	CreateImage(imageReqInfo ImageReqInfo) (ImageInfo, error)
	ListImage() ([]*ImageInfo, error)
	GetImage(imageIID IID) (ImageInfo, error)
	CheckWindowsImage(imageIID IID) (bool, error)
	DeleteImage(imageIID IID) (bool, error)
}
