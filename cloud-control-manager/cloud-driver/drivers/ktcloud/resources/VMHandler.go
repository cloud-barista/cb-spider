// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud VM Handler
//
// by ETRI, 2021.05.
// Updated by ETRI, 2023.11.

package resources

import (
	"os"
	"errors"
	"fmt"
	"strconv"
	// "encoding/base64"
	// "log"
	"io"
	"strings"
	"time"
	"github.com/davecgh/go-spew/spew"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	
	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"
)

type KtCloudVMHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	Client         *ktsdk.KtCloudClient
}

const (
	vmUserName string = "cb-user"
	cloudInitFilePath string = "/cloud-driver-libs/.cloud-init-ktcloud/cloud-init"
)

// Already declared in CommonNcpFunc.go
// var cblogger *logrus.Logger
func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud VM Handler")
}

func (vmHandler *KtCloudVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called StartVM()!!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, vmReqInfo.IId.NameId, "StartVM()")

	securityHandler := KtCloudSecurityHandler{
		CredentialInfo: vmHandler.CredentialInfo,
		RegionInfo:		vmHandler.RegionInfo,
		Client:         vmHandler.Client,
	}

	zoneId := vmHandler.RegionInfo.Zone
	cblogger.Info("RegionInfo.Zone : ", zoneId)

	instanceName := vmReqInfo.IId.NameId
	instanceHostName := vmReqInfo.IId.NameId
	instanceSpecId := vmReqInfo.VMSpecName
	instanceImageId := vmReqInfo.ImageIID.SystemId
	keyPairId := vmReqInfo.KeyPairIID.SystemId	
	usagePlanType := "hourly" // KT Cloud Rate Type (default : hourly)

	var sgSystemIDs []string
	for _, sgIID := range vmReqInfo.SecurityGroupIIDs {
		cblogger.Infof("S/G ID : [%s]", sgIID)
		sgSystemIDs = append(sgSystemIDs, sgIID.SystemId)
	}
	cblogger.Infof("The SystemIds of the Security Group IIDs : [%s]", sgSystemIDs)

	// # To Check if the Requested S/G exits	
	for _, sgSystemID := range sgSystemIDs {
		cblogger.Infof("S/G System ID : [%s]", sgSystemID)
		sgInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: sgSystemID})
		if err != nil {
			cblogger.Errorf("Failed to Find the S/G on the Zone : %s", sgSystemID)
			return irs.VMInfo{}, err
		}
		// cblogger.Info("\n*sgInfo : ")
		// spew.Dump(sgInfo)

		if len(*sgInfo.SecurityRules) < 1 {
			cblogger.Errorf("Failed to Find Any Security Rule of [%s]", sgSystemID)
			return irs.VMInfo{}, err
		}
	}

	// Preparing for UserData String
	filePath := os.Getenv("CBSPIDER_ROOT") + cloudInitFilePath
	openFile, err := os.Open(filePath)
    if err != nil {
		cblogger.Errorf("Failed to Find and Open the file : [%v]", err)
		return irs.VMInfo{}, err
    }
	cblogger.Info("Succeeded in Finding and Opening the S/G file!!")
    defer openFile.Close()

	cmdStringByte, readErr := io.ReadAll(openFile)
	if readErr != nil {
		cblogger.Errorf("Failed to Read the file : [%v]", readErr)
    }
	cmdString := string(cmdStringByte)
	// Set cloud-init script
	cmdString = strings.ReplaceAll(cmdString, "{{username}}", vmUserName)

	cblogger.Infof("cmdString : [%s]", cmdString)
	// (Caution!!) Upon deployVirtualMachine() request, Base64 encoding of UserData is not required because Base64 encoding is applied in KT Cloud SDK GO.

	// Check whether the VM name exists
	vmId, nameCheckErr := vmHandler.GetVmIdWithName(instanceName)
	if vmId != "" {
		cblogger.Errorf("Failed to Create the VM. The VM Name already Exists!! : [%s]", instanceName)
		return irs.VMInfo{}, nameCheckErr
	}

	cblogger.Infof("instanceImageId : [%s]", instanceImageId)

	ktVMSpecId, ktDiskOfferingId, DiskSize := GetKTVMSpecIdAndDiskSize(instanceSpecId)
	cblogger.Infof("vmSpecID : [%s]", ktVMSpecId)
	cblogger.Infof("ktDiskOfferingId : [%s]", ktDiskOfferingId)
	cblogger.Infof("DiskSize : [%s]", DiskSize)

	cblogger.Info("\n\n### Starting VM Creation process!!")
	newVMReqInfo := ktsdk.DeployVMReqInfo {
		ZoneId: 			zoneId,
		ServiceOfferingId:  ktVMSpecId,
		TemplateId: 		instanceImageId,
		DiskOfferingId: 	ktDiskOfferingId, // ***Data disk로 Linux 계열은 80GB 추가***
		//ProductCode: 		"",
		VMHostName: 		instanceHostName,
		DisplayName: 		instanceName,
		UsagePlanType: 		usagePlanType,
		RunSysPrep: 		false,
		KeyPair: 			keyPairId,
		UserData:			cmdString,
	}
	callLogStart := call.Start()
	newVM, err := vmHandler.Client.DeployVirtualMachine(newVMReqInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New VM instance : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)
	//spew.Dump(newVM)
	
	jobResult, err := vmHandler.Client.QueryAsyncJobResult(newVM.Deployvirtualmachineresponse.JobId)
	if err != nil {
		cblogger.Errorf("Failed to Get the Job Result: [%v]", err)
		return irs.VMInfo{}, err
	}
	cblogger.Info("\n### QueryAsyncJobResultResponse : ")
	// spew.Dump(jobResult.Queryasyncjobresultresponse.JobResult.Virtualmachine[0])
	// spew.Dump(jobResult.Queryasyncjobresultresponse.JobResult)
	spew.Dump(jobResult)

	if strings.EqualFold(newVM.Deployvirtualmachineresponse.ID, "") {
		cblogger.Error("Failed to Find the VM Instance ID!!")
	} else {
		// cblogger.Info("Start Get VM Status...")
		// vmStatus, err := vmHandler.GetVMStatus(newVM.Deployvirtualmachineresponse.ID)
		// if err != nil {
		// 	cblogger.Errorf("[%s] Failed to get VM Status", newVM.Deployvirtualmachineresponse.ID)
		// } else {
		// 	cblogger.Infof("[%s] Succeeded to get VM Status : [%s]", newVM.Deployvirtualmachineresponse.ID, vmStatus)
		// }

		newVMIID := irs.IID{SystemId: newVM.Deployvirtualmachineresponse.ID}
		cblogger.Infof("Created New VM Instance ID : [%s]", newVMIID)

		// Wait for VM information to be inquired (until when VM status is Running)
		curStatus, errStatus := vmHandler.WaitToGetInfo(newVMIID)
		if errStatus != nil {
			cblogger.Error(errStatus.Error())
			return irs.VMInfo{}, errStatus
		}
		cblogger.Infof("==> The VM Status of : [%s] , : [%s]", newVMIID, curStatus)

		//Check Job Status of Deploy virtualmachine to Confirm the termination of new VM deployment process (Wait 700sec)
		waitErr := vmHandler.Client.WaitForAsyncJob(newVM.Deployvirtualmachineresponse.JobId, 700000000000)
		if waitErr != nil {
			cblogger.Errorf("Failed to Wait the Job : [%v]", waitErr)
			return irs.VMInfo{}, waitErr
		}	

		// To require the New VM info.
		vmListReqInfo := ktsdk.ListVMReqInfo{
			ZoneId: 	vmHandler.RegionInfo.Zone,
			VMId:       newVM.Deployvirtualmachineresponse.ID,
		}
		callLogStart := call.Start()
		result, err := vmHandler.Client.ListVirtualMachines(vmListReqInfo)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the VM Instance info : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
		LoggingInfo(callLogInfo, callLogStart)
	
		if len(result.Listvirtualmachinesresponse.Virtualmachine) < 1 {
			return irs.VMInfo{}, errors.New("Failed to Find the VM Instance ID '" + newVM.Deployvirtualmachineresponse.ID + "'!!")
		}
		// cblogger.Infof("==> \n### result : [%s]", result.Listvirtualmachinesresponse.Virtualmachine[0])
		// spew.Dump(result)

		publicIp, err := vmHandler.AssociateIpAddress()
		if err != nil {
			cblogger.Errorf("Failed to Create New Public IP : [%v]", err)	
			return irs.VMInfo{}, err
		}

		// To get list of the PulbicIP info
		callLogStart2 := call.Start()
		ipListReqInfo := ktsdk.ListPublicIpReqInfo {}
		ipListResponse, err := vmHandler.Client.ListPublicIpAddresses(ipListReqInfo)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get PublicIP List : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}
		LoggingInfo(callLogInfo, callLogStart2)
		
		// To get the publicIP 'ID' according to the publicIP Address from the PublicIP list
		var publicIpId string
		for _, ipAddress := range ipListResponse.Listpublicipaddressesresponse.PublicIpAddress {
			if ipAddress.IpAddress == publicIp {
				publicIpId = ipAddress.ID
				break
			}
		}
		cblogger.Infof("==> The PublicIP ID : [%s]", publicIpId)

		// Caution!!) If execute DeleteFirewall(), PortFording rule also deleted via KT Cloud API           
		// Delete Firewall Rule(Open : tcp/22) created when setting PORT Forwarding.
		// The port No. 22 is opened already when the PortFording rule is created.
		// _, error := vmHandler.DeleteFirewall(publicIpId)
		// if error != nil {
		// 	cblogger.Error(error.Error())
	
		// 	return irs.VMInfo{}, err
		// } else {
		// 	cblogger.Info("Succeeded in Deleting the Firewall rules!!")
		// }

		_, ruleErr := vmHandler.CreatePortForwardingFirewallRules(sgSystemIDs, publicIpId, newVM.Deployvirtualmachineresponse.ID) 
		if ruleErr != nil {
			newErr := fmt.Errorf("Failed to Create PortForwarding Rules and Firewall Rules : [%v]", ruleErr)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.VMInfo{}, newErr
		}

		// Converts string slice to string
		sgIDsString := strings.Join(sgSystemIDs, ",")
		vmTags:= []ktsdk.TagArg {
			{
				Key: "vpcId", 
				Value: vmReqInfo.VpcIID.NameId,
			},
			{
				Key: "subnetId", 
				Value: vmReqInfo.SubnetIID.NameId,
			},
			{
				Key: "SecurityGroups", 
				Value: sgIDsString,
			},
			{
				Key: "vmPublicIpId", 
				Value: publicIpId,
			},
			{
				Key: "vmSpecId", 
				Value: instanceSpecId,
			},
		}

		createTagsReq := ktsdk.CreateTags {
			ResourceIds: []string{newVM.Deployvirtualmachineresponse.ID, },
			ResourceType: "userVm",
			Tags: vmTags,
		}
		createTagsResult, err := vmHandler.Client.CreateTags(&createTagsReq)
		if err != nil {
			cblogger.Errorf("Failed to Create the Tags : [%v]", err)
			return irs.VMInfo{}, err
		}
			
		cblogger.Info("### Waiting for Tags to be Created(300sec)!!\n")
		waitJobErr := vmHandler.Client.WaitForAsyncJob(createTagsResult.Createtagsresponse.JobId, 300000000000)
		if waitJobErr != nil {
			cblogger.Errorf("Failed to Wait the Job : [%v]", waitJobErr)
			return irs.VMInfo{}, waitJobErr
		}

		_, jobErr := vmHandler.Client.QueryAsyncJobResult(createTagsResult.Createtagsresponse.JobId)
		if err != nil {
			cblogger.Errorf("Failed to Find the Job: [%v]", jobErr)
			return irs.VMInfo{}, jobErr
		}

		// $$$ Time sleep after Public IP setting process!! $$$
		cblogger.Info("\n\n### Waiting for Setting New PublicIP and Firewall Rules on KT Cloud!!")
		time.Sleep(time.Second * 10)

		newVMInfo, error := vmHandler.GetVM(newVMIID)
		if error != nil {
			cblogger.Errorf("Failed to Get Created VM Info : [%v]", err)
			return irs.VMInfo{}, err
		}
		cblogger.Info("### VM Creation Processes have been Finished !!")
		return newVMInfo, nil
	}
	return irs.VMInfo{}, err
}

