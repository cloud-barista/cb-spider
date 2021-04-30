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
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	ProvisioningStateCode string = "ProvisioningState/succeeded"
	VM                           = "VM"
)

type AzureVMHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Ctx            context.Context
	Client         *compute.VirtualMachinesClient
	SubnetClient   *network.SubnetsClient
	NicClient      *network.InterfacesClient
	PublicIPClient *network.PublicIPAddressesClient
	DiskClient     *compute.DisksClient
}

func (vmHandler *AzureVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmReqInfo.IId.NameId, "StartVM()")

	// Check VM Exists
	vm, err := vmHandler.Client.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmReqInfo.IId.NameId, compute.InstanceView)
	if vm.ID != nil {
		createErr := errors.New(fmt.Sprintf("virtualMachine with name %s already exist", vmReqInfo.IId.NameId))
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	// Check login method (keypair, password)
	if vmReqInfo.VMUserPasswd != "" && vmReqInfo.KeyPairIID.NameId != "" {
		createErr := errors.New("specify one login method, Password or Keypair")
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	// TODO: nested flow 개선
	// PublicIP 생성
	publicIPIId, err := CreatePublicIP(vmHandler, vmReqInfo)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}

	// TODO: nested flow 개선
	// VNic 생성
	vNicIId, err := CreateVNic(vmHandler, vmReqInfo, publicIPIId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}

	vmOpts := compute.VirtualMachine{
		Location: &vmHandler.Region.Region,
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(vmReqInfo.VMSpecName),
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  &vmReqInfo.IId.NameId,
				AdminUsername: to.StringPtr(CBVMUser),
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					{
						//ID: &vmReqInfo.NetworkInterfaceId,
						ID: &vNicIId.SystemId,
						NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
							Primary: to.BoolPtr(true),
						},
					},
				},
			},
		},
	}

	// Image 설정
	if strings.Contains(vmReqInfo.ImageIID.SystemId, ":") {
		imageArr := strings.Split(vmReqInfo.ImageIID.SystemId, ":")
		// URN 기반 퍼블릭 이미지 설정
		vmOpts.StorageProfile = &compute.StorageProfile{
			ImageReference: &compute.ImageReference{
				Publisher: &imageArr[0],
				Offer:     &imageArr[1],
				Sku:       &imageArr[2],
				Version:   &imageArr[3],
			},
		}
	} else {
		// 사용자 프라이빗 이미지 설정
		vmOpts.StorageProfile = &compute.StorageProfile{
			ImageReference: &compute.ImageReference{
				ID: &vmReqInfo.ImageIID.NameId,
			},
		}
	}

	// KeyPair 설정
	if vmReqInfo.KeyPairIID.NameId != "" {
		publicKey, err := GetPublicKey(vmHandler.CredentialInfo, vmReqInfo.KeyPairIID.NameId)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.VMInfo{}, err
		}
		vmOpts.OsProfile.LinuxConfiguration = &compute.LinuxConfiguration{
			SSH: &compute.SSHConfiguration{
				PublicKeys: &[]compute.SSHPublicKey{
					{
						Path:    to.StringPtr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", CBVMUser)),
						KeyData: to.StringPtr(publicKey),
					},
				},
			},
		}
	} else {
		vmOpts.OsProfile.AdminPassword = to.StringPtr(vmReqInfo.VMUserPasswd)
	}

	// VM 정보 태깅 설정
	if vmReqInfo.KeyPairIID.NameId != "" {
		vmOpts.Tags = map[string]*string{
			"keypair":  to.StringPtr(vmReqInfo.KeyPairIID.NameId),
			"publicip": to.StringPtr(publicIPIId.NameId),
		}
	}

	start := call.Start()
	future, err := vmHandler.Client.CreateOrUpdate(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmReqInfo.IId.NameId, vmOpts)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	vm, err = vmHandler.Client.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmReqInfo.IId.NameId, compute.InstanceView)
	if err != nil {
		LoggingError(hiscallInfo, err)
	}

	vmInfo := vmHandler.mappingServerInfo(vm)
	return vmInfo, nil
}

func (vmHandler *AzureVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "SuspendVM()")

	start := call.Start()
	future, err := vmHandler.Client.PowerOff(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmIID.NameId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}

	// Get VM Status
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}

	LoggingInfo(hiscallInfo, start)

	return vmStatus, nil
}

