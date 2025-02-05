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

// Global map to hold Subnet Zone info against the VM. Initialize the map (with {})
var vmSubnetZoneMap = map[string]string{}

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

	// Get Zone ID of the Requested Subnet (For Zone-based control)
	vpcHandler := NcpVPCHandler{
		CredentialInfo: vmHandler.CredentialInfo,
		RegionInfo: 	vmHandler.RegionInfo,
		VMClient:   	vmHandler.VMClient,
	}
	reqZoneId, err := vpcHandler.getSubnetZone(vmReqInfo.VpcIID, vmReqInfo.SubnetIID)
	if err != err {
		cblogger.Errorf("Failed to Get the Subnet Zone info!! : [%v]", err)
		return irs.VMInfo{}, err
	}
	cblogger.Infof("\n\n### reqZoneId : [%s]", reqZoneId)

	reqZoneNo, err := vmHandler.getZoneNo(vmHandler.RegionInfo.Region, reqZoneId) // Not vmHandler.RegionInfo.Zone
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Zone 'No' of the Zone : ", err)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}

	// CAUTION!! : Instance Name is Converted to lowercase.(strings.ToLower())
	// NCP에서는 VM instance 이름에 영문 대문자 허용 안되므로 여기서 변환하여 반영.(대문자가 포함되면 Error 발생)
	instanceName := strings.ToLower(vmReqInfo.IId.NameId)
	instanceType := vmReqInfo.VMSpecName
	keyPairId := vmReqInfo.KeyPairIID.SystemId
	minCount := ncloud.Int32(1)

	// Check whether the VM name exists. Search by instanceName converted to lowercase
	vmId, err := vmHandler.getVmIdByName(instanceName)
	if err != nil {
		cblogger.Debug("The VM with the name does not exists : " + instanceName)
		// return irs.VMInfo{}, err  //Caution!!
    }
	if vmId != "" {
		cblogger.Info("The vmId : ", vmId)
		createErr := fmt.Errorf("VM has the name '%s' already exist!!", vmReqInfo.IId.NameId)
		LoggingError(callLogInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	// Set S/G IID (SystemId based)
	var newSecurityGroupIds []*string
	for _, sgID := range vmReqInfo.SecurityGroupIIDs {
		// cblogger.Infof("Security Group IID : [%s]", sgID)
		newSecurityGroupIds = append(newSecurityGroupIds, ncloud.String(sgID.SystemId))
	}

	// Set cloud-init script
	var publicImageId string
	var myImageId string
	var initUserData *string

	if vmReqInfo.ImageType == irs.PublicImage || vmReqInfo.ImageType == "" || vmReqInfo.ImageType == "default" {
		imageHandler := NcpImageHandler{
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
			rtnErr := logAndReturnError(callLogInfo, "Failed to Check Whether the Image is MS Windows Image : ", err)
			return irs.VMInfo{}, rtnErr
		}
		if isPublicWindowsImage {
			var createErr error
			initUserData, createErr = vmHandler.createWinInitUserData(vmReqInfo.VMUserPasswd)
			if createErr != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Create Cloud-Init Script with the Password : ", createErr)
				return irs.VMInfo{}, rtnErr
			}
		} else {
			var createErr error
			initUserData, createErr = vmHandler.createLinuxInitUserData(vmReqInfo.ImageIID, keyPairId)
			if createErr != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Create Cloud-Init Script with the KeyPairId : ", createErr)
				return irs.VMInfo{}, rtnErr
			}
		}
	} else {
		imageHandler := NcpImageHandler{
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

		myImageHandler := NcpMyImageHandler{
			RegionInfo:  vmHandler.RegionInfo,
			VMClient:    vmHandler.VMClient,
		}
		isMyWindowsImage, err := myImageHandler.CheckWindowsImage(vmReqInfo.ImageIID)
		if err != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Check Whether My Image is MS Windows Image : ", err)
			return irs.VMInfo{}, rtnErr
		}
		if isMyWindowsImage {
			var createErr error
			initUserData, createErr = vmHandler.createWinInitUserData(vmReqInfo.VMUserPasswd)
			if createErr != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Create Cloud-Init Script with the Password : ", createErr)
				return irs.VMInfo{}, rtnErr
			}
		} else {
			var createErr error
			initUserData, createErr = vmHandler.createLinuxInitUserData(vmReqInfo.ImageIID, keyPairId)
			if createErr != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Create Cloud-Init Script with the KeyPairId : ", createErr)
				return irs.VMInfo{}, rtnErr
			}
		}
	}
	// cblogger.Info("### Succeeded in Creating Init UserData!!")
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
		ZoneNo: 								reqZoneNo, 			// For Zone-based control
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
	createTagResult, error := vmHandler.createVPCnSubnetTag(ncloud.String(newVMIID.SystemId), vpcNameId, subnetNameId)
	if error != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Create VPC and Subnet Tag : ", error)
		return irs.VMInfo{}, rtnErr
	}
	cblogger.Info("# createTagResult : ", createTagResult)

	// Wait while being created to get VM information.
	curStatus, statusErr := vmHandler.waitToGetVMInfo(newVMIID)
	if statusErr != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Wait to Get the VM info. : ", statusErr)
		return irs.VMInfo{}, rtnErr
	}
	cblogger.Infof("==> VM status of [%s] : [%s]", newVMIID.NameId, curStatus)

	// Create a Public IP for the New VM
	// Caution!!) The number of Public IPs cannot be more than the number of instances on NCP cloud default service.
	time.Sleep(time.Second * 5)
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

	// Create the Tag List on the VM
	if len(vmReqInfo.TagList) > 0 {
		tagHandler := NcpTagHandler {
			RegionInfo:  vmHandler.RegionInfo,
			VMClient:    vmHandler.VMClient,
		}
		_, createErr := tagHandler.createVMTagList(runResult.ServerInstanceList[0].ServerInstanceNo, vmReqInfo.TagList)
		if err != nil {		
			newErr := fmt.Errorf("Failed to Create the Tag List on the VM : [%v]", createErr)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
	}
	
	// Get Created New VM Info
	vmInfo, error := vmHandler.GetVM(newVMIID)
	if error != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the VM Info : ", error)
		return irs.VMInfo{}, rtnErr
	}
	
	if vmInfo.Platform == irs.WINDOWS {
		vmInfo.VMUserPasswd = vmReqInfo.VMUserPasswd
	}
	return vmInfo, nil
}

