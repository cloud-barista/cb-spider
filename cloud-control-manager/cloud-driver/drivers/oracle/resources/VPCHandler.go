package resources

import (
	"context"
	"errors"
	"fmt"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
)

type OracleVPCHandler struct {
	Region        idrv.RegionInfo
	CompartmentID string
	Client        core.VirtualNetworkClient
	Ctx           context.Context
}

const (
	subnetListRetryCount = 10
	resourcePollInterval = 3 * time.Second
)

func (handler *OracleVPCHandler) CreateVPC(req irs.VPCReqInfo) (irs.VPCInfo, error) {
	hiscallInfo := getCallLogScheme(handler.Region, call.VPCSUBNET, req.IId.NameId, "CreateVPC()")
	start := call.Start()
	if req.IId.NameId == "" || req.IPv4_CIDR == "" {
		err := errors.New("invalid VPC request")
		logError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	vcnResp, err := handler.Client.CreateVcn(handler.Ctx, core.CreateVcnRequest{CreateVcnDetails: core.CreateVcnDetails{CompartmentId: common.String(handler.CompartmentID), CidrBlocks: []string{req.IPv4_CIDR}, DisplayName: common.String(req.IId.NameId), DnsLabel: common.String(dnsLabel(req.IId.NameId)), FreeformTags: freeformTags(req.TagList)}})
	if err != nil {
		wrapped := statusErr("failed to create Oracle VCN", err)
		logError(hiscallInfo, wrapped)
		return irs.VPCInfo{}, wrapped
	}
	vcn := vcnResp.Vcn
	vcn, err = handler.waitVcnAvailable(stringValue(vcn.Id))
	if err != nil {
		_, _ = handler.DeleteVPC(irs.IID{SystemId: stringValue(vcnResp.Vcn.Id), NameId: stringValue(vcnResp.Vcn.DisplayName)})
		logError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	if err := handler.enableInternetGateway(vcn); err != nil {
		_, _ = handler.DeleteVPC(irs.IID{SystemId: stringValue(vcn.Id), NameId: stringValue(vcn.DisplayName)})
		logError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	for _, subnet := range req.SubnetInfoList {
		if subnet.Zone == "" {
			subnet.Zone = handler.Region.Zone
		}
		_, err := handler.AddSubnet(irs.IID{SystemId: stringValue(vcn.Id), NameId: stringValue(vcn.DisplayName)}, subnet)
		if err != nil {
			_, _ = handler.DeleteVPC(irs.IID{SystemId: stringValue(vcn.Id), NameId: stringValue(vcn.DisplayName)})
			logError(hiscallInfo, err)
			return irs.VPCInfo{}, err
		}
	}
	info, err := handler.GetVPC(irs.IID{SystemId: stringValue(vcn.Id), NameId: stringValue(vcn.DisplayName)})
	if err != nil {
		logError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	logInfo(hiscallInfo, start)
	return info, nil
}

func (handler *OracleVPCHandler) enableInternetGateway(vcn core.Vcn) error {
	if vcn.Id == nil || vcn.DefaultRouteTableId == nil {
		return nil
	}
	igwResp, err := handler.Client.CreateInternetGateway(handler.Ctx, core.CreateInternetGatewayRequest{CreateInternetGatewayDetails: core.CreateInternetGatewayDetails{CompartmentId: common.String(handler.CompartmentID), VcnId: vcn.Id, IsEnabled: common.Bool(true), DisplayName: common.String(stringValue(vcn.DisplayName) + "-igw")}})
	if err != nil {
		return statusErr("failed to create Oracle internet gateway", err)
	}
	if err := handler.waitInternetGatewayAvailable(stringValue(igwResp.InternetGateway.Id)); err != nil {
		return err
	}
	_, err = handler.Client.UpdateRouteTable(handler.Ctx, core.UpdateRouteTableRequest{RtId: vcn.DefaultRouteTableId, UpdateRouteTableDetails: core.UpdateRouteTableDetails{RouteRules: []core.RouteRule{{NetworkEntityId: igwResp.InternetGateway.Id, Destination: common.String("0.0.0.0/0"), DestinationType: core.RouteRuleDestinationTypeCidrBlock}}}})
	if err != nil {
		return statusErr("failed to update Oracle route table", err)
	}
	return nil
}

func (handler *OracleVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	hiscallInfo := getCallLogScheme(handler.Region, call.VPCSUBNET, "VPC", "ListVPC()")
	start := call.Start()
	var infos []*irs.VPCInfo
	page := ""
	for {
		req := core.ListVcnsRequest{CompartmentId: common.String(handler.CompartmentID), LifecycleState: core.VcnLifecycleStateAvailable}
		if page != "" {
			req.Page = common.String(page)
		}
		resp, err := handler.Client.ListVcns(handler.Ctx, req)
		if err != nil {
			wrapped := statusErr("failed to list Oracle VCNs", err)
			logError(hiscallInfo, wrapped)
			return nil, wrapped
		}
		for _, vcn := range resp.Items {
			info, err := handler.vpcInfo(vcn)
			if err == nil {
				infos = append(infos, &info)
			}
		}
		if resp.OpcNextPage == nil || *resp.OpcNextPage == "" {
			break
		}
		page = *resp.OpcNextPage
	}
	logInfo(hiscallInfo, start)
	return infos, nil
}

func (handler *OracleVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	vcn, err := handler.getVcn(vpcIID)
	if err != nil {
		return irs.VPCInfo{}, err
	}
	return handler.vpcInfo(vcn)
}

func (handler *OracleVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	hiscallInfo := getCallLogScheme(handler.Region, call.VPCSUBNET, vpcIID.NameId, "DeleteVPC()")
	start := call.Start()
	vcn, err := handler.getVcn(vpcIID)
	if err != nil {
		logError(hiscallInfo, err)
		return false, err
	}
	for _, subnet := range handler.listSubnetsForDelete(stringValue(vcn.Id)) {
		_, err := handler.Client.DeleteSubnet(handler.Ctx, core.DeleteSubnetRequest{SubnetId: subnet.Id})
		if err != nil && !isNotFound(err) {
			wrapped := statusErr("failed to delete Oracle subnet", err)
			logError(hiscallInfo, wrapped)
			return false, wrapped
		}
		if err := handler.waitSubnetDeleted(stringValue(subnet.Id)); err != nil {
			logError(hiscallInfo, err)
			return false, err
		}
	}
	for _, nsg := range handler.listNsgs(stringValue(vcn.Id)) {
		_, err := handler.Client.DeleteNetworkSecurityGroup(handler.Ctx, core.DeleteNetworkSecurityGroupRequest{NetworkSecurityGroupId: nsg.Id})
		if err != nil && !isNotFound(err) {
			wrapped := statusErr("failed to delete Oracle network security group", err)
			logError(hiscallInfo, wrapped)
			return false, wrapped
		}
		if err := handler.waitNetworkSecurityGroupDeleted(stringValue(nsg.Id)); err != nil {
			logError(hiscallInfo, err)
			return false, err
		}
	}
	if err := handler.removeInternetGatewayRoutes(vcn); err != nil {
		logError(hiscallInfo, err)
		return false, err
	}
	for _, igw := range handler.listInternetGateways(stringValue(vcn.Id)) {
		_, err := handler.Client.DeleteInternetGateway(handler.Ctx, core.DeleteInternetGatewayRequest{IgId: igw.Id})
		if err != nil && !isNotFound(err) {
			wrapped := statusErr("failed to delete Oracle internet gateway", err)
			logError(hiscallInfo, wrapped)
			return false, wrapped
		}
		if err := handler.waitInternetGatewayDeleted(stringValue(igw.Id)); err != nil {
			logError(hiscallInfo, err)
			return false, err
		}
	}
	_, err = handler.Client.DeleteVcn(handler.Ctx, core.DeleteVcnRequest{VcnId: vcn.Id})
	if err != nil && !isNotFound(err) {
		wrapped := statusErr("failed to delete Oracle VCN", err)
		logError(hiscallInfo, wrapped)
		return false, wrapped
	}
	if err := handler.waitVcnDeleted(stringValue(vcn.Id)); err != nil {
		logError(hiscallInfo, err)
		return false, err
	}
	logInfo(hiscallInfo, start)
	return true, nil
}

func (handler *OracleVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	vcn, err := handler.getVcn(vpcIID)
	if err != nil {
		return irs.VPCInfo{}, err
	}
	if subnetInfo.IId.NameId == "" || subnetInfo.IPv4_CIDR == "" {
		return irs.VPCInfo{}, errors.New("invalid subnet request")
	}
	zone := subnetInfo.Zone
	if zone == "" {
		zone = handler.Region.Zone
	}
	resp, err := handler.Client.CreateSubnet(handler.Ctx, core.CreateSubnetRequest{CreateSubnetDetails: core.CreateSubnetDetails{CompartmentId: common.String(handler.CompartmentID), VcnId: vcn.Id, CidrBlock: common.String(subnetInfo.IPv4_CIDR), AvailabilityDomain: common.String(zone), DisplayName: common.String(subnetInfo.IId.NameId), DnsLabel: common.String(dnsLabel(subnetInfo.IId.NameId)), ProhibitPublicIpOnVnic: common.Bool(false), FreeformTags: freeformTags(subnetInfo.TagList)}})
	if err != nil {
		return irs.VPCInfo{}, statusErr("failed to create Oracle subnet", err)
	}
	if err := handler.waitSubnetAvailable(stringValue(resp.Subnet.Id)); err != nil {
		return irs.VPCInfo{}, err
	}
	return handler.GetVPC(irs.IID{SystemId: stringValue(vcn.Id), NameId: stringValue(vcn.DisplayName)})
}

func (handler *OracleVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	subnet, err := handler.getSubnet(subnetIID, vpcIID.SystemId)
	if err != nil {
		return false, err
	}
	_, err = handler.Client.DeleteSubnet(handler.Ctx, core.DeleteSubnetRequest{SubnetId: subnet.Id})
	if err != nil && !isNotFound(err) {
		return false, statusErr("failed to delete Oracle subnet", err)
	}
	if err := handler.waitSubnetDeleted(stringValue(subnet.Id)); err != nil {
		return false, err
	}
	return true, nil
}

func (handler *OracleVPCHandler) ListIID() ([]*irs.IID, error) {
	infos, err := handler.ListVPC()
	if err != nil {
		return nil, err
	}
	iids := make([]*irs.IID, 0, len(infos))
	for _, info := range infos {
		iids = append(iids, &info.IId)
	}
	return iids, nil
}

func (handler *OracleVPCHandler) getVcn(iid irs.IID) (core.Vcn, error) {
	id, displayName := idFilter(iid)
	if id != nil {
		resp, err := handler.Client.GetVcn(handler.Ctx, core.GetVcnRequest{VcnId: id})
		return resp.Vcn, err
	}
	resp, err := handler.Client.ListVcns(handler.Ctx, core.ListVcnsRequest{CompartmentId: common.String(handler.CompartmentID), DisplayName: displayName, LifecycleState: core.VcnLifecycleStateAvailable})
	if err != nil {
		return core.Vcn{}, err
	}
	if len(resp.Items) == 0 {
		return core.Vcn{}, fmt.Errorf("Oracle VCN not found: %s", iid.NameId)
	}
	return resp.Items[0], nil
}

func (handler *OracleVPCHandler) vpcInfo(vcn core.Vcn) (irs.VPCInfo, error) {
	subnets := handler.listSubnetsWithRetry(stringValue(vcn.Id))
	subnetInfos := make([]irs.SubnetInfo, 0, len(subnets))
	for _, subnet := range subnets {
		subnetInfos = append(subnetInfos, irs.SubnetInfo{IId: irs.IID{NameId: stringValue(subnet.DisplayName), SystemId: stringValue(subnet.Id)}, Zone: stringValue(subnet.AvailabilityDomain), IPv4_CIDR: stringValue(subnet.CidrBlock), TagList: tagList(subnet.FreeformTags)})
	}
	cidr := stringValue(vcn.CidrBlock)
	if cidr == "" && len(vcn.CidrBlocks) > 0 {
		cidr = vcn.CidrBlocks[0]
	}
	return irs.VPCInfo{IId: irs.IID{NameId: stringValue(vcn.DisplayName), SystemId: stringValue(vcn.Id)}, IPv4_CIDR: cidr, SubnetInfoList: subnetInfos, TagList: tagList(vcn.FreeformTags)}, nil
}

func (handler *OracleVPCHandler) listSubnets(vcnID string) []core.Subnet {
	resp, err := handler.Client.ListSubnets(handler.Ctx, core.ListSubnetsRequest{CompartmentId: common.String(handler.CompartmentID), VcnId: common.String(vcnID), LifecycleState: core.SubnetLifecycleStateAvailable})
	if err != nil {
		if cblogger != nil {
			cblogger.Warnf("failed to list Oracle subnets for VCN %s: %v", vcnID, err)
		}
		return nil
	}
	return resp.Items
}

func (handler *OracleVPCHandler) listSubnetsForDelete(vcnID string) []core.Subnet {
	resp, err := handler.Client.ListSubnets(handler.Ctx, core.ListSubnetsRequest{CompartmentId: common.String(handler.CompartmentID), VcnId: common.String(vcnID)})
	if err != nil {
		if cblogger != nil {
			cblogger.Warnf("failed to list Oracle subnets for VCN %s: %v", vcnID, err)
		}
		return nil
	}
	subnets := make([]core.Subnet, 0, len(resp.Items))
	for _, subnet := range resp.Items {
		if subnet.LifecycleState != core.SubnetLifecycleStateTerminated {
			subnets = append(subnets, subnet)
		}
	}
	return subnets
}

func (handler *OracleVPCHandler) listSubnetsWithRetry(vcnID string) []core.Subnet {
	for attempt := 0; attempt < subnetListRetryCount; attempt++ {
		subnets := handler.listSubnets(vcnID)
		if len(subnets) > 0 {
			return subnets
		}
		select {
		case <-handler.Ctx.Done():
			return nil
		case <-time.After(time.Second):
		}
	}
	return handler.listSubnets(vcnID)
}

func (handler *OracleVPCHandler) waitVcnAvailable(vcnID string) (core.Vcn, error) {
	for {
		resp, err := handler.Client.GetVcn(handler.Ctx, core.GetVcnRequest{VcnId: common.String(vcnID)})
		if err != nil {
			return core.Vcn{}, statusErr("failed to get Oracle VCN", err)
		}
		if resp.Vcn.LifecycleState == core.VcnLifecycleStateAvailable {
			return resp.Vcn, nil
		}
		if resp.Vcn.LifecycleState == core.VcnLifecycleStateTerminated || resp.Vcn.LifecycleState == core.VcnLifecycleStateTerminating {
			return core.Vcn{}, fmt.Errorf("Oracle VCN %s entered unexpected state %s", vcnID, resp.Vcn.LifecycleState)
		}
		if err := handler.waitNextPoll(); err != nil {
			return core.Vcn{}, err
		}
	}
}

func (handler *OracleVPCHandler) waitVcnDeleted(vcnID string) error {
	for {
		resp, err := handler.Client.GetVcn(handler.Ctx, core.GetVcnRequest{VcnId: common.String(vcnID)})
		if isNotFound(err) {
			return nil
		}
		if err != nil {
			return statusErr("failed to get Oracle VCN", err)
		}
		if resp.Vcn.LifecycleState == core.VcnLifecycleStateTerminated {
			return nil
		}
		if err := handler.waitNextPoll(); err != nil {
			return err
		}
	}
}

func (handler *OracleVPCHandler) waitSubnetAvailable(subnetID string) error {
	for {
		resp, err := handler.Client.GetSubnet(handler.Ctx, core.GetSubnetRequest{SubnetId: common.String(subnetID)})
		if err != nil {
			return statusErr("failed to get Oracle subnet", err)
		}
		if resp.Subnet.LifecycleState == core.SubnetLifecycleStateAvailable {
			return nil
		}
		if resp.Subnet.LifecycleState == core.SubnetLifecycleStateTerminated || resp.Subnet.LifecycleState == core.SubnetLifecycleStateTerminating {
			return fmt.Errorf("Oracle subnet %s entered unexpected state %s", subnetID, resp.Subnet.LifecycleState)
		}
		if err := handler.waitNextPoll(); err != nil {
			return err
		}
	}
}

func (handler *OracleVPCHandler) waitSubnetDeleted(subnetID string) error {
	for {
		resp, err := handler.Client.GetSubnet(handler.Ctx, core.GetSubnetRequest{SubnetId: common.String(subnetID)})
		if isNotFound(err) {
			return nil
		}
		if err != nil {
			return statusErr("failed to get Oracle subnet", err)
		}
		if resp.Subnet.LifecycleState == core.SubnetLifecycleStateTerminated {
			return nil
		}
		if err := handler.waitNextPoll(); err != nil {
			return err
		}
	}
}

func (handler *OracleVPCHandler) waitInternetGatewayAvailable(igwID string) error {
	for {
		resp, err := handler.Client.GetInternetGateway(handler.Ctx, core.GetInternetGatewayRequest{IgId: common.String(igwID)})
		if err != nil {
			return statusErr("failed to get Oracle internet gateway", err)
		}
		if resp.InternetGateway.LifecycleState == core.InternetGatewayLifecycleStateAvailable {
			return nil
		}
		if resp.InternetGateway.LifecycleState == core.InternetGatewayLifecycleStateTerminated || resp.InternetGateway.LifecycleState == core.InternetGatewayLifecycleStateTerminating {
			return fmt.Errorf("Oracle internet gateway %s entered unexpected state %s", igwID, resp.InternetGateway.LifecycleState)
		}
		if err := handler.waitNextPoll(); err != nil {
			return err
		}
	}
}

func (handler *OracleVPCHandler) waitInternetGatewayDeleted(igwID string) error {
	for {
		resp, err := handler.Client.GetInternetGateway(handler.Ctx, core.GetInternetGatewayRequest{IgId: common.String(igwID)})
		if isNotFound(err) {
			return nil
		}
		if err != nil {
			return statusErr("failed to get Oracle internet gateway", err)
		}
		if resp.InternetGateway.LifecycleState == core.InternetGatewayLifecycleStateTerminated {
			return nil
		}
		if err := handler.waitNextPoll(); err != nil {
			return err
		}
	}
}

func (handler *OracleVPCHandler) waitNetworkSecurityGroupDeleted(nsgID string) error {
	for {
		resp, err := handler.Client.GetNetworkSecurityGroup(handler.Ctx, core.GetNetworkSecurityGroupRequest{NetworkSecurityGroupId: common.String(nsgID)})
		if isNotFound(err) {
			return nil
		}
		if err != nil {
			return statusErr("failed to get Oracle network security group", err)
		}
		if resp.NetworkSecurityGroup.LifecycleState == core.NetworkSecurityGroupLifecycleStateTerminated {
			return nil
		}
		if err := handler.waitNextPoll(); err != nil {
			return err
		}
	}
}

func (handler *OracleVPCHandler) removeInternetGatewayRoutes(vcn core.Vcn) error {
	if vcn.DefaultRouteTableId == nil {
		return nil
	}
	resp, err := handler.Client.GetRouteTable(handler.Ctx, core.GetRouteTableRequest{RtId: vcn.DefaultRouteTableId})
	if isNotFound(err) {
		return nil
	}
	if err != nil {
		return statusErr("failed to get Oracle route table", err)
	}
	rules := make([]core.RouteRule, 0, len(resp.RouteTable.RouteRules))
	changed := false
	for _, rule := range resp.RouteTable.RouteRules {
		if stringValue(rule.Destination) == "0.0.0.0/0" && rule.DestinationType == core.RouteRuleDestinationTypeCidrBlock && handler.isInternetGatewayRoute(rule.NetworkEntityId) {
			changed = true
			continue
		}
		rules = append(rules, rule)
	}
	if !changed {
		return nil
	}
	_, err = handler.Client.UpdateRouteTable(handler.Ctx, core.UpdateRouteTableRequest{RtId: vcn.DefaultRouteTableId, UpdateRouteTableDetails: core.UpdateRouteTableDetails{RouteRules: rules}})
	if err != nil {
		return statusErr("failed to update Oracle route table", err)
	}
	return nil
}

func (handler *OracleVPCHandler) isInternetGatewayRoute(networkEntityID *string) bool {
	if networkEntityID == nil || *networkEntityID == "" {
		return false
	}
	_, err := handler.Client.GetInternetGateway(handler.Ctx, core.GetInternetGatewayRequest{IgId: networkEntityID})
	return err == nil
}

func (handler *OracleVPCHandler) waitNextPoll() error {
	select {
	case <-handler.Ctx.Done():
		return handler.Ctx.Err()
	case <-time.After(resourcePollInterval):
		return nil
	}
}

func (handler *OracleVPCHandler) listInternetGateways(vcnID string) []core.InternetGateway {
	resp, err := handler.Client.ListInternetGateways(handler.Ctx, core.ListInternetGatewaysRequest{CompartmentId: common.String(handler.CompartmentID), VcnId: common.String(vcnID)})
	if err != nil {
		return nil
	}
	igws := make([]core.InternetGateway, 0, len(resp.Items))
	for _, igw := range resp.Items {
		if igw.LifecycleState != core.InternetGatewayLifecycleStateTerminated {
			igws = append(igws, igw)
		}
	}
	return igws
}

func (handler *OracleVPCHandler) listNsgs(vcnID string) []core.NetworkSecurityGroup {
	resp, err := handler.Client.ListNetworkSecurityGroups(handler.Ctx, core.ListNetworkSecurityGroupsRequest{CompartmentId: common.String(handler.CompartmentID), VcnId: common.String(vcnID)})
	if err != nil {
		return nil
	}
	nsgs := make([]core.NetworkSecurityGroup, 0, len(resp.Items))
	for _, nsg := range resp.Items {
		if nsg.LifecycleState != core.NetworkSecurityGroupLifecycleStateTerminated {
			nsgs = append(nsgs, nsg)
		}
	}
	return nsgs
}

func (handler *OracleVPCHandler) getSubnet(iid irs.IID, vcnID string) (core.Subnet, error) {
	id, displayName := idFilter(iid)
	if id != nil {
		resp, err := handler.Client.GetSubnet(handler.Ctx, core.GetSubnetRequest{SubnetId: id})
		return resp.Subnet, err
	}
	resp, err := handler.Client.ListSubnets(handler.Ctx, core.ListSubnetsRequest{CompartmentId: common.String(handler.CompartmentID), VcnId: common.String(vcnID), DisplayName: displayName, LifecycleState: core.SubnetLifecycleStateAvailable})
	if err != nil {
		return core.Subnet{}, err
	}
	if len(resp.Items) == 0 {
		return core.Subnet{}, fmt.Errorf("Oracle subnet not found: %s", iid.NameId)
	}
	return resp.Items[0], nil
}
