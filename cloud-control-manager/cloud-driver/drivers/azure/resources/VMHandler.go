// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by hyokyung.kim@innogrid.co.kr, 2019.07.

package resources

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"

	cdcom "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	ProvisioningStateCode string = "ProvisioningState/succeeded"
	VM                           = "VM"
	PremiumSSD                   = "PremiumSSD"
	StandardSSD                  = "StandardSSD"
	StandardHDD                  = "StandardHDD"
	WindowBaseUser               = "Administrator"
	WindowBaseGroup              = "Administrators"
	WindowBuitinUser             = CBVMUser
)

type AzureVMHandler struct {
	CredentialInfo                  idrv.CredentialInfo
	Region                          idrv.RegionInfo
	Ctx                             context.Context
	Client                          *armcompute.VirtualMachinesClient
	SubnetClient                    *armnetwork.SubnetsClient
	NicClient                       *armnetwork.InterfacesClient
	PublicIPClient                  *armnetwork.PublicIPAddressesClient
	DiskClient                      *armcompute.DisksClient
	SshKeyClient                    *armcompute.SSHPublicKeysClient
	ImageClient                     *armcompute.ImagesClient
	VirtualMachineRunCommandsClient *armcompute.VirtualMachineRunCommandsClient
}

func (vmHandler *AzureVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmReqInfo.IId.NameId, "StartVM()")
	// 0. Check vmReqInfo
	err := checkVMReqInfo(vmReqInfo)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = Invalid VM Crate Require Infomation"))
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	// 1. pre Check
	// 1-1. Exist VM
	vmExist, err := CheckExistVM(vmReqInfo.IId, vmHandler.Region.Region, vmHandler.Client, vmHandler.Ctx)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	if vmExist {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = The VM name %s already exists", vmReqInfo.IId.NameId))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	// 1-2. Check VMImageIID Image format, Exist Image, AuthInfo (Linux : SSHKey, Window: Password)
	imageOsType, err := CheckVMReqInfoOSType(vmReqInfo, vmHandler.ImageClient, vmHandler.CredentialInfo, vmHandler.Region, vmHandler.Ctx)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	err = checkAuthInfoOSType(vmReqInfo, imageOsType)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	vmImage := vmReqInfo.ImageIID.SystemId
	if vmImage == "" {
		vmImage = vmReqInfo.ImageIID.NameId
	}
	if vmReqInfo.ImageType == "" || vmReqInfo.ImageType == irs.PublicImage {
		//PublicImage
		if strings.Contains(vmImage, ":") {
			imageArr := strings.Split(vmImage, ":")
			if len(imageArr) < 4 {
				createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = Invalid Public Image IID"))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
		} else {
			createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = Invalid Public Image IID"))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
	} else {
		convertMyImageIId, err := ConvertMyImageIID(vmReqInfo.ImageIID, vmHandler.CredentialInfo, vmHandler.Region)
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		_, err = vmHandler.ImageClient.Get(vmHandler.Ctx, vmHandler.Region.Region, convertMyImageIId.NameId, nil)
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
	}
	rawDataDiskList := make([]armcompute.Disk, len(vmReqInfo.DataDiskIIDs))
	// 1-3. Check DataDisk, Check DataDisk Status
	if len(vmReqInfo.DataDiskIIDs) > 0 {
		for i, dataDiskIID := range vmReqInfo.DataDiskIIDs {
			convertedDiskIId, err := ConvertDiskIID(dataDiskIID, vmHandler.CredentialInfo, vmHandler.Region)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to Start VM. Failed to get DataDisk err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			disk, err := GetRawDisk(convertedDiskIId, vmHandler.Region.Region, vmHandler.DiskClient, vmHandler.Ctx)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to Start VM. Failed to get DataDisk err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			err = CheckAttachStatus(&disk)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to Start VM. Failed to check DataDisk Status err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			rawDataDiskList[i] = disk
		}
	}

	cleanVMClientSet := CleanVMClientSet{
		VPCName:    vmReqInfo.VpcIID.NameId,
		SubnetName: vmReqInfo.SubnetIID.NameId,
	}
	cleanResources := CleanVMClientRequestResource{}

	// 2. related Resource Create // publicip, vnic
	// 2-1. related Resource Create - PublicIP
	publicIPIId, err := CreatePublicIP(vmHandler, vmReqInfo)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	cleanResources.PublicIPName = publicIPIId.NameId

	// 2-1. related Resource Create - VNIC
	vNicIId, err := CreateVNic(vmHandler, vmReqInfo, publicIPIId)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Finished to rollback deleting", err.Error()))
		clean, deperr := vmHandler.cleanVMRelatedResource(VMCleanRelatedResource{
			RequiredSet:         cleanVMClientSet,
			CleanTargetResource: cleanResources,
		})
		if deperr != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback err = %s", err.Error(), deperr.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		if !clean {
			createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback deleting", err.Error()))
		}
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	vmSize := armcompute.VirtualMachineSizeTypes(vmReqInfo.VMSpecName)
	cleanResources.NetworkInterfaceName = vNicIId.NameId
	// 3. Set VmReqInfo
	// 3-1. Set VmReqInfo & Vnic
	vmOpts := armcompute.VirtualMachine{
		Location: toStrPtr(vmHandler.Region.Region),
		Properties: &armcompute.VirtualMachineProperties{
			HardwareProfile: &armcompute.HardwareProfile{
				VMSize: &vmSize,
			},
			OSProfile: &armcompute.OSProfile{
				ComputerName:  &vmReqInfo.IId.NameId,
				AdminUsername: toStrPtr(CBVMUser),
			},
			NetworkProfile: &armcompute.NetworkProfile{
				NetworkInterfaces: []*armcompute.NetworkInterfaceReference{
					{
						//ID: &vmReqInfo.NetworkInterfaceId,
						ID: &vNicIId.SystemId,
						Properties: &armcompute.NetworkInterfaceReferenceProperties{
							Primary: toBoolPtr(true),
						},
					},
				},
			},
		},
	}

	// Setting zone if available
	if vmHandler.Region.TargetZone != "" {
		vmOpts.Zones = []*string{
			&vmHandler.Region.TargetZone,
		}
	} else if vmHandler.Region.Zone != "" {
		vmOpts.Zones = []*string{
			&vmHandler.Region.Zone,
		}
	}

	// 3-2. Set VmReqInfo - vmImage & storageType
	var managedDisk = new(armcompute.ManagedDiskParameters)
	if vmReqInfo.RootDiskType != "" && strings.ToLower(vmReqInfo.RootDiskType) != "default" {
		storageType := GetVMDiskTypeInitType(vmReqInfo.RootDiskType)
		managedDisk.StorageAccountType = &storageType
	}
	// snapshotPoint Start

	createOption := armcompute.DiskCreateOptionTypesFromImage
	deleteOption := armcompute.DiskDeleteOptionTypesDelete

	//storageType := getVMDiskTypeInitType(vmReqInfo.RootDiskType)
	vmOpts.Properties.StorageProfile = &armcompute.StorageProfile{
		OSDisk: &armcompute.OSDisk{
			CreateOption: &createOption,
			//ManagedDisk: &compute.ManagedDiskParameters{
			//	StorageAccountType: storageType,
			//},
			ManagedDisk:  managedDisk,
			DeleteOption: &deleteOption,
		},
	}

	if vmReqInfo.ImageType == "" || vmReqInfo.ImageType == irs.PublicImage {
		//PublicImage
		imageArr := strings.Split(vmImage, ":")

		if len(imageArr) != 4 {
			return irs.VMInfo{}, errors.New("Failed to Start VM. err = Invalid image")
		}

		// URN 기반 퍼블릭 이미지 설정
		vmOpts.Properties.StorageProfile.ImageReference = &armcompute.ImageReference{
			Publisher: toStrPtr(imageArr[0]),
			Offer:     toStrPtr(imageArr[1]),
			SKU:       toStrPtr(imageArr[2]),
			Version:   toStrPtr(imageArr[3]),
		}
	} else {
		//MyImage
		convertMyImageIId, convertedErr := ConvertMyImageIID(vmReqInfo.ImageIID, vmHandler.CredentialInfo, vmHandler.Region)
		if convertedErr != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Finished to rollback deleting", convertedErr.Error()))
			cleanResource := CleanVMClientRequestResource{
				publicIPIId.NameId, vNicIId.NameId, "",
			}
			clean, deperr := vmHandler.cleanVMRelatedResource(VMCleanRelatedResource{
				RequiredSet:         cleanVMClientSet,
				CleanTargetResource: cleanResource,
			})
			if deperr != nil {
				createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback err = %s", convertedErr.Error(), deperr.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			if !clean {
				createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback deleting", convertedErr.Error()))
			}
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		vmOpts.Properties.StorageProfile.ImageReference = &armcompute.ImageReference{
			ID: &convertMyImageIId.SystemId,
		}
	}

	if imageOsType == irs.LINUX_UNIX {
		// 3-2. Set VmReqInfo - KeyPair & tagging
		if vmReqInfo.KeyPairIID.NameId != "" {
			key, keyErr := GetRawKey(vmReqInfo.KeyPairIID, vmHandler.Region.Region, vmHandler.SshKeyClient, vmHandler.Ctx)
			if keyErr != nil {
				createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Finished to rollback deleting", keyErr.Error()))
				cleanResource := CleanVMClientRequestResource{
					publicIPIId.NameId, vNicIId.NameId, "",
				}
				clean, deperr := vmHandler.cleanVMRelatedResource(VMCleanRelatedResource{
					RequiredSet:         cleanVMClientSet,
					CleanTargetResource: cleanResource,
				})
				if deperr != nil {
					createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback err = %s", keyErr.Error(), deperr.Error()))
					cblogger.Error(createErr.Error())
					LoggingError(hiscallInfo, createErr)
					return irs.VMInfo{}, createErr
				}
				if !clean {
					createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback deleting", keyErr.Error()))
				}
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			publicKey := *key.Properties.PublicKey
			keyData := fmt.Sprintf("/home/%s/.ssh/authorized_keys", CBVMUser)
			vmOpts.Properties.OSProfile.LinuxConfiguration = &armcompute.LinuxConfiguration{
				SSH: &armcompute.SSHConfiguration{
					PublicKeys: []*armcompute.SSHPublicKey{
						{
							Path:    &keyData,
							KeyData: &publicKey,
						},
					},
				},
			}
			vmOpts.Tags = map[string]*string{
				"keypair":   &vmReqInfo.KeyPairIID.NameId,
				"publicip":  &publicIPIId.NameId,
				"createdBy": &vmReqInfo.IId.NameId,
			}
		} else {
			vmOpts.Properties.OSProfile.AdminPassword = &vmReqInfo.VMUserPasswd
			vmOpts.Tags = map[string]*string{
				"publicip":  &publicIPIId.NameId,
				"createdBy": &vmReqInfo.IId.NameId,
			}
		}
	} else {
		if len(vmReqInfo.IId.NameId) > 15 {
			computerName := vmReqInfo.IId.NameId[:15]
			vmOpts.Properties.OSProfile.ComputerName = &computerName
		}
		vmOpts.Properties.OSProfile.AdminPassword = &vmReqInfo.VMUserPasswd
		adminUserName := WindowBuitinUser
		vmOpts.Properties.OSProfile.AdminUsername = &adminUserName
		vmOpts.Tags = map[string]*string{
			"publicip":  &publicIPIId.NameId,
			"createdBy": &vmReqInfo.IId.NameId,
		}
	}
	// tags := setTags(vmReqInfo.TagList)
	if vmReqInfo.TagList != nil {
		for _, tag := range vmReqInfo.TagList {
			vmOpts.Tags[tag.Key] = &tag.Value
		}
	}

	// 4. CreateVM
	start := call.Start()
	poller, err := vmHandler.Client.BeginCreateOrUpdate(vmHandler.Ctx, vmHandler.Region.Region, vmReqInfo.IId.NameId, vmOpts, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Finished to rollback deleting", err.Error()))
		clean, deperr := vmHandler.cleanVMRelatedResource(VMCleanRelatedResource{
			RequiredSet:         cleanVMClientSet,
			CleanTargetResource: cleanResources,
		})
		if deperr != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback err = %s", err.Error(), deperr.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		if !clean {
			createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback deleting", err.Error()))
		}
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	_, err = poller.PollUntilDone(vmHandler.Ctx, nil)
	if err != nil {
		// Exist VM? exist => vm delete, ResourceClean, not exist => ResourceClean
		createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Finished to rollback deleting", err.Error()))
		exist, err := CheckExistVM(vmReqInfo.IId, vmHandler.Region.Region, vmHandler.Client, vmHandler.Ctx)
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		if exist {
			cleanErr := vmHandler.cleanDeleteVm(vmReqInfo.IId)
			if cleanErr != nil {
				createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback err = %s", errMsg, cleanErr.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
		}
		clean, deperr := vmHandler.cleanVMRelatedResource(VMCleanRelatedResource{
			RequiredSet:         cleanVMClientSet,
			CleanTargetResource: cleanResources,
		})
		if deperr != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback err = %s", errMsg, deperr.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		if !clean {
			createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback deleting", errMsg))
		}
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	// 4-1. ResizeVMDisk
	_, err = resizeVMOsDisk(vmReqInfo.RootDiskSize, vmReqInfo.IId, vmHandler.Region.Region, vmHandler.Client, vmHandler.DiskClient, vmHandler.Ctx)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Finished to rollback deleting", err.Error()))
		cleanErr := vmHandler.cleanDeleteVm(vmReqInfo.IId)
		if cleanErr != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback err = %s", err.Error(), cleanErr.Error()))
		}
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	curRetryCnt := 0
	maxRetryCnt := 120
	// 5. Wait Running
	for {
		resp, _ := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.Region, vmReqInfo.IId.NameId, nil)
		// Get powerState, provisioningState
		vmStatus := getVmStatus(resp.VirtualMachineInstanceView)
		if vmStatus == irs.Running {
			break
		}
		curRetryCnt++
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			createErr := errors.New(fmt.Sprintf("Failed to Start VM. exceeded maximum retry count %d and Finished to rollback deleting", maxRetryCnt))
			cleanErr := vmHandler.cleanDeleteVm(vmReqInfo.IId)
			if cleanErr != nil {
				createErr = errors.New(fmt.Sprintf("Failed to Start VM. exceeded maximum retry count %d and Failed to rollback err = %s", maxRetryCnt, cleanErr.Error()))
			}
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
	}
	// 6. Window user Change
	if imageOsType == irs.WINDOWS {
		if vmReqInfo.ImageType == "" || vmReqInfo.ImageType == irs.PublicImage {
			err = createAdministratorUser(vmReqInfo.IId, WindowBaseUser, vmReqInfo.VMUserPasswd, vmHandler.Client, vmHandler.VirtualMachineRunCommandsClient, vmHandler.Ctx, vmHandler.Region)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Finished to rollback deleting", err.Error()))
				cleanErr := vmHandler.cleanDeleteVm(vmReqInfo.IId)
				if cleanErr != nil {
					createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback err = %s", err.Error(), cleanErr.Error()))
				}
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
		} else {
			err = changeUserPassword(vmReqInfo.IId, WindowBaseUser, vmReqInfo.VMUserPasswd, vmHandler.Client, vmHandler.VirtualMachineRunCommandsClient, vmHandler.Ctx, vmHandler.Region)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Finished to rollback deleting", err.Error()))
				cleanErr := vmHandler.cleanDeleteVm(vmReqInfo.IId)
				if cleanErr != nil {
					createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback err = %s", err.Error(), cleanErr.Error()))
				}
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
		}

	}
	// 7. If DataDisk Exist
	if len(vmReqInfo.DataDiskIIDs) > 0 {
		vm, err := AttachList(vmReqInfo.DataDiskIIDs, vmReqInfo.IId, vmHandler.CredentialInfo, vmHandler.Region, vmHandler.Ctx, vmHandler.Client, vmHandler.DiskClient)
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Finished to rollback deleting", err.Error()))
			cleanErr := vmHandler.cleanDeleteVm(vmReqInfo.IId)
			if cleanErr != nil {
				createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback err = %s", err.Error(), cleanErr.Error()))
			}
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		vmInfo := vmHandler.mappingServerInfo(vm)
		LoggingInfo(hiscallInfo, start)
		return vmInfo, nil
	} else {
		resp, err := vmHandler.Client.Get(vmHandler.Ctx, vmHandler.Region.Region, vmReqInfo.IId.NameId, nil)
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback deleting", err.Error()))
			cleanErr := vmHandler.cleanDeleteVm(vmReqInfo.IId)
			if cleanErr != nil {
				createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback err = %s", err.Error(), cleanErr.Error()))
			}
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		vmInfo := vmHandler.mappingServerInfo(resp.VirtualMachine)
		if imageOsType == irs.WINDOWS {
			vmInfo.VMUserPasswd = vmReqInfo.VMUserPasswd
		}
		LoggingInfo(hiscallInfo, start)
		return vmInfo, nil
	}
}

func (vmHandler *AzureVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "SuspendVM()")
	// running => suspend

	convertedIID, err := ConvertVMIID(vmIID, vmHandler.CredentialInfo, vmHandler.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Suspend VM. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}

	// Check VM Exist
	exist, err := CheckExistVM(convertedIID, vmHandler.Region.Region, vmHandler.Client, vmHandler.Ctx)

	if err != nil {
		suspendErr := errors.New(fmt.Sprintf("Failed to Suspend VM. err = %s", err))
		cblogger.Error(suspendErr.Error())
		LoggingError(hiscallInfo, suspendErr)
		return irs.Failed, suspendErr
	}
	if !exist {
		suspendErr := errors.New(fmt.Sprintf("Failed to Suspend VM. err = not exist vm"))
		cblogger.Error(suspendErr.Error())
		LoggingError(hiscallInfo, suspendErr)
		return irs.Failed, suspendErr
	}
	resp, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.Region, convertedIID.NameId, nil)
	if err != nil {
		suspendErr := errors.New(fmt.Sprintf("Failed to Suspend VM. err = %s", err))
		cblogger.Error(suspendErr.Error())
		LoggingError(hiscallInfo, suspendErr)
		return irs.Failed, suspendErr
	}
	vmStatus := getVmStatus(resp.VirtualMachineInstanceView)
	if vmStatus == irs.Running {
		start := call.Start()
		poller, err := vmHandler.Client.BeginPowerOff(vmHandler.Ctx, vmHandler.Region.Region, convertedIID.NameId, nil)
		if err != nil {
			suspendErr := errors.New(fmt.Sprintf("Failed to Suspend VM. err = %s", err))
			cblogger.Error(suspendErr.Error())
			LoggingError(hiscallInfo, suspendErr)
			return irs.Failed, suspendErr
		}
		_, err = poller.PollUntilDone(vmHandler.Ctx, nil)
		if err != nil {
			suspendErr := errors.New(fmt.Sprintf("Failed to Suspend VM. err = %s", err))
			cblogger.Error(suspendErr.Error())
			LoggingError(hiscallInfo, suspendErr)
			return irs.Failed, suspendErr
		}
		resp, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.Region, convertedIID.NameId, nil)
		if err != nil {
			suspendErr := errors.New(fmt.Sprintf("Failed to Suspend VM. but Failed Get Status err = %s", err))
			cblogger.Error(suspendErr.Error())
			LoggingError(hiscallInfo, suspendErr)
			return irs.Failed, suspendErr
		}
		vmStatus = getVmStatus(resp.VirtualMachineInstanceView)
		LoggingInfo(hiscallInfo, start)
		return vmStatus, nil
	}
	suspendErr := errors.New(fmt.Sprintf("Failed to Suspend VM. err = Cannot Suspend VM Status is %s ", vmStatus))
	cblogger.Error(suspendErr.Error())
	LoggingError(hiscallInfo, suspendErr)
	return irs.Failed, suspendErr
}

