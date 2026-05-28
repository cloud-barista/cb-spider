package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
)

type OracleTagHandler struct {
	Region        idrv.RegionInfo
	CompartmentID string
	ComputeClient core.ComputeClient
	NetworkClient core.VirtualNetworkClient
	Ctx           context.Context
}

func (handler *OracleTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	if tag.Key == "" {
		return irs.KeyValue{}, errors.New("tag key is empty")
	}
	tags, err := handler.getFreeformTags(resType, resIID)
	if err != nil {
		return irs.KeyValue{}, err
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	tags[tag.Key] = tag.Value
	if err := handler.updateFreeformTags(resType, resIID, tags); err != nil {
		return irs.KeyValue{}, err
	}
	return tag, nil
}

func (handler *OracleTagHandler) ListTag(resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
	tags, err := handler.getFreeformTags(resType, resIID)
	if err != nil {
		return nil, err
	}
	return tagList(tags), nil
}

func (handler *OracleTagHandler) GetTag(resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	tags, err := handler.getFreeformTags(resType, resIID)
	if err != nil {
		return irs.KeyValue{}, err
	}
	value, ok := tags[key]
	if !ok {
		return irs.KeyValue{}, nil
	}
	return irs.KeyValue{Key: key, Value: value}, nil
}

func (handler *OracleTagHandler) RemoveTag(resType irs.RSType, resIID irs.IID, key string) (bool, error) {
	tags, err := handler.getFreeformTags(resType, resIID)
	if err != nil {
		return false, err
	}
	if _, ok := tags[key]; !ok {
		return false, fmt.Errorf("tag key does not exist: %s", key)
	}
	delete(tags, key)
	if err := handler.updateFreeformTags(resType, resIID, tags); err != nil {
		return false, err
	}
	return true, nil
}

func (handler *OracleTagHandler) FindTag(resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	resType = irs.RSType(strings.ToLower(string(resType)))
	if resType != irs.ALL && resType != irs.VPC && resType != irs.SUBNET && resType != irs.SG && resType != irs.VM {
		return nil, fmt.Errorf("unsupported resource type for Oracle tag: %s", resType)
	}
	infos := make([]*irs.TagInfo, 0)
	if resType == irs.ALL || resType == irs.VPC {
		vpcs, err := handler.findVpcTags(keyword)
		if err != nil {
			return nil, err
		}
		infos = append(infos, vpcs...)
	}
	if resType == irs.ALL || resType == irs.SUBNET {
		subnets, err := handler.findSubnetTags(keyword)
		if err != nil {
			return nil, err
		}
		infos = append(infos, subnets...)
	}
	if resType == irs.ALL || resType == irs.SG {
		sgs, err := handler.findSecurityGroupTags(keyword)
		if err != nil {
			return nil, err
		}
		infos = append(infos, sgs...)
	}
	if resType == irs.ALL || resType == irs.VM {
		vms, err := handler.findVMTags(keyword)
		if err != nil {
			return nil, err
		}
		infos = append(infos, vms...)
	}
	return infos, nil
}

func (handler *OracleTagHandler) getFreeformTags(resType irs.RSType, resIID irs.IID) (map[string]string, error) {
	resType = irs.RSType(strings.ToLower(string(resType)))
	switch resType {
	case irs.VPC:
		vcn, err := handler.getVcn(resIID)
		if err != nil {
			return nil, err
		}
		return copyTags(vcn.FreeformTags), nil
	case irs.SUBNET:
		subnet, err := handler.getSubnet(resIID)
		if err != nil {
			return nil, err
		}
		return copyTags(subnet.FreeformTags), nil
	case irs.SG:
		nsg, err := handler.getNsg(resIID)
		if err != nil {
			return nil, err
		}
		return copyTags(nsg.FreeformTags), nil
	case irs.VM:
		instance, err := handler.getInstance(resIID)
		if err != nil {
			return nil, err
		}
		return copyTags(instance.FreeformTags), nil
	default:
		return nil, fmt.Errorf("unsupported resource type for Oracle tag: %s", resType)
	}
}

func (handler *OracleTagHandler) updateFreeformTags(resType irs.RSType, resIID irs.IID, tags map[string]string) error {
	resType = irs.RSType(strings.ToLower(string(resType)))
	switch resType {
	case irs.VPC:
		vcn, err := handler.getVcn(resIID)
		if err != nil {
			return err
		}
		_, err = handler.NetworkClient.UpdateVcn(handler.Ctx, core.UpdateVcnRequest{VcnId: vcn.Id, UpdateVcnDetails: core.UpdateVcnDetails{FreeformTags: tags}})
		if err != nil {
			return statusErr("failed to update Oracle VCN tags", err)
		}
		return nil
	case irs.SUBNET:
		subnet, err := handler.getSubnet(resIID)
		if err != nil {
			return err
		}
		_, err = handler.NetworkClient.UpdateSubnet(handler.Ctx, core.UpdateSubnetRequest{SubnetId: subnet.Id, UpdateSubnetDetails: core.UpdateSubnetDetails{FreeformTags: tags}})
		if err != nil {
			return statusErr("failed to update Oracle subnet tags", err)
		}
		return nil
	case irs.SG:
		nsg, err := handler.getNsg(resIID)
		if err != nil {
			return err
		}
		_, err = handler.NetworkClient.UpdateNetworkSecurityGroup(handler.Ctx, core.UpdateNetworkSecurityGroupRequest{NetworkSecurityGroupId: nsg.Id, UpdateNetworkSecurityGroupDetails: core.UpdateNetworkSecurityGroupDetails{FreeformTags: tags}})
		if err != nil {
			return statusErr("failed to update Oracle NSG tags", err)
		}
		return nil
	case irs.VM:
		instance, err := handler.getInstance(resIID)
		if err != nil {
			return err
		}
		_, err = handler.ComputeClient.UpdateInstance(handler.Ctx, core.UpdateInstanceRequest{InstanceId: instance.Id, UpdateInstanceDetails: core.UpdateInstanceDetails{FreeformTags: tags}})
		if err != nil {
			return statusErr("failed to update Oracle instance tags", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported resource type for Oracle tag: %s", resType)
	}
}

func (handler *OracleTagHandler) getVcn(iid irs.IID) (core.Vcn, error) {
	id, displayName := idFilter(iid)
	if id != nil {
		resp, err := handler.NetworkClient.GetVcn(handler.Ctx, core.GetVcnRequest{VcnId: id})
		if err != nil {
			return core.Vcn{}, statusErr("failed to get Oracle VCN", err)
		}
		return resp.Vcn, nil
	}
	resp, err := handler.NetworkClient.ListVcns(handler.Ctx, core.ListVcnsRequest{CompartmentId: common.String(handler.CompartmentID), DisplayName: displayName, LifecycleState: core.VcnLifecycleStateAvailable})
	if err != nil {
		return core.Vcn{}, statusErr("failed to list Oracle VCNs", err)
	}
	if len(resp.Items) == 0 {
		return core.Vcn{}, fmt.Errorf("Oracle VCN not found: %s", iid.NameId)
	}
	return resp.Items[0], nil
}

func (handler *OracleTagHandler) getSubnet(iid irs.IID) (core.Subnet, error) {
	id, displayName := idFilter(iid)
	if id != nil {
		resp, err := handler.NetworkClient.GetSubnet(handler.Ctx, core.GetSubnetRequest{SubnetId: id})
		if err != nil {
			return core.Subnet{}, statusErr("failed to get Oracle subnet", err)
		}
		return resp.Subnet, nil
	}
	resp, err := handler.NetworkClient.ListSubnets(handler.Ctx, core.ListSubnetsRequest{CompartmentId: common.String(handler.CompartmentID), DisplayName: displayName, LifecycleState: core.SubnetLifecycleStateAvailable})
	if err != nil {
		return core.Subnet{}, statusErr("failed to list Oracle subnets", err)
	}
	if len(resp.Items) == 0 {
		return core.Subnet{}, fmt.Errorf("Oracle subnet not found: %s", iid.NameId)
	}
	return resp.Items[0], nil
}

func (handler *OracleTagHandler) getNsg(iid irs.IID) (core.NetworkSecurityGroup, error) {
	id, displayName := idFilter(iid)
	if id != nil {
		resp, err := handler.NetworkClient.GetNetworkSecurityGroup(handler.Ctx, core.GetNetworkSecurityGroupRequest{NetworkSecurityGroupId: id})
		if err != nil {
			return core.NetworkSecurityGroup{}, statusErr("failed to get Oracle NSG", err)
		}
		return resp.NetworkSecurityGroup, nil
	}
	resp, err := handler.NetworkClient.ListNetworkSecurityGroups(handler.Ctx, core.ListNetworkSecurityGroupsRequest{CompartmentId: common.String(handler.CompartmentID), DisplayName: displayName, LifecycleState: core.NetworkSecurityGroupLifecycleStateAvailable})
	if err != nil {
		return core.NetworkSecurityGroup{}, statusErr("failed to list Oracle NSGs", err)
	}
	if len(resp.Items) == 0 {
		return core.NetworkSecurityGroup{}, fmt.Errorf("Oracle NSG not found: %s", iid.NameId)
	}
	return resp.Items[0], nil
}

func (handler *OracleTagHandler) getInstance(iid irs.IID) (core.Instance, error) {
	id, displayName := idFilter(iid)
	if id != nil {
		resp, err := handler.ComputeClient.GetInstance(handler.Ctx, core.GetInstanceRequest{InstanceId: id})
		if err != nil {
			return core.Instance{}, statusErr("failed to get Oracle instance", err)
		}
		return resp.Instance, nil
	}
	resp, err := handler.ComputeClient.ListInstances(handler.Ctx, core.ListInstancesRequest{CompartmentId: common.String(handler.CompartmentID), DisplayName: displayName})
	if err != nil {
		return core.Instance{}, statusErr("failed to list Oracle instances", err)
	}
	for _, instance := range resp.Items {
		if instance.LifecycleState != core.InstanceLifecycleStateTerminated {
			return instance, nil
		}
	}
	return core.Instance{}, fmt.Errorf("Oracle instance not found: %s", iid.NameId)
}

func (handler *OracleTagHandler) findVpcTags(keyword string) ([]*irs.TagInfo, error) {
	resp, err := handler.NetworkClient.ListVcns(handler.Ctx, core.ListVcnsRequest{CompartmentId: common.String(handler.CompartmentID), LifecycleState: core.VcnLifecycleStateAvailable})
	if err != nil {
		return nil, statusErr("failed to list Oracle VCNs", err)
	}
	infos := make([]*irs.TagInfo, 0)
	for _, vcn := range resp.Items {
		tags := filterTags(vcn.FreeformTags, keyword)
		if len(tags) > 0 {
			infos = append(infos, &irs.TagInfo{ResType: irs.VPC, ResIId: irs.IID{NameId: stringValue(vcn.DisplayName), SystemId: stringValue(vcn.Id)}, TagList: tags, KeyValueList: tags})
		}
	}
	return infos, nil
}

func (handler *OracleTagHandler) findSubnetTags(keyword string) ([]*irs.TagInfo, error) {
	resp, err := handler.NetworkClient.ListSubnets(handler.Ctx, core.ListSubnetsRequest{CompartmentId: common.String(handler.CompartmentID), LifecycleState: core.SubnetLifecycleStateAvailable})
	if err != nil {
		return nil, statusErr("failed to list Oracle subnets", err)
	}
	infos := make([]*irs.TagInfo, 0)
	for _, subnet := range resp.Items {
		tags := filterTags(subnet.FreeformTags, keyword)
		if len(tags) > 0 {
			infos = append(infos, &irs.TagInfo{ResType: irs.SUBNET, ResIId: irs.IID{NameId: stringValue(subnet.DisplayName), SystemId: stringValue(subnet.Id)}, TagList: tags, KeyValueList: tags})
		}
	}
	return infos, nil
}

func (handler *OracleTagHandler) findSecurityGroupTags(keyword string) ([]*irs.TagInfo, error) {
	resp, err := handler.NetworkClient.ListNetworkSecurityGroups(handler.Ctx, core.ListNetworkSecurityGroupsRequest{CompartmentId: common.String(handler.CompartmentID), LifecycleState: core.NetworkSecurityGroupLifecycleStateAvailable})
	if err != nil {
		return nil, statusErr("failed to list Oracle NSGs", err)
	}
	infos := make([]*irs.TagInfo, 0)
	for _, nsg := range resp.Items {
		tags := filterTags(nsg.FreeformTags, keyword)
		if len(tags) > 0 {
			infos = append(infos, &irs.TagInfo{ResType: irs.SG, ResIId: irs.IID{NameId: stringValue(nsg.DisplayName), SystemId: stringValue(nsg.Id)}, TagList: tags, KeyValueList: tags})
		}
	}
	return infos, nil
}
func (handler *OracleTagHandler) findVMTags(keyword string) ([]*irs.TagInfo, error) {
	resp, err := handler.ComputeClient.ListInstances(handler.Ctx, core.ListInstancesRequest{CompartmentId: common.String(handler.CompartmentID)})
	if err != nil {
		return nil, statusErr("failed to list Oracle instances", err)
	}
	infos := make([]*irs.TagInfo, 0)
	for _, instance := range resp.Items {
		if instance.LifecycleState == core.InstanceLifecycleStateTerminated {
			continue
		}
		tags := filterTags(instance.FreeformTags, keyword)
		if len(tags) > 0 {
			infos = append(infos, &irs.TagInfo{ResType: irs.VM, ResIId: irs.IID{NameId: stringValue(instance.DisplayName), SystemId: stringValue(instance.Id)}, TagList: tags, KeyValueList: tags})
		}
	}
	return infos, nil
}

func copyTags(tags map[string]string) map[string]string {
	if tags == nil {
		return map[string]string{}
	}
	copyMap := make(map[string]string, len(tags))
	for key, value := range tags {
		copyMap[key] = value
	}
	return copyMap
}

func filterTags(tags map[string]string, keyword string) []irs.KeyValue {
	keyword = strings.ToLower(keyword)
	matchAll := keyword == "" || keyword == "*"
	list := make([]irs.KeyValue, 0, len(tags))
	for key, value := range tags {
		if matchAll || strings.Contains(strings.ToLower(key), keyword) || strings.Contains(strings.ToLower(value), keyword) {
			list = append(list, irs.KeyValue{Key: key, Value: value})
		}
	}
	return list
}
