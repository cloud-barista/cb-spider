// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP Security Group Handler
//
// by ETRI, 2024.07.

package resources

import (
	"fmt"
	"strings"
	"time"

	ncloud "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	server "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/server"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpTagHandler struct {
	RegionInfo     	idrv.RegionInfo
	VMClient        *server.APIClient
}

func (tagHandler *NcpTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	cblogger.Info("NCP Classic Cloud driver: called AddTag()!!")

	if resType != irs.VM {
		newErr := fmt.Errorf("Only 'VM' type are supported as a resource type to Add a Tag Info on NCP Classic!!")
		cblogger.Error(newErr.Error())
		return irs.KeyValue{}, newErr
	}

	if strings.EqualFold(resIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Resource SystemId!!")
		cblogger.Error(newErr.Error())
		return irs.KeyValue{}, newErr
	}

	tagList := []irs.KeyValue {
		{ 
			Key: 	tag.Key,
			Value: 	tag.Value,
		},
	}
	_, err := tagHandler.createVMTagList(&resIID.SystemId, tagList)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Add New Tag : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.KeyValue{}, newErr
	}

	time.Sleep(time.Second * 1)
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

	if resType != irs.VM {
		newErr := fmt.Errorf("Only 'VM' type are supported as a resource type to Get Tag Info List on NCP Classic!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if strings.EqualFold(resIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Resource SystemId!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	tagList, err := tagHandler.getVMTagListWithVMId(&resIID.SystemId)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Get the Tag List with the VM SystemID : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	var kvList []irs.KeyValue
	for _, curTag := range tagList {
		kv := irs.KeyValue {
			Key : 	ncloud.StringValue(curTag.TagKey),
			Value:  ncloud.StringValue(curTag.TagValue),
		}
		kvList = append(kvList, kv)
	}
	return kvList, nil
}

func (tagHandler *NcpTagHandler) GetTag(resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetTag()!")	

	if resType != irs.VM {
		newErr := fmt.Errorf("Only 'VM' type are supported as a resource type to Get a Tag Info on NCP Classic!!")
		cblogger.Error(newErr.Error())
		return irs.KeyValue{}, newErr
	}

	if strings.EqualFold(resIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Resource SystemId!!")
		cblogger.Error(newErr.Error())
		return irs.KeyValue{}, newErr
	}

	tagValue, err := tagHandler.getVMTagValue(&resIID.SystemId, &key)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Get the Tag Value with the Key : [%s], [%v]", key, err)
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
		newErr := fmt.Errorf("Only 'VM' type are supported as a resource type to Remove a Tag Info on NCP Classic!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	if strings.EqualFold(resIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Resource SystemId!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	if strings.EqualFold(key, "") {
		newErr := fmt.Errorf("Invalid 'key' value!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	_, err := tagHandler.getVMTagValue(&resIID.SystemId, &key)
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

	if (resType != irs.VM) && (resType != irs.ALL) {
		newErr := fmt.Errorf("Only 'VM' and 'ALL' type are supported as a resource type to Find a Tag Info on NCP Classic!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if strings.EqualFold(keyword, "") {
		newErr := fmt.Errorf("Invalid Keyword!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	tagList, err := tagHandler.getVMTagList()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Tag List : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	var tagInfoList []*irs.TagInfo
	for _, curTag := range tagList {
		if strings.Contains(ncloud.StringValue(curTag.TagKey), keyword) || strings.Contains(ncloud.StringValue(curTag.TagValue), keyword) {
			var tagKVList []irs.KeyValue 
			tagKV := irs.KeyValue {
				Key: 	ncloud.StringValue(curTag.TagKey),
				Value: 	ncloud.StringValue(curTag.TagValue),
			}
			tagKVList = append(tagKVList, tagKV)

			tagInfo := &irs.TagInfo {
				ResType : "VM",
				ResIId  : irs.IID {
							NameId : "",
							SystemId: *curTag.InstanceNo,
						},
				TagList : tagKVList,
				// KeyValueList: 		,	// reserved for optinal usage
			}
			tagInfoList = append(tagInfoList, tagInfo)
		}
	}
	return tagInfoList, nil
}

func (tagHandler *NcpTagHandler) createVMTagList(vmID *string, tagList []irs.KeyValue) (bool, error) {
	cblogger.Info("NCP Classic Cloud driver: called createVMTagList()!")

	var instanceTags []*server.InstanceTagParameter
	for _, curTag := range tagList {
		instanceTag := server.InstanceTagParameter {
			TagKey: 	ncloud.String(curTag.Key),
			TagValue: 	ncloud.String(curTag.Value),
		}
		instanceTags = append(instanceTags, &instanceTag)
	}

	tagReq := server.CreateInstanceTagsRequest{
		InstanceNoList:     []*string {vmID,},
		InstanceTagList: 	instanceTags,
	}
	tagResult, err := tagHandler.VMClient.V2Api.CreateInstanceTags(&tagReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create the Tag List on the VM : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	} else {
		cblogger.Infof("Tag List Creation Result : [%s]", *tagResult.ReturnMessage)
	}

	return true, nil
}

func (tagHandler *NcpTagHandler) getVMTagList() ([]*server.InstanceTag, error) {
	cblogger.Info("NCP Classic Cloud driver: called getVMTagList()!")

	tagReq := server.GetInstanceTagListRequest{}
	tagListResult, err := tagHandler.VMClient.V2Api.GetInstanceTagList(&tagReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VM Tag List from NCP : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	if len(tagListResult.InstanceTagList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any Tag!!")
		cblogger.Debug(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Infof("Tag Listing Result : [%s]", *tagListResult.ReturnMessage)
	}
	return tagListResult.InstanceTagList, nil
}

func (tagHandler *NcpTagHandler) getVMTagListWithVMId(vmID *string) ([]*server.InstanceTag, error) {
	cblogger.Info("NCP Classic Cloud driver: called getVMTagListWithVMId()!")

	tagReq := server.GetInstanceTagListRequest{
		InstanceNoList: []*string {vmID,},
	}
	tagListResult, err := tagHandler.VMClient.V2Api.GetInstanceTagList(&tagReq)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Get VM Tag List from NCP : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	if len(tagListResult.InstanceTagList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any Tag with the VM SystemID!!")
		cblogger.Debug(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Infof("Tag Listing Result : [%s]", *tagListResult.ReturnMessage)
	}
	return tagListResult.InstanceTagList, nil
}

func (tagHandler *NcpTagHandler) getVMTagValue(vmID *string, key *string) (string, error) {
	cblogger.Info("NCP Classic Cloud driver: called getVMTagValue()!")

	tagReq := server.GetInstanceTagListRequest{
		InstanceNoList: []*string {vmID,},
		TagKeyList: 	[]*string {key,},
	}
	tagListResult, err := tagHandler.VMClient.V2Api.GetInstanceTagList(&tagReq)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Get VM Tag List from the NCP : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	if len(tagListResult.InstanceTagList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any Tag with the VM SystemID and Key!!")
		cblogger.Debug(newErr.Error())
		return "", newErr
	} else {
		cblogger.Infof("Tag Listing Result : [%s]", *tagListResult.ReturnMessage)
	}
	return *tagListResult.InstanceTagList[0].TagValue, nil
}

func (tagHandler *NcpTagHandler) deleteVMTag(vmID *string, key string) (bool, error) {
	cblogger.Info("NCP Classic Cloud driver: called deleteVMTag()!")

	instanceTags := []*server.InstanceTagParameter {
		{
			TagKey: 	ncloud.String(key),
		},
	}
	tagReq := server.DeleteInstanceTagsRequest {
		InstanceNoList:     []*string {vmID,},
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
