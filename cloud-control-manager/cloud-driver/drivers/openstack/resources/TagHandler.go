package resources

import (
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	computeTags "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/tags"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	networkTags "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"strings"
)

type OpenStackTagHandler struct {
	CredentialInfo idrv.CredentialInfo
	IdentityClient *gophercloud.ServiceClient
	ComputeClient  *gophercloud.ServiceClient
	NetworkClient  *gophercloud.ServiceClient
	NLBClient      *gophercloud.ServiceClient
}

func tagToKeyValueString(tag irs.KeyValue) (string, error) {
	if tag.Key == "" {
		return "", errors.New("tag key is empty")
	}

	if strings.Contains(tag.Key, "=") {
		return "", errors.New("key should not contain '='")
	}

	if strings.Contains(tag.Value, "=") {
		return "", errors.New("value should not contain '='")
	}

	return tag.Key + "=" + tag.Value, nil
}

func getResourceSystemID(tagHandler *OpenStackTagHandler, resType irs.RSType, resIID irs.IID) (string, error) {
	if resIID.SystemId == "" && resIID.NameId == "" {
		return "", errors.New("invalid IID")
	}

	if resIID.SystemId != "" {
		return resIID.SystemId, nil
	}

	switch resType {
	case irs.VPC:
		pager, err := networks.List(tagHandler.NetworkClient, external.ListOptsExt{
			ListOptsBuilder: networks.ListOpts{
				Name: resIID.NameId,
			},
		}).AllPages()
		if err != nil {
			return "", err
		}
		var vpcList []NetworkWithExt
		err = networks.ExtractNetworksInto(pager, &vpcList)
		if err != nil {
			return "", err
		}
		for _, vpc := range vpcList {
			if vpc.Name == resIID.NameId {
				return vpc.ID, nil
			}
		}
		return "", errors.New("vpc not found")
	case irs.SUBNET:
		pager, err := subnets.List(tagHandler.NetworkClient, subnets.ListOpts{
			Name: resIID.NameId,
		}).AllPages()
		if err != nil {
			return "", err
		}
		ss, err := subnets.ExtractSubnets(pager)
		if err != nil {
			return "", err
		}
		for _, subnet := range ss {
			if subnet.Name == resIID.NameId {
				return subnet.ID, nil
			}
		}
		return "", errors.New("subnet not found")
	case irs.SG:
		pager, err := secgroups.List(tagHandler.ComputeClient).AllPages()
		if err != nil {
			return "", err
		}
		sgs, err := secgroups.ExtractSecurityGroups(pager)
		if err != nil {
			return "", err
		}
		for _, sg := range sgs {
			if sg.Name == resIID.NameId {
				return sg.ID, nil
			}
		}
		return "", errors.New("security group not found")
	case irs.NLB:
		pager, err := loadbalancers.List(tagHandler.NLBClient, loadbalancers.ListOpts{
			ProjectID: tagHandler.CredentialInfo.ProjectID,
			Name:      resIID.NameId,
		}).AllPages()
		if err != nil {
			return "", err
		}
		nlbs, err := loadbalancers.ExtractLoadBalancers(pager)
		if err != nil {
			return "", err
		}
		for _, nlb := range nlbs {
			if nlb.Name == resIID.NameId {
				return nlb.ID, nil
			}
		}
		return "", errors.New("nlb not found")
	case irs.VM:
		pager, err := servers.List(tagHandler.ComputeClient, servers.ListOpts{Name: resIID.NameId}).AllPages()
		if err != nil {
			return "", err
		}
		vms, err := servers.ExtractServers(pager)
		if err != nil {
			return "", err
		}
		for _, vm := range vms {
			if vm.Name == resIID.NameId {
				return vm.ID, nil
			}
		}
		return "", errors.New("no vm found")
	default:
		return "", errors.New(string(resType) + " is not supported Resource!!")
	}
}

