package resources

import (
	"encoding/json"
	"errors"

	cs "github.com/alibabacloud-go/cs-20151215/v4/client" // cs  : container service
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs" // ecs : elastic compute service

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaTagHandler struct {
	Region   idrv.RegionInfo
	Client   *ecs.Client
	CsClient *cs.Client
}

type AliTagResponse struct {
	RequestId  string `json:"RequestId"`
	PageSize   int    `json:"PageSize"`
	PageNumber int    `json:"PageNumber"`
	TotalCount int    `json:"TotalCount"`
	AliTags    struct {
		AliTag []AliTag `json:"Tag"`
	} `json:"Tags"`
}

type AliTag struct {
	TagKey            string               `json:"TagKey"`
	TagValue          string               `json:"TagValue"`
	ResourceTypeCount AliResourceTypeCount `json:"ResourceTypeCount"`
}

type AliTagResource struct {
	RegionId     string `json:"RegionId" xml:"RegionId"`
	ResourceType string `json:"ResourceType" xml:"ResourceType"`
	ResourceId   string `json:"ResourceId" xml:"ResourceId"`
	TagKey       string `json:"TagKey" xml:"TagKey"`
	TagValue     string `json:"TagValue" xml:"TagValue"`
}
type AliTagResources struct {
	Resources []AliTagResource `json:"Resource" xml:"Resource"`
}
type AliTagResourcesResponse struct {
	RequestId       string          `json:"RequestId" xml:"RequestId"`
	PageSize        int             `json:"PageSize" xml:"PageSize"`
	PageNumber      int             `json:"PageNumber" xml:"PageNumber"`
	TotalCount      int             `json:"TotalCount" xml:"TotalCount"`
	AliTagResources AliTagResources `json:"Resources" xml:"Resources"`
}

type AliResourceTypeCount struct {
	Instance         int `json:"Instance"`
	Image            int `json:"Image"`
	Ddh              int `json:"Ddh"`
	SnapshotPolicy   int `json:"SnapshotPolicy"`
	Snapshot         int `json:"Snapshot"`
	ReservedInstance int `json:"ReservedInstance"`
	LaunchTemplate   int `json:"LaunchTemplate"`
	Eni              int `json:"Eni"`
	Disk             int `json:"Disk"`
	KeyPair          int `json:"KeyPair"`
	Volume           int `json:"Volume"`
}

type DescribeTagsResponse struct {
	RequestId  string                 `json:"RequestId" xml:"RequestId"`
	TotalCount int                    `json:"TotalCount" xml:"TotalCount"`
	PageSize   int                    `json:"PageSize" xml:"PageSize"`
	PageNumber int                    `json:"PageNumber" xml:"PageNumber"`
	Tags       ecs.TagsInDescribeTags `json:"Tags" xml:"Tags"`
}

/*
* ECS Instances 아래에 Tags and ResourceGroup이 있음.
AddTag(resType RSType, resIID IID, tag KeyValue) (KeyValue, error)
*/
func (tagHandler *AlibabaTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	//hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, resIID.NameId, "CreateTag()")

	cblogger.Info("Start AddTag : ", tag)
	//regionID := tagHandler.Region.Region

	// 생성된 Tag 정보 획득 후, Tag 정보 리턴
	//tagInfo := irs.TagInfo{}

	// 지원하는 resource Type인지 확인
	alibabaResourceType, err := GetAlibabaResourceType(resType)
	if err != nil {
		//return tagInfo, err
		return tag, err
	}
	cblogger.Debug(alibabaResourceType)

	alibabaApiType, err := GetAliTargetApi(resType)
	if err != nil {
		return tag, err
	}

	switch alibabaApiType {
	case "ecs":
		// queryParams := map[string]string{}
		// queryParams["RegionId"] = regionID
		// queryParams["ResourceType"] = alibabaResourceType
		// queryParams["ResourceId"] = resIID.SystemId
		// queryParams["Tag.1.Key"] = tag.Key
		// queryParams["Tag.1.Value"] = tag.Value

		// start := call.Start()
		// response, err := CallEcsRequest(resType, tagHandler.Client, tagHandler.Region, "AddTags", queryParams)
		// LoggingInfo(hiscallInfo, start)

		// if err != nil {
		// 	cblogger.Error(err.Error())
		// 	LoggingError(hiscallInfo, err)
		// }
		// cblogger.Debug(response.GetHttpContentString())

		// expectStatus := true // 예상되는 상태 : 있어야 하므로 true
		// result, err := waitForTagExist(tagHandler.Client, tagHandler.Region, resType, resIID, tag.Key, expectStatus)
		// if err != nil {
		// 	return tag, err
		// }
		// cblogger.Debug("Expect Status ", expectStatus, ", result Status ", result)
		// if !result {
		// 	return tag, errors.New("waitForTagExist Error ")
		// }

		response, err := AddEcsTags(tagHandler.Client, tagHandler.Region, resType, resIID, tag)
		if err != nil {
			return tag, err
		}
		cblogger.Debug("AddEcsTags response", response)

		// 성공했으면 해당 Tag return.
	case "cs":
		response, err := aliAddCsTag(tagHandler.CsClient, tagHandler.Region, resType, resIID, tag)
		if err != nil {
			return tag, err
		}
		cblogger.Debug("AddCsTags response", response)
	}

	// request Tag를 retrun하므로 해당 Tag 정보조회 필요없음
	// // 해당 resource의 tag를 가져온다.
	// tagResponse, err := DescribeDescribeTags(tagHandler.Client, tagHandler.Region, resType, resIID, tag.Key)
	// if err != nil {
	// 	return tag, err
	// }

	// // tag들 추출
	// resTags := ecs.DescribeTagsResponse{}
	// tagResponseStr := tagResponse.GetHttpContentString()
	// err = json.Unmarshal([]byte(tagResponseStr), &resTags)
	// if err != nil {
	// 	cblogger.Error(err.Error())
	// 	return tag, err
	// }

	// // extract Tag
	// for _, aliTag := range resTags.Tags.Tag {
	// 	cblogger.Debug("aliTag ", aliTag)
	// 	aTagInfo, err := ExtractTagsDescribeInfo(&aliTag)
	// 	if err != nil {
	// 		cblogger.Error(err.Error())
	// 		continue
	// 	}

	// 	aTagInfo.ResType = resType
	// 	aTagInfo.ResIId = resIID

	// 	//break // TagList와 같은 function을 호출하나 1개만 사용
	// 	return aTagInfo, err
	// }
	return tag, nil
}

/*
*
Tag 목록을 제공한다.
ListTag(resType RSType, resIID IID) ([]KeyValue, error)
*/
func (tagHandler *AlibabaTagHandler) ListTag(resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, "TAG", "ListTag()")

	var tagInfoList []irs.KeyValue

	alibabaApiType, err := GetAliTargetApi(resType)
	if err != nil {
		return tagInfoList, err
	}

	switch alibabaApiType {
	case "ecs":
		response, err := DescribeDescribeEcsTags(tagHandler.Client, tagHandler.Region, resType, resIID, "")

		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
		}
		cblogger.Debug(response)

		resTags := ecs.DescribeTagsResponse{}

		tagResponseStr := response.GetHttpContentString()
		err = json.Unmarshal([]byte(tagResponseStr), &resTags)
		if err != nil {
			cblogger.Error(err.Error())
			return tagInfoList, nil
		}

		cblogger.Debug("resTags ", resTags)

		for _, aliTag := range resTags.Tags.Tag {
			cblogger.Debug("aliTag ", aliTag)
			// aTagInfo, err := ExtractTagsDescribeInfo(&aliTag)
			// if err != nil {
			// 	cblogger.Error(err.Error())
			// 	continue
			// }

			// aTagInfo.ResType = resType
			// aTagInfo.ResIId = resIID

			// cblogger.Debug("tag ", aliTag)
			// cblogger.Debug("TagKey ", aliTag.TagKey)
			// cblogger.Debug("TagValue ", aliTag.TagValue)
			aTagInfo := irs.KeyValue{Key: aliTag.TagKey, Value: aliTag.TagValue}
			cblogger.Debug("tagInfo ", aTagInfo)
			tagInfoList = append(tagInfoList, aTagInfo)
		}
	case "cs":
		response, err := aliCsListTag(tagHandler.CsClient, tagHandler.Region, resType, resIID)
		if err != nil {
			cblogger.Error(err.Error())
			return tagInfoList, nil
		}
		cblogger.Debug("aliCsListTag ", response)

		resTagResources := AliTagResourcesResponse{}

		cblogger.Debug("resTagResources ", resTagResources)
		cblogger.Debug("resTagResources.AliTagResources ", resTagResources.AliTagResources)
		cblogger.Debug("resTagResources.AliTagResources.Resources ", resTagResources.AliTagResources.Resources)

		for _, aliTagResource := range response.TagResource {
			cblogger.Debug("aliTagResource ", aliTagResource)

			aTagInfo := irs.KeyValue{Key: *aliTagResource.TagKey, Value: *aliTagResource.TagValue}
			cblogger.Debug("tagInfo ", aTagInfo)
			tagInfoList = append(tagInfoList, aTagInfo)
		}

		// resTags := cs.ListTagResourcesResponseBodyTagResources{}
		// for _, aliTag := range response.TagResources {
		// // }
		// // tagResponseStr := response.TagResource
		// // err = json.Unmarshal([]byte(tagResponseStr), &resTags)
		// // if err != nil {
		// // 	cblogger.Error(err.Error())
		// // 	return tagInfoList, nil
		// }
		// // cblogger.Debug("resTags ", resTags)

		// for _, aliTag := range resTags.Tags.Tag {
		// 	cblogger.Debug("aliTag ", aliTag)
		// 	// aTagInfo, err := ExtractTagsDescribeInfo(&aliTag)
		// 	// if err != nil {
		// 	// 	cblogger.Error(err.Error())
		// 	// 	continue
		// 	// }

		// 	// aTagInfo.ResType = resType
		// 	// aTagInfo.ResIId = resIID

		// 	// cblogger.Debug("tag ", aliTag)
		// 	// cblogger.Debug("TagKey ", aliTag.TagKey)
		// 	// cblogger.Debug("TagValue ", aliTag.TagValue)
		// 	aTagInfo := irs.KeyValue{Key: aliTag.TagKey, Value: aliTag.TagValue}
		// 	cblogger.Debug("tagInfo ", aTagInfo)
		// 	tagInfoList = append(tagInfoList, aTagInfo)
		// }
	}

	return tagInfoList, nil
}

