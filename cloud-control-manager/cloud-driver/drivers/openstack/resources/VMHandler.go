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
	cdcom "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	"github.com/gophercloud/gophercloud"
	volumes3 "github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/floatingips"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/startstop"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type OpenstackOSTYPE string

const (
	WindowBaseUser                 = "Administrator"
	VM                             = "VM"
	SSHDefaultUser                 = "cb-user"
	WindowOS       OpenstackOSTYPE = "windows"
	LinuxOS        OpenstackOSTYPE = "linux"
	UnknownOS      OpenstackOSTYPE = "unknown"
)

type OpenStackVMHandler struct {
	Region        idrv.RegionInfo
	ComputeClient *gophercloud.ServiceClient
	NetworkClient *gophercloud.ServiceClient
	VolumeClient  *gophercloud.ServiceClient
}

func (vmHandler *OpenStackVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (startvm irs.VMInfo, createErr error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.ComputeClient.IdentityEndpoint, call.VM, vmReqInfo.IId.NameId, "StartVM()")
	err := notSupportRootDiskCustom(vmReqInfo)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to startVM err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	// 가상서버 이름 중복 체크
	pager, err := servers.List(vmHandler.ComputeClient, servers.ListOpts{Name: vmReqInfo.IId.NameId}).AllPages()
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
	if len(vmReqInfo.DataDiskIIDs) > 0 {
		if vmHandler.VolumeClient == nil {
			createErr := errors.New(fmt.Sprintf("Failed to startVM err = this Openstack cannot provide VolumeClient. DataDisk cannot be attach"))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		for _, dataDiskIID := range vmReqInfo.DataDiskIIDs {
			disk, err := getRawDisk(dataDiskIID, vmHandler.VolumeClient)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to startVM err = Failed to get DataDisk err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			if disk.Status != "available" {
				createErr := errors.New(fmt.Sprintf("Failed to startVM err = Attach is only available when available Status"))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
		}
	}
	// Flavor 정보 조회 (Name)
	vmSpec, err := GetFlavorByName(vmHandler.ComputeClient, vmReqInfo.VMSpecName)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to startVM err = failed to get vmspec, err : %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	// Private IP 할당 서브넷 매핑
	// Private IP 할당 서브넷 매핑 - vpc 및 서브넷 확인
	vpcHandler := OpenStackVPCHandler{
		Client:   vmHandler.NetworkClient,
		VMClient: vmHandler.ComputeClient,
	}
	rawVpc, err := vpcHandler.getRawVPC(vmReqInfo.VpcIID)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to startVM err %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	fixedIPSubnet := irs.IID{}
	for _, rawsubnetId := range rawVpc.Subnets {
		subnet, err := subnets.Get(vmHandler.NetworkClient, rawsubnetId).Extract()
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to startVM err %s", err))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		if subnet.ID == vmReqInfo.SubnetIID.SystemId || subnet.Name == vmReqInfo.SubnetIID.NameId {
			fixedIPSubnet.SystemId = subnet.ID
			fixedIPSubnet.NameId = subnet.Name
			break
		}
	}
	if fixedIPSubnet.SystemId == "" {
		createErr := errors.New(fmt.Sprintf("Failed to startVM err not found subnet"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	fixedIp, err := vmHandler.availableFixedIP(fixedIPSubnet)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to startVM err %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	// SecurityGroup 준비
	segHandler := OpenStackSecurityHandler{
		Client:        vmHandler.ComputeClient,
		NetworkClient: vmHandler.NetworkClient,
	}

	sgIdArr := make([]string, len(vmReqInfo.SecurityGroupIIDs))
	for i, sg := range vmReqInfo.SecurityGroupIIDs {
		SecurityGroup, err := segHandler.getRawSecurity(sg)
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to startVM err %s", err))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		sgIdArr[i] = SecurityGroup.ID
	}
	serverCreateOpts := servers.CreateOpts{
		Name:      vmReqInfo.IId.NameId,
		FlavorRef: vmSpec.ID,
		Networks: []servers.Network{
			{UUID: rawVpc.ID, FixedIP: fixedIp},
		},
		SecurityGroups: sgIdArr,
	}

	var server servers.Server
	// Public
	if vmReqInfo.ImageType != irs.MyImage {
		imageOSType, err := getOSTypeByImage(vmReqInfo.ImageIID, vmHandler.ComputeClient)
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to startVM err = failed to get image os type, err : %s", err))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		if imageOSType == WindowOS {
			server, err = severCreatePublicImageWindowOS(serverCreateOpts, vmReqInfo, vmHandler.VolumeClient, vmHandler.ComputeClient)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to startVM err =  %s", err))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
		} else {
			server, err = severCreatePublicImageLinuxOS(serverCreateOpts, vmReqInfo, vmHandler.VolumeClient, vmHandler.ComputeClient)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to startVM err = %s", err))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
		}
	} else {
		//MyImage
		imageOSType, err := getOSTypeByMyImage(vmReqInfo.ImageIID, vmHandler.ComputeClient)
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to startVM err = failed to get image os type, err : %s", err))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		if imageOSType == WindowOS {
			server, err = severCreateMyImageWindowOS(serverCreateOpts, vmReqInfo, vmHandler.VolumeClient, vmHandler.ComputeClient)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to startVM err = %s", err))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
		} else {
			server, err = severCreateMyImageLinuxOS(serverCreateOpts, vmReqInfo, vmHandler.VolumeClient, vmHandler.ComputeClient)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to startVM err = %s", err))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
		}
	}

	defer func() {
		if createErr != nil {
			cleanVMIId := irs.IID{
				SystemId: server.ID,
			}
			cleanErr := vmHandler.vmCleaner(cleanVMIId)
			if cleanErr != nil {
				createErr = errors.New(fmt.Sprintf("%s Failed to rollback deleting err = %s", createErr, cleanErr))
			}
		}
	}()

	var serverResult *servers.Server
	var serverInfo irs.VMInfo
	start := call.Start()
	// VM 생성 완료까지 wait
	curRetryCnt := 0
	maxRetryCnt := 240
	for {
		// Check VM Deploy Status
		serverResult, err = servers.Get(vmHandler.ComputeClient, server.ID).Extract()
		if err != nil {
			createErr = errors.New(fmt.Sprintf("Failed to startVM err = failed to get vmInfo, err : %s", err))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		if strings.ToLower(serverResult.Status) == "active" {
			// Associate Public IP
			if ok, err := vmHandler.AssociatePublicIP(serverResult.ID); !ok {
				createErr = errors.New(fmt.Sprintf("Failed to startVM err = failed to Associate PublicIP, err : %s", err))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			// Get server info
			break
		}
		curRetryCnt++
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			createErr = errors.New(fmt.Sprintf("failed to start vm, exceeded maximum retry count %d", maxRetryCnt))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
	}
	//  If DataDisk Exist
	if len(vmReqInfo.DataDiskIIDs) > 0 {
		serverResult, err = AttachList(vmReqInfo.DataDiskIIDs, vmReqInfo.IId, vmHandler.ComputeClient, vmHandler.VolumeClient)
		if err != nil {
			createErr = errors.New(fmt.Sprintf("Failed to startVM err =  %s", err))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
	} else {
		serverResult, err = servers.Get(vmHandler.ComputeClient, server.ID).Extract()
		if err != nil {
			createErr = errors.New(fmt.Sprintf("Failed to startVM err =  %s", err))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
	}
	serverInfo = vmHandler.mappingServerInfo(*serverResult)
	password, err := getPassword(*serverResult)
	if err == nil {
		serverInfo.VMUserPasswd = password
	}
	LoggingInfo(hiscallInfo, start)
	return serverInfo, nil
}

func (vmHandler *OpenStackVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.ComputeClient.IdentityEndpoint, call.VM, vmIID.NameId, "SuspendVM()")

	/*vmID, err := vmHandler.getVmIdByName(vmNameID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}*/
	start := call.Start()
	err := startstop.Stop(vmHandler.ComputeClient, vmIID.SystemId).Err
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
	hiscallInfo := GetCallLogScheme(vmHandler.ComputeClient.IdentityEndpoint, call.VM, vmIID.NameId, "ResumeVM()")

	/*vmID, err := vmHandler.getVmIdByName(vmNameID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}*/
	start := call.Start()
	err := startstop.Start(vmHandler.ComputeClient, vmIID.SystemId).Err
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
	hiscallInfo := GetCallLogScheme(vmHandler.ComputeClient.IdentityEndpoint, call.VM, vmIID.NameId, "RebootVM()")

	/*vmID, err := vmHandler.getVmIdByName(vmNameID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}*/
	start := call.Start()
	rebootOpts := servers.RebootOpts{
		Type: servers.SoftReboot,
	}

	err := servers.Reboot(vmHandler.ComputeClient, vmIID.SystemId, rebootOpts).ExtractErr()
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
	hiscallInfo := GetCallLogScheme(vmHandler.ComputeClient.IdentityEndpoint, call.VM, vmIID.NameId, "TerminateVM()")
	start := call.Start()
	cleanErr := vmHandler.vmCleaner(vmIID)
	if cleanErr != nil {
		return irs.Failed, cleanErr
	}
	LoggingInfo(hiscallInfo, start)

	// 자체생성상태 반환 (OpenStack은 진행 중인 상태에 대한 정보 미제공)
	return irs.Terminated, nil
}

func (vmHandler *OpenStackVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.ComputeClient.IdentityEndpoint, call.VM, VM, "ListVMStatus()")

	start := call.Start()
	pager, err := servers.List(vmHandler.ComputeClient, nil).AllPages()
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
	hiscallInfo := GetCallLogScheme(vmHandler.ComputeClient.IdentityEndpoint, call.VM, vmIID.NameId, "GetVMStatus()")

	start := call.Start()
	serverResult, err := servers.Get(vmHandler.ComputeClient, vmIID.SystemId).Extract()
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
	hiscallInfo := GetCallLogScheme(vmHandler.ComputeClient.IdentityEndpoint, call.VM, VM, "ListVM()")

	// 가상서버 목록 조회
	start := call.Start()
	pager, err := servers.List(vmHandler.ComputeClient, nil).AllPages()
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
	hiscallInfo := GetCallLogScheme(vmHandler.ComputeClient.IdentityEndpoint, call.VM, vmIID.NameId, "GetVM()")

	// 기존의 vmID 기준 가상서버 조회 (old)
	/*serverResult, err := servers.Get(vmHandler.ComputeClient, vmID).Extract()
	if err != nil {
		cblogger.Info(err)
		return irs.VMInfo{}, err
	}*/
	/*vmID, err := vmHandler.getVmIdByName(vmIID.NameId)
	if err != nil {
		return irs.VMInfo{}, err
	}*/

	start := call.Start()
	serverResult, err := vmHandler.getRawVM(vmIID)
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
	publicIP, err := floatingips.Create(vmHandler.ComputeClient, createOpts).Extract()
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
		err = floatingips.AssociateInstance(vmHandler.ComputeClient, serverID, associateOpts).ExtractErr()
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

		//VMUserId:          server.UserID,
		//VMUserPasswd:      server.AdminPass,
		NetworkInterface:  server.HostID,
		KeyValueList:      nil,
		SecurityGroupIIds: nil,
	}
	OSType, err := getOSTypeByServer(server)
	if err == nil {
		if OSType == WindowOS {
			vmInfo.VMUserId = WindowBaseUser
		} else {
			vmInfo.KeyPairIId = irs.IID{
				NameId:   server.KeyName,
				SystemId: server.KeyName,
			}
		}
	}
	if creatTime, err := time.Parse(time.RFC3339, server.Created.String()); err == nil {
		vmInfo.StartTime = creatTime
	}
	// VM Image 정보 설정
	for key, value := range server.Metadata {
		if key == "imagekey" {
			imageInfo := irs.IID{
				NameId: value,
			}
			image, err := getRawImage(imageInfo, vmHandler.ComputeClient)
			if err == nil {
				imageInfo.SystemId = image.ID
			}
			vmInfo.ImageIId = imageInfo
		}
	}
	// VM DiskSize Custom
	if len(server.AttachedVolumes) > 0 && vmHandler.VolumeClient != nil {
		var dataDisks []irs.IID
		allVolume, err := getAllVolumeByServerAttachedVolume(server.AttachedVolumes, vmHandler.VolumeClient)
		if err == nil {
			for _, vol := range allVolume {
				if vol.Bootable == "true" {
					vmInfo.RootDiskSize = strconv.Itoa(vol.Size)
					for _, att := range vol.Attachments {
						if att.ServerID == server.ID {
							vmInfo.RootDeviceName = att.Device
						}
					}
				} else {
					dataDisks = append(dataDisks, irs.IID{NameId: vol.Name, SystemId: vol.ID})
				}
			}
			vmInfo.DataDiskIIDs = dataDisks
		}
	}
	// VM Flavor 정보 설정
	flavorId := server.Flavor["id"].(string)
	flavor, _ := flavors.Get(vmHandler.ComputeClient, flavorId).Extract()
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
			secGroup, _ := GetSecurityByName(vmHandler.ComputeClient, secGroupName)
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

	return vmInfo
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func Hosts(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	// remove network address and broadcast address and gateway, dhcp
	if ips != nil && len(ips) > 3 {
		return ips[3 : len(ips)-1], nil
	}
	return nil, errors.New("Not Exist Available IPs")
}

func difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

func (vmHandler *OpenStackVMHandler) availableFixedIP(subnetIId irs.IID) (string, error) {
	if subnetIId.SystemId == "" {
		return "", errors.New(fmt.Sprintf("Failed to Create SubnetIP err = invalid subnetIId"))
	}
	subnet, err := subnets.Get(vmHandler.NetworkClient, subnetIId.SystemId).Extract()
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to Create SubnetIP err = %s", err))
	}

	pager, err := servers.List(vmHandler.ComputeClient, nil).AllPages()
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to Create SubnetIP err = %s", err))
	}
	allVMList, err := servers.ExtractServers(pager)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to Create SubnetIP err = %s", err))
	}
	//vms, err := vmHandler.ListVM()
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to Create SubnetIP err = %s", err))
	}
	subnetIps, err := Hosts(subnet.CIDR)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to Create SubnetIP err = %s", err))
	}
	var vmIps []string
	for _, vm := range allVMList {
		privateIP := ""
		for _, addrs := range vm.Addresses {
			// PrivateIP 설정
			for _, addr := range addrs.([]interface{}) {
				addrMap := addr.(map[string]interface{})
				if addrMap["OS-EXT-IPS:type"] == "fixed" {
					privateIP = addrMap["addr"].(string)
				}
			}
		}
		if privateIP != "" {
			vmIps = append(vmIps, privateIP)
		}

	}
	filteredIps := difference(subnetIps, vmIps)
	if len(filteredIps) > 0 {
		rand.Seed(time.Now().UnixNano())
		index := rand.Intn(len(filteredIps))
		return filteredIps[index], nil
	}
	return "", errors.New(fmt.Sprintf("Failed to Create SubnetIP err = Not Exist Available IPs"))
}

