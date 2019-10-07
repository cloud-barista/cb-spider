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
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/server"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

/*var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}*/

type ClouditVMHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func (vmHandler *ClouditVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	vmHandler.Client.TokenID = vmHandler.CredentialInfo.AuthToken
	authHeader := vmHandler.Client.AuthenticatedHeaders()

	// @TODO: VM 생성 요청 파라미터 정의 필요
	type SecGroupInfo struct {
		Id string `json:"id" required:"true"`
	}
	type VMReqInfo struct {
		TemplateId   string         `json:"templateId" required:"true"`
		SpecId       string         `json:"specId" required:"true"`
		Name         string         `json:"name" required:"true"`
		HostName     string         `json:"hostName" required:"true"`
		RootPassword string         `json:"rootPassword" required:"true"`
		SubnetAddr   string         `json:"subnetAddr" required:"true"`
		Secgroups    []SecGroupInfo `json:"secgroups" required:"true"`
		Description  int            `json:"description" required:"false"`
		Protection   int            `json:"protection" required:"false"`
	}

	reqInfo := VMReqInfo{
		TemplateId:   vmReqInfo.ImageInfo.Id,
		SpecId:       vmReqInfo.SpecID,
		Name:         vmReqInfo.Name,
		HostName:     vmReqInfo.Name,
		RootPassword: vmReqInfo.LoginInfo.AdminPassword,
		SubnetAddr:   vmReqInfo.VNetworkInfo.Id,
		Secgroups: []SecGroupInfo{
			{Id: vmReqInfo.SecurityInfo.Id},
		},
	}

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
		JSONBody:    reqInfo,
	}

	//var vmInfo server.ServerInfo
	if vm, err := server.Start(vmHandler.Client, &requestOpts); err != nil {
		return irs.VMInfo{}, err
	} else {
		if vmDetailInfo, err := server.Get(vmHandler.Client, vm.ID, &requestOpts); err != nil {
			return irs.VMInfo{}, err
		} else {
			vmInfo := mappingServerInfo(*vmDetailInfo)
			return vmInfo, nil
		}
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

func mappingServerInfo(server server.ServerInfo) irs.VMInfo {

	// Get Default VM Info
	vmInfo := irs.VMInfo{
		Name:         server.Name,
		Id:           server.ID,
		ImageID:      server.TemplateID,
		SpecID:       server.SpecId,
		SubNetworkID: server.SubnetAddr,
		PublicIP:     server.AdaptiveIp,
		PrivateIP:    server.PrivateIp,
		KeyPairID:    server.RootPassword,
	}

	return vmInfo
}