func (vmHandler *KtCloudVMHandler) MappingServerInfo(KtCloudInstance ktsdk.Virtualmachine) (irs.VMInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called MappingServerInfo()!!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, KtCloudInstance.Name, "MappingServerInfo()")
	// cblogger.Info("# KtCloudInstance : ")
	// spew.Dump(KtCloudInstance)

	// To get list of the PortForwarding Rule info
	callLogStart := call.Start()
	pfRulesListReqInfo := ktsdk.ListPortForwardingRulesReqInfo{}
	pfResponse, err := vmHandler.Client.ListPortForwardingRules(pfRulesListReqInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Port Forwarding Rules List : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)
	// spew.Dump(pfResponse.Listportforwardingrulesresponse.PortForwardingRule)

	// To get the public IP info according to the VM_ID from the PortForwarding Rule list
	var publicIp string
	for _, pfRule := range pfResponse.Listportforwardingrulesresponse.PortForwardingRule {
		if pfRule.VirtualmachineId == KtCloudInstance.ID {
		publicIp = pfRule.IpAddress
		break
		}
	}

	vpcId, err := vmHandler.GetVPCFromTags(KtCloudInstance.ID)
	if err != nil {
		cblogger.Errorf("Failed to Get VPC ID from tags : [%v]", err)
		return irs.VMInfo{}, err
	}
	time.Sleep(time.Second * 1) 
	// To Prevent the Error : "Unable to execute API command listTags due to ratelimit timeout"

	subnetId, err := vmHandler.GetSubnetFromTags(KtCloudInstance.ID)
	if err != nil {
		cblogger.Errorf("Failed to Get Subnet ID from tags : [%v]", err)
		return irs.VMInfo{}, err
	}
	time.Sleep(time.Second * 1)

	vmSpecId, err := vmHandler.GetVMSpecFromTags(KtCloudInstance.ID)
	if err != nil {
		cblogger.Errorf("Failed to Get vmSpec ID from tags : [%v]", err)
		return irs.VMInfo{}, err
	}
	time.Sleep(time.Second * 1)

	sgList, err := vmHandler.GetSGListFromTags(KtCloudInstance.ID)
	if err != nil {
		cblogger.Errorf("Failed to Get the List of S/G from tags : [%v]", err)
		return irs.VMInfo{}, err
	}

	vmStatus, errStatus := ConvertVMStatusToString(KtCloudInstance.State)
	if errStatus != nil {
		cblogger.Errorf("Failed to Convert the VM Status : [%v]", errStatus)
		return irs.VMInfo{}, errStatus
	}

	convertedTime, err := convertTimeFormat(KtCloudInstance.Created)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert the Time Format!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMInfo{}, newErr
	}
	
	var diskSize string
	if !strings.EqualFold(vmSpecId, "") {
		_, _, diskSize = GetKTVMSpecIdAndDiskSize(vmSpecId)
	}

	// To Get the VM resources Info.
	// PublicIpID : To use it when delete the PublicIP
	vmInfo := irs.VMInfo{
		IId: irs.IID{
			NameId:   KtCloudInstance.Name,
			SystemId: KtCloudInstance.ID,
		},

		StartTime: convertedTime,

		Region: irs.RegionInfo{
			Region: vmHandler.RegionInfo.Region,
			// Zone info is bellow.
		},

		ImageIId: irs.IID{
			NameId: KtCloudInstance.TemplateName, 
			SystemId: KtCloudInstance.TemplateId,
		},

		VMSpecName: vmSpecId, //Server Spec code

		VpcIID: irs.IID{
			NameId:   vpcId,
			SystemId: vpcId,
		},

		SubnetIID: irs.IID{
			NameId:   subnetId,
			SystemId: subnetId,
		},

		SecurityGroupIIds: sgList,

		//KT Cloud KeyPair에는 SystemID가 없으므로, 고유한 NameID 값을 SystemID에도 반영
		KeyPairIId: irs.IID{
			NameId: KtCloudInstance.KeyPair,
			SystemId: KtCloudInstance.KeyPair,
		},

		RootDiskSize: diskSize,

		VMUserId : vmUserName,
		//vmInfo.VMUserId = "root" //KT Cloud default user account
		VMUserPasswd: "N/A",
		//VMUserPasswd: KtCloudInstance.Password,

		PublicIP:   publicIp,
		PrivateIP:  KtCloudInstance.Nic[0].IpAddress,

		SSHAccessPoint: publicIp + ":22",
		VMBootDisk: "/dev/xvda",
		VMBlockDisk: "/dev/xvda",
		// VMBootDisk: ncloud.StringValue(blockStorageResult.BlockStorageInstanceList[0].DeviceName),
		// VMBlockDisk: ncloud.StringValue(blockStorageResult.BlockStorageInstanceList[0].DeviceName),

		KeyValueList: []irs.KeyValue{
			{Key: "CpuCount", Value: strconv.FormatFloat(float64(KtCloudInstance.CpuNumber), 'f', 0, 64)},
			{Key: "CpuSpeed", Value: strconv.FormatFloat(float64(KtCloudInstance.CpuSpeed), 'f', 0, 64)},
			{Key: "MemorySize(GB)", Value: strconv.FormatFloat(float64(KtCloudInstance.Memory)/(1024), 'f', 0, 64)},
			{Key: "KTCloudVMSpecInfo", Value: KtCloudInstance.ServiceOfferingName},
			{Key: "ZoneId", Value: KtCloudInstance.ZoneId},
			{Key: "VMStatus", Value: vmStatus},			
			{Key: "VMNetworkID", Value: KtCloudInstance.Nic[0].NetworkId},
			{Key: "Hypervisor", Value: KtCloudInstance.Hypervisor},			
			// {Key: "VM Secondary IP", Value: KtCloudInstance.Nic[0].SecondaryIp},
			// {Key: "PublicIpID", Value: publicIpId},
		},
	}

	// Set VM Zone Info
	if KtCloudInstance.ZoneName != "" {
		if strings.EqualFold(KtCloudInstance.ZoneName, "kr-0") {  // ???
			vmInfo.Region.Zone = "KOR-Seoul M"
		} else if strings.EqualFold(KtCloudInstance.ZoneName, "kr-md2-1") {
			vmInfo.Region.Zone = "KOR-Seoul M2"
		} else if strings.EqualFold(KtCloudInstance.ZoneName, "kr-1") {
			vmInfo.Region.Zone = "KOR-Central A"
		} else if strings.EqualFold(KtCloudInstance.ZoneName, "kr-2") {
			vmInfo.Region.Zone = "KOR-Central B"
		} else if strings.EqualFold(KtCloudInstance.ZoneName, "kr-3") {
			vmInfo.Region.Zone = "KOR-HA"
		} else {
		vmInfo.Region.Zone = KtCloudInstance.ZoneName 
		}
	}
	cblogger.Infof("KT Cloud Instance Uptime : [%s]", KtCloudInstance.Created)
	return vmInfo, nil
}

