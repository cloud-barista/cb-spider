// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"strconv"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaImageHandler struct {
	Region idrv.RegionInfo
	Client *ecs.Client
}

func (imageHandler *AlibabaImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	cblogger.Info("Start CreateImage : ", imageReqInfo)
	//imageIdArr := strings.Split(imageReqInfo.Id, ":")

	request := ecs.CreateCreateImageRequest()
	request.Scheme = "https"

	// 필수 Req Name
	request.ImageName = imageReqInfo.IId.NameId // ImageName
	request.Tag = &[]ecs.CreateImageTag{        // Default Hidden Tags Info
		{
			Key:   CBMetaDefaultTagName,  // "cbCat",
			Value: CBMetaDefaultTagValue, // "cbAlibaba",
		},
	}

	// 요청 매개 변수의 우선 순위는 InstanceId, DiskDeviceMapping, SnapshotId 순서

	// Case1 - 인스턴스 ID (InstanceId)를 지정하여 사용자 지정 이미지를 생성
	// 향후 추가를 고려, for create Case 1 (InstanceId)
	// request.InstanceId = imageReqInfo.InstanceId // "i-t4n98732cvvbbhhbsd4r"

	// >>>> Case2 - 시스템 디스크 또는 스냅 샷 (SnapshotId) 중 하나를 지정하여 사용자 정의 이미지를 생성
	// for create Case 2 (SnapshotId)
	//request.SnapshotId = imageReqInfo.Id // SnapshotId

	// Case3 - 여러 디스크의 스냅 샷을 이미지 템플릿으로 결합하려는 경우 DiskDeviceMapping을 지정하여 사용자 지정 이미지를 만들 수 있습니다.
	// 향후 추가를 고려, for create Case 3 (DiskDeviceMapping)
	// request.DiskDeviceMapping = &[]ecs.CreateImageDiskDeviceMapping{
	// 	{
	// 	  Size: imageReqInfo.DiskDevice[0].Size, // "20",
	// 	  SnapshotId: imageReqInfo.DiskDevice[0].Id, // "s-t4nhjof9caedzwd4929k",
	// 	  Device: imageReqInfo.DiskDevice[0].Device, // "/dev/xvda",
	// 	  DiskType: imageReqInfo.DiskDevice[0].DiskType, // "system",
	// 	},
	// 	{
	// 	  Size: imageReqInfo.DiskDevice[1].Size, // "20",
	// 	  SnapshotId: imageReqInfo.DiskDevice[1].Id, // "s-t4nhjof9caedzwd4929l",
	// 	  Device: imageReqInfo.DiskDevice[1].Device, // "/dev/xvdb",
	// 	  DiskType: imageReqInfo.DiskDevice[1].DiskType, // "data",
	// 	},
	//   }

	// 추가 옵션 Req
	// request.Description = imageReqInfo.Description // "cb custom img01"
	// request.Platform = imageReqInfo.Platform // "Ubuntu"
	// request.Architecture = imageReqInfo.Architecture // "x86_64"
	// request.OSType = imageReqInfo.OSType // OSType "linux"

	// Check Image Exists

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: imageReqInfo.IId.SystemId,
		CloudOSAPI:   "CreateImage()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	// Creates a new custom Image with the given name
	result, err := imageHandler.Client.CreateImage(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to create Image: %s, %v.", imageReqInfo.IId.NameId, err)
		return irs.ImageInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Infof("Created Image %q %s\n %s\n", result.ImageId, imageReqInfo.IId.NameId, result.RequestId)
	cblogger.Debug(result)

	/*
		ImageInfo := irs.ImageInfo{
			Id:          result.ImageId,
			Name:        *imageReqInfo.ImageName,
		}
	*/

	// 생성된 Image 정보 획득 후, Image 정보 리턴
	imageInfo, err := imageHandler.GetImage(imageReqInfo.IId)
	if err != nil {
		return irs.ImageInfo{}, err
	}

	return imageInfo, nil
}

func (imageHandler *AlibabaImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	cblogger.Debug("Start")
	var imageInfoList []*irs.ImageInfo

	request := ecs.CreateDescribeImagesRequest()
	request.Scheme = "https"

	request.Status = "Available"
	request.ActionType = "*"
	if CBPageOn {
		request.PageNumber = requests.NewInteger(CBPageNumber)
		request.PageSize = requests.NewInteger(CBPageSize)
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: "ListImage()",
		CloudOSAPI:   "DescribeImages()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()

	var totalCount = 0
	curPage := CBPageNumber
	for {
		result, err := imageHandler.Client.DescribeImages(request)
		callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
		//cblogger.Debug(result) //출력 정보가 너무 많아서 생략
		if err != nil {
			callLogInfo.ErrorMSG = err.Error()
			callogger.Error(call.String(callLogInfo))

			cblogger.Errorf("Unable to get Images, %v", err)
			return nil, err
		}
		callogger.Info(call.String(callLogInfo))

		//cnt := 0
		for _, cur := range result.Images.Image {
			cblogger.Debugf("[%s] Processing image information", cur.ImageId)
			imageInfo := ExtractImageDescribeInfo(&cur)
			imageInfoList = append(imageInfoList, &imageInfo)
		}

		if CBPageOn {
			totalCount = len(imageInfoList)
			cblogger.Infof("Total number of images across CSP: [%d] - Current page: [%d] - Accumulated result count: [%d]", result.TotalCount, curPage, totalCount)
			if totalCount >= result.TotalCount {
				break
			}
			curPage++
			request.PageNumber = requests.NewInteger(curPage)
		} else {
			break
		}
	}
	//cblogger.Debug(imageInfoList)
	return imageInfoList, nil
}

func extractOsPlatform(image *ecs.Image) irs.OSPlatform {
	// Ubuntu
	// Rocky Linux
	// Debian
	// Aliyun
	// AlmaLinux
	// CentOS Stream
	// Windows Server 2025
	// Freebsd
	// Anolis
	// openSUSE
	// Gentoo
	platformInfo := image.Platform
	osPlatform := irs.PlatformNA

	lowerCasePlatformInfo := strings.ToLower(platformInfo)

	if strings.Contains(lowerCasePlatformInfo, "windows") {
		osPlatform = irs.Windows
	} else if strings.Contains(lowerCasePlatformInfo, "ubuntu") ||
		strings.Contains(lowerCasePlatformInfo, "rocky") ||
		strings.Contains(lowerCasePlatformInfo, "debian") ||
		strings.Contains(lowerCasePlatformInfo, "aliyun") ||
		strings.Contains(lowerCasePlatformInfo, "almalinux") ||
		strings.Contains(lowerCasePlatformInfo, "centos") ||
		strings.Contains(lowerCasePlatformInfo, "freebsd") ||
		strings.Contains(lowerCasePlatformInfo, "anolis") ||
		strings.Contains(lowerCasePlatformInfo, "opensuse") ||
		strings.Contains(lowerCasePlatformInfo, "gentoo") {
		osPlatform = irs.Linux_UNIX
	}
	return osPlatform
}

func extractOsArchitecture(image *ecs.Image) irs.OSArchitecture {

	architectureInfo := image.Architecture
	osArchitecture := irs.ArchitectureNA

	lowerCaseArchitectureInfo := strings.ToLower(architectureInfo)

	if strings.Contains(lowerCaseArchitectureInfo, "x86_64") {
		osArchitecture = irs.X86_64
	} else if strings.Contains(lowerCaseArchitectureInfo, "arm64") {
		osArchitecture = irs.ARM64
	} else if strings.Contains(lowerCaseArchitectureInfo, "arm64_mac") {
		osArchitecture = irs.ARM64_MAC
	} else if strings.Contains(lowerCaseArchitectureInfo, "x86_32") || strings.Contains(lowerCaseArchitectureInfo, "i386") {
		osArchitecture = irs.X86_32
	} else if strings.Contains(lowerCaseArchitectureInfo, "x86_32_mac") {
		osArchitecture = irs.X86_32_MAC
	} else if strings.Contains(lowerCaseArchitectureInfo, "x86_64_mac") {
		osArchitecture = irs.X86_64_MAC
	}

	return osArchitecture
}

// https://pkg.go.dev/github.com/aliyun/alibaba-cloud-sdk-go/services/ecs?tab=doc#Image
// package ecs v1.61.170 Latest Published: Apr 30, 2020
// Image 정보를 추출함
func ExtractImageDescribeInfo(image *ecs.Image) irs.ImageInfo {
	//@TODO : 2020-03-26 Ali클라우드 API 구조가 바뀐 것 같아서 임시로 변경해 놓음.
	//func ExtractImageDescribeInfo(image *ecs.ImageInDescribeImages) irs.ImageInfo {
	//@TODO : 2020-04-20 ecs.ImageInDescribeImages를 인식 못해서 다시 ecs.Image로 변경해 놓음.
	//func ExtractImageDescribeInfo(image *ecs.Image) irs.ImageInfo {
	//*ecs.DescribeImagesResponse
	cblogger.Debug(image)

	imageInfo := irs.ImageInfo{
		// 2025-01-18: Postpone the deprecation of IID, so revoke IID changes.
		IId: irs.IID{NameId: image.ImageId, SystemId: image.ImageId},
		// //Name:    image.ImageName,
		// Status:  image.Status,
		// GuestOS: image.OSNameEn,
		Name:           image.ImageName,
		OSArchitecture: extractOsArchitecture(image),
		OSPlatform:     extractOsPlatform(image),
		OSDistribution: image.OSNameEn,
		OSDiskType:     "NA",
		OSDiskSizeGB:   strconv.Itoa(image.Size),
		ImageStatus:    irs.ImageStatus(image.Status),
		KeyValueList:   irs.StructToKeyValueList(image),
	}

	return imageInfo
}

func (imageHandler *AlibabaImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	cblogger.Infof("imageID : ", imageIID.SystemId)

	// request := ecs.CreateDescribeImagesRequest()
	// request.Scheme = "https"

	// // request.Status = "Available"
	// // request.ActionType = "*"

	// request.ImageId = imageIID.SystemId

	// // logger for HisCall
	// callogger := call.GetLogger("HISCALL")
	// callLogInfo := call.CLOUDLOGSCHEMA{
	// 	CloudOS:      call.ALIBABA,
	// 	RegionZone:   imageHandler.Region.Zone,
	// 	ResourceType: call.VMIMAGE,
	// 	ResourceName: imageIID.SystemId,
	// 	CloudOSAPI:   "DescribeImages()",
	// 	ElapsedTime:  "",
	// 	ErrorMSG:     "",
	// }

	// callLogStart := call.Start()
	// result, err := imageHandler.Client.DescribeImages(request)
	// callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	// //ecs.DescribeImagesResponse.Images.Image
	// //cblogger.Debug(result)
	// cblogger.Info(result)
	// if err != nil {
	// 	callLogInfo.ErrorMSG = err.Error()
	// 	callogger.Error(call.String(callLogInfo))

	// 	cblogger.Errorf("Unable to get Images, %v", err)
	// 	return irs.ImageInfo{}, err
	// }
	// callogger.Info(call.String(callLogInfo))

	// if result.TotalCount < 1 {
	// 	return irs.ImageInfo{}, errors.New("Notfound: '" + imageIID.SystemId + "' Images Not found")
	// }

	result, err := DescribeImageByImageId(imageHandler.Client, imageHandler.Region, imageIID, false)

	if err != nil {
		return irs.ImageInfo{}, err
	}

	imageInfo := ExtractImageDescribeInfo(&result)

	return imageInfo, nil
}

func (imageHandler *AlibabaImageHandler) GetImageN(Name string) (irs.ImageInfo, error) {
	cblogger.Infof("imageId : ", Name)

	result, err := DescribeImageByImageName(imageHandler.Client, imageHandler.Region, Name, false)

	if err != nil {
		return irs.ImageInfo{}, err
	}

	imageInfo := ExtractImageDescribeInfo(&result)

	return imageInfo, nil
}

func (imageHandler *AlibabaImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	cblogger.Infof("DeleteImage : [%s]", imageIID.SystemId)
	// Delete the Image by Id

	request := ecs.CreateDeleteImageRequest()
	request.Scheme = "https"

	//request.ImageId = to.StringPtr(imageID)
	request.ImageId = imageIID.SystemId
	// 추가 옵션 Req
	// request.Force = requests.NewBoolean(true)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: imageIID.SystemId,
		CloudOSAPI:   "DeleteImage()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	result, err := imageHandler.Client.DeleteImage(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Debug(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to delete Image: %s, %v.", imageIID.SystemId, err)
		return false, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Infof("Successfully deleted %q Image\n", imageIID.SystemId)

	return true, nil
}

// WindowOS 여부
func (imageHandler *AlibabaImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	isWindows := false
	isMyImage := false

	osType, err := DescribeImageOsType(imageHandler.Client, imageHandler.Region, imageIID, isMyImage)

	if err != nil {
		return isWindows, err
	}

	if osType == "windows" {
		isWindows = true
	}
	cblogger.Info("isWindows = ", isWindows)
	return isWindows, nil
}
