package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"

	cs "github.com/alibabacloud-go/cs-20151215/v4/client" // cs  : container service
	bssopenapi "github.com/aliyun/alibaba-cloud-sdk-go/services/bssopenapi"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs" // ecs : elastic compute service
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc" // vpc
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
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
		//cblogger.Debug(result) //출력 정보가 너무 많아서 생략
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
			cblogger.Infof("Total number of disks across CSP: [%d] - Current page: [%d] - Accumulated result count: [%d]", result.TotalCount, curPage, totalCount)
			if totalCount >= result.TotalCount {
				break
			}
			curPage++
			request.PageNumber = requests.NewInteger(curPage)
		} else {
			break
		}
	}

	cblogger.Debug(resultDiskList)

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
	//cblogger.Debug(request)
	result, err := client.DescribeAvailableResource(request)
	cblogger.Debug(result)
	if err != nil {
		cblogger.Errorf("DescribeAvailableResource %v.", err)
	}
	//cblogger.Debug(result)

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
리전별로 AvailableResource 중에서 InstanceType 별로 사용 가능한 SystemDisk 목록 조회
*/
func DescribeAvailableSystemDisksByInstanceType(client *ecs.Client, regionId string, zoneId string, instanceChargeType string, destinationResource string, insTanceType string) (ecs.AvailableZonesInDescribeAvailableResource, error) {
	request := ecs.CreateDescribeAvailableResourceRequest()
	request.Scheme = "https"

	request.RegionId = regionId
	request.ZoneId = zoneId

	request.DestinationResource = destinationResource
	request.InstanceChargeType = instanceChargeType
	request.InstanceType = insTanceType

	result, err := client.DescribeAvailableResource(request)
	cblogger.Debug(result)
	if err != nil {
		cblogger.Errorf("DescribeAvailableResource %v.", err)
	}

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
	cblogger.Debug(result)
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
	if imageIIDs != nil {
		for _, imageIID := range imageIIDs {
			imageIIDList = append(imageIIDList, imageIID.SystemId)
		}

		if len(imageIIDList) > 0 {
			request.ImageId = strings.Join(imageIIDList, ",")
		}
	}
	// MyImage 여부
	if isMyImage {
		request.ImageOwnerAlias = "self"
	}

	//cblogger.Debug(request)
	result, err := client.DescribeImages(request)
	if err != nil {
		return nil, err
	}

	//cblogger.Debug(result)
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
		return ecs.Image{}, errors.New("no result with request image IID(NameId/SystemId) : " + imageIID.NameId + "/" + imageIID.SystemId)
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
	//cblogger.Debug(result)
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
		cblogger.Debugf("Since the state of MyImage is not [%s], we will wait for 1 second and then check again. The current state is [%s].", targetStatus, aliImageState)
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf("Even after waiting for a long time (%d seconds), the status of MyImage did not change to [%s], so we are forcibly terminating it.", maxRetryCnt, targetStatus)
			return irs.MyImageStatus(failStatus), errors.New("After waiting for a long time, the status of the created MyImage did not change to [" + string(targetStatus) + "], so we are terminating it.")
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

// Region status : available, soldOut
// deprecated : Region에 대한 Status는 따로 관리하지 않음.
// func GetRegionStatus(status string) irs.ZoneStatus {
// 	if status == "available" || status == "Available" {
// 		return irs.ZoneAvailable
// 	} else if status == "soldOut" || status == "soldout" {
// 		return irs.ZoneUnavailable
// 	} else {
// 		return irs.NotSupported
// 	}
// }

// Alibaba에서 Zone에 대한 status는 관리하고 있지 않음
func GetZoneStatus(status string) irs.ZoneStatus {
	return irs.NotSupported
}

func DescribeRegions(client *ecs.Client) (*ecs.DescribeRegionsResponse, error) {
	request := ecs.CreateDescribeRegionsRequest()
	request.AcceptLanguage = "en-US" // Only Chinese (zh-CN : default), English (en-US), and Japanese (ja) are allowed

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   "",
		ResourceType: call.REGIONZONE,
		ResourceName: "Regions",
		CloudOSAPI:   "ListRegionZone()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	result, err := client.DescribeRegions(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	// request := ecs.CreateDescribeRegionsRequest()
	// request.AcceptLanguage = "en-US" // Only Chinese (zh-CN : default), English (en-US), and Japanese (ja) are allowed

	// callogger := call.GetLogger("HISCALL")
	// callLogInfo := call.CLOUDLOGSCHEMA{
	// 	CloudOS:      call.ALIBABA,
	// 	RegionZone:   "",
	// 	ResourceType: call.REGIONZONE,
	// 	ResourceName: "",
	// 	CloudOSAPI:   "ListRegions()",
	// 	ElapsedTime:  "",
	// 	ErrorMSG:     "",
	// }

	// callLogStart := call.Start()
	// result, err := client.DescribeRegions(request)
	// callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	// if err != nil {
	// 	callLogInfo.ErrorMSG = err.Error()
	// 	callogger.Error(call.String(callLogInfo))
	// 	return nil, err
	// }
	// callogger.Info(call.String(callLogInfo))
	return result, nil
}

func DescribeZonesByRegion(client *ecs.Client, regionId string) (*ecs.DescribeZonesResponse, error) {
	request := ecs.CreateDescribeZonesRequest()
	request.AcceptLanguage = "en-US" // Only Chinese (zh-CN : default), English (en-US), and Japanese (ja) are allowed
	request.RegionId = regionId

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   "",
		ResourceType: call.REGIONZONE,
		ResourceName: "",
		CloudOSAPI:   "ListZones()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	result, err := client.DescribeZones(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))
	return result, nil
}

// Alibaba 가용한 모든 서비스 호출
func QueryProductList(bssClient *bssopenapi.Client) (*bssopenapi.QueryProductListResponse, error) {
	request := bssopenapi.CreateQueryProductListRequest()
	request.Scheme = "https"
	// request.Language = "en" //
	// request.Lang = "en"
	request.QueryTotalCount = requests.Boolean("true") // 전체 서비스 카운트 리턴 옵션
	request.PageNum = requests.NewInteger(1)
	request.PageSize = requests.NewInteger(1) // 전체 서비스 카운트를 얻어오기 위해 PageNum과 PageSize를 1로 설정하여 QueryTotalCount 획득

	responseTotalcount, err := bssClient.QueryProductList(request)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	// QueryTotalCount 설정. 23.12.18 : 123개
	request.PageSize = requests.NewInteger(responseTotalcount.Data.TotalCount)
	productListresponse, err := bssClient.QueryProductList(request)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	return productListresponse, nil
}

func AddEcsTags(client *ecs.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (*responses.CommonResponse, error) {
	apiName := "AddTags"
	regionID := regionInfo.Region

	cblogger.Info("Start Add EcsTag : ", tag)
	hiscallInfo := GetCallLogScheme(regionInfo, call.TAG, resIID.NameId, apiName)

	// 생성된 Tag 정보 획득 후, Tag 정보 리턴
	//tagInfo := irs.TagInfo{}

	// 지원하는 resource Type인지 확인
	alibabaResourceType, err := GetAlibabaResourceType(resType)
	if err != nil {
		return nil, err
	}

	queryParams := map[string]string{}
	queryParams["RegionId"] = regionID
	queryParams["ResourceType"] = alibabaResourceType
	queryParams["ResourceId"] = resIID.SystemId
	queryParams["Tag.1.Key"] = tag.Key
	queryParams["Tag.1.Value"] = tag.Value

	start := call.Start()
	response, err := CallEcsRequest(resType, client, regionInfo, apiName, queryParams)
	LoggingInfo(hiscallInfo, start)

	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
	}
	cblogger.Debug(response.GetHttpContentString())

	expectStatus := true // 예상되는 상태 : 있어야 하므로 true
	result, err := WaitForEcsTagExist(client, regionInfo, resType, resIID, tag.Key, expectStatus)
	if err != nil {
		return nil, err
	}
	cblogger.Debug("Expect Status ", expectStatus, ", result Status ", result)
	if !result {
		return nil, errors.New("waitForTagExist Error ")
	}

	return response, nil
}

// 해당 ECS Resource에 특정 tag가 있는지 조회 :
func WaitForEcsTagExist(client *ecs.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID, tag string, expectStatus bool) (bool, error) {

	//waitStatus := false
	curRetryCnt := 0
	maxRetryCnt := 10 // 최대 10초 기다림
	for {

		// 해당 resource의 tag를 가져온다.
		response, err := DescribeDescribeEcsTags(client, regionInfo, resType, resIID, "") //tag.Key
		if err != nil {
			return false, err
		}

		// tag들 추출
		resTags := ecs.DescribeTagsResponse{}
		tagResponseStr := response.GetHttpContentString()
		err = json.Unmarshal([]byte(tagResponseStr), &resTags)
		if err != nil {
			cblogger.Error(err.Error())
			return false, err
		}

		// extract Tag
		existTag := false
		for _, aliTag := range resTags.Tags.Tag {
			//cblogger.Info(aliTag)
			cblogger.Info(aliTag.TagKey + ":" + tag)
			if aliTag.TagKey == tag {
				existTag = true
				break
			}
		}

		// expectStatus : 예상되는 상태.
		if expectStatus == existTag { // tag가 존재할 때까지 반복
			return true, nil
		}

		//if curStatus != irs.VMStatus(waitStatus) {
		curRetryCnt++
		cblogger.Debug("Waiting for 1 second and then querying")
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			return false, errors.New("After waiting for a long time")
		}
	}

	//return false, nil
}