func (vmHandler *OpenStackVMHandler) vmCleaner(vmIId irs.IID) error {
	// VM 정보 조회
	server, err := vmHandler.GetVM(vmIId)
	if err != nil {
		return err
	}
	if server.PublicIP != "" {
		// VM에 연결된 PublicIP 삭제
		pager, err := floatingips.List(vmHandler.ComputeClient).AllPages()
		if err != nil {
			return err
		}
		publicIPList, err := floatingips.ExtractFloatingIPs(pager)
		if err != nil {
			return err
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
			err := floatingips.Delete(vmHandler.ComputeClient, publicIPId).ExtractErr()
			if err != nil {
				return err
			}
		}
	}
	err = servers.Delete(vmHandler.ComputeClient, server.IId.SystemId).ExtractErr()
	if err != nil {
		return err
	}
	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		listopts := servers.ListOpts{
			Name: server.IId.NameId,
		}
		pager, err := servers.List(vmHandler.ComputeClient, listopts).AllPages()
		if err != nil {
			curRetryCnt++
			time.Sleep(1 * time.Second)
			continue
		}

		servers, err := servers.ExtractServers(pager)
		if err != nil {
			curRetryCnt++
			time.Sleep(1 * time.Second)
			continue
		}
		if len(servers) == 0 {
			return nil
		}
		curRetryCnt++
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			return errors.New(fmt.Sprintf("Success to Terminate. but Failed to confirm Terminate VM err = exceeded maximum retry count %d", maxRetryCnt))
		}
	}
}

