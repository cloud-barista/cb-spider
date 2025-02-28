// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// program by ysjeon@mz.co.kr, 2019.07.
// modify by devunet@mz.co.kr, 2019.11.

package resources

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	compute "google.golang.org/api/compute/v1"
)

type GCPImageHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

var arrImageProjectList = []string{
	"gce-uefi-images", // 보안 VM을 지원하는 이미지

	//보안 VM을 지원하지 않는 이미지들
	"centos-cloud",
	"cos-cloud",
	"coreos-cloud",
	"debian-cloud",
	"rhel-cloud",
	"rhel-sap-cloud",
	"suse-cloud",
	"suse-sap-cloud",
	"ubuntu-os-cloud",
	"windows-cloud",
	"windows-sql-cloud",
}

/*
이미지를 생성할 때 GCP 같은 경우는 내가 생성한 이미지에서만 리스트를 가져 올 수 있다.
퍼블릭 이미지를 가져 올 수 없다.
가져올라면 다르게 해야 함.
Insert할때 필수 값
name, sourceDisk(sourceImage),storageLocations(배열 ex : ["asia"])
이미지를 어떻게 생성하는냐에 따라서 키 값이 변경됨
디스크, 스냅샷,이미지, 가상디스크, Cloud storage
1) Disk일 경우 :
	{"sourceDisk": "projects/mcloud-barista-251102/zones/asia-northeast1-b/disks/my-root-pd",}
2) Image일 경우 :
	{"sourceImage": "projects/mcloud-barista-251102/global/images/image-1",}



*/

func (imageHandler *GCPImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	return irs.ImageInfo{}, errors.New("Feature not implemented.")
}

/*
//리스트의 경우 Name 기반으로 조회해서 처리하기에는 너무 느리기 때문에 직접 컨버팅함.
func (imageHandler *GCPImageHandler) ListImage() ([]*irs.ImageInfo, error) {

	//projectId := imageHandler.Credential.ProjectID
	projectId := "gce-uefi-images"

	// list, err := imageHandler.Client.Images.List(projectId).Do()
	list, err := imageHandler.Client.Images.List(projectId).Do()
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	var imageList []*irs.ImageInfo
	for _, item := range list.Items {
		info := mappingImageInfo(item)
		imageList = append(imageList, &info)
	}

	//cblogger.Debug(imageList)
	return imageList, nil
}
*/

// 리스트의 경우 Name 기반으로 조회해서 처리하기에는 너무 느리기 때문에 직접 컨버팅함.
// filter := "NOT deprecated:*" 적용
// 아래와 같이 deprecated가 있는 image는 다른 image로 대체된 것임
//
//	      "deprecated": {
//				"state": "DEPRECATED",
//				"replacement": "https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/ubuntu-2204-jammy-v20241218"
//			 },
func (imageHandler *GCPImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	cblogger.Debug("Retrieve All VM Images")

	//https://cloud.google.com/compute/docs/images?hl=ko

	var imageList []*irs.ImageInfo

	cnt := 0
	nextPageToken := ""
	var req *compute.ImagesListCall
	var res *compute.ImageList
	var err error
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: "",
		CloudOSAPI:   "List()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	for _, projectId := range arrImageProjectList {
		cblogger.Infof("Processing image list owned by [%s] project", projectId)

		// filter := "NOT deprecated:*" // deprecated가 있는 항목은 다른 image로 대체된 것임
		// req = imageHandler.Client.Images.List(projectId).Filter(filter)
		//첫번째 호출
		req = imageHandler.Client.Images.List(projectId)
		res, err = req.Do()
		if err != nil {
			callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
			callLogInfo.ErrorMSG = err.Error()
			callogger.Info(call.String(callLogInfo))
			cblogger.Errorf("Failed to retrieve image list owned by [%s] project!", projectId)
			cblogger.Error(err)
			return nil, err
		}

		nextPageToken = res.NextPageToken
		cblogger.Debug("NextPageToken : ", nextPageToken)

		//현재 페이지부터 마지막 페이지까지 조회
		for {
			for _, item := range res.Items {
				cnt++
				if cblogger.Level.String() == "debug" {
					cblogger.Debug(item)
				}
				info := mappingImageInfo(item)
				imageList = append(imageList, &info)
			} // for : 페이지 데이터 추출

			//다음 페이지가 존재하면 호출
			if nextPageToken != "" {
				res, err = req.PageToken(nextPageToken).Do()
				nextPageToken = res.NextPageToken
				cblogger.Debug("NextPageToken : ", nextPageToken)
			} else {
				break
			}
		} // for : 멀티 페이지 처리
	}
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))

	return imageList, nil
}

