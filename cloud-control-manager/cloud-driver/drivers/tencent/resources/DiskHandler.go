package resources

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
	cbs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs/v20170312"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

type TencentDiskHandler struct {
	Region idrv.RegionInfo
	Client *cbs.Client
}

func (DiskHandler *TencentDiskHandler) CreateDisk(diskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	diskSize, sizeErr := strconv.ParseUint(diskReqInfo.DiskSize, 10, 64)
	if sizeErr != nil {
		return irs.DiskInfo{}, sizeErr
	}

	request := cbs.NewCreateDisksRequest()
	request.Placement = &cbs.Placement{Zone: common.StringPtr(DiskHandler.Region.Zone)}
	request.DiskChargeType = common.StringPtr("POSTPAID_BY_HOUR")
	request.DiskType = common.StringPtr(diskReqInfo.DiskType)
	request.DiskSize = common.Uint64Ptr(diskSize)
	request.DiskName = common.StringPtr(diskReqInfo.IId.NameId)

	response, err := DiskHandler.Client.CreateDisks(request)
	if err != nil {
		cblogger.Error(err)
		return irs.DiskInfo{}, err
	}

	newDiskId := *response.Response.DiskIdSet[0]
	cblogger.Debug(newDiskId)

	diskInfo, diskInfoErr := DiskHandler.GetDisk(irs.IID{SystemId: newDiskId})
	if diskInfoErr != nil {
		cblogger.Error(diskInfoErr)
		return irs.DiskInfo{}, diskInfoErr
	}

	return diskInfo, nil
}

func (DiskHandler *TencentDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	diskInfoList := []*irs.DiskInfo{}

	request := cbs.NewDescribeDisksRequest()

	response, err := DiskHandler.Client.DescribeDisks(request)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	for _, disk := range response.Response.DiskSet {
		diskInfo, diskInfoErr := convertDiskInfo(disk)
		if diskInfoErr != nil {
			cblogger.Error(diskInfoErr)
			return nil, diskInfoErr
		}

		diskInfoList = append(diskInfoList, &diskInfo)
	}

	return diskInfoList, nil
}

func (DiskHandler *TencentDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {

	request := cbs.NewDescribeDisksRequest()

	request.DiskIds = common.StringPtrs([]string{diskIID.SystemId})

	response, err := DiskHandler.Client.DescribeDisks(request)
	if err != nil {
		cblogger.Error(err)
		return irs.DiskInfo{}, err
	}

	targetDisk := *response.Response.DiskSet[0]

	diskInfo, diskInfoErr := convertDiskInfo(&targetDisk)
	if diskInfoErr != nil {
		cblogger.Error(diskInfoErr)
		return irs.DiskInfo{}, diskInfoErr
	}

	return diskInfo, nil
}

func (DiskHandler *TencentDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
	diskSize, sizeErr := strconv.ParseUint(size, 10, 64)
	if sizeErr != nil {
		return false, sizeErr
	}

	request := cbs.NewResizeDiskRequest()

	request.DiskId = common.StringPtr(diskIID.SystemId)
	request.DiskSize = common.Uint64Ptr(diskSize)

	_, err := DiskHandler.Client.ResizeDisk(request)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	return true, nil
}

func (DiskHandler *TencentDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	request := cbs.NewTerminateDisksRequest()

	request.DiskIds = common.StringPtrs([]string{diskIID.SystemId})

	_, err := DiskHandler.Client.TerminateDisks(request)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	return true, nil
}

func (DiskHandler *TencentDiskHandler) AttachDisk(diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {
	request := cbs.NewAttachDisksRequest()

	request.InstanceId = common.StringPtr(ownerVM.SystemId)
	request.DiskIds = common.StringPtrs([]string{diskIID.SystemId})

	_, err := DiskHandler.Client.AttachDisks(request)
	if err != nil {
		cblogger.Error(err)
		return irs.DiskInfo{}, err
	}

	diskInfo, diskInfoErr := DiskHandler.GetDisk(irs.IID{SystemId: diskIID.SystemId})
	if diskInfoErr != nil {
		cblogger.Error(diskInfoErr)
		return irs.DiskInfo{}, diskInfoErr
	}

	return diskInfo, nil
}

func (DiskHandler *TencentDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {

	request := cbs.NewDetachDisksRequest()

	request.DiskIds = common.StringPtrs([]string{diskIID.SystemId})

	_, err := DiskHandler.Client.DetachDisks(request)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	return true, nil
}

func convertDiskInfo(diskResp *cbs.Disk) (irs.DiskInfo, error) {
	diskInfo := irs.DiskInfo{}

	diskInfo.IId = irs.IID{NameId: *diskResp.DiskName, SystemId: *diskResp.DiskId}
	diskInfo.DiskType = *diskResp.DiskType
	diskInfo.DiskSize = strconv.FormatInt(int64(*diskResp.DiskSize), 10)
	diskInfo.OwnerVM.SystemId = *diskResp.InstanceId
	diskInfo.CreatedTime, _ = time.Parse(time.RFC3339, *diskResp.CreateTime)

	return diskInfo, nil
}

func validateDiskSize(diskInfo irs.DiskInfo) error {
	cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("TENCENT")
	arrDiskSizeOfType := cloudOSMetaInfo.DiskSize

	diskSize, err := strconv.ParseInt(diskInfo.DiskSize, 10, 64)
	if err != nil {
		cblogger.Error(err)
		return err
	}

	type diskSizeModel struct {
		diskType    string
		diskMinSize int64
		diskMaxSize int64
		unit        string
	}

	diskSizeValue := diskSizeModel{}
	isExists := false

	for _, diskSizeInfo := range arrDiskSizeOfType {
		diskSizeArr := strings.Split(diskSizeInfo, "|")
		if strings.EqualFold(diskInfo.DiskType, diskSizeArr[0]) {
			diskSizeValue.diskType = diskSizeArr[0]
			diskSizeValue.unit = diskSizeArr[3]
			diskSizeValue.diskMinSize, err = strconv.ParseInt(diskSizeArr[1], 10, 64)
			if err != nil {
				cblogger.Error(err)
				return err
			}

			diskSizeValue.diskMaxSize, err = strconv.ParseInt(diskSizeArr[2], 10, 64)
			if err != nil {
				cblogger.Error(err)
				return err
			}
			isExists = true
		}
	}

	if !isExists {
		return errors.New("Invalid Disk Type : " + diskInfo.DiskType)
	}

	if diskSize < diskSizeValue.diskMinSize {
		fmt.Println("Disk Size Error!!: ", diskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be at least the minimum size (" + strconv.FormatInt(diskSizeValue.diskMinSize, 10) + " GB).")
	}

	if diskSize > diskSizeValue.diskMaxSize {
		fmt.Println("Disk Size Error!!: ", diskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be smaller than or equal to the maximum size (" + strconv.FormatInt(diskSizeValue.diskMaxSize, 10) + " GB).")
	}

	return nil
}
