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
	cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/server"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/dna/adaptiveip"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

const (
	vmDefaultUser = "root"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type ClouditVMHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func (vmHandler *ClouditVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	// 가상서버 이름 중복 체크
	vmId, _ := vmHandler.getVmIdByName(vmReqInfo.IId.NameId)
	if vmId != "" {
		errMsg := fmt.Sprintf("VirtualMachine with name %s already exist", vmReqInfo.IId.NameId)
		createErr := errors.New(errMsg)
		return irs.VMInfo{}, createErr
	}

	// 이미지 정보 조회 (Name)
	imageHandler := ClouditImageHandler{
		Client:         vmHandler.Client,
		CredentialInfo: vmHandler.CredentialInfo,
	}
	image, err := imageHandler.GetImage(vmReqInfo.ImageIID)
	if err != nil {
		cblogger.Error(fmt.Sprintf("failed to get image, err : %s", err))
		return irs.VMInfo{}, err
	}

	//  네트워크 정보 조회 (Name)
	VPCHandler := ClouditVPCHandler{
		Client:         vmHandler.Client,
		CredentialInfo: vmHandler.CredentialInfo,
	}
	VPC, err := VPCHandler.GetSubnet(vmReqInfo.SubnetIID)
	if err != nil {
		cblogger.Error(fmt.Sprintf("failed to get virtual network, err : %s", err))
		return irs.VMInfo{}, err
	}

	// 보안그룹 정보 조회 (Name)
	securityHandler := ClouditSecurityHandler{
		Client:         vmHandler.Client,
		CredentialInfo: vmHandler.CredentialInfo,
	}
	secGroups := make([]server.SecGroupInfo, len(vmReqInfo.SecurityGroupIIDs))
	for i, s := range vmReqInfo.SecurityGroupIIDs {
		security, err := securityHandler.GetSecurity(s)
		if err != nil {
			cblogger.Error(fmt.Sprintf("failed to get security group, err : %s", err))
			continue
		}
		secGroups[i] = server.SecGroupInfo{
			Id: security.IId.SystemId,
		}
	}

	// Spec 정보 조회 (Name)
	vmSpecId, err := GetVMSpecByName(vmHandler.Client.AuthenticatedHeaders(), vmHandler.Client, vmReqInfo.VMSpecName)
	if err != nil {
		cblogger.Error(fmt.Sprintf("failed to get vm spec, err : %s", err))
		return irs.VMInfo{}, err
	}

	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	reqInfo := server.VMReqInfo{
		TemplateId:   image.IId.SystemId,
		SpecId:       *vmSpecId,
		Name:         vmReqInfo.IId.NameId,
		HostName:     vmReqInfo.IId.NameId,
		RootPassword: vmReqInfo.VMUserPasswd,
		SubnetAddr:   VPC.Addr,
		Secgroups:    secGroups,
	}

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
		JSONBody:    reqInfo,
	}

	// VM 생성
	creatingVm, err := server.Start(vmHandler.Client, &requestOpts)
	if err != nil {
		return irs.VMInfo{}, err
	}

	// VM 생성 완료까지 wait
	for {
		// Check VM Deploy Status
		vmInfo, err := server.Get(vmHandler.Client, creatingVm.ID, &requestOpts)
		if err != nil {
			return irs.VMInfo{}, err
		}

		if vmInfo.PrivateIp == "" {
			time.Sleep(1 * time.Second)
			continue
		} else {
			ok, err := vmHandler.AssociatePublicIP(creatingVm.Name, vmInfo.PrivateIp)
			if !ok {
				return irs.VMInfo{}, err
			}
			break
		}
	}

	vm, err := server.Get(vmHandler.Client, creatingVm.ID, &requestOpts)
	if err != nil {
		return irs.VMInfo{}, err
	}
	vmInfo := vmHandler.mappingServerInfo(*vm)
	return vmInfo, nil
}

func (vmHandler *ClouditVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := server.Suspend(vmHandler.Client, vmIID.SystemId, &requestOpts); err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// VM 상태 정보 반환
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	return vmStatus, nil
}

