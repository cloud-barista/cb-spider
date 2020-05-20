// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by hyokyung.kim@innogrid.co.kr, 2019.07.

package resources

import (
	"errors"
	"fmt"
	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/blockstorage/v2/volumes"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/floatingip"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/startstop"
	"github.com/rackspace/gophercloud/openstack/compute/v2/flavors"
	"github.com/rackspace/gophercloud/openstack/compute/v2/images"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/pagination"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type OpenStackVMHandler struct {
	Region        idrv.RegionInfo
	Client        *gophercloud.ServiceClient
	NetworkClient *gophercloud.ServiceClient
	VolumeClient  *gophercloud.ServiceClient
}

func (vmHandler *OpenStackVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	// 가상서버 이름 중복 체크
	pager, err := servers.List(vmHandler.Client, servers.ListOpts{Name: vmReqInfo.IId.NameId}).AllPages()
	if err != nil {
		cblogger.Error(fmt.Sprintf("failed to get vm with name %s", vmReqInfo.IId.NameId))
		return irs.VMInfo{}, err
	}
	existServer, err := servers.ExtractServers(pager)
	if err != nil {
		cblogger.Error(fmt.Sprintf("failed to extract vm information with name %s", vmReqInfo.IId.NameId))
		return irs.VMInfo{}, err
	}
	if len(existServer) != 0 {
		createErr := errors.New(fmt.Sprintf("VirtualMachine with name %s already exist", vmReqInfo.IId.NameId))
		return irs.VMInfo{}, createErr
	}

	/*vNetId, err := GetCBVNetId(vmHandler.NetworkClient)
	if err != nil {
		return irs.VMInfo{}, err
	}
	if vNetId == "" {
		cblogger.Error(fmt.Sprintf("failed to get vnetwork"))
		return irs.VMInfo{}, err
	}*/

	//  이미지 정보 조회 (Name)
	/*imageHandler := OpenStackImageHandler{
		Client: vmHandler.Client,
	}
	image, err := imageHandler.GetImage(vmReqInfo.IId)
	if err != nil {
		cblogger.Error(fmt.Sprintf("failed to get image, err : %s", err))
		return irs.VMInfo{}, err
	}*/

	//  네트워크 정보 조회 (Name)
	/*vNetworkHandler := OpenStackVPCHandler{
		Client: vmHandler.Client,
	}
	vNetwork, err := vNetworkHandler.GetVNetwork(vmReqInfo.VirtualNetworkId)
	if err != nil {
		cblogger.Error(fmt.Sprintf("failed to get virtual network, err : %s", err))
		return irs.VMInfo{}, err
	}*/

	// 보안그룹 정보 조회 (Name)
	/*securityHandler := OpenStackSecurityHandler{
		Client:        vmHandler.Client,
		NetworkClient: vmHandler.NetworkClient,
	}
	secGroups := make([]string, len(vmReqInfo.SecurityGroupIIDs))
	for i, s := range vmReqInfo.SecurityGroupIIDs {
		security, err := securityHandler.GetSecurity(s)
		if err != nil {
			cblogger.Error(fmt.Sprintf("failed to get security group, err : %s", err))
			return irs.VMInfo{}, err
			//continue
		}
		secGroups[i] = security.IId.SystemId
	}*/

	// Flavor 정보 조회 (Name)
	/*flavorId, err := GetFlavor(vmHandler.Client, vmReqInfo.VMSpecName)
	if err != nil {
		cblogger.Error(fmt.Sprintf("failed to get vm spec, err : %s", err))
		return irs.VMInfo{}, err
	}*/

	// VM 생성
	serverCreateOpts := servers.CreateOpts{
		Name:      vmReqInfo.IId.NameId,
		ImageName: vmReqInfo.ImageIID.NameId,
		//ImageRef:  vmReqInfo.ImageIID.SystemId,
		FlavorName: vmReqInfo.VMSpecName,
		//FlavorRef: *flavorId,
		Networks: []servers.Network{
			{UUID: vmReqInfo.VpcIID.SystemId},
		},
	}

	sgIdArr := make([]string, len(vmReqInfo.SecurityGroupIIDs))
	for i, sg := range vmReqInfo.SecurityGroupIIDs {
		sgIdArr[i] = sg.SystemId
	}
	serverCreateOpts.SecurityGroups = sgIdArr

	// Add KeyPair
	createOpts := keypairs.CreateOptsExt{
		CreateOptsBuilder: serverCreateOpts,
		KeyName:           vmReqInfo.KeyPairIID.NameId,
	}

	server, err := servers.Create(vmHandler.Client, createOpts).Extract()
	if err != nil {
		return irs.VMInfo{}, err
	}

	// VM 생성 완료까지 wait
	vmId := server.ID
	var isDeployed bool
	var serverInfo irs.VMInfo
	var serverResult *servers.Server
	for {
		if isDeployed {
			serverResult, err = servers.Get(vmHandler.Client, vmId).Extract()
			serverInfo = vmHandler.mappingServerInfo(*serverResult)
			break
		}

		time.Sleep(5 * time.Second)

		// Check VM Deploy Status
		serverResult, err = servers.Get(vmHandler.Client, vmId).Extract()
		if err != nil {
			return irs.VMInfo{}, err
		}
		if strings.ToLower(serverResult.Status) == "active" {
			// Associate Public IP
			if ok, err := vmHandler.AssociatePublicIP(serverResult.ID); !ok {
				return irs.VMInfo{}, err
			}
			isDeployed = true

		}
	}
	return serverInfo, nil
}

