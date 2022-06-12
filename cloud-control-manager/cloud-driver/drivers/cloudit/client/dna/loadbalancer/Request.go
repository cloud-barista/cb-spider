package loadbalancer

import (
	"errors"
	"fmt"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/sirupsen/logrus"
	"strings"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type LoadBalancerAlgorithmType string
type LoadBalancerType string

const (
	LoadBalancerAlgorithmRoundRobin      LoadBalancerAlgorithmType = "rr"
	LoadBalancerAlgorithmLeastConnection LoadBalancerAlgorithmType = "lc"
	LoadBalancerAlgorithmSourceHash      LoadBalancerAlgorithmType = "sh"
	LoadBalancerExternalType             LoadBalancerType          = "EXTERNAL"
	LoadBalancerInternalType             LoadBalancerType          = "INTERNAL"
)

type LoadBalancerMember struct {
	Id                      string `json:"id"`
	MemberIp                string `json:"memberIp"`
	MemberPort              string `json:"memberPort"`
	LbId                    string `json:"lbId"`
	Creator                 string `json:"creator"`
	Network                 string `json:"network"`
	ServerName              string `json:"serverName"`
	State                   string `json:"state"`
	HealthState             string `json:"healthState"` // Health-Monitor
	Template                string `json:"template"`
	Spec                    string `json:"spec"`
	OsType                  string `json:"osType"`
	CpuNum                  int    `json:"cpuNum"`
	MemSize                 int    `json:"memSize"`
	VolumeSize              int    `json:"volumeSize"`
	HealthCheckSuccessCount int    `json:"healthCheckSuccessCount"`
	HealthCheckFailCount    int    `json:"healthCheckFailCount"`
	HostName                string `json:"hostName"`
	CreatedAt               string `json:"createdAt"`
}

type LoadBalancerReqMember struct {
	MemberIp   string `json:"memberIp"`
	MemberPort string `json:"memberPort"`
}

type LoadBalancerReqInfo struct {
	Name               string                  `json:"name" required:"true"`
	IP                 string                  `json:"ip" required:"true"`
	Scheduler          string                  `json:"scheduler" required:"true"`
	Port               int                     `json:"port"`
	Protocol           string                  `json:"protocol" required:"true"`
	Members            []LoadBalancerReqMember `json:"members" required:"true"`
	MonitorType        string                  `json:"monitorType" required:"true"`
	MaxConn            int                     `json:"maxConn" required:"true"`
	StatsPort          int                     `json:"statsPort" required:"true"`
	Type               string                  `json:"type" required:"true"`
	ResponseTime       int                     `json:"responseTime" required:"true"`       // Health-Monitor
	IntervalTime       int                     `json:"intervalTime" required:"true"`       // Health-Monitor
	UnhealthyThreshold int                     `json:"unhealthyThreshold" required:"true"` // Health-Monitor
	HealthyThreshold   int                     `json:"healthyThreshold" required:"true"`   // Health-Monitor
	HttpUrl            string                  `json:"httpUrl"`                            // Health-Monitor
	Description        string                  `json:"description"`
}

type LoadBalancerInfo struct {
	Id                 string               `json:"id"`
	TenantId           string               `json:"tenantId"`
	Name               string               `json:"name"`
	State              string               `json:"state"`
	Ip                 string               `json:"ip"`
	Port               int                  `json:"port"`
	Protocol           string               `json:"protocol"`
	HaCaId             string               `json:"haCaId"`
	Type               string               `json:"type"`
	Scheduler          string               `json:"scheduler"`
	Members            []LoadBalancerMember `json:"members" required:"true"`
	MemberCount        int                  `json:"memberCount"`
	Creator            string               `json:"creator"`
	Protection         int                  `json:"protection"`
	MonitorType        string               `json:"monitorType"`        // Health-Monitor
	ResponseTime       int                  `json:"responseTime"`       // Health-Monitor
	IntervalTime       int                  `json:"intervalTime"`       // Health-Monitor
	UnhealthyThreshold int                  `json:"unhealthyThreshold"` // Health-Monitor
	HealthyThreshold   int                  `json:"healthyThreshold"`   // Health-Monitor
	HttpUrl            string               `json:"httpUrl"`            // Health-Monitor
	Description        string               `json:"description"`
	CreatedAt          string               `json:"createdAt"`
	Url                string               `json:"url"`
	Tlsv12Enabled      int                  `json:"tlsv12Enabled"`
	MaxConn            int                  `json:"maxConn"`
	StatsPort          int                  `json:"statsPort"`
}

type AddMemberInfo struct {
	Network    string `json:"network"`
	MemberIp   string `json:"memberIp"`
	MemberPort string `json:"memberPort"`
	HostName   string `json:"hostName"`
}

type LoadBalancerUpdateInfo struct {
	Name        string `json:"name,omitempty"`
	Protection  int    `json:"protection,omitempty"`
	Description string `json:"description"`
}

type LoadBalancerHealthCheckerUpdateInfo struct {
	Scheduler          string `json:"scheduler,omitempty"`
	ResponseTime       int    `json:"responseTime,omitempty"`
	IntervalTime       int    `json:"intervalTime,omitempty"`
	UnhealthyThreshold int    `json:"unhealthyThreshold,omitempty"`
	HealthyThreshold   int    `json:"healthyThreshold,omitempty"`
}

func UpdateHealthChecker(restClient *client.RestClient, requestOpts *client.RequestOpts, nlbId string) (bool, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers", nlbId, "policy")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Put(requestURL, nil, &result.Body, requestOpts); result.Err != nil {
		if result.Err.Error() != "EOF" {
			return false, result.Err
		}
	}
	return true, nil
}

