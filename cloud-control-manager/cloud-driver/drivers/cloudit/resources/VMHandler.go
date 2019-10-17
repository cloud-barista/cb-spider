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
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
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
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	reqInfo := server.VMReqInfo{
		TemplateId:   vmReqInfo.ImageId,
		SpecId:       vmReqInfo.VMSpecId,
		Name:         vmReqInfo.VMName,
		HostName:     vmReqInfo.VMUserId,
		RootPassword: vmReqInfo.VMUserPasswd,
		SubnetAddr:   vmReqInfo.VirtualNetworkId,
	}

	secGroupList := make([]server.SecGroupInfo, len(vmReqInfo.SecurityGroupIds))
	for _, sec := range vmReqInfo.SecurityGroupIds {
		secGroupInfo := server.SecGroupInfo{
			Id: sec,
		}
		secGroupList = append(secGroupList, secGroupInfo)
	}
	reqInfo.Secgroups = secGroupList

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
		JSONBody:    reqInfo,
	}

	// VM 생성
	if vm, err := server.Start(vmHandler.Client, &requestOpts); err != nil {
		return irs.VMInfo{}, err
	} else {
		// VM 정보 가져오기
		vmDetailInfo, err := server.Get(vmHandler.Client, vm.ID, &requestOpts)
		if err != nil {
			return irs.VMInfo{}, err
		}

		// PublicIP 생성
		publicIPReqInfo := irs.PublicIPReqInfo{
			Name: vm.Name + "-PublicIP",
			KeyValueList: []irs.KeyValue{
				{
					Key:   "PrivateIP",
					Value: vm.PrivateIp,
				},
			},
		}
		if ok, err := vmHandler.AssociatePublicIP(publicIPReqInfo); !ok {
			return irs.VMInfo{}, err
		}

		// 생성된 VM 정보 리턴
		vmInfo := mappingServerInfo(*vmDetailInfo)
		return vmInfo, nil
	}
}

func (vmHandler *ClouditVMHandler) SuspendVM(vmID string) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := server.Suspend(vmHandler.Client, vmID, &requestOpts); err != nil {
		cblogger.Error(err)
	}
}

func (vmHandler *ClouditVMHandler) ResumeVM(vmID string) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := server.Resume(vmHandler.Client, vmID, &requestOpts); err != nil {
		cblogger.Error(err)
	}
}

func (vmHandler *ClouditVMHandler) RebootVM(vmID string) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := server.Reboot(vmHandler.Client, vmID, &requestOpts); err != nil {
		cblogger.Error(err)
	}
}

func (vmHandler *ClouditVMHandler) TerminateVM(vmID string) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := server.Terminate(vmHandler.Client, vmID, &requestOpts); err != nil {
		cblogger.Error(err)
	}
}

func (vmHandler *ClouditVMHandler) ListVMStatus() []*irs.VMStatusInfo {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if vmList, err := server.List(vmHandler.Client, &requestOpts); err != nil {
		cblogger.Error(err)
		return []*irs.VMStatusInfo{}
	} else {
		var vmStatusList []*irs.VMStatusInfo
		for _, vm := range *vmList {
			vmStatusInfo := irs.VMStatusInfo{
				VmId:     vm.ID,
				VmStatus: irs.VMStatus(vm.State),
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		}
		return vmStatusList
	}
}

func (vmHandler *ClouditVMHandler) GetVMStatus(vmID string) irs.VMStatus {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if vm, err := server.Get(vmHandler.Client, vmID, &requestOpts); err != nil {
		cblogger.Error(err)
		//TODO ??
		return ""
	} else {
		return irs.VMStatus(vm.State)
	}
}

func (vmHandler *ClouditVMHandler) ListVM() []*irs.VMInfo {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if vmList, err := server.List(vmHandler.Client, &requestOpts); err != nil {
		cblogger.Error(err)
		return []*irs.VMInfo{}
	} else {
		var vmInfoList []*irs.VMInfo
		for _, vm := range *vmList {
			vmInfo := mappingServerInfo(vm)
			vmInfoList = append(vmInfoList, &vmInfo)
		}
		return vmInfoList
	}
}

func (vmHandler *ClouditVMHandler) GetVM(vmID string) irs.VMInfo {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if vm, err := server.Get(vmHandler.Client, vmID, &requestOpts); err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}
	} else {
		vmInfo := mappingServerInfo(*vm)
		return vmInfo
	}
}

// VM에 PublicIP 연결
func (vmHandler *ClouditVMHandler) AssociatePublicIP(publicIPReqInfo irs.PublicIPReqInfo) (bool, error) {
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
		IP:   availableIP.IP,
		Name: publicIPReqInfo.Name,
	}
	// VM PrivateIP 값 설정
	for _, meta := range publicIPReqInfo.KeyValueList {
		if meta.Key == "PrivateIP" {
			reqInfo.PrivateIP = meta.Value
		}
	}

	createOpts := client.RequestOpts{
		JSONBody:    reqInfo,
		MoreHeaders: authHeader,
	}
	publicIP, err := adaptiveip.Create(vmHandler.Client, &createOpts)
	if err != nil {
		cblogger.Error(err)
		return false, err
	} else {
		spew.Dump(publicIP)
		return true, nil
	}
}

func mappingServerInfo(server server.ServerInfo) irs.VMInfo {

	// Get Default VM Info
	vmInfo := irs.VMInfo{
		Name:             server.Name,
		Id:               server.ID,
		ImageId:          server.TemplateID,
		VMSpecId:         server.SpecId,
		VirtualNetworkId: server.SubnetAddr,
		PublicIP:         server.AdaptiveIp,
		PrivateIP:        server.PrivateIp,
		KeyPairName:      server.RootPassword,
	}

	return vmInfo
}