func DescribeDescribeEcsTags(client *ecs.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID, key string) (*responses.CommonResponse, error) {
	apiName := "DescribeTags"
	regionID := regionInfo.Region

	// call logger set
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   "",
		ResourceType: call.TAG,
		ResourceName: "",
		CloudOSAPI:   apiName,
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	apiProductCode, err := GetAlibabaProductCode(irs.RSType(resType))
	if err != nil {
		return nil, err
	}

	alibabaResourceType, err2 := GetAlibabaResourceType(resType)
	if err2 != nil {
		return nil, err2
	}

	request := requests.NewCommonRequest()

	request.Method = "POST"
	request.Scheme = "https" // https | http
	//request.Domain = "ecs.cn-hongkong.aliyuncs.com"
	request.Domain = GetAlibabaApiEndPoint(regionID, apiProductCode)
	request.Version = "2014-05-26"
	request.ApiName = apiName
	//request.QueryParams["RegionId"] = regionID

	queryParams := map[string]string{}
	queryParams["RegionId"] = regionID
	queryParams["ResourceType"] = alibabaResourceType //string(resType)
	queryParams["ResourceId"] = resIID.SystemId
	if key != "" {
		queryParams["Tag.1.Key"] = key // 한번에 1개씩만 가져온다.
	}
	request.QueryParams = queryParams
	callLogStart := call.Start()
	response, err := client.ProcessCommonRequest(request)

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err.Error())
		return nil, err
	}
	cblogger.Debug(response.GetHttpContentString())

	return response, nil
}

