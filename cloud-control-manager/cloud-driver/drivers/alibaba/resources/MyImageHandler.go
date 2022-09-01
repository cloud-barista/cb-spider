package resources

// https://www.alibabacloud.com/help/en/elastic-compute-service/latest/deleteimage

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strconv"
	"strings"
	"time"
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
		return irs.MyImageInfo{}, errStatus
	}
	cblogger.Info("==>생성된 Image[%s]의 현재 상태[%s]", imageIID, curStatus)

	myImageInfo, err := myImageHandler.GetMyImage(imageIID)
	return myImageInfo, err

}

/*
*
owner=sef인 Image목록 조회
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

func (myImageHandler AlibabaMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {

	// 상태체크해서 available일 때 삭제
	imageStatus, err := DescribeImageStatus(myImageHandler.Client, myImageHandler.Region, myImageIID, ALIBABA_IMAGE_STATE_AVAILABLE)
	if err != nil {
		cblogger.Info("DeleteMyImage : status " + imageStatus)
		return false, err
	}

	cblogger.Info("DeleteMyImage : status " + imageStatus)
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

	cblogger.Info("MyImage deleted by requestId " + response.RequestId)
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
