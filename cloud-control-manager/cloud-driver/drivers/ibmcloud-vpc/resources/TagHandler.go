package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/globalsearchv2"
	globalTaggingService "github.com/IBM/platform-services-go-sdk/globaltaggingv1"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type IbmTagHandler struct {
	Region			idrv.RegionInfo
	CredentialInfo	idrv.CredentialInfo
	VpcService     	*vpcv1.VpcV1
	Ctx    			context.Context
}

type TagList struct {
	TotalCount *int64 `json:"total_count"`
	Offset     *int64 `json:"offset"`
	Limit      *int64 `json:"limit"`
	Items      []Tag  `json:"items"`
}

type Tag struct {
	Name *string `json:"name" validate:"required"`
}

type TagResults struct {
	Results []TagResultsItem `json:"results"`
}

type TagResultsItem struct {
	ResourceID *string `json:"resource_id" validate:"required"`
	IsError    *bool   `json:"is_error"`
}

type Item struct {
    CRN  string
    Name string
    Tags []interface{}
    Type string
}

// ListTag implements resources.TagHandler.
func (tagHandler *IbmTagHandler) ListTag(resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
    // IBM Cloud API Key 설정
    authenticator := &core.IamAuthenticator{
        ApiKey: tagHandler.CredentialInfo.ApiKey,
    }

    // GlobalTaggingV1 인스턴스 생성
    service, err := globalTaggingService.NewGlobalTaggingV1(&globalTaggingService.GlobalTaggingV1Options{
        Authenticator: authenticator,
    })
    if err != nil {
        log.Fatalf("Failed to create service: %v", err)
    }

    // ListTagsOptions 생성
    listTagsOptions := service.NewListTagsOptions()
	listTagsOptions.SetTagType("user")
    // listTagsOptions.SetAttachedOnly(false) // 리소스에 연결된 태그만 가져오기
    listTagsOptions.SetProviders([]string{"ghost"})
	listTagsOptions.SetOrderByName("asc")

    // ListTags 호출
    tagList, response, err := service.ListTags(listTagsOptions)
    if err != nil {
        panic(err)
    }

    // 응답 디버깅
    if response != nil {
        fmt.Printf("Response status code: %d\n", response.StatusCode)
    }

    // 태그 데이터 파싱
    var tags []irs.KeyValue

    for _, tag := range tagList.Items {
        tags = append(tags, irs.KeyValue{
            Key:   *tag.Name,
            Value: "",
        })
    }

    // 디버그 출력
    b, _ := json.MarshalIndent(tagList, "", "  ")
    fmt.Println(string(b))

    return tags, nil
}

// GetTag implements resources.TagHandler.
func (tagHandler *IbmTagHandler) GetTag(resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	// IBM Cloud API Key 설정
	authenticator := &core.IamAuthenticator{
		ApiKey: tagHandler.CredentialInfo.ApiKey,
	}

	// Create Global Search service instance
	options := &globalsearchv2.GlobalSearchV2Options{
		Authenticator: authenticator,
	}
	service, err := globalsearchv2.NewGlobalSearchV2(options)
	if err != nil {
		fmt.Println("Error creating Global Search service:", err)
		return irs.KeyValue{}, err
	}

	// Define search options
	searchOptions := service.NewSearchOptions()
	// 검색 쿼리를 설정합니다.
	resourceType := resType
	resourceName := resIID.NameId
	query := fmt.Sprintf("type:%s AND name:%s", resourceType, resourceName)
	// searchQuery := resIID.SystemId
	searchOptions.SetQuery(query)
	searchOptions.SetFields([]string{"name","type","crn","tags"})
	searchOptions.SetLimit(100)

	// Call the service
	scanResult, response, err := service.Search(searchOptions)
	if err != nil {
		fmt.Printf("Error searching for resources: %s\n", err.Error())
		if response != nil {
			fmt.Printf("Response status code: %d\n", response.StatusCode)
			fmt.Printf("Response headers: %v\n", response.Headers)
			fmt.Printf("Response result: %v\n", response.Result)
		}
		return irs.KeyValue{}, err
	}

	var result irs.KeyValue
	// 결과 필터링
    for _, item := range scanResult.Items {
        tags, ok := item.GetProperty("tags").([]interface{})
        if !ok {
            fmt.Println("Error: Tags are not in expected format")
            continue
        }
        for _, tag := range tags {
            tagStr, ok := tag.(string)
            if !ok {
                fmt.Println("Error: Tag is not a string")
                continue
            }

			parts := strings.SplitN(tagStr, ":", 2)
            if parts[0] == key {
                result = irs.KeyValue{Key: parts[0], Value: parts[1]}
				break
            }
        }
    }

	// 응답 디버깅
	if response != nil {
		fmt.Printf("Response status code: %d\n", response.StatusCode)
	}

	return result, err
}


