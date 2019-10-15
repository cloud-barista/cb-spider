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
	_ "errors"
	"fmt"
	"log"

	compute "google.golang.org/api/compute/v1"

	idrv "../../../interfaces"
	irs "../../../interfaces/resources"
	_ "github.com/Azure/go-autorest/autorest/to"
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
	vmName := vmReqInfo.Name
	projectID := vmHandler.Credential.ProjectID
	prefix := "https://www.googleapis.com/compute/v1/projects/" + projectID
	imageURL := "https://www.googleapis.com/compute/v1/projects/debian-cloud/global/images/debian-7-wheezy-v20140606"
	zone := vmHandler.Region.Zone
	// email을 어디다가 넣지? 이것또한 문제넹
	clientEmail := vmHandler.Credential.ClientEmail
	// instanceName := "cscmcloud"

	instance := &compute.Instance{
		Name:        vmName,
		Description: "compute sample instance",
		MachineType: prefix + "/zones/" + zone + "/machineTypes/n1-standard-1",
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					DiskName:    "my-root-pd", //disk name 도 매번 바뀌어야 하는 값
					SourceImage: imageURL,
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				AccessConfigs: []*compute.AccessConfig{
					{
						Type: "ONE_TO_ONE_NAT",
						Name: "External NAT",
					},
				},
				Network: prefix + "/global/networks/default",
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

	op, err := vmHandler.Client.Instances.Insert(projectID, zone, instance).Do()
	js, err := op.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Insert vm to marshal Json : ", string(js))
	log.Printf("Got compute.Operation, err: %#v, %v", op, err)

	// 이게 시작하는  api Start 내부 매개변수로 projectID, zone, InstanceID
	//vm, err := vmHandler.Client.Instances.Start(project string, zone string, instance string)
	vm, err := vmHandler.Client.Instances.Get(projectID, zone, vmName).Context(ctx).Do()
	if err != nil {
		panic(err)
	}
	vmInfo := mappingServerInfo(vm)

	return vmInfo, nil
}

// stop이라고 보면 될듯
func (vmHandler *GCPVMHandler) SuspendVM(vmID string) {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone
	ctx := vmHandler.Ctx

	inst, err := vmHandler.Client.Instances.Stop(projectID, zone, vmID).Context(ctx).Do()

	if err != nil {
		panic(err)
	}

	fmt.Println("instance stop status :", inst.Status)
}

func (vmHandler *GCPVMHandler) ResumeVM(vmID string) {

	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone
	ctx := vmHandler.Ctx

	inst, err := vmHandler.Client.Instances.Start(projectID, zone, vmID).Context(ctx).Do()

	if err != nil {
		panic(err)
	}

	fmt.Println("instance resume status :", inst.Status)

}

func (vmHandler *GCPVMHandler) RebootVM(vmID string) {

	vmHandler.SuspendVM(vmID)
	vmHandler.ResumeVM(vmID)
}

func (vmHandler *GCPVMHandler) TerminateVM(vmID string) {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone
	ctx := vmHandler.Ctx

	inst, err := vmHandler.Client.Instances.Delete(projectID, zone, vmID).Context(ctx).Do()

	if err != nil {
		panic(err)
	}

	fmt.Println("instance status :", inst.Status)
}

func (vmHandler *GCPVMHandler) ListVMStatus() []*irs.VMStatusInfo {
	//serverList, err := vmHandler.Client.ListAll(vmHandler.Ctx)
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone

	serverList, err := vmHandler.Client.Instances.List(projectID, zone).Do()
	if err != nil {
		panic(err)
	}

	var vmStatusList []*irs.VMStatusInfo
	for _, s := range serverList.Items {
		if s.Name != "" {
			vmId := s.Name
			status := vmHandler.GetVMStatus(vmId)
			vmStatusInfo := irs.VMStatusInfo{
				VmId:     vmId,
				VmStatus: status,
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		}
	}

	return vmStatusList
}

func (vmHandler *GCPVMHandler) GetVMStatus(vmID string) irs.VMStatus { // GCP의 ID는 uint64 이므로 GCP에서는 Name을 ID값으로 사용한다.
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone

	instanceView, err := vmHandler.Client.Instances.Get(projectID, zone, vmID).Do()
	if err != nil {
		panic(err)
	}

	// Get powerState, provisioningState
	vmStatus := instanceView.Status
	return irs.VMStatus(vmStatus)
}

func (vmHandler *GCPVMHandler) ListVM() []*irs.VMInfo {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone

	serverList, err := vmHandler.Client.Instances.List(projectID, zone).Do()
	if err != nil {
		panic(err)
	}

	var vmList []*irs.VMInfo
	for _, server := range serverList.Items {
		vmInfo := mappingServerInfo(server)
		vmList = append(vmList, &vmInfo)
	}

	return vmList
}

func (vmHandler *GCPVMHandler) GetVM(vmName string) irs.VMInfo {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone

	vm, err := vmHandler.Client.Instances.Get(projectID, zone, vmName).Do()
	if err != nil {
		panic(err)
	}

	vmInfo := mappingServerInfo(vm)
	return vmInfo
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
		Id:   string(server.Id),
		Region: irs.RegionInfo{
			Zone: server.Zone,
		},
		VNIC:         server.NetworkInterfaces[0].Name,
		SpecID:       server.MachineType,
		PublicIP:     server.NetworkInterfaces[0].AccessConfigs[0].NatIP,
		PrivateIP:    server.NetworkInterfaces[0].NetworkIP,
		VNetworkID:   server.NetworkInterfaces[0].Network,
		SubNetworkID: server.NetworkInterfaces[0].Subnetwork,
	}

	return vmInfo
}
