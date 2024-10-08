// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2022.08.

package resources

import (
	"os"
	"fmt"
	"strings"
	"sync"
	"time"
	"net"
	"encoding/json"
	"github.com/sirupsen/logrus"

	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/extensions/secgroups"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/flavors"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/networks"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/ports"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/subnets"
	
	cblog 	 "github.com/cloud-barista/cb-log"
	call 	 "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
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

func loggingError(hiscallInfo call.CLOUDLOGSCHEMA, err error) {
	hiscallInfo.ErrorMSG = err.Error()
	calllogger.Info(call.String(hiscallInfo))
}

func loggingInfo(hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) {
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
}

func getCallLogScheme(zone string, resourceType call.RES_TYPE, resourceName string, apiName string) call.CLOUDLOGSCHEMA {
	cblogger.Infof("Calling %s %s", call.KTCLOUDVPC, apiName)

	return call.CLOUDLOGSCHEMA{
		CloudOS:      call.KTCLOUDVPC,
		RegionZone:   zone,
		ResourceType: resourceType,
		ResourceName: resourceName,
		CloudOSAPI:   apiName,
	}
}

func logAndReturnError(callLogInfo call.CLOUDLOGSCHEMA, givenErrString string, errMsg error) (error) {
	newErr := fmt.Errorf(givenErrString + "[%s]", errMsg)
	cblogger.Error(newErr.Error())
	loggingError(callLogInfo, newErr)
	return newErr
}

func getFlavorIdWithName(client *ktvpcsdk.ServiceClient, flavorName string) (string, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetFlavorIdWithName()")

	allPages, err := flavors.ListDetail(client, nil).AllPages()
	if err != nil {
		return "", err
	}
	flavorList, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		return "", err
	}
	for _, flavor := range flavorList {
		if flavor.Name == flavorName {
			return flavor.ID, nil
		}
	}
	return "", fmt.Errorf("Failed to Find the Flavor with the Name [%s]", flavorName)
}

func getSGWithName(networkClient *ktvpcsdk.ServiceClient, securityName string) (*secgroups.SecurityGroup, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetSGWithName()")

	allPages, err := secgroups.List(networkClient).AllPages()
	if err != nil {
		return nil, err
	}
	secGroupList, err := secgroups.ExtractSecurityGroups(allPages)
	if err != nil {
		return nil, err
	}
	for _, sg := range secGroupList {
		if strings.EqualFold(sg.Name, securityName) {
			return &sg, nil
		}
	}

	return nil, fmt.Errorf("Failed to Find the SecurityGroup with the Name [%s]", securityName)
}

func getNetworkWithName(networkClient *ktvpcsdk.ServiceClient, networkName string) (*networks.Network, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetNetworkWithName()")

	allPages, err := networks.List(networkClient, networks.ListOpts{Name: networkName}).AllPages()
	if err != nil {
		return nil, err
	}
	netList, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return nil, err
	}
	for _, net := range netList {
		if strings.EqualFold(net.Name, networkName) {
			return &net, nil
		}
	}

	return nil, fmt.Errorf("Failed to Find KT Cloud Network Info with the name [%s]", networkName)
}

func getSubnetWithId(networkClient *ktvpcsdk.ServiceClient, subnetId string) (*subnets.Subnet, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetSubnetWithId()")

	subnet, err := subnets.Get(networkClient, subnetId).Extract()
	if err != nil {
		return nil, err
	}
	return subnet, nil
}

func getPortWithDeviceId(networkClient *ktvpcsdk.ServiceClient, deviceID string) (*ports.Port, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetPortWithDeviceId()")
	
	allPages, err := ports.List(networkClient, ports.ListOpts{}).AllPages()
	if err != nil {
		return nil, err
	}
	portList, err := ports.ExtractPorts(allPages)
	if err != nil {
		return nil, err
	}
	for _, port := range portList {
		if strings.EqualFold(port.DeviceID, deviceID) {
			return &port, nil
		}
	}

	return nil, fmt.Errorf("Failed to Find Port with the DeviceID [%s]", deviceID)
}

func checkFolderAndCreate(folderPath string) error {
	cblogger.Info("KT Cloud VPC Driver: called CheckFolderAndCreate()")

	// If the Folder doesn't Exist, Create it
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		if err := os.Mkdir(folderPath, 0700); err != nil {
			return err
		}
	}
	return nil
}

func reverse(s string) (result string) {
	for _,v := range s {
		result = string(v) + result
	}
	return 
}

// Convert Cloud Object to JSON String type
func convertJsonString(v interface{}) (string, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert Json to String. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	return string(jsonBytes), nil
}

// Convert time to KTC
func convertTimeToKTC(givenTime time.Time) (time.Time, error) {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert the Time to KTC. [%v]", err)
		cblogger.Error(newErr.Error())
		return givenTime, newErr
	}
	return givenTime.In(loc), nil
}

func ipToCidr32(ipStr string) (string, error) {
    ip := net.ParseIP(ipStr)
    if ip == nil {
        return "", fmt.Errorf("Invalid IP address!!")
    }

    // Assuming IPv4 and /24 subnet
    mask := net.CIDRMask(32, 32) // for ~/32 subnet
	network := ip.Mask(mask)
    return fmt.Sprintf("%s/32", network), nil
}

func ipToCidr24(ipStr string) (string, error) {
    ip := net.ParseIP(ipStr)
    if ip == nil {
        return "", fmt.Errorf("Invalid IP address!!")
    }

    // Assuming IPv4 and /24 subnet
	mask := net.CIDRMask(24, 32) // for ~/24 subnet
    network := ip.Mask(mask)
	return fmt.Sprintf("%s/24", network), nil
}

func getSeoulCurrentTime() string {
	loc, _ := time.LoadLocation("Asia/Seoul")
	currentTime := time.Now().In(loc)	
	return currentTime.Format("2006-01-02 15:04:05")
}
