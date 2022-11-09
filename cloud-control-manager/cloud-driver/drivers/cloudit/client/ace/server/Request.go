package server

import (
	cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/disk"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/iam/securitygroup"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type SecGroupInfo struct {
	Id string `json:"id" required:"true"`
}

type VmTagInfo struct {
	MyImageIID *irs.IID `json:"MyImageIID,omitempty"`
	Keypair    string
}

type VMReqInfo struct {
	TemplateId   string         `json:"templateId,omitempty"`
	SnapshotId   string         `json:"snapshotId,omitempty"`
	SpecId       string         `json:"specId" required:"true"`
	Name         string         `json:"name" required:"true"`
	HostName     string         `json:"hostName" required:"true"`
	RootPassword string         `json:"rootPassword" required:"true"`
	SubnetAddr   string         `json:"subnetAddr" required:"true"`
	Secgroups    []SecGroupInfo `json:"secgroups" required:"true"`
	Description  string         `json:"description" required:"false"`
	Protection   int            `json:"protection" required:"false"`
	ClusterId    string         `json:"clusterId,omitempty"`
}

type ServerInfo struct {
	VolumeInfoList interface{}
	VmNicInfoList  interface{}
	NicMapInfo     []struct {
		Name    string
		Count   int
		Address string `json:"addr"`
	}
	PoolMapInfo []struct {
		Name       string
		Count      int
		PoolID     string `json:"pool_id"`
		FileSystem string
	}
	AdaptiveIpMapInfo []struct {
		IP        string
		Count     int
		PrivateIP string `json:"private_ip"`
	}
	ID                string
	TenantID          string
	CpuNum            float32
	MemSize           float32
	VncPort           int
	RepeaterPort      int
	State             string
	NodeIp            string
	NodeHostName      string
	Name              string
	Protection        int
	CreatedAt         string
	IsoId             string
	IsoPath           string
	Iso               string
	Template          string
	TemplateID        string
	OsType            string
	RootPassword      string
	HostName          string
	Creator           string
	VolumeId          string
	VolumeSize        int
	VolumeMode        string
	MacAddr           string
	Spec              string
	SpecId            string
	Pool              string
	PoolId            string
	Cycle             int
	Metric            int
	MigrationPort     int
	MigrationIp       string
	Cloudinit         bool
	DeleteVolume      bool
	ServerCount       int
	PrivateIp         string
	AdaptiveIp        string
	InitCloud         int
	ClusterId         string
	ClusterName       string
	NicType           string
	Secgroups         []securitygroup.SecurityGroupInfo
	Ip                string
	SubnetAddr        string
	DeviceId          string
	Gpu               string
	GpuCount          int
	GpuId             string
	Description       string
	DiskSize          int
	DiskCount         int
	IsoInsertedAt     string
	Puppet            int
	SshKeyName        string
	SshPublicKey      string
	TemplateOwnership string
	TemplateType      string
	VmStatInfo        string
}

func List(restClient *client.RestClient, requestOpts *client.RequestOpts) (*[]ServerInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "servers")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var server []ServerInfo
	if err := result.ExtractInto(&server); err != nil {
		return nil, err
	}
	return &server, nil
}

func Get(restClient *client.RestClient, id string, requestOpts *client.RequestOpts) (*ServerInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "servers", id)
	cblogger.Info(requestURL)

	var result client.Result
	_, result.Err = restClient.Get(requestURL, &result.Body, requestOpts)

	var server ServerInfo
	if err := result.ExtractInto(&server); err != nil {
		return nil, err
	}
	return &server, nil
}

// create
func Start(restClient *client.RestClient, requestOpts *client.RequestOpts) (*ServerInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "servers")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Post(requestURL, nil, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var server ServerInfo
	if err := result.ExtractInto(&server); err != nil {
		return nil, err
	}

	return &server, nil
}

// shutdown
func Suspend(restClient *client.RestClient, id string, requestOpts *client.RequestOpts) error {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "servers", id, "shutdown")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Post(requestURL, nil, nil, requestOpts); result.Err != nil {
		return result.Err
	}
	return nil
}

// start
func Resume(restClient *client.RestClient, id string, requestOpts *client.RequestOpts) error {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "servers", id, "start")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Post(requestURL, nil, nil, requestOpts); result.Err != nil {
		return result.Err
	}
	return nil
}

// reboot
func Reboot(restClient *client.RestClient, id string, requestOpts *client.RequestOpts) error {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "servers", id, "reboot")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Post(requestURL, nil, nil, requestOpts); result.Err != nil {
		return result.Err
	}
	return nil
}

// delete
func Terminate(restClient *client.RestClient, id string, requestOpts *client.RequestOpts) error {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "servers", id)
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Delete(requestURL, requestOpts); result.Err != nil {
		return result.Err
	}
	return nil
}

func AttachVolume(restClient *client.RestClient, serverId string, requestOpts *client.RequestOpts) error {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "servers", serverId, "volumes")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Post(requestURL, nil, nil, requestOpts); result.Err != nil {
		return result.Err
	}
	return nil
}

func DetachVolume(restClient *client.RestClient, serverId string, volumeId string, requestOpts *client.RequestOpts) error {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "servers", serverId, "volumes", volumeId)
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Delete(requestURL, requestOpts); result.Err != nil {
		return result.Err
	}
	return nil
}

// get VM attached Volumes
func GetRawVmVolumes(restClient *client.RestClient, id string, requestOpts *client.RequestOpts) (*[]disk.DiskInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "servers", id, "volumes")
	cblogger.Info(requestURL)

	var result client.Result
	_, result.Err = restClient.Get(requestURL, &result.Body, requestOpts)
	if result.Err != nil {
		return nil, result.Err
	}

	var responseList []struct {
		VolumeId    string
		VolumeName  string
		Dev         string
		Description string
	}
	if err := result.ExtractInto(&responseList); err != nil {
		return nil, err
	}

	var volumeList []disk.DiskInfo
	for _, response := range responseList {
		volumeList = append(volumeList, disk.DiskInfo{
			ID:          response.VolumeId,
			Name:        response.VolumeName,
			Dev:         response.Dev,
			Description: response.Description,
		})
	}

	return &volumeList, nil
}
