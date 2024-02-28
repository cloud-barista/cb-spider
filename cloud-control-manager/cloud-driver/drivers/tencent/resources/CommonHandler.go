package resources

import (
	"errors"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cbs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs/v20170312"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"

	tencentError "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
)

func DescribeDisks(client *cbs.Client, diskIIDs []irs.IID) ([]*cbs.Disk, error) {
	request := cbs.NewDescribeDisksRequest()

	if diskIIDs != nil {
		request.DiskIds = common.StringPtrs([]string{diskIIDs[0].SystemId})
	}

	response, err := client.DescribeDisks(request)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	return response.Response.DiskSet, nil
}

func DescribeDisksByDiskID(client *cbs.Client, diskIID irs.IID) (cbs.Disk, error) {
	var diskIIDList []irs.IID
	diskIIDList = append(diskIIDList, diskIID)

	diskList, err := DescribeDisks(client, diskIIDList)
	if err != nil {
		return cbs.Disk{}, err
	}

	if len(diskList) != 1 {
		return cbs.Disk{}, errors.New("search failed")
	}

	return *diskList[0], nil
}

func WaitForDelete(client *cvm.Client, imageIID irs.IID) (bool, error) {
	var imageIIDList []irs.IID
	imageIIDList = append(imageIIDList, imageIID)

	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		imageList, err := DescribeImages(client, imageIIDList, nil)
		if err != nil {
			cblogger.Error(err.Error())
		}

		if len(imageList) == 0 {
			cblogger.Info("Image deletion is complete and stops waiting.")
			break
		}

		curRetryCnt++
		cblogger.Info("Image deletion has not been completed, so we will look up the climate for 1 second.")
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf("If you wait for a long time (%d seconds), the deletion of the image does not complete, so forcefully stop.", maxRetryCnt)
			return false, errors.New("Failed to delete image")
		}
	}

	return true, nil
}

func WaitForDone(client *cbs.Client, diskIID irs.IID, status string) (string, error) {

	waitStatus := status

	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		request := cbs.NewDescribeDisksRequest()

		request.DiskIds = common.StringPtrs([]string{diskIID.SystemId})

		response, errStatus := client.DescribeDisks(request)
		if errStatus != nil {
			cblogger.Error(errStatus.Error())
		}

		curStatus := *response.Response.DiskSet[0].DiskState

		cblogger.Info("===>Disk Status : ", curStatus)

		if curStatus == waitStatus {
			cblogger.Infof("===>Suspends standby because disk state is [%s].", curStatus)
			break
		}

		curRetryCnt++
		cblogger.Infof("Disk status is not [%s] and climate lookup is performed in 1 second.", waitStatus)
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf("Waiting for a long time (%d seconds) does not change the disk status value to [%s] and forces it to stop.", maxRetryCnt, waitStatus)
			return "Failed", errors.New("After waiting a long time, the status of the created disk does not change to [" + waitStatus + "] and it is interrupted.")
		}
	}

	return waitStatus, nil
}

