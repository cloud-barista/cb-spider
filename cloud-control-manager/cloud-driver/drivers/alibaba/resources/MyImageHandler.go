package resources

// https://www.alibabacloud.com/help/en/elastic-compute-service/latest/deleteimage

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaMyImageHandler struct {
	Region idrv.RegionInfo
	Client *ecs.Client
}

const (
	ALIBABA_IMAGE_STATE_CREATING     = "Creating"
	ALIBABA_IMAGE_STATE_WAITING      = "Waiting"
	ALIBABA_IMAGE_STATE_AVAILABLE    = "Available"
	ALIBABA_IMAGE_STATE_UNAVAILABLE  = "Unavailable"
	ALIBABA_IMAGE_STATE_CREATEFAILED = "Createfailed"
	ALIBABA_IMAGE_STATE_DEPRECATED   = "Deprecated"
	ALIBABA_IMAGE_STATE_ERROR        = "Error"

	RESOURCE_TYPE_MYIMAGE = "image"
	IMAGE_TAG_DEFAULT     = "Name"
	IMAGE_TAG_SOURCE_VM   = "CB-VMSNAPSHOT-SOURCEVM-ID"
)

func (myImageHandler AlibabaMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {

	request := ecs.CreateCreateImageRequest()
	request.Scheme = "https"
	// 필수 Req Name
	request.RegionId = myImageHandler.Region.Region
	request.InstanceId = snapshotReqInfo.SourceVM.SystemId
	request.ImageName = snapshotReqInfo.IId.NameId

	// TAG에 연관 instanceID set 할 것
	request.Tag = &[]ecs.CreateImageTag{ // Default Hidden Tags Info
		{
			Key:   CBMetaDefaultTagName,  // "cbCat",
			Value: CBMetaDefaultTagValue, // "cbAlibaba",
		},
		{
			Key:   IMAGE_TAG_DEFAULT, // "Name",
			Value: snapshotReqInfo.IId.NameId,
		},
		{
			Key:   IMAGE_TAG_SOURCE_VM,
			Value: snapshotReqInfo.SourceVM.SystemId,
		},
	}

	//spew.Dump(request)
	result, err := myImageHandler.Client.CreateImage(request)
	if err != nil {
		return irs.MyImageInfo{}, err
	}

	imageIID := irs.IID{SystemId: result.ImageId}
	// ImageId 로 해당 Image의 Status 조회

	// 현재요청 -> 예상상태, 오류상태.
	curStatus, errStatus := WaitForImageStatus(myImageHandler.Client, myImageHandler.Region, imageIID, "")
	if errStatus != nil {
		cblogger.Error(errStatus)
		_, deleteErr := myImageHandler.DeleteMyImage(imageIID)
		if deleteErr != nil { // 중간 생성 자원 삭제 실패
			return irs.MyImageInfo{}, deleteErr
		}
		return irs.MyImageInfo{}, errStatus
	}
	cblogger.Info("==>생성된 Image[%s]의 현재 상태[%s]", imageIID, curStatus)

	myImageInfo, err := myImageHandler.GetMyImage(imageIID)
	return myImageInfo, err

}

/*
*
owner=self인 Image목록 조회
공통으로 DescribeImages를 사용하기 때문에 구분으로 isMyImage = true 로 전송 필요
*/
func (myImageHandler AlibabaMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {

	result, err := DescribeImages(myImageHandler.Client, myImageHandler.Region, nil, true)
	if err != nil {
		return nil, err
	}

	var myImageInfoList []*irs.MyImageInfo
	for _, image := range result {
		myImageInfo, err := ExtractMyImageDescribeInfo(&image)
		if err != nil {

		} else {
			myImageInfoList = append(myImageInfoList, &myImageInfo)
		}
	}
	//spew.Dump(myImageInfoList)
	return myImageInfoList, err
}

func (myImageHandler AlibabaMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {

	result, err := DescribeImageByImageId(myImageHandler.Client, myImageHandler.Region, myImageIID, true)
	if err != nil {
		return irs.MyImageInfo{}, err
	}

	myImageInfo, err := ExtractMyImageDescribeInfo(&result)
	return myImageInfo, err
}

// MyImage 삭제
// Image 삭제 후 Snapshot 삭제 가능 :
// - A snapshot that has been used to create custom images cannot be deleted. The snapshot can be deleted only after the created custom images are deleted
// Image를 Instance가 사용중이면 삭제 불가. image로 작업중이면 삭제 불가
// A custom image cannot be deleted in the following scenarios:
//
//	The image is being imported. You can go to the Task Logs page in the Elastic Compute Service (ECS) console to cancel the image import task. After the image import task is canceled, you can delete the image. For more information, see Import custom images.
//	The image is being exported. You can go to the Task Logs page in the ECS console to cancel the image export task. After the image export task is canceled, you can delete the image. For more information, see Export a custom image.
//	The image is being used by ECS instances
func (myImageHandler AlibabaMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {

	// 상태체크해서 available일 때 삭제
	// imageStatus, err := DescribeImageStatus(myImageHandler.Client, myImageHandler.Region, myImageIID, ALIBABA_IMAGE_STATE_AVAILABLE)
	// if err != nil {
	// 	cblogger.Info("DeleteMyImage : status " + imageStatus)
	// 	return false, err
	// }

	ecsImage, err := DescribeImageByImageId(myImageHandler.Client, myImageHandler.Region, myImageIID, true)
	if err != nil {
		return false, err
	}

	imageStatus := GetImageStatus(ecsImage)

	if imageStatus == "" {

	}

	snapShotIdList := GetSnapShotIdList(ecsImage)

	// cblogger.Info("DeleteMyImage : status " + imageStatus)
	request := ecs.CreateDeleteImageRequest()
	request.Scheme = "https"
	// 필수 Req Name
	request.RegionId = myImageHandler.Region.Region
	request.ImageId = myImageIID.SystemId

	//spew.Dump(request)
	response, err := myImageHandler.Client.DeleteImage(request)
	if err != nil {
		return false, err
	}

	// 이미지가 삭제될 때까지 대기
	curRetryCnt := 0
	maxRetryCnt := 600
	for {
		aliImage, err := DescribeImages(myImageHandler.Client, myImageHandler.Region, []irs.IID{myImageIID}, true)
		if err != nil {
			cblogger.Error(err.Error())
			break
		}

		if len(aliImage) == 0 { // return을 empty string 으로 할까?
			break
		}

		aliImageState := ""
		if !reflect.ValueOf(aliImage[0]).IsNil() {
			aliImageState = aliImage[0].Status
		}

		curRetryCnt++
		cblogger.Errorf("MyImage의 상태 1초 대기후 조회합니다. 현재 [%s]", aliImageState)
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf("장시간(%d 초) 대기해도 MyImage의 Status 값이 [%s]으로 변경되지 않아서 강제로 중단합니다.", maxRetryCnt)
			return false, errors.New("장시간 기다렸으나 생성된 MyImage의 상태가 [" + string(aliImageState) + "]으로 바뀌지 않아서 중단 합니다.")
		}
	}
	cblogger.Info("MyImage deleted. requestId =" + response.RequestId)

	// myImage에 연결된 snapshot들도 삭제
	for _, snapShotId := range snapShotIdList {
		result, err := myImageHandler.DeleteSnapshotBySnapshotID(irs.IID{SystemId: snapShotId})
		if err != nil {
			cblogger.Info("Deleting SnapShot failed" + response.RequestId)
		}
		cblogger.Info("SnapShot deleted ", result, snapShotId)
	}

	//spew.Dump(response)
	return true, err
}

func ExtractMyImageDescribeInfo(aliMyImage *ecs.Image) (irs.MyImageInfo, error) {
	returnMyImageInfo := irs.MyImageInfo{}

	//IId	IID 	// {NameId, SystemId}
	//
	//SourceVM IID
	//
	//Status 		MyImageStatus  // Creating | Available | Deleting
	//
	//CreatedTime	time.Time
	//KeyValueList 	[]KeyValue

	returnMyImageInfo.IId.NameId = aliMyImage.ImageName
	returnMyImageInfo.IId.SystemId = aliMyImage.ImageId
	returnMyImageInfo.Status = convertImageStateToMyImageStatus(&aliMyImage.Status)

	sourceVMTag := ""
	for _, tag := range aliMyImage.Tags.Tag {
		//if strings.EqualFold(*tag.Key, IMAGE_TAG_DEFAULT) {
		//	nameTagValue = *tag.Value
		//}

		//if strings.EqualFold(tag.Key, IMAGE_TAG_SOURCE_VM) {
		if strings.EqualFold(tag.TagKey, IMAGE_TAG_SOURCE_VM) {
			//sourceVMTag = tag.Value
			sourceVMTag = tag.TagValue
		}
	}

	returnMyImageInfo.SourceVM.SystemId = sourceVMTag

	createdTime, _ := time.Parse(
		time.RFC3339,
		aliMyImage.CreationTime) // RFC3339형태이므로 해당 시간으로 다시 생성
	returnMyImageInfo.CreatedTime = createdTime

	keyValueList := []irs.KeyValue{}
	keyValueList = append(keyValueList, irs.KeyValue{Key: "ImageOwnerAlias", Value: aliMyImage.ImageOwnerAlias})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "IsSelfShared", Value: aliMyImage.IsSelfShared})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "Description", Value: aliMyImage.Description})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "Platform", Value: aliMyImage.Platform})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "ResourceGroupId", Value: aliMyImage.ResourceGroupId})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "Size", Value: strconv.Itoa(aliMyImage.Size)})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "IsSubscribed", Value: strconv.FormatBool(aliMyImage.IsSubscribed)})
	//keyValueList = append(keyValueList, irs.KeyValue{Key: "architecture", Value: strconv.FormatBool(aliMyImage.)})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "OSName", Value: aliMyImage.OSName})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "OSNameEn", Value: aliMyImage.OSNameEn})

	keyValueList = append(keyValueList, irs.KeyValue{Key: "Progress", Value: aliMyImage.Progress})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "Usage", Value: aliMyImage.Usage})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "Architecture", Value: aliMyImage.Architecture})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "IsCopied", Value: strconv.FormatBool(aliMyImage.IsCopied)})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "IsSupportCloudinit", Value: strconv.FormatBool(aliMyImage.IsSupportCloudinit)})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "ImageVersion", Value: aliMyImage.ImageVersion})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "OSType", Value: aliMyImage.OSType})

	returnMyImageInfo.KeyValueList = keyValueList
	return returnMyImageInfo, nil
}