func (vmHandler *AzureVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "ResumeVM()")

	start := call.Start()
	future, err := vmHandler.Client.Start(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmIID.NameId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	LoggingInfo(hiscallInfo, start)

	// 자체생성상태 반환
	return irs.Resuming, nil
}

func (vmHandler *AzureVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "RebootVM()")

	start := call.Start()
	future, err := vmHandler.Client.Restart(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmIID.NameId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	LoggingInfo(hiscallInfo, start)

	// 자체생성상태 반환
	return irs.Rebooting, nil
}

func (vmHandler *AzureVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "TerminateVM()")

	// VM 삭제 시 OS Disk도 함께 삭제 처리
	// VM OSDisk 이름 가져오기
	vmInfo, err := vmHandler.GetVM(vmIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	osDiskName := vmInfo.VMBootDisk
/* Detach may not be required for dynamic public IP mode. by powerkim. 2021.04.30.
	// TODO: nested flow 개선
	// VNic에서 PublicIP 연결해제
	vNicDetachStatus, err := DetachVNic(vmHandler, vmInfo)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return vNicDetachStatus, err
	}
*/

	// VM 삭제
	start := call.Start()
	future, err := vmHandler.Client.Delete(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmIID.NameId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	LoggingInfo(hiscallInfo, start)

	// TODO: nested flow 개선
	// VNic 삭제
	vNicStatus, err := DeleteVNic(vmHandler, vmInfo)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return vNicStatus, err
	}

	// TODO: nested flow 개선
	// PublicIP 삭제
	publicIPStatus, err := DeletePublicIP(vmHandler, vmInfo)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return publicIPStatus, err
	}

	// TODO: nested flow 개선
	// OS Disk 삭제
	diskStatus, err := DeleteVMDisk(vmHandler, osDiskName)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return diskStatus, err
	}

	// 자체생성상태 반환
	return irs.NotExist, nil
}

func (vmHandler *AzureVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, VM, "ListVMStatus()")

	start := call.Start()
	serverList, err := vmHandler.Client.List(vmHandler.Ctx, vmHandler.Region.ResourceGroup)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return []*irs.VMStatusInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	var vmStatusList []*irs.VMStatusInfo
	for _, s := range serverList.Values() {
		if s.InstanceView != nil {
			statusStr := getVmStatus(*s.InstanceView)
			status := irs.VMStatus(statusStr)
			vmStatusInfo := irs.VMStatusInfo{
				IId: irs.IID{
					NameId:   *s.Name,
					SystemId: *s.ID,
				},
				VmStatus: status,
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		} else {
			vmIdArr := strings.Split(*s.ID, "/")
			vmName := vmIdArr[8]
			status, _ := vmHandler.GetVMStatus(irs.IID{NameId: vmName, SystemId: *s.ID})
			vmStatusInfo := irs.VMStatusInfo{
				IId: irs.IID{
					NameId:   *s.Name,
					SystemId: *s.ID,
				},
				VmStatus: status,
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		}
	}

	return vmStatusList, nil
}

func (vmHandler *AzureVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "GetVMStatus()")

	start := call.Start()
	instanceView, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmIID.NameId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	LoggingInfo(hiscallInfo, start)

	// Get powerState, provisioningState
	vmStatus := getVmStatus(instanceView)
	return irs.VMStatus(vmStatus), nil
}

func (vmHandler *AzureVMHandler) ListVM() ([]*irs.VMInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, VM, "ListVM()")

	start := call.Start()
	serverList, err := vmHandler.Client.List(vmHandler.Ctx, vmHandler.Region.ResourceGroup)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return []*irs.VMInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	var vmList []*irs.VMInfo
	for _, server := range serverList.Values() {
		vmInfo := vmHandler.mappingServerInfo(server)
		vmList = append(vmList, &vmInfo)
	}

	return vmList, nil
}

func (vmHandler *AzureVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "GetVM()")

	start := call.Start()
	vm, err := vmHandler.Client.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmIID.NameId, compute.InstanceView)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	vmInfo := vmHandler.mappingServerInfo(vm)
	return vmInfo, nil
}