func (vmHandler *KtCloudVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called GetVM()!!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmHandler.RegionInfo.Zone, call.VM, "GetVM()", "GetVM()")

	cblogger.Infof("vmID : [%s]", vmIID.SystemId)
	cblogger.Infof("vmHandler.RegionInfo.Zone : [%s]", vmHandler.RegionInfo.Zone)

	vmListReqInfo := ktsdk.ListVMReqInfo{
		ZoneId: 	vmHandler.RegionInfo.Zone,
		VMId:       vmIID.SystemId,
	}
	start := call.Start()
	result, err := vmHandler.Client.ListVirtualMachines(vmListReqInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the List of VMs : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	if len(result.Listvirtualmachinesresponse.Virtualmachine) < 1 {
		return irs.VMInfo{}, errors.New("Failed to Find the VM with the SystemId : " + vmIID.SystemId)
	}
	// spew.Dump(result)
	
	vmInfo, err := vmHandler.MappingServerInfo(result.Listvirtualmachinesresponse.Virtualmachine[0])
	if err != nil {
		cblogger.Errorf("Failed to Map the VM info: [%v]", err)
		return irs.VMInfo{}, err
	}
	return vmInfo, nil
}

func (vmHandler *KtCloudVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("KT Cloud cloud driver: called SuspendVM()!!")
	cblogger.Info("Start Get VM Status...")
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		cblogger.Errorf("[%s] Failed to Get the VM Status of VM : ", vmIID)
		cblogger.Error(err)
	} else {
		cblogger.Infof("[%s] Succeeded in Getting the VM Status : [%s]", vmIID, vmStatus)
	}

	var resultStatus string
	if strings.EqualFold(string(vmStatus), "Suspending") {
		resultStatus = "The VM is already in the process of 'Suspending'."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Suspended") {
		resultStatus = "The VM is already 'Suspended'."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Rebooting") {
		resultStatus = "The VM is in the process of 'Rebooting'."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Terminating") {
		resultStatus = "The VM is in the process of 'Terminating'."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Booting") {
		resultStatus = "The VM is in the process of 'Booting'."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else {
		result, err := vmHandler.Client.StopVirtualMachine(vmIID.SystemId)
		if err != nil {
			cblogger.Errorf("Failed to Stop the VM : [%v]", err)	
			return "", err
		}
		
		jobResult, err := vmHandler.Client.QueryAsyncJobResult(result.Stopvirtualmachineresponse.JobId)
		if err != nil {
			cblogger.Errorf("Failed to Get Job Result: [%v]", err)	
			return "", err
		}		
		spew.Dump(jobResult)

		//===================================
		// 15-second wait for Suspending
		//===================================
		curRetryCnt := 0
		maxRetryCnt := 16
		for {
			curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
			if errStatus != nil {
				cblogger.Error(errStatus.Error())
			}

			cblogger.Info("===> VM Status : ", curStatus)
			if curStatus != irs.VMStatus("Suspended") {
				curRetryCnt++
				cblogger.Infof("The VM status is not 'Suspended' yet, so wait more!!")
				time.Sleep(time.Second * 2)
				if curRetryCnt > maxRetryCnt {
					cblogger.Error("Despite waiting for a long time(%d sec), the VM is not 'Suspended' yet, so it is forcibly terminated.", maxRetryCnt)
				}
			} else {
				break
			}
		}

	}
	return irs.VMStatus("Suspended"), nil
	// return irs.VMStatus("NotExist"), nil
}

func (vmHandler *KtCloudVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("KT Cloud cloud driver: called ResumeVM()!")
	cblogger.Info("Start Get VM Status...")
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		cblogger.Errorf("[%s] Failed to Get the VM Status", vmIID)
		cblogger.Error(err)
	} else {
		cblogger.Infof("[%s] Succeeded in Getting the VM Status : [%s]", vmIID, vmStatus)
	}

	var resultStatus string
	if strings.EqualFold(string(vmStatus), "Running") {
		resultStatus = "The VM is 'Running'. Cannot be Resumed!!"
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Suspending") {
		resultStatus = "The VM is in the process of 'Suspending'. Cannot be Resumed"
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Rebooting") {
		resultStatus = "The VM is in the process of 'Rebooting'. Cannot be Resumed"
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Terminating") {
		resultStatus = "The VM is already in the process of 'Terminating'. Cannot be Resumed"
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Booting") {
		resultStatus = "The VM is in the process of 'Booting'. Cannot be Resumed"
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Creating") {
		resultStatus = "The VM is in the process of 'Creating'. Cannot be Resumed"
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else {
		result, err := vmHandler.Client.StartVirtualMachine(vmIID.SystemId)
		if err != nil {
			cblogger.Errorf("Failed to Resume the VM : [%v]", err)	
			return "", err
		}
		
		jobResult, err := vmHandler.Client.QueryAsyncJobResult(result.Startvirtualmachineresponse.JobId)
		if err != nil {
			cblogger.Errorf("Failed to Get the Job Result : [%v]", err)	
			return "", err
		}		
		spew.Dump(jobResult)

		//===================================
		// 60-second wait for Suspending
		//===================================
		curRetryCnt := 0
		maxRetryCnt := 30
		for {
			curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
			if errStatus != nil {
				cblogger.Error(errStatus.Error())
			}

			cblogger.Info("===> VM Status : ", curStatus)
			if curStatus != irs.VMStatus("Running") {
				curRetryCnt++
				cblogger.Infof("The VM is not 'Resumed' yet, so wait more!!")
				time.Sleep(time.Second * 2)
				if curRetryCnt > maxRetryCnt {
					cblogger.Error("Despite waiting for a long time(%d sec), the VM is not 'Resumed' yet, so it is forcibly terminated.", maxRetryCnt)
				}
			} else {
				break
			}
		}

	}
	return irs.VMStatus("Resumed"), nil
}