func (vmHandler *OpenStackVMHandler) getRawVM(vmIId irs.IID) (*servers.Server, error) {
	if !CheckIIDValidation(vmIId) {
		return nil, errors.New("invalid IID")
	}
	if vmIId.SystemId == "" {
		pager, err := servers.List(vmHandler.ComputeClient, nil).AllPages()
		if err != nil {
			return nil, err
		}
		rawServers, err := servers.ExtractServers(pager)
		if err != nil {
			return nil, err
		}
		for _, vm := range rawServers {
			if vm.Name == vmIId.NameId {
				return &vm, nil
			}
		}
		return nil, errors.New("not found vm")
	} else {
		return servers.Get(vmHandler.ComputeClient, vmIId.SystemId).Extract()
	}
}

func notSupportRootDiskCustom(vmReqInfo irs.VMReqInfo) error {
	if vmReqInfo.RootDiskType != "" && strings.ToLower(vmReqInfo.RootDiskType) != "default" {
		return errors.New("OPENSTACK_CANNOT_CHANGE_ROOTDISKTYPE")
	}
	return nil
}

func getAllVolumeByServerAttachedVolume(attachedVolumes []servers.AttachedVolume, volumeClient *gophercloud.ServiceClient) (volumeList []volumes3.Volume, err error) {
	if volumeClient == nil {
		return make([]volumes3.Volume, 0), nil
	}
	volumeList = make([]volumes3.Volume, len(attachedVolumes))
	for i, attachedVolume := range attachedVolumes {
		vol, err := getRawDisk(irs.IID{SystemId: attachedVolume.ID}, volumeClient)
		if err != nil {
			return nil, err
		}
		volumeList[i] = vol
	}
	return volumeList, nil
}