func getVmStatus(instanceView compute.VirtualMachineInstanceView) irs.VMStatus {
	var powerState, provisioningState string

	for _, stat := range *instanceView.Statuses {
		statArr := strings.Split(*stat.Code, "/")

		if statArr[0] == "PowerState" {
			powerState = strings.ToLower(statArr[1])
		} else if statArr[0] == "ProvisioningState" {
			provisioningState = strings.ToLower(statArr[1])
		}
	}

	if strings.EqualFold(provisioningState, "failed") {
		return irs.Failed
	}

	// Set VM Status Info
	var resultStatus string
	switch powerState {
	case "starting":
		resultStatus = "Creating"
	case "running":
		resultStatus = "Running"
	case "stopping":
		resultStatus = "Suspending"
	case "stopped":
		resultStatus = "Suspended"
	case "deleting":
		resultStatus = "Terminating"
	default:
		resultStatus = "Failed"
	}
	return irs.VMStatus(resultStatus)
}

func (vmHandler *AzureVMHandler) mappingServerInfo(server compute.VirtualMachine) irs.VMInfo {

	// Get Default VM Info
	vmInfo := irs.VMInfo{
		IId: irs.IID{
			NameId:   *server.Name,
			SystemId: *server.ID,
		},
		Region: irs.RegionInfo{
			Region: *server.Location,
		},
		VMSpecName: string(server.VirtualMachineProperties.HardwareProfile.VMSize),
	}

	// Set VM Zone
	if server.Zones != nil {
		vmInfo.Region.Zone = (*server.Zones)[0]
	}

	// Set VM Image Info
	if reflect.ValueOf(server.StorageProfile.ImageReference.ID).IsNil() {
		imageRef := server.VirtualMachineProperties.StorageProfile.ImageReference
		vmInfo.ImageIId.SystemId = *imageRef.Publisher + ":" + *imageRef.Offer + ":" + *imageRef.Sku + ":" + *imageRef.Version
		//vmInfo.ImageIId.SystemId = vmInfo.ImageIId.NameId
	} else {
		vmInfo.ImageIId.SystemId = *server.VirtualMachineProperties.StorageProfile.ImageReference.ID
		//vmInfo.ImageIId.SystemId = vmInfo.ImageIId.NameId
	}

	// Get VNic ID
	niList := *server.NetworkProfile.NetworkInterfaces
	var VNicId string
	for _, ni := range niList {
		if ni.ID != nil {
			VNicId = *ni.ID
		}
	}

	// Get VNic
	nicIdArr := strings.Split(VNicId, "/")
	nicName := nicIdArr[len(nicIdArr)-1]
	vNic, _ := vmHandler.NicClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, nicName, "")
	vmInfo.NetworkInterface = nicName

	// Get SecurityGroup
	sgGroupIdArr := strings.Split(*vNic.NetworkSecurityGroup.ID, "/")
	sgGroupName := sgGroupIdArr[len(sgGroupIdArr)-1]
	vmInfo.SecurityGroupIIds = []irs.IID{
		{
			NameId:   sgGroupName,
			SystemId: *vNic.NetworkSecurityGroup.ID,
		},
	}

	// Get PrivateIP, PublicIpId
	for _, ip := range *vNic.IPConfigurations {
		if *ip.Primary {
			// PrivateIP 정보 설정
			vmInfo.PrivateIP = *ip.PrivateIPAddress

			// PublicIP 정보 조회 및 설정
			if ip.PublicIPAddress != nil {
				publicIPId := *ip.PublicIPAddress.ID
				publicIPIdArr := strings.Split(publicIPId, "/")
				publicIPName := publicIPIdArr[len(publicIPIdArr)-1]

				publicIP, _ := vmHandler.PublicIPClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, publicIPName, "")
				vmInfo.PublicIP = *publicIP.IPAddress
			}

			// Get Subnet
			subnetIdArr := strings.Split(*ip.InterfaceIPConfigurationPropertiesFormat.Subnet.ID, "/")
			subnetName := subnetIdArr[len(subnetIdArr)-1]
			vmInfo.SubnetIID = irs.IID{NameId: subnetName, SystemId: *ip.InterfaceIPConfigurationPropertiesFormat.Subnet.ID}

			// Get VPC
			vpcIdArr := subnetIdArr[:len(subnetIdArr)-2]
			vpcName := vpcIdArr[len(vpcIdArr)-1]
			vmInfo.VpcIID = irs.IID{NameId: vpcName, SystemId: strings.Join(vpcIdArr, "/")}
		}
	}

	// Set GuestUser Id/Pwd
	if server.VirtualMachineProperties.OsProfile.AdminUsername != nil {
		vmInfo.VMUserId = *server.VirtualMachineProperties.OsProfile.AdminUsername
	}
	if server.VirtualMachineProperties.OsProfile.AdminPassword != nil {
		vmInfo.VMUserPasswd = *server.VirtualMachineProperties.OsProfile.AdminPassword
	}

	// Set BootDisk
	if server.VirtualMachineProperties.StorageProfile.OsDisk.Name != nil {
		vmInfo.VMBootDisk = *server.VirtualMachineProperties.StorageProfile.OsDisk.Name
	}

	// Get StartTime
	if server.VirtualMachineProperties.InstanceView != nil {
		for _, status := range *server.VirtualMachineProperties.InstanceView.Statuses {
			if strings.EqualFold(*status.Code, ProvisioningStateCode) {
				vmInfo.StartTime = status.Time.Local()
				break
			}
		}
	}

	// Get Keypair
	tagList := server.Tags
	for key, val := range tagList {
		if key == "keypair" {
			vmInfo.KeyPairIId = irs.IID{NameId: *val, SystemId: *val}
		}
		if key == "publicip" {
			vmInfo.KeyValueList = []irs.KeyValue{
				{Key: "publicip", Value: *val},
			}
		}
	}

	return vmInfo
}

