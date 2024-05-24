// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP Classic VM Handler
//
// Created by ETRI, 2020.09.
// Updated by ETRI, 2023.08.
//==================================================================================================

package resources

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	// "github.com/davecgh/go-spew/spew"

	ncloud "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	server "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/server"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	keycommon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVMHandler struct {
	CredentialInfo 	idrv.CredentialInfo
	RegionInfo     	idrv.RegionInfo
	VMClient        *server.APIClient
}

// Global map to hold RegionNo against RegionCode. Initialize the map (with {})
var regionMap = map[string]string{}

// Global map to hold ZoneNo against ZoneCode within a specific RegionNo. Initialize the map (with {})
var regionZoneMap = map[string]map[string]string{}

const (
	ubuntuCloudInitFilePath string 	= "/cloud-driver-libs/.cloud-init-ncp/cloud-init-ubuntu"
	centosCloudInitFilePath string 	= "/cloud-driver-libs/.cloud-init-ncp/cloud-init-centos"
	winCloudInitFilePath string 	= "/cloud-driver-libs/.cloud-init-ncp/cloud-init-windows"

	lnxUserName 	string = "cb-user"
	winUserName 	string = "Administrator"

	LnxTypeOs 		string = "LINUX"
	WinTypeOS 		string = "WINDOWS"
)

// Already declared in CommonNcpFunc.go
// var cblogger *logrus.Logger
func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP Classic VMHandler")
}

func (vmHandler *NcpVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	cblogger.Info("NCP Classic Cloud driver: called StartVM()!!")

	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmReqInfo.IId.NameId, "StartVM()")

	zoneNo, err := vmHandler.GetZoneNo(vmHandler.RegionInfo.Region, vmHandler.RegionInfo.Zone)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Zone 'No' of the Zone : ", err)
		return irs.VMInfo{}, rtnErr
	}

	// CAUTION!! : Instance Name is Converted to lowercase.(strings.ToLower())
	// NCP에서는 VM instance 이름에 영문 대문자 허용 안되므로 여기서 변환하여 반영.(대문자가 포함되면 Error 발생)
	instanceName := strings.ToLower(vmReqInfo.IId.NameId)
	instanceType := vmReqInfo.VMSpecName
	keyPairId := vmReqInfo.KeyPairIID.SystemId
	minCount := ncloud.Int32(1)

	// Check whether the VM name exists. Search by instanceName converted to lowercase
	vmId, err := vmHandler.GetVmIdByName(instanceName)
	if err != nil {
		cblogger.Error("The VM with the name is not exists : " + instanceName)
		// return irs.VMInfo{}, err  //Caution!!
    }
	if vmId != "" {
		cblogger.Info("The vmId : ", vmId)
		createErr := fmt.Errorf("VM with the name '%s' already exist!!", vmReqInfo.IId.NameId)
		LoggingError(callLogInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	// Security Group IID 처리 - SystemId 기반
	cblogger.Info("Security Group IID 변환")
	var newSecurityGroupIds []*string
	for _, sgID := range vmReqInfo.SecurityGroupIIDs {
		cblogger.Infof("Security Group IID : [%s]", sgID)
		newSecurityGroupIds = append(newSecurityGroupIds, ncloud.String(sgID.SystemId))
	}
	cblogger.Info(newSecurityGroupIds)

	// Set cloud-init script
	var publicImageId string
	var myImageId string
	var initUserData *string

	if vmReqInfo.ImageType == irs.PublicImage || vmReqInfo.ImageType == "" || vmReqInfo.ImageType == "default" {
		myImageHandler := NcpMyImageHandler{
			RegionInfo:  vmHandler.RegionInfo,
			VMClient:    vmHandler.VMClient,
		}
		isPublicImage, err := myImageHandler.isPublicImage(vmReqInfo.ImageIID)
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

		isPublicWindowsImage, err := myImageHandler.CheckWindowsImage(vmReqInfo.ImageIID)
		if err != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Check Whether the Image is MS Windows Image : ", err)
			return irs.VMInfo{}, rtnErr
		}
		if isPublicWindowsImage {
			var createErr error
			initUserData, createErr = vmHandler.CreateWinInitUserData(vmReqInfo.VMUserPasswd)
			if createErr != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Create Cloud-Init Script with the Password : ", createErr)
				return irs.VMInfo{}, rtnErr
			}
		} else {
			var createErr error
			initUserData, createErr = vmHandler.CreateLinuxInitUserData(vmReqInfo.ImageIID, keyPairId)
			if createErr != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Create Cloud-Init Script with the KeyPairId : ", createErr)
				return irs.VMInfo{}, rtnErr
			}
		}
	} else {
		myImageHandler := NcpMyImageHandler{
			RegionInfo:  vmHandler.RegionInfo,
			VMClient:    vmHandler.VMClient,
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
		} else {
			myImageId = vmReqInfo.ImageIID.SystemId
		}

		isMyWindowsImage, err := myImageHandler.CheckWindowsImage(vmReqInfo.ImageIID)
		if err != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Check Whether My Image is MS Windows Image : ", err)
			return irs.VMInfo{}, rtnErr
		}
		if isMyWindowsImage {
			var createErr error
			initUserData, createErr = vmHandler.CreateWinInitUserData(vmReqInfo.VMUserPasswd)
			if createErr != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Create Cloud-Init Script with the Password : ", createErr)
				return irs.VMInfo{}, rtnErr
			}
		} else {
			var createErr error
			initUserData, createErr = vmHandler.CreateLinuxInitUserData(vmReqInfo.ImageIID, keyPairId)
			if createErr != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Create Cloud-Init Script with the KeyPairId : ", createErr)
				return irs.VMInfo{}, rtnErr
			}
		}
	}
	cblogger.Info("### Succeeded in Creating Init UserData!!")
	// cblogger.Infof("Init UserData : [%s]", *initUserData)
	// Note) NCP에서는 UserData용 string에 Base64 인코딩 불필요
	// cmdStringBase64 := base64.StdEncoding.EncodeToString([]byte(cmdString))

	cblogger.Info("### Start to Create NCP VM Instance!!")
	// VM Creation req. info. setting
	// Note) Since NCP Classic is based on a physical network, it does not support VPC and Subnet specifications when creating VM.
	instanceReq := server.CreateServerInstancesRequest{
		ServerName:             				ncloud.String(instanceName),
		ServerImageProductCode: 				ncloud.String(publicImageId),
		MemberServerImageNo:					ncloud.String(myImageId),
		ServerProductCode:      				ncloud.String(instanceType),
		ServerDescription:  					ncloud.String(vmReqInfo.ImageIID.SystemId), // Caution!!
		LoginKeyName:           				ncloud.String(keyPairId),
		ZoneNo: 								zoneNo,		
		IsProtectServerTermination: 			ncloud.Bool(false), // NOTE Caution!! : 'true'로 설정하면 API로 Terminate(VM 반환) 제어 안됨.
		ServerCreateCount:      				minCount,
		AccessControlGroupConfigurationNoList: 	newSecurityGroupIds,
		UserData: 								initUserData,
	}
	// cblogger.Info(instanceReq)
	callLogStart := call.Start()
	runResult, err := vmHandler.VMClient.V2Api.CreateServerInstances(&instanceReq)
	if err != nil {		
		rtnErr := logAndReturnError(callLogInfo, "Failed to Create NCP VM instance. : ", err)
		return irs.VMInfo{}, rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	// Because there are functions that use NameID
	newVMIID := irs.IID{NameId: instanceName, SystemId: ncloud.StringValue(runResult.ServerInstanceList[0].ServerInstanceNo)}

	// Save VPC name to tag on the VM instance
	var vpcNameId string
	var subnetNameId string
	if !strings.EqualFold(vmReqInfo.VpcIID.NameId, "") {
		vpcNameId = vmReqInfo.VpcIID.NameId
	} else {
		vpcNameId = vmReqInfo.VpcIID.SystemId
	}	
	if !strings.EqualFold(vmReqInfo.SubnetIID.NameId, "") {
		subnetNameId = vmReqInfo.SubnetIID.NameId
	} else {
		subnetNameId = vmReqInfo.SubnetIID.SystemId
	}
	createTagResult, error := vmHandler.CreateVPCnSubnetTag(ncloud.String(newVMIID.SystemId), vpcNameId, subnetNameId)
	if error != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Create VPC and Subnet Tag : ", error)
		return irs.VMInfo{}, rtnErr
	}
	cblogger.Info("# createTagResult : ", createTagResult)

	// Wait while being created to get VM information.
	curStatus, statusErr := vmHandler.WaitToGetInfo(newVMIID)
	if statusErr != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Wait to Get the VM info. : ", statusErr)
		return irs.VMInfo{}, rtnErr
	}
	cblogger.Infof("==> VM status of [%s] : [%s]", newVMIID.NameId, curStatus)

	// Create a Public IP for the New VM
	// Caution!!) The number of Public IPs cannot be more than the number of instances on NCP cloud default service.
	time.Sleep(time.Second * 4)
	cblogger.Info("### Start Creating a Public IP!!")
	publicIpReq := server.CreatePublicIpInstanceRequest{
		ServerInstanceNo: 	runResult.ServerInstanceList[0].ServerInstanceNo,
		ZoneNo: 			runResult.ServerInstanceList[0].Zone.ZoneNo,
	}
	result, err := vmHandler.VMClient.V2Api.CreatePublicIpInstance(&publicIpReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Create PublicIp : ", err)
		return irs.VMInfo{}, rtnErr
	}
	if len(result.PublicIpInstanceList) < 1 {
		newErr := fmt.Errorf("Failed to Create any PublicIp.")
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}
	cblogger.Infof("*** NcpInstance.PublicIp : [%s]", ncloud.StringValue(result.PublicIpInstanceList[0].PublicIp))
	time.Sleep(time.Second * 2)

	// Get Created New VM Info
	vmInfo, error := vmHandler.GetVM(newVMIID)
	if error != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the VM Info : ", error)
		return irs.VMInfo{}, rtnErr
	}
	cblogger.Info("### VM Creation Processes have been Finished !!")
	
	if vmInfo.Platform == irs.WINDOWS {
		vmInfo.VMUserPasswd = vmReqInfo.VMUserPasswd
	}
	return vmInfo, nil
}