func getOSTypeByImage(imageIID irs.IID, computeClient *gophercloud.ServiceClient) (OpenstackOSTYPE, error) {
	image, err := getRawImage(imageIID, computeClient)
	if err != nil {
		return UnknownOS, err
	}
	value, exist := image.Metadata["os_type"]
	if !exist {
		return LinuxOS, nil
	}
	if value == "windows" {
		return WindowOS, nil
	}
	return LinuxOS, nil
}

func getOSTypeByMyImage(imageIID irs.IID, computeClient *gophercloud.ServiceClient) (OpenstackOSTYPE, error) {
	image, err := getRawSnapshot(imageIID, computeClient)
	if err != nil {
		return UnknownOS, err
	}
	value, exist := image.Metadata["os_type"]
	if !exist {
		return LinuxOS, nil
	}
	if value == "windows" {
		return WindowOS, nil
	}
	return LinuxOS, nil
}

func getOSTypeByServer(server servers.Server) (OpenstackOSTYPE, error) {
	value, exist := server.Metadata["os_type"]
	if !exist {
		return LinuxOS, nil
	}
	if value == "windows" {
		return WindowOS, nil
	}
	return LinuxOS, nil
}

func getPassword(server servers.Server) (string, error) {
	pw, exist := server.Metadata["admin_pass"]
	if exist {
		return pw, nil
	}
	return "", errors.New("failed get password err = password information not found")
}