// Name 기반으로 VM생성에 필요한 URL및 Image API 호출과 CB 리턴 정보 조회용
type GcpImageInfo struct {
	ImageUrl string //for CB(VM Start)
	Name     string //for CB
	GuestOS  string //for CB (Item.Family)
	Status   string //for CB

	ProjectId string //for image api call
	//Id        uint64 //for image api call
	Id string

	SourceType  string //for keyValue
	SourceImage string //for keyValue
	SelfLink    string //for keyValue
	Family      string //for keyValue
}

// GCP 호출을 줄이기 위해 조회된 정보를 CB형태로 직접 변환해서 전달 함.
func (imageHandler *GCPImageHandler) ConvertGcpImageInfoToCbImageInfo(imageInfo GcpImageInfo) irs.ImageInfo {
	cblogger.Debug(imageInfo)
	if cblogger.Level.String() == "debug" {
		cblogger.Debug(imageInfo)
	}

	cbImageInfo := irs.ImageInfo{
		IId: irs.IID{
			NameId:   imageInfo.Name,
			SystemId: imageInfo.Name,
		},

		KeyValueList: []irs.KeyValue{
			{"Name", imageInfo.Name},
			//{"Id", strconv.FormatUint(imageInfo.Id, 10)},
			{"Id", imageInfo.Id},
			{"ImageUrl", imageInfo.ImageUrl},
			{"SourceImage", imageInfo.SourceImage}, // VM생성 시에는 SourceImage나 SelfLink 값을 이용해야 함.
			{"SourceType", imageInfo.SourceType},
			{"SelfLink", imageInfo.SelfLink},
			{"Family", imageInfo.Family},
			{"ProjectId", imageInfo.ProjectId},
		},
	}

	return cbImageInfo
}

