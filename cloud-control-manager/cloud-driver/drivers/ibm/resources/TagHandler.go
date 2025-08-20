package resources

import (
	"context"
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibm/utils/kubernetesserviceapiv1"
	"strings"
	"time"

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

func getTagFromResource(searchService *globalsearchv2.GlobalSearchV2, crn string, key string) (irs.KeyValue, error) {
	query := strings.ReplaceAll(crn, ":", "\\:")
	query = strings.ReplaceAll(query, "/", "\\/")

	searchOptions := searchService.NewSearchOptions()
	searchOptions.SetQuery(fmt.Sprintf("crn:%s", query))
	searchOptions.SearchCursor = nil
	searchOptions.SetFields([]string{"name", "type", "crn", "tags"})
	searchOptions.SetLimit(1)

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

func tagValidation(tag irs.KeyValue) error {
	if tag.Key == "" {
		return errors.New("tag key is empty")
	}

	if strings.Contains(tag.Key, ":") {
		return errors.New("key should not contain ':'")
	}

	if strings.Contains(tag.Value, ":") {
		return errors.New("value should not contain ':'")
	}

	return nil
}

func getCRN(tagHandler *IbmTagHandler, resType irs.RSType, resIID irs.IID) (string, error) {
	switch resType {
	case irs.VPC:
		vpc, err := GetRawVPC(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil {
			return "", err
		}
		if vpc.CRN == nil {
			return "", errors.New("failed to get VPC CRN")
		}

		return *vpc.CRN, nil
	case irs.SUBNET:
		subnet, err := getRawSubnet(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil {
			return "", err
		}
		if subnet.CRN == nil {
			return "", errors.New("failed to get subnet CRN")
		}

		return *subnet.CRN, nil
	case irs.SG:
		securityGroup, err := getRawSecurityGroup(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil {
			return "", err
		}
		if securityGroup.CRN == nil {
			return "", errors.New("failed to get security group CRN")
		}

		return *securityGroup.CRN, nil
	case irs.KEY:
		vmKeyPair, err := getRawKey(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil {
			return "", err
		}
		if vmKeyPair.CRN == nil {
			return "", errors.New("failed to get keypair CRN")
		}

		return *vmKeyPair.CRN, nil
	case irs.VM:
		vm, err := getRawInstance(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil {
			return "", err
		}
		if vm.CRN == nil {
			return "", errors.New("failed to get VM CRN")
		}

		return *vm.CRN, nil
	case irs.DISK:
		disk, err := getRawVolume(resIID, tagHandler.VpcService, tagHandler.Ctx)
		if err != nil {
			return "", err
		}
		if disk.CRN == nil {
			return "", errors.New("failed to get disk CRN")
		}

		return *disk.CRN, nil
	case irs.MYIMAGE:
		imageHandler := &IbmMyImageHandler{
			CredentialInfo: tagHandler.CredentialInfo,
			Region:         tagHandler.Region,
			VpcService:     tagHandler.VpcService,
			Ctx:            tagHandler.Ctx,
			TaggingService: tagHandler.TaggingService,
			SearchService:  tagHandler.SearchService,
		}
		rawMyimage, err := imageHandler.GetRawMyImage(resIID)
		if err != nil {
			return "", err
		}
		if rawMyimage.CRN == nil {
			return "", errors.New("failed to get myimage CRN")
		}

		return *rawMyimage.CRN, nil
	case irs.NLB:
		nlbHandler := &IbmNLBHandler{
			CredentialInfo: tagHandler.CredentialInfo,
			Region:         tagHandler.Region,
			VpcService:     tagHandler.VpcService,
			Ctx:            tagHandler.Ctx,
			TaggingService: tagHandler.TaggingService,
			SearchService:  tagHandler.SearchService,
		}

		var rawNLB vpcv1.LoadBalancer
		var err error
		if resIID.NameId != "" {
			rawNLB, err = nlbHandler.getRawNLBByName(resIID.NameId)

		} else if resIID.SystemId != "" {
			rawNLB, err = nlbHandler.getRawNLBById(resIID.SystemId)
		}
		if err != nil {
			return "", err
		}

		if rawNLB.CRN == nil {
			return "", errors.New("failed to get NLB CRN")
		}

		return *rawNLB.CRN, nil
	case irs.CLUSTER:
		clusterHandler := &IbmClusterHandler{
			CredentialInfo: tagHandler.CredentialInfo,
			Region:         tagHandler.Region,
			Ctx:            tagHandler.Ctx,
			VpcService:     tagHandler.VpcService,
			ClusterService: tagHandler.ClusterService,
			TaggingService: tagHandler.TaggingService,
			SearchService:  tagHandler.SearchService,
		}
		rawCluster, err := clusterHandler.getRawCluster(resIID)
		if err != nil {
			return "", err
		}

		return rawCluster.Crn, nil
	default:
		return "", errors.New("invalid resource type")
	}
}

func attachOrDetachTag(tagService *globaltaggingv1.GlobalTaggingV1, tag irs.KeyValue, CRN string, action string) error {
	var tagName string
	if tag.Value == "" {
		tagName = tag.Key
	} else {
		tagName = tag.Key + ":" + tag.Value
	}

	resourceModel := globaltaggingv1.Resource{
		ResourceID: &CRN,
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

func handleTagAddOrRemove(tagHandler *IbmTagHandler, resType irs.RSType, crn string,
	tag irs.KeyValue, action string) error {
	var err error

	ibmType := rsTypeToIBMType(resType)
	if ibmType == "" {
		return errors.New("invalid resource type")
	} else if ibmType == "*" {
		return errors.New("all is not supported for getting tag from the resource")
	}

	if action == "remove" {
		tag, err = getTagFromResource(tagHandler.SearchService, crn, tag.Key)
		if err != nil {
			return err
		}
	}

	return attachOrDetachTag(tagHandler.TaggingService, tag, crn, action)
}

func (tagHandler *IbmTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, resIID.NameId, "AddTag()")
	start := call.Start()

	err := tagValidation(tag)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to add a tag. err = %s", err))
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.KeyValue{}, err
	}

	crn, err := getCRN(tagHandler, resType, resIID)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to add a tag. err = %s", err))
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.KeyValue{}, err
	}

	tagFound, _ := getTagFromResource(tagHandler.SearchService, crn, tag.Key)
	if tagFound.Key == tag.Key {
		return tagFound, errors.New("tag with provided key is already exists")
	}

	err = handleTagAddOrRemove(tagHandler, resType, crn, tag, "add")
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to add a tag. err = %s", err))
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.KeyValue{}, err
	}

	var ok bool
	for i := 0; i < 30; i++ {
		_, err = getTagFromResource(tagHandler.SearchService, crn, tag.Key)
		if err == nil {
			ok = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !ok {
		err = errors.New(fmt.Sprintf("Failed to add a tag. err = Complete wait timeout exceeded"))
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.KeyValue{}, err
	}

	LoggingInfo(hiscallInfo, start)

	return irs.KeyValue{Key: tag.Key, Value: tag.Value}, nil
}

func (tagHandler *IbmTagHandler) ListTag(resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, resIID.NameId, "ListTag()")
	start := call.Start()

	crn, err := getCRN(tagHandler, resType, resIID)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to list tag. err = %s", err))
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return []irs.KeyValue{}, err
	}

	query := strings.ReplaceAll(crn, ":", "\\:")
	query = strings.ReplaceAll(query, "/", "\\/")

	searchOptions := tagHandler.SearchService.NewSearchOptions()
	searchOptions.SetQuery(fmt.Sprintf("crn:%s", query))
	searchOptions.SearchCursor = nil
	searchOptions.SetFields([]string{"name", "type", "crn", "tags"})
	searchOptions.SetLimit(1)

	var tagList []irs.KeyValue

	scanResult, _, err := tagHandler.SearchService.Search(searchOptions)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to list tag. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return tagList, err
	}

	if len(scanResult.Items) == 0 {
		return []irs.KeyValue{}, nil
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

	crn, err := getCRN(tagHandler, resType, resIID)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to get tag. err = %s", err))
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.KeyValue{}, err
	}

	tag, err := getTagFromResource(tagHandler.SearchService, crn, key)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to get tag. err = %s", err))
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.KeyValue{}, err
	}

	LoggingInfo(hiscallInfo, start)

	return tag, nil
}

func (tagHandler *IbmTagHandler) RemoveTag(resType irs.RSType, resIID irs.IID, key string) (bool, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.Region, call.TAG, resIID.NameId, "RemoveTag()")
	start := call.Start()

	crn, err := getCRN(tagHandler, resType, resIID)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to remove a tag. err = %s", err))
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}

	err = handleTagAddOrRemove(tagHandler, resType, crn, irs.KeyValue{Key: key}, "remove")
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to remove a tag. err = %s", err))
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}

	var ok bool
	for i := 0; i < 30; i++ {
		_, err = getTagFromResource(tagHandler.SearchService, crn, key)
		if err != nil {
			ok = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !ok {
		err = errors.New(fmt.Sprintf("Failed to remove a tag. err = Complete wait timeout exceeded"))
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
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
					SearchService:  tagHandler.SearchService,
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