func DescribeDescribeNlbTags(client *slb.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID, key string) (*responses.CommonResponse, error) {
	apiName := "DescribeTags"
	regionID := regionInfo.Region

	// call logger set
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   "",
		ResourceType: call.TAG,
		ResourceName: "",
		CloudOSAPI:   apiName,
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	apiProductCode, err := GetAlibabaProductCode(irs.RSType(resType))
	if err != nil {
		return nil, err
	}

	alibabaResourceType, err2 := GetAlibabaResourceType(resType)
	if err2 != nil {
		return nil, err2
	}

	request := requests.NewCommonRequest()

	request.Method = "POST"
	request.Scheme = "https" // https | http
	//request.Domain = "ecs.cn-hongkong.aliyuncs.com"
	request.Domain = GetAlibabaApiEndPoint(regionID, apiProductCode)
	request.Version = "2014-05-15"
	request.ApiName = apiName
	//request.QueryParams["RegionId"] = regionID

	queryParams := map[string]string{}
	queryParams["RegionId"] = regionID
	queryParams["ResourceType"] = alibabaResourceType //string(resType)
	queryParams["ResourceId"] = resIID.SystemId
	if key != "" {
		queryParams["Tag.1.Key"] = key // 한번에 1개씩만 가져온다.
	}
	request.QueryParams = queryParams
	callLogStart := call.Start()
	response, err := client.ProcessCommonRequest(request)

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err.Error())
		return nil, err
	}
	cblogger.Debug(response.GetHttpContentString())

	return response, nil
}

// //////// VPC begin /////////////
func AddVpcTags(client *vpc.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (*responses.CommonResponse, error) {
	apiName := "AddTags"
	regionID := regionInfo.Region

	cblogger.Info("Start Add EcsTag : ", tag)
	hiscallInfo := GetCallLogScheme(regionInfo, call.TAG, resIID.NameId, apiName)

	// 생성된 Tag 정보 획득 후, Tag 정보 리턴
	//tagInfo := irs.TagInfo{}

	// 지원하는 resource Type인지 확인
	alibabaResourceType, err := GetAlibabaResourceType(resType)
	if err != nil {
		return nil, err
	}

	queryParams := map[string]string{}
	queryParams["RegionId"] = regionID
	queryParams["ResourceType"] = alibabaResourceType
	queryParams["ResourceId"] = resIID.SystemId
	queryParams["Tag.1.Key"] = tag.Key
	queryParams["Tag.1.Value"] = tag.Value

	start := call.Start()
	response, err := CallVpcRequest(resType, client, regionInfo, apiName, queryParams)
	LoggingInfo(hiscallInfo, start)

	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
	}
	cblogger.Debug(response.GetHttpContentString())

	expectStatus := true // 예상되는 상태 : 있어야 하므로 true
	result, err := WaitForVpcTagExist(client, regionInfo, resType, resIID, tag.Key, expectStatus)
	if err != nil {
		return nil, err
	}
	cblogger.Debug("Expect Status ", expectStatus, ", result Status ", result)
	if !result {
		return nil, errors.New("waitForTagExist Error ")
	}

	return response, nil
}

