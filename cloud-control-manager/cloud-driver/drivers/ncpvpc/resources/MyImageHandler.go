// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2022.11.

package resources

import (
	"errors"
	"fmt"
	"strings"
	"time"

	// "github.com/davecgh/go-spew/spew"

	// "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
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
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, snapshotReqInfo.IId.SystemId, "SnapshotVM()") // HisCall logging

	if strings.EqualFold(snapshotReqInfo.SourceVM.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}

	snapshotReq := vserver.CreateMemberServerImageInstanceRequest{ // Not CreateBlockStorageSnapshotInstanceRequest{}
		RegionCode:            &myImageHandler.RegionInfo.Region,
		MemberServerImageName: &snapshotReqInfo.IId.NameId,
		ServerInstanceNo:      &snapshotReqInfo.SourceVM.SystemId,
	}

	callLogStart := call.Start()
	result, err := myImageHandler.VMClient.V2Api.CreateMemberServerImageInstance(&snapshotReq) // Not CreateBlockStorageSnapshotInstance
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New VM Snapshot : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if *result.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Create New VM Snapshot. Snapshot does Not Exist!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	} else {
		cblogger.Info("Succeeded in Creating New Snapshot.")
	}

	newImageIID := irs.IID{SystemId: *result.MemberServerImageInstanceList[0].MemberServerImageInstanceNo}
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
func (myImageHandler *NcpVpcMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called ListMyImage()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, "ListMyImage()", "ListMyImage()") // HisCall logging

	imageListReq := vserver.GetMemberServerImageInstanceListRequest{ // Not GetBlockStorageSnapshotInstanceListRequest
		RegionCode: &myImageHandler.RegionInfo.Region,
	}

	callLogStart := call.Start()
	result, err := myImageHandler.VMClient.V2Api.GetMemberServerImageInstanceList(&imageListReq) // Caution : Not GetBlockStorageSnapshotInstanceList()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Snapshot Image List from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var imageInfoList []*irs.MyImageInfo
	if *result.TotalRows < 1 {
		cblogger.Info("# Snapshot Image does Not Exist!!")
	} else {
		cblogger.Info("Succeeded in Getting the Snapshot Info List.")
		for _, snapshotImage := range result.MemberServerImageInstanceList {
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

func (myImageHandler *NcpVpcMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetMyImage()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "GetMyImage()") // HisCall logging

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}

	imageReq := vserver.GetMemberServerImageInstanceDetailRequest{ // Not GetBlockStorageSnapshotInstanceDetailRequest{}
		RegionCode:                  &myImageHandler.RegionInfo.Region,
		MemberServerImageInstanceNo: &myImageIID.SystemId,
	}

	callLogStart := call.Start()
	result, err := myImageHandler.VMClient.V2Api.GetMemberServerImageInstanceDetail(&imageReq) // Caution : Not GetBlockStorageSnapshotInstanceDetail()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Snapshot Image Info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if *result.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Get the Snapshot Image Info.Snapshot Image does Not Exist!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	} else {
		cblogger.Info("Succeeded in Getting the Snapshot Image Info.")
	}

	imageInfo, err := myImageHandler.MappingMyImageInfo(result.MemberServerImageInstanceList[0])
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
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "CheckWindowsImage()") // HisCall logging

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

	var originalImageProductCode string
	// Use Key/Value info of the myImageInfo.
	for _, keyInfo := range myImageInfo.KeyValueList {
		if keyInfo.Key == "OriginalServerImageProductCode" {
			originalImageProductCode = keyInfo.Value
			break
		}
	}
	cblogger.Infof("\n### OriginalServerImageProductCode : [%s]", originalImageProductCode)

	isWindowsImage := false
	if strings.Contains(strings.ToUpper(originalImageProductCode), "WND") {
		isWindowsImage = true
	}

	return isWindowsImage, nil
}

