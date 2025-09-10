// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2022.12.
// Updated by ETRI 2024.01.

package resources

import (
	"errors"
	"fmt"

	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // To prevent 'unknown time zone Asia/Seoul' error

	// "encoding/json"
	// "github.com/davecgh/go-spew/spew"

	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"
	volumes2 "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/blockstorage/v2/volumes"
	volumeboot "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/extensions/bootfromvolume"
	keys "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/extensions/keypairs"
	startstop "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/extensions/startstop"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/pagination"

	// flavors  "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/flavors"
	servers "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/servers"

	images "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/imageservice/v2/images" // Caution!!
	//Ref) 'Image API' return struct of image : ktcloudvpc-sdk-go/openstack/imageservice/v2/images/results.go
	// "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/images"
	//Ref) 'Compute API' return struct of image : ktcloudvpc-sdk-go/openstack/compute/v2/images/results.go

	job "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/job"
	ips "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/extensions/floatingips"
	portforward "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/extensions/layer3/portforwarding"
	rules "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/extensions/fwaas_v2/rules"	

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	keycommon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	sim "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/kt/resources/info_manager/security_group_info_manager"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	LnxUserName string = "cb-user"
	WinUserName string = "Administrator"

	UbuntuCloudInitFilePath string = "/cloud-driver-libs/.cloud-init-kt/cloud-init-ubuntu"
	CentosCloudInitFilePath string = "/cloud-driver-libs/.cloud-init-kt/cloud-init-centos"
	WinCloudInitFilePath    string = "/cloud-driver-libs/.cloud-init-kt/cloud-init-windows"

	DefaultUsagePlan        string = "hourly"
	DefaultDiskSize         string = "50"
	DefaultDiskSize2        string = "100"
	DefaultWinRootDiskSize  string = "100"
	DefaultWinRootDiskSize2 string = "150"
)

type KTVpcVMHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *ktvpcsdk.ServiceClient
	ImageClient    *ktvpcsdk.ServiceClient
	NetworkClient  *ktvpcsdk.ServiceClient
	VolumeClient   *ktvpcsdk.ServiceClient
}

// NetworkInfo holds information about a VM's public network interface.
// PublicIP is the public IP address assigned to the VM.
// PublicIPID is the identifier of the public IP resource.
type NetworkInfo struct {
	PublicIP   string
	PublicIPID string
}