// 해당 VPC Resource에 특정 tag가 있는지 조회 :
func WaitForVpcTagExist(client *vpc.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID, tag string, expectStatus bool) (bool, error) {

	//waitStatus := false
	curRetryCnt := 0
	maxRetryCnt := 3 // 최대 10초 기다림
	for {

		// 해당 resource의 tag를 가져온다.
		response, err := DescribeDescribeVpcTags(client, regionInfo, resType, resIID, "") //tag.Key
		if err != nil {
			return false, err
		}

		// tag들 추출
		resTags := ecs.DescribeTagsResponse{}
		tagResponseStr := response.GetHttpContentString()
		err = json.Unmarshal([]byte(tagResponseStr), &resTags)
		if err != nil {
			cblogger.Error(err.Error())
			return false, err
		}

		// extract Tag
		existTag := false
		for _, aliTag := range resTags.Tags.Tag {
			//cblogger.Info(aliTag)
			cblogger.Info(aliTag.TagKey + ":" + tag)
			if aliTag.TagKey == tag {
				existTag = true
				break
			}
		}

		// expectStatus : 예상되는 상태.
		if expectStatus == existTag { // tag가 존재할 때까지 반복
			return true, nil
		}

		//if curStatus != irs.VMStatus(waitStatus) {
		curRetryCnt++
		cblogger.Errorf("Waiting for 1 second and then querying")
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			return false, errors.New("After waiting for a long time")
		}
	}

	//return false, nil
}

func DescribeDescribeVpcTags(client *vpc.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID, key string) (*responses.CommonResponse, error) {
	apiName := "DescribeTags"
	regionID := regionInfo.Region

	// call logger set
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   "",
		ResourceType: call.TAG,
		ResourceName: "",
		CloudOSAPI:   apiName,
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	apiProductCode, err := GetAlibabaProductCode(irs.RSType(resType))
	if err != nil {
		return nil, err
	}

	alibabaResourceType, err2 := GetAlibabaResourceType(resType)
	if err2 != nil {
		return nil, err2
	}

	request := requests.NewCommonRequest()

	request.Method = "POST"
	request.Scheme = "https" // https | http
	//request.Domain = "ecs.cn-hongkong.aliyuncs.com"
	request.Domain = GetAlibabaApiEndPoint(regionID, apiProductCode)
	request.Version = "2014-05-26"
	request.ApiName = apiName
	request.QueryParams["RegionId"] = regionID

	queryParams := map[string]string{}
	queryParams["RegionId"] = regionID
	queryParams["ResourceType"] = alibabaResourceType //string(resType)
	queryParams["ResourceId"] = resIID.SystemId
	if key != "" {
		queryParams["Tag.1.Key"] = key // 한번에 1개씩만 가져온다.
	}

	callLogStart := call.Start()
	response, err := client.ProcessCommonRequest(request)

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err.Error())
		return nil, err
	}
	cblogger.Debug(response.GetHttpContentString())

	return response, nil
}

////////// VPC end /////////

// //// Call 공통 ///////////
// EcsRequest : Elastic Compute Service(ECS)
func CallEcsRequest(resType irs.RSType, client *ecs.Client, regionInfo idrv.RegionInfo, apiName string, queryParams map[string]string) (*responses.CommonResponse, error) {
	regionID := regionInfo.Region
	apiProductCode, err := GetAlibabaProductCode(resType)
	if err != nil {
		cblogger.Error(err.Error())
		return nil, err
	}

	// call logger set
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   "",
		ResourceType: call.TAG,
		ResourceName: "",
		CloudOSAPI:   apiName,
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := requests.NewCommonRequest()

	request.Method = "POST"
	request.Scheme = "https" // https | http
	request.Domain = GetAlibabaApiEndPoint(regionID, apiProductCode)
	request.Version = "2014-05-26"
	request.ApiName = apiName
	request.QueryParams = queryParams

	cblogger.Debug("API Request : ", request)

	callLogStart := call.Start()
	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		cblogger.Error(err.Error())
		return nil, err
	}

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))

	cblogger.Debug(response.GetHttpContentString())
	return response, nil
}

