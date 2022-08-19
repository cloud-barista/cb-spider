package resources

import (
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/server"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/snapshot"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strings"
)

type ClouditMyImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

const DEV = "/dev:"

func (myImageHandler *ClouditMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "MYIMAGE", "MYIMAGE", "SnapshotVM()")

	var createErr error

	// MyImage 이름 중복 체크
	exist, err := myImageHandler.getExistMyImageName(snapshotReqInfo.IId.NameId)
	if exist {
		createErr = errors.New(fmt.Sprintf("Failed to Create MyImage. err = %s already exist", snapshotReqInfo.IId.NameId))
		if err != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Create MyImage. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.MyImageInfo{}, createErr
		}
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.MyImageInfo{}, createErr
	}

	// VM, Volume Snapshot 생성 (실패 시 롤백)
	start := call.Start()
	_, createVmSnapshotErr := myImageHandler.createMyImageSnapshots(snapshotReqInfo.IId.NameId, snapshotReqInfo.SourceVM)
	if createVmSnapshotErr != nil {
		createErr = createVmSnapshotErr
		defer func(myImageNameId string) (irs.MyImageInfo, error) {
			cleanErr := myImageHandler.cleanSnapshotsByMyImage(myImageNameId)
			if cleanErr != nil {
				createErr = errors.New(fmt.Sprintf("Failed to Create and Clean MyImage. err = %s", cleanErr.Error()))
			}
			if createErr != nil {
				return irs.MyImageInfo{}, createErr
			}
			return irs.MyImageInfo{}, nil
		}(snapshotReqInfo.IId.NameId)
	}

	// MyImageInfo 반환
	myImageInfo, getMyImageInfoErr := myImageHandler.getMyImageInfo(snapshotReqInfo.IId.NameId)
	if getMyImageInfoErr != nil {
		return irs.MyImageInfo{}, errors.New(fmt.Sprintf("Failed to Get MyImage Info. err = %s", getMyImageInfoErr.Error()))
	}
	LoggingInfo(hiscallInfo, start)

	return myImageInfo, createErr
}

func (myImageHandler *ClouditMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "MYIMAGE", "MYIMAGE", "ListMyImage()")

	myImageHandler.Client.TokenID = myImageHandler.CredentialInfo.AuthToken
	authHeader := myImageHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	start := call.Start()
	vmSnapshotList, err := snapshot.List(myImageHandler.Client, &requestOpts)
	if err != nil {
		return nil, err
	}

	groupByMyImageResult := make(map[string][]snapshot.SnapshotInfo)
	for _, vmSnapshot := range *vmSnapshotList {
		groupByKey := strings.Split(vmSnapshot.Name, DEV)[0]
		groupByMyImageResult[groupByKey] = append(groupByMyImageResult[groupByKey], vmSnapshot)
	}

	var myImageInfoList []*irs.MyImageInfo
	for _, associcatedSnapshots := range groupByMyImageResult {
		myImage, toMyImageErr := snapshot.ToIRSMyImage(myImageHandler.Client, &associcatedSnapshots)
		if toMyImageErr != nil {
			return nil, toMyImageErr
		}
		myImageInfoList = append(myImageInfoList, &myImage)
	}
	LoggingInfo(hiscallInfo, start)

	return myImageInfoList, nil
}

func (myImageHandler *ClouditMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "MYIMAGE", "MYIMAGE", "GetMyImage()")

	if myImageIID.NameId == "" {
		return irs.MyImageInfo{}, errors.New(fmt.Sprintf("Failed to Get MyImage. err = MyImage Name ID is required"))
	}

	start := call.Start()
	myImageList, err := myImageHandler.ListMyImage()
	if err != nil {
		return irs.MyImageInfo{}, err
	}

	for _, myImage := range myImageList {
		if myImage.IId.NameId == myImageIID.NameId {
			return *myImage, nil
		}
	}
	LoggingInfo(hiscallInfo, start)

	return irs.MyImageInfo{}, nil
}

