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
	"reflect"
	"strconv"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
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
	request.ImageName = imageReqInfo.Name // ImageName
	request.Tag = &[]ecs.CreateImageTag{  // Default Hidden Tags Info
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
	request.SnapshotId = imageReqInfo.Id // SnapshotId

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

	// Creates a new custom Image with the given name
	result, err := imageHandler.Client.CreateImage(request)
	if err != nil {
		// if aerr, ok := err.(errors.Error); ok && aerr.Code() == "InvalidKeyPair.Duplicate" {
		// 	cblogger.Errorf("Image %q already exists.", imageReqInfo.ImageName)
		// 	return irs.ImageReqInfo{}, err
		// }
		cblogger.Errorf("Unable to create Image: %s, %v.", imageReqInfo.Name, err)
		return irs.ImageInfo{}, err
	}

	cblogger.Infof("Created Image %q %s\n %s\n", result.ImageId, imageReqInfo.Name, result.RequestId)
	spew.Dump(result)

	/*
		ImageInfo := irs.ImageInfo{
			Id:          result.ImageId,
			Name:        *imageReqInfo.ImageName,
		}
	*/

	// 생성된 Image 정보 획득 후, Image 정보 리턴
	imageInfo, err := imageHandler.GetImage(imageReqInfo.Name)
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

	result, err := imageHandler.Client.DescribeImages(request)
	//spew.Dump(result)	//출력 정보가 너무 많아서 생략
	if err != nil {
		// if aerr, ok := err.(errors.Error); ok {
		// 	switch aerr.Code() {
		// 	default:
		// 		cblogger.Error(aerr.Error())
		// 	}
		// } else {
		// 	// Print the error, cast err to awserr.Error to get the Code and
		// 	// Message from an error.
		// 	cblogger.Error(err.Error())
		// }
		cblogger.Errorf("Unable to get Images, %v", err)
		return nil, err
	}

	//cnt := 0
	for _, cur := range result.Images.Image {
		cblogger.Infof("[%s] Image 정보 처리", cur.ImageId)
		imageInfo := ExtractImageDescribeInfo(&cur)
		imageInfoList = append(imageInfoList, &imageInfo)
		/*
			cnt++
			if cnt > 20 {
				break
			}
		*/
	}

	//spew.Dump(imageInfoList)
	return imageInfoList, nil
}

//Image 정보를 추출함
func ExtractImageDescribeInfo(image *ecs.Image) irs.ImageInfo {
	//spew.Dump(image)
	imageInfo := irs.ImageInfo{
		Id:     image.ImageId,
		Name:   image.ImageName,
		Status: image.Status,
	}

	keyValueList := []irs.KeyValue{
		{Key: "CreationTime", Value: image.CreationTime},
		{Key: "Architecture", Value: image.Architecture},

		{Key: "OSNameEn", Value: image.OSNameEn},
		{Key: "ProductCode", Value: image.ProductCode},
		{Key: "OSType", Value: image.OSType},
		{Key: "OSName", Value: image.OSName},
		{Key: "Progress", Value: image.Progress},
		{Key: "IsSupportCloudinit", Value: strconv.FormatBool(image.IsSupportCloudinit)},
		{Key: "Usage", Value: image.Usage},
		{Key: "ImageVersion", Value: image.ImageVersion},
		{Key: "IsSupportIoOptimized", Value: strconv.FormatBool(image.IsSupportIoOptimized)},
		{Key: "IsSelfShared", Value: image.IsSelfShared},
		{Key: "IsCopied", Value: strconv.FormatBool(image.IsCopied)},
		{Key: "IsSubscribed", Value: strconv.FormatBool(image.IsSubscribed)},
		{Key: "Platform", Value: image.Platform},
		{Key: "Size", Value: strconv.Itoa(image.Size)},

		// {Key: "ImageOwnerId", Value: *image.ImageOwnerId},
		// {Key: "ImageType", Value: *image.ImageType},
		// {Key: "ImageLocation", Value: *image.ImageLocation},
		// {Key: "VirtualizationType", Value: *image.VirtualizationType},
		// {Key: "Public", Value: strconv.FormatBool(*image.Public)},
	}

	// 일부 이미지들은 아래 정보가 없어서 예외 처리 함.
	if !reflect.ValueOf(image.Description).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "Description", Value: image.Description})
	}
	// if !reflect.ValueOf(image.ImageOwnerAlias).IsNil() {
	// 	keyValueList = append(keyValueList, irs.KeyValue{Key: "ImageOwnerAlias", Value: *image.ImageOwnerAlias})
	// }
	// if !reflect.ValueOf(image.RootDeviceName).IsNil() {
	// 	keyValueList = append(keyValueList, irs.KeyValue{Key: "RootDeviceName", Value: *image.RootDeviceName})
	// 	keyValueList = append(keyValueList, irs.KeyValue{Key: "RootDeviceType", Value: *image.RootDeviceType})
	// }
	// if !reflect.ValueOf(image.EnaSupport).IsNil() {
	// 	keyValueList = append(keyValueList, irs.KeyValue{Key: "EnaSupport", Value: strconv.FormatBool(*image.EnaSupport)})
	// }

	imageInfo.KeyValueList = keyValueList

	return imageInfo
}

func (imageHandler *AlibabaImageHandler) GetImage(imageID string) (irs.ImageInfo, error) {
	cblogger.Infof("imageID : ", imageID)

	request := ecs.CreateDescribeImagesRequest()
	request.Scheme = "https"

	// request.Status = "Available"
	// request.ActionType = "*"

	request.ImageId = imageID

	result, err := imageHandler.Client.DescribeImages(request)
	//spew.Dump(result)
	cblogger.Info(result)
	if err != nil {
		// if aerr, ok := err.(errors.Error); ok {
		// 	switch aerr.Code() {
		// 	default:
		// 		cblogger.Error(aerr.Error())
		// 	}
		// } else {
		// 	// Print the error, cast err to awserr.Error to get the Code and
		// 	// Message from an error.
		// 	cblogger.Error(err.Error())
		// }
		cblogger.Errorf("Unable to get Images, %v", err)
		return irs.ImageInfo{}, err
	}

	imageInfo := ExtractImageDescribeInfo(&result.Images.Image[0])

	return imageInfo, nil
}

func (imageHandler *AlibabaImageHandler) DeleteImage(imageID string) (bool, error) {
	cblogger.Infof("DeleteImage : [%s]", imageID)
	// Delete the Image by Id

	request := ecs.CreateDeleteImageRequest()
	request.Scheme = "https"

	//request.ImageId = to.StringPtr(imageID)
	request.ImageId = imageID
	// 추가 옵션 Req
	// request.Force = requests.NewBoolean(true)

	result, err := imageHandler.Client.DeleteImage(request)
	cblogger.Info(result)
	if err != nil {
		// if aerr, ok := err.(errors.Error); ok && aerr.Code() == "InvalidKeyPair.Duplicate" {
		// 	cblogger.Error("Image %q does not exist.", keyPairName)
		// 	return false, err
		// }
		cblogger.Errorf("Unable to delete Image: %s, %v.", imageID, err)
		return false, err
	}

	cblogger.Infof("Successfully deleted %q Image\n", imageID)

	return true, nil
}