func checkWindowVMReqInfo(vmReqInfo irs.VMReqInfo) error {
	//if len(vmReqInfo.IId.NameId) > 15 {
	//	return errors.New("for Windows, VM's computeName cannot exceed 15 characters")
	//}
	// https://learn.microsoft.com/ko-KR/troubleshoot/windows-server/identity/naming-conventions-for-computer-domain-site-ou
	matchCase, _ := regexp.MatchString(`[\/?:|*<>\\\"]+`, vmReqInfo.IId.NameId)
	if matchCase {
		return errors.New("for Windows, VM's computeName contains unacceptable special characters")
	}
	if vmReqInfo.VMUserId != WindowBaseUser {
		return errors.New("for Windows, the userId only provides Administrator")
	}
	// password
	err := cdcom.ValidateWindowsPassword(vmReqInfo.VMUserPasswd)
	if err != nil {
		return err
	}
	if vmReqInfo.KeyPairIID.NameId != "" || vmReqInfo.KeyPairIID.SystemId != "" {
		return errors.New("for Windows, SSH key login method is not supported")
	}
	return nil
}

func linuxServerCreatOptConvertKeyPairWrapping(baseServerCreateOpt servers.CreateOpts, keyPairIID irs.IID, computeClient *gophercloud.ServiceClient) (keypairs.CreateOptsExt, error) {
	keyPair, err := GetRawKey(computeClient, keyPairIID)
	if err != nil {
		return keypairs.CreateOptsExt{}, err
	}
	rootPath := os.Getenv("CBSPIDER_ROOT")
	fileData, err := ioutil.ReadFile(rootPath + "/cloud-driver-libs/.cloud-init-openstack/cloud-init")
	if err != nil {
		return keypairs.CreateOptsExt{}, err
	}
	fileStr := string(fileData)
	fileStr = strings.ReplaceAll(fileStr, "{{username}}", SSHDefaultUser)
	fileStr = strings.ReplaceAll(fileStr, "{{public_key}}", keyPair.PublicKey)

	baseServerCreateOpt.UserData = []byte(fileStr)
	createOptsExt := keypairs.CreateOptsExt{
		KeyName: keyPair.Name,
	}
	createOptsExt.CreateOptsBuilder = baseServerCreateOpt
	return createOptsExt, nil
}

