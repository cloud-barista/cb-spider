package resources

import (
	"errors"
	"fmt"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strings"
	"sync"
	"time"

	cblog "github.com/cloud-barista/cb-log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/sirupsen/logrus"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
)

const (
	DNSNameservers   = "8.8.8.8"
	ResourceNotFound = "Resource not found"
)

var once sync.Once
var cblogger *logrus.Logger
var calllogger *logrus.Logger

func InitLog() {
	once.Do(func() {
		// cblog is a global variable.
		cblogger = cblog.GetLogger("CB-SPIDER")
		calllogger = call.GetLogger("HISCALL")
	})
}

func LoggingError(hiscallInfo call.CLOUDLOGSCHEMA, err error) {
	hiscallInfo.ErrorMSG = err.Error()
	calllogger.Info(call.String(hiscallInfo))
}

func LoggingInfo(hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) {
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
}

func GetCallLogScheme(endpoint string, resourceType call.RES_TYPE, resourceName string, apiName string) call.CLOUDLOGSCHEMA {
	cblogger.Info(fmt.Sprintf("Call %s %s", call.OPENSTACK, apiName))
	return call.CLOUDLOGSCHEMA{
		CloudOS:      call.OPENSTACK,
		RegionZone:   endpoint,
		ResourceType: resourceType,
		ResourceName: resourceName,
		CloudOSAPI:   apiName,
	}
}

func GetPublicVPCInfo(client *gophercloud.ServiceClient, typeName string) (string, error) {
	// VPC 목록 조회
	iTrue := true
	listOpts := external.ListOptsExt{
		ListOptsBuilder: networks.ListOpts{},
		External:        &iTrue,
	}
	page, err := networks.List(client, listOpts).AllPages()
	if err != nil {
		cblogger.Error("Failed to get vpc list, err=%s", err)
		return "", err
	}
	// external VPC 필터링
	var extVpcList []NetworkWithExt
	err = networks.ExtractNetworksInto(page, &extVpcList)
	if err != nil {
		cblogger.Error("Failed to get vpc list, err=%s", err)
		getErr := errors.New(fmt.Sprintf("Failed to get vpc list, err=%s", err.Error()))
		return "", getErr
	}
	if len(extVpcList) == 0 {
		cblogger.Error("Failed to get vpc list")
		return "", errors.New(fmt.Sprintf("Failed to get vpc list, external vpc not exist"))
	}
	extVpc := extVpcList[0]
	if typeName == "ID" {
		return extVpc.ID, nil
	} else if typeName == "NAME" {
		return extVpc.Name, nil
	}
	return "", nil
}

func GetFlavorByName(client *gophercloud.ServiceClient, flavorName string) (flavors.Flavor, error) {
	pages, err := flavors.ListDetail(client, nil).AllPages()
	if err != nil {
		return flavors.Flavor{}, err
	}
	flavorList, err := flavors.ExtractFlavors(pages)
	for _, flavor := range flavorList {
		if flavor.Name == flavorName {
			return flavor, nil
		}
	}
	return flavors.Flavor{}, errors.New(fmt.Sprintf("could not found Flavor with name %s ", flavorName))
}

func GetSecurityByName(networkClient *gophercloud.ServiceClient, securityName string) (*secgroups.SecurityGroup, error) {
	pages, err := secgroups.List(networkClient).AllPages()
	if err != nil {
		return nil, err
	}
	secGroupList, err := secgroups.ExtractSecurityGroups(pages)
	if err != nil {
		return nil, err
	}

	for _, s := range secGroupList {
		if strings.EqualFold(s.Name, securityName) {
			return &s, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("could not found SecurityGroups with name %s ", securityName))
}

func GetNetworkByName(networkClient *gophercloud.ServiceClient, networkName string) (*networks.Network, error) {
	pages, err := networks.List(networkClient, networks.ListOpts{Name: networkName}).AllPages()
	if err != nil {
		return nil, err
	}
	netList, err := networks.ExtractNetworks(pages)
	if err != nil {
		return nil, err
	}

	for _, s := range netList {
		if strings.EqualFold(s.Name, networkName) {
			return &s, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("could not found SecurityGroups with name %s ", networkName))
}

func GetSubnetByID(networkClient *gophercloud.ServiceClient, subnetId string) (*subnets.Subnet, error) {
	subnet, err := subnets.Get(networkClient, subnetId).Extract()
	if err != nil {
		return nil, err
	}
	return subnet, nil
}

func GetPortByDeviceID(networkClient *gophercloud.ServiceClient, deviceID string) (*ports.Port, error) {
	pages, err := ports.List(networkClient, ports.ListOpts{}).AllPages()
	if err != nil {
		return nil, err
	}
	portList, err := ports.ExtractPorts(pages)
	if err != nil {
		return nil, err
	}

	for _, s := range portList {
		if strings.EqualFold(s.DeviceID, deviceID) {
			return &s, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("could not found SecurityGroups with name %s ", deviceID))
}

func CheckIIDValidation(IId irs.IID) bool {
	if IId.NameId == "" && IId.SystemId == "" {
		return false
	}
	return true
}
