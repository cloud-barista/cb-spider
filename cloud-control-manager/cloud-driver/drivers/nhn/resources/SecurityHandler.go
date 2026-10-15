// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, Innogrid, 2021.12.
// by ETRI, 2022.04.

package resources

import (
	// "errors"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	// "github.com/davecgh/go-spew/spew"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/extensions/secgroups"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/extensions/security/rules"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NhnCloudSecurityHandler struct {
	RegionInfo    idrv.RegionInfo
	VMClient      *nhnsdk.ServiceClient
	NetworkClient *nhnsdk.ServiceClient
}

func (securityHandler *NhnCloudSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Info("NHN Cloud Driver: called CreateSecurity()!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Region, call.SECURITYGROUP, securityReqInfo.IId.NameId, "CreateSecurity()")

	// Check if the SecurityGroup Exists
	sgInfoList, err := securityHandler.ListSecurity()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get SG List!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

	for _, sgInfo := range sgInfoList {
		if sgInfo.IId.NameId == securityReqInfo.IId.NameId {
			newErr := fmt.Errorf("Security Group with name [%s] exists already!!", securityReqInfo.IId.NameId)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.SecurityInfo{}, newErr
		}
	}

	// Create SecurityGroup
	createOpts := secgroups.CreateOpts{
		Name:        securityReqInfo.IId.NameId,
		Description: securityReqInfo.IId.NameId,
	}
	start := call.Start()
	newSG, err := secgroups.Create(securityHandler.VMClient, createOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New S/G on NHN!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	} else {
		cblogger.Infof("Succeeded in Creating New S/G : [%s]", securityReqInfo.IId.NameId)
	}
	LoggingInfo(callLogInfo, start)
	cblogger.Infof("New S/G SystemId : [%s]", newSG.ID)

	newSGIID := irs.IID{
		SystemId: newSG.ID,
	}

	// Add Requested S/G Rules to the New S/G
	_, err = securityHandler.AddRules(newSGIID, securityReqInfo.SecurityRules)
	if err != nil {
		newErr := fmt.Errorf("Failed to Add Rule on the S/G!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

	// Basically, Open 'Outbound' All Protocol for Any S/G (<= CB-Spider Rule)
	openErr := securityHandler.openOutboundAllProtocol(newSGIID)
	if openErr != nil {
		cblogger.Error(openErr)
		LoggingError(callLogInfo, openErr)
		// return irs.SecurityInfo{}, openErr
	}

	// Return Created S/G Info.
	newSGInfo, err := securityHandler.GetSecurity(newSGIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get New S/G info!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}
	return newSGInfo, nil
}

func (securityHandler *NhnCloudSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ListSecurity()!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Region, call.SECURITYGROUP, "ListSecurity()", "ListSecurity()")

	// Get Security Group list
	start := call.Start()
	allPages, err := secgroups.List(securityHandler.VMClient).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get SG List from NhnCloud!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	nhnSGList, err := secgroups.ExtractSecurityGroups(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Extract SG List from NhnCloud!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, start)

	// Mapping S/G list info.
	var sgInfoList []*irs.SecurityInfo
	for _, nhnSG := range nhnSGList {
		sgInfo, err := securityHandler.mappingSecurityInfo(nhnSG)
		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(callLogInfo, err)
			return nil, err
		}
		sgInfoList = append(sgInfoList, sgInfo)
	}
	return sgInfoList, nil
}

func (securityHandler *NhnCloudSecurityHandler) getRawSecurity(securityIID irs.IID) (*secgroups.SecurityGroup, error) {
	if securityIID.SystemId == "" && securityIID.NameId == "" {
		return nil, errors.New("invalid IID")
	}
	if securityIID.SystemId != "" {
		return secgroups.Get(securityHandler.VMClient, securityIID.SystemId).Extract()
	} else {
		pager, err := secgroups.List(securityHandler.VMClient).AllPages()
		if err != nil {
			return nil, err
		}
		rawSecurityGroups, err := secgroups.ExtractSecurityGroups(pager)
		for _, rawSeg := range rawSecurityGroups {
			if securityIID.NameId == rawSeg.Name {
				return &rawSeg, nil
			}
		}
		return nil, errors.New("SecurityGroup not found")
	}
}

func (securityHandler *NhnCloudSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetSecurity()!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Region, call.SECURITYGROUP, securityIID.SystemId, "GetSecurity()")

	start := call.Start()
	nhnSG, err := securityHandler.getRawSecurity(securityIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the S/G info from NHN!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)
	// spew.Dump(nhnSG)

	securityInfo, err := securityHandler.mappingSecurityInfo(*nhnSG)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		return irs.SecurityInfo{}, err
	}
	return *securityInfo, nil
}