func (vmHandler *NcpVMHandler) MappingServerInfo(NcpInstance *server.ServerInstance) (irs.VMInfo, error) {
	cblogger.Info("NCP Classic Cloud driver: called MappingServerInfo()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "MappingServerInfo()", "MappingServerInfo()")

	// cblogger.Info("\n NcpInstance : ")
	// spew.Dump(NcpInstance)

	convertedTime, err := convertTimeFormat(*NcpInstance.CreateDate)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert the Time Format!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}
	
	var publicIp *string = nil
	var publicIpInstanceNo *string = nil

	// Create a PublicIp, if the instance doesn't have a 'Public IP' after creation.
	if strings.EqualFold(ncloud.StringValue(NcpInstance.PublicIp), "") {
		publicIpReq := server.CreatePublicIpInstanceRequest{
			ServerInstanceNo: 	NcpInstance.ServerInstanceNo,
			ZoneNo: 			NcpInstance.Zone.ZoneNo,
		}
		// CAUTION!!) The number of Public IPs cannot be more than the number of instances on NCP cloud default service.
		result, err := vmHandler.VMClient.V2Api.CreatePublicIpInstance(&publicIpReq)
		if err != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Create PublicIp : ", err)
			return irs.VMInfo{}, rtnErr
		}
		if len(result.PublicIpInstanceList) < 1 {
			newErr := fmt.Errorf("Failed to Create any PublicIp.")
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
		publicIp = result.PublicIpInstanceList[0].PublicIp
		publicIpInstanceNo = result.PublicIpInstanceList[0].PublicIpInstanceNo
		time.Sleep(time.Second * 1)
		// cblogger.Infof("*** NcpInstance.PublicIp : [%s]", ncloud.StringValue(publicIp))
		// cblogger.Infof("*** NcpInstance.publicIpInstanceNo [%s]: ", ncloud.StringValue(publicIpInstanceNo))

		cblogger.Infof("Finished to Create New Public IP.")
	} else {
		publicIp = NcpInstance.PublicIp
		// cblogger.Infof("*** NcpInstance.PublicIp : [%s]", ncloud.StringValue(publicIp))

		instanceReq := server.GetPublicIpInstanceListRequest{
			PublicIpList: 	[]*string{publicIp},
			ZoneNo: 		NcpInstance.Zone.ZoneNo,
		}
		// Get the Public IP list info. to search the PublicIp InstanceNo
		result, err := vmHandler.VMClient.V2Api.GetPublicIpInstanceList(&instanceReq)
		if err != nil {
			rtnErr := logAndReturnError(callLogInfo, "Get PublicIp InstanceList : ", err)
			return irs.VMInfo{}, rtnErr
		}
		if len(result.PublicIpInstanceList) < 1 {
			cblogger.Errorf("Failed to Get Any PublicIp : [%v]", err)
			return irs.VMInfo{}, errors.New("Failed to Get Any PublicIp.")
		}
		publicIpInstanceNo = result.PublicIpInstanceList[0].PublicIpInstanceNo
		// cblogger.Infof("*** NcpInstance.publicIpInstanceNo : [%s]", ncloud.StringValue(publicIpInstanceNo))
	}

	// To Get the BlockStorage info. of the VM instance
	blockStorageReq := server.GetBlockStorageInstanceListRequest{
		ServerInstanceNo: 	NcpInstance.ServerInstanceNo,
	}
	blockStorageResult, err := vmHandler.VMClient.V2Api.GetBlockStorageInstanceList(&blockStorageReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Block Storage InstanceList : ", err)
		return irs.VMInfo{}, rtnErr
	}
	if len(blockStorageResult.BlockStorageInstanceList) < 1 {
		cblogger.Errorf("Failed to Find Any BlockStorageInstance!!")
		return irs.VMInfo{}, errors.New("Failed to Find Any BlockStorageInstance.")
	}
	cblogger.Infof("Succeeded in Getting Block Storage InstanceList!!")
	// spew.Dump(blockStorageResult.BlockStorageInstanceList[0])

	var sgList []irs.IID
	if NcpInstance.AccessControlGroupList != nil {
		for _, acg := range NcpInstance.AccessControlGroupList {
			sgList = append(sgList, irs.IID{NameId: *acg.AccessControlGroupName, SystemId: *acg.AccessControlGroupConfigurationNo})
		}
	} else {
		fmt.Println("AccessControlGroupList is empty or nil")
	}

	// To Get the VM resources Info.
	vmInfo := irs.VMInfo{
		IId: irs.IID{
			NameId:   	*NcpInstance.ServerName,
			SystemId: 	*NcpInstance.ServerInstanceNo,
		},
		StartTime: 		convertedTime,

		// (Ref) NCP region/zone info. example
		// Region:  "KR", Zone: "KR-2"
		Region: irs.RegionInfo{
			Region: 	*NcpInstance.Region.RegionCode,
			//  *NcpInstance.Zone.ZoneCode,  // Zone info is bellow.
		},
		VMSpecName: 		ncloud.StringValue(NcpInstance.ServerProductCode), //Server Spec code임.
		SecurityGroupIIds: 	sgList,

		//NCP Key에는 SystemID가 없으므로, 고유한 NameID 값을 SystemID에도 반영
		KeyPairIId: irs.IID{
			NameId: 	*NcpInstance.LoginKeyName,
			SystemId: 	*NcpInstance.LoginKeyName,
		},

		PublicIP:   	*publicIp,
		PrivateIP:  	*NcpInstance.PrivateIp,		
		VMBootDisk: 	ncloud.StringValue(blockStorageResult.BlockStorageInstanceList[0].DeviceName),
		VMBlockDisk: 	ncloud.StringValue(blockStorageResult.BlockStorageInstanceList[0].DeviceName),
		RootDiskType: 	*NcpInstance.BaseBlockStorageDiskDetailType.CodeName,
		SSHAccessPoint: *publicIp + ":22",

		KeyValueList: []irs.KeyValue{
			{Key: "ServerInstanceType", Value: *NcpInstance.ServerInstanceType.CodeName},
			{Key: "CpuCount", Value: String(*NcpInstance.CpuCount)},
			{Key: "MemorySize(GB)", Value: strconv.FormatFloat(float64(*NcpInstance.MemorySize)/(1024*1024*1024), 'f', 0, 64)},
			{Key: "BaseBlockStorageSize(GB)", Value: strconv.FormatFloat(float64(*NcpInstance.BaseBlockStorageSize)/(1024*1024*1024), 'f', 0, 64)}, //GB로 변환
			{Key: "DiskType", Value: ncloud.StringValue(blockStorageResult.BlockStorageInstanceList[0].DiskType.CodeName)},
			{Key: "DiskDetailType", Value: ncloud.StringValue(blockStorageResult.BlockStorageInstanceList[0].DiskDetailType.CodeName)},
			{Key: "PlatformType", Value: *NcpInstance.PlatformType.CodeName},
			{Key: "ZoneCode", Value: *NcpInstance.Zone.ZoneCode},
			//{Key: "ZoneNo", Value: *NcpInstance.Zone.ZoneNo},
			{Key: "PublicIpID", Value: *publicIpInstanceNo}, // # To use it when delete the PublicIP
		},
	}

	myImageHandler := NcpMyImageHandler{
		RegionInfo:  vmHandler.RegionInfo,
		VMClient:    vmHandler.VMClient,
	}
	// Set the VM Image Type : 'PublicImage' or 'MyImage'
	if !strings.EqualFold(*NcpInstance.ServerDescription, "") {
		vmInfo.ImageIId.SystemId = *NcpInstance.ServerDescription // Note!! : Since MyImage ID is not included in the 'NcpInstance' info 
		vmInfo.ImageIId.NameId = *NcpInstance.ServerDescription
		
		isPublicImage, err := myImageHandler.isPublicImage(irs.IID{SystemId: *NcpInstance.ServerDescription}) // Caution!! : Not '*NcpInstance.ServerImageProductCode'
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
		rtnErr := logAndReturnError(callLogInfo, "Failed to Find BlockStorage Info : ", err)
		return irs.VMInfo{}, rtnErr
	}
	if !strings.EqualFold(*storageSize, "") {
		vmInfo.RootDiskSize = *storageSize
	}
	if !strings.EqualFold(*deviceName, "") {
		vmInfo.RootDeviceName = *deviceName
	}

	dataDiskList, err := vmHandler.GetVmDataDiskList(NcpInstance.ServerInstanceNo)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Data Disk List : ", err)
		return irs.VMInfo{}, rtnErr
	}
	if len(dataDiskList) > 0 {
		vmInfo.DataDiskIIDs = dataDiskList
	}

	// Set VM Zone Info
	if NcpInstance.Zone != nil {
		vmInfo.Region.Zone = *NcpInstance.Zone.ZoneCode
		// *NcpInstance.Zone.ZoneCode or *NcpInstance.Zone.ZoneName etc...
	}

	// To Get the VPC Name from Tag of the VM instance
	vpcName, subnetName, error := vmHandler.GetVPCnSubnetNameFromTag(NcpInstance.ServerInstanceNo)
	if error != nil {
		cblogger.Debug(error.Error())
		cblogger.Debug("Failed to Get VPC Name from Tag of the VM instance!!")
		// return irs.VMInfo{}, error  // Caution!!
	} else {
		cblogger.Infof("# vpcName : [%s]", vpcName)
		cblogger.Infof("# subnetName : [%s]", subnetName)
	}

	if len(vpcName) < 1 {
		cblogger.Debug("Failed to Get VPC Name from Tag!!")
	} else {
		// To get the VPC info.
		vpcHandler := NcpVPCHandler {
			CredentialInfo: 	vmHandler.CredentialInfo,
			RegionInfo:			vmHandler.RegionInfo,
			VMClient:         	vmHandler.VMClient,
		}
		vpcInfo, err := vpcHandler.GetVPC(irs.IID{SystemId: vpcName})
		if err != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Find the VPC : ", err)
			return irs.VMInfo{}, rtnErr
		}

		vmInfo.VpcIID.NameId = vpcName
		vmInfo.VpcIID.SystemId = vpcInfo.IId.SystemId
		vmInfo.SubnetIID.NameId = subnetName
		for _, curSubnet := range vpcInfo.SubnetInfoList {
			cblogger.Infof("Subnet NameId : [%s]", curSubnet.IId.NameId)
			if strings.EqualFold(curSubnet.IId.NameId, subnetName) {
				vmInfo.SubnetIID.SystemId = curSubnet.IId.SystemId
				break
			}
		}
	}
	cblogger.Infof("NCP Instance Uptime : [%s]", *NcpInstance.Uptime)

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

