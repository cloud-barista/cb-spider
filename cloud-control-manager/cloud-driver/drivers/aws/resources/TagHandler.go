// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by devunet@mz.co.kr

// https://github.com/cloud-barista/cb-spider/wiki/Tag-and-Cloud-Driver-API
package resources

import (
	//"errors"
	//"reflect"
	//"strconv"

	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/davecgh/go-spew/spew"

	//"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsTagHandler struct {
	Region    idrv.RegionInfo
	Client    *ec2.EC2
	NLBClient *elbv2.ELBV2
}

// Map of RSType to AWS resource types
var rsTypeToAwsResourceTypeMap = map[irs.RSType]string{
	irs.IMAGE:     "image",
	irs.VPC:       "vpc",
	irs.SUBNET:    "subnet",
	irs.SG:        "security-group",
	irs.KEY:       "key-pair",
	irs.VM:        "instance",
	irs.NLB:       "network-load-balancer",
	irs.DISK:      "volume",
	irs.MYIMAGE:   "image",
	irs.CLUSTER:   "cluster",
	irs.NODEGROUP: "nodegroup",
}

// Map of AWS resource types to RSType for response handling
var awsResourceTypeToRSTypeMap = map[string]irs.RSType{
	"image":                 irs.IMAGE,
	"vpc":                   irs.VPC,
	"subnet":                irs.SUBNET,
	"security-group":        irs.SG,
	"key-pair":              irs.KEY,
	"instance":              irs.VM,
	"network-load-balancer": irs.NLB,
	"volume":                irs.DISK,
	//"image":                 irs.MYIMAGE,
	"cluster":   irs.CLUSTER,
	"nodegroup": irs.NODEGROUP,
}

func (tagHandler *AwsTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	cblogger.Debugf("Req resTyp:[%s] / resIID:[%s] / Tag Key:[%s] / Tag Value:[%s]", resType, resIID, tag.Key, tag.Value)

	if resIID.SystemId == "" || tag.Key == "" {
		msg := "tag will not be add because resIID.SystemId or tag.Key is not provided"
		cblogger.Error(msg)
		return irs.KeyValue{}, errors.New(msg)
	}

	resIID = tagHandler.GetRealResourceId(resType, resIID) // fix some resource id error

	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, resIID.SystemId, "CreateTags()")
	start := call.Start()

	// 리소스에 신규 태그 추가
	result, errtag := tagHandler.Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{&resIID.SystemId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String(tag.Key),
				Value: aws.String(tag.Value),
			},
		},
	})

	if errtag != nil {
		cblogger.Errorf("Failed to add [%s] Tag for [%s]", tag.Key, resIID.SystemId)
		cblogger.Error(errtag)
		return irs.KeyValue{}, errtag
	}
	LoggingInfo(hiscallInfo, start)

	if cblogger.Level.String() == "debug" {
		cblogger.Info(result)
	}

	return tag, nil
}

func (tagHandler *AwsTagHandler) GetAllNLBTags() ([]*irs.TagInfo, error) {
	// Step 1: List all load balancers and store their ARNs and Names
	lbArnToName := make(map[string]string)

	err := tagHandler.NLBClient.DescribeLoadBalancersPages(&elbv2.DescribeLoadBalancersInput{}, func(page *elbv2.DescribeLoadBalancersOutput, lastPage bool) bool {
		for _, lb := range page.LoadBalancers {
			lbArnToName[*lb.LoadBalancerArn] = *lb.LoadBalancerName
		}
		return !lastPage
	})

	if err != nil {
		return nil, fmt.Errorf("failed to describe load balancers: %w", err)
	}

	if len(lbArnToName) == 0 {
		return nil, fmt.Errorf("no load balancers found")
	}

	// Step 2: Describe tags for each load balancer using the GetNLBTags function
	var allTagInfos []*irs.TagInfo

	for arn, name := range lbArnToName {
		resIID := irs.IID{
			NameId:   name,
			SystemId: arn,
		}

		tagInfos, err := tagHandler.GetNLBTags(resIID)
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for load balancer %s: %w", arn, err)
		}

		// Process only if Tag exists
		if len(tagInfos) == 0 {
			continue
		}

		// Convert tags into TagInfo with the correct ResType
		tagInfo := &irs.TagInfo{
			ResType: irs.NLB,
			ResIId:  resIID,
			TagList: tagInfos,
		}

		allTagInfos = append(allTagInfos, tagInfo)
	}

	return allTagInfos, nil
}

