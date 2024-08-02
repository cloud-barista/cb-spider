package resources

import (
	"context"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type DockerVMHandler struct {
	Region  idrv.RegionInfo
	Context context.Context
	Client  *client.Client
}

func (vmHandler *DockerVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	cblogger.Info("Docker Cloud Driver: called StartVM()!")

	// set Port binding
	config := &container.Config{
		Image: vmReqInfo.ImageIID.NameId,
		ExposedPorts: nat.PortSet{
			"80/tcp": struct{}{},
		},
	}
	// @todo now, fixed port binding. by powerkim, 2020.05.19
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"80/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "8080",
				},
			},
		},
	}

	resp, err := vmHandler.Client.ContainerCreate(vmHandler.Context, config, hostConfig, nil, nil, "")
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}

	if err := vmHandler.Client.ContainerStart(vmHandler.Context, resp.ID, container.StartOptions{}); err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}

	contJson, err := vmHandler.Client.ContainerInspect(vmHandler.Context, resp.ID)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}

	return getVMInfoByContainerJSON(vmHandler.Region, vmReqInfo.IId, contJson), nil
}

func getVMInfoByContainerJSON(regionInfo idrv.RegionInfo, vmReqIID irs.IID, contJson types.ContainerJSON) irs.VMInfo {
	container := contJson.ContainerJSONBase
	networks := contJson.NetworkSettings.Networks["bridge"]

	iid := vmReqIID
	iid.SystemId = container.ID

	int64Time, _ := strconv.ParseInt(container.Created, 10, 64)

	vmInfo := irs.VMInfo{
		IId:               iid,
		StartTime:         time.Unix(int64Time, 0),
		Region:            irs.RegionInfo{regionInfo.Region, regionInfo.Zone},
		ImageIId:          irs.IID{container.Image, container.Image},
		VMSpecName:        "",
		VpcIID:            irs.IID{},
		SubnetIID:         irs.IID{},
		SecurityGroupIIds: []irs.IID{},
		KeyPairIId:        irs.IID{},
		VMUserId:          "",
		VMUserPasswd:      "",
		NetworkInterface:  networks.NetworkID,
		PublicIP:          "",
		PublicDNS:         "",
		PrivateIP:         networks.IPAddress,
		PrivateDNS:        "",
		VMBootDisk:        "",
		VMBlockDisk:       "",
		KeyValueList:      []irs.KeyValue{},
	}
	return vmInfo
}

func (vmHandler *DockerVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("Docker Cloud Driver: called SuspendVM()!")

	err := vmHandler.Client.ContainerPause(vmHandler.Context, vmIID.SystemId)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	return irs.Suspending, nil
}

func (vmHandler *DockerVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("Docker Cloud Driver: called ResumeVM()!")

	err := vmHandler.Client.ContainerUnpause(vmHandler.Context, vmIID.SystemId)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	return irs.Resuming, nil
}

func (vmHandler *DockerVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("Docker Cloud Driver: called RebootVM()!")

	err := vmHandler.Client.ContainerRestart(vmHandler.Context, vmIID.SystemId, container.StopOptions{})
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	return irs.Rebooting, nil
}

// (1) docker stop
// (2) docker rm
func (vmHandler *DockerVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("Docker Cloud Driver: called TerminateVM()!")

	err := vmHandler.Client.ContainerStop(vmHandler.Context, vmIID.SystemId, container.StopOptions{})
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	statusCh, errCh := vmHandler.Client.ContainerWait(vmHandler.Context, vmIID.SystemId, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			cblogger.Error(err)
			return "", err
		}
	case <-statusCh:
	}

	err = vmHandler.Client.ContainerRemove(vmHandler.Context, vmIID.SystemId, container.RemoveOptions{})
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	return irs.NotExist, nil
}

func (vmHandler *DockerVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	cblogger.Info("Docker Cloud Driver: called ListVMStatus()!")

	containers, err := vmHandler.Client.ContainerList(vmHandler.Context, container.ListOptions{})
	if err != nil {
		cblogger.Error(err)
		return []*irs.VMStatusInfo{}, err
	}

	var vmStatusInfoList []*irs.VMStatusInfo
	for _, container := range containers {
		vmStatusInfo := irs.VMStatusInfo{irs.IID{"", container.ID}, getMappedStatus(container.State)}
		vmStatusInfoList = append(vmStatusInfoList, &vmStatusInfo)
	}

	return vmStatusInfoList, nil
}

func getMappedStatus(containerStatus string) irs.VMStatus {
	switch containerStatus {
	case "created":
		return irs.Creating
	case "running":
		return irs.Running
	case "paused":
		return irs.Suspended
	case "restarting":
		return irs.Rebooting
	case "removing":
		return irs.Terminating
	default:
		return irs.Failed
	}
}

func (vmHandler *DockerVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("Docker Cloud Driver: called GetVMStatus()!")

	container, err := vmHandler.Client.ContainerInspect(vmHandler.Context, vmIID.SystemId)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	return getMappedStatus(container.State.Status), nil
}

func (vmHandler *DockerVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger.Info("Docker Cloud Driver: called ListVM()!")

	containers, err := vmHandler.Client.ContainerList(vmHandler.Context, container.ListOptions{})
	if err != nil {
		cblogger.Error(err)
		return []*irs.VMInfo{}, err
	}

	var vmList []*irs.VMInfo
	for _, container := range containers {
		vmInfo := getVMInfoByContainer(vmHandler.Region, container)
		vmList = append(vmList, &vmInfo)
	}

	return vmList, nil
}

func getVMInfoByContainer(regionInfo idrv.RegionInfo, container types.Container) irs.VMInfo {
	vmIID := irs.IID{"", container.ID}
	networks := container.NetworkSettings.Networks["bridge"]

	vmInfo := irs.VMInfo{
		IId:               vmIID,
		StartTime:         time.Unix(container.Created, 0),
		Region:            irs.RegionInfo{regionInfo.Region, regionInfo.Zone},
		ImageIId:          irs.IID{container.Image, container.ImageID},
		VMSpecName:        "",
		VpcIID:            irs.IID{},
		SubnetIID:         irs.IID{},
		SecurityGroupIIds: []irs.IID{},
		KeyPairIId:        irs.IID{},
		VMUserId:          "",
		VMUserPasswd:      "",
		NetworkInterface:  networks.NetworkID,
		PublicIP:          "",
		PublicDNS:         "",
		PrivateIP:         networks.IPAddress,
		PrivateDNS:        "",
		VMBootDisk:        "",
		VMBlockDisk:       "",
		KeyValueList:      []irs.KeyValue{},
	}

	return vmInfo
}

func (vmHandler *DockerVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	cblogger.Info("Docker Cloud Driver: called GetVM()!")

	container, err := vmHandler.Client.ContainerInspect(vmHandler.Context, vmIID.SystemId)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}
	return getVMInfoByContainerJSON(vmHandler.Region, vmIID, container), nil
}
