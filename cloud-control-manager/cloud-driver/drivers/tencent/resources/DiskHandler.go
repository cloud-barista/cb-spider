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

const (
	Disk_Status_Attached   = "ATTACHED"
	Disk_Status_Unattached = "UNATTACHED"
)

/*
CreateDisk 이후에 DescribeDisks 호출하여 상태가 UNATTACHED 또는 ATTACHED면 정상적으로 생성된 것임
비동기로 처리되기는 하나 생성 직후 호출해도 정상적으로 상태값을 받아옴
따라서 Operation이 완료되길 기다리는 function(WaitForXXX)은 만들지 않음
*/
func (DiskHandler *TencentDiskHandler) CreateDisk(diskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {

	existName, errExist := DiskHandler.diskExist(diskReqInfo.IId.NameId)
	if errExist != nil {
		cblogger.Error(errExist)
		return irs.DiskInfo{}, errExist
	}
	if existName {
		return irs.DiskInfo{}, errors.New("A disk with the name " + diskReqInfo.IId.NameId + " already exists.")
	}

	request := cbs.NewCreateDisksRequest()
	request.Placement = &cbs.Placement{Zone: common.StringPtr(DiskHandler.Region.Zone)}
	request.DiskChargeType = common.StringPtr("POSTPAID_BY_HOUR")

	diskErr := validateDisk(&diskReqInfo)
	if diskErr != nil {
		cblogger.Error(diskErr)
		return irs.DiskInfo{}, diskErr
	}

	diskSize, sizeErr := strconv.ParseUint(diskReqInfo.DiskSize, 10, 64)
	if sizeErr != nil {
		return irs.DiskInfo{}, sizeErr
	}

	request.DiskSize = common.Uint64Ptr(diskSize)
	request.DiskType = common.StringPtr(diskReqInfo.DiskType)
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

	diskSet, err := DescribeDisks(DiskHandler.Client, nil)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	for _, disk := range diskSet {
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

	targetDisk, err := DescribeDisksByDiskID(DiskHandler.Client, diskIID)
	if err != nil {
		cblogger.Error(err)
		return irs.DiskInfo{}, err
	}

	diskInfo, diskInfoErr := convertDiskInfo(&targetDisk)
	if diskInfoErr != nil {
		cblogger.Error(diskInfoErr)
		return irs.DiskInfo{}, diskInfoErr
	}

	return diskInfo, nil
}

func (DiskHandler *TencentDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
	diskInfo, diskInfoErr := DiskHandler.GetDisk(diskIID)
	if diskInfoErr != nil {
		return false, diskInfoErr
	}

	diskSizeErr := validateChangeDiskSize(diskInfo, size)
	if diskSizeErr != nil {
		return false, diskSizeErr
	}

	newSize, sizeErr := strconv.ParseUint(size, 10, 64)
	if sizeErr != nil {
		cblogger.Error(sizeErr)
		return false, sizeErr
	}

	request := cbs.NewResizeDiskRequest()

	request.DiskId = common.StringPtr(diskIID.SystemId)
	request.DiskSize = common.Uint64Ptr(newSize)

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

	_, attachErr := AttachDisk(DiskHandler.Client, irs.IID{SystemId: diskIID.SystemId}, irs.IID{SystemId: ownerVM.SystemId})
	if attachErr != nil {
		return irs.DiskInfo{}, attachErr
	}

	_, statusErr := WaitForDone(DiskHandler.Client, irs.IID{SystemId: diskIID.SystemId}, Disk_Status_Attached)
	if statusErr != nil {
		return irs.DiskInfo{}, statusErr
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

	_, statusErr := WaitForDone(DiskHandler.Client, irs.IID{SystemId: diskIID.SystemId}, Disk_Status_Unattached)
	if statusErr != nil {
		return false, statusErr
	}

	return true, nil
}

func convertDiskInfo(diskResp *cbs.Disk) (irs.DiskInfo, error) {
	diskInfo := irs.DiskInfo{}

	diskInfo.IId = irs.IID{NameId: *diskResp.DiskName, SystemId: *diskResp.DiskId}
	diskInfo.DiskType = *diskResp.DiskType
	diskInfo.DiskSize = strconv.FormatInt(int64(*diskResp.DiskSize), 10)
	diskInfo.OwnerVM.SystemId = *diskResp.InstanceId
	diskInfo.CreatedTime, _ = time.Parse("2006-01-02 15:04:05", *diskResp.CreateTime)
	diskInfo.Status = convertTenStatusToDiskStatus(diskResp)

	return diskInfo, nil
}

func convertTenStatusToDiskStatus(diskInfo *cbs.Disk) irs.DiskStatus {
	var returnStatus irs.DiskStatus

	if *diskInfo.Attached {
		returnStatus = irs.DiskAttached
	} else {
		returnStatus = irs.DiskAvailable
	}

	return returnStatus
}

func validateDisk(diskReqInfo *irs.DiskInfo) error {
	cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("TENCENT")
	arrDiskType := cloudOSMetaInfo.DiskType
	arrDiskSizeOfType := cloudOSMetaInfo.DiskSize
	arrRootDiskSizeOfType := cloudOSMetaInfo.RootDiskSize

	reqDiskType := diskReqInfo.DiskType
	reqDiskSize := diskReqInfo.DiskSize

	if reqDiskType == "" || reqDiskType == "default" {
		diskSizeArr := strings.Split(arrRootDiskSizeOfType[0], "|")
		reqDiskType = diskSizeArr[0]          //
		diskReqInfo.DiskType = diskSizeArr[0] // set default value
	}
	// 정의된 type인지
	if !ContainString(arrDiskType, reqDiskType) {
		return errors.New("Disktype : " + reqDiskType + "' is not valid")
	}

	if reqDiskSize == "" || reqDiskSize == "default" {
		diskSizeArr := strings.Split(arrRootDiskSizeOfType[0], "|")
		reqDiskSize = diskSizeArr[1]
		diskReqInfo.DiskSize = diskSizeArr[1] // set default value
	}

	diskSize, err := strconv.ParseInt(reqDiskSize, 10, 64)
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
		if strings.EqualFold(reqDiskType, diskSizeArr[0]) {
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
		return errors.New("Invalid Disk Type : " + reqDiskType)
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

func validateChangeDiskSize(diskInfo irs.DiskInfo, newSize string) error {
	cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("TENCENT")
	arrDiskSizeOfType := cloudOSMetaInfo.DiskSize

	diskSize, err := strconv.ParseInt(diskInfo.DiskSize, 10, 64)
	if err != nil {
		cblogger.Error(err)
		return err
	}

	newDiskSize, err := strconv.ParseInt(newSize, 10, 64)
	if err != nil {
		cblogger.Error(err)
		return err
	}

	if diskSize >= newDiskSize {
		return errors.New("Target Disk Size: " + newSize + " must be larger than existing Disk Size " + diskInfo.DiskSize)
	}

	type diskSizeModel struct {
		diskType    string
		diskMinSize int64
		diskMaxSize int64
		unit        string
	}

	diskSizeValue := diskSizeModel{}

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
		}
	}

	if newDiskSize > diskSizeValue.diskMaxSize {
		fmt.Println("Disk Size Error!!: ", diskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be smaller than or equal to the maximum size (" + strconv.FormatInt(diskSizeValue.diskMaxSize, 10) + " GB).")
	}

	return nil
}

/*
disk가 존재하는지 check
동일이름이 없으면 false, 있으면 true
*/
func (DiskHandler *TencentDiskHandler) diskExist(chkName string) (bool, error) {
	cblogger.Debugf("chkName : %s", chkName)

	request := cbs.NewDescribeDisksRequest()

	request.Filters = []*cbs.Filter{
		{
			Name:   common.StringPtr("disk-name"),
			Values: common.StringPtrs([]string{chkName}),
		},
	}

	response, err := DiskHandler.Client.DescribeDisks(request)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	if *response.Response.TotalCount < 1 {
		return false, nil
	}

	cblogger.Infof("Disk 정보 찾음 - DiskId:[%s] / DiskName:[%s]", *response.Response.DiskSet[0].DiskId, *response.Response.DiskSet[0].DiskName)
	return true, nil
}
