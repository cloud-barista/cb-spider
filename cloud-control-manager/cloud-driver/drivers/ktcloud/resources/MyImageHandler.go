package resources

import (
	"fmt"
	"strings"
	"strconv"
	"time"
	// "google.golang.org/grpc/metadata"
	// "github.com/davecgh/go-spew/spew"

	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type KtCloudMyImageHandler struct {
	RegionInfo     idrv.RegionInfo
	Client         *ktsdk.KtCloudClient
}

// To Take a Snapshot with a VM ID (To Create My Image) 
func (myImageHandler *KtCloudMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	cblogger.Info("KT Cloud Driver: called SnapshotVM()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, snapshotReqInfo.SourceVM.SystemId, "SnapshotVM()")

	if strings.EqualFold(snapshotReqInfo.SourceVM.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}

	if strings.EqualFold(snapshotReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid Disk NameId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}

	// Get OSTypeId of the VM
	vmHandler := KtCloudVMHandler{
		RegionInfo: myImageHandler.RegionInfo,
		Client:   	myImageHandler.Client,
	}
	ktVM, err := vmHandler.GetKTCloudVM(snapshotReqInfo.SourceVM.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM Info from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.MyImageInfo{}, newErr
	}
	osTypeId := strconv.Itoa(ktVM.OsTypeId)

	// Get VolumeId of the VM
	diskHandler := KtCloudDiskHandler{
		RegionInfo: myImageHandler.RegionInfo,
		Client:   	myImageHandler.Client,
	}
	volumeId, err := diskHandler.GetVolumeIdWithVMid(snapshotReqInfo.SourceVM.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM Info from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.MyImageInfo{}, newErr
	}

	// Create Server Image (Template)
	imgReq := ktsdk.CreateTemplateReqInfo{
		Name:           snapshotReqInfo.IId.NameId,  		// Required
		DisplayText: 	snapshotReqInfo.IId.NameId,  		// Required
		OsTypeId: 		osTypeId, 							// Required
		VolumeId: 		volumeId, 							// Required
		VMId:        	snapshotReqInfo.SourceVM.SystemId,
	}
	start := call.Start()
	imgResp, err := myImageHandler.Client.CreateTemplate(&imgReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New Disk Volume. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	cblogger.Info("### Waiting for the Disk to be Created(600sec)!!\n")
	waitErr := myImageHandler.Client.WaitForAsyncJob(imgResp.Createtemplateresponse.JobId, 600000000000)
	if waitErr != nil {
		cblogger.Errorf("Failed to Wait the Job : [%v]", waitErr)
		return irs.MyImageInfo{}, waitErr
	}

	newImgIID := irs.IID{SystemId: imgResp.Createtemplateresponse.ID}
	myImageInfo, err := myImageHandler.GetMyImage(newImgIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait to Get Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	return myImageInfo, nil
}

func (myImageHandler *KtCloudMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	cblogger.Info("KT Cloud Driver: called ListMyImage()")
	
	ktImages, err := myImageHandler.ListKTImages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud Image List!! [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	var imageInfoList []*irs.MyImageInfo
    for _, ktImage := range *ktImages {
		imgInfo, err := myImageHandler.MappingMyImageInfo(&ktImage)
		if err != nil {
			newErr := fmt.Errorf("Failed to Map the Image Info. [%v]", err)
			cblogger.Error(newErr.Error())
			return nil, newErr
		}
		imageInfoList = append(imageInfoList, imgInfo)
    }
	return imageInfoList, nil
}

func (myImageHandler *KtCloudMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	cblogger.Info("KT Cloud Driver: called GetMyImage()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		// LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}

	ktImages, err := myImageHandler.ListKTImages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud Image List!! [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.MyImageInfo{}, newErr
	}

	var imgInfo *irs.MyImageInfo
	var mapErr error
    for _, ktImage := range *ktImages {
		// cblogger.Infof("\n# ktImage.ID : %s", ktImage.ID)		
		if strings.EqualFold(ktImage.ID, myImageIID.SystemId) {
			imgInfo, mapErr = myImageHandler.MappingMyImageInfo(&ktImage)
			if mapErr != nil {
				newErr := fmt.Errorf("Failed to Map the Image Info. [%v]", mapErr)
				cblogger.Error(newErr.Error())
				return irs.MyImageInfo{}, newErr
			}
			return *imgInfo, nil
		}		
    }
	if imgInfo == nil {
		newErr := fmt.Errorf("Failed to Find the Image Info with the Image ID!!")
		cblogger.Error(newErr.Error())
		return irs.MyImageInfo{}, newErr
	}
	return irs.MyImageInfo{}, nil
}

func (myImageHandler *KtCloudMyImageHandler) CheckWindowsImage(myImageIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called CheckWindowsImage()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Image SystemId!!")
		cblogger.Error(newErr.Error())
		// LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	ktImages, err := myImageHandler.ListKTImages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud Image List!! [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	var osType string
    for _, ktImage := range *ktImages {
		// cblogger.Infof("\n# ktImage.ID : %s", ktImage.ID)		
		if strings.EqualFold(ktImage.ID, myImageIID.SystemId) {
			osType = ktImage.OSTypeName
			break
		}		
    }
	if strings.EqualFold(osType, "") {
		newErr := fmt.Errorf("Failed to Find the Image Info!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	isWindowsImage := false
	if strings.Contains(osType, "Windows") {
		isWindowsImage = true
	}
	return isWindowsImage, nil
}

func (myImageHandler *KtCloudMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called DeleteMyImage()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "DeleteMyImage()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	start := call.Start()
	delResult, err := myImageHandler.Client.DeleteTemplate(myImageIID.SystemId, myImageHandler.RegionInfo.Zone)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Disk Volume!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, start)
	// cblogger.Info("\n\n### delResult : ")
	// spew.Dump(delResult)

	cblogger.Info("### Waiting for the Disk to be Created(600sec)!!\n")
	waitErr := myImageHandler.Client.WaitForAsyncJob(delResult.Deletetemplateresponse.JobId, 600000000000)
	if waitErr != nil {
		cblogger.Errorf("Failed to Wait the Job : [%v]", waitErr)
		return false, waitErr
	}

	return true, nil
}

func (myImageHandler *KtCloudMyImageHandler) MappingMyImageInfo(myImage *ktsdk.Template) (*irs.MyImageInfo, error) {
	cblogger.Info("KT Cloud Driver: called MappingMyImageInfo()!")
	// cblogger.Info("\n\n### myImage : ")
	// spew.Dump(myImage)
	// cblogger.Info("\n")

	var convertedTime time.Time
	var convertErr error
	if !strings.EqualFold(myImage.Created, "") {
		convertedTime, convertErr = convertTimeFormat(myImage.Created)
		if convertErr != nil {
			newErr := fmt.Errorf("Failed to Convert the Time Format!! : [%v]", convertErr)
			cblogger.Error(newErr.Error())
			return nil, newErr
		}
	}
	
	myImageInfo := &irs.MyImageInfo {
		IId: irs.IID{
			NameId:   myImage.Name,
			SystemId: myImage.ID,
		},
		Status: 	  ConvertImageStatus(myImage.IsReady),
		CreatedTime:  convertedTime,
	}

	keyValueList := []irs.KeyValue{
		{Key: "OSType", Value: myImage.OSTypeName},
		{Key: "DiskSize(GB)", Value: strconv.Itoa(myImage.Size/(1024*1024*1024))},
		{Key: "Region", Value: myImageHandler.RegionInfo.Region},
	}
	myImageInfo.KeyValueList = keyValueList
	return myImageInfo, nil
}

func ConvertImageStatus(isReady bool) irs.MyImageStatus {
	cblogger.Info("KT Cloud Driver: called ConvertImageStatus()")

	var resultStatus irs.MyImageStatus
	switch isReady {
	case true:
		resultStatus = irs.MyImageAvailable
	case false :
		resultStatus = irs.MyImageUnavailable
	default:
		resultStatus = "Unknown"
	}
	return resultStatus
}

func (myImageHandler *KtCloudMyImageHandler) ListKTImages() (*[]ktsdk.Template, error) {
	cblogger.Info("KT Cloud Driver: called GetKTCloudNLB()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, "ListKTImages()", "ListKTImages()")
	
	// TemplateFilter => 'self' : image created by the user. 'selfexecutable' : created by the user and currently available.
	imgReq := ktsdk.ListTemplateReqInfo{
		TemplateFilter: "self",
	}
	start := call.Start()
	imgResp, err := myImageHandler.Client.ListTemplates(&imgReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Image Templage List from KT Cloud : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, start)

	if len(imgResp.Listtemplatesresponse.Template) < 1 {
		cblogger.Info("# KT Cloud Image Template does Not Exist!!")
		return nil, nil // Not Return Error
	}
	// spew.Dump(imgResp.Listtemplatesresponse.Template)
	return &imgResp.Listtemplatesresponse.Template, nil
}
