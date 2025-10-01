// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP VPC VM Handler
//
// by ETRI, 2020.12.
// by ETRI, 2022.02. updated
// by ETRI, 2025.01. updated
// by ETRI, 2025.10. updated
//==================================================================================================

package resources

import (
	"errors"
	"fmt"
	// "reflect"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	// "github.com/davecgh/go-spew/spew"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	keycommon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	sim "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncp/resources/info_manager/security_group_info_manager"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVpcVMHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *vserver.APIClient
}

const (
	lnxUserName 			string = "cb-user"
	winUserName             string = "Administrator"
	ubuntuCloudInitFilePath	string = "/cloud-driver-libs/.cloud-init-ncp/cloud-init"
	centosCloudInitFilePath	string = "/cloud-driver-libs/.cloud-init-ncp/cloud-init-centos"
	winCloudInitFilePath 	string = "/cloud-driver-libs/.cloud-init-ncp/cloud-init-windows"
	LnxTypeOs 				string = "LNX" // LNX (LINUX)
	WinTypeOS 				string = "WND" // WND (WINDOWS)
	KVMRootDiskType 		string = "CB1" // Default root disk type for KVM-based VMs
)

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP VPC VMHandler")
}

func (vmHandler *NcpVpcVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	cblogger.Info("NCP VPC Cloud driver: called StartVM()!!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Region, call.VM, vmReqInfo.IId.NameId, "StartVM()")

	if strings.EqualFold(vmReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid VM Name required")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	// CAUTION!! : Instance Name is Convert to lowercase.(strings.ToLower())
	// NCP VPC does not allow uppercase letters in VM instance name, so convert to lowercase here to reflect.(uppercase : Error occurred)
	instanceName := strings.ToLower(vmReqInfo.IId.NameId)
	keyPairId := vmReqInfo.KeyPairIID.SystemId
	vpcId := vmReqInfo.VpcIID.SystemId
	subnetId := vmReqInfo.SubnetIID.SystemId

	minCount := ncloud.Int32(1)

	var securityGroupIds []*string
	for _, sgID := range vmReqInfo.SecurityGroupIIDs {
		// cblogger.Infof("Security Group IID : [%s]", sgID)
		securityGroupIds = append(securityGroupIds, ncloud.String(sgID.SystemId))
	}

	// Check whether the VM name exists
	// Search by instanceName converted to lowercase
	vmId, getErr := vmHandler.getVmIdByName(instanceName)
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Get VmId with the Name : [%s], [%v]", instanceName, getErr)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}
	if vmId != "" {
		newErr := fmt.Errorf("The VM Name [%s] is already In Use.", instanceName)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	var publicImageId string
	var publicImageSpecId string
	var myImageId string
	// var myImageSpecId string
	// var serverProductCode string

	var initScriptNo *string
	var instanceReq vserver.CreateServerInstancesRequest
	orderInt32 := ncloud.Int32(0) // Convert numer 0 to *int32 type

	// In case of Public Image
	if vmReqInfo.ImageType == irs.PublicImage || vmReqInfo.ImageType == "" || vmReqInfo.ImageType == "default" {
		imageHandler := NcpVpcImageHandler{
			RegionInfo: vmHandler.RegionInfo,
			VMClient:   vmHandler.VMClient,
		}

		isPublicImage, err := imageHandler.isPublicImage(vmReqInfo.ImageIID.SystemId)
		if err != nil {
			newErr := fmt.Errorf("Failed to Check Whether the Image is Public Image : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
		if !isPublicImage {
			newErr := fmt.Errorf("'PublicImage' type is selected, but Specified image is Not a PublicImage in the region!!")
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		} else {
			publicImageId = vmReqInfo.ImageIID.SystemId
			publicImageSpecId = vmReqInfo.VMSpecName

			cblogger.Infof("publicImageId : [%s]", publicImageId)
			cblogger.Infof("publicImageSpecId : [%s]", publicImageSpecId)
		}

		isPublicWindowsImage, err := imageHandler.CheckWindowsImage(vmReqInfo.ImageIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Check Whether the Image is MS Windows Image : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
		if isPublicWindowsImage {
			var createErr error
			initScriptNo, createErr = vmHandler.createWinInitScript(vmReqInfo.VMUserPasswd)
			if createErr != nil {
				newErr := fmt.Errorf("Failed to Create Cloud-Init Script with the Password : [%v]", createErr)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
		} else {
			var createErr error
			initScriptNo, createErr = vmHandler.createLinuxInitScript(vmReqInfo.ImageIID, keyPairId)
			if createErr != nil {
				newErr := fmt.Errorf("Failed to Create Cloud-Init Script with the KeyPairId : [%v]", createErr)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
		}

		var reqDiskType string
		if strings.EqualFold(vmReqInfo.RootDiskType, "default") || strings.EqualFold(vmReqInfo.RootDiskType, "HDD") {
			reqDiskType = KVMRootDiskType
		} else if strings.EqualFold(vmReqInfo.RootDiskType, "SSD") {
			newErr := fmt.Errorf("Invalid root disk type. KVM-based VMs only support root disks of the ‘HDD’ type.")
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}

		instanceReq = vserver.CreateServerInstancesRequest{
			RegionCode: ncloud.String(vmHandler.RegionInfo.Region),
			ServerName: ncloud.String(instanceName),
			// MemberServerImageInstanceNo:	ncloud.String(myImageId),
			// ServerImageProductCode: 		ncloud.String(publicImageId), // In case using New publicImageId(from New API). Use 'ServerImageNo' parameter!!
			// ServerProductCode:      		ncloud.String(serverProductCode), // In case using New vmSpecId(from New API). Use 'ServerSpecCode' parameter!!
			LoginKeyName: ncloud.String(keyPairId),
			VpcNo:        ncloud.String(vpcId),
			SubnetNo:     ncloud.String(subnetId), // Applied for Zone-based control!!

			ServerImageNo:  ncloud.String(publicImageId),     // Added for using imageId from New API
			ServerSpecCode: ncloud.String(publicImageSpecId), // Added for using specId from New API

			// ### Caution!! : AccessControlGroup corresponds to Server > 'ACG', not VPC > 'Network ACL' in the NCP VPC console.
			NetworkInterfaceList: []*vserver.NetworkInterfaceParameter{
				{
					NetworkInterfaceOrder: 		orderInt32,					
					// If you don't specify 'NetworkInterfaceNo', a NetworkInterface is automatically generated and applied.
					AccessControlGroupNoList: 	securityGroupIds,
				},
			},
			
			BlockStorageMappingList: []*vserver.BlockStorageMappingParameter{
				{
					Order:						orderInt32,
					BlockStorageVolumeTypeCode: ncloud.String(reqDiskType),
					BlockStorageSize: 			ncloud.String(vmReqInfo.RootDiskSize),
				},
			}, 

			IsProtectServerTermination: ncloud.Bool(false), // Caution!! : If set to 'true', Terminate (VM return) is not controlled by API.
			ServerCreateCount:          minCount,
			InitScriptNo:               initScriptNo,
		}

	} else { // In case of My Image
		imageHandler := NcpVpcImageHandler{
			RegionInfo: vmHandler.RegionInfo,
			VMClient:   vmHandler.VMClient,
		}
		isPublicImage, err := imageHandler.isPublicImage(vmReqInfo.ImageIID.SystemId)
		if err != nil {
			newErr := fmt.Errorf("Failed to Check Whether the Image is Public Image : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
		if isPublicImage {
			newErr := fmt.Errorf("'MyImage' type is selected, but Specified image is Not a MyImage!!")
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		} else {
			myImageId = vmReqInfo.ImageIID.SystemId
			// myImageSpecId = vmReqInfo.VMSpecName
		}

		// vmSpecHandler := NcpVpcVMSpecHandler{
		// 	RegionInfo:  vmHandler.RegionInfo,
		// 	VMClient:    vmHandler.VMClient,
		// }
		// var getErr error
		// serverProductCode, getErr = vmSpecHandler.getNcpVpcServerProductCode(myImageSpecId)
		// if err != nil {
		// 	newErr := fmt.Errorf("Failed to Get ServerProductCode from NCP VPC : ", getErr)
		// 	cblogger.Error(newErr.Error())
		// 	LoggingError(callLogInfo, newErr)
		// 	return irs.VMInfo{}, newErr
		// }

		myImageHandler := NcpVpcMyImageHandler{
			RegionInfo: vmHandler.RegionInfo,
			VMClient:   vmHandler.VMClient,
		}
		isMyWindowsImage, err := myImageHandler.CheckWindowsImage(vmReqInfo.ImageIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Check Whether My Image is MS Windows Image : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
		if isMyWindowsImage {
			var createErr error
			initScriptNo, createErr = vmHandler.createWinInitScript(vmReqInfo.VMUserPasswd)
			if createErr != nil {
				newErr := fmt.Errorf("Failed to Create Cloud-Init Script with the Password : [%v]", createErr)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
		} else {
			var createErr error
			initScriptNo, createErr = vmHandler.createLinuxInitScript(vmReqInfo.ImageIID, keyPairId)
			if createErr != nil {
				newErr := fmt.Errorf("Failed to Create Cloud-Init Script with the KeyPairId : [%v]", createErr)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
		}

		// ### Note) "These parameters cannot be used at the same time : [memberServerImageInstanceNo, serverImageProductCode, serverSpecCode]"
		// $$$ Need to check what to set as a vmSpec when creating a VM with a MyImge(MemberServerImageInstanceNo).
		instanceReq = vserver.CreateServerInstancesRequest{
			RegionCode:                  ncloud.String(vmHandler.RegionInfo.Region),
			ServerName:                  ncloud.String(instanceName),
			MemberServerImageInstanceNo: ncloud.String(myImageId),
			// ServerImageProductCode: 		ncloud.String(publicImageId), // In case using New publicImageId(from New API). Use 'ServerImageNo' parameter!!
			// ServerProductCode:      		ncloud.String(serverProductCode), // In case using New vmSpecId(from New API). Use 'ServerSpecCode' parameter!!
			LoginKeyName: 				ncloud.String(keyPairId),
			VpcNo:        				ncloud.String(vpcId),
			SubnetNo:     				ncloud.String(subnetId), // Applied for Zone-based control!!

			// Note) If enabled and set "", an error will occur on VM creation with 'MemberServerImageInstanceNo'.
			// ServerImageNo: 				ncloud.String(publicImageId), // Added for using imageId from New API
			// ServerSpecCode: 				ncloud.String(publicImageSpecId), // Added for using specId from New API

			// ### Caution!! : AccessControlGroup corresponds to Server > 'ACG', not VPC > 'Network ACL' in the NCP VPC console.
			NetworkInterfaceList: []*vserver.NetworkInterfaceParameter{
				{
					NetworkInterfaceOrder: 		orderInt32, 
					// If you don't specify 'NetworkInterfaceNo', a NetworkInterface is automatically generated and applied.
					AccessControlGroupNoList: 	securityGroupIds,
				},				
			},

			IsProtectServerTermination: ncloud.Bool(false), // Caution!! : If set to 'true', Terminate (VM return) is not controlled by API.
			ServerCreateCount:          minCount,
			InitScriptNo:               initScriptNo,
		}
	}
	// cblogger.Info("# instanceReq")
	// spew.Dump(instanceReq)

	callLogStart := call.Start()
	cblogger.Info("# Start to Create NCP VPC VM Instance!!")
	runResult, err := vmHandler.VMClient.V2Api.CreateServerInstances(&instanceReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create VM instance : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		scriptDelResult, err := vmHandler.deleteInitScript(initScriptNo)
		if err != nil {
			newErr := fmt.Errorf("Failed to Delete the Cloud-Init Script with the initScriptNo : [%s], [%v]", ncloud.StringValue(initScriptNo), err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
		cblogger.Infof("deleteInitScript Result : [%s]", *scriptDelResult)
		return irs.VMInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	newVMIID := irs.IID{SystemId: ncloud.StringValue(runResult.ServerInstanceList[0].ServerInstanceNo)}

	//=========================================
	// Wait for VM information to be inquired
	//=========================================
	cblogger.Infof("# Waitting while Initializing New VM!!")
	time.Sleep(time.Second * 15) // Waitting Before Getting New VM Status Info!!

	curStatus, statusErr := vmHandler.WaitToGetVMInfo(newVMIID) // # Waitting while Creating VM!!")
	if statusErr != nil {
		cblogger.Error(statusErr.Error())
		LoggingError(callLogInfo, statusErr)
		return irs.VMInfo{}, statusErr
	}
	cblogger.Infof("==> VM [%s] status : [%s]", newVMIID.SystemId, curStatus)
	cblogger.Info("VM Creation Processes are Finished !!")

	scriptDelResult, err := vmHandler.deleteInitScript(initScriptNo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Cloud-Init Script with the initScriptNo : [%s], [%v]", ncloud.StringValue(initScriptNo), err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	cblogger.Infof("deleteInitScript Result : [%s]", *scriptDelResult)

	// Register SecurityGroupInfo to DB
	var keyValueList []irs.KeyValue
	for _, sgIID := range vmReqInfo.SecurityGroupIIDs {
		keyValueList = append(keyValueList, irs.KeyValue{
			Key:   sgIID.SystemId,
			Value: sgIID.SystemId,
		})
	}

	providerName := "NCP"
	sgInfo, regErr := sim.RegisterSecurityGroup(newVMIID.SystemId, providerName, keyValueList)
	if regErr != nil {
		cblogger.Error(regErr)
		return irs.VMInfo{}, regErr
	}
	cblogger.Infof(" === S/G Info to Register to DB : [%v]", sgInfo)

	vmInfo, error := vmHandler.GetVM(newVMIID)
	if error != nil {
		newErr := fmt.Errorf("Failed to Get the New VM Info with the VM ID : [%s], [%v]", newVMIID.SystemId, error)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	if vmInfo.Platform == irs.WINDOWS {
		vmInfo.VMUserPasswd = vmReqInfo.VMUserPasswd
	}
	return vmInfo, nil
}

func (vmHandler *NcpVpcVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	cblogger.Info("NCP VPC Cloud driver: called GetVM()!!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "GetVM()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId required")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	ncpVMInfo, err := vmHandler.getNcpVMInfo(vmIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}

	vmInfo, err := vmHandler.mappingVMInfo(ncpVMInfo)
	if err != nil {
		LoggingError(callLogInfo, err)
		return irs.VMInfo{}, err
	}
	return vmInfo, nil
}

func (vmHandler *NcpVpcVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCP VPC Cloud driver: called SuspendVM()!!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "SuspendVM()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId required")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMStatus("Failed"), newErr
	}

	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM Status with the VM ID : [%s], [%v]", vmIID.SystemId, err)
		cblogger.Debug(newErr.Error())
		return irs.VMStatus("Failed"), newErr
	} else {
		cblogger.Infof("Succeed in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, vmStatus)
	}

	instanceNoList := []*string{ncloud.String(vmIID.SystemId),}
	var resultStatus string
	if strings.EqualFold(string(vmStatus), "Suspending") {
		resultStatus = "The VM is already in the process of Suspending."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Suspended") {
		resultStatus = "The VM is already Suspended."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Rebooting") {
		resultStatus = "The VM is in the process of Rebooting."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Terminating") {
		resultStatus = "The VM is in the process of Terminating."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Booting") {
		resultStatus = "The VM is in the process of Booting."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else {
		curStatus, statusErr := vmHandler.WaitForDiskAttach(vmIID) // # Waitting while Root disk is fully attached!!"
		if statusErr != nil {
			cblogger.Error(statusErr.Error())
			LoggingError(callLogInfo, statusErr)
			return irs.VMStatus("Failed to wait while Root disk is attaching!!"), statusErr
		}
		cblogger.Infof("==> Root disk [%s] status : [%s]", vmIID.SystemId, curStatus)
		cblogger.Info("The Root disk has been fully Attatched to the VM!!")

		retryCount := 0
		timeout := 7 * time.Second
		req := vserver.StopServerInstancesRequest{
			RegionCode:           ncloud.String(vmHandler.RegionInfo.Region), // $$$ Caution!!
			ServerInstanceNoList: instanceNoList,
		}
		callLogStart := call.Start()
		runResult, err := vmHandler.VMClient.V2Api.StopServerInstances(&req)
		if err != nil {
			cblogger.Infof("Return message : [%v]", err.Error())

			if strings.Contains(err.Error(), "The storage allocated to the server is being manipulated.") || strings.Contains(err.Error(), "The storage assigned to the server is in operation.") {
				retryCount++
				if retryCount >= 6 {
					_, err := vmHandler.VMClient.V2Api.StopServerInstances(&req)
					if err != nil {
						cblogger.Error(err.Error())
						LoggingError(callLogInfo, err)
						return irs.VMStatus("Suspending"), err
					}
				}
				time.Sleep(timeout)
        	}
		}
		LoggingInfo(callLogInfo, callLogStart)
		cblogger.Infof("[%v]", runResult)
	}
	return irs.VMStatus("Suspending"), nil
}

func (vmHandler *NcpVpcVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCP VPC Cloud driver: called ResumeVM()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "ResumeVM()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId required")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMStatus("Failed"), newErr
	}

	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM Status with the VM ID : [%s], [%v]", vmIID.SystemId, err)
		cblogger.Debug(newErr.Error())
		return irs.VMStatus("Failed"), newErr
	} else {
		cblogger.Infof("Succeed in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, vmStatus)
	}

	serverInstanceNo := []*string{ncloud.String(vmIID.SystemId)}
	var resultStatus string
	if strings.EqualFold(string(vmStatus), "Running") {
		resultStatus = "The VM is Running. Cannot be Resumed!!"
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Suspending") {
		resultStatus = "The VM is in the process of Suspending. Cannot be Resumed"
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

	} else if strings.EqualFold(string(vmStatus), "Booting") {
		resultStatus = "The VM is in the process of Booting. Cannot be Resumed"
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Creating") {
		resultStatus = "The VM is in the process of Creating. Cannot be Resumed"
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else {
		cblogger.Infof("vmID : [%s]", *serverInstanceNo[0])
		req := vserver.StartServerInstancesRequest{
			RegionCode:           ncloud.String(vmHandler.RegionInfo.Region), // $$$ Caution!!
			ServerInstanceNoList: serverInstanceNo,
		}
		callLogStart := call.Start()
		runResult, err := vmHandler.VMClient.V2Api.StartServerInstances(&req)
		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(callLogInfo, err)
			return irs.VMStatus("Failed"), err
		}
		LoggingInfo(callLogInfo, callLogStart)
		cblogger.Infof("[%v]", runResult)

		return irs.VMStatus("Resuming"), nil
	}
}

func (vmHandler *NcpVpcVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCP VPC Cloud driver: called RebootVM()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "RebootVM()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId required")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMStatus("Failed"), newErr
	}

	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM Status with the VM ID : [%s], [%v]", vmIID.SystemId, err)
		cblogger.Debug(newErr.Error())
		return irs.VMStatus("Failed"), newErr
	} else {
		cblogger.Infof("Succeed in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, vmStatus)
	}

	serverInstanceNo := []*string{ncloud.String(vmIID.SystemId)}
	var resultStatus string
	if strings.EqualFold(string(vmStatus), "Suspending") {
		resultStatus = "The VM is in the process of Suspending."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Suspended") {
		resultStatus = "The VM is Suspended."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Rebooting") {
		resultStatus = "The VM is already in the process of Rebooting."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Terminating") {
		resultStatus = "The VM is in the process of Terminating."
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

	} else {
		cblogger.Infof("vmID : [%s]", *serverInstanceNo[0])
		req := vserver.RebootServerInstancesRequest{
			RegionCode:           ncloud.String(vmHandler.RegionInfo.Region), // $$$ Caution!!
			ServerInstanceNoList: serverInstanceNo,
		}
		callLogStart := call.Start()
		runResult, err := vmHandler.VMClient.V2Api.RebootServerInstances(&req)
		if err != nil {
			newErr := fmt.Errorf("Failed to Reboot the VM Instance on NCP VPC!! : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMStatus("Failed. "), newErr
		}
		LoggingInfo(callLogInfo, callLogStart)
		cblogger.Infof("[%v]", runResult)

		return irs.VMStatus("Rebooting"), nil
	}
}

func (vmHandler *NcpVpcVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCP VPC Cloud driver: called TerminateVM()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "TerminateVM()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId required")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMStatus("Failed"), newErr
	}

	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM Status with the VM ID : [%s], [%v]", vmIID.SystemId, err)
		cblogger.Error(newErr.Error())
		return irs.VMStatus("Failed"), newErr
	} else {
		cblogger.Infof("Succeed in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, vmStatus)
	}

	vmInfo, error := vmHandler.GetVM(vmIID)
	if error != nil {
		LoggingError(callLogInfo, error)
		cblogger.Error(error.Error())
		return irs.VMStatus("Failed to get the VM info"), err
	}

	serverInstanceNos := []*string{ncloud.String(vmIID.SystemId)}
	switch string(vmStatus) {
	case "Suspended":
		cblogger.Info("VM Status : 'Suspended'. so it Can be Terminated!!")

		retryCount := 0
		timeout := 7 * time.Second
		req := vserver.TerminateServerInstancesRequest{
			RegionCode:           ncloud.String(vmHandler.RegionInfo.Region), // $$$ Caution!!
			ServerInstanceNoList: serverInstanceNos,
		}
		start := call.Start()
		runResult, err := vmHandler.VMClient.V2Api.TerminateServerInstances(&req)
		if err != nil {
			cblogger.Infof("Return message : [%v]", err.Error())
			
			if strings.Contains(err.Error(), "The storage allocated to the server is being manipulated.") || strings.Contains(err.Error(), "The storage assigned to the server is in operation.") {
				retryCount++
				if retryCount >= 6 {
					_, err := vmHandler.VMClient.V2Api.TerminateServerInstances(&req)
					if err != nil {
						cblogger.Error(err.Error())
						LoggingError(callLogInfo, err)
						return irs.VMStatus("Suspending"), err
					}
				}
				time.Sleep(timeout)
        	}

			newErr := fmt.Errorf("Failed to Terminate the VM instance on NCP VPC. : [%v]", err)
			cblogger.Error(newErr.Error())
			cblogger.Error(*runResult.ReturnMessage)
			LoggingError(callLogInfo, newErr)
			return irs.VMStatus("Failed to Terminate!!"), newErr
		}
		LoggingInfo(callLogInfo, start)
		// cblogger.Infof("[%v]", runResult)

		// If the NCP instance has a 'Public IP', delete it after termination of the instance.
		if !strings.EqualFold(vmInfo.PublicIP, "") {
			vmStatus, err := vmHandler.DeletePublicIP(vmInfo)
			if err != nil {
				cblogger.Error(err)
				return vmStatus, err
			}
		}

		// Delete the S/G info from DB
		_, unRegErr := sim.UnRegisterSecurityGroup(vmIID.SystemId)
		if unRegErr != nil {
			cblogger.Debug(unRegErr.Error())
			// return irs.Failed, unRegErr
		}

		return irs.VMStatus("Terminating"), nil

	case "Running":
		cblogger.Infof("VM Status : 'Running'. so it Can be Terminated AFTER SUSPENSION !!")
		cblogger.Infof("vmID : [%s]", *serverInstanceNos[0])

		curStatus, statusErr := vmHandler.WaitForDiskAttach(vmIID) // # Waitting while Root disk is fully attached!!"
		if statusErr != nil {
			cblogger.Error(statusErr.Error())
			LoggingError(callLogInfo, statusErr)
			return irs.VMStatus("Failed to wait while Root disk is attaching!!"), statusErr
		}
		cblogger.Infof("==> Root disk [%s] status : [%s]", vmIID.SystemId, curStatus)
		cblogger.Info("The Root disk has been fully Attatched to the VM!!")

		cblogger.Info("Start Suspend VM !!")
		result, err := vmHandler.SuspendVM(vmIID)
		if err != nil {
			cblogger.Errorf("Failed to Suspend the VM [%s] :  [%s]", vmIID.SystemId, result)
			cblogger.Error(err)
		} else {
			cblogger.Infof("Succeed in Suspending the VM [%s] : [%s]", vmIID.SystemId, result)
		}

		//===================================
		// 15-second wait for Suspending
		//===================================
		curRetryCnt := 0
		maxRetryCnt := 15
		for {
			curStatus, statusErr := vmHandler.GetVMStatus(vmIID)
			if statusErr != nil {
				cblogger.Error(statusErr.Error())
			}

			cblogger.Infof("===> VM Status : [%s]", curStatus)
			if curStatus != irs.VMStatus("Suspended") {
				curRetryCnt++
				cblogger.Infof("The VM is not 'Suspended' yet, so wait for a second more before inquiring Termination.")
				time.Sleep(time.Second * 5)
				if curRetryCnt > maxRetryCnt {
					cblogger.Errorf("Despite waiting for a long time(%d sec), the VM is not 'suspended', so it is forcibly terminated.", maxRetryCnt)
				}
			} else {
				break
			}
		}
		cblogger.Info("# SuspendVM() Finished")

		retryCount := 0
		timeout := 7 * time.Second
		req := vserver.TerminateServerInstancesRequest{
			RegionCode:           ncloud.String(vmHandler.RegionInfo.Region), // $$$ Caution!!
			ServerInstanceNoList: serverInstanceNos,
		}
		callLogStart := call.Start()
		runResult, err := vmHandler.VMClient.V2Api.TerminateServerInstances(&req)
		if err != nil {
			cblogger.Infof("Return message : [%v]", err.Error())

			if strings.Contains(err.Error(), "The storage allocated to the server is being manipulated.") || strings.Contains(err.Error(), "The storage assigned to the server is in operation.") {
				retryCount++
				if retryCount >= 6 {
					_, err := vmHandler.VMClient.V2Api.TerminateServerInstances(&req)
					if err != nil {
						cblogger.Error(err.Error())
						LoggingError(callLogInfo, err)
						return irs.VMStatus("Suspending"), err
					}
				}
				time.Sleep(timeout)
        	}

			newErr := fmt.Errorf("Failed to Terminate the VM instance on NCP VPC. : [%v]", err)
			cblogger.Error(newErr.Error())
			cblogger.Error(*runResult.ReturnMessage)
			LoggingError(callLogInfo, newErr)
			return irs.VMStatus("Failed to Terminate!!"), newErr
		}
		LoggingInfo(callLogInfo, callLogStart)
		// cblogger.Infof("[%v]", runResult)

		// If the NCP instance has a 'Public IP', delete it after termination of the instance.
		if !strings.EqualFold(vmInfo.PublicIP, "") {
			vmStatus, err := vmHandler.DeletePublicIP(vmInfo)
			if err != nil {
				cblogger.Error(err)
				return vmStatus, err
			}
		}

		return irs.VMStatus("Terminating"), nil

	case "Not Exist!!":
		newErr := fmt.Errorf("The VM instance does Not Exist!!")
		cblogger.Error(newErr.Error())
		return irs.VMStatus("Failed to Terminate!!"), newErr

	default:
		newErr := fmt.Errorf("The VM status is not 'Running' or 'Suspended' yet. so it Can NOT be Terminated!! Run or Suspend the VM before terminating.")
		cblogger.Error(newErr.Error())
		return irs.VMStatus("Failed to Terminate!!"), newErr
	}
}

/*
# NCP serverInstanceStatusName
init
creating
booting //Caution!! : During Creating or Resuming
setting up

running
rebooting
hard rebooting
shutting down //Caution!! : During Suspending
hard shutting down
terminating

changingSpec
copying
repairing
*/

func convertVMStatusString(vmStatus string) (irs.VMStatus, error) {
	// cblogger.Info("NCP VPC Cloud driver: called convertVMStatusString()!")

	if strings.EqualFold(vmStatus, "") {
		newErr := fmt.Errorf("Invalid VM Status")
		cblogger.Error(newErr.Error())
		return irs.VMStatus("Failed"), newErr
	}

	var resultStatus string
	if strings.EqualFold(vmStatus, "creating") {
		resultStatus = "Creating"
	} else if strings.EqualFold(vmStatus, "init") {
		resultStatus = "Creating"
	} else if strings.EqualFold(vmStatus, "booting") {
		//Caution!!
		resultStatus = "Creating"
	} else if strings.EqualFold(vmStatus, "setting up") {
		resultStatus = "Creating"
	} else if strings.EqualFold(vmStatus, "running") {
		resultStatus = "Running"
	} else if strings.EqualFold(vmStatus, "shutting down") {
		//Caution!!
		resultStatus = "Suspending"
	} else if strings.EqualFold(vmStatus, "running") {
		resultStatus = "Running"
	} else if strings.EqualFold(vmStatus, "stopped") {
		resultStatus = "Suspended"
	} else if strings.EqualFold(vmStatus, "rebooting") {
		resultStatus = "Rebooting"
	} else if strings.EqualFold(vmStatus, "hard rebooting") {
		resultStatus = "Rebooting"
	} else if strings.EqualFold(vmStatus, "hard shutting down") {
		resultStatus = "Terminating"
	} else if strings.EqualFold(vmStatus, "terminating") {
		resultStatus = "Terminating"
	} else {
		cblogger.Errorf("No mapping information found matching with the vmStatus [%s].", string(vmStatus))
		return irs.VMStatus("Failed. "), errors.New(vmStatus + "No mapping information found matching with the vmStatus.")
	}
	// cblogger.Infof("VM Status Conversion Completed : [%s] ==> [%s]", vmStatus, resultStatus)

	return irs.VMStatus(resultStatus), nil
}

func (vmHandler *NcpVpcVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCP VPC Cloud driver: called GetVMStatus()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "GetVMStatus()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId required")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMStatus("Failed"), newErr
	}

	instanceReq := vserver.GetServerInstanceListRequest{
		RegionCode: ncloud.String(vmHandler.RegionInfo.Region), // $$$ Caution!!
		ServerInstanceNoList: []*string{
			ncloud.String(vmIID.SystemId),
		},
	}
	callLogStart := call.Start()
	result, err := vmHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find VM Instance List from NCP VPC!! : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMStatus("Failed. "), newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.ServerInstanceList) < 1 {
		newErr := fmt.Errorf("The VM instance does Not Exist!!")
		cblogger.Debug(newErr.Error())
		return irs.VMStatus("Not Exist!!"), newErr
	}

	vmStatus, statusErr := convertVMStatusString(*result.ServerInstanceList[0].ServerInstanceStatusName)
	// cblogger.Infof("VM Status of [%s] : [%s]", vmIID.SystemId, vmStatus)
	return vmStatus, statusErr
}

func (vmHandler *NcpVpcVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	cblogger.Info("NCP VPC Cloud driver: called ListVMStatus()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "ListVMStatus()", "ListVMStatus()")

	instanceReq := vserver.GetServerInstanceListRequest{
		RegionCode:           ncloud.String(vmHandler.RegionInfo.Region), // $$$ Caution!!
		ServerInstanceNoList: []*string{},
	}
	callLogStart := call.Start()
	result, err := vmHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find VM Instance List from NCP VPC!! : [%v]", err)
		cblogger.Error(newErr.Error())
		cblogger.Error(*result.ReturnMessage)
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)
	cblogger.Info("Succeeded in Getting ServerInstanceList from NCP VPC!!")

	var vmStatusList []*irs.VMStatusInfo
	for _, vm := range result.ServerInstanceList {
		vmStatus, _ := convertVMStatusString(*vm.ServerInstanceStatusName)
		vmStatusInfo := irs.VMStatusInfo{
			IId: irs.IID{
				NameId:   *vm.ServerName,
				SystemId: *vm.ServerInstanceNo,
			},
			VmStatus: vmStatus,
		}
		cblogger.Infof("VM Status of [%s] : [%s]", vmStatusInfo.IId.SystemId, vmStatusInfo.VmStatus)
		vmStatusList = append(vmStatusList, &vmStatusInfo)
	}
	return vmStatusList, nil
}

func (vmHandler *NcpVpcVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger.Info("NCP VPC Cloud driver: called ListVM()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "ListVMS()", "ListVM()")

	instanceReq := vserver.GetServerInstanceListRequest{
		RegionCode:           ncloud.String(vmHandler.RegionInfo.Region), // $$$ Caution!!
		ServerInstanceNoList: []*string{},
	}
	callLogStart := call.Start()
	result, err := vmHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VM Instance List from NCP VPC!! : [%v]", err)
		cblogger.Error(newErr.Error())
		cblogger.Error(*result.ReturnMessage)
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var vmInfoList []*irs.VMInfo
	for _, vm := range result.ServerInstanceList {
		curStatus, statusErr := vmHandler.GetVMStatus(irs.IID{SystemId: *vm.ServerInstanceNo})
		if statusErr != nil {
			newErr := fmt.Errorf("Failed to Get the Status of VM : [%s], [%v]", *vm.ServerInstanceNo, statusErr.Error())
			cblogger.Error(newErr.Error())
			return nil, newErr
		} else {
			cblogger.Infof("Succeeded in Getting the Status of VM [%s] : [%s]", *vm.ServerInstanceNo, string(curStatus))
		}
		cblogger.Infof("===> VM Status : [%s]", string(curStatus))

		if (string(curStatus) != "Creating") && (string(curStatus) != "Terminating") {
			cblogger.Infof("===> The VM Status not 'Creating' or 'Terminating', you can get the VM info.")
			vmInfo, error := vmHandler.GetVM(irs.IID{SystemId: *vm.ServerInstanceNo})
			if error != nil {
				cblogger.Error(error.Error())
				return nil, error
			}
			vmInfoList = append(vmInfoList, &vmInfo)
		}
	}
	return vmInfoList, nil
}

func (vmHandler *NcpVpcVMHandler) mappingVMInfo(NcpInstance *vserver.ServerInstance) (irs.VMInfo, error) {
	cblogger.Info("NCP VPC Cloud driver: called mappingVMInfo()!")
	// cblogger.Infof("# NcpInstance Info :")
	// spew.Dump(NcpInstance)

	var publicIp *string
	var privateIp *string

	convertedTime, err := convertTimeFormat(*NcpInstance.CreateDate)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert the Time Format!!")
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}

	// Create a PublicIp, if the instance doesn't have a 'Public IP' after creation.
	if strings.EqualFold(ncloud.StringValue(NcpInstance.PublicIp), "") {
		publicIpReq := vserver.CreatePublicIpInstanceRequest{
			ServerInstanceNo: NcpInstance.ServerInstanceNo,
			RegionCode:       ncloud.String(vmHandler.RegionInfo.Region),
		}

		// CAUTION!! : The number of Public IPs cannot be more than the number of instances on NCP cloud default service.
		result, err := vmHandler.VMClient.V2Api.CreatePublicIpInstance(&publicIpReq)
		if err != nil {
			newErr := fmt.Errorf("Failed to Create Public IP : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
		if len(result.PublicIpInstanceList) < 1 {
			newErr := fmt.Errorf("Failed to Create Any Public IP!!")
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}

		publicIp = result.PublicIpInstanceList[0].PublicIp
		privateIp = result.PublicIpInstanceList[0].PrivateIp

		cblogger.Infof("*** PublicIp : %s ", ncloud.StringValue(publicIp))
		cblogger.Infof("Finished to Create Public IP")
	} else {
		publicIp = NcpInstance.PublicIp
		cblogger.Infof("*** NcpInstance.PublicIp : %s ", ncloud.StringValue(publicIp))

		instanceReq := vserver.GetPublicIpInstanceListRequest{
			RegionCode: ncloud.String(vmHandler.RegionInfo.Region), // $$$ Caution!!
			PublicIp:   publicIp,
		}
		// Search the Public IP list info. to get the PublicIp InstanceNo
		result, err := vmHandler.VMClient.V2Api.GetPublicIpInstanceList(&instanceReq)
		if err != nil {
			newErr := fmt.Errorf("Failed to Find PublicIp InstanceList from NCP VPC : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
		if len(result.PublicIpInstanceList) < 1 {
			newErr := fmt.Errorf("Failed to Find Any PublicIpInstance!!")
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
		privateIp = result.PublicIpInstanceList[0].PrivateIp

		cblogger.Infof("Finished to Get PublicIP InstanceNo")
	}

	netInterfaceName, err := vmHandler.getNetworkInterfaceName(NcpInstance.NetworkInterfaceNoList[0])
	if err != nil {
		newErr := fmt.Errorf("Failed to Find NetworkInterface Name : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}

	// Caution!!) Because disk info within the NCP VM info is being returned with incorrect value.
	var rootDiskType string
	if strings.EqualFold(*NcpInstance.BaseBlockStorageDiskDetailType.CodeName, "CB1") || strings.EqualFold(*NcpInstance.BaseBlockStorageDiskDetailType.CodeName, "CB2") {
		rootDiskType = "SSD"
	} else if strings.EqualFold(*NcpInstance.BaseBlockStorageDiskDetailType.CodeName, "SSD") {
		rootDiskType = "HDD"
	}

	// PublicIpID : Using for deleting the PublicIP
	vmInfo := irs.VMInfo{
		IId: irs.IID{
			NameId:   *NcpInstance.ServerName,
			SystemId: *NcpInstance.ServerInstanceNo,
		},

		StartTime: convertedTime,

		// (Ref) NCP region/zone info. Ex) Region: "KR", Zone: "KR-2"
		Region: irs.RegionInfo{
			Region: *NcpInstance.RegionCode,
			Zone:   *NcpInstance.ZoneCode,
		},

		ImageIId: irs.IID{
			NameId:   *NcpInstance.ServerImageNo,
			SystemId: *NcpInstance.ServerImageNo,
		},

		VMSpecName:       	ncloud.StringValue(NcpInstance.ServerSpecCode), // Old : ~.ServerProductCode
		VpcIID:           	irs.IID{SystemId: *NcpInstance.VpcNo},          // Cauton!!) 'NameId: "N/A"' makes an Error on CB-Spider
		SubnetIID:        	irs.IID{SystemId: *NcpInstance.SubnetNo},       // Cauton!!) 'NameId: "N/A"' makes an Error on CB-Spider
		KeyPairIId:       	irs.IID{NameId: *NcpInstance.LoginKeyName, SystemId: *NcpInstance.LoginKeyName},
		NetworkInterface: 	*netInterfaceName,
		PublicIP:         	*publicIp,
		PrivateIP:        	*privateIp,
		RootDiskType:     	rootDiskType,
		SSHAccessPoint:   	*publicIp + ":22",
		KeyValueList:   	irs.StructToKeyValueList(NcpInstance),
	}

	// Get SecurityGroupInfo from DB
	sgInfo, getSGErr := sim.GetSecurityGroup(*NcpInstance.ServerInstanceNo)
	if getSGErr != nil {
		cblogger.Debug(getSGErr)
		// return irs.VMInfo{}, getSGErr
	}
	securityHandler := NcpVpcSecurityHandler{
		RegionInfo: vmHandler.RegionInfo,
		VMClient:   vmHandler.VMClient,
	}
	if countSgKvList(*sgInfo) > 0 {
		var sgIIDs []irs.IID
		for _, kv := range sgInfo.KeyValueInfoList {
			sgInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: kv.Value})
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the S/G info : [%v]", err)
				cblogger.Debug(newErr.Error())
				// return irs.VMInfo{}, newErr
			}
			sgIIDs = append(sgIIDs, irs.IID{NameId: sgInfo.IId.NameId, SystemId: kv.Value})
		}
		vmInfo.SecurityGroupIIds = sgIIDs
	}

	// Set the VM Image Info
	imageHandler := NcpVpcImageHandler{
		RegionInfo: vmHandler.RegionInfo,
		VMClient:   vmHandler.VMClient,
	}
	if !strings.EqualFold(*NcpInstance.ServerImageNo, "") {
		isPublicImage, err := imageHandler.isPublicImage(*NcpInstance.ServerImageNo) // Caution!! : Not '*NcpInstance.ServerImageProductCode'
		if err != nil {
			newErr := fmt.Errorf("Failed to Check Whether the Image is Public Image : [%v]", err)
			cblogger.Debug(newErr.Error())
			
			vmInfo.ImageType = "NA"
			// return irs.VMInfo{}, newErr // Caution!! Consider what happens when an image that was supported in the past is no longer available.
		} else if isPublicImage {
			vmInfo.ImageType = irs.PublicImage
		} else {
			vmInfo.ImageType = irs.MyImage
		}
	}

	_, storageSize, deviceName, err := vmHandler.getVmRootDiskInfo(NcpInstance.ServerInstanceNo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get BlockStorage Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}
	if !strings.EqualFold(*storageSize, "") {
		vmInfo.RootDiskSize = *storageSize
	}
	if !strings.EqualFold(*deviceName, "") {
		vmInfo.RootDeviceName = *deviceName
	}

	dataDiskList, err := vmHandler.getVmDataDiskList(NcpInstance.ServerInstanceNo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Data Disk List : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}
	if len(dataDiskList) > 0 {
		vmInfo.DataDiskIIDs = dataDiskList
	}

	// Note : NCP VPC PlatformType : LNX32, LNX64, WND32, WND64, UBD64, UBS64
	if strings.Contains(*NcpInstance.PlatformType.Code, "LNX") || strings.Contains(*NcpInstance.PlatformType.Code, "UB") {
		vmInfo.VMUserId = lnxUserName
		vmInfo.Platform = irs.LINUX_UNIX
	} else if strings.Contains(*NcpInstance.PlatformType.Code, "WND") {
		vmInfo.VMUserId = winUserName
		vmInfo.Platform = irs.WINDOWS
	}

	return vmInfo, nil
}

func (vmHandler *NcpVpcVMHandler) createLinuxInitScript(imageIID irs.IID, keyPairId string) (*string, error) {
	cblogger.Info("NCP VPC Cloud driver: called createLinuxInitScript()!!")

	var originImagePlatform string

	imageHandler := NcpVpcImageHandler{
		RegionInfo: vmHandler.RegionInfo,
		VMClient:   vmHandler.VMClient,
	}
	isPublicImage, err := imageHandler.isPublicImage(imageIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Check Whether the Image is Public Image : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	if isPublicImage {
		ncpImage, err := imageHandler.getNcpVpcImage(imageIID.SystemId)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Image Info from NCP : [%v]", err)
			cblogger.Error(newErr.Error())
			return nil, newErr
		}

		if strings.Contains(strings.ToUpper(*ncpImage.ServerImageName), "UBUNTU") { // Ex) "tensorflow-Ubuntu-20.04-64"
			originImagePlatform = "UBUNTU"
		} else if strings.Contains(strings.ToUpper(*ncpImage.ServerImageName), "CENTOS") {
			originImagePlatform = "CENTOS"
		} else if strings.Contains(strings.ToUpper(*ncpImage.ServerImageName), "ROCKY") {
			originImagePlatform = "ROCKY"
		} else if strings.Contains(strings.ToUpper(*ncpImage.ServerImageName), "WIN") { // Ex) "win-2019-64-en", "mssql(2019std)-win-2016-64-en"
			originImagePlatform = "WINDOWS"
		} else {
			newErr := fmt.Errorf("Failed to Get OriginImageOSPlatform of the Public Image!!")
			cblogger.Error(newErr.Error())
			return nil, newErr
		}
	} else {
		myImageHandler := NcpVpcMyImageHandler{
			RegionInfo: vmHandler.RegionInfo,
			VMClient:   vmHandler.VMClient,
		}
		var getErr error
		originImagePlatform, getErr = myImageHandler.getOriginImageOSPlatform(imageIID)
		if getErr != nil {
			newErr := fmt.Errorf("Failed to Get OriginImageOSPlatform of the My Image : [%v]", getErr)
			cblogger.Error(newErr.Error())
			return nil, newErr
		}
	}

	var initFilePath string
	switch originImagePlatform {
	case "UBUNTU":
		initFilePath = os.Getenv("CBSPIDER_ROOT") + ubuntuCloudInitFilePath
	case "CENTOS":
		initFilePath = os.Getenv("CBSPIDER_ROOT") + centosCloudInitFilePath
	case "ROCKY":
		initFilePath = os.Getenv("CBSPIDER_ROOT") + centosCloudInitFilePath
	default:
		initFilePath = os.Getenv("CBSPIDER_ROOT") + centosCloudInitFilePath
	}
	cblogger.Infof("\n# initFilePath : [%s]", initFilePath)

	openFile, err := os.Open(initFilePath)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find and Open the Cloud-Init File : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Infof("Succeeded in Finding and Opening the Cloud-Init File : ")
	}
	defer openFile.Close()

	cmdStringByte, readErr := io.ReadAll(openFile)
	if readErr != nil {
		newErr := fmt.Errorf("Failed to Read the open file : [%v]", readErr)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	cmdString := string(cmdStringByte)

	// For GetKey()
	strList := []string{
		vmHandler.CredentialInfo.ClientId,
		vmHandler.CredentialInfo.ClientSecret,
	}

	hashString, err := keycommon.GenHash(strList)
	if err != nil {
		newErr := fmt.Errorf("Failed to Generate Hash String : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Get the publicKey from DB // Caution!! ~.KeyPairIID."SystemId"
	keyValue, getKeyErr := keycommon.GetKey("NCP", hashString, keyPairId)
	if getKeyErr != nil {
		newErr := fmt.Errorf("Failed to Get the Public Key from DB : [%v]", getKeyErr)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Set cloud-init script
	cmdString = strings.ReplaceAll(cmdString, "{{username}}", lnxUserName)
	cmdString = strings.ReplaceAll(cmdString, "{{public_key}}", keyValue.Value)
	// cblogger.Info("cmdString : ", cmdString)

	// Create Cloud-Init Script
	// LnxTypeOs string = "LNX" // LNX (LINUX)
	// WinTypeOS string = "WND" // WND (WINDOWS)
	createInitReq := vserver.CreateInitScriptRequest{
		RegionCode:        ncloud.String(vmHandler.RegionInfo.Region),
		InitScriptContent: ncloud.String(cmdString),
		OsTypeCode:        ncloud.String(LnxTypeOs),
	}

	result, err := vmHandler.VMClient.V2Api.CreateInitScript(&createInitReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Linux type Cloud-Init Script : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	if len(result.InitScriptList) < 1 {
		newErr := fmt.Errorf("Failed to Create any Linux type Cloud-Init Script!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Creating Linux type Cloud-Init Script!!")
	}
	return result.InitScriptList[0].InitScriptNo, nil
}

func (vmHandler *NcpVpcVMHandler) createWinInitScript(passWord string) (*string, error) {
	cblogger.Info("NCP VPC Cloud driver: called createInitScript()!!")

	// Preparing for UserData String
	initFilePath := os.Getenv("CBSPIDER_ROOT") + winCloudInitFilePath
	openFile, err := os.Open(initFilePath)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find and Open the Cloud-Init File : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Infof("Succeeded in Finding and Opening the S/G file: ")
	}
	defer openFile.Close()

	cmdStringByte, readErr := io.ReadAll(openFile)
	if readErr != nil {
		newErr := fmt.Errorf("Failed to Read the open file : [%v]", readErr)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	cmdString := string(cmdStringByte)

	// Set cloud-init script
	cmdString = strings.ReplaceAll(cmdString, "{{PASSWORD}}", passWord)
	// cblogger.Info("cmdString : ", cmdString)

	// Create Cloud-Init Script
	// LnxTypeOs string = "LNX" // LNX (LINUX)
	// WinTypeOS string = "WND" // WND (WINDOWS)
	createInitReq := vserver.CreateInitScriptRequest{
		RegionCode:        ncloud.String(vmHandler.RegionInfo.Region),
		InitScriptContent: ncloud.String(cmdString),
		OsTypeCode:        ncloud.String(WinTypeOS),
	}
	result, err := vmHandler.VMClient.V2Api.CreateInitScript(&createInitReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Windows Cloud-Init Script : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	if len(result.InitScriptList) < 1 {
		newErr := fmt.Errorf("Failed to Create any Windows Cloud-Init Script!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Creating Windows Cloud-Init Script!!")
	}
	return result.InitScriptList[0].InitScriptNo, nil
}

func (vmHandler *NcpVpcVMHandler) deleteInitScript(initScriptNum *string) (*string, error) {
	cblogger.Info("NCP VPC Cloud driver: called deleteInitScript()!!")

	// Delete Cloud-Init Script with the No.
	InitScriptNums := []*string{initScriptNum}
	deleteInitReq := vserver.DeleteInitScriptsRequest{
		RegionCode:       ncloud.String(vmHandler.RegionInfo.Region),
		InitScriptNoList: InitScriptNums,
	}
	result, err := vmHandler.VMClient.V2Api.DeleteInitScripts(&deleteInitReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Cloud-Init Script : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if !strings.EqualFold(*result.ReturnMessage, "success") {
		newErr := fmt.Errorf("Failed to Delete any Cloud-Init Script!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Deleting the Cloud-Init Script!!")
	}
	return result.ReturnMessage, nil
}

// Waiting for up to 500 seconds until VM info. can be Get
func (vmHandler *NcpVpcVMHandler) WaitToGetVMInfo(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("===> As VM info. cannot be retrieved immediately after VM creation, it waits until running.")

	curRetryCnt := 0
	maxRetryCnt := 500

	for {
		curStatus, statusErr := vmHandler.GetVMStatus(vmIID)
		if statusErr != nil {
			cblogger.Errorf("Failed to Get the VM Status of [%s]", vmIID.SystemId)
			cblogger.Error(statusErr.Error())
		} else {
			cblogger.Infof("==> VM Status : [%s]", curStatus)
		}

		switch string(curStatus) {
		case "Creating":
			curRetryCnt++
			cblogger.Infof("The VM is 'Creating', so wait for a second more before inquiring the VM info.")
			time.Sleep(time.Second * 5)
			if curRetryCnt > maxRetryCnt {
				cblogger.Errorf("Despite waiting for a long time(%d sec), the VM status is '%s', so it is forcibly finishied.", maxRetryCnt, curStatus)
				return irs.VMStatus("Failed"), errors.New("Despite waiting for a long time, the VM status is 'Creating', so it is forcibly finishied.")
			}

		default:
			cblogger.Infof("===>The VM status not 'Creating', stopping the waiting.")
			return irs.VMStatus(curStatus), nil
		}
	}
}

// Waiting for up to 600 seconds until Public IP can be Deleted.
func (vmHandler *NcpVpcVMHandler) WaitToDelPublicIp(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("======> As Public IP cannot be deleted immediately after VM termination call, it waits until termination is finished.")

	curRetryCnt := 0
	maxRetryCnt := 600

	for {
		curStatus, statusErr := vmHandler.GetVMStatus(vmIID)
		if statusErr != nil {
			newErr := fmt.Errorf("Failed to Get the VM Status with the VM ID : [%s], [%v]", vmIID.SystemId, statusErr)
			cblogger.Debug(newErr.Error())
			return irs.VMStatus("Not Exist!!"), newErr
		} else {
			cblogger.Infof("==> VM Status : [%s]", curStatus)
		}

		switch string(curStatus) {
		case "Suspended", "Terminating":
			curRetryCnt++
			cblogger.Infof("The VM is still 'Terminating', so wait for a second more before inquiring the VM info.")
			time.Sleep(time.Second * 5)
			if curRetryCnt > maxRetryCnt {
				cblogger.Errorf("Despite waiting for a long time(%d sec), the VM status is '%s', so it is forcibly finishied.", maxRetryCnt, curStatus)
				return irs.VMStatus("Failed"), errors.New("Despite waiting for a long time, the VM status is 'Creating', so it is forcibly finishied.")
			}

		case "Not Exist!!":
			return irs.VMStatus(curStatus), nil

		default:
			cblogger.Infof("===>### The VM Termination is finished, so stopping the waiting.")
			return irs.VMStatus(curStatus), nil
		}
	}
}

// Waiting for up to 500 seconds until Root disk is fully attached
func (vmHandler *NcpVpcVMHandler) WaitForDiskAttach(vmIID irs.IID) (irs.DiskStatus, error) {
	curRetryCnt := 0
	maxRetryCnt := 500

	for {
		storageNo, _, _, err := vmHandler.getVmRootDiskInfo(&vmIID.SystemId)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get BlockStorage Info : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.DiskStatus("Failed"), newErr
		}

		diskHandler := NcpVpcDiskHandler{
			RegionInfo: vmHandler.RegionInfo,
			VMClient:   vmHandler.VMClient,
		}
		curStatus, err := diskHandler.GetDiskStatus(irs.IID{SystemId: *storageNo,})
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Disk Status : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.DiskStatus("Failed"), newErr
		}

		if !strings.EqualFold(string(curStatus), string(irs.DiskAttached)) {
			curRetryCnt++
			cblogger.Infof("The Root disk is not 'Attached' yet, so wait for a second more before inquiring the VM Termination.")
			time.Sleep(time.Second * 5)
			if curRetryCnt > maxRetryCnt {
				cblogger.Errorf("Despite waiting for a long time(%d sec), the Root disk status is '%s', so it is forcibly finishied.", maxRetryCnt, curStatus)
				return irs.DiskStatus("Failed"), errors.New("Despite waiting for a long time, the Root disk is not 'Attached', so it is forcibly finishied.")
			}
		} else {
			return irs.DiskStatus("Succeeded"), nil
		}		
	}
}

// Whenever a VM is terminated, Delete the public IP that the VM has
func (vmHandler *NcpVpcVMHandler) DeletePublicIP(vmInfo irs.VMInfo) (irs.VMStatus, error) {
	cblogger.Info("NCP VPC Cloud driver: called DeletePublicIP()!")

	var publicIPId string
	for _, keyInfo := range vmInfo.KeyValueList {
		if strings.EqualFold(keyInfo.Key, "PublicIpInstanceNo") {  // Public IP ID 
			publicIPId = keyInfo.Value
			break
		}
	}
	cblogger.Infof("Public IP : [%s]", vmInfo.PublicIP)
	cblogger.Infof("Public IP ID : [%s]", publicIPId)

	//=========================================
	// Wait for that the VM is terminated
	//=========================================
	curStatus, statusErr := vmHandler.WaitToDelPublicIp(vmInfo.IId)
	if statusErr != nil {
		cblogger.Debug(statusErr.Error())
		// return irs.VMStatus("Failed. "), statusErr   // Caution!! For in case 'VM instance does Not Exist' after VM Termination finished
	}
	cblogger.Infof("==> VM status of [%s] : [%s]", vmInfo.IId.NameId, curStatus)

	deleteReq := vserver.DeletePublicIpInstanceRequest{
		RegionCode:         ncloud.String(vmHandler.RegionInfo.Region), // $$$ Caution!!
		PublicIpInstanceNo: ncloud.String(publicIPId),
	}
	result, err := vmHandler.VMClient.V2Api.DeletePublicIpInstance(&deleteReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Public IP of the VM instance. : [%v]", err)
		cblogger.Error(newErr.Error())
		cblogger.Error(*result.ReturnMessage)
		return irs.VMStatus("Failed. "), newErr
	} else {
		cblogger.Infof("Succeeded in Deleting the PublicIP of the instance. : [%s]", vmInfo.PublicIP)
	}

	return irs.VMStatus("Terminating"), nil
}

func (vmHandler *NcpVpcVMHandler) getVmRootDiskInfo(vmId *string) (*string, *string, *string, error) {
	cblogger.Info("NCP VPC Cloud driver: called getVmRootDiskInfo()!!")

	if strings.EqualFold(*vmId, "") {
		newErr := fmt.Errorf("Invalid VM ID!!")
		cblogger.Error(newErr.Error())
		return nil, nil, nil, newErr
	}

	storageReq := vserver.GetBlockStorageInstanceListRequest{
		RegionCode:       ncloud.String(vmHandler.RegionInfo.Region),
		ServerInstanceNo: vmId,
	}
	storageResult, err := vmHandler.VMClient.V2Api.GetBlockStorageInstanceList(&storageReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Block Storage List!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, nil, nil, newErr
	}

	if len(storageResult.BlockStorageInstanceList) < 1 {
		cblogger.Info("No BlockStorage Found!!") // Caution) No Error message!!
	} else {
		cblogger.Info("Succeeded in Getting BlockStorage Info!!")
	}

	var storageInstanceNo *string
	var storageSize string
	var deviceName *string
	for _, disk := range storageResult.BlockStorageInstanceList {
		if strings.EqualFold(*disk.ServerInstanceNo, *vmId) && strings.EqualFold(*disk.BlockStorageType.Code, "BASIC") {

			storageInstanceNo = disk.BlockStorageInstanceNo
			storageSize 	  = strconv.FormatFloat(float64(*disk.BlockStorageSize)/(1024*1024*1024), 'f', 0, 64)
			deviceName 		  = disk.DeviceName
			break
		}
	}
	return storageInstanceNo, &storageSize, deviceName, nil
}

func (vmHandler *NcpVpcVMHandler) getVmDataDiskList(vmId *string) ([]irs.IID, error) {
	cblogger.Info("NCP VPC Cloud Driver: called getVmDataDiskList()")

	if strings.EqualFold(*vmId, "") {
		newErr := fmt.Errorf("Invalid VM Instance ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	storageReq := vserver.GetBlockStorageInstanceListRequest{
		RegionCode:       ncloud.String(vmHandler.RegionInfo.Region),
		ServerInstanceNo: vmId,
	}
	storageResult, err := vmHandler.VMClient.V2Api.GetBlockStorageInstanceList(&storageReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Block Storage List!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if len(storageResult.BlockStorageInstanceList) < 1 {
		newErr := fmt.Errorf("Failed to Get any BlockStorage Info!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Getting BlockStorage Info!!")
	}

	var dataDiskIIDList []irs.IID
	for _, disk := range storageResult.BlockStorageInstanceList {
		if strings.EqualFold(*disk.ServerInstanceNo, *vmId) && !strings.EqualFold(*disk.BlockStorageType.Code, "BASIC") {
			dataDiskIIDList = append(dataDiskIIDList, irs.IID{NameId: *disk.BlockStorageName, SystemId: *disk.BlockStorageInstanceNo})
			// break
		}
	}
	return dataDiskIIDList, nil
}

func (vmHandler *NcpVpcVMHandler) getNetworkInterfaceName(netInterfaceNo *string) (*string, error) {
	cblogger.Info("NCP VPC Cloud driver: called getNetworkInterfaceName()!!")

	if strings.EqualFold(*netInterfaceNo, "") {
		newErr := fmt.Errorf("Invalid Net Interface ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	netReq := vserver.GetNetworkInterfaceDetailRequest{
		RegionCode:         ncloud.String(vmHandler.RegionInfo.Region),
		NetworkInterfaceNo: netInterfaceNo,
	}
	netResult, err := vmHandler.VMClient.V2Api.GetNetworkInterfaceDetail(&netReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NetworkInterface Info!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if len(netResult.NetworkInterfaceList) < 1 {
		newErr := fmt.Errorf("Failed to Get any NetworkInterface Info with the Network Interface ID!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Getting NetworkInterface Info!!")
	}

	return netResult.NetworkInterfaceList[0].DeviceName, nil
}

func (vmHandler *NcpVpcVMHandler) getVmIdByName(vmNameId string) (string, error) {
	cblogger.Info("NCP VPC Cloud Driver: called getVmIdByName()")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Region, call.VM, vmNameId, "getVmIdByName()")

	if strings.EqualFold(vmNameId, "") {
		newErr := fmt.Errorf("Invalid VM Instance ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	instanceListReq := vserver.GetServerInstanceListRequest{
		RegionCode: &vmHandler.RegionInfo.Region,
	}
	callLogStart := call.Start()
	instanceResult, err := vmHandler.VMClient.V2Api.GetServerInstanceList(&instanceListReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VM Instance List from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	// Search by Name in the VM list
	var vmId string
	if len(instanceResult.ServerInstanceList) < 1 {
		cblogger.Info("### VM Instance does Not Exist on NCP VPC!!")
	} else {
		cblogger.Info("Succeeded in Getting VM Instance List from NCP VPC.")
		for _, vm := range instanceResult.ServerInstanceList {
			if strings.EqualFold(*vm.ServerName, vmNameId) {
				vmId = *vm.ServerInstanceNo
				break
			}
		}

		if strings.EqualFold(vmId, "") {
			cblogger.Info("### VM Instance does Not Exist with the Name!!")
		}
	}
	return vmId, nil
}

func (vmHandler *NcpVpcVMHandler) getNcpVMInfo(instanceId string) (*vserver.ServerInstance, error) {
	cblogger.Info("NCP VPC Cloud Driver: called getNcpVMInfo()")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Region, call.VM, instanceId, "getNcpVMInfo()")

	if strings.EqualFold(instanceId, "") {
		newErr := fmt.Errorf("Invalid VM Instance ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	instanceReq := vserver.GetServerInstanceDetailRequest{
		RegionCode:       &vmHandler.RegionInfo.Region,
		ServerInstanceNo: &instanceId, // *** Required (Not Optional)
	}
	callLogStart := call.Start()
	instanceResult, err := vmHandler.VMClient.V2Api.GetServerInstanceDetail(&instanceReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find the VM Instance Info from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(instanceResult.ServerInstanceList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any NCP VPC VM Instance Info!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Getting NCP VPC VM Instance Info.")
	}

	return instanceResult.ServerInstanceList[0], nil
}

func (vmHandler *NcpVpcVMHandler) GetRootPassword(vmId *string, privateKey *string) (*string, error) {
	cblogger.Info("NCP VPC Cloud driver: called GetRootPassword()!!")

	if strings.EqualFold(*vmId, "") {
		newErr := fmt.Errorf("Invalid VM Instance ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	pwdReq := vserver.GetRootPasswordRequest{
		RegionCode:       ncloud.String(vmHandler.RegionInfo.Region),
		ServerInstanceNo: vmId,
		PrivateKey:       privateKey,
	}
	result, err := vmHandler.VMClient.V2Api.GetRootPassword(&pwdReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Root Password of the VM!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	if strings.EqualFold(*result.RootPassword, "") {
		newErr := fmt.Errorf("Failed to Get the Root Password of the VM!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Getting the Root Password of the VM!!")
	}

	return result.RootPassword, nil
}

func countSgKvList(sg sim.SecurityGroupInfo) int {
	if sg.KeyValueInfoList == nil {
		return 0
	}
	return len(sg.KeyValueInfoList)
}

func (vmHandler *NcpVpcVMHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("NCP VPC Cloud driver: called vmHandler ListIID()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "ListIID()", "ListIID()")

	instanceReq := vserver.GetServerInstanceListRequest{
		RegionCode:           ncloud.String(vmHandler.RegionInfo.Region), // $$$ Caution!!
		ServerInstanceNoList: []*string{},
	}
	callLogStart := call.Start()
	result, err := vmHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VM Instance List from NCP VPC!! : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var iidList []*irs.IID
	if len(result.ServerInstanceList) < 1 {
		cblogger.Debug("### VM does Not Exist!!")
		return nil, nil
	} else {
		for _, vm := range result.ServerInstanceList {
			var iid irs.IID
			iid.NameId = *vm.ServerName
			iid.SystemId = *vm.ServerInstanceNo

			iidList = append(iidList, &iid)
		}
	}
	return iidList, nil
}