// VM 생성 시 Public IP 자동 생성 (nested flow 적용)
func CreatePublicIP(vmHandler *AzureVMHandler, vmReqInfo irs.VMReqInfo) (irs.IID, error) {

	// PublicIP 이름 생성
	/*var publicIPName string
	uuid, err := uuid.NewUUID()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to generate UUID, error=%s", err))
		return irs.IID{}, createErr
	}*/
	//publicIPName = fmt.Sprintf("%s-%s-PublicIP", vmReqInfo.IId.NameId, uuid)
	publicIPName := fmt.Sprintf("%s-PublicIP", vmReqInfo.IId.NameId)

	createOpts := network.PublicIPAddress{
		Name: to.StringPtr(publicIPName),
		Sku: &network.PublicIPAddressSku{
			Name: network.PublicIPAddressSkuName("Basic"),
		},
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAddressVersion:   network.IPVersion("IPv4"),
			//PublicIPAllocationMethod: network.IPAllocationMethod("Static"),
			PublicIPAllocationMethod: network.IPAllocationMethod("Dynamic"),
			IdleTimeoutInMinutes:     to.Int32Ptr(4),
		},
		Location: &vmHandler.Region.Region,
	}

	future, err := vmHandler.PublicIPClient.CreateOrUpdate(vmHandler.Ctx, vmHandler.Region.ResourceGroup, publicIPName, createOpts)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create PublicIP, error=%s", err))
		return irs.IID{}, createErr
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.PublicIPClient.Client)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create PublicIP, error=%s", err))
		return irs.IID{}, createErr
	}

	// 생성된 PublicIP 정보 리턴
	publicIPInfo, err := vmHandler.PublicIPClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, publicIPName, "")
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get PublicIP, error=%s", err))
		return irs.IID{}, getErr
	}
	publicIPIId := irs.IID{NameId: *publicIPInfo.Name, SystemId: *publicIPInfo.ID}
	return publicIPIId, nil
}

// VM 삭제 시 Public IP 자동 삭제 (nested flow 적용)
func DeletePublicIP(vmHandler *AzureVMHandler, vmInfo irs.VMInfo) (irs.VMStatus, error) {
	var publicIPId string
	for _, keyInfo := range vmInfo.KeyValueList {
		if keyInfo.Key == "publicip" {
			publicIPId = keyInfo.Value
			break
		}
	}

	publicIPFuture, err := vmHandler.PublicIPClient.Delete(vmHandler.Ctx, vmHandler.Region.ResourceGroup, publicIPId)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	err = publicIPFuture.WaitForCompletionRef(vmHandler.Ctx, vmHandler.PublicIPClient.Client)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	return irs.Terminating, nil
}

