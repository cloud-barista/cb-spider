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
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"math/rand"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	cdcom "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzureOSTYPE string

const (
	ProvisioningStateCode string      = "ProvisioningState/succeeded"
	VM                                = "VM"
	PremiumSSD                        = "PremiumSSD"
	StandardSSD                       = "StandardSSD"
	StandardHDD                       = "StandardHDD"
	WindowBaseUser                    = "Administrator"
	WindowBaseGroup                   = "Administrators"
	WindowBuitinUser                  = CBVMUser
	UnknownOS             AzureOSTYPE = "UnknownOS"
	WindowOS              AzureOSTYPE = "WindowOS"
	LinuxOS               AzureOSTYPE = "LinuxOS"
)

type AzureVMHandler struct {
	CredentialInfo                  idrv.CredentialInfo
	Region                          idrv.RegionInfo
	Ctx                             context.Context
	Client                          *compute.VirtualMachinesClient
	SubnetClient                    *network.SubnetsClient
	NicClient                       *network.InterfacesClient
	PublicIPClient                  *network.PublicIPAddressesClient
	DiskClient                      *compute.DisksClient
	SshKeyClient                    *compute.SSHPublicKeysClient
	ImageClient                     *compute.ImagesClient
	VirtualMachineRunCommandsClient *compute.VirtualMachineRunCommandsClient
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
	vmExist, err := CheckExistVM(vmReqInfo.IId, vmHandler.Region.ResourceGroup, vmHandler.Client, vmHandler.Ctx)
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
		_, err = vmHandler.ImageClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, convertMyImageIId.NameId, "")
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
	}
	rawDataDiskList := make([]compute.Disk, len(vmReqInfo.DataDiskIIDs))
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
			disk, err := GetRawDisk(convertedDiskIId, vmHandler.Region.ResourceGroup, vmHandler.DiskClient, vmHandler.Ctx)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to Start VM. Failed to get DataDisk err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			err = CheckAttachStatus(disk)
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
	cleanResources.NetworkInterfaceName = vNicIId.NameId
	// 3. Set VmReqInfo
	// 3-1. Set VmReqInfo & Vnic
	vmOpts := compute.VirtualMachine{
		Location: &vmHandler.Region.Region,
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(vmReqInfo.VMSpecName),
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  to.StringPtr(vmReqInfo.IId.NameId),
				AdminUsername: to.StringPtr(CBVMUser),
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					{
						//ID: &vmReqInfo.NetworkInterfaceId,
						ID: &vNicIId.SystemId,
						NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
							Primary: to.BoolPtr(true),
						},
					},
				},
			},
		},
	}
	// 3-2. Set VmReqInfo - vmImage & storageType

	var managedDisk = new(compute.ManagedDiskParameters)
	if vmReqInfo.RootDiskType != "" && strings.ToLower(vmReqInfo.RootDiskType) != "default" {
		storageType := GetVMDiskTypeInitType(vmReqInfo.RootDiskType)
		managedDisk.StorageAccountType = storageType
	}
	// snapshotPoint Start

	//storageType := getVMDiskTypeInitType(vmReqInfo.RootDiskType)
	vmOpts.StorageProfile = &compute.StorageProfile{
		OsDisk: &compute.OSDisk{
			CreateOption: compute.DiskCreateOptionTypesFromImage,
			//ManagedDisk: &compute.ManagedDiskParameters{
			//	StorageAccountType: storageType,
			//},
			ManagedDisk:  managedDisk,
			DeleteOption: compute.DiskDeleteOptionTypesDelete,
		},
	}

	if vmReqInfo.ImageType == "" || vmReqInfo.ImageType == irs.PublicImage {
		//PublicImage
		imageArr := strings.Split(vmImage, ":")
		// URN 기반 퍼블릭 이미지 설정
		vmOpts.StorageProfile.ImageReference = &compute.ImageReference{
			Publisher: to.StringPtr(imageArr[0]),
			Offer:     to.StringPtr(imageArr[1]),
			Sku:       to.StringPtr(imageArr[2]),
			Version:   to.StringPtr(imageArr[3]),
		}
	} else {
		//MyImage
		convertMyImageIId, convertedErr := ConvertMyImageIID(vmReqInfo.ImageIID, vmHandler.CredentialInfo, vmHandler.Region)
		if convertedErr != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Finished to rollback deleting", err.Error()))
			cleanResource := CleanVMClientRequestResource{
				publicIPIId.NameId, vNicIId.NameId, "",
			}
			clean, deperr := vmHandler.cleanVMRelatedResource(VMCleanRelatedResource{
				RequiredSet:         cleanVMClientSet,
				CleanTargetResource: cleanResource,
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
		vmOpts.StorageProfile.ImageReference = &compute.ImageReference{
			ID: to.StringPtr(convertMyImageIId.SystemId),
		}
	}

	if imageOsType == LinuxOS {
		// 3-2. Set VmReqInfo - KeyPair & tagging
		if vmReqInfo.KeyPairIID.NameId != "" {
			key, keyErr := GetRawKey(vmReqInfo.KeyPairIID, vmHandler.Region.ResourceGroup, vmHandler.SshKeyClient, vmHandler.Ctx)
			if keyErr != nil {
				createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Finished to rollback deleting", err.Error()))
				cleanResource := CleanVMClientRequestResource{
					publicIPIId.NameId, vNicIId.NameId, "",
				}
				clean, deperr := vmHandler.cleanVMRelatedResource(VMCleanRelatedResource{
					RequiredSet:         cleanVMClientSet,
					CleanTargetResource: cleanResource,
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
			publicKey := *key.PublicKey
			vmOpts.OsProfile.LinuxConfiguration = &compute.LinuxConfiguration{
				SSH: &compute.SSHConfiguration{
					PublicKeys: &[]compute.SSHPublicKey{
						{
							Path:    to.StringPtr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", CBVMUser)),
							KeyData: to.StringPtr(publicKey),
						},
					},
				},
			}
			vmOpts.Tags = map[string]*string{
				"keypair":   to.StringPtr(vmReqInfo.KeyPairIID.NameId),
				"publicip":  to.StringPtr(publicIPIId.NameId),
				"createdBy": to.StringPtr(vmReqInfo.IId.NameId),
			}
		} else {
			vmOpts.OsProfile.AdminPassword = to.StringPtr(vmReqInfo.VMUserPasswd)
			vmOpts.Tags = map[string]*string{
				"publicip":  to.StringPtr(publicIPIId.NameId),
				"createdBy": to.StringPtr(vmReqInfo.IId.NameId),
			}
		}
	} else {
		if len(vmReqInfo.IId.NameId) > 15 {
			vmOpts.OsProfile.ComputerName = to.StringPtr(vmReqInfo.IId.NameId[:15])
		}
		vmOpts.OsProfile.AdminPassword = to.StringPtr(vmReqInfo.VMUserPasswd)
		vmOpts.OsProfile.AdminUsername = to.StringPtr(WindowBuitinUser)
		vmOpts.Tags = map[string]*string{
			"publicip":  to.StringPtr(publicIPIId.NameId),
			"createdBy": to.StringPtr(vmReqInfo.IId.NameId),
		}
	}

	// 4. CreateVM
	start := call.Start()
	future, err := vmHandler.Client.CreateOrUpdate(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmReqInfo.IId.NameId, vmOpts)
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

	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		// Exist VM? exist => vm delete, ResourceClean, not exist => ResourceClean
		createErr := errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Finished to rollback deleting", err.Error()))
		exist, err := CheckExistVM(vmReqInfo.IId, vmHandler.Region.ResourceGroup, vmHandler.Client, vmHandler.Ctx)
		if exist {
			cleanErr := vmHandler.cleanDeleteVm(vmReqInfo.IId)
			if cleanErr != nil {
				createErr = errors.New(fmt.Sprintf("Failed to Start VM. err = %s, and Failed to rollback err = %s", err.Error(), cleanErr.Error()))
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
	// 4-1. ResizeVMDisk
	_, err = resizeVMOsDisk(vmReqInfo.RootDiskSize, vmReqInfo.IId, vmHandler.Region.ResourceGroup, vmHandler.Client, vmHandler.DiskClient, vmHandler.Ctx)
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
		instanceView, _ := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmReqInfo.IId.NameId)
		// Get powerState, provisioningState
		vmStatus := getVmStatus(instanceView)
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
	if imageOsType == WindowOS {
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
		vm, err := vmHandler.Client.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmReqInfo.IId.NameId, compute.InstanceViewTypesInstanceView)
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
		vmInfo := vmHandler.mappingServerInfo(vm)
		if imageOsType == WindowOS {
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
		getErr := errors.New(fmt.Sprintf("Failed to Resume VM. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}

	// Check VM Exist
	exist, err := CheckExistVM(convertedIID, vmHandler.Region.ResourceGroup, vmHandler.Client, vmHandler.Ctx)

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
	instanceView, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.ResourceGroup, convertedIID.NameId)
	if err != nil {
		suspendErr := errors.New(fmt.Sprintf("Failed to Suspend VM. err = %s", err))
		cblogger.Error(suspendErr.Error())
		LoggingError(hiscallInfo, suspendErr)
		return irs.Failed, suspendErr
	}
	vmStatus := getVmStatus(instanceView)
	if vmStatus == irs.Running {
		start := call.Start()
		future, err := vmHandler.Client.PowerOff(vmHandler.Ctx, vmHandler.Region.ResourceGroup, convertedIID.NameId, to.BoolPtr(false))
		if err != nil {
			suspendErr := errors.New(fmt.Sprintf("Failed to Suspend VM. err = %s", err))
			cblogger.Error(suspendErr.Error())
			LoggingError(hiscallInfo, suspendErr)
			return irs.Failed, suspendErr
		}
		err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
		if err != nil {
			suspendErr := errors.New(fmt.Sprintf("Failed to Suspend VM. err = %s", err))
			cblogger.Error(suspendErr.Error())
			LoggingError(hiscallInfo, suspendErr)
			return irs.Failed, suspendErr
		}
		instanceView, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.ResourceGroup, convertedIID.NameId)
		if err != nil {
			suspendErr := errors.New(fmt.Sprintf("Failed to Suspend VM. but Failed Get Status err = %s", err))
			cblogger.Error(suspendErr.Error())
			LoggingError(hiscallInfo, suspendErr)
			return irs.Failed, suspendErr
		}
		vmStatus = getVmStatus(instanceView)
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
	exist, err := CheckExistVM(convertedIID, vmHandler.Region.ResourceGroup, vmHandler.Client, vmHandler.Ctx)

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
	instanceView, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.ResourceGroup, convertedIID.NameId)
	if err != nil {
		resumeErr := errors.New(fmt.Sprintf("Failed to Resume VM. err = %s", err))
		cblogger.Error(resumeErr.Error())
		LoggingError(hiscallInfo, resumeErr)
		return irs.Failed, resumeErr
	}
	vmStatus := getVmStatus(instanceView)
	if vmStatus == irs.Suspended {
		start := call.Start()
		future, err := vmHandler.Client.Start(vmHandler.Ctx, vmHandler.Region.ResourceGroup, convertedIID.NameId)
		if err != nil {
			resumeErr := errors.New(fmt.Sprintf("Failed to Resume VM. err = %s", err))
			cblogger.Error(resumeErr.Error())
			LoggingError(hiscallInfo, resumeErr)
			return irs.Failed, resumeErr
		}
		err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
		if err != nil {
			resumeErr := errors.New(fmt.Sprintf("Failed to Resume VM. err = %s", err))
			cblogger.Error(resumeErr.Error())
			LoggingError(hiscallInfo, resumeErr)
			return irs.Failed, resumeErr
		}
		instanceView, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.ResourceGroup, convertedIID.NameId)
		if err != nil {
			suspendErr := errors.New(fmt.Sprintf("Finish to Suspend VM. but Failed Get Status err = %s", err))
			cblogger.Error(suspendErr.Error())
			LoggingError(hiscallInfo, suspendErr)
			return irs.Failed, suspendErr
		}
		vmStatus = getVmStatus(instanceView)
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
	exist, err := CheckExistVM(convertedIID, vmHandler.Region.ResourceGroup, vmHandler.Client, vmHandler.Ctx)

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
	instanceView, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.ResourceGroup, convertedIID.NameId)
	if err != nil {
		rebootErr := errors.New(fmt.Sprintf("Failed to Reboot VM. err = %s", err))
		cblogger.Error(rebootErr.Error())
		LoggingError(hiscallInfo, rebootErr)
		return irs.Failed, rebootErr
	}
	vmStatus := getVmStatus(instanceView)
	if vmStatus == irs.Running {
		start := call.Start()
		future, err := vmHandler.Client.Restart(vmHandler.Ctx, vmHandler.Region.ResourceGroup, convertedIID.NameId)
		if err != nil {
			rebootErr := errors.New(fmt.Sprintf("Failed to Reboot VM. err = %s", err))
			cblogger.Error(rebootErr.Error())
			LoggingError(hiscallInfo, rebootErr)
			return irs.Failed, rebootErr
		}
		err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
		if err != nil {
			rebootErr := errors.New(fmt.Sprintf("Failed to Reboot VM. err = %s", err))
			cblogger.Error(rebootErr.Error())
			LoggingError(hiscallInfo, rebootErr)
			return irs.Failed, rebootErr
		}
		instanceView, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.ResourceGroup, convertedIID.NameId)
		if err != nil {
			suspendErr := errors.New(fmt.Sprintf("Failed to Suspend VM. but Failed Get Status err = %s", err))
			cblogger.Error(suspendErr.Error())
			LoggingError(hiscallInfo, suspendErr)
			return irs.Failed, suspendErr
		}
		vmStatus = getVmStatus(instanceView)
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
	LoggingInfo(hiscallInfo, start)

	err := vmHandler.cleanDeleteVm(vmIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Terminate VM. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	return irs.NotExist, nil
}

func (vmHandler *AzureVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, VM, "ListVMStatus()")

	start := call.Start()
	serverList, err := vmHandler.Client.List(vmHandler.Ctx, vmHandler.Region.ResourceGroup)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return []*irs.VMStatusInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	var vmStatusList []*irs.VMStatusInfo
	for _, s := range serverList.Values() {
		if s.InstanceView != nil {
			statusStr := getVmStatus(*s.InstanceView)
			status := statusStr
			vmStatusInfo := irs.VMStatusInfo{
				IId: irs.IID{
					NameId:   *s.Name,
					SystemId: *s.ID,
				},
				VmStatus: status,
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		} else {
			vmIdArr := strings.Split(*s.ID, "/")
			vmName := vmIdArr[8]
			status, _ := vmHandler.GetVMStatus(irs.IID{NameId: vmName, SystemId: *s.ID})
			vmStatusInfo := irs.VMStatusInfo{
				IId: irs.IID{
					NameId:   *s.Name,
					SystemId: *s.ID,
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
	instanceView, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.ResourceGroup, convertedIID.NameId)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	LoggingInfo(hiscallInfo, start)

	// Get powerState, provisioningState
	vmStatus := getVmStatus(instanceView)
	return vmStatus, nil
}

func (vmHandler *AzureVMHandler) ListVM() ([]*irs.VMInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, VM, "ListVM()")

	start := call.Start()
	serverList, err := vmHandler.Client.List(vmHandler.Ctx, vmHandler.Region.ResourceGroup)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMList. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return []*irs.VMInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)

	var vmList []*irs.VMInfo
	for _, server := range serverList.Values() {
		vmInfo := vmHandler.mappingServerInfo(server)
		vmList = append(vmList, &vmInfo)
	}

	return vmList, nil
}

func (vmHandler *AzureVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "GetVM()")

	convertedIID, err := ConvertVMIID(vmIID, vmHandler.CredentialInfo, vmHandler.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VM. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VMInfo{}, getErr
	}

	start := call.Start()

	vm, err := GetRawVM(convertedIID, vmHandler.Region.ResourceGroup, vmHandler.Client, vmHandler.Ctx)
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

func getVmStatus(instanceView compute.VirtualMachineInstanceView) irs.VMStatus {
	var powerState, provisioningState string

	for _, stat := range *instanceView.Statuses {
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
	exist, err := CheckExistVM(convertedIID, vmHandler.Region.ResourceGroup, vmHandler.Client, vmHandler.Ctx)
	if err != nil {
		return err
	}
	if exist {
		vm, err := GetRawVM(convertedIID, vmHandler.Region.ResourceGroup, vmHandler.Client, vmHandler.Ctx)
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
		if vm.VirtualMachineProperties.StorageProfile.OsDisk.Name != nil {
			cleanResources.VmDiskName = *vm.VirtualMachineProperties.StorageProfile.OsDisk.Name
		}
		vNic, vNicErr := vmHandler.NicClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmInfo.NetworkInterface, "")
		if vNicErr != nil {
			return vNicErr
		}
		for _, ip := range *vNic.IPConfigurations {
			if *ip.Primary {
				if ip.PublicIPAddress != nil {
					publicipIdAddr := strings.Split(*ip.PublicIPAddress.ID, "/")
					cleanResources.PublicIPName = publicipIdAddr[len(publicipIdAddr)-1]
				}
			}
		}
		vmDelete, vmDeleteErr := vmHandler.Client.Delete(vmHandler.Ctx, vmHandler.Region.ResourceGroup, *vm.Name, to.BoolPtr(true))
		if vmDeleteErr != nil {
			return vmDeleteErr
		}
		vmDeleteWaitErr := vmDelete.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
		if vmDeleteWaitErr != nil {
			return vmDeleteWaitErr
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

func (vmHandler *AzureVMHandler) mappingServerInfo(server compute.VirtualMachine) irs.VMInfo {

	// Get Default VM Info
	vmInfo := irs.VMInfo{
		IId: irs.IID{
			NameId:   *server.Name,
			SystemId: *server.ID,
		},
		Region: irs.RegionInfo{
			Region: *server.Location,
		},
		VMSpecName:     string(server.VirtualMachineProperties.HardwareProfile.VMSize),
		RootDeviceName: "Not visible in Azure",
		VMBlockDisk:    "Not visible in Azure",
	}

	// Set VM Zone
	if server.Zones != nil {
		vmInfo.Region.Zone = (*server.Zones)[0]
	}

	// Set VM Image Info
	if reflect.ValueOf(server.StorageProfile.ImageReference.ID).IsNil() {
		imageRef := server.VirtualMachineProperties.StorageProfile.ImageReference
		vmInfo.ImageIId.SystemId = *imageRef.Publisher + ":" + *imageRef.Offer + ":" + *imageRef.Sku + ":" + *imageRef.Version
		vmInfo.ImageIId.NameId = *imageRef.Publisher + ":" + *imageRef.Offer + ":" + *imageRef.Sku + ":" + *imageRef.Version
		//vmInfo.ImageIId.SystemId = vmInfo.ImageIId.NameId
	} else {
		vmInfo.ImageIId.SystemId = *server.VirtualMachineProperties.StorageProfile.ImageReference.ID
		vmInfo.ImageIId.NameId = *server.VirtualMachineProperties.StorageProfile.ImageReference.ID
		//vmInfo.ImageIId.SystemId = vmInfo.ImageIId.NameId
	}

	// Get VNic ID
	niList := *server.NetworkProfile.NetworkInterfaces
	var VNicId string
	for _, ni := range niList {
		if ni.ID != nil {
			VNicId = *ni.ID
		}
	}

	// Get VNic
	nicIdArr := strings.Split(VNicId, "/")
	nicName := nicIdArr[len(nicIdArr)-1]
	vNic, _ := vmHandler.NicClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, nicName, "")
	vmInfo.NetworkInterface = nicName

	// Get SecurityGroup
	sgGroupIdArr := strings.Split(*vNic.NetworkSecurityGroup.ID, "/")
	sgGroupName := sgGroupIdArr[len(sgGroupIdArr)-1]
	vmInfo.SecurityGroupIIds = []irs.IID{
		{
			NameId:   sgGroupName,
			SystemId: *vNic.NetworkSecurityGroup.ID,
		},
	}

	// Get PrivateIP, PublicIpId
	for _, ip := range *vNic.IPConfigurations {
		if *ip.Primary {
			// PrivateIP 정보 설정
			vmInfo.PrivateIP = *ip.PrivateIPAddress

			// PublicIP 정보 조회 및 설정
			if ip.PublicIPAddress != nil {
				publicIPId := *ip.PublicIPAddress.ID
				publicIPIdArr := strings.Split(publicIPId, "/")
				publicIPName := publicIPIdArr[len(publicIPIdArr)-1]

				publicIP, _ := vmHandler.PublicIPClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, publicIPName, "")
				if publicIP.IPAddress != nil {
					vmInfo.PublicIP = *publicIP.IPAddress
				}
			}

			// Get Subnet
			subnetIdArr := strings.Split(*ip.InterfaceIPConfigurationPropertiesFormat.Subnet.ID, "/")
			subnetName := subnetIdArr[len(subnetIdArr)-1]
			vmInfo.SubnetIID = irs.IID{NameId: subnetName, SystemId: *ip.InterfaceIPConfigurationPropertiesFormat.Subnet.ID}

			// Get VPC
			vpcIdArr := subnetIdArr[:len(subnetIdArr)-2]
			vpcName := vpcIdArr[len(vpcIdArr)-1]
			vmInfo.VpcIID = irs.IID{NameId: vpcName, SystemId: strings.Join(vpcIdArr, "/")}
		}
	}
	osType, err := getOSTypeByVM(server)
	if err == nil {
		if osType == WindowOS {
			vmInfo.VMUserId = WindowBaseUser
		}
		if osType == LinuxOS {
			vmInfo.VMUserId = CBVMUser
		}
	}
	// Set GuestUser Id/Pwd
	//if server.VirtualMachineProperties.OsProfile.AdminUsername != nil {
	//	vmInfo.VMUserId = *server.VirtualMachineProperties.OsProfile.AdminUsername
	//}
	if server.VirtualMachineProperties.OsProfile.AdminPassword != nil {
		vmInfo.VMUserPasswd = *server.VirtualMachineProperties.OsProfile.AdminPassword
	}

	// Set BootDisk
	if server.VirtualMachineProperties.StorageProfile.OsDisk.Name != nil {
		vmInfo.VMBootDisk = *server.VirtualMachineProperties.StorageProfile.OsDisk.Name
	}
	if server.VirtualMachineProperties.StorageProfile.OsDisk.DiskSizeGB != nil {
		vmInfo.RootDiskSize = strconv.Itoa(int(*server.VirtualMachineProperties.StorageProfile.OsDisk.DiskSizeGB))
	}
	if server.VirtualMachineProperties.StorageProfile.OsDisk.ManagedDisk != nil {
		vmInfo.RootDiskType = GetVMDiskInfoType(server.VirtualMachineProperties.StorageProfile.OsDisk.ManagedDisk.StorageAccountType)
	}

	// Get StartTime
	if server.VirtualMachineProperties.InstanceView != nil {
		for _, status := range *server.VirtualMachineProperties.InstanceView.Statuses {
			if strings.EqualFold(*status.Code, ProvisioningStateCode) {
				vmInfo.StartTime = status.Time.Local()
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

	if server.StorageProfile != nil && server.StorageProfile.DataDisks != nil && len(*server.StorageProfile.DataDisks) > 0 {
		dataDisks := *server.StorageProfile.DataDisks
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

	return vmInfo
}

// VM 생성 시 Public IP 자동 생성 (nested flow 적용)
func CreatePublicIP(vmHandler *AzureVMHandler, vmReqInfo irs.VMReqInfo) (irs.IID, error) {

	// PublicIP 이름 생성
	publicIPName := generatePublicIPName(vmReqInfo.IId.NameId)

	createOpts := network.PublicIPAddress{
		Name: to.StringPtr(publicIPName),
		Sku: &network.PublicIPAddressSku{
			Name: network.PublicIPAddressSkuName("Basic"),
		},
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAddressVersion: network.IPVersion("IPv4"),
			//PublicIPAllocationMethod: network.IPAllocationMethod("Static"),
			PublicIPAllocationMethod: network.IPAllocationMethod("Dynamic"),
			IdleTimeoutInMinutes:     to.Int32Ptr(4),
		},
		Location: &vmHandler.Region.Region,
		Tags: map[string]*string{
			"createdBy": to.StringPtr(vmReqInfo.IId.NameId),
		},
	}

	future, err := vmHandler.PublicIPClient.CreateOrUpdate(vmHandler.Ctx, vmHandler.Region.ResourceGroup, publicIPName, createOpts)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create PublicIP, error=%s", err))
		return irs.IID{}, createErr
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.PublicIPClient.Client)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create PublicIP, error=%s", err))
		return irs.IID{}, createErr
	}

	// 생성된 PublicIP 정보 리턴
	publicIPInfo, err := vmHandler.PublicIPClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, publicIPName, "")
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
	networkInterfaceName := cleanRelatedResource.CleanTargetResource.NetworkInterfaceName
	publicIPId := cleanRelatedResource.CleanTargetResource.PublicIPName
	vmDiskId := cleanRelatedResource.CleanTargetResource.VmDiskName
	resourceGroup := vmHandler.Region.ResourceGroup
	// VNic Delete
	if networkInterfaceName != "" {
		vnicExist, _ := CheckExistVNic(networkInterfaceName, resourceGroup, vmHandler.NicClient, vmHandler.Ctx)
		subnet, subnetgetErr := vmHandler.SubnetClient.Get(vmHandler.Ctx, resourceGroup, cleanRelatedResource.RequiredSet.VPCName, cleanRelatedResource.RequiredSet.SubnetName, "")
		if subnetgetErr != nil {
			return false, subnetgetErr
		}
		var ipConfigArr []network.InterfaceIPConfiguration
		ipConfig := network.InterfaceIPConfiguration{
			Name: to.StringPtr("ipConfig1"),
			InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
				Subnet:                    &subnet,
				PrivateIPAllocationMethod: "Dynamic",
				PublicIPAddress:           nil,
			},
		}
		ipConfigArr = append(ipConfigArr, ipConfig)

		detachOpts := network.Interface{
			InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
				IPConfigurations: &ipConfigArr,
			},
			Location: &vmHandler.Region.Region,
		}
		if vnicExist {
			nicDetachFuture, err := vmHandler.NicClient.CreateOrUpdate(vmHandler.Ctx, resourceGroup, networkInterfaceName, detachOpts)
			if err != nil {
				return false, err
			}
			err = nicDetachFuture.WaitForCompletionRef(vmHandler.Ctx, vmHandler.NicClient.Client)
			if err != nil {
				return false, err
			}
			nicDeleteFuture, err := vmHandler.NicClient.Delete(vmHandler.Ctx, resourceGroup, networkInterfaceName)
			if err != nil {
				cblogger.Error(err)
				return false, err
			}
			err = nicDeleteFuture.WaitForCompletionRef(vmHandler.Ctx, vmHandler.NicClient.Client)
			if err != nil {
				cblogger.Error(err)
				return false, err
			}
		}
	}
	if publicIPId != "" {
		publicIPExist, err := CheckExistPublicIp(publicIPId, resourceGroup, vmHandler.PublicIPClient, vmHandler.Ctx)
		if err != nil {
			return false, err
		}
		if publicIPExist {
			publicIPFuture, delErr := vmHandler.PublicIPClient.Delete(vmHandler.Ctx, resourceGroup, publicIPId)
			if delErr != nil {
				return false, delErr
			}
			delWaitErr := publicIPFuture.WaitForCompletionRef(vmHandler.Ctx, vmHandler.PublicIPClient.Client)
			if delWaitErr != nil {
				return false, delWaitErr
			}
		}
	}
	if vmDiskId != "" {
		vmDiskExist, err := CheckExistVMDisk(vmDiskId, vmHandler.DiskClient, vmHandler.Ctx)
		if err != nil {
			return false, err
		}
		if vmDiskExist {
			diskFuture, delErr := vmHandler.DiskClient.Delete(vmHandler.Ctx, resourceGroup, vmDiskId)
			if delErr != nil {
				return false, delErr
			}
			delWaitErr := diskFuture.WaitForCompletionRef(vmHandler.Ctx, vmHandler.DiskClient.Client)
			if delWaitErr != nil {
				return false, delWaitErr
			}
		}
	}
	return true, nil
}

func CheckExistPublicIp(publicIPId string, resourceGroup string, client *network.PublicIPAddressesClient, ctx context.Context) (bool, error) {
	publicIpList, err := client.List(ctx, resourceGroup)
	if err != nil {
		return false, err
	}
	for _, publicIp := range publicIpList.Values() {
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
	subnet, err := vmHandler.SubnetClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmReqInfo.VpcIID.NameId, vmReqInfo.SubnetIID.NameId, "")

	var ipConfigArr []network.InterfaceIPConfiguration
	ipConfig := network.InterfaceIPConfiguration{
		Name: to.StringPtr("ipConfig1"),
		InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
			Subnet:                    &subnet,
			PrivateIPAllocationMethod: "Dynamic",
			PublicIPAddress: &network.PublicIPAddress{
				ID: to.StringPtr(publicIPIId.SystemId),
			},
		},
	}
	ipConfigArr = append(ipConfigArr, ipConfig)

	createOpts := network.Interface{
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &ipConfigArr,
			NetworkSecurityGroup: &network.SecurityGroup{
				ID: to.StringPtr(secGroupId),
			},
		},
		Location: &vmHandler.Region.Region,
		Tags: map[string]*string{
			"createdBy": to.StringPtr(vmReqInfo.IId.NameId),
		},
	}
	future, err := vmHandler.NicClient.CreateOrUpdate(vmHandler.Ctx, vmHandler.Region.ResourceGroup, VNicName, createOpts)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create NetworkInterface, error=%s", err))
		return irs.IID{}, createErr
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.NicClient.Client)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create NetworkInterface, error=%s", err))
		return irs.IID{}, createErr
	}

	// 생성된 VNic 정보 리턴
	VNic, err := vmHandler.NicClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, VNicName, "")
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create NetworkInterface, error=%s", err))
		return irs.IID{}, createErr
	}
	VNicIId := irs.IID{NameId: *VNic.Name, SystemId: *VNic.ID}
	return VNicIId, nil
}