func (tagHandler *AwsTagHandler) GetNLBTags(resIID irs.IID) ([]irs.KeyValue, error) {
	input := &elbv2.DescribeTagsInput{
		ResourceArns: []*string{
			aws.String(resIID.SystemId),
		},
	}

	result, err := tagHandler.NLBClient.DescribeTags(input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe load balancer tags: %w", err)
	}

	if len(result.TagDescriptions) == 0 {
		return nil, fmt.Errorf("no tags found for load balancer: %s", resIID.SystemId)
	}

	if cblogger.Level.String() == "debug" {
		cblogger.Debug(result)
	}

	var retTagList []irs.KeyValue
	for _, tag := range result.TagDescriptions[0].Tags {
		retTagList = append(retTagList, irs.KeyValue{
			Key:   aws.StringValue(tag.Key),
			Value: aws.StringValue(tag.Value),
		})
	}

	return retTagList, nil
}

func (tagHandler *AwsTagHandler) ListTag(resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
	cblogger.Debugf("Req resTyp:[%s] / resIID:[%s]", resType, resIID)

	if resIID.SystemId == "" {
		msg := "resIID.SystemId is not provided"
		cblogger.Error(msg)
		return nil, errors.New(msg)
	}

	if resType == irs.NLB {
		return tagHandler.GetNLBTags(resIID)
	}

	resIID = tagHandler.GetRealResourceId(resType, resIID) // fix some resource id error
	input := &ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("resource-id"),
				Values: []*string{
					aws.String(resIID.SystemId),
				},
			},
		},
	}
	if cblogger.Level.String() == "debug" {
		cblogger.Debug(input)
	}

	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, resIID.SystemId, "DescribeTags()")
	start := call.Start()

	result, errtag := tagHandler.Client.DescribeTags(input)
	if errtag != nil {
		cblogger.Errorf("Failed to look up tags for [%s]", resIID.NameId)
		cblogger.Error(errtag)
		LoggingError(hiscallInfo, errtag)
		return nil, errtag
	}
	LoggingInfo(hiscallInfo, start)

	if cblogger.Level.String() == "debug" {
		cblogger.Debug(result)
	}

	var retTagList []irs.KeyValue
	for _, tag := range result.Tags {
		retTagList = append(retTagList, irs.KeyValue{
			Key:   aws.StringValue(tag.Key),
			Value: aws.StringValue(tag.Value),
		})
	}

	return retTagList, nil
}

// describetags
func (tagHandler *AwsTagHandler) GetTag(resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	cblogger.Debugf("Req resTyp:[%s] / resIID:[%s] / key:[%s]", resType, resIID, key)

	if resIID.SystemId == "" || key == "" {
		msg := "resIID.SystemId or key is not provided"
		cblogger.Error(msg)
		return irs.KeyValue{}, errors.New(msg)
	}
	resIID = tagHandler.GetRealResourceId(resType, resIID) // fix some resource id error

	input := &ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("resource-id"),
				Values: []*string{
					aws.String(resIID.SystemId),
				},
			},
			{
				Name: aws.String("key"),
				Values: []*string{
					aws.String(key),
				},
			},
		},
	}

	if cblogger.Level.String() == "debug" {
		cblogger.Debug(input)
	}

	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, resIID.SystemId, "DescribeTags()")
	start := call.Start()

	result, errtag := tagHandler.Client.DescribeTags(input)
	if errtag != nil {
		cblogger.Errorf("Failed to lookup the [%s] tag key of an [%s] object", key, resIID.NameId)
		cblogger.Error(errtag)
		LoggingError(hiscallInfo, errtag)
		return irs.KeyValue{}, errtag
	}
	LoggingInfo(hiscallInfo, start)

	if cblogger.Level.String() == "debug" {
		cblogger.Info("---------------------")
		cblogger.Info(result)
		cblogger.Info("---------------------")
	}

	if len(result.Tags) == 0 {
		msg := "tag with key " + key + " not found"
		cblogger.Error(msg)
		return irs.KeyValue{}, errors.New(msg)
	}

	var retTag irs.KeyValue
	for _, tag := range result.Tags {
		if aws.StringValue(tag.Key) == key {
			retTag.Key = aws.StringValue(tag.Key)
			retTag.Value = aws.StringValue(tag.Value)
			break
		}
	}

	return retTag, nil
}