func (vmHandler *NcpVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetVM()!!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "GetVM()")

	instanceNumList := []*string{ncloud.String(vmIID.SystemId)}
	// spew.Dump(instanceNumList)

	curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
	if errStatus != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the VM Status : ", errStatus)
		return irs.VMInfo{}, rtnErr
	}
	cblogger.Info("===> VM Status : ", curStatus)

	// Since it's impossible to get VM info. during Creation, ...
	switch string(curStatus) {
	case "Creating", "Booting":
		cblogger.Infof("Wait for the VM creation before inquiring VM info. The VM status : [%s]", string(curStatus))
		return irs.VMInfo{}, errors.New("The VM status is 'Creating' or 'Booting', wait for the VM creation before inquiring VM info. : " + vmIID.SystemId)
	default:
		cblogger.Infof("===> The VM status not 'Creating' or 'Booting', you can get the VM info.")
	}

	regionNo, err := vmHandler.GetRegionNo(vmHandler.RegionInfo.Region)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Region No :", err)
		return irs.VMInfo{}, rtnErr
	}
	zoneNo, err := vmHandler.GetZoneNo(vmHandler.RegionInfo.Region, vmHandler.RegionInfo.Zone)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Zone No of the Zone Code :", err)
		return irs.VMInfo{}, rtnErr
	}
	instanceReq := server.GetServerInstanceListRequest{
		ServerInstanceNoList: 	instanceNumList,
		RegionNo: 				regionNo,	
		ZoneNo: 				zoneNo,
	}
	callLogStart := call.Start()
	result, err := vmHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get VM list from NCP :", err)
		return irs.VMInfo{}, rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.ServerInstanceList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any VM info with the SystemId : [%s]", vmIID.SystemId)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}

	vmInfo, err := vmHandler.MappingServerInfo(result.ServerInstanceList[0])
	if err != nil {
		LoggingError(callLogInfo, err)
		return irs.VMInfo{}, err
	}
	return vmInfo, nil
}

