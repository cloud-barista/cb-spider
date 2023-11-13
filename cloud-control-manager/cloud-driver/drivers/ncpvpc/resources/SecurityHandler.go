// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP VPC Security Group Handler
//
// by ETRI, 2020.10.
// Updated by ETRI, 2022.11.

package resources

import (
	"fmt"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"
	"github.com/davecgh/go-spew/spew"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVpcSecurityHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *vserver.APIClient
}

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP Security Group Handler")
}

func (securityHandler *NcpVpcSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Info("NCP VPC cloud driver: called CreateSecurity()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(securityHandler.RegionInfo.Region, call.SECURITYGROUP, securityReqInfo.IId.NameId, "CreateSecurity()")

	if securityReqInfo.IId.NameId == "" {
		createErr := fmt.Errorf("Invalid S/G Name")
		cblogger.Error(createErr.Error())
		LoggingError(callLogInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}

	// Check if the S/G exists
	sgList, err := securityHandler.ListSecurity()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		return irs.SecurityInfo{}, err
	}

	for _, sg := range sgList {
		if strings.EqualFold(sg.IId.NameId, securityReqInfo.IId.NameId) {
			createErr := fmt.Errorf("Specified S/G [%s] already exists", securityReqInfo.IId.NameId)
			cblogger.Error(createErr.Error())
			LoggingError(callLogInfo, createErr)
			return irs.SecurityInfo{}, createErr
		}
	}

	// Create New S/G
	sgReq := vserver.CreateAccessControlGroupRequest {
		RegionCode: 				&securityHandler.RegionInfo.Region,
		AccessControlGroupName: 	&securityReqInfo.IId.NameId,
		VpcNo: 						&securityReqInfo.VpcIID.SystemId,
	}

	callLogStart := call.Start()
	ncpVpcSG, err := securityHandler.VMClient.V2Api.CreateAccessControlGroup(&sgReq)
	if err != nil {
		cblogger.Error(*ncpVpcSG.ReturnMessage)
		newErr := fmt.Errorf("Failed to Create New S/G : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if *ncpVpcSG.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Create Any S/G!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	} else {
		cblogger.Infof("Succeeded in Creating New S/G : [%s]", *ncpVpcSG.AccessControlGroupList[0].AccessControlGroupName)
	}

	sgNo := ncpVpcSG.AccessControlGroupList[0].AccessControlGroupNo

	// Create S/G Rules
	for _, curRule := range *securityReqInfo.SecurityRules {	
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

		var ruleDirection string
		if strings.EqualFold(curRule.Direction, "inbound") {
			ruleDirection = "inbound"
		} else if strings.EqualFold(curRule.Direction, "outbound") {
			ruleDirection = "outbound"
		} else {
			return irs.SecurityInfo{}, errors.New("Specified Invalid Security Rule Direction!!")
		}

		if strings.EqualFold(ruleDirection, "inbound") {
			protocolType := strings.ToUpper(curRule.IPProtocol) // Caution : Valid values : [TCP, UDP, ICMP]

			if strings.EqualFold(protocolType, "ALL") {
				if strings.EqualFold(curRule.FromPort, "-1") && strings.EqualFold(curRule.ToPort, "-1") {
					allProtocolTypeCode := []string {"TCP", "UDP", "ICMP"}
					var allPortRange string
					allCIDR := "0.0.0.0/0"

					for _, curProtocolType := range allProtocolTypeCode {
						if strings.EqualFold(curProtocolType, "ICMP") {
							allPortRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
						} else {
							allPortRange = "1-65535" // All Range
						}

						var ruleParameterList []*vserver.AddAccessControlGroupRuleParameter

						ruleParameter := vserver.AddAccessControlGroupRuleParameter {
							IpBlock: 					&allCIDR,
							PortRange:					&allPortRange,
							ProtocolTypeCode: 			&curProtocolType,
						}
			
						ruleParameterList = append(ruleParameterList, &ruleParameter)
			
						// Create Inbound Security Rule
						inboundReq := vserver.AddAccessControlGroupInboundRuleRequest {
							RegionCode: 				&securityHandler.RegionInfo.Region,
							AccessControlGroupNo: 		sgNo,
							VpcNo: 						&securityReqInfo.VpcIID.SystemId,
							AccessControlGroupRuleList: ruleParameterList,
						}
			
						callLogStart := call.Start()
						sgInboundRule, err := securityHandler.VMClient.V2Api.AddAccessControlGroupInboundRule(&inboundReq)
						if err != nil {
							cblogger.Error(err)
							cblogger.Error(*sgInboundRule.ReturnMessage)
							LoggingError(callLogInfo, err)
							return irs.SecurityInfo{}, err
						}
						LoggingInfo(callLogInfo, callLogStart)
			
						if *sgInboundRule.TotalRows < 1 {
							return irs.SecurityInfo{}, errors.New("Failed to Create New S/G Inbound Rule!!")
						} else {
							cblogger.Infof("Succeeded in Creating New [%s] Inbound Rule!!", curProtocolType)

							time.Sleep(time.Second * 5)  // Caution!!
						}
					}
				} else {
					return irs.SecurityInfo{}, errors.New("To Specify 'All Traffic Allow Rule', Specify '-1' as FromPort/ToPort!!")
				}
			} else {
				var portRange string
			
				if strings.EqualFold(curRule.IPProtocol, "ICMP") {
					portRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
				} else if strings.EqualFold(curRule.FromPort, curRule.ToPort) {
					portRange = strings.ToLower(curRule.FromPort)	// string				
				} else if !strings.EqualFold(curRule.FromPort, curRule.ToPort) {
					portRange = strings.ToLower(curRule.FromPort) + "-"+ strings.ToLower(curRule.ToPort) // string				
				}
	
				if strings.EqualFold(curRule.FromPort, "-1") || strings.EqualFold(curRule.ToPort, "-1") {
					if strings.EqualFold(curRule.IPProtocol, "ICMP") {
						portRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
					} else {
						portRange = "1-65535" // All Range
					}
				}
	
				var ruleParameterList []*vserver.AddAccessControlGroupRuleParameter

				ruleParameter := vserver.AddAccessControlGroupRuleParameter {
					IpBlock: 					&curRule.CIDR,
					PortRange:					&portRange,
					ProtocolTypeCode: 			&protocolType,
				}	
				ruleParameterList = append(ruleParameterList, &ruleParameter)
	
				// Create Inbound Security Rule
				inboundReq := vserver.AddAccessControlGroupInboundRuleRequest {
					RegionCode: 				&securityHandler.RegionInfo.Region,
					AccessControlGroupNo: 		sgNo,
					VpcNo: 						&securityReqInfo.VpcIID.SystemId,
					AccessControlGroupRuleList: ruleParameterList,
				}
	
				callLogStart := call.Start()
				sgInboundRule, err := securityHandler.VMClient.V2Api.AddAccessControlGroupInboundRule(&inboundReq)
				if err != nil {
					cblogger.Error(err)
					cblogger.Error(*sgInboundRule.ReturnMessage)
					LoggingError(callLogInfo, err)
					return irs.SecurityInfo{}, err
				}
				LoggingInfo(callLogInfo, callLogStart)
	
				if *sgInboundRule.TotalRows < 1 {
					return irs.SecurityInfo{}, errors.New("Failed to Create S/G Inbound Rule!!")
				} else {
					cblogger.Infof("Succeeded in Creating New [%s] Inbound Rule!!", protocolType)
				}
			}
		}

		if strings.EqualFold(ruleDirection, "outbound") {
			protocolType := strings.ToUpper(curRule.IPProtocol) // Caution : Valid values : [TCP, UDP, ICMP] 

			if strings.EqualFold(protocolType, "ALL") {
				if strings.EqualFold(curRule.FromPort, "-1") && strings.EqualFold(curRule.ToPort, "-1") {
					allProtocolTypeCode := []string {"TCP", "UDP", "ICMP"}
					var allPortRange string // All Range
					allCIDR := "0.0.0.0/0"

					for _, curProtocolType := range allProtocolTypeCode {
						if strings.EqualFold(curProtocolType, "ICMP") {
							allPortRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
						} else {
							allPortRange = "1-65535" // All Range
						}

						var ruleParameterList []*vserver.AddAccessControlGroupRuleParameter		
						ruleParameter := vserver.AddAccessControlGroupRuleParameter {
							IpBlock: 					&allCIDR,
							PortRange:					&allPortRange,
							ProtocolTypeCode: 			&curProtocolType,
						}			
						ruleParameterList = append(ruleParameterList, &ruleParameter)
			
						// Create Inbound Security Rule
						outboundReq := vserver.AddAccessControlGroupOutboundRuleRequest {
							RegionCode: 				&securityHandler.RegionInfo.Region,
							AccessControlGroupNo: 		sgNo,
							VpcNo: 						&securityReqInfo.VpcIID.SystemId,
							AccessControlGroupRuleList: ruleParameterList,
						}
			
						callLogStart := call.Start()
						sgOutboundRule, err := securityHandler.VMClient.V2Api.AddAccessControlGroupOutboundRule(&outboundReq)
						if err != nil {
							cblogger.Error(err)
							cblogger.Error(*sgOutboundRule.ReturnMessage)
							LoggingError(callLogInfo, err)
							return irs.SecurityInfo{}, err
						}
						LoggingInfo(callLogInfo, callLogStart)
			
						if *sgOutboundRule.TotalRows < 1 {
							return irs.SecurityInfo{}, errors.New("Failed to Create New S/G Outbound Rule!!")
						} else {
							cblogger.Infof("Succeeded in Creating New [%s] Outbound Rule!!", curProtocolType)

							time.Sleep(time.Second * 5)  // Caution!!
						}
					}
				} else {
					return irs.SecurityInfo{}, errors.New("To Specify 'All Traffic Allow Rule', Specify '-1' as FromPort/ToPort!!")
				}
			} else {
				var portRange string
			
				if strings.EqualFold(curRule.IPProtocol, "ICMP") {
					portRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
				} else if strings.EqualFold(curRule.FromPort, curRule.ToPort) {
					portRange = strings.ToLower(curRule.FromPort)	// string				
				} else if !strings.EqualFold(curRule.FromPort, curRule.ToPort) {
					portRange = strings.ToLower(curRule.FromPort) + "-"+ strings.ToLower(curRule.ToPort) // string				
				}
	
				if strings.EqualFold(curRule.FromPort, "-1") || strings.EqualFold(curRule.ToPort, "-1") {
					if strings.EqualFold(curRule.IPProtocol, "ICMP") {
						portRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
					} else {
						portRange = "1-65535" // All Range
					}	
				}

				var ruleParameterList []*vserver.AddAccessControlGroupRuleParameter	
				ruleParameter := vserver.AddAccessControlGroupRuleParameter {
					IpBlock: 					&curRule.CIDR,
					PortRange:					&portRange,
					ProtocolTypeCode: 			&protocolType,
				}	
				ruleParameterList = append(ruleParameterList, &ruleParameter)
	
				// Create Inbound Security Rule
				outboundReq := vserver.AddAccessControlGroupOutboundRuleRequest {
					RegionCode: 				&securityHandler.RegionInfo.Region,
					AccessControlGroupNo: 		sgNo,
					VpcNo: 						&securityReqInfo.VpcIID.SystemId,
					AccessControlGroupRuleList: ruleParameterList,
				}
	
				callLogStart := call.Start()
				sgOutboundRule, err := securityHandler.VMClient.V2Api.AddAccessControlGroupOutboundRule(&outboundReq)
				if err != nil {
					cblogger.Error(err)
					cblogger.Error(*sgOutboundRule.ReturnMessage)
					LoggingError(callLogInfo, err)
					return irs.SecurityInfo{}, err
				}
				LoggingInfo(callLogInfo, callLogStart)
	
				if *sgOutboundRule.TotalRows < 1 {
					return irs.SecurityInfo{}, errors.New("Failed to Create New S/G Outbound Rule!!")
				} else {
					cblogger.Infof("Succeeded in Creating New [%s] Outbound Rule!!", protocolType)
				}
			}
		}
	}

	// Basically, Open 'Outbound' All Protocol for Any S/G (<= CB-Spider Rule)
	openErr := securityHandler.OpenOutboundAllProtocol(irs.IID{SystemId: *sgNo})
	if openErr != nil {
		cblogger.Error(openErr)
		LoggingError(callLogInfo, openErr)
		// return irs.SecurityInfo{}, openErr
	}

	// Return Created S/G Info.
	sgInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: *sgNo})
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		return irs.SecurityInfo{}, err
	}

	return sgInfo, nil
}