func (vmHandler *AzureVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "ResumeVM()")

	convertedIID, err := ConvertVMIID(vmIID, vmHandler.CredentialInfo, vmHandler.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Resume VM. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	// Suspend => running

	// Check VM Exist
	exist, err := CheckExistVM(convertedIID, vmHandler.Region.Region, vmHandler.Client, vmHandler.Ctx)

	if err != nil {
		resumeErr := errors.New(fmt.Sprintf("Failed to Resume VM. err = %s", err))
		cblogger.Error(resumeErr.Error())
		LoggingError(hiscallInfo, resumeErr)
		return irs.Failed, resumeErr
	}
	if !exist {
		resumeErr := errors.New(fmt.Sprintf("Failed to Resume VM. err = not exist vm"))
		cblogger.Error(resumeErr.Error())
		LoggingError(hiscallInfo, resumeErr)
		return irs.Failed, resumeErr
	}
	resp, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.Region, convertedIID.NameId, nil)
	if err != nil {
		resumeErr := errors.New(fmt.Sprintf("Failed to Resume VM. err = %s", err))
		cblogger.Error(resumeErr.Error())
		LoggingError(hiscallInfo, resumeErr)
		return irs.Failed, resumeErr
	}
	vmStatus := getVmStatus(resp.VirtualMachineInstanceView)
	if vmStatus == irs.Suspended {
		start := call.Start()
		poller, err := vmHandler.Client.BeginStart(vmHandler.Ctx, vmHandler.Region.Region, convertedIID.NameId, nil)
		if err != nil {
			resumeErr := errors.New(fmt.Sprintf("Failed to Resume VM. err = %s", err))
			cblogger.Error(resumeErr.Error())
			LoggingError(hiscallInfo, resumeErr)
			return irs.Failed, resumeErr
		}
		_, err = poller.PollUntilDone(vmHandler.Ctx, nil)
		if err != nil {
			resumeErr := errors.New(fmt.Sprintf("Failed to Resume VM. err = %s", err))
			cblogger.Error(resumeErr.Error())
			LoggingError(hiscallInfo, resumeErr)
			return irs.Failed, resumeErr
		}
		resp, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.Region, convertedIID.NameId, nil)
		if err != nil {
			suspendErr := errors.New(fmt.Sprintf("Finish to Suspend VM. but Failed Get Status err = %s", err))
			cblogger.Error(suspendErr.Error())
			LoggingError(hiscallInfo, suspendErr)
			return irs.Failed, suspendErr
		}
		vmStatus = getVmStatus(resp.VirtualMachineInstanceView)
		LoggingInfo(hiscallInfo, start)
		return vmStatus, nil
	}
	resumeErr := errors.New(fmt.Sprintf("Failed to Resume VM. err = Cannot Resume VM Status is %s ", vmStatus))
	cblogger.Error(resumeErr.Error())
	LoggingError(hiscallInfo, resumeErr)
	return irs.Failed, resumeErr
}

