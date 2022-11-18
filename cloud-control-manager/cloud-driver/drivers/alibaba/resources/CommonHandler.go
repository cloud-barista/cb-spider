package resources

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

//// Alibaba API 1:1로 대응

/*
*
디스크 목록 조회
*/
func DescribeDisks(client *ecs.Client, regionInfo idrv.RegionInfo, instanceIID irs.IID, diskIIDs []irs.IID) ([]ecs.Disk, error) {
	regionID := regionInfo.Region

	request := ecs.CreateDescribeDisksRequest()
	request.Scheme = "https"
	request.RegionId = regionID

	if CBPageOn {
		request.PageNumber = requests.NewInteger(CBPageNumber)
		request.PageSize = requests.NewInteger(CBPageSize)
	}

	if instanceIID != (irs.IID{}) {
		request.InstanceId = instanceIID.SystemId
	}

	var diskIIDList []string
	for _, diskIID := range diskIIDs {
		diskIIDList = append(diskIIDList, diskIID.SystemId)
	}
	diskJson, err := json.Marshal(diskIIDList)
	if err != nil {

	}
	if len(diskIIDList) > 0 {
		request.DiskIds = string(diskJson)
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   regionInfo.Zone,
		ResourceType: call.DISK,
		ResourceName: "ListDisk()",
		CloudOSAPI:   "DescribeDisks()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()

	var totalCount = 0
	curPage := CBPageNumber
	var resultDiskList []ecs.Disk
	for {
		result, err := client.DescribeDisks(request)
		callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
		//spew.Dump(result) //출력 정보가 너무 많아서 생략
		if err != nil {
			callLogInfo.ErrorMSG = err.Error()
			callogger.Error(call.String(callLogInfo))

			cblogger.Errorf("Unable to get Disks, %v", err)
			return resultDiskList, err
		}
		callogger.Info(call.String(callLogInfo))

		resultDiskList = append(resultDiskList, result.Disks.Disk...)
		if CBPageOn {
			totalCount = len(resultDiskList)
			cblogger.Infof("CSP 전체 Disk 갯수 : [%d] - 현재 페이지:[%d] - 누적 결과 개수:[%d]", result.TotalCount, curPage, totalCount)
			if totalCount >= result.TotalCount {
				break
			}
			curPage++
			request.PageNumber = requests.NewInteger(curPage)
		} else {
			break
		}
	}
	cblogger.Info(cblogger.Level.String())
	if cblogger.Level.String() == "debug" {
		spew.Dump(resultDiskList)
	}
	return resultDiskList, nil
}

/*
*
DiskID로 1개 Disk의 정보 조회
*/
func DescribeDiskByDiskId(client *ecs.Client, regionInfo idrv.RegionInfo, diskIID irs.IID) (ecs.Disk, error) {

	var diskIIDList []irs.IID
	diskIIDList = append(diskIIDList, diskIID)

	diskList, err := DescribeDisks(client, regionInfo, irs.IID{}, diskIIDList)
	if err != nil {
		return ecs.Disk{}, err
	}

	if len(diskList) != 1 {
		return ecs.Disk{}, errors.New("search failed")
	}

	return diskList[0], nil
}

/*
*
InstanceID로 1개 Disk의 정보 조회
*/
func DescribeDisksByInstanceId(client *ecs.Client, regionInfo idrv.RegionInfo, instanceIID irs.IID) ([]ecs.Disk, error) {

	diskList, err := DescribeDisks(client, regionInfo, instanceIID, nil)
	//if err != nil {
	//	return nil, err
	//}
	//
	//if len(diskList) != 1 {
	//	return nil, errors.New("search failed")
	//}

	return diskList, err
}

/*
*
해당 리소스가 사용가능한지 조회

	https://help.aliyun.com/document_detail/66186.html?spm=api-workbench.Troubleshoot.0.0.43651e0folUpip#doc-api-Ecs-DescribeAvailableResource
	https://next.api.alibabacloud.com/api/Ecs/2014-05-26/DescribeAvailableResource?lang=GO&params={}

필수 parameter

	RegionId
	DestinationResource : "Zone", "IoOptimized", "InstanceType", "SystemDisk", "DataDisk", "Network", "ddh"

결과 : AvailableZone 값이 들어있음. 배열형태임.

	비정상 : requestID만 반환. ex) {"RequestId":"7F2E6252-7FF6-31AF-9067-1EECF1B6B3FA"}
	정상 : requestID 외에 Available
		ex) {"RequestId":"7F2E6252-7FF6-31AF-9067-1EECF1B6B3FA","AvailableZones":{"AvailableZone":[{"Status":"Available","StatusCategory":"WithStock","ZoneId":"ap-southeast-1b","AvailableResources":{"AvailableResource":[{"Type":"DataDisk","SupportedResources":{"SupportedResource":[{"Status":"Available","Min":20,"Max":32768,"Value":"cloud_efficiency","Unit":"GiB"}]}}]},"RegionId":"ap-southeast-1"}]}}
*/
func DescribeAvailableResource(client *ecs.Client, regionId string, zoneId string, resourceType string, destinationResource string, categoryValue string) (ecs.AvailableZonesInDescribeAvailableResource, error) {

	request := ecs.CreateDescribeAvailableResourceRequest()
	request.Scheme = "https"

	request.RegionId = regionId
	if zoneId != "" {
		request.ZoneId = zoneId
	}

	request.ResourceType = resourceType

	request.DestinationResource = destinationResource
	switch destinationResource {

	case "Zone":
		request.ZoneId = categoryValue
	case "IoOptimized":
		request.IoOptimized = categoryValue
	case "InstanceType":
		request.InstanceType = categoryValue
	case "SystemDisk":
		request.SystemDiskCategory = categoryValue
	case "DataDisk":
		request.DataDiskCategory = categoryValue
	case "Network":
		request.NetworkCategory = categoryValue
	case "ddh":
		request.DedicatedHostId = categoryValue
	}
	//request.DataDiskCategory = "cloud"
	//spew.Dump(request)
	result, err := client.DescribeAvailableResource(request)
	cblogger.Info(result)
	if err != nil {
		cblogger.Errorf("DescribeAvailableResource %v.", err)
	}
	//spew.Dump(result)

	metaValue := reflect.ValueOf(result).Elem()
	fieldAvailableZones := metaValue.FieldByName("AvailableZones")
	if fieldAvailableZones == (reflect.Value{}) {
		cblogger.Errorf("Field not exist")
		cblogger.Errorf("Not available in this region")
		return ecs.AvailableZonesInDescribeAvailableResource{}, errors.New("Not available in this region")
	}

	return result.AvailableZones, nil
}

/*
*
Instance에 Disk Attach
한번에 1개씩.
*/
func AttachDisk(client *ecs.Client, regionInfo idrv.RegionInfo, ownerVM irs.IID, diskIID irs.IID) error {

	cblogger.Infof("AttachDisk : [%s]", diskIID.SystemId)

	request := ecs.CreateAttachDiskRequest()
	request.Scheme = "https"

	request.DiskId = diskIID.SystemId
	request.InstanceId = ownerVM.SystemId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   regionInfo.Zone,
		ResourceType: call.DISK,
		ResourceName: diskIID.SystemId,
		CloudOSAPI:   "AttachDisk()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	result, err := client.AttachDisk(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to attach Disk: %s, %v.", diskIID.SystemId, err)
		return err
	}
	callogger.Info(call.String(callLogInfo))
	return nil
}

