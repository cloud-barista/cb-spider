package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
	"github.com/Azure/go-autorest/autorest/to"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzureTagHandler struct {
	Region idrv.RegionInfo
	Ctx    context.Context
	Client *resources.TagsClient
}

// AddTag adds a tag to the specified resource
func (tagHandler *AzureTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, string(resType), "AddTag()")

	// Fetch existing tags
	tagsResource, err := tagHandler.Client.GetAtScope(tagHandler.Ctx, resIID.SystemId)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to get existing tags for resource ID %s: %s", resIID.SystemId, err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyValue{}, createErr
	}

	// Add new tag
	if tagsResource.Properties.Tags == nil {
		tagsResource.Properties.Tags = make(map[string]*string)
	}
	tagsResource.Properties.Tags[tag.Key] = to.StringPtr(tag.Value)

	// Update tags
	start := call.Start()
	_, err = tagHandler.Client.CreateOrUpdateAtScope(tagHandler.Ctx, resIID.SystemId, tagsResource)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to add tag to resource ID %s: %s", resIID.SystemId, err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyValue{}, createErr
	}
	LoggingInfo(hiscallInfo, start)

	return tag, nil
}

// ListTag lists all tags of the specified resource
func (tagHandler *AzureTagHandler) ListTag(resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, string(resType), "ListTag()")

	start := call.Start()
	// tagsResource, err := tagHandler.Client.GetAtScope(tagHandler.Ctx, resIID.SystemId)
	tagsResource, err := tagHandler.Client.GetAtScope(tagHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to list tags for resource ID %s: %s", resIID.SystemId, err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	fmt.Println("test : {} ", tagsResource)
	LoggingInfo(hiscallInfo, start)

	var tagList []irs.KeyValue
	for key, value := range tagsResource.Properties.Tags {
		tagList = append(tagList, irs.KeyValue{Key: key, Value: *value})
	}

	return tagList, nil
}

// GetTag gets a specific tag of the specified resource
func (tagHandler *AzureTagHandler) GetTag(resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, string(resType), "GetTag()")

	start := call.Start()
	tagsResource, err := tagHandler.Client.GetAtScope(tagHandler.Ctx, resIID.SystemId)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get tag for resource ID %s: %s", resIID.SystemId, err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.KeyValue{}, getErr
	}
	LoggingInfo(hiscallInfo, start)

	if value, exists := tagsResource.Properties.Tags[key]; exists {
		return irs.KeyValue{Key: key, Value: *value}, nil
	}

	return irs.KeyValue{}, errors.New("tag not found")
}

// RemoveTag removes a specific tag from the specified resource
func (tagHandler *AzureTagHandler) RemoveTag(resType irs.RSType, resIID irs.IID, key string) (bool, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, string(resType), "RemoveTag()")

	// Fetch existing tags
	tagsResource, err := tagHandler.Client.GetAtScope(tagHandler.Ctx, resIID.SystemId)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to get existing tags for resource ID %s: %s", resIID.SystemId, err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}

	// Remove the tag
	if _, exists := tagsResource.Properties.Tags[key]; !exists {
		return false, errors.New("tag not found")
	}
	delete(tagsResource.Properties.Tags, key)

	// Update tags
	start := call.Start()
	_, err = tagHandler.Client.CreateOrUpdateAtScope(tagHandler.Ctx, resIID.SystemId, tagsResource)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to remove tag from resource ID %s: %s", resIID.SystemId, err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

//FindTag finds tags by key or value
func (tagHandler *AzureTagHandler) FindTag(resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, string(resType), "FindTag()")

	start := call.Start()
	tagList, err := tagHandler.Client.List(tagHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to find tags: %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	LoggingInfo(hiscallInfo, start)

	var foundTags []*irs.TagInfo
	for _, tag := range tagList.Values() {
		fmt.Println(tag)
		// for key, value := range tag.Properties.Tags {
			
			// if keyword == "" || keyword == "*" || key == keyword || *value == keyword {
			// 	foundTags = append(foundTags, &irs.TagInfo{
			// 		ResType: resType,
			// 		ResIId:  irs.IID{NameId: *tag.Name, SystemId: *tag.ID},
			// 		TagList: []irs.KeyValue{{Key: key, Value: *value}},
			// 	})
			// }
		// }
	}

	return foundTags, nil
}