func handleAddTag(tagHandler *OpenStackTagHandler, resType irs.RSType, resIID irs.IID, tag irs.KeyValue) error {
	tagString, err := tagToKeyValueString(tag)
	if err != nil {
		return err
	}

	systemId, err := getResourceSystemID(tagHandler, resType, resIID)
	if err != nil {
		return err
	}

	switch resType {
	case irs.VM:
		cc := tagHandler.ComputeClient
		cc.Microversion = "2.52"
		err = computeTags.Add(cc, systemId, tagString).ExtractErr()
	case irs.VPC:
		err = networkTags.Add(tagHandler.NetworkClient, "networks", systemId, tagString).ExtractErr()
	case irs.SUBNET:
		err = networkTags.Add(tagHandler.NetworkClient, "subnets", systemId, tagString).ExtractErr()
	case irs.NLB:
		var keyValues []irs.KeyValue
		keyValues, err = handleListTag(tagHandler, resType, resIID)
		if err != nil {
			return err
		}

		var tags []string
		for _, keyValue := range keyValues {
			tags = append(tags, keyValue.Key+"="+keyValue.Value)
		}
		tags = append(tags, tag.Key+"="+tag.Value)

		_, err = loadbalancers.Update(tagHandler.NLBClient, systemId, loadbalancers.UpdateOpts{
			Tags: &tags,
		}).Extract()
	case irs.SG:
		err = networkTags.Add(tagHandler.NetworkClient, "security-groups", systemId, tagString).ExtractErr()
	default:
		return errors.New(string(resType) + " is not supported Resource!!")
	}

	if err != nil {
		return fmt.Errorf("Failed to add tag to "+string(resType)+" ("+systemId+") : %v\n", err)
	}

	return nil
}

func handleListTag(tagHandler *OpenStackTagHandler, resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
	var tagStrings []string
	var err error

	systemId, err := getResourceSystemID(tagHandler, resType, resIID)
	if err != nil {
		return nil, err
	}

	switch resType {
	case irs.VM:
		cc := tagHandler.ComputeClient
		cc.Microversion = "2.52"
		tagStrings, err = computeTags.List(cc, systemId).Extract()
	case irs.VPC:
		tagStrings, err = networkTags.List(tagHandler.NetworkClient, "networks", systemId).Extract()
	case irs.SUBNET:
		tagStrings, err = networkTags.List(tagHandler.NetworkClient, "subnets", systemId).Extract()
	case irs.NLB:
		nlb, err := loadbalancers.Get(tagHandler.NLBClient, systemId).Extract()
		if err != nil {
			return nil, fmt.Errorf("Failed to list tags for "+string(resType)+" ("+systemId+") : %v\n", err)
		}
		tagStrings = nlb.Tags
	case irs.SG:
		tagStrings, err = networkTags.List(tagHandler.NetworkClient, "security-groups", systemId).Extract()
	default:
		return nil, errors.New(string(resType) + " is not supported Resource!!")
	}

	if err != nil {
		return nil, fmt.Errorf("Failed to list tags for "+string(resType)+" ("+systemId+") : %v\n", err)
	}

	var keyValues []irs.KeyValue
	for _, tagString := range tagStrings {
		keyValue := strings.Split(tagString, "=")
		if len(keyValue) == 2 {
			keyValues = append(keyValues, irs.KeyValue{
				Key:   keyValue[0],
				Value: keyValue[1],
			})
		}
	}

	return keyValues, nil
}

func handleRemoveTag(tagHandler *OpenStackTagHandler, resType irs.RSType, resIID irs.IID, tagString string) error {
	var err error

	systemId, err := getResourceSystemID(tagHandler, resType, resIID)
	if err != nil {
		return err
	}

	switch resType {
	case irs.VM:
		cc := tagHandler.ComputeClient
		cc.Microversion = "2.52"
		err = computeTags.Delete(cc, systemId, tagString).ExtractErr()
	case irs.VPC:
		err = networkTags.Delete(tagHandler.NetworkClient, "networks", systemId, tagString).ExtractErr()
	case irs.SUBNET:
		err = networkTags.Delete(tagHandler.NetworkClient, "subnets", systemId, tagString).ExtractErr()
	case irs.NLB:
		nlb, err := loadbalancers.Get(tagHandler.NLBClient, systemId).Extract()
		if err != nil {
			return fmt.Errorf("Failed to remove tag from "+string(resType)+" ("+systemId+") : %v\n", err)
		}

		var tags []string
		for _, tag := range nlb.Tags {
			if tag == tagString {
				continue
			}
			tags = append(tags, tag)
		}

		_, err = loadbalancers.Update(tagHandler.NLBClient, systemId, loadbalancers.UpdateOpts{
			Tags: &tags,
		}).Extract()
	case irs.SG:
		err = networkTags.Delete(tagHandler.NetworkClient, "security-groups", systemId, tagString).ExtractErr()
	default:
		return errors.New(string(resType) + " is not supported Resource!!")
	}

	if err != nil {
		return fmt.Errorf("Failed to remove tag from "+string(resType)+" ("+systemId+") : %v\n", err)
	}

	return nil
}