func (securityHandler *NcpVpcSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	cblogger.Info("NCP VPC cloud driver: called GetSecurity()!!")
	cblogger.Infof("NCP securityIID.SystemId : [%s]", securityIID.SystemId)

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, securityIID.SystemId, "GetSecurity()")

	sgNumList := []*string{ncloud.String(securityIID.SystemId),}
	sgReq := vserver.GetAccessControlGroupListRequest {
		RegionCode: 				&securityHandler.RegionInfo.Region, 
		AccessControlGroupNoList: 	sgNumList,
	}

	// Search NCP VPC AccessControlGroup with securityIID.SystemId
	callLogStart := call.Start()
	ncpVpcSG, err := securityHandler.VMClient.V2Api.GetAccessControlGroupList(&sgReq)
	if err != nil {
		cblogger.Error(err)
		cblogger.Error(*ncpVpcSG.ReturnMessage)
		LoggingError(callLogInfo, err)
		return irs.SecurityInfo{}, err
	}
	LoggingInfo(callLogInfo, callLogStart)

	if *ncpVpcSG.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Find Any S/G with the SystemId : [%s]", securityIID.SystemId)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

	cblogger.Info("Succeeded in Getting S/G info.")

	sgInfo, sgInfoErr := securityHandler.MappingSecurityInfo(*ncpVpcSG.AccessControlGroupList[0])
	if sgInfoErr != nil {
		cblogger.Error(sgInfoErr)
		return irs.SecurityInfo{}, sgInfoErr
	}

	return sgInfo, nil
}

