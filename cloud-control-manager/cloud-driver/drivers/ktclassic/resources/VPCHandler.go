// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud VPC Handler
//
// by ETRI, 2021.05.
// Updated by ETRI, 2025.02.

package resources

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"

	// "encoding/base64"
	"fmt"
	// "github.com/davecgh/go-spew/spew"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"
)

type KtCloudVPCHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	Client         *ktsdk.KtCloudClient
}

const (
	vpcDir string = "/cloud-driver-libs/.vpc-kt/"
)

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud VPC Handler")
}

type VPCFileInfo struct {
	IID           IId        `json:"IId"`
	Cidr          string     `json:"IPv4_CIDR"`
	Subnet_List   []Subnet   `json:"SubnetInfoList"`
	KeyValue_List []KeyValue `json:"KeyValueList"`
}

type Subnet struct {
	IID           IId        `json:"IId"`
	Zone          string     `json:"Zone"`
	Cidr          string     `json:"IPv4_CIDR"`
	KeyValue_List []KeyValue `json:"KeyValueList"`
}

type KeyValue struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

func (vpcHandler *KtCloudVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger.Info("KT Classic Driver: called CreateVPC()!")
	// Check if the VPC Name already Exists
	vpcInfo, checkErr := vpcHandler.GetVPC(irs.IID{SystemId: vpcReqInfo.IId.NameId})
	if checkErr != nil {
		if strings.Contains(checkErr.Error(), "does not exist") {
			cblogger.Info("The VPC does not exist.")
		} else {
			cblogger.Error("Failed to Get the VPC info.")
			return irs.VPCInfo{}, checkErr
		}
	}

	if vpcInfo.IId.SystemId != "" {
		newErr := fmt.Errorf("The VPC already exists.")
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	if strings.EqualFold(vpcHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Failed to Get Zone info. from the connection info.")
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	var subnetList []irs.SubnetInfo
	for _, curSubnet := range vpcReqInfo.SubnetInfoList {
		cblogger.Infof("Subnet NameId : %s", curSubnet.IId.NameId)

		newSubnet, subnetErr := vpcHandler.CreateSubnet(curSubnet)
		if subnetErr != nil {
			return irs.VPCInfo{}, subnetErr
		}
		subnetList = append(subnetList, newSubnet)
	}

	currentTime := getSeoulCurrentTime()

	newVpcInfo := irs.VPCInfo{
		IId: irs.IID{
			NameId: vpcReqInfo.IId.NameId,
			// Caution!! : vpcReqInfo.IId.NameId -> SystemId
			SystemId: vpcReqInfo.IId.NameId,
		},
		IPv4_CIDR:      vpcReqInfo.IPv4_CIDR,
		SubnetInfoList: subnetList,
		KeyValueList: []irs.KeyValue{
			{Key: "KTCloud-VPC-info.", Value: "This VPC info. is temporary."},
			{Key: "CreateTime", Value: currentTime},
		},
	}
	// spew.Dump(newVpcInfo)

	vpcPath := os.Getenv("CBSPIDER_ROOT") + vpcDir
	vpcFilePath := vpcPath + vpcHandler.RegionInfo.Zone + "/"
	jsonFileName := vpcFilePath + vpcReqInfo.IId.NameId + ".json"
	// cblogger.Infof("jsonFileName to Create : " + jsonFileName)

	// Check if the VPC Folder Exists, and Create it
	if err := CheckFolderAndCreate(vpcPath); err != nil {
		newErr := fmt.Errorf("Failed to Create the VPC File Dir : %s, [%v]"+vpcPath, err)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	// Check if the VPC Folder Exists, and Create it
	if err := CheckFolderAndCreate(vpcFilePath); err != nil {
		newErr := fmt.Errorf("Failed to Create the VPC File Path : %s, [%v]"+vpcFilePath, err)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	// Write 'newVpcInfo' to a JSON File
	file, _ := json.MarshalIndent(newVpcInfo, "", " ")
	writeErr := os.WriteFile(jsonFileName, file, 0644)
	if writeErr != nil {
		cblogger.Error("Failed to write the file: "+jsonFileName, writeErr)
		return irs.VPCInfo{}, writeErr
	}
	// cblogger.Infof("Succeeded in writing the VPC info file: " + jsonFileName)
	// cblogger.Info("Succeeded in Creating the VPC : " + newVpcInfo.IId.NameId)

	// # There's no SystemId created because it's managed as a file.
	vpcInfo, vpcErr := vpcHandler.GetVPC(irs.IID{SystemId: vpcReqInfo.IId.NameId})
	if vpcErr != nil {
		cblogger.Error("Failed to Get the VPC info.")
		return irs.VPCInfo{}, vpcErr
	}
	return vpcInfo, nil
}

func (vpcHandler *KtCloudVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	cblogger.Info("KT Classic Driver: called GetVPC()!")

	// Caution!!
	if vpcIID.SystemId != "" {
		vpcIID.NameId = vpcIID.SystemId
	}

	if strings.EqualFold(vpcHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Failed to Get Zone info. from the connection info.")
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	vpcPath := os.Getenv("CBSPIDER_ROOT") + vpcDir
	vpcFilePath := vpcPath + vpcHandler.RegionInfo.Zone + "/"
	jsonFileName := vpcFilePath + vpcIID.NameId + ".json"

	// cblogger.Infof("vpcIID.NameId : " + vpcIID.NameId)
	// cblogger.Infof("jsonFileName : " + jsonFileName)

	// Check if the VPC Folder Exists, and Create it
	if err := CheckFolderAndCreate(vpcPath); err != nil {
		newErr := fmt.Errorf("Failed to Create the VPC File Dir : %s, [%v]"+vpcPath, err)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	// Check if the VPC Folder Exists, and Create it
	if err := CheckFolderAndCreate(vpcFilePath); err != nil {
		newErr := fmt.Errorf("Failed to Create the VPC File Path : %s, [%v]"+vpcFilePath, err)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	jsonFile, err := os.Open(jsonFileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cblogger.Info("Failed to Find the VPC file : "+jsonFileName+" ", err)
			err := fmt.Errorf("%s", "VPC:"+vpcIID.NameId+" does not exist!")
			return irs.VPCInfo{}, err
		}
		cblogger.Error("Failed to open the VPC file : "+jsonFileName+" ", err)
		return irs.VPCInfo{}, err
	}
	defer jsonFile.Close()
	// cblogger.Infof("Succeeded in Finding and Opening the S/G file: " + jsonFileName)

	byteValue, readErr := io.ReadAll(jsonFile)
	if readErr != nil {
		cblogger.Error("Failed to Read the S/G file : "+jsonFileName, readErr)
	}

	var vpcFileInfo VPCFileInfo
	json.Unmarshal(byteValue, &vpcFileInfo)

	vpcInfo, vpcInfoError := vpcHandler.mappingVPCInfo(vpcFileInfo)
	if vpcInfoError != nil {
		cblogger.Error(vpcInfoError)
		return irs.VPCInfo{}, vpcInfoError
	}
	return vpcInfo, nil
}

func (vpcHandler *KtCloudVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger.Info("KT Classic Driver: called ListVPC()!")

	if strings.EqualFold(vpcHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Failed to Get Zone info. from the connection info.")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	vpcFilePath := os.Getenv("CBSPIDER_ROOT") + vpcDir + vpcHandler.RegionInfo.Zone + "/"
	// File list on the local directory
	dirFiles, readRrr := os.ReadDir(vpcFilePath)
	if readRrr != nil {
		return nil, readRrr
	}

	var vpcInfoList []*irs.VPCInfo
	if len(dirFiles) > 0 {
		var vpcIID irs.IID
		for _, file := range dirFiles {
			vpcIID.NameId = strings.TrimSuffix(file.Name(), ".json") // Remove suffix from the string
			// cblogger.Infof("# VPC Name : " + vpcIID.NameId)

			vpcInfo, getVpcErr := vpcHandler.GetVPC(irs.IID{SystemId: vpcIID.NameId})
			if getVpcErr != nil {
				cblogger.Errorf("Failed to Get the VPC info : %s", vpcIID.SystemId)
				return nil, getVpcErr
			}
			vpcInfoList = append(vpcInfoList, &vpcInfo)
		}
		return vpcInfoList, nil
	}
	return nil, nil
}

func (vpcHandler *KtCloudVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	cblogger.Info("KT Classic Driver: called DeleteVPC()!")

	if vpcIID.SystemId != "" {
		vpcIID.NameId = vpcIID.SystemId
	}

	if strings.EqualFold(vpcHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Failed to Get Zone info. from the connection info.")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	//To check whether the VPC exists.
	_, getVpcErr := vpcHandler.GetVPC(irs.IID{SystemId: vpcIID.NameId})
	if getVpcErr != nil {
		cblogger.Errorf("Failed to Find the VPC : %s", vpcIID.NameId)
		return false, getVpcErr
	}

	vpcPath := os.Getenv("CBSPIDER_ROOT") + vpcDir
	vpcFilePath := vpcPath + vpcHandler.RegionInfo.Zone + "/"
	jsonFileName := vpcFilePath + vpcIID.NameId + ".json"
	// cblogger.Infof("VPC info file to Delete : [%s]", jsonFileName)

	// Check if the VPC Folder Exists, and Create it
	if err := CheckFolderAndCreate(vpcPath); err != nil {
		newErr := fmt.Errorf("Failed to Create the VPC File Dir : %s, [%v]"+vpcPath, err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	// Check if the VPC Folder Exists, and Create it
	if err := CheckFolderAndCreate(vpcFilePath); err != nil {
		newErr := fmt.Errorf("Failed to Create the VPC File Path : %s, [%v]"+vpcFilePath, err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	// To Remove the VPC file on the Local machine.
	delErr := os.Remove(jsonFileName)
	if delErr != nil {
		newErr := fmt.Errorf("Failed to Delete the file : %s, [%v]", jsonFileName, delErr)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	// cblogger.Infof("Succeeded in Deleting the VPC : " + vpcIID.NameId)

	return true, nil
}

func (vpcHandler *KtCloudVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	cblogger.Info("KT Classic driver: called AddSubnet()!!")
	return irs.VPCInfo{}, errors.New("Does not support AddSubnet() yet!!")
}

func (vpcHandler *KtCloudVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Info("KT Classic driver: called GetImage()!!")
	return true, errors.New("Does not support RemoveSubnet() yet!!")
}

func (vpcHandler *KtCloudVPCHandler) CreateSubnet(subnetReqInfo irs.SubnetInfo) (irs.SubnetInfo, error) {
	cblogger.Info("KT Classic driver: called CreateSubnet()!!")

	if strings.EqualFold(vpcHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Failed to Get Zone info. from the connection info.")
		cblogger.Error(newErr.Error())
		return irs.SubnetInfo{}, newErr
	}

	var zoneInfo string
	if !strings.EqualFold(subnetReqInfo.Zone, "") {
		zoneInfo = subnetReqInfo.Zone
	} else {
		zoneInfo = vpcHandler.RegionInfo.Zone
	}
	// cblogger.Infof("# zoneInfo : %s", zoneInfo)

	currentTime := getSeoulCurrentTime()

	newSubnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId: subnetReqInfo.IId.NameId,
			// Caution!! : subnetReqInfo.IId.NameId -> SystemId
			SystemId: subnetReqInfo.IId.NameId,
		},
		Zone:      zoneInfo,
		IPv4_CIDR: "N/A",
		KeyValueList: []irs.KeyValue{
			{Key: "KTCloud-Subnet-info.", Value: "This Subne info. is temporary."},
			{Key: "CreateTime", Value: currentTime},
		},
	}
	return newSubnetInfo, nil
}

func (vpcHandler *KtCloudVPCHandler) mappingVPCInfo(vpcFileInfo VPCFileInfo) (irs.VPCInfo, error) {
	cblogger.Info("KT Classic driver: called mappingVPCInfo()!!")

	var subnetInfo irs.SubnetInfo
	var subnetInfoList []irs.SubnetInfo
	var subnetKeyValue irs.KeyValue
	var subnetKeyValueList []irs.KeyValue
	var vpcKeyValue irs.KeyValue
	var vpcKeyValueList []irs.KeyValue

	for i := 0; i < len(vpcFileInfo.Subnet_List); i++ {
		subnetInfo.IId.NameId = vpcFileInfo.Subnet_List[i].IID.NameID
		subnetInfo.IId.SystemId = vpcFileInfo.Subnet_List[i].IID.SystemID
		subnetInfo.Zone = vpcFileInfo.Subnet_List[i].Zone
		subnetInfo.IPv4_CIDR = vpcFileInfo.Subnet_List[i].Cidr

		for j := 0; j < len(vpcFileInfo.Subnet_List[i].KeyValue_List); j++ {
			subnetKeyValue.Key = vpcFileInfo.Subnet_List[i].KeyValue_List[j].Key
			subnetKeyValue.Value = vpcFileInfo.Subnet_List[i].KeyValue_List[j].Value

			subnetKeyValueList = append(subnetKeyValueList, subnetKeyValue)
		}
		subnetInfo.KeyValueList = subnetKeyValueList
		subnetInfoList = append(subnetInfoList, subnetInfo)
	}

	for k := 0; k < len(vpcFileInfo.KeyValue_List); k++ {
		vpcKeyValue.Key = vpcFileInfo.KeyValue_List[k].Key
		vpcKeyValue.Value = vpcFileInfo.KeyValue_List[k].Value
		vpcKeyValueList = append(vpcKeyValueList, vpcKeyValue)
	}

	vpcInfo := irs.VPCInfo{
		// Since KT Cloud's VPC information is managed as a file in CB, SystemId is the same as NameId.
		IId:            irs.IID{NameId: vpcFileInfo.IID.NameID, SystemId: vpcFileInfo.IID.NameID},
		IPv4_CIDR:      vpcFileInfo.Cidr,
		SubnetInfoList: subnetInfoList,
		KeyValueList:   vpcKeyValueList,
	}
	return vpcInfo, nil
}

func (vpcHandler *KtCloudVPCHandler) getSubnetZone(vpcIID irs.IID, subnetIID irs.IID) (string, error) {
	cblogger.Info("KT Classic driver: called getSubnetZone()!!")

	if strings.EqualFold(vpcIID.SystemId, "") && strings.EqualFold(vpcIID.NameId, "") {
		newErr := fmt.Errorf("Invalid VPC Id!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	if strings.EqualFold(subnetIID.SystemId, "") && strings.EqualFold(subnetIID.NameId, "") {
		newErr := fmt.Errorf("Invalid Subnet Id!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	// Get the VPC information
	vpcInfo, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VPC Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	//  cblogger.Info("\n\n### vpcInfo : ")
	//  spew.Dump(vpcInfo)
	//  cblogger.Info("\n")

	// Get the Zone info of the specified Subnet
	var subnetZone string
	for _, subnet := range vpcInfo.SubnetInfoList {
		if strings.EqualFold(subnet.IId.SystemId, subnetIID.SystemId) {
			subnetZone = subnet.Zone
			break
		}
	}
	if strings.EqualFold(subnetZone, "") {
		newErr := fmt.Errorf("Failed to Get the Zone info of the specified Subnet!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	return subnetZone, nil
}

func (vpcHandler *KtCloudVPCHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("KT Classic Driver: called ListIID()!")

	if strings.EqualFold(vpcHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Failed to Get Zone info. from the connection info.")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	vpcFilePath := os.Getenv("CBSPIDER_ROOT") + vpcDir + vpcHandler.RegionInfo.Zone + "/"
	// File list on the local directory
	dirFiles, readRrr := os.ReadDir(vpcFilePath)
	if readRrr != nil {
		return nil, readRrr
	}

	// Since KT Cloud's VPC information is managed as a file in CB, SystemId is the same as NameId.
	var iidList []*irs.IID
	if len(dirFiles) > 0 {
		for _, file := range dirFiles {
			vpcId := strings.TrimSuffix(file.Name(), ".json") // Remove suffix from the string
			iidList = append(iidList, &irs.IID{NameId: vpcId, SystemId: vpcId})
		}
		return iidList, nil
	}
	return nil, nil
}
