package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"net/url"
	"strconv"
)

type IbmSecurityHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VpcService     *vpcv1.VpcV1
	Ctx            context.Context
}

func (securityHandler *IbmSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error){
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityReqInfo.IId.NameId, "CreateSecurity()")
	start := call.Start()

	// req 체크
	err := checkSecurityReqInfo(securityReqInfo)
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	exist, err := existSecurityGroup(securityReqInfo.IId, securityHandler.VpcService,securityHandler.Ctx)
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}else if exist{
		err = errors.New(fmt.Sprintf("already exist %s",securityReqInfo.IId.NameId))
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	vpc, err := getRawVPC(securityReqInfo.VpcIID,securityHandler.VpcService,securityHandler.Ctx)
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	options := &vpcv1.CreateSecurityGroupOptions{}
	options.SetVPC(&vpcv1.VPCIdentity{
		ID: vpc.ID,
	})
	options.SetName(securityReqInfo.IId.NameId)
	securityGroup, _,  err := securityHandler.VpcService.CreateSecurityGroupWithContext(securityHandler.Ctx,options)
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	if securityReqInfo.SecurityRules != nil {
		for _, securityRule := range *securityReqInfo.SecurityRules{
			if securityRule.IPProtocol == "tcp" || securityRule.IPProtocol == "udp"{
				portMinInt, _ := strconv.ParseInt(securityRule.FromPort,10,64)
				portMaxInt, _ := strconv.ParseInt(securityRule.ToPort,10,64)
				ruleOptions := &vpcv1.CreateSecurityGroupRuleOptions{}
				ruleOptions.SetSecurityGroupID(*securityGroup.ID)
				ruleOptions.SetSecurityGroupRulePrototype(&vpcv1.SecurityGroupRulePrototype{
					Direction: core.StringPtr(securityRule.Direction),
					Protocol:  core.StringPtr(securityRule.IPProtocol),
					PortMax: core.Int64Ptr(portMaxInt),
					PortMin: core.Int64Ptr(portMinInt),
					IPVersion: core.StringPtr("ipv4"),
					Remote: &vpcv1.SecurityGroupRuleRemotePrototype{
						CIDRBlock: &securityRule.CIDR,
					},
				})
				_, _, err := securityHandler.VpcService.CreateSecurityGroupRuleWithContext(securityHandler.Ctx, ruleOptions)
				if err != nil{
					options := &vpcv1.DeleteSecurityGroupOptions{}
					options.SetID(*securityGroup.ID)
					_, deleteError := securityHandler.VpcService.DeleteSecurityGroupWithContext(securityHandler.Ctx, options)
					if deleteError!= nil{
						err = errors.New(err.Error() + deleteError.Error())
					}
					LoggingError(hiscallInfo, err)
					return irs.SecurityInfo{}, err
				}
			} else{
				ruleOptions := &vpcv1.CreateSecurityGroupRuleOptions{}
				ruleOptions.SetSecurityGroupID(*securityGroup.ID)
				ruleOptions.SetSecurityGroupRulePrototype(&vpcv1.SecurityGroupRulePrototype{
					Direction: core.StringPtr(securityRule.Direction),
					Protocol:  core.StringPtr(securityRule.IPProtocol),
					IPVersion: core.StringPtr("ipv4"),
					Remote: &vpcv1.SecurityGroupRuleRemotePrototype{
						CIDRBlock: &securityRule.CIDR,
					},
				})
				_, _, err := securityHandler.VpcService.CreateSecurityGroupRuleWithContext(securityHandler.Ctx, ruleOptions)
				if err != nil{
					options := &vpcv1.DeleteSecurityGroupOptions{}
					options.SetID(*securityGroup.ID)
					_, deleteError := securityHandler.VpcService.DeleteSecurityGroupWithContext(securityHandler.Ctx, options)
					if deleteError!= nil{
						err = errors.New(err.Error() + deleteError.Error())
					}
					LoggingError(hiscallInfo, err)
					return irs.SecurityInfo{}, err
				}
			}

		}
	}

	rawSecurityGroup, err := getRawSecurityGroup(irs.IID{SystemId: *securityGroup.ID},securityHandler.VpcService,securityHandler.Ctx)
	if err != nil {
		options := &vpcv1.DeleteSecurityGroupOptions{}
		options.SetID(*securityGroup.ID)
		_, deleteError := securityHandler.VpcService.DeleteSecurityGroupWithContext(securityHandler.Ctx, options)
		if deleteError!= nil{
			err = errors.New(err.Error() + deleteError.Error())
		}
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	securityGroupInfo, err := setSecurityGroupInfo(rawSecurityGroup)
	if err != nil {
		options := &vpcv1.DeleteSecurityGroupOptions{}
		options.SetID(*securityGroup.ID)
		_, deleteError := securityHandler.VpcService.DeleteSecurityGroupWithContext(securityHandler.Ctx, options)
		if deleteError!= nil{
			err = errors.New(err.Error() + deleteError.Error())
		}
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)
	return securityGroupInfo, nil
}

func (securityHandler *IbmSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error){
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, "SECURITYGROUP", "ListSecurity()")
	start := call.Start()
	options := &vpcv1.ListSecurityGroupsOptions{}
	securityGroups, _, err := securityHandler.VpcService.ListSecurityGroupsWithContext(securityHandler.Ctx, options)
	if err != nil{
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	var securityGroupList []*irs.SecurityInfo
	for {
		for _, securityGroup :=range securityGroups.SecurityGroups {
			securityInfo, err := setSecurityGroupInfo(securityGroup)
			if err != nil{
				LoggingError(hiscallInfo, err)
				return nil, err
			}
			securityGroupList = append(securityGroupList, &securityInfo)
		}
		nextstr, _ := getSecurityGroupNextHref(securityGroups.Next)
		if nextstr != "" {
			options2 := &vpcv1.ListSecurityGroupsOptions{
				Start: core.StringPtr(nextstr),
			}
			securityGroups, _, err = securityHandler.VpcService.ListSecurityGroupsWithContext(securityHandler.Ctx, options2)
			if err != nil{
				LoggingError(hiscallInfo, err)
				return nil, err
			}
		} else {
			break
		}
	}
	LoggingInfo(hiscallInfo, start)
	return securityGroupList, nil
}

func (securityHandler *IbmSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error){
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityIID.NameId, "GetSecurity()")
	start := call.Start()

	err:= checkSecurityGroupIID(securityIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	securityGroup, err := getRawSecurityGroup(securityIID,securityHandler.VpcService,securityHandler.Ctx)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	securityGroupInfo, err := setSecurityGroupInfo(securityGroup)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)
	return securityGroupInfo, nil
}