func CallVpcRequest(resType irs.RSType, client *vpc.Client, regionInfo idrv.RegionInfo, apiName string, queryParams map[string]string) (*responses.CommonResponse, error) {
	regionID := regionInfo.Region

	apiProductCode, err := GetAlibabaProductCode(irs.RSType(resType))

	// call logger set
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   "",
		ResourceType: call.TAG,
		ResourceName: "",
		CloudOSAPI:   apiName,
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := requests.NewCommonRequest()

	request.Method = "POST"
	request.Scheme = "https" // https | http
	request.Domain = GetAlibabaApiEndPoint(regionID, apiProductCode)
	request.Version = "2016-04-28"
	request.ApiName = apiName
	request.QueryParams["RegionId"] = regionID

	// Tag가 있으면
	if queryParams != nil {
		request.QueryParams = queryParams
	}

	callLogStart := call.Start()
	response, err := client.ProcessCommonRequest(request)

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err.Error())
		return nil, err
	}
	cblogger.Debug(response.GetHttpContentString())
	return response, nil
}

func CallNlbRequest(resType irs.RSType, client *slb.Client, regionInfo idrv.RegionInfo, apiName string, queryParams map[string]string) (*responses.CommonResponse, error) {
	regionID := regionInfo.Region

	apiProductCode, err := GetAlibabaProductCode(irs.RSType(resType))

	// call logger set
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   "",
		ResourceType: call.TAG,
		ResourceName: "",
		CloudOSAPI:   apiName,
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := requests.NewCommonRequest()

	request.Method = "POST"
	request.Scheme = "https" // https | http
	request.Domain = GetAlibabaApiEndPoint(regionID, apiProductCode)
	request.Version = "2014-05-15"
	request.ApiName = apiName
	request.QueryParams["RegionId"] = regionID

	// Tag가 있으면
	if queryParams != nil {
		request.QueryParams = queryParams
	}

	callLogStart := call.Start()
	response, err := client.ProcessCommonRequest(request)

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err.Error())
		return nil, err
	}
	cblogger.Debug(response.GetHttpContentString())
	return response, nil
}

///// Call 공통 end /////////

// Container service TagList : 원래는 ResourceIds이나 받는 Param이 1개이므로 1개Resource의 Tags
func aliCsListTag(csClient *cs.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID) (*cs.ListTagResourcesResponseBodyTagResources, error) {
	regionID := regionInfo.Region

	alibabaResourceType, err2 := GetAlibabaResourceType(resType)
	if err2 != nil {
		return nil, err2
	}

	// reqTag := &cs.Tag{
	// 	Key: tea.String("tk"),
	// 	Value: tea.String("tv"),
	//   }

	listTagResourcesRequest := &cs.ListTagResourcesRequest{
		RegionId:     tea.String(regionID),
		ResourceType: tea.String(alibabaResourceType),
		ResourceIds:  []*string{tea.String(resIID.SystemId)}, // clusterId
		//Tags:         []*cs.Tag{tag0},
	}
	//cblogger.Debug(describeClustersV1Request)
	listTagResponse, err := csClient.ListTagResources(listTagResourcesRequest)
	//describeClustersV1Response, err := csClient.ListTagResourcesWithOptions(listTagResourcesRequest, headers, runtime)
	if err != nil {
		return nil, err
	}
	cblogger.Debug(listTagResponse.Body)

	return listTagResponse.Body.TagResources, nil
}

// Container service Tag : 호출자체는 csListTag와 같으나 tagKey를 filter조건으로 추가
func aliCsTag(csClient *cs.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID, tagKey string) (*cs.ListTagResourcesResponseBodyTagResources, error) {
	regionID := regionInfo.Region

	alibabaResourceType, err2 := GetAlibabaResourceType(resType)
	if err2 != nil {
		return nil, err2
	}

	// reqTag := &cs.Tag{
	// 	Key: tea.String("tk"),
	// 	Value: tea.String("tv"),
	//   }

	listTagResourcesRequest := &cs.ListTagResourcesRequest{
		RegionId:     tea.String(regionID),
		ResourceType: tea.String(alibabaResourceType),
		ResourceIds:  []*string{tea.String(resIID.SystemId)}, // clusterId
		//Tags:         []*cs.Tag{tag0},
	}

	if tagKey != "" {
		reqTag := &cs.Tag{
			Key: tea.String(tagKey),
		}
		listTagResourcesRequest.Tags = []*cs.Tag{reqTag}
	}
	//cblogger.Debug(describeClustersV1Request)
	listTagResponse, err := csClient.ListTagResources(listTagResourcesRequest)
	//describeClustersV1Response, err := csClient.ListTagResourcesWithOptions(listTagResourcesRequest, headers, runtime)
	if err != nil {
		return nil, err
	}
	cblogger.Debug(listTagResponse.Body)

	return listTagResponse.Body.TagResources, nil
}