func createBlockDeviceSet(imageUUID string, diskSize string) (bootfromvolume.BlockDevice, error) {
	volumeSize, err := strconv.Atoi(diskSize)
	if err != nil {
		return bootfromvolume.BlockDevice{}, errors.New(fmt.Sprintf("Invalid RootDiskSize"))
	}
	return bootfromvolume.BlockDevice{
		UUID:                imageUUID,
		SourceType:          bootfromvolume.SourceImage,
		VolumeSize:          volumeSize,
		DestinationType:     bootfromvolume.DestinationVolume,
		DeleteOnTermination: true,
	}, nil
}

func severCreatePublicImageLinuxOS(baseServerCreateOpt servers.CreateOpts, vmReqInfo irs.VMReqInfo, VolumeClient *gophercloud.ServiceClient, computeClient *gophercloud.ServiceClient) (servers.Server, error) {
	image, err := getRawImage(vmReqInfo.ImageIID, computeClient)
	if err != nil {
		return servers.Server{}, err
	}
	baseServerCreateOpt.ImageRef = image.ID
	baseServerCreateOpt.Metadata = map[string]string{
		"imagekey": image.ID,
		"os_type":  string(LinuxOS),
	}

	if !(vmReqInfo.RootDiskSize == "" || vmReqInfo.RootDiskSize == "default") {
		if VolumeClient == nil {
			// Disk Size 변경 && vmHandler.VolumeClient == nil
			return servers.Server{}, errors.New(fmt.Sprintf("this Openstack cannot provide VolumeClient. RootDiskSize cannot be changed"))
		}
		rootBlockDeviceSet, err := createBlockDeviceSet(image.ID, vmReqInfo.RootDiskSize)
		if err != nil {
			return servers.Server{}, err
		}
		blockDeviceSet := []bootfromvolume.BlockDevice{
			rootBlockDeviceSet,
		}
		// Linux
		createOptsExt, err := linuxServerCreatOptConvertKeyPairWrapping(baseServerCreateOpt, vmReqInfo.KeyPairIID, computeClient)
		if err != nil {
			return servers.Server{}, err
		}
		bootOpt := bootfromvolume.CreateOptsExt{
			CreateOptsBuilder: createOptsExt,
			BlockDevice:       blockDeviceSet,
		}
		server, err := bootfromvolume.Create(computeClient, bootOpt).Extract()
		if err != nil {
			return servers.Server{}, err
		}
		return *server, nil
	} else {
		// Disk Size 변경 X
		if VolumeClient == nil { // Disk Size 변경 X && VolumeClient == nil
			server, err := servers.Create(computeClient, baseServerCreateOpt).Extract()
			if err != nil {
				return servers.Server{}, err
			}
			return *server, nil
		} else { // Disk Size 변경 X && VolumeClient != nil
			vmSpec, err := GetFlavorByName(computeClient, vmReqInfo.VMSpecName)
			if err != nil {
				return servers.Server{}, errors.New(fmt.Sprintf("failed to get vmspec, err : %s", err))
			}
			rootBlockDeviceSet, err := createBlockDeviceSet(image.ID, strconv.Itoa(vmSpec.Disk))
			if err != nil {
				return servers.Server{}, err
			}
			blockDeviceSet := []bootfromvolume.BlockDevice{
				rootBlockDeviceSet,
			}
			createOptsExt, err := linuxServerCreatOptConvertKeyPairWrapping(baseServerCreateOpt, vmReqInfo.KeyPairIID, computeClient)
			if err != nil {
				return servers.Server{}, err
			}
			bootOpt := bootfromvolume.CreateOptsExt{
				CreateOptsBuilder: createOptsExt,
				BlockDevice:       blockDeviceSet,
			}
			server, err := bootfromvolume.Create(computeClient, bootOpt).Extract()
			if err != nil {
				return servers.Server{}, err
			}
			return *server, nil
		}
	}

}