func (securityHandler *NcpVpcSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	cblogger.Info("NCP VPC cloud driver: called ListSecurity()!!")

	InitLog() // Caution!!
    callLogInfo := GetCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, "ListSecurity()", "ListSecurity()")

	var securityGroupList []*irs.SecurityInfo

	sgReq := vserver.GetAccessControlGroupListRequest{
		RegionCode: 				&securityHandler.RegionInfo.Region,
		AccessControlGroupNoList: 	nil,
	}

	// Search NCP VPC AccessControlGroup list
	callLogStart := call.Start()
	sgList, err := securityHandler.VMClient.V2Api.GetAccessControlGroupList(&sgReq)
	if err != nil {
		cblogger.Error(err)
		cblogger.Error(*sgList.ReturnMessage)
		LoggingError(callLogInfo, err)
		return nil, err
	}
	LoggingInfo(callLogInfo, callLogStart)

	if *sgList.TotalRows < 1 {
		return nil, errors.New("Failed to Find any S/G list!!")
	}

	cblogger.Info("Succeeded in Getting S/G List.")

	for _, sg := range sgList.AccessControlGroupList {		
		cblogger.Infof("NCP VPC S/G No : [%s]", *sg.AccessControlGroupNo)
		sgInfo, sgInfoErr := securityHandler.MappingSecurityInfo(*sg)
		if sgInfoErr != nil {
			cblogger.Error(sgInfoErr)
			return nil, sgInfoErr
		}
		securityGroupList = append(securityGroupList, &sgInfo)
	}
	return securityGroupList, nil
}