func (tagHandler *AlibabaTagHandler) GetTag(resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, resIID.NameId, "GetTag()")

	tagInfo := irs.KeyValue{}

	alibabaApiType, err := GetAliTargetApi(resType)
	if err != nil {
		return tagInfo, err
	}

	switch alibabaApiType {
	case "ecs":
		response, err := DescribeDescribeEcsTags(tagHandler.Client, tagHandler.Region, resType, resIID, key)

		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
		}
		cblogger.Debug(response.GetHttpContentString())
		//spew.Dump(response)

		resTags := ecs.DescribeTagsResponse{}
		tagResponseStr := response.GetHttpContentString()
		err = json.Unmarshal([]byte(tagResponseStr), &resTags)
		if err != nil {
			cblogger.Error(err.Error())
			return tagInfo, nil
		}

		// extract Tag
		for _, aliTag := range resTags.Tags.Tag {
			tagInfo = irs.KeyValue{Key: aliTag.TagKey, Value: aliTag.TagValue}
		}
	case "cs": // cs : container service
		response, err := aliCsTag(tagHandler.CsClient, tagHandler.Region, resType, resIID, key)
		//response, err := aliCsListTag(tagHandler.CsClient, tagHandler.Region, resType, resIID)
		if err != nil {
			cblogger.Error(err.Error())
			return tagInfo, nil
		}

		cblogger.Debug("aliCsTag ", response)

		// TODO : 단건으로 parsing
		resTagResources := AliTagResourcesResponse{}

		cblogger.Debug("resTagResources ", resTagResources)
		cblogger.Debug("resTagResources.AliTagResources ", resTagResources.AliTagResources)
		cblogger.Debug("resTagResources.AliTagResources.Resources ", resTagResources.AliTagResources.Resources)

		for _, aliTagResource := range response.TagResource {
			cblogger.Debug("aliTagResource ", aliTagResource)

			tagInfo = irs.KeyValue{Key: *aliTagResource.TagKey, Value: *aliTagResource.TagValue}
			cblogger.Debug("tagInfo ", tagInfo)

		}
	}

	// for _, aliTag := range resTags.Tags.Tag {
	// 	cblogger.Debug("aliTag ", aliTag)
	// 	aTagInfo, err := ExtractTagsDescribeInfo(&aliTag)
	// 	if err != nil {
	// 		cblogger.Error(err.Error())
	// 		continue
	// 	}

	// 	aTagInfo.ResType = resType
	// 	aTagInfo.ResIId = resIID

	// 	//break // TagList와 같은 function을 호출하나 1개만 사용
	// 	return aTagInfo, err
	// }

	return tagInfo, nil
}