func (vmHandler *OpenStackVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	/*vmID, err := vmHandler.getVmIdByName(vmNameID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}*/
	err := startstop.Stop(vmHandler.Client, vmIID.SystemId).Err
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// 자체생성상태 반환 (OpenStack은 진행 중인 상태에 대한 정보 미제공)
	return irs.Suspending, nil
}

func (vmHandler *OpenStackVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	/*vmID, err := vmHandler.getVmIdByName(vmNameID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}*/
	err := startstop.Start(vmHandler.Client, vmIID.SystemId).Err
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// 자체생성상태 반환 (OpenStack은 진행 중인 상태에 대한 정보 미제공)
	return irs.Resuming, nil
}

func (vmHandler *OpenStackVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	/*vmID, err := vmHandler.getVmIdByName(vmNameID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}*/
	rebootOpts := servers.SoftReboot
	err := servers.Reboot(vmHandler.Client, vmIID.SystemId, rebootOpts).ExtractErr()
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// 자체생성상태 반환 (OpenStack은 진행 중인 상태에 대한 정보 미제공)
	return irs.Rebooting, nil
}

func (vmHandler *OpenStackVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	// VM 정보 조회
	server, err := vmHandler.GetVM(vmIID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// VM에 연결된 PublicIP 삭제
	pager, err := floatingip.List(vmHandler.Client).AllPages()
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	publicIPList, err := floatingip.ExtractFloatingIPs(pager)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// IP 기준 PublicIP 검색
	var publicIPId string
	for _, p := range publicIPList {
		if strings.EqualFold(server.PublicIP, p.IP) {
			publicIPId = p.ID
			break
		}
	}
	// Public IP 삭제
	if publicIPId != "" {
		err := floatingip.Delete(vmHandler.Client, publicIPId).ExtractErr()
		if err != nil {
			cblogger.Error(err)
			return irs.Failed, err
		}
	}

	// VM 삭제
	err = servers.Delete(vmHandler.Client, server.IId.SystemId).ExtractErr()
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// 자체생성상태 반환 (OpenStack은 진행 중인 상태에 대한 정보 미제공)
	return irs.Terminating, nil
}

func (vmHandler *OpenStackVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	var vmStatusList []*irs.VMStatusInfo

	pager := servers.List(vmHandler.Client, nil)
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		// Get VM Status
		list, err := servers.ExtractServers(page)
		if err != nil {
			return false, err
		}
		// Add to List
		for _, s := range list {
			vmStatus := getVmStatus(s.Status)
			vmStatusInfo := irs.VMStatusInfo{
				IId: irs.IID{
					NameId:   s.Name,
					SystemId: s.ID,
				},
				VmStatus: vmStatus,
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		}
		return true, nil
	})
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	return vmStatusList, nil
}

func (vmHandler *OpenStackVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	serverResult, err := servers.Get(vmHandler.Client, vmIID.SystemId).Extract()
	if err != nil {
		cblogger.Error(err)
		return irs.VMStatus(""), err
	}

	vmStatus := getVmStatus(serverResult.Status)
	return vmStatus, nil
}