func (tagHandler *OpenStackTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.IdentityClient.IdentityEndpoint, call.TAG, resIID.NameId, "AddTag()")
	start := call.Start()

	_, err := tagHandler.GetTag(resType, resIID, tag.Key)
	if err == nil {
		getErr := errors.New("duplicated tag key found")
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.KeyValue{}, getErr
	}
	err = handleAddTag(tagHandler, resType, resIID, tag)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to add tag. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.KeyValue{}, getErr
	}

	LoggingInfo(hiscallInfo, start)

	return tag, nil
}

func (tagHandler *OpenStackTagHandler) ListTag(resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.IdentityClient.IdentityEndpoint, call.TAG, resIID.NameId, "ListTag()")
	start := call.Start()

	keyValues, err := handleListTag(tagHandler, resType, resIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to list tags. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return []irs.KeyValue{}, getErr
	}

	LoggingInfo(hiscallInfo, start)

	return keyValues, nil
}

func (tagHandler *OpenStackTagHandler) GetTag(resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.IdentityClient.IdentityEndpoint, call.TAG, resIID.NameId, "GetTag()")
	start := call.Start()

	keyValues, err := handleListTag(tagHandler, resType, resIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get tag. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.KeyValue{}, getErr
	}

	LoggingInfo(hiscallInfo, start)

	for _, keyValue := range keyValues {
		if keyValue.Key == key {
			return keyValue, nil
		}
	}

	return irs.KeyValue{}, errors.New("tag not found")
}

func (tagHandler *OpenStackTagHandler) RemoveTag(resType irs.RSType, resIID irs.IID, key string) (bool, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.IdentityClient.IdentityEndpoint, call.TAG, resIID.NameId, "RemoveTag()")
	start := call.Start()

	keyValues, err := handleListTag(tagHandler, resType, resIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to delete tag. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}

	for _, keyValue := range keyValues {
		if keyValue.Key == key {
			err := handleRemoveTag(tagHandler, resType, resIID, keyValue.Key+"="+keyValue.Value)
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to delete tag. err = %s", err))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return false, getErr
			}

			return true, nil
		}
	}

	LoggingInfo(hiscallInfo, start)

	return false, errors.New("tag not found")
}

func getTagInfo(resType irs.RSType, resName string, resId string, tag string, allTags []string, keyword string) *irs.TagInfo {
	if strings.Contains(tag, keyword) {
		var keyValues []irs.KeyValue
		for _, tagString := range allTags {
			keyValue := strings.Split(tagString, "=")
			if len(keyValue) == 2 {
				keyValues = append(keyValues, irs.KeyValue{
					Key:   keyValue[0],
					Value: keyValue[1],
				})
			}
		}

		return &irs.TagInfo{
			ResType: resType,
			ResIId: irs.IID{
				NameId:   resName,
				SystemId: resId,
			},
			TagList:      keyValues,
			KeyValueList: []irs.KeyValue{}, // reserved for optional usage
		}
	}

	return nil
}

