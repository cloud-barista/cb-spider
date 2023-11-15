// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, Innogrid, 2021.12.

package resources

import (
	"os"
	"os/exec"
	// "errors"
	"fmt"
	"encoding/json"

	"strings"
	"sync"
	"time"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/extensions/secgroups"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/flavors"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/extensions/external"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/networks"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/ports"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/subnets"
	
	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
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
	cblogger.Error(err.Error())
	hiscallInfo.ErrorMSG = err.Error()
	calllogger.Error(call.String(hiscallInfo))
}

func LoggingInfo(hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) {
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
}

func GetCallLogScheme(zone string, resourceType call.RES_TYPE, resourceName string, apiName string) call.CLOUDLOGSCHEMA {
	cblogger.Info(fmt.Sprintf("Call %s %s", call.NHNCLOUD, apiName))

	return call.CLOUDLOGSCHEMA{
		CloudOS:      call.NHNCLOUD,
		RegionZone:   zone,
		ResourceType: resourceType,
		ResourceName: resourceName,
		CloudOSAPI:   apiName,
	}
}

func logAndReturnError(callLogInfo call.CLOUDLOGSCHEMA, givenErrString string, v interface{}) (error) {
	newErr := fmt.Errorf(givenErrString + " %v", v)
	cblogger.Error(newErr.Error())
	LoggingError(callLogInfo, newErr)
	return newErr
}

func GetPublicVPCInfo(client *nhnsdk.ServiceClient, typeName string) (string, error) {
	cblogger.Info("NHN Cloud Driver: called GetPublicVPCInfo()")

	exTrue := true
	listOpts := external.ListOptsExt{
		ListOptsBuilder: networks.ListOpts{},
		External:        &exTrue,
	}
	allPages, err := networks.List(client, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VPC Pages. %s", err.Error())
		cblogger.Error(newErr)
		return "", newErr
	}

	// external VPC 필터링
	var extVpcList []NetworkWithExt
	err = networks.ExtractNetworksInto(allPages, &extVpcList)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VPC list. %s", err.Error())
		cblogger.Error(newErr)
		return "", newErr
	}
	if len(extVpcList) == 0 {
		newErr := fmt.Errorf("Failed to Get VPC list, external VPC not exist")
		cblogger.Error(newErr)
		return "", newErr
	}

	extVpc := extVpcList[0]

	if strings.EqualFold(typeName, "ID") {
		return extVpc.ID, nil
	} else if strings.EqualFold(typeName, "NAME") {
		return extVpc.Name, nil
	}

	return "", nil
}

func GetVMSpecIdWithName(client *nhnsdk.ServiceClient, flavorName string) (string, error) {
	cblogger.Info("NHN Cloud Driver: called GetVMSpecIdWithName()")

	allPages, err := flavors.ListDetail(client, nil).AllPages()
	if err != nil {
		return "", err
	}
	nhnFlavorList, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		return "", err
	}

	for _, nhnFlavor := range nhnFlavorList {
		if strings.EqualFold(nhnFlavor.Name, flavorName) {
			return nhnFlavor.ID, nil
		}
	}

	return "", fmt.Errorf("Failed to Find Flavor with the name [%s]", flavorName)
}

func GetSGWithName(networkClient *nhnsdk.ServiceClient, securityGroupName string) (*secgroups.SecurityGroup, error) {
	cblogger.Info("NHN Cloud Driver: called GetSGWithName()")

	allPages, err := secgroups.List(networkClient).AllPages()
	if err != nil {
		return nil, err
	}
	nhnSGList, err := secgroups.ExtractSecurityGroups(allPages)
	if err != nil {
		return nil, err
	}

	for _, nhnSG := range nhnSGList {
		if strings.EqualFold(nhnSG.Name, securityGroupName) {
			return &nhnSG, nil
		}
	}

	return nil, fmt.Errorf("Failed to Find SecurityGroups with the name [%s]", securityGroupName)
}

