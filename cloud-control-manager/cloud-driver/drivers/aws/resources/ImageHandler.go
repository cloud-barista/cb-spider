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
	"fmt"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
	"github.com/davecgh/go-spew/spew"
)

type AwsImageHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

func (imageHandler *AwsImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {

	return irs.ImageInfo{}, nil
}

func (imageHandler *AwsImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	cblogger.Debug("Start")
	var imageInfoList []*irs.ImageInfo

	/*
		input := &ec2.DescribeImagesInput{
			ImageIds: []*string{
				aws.String("ami-5731123e"),
			},
		}
	*/

	result, err := imageHandler.Client.DescribeImages(&ec2.DescribeImagesInput{})
	spew.Dump(result)

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

	cnt := 0
	for _, cur := range result.Images {
		cblogger.Infof("[%s] AMI 정보 조회", *cur.ImageId)
		//imageInfo := ExtractImageDescribeInfo(cur)
		//imageInfoList = append(imageInfoList, &imageInfo)

		cnt++
		if cnt > 20 {
			break
		}
	}

	spew.Dump(imageInfoList)

	/*
		type ImageInfo struct {
		     Id   string
		     Name string
		     GuestOS string // Windows7, Ubuntu etc.
		     Status string  // available, unavailable

		     keyValueList []KeyValue
		}
	*/

	return imageInfoList, nil
}

//Image 정보를 추출함
func ExtractImageDescribeInfo(image *ec2.Image) irs.ImageInfo {
	imageInfo := irs.ImageInfo{
		Id:     *image.ImageId,
		Name:   *image.Name,
		Status: *image.State,
	}

	//spew.Dump(reflect.ValueOf(image).Elem().NumField)
	target := reflect.ValueOf(image)
	elements := target.Elem()
	fmt.Printf("Type: %s\n", target.Type()) // 구조체 타입명

	for i := 0; i < elements.NumField(); i++ {

		mValue := elements.Field(i)
		mType := elements.Type().Field(i)
		spew.Dump(mValue)
		spew.Dump(mType)

		//spew.Dump(elements.Field(i).Kind())
		//fmt.Printf(elements.Field(i))
		//spew.Dump(elements.Field(i).Kind() == reflect.String)
		//fmt.Printf("")

		//v.Kind() == reflect.Int64

		//spew.Dump(reflect.TypeOf(elements.Field(i)).Kind())

		//spew.Dump(elements.Type().Field(i))
		//spew.Dump(elements.Field(i).Interface())

		/*
			mValue := elements.Field(i)
			mType := elements.Type().Field(i)
			tag := mType.Tag

				fmt.Printf("%10s %10s ==> %10v, json: %10s\n",
					mType.Name,         // 이름
					mType.Type,         // 타입
					mValue.Interface(), // 값
					tag.Get("json"))    // json 태그
		*/
	}
	//LoopObjectField(image)
	return imageInfo
}

func LoopObjectField(object interface{}) {
	e := reflect.ValueOf(object).Elem()
	fieldNum := e.NumField()
	for i := 0; i < fieldNum; i++ {
		v := e.Field(i)
		t := e.Type().Field(i)
		fmt.Printf("Name: %s / Type: %s / Value: %v / Tag: %s \n",
			t.Name, t.Type, v.Interface(), t.Tag.Get("custom"))
	}
}

func (imageHandler *AwsImageHandler) GetImage(imageID string) (irs.ImageInfo, error) {
	cblogger.Infof("imageID : ", imageID)

	input := &ec2.DescribeImagesInput{
		ImageIds: []*string{
			aws.String(imageID),
		},
	}

	result, err := imageHandler.Client.DescribeImages(input)
	spew.Dump(result)

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

	imageInfo := ExtractImageDescribeInfo(result.Images[0])

	return imageInfo, nil
}

func (imageHandler *AwsImageHandler) DeleteImage(imageID string) (bool, error) {
	return true, nil
}
