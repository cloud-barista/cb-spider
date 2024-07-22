package resources

import (
	"fmt"
	"strings"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	taglib "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tag/v20180813"

	cbs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs/v20170312"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"

	tke "github.com/tencentcloud/tencentcloud-sdk-go-intl-en/tencentcloud/tke/v20180525"
)

type TencentTagHandler struct {
	Region    idrv.RegionInfo
	TagClient *taglib.Client

	VNetworkClient *vpc.Client
	VMClient       *cvm.Client
	NLBClient      *clb.Client
	DiskClient     *cbs.Client
	ClusterClient  *tke.Client
}

// Map of RSType to Tencent resource types
// tencent need to provide service Type arg, {ServiceType}:{ResourcePrefix}
var rsTypeToTencentTypeMap = map[irs.RSType]string{
	irs.VPC:     "vpc:vpc",
	irs.SUBNET:  "vpc:subnet",
	irs.SG:      "cvm:sg",
	irs.KEY:     "cvm:keypair",
	irs.VM:      "cvm:instance",
	irs.NLB:     "clb:clb",
	irs.DISK:    "cvm:volume",
	irs.MYIMAGE: "cvm:image",
	irs.CLUSTER: "ccs:cluster",
}

// Map of RSType to Tencent resource types reversed
// tencent need to provide service Type arg, {resourceIdStartStr}:irs.Type
var resourceIdStartStrToRsType = map[string]irs.RSType{
	"vpc":    irs.VPC,
	"subnet": irs.SUBNET,
	"sg":     irs.SG,
	"skey":   irs.KEY,
	"ins":    irs.VM,
	"lb":     irs.NLB,
	"disk":   irs.DISK,
	"img":    irs.MYIMAGE,
	"cls":    irs.CLUSTER,
}

// tencent need to provide service Type arg.
// rsTypeToTencentTypeMap arg return split by ':' (ServiceType, ResourcePrefix, resourceIdStartStr)
// return ServiceType and ResourcePrefix
func rsTypeToTencentTypeMapParse(rstypeTencent string) (string, string) {
	rstypeTencentArr := strings.SplitN(rstypeTencent, ":", 2)
	return rstypeTencentArr[0], rstypeTencentArr[1]
}

func uniqueStringSlice(strings []string) []string {
	stringMap := make(map[string]struct{})
	uniqueStrings := []string{}

	for _, str := range strings {
		if _, exists := stringMap[str]; !exists {
			stringMap[str] = struct{}{}
			uniqueStrings = append(uniqueStrings, str)
		}
	}

	return uniqueStrings
}

