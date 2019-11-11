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
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	"reflect"
	"strings"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

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
	// Check VM Exists
	vm, err := vmHandler.Client.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmReqInfo.VMName, compute.InstanceView)
	if vm.ID != nil {
		errMsg := fmt.Sprintf("VirtualMachine with name %s already exist", vmReqInfo.VMName)
		createErr := errors.New(errMsg)
		return irs.VMInfo{}, createErr
	}

	// Check login method (keypair, password)
	if vmReqInfo.VMUserPasswd != "" && vmReqInfo.KeyPairName != "" {
		createErr := errors.New("Specifiy one login method, Password or Keypair")
		return irs.VMInfo{}, createErr
	}

	// (old) 리소스 Id 정보 매핑
	//vNicId := GetVNicIdByName(vmHandler.CredentialInfo, vmHandler.Region, vmReqInfo.NetworkInterfaceId)

	// VNic 생성
	vNicId, err := CreateVNic(vmHandler, vmReqInfo)
	if err != nil {
		return irs.VMInfo{}, err
	}

	vmOpts := compute.VirtualMachine{
		Location: &vmHandler.Region.Region,
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(vmReqInfo.VMSpecId),
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  &vmReqInfo.VMName,
				AdminUsername: to.StringPtr(CBVMUser),
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					{
						//ID: &vmReqInfo.NetworkInterfaceId,
						ID: vNicId,
						NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
							Primary: to.BoolPtr(true),
						},
					},
				},
			},
		},
	}

	// Image 설정
	if strings.Contains(vmReqInfo.ImageId, ":") {
		imageArr := strings.Split(vmReqInfo.ImageId, ":")
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
				ID: &vmReqInfo.ImageId,
			},
		}
	}

	// KeyPair 설정
	if vmReqInfo.KeyPairName != "" {
		publicKey, err := GetPublicKey(vmHandler.CredentialInfo, vmReqInfo.KeyPairName)
		if err != nil {
			cblogger.Error(err)
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
	/*vmOpts.Tags = map[string]*string{
		"vmName": to.StringPtr(vmReqInfo.VMName),
	}*/
	if vmReqInfo.KeyPairName != "" {
		vmOpts.Tags = map[string]*string{
			"keypair": to.StringPtr(vmReqInfo.KeyPairName),
		}
	}

	future, err := vmHandler.Client.CreateOrUpdate(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmReqInfo.VMName, vmOpts)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}

	vm, err = vmHandler.Client.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmReqInfo.VMName, compute.InstanceView)
	if err != nil {
		cblogger.Error(err)
	}
	vmInfo := vmHandler.mappingServerInfo(vm)

	return vmInfo, nil
}

