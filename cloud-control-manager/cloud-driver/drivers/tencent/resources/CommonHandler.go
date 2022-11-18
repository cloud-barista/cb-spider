package resources

import (
	"errors"
	"time"

	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cbs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs/v20170312"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
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
			cblogger.Infof("===>Disk 상태가 [%s]라서 대기를 중단합니다.", curStatus)
			break
		}

		curRetryCnt++
		cblogger.Infof("Disk 상태가 [%s]이 아니라서 1초 대기후 조회합니다.", waitStatus)
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf("장시간(%d 초) 대기해도 Disk Status 값이 [%s]으로 변경되지 않아서 강제로 중단합니다.", maxRetryCnt, waitStatus)
			return "Failed", errors.New("장시간 기다렸으나 생성된 Disk의 상태가 [" + waitStatus + "]으로 바뀌지 않아서 중단 합니다.")
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

// Image에서 OS Type 추출
// "OsName": "TencentOS Server 3.1 (TK4)",
// "Platform": "TencentOS",
func GetOsType(cvmImage cvm.Image) string {
	cblogger.Info("OsName,", *cvmImage.Platform)
	return *cvmImage.Platform
}
