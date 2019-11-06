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
	_ "errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	compute "google.golang.org/api/compute/v1"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type GCPVMHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

func (vmHandler *GCPVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	// Set VM Create Information
	// GCP 는 reqinfo에 ProjectID를 받아야 함.

	ctx := vmHandler.Ctx
	vmName := vmReqInfo.VMName
	projectID := vmHandler.Credential.ProjectID
	prefix := "https://www.googleapis.com/compute/v1/projects/" + projectID
	imageURL := "projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20191024"
	zone := vmHandler.Region.Zone
	// email을 어디다가 넣지? 이것또한 문제넹
	clientEmail := vmHandler.Credential.ClientEmail

	cblogger.Info("PublicIp 생성 시작")
	// PublicIPHandler  불러서 처리 해야 함.
	publicIpHandler := GCPPublicIPHandler{
		vmHandler.Region, vmHandler.Ctx, vmHandler.Client, vmHandler.Credential}
	publicIpName := vmReqInfo.PublicIPId
	publicIpReqInfo := irs.PublicIPReqInfo{Name: publicIpName}
	publicIPInfo, err := publicIpHandler.CreatePublicIP(publicIpReqInfo)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}
	cblogger.Info("PublicIp 생성됨")
	cblogger.Info(publicIPInfo)

	networkURL := prefix + "/global/networks/" + vmReqInfo.VirtualNetworkId
	publicIPAddress := publicIPInfo.PublicIP

	instance := &compute.Instance{
		Name:        vmName,
		Description: "compute sample instance",
		MachineType: prefix + "/zones/" + zone + "/machineTypes/" + vmReqInfo.VMSpecId,
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					DiskName:    vmName + "-" + zone, //disk name 도 매번 바뀌어야 하는 값
					SourceImage: imageURL,
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				AccessConfigs: []*compute.AccessConfig{
					{
						Type:  "ONE_TO_ONE_NAT",
						Name:  "External NAT", // default
						NatIP: publicIPAddress,
					},
				},
				Network: networkURL,
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: clientEmail,
				Scopes: []string{
					compute.DevstorageFullControlScope,
					compute.ComputeScope,
				},
			},
		},
	}

	cblogger.Info("VM 생성 시작")
	cblogger.Info(instance)
	op, err := vmHandler.Client.Instances.Insert(projectID, zone, instance).Do()
	js, err := op.MarshalJSON()
	if err != nil {
		cblogger.Info("VM 생성 실패")
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}

	cblogger.Info("Insert vm to marshal Json : ", string(js))
	cblogger.Infof("Got compute.Operation, err: %#v, %v", op, err)

	// 이게 시작하는  api Start 내부 매개변수로 projectID, zone, InstanceID
	//vm, err := vmHandler.Client.Instances.Start(project string, zone string, instance string)
	time.Sleep(time.Second * 10)
	vm, err := vmHandler.Client.Instances.Get(projectID, zone, vmName).Context(ctx).Do()
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}
	vmInfo := mappingServerInfo(vm)

	return vmInfo, nil
}

// stop이라고 보면 될듯
func (vmHandler *GCPVMHandler) SuspendVM(vmID string) error {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone
	ctx := vmHandler.Ctx

	inst, err := vmHandler.Client.Instances.Stop(projectID, zone, vmID).Context(ctx).Do()
	if err != nil {
		cblogger.Error(err)
		return err
	}

	fmt.Println("instance stop status :", inst.Status)
	return nil
}

func (vmHandler *GCPVMHandler) ResumeVM(vmID string) error {

	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone
	ctx := vmHandler.Ctx

	inst, err := vmHandler.Client.Instances.Start(projectID, zone, vmID).Context(ctx).Do()
	if err != nil {
		cblogger.Error(err)
		return err
	}

	fmt.Println("instance resume status :", inst.Status)
	return nil
}

func (vmHandler *GCPVMHandler) RebootVM(vmID string) error {

	err := vmHandler.SuspendVM(vmID)
	if err != nil {
		return err
	}

	err2 := vmHandler.ResumeVM(vmID)
	if err2 != nil {
		return err2
	}

	return nil
}

func (vmHandler *GCPVMHandler) TerminateVM(vmID string) error {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone
	ctx := vmHandler.Ctx

	inst, err := vmHandler.Client.Instances.Delete(projectID, zone, vmID).Context(ctx).Do()
	if err != nil {
		cblogger.Error(err)
		return err
	}

	fmt.Println("instance status :", inst.Status)

	return nil
}

