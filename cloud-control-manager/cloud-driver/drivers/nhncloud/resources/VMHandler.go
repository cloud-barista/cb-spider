// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2021.12.
// Updated by ETRI, 2024.01.
// Updated by ETRI, 2024.04.

package resources

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/extensions/bootfromvolume"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // To prevent 'unknown time zone Asia/Seoul' error
	// "github.com/davecgh/go-spew/spew"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/blockstorage/v2/volumes"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/extensions/floatingips"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/extensions/keypairs"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/extensions/startstop"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/flavors"
	comimages "github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/images" // compute/v2/images
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/servers"
	//	images "github.com/cloud-barista/nhncloud-sdk-go/openstack/imageservice/v2/images" // imageservice/v2/images : For Visibility parameter

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	DefaultVMUserName       string = "cb-user"
	DefaultWindowsUserName  string = "cb-user"
	UbuntuCloudInitFilePath string = "/cloud-driver-libs/.cloud-init-nhncloud/cloud-init-ubuntu"
	WinCloudInitFilePath    string = "/cloud-driver-libs/.cloud-init-nhncloud/cloud-init-windows"
	DefaultDiskSize         string = "20"
	DefaultWinRootDiskSize  string = "50"
)

type NhnCloudVMHandler struct {
	RegionInfo    idrv.RegionInfo
	VMClient      *nhnsdk.ServiceClient
	ImageClient   *nhnsdk.ServiceClient
	NetworkClient *nhnsdk.ServiceClient
	VolumeClient  *nhnsdk.ServiceClient
}

func (vmHandler *NhnCloudVMHandler) getRawVM(vmIId irs.IID) (servers.Server, error) {
	if vmIId.SystemId == "" && vmIId.NameId == "" {
		return servers.Server{}, errors.New("invalid IID")
	}
	if vmIId.SystemId == "" {
		pager, err := servers.List(vmHandler.VMClient, nil).AllPages()
		if err != nil {
			return servers.Server{}, err
		}
		rawServers, err := servers.ExtractServers(pager)
		if err != nil {
			return servers.Server{}, err
		}
		for _, vm := range rawServers {
			if vm.Name == vmIId.NameId {
				return vm, nil
			}
		}
		return servers.Server{}, errors.New("VM not found")
	} else {
		vm, err := servers.Get(vmHandler.VMClient, vmIId.SystemId).Extract()
		if err != nil {
			return servers.Server{}, err
		}

		return *vm, nil
	}
}