func Update(restClient *client.RestClient, requestOpts *client.RequestOpts, nlbId string) (*LoadBalancerInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers", nlbId, "update")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Put(requestURL, nil, &result.Body, requestOpts); result.Err != nil {
		if result.Err.Error() != "EOF" {
			return nil, result.Err
		}
	}

	cblogger.Info(requestURL)

	getRequestUrl := restClient.CreateRequestBaseURL(client.DNA, "load-balancers")

	getRequestOpts := requestOpts
	getRequestOpts.JSONBody = nil

	var getResult client.Result
	if _, getResult.Err = restClient.Get(getRequestUrl, &getResult.Body, getRequestOpts); getResult.Err != nil {
		return nil, getResult.Err
	}
	var nlbs []LoadBalancerInfo
	if err := getResult.ExtractInto(&nlbs); err != nil {
		return nil, err
	}
	for _, nlb := range nlbs {
		if strings.EqualFold(nlbId, nlb.Id) {
			memberRequestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers", nlb.Id)
			var memberResult client.Result
			if _, memberResult.Err = restClient.Get(memberRequestURL, &memberResult.Body, getRequestOpts); memberResult.Err != nil {
				return nil, memberResult.Err
			}
			var nlbMembers []LoadBalancerMember
			if err := memberResult.ExtractInto(&nlbMembers); err != nil {
				return nil, err
			}
			nlb.Members = nlbMembers
			return &nlb, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("not found load-balancer ID : %s", nlbId))
}

func List(restClient *client.RestClient, requestOpts *client.RequestOpts) (*[]LoadBalancerInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var nlbs []LoadBalancerInfo
	if err := result.ExtractInto(&nlbs); err != nil {
		return nil, err
	}
	for i, nlb := range nlbs {
		memberRequestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers", nlb.Id)
		var memberResult client.Result
		if _, memberResult.Err = restClient.Get(memberRequestURL, &memberResult.Body, requestOpts); memberResult.Err != nil {
			return nil, memberResult.Err
		}
		var nlbMembers []LoadBalancerMember
		if err := memberResult.ExtractInto(&nlbMembers); err != nil {
			return nil, err
		}
		nlbs[i].Members = nlbMembers
	}
	return &nlbs, nil
}

func MemberList(restClient *client.RestClient, requestOpts *client.RequestOpts, nlbId string) (*[]LoadBalancerMember, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers", nlbId)
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var nlbMembers []LoadBalancerMember
	if err := result.ExtractInto(&nlbMembers); err != nil {
		return nil, err
	}
	return &nlbMembers, nil
}