func (securityHandler *NhnCloudSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called DeleteSecurity()!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Region, call.SECURITYGROUP, securityIID.SystemId, "DeleteSecurity()")

	start := call.Start()
	nhnSG, err := securityHandler.getRawSecurity(securityIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the S/G info from NHN!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	result := secgroups.Delete(securityHandler.VMClient, nhnSG.ID)
	if result.Err != nil {
		newErr := fmt.Errorf("Failed to Delete the S/G on NHN!! : [%v] ", result.Err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, start)

	return true, nil
}

func (securityHandler *NhnCloudSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	cblogger.Info("NHN Cloud Driver: called AddRules()!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Region, call.SECURITYGROUP, sgIID.SystemId, "AddRules()")

	nhnSG, err := securityHandler.getRawSecurity(sgIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the S/G info from NHN!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

	// Add SecurityGroup Rules to the S/G
	for _, curRule := range *securityRules {
		if curRule.Direction == "" {
			return irs.SecurityInfo{}, errors.New("Failed to Find 'Direction' Value in the requested rule!!")
		} else if curRule.IPProtocol == "" {
			return irs.SecurityInfo{}, errors.New("Failed to Find 'IPProtocol' Value in the requested rule!!")
		} else if curRule.FromPort == "" {
			return irs.SecurityInfo{}, errors.New("Failed to Find 'FromPort' Value in the requested rule!!")
		} else if curRule.ToPort == "" {
			return irs.SecurityInfo{}, errors.New("Failed to Find 'ToPort' Value in the requested rule!!")
		} else if curRule.CIDR == "" {
			return irs.SecurityInfo{}, errors.New("Failed to Find 'CIDR' Value in the requested rule!!")
		}

		if strings.EqualFold(curRule.IPProtocol, "ALL") { // Add SecurityGroup Rules in case of 'All Traffic Open Rule'
			if strings.EqualFold(curRule.FromPort, "-1") && strings.EqualFold(curRule.ToPort, "-1") {
				var direction string
				if strings.EqualFold(curRule.Direction, "inbound") {
					direction = string(rules.DirIngress)
				} else if strings.EqualFold(curRule.Direction, "outbound") {
					direction = string(rules.DirEgress)
				} else {
					return irs.SecurityInfo{}, errors.New("Invalid Rule Direction!!")
				}

				etherType, err := getEtherTypeFromCIDR(curRule.CIDR)
				if err != nil {
					newErr := fmt.Errorf("Failed to Get the EtherType of the requested rule : [%v]", err)
					cblogger.Error(newErr.Error())
					LoggingError(callLogInfo, newErr)
					return irs.SecurityInfo{}, newErr
				}

				// NHN(Neutron) allows every protocol when 'protocol' is omitted, and rejects
				// a port range that comes without a protocol. So neither of them is set here.
				createRuleOpts := rules.CreateOpts{
					Direction:      rules.RuleDirection(direction),
					EtherType:      etherType,
					SecGroupID:     nhnSG.ID,
					RemoteIPPrefix: curRule.CIDR,
				}

				start := call.Start()
				_, err = rules.Create(securityHandler.NetworkClient, createRuleOpts).Extract()
				if err != nil {
					if strings.Contains(strings.ToLower(err.Error()), "already exists") {
						cblogger.Infof("Rule already exists in S/G [%s], skipping: [%v]", nhnSG.ID, err)
					} else {
						newErr := fmt.Errorf("Failed to Create New Rule to the S/G : [%s] : [%v]", nhnSG.ID, err)
						cblogger.Error(newErr.Error())
						LoggingError(callLogInfo, newErr)
						return irs.SecurityInfo{}, newErr
					}
				}
				LoggingInfo(callLogInfo, start)
				cblogger.Infof("Succeeded in Adding New [%s], [ALL] Rule!!", curRule.Direction)
			} else {
				return irs.SecurityInfo{}, errors.New("To Specify 'All Traffic Allow Rule', Specify '-1' as FromPort/ToPort!!")
			}
		} else {
			// Add SecurityGroup Rules if not 'All Traffic Open Rule'
			var direction string
			if strings.EqualFold(curRule.Direction, "inbound") {
				direction = string(rules.DirIngress)
			} else if strings.EqualFold(curRule.Direction, "outbound") {
				direction = string(rules.DirEgress)
			} else {
				return irs.SecurityInfo{}, errors.New("Invalid Rule Direction!!")
			}

			etherType, err := getEtherTypeFromCIDR(curRule.CIDR)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the EtherType of the requested rule : [%v]", err)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return irs.SecurityInfo{}, newErr
			}

			var createRuleOpts rules.CreateOpts

			if strings.EqualFold(curRule.IPProtocol, "icmp") {
				createRuleOpts = rules.CreateOpts{
					Direction:      rules.RuleDirection(direction),
					EtherType:      etherType,
					SecGroupID:     nhnSG.ID,
					Protocol:       rules.RuleProtocol(strings.ToLower(curRule.IPProtocol)),
					RemoteIPPrefix: curRule.CIDR,
				}
			} else {
				fromPortStr, toPortStr := normalizeRulePortRange(curRule.IPProtocol, curRule.FromPort, curRule.ToPort)
				fromPort, _ := strconv.Atoi(fromPortStr)
				toPort, _ := strconv.Atoi(toPortStr)

				createRuleOpts = rules.CreateOpts{
					Direction:      rules.RuleDirection(direction),
					EtherType:      etherType,
					SecGroupID:     nhnSG.ID,
					PortRangeMin:   fromPort,
					PortRangeMax:   toPort,
					Protocol:       rules.RuleProtocol(strings.ToLower(curRule.IPProtocol)),
					RemoteIPPrefix: curRule.CIDR,
				}
			}

			start := call.Start()
			_, err = rules.Create(securityHandler.NetworkClient, createRuleOpts).Extract()
			if err != nil {
				if strings.Contains(strings.ToLower(err.Error()), "already exists") {
					cblogger.Infof("Rule already exists in S/G [%s], skipping: [%v]", nhnSG.ID, err)
				} else {
					newErr := fmt.Errorf("Failed to Create New Rule to the S/G : [%s] : [%v]", nhnSG.ID, err)
					cblogger.Error(newErr.Error())
					LoggingError(callLogInfo, newErr)
					return irs.SecurityInfo{}, newErr
				}
			}
			LoggingInfo(callLogInfo, start)
			cblogger.Infof("Succeeded in Adding New [%s], [%s] Rule!!", curRule.Direction, rules.RuleProtocol(strings.ToLower(curRule.IPProtocol)))
		}
	}

	// Return Current SecurityGroup Info.
	securityInfo, err := securityHandler.GetSecurity(sgIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the S/G Info : [%s] : [%v]", nhnSG.ID, err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

	// // AddServer will associate a server and a security group, enforcing the rules of the group on the server.
	// addServerResult := secgroups.AddServer(securityHandler.VMClient, serverID, securityIID.NameId)

	// // RemoveServer will disassociate a server from a security grou
	// removeServerResult := secgroups.RemoveServer(securityHandler.VMClient, serverID, securityIID.NameId)

	return securityInfo, nil
}

func (securityHandler *NhnCloudSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called RemoveRules()!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Region, call.SECURITYGROUP, sgIID.SystemId, "RemoveRules()")

	nhnSG, err := securityHandler.getRawSecurity(sgIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the S/G info from NHN!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	cblogger.Infof("S/G SystemId to Remove the Rules [%s]", nhnSG.ID)

	// Deletge the given S/G Rules
	for _, curRule := range *securityRules {
		if curRule.Direction == "" {
			return false, errors.New("Failed to Find 'Direction' Value in the requested rule!!")
		} else if curRule.IPProtocol == "" {
			return false, errors.New("Failed to Find 'IPProtocol' Value in the requested rule!!")
		} else if curRule.FromPort == "" {
			return false, errors.New("Failed to Find 'FromPort' Value in the requested rule!!")
		} else if curRule.ToPort == "" {
			return false, errors.New("Failed to Find 'ToPort' Value in the requested rule!!")
		} else if curRule.CIDR == "" {
			return false, errors.New("Failed to Find 'CIDR' Value in the requested rule!!")
		}

		cblogger.Infof("curRule.IPProtocol : [%s]", curRule.IPProtocol)

		if strings.EqualFold(curRule.IPProtocol, "ALL") { // Add SecurityGroup Rules in case of 'All Traffic Open Rule'
			if strings.EqualFold(curRule.FromPort, "-1") && strings.EqualFold(curRule.ToPort, "-1") {
				var direction string
				if strings.EqualFold(curRule.Direction, "inbound") {
					direction = "inbound"
				} else if strings.EqualFold(curRule.Direction, "outbound") {
					direction = "outbound"
				} else {
					return false, errors.New("Invalid Rule Direction!!")
				}

				// The current code creates a single rule without a protocol, while the
				// previous code created one rule per tcp/udp/icmp. Both have to be removable.
				ruleIds, err := securityHandler.getAllProtocolRuleIds(nhnSG.ID, direction, curRule.CIDR)
				if err != nil {
					newErr := fmt.Errorf("Failed to Find the 'All Traffic Open Rule' of the S/G : [%s] : [%v]", nhnSG.ID, err)
					cblogger.Error(newErr.Error())
					LoggingError(callLogInfo, newErr)
					return false, newErr
				}

				for _, ruleId := range ruleIds {
					cblogger.Infof("The RuleID of Current Rule : [%s]", ruleId)

					// Delete the Rule
					start := call.Start()
					delResult := rules.Delete(securityHandler.NetworkClient, ruleId)
					LoggingInfo(callLogInfo, start)
					if delResult.Err != nil {
						newErr := fmt.Errorf("Failed to Remove Rules of the S/G : [%s] : [%v]", nhnSG.ID, delResult.Err)
						cblogger.Error(newErr.Error())
						LoggingError(callLogInfo, newErr)
						return false, newErr
					}

					cblogger.Infof("Succeeded in Removing the [%s], [ALL] Rule!!", direction)
				}
			} else {
				return false, errors.New("To Specify 'All Traffic Allow Rule', Specify '-1' as FromPort/ToPort!!")
			}
		} else {
			// NHN stores '-1' as the whole 1-65535 range when the rule is created, so
			// the same normalization has to be applied before looking the rule up.
			lookupRule := curRule
			lookupRule.FromPort, lookupRule.ToPort = normalizeRulePortRange(curRule.IPProtocol, curRule.FromPort, curRule.ToPort)

			// Get the Rule ID from the S/G
			ruleId, err := securityHandler.getRuleIdFromRuleInfo(nhnSG.ID, lookupRule)
			if err != nil {
				newErr := fmt.Errorf("Failed to Find any S/G info. with the SystemId : [%s], [%v]", nhnSG.ID, err)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return false, newErr
			}

			cblogger.Infof("The RuleID of Current Rule : ", ruleId)

			// Delete the Rule
			start := call.Start()
			delResult := rules.Delete(securityHandler.NetworkClient, ruleId)
			if delResult.Err != nil {
				newErr := fmt.Errorf("Failed to Remove Rules of the S/G : [%s] : [%v]", nhnSG.ID, delResult.Err)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return false, newErr
			}
			LoggingInfo(callLogInfo, start)
			// spew.Dump(delResult)

			cblogger.Infof("Succeeded in Removing the [%s], [%s] Rule!!", curRule.Direction, curRule.IPProtocol)
		}
	}

	// // AddServer will associate a server and a security group, enforcing the rules of the group on the server.
	// addServerResult := secgroups.AddServer(securityHandler.VMClient, serverID, securityIID.NameId)

	// // RemoveServer will disassociate a server from a security group
	// removeServerResult := secgroups.RemoveServer(securityHandler.VMClient, serverID, securityIID.NameId)

	return true, nil
}

func (securityHandler *NhnCloudSecurityHandler) openOutboundAllProtocol(sgIID irs.IID) error {
	cblogger.Info("NHN Cloud driver: called openOutboundAllProtocol()!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Region, call.SECURITYGROUP, sgIID.SystemId, "openOutboundAllProtocol()")

	reqRules := []irs.SecurityRuleInfo{
		{
			Direction:  "outbound",
			IPProtocol: "ALL",
			FromPort:   "-1",
			ToPort:     "-1",
			CIDR:       "0.0.0.0/0",
		},
	}

	_, err := securityHandler.AddRules(sgIID, &reqRules)
	if err != nil {
		newErr := fmt.Errorf("Failed to Add Outbound All Protocol Opening Rule. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return newErr
	}

	return nil
}

func (securityHandler *NhnCloudSecurityHandler) mappingSecurityInfo(nhnSG secgroups.SecurityGroup) (*irs.SecurityInfo, error) {
	cblogger.Info("NHN Cloud Driver: called mappingSecurityInfo()!")

	secInfo := &irs.SecurityInfo{
		IId: irs.IID{
			NameId:   nhnSG.Name,
			SystemId: nhnSG.ID,
		},

		VpcIID: irs.IID{
			//NameId:   "",
			//SystemId: "",
		},

		//KeyValueList: []irs.KeyValue{
		//	{Key: "TenantID", Value: nhnSG.TenantID},
		//},
	}

	listOpts := rules.ListOpts{
		SecGroupID: nhnSG.ID,
	}

	allPages, err := rules.List(securityHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		cblogger.Error(err.Error())
		return nil, err
	}

	nhnRuleList, err := rules.ExtractRules(allPages)
	if err != nil {
		cblogger.Error(err.Error())
		return nil, err
	}

	if len(nhnRuleList) < 1 {
		cblogger.Infof("$$$ The S/G [%s] contains No Rule!!", nhnSG.ID)
		// return nil, nil // Caution!!
	} else {
		// Set Security Rule info. list
		var sgRuleList []irs.SecurityRuleInfo
		for _, nhnRule := range nhnRuleList {
			// NHN adds its own default egress rules, which carry neither a protocol
			// nor a remote IP prefix, to every S/G. Expose IPv4 default egress rule as 0.0.0.0/0.
			if strings.EqualFold(nhnRule.Protocol, "") && strings.EqualFold(nhnRule.RemoteIPPrefix, "") {
				if strings.EqualFold(nhnRule.Direction, string(rules.DirEgress)) && strings.EqualFold(nhnRule.EtherType, string(rules.EtherType4)) {
					nhnRule.RemoteIPPrefix = "0.0.0.0/0"
				} else {
					continue
				}
			}

			var direction string
			if strings.EqualFold(nhnRule.Direction, string(rules.DirIngress)) {
				direction = "inbound"
			} else if strings.EqualFold(nhnRule.Direction, string(rules.DirEgress)) {
				direction = "outbound"
			} else {
				return nil, errors.New("Invalid Rule Direction!!")
			}

			ipProtocol, fromPort, toPort := convertNhnRuleToCBRule(nhnRule)

			sgRuleList = append(sgRuleList, irs.SecurityRuleInfo{
				Direction:  direction,
				IPProtocol: ipProtocol,
				FromPort:   fromPort,
				ToPort:     toPort,
				CIDR:       nhnRule.RemoteIPPrefix,
			})
		}

		secInfo.SecurityRules = &sgRuleList
	}

	secInfo.KeyValueList = irs.StructToKeyValueList(nhnSG)
	return secInfo, nil
}

// NHN(Neutron) requires the ethertype to match the address family of the CIDR.
func getEtherTypeFromCIDR(cidr string) (rules.RuleEtherType, error) {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("Invalid CIDR : [%s] : [%v]", cidr, err)
	}

	if ip.To4() != nil {
		return rules.EtherType4, nil
	}
	return rules.EtherType6, nil
}

// CB-Spider expresses the whole port range as '-1', while NHN stores it as 1-65535.
// ICMP keeps '-1' since it has no port range at all.
func normalizeRulePortRange(ipProtocol string, fromPort string, toPort string) (string, string) {
	if strings.EqualFold(ipProtocol, "icmp") {
		return "-1", "-1"
	}

	if fromPort == "-1" || toPort == "-1" {
		return "1", "65535"
	}
	return fromPort, toPort
}

// Converts an NHN rule into the CB-Spider protocol/port representation.
// NHN leaves the protocol empty for a rule that allows every protocol.
func convertNhnRuleToCBRule(nhnRule rules.SecGroupRule) (string, string, string) {
	if strings.EqualFold(nhnRule.Protocol, "") {
		return "ALL", "-1", "-1"
	}

	if strings.EqualFold(nhnRule.Protocol, "icmp") {
		return strings.ToLower(nhnRule.Protocol), "-1", "-1" // Caution : Not strconv.Itoa(0)
	}
	return strings.ToLower(nhnRule.Protocol), strconv.Itoa(nhnRule.PortRangeMin), strconv.Itoa(nhnRule.PortRangeMax)
}

// Returns the rule IDs that make up an 'All Traffic Open Rule'. The rule created
// by the current code has no protocol, so it is looked up first. When it is not
// found, the tcp/udp/icmp set created by the previous code is looked up instead.
func (securityHandler *NhnCloudSecurityHandler) getAllProtocolRuleIds(systemId string, direction string, cidr string) ([]string, error) {
	ruleId, err := securityHandler.getRuleIdFromRuleInfo(systemId, irs.SecurityRuleInfo{
		Direction:  direction,
		IPProtocol: "ALL",
		FromPort:   "-1",
		ToPort:     "-1",
		CIDR:       cidr,
	})
	if err == nil && !strings.EqualFold(ruleId, "") {
		return []string{ruleId}, nil
	}

	var legacyRuleIds []string
	for _, curProtocolType := range []string{"tcp", "udp", "icmp"} {
		fromPort, toPort := normalizeRulePortRange(curProtocolType, "-1", "-1")
		legacyRuleId, legacyErr := securityHandler.getRuleIdFromRuleInfo(systemId, irs.SecurityRuleInfo{
			Direction:  direction,
			IPProtocol: curProtocolType,
			FromPort:   fromPort,
			ToPort:     toPort,
			CIDR:       cidr,
		})
		if legacyErr != nil || strings.EqualFold(legacyRuleId, "") {
			continue
		}
		legacyRuleIds = append(legacyRuleIds, legacyRuleId)
	}

	if len(legacyRuleIds) < 1 {
		return nil, errors.New("Failed to Find any RuleID of the 'All Traffic Open Rule'!!")
	}
	return legacyRuleIds, nil
}

func (securityHandler *NhnCloudSecurityHandler) getRuleIdFromRuleInfo(systemId string, givenRule irs.SecurityRuleInfo) (string, error) {
	cblogger.Info("NHN Cloud Driver: called getRuleIdFromRuleInfo()!")

	listOpts := rules.ListOpts{
		SecGroupID: systemId,
	}

	allPages, err := rules.List(securityHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		cblogger.Error(err.Error())
		return "", err
	}

	nhnRuleList, err := rules.ExtractRules(allPages)
	if err != nil {
		cblogger.Error(err.Error())
		return "", err
	}
	// spew.Dump(nhnRuleList)

	var ruleId string

	if len(nhnRuleList) < 1 {
		cblogger.Infof("$$$ The S/G [%s] contains No Rule!!", systemId)
		return "", nil // Caution!!
	} else {
		// Set Security Rule info. list
		for _, nhnRule := range nhnRuleList {
			var direction string
			if strings.EqualFold(nhnRule.Direction, string(rules.DirIngress)) {
				direction = "inbound"
			} else if strings.EqualFold(nhnRule.Direction, string(rules.DirEgress)) {
				direction = "outbound"
			} else {
				return "", errors.New("Invalid Rule Direction!!")
			}

			ipProtocol, fromPort, toPort := convertNhnRuleToCBRule(nhnRule)

			nhnCIDR := nhnRule.RemoteIPPrefix
			if nhnCIDR == "" && strings.EqualFold(nhnRule.Direction, string(rules.DirEgress)) && strings.EqualFold(nhnRule.EtherType, string(rules.EtherType4)) {
				nhnCIDR = "0.0.0.0/0"
			}

			if strings.EqualFold(givenRule.Direction, direction) && strings.EqualFold(givenRule.IPProtocol, ipProtocol) && strings.EqualFold(givenRule.FromPort, fromPort) && strings.EqualFold(givenRule.ToPort, toPort) && strings.EqualFold(givenRule.CIDR, nhnCIDR) {
				ruleId = nhnRule.ID
				break
			}
		}
	}

	if strings.EqualFold(ruleId, "") {
		return "", errors.New("Failed to Find RuleID with the Given S/G Rule!!")
	}
	return ruleId, nil
}

func (securityHandler *NhnCloudSecurityHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, "secId", "ListIID()")

	start := call.Start()

	var iidList []*irs.IID

	allPages, err := secgroups.List(securityHandler.VMClient).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get securitygroups information from NhnCloud!! : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return make([]*irs.IID, 0), newErr

	}

	allSecugroups, err := secgroups.ExtractSecurityGroups(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get securitygroups  List from NhnCloud!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return make([]*irs.IID, 0), newErr
	}

	for _, secgroups := range allSecugroups {
		var iid irs.IID
		iid.SystemId = secgroups.ID
		iid.NameId = secgroups.Name

		iidList = append(iidList, &iid)
	}

	LoggingInfo(callLogInfo, start)

	return iidList, nil
}