// Handles targets that have a Name-based to Id conversion task, like Keypair.
// Keypair should use id, not name.
func (tagHandler *AwsTagHandler) GetRealResourceId(resType irs.RSType, resIID irs.IID) irs.IID {

	cblogger.Debugf("resType : [%s] / resIID : [%s]", resType, resIID.SystemId)

	if resType != irs.KEY {
		return resIID
	}

	//
	// Keypair should use id, not name, when using Tag-related APIs.
	//
	input := &ec2.DescribeKeyPairsInput{
		KeyNames: []*string{
			aws.String(resIID.SystemId),
		},
	}

	result, err := tagHandler.Client.DescribeKeyPairs(input)
	spew.Dump(result)
	if err != nil {
		cblogger.Error(err)
		return resIID
	}

	if len(result.KeyPairs) > 0 {
		newIID := irs.IID{NameId: *result.KeyPairs[0].KeyName, SystemId: *result.KeyPairs[0].KeyPairId}
		return newIID
	}

	return resIID
}

func (tagHandler *AwsTagHandler) RemoveTag(resType irs.RSType, resIID irs.IID, key string) (bool, error) {
	cblogger.Debugf("Req resTyp:[%s] / resIID:[%s] / key:[%s]", resType, resIID, key)

	if resIID.SystemId == "" || key == "" {
		msg := "resIID.SystemId or key is not provided"
		cblogger.Error(msg)
		return false, errors.New(msg)
	}
	resIID = tagHandler.GetRealResourceId(resType, resIID) // fix some resource id error

	input := &ec2.DeleteTagsInput{
		Resources: []*string{
			aws.String(resIID.SystemId),
		},
		Tags: []*ec2.Tag{
			{
				Key: aws.String(key),
			},
		},
	}

	if cblogger.Level.String() == "debug" {
		cblogger.Debug(input)
	}

	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, resIID.SystemId, "DeleteTags()")
	start := call.Start()

	result, errtag := tagHandler.Client.DeleteTags(input)
	if errtag != nil {
		cblogger.Errorf("Failed to delete [%s] tag key of an [%s] object", key, resIID.NameId)
		cblogger.Error(errtag)
		LoggingError(hiscallInfo, errtag)
		return false, errtag
	}
	LoggingInfo(hiscallInfo, start)

	if cblogger.Level.String() == "debug" {
		cblogger.Info(result)
	}

	return true, nil
}

// Extracts a list of Key or Value tags corresponding to keyword from the tagInfos array.
func (tagHandler *AwsTagHandler) ExtractTagKeyValue(tagInfos []*irs.TagInfo, keyword string) []*irs.TagInfo {
	var matchingTagInfos []*irs.TagInfo
	cblogger.Debugf("tagInfos count : [%d] / keyword : [%s]", len(tagInfos), keyword)
	if cblogger.Level.String() == "debug" {
		spew.Dump(tagInfos)
	}

	/*
		for _, tagInfo := range tagInfos {
			for _, kv := range tagInfo.TagList {
				if kv.Key == keyword || kv.Value == keyword {
					matchingTagInfos = append(matchingTagInfos, tagInfo)
					break //  If any match, add that tagInfo and move on to the next tagInfo
				}
			}
		}
	*/

	// The DescribeTags() API used by FindTag() only includes matching Keys, so we modified it with the same logic.
	for _, tagInfo := range tagInfos {
		var filteredTagList []irs.KeyValue

		for _, kv := range tagInfo.TagList {
			if kv.Key == keyword || kv.Value == keyword {
				filteredTagList = append(filteredTagList, kv)
			}
		}

		if len(filteredTagList) > 0 {
			matchingTagInfo := &irs.TagInfo{
				ResType:      tagInfo.ResType,
				ResIId:       tagInfo.ResIId,
				TagList:      filteredTagList,
				KeyValueList: tagInfo.KeyValueList,
			}
			matchingTagInfos = append(matchingTagInfos, matchingTagInfo)
		}
	}
	return matchingTagInfos
}