func severCreatePublicImageWindowOS(baseServerCreateOpt servers.CreateOpts, vmReqInfo irs.VMReqInfo, VolumeClient *gophercloud.ServiceClient, computeClient *gophercloud.ServiceClient) (servers.Server, error) {
	err := checkWindowVMReqInfo(vmReqInfo)
	if err != nil {
		return servers.Server{}, err
	}
	image, err := getRawImage(vmReqInfo.ImageIID, computeClient)
	if err != nil {
		return servers.Server{}, err
	}
	baseServerCreateOpt.ImageRef = image.ID
	baseServerCreateOpt.Metadata = map[string]string{
		"imagekey":   image.ID,
		"admin_pass": vmReqInfo.VMUserPasswd,
		"os_type":    string(WindowOS),
	}
	// Disk Size 변경
	if !(vmReqInfo.RootDiskSize == "" || vmReqInfo.RootDiskSize == "default") {
		if VolumeClient == nil {
			// Disk Size 변경 && vmHandler.VolumeClient == nil
			return servers.Server{}, errors.New(fmt.Sprintf("this Openstack cannot provide VolumeClient. RootDiskSize cannot be changed"))
		}
		rootBlockDeviceSet, err := createBlockDeviceSet(image.ID, vmReqInfo.RootDiskSize)
		if err != nil {
			return servers.Server{}, err
		}
		blockDeviceSet := []bootfromvolume.BlockDevice{
			rootBlockDeviceSet,
		}
		// Window
		bootOpt := bootfromvolume.CreateOptsExt{
			CreateOptsBuilder: baseServerCreateOpt,
			BlockDevice:       blockDeviceSet,
		}
		server, err := bootfromvolume.Create(computeClient, bootOpt).Extract()
		if err != nil {
			return servers.Server{}, err
		}
		return *server, nil
	} else {
		// Disk Size 변경 X
		if VolumeClient == nil { // Disk Size 변경 X && VolumeClient == nil
			server, err := servers.Create(computeClient, baseServerCreateOpt).Extract()
			if err != nil {
				return servers.Server{}, err
			}
			return *server, nil
		} else { // Disk Size 변경 X && VolumeClient != nil
			vmSpec, err := GetFlavorByName(computeClient, vmReqInfo.VMSpecName)
			if err != nil {
				return servers.Server{}, errors.New(fmt.Sprintf("failed to get vmspec, err : %s", err))
			}
			rootBlockDeviceSet, err := createBlockDeviceSet(image.ID, strconv.Itoa(vmSpec.Disk))
			if err != nil {
				return servers.Server{}, err
			}
			blockDeviceSet := []bootfromvolume.BlockDevice{
				rootBlockDeviceSet,
			}
			bootOpt := bootfromvolume.CreateOptsExt{
				CreateOptsBuilder: baseServerCreateOpt,
				BlockDevice:       blockDeviceSet,
			}
			server, err := bootfromvolume.Create(computeClient, bootOpt).Extract()
			if err != nil {
				return servers.Server{}, err
			}
			return *server, nil
		}
	}
}

