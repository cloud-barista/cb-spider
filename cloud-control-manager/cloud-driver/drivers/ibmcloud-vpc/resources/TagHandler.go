package resources

import (
	"context"
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibmcloud-vpc/utils/kubernetesserviceapiv1"
	"strings"

	"github.com/IBM/platform-services-go-sdk/globalsearchv2"
	"github.com/IBM/platform-services-go-sdk/globaltaggingv1"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type IbmTagHandler struct {
	Region         idrv.RegionInfo
	CredentialInfo idrv.CredentialInfo
	VpcService     *vpcv1.VpcV1
	ClusterService *kubernetesserviceapiv1.KubernetesServiceApiV1
	Ctx            context.Context
	TaggingService *globaltaggingv1.GlobalTaggingV1
	SearchService  *globalsearchv2.GlobalSearchV2
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

func rsTypeToIBMType(resType irs.RSType) string {
	switch resType {
	case irs.ALL:
		return "*"
	case irs.IMAGE:
		return "image"
	case irs.VPC:
		return "vpc"
	case irs.SUBNET:
		return "subnet"
	case irs.SG:
		return "security-group"
	case irs.KEY:
		return "key"
	case irs.VM:
		return "instance"
	case irs.NLB:
		return "load-balancer"
	case irs.DISK:
		return "volume"
	case irs.MYIMAGE:
		return "snapshot"
	case irs.CLUSTER:
		return "k8-cluster"
	// NODEGROUP's Not Support
	// case irs.NODEGROUP:
	// 	return "instance-group"
	default:
		return ""
	}
}

func ibmTypeToRSType(ibmType string) (irs.RSType, error) {
	switch ibmType {
	case "image":
		return irs.IMAGE, nil
	case "vpc":
		return irs.VPC, nil
	case "subnet":
		return irs.SUBNET, nil
	case "security-group":
		return irs.SG, nil
	case "key":
		return irs.KEY, nil
	case "instance":
		return irs.VM, nil
	case "load-balancer":
		return irs.NLB, nil
	case "volume":
		return irs.DISK, nil
	case "snapshot":
		return irs.MYIMAGE, nil
	case "k8-cluster":
		return irs.CLUSTER, nil
	case "instance-group":
		return irs.NODEGROUP, nil
	default:
		return "", errors.New(fmt.Sprintf("unsupport type %s", ibmType))
	}
}

func getTagFromResource(searchService *globalsearchv2.GlobalSearchV2,
	resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	searchOptions := searchService.NewSearchOptions()

	var query string

	ibmType := rsTypeToIBMType(resType)
	if ibmType == "" {
		return irs.KeyValue{}, errors.New("invalid resource type")
	}

	if resIID.NameId != "" {
		query = fmt.Sprintf("type:%s AND name:%s", ibmType, resIID.NameId)
	} else {
		query = fmt.Sprintf("type:%s AND id:%s", ibmType, resIID.SystemId)
	}

	searchOptions.SetQuery(query)
	searchOptions.SearchCursor = nil
	searchOptions.SetFields([]string{"name", "type", "crn", "tags"})
	searchOptions.SetLimit(100)

	scanResult, _, err := searchService.Search(searchOptions)
	if err != nil {
		return irs.KeyValue{}, err
	}

	if len(scanResult.Items) == 0 {
		return irs.KeyValue{}, errors.New("resource not found")
	}

	searchOptions.SearchCursor = scanResult.SearchCursor

	for _, item := range scanResult.Items {
		tags, ok := item.GetProperty("tags").([]interface{})
		if !ok {
			cblogger.Error("Tags are not in expected format")
			continue
		}
		for _, tag := range tags {
			tagStr, ok := tag.(string)
			if !ok {
				cblogger.Errorf("Tag is not a string (%v)", tag)
				continue
			}

			parts := strings.SplitN(tagStr, ":", 2)
			if parts[0] == key {
				return irs.KeyValue{Key: parts[0], Value: parts[1]}, nil
			}
		}
	}

	return irs.KeyValue{}, errors.New("tag not found")
}

func attachOrDetachTag(tagService *globaltaggingv1.GlobalTaggingV1, tag irs.KeyValue, CRN string, action string) error {
	resourceModel := globaltaggingv1.Resource{
		ResourceID: &CRN,
	}

	var tagName string
	if tag.Value == "" {
		tagName = tag.Key
	} else {
		tagName = tag.Key + ":" + tag.Value
	}

	switch action {
	case "add":
		attachTagOptions := tagService.NewAttachTagOptions(
			[]globaltaggingv1.Resource{resourceModel},
		)

		attachTagOptions.SetTagNames([]string{tagName})
		attachTagOptions.SetTagType("user")

		_, _, err := tagService.AttachTag(attachTagOptions)
		if err != nil {
			return err
		}
	case "remove":
		detachTagOptions := tagService.NewDetachTagOptions(
			[]globaltaggingv1.Resource{resourceModel},
		)

		detachTagOptions.SetTagNames([]string{tagName})
		detachTagOptions.SetTagType("user")

		_, _, err := tagService.DetachTag(detachTagOptions)
		if err != nil {
			return err
		}

		deleteTagAllOptions := tagService.NewDeleteTagAllOptions()
		deleteTagAllOptions.SetTagType("user")

		_, _, err = tagService.DeleteTagAll(deleteTagAllOptions)
		if err != nil {
			return err
		}
	}

	return nil
}

func handleTagAddOrRemove(tagHandler *IbmTagHandler, resType irs.RSType, resIID irs.IID,
	tag irs.KeyValue, action string) error {
	var err2 error

	if action == "remove" {
		tag, err2 = getTagFromResource(tagHandler.SearchService, resType, resIID, tag.Key)
		if err2 != nil {
			return err2
		}
	}

	ibmType := rsTypeToIBMType(resType)
	if ibmType == "" {
		return errors.New("invalid resource type")
	} else if ibmType == "all" {
		return errors.New("all is not supported for getting tag from the resource")
	}

	switch resType {
	case irs.VPC:
		vpc, err := GetRawVPC(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil {
			err2 = errors.New(fmt.Sprintf("Failed to add tag. err = %s", err))
			break
		}
		err2 = attachOrDetachTag(tagHandler.TaggingService, tag, *vpc.CRN, action)
	case irs.SUBNET:
		subnet, err := getRawSubnet(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil {
			err2 = errors.New(fmt.Sprintf("Failed to add tag. err = %s", err))
			break
		}

		err2 = attachOrDetachTag(tagHandler.TaggingService, tag, *subnet.CRN, action)
	case irs.SG:
		securityGroup, err := getRawSecurityGroup(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil {
			err2 = errors.New(fmt.Sprintf("Failed to add tag. err = %s", err))
			break
		}

		err2 = attachOrDetachTag(tagHandler.TaggingService, tag, *securityGroup.CRN, action)
	case irs.KEY:
		vmKeyPair, err := getRawKey(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil {
			err2 = errors.New(fmt.Sprintf("Failed to add tag. err = %s", err))
			break
		}

		err2 = attachOrDetachTag(tagHandler.TaggingService, tag, *vmKeyPair.CRN, action)
	case irs.VM:
		vm, err := getRawInstance(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil {
			err2 = errors.New(fmt.Sprintf("Failed to add tag. err = %s", err))
			break
		}

		err2 = attachOrDetachTag(tagHandler.TaggingService, tag, *vm.CRN, action)
	case irs.DISK:
		disk, err := getRawVolume(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil {
			err2 = errors.New(fmt.Sprintf("Failed to add tag. err = %s", err))
			break
		}

		err2 = attachOrDetachTag(tagHandler.TaggingService, tag, *disk.CRN, action)
	case irs.MYIMAGE:
		imageHandler := &IbmMyImageHandler{
			CredentialInfo: tagHandler.CredentialInfo,
			Region:         tagHandler.Region,
			VpcService:     tagHandler.VpcService,
			Ctx:            tagHandler.Ctx,
		}
		rawMyimage, err := imageHandler.GetRawMyImage(resIID)
		if err != nil {
			err2 = errors.New(fmt.Sprintf("Failed to add tag. err = %s", err))
			break
		}
		err2 = attachOrDetachTag(tagHandler.TaggingService, tag, *rawMyimage.CRN, action)
	case irs.NLB:
		nlbHandler := &IbmNLBHandler{
			CredentialInfo: tagHandler.CredentialInfo,
			Region:         tagHandler.Region,
			VpcService:     tagHandler.VpcService,
			Ctx:            tagHandler.Ctx,
		}
		rawNLB, err := nlbHandler.getRawNLBByName(resIID.NameId)
		if err != nil {
			err2 = errors.New(fmt.Sprintf("Failed to add tag. err = %s", err))
			break
		}

		err2 = attachOrDetachTag(tagHandler.TaggingService, tag, *rawNLB.CRN, action)
	case irs.CLUSTER:
		clusterHandler := &IbmClusterHandler{
			CredentialInfo: tagHandler.CredentialInfo,
			Region:         tagHandler.Region,
			Ctx:            tagHandler.Ctx,
			VpcService:     tagHandler.VpcService,
			ClusterService: tagHandler.ClusterService,
			TaggingService: tagHandler.TaggingService,
		}
		rawCluster, err := clusterHandler.getRawCluster(resIID)
		if err != nil {
			err2 = errors.New(fmt.Sprintf("Failed to add tag. err = %s", err))
			break
		}

		err2 = attachOrDetachTag(tagHandler.TaggingService, tag, rawCluster.Crn, action)
	default:
		return errors.New("invalid resource type")
	}

	return err2
}

func (tagHandler *IbmTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, resIID.NameId, "AddTag()")
	start := call.Start()

	tagFound, _ := getTagFromResource(tagHandler.SearchService, resType, resIID, tag.Key)
	if tagFound.Key == tag.Key {
		return tagFound, errors.New("tag with provided key is already exists")
	}

	err := handleTagAddOrRemove(tagHandler, resType, resIID, tag, "add")
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to add a tag. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.KeyValue{}, getErr
	}

	LoggingInfo(hiscallInfo, start)

	return irs.KeyValue{Key: tag.Key, Value: tag.Value}, nil
}

func (tagHandler *IbmTagHandler) ListTag(resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, resIID.NameId, "ListTag()")
	start := call.Start()

	ibmType := rsTypeToIBMType(resType)
	if ibmType == "" {
		return []irs.KeyValue{}, errors.New("invalid resource type")
	}

	searchOptions := tagHandler.SearchService.NewSearchOptions()

	var query string

	if resIID.NameId != "" {
		query = fmt.Sprintf("type:%s AND name:%s", ibmType, resIID.NameId)
	} else {
		query = fmt.Sprintf("type:%s AND id:%s", ibmType, resIID.SystemId)
	}

	searchOptions.SetQuery(query)
	searchOptions.SearchCursor = nil
	searchOptions.SetFields([]string{"name", "type", "crn", "tags"})
	searchOptions.SetLimit(100)

	var tagList []irs.KeyValue

	scanResult, _, err := tagHandler.SearchService.Search(searchOptions)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to list tag. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return tagList, err
	}

	if len(scanResult.Items) == 0 {
		return []irs.KeyValue{}, errors.New("resource not found")
	}

	searchOptions.SearchCursor = scanResult.SearchCursor

	for _, item := range scanResult.Items {
		tags, ok := item.GetProperty("tags").([]interface{})
		if !ok {
			cblogger.Error("Tags are not in expected format")
			continue
		}
		for _, tag := range tags {
			tagStr, ok := tag.(string)
			if !ok {
				cblogger.Error("Tag is not a string")
				continue
			}

			parts := strings.SplitN(tagStr, ":", 2)
			tagList = append(tagList, irs.KeyValue{
				Key:   parts[0],
				Value: parts[1],
			})
		}
	}

	LoggingInfo(hiscallInfo, start)

	return tagList, nil
}

func (tagHandler *IbmTagHandler) GetTag(resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, resIID.NameId, "GetTag()")
	start := call.Start()

	tag, err := getTagFromResource(tagHandler.SearchService, resType, resIID, key)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get tag. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.KeyValue{}, err
	}

	LoggingInfo(hiscallInfo, start)

	return tag, nil
}

func (tagHandler *IbmTagHandler) RemoveTag(resType irs.RSType, resIID irs.IID, key string) (bool, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, resIID.NameId, "RemoveTag()")
	start := call.Start()

	err := handleTagAddOrRemove(tagHandler, resType, resIID, irs.KeyValue{Key: key}, "remove")
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to remove a tag. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}

	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func (tagHandler *IbmTagHandler) FindTag(resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, keyword, "FindTag()")
	start := call.Start()

	ibmType := rsTypeToIBMType(resType)
	if ibmType == "" {
		return []*irs.TagInfo{}, errors.New("invalid resource type")
	}

	searchOptions := tagHandler.SearchService.NewSearchOptions()

	query := fmt.Sprintf("type:%s", ibmType)
	searchOptions.SetQuery(query)
	searchOptions.SearchCursor = nil
	searchOptions.SetFields([]string{"name", "resource_id", "type", "crn", "tags"})
	searchOptions.SetLimit(100)

	var tagInfo []*irs.TagInfo

	scanResult, _, err := tagHandler.SearchService.Search(searchOptions)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to list tag. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return tagInfo, err
	}

	if len(scanResult.Items) == 0 {
		return []*irs.TagInfo{}, errors.New("resource not found")
	}

	searchOptions.SearchCursor = scanResult.SearchCursor

	for _, item := range scanResult.Items {
		var tagFound bool

		tags, ok := item.GetProperty("tags").([]interface{})
		if !ok {
			cblogger.Error("Tags are not in expected format")
			continue
		}
		for _, tag := range tags {
			tagStr, ok := tag.(string)
			if !ok {
				cblogger.Errorf("Tag is not a string (%v)", tag)
				continue
			}
			if strings.Contains(tagStr, keyword) {
				tagFound = true
				break
			}
		}

		if tagFound {
			var tagKeyValue []irs.KeyValue
			for _, tag := range tags {
				tagStr, ok := tag.(string)
				if !ok {
					cblogger.Errorf("Tag is not a string (%v)", tag)
					continue
				}
				parts := strings.SplitN(tagStr, ":", 2)
				tagKeyValue = append(tagKeyValue, irs.KeyValue{
					Key:   parts[0],
					Value: parts[1],
				})
			}

			rType, ok := item.GetProperty("type").(string)
			if !ok {
				cblogger.Error("type is not a string")
				continue
			}
			rsType, err := ibmTypeToRSType(rType)
			if err != nil {
				cblogger.Error(err)
				continue
			}

			name, ok := item.GetProperty("name").(string)
			if !ok {
				cblogger.Error("name is not a string")
				continue
			}
			resourceId, ok := item.GetProperty("resource_id").(string)
			if !ok {
				cblogger.Error("resource_id is not a string")
				continue
			}

			if rsType == irs.CLUSTER {
				clusterHandler := &IbmClusterHandler{
					CredentialInfo: tagHandler.CredentialInfo,
					Region:         tagHandler.Region,
					Ctx:            tagHandler.Ctx,
					VpcService:     tagHandler.VpcService,
					ClusterService: tagHandler.ClusterService,
					TaggingService: tagHandler.TaggingService,
				}
				rawCluster, err := clusterHandler.getRawCluster(irs.IID{NameId: name})
				if err != nil {
					cblogger.Error(err)
					continue
				}
				resourceId = rawCluster.Id
			}

			tagInfo = append(tagInfo, &irs.TagInfo{
				ResType:      rsType,
				ResIId:       irs.IID{NameId: name, SystemId: resourceId},
				TagList:      tagKeyValue,
				KeyValueList: []irs.KeyValue{}, // reserved for optional usage
			})
		}
	}

	LoggingInfo(hiscallInfo, start)

	return tagInfo, nil
}