func (securityHandler *IbmSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error){
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityIID.NameId, "DeleteSecurity()")
	start := call.Start()

	err:= checkSecurityGroupIID(securityIID)

	if err != nil{
		LoggingError(hiscallInfo, err)
		return false, err
	}
	securityGroup, err := getRawSecurityGroup(securityIID,securityHandler.VpcService,securityHandler.Ctx)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	options := &vpcv1.DeleteSecurityGroupOptions{}
	options.SetID(*securityGroup.ID)
	res, err := securityHandler.VpcService.DeleteSecurityGroupWithContext(securityHandler.Ctx, options)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	if res.StatusCode == 204{
		LoggingInfo(hiscallInfo, start)
		return true, nil
	} else {
		err = errors.New(res.String())
		LoggingError(hiscallInfo, err)
		return false, err
	}
}

func existSecurityGroup(securityIID irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) (bool, error){
	if securityIID.NameId != ""{
		options := &vpcv1.ListSecurityGroupsOptions{}
		securityGroups, _, err := vpcService.ListSecurityGroupsWithContext(ctx, options)
		if err != nil{
			return false, err
		}
		for {
			for _, securityGroup :=range securityGroups.SecurityGroups {
				if *securityGroup.Name == securityIID.NameId{
					return true, nil
				}
			}
			nextstr, _ := getSecurityGroupNextHref(securityGroups.Next)
			if nextstr != "" {
				options2 := &vpcv1.ListSecurityGroupsOptions{
					Start: core.StringPtr(nextstr),
				}
				securityGroups, _, err =vpcService.ListSecurityGroupsWithContext(ctx, options2)
				if err != nil {
					return false, err
				}
			} else {
				break
			}
		}
		return false, nil
	} else {
		err := errors.New("invalid securityIID")
		return false, err
	}
}

func getRawSecurityGroup(securityIID irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) (vpcv1.SecurityGroup,error){
	if securityIID.SystemId == ""{
		options := &vpcv1.ListSecurityGroupsOptions{}
		securityGroups, _, err := vpcService.ListSecurityGroupsWithContext(ctx, options)
		if err != nil{
			return vpcv1.SecurityGroup{}, err
		}
		for {
			for _, securityGroup :=range securityGroups.SecurityGroups {
				if *securityGroup.Name == securityIID.NameId{
					return securityGroup, nil
				}
			}
			nextstr, _ := getSecurityGroupNextHref(securityGroups.Next)
			if nextstr != "" {
				options2 := &vpcv1.ListSecurityGroupsOptions{
					Start: core.StringPtr(nextstr),
				}
				securityGroups, _, err =vpcService.ListSecurityGroupsWithContext(ctx, options2)
				if err != nil {
					return vpcv1.SecurityGroup{}, err
				}
			} else {
				break
			}
		}
		return vpcv1.SecurityGroup{}, errors.New(fmt.Sprintf("not found SecurityGroup %s",securityIID.NameId ))
	}else{
		options := &vpcv1.GetSecurityGroupOptions{}
		options.SetID(securityIID.SystemId)
		sg, _, err := vpcService.GetSecurityGroupWithContext(ctx,options)
		if err != nil {
			return vpcv1.SecurityGroup{}, err
		}
		return *sg, nil
	}
}