func CheckExistVNic(networkInterfaceName string, resourceGroup string, client *network.InterfacesClient, ctx context.Context) (bool, error) {
	networkInterfaceList, err := client.List(ctx, resourceGroup)
	if err != nil {
		return false, err
	}
	for _, networkInterface := range networkInterfaceList.Values() {
		if *networkInterface.Name == networkInterfaceName {
			return true, nil
		}
	}
	return false, nil
}

func CheckExistVMDisk(osDiskName string, client *compute.DisksClient, ctx context.Context) (bool, error) {
	vmDiskList, err := client.List(ctx)
	if err != nil {
		return false, err
	}
	for _, vmDisk := range vmDiskList.Values() {
		if *vmDisk.Name == osDiskName {
			return true, nil
		}
	}
	return false, nil
}

func CheckExistVM(vmIID irs.IID, resourceGroup string, client *compute.VirtualMachinesClient, ctx context.Context) (bool, error) {
	serverList, err := client.List(ctx, resourceGroup)
	if err != nil {
		return false, err
	}
	for _, server := range serverList.Values() {
		if vmIID.SystemId != "" && vmIID.SystemId == *server.ID {
			return true, nil
		}
		if vmIID.NameId != "" && vmIID.NameId == *server.Name {
			return true, nil
		}
	}
	return false, nil
}

