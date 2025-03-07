package resources

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type IbmImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VpcService     *vpcv1.VpcV1
	Ctx            context.Context
}

func (imageHandler *IbmImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, imageReqInfo.IId.NameId, "CreateImage()")

	createErr := errors.New(fmt.Sprintf("Failed to Create Image. err = CreateImage Function Not Offer"))
	cblogger.Error(createErr.Error())
	LoggingError(hiscallInfo, createErr)

	return irs.ImageInfo{}, createErr
}

func (imageHandler *IbmImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, "IMAGE", "ListImage()")

	start := call.Start()

	ListImagesOptions := &vpcv1.ListImagesOptions{}
	ListImagesOptions.SetVisibility("public")
	images, _, err := imageHandler.VpcService.ListImagesWithContext(imageHandler.Ctx, ListImagesOptions)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List Image. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	var imageList []*irs.ImageInfo
	for {
		for _, image := range images.Images {
			if *image.Status == "available" && strings.Contains(*image.Name, "ibm-") {
				imageInfo, err := setImageInfo(&image)
				if err != nil {
					continue
				} else {
					imageList = append(imageList, &imageInfo)
				}
			}
		}
		nextstr, _ := getImageNextHref(images.Next)
		if nextstr != "" {
			ListImagesOptions2 := &vpcv1.ListImagesOptions{
				Start: core.StringPtr(nextstr),
			}
			images, _, err = imageHandler.VpcService.ListImagesWithContext(imageHandler.Ctx, ListImagesOptions2)
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to List Image. err = %s", err.Error()))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return nil, getErr
			}
		} else {
			break
		}
	}
	LoggingInfo(hiscallInfo, start)

	return imageList, nil
}

func (imageHandler *IbmImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, imageIID.NameId, "GetImage()")
	start := call.Start()

	err := checkImageInfoIID(imageIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Image. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.ImageInfo{}, getErr
	}

	image, err := getRawImage(imageIID, imageHandler.VpcService, imageHandler.Ctx)

	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Image. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.ImageInfo{}, getErr
	}

	imageInfo, err := setImageInfo(&image)

	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Image. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.ImageInfo{}, getErr
	}

	LoggingInfo(hiscallInfo, start)
	return imageInfo, nil
}

func (imageHandler *IbmImageHandler) GetImageN(name string) (irs.ImageInfo, error) {
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, name, "GetImage()")
	start := call.Start()

	if name == "" {
		getErr := errors.New(fmt.Sprintf("Failed to Get Image. err = image name is empty"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.ImageInfo{}, getErr
	}

	image, err := getRawImageN(name, imageHandler.VpcService, imageHandler.Ctx)

	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Image. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.ImageInfo{}, getErr
	}

	imageInfo, err := setImageInfo(&image)

	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Image. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.ImageInfo{}, getErr
	}

	LoggingInfo(hiscallInfo, start)
	return imageInfo, nil
}

func (imageHandler *IbmImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, imageIID.NameId, "DeleteImage()")

	createErr := errors.New(fmt.Sprintf("Failed to Delete Image. err = DeleteImage Function Not Offer"))
	cblogger.Error(createErr.Error())
	LoggingError(hiscallInfo, createErr)

	return false, createErr
}

func checkImageInfoIID(imageIID irs.IID) error {
	if imageIID.NameId == "" && imageIID.SystemId == "" {
		return errors.New("invalid IID")
	}
	return nil
}

func getRawImage(imageIID irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) (vpcv1.Image, error) {
	imageId := imageIID.SystemId
	if imageIID.SystemId == "" {
		imageId = imageIID.NameId
	}
	options := &vpcv1.GetImageOptions{}
	options.SetID(imageId)
	image, _, err := vpcService.GetImageWithContext(ctx, options)
	if err != nil {
		return vpcv1.Image{}, err
	}
	return *image, nil
}

