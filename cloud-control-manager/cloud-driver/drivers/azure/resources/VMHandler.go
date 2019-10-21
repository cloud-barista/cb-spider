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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type AzureVMHandler struct {
	Region idrv.RegionInfo
	Ctx    context.Context
	Client *compute.VirtualMachinesClient
}

func (vmHandler *AzureVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	// Set VM Create Information
	imageId := vmReqInfo.ImageId
	imageIdArr := strings.Split(imageId, ":")

	// TODO: golang.org/x/crypto/ssh lib 기반 키 생성 기능 개발
	sshKeyData, err := generateSSHKey("mcb-key")
	if err != nil {
		return irs.VMInfo{}, err
	}

	vmName := vmReqInfo.VMName
	vmNameArr := strings.Split(vmName, ":")

	// Check VM Exists
	vm, err := vmHandler.Client.Get(vmHandler.Ctx, vmNameArr[0], vmNameArr[1], compute.InstanceView)
	if vm.ID != nil {
		errMsg := fmt.Sprintf("VirtualMachine with name %s already exist", vmNameArr[1])
		createErr := errors.New(errMsg)
		return irs.VMInfo{}, createErr
	}

	vmOpts := compute.VirtualMachine{
		Location: &vmHandler.Region.Region,
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(vmReqInfo.VMSpecId),
			},
			StorageProfile: &compute.StorageProfile{
				ImageReference: &compute.ImageReference{
					Publisher: &imageIdArr[0],
					Offer:     &imageIdArr[1],
					Sku:       &imageIdArr[2],
					Version:   &imageIdArr[3],
				},
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  &vmNameArr[1],
				AdminUsername: &vmReqInfo.VMUserId,
				//AdminPassword: &vmReqInfo.VMUserPasswd,
				LinuxConfiguration: &compute.LinuxConfiguration{
					SSH: &compute.SSHConfiguration{
						PublicKeys: &[]compute.SSHPublicKey{
							{
								Path: to.StringPtr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", vmReqInfo.VMUserId)),
								//KeyData: &sshKeyData,
								// TODO: golang.org/x/crypto/ssh lib 기반 키 생성 기능 개발
								KeyData: &sshKeyData,
							},
						},
					},
				},
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					{
						ID: &vmReqInfo.VirtualNetworkId,
						NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
							Primary: to.BoolPtr(true),
						},
					},
				},
			},
		},
	}

	future, err := vmHandler.Client.CreateOrUpdate(vmHandler.Ctx, vmNameArr[0], vmNameArr[1], vmOpts)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}

	vm, err = vmHandler.Client.Get(vmHandler.Ctx, vmNameArr[0], vmNameArr[1], compute.InstanceView)
	if err != nil {
		cblogger.Error(err)
	}
	vmInfo := mappingServerInfo(vm)

	return vmInfo, nil
}

func (vmHandler *AzureVMHandler) SuspendVM(vmID string) error {
	vmIdArr := strings.Split(vmID, ":")

	future, err := vmHandler.Client.PowerOff(vmHandler.Ctx, vmIdArr[0], vmIdArr[1])
	if err != nil {
		cblogger.Error(err)
		return err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return err
	}
	return nil
}

func (vmHandler *AzureVMHandler) ResumeVM(vmID string) error {
	vmIdArr := strings.Split(vmID, ":")

	future, err := vmHandler.Client.Start(vmHandler.Ctx, vmIdArr[0], vmIdArr[1])
	if err != nil {
		cblogger.Error(err)
		return err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return err
	}
	return nil
}

func (vmHandler *AzureVMHandler) RebootVM(vmID string) error {
	vmIdArr := strings.Split(vmID, ":")

	future, err := vmHandler.Client.Restart(vmHandler.Ctx, vmIdArr[0], vmIdArr[1])
	if err != nil {
		cblogger.Error(err)
		return err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return err
	}
	return nil
}

func (vmHandler *AzureVMHandler) TerminateVM(vmID string) error {
	vmIdArr := strings.Split(vmID, ":")

	future, err := vmHandler.Client.Delete(vmHandler.Ctx, vmIdArr[0], vmIdArr[1])
	//future, err := vmHandler.Client.Deallocate(vmHandler.Ctx, vmIdArr[0], vmIdArr[1])
	if err != nil {
		cblogger.Error(err)
		return err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return err
	}
	return nil
}

func (vmHandler *AzureVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	//serverList, err := vmHandler.Client.ListAll(vmHandler.Ctx)
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
			vmId := vmIdArr[4] + ":" + vmIdArr[8]
			status, _ := vmHandler.GetVMStatus(vmId)
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
	vmIdArr := strings.Split(vmID, ":")
	instanceView, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmIdArr[0], vmIdArr[1])
	if err != nil {
		cblogger.Error(err)
		return "", err
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
		vmInfo := mappingServerInfo(server)
		vmList = append(vmList, &vmInfo)
	}

	return vmList, nil
}