// 해당 Resource의 Tag 삭제. 요청이 비동기로 되므로 조회를 통해 삭제 될 때까지 대기. 확인되면  return true.
func (tagHandler *AlibabaTagHandler) RemoveTag(resType irs.RSType, resIID irs.IID, key string) (bool, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, resIID.NameId, "RemoveTag()")

	regionID := tagHandler.Region.Region

	alibabaResourceType, err := GetAlibabaResourceType(resType)
	if err != nil {
		return false, err
	}

	alibabaApiType, err := GetAliTargetApi(resType)
	if err != nil {
		return false, err
	}

	switch alibabaApiType {
	case "ecs":
		queryParams := map[string]string{}
		queryParams["RegionId"] = regionID
		queryParams["ResourceType"] = alibabaResourceType
		queryParams["ResourceId"] = resIID.SystemId
		queryParams["Tag.1.Key"] = key

		start := call.Start()
		response, err := CallEcsRequest(resType, tagHandler.Client, tagHandler.Region, "RemoveTags", queryParams)
		LoggingInfo(hiscallInfo, start)

		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
		}
		cblogger.Debug(response.GetHttpContentString())

		cblogger.Infof("Successfully deleted %q Task\n", resIID.SystemId)

		expectStatus := false // 예상되는 상태 : 없어야 하므로 fasle
		result, err := WaitForEcsTagExist(tagHandler.Client, tagHandler.Region, resType, resIID, key, expectStatus)
		if err != nil {
			return false, err
		}
		cblogger.Debug("Expect Status ", expectStatus, ", result Status ", result)
		if !result {
			return false, errors.New("waitForTagExist Error ")
		}
	case "cs": // cs : container service

		response, err := aliRemoveCsTag(tagHandler.CsClient, tagHandler.Region, resType, resIID, key)
		if err != nil {
			return false, err
		}
		cblogger.Debug("AddCsTags response", response)

	}
	return true, nil
}

