package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strings"
	"time"
)

type IbmMyImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VpcService     *vpcv1.VpcV1
	Ctx            context.Context
}

const DEV = "-dev-"

func (myImageHandler *IbmMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.Region, call.MYIMAGE, snapshotReqInfo.IId.NameId, "SnapshotVM()")

	if len(snapshotReqInfo.IId.NameId) > 55 {
		return irs.MyImageInfo{}, errors.New(fmt.Sprintf("MyImage Name ID cannot be longer than 55 characters"))
	}
	if strings.Contains(snapshotReqInfo.IId.NameId, DEV) {
		return irs.MyImageInfo{}, errors.New(fmt.Sprintf("MyImage Name ID cannot include reserved string : %s", DEV))
	}

	attachedDiskList, listAttachedDiskErr := listRawAttachedDiskByVmIID(myImageHandler.VpcService, myImageHandler.Ctx, snapshotReqInfo.SourceVM)
	if listAttachedDiskErr != nil {
		return irs.MyImageInfo{}, errors.New(fmt.Sprintf("Failed to List Attached Disk. err = %s", listAttachedDiskErr.Error()))
	}

	start := call.Start()
	mountIndex := 0
	for _, attachedDisk := range (*attachedDiskList).VolumeAttachments {
		mountIndex++
		snapshotName := fmt.Sprintf("%s%s%d", snapshotReqInfo.IId.NameId, DEV, mountIndex)
		createSnapshotOptions := vpcv1.CreateSnapshotOptions{
			Name: &snapshotName,
			SourceVolume: &vpcv1.VolumeIdentityByID{
				ID: attachedDisk.Volume.ID,
			},
		}
		_, _, createSnapshotErr := myImageHandler.VpcService.CreateSnapshotWithContext(myImageHandler.Ctx, &createSnapshotOptions)
		cblogger.Error(createSnapshotErr)
	}
	LoggingInfo(hiscallInfo, start)

	// get myimage info
	converted, convertErr := myImageHandler.GetMyImage(irs.IID{NameId: snapshotReqInfo.IId.NameId})
	if convertErr != nil {
		return irs.MyImageInfo{}, convertErr
	}

	return converted, nil
}

func (myImageHandler *IbmMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.Region, call.MYIMAGE, "MYIMAGE", "ListMyImage()")

	start := call.Start()
	snapshotList, _, listSnapshotErr := myImageHandler.VpcService.ListSnapshotsWithContext(myImageHandler.Ctx, &vpcv1.ListSnapshotsOptions{})
	if listSnapshotErr != nil {
		return nil, listSnapshotErr
	}

	groupByImageResult := make(map[string][]vpcv1.Snapshot)
	for _, snapshot := range snapshotList.Snapshots {
		if strings.Contains(*snapshot.Name, DEV) {
			groupByKey := strings.Split(*snapshot.Name, DEV)[0]
			groupByImageResult[groupByKey] = append(groupByImageResult[groupByKey], snapshot)
		}
	}

	var myImageInfoList []*irs.MyImageInfo
	for _, associatedSnapshots := range groupByImageResult {
		myImage, toMyImageErr := myImageHandler.ToISRMyImage(associatedSnapshots)
		if toMyImageErr != nil {
			return nil, toMyImageErr
		}
		myImageInfoList = append(myImageInfoList, &myImage)
	}
	LoggingInfo(hiscallInfo, start)

	return myImageInfoList, nil
}

func (myImageHandler *IbmMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	if myImageIID.NameId == "" && myImageIID.SystemId == "" {
		return irs.MyImageInfo{}, errors.New("Failed to Get MyImage. err = MyImage Name ID or System ID is required")
	}

	hiscallInfo := GetCallLogScheme(myImageHandler.Region, call.MYIMAGE, myImageIID.NameId, "GetMyImage()")

	start := call.Start()
	myImageList, err := myImageHandler.ListMyImage()
	if err != nil {
		return irs.MyImageInfo{}, errors.New(fmt.Sprintf("Failed to Get MyImage. err = %s", err))
	}

	for _, myImage := range myImageList {
		if myImage.IId.SystemId == myImageIID.SystemId {
			return *myImage, nil
		} else if myImage.IId.NameId == myImageIID.NameId {
			return *myImage, nil
		}
	}
	LoggingInfo(hiscallInfo, start)

	return irs.MyImageInfo{}, errors.New("MyImage not found")
}

