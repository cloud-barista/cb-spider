package resources

import (
	"context"
	"strings"
	"time"

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
	projectID := MyImageHandler.Credential.ProjectID
	zone := MyImageHandler.Region.Zone
	myImageName := snapshotReqInfo.IId.NameId

	machineImage := &compute.MachineImage{
		SourceInstance: "projects/" + projectID + "/zones/" + zone + "/instances/" + snapshotReqInfo.SourceVM.SystemId,
		Name:           myImageName,
	}

	op, err := MyImageHandler.Client.MachineImages.Insert(projectID, machineImage).Do()
	if err != nil {
		cblogger.Error(err)
		return irs.MyImageInfo{}, err
	}

	WaitUntilComplete(MyImageHandler.Client, projectID, "", op.Name, true)

	myImageInfo, errMyImage := MyImageHandler.GetMyImage(irs.IID{NameId: myImageName, SystemId: myImageName})
	if errMyImage != nil {
		cblogger.Error(errMyImage)
		return irs.MyImageInfo{}, errMyImage
	}

	return myImageInfo, nil
}

func (MyImageHandler *GCPMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	myImageInfoList := []*irs.MyImageInfo{}

	projectID := MyImageHandler.Credential.ProjectID

	myImageList, err := MyImageHandler.Client.MachineImages.List(projectID).Do()
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

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
	projectID := MyImageHandler.Credential.ProjectID

	myImageResp, err := GetMachineImageInfo(MyImageHandler.Client, projectID, myImageIID.SystemId)
	if err != nil {
		cblogger.Error(err)
		return irs.MyImageInfo{}, err
	}

	myImageInfo, errMyImage := MyImageHandler.convertMyImageInfo(myImageResp)
	if errMyImage != nil {
		cblogger.Error(errMyImage)
		return irs.MyImageInfo{}, errMyImage
	}

	return myImageInfo, nil
}

func (MyImageHandler *GCPMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	projectID := MyImageHandler.Credential.ProjectID
	myImage := myImageIID.SystemId

	op, err := MyImageHandler.Client.MachineImages.Delete(projectID, myImage).Do()
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

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
	projectID := MyImageHandler.Credential.ProjectID
	isWindows := false
	machineImage, err := GetMachineImageInfo(MyImageHandler.Client, projectID, myImageIID.SystemId)

	if err != nil {
		return isWindows, err
	}

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
			cblogger.Info(isWindows)
		}
	}

	//return false, fmt.Errorf("Does not support CheckWindowsImage() yet!!")
	return isWindows, nil
}
