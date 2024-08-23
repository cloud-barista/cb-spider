package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	Image = "IMAGE"
)

type AzureImageHandler struct {
	Region        idrv.RegionInfo
	Ctx           context.Context
	Client        *armcompute.ImagesClient
	VMImageClient *armcompute.VirtualMachineImagesClient
}

func (imageHandler *AzureImageHandler) setterImage(image armcompute.Image) *irs.ImageInfo {
	imageInfo := &irs.ImageInfo{
		IId: irs.IID{
			NameId:   *image.Name,
			SystemId: *image.Name,
		},
		GuestOS:      fmt.Sprint(image.Properties.StorageProfile.OSDisk.OSType),
		Status:       *image.Properties.ProvisioningState,
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: imageHandler.Region.Region}},
	}

	return imageInfo
}

func (imageHandler *AzureImageHandler) setterVMImage(imageName string, os string) *irs.ImageInfo {
	imageInfo := &irs.ImageInfo{
		IId: irs.IID{
			NameId:   imageName,
			SystemId: imageName,
		},
		GuestOS:      os,
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: imageHandler.Region.Region}},
	}

	return imageInfo
}

func (imageHandler *AzureImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {

	// @TODO: PublicIP 생성 요청 파라미터 정의 필요
	type ImageReqInfo struct {
		OSType string
		DiskId string
	}

	reqInfo := ImageReqInfo{
		//BlobUrl: "https://md-ds50xp550wh2.blob.core.windows.net/kt0lhznvgx2h/abcd?sv=2017-04-17&sr=b&si=b9674241-fb8e-4cb2-89c7-614d336dc3a7&sig=uvbqvAZQITSpxas%2BWosG%2FGOf6e%2BIBmWNxlUmvARnxiM%3D",
		OSType: "Linux",
		DiskId: "/subscriptions/cb592624-b77b-4a8f-bb13-0e5a48cae40f/resourceGroups/INNO-PLATFORM1-RSRC-GRUP/providers/Microsoft.Compute/disks/inno-test-vm_OsDisk_1_61bf675b990f4aa381d7ee3d766974aa",
		// edited by powerkim for test, 2019.08.13
		//DiskId: "/subscriptions/f1548292-2be3-4acd-84a4-6df079160846/resourceGroups/CB-RESOURCE-GROUP/providers/Microsoft.Compute/disks/vm_name_OsDisk_1_2d63d9cd754c4094b1b1fb6a98c36b71",
	}

	// Check Image Exists
	resp, err := imageHandler.Client.Get(imageHandler.Ctx, imageHandler.Region.Region, imageReqInfo.IId.NameId, nil)
	if err != nil {
		return irs.ImageInfo{}, err
	}

	if resp.Image.ID != nil {
		errMsg := fmt.Sprintf("Image with name %s already exist", imageReqInfo.IId.NameId)
		createErr := errors.New(errMsg)
		return irs.ImageInfo{}, createErr
	}

	osType := armcompute.OperatingSystemTypes(reqInfo.OSType)

	createOpts := armcompute.Image{
		Properties: &armcompute.ImageProperties{
			StorageProfile: &armcompute.ImageStorageProfile{
				OSDisk: &armcompute.ImageOSDisk{
					//BlobURI: to.StringPtr(reqInfo.BlobUrl),
					ManagedDisk: &armcompute.SubResource{
						ID: &reqInfo.DiskId,
					},
					OSType: &osType,
				},
			},
		},
		Location: &imageHandler.Region.Region,
	}

	poller, err := imageHandler.Client.BeginCreateOrUpdate(imageHandler.Ctx, imageHandler.Region.Region, imageReqInfo.IId.NameId, createOpts, nil)
	if err != nil {
		return irs.ImageInfo{}, err
	}
	_, err = poller.PollUntilDone(imageHandler.Ctx, nil)
	if err != nil {
		return irs.ImageInfo{}, err
	}

	// 생성된 Image 정보 리턴
	imageInfo, err := imageHandler.GetImage(imageReqInfo.IId)
	if err != nil {
		return irs.ImageInfo{}, err
	}
	return imageInfo, nil
}

func checkRequest(errMessage string) (repeat bool) {
	pattern := `Please try again after '(\d+)' seconds`
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(errMessage)

	if len(matches) > 1 {
		number := matches[1]
		sec, _ := strconv.Atoi(number)
		time.Sleep(time.Second * time.Duration(sec))

		return true
	}

	return false
}

func determineOSQuickly(pName, sName, oName string) string {
	merged := pName + ":" + sName + ":" + oName
	merged = strings.ToLower(merged)
	merged = strings.ReplaceAll(merged, "_", "-")

	if strings.Contains(merged, "redhat") {
		return "Linux"
	}

	if strings.Contains(merged, "rhel") {
		return "Linux"
	}

	if strings.Contains(merged, "centos") {
		return "Linux"
	}

	if strings.Contains(merged, "rocky") {
		return "Linux"
	}

	if strings.Contains(merged, "ubuntu") {
		return "Linux"
	}

	if strings.Contains(merged, "debian") {
		return "Linux"
	}

	if strings.Contains(merged, "suse") {
		return "Linux"
	}

	if strings.Contains(merged, "linux") {
		return "Linux"
	}

	if strings.Contains(merged, "windows") {
		return "Windows"
	}

	if strings.Contains(merged, "-win-") {
		return "Windows"
	}

	if strings.HasPrefix(merged, "win-") {
		return "Windows"
	}

	if strings.Contains(merged, "win10") {
		return "Windows"
	}

	if strings.Contains(merged, "win11") {
		return "Windows"
	}

	if strings.Contains(merged, "win20") {
		return "Windows"
	}

	if strings.Contains(merged, "windows") {
		return "Windows"
	}

	if strings.Contains(merged, "visualstudio") && !strings.Contains(merged, "code") {
		return "Windows"
	}

	return ""
}

var reqCount int
var reqCountLock = &sync.Mutex{}
var isClearingReqCount bool
var isClearingReqCountLock = &sync.Mutex{}

// Bucket size 250 / Refill rate per sec 25 (https://learn.microsoft.com/en-us/azure/azure-resource-manager/management/request-limits-and-throttling)
var reqWaitCheck = func() {
	reqCountLock.Lock()
	reqCount++

	if reqCount == 25*8 { // Limit to 25 * 8 = 200, remained tokens is 50
		time.Sleep(time.Second)
		reqCount -= 25 // Refill 25 available tokens
	}
	reqCountLock.Unlock()
}

func (imageHandler *AzureImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, Image, "ListImage()")
	start := call.Start()
	var imageList []*irs.ImageInfo

	go func() {
		isClearingReqCountLock.Lock()
		if isClearingReqCount {
			isClearingReqCountLock.Unlock()
			return
		}

		isClearingReqCount = true

		reqCountLock.Lock()
		for {
			time.Sleep(time.Second)
			reqCount -= 25 // Refill 25 available tokens
			if reqCount <= 0 {
				reqCount = 0
				break
			}
		}
		reqCountLock.Unlock()

		isClearingReqCount = false
		isClearingReqCountLock.Unlock()
	}()

	resp, err := imageHandler.VMImageClient.ListPublishers(imageHandler.Ctx, imageHandler.Region.Region, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to List Image. err = %s", err.Error()))
		cblogger.Error(createErr)
		LoggingError(hiscallInfo, createErr)
		return nil, createErr
	}

	var publisherNames []string
	for _, p := range resp.VirtualMachineImageResourceArray {
		if p.Name == nil {
			continue
		}
		publisherNames = append(publisherNames, *p.Name)
	}

	var wait sync.WaitGroup
	var mutex = &sync.Mutex{}
	var errList []string
	var errMutex = &sync.Mutex{}

	wait.Add(len(publisherNames))
	for _, pName := range publisherNames {
		go func(imageHandler *AzureImageHandler, wait *sync.WaitGroup, mutex *sync.Mutex, errList []string, errMutex *sync.Mutex, pName string) {
			defer wait.Done()

			var oNameRespArray []*armcompute.VirtualMachineImageResource

			for {
				reqWaitCheck()
				resp, err := imageHandler.VMImageClient.ListOffers(imageHandler.Ctx, imageHandler.Region.Region, pName, nil)
				if err != nil {
					if checkRequest(err.Error()) {
						continue
					}

					cblogger.Error(err)
					errMutex.Lock()
					errList = append(errList, err.Error())
					errMutex.Unlock()

					return
				}
				oNameRespArray = resp.VirtualMachineImageResourceArray
				break
			}

			var offerNames []string
			for _, o := range oNameRespArray {
				if o.Name == nil {
					continue
				}
				offerNames = append(offerNames, *o.Name)
			}

			var wait2 sync.WaitGroup
			wait2.Add(len(offerNames))

			for i := range offerNames {
				go func(wait2 *sync.WaitGroup, oName string) {
					defer wait2.Done()

					var sNameRespArray []*armcompute.VirtualMachineImageResource

					for {
						reqWaitCheck()
						resp, err := imageHandler.VMImageClient.ListSKUs(imageHandler.Ctx, imageHandler.Region.Region, pName, oName, nil)
						if err != nil {
							if checkRequest(err.Error()) {
								continue
							}

							cblogger.Error(err)
							errMutex.Lock()
							errList = append(errList, err.Error())
							errMutex.Unlock()

							return
						}
						sNameRespArray = resp.VirtualMachineImageResourceArray
						break
					}

					var skuNames []string
					for _, s := range sNameRespArray {
						if s.Name == nil {
							continue
						}
						skuNames = append(skuNames, *s.Name)
					}

					var wait3 sync.WaitGroup
					wait3.Add(len(skuNames))

					for i := range skuNames {
						go func(wait3 *sync.WaitGroup, sName string) {
							defer wait3.Done()

							var versionRespArray []*armcompute.VirtualMachineImageResource

							for {
								reqWaitCheck()
								resp, err := imageHandler.VMImageClient.List(imageHandler.Ctx, imageHandler.Region.Region, pName, oName, sName,
									&armcompute.VirtualMachineImagesClientListOptions{
										Expand:  nil,
										Orderby: toStrPtr("name desc"),
										Top:     toInt32Ptr(1),
									})
								if err != nil {
									if checkRequest(err.Error()) {
										continue
									}

									cblogger.Error(err)
									errMutex.Lock()
									errList = append(errList, err.Error())
									errMutex.Unlock()

									return
								}
								versionRespArray = resp.VirtualMachineImageResourceArray
								break
							}

							if len(versionRespArray) == 0 {
								return
							}

							latest := versionRespArray[0]
							if latest.ID == nil {
								return
							}

							imageIdArr := strings.Split(*latest.ID, "/")
							imageVersion := imageIdArr[len(imageIdArr)-1]

							os := determineOSQuickly(pName, oName, sName)
							if os == "" {
								for {
									reqWaitCheck()
									resp, err := imageHandler.VMImageClient.Get(imageHandler.Ctx, imageHandler.Region.Region, pName, oName, sName, imageVersion, nil)
									if err != nil {
										if checkRequest(err.Error()) {
											continue
										}

										cblogger.Error(err)
										errMutex.Lock()
										errList = append(errList, err.Error())
										errMutex.Unlock()

										return
									}

									if resp.VirtualMachineImage.Properties != nil &&
										resp.VirtualMachineImage.Properties.OSDiskImage != nil &&
										resp.VirtualMachineImage.Properties.OSDiskImage.OperatingSystem != nil {
										os = string(*resp.VirtualMachineImage.Properties.OSDiskImage.OperatingSystem)
									}

									break
								}
							}

							imageName := pName + ":" + oName + ":" + sName + ":" + imageVersion

							vmImageInfo := imageHandler.setterVMImage(imageName, os)
							mutex.Lock()
							imageList = append(imageList, vmImageInfo)
							mutex.Unlock()
						}(&wait3, skuNames[i])
					}
					wait3.Wait()
				}(&wait2, offerNames[i])
			}
			wait2.Wait()
		}(imageHandler, &wait, mutex, errList, errMutex, pName)
	}
	wait.Wait()

	if len(errList) > 0 {
		cblogger.Error(strings.Join(errList, "\n"))
		return nil, errors.New(strings.Join(errList, "\n"))
	}

	LoggingInfo(hiscallInfo, start)
	return imageList, nil
}

