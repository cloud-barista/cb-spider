// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP VPC Handler
//
// by ETRI, 2021.12.
// Updated by ETRI, 2023.10.

package resources

import (
	"os"
	"io"
	"fmt"
	"encoding/json"
	"errors"
	"strings"

	// "github.com/davecgh/go-spew/spew"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/server"
	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVPCHandler struct {
	CredentialInfo 		idrv.CredentialInfo
	RegionInfo     		idrv.RegionInfo
	VMClient         	*server.APIClient
}

const (
	vpcDir string =  "/cloud-driver-libs/.vpc-ncp/"
)

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP VPC Handler")
}

type VPC struct {
    IID				IId     	`json:"IId"`
    Cidr			string  	`json:"IPv4_CIDR"`
    Subnet_List 	[]Subnet 	`json:"SubnetInfoList"`
    KeyValue_List 	[]KeyValue  `json:"KeyValueList"`
}

type Subnet struct {
    IID				IId     `json:"IId"`
	Zone        	string     `json:"Zone"`
    Cidr			string	`json:"IPv4_CIDR"`
    KeyValue_List 	[]KeyValue `json:"KeyValueList"`
}

type KeyValue struct {
    Key			string	`json:"Key"`
    Value		string	`json:"Value"`
}

type IId struct {
    NameID   	string 		`json:"NameId"`
    SystemID   	string 		`json:"SystemId"`
}

