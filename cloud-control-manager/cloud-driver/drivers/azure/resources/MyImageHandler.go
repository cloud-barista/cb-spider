package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strconv"
	"strings"
	"time"
)

type AzureMyImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Ctx            context.Context
	VMClient       *compute.VirtualMachinesClient
	ImageClient    *compute.ImagesClient
}

func (myImageHandler *AzureMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (myImageInfo irs.MyImageInfo, snapshotErr error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.Region, call.MYIMAGE, snapshotReqInfo.IId.NameId, "SnapshotVM()")
	start := call.Start()
	convertedMyImageIId, err := ConvertMyImageIID(snapshotReqInfo.IId, myImageHandler.CredentialInfo, myImageHandler.Region)
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	// image 이름 확인
	exist, err := CheckExistMyImage(convertedMyImageIId, myImageHandler.ImageClient, myImageHandler.Ctx)
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	if exist {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = already MyImage %s", convertedMyImageIId.NameId))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	// vm 존재 확인
	sourceVM := snapshotReqInfo.SourceVM
	convertedVMIId, err := ConvertVMIID(sourceVM, myImageHandler.CredentialInfo, myImageHandler.Region)
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	exist, err = CheckExistVM(convertedVMIId, myImageHandler.Region.ResourceGroup, myImageHandler.VMClient, myImageHandler.Ctx)
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	if !exist {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = not found vm %s", convertedVMIId.NameId))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	rawVm, err := GetRawVM(convertedVMIId, myImageHandler.Region.ResourceGroup, myImageHandler.VMClient, myImageHandler.Ctx)
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	vmStatus := getVmStatus(*rawVm.InstanceView)

	if vmStatus == irs.Running {
		offFuture, err := myImageHandler.VMClient.PowerOff(myImageHandler.Ctx, myImageHandler.Region.ResourceGroup, *rawVm.Name, to.BoolPtr(false))
		if err != nil {
			snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. Failed to PowerOff err = %s", err))
			cblogger.Error(snapshotErr.Error())
			LoggingError(hiscallInfo, snapshotErr)
			return irs.MyImageInfo{}, snapshotErr
		}
		err = offFuture.WaitForCompletionRef(myImageHandler.Ctx, myImageHandler.VMClient.Client)
		if err != nil {
			snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. Failed to PowerOff err = %s", err))
			cblogger.Error(snapshotErr.Error())
			LoggingError(hiscallInfo, snapshotErr)
			return irs.MyImageInfo{}, snapshotErr
		}
		curRetryCnt := 0
		maxRetryCnt := 60
		for {
			instanceView, instanceViewErr := myImageHandler.VMClient.InstanceView(myImageHandler.Ctx, myImageHandler.Region.ResourceGroup, *rawVm.Name)
			if instanceViewErr == nil && getVmStatus(instanceView) == irs.Suspended {
				break
			}
			curRetryCnt++
			time.Sleep(1 * time.Second)
			if curRetryCnt > maxRetryCnt {
				snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. Failed to PowerOff err = exceeded maximum retry count %d", maxRetryCnt))
				cblogger.Error(snapshotErr.Error())
				LoggingError(hiscallInfo, snapshotErr)
				return irs.MyImageInfo{}, snapshotErr
			}
		}
	} else if vmStatus != irs.Suspended {
		if err != nil {
			snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = Snapshots are only available in the 'Suspended', 'Running' state."))
			cblogger.Error(snapshotErr.Error())
			LoggingError(hiscallInfo, snapshotErr)
			return irs.MyImageInfo{}, snapshotErr
		}
	}
	// 이미지 생성
	imagecreatOpt := compute.Image{
		Location: to.StringPtr(myImageHandler.Region.Region),
		ImageProperties: &compute.ImageProperties{
			SourceVirtualMachine: &compute.SubResource{
				ID: to.StringPtr(convertedVMIId.SystemId),
			},
		},
		Tags: map[string]*string{
			"createdAt": to.StringPtr(strconv.FormatInt(time.Now().Unix(), 10)),
		},
	}
	_, err = myImageHandler.VMClient.Generalize(myImageHandler.Ctx, myImageHandler.Region.ResourceGroup, convertedVMIId.NameId)
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	result, err := myImageHandler.ImageClient.CreateOrUpdate(myImageHandler.Ctx, myImageHandler.Region.ResourceGroup, convertedMyImageIId.NameId, imagecreatOpt)
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	defer func() {
		if snapshotErr != nil {
			result, err := myImageHandler.ImageClient.Delete(myImageHandler.Ctx, myImageHandler.Region.ResourceGroup, convertedMyImageIId.NameId)
			if err == nil {
				result.WaitForCompletionRef(myImageHandler.Ctx, myImageHandler.ImageClient.Client)
			}
		}
	}()
	err = result.WaitForCompletionRef(myImageHandler.Ctx, myImageHandler.ImageClient.Client)
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	myImage, err := myImageHandler.ImageClient.Get(myImageHandler.Ctx, myImageHandler.Region.ResourceGroup, convertedMyImageIId.NameId, "")
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	info, err := setterMyImageInfo(myImage, myImageHandler.CredentialInfo, myImageHandler.Region)
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (myImageHandler *AzureMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.Region, call.MYIMAGE, "MyImage", "ListMyImage()")
	start := call.Start()
	myImageList, err := myImageHandler.ImageClient.List(myImageHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List MyImage. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return []*irs.MyImageInfo{}, getErr
	}
	myImageInfoList := make([]*irs.MyImageInfo, len(myImageList.Values()))
	for i, myImage := range myImageList.Values() {
		info, err := setterMyImageInfo(myImage, myImageHandler.CredentialInfo, myImageHandler.Region)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List MyImage. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return []*irs.MyImageInfo{}, getErr
		}
		myImageInfoList[i] = &info
	}
	LoggingInfo(hiscallInfo, start)
	return myImageInfoList, nil
}
func (myImageHandler *AzureMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.Region, call.MYIMAGE, myImageIID.NameId, "GetMyImage()")
	start := call.Start()
	convertedMyImageIID, err := ConvertMyImageIID(myImageIID, myImageHandler.CredentialInfo, myImageHandler.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get MyImage. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.MyImageInfo{}, getErr
	}
	myImage, err := myImageHandler.ImageClient.Get(myImageHandler.Ctx, myImageHandler.Region.ResourceGroup, convertedMyImageIID.NameId, "")
	info, err := setterMyImageInfo(myImage, myImageHandler.CredentialInfo, myImageHandler.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get MyImage. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.MyImageInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}
func (myImageHandler *AzureMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.Region, call.MYIMAGE, myImageIID.NameId, "GetMyImage()")
	start := call.Start()
	convertedMyImageIID, err := ConvertMyImageIID(myImageIID, myImageHandler.CredentialInfo, myImageHandler.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Delete MyImage. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	exist, err := CheckExistMyImage(convertedMyImageIID, myImageHandler.ImageClient, myImageHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Delete MyImage. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	if !exist {
		getErr := errors.New(fmt.Sprintf("Failed to Delete MyImage. err = not found MyImage : %s", convertedMyImageIID.NameId))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	result, err := myImageHandler.ImageClient.Delete(myImageHandler.Ctx, myImageHandler.Region.ResourceGroup, convertedMyImageIID.NameId)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Delete MyImage. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	err = result.WaitForCompletionRef(myImageHandler.Ctx, myImageHandler.ImageClient.Client)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Delete MyImage. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func setterMyImageInfo(myImage compute.Image, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) (irs.MyImageInfo, error) {
	vmIID := irs.IID{
		SystemId: *myImage.ImageProperties.SourceVirtualMachine.ID,
	}
	convertedVmIID, err := ConvertVMIID(vmIID, credentialInfo, regionInfo)
	if err != nil {
		return irs.MyImageInfo{}, err
	}
	status := irs.MyImageUnavailable
	if *myImage.ImageProperties.ProvisioningState == "Succeeded" {
		status = irs.MyImageAvailable
	}
	myImageInfo := irs.MyImageInfo{
		IId: irs.IID{
			NameId:   *myImage.Name,
			SystemId: *myImage.ID,
		},
		SourceVM: convertedVmIID,
		Status:   status,
	}
	if myImage.Tags["createdAt"] != nil {
		createAt := *myImage.Tags["createdAt"]
		timeInt64, err := strconv.ParseInt(createAt, 10, 64)
		if err == nil {
			myImageInfo.CreatedTime = time.Unix(timeInt64, 0)
		}
	}
	return myImageInfo, nil
}

func ConvertMyImageIID(myImageIID irs.IID, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) (irs.IID, error) {
	if myImageIID.NameId == "" && myImageIID.SystemId == "" {
		return myImageIID, errors.New(fmt.Sprintf("invalid IID"))
	}
	if myImageIID.SystemId == "" {
		sysID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/images/%s", credentialInfo.SubscriptionId, regionInfo.ResourceGroup, myImageIID.NameId)
		return irs.IID{NameId: myImageIID.NameId, SystemId: sysID}, nil
	} else {
		slist := strings.Split(myImageIID.SystemId, "/")
		if len(slist) == 0 {
			return myImageIID, errors.New(fmt.Sprintf("Invalid IID"))
		}
		s := slist[len(slist)-1]
		return irs.IID{NameId: s, SystemId: myImageIID.SystemId}, nil
	}
}

func CheckExistMyImage(myImageIID irs.IID, client *compute.ImagesClient, ctx context.Context) (bool, error) {
	myImageList, err := client.List(ctx)
	if err != nil {
		return false, err
	}
	for _, myImage := range myImageList.Values() {
		if myImageIID.SystemId != "" && myImageIID.SystemId == *myImage.ID {
			return true, nil
		}
		if myImageIID.NameId != "" && myImageIID.NameId == *myImage.Name {
			return true, nil
		}
	}
	return false, nil
}
