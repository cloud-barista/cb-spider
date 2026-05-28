package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
)

type OracleSecurityHandler struct {
	Region        idrv.RegionInfo
	CompartmentID string
	Client        core.VirtualNetworkClient
	Ctx           context.Context
}

func (handler *OracleSecurityHandler) CreateSecurity(req irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	if req.IId.NameId == "" || req.VpcIID.SystemId == "" {
		return irs.SecurityInfo{}, errors.New("invalid security group request")
	}
	resp, err := handler.Client.CreateNetworkSecurityGroup(handler.Ctx, core.CreateNetworkSecurityGroupRequest{CreateNetworkSecurityGroupDetails: core.CreateNetworkSecurityGroupDetails{CompartmentId: common.String(handler.CompartmentID), VcnId: common.String(req.VpcIID.SystemId), DisplayName: common.String(req.IId.NameId), FreeformTags: freeformTags(req.TagList)}})
	if err != nil {
		return irs.SecurityInfo{}, statusErr("failed to create Oracle NSG", err)
	}
	iid := irs.IID{NameId: stringValue(resp.NetworkSecurityGroup.DisplayName), SystemId: stringValue(resp.NetworkSecurityGroup.Id)}
	if req.SecurityRules != nil && len(*req.SecurityRules) > 0 {
		if _, err := handler.AddRules(iid, req.SecurityRules); err != nil {
			_, _ = handler.DeleteSecurity(iid)
			return irs.SecurityInfo{}, err
		}
	}
	return handler.GetSecurity(iid)
}

func (handler *OracleSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	resp, err := handler.Client.ListNetworkSecurityGroups(handler.Ctx, core.ListNetworkSecurityGroupsRequest{CompartmentId: common.String(handler.CompartmentID), LifecycleState: core.NetworkSecurityGroupLifecycleStateAvailable})
	if err != nil {
		return nil, statusErr("failed to list Oracle NSGs", err)
	}
	infos := make([]*irs.SecurityInfo, 0, len(resp.Items))
	for _, nsg := range resp.Items {
		info, err := handler.securityInfo(nsg)
		if err == nil {
			infos = append(infos, &info)
		}
	}
	return infos, nil
}

func (handler *OracleSecurityHandler) GetSecurity(iid irs.IID) (irs.SecurityInfo, error) {
	nsg, err := handler.getNsg(iid, "")
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	return handler.securityInfo(nsg)
}

func (handler *OracleSecurityHandler) DeleteSecurity(iid irs.IID) (bool, error) {
	nsg, err := handler.getNsg(iid, "")
	if err != nil {
		return false, err
	}
	_, err = handler.Client.DeleteNetworkSecurityGroup(handler.Ctx, core.DeleteNetworkSecurityGroupRequest{NetworkSecurityGroupId: nsg.Id})
	if err != nil && !isNotFound(err) {
		return false, statusErr("failed to delete Oracle NSG", err)
	}
	return true, nil
}

func (handler *OracleSecurityHandler) AddRules(sgIID irs.IID, rules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	nsg, err := handler.getNsg(sgIID, "")
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	ociRules := make([]core.AddSecurityRuleDetails, 0, len(*rules))
	for _, rule := range *rules {
		ociRule, err := toOCIAddRule(rule)
		if err != nil {
			return irs.SecurityInfo{}, err
		}
		ociRules = append(ociRules, ociRule)
	}
	_, err = handler.Client.AddNetworkSecurityGroupSecurityRules(handler.Ctx, core.AddNetworkSecurityGroupSecurityRulesRequest{NetworkSecurityGroupId: nsg.Id, AddNetworkSecurityGroupSecurityRulesDetails: core.AddNetworkSecurityGroupSecurityRulesDetails{SecurityRules: ociRules}})
	if err != nil {
		return irs.SecurityInfo{}, statusErr("failed to add Oracle NSG rules", err)
	}
	return handler.GetSecurity(sgIID)
}

