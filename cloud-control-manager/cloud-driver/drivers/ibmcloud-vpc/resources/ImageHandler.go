package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"net/url"
	"strings"
)

type IbmImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VpcService     *vpcv1.VpcV1
	Ctx            context.Context
}

func (imageHandler *IbmImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, imageReqInfo.IId.NameId, "CreateImage()")
	// start := call.Start()
	err := errors.New(fmt.Sprintf("CreateImage Function Not Offer"))
	LoggingError(hiscallInfo, err)
	return irs.ImageInfo{}, errors.New(fmt.Sprintf("CreateImage Function Not Offer"))
}
func (imageHandler *IbmImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, "IMAGE", "ListImage()")
	// start := call.Start()
	ListImagesOptions := &vpcv1.ListImagesOptions{}
	ListImagesOptions.SetVisibility("public")
	images, _, err := imageHandler.VpcService.ListImagesWithContext(imageHandler.Ctx, ListImagesOptions)

	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
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
				LoggingError(hiscallInfo, err)
				return nil, err

			}
		} else {
			break
		}
	}
	return imageList, nil
}
func (imageHandler *IbmImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, imageIID.NameId, "GetImage()")
	start := call.Start()

	err := checkImageInfoIID(imageIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.ImageInfo{}, err
	}
	image, err := getRawImage(imageIID, imageHandler.VpcService, imageHandler.Ctx)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.ImageInfo{}, err
	}
	imageInfo, err := setImageInfo(&image)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.ImageInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)
	return imageInfo, nil
}
func (imageHandler *IbmImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, imageIID.NameId, "DeleteImage()")
	// start := call.Start()
	err := errors.New(fmt.Sprintf("DeleteImage Function Not Offer"))
	LoggingError(hiscallInfo, err)
	return false, err
}

func checkImageInfoIID(imageIID irs.IID) error {
	if imageIID.NameId == "" && imageIID.SystemId == "" {
		return errors.New("invalid IID")
	}
	return nil
}

func getRawImage(imageIID irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) (vpcv1.Image, error) {
	if imageIID.SystemId == "" {
		ListImagesOptions := &vpcv1.ListImagesOptions{}
		ListImagesOptions.SetVisibility("public")
		images, _, err := vpcService.ListImagesWithContext(ctx, ListImagesOptions)
		if err != nil {
			return vpcv1.Image{}, err
		}
		for {
			for _, image := range images.Images {
				if *image.Name == imageIID.NameId {
					return image, nil
				}
			}
			nextstr, _ := getImageNextHref(images.Next)
			if nextstr != "" {
				ListImagesOptions2 := &vpcv1.ListImagesOptions{
					Start: core.StringPtr(nextstr),
				}
				images, _, err = vpcService.ListImagesWithContext(ctx, ListImagesOptions2)
				if err != nil {
					return vpcv1.Image{}, err
				}
			} else {
				break
			}
		}
		return vpcv1.Image{}, errors.New(fmt.Sprintf("not found %s", imageIID.NameId))
	} else {
		options := &vpcv1.GetImageOptions{}
		options.SetID(imageIID.SystemId)
		image, _, err := vpcService.GetImageWithContext(ctx, options)
		if err != nil {
			return vpcv1.Image{}, err
		}
		return *image, nil
	}
}

func setImageInfo(image *vpcv1.Image) (irs.ImageInfo, error) {
	if image != nil {
		imageInfo := irs.ImageInfo{
			IId: irs.IID{
				NameId:   *image.Name,
				SystemId: *image.ID,
			},
			GuestOS: *image.OperatingSystem.DisplayName,
			Status:  "available",
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