// Container service Tag : TagResources를 호출하면 PUT으로 추가 됨
func aliAddCsTag(csClient *cs.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (string, error) {
	regionID := regionInfo.Region

	alibabaResourceType, err2 := GetAlibabaResourceType(resType)
	if err2 != nil {
		return "", err2
	}

	// reqTag := &cs.Tag{
	// 	Key: tea.String("tk"),
	// 	Value: tea.String("tv"),
	//   }

	tagResourcesRequest := &cs.TagResourcesRequest{
		RegionId:     tea.String(regionID),
		ResourceType: tea.String(alibabaResourceType),        // CLUSTER"
		ResourceIds:  []*string{tea.String(resIID.SystemId)}, // clusterId
		//Tags:         []*cs.Tag{tag0},
	}

	reqTag := &cs.Tag{
		Key:   tea.String(tag.Key),
		Value: tea.String(tag.Value),
	}
	tagResourcesRequest.Tags = []*cs.Tag{reqTag}

	//cblogger.Debug(describeClustersV1Request)
	tagResourcesResponse, err := csClient.TagResources(tagResourcesRequest)
	//describeClustersV1Response, err := csClient.ListTagResourcesWithOptions(listTagResourcesRequest, headers, runtime)
	if err != nil {
		return "", err
	}
	cblogger.Debug(tagResourcesResponse.Body)

	return *tagResourcesResponse.Body.RequestId, nil
}

func aliRemoveCsTag(csClient *cs.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID, tagKey string) (bool, error) {
	regionID := regionInfo.Region

	alibabaResourceType, err2 := GetAlibabaResourceType(resType)
	if err2 != nil {
		return false, err2
	}

	// reqTag := &cs.Tag{
	// 	Key: tea.String("tk"),
	// 	Value: tea.String("tv"),
	//   }

	tagResourcesRequest := &cs.UntagResourcesRequest{
		RegionId:     tea.String(regionID),
		ResourceType: tea.String(alibabaResourceType),        // CLUSTER"
		ResourceIds:  []*string{tea.String(resIID.SystemId)}, // clusterId
		TagKeys:      []*string{tea.String(tagKey)},
	}

	cblogger.Debug(tagResourcesRequest)
	tagResourcesResponse, err := csClient.UntagResources(tagResourcesRequest)
	if err != nil {
		return false, err
	}
	cblogger.Debug(tagResourcesResponse.Body)

	return true, nil
}

// ALibaba tag 검색 for ecs
// myimage는 aliMyImageTag() 사용
func aliEcsTagList(Client *ecs.Client, regionInfo idrv.RegionInfo, alibabaResourceType string, resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	hiscallInfo := GetCallLogScheme(regionInfo, call.TAG, keyword, "FindTag()")
	regionID := regionInfo.Region
	var tagInfo []*irs.TagInfo

	queryParams := map[string]string{}
	queryParams["RegionId"] = regionID
	queryParams["ResourceType"] = alibabaResourceType //string(resType)//keypair
	if keyword != "" {
		queryParams["Tag.1.Key"] = keyword
	}

	start := call.Start()
	response, err := CallEcsRequest(resType, Client, regionInfo, "DescribeResourceByTags", queryParams)
	LoggingInfo(hiscallInfo, start)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
	}
	cblogger.Debug(response.GetHttpContentString())
	resResources := AliTagResourcesResponse{}

	tagResponseStr := response.GetHttpContentString()
	err = json.Unmarshal([]byte(tagResponseStr), &resResources)

	if err != nil {
		cblogger.Error(err.Error())
		return tagInfo, nil
	}

	for _, aliTagResource := range resResources.AliTagResources.Resources {

		cblogger.Debug("aliTagResource ", aliTagResource)
		aTagInfo, err := ExtractTagResourceInfo(&aliTagResource)
		if err != nil {
			cblogger.Error(err.Error())
			continue
		}
		cblogger.Info("aTagInfoaTagInfoaTagInfo", aTagInfo)
		// api
		aTagInfo.ResType = resType

		queryParams := map[string]string{
			"ResourceId": aTagInfo.ResIId.SystemId,
		}
		response, err := CallEcsRequest(resType, Client, regionInfo, "DescribeTags", queryParams)
		if err != nil {
			cblogger.Error(err.Error())
			continue
		}

		var tagsResponse DescribeTagsResponse
		err = json.Unmarshal([]byte(response.GetHttpContentString()), &tagsResponse)
		if err != nil {
			cblogger.Error("Failed to unmarshal response: ", err)
			continue
		}

		aTagInfo.TagList = []irs.KeyValue{}
		for _, tag := range tagsResponse.Tags.Tag {
			aTagInfo.TagList = append(aTagInfo.TagList, irs.KeyValue{
				Key:   tag.TagKey,
				Value: tag.TagValue,
			})
		}

		cblogger.Debug("Updated tagInfo ", aTagInfo)
		tagInfo = append(tagInfo, &aTagInfo)
	}
	return tagInfo, nil
}

