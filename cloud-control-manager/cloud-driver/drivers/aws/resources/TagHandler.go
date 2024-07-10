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

	"github.com/aws/aws-sdk-go/aws"
	//"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsTagHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

func (tagHandler *AwsTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	cblogger.Debugf("Req resTyp:[%s] / resIID:[%s] / Tag Key:[%s] / Tag Value:[%s]", resType, resIID, tag.Key, tag.Value)

	if resIID.SystemId == "" {
		msg := "tag will not be add because resIID.SystemId is not provided"
		cblogger.Error(msg)
		return irs.KeyValue{}, errors.New(msg)
	}

	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, resIID.SystemId, "DescribeTags()")
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
		cblogger.Errorf("Failed to set Name Tag for [%s] VM", resIID.NameId)
		cblogger.Error(errtag)
		return irs.KeyValue{}, errtag
	}
	LoggingInfo(hiscallInfo, start)

	if cblogger.Level.String() == "debug" {
		cblogger.Info(result)
	}

	return tag, nil
}

func (tagHandler *AwsTagHandler) ListTag(resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
	cblogger.Debugf("Req resTyp:[%s] / resIID:[%s]", resType, resIID)

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
		cblogger.Errorf("Failed to set Name Tag for [%s] VM", resIID.NameId)
		cblogger.Error(errtag)
		LoggingError(hiscallInfo, errtag)
		return nil, errtag
	}
	LoggingInfo(hiscallInfo, start)

	if cblogger.Level.String() == "debug" {
		cblogger.Info(result)
	}

	return nil, nil
}

// describetags
func (tagHandler *AwsTagHandler) GetTag(resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	cblogger.Info("resTyp : ", resType)
	cblogger.Info("resIID : ", resIID)
	cblogger.Info("key : ", key)

	return irs.KeyValue{}, nil
}

// deletetags
func (tagHandler *AwsTagHandler) RemoveTag(resType irs.RSType, resIID irs.IID, key string) (bool, error) {
	cblogger.Info("resTyp : ", resType)
	cblogger.Info("resIID : ", resIID)
	cblogger.Info("key : ", key)

	return false, nil
}

// Find tags by tag key or value
// resType: ALL | VPC, SUBNET, etc.,.
// keyword: The keyword to search for in the tag key or value.
// if you want to find all tags, set keyword to "" or "*".
func (tagHandler *AwsTagHandler) FindTag(resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	cblogger.Info("resTyp : ", resType)
	cblogger.Info("keyword : ", keyword)
	return nil, nil
}