func (imageHandler *AzureImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, imageIID.NameId, "GetImage()")
	start := call.Start()
	imageArr := strings.Split(imageIID.NameId, ":")

	// 이미지 URN 형식 검사
	if len(imageArr) != 4 {
		createErr := errors.New(fmt.Sprintf("Failed to Get Image. err = %s", "invalid format for image ID, imageId="+imageIID.NameId))
		cblogger.Error(createErr)
		LoggingError(hiscallInfo, createErr)
		return irs.ImageInfo{}, createErr
	}

	// 해당 이미지 publisher, offer, skus 기준 version 목록 조회 (latest 기준 조회 기능 미활용)
	/*
		imageVersion := imageArr[3]
		if strings.EqualFold(imageVersion, "latest") {
			vmImageList, err := imageHandler.VMImageClient.List(imageHandler.Ctx, imageHandler.Region.Region, imageArr[0], imageArr[1], imageArr[2], "", to.Int32Ptr(1), "name desc")
			if err != nil {
				LoggingError(hiscallInfo, err)
				return irs.ImageInfo{}, err
			}
			if &vmImageList == nil {
				getErr := errors.New(fmt.Sprintf("could not found image with imageId %s", imageIID.NameId))
				LoggingError(hiscallInfo, getErr)
				return irs.ImageInfo{}, getErr
			}
			if vmImageList.Value == nil {
				getErr := errors.New(fmt.Sprintf("could not found image with imageId %s", imageIID.NameId))
				LoggingError(hiscallInfo, getErr)
				return irs.ImageInfo{}, getErr
			}
			if len(*vmImageList.Value) == 0 {
				getErr := errors.New(fmt.Sprintf("could not found image with imageId %s", imageIID.NameId))
				LoggingError(hiscallInfo, getErr)
				return irs.ImageInfo{}, getErr
			} else {
				latestVmImage := (*vmImageList.Value)[0]
				imageIdArr := strings.Split(*latestVmImage.ID, "/")
				imageVersion = imageIdArr[len(imageIdArr)-1]
			}
		}
	*/

	// 1개의 버전 정보를 기준으로 이미지 정보 조회

	resp, err := imageHandler.VMImageClient.Get(imageHandler.Ctx, imageHandler.Region.Region, imageArr[0], imageArr[1], imageArr[2], imageArr[3], nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Get Image. err = %s", err.Error()))
		cblogger.Error(createErr)
		LoggingError(hiscallInfo, createErr)
		return irs.ImageInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)

	if resp.VirtualMachineImage.Properties != nil &&
		resp.VirtualMachineImage.Properties.OSDiskImage != nil &&
		resp.VirtualMachineImage.Properties.OSDiskImage.OperatingSystem != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Get Image. err = Failed to get image information"))
		cblogger.Error(createErr)
		LoggingError(hiscallInfo, createErr)
		return irs.ImageInfo{}, createErr
	}

	imageInfo := imageHandler.setterVMImage(strings.Join(imageArr, ":"), string(*resp.VirtualMachineImage.Properties.OSDiskImage.OperatingSystem))
	return *imageInfo, nil
}

