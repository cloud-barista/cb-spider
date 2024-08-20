// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud Tag Handler
//
// by ETRI, 2024.08.

package resources

import (
	"fmt"
	"strings"
	"time"

	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type KtCloudTagHandler struct {
	RegionInfo     	idrv.RegionInfo
	Client        	*ktsdk.KtCloudClient
}

// KT Cloud Resource Types for Tagging => 'userVm' : VM, 'Template' : MyImage, 'Volume' : Disk
const (
	VM 				string = "userVm"
	MyImage 		string = "Template"
	DISK 			string = "Volume"
)

func (tagHandler *KtCloudTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	cblogger.Info("KT Cloud driver: called AddTag()!!")

	if resType == "" {
		newErr := fmt.Errorf("Invalid Resource Type!!")
		cblogger.Error(newErr.Error())
		return irs.KeyValue{}, newErr
	}

	if (resType != irs.VM) && (resType != irs.MYIMAGE) && (resType != irs.DISK) {
		newErr := fmt.Errorf("Only 'VM', 'MyImage' and 'Disk' type are supported as a resource type to Add a Tag Info on KT Classic!!")
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
	_, err := tagHandler.createTagList(resType, &resIID.SystemId, tagList)
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

func (tagHandler *KtCloudTagHandler) ListTag(resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
	cblogger.Info("KT Cloud driver: called ListTag()!!")

	if resType == "" {
		newErr := fmt.Errorf("Invalid Resource Type!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if strings.EqualFold(resIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Resource SystemId!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	tagList, err := tagHandler.getTagListWithResId(resType, &resIID.SystemId)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Get the Tag List with the Resource ID : [%v]", err)
		cblogger.Debug(newErr.Error())
		return nil, newErr
	}

	var kvList []irs.KeyValue
	if len(tagList) > 0 {
		for _, curTag := range tagList {
			kv := irs.KeyValue {
				Key : 	curTag.Key,
				Value:  curTag.Value,
			}
			kvList = append(kvList, kv)
		}
	}
	return kvList, nil
}

func (tagHandler *KtCloudTagHandler) GetTag(resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	cblogger.Info("KT Cloud driver: called GetTag()!")	

	if resType == "" {
		newErr := fmt.Errorf("Invalid Resource Type!!")
		cblogger.Error(newErr.Error())
		return irs.KeyValue{}, newErr
	}

	if strings.EqualFold(resIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Resource SystemId!!")
		cblogger.Error(newErr.Error())
		return irs.KeyValue{}, newErr
	}

	tagValue, err := tagHandler.getTagValue(resType, &resIID.SystemId, &key)
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

func (tagHandler *KtCloudTagHandler) RemoveTag(resType irs.RSType, resIID irs.IID, key string) (bool, error) {
	cblogger.Info("KT Cloud driver: called RemoveTag()!")

	if resType == "" {
		newErr := fmt.Errorf("Invalid Resource Type!!")
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

	value, err := tagHandler.getTagValue(resType, &resIID.SystemId, &key)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Get the Tag Value with the Key [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	tagKV := ktsdk.TagArg{
		Key: 	key, 
		Value: 	value,
	}
	_, delErr := tagHandler.deleteTag(resType, &resIID.SystemId, tagKV)
	if delErr != nil {		
		newErr := fmt.Errorf("Failed to Remove the Tag with the Key : [%v]", delErr)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	return true, nil
}

func (tagHandler *KtCloudTagHandler) FindTag(resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	cblogger.Info("KT Cloud driver: called FindTag()!")

	if resType == "" {
		newErr := fmt.Errorf("Invalid Resource Type!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if strings.EqualFold(keyword, "") {
		newErr := fmt.Errorf("Invalid Keyword!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	tagList, err := tagHandler.getTagList(resType)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Tag List : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	var tagInfoList []*irs.TagInfo
	for _, curTag := range tagList {
		if strings.Contains(curTag.Key, keyword) || strings.Contains(curTag.Value, keyword) {
			var tagKVList []irs.KeyValue 
			tagKV := irs.KeyValue {
				Key: 	curTag.Key,
				Value: 	curTag.Value,
			}
			tagKVList = append(tagKVList, tagKV)

			tagInfo := &irs.TagInfo {
				ResType : resType,
				ResIId  : irs.IID {
							NameId : "",
							SystemId: curTag.ResourceId,
						},
				TagList : tagKVList,
				// KeyValueList: 		,	// reserved for optinal usage
			}
			tagInfoList = append(tagInfoList, tagInfo)
		}
	}
	return tagInfoList, nil
}

func (tagHandler *KtCloudTagHandler) createTagList(resType irs.RSType, resID *string, tagList []irs.KeyValue) (bool, error) {
	cblogger.Info("KT Cloud driver: called createTagList()!")

	var rsType string
    switch resType {
    case irs.VM:
        rsType = VM
	case irs.MYIMAGE:
        rsType = MyImage
	case irs.DISK:
        rsType = DISK
    default:
        newErr := fmt.Errorf("Invalid Resource Type. [%v] type is Not Supported on KT Cloud for Tagging!!", resType)
		cblogger.Debug(newErr.Error())
		return false, newErr
    }

	var tagKVs []ktsdk.TagArg
	for _, curTag := range tagList {
		tagKV := ktsdk.TagArg {
			Key: 	curTag.Key,
			Value: 	curTag.Value,
		}
		tagKVs = append(tagKVs, tagKV)
	}

	tagReq := ktsdk.CreateTagsReqInfo {
		ResourceType: 	rsType,
		ResourceIds: 	[]string{*resID, },
		Tags:			tagKVs,
	}
	tagResult, err := tagHandler.Client.CreateTags(&tagReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create the Tag List on the Resource : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	cblogger.Info("### Waiting for Tags to be Created(300sec)!!\n")
	waitJobErr := tagHandler.Client.WaitForAsyncJob(tagResult.Createtagsresponse.JobId, 300000000000)
	if waitJobErr != nil {
		cblogger.Errorf("Failed to Wait the Job : [%v]", waitJobErr)
		return false, waitJobErr
	}

	_, jobErr := tagHandler.Client.QueryAsyncJobResult(tagResult.Createtagsresponse.JobId)
	if err != nil {
		cblogger.Errorf("Failed to Find the Job: [%v]", jobErr)
		return false, jobErr
	}

	return true, nil
}

func (tagHandler *KtCloudTagHandler) getTagList(resType irs.RSType) ([]ktsdk.Tag, error) {
	cblogger.Info("KT Cloud driver: called getTagList()!")

	var rsType string
    switch resType {
    case irs.VM:
        rsType = VM
	case irs.MYIMAGE:
        rsType = MyImage
	case irs.DISK:
        rsType = DISK
    default:
        newErr := fmt.Errorf("Invalid Resource Type. [%v] type is Not Supported on KT Cloud for Tagging!!", resType)
		cblogger.Debug(newErr.Error())
		return nil, newErr
    }

	tagReq := ktsdk.ListTagsReqInfo {
		ResourceType: rsType,
	}
	result, err := tagHandler.Client.ListTags(&tagReq)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Get Tag List from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	if len(result.Listtagsresponse.Tag) < 1 {
		newErr := fmt.Errorf("Failed to Find Any Tag with the Resource Type!!")
		cblogger.Debug(newErr.Error())
		return nil, nil // Caution!!
	}
	return result.Listtagsresponse.Tag, nil
}

func (tagHandler *KtCloudTagHandler) getTagListWithResId(resType irs.RSType, resourceID *string) ([]ktsdk.Tag, error) {
	cblogger.Info("KT Cloud driver: called getTagListWithResId()!")

	var rsType string
    switch resType {
    case irs.VM:
        rsType = VM
	case irs.MYIMAGE:
        rsType = MyImage
	case irs.DISK:
        rsType = DISK
    default:
        newErr := fmt.Errorf("Invalid Resource Type. [%v] type is Not Supported on KT Cloud for Tagging!!", resType)
		cblogger.Debug(newErr.Error())
		return nil, newErr
    }

	tagReq := ktsdk.ListTagsReqInfo {
		ResourceType: 	rsType,
		ResourceIds: 	*resourceID,
	}
	result, err := tagHandler.Client.ListTags(&tagReq)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Get Tag List from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	if len(result.Listtagsresponse.Tag) < 1 {
		newErr := fmt.Errorf("Failed to Find Any Tag with the Resource ID!!")
		cblogger.Debug(newErr.Error())
		return nil, nil // Caution!!
	}
	return result.Listtagsresponse.Tag, nil
}

func (tagHandler *KtCloudTagHandler) getTagValue(resType irs.RSType, resourceID *string, key *string) (string, error) {
	cblogger.Info("KT Cloud driver: called getTagValue()!")

	var rsType string
    switch resType {
    case irs.VM:
        rsType = VM
	case irs.MYIMAGE:
        rsType = MyImage
	case irs.DISK:
        rsType = DISK
    default:
        newErr := fmt.Errorf("Invalid Resource Type. [%v] type is Not Supported on KT Cloud for Tagging!!", resType)
		cblogger.Debug(newErr.Error())
		return "", newErr
    }

	tagReq := ktsdk.ListTagsReqInfo {
		ResourceType: 	rsType,
		ResourceIds: 	*resourceID,
		Key: 			*key,
	}
	result, err := tagHandler.Client.ListTags(&tagReq)
	if err != nil {		
		newErr := fmt.Errorf("Failed to Get Tag List from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	if len(result.Listtagsresponse.Tag) < 1 {
		newErr := fmt.Errorf("Failed to Find Any Tag with the Resource ID and the Key!!")
		cblogger.Debug(newErr.Error())
		return "", newErr
	}
	return result.Listtagsresponse.Tag[0].Value, nil
}

func (tagHandler *KtCloudTagHandler) deleteTag(resType irs.RSType, resID *string, keyValue ktsdk.TagArg) (bool, error) {
	cblogger.Info("KT Cloud driver: called deleteTag()!")

	var rsType string
    switch resType {
    case irs.VM:
        rsType = VM
	case irs.MYIMAGE:
        rsType = MyImage
	case irs.DISK:
        rsType = DISK
    default:
        newErr := fmt.Errorf("Invalid Resource Type. [%v] type is Not Supported on KT Cloud for Tagging!!", resType)
		cblogger.Debug(newErr.Error())
		return false, newErr
    }

	var tagKVs []ktsdk.TagArg
	tagKV := ktsdk.TagArg {
		Key: 	keyValue.Key,
		Value: 	keyValue.Value,
	}
	tagKVs = append(tagKVs, tagKV)

	tagReq := ktsdk.DeleteTagsReqInfo {
		ResourceType: 	rsType,
		ResourceIds: 	[]string{*resID, },
		Tags: 			tagKVs,
	}
	_, err := tagHandler.Client.DeleteTags(&tagReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Tag with the Key on the Resource : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	return true, nil
}