func (vmHandler *AzureVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "RebootVM()")

	convertedIID, err := ConvertVMIID(vmIID, vmHandler.CredentialInfo, vmHandler.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Reboot VM. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}

	// Check VM Exist
	exist, err := CheckExistVM(convertedIID, vmHandler.Region.Region, vmHandler.Client, vmHandler.Ctx)

	if err != nil {
		rebootErr := errors.New(fmt.Sprintf("Failed to Reboot VM. err = %s", err))
		cblogger.Error(rebootErr.Error())
		LoggingError(hiscallInfo, rebootErr)
		return irs.Failed, rebootErr
	}
	if !exist {
		rebootErr := errors.New(fmt.Sprintf("Failed to Reboot VM. err = not exist vm"))
		cblogger.Error(rebootErr.Error())
		LoggingError(hiscallInfo, rebootErr)
		return irs.Failed, rebootErr
	}
	resp, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.Region, convertedIID.NameId, nil)
	if err != nil {
		rebootErr := errors.New(fmt.Sprintf("Failed to Reboot VM. err = %s", err))
		cblogger.Error(rebootErr.Error())
		LoggingError(hiscallInfo, rebootErr)
		return irs.Failed, rebootErr
	}
	vmStatus := getVmStatus(resp.VirtualMachineInstanceView)
	if vmStatus == irs.Running {
		start := call.Start()
		poller, err := vmHandler.Client.BeginRestart(vmHandler.Ctx, vmHandler.Region.Region, convertedIID.NameId, nil)
		if err != nil {
			rebootErr := errors.New(fmt.Sprintf("Failed to Reboot VM. err = %s", err))
			cblogger.Error(rebootErr.Error())
			LoggingError(hiscallInfo, rebootErr)
			return irs.Failed, rebootErr
		}
		_, err = poller.PollUntilDone(vmHandler.Ctx, nil)
		if err != nil {
			rebootErr := errors.New(fmt.Sprintf("Failed to Reboot VM. err = %s", err))
			cblogger.Error(rebootErr.Error())
			LoggingError(hiscallInfo, rebootErr)
			return irs.Failed, rebootErr
		}
		resp, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.Region, convertedIID.NameId, nil)
		if err != nil {
			suspendErr := errors.New(fmt.Sprintf("Failed to Suspend VM. but Failed Get Status err = %s", err))
			cblogger.Error(suspendErr.Error())
			LoggingError(hiscallInfo, suspendErr)
			return irs.Failed, suspendErr
		}
		vmStatus = getVmStatus(resp.VirtualMachineInstanceView)
		LoggingInfo(hiscallInfo, start)
		return vmStatus, nil
	}
	rebootErr := errors.New(fmt.Sprintf("Failed to Reboot VM. err = Cannot Reboot VM Status is %s ", vmStatus))
	cblogger.Error(rebootErr.Error())
	LoggingError(hiscallInfo, rebootErr)
	return irs.Failed, rebootErr
}

