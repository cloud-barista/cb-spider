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
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v2/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/floatingips"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/startstop"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	VM             = "VM"
	SSHDefaultUser = "cb-user"
)

type OpenStackVMHandler struct {
	Region        idrv.RegionInfo
	Client        *gophercloud.ServiceClient
	NetworkClient *gophercloud.ServiceClient
	VolumeClient  *gophercloud.ServiceClient
}

func (vmHandler *OpenStackVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Client.IdentityEndpoint, call.VM, vmReqInfo.IId.NameId, "StartVM()")

	// 가상서버 이름 중복 체크
	pager, err := servers.List(vmHandler.Client, servers.ListOpts{Name: vmReqInfo.IId.NameId}).AllPages()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to startVM err = failed to get vm with name %s", vmReqInfo.IId.NameId))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	existServer, err := servers.ExtractServers(pager)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to startVM err = failed to extract vm information with name %s", vmReqInfo.IId.NameId))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	if len(existServer) != 0 {
		createErr := errors.New(fmt.Sprintf("Failed to startVM err = VirtualMachine with name %s already exist", vmReqInfo.IId.NameId))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
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

	//  이미지 정보 조회 (Name)
	imageHandler := OpenStackImageHandler{
		Client: vmHandler.Client,
	}
	image, err := imageHandler.GetImage(vmReqInfo.ImageIID)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to startVM err = failed to get image, err : %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	// Flavor 정보 조회 (Name)
	vmSpecId, err := GetFlavorByName(vmHandler.Client, vmReqInfo.VMSpecName)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to startVM err = failed to get vmspec, err : %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	// VM 생성
	createVolumeFlag := true
	serverCreateOpts := servers.CreateOpts{
		Name:      vmReqInfo.IId.NameId,
		FlavorRef: vmSpecId,
		Metadata: map[string]string{
			"imagekey": image.IId.NameId,
		},
		Networks: []servers.Network{
			{UUID: vmReqInfo.VpcIID.SystemId},
		},
	}

	if vmReqInfo.RootDiskSize == "" || vmReqInfo.RootDiskSize == "default" {
		createVolumeFlag = false
		serverCreateOpts.ImageRef = image.IId.SystemId
	}

	sgIdArr := make([]string, len(vmReqInfo.SecurityGroupIIDs))
	for i, sg := range vmReqInfo.SecurityGroupIIDs {
		sgIdArr[i] = sg.SystemId
	}
	serverCreateOpts.SecurityGroups = sgIdArr

	// Add KeyPair
	createOpts := keypairs.CreateOptsExt{
		KeyName: vmReqInfo.KeyPairIID.NameId,
	}

	// KeyPair 정보 가져오기
	keyPair, err := keypairs.Get(vmHandler.Client, vmReqInfo.KeyPairIID.NameId).Extract()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to startVM err = failed to get keypair, err : %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	// cloud-init 스크립트 설정
	rootPath := os.Getenv("CBSPIDER_ROOT")
	fileData, err := ioutil.ReadFile(rootPath + "/cloud-driver-libs/.cloud-init-openstack/cloud-init")
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}
	fileStr := string(fileData)
	fileStr = strings.ReplaceAll(fileStr, "{{username}}", SSHDefaultUser)
	fileStr = strings.ReplaceAll(fileStr, "{{public_key}}", keyPair.PublicKey)

	// cloud-init 스크립트 적용
	serverCreateOpts.UserData = []byte(fileStr)
	createOpts.CreateOptsBuilder = serverCreateOpts

	start := call.Start()
	// VM RootDiskSize Set
	var server *servers.Server
	if !createVolumeFlag {
		server, err = servers.Create(vmHandler.Client, createOpts).Extract()
	} else {
		vmSize, err := strconv.Atoi(vmReqInfo.RootDiskSize)
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to startVM err = Invalid RootDiskSize"))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		blockDeviceSet := []bootfromvolume.BlockDevice{
			bootfromvolume.BlockDevice{
				UUID:                image.IId.SystemId,
				SourceType:          bootfromvolume.SourceImage,
				VolumeSize:          vmSize,
				DestinationType:     bootfromvolume.DestinationVolume,
				DeleteOnTermination: true,
			},
		}
		bootopt := bootfromvolume.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			BlockDevice:       blockDeviceSet,
		}
		server, err = bootfromvolume.Create(vmHandler.Client, bootopt).Extract()
	}

	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to startVM err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)

	var serverResult *servers.Server
	var serverInfo irs.VMInfo

	// VM 생성 완료까지 wait
	curRetryCnt := 0
	maxRetryCnt := 120
	if createVolumeFlag {
		maxRetryCnt = 240
	}
	for {
		// Check VM Deploy Status
		serverResult, err = servers.Get(vmHandler.Client, server.ID).Extract()
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to startVM err = failed to get vmInfo, err : %s", err))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}

		if strings.ToLower(serverResult.Status) == "active" {
			// Associate Public IP
			if ok, err := vmHandler.AssociatePublicIP(serverResult.ID); !ok {
				createErr := errors.New(fmt.Sprintf("Failed to startVM err = failed to Associate PublicIP, err : %s", err))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			// Get server info
			serverResult, err = servers.Get(vmHandler.Client, server.ID).Extract()
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to startVM err =  %s", err))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			serverInfo = vmHandler.mappingServerInfo(*serverResult)
			break
		}
		curRetryCnt++
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			createErr := errors.New(fmt.Sprintf("failed to start vm, exceeded maximum retry count %d", maxRetryCnt))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
	}

	return serverInfo, nil
}