func (vmHandler *NcpVMHandler) mappingServerInfo(NcpInstance *server.ServerInstance) (irs.VMInfo, error) {
	cblogger.Info("NCP Classic Cloud driver: called mappingServerInfo()!")
	InitLog()
	// callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "mappingServerInfo()", "mappingServerInfo()")

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
			newErr := fmt.Errorf("Failed to Create PublicIp : ", err)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
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
			newErr := fmt.Errorf("Get PublicIp InstanceList : ", err)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
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
		newErr := fmt.Errorf("Failed to Get Block Storage List : ", err)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr		
	}
	if len(blockStorageResult.BlockStorageInstanceList) < 1 {
		cblogger.Errorf("Failed to Find Any BlockStorageInstance!!")
		return irs.VMInfo{}, errors.New("Failed to Find Any BlockStorageInstance.")
	}
	cblogger.Info("Succeeded in Getting Block Storage InstanceList!!")
	// spew.Dump(blockStorageResult.BlockStorageInstanceList[0])

	var sgList []irs.IID
	if NcpInstance.AccessControlGroupList != nil {
		for _, acg := range NcpInstance.AccessControlGroupList {
			sgList = append(sgList, irs.IID{NameId: *acg.AccessControlGroupName, SystemId: *acg.AccessControlGroupConfigurationNo})
		}
	} else {
		cblogger.Info("AccessControlGroupList is empty or nil")
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
			{Key: "ServerImageName", Value: *NcpInstance.ServerImageName},
			{Key: "ZoneCode", Value: *NcpInstance.Zone.ZoneCode},
			//{Key: "ZoneNo", Value: *NcpInstance.Zone.ZoneNo},
			{Key: "PublicIpID", Value: *publicIpInstanceNo}, // # To use it when delete the PublicIP
		},
	}

	imageHandler := NcpImageHandler{
		RegionInfo:  vmHandler.RegionInfo,
		VMClient:    vmHandler.VMClient,
	}
	// Set the VM Image Type : 'PublicImage' or 'MyImage'
	if !strings.EqualFold(*NcpInstance.ServerDescription, "") {
		vmInfo.ImageIId.SystemId = *NcpInstance.ServerDescription // Note!! : Since MyImage ID is not included in the 'NcpInstance' info 
		vmInfo.ImageIId.NameId = *NcpInstance.ServerDescription
		
		isPublicImage, err := imageHandler.isPublicImage(irs.IID{SystemId: *NcpInstance.ServerDescription}) // Caution!! : Not '*NcpInstance.ServerImageProductCode'
		if err != nil {
			newErr := fmt.Errorf("Failed to Check Whether the Image is Public Image : [%v]", err)
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

	storageSize, deviceName, err := vmHandler.getVmRootDiskInfo(NcpInstance.ServerInstanceNo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find BlockStorage Info : ", err)
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
		newErr := fmt.Errorf("Failed to Get Data Disk List : ", err)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}
	if len(dataDiskList) > 0 {
		vmInfo.DataDiskIIDs = dataDiskList
	}

	// Set VM Zone Info
	if NcpInstance.Zone != nil {
		vmInfo.Region.Zone = *NcpInstance.Zone.ZoneCode
		// *NcpInstance.Zone.ZoneCode or *NcpInstance.Zone.ZoneName etc...
	}

	// Get the VPC Name from Tag of the VM
	vpcName, subnetName, error := vmHandler.getVPCnSubnetNameFromTag(NcpInstance.ServerInstanceNo)
	if error != nil {
		newErr := fmt.Errorf("Failed to Get VPC Name from Tag of the VM instance!! : [%v]", error)
		cblogger.Debug(newErr.Error())
		// return irs.VMInfo{}, newErr // Caution!!
	}
	// cblogger.Infof("# vpcName : [%s]", vpcName)
	// cblogger.Infof("# subnetName : [%s]", subnetName)

	if len(vpcName) < 1 {
		cblogger.Debug("Failed to Get VPC Name from Tag!!")
	} else {
		// Get the VPC info
		vpcHandler := NcpVPCHandler {
			RegionInfo:			vmHandler.RegionInfo,
			VMClient:         	vmHandler.VMClient,
		}
		vpcInfo, err := vpcHandler.GetVPC(irs.IID{SystemId: vpcName})
		if err != nil {
			newErr := fmt.Errorf("Failed to Find the VPC : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}

		vmInfo.VpcIID.NameId = vpcName
		vmInfo.VpcIID.SystemId = vpcInfo.IId.SystemId
		vmInfo.SubnetIID.NameId = subnetName
		for _, curSubnet := range vpcInfo.SubnetInfoList {
			// cblogger.Infof("Subnet NameId : [%s]", curSubnet.IId.NameId)
			if strings.EqualFold(curSubnet.IId.NameId, subnetName) {
				vmInfo.SubnetIID.SystemId = curSubnet.IId.SystemId
				break
			}
		}
	}
	// cblogger.Infof("NCP Instance Uptime : [%s]", *NcpInstance.Uptime)

	// Note : NCP VPC PlatformType : LNX32, LNX64, WND32, WND64, UBD64, UBS64
	if strings.Contains(*NcpInstance.PlatformType.Code, "LNX") || strings.Contains(*NcpInstance.PlatformType.Code, "UB") {
		vmInfo.VMUserId = lnxUserName
		vmInfo.Platform = irs.LINUX_UNIX
	} else if strings.Contains(*NcpInstance.PlatformType.Code, "WND") {
		vmInfo.VMUserId = winUserName
		vmInfo.Platform = irs.WINDOWS
	}

	// Get the Tag List of the VM
	var kvList []irs.KeyValue
	tagHandler := NcpTagHandler {
		RegionInfo:  vmHandler.RegionInfo,
		VMClient:    vmHandler.VMClient,
	}
	tagList, err := tagHandler.getVMTagListWithVMId(NcpInstance.ServerInstanceNo)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Get the Tag List with the VM SystemID : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}	
	if len(tagList) > 0 {
		for _, curTag := range tagList {
			kv := irs.KeyValue {
				Key : 	ncloud.StringValue(curTag.TagKey),
				Value:  ncloud.StringValue(curTag.TagValue),
			}
			kvList = append(kvList, kv)
		}
		vmInfo.TagList = kvList
	}

	return vmInfo, nil
}

func (vmHandler *NcpVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetVM()!!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "GetVM()")

	cblogger.Infof("\n ### vmHandler.RegionInfo.TargetZone : [%s]", vmHandler.RegionInfo.TargetZone)

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM System ID!!")
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}

	subnetZone, err := vmHandler.getVMSubnetZone(&vmIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Subnet Zone info of the VM!! : [%v]", err)
		cblogger.Debug(newErr.Error())
		// return irs.VMInfo{}, newErr  // Caution!!
	}

	var ncpVMInfo *server.ServerInstance
	vmErr := errors.New("")

	if strings.EqualFold(vmHandler.RegionInfo.Zone, subnetZone){ // Not vmHandler.RegionInfo.Zone
		ncpVMInfo, vmErr = vmHandler.getNcpVMInfo(vmIID.SystemId)
		if vmErr != nil {
			newErr := fmt.Errorf("Failed to Get the VM Info of the Zone : [%s], [%v]", subnetZone, vmErr)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
	} else {
		ncpVMInfo, vmErr = vmHandler.getNcpTargetZoneVMInfo(&vmIID.SystemId)
		if vmErr != nil {
			newErr := fmt.Errorf("Failed to Get the VM Info of the Zone : [%s], [%v]", subnetZone, vmErr)
			cblogger.Error(newErr.Error())
			return irs.VMInfo{}, newErr
		}
	}

	vmInfo, err := vmHandler.mappingServerInfo(ncpVMInfo)
	if err != nil {
		LoggingError(callLogInfo, err)
		return irs.VMInfo{}, err
	}
	return vmInfo, nil
}

func (vmHandler *NcpVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCP Classic Cloud driver: called SuspendVM()!!")
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
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "TerminateVM()")

	dataDiskList, err := vmHandler.getVmDataDiskList(ncloud.String(vmIID.SystemId))
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

		// To Delete Tags of the VM instance
		Result, error := vmHandler.deleteVMTags(serverInstanceNo[0])
		if error != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Delete Tags of the VM instance! : ", error)
			return irs.VMStatus("Failed. "), rtnErr
		} else {
			cblogger.Infof("# DeleteVMTags Result : [%t]", Result)
		}
		
		// If the NCP instance has a 'Public IP', delete it after termination of the instance.
		if ncloud.String(vmInfo.PublicIP) != nil {
			// Delete the PublicIP of the VM
			vmStatus, err := vmHandler.deletePublicIP(vmInfo)
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
		cblogger.Info(*runResult.ReturnMessage)

		// To Delete Tags of the VM instance
		Result, error := vmHandler.deleteVMTags(serverInstanceNo[0])
		if error != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Delete Tags of the VM instance!! : ", error)
			return irs.VMStatus("Failed. "), rtnErr
		} else {
			cblogger.Infof("# DeleteVMTags Result : [%t]", Result)
		}				

		// If the NCP instance has a 'Public IP', delete it after termination of the instance.
		if ncloud.String(vmInfo.PublicIP) != nil {
			// PublicIP 삭제
			vmStatus, err := vmHandler.deletePublicIP(vmInfo)
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

func convertVMStatusString(vmStatus string) (irs.VMStatus, error) {
	var resultStatus string

	// cblogger.Infof("NCP VM Status : [%s]", vmStatus)
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
		newErr := fmt.Errorf("No mapping information found matching with the vmStatus [%s].", string(vmStatus))
		cblogger.Error(newErr.Error())
		return irs.VMStatus("Failed. "), newErr
	}
	cblogger.Infof("Succeeded in Converting the VM Status : [%s] ==> [%s]", vmStatus, resultStatus)
	return irs.VMStatus(resultStatus), nil
}