func (vmHandler *AzureVMHandler) GetVM(vmID string) (irs.VMInfo, error) {
	vmIdArr := strings.Split(vmID, ":")
	vm, err := vmHandler.Client.Get(vmHandler.Ctx, vmIdArr[0], vmIdArr[1], compute.InstanceView)
	if err != nil {
		return irs.VMInfo{}, err
	}

	vmInfo := mappingServerInfo(vm)
	return vmInfo, nil
}

func getVmStatus(instanceView compute.VirtualMachineInstanceView) string {
	var powerState, provisioningState string

	for _, stat := range *instanceView.Statuses {
		statArr := strings.Split(*stat.Code, "/")

		if statArr[0] == "PowerState" {
			powerState = statArr[1]
		} else if statArr[0] == "ProvisioningState" {
			provisioningState = statArr[1]
		}
	}

	// Set VM Status Info
	var vmState string
	if powerState != "" && provisioningState != "" {
		vmState = powerState + "(" + provisioningState + ")"
	} else if powerState != "" && provisioningState == "" {
		vmState = powerState
	} else if powerState == "" && provisioningState != "" {
		vmState = provisioningState
	} else {
		vmState = "-"
	}
	return vmState
}

func mappingServerInfo(server compute.VirtualMachine) irs.VMInfo {

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
	imageRef := server.VirtualMachineProperties.StorageProfile.ImageReference
	imageId := *imageRef.Publisher + ":" + *imageRef.Offer + ":" + *imageRef.Sku + ":" + *imageRef.Version
	vmInfo.ImageId = imageId

	// Set VNic Info
	niList := *server.NetworkProfile.NetworkInterfaces
	for _, ni := range niList {
		if ni.NetworkInterfaceReferenceProperties != nil {
			vmInfo.VirtualNetworkId = *ni.ID
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

	return vmInfo
}

func generateSSHKey(keyName string) (string, error) {

	// TODO: ENV 환경변수 PATH에 키 저장
	rootPath := os.Getenv("CBSPIDER_PATH")
	savePrivateFileTo := rootPath + "/conf/PrivateKey"
	savePublicFileTo := rootPath + "/conf/PublicKey"
	bitSize := 4096

	// 지정된 바이트크기의 RSA 형식 개인키(비공개키)를 만듬
	privateKey, err := generatePrivateKey(bitSize)
	if err != nil {
		//log.Fatal(err.Error())
	}

	// 개인키를 RSA에서 PEM 형식으로 인코딩
	privateKeyBytes := encodePrivateKeyToPEM(privateKey)

	// rsa.PublicKey를 가져와서 .pub 파일에 쓰기 적합한 바이트로 변환
	// "ssh-rsa ..."형식으로 변환
	publicKeyBytes, err := generatePublicKey(&privateKey.PublicKey)
	if err != nil {
		//log.Fatal(err.Error())
	}

	// 파일에 private Key를 쓴다
	err = writeKeyToFile(privateKeyBytes, savePrivateFileTo)
	if err != nil {
		//log.Fatal(err.Error())
	}

	// 파일에 public Key를 쓴다
	err = writeKeyToFile([]byte(publicKeyBytes), savePublicFileTo)
	if err != nil {
		//log.Fatal(err.Error())
	}

	// TODO: 파일 bytes로 읽어들여서 string으로 변환
	var pubKeyStr string

	data, err := ioutil.ReadFile(savePublicFileTo)
	if err != nil {

	}
	pubKeyStr = string(data)

	return pubKeyStr, nil
}

// 지정된 바이트크기의 RSA 형식 개인키(비공개키)를 만듬
func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	// Private Key 생성
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	// Private Key 확인
	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	log.Println("Private Key generated(생성)")
	fmt.Println(privateKey)
	return privateKey, nil
}

// 개인키를 RSA에서 PEM 형식으로 인코딩
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)
	fmt.Println("privateKey Rsa -> Pem 형식으로 변환")
	fmt.Println(privatePEM)
	return privatePEM
}

// rsa.PublicKey를 가져와서 .pub 파일에 쓰기 적합한 바이트로 변환
// "ssh-rsa ..."형식으로 변환
func generatePublicKey(privatekey *rsa.PublicKey) ([]byte, error) {
	publicRsaKey, err := ssh.NewPublicKey(privatekey)
	if err != nil {
		return nil, err
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	log.Println("Public key 생성")
	fmt.Println(pubKeyBytes)
	return pubKeyBytes, nil
}

// 파일에 Key를 쓴다
func writeKeyToFile(keyBytes []byte, saveFileTo string) error {
	err := ioutil.WriteFile(saveFileTo, keyBytes, 0600)
	if err != nil {
		return err
	}

	log.Printf("Key 저장위치: %s", saveFileTo)
	return nil
}