// 이슈 #239에 의해 Name 기반에서 URL 기반으로 로직 변경
// 전달 받은 URL에서 projectId와 Name을 추출해서 조회함.
func (imageHandler *GCPImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	cblogger.Debug(imageIID)

	//"https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20200415"
	//projectId := imageHandler.Credential.ProjectID
	projectId := ""
	imageName := ""

	arrLink := strings.Split(imageIID.SystemId, "/")
	if len(arrLink) > 0 {
		imageName = arrLink[len(arrLink)-1]
		for pos, item := range arrLink {
			if strings.EqualFold(item, "projects") {
				projectId = arrLink[pos+1]
				break
			}
		}
	}
	cblogger.Infof("projectId : [%s] / imageName : [%s]", projectId, imageName)
	if projectId == "" {
		return irs.ImageInfo{}, errors.New("ProjectId information not found in URL.")
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: imageIID.SystemId,
		CloudOSAPI:   "Images.Get()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	image, err := imageHandler.Client.Images.Get(projectId, imageName).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.ImageInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))
	imageInfo := mappingImageInfo(image)
	return imageInfo, nil
}

// getImage by name
//
// ubuntu-1204-precise-v20141031 // OBSOLETE  -> rejected, error
// ubuntu-1804-bionic-v20220505 //DEPRECATED -> create but worning
// ubuntu-2204-jammy-v20241218 // ACTIVE
func (imageHandler *GCPImageHandler) GetImageN(imageName string) (irs.ImageInfo, error) {
	projectId := ""
	imageInfo := irs.ImageInfo{}

	cblogger.SetLevel(logrus.InfoLevel)

	targetOs := strings.Split(imageName, "-")[0]
	cblogger.Debug("targetOs = ", targetOs)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: imageName,
		CloudOSAPI:   "Images.Get()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	// ubuntu, window ....
	for _, imgageProjectId := range arrImageProjectList {
		if strings.Contains(imgageProjectId, targetOs) {
			projectId = imgageProjectId
			break
		}
	}
	cblogger.Debug("projectId = ", projectId)
	// if found
	if projectId != "" {
		gcpImage, err := imageHandler.Client.Images.Get(projectId, imageName).Do()
		callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
		if err != nil {
			callLogInfo.ErrorMSG = err.Error()
			callogger.Info(call.String(callLogInfo))
			cblogger.Info("try to get image by projectId ", projectId)
		}

		if gcpImage != nil {
			imageInfo = mappingImageInfo(gcpImage)
			return imageInfo, nil
		}
	}

	cblogger.Info("image not found at specific project url. try all project.")
	// if not. query all
	for _, imgageProjectId := range arrImageProjectList {
		gcpImage, err := imageHandler.Client.Images.Get(imgageProjectId, imageName).Do()
		callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
		if err != nil {
			callLogInfo.ErrorMSG = err.Error()
			callogger.Info(call.String(callLogInfo))
			cblogger.Info("try to get image by all projectId ", projectId)
			continue
		}
		imageInfo = mappingImageInfo(gcpImage)
		return imageInfo, nil
	}

	// not found Image
	return imageInfo, errors.New("Image not found." + imageName)

}

// 이슈 #239에 의해 Name 기반에서 URL 기반으로 로직 변경
// 전체 목록에서 이미지 정보를 조회 함. - 위의 GetImage()로 검색되지 않는 경우가 발생하면 이 함수를 이용할 것.
func (imageHandler *GCPImageHandler) GetImageByUrl(imageIID irs.IID) (irs.ImageInfo, error) {
	cblogger.Debug(imageIID)

	//이미지 명을 기반으로 이미지 정보를 조회함.
	gcpImageInfo, err := imageHandler.FindImageInfo(imageIID.SystemId)
	//return irs.ImageInfo{IId: irs.IID{SystemId: gcpImageInfo.Url}}, err
	if err != nil {
		cblogger.Error(err)
		return irs.ImageInfo{}, err
	}
	cblogger.Debug(gcpImageInfo)
	//return irs.ImageInfo{}, nil
	return imageHandler.ConvertGcpImageInfoToCbImageInfo(gcpImageInfo), nil

	/*
		//projectId := imageHandler.Credential.ProjectID
		projectId := "gce-uefi-images"

		image, err := imageHandler.Client.Images.Get(projectId, imageIID.SystemId).Do()
		if err != nil {
			cblogger.Error(err)
			return irs.ImageInfo{}, err
		}
		imageInfo := mappingImageInfo(image)
		return imageInfo, nil
	*/
}

// public Image 는 지울 수 없는데 어떻게 해야 하는가?
func (imageHandler *GCPImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {

	//이미지 명을 기반으로 이미지 정보를 조회함.
	gcpImageInfo, err := imageHandler.FindImageInfo(imageIID.SystemId)
	//return irs.ImageInfo{IId: irs.IID{SystemId: gcpImageInfo.Url}}, err
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	//projectId := imageHandler.Credential.ProjectID
	projectId := gcpImageInfo.ProjectId
	imageId := gcpImageInfo.Id

	//res, err := imageHandler.Client.Images.Delete(projectId, imageIID.SystemId).Do()
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: imageIID.SystemId,
		CloudOSAPI:   "CreateVpc()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	res, err := imageHandler.Client.Images.Delete(projectId, imageId).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return false, err
	}
	callogger.Info(call.String(callLogInfo))
	cblogger.Debug(res)
	return true, err
}

// 이슈 #239에 의해 Name 기반에서 URL 기반으로 로직 변경
// 사용의 편의를 위해 이미지 URL을 전달 받아서 이미지 정보를 리턴 함.
// https://cloud.google.com/compute/docs/images?hl=ko
// @TODO : 효율을 위해서 최소한 ProjectId 정보를 입력 받아야 하지만 현재는 이미지 URL만 전달 받기 때문에 하나로 통합해 놓음.
func (imageHandler *GCPImageHandler) FindImageInfo(reqImageName string) (GcpImageInfo, error) {
	cblogger.Infof("Finding information for [%s] image", reqImageName)

	//https://cloud.google.com/compute/docs/images?hl=ko
	arrImageProjectList := []string{
		//"ubuntu-os-cloud",

		"gce-uefi-images", // 보안 VM을 지원하는 이미지

		//보안 VM을 지원하지 않는 이미지들
		"centos-cloud",
		"cos-cloud",
		"coreos-cloud",
		"debian-cloud",
		"rhel-cloud",
		"rhel-sap-cloud",
		"suse-cloud",
		"suse-sap-cloud",
		"ubuntu-os-cloud",
		"windows-cloud",
		"windows-sql-cloud",
	}

	cnt := 0
	//curImageLink := ""
	imageInfo := GcpImageInfo{}
	nextPageToken := ""
	var req *compute.ImagesListCall
	var res *compute.ImageList
	var err error
	for _, projectId := range arrImageProjectList {
		cblogger.Infof("Processing retrieval of image list owned by [%s] project", projectId)

		//첫번째 호출
		req = imageHandler.Client.Images.List(projectId)
		//req.Filter("name=" + reqImageName)
		//req.Filter("SelfLink=" + reqImageName)

		res, err = req.Do()
		if err != nil {
			cblogger.Errorf("Failed to retrieve image list owned by [%s] project!", projectId)
			cblogger.Error(err)
			return GcpImageInfo{}, err
		}

		nextPageToken = res.NextPageToken
		cblogger.Info("NestPageToken : ", nextPageToken)

		//현재 페이지부터 마지막 페이지까지 조회
		for {
			/*
				//list, err := imageHandler.Client.Images.List(projectId).Do() // 1000 // 500
				req := imageHandler.Client.Images.List(projectId)
				ret, err := req.Do()
				cblogger.Info("First -------------> ", ret.NextPageToken)
				list, err := req.PageToken(ret.NextPageToken).Do()
				cblogger.Info("Second -------------> ", list.NextPageToken)
			*/

			//데이터 찾기
			for _, item := range res.Items {
				cnt++

				//curImageLink = imageInfo.SourceImage //보통은 SelfLink에 정보가 있는데 혹시 몰라서 SourceImage 정보와 함께 비교 함. // SourceImage는 Name과 동일할 때가 있음.
				//cblogger.Debugf(" SourceImage : [%s]", curImageLink)

				//SourceImage 정보가 없으면 SelfLink 정보를 이용함.
				//SelfLink: [Output Only] Server-defined URL for the resource.
				//if curImageLink == "" {

				//2020-07-24 Name 기반에서 URL기반으로 바뀌었기 때문에 굳이 Split할 필요는 없음
				/*
					arrLink := strings.Split(item.SelfLink, "/")
					if len(arrLink) > 0 {
						curImageLink = arrLink[len(arrLink)-1]
					}
					cblogger.Debugf("  [%d] : [%s] : [%s]", item.Id, item.SelfLink, curImageLink)
				*/
				//cblogger.Debug("")
				//}

				//2020-07-24 Name 기반에서 URL기반으로 바뀌었기 때문에 직접 SelfLink만 체크 함.
				if strings.EqualFold(reqImageName, item.SelfLink) {
					//if strings.EqualFold(reqImageName, item.Name) || strings.EqualFold(reqImageName, curImageLink) {

					if item.SelfLink == "" {
						cblogger.Errorf("Found information for the requested [%s] image, but Image URL [SelfLink] information is missing.", reqImageName)
						return GcpImageInfo{}, errors.New("Not Found : [" + reqImageName + "] Image information does not contain URL information.")
					}
					//imageInfo.Id = item.Id
					imageInfo.Id = strconv.FormatUint(item.Id, 10)
					imageInfo.ImageUrl = item.SelfLink //item.SourceImage에 URL이 아닌 item.Name이 나와서 SelfLink만 이용함.

					imageInfo.GuestOS = item.Family
					imageInfo.Status = item.Status

					//imageInfo.Name = item.Name
					imageInfo.Name = item.SelfLink //2020-07-24 Name에서 URL로 변경됨. 이슈 #239
					imageInfo.SourceImage = item.SourceImage
					imageInfo.SourceType = item.SourceType
					imageInfo.SelfLink = item.SelfLink
					imageInfo.Family = item.Family
					imageInfo.ProjectId = projectId

					//cblogger.Info("최종 이미지 정보")
					//if cblogger.Level.String() == "debug" {
					//	cblogger.Debug(imageInfo)
					//}
					return imageInfo, nil
				}
			} // for : 조회 결과에서 일치하는 데이터 찾기

			//다음 페이지가 존재하면 호출
			if nextPageToken != "" {
				res, err = req.PageToken(nextPageToken).Do()
				nextPageToken = res.NextPageToken
				cblogger.Info("NestPageToken : ", nextPageToken)
			} else {
				break
			}
		} // for : 멀티 페이지 처리
	}

	cblogger.Errorf("Could not find information for the requested [%s] image. - Total image check count: [%d]", reqImageName, cnt)
	return GcpImageInfo{}, errors.New("Not Found : [" + reqImageName + "] Image information not found")
}

// 목록에서 이미지 Name으로 정보를 찾아서 리턴 함. - 2020-07-24 URL기반으로 변경되어서 이 메소드는 사용 안 함.
// @TODO : 효율을 위해서 최소한 ProjectId 정보를 입력 받아야 하지만 현재는 이미지 명만 전달 받기 때문에 하나로 통합해 놓음.
func (imageHandler *GCPImageHandler) FindImageInfoByName(reqImageName string) (GcpImageInfo, error) {
	cblogger.Infof("Finding information for [%s] image", reqImageName)

	//https://cloud.google.com/compute/docs/images?hl=ko
	arrImageProjectList := []string{
		//"ubuntu-os-cloud",

		"gce-uefi-images", // 보안 VM을 지원하는 이미지

		//보안 VM을 지원하지 않는 이미지들
		"centos-cloud",
		"cos-cloud",
		"coreos-cloud",
		"debian-cloud",
		"rhel-cloud",
		"rhel-sap-cloud",
		"suse-cloud",
		"suse-sap-cloud",
		"ubuntu-os-cloud",
		"windows-cloud",
		"windows-sql-cloud",
	}

	cnt := 0
	curImageLink := ""
	imageInfo := GcpImageInfo{}
	nextPageToken := ""
	var req *compute.ImagesListCall
	var res *compute.ImageList
	var err error
	for _, projectId := range arrImageProjectList {
		cblogger.Infof("[%s] Project Ownership Image List Retrieval Process", projectId)

		//첫번째 호출
		req = imageHandler.Client.Images.List(projectId)
		req.Filter("name=" + reqImageName)

		res, err = req.Do()
		if err != nil {
			cblogger.Errorf("[%s] Failed to Retrieve Image List Owned by the Project!", projectId)
			cblogger.Error(err)
			return GcpImageInfo{}, err
		}

		nextPageToken = res.NextPageToken
		cblogger.Info("NestPageToken : ", nextPageToken)

		//현재 페이지부터 마지막 페이지까지 조회
		for {
			/*
				//list, err := imageHandler.Client.Images.List(projectId).Do() // 1000 // 500
				req := imageHandler.Client.Images.List(projectId)
				ret, err := req.Do()
				cblogger.Info("First -------------> ", ret.NextPageToken)
				list, err := req.PageToken(ret.NextPageToken).Do()
				cblogger.Info("Second -------------> ", list.NextPageToken)
			*/

			//데이터 찾기
			for _, item := range res.Items {
				cnt++

				//curImageLink = imageInfo.SourceImage //보통은 SelfLink에 정보가 있는데 혹시 몰라서 SourceImage 정보와 함께 비교 함. // SourceImage는 Name과 동일할 때가 있음.
				cblogger.Debugf(" SourceImage : [%s]", curImageLink)

				//SourceImage 정보가 없으면 SelfLink 정보를 이용함.
				//SelfLink: [Output Only] Server-defined URL for the resource.
				//if curImageLink == "" {

				arrLink := strings.Split(item.SelfLink, "/")
				if len(arrLink) > 0 {
					curImageLink = arrLink[len(arrLink)-1]
				}
				cblogger.Debugf("  [%d] : [%s] : [%s]", item.Id, item.SelfLink, curImageLink)
				cblogger.Debug("")
				//}

				if strings.EqualFold(reqImageName, item.Name) || strings.EqualFold(reqImageName, curImageLink) {
					//cblogger.Debug("=====************** 찾았다!!! *********======")
					//cblogger.Infof("=====************** [%d]번째에 찾았다!!! *********======", cnt)
					if item.SelfLink == "" {
						cblogger.Errorf("We found information for the requested [%s] image, but there is no Image URL [SelfLink] available.", reqImageName)
						return GcpImageInfo{}, errors.New("Not Found : [" + reqImageName + "] Image information does not contain URL information.")
					}
					//imageInfo.Id = item.Id
					imageInfo.Id = strconv.FormatUint(item.Id, 10)
					imageInfo.ImageUrl = item.SelfLink //item.SourceImage에 URL이 아닌 item.Name이 나와서 SelfLink만 이용함.

					imageInfo.GuestOS = item.Family
					imageInfo.Status = item.Status

					imageInfo.Name = item.Name
					imageInfo.SourceImage = item.SourceImage
					imageInfo.SourceType = item.SourceType
					imageInfo.SelfLink = item.SelfLink
					imageInfo.Family = item.Family
					imageInfo.ProjectId = projectId

					if cblogger.Level.String() == "debug" {
						cblogger.Debug(imageInfo)
					}
					return imageInfo, nil
				}
			} // for : 조회 결과에서 일치하는 데이터 찾기

			//다음 페이지가 존재하면 호출
			if nextPageToken != "" {
				res, err = req.PageToken(nextPageToken).Do()
				nextPageToken = res.NextPageToken
				cblogger.Info("NestPageToken : ", nextPageToken)
			} else {
				break
			}
		} // for : 멀티 페이지 처리
	}

	cblogger.Errorf("We couldn't find information for the requested [%s] image. - Total Image Check Count: [%d]", reqImageName, cnt)
	return GcpImageInfo{}, errors.New("Not Found : [" + reqImageName + "] Image information not found")
}

/*
//향후 필요하면 프로젝트Id를 입력 받도로 구현
//이미지 설명 : https://cloud.google.com/compute/docs/images?hl=ko
func (imageHandler *GCPImageHandler) FindImageInfo(projectId string, reqImageName string) (GcpImageInfo, error) {
	cblogger.Infof("[%s] 프로젝트에서 [%s] 이미지 정보 찾기 ", projectId, reqImageName)

	list, err := imageHandler.Client.Images.List(projectId).Do()
	if err != nil {
		cblogger.Error(err)
		return GcpImageInfo{}, err
	}

	imageInfo := GcpImageInfo{}
	curImageLink := ""
	for _, item := range list.Items {
		curImageLink = ""
		arrLink := strings.Split(item.SelfLink, "/")
		if len(arrLink) > 0 {
			curImageLink = arrLink[len(arrLink)-1]
		}
		cblogger.Infof("  [%s] : [%s] : [%s]", item.Id, item.SelfLink, curImageLink)

		if strings.EqualFold(reqImageName, item.Name) || strings.EqualFold(reqImageName, curImageLink) {
			imageInfo.Id = item.Id
			imageInfo.Url = curImageLink
			return imageInfo, nil
		}
	}

	cblogger.Errorf("요청 받은 [%s] 이미지에 대한 정보를 찾지 못 했습니다.", reqImageName)
	return GcpImageInfo{}, errors.New("Not Found : [" + reqImageName + "] Image information not found")
	//return GcpImageInfo{},nil
}
*/

func mappingImageInfo(imageInfo *compute.Image) irs.ImageInfo {
	//lArr := strings.Split(imageInfo.Licenses[0], "/")
	//os := lArr[len(lArr)-1]

	//cblogger.Info("===================================")

	//"description": "Canonical, Ubuntu, 12.04 LTS,	amd64 precise image built on 2016-03-02",
	//"description": "Microsoft, Windows Server, version 1709 Core for Containers (Beta), Server Core, x64 built on 2017-12-12",
	//"description": "Canonical, Ubuntu, 18.04 LTS, amd64 bionic image built on 2022-05-05, supports Shielded VM features",

	distribution := extractOsDistribution(imageInfo)
	imageStatus := extractImageAvailability(imageInfo)

	if imageInfo.Deprecated != nil {
		gcpImageState := imageInfo.Deprecated.State
		distribution = distribution + ". ImageState : " + gcpImageState
		if imageInfo.Deprecated.Replacement != "" {
			gcpImageReplacement := imageInfo.Deprecated.Replacement
			distribution = distribution + ", Replacement : " + gcpImageReplacement
		}
	}

	// 2024-12-23 ImageInfo changed for meta. IID, GuestOS, Status deprecated.
	// 2025-01-18: Postpone the deprecation of IID, so revoke IID changes.
	imageInfo.ShieldedInstanceInitialState = nil // 너무 길어 임시로 주석처리함.
	returnImageInfo := irs.ImageInfo{
		IId: irs.IID{
			NameId:   imageInfo.SelfLink,
			SystemId: imageInfo.SelfLink,
		},
		// GuestOS: imageInfo.Family,
		// Status:  imageInfo.Status,

		Name:           imageInfo.Name,
		OSArchitecture: extractOsArchitecture(imageInfo),
		OSPlatform:     extractOsPlatform(imageInfo), // imageInfo.Description
		OSDistribution: distribution,
		OSDiskType:     "NA",
		OSDiskSizeGB:   strconv.FormatInt(imageInfo.DiskSizeGb, 10),
		ImageStatus:    imageStatus,
		KeyValueList:   irs.StructToKeyValueList(imageInfo),
	}

	return returnImageInfo
}

// OS Platform (contains description or family, name)
// "name": "ubuntu-1804-bionic-arm64-v20220712",
func extractOsPlatform(orgImage *compute.Image) irs.OSPlatform {
	platformInfo := orgImage.Name

	osPlatform := irs.PlatformNA

	lowerCasePlatformInfo := strings.ToLower(platformInfo)

	if strings.Contains(lowerCasePlatformInfo, "windows") {
		osPlatform = irs.Windows
	} else if strings.Contains(lowerCasePlatformInfo, "ubuntu") ||
		strings.Contains(lowerCasePlatformInfo, "linux") ||
		strings.Contains(lowerCasePlatformInfo, "centos") ||
		strings.Contains(lowerCasePlatformInfo, "debian") ||
		strings.Contains(lowerCasePlatformInfo, "fedora") ||
		strings.Contains(lowerCasePlatformInfo, "rhel") ||
		strings.Contains(lowerCasePlatformInfo, "rocky") ||
		strings.Contains(lowerCasePlatformInfo, "unix") {
		osPlatform = irs.Linux_UNIX
	}

	// "centos-cloud",
	// "cos-cloud",
	// "coreos-cloud",
	// "debian-cloud",
	// "rhel-cloud",
	// "rhel-sap-cloud",
	// "suse-cloud",
	// "suse-sap-cloud",
	// "ubuntu-os-cloud",
	// "windows-cloud",
	// "windows-sql-cloud",
	return osPlatform
}

// OS Architecture
// 2024-12 MAC OS not supported(ARM64, X86_64 from compute.Image)
func extractOsArchitecture(orgImage *compute.Image) irs.OSArchitecture {
	// if Architecture exists
	if orgImage.Architecture != "" {
		arch := strings.ToLower(orgImage.Architecture)
		switch arch {
		case "arm64":
			return irs.OSArchitecture(orgImage.Architecture)
		case "x86_64", "amd64":
			return irs.OSArchitecture(orgImage.Architecture)
		}
	}

	//
	description := orgImage.Description
	archRegex := regexp.MustCompile(`(?i)\b(mac|x64|amd64|x86|arm64|arm)\b`)

	archMatch := archRegex.FindString(description)

	osArchitecture := irs.ArchitectureNA
	switch archMatch {
	case "arm64":
		if strings.Contains(strings.ToLower(description), "mac") {
			osArchitecture = irs.ARM64_MAC
		} else {
			osArchitecture = irs.ARM64
		}
	case "x64", "amd64":
		if strings.Contains(strings.ToLower(description), "mac") {
			osArchitecture = irs.X86_64_MAC
		} else {
			osArchitecture = irs.X86_64
		}
	default:
		osArchitecture = irs.ArchitectureNA
	}

	return osArchitecture
}

func extractOsDistribution(orgImage *compute.Image) string {
	if orgImage.Description != "" {
		return orgImage.Description
	} else {
		return orgImage.Name
	}
}

// Image Status 추출
func extractImageAvailability(orgImage *compute.Image) irs.ImageStatus {
	//imageStatus := irs.ImageNA
	imageStatus := irs.ImageAvailable

	// Image를 만들 때 체크하는 상태로 보임.
	//status := orgImage.Status
	// if strings.Contains(strings.ToUpper(status), "READY") {
	// 	imageStatus = irs.ImageAvailable
	// } else if strings.Contains(strings.ToUpper(status), "FAILED") ||
	// 	strings.Contains(strings.ToUpper(status), "PENDING") {
	// 	imageStatus = irs.ImageUnavailable
	// }

	// deprecated 가 있는 경우 deprecated.state 가 ACTIVE, DEPRECATED, OBSOLETE, DELETED
	// ACTIVE 는 deprecated가 없음.
	// DEPRECATED인 경우 생성은 되나, warning이 발생함.
	// OBSOLETE, DELETED는 생성도 안되고 error 발생함.
	if orgImage.Deprecated != nil { // deprecated 필드가 nil인 경우
		if orgImage.Deprecated.State == "OBSOLETE" || orgImage.Deprecated.State == "DELETED" {
			imageStatus = irs.ImageUnavailable
		}
	}
	return imageStatus
}

// windows os 여부 return
func (imageHandler *GCPImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	isWindows := false
	resultImage, err := GetPublicImageInfo(imageHandler.Client, imageIID)
	// resultImage, err := FindImageByID(imageHandler.Client, imageIID)
	if err != nil {
		return isWindows, err
	}
	osFeatures := resultImage.GuestOsFeatures

	for _, feature := range osFeatures {
		if feature.Type == "WINDOWS" {
			isWindows = true
		}
	}
	return isWindows, nil
}