func getRawImageN(name string, vpcService *vpcv1.VpcV1, ctx context.Context) (vpcv1.Image, error) {
	options := &vpcv1.GetImageOptions{}
	options.SetID(name)
	image, _, err := vpcService.GetImageWithContext(ctx, options)
	if err != nil {
		return vpcv1.Image{}, err
	}
	return *image, nil
}

func setImageInfo(image *vpcv1.Image) (irs.ImageInfo, error) {
	if image != nil {

		var osPlatform irs.OSPlatform
		if image.OperatingSystem.DisplayName != nil {
			displayName := strings.ToLower(*image.OperatingSystem.DisplayName)
			if strings.Contains(displayName, "windows") {
				osPlatform = irs.Windows
			} else if strings.Contains(displayName, "linux") || strings.Contains(displayName, "z/os") || strings.Contains(displayName, "centos") || strings.Contains(displayName, "fedora") {
				osPlatform = irs.Linux_UNIX
			} else {
				osPlatform = irs.PlatformNA
			}
		}

		var imageStatus irs.ImageStatus
		if image.Status != nil && *image.Status == "available" {
			imageStatus = irs.ImageAvailable
		} else if image.Status != nil {
			imageStatus = irs.ImageUnavailable
		} else {
			imageStatus = irs.ImageNA
		}

		var osArchitecture irs.OSArchitecture
		if image.OperatingSystem.Architecture != nil {
			arch := strings.ToLower(*image.OperatingSystem.Architecture)
			if arch == "arm64" {
				osArchitecture = irs.ARM64
			} else if arch == "arm64_mac" {
				osArchitecture = irs.ARM64_MAC
			} else if arch == "x86_64" || arch == "amd64" {
				osArchitecture = irs.X86_64
			} else if arch == "x86_64_mac" {
				osArchitecture = irs.X86_64_MAC
			} else {
				osArchitecture = irs.ArchitectureNA
			}
		}

		imageInfo := irs.ImageInfo{
			// 2025-01-18: Postpone the deprecation of IID, so revoke IID changes.
			IId: irs.IID{
				NameId:   *image.ID,
				SystemId: *image.ID,
			},
			Name:           *image.ID,
			OSArchitecture: osArchitecture,
			OSPlatform:     osPlatform,
			OSDistribution: *image.OperatingSystem.DisplayName,
			OSDiskType:     "NA",
			OSDiskSizeGB:   "-1",
			ImageStatus:    imageStatus,
			KeyValueList:   irs.StructToKeyValueList(image),
		}

		return imageInfo, nil
	}

	err := errors.New(fmt.Sprintf("operatingSystem invalid"))

	return irs.ImageInfo{}, err
}

func getImageNextHref(next *vpcv1.ImageCollectionNext) (string, error) {
	if next != nil {
		href := *next.Href
		u, err := url.Parse(href)
		if err != nil {
			return "", err
		}
		paramMap, _ := url.ParseQuery(u.RawQuery)
		if paramMap != nil {
			safe := paramMap["start"]
			if safe != nil && len(safe) > 0 {
				return safe[0], nil
			}
		}
	}
	return "", errors.New("NOT NEXT")
}

func (imageHandler *IbmImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, imageIID.NameId, "CheckWindowsImage()")
	start := call.Start()
	var getImageErr error
	rawImage, getImageErr := getRawImage(imageIID, imageHandler.VpcService, imageHandler.Ctx)
	if getImageErr != nil {
		checkWindowsImageErr := errors.New(fmt.Sprintf("Failed to CheckWindowsImage By Image. err = %s", getImageErr.Error()))
		cblogger.Error(checkWindowsImageErr.Error())
		LoggingError(hiscallInfo, checkWindowsImageErr)
		return false, checkWindowsImageErr
	}
	LoggingInfo(hiscallInfo, start)
	isWindows := strings.Contains(strings.ToLower(*rawImage.OperatingSystem.Name), "window")
	return isWindows, nil
}
