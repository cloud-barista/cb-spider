// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by hyokyung.kim@innogrid.co.kr, 2019.08.

package resources

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/nic"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/server"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/dna/adaptiveip"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	VMDefaultUser     = "root"
	VMDefaultPassword = "qwe1212!Q"
	SSHDefaultUser    = "cb-user"
	SSHDefaultPort    = 22
	VM                = "VM"
	DefaultSGName     = "SSH"
)

type ClouditVMHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func (vmHandler *ClouditVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (startVM irs.VMInfo, createError error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VM, VM, "StartVM()")

	// 가상서버 이름 중복 체크
	vmId, _ := vmHandler.getVmIdByName(vmReqInfo.IId.NameId)
	if vmId != "" {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s already exist", vmReqInfo.IId.NameId))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	// 이미지 정보 조회 (Name)
	imageHandler := ClouditImageHandler{
		Client:         vmHandler.Client,
		CredentialInfo: vmHandler.CredentialInfo,
	}
	image, err := imageHandler.GetImage(vmReqInfo.ImageIID)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = Failed to get image, %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	//  네트워크 정보 조회 (Name)
	vpcHandler := ClouditVPCHandler{
		Client:         vmHandler.Client,
		CredentialInfo: vmHandler.CredentialInfo,
	}
    vpcInfo ,err := vpcHandler.GetVPC(vmReqInfo.VpcIID)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = Failed to get VPC, %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	subnet, err := vpcHandler.GetSubnet(vmReqInfo.SubnetIID)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = Failed to get subnet, %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	subnetCheck := false
	for _, sub := range vpcInfo.SubnetInfoList {
		if subnet.ID == sub.IId.SystemId {
			subnetCheck = true
			break
		}
	}
	if !subnetCheck {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = subnet: '%s' cannot be found in VPC: '%s'",vmReqInfo.SubnetIID.NameId, vpcInfo.IId.NameId))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	// 보안그룹 정보 조회 (Name)
	sgHandler := ClouditSecurityHandler{
		Client:         vmHandler.Client,
		CredentialInfo: vmHandler.CredentialInfo,
	}

	securityInfo, err := sgHandler.getSecurityByName(DefaultSGName)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	addUserSSHSG := []server.SecGroupInfo{{securityInfo.ID}}

	// Spec 정보 조회 (Name)
	vmSpecId, err := GetVMSpecByName(vmHandler.Client.AuthenticatedHeaders(), vmHandler.Client, vmReqInfo.VMSpecName)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()
	KeyPairDes := fmt.Sprintf("keypair:%s", vmReqInfo.KeyPairIID.NameId)
	reqInfo := server.VMReqInfo{
		TemplateId:   image.IId.SystemId,
		SpecId:       *vmSpecId,
		Name:         vmReqInfo.IId.NameId,
		HostName:     vmReqInfo.IId.NameId,
		RootPassword: VMDefaultPassword,
		SubnetAddr:   subnet.Addr,
		Secgroups:    addUserSSHSG,
		Description:  KeyPairDes,
	}

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
		JSONBody:    reqInfo,
	}
	// Check PublicIP
	_, err = vmHandler.creatablePublicIP()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	// VM 생성
	start := call.Start()
	creatingVm, err := server.Start(vmHandler.Client, &requestOpts)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)
    cleanVMIID := irs.IID{
		NameId: creatingVm.Name, SystemId: creatingVm.ID,
	}

	var createErr error
    defer func(){
    	if createError != nil{
			cleanerErr := vmHandler.vmCleaner(cleanVMIID)
			if cleanerErr != nil{
				createError = errors.New(fmt.Sprintf("%s and Failed to rollback err = %s", createError.Error(), cleanerErr.Error()))
			}
		}
	}()
	// VM 생성 완료까지 wait
	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		// Check VM Deploy Status
		vmInfo, err := server.Get(vmHandler.Client, creatingVm.ID, &requestOpts)
		if err != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}

		if vmInfo.PrivateIp != "" && getVmStatus(vmInfo.State) == irs.Running {
			ok, err := vmHandler.AssociatePublicIP(creatingVm.Name, vmInfo.PrivateIp)
			if !ok {
				createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = Failed AssociatePublicIP"))
				if err != nil {
					createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = Failed AssociatePublicIP err= %s", err.Error()))
				}
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			break
		}
		time.Sleep(1 * time.Second)
		curRetryCnt++
		if curRetryCnt > maxRetryCnt {
			createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = exceeded maximum retry count %d", maxRetryCnt))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
	}

	vm, err := server.Get(vmHandler.Client, creatingVm.ID, &requestOpts)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	vmInfo := vmHandler.mappingServerInfo(*vm)

	// SSH 접속 사용자 및 공개키 등록
	loginUserId := SSHDefaultUser
	createUserErr := errors.New(fmt.Sprintf("Failed adding cb-User to new VM"))

	// SSH 접속까지 시도
	curConnectionCnt := 0
	maxConnectionRetryCnt := 30
	for {
		cblogger.Info("Trying to connect via root user ...")
		_, err := RunCommand(vmInfo.PublicIP, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, "echo test")
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
		curConnectionCnt++
		if curConnectionCnt > maxConnectionRetryCnt {
			createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s, not Connected", createUserErr.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
	}

	// 사용자 등록 및 sudoer 권한 추가
	_, err = RunCommand(vmInfo.PublicIP, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, fmt.Sprintf("useradd -s /bin/bash %s -rm", loginUserId))
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s = %s", createUserErr.Error(), err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	_, err = RunCommand(vmInfo.PublicIP, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, fmt.Sprintf("echo \"%s ALL=(root) NOPASSWD:ALL\" >> /etc/sudoers", loginUserId))
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s = %s", createUserErr.Error(), err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	// 공개키 등록
	_, err = RunCommand(vmInfo.PublicIP, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, fmt.Sprintf("mkdir -p /home/%s/.ssh", loginUserId))
	publicKey, err := GetPublicKey(vmHandler.CredentialInfo, vmReqInfo.KeyPairIID.NameId)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s = %s", createUserErr.Error(), err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	_, err = RunCommand(vmInfo.PublicIP, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, fmt.Sprintf("echo \"%s\" > /home/%s/.ssh/authorized_keys", publicKey, loginUserId))
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s = %s", createUserErr.Error(), err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	// ssh 접속 방법 변경 (sshd_config 파일 변경)
	_, err = RunCommand(vmInfo.PublicIP, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, "sed -i 's/PasswordAuthentication yes/PasswordAuthentication no/g' /etc/ssh/sshd_config")
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s = %s", createUserErr.Error(), err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	_, err = RunCommand(vmInfo.PublicIP, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, "sed -i 's/#PubkeyAuthentication yes/PubkeyAuthentication yes/g' /etc/ssh/sshd_config")
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s = %s", createUserErr.Error(), err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	_, err = RunCommand(vmInfo.PublicIP, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, "systemctl restart sshd")
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s = %s", createUserErr.Error(), err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	secGroups := make([]string, len(vmReqInfo.SecurityGroupIIDs))
	if len(vmReqInfo.SecurityGroupIIDs) > 0 {
		for i, s := range vmReqInfo.SecurityGroupIIDs {
			secGroups[i] = s.SystemId
		}
	} else {
		secGroups = append(secGroups, "")
	}

	vnicList, err := ListVNic(authHeader, vmHandler.Client, vmInfo.IId.SystemId)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create VM. Not found default VNic err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	changesgrequestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	if vnicList != nil {
		defaultVnic := (*vnicList)[0]
		err := nic.ChangeSecurityGroup(vmHandler.Client, vmInfo.IId.SystemId, &changesgrequestOpts, defaultVnic.Mac, secGroups)
		if err != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = change Security Groups err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
	} else {
		createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = Not found default VNic"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	vm, err = server.Get(vmHandler.Client, creatingVm.ID, &requestOpts)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	vmInfo = vmHandler.mappingServerInfo(*vm)

	return vmInfo, nil
}

