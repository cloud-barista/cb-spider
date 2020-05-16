// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Docker Driver.
//
// by CB-Spider Team, 2020.05.

package resources

import (
	"context"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"

	"os"
//	"errors"
//	"fmt"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
//	"reflect"
//	"strings"
)

type DockerVMHandler struct {
        Region        idrv.RegionInfo
        Context       context.Context
        Client        *client.Client
}

func (vmHandler *DockerVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
        cblogger.Info("Docker Cloud Driver: called StartVM()!")

	resp, err := vmHandler.Client.ContainerCreate(vmHandler.Context, &container.Config{
                Image: vmReqInfo.IId.NameId,
                //Cmd:   []string{"echo", "hello world"},
                //Tty:   true,
        }, nil, nil, "")
        if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
        }

        if err := vmHandler.Client.ContainerStart(vmHandler.Context, resp.ID, types.ContainerStartOptions{}); err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
        }

        statusCh, errCh := vmHandler.Client.ContainerWait(vmHandler.Context, resp.ID, container.WaitConditionNotRunning)
        select {
        case err := <-errCh:
                if err != nil {
			cblogger.Error(err)
			return irs.VMInfo{}, err
                }
        case <-statusCh:
        }
/*
        out, err := vmHandler.Client.ContainerLogs(vmHandler.Context, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
        if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
        }
*/

	contJson, err := ContainerInspect(vmHandler.Context, resp.ID)
	if err != nil {
                cblogger.Error(err)
                return irs.VMInfo{}, err
        }

	fmt.Printf("============= \n %#v \n", contJson)

/*
	vm, err = vmHandler.Client.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmReqInfo.IId.NameId, compute.InstanceView)
	vmInfo := vmHandler.mappingServerInfo(vm)
*/



	return irs.VMInfo{}, nil
}

func (vmHandler *DockerVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
/*
	future, err := vmHandler.Client.PowerOff(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmIID.NameId)
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
	vmStatus, err := vmHandler.GetVMStatus(vmIID)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	return vmStatus, nil
*/
	return "", nil
}

func (vmHandler *DockerVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
/*
	future, err := vmHandler.Client.Start(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmIID.NameId)
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
*/
	return "", nil
}

func (vmHandler *DockerVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
/*
	future, err := vmHandler.Client.Restart(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmIID.NameId)
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
*/
	return "", nil
}

func (vmHandler *DockerVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
/*
	// VM 삭제 시 OS Disk도 함께 삭제 처리
	// VM OSDisk 이름 가져오기
	vmInfo, err := vmHandler.GetVM(vmIID)
	if err != nil {
		return irs.Failed, err
	}
	osDiskName := vmInfo.VMBootDisk

	// TODO: nested flow 개선
	// VNic에서 PublicIP 연결해제
	vNicDetachStatus, err := DetachVNic(vmHandler, vmInfo)
	if err != nil {
		cblogger.Error(err)
		return vNicDetachStatus, err
	}

	// VM 삭제
	future, err := vmHandler.Client.Delete(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmIID.NameId)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	return irs.NotExist, nil
*/
	return "", nil
}

func (vmHandler *DockerVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
/*
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
*/
	return []*irs.VMStatusInfo{}, nil
}

func (vmHandler *DockerVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
/*
	instanceView, err := vmHandler.Client.InstanceView(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmIID.NameId)
	if err != nil {
		cblogger.Error(err)
		return irs.Failed, err
	}

	// Get powerState, provisioningState
	vmStatus := getVmStatus(instanceView)
	return irs.VMStatus(vmStatus), nil
*/
	return "", nil
}

func (vmHandler *DockerVMHandler) ListVM() ([]*irs.VMInfo, error) {
/*
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
*/
	return []*irs.VMInfo{}, nil
}

func (vmHandler *DockerVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
/*
	vm, err := vmHandler.Client.Get(vmHandler.Ctx, vmHandler.Region.ResourceGroup, vmIID.NameId, compute.InstanceView)
	if err != nil {
		return irs.VMInfo{}, err
	}

	vmInfo := vmHandler.mappingServerInfo(vm)
	return vmInfo, nil
*/
	return irs.VMInfo{}, nil
}

/*
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
*/