/*
*
Instance 목록 조회

	iid를 parameter로 주면 해당 iid들만 조회.
*/
func DescribeInstances(client *ecs.Client, regionInfo idrv.RegionInfo, vmIIDs []irs.IID) ([]ecs.Instance, error) {
	request := ecs.CreateDescribeInstancesRequest()
	request.Scheme = "https"

	var instanceIdList []string
	for _, instanceIID := range vmIIDs {
		instanceIdList = append(instanceIdList, instanceIID.SystemId)
	}
	if len(instanceIdList) > 0 {
		vmsJson, err := json.Marshal(instanceIdList)
		if err != nil {

		}
		request.InstanceIds = string(vmsJson)
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   regionInfo.Zone,
		ResourceType: call.VM,
		ResourceName: "ListVM()",
		CloudOSAPI:   "DescribeInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	response, err := client.DescribeInstances(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))

	return response.Instances.Instance, err
}

/*
*
인스턴스 1개만 조회
*/
func DescribeInstanceById(client *ecs.Client, regionInfo idrv.RegionInfo, vmIID irs.IID) (ecs.Instance, error) {

	var vmIIDs []irs.IID
	vmIIDs = append(vmIIDs, vmIID)
	response, err := DescribeInstances(client, regionInfo, vmIIDs)
	if err != nil {
		return ecs.Instance{}, err
	}
	if len(response) < 1 {
		return ecs.Instance{}, errors.New("Notfound: '" + vmIID.SystemId + "' VM Not found")
	}

	return response[0], nil
}

/*
*
Image 목록 조회

imageOwnerAlias 종류 : self, system, others, marketplace
myimage 는 imageOwnerAlias = self 로 isMyImage로 구분
향후 system, others, marketplace 등이 추가되면 해당 구분자 변경필요.
*/
func DescribeImages(client *ecs.Client, regionInfo idrv.RegionInfo, imageIIDs []irs.IID, isMyImage bool) ([]ecs.Image, error) {
	request := ecs.CreateDescribeImagesRequest()
	request.Scheme = "https"
	// 필수 Req Name
	request.RegionId = regionInfo.Region

	var imageIIDList []string
	for _, imageIID := range imageIIDs {
		imageIIDList = append(imageIIDList, imageIID.SystemId)
	}

	if len(imageIIDList) > 0 {
		request.ImageId = strings.Join(imageIIDList, ",")
	}

	// MyImage 여부
	if isMyImage {
		request.ImageOwnerAlias = "self"
	}
	//spew.Dump(request)
	result, err := client.DescribeImages(request)
	if err != nil {
		return nil, err
	}

	//spew.Dump(result)
	return result.Images.Image, nil
}

/*
*
ImageID로 1개 Image의 정보 조회
*/
func DescribeImageByImageId(client *ecs.Client, regionInfo idrv.RegionInfo, imageIID irs.IID, isMyImage bool) (ecs.Image, error) {

	var imageIIDList []irs.IID
	imageIIDList = append(imageIIDList, imageIID)

	imageList, err := DescribeImages(client, regionInfo, imageIIDList, isMyImage)
	if err != nil {
		return ecs.Image{}, err
	}

	//if len(imageList) != 1 {
	//	return ecs.Image{}, errors.New("search failed")
	//}

	if len(imageList) == 0 {
		return ecs.Image{}, errors.New("no result")
	} else if len(imageList) > 1 {
		return ecs.Image{}, errors.New("search failed. too many results")
	}

	return imageList[0], nil
}

/*
*
이미지의 상태 조회
조회하고 싶은 상태값을 줘야 정상적으로 조회가 됨.(default = available )
그래서 request 객체에 status를 set하고 DescribeImage를 직접호출함.
*/
func DescribeImageStatus(client *ecs.Client, regionInfo idrv.RegionInfo, myImageIID irs.IID, aliImageStatus string) (string, error) {
	request := ecs.CreateDescribeImagesRequest()
	request.Scheme = "https"
	// 필수 Req Name
	request.RegionId = regionInfo.Region

	request.ImageId = myImageIID.SystemId

	//if aliImageStatus != "" {
	//	request.Status = aliImageStatus
	//}
	if aliImageStatus == "" || aliImageStatus == "ALL" {
		// 없는 상태를 넣으면 InvalidParameter오류가 나므로 Spider에서 정의한 Error 상태는 제외 함.
		request.Status = ALIBABA_IMAGE_STATE_CREATING + "," + ALIBABA_IMAGE_STATE_WAITING + "," + ALIBABA_IMAGE_STATE_AVAILABLE + "," + ALIBABA_IMAGE_STATE_UNAVAILABLE + "," + ALIBABA_IMAGE_STATE_CREATEFAILED + "," + ALIBABA_IMAGE_STATE_DEPRECATED
	} else {
		request.Status = aliImageStatus
	}

	result, err := client.DescribeImages(request)
	if err != nil {
		return ALIBABA_IMAGE_STATE_ERROR, err
	}

	// 받아온 결과가 없으면 에러
	if len(result.Images.Image) == 0 { // return을 empty string 으로 할까?
		return ALIBABA_IMAGE_STATE_ERROR, errors.New("no result")
	}
	//spew.Dump(result)
	return result.Images.Image[0].Status, nil
}

/*
*
이미지의 크기 조회
*/
func DescribeImageSize(client *ecs.Client, regionInfo idrv.RegionInfo, myImageIID irs.IID) (int64, error) {
	result, err := DescribeImageByImageId(client, regionInfo, myImageIID, false)
	if err != nil {
		return -1, err
	}

	imageSize := int64(result.Size)
	return imageSize, nil
}

/*
*
상태조회 처리 :
DescribeImages에서 status값이 필수임. 없는 경우 default available이므로 다른 상태값이 조회되게 하려면
콤마(,)를 구분자로 여러개 넣을 수 있음.

요청시 상태 -> 원하는 상태 또는 실패상태를 Return
*/
func WaitForImageStatus(client *ecs.Client, regionInfo idrv.RegionInfo, imageIID irs.IID, requestStatus string) (irs.MyImageStatus, error) {
	//cblogger.Info("======> MyImage 생성 직후에는 정보 조회가 안되기 때문에 원하는 상태(ex.Available) 될 때까지 대기함.")

	curRetryCnt := 0
	maxRetryCnt := 600

	targetStatus := ""
	failStatus := ALIBABA_IMAGE_STATE_ERROR
	switch requestStatus {
	case ALIBABA_IMAGE_STATE_CREATING: // 생성일 때
		targetStatus = ALIBABA_IMAGE_STATE_AVAILABLE
		failStatus = ALIBABA_IMAGE_STATE_CREATEFAILED
	default:
		targetStatus = ALIBABA_IMAGE_STATE_AVAILABLE
		//failStatus = ALIBABA_IMAGE_STATE_UNAVAILABLE

	}

	resultImageState := ""
	for {
		aliImageState, err := DescribeImageStatus(client, regionInfo, imageIID, "ALL") // 특정 image의 상태조회(특정상태별로 조회하므로 모든 status를 조회하도록)
		if err != nil {
			cblogger.Error(err.Error())
			return irs.MyImageStatus(failStatus), err
		}
		resultImageState = aliImageState
		if aliImageState == targetStatus || aliImageState == failStatus {
			break
		}

		curRetryCnt++
		cblogger.Errorf("MyImage의 상태가 [%s]이 아니라서 1초 대기후 조회합니다. 현재 [%s]", targetStatus, aliImageState)
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf("장시간(%d 초) 대기해도 MyImage의 Status 값이 [%s]으로 변경되지 않아서 강제로 중단합니다.", maxRetryCnt, targetStatus)
			return irs.MyImageStatus(failStatus), errors.New("장시간 기다렸으나 생성된 MyImage의 상태가 [" + string(targetStatus) + "]으로 바뀌지 않아서 중단 합니다.")
		}
	}
	return irs.MyImageStatus(resultImageState), nil

	// 현재상태가 Target상태인지(normal) 조회
	//resultImageState, err := DescribeImageStatus(client, regionInfo, imageIID, targetStatus)
	//if err != nil {
	//	// 조회된 상태가 없으면 비정상 상태인지 조회
	//	failImageState, failerr := DescribeImageStatus(client, regionInfo, imageIID, failStatus)
	//	if failerr != nil {
	//		return irs.MyImageStatus("Failed"), failerr
	//	}
	//	return irs.MyImageStatus(failImageState), nil
	//}
	//return irs.MyImageStatus(resultImageState), nil

	//for {
	//	aliImageState, err := DescribeImageStatus(client, regionInfo, imageIID, waitStatus)
	//	if err != nil {
	//		cblogger.Error(err.Error())
	//	}
	//
	//	//cblogger.Info("===>image Status : ", image.Status)
	//	//imageStatus := convertImageStateToMyImageStatus(&image.Status)
	//
	//	if targetStatus == aliImageState {
	//		cblogger.Infof("===>Image 상태가 [%s]라서 대기를 중단합니다.", targetStatus)
	//		break
	//	}
	//
	//	curRetryCnt++
	//	cblogger.Errorf("Image 상태가 [%s]이 아니라서 1초 대기후 조회합니다. 현재 [%s]", targetStatus, aliImageState)
	//	time.Sleep(time.Second * 1)
	//	if curRetryCnt > maxRetryCnt {
	//		cblogger.Errorf("장시간(%d 초) 대기해도 Image의 Status 값이 [%s]으로 변경되지 않아서 강제로 중단합니다.", maxRetryCnt, targetStatus)
	//		return irs.MyImageStatus("Failed"), errors.New("장시간 기다렸으나 생성된 Image의 상태가 [" + string(targetStatus) + "]으로 바뀌지 않아서 중단 합니다.")
	//	}
	//	//} else {
	//	//break
	//	//}
	//}
	//
	//// 정상 status인지 확인
	//aliImageState, err := DescribeImageStatus(client, regionInfo, imageIID, targetStatus)
	//if err != nil {
	//	// 비정상 status인지 확인
	//	abnormalAliImageState, abnormalAliStatusErr := DescribeImageStatus(client, regionInfo, imageIID, failStatus)
	//	if abnormalAliStatusErr != nil {
	//		return irs.MyImageStatus("Failed"), errors.New("장시간 기다렸으나 생성된 Image의 상태가 [" + string(targetStatus) + "]으로 바뀌지 않아서 중단 합니다.")
	//	}
	//	return irs.MyImageStatus(abnormalAliImageState), nil
	//}
	//return irs.MyImageStatus(aliImageState), nil
}

// Image의 OS Type을 string으로 반환
func DescribeImageOsType(client *ecs.Client, regionInfo idrv.RegionInfo, imageIID irs.IID, isMyImage bool) (string, error) {

	result, err := DescribeImageByImageId(client, regionInfo, imageIID, isMyImage)

	if err != nil {
		return "", err
	}

	osType := GetOsType(result)
	return osType, nil
}

// Image에서 OSTYPE 만 추출
func GetOsType(ecsImage ecs.Image) string {
	osType := ecsImage.OSType //"OSType": "windows"
	cblogger.Info("osType = ", osType)
	return osType
}

// Image에서 상태만 추출
func GetImageStatus(ecsImage ecs.Image) string {
	return ecsImage.Status
}

// Image에서 SnapShotID 목록 추출
func GetSnapShotIdList(ecsImage ecs.Image) []string {
	var snapShotIdList []string

	devices := ecsImage.DiskDeviceMappings
	for _, diskDeviceMapping := range devices.DiskDeviceMapping {
		snapShotIdList = append(snapShotIdList, diskDeviceMapping.SnapshotId)
	}

	return snapShotIdList
}
