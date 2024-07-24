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
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/elbv2"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsTagHandler struct {
	Region    idrv.RegionInfo
	Client    *ec2.EC2
	NLBClient *elbv2.ELBV2
	EKSClient *eks.EKS
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

	if resType == irs.NLB || resType == irs.CLUSTER {
		var err error
		if resType == irs.NLB {
			err = tagHandler.AddNLBTag(resIID, tag)
		} else if resType == irs.CLUSTER {
			err = tagHandler.AddClusterTag(resIID, tag)
		}

		if err != nil {
			return irs.KeyValue{}, err
		}
		return tagHandler.GetTag(resType, resIID, tag.Key)
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

func (tagHandler *AwsTagHandler) AddNLBTag(resIID irs.IID, tag irs.KeyValue) error {
	input := &elbv2.AddTagsInput{
		ResourceArns: []*string{aws.String(resIID.SystemId)},
		Tags: []*elbv2.Tag{
			{
				Key:   aws.String(tag.Key),
				Value: aws.String(tag.Value),
			},
		},
	}

	_, err := tagHandler.NLBClient.AddTags(input)
	if err != nil {
		return fmt.Errorf("failed to add tag to NLB: %w", err)
	}

	return nil
}

func (tagHandler *AwsTagHandler) AddClusterTag(resIID irs.IID, tag irs.KeyValue) error {
	input := &eks.TagResourceInput{
		ResourceArn: aws.String(resIID.SystemId),
		Tags: map[string]*string{
			tag.Key: aws.String(tag.Value),
		},
	}

	_, err := tagHandler.EKSClient.TagResource(input)
	if err != nil {
		return fmt.Errorf("failed to add tag to EKS cluster: %w", err)
	}

	return nil
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

// GetAllClusterTags retrieves tags for all EKS clusters.
func (tagHandler *AwsTagHandler) GetAllClusterTags() ([]*irs.TagInfo, error) {
	var allTagInfos []*irs.TagInfo

	err := tagHandler.EKSClient.ListClustersPages(&eks.ListClustersInput{}, func(page *eks.ListClustersOutput, lastPage bool) bool {
		for _, clusterName := range page.Clusters {
			resIID := irs.IID{
				NameId:   *clusterName,
				SystemId: *clusterName,
			}

			tagInfos, err := tagHandler.GetClusterTags(resIID)
			if err != nil {
				cblogger.Errorf("failed to get tags for EKS cluster %s: %v", *clusterName, err)
				continue
			}

			// Process only if Tag exists
			if len(tagInfos) == 0 {
				continue
			}

			tagInfo := &irs.TagInfo{
				ResType: irs.CLUSTER,
				ResIId:  resIID,
				TagList: tagInfos,
			}

			allTagInfos = append(allTagInfos, tagInfo)
		}
		return !lastPage
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list EKS clusters: %w", err)
	}

	if len(allTagInfos) == 0 {
		return nil, fmt.Errorf("no EKS clusters found")
	}

	return allTagInfos, nil
}

func (tagHandler *AwsTagHandler) GetClusterTags(resIID irs.IID, key ...string) ([]irs.KeyValue, error) {
	cblogger.Debugf("Req resIID:[%s]", resIID)
	input := &eks.DescribeClusterInput{
		Name: aws.String(resIID.SystemId),
	}

	result, err := tagHandler.EKSClient.DescribeCluster(input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe EKS cluster: %w", err)
	}

	if cblogger.Level.String() == "debug" {
		cblogger.Debug(result)
	}

	var retTagList []irs.KeyValue
	if len(key) > 0 { // If the key value exists
		specificKey := key[0]
		if value, exists := result.Cluster.Tags[specificKey]; exists {
			retTagList = append(retTagList, irs.KeyValue{
				Key:   specificKey,
				Value: *value,
			})
		}
	} else { // If the key value not exists
		for k, v := range result.Cluster.Tags {
			retTagList = append(retTagList, irs.KeyValue{
				Key:   k,
				Value: *v,
			})
		}
	}

	return retTagList, nil
}

func (tagHandler *AwsTagHandler) GetNLBTags(resIID irs.IID, key ...string) ([]irs.KeyValue, error) {
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

	cblogger.Debug(result)

	var retTagList []irs.KeyValue
	if len(key) > 0 { // If the key value exists
		specificKey := key[0]
		for _, tag := range result.TagDescriptions[0].Tags {
			if aws.StringValue(tag.Key) == specificKey {
				retTagList = append(retTagList, irs.KeyValue{
					Key:   aws.StringValue(tag.Key),
					Value: aws.StringValue(tag.Value),
				})
				break
			}
		}
	} else { // If the key value not exists
		for _, tag := range result.TagDescriptions[0].Tags {
			retTagList = append(retTagList, irs.KeyValue{
				Key:   aws.StringValue(tag.Key),
				Value: aws.StringValue(tag.Value),
			})
		}
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
	} else if resType == irs.CLUSTER {
		return tagHandler.GetClusterTags(resIID)
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

	// NLB and Cluster use different APIs
	if resType == irs.NLB || resType == irs.CLUSTER {
		var tagList []irs.KeyValue
		var err error

		if resType == irs.NLB {
			tagList, err = tagHandler.GetNLBTags(resIID, key)
		} else if resType == irs.CLUSTER {
			tagList, err = tagHandler.GetClusterTags(resIID, key)
		}

		if err != nil {
			return irs.KeyValue{}, fmt.Errorf("failed to get tags: %w", err)
		}

		if len(tagList) == 0 {
			return irs.KeyValue{}, nil
		} else {
			return tagList[0], nil
		}
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

	if resType == irs.NLB {
		return tagHandler.RemoveNLBTag(resIID, key)
	} else if resType == irs.CLUSTER {
		return tagHandler.RemoveClusterTag(resIID, key)
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

func (tagHandler *AwsTagHandler) RemoveNLBTag(resIID irs.IID, tagKey string) (bool, error) {
	input := &elbv2.RemoveTagsInput{
		ResourceArns: []*string{aws.String(resIID.SystemId)},
		TagKeys: []*string{
			aws.String(tagKey),
		},
	}

	_, err := tagHandler.NLBClient.RemoveTags(input)
	if err != nil {
		return false, fmt.Errorf("failed to remove tag from NLB: %w", err)
	}

	return true, nil
}

func (tagHandler *AwsTagHandler) RemoveClusterTag(resIID irs.IID, tagKey string) (bool, error) {
	input := &eks.UntagResourceInput{
		ResourceArn: aws.String(resIID.SystemId),
		TagKeys: []*string{
			aws.String(tagKey),
		},
	}

	_, err := tagHandler.EKSClient.UntagResource(input)
	if err != nil {
		return false, fmt.Errorf("failed to remove tag from EKS cluster: %w", err)
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
		// All tags
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
		} else if resType == irs.CLUSTER {
			// Add a list of k8s Tags if this is a all search
			if keyword == "" || keyword == "*" {
				return tagHandler.GetAllClusterTags()
			} else {
				k8sTaginfos, _ := tagHandler.GetAllClusterTags()
				//spew.Dump(k8sTaginfos)
				return tagHandler.ExtractTagKeyValue(k8sTaginfos, keyword), nil
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
			//cblogger.Debug("=================================")
			//spew.Dump(result)
			//cblogger.Debug("=================================")
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
		//nlb
		nlbTaginfos, _ := tagHandler.GetAllNLBTags()
		if keyword != "" && keyword != "*" {
			nlbTaginfos = tagHandler.ExtractTagKeyValue(nlbTaginfos, keyword)
		}
		tagInfos = append(tagInfos, nlbTaginfos...)

		//cluster
		k8sTaginfos, _ := tagHandler.GetAllClusterTags()
		if keyword != "" && keyword != "*" {
			k8sTaginfos = tagHandler.ExtractTagKeyValue(k8sTaginfos, keyword)
		}
		tagInfos = append(tagInfos, k8sTaginfos...)
	}

	return tagInfos, nil
}