// Find tags by tag key or value
// resType: ALL | VPC, SUBNET, etc.,.  ecs기준 : Ddh, Disk, Eni, Image, Instance, KeyPair, LaunchTemplate, ReservedInstance, Securitygroup, Snapshot, SnapshotPolicy, Volume,
// keyword: The keyword to search for in the tag key or value.
// if you want to find all tags, set keyword to "" or "*".
// 해당 Resource Type에 tag가 있는 것들. ListTag는 resourceId가 있으나 당 function은 더 넒음
func (tagHandler *AlibabaTagHandler) FindTag(resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	var tagInfoList []*irs.TagInfo

	//start := call.Start()
	//regionID := tagHandler.Region.Region
	//regionID = "ap-northeast-1" // for the test

	// alibabaResourceType, err := GetAlibabaResourceType(resType)
	// if err != nil {
	// 	return tagInfoList, err
	// }

	// alibabaApiType, err := GetAliTargetApi(resType)
	// if err != nil {
	// 	return tagInfoList, err
	// }

	// switch alibabaApiType {
	// case "ecs":

	// 	queryParams := map[string]string{}
	// 	queryParams["RegionId"] = regionID
	// 	queryParams["ResourceType"] = alibabaResourceType //string(resType)
	// 	queryParams["Tag.1.Key"] = keyword

	// 	start := call.Start()
	// 	response, err := CallEcsRequest(resType, tagHandler.Client, tagHandler.Region, "DescribeResourceByTags", queryParams)
	// 	LoggingInfo(hiscallInfo, start)

	// 	if err != nil {
	// 		cblogger.Error(err.Error())
	// 		LoggingError(hiscallInfo, err)
	// 	}
	// 	cblogger.Debug(response.GetHttpContentString())

	// 	resTagResources := AliTagResourcesResponse{}

	// 	tagResponseStr := response.GetHttpContentString()
	// 	err = json.Unmarshal([]byte(tagResponseStr), &resTagResources)
	// 	if err != nil {
	// 		cblogger.Error(err.Error())
	// 		return tagInfoList, nil
	// 	}
	// 	cblogger.Debug("resTagResources ", resTagResources)
	// 	cblogger.Debug("resTagResources.AliTagResources ", resTagResources.AliTagResources)
	// 	cblogger.Debug("resTagResources.AliTagResources.Resources ", resTagResources.AliTagResources.Resources)

	// 	for _, aliTagResource := range resTagResources.AliTagResources.Resources {
	// 		cblogger.Debug("aliTagResource ", aliTagResource)
	// 		aTagInfo, err := ExtractTagResourceInfo(&aliTagResource)
	// 		if err != nil {
	// 			cblogger.Error(err.Error())
	// 			continue
	// 		}

	// 		aTagInfo.ResType = resType

	// 		cblogger.Debug("tagInfo ", aTagInfo)
	// 		tagInfoList = append(tagInfoList, &aTagInfo)
	// 	}
	// case "cs": // cs : container service
	// 	clusters, err := aliDescribeClustersV1(tagHandler.CsClient, regionID)
	// 	if err != nil {
	// 		cblogger.Error(err)
	// 		LoggingError(hiscallInfo, err)
	// 		return nil, err
	// 	}

	// 	//cblogger.Debug("clusters ", clusters)
	// 	// 모든 cluster를 돌면서 Tag 찾기
	// 	for _, cluster := range clusters {
	// 		cblogger.Debug("inCluster ")
	// 		for _, aliTag := range cluster.Tags {
	// 			//cblogger.Debug("aliTag ", aliTag)
	// 			//cblogger.Debug("keyword ", keyword)
	// 			//cblogger.Debug("aliTag.Key ", *(aliTag.Key))
	// 			if *(aliTag.Key) == keyword {
	// 				var tagInfo irs.TagInfo
	// 				tagInfo.ResIId = irs.IID{SystemId: *cluster.ClusterId}
	// 				tagInfo.ResType = resType

	// 				tagList := []irs.KeyValue{}
	// 				tagList = append(tagList, irs.KeyValue{Key: "TagKey", Value: *aliTag.Key})
	// 				tagList = append(tagList, irs.KeyValue{Key: "TagValue", Value: *aliTag.Value})
	// 				tagInfo.TagList = tagList
	// 				//cblogger.Debug("append Tag ", &tagInfo)
	// 				tagInfoList = append(tagInfoList, &tagInfo)
	// 			}
	// 		}
	// 	}
	// }

	switch string(resType) {
	case "ALL":
		// for 모든 resource
	default:

		tagInfo, err := FindTag(tagHandler, resType, keyword)
		if err != nil {
			cblogger.Error(err.Error())
			return tagInfoList, err
		}
		tagInfoList = append(tagInfoList, tagInfo)
	}

	return tagInfoList, nil
}

