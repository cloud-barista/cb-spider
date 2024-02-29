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
	ktVM, err := vmHandler.getKTCloudVM(snapshotReqInfo.SourceVM.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM Info from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.MyImageInfo{}, newErr
	}
	osTypeId := strconv.Itoa(ktVM.OsTypeId)

	// Get VolumeIds of the VM
	diskHandler := KtCloudDiskHandler{
		RegionInfo: myImageHandler.RegionInfo,
		Client:   	myImageHandler.Client,
	}
	volumeId, err := diskHandler.getRootVolumeIdWithVMId(snapshotReqInfo.SourceVM.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Volume Info from KT Cloud : [%v]", err)
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
		newErr := fmt.Errorf("Failed to Create New Image. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	cblogger.Info("### Waiting for the Image to be Created(600sec)!!\n")
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
	
	ktImages, err := myImageHandler.listKTImages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud Image List!! [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	var imageInfoList []*irs.MyImageInfo
    for _, ktImage := range *ktImages {
		imgInfo, err := myImageHandler.mappingMyImageInfo(&ktImage)
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
		return irs.MyImageInfo{}, newErr
	}

	ktImage, err := myImageHandler.getKTImage(myImageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud Image Info!! [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.MyImageInfo{}, newErr
	}

	imgInfo, mapErr := myImageHandler.mappingMyImageInfo(ktImage)
	if mapErr != nil {
		newErr := fmt.Errorf("Failed to Map the Image Info. [%v]", mapErr)
		cblogger.Error(newErr.Error())
		return irs.MyImageInfo{}, newErr
	}
	return *imgInfo, nil
}

func (myImageHandler *KtCloudMyImageHandler) CheckWindowsImage(myImageIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called CheckWindowsImage()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Image SystemId!!")
		cblogger.Error(newErr.Error())
		// LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	ktImages, err := myImageHandler.listKTImages()
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

func (myImageHandler *KtCloudMyImageHandler) mappingMyImageInfo(myImage *ktsdk.Template) (*irs.MyImageInfo, error) {
	cblogger.Info("KT Cloud Driver: called mappingMyImageInfo()!")
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
		SourceVM: irs.IID{
			NameId:   "N/A",
			SystemId: "N/A",
		},
		Status: 	  convertImageStatus(myImage.IsReady),
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

func convertImageStatus(isReady bool) irs.MyImageStatus {
	cblogger.Info("KT Cloud Driver: called convertImageStatus()")

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

// # Get 'MyImage' Info List from KT Cloud
func (myImageHandler *KtCloudMyImageHandler) listKTImages() (*[]ktsdk.Template, error) {
	cblogger.Info("KT Cloud Driver: called listKTImages()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, "listKTImages()", "listKTImages()")
	
	// TemplateFilter => 'self' : image created by the user. 'selfexecutable' : created by the user and currently available.
	imgReq := ktsdk.ListTemplateReqInfo{
		TemplateFilter: "self",
	}
	start := call.Start()
	imgResp, err := myImageHandler.Client.ListTemplates(&imgReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get My Image(Image Templage) List from KT Cloud : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, start)

	if len(imgResp.Listtemplatesresponse.Template) < 1 {
		newErr := fmt.Errorf("# KT Cloud My Image Template does Not Exist!!")
		cblogger.Debug(newErr.Error())
		return nil, newErr
	}	
	// spew.Dump(imgResp.Listtemplatesresponse.Template)
	return &imgResp.Listtemplatesresponse.Template, nil
}

// # Get 'MyImage' Info from KT Cloud
func (myImageHandler *KtCloudMyImageHandler) getKTImage(myImageIID irs.IID) (*ktsdk.Template, error) {
	cblogger.Info("KT Cloud Driver: called getKTImage()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	ktImages, err := myImageHandler.listKTImages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud Image Template List!! [%v]", err)
		cblogger.Debug(newErr.Error())
		return nil, newErr
	}

	var imgInfo ktsdk.Template
	if len(*ktImages) < 1 {
		newErr := fmt.Errorf("# KT Cloud My Image Template does Not Exist!!")
		cblogger.Debug(newErr.Error())
		return nil, nil // Not Return Error
	} else {
		for _, ktImage := range *ktImages {
			// cblogger.Infof("\n# ktImage.ID : %s", ktImage.ID)		
			if strings.EqualFold(ktImage.ID, myImageIID.SystemId) {
				imgInfo = ktImage
				return &imgInfo, nil
			}
		}
		if imgInfo.ID == "" {
			newErr := fmt.Errorf("Failed to Find any My Image(Image Template) with the Image ID!!")
			cblogger.Error(newErr.Error())
			return nil, newErr
		}
	}	
	return nil, nil
}

func (myImageHandler *KtCloudMyImageHandler) isPublicImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called isPublicImage()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, imageIID.SystemId, "isPublicImage()")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Image SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}	

	isPublicImage := false
	ktMyImage, err := myImageHandler.getKTImage(imageIID) // In case of 'MyImage'
	if err != nil {
		cblogger.Infof("MyImage(Image Template) having the ID does Not Exist!! [%v]", err)

		imageHandler := KtCloudImageHandler{
			RegionInfo: myImageHandler.RegionInfo,
			Client:    	myImageHandler.Client,
		}	
		productType, err := imageHandler.getKTProductType(imageIID) // In case of 'Public Image'
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the ProductType Info : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return false, newErr
		} else {
			if strings.EqualFold(productType.TemplateId, imageIID.SystemId) { // Not productType.ProductId 
				isPublicImage = true
			}
			return isPublicImage, nil
		}
	} else if ktMyImage.IsPublic {
		isPublicImage = true
	}	
	return isPublicImage, nil
}