func (vmHandler *ClouditVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := server.Resume(vmHandler.Client, vmIID.SystemId, &requestOpts); err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// VM 상태 정보 반환
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	return vmStatus, nil
}

func (vmHandler *ClouditVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := server.Reboot(vmHandler.Client, vmIID.SystemId, &requestOpts); err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// VM 상태 정보 반환
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	return vmStatus, nil
}

func (vmHandler *ClouditVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	// VM 정보 조회
	vmInfo, err := vmHandler.GetVM(vmIID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// 연결된 PublicIP 반환
	if vmInfo.PublicIP != "" {
		if ok, err := vmHandler.DisassociatePublicIP(vmInfo.PublicIP); !ok {
			return irs.Failed, err
		}
		time.Sleep(5 * time.Second)
	}

	if err := server.Terminate(vmHandler.Client, vmInfo.IId.SystemId, &requestOpts); err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// VM 상태 정보 반환
	return irs.Terminating, nil
}

func (vmHandler *ClouditVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	vmList, err := server.List(vmHandler.Client, &requestOpts)
	if err != nil {
		cblogger.Error(err)
		return []*irs.VMStatusInfo{}, err
	}
	vmStatusList := make([]*irs.VMStatusInfo, len(*vmList))
	for i, vm := range *vmList {
		vmStatusInfo := irs.VMStatusInfo{
			IId: irs.IID{
				NameId: vm.ID,
			},
			VmStatus: irs.VMStatus(vm.State),
		}
		vmStatusList[i] = &vmStatusInfo
	}
	return vmStatusList, nil
}

func (vmHandler *ClouditVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	vmID, err := vmHandler.getVmIdByName(vmIID.NameId)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	vm, err := server.Get(vmHandler.Client, vmID, &requestOpts)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// Set VM Status Info
	var resultStatus string
	switch strings.ToLower(vm.State) {
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
	return irs.VMStatus(resultStatus), nil
}

func (vmHandler *ClouditVMHandler) ListVM() ([]*irs.VMInfo, error) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	vmList, err := server.List(vmHandler.Client, &requestOpts)
	if err != nil {
		cblogger.Error(err)
		return []*irs.VMInfo{}, err
	}

	vmInfoList := make([]*irs.VMInfo, len(*vmList))
	for i, vm := range *vmList {
		vmInfo := vmHandler.mappingServerInfo(vm)
		vmInfoList[i] = &vmInfo
	}
	return vmInfoList, nil
}

func (vmHandler *ClouditVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	vm, err := server.Get(vmHandler.Client, vmIID.SystemId, &requestOpts)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}
	vmInfo := vmHandler.mappingServerInfo(*vm)
	return vmInfo, nil
}

// VM에 PublicIP 연결
func (vmHandler *ClouditVMHandler) AssociatePublicIP(vmName string, vmIp string) (bool, error) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	var availableIP adaptiveip.IPInfo

	// 1. 사용 가능한 PublicIP 목록 가져오기
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	if availableIPList, err := adaptiveip.ListAvailableIP(vmHandler.Client, &requestOpts); err != nil {
		return false, err
	} else {
		if len(*availableIPList) == 0 {
			allocateErr := errors.New(fmt.Sprintf("There is no PublicIPs to allocate"))
			return false, allocateErr
		} else {
			availableIP = (*availableIPList)[0]
		}
	}

	// 2. PublicIP 생성 및 할당
	reqInfo := adaptiveip.PublicIPReqInfo{
		IP:        availableIP.IP,
		Name:      vmName + "-PublicIP",
		PrivateIP: vmIp,
	}

	createOpts := client.RequestOpts{
		JSONBody:    reqInfo,
		MoreHeaders: authHeader,
	}
	_, err := adaptiveip.Create(vmHandler.Client, &createOpts)
	if err != nil {
		cblogger.Error(err)
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
		VMUserId:  vmDefaultUser,
		PublicIP:  server.AdaptiveIp,
		PrivateIP: server.PrivateIp,
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
			vmId = v.IId.NameId
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
