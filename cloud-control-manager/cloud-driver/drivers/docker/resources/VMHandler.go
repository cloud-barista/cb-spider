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
	"github.com/docker/go-connections/nat"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"time"
	"strconv"
)

type DockerVMHandler struct {
        Region        idrv.RegionInfo
        Context       context.Context
        Client        *client.Client
}

func (vmHandler *DockerVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
        cblogger.Info("Docker Cloud Driver: called StartVM()!")

/*
 ref) https://godoc.org/github.com/docker/docker/api/types/container#Config
type Config struct {
    Hostname        string              // Hostname
    Domainname      string              // Domainname
    User            string              // User that will run the command(s) inside the container, also support user:group
    AttachStdin     bool                // Attach the standard input, makes possible user interaction
    AttachStdout    bool                // Attach the standard output
    AttachStderr    bool                // Attach the standard error
    ExposedPorts    nat.PortSet         `json:",omitempty"` // List of exposed ports
    Tty             bool                // Attach standard streams to a tty, including stdin if it is not closed.
    OpenStdin       bool                // Open stdin
    StdinOnce       bool                // If true, close stdin after the 1 attached client disconnects.
    Env             []string            // List of environment variable to set in the container
    Cmd             strslice.StrSlice   // Command to run when starting the container
    Healthcheck     *HealthConfig       `json:",omitempty"` // Healthcheck describes how to check the container is healthy
    ArgsEscaped     bool                `json:",omitempty"` // True if command is already escaped (meaning treat as a command line) (Windows specific).
    Image           string              // Name of the image as it was passed by the operator (e.g. could be symbolic)
    Volumes         map[string]struct{} // List of volumes (mounts) used for the container
    WorkingDir      string              // Current directory (PWD) in the command will be launched
    Entrypoint      strslice.StrSlice   // Entrypoint to run when starting the container
    NetworkDisabled bool                `json:",omitempty"` // Is network disabled
    MacAddress      string              `json:",omitempty"` // Mac Address of the container
    OnBuild         []string            // ONBUILD metadata that were defined on the image Dockerfile
    Labels          map[string]string   // List of labels set to this container
    StopSignal      string              `json:",omitempty"` // Signal to stop a container
    StopTimeout     *int                `json:",omitempty"` // Timeout (in seconds) to stop a container
    Shell           strslice.StrSlice   `json:",omitempty"` // Shell for shell-form of RUN, CMD, ENTRYPOINT
}
*/


	// set Port binding
	config := &container.Config{
		Image: vmReqInfo.ImageIID.NameId,
		//Image: "panubo/sshd",
		//Cmd:   []string{"echo", "hello world"},
                //Tty:   true,
		ExposedPorts: nat.PortSet{
				//"80/tcp": struct{}{},
			},
	}
	// @todo now, fixed port binding. by powerkim, 2020.05.19
        hostConfig := &container.HostConfig{
                PortBindings: nat.PortMap{
                        "80/tcp": []nat.PortBinding{
                                {
                                        HostIP: "0.0.0.0",
                                        HostPort: "8080",
                                },
                        },
                },
        }

/*
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"22/tcp": []nat.PortBinding{
				{
					HostIP: "0.0.0.0",
					HostPort: "44",
				},
			},
		},
	}
*/
	resp, err := vmHandler.Client.ContainerCreate(vmHandler.Context, config, hostConfig, nil, nil, "")
        if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
        }

        if err := vmHandler.Client.ContainerStart(vmHandler.Context, resp.ID, types.ContainerStartOptions{}); err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
        }
/*
        statusCh, errCh := vmHandler.Client.ContainerWait(vmHandler.Context, resp.ID, container.WaitConditionNotRunning)
        select {
        case err := <-errCh:
                if err != nil {
			cblogger.Error(err)
			return irs.VMInfo{}, err
                }
        case <-statusCh:
        }

        out, err := vmHandler.Client.ContainerLogs(vmHandler.Context, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
        if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
        }
	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
*/

	contJson, err := vmHandler.Client.ContainerInspect(vmHandler.Context, resp.ID)
	if err != nil {
                cblogger.Error(err)
                return irs.VMInfo{}, err
        }

	return getVMInfoByContainerJSON(vmHandler.Region, vmReqInfo.IId, contJson), nil
}

