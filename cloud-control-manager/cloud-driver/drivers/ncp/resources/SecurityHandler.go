// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP Security Group Handler
//
// by ETRI, 2020.10.
// by ETRI, 2022.02. updated

package resources

import (
	"fmt"
	"errors"
	"strings"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/server"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpSecurityHandler struct {
	CredentialInfo 		idrv.CredentialInfo
	RegionInfo     		idrv.RegionInfo
	VMClient         	*server.APIClient
}

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP Security Group Handler")
}

func (securityHandler *NcpSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Info("NCP Cloud Driver: called CreateSecurity()!")

	InitLog()
	callLogInfo := GetCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, securityReqInfo.IId.NameId, "CreateSecurity()")

	// CB-Spider에서 IID2 포멧(최대 30자)의 이름으로 변경되어 사용되므로
	// 사용자가 지정한 본래의 이름으로 사용하기 위해 맨뒤에서부터의 21자리 문자 제거
	// originalNameId := GetOriginalNameId(securityReqInfo.IId.NameId)
	// cblogger.Infof("# VPC NameID requested by user : [%s]", originalNameId)

	if securityReqInfo.IId.NameId == "" {
		newErr := fmt.Errorf("Invalid S/G Name Requested")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

	reqNameId := securityReqInfo.IId.NameId

	var sgSystemId string

	// Get SecurityGroup list
	securityList, err := securityHandler.ListSecurity()
	if err != nil {
		newErr := fmt.Errorf("Failed to Find SecurityGroup list : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

	// Search S/G SystemId by NameID in the SecurityGroup list
	for _, security := range securityList {
		if strings.EqualFold(security.IId.NameId, reqNameId) {
			sgSystemId = security.IId.SystemId
			cblogger.Infof("# SystemId of the matched S/G : [%s]", sgSystemId)
			break
		} 
	}

	// When the SecurityGroup is not found
	if strings.EqualFold(sgSystemId, "") {
		err := fmt.Errorf("Failed to Find S/G SystemId with the NameID.")
		return irs.SecurityInfo{}, err
	}

	sgInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: sgSystemId})
	if err != nil {
		newErr := fmt.Errorf("Failed to Get SecurityGroup Info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
	} else {
		sgInfo.IId.NameId = securityReqInfo.IId.NameId  // Caution!! For IID2 NameID validation check
	}
	
	return sgInfo, err
}

func (securityHandler *NcpSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	cblogger.Info("NCP Cloud Driver: called GetSecurity()!!")

	cblogger.Infof("NCP securityIID.SystemId : [%s]", securityIID.SystemId)

	InitLog()
	callLogInfo := GetCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, securityIID.NameId, "GetSecurity()")

	if securityIID.SystemId == "" {
		newErr := fmt.Errorf("Invalid S/G SystemId")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

	// spew.Dump(securityIID.SystemId)
	sgNumList := []*string{ncloud.String(securityIID.SystemId)}

	sgReq := server.GetAccessControlGroupListRequest{AccessControlGroupConfigurationNoList: sgNumList}

	// Search NCP AccessControlGroup with securityIID.SystemId
	callLogStart := call.Start()
	ncpSG, err := securityHandler.VMClient.V2Api.GetAccessControlGroupList(&sgReq)
	if err != nil {
		cblogger.Error(*ncpSG.ReturnMessage)
		newErr := fmt.Errorf("Failed to Get SecurityGroup list from NCP : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)
	
	if len(ncpSG.AccessControlGroupList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any SecurityGroup with the SystemId : [%s]", securityIID.SystemId)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

	cblogger.Info("Succeeded in Getting NCP SecurityGroup info.")
	// spew.Dump(ncpSG)

	securityGroupInfo, securityInfoErr := securityHandler.MappingSecurityInfo(*ncpSG.AccessControlGroupList[0])
	if securityInfoErr != nil {
		cblogger.Error(securityInfoErr)
		return irs.SecurityInfo{}, securityInfoErr
	}

	return securityGroupInfo, nil
}

func (securityHandler *NcpSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	cblogger.Info("NCP Cloud Driver: called ListSecurity()!!")

	InitLog()
    callLogInfo := GetCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, "ListSecurity()", "ListSecurity()")

	var securityGroupList []*irs.SecurityInfo

	sgReq := server.GetAccessControlGroupListRequest{AccessControlGroupConfigurationNoList: nil}

	// Search NCP AccessControlGroup with securityIID.SystemId
	callLogStart := call.Start()
	ncpSG, err := securityHandler.VMClient.V2Api.GetAccessControlGroupList(&sgReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find SecurityGroup list from NCP : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(ncpSG.AccessControlGroupList) < 1 {
		return nil, errors.New("Failed to Find Any SecurityGroup info!!")
	}

	cblogger.Info("Succeeded in Getting NCP SecurityGroup info.")

	for _, sg := range ncpSG.AccessControlGroupList {
		cblogger.Info("NCP SecurityGroup No : ", *sg.AccessControlGroupConfigurationNo)

		sgInfo, _ := securityHandler.GetSecurity(irs.IID{SystemId: *sg.AccessControlGroupConfigurationNo})
		securityGroupList = append(securityGroupList, &sgInfo)
	}

	return securityGroupList, nil
}

func (securityHandler *NcpSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	cblogger.Info("NCP Cloud Driver: called DeleteSecurity()!")

	cblogger.Infof("securityIID.SystemId to Delete : [%s]", securityIID.SystemId)

	InitLog()
	callLogInfo := GetCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, securityIID.NameId, "DeleteSecurity()")

	if securityIID.SystemId == "" {
		newErr := fmt.Errorf("Invalid S/G SystemId")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	sgInfo, err := securityHandler.GetSecurity(securityIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find the SecurityGroup info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	cblogger.Info("Succeeded in Deleting the SecurityGroup : " + sgInfo.IId.NameId)
	return true, nil
}

func (securityHandler *NcpSecurityHandler) MappingSecurityInfo(ncpSecurityGroup server.AccessControlGroup) (irs.SecurityInfo, error) {
	cblogger.Info("NCP Cloud Driver: called MappingSecurityInfo()!")

	var ncpSgRuleList []irs.SecurityRuleInfo

	ncpSgRuleList, ruleInfoError := securityHandler.ExtractSecurityRuleInfo(ncloud.StringValue(ncpSecurityGroup.AccessControlGroupConfigurationNo))
	if ruleInfoError != nil {
		cblogger.Error(ruleInfoError)
		// return irs.SecurityInfo{}, ruleInfoError    // Caution!!
	}

	securityInfo := irs.SecurityInfo{
		IId:           irs.IID{NameId: ncloud.StringValue(ncpSecurityGroup.AccessControlGroupName), SystemId: ncloud.StringValue(ncpSecurityGroup.AccessControlGroupConfigurationNo)},
		VpcIID:        irs.IID{NameId: "", SystemId: ""},
		SecurityRules: &ncpSgRuleList,

		KeyValueList: []irs.KeyValue{
			{Key: "SecurityGroupDescription", Value: ncloud.StringValue(ncpSecurityGroup.AccessControlGroupDescription)},
			{Key: "CreateTime", Value: ncloud.StringValue(ncpSecurityGroup.CreateDate)},
			// {Key: "IsDefaultGroup", Value: strconv.FormatBool(*ncpSecurityGroup.IsDefaultGroup)},
		},
	}

	return securityInfo, nil
}

// Search NCP SecurityGroup Rule List
func (securityHandler *NcpSecurityHandler) ExtractSecurityRuleInfo(ncpSecurityGroupId string) ([]irs.SecurityRuleInfo, error) {
	cblogger.Info("NCP Cloud Driver: called ExtractSecurityRuleInfo()!")
	
	var securityRuleList []irs.SecurityRuleInfo

	sgRuleReq := server.GetAccessControlRuleListRequest{
		AccessControlGroupConfigurationNo: ncloud.String(ncpSecurityGroupId),
	}

	// securityIID.SystemId와 NCP의 AccessControlGroupConfigurationNo 비교
	ncpRuleList, err := securityHandler.VMClient.V2Api.GetAccessControlRuleList(&sgRuleReq)
	if err != nil {
		cblogger.Error(*ncpRuleList.ReturnMessage)
		newErr := fmt.Errorf("Failed to Get SecurityRule List from NCP : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if len(ncpRuleList.AccessControlRuleList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any SecurityRule of S/G SystemId : [%s]", ncpSecurityGroupId)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// spew.Dump(ncpRuleList)

	// Caution!!) FromPort string, ToPort string
	var curSecurityRuleInfo irs.SecurityRuleInfo

	for _, curRule := range ncpRuleList.AccessControlRuleList {
		curSecurityRuleInfo.IPProtocol = ncloud.StringValue(curRule.ProtocolType.CodeName)

		if curSecurityRuleInfo.IPProtocol == "icmp" {
			curSecurityRuleInfo.FromPort = "-1"
			curSecurityRuleInfo.ToPort = "-1"
		} else if ncloud.StringValue(curRule.DestinationPort) == "1-65535" {
			curSecurityRuleInfo.FromPort = "1"
			curSecurityRuleInfo.ToPort = "65535"
		} else {
			curSecurityRuleInfo.FromPort = ncloud.StringValue(curRule.DestinationPort)
			curSecurityRuleInfo.ToPort = ncloud.StringValue(curRule.DestinationPort)
		}

		curSecurityRuleInfo.Direction = "inbound" //NCP Classic S/G : inbound rule만 지원
		curSecurityRuleInfo.CIDR = ncloud.StringValue(curRule.SourceIp) //CIDR

		securityRuleList = append(securityRuleList, curSecurityRuleInfo)
	}

	return securityRuleList, nil
}

func (securityHandler *NcpSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	cblogger.Info("NCP Cloud Driver: called AddRules()!")

    return irs.SecurityInfo{}, fmt.Errorf("Does not support AddRules() yet!!")
}

func (securityHandler *NcpSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	cblogger.Info("NCP Cloud Driver: called RemoveRules()!")

    return false, fmt.Errorf("Does not support RemoveRules() yet!!")
}

func (securityHandler *NcpSecurityHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}