func aliMyImageTagList(Client *ecs.Client, regionInfo idrv.RegionInfo, keyword string) ([]*irs.TagInfo, error) {
	var tagInfo []*irs.TagInfo
	// TODO: keyword tag 검색 기능
	res, err := DescribeImages(Client, regionInfo, nil, true)

	cblogger.Info("resresresresres", err)
	cblogger.Info("resresresresres", res)
	// spew.Dump(res)

	return tagInfo, nil
}

func aliVpcTagList(VpcClient *vpc.Client, regionInfo idrv.RegionInfo, alibabaResourceType string, resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	hiscallInfo := GetCallLogScheme(regionInfo, call.TAG, keyword, "FindTag()")
	regionID := regionInfo.Region
	var tagInfo []*irs.TagInfo

	queryParams := map[string]string{}
	queryParams["RegionId"] = regionID
	queryParams["ResourceType"] = alibabaResourceType //string(resType)//keypair

	start := call.Start()

	response, err := CallVpcRequest(resType, VpcClient, regionInfo, "DescribeVpcs", queryParams)
	LoggingInfo(hiscallInfo, start)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
	}
	cblogger.Debug(response.GetHttpContentString())
	resResources := DescribeVpcsResponse{}

	tagResponseStr := response.GetHttpContentString()
	err = json.Unmarshal([]byte(tagResponseStr), &resResources)

	if err != nil {
		cblogger.Error(err.Error())
		return tagInfo, nil
	}

	for _, vpc := range resResources.Vpcs.Vpc {
		for _, tag := range vpc.Tags.Tag {
			aliTagResource := AliTagResource{
				ResourceType: "VPC",
				ResourceId:   vpc.VpcId,
				TagKey:       tag.Key,
				TagValue:     tag.Value,
			}
			aTagInfo, err := ExtractTagResourceInfo(&aliTagResource)
			if err != nil {
				cblogger.Error(err.Error())
				continue
			}
			tagInfo = append(tagInfo, &aTagInfo)
		}
	}
	return tagInfo, nil
}

func aliSubnetTagList(VpcClient *vpc.Client, regionInfo idrv.RegionInfo, alibabaResourceType string, resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	hiscallInfo := GetCallLogScheme(regionInfo, call.TAG, keyword, "FindTag()")
	regionID := regionInfo.Region
	var tagInfo []*irs.TagInfo

	queryParams := map[string]string{}
	queryParams["RegionId"] = regionID
	queryParams["ResourceType"] = alibabaResourceType //string(resType)//keypair

	start := call.Start()
	response, err := CallVpcRequest(resType, VpcClient, regionInfo, "DescribeVSwitches", queryParams)
	LoggingInfo(hiscallInfo, start)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
	}
	cblogger.Debug(response.GetHttpContentString())
	resResources := DescribeVSwitchesResponse{}

	tagResponseStr := response.GetHttpContentString()
	err = json.Unmarshal([]byte(tagResponseStr), &resResources)

	if err != nil {
		cblogger.Error("Failed to unmarshal response: ", err)
	}

	for _, vswitch := range resResources.VSwitches.VSwitch {
		if len(vswitch.Tags.Tag) == 0 {
			continue
		}
		for _, tag := range vswitch.Tags.Tag {
			aliTagResource := AliTagResource{
				ResourceType: "VSwitch",
				ResourceId:   vswitch.VSwitchId,
				TagKey:       tag.Key,
				TagValue:     tag.Value,
			}
			cblogger.Debug("aliTagResourcealiTagResourcealiTagResourcealiTagResource", resType)

			vswitchTagInfo, err := ExtractTagResourceInfo(&aliTagResource)
			cblogger.Debug("vswitchTagInfovswitchTagInfovswitchTagInfo", vswitchTagInfo)
			if err != nil {
				cblogger.Error(err.Error())
				continue
			}
			tagInfo = append(tagInfo, &vswitchTagInfo)
		}
	}
	return tagInfo, nil
}

func aliClusterTagList(CsClient *cs.Client, regionInfo idrv.RegionInfo, resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	hiscallInfo := GetCallLogScheme(regionInfo, call.TAG, keyword, "FindTag()")
	var tagInfoList []*irs.TagInfo

	regionID := regionInfo.Region
	clusters, err := aliDescribeClustersV1(CsClient, regionID)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	//cblogger.Debug("clusters ", clusters)
	// 모든 cluster를 돌면서 Tag 찾기
	for _, cluster := range clusters {
		cblogger.Debug("inCluster ")
		for _, aliTag := range cluster.Tags {
			//cblogger.Debug("aliTag ", aliTag)
			//cblogger.Debug("keyword ", keyword)
			//cblogger.Debug("aliTag.Key ", *(aliTag.Key))
			if *(aliTag.Key) == keyword {
				var aTagInfo irs.TagInfo
				aTagInfo.ResIId = irs.IID{SystemId: *cluster.ClusterId}
				aTagInfo.ResType = resType

				tagList := []irs.KeyValue{}
				tagList = append(tagList, irs.KeyValue{Key: "TagKey", Value: *aliTag.Key})
				tagList = append(tagList, irs.KeyValue{Key: "TagValue", Value: *aliTag.Value})
				aTagInfo.TagList = tagList
				//cblogger.Debug("append Tag ", &tagInfo)
				// tagInfo = &aTagInfo
			}
		}
	}
	return tagInfoList, err
}