func (vmHandler *KTVpcVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called StartVM()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmReqInfo.IId.NameId, "StartVM()")

	if strings.EqualFold(vmReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid VM Name!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	// Check VM Name Duplication
	vmExist, err := vmHandler.vmExists(vmReqInfo.IId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create VM. : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	if vmExist {
		newErr := fmt.Errorf("Failed to Create VM. The Name [%s] already exists", vmReqInfo.IId.NameId)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	// Check Flavor Info. (Change Name to ID)
	vmSpecId, err := getFlavorIdWithName(vmHandler.VMClient, vmReqInfo.VMSpecName)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VMSpec ID with the name : %v", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	cblogger.Infof("# vmSpec ID : [%s]", vmSpecId)

	// Check Image Info.
	// imageExist, imageErr := vmHandler.imageExists(vmReqInfo.ImageIID)
	// if imageErr != nil {
	// 	newErr := fmt.Errorf("Failed to Create VM. : [%v]", imageErr)
	// 	cblogger.Error(newErr.Error())
	// 	loggingError(callLogInfo, newErr)
	// 	return irs.VMInfo{}, newErr
	// }
	// if !imageExist {
	// 	newErr := fmt.Errorf("Failed to Create VM. The Image System ID [%s] does Not Exist", vmReqInfo.ImageIID.SystemId)
	// 	cblogger.Error(newErr.Error())
	// 	loggingError(callLogInfo, newErr)
	// 	return irs.VMInfo{}, newErr
	// }

	createOpts := keys.CreateOptsExt{
		KeyName: vmReqInfo.KeyPairIID.NameId, // Set KeyPair Name
	}

	// Preparing VM Creation Options
	vmCreateOpts := servers.CreateOpts{
		Name:             vmReqInfo.IId.NameId,
		KeyName:          vmReqInfo.KeyPairIID.NameId,
		FlavorRef:        vmSpecId,
		AvailabilityZone: vmHandler.RegionInfo.Zone, // Caution : D1 flatform supports only 'zone'.
		Networks: []servers.Network{
			{UUID: vmReqInfo.SubnetIID.SystemId}, // Caution : Network 'Tier'의 id 값
		},
		UsagePlanType: DefaultUsagePlan,
	}

	// // Get KeyPair Info (To get PublicKey info for cloud-init)
	// var getOptsBuilder keys.GetOptsBuilder
	// keyPair, err := keys.Get(vmHandler.VMClient, vmReqInfo.KeyPairIID.NameId, getOptsBuilder).Extract()
	// if err != nil {
	// 	newErr := fmt.Errorf("Failed to Get KeyPair Info. with the name : %v", err)
	// 	cblogger.Error(newErr.Error())
	// 	loggingError(callLogInfo, newErr)
	// 	return irs.VMInfo{}, newErr
	// }
	// cblogger.Info("\n ### PublicKey : ")
	// spew.Dump(keyPair.PublicKey)

	// # Preparing for UserData String for Linux and Windows Platform
	var initUserData *string
	var keyPairId string
	var rootDiskSize string
	if !strings.EqualFold(vmReqInfo.KeyPairIID.SystemId, "") {
		keyPairId = vmReqInfo.KeyPairIID.SystemId
	} else {
		keyPairId = vmReqInfo.KeyPairIID.NameId
	}
	if vmReqInfo.ImageType == irs.PublicImage || vmReqInfo.ImageType == "" || vmReqInfo.ImageType == "default" {
		// isPublicImage() in ImageHandler
		imageHandler := KTVpcImageHandler{
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
			loggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
		if isPublicWindowsImage { // # Incase of Public Windows image
			// Root Disk Size is supported at only 50GB for Linux and 100GB for Windows OS.
			if !strings.EqualFold(vmReqInfo.RootDiskSize, "") && !strings.EqualFold(vmReqInfo.RootDiskSize, "default") && !strings.EqualFold(vmReqInfo.RootDiskSize, DefaultWinRootDiskSize) && !strings.EqualFold(vmReqInfo.RootDiskSize, DefaultWinRootDiskSize2) {
				newErr := fmt.Errorf("Invalid RootDiskSize!! Root Disk Size is supported at only 50GB/100GB for Linux and 100GB/150GB for Windows OS.")
				cblogger.Error(newErr.Error())
				loggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}

			// In case the Root Volume Size is not specified.
			reqDiskSize := vmReqInfo.RootDiskSize
			if strings.EqualFold(reqDiskSize, "") || strings.EqualFold(reqDiskSize, "default") {
				rootDiskSize = DefaultWinRootDiskSize
			} else {
				rootDiskSize = reqDiskSize
			}

			var createErr error
			initUserData, createErr = vmHandler.createWinInitUserData(vmReqInfo.VMUserPasswd)
			if createErr != nil {
				newErr := fmt.Errorf("Failed to Create Cloud-Init Script with the Password : [%v]", createErr)
				cblogger.Error(newErr.Error())
				loggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
		} else { // # Incase of Public Linux image
			// Root Disk Size is supported at only 50GB for Linux and 100GB for Windows OS.
			if !strings.EqualFold(vmReqInfo.RootDiskSize, "") && !strings.EqualFold(vmReqInfo.RootDiskSize, "default") && !strings.EqualFold(vmReqInfo.RootDiskSize, DefaultDiskSize) && !strings.EqualFold(vmReqInfo.RootDiskSize, DefaultDiskSize2) {
				newErr := fmt.Errorf("Invalid RootDiskSize!! Root Disk Size is supported at only 50GB/100GB for Linux and 100GB/150GB for Windows OS.")
				cblogger.Error(newErr.Error())
				loggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}

			// In case the Root Volume Size is not specified.
			reqDiskSize := vmReqInfo.RootDiskSize
			if strings.EqualFold(reqDiskSize, "") || strings.EqualFold(reqDiskSize, "default") {
				rootDiskSize = DefaultDiskSize
			} else {
				rootDiskSize = reqDiskSize
			}

			var createErr error
			initUserData, createErr = vmHandler.createLinuxInitUserData(keyPairId)
			if createErr != nil {
				newErr := fmt.Errorf("Failed to Create Cloud-Init Script with the KeyPairId : [%v]", createErr)
				cblogger.Error(newErr.Error())
				loggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
		}
	} else { // In case of MyImage
		var createErr error
		initUserData, createErr = vmHandler.createLinuxInitUserData(keyPairId)
		if createErr != nil {
			newErr := fmt.Errorf("Failed to Create Cloud-Init Script with the KeyPairId : [%v]", createErr)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
	}

	vmCreateOpts.UserData = []byte(*initUserData) // Apply cloud-init script
	createOpts.CreateOptsBuilder = vmCreateOpts

	// cblogger.Infof("# Image ID : [%s]", vmReqInfo.ImageIID.SystemId)

	// Set VM Booting Source Type
	// Note) In case of 'MyImage', SourceType is 'SourceImage', too.  Not 'volumeboot.SourceSnapshot'
	bootSourceType := volumeboot.SourceImage
	if vmReqInfo.ImageType == irs.PublicImage || vmReqInfo.ImageType == "" || vmReqInfo.ImageType == "default" {
		bootSourceType = volumeboot.SourceImage // volumeboot.SourceType => "image"
	} else if vmReqInfo.ImageType == irs.MyImage {
		bootSourceType = volumeboot.SourceImage // volumeboot.SourceType => "image". Not "snapshot"
		// bootSourceType = volumeboot.SourceSnapshot	// volumeboot.SourceType => "snapshot"
	}

	// When Root Volume Size is not specified.
	reqDiskSize := vmReqInfo.RootDiskSize
	if strings.EqualFold(reqDiskSize, "") || strings.EqualFold(reqDiskSize, "default") {
		reqDiskSize = DefaultDiskSize
	}

	blockDeviceSet := []volumeboot.BlockDevice{
		{
			DestinationType: volumeboot.DestinationVolume, // DestinationType is the type that gets created. Possible values are "volume" and "local". volumeboot.DestinationType => "volume"
			BootIndex:       0,                            // BootIndex is the boot index. It defaults to 0. Set as the Root Volume.
			SourceType:      bootSourceType,               // volumeboot.SourceImage
			VolumeSize:      rootDiskSize,                 // VolumeSize is the size of the volume to create (in gigabytes). This can be omitted for existing volumes.
			VolumeType:      vmReqInfo.RootDiskType,
			UUID:            vmReqInfo.ImageIID.SystemId,
		},
	}

	bootOpts := volumeboot.CreateOptsExt{
		CreateOptsBuilder: createOpts,
		BlockDevice:       blockDeviceSet,
	}
	// cblogger.Info("\n ### Boot Options : ")
	// spew.Dump(bootOpts)

	vm, err := volumeboot.Create(vmHandler.VMClient, bootOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create VM!! [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	// cblogger.Infof("# New VM ID : [%s]", vm.ID)

	// Because there are some functions that use NameID, so input NameId too
	newVMIID := irs.IID{NameId: vmReqInfo.IId.NameId, SystemId: vm.ID}

	// Wait for created VM info to be inquired
	curStatus, errStatus := vmHandler.waitToGetVMInfo(newVMIID)
	if errStatus != nil {
		cblogger.Error(errStatus.Error())
		loggingError(callLogInfo, errStatus)
		return irs.VMInfo{}, errStatus
	}
	cblogger.Infof("==> VM status of [%s] : [%s]", newVMIID.NameId, curStatus)

	// Check VM Deploy Status
	vmResult, err := servers.Get(vmHandler.VMClient, vm.ID).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Start VM. Failed to Get VMInfo, err : %v", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	if strings.ToLower(vmResult.Status) == "active" {
		var privateIP string
		for _, subnet := range vmResult.Addresses {
			// Get PrivateIP Info
			for _, addr := range subnet.([]interface{}) {
				addrMap := addr.(map[string]interface{})
				if addrMap["OS-EXT-IPS:type"] == "fixed" {
					privateIP = addrMap["addr"].(string)
				}
			}
		}
		cblogger.Infof("# PrivateIP of New VM : [%s]", privateIP)

		cblogger.Info("# Start to Create New Public IP!!")
		var publicIPId string
		var creatErr error
		var ok bool
		if ok, publicIPId, creatErr = vmHandler.createPublicIP(); !ok {
			newErr := fmt.Errorf("Failed to Create a PublicIP : [%v]", creatErr)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
		cblogger.Infof("# New PublicIP ID : [%s]", publicIPId)
		time.Sleep(time.Second * 2)

		publicIp, err := vmHandler.FindPublicIPByID(publicIPId)
		if err != nil {
			newErr := fmt.Errorf("Failed to Find the PublicIP with the ID : [%v]", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}		
		
		var sgSystemIDs []string
		var keyValueList []irs.KeyValue
		for _, sgIID := range vmReqInfo.SecurityGroupIIDs {
			cblogger.Infof("S/G ID : [%s]", sgIID.SystemId)

			// To Create PortForwarding and Firewall Rules
			sgSystemIDs = append(sgSystemIDs, sgIID.SystemId)

			// To Register SecurityGroupInfo to DB
			keyValueList = append(keyValueList, irs.KeyValue{
				Key:   sgIID.SystemId,
				Value: sgIID.SystemId,
			})
		}
		cblogger.Infof("The SystemIds of the Security Group IIDs : [%s]", sgSystemIDs)

		// Register SecurityGroupInfo to DB
		providerName := "KTVPC"
		sgInfo, regErr := sim.RegisterSecurityGroup(vm.ID, providerName, keyValueList)
		if regErr != nil {
			cblogger.Error(regErr)
			return irs.VMInfo{}, regErr
		}
		cblogger.Infof(" === S/G Info to Register to DB : [%v]", sgInfo)

		// // # Get Tier NameId
		// var tierId string
		// for key, _ := range vmResult.Addresses {
		// 	tierId = key
		// }

		vpcHandler := KTVpcVPCHandler{
			RegionInfo:    vmHandler.RegionInfo,
			NetworkClient: vmHandler.NetworkClient, // Required!!
		}
		tierNetworkId, err := vpcHandler.getNetworkIdWithTierId(vmReqInfo.SubnetIID.SystemId)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Network ID!!")
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
		cblogger.Infof("# Subnet(Tier) NetworkId : %s", *tierNetworkId)

		// Create PortForwarding and Firewall Rules
		secRuleSet := SecurityRuleSet {
			TierNetworkId:  		*tierNetworkId,
			SecurityGroupSystemIDs: sgSystemIDs,
			PrivateIP:				privateIP,
			PublicIP:				publicIp,
			PublicIPId:				publicIPId,
		}
		if ok, err := vmHandler.createPortForwardingFirewallRules(&secRuleSet); !ok {
			newErr := fmt.Errorf("Failed to Create PortForwarding and Firewall Rules : [%v]", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}

		// Get vm info
		vmResult, err = servers.Get(vmHandler.VMClient, vm.ID).Extract()
		if err != nil {
			newErr := fmt.Errorf("Failed to Get New VM Info from KT Cloud VPC. %s", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}

		vmInfo, err := vmHandler.mappingVMInfo(*vmResult)
		if err != nil {
			newErr := fmt.Errorf("Failed to Map New VM Info. %s", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
		// vmInfo.SecurityGroupIIds = sgIIDs
		return vmInfo, nil
	}
	return irs.VMInfo{}, nil
}

func (vmHandler *KTVpcVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetVM()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.SystemId, "GetVM()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	start := call.Start()
	vmResult, err := servers.Get(vmHandler.VMClient, vmIID.SystemId).Extract()
	if err != nil {
		cblogger.Error(err.Error())
		loggingError(callLogInfo, err)
		return irs.VMInfo{}, err
	}
	loggingInfo(callLogInfo, start)

	vmInfo, err := vmHandler.mappingVMInfo(*vmResult)
	if err != nil {
		cblogger.Error(err.Error())
		loggingError(callLogInfo, err)
		return irs.VMInfo{}, err
	}
	return vmInfo, nil
}

func (vmHandler *KTVpcVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("KT Cloud VPC Driver: called SuspendVM()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.SystemId, "SuspendVM()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.Failed, newErr
	}

	var resultStatus string
	cblogger.Info("Start Get VM Status...")
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		cblogger.Errorf("[%s] Failed to Get the VM Status of VM : ", vmIID.SystemId)
		cblogger.Error(err)
		loggingError(callLogInfo, err)
		return irs.VMStatus("Failed to Get the VM Status of VM. "), err
	} else {
		cblogger.Infof("Succeeded in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, vmStatus)
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
		err := startstop.Stop(vmHandler.VMClient, vmIID.SystemId).Err
		if err != nil {
			cblogger.Error(err.Error())
			loggingError(callLogInfo, err)
			return irs.Failed, err
		}
		loggingInfo(callLogInfo, start)
	}

	// Return of the progress status (KT VPC is not provided with information about in progress)
	return irs.Suspending, nil
}

func (vmHandler *KTVpcVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("KT Cloud VPC Driver: called ResumeVM()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "ResumeVM()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.Failed, newErr
	}

	cblogger.Info("Start Get VM Status...")
	var resultStatus string
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		cblogger.Errorf("Failed to Get the VM Status of : [%s]", vmIID.SystemId)
		cblogger.Error(err)
		loggingError(callLogInfo, err)
		return irs.VMStatus("Failed. "), err
	} else {
		cblogger.Infof("Succeeded in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, vmStatus)
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
		err := startstop.Start(vmHandler.VMClient, vmIID.SystemId).Err
		if err != nil {
			cblogger.Error(err.Error())
			loggingError(callLogInfo, err)
			return irs.Failed, err
		}
		loggingInfo(callLogInfo, start)

		// Return of the progress status (KT VPC is not provided with information about in progress)
		return irs.Resuming, nil
	}
}

func (vmHandler *KTVpcVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("KT Cloud VPC Driver: called RebootVM()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.SystemId, "RebootVM()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.Failed, newErr
	}

	cblogger.Info("Start Get VM Status...")
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		cblogger.Errorf("[%s] Failed to Get the VM Status.", vmIID)
		cblogger.Error(err)
		loggingError(callLogInfo, err)
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

		err := servers.Reboot(vmHandler.VMClient, vmIID.SystemId, rebootOpts).ExtractErr()
		if err != nil {
			cblogger.Error(err.Error())
			loggingError(callLogInfo, err)
			return irs.Failed, err
		}
		loggingInfo(callLogInfo, start)

		// Return of the progress status (KT VPC is not provided with information about in progress)
		return irs.Rebooting, nil
	}
}

func (vmHandler *KTVpcVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("KT Cloud VPC Driver: called TerminateVM()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.SystemId, "TerminateVM()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.Failed, newErr
	}

	vm, err := vmHandler.GetVM(vmIID)
	if err != nil {
		cblogger.Error(err.Error())
		loggingError(callLogInfo, err)
		return irs.Failed, err
	}

	if !strings.EqualFold(vm.PublicIP, "") {
		// Delete Firewall Rules
		if !strings.EqualFold(vm.PublicIP, "") {
			_, dellFwErr := vmHandler.removeFirewallRules(vm.PublicIP, vm.PrivateIP)
			if dellFwErr != nil {
				cblogger.Error(dellFwErr.Error())
				loggingError(callLogInfo, dellFwErr)
				return irs.Failed, dellFwErr
			}
		}

		// Delete Port Forwarding Rules
		if !strings.EqualFold(vm.PublicIP, "") {
			_, dellPfErr := vmHandler.removePortForwardingRules(vm.PublicIP)
			if dellPfErr != nil {
				cblogger.Error(dellPfErr.Error())
				loggingError(callLogInfo, dellPfErr)
				return irs.Failed, dellPfErr
			}
		}

		// Delete PublicIP connected VM
		if !strings.EqualFold(vm.PublicIP, "") {
			_, dellIpErr := vmHandler.removePublicIP(vm.PublicIP)
			if dellIpErr != nil {
				cblogger.Error(dellIpErr.Error())
				loggingError(callLogInfo, dellIpErr)
				return irs.Failed, dellIpErr
			}
		}
	} else {
		cblogger.Info("The VM doesn't have any connected Pulbic IP!! Waitting for Termination!!")
	}

	// Terminate VM
	start := call.Start()
	err = servers.Delete(vmHandler.VMClient, vm.IId.SystemId).ExtractErr()
	if err != nil {
		cblogger.Error(err.Error())
		loggingError(callLogInfo, err)
		return irs.Failed, err
	}
	loggingInfo(callLogInfo, start)

	// Delete the S/G info from DB
	_, unRegErr := sim.UnRegisterSecurityGroup(vm.IId.SystemId)
	if unRegErr != nil {
		cblogger.Debug(unRegErr.Error())
		loggingError(callLogInfo, unRegErr)
		// return irs.Failed, unRegErr
	}

	// Return of the progress status (KT VPC is not provided with information about in progress)
	return irs.Terminating, nil
}

func (vmHandler *KTVpcVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetVMStatus()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.SystemId, "GetVMStatus()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.Failed, newErr
	}

	start := call.Start()
	vmResult, err := servers.Get(vmHandler.VMClient, vmIID.SystemId).Extract()
	if err != nil {
		cblogger.Debug(err.Error()) // For after termination
		loggingError(callLogInfo, err)
		return "", err
	}
	loggingInfo(callLogInfo, start)

	// cblogger.Infof("# vmResult.Status of KT Cloud VPC : [%s]", vmResult.Status)
	vmStatus := getVmStatus(vmResult.Status)
	return vmStatus, nil
}

func (vmHandler *KTVpcVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListVMStatus()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "ListVMStatus()", "ListVMStatus()")

	start := call.Start()
	pager, err := servers.List(vmHandler.VMClient, nil).AllPages()
	if err != nil {
		cblogger.Error(err.Error())
		loggingError(callLogInfo, err)
		return nil, err
	}
	loggingInfo(callLogInfo, start)

	vms, err := servers.ExtractServers(pager)
	if err != nil {
		cblogger.Error(err.Error())
		loggingError(callLogInfo, err)
		return nil, err
	}

	// Add to List
	vmStatusList := make([]*irs.VMStatusInfo, len(vms))
	for idx, s := range vms {
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

func (vmHandler *KTVpcVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListVM()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "ListVM()", "ListVM()")

	// Get VM list
	listOpts := servers.ListOpts{
		Limit: 100,
	}
	start := call.Start()
	pager, err := servers.List(vmHandler.VMClient, listOpts).AllPages()
	if err != nil {
		cblogger.Error(err.Error())
		loggingError(callLogInfo, err)
		return nil, err
	}
	loggingInfo(callLogInfo, start)

	vmList, err := servers.ExtractServers(pager)
	if err != nil {
		cblogger.Error(err.Error())
		loggingError(callLogInfo, err)
		return nil, err
	}

	// Mapping VM info list
	var vmInfoList []*irs.VMInfo
	for _, vm := range vmList {
		vmInfo, err := vmHandler.mappingVMInfo(vm)
		if err != nil {
			cblogger.Error(err.Error())
			loggingError(callLogInfo, err)
			return nil, err
		}
		vmInfoList = append(vmInfoList, &vmInfo)
	}
	return vmInfoList, nil
}

func (vmHandler *KTVpcVMHandler) createPublicIP() (bool, string, error) {
	cblogger.Info("KT Cloud VPC Driver: called createPublicIP()")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "createPublicIP()", "createPublicIP()")

	cblogger.Info("### Creating New Public IP!!")
	createOpts := ips.CreateOpts{}
	start := call.Start()
	result, err := ips.Create(vmHandler.NetworkClient, createOpts).ExtractCreate()
	if err != nil {
		if !strings.Contains(err.Error(), ":true") {
			newErr := fmt.Errorf("Failed to create Public IP: %v", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return false, "", newErr
		}
	} else if strings.EqualFold(result.Data.PublicIpID, "") {
		newErr := fmt.Errorf("Failed to create Public IP")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, "", newErr
	}
	loggingInfo(callLogInfo, start)

	return true, result.Data.PublicIpID, nil
}

type SecurityRuleSet struct {
	TierNetworkId 			string
	SecurityGroupSystemIDs 	[]string
	PrivateIP				string
	PublicIP				string
	PublicIPId				string
}

// ### To Apply 'PortForwarding Rules' and 'Firewall Rules' to the PublicIP ID.
func (vmHandler *KTVpcVMHandler) createPortForwardingFirewallRules(ruleSet *SecurityRuleSet) (bool, error) {
	cblogger.Info("KT Cloud driver: called createPortForwardingFirewallRules()!")

	if strings.EqualFold(ruleSet.TierNetworkId, "") {
		newErr := fmt.Errorf("Invalid Tier Network ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	if strings.EqualFold(ruleSet.PrivateIP, "") {
		newErr := fmt.Errorf("Invalid Private IP!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	if strings.EqualFold(ruleSet.PublicIP, "") {
		newErr := fmt.Errorf("Invalid Public IP!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	if strings.EqualFold(ruleSet.PublicIPId, "") {
		newErr := fmt.Errorf("Invalid Public IP ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	extNetId, getErr := vmHandler.getNetworkID("external")
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Get External VlanName : [%v]", getErr)
		cblogger.Error(newErr.Error())
		return false, newErr
	} else {
		cblogger.Infof("# External VlanName : %s", extNetId)
	}

	for _, sgSystemID := range ruleSet.SecurityGroupSystemIDs {
		cblogger.Infof("S/G System ID : [%s]", sgSystemID)

		securityHandler := KTVpcSecurityHandler{
			RegionInfo: 	vmHandler.RegionInfo,
			VMClient:       vmHandler.VMClient,
			NetworkClient: 	vmHandler.NetworkClient,
		}
		sgInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: sgSystemID})
		if err != nil {
			cblogger.Errorf("Failed to Find the SecurityGroup : %s", sgSystemID)
			return false, err
		}

		var protocols []string
		for _, sgRule := range *sgInfo.SecurityRules {
			if strings.EqualFold(sgRule.IPProtocol, "tcp") { // case insensitive comparing and returns true.
				protocols = []string{"TCP"}
			} else if strings.EqualFold(sgRule.IPProtocol, "udp") {
				protocols = []string{"UDP"}
			} else if strings.EqualFold(sgRule.IPProtocol, "icmp") {
				protocols = []string{"ICMP"}
			} else if strings.EqualFold(sgRule.IPProtocol, "ALL") {
				protocols = []string{"TCP", "UDP", "ICMP"}
			} else {
				cblogger.Errorf("Failed to Find mapping Protocol matching with the given Protocol [%s].", sgRule.IPProtocol)
				return false, errors.New("Failed to Find mapping Protocol matching with the given Protocol." + sgRule.IPProtocol)
			}

			for _, curProtocol := range protocols {
				cblogger.Infof("\n")
				cblogger.Infof("### Current Protocol : [%s]", curProtocol)
				// When the request port number is '-1', all ports are opened.
				if (sgRule.FromPort == "-1") && (sgRule.ToPort == "-1") {
					sgRule.FromPort = "1"
					sgRule.ToPort = "65535"
				}

				// It's impossible to input any port number when the curProtocol is ICMP on KT Cloud firewall.
				// Caution!!) KT Cloud does Not support 'ICMP' protocol for PortForwarding Rule.
				if curProtocol == "ICMP" {
					sgRule.FromPort = ""
					sgRule.ToPort = ""
				}
				
				pfRuleId := "" // Init PortForwarding Rule ID
				if strings.EqualFold(sgRule.Direction, "inbound") {
					if !(strings.EqualFold(curProtocol, "ICMP")) {
						cblogger.Info("### Start to Create PortForwarding Rules!!")

						// ### Set Port Forwarding Rules
						createPfOpts := &portforward.CreateOpts{
							PublicIpID:        	ruleSet.PublicIPId,			// Required
							MappedIP:     	   	ruleSet.PrivateIP,			// Required
							Protocol:          	curProtocol,		// Required
							StartPrivatePort:  	sgRule.FromPort,	// Required
							EndPrivatePort:   	sgRule.ToPort,		// Required
							StartPublicPort:    sgRule.FromPort,	// Required
							EndPublicPort:   	sgRule.ToPort,		// Required
						}
						// cblogger.Info("\n ### createPfOpts : ")
						// spew.Dump(createPfOpts)
						// cblogger.Info("\n")

						pfResult := portforward.Create(vmHandler.NetworkClient, createPfOpts) // NetworkClient!!. and, Not ~.Extract()
						if pfResult.Err != nil {
							cblogger.Errorf("Failed to Create the Port Forwarding Rule : [%v]", pfResult.Err)
							return false, pfResult.Err
						}

						// Extract the created PortForwarding ID
						portForwardingId, err := portforward.ExtractPortForwardingID(pfResult)
						if err != nil {
							fmt.Printf("Failed to extract portForwardingId: %v\n", err)
						} else {
							fmt.Printf("Created portForwardingId: %s\n", portForwardingId)
							pfRuleId = portForwardingId
						}

						cblogger.Info("### Waiting for PortForwarding Rules to be Created!!")
						// To prevent - json: cannot unmarshal string into Go struct field AsyncJobResult.nc_queryasyncjobresultresponse.result of type job.JobResult
						time.Sleep(time.Second * 2)
					}

					// Caution!!) KT Cloud VPC 'Firewall Rules' Support "inbound" and "outbound"
					// ### Set FireWall Rules (In case of "Inbound" FireWall Rules)
					cblogger.Info("### Start to Create Firewall 'inbound' Rules!!")

					destCIDR, err := ipToCidr32(ruleSet.PublicIP) // Output format ex) "172.25.1.0/32". Not Private IP to login from External
					if err != nil {
						cblogger.Errorf("Failed to Get Dest Net Band : [%v]", err)
						return false, err
					} else {
						cblogger.Infof("Dest CIDR : %s", destCIDR)
					}

					srcIPAdds := "0.0.0.0/0"
					inboundFWOpts := &rules.CreateOpts{}
					if !(strings.EqualFold(curProtocol, "ICMP")) { // Incase of 'TCP' and 'UDP'
						comment := "Allow inbound - " + curProtocol + " - " + sgRule.FromPort + " to " + sgRule.ToPort
						inboundFWOpts = &rules.CreateOpts{
							Action:           	true, // accept
							Protocol:      		curProtocol,
							StartPort:     		sgRule.FromPort,
							EndPort:       		sgRule.ToPort,
							SrcNetwork:       	[]string{extNetId},
							PortForwardingId: 	pfRuleId,							
							SrcAddress:       	[]string{srcIPAdds},
        					DstAddress:       	[]string{destCIDR}, // Cannot be entered simultaneously with portForwardingId
							Comment:          	comment,
							SrcNat:           	false,
						}
					} else { // Incase of 'ICMP'. Note) 'ICMP' does not have Port Forwarding rule and 'its ID'
						comment := "Allow inbound - " + curProtocol
						inboundFWOpts = &rules.CreateOpts{
							Action:           	true, // accept
							Protocol:      		curProtocol,
							// StartPort:     		sgRule.FromPort,
							// EndPort:       		sgRule.ToPort,
							SrcNetwork:       	[]string{extNetId},
							DstNetwork:       	[]string{ruleSet.TierNetworkId},
							SrcAddress:       	[]string{srcIPAdds},
        					DstAddress:       	[]string{destCIDR}, // Cannot be entered simultaneously with portForwardingId
							Comment:          	comment,
							SrcNat:           	false,
						}
					}
					// cblogger.Info("\n# Inbound FireWall Options : ")
					// spew.Dump(inboundFWOpts)
					// cblogger.Info("\n")


					// fwRuleJobId := ""
					fwResult := rules.Create(vmHandler.NetworkClient, inboundFWOpts)
					if fwResult.Err != nil {
						newErr := fmt.Errorf("Failed to Create the FireWall 'inbound' Rule : %v", fwResult.Err)
						cblogger.Error(newErr.Error())
						return false, newErr
					} else {
						jobId, err := rules.ExtractJobID(fwResult)
						if err != nil {
							fmt.Printf("Failed to Extract the JobId: %v\n", err)
						} else {
							fmt.Printf("Created F/W rule JobId: %s\n", jobId)
							// fwRuleJobId = jobId
						}
					}

					cblogger.Info("### Waiting for FireWall 'inbound' Rules to be Created(600sec) !!")
					// $$$ To prevent - json: cannot unmarshal string into Go struct field AsyncJobResult.nc_queryasyncjobresultresponse.result of type job.JobResult
					time.Sleep(time.Second * 2)

					// jobWaitErr := vmHandler.waitForAsyncJob(fwRuleJobId, 600000000000)
					// if jobWaitErr != nil {
					// 	cblogger.Errorf("Failed to Wait the Job : [%v]", jobWaitErr)
					// 	return false, jobWaitErr
					// }
				}

				// ### In case of "Outbound" FireWall Rules
				if strings.EqualFold(sgRule.Direction, "outbound") {
					cblogger.Info("### Start to Create Firewall 'outbound' Rules!!")

					srcCIDR, err := ipToCidr32(ruleSet.PrivateIP) // Output format ex) "172.25.1.5/32",  ipToCidr24() : Output format ex) "172.25.1.0/24"
					if err != nil {
						cblogger.Errorf("Failed to Get Source Net Band : [%v]", err)
						return false, err
					} else {
						cblogger.Infof("Source CIDR : %s", srcCIDR)
					}

					destIPAdds := "0.0.0.0/0"
					comment := "Allow outbound - " + curProtocol + " - " + sgRule.FromPort + " to " + sgRule.ToPort
					outboundFWOpts := &rules.CreateOpts{
							Action:           	true, // accept
							Protocol:      		curProtocol,
							StartPort:     		sgRule.FromPort,
							EndPort:       		sgRule.ToPort,
							SrcNetwork:       	[]string{ruleSet.TierNetworkId},
							DstNetwork: 	  	[]string{extNetId},
							SrcAddress:       	[]string{srcCIDR},
        					DstAddress:       	[]string{destIPAdds}, // Cannot be entered simultaneously with portForwardingId
							Comment:          	comment,
							SrcNat:           	false,
					}
					// cblogger.Info("\n# Outbound FireWall Options : ")
					// spew.Dump(outboundFWOpts)
					// cblogger.Info("\n")
					
					// fwRuleJobId := ""
					fwResult := rules.Create(vmHandler.NetworkClient, outboundFWOpts)
					if fwResult.Err != nil {
						newErr := fmt.Errorf("Failed to Create the FireWall 'inbound' Rule : %v", fwResult.Err)
						cblogger.Error(newErr.Error())
						return false, newErr
					} else {
						jobId, err := rules.ExtractJobID(fwResult)
						if err != nil {
							cblogger.Infof("Failed to Extract the JobId: %v", err)
						} else {
							cblogger.Infof("Created F/W rule JobId: %s", jobId)
							// fwRuleJobId = jobId
						}
					}

					cblogger.Info("### Waiting for FireWall 'outbound' Rules to be Created!!")
					// $$$ To prevent - json: cannot unmarshal string into Go struct field AsyncJobResult.nc_queryasyncjobresultresponse.result of type job.JobResult
					time.Sleep(time.Second * 2)

					// jobWaitErr := vmHandler.waitForAsyncJob(fwRuleJobId, 600000000000)
					// if jobWaitErr != nil {
					// 	cblogger.Errorf("Failed to Wait the Job : [%v]", jobWaitErr)
					// 	return false, jobWaitErr
					// }
				}
			}
		}
	}
	return true, nil
}

func getVmStatus(vmStatus string) irs.VMStatus {
	cblogger.Info("KT Cloud VPC Driver: called getVmStatus()")

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

func (vmHandler *KTVpcVMHandler) mappingVMInfo(vm servers.Server) (irs.VMInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called mappingVMInfo()")
	// cblogger.Info("\n# VM from KT Cloud VPC :")
	// spew.Dump(vm)

	convertedTime, err := convertTimeToKTC(vm.Created)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Converted Time. [%v]", err)
		return irs.VMInfo{}, newErr
	}

	vmInfo := irs.VMInfo{
		IId: irs.IID{
			NameId:   vm.Name,
			SystemId: vm.ID,
		},
		Region: irs.RegionInfo{
			// Region: "N/A",
			Zone: vmHandler.RegionInfo.Zone,
		},
		KeyPairIId: irs.IID{
			NameId:   vm.KeyName,
			SystemId: vm.KeyName,
		},
		VMUserId: LnxUserName,
		// VMUserPasswd:      "N/A",

		KeyValueList: irs.StructToKeyValueList(vm),
	}
	vmInfo.StartTime = convertedTime
	vmInfo.VMSpecName = vm.Flavor["original_name"].(string)

	// Get SecurityGroupInfo from DB
	sgInfo, getSGErr := sim.GetSecurityGroup(vm.ID)
	if getSGErr != nil {
		cblogger.Debug(getSGErr)
		// return irs.VMInfo{}, getSGErr
	}
	if countSgKvList(*sgInfo) > 0 {
		// Since S/G is managed as a file, the systemID is the same as the name ID.
		var sgIIDs []irs.IID
		for _, kv := range sgInfo.KeyValueInfoList {
			sgIIDs = append(sgIIDs, irs.IID{NameId: kv.Key, SystemId: kv.Value})
		}
		vmInfo.SecurityGroupIIds = sgIIDs
	}

	// # Get Network Info
	for key, subnet := range vm.Addresses {
		// VPC Info
		vmInfo.SubnetIID.NameId = key
		// Get PrivateIP Info
		for _, addr := range subnet.([]interface{}) {
			addrMap := addr.(map[string]interface{})
			if addrMap["OS-EXT-IPS:type"] == "fixed" {
				vmInfo.PrivateIP = addrMap["addr"].(string)
			}
		}
	}

	netInfo, err := vmHandler.getNetIDsWithPrivateIP(vmInfo.PrivateIP)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get PortForwarding Info. [%v]", err)
		cblogger.Debug(newErr.Error())
		// return irs.VMInfo{}, nil
		// return irs.VMInfo{}, newErr
	}

	// cblogger.Info("\n\n### netInfo : ")
	// spew.Dump(netInfo)
	// cblogger.Info("\n")

	if (netInfo != nil) && !strings.EqualFold(netInfo.PublicIP, "") {
		vmInfo.PublicIP = netInfo.PublicIP
	}

	vpcHandler := KTVpcVPCHandler{
		RegionInfo:    vmHandler.RegionInfo,
		NetworkClient: vmHandler.NetworkClient, // Required!!
	}
	// OsNetId, getError := vpcHandler.getOsNetworkIdWithTierId(netInfo.VpcID, netInfo.SubnetID)
	// if getError != nil {
	// 	newErr := fmt.Errorf("Failed to Get the OsNetwork ID of the Tier : [%v]", getError)
	// 	cblogger.Error(newErr.Error())
	// 	return irs.VMInfo{}, newErr
	// } else {
	// 	cblogger.Infof("# OsNetwork ID : %s", OsNetId)
	// }

	tierId, getNetErr := vpcHandler.getTierIdWithTierName(vmInfo.SubnetIID.NameId)
	if getNetErr != nil {
		newErr := fmt.Errorf("Failed to Get the OsNetwork ID with the Tier Name : [%v]", getNetErr)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}
	if tierId != nil {
		vmInfo.SubnetIID.SystemId = *tierId // Caution!!) Not Tier 'NetworkId' but 'TierId' to Create VM through REST API!!
	}

	vpcId, err := vpcHandler.getVPCIdWithTierId(*tierId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VPC ID with teh OsNetwork ID. [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}
	if vpcId != nil {
		vmInfo.VpcIID.SystemId = *vpcId
	}

	// # Get ImageInfo frome the Disk Volume
	diskHandler := KTVpcDiskHandler{
		RegionInfo:   vmHandler.RegionInfo,
		VMClient:     vmHandler.VMClient,
		VolumeClient: vmHandler.VolumeClient,
	}

	var diskIIDs []irs.IID
	var imageIID irs.IID
	var getErr error
	if len(vm.AttachedVolumes) > 0 {
		for _, volume := range vm.AttachedVolumes {
			cblogger.Infof("# Volume ID : %s", volume.ID)

			// ktVolume, _ := volumes3.Get(vmHandler.VolumeClient, volume.ID).Extract()
			ktVolume, err := volumes2.Get(vmHandler.VolumeClient, volume.ID).Extract()
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the KT Disk Info!! : [%v] ", err)
				cblogger.Error(newErr.Error())
				return irs.VMInfo{}, newErr
			}

			if ktVolume.Bootable == "true" {
				vmInfo.RootDiskType = ktVolume.VolumeType
				vmInfo.RootDiskSize = strconv.Itoa(ktVolume.Size)
				vmInfo.RootDeviceName = ktVolume.Attachments[0].Device

				imageIID, getErr = diskHandler.getImageNameandIDWithDiskID(volume.ID)
				if getErr != nil {
					cblogger.Infof("Failed to Get Image Info from the Disk Info : [%v]", getErr)
					// return irs.VMInfo{}, err
				}
			} else {
				diskIIDs = append(diskIIDs, irs.IID{SystemId: volume.ID}) // Data Disk. (Not bootable)
			}
		}
	}
	vmInfo.DataDiskIIDs = diskIIDs

	// Set the VM Image Info
	imageHandler := KTVpcImageHandler{
		RegionInfo:  vmHandler.RegionInfo,
		VMClient:    vmHandler.VMClient,
		ImageClient: vmHandler.ImageClient,
	}
	if !strings.EqualFold(imageIID.SystemId, "") {
		vmInfo.ImageIId.NameId = imageIID.NameId
		vmInfo.ImageIId.SystemId = imageIID.SystemId

		isPublicImage, err := imageHandler.isPublicImage(irs.IID{SystemId: imageIID.SystemId})
		if err != nil {
			newErr := fmt.Errorf("Failed to Check Whether the Image is Public Image : [%v]", err)
			cblogger.Error(newErr.Error())
			// return irs.VMInfo{}, newErr // Caution!!
		}
		if isPublicImage {
			vmInfo.ImageType = irs.PublicImage
		} else {
			vmInfo.ImageType = irs.MyImage
		}
	}

	if !strings.EqualFold(imageIID.NameId, "") {
		if strings.Contains(strings.ToLower(imageIID.NameId), "windows") {
			vmInfo.Platform = irs.WINDOWS
			vmInfo.VMUserId = WinUserName
			vmInfo.VMUserPasswd = "User Specified Passwd"
			if (netInfo != nil) && !strings.EqualFold(netInfo.PublicIP, "") {
				vmInfo.SSHAccessPoint = netInfo.PublicIP + ":3389"
			}
		} else {
			vmInfo.Platform = irs.LINUX_UNIX
			vmInfo.VMUserId = LnxUserName
			vmInfo.RootDeviceName = "/dev/xvda"
			// vmInfo.VMUserPasswd		= "N/A"
			if (netInfo != nil) && !strings.EqualFold(netInfo.PublicIP, "") {
				vmInfo.SSHAccessPoint = netInfo.PublicIP + ":22"
			}
		}
	}

	float64Vcpus := vm.Flavor["vcpus"].(float64)
	float64Ram := vm.Flavor["ram"].(float64)

	var vCPU string
	var vRAM string
	if float64Vcpus != 0 {
		vCPU = strconv.FormatFloat(vm.Flavor["vcpus"].(float64), 'f', -1, 64)
	}
	if float64Ram != 0 {
		vRAM = strconv.FormatFloat(vm.Flavor["ram"].(float64), 'f', -1, 64)
	}

	//# Append to KeyValueList
	var keyValueList []irs.KeyValue
	if vCPU != "" {
		keyValue := irs.KeyValue{Key: "vCPU", Value: vCPU}
		keyValueList = append(keyValueList, keyValue)
	}
	if vRAM != "" {
		keyValue := irs.KeyValue{Key: "vRAM(GB)", Value: vRAM}
		keyValueList = append(keyValueList, keyValue)
	}
	if vm.Status != "" {
		keyValue := irs.KeyValue{Key: "VM_Status", Value: string(getVmStatus(vm.Status))}
		keyValueList = append(keyValueList, keyValue)
	}
	if (netInfo != nil) && !strings.EqualFold(netInfo.PublicIP, "") {
		keyValue := irs.KeyValue{Key: "PublicIPID", Value: netInfo.PublicIPID}
		keyValueList = append(keyValueList, keyValue)
	}
	vmInfo.KeyValueList = append(vmInfo.KeyValueList, keyValueList...)

	return vmInfo, nil
}

// Waiting for up to 500 seconds during VM creation until VM info. can be get
func (vmHandler *KTVpcVMHandler) waitToGetVMInfo(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("KT Cloud VPC Driver: called waitToGetVMInfo()")

	cblogger.Info("======> As VM info. cannot be retrieved immediately after VM creation, it waits until running.")
	curRetryCnt := 0
	maxRetryCnt := 500

	for {
		curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
		if errStatus != nil {
			cblogger.Errorf("Failed to Get the VM Status of : [%s]", vmIID)
			cblogger.Error(errStatus.Error())
			return irs.VMStatus("Failed. "), errors.New("Failed to Get the VM Status.")
		} else {
			// cblogger.Infof("Succeeded in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, curStatus)
			cblogger.Info("===> VM Status : ", curStatus)
		}

		switch string(curStatus) {
		case "Creating", "Booting":
			curRetryCnt++
			cblogger.Infof("The VM is still 'Creating', so wait for a second more before inquiring the VM info.")
			time.Sleep(time.Second * 5)
			if curRetryCnt > maxRetryCnt {
				cblogger.Errorf("Despite waiting for a long time(%d sec), the VM status is %s, so it is forcibly finishied.", maxRetryCnt, curStatus)
				return irs.VMStatus("Failed. "), errors.New("Despite waiting for a long time, the VM status is 'Creating', so it is forcibly finishied.")
			}

		default:
			cblogger.Infof("===> ### The VM Creation is finished, stopping the waiting.")
			return irs.VMStatus(curStatus), nil
			//break
		}
	}
}

func (vmHandler *KTVpcVMHandler) vmExists(vmIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called vmExists()")

	if strings.EqualFold(vmIID.NameId, "") {
		newErr := fmt.Errorf("Invalid VM Name!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	allPagers, err := servers.List(vmHandler.VMClient, servers.ListOpts{Name: vmIID.NameId}).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VM Pages from KT Cloud. : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	vmList, err := servers.ExtractServers(allPagers)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VM list. : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	for _, vm := range vmList {
		if strings.EqualFold(vm.Name, vmIID.NameId) {
			return true, nil
		}
	}

	return false, nil
}

func (vmHandler *KTVpcVMHandler) imageExists(imageIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called imageExists()")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Image System ID!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	imageHandler := KTVpcImageHandler{
		RegionInfo:  vmHandler.RegionInfo,
		VMClient:    vmHandler.VMClient,
		ImageClient: vmHandler.ImageClient,
	}
	listOpts := images.ListOpts{
		Limit:      300,                          //default : 20
		Visibility: images.ImageVisibilityPublic, // Note : Public image only
	}
	allPages, err := images.List(imageHandler.ImageClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VPC Image Pages. [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	imageList, err := images.ExtractImages(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VPC Image List. [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	for _, image := range imageList {
		if strings.EqualFold(image.ID, imageIID.SystemId) {
			return true, nil
		}
	}
	return false, nil
}

func (vmHandler *KTVpcVMHandler) listFirewallRule() ([]rules.FirewallRule, error) {
	cblogger.Info("KT Cloud VPC Driver: called listFirewallRule()!")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Zone, call.VPCSUBNET, "listFirewallRule()", "listFirewallRule()")

	var firewallRuleList []rules.FirewallRule
	// ### If enter a different number to ListOpts, the value will not be retrieved correctly.
    listOpts := rules.ListOpts{
        Page: 1,
        Size: 20,    
	}
	pager := rules.List(vmHandler.NetworkClient, listOpts) // Caution!!) Not VMClient but NetworkClient
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
        rules, err := rules.ExtractRules(page)
        if err != nil {
			newErr := fmt.Errorf("Failed to Extract FirewallRule list : [%v]", err)
			cblogger.Error(newErr.Error())
		    return false, newErr
		}

		if len(rules) < 1 {
			newErr := fmt.Errorf("Failed to Get Any FirewallRule Info.")
			cblogger.Debug("No FirewallRule found : %v", newErr)
			return false, newErr
		}

		firewallRuleList = rules    
 		return true, nil
	})
    if err != nil {
		newErr := fmt.Errorf("Failed to Get FirewallRule list : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
    }

	return firewallRuleList, nil
}

func (vmHandler *KTVpcVMHandler) listPortForwarding() ([]portforward.PortForwarding, error) {
	cblogger.Info("KT Cloud VPC Driver: called listPortForwarding()!")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Zone, call.VPCSUBNET, "listPortForwarding()", "listPortForwarding()")

	var pfRuls []portforward.PortForwarding
	var extErr error

	// ### If enter a different number to ListOpts, the value will not be retrieved correctly.
	listOpts := portforward.ListOpts{
		Page: 1,
		Size: 20,
	}
	pager := portforward.List(vmHandler.NetworkClient, listOpts)
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		pfRuls, extErr = portforward.ExtractPFs(page)
		if extErr != nil {
			newErr := fmt.Errorf("Failed to Extract Port Forwarding list : [%v]", extErr)
			cblogger.Error(newErr.Error())
			return false, newErr
		}
		return true, nil
	})
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Port Forwarding list : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	if len(pfRuls) < 1 {
		newErr := fmt.Errorf("Failed to Get Any Port Forwarding Info.")
		cblogger.Debug("No Port Forwarding found : %v", newErr)
		return nil, nil
		// return nil, newErr
	}

	return pfRuls, nil
}

/*
type NetworkInfo struct {
	VpcID  			string
	SubnetID		string
	PublicIP 		string
	PublicIPID		string
}
*/

func (vmHandler *KTVpcVMHandler) getNetIDsWithPrivateIP(privateIpAddr string) (*NetworkInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called getNetIDsWithPrivateIP()!")

	if strings.EqualFold(privateIpAddr, "") {
		newErr := fmt.Errorf("Invalid Private IP Address!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	pfRuleList, err := vmHandler.listPortForwarding()
	if err != nil {
		if strings.Contains(err.Error(), "Failed to Get Any Port Forwarding Info") {
			cblogger.Debug(err.Error())
			return nil, nil
		}
		newErr := fmt.Errorf("Failed to Get PortForwarding Rule List. [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	if len(pfRuleList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any PortForwarding Rule : [%v]", err)
		cblogger.Debug(newErr.Error())
		return nil, nil
		// return nil, newErr
	}

	netInfo := NetworkInfo{}
	for _, rule := range pfRuleList {
		if strings.EqualFold(rule.MappedIP, privateIpAddr) {
			// cblogger.Infof("\n# PortForwardingName : [%s]", rule.PortForwardingName)
			netInfo.PublicIP 	= rule.PublicIP
			netInfo.PublicIPID 	= rule.PublicIPID
			break
		}
	}

	if strings.EqualFold(netInfo.PublicIP, "") {
		newErr := fmt.Errorf("Failed to Find any PublicIP Address with the Private IP!!")
		cblogger.Debug(newErr.Error())
		// return nil, newErr
	}

	return &netInfo, nil
}

func (vmHandler *KTVpcVMHandler) getPortForwardingIDs(privateIpAddr string) ([]string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getPortForwardingIDs()!")

	if strings.EqualFold(privateIpAddr, "") {
		newErr := fmt.Errorf("Invalid Private IP Address!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	pfRuleList, err := vmHandler.listPortForwarding()
	if err != nil {
		if strings.Contains(err.Error(), "Failed to Get Any Port Forwarding Info") {
			return nil, nil
		}
		newErr := fmt.Errorf("Failed to Get PortForwarding ID. [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	if len(pfRuleList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any PortForwarding Rule : [%v]", err)
		cblogger.Debug(newErr.Error())
		return nil, nil
		// return nil, newErr
	}

	var portForwardingIds []string
	for _, rule := range pfRuleList {
		if strings.EqualFold(rule.MappedIP, privateIpAddr) {
			portForwardingIds = append(portForwardingIds, rule.ID) // Caution!!
		}
	}
	return portForwardingIds, nil
}

func (vmHandler *KTVpcVMHandler) getPortForwardingID(privateIpAddr string, protocol string) (string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getPortForwardingID()!")

	if strings.EqualFold(privateIpAddr, "") {
		newErr := fmt.Errorf("Invalid Private IP Address!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	if strings.EqualFold(protocol, "") {
		newErr := fmt.Errorf("Invalid Protocol!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	pfRuleList, err := vmHandler.listPortForwarding()
	if err != nil {
		if strings.Contains(err.Error(), "Failed to Get Any Port Forwarding Info") {
			return "", nil
		}
		newErr := fmt.Errorf("Failed to Get PortForwarding ID. [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	if len(pfRuleList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any PortForwarding Rule : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	var portForwardingId string
	for _, rule := range pfRuleList {
		if strings.EqualFold(rule.MappedIP, privateIpAddr) && strings.EqualFold(rule.Protocol, protocol) {
			portForwardingId = rule.ID
			break
		}
	}
	return portForwardingId, nil
}

func (vmHandler *KTVpcVMHandler) removeFirewallRules(publicIp string, privateIp string) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called removeFirewallRules()!")

	if strings.EqualFold(publicIp, "") {
		newErr := fmt.Errorf("Invalid Public IP Address!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	if strings.EqualFold(privateIp, "") {
		newErr := fmt.Errorf("Invalid Public IP Address!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	
	cblogger.Info("Cloud driver: called listFirewallRule()!!")
	fwRuleList, err := vmHandler.listFirewallRule()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Firewall Rule List. [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	if len(fwRuleList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any Firewall Rule : [%v]", err)
		cblogger.Debug(newErr.Error())
		return true, newErr // Not 'false'
	}
	// cblogger.Info("\n\n### fwRuleList : ")
	// spew.Dump(fwRuleList)
	// cblogger.Info("\n\n")

    policyIDsWithPublicIp, err := vmHandler.FindFirewallPolicyIDsByIP(fwRuleList, publicIp)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find FirewallPolicy IDs [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
    // fmt.Printf("PolicyIDs for IP %s: %v\n", publicIp, policyIDs)

	policyIDsWithPrivateIp, err := vmHandler.FindFirewallPolicyIDsByIP(fwRuleList, privateIp)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find FirewallPolicy IDs [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
    // fmt.Printf("PolicyIDs for IP %s: %v\n", publicIp, policyIDs)

	allPolicyIDs := append(policyIDsWithPublicIp, policyIDsWithPrivateIp...)

	// Deletes all firewall rules matching the given policyIDs.
    for _, policyID := range allPolicyIDs {
        result := rules.Delete(vmHandler.NetworkClient, policyID)
        if result.Err != nil {
			newErr := fmt.Errorf("Failed to delete firewall rule (PolicyID: %s): %v", policyID, result.Err)
			cblogger.Error(newErr.Error())
			return false, newErr
        } else {
            cblogger.Infof("Successfully deleted firewall rule (PolicyID: %s)", policyID)
        }
    }

	return true, nil
}

func (vmHandler *KTVpcVMHandler) removePortForwardingRules(publicIp string) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called removePortForwardingRules()!")

	if strings.EqualFold(publicIp, "") {
		newErr := fmt.Errorf("Invalid Private IP Address!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

    // 1. List all port forwarding rules with pagination
    listOpts := portforward.ListOpts{
        Page:      1,
        Size: 	   20,    
    }
    pager := portforward.List(vmHandler.NetworkClient, listOpts)

    var pfIDs []string

    err := pager.EachPage(func(page pagination.Page) (bool, error) {
        pfs, err := portforward.ExtractPFs(page)
        if err != nil {
            newErr := fmt.Errorf("Failed to Extract Portforwarding Rules : [%v]", err)
			cblogger.Error(newErr.Error())
			return false, newErr
        }
        // Filter by public IP since ListOpts doesn't support PublicIP filter
        for _, pf := range pfs {
            if pf.PublicIP == publicIp {
                pfIDs = append(pfIDs, pf.ID)
            }
        }
        return true, nil
    })
    if err != nil {
		newErr := fmt.Errorf("Failed to Get Portforwarding Rule ID List : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
    }

    // 2. Delete each port forwarding rule by ID
    for _, pfID := range pfIDs {
        result := portforward.Delete(vmHandler.NetworkClient, pfID)
        if result.Err != nil {
            newErr := fmt.Errorf("Failed to Delete Portforwarding Rule (ID: %s): [%v]", pfID, result.Err)
			cblogger.Error(newErr.Error())
			return false, newErr
        } else {
            cblogger.Infof("Successfully Deleted Portforwarding Rule (ID: %s)", pfID)
        }
    }

	// Wait 3 seconds for deletion to complete
	time.Sleep(3 * time.Second)

	return true, nil
}

func (vmHandler *KTVpcVMHandler) removePublicIP(publicIpAddr string) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called removePublicIP()!")

	if strings.EqualFold(publicIpAddr, "") {
		newErr := fmt.Errorf("Invalid Public IP Address!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

    // 1. Find the public IP ID by IP address
    publicIPID, err := vmHandler.FindPublicIPIDByIP(publicIpAddr)
    if err != nil {
		newErr := fmt.Errorf("Failed to Find the PublicIP ID: [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
    }
	cblogger.Infof("PublicIP ID : %s", publicIPID)

    // 2. Delete the public IP using the found ID
    result := ips.Delete(vmHandler.NetworkClient, publicIPID)
    if result.Err != nil {
        newErr := fmt.Errorf("Failed to Delete the Public IP (ID: %s): [%v]", publicIPID, result.Err)
		cblogger.Error(newErr.Error())
		return false, newErr
    }
    cblogger.Infof("Successfully deleted public IP: %s (ID: %s)", publicIpAddr, publicIPID)

	return true, nil
}

// Blocks until the the asynchronous job has executed or has timed out.
// time.Duration unit => 1 nanosecond.  timeOut * 1,000,000,000 => 1 second
func (vmHandler *KTVpcVMHandler) waitForAsyncJob(jobId string, timeOut time.Duration) error {
	cblogger.Info("KT Cloud VPC Driver: called waitForAsyncJob()!")

	c := vmHandler.NetworkClient

	done := make(chan struct{})
	defer close(done)

	result := make(chan error, 1)
	go func() {
		attempts := 0
		for {
			attempts += 1

			cblogger.Infof("Checking the async job status... (attempt: %d)", attempts)
			jobResult, err := job.GetJobResult(c, jobId)
			if err != nil {
				result <- err
				return
			}

			// # Check status of the job
			// 0 - Pending / In progress, Continue job
			// 1 - Succeeded
			// 2 - Failed
			status := (*jobResult).JobState
			cblogger.Infof("The job status : %d", status)
			switch status {
			case 1:
				result <- nil
				return
			case 2:
				err := fmt.Errorf("waitForAsyncJob() failed")
				result <- err
				return
			}

			// Wait 3 seconds between requests
			time.Sleep(3 * time.Second)

			// Verify whether we shouldn't exit or ...
			select {
			case <-done:
				// Finished, so just exit the goroutine
				return
			default:
				// Keep going
			}
		}
	}()

	cblogger.Infof("Waiting for up to %f seconds for async job : %s", timeOut.Seconds(), jobId)
	select {
	case err := <-result:
		return err
	case <-time.After(timeOut):
		err := fmt.Errorf("Timeout while waiting to for the async job to finish")
		return err
	}
}

func (vmHandler *KTVpcVMHandler) listKTCloudVM() ([]servers.Server, error) {
	cblogger.Info("KT Cloud driver: called listKTCloudVM()!")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "listKTCloudVM()", "listKTCloudVM()")

	// Get VM list
	listOpts := servers.ListOpts{
		Limit:            100,
		AvailabilityZone: vmHandler.RegionInfo.Zone,
	}
	start := call.Start()
	pager, err := servers.List(vmHandler.VMClient, listOpts).AllPages()
	if err != nil {
		cblogger.Error(err.Error())
		loggingError(callLogInfo, err)
		return nil, err
	}
	loggingInfo(callLogInfo, start)

	vmList, err := servers.ExtractServers(pager)
	if err != nil {
		cblogger.Error(err.Error())
		loggingError(callLogInfo, err)
		return nil, err
	}
	// spew.Dump(vmList)
	return vmList, nil
}

func (vmHandler *KTVpcVMHandler) getKTCloudVM(vmId string) (servers.Server, error) {
	cblogger.Info("KT Cloud driver: called getKTCloudVM()!")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmId, "GetVM()")

	if strings.EqualFold(vmId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return servers.Server{}, newErr
	}

	start := call.Start()
	vmResult, err := servers.Get(vmHandler.VMClient, vmId).Extract()
	if err != nil {
		cblogger.Error(err.Error())
		loggingError(callLogInfo, err)
		return servers.Server{}, err
	}
	loggingInfo(callLogInfo, start)

	// spew.Dump(*vmResult)
	return *vmResult, nil
}

func (vmHandler *KTVpcVMHandler) getVmIdAndPrivateIPWithName(vmName string) (string, string, error) {
	cblogger.Info("KT Cloud driver: called getVmIdAndPrivateIPWithName()!")

	if strings.EqualFold(vmName, "") {
		newErr := fmt.Errorf("Invalid VM Name!!")
		cblogger.Error(newErr.Error())
		return "", "", newErr
	}

	// Get KT Cloud VM list
	ktVMList, err := vmHandler.listKTCloudVM()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VM List : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", "", newErr
	}
	if len(ktVMList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any VM form KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", "", newErr
	}

	var vmId string
	var vmPrivateIP string
	for _, vm := range ktVMList {
		if strings.EqualFold(vm.Name, vmName) {
			vmId = vm.ID

			for _, subnet := range vm.Addresses {
				// Get PrivateIP Info
				for _, addr := range subnet.([]interface{}) {
					addrMap := addr.(map[string]interface{})
					if addrMap["OS-EXT-IPS:type"] == "fixed" {
						vmPrivateIP = addrMap["addr"].(string)
					}
				}
			}
			break
		}
	}

	if vmId == "" {
		err := fmt.Errorf("Failed to Find the VM ID with the VM Name %s", vmName)
		return "", "", err
	} else if vmPrivateIP == "" {
		err := fmt.Errorf("Failed to Find the VM Private IP with the VM Name %s", vmName)
		return "", "", err
	} else {
		return vmId, vmPrivateIP, nil
	}
}

func (vmHandler *KTVpcVMHandler) getVmNameWithId(vmId string) (string, error) {
	cblogger.Info("KT Cloud driver: called getVmNameWithId()!")

	if strings.EqualFold(vmId, "") {
		newErr := fmt.Errorf("Invalid VM ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	ktVM, err := vmHandler.getKTCloudVM(vmId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM Info from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	vmName := ktVM.Name
	if vmName == "" {
		err := fmt.Errorf("Failed to Find the VM Name with the VM ID %s", vmId)
		return "", err
	} else {
		return vmName, nil
	}
}

func (vmHandler *KTVpcVMHandler) getPublicIPWithVMId(vmId string) (string, error) {
	cblogger.Info("KT Cloud driver: called getPublicIPWithVMId()!")

	if strings.EqualFold(vmId, "") {
		newErr := fmt.Errorf("Invalid VM ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	ktVM, err := vmHandler.getKTCloudVM(vmId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM Info from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	// RF) VMHandler mappingVMInfo()
	var subnetNameId string
	var privateIP string
	for key, subnet := range ktVM.Addresses {
		// VPC Info
		subnetNameId = key
		// Get PrivateIP Info
		for _, addr := range subnet.([]interface{}) {
			addrMap := addr.(map[string]interface{})
			if addrMap["OS-EXT-IPS:type"] == "fixed" {
				privateIP = addrMap["addr"].(string)
			}
		}
	}
	cblogger.Infof("Subnet NameId and Private IP : [%s], [%s]", subnetNameId, privateIP)

	netInfo, err := vmHandler.getNetIDsWithPrivateIP(privateIP)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get PortForwarding Info. [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	if netInfo.PublicIP == "" {
		newErr := fmt.Errorf("Failed to Find the Public IP of the VM ID(%s)", vmId)
		return "", newErr
	} else {
		return netInfo.PublicIP, nil
	}
}

// Get VM PrivateIP and OSNetworkID with VMID
func (vmHandler *KTVpcVMHandler) getVmPrivateIpAndNetIdWithVMId(vmId string) (string, string, error) {
	cblogger.Info("KT Cloud driver: called getVmPrivateIpAndNetIdWithVMId()!")

	if strings.EqualFold(vmId, "") {
		newErr := fmt.Errorf("Invalid VM ID!!")
		cblogger.Error(newErr.Error())
		return "", "", newErr
	}

	ktVM, err := vmHandler.getKTCloudVM(vmId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM Info from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", "", newErr
	}

	var subnetName string
	var privateIP string
	for key, subnet := range ktVM.Addresses {
		subnetName = key
		// Get PrivateIP Info
		for _, addr := range subnet.([]interface{}) {
			addrMap := addr.(map[string]interface{})
			if addrMap["OS-EXT-IPS:type"] == "fixed" {
				privateIP = addrMap["addr"].(string)
			}
		}
	}
	cblogger.Infof("Subnet Name and Private IP : [%s], [%s]", subnetName, privateIP)

	vpcHandler := KTVpcVPCHandler{
		RegionInfo:    vmHandler.RegionInfo,
		NetworkClient: vmHandler.NetworkClient, // Required!!
	}
	tierId, getOsNetErr := vpcHandler.getTierIdWithTierName(subnetName)
	if getOsNetErr != nil {
		newErr := fmt.Errorf("Failed to Get the OsNetwork ID with the Tier Name : [%v]", getOsNetErr)
		cblogger.Error(newErr.Error())
		return "", "", newErr
	}

	if privateIP == "" {
		err := fmt.Errorf("Failed to Find the Privatge IP with the VM ID %s", vmId)
		return "", "", err
	} else if *tierId == "" {
		err := fmt.Errorf("Failed to Find the OsNetworkId with the VM ID %s", vmId)
		return "", "", err
	} else {
		return privateIP, *tierId, nil
	}
}

func (vmHandler *KTVpcVMHandler) createLinuxInitUserData(keyPairId string) (*string, error) {
	cblogger.Info("KT Cloud driver: called createLinuxInitUserData()!!")

	initFilePath := os.Getenv("CBSPIDER_ROOT") + UbuntuCloudInitFilePath
	fileData, readErr := os.ReadFile(initFilePath)
	if readErr != nil {
		newErr := fmt.Errorf("Failed to Read the file : [%v]", readErr)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	cmdString := string(fileData)

	// # Get the publicKey from DB // Caution!! ~.KeyPairIID."SystemId"
	strList := []string{
		vmHandler.CredentialInfo.Username,
		vmHandler.CredentialInfo.Password,
	}
	hashString, err := keycommon.GenHash(strList)
	if err != nil {
		newErr := fmt.Errorf("Failed to Generate Hash String : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	keyValue, getKeyErr := keycommon.GetKey("KT", hashString, keyPairId)
	if getKeyErr != nil {
		newErr := fmt.Errorf("Failed to Get the Public Key from DB : [%v]", getKeyErr)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Set Linux cloud-init script
	cmdString = strings.ReplaceAll(cmdString, "{{username}}", LnxUserName)
	cmdString = strings.ReplaceAll(cmdString, "{{public_key}}", keyValue.Value)
	// cblogger.Info("cmdString : ", cmdString)
	return &cmdString, nil
}

func (vmHandler *KTVpcVMHandler) createWinInitUserData(passWord string) (*string, error) {
	cblogger.Info("KT Cloud driver: called createWinInitUserData()!!")

	// Preparing for UserData String
	initFilePath := os.Getenv("CBSPIDER_ROOT") + WinCloudInitFilePath
	fileData, readErr := os.ReadFile(initFilePath)
	if readErr != nil {
		newErr := fmt.Errorf("Failed to Read the file : [%v]", readErr)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	cmdString := string(fileData)

	// Set Windows cloud-init script
	cmdString = strings.ReplaceAll(cmdString, "{{PASSWORD}}", passWord)
	// cblogger.Info("cmdString : ", cmdString)
	return &cmdString, nil
}

func countSgKvList(sg sim.SecurityGroupInfo) int {
	if sg.KeyValueInfoList == nil {
		return 0
	}
	return len(sg.KeyValueInfoList)
}

func (vmHandler *KTVpcVMHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	callLogInfo := getCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "ListIID()", "ListIID()")

	// Get VM list
	listOpts := servers.ListOpts{
		Limit: 100,
	}
	start := call.Start()
	pager, err := servers.List(vmHandler.VMClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VM Pages from KT Cloud. : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	loggingInfo(callLogInfo, start)

	vmList, err := servers.ExtractServers(pager)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VM list. : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	var iidList []*irs.IID
	for _, vm := range vmList {
		iid := &irs.IID{
			NameId:   vm.Name,
			SystemId: vm.ID,
		}
		iidList = append(iidList, iid)
	}
	return iidList, nil
}

// getNetworkID() Gets Network ID of targetName from FirewallRule List
func (vmHandler *KTVpcVMHandler) getNetworkID(targetName string) (string, error) {
	cblogger.Info("KT Cloud driver: called getNetworkID()!!")

	if strings.EqualFold(targetName, "") {
		newErr := fmt.Errorf("Invalid targetName!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	fwRuleList, err := vmHandler.listFirewallRule()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Firewall Rule List. [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	if len(fwRuleList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any Firewall Rule : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	var netId string
	for _, rule := range fwRuleList {
		for _, dstInterface := range rule.DstInterface {
			if strings.EqualFold(dstInterface.Name, targetName) {				
				netId = dstInterface.NetworkID			
			}
		}
		for _, srcInterface := range rule.SrcInterface {
			if strings.EqualFold(srcInterface.Name, targetName) {
				netId = srcInterface.NetworkID				
			}
		}
	}

	if strings.EqualFold(netId, "") {
		return "", fmt.Errorf("No NetworkID found with targetName: %s", targetName)
	} else {
		return netId, nil
	}	
}

// FindFirewallPolicyIDsByIP returns a list of PolicyIDs from the given FirewallRule list
// that match the given 'public IP' or 'private IP' (as substring, CIDR, or exact match).
func (vmHandler *KTVpcVMHandler) FindFirewallPolicyIDsByIP(fwRules []rules.FirewallRule, ip string) ([]string, error) {

	if strings.EqualFold(ip, "") {
		newErr := fmt.Errorf("Invalid IP Address!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

    var policyIDs []string
    // Check for both CIDR (e.g., "211.34.246.113/32") and plain IP (e.g., "211.34.246.113")
    ipCidr := ip
    if !strings.Contains(ip, "/") {
        ipCidr = ip + "/32"
    }

    for _, rule := range fwRules {
        // Check SrcAddress
        for _, src := range rule.SrcAddress {
            if strings.Contains(src.Name, ip) || strings.Contains(src.Name, ipCidr) {
                policyIDs = append(policyIDs, rule.PolicyID)
                break
            }
        }
        // Check DstAddress
        for _, dst := range rule.DstAddress {
            if strings.Contains(dst.Name, ip) || strings.Contains(dst.Name, ipCidr) {
                policyIDs = append(policyIDs, rule.PolicyID)
                break
            }
        }
    }

	if policyIDs == nil {
		return nil, fmt.Errorf("No PolicyIDs found with IP Address: %s", ip)
	} else {
		return policyIDs, nil
	} 
}

// FindPublicIPIDByIP searches for a public IP ID based on the given public IP address
func (vmHandler *KTVpcVMHandler) FindPublicIPIDByIP(publicIPAddress string) (string, error) {

	if strings.EqualFold(publicIPAddress, "") {
		newErr := fmt.Errorf("Invalid PublicIP Address!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

    // List all floating IPs with pagination
    listOpts := ips.ListOpts{
        Page: 1,
        Size: 20,
    }
    pager := ips.List(vmHandler.NetworkClient, listOpts)

    var foundPublicIPID string

    err := pager.EachPage(func(page pagination.Page) (bool, error) {
        fips, err := ips.ExtractFloatingIPs(page)
        if err != nil {
			newErr := fmt.Errorf("Failed to Extract FloatingIPs Info [%v]", err)
			cblogger.Error(newErr.Error())
			return false, newErr            
        }
        
        // Search for matching public IP address
        for _, fip := range fips {
            if strings.EqualFold(fip.PublicIP, publicIPAddress) {
                foundPublicIPID = fip.PublicIpID
                return false, nil // Stop iteration when found
            }
        }
        return true, nil // Continue to next page
    })
    if err != nil {
        newErr := fmt.Errorf("Failed to Get the FloatingIP ID [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
    }

    if foundPublicIPID == "" {
        newErr := fmt.Errorf("The PublicIP address %s not found", publicIPAddress)
		cblogger.Error(newErr.Error())
		return "", newErr
    }

    return foundPublicIPID, nil
}

// FindPublicIPByID searches for a public IP address based on the given public IP ID
func (vmHandler *KTVpcVMHandler) FindPublicIPByID(publicIPId string) (string, error) {

	if strings.EqualFold(publicIPId, "") {
		newErr := fmt.Errorf("Invalid PublicIP ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	// Get the floating IP details
	result := ips.Get(vmHandler.NetworkClient, publicIPId)
	if result.Err != nil {
		newErr := fmt.Errorf("Floating IP with ID [%s] does NOT exist or error occurred: %v\n", publicIPId, result.Err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	ipInfo, extErr := result.ExtractFloatingIP()
	if extErr != nil {
		newErr := fmt.Errorf("Failed to Extract the PublicIP Info [%v]", extErr)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	cblogger.Infof("# PublicIP Address : [%s]", ipInfo.PublicIP)

	if strings.EqualFold(ipInfo.PublicIP, "") {
		return "", fmt.Errorf("No Public IP found with the ID: %s", publicIPId)
	} else {
		return ipInfo.PublicIP, nil
	}    
}