func getVMInfoByContainerJSON(regionInfo idrv.RegionInfo, vmReqIID irs.IID, contJson types.ContainerJSON) irs.VMInfo {
/* ref) https://godoc.org/github.com/docker/docker/api/types#ContainerJSON
	type ContainerJSON struct {
	    *ContainerJSONBase
	    Mounts          []MountPoint
	    Config          *container.Config
	    NetworkSettings *NetworkSettings
	}
	type ContainerJSONBase struct {
	    ID              string `json:"Id"`
	    Created         string
	    Path            string
	    Args            []string
	    State           *ContainerState
	    Image           string
	    ResolvConfPath  string
	    HostnamePath    string
	    HostsPath       string
	    LogPath         string
	    Node            *ContainerNode `json:",omitempty"` // Node is only propagated by Docker Swarm standalone API
	    Name            string
	    RestartCount    int
	    Driver          string
	    Platform        string
	    MountLabel      string
	    ProcessLabel    string
	    AppArmorProfile string
	    ExecIDs         []string
	    HostConfig      *container.HostConfig
	    GraphDriver     GraphDriverData
	    SizeRw          *int64 `json:",omitempty"`
	    SizeRootFs      *int64 `json:",omitempty"`
	}
*/
	container := contJson.ContainerJSONBase
	networks := contJson.NetworkSettings.Networks["bridge"] // @todo Now, only bridge.

	iid := vmReqIID
	iid.SystemId = container.ID

	int64Time, _ := strconv.ParseInt(container.Created, 10, 64)

	vmInfo := irs.VMInfo{
		IId:	iid,
                StartTime:       time.Unix(int64Time, 0),
                Region:          irs.RegionInfo {regionInfo.Region, regionInfo.Zone},
		ImageIId:	 irs.IID{container.Image, container.Image},
                VMSpecName:      "",
                VpcIID:          irs.IID{},
                SubnetIID:       irs.IID{},
                SecurityGroupIIds: []irs.IID{},

                KeyPairIId:     irs.IID{},

                VMUserId:       "",
                VMUserPasswd:   "",

                NetworkInterface: networks.NetworkID,
                PublicIP:         "",
                PublicDNS:        "",
                PrivateIP:        networks.IPAddress,
                PrivateDNS:       "",

                VMBootDisk:     "", // ex) /dev/sda1
                VMBlockDisk:    "", // ex)

                KeyValueList: []irs.KeyValue{},
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

        err = vmHandler.Client.ContainerRemove(vmHandler.Context, vmIID.SystemId, types.ContainerRemoveOptions{})
        if err != nil {
                cblogger.Error(err)
                return "", err
        }

	return irs.NotExist, nil
}

func (vmHandler *DockerVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
        cblogger.Info("Docker Cloud Driver: called ListVMStatus()!")

        // []types.Container
        containers, err := vmHandler.Client.ContainerList(vmHandler.Context, types.ContainerListOptions{})
        if err != nil {
                cblogger.Error(err)
                return []*irs.VMStatusInfo{}, err
        }

        var vmStatusInfoList []*irs.VMStatusInfo
        // Container = CM = VM
        for _, container := range containers {
		vmStatusInfo := irs.VMStatusInfo{irs.IID{"",container.ID}, getMappedStatus(container.State)}
                vmStatusInfoList = append(vmStatusInfoList, &vmStatusInfo)
        }

        return vmStatusInfoList, nil
}

func getMappedStatus(containerStatus string) irs.VMStatus {
// Container Status: "created", "running", "paused", "restarting", "removing", "exited", "dead"	
// Spider Status:     Creating,  Running,  Suspended,   Rebooting,  Terminating, 

        // Set VM Status Info
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

        // types.Container
        container, err := vmHandler.Client.ContainerInspect(vmHandler.Context, vmIID.SystemId)
        if err != nil {
                cblogger.Error(err)
                return "", err
        }

	return getMappedStatus(container.State.Status), nil
}

func (vmHandler *DockerVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger.Info("Docker Cloud Driver: called ListVM()!")

	// []types.Container
	containers, err := vmHandler.Client.ContainerList(vmHandler.Context, types.ContainerListOptions{})
        if err != nil {
                cblogger.Error(err)
                return []*irs.VMInfo{}, err
        }

	var vmList []*irs.VMInfo
	// Container = CM = VM
	for _, container := range containers {
		vmInfo := getVMInfoByContainer(vmHandler.Region, container)	
		vmList = append(vmList, &vmInfo)
	}

	return vmList, nil
}

func getVMInfoByContainer(regionInfo idrv.RegionInfo, container types.Container) irs.VMInfo {
/* 
type Container struct {
    ID         string `json:"Id"`
    Names      []string
    Image      string
    ImageID    string
    Command    string
    Created    int64
    Ports      []Port
    SizeRw     int64 `json:",omitempty"`
    SizeRootFs int64 `json:",omitempty"`
    Labels     map[string]string
    State      string
    Status     string
    HostConfig struct {
        NetworkMode string `json:",omitempty"`
    }
    NetworkSettings *SummaryNetworkSettings
    Mounts          []MountPoint
}
*/		

	// @todo NameId
	vmIID := irs.IID{"", container.ID}
	networks := container.NetworkSettings.Networks["bridge"] // @todo Now, only bridge.
 
	vmInfo := irs.VMInfo {
		IId:		 vmIID,
		StartTime:	 time.Unix(container.Created, 0),  // @todo refine time display.
		Region:          irs.RegionInfo {regionInfo.Region, regionInfo.Zone},
		ImageIId:  irs.IID{container.Image, container.ImageID},
		VMSpecName:      "",
		VpcIID:          irs.IID{}, 
		SubnetIID:       irs.IID{},
		SecurityGroupIIds: []irs.IID{},

		KeyPairIId:	irs.IID{},

		VMUserId:	"",
		VMUserPasswd: 	"",

		NetworkInterface: networks.NetworkID,
		PublicIP:         "",
		PublicDNS:        "",
		PrivateIP:        networks.IPAddress,
		PrivateDNS:       "",

		VMBootDisk:  	"", // ex) /dev/sda1
		VMBlockDisk: 	"", // ex)

		KeyValueList: []irs.KeyValue{},
	}

	return vmInfo
}


func (vmHandler *DockerVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
        cblogger.Info("Docker Cloud Driver: called GetVM()!")

        // types.Container
        container, err := vmHandler.Client.ContainerInspect(vmHandler.Context, vmIID.SystemId)
        if err != nil {
                cblogger.Error(err)
                return irs.VMInfo{}, err
        }
	return getVMInfoByContainerJSON(vmHandler.Region, vmIID, container), nil
}