func GetSubnet(VpcClient *vpc.Client, reqSubnetId string, zone string) (irs.SubnetInfo, error) {
	cblogger.Infof("SubnetId : [%s]", reqSubnetId)

	request := vpc.CreateDescribeVSwitchesRequest()
	request.Scheme = "https"
	request.VSwitchId = reqSubnetId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: reqSubnetId,
		CloudOSAPI:   "DescribeVSwitches()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := VpcClient.DescribeVSwitches(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//cblogger.Debug(result)
	//cblogger.Info(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.SubnetInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	if result.TotalCount < 1 {
		return irs.SubnetInfo{}, errors.New("Notfound: '" + reqSubnetId + "' Subnet Not found")
	}

	if !reflect.ValueOf(result.VSwitches.VSwitch).IsNil() {
		retSubnetInfo := ExtractSubnetDescribeInfo(result.VSwitches.VSwitch[0])
		return retSubnetInfo, nil
	} else {
		return irs.SubnetInfo{}, errors.New("InvalidVSwitch.NotFound: The '" + reqSubnetId + "' does not exist")
	}
}

func aliAddSlbTag(Client *slb.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (*responses.CommonResponse, error) {
	apiName := "AddTags"
	regionID := regionInfo.Region

	cblogger.Info("Start Add NlbTag : ", tag)
	hiscallInfo := GetCallLogScheme(regionInfo, call.TAG, resIID.NameId, apiName)

	// 생성된 Tag 정보 획득 후, Tag 정보 리턴
	//tagInfo := irs.TagInfo{}
	tagJson := fmt.Sprintf(`[{"TagKey":"%s","TagValue":"%s"}]`, tag.Key, tag.Value)

	// 지원하는 resource Type인지 확인
	alibabaResourceType, err := GetAlibabaResourceType(resType)
	if err != nil {
		return nil, err
	}

	queryParams := map[string]string{}
	queryParams["RegionId"] = regionID
	queryParams["ResourceType"] = alibabaResourceType
	queryParams["LoadBalancerId"] = resIID.SystemId
	queryParams["Tags"] = tagJson

	start := call.Start()
	response, err := CallNlbRequest(resType, Client, regionInfo, apiName, queryParams)
	LoggingInfo(hiscallInfo, start)

	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
	}
	cblogger.Debug(response.GetHttpContentString())

	expectStatus := true // 예상되는 상태 : 있어야 하므로 true
	result, err := WaitForNlbTagExist(Client, regionInfo, resType, resIID, tag.Key, expectStatus)
	if err != nil {
		return nil, err
	}
	cblogger.Debug("Expect Status ", expectStatus, ", result Status ", result)
	if !result {
		return nil, errors.New("waitForTagExist Error ")
	}

	return response, nil
}

// 해당 NLB Resource에 특정 tag가 있는지 조회 :
func WaitForNlbTagExist(client *slb.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID, tag string, expectStatus bool) (bool, error) {

	//waitStatus := false
	curRetryCnt := 0
	maxRetryCnt := 15 // 최대 10초 기다림
	for {

		// 해당 resource의 tag를 가져온다.
		response, err := DescribeDescribeNlbTags(client, regionInfo, resType, resIID, "") //tag.Key
		if err != nil {
			return false, err
		}

		// tag들 추출
		resTags := slb.DescribeTagsResponse{}
		tagResponseStr := response.GetHttpContentString()
		err = json.Unmarshal([]byte(tagResponseStr), &resTags)
		if err != nil {
			cblogger.Error(err.Error())
			return false, err
		}

		// extract Tag
		existTag := false
		for _, aliTag := range resTags.TagSets.TagSet {
			//cblogger.Info(aliTag)
			cblogger.Info(aliTag.TagKey + ":" + tag)
			if aliTag.TagKey == tag {
				existTag = true
				break
			}
		}

		// expectStatus : 예상되는 상태.
		if expectStatus == existTag { // tag가 존재할 때까지 반복
			return true, nil
		}

		//if curStatus != irs.VMStatus(waitStatus) {
		curRetryCnt++
		cblogger.Errorf("Waiting for 1 second and then querying")
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			return false, errors.New("After waiting for a long time")
		}
	}

	//return false, nil
}
