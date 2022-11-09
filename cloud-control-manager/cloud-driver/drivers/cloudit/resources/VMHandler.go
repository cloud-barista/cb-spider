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
	"encoding/json"
	"errors"
	"fmt"
	cdcom "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/disk"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/snapshot"
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

	err := notSupportRootDiskCustom(vmReqInfo)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	// 가상서버 이름 중복 체크
	exist, err := vmHandler.getExistVmName(vmReqInfo.IId.NameId)
	if exist {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s already exist", vmReqInfo.IId.NameId))
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	// 이미지, MyImage 정보 조회 (Name)
	imageHandler := ClouditImageHandler{
		Client:         vmHandler.Client,
		CredentialInfo: vmHandler.CredentialInfo,
	}
	myImageHandler := ClouditMyImageHandler{
		Client:         vmHandler.Client,
		CredentialInfo: vmHandler.CredentialInfo,
	}

	var image irs.ImageInfo
	var myImage irs.MyImageInfo
	if vmReqInfo.ImageType == irs.MyImage {
		var getMyImageErr error
		myImage, getMyImageErr = myImageHandler.GetMyImage(vmReqInfo.ImageIID)
		if getMyImageErr != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = Failed to get image, %s", getMyImageErr.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		if myImage.Status != irs.MyImageAvailable {
			myImageStatusErr := errors.New("Failed to Create VM. err = Given MyImage is not Available")
			cblogger.Error(myImageStatusErr.Error())
			LoggingError(hiscallInfo, myImageStatusErr)
			return irs.VMInfo{}, myImageStatusErr
		}
	} else {
		var getImageErr error
		image, getImageErr = imageHandler.GetImage(vmReqInfo.ImageIID)
		if getImageErr != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = Failed to get image, %s", getImageErr.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
	}

	//  네트워크 정보 조회 (Name)
	vpcHandler := ClouditVPCHandler{
		Client:         vmHandler.Client,
		CredentialInfo: vmHandler.CredentialInfo,
	}
	vpcInfo, err := vpcHandler.GetVPC(vmReqInfo.VpcIID)
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
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = subnet: '%s' cannot be found in VPC: '%s'", vmReqInfo.SubnetIID.NameId, vpcInfo.IId.NameId))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	// 보안그룹 정보 조회 (Name)
	sgHandler := ClouditSecurityHandler{
		Client:         vmHandler.Client,
		CredentialInfo: vmHandler.CredentialInfo,
	}

	securityInfo, err := sgHandler.getRawSecurityGroup(irs.IID{NameId: DefaultSGName})
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

	vmTag := server.VmTagInfo{
		MyImageIID: nil,
	}
	if vmReqInfo.ImageType == irs.MyImage {
		vmTag.MyImageIID = &myImage.IId
	}
	vmTag.Keypair = vmReqInfo.KeyPairIID.NameId
	vmTagByte, jsonMarshalErr := json.Marshal(vmTag)
	if jsonMarshalErr != nil {
		cblogger.Error(jsonMarshalErr.Error())
		LoggingError(hiscallInfo, jsonMarshalErr)
		return irs.VMInfo{}, jsonMarshalErr
	}
	vmTagStr := string(vmTagByte)
	//KeyPairDes := fmt.Sprintf("keypair:%s", vmReqInfo.KeyPairIID.NameId)

	clusterNameId := vmHandler.CredentialInfo.ClusterId
	clusterSystemId := ""
	if clusterNameId == "" {
		return irs.VMInfo{}, errors.New("Failed to Create Disk. err = ClusterId is required.")
	} else if clusterNameId == "default" {
		return irs.VMInfo{}, errors.New("Failed to Create Disk. err = Cloudit does not supports \"default\" cluster.")
	}

	requestURL := vmHandler.Client.CreateRequestBaseURL(client.ACE, "clusters")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = vmHandler.Client.Get(requestURL, &result.Body, &client.RequestOpts{
		MoreHeaders: authHeader,
	}); result.Err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	var clusterList []struct {
		Id   string
		Name string
	}
	if err := result.ExtractInto(&clusterList); err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	for _, cluster := range clusterList {
		if cluster.Name == clusterNameId {
			clusterSystemId = cluster.Id
		}
	}

	var reqInfo server.VMReqInfo
	rawRootImage, getRawRootImageErr := imageHandler.GetRawRootImage(irs.IID{SystemId: vmReqInfo.ImageIID.SystemId, NameId: vmReqInfo.ImageIID.NameId}, vmReqInfo.ImageType == irs.MyImage)
	if getRawRootImageErr != nil {
		return irs.VMInfo{}, errors.New(fmt.Sprintf("Failed to Create VM. err = %s", getRawRootImageErr.Error()))
	}
	isWindows := strings.Contains(strings.ToLower(rawRootImage.OS), "windows")

	reqInfo = server.VMReqInfo{
		SpecId:       *vmSpecId,
		Name:         vmReqInfo.IId.NameId,
		HostName:     vmReqInfo.IId.NameId,
		RootPassword: VMDefaultPassword,
		SubnetAddr:   subnet.Addr,
		Secgroups:    addUserSSHSG,
		Description:  vmTagStr,
		ClusterId:    clusterSystemId,
	}

	if isWindows {
		if len(vmReqInfo.IId.NameId) > 15 {
			reqInfo.HostName = vmReqInfo.IId.NameId[:15]
		}
		pwValidErr := cdcom.ValidateWindowsPassword(vmReqInfo.VMUserPasswd)
		if pwValidErr != nil {
			return irs.VMInfo{}, errors.New(fmt.Sprintf("Failed to Create VM. err = %s", pwValidErr))
		}

		reqInfo.RootPassword = vmReqInfo.VMUserPasswd
	}

	if vmReqInfo.ImageType == irs.MyImage {
		snapshotReqOpts := client.RequestOpts{
			MoreHeaders: authHeader,
		}
		snapshot, getSnapshotErr := snapshot.Get(myImageHandler.Client, myImage.IId.SystemId, &snapshotReqOpts)
		if getSnapshotErr != nil {
			return irs.VMInfo{}, errors.New(fmt.Sprintf("Failed to Create VM. err = %s", getSnapshotErr.Error()))
		}
		reqInfo.TemplateId = snapshot.TemplateId
		reqInfo.SnapshotId = myImage.IId.SystemId
	} else {
		reqInfo.TemplateId = image.IId.SystemId
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
	defer func() {
		if createError != nil {
			cleanerErr := vmHandler.vmCleaner(cleanVMIID)
			if cleanerErr != nil {
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

	if !isWindows {
		// SSH 접속 사용자 및 공개키 등록
		loginUserId := SSHDefaultUser
		createUserErr := errors.New(fmt.Sprintf("Failed adding cb-User to new VM"))

		// SSH 접속까지 시도
		curConnectionCnt := 0
		maxConnectionRetryCnt := 120
		for {
			cblogger.Info("Trying to connect via root user ...")
			_, err := RunCommand(vm.AdaptiveIp, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, "echo test")
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
		_, err = RunCommand(vm.AdaptiveIp, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, fmt.Sprintf("useradd -s /bin/bash %s -rm", loginUserId))
		if err != nil && vmReqInfo.ImageType != irs.MyImage {
			createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s = %s", createUserErr.Error(), err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		_, err = RunCommand(vm.AdaptiveIp, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, fmt.Sprintf("echo \"%s ALL=(root) NOPASSWD:ALL\" >> /etc/sudoers", loginUserId))
		if err != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s = %s", createUserErr.Error(), err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}

		// 공개키 등록
		_, err = RunCommand(vm.AdaptiveIp, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, fmt.Sprintf("mkdir -p /home/%s/.ssh", loginUserId))
		publicKey, err := GetPublicKey(vmHandler.CredentialInfo, vmReqInfo.KeyPairIID.NameId)
		if err != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s = %s", createUserErr.Error(), err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		_, err = RunCommand(vm.AdaptiveIp, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, fmt.Sprintf("echo \"%s\" > /home/%s/.ssh/authorized_keys", publicKey, loginUserId))
		if err != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s = %s", createUserErr.Error(), err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		// ssh 접속 방법 변경 (sshd_config 파일 변경)
		_, err = RunCommand(vm.AdaptiveIp, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, "sed -i 's/PasswordAuthentication yes/PasswordAuthentication no/g' /etc/ssh/sshd_config")
		if err != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s = %s", createUserErr.Error(), err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		_, err = RunCommand(vm.AdaptiveIp, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, "sed -i 's/#PubkeyAuthentication yes/PubkeyAuthentication yes/g' /etc/ssh/sshd_config")
		if err != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s = %s", createUserErr.Error(), err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		_, err = RunCommand(vm.AdaptiveIp, SSHDefaultPort, VMDefaultUser, VMDefaultPassword, "systemctl restart sshd")
		if err != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s = %s", createUserErr.Error(), err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
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

	vnicList, err := ListVNic(authHeader, vmHandler.Client, vm.ID)
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
		err := nic.ChangeSecurityGroup(vmHandler.Client, vm.ID, &changesgrequestOpts, defaultVnic.Mac, secGroups)
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

	// MyImage로 VM 생성 시, 볼륨 스냅샷으로 볼륨을 생성하고 attach
	var attachedVolumeList []irs.IID
	if vmReqInfo.ImageType == irs.MyImage {
		isFailed := false
		if createVolumeErr := myImageHandler.CreateAssociatedVolumeSnapshots(myImage.IId.NameId, vm.Name); createVolumeErr != nil {
			isFailed = true
		}
		if volumeAttachError := myImageHandler.AttachAssociatedVolumesToVM(myImage.IId.NameId, vm.ID); volumeAttachError != nil {
			isFailed = true
		}
		if isFailed {
			defer func() {
				cleanerErr := vmHandler.vmCleaner(cleanVMIID)
				if cleanerErr != nil {
					createError = errors.New(fmt.Sprintf("%s and Failed to rollback err = %s", createError.Error(), cleanerErr.Error()))
				}
			}()
		}

		vmVolumeList, getVmVolumesErr := server.GetRawVmVolumes(myImageHandler.Client, vm.ID, &requestOpts)
		if getVmVolumesErr != nil {
			createError = errors.New("Failed to Get VM Volume List")
		}

		for _, vmVolume := range *vmVolumeList {
			attachedVolumeList = append(attachedVolumeList, irs.IID{SystemId: vmVolume.ID, NameId: vmVolume.Name})
		}
	}

	diskHandler := ClouditDiskHandler{
		CredentialInfo: vmHandler.CredentialInfo,
		Client:         vmHandler.Client,
	}
	for _, dataDisk := range vmReqInfo.DataDiskIIDs {
		rawDisk, getDiskErr := diskHandler.getRawDisk(dataDisk)
		if getDiskErr != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s", getDiskErr.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		if attachDiskErr := diskHandler.attachDisk(irs.IID{SystemId: rawDisk.ID}, irs.IID{SystemId: vm.ID}); attachDiskErr != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
	}

	vmInfo, err := vmHandler.mappingServerInfo(*vm)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	if len(attachedVolumeList) != 0 {
		vmInfo.DataDiskIIDs = attachedVolumeList
	}

	if isWindows {
		vmInfo.VMUserId = "Administrator"
		vmInfo.VMUserPasswd = vmReqInfo.VMUserPasswd
	}

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

	rawVm, err := vmHandler.getRawVm(vmIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMStatus. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	// Set VM Status Info
	vmStatus := getVmStatus(rawVm.State)

	if vmStatus != irs.Running {
		getErr := errors.New(fmt.Sprintf("Failed to SuspendVM. err = not Running"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}

	start := call.Start()
	err = server.Suspend(vmHandler.Client, rawVm.ID, &requestOpts)
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

	rawVm, err := vmHandler.getRawVm(vmIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMStatus. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	vmStatus := getVmStatus(rawVm.State)
	if vmStatus != irs.Suspended {
		getErr := errors.New(fmt.Sprintf("Failed to ResumeVM. err = not Suspended"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	start := call.Start()
	err = server.Resume(vmHandler.Client, rawVm.ID, &requestOpts)
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
		rawVm, _ := vmHandler.getRawVm(vmIID)
		status := getVmStatus(rawVm.State)
		if status == irs.Running {
			break
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

	serverIP := rawVm.AdaptiveIp
	if serverIP == "" {
		createErr := errors.New(fmt.Sprintf("Failed to ResumeVM. err = exceeded maximum retry count %d", maxRetryCnt))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.Failed, createErr
	}

	isWindows := strings.Contains(strings.ToLower(rawVm.Template), "windows")
	if !isWindows {
		curRetryCnt = 0
		for {
			_, commandError := RunCommand(serverIP, 22, "dumy", "", "")
			errStr := commandError.Error()
			if strings.Contains(errStr, "ssh") {
				LoggingInfo(hiscallInfo, start)
				return irs.Running, nil
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
	return irs.Running, nil
}

func (vmHandler *ClouditVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VM, vmIID.NameId, "RebootVM()")

	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	rawVm, err := vmHandler.getRawVm(vmIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMStatus. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	vmStatus := getVmStatus(rawVm.State)
	if vmStatus != irs.Running {
		getErr := errors.New(fmt.Sprintf("Failed to RebootVM. err = not Running"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	start := call.Start()
	err = server.Reboot(vmHandler.Client, rawVm.ID, &requestOpts)
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
	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		rawVm, _ := vmHandler.getRawVm(vmIID)
		status := getVmStatus(rawVm.State)
		// Check VM Deploy Status
		if status == irs.Running {
			break
		}
		time.Sleep(1 * time.Second)
		curRetryCnt++
		if curRetryCnt > maxRetryCnt {
			createErr := errors.New(fmt.Sprintf("Failed to RebootVM. err = exceeded maximum retry count %d", maxRetryCnt))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.Failed, createErr
		}
	}

	serverIP := rawVm.AdaptiveIp
	if serverIP == "" {
		createErr := errors.New(fmt.Sprintf("Failed to RebootVM. err = exceeded maximum retry count %d", maxRetryCnt))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.Failed, createErr
	}
	isWindows := strings.Contains(strings.ToLower(rawVm.Template), "windows")
	curRetryCnt = 0
	if !isWindows {
		for {
			_, commandError := RunCommand(serverIP, 22, "dumy", "", "")
			errStr := commandError.Error()
			if strings.Contains(errStr, "ssh") {
				LoggingInfo(hiscallInfo, start)
				return irs.Running, nil
			}
			time.Sleep(1 * time.Second)
			curRetryCnt++
			if curRetryCnt > maxRetryCnt {
				createErr := errors.New(fmt.Sprintf("Failed to RebootVM. err = exceeded maximum retry count %d", maxRetryCnt))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.Failed, createErr
			}
		}
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

	start := call.Start()
	rawVm, err := vmHandler.getRawVm(vmIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMStatus. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	LoggingInfo(hiscallInfo, start)

	// Set VM Status Info
	status := getVmStatus(rawVm.State)
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
		vmInfo, err := vmHandler.mappingServerInfo(vm)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Get VMList. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}
		vmInfoList[i] = &vmInfo
	}
	return vmInfoList, nil
}

func (vmHandler *ClouditVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VM, vmIID.NameId, "GetVM()")

	start := call.Start()
	vm, err := vmHandler.getRawVm(vmIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VMInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	vmInfo, err := vmHandler.mappingServerInfo(*vm)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VMInfo{}, getErr
	}
	return vmInfo, nil
}
func (vmHandler *ClouditVMHandler) vmCleaner(vmIID irs.IID) error {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	rawVm, err := vmHandler.getRawVm(vmIID)
	if err != nil {
		cleanErr := errors.New(fmt.Sprintf("Failed to Get. err = %s", err.Error()))
		return cleanErr
	}
	// DisassociatePublicIP
	if rawVm.AdaptiveIp != "" {
		if ok, err := vmHandler.DisassociatePublicIP(rawVm.AdaptiveIp); !ok {
			TerminateErr := errors.New(fmt.Sprintf("Failed DisassociatePublicIP"))
			if err != nil {
				TerminateErr = errors.New(fmt.Sprintf("Failed DisassociatePublicIP err= %s", err.Error()))
			}
			return TerminateErr
		}
		time.Sleep(5 * time.Second)
	}
	// Terminate
	if err := server.Terminate(vmHandler.Client, rawVm.ID, &requestOpts); err != nil {
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
				if checkVm.ID == rawVm.ID {
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

func (vmHandler *ClouditVMHandler) mappingServerInfo(serverInfo server.ServerInfo) (irs.VMInfo, error) {
	// Get Default VM Info

	vmTag := server.VmTagInfo{}
	vmTagInfoByte := []byte(serverInfo.Description)
	json.Unmarshal(vmTagInfoByte, &vmTag)

	var vmImageIId irs.IID
	var imageType irs.ImageType
	if vmTag.MyImageIID == nil {
		vmImageIId.NameId = serverInfo.Template
		vmImageIId.SystemId = serverInfo.TemplateID
		imageType = irs.PublicImage
	} else {
		vmImageIId = *vmTag.MyImageIID
		imageType = irs.MyImage
	}

	var vmUser string
	if strings.Contains(strings.ToLower(serverInfo.Template), "window") {
		vmUser = "Administrator"
	} else {
		vmUser = SSHDefaultUser
	}

	vmInfo := irs.VMInfo{
		IId: irs.IID{
			NameId:   serverInfo.Name,
			SystemId: serverInfo.ID,
		},
		Region: irs.RegionInfo{
			Region: serverInfo.TenantID,
			Zone:   serverInfo.TenantID,
		},
		ImageType:      imageType,
		ImageIId:       vmImageIId,
		VMSpecName:     serverInfo.Spec,
		KeyPairIId:     irs.IID{NameId: vmTag.Keypair, SystemId: vmTag.Keypair},
		VMUserId:       vmUser,
		PublicIP:       serverInfo.AdaptiveIp,
		PrivateIP:      serverInfo.PrivateIp,
		SSHAccessPoint: fmt.Sprintf("%s:%d", serverInfo.AdaptiveIp, SSHDefaultPort),
		RootDiskSize:   strconv.Itoa(serverInfo.VolumeSize),
		RootDeviceName: "Not visible in Cloudit",
		VMBlockDisk:    "Not visible in Cloudit",
	}
	if serverInfo.CreatedAt != "" {
		timeArr := strings.Split(serverInfo.CreatedAt, " ")
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
	defaultVPC, err := VPCHandler.GetDefaultVPC()
	if err == nil {
		vmInfo.VpcIID = defaultVPC.IId
	}
	subnet, err := VPCHandler.GetSubnet(irs.IID{NameId: serverInfo.SubnetAddr})
	if err != nil {
		return irs.VMInfo{}, errors.New(fmt.Sprintf("Failed Get Subnet err= %s", err.Error()))
	}
	vmInfo.SubnetIID = irs.IID{
		NameId:   subnet.Name,
		SystemId: subnet.ID,
	}

	// Get SecurityGroup Info
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()
	vnicList, err := ListVNic(authHeader, vmHandler.Client, serverInfo.ID)
	if err != nil {
		return irs.VMInfo{}, errors.New(fmt.Sprintf("Failed Get VNic err= %s", err.Error()))
	}
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

	// Get Attached Disk Info
	vmDataVolumeList, getVmDataVolumeErr := vmHandler.getAttachedDiskList(vmInfo.IId)
	if getVmDataVolumeErr != nil {
		return irs.VMInfo{}, errors.New(fmt.Sprintf("Failed Get Attached Disk err= %s", err.Error()))
	}

	var dataDiskIIDs []irs.IID
	for _, vmDataVolume := range *vmDataVolumeList {
		dataDiskIIDs = append(dataDiskIIDs, irs.IID{NameId: vmDataVolume.Name, SystemId: vmDataVolume.ID})
	}
	vmInfo.DataDiskIIDs = dataDiskIIDs

	return vmInfo, nil
}

func (vmHandler *ClouditVMHandler) getExistVmName(name string) (bool, error) {
	if name == "" {
		return true, errors.New("invalid vmName")
	}
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	vmList, err := server.List(vmHandler.Client, &requestOpts)
	if err != nil {
		return true, err
	}
	for _, rawvm := range *vmList {
		if strings.EqualFold(name, rawvm.Name) {
			return true, nil
		}
	}
	return false, nil
}

func (vmHandler *ClouditVMHandler) getRawVm(vmIID irs.IID) (*server.ServerInfo, error) {
	if vmIID.SystemId == "" && vmIID.NameId == "" {
		return nil, errors.New("invalid IID")
	}
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	if vmIID.SystemId == "" {
		vmList, err := server.List(vmHandler.Client, &requestOpts)
		if err != nil {
			return nil, err
		}
		for _, rawvm := range *vmList {
			if strings.EqualFold(vmIID.NameId, rawvm.Name) {
				return &rawvm, nil
			}
		}
	} else {
		return server.Get(vmHandler.Client, vmIID.SystemId, &requestOpts)
	}
	return nil, errors.New("not found vm")
}

func (vmHandler *ClouditVMHandler) getAttachedDiskList(vmIID irs.IID) (*[]disk.DiskInfo, error) {
	vm, getVmError := vmHandler.getRawVm(vmIID)
	if getVmError != nil {
		return nil, errors.New(fmt.Sprintf("Failed to Get Attached Disk List err = %s", getVmError))
	}

	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	vmVolumeList, getVmVolumeListErr := server.GetRawVmVolumes(vmHandler.Client, vm.ID, &requestOpts)
	if getVmVolumeListErr != nil {
		return nil, errors.New(fmt.Sprintf("Failed to Get Attached Disk List err = %s", getVmVolumeListErr))
	}

	var vmDataVolumeList []disk.DiskInfo
	for _, vmVolume := range *vmVolumeList {
		if vmVolume.Dev != "vda" {
			vmDataVolumeList = append(vmDataVolumeList, vmVolume)
		}
	}

	return &vmDataVolumeList, nil
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

func notSupportRootDiskCustom(vmReqInfo irs.VMReqInfo) error {
	if vmReqInfo.RootDiskType != "" && strings.ToLower(vmReqInfo.RootDiskType) != "default" {
		return errors.New("CLOUDIT_CANNOT_CHANGE_ROOTDISKTYPE")
	}
	if vmReqInfo.RootDiskSize != "" && strings.ToLower(vmReqInfo.RootDiskSize) != "default" {
		return errors.New("CLOUDIT_CANNOT_CHANGE_ROOTDISKSIZE")
	}
	return nil
}