func (myImageHandler *NcpVpcMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called DeleteMyImage()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "DeleteMyImage()") // HisCall logging

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	snapshotImageNoList := []*string{&myImageIID.SystemId}
	delReq := vserver.DeleteMemberServerImageInstancesRequest{ // Not DeleteBlockStorageSnapshotInstancesRequest{}
		RegionCode:                      &myImageHandler.RegionInfo.Region,
		MemberServerImageInstanceNoList: snapshotImageNoList,
	}

	callLogStart := call.Start()
	result, err := myImageHandler.VMClient.V2Api.DeleteMemberServerImageInstances(&delReq) // Not DeleteBlockStorageSnapshotInstances()
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
func (myImageHandler *NcpVpcMyImageHandler) WaitForImageSnapshot(myImageIID irs.IID) (irs.MyImageStatus, error) {
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

func (myImageHandler *NcpVpcMyImageHandler) GetImageStatus(myImageIID irs.IID) (irs.MyImageStatus, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetImageStatus()")
	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "GetImageStatus()") // HisCall logging

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	imageReq := vserver.GetMemberServerImageInstanceDetailRequest{ // Not GetBlockStorageSnapshotInstanceDetailRequest{}
		RegionCode:                  &myImageHandler.RegionInfo.Region,
		MemberServerImageInstanceNo: &myImageIID.SystemId,
	}

	callLogStart := call.Start()
	result, err := myImageHandler.VMClient.V2Api.GetMemberServerImageInstanceDetail(&imageReq) // Caution : Not GetBlockStorageSnapshotInstanceDetail()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Snapshot Image Info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if *result.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Get the Snapshot Image Info.Snapshot Image does Not Exist!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	} else {
		cblogger.Info("Succeeded in Getting the Snapshot Image Info.")
	}

	myImageStatus := ConvertImageStatus(*result.MemberServerImageInstanceList[0].MemberServerImageInstanceStatus.Code)
	return myImageStatus, nil
}

func ConvertImageStatus(myImageStatus string) irs.MyImageStatus {
	cblogger.Info("NCP VPC Cloud Driver: called ConvertImageStatus()")
	// Ref) https://api.ncloud-docs.com/docs/common-vapidatatype-blockstoragesnapshotinstance
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

func (myImageHandler *NcpVpcMyImageHandler) MappingMyImageInfo(myImage *vserver.MemberServerImageInstance) (*irs.MyImageInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called MappingMyImageInfo()!")

	// cblogger.Info("\n\n### myImage in MappingMyImageInfo() : ")
	// spew.Dump(myImage)
	// cblogger.Info("\n")

	convertedTime, err := convertTimeFormat(*myImage.CreateDate)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert the Time Format!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	myImageInfo := &irs.MyImageInfo{
		IId: irs.IID{
			NameId:   *myImage.MemberServerImageName,
			SystemId: *myImage.MemberServerImageInstanceNo,
		},
		SourceVM:    irs.IID{SystemId: *myImage.OriginalServerInstanceNo},
		Status:      ConvertImageStatus(*myImage.MemberServerImageInstanceStatus.Code),
		CreatedTime: convertedTime,
	}

	keyValueList := []irs.KeyValue{
		{Key: "Region", Value: myImageHandler.RegionInfo.Region},
		{Key: "OriginalServerImageProductCode", Value: *myImage.OriginalServerImageProductCode},
		{Key: "CreateDate", Value: *myImage.CreateDate},
	}

	myImageInfo.KeyValueList = keyValueList
	return myImageInfo, nil
}

func (myImageHandler *NcpVpcMyImageHandler) GetOriginImageOSPlatform(myImageIID irs.IID) (string, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetOriginImageOSPlatform()")

	InitLog()
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "GetOriginImageOSPlatform()") // HisCall logging

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
		if keyInfo.Key == "OriginalServerImageProductCode" {
			originalImageProductCode = keyInfo.Value
			break
		}
	}
	cblogger.Infof("\n### OriginalServerImageProductCode : [%s]", originalImageProductCode)

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

func (ImageHandler *NcpVpcMyImageHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}
