package adaptiveip

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

type PublicIPReqInfo struct {
	IP         string `json:"ip" required:"true"`
	Name       string `json:"name" required:"true"`
	PrivateIP  string `json:"privateIp" required:"true"` // PublicIP가 적용되는 VM의 Private IP
	Protection int    `json:"protection" required:"false"`
}

type IPInfo struct {
	IP      string `json:"addr"`
	gateway string
	prefix  string
	state   string
	netmask string
}

type AdaptiveIPInfo struct {
	ID          string
	IP          string
	Name        string
	Rules       interface{}
	TenantId    string
	Creator     string
	State       string
	CreatedAt   string
	PrivateIp   string
	Protection  int
	RuleCount   int
	VmName      string
	Description string
}

func List(restClient *client.RestClient, requestOpts *client.RequestOpts) (*[]AdaptiveIPInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "adaptive-ips")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var adaptiveIP []AdaptiveIPInfo
	if err := result.ExtractInto(&adaptiveIP); err != nil {
		return nil, err
	}
	return &adaptiveIP, nil
}

func ListAvailableIP(restClient *client.RestClient, requestOpts *client.RequestOpts) (*[]IPInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "ips")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var availableIP []IPInfo
	if err := result.ExtractInto(&availableIP); err != nil {
		return nil, err
	}
	return &availableIP, nil
}

func Get(restClient *client.RestClient, adaptiveIPId string, requestOpts *client.RequestOpts) (*AdaptiveIPInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "adaptive-ips", adaptiveIPId, "detail")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var adaptiveIP AdaptiveIPInfo
	if err := result.ExtractInto(&adaptiveIP); err != nil {
		return nil, err
	}
	return &adaptiveIP, nil
}

func Create(restClient *client.RestClient, requestOpts *client.RequestOpts) (*AdaptiveIPInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "adaptive-ips")
	cblogger.Info(requestURL)

	var result client.Result

	if _, result.Err = restClient.Post(requestURL, nil, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var adaptiveIP AdaptiveIPInfo
	if err := result.ExtractInto(&adaptiveIP); err != nil {
		return nil, err
	}
	return &adaptiveIP, nil
}

func Delete(restClient *client.RestClient, adaptiveIP string, requestOpts *client.RequestOpts) error {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "adaptive-ips", adaptiveIP)
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Delete(requestURL, requestOpts); result.Err != nil {
		return result.Err
	}
	return nil
}
