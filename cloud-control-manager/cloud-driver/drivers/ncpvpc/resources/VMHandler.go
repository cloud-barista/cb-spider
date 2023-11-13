// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP VPC VM Handler
//
// by ETRI, 2020.12.
// by ETRI, 2022.02. updated
//==================================================================================================

package resources

import (
	"errors"
	"fmt"
	// "reflect"
	"strings"
	"strconv"
	"time"
	"os"
	"io"
	// "github.com/davecgh/go-spew/spew"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	keycommon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
)

type NcpVpcVMHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *vserver.APIClient
}

const (
	lnxUserName string = "cb-user"
	winUserName string = "Administrator"
	ubuntuCloudInitFilePath string = "/cloud-driver-libs/.cloud-init-ncpvpc/cloud-init"
	centosCloudInitFilePath string = "/cloud-driver-libs/.cloud-init-ncpvpc/cloud-init-centos"
	winCloudInitFilePath string = "/cloud-driver-libs/.cloud-init-ncpvpc/cloud-init-windows"
	LnxTypeOs string = "LNX" // LNX (LINUX)
	WinTypeOS string = "WND" // WND (WINDOWS)
)

// Already declared in CommonNcpFunc.go
// var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP VPC VMHandler")
}

func (vmHandler *NcpVpcVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	cblogger.Info("NCPVPC Cloud driver: called StartVM()!!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vmHandler.CredentialInfo.ClientId, call.VM, vmReqInfo.IId.NameId, "StartVM()")

	if strings.EqualFold(vmReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid VM Name required")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	// CAUTION!! : Instance Name is Convert to lowercase.(strings.ToLower())
	// NCP VPC에서는 VM instance 이름에 영문 대문자 허용 안되므로 여기서 소문자로 변환하여 반영.(대문자 : Error 발생)
	instanceName := strings.ToLower(vmReqInfo.IId.NameId)
	instanceType := vmReqInfo.VMSpecName
	keyPairId := vmReqInfo.KeyPairIID.SystemId
	vpcId := vmReqInfo.VpcIID.SystemId
	subnetId := vmReqInfo.SubnetIID.SystemId
	minCount := ncloud.Int32(1)

	var publicImageId string
	var myImageId string
	var initScriptNo *string

	if vmReqInfo.ImageType == irs.PublicImage || vmReqInfo.ImageType == "" || vmReqInfo.ImageType == "default" {
		imageHandler := NcpVpcImageHandler{
			RegionInfo:  vmHandler.RegionInfo,
			VMClient:    vmHandler.VMClient,
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
		} else {
			publicImageId = vmReqInfo.ImageIID.SystemId
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
			initScriptNo, createErr = vmHandler.CreateWinInitScript(vmReqInfo.VMUserPasswd)
			if createErr != nil {
				newErr := fmt.Errorf("Failed to Create Cloud-Init Script with the Password : [%v]", createErr)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
		} else {
			var createErr error
			initScriptNo, createErr = vmHandler.CreateLinuxInitScript(vmReqInfo.ImageIID, keyPairId)
			if createErr != nil {
				newErr := fmt.Errorf("Failed to Create Cloud-Init Script with the KeyPairId : [%v]", createErr)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
		}
	} else {
		imageHandler := NcpVpcImageHandler{
			RegionInfo:  vmHandler.RegionInfo,
			VMClient:    vmHandler.VMClient,
		}
		isPublicImage, err := imageHandler.isPublicImage(vmReqInfo.ImageIID)
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
		}
		
		myImageHandler := NcpVpcMyImageHandler{
			RegionInfo:  vmHandler.RegionInfo,
			VMClient:    vmHandler.VMClient,
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
			initScriptNo, createErr = vmHandler.CreateWinInitScript(vmReqInfo.VMUserPasswd)
			if createErr != nil {
				newErr := fmt.Errorf("Failed to Create Cloud-Init Script with the Password : [%v]", createErr)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
		} else {
			var createErr error
			initScriptNo, createErr = vmHandler.CreateLinuxInitScript(vmReqInfo.ImageIID, keyPairId)
			if createErr != nil {
				newErr := fmt.Errorf("Failed to Create Cloud-Init Script with the KeyPairId : [%v]", createErr)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			}
		}
	}
	cblogger.Infof("Init Script No : [%s]", *initScriptNo)

	// Check whether the VM name exists
	// Search by instanceName converted to lowercase
	vmId, getErr := vmHandler.GetVmIdByName(instanceName)
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

	//=========================================================
	// Security Group IID(SystemId 기반) 변환 -> []*string 으로
	//=========================================================
	cblogger.Info("Convert : Security Group IID -> []*string")
	var securityGroupIds []*string

	for _, sgID := range vmReqInfo.SecurityGroupIIDs {
		cblogger.Infof("Security Group IID : [%s]", sgID)
		securityGroupIds = append(securityGroupIds, ncloud.String(sgID.SystemId))
	}

	type intType struct {
		nicOrder *int32
	}

	temp := int32(0) // Convert Int data type to Int32 !!
	i32 := intType{
		nicOrder: &temp,
	}
	fmt.Println(*i32.nicOrder)

	//=========================================================
	// VM Creation info. setting
	//=========================================================
	cblogger.Info("# Start to Create NCP VPC VM Instance!!")
	cblogger.Info("Preparation of CreateServerInstancesRequest!!")

	instanceReq := vserver.CreateServerInstancesRequest{
		RegionCode: 					ncloud.String(vmHandler.RegionInfo.Region),
		ServerName:             		ncloud.String(instanceName),
		ServerImageProductCode: 		ncloud.String(publicImageId),
		MemberServerImageInstanceNo:	ncloud.String(myImageId),
		ServerProductCode:      		ncloud.String(instanceType),
		ServerDescription:  			ncloud.String(vmReqInfo.ImageIID.SystemId), // Caution!!
		LoginKeyName:           		ncloud.String(keyPairId),
		VpcNo:    						ncloud.String(vpcId),
		SubnetNo: 						ncloud.String(subnetId),

		// ### Caution!! : AccessControlGroup은 NCPVPC console의 VPC > 'Network ACL'이 아닌 Server > 'ACG'에 해당됨.
		NetworkInterfaceList: 		[]*vserver.NetworkInterfaceParameter{
			{ NetworkInterfaceOrder: i32.nicOrder, AccessControlGroupNoList: securityGroupIds}, 
			// NetworkInterfaceNo를 입력하지 않으면 NetworkInterface가 자동 생성되어 적용됨.
		},

		IsProtectServerTermination: ncloud.Bool(false), // NOTE Caution!! : 'true'로 설정하면 API로 Terminate(VM 반환) 제어 안됨.
		ServerCreateCount: 			minCount,
		InitScriptNo: 				initScriptNo,
	}
	// cblogger.Info(instanceReq)

	callLogStart := call.Start()
	runResult, err := vmHandler.VMClient.V2Api.CreateServerInstances(&instanceReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create VM instance : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
			scriptDelResult, err := vmHandler.DeleteInitScript(initScriptNo)
			if err != nil {
				newErr := fmt.Errorf("Failed to Delete the Cloud-Init Script with the initScriptNo : [%s], [%v]", ncloud.StringValue(initScriptNo), err)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.VMInfo{}, newErr
			} 
			cblogger.Infof("DeleteInitScript Result : [%s]", *scriptDelResult)
		return irs.VMInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	newVMIID := irs.IID{SystemId: ncloud.StringValue(runResult.ServerInstanceList[0].ServerInstanceNo)}

	//=========================================
	// Wait for VM information to be inquired
	//=========================================
	cblogger.Infof("# Waitting while Initializing New VM!!")
	time.Sleep(time.Second * 15) // Waitting Before Getting New VM Status Info!!

	curStatus, errStatus := vmHandler.WaitToGetInfo(newVMIID) // # Waitting while Creating VM!!")
	if errStatus != nil {
		cblogger.Error(errStatus.Error())
		LoggingError(callLogInfo, errStatus)
		return irs.VMInfo{}, errStatus
	}
	cblogger.Infof("==> VM [%s] status : [%s]", newVMIID.SystemId, curStatus)
	cblogger.Info("VM Creation Processes are Finished !!")

	scriptDelResult, err := vmHandler.DeleteInitScript(initScriptNo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Cloud-Init Script with the initScriptNo : [%s], [%v]", ncloud.StringValue(initScriptNo), err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	} 
	cblogger.Infof("DeleteInitScript Result : [%s]", *scriptDelResult)

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
	cblogger.Info("NCPVPC Cloud driver: called GetVM()!!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "GetVM()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId required")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	cblogger.Infof("vmID : [%s]", vmIID.SystemId)
	instanceNumList := []*string{ncloud.String(vmIID.SystemId),}

	curStatus, statusErr := vmHandler.GetVMStatus(vmIID)
	if statusErr != nil {
		newErr := fmt.Errorf("Failed to Get the VM Status with the VM ID : [%s], [%v]", vmIID.SystemId, statusErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}

	cblogger.Infof("===> VM Status : [%s]", curStatus)

	// Since it's impossible to get VM info. during Creation, ...
	switch string(curStatus) {
	case "Creating", "Booting":
		cblogger.Infof("The VM status is '%s', so wait for the VM creation before inquiring the info.", string(curStatus))
		return irs.VMInfo{}, errors.New("The VM status is 'Creating' or 'Booting', so wait for the VM creation before inquiring the info. : " + vmIID.SystemId)

	default:
		cblogger.Infof("===> The VM status is not 'Creating' or 'Booting', you can get the VM info.")
	}

	/*
		newVMIID := irs.IID{SystemId: systemId}

		curStatus, errStatus := vmHandler.WaitToGetInfo(newVMIID)
		if errStatus != nil {
			cblogger.Error(errStatus.Error())
			return irs.VMInfo{}, nil
		}
	*/

	instanceReq := vserver.GetServerInstanceListRequest{
		RegionCode: 			ncloud.String(vmHandler.RegionInfo.Region),   // $$$ Caution!!
		ServerInstanceNoList: 	instanceNumList,
	}

	start := call.Start()
	result, err := vmHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find VM Instance List from NCP VPC!! : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	if *result.TotalRows < 1 {
		cblogger.Info("### VM instance does Not Exist!!")
	}

	vmInfo, err := vmHandler.MappingServerInfo(result.ServerInstanceList[0])
	if err != nil {
		newErr := fmt.Errorf("Failed to Map the VM Info!! : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	return vmInfo, nil
}

func (vmHandler *NcpVpcVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCPVPC Cloud driver: called SuspendVM()!!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "SuspendVM()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId required")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMStatus("Failed"), newErr
	}

	cblogger.Infof("vmID : [%s]", vmIID.SystemId)
	serverInstanceNo := []*string{ncloud.String(vmIID.SystemId)}

	var resultStatus string

	cblogger.Info("Start to Get VM Status...")
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		cblogger.Errorf("Failed to Get the VM Status of [%s]", vmIID.SystemId)
		cblogger.Error(err)
	} else {
		cblogger.Infof("Succeed in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, vmStatus)
	}

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
		cblogger.Infof("vmID : [%s]", *serverInstanceNo[0])

		req := vserver.StopServerInstancesRequest{
			RegionCode: 			ncloud.String(vmHandler.RegionInfo.Region),   // $$$ Caution!!
			ServerInstanceNoList: 	serverInstanceNo,
		}

		callLogStart := call.Start()
		runResult, err := vmHandler.VMClient.V2Api.StopServerInstances(&req)
		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(callLogInfo, err)
			return irs.VMStatus("Failed"), err
		}
		LoggingInfo(callLogInfo, callLogStart)
		cblogger.Info(runResult)
	}
	return irs.VMStatus("Suspending"), nil
}

func (vmHandler *NcpVpcVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCPVPC Cloud driver: called ResumeVM()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "ResumeVM()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId required")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMStatus("Failed"), newErr
	}

	cblogger.Infof("vmID : [%s]", vmIID.SystemId)
	serverInstanceNo := []*string{ncloud.String(vmIID.SystemId)}

	var resultStatus string

	cblogger.Info("Start to Get the VM Status...")
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		cblogger.Errorf("Failed to Get the VM Status of [%s]", vmIID.SystemId)
		cblogger.Error(err)
	} else {
		cblogger.Infof("Succeed in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, vmStatus)
	}

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
			RegionCode: 			ncloud.String(vmHandler.RegionInfo.Region),   // $$$ Caution!!
			ServerInstanceNoList: 	serverInstanceNo,
		}

		callLogStart := call.Start()
		runResult, err := vmHandler.VMClient.V2Api.StartServerInstances(&req)
		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(callLogInfo, err)
			return irs.VMStatus("Failed"), err
		}
		LoggingInfo(callLogInfo, callLogStart)
		cblogger.Info(runResult)

		return irs.VMStatus("Resuming"), nil
	}
}

func (vmHandler *NcpVpcVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCPVPC Cloud driver: called RebootVM()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "RebootVM()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId required")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMStatus("Failed"), newErr
	}

	cblogger.Infof("vmID : [%s]", vmIID.SystemId)
	serverInstanceNo := []*string{ncloud.String(vmIID.SystemId)}

	var resultStatus string

	cblogger.Info("Start to Get the VM Status...")
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		cblogger.Errorf("Failed to Get the VM Status of [%s]", vmIID.SystemId)
		cblogger.Error(err)
	} else {
		cblogger.Infof("Succeed in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, vmStatus)
	}

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
			RegionCode: 			ncloud.String(vmHandler.RegionInfo.Region),   // $$$ Caution!!
			ServerInstanceNoList: 	serverInstanceNo,
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
		cblogger.Info(runResult)

		return irs.VMStatus("Rebooting"), nil
	}
}