func GetNetworkWithName(networkClient *nhnsdk.ServiceClient, networkName string) (*networks.Network, error) {
	cblogger.Info("NHN Cloud Driver: called GetNetworkWithName()")

	allPages, err := networks.List(networkClient, networks.ListOpts{Name: networkName}).AllPages()
	if err != nil {
		return nil, err
	}
	nhnNetList, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return nil, err
	}

	for _, nhnNetwork := range nhnNetList {
		if strings.EqualFold(nhnNetwork.Name, networkName) {
			return &nhnNetwork, nil
		}
	}

	return nil, fmt.Errorf("Failed to Find SecurityGroups Info with name [%s]", networkName)
}

func GetSubnetWithId(networkClient *nhnsdk.ServiceClient, subnetId string) (*subnets.Subnet, error) {
	cblogger.Info("NHN Cloud Driver: called GetSubnetWithId()")

	nhnSubnet, err := subnets.Get(networkClient, subnetId).Extract()
	if err != nil {
		return nil, err
	}

	return nhnSubnet, nil
}

func GetPortWithDeviceId(networkClient *nhnsdk.ServiceClient, deviceID string) (*ports.Port, error) {
	cblogger.Info("NHN Cloud Driver: called GetPortWithDeviceId()")

	allPages, err := ports.List(networkClient, ports.ListOpts{}).AllPages()
	if err != nil {
		return nil, err
	}
	nhnPortList, err := ports.ExtractPorts(allPages)
	if err != nil {
		return nil, err
	}

	for _, nhnPort := range nhnPortList {
		if strings.EqualFold(nhnPort.DeviceID, deviceID) {
			return &nhnPort, nil
		}
	}

	return nil, fmt.Errorf("Failed to Get Port Info. with the DeviceID [%s]", deviceID)
}

func CheckIIDValidation(IId irs.IID) bool {
	if strings.EqualFold(IId.NameId, "") && strings.EqualFold(IId.SystemId, "") {
		newErr := fmt.Errorf("Invalid NameId and SystemId!!")
		cblogger.Error(newErr.Error())
		return false
	}
	return true
}

func CheckFolderAndCreate(folderPath string) error {
	// Check if the Folder Exists and Create it
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		if err := os.Mkdir(folderPath, 0700); err != nil {
			return err
		}
	}
	return nil
}

func GetOriginalNameId(IID2NameId string) string {
	var originalNameId string
	
	if len(IID2NameId) <= 9 {  	// For local test
		originalNameId = IID2NameId
	} else { 					// For CB-Spider IID2 NameId
		reversedNameId := Reverse(IID2NameId)
		originalNameId = reversedNameId[:21]
		originalNameId = strings.TrimSuffix(IID2NameId, Reverse(originalNameId))	
	}
	cblogger.Infof("# originalNameId : %s", originalNameId)
	return originalNameId
}

func Reverse(s string) (result string) {
	for _,v := range s {
		result = string(v) + result
	}
	return 
}

func RunCommand(cmdName string, cmdArgs []string) (string, error) {

	/*
	Ref)
	var (
		cmdOut []byte
		cmdErr   error		
	)
	*/

	cblogger.Infof("cmdName : %s", cmdName)
	cblogger.Infof("cmdArgs : %s", cmdArgs)

	//if cmdOut, cmdErr = exec.Command(cmdName, cmdArgs...).Output(); cmdErr != nil {
	if cmdOut, cmdErr := exec.Command(cmdName, cmdArgs...).CombinedOutput(); cmdErr != nil {
		fmt.Fprintln(os.Stderr, "There was an Error running command : ", cmdErr)
		//panic("Can't exec the command: " + cmdErr1.Error())
		fmt.Println(fmt.Sprint(cmdErr) + ": " + string(cmdOut))
		os.Exit(1)

		return string(cmdOut), cmdErr
	} else {
	fmt.Println("cmdOut : ", string(cmdOut))

	return string(cmdOut), nil
	}
}

// Convert Cloud Object to JSON String type
func ConvertJsonString(v interface{}) (string, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert Json to String. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	jsonString := string(jsonBytes)
	return jsonString, nil
}