func (vmHandler *AzureVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "TerminateVM()")
	start := call.Start()
	err := vmHandler.cleanDeleteVm(vmIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Terminate VM. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	LoggingInfo(hiscallInfo, start)

	return irs.NotExist, nil
}

func (vmHandler *AzureVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, VM, "ListVMStatus()")

	start := call.Start()

	var vmList []*armcompute.VirtualMachine

	pager := vmHandler.Client.NewListPager(vmHandler.Region.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(vmHandler.Ctx)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List VMStatus. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return []*irs.VMStatusInfo{}, getErr
		}

		for _, vm := range page.Value {
			vmList = append(vmList, vm)
		}
	}

	LoggingInfo(hiscallInfo, start)

	var vmStatusList []*irs.VMStatusInfo

	for _, vm := range vmList {
		if vm.Properties.InstanceView != nil {
			statusStr := getVmStatus(*vm.Properties.InstanceView)
			status := statusStr
			vmStatusInfo := irs.VMStatusInfo{
				IId: irs.IID{
					NameId:   *vm.Name,
					SystemId: *vm.ID,
				},
				VmStatus: status,
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		} else {
			vmIdArr := strings.Split(*vm.ID, "/")
			vmName := vmIdArr[8]
			status, _ := vmHandler.GetVMStatus(irs.IID{NameId: vmName, SystemId: *vm.ID})
			vmStatusInfo := irs.VMStatusInfo{
				IId: irs.IID{
					NameId:   *vm.Name,
					SystemId: *vm.ID,
				},
				VmStatus: status,
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		}
	}
	return vmStatusList, nil
}

func (vmHandler *AzureVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "GetVMStatus()")

	convertedIID, err := ConvertVMIID(vmIID, vmHandler.CredentialInfo, vmHandler.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VM. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	start := call.Start()
	resp, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.Region, convertedIID.NameId, nil)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VM. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	LoggingInfo(hiscallInfo, start)

	// Get powerState, provisioningState
	vmStatus := getVmStatus(resp.VirtualMachineInstanceView)
	return vmStatus, nil
}

func (vmHandler *AzureVMHandler) ListVM() ([]*irs.VMInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, VM, "ListVM()")

	start := call.Start()

	var vmList []*armcompute.VirtualMachine

	pager := vmHandler.Client.NewListPager(vmHandler.Region.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(vmHandler.Ctx)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List VMStatus. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return []*irs.VMInfo{}, getErr
		}

		for _, vm := range page.Value {
			vmList = append(vmList, vm)
		}
	}

	LoggingInfo(hiscallInfo, start)

	var vmInfoList []*irs.VMInfo
	for _, vm := range vmList {
		vmInfo := vmHandler.mappingServerInfo(*vm)
		vmInfoList = append(vmInfoList, &vmInfo)
	}

	return vmInfoList, nil
}

func (vmHandler *AzureVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "GetVM()")
	start := call.Start()
	convertedIID, err := ConvertVMIID(vmIID, vmHandler.CredentialInfo, vmHandler.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VM. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VMInfo{}, getErr
	}

	vm, err := GetRawVM(convertedIID, vmHandler.Region.Region, vmHandler.Client, vmHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VM. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VMInfo{}, getErr
	}
	// addAdministratorUser(convertedIID, vmHandler.Client, vmHandler.VirtualMachineRunCommandsClient, vmHandler.Ctx, vmHandler.Region)

	LoggingInfo(hiscallInfo, start)

	vmInfo := vmHandler.mappingServerInfo(vm)
	return vmInfo, nil
}

func getVmStatus(instanceView armcompute.VirtualMachineInstanceView) irs.VMStatus {
	var powerState, provisioningState string

	for _, stat := range instanceView.Statuses {
		statArr := strings.Split(*stat.Code, "/")

		if statArr[0] == "PowerState" {
			powerState = strings.ToLower(statArr[1])
		} else if statArr[0] == "ProvisioningState" {
			provisioningState = strings.ToLower(statArr[1])
		}
	}

	if strings.EqualFold(provisioningState, "failed") {
		return irs.Failed
	}

	// Set VM Status Info
	resultStatus := irs.Failed

	if provisioningState == "creating" {
		resultStatus = irs.Creating
	}
	if provisioningState == "succeeded" && powerState == "running" {
		resultStatus = irs.Running
	}
	if provisioningState == "updating" && powerState == "stopping" {
		resultStatus = irs.Suspending
	}
	if provisioningState == "succeeded" && powerState == "stopped" {
		resultStatus = irs.Suspended
	}
	if provisioningState == "updating" && powerState == "starting" {
		resultStatus = irs.Resuming
	}
	if provisioningState == "succeeded" && powerState == "starting" {
		resultStatus = irs.Rebooting
	}
	if provisioningState == "deleting" {
		resultStatus = irs.Terminating
	}
	return resultStatus
}

func (vmHandler *AzureVMHandler) cleanDeleteVm(vmIId irs.IID) error {
	convertedIID, err := ConvertVMIID(vmIId, vmHandler.CredentialInfo, vmHandler.Region)
	exist, err := CheckExistVM(convertedIID, vmHandler.Region.Region, vmHandler.Client, vmHandler.Ctx)
	if err != nil {
		return err
	}
	if exist {
		vm, err := GetRawVM(convertedIID, vmHandler.Region.Region, vmHandler.Client, vmHandler.Ctx)
		if err != nil {
			return err
		}
		vmInfo := vmHandler.mappingServerInfo(vm)
		cleanVMClientSet := CleanVMClientSet{
			VPCName:    vmInfo.VpcIID.NameId,
			SubnetName: vmInfo.SubnetIID.NameId,
		}
		cleanResources := CleanVMClientRequestResource{
			"", vmInfo.NetworkInterface, "",
		}
		if vm.Properties.StorageProfile.OSDisk.Name != nil {
			cleanResources.VmDiskName = *vm.Properties.StorageProfile.OSDisk.Name
		}
		vNic, vNicErr := vmHandler.NicClient.Get(vmHandler.Ctx, vmHandler.Region.Region, vmInfo.NetworkInterface, nil)
		if vNicErr != nil {
			return vNicErr
		}
		for _, ip := range vNic.Properties.IPConfigurations {
			if ip.Properties.Primary != nil && *ip.Properties.Primary {
				if ip.Properties.PublicIPAddress != nil && ip.Properties.PublicIPAddress.ID != nil {
					publicipIdAddr := strings.Split(*ip.Properties.PublicIPAddress.ID, "/")
					cleanResources.PublicIPName = publicipIdAddr[len(publicipIdAddr)-1]
				}
			}
		}
		forceDelete := true
		poller, vmDeleteErr := vmHandler.Client.BeginDelete(vmHandler.Ctx, vmHandler.Region.Region, *vm.Name, &armcompute.VirtualMachinesClientBeginDeleteOptions{
			ForceDeletion: &forceDelete,
		})
		if vmDeleteErr != nil {
			return vmDeleteErr
		}
		_, err = poller.PollUntilDone(vmHandler.Ctx, nil)
		if err != nil {
			return err
		}
		_, deperr := vmHandler.cleanVMRelatedResource(VMCleanRelatedResource{
			RequiredSet:         cleanVMClientSet,
			CleanTargetResource: cleanResources,
		})
		if deperr != nil {
			return vmDeleteErr
		}
	}
	return nil
}

