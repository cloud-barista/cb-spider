package specs

import (
	cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/sirupsen/logrus"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type VMSpecInfo struct {
	Id      string `json:"id" required:"true"`
	Name    string `json:"name" required:"true"`
	Cpu     int    `json:"cpu" required:"true"`
	Mem     int    `json:"mem" required:"true"`
	Disk    int    `json:"disk" required:"true"`
	Enabled int    `json:"enabled" required:"true"`
}

func List(restClient *client.RestClient, requestOpts *client.RequestOpts) (*[]VMSpecInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.ACE, "specs")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var spec []VMSpecInfo
	if err := result.ExtractInto(&spec); err != nil {
		return nil, err
	}
	return &spec, nil
}