// 1개의 resource Type에 대한 Tag 정보
func FindTag(tagHandler *AlibabaTagHandler, resType irs.RSType, keyword string) (*irs.TagInfo, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, keyword, "FindTag()")
	var tagInfo *irs.TagInfo

	regionID := tagHandler.Region.Region

	alibabaResourceType, err := GetAlibabaResourceType(resType)
	if err != nil {
		return tagInfo, err
	}

	alibabaApiType, err := GetAliTargetApi(resType)
	if err != nil {
		return tagInfo, err
	}

	switch alibabaApiType {
	case "ecs":

		queryParams := map[string]string{}
		queryParams["RegionId"] = regionID
		queryParams["ResourceType"] = alibabaResourceType //string(resType)
		queryParams["Tag.1.Key"] = keyword

		start := call.Start()
		response, err := CallEcsRequest(resType, tagHandler.Client, tagHandler.Region, "DescribeResourceByTags", queryParams)
		LoggingInfo(hiscallInfo, start)

		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
		}
		cblogger.Debug(response.GetHttpContentString())

		resTagResources := AliTagResourcesResponse{}

		tagResponseStr := response.GetHttpContentString()
		err = json.Unmarshal([]byte(tagResponseStr), &resTagResources)
		if err != nil {
			cblogger.Error(err.Error())
			return tagInfo, nil
		}
		cblogger.Debug("resTagResources ", resTagResources)
		cblogger.Debug("resTagResources.AliTagResources ", resTagResources.AliTagResources)
		cblogger.Debug("resTagResources.AliTagResources.Resources ", resTagResources.AliTagResources.Resources)

		for _, aliTagResource := range resTagResources.AliTagResources.Resources {
			cblogger.Debug("aliTagResource ", aliTagResource)
			aTagInfo, err := ExtractTagResourceInfo(&aliTagResource)
			if err != nil {
				cblogger.Error(err.Error())
				continue
			}

			aTagInfo.ResType = resType

			cblogger.Debug("tagInfo ", aTagInfo)
			tagInfo = &aTagInfo
		}
	case "cs": // cs : container service
		clusters, err := aliDescribeClustersV1(tagHandler.CsClient, regionID)
		if err != nil {
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}

		//cblogger.Debug("clusters ", clusters)
		// 모든 cluster를 돌면서 Tag 찾기
		for _, cluster := range clusters {
			cblogger.Debug("inCluster ")
			for _, aliTag := range cluster.Tags {
				//cblogger.Debug("aliTag ", aliTag)
				//cblogger.Debug("keyword ", keyword)
				//cblogger.Debug("aliTag.Key ", *(aliTag.Key))
				if *(aliTag.Key) == keyword {
					var aTagInfo irs.TagInfo
					aTagInfo.ResIId = irs.IID{SystemId: *cluster.ClusterId}
					aTagInfo.ResType = resType

					tagList := []irs.KeyValue{}
					tagList = append(tagList, irs.KeyValue{Key: "TagKey", Value: *aliTag.Key})
					tagList = append(tagList, irs.KeyValue{Key: "TagValue", Value: *aliTag.Value})
					aTagInfo.TagList = tagList
					//cblogger.Debug("append Tag ", &tagInfo)
					tagInfo = &aTagInfo
				}
			}
		}
	}
	return tagInfo, nil
}