func (securityHandler *NcpVpcSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC cloud driver: called DeleteSecurity()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(securityHandler.RegionInfo.Region, call.SECURITYGROUP, securityIID.SystemId, "DeleteSecurity()")

	if securityIID.SystemId == "" {
		createErr := fmt.Errorf("Invalid S/G SystemId")
		cblogger.Error(createErr.Error())
		LoggingError(callLogInfo, createErr)
		return false, createErr
	}

	// Check if the S/G exists
	ncpVpcSG, err := securityHandler.GetSecurity(securityIID)
	if err != nil {
		createErr := fmt.Errorf("Failed to Find any S/G info. with the SystemId : " + securityIID.SystemId)
		cblogger.Error(err.Error())
		cblogger.Error(createErr.Error())
		LoggingError(callLogInfo, createErr)
		return false, err
	}
	cblogger.Infof("The S/G Name to Delete : [%s]", ncpVpcSG.IId.NameId)

	// Delete the S/G
	sgDelReq := vserver.DeleteAccessControlGroupRequest {
		RegionCode: 				&securityHandler.RegionInfo.Region,
		VpcNo: 						&ncpVpcSG.VpcIID.SystemId,
		AccessControlGroupNo: 		&securityIID.SystemId,
	}

	callLogStart := call.Start()
	delResult, err := securityHandler.VMClient.V2Api.DeleteAccessControlGroup(&sgDelReq)
	if err != nil {
		cblogger.Error(err)
		cblogger.Error(*delResult.ReturnMessage)
		LoggingError(callLogInfo, err)
		return false, err
	}
	LoggingInfo(callLogInfo, callLogStart)

	cblogger.Infof("Succeeded in Deleting the S/G : [%s]", ncpVpcSG.IId.NameId)

	return true, nil
}

