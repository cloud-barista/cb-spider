package resources

import (
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/disk"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/server"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/snapshot"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"sort"
	"strings"
	"time"
)

type ClouditMyImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

const DEV = "-dev-"

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
		defer func(myImageIID irs.IID) (irs.MyImageInfo, error) {
			cleanErr := myImageHandler.cleanSnapshotsByMyImage(myImageIID)
			if cleanErr != nil {
				createErr = errors.New(fmt.Sprintf("Failed to Create and Clean MyImage. err = %s", cleanErr.Error()))
			}
			if createErr != nil {
				return irs.MyImageInfo{}, createErr
			}
			return irs.MyImageInfo{}, nil
		}(snapshotReqInfo.IId)
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

	if myImageIID.NameId == "" && myImageIID.SystemId == "" {
		return irs.MyImageInfo{}, errors.New(fmt.Sprintf("Failed to Get MyImage. err = MyImage Name ID or System ID is required"))
	}

	start := call.Start()
	myImageList, err := myImageHandler.ListMyImage()
	if err != nil {
		return irs.MyImageInfo{}, err
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

func (myImageHandler *ClouditMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "MYIMAGE", "MYIMAGE", "DeleteMyImage()")

	if myImageIID.NameId == "" && myImageIID.SystemId == "" {
		return false, errors.New(fmt.Sprintf("Failed to Delete MyImage. err = MyImage Name ID or System ID is required"))
	}

	start := call.Start()
	if err := myImageHandler.cleanSnapshotsByMyImage(myImageIID); err != nil {
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
	if len(myImageNameId) > 35 {
		return irs.MyImageInfo{}, errors.New(fmt.Sprintf("Snapshot name cannot be longer than 35"))
	}

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

	vmHandler := ClouditVMHandler{
		CredentialInfo: myImageHandler.CredentialInfo,
		Client:         myImageHandler.Client,
	}

	rawVm, getRawVmErr := vmHandler.getRawVm(irs.IID{SystemId: sourceVm.SystemId})
	if getRawVmErr != nil {
		return irs.MyImageInfo{}, errors.New("Failed to get Source VM Info")
	}
	if strings.Contains(strings.ToLower(rawVm.Template), "window") &&
		rawVm.State != "STOPPED" {
		return irs.MyImageInfo{}, errors.New("Cannot Create Windows VM Snapshot while Source VM is not Stopped")
	}

	vmVolumeList, err := server.GetRawVmVolumes(myImageHandler.Client, sourceVm.SystemId, &requestOpts)
	if err != nil {
		return irs.MyImageInfo{}, errors.New("Failed to get VM attached volumes")
	}

	// Create snapshot of every volume associated with VM
	for _, vmVolume := range *vmVolumeList {
		myImageNameIdWithDev := fmt.Sprintf("%s%s%s", myImageNameId, DEV, vmVolume.Dev)
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

func (myImageHandler *ClouditMyImageHandler) CreateAssociatedVolumeSnapshots(myImageNameId string, vmNameId string) error {
	// Get status of all associated volumeSnapshot and gather into MyImageInfo
	myImageHandler.Client.TokenID = myImageHandler.CredentialInfo.AuthToken
	authHeader := myImageHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	volumeSnapshotList, err := snapshot.List(myImageHandler.Client, &requestOpts)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to Get Associated Volume Snapshots. err = %s", err.Error()))
	}

	// var rollbackTargets []snapshot.SnapshotInfo
	var creatingVolumeList []string
	for _, volumeSnapshot := range *volumeSnapshotList {
		if strings.Split(volumeSnapshot.Name, DEV)[0] == myImageNameId && volumeSnapshot.Bootable == "no" {
			// rollbackTargets = append(rollbackTargets, volumeSnapshot)
			requestOpts = client.RequestOpts{
				MoreHeaders: authHeader,
				JSONBody: struct {
					VolumeName string `json:"volumeName"`
				}{
					VolumeName: vmNameId + DEV + strings.Split(volumeSnapshot.Name, DEV)[1],
				},
			}
			if result, createVolumeErr := snapshot.CreateVolumeBySnapshot(myImageHandler.Client, volumeSnapshot.Id, &requestOpts); result == false {
				rollbackErr := myImageHandler.rollbackCreateVolumeBySnapshot(myImageNameId)
				if rollbackErr != nil {
					errStrings := []string{createVolumeErr.Error(), rollbackErr.Error()}
					createVolumeErr = errors.New(strings.Join(errStrings, "\n\t"))
				}
				return errors.New(fmt.Sprintf("Failed to Create Associated Volumes by Snapshot. err = %s", createVolumeErr))
			}
			creatingVolumeList = append(creatingVolumeList, vmNameId+DEV+strings.Split(volumeSnapshot.Name, DEV)[1])
		}
	}

	curRetryCnt := 0
	maxRetryCnt := 120 * 60
	for {
		volumeList, getVolumeErr := disk.List(myImageHandler.Client, &requestOpts)
		if getVolumeErr != nil {
			return errors.New(fmt.Sprintf("Failed to Get Volumes. err = %s", err.Error()))
		}

		for _, volume := range *volumeList {
			if len(creatingVolumeList) == 0 {
				return nil
			}
			for index, creatingVolume := range creatingVolumeList {
				if volume.Name == creatingVolume && volume.State == "AVAILABLE" {
					ret := make([]string, 0)
					ret = append(ret, creatingVolumeList[:index]...)
					creatingVolumeList = append(ret, creatingVolumeList[index+1:]...)
					break
				}
			}
		}

		if curRetryCnt > maxRetryCnt {
			return errors.New("Failed to Create Associated Volumes by Snapshot. err = Volume create waiting timeout")
		}

		time.Sleep(1 * time.Second)
		curRetryCnt++
	}
}

func (myImageHandler *ClouditMyImageHandler) AttachAssociatedVolumesToVM(myImageNameId string, targetVmSystemId string) error {
	myImageHandler.Client.TokenID = myImageHandler.CredentialInfo.AuthToken
	authHeader := myImageHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	targetVm, err := server.Get(myImageHandler.Client, targetVmSystemId, &requestOpts)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to Get Target VM. err = %s", err.Error()))
	}

	volumeList, err := disk.List(myImageHandler.Client, &requestOpts)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to Get Volumes. err = %s", err.Error()))
	}

	var attachTarget []disk.DiskInfo
	for _, volume := range *volumeList {
		if strings.Split(volume.Name, DEV)[0] == targetVm.Name && volume.Bootable == "no" {
			attachTarget = append(attachTarget, volume)
		}
	}

	//sort attachTarget
	sort.Slice(attachTarget, func(i, j int) bool {
		return attachTarget[i].Name < attachTarget[j].Name
	})

	var attachingOnProgressVolumeList []string
	for _, volume := range attachTarget {
		attachRequestOpts := client.RequestOpts{
			MoreHeaders: authHeader,
			JSONBody: struct {
				VolumeId string `json:"volumeId"`
				Mode     string `json:"mode"`
			}{
				VolumeId: volume.ID,
				Mode:     "w",
			},
		}

		attachErr := server.AttachVolume(myImageHandler.Client, targetVmSystemId, &attachRequestOpts)
		if attachErr != nil {
			return errors.New(fmt.Sprintf("Attaching Associated Volumes to VM Failed. err = %s", attachErr.Error()))
		}
		attachingOnProgressVolumeList = append(attachingOnProgressVolumeList, volume.ID)
	}

	curRetryCnt := 0
	maxRetryCnt := 120 * 60
	attachFailedList := []string{"Attaching Associated Volumes to VM Failed, Attach waiting timeout (120 minutes): err ="}
	for {
		for index, attachingVolume := range attachingOnProgressVolumeList {
			rawDisk, getDiskErr := disk.Get(myImageHandler.Client, attachingVolume, &requestOpts)
			if getDiskErr != nil {
				return errors.New(fmt.Sprintf("Failed to Get Volume. err = %s", getDiskErr.Error()))
			}

			if rawDisk.State == "IN_USE" {
				ret := make([]string, 0)
				ret = append(ret, attachingOnProgressVolumeList[:index]...)
				attachingOnProgressVolumeList = append(ret, attachingOnProgressVolumeList[index+1:]...)
				break
			}
		}

		if len(attachingOnProgressVolumeList) == 0 {
			return nil
		}

		if curRetryCnt > maxRetryCnt {
			for _, attachingVolume := range attachingOnProgressVolumeList {
				rawDisk, getDiskErr := disk.Get(myImageHandler.Client, attachingVolume, &requestOpts)
				if getDiskErr != nil {
					return errors.New(fmt.Sprintf("Failed to Get Volume. err = %s", getDiskErr.Error()))
				}
				attachFailedList = append(attachFailedList, fmt.Sprintf("Failed Disk Name ID: %s", rawDisk.Name))
			}
			myImageHandler.rollbackCreateVolumeBySnapshot(myImageNameId)
			return errors.New(strings.Join(attachFailedList, "\t\n"))
		}

		time.Sleep(1 * time.Second)
		curRetryCnt++
	}
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

func (myImageHandler *ClouditMyImageHandler) cleanSnapshotsByMyImage(myImageIID irs.IID) error {
	myImageHandler.Client.TokenID = myImageHandler.CredentialInfo.AuthToken
	authHeader := myImageHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	snapshotList, err := snapshot.List(myImageHandler.Client, &requestOpts)
	if err != nil {
		return err
	}

	myImageNameId := ""
	if myImageIID.NameId != "" {
		myImageNameId = myImageIID.NameId
	} else {
		for _, rawVmSnapshot := range *snapshotList {
			if rawVmSnapshot.Id == myImageIID.SystemId {
				myImageNameId = strings.Split(rawVmSnapshot.Name, DEV)[0]
			}
		}
	}

	if myImageNameId != "" {
		for _, rawVmSnapshot := range *snapshotList {
			parsed := strings.Split(rawVmSnapshot.Name, DEV)[0]
			if parsed == myImageNameId {
				snapshot.DeleteSnapshot(myImageHandler.Client, rawVmSnapshot.Id, &requestOpts)
			}
		}
	}

	return nil
}

func (myImageHandler *ClouditMyImageHandler) rollbackCreateVolumeBySnapshot(myImageNameId string) error {
	myImageHandler.Client.TokenID = myImageHandler.CredentialInfo.AuthToken
	authHeader := myImageHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	volumeList, getVolumeListErr := disk.List(myImageHandler.Client, &requestOpts)
	if getVolumeListErr != nil {
		return errors.New(fmt.Sprintf("Failed to List Volumes. err = %s", getVolumeListErr.Error()))
	}

	var rollbackTargets []disk.DiskInfo
	for _, volume := range *volumeList {
		if strings.Split(volume.Name, DEV)[0] == myImageNameId {
			rollbackTargets = append(rollbackTargets, volume)
		}
	}

	curRetryCnt := 0
	maxRetryCnt := 120
	rollbackFailedList := []string{"Create Volume By Snapshot Failed: err = "}
	for {
		for index, target := range rollbackTargets {
			deleteErr := disk.Delete(myImageHandler.Client, target.ID, &requestOpts)
			if deleteErr == nil {
				ret := make([]disk.DiskInfo, 0)
				ret = append(ret, rollbackTargets[:index]...)
				rollbackTargets = append(ret, rollbackTargets[index+1:]...)
			}
			if curRetryCnt > maxRetryCnt {
				rollbackFailedList = append(rollbackFailedList, fmt.Sprintf("Volume ID: %s: %s", target.Name, deleteErr.Error()))
			}
		}

		if curRetryCnt > maxRetryCnt {
			break
		}

		time.Sleep(1 * time.Second)
		curRetryCnt++
	}

	if len(rollbackFailedList) > 1 {
		return errors.New(strings.Join(rollbackFailedList, "\n\t"))
	}

	return nil
}

func (myImageHandler *ClouditMyImageHandler) CheckWindowsImage(myImageIID irs.IID) (bool, error) {
	imageHandler := ClouditImageHandler{
		CredentialInfo: myImageHandler.CredentialInfo,
		Client:         myImageHandler.Client,
	}
	rawRootImage, getRawRootImageErr := imageHandler.GetRawRootImage(myImageIID, true)
	if getRawRootImageErr != nil {
		return false, errors.New(fmt.Sprintf("Failed to Check Windows Image. err = %s", getRawRootImageErr.Error()))
	}

	isWindows := strings.Contains(strings.ToLower(rawRootImage.OS), "windows")
	return isWindows, nil
}
