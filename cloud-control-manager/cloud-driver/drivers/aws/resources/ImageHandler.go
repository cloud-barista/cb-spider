// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by powerkim@etri.re.kr, 2019.06.

package resources

import (
	"errors"
	"reflect"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	//irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
)

type AwsImageHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

//@TODO : 작업해야 함.
func (imageHandler *AwsImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {

	return irs.ImageInfo{}, nil
}

//@TODO : 목록이 너무 많기 때문에 amazon 계정으로 공유된 퍼블릭 이미지중 AMI만 조회 함.
func (imageHandler *AwsImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	cblogger.Debug("Start")
	var imageInfoList []*irs.ImageInfo
	input := &ec2.DescribeImagesInput{
		//ImageIds: []*string{aws.String("ami-0d097db2fb6e0f05e")},
		Owners: []*string{
			aws.String("amazon"), //사용자 계정 번호를 넣으면 사용자의 이미지를 대상으로 조회 함.
		},
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("image-type"),
				Values: aws.StringSlice([]string{"machine"}),
			},
			{
				Name:   aws.String("is-public"),
				Values: aws.StringSlice([]string{"true"}),
			},
		},
	}
	result, err := imageHandler.Client.DescribeImages(input)
	//spew.Dump(result)	//출력 정보가 너무 많아서 생략

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return nil, err
	}

	//cnt := 0
	for _, cur := range result.Images {
		cblogger.Infof("[%s] AMI 정보 처리", *cur.ImageId)
		imageInfo := ExtractImageDescribeInfo(cur)
		imageInfoList = append(imageInfoList, &imageInfo)
		/*
			cnt++
			if cnt > 20 {
				break
			}
		*/
	}

	//spew.Dump(imageInfoList)

	return imageInfoList, nil
}

//Image 정보를 추출함
func ExtractImageDescribeInfo(image *ec2.Image) irs.ImageInfo {
	//spew.Dump(image)
	imageInfo := irs.ImageInfo{
		Id:     *image.ImageId,
		Name:   *image.Name,
		Status: *image.State,
	}

	keyValueList := []irs.KeyValue{
		{Key: "CreationDate", Value: *image.CreationDate},
		{Key: "Architecture", Value: *image.Architecture},
		{Key: "OwnerId", Value: *image.OwnerId},
		{Key: "ImageType", Value: *image.ImageType},
		{Key: "ImageLocation", Value: *image.ImageLocation},
		{Key: "VirtualizationType", Value: *image.VirtualizationType},
		{Key: "Public", Value: strconv.FormatBool(*image.Public)},
	}

	// 일부 이미지들은 아래 정보가 없어서 예외 처리 함.
	if !reflect.ValueOf(image.Description).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "Description", Value: *image.Description})
	}
	if !reflect.ValueOf(image.ImageOwnerAlias).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "ImageOwnerAlias", Value: *image.ImageOwnerAlias})
	}
	if !reflect.ValueOf(image.RootDeviceName).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "RootDeviceName", Value: *image.RootDeviceName})
		keyValueList = append(keyValueList, irs.KeyValue{Key: "RootDeviceType", Value: *image.RootDeviceType})
	}
	if !reflect.ValueOf(image.EnaSupport).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "EnaSupport", Value: strconv.FormatBool(*image.EnaSupport)})
	}

	imageInfo.KeyValueList = keyValueList

	return imageInfo
}

func (imageHandler *AwsImageHandler) GetImage(imageID string) (irs.ImageInfo, error) {
	cblogger.Infof("imageID : [%s]", imageID)

	input := &ec2.DescribeImagesInput{
		ImageIds: []*string{
			aws.String(imageID),
		},
	}

	result, err := imageHandler.Client.DescribeImages(input)
	//spew.Dump(result)
	cblogger.Info(result)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return irs.ImageInfo{}, err
	}

	if len(result.Images) > 0 {
		imageInfo := ExtractImageDescribeInfo(result.Images[0])
		return imageInfo, nil
	} else {
		return irs.ImageInfo{}, errors.New("조회된 Image 정보가 없습니다.")
	}

}

//@TODO : 삭제 API 찾아야 함.
func (imageHandler *AwsImageHandler) DeleteImage(imageID string) (bool, error) {
	return false, nil
}
