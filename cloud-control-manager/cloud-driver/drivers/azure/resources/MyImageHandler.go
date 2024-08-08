package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"reflect"
	"strconv"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzureMyImageHandler struct {
	CredentialInfo                  idrv.CredentialInfo
	Region                          idrv.RegionInfo
	Ctx                             context.Context
	VMClient                        *armcompute.VirtualMachinesClient
	ImageClient                     *armcompute.ImagesClient
	VirtualMachineRunCommandsClient *armcompute.VirtualMachineRunCommandsClient
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
	exist, err = CheckExistVM(convertedVMIId, myImageHandler.Region.Region, myImageHandler.VMClient, myImageHandler.Ctx)
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
	rawVm, err := GetRawVM(convertedVMIId, myImageHandler.Region.Region, myImageHandler.VMClient, myImageHandler.Ctx)
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	err = preparationOperationForGeneralize(rawVm, myImageHandler.VMClient, myImageHandler.VirtualMachineRunCommandsClient, myImageHandler.Ctx, myImageHandler.Region)
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}

	// 이미지 생성
	imagecreatOpt := armcompute.Image{
		Location: &myImageHandler.Region.Region,
		Properties: &armcompute.ImageProperties{
			SourceVirtualMachine: &armcompute.SubResource{
				ID: &convertedVMIId.SystemId,
			},
		},
		Tags: map[string]*string{
			"createdAt": toStrPtr(strconv.FormatInt(time.Now().Unix(), 10)),
		},
	}
	if snapshotReqInfo.TagList != nil {
		for _, tag := range snapshotReqInfo.TagList {
			imagecreatOpt.Tags[tag.Key] = &tag.Value
		}
	}

	_, err = myImageHandler.VMClient.Generalize(myImageHandler.Ctx, myImageHandler.Region.Region, convertedVMIId.NameId, nil)
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	poller, err := myImageHandler.ImageClient.BeginCreateOrUpdate(myImageHandler.Ctx, myImageHandler.Region.Region, convertedMyImageIId.NameId, imagecreatOpt, nil)
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	defer func() {
		if snapshotErr != nil {
			poller, err := myImageHandler.ImageClient.BeginDelete(myImageHandler.Ctx, myImageHandler.Region.Region, convertedMyImageIId.NameId, nil)
			if err == nil {
				_, _ = poller.PollUntilDone(myImageHandler.Ctx, nil)
			}
		}
	}()
	_, err = poller.PollUntilDone(myImageHandler.Ctx, nil)
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	resp, err := myImageHandler.ImageClient.Get(myImageHandler.Ctx, myImageHandler.Region.Region, convertedMyImageIId.NameId, nil)
	if err != nil {
		snapshotErr = errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(snapshotErr.Error())
		LoggingError(hiscallInfo, snapshotErr)
		return irs.MyImageInfo{}, snapshotErr
	}
	info, err := setterMyImageInfo(&resp.Image, myImageHandler.CredentialInfo, myImageHandler.Region)
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

	var myImageList []*armcompute.Image

	pager := myImageHandler.ImageClient.NewListByResourceGroupPager(myImageHandler.Region.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(myImageHandler.Ctx)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List MyImage. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}

		for _, myImage := range page.Value {
			myImageList = append(myImageList, myImage)
		}
	}

	var myImageInfoList []*irs.MyImageInfo

	for _, myImage := range myImageList {
		info, err := setterMyImageInfo(myImage, myImageHandler.CredentialInfo, myImageHandler.Region)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List MyImage. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return []*irs.MyImageInfo{}, getErr
		}
		myImageInfoList = append(myImageInfoList, &info)
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

	resp, err := myImageHandler.ImageClient.Get(myImageHandler.Ctx, myImageHandler.Region.Region, convertedMyImageIID.NameId, nil)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get MyImage. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.MyImageInfo{}, getErr
	}

	info, err := setterMyImageInfo(&resp.Image, myImageHandler.CredentialInfo, myImageHandler.Region)
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

	poller, err := myImageHandler.ImageClient.BeginDelete(myImageHandler.Ctx, myImageHandler.Region.Region, convertedMyImageIID.NameId, nil)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Delete MyImage. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	_, err = poller.PollUntilDone(myImageHandler.Ctx, nil)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Delete MyImage. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}

	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func setterMyImageInfo(myImage *armcompute.Image, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) (irs.MyImageInfo, error) {
	vmIID := irs.IID{
		SystemId: *myImage.Properties.SourceVirtualMachine.ID,
	}
	convertedVmIID, err := ConvertVMIID(vmIID, credentialInfo, regionInfo)
	if err != nil {
		return irs.MyImageInfo{}, err
	}
	status := irs.MyImageUnavailable
	if *myImage.Properties.ProvisioningState == "Succeeded" {
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
	if myImage.Tags != nil {
		myImageInfo.TagList = setTagList(myImage.Tags)
	}

	return myImageInfo, nil
}

func ConvertMyImageIID(myImageIID irs.IID, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) (irs.IID, error) {
	if myImageIID.NameId == "" && myImageIID.SystemId == "" {
		return myImageIID, errors.New(fmt.Sprintf("invalid IID"))
	}

	if myImageIID.SystemId == "" {
		sysID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/images/%s", credentialInfo.SubscriptionId, regionInfo.Region, myImageIID.NameId)
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

func CheckExistMyImage(myImageIID irs.IID, client *armcompute.ImagesClient, ctx context.Context) (bool, error) {
	var myImageList []*armcompute.Image

	pager := client.NewListPager(nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return false, err
		}

		for _, myImage := range page.Value {
			myImageList = append(myImageList, myImage)
		}
	}

	for _, myImage := range myImageList {
		if myImageIID.SystemId != "" && myImageIID.SystemId == *myImage.ID {
			return true, nil
		}

		if myImageIID.NameId != "" && myImageIID.NameId == *myImage.Name {
			return true, nil
		}
	}

	return false, nil
}

func preparationOperationForGeneralize(rawVm armcompute.VirtualMachine, vmClient *armcompute.VirtualMachinesClient, virtualMachineRunCommandsClient *armcompute.VirtualMachineRunCommandsClient, ctx context.Context, region idrv.RegionInfo) error {
	sourceVMOSType, err := getOSTypeByVM(rawVm)
	if err != nil {
		return err
	}
	vmStatus := getVmStatus(*rawVm.Properties.InstanceView)
	if sourceVMOSType == irs.WINDOWS {
		if vmStatus == irs.Running {
			err = windowShellPreparationOperationForGeneralize(*rawVm.Name, virtualMachineRunCommandsClient, ctx, region)
			if err != nil {
				return err
			}
			err = suspendCheck(*rawVm.Name, vmClient, ctx, region)
			if err != nil {
				return errors.New(fmt.Sprintf("failed to PowerOff err = %s", err))
			}
		} else if vmStatus == irs.Suspended {
			poller, err := vmClient.BeginStart(ctx, region.Region, *rawVm.Name, nil)
			if err != nil {
				return errors.New(fmt.Sprintf("The VM failed to runnig to prepare for virtualization inside the VM err = %s", err))
			}
			_, err = poller.PollUntilDone(ctx, nil)
			if err != nil {
				return errors.New(fmt.Sprintf("The VM failed to prepare for virtualization inside the VM err = %s", err))
			}
			curRetryCnt := 0
			maxRetryCnt := 60
			for {
				resp, instanceViewErr := vmClient.InstanceView(ctx, region.Region, *rawVm.Name, nil)
				if instanceViewErr == nil && getVmStatus(resp.VirtualMachineInstanceView) == irs.Running {
					break
				}
				curRetryCnt++
				time.Sleep(1 * time.Second)
				if curRetryCnt > maxRetryCnt {
					return errors.New(fmt.Sprintf("The VM failed to runnig to prepare for virtualization inside the VM err = exceeded maximum retry count %d", maxRetryCnt))
				}
			}
			err = windowShellPreparationOperationForGeneralize(*rawVm.Name, virtualMachineRunCommandsClient, ctx, region)
			if err != nil {
				return errors.New(fmt.Sprintf("virtualization preparation operation failed inside the VM. err = %s", err))
			}
			err = suspendCheck(*rawVm.Name, vmClient, ctx, region)
			if err != nil {
				return errors.New(fmt.Sprintf("failed to PowerOff err = %s", err))
			}
		} else {
			return errors.New(fmt.Sprintf("snapshots are only available in the 'Suspended', 'Running' state."))
		}
		return nil
	} else {
		// Linux
		if vmStatus == irs.Running {
			err = waitingVMSuspend(rawVm, vmClient, ctx, region)
			if err != nil {
				return err
			}
		} else if vmStatus != irs.Suspended {
			return errors.New(fmt.Sprintf("snapshots are only available in the 'Suspended', 'Running' state."))
		}
		return nil
	}
}

func suspendCheck(vmName string, vmClient *armcompute.VirtualMachinesClient, ctx context.Context, region idrv.RegionInfo) error {
	curRetryCnt := 0
	maxRetryCnt := 60
	for {
		resp, instanceViewErr := vmClient.InstanceView(ctx, region.Region, vmName, nil)
		if instanceViewErr == nil && getVmStatus(resp.VirtualMachineInstanceView) == irs.Suspended {
			break
		}
		curRetryCnt++
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			return errors.New(fmt.Sprintf("failed to PowerOff err = exceeded maximum retry count %d", maxRetryCnt))
		}
	}
	return nil
}

func waitingVMSuspend(rawVm armcompute.VirtualMachine, vmClient *armcompute.VirtualMachinesClient, ctx context.Context, region idrv.RegionInfo) error {
	poller, err := vmClient.BeginPowerOff(ctx, region.Region, *rawVm.Name, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to PowerOff err = %s", err))
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to PowerOff err = %s", err))
	}
	err = suspendCheck(*rawVm.Name, vmClient, ctx, region)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to PowerOff err = %s", err))
	}
	return nil
}

