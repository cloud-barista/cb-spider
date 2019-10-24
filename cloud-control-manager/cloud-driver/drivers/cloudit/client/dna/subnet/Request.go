package subnet

import (
	//"fmt"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/sirupsen/logrus"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type VNetworkReqInfo struct {
	Name       string `json:"name" required:"true"`
	Addr       string `json:"addr" required:"true"`
	Prefix     string `json:"prefix" required:"true"`
	Gateway    string `json:"gateway" required:"false"`
	Protection int    `json:"protection" required:"false"`
}

type SubnetInfo struct {
	ID          string
	TenantId    string
	Addr        string
	Prefix      string
	Gateway     string
	Creator     string
	Protection  int
	Name        string
	State       string
	Vlan        int
	CreatedAt   string
	NicCount    int
	Description string
}

func List(restClient *client.RestClient, requestOpts *client.RequestOpts) (*[]SubnetInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "subnets")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var subnet []SubnetInfo
	if err := result.ExtractInto(&subnet); err != nil {
		return nil, err
	}
	return &subnet, nil
}

func ListCreatableSubnet(restClient *client.RestClient, requestOpts *client.RequestOpts) (*[]SubnetInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "subnets", "creatable")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var subnet []SubnetInfo
	if err := result.ExtractInto(&subnet); err != nil {
		return nil, err
	}
	return &subnet, nil
}

func Get(restClient *client.RestClient, subnetId string, requestOpts *client.RequestOpts) (*SubnetInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, subnetId, "detail")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var subnet SubnetInfo
	if err := result.ExtractInto(&subnet); err != nil {
		return nil, err
	}
	return &subnet, nil
}

func Create(restClient *client.RestClient, requestOpts *client.RequestOpts) (*SubnetInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "subnets")
	cblogger.Info(requestURL)

	var result client.Result

	if _, result.Err = restClient.Post(requestURL, nil, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var subnet SubnetInfo
	if err := result.ExtractInto(&subnet); err != nil {
		return nil, err
	}
	return &subnet, nil
}

func Delete(restClient *client.RestClient, addr string, requestOpts *client.RequestOpts) error {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "subnets", addr)
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Delete(requestURL, requestOpts); result.Err != nil {
		return result.Err
	}
	return nil
}
