package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/IBM/platform-services-go-sdk/globalsearchv2"
	"github.com/IBM/platform-services-go-sdk/globaltaggingv1"
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
	TaggingService *globaltaggingv1.GlobalTaggingV1
	SearchService  *globalsearchv2.GlobalSearchV2
}

const DEV = "-dev-"

func (myImageHandler *IbmMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.Region, call.MYIMAGE, snapshotReqInfo.IId.NameId, "SnapshotVM()")

	if len(snapshotReqInfo.IId.NameId) > 55 {
		createErr := errors.New(fmt.Sprintf("Failed to SnapshotVM VM. err = MyImage Name ID cannot be longer than 55 characters"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.MyImageInfo{}, createErr
	}
	if strings.Contains(snapshotReqInfo.IId.NameId, DEV) {
		createErr := errors.New(fmt.Sprintf("Failed to SnapshotVM VM. err = MyImage Name ID cannot include reserved string : %s", DEV))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.MyImageInfo{}, createErr
	}

	attachedDiskList, listAttachedDiskErr := listRawAttachedDiskByVmIID(myImageHandler.VpcService, myImageHandler.Ctx, snapshotReqInfo.SourceVM)
	if listAttachedDiskErr != nil {
		createErr := errors.New(fmt.Sprintf("Failed to SnapshotVM VM. err = %s", listAttachedDiskErr.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.MyImageInfo{}, createErr
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
		if createSnapshotErr != nil {
			createErr := errors.New(fmt.Sprintf("Failed to SnapshotVM VM. err = %s", createSnapshotErr.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.MyImageInfo{}, createErr
		}
	}
	LoggingInfo(hiscallInfo, start)

	// get myimage info
	converted, convertErr := myImageHandler.GetMyImage(irs.IID{NameId: snapshotReqInfo.IId.NameId})
	if convertErr != nil {
		createErr := errors.New(fmt.Sprintf("Failed to SnapshotVM VM. err = %s", convertErr.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.MyImageInfo{}, createErr
	}

	// Attach Tag
	if snapshotReqInfo.TagList != nil && len(snapshotReqInfo.TagList) > 0 {
		var tagHandler irs.TagHandler // TagHandler 초기화
		for _, tag := range snapshotReqInfo.TagList {
			_, err := tagHandler.AddTag("MYIMAGE", snapshotReqInfo.IId, tag)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to Attach Tag on MyImage err = %s", err.Error()))
				cblogger.Error(createErr.Error())
			}
		}
	}

	return converted, nil
}

func (myImageHandler *IbmMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.Region, call.MYIMAGE, "MYIMAGE", "ListMyImage()")

	start := call.Start()
	snapshotList, _, listSnapshotErr := myImageHandler.VpcService.ListSnapshotsWithContext(myImageHandler.Ctx, &vpcv1.ListSnapshotsOptions{})
	if listSnapshotErr != nil {
		createErr := errors.New(fmt.Sprintf("Failed to List MyImage. err = %s", listSnapshotErr.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return nil, createErr
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
			createErr := errors.New(fmt.Sprintf("Failed to List MyImage. err = %s", toMyImageErr.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return nil, createErr
		}
		myImageInfoList = append(myImageInfoList, &myImage)
	}
	LoggingInfo(hiscallInfo, start)

	return myImageInfoList, nil
}

// Create Function: GetRawMyImage
// Return Type: vpc.snapshot
// Using: TagHandler
func (myImageHandler *IbmMyImageHandler) GetRawMyImage(myImageIID irs.IID) (*vpcv1.Snapshot, error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.Region, call.MYIMAGE, "MYIMAGE", "GetRawMyImage()")
	start := call.Start()

	myImage, err := myImageHandler.GetMyImage(myImageIID)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to Get MyImage. err = %s", err.Error()))
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	snapshot, _, err := myImageHandler.VpcService.GetSnapshotWithContext(myImageHandler.Ctx, &vpcv1.GetSnapshotOptions{ID: &myImage.IId.SystemId})
	if err != nil {
		err := errors.New(fmt.Sprintf("Failed to Get MyImage. err = %s", err.Error()))
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	LoggingInfo(hiscallInfo, start)

	return snapshot, nil
}

func (myImageHandler *IbmMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.Region, call.MYIMAGE, myImageIID.NameId, "GetMyImage()")
	start := call.Start()
	if myImageIID.NameId == "" && myImageIID.SystemId == "" {
		createErr := errors.New(fmt.Sprintf("Failed to get raw info of the MyImage. err = MyImage Name ID or System ID is required"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.MyImageInfo{}, createErr
	}

	myImageList, err := myImageHandler.ListMyImage()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to get raw info of the MyImage. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.MyImageInfo{}, createErr
	}

	for _, myImage := range myImageList {
		if myImage.IId.SystemId == myImageIID.SystemId {
			return *myImage, nil
		} else if myImage.IId.NameId == myImageIID.NameId {
			return *myImage, nil
		}
	}
	LoggingInfo(hiscallInfo, start)

	return irs.MyImageInfo{}, errors.New(fmt.Sprintf("Failed to Get MyImage. err = MyImage not found"))
}

func (myImageHandler *IbmMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.Region, call.MYIMAGE, myImageIID.NameId, "DeleteMyImage()")
	start := call.Start()
	if myImageIID.NameId == "" && myImageIID.SystemId == "" {
		delErr := errors.New(fmt.Sprintf("Failed to Delete MyImage. err = MyImage Name ID or System ID is required"))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	if err := myImageHandler.cleanSnapshotByMyImage(myImageIID); err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete MyImage. err = %s", err))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)

	// Detach Tag Auto Delete
	var tagService *globaltaggingv1.GlobalTaggingV1
	deleteTagAllOptions := tagService.NewDeleteTagAllOptions()
	deleteTagAllOptions.SetTagType("user")

	_, _, err := tagService.DeleteTagAll(deleteTagAllOptions)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete MyImage Detached Tag err = %s", err.Error()))
		cblogger.Error(delErr.Error())
	}

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
	hiscallInfo := GetCallLogScheme(myImageHandler.Region, call.MYIMAGE, myImageIID.NameId, "CheckWindowsImage()")
	start := call.Start()
	var getMyImageErr error
	myImage, getMyImageErr := myImageHandler.GetMyImage(myImageIID)
	if getMyImageErr != nil {
		checkWindowsImageErr := errors.New(fmt.Sprintf("Failed to CheckWindowsImage By MyImage. err = %s", getMyImageErr.Error()))
		cblogger.Error(checkWindowsImageErr.Error())
		LoggingError(hiscallInfo, checkWindowsImageErr)
		return false, checkWindowsImageErr
	}
	if myImage.Status != irs.MyImageAvailable {
		checkWindowsImageErr := errors.New(fmt.Sprintf("Failed to CheckWindowsImage By MyImage. err = Source Image status is not Available"))
		cblogger.Error(checkWindowsImageErr.Error())
		LoggingError(hiscallInfo, checkWindowsImageErr)
		return false, checkWindowsImageErr
	}
	rawSnapshot, _, getRawSnapshotErr := myImageHandler.VpcService.GetSnapshotWithContext(myImageHandler.Ctx, &vpcv1.GetSnapshotOptions{ID: &myImage.IId.SystemId})
	if getRawSnapshotErr != nil {
		checkWindowsImageErr := errors.New(fmt.Sprintf("Failed to CheckWindowsImage By MyImage. err = %s", getRawSnapshotErr.Error()))
		cblogger.Error(checkWindowsImageErr.Error())
		LoggingError(hiscallInfo, checkWindowsImageErr)
		return false, checkWindowsImageErr
	}

	isWindows := strings.Contains(strings.ToLower(*rawSnapshot.OperatingSystem.Name), "windows")
	LoggingInfo(hiscallInfo, start)
	return isWindows, nil
}