func (vmHandler *GCPVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	//serverList, err := vmHandler.Client.ListAll(vmHandler.Ctx)
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone

	serverList, err := vmHandler.Client.Instances.List(projectID, zone).Do()
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	var vmStatusList []*irs.VMStatusInfo
	for _, s := range serverList.Items {
		if s.Name != "" {
			vmId := s.Name
			status, _ := vmHandler.GetVMStatus(vmId)
			vmStatusInfo := irs.VMStatusInfo{
				VmId:     vmId,
				VmStatus: status,
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		}
	}

	return vmStatusList, nil
}

func ConvertVMStatusString(vmStatus string) (irs.VMStatus, error) {
	var resultStatus string
	cblogger.Infof("vmStatus : [%s]", vmStatus)

	if strings.EqualFold(vmStatus, "PROVISIONING") {
		resultStatus = "Creating"
		//resultStatus = "Resuming" // Resume 요청을 받아서 재기동되는 단계에도 Pending이 있기 때문에 Pending은 Resuming으로 맵핑함.
	} else if strings.EqualFold(vmStatus, "RUNNING") {
		resultStatus = "Running"
	} else if strings.EqualFold(vmStatus, "STOPPING") {
		resultStatus = "Suspending"
	} else if strings.EqualFold(vmStatus, "Terminated") {
		resultStatus = "Suspended"
	} else if strings.EqualFold(vmStatus, "STAGING") {
		resultStatus = "Resuming"
	} else {
		//resultStatus = "Failed"
		cblogger.Errorf("vmStatus [%s]와 일치하는 맵핑 정보를 찾지 못 함.", vmStatus)
		return irs.VMStatus("Failed"), errors.New(vmStatus + "와 일치하는 CB VM 상태정보를 찾을 수 없습니다.")
	}
	cblogger.Infof("VM 상태 치환 : [%s] ==> [%s]", vmStatus, resultStatus)
	return irs.VMStatus(resultStatus), nil
}

func (vmHandler *GCPVMHandler) GetVMStatus(vmID string) (irs.VMStatus, error) { // GCP의 ID는 uint64 이므로 GCP에서는 Name을 ID값으로 사용한다.
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone

	instanceView, err := vmHandler.Client.Instances.Get(projectID, zone, vmID).Do()
	if err != nil {
		cblogger.Error(err)
		return irs.VMStatus("Failed"), err
	}

	// Get powerState, provisioningState
	//vmStatus := instanceView.Status
	vmStatus, errStatus := ConvertVMStatusString(instanceView.Status)
	//return irs.VMStatus(vmStatus), err
	return vmStatus, errStatus
}

func (vmHandler *GCPVMHandler) ListVM() ([]*irs.VMInfo, error) {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone

	serverList, err := vmHandler.Client.Instances.List(projectID, zone).Do()
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	var vmList []*irs.VMInfo
	for _, server := range serverList.Items {
		vmInfo := mappingServerInfo(server)
		vmList = append(vmList, &vmInfo)
	}

	return vmList, nil
}

func (vmHandler *GCPVMHandler) GetVM(vmName string) (irs.VMInfo, error) {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone

	vm, err := vmHandler.Client.Instances.Get(projectID, zone, vmName).Do()
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}

	vmInfo := mappingServerInfo(vm)
	return vmInfo, nil
}

// func getVmStatus(vl *compute.Service) string {
// 	var powerState, provisioningState string

// 	for _, stat := range vl {
// 		statArr := strings.Split(*stat.Code, "/")

// 		if statArr[0] == "PowerState" {
// 			powerState = statArr[1]
// 		} else if statArr[0] == "ProvisioningState" {
// 			provisioningState = statArr[1]
// 		}
// 	}

// 	// Set VM Status Info
// 	var vmState string
// 	if powerState != "" && provisioningState != "" {
// 		vmState = powerState + "(" + provisioningState + ")"
// 	} else if powerState != "" && provisioningState == "" {
// 		vmState = powerState
// 	} else if powerState == "" && provisioningState != "" {
// 		vmState = provisioningState
// 	} else {
// 		vmState = "-"
// 	}
// 	return vmState
// }

func mappingServerInfo(server *compute.Instance) irs.VMInfo {

	// Get Default VM Info
	vmInfo := irs.VMInfo{
		Name: server.Name,
		Id:   strconv.FormatUint(server.Id, 10),
		Region: irs.RegionInfo{
			Zone: server.Zone,
		},
		NetworkInterfaceId: server.NetworkInterfaces[0].Name,
		VMSpecId:           server.MachineType,
		PublicIP:           server.NetworkInterfaces[0].AccessConfigs[0].NatIP,
		PrivateIP:          server.NetworkInterfaces[0].NetworkIP,
		VirtualNetworkId:   server.NetworkInterfaces[0].Network,
		// SubNetworkID:       server.NetworkInterfaces[0].Subnetwork,
		KeyValueList: []irs.KeyValue{
			{"SubNetwork", server.NetworkInterfaces[0].Subnetwork},
			{"AccessConfigName", server.NetworkInterfaces[0].AccessConfigs[0].Name},
			{"NetworkTier", server.NetworkInterfaces[0].AccessConfigs[0].NetworkTier},
			{"DiskDeviceName", server.Disks[0].DeviceName},
			{"DiskName", server.Disks[0].Source},
		},
	}

	return vmInfo
}