func (vmHandler *NcpVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCP Classic Cloud driver: called SuspendVM()!!")
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)

	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "SuspendVM()")

	var serverInstanceNo []*string
	serverInstanceNo = []*string{ncloud.String(vmIID.SystemId)}
	var resultStatus string

	cblogger.Info("Start Get VM Status...")
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the VM Status : ", err)
		return irs.VMStatus("Failed. "), rtnErr
	} else {
		cblogger.Infof("Succeeded in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, vmStatus)
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

		stopReq := server.StopServerInstancesRequest{
			ServerInstanceNoList: serverInstanceNo,
		}
		callLogStart := call.Start()
		runResult, err := vmHandler.VMClient.V2Api.StopServerInstances(&stopReq)
		if err != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Stop the VM instance : ", err)
			return irs.VMStatus("Failed. "), rtnErr
		}
		LoggingInfo(callLogInfo, callLogStart)
		cblogger.Info(runResult)
	}
	return irs.VMStatus("Suspending"), nil
}

func (vmHandler *NcpVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCP Classic Cloud driver: called ResumeVM()!")
	cblogger.Infof("vmID : " + vmIID.SystemId)

	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "ResumeVM()")

	serverInstanceNo := []*string{ncloud.String(vmIID.SystemId)}
	var resultStatus string

	cblogger.Info("Start Get VM Status...")
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the VM Status : ", err)
		return irs.VMStatus("Failed. "), rtnErr
	} else {
		cblogger.Infof("Succeeded in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, vmStatus)
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

		startReq := server.StartServerInstancesRequest{
			ServerInstanceNoList: serverInstanceNo,
		}
		callLogStart := call.Start()
		runResult, err := vmHandler.VMClient.V2Api.StartServerInstances(&startReq)
		if err != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Start the VM instance : ", err)
			return irs.VMStatus("Failed. "), rtnErr
		}
		LoggingInfo(callLogInfo, callLogStart)
		cblogger.Info(runResult)

		return irs.VMStatus("Resuming"), nil
	}
}

func (vmHandler *NcpVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCP Classic Cloud driver: called RebootVM()!")
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)

	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "RebootVM()")

	var resultStatus string
	serverInstanceNo := []*string{ncloud.String(vmIID.SystemId)}

	cblogger.Info("Start Get VM Status...")
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the VM Status : ", err)
		return irs.VMStatus("Failed. "), rtnErr
	} else {
		cblogger.Infof("Succeeded in Getting the VM Status of [%s] : [%s]", vmIID, vmStatus)
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
		// spew.Dump(serverInstanceNo)
		cblogger.Infof("vmID : [%s]", *serverInstanceNo[0])

		req := server.RebootServerInstancesRequest{
			ServerInstanceNoList: serverInstanceNo,
		}
		callLogStart := call.Start()
		runResult, err := vmHandler.VMClient.V2Api.RebootServerInstances(&req)
		if err != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Reboot the VM instance : ", err)
			return irs.VMStatus("Failed. "), rtnErr
		}
		LoggingInfo(callLogInfo, callLogStart)
		cblogger.Info(runResult)

		return irs.VMStatus("Rebooting"), nil
	}
}

func (vmHandler *NcpVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCP Classic Cloud driver: called TerminateVM()!")
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)

	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "TerminateVM()")

	dataDiskList, err := vmHandler.GetVmDataDiskList(ncloud.String(vmIID.SystemId))
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Data Disk List : ", err)
		return irs.VMStatus("Failed. "), rtnErr
	}
	if len(dataDiskList) > 0 {
		newErr := fmt.Errorf("Please Detach the Storage Attached to the VM and Try again.")
		cblogger.Error(newErr.Error())
		return irs.VMStatus("Failed to Terminate the VM instance."), newErr
	}

	serverInstanceNo := []*string{ncloud.String(vmIID.SystemId)}

	cblogger.Info("Start Get VM Status...")
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the VM Status : ", err)
		return irs.VMStatus("Failed. "), rtnErr
	} else {
		cblogger.Infof("Succeeded in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, vmStatus)
	}

	vmInfo, error := vmHandler.GetVM(vmIID)
	if error != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the VM Info : ", error)
		return irs.VMStatus("Failed. "), rtnErr
	}

	switch string(vmStatus) {
	case "Suspended":
		cblogger.Infof("VM Status : 'Suspended'. so it Can be Terminated!!")		
		cblogger.Infof("vmID : [%s]", *serverInstanceNo[0])

		// To Delete Tags of the VM instance
		Result, error := vmHandler.DeleteVMTags(serverInstanceNo[0])
		if error != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Delete Tags of the VM instance! : ", error)
			return irs.VMStatus("Failed. "), rtnErr
		} else {
			cblogger.Infof("# DeleteVMTags Result : [%t]", Result)
		}

		// To Terminate the VM instance
		req := server.TerminateServerInstancesRequest{
			ServerInstanceNoList: serverInstanceNo,
		}
		callLogStart := call.Start()
		runResult, err := vmHandler.VMClient.V2Api.TerminateServerInstances(&req)
		if err != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Terminate the VM instance. : ", err)
			return irs.VMStatus("Failed. "), rtnErr
		}
		LoggingInfo(callLogInfo, callLogStart)
		cblogger.Info(runResult)

		// If the NCP instance has a 'Public IP', delete it after termination of the instance.
		if ncloud.String(vmInfo.PublicIP) != nil {
			// Delete the PublicIP of the VM
			vmStatus, err := vmHandler.DeletePublicIP(vmInfo)
			if err != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Delete the PublicIP. : ", err)
				return irs.VMStatus("Failed. "), rtnErr
			}
			cblogger.Info(vmStatus)
		}

		return irs.VMStatus("Terminating"), nil

	case "Running":
		cblogger.Infof("VM Status : 'Running'. so it Can be Terminated AFTER SUSPENSION !!")

		// spew.Dump(serverInstanceNo)
		cblogger.Infof("vmID : [%s]", *serverInstanceNo[0])

		cblogger.Info("Start Suspend VM !!")
		result, err := vmHandler.SuspendVM(vmIID)
		if err != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Suspend the VM. : ", err)
			return irs.VMStatus("Failed. "), rtnErr
		} else {
			cblogger.Infof("Succeeded in Suspending the VM [%s] : [%s]", vmIID, result)
		}

		//===================================
		// 15-second wait for Suspending
		//===================================
		curRetryCnt := 0
		maxRetryCnt := 15
		for {
			curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
			if errStatus != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Get the VM Status : ", errStatus)
				return irs.VMStatus("Failed. "), rtnErr
			}

			cblogger.Info("===> VM Status : ", curStatus)
			if curStatus != irs.VMStatus("Suspended") {
				curRetryCnt++
				cblogger.Infof("The VM is not 'Suspended', so wait for a second more before inquiring Termination.")
				time.Sleep(time.Second * 5)
				if curRetryCnt > maxRetryCnt {
					cblogger.Errorf("Despite waiting for a long time(%d sec), the VM is not 'suspended', so it is forcibly terminated.", maxRetryCnt)
				}
			} else {
				break
			}
		}

		cblogger.Info("# SuspendVM() Finished")

		// To Delete Tags of the VM instance
		Result, error := vmHandler.DeleteVMTags(serverInstanceNo[0])
		if error != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Delete Tags of the VM instance!! : ", error)
			return irs.VMStatus("Failed. "), rtnErr
		} else {
			cblogger.Infof("# DeleteVMTags Result : [%t]", Result)
		}		

		// To Terminate the VM instance
		req := server.TerminateServerInstancesRequest{
			ServerInstanceNoList: serverInstanceNo,
		}
		callLogStart := call.Start()
		runResult, err := vmHandler.VMClient.V2Api.TerminateServerInstances(&req)
		if err != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Terminate the NCP VM instance. : ", err)
			return irs.VMStatus("Failed. "), rtnErr
		}
		LoggingInfo(callLogInfo, callLogStart)
		cblogger.Info(runResult)

		// If the NCP instance has a 'Public IP', delete it after termination of the instance.
		if ncloud.String(vmInfo.PublicIP) != nil {
			// PublicIP 삭제
			vmStatus, err := vmHandler.DeletePublicIP(vmInfo)
			if err != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Delete the PublicIP : ", err)
				return irs.VMStatus("Failed. "), rtnErr
			}
			cblogger.Info(vmStatus)
		}

		return irs.VMStatus("Terminateding"), nil

	default:
		resultStatus := "The VM status is not 'Running' or 'Suspended'. so it Can NOT be Terminated!! Run or Suspend the VM before terminating."

		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	}
}