func (securityHandler *NcpVpcSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	cblogger.Info("NCP VPC cloud driver: called AddRules()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(securityHandler.RegionInfo.Region, call.SECURITYGROUP, sgIID.SystemId, "AddRules()")

	if sgIID.SystemId == "" {
		createErr := fmt.Errorf("Invalid S/G SystemId")
		cblogger.Error(createErr.Error())
		LoggingError(callLogInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}

	// Check if the S/G exists
	ncpVpcSG, err := securityHandler.GetSecurity(sgIID)
	if err != nil {
		createErr := fmt.Errorf("Failed to Find any S/G info. with the SystemId : " + sgIID.SystemId)
		cblogger.Error(err.Error())
		cblogger.Error(createErr.Error())
		LoggingError(callLogInfo, createErr)
		return irs.SecurityInfo{}, err
	}

	cblogger.Infof("The S/G Name to Add the Requested Rules : [%s]", ncpVpcSG.IId.NameId)

	sgNo := sgIID.SystemId
	vpcNo := ncpVpcSG.VpcIID.SystemId

	// Create S/G Rules
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

		var ruleDirection string
		if strings.EqualFold(curRule.Direction, "inbound") {
			ruleDirection = "inbound"
		} else if strings.EqualFold(curRule.Direction, "outbound") {
			ruleDirection = "outbound"
		} else {
			return irs.SecurityInfo{}, errors.New("Specified Invalid Security Rule Direction!!")
		}

		if strings.EqualFold(ruleDirection, "inbound") {
			protocolType := strings.ToUpper(curRule.IPProtocol) // Caution : Valid values : [TCP, UDP, ICMP]

			if strings.EqualFold(protocolType, "ALL") {
				if strings.EqualFold(curRule.FromPort, "-1") && strings.EqualFold(curRule.ToPort, "-1") {
					allProtocolTypeCode := []string {"TCP", "UDP", "ICMP"}
					var allPortRange string
					allCIDR := "0.0.0.0/0"

					for _, curProtocolType := range allProtocolTypeCode {
						if strings.EqualFold(curProtocolType, "ICMP") {
							allPortRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
						} else {
							allPortRange = "1-65535" // All Range
						}

						var ruleParameterList []*vserver.AddAccessControlGroupRuleParameter

						ruleParameter := vserver.AddAccessControlGroupRuleParameter {
							IpBlock: 					&allCIDR,
							PortRange:					&allPortRange,
							ProtocolTypeCode: 			&curProtocolType,
						}
			
						ruleParameterList = append(ruleParameterList, &ruleParameter)
			
						// Create Inbound Security Rule
						inboundReq := vserver.AddAccessControlGroupInboundRuleRequest {
							RegionCode: 				&securityHandler.RegionInfo.Region,
							AccessControlGroupNo: 		&sgNo,
							VpcNo: 						&vpcNo,
							AccessControlGroupRuleList: ruleParameterList,
						}
			
						callLogStart := call.Start()
						sgInboundRule, err := securityHandler.VMClient.V2Api.AddAccessControlGroupInboundRule(&inboundReq)
						if err != nil {
							cblogger.Error(err)
							cblogger.Error(*sgInboundRule.ReturnMessage)
							LoggingError(callLogInfo, err)
							return irs.SecurityInfo{}, err
						}
						LoggingInfo(callLogInfo, callLogStart)
			
						if *sgInboundRule.TotalRows < 1 {
							return irs.SecurityInfo{}, errors.New("Failed to Add New S/G Inbound Rule!!")
						} else {
							cblogger.Infof("Succeeded in Adding New [%s] Inbound Rule!!", curProtocolType)

							time.Sleep(time.Second * 5)  // Caution!!
						}
					}
				} else {
					return irs.SecurityInfo{}, errors.New("To Specify 'All Traffic Allow Rule', Specify '-1' as FromPort/ToPort!!")
				}
			} else {
				var portRange string
			
				if strings.EqualFold(curRule.IPProtocol, "ICMP") {
					portRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
				} else if strings.EqualFold(curRule.FromPort, curRule.ToPort) {
					portRange = strings.ToLower(curRule.FromPort)	// string				
				} else if !strings.EqualFold(curRule.FromPort, curRule.ToPort) {
					portRange = strings.ToLower(curRule.FromPort) + "-"+ strings.ToLower(curRule.ToPort) // string				
				}
	
				if strings.EqualFold(curRule.FromPort, "-1") || strings.EqualFold(curRule.ToPort, "-1") {
					if strings.EqualFold(curRule.IPProtocol, "ICMP") {
						portRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
					} else {
						portRange = "1-65535" // All Range
					}
				}
	
				var ruleParameterList []*vserver.AddAccessControlGroupRuleParameter
				ruleParameter := vserver.AddAccessControlGroupRuleParameter {
					IpBlock: 					&curRule.CIDR,
					PortRange:					&portRange,
					ProtocolTypeCode: 			&protocolType,
				}	
				ruleParameterList = append(ruleParameterList, &ruleParameter)
	
				// Create Inbound Security Rule
				inboundReq := vserver.AddAccessControlGroupInboundRuleRequest {
					RegionCode: 				&securityHandler.RegionInfo.Region,
					AccessControlGroupNo: 		&sgNo,
					VpcNo: 						&vpcNo,
					AccessControlGroupRuleList: ruleParameterList,
				}
	
				callLogStart := call.Start()
				sgInboundRule, err := securityHandler.VMClient.V2Api.AddAccessControlGroupInboundRule(&inboundReq)
				if err != nil {
					cblogger.Error(err)
					cblogger.Error(*sgInboundRule.ReturnMessage)
					LoggingError(callLogInfo, err)
					return irs.SecurityInfo{}, err
				}
				LoggingInfo(callLogInfo, callLogStart)
	
				if *sgInboundRule.TotalRows < 1 {
					return irs.SecurityInfo{}, errors.New("Failed to Add New S/G Inbound Rule!!")
				} else {
					cblogger.Infof("Succeeded in Adding New [%s] Inbound Rule!!", protocolType)					
				}
			}
		}

		if strings.EqualFold(ruleDirection, "outbound") {
			protocolType := strings.ToUpper(curRule.IPProtocol) // Caution : Valid values : [TCP, UDP, ICMP] 

			if strings.EqualFold(protocolType, "ALL") {
				if strings.EqualFold(curRule.FromPort, "-1") && strings.EqualFold(curRule.ToPort, "-1") {
					allProtocolTypeCode := []string {"TCP", "UDP", "ICMP"}
					var allPortRange string // All Range
					allCIDR := "0.0.0.0/0"

					for _, curProtocolType := range allProtocolTypeCode {
						if strings.EqualFold(curProtocolType, "ICMP") {
							allPortRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
						} else {
							allPortRange = "1-65535" // All Range
						}

						var ruleParameterList []*vserver.AddAccessControlGroupRuleParameter
		
						ruleParameter := vserver.AddAccessControlGroupRuleParameter {
							IpBlock: 					&allCIDR,
							PortRange:					&allPortRange,
							ProtocolTypeCode: 			&curProtocolType,
						}
			
						ruleParameterList = append(ruleParameterList, &ruleParameter)
			
						// Create Inbound Security Rule
						outboundReq := vserver.AddAccessControlGroupOutboundRuleRequest {
							RegionCode: 				&securityHandler.RegionInfo.Region,
							AccessControlGroupNo: 		&sgNo,
							VpcNo: 						&vpcNo,
							AccessControlGroupRuleList: ruleParameterList,
						}
			
						callLogStart := call.Start()
						sgOutboundRule, err := securityHandler.VMClient.V2Api.AddAccessControlGroupOutboundRule(&outboundReq)
						if err != nil {
							cblogger.Error(err)
							cblogger.Error(*sgOutboundRule.ReturnMessage)
							LoggingError(callLogInfo, err)
							return irs.SecurityInfo{}, err
						}
						LoggingInfo(callLogInfo, callLogStart)
			
						if *sgOutboundRule.TotalRows < 1 {
							return irs.SecurityInfo{}, errors.New("Failed to Add New S/G Outbound Rule!!")
						} else {
							cblogger.Infof("Succeeded in Adding New [%s] Outbound Rule!!", curProtocolType)

							time.Sleep(time.Second * 5)  // Caution!!
						}
					}
				} else {
					return irs.SecurityInfo{}, errors.New("To Specify 'All Traffic Allow Rule', Specify '-1' as FromPort/ToPort!!")
				}
			} else {
				var portRange string
			
				if strings.EqualFold(curRule.IPProtocol, "ICMP") {
					portRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
				} else if strings.EqualFold(curRule.FromPort, curRule.ToPort) {
					portRange = strings.ToLower(curRule.FromPort)	// string				
				} else if !strings.EqualFold(curRule.FromPort, curRule.ToPort) {
					portRange = strings.ToLower(curRule.FromPort) + "-"+ strings.ToLower(curRule.ToPort) // string				
				}

				if strings.EqualFold(curRule.FromPort, "-1") || strings.EqualFold(curRule.ToPort, "-1") {
					if strings.EqualFold(curRule.IPProtocol, "ICMP") {
						portRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
					} else {
						portRange = "1-65535" // All Range
					}
				}

				var ruleParameterList []*vserver.AddAccessControlGroupRuleParameter	
				ruleParameter := vserver.AddAccessControlGroupRuleParameter {
					IpBlock: 					&curRule.CIDR,
					PortRange:					&portRange,
					ProtocolTypeCode: 			&protocolType,
				}	
				ruleParameterList = append(ruleParameterList, &ruleParameter)
	
				// Create Inbound Security Rule
				outboundReq := vserver.AddAccessControlGroupOutboundRuleRequest {
					RegionCode: 				&securityHandler.RegionInfo.Region,
					AccessControlGroupNo: 		&sgNo,
					VpcNo: 						&vpcNo,
					AccessControlGroupRuleList: ruleParameterList,
				}
	
				callLogStart := call.Start()
				sgOutboundRule, err := securityHandler.VMClient.V2Api.AddAccessControlGroupOutboundRule(&outboundReq)
				if err != nil {
					cblogger.Error(err)
					cblogger.Error(*sgOutboundRule.ReturnMessage)
					LoggingError(callLogInfo, err)
					return irs.SecurityInfo{}, err
				}
				LoggingInfo(callLogInfo, callLogStart)
	
				if *sgOutboundRule.TotalRows < 1 {
					return irs.SecurityInfo{}, errors.New("Failed to Add New S/G Outbound Rule!!")
				} else {
					cblogger.Infof("Succeeded in Adding New [%s] Outbound Rule!!", protocolType)
				}
			}
		}
	}

	// Return Current S/G Info.
	sgInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: sgNo})
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		return irs.SecurityInfo{}, err
	}
	return sgInfo, nil
}