func (vmHandler *NcpVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetVMStatus()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmIID.NameId, "GetVMStatus()")

	cblogger.Infof("### Target Zone : [%s]", vmHandler.RegionInfo.TargetZone)

	subnetZone, err := vmHandler.getVMSubnetZone(&vmIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Subnet Zone info of the VM!! : [%v]", err)
		cblogger.Debug(newErr.Error())
		// return irs.VMInfo{}, newErr  // Caution!!
	}

	regionNo, err := vmHandler.getRegionNo(vmHandler.RegionInfo.Region)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Region No :", err)
		return irs.VMStatus(""), rtnErr
	}
	// Note) ~.RegionInfo.TargetZone is specified by CB-Spider Server.
	reqZoneNo, err := vmHandler.getZoneNo(vmHandler.RegionInfo.Region, subnetZone) // Not vmHandler.RegionInfo.Zone
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Zone No of the Zone Code :", err)
		return irs.VMStatus(""), rtnErr
	}
	instanceReq := server.GetServerInstanceListRequest{
		ServerInstanceNoList: []*string{
			ncloud.String(vmIID.SystemId),
		},
		RegionNo: 				regionNo,
		ZoneNo: 				reqZoneNo, // For Zone-based control!!
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
	// cblogger.Info("Succeeded in Getting ServerInstanceList!!")

	vmStatus, errStatus := convertVMStatusString(*result.ServerInstanceList[0].ServerInstanceStatusName)
	// cblogger.Info("# Converted VM Status : " + vmStatus)
	return vmStatus, errStatus
}