func (vmHandler *AzureVMHandler) mappingServerInfo(server armcompute.VirtualMachine) irs.VMInfo {
	// Get Default VM Info
	vmInfo := irs.VMInfo{
		IId: irs.IID{
			NameId:   *server.Name,
			SystemId: *server.ID,
		},
		Region: irs.RegionInfo{
			Region: *server.Location,
		},
		VMSpecName:     string(*server.Properties.HardwareProfile.VMSize),
		RootDeviceName: "Not visible in Azure",
		VMBlockDisk:    "Not visible in Azure",
	}

	// Set VM Zone
	if server.Zones != nil && len(server.Zones) > 0 {
		vmInfo.Region.Zone = *server.Zones[0]
	}

	// Set VM Image Info
	if reflect.ValueOf(server.Properties.StorageProfile.ImageReference.ID).IsNil() {
		imageRef := server.Properties.StorageProfile.ImageReference
		vmInfo.ImageIId.SystemId = *imageRef.Publisher + ":" + *imageRef.Offer + ":" + *imageRef.SKU + ":" + *imageRef.Version
		vmInfo.ImageIId.NameId = *imageRef.Publisher + ":" + *imageRef.Offer + ":" + *imageRef.SKU + ":" + *imageRef.Version
		//vmInfo.ImageIId.SystemId = vmInfo.ImageIId.NameId
	} else {
		vmInfo.ImageIId.SystemId = *server.Properties.StorageProfile.ImageReference.ID
		vmInfo.ImageIId.NameId = *server.Properties.StorageProfile.ImageReference.ID
		//vmInfo.ImageIId.SystemId = vmInfo.ImageIId.NameId
	}

	// Get VNic ID
	niList := server.Properties.NetworkProfile.NetworkInterfaces
	var VNicId string
	for _, ni := range niList {
		if ni.ID != nil {
			VNicId = *ni.ID
		}
	}

	// Get VNic
	nicIdArr := strings.Split(VNicId, "/")
	nicName := nicIdArr[len(nicIdArr)-1]
	vNic, _ := vmHandler.NicClient.Get(vmHandler.Ctx, vmHandler.Region.Region, nicName, nil)
	vmInfo.NetworkInterface = nicName

	// Get SecurityGroup
	sgGroupIdArr := strings.Split(*vNic.Properties.NetworkSecurityGroup.ID, "/")
	sgGroupName := sgGroupIdArr[len(sgGroupIdArr)-1]
	vmInfo.SecurityGroupIIds = []irs.IID{
		{
			NameId:   sgGroupName,
			SystemId: *vNic.Properties.NetworkSecurityGroup.ID,
		},
	}

	// Get PrivateIP, PublicIpId
	for _, ip := range vNic.Properties.IPConfigurations {
		if ip.Properties.Primary != nil && *ip.Properties.Primary {
			// PrivateIP 정보 설정
			vmInfo.PrivateIP = *ip.Properties.PrivateIPAddress

			// PublicIP 정보 조회 및 설정
			if ip.Properties.PublicIPAddress != nil {
				publicIPId := *ip.Properties.PublicIPAddress.ID
				publicIPIdArr := strings.Split(publicIPId, "/")
				publicIPName := publicIPIdArr[len(publicIPIdArr)-1]

				publicIP, _ := vmHandler.PublicIPClient.Get(vmHandler.Ctx, vmHandler.Region.Region, publicIPName, nil)
				if publicIP.Properties.IPAddress != nil {
					vmInfo.PublicIP = *publicIP.Properties.IPAddress
				}
			}

			// Get Subnet
			subnetIdArr := strings.Split(*ip.Properties.Subnet.ID, "/")
			subnetName := subnetIdArr[len(subnetIdArr)-1]
			vmInfo.SubnetIID = irs.IID{NameId: subnetName, SystemId: *ip.Properties.Subnet.ID}

			// Get VPC
			vpcIdArr := subnetIdArr[:len(subnetIdArr)-2]
			vpcName := vpcIdArr[len(vpcIdArr)-1]
			vmInfo.VpcIID = irs.IID{NameId: vpcName, SystemId: strings.Join(vpcIdArr, "/")}
		}
	}
	osType := getOSTypeByVM(server)
	if osType == irs.WINDOWS {
		vmInfo.VMUserId = WindowBaseUser
	}
	if osType == irs.LINUX_UNIX {
		vmInfo.VMUserId = CBVMUser
	}
	// Set GuestUser Id/Pwd
	//if server.VirtualMachineProperties.OsProfile.AdminUsername != nil {
	//	vmInfo.VMUserId = *server.VirtualMachineProperties.OsProfile.AdminUsername
	//}
	if server.Properties.OSProfile.AdminPassword != nil {
		vmInfo.VMUserPasswd = *server.Properties.OSProfile.AdminPassword
	}

	// Set BootDisk
	diskHandler := AzureDiskHandler{
		CredentialInfo: vmHandler.CredentialInfo,
		Region:         vmHandler.Region,
		Ctx:            vmHandler.Ctx,
		VMClient:       vmHandler.Client,
		DiskClient:     vmHandler.DiskClient,
	}
	diskInfo, _ := diskHandler.GetDisk(irs.IID{NameId: *server.Properties.StorageProfile.OSDisk.Name})
	vmInfo.VMBootDisk = diskInfo.IId.NameId
	vmInfo.RootDiskSize = diskInfo.DiskSize
	vmInfo.RootDiskType = diskInfo.DiskType

	// Get StartTime
	if server.Properties.InstanceView != nil {
		for _, status := range server.Properties.InstanceView.Statuses {
			if strings.EqualFold(*status.Code, ProvisioningStateCode) {
				vmInfo.StartTime = status.Time.UTC()
				break
			}
		}
	}

	// Get Keypair
	tagList := server.Tags
	for key, val := range tagList {
		if key == "keypair" && val != nil {
			vmInfo.KeyPairIId = irs.IID{NameId: *val, SystemId: GetSshKeyIdByName(vmHandler.CredentialInfo, vmHandler.Region, *val)}
		}
		if key == "publicip" && val != nil {
			vmInfo.KeyValueList = []irs.KeyValue{
				{Key: "publicip", Value: *val},
			}
		}
	}

	if server.Properties.StorageProfile != nil && server.Properties.StorageProfile.DataDisks != nil && len(server.Properties.StorageProfile.DataDisks) > 0 {
		dataDisks := server.Properties.StorageProfile.DataDisks
		dataDiskIIDList := make([]irs.IID, len(dataDisks))
		for i, dataDisk := range dataDisks {
			diskId := *dataDisk.ManagedDisk.ID
			dataDiskIIDList[i] = irs.IID{
				NameId:   GetResourceNameById(diskId),
				SystemId: diskId,
			}
		}
		vmInfo.DataDiskIIDs = dataDiskIIDList
	}
	osPlatform := getOSTypeByVM(server)
	vmInfo.Platform = osPlatform
	if vmInfo.PublicIP != "" {
		if osPlatform == irs.WINDOWS {
			vmInfo.AccessPoint = fmt.Sprintf("%s:%s", vmInfo.PublicIP, "3389")
		} else {
			vmInfo.AccessPoint = fmt.Sprintf("%s:%s", vmInfo.PublicIP, "22")
		}
	}
	if server.Tags != nil {
		vmInfo.TagList = setTagList(server.Tags)
	}

	return vmInfo
}

// VM 생성 시 Public IP 자동 생성 (nested flow 적용)
func CreatePublicIP(vmHandler *AzureVMHandler, vmReqInfo irs.VMReqInfo) (irs.IID, error) {
	// PublicIP 이름 생성
	publicIPName := generatePublicIPName(vmReqInfo.IId.NameId)

	publicIPAddressSKUNameBasic := armnetwork.PublicIPAddressSKUNameBasic
	publicIPAddressVersion := armnetwork.IPVersionIPv4
	publicIPAllocationMethod := armnetwork.IPAllocationMethodStatic
	createOpts := armnetwork.PublicIPAddress{
		Name: &publicIPName,
		SKU: &armnetwork.PublicIPAddressSKU{
			Name: &publicIPAddressSKUNameBasic,
		},
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAddressVersion:   &publicIPAddressVersion,
			PublicIPAllocationMethod: &publicIPAllocationMethod,
			IdleTimeoutInMinutes:     toInt32Ptr(4),
		},
		Location: &vmHandler.Region.Region,
		Tags: map[string]*string{
			"createdBy": &vmReqInfo.IId.NameId,
		},
	}

	publicIPAddressSKUNameStandard := armnetwork.PublicIPAddressSKUNameStandard
	// Setting zone if available
	if vmHandler.Region.TargetZone != "" || vmHandler.Region.Zone != "" {
		createOpts.SKU = &armnetwork.PublicIPAddressSKU{
			Name: &publicIPAddressSKUNameStandard,
		}
		createOpts.Properties.PublicIPAllocationMethod = &publicIPAllocationMethod
		if vmHandler.Region.TargetZone != "" {
			createOpts.Zones = []*string{
				toStrPtr(vmHandler.Region.TargetZone),
			}
		} else {
			createOpts.Zones = []*string{
				toStrPtr(vmHandler.Region.Zone),
			}
		}
	}

	poller, err := vmHandler.PublicIPClient.BeginCreateOrUpdate(vmHandler.Ctx, vmHandler.Region.Region, publicIPName, createOpts, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create PublicIP, error=%s", err))
		return irs.IID{}, createErr
	}
	_, err = poller.PollUntilDone(vmHandler.Ctx, nil)
	if err != nil {
		return irs.IID{}, err
	}

	// 생성된 PublicIP 정보 리턴
	publicIPInfo, err := vmHandler.PublicIPClient.Get(vmHandler.Ctx, vmHandler.Region.Region, publicIPName, nil)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get PublicIP, error=%s", err))
		return irs.IID{}, getErr
	}
	publicIPIId := irs.IID{NameId: *publicIPInfo.Name, SystemId: *publicIPInfo.ID}
	return publicIPIId, nil
}

