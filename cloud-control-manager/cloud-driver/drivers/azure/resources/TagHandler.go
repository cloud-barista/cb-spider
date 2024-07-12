package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
	"github.com/Azure/go-autorest/autorest/to"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzureTagHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region idrv.RegionInfo
	Ctx    context.Context
	Client *resources.TagsClient
}
type Resource struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Tags map[string]string `json:"tags"`
}

type Response struct {
	Value []Resource `json:"value"`
}

// find all resource by subscription ID
func GetResourceInfo(credentailInfo idrv.CredentialInfo, url string) (*Response, error){
	token, err := getToken(credentailInfo.TenantId, credentailInfo.ClientId, credentailInfo.ClientSecret)
	if err != nil {
		return nil, err
	}
	URL := url
	
	
	var bearer = "Bearer " + token

	ctx := context.Background()
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", bearer)
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	
	var response Response
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}
// find SystemId by NameId
func FindIdByName(credentailInfo idrv.CredentialInfo, resIID irs.IID) (string, error) {
	if resIID.SystemId != "" {
		return resIID.SystemId, nil
	}

	url := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resources?api-version=2021-04-01", credentailInfo.SubscriptionId)
	response, err := GetResourceInfo(credentailInfo, url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch resource info: %v", err)
	}

	for _, resource := range response.Value {
		if strings.Contains(resource.Name, resIID.NameId) {
			return resource.Id, nil
		}
	}

	return "", fmt.Errorf("resource with name containing '%s' not found", resIID.NameId)
}

func findRSType(azureType string) (irs.RSType, error) {
	switch azureType {
	case "Microsoft.Compute/virtualMachines":
		return irs.VM, nil
	case "Microsoft.Compute/disks":
		return irs.DISK, nil
	case "Microsoft.Network/virtualNetworks":
		return irs.VPC, nil
	case "Microsoft.Compute/snapshots":
		return irs.MYIMAGE, nil
	case "Microsoft.Network/loadBalancers":
		return irs.NLB, nil
	case "Microsoft.Network/networkSecurityGroups":
		return irs.SG, nil
	case "Microsoft.Compute/sshPublicKeys":
		return irs.KEY, nil
	case "Microsoft.ContainerService/managedClusters":
		return irs.CLUSTER, nil
	default:
		return "", errors.New(azureType + " is not supported Resource!!")
	}
}
// AddTag adds a tag to the specified resource
func (tagHandler *AzureTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	resourceID, err := FindIdByName(tagHandler.CredentialInfo, resIID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	resIID.SystemId = resourceID
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

func (tagHandler *AzureTagHandler) ListTag(resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
	resourceID, err := FindIdByName(tagHandler.CredentialInfo, resIID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	resIID.SystemId = resourceID
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, string(resType), "ListTag()")

	start := call.Start()
	tagsResource, err := tagHandler.Client.GetAtScope(tagHandler.Ctx, resIID.SystemId)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to list tags for resource ID %s: %s", resIID.SystemId, err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	LoggingInfo(hiscallInfo, start)

	var tagList []irs.KeyValue
	for key, value := range tagsResource.Properties.Tags {
		tagList = append(tagList, irs.KeyValue{Key: key, Value: *value})
	}

	return tagList, nil
}

// GetTag gets a specific tag of the specified resource
func (tagHandler *AzureTagHandler) GetTag(resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	resourceID, err := FindIdByName(tagHandler.CredentialInfo, resIID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	resIID.SystemId = resourceID
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
	resourceID, err := FindIdByName(tagHandler.CredentialInfo, resIID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	resIID.SystemId = resourceID
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
	urlByProvider:="https://management.azure.com/subscriptions/%s/providers/%s?api-version=2021-04-01"
	var url string
	switch resType {
	case irs.ALL:
		url = fmt.Sprintf("https://management.azure.com/subscriptions/%s/resources?api-version=2021-04-01",tagHandler.CredentialInfo.SubscriptionId)
	case irs.VM:
		url = fmt.Sprintf(urlByProvider,tagHandler.CredentialInfo.SubscriptionId,"Microsoft.Compute/virtualMachines")
	case irs.DISK:
		url = fmt.Sprintf(urlByProvider,tagHandler.CredentialInfo.SubscriptionId,"Microsoft.Compute/disks")
	case irs.VPC:
		url = fmt.Sprintf(urlByProvider,tagHandler.CredentialInfo.SubscriptionId,"Microsoft.Network/virtualNetworks")
	case irs.MYIMAGE:
		url = fmt.Sprintf(urlByProvider,tagHandler.CredentialInfo.SubscriptionId,"Microsoft.Compute/snapshots")
	case irs.NLB:
		url = fmt.Sprintf(urlByProvider,tagHandler.CredentialInfo.SubscriptionId,"Microsoft.Network/loadBalancers")
	case irs.SG:
		url = fmt.Sprintf(urlByProvider,tagHandler.CredentialInfo.SubscriptionId,"Microsoft.Network/networkSecurityGroups")
	case irs.KEY:
		url = fmt.Sprintf(urlByProvider,tagHandler.CredentialInfo.SubscriptionId,"Microsoft.Compute/sshPublicKeys")
	case irs.CLUSTER:
		url = fmt.Sprintf(urlByProvider,tagHandler.CredentialInfo.SubscriptionId,"Microsoft.ContainerService/managedClusters")
	default:
		fmt.Println(errors.New(string(resType) + " is not supported Resource!!"))
	}
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, string(resType), "FindTag()")
	start := call.Start()
	response, _:= GetResourceInfo(tagHandler.CredentialInfo, url)
	LoggingInfo(hiscallInfo, start)

	var foundTags []*irs.TagInfo
	for _, resource := range response.Value {
				var tagList []irs.KeyValue
		for key, value := range resource.Tags {
			if strings.Contains(key, keyword) || strings.Contains(value, keyword) {
				tagList = append(tagList, irs.KeyValue{Key: key, Value: value})
			}
		}

		if len(tagList) > 0 {
			resType, err := findRSType(resource.Type)
			if err != nil || resType == "" {
				continue // resType이 유효하지 않거나 지원되지 않는 경우 pass
			}
			tagInfo := &irs.TagInfo{
				ResType: resType,
				ResIId:  irs.IID{NameId: resource.Name, SystemId: resource.Id},
				TagList: tagList,
			}
			foundTags = append(foundTags, tagInfo)
		}
	}

	return foundTags, nil
}