func (vmHandler *ClouditVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VM, vmIID.NameId, "SuspendVM()")

	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	sysId := vmIID.SystemId
	if sysId == "" {
		id, err := vmHandler.getVmIdByName(vmIID.NameId)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to SuspendVM. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return irs.Failed, getErr
		}
		sysId = id
	}
	vmStatus, _ := vmHandler.GetVMStatus(vmIID)
	if vmStatus != irs.Running {
		getErr := errors.New(fmt.Sprintf("Failed to SuspendVM. err = not Running"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	start := call.Start()
	err := server.Suspend(vmHandler.Client, sysId, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to SuspendVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		// Check VM Deploy Status
		vmStatus, _ := vmHandler.GetVMStatus(vmIID)
		if vmStatus == irs.Suspended {
			LoggingInfo(hiscallInfo, start)
			return vmStatus, nil
		}
		time.Sleep(1 * time.Second)
		curRetryCnt++
		if curRetryCnt > maxRetryCnt {
			createErr := errors.New(fmt.Sprintf("Failed to SuspendVM. err = exceeded maximum retry count %d", maxRetryCnt))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.Failed, createErr
		}
	}
}

func (vmHandler *ClouditVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VM, vmIID.NameId, "ResumeVM()")

	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	sysId := vmIID.SystemId
	if sysId == "" {
		id, err := vmHandler.getVmIdByName(vmIID.NameId)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to ResumeVM. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return irs.Failed, getErr
		}
		sysId = id
	}
	start := call.Start()
	vmStatus, _ := vmHandler.GetVMStatus(vmIID)
	if vmStatus != irs.Suspended {
		getErr := errors.New(fmt.Sprintf("Failed to ResumeVM. err = not Suspended"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	err := server.Resume(vmHandler.Client, sysId, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to ResumeVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		// Check VM Deploy Status
		vmStatus, _ := vmHandler.GetVMStatus(vmIID)
		if vmStatus == irs.Running {
			LoggingInfo(hiscallInfo, start)
			return vmStatus, nil
		}
		time.Sleep(1 * time.Second)
		curRetryCnt++
		if curRetryCnt > maxRetryCnt {
			createErr := errors.New(fmt.Sprintf("Failed to ResumeVM. err = exceeded maximum retry count %d", maxRetryCnt))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.Failed, createErr
		}
	}
}

func (vmHandler *ClouditVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VM, vmIID.NameId, "RebootVM()")

	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	sysId := vmIID.SystemId
	if sysId == "" {
		id, err := vmHandler.getVmIdByName(vmIID.NameId)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to RebootVM. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return irs.Failed, getErr
		}
		sysId = id
	}
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to RebootVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	if vmStatus != irs.Running {
		getErr := errors.New(fmt.Sprintf("Failed to SuspendVM. err = not Running"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	start := call.Start()
	err = server.Reboot(vmHandler.Client, sysId, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to RebootVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	LoggingInfo(hiscallInfo, start)

	// VM 상태 정보 반환
	vmStatus, err = vmHandler.GetVMStatus(vmIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to RebootVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	return vmStatus, nil
}

func (vmHandler *ClouditVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VM, vmIID.NameId, "TerminateVM()")
	start := call.Start()

	err := vmHandler.vmCleaner(vmIID)
	if err != nil {
		TerminateErr := errors.New(fmt.Sprintf("Failed to Terminate VM err = %s", err.Error()))
		cblogger.Error(TerminateErr.Error())
		LoggingError(hiscallInfo, TerminateErr)
		return irs.Failed, TerminateErr
	}
	LoggingInfo(hiscallInfo, start)

	// VM 상태 정보 반환
	return irs.Terminated, nil
}

func (vmHandler *ClouditVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VM, VM, "ListVMStatus()")

	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	start := call.Start()
	vmList, err := server.List(vmHandler.Client, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get ListVMStatus. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	LoggingInfo(hiscallInfo, start)

	vmStatusList := make([]*irs.VMStatusInfo, len(*vmList))
	for i, vm := range *vmList {
		vmStatusInfo := irs.VMStatusInfo{
			IId: irs.IID{
				NameId:   vm.Name,
				SystemId: vm.ID,
			},
			VmStatus: irs.VMStatus(vm.State),
		}
		vmStatusList[i] = &vmStatusInfo
	}
	return vmStatusList, nil
}

func (vmHandler *ClouditVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VM, vmIID.NameId, "GetVMStatus()")

	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	vmSystemID, err := vmHandler.getVmIdByName(vmIID.NameId)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMStatus. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	start := call.Start()
	vm, err := server.Get(vmHandler.Client, vmSystemID, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMStatus. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	LoggingInfo(hiscallInfo, start)

	// Set VM Status Info
	status := getVmStatus(vm.State)
	return status, nil
}

func (vmHandler *ClouditVMHandler) ListVM() ([]*irs.VMInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VM, VM, "ListVM()")

	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	start := call.Start()
	vmList, err := server.List(vmHandler.Client, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMList. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	LoggingInfo(hiscallInfo, start)

	vmInfoList := make([]*irs.VMInfo, len(*vmList))
	for i, vm := range *vmList {
		vmInfo := vmHandler.mappingServerInfo(vm)
		vmInfoList[i] = &vmInfo
	}
	return vmInfoList, nil
}

func (vmHandler *ClouditVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VM, vmIID.NameId, "GetVM()")

	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	start := call.Start()
	sysId := vmIID.SystemId
	if sysId == "" {
		id, err := vmHandler.getVmIdByName(vmIID.NameId)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Get. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return irs.VMInfo{}, getErr
		}
		sysId = id
	}
	vm, err := server.Get(vmHandler.Client, sysId, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VMInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)

	vmInfo := vmHandler.mappingServerInfo(*vm)

	return vmInfo, nil
}
func (vmHandler *ClouditVMHandler) vmCleaner(vmIID irs.IID) error{
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	// GetVM
	sysId := vmIID.SystemId
	if sysId == "" {
		id, err := vmHandler.getVmIdByName(vmIID.NameId)
		if err != nil {
			cleanErr := errors.New(fmt.Sprintf("Failed to Get. err = %s", err.Error()))
			return cleanErr
		}
		sysId = id
	}
	vm, err := server.Get(vmHandler.Client, sysId, &requestOpts)
	if err != nil {
		cleanErr := errors.New(fmt.Sprintf("Failed to Get. err = %s", err.Error()))
		return cleanErr
	}
	// DisassociatePublicIP
	if vm.AdaptiveIp != ""{
		if ok, err := vmHandler.DisassociatePublicIP(vm.AdaptiveIp); !ok {
			TerminateErr := errors.New(fmt.Sprintf("Failed DisassociatePublicIP"))
			if err != nil {
				TerminateErr = errors.New(fmt.Sprintf("Failed DisassociatePublicIP err= %s", err.Error()))
			}
			return TerminateErr
		}
		time.Sleep(5 * time.Second)
	}
	// Terminate
	if err := server.Terminate(vmHandler.Client, vm.ID, &requestOpts); err != nil {
		if err.Error() != "EOF" {
			cleanErr := errors.New(fmt.Sprintf("Failed to Terminate VM err = %s", err.Error()))
			return cleanErr
		}
	}
	curRetryCnt := 0
	maxRetryCnt := 120
	// Terminate Check
	for {
		checkVMList, err := server.List(vmHandler.Client, &requestOpts)
		if err == nil {
			terminateChk := true
			for _, checkVm := range *checkVMList {
				if checkVm.ID == vm.ID{
					terminateChk = false
				}
			}
			if terminateChk {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
		curRetryCnt++
		if curRetryCnt > maxRetryCnt {
			cleanErr := errors.New(fmt.Sprintf("Success to Terminate. but Failed to confirm Terminate VM err = exceeded maximum retry count %d", maxRetryCnt))
			return cleanErr
		}
	}
}

func (vmHandler *ClouditVMHandler) creatablePublicIP() (adaptiveip.IPInfo, error) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	if availableIPList, err := adaptiveip.ListAvailableIP(vmHandler.Client, &requestOpts); err != nil {
		return adaptiveip.IPInfo{}, err
	} else {
		if len(*availableIPList) == 0 {
			return adaptiveip.IPInfo{}, errors.New(fmt.Sprintf("There is no PublicIPs to allocate"))
		} else {
			return (*availableIPList)[0], nil
		}
	}
}

// VM에 PublicIP 연결
func (vmHandler *ClouditVMHandler) AssociatePublicIP(vmName string, vmIp string) (bool, error) {
	// 1. 사용 가능한 PublicIP 가져오기
	availableIP, err := vmHandler.creatablePublicIP()
	if err != nil {
		allocateErr := errors.New(fmt.Sprintf("There is no PublicIPs to allocate"))
		return false, allocateErr
	}

	// 2. PublicIP 생성 및 할당
	reqInfo := adaptiveip.PublicIPReqInfo{
		IP:        availableIP.IP,
		Name:      vmName,
		PrivateIP: vmIp,
	}
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	createOpts := client.RequestOpts{
		JSONBody:    reqInfo,
		MoreHeaders: authHeader,
	}
	_, err = adaptiveip.Create(vmHandler.Client, &createOpts)
	if err != nil {
		return false, err
	}
	return true, nil
}

// VM에 PublicIP 해제
func (vmHandler *ClouditVMHandler) DisassociatePublicIP(publicIP string) (bool, error) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := adaptiveip.Delete(vmHandler.Client, publicIP, &requestOpts); err != nil {
		cblogger.Error(err)
		return false, err
	} else {
		return true, nil
	}
}

func (vmHandler *ClouditVMHandler) mappingServerInfo(server server.ServerInfo) irs.VMInfo {
	// Get Default VM Info

	vmInfo := irs.VMInfo{
		IId: irs.IID{
			NameId:   server.Name,
			SystemId: server.ID,
		},
		Region: irs.RegionInfo{
			Region: server.TenantID,
			Zone:   server.TenantID,
		},
		ImageIId: irs.IID{
			NameId:   server.Template,
			SystemId: server.TemplateID,
		},
		VMSpecName: server.Spec,
		VpcIID: irs.IID{
			NameId:   defaultVPCName,
			SystemId: defaultVPCName,
		},
		KeyPairIId: irs.IID{
			NameId:   strings.Replace(server.Description, "keypair:", "", 1),
			SystemId: strings.Replace(server.Description, "keypair:", "", 1),
		},
		VMUserId:       SSHDefaultUser,
		PublicIP:       server.AdaptiveIp,
		PrivateIP:      server.PrivateIp,
		SSHAccessPoint: fmt.Sprintf("%s:%d", server.AdaptiveIp, SSHDefaultPort),
		RootDiskSize:   strconv.Itoa(server.VolumeSize),
		RootDeviceName: "Not visible in Cloudit",
		VMBlockDisk: "Not visible in Cloudit",
	}
	if server.CreatedAt != "" {
		timeArr := strings.Split(server.CreatedAt, " ")
		timeFormatStr := fmt.Sprintf("%sT%sZ", timeArr[0], timeArr[1])
		if createTime, err := time.Parse(time.RFC3339, timeFormatStr); err == nil {
			vmInfo.StartTime = createTime
		}
	}

	// Get Subnet Info
	VPCHandler := ClouditVPCHandler{
		Client:         vmHandler.Client,
		CredentialInfo: vmHandler.CredentialInfo,
	}
	subnet, err := VPCHandler.GetSubnet(irs.IID{NameId: server.SubnetAddr})
	if err == nil {
		vmInfo.SubnetIID = irs.IID{
			NameId:   subnet.Name,
			SystemId: subnet.ID,
		}
	}

	// Get SecurityGroup Info
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()
	vnicList, _ := ListVNic(authHeader, vmHandler.Client, server.ID)
	if vnicList != nil {
		defaultVnic := (*vnicList)[0]
		segGroupList := make([]irs.IID, len(defaultVnic.SecGroups))
		for i, s := range defaultVnic.SecGroups {
			segGroupList[i] = irs.IID{
				NameId:   s.Name,
				SystemId: s.Id,
			}
		}
		vmInfo.SecurityGroupIIds = segGroupList
	}
	return vmInfo
}

func (vmHandler *ClouditVMHandler) getVmIdByName(vmNameID string) (string, error) {
	var vmId string

	// VM 목록 검색
	vmList, err := vmHandler.ListVM()
	if err != nil {
		return "", err
	}

	// VM 목록에서 Name 기준 검색
	for _, v := range vmList {
		if strings.EqualFold(v.IId.NameId, vmNameID) {
			vmId = v.IId.SystemId
			break
		}
	}

	// 만약 VM이 검색되지 않을 경우 에러 처리
	if vmId == "" {
		err := errors.New(fmt.Sprintf("failed to find vm with name %s", vmNameID))
		return "", err
	}
	return vmId, nil
}

func getVmStatus(vmStatus string) irs.VMStatus {
	var resultStatus string
	switch strings.ToLower(vmStatus) {
	case "creating":
		resultStatus = "Creating"
	case "running":
		resultStatus = "Running"
	case "stopping":
		resultStatus = "Suspending"
	case "stopped":
		resultStatus = "Suspended"
	case "starting":
		resultStatus = "Resuming"
	case "rebooting":
		resultStatus = "Rebooting"
	case "terminating":
		resultStatus = "Terminating"
	case "terminated":
		resultStatus = "Terminated"
	case "failed":
	default:
		resultStatus = "Failed"
	}
	return irs.VMStatus(resultStatus)
}

func (vmHandler *ClouditVMHandler) attachSgToVnic(authHeader map[string]string, vmID string, reqClient *client.RestClient, vnicMac string, sgGroup []server.SecGroupInfo) {

	reqInfo := server.VMReqInfo{
		Secgroups: sgGroup,
	}
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
		JSONBody:    reqInfo,
	}
	nic.Put(reqClient, vmID, &requestOpts, vnicMac)
}