// FindTag implements resources.TagHandler.
func (tagHandler *IbmTagHandler) FindTag(resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
    	// IBM Cloud API Key 설정
	authenticator := &core.IamAuthenticator{
		ApiKey: tagHandler.CredentialInfo.ApiKey,
	}

	// Create Global Search service instance
	options := &globalsearchv2.GlobalSearchV2Options{
		Authenticator: authenticator,
	}
	service, err := globalsearchv2.NewGlobalSearchV2(options)
	if err != nil {
		fmt.Println("Error creating Global Search service:", err)
		return nil, err
	}

	// Define search options
	searchOptions := service.NewSearchOptions()
	// 검색 쿼리를 설정합니다.
	resourceType := resType
	query := fmt.Sprintf("type:%s", resourceType)
	// searchQuery := resIID.SystemId
	searchOptions.SetQuery(query)
	searchOptions.SetFields([]string{"name","resource_id","type","crn","tags"})
	searchOptions.SetLimit(100)

	// Call the service
	scanResult, response, err := service.Search(searchOptions)
	if err != nil {
		fmt.Printf("Error searching for resources: %s\n", err.Error())
		if response != nil {
			fmt.Printf("Response status code: %d\n", response.StatusCode)
			fmt.Printf("Response headers: %v\n", response.Headers)
			fmt.Printf("Response result: %v\n", response.Result)
		}
		return nil, err
	}

	var result []*irs.TagInfo
	// 결과 필터링
    for _, item := range scanResult.Items {
		matchedTag := []irs.KeyValue{}
        tags, ok := item.GetProperty("tags").([]interface{})
        if !ok {
            fmt.Println("Error: Tags are not in expected format")
            continue
        }
        for _, tag := range tags {
            tagStr, ok := tag.(string)
            if !ok {
                fmt.Println("Error: Tag is not a string")
                continue
            }
			parts := strings.SplitN(tagStr, ":", 2)
			Key := parts[0]
			Value := parts[1]

            if Key == keyword || Value == keyword {
                matchedTag = append(matchedTag, irs.KeyValue{Key: Key, Value: Value})
            }
        }
		if len(matchedTag) > 0 {
			item.SetProperty("tags", matchedTag)
			result = append(result, &irs.TagInfo{
				ResType: resType,
				ResIId: irs.IID{NameId: (item.GetProperty("name")).(string),SystemId: (item.GetProperty("resource_id").(string))},
				TagList: matchedTag,
				KeyValueList: matchedTag,
			})
		}
    }



	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))

	// 응답 디버깅
	if response != nil {
		fmt.Printf("Response status code: %d\n", response.StatusCode)
	}

	return result, err
}