// Find tags by tag key or value
// resType: ALL | VPC, SUBNET, etc.,.
// keyword: The keyword to search for in the tag key or value.
// if you want to find all tags, set keyword to "" or "*".
func (tagHandler *AwsTagHandler) FindTag(resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	cblogger.Debugf("resType : [%s] / keyword : [%s]", resType, keyword)

	var tagInfos []*irs.TagInfo
	var filters []*ec2.Filter

	// Add resource type filter if resType is not ALL
	if resType != irs.ALL {
		if resType == irs.NLB {
			// Add a list of NLB Tags if this is a all search
			if keyword == "" || keyword == "*" {
				return tagHandler.GetAllNLBTags()
			} else {
				nlbTaginfos, _ := tagHandler.GetAllNLBTags()
				//spew.Dump(nlbTaginfos)
				return tagHandler.ExtractTagKeyValue(nlbTaginfos, keyword), nil
			}
		}

		if awsResType, ok := rsTypeToAwsResourceTypeMap[resType]; ok {
			filters = append(filters, &ec2.Filter{
				Name: aws.String("resource-type"),
				Values: []*string{
					aws.String(awsResType),
				},
			})
		} else {
			return nil, fmt.Errorf("unsupported resource type: %s", resType)
		}
	}

	tagInfoMap := make(map[string]*irs.TagInfo)

	// Function to process tags and add them to tagInfoMap
	processTags := func(result *ec2.DescribeTagsOutput) {
		if cblogger.Level.String() == "debug" {
			//cblogger.Debug(result)
			cblogger.Debug("=================================")
			spew.Dump(result)
			cblogger.Debug("=================================")
		}

		for _, tag := range result.Tags {
			resID := aws.StringValue(tag.ResourceId)

			awsResType := aws.StringValue(tag.ResourceType)
			rType, exists := awsResourceTypeToRSTypeMap[awsResType]
			if !exists {
				//@TODO - 변환 실패한 리소스의 경우 UNKNOWN을 만들거나 에러 로그만 찍거나 결정 필요할 듯
				cblogger.Errorf("No RSType matching [%s] found.", awsResType)

				rType = irs.RSType(awsResType) // Use the raw AWS resource type if not mapped
			}

			if _, exists := tagInfoMap[resID]; !exists {
				tagInfoMap[resID] = &irs.TagInfo{
					ResType: rType,
					ResIId: irs.IID{
						SystemId: resID,
					},
				}
			}
			tagInfoMap[resID].TagList = append(tagInfoMap[resID].TagList, irs.KeyValue{
				Key:   aws.StringValue(tag.Key),
				Value: aws.StringValue(tag.Value),
			})
		}
	}

	// Search by tag-key if keyword is not empty or "*"
	if keyword != "" && keyword != "*" {
		keyInput := &ec2.DescribeTagsInput{
			Filters: append(filters, &ec2.Filter{
				Name: aws.String("tag-key"),
				Values: []*string{
					aws.String(keyword),
				},
			}),
		}

		if cblogger.Level.String() == "debug" {
			cblogger.Debug(keyInput)
		}

		hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, keyword, "FindTag(key):DescribeTags()")
		start := call.Start()

		keyResult, err := tagHandler.Client.DescribeTags(keyInput)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, fmt.Errorf("failed to describe tags by key: %w", err)
		}
		LoggingInfo(hiscallInfo, start)
		processTags(keyResult)

		valueInput := &ec2.DescribeTagsInput{
			Filters: append(filters, &ec2.Filter{
				Name: aws.String("tag-value"),
				Values: []*string{
					aws.String(keyword),
				},
			}),
		}

		if cblogger.Level.String() == "debug" {
			cblogger.Debug(valueInput)
		}

		hiscallInfo2 := GetCallLogScheme(tagHandler.Region, call.TAG, keyword, "FindTag(value):DescribeTags()")
		start2 := call.Start()

		valueResult, err := tagHandler.Client.DescribeTags(valueInput)
		if err != nil {
			LoggingError(hiscallInfo2, err)
			return nil, fmt.Errorf("failed to describe tags by value: %w", err)
		}
		LoggingInfo(hiscallInfo2, start2)
		processTags(valueResult)
	} else {
		// Search all tags if keyword is empty or "*"
		input := &ec2.DescribeTagsInput{
			Filters: filters,
		}

		hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, keyword, "FindTag(all):DescribeTags()")
		start := call.Start()

		result, err := tagHandler.Client.DescribeTags(input)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, fmt.Errorf("failed to describe tags: %w", err)
		}
		LoggingInfo(hiscallInfo, start)
		processTags(result)
	}

	//var tagInfos []*irs.TagInfo
	for _, tagInfo := range tagInfoMap {
		tagInfos = append(tagInfos, tagInfo)
	}

	// Add a list of NLB Tags if this is a all search
	if resType == irs.ALL {
		nlbTaginfos, _ := tagHandler.GetAllNLBTags()
		if keyword != "" && keyword != "*" {
			nlbTaginfos = tagHandler.ExtractTagKeyValue(nlbTaginfos, keyword)
		}
		tagInfos = append(tagInfos, nlbTaginfos...)
	}

	return tagInfos, nil
}