func (vmHandler *KtCloudVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("KT Cloud cloud driver: called RebootVM()!")
	cblogger.Info("Start Get VM Status...")
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		cblogger.Errorf("[%s] Failed to Get the VM Status of VM : ", vmIID)
		cblogger.Error(err)
	} else {
		cblogger.Infof("[%s] Succeeded in Getting the VM Status : [%s]", vmIID, vmStatus)
	}

	var resultStatus string
	if strings.EqualFold(string(vmStatus), "Suspending") {
		resultStatus = "The VM is in the process of 'Suspending'."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Suspended") {
		resultStatus = "The VM is 'Suspended'."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Rebooting") {
		resultStatus = "The VM is already in the process of 'Rebooting'."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Terminating") {
		resultStatus = "The VM is in the process of 'Terminating'."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else if strings.EqualFold(string(vmStatus), "Booting") {
		resultStatus = "The VM is in the process of 'Booting'."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err

	} else {
		result, err := vmHandler.Client.RebootVirtualMachine(vmIID.SystemId)
		if err != nil {
			cblogger.Errorf("Failed to Reboot the VM : [%v]", err)	
			return "", err
		}
		
		jobResult, err := vmHandler.Client.QueryAsyncJobResult(result.Rebootvirtualmachineresponse.JobId)
		if err != nil {
			cblogger.Errorf("Failed to Get the Job Result : [%v]", err)	
			return "", err
		}		
		spew.Dump(jobResult)

		// ===================================
		// 15-second wait for Rebooting
		// ===================================
		curRetryCnt := 0
		maxRetryCnt := 16
		for {
			curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
			if errStatus != nil {
				cblogger.Error(errStatus.Error())
			}

			cblogger.Info("===> VM Status : ", curStatus)
			if curStatus != irs.VMStatus("Running") {
				curRetryCnt++
				cblogger.Infof("The VM is not 'Running' yet, so wait more!!")
				time.Sleep(time.Second * 2)
				if curRetryCnt > maxRetryCnt {
					cblogger.Error("Despite waiting for a long time(%d sec), the VM is not 'Running' yet, so it is forcibly terminated.", maxRetryCnt)
				}
			} else {
				break
			}
		}
	}
	return irs.VMStatus("Running"), nil
}

func (vmHandler *KtCloudVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("KT Cloud cloud driver: called TerminateVM()!")
	cblogger.Info("Start Getting the VM Status")
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		cblogger.Errorf("Failed to Get the VM Status : [%s]", vmIID)
		cblogger.Error(err)
	} else {
		cblogger.Infof("Succeed in Getting the VM Status : [%s][%s]", vmIID, vmStatus)
	}

	vmInfo, error := vmHandler.GetVM(vmIID)
	if error != nil {
		cblogger.Error(error.Error())
		return irs.VMStatus("Failed to get the VM info"), err
	}

	switch string(vmStatus) {
	case "Suspended":
		cblogger.Infof("VM Status : 'Suspended'. so it Can be Terminated!!")
		cblogger.Info("==> The PublicIP of the VM : " + vmInfo.PublicIP)
	
		// To Get the PulbicIP info
		ipListReqInfo := ktsdk.ListPublicIpReqInfo {
			IpAddress: vmInfo.PublicIP,
		}
		ipListResponse, err := vmHandler.Client.ListPublicIpAddresses(ipListReqInfo)
		if err != nil {
			cblogger.Error(error.Error())
			return irs.VMStatus("Failed to Get the Public IP List"), err
		} else {
			cblogger.Info("Succeeded in Getting the Public IP List!!")
		}
		publicIpId := ipListResponse.Listpublicipaddressesresponse.PublicIpAddress[0].ID

		vmStatus, error := vmHandler.DeleteFirewall(publicIpId)
		if error != nil {
			cblogger.Error(error.Error())
			return irs.VMStatus("Failed to Delete the Firewall rules"), err
		} else {
			cblogger.Info("Succeeded in Deleting the Firewall rules!!")
		}
		cblogger.Infof("VM Status : " + string(vmStatus))

		vStatus, error := vmHandler.DeletePortForwarding(publicIpId)
		if error != nil {
			cblogger.Error(error.Error())
			return irs.VMStatus("Failed to Delete the PortForwarding rule"), err
		} else {
			cblogger.Info("Succeeded in Deleting the PortForwarding rule!!")
		}
		cblogger.Infof("VM Status : " + string(vStatus))

		status, error := vmHandler.DisassociatePublicIp(publicIpId)
		if error != nil {
			cblogger.Error(error.Error())	
			return irs.VMStatus("Failed to Disassociate the Public IP"), err
		} else {
			cblogger.Info("Succeeded in Disassociating the Public IP!!")
		}
		cblogger.Infof("VM Status : " + string(status))

		destroyVMResponse, err := vmHandler.Client.DestroyVirtualMachine(vmIID.SystemId)
		if err != nil {
			cblogger.Errorf("Failed to terminate the VM : [%v]", err)
			return "Error", err
		}
		spew.Dump(destroyVMResponse)
		
		return irs.VMStatus("Terminating"), nil

	case "Running":
		cblogger.Infof("VM Status : 'Running'. so it Can be Terminated AFTER SUSPENSION !!")
		cblogger.Info("Start Suspending the VM !!")
		result, err := vmHandler.SuspendVM(vmIID)
		if err != nil {
			cblogger.Errorf("[%s] Failed to Suspend the VM - [%s]", vmIID, result)
			cblogger.Error(err)
		} else {
			cblogger.Infof("[%s] Succeeded in Suspending the VM - [%s]", vmIID, result)
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

			cblogger.Info("===> VM Status : ", curStatus)
			if curStatus != irs.VMStatus("Suspended") {
				curRetryCnt++
				cblogger.Infof("The VM is not 'Suspended' yet, so wait more before inquiring Termination.")
				time.Sleep(time.Second * 1)
				if curRetryCnt > maxRetryCnt {
					cblogger.Error("Despite waiting for a long time(%d sec), the VM is not 'Suspended' yet, so it is forcibly terminated.", maxRetryCnt)
				}
			} else {
				break
			}
		}

		cblogger.Info("# SuspendVM() Finished")

		// vmStatus, error := vmHandler.DeleteFirewall(vmInfo)
		// if error != nil {
		// 	cblogger.Error(error.Error())
	
		// 	return irs.VMStatus("Failed to Delete the Firewall rules"), err
		// }

		// cblogger.Infof("VM Status : " + string(vmStatus))

		cblogger.Info("Start Terminating the VM !!")
		vmStatus, err := vmHandler.TerminateVM(vmIID)
		if err != nil {
			cblogger.Errorf("[%s] Failed to Terminate the VM.", vmIID)
			cblogger.Error(err)
		} else {
			cblogger.Infof("[%s] Succeed in Terminating the VM : [%s]", vmIID, vmStatus)
		}

		return irs.VMStatus("Terminateding"), nil

	default:
		resultStatus := "The VM status is not 'Running' or 'Suspended'. so it Can NOT be Terminated!! Run or Suspend the VM before terminating."
		cblogger.Error(resultStatus)
		return irs.VMStatus("Failed. " + resultStatus), err
	}
}

/*
# KT Cloud serverInstanceStatusName ??
Stopped
Starting
Running
Stopping

rebooting
hard rebooting
shutting down //Caution!! : During Suspending
hard shutting down
terminating

*/