func (myImageHandler *ClouditMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "MYIMAGE", "MYIMAGE", "DeleteMyImage()")

	if myImageIID.NameId == "" {
		return false, errors.New(fmt.Sprintf("Failed to Delete MyImage. err = MyImage Name ID is required"))
	}

	start := call.Start()
	if err := myImageHandler.cleanSnapshotsByMyImage(myImageIID.NameId); err != nil {
		return false, errors.New(fmt.Sprintf("Failed to Delete MyImage. err = %s", err))
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func (myImageHandler *ClouditMyImageHandler) getRawVmSnapshotList() (*[]snapshot.SnapshotInfo, error) {
	myImageHandler.Client.TokenID = myImageHandler.CredentialInfo.AuthToken
	authHeader := myImageHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	vmSnapshotList, err := snapshot.List(myImageHandler.Client, &requestOpts)
	if err != nil {
		return nil, err
	}

	var rawVmSnapshotList []snapshot.SnapshotInfo
	for _, rawSnapshot := range *vmSnapshotList {
		if strings.EqualFold("yes", rawSnapshot.Bootable) {
			rawVmSnapshotList = append(rawVmSnapshotList, rawSnapshot)
		}
	}

	return &rawVmSnapshotList, nil
}

func (myImageHandler *ClouditMyImageHandler) getExistMyImageName(myImageNameId string) (bool, error) {
	if myImageNameId == "" {
		return false, errors.New("MyImage Name ID is required")
	}

	rawVmSnapshotList, err := myImageHandler.getRawVmSnapshotList()
	if err != nil {
		return false, err
	}

	for _, rawVmSnapshot := range *rawVmSnapshotList {
		if strings.EqualFold(myImageNameId, strings.Split(rawVmSnapshot.Name, DEV)[0]) {
			return true, nil
		}
	}

	return false, nil
}

func (myImageHandler *ClouditMyImageHandler) createMyImageSnapshots(myImageNameId string, sourceVm irs.IID) (irs.MyImageInfo, error) {
	// Find VM root volume
	myImageHandler.Client.TokenID = myImageHandler.CredentialInfo.AuthToken
	authHeader := myImageHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if sourceVm.SystemId == "" {
		if sourceVm.NameId == "" {
			return irs.MyImageInfo{}, errors.New("SourceVM Name ID or System ID is required")
		}
		if strings.Contains(sourceVm.NameId, DEV) {
			return irs.MyImageInfo{}, errors.New(fmt.Sprintf("SourceVM Name ID cannot include reserved string: %s", DEV))
		}
		serverList, err := server.List(myImageHandler.Client, &requestOpts)
		if err != nil {
			return irs.MyImageInfo{}, errors.New(fmt.Sprintf("Failed to get VM List. err = %s", err))
		}
		founded := false
		for _, server := range *serverList {
			if server.Name == sourceVm.NameId {
				sourceVm.SystemId = server.ID
				founded = true
				break
			}
		}
		if !founded {
			return irs.MyImageInfo{}, errors.New(fmt.Sprintf("SourceVM with Name ID: %s does not exists", sourceVm.NameId))
		}
	}

	vmVolumeList, err := server.GetRawVmVolumes(myImageHandler.Client, sourceVm.SystemId, &requestOpts)
	if err != nil {
		return irs.MyImageInfo{}, errors.New("Failed to get VM attached volumes")
	}

	// Create snapshot of every volume associated with VM
	for _, vmVolume := range *vmVolumeList {
		myImageNameIdWithDev := fmt.Sprintf("%s%s%s", myImageNameId, DEV, vmVolume.Dev)
		if len(myImageNameIdWithDev) > 45 {
			return irs.MyImageInfo{}, errors.New(fmt.Sprintf("Snapshot name cannot be longer than 45. Generated name: %s", myImageNameIdWithDev))
		}
		snapshotCreateReqInfo := snapshot.SnapshotReqInfo{
			Name:     myImageNameIdWithDev,
			VolumeId: vmVolume.ID,
		}
		snapshotCreateRequestOpts := client.RequestOpts{
			MoreHeaders: authHeader,
			JSONBody:    snapshotCreateReqInfo,
		}
		_, createSnapshotErr := snapshot.CreateSnapshot(myImageHandler.Client, &snapshotCreateRequestOpts)
		if createSnapshotErr != nil {
			return irs.MyImageInfo{}, errors.New(fmt.Sprintf("Failed to Create MyImage. err = %s", createSnapshotErr.Error()))
		}
	}

	// Get MyImageInfo
	result, getResultErr := myImageHandler.getMyImageInfo(myImageNameId)
	if getResultErr != nil {
		return irs.MyImageInfo{}, errors.New(fmt.Sprintf("Failed to Get Create MyImage Result. err = %s", getResultErr.Error()))
	}

	return result, nil
}

func (myImageHandler *ClouditMyImageHandler) getMyImageInfo(myImageNameId string) (irs.MyImageInfo, error) {
	// Get status of all associated snapshot and gather into MyImageInfo
	myImageHandler.Client.TokenID = myImageHandler.CredentialInfo.AuthToken
	authHeader := myImageHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	snapshotList, err := snapshot.List(myImageHandler.Client, &requestOpts)
	if err != nil {
		return irs.MyImageInfo{}, errors.New(fmt.Sprintf("Failed to Get MyImage. err = %s", err.Error()))
	}

	var associatedSnapshots []snapshot.SnapshotInfo
	for _, snapshot := range *snapshotList {
		if strings.Split(snapshot.Name, DEV)[0] == myImageNameId {
			associatedSnapshots = append(associatedSnapshots, snapshot)
		}
	}

	myImageInfo, err := snapshot.ToIRSMyImage(myImageHandler.Client, &associatedSnapshots)
	if err != nil {
		return irs.MyImageInfo{}, errors.New(fmt.Sprintf("Failed to Get MyImage. err = %s", err.Error()))
	}

	return myImageInfo, nil
}

func (myImageHandler *ClouditMyImageHandler) cleanSnapshotsByMyImage(myImageNameId string) error {
	myImageHandler.Client.TokenID = myImageHandler.CredentialInfo.AuthToken
	authHeader := myImageHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	rawVmSnapshotList, err := snapshot.List(myImageHandler.Client, &requestOpts)
	if err != nil {
		return err
	}

	for _, rawVmSnapshot := range *rawVmSnapshotList {
		parsed := strings.Split(rawVmSnapshot.Name, DEV)[0]
		if parsed == myImageNameId {
			snapshot.DeleteSnapshot(myImageHandler.Client, rawVmSnapshot.Id, &requestOpts)
		}
	}

	return nil
}