type VMCleanRelatedResource struct {
	CleanTargetResource CleanVMClientRequestResource
	RequiredSet         CleanVMClientSet
}

type CleanVMClientRequestResource struct {
	PublicIPName         string
	NetworkInterfaceName string
	VmDiskName           string
}

type CleanVMClientSet struct {
	VPCName    string
	SubnetName string
}

// VMCleanRelatedResource
func (vmHandler *AzureVMHandler) cleanVMRelatedResource(cleanRelatedResource VMCleanRelatedResource) (bool, error) {
	curRetryCnt := 0
	maxRetryCnt := 120

	networkInterfaceName := cleanRelatedResource.CleanTargetResource.NetworkInterfaceName
	publicIPId := cleanRelatedResource.CleanTargetResource.PublicIPName
	vmDiskId := cleanRelatedResource.CleanTargetResource.VmDiskName
	resourceGroup := vmHandler.Region.Region

	// VNic Delete
	if networkInterfaceName != "" {
		vnicExist, _ := CheckExistVNic(networkInterfaceName, resourceGroup, vmHandler.NicClient, vmHandler.Ctx)
		resp, subnetgetErr := vmHandler.SubnetClient.Get(vmHandler.Ctx, resourceGroup, cleanRelatedResource.RequiredSet.VPCName, cleanRelatedResource.RequiredSet.SubnetName, nil)
		if subnetgetErr != nil {
			return false, subnetgetErr
		}
		var ipConfigArr []*armnetwork.InterfaceIPConfiguration
		privateIPAllocationMethod := armnetwork.IPAllocationMethodDynamic
		ipConfig := &armnetwork.InterfaceIPConfiguration{
			Name: toStrPtr("ipConfig1"),
			Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
				Subnet:                    &resp.Subnet,
				PrivateIPAllocationMethod: &privateIPAllocationMethod,
				PublicIPAddress:           nil,
			},
		}
		ipConfigArr = append(ipConfigArr, ipConfig)

		detachOpts := armnetwork.Interface{
			Properties: &armnetwork.InterfacePropertiesFormat{
				IPConfigurations: ipConfigArr,
			},
			Location: toStrPtr(vmHandler.Region.Region),
		}
		if vnicExist {
			for {
				vnicExist, _ = CheckExistVNic(networkInterfaceName, resourceGroup, vmHandler.NicClient, vmHandler.Ctx)
				if !vnicExist {
					break
				}

				curRetryCnt++
				time.Sleep(1 * time.Second)
				if curRetryCnt > maxRetryCnt {
					createErr := errors.New(fmt.Sprintf("Failed to clean remained vnic ("+networkInterfaceName+"). exceeded maximum retry count %d", maxRetryCnt))
					cblogger.Warn(createErr.Error())
				}

				poller, _ := vmHandler.NicClient.BeginCreateOrUpdate(vmHandler.Ctx, resourceGroup, networkInterfaceName, detachOpts, nil)
				if poller != nil {
					_, _ = poller.PollUntilDone(vmHandler.Ctx, nil)
				}
				poller2, _ := vmHandler.NicClient.BeginDelete(vmHandler.Ctx, resourceGroup, networkInterfaceName, nil)
				if poller2 != nil {
					_, _ = poller2.PollUntilDone(vmHandler.Ctx, nil)
				}
			}
		}
	}

	// publicIPId Delete
	if publicIPId != "" {
		publicIPExist, err := CheckExistPublicIp(publicIPId, resourceGroup, vmHandler.PublicIPClient, vmHandler.Ctx)
		if err != nil {
			return false, err
		}
		if publicIPExist {
			for {
				publicIPExist, _ = CheckExistPublicIp(publicIPId, resourceGroup, vmHandler.PublicIPClient, vmHandler.Ctx)
				if !publicIPExist {
					break
				}

				curRetryCnt++
				time.Sleep(1 * time.Second)
				if curRetryCnt > maxRetryCnt {
					createErr := errors.New(fmt.Sprintf("Failed to clean remained public IP ("+publicIPId+"). exceeded maximum retry count %d", maxRetryCnt))
					cblogger.Warn(createErr.Error())
				}

				poller, _ := vmHandler.PublicIPClient.BeginDelete(vmHandler.Ctx, resourceGroup, publicIPId, nil)
				if poller != nil {
					_, _ = poller.PollUntilDone(vmHandler.Ctx, nil)
				}
			}
		}
	}

	// Disk Delete
	if vmDiskId != "" {
		vmDiskExist, err := CheckExistVMDisk(vmDiskId, vmHandler.DiskClient, vmHandler.Ctx)
		if err != nil {
			return false, err
		}
		if vmDiskExist {
			for {
				vmDiskExist, _ = CheckExistVMDisk(vmDiskId, vmHandler.DiskClient, vmHandler.Ctx)
				if !vmDiskExist {
					break
				}

				curRetryCnt++
				time.Sleep(1 * time.Second)
				if curRetryCnt > maxRetryCnt {
					createErr := errors.New(fmt.Sprintf("Failed to clean remained disk ("+vmDiskId+"). exceeded maximum retry count %d", maxRetryCnt))
					cblogger.Warn(createErr.Error())
				}

				poller, _ := vmHandler.DiskClient.BeginDelete(vmHandler.Ctx, resourceGroup, vmDiskId, nil)
				if poller != nil {
					_, _ = poller.PollUntilDone(vmHandler.Ctx, nil)
				}
			}
		}
	}
	return true, nil
}

func CheckExistPublicIp(publicIPId string, resourceGroup string, client *armnetwork.PublicIPAddressesClient, ctx context.Context) (bool, error) {
	var publicIPList []*armnetwork.PublicIPAddress

	pager := client.NewListPager(resourceGroup, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return false, err
		}

		for _, publicIP := range page.Value {
			publicIPList = append(publicIPList, publicIP)
		}
	}

	for _, publicIp := range publicIPList {
		if *publicIp.Name == publicIPId {
			return true, nil
		}
	}
	return false, nil
}