func (vmHandler *OpenStackVMHandler) ListVM() ([]*irs.VMInfo, error) {

	// 가상서버 목록 조회
	pager, err := servers.List(vmHandler.Client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	servers, err := servers.ExtractServers(pager)
	if err != nil {
		return nil, err
	}

	// 가상서버 목록 정보 매핑
	vmList := make([]*irs.VMInfo, len(servers))
	for i, v := range servers {
		serverInfo := vmHandler.mappingServerInfo(v)
		vmList[i] = &serverInfo
	}
	return vmList, nil
}

func (vmHandler *OpenStackVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	// 기존의 vmID 기준 가상서버 조회 (old)
	/*serverResult, err := servers.Get(vmHandler.Client, vmID).Extract()
	if err != nil {
		cblogger.Info(err)
		return irs.VMInfo{}, err
	}*/
	/*vmID, err := vmHandler.getVmIdByName(vmIID.NameId)
	if err != nil {
		return irs.VMInfo{}, err
	}*/

	serverResult, err := servers.Get(vmHandler.Client, vmIID.SystemId).Extract()
	if err != nil {
		cblogger.Info(err)
		return irs.VMInfo{}, err
	}

	vmInfo := vmHandler.mappingServerInfo(*serverResult)
	return vmInfo, nil
}

func (vmHandler *OpenStackVMHandler) AssociatePublicIP(serverID string) (bool, error) {
	// PublicIP 생성
	createOpts := floatingip.CreateOpts{
		Pool: CBPublicIPPool,
	}
	publicIP, err := floatingip.Create(vmHandler.Client, createOpts).Extract()
	if err != nil {
		return false, err
	}
	// PublicIP VM 연결
	associateOpts := floatingip.AssociateOpts{
		ServerID:   serverID,
		FloatingIP: publicIP.IP,
	}
	err = floatingip.AssociateInstance(vmHandler.Client, associateOpts).ExtractErr()
	if err != nil {
		return false, err
	}
	return true, nil
}

func getVmStatus(vmStatus string) irs.VMStatus {
	var resultStatus string
	switch strings.ToLower(vmStatus) {
	case "build":
		resultStatus = "Creating"
	case "active":
		resultStatus = "Running"
	case "shutoff":
		resultStatus = "Suspended"
	case "reboot":
		resultStatus = "Rebooting"
	case "error":
	default:
		resultStatus = "Failed"
	}
	return irs.VMStatus(resultStatus)
}

func (vmHandler *OpenStackVMHandler) mappingServerInfo(server servers.Server) irs.VMInfo {

	// Get Default VM Info
	vmInfo := irs.VMInfo{
		IId: irs.IID{
			NameId:   server.Name,
			SystemId: server.ID,
		},
		Region: irs.RegionInfo{
			Zone:   vmHandler.Region.Zone,
			Region: vmHandler.Region.Region,
		},
		KeyPairIId: irs.IID{
			NameId:   server.KeyName,
			SystemId: server.KeyName,
		},
		//VMUserId:          server.UserID,
		//VMUserPasswd:      server.AdminPass,
		NetworkInterface:  server.HostID,
		KeyValueList:      nil,
		SecurityGroupIIds: nil,
	}
	if creatTime, err := time.Parse(time.RFC3339, server.Created); err == nil {
		vmInfo.StartTime = creatTime
	}

	// VM Image 정보 설정
	if len(server.Image) != 0 {
		imageId := server.Image["id"].(string)
		vmInfo.ImageIId = irs.IID{
			SystemId: imageId,
		}
		image, _ := images.Get(vmHandler.Client, imageId).Extract()
		if image != nil {
			vmInfo.ImageIId.NameId = image.Name
		}
	}

	// VM Flavor 정보 설정
	flavorId := server.Flavor["id"].(string)
	flavor, _ := flavors.Get(vmHandler.Client, flavorId).Extract()
	if flavor != nil {
		vmInfo.VMSpecName = flavor.Name
	}

	// VM SecurityGroup 정보 설정
	if len(server.SecurityGroups) != 0 {
		securityGroupIdArr := make([]irs.IID, len(server.SecurityGroups))
		for i, secGroupMap := range server.SecurityGroups {
			secGroupName := secGroupMap["name"].(string)
			securityGroupIdArr[i] = irs.IID{
				NameId: secGroupName,
			}
			secGroup, _ := GetSecurityByName(vmHandler.Client, secGroupName)
			if secGroup != nil {
				securityGroupIdArr[i].SystemId = secGroup.ID
			}
		}
		vmInfo.SecurityGroupIIds = securityGroupIdArr
	}

	for k, subnet := range server.Addresses {
		// VPC 정보 설정
		vmInfo.VpcIID.NameId = k
		network, _ := GetNetworkByName(vmHandler.NetworkClient, vmInfo.VpcIID.NameId)
		if network != nil {
			vmInfo.VpcIID.SystemId = network.ID
		}
		// PrivateIP, PublicIp 설정
		for _, addr := range subnet.([]interface{}) {
			addrMap := addr.(map[string]interface{})
			if addrMap["OS-EXT-IPS:type"] == "floating" {
				vmInfo.PublicIP = addrMap["addr"].(string)
			} else if addrMap["OS-EXT-IPS:type"] == "fixed" {
				vmInfo.PrivateIP = addrMap["addr"].(string)
			}
		}
	}

	// Subnet, Network Interface 정보 설정
	port, _ := GetPortByDeviceID(vmHandler.NetworkClient, vmInfo.IId.SystemId)
	if port != nil {
		// Subnet 정보 설정
		if len(port.FixedIPs) > 0 {
			ipInfo := port.FixedIPs[0]
			vmInfo.SubnetIID.SystemId = ipInfo.SubnetID
		}
		subnet, _ := GetSubnetByID(vmHandler.NetworkClient, vmInfo.SubnetIID.SystemId)
		if subnet != nil {
			vmInfo.SubnetIID.NameId = subnet.Name
		}

		// Network Interface 정보 설정
		vmInfo.NetworkInterface = port.ID
	}

	// Volume Disk 조회
	pages, _ := volumes.List(vmHandler.VolumeClient, volumes.ListOpts{}).AllPages()
	volList, _ := volumes.ExtractVolumes(pages)

	for _, vol := range volList {
		for _, attach := range vol.Attachments {
			if val, ok := attach["server_id"].(string); ok {
				if strings.EqualFold(val, vmInfo.IId.SystemId) {
					vmInfo.VMBlockDisk = attach["device"].(string)
				}
			}
		}
	}

	return vmInfo
}