func (VPCHandler *NcpVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger.Info("NCP Cloud Driver: called CreateVPC()!")

	// Check if the VPC Name already Exists
	vpcInfo, _ := VPCHandler.GetVPC(irs.IID{SystemId: vpcReqInfo.IId.NameId}) 	// '_' : For when there is no VPC.
	
	if vpcInfo.IId.SystemId != "" {
		newErr := fmt.Errorf("The VPC already exists.")
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	zoneId := VPCHandler.RegionInfo.Zone
	if zoneId == "" {
		newErr := fmt.Errorf("Failed to Get Zone info. from the connection info..")
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
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
			NameId:   vpcReqInfo.IId.NameId,
			// Note!! : vpcReqInfo.IId.NameId -> SystemId
			SystemId: vpcReqInfo.IId.NameId,
		},
		IPv4_CIDR: vpcReqInfo.IPv4_CIDR,
		SubnetInfoList: subnetList,
		KeyValueList: []irs.KeyValue{
			{Key: "NCP-VPC-info.", Value: "This VPC info. is temporary."},
		},
	}
	// spew.Dump(newVpcInfo)

	vpcPath := os.Getenv("CBSPIDER_ROOT") + vpcDir	
	vpcFilePath := vpcPath + zoneId + "/"
	jsonFileName := vpcFilePath + vpcReqInfo.IId.NameId + ".json"
	// cblogger.Infof("jsonFileName to Create : [%s]", jsonFileName)

	// Check if the VPC Folder Exists, and Create it
	if err := CheckFolderAndCreate(vpcPath); err != nil {
		newErr := fmt.Errorf("Failed to Create the VPC File Dir : %s, [%v]" + vpcPath, err)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	// Check if the VPC Folder Exists, and Create it
	if err := CheckFolderAndCreate(vpcFilePath); err != nil {
		newErr := fmt.Errorf("Failed to Create the VPC File Path : %s, [%v]" + vpcFilePath, err)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	// Write 'newVpcInfo' to a JSON File
	file, _ := json.MarshalIndent(newVpcInfo, "", " ")
	writeErr := os.WriteFile(jsonFileName, file, 0644)
	if writeErr != nil {
		newErr := fmt.Errorf("Failed to write the file : %s, [%v]", jsonFileName, writeErr)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}
	// cblogger.Infof("Succeeded in writing the VPC info on file : [%s]", jsonFileName)
	// cblogger.Infof("Succeeded in Creating the VPC : [%s]", newVpcInfo.IId.NameId)

	// Because it's managed as a file, there's no SystemId created.
	// Return the created SecurityGroup info.
	vpcInfo, vpcErr := VPCHandler.GetVPC(irs.IID{SystemId: vpcReqInfo.IId.NameId})
	if vpcErr != nil {
		newErr := fmt.Errorf("Failed to Get the VPC Info : [%v]", vpcErr)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}
	return vpcInfo, nil
}

func (VPCHandler *NcpVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	cblogger.Info("NCP Cloud Driver: called GetVPC()!")

	// Note!!
	if vpcIID.SystemId != "" {
		vpcIID.NameId = vpcIID.SystemId
    }
	
	zoneId := VPCHandler.RegionInfo.Zone
	if zoneId == "" {
		newErr := fmt.Errorf("Failed to Get Zone info. from the connection info.")
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	} else {
		cblogger.Infof("ZoneId : %s", zoneId)
	}

	vpcPath := os.Getenv("CBSPIDER_ROOT") + vpcDir	
	vpcFilePath := vpcPath + zoneId + "/"
	jsonFileName := vpcFilePath + vpcIID.NameId + ".json"

	// cblogger.Infof("vpcIID.NameId : %s", vpcIID.NameId)
	// cblogger.Infof("jsonFileName : %s", jsonFileName)

	// Check if the VPC Folder Exists, and Create it
	if err := CheckFolderAndCreate(vpcPath); err != nil {
		newErr := fmt.Errorf("Failed to Create the VPC File Dir : %s, [%v]" + vpcPath, err)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	// Check if the VPC Folder Exists, and Create it
	if err := CheckFolderAndCreate(vpcFilePath); err != nil {
		newErr := fmt.Errorf("Failed to Create the VPC File Path : %s, [%v]" + vpcFilePath, err)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}

	jsonFile, err := os.Open(jsonFileName)
    if err != nil {
		newErr := fmt.Errorf("Failed to Find the VPC file : %s, [%v]"+ jsonFileName +" ", err)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
    }
    defer jsonFile.Close()
	// cblogger.Info("Succeeded in Finding and Opening the VPC file: "+ jsonFileName)
	
	var vpcJSON VPC
	byteValue, readErr := io.ReadAll(jsonFile)
	if readErr != nil {
		newErr := fmt.Errorf("Failed to Read the VPC file : %s, [%v]"+ jsonFileName, readErr)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
    }
    json.Unmarshal(byteValue, &vpcJSON)

	vpcInfo, vpcInfoErr := VPCHandler.MappingVPCInfo(vpcJSON)
	if vpcInfoErr != nil {
		newErr := fmt.Errorf("Failed to Map the VPC Info : [%v]", vpcInfoErr)
		cblogger.Error(newErr.Error())
		return irs.VPCInfo{}, newErr
	}
	return vpcInfo, nil
}

func (VPCHandler *NcpVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger.Info("NCP Cloud Driver: called ListVPC()!")

	var vpcIID irs.IID
	var vpcInfoList []*irs.VPCInfo

	zoneId := VPCHandler.RegionInfo.Zone
	if zoneId == "" {
		newErr := fmt.Errorf("Failed to Get Zone info. from the connection info.")
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Infof("ZoneId : %s", zoneId)
	}

	vpcFilePath := os.Getenv("CBSPIDER_ROOT") + vpcDir + zoneId + "/"
	// File list on the local directory 
	dirFiles, readErr := os.ReadDir(vpcFilePath)
	if readErr != nil {
		newErr := fmt.Errorf("Failed to Read the VPC file : [%v]", readErr)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	for _, file := range dirFiles {
		fileName := strings.TrimSuffix(file.Name(), ".json")  // 접미사 제거
		vpcIID.NameId = fileName
		cblogger.Infof("# VPC Name : " + vpcIID.NameId)

		vpcInfo, vpcErr := VPCHandler.GetVPC(irs.IID{SystemId: vpcIID.NameId})
		if vpcErr != nil {
			newErr := fmt.Errorf("Failed to Get the VPC Info : [%v]", vpcErr)
			cblogger.Error(newErr.Error())
			return nil, newErr
		}		
		vpcInfoList = append(vpcInfoList, &vpcInfo)
	}
	return vpcInfoList, nil
}

func (VPCHandler *NcpVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	cblogger.Info("NCP Cloud Driver: called DeleteVPC()!")

	if vpcIID.SystemId != "" {
		vpcIID.NameId = vpcIID.SystemId
    }

	//To check whether the VPC exists.
	_, vpcErr := VPCHandler.GetVPC(irs.IID{SystemId: vpcIID.NameId})
	if vpcErr != nil {
		newErr := fmt.Errorf("Failed to Get the VPC Info : [%v]", vpcErr)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	
	zoneId := VPCHandler.RegionInfo.Zone
	if zoneId == "" {
		newErr := fmt.Errorf("Failed to Get Zone info. from the connection info.")
		cblogger.Error(newErr.Error())
		return false, newErr
	} else {
		cblogger.Infof("ZoneId : %s", zoneId)
	}

	vpcPath := os.Getenv("CBSPIDER_ROOT") + vpcDir	
	vpcFilePath := vpcPath + zoneId + "/"
	jsonFileName := vpcFilePath + vpcIID.NameId + ".json"

	cblogger.Infof("VPC info file to Delete : [%s]", jsonFileName)

	// Check if the VPC Folder Exists, and Create it
	if err := CheckFolderAndCreate(vpcPath); err != nil {
		newErr := fmt.Errorf("Failed to Create the VPC File Dir : %s, [%v]" + vpcPath, err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	// Check if the VPC Folder Exists, and Create it
	if err := CheckFolderAndCreate(vpcFilePath); err != nil {
		newErr := fmt.Errorf("Failed to Create the VPC File Path : %s, [%v]" + vpcFilePath, err)
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

func (VPCHandler *NcpVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	cblogger.Info("NCP cloud driver: called AddSubnet()!!")
	return irs.VPCInfo{}, errors.New("Does not support AddSubnet() yet!!")
}

func (VPCHandler *NcpVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Info("NCP cloud driver: called GetImage()!!")
	return true, errors.New("Does not support RemoveSubnet() yet!!")
}

func (VPCHandler *NcpVPCHandler) CreateSubnet(subnetReqInfo irs.SubnetInfo) (irs.SubnetInfo, error) {
	cblogger.Info("NCP cloud driver: called CreateSubnet()!!")

	newSubnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId: 	subnetReqInfo.IId.NameId,
			// Note!! : subnetReqInfo.IId.NameId -> SystemId
			SystemId: 	subnetReqInfo.IId.NameId,
		},
		Zone: 			subnetReqInfo.Zone,
		IPv4_CIDR: 		subnetReqInfo.IPv4_CIDR,
		KeyValueList: 	[]irs.KeyValue{
			{Key: "NCP-Subnet-info.", Value: "This Subnet info. is temporary."},
		},
	}
	return newSubnetInfo, nil
}

func (VPCHandler *NcpVPCHandler) MappingVPCInfo(vpcJSON VPC) (irs.VPCInfo, error) {
	cblogger.Info("NCP cloud driver: called MappingVPCInfo()!!")

	var subnetInfoList []irs.SubnetInfo
	var subnetInfo irs.SubnetInfo
	var subnetKeyValue irs.KeyValue
	var subnetKeyValueList []irs.KeyValue
	var vpcKeyValue irs.KeyValue
	var vpcKeyValueList []irs.KeyValue

	for i := 0; i < len(vpcJSON.Subnet_List); i++ {
		subnetInfo.IId.NameId = 	vpcJSON.Subnet_List[i].IID.NameID
		subnetInfo.IId.SystemId = 	vpcJSON.Subnet_List[i].IID.SystemID
		subnetInfo.Zone = 			vpcJSON.Subnet_List[i].Zone
		subnetInfo.IPv4_CIDR = 		vpcJSON.Subnet_List[i].Cidr

		for j := 0; j < len(vpcJSON.Subnet_List[i].KeyValue_List); j++ {
			subnetKeyValue.Key = 	vpcJSON.Subnet_List[i].KeyValue_List[j].Key
			subnetKeyValue.Value = 	vpcJSON.Subnet_List[i].KeyValue_List[j].Value
			subnetKeyValueList = append(subnetKeyValueList, subnetKeyValue)
		}
		subnetInfo.KeyValueList = subnetKeyValueList
		subnetInfoList = append(subnetInfoList, subnetInfo)
    }

	for k := 0; k < len(vpcJSON.KeyValue_List); k++ {
		vpcKeyValue.Key = vpcJSON.KeyValue_List[k].Key
		vpcKeyValue.Value  = vpcJSON.KeyValue_List[k].Value
		vpcKeyValueList = append(vpcKeyValueList, vpcKeyValue)
    }

	vpcInfo := irs.VPCInfo{
		// Since the VPC information is managed as a file, the systemID is the same as the nameID.
		IId:        	irs.IID{NameId: vpcJSON.IID.NameID, SystemId: vpcJSON.IID.NameID},
		IPv4_CIDR: 		vpcJSON.Cidr,
		SubnetInfoList: subnetInfoList,
		KeyValueList:   vpcKeyValueList,
	}
	return vpcInfo, nil
}