func validateResource(t *TencentTagHandler, resType irs.RSType, resIID irs.IID) (bool, error) {
	switch resType {
	case irs.VPC: // region Require
		request := vpc.NewDescribeVpcsRequest()
		request.VpcIds = common.StringPtrs([]string{resIID.SystemId})
		response, err := t.VNetworkClient.DescribeVpcs(request)
		if err != nil {
			err := fmt.Errorf("An VPC API error has returned: %s", err.Error())
			return false, err
		}
		if *response.Response.TotalCount > 0 {
			return true, nil
		}
		return false, nil

	case irs.SUBNET: // region Require
		request := vpc.NewDescribeSubnetsRequest()
		request.SubnetIds = common.StringPtrs([]string{resIID.SystemId})
		response, err := t.VNetworkClient.DescribeSubnets(request)
		if err != nil {
			err := fmt.Errorf("An SUBNET API error has returned: %s", err.Error())
			return false, err
		}
		if *response.Response.TotalCount > 0 {
			return true, nil
		}
		return false, nil

	// VPC와 클라이언트 공유함.
	case irs.SG: // region Require
		request := vpc.NewDescribeSecurityGroupsRequest()
		request.SecurityGroupIds = common.StringPtrs([]string{resIID.SystemId})
		response, err := t.VNetworkClient.DescribeSecurityGroups(request)
		if err != nil {
			err := fmt.Errorf("An SG API error has returned: %s", err.Error())
			return false, err
		}
		if *response.Response.TotalCount > 0 {
			return true, nil
		}
		return false, nil

	// VM과 클라이언트 공유함.
	case irs.KEY: // region Require
		request := cvm.NewDescribeKeyPairsRequest()
		request.KeyIds = common.StringPtrs([]string{resIID.SystemId})
		response, err := t.VMClient.DescribeKeyPairs(request)
		if err != nil {
			err := fmt.Errorf("An KEY API error has returned: %s", err.Error())
			return false, err
		}
		if *response.Response.TotalCount > 0 {
			return true, nil
		}
		return false, nil

	case irs.VM: // region Require
		request := cvm.NewDescribeInstancesRequest()
		request.InstanceIds = common.StringPtrs([]string{resIID.SystemId})
		response, err := t.VMClient.DescribeInstances(request)
		if err != nil {
			err := fmt.Errorf("An VM API error has returned: %s", err.Error())
			return false, err
		}
		if *response.Response.TotalCount > 0 {
			return true, nil
		}
		return false, nil

	case irs.NLB: // region Require
		request := clb.NewDescribeLoadBalancersRequest()
		request.LoadBalancerIds = common.StringPtrs([]string{resIID.SystemId})
		response, err := t.NLBClient.DescribeLoadBalancers(request)
		if err != nil {
			err := fmt.Errorf("An NLB API error has returned: %s", err.Error())
			return false, err
		}
		if *response.Response.TotalCount > 0 {
			return true, nil
		}
		return false, nil

	case irs.DISK: // region Require
		request := cbs.NewDescribeDisksRequest()
		request.DiskIds = common.StringPtrs([]string{resIID.SystemId})
		response, err := t.DiskClient.DescribeDisks(request)
		if err != nil {
			err := fmt.Errorf("An DISK API error has returned: %s", err.Error())
			return false, err
		}
		if *response.Response.TotalCount > 0 {
			return true, nil
		}
		return false, nil

	// VM과 클라이언트 공유함.
	case irs.MYIMAGE: // region Require
		request := cvm.NewDescribeImagesRequest()
		request.ImageIds = common.StringPtrs([]string{resIID.SystemId})
		response, err := t.VMClient.DescribeImages(request)
		if err != nil {
			err := fmt.Errorf("An MYIMAGE API error has returned: %s", err.Error())
			return false, err
		}
		if *response.Response.TotalCount > 0 {
			return true, nil
		}
		return false, nil

	case irs.CLUSTER: // region Require
		request := tke.NewDescribeClustersRequest()
		request.ClusterIds = common.StringPtrs([]string{resIID.SystemId})
		response, err := t.ClusterClient.DescribeClusters(request)
		if err != nil {
			err := fmt.Errorf("An CLUSTER API error has returned: %s", err.Error())
			return false, err
		}
		if *response.Response.TotalCount > 0 {
			return true, nil
		}
		return false, nil

	default:
		msg := "no resType to validate"
		return false, fmt.Errorf("%s", msg)
	}
}

func (t *TencentTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	cblogger.Debugf("Req resTyp:[%s] / resIID:[%s] / Tag Key:[%s] / Tag Value:[%s]", resType, resIID, tag.Key, tag.Value)

	if resIID.SystemId == "" || tag.Key == "" {
		msg := "tag will not be add because resIID.SystemId or tag.Key is not provided"
		cblogger.Error(msg)
		return irs.KeyValue{}, fmt.Errorf("%s", msg)
	}
	if resType == irs.ALL {
		msg := "addTag is not allow to return for all rstype"
		cblogger.Error(msg)
		return irs.KeyValue{}, fmt.Errorf("%s", msg)
	}

	hiscallInfo := GetCallLogScheme(t.Region, call.TAG, resIID.SystemId, "AddTag()")
	start := call.Start()

	isValid, err := validateResource(t, resType, resIID)
	if err != nil {
		msg := "error while validate resource: " + err.Error()
		cblogger.Error(msg)
		return irs.KeyValue{}, err
	}
	if !isValid {
		msg := fmt.Sprintf("resource is not exist, Region: %s, ResourceIID: %s\n", t.Region.Region, resIID.SystemId)
		cblogger.Error(msg)
		return irs.KeyValue{}, fmt.Errorf("%s", msg)
	}

	// create Tag
	createTagReq := taglib.NewCreateTagRequest()
	createTagReq.TagKey = common.StringPtr(tag.Key)
	createTagReq.TagValue = common.StringPtr(tag.Value)

	createTagRes, err := t.TagClient.CreateTag(createTagReq)
	if err != nil && err.(*errors.TencentCloudSDKError).GetCode() != taglib.RESOURCEINUSE_TAGDUPLICATE {
		msg := "createTag error has returned: " + err.Error()
		cblogger.Error(msg)
		return irs.KeyValue{}, err
	}

	if cblogger.Level.String() == "debug" {
		cblogger.Infof("createTagRes: %s\n", createTagRes.ToJsonString())
	}

	serviceTypeStr, resourcePrefixStr := rsTypeToTencentTypeMapParse(rsTypeToTencentTypeMap[resType])

	// attach Tag
	attachTagReq := taglib.NewAttachResourcesTagRequest()
	attachTagReq.ServiceType = common.StringPtr(serviceTypeStr)
	attachTagReq.ResourceRegion = common.StringPtr(t.Region.Region)
	attachTagReq.ResourcePrefix = common.StringPtr(resourcePrefixStr)
	attachTagReq.ResourceIds = common.StringPtrs([]string{resIID.SystemId})
	attachTagReq.TagKey = common.StringPtr(tag.Key)
	attachTagReq.TagValue = common.StringPtr(tag.Value)

	attachTagRes, err := t.TagClient.AttachResourcesTag(attachTagReq)
	if err != nil {
		msg := "attachTag error has returned: " + err.Error()
		cblogger.Error(msg)
		return irs.KeyValue{}, err
	}

	if cblogger.Level.String() == "debug" {
		cblogger.Infof("attachTagRes: %s\n", attachTagRes.ToJsonString())
	}

	LoggingInfo(hiscallInfo, start)

	return tag, nil
}