/*
*
 */
func validateCreateTag(client *ecs.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID, tag irs.KeyValue) error {
	hiscallInfo := GetCallLogScheme(regionInfo, call.TAG, resIID.NameId, "GetTag()")

	regionID := regionInfo.Region

	// Check Tag Exists
	queryParams := map[string]string{}
	queryParams["RegionId"] = regionID
	queryParams["ResourceType"] = string(resType)
	queryParams["ResourceId"] = resIID.SystemId
	queryParams["Tag.1.Key"] = tag.Key

	start := call.Start()
	response, err := CallEcsRequest(resType, client, regionInfo, "DescribeTags", queryParams)
	LoggingInfo(hiscallInfo, start)

	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
	}
	cblogger.Debug(response.GetHttpContentString())

	return nil
}

// tag 자체가 arr임.
func ExtractTagsDescribeInfo(aliTag *ecs.Tag) (irs.TagInfo, error) {
	var tagInfo irs.TagInfo
	//cblogger.Debug("tag ", aliTag)
	//cblogger.Debug("TagKey ", aliTag.TagKey)
	//cblogger.Debug("TagValue ", aliTag.TagValue)

	tagList := []irs.KeyValue{}
	tagList = append(tagList, irs.KeyValue{Key: "TagKey", Value: aliTag.TagKey})
	tagList = append(tagList, irs.KeyValue{Key: "TagValue", Value: aliTag.TagValue})
	tagInfo.TagList = tagList

	// KeyValueList 추가
	keyValueList, errKeyValue := ConvertKeyValueList(aliTag)
	if errKeyValue != nil {
		cblogger.Error(errKeyValue)
	} else {
		tagInfo.KeyValueList = keyValueList
	}

	cblogger.Debug("tagInfo ", tagInfo)

	return tagInfo, nil
}

// FindTag의 경우 Tag가 아닌 TagResource 를 추출해야 함.
// func ExtractTagResourceInfo(tagResource *ecs.TagResource) (irs.TagInfo, error) {
func ExtractTagResourceInfo(tagResource *AliTagResource) (irs.TagInfo, error) {

	var tagInfo irs.TagInfo
	//cblogger.Debug("tag ", aliTag)
	//cblogger.Debug("TagKey ", aliTag.TagKey)
	//cblogger.Debug("TagValue ", aliTag.TagValue)

	tagInfo.ResType = irs.RSType(tagResource.ResourceType)
	tagInfo.ResIId = irs.IID{SystemId: tagResource.ResourceId}

	tagList := []irs.KeyValue{}
	tagList = append(tagList, irs.KeyValue{Key: "TagKey", Value: tagResource.TagKey})
	tagList = append(tagList, irs.KeyValue{Key: "TagValue", Value: tagResource.TagValue})
	tagInfo.TagList = tagList

	return tagInfo, nil
}
