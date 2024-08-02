package resources

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	"github.com/Azure/go-autorest/autorest/to"

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
	Client        *compute.ImagesClient
	VMImageClient *compute.VirtualMachineImagesClient
}

func (imageHandler *AzureImageHandler) setterImage(image compute.Image) *irs.ImageInfo {
	imageInfo := &irs.ImageInfo{
		IId: irs.IID{
			NameId:   *image.Name,
			SystemId: *image.Name,
		},
		GuestOS:      fmt.Sprint(image.ImageProperties.StorageProfile.OsDisk.OsType),
		Status:       *image.ProvisioningState,
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: imageHandler.Region.Region}},
	}

	return imageInfo
}

func (imageHandler *AzureImageHandler) setterVMImage(image compute.VirtualMachineImage) *irs.ImageInfo {
	imageIdArr := strings.Split(*image.ID, "/")
	imageName := fmt.Sprintf("%s:%s:%s:%s", imageIdArr[8], imageIdArr[12], imageIdArr[14], imageIdArr[16])
	imageInfo := &irs.ImageInfo{
		IId: irs.IID{
			NameId:   imageName,
			SystemId: imageName,
		},
		GuestOS:      fmt.Sprint(image.OsDiskImage.OperatingSystem),
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: imageHandler.Region.Region}},
	}

	return imageInfo
}