// VM 생성 시 VNic 자동 생성 (nested flow 적용)
func CreateVNic(vmHandler *AzureVMHandler, vmReqInfo irs.VMReqInfo, publicIPIId irs.IID) (irs.IID, error) {

	// VNic 이름 생성
	/*var VNicName string
	uuid, err := uuid.NewUUID()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to generate UUID, error=%s", err))
		return irs.IID{}, createErr
	}*/
	//VNicName = fmt.Sprintf("%s-%s-VNic", vmReqInfo.IId.NameId, uuid)
	VNicName := fmt.Sprintf("%s-VNic", vmReqInfo.IId.NameId)

	// 리소스 Id 정보 매핑
	// Azure의 경우 VNic에 1개의 보안그룹만 할당 가능
	secGroupId := GetSecGroupIdByName(vmHandler.CredentialInfo, vmHandler.Region, vmReqInfo.SecurityGroupIIDs[0].NameId)
	subnet, err := vmHandler.SubnetClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmReqInfo.VpcIID.NameId, vmReqInfo.SubnetIID.NameId, "")

	var ipConfigArr []network.InterfaceIPConfiguration
	ipConfig := network.InterfaceIPConfiguration{
		Name: to.StringPtr("ipConfig1"),
		InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
			Subnet:                    &subnet,
			PrivateIPAllocationMethod: "Dynamic",
			PublicIPAddress: &network.PublicIPAddress{
				ID: to.StringPtr(publicIPIId.SystemId),
			},
		},
	}
	ipConfigArr = append(ipConfigArr, ipConfig)

	/*
	 test VM is interfacingProperties
	*/

	createOpts := network.Interface{
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &ipConfigArr,
			NetworkSecurityGroup: &network.SecurityGroup{
				ID: to.StringPtr(secGroupId),
			},
		},
		Location: &vmHandler.Region.Region,
	}

	future, err := vmHandler.NicClient.CreateOrUpdate(vmHandler.Ctx, vmHandler.Region.ResourceGroup, VNicName, createOpts)
	if err != nil {
		return irs.IID{}, err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.NicClient.Client)
	if err != nil {
		return irs.IID{}, err
	}

	// 생성된 VNic 정보 리턴
	VNic, err := vmHandler.NicClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, VNicName, "")
	if err != nil {
		return irs.IID{}, err
	}
	VNicIId := irs.IID{NameId: *VNic.Name, SystemId: *VNic.ID}
	return VNicIId, nil
}

// VNic 삭제 전 PublicIP 연결 해제
func DetachVNic(vmHandler *AzureVMHandler, vmInfo irs.VMInfo) (irs.VMStatus, error) {
	var ipConfigArr []network.InterfaceIPConfiguration
	ipConfig := network.InterfaceIPConfiguration{
		Name: to.StringPtr("ipConfig1"),
		InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
			Subnet: &network.Subnet{
				Name: to.StringPtr(vmInfo.SubnetIID.NameId),
				ID:   to.StringPtr(vmInfo.SubnetIID.SystemId),
			},
			PrivateIPAllocationMethod: "Dynamic",
			PublicIPAddress:           nil,
		},
	}
	ipConfigArr = append(ipConfigArr, ipConfig)

	detachOpts := network.Interface{
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &ipConfigArr,
		},
		Location: &vmHandler.Region.Region,
	}

	nicDetachFuture, err := vmHandler.NicClient.CreateOrUpdate(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmInfo.NetworkInterface, detachOpts)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	err = nicDetachFuture.WaitForCompletionRef(vmHandler.Ctx, vmHandler.NicClient.Client)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	return irs.Terminating, nil
}

// VM 삭제 시 VNic 자동 삭제 (nested flow 적용)
func DeleteVNic(vmHandler *AzureVMHandler, vmInfo irs.VMInfo) (irs.VMStatus, error) {
	nicDeleteFuture, err := vmHandler.NicClient.Delete(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmInfo.NetworkInterface)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	err = nicDeleteFuture.WaitForCompletionRef(vmHandler.Ctx, vmHandler.NicClient.Client)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	return irs.Terminating, nil
}

// VM 삭제 시 VM Disk 자동 삭제 (nested flow 적용)
func DeleteVMDisk(vmHandler *AzureVMHandler, osDiskName string) (irs.VMStatus, error) {
	diskFuture, err := vmHandler.DiskClient.Delete(vmHandler.Ctx, vmHandler.Region.ResourceGroup, osDiskName)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	err = diskFuture.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	return irs.Terminating, nil
}