func (vmHandler *KtCloudVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("KT Cloud cloud driver: called GetVMStatus()!")

	vmListReqInfo := ktsdk.ListVMReqInfo{
		ZoneId: 	vmHandler.RegionInfo.Zone,
		VMId:       vmIID.SystemId,
	}
	result, err := vmHandler.Client.ListVirtualMachines(vmListReqInfo)
	if err != nil {
		cblogger.Errorf("Failed to Get the list of VMs : [%v]", err)
		return irs.VMStatus("Failed. "), err
	}

	if len(result.Listvirtualmachinesresponse.Virtualmachine) < 1 {
		return irs.VMStatus("Failed. "), errors.New("Failed to Find the VM SystemId '" + vmIID.SystemId + "'!!")
	}
	// spew.Dump(result)
	ktVMStatus := result.Listvirtualmachinesresponse.Virtualmachine[0].State
	cblogger.Info("Succeeded in Getting the VM Status info!!")

	vmStatus, statusErr := ConvertVMStatus(ktVMStatus)
	if statusErr != nil {
		cblogger.Errorf("Failed to Convert the VM Status : [%v]", statusErr)
		return irs.VMStatus("Failed. "), statusErr
	}
	cblogger.Info("# Converted VM Status : " + vmStatus)
	return vmStatus, statusErr
}

func ConvertVMStatus(vmStatus string) (irs.VMStatus, error) {
	var resultStatus string
	cblogger.Infof("vmStatus : [%s]", vmStatus)
	if strings.EqualFold(vmStatus, "creating") {
		resultStatus = "Creating"
	} else if strings.EqualFold(vmStatus, "booting") {
		resultStatus = "Booting"
	} else if strings.EqualFold(vmStatus, "Starting") {
		resultStatus = "Booting"
	} else if strings.EqualFold(vmStatus, "Running") {
		resultStatus = "Running"
	} else if strings.EqualFold(vmStatus, "Stopping") {
		resultStatus = "Suspending"
	} else if strings.EqualFold(vmStatus, "Stopped") {
		resultStatus = "Suspended"
	} else if strings.EqualFold(vmStatus, "rebooting") {
		resultStatus = "Rebooting"
	} else if strings.EqualFold(vmStatus, "terminating") {
		resultStatus = "Terminating"
	} else if strings.EqualFold(vmStatus, "Error") {
		resultStatus = "Error"
	} else {
		cblogger.Errorf("Failed to Find mapping information matching with the vmStatus [%s].", string(vmStatus))
		return irs.VMStatus("Failed. "), errors.New(vmStatus + "Failed to Find mapping information matching with the vmStatus.")
	}

	cblogger.Infof("Succeeded in Converting the VM Status : [%s] ==> [%s]", vmStatus, resultStatus)
	return irs.VMStatus(resultStatus), nil
}

func ConvertVMStatusToString(vmStatus string) (string, error) {
	var resultStatus string
	cblogger.Infof("vmStatus : [%s]", vmStatus)
	if strings.EqualFold(vmStatus, "creating") {
		resultStatus = "Creating"
	} else if strings.EqualFold(vmStatus, "booting") {
		resultStatus = "Booting"
	} else if strings.EqualFold(vmStatus, "Starting") {
		resultStatus = "Booting"
	} else if strings.EqualFold(vmStatus, "Running") {
		resultStatus = "Running"
	} else if strings.EqualFold(vmStatus, "Stopping") {
		resultStatus = "Suspending"
	} else if strings.EqualFold(vmStatus, "Stopped") {
		resultStatus = "Suspended"
	} else if strings.EqualFold(vmStatus, "rebooting") {
		resultStatus = "Rebooting"
	} else if strings.EqualFold(vmStatus, "terminating") {
		resultStatus = "Terminating"
	} else if strings.EqualFold(vmStatus, "Error") {
		resultStatus = "Error"
	} else {
		cblogger.Errorf("Failed to Find mapping information matching with the vmStatus [%s].", string(vmStatus))
		return "Failed. ", errors.New(vmStatus + "Failed to Find mapping information matching with the vmStatus.")
	}

	cblogger.Infof("\nSucceeded in Convertting the VM Status : [%s] ==> [%s]", vmStatus, resultStatus)
	return resultStatus, nil
}

func (vmHandler *KtCloudVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called ListVMStatus()!")

	ktVMList, err := vmHandler.ListKTCloudVM()
	if err != nil {
		cblogger.Errorf("Failed to Get the List of VMs : [%v]", err)
		return []*irs.VMStatusInfo{}, err
	}
	if len(ktVMList) < 1 {
		cblogger.Info("### There is No VM!!")
		return []*irs.VMStatusInfo{}, nil
		// return []*irs.VMStatusInfo{}, errors.New("Failed to Find VM list!!")
	}

	var vmStatusList []*irs.VMStatusInfo
	for _, vm := range ktVMList {
		vmStatus, err := ConvertVMStatus(vm.State)
		if err != nil {
			cblogger.Errorf("Failed to Convert the VM Status : [%v]", err)
			return []*irs.VMStatusInfo{}, nil
		}

		vmStatusInfo := irs.VMStatusInfo{
			IId:      irs.IID{
				NameId: 	vm.Name, 
				SystemId: 	vm.ID,
			},
			VmStatus: vmStatus,
		}		
		cblogger.Info(vmStatusInfo.IId.SystemId, " VM Status : ", vmStatusInfo.VmStatus)
		vmStatusList = append(vmStatusList, &vmStatusInfo)
	}
	return vmStatusList, err
}

func (vmHandler *KtCloudVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called ListVM()!")

	ktVMList, err := vmHandler.ListKTCloudVM()
	if err != nil {
		cblogger.Errorf("Failed to Get the List of VMs : [%v]", err)
		return []*irs.VMInfo{}, err
	}
	if len(ktVMList) < 1 {
		cblogger.Info("### There is No VM!!")
		return []*irs.VMInfo{}, nil
		// return []*irs.VMStatusInfo{}, errors.New("Failed to Find VM list!!")
	}

	var vmInfoList []*irs.VMInfo	
	for _, ktVM := range ktVMList {
		vmInfo, err:= vmHandler.MappingServerInfo(ktVM)
		if err != nil {
			cblogger.Errorf("Failed to Map the VM info : [%v]", err)
			return []*irs.VMInfo{}, err
		}		
		vmInfoList = append(vmInfoList, &vmInfo)
	}
	return vmInfoList, nil
}

func (vmHandler *KtCloudVMHandler) ListKTCloudVM() ([]ktsdk.Virtualmachine, error) {
	cblogger.Info("KT Cloud cloud driver: called ListKTCloudVM()!")
	cblogger.Infof("vmHandler.RegionInfo.Zone : [%s]", vmHandler.RegionInfo.Region)

	vmListReqInfo := ktsdk.ListVMReqInfo{
		ZoneId: 	vmHandler.RegionInfo.Zone,
	}
	result, err := vmHandler.Client.ListVirtualMachines(vmListReqInfo)
	if err != nil {
		cblogger.Errorf("Failed to Get the VM List from KT Cloud : [%v]", err)
		return []ktsdk.Virtualmachine{}, err
	}
	if len(result.Listvirtualmachinesresponse.Virtualmachine) < 1 {
		cblogger.Info("### There is No VM!!")
		return []ktsdk.Virtualmachine{}, nil
		// return []*irs.VMInfo{}, errors.New("Failed to Find the VM list!!")
	}
	// spew.Dump(result)
	return result.Listvirtualmachinesresponse.Virtualmachine, nil
}