func (myImageHandler *IbmMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	if myImageIID.NameId == "" && myImageIID.SystemId == "" {
		return false, errors.New("Failed to Delete MyImage. err = MyImage Name ID or System ID is required")
	}

	hiscallInfo := GetCallLogScheme(myImageHandler.Region, call.MYIMAGE, myImageIID.NameId, "DeleteMyImage()")

	start := call.Start()
	if err := myImageHandler.cleanSnapshotByMyImage(myImageIID); err != nil {
		return false, errors.New(fmt.Sprintf("Failed to Delte MyImage. err = %s", err))
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func (myImageHandler *IbmMyImageHandler) ToISRMyImage(snapshotList []vpcv1.Snapshot) (irs.MyImageInfo, error) {
	if len(snapshotList) == 0 {
		return irs.MyImageInfo{}, errors.New("Cannot find MyImage")
	}

	var myImageNameId string
	var myImageSystemId string
	var sourceVmNameId string
	var sourceVmSystemId string
	myImageStatus := irs.MyImageAvailable
	var myImageCreatedTime time.Time
	for _, snapshot := range snapshotList {
		if *snapshot.Bootable == true {
			myImageNameId = strings.Split(*snapshot.Name, DEV)[0]
			myImageSystemId = *snapshot.ID

			getVolumeOptions := vpcv1.GetVolumeOptions{
				ID: snapshot.SourceVolume.ID,
			}
			rawSourceVolume, _, getSourceVolumeErr := myImageHandler.VpcService.GetVolumeWithContext(myImageHandler.Ctx, &getVolumeOptions)
			sourceVmNameId = "Deleted"
			sourceVmSystemId = "Deleted"
			if getSourceVolumeErr == nil && len((*rawSourceVolume).VolumeAttachments) != 0 {
				sourceVmNameId = *(*rawSourceVolume).VolumeAttachments[0].Instance.Name
				sourceVmSystemId = *(*rawSourceVolume).VolumeAttachments[0].Instance.ID
			}

			myImageCreatedTime = time.Time(*snapshot.CreatedAt).Local()
		}

		if myImageStatus == irs.MyImageAvailable && getSnapshotStatus(*snapshot.LifecycleState) == irs.MyImageUnavailable {
			myImageStatus = irs.MyImageUnavailable
		}
	}

	return irs.MyImageInfo{
		IId:         irs.IID{NameId: myImageNameId, SystemId: myImageSystemId},
		SourceVM:    irs.IID{NameId: sourceVmNameId, SystemId: sourceVmSystemId},
		Status:      myImageStatus,
		CreatedTime: myImageCreatedTime,
	}, nil
}

func (myImageHandler *IbmMyImageHandler) cleanSnapshotByMyImage(myImageIID irs.IID) error {
	snapshotList, _, listSnapshotErr := myImageHandler.VpcService.ListSnapshotsWithContext(myImageHandler.Ctx, &vpcv1.ListSnapshotsOptions{})
	if listSnapshotErr != nil {
		return listSnapshotErr
	}

	myImageNameId := ""
	if myImageIID.NameId != "" {
		myImageNameId = myImageIID.NameId
	} else {
		for _, snapshot := range snapshotList.Snapshots {
			if *snapshot.ID == myImageIID.SystemId {
				myImageNameId = strings.Split(*snapshot.Name, DEV)[0]
			}
		}
	}

	if myImageNameId != "" {
		for _, snapshot := range snapshotList.Snapshots {
			parsed := strings.Split(*snapshot.Name, DEV)[0]
			if parsed == myImageNameId {
				deleteSnapshotOptions := vpcv1.DeleteSnapshotOptions{
					ID: snapshot.ID,
				}
				myImageHandler.VpcService.DeleteSnapshotWithContext(myImageHandler.Ctx, &deleteSnapshotOptions)
			}
		}
	}

	return nil
}

func getSnapshotStatus(status string) irs.MyImageStatus {
	switch status {
	case "deleting", "failed", "pending", "suspended", "updating", "waiting":
		return irs.MyImageUnavailable
	case "stable":
		return irs.MyImageAvailable
	default:
		return irs.MyImageUnavailable
	}
}

func (myImageHandler *IbmMyImageHandler) CheckWindowsImage(myImageIID irs.IID) (bool, error) {
	var getMyImageErr error
	myImage, getMyImageErr := myImageHandler.GetMyImage(myImageIID)
	if getMyImageErr != nil {
		return false, getMyImageErr
	}
	if myImage.Status != irs.MyImageAvailable {
		myImageUnavailableErr := errors.New("Failed to Check Image OS. err = Source Image status is not Available")
		return false, myImageUnavailableErr
	}
	rawSnapshot, _, getRawSnapshotErr := myImageHandler.VpcService.GetSnapshotWithContext(myImageHandler.Ctx, &vpcv1.GetSnapshotOptions{ID: &myImage.IId.SystemId})
	if getRawSnapshotErr != nil {
		return false, getRawSnapshotErr
	}

	isWindows := strings.Contains(strings.ToLower(*rawSnapshot.OperatingSystem.Name), "windows")
	return isWindows, nil
}