func (securityHandler *NcpVpcSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	cblogger.Info("NCP VPC cloud driver: called RemoveRules()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(securityHandler.RegionInfo.Region, call.SECURITYGROUP, sgIID.SystemId, "RemoveRules()")

	if sgIID.SystemId == "" {
		createErr := fmt.Errorf("Invalid S/G SystemId")
		cblogger.Error(createErr.Error())
		LoggingError(callLogInfo, createErr)
		return false, createErr
	}

	// Check if the S/G exists
	ncpVpcSG, err := securityHandler.GetSecurity(sgIID)
	if err != nil {
		createErr := fmt.Errorf("Failed to Find any S/G info. with the SystemId : " + sgIID.SystemId)
		cblogger.Error(err.Error())
		cblogger.Error(createErr.Error())
		LoggingError(callLogInfo, createErr)
		return false, err
	}

	cblogger.Infof("The S/G Name to Remove the Requested Rules : [%s]", ncpVpcSG.IId.NameId)

	sgNo := sgIID.SystemId
	vpcNo := ncpVpcSG.VpcIID.SystemId

	// Create S/G Rules
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

		var ruleDirection string
		if strings.EqualFold(curRule.Direction, "inbound") {
			ruleDirection = "inbound"
		} else if strings.EqualFold(curRule.Direction, "outbound") {
			ruleDirection = "outbound"
		} else {
			return false, errors.New("Specified Invalid Security Rule Direction!!")
		}

		if strings.EqualFold(ruleDirection, "inbound") {
			protocolType := strings.ToUpper(curRule.IPProtocol) // Caution : Valid values : [TCP, UDP, ICMP]

			if strings.EqualFold(protocolType, "ALL") {
				if strings.EqualFold(curRule.FromPort, "-1") && strings.EqualFold(curRule.ToPort, "-1") {
					allProtocolTypeCode := []string {"TCP", "UDP", "ICMP"}
					var allPortRange string
					allCIDR := "0.0.0.0/0"

					for _, curProtocolType := range allProtocolTypeCode {
						if strings.EqualFold(curProtocolType, "ICMP") {
							allPortRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
						} else {
							allPortRange = "1-65535" // All Range
						}

						var ruleParameterList []*vserver.RemoveAccessControlGroupRuleParameter
						ruleParameter := vserver.RemoveAccessControlGroupRuleParameter {
							IpBlock: 					&allCIDR,
							PortRange:					&allPortRange,
							ProtocolTypeCode: 			&curProtocolType,
						}			
						ruleParameterList = append(ruleParameterList, &ruleParameter)
			
						// Create Inbound Security Rule
						inboundReq := vserver.RemoveAccessControlGroupInboundRuleRequest {
							RegionCode: 				&securityHandler.RegionInfo.Region,
							AccessControlGroupNo: 		&sgNo,
							VpcNo: 						&vpcNo,
							AccessControlGroupRuleList: ruleParameterList,
						}
			
						callLogStart := call.Start()
						sgInboundRule, err := securityHandler.VMClient.V2Api.RemoveAccessControlGroupInboundRule(&inboundReq)
						if err != nil {
							cblogger.Error(err)
							cblogger.Error(*sgInboundRule.ReturnMessage)
							LoggingError(callLogInfo, err)
							return false, err
						}
						LoggingInfo(callLogInfo, callLogStart)
			
						cblogger.Infof("Succeeded in Removing the [%s] Inbound Rule!!", curProtocolType)

						time.Sleep(time.Second * 5)  // Caution!!
					}
				} else {
					return false, errors.New("To Specify 'All Traffic Allow Rule', Specify '-1' as FromPort/ToPort!!")
				}
			} else {
				var portRange string
			
				if strings.EqualFold(curRule.IPProtocol, "ICMP") {
					portRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
				} else if strings.EqualFold(curRule.FromPort, curRule.ToPort) {
					portRange = strings.ToLower(curRule.FromPort)	// string				
				} else if !strings.EqualFold(curRule.FromPort, curRule.ToPort) {
					portRange = strings.ToLower(curRule.FromPort) + "-"+ strings.ToLower(curRule.ToPort) // string				
				}

				if strings.EqualFold(curRule.FromPort, "-1") || strings.EqualFold(curRule.ToPort, "-1") {
					if strings.EqualFold(curRule.IPProtocol, "ICMP") {
						portRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
					} else {
						portRange = "1-65535" // All Range
					}
				}
	
				var ruleParameterList []*vserver.RemoveAccessControlGroupRuleParameter
				ruleParameter := vserver.RemoveAccessControlGroupRuleParameter {
					IpBlock: 					&curRule.CIDR,
					PortRange:					&portRange,
					ProtocolTypeCode: 			&protocolType,
				}	
				ruleParameterList = append(ruleParameterList, &ruleParameter)
	
				// Create Inbound Security Rule
				inboundReq := vserver.RemoveAccessControlGroupInboundRuleRequest {
					RegionCode: 				&securityHandler.RegionInfo.Region,
					AccessControlGroupNo: 		&sgNo,
					VpcNo: 						&vpcNo,
					AccessControlGroupRuleList: ruleParameterList,
				}
	
				callLogStart := call.Start()
				sgInboundRule, err := securityHandler.VMClient.V2Api.RemoveAccessControlGroupInboundRule(&inboundReq)
				if err != nil {
					cblogger.Error(err)
					cblogger.Error(*sgInboundRule.ReturnMessage)
					LoggingError(callLogInfo, err)
					return false, err
				}
				LoggingInfo(callLogInfo, callLogStart)
	
				cblogger.Infof("Succeeded in Removing the [%s] Inbound Rule!!", protocolType)
			}
		}

		if strings.EqualFold(ruleDirection, "outbound") {
			protocolType := strings.ToUpper(curRule.IPProtocol) // Caution : Valid values : [TCP, UDP, ICMP] 

			if strings.EqualFold(protocolType, "ALL") {
				if strings.EqualFold(curRule.FromPort, "-1") && strings.EqualFold(curRule.ToPort, "-1") {
					allProtocolTypeCode := []string {"TCP", "UDP", "ICMP"}
					var allPortRange string // All Range
					allCIDR := "0.0.0.0/0"

					for _, curProtocolType := range allProtocolTypeCode {
						if strings.EqualFold(curProtocolType, "ICMP") {
							allPortRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
						} else {
							allPortRange = "1-65535" // All Range
						}

						var ruleParameterList []*vserver.RemoveAccessControlGroupRuleParameter		
						ruleParameter := vserver.RemoveAccessControlGroupRuleParameter {
							IpBlock: 					&allCIDR,
							PortRange:					&allPortRange,
							ProtocolTypeCode: 			&curProtocolType,
						}			
						ruleParameterList = append(ruleParameterList, &ruleParameter)
			
						// Create Inbound Security Rule
						outboundReq := vserver.RemoveAccessControlGroupOutboundRuleRequest {
							RegionCode: 				&securityHandler.RegionInfo.Region,
							AccessControlGroupNo: 		&sgNo,
							VpcNo: 						&vpcNo,
							AccessControlGroupRuleList: ruleParameterList,
						}
			
						callLogStart := call.Start()
						sgOutboundRule, err := securityHandler.VMClient.V2Api.RemoveAccessControlGroupOutboundRule(&outboundReq)
						if err != nil {
							cblogger.Error(err)
							cblogger.Error(*sgOutboundRule.ReturnMessage)
							LoggingError(callLogInfo, err)
							return false, err
						}
						LoggingInfo(callLogInfo, callLogStart)
			
						cblogger.Infof("Succeeded in Removing the [%s] Outbound Rule!!", curProtocolType)

						time.Sleep(time.Second * 5)  // Caution!!
					}
				} else {
					return false, errors.New("To Specify 'All Traffic Allow Rule', Specify '-1' as FromPort/ToPort!!")
				}
			} else {
				var portRange string
			
				if strings.EqualFold(curRule.IPProtocol, "ICMP") {
					portRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
				} else if strings.EqualFold(curRule.FromPort, curRule.ToPort) {
					portRange = strings.ToLower(curRule.FromPort)	// string				
				} else if !strings.EqualFold(curRule.FromPort, curRule.ToPort) {
					portRange = strings.ToLower(curRule.FromPort) + "-"+ strings.ToLower(curRule.ToPort) // string				
				}
	
				if strings.EqualFold(curRule.FromPort, "-1") || strings.EqualFold(curRule.ToPort, "-1") {
					if strings.EqualFold(curRule.IPProtocol, "ICMP") {
						portRange = ""		// string. In case protocolTypeCode is 'ICMP', Do Not input Value
					} else {
						portRange = "1-65535" // All Range
					}
				}

				var ruleParameterList []*vserver.RemoveAccessControlGroupRuleParameter	
				ruleParameter := vserver.RemoveAccessControlGroupRuleParameter {
					IpBlock: 					&curRule.CIDR,
					PortRange:					&portRange,
					ProtocolTypeCode: 			&protocolType,
				}	
				ruleParameterList = append(ruleParameterList, &ruleParameter)
	
				// Create Inbound Security Rule
				outboundReq := vserver.RemoveAccessControlGroupOutboundRuleRequest {
					RegionCode: 				&securityHandler.RegionInfo.Region,
					AccessControlGroupNo: 		&sgNo,
					VpcNo: 						&vpcNo,
					AccessControlGroupRuleList: ruleParameterList,
				}
	
				callLogStart := call.Start()
				sgOutboundRule, err := securityHandler.VMClient.V2Api.RemoveAccessControlGroupOutboundRule(&outboundReq)
				if err != nil {
					cblogger.Error(err)
					cblogger.Error(*sgOutboundRule.ReturnMessage)
					LoggingError(callLogInfo, err)
					return false, err
				}
				LoggingInfo(callLogInfo, callLogStart)
	
				cblogger.Infof("Succeeded in Removing the [%s] Outbound Rule!!", protocolType)
			}
		}
	}

	// Return Current S/G Info.
	sgInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: sgNo})
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(callLogInfo, err)
		return false, err
	}
	spew.Dump(sgInfo)

	return true, nil
}