func (vmHandler *KtCloudVMHandler) GetKTCloudVM(vmId string) (ktsdk.Virtualmachine, error) {
	cblogger.Info("KT Cloud cloud driver: called GetKTCloudVM()!")

	if strings.EqualFold(vmHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Zone Info!!")
		cblogger.Error(newErr.Error())
		return ktsdk.Virtualmachine{}, newErr
	}

	if strings.EqualFold(vmId, "") {
		newErr := fmt.Errorf("Invalid VM ID!!")
		cblogger.Error(newErr.Error())
		return ktsdk.Virtualmachine{}, newErr
	}

	vmListReqInfo := ktsdk.ListVMReqInfo{
		ZoneId: 	vmHandler.RegionInfo.Zone,
		VMId:       vmId,
	}
	result, err := vmHandler.Client.ListVirtualMachines(vmListReqInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the List of VMs : [%v]", err)
		cblogger.Error(newErr.Error())
		return ktsdk.Virtualmachine{}, newErr
	}

	if len(result.Listvirtualmachinesresponse.Virtualmachine) < 1 {
		return ktsdk.Virtualmachine{}, errors.New("Failed to Find the VM with the SystemId : " + vmId)
	}
	// spew.Dump(result)
	return result.Listvirtualmachinesresponse.Virtualmachine[0], nil
}