func (vmHandler *NcpVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	cblogger.Info("NCP Classic Cloud driver: called ListVMStatus()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "ListVMStatus()", "ListVMStatus()")

	regionNo, err := vmHandler.getRegionNo(vmHandler.RegionInfo.Region)
	if err != nil {
		cblogger.Errorf("Failed to Get RegionNo : [%v]", err)
		return nil, err
	}
	zoneNo, err := vmHandler.getZoneNo(vmHandler.RegionInfo.Region, vmHandler.RegionInfo.Zone)
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
		cblogger.Debug(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Getting ServerInstanceList in the Zone!!")

		var vmStatusList []*irs.VMStatusInfo
		for _, vm := range result.ServerInstanceList {
			//*vm.ServerInstanceStatusName
			//*vm.ServerName
			vmStatus, err := convertVMStatusString(*vm.ServerInstanceStatusName)
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
}

// VM List on 'All Zone' in the Region
func (vmHandler *NcpVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger.Info("NCP Classic Cloud driver: called ListVM()!")

	ncpVMList, err := vmHandler.getNcpVMListWithRegion(vmHandler.RegionInfo.Region)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP VM List : [%v]", err.Error())
		cblogger.Debug(newErr.Error())
		return nil, newErr
	}

	var vmInfoList []*irs.VMInfo
	for _, vm := range ncpVMList {
		vmStatus, statusErr := convertVMStatusString(*vm.ServerInstanceStatusName)
		if statusErr != nil {
			newErr := fmt.Errorf("Failed to Get the Status of VM : [%s], [%v]", *vm.ServerInstanceNo, statusErr.Error())
			cblogger.Debug(newErr.Error())
			return nil, newErr
		} else {
			// cblogger.Infof("Succeeded to Get the Status of VM [%s] : [%s]", *vm.ServerInstanceNo, string(vmStatus))
			cblogger.Infof("===> VM Status : [%s]", string(vmStatus))
		}

		if (string(vmStatus) != "Creating") && (string(vmStatus) != "Terminating") {
			// cblogger.Infof("===> The VM Status not 'Creating' or 'Terminating', you can get the VM info.")
			vmInfo, err := vmHandler.mappingServerInfo(vm)
			if err != nil {
				newErr := fmt.Errorf("Failed to Map the VM info : [%s], [%v]", *vm.ServerInstanceNo, err.Error())
				cblogger.Error(newErr.Error())
				return nil, newErr
			}
			vmInfoList = append(vmInfoList, &vmInfo)
		}
	}
	return vmInfoList, nil
	
/*
	regionNo, err := vmHandler.getRegionNo(vmHandler.RegionInfo.Region)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Region No :", err)
		return nil, rtnErr
	}
	zoneNo, err := vmHandler.getZoneNo(vmHandler.RegionInfo.Region, vmHandler.RegionInfo.Zone)
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
		cblogger.Info("Succeeded in Getting ServerInstanceList from NCP!!")
	}	
	
	var vmInfoList []*irs.VMInfo
	for _, vm := range result.ServerInstanceList {
		curStatus, statusErr := vmHandler.GetVMStatus(irs.IID{SystemId: *vm.ServerInstanceNo})
		if statusErr != nil {
			newErr := fmt.Errorf("Failed to Get the Status of VM : [%s], [%v]", *vm.ServerInstanceNo, statusErr.Error())
			cblogger.Debug(newErr.Error())  // For Zone-based control, different Zone VMs are included.
			return nil, newErr
		} else {
			cblogger.Infof("Succeeded to Get the Status of VM [%s] : [%s]", *vm.ServerInstanceNo, string(curStatus))
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
*/

}

func (vmHandler *NcpVMHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("NCP Classic Cloud driver: called ListIID()!")

	ncpVMList, err := vmHandler.getNcpVMListWithRegion(vmHandler.RegionInfo.Region)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP VM List : [%v]", err.Error())
		cblogger.Debug(newErr.Error())
		return nil, newErr
	}

	var vmIIDList []*irs.IID
	for _, vm := range ncpVMList {
		vmIID := irs.IID{
			NameId:    *vm.ServerName,
			SystemId:  *vm.ServerInstanceNo,
		}
		vmIIDList = append(vmIIDList, &vmIID)
	}
	return vmIIDList, nil
}

// Waiting for up to 300 seconds until VM info. can be get
func (vmHandler *NcpVMHandler) waitToGetVMInfo(vmIID irs.IID) (irs.VMStatus, error) {
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
			cblogger.Infof("===> VM Status : [%s]", curStatus)
		}

		switch string(curStatus) {
		case "Creating":
			curRetryCnt++
			cblogger.Infof("The VM Status is still 'Creating', so wait for a second more before inquiring the VM info.")
			time.Sleep(time.Second * 5)
			if curRetryCnt > maxRetryCnt {
				cblogger.Errorf("Despite waiting for a long time(%d sec), the VM status is %s, so it is forcibly finishied.", maxRetryCnt, curStatus)
				return irs.VMStatus("Failed. "), errors.New("Despite waiting for a long time, the VM status is 'Creating', so it is forcibly finishied.")
			}

		default:
			cblogger.Infof("===> ### The VM Creation has ended, stopping the waiting.")
			return irs.VMStatus(curStatus), nil
			//break
		}
	}
}