func (securityHandler *NcpVpcSecurityHandler) OpenOutboundAllProtocol(sgIID irs.IID) (error) {
	cblogger.Info("NCP VPC cloud driver: called OpenOutboundAllProtocol()!")
	
	InitLog() // Caution!!
    callLogInfo := GetCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, sgIID.SystemId, "OpenOutboundAllProtocol()")

	reqRules := []irs.SecurityRuleInfo {
			{
				Direction: 		"outbound",
				IPProtocol:  	"ALL",
				FromPort: 		"-1",
				ToPort: 		"-1",
				CIDR: 			"0.0.0.0/0",		
			},
		}	
	_, err := securityHandler.AddRules(sgIID, &reqRules)
	if err != nil {
		newErr := fmt.Errorf("Failed to Add Outbound Protocol Opening Rule. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return newErr
	}

	return nil
}

func (securityHandler *NcpVpcSecurityHandler) MappingSecurityInfo(ncpVpcSG vserver.AccessControlGroup) (irs.SecurityInfo, error) {
	cblogger.Info("NCP VPC cloud driver: called MappingSecurityInfo()!")

	sgRuleList, ruleInfoErr := securityHandler.ExtractSecurityRuleInfo(ncloud.StringValue(ncpVpcSG.AccessControlGroupNo))
	if ruleInfoErr != nil {
		cblogger.Error(ruleInfoErr)
		return irs.SecurityInfo{}, ruleInfoErr
	}

	securityInfo := irs.SecurityInfo{
		IId: 		irs.IID{NameId: ncloud.StringValue(ncpVpcSG.AccessControlGroupName), SystemId: ncloud.StringValue(ncpVpcSG.AccessControlGroupNo)},
		VpcIID: 	irs.IID{SystemId: ncloud.StringValue(ncpVpcSG.VpcNo)},
		SecurityRules: &sgRuleList,

		KeyValueList: []irs.KeyValue{
			{Key: "SecurityGroupDescription", Value: ncloud.StringValue(ncpVpcSG.AccessControlGroupDescription)},
			{Key: "SecurityGroupStatus", Value: ncloud.StringValue(ncpVpcSG.AccessControlGroupStatus.CodeName)},
			{Key: "IsDefault", Value: strconv.FormatBool(*ncpVpcSG.IsDefault)},
		},
	}
	return securityInfo, nil
}