func (vmHandler *AzureVMHandler) SuspendVM(vmID string) (irs.VMStatus, error) {
	future, err := vmHandler.Client.PowerOff(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// Get VM Status
	vmStatus, err := vmHandler.GetVMStatus(vmID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	return vmStatus, nil
}

func (vmHandler *AzureVMHandler) ResumeVM(vmID string) (irs.VMStatus, error) {
	future, err := vmHandler.Client.Start(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// 자체생성상태 반환
	return irs.Resuming, nil
}

func (vmHandler *AzureVMHandler) RebootVM(vmID string) (irs.VMStatus, error) {
	future, err := vmHandler.Client.Restart(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// 자체생성상태 반환
	return irs.Rebooting, nil
}

func (vmHandler *AzureVMHandler) TerminateVM(vmID string) (irs.VMStatus, error) {

	// VM 삭제 시 OS Disk도 함께 삭제 처리
	// VM OSDisk 이름 가져오기
	vmInfo, err := vmHandler.GetVM(vmID)
	if err != nil {
		return irs.Failed, err
	}
	osDiskName := vmInfo.VMBootDisk

	// VM 삭제
	future, err := vmHandler.Client.Delete(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// vNic 삭제
	nicFuture, err := vmHandler.NicClient.Delete(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmInfo.Name+"-NIC")
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	err = nicFuture.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// OS Disk 삭제
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

	// 자체생성상태 반환
	return irs.NotExist, nil
}

func (vmHandler *AzureVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	serverList, err := vmHandler.Client.List(vmHandler.Ctx, vmHandler.Region.ResourceGroup)
	if err != nil {
		cblogger.Error(err)
		return []*irs.VMStatusInfo{}, err
	}

	var vmStatusList []*irs.VMStatusInfo
	for _, s := range serverList.Values() {
		if s.InstanceView != nil {
			statusStr := getVmStatus(*s.InstanceView)
			status := irs.VMStatus(statusStr)
			vmStatusInfo := irs.VMStatusInfo{
				VmId:     *s.ID,
				VmStatus: status,
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		} else {
			vmIdArr := strings.Split(*s.ID, "/")
			vmName := vmIdArr[8]
			status, _ := vmHandler.GetVMStatus(vmName)
			vmStatusInfo := irs.VMStatusInfo{
				VmId:     *s.ID,
				VmStatus: status,
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		}
	}

	return vmStatusList, nil
}

func (vmHandler *AzureVMHandler) GetVMStatus(vmID string) (irs.VMStatus, error) {
	instanceView, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// Get powerState, provisioningState
	vmStatus := getVmStatus(instanceView)
	return irs.VMStatus(vmStatus), nil
}

func (vmHandler *AzureVMHandler) ListVM() ([]*irs.VMInfo, error) {
	//serverList, err := vmHandler.Client.ListAll(vmHandler.Ctx)
	serverList, err := vmHandler.Client.List(vmHandler.Ctx, vmHandler.Region.ResourceGroup)
	if err != nil {
		cblogger.Error(err)
		return []*irs.VMInfo{}, err
	}

	var vmList []*irs.VMInfo
	for _, server := range serverList.Values() {
		vmInfo := vmHandler.mappingServerInfo(server)
		vmList = append(vmList, &vmInfo)
	}

	return vmList, nil
}

func (vmHandler *AzureVMHandler) GetVM(vmID string) (irs.VMInfo, error) {
	vm, err := vmHandler.Client.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmID, compute.InstanceView)
	if err != nil {
		return irs.VMInfo{}, err
	}

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
		Name: *server.Name,
		Id:   *server.ID,
		Region: irs.RegionInfo{
			Region: *server.Location,
		},
		VMSpecId: string(server.VirtualMachineProperties.HardwareProfile.VMSize),
	}

	// Set VM Zone
	if server.Zones != nil {
		vmInfo.Region.Zone = (*server.Zones)[0]
	}

	// Set VM Image Info
	if reflect.ValueOf(server.StorageProfile.ImageReference.ID).IsNil() {
		imageRef := server.VirtualMachineProperties.StorageProfile.ImageReference
		vmInfo.ImageId = *imageRef.Publisher + ":" + *imageRef.Offer + ":" + *imageRef.Sku + ":" + *imageRef.Version
	} else {
		vmInfo.ImageId = *server.VirtualMachineProperties.StorageProfile.ImageReference.ID
	}

	// Set VNic Info
	niList := *server.NetworkProfile.NetworkInterfaces
	for _, ni := range niList {
		if ni.ID != nil {
			vmInfo.NetworkInterfaceId = *ni.ID
		}
	}

	// Get VNic
	nicIdArr := strings.Split(vmInfo.NetworkInterfaceId, "/")
	nicName := nicIdArr[len(nicIdArr)-1]
	vNic, _ := vmHandler.NicClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, nicName, "")

	vmInfo.SecurityGroupIds = []string{*vNic.NetworkSecurityGroup.ID}

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

			// Subnet 정보 설정
			vmInfo.VirtualNetworkId = *ip.InterfaceIPConfigurationPropertiesFormat.Subnet.ID
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

	// Get Keypair
	tagList := server.Tags
	for key, val := range tagList {
		if key == "keypair" {
			vmInfo.KeyPairName = *val
		}
	}

	return vmInfo
}

// VM 생성 시 VNic 자동 생성
func CreateVNic(vmHandler *AzureVMHandler, vmReqInfo irs.VMReqInfo) (*string, error) {
	vNicName := vmReqInfo.VMName + "-NIC"

	// Check VNic Exists
	vNic, _ := vmHandler.NicClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vNicName, "")
	if vNic.ID != nil {
		errMsg := fmt.Sprintf("Virtual Network Interface with name %s already exist", vNicName)
		createErr := errors.New(errMsg)
		return nil, createErr
	}

	// 리소스 Id 정보 매핑
	secGroupId := GetSecGroupIdByName(vmHandler.CredentialInfo, vmHandler.Region, vmReqInfo.SecurityGroupIds[0])
	publicIPId := GetPublicIPIdByName(vmHandler.CredentialInfo, vmHandler.Region, vmReqInfo.PublicIPId)

	subnet, err := vmHandler.SubnetClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, CBVirutalNetworkName, vmReqInfo.VirtualNetworkId, "")

	var ipConfigArr []network.InterfaceIPConfiguration
	ipConfig := network.InterfaceIPConfiguration{
		Name: to.StringPtr("ipConfig1"),
		InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
			Subnet:                    &subnet,
			PrivateIPAllocationMethod: "Dynamic",
		},
	}
	if publicIPId != "" {
		ipConfig.PublicIPAddress = &network.PublicIPAddress{
			ID: to.StringPtr(publicIPId),
		}
	}
	ipConfigArr = append(ipConfigArr, ipConfig)

	createOpts := network.Interface{
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &ipConfigArr,
		},
		Location: &vmHandler.Region.Region,
	}

	if len(vmReqInfo.SecurityGroupIds) != 0 {
		createOpts.NetworkSecurityGroup = &network.SecurityGroup{
			ID: to.StringPtr(secGroupId),
		}
	}

	future, err := vmHandler.NicClient.CreateOrUpdate(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vNicName, createOpts)
	if err != nil {
		return nil, err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.NicClient.Client)
	if err != nil {
		return nil, err
	}

	vNic, err = vmHandler.NicClient.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vNicName, "")
	if err != nil {
		return nil, err
	}

	return vNic.ID, nil
}