func (vmHandler *NhnCloudVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	cblogger.Info("NHN Cloud Driver: called StartVM()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Region, call.VM, vmReqInfo.IId.NameId, "StartVM()")

	if strings.EqualFold(vmReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid VM NameId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	// # Check whether the Routing Table (of the VPC) is connected to an Internet Gateway
	vpcHandler := NhnCloudVPCHandler{
		NetworkClient: vmHandler.NetworkClient,
	}
	vpc, err := vpcHandler.getRawVPC(vmReqInfo.VpcIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to get VPC info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	isConnectedToGateway, err := vpcHandler.isConnectedToGateway(vpc.ID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Check whether the VPC connected to an Internet Gateway : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	if !isConnectedToGateway {
		newErr := fmt.Errorf("Routing Table of the VPC need to be connected to an Internet Gateway to use Public IP!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	// Check if VM Name is Duplicated
	listOpts := servers.ListOpts{Name: vmReqInfo.IId.NameId}
	allPages, err := servers.List(vmHandler.VMClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VM with the name : %s", vmReqInfo.IId.NameId)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	vmList, err := servers.ExtractServers(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VM Info with the name : %s", vmReqInfo.IId.NameId)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	if len(vmList) != 0 {
		newErr := fmt.Errorf("The VM Name [%s] already exists!!", vmReqInfo.IId.NameId)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	// Get VM SpecId with the name
	vmSpecId, err := getVMSpecIdWithName(vmHandler.VMClient, vmReqInfo.VMSpecName)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VMSpec ID with the name : %v", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	cblogger.Infof("# vmSpecId : [%s]", vmSpecId)

	// Get SecurityGroupId list
	var sgIdList []string
	for _, sgIID := range vmReqInfo.SecurityGroupIIDs {
		if sgIID.SystemId == "" {
			sgHandler := NhnCloudSecurityHandler{
				VMClient: vmHandler.VMClient,
			}
			sg, err := sgHandler.getRawSecurity(sgIID)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get Security Group with the name : %v", err)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
			sgIID.SystemId = sg.ID
		}
		sgIdList = append(sgIdList, sgIID.SystemId)
	}

	// # Preparing for UserData String for Linux and Windows Platform
	var initUserData *string
	var keyPairId string
	if !strings.EqualFold(vmReqInfo.KeyPairIID.SystemId, "") {
		keyPairId = vmReqInfo.KeyPairIID.SystemId
	} else {
		keyPairId = vmReqInfo.KeyPairIID.NameId
	}
	if vmReqInfo.ImageType == irs.PublicImage || vmReqInfo.ImageType == "" || vmReqInfo.ImageType == "default" {
		// isPublicImage() in ImageHandler
		imageHandler := NhnCloudImageHandler{
			RegionInfo:  vmHandler.RegionInfo,
			VMClient:    vmHandler.VMClient,
			ImageClient: vmHandler.ImageClient,
		}
		isPublicImage, err := imageHandler.isPublicImage(vmReqInfo.ImageIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Check Whether the Image is Public Image : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
		if !isPublicImage {
			newErr := fmt.Errorf("'PublicImage' type is selected, but Specified image is Not a PublicImage in the region!!")
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}

		// CheckWindowsImage() in ImageHandler
		isPublicWindowsImage, err := imageHandler.CheckWindowsImage(vmReqInfo.ImageIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Check Whether the Image is MS Windows Image : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
		if isPublicWindowsImage {
			var createErr error
			initUserData, createErr = vmHandler.createWinInitUserData(vmReqInfo.VMUserPasswd)
			if createErr != nil {
				newErr := fmt.Errorf("Failed to Create Cloud-Init Script with the Password : [%v]", createErr)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
		} else {
			var createErr error
			initUserData, createErr = vmHandler.createLinuxInitUserData(vmReqInfo.ImageIID, keyPairId)
			if createErr != nil {
				newErr := fmt.Errorf("Failed to Create Cloud-Init Script with the KeyPairId : [%v]", createErr)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
		}
	} else { // In case of MyImage
		// isPublicImage() in 'MyImage'Handler
		myImageHandler := NhnCloudMyImageHandler{
			RegionInfo:  vmHandler.RegionInfo,
			VMClient:    vmHandler.VMClient,
			ImageClient: vmHandler.ImageClient,
		}
		isPublicImage, err := myImageHandler.isPublicImage(vmReqInfo.ImageIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Check Whether the Image is Public Image : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
		if isPublicImage {
			newErr := fmt.Errorf("'MyImage' type is selected, but Specified image is Not a MyImage!!")
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}

		// CheckWindowsImage() in 'MyImage'Handler
		isMyWindowsImage, err := myImageHandler.CheckWindowsImage(vmReqInfo.ImageIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Check Whether My Image is MS Windows Image : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
		if isMyWindowsImage {
			var createErr error
			initUserData, createErr = vmHandler.createWinInitUserData(vmReqInfo.VMUserPasswd)
			if createErr != nil {
				newErr := fmt.Errorf("Failed to Create Cloud-Init Script with the Password : [%v]", createErr)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
		} else {
			var createErr error
			initUserData, createErr = vmHandler.createLinuxInitUserData(vmReqInfo.ImageIID, keyPairId)
			if createErr != nil {
				newErr := fmt.Errorf("Failed to Create Cloud-Init Script with the KeyPairId : [%v]", createErr)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
		}
	}
	// cblogger.Infof("init UserData : [%s]", *initUserData)

	// Preparing VM Creation Options
	serverCreateOpts := servers.CreateOpts{
		Name:           vmReqInfo.IId.NameId,
		SecurityGroups: sgIdList,
		ImageRef:       vmReqInfo.ImageIID.SystemId,
		FlavorRef:      vmSpecId,
		Networks: []servers.Network{
			{UUID: vpc.ID},
		},
		UserData: []byte(*initUserData), // Apply cloud-init script
	}
	if vmHandler.RegionInfo.TargetZone != "" {
		serverCreateOpts.AvailabilityZone = vmHandler.RegionInfo.TargetZone
	} else if vmHandler.RegionInfo.Zone != "" {
		serverCreateOpts.AvailabilityZone = vmHandler.RegionInfo.Zone
	}

	// Add KeyPair Name
	createOpts := keypairs.CreateOptsExt{
		KeyName: vmReqInfo.KeyPairIID.NameId,
	}

	nhnVMSpecType := vmReqInfo.VMSpecName[:2] // Ex) u2 or m2 or c2 ...
	cblogger.Infof("# nhnVMSpecType : [%s]", nhnVMSpecType)

	reqDiskType := vmReqInfo.RootDiskType // 'default', 'General_HDD' or 'General_SSD'
	reqDiskSize := vmReqInfo.RootDiskSize

	// Set VM RootDiskType
	if strings.EqualFold(reqDiskType, "General_HDD") {
		reqDiskType = HDD // "General HDD"
	} else if strings.EqualFold(reqDiskType, "General_SSD") {
		reqDiskType = SSD // "General SSD"
	}

	// In case, Volume Type is not specified.
	if strings.EqualFold(reqDiskType, "") || strings.EqualFold(reqDiskType, "default") {
		reqDiskType = HDD
	}

	// When Volume Type is Incorrect
	if strings.EqualFold(nhnVMSpecType, "u2") && !strings.EqualFold(reqDiskType, HDD) {
		newErr := fmt.Errorf("Invalid RootDiskType!! Specified VMSpec [%s] supports only 'default' or 'General_HDD' RootDiskType!!", vmReqInfo.VMSpecName)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	if strings.EqualFold(nhnVMSpecType, "u2") && (!strings.EqualFold(reqDiskSize, "") && !strings.EqualFold(reqDiskSize, "default")) {

		vmSpecHandler := NhnCloudVMSpecHandler{
			RegionInfo: vmHandler.RegionInfo,
			VMClient:   vmHandler.VMClient,
		}
		vmSpec, err := vmSpecHandler.GetVMSpec(vmReqInfo.VMSpecName) // Check vmSpec info.
		if err != nil {
			newErr := fmt.Errorf("Failed to Get VMSpec Info. with the name : %v", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}

		// Use Key/Value info of the vmSpec Info.
		var localDisk string
		for _, keyInfo := range vmSpec.KeyValueList {
			if strings.EqualFold(keyInfo.Key, "LocalDiskSize(GB)") {
				localDisk = keyInfo.Value
				break
			}
		}

		if reqDiskSize != localDisk {
			newErr := fmt.Errorf("Invalid RootDiskSize!! Specified VMSpec [%s] supports only [%s](GB)!!", vmReqInfo.VMSpecName, localDisk)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
	}

	if nhnVMSpecType != "u2" && (reqDiskType != HDD && reqDiskType != SSD) {
		newErr := fmt.Errorf("Invalid RootDiskType!! Must be 'default', 'General_HDD' or 'General_SSD'")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	var reqDiskSizeInt int
	// Set VM RootDiskSize
	// When Volume Size is not specified.
	imageOSPlatform, err := vmHandler.getOSPlatformWithImageID(vmReqInfo.ImageIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Image OSPlatform Info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	if imageOSPlatform == irs.WINDOWS {
		if strings.EqualFold(reqDiskSize, "") || strings.EqualFold(reqDiskSize, "default") {
			reqDiskSize = DefaultWinRootDiskSize
		}
		reqDiskSizeInt, err = strconv.Atoi(reqDiskSize)
		if err != nil {
			newErr := fmt.Errorf("Failed to Convert diskSize to int type. [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}

		// Volume Size must be more than 50GB and less than 1000GB (for Windows OS)
		if nhnVMSpecType != "u2" && (reqDiskSizeInt < 50 || reqDiskSizeInt > 1000) {
			newErr := fmt.Errorf("Invalid RootDiskSize!! RootDiskSize range should be 50 to 1000(GB) for Windows OS!!")
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
	} else {
		if strings.EqualFold(reqDiskSize, "") || strings.EqualFold(reqDiskSize, "default") {
			reqDiskSize = DefaultDiskSize
		}
		reqDiskSizeInt, err = strconv.Atoi(reqDiskSize)
		if err != nil {
			newErr := fmt.Errorf("Failed to Convert diskSize to int type. [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}

		// Volume Size must be more than 20GB and less than 1000GB (for Linux OS)
		if nhnVMSpecType != "u2" && (reqDiskSizeInt < 20 || reqDiskSizeInt > 1000) {
			newErr := fmt.Errorf("Invalid RootDiskSize!! RootDiskSize range should be 20 to 1000(GB) for Linux OS!!")
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
	}

	start := call.Start()
	createOpts.CreateOptsBuilder = serverCreateOpts

	var newNhnVM *servers.Server
	if strings.EqualFold(nhnVMSpecType, "u2") { // Only HDD and Default RootDiskSize according to the VMSpec
		newNhnVM, err = servers.Create(vmHandler.VMClient, createOpts).Extract()
		if err != nil {
			newErr := fmt.Errorf("Failed to Create a VM with the Local Disk!! [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
	} else {
		blockDeviceSet := []bootfromvolume.BlockDevice{
			{
				UUID:       vmReqInfo.ImageIID.SystemId,
				SourceType: bootfromvolume.SourceImage,
				// Note) In case of 'MyImage', SourceType is 'SourceImage', too.  Not 'bootfromvolume.SourceSnapshot'
				VolumeType:          reqDiskType,
				VolumeSize:          reqDiskSizeInt,
				DestinationType:     bootfromvolume.DestinationVolume, // Destination_type must be 'Volume'. Not 'bootfromvolume.DestinationLocal' when Not u2 type.
				DeleteOnTermination: true,
			},
		}

		bootOpts := bootfromvolume.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			BlockDevice:       blockDeviceSet,
		}
		newNhnVM, err = bootfromvolume.Create(vmHandler.VMClient, bootOpts).Extract()
		if err != nil {
			newErr := fmt.Errorf("Failed to Create a VM with the Block Storage Volume!! [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
	}
	LoggingInfo(callLogInfo, start)

	// Because there are functions that use NameID, input NameId too
	newVMIID := irs.IID{NameId: vmReqInfo.IId.NameId, SystemId: newNhnVM.ID}

	// Wait for created VM info to be inquired
	curStatus, errStatus := vmHandler.waitToGetVMInfo(newVMIID)
	if errStatus != nil {
		newErr := fmt.Errorf("Failed to Wait to Get VM Info!! [%v]", errStatus)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	cblogger.Infof("==> VM status of [%s] : [%s]", newVMIID.NameId, curStatus)

	// Set Disk Attachment Info
	diskHandler := NhnCloudDiskHandler{
		RegionInfo:   vmHandler.RegionInfo,
		VMClient:     vmHandler.VMClient,
		VolumeClient: vmHandler.VolumeClient,
	}
	if len(vmReqInfo.DataDiskIIDs) != 0 {
		for _, DataDiskIID := range vmReqInfo.DataDiskIIDs {
			_, err := diskHandler.AttachDisk(DataDiskIID, newVMIID)
			if err != nil {
				newErr := fmt.Errorf("Failed to Attach the Disk Volume to the VM!! [%v]", err)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
			cblogger.Infof("# Disk [%s] Attached Successfully!!", DataDiskIID.SystemId)
		}
	}

	// To Check VM Deployment Status
	nhnVM, err := servers.Get(vmHandler.VMClient, newNhnVM.ID).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VMInfo : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	vmInfo := irs.VMInfo{}
	if strings.EqualFold(nhnVM.Status, "active") {
		// Associate Public IP to the VM
		if ok, err := vmHandler.associatePublicIP(nhnVM.ID); !ok {
			newErr := fmt.Errorf("Failed to Start VM. Failed to Associate PublicIP : %v", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
		// Get Final VM info
		nhnVM, err := servers.Get(vmHandler.VMClient, nhnVM.ID).Extract()
		if err != nil {
			newErr := fmt.Errorf("Failed to Get New VM Info. %s", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}

		var mappingErr error
		vmInfo, mappingErr = vmHandler.mappingVMInfo(*nhnVM)
		if mappingErr != nil {
			newErr := fmt.Errorf("Failed to Map New VM Info. %s", mappingErr)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
	}
	return vmInfo, nil
}

func (vmHandler *NhnCloudVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NHN Cloud Driver: called SuspendVM()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Region, call.VM, vmIID.SystemId, "SuspendVM()")

	var resultStatus string

	cblogger.Info("Start Get VM Status...")
	vm, vmStatus, err := vmHandler.getVMStatus(vmIID)
	if err != nil {
		cblogger.Errorf("[%s] Failed to Get the VM Status of VM : ", vm.ID)
		cblogger.Error(err)
		LoggingError(callLogInfo, err)
		return irs.VMStatus("Failed to Get the VM Status of VM. "), err
	} else {
		cblogger.Infof("Succeeded in Getting the VM Status of [%s] : [%s]", vm.ID, vmStatus)
	}

	if strings.EqualFold(string(vmStatus), "Suspended") {
		resultStatus = "The VM has already been Suspended."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else if strings.EqualFold(string(vmStatus), "Rebooting") {
		resultStatus = "The VM is in the process of Rebooting."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else if strings.EqualFold(string(vmStatus), "Deleted") {
		resultStatus = "The VM has been Deleted."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else if strings.EqualFold(string(vmStatus), "Creating") {
		resultStatus = "The VM is in the process of Creating."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else if strings.EqualFold(string(vmStatus), "Terminating") {
		resultStatus = "The VM is in the process of Terminating."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else {
		start := call.Start()
		err := startstop.Stop(vmHandler.VMClient, vm.ID).Err
		if err != nil {
			newErr := fmt.Errorf("Failed to Suspend the VM!! : [%v] ", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.Failed, newErr
		}
		LoggingInfo(callLogInfo, start)
	}

	return irs.Suspending, nil
}

func (vmHandler *NhnCloudVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NHN Cloud Driver: called ResumeVM()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Region, call.VM, vmIID.NameId, "ResumeVM()")

	var resultStatus string

	cblogger.Info("Start Get VM Status...")
	vm, vmStatus, err := vmHandler.getVMStatus(vmIID)
	if err != nil {
		cblogger.Errorf("Failed to Get the VM Status of : [%s]", vm.ID)
		cblogger.Error(err)
		LoggingError(callLogInfo, err)
		return irs.VMStatus("Failed. "), err
	} else {
		cblogger.Infof("Succeeded in Getting the VM Status of [%s] : [%s]", vm.ID, vmStatus)
	}

	if strings.EqualFold(string(vmStatus), "Running") {
		resultStatus = "The VM is Running. Cannot be Resumed!!"
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else if strings.EqualFold(string(vmStatus), "Rebooting") {
		resultStatus = "The VM is in the process of Rebooting. Cannot be Resumed"
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else if strings.EqualFold(string(vmStatus), "Terminating") {
		resultStatus = "The VM is already in the process of Terminating. Cannot be Resumed"
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else if strings.EqualFold(string(vmStatus), "Deleted") {
		resultStatus = "The VM has been Deleted. Cannot be Resumed"
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else if strings.EqualFold(string(vmStatus), "Creating") {
		resultStatus = "The VM is in the process of Creating. Cannot be Resumed"
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else {
		start := call.Start()
		err := startstop.Start(vmHandler.VMClient, vm.ID).Err
		if err != nil {
			newErr := fmt.Errorf("Failed to Start the VM!! : [%v] ", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.Failed, newErr
		}
		LoggingInfo(callLogInfo, start)

		return irs.Resuming, nil
	}
}

func (vmHandler *NhnCloudVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NHN Cloud Driver: called RebootVM()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Region, call.VM, vmIID.SystemId, "RebootVM()")

	cblogger.Info("Start Get VM Status...")
	vm, vmStatus, err := vmHandler.getVMStatus(vmIID)
	if err != nil {
		cblogger.Errorf("[%s] Failed to Get the VM Status.", vmIID)
		cblogger.Error(err)
		LoggingError(callLogInfo, err)
		return irs.VMStatus("Failed to Get the VM Status."), err
	} else {
		cblogger.Infof("Succeeded in Getting the VM Status of [%s] : [%s]", vmIID, vmStatus)
	}

	var resultStatus string

	if strings.EqualFold(string(vmStatus), "Suspended") {
		resultStatus = "The VM had been Suspended."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else if strings.EqualFold(string(vmStatus), "Rebooting") {
		resultStatus = "The VM is already in the process of Rebooting."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else if strings.EqualFold(string(vmStatus), "Deleted") {
		resultStatus = "The VM has been Deleted."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else if strings.EqualFold(string(vmStatus), "Booting") {
		resultStatus = "The VM is in the process of Booting."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else if strings.EqualFold(string(vmStatus), "Creating") {
		resultStatus = "The VM is in the process of Creating."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else if strings.EqualFold(string(vmStatus), "Terminating") {
		resultStatus = "The VM is in the process of Terminating."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	} else {
		start := call.Start()
		rebootOpts := servers.RebootOpts{
			Type: servers.SoftReboot,
		}

		err := servers.Reboot(vmHandler.VMClient, vm.ID, rebootOpts).ExtractErr()
		if err != nil {
			newErr := fmt.Errorf("Failed to Reboot the VM!! : [%v] ", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.Failed, newErr
		}
		LoggingInfo(callLogInfo, start)

		return irs.Rebooting, nil
	}
}

func (vmHandler *NhnCloudVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NHN Cloud Driver: called TerminateVM()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Region, call.VM, vmIID.SystemId, "TerminateVM()")

	server, err := vmHandler.GetVM(vmIID)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		return irs.Failed, err
	}

	allPages, err := floatingips.List(vmHandler.VMClient).AllPages()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		return irs.Failed, err
	}
	publicIPList, err := floatingips.ExtractFloatingIPs(allPages)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		return irs.Failed, err
	}

	var publicIPId string
	for _, p := range publicIPList {
		if strings.EqualFold(server.PublicIP, p.IP) {
			publicIPId = p.ID
			break
		}
	}

	if publicIPId != "" {
		err := floatingips.Delete(vmHandler.VMClient, publicIPId).ExtractErr()
		if err != nil {
			newErr := fmt.Errorf("Failed to Delete the Floating IP!! : [%v] ", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.Failed, newErr
		}
	}

	start := call.Start()
	err = servers.Delete(vmHandler.VMClient, server.IId.SystemId).ExtractErr()
	if err != nil {
		newErr := fmt.Errorf("Failed to Terminate the VM!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.Failed, newErr
	}
	LoggingInfo(callLogInfo, start)

	return irs.Terminating, nil
}

func (vmHandler *NhnCloudVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ListVMStatus()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Region, call.VM, "ListVMStatus()", "ListVMStatus()")

	start := call.Start()
	allPages, err := servers.List(vmHandler.VMClient, nil).AllPages()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		return nil, err
	}
	LoggingInfo(callLogInfo, start)

	ss, err := servers.ExtractServers(allPages)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		return nil, err
	}

	// Add to List
	vmStatusList := make([]*irs.VMStatusInfo, len(ss))
	for idx, s := range ss {
		vmStatus := getVmStatus(s.Status)
		vmStatusInfo := irs.VMStatusInfo{
			IId: irs.IID{
				NameId:   s.Name,
				SystemId: s.ID,
			},
			VmStatus: vmStatus,
		}
		vmStatusList[idx] = &vmStatusInfo
	}

	return vmStatusList, nil
}

func (vmHandler *NhnCloudVMHandler) getVMStatus(vmIID irs.IID) (servers.Server, irs.VMStatus, error) {
	cblogger.Info("NHN Cloud Driver: called GetVMStatus()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Region, call.VM, vmIID.SystemId, "GetVMStatus()")

	start := call.Start()
	nhnVM, err := vmHandler.getRawVM(vmIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM info.!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return servers.Server{}, irs.Failed, newErr
	}
	LoggingInfo(callLogInfo, start)

	cblogger.Infof("# serverResult.Status of NHN Cloud : [%s]", nhnVM.Status)
	vmStatus := getVmStatus(nhnVM.Status)
	return nhnVM, vmStatus, nil
}

func (vmHandler *NhnCloudVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	_, vmStatus, err := vmHandler.getVMStatus(vmIID)
	return vmStatus, err
}

func (vmHandler *NhnCloudVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ListVM()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Region, call.VM, "ListVM()", "ListVM()")

	start := call.Start()
	listOpts := servers.ListOpts{
		Limit: 100,
	}
	allPages, err := servers.List(vmHandler.VMClient, listOpts).AllPages()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		return nil, err
	}
	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		return nil, err
	}
	LoggingInfo(callLogInfo, start)

	// Mapping VM info list
	var vmInfoList []*irs.VMInfo
	for _, server := range serverList {
		vmInfo, err := vmHandler.mappingVMInfo(server)
		if err != nil {
			newErr := fmt.Errorf("Failed to Map New VM Info. %s", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return nil, newErr
		}
		vmInfoList = append(vmInfoList, &vmInfo)
	}
	return vmInfoList, nil
}

func (vmHandler *NhnCloudVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetVM()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Region, call.VM, vmIID.SystemId, "GetVM()")

	start := call.Start()
	nhnVM, err := vmHandler.getRawVM(vmIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM info.!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	vmInfo, err := vmHandler.mappingVMInfo(nhnVM)
	if err != nil {
		newErr := fmt.Errorf("Failed to Map New VM Info. %s", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	return vmInfo, nil
}

func (vmHandler *NhnCloudVMHandler) associatePublicIP(serverID string) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called associatePublicIP()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Region, call.VM, "associatePublicIP()", "associatePublicIP()")

	if strings.EqualFold(serverID, "") {
		newErr := fmt.Errorf("Invalid serverID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	// Create PublicIP
	extVPCName, _ := getPublicVPCInfo(vmHandler.NetworkClient, "NAME")
	createOpts := floatingips.CreateOpts{
		Pool: extVPCName,
	}
	publicIP, err := floatingips.Create(vmHandler.VMClient, createOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Public IP!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	// Associate Floating IP to the VM
	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		associateOpts := floatingips.AssociateOpts{
			FloatingIP: publicIP.IP,
		}
		err = floatingips.AssociateInstance(vmHandler.VMClient, serverID, associateOpts).ExtractErr()
		if err == nil {
			break
		} else {
			newErr := fmt.Errorf("Failed to AssociateInstance the Public IP!! : [%v] ", err)
			cblogger.Error(newErr.Error())
		}

		time.Sleep(1 * time.Second)
		curRetryCnt++
		if curRetryCnt > maxRetryCnt {
			newErr := fmt.Errorf("Failed to Associate Floating IP to VM, Exceeded Maximum Retry Count %d", maxRetryCnt)
			cblogger.Error(newErr.Error())
			return false, newErr
		}
	}

	return true, nil
}

func getVmStatus(vmStatus string) irs.VMStatus {
	cblogger.Info("NHN Cloud Driver: called getVmStatus()")

	var resultStatus string
	switch strings.ToLower(vmStatus) {
	case "build":
		resultStatus = "Creating"
	case "active":
		resultStatus = "Running"
	case "shutoff":
		resultStatus = "Suspended"
	case "paused":
		resultStatus = "Suspended"
	case "reboot":
		resultStatus = "Rebooting"
	case "hard_reboot":
		resultStatus = "Rebooting"
	case "deleted":
		resultStatus = "Deleted"
	case "error":
		resultStatus = "Error"
	default:
		resultStatus = "Unknown"
	}

	return irs.VMStatus(resultStatus)
}

func getAvailabilityZoneFromAPI(computeClient *nhnsdk.ServiceClient, serverID string) (string, error) {
	url := computeClient.ServiceURL("servers", serverID)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("X-Auth-Token", computeClient.TokenID)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get server details: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var serverResponse map[string]interface{}
	if err := json.Unmarshal(body, &serverResponse); err != nil {
		return "", err
	}
	if server, ok := serverResponse["server"].(map[string]interface{}); ok {
		if zone, ok := server["OS-EXT-AZ:availability_zone"].(string); ok {
			return zone, nil
		}
	}
	return "", fmt.Errorf("availability zone not found")
}

func (vmHandler *NhnCloudVMHandler) mappingVMInfo(server servers.Server) (irs.VMInfo, error) {
	cblogger.Info("NHN Cloud Driver: called mappingVMInfo()")
	// cblogger.Infof("\n\n### Server from NHN :")
	// spew.Dump(server)
	// cblogger.Infof("\n\n")

	convertedTime, err := convertTimeToKTC(server.Created)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Converted Time. [%v]", err)
		return irs.VMInfo{}, newErr
	}

	vmInfo := irs.VMInfo{
		IId: irs.IID{
			NameId:   server.Name,
			SystemId: server.ID,
		},
		Region: irs.RegionInfo{
			Region: vmHandler.RegionInfo.Region,
		},
		KeyPairIId: irs.IID{
			NameId:   server.KeyName,
			SystemId: server.KeyName,
		},
		// VMUserPasswd:      "N/A",
		NetworkInterface: server.HostID,
	}
	vmInfo.StartTime = convertedTime

	zone, err := getAvailabilityZoneFromAPI(vmHandler.VMClient, server.ID)
	if err != nil {
		cblogger.Warn(err)
	}
	if zone != "" {
		vmInfo.Region.Zone = zone
	} else if vmHandler.RegionInfo.TargetZone != "" {
		vmInfo.Region.Zone = vmHandler.RegionInfo.TargetZone
	} else {
		vmInfo.Region.Zone = vmHandler.RegionInfo.Zone
	}

	// Image Info
	imageId := server.Image["id"].(string)
	nhnImage, err := comimages.Get(vmHandler.VMClient, imageId).Extract() // Caution!!) Wtih VMClient (Not Like ImageHandler)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Image info from NHN Cloud!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	} else if nhnImage != nil {
		vmInfo.ImageIId.NameId = nhnImage.ID
		vmInfo.ImageIId.SystemId = nhnImage.ID
	}

	// Flavor Info
	var vRam string
	var vCPU string
	flavorId, ok := server.Flavor["id"].(string)
	if ok {
		nhnFlavor, err := flavors.Get(vmHandler.VMClient, flavorId).Extract()
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Flavor info from NHN Cloud!! : [%v] ", err)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		} else if nhnFlavor != nil {
			vCPU = strconv.Itoa(nhnFlavor.VCPUs)
			vRam = strconv.Itoa(nhnFlavor.RAM)
		}
	}

	// Get Disk Type, Size Info and DataDiskIIDs
	var diskIIDs []irs.IID
	if len(server.AttachedVolumes) != 0 {
		for _, volume := range server.AttachedVolumes {
			cblogger.Infof("\n# Volume ID : %s", volume.ID)

			nhnVolume, err := volumes.Get(vmHandler.VolumeClient, volume.ID).Extract()
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the Volume Info from NHN Cloud!! : [%v] ", err)
				cblogger.Error(newErr.Error())
				return irs.VMInfo{}, newErr
			}
			if nhnVolume.Bootable == "true" { // Only For 'Root' disk
				// cblogger.Info("# nhnVolume : ")
				// spew.Dump(nhnVolume)
				switch nhnVolume.VolumeType {
				case HDD:
					vmInfo.RootDiskType = "General_HDD"
				case SSD:
					vmInfo.RootDiskType = "General_SSD"
				case "":
					vmInfo.RootDiskType = "N/A"
				}

				vmInfo.RootDiskSize = strconv.Itoa(nhnVolume.Size)
				vmInfo.RootDeviceName = nhnVolume.Attachments[0].Device
			} else {
				diskIIDs = append(diskIIDs, irs.IID{NameId: nhnVolume.Name, SystemId: nhnVolume.ID})
			}
		}
	}
	vmInfo.DataDiskIIDs = diskIIDs

	for key, subnet := range server.Addresses {
		// VPC Info
		vmInfo.VpcIID.NameId = key
		nhnVPC, err := getVPCWithName(vmHandler.NetworkClient, vmInfo.VpcIID.NameId)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the NHN Cloud VPC Info!! : [%v] ", err)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		} else if nhnVPC != nil {
			vmInfo.VpcIID.SystemId = nhnVPC.ID
		}
		// PrivateIP, PublicIp Info
		for _, addr := range subnet.([]interface{}) {
			addrMap := addr.(map[string]interface{})
			if addrMap["OS-EXT-IPS:type"] == "floating" {
				vmInfo.PublicIP = addrMap["addr"].(string)
			} else if addrMap["OS-EXT-IPS:type"] == "fixed" {
				vmInfo.PrivateIP = addrMap["addr"].(string)
			}
		}
	}

	// # Get Subnet and NetworkInterface Info
	if !strings.EqualFold(vmInfo.PublicIP, "") {
		// Subnet, Network Interface Info
		nhnPort, err := getPortWithDeviceId(vmHandler.NetworkClient, vmInfo.IId.SystemId)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the NHN Cloud Port Info!! : [%v] ", err)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		} else if nhnPort != nil {
			// Subnet Info
			if len(nhnPort.FixedIPs) > 0 {
				vmInfo.SubnetIID.SystemId = nhnPort.FixedIPs[0].SubnetID
			}

			nhnVpcsubnet, err := getVpcsubnetWithId(vmHandler.NetworkClient, vmInfo.SubnetIID.SystemId)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the Subnet Info!! : [%v] ", err)
				cblogger.Error(newErr.Error())
				return irs.VMInfo{}, newErr
			} else if nhnVpcsubnet != nil {
				vmInfo.SubnetIID.NameId = nhnVpcsubnet.Name
			}
			// Network Interface Info
			vmInfo.NetworkInterface = nhnPort.ID
		}
	}

	// # Get SecurityGroup Info
	if len(server.SecurityGroups) != 0 {
		sgIIds := make([]irs.IID, len(server.SecurityGroups))
		for i, secGroupMap := range server.SecurityGroups {
			secGroupName := secGroupMap["name"].(string)
			sgIIds[i] = irs.IID{
				NameId: secGroupName,
			}
			secGroup, err := getSGWithName(vmHandler.VMClient, secGroupName)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the Security Group Info!! : [%v] ", err)
				cblogger.Error(newErr.Error())
				return irs.VMInfo{}, newErr
			} else if secGroup != nil {
				sgIIds[i].SystemId = secGroup.ID
			}
		}
		vmInfo.SecurityGroupIIds = sgIIds
	}

	imageOSPlatform, err := vmHandler.getOSPlatformWithImageID(vmInfo.ImageIId.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Image OSPlatform Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}
	vmInfo.Platform = imageOSPlatform

	if (vmInfo.PublicIP != "") && (vmInfo.Platform == irs.WINDOWS) {
		vmInfo.VMUserId = DefaultWindowsUserName
		vmInfo.SSHAccessPoint = vmInfo.PublicIP + ":3389"
	} else if (vmInfo.PublicIP != "") && (vmInfo.Platform == irs.LINUX_UNIX) {
		vmInfo.VMUserId = DefaultVMUserName
		vmInfo.SSHAccessPoint = vmInfo.PublicIP + ":22"
	}

	vmInfo.KeyValueList = irs.StructToKeyValueList(server)

	vmInfo.KeyValueList = append(vmInfo.KeyValueList,
		irs.KeyValue{Key: "vCPU", Value: vCPU},
		irs.KeyValue{Key: "vRAM(GB)", Value: vRam},
	)

	return vmInfo, nil
}

// Waiting for up to 500 seconds during VM creation until VM info. can be get
func (vmHandler *NhnCloudVMHandler) waitToGetVMInfo(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("===> Since VM info. cannot be retrieved immediately after VM creation, it waits until running.")

	curRetryCnt := 0
	maxRetryCnt := 500
	for {
		_, curStatus, err := vmHandler.getVMStatus(vmIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the VM Status of [%s] : [%v] ", vmIID.NameId, err)
			cblogger.Error(newErr.Error())
			return irs.VMStatus("Failed. "), newErr
		} else {
			cblogger.Infof("Succeeded in Getting the VM Status of [%s] : [%s]", vmIID.NameId, curStatus)
		}

		cblogger.Info("===> VM Status : ", curStatus)

		switch string(curStatus) {
		case "Creating", "Booting":
			curRetryCnt++
			cblogger.Infof("The VM is still 'Creating', so wait for a second more before inquiring the VM info.")
			time.Sleep(time.Second * 5)
			if curRetryCnt > maxRetryCnt {
				newErr := fmt.Errorf("Despite waiting for a long time(%d sec), the VM status is %s, so it is forcibly finished.", maxRetryCnt, curStatus)
				cblogger.Error(newErr.Error())
				return irs.VMStatus("Failed. "), newErr
			}
		default:
			cblogger.Infof("===> ### The VM Creation is finished, stopping the waiting.")
			return irs.VMStatus(curStatus), nil
			//break
		}
	}
}

func (vmHandler *NhnCloudVMHandler) getOSPlatformWithImageID(imageId string) (irs.Platform, error) {
	cblogger.Info("NHN Cloud Driver: called getOSPlatformWithImageID()")

	if strings.EqualFold(imageId, "") {
		newErr := fmt.Errorf("Invalid Image ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	nhnImage, err := comimages.Get(vmHandler.VMClient, imageId).Extract() // Caution!!) With VMClient (Not Like NHN Cloud ImageHandler)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	osType, exist := nhnImage.Metadata["os_type"].(string)
	if !exist {
		newErr := fmt.Errorf("Failed to Find OSType Info from the Image Info!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	if strings.EqualFold(osType, "windows") {
		return irs.WINDOWS, nil
	} else if strings.EqualFold(osType, "linux") {
		return irs.LINUX_UNIX, nil
	}
	return irs.LINUX_UNIX, nil
}

func (vmHandler *NhnCloudVMHandler) createLinuxInitUserData(imageIID irs.IID, keyPairId string) (*string, error) {
	cblogger.Info("NHN Cloud driver: called createLinuxInitUserData()!!")

	// Get KeyPair Info from NHN Cloud (to Get PublicKey info for cloud-init)
	var getOptsBuilder keypairs.GetOptsBuilder
	keyPair, err := keypairs.Get(vmHandler.VMClient, keyPairId, getOptsBuilder).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KeyPair Info. with the name : %v", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Set cloud-init script
	rootPath := os.Getenv("CBSPIDER_ROOT")
	fileData, err := os.ReadFile(rootPath + UbuntuCloudInitFilePath)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find and Open the Cloud-Init File : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Infof("Succeeded in Finding and Opening the S/G file: ")
	}
	fileStr := string(fileData)
	fileStr = strings.ReplaceAll(fileStr, "{{username}}", DefaultVMUserName)
	fileStr = strings.ReplaceAll(fileStr, "{{public_key}}", keyPair.PublicKey)
	// cblogger.Info("\n# fileStr : ")
	// spew.Dump(fileStr)

	return &fileStr, nil
}

func (vmHandler *NhnCloudVMHandler) createWinInitUserData(passWord string) (*string, error) {
	cblogger.Info("NHN Cloud driver: called createWinInitUserData()!!")

	// Set cloud-init script
	rootPath := os.Getenv("CBSPIDER_ROOT")
	fileData, err := os.ReadFile(rootPath + WinCloudInitFilePath)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find and Open the Cloud-Init File : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Infof("Succeeded in Finding and Opening the S/G file: ")
	}
	fileStr := string(fileData)
	fileStr = strings.ReplaceAll(fileStr, "{{username}}", DefaultWindowsUserName)
	fileStr = strings.ReplaceAll(fileStr, "{{PASSWORD}}", passWord) // For Windows VM
	// cblogger.Info("\n# fileStr : ")
	// spew.Dump(fileStr)
	return &fileStr, nil
}

func (vmHandler *NhnCloudVMHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("NHN Cloud Driver: called ListIID()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Region, call.VM, "ListIID()", "ListIID()")

	start := call.Start()

	var iidList []*irs.IID

	listOpts := servers.ListOpts{
		Limit: 100,
	}
	allPages, err := servers.List(vmHandler.VMClient, listOpts).AllPages()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		return nil, err
	}
	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		return nil, err
	}

	for _, server := range serverList {
		var iid irs.IID
		iid.SystemId = server.ID
		iid.NameId = server.Name

		iidList = append(iidList, &iid)
	}

	LoggingInfo(callLogInfo, start)

	return iidList, nil
}