func severCreateMyImageWindowOS(baseServerCreateOpt servers.CreateOpts, vmReqInfo irs.VMReqInfo, VolumeClient *gophercloud.ServiceClient, ComputeClient *gophercloud.ServiceClient) (servers.Server, error) {
	err := checkWindowVMReqInfo(vmReqInfo)
	if err != nil {
		return servers.Server{}, err
	}
	image, err := getRawSnapshot(vmReqInfo.ImageIID, ComputeClient)
	if err != nil {
		return servers.Server{}, err
	}
	baseServerCreateOpt.ImageRef = image.ID
	baseServerCreateOpt.Metadata = map[string]string{
		"imagekey":   image.ID,
		"os_type":    string(WindowOS),
		"admin_pass": vmReqInfo.VMUserPasswd,
	}
	_, exist := image.Metadata["block_device_mapping"]
	if exist && VolumeClient == nil {
		return servers.Server{}, errors.New(fmt.Sprintf("Failed to startVM err = this Openstack cannot provide VolumeClient. BlockDevice information is located within the snapshot."))
	}
	server, err := servers.Create(ComputeClient, baseServerCreateOpt).Extract()
	if err != nil {
		return servers.Server{}, err
	}
	return *server, err
}

func severCreateMyImageLinuxOS(baseServerCreateOpt servers.CreateOpts, vmReqInfo irs.VMReqInfo, volumeClient *gophercloud.ServiceClient, computeClient *gophercloud.ServiceClient) (servers.Server, error) {
	image, err := getRawSnapshot(vmReqInfo.ImageIID, computeClient)
	if err != nil {
		return servers.Server{}, errors.New(fmt.Sprintf("failed to get image, err : %s", err))
	}
	baseServerCreateOpt.ImageRef = image.ID
	baseServerCreateOpt.Metadata = map[string]string{
		"imagekey": image.ID,
		"os_type":  string(LinuxOS),
	}
	// snapshot block_device_mapping Check (volumeClient)
	_, exist := image.Metadata["block_device_mapping"]
	if exist && volumeClient == nil {
		return servers.Server{}, errors.New(fmt.Sprintf("Failed to startVM err = this Openstack cannot provide VolumeClient. BlockDevice information is located within the snapshot."))
	}

	createOptsExt, err := linuxServerCreatOptConvertKeyPairWrapping(baseServerCreateOpt, vmReqInfo.KeyPairIID, computeClient)
	server, err := servers.Create(computeClient, createOptsExt).Extract()
	if err != nil {
		return servers.Server{}, err
	}
	return *server, err
}