// Waiting for up to 300 seconds until Public IP can be deleted.
func (vmHandler *NcpVMHandler) waitToDelPublicIp(vmIID irs.IID) (irs.VMStatus, error) {
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
			cblogger.Infof("The VM Status is still 'Terminating', so wait for a second more before inquiring the VM info.")
			time.Sleep(time.Second * 5)
			if curRetryCnt > maxRetryCnt {
				cblogger.Errorf("Despite waiting for a long time(%d sec), the VM status is '%s', so it is forcibly finished.", maxRetryCnt, curStatus)
				return irs.VMStatus("Failed"), errors.New("Despite waiting for a long time, the VM status is 'Terminating', so it is forcibly finishied.")
			}

		default:
			cblogger.Infof("===>### The VM Termination is finished, so stopping the waiting.")
			return irs.VMStatus(curStatus), nil
			//break
		}
	}
}

// Whenever a VM is terminated, Delete the public IP that the VM has
func (vmHandler *NcpVMHandler) deletePublicIP(vmInfo irs.VMInfo) (irs.VMStatus, error) {
	cblogger.Info("NCP Classic Cloud driver: called deletePublicIP()!")

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
	curStatus, errStatus := vmHandler.waitToDelPublicIp(vmInfo.IId)
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

func (vmHandler *NcpVMHandler) getVmIdByName(vmNameID string) (string, error) {
	cblogger.Info("NCP Classic Cloud driver: called getVmIdByName()!")

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

// Save VPC/Subnet Name info as Tags on the VM
func (vmHandler *NcpVMHandler) createVPCnSubnetTag(vmID *string, vpcName string, subnetName string) (bool, error) {
	cblogger.Info("NCP Classic Cloud driver: called createVPCnSubnetTag()!")

	tagKey := []string {"VPCName", "SubnetName"}

	tagHandler := NcpTagHandler {
		RegionInfo:  vmHandler.RegionInfo,
		VMClient:    vmHandler.VMClient,
	}
	tagKVList := []irs.KeyValue {
		{
			Key: 	tagKey[0], 
			Value: 	vpcName,
		}, 
		{
			Key: 	tagKey[1], 
			Value: 	subnetName,
		},
	}
	_, err := tagHandler.createVMTagList(vmID, tagKVList)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Add New Tag List : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	return true, nil
}

func (vmHandler *NcpVMHandler) getVPCnSubnetNameFromTag(vmID *string) (string, string, error) {
	cblogger.Info("NCP Classic Cloud driver: called getVPCnSubnetNameFromTag()!")

	tagHandler := NcpTagHandler {
		RegionInfo:  vmHandler.RegionInfo,
		VMClient:    vmHandler.VMClient,
	}
	tagList, err := tagHandler.getVMTagListWithVMId(vmID)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Get the Tag List with the VM SystemID : [%v]", err)
		cblogger.Debug(newErr.Error())
		return "", "", newErr
	}
	if len(tagList) < 1 {
		newErr := fmt.Errorf("Failed to Get Any Tag info with the VM SystemID!!")
		return "", "", newErr
	}

	var kvList []irs.KeyValue
	for _, curTag := range tagList {
		kv := irs.KeyValue {
			Key : 	ncloud.StringValue(curTag.TagKey),
			Value:  ncloud.StringValue(curTag.TagValue),
		}
		kvList = append(kvList, kv)
	}

	var vpcName string
	var subnetName string
	for _, kv := range kvList {
		if strings.EqualFold(kv.Key, "VPCName") {			
			vpcName = kv.Value
		} else if strings.EqualFold(kv.Key, "SubnetName") {			
			subnetName = kv.Value
		}
	}
	if len(vpcName) < 1 {
		newErr := fmt.Errorf("Failed to Get VPC Name from the Tag!!")
		cblogger.Debug(newErr.Error())
		return "", "", newErr
	}
	if len(subnetName) < 1 {
		newErr := fmt.Errorf("Failed to Get Subnet Name from the Tag!!")
		cblogger.Debug(newErr.Error())
		return "", "", newErr
	}
	return vpcName, subnetName, nil
}

func (vmHandler *NcpVMHandler) deleteVMTags(vmID *string) (bool, error) {
	cblogger.Info("NCP Classic Cloud driver: called deleteVMTags()!")

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

func (vmHandler *NcpVMHandler) getNcpVMInfo(vmId string) (*server.ServerInstance, error) {
	cblogger.Info("NCP Classic Cloud driver: called getNcpVMInfo()")
	
	if strings.EqualFold(vmId, "") {
		newErr := fmt.Errorf("Invalid VM ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	regionNo, err := vmHandler.getRegionNo(vmHandler.RegionInfo.Region)
	if err != nil {
		cblogger.Errorf("Failed to Get RegionNo : [%v]", err)
		return nil, err
	}
	zoneNo, err := vmHandler.getZoneNo(vmHandler.RegionInfo.Region, vmHandler.RegionInfo.Zone)
	if err != nil {
		cblogger.Errorf("Failed to Get NCP Zone No of the Zone Code : [%v]", err)
		return nil, err
	}
	instanceReq := server.GetServerInstanceListRequest{
		ServerInstanceNoList: 	[]*string{ncloud.String(vmId)},
		RegionNo: 				regionNo,	
		ZoneNo: 				zoneNo,
	}
	result, err := vmHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find VM list with the SystemId from NCP : [%s], [%v]", vmId, err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	if len(result.ServerInstanceList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any VM info with the SystemId : [%s]", vmId)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	return result.ServerInstanceList[0], nil
}

func (vmHandler *NcpVMHandler) getNcpTargetZoneVMInfo(vmId *string) (*server.ServerInstance, error) {
	cblogger.Info("NCP Classic Cloud driver: called getNcpTargetZoneVMInfo()")

	if strings.EqualFold(*vmId, "") {
		newErr := fmt.Errorf("Invalid VM ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	subnetZone, err := vmHandler.getVMSubnetZone(vmId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Subnet Zone info of the VM!! : [%v]", err)
		cblogger.Debug(newErr.Error())
		// return irs.VMInfo{}, newErr  // Caution!!
	}

	regionNo, err := vmHandler.getRegionNo(vmHandler.RegionInfo.Region)
	if err != nil {
		cblogger.Errorf("Failed to Get RegionNo : [%v]", err)
		return nil, err
	}
	// Note) ~.RegionInfo.TargetZone is specified by CB-Spider Server.
	reqZoneNo, err := vmHandler.getZoneNo(vmHandler.RegionInfo.Region, subnetZone) // Not vmHandler.RegionInfo.Zone
	if err != nil {
		cblogger.Errorf("Failed to Get NCP Zone No of the Zone Code : [%v]", err)
		return nil, err
	}
	instanceReq := server.GetServerInstanceListRequest{
		ServerInstanceNoList: 	[]*string{vmId},
		RegionNo: 				regionNo,	
		ZoneNo: 				reqZoneNo, // For Zone-based control!!
	}
	result, err := vmHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find VM list with the SystemId from NCP : [%s], [%v]", *vmId, err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	if len(result.ServerInstanceList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any VM info with the SystemId : [%s]", *vmId)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	return result.ServerInstanceList[0], nil
}

func (vmHandler *NcpVMHandler) getVmRootDiskInfo(vmId *string) (*string, *string, error) {
	cblogger.Info("NCPVPC Cloud driver: called getVmRootDiskInfo()!!")

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

func (vmHandler *NcpVMHandler) getVmDataDiskList(vmId *string) ([]irs.IID, error) {
	cblogger.Info("NCP Classic Cloud driver: called getVmDataDiskList()")

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

func (vmHandler *NcpVMHandler) createLinuxInitUserData(imageIID irs.IID, keyPairId string) (*string, error) {
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

func (vmHandler *NcpVMHandler) createWinInitUserData(passWord string) (*string, error) {
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
func (vmHandler *NcpVMHandler) getRegionNo(regionCode string) (*string, error) {
	cblogger.Info("NCP Classic Cloud driver: called getRegionNo()!")
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
func (vmHandler *NcpVMHandler) getZoneNo(regionCode string, zoneCode string) (*string, error) {
	cblogger.Info("NCP Classic Cloud driver: called getZoneNo()!")
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

	regionNo, getErr := vmHandler.getRegionNo(regionCode)
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

// Check the global 'regionMap' parameter and Populate the map with the regions data.
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
		cblogger.Infof("Region No '%s' not found from the Map. So Set ZoneNo List!!\n", regionNo)

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

// Get NCP VM info list (in the All zone of the specified Region)
func (vmHandler *NcpVMHandler) getNcpVMListWithRegion(regionCode string) ([]*server.ServerInstance, error) {
	cblogger.Info("KT Cloud Driver: called getNcpVMListWithRegion()")
	
	regionNo, err := vmHandler.getRegionNo(vmHandler.RegionInfo.Region)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Region No :", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	regionZoneHandler := NcpRegionZoneHandler{
		CredentialInfo: vmHandler.CredentialInfo,
		RegionInfo:    	vmHandler.RegionInfo,
		VMClient:      	vmHandler.VMClient,
	}
	// Get Zone Name(Zone Code) list (in the Region)
	regionInfo, err := regionZoneHandler.GetRegionZone(regionCode)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Region Info : [%v]", err.Error())
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Get NCP VM info list on 'Every Zone' in the Region
	var ncpVMList []*server.ServerInstance
	for _, zone := range regionInfo.ZoneList {
		// For Zone-based control!!
		// connInfo := idrv.ConnectionInfo {			
		// 	RegionInfo: idrv.RegionInfo {
		// 		Region: 		vmHandler.RegionInfo.Region,
		// 		Zone: 			zone.Name, // Not vmHandler.RegionInfo.Zone
		// 	},
		// 	CredentialInfo: idrv.CredentialInfo {
		// 		ClientId: 		vmHandler.CredentialInfo.ClientId,
		// 		ClientSecret: 	vmHandler.CredentialInfo.ClientSecret,
		// 	},
		// }
		// newClient, err := createClient(connInfo)
		// if err != nil {
		// 	newErr := fmt.Errorf("Failed to Create Client : [%v]", err.Error())
		// 	cblogger.Error(newErr.Error())
		// 	return nil, newErr
		// }

		zoneNo, err := vmHandler.getZoneNo(vmHandler.RegionInfo.Region, zone.Name) // Not vmHandler.RegionInfo.Zone
		if err != nil {
			newErr := fmt.Errorf("Failed to Get NCP Zone No of the Zone Code :", err)
			cblogger.Error(newErr.Error())
			return nil, newErr
		}
	
		instanceReq := server.GetServerInstanceListRequest{
			ServerInstanceNoList: 	[]*string{},
			RegionNo:             	regionNo,
			ZoneNo: 				zoneNo,
		}
		result, err := vmHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get GetServerInstanceList : [%v]", err.Error())
			cblogger.Error(newErr.Error())
			return nil, newErr
		}
		ncpVMList = append(ncpVMList, result.ServerInstanceList...)
	}
	// cblogger.Info("\n\n### ncpVMList : \n")
	// spew.Dump(ncpVMList)

	if len(ncpVMList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any VM in the Region!!")
		cblogger.Debug(newErr.Error())
		return nil, newErr
	}
	return ncpVMList, nil
}

func (vmHandler *NcpVMHandler) getVMSubnetZone(vmId *string) (string, error) {
	cblogger.Info("KT Cloud Driver: called getVMSubnetZone()")

	if strings.EqualFold(*vmId, "") {
		newErr := fmt.Errorf("Invalid Region Code!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	// Populate the global map : vmSubnetZoneMap
	err := vmHandler.checkAndSetVMSubnetZoneInfo(vmId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Init VM SubnetZone Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	subnetZone, exists := vmSubnetZoneMap[*vmId]
	if exists {
		return subnetZone, nil
	} else {
		newErr := fmt.Errorf("Failed to Find the Subnet Zone Info that corresponds to the VM.")
		cblogger.Error(newErr.Error())
		return "", newErr
	}
}

// Check the global 'vmSubnetZoneMap' parameter and Populate the map with the subnet zone info of the VM.
func (vmHandler *NcpVMHandler) checkAndSetVMSubnetZoneInfo(vmId *string) error {
	cblogger.Info("NCP Classic Cloud driver: called checkAndSetVMSubnetZoneInfo()!")

	if strings.EqualFold(*vmId, "") {
		newErr := fmt.Errorf("Invalid VM ID!!")
		cblogger.Error(newErr.Error())
		return newErr
	}
	
	subnetZone, exists := vmSubnetZoneMap[*vmId]
	if exists {
		cblogger.Infof("# The VM '%s' has Subnet Zone Info : %s", *vmId, subnetZone)
	} else {
		cblogger.Infof("# Subnet Zone Info not found. So Set the Info!!")

		// Get the VPC Name from Tag of the VM
		vpcName, subnetName, error := vmHandler.getVPCnSubnetNameFromTag(vmId)
		if error != nil {
			newErr := fmt.Errorf("Failed to Get VPC Name from Tag of the VM instance!! : [%v]", error)
			cblogger.Debug(newErr.Error())
			// return irs.VMInfo{}, newErr  // Caution!!
		}
		// cblogger.Infof("# vpcName : [%s]", vpcName)
		// cblogger.Infof("# subnetName : [%s]", subnetName)

		var subnetZoneId string
		getErr := errors.New("")

		if strings.EqualFold(vpcName, "") || strings.EqualFold(subnetName, ""){
			cblogger.Debug("Failed to Get the VPC and Subnet Name from Tag!!")
		} else {
			// Get Zone ID of the Requested Subnet
			vpcHandler := NcpVPCHandler {
				CredentialInfo: 	vmHandler.CredentialInfo,
				RegionInfo:			vmHandler.RegionInfo,
				VMClient:         	vmHandler.VMClient,
			}
			subnetZoneId, getErr = vpcHandler.getSubnetZone(irs.IID{SystemId: vpcName}, irs.IID{SystemId: subnetName})
			if getErr != nil {
				newErr := fmt.Errorf("Failed to Get the Subnet Zone info!! : [%v]", getErr)
				cblogger.Debug(newErr.Error())
				return newErr
			}
			// cblogger.Infof("\n\n### subnetZoneId : [%s]", subnetZoneId)
		}

		if strings.EqualFold(subnetZoneId, "") {
			newErr := fmt.Errorf("Failed to Get the Subnet Zone ID of the VM!!")
			cblogger.Error(newErr.Error())
			return newErr
		}

		vmSubnetZoneMap[*vmId] = subnetZoneId
	}

	return nil
}
