package nic

import (
	cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/iam/securitygroup"
	"github.com/sirupsen/logrus"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type VNicReqInfo struct {
	SubnetAddr string                             `json:"subnetAddr" required:"true"`
	VmId       string                             `json:"vmId" required:"true"`
	Type       string                             `json:"type" required:"true"`
	Secgroups  []securitygroup.SecurityGroupRules `json:"secgroups" required:"true"`
	IP         string                             `json:"ip" required:"true"`
}

type VmNicInfo struct {
	TenantId        string
	VmId            string
	Type            string
	Mac             string
	Dev             string
	Ip              string
	SubnetAddr      string
	Creator         string
	CreatedAt       string
	VmName          string
	NetworkName     string
	AdaptiveIp      string
	State           string
	Template        string
	SpecName        string
	CpuNum          string
	MemSize         string
	VolumeSize      string
	Qos             int
	SecGroups       []securitygroup.SecurityGroupRules
	AdaptiveMapInfo interface{}
}

func List(restClient *client.RestClient, serverId string, requestOpts *client.RequestOpts) (*[]VmNicInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "servers", serverId, "nics")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var nic []VmNicInfo
	if err := result.ExtractInto(&nic); err != nil {
		return nil, err
	}
	return &nic, nil
}

func Get(restClient *client.RestClient, serverId string, macAddr string, requestOpts *client.RequestOpts) (*VmNicInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "servers", serverId, "nics", macAddr)
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var nic VmNicInfo
	if err := result.ExtractInto(&nic); err != nil {
		return nil, err
	}
	return &nic, nil
}

func Create(restClient *client.RestClient, serverId string, requestOpts *client.RequestOpts) (*VmNicInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "servers", serverId, "nics")

	var result client.Result
	if _, result.Err = restClient.Post(requestURL, nil, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var nicInfo VmNicInfo
	if err := result.ExtractInto(&nicInfo); err != nil {
		return nil, err
	} else {
		return &nicInfo, nil
	}

}
func Delete(restClient *client.RestClient, serverId string, macAddr string, requestOpts *client.RequestOpts) error {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "servers", serverId, "nics", macAddr)

	var result client.Result
	if _, result.Err = restClient.Delete(requestURL, requestOpts); result.Err != nil {
		return result.Err
	}
	return nil
}