func (t *TencentTagHandler) ListTag(resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
	cblogger.Debugf("Req resTyp:[%s] / resIID:[%s]", resType, resIID)
	if resIID.SystemId == "" {
		msg := "resIID.SystemId is not provided"
		cblogger.Error(msg)
		return nil, fmt.Errorf("%s", msg)
	}
	if resType == irs.ALL {
		msg := "listTag is not allow to return for all rstype"
		cblogger.Error(msg)
		return nil, fmt.Errorf("%s", msg)
	}

	serviceTypeStr, resourcePrefixStr := rsTypeToTencentTypeMapParse(rsTypeToTencentTypeMap[resType])

	hiscallInfo := GetCallLogScheme(t.Region, call.TAG, resIID.SystemId, "ListTag()")
	start := call.Start()

	req := taglib.NewDescribeResourceTagsRequest()
	req.ResourceRegion = common.StringPtr(t.Region.Region)
	req.ServiceType = common.StringPtr(serviceTypeStr)
	req.ResourcePrefix = common.StringPtr(resourcePrefixStr)
	req.ResourceId = common.StringPtr(resIID.SystemId)

	res, err := t.TagClient.DescribeResourceTags(req)
	if err != nil {
		return nil, err
	}

	LoggingInfo(hiscallInfo, start)

	var tagList []irs.KeyValue
	for _, tag := range res.Response.Rows {
		tagList = append(tagList, irs.KeyValue{
			Key:   *tag.TagKey,
			Value: *tag.TagValue,
		})
	}

	return tagList, nil
}

func (t *TencentTagHandler) GetTag(resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	cblogger.Debugf("Req resTyp:[%s] / resIID:[%s] / key:[%s]", resType, resIID, key)
	if resIID.SystemId == "" || key == "" {
		msg := "resIID.SystemId or key is not provided"
		cblogger.Error(msg)
		return irs.KeyValue{}, fmt.Errorf("%s", msg)
	}
	if resType == irs.ALL {
		msg := "getTag is not allow to return for all rstype"
		cblogger.Error(msg)
		return irs.KeyValue{}, fmt.Errorf("%s", msg)
	}

	serviceTypeStr, resourcePrefixStr := rsTypeToTencentTypeMapParse(rsTypeToTencentTypeMap[resType])

	hiscallInfo := GetCallLogScheme(t.Region, call.TAG, resIID.SystemId, "GetTag()")
	start := call.Start()

	req := taglib.NewDescribeResourceTagsByTagKeysRequest()
	req.ServiceType = common.StringPtr(serviceTypeStr)
	req.ResourcePrefix = common.StringPtr(resourcePrefixStr)
	req.ResourceRegion = common.StringPtr(t.Region.Region)
	req.ResourceIds = common.StringPtrs([]string{resIID.SystemId})
	req.TagKeys = common.StringPtrs([]string{key})
	req.Limit = common.Uint64Ptr(1)

	res, err := t.TagClient.DescribeResourceTagsByTagKeys(req)
	if err != nil {
		return irs.KeyValue{}, err
	}

	LoggingInfo(hiscallInfo, start)

	if len(res.Response.Rows) == 0 {
		msg := "tag with key " + key + " not found"
		cblogger.Error(msg)
		return irs.KeyValue{}, fmt.Errorf("%s", msg)
	}

	resTag := irs.KeyValue{Key: key, Value: *res.Response.Rows[0].TagKeyValues[0].TagValue}

	return resTag, nil
}