func (vmHandler *NcpVpcVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCPVPC Cloud driver: called TerminateVM()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "TerminateVM()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId required")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMStatus("Failed"), newErr
	}

	cblogger.Infof("vmID : [%s]", vmIID.SystemId)
	serverInstanceNos := []*string{ncloud.String(vmIID.SystemId),}

	cblogger.Info("Start to Get the VM Status...")
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		LoggingError(callLogInfo, err)
		cblogger.Errorf("Failed to Get the VM Status of [%s]", vmIID.SystemId)
		cblogger.Error(err)
	} else {
		cblogger.Infof("Succeed in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, vmStatus)
	}

	vmInfo, error := vmHandler.GetVM(vmIID)
	if error != nil {
		LoggingError(callLogInfo, error)
		cblogger.Error(error.Error())

		return irs.VMStatus("Failed to get the VM info"), err
	}

	switch string(vmStatus) {
	case "Suspended":
		cblogger.Infof("VM Status : 'Suspended'. so it Can be Terminated!!")

		req := vserver.TerminateServerInstancesRequest{
			RegionCode: 			ncloud.String(vmHandler.RegionInfo.Region),   // $$$ Caution!!
			ServerInstanceNoList: 	serverInstanceNos,
		}

		start := call.Start()
		runResult, err := vmHandler.VMClient.V2Api.TerminateServerInstances(&req)
		if err != nil {
			newErr := fmt.Errorf("Failed to Terminate the VM instance on NCP VPC. : [%v]", err)
			cblogger.Error(newErr.Error())
			cblogger.Error(*runResult.ReturnMessage)
			LoggingError(callLogInfo, newErr)
			return irs.VMStatus("Failed to Terminate!!"), newErr
		}
		LoggingInfo(callLogInfo, start)
		cblogger.Info(runResult)

		// If the NCP instance has a 'Public IP', delete it after termination of the instance.
		if ncloud.String(vmInfo.PublicIP) != nil {
			// PublicIP 삭제
			vmStatus, err := vmHandler.DeletePublicIP(vmInfo)
			if err != nil {
				cblogger.Error(err)
				return vmStatus, err
			}
		}

		return irs.VMStatus("Terminating"), nil

	case "Running":
		cblogger.Infof("VM Status : 'Running'. so it Can be Terminated AFTER SUSPENSION !!")
		cblogger.Infof("vmID : [%s]", *serverInstanceNos[0])

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
			curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
			if errStatus != nil {
				cblogger.Error(errStatus.Error())
			}

			cblogger.Infof("===> VM Status : [%s]", curStatus)
			if curStatus != irs.VMStatus("Suspended") {
				curRetryCnt++
				cblogger.Infof("The VM is not 'Suspended' yet, so wait for a second more before inquiring Termination.")
				time.Sleep(time.Second * 3)
				if curRetryCnt > maxRetryCnt {
					cblogger.Errorf("Despite waiting for a long time(%d sec), the VM is not 'suspended', so it is forcibly terminated.", maxRetryCnt)
				}
			} else {
				break
			}
		}

		cblogger.Info("# SuspendVM() Finished")

		req := vserver.TerminateServerInstancesRequest{
			RegionCode: 			ncloud.String(vmHandler.RegionInfo.Region),   // $$$ Caution!!
			ServerInstanceNoList: 	serverInstanceNos,
		}

		callLogStart := call.Start()
		runResult, err := vmHandler.VMClient.V2Api.TerminateServerInstances(&req)
		if err != nil {
			newErr := fmt.Errorf("Failed to Terminate the VM instance on NCP VPC. : [%v]", err)
			cblogger.Error(newErr.Error())
			cblogger.Error(*runResult.ReturnMessage)
			LoggingError(callLogInfo, newErr)
			return irs.VMStatus("Failed to Terminate!!"), newErr
		}
		LoggingInfo(callLogInfo, callLogStart)
		cblogger.Info(runResult)

		// If the NCP instance has a 'Public IP', delete it after termination of the instance.
		if ncloud.String(vmInfo.PublicIP) != nil {
			// PublicIP 삭제
			vmStatus, err := vmHandler.DeletePublicIP(vmInfo)
			if err != nil {
				cblogger.Error(err)
				return vmStatus, err
			}
		}

		return irs.VMStatus("Terminating"), nil

	default:
		resultStatus := "The VM status is not 'Running' or 'Suspended' yet. so it Can NOT be Terminated!! Run or Suspend the VM before terminating."

		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
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

func ConvertVMStatusString(vmStatus string) (irs.VMStatus, error) {
	var resultStatus string

	cblogger.Infof("NCP VPC vmStatus to Convert : [%s]", vmStatus)

	if strings.EqualFold(vmStatus, "creating") {
		resultStatus = "Creating"
	} else if strings.EqualFold(vmStatus, "init") {
		resultStatus = "Creating"
	} else if strings.EqualFold(vmStatus, "booting") {
		//Caution!!
		resultStatus = "Booting"
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

	cblogger.Infof("VM Status Conversion Completed : [%s] ==> [%s]", vmStatus, resultStatus)

	return irs.VMStatus(resultStatus), nil
}

func (vmHandler *NcpVpcVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCPVPC Cloud driver: called GetVMStatus()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "GetVMStatus()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId required")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMStatus("Failed"), newErr
	}

	cblogger.Infof("VM SystemId : [%s]", vmIID.SystemId)
	systemId := vmIID.SystemId

	// instanceReq := vserver.GetServerInstanceListRequest{
	// 	ServerInstanceNoList: []*string{nil},
	// }
	instanceReq := vserver.GetServerInstanceListRequest{
		RegionCode: 			ncloud.String(vmHandler.RegionInfo.Region),   // $$$ Caution!!
		ServerInstanceNoList: 	[]*string{
			ncloud.String(systemId),
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

	if *result.TotalRows < 1 {
		cblogger.Info("The VM instance does Not Exist!!")
		return irs.VMStatus("Not Exist!!"), nil //Caution!!
	} else {
		cblogger.Info("Succeeded in Getting ServerInstanceList from NCP VPC!!")
	}

	for _, vm := range result.ServerInstanceList {
		//*vm.ServerInstanceStatusName
		vmStatus, errStatus := ConvertVMStatusString(*vm.ServerInstanceStatusName)
		cblogger.Infof("VM Status of [%s] : [%s]", systemId, vmStatus)
		return vmStatus, errStatus
	}

	return irs.VMStatus("Failed."), errors.New("Failed to Get the VM Status info!!")
}

func (vmHandler *NcpVpcVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	cblogger.Info("NCPVPC Cloud driver: called ListVMStatus()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "ListVMStatus()", "ListVMStatus()")

	var vmStatusList []*irs.VMStatusInfo

	instanceReq := vserver.GetServerInstanceListRequest{
		RegionCode: 		  ncloud.String(vmHandler.RegionInfo.Region),   // $$$ Caution!!
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

	for _, vm := range result.ServerInstanceList {
		//*vm.ServerInstanceStatusName
		//*vm.ServerName
		vmStatus, _ := ConvertVMStatusString(*vm.ServerInstanceStatusName)

		vmStatusInfo := irs.VMStatusInfo{
			IId:      irs.IID{
				NameId:	 	*vm.ServerName,
				SystemId: 	*vm.ServerInstanceNo,
			},
			VmStatus: vmStatus,
		}
		cblogger.Infof("VM Status of [%s] : [%s]", vmStatusInfo.IId.SystemId, vmStatusInfo.VmStatus)
		vmStatusList = append(vmStatusList, &vmStatusInfo)
	}

	return vmStatusList, nil
}

func (vmHandler *NcpVpcVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger.Info("NCPVPC Cloud driver: called ListVM()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "ListVMS()", "ListVM()")

	var vmInfoList []*irs.VMInfo

	instanceReq := vserver.GetServerInstanceListRequest{
		RegionCode: 			ncloud.String(vmHandler.RegionInfo.Region),   // $$$ Caution!!
		ServerInstanceNoList: []*string{},
		//		ServerInstanceNoList: []*string{
		//			nil,
		//		},
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

	cblogger.Info("Succeeded in Getting ServerInstanceList from NCP VPC!!")

	for _, vm := range result.ServerInstanceList {
		cblogger.Infof("Inquiry of NCP VM Instance info : [%s]", *vm.ServerInstanceNo)

		curStatus, errStatus := vmHandler.GetVMStatus(irs.IID{SystemId: *vm.ServerInstanceNo})
		if errStatus != nil {
			cblogger.Errorf("Failed to Get the VM Status of VM : [%s]", *vm.ServerInstanceNo)
			cblogger.Error(errStatus.Error())
		} else {
			cblogger.Infof("Succeed in Getting the VM Status of [%s] : [%s]", *vm.ServerInstanceNo, curStatus)
		}

		cblogger.Infof("===> VM Status : [%s]", curStatus)

		switch string(curStatus) {
		case "Creating", "Booting":
			return []*irs.VMInfo{}, nil

		default:
			cblogger.Infof("===> The VM status not 'Creating' or 'Booting', you can get the VM info.")
			vmInfo, error := vmHandler.GetVM(irs.IID{SystemId: *vm.ServerInstanceNo})
			if error != nil {
				cblogger.Error(error.Error())
				return []*irs.VMInfo{}, error
			}

			vmInfoList = append(vmInfoList, &vmInfo)
		}
	}

	return vmInfoList, nil
}

func (vmHandler *NcpVpcVMHandler) MappingServerInfo(NcpInstance *vserver.ServerInstance) (irs.VMInfo, error) {
	cblogger.Info("NCPVPC Cloud driver: called MappingServerInfo()!")

	var publicIp *string
	var privateIp *string
	var publicIpInstanceNo *string

	// cblogger.Infof("# NcpInstance Info :")
	// spew.Dump(NcpInstance)

	convertedTime, err := convertTimeFormat(*NcpInstance.CreateDate)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert the Time Format!!")
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}

	// Create a PublicIp, if the instance doesn't have a 'Public IP' after creation.
	if strings.EqualFold(ncloud.StringValue(NcpInstance.PublicIp), "") {
		publicIpReq := vserver.CreatePublicIpInstanceRequest{
			ServerInstanceNo: 	NcpInstance.ServerInstanceNo,
			RegionCode: 		ncloud.String(vmHandler.RegionInfo.Region),
		}

		// CAUTION!! : The number of Public IPs cannot be more than the number of instances on NCP cloud default service.
		result, err := vmHandler.VMClient.V2Api.CreatePublicIpInstance(&publicIpReq)
		if err != nil {
			newErr := fmt.Errorf("Failed to Create Public IP : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
		if *result.TotalRows < 1 {
			newErr := fmt.Errorf("Failed to Create Any Public IP!!")
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}

		publicIp = result.PublicIpInstanceList[0].PublicIp
		publicIpInstanceNo = result.PublicIpInstanceList[0].PublicIpInstanceNo
		privateIp = result.PublicIpInstanceList[0].PrivateIp

		cblogger.Infof("*** PublicIp : %s ", ncloud.StringValue(publicIp))
		cblogger.Infof("Finished to Create Public IP")
	} else {
		publicIp = NcpInstance.PublicIp
		cblogger.Infof("*** NcpInstance.PublicIp : %s ", ncloud.StringValue(publicIp))

		instanceReq := vserver.GetPublicIpInstanceListRequest{
			RegionCode: ncloud.String(vmHandler.RegionInfo.Region), // $$$ Caution!!
			PublicIp: 	publicIp,
		}

		// Search the Public IP list info. to get the PublicIp InstanceNo
		result, err := vmHandler.VMClient.V2Api.GetPublicIpInstanceList(&instanceReq)
		if err != nil {
			newErr := fmt.Errorf("Failed to Find PublicIp InstanceList from NCP VPC : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
		if *result.TotalRows < 1 {
			newErr := fmt.Errorf("Failed to Find Any PublicIpInstance!!")
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}

		publicIpInstanceNo = result.PublicIpInstanceList[0].PublicIpInstanceNo
		privateIp = result.PublicIpInstanceList[0].PrivateIp

		cblogger.Infof("Finished to Get PublicIP InstanceNo")
	}

	netInterfaceName, err := vmHandler.GetNetworkInterfaceName(NcpInstance.NetworkInterfaceNoList[0])
	if err != nil {
		newErr := fmt.Errorf("Failed to Find NetworkInterface Name : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}

	// To Get the VM resources Info.
	// PublicIpID : To use it when delete the PublicIP
	vmInfo := irs.VMInfo {
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

		VMSpecName: ncloud.StringValue(NcpInstance.ServerProductCode), //Server Spec code

		VpcIID:    irs.IID{NameId: "N/A", SystemId: *NcpInstance.VpcNo},
		SubnetIID: irs.IID{NameId: "N/A", SystemId: *NcpInstance.SubnetNo},

		SecurityGroupIIds: []irs.IID{
			{NameId: "N/A", SystemId: "N/A"},
		},

		KeyPairIId: 	irs.IID{NameId: *NcpInstance.LoginKeyName, SystemId: *NcpInstance.LoginKeyName},
		NetworkInterface: *netInterfaceName, 
		PublicIP:   	  *publicIp,
		PrivateIP:  	  *privateIp,
		RootDiskType: 	  *NcpInstance.BaseBlockStorageDiskDetailType.CodeName,
		SSHAccessPoint:   *publicIp + ":22",

		KeyValueList: []irs.KeyValue{
			{Key: "ServerInstanceType", Value: *NcpInstance.ServerInstanceType.CodeName},
			{Key: "CpuCount", Value: String(*NcpInstance.CpuCount)},
			{Key: "MemorySize(GB)", Value: strconv.FormatFloat(float64(*NcpInstance.MemorySize)/(1024*1024*1024), 'f', 0, 64)},
			{Key: "PlatformType", Value: *NcpInstance.PlatformType.CodeName},
			{Key: "PublicIpID", Value: *publicIpInstanceNo},
		},
	}

	imageHandler := NcpVpcImageHandler{
		RegionInfo:  vmHandler.RegionInfo,
		VMClient:    vmHandler.VMClient,
	}
	
	// Set the VM Image Info
	if !strings.EqualFold(*NcpInstance.ServerDescription, "") {
		vmInfo.ImageIId.SystemId = *NcpInstance.ServerDescription // Note!! : Since MyImage ID is not included in the 'NcpInstance' info 
		vmInfo.ImageIId.NameId = *NcpInstance.ServerDescription
		
		isPublicImage, err := imageHandler.isPublicImage(irs.IID{SystemId: *NcpInstance.ServerDescription}) // Caution!! : Not '*NcpInstance.ServerImageProductCode'
		if err != nil {
			newErr := fmt.Errorf("Failed to Check Whether the Image is Public Image : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
		if isPublicImage {
			vmInfo.ImageType = irs.PublicImage
		} else {
			vmInfo.ImageType = irs.MyImage
		}
	} else {
		vmInfo.ImageIId.SystemId = *NcpInstance.ServerImageProductCode
		vmInfo.ImageIId.NameId = *NcpInstance.ServerImageProductCode
	}
	
	storageSize, deviceName, err := vmHandler.GetVmRootDiskInfo(NcpInstance.ServerInstanceNo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find BlockStorage Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}
	if !strings.EqualFold(*storageSize, "") {
		vmInfo.RootDiskSize = *storageSize
	}
	if !strings.EqualFold(*deviceName, "") {
		vmInfo.RootDeviceName = *deviceName
	}

	dataDiskList, err := vmHandler.GetVmDataDiskList(NcpInstance.ServerInstanceNo)
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

func (vmHandler *NcpVpcVMHandler) CreateLinuxInitScript(imageIID irs.IID, keyPairId string) (*string, error) {
	cblogger.Info("NCPVPC Cloud driver: called CreateLinuxInitScript()!!")

	var originImagePlatform string

	imageHandler := NcpVpcImageHandler{
		RegionInfo:  vmHandler.RegionInfo,
		VMClient:    vmHandler.VMClient,
	}
	isPublicImage, err := imageHandler.isPublicImage(imageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Check Whether the Image is Public Image : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}	
	if isPublicImage {
		if strings.Contains(strings.ToUpper(imageIID.SystemId), "UBNTU") {
			originImagePlatform = "UBUNTU"
		} else if strings.Contains(strings.ToUpper(imageIID.SystemId), "CNTOS") {
			originImagePlatform = "CENTOS"
		} else if strings.Contains(strings.ToUpper(imageIID.SystemId), "ROCKY") {
			originImagePlatform = "ROCKY"
		} else if strings.Contains(strings.ToUpper(imageIID.SystemId), "WND") {
			originImagePlatform = "WINDOWS"
		} else {
			newErr := fmt.Errorf("Failed to Get OriginImageOSPlatform of the Public Image!!")
			cblogger.Error(newErr.Error())
			return nil, newErr
		}
	} else {
		myImageHandler := NcpVpcMyImageHandler{
			RegionInfo:  vmHandler.RegionInfo,
			VMClient:    vmHandler.VMClient,
		}
		var getErr error
		originImagePlatform, getErr = myImageHandler.GetOriginImageOSPlatform(imageIID)
		if getErr != nil {
			newErr := fmt.Errorf("Failed to Get OriginImageOSPlatform of the My Image : [%v]", getErr)
			cblogger.Error(newErr.Error())
			return nil, newErr
		}
	}	

	var initFilePath string
	switch originImagePlatform {
	case "UBUNTU" :
		initFilePath = os.Getenv("CBSPIDER_ROOT") + ubuntuCloudInitFilePath
	case "CENTOS" :
		initFilePath = os.Getenv("CBSPIDER_ROOT") + centosCloudInitFilePath
	case "ROCKY" :
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
	strList:= []string{
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
	keyValue, getKeyErr := keycommon.GetKey("NCPVPC", hashString, keyPairId)
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
	createInitReq := vserver.CreateInitScriptRequest {
		RegionCode: 		ncloud.String(vmHandler.RegionInfo.Region),
		InitScriptContent:	ncloud.String(cmdString),
		OsTypeCode: 		ncloud.String(LnxTypeOs),
	}

	result, err := vmHandler.VMClient.V2Api.CreateInitScript(&createInitReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Linux type Cloud-Init Script : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}	
	if *result.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Create any Linux type Cloud-Init Script!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Creating Linux type Cloud-Init Script!!")
	}
	return result.InitScriptList[0].InitScriptNo, nil
}

func (vmHandler *NcpVpcVMHandler) CreateWinInitScript(passWord string) (*string, error) {
	cblogger.Info("NCPVPC Cloud driver: called createInitScript()!!")

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
	createInitReq := vserver.CreateInitScriptRequest {
		RegionCode: 		ncloud.String(vmHandler.RegionInfo.Region),
		InitScriptContent:	ncloud.String(cmdString),
		OsTypeCode: 		ncloud.String(WinTypeOS),
	}

	result, err := vmHandler.VMClient.V2Api.CreateInitScript(&createInitReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Windows Cloud-Init Script : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}	
	if *result.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Create any Windows Cloud-Init Script!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Creating Windows Cloud-Init Script!!")
	}
	return result.InitScriptList[0].InitScriptNo, nil
}

func (vmHandler *NcpVpcVMHandler) DeleteInitScript(initScriptNum *string) (*string, error) {
	cblogger.Info("NCPVPC Cloud driver: called DeleteInitScript()!!")

	InitScriptNums := []*string{initScriptNum,}

	// Delete Cloud-Init Script
	deleteInitReq := vserver.DeleteInitScriptsRequest {
		RegionCode: 		ncloud.String(vmHandler.RegionInfo.Region),
		InitScriptNoList: 	InitScriptNums,
	}

	result, err := vmHandler.VMClient.V2Api.DeleteInitScripts(&deleteInitReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Cloud-Init Script : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}	
	if *result.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Delete any Cloud-Init Script!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Deleting the Cloud-Init Script!!")
	}
	return result.ReturnMessage, nil
}

// Waiting for up to 500 seconds until VM info. can be Get
func (vmHandler *NcpVpcVMHandler) WaitToGetInfo(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("===> As VM info. cannot be retrieved immediately after VM creation, it waits until running.")

	curRetryCnt := 0
	maxRetryCnt := 500

	for {
		curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
		if errStatus != nil {
			cblogger.Errorf("Failed to Get the VM Status of [%s]", vmIID.SystemId)
			cblogger.Error(errStatus.Error())
		} else {
			cblogger.Infof("Succeeded in Getting the Status of VM [%s] : [%s]", vmIID.SystemId, curStatus)
		}

		cblogger.Infof("===> VM Status : [%s]", curStatus)

		switch string(curStatus) {
		case "Creating", "Booting":

			curRetryCnt++
			cblogger.Infof("The VM is 'Creating', so wait for a second more before inquiring the VM info.")
			time.Sleep(time.Second * 3)
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
		curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
		if errStatus != nil {
			cblogger.Errorf("Failed to Get the VM Status of : [%s]", vmIID.SystemId)
			cblogger.Error(errStatus.Error())
			// return irs.VMStatus("Failed. "), errors.New("Failed to Get the VM Status.")   // Caution!!
		} else {
			cblogger.Infof("Succeeded in Getting the VM Status of [%s]", vmIID.SystemId)
		}

		cblogger.Infof("===> VM Status [%s] : ", curStatus)

		switch string(curStatus) {
		case "Suspended", "Terminating":
			curRetryCnt++
			cblogger.Infof("The VM is still 'Terminating', so wait for a second more before inquiring the VM info.")
			time.Sleep(time.Second * 3)
			if curRetryCnt > maxRetryCnt {
				cblogger.Errorf("Despite waiting for a long time(%d sec), the VM status is '%s', so it is forcibly finishied.", maxRetryCnt, curStatus)
				return irs.VMStatus("Failed"), errors.New("Despite waiting for a long time, the VM status is 'Creating', so it is forcibly finishied.")
			}

		default:
			cblogger.Infof("===>### The VM Termination is finished, so stopping the waiting.")
			return irs.VMStatus(curStatus), nil
		}
	}
}

// Whenever a VM is terminated, Delete the public IP that the VM has
func (vmHandler *NcpVpcVMHandler) DeletePublicIP(vmInfo irs.VMInfo) (irs.VMStatus, error) {
	cblogger.Info("NCPVPC Cloud driver: called DeletePublicIP()!")

	var publicIPId string

	for _, keyInfo := range vmInfo.KeyValueList {
		if keyInfo.Key == "PublicIpID" {
			publicIPId = keyInfo.Value
			break
		}
	}

	cblogger.Infof("vmInfo.PublicIP : [%s]", vmInfo.PublicIP)
	cblogger.Infof("publicIPId : [%s]", publicIPId)

	//=========================================
	// Wait for that the VM is terminated
	//=========================================
	curStatus, errStatus := vmHandler.WaitToDelPublicIp(vmInfo.IId)
	if errStatus != nil {
		cblogger.Error(errStatus.Error())
		// return irs.VMStatus("Failed. "), errStatus   // Caution!!
	}
	cblogger.Infof("==> VM status of [%s] : [%s]", vmInfo.IId.NameId, curStatus)

	deleteReq := vserver.DeletePublicIpInstanceRequest{
		RegionCode: 		ncloud.String(vmHandler.RegionInfo.Region),   // $$$ Caution!!
		PublicIpInstanceNo: ncloud.String(publicIPId),
	}

	cblogger.Infof("DeletePublicIPReq Ready!!")

	result, err := vmHandler.VMClient.V2Api.DeletePublicIpInstance(&deleteReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Public IP of the VM instance. : [%v]", err)
		cblogger.Error(newErr.Error())
		cblogger.Error(*result.ReturnMessage)
		return irs.VMStatus("Failed. "), newErr
	}

	if *result.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Delete any Public IP of the VM instance.")
		cblogger.Error(newErr.Error())
		return irs.VMStatus("Failed. "), newErr
	} else {
		cblogger.Infof("Succeed in Deleting the PublicIP of the instance. : [%s]", vmInfo.PublicIP)
	}

	return irs.VMStatus("Terminating"), nil
}

func (vmHandler *NcpVpcVMHandler) GetVmRootDiskInfo(vmId *string) (*string, *string, error) {
	cblogger.Info("NCPVPC Cloud driver: called GetVmRootDiskInfo()!!")

	if strings.EqualFold(*vmId, "") {
		newErr := fmt.Errorf("Invalid VM ID!!")
		cblogger.Error(newErr.Error())
		return nil, nil, newErr
	}

	storageReq := vserver.GetBlockStorageInstanceListRequest {
		RegionCode: 		ncloud.String(vmHandler.RegionInfo.Region),
		ServerInstanceNo:   vmId,
	}

	storageResult, err := vmHandler.VMClient.V2Api.GetBlockStorageInstanceList(&storageReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Block Storage List!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, nil, newErr
	}

	if *storageResult.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Get any BlockStorage Info!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, nil, newErr
	} else {
		cblogger.Info("Succeeded in Getting BlockStorage Info!!")
	}

	var storageSize string
	var deviceName *string
	for _, disk := range storageResult.BlockStorageInstanceList {
		if strings.EqualFold(*disk.ServerInstanceNo, *vmId) && strings.EqualFold(*disk.BlockStorageType.Code, "BASIC") {
			storageSize = strconv.FormatFloat(float64(*disk.BlockStorageSize)/(1024*1024*1024), 'f', 0, 64)
			deviceName = disk.DeviceName
			break
		}
	}
	return &storageSize, deviceName, nil
}

func (vmHandler *NcpVpcVMHandler) GetVmDataDiskList(vmId *string) ([]irs.IID, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetVmDataDiskList()")

	if strings.EqualFold(*vmId, "") {
		newErr := fmt.Errorf("Invalid VM Instance ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	storageReq := vserver.GetBlockStorageInstanceListRequest {
		RegionCode: 		ncloud.String(vmHandler.RegionInfo.Region),
		ServerInstanceNo:   vmId,
	}
	storageResult, err := vmHandler.VMClient.V2Api.GetBlockStorageInstanceList(&storageReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Block Storage List!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if *storageResult.TotalRows < 1 {
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

func (vmHandler *NcpVpcVMHandler) GetNetworkInterfaceName(netInterfaceNo *string) (*string, error) {
	cblogger.Info("NCPVPC Cloud driver: called GetNetworkInterfaceName()!!")

	if strings.EqualFold(*netInterfaceNo, "") {
		newErr := fmt.Errorf("Invalid Net Interface ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	netReq := vserver.GetNetworkInterfaceDetailRequest {
		RegionCode: 		ncloud.String(vmHandler.RegionInfo.Region),
		NetworkInterfaceNo: netInterfaceNo,
	}
	netResult, err := vmHandler.VMClient.V2Api.GetNetworkInterfaceDetail(&netReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NetworkInterface Info!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if *netResult.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Get any NetworkInterface Info with the Network Interface ID!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Getting NetworkInterface Info!!")
	}

	return netResult.NetworkInterfaceList[0].DeviceName, nil
}

func (vmHandler *NcpVpcVMHandler) GetVmIdByName(vmNameId string) (string, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetVmIdByName()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Region, call.VM, vmNameId, "GetVmIdByName()")

	if strings.EqualFold(vmNameId, "") {
		newErr := fmt.Errorf("Invalid VM Instance ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	instanceListReq := vserver.GetServerInstanceListRequest{
		RegionCode:       &vmHandler.RegionInfo.Region,
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
	if *instanceResult.TotalRows < 1 {
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

func (vmHandler *NcpVpcVMHandler) GetNcpVMInfo(instanceId string) (*vserver.ServerInstance, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetNcpVMInfo()")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Region, call.VM, instanceId, "GetNcpVMInfo()")

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

	if *instanceResult.TotalRows < 1 {
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
	cblogger.Info("NCPVPC Cloud driver: called GetRootPassword()!!")

	if strings.EqualFold(*vmId, "") {
		newErr := fmt.Errorf("Invalid VM Instance ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	pwdReq := vserver.GetRootPasswordRequest {
		RegionCode: 		ncloud.String(vmHandler.RegionInfo.Region),
		ServerInstanceNo: 	vmId,
		PrivateKey: 		privateKey,
	}
	result, err := vmHandler.VMClient.V2Api.GetRootPassword(&pwdReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Root Password of the VM!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	return result.RootPassword, nil
}