func (vmHandler *OpenStackVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Client.IdentityEndpoint, call.VM, vmIID.NameId, "SuspendVM()")

	/*vmID, err := vmHandler.getVmIdByName(vmNameID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}*/
	start := call.Start()
	err := startstop.Stop(vmHandler.Client, vmIID.SystemId).Err
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	LoggingInfo(hiscallInfo, start)

	// 자체생성상태 반환 (OpenStack은 진행 중인 상태에 대한 정보 미제공)
	return irs.Suspending, nil
}

func (vmHandler *OpenStackVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Client.IdentityEndpoint, call.VM, vmIID.NameId, "ResumeVM()")

	/*vmID, err := vmHandler.getVmIdByName(vmNameID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}*/
	start := call.Start()
	err := startstop.Start(vmHandler.Client, vmIID.SystemId).Err
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	LoggingInfo(hiscallInfo, start)

	// 자체생성상태 반환 (OpenStack은 진행 중인 상태에 대한 정보 미제공)
	return irs.Resuming, nil
}

func (vmHandler *OpenStackVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Client.IdentityEndpoint, call.VM, vmIID.NameId, "RebootVM()")

	/*vmID, err := vmHandler.getVmIdByName(vmNameID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}*/
	start := call.Start()
	rebootOpts := servers.RebootOpts{
		Type: servers.SoftReboot,
	}

	err := servers.Reboot(vmHandler.Client, vmIID.SystemId, rebootOpts).ExtractErr()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	LoggingInfo(hiscallInfo, start)

	// 자체생성상태 반환 (OpenStack은 진행 중인 상태에 대한 정보 미제공)
	return irs.Rebooting, nil
}

func (vmHandler *OpenStackVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Client.IdentityEndpoint, call.VM, vmIID.NameId, "TerminateVM()")

	// VM 정보 조회
	server, err := vmHandler.GetVM(vmIID)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}

	// VM에 연결된 PublicIP 삭제
	pager, err := floatingips.List(vmHandler.Client).AllPages()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	publicIPList, err := floatingips.ExtractFloatingIPs(pager)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
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
		err := floatingips.Delete(vmHandler.Client, publicIPId).ExtractErr()
		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
			return irs.Failed, err
		}
	}

	// VM 삭제
	start := call.Start()
	err = servers.Delete(vmHandler.Client, server.IId.SystemId).ExtractErr()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	LoggingInfo(hiscallInfo, start)

	// 자체생성상태 반환 (OpenStack은 진행 중인 상태에 대한 정보 미제공)
	return irs.Terminating, nil
}

func (vmHandler *OpenStackVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Client.IdentityEndpoint, call.VM, VM, "ListVMStatus()")

	start := call.Start()
	pager, err := servers.List(vmHandler.Client, nil).AllPages()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	servers, err := servers.ExtractServers(pager)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	// Add to List
	vmStatusList := make([]*irs.VMStatusInfo, len(servers))
	for idx, s := range servers {
		vmStatus := getVmStatus(s.Status)
		vmStatusInfo := irs.VMStatusInfo{
			IId: irs.IID{
				NameId:   s.Name,
				SystemId: s.ID,
			},
			VmStatus: vmStatus,
		}
		vmStatusList[idx] = &vmStatusInfo
	}
	return vmStatusList, nil
}

