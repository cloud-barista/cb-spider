package resources

import (
	"context"
	"errors"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	compute "google.golang.org/api/compute/v1"
)

type GCPMyImageHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

const (
	GCPMyImageReady string = "READY"
)

func (MyImageHandler *GCPMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(MyImageHandler.Region, call.MYIMAGE, snapshotReqInfo.IId.NameId, "SnapshotVM()")
	start := call.Start()

	projectID := MyImageHandler.Credential.ProjectID
	zone := MyImageHandler.Region.Zone
	myImageName := snapshotReqInfo.IId.NameId

	machineImage := &compute.MachineImage{
		SourceInstance: "projects/" + projectID + "/zones/" + zone + "/instances/" + snapshotReqInfo.SourceVM.SystemId,
		Name:           myImageName,
	}

	op, err := MyImageHandler.Client.MachineImages.Insert(projectID, machineImage).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.MyImageInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	WaitUntilComplete(MyImageHandler.Client, projectID, "", op.Name, true)

	myImageInfo, errMyImage := MyImageHandler.GetMyImage(irs.IID{NameId: myImageName, SystemId: myImageName})
	if errMyImage != nil {
		cblogger.Error(errMyImage)
		return irs.MyImageInfo{}, errMyImage
	}

	return myImageInfo, nil
}

func (MyImageHandler *GCPMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(MyImageHandler.Region, call.MYIMAGE, "MyImage", "ListMyImage()")
	start := call.Start()

	myImageInfoList := []*irs.MyImageInfo{}

	projectID := MyImageHandler.Credential.ProjectID

	myImageList, err := MyImageHandler.Client.MachineImages.List(projectID).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	calllogger.Info(call.String(hiscallInfo))

	for _, myImage := range myImageList.Items {
		myImageInfo, err := MyImageHandler.convertMyImageInfo(myImage)
		if err != nil {
			cblogger.Error(err)
			continue
		}
		myImageInfoList = append(myImageInfoList, &myImageInfo)
	}

	return myImageInfoList, nil
}

func (MyImageHandler *GCPMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(MyImageHandler.Region, call.MYIMAGE, myImageIID.NameId, "GetMyImage()")
	start := call.Start()

	projectID := MyImageHandler.Credential.ProjectID

	myImageResp, err := GetMachineImageInfo(MyImageHandler.Client, projectID, myImageIID.SystemId)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.MyImageInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	myImageInfo, errMyImage := MyImageHandler.convertMyImageInfo(myImageResp)
	if errMyImage != nil {
		cblogger.Error(errMyImage)
		return irs.MyImageInfo{}, errMyImage
	}

	return myImageInfo, nil
}

func (MyImageHandler *GCPMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(MyImageHandler.Region, call.MYIMAGE, myImageIID.NameId, "DeleteMyImage()")
	start := call.Start()

	projectID := MyImageHandler.Credential.ProjectID
	myImage := myImageIID.SystemId

	op, err := MyImageHandler.Client.MachineImages.Delete(projectID, myImage).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	calllogger.Info(call.String(hiscallInfo))

	WaitUntilComplete(MyImageHandler.Client, projectID, "", op.Name, true)

	return true, nil
}

func (MyImageHandler *GCPMyImageHandler) convertMyImageInfo(myImageResp *compute.MachineImage) (irs.MyImageInfo, error) {
	myImageInfo := irs.MyImageInfo{}

	myImageInfo.IId = irs.IID{NameId: myImageResp.Name, SystemId: myImageResp.Name}

	arrSourceVM := strings.Split(myImageResp.SourceInstance, "/")
	sourceVM := arrSourceVM[len(arrSourceVM)-1]

	myImageInfo.SourceVM = irs.IID{SystemId: sourceVM}

	myImageStatus, err := convertMyImageStatus(myImageResp.Status)
	if err != nil {
		cblogger.Error(err)
		return irs.MyImageInfo{}, err
	}

	myImageInfo.Status = myImageStatus

	myImageInfo.CreatedTime, _ = time.Parse(time.RFC3339, myImageResp.CreationTimestamp)

	return myImageInfo, nil
}

func convertMyImageStatus(status string) (irs.MyImageStatus, error) {
	var returnStatus irs.MyImageStatus

	if status == GCPMyImageReady {
		returnStatus = irs.MyImageAvailable
	} else {
		returnStatus = irs.MyImageUnavailable
	}

	return returnStatus, nil

}

// https://cloud.google.com/compute/docs/reference/rest/beta/machineImages/list
// machine Image에서 os속성이 없음.
func (MyImageHandler *GCPMyImageHandler) CheckWindowsImage(myImageIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(MyImageHandler.Region, call.MYIMAGE, myImageIID.NameId, "CheckWindowsImage()")
	start := call.Start()

	projectID := MyImageHandler.Credential.ProjectID
	isWindows := false
	machineImage, err := GetMachineImageInfo(MyImageHandler.Client, projectID, myImageIID.SystemId)
	hiscallInfo.ElapsedTime = call.Elapsed(start)

	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return isWindows, err
	}
	calllogger.Info(call.String(hiscallInfo))

	ip := machineImage.InstanceProperties
	disks := ip.Disks
	for _, disk := range disks {
		if disk.Boot { // Boot Device
			osFeatures := disk.GuestOsFeatures
			for _, feature := range osFeatures {
				if feature.Type == "WINDOWS" {
					isWindows = true
					return isWindows, nil

				}
			}
			cblogger.Debug(isWindows)
		}
	}

	return isWindows, nil
}

func (ImageHandler *GCPMyImageHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}