// AddTag implements resources.TagHandler.
func (tagHandler *IbmTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	switch resType {
	case "VPC":
		vpc, err := GetRawVPC(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil{
			log.Fatalf("Failed to Attach Tag: %v", err)
		}
		AttachTag(tagHandler.CredentialInfo.ApiKey, resIID, tag, *vpc.CRN)
	case "Subnet":
		vpc, err := GetRawVPC(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil{
			log.Fatalf("Failed to Attach Tag: %v", err)
		}

		subnet, err := getVPCRawSubnet(vpc, resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil{
			log.Fatalf("Failed to Attach Tag: %v", err)
		}
		AttachTag(tagHandler.CredentialInfo.ApiKey, resIID, tag, *subnet.CRN)
	case "SecurityGroup":
		securityGroup, err := getRawSecurityGroup(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil{
			log.Fatalf("Failed to Attach Tag: %v", err)
		}
		fmt.Println(securityGroup)
		AttachTag(tagHandler.CredentialInfo.ApiKey, resIID, tag, *securityGroup.CRN)
	case "VMKeyPair":
		vmKeyPair, err := getRawKey(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil{
			log.Fatalf("Failed to Attach Tag: %v", err)
		}
		AttachTag(tagHandler.CredentialInfo.ApiKey, resIID, tag, *vmKeyPair.CRN)
	case "VM":
		vm, err := getRawInstance(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil{
			log.Fatalf("Failed to Attach Tag: %v", err)
		}
		AttachTag(tagHandler.CredentialInfo.ApiKey, resIID, tag, *vm.CRN)
	case "Disk":
		disk, err := getRawVolume(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil{
			log.Fatalf("Failed to Attach Tag: %v", err)
		}
		AttachTag(tagHandler.CredentialInfo.ApiKey, resIID, tag, *disk.CRN)
	// case "MyImage":
	// 	imageHandler := &IbmMyImageHandler{}
	// 	rawMyimage, err := imageHandler.GetMyImage(resIID)
    //     if err != nil {
    //         getErr := errors.New(fmt.Sprintf("Failed to Get NLB. err = %s", err.Error()))
    //         cblogger.Error(getErr.Error())
    //     }
	// 	AttachTag(tagHandler.CredentialInfo.ApiKey, resIID, tag, *rawMyimage.CRN)
	case "NLB":
		nlbHandler := &IbmNLBHandler{} // or however you initialize IbmNLBHandler
        rawNLB, err := nlbHandler.getRawNLBByName(resIID.NameId)
        if err != nil {
            getErr := errors.New(fmt.Sprintf("Failed to Get NLB. err = %s", err.Error()))
            cblogger.Error(getErr.Error())
        }
		AttachTag(tagHandler.CredentialInfo.ApiKey, resIID, tag, *rawNLB.CRN)
	}

    return irs.KeyValue{Key: tag.Key, Value: tag.Value}, nil
}

// RemoveTag implements resources.TagHandler.
func (tagHandler *IbmTagHandler) RemoveTag(resType irs.RSType, resIID irs.IID, tagName string) (bool, error) {
	switch resType {
	case "VPC":
		vpc, err := GetRawVPC(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil{
			log.Fatalf("Failed to Attach Tag: %v", err)
		}
		DetachTag(tagHandler.CredentialInfo.ApiKey, resIID, tagName, *vpc.CRN)
	case "Subnet":
		vpc, err := GetRawVPC(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil{
			log.Fatalf("Failed to Attach Tag: %v", err)
		}

		subnet, err := getVPCRawSubnet(vpc, resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil{
			log.Fatalf("Failed to Attach Tag: %v", err)
		}
		DetachTag(tagHandler.CredentialInfo.ApiKey, resIID, tagName, *subnet.CRN)
	case "SecurityGroup":
		securityGroup, err := getRawSecurityGroup(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil{
			log.Fatalf("Failed to Attach Tag: %v", err)
		}
		DetachTag(tagHandler.CredentialInfo.ApiKey, resIID, tagName, *securityGroup.CRN)
	case "VMKeyPair":
		vmKeyPair, err := getRawKey(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil{
			log.Fatalf("Failed to Attach Tag: %v", err)
		}
		DetachTag(tagHandler.CredentialInfo.ApiKey, resIID, tagName, *vmKeyPair.CRN)
	case "VM":
		vm, err := getRawInstance(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil{
			log.Fatalf("Failed to Attach Tag: %v", err)
		}
		DetachTag(tagHandler.CredentialInfo.ApiKey, resIID, tagName, *vm.CRN)
	case "Disk":
		disk, err := getRawVolume(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil{
			log.Fatalf("Failed to Attach Tag: %v", err)
		}
		DetachTag(tagHandler.CredentialInfo.ApiKey, resIID, tagName, *disk.CRN)
	// case "MyImage":
	// 	myImageHandler := &IbmMyImageHandler{}
	// 	rawMyimage, err := myImageHandler.GetMyImage(resIID)
    //     if err != nil {
    //         getErr := errors.New(fmt.Sprintf("Failed to Get NLB. err = %s", err.Error()))
    //         cblogger.Error(getErr.Error())
    //     }
	// 	DetachTag(tagHandler.CredentialInfo.ApiKey, resIID, tagName, *rawMyimage.CRN)
	case "NLB":
		nlbHandler := &IbmNLBHandler{} // or however you initialize IbmNLBHandler
        rawNLB, err := nlbHandler.getRawNLBByName(resIID.NameId)
        if err != nil {
            getErr := errors.New(fmt.Sprintf("Failed to Get NLB. err = %s", err.Error()))
            cblogger.Error(getErr.Error())
        }
		DetachTag(tagHandler.CredentialInfo.ApiKey, resIID, tagName, *rawNLB.CRN)
	}

	return true, nil
}

func AttachTag (apikey string, resIID irs.IID, tag irs.KeyValue, CRN string){
	// IBM Cloud API Key 설정
    authenticator := &core.IamAuthenticator{
        ApiKey: apikey,
    }

    // GlobalTaggingV1 인스턴스 생성
    service, err := globalTaggingService.NewGlobalTaggingV1(&globalTaggingService.GlobalTaggingV1Options{
        Authenticator: authenticator,
    })
    if err != nil {
		fmt.Errorf("failed to create service: %w", err)
    }

	resourceModel := globalTaggingService.Resource{
		ResourceID: &CRN,
	  }

	attachTagOptions := service.NewAttachTagOptions(
	[]globalTaggingService.Resource{resourceModel},
	)

	tagName := ""
	if tag.Value == "" {
		tagName = tag.Key
	} else {
		tagName = tag.Key + ":" + tag.Value
	}

	attachTagOptions.SetTagNames([]string{tagName})
	attachTagOptions.SetTagType("user")

	tagResults, response, err := service.AttachTag(attachTagOptions)
	if err != nil {
	}

	// 응답 디버깅
    if response != nil {
        fmt.Printf("Response status code: %d\n", response.StatusCode)
    }

	b, _ := json.MarshalIndent(tagResults, "", "  ")
	fmt.Println(string(b))
}

func DetachTag (apikey string, resIID irs.IID, tagName string, CRN string){
		// IBM Cloud API Key 설정
		authenticator := &core.IamAuthenticator{
			ApiKey: apikey,
		}

		// GlobalTaggingV1 인스턴스 생성
		service, err := globalTaggingService.NewGlobalTaggingV1(&globalTaggingService.GlobalTaggingV1Options{
			Authenticator: authenticator,
		})
		if err != nil {
			fmt.Errorf("failed to delete service: %w", err)
		}

		// Detach 진행
		resourceCRN := CRN
		resourceModel := globalTaggingService.Resource{
			ResourceID: &resourceCRN,
		}

		detachTagOptions := service.NewDetachTagOptions(
		[]globalTaggingService.Resource{resourceModel},
		)

		detachTagOptions.SetTagNames([]string{tagName})
		detachTagOptions.SetTagType("user")

		detachTagResults, response, err := service.DetachTag(detachTagOptions)
		if err != nil {
			log.Fatalf("Failed to delete service: %v", err)
		}
		b, _ := json.MarshalIndent(detachTagResults, "", "  ")
		fmt.Println(string(b))

		// Delete 진행
		deleteTagOptions := service.NewDeleteTagOptions(tagName)
		deleteTagOptions.SetTagType("user")

		deleteTagResults, response, err := service.DeleteTag(deleteTagOptions)
		if err != nil {
			log.Fatalf("Failed to delete service: %v", err)
		}

		b, _ = json.MarshalIndent(deleteTagResults, "", "  ")
		fmt.Println(string(b))

		// 응답 디버깅
		if response != nil {
			fmt.Printf("Response status code: %d\n", response.StatusCode)
		}
}