// Alibaba Image state 를 CB-SPIDER MyImage 의 statuf 로 변환
func convertImageStateToMyImageStatus(aliImageState *string) irs.MyImageStatus {
	var returnStatus irs.MyImageStatus

	switch *aliImageState {
	case ALIBABA_IMAGE_STATE_CREATING:
		returnStatus = irs.MyImageUnavailable
	case ALIBABA_IMAGE_STATE_AVAILABLE:
		returnStatus = irs.MyImageAvailable // 이것만 available 나머지는 unavailable
	case ALIBABA_IMAGE_STATE_WAITING:
		returnStatus = irs.MyImageUnavailable
	case ALIBABA_IMAGE_STATE_UNAVAILABLE:
		returnStatus = irs.MyImageUnavailable
	case ALIBABA_IMAGE_STATE_CREATEFAILED:
		returnStatus = irs.MyImageUnavailable
	case ALIBABA_IMAGE_STATE_DEPRECATED:
		returnStatus = irs.MyImageUnavailable
	case ALIBABA_IMAGE_STATE_ERROR:
		returnStatus = irs.MyImageUnavailable
	}
	return returnStatus
}

// MyImage 의 window 여부 return
func (myImageHandler AlibabaMyImageHandler) CheckWindowsImage(myImageIID irs.IID) (bool, error) {
	isWindows := false
	isMyImage := true

	osType, err := DescribeImageOsType(myImageHandler.Client, myImageHandler.Region, myImageIID, isMyImage)
	if err != nil {
		return isWindows, err
	}

	if osType == "windows" {
		isWindows = true
	}

	return isWindows, nil
}

// MyImage에 대한 snap 삭제
// If the specified snapshot ID does not exist, the request is ignored.
// A snapshot that has been used to create custom images cannot be deleted. The snapshot can be deleted only after the created custom images are deleted
func (myImageHandler *AlibabaMyImageHandler) DeleteSnapshotBySnapshotID(snapshotIID irs.IID) (bool, error) {

	request := ecs.CreateDeleteSnapshotRequest()
	request.Scheme = "https"

	request.SnapshotId = snapshotIID.SystemId

	response, err := myImageHandler.Client.DeleteSnapshot(request)
	if err != nil {
		return false, err
	}

	//requestId := response.RequestId

	// 삭제 대기 // API Gateway 이용 requestID에 대한 statusCode 확인 로직 : https://www.alibabacloud.com/help/en/log-service/latest/api-gateway
	cblogger.Info("Snapshot deleted. requestId =" + response.RequestId)

	// snapshot disk도 삭제

	//spew.Dump(response)

	return true, err
}