func (tagHandler *OpenStackTagHandler) FindTag(resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	hiscallInfo := GetCallLogScheme(tagHandler.IdentityClient.IdentityEndpoint, call.TAG, keyword, "ListTag()")
	start := call.Start()
	defer func() {
		LoggingInfo(hiscallInfo, start)
	}()

	var tagInfos []*irs.TagInfo

	switch resType {
	case irs.ALL:
		fallthrough
	case irs.VPC:
		pager, err := networks.List(tagHandler.NetworkClient, nil).AllPages()
		if err != nil {
			return nil, err
		}
		var vpcList []NetworkWithExt
		err = networks.ExtractNetworksInto(pager, &vpcList)
		if err != nil {
			return nil, err
		}
		for _, vpc := range vpcList {
			for _, tag := range vpc.Tags {
				tagInfo := getTagInfo(irs.VPC, vpc.Name, vpc.ID, tag, vpc.Tags, keyword)
				if tagInfo != nil {
					tagInfos = append(tagInfos, tagInfo)
					break
				}
			}
		}

		if resType != irs.ALL {
			if len(tagInfos) > 0 {
				return tagInfos, nil
			}
			return nil, errors.New("keyword not found for vpc")
		}

		fallthrough
	case irs.SUBNET:
		pager, err := subnets.List(tagHandler.NetworkClient, nil).AllPages()
		if err != nil {
			return nil, err
		}
		ss, err := subnets.ExtractSubnets(pager)
		if err != nil {
			return nil, err
		}
		for _, subnet := range ss {
			for _, tag := range subnet.Tags {
				tagInfo := getTagInfo(irs.SUBNET, subnet.Name, subnet.ID, tag, subnet.Tags, keyword)
				if tagInfo != nil {
					tagInfos = append(tagInfos, tagInfo)
					break
				}
			}
		}

		if resType != irs.ALL {
			if len(tagInfos) > 0 {
				return tagInfos, nil
			}
			return nil, errors.New("keyword not found for subnet")
		}

		fallthrough
	case irs.SG:
		pager, err := secgroups.List(tagHandler.ComputeClient).AllPages()
		if err != nil {
			return nil, err
		}
		sgs, err := secgroups.ExtractSecurityGroups(pager)
		if err != nil {
			return nil, err
		}
		for _, sg := range sgs {
			keyValues, err := handleListTag(tagHandler, irs.SG, irs.IID{SystemId: sg.ID})
			if err != nil {
				return nil, err
			}

			var tags []string
			for _, keyValue := range keyValues {
				tags = append(tags, keyValue.Key+"="+keyValue.Value)
			}

			for _, tag := range tags {
				tagInfo := getTagInfo(irs.SG, sg.Name, sg.ID, tag, tags, keyword)
				if tagInfo != nil {
					tagInfos = append(tagInfos, tagInfo)
					break
				}
			}
		}

		if resType != irs.ALL {
			if len(tagInfos) > 0 {
				return tagInfos, nil
			}
			return nil, errors.New("keyword not found for security group")
		}

		fallthrough
	case irs.NLB:
		pager, err := loadbalancers.List(tagHandler.NLBClient, nil).AllPages()
		if err != nil {
			return nil, err
		}
		nlbs, err := loadbalancers.ExtractLoadBalancers(pager)
		if err != nil {
			return nil, err
		}
		for _, nlb := range nlbs {
			for _, tag := range nlb.Tags {
				tagInfo := getTagInfo(irs.NLB, nlb.Name, nlb.ID, tag, nlb.Tags, keyword)
				if tagInfo != nil {
					tagInfos = append(tagInfos, tagInfo)
					break
				}
			}
		}

		if resType != irs.ALL {
			if len(tagInfos) > 0 {
				return tagInfos, nil
			}
			return nil, errors.New("keyword not found for nlb")
		}

		fallthrough
	case irs.VM:
		pager, err := servers.List(tagHandler.ComputeClient, nil).AllPages()
		if err != nil {
			return nil, err
		}
		vms, err := servers.ExtractServers(pager)
		for _, vm := range vms {
			if vm.Tags != nil {
				for _, tag := range *vm.Tags {
					tagInfo := getTagInfo(irs.VM, vm.Name, vm.ID, tag, *vm.Tags, keyword)
					if tagInfo != nil {
						tagInfos = append(tagInfos, tagInfo)
						break
					}
				}
			}
		}

		if resType != irs.ALL {
			if len(tagInfos) > 0 {
				return tagInfos, nil
			}
			return nil, errors.New("keyword not found for vm")
		}

		fallthrough
	default:
		if resType == irs.ALL {
			if len(tagInfos) > 0 {
				return tagInfos, nil
			}
			return nil, errors.New("keyword not found for all resources")
		}

		return nil, errors.New(string(resType) + " is not supported Resource!!")
	}
}