func (imageHandler *AzureImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, imageIID.NameId, "DeleteImage()")

	start := call.Start()
	poller, err := imageHandler.Client.BeginDelete(imageHandler.Ctx, imageHandler.Region.Region, imageIID.NameId, nil)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}
	_, err = poller.PollUntilDone(imageHandler.Ctx, nil)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func (imageHandler *AzureImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, imageIID.NameId, "CheckWindowsImage()")
	start := call.Start()
	if imageIID.NameId == "" && imageIID.SystemId == "" {
		checkWindowsImageErr := errors.New(fmt.Sprintf("Failed to CheckWindowsImage By Image. err = empty ImageIID"))
		cblogger.Error(checkWindowsImageErr.Error())
		LoggingError(hiscallInfo, checkWindowsImageErr)
		return false, checkWindowsImageErr
	}
	imageName := imageIID.NameId
	if imageIID.NameId == "" {
		imageName = imageIID.SystemId
	}
	imageNameSplits := strings.Split(imageName, ":")
	if len(imageNameSplits) != 4 {
		checkWindowsImageErr := errors.New(fmt.Sprintf("Failed to CheckWindowsImage By Image. err = invalid ImageIID, Image Name must be in the form of 'Publisher:Offer:Sku:Version'. "))
		cblogger.Error(checkWindowsImageErr.Error())
		LoggingError(hiscallInfo, checkWindowsImageErr)
		return false, checkWindowsImageErr
	}
	offer := imageNameSplits[1]
	LoggingInfo(hiscallInfo, start)
	if strings.Contains(strings.ToLower(offer), "window") {
		return true, nil
	}
	return false, nil
}
