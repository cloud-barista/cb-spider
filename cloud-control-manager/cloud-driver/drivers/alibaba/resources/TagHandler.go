package resources

import (
	"encoding/json"
	"errors"
	"fmt"

	cs "github.com/alibabacloud-go/cs-20151215/v4/client" // cs  : container service
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs" // ecs : elastic compute service
	"github.com/aliyun/alibaba-cloud-sdk-go/services/nas" // nas : network attached storage
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaTagHandler struct {
	Region    idrv.RegionInfo
	Client    *ecs.Client
	CsClient  *cs.Client
	VpcClient *vpc.Client
	SlbClient *slb.Client
	NasClient *nas.Client
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

	// AliTagResources AliTagResources `json:"Resources" xml:"Resources"`
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
type VSwitchIds struct {
	VSwitchId []string `json:"VSwitchId"`
}

type SecondaryCidrBlocks struct {
	SecondaryCidrBlock []string `json:"SecondaryCidrBlock"`
}

type RouterTableIds struct {
	RouterTableIds []string `json:"RouterTableIds"`
}

type UserCidrs struct {
	UserCidr []string `json:"UserCidr"`
}

type NatGatewayIds struct {
	NatGatewayIds []string `json:"NatGatewayIds"`
}

type Vpc struct {
	Status              string              `json:"Status"`
	IsDefault           bool                `json:"IsDefault"`
	CenStatus           string              `json:"CenStatus"`
	Description         string              `json:"Description"`
	ResourceGroupId     string              `json:"ResourceGroupId"`
	VSwitchIds          VSwitchIds          `json:"VSwitchIds"`
	SecondaryCidrBlocks SecondaryCidrBlocks `json:"SecondaryCidrBlocks"`
	CidrBlock           string              `json:"CidrBlock"`
	RouterTableIds      RouterTableIds      `json:"RouterTableIds"`
	UserCidrs           UserCidrs           `json:"UserCidrs"`
	NetworkAclNum       int                 `json:"NetworkAclNum"`
	AdvancedResource    bool                `json:"AdvancedResource"`
	VRouterId           string              `json:"VRouterId"`
	NatGatewayIds       NatGatewayIds       `json:"NatGatewayIds"`
	VpcId               string              `json:"VpcId"`
	OwnerId             int64               `json:"OwnerId"`
	CreationTime        string              `json:"CreationTime"`
	VpcName             string              `json:"VpcName"`
	EnabledIpv6         bool                `json:"EnabledIpv6"`
	RegionId            string              `json:"RegionId"`
	Ipv6CidrBlock       string              `json:"Ipv6CidrBlock"`
	Tags                Tags                `json:"Tags"`
}

type Vpcs struct {
	Vpc []Vpc `json:"Vpc"`
}

type DescribeVpcsResponse struct {
	TotalCount int    `json:"TotalCount"`
	PageSize   int    `json:"PageSize"`
	RequestId  string `json:"RequestId"`
	PageNumber int    `json:"PageNumber"`
	Vpcs       Vpcs   `json:"Vpcs"`
}

type Tag struct {
	Value string `json:"Value"`
	Key   string `json:"Key"`
}

type Tags struct {
	Tag []Tag `json:"Tag"`
}

type RouteTable struct {
	RouteTableId   string `json:"RouteTableId"`
	RouteTableType string `json:"RouteTableType"`
}

type VSwitch struct {
	Status                  string     `json:"Status"`
	IsDefault               bool       `json:"IsDefault"`
	Description             string     `json:"Description"`
	ResourceGroupId         string     `json:"ResourceGroupId"`
	ZoneId                  string     `json:"ZoneId"`
	NetworkAclId            string     `json:"NetworkAclId"`
	AvailableIpAddressCount int        `json:"AvailableIpAddressCount"`
	VSwitchId               string     `json:"VSwitchId"`
	CidrBlock               string     `json:"CidrBlock"`
	RouteTable              RouteTable `json:"RouteTable"`
	VpcId                   string     `json:"VpcId"`
	OwnerId                 int64      `json:"OwnerId"`
	CreationTime            string     `json:"CreationTime"`
	VSwitchName             string     `json:"VSwitchName"`
	Ipv6CidrBlock           string     `json:"Ipv6CidrBlock"`
	Tags                    Tags       `json:"Tags"`
	ShareType               string     `json:"ShareType"`
}

type VSwitches struct {
	VSwitch []VSwitch `json:"VSwitch"`
}
type DescribeVSwitchesResponse struct {
	TotalCount int       `json:"TotalCount"`
	PageSize   int       `json:"PageSize"`
	RequestId  string    `json:"RequestId"`
	PageNumber int       `json:"PageNumber"`
	VSwitches  VSwitches `json:"VSwitches"`
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

	case "slb":

		response, err := aliAddSlbTag(tagHandler.SlbClient, tagHandler.Region, resType, resIID, tag)
		if err != nil {
			return tag, err
		}
		cblogger.Debug("AddNlbTags response", response)

	case "nas":
		response, err := aliAddNasTag(tagHandler.NasClient, tagHandler.Region, resType, resIID, tag)
		if err != nil {
			return tag, err
		}
		cblogger.Debug("AddNasTags response", response)

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
		// cblogger.Debug("resTagResources.AliTagResources ", resTagResources.AliTagResources)
		// cblogger.Debug("resTagResources.AliTagResources.Resources ", resTagResources.AliTagResources.Resources)

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
	case "nas":
		response, err := aliNasListTag(tagHandler.NasClient, tagHandler.Region, resType, resIID)
		if err != nil {
			cblogger.Error(err.Error())
			return tagInfoList, nil
		}
		cblogger.Debug("aliNasListTag ", response)

		// NAS 태그 응답에서 태그 정보 추출
		for _, nasTag := range response.Tags.Tag {
			aTagInfo := irs.KeyValue{Key: nasTag.Key, Value: nasTag.Value}
			cblogger.Debug("tagInfo ", aTagInfo)
			tagInfoList = append(tagInfoList, aTagInfo)
		}
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
		// cblogger.Debug("resTagResources.AliTagResources ", resTagResources.AliTagResources)
		// cblogger.Debug("resTagResources.AliTagResources.Resources ", resTagResources.AliTagResources.Resources)

		for _, aliTagResource := range response.TagResource {
			cblogger.Debug("aliTagResource ", aliTagResource)

			tagInfo = irs.KeyValue{Key: *aliTagResource.TagKey, Value: *aliTagResource.TagValue}
			cblogger.Debug("tagInfo ", tagInfo)

		}
	case "nas":
		response, err := aliNasListTag(tagHandler.NasClient, tagHandler.Region, resType, resIID)
		if err != nil {
			cblogger.Error(err.Error())
			return tagInfo, nil
		}
		cblogger.Debug("aliNasTag ", response)

		// 특정 키에 해당하는 태그 찾기
		for _, nasTag := range response.Tags.Tag {
			if nasTag.Key == key {
				tagInfo = irs.KeyValue{Key: nasTag.Key, Value: nasTag.Value}
				cblogger.Debug("tagInfo ", tagInfo)
				break
			}
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

	case "slb":

		// 삭제할 수 있는 tag의 형태가
		//[{"TagKey":"Key1","TagValue":"Value1"},{"TagKey":"Key2","TagValue":"Value2"}]
		// 따라서 해당 nlb 조회 후 tag 포멧을 맞춰서 삭제함
		tagListResponse, err := DescribeDescribeNlbTags(tagHandler.SlbClient, tagHandler.Region, resType, resIID, key)
		if err != nil {
			// DescribeTags API 호출 실패 시 에러 처리
			cblogger.Error("Error in DescribeDescribeNlbTags:", err)
			return false, err
		}
		cblogger.Debug("tagListResponse", tagListResponse)

		// var tagResponse DescribeTagsResponse
		var tagResponse slb.DescribeTagsResponse
		err = json.Unmarshal([]byte(tagListResponse.GetHttpContentString()), &tagResponse)
		if err != nil {
			cblogger.Error("Failed to parse DescribeTags response:", err)
			return false, err
		}

		var targetTagValue string
		for _, tag := range tagResponse.TagSets.TagSet {
			if tag.TagKey == key {
				targetTagValue = tag.TagValue
				break
			}
		}

		// 만약 TagKey에 해당하는 TagValue를 찾지 못했다면
		if targetTagValue == "" {
			cblogger.Error("TagKey not found in DescribeTags response")
			return false, fmt.Errorf("TagKey %s not found", key)
		}

		queryParams := map[string]string{}
		queryParams["RegionId"] = regionID
		queryParams["ResourceType"] = alibabaResourceType
		queryParams["LoadBalancerId"] = resIID.SystemId
		queryParams["Tags"] = fmt.Sprintf(`[{"TagKey":"%s","TagValue":"%s"}]`, key, targetTagValue)

		start := call.Start()
		response, err := CallNlbRequest(resType, tagHandler.SlbClient, tagHandler.Region, "RemoveTags", queryParams)
		LoggingInfo(hiscallInfo, start)

		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
		}
		cblogger.Debug(response.GetHttpContentString())

		cblogger.Infof("Successfully deleted %q Task\n", resIID.SystemId)

		expectStatus := false // 예상되는 상태 : 없어야 하므로 fasle
		result, err := WaitForNlbTagExist(tagHandler.SlbClient, tagHandler.Region, resType, resIID, key, expectStatus)
		if err != nil {
			return false, err
		}
		cblogger.Debug("Expect Status ", expectStatus, ", result Status ", result)
		if !result {
			return false, errors.New("waitForTagExist Error ")
		}
	case "nas":
		response, err := aliRemoveNasTag(tagHandler.NasClient, tagHandler.Region, resType, resIID, key)
		if err != nil {
			return false, err
		}
		cblogger.Debug("RemoveNasTags response", response)
	}
	return true, nil
}

// Find tags by tag key or value
// resType: ALL | VPC, SUBNET, etc.,.  ecs기준 : Ddh, Disk, Eni, Image, Instance, KeyPair, LaunchTemplate, ReservedInstance, Securitygroup, Snapshot, SnapshotPolicy, Volume,
// keyword: The keyword to search for in the tag key or value.
// if you want to find all tags, set keyword to "" or "*".
// 해당 Resource Type에 tag가 있는 것들. ListTag는 resourceId가 있으나 당 function은 더 넓음
func (tagHandler *AlibabaTagHandler) FindTag(resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {

	var tagInfoList []*irs.TagInfo

	regionInfo := tagHandler.Region
	regionID := tagHandler.Region.Region

	alibabaResourceType, err := GetAlibabaResourceType(resType)

	if err != nil {
		return tagInfoList, err
	}

	cblogger.Debug("resType : ", resType)
	switch resType {

	case "VM", irs.VM:
		responseTagList, err := aliEcsTagList(tagHandler.Client, regionInfo, alibabaResourceType, resType, keyword)
		if err != nil {
			cblogger.Error(err)
		}
		cblogger.Debug("aliEcsTag response : ", responseTagList)

		tagInfoList = append(tagInfoList, responseTagList...)

	case "KEY", irs.KEY:
		responseTagList, err := aliEcsTagList(tagHandler.Client, regionInfo, alibabaResourceType, resType, keyword)
		if err != nil {
			cblogger.Error(err)
		}
		cblogger.Debug("aliEcsTag response : ", responseTagList)

		tagInfoList = append(tagInfoList, responseTagList...)

	case "SG", irs.SG:
		responseTagList, err := aliEcsTagList(tagHandler.Client, regionInfo, alibabaResourceType, resType, keyword)
		if err != nil {
			cblogger.Error(err)
		}
		cblogger.Debug("aliEcsTag response : ", responseTagList)

		tagInfoList = append(tagInfoList, responseTagList...)

	case "DISK", irs.DISK:
		responseTagList, err := aliEcsTagList(tagHandler.Client, regionInfo, alibabaResourceType, resType, keyword)
		if err != nil {
			cblogger.Error(err)
		}
		cblogger.Debug("aliEcsTag response : ", responseTagList)

		tagInfoList = append(tagInfoList, responseTagList...)

	case "MYIMAGE", irs.MYIMAGE:
		responseTagList, err := aliMyImageTagList(tagHandler.Client, regionInfo, keyword)
		if err != nil {
			cblogger.Error(err)
		}
		cblogger.Debug("aliEcsTag response : ", responseTagList)

		tagInfoList = append(tagInfoList, responseTagList...)

	case "VPC", irs.VPC:
		responseTagList, err := aliVpcTagList(tagHandler.VpcClient, regionInfo, alibabaResourceType, resType, keyword)
		if err != nil {
			cblogger.Error(err)
		}
		cblogger.Debug("aliEcsTag response : ", responseTagList)

		tagInfoList = append(tagInfoList, responseTagList...)

	case "SUBNET", irs.SUBNET:
		responseTagList, err := aliSubnetTagList(tagHandler.VpcClient, regionInfo, alibabaResourceType, resType, keyword)
		if err != nil {
			cblogger.Error(err)
		}
		cblogger.Debug("aliEcsTag response : ", responseTagList)

		tagInfoList = append(tagInfoList, responseTagList...)

	case "CLUSTER", irs.CLUSTER: // cs : container service
		responseTagList, err := aliClusterTagList(tagHandler.CsClient, regionInfo, resType, keyword)
		if err != nil {
			cblogger.Error(err)
		}
		cblogger.Debug("aliEcsTag response : ", responseTagList)

		tagInfoList = append(tagInfoList, responseTagList...)

	case "FILESYSTEM", irs.FILESYSTEM:
		responseTagList, err := aliNasTagList(tagHandler.NasClient, tagHandler.Region, resType, keyword)
		if err != nil {
			cblogger.Error(err)
		}
		cblogger.Debug("aliNasTag response : ", responseTagList)

		tagInfoList = append(tagInfoList, responseTagList...)

	case "ALL", irs.ALL:
		// 모든 자원 유형을 포함하는 슬라이스를 선언
		allResourceTypes := []irs.RSType{irs.VM, irs.KEY, irs.SG, irs.DISK, irs.MYIMAGE, irs.VPC, irs.SUBNET, irs.CLUSTER, irs.FILESYSTEM}

		// 각 자원 유형별로 태그 정보 조회
		for _, resourceType := range allResourceTypes {
			switch resourceType {
			case irs.VM, irs.KEY, irs.SG, irs.DISK:
				responseTagList, err := aliEcsTagList(tagHandler.Client, regionInfo, alibabaResourceType, resourceType, keyword)
				if err != nil {
					cblogger.Errorf("Error retrieving tags for %s: %v", resourceType, err)
				} else {
					tagInfoList = append(tagInfoList, responseTagList...)
				}

			case irs.MYIMAGE:
				responseTagList, err := aliMyImageTagList(tagHandler.Client, regionInfo, keyword)
				if err != nil {
					cblogger.Errorf("Error retrieving tags for MYIMAGE: %v", err)
				} else {
					tagInfoList = append(tagInfoList, responseTagList...)
				}

			case irs.VPC:
				responseTagList, err := aliVpcTagList(tagHandler.VpcClient, regionInfo, alibabaResourceType, resourceType, keyword)
				if err != nil {
					cblogger.Errorf("Error retrieving tags for VPC: %v", err)
				} else {
					tagInfoList = append(tagInfoList, responseTagList...)
				}

			case irs.SUBNET:
				responseTagList, err := aliSubnetTagList(tagHandler.VpcClient, regionInfo, alibabaResourceType, resourceType, keyword)
				if err != nil {
					cblogger.Errorf("Error retrieving tags for SUBNET: %v", err)
				} else {
					tagInfoList = append(tagInfoList, responseTagList...)
				}

			case irs.CLUSTER:
				// CLUSTER 태그 정보 조회 로직
				clusters, err := aliDescribeClustersV1(tagHandler.CsClient, regionID)
				if err != nil {
					cblogger.Errorf("Error retrieving clusters: %v", err)
				} else {
					for _, cluster := range clusters {
						for _, aliTag := range cluster.Tags {
							if *(aliTag.Key) == keyword {
								aTagInfo := irs.TagInfo{
									ResIId:  irs.IID{SystemId: *cluster.ClusterId},
									ResType: resourceType,
									TagList: []irs.KeyValue{
										{Key: "TagKey", Value: *aliTag.Key},
										{Key: "TagValue", Value: *aliTag.Value},
									},
								}
								tagInfoList = append(tagInfoList, &aTagInfo)
							}
						}
					}
				}

			case irs.FILESYSTEM:
				responseTagList, err := aliNasTagList(tagHandler.NasClient, regionInfo, resourceType, keyword)
				if err != nil {
					cblogger.Errorf("Error retrieving tags for FILESYSTEM: %v", err)
				} else {
					tagInfoList = append(tagInfoList, responseTagList...)
				}
			}
		}
	}
	return tagInfoList, nil
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

// NAS 태그 관리 함수들

// aliAddNasTag adds a tag to NAS file system
func aliAddNasTag(client *nas.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (*nas.AddTagsResponse, error) {
	request := nas.CreateAddTagsRequest()
	request.Scheme = "https"
	request.FileSystemId = resIID.SystemId
	request.Tag = &[]nas.AddTagsTag{
		{
			Key:   tag.Key,
			Value: tag.Value,
		},
	}

	response, err := client.AddTags(request)
	if err != nil {
		return nil, fmt.Errorf("failed to add tag to NAS file system: %w", err)
	}

	return response, nil
}

// aliNasListTag lists tags for NAS file system
func aliNasListTag(client *nas.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID) (*nas.DescribeTagsResponse, error) {
	request := nas.CreateDescribeTagsRequest()
	request.Scheme = "https"
	request.FileSystemId = resIID.SystemId

	response, err := client.DescribeTags(request)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags for NAS file system: %w", err)
	}

	return response, nil
}

// aliNasTagList lists all NAS file systems and their tags that match the keyword
func aliNasTagList(client *nas.Client, regionInfo idrv.RegionInfo, resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	var tagInfoList []*irs.TagInfo

	// 먼저 모든 파일시스템을 조회
	request := nas.CreateDescribeFileSystemsRequest()
	request.Scheme = "https"

	response, err := client.DescribeFileSystems(request)
	if err != nil {
		return nil, fmt.Errorf("failed to describe file systems: %w", err)
	}

	// 각 파일시스템의 태그를 조회
	for _, fs := range response.FileSystems.FileSystem {
		// 파일시스템의 태그 조회
		tagRequest := nas.CreateDescribeTagsRequest()
		tagRequest.Scheme = "https"
		tagRequest.FileSystemId = fs.FileSystemId

		tagResponse, err := client.DescribeTags(tagRequest)
		if err != nil {
			cblogger.Warnf("Failed to get tags for file system %s: %v", fs.FileSystemId, err)
			continue
		}

		// 키워드와 일치하는 태그 찾기
		for _, tag := range tagResponse.Tags.Tag {
			if keyword == "" || keyword == "*" || tag.Key == keyword || tag.Value == keyword {
				aTagInfo := irs.TagInfo{
					ResIId:  irs.IID{SystemId: fs.FileSystemId},
					ResType: resType,
					TagList: []irs.KeyValue{
						{Key: "TagKey", Value: tag.Key},
						{Key: "TagValue", Value: tag.Value},
					},
				}
				tagInfoList = append(tagInfoList, &aTagInfo)
			}
		}
	}

	return tagInfoList, nil
}

// aliRemoveNasTag removes a tag from NAS file system
func aliRemoveNasTag(client *nas.Client, regionInfo idrv.RegionInfo, resType irs.RSType, resIID irs.IID, key string) (*nas.RemoveTagsResponse, error) {
	request := nas.CreateRemoveTagsRequest()
	request.Scheme = "https"
	request.FileSystemId = resIID.SystemId
	request.Tag = &[]nas.RemoveTagsTag{
		{
			Key: key,
		},
	}

	response, err := client.RemoveTags(request)
	if err != nil {
		return nil, fmt.Errorf("failed to remove tag from NAS file system: %w", err)
	}

	return response, nil
}