/*
# NCP 'serverInstanceStatusName' : Not 'serverInstanceStatus'
	: https://api.ncloud-docs.com/docs/en/common-apidatatype-serverinstance

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

	// cblogger.Infof("NCP VM Status : [%s]", vmStatus)
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
	cblogger.Infof("Succeeded in Converting the VM Status : [%s] ==> [%s]", vmStatus, resultStatus)
	return irs.VMStatus(resultStatus), nil
}

func (vmHandler *NcpVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetVMStatus()!")

	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "GetVMStatus()")

	regionNo, err := vmHandler.GetRegionNo(vmHandler.RegionInfo.Region)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Region No :", err)
		return irs.VMStatus(""), rtnErr
	}
	zoneNo, err := vmHandler.GetZoneNo(vmHandler.RegionInfo.Region, vmHandler.RegionInfo.Zone)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Zone No of the Zone Code :", err)
		return irs.VMStatus(""), rtnErr
	}
	instanceReq := server.GetServerInstanceListRequest{
		ServerInstanceNoList: []*string{
			ncloud.String(vmIID.SystemId),
		},
		RegionNo: 				regionNo,	
		ZoneNo: 				zoneNo,
	}
	callLogStart := call.Start()
	result, err := vmHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
	if err != nil {
		cblogger.Error(err)
		cblogger.Error(*result.ReturnMessage)
		LoggingError(callLogInfo, err)
		return irs.VMStatus(""), err   // Caution!!) Do not fill in "Failed."
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.ServerInstanceList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any VM with the SystemId : [%s]", vmIID.SystemId)
		cblogger.Debug(newErr.Error()) // For after Termination!!
		return irs.VMStatus(""), newErr  // Caution!!) Do not fill in "Failed."
	}
	cblogger.Info("Succeeded in Getting ServerInstanceList!!")

	vmStatus, errStatus := ConvertVMStatusString(*result.ServerInstanceList[0].ServerInstanceStatusName)
	cblogger.Info("# Converted VM Status : " + vmStatus)
	return vmStatus, errStatus
}

func (vmHandler *NcpVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	cblogger.Info("NCP Classic Cloud driver: called ListVMStatus()!")

	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "ListVMStatus()", "ListVMStatus()")

	regionNo, err := vmHandler.GetRegionNo(vmHandler.RegionInfo.Region)
	if err != nil {
		cblogger.Errorf("Failed to Get RegionNo : [%v]", err)
		return nil, err
	}
	zoneNo, err := vmHandler.GetZoneNo(vmHandler.RegionInfo.Region, vmHandler.RegionInfo.Zone)
	if err != nil {
		cblogger.Errorf("Failed to Get NCP Zone No of the Zone Code : [%v]", err)
		return nil, err
	}
	instanceReq := server.GetServerInstanceListRequest{
		ServerInstanceNoList: 	[]*string{},
		RegionNo:            	regionNo,
		ZoneNo: 				zoneNo,
	}
	callLogStart := call.Start()
	result, err := vmHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get GetServerInstanceList : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return []*irs.VMStatusInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)
	if len(result.ServerInstanceList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any VM info in the Zone : [%s]", vmHandler.RegionInfo.Zone)
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Getting ServerInstanceList in the Zone!!")
	}

	var vmStatusList []*irs.VMStatusInfo
	for _, vm := range result.ServerInstanceList {
		//*vm.ServerInstanceStatusName
		//*vm.ServerName
		vmStatus, err := ConvertVMStatusString(*vm.ServerInstanceStatusName)
		if err != nil {
			cblogger.Error(err)
			LoggingError(callLogInfo, err)
			return []*irs.VMStatusInfo{}, err
		}
		// cblogger.Info(" VM Status : ", vmStatus)

		vmStatusInfo := irs.VMStatusInfo{
			IId:      irs.IID{NameId: *vm.ServerName, SystemId: *vm.ServerInstanceNo},
			VmStatus: vmStatus,
		}
		cblogger.Infof(" VM Status of [%s] : [%s]", vmStatusInfo.IId.SystemId, vmStatusInfo.VmStatus)
		vmStatusList = append(vmStatusList, &vmStatusInfo)
	}
	return vmStatusList, err
}

func (vmHandler *NcpVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger.Info("NCP Classic Cloud driver: called ListVM()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "ListVM()", "ListVM()")

	regionNo, err := vmHandler.GetRegionNo(vmHandler.RegionInfo.Region)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Region No :", err)
		return nil, rtnErr
	}
	zoneNo, err := vmHandler.GetZoneNo(vmHandler.RegionInfo.Region, vmHandler.RegionInfo.Zone)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Zone No of the Zone Code :", err)
		return nil, rtnErr
	}
	
	instanceReq := server.GetServerInstanceListRequest{
		ServerInstanceNoList: 	[]*string{},
		RegionNo:             	regionNo,
		ZoneNo: 				zoneNo,
	}
	callLogStart := call.Start()
	result, err := vmHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get GetServerInstanceList : [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)		
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.ServerInstanceList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any VM in the Zone: [%s]", vmHandler.RegionInfo.Zone)
		cblogger.Debug(newErr.Error())
		return nil, nil // Not returns Error message		
	} else {
		cblogger.Info("Succeeded in Getting ServerInstanceList!!")
	}	
	
	var vmInfoList []*irs.VMInfo
	for _, vm := range result.ServerInstanceList {
		cblogger.Info("NCP VM Instance Info. inquiry : ", *vm.ServerInstanceNo)

		curStatus, errStatus := vmHandler.GetVMStatus(irs.IID{SystemId: *vm.ServerInstanceNo})
		if errStatus != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Get the VM Status : ", errStatus)
			return nil, rtnErr
		} else {
			cblogger.Infof("Succeeded to Get the VM Status of [%s] : [%s]", irs.IID{SystemId: *vm.ServerInstanceNo}, curStatus)
		}
		cblogger.Info("===> VM Status : ", curStatus)

		switch string(curStatus) {
		case "Creating", "Booting":
			cblogger.Errorf("The VM status : [%s], Can Not Get the VM info.", string(curStatus))
			return nil, nil

		default:
			cblogger.Infof("===> The VM status not 'Creating' or 'Booting', you can get the VM info.")
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

// Waiting for up to 300 seconds until VM info. can be get
func (vmHandler *NcpVMHandler) WaitToGetInfo(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("======> As VM info. cannot be retrieved immediately after VM creation, it waits until running.")

	curRetryCnt := 0
	maxRetryCnt := 500

	for {
		curStatus, err := vmHandler.GetVMStatus(vmIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Status of the VM : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.VMStatus("Failed. "), newErr
		} else {
			cblogger.Infof("Succeeded in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, curStatus)
		}
		cblogger.Infof("===> VM Status : [%s]", curStatus)

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

// Waiting for up to 300 seconds until Public IP can be deleted.
func (vmHandler *NcpVMHandler) WaitToDelPublicIp(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("======> As Public IP cannot be deleted immediately after VM termination call, it waits until termination is finished.")

	curRetryCnt := 0
	maxRetryCnt := 600

	for {
		curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
		if errStatus != nil {
			newErr := fmt.Errorf("Failed to Get the Status of the VM : [%s], [%v]", vmIID, errStatus)
			cblogger.Debug(newErr.Error())
			// return irs.VMStatus("Failed. "), newErr // Caution!!) For after Termination!!
		} else {
			cblogger.Infof("Succeeded in Getting the VM Status of [%s] : [%s]", vmIID, curStatus)
		}
		cblogger.Info("===> VM Status : ", curStatus)

		switch string(curStatus) {
		case "Suspended", "Terminating":
			curRetryCnt++
			cblogger.Infof("The VM is still 'Terminating', so wait for a second more before inquiring the VM info.")
			time.Sleep(time.Second * 5)
			if curRetryCnt > maxRetryCnt {
				cblogger.Errorf("Despite waiting for a long time(%d sec), the VM status is '%s', so it is forcibly finished.", maxRetryCnt, curStatus)
				return irs.VMStatus("Failed"), errors.New("Despite waiting for a long time, the VM status is 'Creating', so it is forcibly finishied.")
			}

		default:
			cblogger.Infof("===>### The VM Termination is finished, so stopping the waiting.")
			return irs.VMStatus(curStatus), nil
			//break
		}
	}
}

// Whenever a VM is terminated, Delete the public IP that the VM has
func (vmHandler *NcpVMHandler) DeletePublicIP(vmInfo irs.VMInfo) (irs.VMStatus, error) {
	cblogger.Info("NCP Classic Cloud driver: called DeletePublicIP()!")

	var publicIPId string

	// Use Key/Value info of the vmInfo.
	for _, keyInfo := range vmInfo.KeyValueList {
		if keyInfo.Key == "PublicIpID" {
			publicIPId = keyInfo.Value
			break
		}
	}

	// spew.Dump(vmInfo.PublicIP)
	// spew.Dump(publicIPId)

	//=========================================
	// Wait for that the VM is terminated
	//=========================================
	curStatus, errStatus := vmHandler.WaitToDelPublicIp(vmInfo.IId)
	if errStatus != nil {
		cblogger.Error(errStatus.Error())
		// return irs.VMStatus("Failed. "), errStatus   // Caution!!
	}
	cblogger.Infof("==> VM status of [%s] : [%s]", vmInfo.IId.NameId, curStatus)

	deleteReq := server.DeletePublicIpInstancesRequest{
		PublicIpInstanceNoList: []*string{
			ncloud.String(publicIPId),
		},
	}
	cblogger.Infof("DeletePublicIPReq Ready!!")
	result, err := vmHandler.VMClient.V2Api.DeletePublicIpInstances(&deleteReq)
	if err != nil {
		cblogger.Error(*result.ReturnMessage)
		newErr := fmt.Errorf("Failed to Delete the PublicIP of the instance : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMStatus("Failed. "), newErr
	}
	if *result.TotalRows < 1 {
		return irs.VMStatus("Failed to Delete any PublicIP of the instance. [%v]"), err
	} else {
		cblogger.Infof(" ### Succeeded in Deleting the PublicIP of the instance.!!")
	}

	return irs.VMStatus("Terminating"), nil
}

func (vmHandler *NcpVMHandler) GetVmIdByName(vmNameID string) (string, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetVmIdByName()!")

	var vmId string
	// Get VM list
	vmList, err := vmHandler.ListVM()
	if err != nil {
		return "", err
	}

	// Search VM by Name in the VM list
	for _, vm := range vmList {
		if strings.EqualFold(vm.IId.NameId, vmNameID) {
			vmId = vm.IId.SystemId
			break
		}
	}

	// Error handling when the VM is not found
	if vmId == "" {
		newErr := fmt.Errorf("Failed to Find the VM with the name : [%s]", vmNameID)
		return "", newErr
	} else {
	return vmId, nil
	}
}

func (vmHandler *NcpVMHandler) CreateVPCnSubnetTag(vmID *string, vpcName string, subnetName string) (bool, error) {
	cblogger.Info("NCP Classic Cloud driver: called CreateVPCnSubnetTag()!")

	var instanceNos []*string
	var instanceTags []*server.InstanceTagParameter

	tagKey := []string {"VPCName", "SubnetName"}

	instanceNos = append(instanceNos, vmID)
	instanceTags = []*server.InstanceTagParameter {
		{
			TagKey: 	ncloud.String(tagKey[0]), 
			TagValue: 	ncloud.String(vpcName),
		}, 
		{
			TagKey: 	ncloud.String(tagKey[1]), 
			TagValue: 	ncloud.String(subnetName),
		},
	}
	// cblogger.Info("\n instanceTags : ")
	// spew.Dump(instanceTags)

	tagReq := server.CreateInstanceTagsRequest{
		InstanceNoList:     instanceNos,
		InstanceTagList: 	instanceTags,
	}
	// cblogger.Info("\n tagReq : ")
	// spew.Dump(tagReq)
	tagResult, err := vmHandler.VMClient.V2Api.CreateInstanceTags(&tagReq)
	if err != nil {		
		cblogger.Errorf("Failed to Create NCP VM Tag. : [%v]", err)
		cblogger.Error(*tagResult.ReturnMessage)
		return false, err
	} else {
		cblogger.Infof("tagResult.ReturnMessage : [%s]", *tagResult.ReturnMessage)
	}

	return true, nil
}

func (vmHandler *NcpVMHandler) GetVPCnSubnetNameFromTag(vmID *string) (string, string, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetVPCnSubnetNameFromTag()!")

	var instanceNos []*string
	instanceNos = append(instanceNos, vmID)
	var vpcName string
	var subnetName string

	instanceTagReq := server.GetInstanceTagListRequest{InstanceNoList: instanceNos}
	// spew.Dump(instanceTagReq)
	getTagListResult, err := vmHandler.VMClient.V2Api.GetInstanceTagList(&instanceTagReq)
	if err != nil {
		cblogger.Error(*getTagListResult.ReturnMessage)
		newErr := fmt.Errorf("Failed to Find VM Tag List from NCP : [%v]", err)
		return "", "", newErr
	}
	if len(getTagListResult.InstanceTagList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any Tag from the VM SystemID!!")
		return "", "", newErr
	} else {
		cblogger.Infof("getTagListResult.ReturnMessage : [%s]", *getTagListResult.ReturnMessage)
		// spew.Dump("\n getTagListResult : ", *getTagListResult)		
	}
	cblogger.Infof("Succeeded in Getting Tag info from the VM!!")

	for _, curTag := range getTagListResult.InstanceTagList {
		if ncloud.StringValue(curTag.TagKey) == "VPCName" {			
			vpcName = ncloud.StringValue(curTag.TagValue)
		} else if ncloud.StringValue(curTag.TagKey) == "SubnetName" {			
			subnetName = ncloud.StringValue(curTag.TagValue)
		}
	}
	if len(vpcName) < 1 {
		cblogger.Errorf("Failed to Get VPC Name from the Tag!!")
		return "", "", errors.New("Failed to Get VPC Name from the Tag!!")
	}
	if len(subnetName) < 1 {
		cblogger.Errorf("Failed to Get Subnet Name from the Tag!!")
		return "", "", errors.New("Failed to Get Subnet Name from the Tag!!")
	}
	return vpcName, subnetName, nil
}

func (vmHandler *NcpVMHandler) DeleteVMTags(vmID *string) (bool, error) {
	cblogger.Info("NCP Classic Cloud driver: called DeleteVMTags()!")

	var instanceNos []*string
	instanceNos = append(instanceNos, vmID)

	tagReq := server.DeleteInstanceTagsRequest {
		InstanceNoList:     instanceNos,
	}
	tagResult, err := vmHandler.VMClient.V2Api.DeleteInstanceTags(&tagReq)
	if err != nil {		
		cblogger.Errorf("Failed to Delete NCP VM Tag. : [%v]", err)
		cblogger.Error(*tagResult.ReturnMessage)
		return false, err
	} else {
		cblogger.Infof("tagResult.ReturnMessage : [%s]", *tagResult.ReturnMessage)
	}

	return true, nil
}

func (vmHandler *NcpVMHandler) GetNcpVMInfo(instanceId string) (*server.ServerInstance, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetNcpVMInfo()")

	vmIID := irs.IID{SystemId: instanceId}
	instanceNumList := []*string{ncloud.String(instanceId)}

	curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
	if errStatus != nil {
		newErr := fmt.Errorf("Failed to Get the Status of the VM : [%v]", errStatus)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	cblogger.Info("===> VM Status : ", curStatus)

	// Since it's impossible to get VM info. during Creation, ...
	switch string(curStatus) {
	case "Creating", "Booting":
		cblogger.Infof("Wait for the VM creation before inquiring VM info. The VM status : [%s]", string(curStatus))
		return nil, errors.New("The VM status is 'Creating' or 'Booting', wait for the VM creation before inquiring VM info. : " + vmIID.SystemId)
	default:
		cblogger.Infof("===> The VM status not 'Creating' or 'Booting', you can get the VM info.")
	}

	regionNo, err := vmHandler.GetRegionNo(vmHandler.RegionInfo.Region)
	if err != nil {
		cblogger.Errorf("Failed to Get RegionNo : [%v]", err)
		return nil, err
	}
	zoneNo, err := vmHandler.GetZoneNo(vmHandler.RegionInfo.Region, vmHandler.RegionInfo.Zone)
	if err != nil {
		cblogger.Errorf("Failed to Get NCP Zone No of the Zone Code : [%v]", err)
		return nil, err
	}
	instanceReq := server.GetServerInstanceListRequest{
		ServerInstanceNoList: 	instanceNumList,
		RegionNo: 				regionNo,	
		ZoneNo: 				zoneNo,
	}
	result, err := vmHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find VM list with the SystemId from NCP : [%s], [%v]", vmIID.SystemId, err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	if len(result.ServerInstanceList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any VM info with the SystemId : [%s]", vmIID.SystemId)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	return result.ServerInstanceList[0], nil
}

func (vmHandler *NcpVMHandler) GetVmRootDiskInfo(vmId *string) (*string, *string, error) {
	cblogger.Info("NCPVPC Cloud driver: called GetVmRootDiskInfo()!!")

	if strings.EqualFold(*vmId, "") {
		newErr := fmt.Errorf("Invalid VM ID!!")
		cblogger.Error(newErr.Error())
		return nil, nil, newErr
	}

	storageReq := server.GetBlockStorageInstanceListRequest {
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

func (vmHandler *NcpVMHandler) GetVmDataDiskList(vmId *string) ([]irs.IID, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetVmDataDiskList()")

	if strings.EqualFold(*vmId, "") {
		newErr := fmt.Errorf("Invalid VM Instance ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	storageReq := server.GetBlockStorageInstanceListRequest {
		ServerInstanceNo:   vmId,
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

func (vmHandler *NcpVMHandler) CreateLinuxInitUserData(imageIID irs.IID, keyPairId string) (*string, error) {
	cblogger.Info("NCPVPC Cloud driver: called CreateLinuxInitScript()!!")

	myImageHandler := NcpMyImageHandler{
		RegionInfo:  vmHandler.RegionInfo,
		VMClient:    vmHandler.VMClient,
	}
	var getErr error
	originImagePlatform, getErr := myImageHandler.GetOriginImageOSPlatform(imageIID)
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Get OriginImageOSPlatform of the Image : [%v]", getErr)
		cblogger.Error(newErr.Error())
		return nil, newErr	
	}	

	var initFilePath string
	switch originImagePlatform {
	case "UBUNTU" :
		initFilePath = os.Getenv("CBSPIDER_ROOT") + ubuntuCloudInitFilePath
	case "CENTOS" :
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
	keyValue, getKeyErr := keycommon.GetKey("NCP", hashString, keyPairId)
	if getKeyErr != nil {
		newErr := fmt.Errorf("Failed to Get the Public Key from DB : [%v]", getKeyErr)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Set Linux cloud-init script
	cmdString = strings.ReplaceAll(cmdString, "{{username}}", lnxUserName)
	cmdString = strings.ReplaceAll(cmdString, "{{public_key}}", keyValue.Value)
	// cblogger.Info("cmdString : ", cmdString)
	return &cmdString, nil
}

func (vmHandler *NcpVMHandler) CreateWinInitUserData(passWord string) (*string, error) {
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

	// Set Windows cloud-init script
	cmdString = strings.ReplaceAll(cmdString, "{{PASSWORD}}", passWord)
	// cblogger.Info("cmdString : ", cmdString)
	return &cmdString, nil
}

// Find the RegionNo that corresponds to a RegionCode
func (vmHandler *NcpVMHandler) GetRegionNo(regionCode string) (*string, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetRegionNo()!")
	// Search NCP Instance Region'No' corresponding to the NCP Region'Code'

	if strings.EqualFold(regionCode, "") {
		newErr := fmt.Errorf("Invalid Region Code!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Populate the global map : regionMap
	err := vmHandler.checkAndSetRegionNoList(regionCode)
	if err != nil {
		newErr := fmt.Errorf("Failed to Init RegionNoList : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	regionNo, exists := regionMap[regionCode]
	if exists {
		return &regionNo, nil
	} else {
		newErr := fmt.Errorf("Failed to Find the RegionNo that corresponds to the RegionCode.")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
}

// Find the ZoneNo that corresponds to a ZoneCode
func (vmHandler *NcpVMHandler) GetZoneNo(regionCode string, zoneCode string) (*string, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetZoneNo()!")
	// Search NCP Instance Zone'No' corresponding to the NCP Zone'Code'

	if strings.EqualFold(regionCode, "") {
		newErr := fmt.Errorf("Invalid Region Code!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if strings.EqualFold(zoneCode, "") {
		newErr := fmt.Errorf("Invalid Zone Code!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	regionNo, getErr := vmHandler.GetRegionNo(regionCode)
	if getErr != nil {
		newErr := fmt.Errorf("Failed to Init ZoneNoList : [%v]", getErr)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Populate the global map : regionZoneMap
	err := vmHandler.checkAndSetZoneNoList(*regionNo, zoneCode)
	if err != nil {
		newErr := fmt.Errorf("Failed to Init ZoneNoList : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if zones, exists := regionZoneMap[*regionNo]; exists {
		if zoneNo, ok := zones[zoneCode]; ok {
			return &zoneNo, nil
		} else {
			newErr := fmt.Errorf("Zone Code '%s' not found in Region No '%s'\n", zoneCode, regionNo)
			cblogger.Error(newErr.Error())
			return nil, newErr
		}
	} else {
		newErr := fmt.Errorf("Region No '%s' not found from the Map\n", regionNo)
			cblogger.Error(newErr.Error())
			return nil, newErr		
	}
}

// Check the global 'regionMap' and Populate the map with the regions data.
func (vmHandler *NcpVMHandler) checkAndSetRegionNoList(regionCode string) error {
	cblogger.Info("NCP Classic Cloud driver: called checkAndSetRegionNoList()!")

	if strings.EqualFold(regionCode, "") {
		newErr := fmt.Errorf("Invalid Region Code!!")
		cblogger.Error(newErr.Error())
		return newErr
	}
	
	regionNo, exists := regionMap[regionCode]
	if exists {
		cblogger.Infof("# Region Code '%s' has Region No: %s", regionCode, regionNo)
	} else {
		cblogger.Infof("# Region Code '%s' not found. So Set RegionNo List!!", regionCode)

		cblogger.Info("# SetRegionNoList()!")
		regionListReq := server.GetRegionListRequest{}
		regionListResult, err := vmHandler.VMClient.V2Api.GetRegionList(&regionListReq)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get RegionList from NCP Cloud : [%v]", err)
			cblogger.Error(newErr.Error())
			return newErr
		}
		if len(regionListResult.RegionList) < 1 {
			newErr := fmt.Errorf("Failed to Find Any Region Info.")
			cblogger.Error(newErr.Error())
			return newErr
		}
			
		for _, region := range regionListResult.RegionList {
			regionMap[*region.RegionCode] = *region.RegionNo
		}
	}

	return nil
}

// Check the global 'regionZoneMap' and Populate the map with the regions data.
func (vmHandler *NcpVMHandler) checkAndSetZoneNoList(regionNo string, zoneCode string) error {
	cblogger.Info("NCP Classic Cloud driver: called checkAndSetZoneNoList()!")

	if strings.EqualFold(regionNo, "") {
		newErr := fmt.Errorf("Invalid Region Code!!")
		cblogger.Error(newErr.Error())
		return newErr
	}

	if strings.EqualFold(zoneCode, "") {
		newErr := fmt.Errorf("Invalid Zone Code!!")
		cblogger.Error(newErr.Error())
		return newErr
	}

	if zones, exists := regionZoneMap[regionNo]; exists {
		if zoneNo, ok := zones[zoneCode]; ok {
			cblogger.Infof("# Region No '%s', Zone Code '%s' has Zone No: %s", regionNo, zoneCode, zoneNo)
		} else {
			cblogger.Infof("Zone Code '%s' not found in Region No. So Set ZoneNo List!!'%s'", zoneCode, regionNo)

			err := vmHandler.setZoneNoList()
			if err != nil {
				newErr := fmt.Errorf("Failed to Set ZoneNoList : [%v]", err)
				cblogger.Error(newErr.Error())
				return newErr
			}
		}
	} else {
		fmt.Printf("Region No '%s' not found from the Map. So Set ZoneNo List!!\n", regionNo)

		err := vmHandler.setZoneNoList()
		if err != nil {
			newErr := fmt.Errorf("Failed to Set ZoneNoList : [%v]", err)
			cblogger.Error(newErr.Error())
			return newErr
		}
	}

	return nil
}

// Populate the global map with the zones data.
func (vmHandler *NcpVMHandler) setZoneNoList() error {
	cblogger.Info("NCP Classic Cloud driver: called setZoneNoList()!")

	regionListReq := server.GetRegionListRequest{}
	regionListResult, err := vmHandler.VMClient.V2Api.GetRegionList(&regionListReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get RegionList from NCP Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		return newErr
	}
	if len(regionListResult.RegionList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any Region Info.")
		cblogger.Error(newErr.Error())
		return newErr
	}

	for _, region := range regionListResult.RegionList {
		cblogger.Info("# SetZoneNoList() of the Region!")
		zoneListReq := server.GetZoneListRequest{
			RegionNo: 	region.RegionNo,
			//RegionNo: nil, //CAUTION!! : If you look up like this, only two Zones in Korea will come out.
		}
		zoneListResult, err := vmHandler.VMClient.V2Api.GetZoneList(&zoneListReq)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get zoneList from NCP Cloud : [%v]", err)
			cblogger.Error(newErr.Error())
			return newErr
		}
		if len(zoneListResult.ZoneList) < 1 {
			newErr := fmt.Errorf("Failed to Find Any Zone Info.")
			cblogger.Error(newErr.Error())
			return newErr
		}

		for _, zone := range zoneListResult.ZoneList {
			if _, exists := regionZoneMap[*zone.RegionNo]; !exists {
				regionZoneMap[*zone.RegionNo] = make(map[string]string)
			}
			regionZoneMap[*zone.RegionNo][*zone.ZoneCode] = *zone.ZoneNo			
		}
	}

	return nil
}
