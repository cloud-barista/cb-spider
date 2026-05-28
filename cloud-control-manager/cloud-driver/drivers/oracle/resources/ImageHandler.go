package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
)

type OracleImageHandler struct {
	Region        idrv.RegionInfo
	CompartmentID string
	Client        core.ComputeClient
	Ctx           context.Context
}

func (handler *OracleImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	return irs.ImageInfo{}, errors.New("Oracle Driver: CreateImage is not supported by ImageHandler")
}

func (handler *OracleImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	images, err := handler.listImages()
	if err != nil {
		return nil, err
	}
	infos := make([]*irs.ImageInfo, 0, len(images))
	for _, image := range images {
		info := handler.imageInfo(image)
		infos = append(infos, &info)
	}
	return infos, nil
}

func (handler *OracleImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	image, err := handler.getImage(imageIID)
	if err != nil {
		return irs.ImageInfo{}, err
	}
	return handler.imageInfo(image), nil
}

func (handler *OracleImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	image, err := handler.getImage(imageIID)
	if err != nil {
		return false, err
	}
	return strings.Contains(strings.ToLower(stringValue(image.OperatingSystem)), "windows"), nil
}

func (handler *OracleImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	return false, errors.New("Oracle Driver: DeleteImage is not supported by ImageHandler")
}

func (handler *OracleImageHandler) getImage(iid irs.IID) (core.Image, error) {
	id := iid.SystemId
	if id == "" {
		id = iid.NameId
	}
	if isOracleOCID(id) {
		resp, err := handler.Client.GetImage(handler.Ctx, core.GetImageRequest{ImageId: common.String(id)})
		if err != nil {
			return core.Image{}, statusErr("failed to get Oracle image", err)
		}
		return resp.Image, nil
	}

	images, err := handler.listImagesByDisplayName(id)
	if err != nil {
		return core.Image{}, err
	}
	if len(images) == 0 {
		return core.Image{}, fmt.Errorf("Oracle image not found: %s", id)
	}
	return images[0], nil
}

func (handler *OracleImageHandler) listImages() ([]core.Image, error) {
	return handler.listImagesWithRequest(core.ListImagesRequest{CompartmentId: common.String(handler.CompartmentID), LifecycleState: core.ImageLifecycleStateAvailable, SortBy: core.ListImagesSortByTimecreated, SortOrder: core.ListImagesSortOrderDesc})
}

func (handler *OracleImageHandler) listImagesByDisplayName(displayName string) ([]core.Image, error) {
	return handler.listImagesWithRequest(core.ListImagesRequest{CompartmentId: common.String(handler.CompartmentID), DisplayName: common.String(displayName), LifecycleState: core.ImageLifecycleStateAvailable, SortBy: core.ListImagesSortByTimecreated, SortOrder: core.ListImagesSortOrderDesc})
}

func (handler *OracleImageHandler) listImagesWithRequest(req core.ListImagesRequest) ([]core.Image, error) {
	images := make([]core.Image, 0)
	page := ""
	for {
		if page != "" {
			req.Page = common.String(page)
		}
		resp, err := handler.Client.ListImages(handler.Ctx, req)
		if err != nil {
			return nil, statusErr("failed to list Oracle images", err)
		}
		images = append(images, resp.Items...)
		if resp.OpcNextPage == nil || *resp.OpcNextPage == "" {
			break
		}
		page = *resp.OpcNextPage
	}
	return images, nil
}

// extractOsArchitecture derives OSArchitecture from the image DisplayName.
// OCI Image struct has no dedicated architecture field; the architecture is
// embedded in the image name (e.g. "aarch64" for ARM64, otherwise x86_64).
func extractOsArchitecture(image core.Image) irs.OSArchitecture {
	nameLower := strings.ToLower(stringValue(image.DisplayName))
	if strings.Contains(nameLower, "aarch64") || strings.Contains(nameLower, "arm64") {
		return irs.ARM64
	}
	return irs.X86_64
}

// extractOsPlatform derives OSPlatform from the OperatingSystem field.
// OCI OperatingSystem values: "Oracle Linux", "Canonical Ubuntu", "CentOS",
// "Windows", "Rocky Linux", "Debian", etc.
func extractOsPlatform(image core.Image) irs.OSPlatform {
	osLower := strings.ToLower(stringValue(image.OperatingSystem))
	osPlatform := irs.PlatformNA

	if strings.Contains(osLower, "windows") {
		osPlatform = irs.Windows
	} else if strings.Contains(osLower, "oracle") ||
		strings.Contains(osLower, "ubuntu") ||
		strings.Contains(osLower, "centos") ||
		strings.Contains(osLower, "rocky") ||
		strings.Contains(osLower, "debian") ||
		strings.Contains(osLower, "red hat") ||
		strings.Contains(osLower, "redhat") ||
		strings.Contains(osLower, "opensuse") ||
		strings.Contains(osLower, "suse") ||
		strings.Contains(osLower, "almalinux") ||
		strings.Contains(osLower, "freebsd") {
		osPlatform = irs.Linux_UNIX
	}
	return osPlatform
}

func (handler *OracleImageHandler) imageInfo(image core.Image) irs.ImageInfo {
	name := stringValue(image.DisplayName)
	if name == "" {
		name = stringValue(image.Id)
	}
	status := irs.ImageUnavailable
	if image.LifecycleState == core.ImageLifecycleStateAvailable {
		status = irs.ImageAvailable
	}
	diskSize := "-1"
	if image.SizeInMBs != nil && *image.SizeInMBs > 0 {
		diskSize = strconv.FormatInt((*image.SizeInMBs+1023)/1024, 10)
	}
	return irs.ImageInfo{
		IId:            irs.IID{NameId: name, SystemId: stringValue(image.Id)},
		Name:           name,
		OSArchitecture: extractOsArchitecture(image),
		OSPlatform:     extractOsPlatform(image),
		OSDistribution: strings.TrimSpace(stringValue(image.OperatingSystem) + " " + stringValue(image.OperatingSystemVersion)),
		OSDiskType:     "block",
		OSDiskSizeGB:   diskSize,
		ImageStatus:    status,
		KeyValueList: []irs.KeyValue{
			{Key: "OperatingSystem", Value: stringValue(image.OperatingSystem)},
			{Key: "OperatingSystemVersion", Value: stringValue(image.OperatingSystemVersion)},
			{Key: "LaunchMode", Value: string(image.LaunchMode)},
			{Key: "ListingType", Value: string(image.ListingType)},
		},
	}
}

func isOracleOCID(value string) bool {
	return strings.HasPrefix(value, "ocid1.")
}