func DeleteMember(restClient *client.RestClient, requestOpts *client.RequestOpts, nlbId string, memberId string) error {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers", nlbId, memberId)
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Delete(requestURL, requestOpts); result.Err != nil {
		if result.Err.Error() =="EOF" {
			return nil
		}
		return result.Err
	}
	return nil
}

func AddMember(restClient *client.RestClient, requestOpts *client.RequestOpts, nlbId string) (bool, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers", nlbId)
	cblogger.Info(requestURL)
	var result client.Result

	if _, result.Err = restClient.Post(requestURL, nil, &result.Body, requestOpts); result.Err != nil {
		return false, result.Err
	}
	return true, nil

}

func Get(restClient *client.RestClient, requestOpts *client.RequestOpts, nlbId string) (*LoadBalancerInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var nlbs []LoadBalancerInfo
	if err := result.ExtractInto(&nlbs); err != nil {
		return nil, err
	}
	for _, nlb := range nlbs {
		if strings.EqualFold(nlbId, nlb.Id) {
			memberRequestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers", nlb.Id)
			var memberResult client.Result
			if _, memberResult.Err = restClient.Get(memberRequestURL, &memberResult.Body, requestOpts); memberResult.Err != nil {
				return nil, memberResult.Err
			}
			var nlbMembers []LoadBalancerMember
			if err := memberResult.ExtractInto(&nlbMembers); err != nil {
				return nil, err
			}
			nlb.Members = nlbMembers
			return &nlb, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("not found load-balancer ID : %s", nlbId))
}

func GetSimple(restClient *client.RestClient, requestOpts *client.RequestOpts, nlbId string) (*LoadBalancerInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var nlbs []LoadBalancerInfo
	if err := result.ExtractInto(&nlbs); err != nil {
		return nil, err
	}
	for _, nlb := range nlbs {
		if strings.EqualFold(nlbId, nlb.Id) {
			return &nlb, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("not found load-balancer ID : %s", nlbId))
}

func GetByName(restClient *client.RestClient, requestOpts *client.RequestOpts, nlbName string) (*LoadBalancerInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Get(requestURL, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var nlbs []LoadBalancerInfo
	if err := result.ExtractInto(&nlbs); err != nil {
		return nil, err
	}
	for _, nlb := range nlbs {
		if strings.EqualFold(nlbName, nlb.Name) {
			memberRequestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers", nlb.Id)
			var memberResult client.Result
			if _, memberResult.Err = restClient.Get(memberRequestURL, &memberResult.Body, requestOpts); memberResult.Err != nil {
				return nil, memberResult.Err
			}
			var nlbMembers []LoadBalancerMember
			if err := memberResult.ExtractInto(&nlbMembers); err != nil {
				return nil, err
			}
			nlb.Members = nlbMembers
			return &nlb, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("not found load-balancer Name : %s", nlbName))
}

func Create(restClient *client.RestClient, requestOpts *client.RequestOpts) (*LoadBalancerInfo, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers")
	cblogger.Info(requestURL)
	var result client.Result
	if _, result.Err = restClient.Post(requestURL, nil, &result.Body, requestOpts); result.Err != nil {
		return nil, result.Err
	}

	var nlb LoadBalancerInfo
	if err := result.ExtractInto(&nlb); err != nil {
		return nil, err
	}

	memberRequestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers", nlb.Id)
	cblogger.Info(memberRequestURL)

	memberRequestOpts := client.RequestOpts{
		MoreHeaders: requestOpts.MoreHeaders,
	}

	var memberResult client.Result
	if _, memberResult.Err = restClient.Get(memberRequestURL, &memberResult.Body, &memberRequestOpts); memberResult.Err != nil {
		return nil, memberResult.Err
	}

	var nlbMembers []LoadBalancerMember
	if err := memberResult.ExtractInto(&nlbMembers); err != nil {
		return nil, err
	}
	nlb.Members = nlbMembers

	return &nlb, nil
}

func Delete(restClient *client.RestClient, requestOpts *client.RequestOpts, nlbId string) (bool, error) {
	requestURL := restClient.CreateRequestBaseURL(client.DNA, "load-balancers", nlbId)
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = restClient.Delete(requestURL, requestOpts); result.Err != nil {
		return false, result.Err
	}
	return true, nil

}
