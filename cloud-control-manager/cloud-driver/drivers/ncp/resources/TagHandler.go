package resources

import (
	"fmt"
	"strings"

	ncloud "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	server "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/server"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpTagHandler struct {
	CredentialInfo 	idrv.CredentialInfo
	RegionInfo     	idrv.RegionInfo
	VMClient        *server.APIClient
}

func (tagHandler *NcpTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	cblogger.Info("NCP Classic Cloud driver: called AddTag()!!")

	if resType != irs.VM {
		newErr := fmt.Errorf("Only 'VM' type are supported as a resource type to add a Tag on NCP Classic!!")
		cblogger.Error(newErr.Error())
		return irs.KeyValue{}, newErr
	}

	if strings.EqualFold(resIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Resource SystemId")
		cblogger.Error(newErr.Error())
		return irs.KeyValue{}, newErr
	}

	_, err := tagHandler.createVMTag(&resIID.SystemId, tag)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Add New Tag : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.KeyValue{}, newErr
	}

	tagKV, err := tagHandler.GetTag(resType, resIID, tag.Key)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Get the Tag Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.KeyValue{}, newErr
	}
	return tagKV, nil
}

func (tagHandler *NcpTagHandler) ListTag(resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
	cblogger.Info("NCP Classic Cloud driver: called ListTag()!!")

	return nil, fmt.Errorf("NCP Cloud Driver does not support ListTag yet.")
}

func (tagHandler *NcpTagHandler) GetTag(resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetTag()!")	

	if resType != irs.VM {
		newErr := fmt.Errorf("Only 'VM' type are supported as a resource type to add a Tag on NCP Classic!!")
		cblogger.Error(newErr.Error())
		return irs.KeyValue{}, newErr
	}

	if strings.EqualFold(resIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Resource SystemId")
		cblogger.Error(newErr.Error())
		return irs.KeyValue{}, newErr
	}

	tagValue, err := tagHandler.getVMTagValue(&resIID.SystemId, key)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Get the Tag Value with the Key : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.KeyValue{}, newErr
	}

	tagKV := irs.KeyValue{
		Key: 	key, 
		Value: 	tagValue,
	}
	return tagKV, nil
}

func (tagHandler *NcpTagHandler) RemoveTag(resType irs.RSType, resIID irs.IID, key string) (bool, error) {
	cblogger.Info("NCP Classic Cloud driver: called RemoveTag()!")

	if resType != irs.VM {
		newErr := fmt.Errorf("Only 'VM' type are supported as a resource type to add a Tag on NCP Classic!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	_, err := tagHandler.GetTag(resType, resIID, key)
	if err != nil {		
		newErr := fmt.Errorf("The Tag with the Key does not exist [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	_, delErr := tagHandler.deleteVMTag(&resIID.SystemId, key)
	if delErr != nil {		
		newErr := fmt.Errorf("Failed to Remove the Tag with the Key : [%v]", delErr)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	return true, nil
}

func (tagHandler *NcpTagHandler) FindTag(resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	cblogger.Info("NCP Classic Cloud driver: called FindTag()!")

	if resType != irs.VM {
		newErr := fmt.Errorf("Only 'VM' type are supported as a resource type to add a Tag on NCP Classic!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	return nil, fmt.Errorf("NCP Cloud Driver does not support FindTag yet.")
}

func (tagHandler *NcpTagHandler) createVMTag(vmID *string, tag irs.KeyValue) (bool, error) {
	cblogger.Info("NCP Classic Cloud driver: called createVMTag()!")

	var instanceNos  []*string
	var instanceTags []*server.InstanceTagParameter

	instanceNos = append(instanceNos, vmID)

	instanceTags = []*server.InstanceTagParameter {
		{
			TagKey: 	ncloud.String(tag.Key), 
			TagValue: 	ncloud.String(tag.Value),
		},
	}
	tagReq := server.CreateInstanceTagsRequest{
		InstanceNoList:     instanceNos,
		InstanceTagList: 	instanceTags,
	}
	tagResult, err := tagHandler.VMClient.V2Api.CreateInstanceTags(&tagReq)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Create the Tag on the VM : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	} else {
		cblogger.Infof("Tag Creation Result : [%s]", *tagResult.ReturnMessage)
	}

	return true, nil
}

func (tagHandler *NcpTagHandler) getVMTagValue(vmID *string, key string) (string, error) {
	cblogger.Info("NCP Classic Cloud driver: called getVMTagValue()!")

	var instanceNos []*string
	var tagValue string
	instanceNos = append(instanceNos, vmID)

	tagReq := server.GetInstanceTagListRequest{InstanceNoList: instanceNos}
	tagListResult, err := tagHandler.VMClient.V2Api.GetInstanceTagList(&tagReq)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Get VM Tag List from the VM : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	if len(tagListResult.InstanceTagList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any Tag with the VM SystemID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	} else {
		cblogger.Infof("Tag Info Listing Result : [%s]", *tagListResult.ReturnMessage)
	}

	for _, curTag := range tagListResult.InstanceTagList {
		if strings.EqualFold(ncloud.StringValue(curTag.TagKey), key) {			
			tagValue = ncloud.StringValue(curTag.TagValue)
		}
	}
	if len(tagValue) < 1 {
		newErr := fmt.Errorf("Failed to Get the Tag Value with the Key!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	return tagValue, nil
}

func (tagHandler *NcpTagHandler) deleteVMTag(vmID *string, key string) (bool, error) {
	cblogger.Info("NCP Classic Cloud driver: called deleteVMTag()!")

	var instanceNos  []*string
	var instanceTags []*server.InstanceTagParameter

	instanceNos = append(instanceNos, vmID)

	instanceTags = []*server.InstanceTagParameter {
		{
			TagKey: 	ncloud.String(key),
		},
	}
	tagReq := server.DeleteInstanceTagsRequest {
		InstanceNoList:     instanceNos,
		InstanceTagList: 	instanceTags,
	}
	tagResult, err := tagHandler.VMClient.V2Api.DeleteInstanceTags(&tagReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Tag with the Key on the VM : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	} else {
		cblogger.Infof("Tag Deletion Result : [%s]", *tagResult.ReturnMessage)
	}

	return true, nil
}