// VM 생성 시 VNic 자동 생성 (nested flow 적용)
func CreateVNic(vmHandler *AzureVMHandler, vmReqInfo irs.VMReqInfo, publicIPIId irs.IID) (irs.IID, error) {
	// VNic 이름 생성
	VNicName := generateVNicName(vmReqInfo.IId.NameId)
	// 리소스 Id 정보 매핑
	// Azure의 경우 VNic에 1개의 보안그룹만 할당 가능
	secGroupId := GetSecGroupIdByName(vmHandler.CredentialInfo, vmHandler.Region, vmReqInfo.SecurityGroupIIDs[0].NameId)
	resp, err := vmHandler.SubnetClient.Get(vmHandler.Ctx, vmHandler.Region.Region, vmReqInfo.VpcIID.NameId, vmReqInfo.SubnetIID.NameId, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create NetworkInterface, error=%s", err))
		return irs.IID{}, createErr
	}

	var ipConfigArr []*armnetwork.InterfaceIPConfiguration
	privateIPAllocationMethod := armnetwork.IPAllocationMethodDynamic
	ipConfig := &armnetwork.InterfaceIPConfiguration{
		Name: toStrPtr("ipConfig1"),
		Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
			Subnet:                    &resp.Subnet,
			PrivateIPAllocationMethod: &privateIPAllocationMethod,
			PublicIPAddress: &armnetwork.PublicIPAddress{
				ID: &publicIPIId.SystemId,
			},
		},
	}
	ipConfigArr = append(ipConfigArr, ipConfig)

	createOpts := armnetwork.Interface{
		Properties: &armnetwork.InterfacePropertiesFormat{
			IPConfigurations: ipConfigArr,
			NetworkSecurityGroup: &armnetwork.SecurityGroup{
				ID: &secGroupId,
			},
		},
		Location: &vmHandler.Region.Region,
		Tags: map[string]*string{
			"createdBy": &vmReqInfo.IId.NameId,
		},
	}
	poller, err := vmHandler.NicClient.BeginCreateOrUpdate(vmHandler.Ctx, vmHandler.Region.Region, VNicName, createOpts, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create NetworkInterface, error=%s", err))
		return irs.IID{}, createErr
	}
	_, err = poller.PollUntilDone(vmHandler.Ctx, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create NetworkInterface, error=%s", err))
		return irs.IID{}, createErr
	}

	// 생성된 VNic 정보 리턴
	resp2, err := vmHandler.NicClient.Get(vmHandler.Ctx, vmHandler.Region.Region, VNicName, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create NetworkInterface, error=%s", err))
		return irs.IID{}, createErr
	}
	VNicIId := irs.IID{NameId: *resp2.Interface.Name, SystemId: *resp2.Interface.ID}
	return VNicIId, nil
}

func CheckExistVNic(networkInterfaceName string, resourceGroup string, client *armnetwork.InterfacesClient, ctx context.Context) (bool, error) {
	var interfaceList []*armnetwork.Interface

	pager := client.NewListPager(resourceGroup, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return false, err
		}

		for _, iface := range page.Value {
			interfaceList = append(interfaceList, iface)
		}
	}

	for _, iface := range interfaceList {
		if *iface.Name == networkInterfaceName {
			return true, nil
		}
	}
	return false, nil
}

func CheckExistVMDisk(osDiskName string, client *armcompute.DisksClient, ctx context.Context) (bool, error) {
	var diskList []*armcompute.Disk

	pager := client.NewListPager(nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return false, err
		}

		for _, disk := range page.Value {
			diskList = append(diskList, disk)
		}
	}

	for _, vmDisk := range diskList {
		if *vmDisk.Name == osDiskName {
			return true, nil
		}
	}
	return false, nil
}

func CheckExistVM(vmIID irs.IID, resourceGroup string, client *armcompute.VirtualMachinesClient, ctx context.Context) (bool, error) {
	var vmList []*armcompute.VirtualMachine

	pager := client.NewListPager(resourceGroup, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return false, err
		}

		for _, vm := range page.Value {
			vmList = append(vmList, vm)
		}
	}

	for _, vm := range vmList {
		if vmIID.SystemId != "" && vmIID.SystemId == *vm.ID {
			return true, nil
		}
		if vmIID.NameId != "" && vmIID.NameId == *vm.Name {
			return true, nil
		}
	}
	return false, nil
}

func GetRawVM(vmIID irs.IID, resourceGroup string, client *armcompute.VirtualMachinesClient, ctx context.Context) (armcompute.VirtualMachine, error) {
	if vmIID.NameId == "" {
		var vmList []*armcompute.VirtualMachine

		pager := client.NewListPager(resourceGroup, nil)

		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return armcompute.VirtualMachine{}, nil
			}

			for _, vm := range page.Value {
				vmList = append(vmList, vm)
			}
		}

		for _, vm := range vmList {
			if *vm.ID == vmIID.SystemId {
				return *vm, nil
			}
		}
		notExistVpcErr := errors.New(fmt.Sprintf("The VM id %s not found", vmIID.SystemId))
		return armcompute.VirtualMachine{}, notExistVpcErr
	} else {
		resp, err := client.Get(ctx, resourceGroup, vmIID.NameId, &armcompute.VirtualMachinesClientGetOptions{
			Expand: (*armcompute.InstanceViewTypes)(toStrPtr(string(armcompute.InstanceViewTypesInstanceView))),
		})
		if err != nil {
			return armcompute.VirtualMachine{}, err
		}

		return resp.VirtualMachine, nil
	}
}

func generatePublicIPName(vmName string) string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%s-%s-PublicIP", vmName, strconv.FormatInt(rand.Int63n(100000), 10))
}

func generateVNicName(vmName string) string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%s-%s-VNic", vmName, strconv.FormatInt(rand.Int63n(100000), 10))
}

func resizeVMOsDisk(RootDiskSize string, vmReqIId irs.IID, resourceGroup string,
	client *armcompute.VirtualMachinesClient, diskClient *armcompute.DisksClient, ctx context.Context) (bool, error) {
	var desiredVmSize int32
	// Check desiredVmSize
	if RootDiskSize == "" || RootDiskSize == "default" {
		//
		return true, nil
	} else {
		size, err := strconv.Atoi(RootDiskSize)
		desiredVmSize = int32(size)
		if err != nil {
			return false, err
		}
	}

	// check curDisk
	startVM, err := GetRawVM(vmReqIId, resourceGroup, client, ctx)
	if err != nil {
		return false, err
	}

	var rootOSDisk armcompute.OSDisk
	if startVM.Properties.StorageProfile.OSDisk != nil {
		rootOSDisk = *startVM.Properties.StorageProfile.OSDisk
	}

	var curVmSize int32
	if rootOSDisk.DiskSizeGB != nil {
		curVmSize = *rootOSDisk.DiskSizeGB
	}
	// Check available expand
	if curVmSize > desiredVmSize {
		return false, errors.New(fmt.Sprintf("The vmSize can only be expanded."))
	} else if curVmSize == desiredVmSize {
		return true, nil
	}
	// curVmSize < desiredVmSize

	// Deallocate Vm to expand Size
	poller, err := client.BeginDeallocate(ctx, resourceGroup, vmReqIId.NameId, nil)
	if err != nil {
		return false, err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return false, err
	}
	// Get deallocated vm
	deallocateVm, err := GetRawVM(vmReqIId, resourceGroup, client, ctx)
	if err != nil {
		return false, err
	}
	// set disk updateOpt
	rootdiskname := ""

	if deallocateVm.Properties.StorageProfile.OSDisk.Name != nil {
		rootdiskname = *deallocateVm.Properties.StorageProfile.OSDisk.Name
	}
	upd := armcompute.DiskUpdate{
		Properties: &armcompute.DiskUpdateProperties{
			DiskSizeGB: &desiredVmSize,
		},
	}
	// Update disk
	poller2, err := diskClient.BeginUpdate(ctx, resourceGroup, rootdiskname, upd, nil)
	if err != nil {
		return false, err
	}
	_, err = poller2.PollUntilDone(ctx, nil)
	if err != nil {
		return false, err
	}
	// restart vm
	poller3, err := client.BeginStart(ctx, resourceGroup, vmReqIId.NameId, nil)
	if err != nil {
		return false, err
	}
	_, err = poller3.PollUntilDone(ctx, nil)
	if err != nil {
		return false, err
	}
	return true, nil
}

func ConvertVMIID(vmIID irs.IID, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) (irs.IID, error) {
	if vmIID.NameId == "" && vmIID.SystemId == "" {
		return vmIID, errors.New(fmt.Sprintf("nvalid IID"))
	}
	if vmIID.SystemId == "" {
		sysID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s", credentialInfo.SubscriptionId, regionInfo.Region, vmIID.NameId)
		return irs.IID{NameId: vmIID.NameId, SystemId: sysID}, nil
	} else {
		slist := strings.Split(vmIID.SystemId, "/")
		if len(slist) == 0 {
			return vmIID, errors.New(fmt.Sprintf("Invalid IID"))
		}
		s := slist[len(slist)-1]
		return irs.IID{NameId: s, SystemId: vmIID.SystemId}, nil
	}
}