// Search NCP VPC SecurityGroup Rule List
func (securityHandler *NcpVpcSecurityHandler) ExtractSecurityRuleInfo(ncpVpcSGId string) ([]irs.SecurityRuleInfo, error) {
	var securityRuleList []irs.SecurityRuleInfo

	sgRuleReq := vserver.GetAccessControlGroupRuleListRequest {
		RegionCode: 			&securityHandler.RegionInfo.Region,
		AccessControlGroupNo: 	ncloud.String(ncpVpcSGId),
	}
	ruleListResult, err := securityHandler.VMClient.V2Api.GetAccessControlGroupRuleList(&sgRuleReq)
	if err != nil {
		cblogger.Error(*ruleListResult.ReturnMessage)
		return nil, err
	}
	if *ruleListResult.TotalRows < 1 {
		cblogger.Infof("$$$ The S/G [%s] contains No Rule!!", ncpVpcSGId)
		return nil, nil // Caution!!
	}

	for _, curRule := range ruleListResult.AccessControlGroupRuleList {
		curRuleInfo := irs.SecurityRuleInfo {
			IPProtocol: ncloud.StringValue(curRule.ProtocolType.CodeName),
			Direction: 	strings.ToLower(ncloud.StringValue(curRule.AccessControlGroupRuleType.CodeName)),			
			CIDR: 		ncloud.StringValue(curRule.IpBlock),
		}

		if strings.EqualFold(curRuleInfo.IPProtocol, "icmp") {
			curRuleInfo.FromPort = "-1"		// string
			curRuleInfo.ToPort = "-1"		// string
		} else if strings.Contains(ncloud.StringValue(curRule.PortRange), "-") {
			portNo := strings.Split(ncloud.StringValue(curRule.PortRange), "-")

			curRuleInfo.FromPort = portNo[0]		// string
			curRuleInfo.ToPort = portNo[1]	// string
		} else {
			curRuleInfo.FromPort = ncloud.StringValue(curRule.PortRange)
			curRuleInfo.ToPort = ncloud.StringValue(curRule.PortRange)
		}

		securityRuleList = append(securityRuleList, curRuleInfo)
	}
	return securityRuleList, nil
}
