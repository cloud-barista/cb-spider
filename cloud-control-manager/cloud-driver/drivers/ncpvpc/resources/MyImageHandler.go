// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2022.11.
// by ETRI, 2025.03. (Updated to support KVM Hypervisor)

package resources

import (
	"fmt"
	"strings"
	"time"
	// "github.com/davecgh/go-spew/spew"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	vserver "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVpcMyImageHandler struct {
	RegionInfo idrv.RegionInfo
	VMClient   *vserver.APIClient
}

const (
	// NCP VPC Snapshop Options : FULL | INCREMENTAL, Default : FULL
	NcpDefaultSnapshotTypeCode string = "FULL"
)

// To Take a Snapshot with VM ID (To Create My Image)
func (myImageHandler *NcpVpcMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called SnapshotVM()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, snapshotReqInfo.IId.SystemId, "SnapshotVM()")

	if strings.EqualFold(snapshotReqInfo.SourceVM.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}

	// Note) CreateMemberServerImageInstance() : For XEN/RHV
	// Note) CreateServerImage() : For XEN/RHV/KVM
	snapshotReq := vserver.CreateServerImageRequest{ // Not CreateBlockStorageSnapshotInstanceRequest{}
		RegionCode:         &myImageHandler.RegionInfo.Region,
		ServerInstanceNo:   &snapshotReqInfo.SourceVM.SystemId,
		ServerImageName: 	&snapshotReqInfo.IId.NameId,
	}
	callLogStart := call.Start()
	result, err := myImageHandler.VMClient.V2Api.CreateServerImage(&snapshotReq) // Not CreateBlockStorageSnapshotInstance
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New VM Snapshot : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)


	// cblogger.Info("\n\n### result.ServerImageList : ")
	// spew.Dump(result.ServerImageList)
	// cblogger.Info("\n")

	if len(result.ServerImageList) < 1 {
		newErr := fmt.Errorf("Failed to Create New VM Snapshot. Snapshot does Not Exist!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	} else {
		cblogger.Info("Succeeded in Creating New Snapshot.")
	}

	newImageIID := irs.IID{SystemId: *result.ServerImageList[0].ServerImageNo}
	// To Wait for Creating a Snapshot Image
	curStatus, err := myImageHandler.waitForImageSnapshot(newImageIID)
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

func (myImageHandler *NcpVpcMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called ListMyImage()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, "ListMyImage()", "ListMyImage()")

	imageListReq := vserver.GetServerImageListRequest{ // Not GetBlockStorageSnapshotInstanceListRequest
		RegionCode: 				&myImageHandler.RegionInfo.Region,
		ServerImageTypeCodeList: 	[]*string{ncloud.String("SELF")}, // Caution) Options: SELF | NCP
	}
	callLogStart := call.Start()
	result, err := myImageHandler.VMClient.V2Api.GetServerImageList(&imageListReq) // Caution : Not GetBlockStorageSnapshotInstanceList()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Snapshot Image List from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	// cblogger.Info("\n\n### MyImageList : ")
	// spew.Dump(result)
	// cblogger.Info("\n")

	var imageInfoList []*irs.MyImageInfo
	if len(result.ServerImageList) < 1 {
		cblogger.Info("# Snapshot Image does Not Exist!!")
	} else {
		cblogger.Info("Succeeded in Getting the Snapshot Info List.")
		for _, snapshotImage := range result.ServerImageList {
			imageInfo, err := myImageHandler.mappingMyImageInfo(snapshotImage)
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

func (myImageHandler *NcpVpcMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetMyImage()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "GetMyImage()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}

	imageReq := vserver.GetServerImageDetailRequest{ // Not GetBlockStorageSnapshotInstanceDetailRequest{}
		RegionCode:    &myImageHandler.RegionInfo.Region,
		ServerImageNo: &myImageIID.SystemId,
	}
	callLogStart := call.Start()
	result, err := myImageHandler.VMClient.V2Api.GetServerImageDetail(&imageReq) // Caution : Not GetBlockStorageSnapshotInstanceDetail()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Snapshot Image Info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.ServerImageList) < 1 {
		newErr := fmt.Errorf("Failed to Get the Snapshot Image Info.Snapshot Image does Not Exist!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	} else {
		cblogger.Info("Succeeded in Getting the Snapshot Image Info.")
	}

	imageInfo, err := myImageHandler.mappingMyImageInfo(result.ServerImageList[0])
	if err != nil {
		newErr := fmt.Errorf("Failed to Map MyImage Info!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
	}
	return *imageInfo, nil
}

func (myImageHandler *NcpVpcMyImageHandler) CheckWindowsImage(myImageIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CheckWindowsImage()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "CheckWindowsImage()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	myImageInfo, err := myImageHandler.GetMyImage(myImageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get MyImage Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	// Ref) https://api.ncloud-docs.com/docs/common-vapidatatype-serverimage
	var OsTypeCode string
	// Use Key/Value info of the myImageInfo.
	for _, keyInfo := range myImageInfo.KeyValueList {
		if keyInfo.Key == "OsType" {
			OsTypeCode = keyInfo.Value
			break
		}
	}
	cblogger.Infof("\n### OsTypeCode : [%s]", OsTypeCode)

	isWindowsImage := false
	if strings.Contains(strings.ToUpper(OsTypeCode), "WINDOWS") {
		isWindowsImage = true
	}

	return isWindowsImage, nil
}

func (myImageHandler *NcpVpcMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called DeleteMyImage()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "DeleteMyImage()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	snapshotImageNoList := []*string{&myImageIID.SystemId}
	delReq := vserver.DeleteServerImageRequest{ // Not DeleteBlockStorageSnapshotInstancesRequest{}
		RegionCode:        &myImageHandler.RegionInfo.Region,
		ServerImageNoList: snapshotImageNoList,
	}

	callLogStart := call.Start()
	result, err := myImageHandler.VMClient.V2Api.DeleteServerImage(&delReq) // Not DeleteBlockStorageSnapshotInstances()
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
func (myImageHandler *NcpVpcMyImageHandler) waitForImageSnapshot(myImageIID irs.IID) (irs.MyImageStatus, error) {
	cblogger.Info("NCP VPC Cloud Driver: called waitForImageSnapshot()")
	cblogger.Info("===> Since Snapshot info. cannot be retrieved immediately after taking a snapshot, waits ....")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	curRetryCnt := 0
	maxRetryCnt := 500
	for {
		curStatus, err := myImageHandler.GetImageStatus(myImageIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Image Status. : [%v] ", err)
			cblogger.Error(newErr.Error())
			return "Failed. ", newErr
		} else {
			cblogger.Infof("Succeeded in Getting the Image Status : [%s]", string(curStatus))
		}
		// cblogger.Infof("\n ### Image Status : [%s]", string(curStatus))

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

func (myImageHandler *NcpVpcMyImageHandler) GetImageStatus(myImageIID irs.IID) (irs.MyImageStatus, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetImageStatus()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "GetImageStatus()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	imageReq := vserver.GetServerImageDetailRequest{ // Not GetBlockStorageSnapshotInstanceDetailRequest{}
		RegionCode:     &myImageHandler.RegionInfo.Region,
		ServerImageNo: 	&myImageIID.SystemId,
	}
	callLogStart := call.Start()
	result, err := myImageHandler.VMClient.V2Api.GetServerImageDetail(&imageReq) // Caution : Not GetBlockStorageSnapshotInstanceDetail()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Snapshot Image Info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.ServerImageList) < 1 {
		newErr := fmt.Errorf("Failed to Get the Snapshot Image Info.Snapshot Image does Not Exist!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	} else {
		cblogger.Info("Succeeded in Getting the Snapshot Image Info.")
	}

	myImageStatus := convertImageStatus(*result.ServerImageList[0].ServerImageStatus.Code)
	return myImageStatus, nil
}

func convertImageStatus(myImageStatus string) irs.MyImageStatus {
	cblogger.Info("NCP VPC Cloud Driver: called convertImageStatus()")
	// Ref) https://api.ncloud-docs.com/docs/common-vapidatatype-serverimage
	var resultStatus irs.MyImageStatus
	switch myImageStatus {
	case "INIT":
		resultStatus = irs.MyImageUnavailable
	case "CREAT": // CREATED
		resultStatus = irs.MyImageAvailable
	default:
		resultStatus = "Unknown"
	}

	return resultStatus
}

func (myImageHandler *NcpVpcMyImageHandler) mappingMyImageInfo(myImage *vserver.ServerImage) (*irs.MyImageInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called mappingMyImageInfo()!")

	// cblogger.Info("\n\n### myImage in mappingMyImageInfo() : ")
	// spew.Dump(myImage)
	// cblogger.Info("\n")

	// convertedTime, err := convertTimeFormat(*myImage.CreateDate)
	// if err != nil {
	// 	newErr := fmt.Errorf("Failed to Convert the Time Format!!")
	// 	cblogger.Error(newErr.Error())
	// 	return nil, newErr
	// }

	myImageInfo := &irs.MyImageInfo{
		IId: irs.IID{
			NameId:   	*myImage.ServerImageName,
			SystemId: 	*myImage.ServerImageNo,
		},
		// SourceVM:    	irs.IID{SystemId: *myImage.OriginalServerInstanceNo},
		Status:      	convertImageStatus(*myImage.ServerImageStatus.Code),
		// CreatedTime: 	convertedTime,
		KeyValueList:   irs.StructToKeyValueList(myImage),
	}
	return myImageInfo, nil
}

func (myImageHandler *NcpVpcMyImageHandler) getOriginImageOSPlatform(myImageIID irs.IID) (string, error) {
	cblogger.Info("NCP VPC Cloud Driver: called getOriginImageOSPlatform()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "getOriginImageOSPlatform()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	myImageInfo, err := myImageHandler.GetMyImage(myImageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get MyImage Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	var originalImageProductCode string
	// Use Key/Value info of the myImageInfo.
	for _, keyInfo := range myImageInfo.KeyValueList {
		if keyInfo.Key == "osType" {
			originalImageProductCode = keyInfo.Value
			break
		}
	}
	// cblogger.Infof("\n### OriginalServerImageProductCode : [%s]", originalImageProductCode)

	var originImagePlatform string
	if strings.Contains(strings.ToUpper(originalImageProductCode), "UBNTU") {
		originImagePlatform = "UBUNTU"
	} else if strings.Contains(strings.ToUpper(originalImageProductCode), "CNTOS") {
		originImagePlatform = "CENTOS"
	} else if strings.Contains(strings.ToUpper(originalImageProductCode), "ROCKY") {
		originImagePlatform = "ROCKY"
	} else if strings.Contains(strings.ToUpper(originalImageProductCode), "WND") {
		originImagePlatform = "WINDOWS"
	} else {
		newErr := fmt.Errorf("Failed to Get OriginImageOSPlatform of the MyImage!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	return originImagePlatform, nil
}

func (myImageHandler *NcpVpcMyImageHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("NCP VPC Cloud Driver: called myImageHandler ListIID()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, "ListIID()", "ListIID()")

	imageListReq := vserver.GetServerImageListRequest{ // Not GetBlockStorageSnapshotInstanceListRequest
		RegionCode: &myImageHandler.RegionInfo.Region,
	}

	callLogStart := call.Start()
	result, err := myImageHandler.VMClient.V2Api.GetServerImageList(&imageListReq) // Caution : Not GetBlockStorageSnapshotInstanceList()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Snapshot Image List from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var iidList []*irs.IID
	if len(result.ServerImageList) < 1 {
		cblogger.Debug("### MyImage does Not Exist!!")
		return nil, nil
	} else {
		for _, myImage := range result.ServerImageList {
			iid := irs.IID{NameId: *myImage.ServerImageName, SystemId: *myImage.ServerImageNo}
			iidList = append(iidList, &iid)
		}
	}
	return iidList, nil
}