func checkAuthInfoOSType(vmReqInfo irs.VMReqInfo, OSType irs.Platform) error {
	if OSType == irs.WINDOWS {
		_, idErr := windowUserIdCheck(vmReqInfo.VMUserId)
		if idErr != nil {
			return idErr
		}
		pwErr := cdcom.ValidateWindowsPassword(vmReqInfo.VMUserPasswd)
		if pwErr != nil {
			return pwErr
		}
		//if vmReqInfo.KeyPairIID.NameId != "" || vmReqInfo.KeyPairIID.SystemId != "" {
		//	return errors.New("for Windows, SSH key login method is not supported")
		//}
		computeErr := checkComputerNameWindow(vmReqInfo)
		if computeErr != nil {
			return computeErr
		}
	}
	if OSType == irs.LINUX_UNIX {
		if vmReqInfo.KeyPairIID.NameId == "" && vmReqInfo.KeyPairIID.SystemId == "" {
			return errors.New("for Linux, KeyPairIID is required")
		}
	}
	return nil
}

func checkComputerNameWindow(vmReqInfo irs.VMReqInfo) error {
	//if len(vmReqInfo.IId.NameId) > 15 {
	//	return errors.New("for Windows, VM's computeName cannot exceed 15 characters")
	//}
	// https://learn.microsoft.com/ko-KR/troubleshoot/windows-server/identity/naming-conventions-for-computer-domain-site-ou
	matchCase, _ := regexp.MatchString(`[\/?:|*<>\\\"]+`, vmReqInfo.IId.NameId)
	if matchCase {
		return errors.New("for Windows, VM's computeName contains unacceptable special characters")
	}
	return nil
}

func checkVMReqInfo(vmReqInfo irs.VMReqInfo) error {
	if vmReqInfo.IId.NameId == "" {
		return errors.New("invalid VM IID")
	}
	if vmReqInfo.ImageIID.NameId == "" && vmReqInfo.ImageIID.SystemId == "" {
		return errors.New("invalid VM ImageIID")
	}
	if vmReqInfo.VpcIID.NameId == "" && vmReqInfo.VpcIID.SystemId == "" {
		return errors.New("invalid VM VpcIID")
	}
	if vmReqInfo.SubnetIID.NameId == "" && vmReqInfo.SubnetIID.SystemId == "" {
		return errors.New("invalid VM SubnetIID")
	}
	//if vmReqInfo.KeyPairIID.NameId == "" && vmReqInfo.KeyPairIID.SystemId == "" && vmReqInfo.VMUserPasswd == "" {
	//	return errors.New("specify one login method, Password or Keypair")
	//}
	if vmReqInfo.VMSpecName == "" {
		return errors.New("invalid VM VMSpecName")
	}

	return nil
}

func createAdministratorUser(vmIID irs.IID, newusername string, newpassword string,
	virtualMachinesClient *armcompute.VirtualMachinesClient,
	virtualMachineRunCommandsClient *armcompute.VirtualMachineRunCommandsClient, ctx context.Context, region idrv.RegionInfo) error {
	rawVm, err := GetRawVM(vmIID, region.Region, virtualMachinesClient, ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("failed window User Add %s", err.Error()))
	}

	script := fmt.Sprintf("net user /add %s %s /Y; net localgroup %s %s /add;", newusername, newpassword, WindowBaseGroup, newusername)

	runOpt := armcompute.VirtualMachineRunCommand{
		Properties: &armcompute.VirtualMachineRunCommandProperties{
			Source: &armcompute.VirtualMachineRunCommandScriptSource{
				// Script: to.StringPtr(fmt.Sprintf("net user /add administrator qwe1212!Q; net localgroup administrators cb-user /add; net user /delete administrator;")),
				Script: &script,
			},
		},
		Location: toStrPtr(region.Region),
	}
	poller, err := virtualMachineRunCommandsClient.BeginCreateOrUpdate(ctx, region.Region, *rawVm.Name, "RunPowerShellScript", runOpt, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("failed window User Add %s", err.Error()))
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("failed window User Add %s", err.Error()))
	}
	return nil
}

func changeUserPassword(vmIID irs.IID, username string, newpassword string, virtualMachinesClient *armcompute.VirtualMachinesClient, virtualMachineRunCommandsClient *armcompute.VirtualMachineRunCommandsClient, ctx context.Context, region idrv.RegionInfo) error {
	rawVm, err := GetRawVM(vmIID, region.Region, virtualMachinesClient, ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("failed window User Add %s", err.Error()))
	}

	script := fmt.Sprintf("net user %s %s; net user %s %s;", username, newpassword, WindowBuitinUser, newpassword)

	runOpt := armcompute.VirtualMachineRunCommand{
		Properties: &armcompute.VirtualMachineRunCommandProperties{
			Source: &armcompute.VirtualMachineRunCommandScriptSource{
				Script: &script,
			},
		},
		Location: toStrPtr(region.Region),
	}
	pager, err := virtualMachineRunCommandsClient.BeginCreateOrUpdate(ctx, region.Region, *rawVm.Name, "RunPowerShellScript", runOpt, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("failed window User Add %s", err.Error()))
	}
	_, err = pager.PollUntilDone(ctx, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("failed window User Add %s", err.Error()))
	}
	return nil
}

func CheckVMReqInfoOSType(vmReqInfo irs.VMReqInfo, imageClient *armcompute.ImagesClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (irs.Platform, error) {
	if vmReqInfo.ImageType == "" || vmReqInfo.ImageType == irs.PublicImage {
		return getOSTypeByPublicImage(vmReqInfo.ImageIID)
	} else {
		return getOSTypeByMyImage(vmReqInfo.ImageIID, imageClient, credentialInfo, region, ctx)
	}
}

func getOSTypeByVM(server armcompute.VirtualMachine) irs.Platform {
	if server.Properties.OSProfile.LinuxConfiguration != nil {
		return irs.LINUX_UNIX
	}
	return irs.WINDOWS
}

func getOSTypeByPublicImage(imageIID irs.IID) (irs.Platform, error) {
	if imageIID.NameId == "" && imageIID.SystemId == "" {
		return "", errors.New("failed get OSType By ImageIID err = empty ImageIID")
	}
	imageName := imageIID.NameId
	if imageIID.NameId == "" {
		imageName = imageIID.SystemId
	}
	imageNameSplits := strings.Split(imageName, ":")
	if len(imageNameSplits) != 4 {
		return "", errors.New("failed get OSType By ImageIID err = invalid ImageIID, Image Name must be in the form of 'Publisher:Offer:Sku:Version'. ")
	}
	offer := imageNameSplits[1]
	if strings.Contains(strings.ToLower(offer), "window") {
		return irs.WINDOWS, nil
	}
	return irs.LINUX_UNIX, nil
}

func getOSTypeByMyImage(myImageIID irs.IID, imageClient *armcompute.ImagesClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (irs.Platform, error) {
	convertedMyImageIID, err := ConvertMyImageIID(myImageIID, credentialInfo, region)
	if err != nil {
		return "", errors.New(fmt.Sprintf("failed get OSType By MyImageIID err = %s", err.Error()))
	}
	myImage, err := imageClient.Get(ctx, region.Region, convertedMyImageIID.NameId, nil)
	if err != nil {
		return "", errors.New(fmt.Sprintf("failed get OSType By MyImageIID err = failed get MyImage err = %s", err.Error()))
	}
	if reflect.ValueOf(myImage.Properties.StorageProfile.OSDisk).IsNil() {
		return "", errors.New(fmt.Sprintf("failed get OSType By MyImageIID err = empty MyImage OSType"))
	}
	if *myImage.Properties.StorageProfile.OSDisk.OSType == armcompute.OperatingSystemTypesLinux {
		return irs.LINUX_UNIX, nil
	}
	if *myImage.Properties.StorageProfile.OSDisk.OSType == armcompute.OperatingSystemTypesWindows {
		return irs.WINDOWS, nil
	}
	return "", errors.New(fmt.Sprintf("failed get OSType By MyImageIID err = empty MyImage OSType"))
}
func windowUserIdCheck(userId string) (bool, error) {
	if userId == "Administrator" {
		return true, nil
	}
	return false, errors.New("for Windows, the userId only provides Administrator")
}

func (vmHandler *AzureVMHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, VM, "ListIID()")

	start := call.Start()

	var iidList []*irs.IID

	pager := vmHandler.Client.NewListPager(vmHandler.Region.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(vmHandler.Ctx)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List VM. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return make([]*irs.IID, 0), err
		}

		for _, vm := range page.Value {
			var iid irs.IID

			if vm.ID != nil {
				iid.SystemId = *vm.ID
			}
			if vm.Name != nil {
				iid.NameId = *vm.Name
			}

			iidList = append(iidList, &iid)
		}
	}

	LoggingInfo(hiscallInfo, start)

	return iidList, nil
}