func (t *TencentTagHandler) RemoveTag(resType irs.RSType, resIID irs.IID, key string) (bool, error) {
	cblogger.Debugf("Req resTyp:[%s] / resIID:[%s] / key:[%s]", resType, resIID, key)
	if resIID.SystemId == "" || key == "" {
		msg := "resIID.SystemId or key is not provided"
		cblogger.Error(msg)
		return false, fmt.Errorf("%s", msg)
	}
	if resType == irs.ALL {
		msg := "getTag is not allow to return for all rstype"
		cblogger.Error(msg)
		return false, fmt.Errorf("%s", msg)
	}

	serviceTypeStr, resourcePrefixStr := rsTypeToTencentTypeMapParse(rsTypeToTencentTypeMap[resType])

	hiscallInfo := GetCallLogScheme(t.Region, call.TAG, resIID.SystemId, "GetTag()")
	start := call.Start()

	req := taglib.NewDetachResourcesTagRequest()

	req.ServiceType = common.StringPtr(serviceTypeStr)
	req.ResourcePrefix = common.StringPtr(resourcePrefixStr)
	req.ResourceRegion = common.StringPtr(t.Region.Region)
	req.ResourceIds = common.StringPtrs([]string{resIID.SystemId})
	req.TagKey = common.StringPtr(key)

	res, err := t.TagClient.DetachResourcesTag(req)
	if err != nil {
		return false, err
	}

	LoggingInfo(hiscallInfo, start)

	if cblogger.Level.String() == "debug" {
		cblogger.Infof("%s", res.ToJsonString())
	}

	return true, nil
}

func (t *TencentTagHandler) FindTag(resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	cblogger.Debugf("Req resTyp:[%s] / keyword:[%s]", resType, keyword)
	if resType == "" {
		msg := "resType is not provided"
		cblogger.Error(msg)
		return nil, fmt.Errorf("%s", msg)
	}

	req := taglib.NewDescribeResourceTagsRequest()
	if resType != irs.ALL {
		serviceTypeStr, resourcePrefixStr := rsTypeToTencentTypeMapParse(rsTypeToTencentTypeMap[resType])
		req.ServiceType = common.StringPtr(serviceTypeStr)
		req.ResourcePrefix = common.StringPtr(resourcePrefixStr)
	}
	req.ResourceRegion = common.StringPtr(t.Region.Region)
	req.Limit = common.Uint64Ptr(1)

	initRes, err := t.TagClient.DescribeResourceTags(req)
	if err != nil {
		return nil, err
	}

	req.Limit = common.Uint64Ptr(*initRes.Response.TotalCount)

	res, err := t.TagClient.DescribeResourceTags(req)
	if err != nil {
		return nil, err
	}

	tagInfoResIdArr := map[irs.RSType][]string{}
	for _, tag := range res.Response.Rows {
		if strings.Contains(*tag.TagKey, keyword) || strings.Contains(*tag.TagValue, keyword) || keyword == "" || keyword == "*" {
			resourceIdSplitArr := strings.Split(*tag.ResourceId, "-")
			tagInfoResIdArr[resourceIdStartStrToRsType[resourceIdSplitArr[0]]] = append(tagInfoResIdArr[resourceIdStartStrToRsType[resourceIdSplitArr[0]]], *tag.ResourceId)
		}
	}

	var resultTagInfos []*irs.TagInfo
	for rsType, idArr := range tagInfoResIdArr {
		uniqueIds := uniqueStringSlice(idArr)
		if rsType == "" {
			continue
		}

		for _, id := range uniqueIds {
			tagInfo := &irs.TagInfo{
				ResType: rsType,
				ResIId:  irs.IID{SystemId: id},
			}
			resultTagInfos = append(resultTagInfos, tagInfo)
		}

		request := taglib.NewDescribeResourceTagsByResourceIdsRequest()

		serviceTypeStr, resourcePrefixStr := rsTypeToTencentTypeMapParse(rsTypeToTencentTypeMap[rsType])
		request.ServiceType = common.StringPtr(serviceTypeStr)
		request.ResourcePrefix = common.StringPtr(resourcePrefixStr)
		request.ResourceIds = common.StringPtrs(uniqueIds)
		request.ResourceRegion = common.StringPtr(t.Region.Region)

		response, err := t.TagClient.DescribeResourceTagsByResourceIds(request)
		if err != nil {
			return nil, err
		}

		for _, tag := range response.Response.Tags {
			for idx, tagInfos := range resultTagInfos {
				if *tag.ResourceId == tagInfos.ResIId.SystemId {
					kv := irs.KeyValue{
						Key:   *tag.TagKey,
						Value: *tag.TagValue,
					}
					resultTagInfos[idx].TagList = append(resultTagInfos[idx].TagList, kv)
					break
				}
			}
		}

	}

	return resultTagInfos, nil
}
