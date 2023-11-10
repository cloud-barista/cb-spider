// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2023.08.

package resources

import (
	"fmt"
	"strings"
	"time"
	// "github.com/davecgh/go-spew/spew"

	server "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/server"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpMyImageHandler struct {
	RegionInfo    idrv.RegionInfo
	VMClient      *server.APIClient
}

func (myImageHandler *NcpMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called SnapshotVM()")

	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, snapshotReqInfo.IId.SystemId, "SnapshotVM()")

	if strings.EqualFold(snapshotReqInfo.SourceVM.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}

	snapshotReq := server.CreateMemberServerImageRequest{ // Not CreateBlockStorageSnapshotInstanceRequest{}
		MemberServerImageName:		&snapshotReqInfo.IId.NameId,
		ServerInstanceNo: 			&snapshotReqInfo.SourceVM.SystemId,
	}
	callLogStart := call.Start()
	result, err := myImageHandler.VMClient.V2Api.CreateMemberServerImage(&snapshotReq) // Not CreateBlockStorageSnapshotInstance
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New VM Snapshot : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.MemberServerImageList) < 1 {
		newErr := fmt.Errorf("Failed to Create New VM Snapshot. Snapshot does Not Exist!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	} else {
		cblogger.Info("Succeeded in Creating New Snapshot.")
	}

	newImageIID := irs.IID{SystemId: *result.MemberServerImageList[0].MemberServerImageNo}
	// To Wait for Creating a Snapshot Image
	curStatus, err := myImageHandler.WaitForImageSnapshot(newImageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait for Image Snapshot. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	cblogger.Infof("==> Image Status of the Snapshot : [%s]", string(curStatus))

	myImageInfo, err := myImageHandler.GetMyImage(newImageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get MyImage Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	return myImageInfo, nil
}

// To Manage My Images
func (myImageHandler *NcpMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called ListMyImage()")

	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, "ListMyImage()", "ListMyImage()")
	
	vmHandler := NcpVMHandler{
		RegionInfo:     	myImageHandler.RegionInfo,
		VMClient:         	myImageHandler.VMClient,
	}
	regionNo, err := vmHandler.GetRegionNo(myImageHandler.RegionInfo.Region)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Region No of the Region Code: [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	imageReq := server.GetMemberServerImageListRequest{ // Not GetBlockStorageSnapshotInstanceDetailRequest{}
		RegionNo:                 	regionNo, 			// Caution!! : RegionNo (Not RegionCode)
	}
	callLogStart := call.Start()
	result, err := myImageHandler.VMClient.V2Api.GetMemberServerImageList(&imageReq) // Caution : Not GetBlockStorageSnapshotInstanceDetail()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Snapshot Image Info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var imageInfoList []*irs.MyImageInfo
	if len(result.MemberServerImageList) < 1 {
		cblogger.Info("# Snapshot Image does Not Exist!!")
	} else {
		cblogger.Info("Succeeded in Getting the Snapshot Info List.")
		for _, snapshotImage := range result.MemberServerImageList {
			imageInfo, err := myImageHandler.MappingMyImageInfo(snapshotImage)
			if err != nil {
				newErr := fmt.Errorf("Failed to Map MyImage Info!!")
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
			}
			imageInfoList = append(imageInfoList, imageInfo)
		}
	}
	return imageInfoList, nil
}

func (myImageHandler *NcpMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called GetMyImage()")

	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "GetMyImage()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid myImage SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}	

	memberServerImageInfo, err := myImageHandler.GetNcpMemberServerImageInfo(myImageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Member Server Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	imageInfo, err := myImageHandler.MappingMyImageInfo(&memberServerImageInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Map MyImage Info!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
	}
	return *imageInfo, nil
}

func (myImageHandler *NcpMyImageHandler) CheckWindowsImage(myImageIID irs.IID) (bool, error) {
	cblogger.Info("NCP Classic Cloud Driver: called CheckWindowsImage()")

	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "CheckWindowsImage()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid myImage SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}	

	myImagePlatform, err := myImageHandler.GetOriginImageOSPlatform(myImageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get MyImage Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	
	isWindowsImage := false
	if strings.Contains(myImagePlatform, "WINDOWS") {
		isWindowsImage = true
	}
	return isWindowsImage, nil
}

func (myImageHandler *NcpMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	cblogger.Info("NCP Classic Cloud Driver: called DeleteMyImage()")

	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "DeleteMyImage()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid myImage SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}	

	snapshotImageNoList := []*string {&myImageIID.SystemId}
	delReq := server.DeleteMemberServerImagesRequest{ 	// Not DeleteBlockStorageSnapshotInstancesRequest{}
		MemberServerImageNoList: 	snapshotImageNoList,
	}
	callLogStart := call.Start()
	result, err := myImageHandler.VMClient.V2Api.DeleteMemberServerImages(&delReq) // Not DeleteBlockStorageSnapshotInstances()
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Snapshot Image. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if !strings.EqualFold(*result.ReturnMessage, "success") {
		newErr := fmt.Errorf("Failed to Delete the Snapshot Image.")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	} else {
		cblogger.Info("Succeeded in Deleting the Snapshot Image.")
	}

	return true, nil
}

// Waiting for up to 500 seconds during Taking a Snapshot from a VM
func (myImageHandler *NcpMyImageHandler) WaitForImageSnapshot(myImageIID irs.IID) (irs.MyImageStatus, error) {
	cblogger.Info("===> Since Snapshot info. cannot be retrieved immediately after taking a snapshot, waits ....")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid myImage SystemId!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	curRetryCnt := 0
	maxRetryCnt := 500
	for {
		curStatus, err := myImageHandler.GetMyImageStatus(myImageIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Image Status. : [%v] ", err)
			cblogger.Error(newErr.Error())
			return "Failed. ", newErr
		} else {
			cblogger.Infof("Succeeded in Getting the Image Status : [%s]", string(curStatus))
		}
		cblogger.Infof("\n ### Image Status : [%s]", string(curStatus))

		if strings.EqualFold(string(curStatus), "Unavailable") {
			curRetryCnt++
			cblogger.Infof("The Image is still 'Unavailable', so wait for a second more before inquiring the Image info.")
			time.Sleep(time.Second * 3)
			if curRetryCnt > maxRetryCnt {
				newErr := fmt.Errorf("Despite waiting for a long time(%d sec), the Image status is %s, so it is forcibly finished.", maxRetryCnt, string(curStatus))
				cblogger.Error(newErr.Error())
				return "Failed. ", newErr
			}
		} else {
			cblogger.Infof("===> ### The Image Snapshot is finished, stopping the waiting.")
			return curStatus, nil
			//break
		}
	}
}

func (myImageHandler *NcpMyImageHandler) GetMyImageStatus(myImageIID irs.IID) (irs.MyImageStatus, error) {
	cblogger.Info("NCP Classic Cloud Driver: called GetMyImageStatus()")

	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "GetMyImageStatus()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid myImage SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	memberServerImageInfo, err := myImageHandler.GetNcpMemberServerImageInfo(myImageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Member Server Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}
	cblogger.Infof("### NCP Member Server Image Status : [%s]" , *memberServerImageInfo.MemberServerImageStatus.Code)
	myImageStatus := ConvertImageStatus(*memberServerImageInfo.MemberServerImageStatus.Code)
	return myImageStatus, nil
}

func ConvertImageStatus(myImageStatus string) irs.MyImageStatus {
	cblogger.Info("NCP Classic Cloud Driver: called ConvertImageStatus()")	
	// Ref) https://api.ncloud-docs.com/docs/common-vapidatatype-blockstoragesnapshotinstance
	var resultStatus irs.MyImageStatus
	switch myImageStatus {
	case "INIT" :
		resultStatus = irs.MyImageUnavailable	// "Unavailable"
	case "CREAT" : // CREATED
		resultStatus = irs.MyImageAvailable		// "Available"
	default:
		resultStatus = "Unknown"
	}
	return resultStatus
}

func (myImageHandler *NcpMyImageHandler) MappingMyImageInfo(myImage *server.MemberServerImage) (*irs.MyImageInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called MappingMyImageInfo()!")

	// cblogger.Info("\n\n### myImage in MappingMyImageInfo() : ")
	// spew.Dump(myImage)
	// cblogger.Info("\n")

	convertedTime, err := convertTimeFormat(*myImage.CreateDate)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert the Time Format!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	myImageInfo := &irs.MyImageInfo {
		IId: irs.IID{
			NameId:   *myImage.MemberServerImageName,
			SystemId: *myImage.MemberServerImageNo,
		},
		SourceVM: 	  irs.IID{NameId: *myImage.OriginalServerName, SystemId: *myImage.OriginalServerInstanceNo},
		Status: 	  ConvertImageStatus(*myImage.MemberServerImageStatus.Code),
		CreatedTime:  convertedTime,
	}

	keyValueList := []irs.KeyValue{
		{Key: "Region", 						Value: myImageHandler.RegionInfo.Region},
		{Key: "OriginalImageProductCode",		Value: *myImage.OriginalServerImageProductCode},
		{Key: "MyImagePlatformType",			Value: *myImage.MemberServerImagePlatformType.CodeName},
		{Key: "CreateDate",						Value: *myImage.CreateDate},
	}
	myImageInfo.KeyValueList = keyValueList
	return myImageInfo, nil
}

func (myImageHandler *NcpMyImageHandler) GetNcpMemberServerImageInfo(myImageIID irs.IID) (server.MemberServerImage, error) {
	cblogger.Info("NCP Classic Cloud Driver: called GetMyImage()")

	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "GetNcpMemberServerImageInfo()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid myImage ystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return server.MemberServerImage{}, newErr
	}	

	vmHandler := NcpVMHandler{
		RegionInfo:     	myImageHandler.RegionInfo,
		VMClient:         	myImageHandler.VMClient,
	}
	regionNo, err := vmHandler.GetRegionNo(myImageHandler.RegionInfo.Region)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Region No of the Region Code: [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return server.MemberServerImage{}, newErr
	}
	imageReq := server.GetMemberServerImageListRequest{ // Not GetBlockStorageSnapshotInstanceDetailRequest{}
		RegionNo:                 	regionNo, 			// Caution!! : RegionNo (Not RegionCode)
		MemberServerImageNoList: 	[]*string{&myImageIID.SystemId},
	}
	callLogStart := call.Start()
	result, err := myImageHandler.VMClient.V2Api.GetMemberServerImageList(&imageReq) // Caution : Not GetBlockStorageSnapshotInstanceDetail()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Member Server Image List from NCP: [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return server.MemberServerImage{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.MemberServerImageList) < 1 {
		newErr := fmt.Errorf("Failed to Get the Member Server Image List from NCP. Member Server Image does Not Exist!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return server.MemberServerImage{}, newErr
	} else {
		cblogger.Info("Succeeded in Getting the Member Server Image Info.")
	}
	return *result.MemberServerImageList[0], nil
}

func (myImageHandler *NcpMyImageHandler) GetOriginImageOSPlatform(imageIID irs.IID) (string, error) {
	cblogger.Info("NCP Classic Cloud Driver: called GetOriginImageOSPlatform()")

	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, imageIID.SystemId, "GetOriginImageOSPlatform()")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}	

	isPublicImage, err := myImageHandler.isPublicImage(imageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Check Whether the Image is Public Image : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}	
	if isPublicImage {
		imageHandler := NcpImageHandler{
			RegionInfo:  myImageHandler.RegionInfo,
			VMClient:    myImageHandler.VMClient,
		}	
		ncpImageInfo, err := imageHandler.GetNcpImageInfo(imageIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Image Info from NCP : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return "", newErr
		} else {				
			imagePlatformType := strings.ToUpper(*ncpImageInfo.PlatformType.CodeName)
			var originImagePlatform string
			if strings.Contains(imagePlatformType, "UBUNTU") {
				originImagePlatform = "UBUNTU"
			} else if strings.Contains(imagePlatformType, "CENTOS") {
				originImagePlatform = "CENTOS"
			} else if strings.Contains(imagePlatformType, "WINDOWS") {
				originImagePlatform = "WINDOWS"
			} else {
				newErr := fmt.Errorf("Failed to Get OriginImageOSPlatform of the Image!!")
				cblogger.Error(newErr.Error())
				return "", newErr
			}
			cblogger.Infof("### OriginImagePlatform : [%s]", originImagePlatform)
			return originImagePlatform, nil
		}
	} else {
		memberServerImageInfo, err := myImageHandler.GetNcpMemberServerImageInfo(imageIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get NCP Member Server Image Info. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return "", newErr
		}
		cblogger.Infof("### MyImagePlatformType : [%s]", *memberServerImageInfo.MemberServerImagePlatformType.CodeName)
		
		imagePlatformType := strings.ToUpper(*memberServerImageInfo.MemberServerImagePlatformType.CodeName)
		var originImagePlatform string
		if strings.Contains(imagePlatformType, "UBUNTU") {
			originImagePlatform = "UBUNTU"
		} else if strings.Contains(imagePlatformType, "CENTOS") {
			originImagePlatform = "CENTOS"
		} else if strings.Contains(imagePlatformType, "WINDOWS") {
			originImagePlatform = "WINDOWS"
		} else {
			newErr := fmt.Errorf("Failed to Get OriginImageOSPlatform of the MyImage!!")
			cblogger.Error(newErr.Error())
			return "", newErr
		}
		cblogger.Infof("### OriginImagePlatform : [%s]", originImagePlatform)
		return originImagePlatform, nil
	}	
}

func (myImageHandler *NcpMyImageHandler) isPublicImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("NCP Classic Cloud Driver: called isPublicImage()")

	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, imageIID.SystemId, "isPublicImage()") // HisCall logging

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Image SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}	

	imageHandler := NcpImageHandler{
		RegionInfo:  myImageHandler.RegionInfo,
		VMClient:    myImageHandler.VMClient,
	}	
	ncpImageInfo, err := imageHandler.GetNcpImageInfo(imageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Image Info from NCP : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, nil // Caution!!
	} else {
		isPublicImage := false
		if strings.EqualFold(*ncpImageInfo.ProductCode, imageIID.SystemId) {
			isPublicImage = true
		}
		return isPublicImage, nil
	}
}