func (vmHandler *OpenStackVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Client.IdentityEndpoint, call.VM, vmIID.NameId, "GetVMStatus()")

	start := call.Start()
	serverResult, err := servers.Get(vmHandler.Client, vmIID.SystemId).Extract()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return "", err
	}
	LoggingInfo(hiscallInfo, start)

	vmStatus := getVmStatus(serverResult.Status)
	return vmStatus, nil
}

func (vmHandler *OpenStackVMHandler) ListVM() ([]*irs.VMInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Client.IdentityEndpoint, call.VM, VM, "ListVM()")

	// 가상서버 목록 조회
	start := call.Start()
	pager, err := servers.List(vmHandler.Client, nil).AllPages()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	servers, err := servers.ExtractServers(pager)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
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
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Client.IdentityEndpoint, call.VM, vmIID.NameId, "GetVM()")

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

	start := call.Start()
	serverResult, err := servers.Get(vmHandler.Client, vmIID.SystemId).Extract()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	vmInfo := vmHandler.mappingServerInfo(*serverResult)
	return vmInfo, nil
}

func (vmHandler *OpenStackVMHandler) AssociatePublicIP(serverID string) (bool, error) {
	// PublicIP 생성
	externVPCName, _ := GetPublicVPCInfo(vmHandler.NetworkClient, "NAME")
	createOpts := floatingips.CreateOpts{
		Pool: externVPCName,
	}
	publicIP, err := floatingips.Create(vmHandler.Client, createOpts).Extract()
	if err != nil {
		return false, err
	}

	// PublicIP VM 연결
	curRetryCnt := 0
	//maxRetryCnt := 60
	maxRetryCnt := 120
	for {
		associateOpts := floatingips.AssociateOpts{
			FloatingIP: publicIP.IP,
		}
		err = floatingips.AssociateInstance(vmHandler.Client, serverID, associateOpts).ExtractErr()
		if err == nil {
			break
		} else {
			fmt.Println(fmt.Sprintf("[%d] err = %s", curRetryCnt, err))
		}

		time.Sleep(1 * time.Second)
		curRetryCnt++

		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf(fmt.Sprintf("failed to associate floating ip to vm, exceeded maximum retry count %d", maxRetryCnt))
			return false, errors.New(fmt.Sprintf("failed to associate floating ip to vm, exceeded maximum retry count %d", maxRetryCnt))
		}
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
	if creatTime, err := time.Parse(time.RFC3339, server.Created.String()); err == nil {
		vmInfo.StartTime = creatTime
	}
	imageHandler := OpenStackImageHandler{
		Client: vmHandler.Client,
	}
	// VM Image 정보 설정
	for key, value := range server.Metadata {
		if key == "imagekey" {
			imageInfo := irs.IID{
				NameId: value,
			}
			imageId, err := imageHandler.IDFromName(vmHandler.Client, value)
			if err == nil {
				imageInfo.SystemId = imageId
			}
			vmInfo.ImageIId = imageInfo
		}
	}
	// VM DiskSize Custom
	if len(server.AttachedVolumes) != 0 {
		for _, volume := range server.AttachedVolumes {
			rawVolume, _ := volumes.Get(vmHandler.VolumeClient, volume.ID).Extract()
			if rawVolume.Bootable == "true" {
				vmInfo.RootDiskSize = strconv.Itoa(rawVolume.Size)
			}
		}
	}

	// VM Flavor 정보 설정
	flavorId := server.Flavor["id"].(string)
	flavor, _ := flavors.Get(vmHandler.Client, flavorId).Extract()
	if flavor != nil {
		vmInfo.VMSpecName = flavor.Name
		if vmInfo.RootDiskSize == "" {
			vmInfo.RootDiskSize = strconv.Itoa(flavor.Disk)
		}
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
			if attach.ServerID == vmInfo.IId.SystemId {
				vmInfo.VMBlockDisk = attach.Device
			}
		}
	}

	return vmInfo
}
