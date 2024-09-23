// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud VPC Handler
//
// by ETRI, 2021.05.

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

func (VPCHandler *KtCloudVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger.Info("KT Cloud Cloud Driver: called CreateVPC()!")
	// Check if the VPC Name already Exists
	vpcInfo, _ := VPCHandler.GetVPC(irs.IID{SystemId: vpcReqInfo.IId.NameId})

	if vpcInfo.IId.SystemId != "" {
		cblogger.Error("The VPC already exists .")
		return irs.VPCInfo{}, errors.New("The VPC already exists.")
	}

	zoneId := VPCHandler.RegionInfo.Zone
	if zoneId == "" {
		cblogger.Error("Failed to Get Zone info. from the connection info.")
		return irs.VPCInfo{}, errors.New("Failed to Get Zone info. from the connection info.")
	} else {
		cblogger.Infof("ZoneId : %s", zoneId)
	}

	var subnetList []irs.SubnetInfo
	for _, curSubnet := range vpcReqInfo.SubnetInfoList {
		cblogger.Infof("Subnet NameId : %s", curSubnet.IId.NameId)

		newSubnet, subnetErr := VPCHandler.CreateSubnet(curSubnet)
		if subnetErr != nil {
			return irs.VPCInfo{}, subnetErr
		}
		subnetList = append(subnetList, newSubnet)
	}

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
		},
	}
	// spew.Dump(newVpcInfo)

	vpcPath := os.Getenv("CBSPIDER_ROOT") + vpcDir
	vpcFilePath := vpcPath + zoneId + "/"
	jsonFileName := vpcFilePath + vpcReqInfo.IId.NameId + ".json"
	cblogger.Infof("jsonFileName to Create : " + jsonFileName)

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
	cblogger.Infof("Succeeded in writing the VPC info file: " + jsonFileName)
	cblogger.Info("Succeeded in Creating the VPC : " + newVpcInfo.IId.NameId)

	// Because it's managed as a file, there's no SystemId created.
	// Return the created SecurityGroup info.
	vpcInfo, vpcErr := VPCHandler.GetVPC(irs.IID{SystemId: vpcReqInfo.IId.NameId})
	if vpcErr != nil {
		cblogger.Error("Failed to Get the VPC info.")
		return irs.VPCInfo{}, vpcErr
	}
	return vpcInfo, nil
}

func (VPCHandler *KtCloudVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	cblogger.Info("KT Cloud Cloud Driver: called GetVPC()!")

	//Caution!!
	if vpcIID.SystemId != "" {
		vpcIID.NameId = vpcIID.SystemId
	}

	zoneId := VPCHandler.RegionInfo.Zone
	if zoneId == "" {
		cblogger.Error("Failed to Get Zone info. from the connection info.")

		return irs.VPCInfo{}, errors.New("Failed to Get Zone info. from the connection info.")
	} else {
		cblogger.Infof("ZoneId : %s", zoneId)
	}

	vpcPath := os.Getenv("CBSPIDER_ROOT") + vpcDir
	vpcFilePath := vpcPath + zoneId + "/"
	jsonFileName := vpcFilePath + vpcIID.NameId + ".json"

	cblogger.Infof("vpcIID.NameId : " + vpcIID.NameId)
	cblogger.Infof("jsonFileName : " + jsonFileName)

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
		cblogger.Error("Failed to Find the VPC file : "+jsonFileName+" ", err)
		cblogger.Error("Failed to Find the VPC info!!")
		return irs.VPCInfo{}, err
	}
	cblogger.Infof("Succeeded in Finding and Opening the S/G file: " + jsonFileName)
	defer jsonFile.Close()

	byteValue, readErr := io.ReadAll(jsonFile)
	if readErr != nil {
		cblogger.Error("Failed to Read the S/G file : "+jsonFileName, readErr)
	}

	var vpcFileInfo VPCFileInfo
	json.Unmarshal(byteValue, &vpcFileInfo)

	vpcInfo, vpcInfoError := VPCHandler.mappingVPCInfo(vpcFileInfo)
	if vpcInfoError != nil {
		cblogger.Error(vpcInfoError)
		return irs.VPCInfo{}, vpcInfoError
	}
	return vpcInfo, nil
}

