package resources

import (
	"fmt"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	//cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

//https://intl.cloud.tencent.com/document/product/213/4940
//https://intl.cloud.tencent.com/document/product/213/33276
//https://console.intl.cloud.tencent.com/api/explorer?Product=cvm&Version=2017-03-12&Action=DescribeImages&SignVersion=

// 비슷한 function 으로 ecm에 있는 서비스
//https://console.intl.cloud.tencent.com/api/explorer?Product=ecm&Version=2019-07-19&Action=CreateImage&SignVersion=

const (
	TENCENT_IMAGE_STATE_CREATING = "CREATING"
	TENCENT_IMAGE_STATE_NORMAL   = "NORMAL"
	TENCENT_IMAGE_STATE_USING    = "USING"

	TENCENT_IMAGE_STATE_ERROR = "Error"

	RESOURCE_TYPE_MYIMAGE = "image"
	IMAGE_TAG_DEFAULT     = "Name"
	IMAGE_TAG_SOURCE_VM   = "CB-VMSNAPSHOT-SOURCEVM-ID"
)

type TencentMyImageHandler struct {
	Region idrv.RegionInfo
	Client *cvm.Client
}

func (myImageHandler TencentMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {

	request := cvm.NewCreateImageRequest()

	//ImageName        *string `json:"ImageName,omitempty" name:"ImageName"`
	//InstanceId       *string `json:"InstanceId,omitempty" name:"InstanceId"`
	//ImageDescription *string `json:"ImageDescription,omitempty" name:"ImageDescription"`
	//ForcePoweroff    *string `json:"ForcePoweroff,omitempty" name:"ForcePoweroff"

	request.ImageName = common.StringPtr(snapshotReqInfo.IId.NameId)
	request.InstanceId = common.StringPtr(snapshotReqInfo.SourceVM.SystemId)

	// DataDisk 가 있으면 해당 경로
	//request.DataDiskIds = common.StringPtrs([]string{ "datade" })

	// Tag 추가 ResourceType : instance(for CVM), host(for CDH), image(for image), keypair(for key)
	request.TagSpecification = []*cvm.TagSpecification{
		&cvm.TagSpecification{
			ResourceType: common.StringPtr("instance"),
			Tags: []*cvm.Tag{
				&cvm.Tag{
					Key:   common.StringPtr(IMAGE_TAG_SOURCE_VM),
					Value: common.StringPtr("aaa"),
				},
			},
		},
	}

	// The returned "resp" is an instance of the CreateImageResponse class which corresponds to the request object
	response, err := myImageHandler.Client.CreateImage(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		return irs.MyImageInfo{}, err
	}

	spew.Dump(response)
	//response.Response.RequestId
	return irs.MyImageInfo{}, nil
}

/*
*
TODO : CommonHandlerm에 DescribeImages, DescribeImageById, DescribeImageStatus 추가할 것.
*/
func (myImageHandler TencentMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	request := cvm.NewDescribeImagesRequest()

	request.ImageIds = common.StringPtrs([]string{"aaa"})
	request.Filters = []*cvm.Filter{
		&cvm.Filter{
			Name:   common.StringPtr("image-type"),
			Values: common.StringPtrs([]string{"PUBLIC_IMAGE"}),
		},
	}

	// The returned "resp" is an instance of the DescribeImagesResponse class which corresponds to the request object
	response, err := myImageHandler.Client.DescribeImages(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		fmt.Printf("An API error has returned: %s", err)
		return nil, err
	}

	myImageInfoList := []*irs.MyImageInfo{}
	for _, image := range response.Response.ImageSet {
		myImageInfo, err := convertImageSetToMyImageInfo(image)
		if err != nil {

			continue
		}
		myImageInfoList = append(myImageInfoList, &myImageInfo)
	}
	return myImageInfoList, nil
}

func (myImageHandler TencentMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	//myImageInfo, err := DescribeImageById()
	return irs.MyImageInfo{}, nil
}

/*
*
If the ImageState of an image is CREATING or USING, the image cannot be deleted. Call the DescribeImages API to query the image status.
Up to 10 custom images are allowed in each region. If you have run out of the quota, delete unused images to create new ones.
A shared image cannot be deleted.
*/
func (myImageHandler TencentMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {

	// Image 상태 조회

	// 삭제 처리
	request := cvm.NewDeleteImagesRequest()

	request.ImageIds = common.StringPtrs([]string{"aaa"})

	// The returned "resp" is an instance of the DeleteImagesResponse class which corresponds to the request object
	response, err := myImageHandler.Client.DeleteImages(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		fmt.Printf("An API error has returned: %s", err)
		return false, err
	}

	requestId := response.Response.RequestId
	cblogger.Info("requestId : %s", requestId)
	// image 조회 : 없어야 정상임.

	return true, nil
}

func convertImageSetToMyImageInfo(tencentImage *cvm.Image) (irs.MyImageInfo, error) {
	returnMymageInfo := irs.MyImageInfo{}
	return returnMymageInfo, nil
}