func (vmHandler *KtCloudVMHandler) GetVmIdWithName(vmNameId string) (string, error) {
	cblogger.Info("KT Cloud cloud driver: called GetVmIdWithName()!")

	if strings.EqualFold(vmNameId, "") {
		newErr := fmt.Errorf("Invalid VM NameId!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	// Get KT Cloud VM list
	ktVMList, err := vmHandler.ListKTCloudVM()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VM List : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	if len(ktVMList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any VM form KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	var vmId string
	for _, vm := range ktVMList {
		if strings.EqualFold(vm.Name, vmNameId) {
			vmId = vm.ID
			break
		}
	}

	if vmId == "" {
		err := errors.New(fmt.Sprintf("Failed to Find the VM ID with the VM Name %s", vmNameId))
		return "", err
	} else {
	return vmId, nil
	}
}

func (vmHandler *KtCloudVMHandler) GetVmNameWithId(vmId string) (string, error) {
	cblogger.Info("KT Cloud cloud driver: called GetVmNameWithId()!")

	if strings.EqualFold(vmId, "") {
		newErr := fmt.Errorf("Invalid VM ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	ktVM, err := vmHandler.GetKTCloudVM(vmId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM Info from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	vmName := ktVM.Name
	if vmName == "" {
		err := errors.New(fmt.Sprintf("Failed to Find the VM Name with the VM ID %s", vmId))
		return "", err
	} else {
	return vmName, nil
	}
}

// Waiting for up to 300 seconds until VM info. can be get
func (vmHandler *KtCloudVMHandler) WaitToGetInfo(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("======> As VM info. cannot be retrieved immediately after VM creation, it waits until Running.")

	curRetryCnt := 0
	maxRetryCnt := 500

	var returnStatus irs.VMStatus
	for {
		curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
		if errStatus != nil {
			cblogger.Errorf("Failed to Get the VM Status of : [%s]", vmIID)
			cblogger.Error(errStatus.Error())
		} else {
			cblogger.Infof("Succeeded in Getting the VM Status of [%s] : [%s]", vmIID.SystemId, curStatus)
		}

		cblogger.Info("===> VM Status : ", curStatus)

		switch string(curStatus) {
		case "Suspended", "Creating", "Booting" : // KT Cloud는 VM이 Suspended 상태로 시작함.
			curRetryCnt++
			cblogger.Infof("The VM status is still 'Creating', so wait more before inquiring the VM info.")
			time.Sleep(time.Second * 3)
			if curRetryCnt > maxRetryCnt {
				cblogger.Errorf("Despite waiting for a long time(%d sec), the VM status is '%S', so it is forcibly finishied.", maxRetryCnt, curStatus)
				return irs.VMStatus("Failed"), errors.New("Despite waiting for a long time, the VM status is 'Creating', so it is forcibly finishied.")
			}

		default:
			cblogger.Infof("===> The VM Creation is finished, stopping the waiting.")
			time.Sleep(time.Second * 4) // Additional time sleep!!
			return irs.VMStatus(curStatus), nil
			//break
		}

		returnStatus = curStatus
	}
	return irs.VMStatus(returnStatus), nil
}

// Whenever a VM is terminated, Delete the Firewall rules that the PublicIP has first.
func (vmHandler *KtCloudVMHandler) DeleteFirewall(publicIpId string) (irs.VMStatus, error) {
	cblogger.Info("KT Cloud cloud driver: called DeleteFirewall()!")
	firewallListReqInfo := ktsdk.ListFirewallRulesReqInfo {
		IpAddressId:	publicIpId,
	}
	firewallRulesResult, err := vmHandler.Client.ListFirewallRules(firewallListReqInfo)
	if err != nil {
		cblogger.Errorf("Failed to Get List of Firewall Rules : [%v]", err)
		return "", err
	} else {
		cblogger.Info("# Succeeded in Getting List of Firewall Rules!!")
	}
	// spew.Dump(firewallRulesResult.Listfirewallrulesresponse.FirewallRule)

	for _, firewallRule := range firewallRulesResult.Listfirewallrulesresponse.FirewallRule {
		// Delete any firewall rule (without leaving port number 22)
		deleteRulesResult, err := vmHandler.Client.DeleteFirewallRule(firewallRule.ID)
		if err != nil {
			cblogger.Errorf("Failed to Delete the firewall Rule : [%v]", err)	
			return "", err
		} else {
			cblogger.Info("### Waiting for Firewall Rule to be Deleted(300sec)!!\n")
			waitJobErr := vmHandler.Client.WaitForAsyncJob(deleteRulesResult.Deletefirewallruleresponse.JobId, 300000000000)
			if waitJobErr != nil {
				cblogger.Errorf("Failed to Wait the Job : [%v]", waitJobErr)	
				return irs.VMStatus("Terminating"), waitJobErr
			}
			
			cblogger.Info("# Succeeded in Deleting the firewall Rule : " + firewallRule.IpAddress + ", " + firewallRule.Protocol, firewallRule.StartPort, firewallRule.EndPort)
		}
		// spew.Dump(deleteRulesResult)
	}	

	return irs.VMStatus("Terminating"), nil
}

// Whenever a VM is terminated, Delete the PortForwarding rule that the PublicIP has.
func (vmHandler *KtCloudVMHandler) DeletePortForwarding(publicIpId string) (irs.VMStatus, error) {
	cblogger.Info("KT Cloud cloud driver: called DeletePortForwarding()!")
	// To Get the PortForwarding Rule info
	pfRulesListReqInfo := ktsdk.ListPortForwardingRulesReqInfo {
		IpAddressId:	publicIpId,
	}
	pfRulesResult, err := vmHandler.Client.ListPortForwardingRules(pfRulesListReqInfo)
	if err != nil {
		cblogger.Errorf("Failed to Get list of PortForwarding Rule : [%v]", err)
		return "", err
	} else {
		cblogger.Info("# Succeeded in Getting list of PortForwarding Rule!!")
	}

	if len(pfRulesResult.Listportforwardingrulesresponse.PortForwardingRule) > 0 {
		pfRule := pfRulesResult.Listportforwardingrulesresponse.PortForwardingRule[0]
		// spew.Dump(pfRule)

		pfRuleId := pfRule.ID
		pfRuleIpAddress := pfRule.IpAddress
		pfRuleProtocol := pfRule.Protocol
		pfRulePublicPort := pfRule.PublicPort
		pfRulePublicEndPort := pfRule.PublicEndPort
	
		deleteRuleResult, err := vmHandler.Client.DeletePortForwardingRule(pfRuleId)
		if err != nil {
			cblogger.Errorf("Failed to Delete the PortForwarding Rule : [%v]", err)	
			return "", err
		} else {
			cblogger.Info("### Waiting for PortForwarding Rule to be Deleted(600sec)!!\n")
			waitJobErr := vmHandler.Client.WaitForAsyncJob(deleteRuleResult.Deleteportforwardingruleresponse.JobId, 600000000000)
			if waitJobErr != nil {
				cblogger.Errorf("Failed to Wait the Job : [%v]", waitJobErr)	
				return irs.VMStatus("Terminating"), waitJobErr
			}
			cblogger.Info("# Succeeded in Deleting the PortForwarding Rule : " + pfRuleIpAddress + ", " + pfRuleProtocol + ", " + pfRulePublicPort + ", " + pfRulePublicEndPort)
		}
		// spew.Dump(deleteRuleResult)

	} else {
		cblogger.Info("# PortForwarding Rule is not set yet!!")
		return irs.VMStatus("Terminating"), nil
	}

	return irs.VMStatus("Terminating"), nil
}

// Whenever a VM is terminated, Delete the PortForwarding rule that the PublicIP has.
func (vmHandler *KtCloudVMHandler) DisassociatePublicIp(publicIpId string) (irs.VMStatus, error) {
	cblogger.Info("KT Cloud cloud driver: called DisassociatePublicIp()!")
	disassociateIpResult, err := vmHandler.Client.DisassociateIpAddress(publicIpId)
	if err != nil {
		cblogger.Errorf("Failed to Disassociate the IP Address : [%v]", err)
		return "", err
	} else {
		cblogger.Info("### Waiting for Public IP to be Disassociated(300sec)!!\n")
		waitJobErr := vmHandler.Client.WaitForAsyncJob(disassociateIpResult.Disassociateipaddressresponse.JobId, 300000000000)
		if waitJobErr != nil {
			cblogger.Errorf("Failed to Wait the Job : [%v]", waitJobErr)	
			return irs.VMStatus("Terminating"), waitJobErr
		}
		cblogger.Info("# Succeeded in Disassociating the IP Address!! IP ID : " + publicIpId)
	}
	// spew.Dump(disassociateIpResult)
	return irs.VMStatus("Terminating"), nil
}

func (vmHandler *KtCloudVMHandler) GetVPCFromTags(instanceId string) (string, error) {
	listTagsReq := ktsdk.ListTags {
		Key: "vpcId",
		ResourceType: "userVm",
		ResourceIds: instanceId,
	}
	listTagsResult, tagListErr := vmHandler.Client.ListTags(&listTagsReq)
	if tagListErr != nil {
		cblogger.Errorf("Failed to Get the Tags List : [%v]", tagListErr)
		return "", tagListErr
	}

	var vpcId string
	if len(listTagsResult.Listtagsresponse.Tag) > 0 {
		vpcId = listTagsResult.Listtagsresponse.Tag[0].Value
	} else {
		cblogger.Info("VPCId Tag is not set yet!!")
	}
	return vpcId, nil
}

func (vmHandler *KtCloudVMHandler) GetSubnetFromTags(instanceId string) (string, error) {
	listTagsReq := ktsdk.ListTags {
		Key: "subnetId",
		ResourceType: "userVm",
		ResourceIds: instanceId,
	}
	listTagsResult, tagListErr := vmHandler.Client.ListTags(&listTagsReq)
	if tagListErr != nil {
		cblogger.Errorf("Failed to Get the Tags List : [%v]", tagListErr)
		return "", tagListErr
	}

	var subnetId string
	if len(listTagsResult.Listtagsresponse.Tag) > 0 {
		subnetId = listTagsResult.Listtagsresponse.Tag[0].Value
	} else {
		cblogger.Info("SubnetId Tag is not set yet!!")
	}
	return subnetId, nil
}

func (vmHandler *KtCloudVMHandler) GetVMSpecFromTags(instanceId string) (string, error) {
	listTagsReq := ktsdk.ListTags {
		Key: "vmSpecId",
		ResourceType: "userVm",
		ResourceIds: instanceId,
	}
	listTagsResult, tagListErr := vmHandler.Client.ListTags(&listTagsReq)
	if tagListErr != nil {
		cblogger.Errorf("Failed to Get the Tags List : [%v]", tagListErr)
		return "", tagListErr
	}
	var vmSpecId string
	if len(listTagsResult.Listtagsresponse.Tag) > 0 {
		vmSpecId = listTagsResult.Listtagsresponse.Tag[0].Value
	} else {
		cblogger.Info("vmSpecId Tag is not set yet!!")
	}
	return vmSpecId, nil
}

func (vmHandler *KtCloudVMHandler) GetSGListFromTags(instanceId string) ([]irs.IID, error) {
	listTagsReq := ktsdk.ListTags {
		Key: "SecurityGroups",
		ResourceType: "userVm",
		ResourceIds: instanceId,
	}
	listTagsResult, tagListErr := vmHandler.Client.ListTags(&listTagsReq)
	if tagListErr != nil {
		cblogger.Errorf("Failed to Get the Tags List : [%v] ", tagListErr)
		return []irs.IID {}, tagListErr
	}

	var securityGroupsString string
	if len(listTagsResult.Listtagsresponse.Tag) > 0 {
		securityGroupsString = listTagsResult.Listtagsresponse.Tag[0].Value
	} else {
		cblogger.Info("SecurityGroups Tag is not set yet!!")
	}

	// Splits a string into a slice
	sgSlice := strings.Split(securityGroupsString, ",")
	sgList := []irs.IID {}
	for _, sgID := range sgSlice {
		cblogger.Infof("S/G ID : [%s]", sgID)
		sg := irs.IID{
			NameId: sgID,
			SystemId: sgID,
		}
		sgList = append(sgList, sg)
	}
	return sgList, nil
}

func (vmHandler *KtCloudVMHandler) GetIPFromPortForwardingRules(instanceId string) (string, error) {
	// To get list of the PortForwarding Rule info
	pfRulesListReqInfo := ktsdk.ListPortForwardingRulesReqInfo {}
	pfResponse, err := vmHandler.Client.ListPortForwardingRules(pfRulesListReqInfo)
	if err != nil {
		cblogger.Errorf("Failed to Get Port Forwarding Rules List : [%v]", err)
		return "", err
	}
	//spew.Dump(pfResponse.Listportforwardingrulesresponse.PortForwardingRule)

	// To get the public IP info according to the VM_ID from the PortForwarding Rule list
	var publicIp string
	for _, pfRule := range pfResponse.Listportforwardingrulesresponse.PortForwardingRule {
		if pfRule.VirtualmachineId == instanceId {
		publicIp = pfRule.IpAddress
		}
	}
	if publicIp == "" {  // If there is NO publicIP, then Create PublicIP and PortForwarding Rule
		cblogger.Error("Failed to Find the IP info from the Port forwarding rules.")	
	}
	return publicIp, nil
}

func GetKTVMSpecIdAndDiskSize(instanceSpecId string) (string, string, string) {
	// # instanceSpecId Ex) d3530ad2-462b-43ad-97d5-e1087b952b7d!87c0a6f6-c684-4fbe-a393-d8412bcf788d_disk100GB
	// # !와 _로 구분했음.
	// Caution : 아래의 string split 기호 중 ! 대신 #을 사용하면 CB-Spider API를 통해 call할 시 전체의 string이 전달되지 않고 # 전까지만 전달됨. 
	instanceSpecString := strings.Split(instanceSpecId, "!")
	// Check 'instanceSpecString'
	// for _, str := range instanceSpecString {
	// 	cblogger.Infof("instanceSpecString : [%s]", str)
	// }

	ktVMSpecId := instanceSpecString[0]

    // instanceSpecString[1] : Ex) 87c0a6f6-c684-4fbe-a393-d8412bcf788d_disk100GB
	diskOfferingString := strings.Split(instanceSpecString[1], "_")

	ktDiskOfferingId := diskOfferingString[0]
	// ktDiskOfferingId : Ex) 87c0a6f6-c684-4fbe-a393-d8412bcf788d

	ktDiskOfferingSize := diskOfferingString[1]
	// ktDiskOfferingSize : Ex) disk100GB

	ktDiskSize := strings.Split(ktDiskOfferingSize, "disk")
	DiskSize := ktDiskSize[1]

	return ktVMSpecId, ktDiskOfferingId, DiskSize
}

func (vmHandler *KtCloudVMHandler) AssociateIpAddress() (string, error) {	
	cblogger.Info("KT Cloud cloud driver: called AssociateIpAddress()!")
	IPReqInfo := ktsdk.AssociatePublicIpReqInfo {
		ZoneId: 		vmHandler.RegionInfo.Zone,
		UsagePlanType: 	"hourly", 
	}
	createIpResponse, err := vmHandler.Client.AssociateIpAddress(IPReqInfo)
	if err != nil {
		cblogger.Errorf("Failed to Create new Public IP : [%v]", err)
		return "", err
	}

	cblogger.Info("### Waiting for IP Address to be Associated(300sec)!!\n")
	waitErr := vmHandler.Client.WaitForAsyncJob(createIpResponse.Associateipaddressresponse.JobId, 300000000000)
	if waitErr != nil {
		cblogger.Errorf("Failed to Wait the Job : [%v]", waitErr)
		return "", waitErr
	}

	var publicIp string
	publicIpId := createIpResponse.Associateipaddressresponse.ID // PublicIP ID
	if publicIpId == "" {
			cblogger.Error("Failed to Find Public IP info.\n")
	} else {
		// To Get the Public IP info which is created.
		IPListReqInfo := ktsdk.ListPublicIpReqInfo {
			ID: publicIpId, 
		}
		response, err := vmHandler.Client.ListPublicIpAddresses(IPListReqInfo)
		if err != nil {
			cblogger.Errorf("Failed to Get the List of Public IP : [%v]", err)
			return "", err
		}

		if len(response.Listpublicipaddressesresponse.PublicIpAddress) > 0 {
			publicIp = response.Listpublicipaddressesresponse.PublicIpAddress[0].IpAddress
			ipState := response.Listpublicipaddressesresponse.PublicIpAddress[0].State
			fmt.Printf("New Public IP : %s, IP State : %s\n", publicIp, ipState)
		} else {
			return "", errors.New("Failed to Find Public IP!!\n")
		}
	}
	return publicIp, nil
}
	
// ### To Apply 'PortForwarding Rules' and 'Firewall Rules' to the Public IP ID.
func (vmHandler *KtCloudVMHandler) CreatePortForwardingFirewallRules(sgSystemIDs []string, publicIpId string, vmID string) (bool, error) {
	cblogger.Info("KT Cloud cloud driver: called CreatePortForwardingFirewallRules()!")
	securityHandler := KtCloudSecurityHandler{
		CredentialInfo: vmHandler.CredentialInfo,
		RegionInfo:		vmHandler.RegionInfo,
		Client:         vmHandler.Client,
	}

	for _, sgSystemID := range sgSystemIDs {
		cblogger.Infof("S/G System ID : [%s]", sgSystemID)

		sgInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: sgSystemID})
		if err != nil {
			cblogger.Errorf("Failed to Find the SecurityGroup : %s", sgSystemID)
			return false, err
		}
		// cblogger.Info("\n*sgInfo : ")
		// spew.Dump(sgInfo)

		var resultProtocol string
		for _, sgRule := range *sgInfo.SecurityRules {
			if strings.EqualFold(sgRule.IPProtocol , "tcp") { // case insensitive comparing and returns true.
				resultProtocol = "TCP"
			} else if strings.EqualFold(sgRule.IPProtocol , "udp") {
				resultProtocol = "UDP"
			} else if strings.EqualFold(sgRule.IPProtocol , "icmp") {
				resultProtocol = "ICMP"
			} else {
				cblogger.Errorf("Failed to Find mapping Protocol matching with the given Protocol [%s].", sgRule.IPProtocol)
				return false, errors.New("Failed to Find mapping Protocol matching with the given Protocol." + sgRule.IPProtocol)
			}

			// When the request port number is '-1', all ports are opened.
			if (sgRule.FromPort == "-1") && (sgRule.ToPort == "-1") {
				sgRule.FromPort = "1"
				sgRule.ToPort = "65535"
			}

			// It's impossible to input any port number when the protocol is ICMP on KT Cloud firewall. 
			// Caution!!) KT Cloud does Not support 'ICMP' protocol for PortForwarding Rule.
			if resultProtocol == "ICMP" {
				sgRule.FromPort = ""
				sgRule.ToPort = ""
			}

			var sgCIDR string
			if sgRule.CIDR == "" {
				sgCIDR = "0.0.0.0/0"
			} else { 
				sgCIDR = sgRule.CIDR
			}

			// Caution!!) KT Cloud 'PortForwarding Rules' and 'Firewall Rules' Support Only "inbound".
			if strings.EqualFold(sgRule.Direction, "inbound") {
				if !(strings.EqualFold(resultProtocol, "ICMP")) {
				// if !(strings.EqualFold(resultProtocol, "TCP") || strings.EqualFold(sgRule.FromPort, "22")) {
					cblogger.Info("##### FromPort : " + sgRule.FromPort)
					pfRuleReqInfo := ktsdk.CreatePortForwardingRuleReqInfo {
						IpAddressId: 		publicIpId,
						
						PrivatePort:		sgRule.FromPort, // Port number of the server to set Port-ForWARDING
						PrivateEndPort:		sgRule.ToPort,

						PublicPort:			sgRule.FromPort, // Port number of Public IP to set Port-ForWARDING
						PublicEndPort:		sgRule.ToPort,

						Protocol: 			resultProtocol,
						VirtualmachineId:   vmID,
						OpenFirewall:		true,	// ### Caution!!) When setting up Port-Forwarding, whether it is automatically registered in the firewall. (Default : true)
					}			

					pfRuleResponse, err := vmHandler.Client.CreatePortForwardingRule(pfRuleReqInfo)
					if err != nil {
							cblogger.Errorf("Failed to Create the PortForwarding Rule : [%v]", err)
							return false, err
					}

					cblogger.Info("### Waiting for PortForwarding Rules and Firewall Rules to be Created(300sec) !!\n")
					waitErr := vmHandler.Client.WaitForAsyncJob(pfRuleResponse.Createportforwardingruleresponse.JobId, 300000000000)
					if waitErr != nil {
						cblogger.Errorf("Failed to Wait the Job : [%v]", waitErr)			
						return false, waitErr
					}	

					pfRulesReqInfo := ktsdk.ListPortForwardingRulesReqInfo {
						ID:			pfRuleResponse.Createportforwardingruleresponse.ID,
					}					
					pfRulesResult, err := vmHandler.Client.ListPortForwardingRules(pfRulesReqInfo)
					if err != nil {
						cblogger.Errorf("Failed to Get PortForwarding Rule List : [%v]", err)
						return false, err
					} else {
						cblogger.Info("# Succeeded in Getting PortForwarding Rule List!!")
					}
					cblogger.Info("### PortForwarding Rule List : ")
					spew.Dump(pfRulesResult.Listportforwardingrulesresponse.PortForwardingRule)
				// }
				}

				// ### KT Cloud 'Firewall Rules' Setting for 'ICMP' protocol
				if (strings.EqualFold(resultProtocol, "ICMP")) {
					newfirewallRuleReqInfo := ktsdk.CreateFirewallRuleReqInfo {
						IpAddressId: 		publicIpId,
						Protocol: 			resultProtocol,
						CidrList:      		sgCIDR,
						StartPort:			sgRule.FromPort,
						EndPort:     		sgRule.ToPort,
						Type:				"user", // KT Cloud Firewall setting type : 'user' or 'system' (Default : user)
					}			

					firewallRuleResponse, err := vmHandler.Client.CreateFirewallRule(newfirewallRuleReqInfo)
					if err != nil {
							cblogger.Errorf("Failed to Create the Firewall Rule : [%v]", err)
							return false, err
					}

					cblogger.Info("### Waiting for Firewall Rule to be Created(300sec) !!\n")
					waitErr := vmHandler.Client.WaitForAsyncJob(firewallRuleResponse.Createfirewallruleresponse.JobId, 300000000000)
					if waitErr != nil {
						cblogger.Errorf("Failed to Wait the Job : [%v]", waitErr)			
						return false, waitErr
					}	

					firewallListReqInfo := ktsdk.ListFirewallRulesReqInfo {
						ID:	firewallRuleResponse.Createfirewallruleresponse.ID,
					}					
					firewallRulesResult, err := vmHandler.Client.ListFirewallRules(firewallListReqInfo)
					if err != nil {
						cblogger.Errorf("Failed to Get List of Firewall Rules : [%v]", err)
						return false, err
					} else {
						cblogger.Info("# Succeeded in Getting List of Firewall Rules!!")
					}
					spew.Dump(firewallRulesResult.Listfirewallrulesresponse.FirewallRule)
				}
			}
		}
	}
	return true, nil
}