func GetRawVM(vmIID irs.IID, resourceGroup string, client *compute.VirtualMachinesClient, ctx context.Context) (compute.VirtualMachine, error) {
	if vmIID.NameId == "" {
		serverList, err := client.List(ctx, resourceGroup)
		if err != nil {
			return compute.VirtualMachine{}, err
		}
		for _, server := range serverList.Values() {
			if *server.ID == vmIID.SystemId {
				return server, nil
			}
		}
		notExistVpcErr := errors.New(fmt.Sprintf("The VM id %s not found", vmIID.SystemId))
		return compute.VirtualMachine{}, notExistVpcErr
	} else {
		return client.Get(ctx, resourceGroup, vmIID.NameId, compute.InstanceViewTypesInstanceView)
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

func resizeVMOsDisk(RootDiskSize string, vmReqIId irs.IID, resourceGroup string, client *compute.VirtualMachinesClient, diskClient *compute.DisksClient, ctx context.Context) (bool, error) {
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

	var rootOSDisk compute.OSDisk
	if startVM.VirtualMachineProperties.StorageProfile.OsDisk != nil {
		rootOSDisk = *startVM.VirtualMachineProperties.StorageProfile.OsDisk
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
	deallocate, err := client.Deallocate(ctx, resourceGroup, vmReqIId.NameId)
	if err != nil {
		return false, err
	}
	err = deallocate.WaitForCompletionRef(ctx, client.Client)
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

	if deallocateVm.VirtualMachineProperties.StorageProfile.OsDisk.Name != nil {
		rootdiskname = *deallocateVm.VirtualMachineProperties.StorageProfile.OsDisk.Name
	}
	upd := compute.DiskUpdate{
		DiskUpdateProperties: &compute.DiskUpdateProperties{
			DiskSizeGB: to.Int32Ptr(desiredVmSize),
		},
	}
	// Update disk
	vmdiskUpdatefuture, err := diskClient.Update(ctx, resourceGroup, rootdiskname, upd)
	if err != nil {
		return false, err
	}
	err = vmdiskUpdatefuture.WaitForCompletionRef(ctx, diskClient.Client)
	// restart vm
	restart, err := client.Start(ctx, resourceGroup, vmReqIId.NameId)
	if err != nil {
		return false, err
	}
	err = restart.WaitForCompletionRef(ctx, client.Client)
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
		sysID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s", credentialInfo.SubscriptionId, regionInfo.ResourceGroup, vmIID.NameId)
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

func checkAuthInfoOSType(vmReqInfo irs.VMReqInfo, OSType AzureOSTYPE) error {
	if OSType == WindowOS {
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
	if OSType == LinuxOS {
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

func createAdministratorUser(vmIID irs.IID, newusername string, newpassword string, virtualMachinesClient *compute.VirtualMachinesClient, virtualMachineRunCommandsClient *compute.VirtualMachineRunCommandsClient, ctx context.Context, region idrv.RegionInfo) error {
	rawVm, err := GetRawVM(vmIID, region.ResourceGroup, virtualMachinesClient, ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("failed window User Add %s", err.Error()))
	}
	runOpt := compute.VirtualMachineRunCommand{
		VirtualMachineRunCommandProperties: &compute.VirtualMachineRunCommandProperties{
			Source: &compute.VirtualMachineRunCommandScriptSource{
				// Script: to.StringPtr(fmt.Sprintf("net user /add administrator qwe1212!Q; net localgroup administrators cb-user /add; net user /delete administrator;")),
				Script: to.StringPtr(fmt.Sprintf("net user /add %s %s; net localgroup %s %s /add", newusername, newpassword, WindowBaseGroup, newusername)),
			},
		},
		Location: to.StringPtr(region.Region),
	}
	runCommandResult, err := virtualMachineRunCommandsClient.CreateOrUpdate(ctx, region.ResourceGroup, *rawVm.Name, "RunPowerShellScript", runOpt)
	if err != nil {
		return errors.New(fmt.Sprintf("failed window User Add %s", err.Error()))
	}
	err = runCommandResult.WaitForCompletionRef(ctx, virtualMachineRunCommandsClient.Client)
	if err != nil {
		return errors.New(fmt.Sprintf("failed window User Add %s", err.Error()))
	}
	return nil
}

func changeUserPassword(vmIID irs.IID, username string, newpassword string, virtualMachinesClient *compute.VirtualMachinesClient, virtualMachineRunCommandsClient *compute.VirtualMachineRunCommandsClient, ctx context.Context, region idrv.RegionInfo) error {
	rawVm, err := GetRawVM(vmIID, region.ResourceGroup, virtualMachinesClient, ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("failed window User Add %s", err.Error()))
	}
	runOpt := compute.VirtualMachineRunCommand{
		VirtualMachineRunCommandProperties: &compute.VirtualMachineRunCommandProperties{
			Source: &compute.VirtualMachineRunCommandScriptSource{
				Script: to.StringPtr(fmt.Sprintf("net user %s %s; net user %s %s;", username, newpassword, WindowBuitinUser, newpassword)),
			},
		},
		Location: to.StringPtr(region.Region),
	}
	runCommandResult, err := virtualMachineRunCommandsClient.CreateOrUpdate(ctx, region.ResourceGroup, *rawVm.Name, "RunPowerShellScript", runOpt)
	if err != nil {
		return errors.New(fmt.Sprintf("failed window User Add %s", err.Error()))
	}
	err = runCommandResult.WaitForCompletionRef(ctx, virtualMachineRunCommandsClient.Client)
	if err != nil {
		return errors.New(fmt.Sprintf("failed window User Add %s", err.Error()))
	}
	return nil
}

func CheckVMReqInfoOSType(vmReqInfo irs.VMReqInfo, imageClient *compute.ImagesClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (AzureOSTYPE, error) {
	if vmReqInfo.ImageType == "" || vmReqInfo.ImageType == irs.PublicImage {
		return getOSTypeByPublicImage(vmReqInfo.ImageIID)
	} else {
		return getOSTypeByMyImage(vmReqInfo.ImageIID, imageClient, credentialInfo, region, ctx)
	}
}

func getOSTypeByVM(server compute.VirtualMachine) (AzureOSTYPE, error) {
	if server.OsProfile.LinuxConfiguration != nil {
		return LinuxOS, nil
	}
	return WindowOS, nil
}

func getOSTypeByPublicImage(imageIID irs.IID) (AzureOSTYPE, error) {
	if imageIID.NameId == "" && imageIID.SystemId == "" {
		return UnknownOS, errors.New("failed get OSType By ImageIID err = empty ImageIID")
	}
	imageName := imageIID.NameId
	if imageIID.NameId == "" {
		imageName = imageIID.SystemId
	}
	imageNameSplits := strings.Split(imageName, ":")
	if len(imageNameSplits) != 4 {
		return UnknownOS, errors.New("failed get OSType By ImageIID err = invalid ImageIID, Image Name must be in the form of 'Publisher:Offer:Sku:Version'. ")
	}
	offer := imageNameSplits[1]
	if strings.Contains(strings.ToLower(offer), "window") {
		return WindowOS, nil
	}
	return LinuxOS, nil
}

func getOSTypeByMyImage(myImageIID irs.IID, imageClient *compute.ImagesClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (AzureOSTYPE, error) {
	convertedMyImageIID, err := ConvertMyImageIID(myImageIID, credentialInfo, region)
	if err != nil {
		return UnknownOS, errors.New(fmt.Sprintf("failed get OSType By MyImageIID err = %s", err.Error()))
	}
	myImage, err := imageClient.Get(ctx, region.ResourceGroup, convertedMyImageIID.NameId, "")
	if err != nil {
		return UnknownOS, errors.New(fmt.Sprintf("failed get OSType By MyImageIID err = failed get MyImage err = %s", err.Error()))
	}
	if reflect.ValueOf(myImage.StorageProfile.OsDisk).IsNil() {
		return UnknownOS, errors.New(fmt.Sprintf("failed get OSType By MyImageIID err = empty MyImage OSType"))
	}
	if myImage.StorageProfile.OsDisk.OsType == compute.OperatingSystemTypesLinux {
		return LinuxOS, nil
	}
	if myImage.StorageProfile.OsDisk.OsType == compute.OperatingSystemTypesWindows {
		return WindowOS, nil
	}
	return UnknownOS, errors.New(fmt.Sprintf("failed get OSType By MyImageIID err = empty MyImage OSType"))
}
func windowUserIdCheck(userId string) (bool, error) {
	if userId == "Administrator" {
		return true, nil
	}
	return false, errors.New("for Windows, the userId only provides Administrator")
}