func (handler *OracleSecurityHandler) RemoveRules(sgIID irs.IID, rules *[]irs.SecurityRuleInfo) (bool, error) {
	nsg, err := handler.getNsg(sgIID, "")
	if err != nil {
		return false, err
	}
	existing, err := handler.listSecurityRules(stringValue(nsg.Id))
	if err != nil {
		return false, err
	}
	ids := make([]string, 0)
	for _, target := range *rules {
		for _, rule := range existing {
			if securityRuleMatches(rule, target) && rule.Id != nil {
				ids = append(ids, *rule.Id)
			}
		}
	}
	if len(ids) == 0 {
		return true, nil
	}
	_, err = handler.Client.RemoveNetworkSecurityGroupSecurityRules(handler.Ctx, core.RemoveNetworkSecurityGroupSecurityRulesRequest{NetworkSecurityGroupId: nsg.Id, RemoveNetworkSecurityGroupSecurityRulesDetails: core.RemoveNetworkSecurityGroupSecurityRulesDetails{SecurityRuleIds: ids}})
	if err != nil {
		return false, statusErr("failed to remove Oracle NSG rules", err)
	}
	return true, nil
}

func (handler *OracleSecurityHandler) ListIID() ([]*irs.IID, error) {
	infos, err := handler.ListSecurity()
	if err != nil {
		return nil, err
	}
	iids := make([]*irs.IID, 0, len(infos))
	for _, info := range infos {
		iids = append(iids, &info.IId)
	}
	return iids, nil
}

func (handler *OracleSecurityHandler) getNsg(iid irs.IID, vcnID string) (core.NetworkSecurityGroup, error) {
	id, displayName := idFilter(iid)
	if id != nil {
		resp, err := handler.Client.GetNetworkSecurityGroup(handler.Ctx, core.GetNetworkSecurityGroupRequest{NetworkSecurityGroupId: id})
		return resp.NetworkSecurityGroup, err
	}
	req := core.ListNetworkSecurityGroupsRequest{CompartmentId: common.String(handler.CompartmentID), DisplayName: displayName, LifecycleState: core.NetworkSecurityGroupLifecycleStateAvailable}
	if vcnID != "" {
		req.VcnId = common.String(vcnID)
	}
	resp, err := handler.Client.ListNetworkSecurityGroups(handler.Ctx, req)
	if err != nil {
		return core.NetworkSecurityGroup{}, err
	}
	if len(resp.Items) == 0 {
		return core.NetworkSecurityGroup{}, fmt.Errorf("Oracle NSG not found: %s", iid.NameId)
	}
	return resp.Items[0], nil
}

func (handler *OracleSecurityHandler) securityInfo(nsg core.NetworkSecurityGroup) (irs.SecurityInfo, error) {
	rules, err := handler.listSecurityRules(stringValue(nsg.Id))
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	cbRules := make([]irs.SecurityRuleInfo, 0, len(rules))
	for _, rule := range rules {
		cbRules = append(cbRules, fromOCIRule(rule))
	}
	return irs.SecurityInfo{IId: irs.IID{NameId: stringValue(nsg.DisplayName), SystemId: stringValue(nsg.Id)}, VpcIID: irs.IID{SystemId: stringValue(nsg.VcnId)}, SecurityRules: &cbRules, TagList: tagList(nsg.FreeformTags)}, nil
}

func (handler *OracleSecurityHandler) listSecurityRules(nsgID string) ([]core.SecurityRule, error) {
	resp, err := handler.Client.ListNetworkSecurityGroupSecurityRules(handler.Ctx, core.ListNetworkSecurityGroupSecurityRulesRequest{NetworkSecurityGroupId: common.String(nsgID)})
	if err != nil {
		return nil, statusErr("failed to list Oracle NSG rules", err)
	}
	return resp.Items, nil
}