func (imageHandler *AzureImageHandler) setterVMImageforList(image compute.VirtualMachineImageResource) *irs.ImageInfo {
	imageIdArr := strings.Split(*image.ID, "/")
	imageName := fmt.Sprintf("%s:%s:%s:%s", imageIdArr[8], imageIdArr[12], imageIdArr[14], imageIdArr[16])
	imageInfo := &irs.ImageInfo{
		IId: irs.IID{
			NameId:   imageName,
			SystemId: imageName,
		},
		//GuestOS:      fmt.Sprint(image.OsDiskImage.OperatingSystem),
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
	image, err := imageHandler.Client.Get(imageHandler.Ctx, imageHandler.Region.Region, imageReqInfo.IId.NameId, "")
	if image.ID != nil {
		errMsg := fmt.Sprintf("Image with name %s already exist", imageReqInfo.IId.NameId)
		createErr := errors.New(errMsg)
		return irs.ImageInfo{}, createErr
	}

	createOpts := compute.Image{
		ImageProperties: &compute.ImageProperties{
			StorageProfile: &compute.ImageStorageProfile{
				OsDisk: &compute.ImageOSDisk{
					//BlobURI: to.StringPtr(reqInfo.BlobUrl),
					ManagedDisk: &compute.SubResource{
						ID: to.StringPtr(reqInfo.DiskId),
					},
					OsType: compute.OperatingSystemTypes(reqInfo.OSType),
				},
			},
		},
		Location: &imageHandler.Region.Region,
	}

	future, err := imageHandler.Client.CreateOrUpdate(imageHandler.Ctx, imageHandler.Region.Region, imageReqInfo.IId.NameId, createOpts)
	if err != nil {
		return irs.ImageInfo{}, err
	}
	err = future.WaitForCompletionRef(imageHandler.Ctx, imageHandler.Client.Client)
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

func (imageHandler *AzureImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, Image, "ListImage()")
	start := call.Start()
	var imageList []*irs.ImageInfo

	publishers, err := imageHandler.VMImageClient.ListPublishers(imageHandler.Ctx, imageHandler.Region.Region)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to List Image. err = %s", err.Error()))
		cblogger.Error(createErr)
		LoggingError(hiscallInfo, createErr)
		return nil, createErr
	}

	var publisherNames []string
	for _, p := range *publishers.Value {
		if p.Name == nil {
			continue
		}
		publisherNames = append(publisherNames, *p.Name)
	}

	var routineMax = 300
	var wait sync.WaitGroup
	var mutex = &sync.Mutex{}
	var lenPublisherNames = len(publisherNames)
	var errList []string
	var errMutex = &sync.Mutex{}

	type imageSearchInfo struct {
		PublisherName string
		OfferName     string
		Sku           string
		Version       string
	}
	var imageSearchInfos []imageSearchInfo
	var imageSearchInfosMutex = &sync.Mutex{}

	for i := 0; i < lenPublisherNames; {
		if lenPublisherNames-i < routineMax {
			routineMax = lenPublisherNames - i
		}

		wait.Add(routineMax)

		for j := 0; j < routineMax; j++ {
			pName := publisherNames[i]
			go func(imageHandler *AzureImageHandler, wait *sync.WaitGroup, mutex *sync.Mutex, errList []string, errMutex *sync.Mutex, pName string) {
				defer wait.Done()

				var err error
				var offers compute.ListVirtualMachineImageResource

				for {
					offers, err = imageHandler.VMImageClient.ListOffers(imageHandler.Ctx, imageHandler.Region.Region, pName)
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
					break
				}

				if offers.Value == nil {
					return
				}

				var offerNames []string
				for _, o := range *offers.Value {
					if o.Name == nil {
						continue
					}
					offerNames = append(offerNames, *o.Name)
				}

				var lenOfferNames = len(offerNames)
				var wait2 sync.WaitGroup

				for i := 0; i < lenOfferNames; {
					if lenOfferNames-i < routineMax {
						routineMax = lenOfferNames - i
					}

					wait2.Add(routineMax)

					for j := 0; j < routineMax; j++ {
						go func(wait2 *sync.WaitGroup, oName string) {
							defer wait2.Done()

							var err error
							var skus compute.ListVirtualMachineImageResource

							for {
								skus, err = imageHandler.VMImageClient.ListSkus(imageHandler.Ctx, imageHandler.Region.Region, pName, oName)
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
								break
							}

							if skus.Value == nil {
								return
							}

							var skuNames []string
							for _, s := range *skus.Value {
								if s.Name == nil {
									continue
								}
								skuNames = append(skuNames, *s.Name)
							}

							var lenSkuNames = len(skuNames)
							var wait3 sync.WaitGroup

							for i := 0; i < lenSkuNames; {
								if lenSkuNames-i < routineMax {
									routineMax = lenSkuNames - i
								}

								wait3.Add(routineMax)

								for j := 0; j < routineMax; j++ {
									go func(wait3 *sync.WaitGroup, sName string) {
										defer wait3.Done()

										var err error
										var imageVersionList compute.ListVirtualMachineImageResource

										for {
											imageVersionList, err = imageHandler.VMImageClient.List(imageHandler.Ctx, imageHandler.Region.Region, pName, oName, sName, "", nil, "")
											if err != nil {
												if checkRequest(err.Error()) {
													continue
												}

												errMutex.Lock()
												cblogger.Error(err)
												errList = append(errList, err.Error())
												errMutex.Unlock()

												return
											}
											break
										}

										if imageVersionList.Value == nil {
											return
										}

										for _, iv := range *imageVersionList.Value {
											if iv.ID == nil {
												continue
											}

											imageIdArr := strings.Split(*iv.ID, "/")
											imageVersion := imageIdArr[len(imageIdArr)-1]

											imageSearchInfosMutex.Lock()
											imageSearchInfos = append(imageSearchInfos, imageSearchInfo{
												PublisherName: pName,
												OfferName:     oName,
												Sku:           sName,
												Version:       imageVersion,
											})
											imageSearchInfosMutex.Unlock()
										}
									}(&wait3, skuNames[i])

									i++
									if i == lenSkuNames {
										break
									}
								}

								wait3.Wait()
							}

						}(&wait2, offerNames[i])

						i++
						if i == lenOfferNames {
							break
						}
					}

					wait2.Wait()
				}
			}(imageHandler, &wait, mutex, errList, errMutex, pName)

			i++
			if i == lenPublisherNames {
				break
			}
		}

		wait.Wait()
	}

	if len(errList) == 0 {
		var lenImageSearchInfos = len(imageSearchInfos)
		var wait4 sync.WaitGroup

		for i := 0; i < lenImageSearchInfos; {
			if lenImageSearchInfos-i < routineMax {
				routineMax = lenImageSearchInfos - i
			}

			wait4.Add(routineMax)

			for j := 0; j < routineMax; j++ {
				go func(imageHandler *AzureImageHandler, wait4 *sync.WaitGroup, isInfo imageSearchInfo) {
					defer wait4.Done()

					var err error
					var vmImage compute.VirtualMachineImage

					for {
						vmImage, err = imageHandler.VMImageClient.Get(imageHandler.Ctx, imageHandler.Region.Region, isInfo.PublisherName, isInfo.OfferName, isInfo.Sku, isInfo.Version)
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
						break
					}

					vmImageInfo := imageHandler.setterVMImage(vmImage)
					mutex.Lock()
					imageList = append(imageList, vmImageInfo)
					mutex.Unlock()
				}(imageHandler, &wait4, imageSearchInfos[i])

				i++
				if i == lenImageSearchInfos {
					break
				}
			}

			wait4.Wait()
		}
	}

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

	vmImage, err := imageHandler.VMImageClient.Get(imageHandler.Ctx, imageHandler.Region.Region, imageArr[0], imageArr[1], imageArr[2], imageArr[3])
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Get Image. err = %s", err.Error()))
		cblogger.Error(createErr)
		LoggingError(hiscallInfo, createErr)
		return irs.ImageInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)

	imageInfo := imageHandler.setterVMImage(vmImage)
	return *imageInfo, nil
}

func (imageHandler *AzureImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, imageIID.NameId, "DeleteImage()")

	start := call.Start()
	future, err := imageHandler.Client.Delete(imageHandler.Ctx, imageHandler.Region.Region, imageIID.NameId)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	err = future.WaitForCompletionRef(imageHandler.Ctx, imageHandler.Client.Client)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}

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