func setSecurityGroupInfo (securityGroup vpcv1.SecurityGroup) (irs.SecurityInfo, error){
	securityInfo := irs.SecurityInfo{
		IId : irs.IID{
			NameId: *securityGroup.Name,
			SystemId: *securityGroup.ID,
		},
		VpcIID: irs.IID{
			NameId: *securityGroup.VPC.Name,
			SystemId: *securityGroup.VPC.ID,
		},
	}
	ruleList, err := setRule(securityGroup)
	if err != nil{
		return irs.SecurityInfo{}, err
	}
	securityInfo.SecurityRules = &ruleList
	return securityInfo, nil
}

func setRule(securityGroup vpcv1.SecurityGroup) ([]irs.SecurityRuleInfo, error){
	var ruleList []irs.SecurityRuleInfo
	for _,rule := range securityGroup.Rules{
		jsonRuleBytes, err := json.Marshal(rule)
		if err != nil{
			return nil, err
		}
		jsonRuleMap := make(map[string]json.RawMessage)
		unmarshalErr := json.Unmarshal(jsonRuleBytes, &jsonRuleMap)
		if unmarshalErr != nil {
			return nil, err
		}
		remoteJson := jsonRuleMap["remote"]
		var remote vpcv1.SecurityGroupRuleRemote
		unmarshalErr = json.Unmarshal(remoteJson,&remote)
		if unmarshalErr != nil {
			return nil, err
		}
		if remote.Name != nil && *remote.Name == *securityGroup.Name{
			continue
		}
		var ruleProtocolAll vpcv1.SecurityGroupRulePrototypeSecurityGroupRuleProtocolAll
		_ = json.Unmarshal(jsonRuleBytes,&ruleProtocolAll)

		if *ruleProtocolAll.Protocol == "tcp" || *ruleProtocolAll.Protocol == "udp"{
			var ruleProtocolTcpudp vpcv1.SecurityGroupRulePrototypeSecurityGroupRuleProtocolTcpudp
			_ = json.Unmarshal(jsonRuleBytes,&ruleProtocolTcpudp)
			ruleInfo:= irs.SecurityRuleInfo{
				IPProtocol: *ruleProtocolTcpudp.Protocol,
				Direction: *ruleProtocolTcpudp.Direction,
				FromPort: strconv.FormatInt(*ruleProtocolTcpudp.PortMin, 10),
				ToPort: strconv.FormatInt(*ruleProtocolTcpudp.PortMax, 10),
				CIDR: *remote.CIDRBlock,
			}
			ruleList = append(ruleList,ruleInfo)
		} else if *ruleProtocolAll.Protocol == "icmp" {
			var ruleProtocolIcmp vpcv1.SecurityGroupRulePrototypeSecurityGroupRuleProtocolIcmp
			_ = json.Unmarshal(jsonRuleBytes,&ruleProtocolIcmp)
			ruleInfo := irs.SecurityRuleInfo{
				IPProtocol: *ruleProtocolIcmp.Protocol,
				Direction: *ruleProtocolIcmp.Direction,
				CIDR: *remote.CIDRBlock,
			}
			ruleList = append(ruleList,ruleInfo)
		} else {
			ruleInfo := irs.SecurityRuleInfo{
				IPProtocol: *ruleProtocolAll.Protocol,
				Direction: *ruleProtocolAll.Direction,
				CIDR: *remote.CIDRBlock,
			}
			ruleList = append(ruleList,ruleInfo)
		}
	}
	return ruleList, nil
}

func getSecurityGroupNextHref (next *vpcv1.SecurityGroupCollectionNext) (string, error){
	if next != nil{
		href := *next.Href
		u, err := url.Parse(href)
		if err != nil{
			return "", err
		}
		paramMap, _ := url.ParseQuery(u.RawQuery)
		if paramMap != nil  {
			safe := paramMap["start"]
			if safe != nil && len(safe) > 0 {
				return safe[0],nil
			}
		}
	}
	return "", errors.New("NOT NEXT")
}

func checkSecurityGroupIID(securityIID irs.IID) error {
	if securityIID.SystemId == "" && securityIID.NameId == ""{
		err := errors.New("invalid IID")
		return err
	}
	return nil
}

func checkSecurityReqInfo (securityReqInfo irs.SecurityReqInfo) error{
	if securityReqInfo.IId.NameId == ""{
		return errors.New("invalid securityReqInfo IID")
	}
	if securityReqInfo.VpcIID.NameId == "" && securityReqInfo.VpcIID.SystemId == ""{
		return errors.New("invalid securityReqInfo VpcIID")
	}
	return nil
}