func (VPCHandler *KtCloudVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger.Info("KT Cloud Cloud Driver: called ListVPC()!")

	zoneId := VPCHandler.RegionInfo.Zone
	if zoneId == "" {
		cblogger.Error("Failed to Get Zone info. from the connection info.")

		return []*irs.VPCInfo{}, errors.New("Failed to Get Zone info. from the connection info.")
	} else {
		cblogger.Infof("ZoneId : %s", zoneId)
	}

	vpcFilePath := os.Getenv("CBSPIDER_ROOT") + vpcDir + zoneId + "/"
	// File list on the local directory
	dirFiles, readRrr := os.ReadDir(vpcFilePath)
	if readRrr != nil {
		return []*irs.VPCInfo{}, readRrr
	}

	var vpcIID irs.IID
	var vpcInfoList []*irs.VPCInfo

	for _, file := range dirFiles {
		fileName := strings.TrimSuffix(file.Name(), ".json") // 접미사 제거
		vpcIID.NameId = fileName
		cblogger.Infof("# VPC Name : " + vpcIID.NameId)

		vpcInfo, getVpcErr := VPCHandler.GetVPC(irs.IID{SystemId: vpcIID.NameId})
		if getVpcErr != nil {
			cblogger.Errorf("Failed to Find the VPC : %s", vpcIID.SystemId)
			return []*irs.VPCInfo{}, getVpcErr
		}
		vpcInfoList = append(vpcInfoList, &vpcInfo)
	}
	return vpcInfoList, nil
}

func (VPCHandler *KtCloudVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Cloud Driver: called DeleteVPC()!")

	if vpcIID.SystemId != "" {
		vpcIID.NameId = vpcIID.SystemId
	}

	//To check whether the VPC exists.
	_, getVpcErr := VPCHandler.GetVPC(irs.IID{SystemId: vpcIID.NameId})
	if getVpcErr != nil {
		cblogger.Errorf("Failed to Find the VPC : %s", vpcIID.NameId)
		return false, getVpcErr
	}

	zoneId := VPCHandler.RegionInfo.Zone
	if zoneId == "" {
		cblogger.Error("Failed to Get Zone info. from the connection info.")
		return false, errors.New("Failed to Get Zone info. from the connection info.")
	} else {
		cblogger.Infof("ZoneId : %s", zoneId)
	}

	vpcPath := os.Getenv("CBSPIDER_ROOT") + vpcDir
	vpcFilePath := vpcPath + zoneId + "/"
	jsonFileName := vpcFilePath + vpcIID.NameId + ".json"

	cblogger.Infof("VPC info file to Delete : [%s]", jsonFileName)

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
	cblogger.Infof("Succeeded in Deleting the VPC : " + vpcIID.NameId)

	return true, nil
}

func (VPCHandler *KtCloudVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called AddSubnet()!!")
	return irs.VPCInfo{}, errors.New("Does not support AddSubnet() yet!!")
}

func (VPCHandler *KtCloudVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud cloud driver: called GetImage()!!")
	return true, errors.New("Does not support RemoveSubnet() yet!!")
}

func (VPCHandler *KtCloudVPCHandler) CreateSubnet(subnetReqInfo irs.SubnetInfo) (irs.SubnetInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called CreateSubnet()!!")

	// zoneId := VPCHandler.RegionInfo.Zone
	// cblogger.Infof("ZoneId : %s", zoneId)
	// if zoneId == "" {
	// 	cblogger.Error("Failed to Get Zone info. from the connection info.")
	// 	return irs.SubnetInfo{}, errors.New("Failed to Get Zone info. from the connection info.")
	// }

	newSubnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId: subnetReqInfo.IId.NameId,
			// Caution!! : subnetReqInfo.IId.NameId -> SystemId
			SystemId: subnetReqInfo.IId.NameId,
		},
		Zone:      subnetReqInfo.Zone,
		IPv4_CIDR: "N/A",
		KeyValueList: []irs.KeyValue{
			{Key: "KTCloud-Subnet-info.", Value: "This Subne info. is temporary."},
		},
	}
	return newSubnetInfo, nil
}

func (VPCHandler *KtCloudVPCHandler) mappingVPCInfo(vpcFileInfo VPCFileInfo) (irs.VPCInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called mappingVPCInfo()!!")

	var subnetInfoList []irs.SubnetInfo
	var subnetInfo irs.SubnetInfo
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
		//KT Cloud의 VPC 정보는 CB에서 파일로 관리되므로 SystemId는 NameId와 동일하게
		IId:            irs.IID{NameId: vpcFileInfo.IID.NameID, SystemId: vpcFileInfo.IID.NameID},
		IPv4_CIDR:      vpcFileInfo.Cidr,
		SubnetInfoList: subnetInfoList,
		KeyValueList:   vpcKeyValueList,
	}
	return vpcInfo, nil
}

func (vpcHandler *KtCloudVPCHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}
