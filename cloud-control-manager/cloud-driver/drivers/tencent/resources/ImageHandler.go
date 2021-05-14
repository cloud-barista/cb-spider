// Cloud Driver Interface of CB-Spider.  // The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2019.06.

package resources

import (
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	//irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"

	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type TencentImageHandler struct {
	Region idrv.RegionInfo
	Client *cvm.Client
}

func (imageHandler *TencentImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	return irs.ImageInfo{imageReqInfo.IId, "", "", nil}, nil
}

func (imageHandler *TencentImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	cblogger.Debug("Start")
	return nil, nil
}
func (imageHandler *TencentImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {

	cblogger.Infof("imageID : [%s]", imageIID.SystemId)
	return irs.ImageInfo{}, nil
}

func (imageHandler *TencentImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	return false, nil
}

/*
//@TODO : 작업해야 함.
func (imageHandler *TencentImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: imageReqInfo.IId.NameId,
		CloudOSAPI:   "-",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	imageReqInfo.IId.SystemId = imageReqInfo.IId.NameId

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))

	return irs.ImageInfo{imageReqInfo.IId, "", "", nil}, nil
}

//@TODO : 목록이 너무 많기 때문에 amazon 계정으로 공유된 퍼블릭 이미지중 AMI만 조회 함.
func (imageHandler *TencentImageHandler) ListImage() ([]*irs.ImageInfo, error) {
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

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: "ListImage()",
		CloudOSAPI:   "DescribeImages",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := imageHandler.Client.DescribeImages(input)
	//spew.Dump(result)	//출력 정보가 너무 많아서 생략

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

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
	callogger.Info(call.String(callLogInfo))

	//cnt := 0
	for _, cur := range result.Images {
		cblogger.Infof("[%s] AMI 정보 처리", *cur.ImageId)
		imageInfo := ExtractImageDescribeInfo(cur)
		imageInfoList = append(imageInfoList, &imageInfo)
	}

	//spew.Dump(imageInfoList)

	return imageInfoList, nil
}

//Image 정보를 추출함
//@TODO : GuestOS 쳌크할 것
func ExtractImageDescribeInfo(image *ec2.Image) irs.ImageInfo {
	//spew.Dump(image)
	imageInfo := irs.ImageInfo{
		//IId: irs.IID{*image.Name, *image.ImageId},
		IId: irs.IID{*image.ImageId, *image.ImageId},
		//Id:     *image.ImageId,
		//Name:   *image.Name,
		Status: *image.State,
	}

	keyValueList := []irs.KeyValue{
		//{Key: "Name", Value: *image.Name}, //20200723-Name이 없는 이미지 존재 - 예)ami-0008a301
		{Key: "CreationDate", Value: *image.CreationDate},
		{Key: "Architecture", Value: *image.Architecture},
		{Key: "OwnerId", Value: *image.OwnerId},
		{Key: "ImageType", Value: *image.ImageType},
		{Key: "ImageLocation", Value: *image.ImageLocation},
		{Key: "VirtualizationType", Value: *image.VirtualizationType},
		{Key: "Public", Value: strconv.FormatBool(*image.Public)},
	}

	// 일부 이미지들은 아래 정보가 없어서 예외 처리 함.
	if !reflect.ValueOf(image.Name).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "Name", Value: *image.Name})
	}
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

//func (imageHandler *TencentImageHandler) GetImage(imageID string) (irs.ImageInfo, error) {
func (imageHandler *TencentImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {

	cblogger.Infof("imageID : [%s]", imageIID.SystemId)

	input := &ec2.DescribeImagesInput{
		ImageIds: []*string{
			aws.String(imageIID.SystemId),
		},
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: imageIID.SystemId,
		CloudOSAPI:   "DescribeImages",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := imageHandler.Client.DescribeImages(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//spew.Dump(result)
	cblogger.Info(result)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

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
	callogger.Info(call.String(callLogInfo))

	if len(result.Images) > 0 {
		imageInfo := ExtractImageDescribeInfo(result.Images[0])
		return imageInfo, nil
	} else {
		return irs.ImageInfo{}, errors.New("조회된 Image 정보가 없습니다.")
	}

}

//@TODO : 삭제 API 찾아야 함.
func (imageHandler *TencentImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: imageIID.SystemId,
		CloudOSAPI:   "-",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))

	return false, nil
}
*/