func windowShellPreparationOperationForGeneralize(vmName string, virtualMachineRunCommandsClient *armcompute.VirtualMachineRunCommandsClient, ctx context.Context, region idrv.RegionInfo) error {
	runOpt := armcompute.VirtualMachineRunCommand{
		Properties: &armcompute.VirtualMachineRunCommandProperties{
			Source: &armcompute.VirtualMachineRunCommandScriptSource{
				//Script: toStrPtr(fmt.Sprintf("net user /add administrator qwe1212!Q; net localgroup administrators cb-user /add; net user /delete administrator;")),
				Script: toStrPtr(`RD C:\Windows\Panther -Recurse; C:\Windows\system32\sysprep\sysprep.exe /oobe /generalize /mode:vm /shutdown;`),
			},
		},
		Location: toStrPtr(region.Region),
	}

	poller, err := virtualMachineRunCommandsClient.BeginCreateOrUpdate(ctx, region.Region, vmName, "RunPowerShellScript", runOpt, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("failed window PreworkForGeneralize %s", err.Error()))
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("failed window PreworkForGeneralize %s", err.Error()))
	}

	return nil
}

func (myImageHandler *AzureMyImageHandler) CheckWindowsImage(myImageIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.Region, call.MYIMAGE, myImageIID.NameId, "CheckWindowsImage()")
	start := call.Start()
	convertedMyImageIID, err := ConvertMyImageIID(myImageIID, myImageHandler.CredentialInfo, myImageHandler.Region)
	if err != nil {
		checkWindowsImageErr := errors.New(fmt.Sprintf("Failed to CheckWindowsImage By MyImage. err = %s", err))
		cblogger.Error(checkWindowsImageErr.Error())
		LoggingError(hiscallInfo, checkWindowsImageErr)
		return false, checkWindowsImageErr
	}
	resp, err := myImageHandler.ImageClient.Get(myImageHandler.Ctx, myImageHandler.Region.Region, convertedMyImageIID.NameId, nil)
	if err != nil {
		checkWindowsImageErr := errors.New(fmt.Sprintf("Failed to CheckWindowsImage By MyImage. err = failed get MyImage err %s", err.Error()))
		cblogger.Error(checkWindowsImageErr.Error())
		LoggingError(hiscallInfo, checkWindowsImageErr)
		return false, checkWindowsImageErr
	}
	if reflect.ValueOf(resp.Image.Properties.StorageProfile.OSDisk).IsNil() {
		checkWindowsImageErr := errors.New(fmt.Sprintf("Failed to CheckWindowsImage By MyImage. err = empty MyImage OSType"))
		cblogger.Error(checkWindowsImageErr.Error())
		LoggingError(hiscallInfo, checkWindowsImageErr)
		return false, checkWindowsImageErr
	}

	if *resp.Image.Properties.StorageProfile.OSDisk.OSType == armcompute.OperatingSystemTypesLinux {
		LoggingInfo(hiscallInfo, start)
		return false, nil
	} else if *resp.Image.Properties.StorageProfile.OSDisk.OSType == armcompute.OperatingSystemTypesWindows {
		LoggingInfo(hiscallInfo, start)
		return true, nil
	}
	checkWindowsImageErr := errors.New(fmt.Sprintf("Failed to CheckWindowsImage By MyImage. err = empty MyImage OSType"))
	cblogger.Error(checkWindowsImageErr.Error())
	LoggingError(hiscallInfo, checkWindowsImageErr)
	return false, checkWindowsImageErr
}