func AttachDisk(client *cbs.Client, diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {
	request := cbs.NewAttachDisksRequest()

	request.InstanceId = common.StringPtr(ownerVM.SystemId)
	request.DiskIds = common.StringPtrs([]string{diskIID.SystemId})

	_, err := client.AttachDisks(request)
	if err != nil {
		cblogger.Error(err)
		return irs.DiskInfo{}, err
	}

	return irs.DiskInfo{}, nil
}

func DescribeImages(client *cvm.Client, myImageIIDs []irs.IID, imageTypes []string) ([]*cvm.Image, error) {
	request := cvm.NewDescribeImagesRequest()

	if myImageIIDs != nil {
		request.ImageIds = common.StringPtrs([]string{myImageIIDs[0].SystemId})
	} else {
		if imageTypes != nil && len(imageTypes) > 0 {
			request.Filters = []*cvm.Filter{
				{
					Name:   common.StringPtr("image-type"),
					Values: common.StringPtrs(imageTypes),
				},
			}
		}
	}

	response, err := client.DescribeImages(request)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	return response.Response.ImageSet, nil
}

// imageTypes : PUBLIC_IMAGE, SHARED_IMAGE, PRIVATE_IMAGE
func DescribeImagesByID(client *cvm.Client, myImageIID irs.IID, imageTypes []string) (cvm.Image, error) {
	var myImageIIDList []irs.IID
	myImageIIDList = append(myImageIIDList, myImageIID)

	myImageList, err := DescribeImages(client, myImageIIDList, imageTypes)
	if err != nil {
		return cvm.Image{}, err
	}

	if len(myImageList) != 1 {
		return cvm.Image{}, errors.New("search failed")
	}

	return *myImageList[0], nil
}

func DescribeImageStatus(client *cvm.Client, imageIID irs.IID, imageTypes []string) (string, error) {
	cvmImage, err := DescribeImagesByID(client, imageIID, imageTypes)
	if err != nil {
		return "", err
	}

	status := *cvmImage.ImageState

	return status, nil
}

func GetSnapshotIdsFromImage(myImage cvm.Image) []string {
	var snapshotIds []string

	for _, snapshot := range myImage.SnapshotSet {
		snapshotId := *snapshot.SnapshotId
		snapshotIds = append(snapshotIds, snapshotId)
	}
	return snapshotIds
}

func DescribeSnapshotByID(client *cbs.Client, snapshotIID irs.IID) (cbs.Snapshot, error) {
	request := cbs.NewDescribeSnapshotsRequest()
	request.SnapshotIds = common.StringPtrs([]string{snapshotIID.SystemId})

	response, err := client.DescribeSnapshots(request)
	if err != nil {
		return cbs.Snapshot{}, err
	}

	if len(response.Response.SnapshotSet) != 1 {
		return cbs.Snapshot{}, errors.New("search failed")
	}

	return *response.Response.SnapshotSet[0], nil
}

func DescribeSnapshotStatus(client *cbs.Client, snapshotIID irs.IID) (string, error) {
	snapshot, err := DescribeSnapshotByID(client, snapshotIID)
	if err != nil {
		return "", err
	}

	status := *snapshot.SnapshotState

	return status, nil
}

// Image에서 OS Type 추출
// "OsName": "TencentOS Server 3.1 (TK4)",
// "Platform": "TencentOS",
func GetOsType(cvmImage cvm.Image) string {
	cblogger.Info("OsName,", *cvmImage.Platform)
	return *cvmImage.Platform
}

// ListOrgRegion
func DescribeRegions(client *cvm.Client) (*cvm.DescribeRegionsResponse, error) {
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   client.GetRegion(),
		ResourceType: call.REGIONZONE,
		ResourceName: "",
		CloudOSAPI:   "DescribeRegions()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	inputRegions := cvm.NewDescribeRegionsRequest()

	callLogStart := call.Start()
	responseRegions, err := client.DescribeRegions(inputRegions)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))

	if _, ok := err.(*tencentError.TencentCloudSDKError); ok {
		cblogger.Error(err)
		return nil, err
	}

	return responseRegions, nil
}

// ListOrgZone : 클라이언트가 Region 정보를 갖고 있음.
func DescribeZones(client *cvm.Client) (*cvm.DescribeZonesResponse, error) {
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   client.GetRegion(),
		ResourceType: call.REGIONZONE,
		ResourceName: "",
		CloudOSAPI:   "DescribeZones()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	inputZones := cvm.NewDescribeZonesRequest()

	callLogStart := call.Start()
	responseZones, err := client.DescribeZones(inputZones)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))

	if _, ok := err.(*tencentError.TencentCloudSDKError); ok {
		cblogger.Error(err)
		return nil, err
	}

	return responseZones, nil
}