func toOCIAddRule(rule irs.SecurityRuleInfo) (core.AddSecurityRuleDetails, error) {
	protocol := protocolNumber(rule.IPProtocol)
	cidr := rule.CIDR
	if cidr == "" {
		cidr = "0.0.0.0/0"
	}
	ociRule := core.AddSecurityRuleDetails{Protocol: common.String(protocol), IsStateless: common.Bool(false)}
	if strings.EqualFold(rule.Direction, "outbound") || strings.EqualFold(rule.Direction, "egress") {
		ociRule.Direction = core.AddSecurityRuleDetailsDirectionEgress
		ociRule.Destination = common.String(cidr)
		ociRule.DestinationType = core.AddSecurityRuleDetailsDestinationTypeCidrBlock
	} else {
		ociRule.Direction = core.AddSecurityRuleDetailsDirectionIngress
		ociRule.Source = common.String(cidr)
		ociRule.SourceType = core.AddSecurityRuleDetailsSourceTypeCidrBlock
	}
	if protocol == "6" || protocol == "17" {
		portRange, err := portRange(rule.FromPort, rule.ToPort)
		if err != nil {
			return core.AddSecurityRuleDetails{}, err
		}
		if protocol == "6" {
			ociRule.TcpOptions = &core.TcpOptions{DestinationPortRange: portRange}
		} else {
			ociRule.UdpOptions = &core.UdpOptions{DestinationPortRange: portRange}
		}
	}
	return ociRule, nil
}

func fromOCIRule(rule core.SecurityRule) irs.SecurityRuleInfo {
	direction := "inbound"
	cidr := stringValue(rule.Source)
	if rule.Direction == core.SecurityRuleDirectionEgress {
		direction = "outbound"
		cidr = stringValue(rule.Destination)
	}
	fromPort, toPort := "-1", "-1"
	if rule.TcpOptions != nil && rule.TcpOptions.DestinationPortRange != nil {
		fromPort = strconv.Itoa(*rule.TcpOptions.DestinationPortRange.Min)
		toPort = strconv.Itoa(*rule.TcpOptions.DestinationPortRange.Max)
	}
	if rule.UdpOptions != nil && rule.UdpOptions.DestinationPortRange != nil {
		fromPort = strconv.Itoa(*rule.UdpOptions.DestinationPortRange.Min)
		toPort = strconv.Itoa(*rule.UdpOptions.DestinationPortRange.Max)
	}
	return irs.SecurityRuleInfo{Direction: direction, IPProtocol: protocolName(stringValue(rule.Protocol)), FromPort: fromPort, ToPort: toPort, CIDR: cidr}
}

func securityRuleMatches(rule core.SecurityRule, target irs.SecurityRuleInfo) bool {
	candidate := fromOCIRule(rule)
	return strings.EqualFold(candidate.Direction, target.Direction) && strings.EqualFold(candidate.IPProtocol, target.IPProtocol) && candidate.FromPort == target.FromPort && candidate.ToPort == target.ToPort && candidate.CIDR == target.CIDR
}

func protocolNumber(protocol string) string {
	switch strings.ToUpper(protocol) {
	case "TCP", "6":
		return "6"
	case "UDP", "17":
		return "17"
	case "ICMP", "1":
		return "1"
	default:
		return "all"
	}
}

func protocolName(protocol string) string {
	switch protocol {
	case "6":
		return "TCP"
	case "17":
		return "UDP"
	case "1":
		return "ICMP"
	default:
		return "ALL"
	}
}

func portRange(fromPort string, toPort string) (*core.PortRange, error) {
	if fromPort == "" || fromPort == "-1" {
		return nil, nil
	}
	from, err := strconv.Atoi(fromPort)
	if err != nil {
		return nil, err
	}
	to := from
	if toPort != "" && toPort != "-1" {
		to, err = strconv.Atoi(toPort)
		if err != nil {
			return nil, err
		}
	}
	return &core.PortRange{Min: common.Int(from), Max: common.Int(to)}, nil
}
