// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// program by ysjeon@mz.co.kr, 2019.07.
// modify by devunet@mz.co.kr, 2019.11.

package resources

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	compute "google.golang.org/api/compute/v1"
)

type GCPSecurityHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

const (
	Const_SecurityRule_Add    = "add"
	Const_SecurityRule_Remove = "remove"

	Const_Firewall_Allow = true
	Const_Firewall_Deny  = false

	Const_GCP_Direction_INGRESS = "INGRESS"
	Const_GCP_Direction_EGRESS  = "EGRESS"

	Const_Spider_Direction_INBOUND  = "inbound"
	Const_Spider_Direction_OUTBOUND = "outbound"

	Const_IPPROTOCOL_ALL  = "ALL"
	Const_IPPROTOCOL_TCP  = "TCP"
	Const_IPPROTOCOL_UDP  = "UDP"
	Const_IPPROTOCOL_ICMP = "ICMP"
	Const_IPPROTOCOL_ETC  = "ETC"
)

//+ 공통이슈개발방안
//. spider 는 rule 간의 priority 제공안함
//. 동일한 rule의 중복 정의를 제공하지 않음
//. 동일한 rule 추가 요청시 오류
//. 존재하지 않는 rule 삭제 요청시 오류
//. 동일한 rule 판단 조건 : direction, ipprotocol, from port, toport, cidr
//. 여러 rule을 포함한 추가 요청시 기존 rule과 중복된 rule이 하나이상 포함되어있는 경우
//.. 이미 존재하는 rule 정보와 함께 에러 메세지 반환
//.. 존재하는 rule이 하나 이상의 경우에도 에러 메시지에 모두 포함하여 반환
//.. AddRules/RemoveRules에서 모든 rule에대해 존재하는지 check

//+ GCP Issue 방안
//. GCP는 Security Group(SG) 개념 없음.
//. GCP는 개별 Firewall을 설정, 복수개의 Firewall을 vm에 적용
//. SG 제공 방안 : tag를 sg 단위로 생각하고 각 rule을 firewall로 각각 추가
//.. SG에 Rule이 존재하지 않는 경우 사용되지 않는 0port를 포함하는 firewall을 유지 : GCP는 빈 firewall ruleset을 허용하지 않음으로
//.. direction 및 cidr는 하나의 firewall에서 하나의 direction, cidr 만 사용가능 : direction, cidr이 다르게 오면 여러개의 firewall을 추가하는 것으로? -> 관리가 가능한가? 문의
//.. valid sg name 은 maxlen=63-6 으로 57자까지. 6자는 ‘-basic’, ‘-I-xxx’, ‘-o-xxx’ 를 붙임.
//Ex) sg name = sg-test 일 때 firewallname은 조합하여 만들고, tag로 sg를 묶는다                   inbound TCP/22/22/0.0.0.0/0 이면  firewall name=sg-test-basic, tag=sg-test   : 0번 포트는 무조건 추가
//Inbound TCP/80/80/0.0.0.0/0 이면 firewall name=sg-test-i-001, tag=sg-test   : inbound rule 첫번째
//Outbound UDP/1000/1000/1.2.3.4/32 이면 firewall name=sg-test-o-001, tag=sg-test : outbound rule 첫번째
//-> GCP CreateSecurityRule 에 default 추가

// TODO : old로직 남겨놓음 완성되면 삭제 처리
//func (securityHandler *GCPSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
//	cblogger.Info(securityReqInfo)
//
//	vNetworkHandler := GCPVPCHandler{
//		Client:     securityHandler.Client,
//		Region:     securityHandler.Region,
//		Ctx:        securityHandler.Ctx,
//		Credential: securityHandler.Credential,
//	}
//
//	vNetInfo, errVnet := vNetworkHandler.GetVPC(securityReqInfo.VpcIID)
//	spew.Dump(vNetInfo)
//	if errVnet != nil {
//		cblogger.Error(errVnet)
//		return irs.SecurityInfo{}, errVnet
//	}
//
//	if len(*securityReqInfo.SecurityRules) < 1 {
//		return irs.SecurityInfo{}, errors.New("invalid value - The SecurityRules policy to add is empty")
//	}
//
//	//GCP의 경우 1개의 보안그룹에 Inbound나 Outbound를 1개만 지정할 수 있으며 CIDR도 1개의 보안그룹에 1개만 공통으로 지정됨.
//	//즉, 1개의 보안 정책에 다중 포트를 선언하는 형태라서  irs.SecurityReqInfo의 정보를 사용할 것인지
//	// irs.SecurityReqInfo의 *[]SecurityRuleInfo 배열의 첫 번째 값을 사용할 것인지 미정이라 공통 변수를 만들어서 처리함.
//	commonPolicy := *securityReqInfo.SecurityRules
//	commonDirection := commonPolicy[0].Direction
//	commonCidr := strings.Split(commonPolicy[0].CIDR, ",")
//
//	if len(commonCidr[0]) < 2 {
//		return irs.SecurityInfo{}, errors.New("invalid value - The CIDR is empty")
//	}
//
//	projectID := securityHandler.Credential.ProjectID
//	// @TODO: SecurityGroup 생성 요청 파라미터 정의 필요
//	ports := *securityReqInfo.SecurityRules
//	var firewallAllowed []*compute.FirewallAllowed
//
//	//다른 드라이버와의 통일을 위해 All은 -1로 처리함.
//	//GCP는 포트 번호를 적지 않으면 All임.
//	//GCP 방화벽 정책
//	//https://cloud.google.com/vpc/docs/firewalls?hl=ko&_ga=2.238147008.-1577666838.1589162755#protocols_and_ports
//	for _, item := range ports {
//		var port string
//		fp := item.FromPort
//		tp := item.ToPort
//
//		//GCP는 1개의 정책에 1가지 Direction만 지정 가능하기 때문에 Inbound와 Outbound 모두 지정되었을 경우 에러 처리함.
//		if !strings.EqualFold(item.Direction, commonDirection) {
//			return irs.SecurityInfo{}, errors.New("invalid value - GCP can only use one Direction for one security policy")
//		}
//
//		// CB Rule에 의해 Port 번호에 -1이 기입된 경우 GCP Rule에 맞게 치환함.
//		if fp == "-1" || tp == "-1" {
//			if (fp == "-1" && tp == "-1") || (fp == "-1" && tp == "") || (fp == "" && tp == "-1") {
//				port = ""
//			} else if fp == "-1" {
//				port = tp
//			} else {
//				port = fp
//			}
//		} else {
//			//둘 다 있는 경우
//			if tp != "" && fp != "" {
//				port = fp + "-" + tp
//				//From Port가 없는 경우
//			} else if tp != "" && fp == "" {
//				port = tp
//				//To Port가 없는 경우
//			} else if tp == "" && fp != "" {
//				port = fp
//			} else {
//				port = ""
//			}
//		}
//
//		if port == "" {
//			firewallAllowed = append(firewallAllowed, &compute.FirewallAllowed{
//				IPProtocol: item.IPProtocol,
//			})
//		} else {
//			firewallAllowed = append(firewallAllowed, &compute.FirewallAllowed{
//				IPProtocol: item.IPProtocol,
//				Ports: []string{
//					port,
//				},
//			})
//		}
//	}
//
//	if strings.EqualFold(commonDirection, "inbound") || strings.EqualFold(commonDirection, "INGRESS") {
//		commonDirection = "INGRESS"
//	} else if strings.EqualFold(commonDirection, "outbound") || strings.EqualFold(commonDirection, "EGRESS") {
//		commonDirection = "EGRESS"
//	} else {
//		return irs.SecurityInfo{}, errors.New("invalid value - The direction[" + commonDirection + "] information is unknown")
//	}
//
//	prefix := "https://www.googleapis.com/compute/v1/projects/" + projectID
//	//networkURL := prefix + "/global/networks/" + securityReqInfo.VpcIID.NameId
//	networkURL := prefix + "/global/networks/" + securityReqInfo.VpcIID.SystemId
//
//	fireWall := &compute.Firewall{
//		Allowed:   firewallAllowed,
//		Direction: commonDirection, //INGRESS(inbound), EGRESS(outbound)
//		// SourceRanges: []string{
//		// 	// "0.0.0.0/0",
//		// 	commonCidr,
//		// },
//		Name: securityReqInfo.IId.NameId,
//		TargetTags: []string{
//			securityReqInfo.IId.NameId,
//		},
//		Network: networkURL,
//	}
//
//	//CIDR 처리
//	if strings.EqualFold(commonDirection, "INGRESS") {
//		//fireWall.SourceRanges = []string{commonCidr}
//		fireWall.SourceRanges = commonCidr
//	} else {
//		//fireWall.DestinationRanges = []string{commonCidr}
//		fireWall.DestinationRanges = commonCidr
//	}
//
//	cblogger.Info("생성할 방화벽 정책")
//	cblogger.Debug(fireWall)
//	//spew.Dump(fireWall)
//
//	// logger for HisCall
//	callogger := call.GetLogger("HISCALL")
//	callLogInfo := call.CLOUDLOGSCHEMA{
//		CloudOS:      call.GCP,
//		RegionZone:   securityHandler.Region.Zone,
//		ResourceType: call.SECURITYGROUP,
//		ResourceName: securityReqInfo.IId.NameId,
//		CloudOSAPI:   "Firewalls.Insert()",
//		ElapsedTime:  "",
//		ErrorMSG:     "",
//	}
//	callLogStart := call.Start()
//
//	res, err := securityHandler.Client.Firewalls.Insert(projectID, fireWall).Do()
//	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
//	if err != nil {
//		callLogInfo.ErrorMSG = err.Error()
//		callogger.Error(call.String(callLogInfo))
//		cblogger.Error(err)
//		return irs.SecurityInfo{}, err
//	}
//	callogger.Info(call.String(callLogInfo))
//	cblogger.Debug("create result : ", res)
//	time.Sleep(time.Second * 3)
//	//secInfo, _ := securityHandler.GetSecurity(securityReqInfo.IId)
//	secInfo, _ := securityHandler.GetSecurity(irs.IID{SystemId: securityReqInfo.IId.NameId})
//	return secInfo, nil
//}
// securityGroup = GCP 의 Tag

/*
SecurityGroup 생성. GCP는 firewall 추가 시 tag = securityGroupName
.GCP 기본 정책이 outbound에 대해 all allow이므로
  - 우선순위가 가장 낮은(65535) all deny  outbound rule 추가
  - 우선순위 = 100 인 all allow outbound rule 추가

.사용자의 요청에서 outbound all open 이 있는 경우. default로 생성하므로 skip
*/
func (securityHandler *GCPSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Info(securityReqInfo)

	var addFilewallList []compute.Firewall // 추가할 firewall 목록
	var errorFirewallList []string         // 에러발생시 error 항목을 담을 목록

	vNetworkHandler := GCPVPCHandler{
		Client:     securityHandler.Client,
		Region:     securityHandler.Region,
		Ctx:        securityHandler.Ctx,
		Credential: securityHandler.Credential,
	}

	vNetInfo, errVnet := vNetworkHandler.GetVPC(securityReqInfo.VpcIID)
	spew.Dump(vNetInfo)
	if errVnet != nil {
		cblogger.Error(errVnet)
		return irs.SecurityInfo{}, errVnet
	}

	if len(*securityReqInfo.SecurityRules) < 1 {
		return irs.SecurityInfo{}, errors.New("invalid value - The SecurityRules policy to add is empty")
	}

	// 해당 securityGroup Tag가 존재하는지 check

	projectID := securityHandler.Credential.ProjectID

	// default firewall 추가 : default로 inbound 1개(-basic), outbound 1개(-o-001)
	reqEgressCount := 1
	reqIngressCount := 1

	cblogger.Info("기본outbound deny 추가")
	_, err := securityHandler.insertDefaultOutboundPolicy(projectID, securityReqInfo.VpcIID.SystemId, securityReqInfo.IId.NameId, reqEgressCount)
	if err != nil {

	}
	//defaultOutboundDenySecurityRuleInfo := irs.SecurityRuleInfo{
	//	FromPort:   "", // 지정하지 않으면 전체임.
	//	IPProtocol: "ALL",
	//	Direction:  "EGRESS",
	//	CIDR:       "0.0.0.0/0",
	//}
	//defaultOutboundDenyFireWall := setNewFirewall(defaultOutboundDenySecurityRuleInfo, projectID, securityReqInfo.VpcIID.SystemId, securityReqInfo.IId.NameId, "-o-", reqEgressCount, Const_Firewall_Deny)
	//defaultOutboundDenyFireWall.Priority = 65535 // defaultFirewall의 우선순위는 가장 낮게: ALL Deny
	//_, err := securityHandler.firewallInsert(defaultOutboundDenyFireWall)
	//if err != nil {
	//	cblogger.Debug(err)
	//	return irs.SecurityInfo{}, err
	//}
	//cblogger.Debug(defaultOutboundDenyFireWall)

	cblogger.Info("기본outbound allow 추가")
	reqEgressCount++ // count 증가

	defaultOutboundAllowSecurityRuleInfo := irs.SecurityRuleInfo{
		FromPort:   "", // 지정하지 않으면 전체임.
		IPProtocol: Const_IPPROTOCOL_ALL,
		Direction:  Const_GCP_Direction_EGRESS,
		CIDR:       "0.0.0.0/0",
	}
	defaultOutboundAllowFireWall, err := setNewFirewall(defaultOutboundAllowSecurityRuleInfo, projectID, securityReqInfo.VpcIID.SystemId, securityReqInfo.IId.NameId, reqEgressCount, Const_Firewall_Allow)
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	defaultOutboundAllowFireWall.Priority = 1000 // defaultFirewall의 우선순위는 가장 낮게: ALL Deny
	_, err = securityHandler.firewallInsert(defaultOutboundAllowFireWall)
	if err != nil {
		cblogger.Debug(err)
		return irs.SecurityInfo{}, err
	}
	cblogger.Debug(defaultOutboundAllowFireWall)
	reqEgressCount++ // count 증가

	reqSecurityRules := *securityReqInfo.SecurityRules

	for itemIndex, item := range reqSecurityRules {
		firewallFromPort := item.FromPort
		firewallToPort := item.ToPort
		firewallIPProtocol := item.IPProtocol
		firewallCIDR := item.CIDR

		firewallDirection := switchDirectionSpiderAndGCP(item.Direction, "GCP") // GCP로 날 릴 때에는 "GCP", SPIDER에서 사용할 때에는 "SPIDER"

		// SecurityGroup 생성 시. outbound에대한 allow/deny all을 정의하기 떄문에 동일한 요청이 있으면 skip
		//FromPort:   "-1",
		//ToPort:     "-1",
		//IPProtocol: "all",
		//Direction:  "outbound",
		//CIDR:       "0.0.0.0/0",
		//cblogger.Debug("default firewallFromPort : ", firewallFromPort)
		//cblogger.Debug("default firewallToPort : ", firewallToPort)
		//cblogger.Debug("default firewallIPProtocol : ", firewallIPProtocol)
		//cblogger.Debug("default firewallDirection : ", firewallDirection)
		//cblogger.Debug("default firewallCIDR : ", firewallCIDR)

		// outbound all open는 생성시 자동으로 추가하므로 사용자 요청이 있으면 skip한다.
		if strings.EqualFold(firewallFromPort, "-1") && strings.EqualFold(firewallToPort, "-1") && strings.EqualFold(firewallIPProtocol, "all") && strings.EqualFold(firewallDirection, Const_GCP_Direction_EGRESS) && strings.EqualFold(firewallCIDR, "0.0.0.0/0") {
			cblogger.Info("outbound all opened rule already exists. continue")
			errorFirewallList = append(errorFirewallList, "outbound all opened rule already exists. continue")
			continue
		}

		var fireWall compute.Firewall
		if strings.EqualFold(firewallDirection, Const_GCP_Direction_INGRESS) {
			fireWall, err = setNewFirewall(item, projectID, securityReqInfo.VpcIID.SystemId, securityReqInfo.IId.NameId, reqIngressCount, Const_Firewall_Allow)
			if err != nil {
				errorFirewallList = append(errorFirewallList, err.Error())
			}
			reqIngressCount++
		} else if strings.EqualFold(firewallDirection, Const_GCP_Direction_EGRESS) {
			fireWall, err = setNewFirewall(item, projectID, securityReqInfo.VpcIID.SystemId, securityReqInfo.IId.NameId, reqEgressCount, Const_Firewall_Allow)
			if err != nil {
				errorFirewallList = append(errorFirewallList, err.Error())
			}
			reqEgressCount++
		} else {
			// direction 이 없는데.... continue
			errorFirewallList = append(errorFirewallList, "there no direction")
			continue
		}

		cblogger.Info("생성할 방화벽 정책 ", itemIndex, firewallDirection, reqEgressCount, reqIngressCount)
		cblogger.Debug(fireWall)
		//spew.Dump(fireWall)

		addFilewallList = append(addFilewallList, fireWall)

	}

	if len(errorFirewallList) > 0 {
		return irs.SecurityInfo{}, errors.New(strings.Join(errorFirewallList, ","))
	}

	for _, addFirewall := range addFilewallList {
		_, err := securityHandler.firewallInsert(addFirewall)
		if err != nil {
			errorFirewallList = append(errorFirewallList, err.Error())
			return irs.SecurityInfo{}, errors.New(strings.Join(errorFirewallList, ","))
		}
	}

	securityInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: securityReqInfo.IId.NameId})
	//securityInfo, _ := securityHandler.GetSecurityByTag(irs.IID{SystemId: securityReqInfo.IId.NameId})
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	// tag기반으로 security를 묶어서 가져와야 함.
	return securityInfo, nil
	//return irs.SecurityInfo{}, nil
}

// func getOperationsStatus(securityHandler GCPSecurityHandler, projectID string, operationName string, operationType string) {
func (securityHandler GCPSecurityHandler) getOperationsStatus(ch chan string, projectID string, operationName string, operationType string) {
	// global : firewall
	// region : vpc
	//operation2, err := Client.GlobalOperations.Get(projectID, res.Name).Do()

	errWait := securityHandler.WaitUntilComplete(operationName)
	if errWait != nil {
		cblogger.Errorf("SecurityGroup create 완료 대기 실패")
		cblogger.Error(errWait)
	}
	cblogger.Debug("getOperationsStatus ", operationName)
	ch <- operationName
	//waitGroup.Done()
}

// firewall rule 설정.
// direction, port 마다 1개의 firewall로.
func setNewFirewall(ruleInfo irs.SecurityRuleInfo, projectID string, vpcSystemId string, securityGroupName string, sequence int, isAllow bool) (compute.Firewall, error) {

	port, err := setFromPortToPort(ruleInfo.IPProtocol, ruleInfo.FromPort, ruleInfo.ToPort)
	if err != nil {
		return compute.Firewall{}, err
	}

	var firewallAllowed []*compute.FirewallAllowed
	var firewallDenied []*compute.FirewallDenied

	if isAllow {
		if port == "" {
			firewallAllowed = append(firewallAllowed, &compute.FirewallAllowed{
				IPProtocol: ruleInfo.IPProtocol,
			})
		} else {
			firewallAllowed = append(firewallAllowed, &compute.FirewallAllowed{
				IPProtocol: ruleInfo.IPProtocol,
				Ports: []string{
					port,
				},
			})
		}
	} else {
		if port == "" {
			firewallDenied = append(firewallDenied, &compute.FirewallDenied{
				IPProtocol: ruleInfo.IPProtocol,
			})
		} else {
			firewallDenied = append(firewallDenied, &compute.FirewallDenied{
				IPProtocol: ruleInfo.IPProtocol,
				Ports: []string{
					port,
				},
			})
		}
	}

	cidr := ruleInfo.CIDR
	firewallDirection := switchDirectionSpiderAndGCP(ruleInfo.Direction, "GCP") // GCP로 날 릴 때에는 "GCP", SPIDER에서 사용할 때에는 "SPIDER"

	prefix := "https://www.googleapis.com/compute/v1/projects/" + projectID
	networkURL := prefix + "/global/networks/" + vpcSystemId

	// default = -basic, inbound = -i-xxx, outbount = -o-xxx
	firewallName := ""
	if strings.EqualFold(firewallDirection, Const_GCP_Direction_INGRESS) {
		sequenceStr := lpad(strconv.Itoa(sequence), "0", 3)
		firewallName = securityGroupName + "-i-" + sequenceStr
	} else if strings.EqualFold(firewallDirection, Const_GCP_Direction_EGRESS) {
		sequenceStr := lpad(strconv.Itoa(sequence), "0", 3)
		firewallName = securityGroupName + "-o-" + sequenceStr
	}

	//if strings.EqualFold(firewallType, "-i-") {
	//	sequenceStr := lpad(strconv.Itoa(sequence), "0", 3)
	//	firewallName = securityGroupName + "-i-" + sequenceStr
	//	firewallDirection = "INGRESS"
	//
	//} else if strings.EqualFold(firewallType, "-o-") {
	//	cblogger.Debug("create sequence : ", sequence, strconv.Itoa(sequence))
	//	sequenceStr := lpad(strconv.Itoa(sequence), "0", 3)
	//	firewallName = securityGroupName + "-o-" + sequenceStr
	//	firewallDirection = "EGRESS"
	//} else {
	//	firewallName = securityGroupName + "-basic"
	//	firewallDirection = "INGRESS"
	//}

	fireWall := compute.Firewall{
		Name:      firewallName,
		Allowed:   firewallAllowed,
		Denied:    firewallDenied,
		Direction: firewallDirection,
		Network:   networkURL,
		TargetTags: []string{
			securityGroupName,
		},
	}

	//CIDR 처리 : ingress=>sourceRanges, egress=>destination  둘 중 하나만 선택 가능
	if strings.EqualFold(firewallDirection, Const_GCP_Direction_INGRESS) {
		fireWall.SourceRanges = []string{cidr}
	} else if strings.EqualFold(firewallDirection, Const_GCP_Direction_EGRESS) {
		fireWall.DestinationRanges = []string{cidr}
	}

	cblogger.Debug("firewallset : ", fireWall)
	return fireWall, nil
}

// ipProtocol에 따른 port 값 set.
// all : from=-1, to=-1
// tcp : from= 1~65535, to=1~65535
// udp : from= 1~65535, to=1~65535  (GCP는 미지정 시 전체로 가능하나 Spider는 1~65535 로 쓰기로 함 )
// icmp : from=-1, to=-1

func setFromPortToPort(ipProtocol string, fromPort string, toPort string) (string, error) {
	returnPort := ""
	if strings.EqualFold(ipProtocol, "all") || strings.EqualFold(ipProtocol, "icmp") {
		returnPort = ""
	} else if strings.EqualFold(ipProtocol, "tcp") || strings.EqualFold(ipProtocol, "udp") {
		// fromPort, toPort 는 1 ~ 65535
		fp, err := strconv.ParseInt(fromPort, 0, 64)
		if err != nil {
			return "", err
		}
		if fp < 1 || fp > 65535 {
			return "", errors.New("invalid value - port range : 1~65535 but fromPort is " + fromPort + ". ")
		}

		tp, err := strconv.ParseInt(toPort, 0, 64)
		if err != nil {
			return "", err
		}
		if tp < 1 || tp > 65535 {
			return "", errors.New("invalid value - port range : 1~65535 but toPort is " + toPort + ". ")
		}

		if fromPort == "-1" || toPort == "-1" {
			if (fromPort == "-1" && toPort == "-1") || (fromPort == "-1" && toPort == "") || (fromPort == "" && toPort == "-1") {
				returnPort = ""
			} else if fromPort == "-1" {
				returnPort = toPort
			} else {
				returnPort = fromPort
			}
		} else {
			//둘 다 있는 경우
			if toPort != "" && fromPort != "" {
				returnPort = fromPort + "-" + toPort
				//From Port가 없는 경우
			} else if toPort != "" && fromPort == "" {
				returnPort = toPort
				//To Port가 없는 경우
			} else if toPort == "" && fromPort != "" {
				returnPort = fromPort
			} else {
				returnPort = ""
			}
		}
	}
	return returnPort, nil
}

//func setFromPortToPort(fp string, tp string) string {
//	var port string
//	if fp == "-1" || tp == "-1" {
//		if (fp == "-1" && tp == "-1") || (fp == "-1" && tp == "") || (fp == "" && tp == "-1") {
//			port = ""
//		} else if fp == "-1" {
//			port = tp
//		} else {
//			port = fp
//		}
//	} else {
//		//둘 다 있는 경우
//		if tp != "" && fp != "" {
//			port = fp + "-" + tp
//			//From Port가 없는 경우
//		} else if tp != "" && fp == "" {
//			port = tp
//			//To Port가 없는 경우
//		} else if tp == "" && fp != "" {
//			port = fp
//		} else {
//			port = ""
//		}
//	}
//	return port
//}

// string 원본, 앞에 붙일 값, 전체 길이
func lpad(sequence string, pad string, plength int) string {
	for i := len(sequence); i < plength; i++ {
		sequence = pad + sequence
	}
	return sequence
}

func (securityHandler *GCPSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {

	firewallList, err := securityHandler.firewallList("")
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	var securityInfoList []*irs.SecurityInfo
	for _, firewallInfo := range firewallList {
		securityInfo, err := convertFromFirewallToSecurityInfo(firewallInfo)
		if err != nil {
			//500  convert Error
			return nil, err
		}
		securityInfoList = append(securityInfoList, &securityInfo)
	}
	cblogger.Debug("securityInfoList = ", securityInfoList)

	return securityInfoList, nil
}

// TAG를 이용해서 해당 security(firewall)를 모두 가와야 하기 때문에
// 해당 project의 모든 list에서 해당 하는 TAG를 추출
func (securityHandler *GCPSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {

	securityGroupTag := securityIID.SystemId

	//// Inbound는 sourceTag에서 outbound는 targetTag에서 추출.
	firewallList, err := securityHandler.firewallList(securityGroupTag)
	if err != nil {
		cblogger.Error(err)
		return irs.SecurityInfo{}, err
	}
	var securityInfo irs.SecurityInfo
	for _, firewallInfo := range firewallList {
		itemName := ""
		// itemName 에서 비교할 이름 추출 : tag가 있으면 Tag, 없으면 item.Name
		for _, item := range firewallInfo.Items {
			// tag가 있으면 tag로 조회
			sourceTag := getTagFromTags(item.Name, item.SourceTags)
			if sourceTag != "" {
				itemName = sourceTag
				break
			}
			targetTag := getTagFromTags(item.Name, item.TargetTags)
			if targetTag != "" {
				itemName = targetTag
				break
			}
			itemName = item.Name
		}

		if strings.EqualFold(itemName, securityGroupTag) {
			tempSecurityInfo, err := convertFromFirewallToSecurityInfo(firewallInfo) // securityInfo로 변환. securityInfo에 이름이 있어서 해당 이름 사용
			if err != nil {
				//500  convert Error
				return irs.SecurityInfo{}, err
			}
			securityInfo = tempSecurityInfo
			break
		}
	}

	cblogger.Debug("securityInfo : ", securityInfo)
	return securityInfo, nil
}

//func (securityHandler *GCPSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
//	projectID := securityHandler.Credential.ProjectID
//
//	// logger for HisCall
//	callogger := call.GetLogger("HISCALL")
//	callLogInfo := call.CLOUDLOGSCHEMA{
//		CloudOS:      call.GCP,
//		RegionZone:   securityHandler.Region.Zone,
//		ResourceType: call.SECURITYGROUP,
//		ResourceName: securityIID.SystemId,
//		CloudOSAPI:   "Firewalls.Get()",
//		ElapsedTime:  "",
//		ErrorMSG:     "",
//	}
//	callLogStart := call.Start()
//	security, err := securityHandler.Client.Firewalls.Get(projectID, securityIID.SystemId).Do()
//	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
//	if err != nil {
//		callLogInfo.ErrorMSG = err.Error()
//		callogger.Info(call.String(callLogInfo))
//		cblogger.Error(err)
//		return irs.SecurityInfo{}, err
//	}
//	callogger.Info(call.String(callLogInfo))
//
//	var commonCidr string
//	if strings.EqualFold(security.Direction, "INGRESS") {
//		commonCidr = strings.Join(security.SourceRanges, ", ")
//	} else {
//		commonCidr = strings.Join(security.DestinationRanges, ", ")
//	}
//
//	var securityRules []irs.SecurityRuleInfo
//	for _, item := range security.Allowed {
//		var portArr []string
//		var fromPort string
//		var toPort string
//		if ports := item.Ports; ports != nil {
//			portArr = strings.Split(item.Ports[0], "-")
//			fromPort = portArr[0]
//			if len(portArr) > 1 {
//				toPort = portArr[len(portArr)-1]
//			} else {
//				toPort = ""
//			}
//
//		} else {
//			fromPort = ""
//			toPort = ""
//		}
//
//		securityRules = append(securityRules, irs.SecurityRuleInfo{
//			FromPort:   fromPort,
//			ToPort:     toPort,
//			IPProtocol: item.IPProtocol,
//			Direction:  security.Direction,
//			CIDR:       commonCidr,
//		})
//	}
//	vpcArr := strings.Split(security.Network, "/")
//	vpcName := vpcArr[len(vpcArr)-1]
//	securityInfo := irs.SecurityInfo{
//		IId: irs.IID{
//			NameId: security.Name,
//			//SystemId: strconv.FormatUint(security.Id, 10),
//			SystemId: security.Name,
//		},
//		VpcIID: irs.IID{
//			NameId:   vpcName,
//			SystemId: vpcName,
//		},
//
//		// Direction: security.Direction,
//		KeyValueList: []irs.KeyValue{
//			{Key: "Priority", Value: strconv.FormatInt(security.Priority, 10)},
//			// {"SourceRanges", security.SourceRanges[0]},
//			{Key: "Allowed", Value: security.Allowed[0].IPProtocol},
//			{Key: "Vpc", Value: vpcName},
//		},
//		SecurityRules: &securityRules,
//	}
//
//	return securityInfo, nil
//}

// SecurityGroup 삭제 (해당 Tag를 가진 firewall 삭제)
func (securityHandler *GCPSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	//projectID := securityHandler.Credential.ProjectID
	securityGroupTag := securityIID.SystemId
	//var vpcIID irs.IID

	cblogger.Debug("Delete Security ", securityGroupTag)
	// 해당 Tag를 가진 목록 조회
	firewallList, err := securityHandler.firewallList(securityGroupTag)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	cblogger.Debug("Delete Security 삭제 대상 ", len(firewallList))
	for index, firewallInfo := range firewallList {
		//if index == 0 {
		//	tempSecurityInfo, err := convertFromFirewallToSecurityInfo(firewallInfo) // securityInfo로 변환. securityInfo에 이름이 있어서 해당 이름 사용
		//	if err != nil {
		//		//500  convert Error
		//		return false, err
		//	}
		//	vpcIID = tempSecurityInfo.VpcIID
		//}
		cblogger.Debug("Delete Security 삭제 대상item ", len(firewallInfo.Items), " index = ", index)
		securityHandler.firewallDelete(securityGroupTag, "", firewallInfo)
		if err != nil {
			//500  convert Error
			return false, err
		}
	}

	//// 전체 삭제 후 all deny 추가 : securityGroup을 삭제하는 로직이라 추가할 필요 없음.
	//_, err = securityHandler.insertDefaultOutboundPolicy(projectID, vpcIID.SystemId, securityIID.NameId, 1)
	//if err != nil {
	//	return false, err
	//}
	return true, nil
}

// GCP의 outbound는 ALL Allow 이기 때문에 ALL Deny rule 추가. 우선순위=65535로 낮게.
func (securityHandler *GCPSecurityHandler) insertDefaultOutboundPolicy(projectID string, vpcID string, securityID string, egressCount int) (bool, error) {

	cblogger.Info("기본outbound ")
	defaultOutboundDenySecurityRuleInfo := irs.SecurityRuleInfo{
		FromPort:   "", // 지정하지 않으면 전체임.
		IPProtocol: "ALL",
		Direction:  "EGRESS",
		CIDR:       "0.0.0.0/0",
	}
	defaultOutboundDenyFireWall, err := setNewFirewall(defaultOutboundDenySecurityRuleInfo, projectID, vpcID, securityID, egressCount, Const_Firewall_Deny)
	if err != nil {
		return false, err
	}
	defaultOutboundDenyFireWall.Priority = 65535 // defaultFirewall의 우선순위는 가장 낮게: ALL Deny
	_, err = securityHandler.firewallInsert(defaultOutboundDenyFireWall)
	if err != nil {
		cblogger.Debug(err)
		return false, err
	}
	cblogger.Debug(defaultOutboundDenyFireWall)
	return true, nil
}

//func (securityHandler *GCPSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
//	projectID := securityHandler.Credential.ProjectID
//
//	// logger for HisCall
//	callogger := call.GetLogger("HISCALL")
//	callLogInfo := call.CLOUDLOGSCHEMA{
//		CloudOS:      call.GCP,
//		RegionZone:   securityHandler.Region.Zone,
//		ResourceType: call.SECURITYGROUP,
//		ResourceName: securityIID.SystemId,
//		CloudOSAPI:   "CreateVpc()",
//		ElapsedTime:  "",
//		ErrorMSG:     "",
//	}
//	callLogStart := call.Start()
//	res, err := securityHandler.Client.Firewalls.Delete(projectID, securityIID.SystemId).Do()
//	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
//	if err != nil {
//		callLogInfo.ErrorMSG = err.Error()
//		callogger.Info(call.String(callLogInfo))
//		cblogger.Error(err)
//		return false, err
//	}
//	callogger.Info(call.String(callLogInfo))
//	cblogger.Debug(res)
//	return true, nil
//}

func (securityHandler *GCPSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	cblogger.Info(*securityRules)

	projectID := securityHandler.Credential.ProjectID
	securityGroupTag := sgIID.SystemId
	vpcId := ""
	existsAllDenyOutbound := false

	var addFilewallList []compute.Firewall // 추가할 firewall 목록
	var errorFirewallList []string         // 에러발생시 error 항목을 담을 목록

	// 기존에 존재하는지
	firewallList, err := securityHandler.firewallList(securityGroupTag)
	if err != nil {
		cblogger.Error(err)
		return irs.SecurityInfo{}, err
	}

	// securityInfo 추출
	var searchSecurityInfo irs.SecurityInfo
	var tempSecurityRules []irs.SecurityRuleInfo

	for _, firewallInfo := range firewallList {
		tempSecurityInfo, err := convertFromFirewallToSecurityInfo(firewallInfo) // securityInfo로 변환. securityInfo에 이름이 있어서 해당 이름 사용
		if err != nil {
			//500  convert Error
			return irs.SecurityInfo{}, err
		}
		for _, ruleInfo := range *tempSecurityInfo.SecurityRules {
			tempSecurityRules = append(tempSecurityRules, ruleInfo)
		}

		// 기본정책인 deny all 이 존재하는지 check, 없으면 추가시킴.
		if !existsAllDenyOutbound { // 찾아서 true인 경우는 다시 찾을 필요 없음.
			for _, firewallItem := range firewallInfo.Items {
				cidr := strings.Join(firewallItem.DestinationRanges, ", ")
				if strings.Index(cidr, "0.0.0.0/0") == -1 {
					continue
				}
				if strings.EqualFold(firewallItem.Direction, Const_GCP_Direction_INGRESS) { // Egress만 체크
					continue
				}

				for _, firewallDeny := range firewallItem.Denied {
					if strings.EqualFold(firewallDeny.IPProtocol, "all") && len(firewallDeny.Ports) == 0 {
						existsAllDenyOutbound = true
						break
					}
				}
			}
		}
		searchSecurityInfo.VpcIID = tempSecurityInfo.VpcIID
		vpcId = tempSecurityInfo.VpcIID.SystemId
	}
	searchSecurityInfo.SecurityRules = &tempSecurityRules

	// 동일한 rule이 존재하면 존재하는 목록 return
	sameRuleList := sameRuleCheck(searchSecurityInfo.SecurityRules, securityRules, Const_SecurityRule_Add)
	if len(*sameRuleList) > 0 {
		return irs.SecurityInfo{}, errors.New("Same SecurityRule exists")
	}

	// 존재하는 item의 max Sequence 찾아와야 함
	reqIngressCount := maxFirewallSequence(firewallList, Const_GCP_Direction_INGRESS)
	reqEgressCount := maxFirewallSequence(firewallList, Const_GCP_Direction_EGRESS)

	reqIngressCount++
	reqEgressCount++

	for _, item := range *securityRules {
		firewallDirection := switchDirectionSpiderAndGCP(item.Direction, "GCP") // GCP로 날 릴 때에는 "GCP", SPIDER에서 사용할 때에는 "SPIDER"

		var fireWall compute.Firewall
		if strings.EqualFold(firewallDirection, Const_GCP_Direction_INGRESS) {
			fireWall, err = setNewFirewall(item, projectID, searchSecurityInfo.VpcIID.SystemId, securityGroupTag, reqIngressCount, Const_Firewall_Allow)
			if err != nil {
				errorFirewallList = append(errorFirewallList, err.Error())
			}
			reqIngressCount++
		} else if strings.EqualFold(firewallDirection, Const_GCP_Direction_EGRESS) {
			fireWall, err = setNewFirewall(item, projectID, searchSecurityInfo.VpcIID.SystemId, securityGroupTag, reqEgressCount, Const_Firewall_Allow)
			if err != nil {
				errorFirewallList = append(errorFirewallList, err.Error())
			}
			reqEgressCount++
		} else {
			// direction 이 없는데.... continue
			cblogger.Debug("no direction : ", firewallDirection)
			errorFirewallList = append(errorFirewallList, "there is no direction ")
			continue
		}

		addFilewallList = append(addFilewallList, fireWall)
	}

	if len(errorFirewallList) > 0 {
		return irs.SecurityInfo{}, errors.New(strings.Join(errorFirewallList, ","))
	}

	for _, addFirewall := range addFilewallList {
		_, err := securityHandler.firewallInsert(addFirewall)
		if err != nil {
			errorFirewallList = append(errorFirewallList, err.Error())
			return irs.SecurityInfo{}, errors.New(strings.Join(errorFirewallList, ","))
		}
	}

	// All Deny Outboun가  없으면 추가한다.
	cblogger.Debug("existsAllDenyOutbound ----------------- ", existsAllDenyOutbound)
	if !existsAllDenyOutbound {
		cblogger.Info("default outbound all deny does not exist, create one")
		maxEgessCount := maxFirewallSequence(firewallList, Const_GCP_Direction_EGRESS)
		maxEgessCount++
		_, err = securityHandler.insertDefaultOutboundPolicy(projectID, vpcId, securityGroupTag, maxEgessCount)
	}
	return securityHandler.GetSecurity(sgIID)
}

//func (securityHandler *GCPSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
//	cblogger.Info(*securityRules)
//
//	projectID := securityHandler.Credential.ProjectID
//
//	security, err := securityHandler.Client.Firewalls.Get(projectID, sgIID.SystemId).Do()
//	vpcArr := strings.Split(security.Network, "/")
//	vpcName := vpcArr[len(vpcArr)-1]
//
//	if len(*securityRules) < 1 {
//		return irs.SecurityInfo{}, errors.New("invalid value - The SecurityRules policy to add is empty")
//	}
//
//	//GCP의 경우 1개의 보안그룹에 Inbound나 Outbound를 1개만 지정할 수 있으며 CIDR도 1개의 보안그룹에 1개만 공통으로 지정됨.
//	//즉, 1개의 보안 정책에 다중 포트를 선언하는 형태라서  irs.SecurityReqInfo의 정보를 사용할 것인지
//	// irs.SecurityReqInfo의 *[]SecurityRuleInfo 배열의 첫 번째 값을 사용할 것인지 미정이라 공통 변수를 만들어서 처리함.
//	commonPolicy := *securityRules
//	commonDirection := commonPolicy[0].Direction
//	commonCidr := strings.Split(commonPolicy[0].CIDR, ",")
//
//	if len(commonCidr[0]) < 2 {
//		return irs.SecurityInfo{}, errors.New("invalid value - The CIDR is empty")
//	}
//
//	// @TODO: SecurityGroup 생성 요청 파라미터 정의 필요
//	ports := *securityRules
//	existedRules := security.Allowed
//	var firewallAllowed []*compute.FirewallAllowed
//
//	for _, rule := range existedRules {
//		firewallAllowed = append(firewallAllowed, rule)
//	}
//
//	//다른 드라이버와의 통일을 위해 All은 -1로 처리함.
//	//GCP는 포트 번호를 적지 않으면 All임.
//	//GCP 방화벽 정책
//	//https://cloud.google.com/vpc/docs/firewalls?hl=ko&_ga=2.238147008.-1577666838.1589162755#protocols_and_ports
//	for _, item := range ports {
//		var port string
//		fp := item.FromPort
//		tp := item.ToPort
//
//		//GCP는 1개의 정책에 1가지 Direction만 지정 가능하기 때문에 Inbound와 Outbound 모두 지정되었을 경우 에러 처리함.
//		if !strings.EqualFold(item.Direction, commonDirection) {
//			return irs.SecurityInfo{}, errors.New("invalid value - GCP can only use one Direction for one security policy")
//		}
//
//		// CB Rule에 의해 Port 번호에 -1이 기입된 경우 GCP Rule에 맞게 치환함.
//		if fp == "-1" || tp == "-1" {
//			if (fp == "-1" && tp == "-1") || (fp == "-1" && tp == "") || (fp == "" && tp == "-1") {
//				port = ""
//			} else if fp == "-1" {
//				port = tp
//			} else {
//				port = fp
//			}
//		} else {
//			//둘 다 있는 경우
//			if tp != "" && fp != "" {
//				port = fp + "-" + tp
//				//From Port가 없는 경우
//			} else if tp != "" && fp == "" {
//				port = tp
//				//To Port가 없는 경우
//			} else if tp == "" && fp != "" {
//				port = fp
//			} else {
//				port = ""
//			}
//		}
//
//		if port == "" {
//			firewallAllowed = append(firewallAllowed, &compute.FirewallAllowed{
//				IPProtocol: item.IPProtocol,
//			})
//		} else {
//			firewallAllowed = append(firewallAllowed, &compute.FirewallAllowed{
//				IPProtocol: item.IPProtocol,
//				Ports: []string{
//					port,
//				},
//			})
//		}
//	}
//
//	if strings.EqualFold(commonDirection, "inbound") || strings.EqualFold(commonDirection, "INGRESS") {
//		commonDirection = "INGRESS"
//	} else if strings.EqualFold(commonDirection, "outbound") || strings.EqualFold(commonDirection, "EGRESS") {
//		commonDirection = "EGRESS"
//	} else {
//		return irs.SecurityInfo{}, errors.New("invalid value - The direction[" + commonDirection + "] information is unknown")
//	}
//
//	if !strings.EqualFold(security.Direction, commonDirection) {
//		return irs.SecurityInfo{}, errors.New("invalid value - GCP can only use one Direction for one security policy")
//	}
//
//	prefix := "https://www.googleapis.com/compute/v1/projects/" + projectID
//	//networkURL := prefix + "/global/networks/" + securityReqInfo.VpcIID.NameId
//	networkURL := prefix + "/global/networks/" + vpcName
//
//	fireWall := &compute.Firewall{
//		Allowed:   firewallAllowed,
//		Direction: commonDirection, //INGRESS(inbound), EGRESS(outbound)
//		// SourceRanges: []string{
//		// 	// "0.0.0.0/0",
//		// 	commonCidr,
//		// },
//		//Name: security.Name,
//		//TargetTags: []string{
//		//security.Name,
//		//},
//		Network: networkURL,
//	}
//
//	//CIDR 처리
//	if strings.EqualFold(commonDirection, "INGRESS") {
//		//fireWall.SourceRanges = []string{commonCidr}
//		fireWall.SourceRanges = commonCidr
//	} else {
//		//fireWall.DestinationRanges = []string{commonCidr}
//		fireWall.DestinationRanges = commonCidr
//	}
//
//	cblogger.Info("생성할 방화벽 정책")
//	cblogger.Debug(fireWall)
//	//spew.Dump(fireWall)
//
//	// logger for HisCall
//	callogger := call.GetLogger("HISCALL")
//	callLogInfo := call.CLOUDLOGSCHEMA{
//		CloudOS:      call.GCP,
//		RegionZone:   securityHandler.Region.Zone,
//		ResourceType: call.SECURITYGROUP,
//		ResourceName: security.Name,
//		CloudOSAPI:   "Firewalls.Update()",
//		ElapsedTime:  "",
//		ErrorMSG:     "",
//	}
//	callLogStart := call.Start()
//
//	res, err := securityHandler.Client.Firewalls.Update(projectID, sgIID.SystemId, fireWall).Do()
//	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
//	if err != nil {
//		callLogInfo.ErrorMSG = err.Error()
//		callogger.Error(call.String(callLogInfo))
//		cblogger.Error(err)
//		return irs.SecurityInfo{}, err
//	}
//	callogger.Info(call.String(callLogInfo))
//	cblogger.Debug("create result : ", res)
//	time.Sleep(time.Second * 3)
//	//secInfo, _ := securityHandler.GetSecurity(securityReqInfo.IId)
//	secInfo, _ := securityHandler.GetSecurity(irs.IID{SystemId: sgIID.SystemId})
//	return secInfo, nil
//	//return irs.SecurityInfo{}, fmt.Errorf("Coming Soon!")
//}

// 요청받은 Security 그룹안의 SecurityRule이 동일한 firewall 삭제
// 추가가 allow만 가능 하므로 삭제도 allow만 가능
func (securityHandler *GCPSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	cblogger.Info(*securityRules)

	projectID := securityHandler.Credential.ProjectID
	securityGroupTag := sgIID.SystemId
	existsAllDenyOutbound := false
	vpcId := ""

	firewallList, err := securityHandler.firewallList(securityGroupTag)
	if err != nil {
		return false, err
	}

	var searchSecurityInfo irs.SecurityInfo
	var tempSecurityRules []irs.SecurityRuleInfo

	for _, firewallInfo := range firewallList {
		tempSecurityInfo, err := convertFromFirewallToSecurityInfo(firewallInfo) // securityInfo로 변환. securityInfo에 이름이 있어서 해당 이름 사용
		if err != nil {
			//500  convert Error
			return false, err
		}
		for _, ruleInfo := range *tempSecurityInfo.SecurityRules {
			tempSecurityRules = append(tempSecurityRules, ruleInfo)
		}
		searchSecurityInfo.VpcIID = tempSecurityInfo.VpcIID
		vpcId = tempSecurityInfo.VpcIID.SystemId

		// 기본정책인 deny all 이 존재하는지 check, 없으면 추가시킴.
		if !existsAllDenyOutbound { // 찾아서 true인 경우는 다시 찾을 필요 없음.
			for _, firewallItem := range firewallInfo.Items {
				cidr := strings.Join(firewallItem.DestinationRanges, ", ")
				if strings.Index(cidr, "0.0.0.0/0") == -1 {
					continue
				}
				if strings.EqualFold(firewallItem.Direction, Const_GCP_Direction_INGRESS) { // Egress만 체크
					continue
				}

				for _, firewallDeny := range firewallItem.Denied {
					if strings.EqualFold(firewallDeny.IPProtocol, "all") && len(firewallDeny.Ports) == 0 {
						existsAllDenyOutbound = true
						break
					}
				}
			}
		}
	}
	searchSecurityInfo.SecurityRules = &tempSecurityRules

	// 동일한 rule이 존재하지 않으면 지울 수 없으므로 존재하는 않는 요청 목록 return
	sameRuleList := sameRuleCheck(searchSecurityInfo.SecurityRules, securityRules, Const_SecurityRule_Remove)
	if len(*sameRuleList) > 0 {
		return false, errors.New("Same SecurityRule does not exist")
	}

	for _, securityRule := range *securityRules {
		// firewall 삭제를 위한 resource ID 추출
		resourceId := ""
		for _, firewallInfo := range firewallList {
			for _, item := range firewallInfo.Items {
				var portArr []string
				var fromPort string
				var toPort string
				var ipProtocol string

				cidr := ""
				spiderDirection := switchDirectionSpiderAndGCP(item.Direction, "SPIDER") // GCP로 날 릴 때에는 "GCP", SPIDER에서 사용할 때에는 "SPIDER"
				if strings.EqualFold(spiderDirection, Const_Spider_Direction_OUTBOUND) {
					cidr = strings.Join(item.DestinationRanges, ", ")
				} else {
					cidr = strings.Join(item.SourceRanges, ", ")
				}

				for _, firewallRule := range item.Allowed {
					cblogger.Debug("firewallRule : ", firewallRule)
					if ports := firewallRule.Ports; ports != nil {

						portArr = strings.Split(firewallRule.Ports[0], "-")
						fromPort = portArr[0]
						if len(portArr) > 1 {
							toPort = portArr[len(portArr)-1]
						} else {
							toPort = ""
						}

					} else { // insert에서는 없으면 빼고, delete에서는 없으면 넣는다.
						fromPort = "-1"
						toPort = "-1"
					}

					ipProtocol = firewallRule.IPProtocol
				} // end of firewall rule

				securityFromPort := securityRule.FromPort
				securityToPort := securityRule.ToPort
				if strings.EqualFold(securityFromPort, securityToPort) && !strings.EqualFold(securityFromPort, "-1") && !strings.EqualFold(securityToPort, "-1") {
					securityToPort = ""
				}

				if !strings.EqualFold(spiderDirection, securityRule.Direction) {
					continue
				}
				if !strings.EqualFold(cidr, securityRule.CIDR) {
					continue
				}
				if !strings.EqualFold(ipProtocol, securityRule.IPProtocol) {
					continue
				}

				// 조건이 동일한 resource ID
				if strings.EqualFold(fromPort, securityFromPort) && strings.EqualFold(toPort, securityToPort) {
					resourceId = item.Name
					break
				}
			}

			if strings.EqualFold(resourceId, "") {
				//return false, errors.New("Cannot get a resourceID")
				cblogger.Debug("cannot get a resourceID : ")
				continue
			}

			// 삭제 호출
			_, err := securityHandler.firewallDelete(securityGroupTag, resourceId, firewallInfo)
			if err != nil {
				return false, err
			}
		}
	}

	// All Deny Outboun가  없으면 추가한다.
	cblogger.Debug("existsAllDenyOutbound ----------------- ", existsAllDenyOutbound)
	if !existsAllDenyOutbound {
		cblogger.Info("default outbound all deny does not exist, create one")
		maxEgessCount := maxFirewallSequence(firewallList, Const_GCP_Direction_EGRESS)
		maxEgessCount++
		_, err = securityHandler.insertDefaultOutboundPolicy(projectID, vpcId, securityGroupTag, maxEgessCount)
	}
	return true, nil
}

//func (securityHandler *GCPSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
//	cblogger.Info(*securityRules)
//
//	projectID := securityHandler.Credential.ProjectID
//
//	security, err := securityHandler.Client.Firewalls.Get(projectID, sgIID.SystemId).Do()
//	vpcArr := strings.Split(security.Network, "/")
//	vpcName := vpcArr[len(vpcArr)-1]
//
//	if len(*securityRules) < 1 {
//		return false, errors.New("invalid value - The SecurityRules policy to delete is empty")
//	}
//
//	//GCP의 경우 1개의 보안그룹에 Inbound나 Outbound를 1개만 지정할 수 있으며 CIDR도 1개의 보안그룹에 1개만 공통으로 지정됨.
//	//즉, 1개의 보안 정책에 다중 포트를 선언하는 형태라서  irs.SecurityReqInfo의 정보를 사용할 것인지
//	// irs.SecurityReqInfo의 *[]SecurityRuleInfo 배열의 첫 번째 값을 사용할 것인지 미정이라 공통 변수를 만들어서 처리함.
//	commonPolicy := security
//	commonDirection := commonPolicy.Direction
//	commonCidr := commonPolicy.SourceRanges
//	if strings.EqualFold(commonPolicy.Direction, "EGRESS") {
//		commonCidr = security.DestinationRanges
//	}
//
//	if len(commonCidr[0]) < 2 {
//		return false, errors.New("invalid value - The CIDR is empty")
//	}
//
//	// @TODO: SecurityGroup 생성 요청 파라미터 정의 필요
//	ports := *securityRules
//	existedAllowed := security.Allowed
//	var firewallAllowed []*compute.FirewallAllowed
//	var newFirewallAllowed []*compute.FirewallAllowed
//
//	//다른 드라이버와의 통일을 위해 All은 -1로 처리함.
//	//GCP는 포트 번호를 적지 않으면 All임.
//	//GCP 방화벽 정책
//	//https://cloud.google.com/vpc/docs/firewalls?hl=ko&_ga=2.238147008.-1577666838.1589162755#protocols_and_ports
//	for _, item := range ports {
//		var port string
//		fp := item.FromPort
//		tp := item.ToPort
//
//		// CB Rule에 의해 Port 번호에 -1이 기입된 경우 GCP Rule에 맞게 치환함.
//		if fp == "-1" || tp == "-1" {
//			if (fp == "-1" && tp == "-1") || (fp == "-1" && tp == "") || (fp == "" && tp == "-1") {
//				port = ""
//			} else if fp == "-1" {
//				port = tp
//			} else {
//				port = fp
//			}
//		} else {
//			//둘 다 있는 경우
//			if tp != "" && fp != "" {
//				port = fp + "-" + tp
//				if tp == fp {
//					port = tp
//				}
//				//From Port가 없는 경우
//			} else if tp != "" && fp == "" {
//				port = tp
//				//To Port가 없는 경우
//			} else if tp == "" && fp != "" {
//				port = fp
//			} else {
//				port = ""
//			}
//		}
//
//		if strings.EqualFold(item.Direction, "inbound") || strings.EqualFold(item.Direction, "INGRESS") {
//			item.Direction = "INGRESS"
//		} else if strings.EqualFold(item.Direction, "outbound") || strings.EqualFold(item.Direction, "EGRESS") {
//			item.Direction = "EGRESS"
//		} else {
//			return false, errors.New("invalid value - The direction[" + item.Direction + "] information is unknown")
//		}
//
//		if strings.EqualFold(commonDirection, item.Direction) && strings.EqualFold(commonCidr[0], item.CIDR) {
//			if port == "" {
//				firewallAllowed = append(firewallAllowed, &compute.FirewallAllowed{
//					IPProtocol: item.IPProtocol,
//				})
//			} else {
//				firewallAllowed = append(firewallAllowed, &compute.FirewallAllowed{
//					IPProtocol: item.IPProtocol,
//					Ports: []string{
//						port,
//					},
//				})
//			}
//		}
//	}
//
//	// 삭제할 rule을 제외시킨 새로운 firewallAllowed 생성, firewallAllowed == 삭제하려는 rule의 모음
//	for _, rule := range existedAllowed {
//		count := 0
//		for _, deleteRule := range firewallAllowed {
//			if len(deleteRule.Ports) == 0 && strings.EqualFold(rule.IPProtocol, deleteRule.IPProtocol) { // port값이 없는 경우
//				break
//			} else if strings.EqualFold(rule.IPProtocol, deleteRule.IPProtocol) && strings.EqualFold(rule.Ports[0], deleteRule.Ports[0]) {
//				break
//			}
//			count++
//		}
//
//		if len(firewallAllowed) != 0 && count < len(firewallAllowed) { // 삭제하려는 rule이 존재하는 경우
//			continue
//		}
//
//		newFirewallAllowed = append(newFirewallAllowed, &compute.FirewallAllowed{
//			IPProtocol: rule.IPProtocol,
//			Ports:      rule.Ports,
//		})
//
//	}
//
//	if len(newFirewallAllowed) == 0 {
//		return false, errors.New("invalid value - Must specify at least one rule")
//	}
//
//	prefix := "https://www.googleapis.com/compute/v1/projects/" + projectID
//	//networkURL := prefix + "/global/networks/" + securityReqInfo.VpcIID.NameId
//	networkURL := prefix + "/global/networks/" + vpcName
//
//	fireWall := &compute.Firewall{
//		Allowed:   newFirewallAllowed,
//		Direction: commonDirection, //INGRESS(inbound), EGRESS(outbound)
//		// SourceRanges: []string{
//		// 	// "0.0.0.0/0",
//		// 	commonCidr,
//		// },
//		//Name: security.Name,
//		//TargetTags: []string{
//		//security.Name,
//		//},
//		Network: networkURL,
//	}
//
//	//CIDR 처리
//	if strings.EqualFold(commonDirection, "INGRESS") {
//		//fireWall.SourceRanges = []string{commonCidr}
//		fireWall.SourceRanges = commonCidr
//	} else {
//		//fireWall.DestinationRanges = []string{commonCidr}
//		fireWall.DestinationRanges = commonCidr
//	}
//
//	cblogger.Info("생성할 방화벽 정책")
//	cblogger.Debug(fireWall)
//	//spew.Dump(fireWall)
//
//	// logger for HisCall
//	callogger := call.GetLogger("HISCALL")
//	callLogInfo := call.CLOUDLOGSCHEMA{
//		CloudOS:      call.GCP,
//		RegionZone:   securityHandler.Region.Zone,
//		ResourceType: call.SECURITYGROUP,
//		ResourceName: security.Name,
//		CloudOSAPI:   "Firewalls.Update()",
//		ElapsedTime:  "",
//		ErrorMSG:     "",
//	}
//	callLogStart := call.Start()
//
//	res, err := securityHandler.Client.Firewalls.Update(projectID, sgIID.SystemId, fireWall).Do()
//	cblogger.Info(res)
//	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
//	if err != nil {
//		callLogInfo.ErrorMSG = err.Error()
//		callogger.Error(call.String(callLogInfo))
//		cblogger.Error(err)
//		return false, err
//	}
//	callogger.Info(call.String(callLogInfo))
//	//cblogger.Debug("create result : ", res)
//	time.Sleep(time.Second * 3)
//	//secInfo, _ := securityHandler.GetSecurity(securityReqInfo.IId)
//	secInfo, _ := securityHandler.GetSecurity(irs.IID{SystemId: sgIID.SystemId})
//	cblogger.Info(secInfo)
//	return true, nil
//	//return irs.SecurityInfo{}, fmt.Errorf("Coming Soon!")
//}

// tag가 있으면 해당 이름을 return
func getFirewallNameFromTags(itemName string, tags []string) string {
	// 해당 tag에 param이 있는가
	for _, tag := range tags {
		// naming rule에 의해 itemName 은 tag + surfix 로 구성되므로 tag가 itemName에 있어야 함.
		if tag != "" && strings.Index(itemName, tag) == 0 {
			if strings.Index(itemName, tag+"-basic") == 0 {
				return itemName
			}
			if strings.Index(itemName, tag+"-i-") == 0 {
				return itemName
			}
			if strings.Index(itemName, tag+"-o-") == 0 {
				return itemName
			}
		}
	}
	return ""
}

func getTagFromTags(itemName string, tags []string) string {
	// 해당 tag에 param이 있는가
	for _, tag := range tags {
		// naming rule에 의해 itemName 은 tag + surfix 로 구성되므로 tag가 itemName에 있어야 함.
		cblogger.Debug("itemName : ", itemName, tag, strings.Index(itemName, tag))
		if tag != "" && strings.Index(itemName, tag) == 0 {
			if strings.Index(itemName, tag+"-basic") == 0 {
				return tag
			}
			if strings.Index(itemName, tag+"-i-") == 0 {
				return tag
			}
			if strings.Index(itemName, tag+"-o-") == 0 {
				return tag
			}
		}
	}
	return ""
}

// tag는 여러개일 수 있으므로 tag에 해당 이름이 있는지 찾기
func existsNameInTags(name string, tags []string) bool {
	//cblogger.Debug("existsNameInTags : ", name, tags)
	for _, tag := range tags {
		if strings.EqualFold(tag, name) {
			return true
		}
	}
	return false
}

// 가져온 firewallList를 securityGroup으로 묶음.
// tag가 있으면 동일한 tag끼리, tag가 없으면 item.Name
func extractFirewallList(firewallList compute.FirewallList, reqTag string) []compute.FirewallList {

	// 가져온 list에서 tag별 group으로 묶기.
	securityGroupNameMap := make(map[string]string)

	if reqTag != "" {
		securityGroupNameMap[reqTag] = reqTag
	} else {
		for _, item := range firewallList.Items {
			// tag가 있으면 tag로, tag가 없으면 이름으로 : 하지만... tag만 있으면 되나? '-basic', '-i-', '-o-'
			// tag, tag + surpix 가 맞으면 group으로 아니면 이름으로
			itemName := item.Name

			sourceTag := getTagFromTags(itemName, item.SourceTags)
			if sourceTag != "" {
				securityGroupNameMap[sourceTag] = sourceTag
				continue
			}

			targetTag := getTagFromTags(itemName, item.TargetTags)
			if targetTag != "" {
				securityGroupNameMap[targetTag] = targetTag
				continue
			}

			// 못찾았으면 item.Name 사용
			securityGroupNameMap[itemName] = itemName
		}
	}

	cblogger.Debug("********* ", securityGroupNameMap)
	var returnFirewallList []compute.FirewallList
	for _, sgKey := range securityGroupNameMap {
		var returnFirewall compute.FirewallList
		var returnFirewallItemList []*compute.Firewall
		//cblogger.Debug("returnFirewallItemList before length  : ", len(returnFirewallItemList))
		for _, item := range firewallList.Items {
			//cblogger.Debug("get security list result : ", sgKey, item)
			if existsNameInTags(sgKey, item.SourceTags) {
				//cblogger.Debug("SourceTags : ", sgKey, item.SourceTags)
				returnFirewallItemList = append(returnFirewallItemList, item)
				continue
			}
			if existsNameInTags(sgKey, item.TargetTags) {
				//cblogger.Debug("TargetTags : ", sgKey, item.TargetTags)
				returnFirewallItemList = append(returnFirewallItemList, item)
				continue
			}

			if strings.EqualFold(sgKey, item.Name) {
				//cblogger.Debug("Name : ", sgKey, item.Name)
				returnFirewallItemList = append(returnFirewallItemList, item)
			}
			//firewallItemList = append(firewallItemList, item)
		}
		//cblogger.Debug("returnFirewallItemList length  : ", len(returnFirewallItemList))
		//cblogger.Debug("returnFirewallItemList  : ", returnFirewallItemList)
		returnFirewall.Items = returnFirewallItemList
		returnFirewallList = append(returnFirewallList, returnFirewall)
	}

	return returnFirewallList
}

// firewallList 를 securityInfo 로 변경(변경만)
// securityGroupTag 가 있으면 해당이름 사용, 없으며 item이름 사용
func convertFromFirewallToSecurityInfo(firewallList compute.FirewallList) (irs.SecurityInfo, error) {
	var securityInfo irs.SecurityInfo
	var securityRules []irs.SecurityRuleInfo

	// length check
	firewallItems := firewallList.Items
	if len(firewallItems) == 0 {
		return irs.SecurityInfo{}, errors.New("The SecurityRules has no items")
	}
	for _, item := range firewallList.Items {
		securityGroupName := item.Name
		hasSecurityGroupNameFound := false

		sourceTag := getTagFromTags(item.Name, item.SourceTags)
		if sourceTag != "" {
			securityGroupName = sourceTag
			hasSecurityGroupNameFound = true
		}

		if !hasSecurityGroupNameFound {
			targetTag := getTagFromTags(item.Name, item.TargetTags)
			if targetTag != "" {
				securityGroupName = targetTag
				hasSecurityGroupNameFound = true
			}
		}
		cblogger.Debug("get security list result : ", item)

		//Allowed []*FirewallAllowed `json:"allowed,omitempty"`
		//CreationTimestamp string `json:"creationTimestamp,omitempty"`
		//Denied []*FirewallDenied `json:"denied,omitempty"`
		//Description string `json:"description,omitempty"`
		//DestinationRanges []string `json:"destinationRanges,omitempty"`
		//Direction string `json:"direction,omitempty"`
		//Disabled bool `json:"disabled,omitempty"`
		//Id uint64 `json:"id,omitempty,string"`
		//Kind string `json:"kind,omitempty"`
		//LogConfig *FirewallLogConfig `json:"logConfig,omitempty"`
		//Name string `json:"name,omitempty"`
		//Network string `json:"network,omitempty"`
		//Priority int64 `json:"priority,omitempty"`
		//SelfLink string `json:"selfLink,omitempty"`
		//SourceRanges []string `json:"sourceRanges,omitempty"`
		//SourceServiceAccounts []string `json:"sourceServiceAccounts,omitempty"`
		//SourceTags []string `json:"sourceTags,omitempty"`
		cblogger.Debug("SourceTags : ", item.SourceTags)
		//TargetServiceAccounts []string `json:"targetServiceAccounts,omitempty"`
		//TargetTags []string `json:"targetTags,omitempty"`
		cblogger.Debug("TargetTags : ", item.TargetTags)
		//googleapi.ServerResponse `json:"-"`
		//ForceSendFields []string `json:"-"`
		//
		//NullFields []string `json:"-"`

		spiderDirection := switchDirectionSpiderAndGCP(item.Direction, "SPIDER") // GCP로 날 릴 때에는 "GCP", SPIDER에서 사용할 때에는 "SPIDER"
		cidr := ""

		if strings.EqualFold(spiderDirection, Const_Spider_Direction_INBOUND) {
			cidr = strings.Join(item.SourceRanges, ", ")
		} else {
			cidr = strings.Join(item.DestinationRanges, ", ")
		}
		cblogger.Debug("cidr : ", cidr)

		var portArr []string
		var fromPort string
		var toPort string
		var ipProtocol string

		for _, firewallRule := range item.Allowed {
			ipProtocol = firewallRule.IPProtocol
			cblogger.Debug("ipProtocol : ", ipProtocol)
			if strings.EqualFold(ipProtocol, "all") || strings.EqualFold(ipProtocol, "icmp") {
				fromPort = "-1"
				toPort = "-1"
			} else if strings.EqualFold(ipProtocol, "tcp") || strings.EqualFold(ipProtocol, "udp") {
				if ports := firewallRule.Ports; ports != nil {
					portArr = strings.Split(firewallRule.Ports[0], "-")
					// fromPort, toPort 는 1 ~ 65535
					fromPort = portArr[0]
					if len(portArr) > 1 {
						toPort = portArr[1]
					} else {
						toPort = fromPort
					}
					//} else {
					//	fromPort = "-1"
					//	toPort = "-1"
				}
			} else {
				fromPort = "-1"
				toPort = "-1"
			}
			//if ports := firewallRule.Ports; ports != nil {
			//	portArr = strings.Split(firewallRule.Ports[0], "-")
			//	fromPort = portArr[0]
			//	if len(portArr) > 1 {
			//		toPort = portArr[len(portArr)-1]
			//	} else {
			//		toPort = ""
			//	}
			//
			//} else {
			//	fromPort = ""
			//	toPort = ""
			//}
			//

			ruleInfo := irs.SecurityRuleInfo{
				FromPort:   fromPort,
				ToPort:     toPort,
				IPProtocol: ipProtocol,
				Direction:  spiderDirection,
				CIDR:       cidr,
			}
			securityRules = append(securityRules, ruleInfo)
		} // end of firewall rule

		vpcArr := strings.Split(item.Network, "/")
		vpcName := vpcArr[len(vpcArr)-1]
		securityInfo = irs.SecurityInfo{
			IId: irs.IID{
				//NameId:   item.Name,
				//SystemId: item.Name,
				NameId:   securityGroupName,
				SystemId: securityGroupName, // Tag를 찾으면 tag로 못찾으면 item 이름으로
			},
			VpcIID: irs.IID{
				NameId:   vpcName,
				SystemId: vpcName,
			},

			// Direction: security.Direction,
			KeyValueList: []irs.KeyValue{
				{Key: "Priority", Value: strconv.FormatInt(item.Priority, 10)},
				// {"SourceRanges", security.SourceRanges[0]},
				{Key: "Allowed", Value: ipProtocol},
				{Key: "Vpc", Value: vpcName},
			},
			SecurityRules: &securityRules,
		}
		cblogger.Debug("securityRules : ", securityRules)
		cblogger.Debug("securityRules length: ", len(securityRules))
	} // end of result.items
	cblogger.Debug("securityInfo : ", securityInfo)
	return securityInfo, nil
}

// Spider에서 온 값은 GCP로 변경 ( "INGRESS", GCP ) => inbound 로 return
// GCP에서 온 값은 Spider로 변경 ( "inbound", SPIDER) => INGRESS 로 return
func switchDirectionSpiderAndGCP(direction string, targetType string) string {
	returnDirection := direction
	// gcp로 변경을 하는 경우 return = INGRESS, EGESS
	if strings.EqualFold(targetType, "GCP") {
		if strings.EqualFold(direction, Const_Spider_Direction_INBOUND) { //"inbound"
			returnDirection = Const_GCP_Direction_INGRESS // INGRESS
		} else {
			returnDirection = Const_GCP_Direction_EGRESS
		}
	} else if strings.EqualFold(targetType, "SPIDER") {
		if strings.EqualFold(direction, Const_GCP_Direction_INGRESS) {
			returnDirection = Const_Spider_Direction_INBOUND
		} else {
			returnDirection = Const_Spider_Direction_OUTBOUND
		}
	}
	return returnDirection
}

// 동일한 rule이 있는지 check
// action = add 면 존재하는 rule 목록 반환 : 이미있는 rule은 추가하지 않음
// action = remove 면  존재하지 않는 rule 목록 반환 : 없는 rule은 삭제하지 않음
func sameRuleCheck(searchedSecurityRules *[]irs.SecurityRuleInfo, requestedSecurityRules *[]irs.SecurityRuleInfo, action string) *[]irs.SecurityRuleInfo {

	var checkResult []irs.SecurityRuleInfo
	for _, reqRule := range *requestedSecurityRules {
		hasFound := false
		reqRulePort := ""

		//////// 작업 할 것
		fromPort := reqRule.FromPort
		toPort := reqRule.ToPort

		// 둘 다 없으면 -1
		// 둘중의 하나만 있으면 똑같이
		// 둘다 있으면
		//		작은지 체크, 큰지 체크

		if strings.EqualFold(fromPort, "") && strings.EqualFold(toPort, "") {
			// reqRulePort 값이 없으면 전체
			fromPort = "-1"
			toPort = "-1"
			reqRulePort = fromPort
		} else if strings.EqualFold(fromPort, "") || strings.EqualFold(toPort, "") {
			if fromPort == "" {
				reqRulePort = toPort
			} else if toPort == "" {
				reqRulePort = fromPort
			} else if fromPort == toPort {
				reqRulePort = fromPort
			} else {
				reqRulePort = fromPort + "-" + toPort
			}
		} else if strings.EqualFold(fromPort, "-1") || strings.EqualFold(toPort, "-1") {
			reqRulePort = fromPort
		} else if fromPort == toPort {
			reqRulePort = fromPort
		} else {
			reqRulePort = fromPort + "-" + toPort
		}

		//
		//if reqRule.FromPort == "" {
		//	reqRulePort = reqRule.ToPort
		//} else if reqRule.ToPort == "" {
		//	reqRulePort = reqRule.FromPort
		//} else if reqRule.FromPort == reqRule.ToPort {
		//	reqRulePort = reqRule.FromPort
		//} else {
		//	reqRulePort = reqRule.FromPort + "-" + reqRule.ToPort
		//}

		for _, searchedRule := range *searchedSecurityRules {
			searchedRulePort := ""
			if searchedRule.FromPort == "" {
				searchedRulePort = searchedRule.ToPort
			} else if searchedRule.ToPort == "" {
				searchedRulePort = searchedRule.FromPort
			} else if searchedRule.FromPort == searchedRule.ToPort {
				searchedRulePort = searchedRule.FromPort
			} else {
				searchedRulePort = searchedRule.FromPort + "-" + searchedRule.ToPort
			}

			if !strings.EqualFold(reqRule.Direction, searchedRule.Direction) {
				continue
			}
			if !strings.EqualFold(reqRule.IPProtocol, searchedRule.IPProtocol) {
				continue
			}
			if !strings.EqualFold(reqRulePort, searchedRulePort) {
				continue
			}
			if !strings.EqualFold(reqRule.CIDR, searchedRule.CIDR) {
				continue
			}
			cblogger.Debug("aaa : ", reqRulePort, ":"+fromPort+" : "+toPort)
			cblogger.Debug("bbb : ", searchedRulePort, ":"+searchedRule.FromPort+" : "+searchedRule.ToPort)
			cblogger.Debug("Direction : ", reqRule.Direction, ":"+searchedRule.Direction)
			cblogger.Debug("IPProtocol : ", reqRule.IPProtocol, ":"+searchedRule.IPProtocol)
			cblogger.Debug("CIDR : ", reqRule.CIDR, ":"+searchedRule.CIDR)

			// add일 때는 존재하는게 있으면 안됨.
			if action == Const_SecurityRule_Add {
				cblogger.Info("add")
				checkResult = append(checkResult, reqRule)
			}
			hasFound = true
			break
		}
		cblogger.Info(action, hasFound)
		// remove일 때는 없으면 안됨(존재해야 함)
		if !hasFound && action == Const_SecurityRule_Remove {
			cblogger.Info("remove")
			checkResult = append(checkResult, reqRule)
		}
	}
	return &checkResult
}

// Tag로 묶인 firewall의 max sequence 추출
func maxFirewallSequence(firewallList []compute.FirewallList, gcpDirection string) int {
	maxSequence := 0

	namingRule := ""
	for _, firewallInfo := range firewallList {
		for _, item := range firewallInfo.Items {
			// naming rule

			if strings.EqualFold(gcpDirection, Const_GCP_Direction_INGRESS) {
				namingRule = "-i-"
			} else if strings.EqualFold(gcpDirection, Const_GCP_Direction_EGRESS) {
				namingRule = "-o-"
			} else {
				continue
			}
			str := item.Name[len(item.Name)-6:]
			if strings.Index(str, namingRule) == 0 {
				curSequence, _ := strconv.Atoi(str[3:]) // 끝 세자리
				if curSequence > maxSequence {
					maxSequence = curSequence
				}
				cblogger.Debug("str : ", str)
				cblogger.Debug("curSequence : ", curSequence)
			}
		}
	}

	return maxSequence
}

// firewall insert를 create, add 등에서 여러번 사용하므로 공통으로 처리
// securityGroup = spider 명칭, firewall  = GCP 명칭
func (securityHandler *GCPSecurityHandler) firewallInsert(firewallInfo compute.Firewall) (compute.Firewall, error) {
	projectID := securityHandler.Credential.ProjectID

	//// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: firewallInfo.Name,
		CloudOSAPI:   "Firewalls.Insert()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	res, err := securityHandler.Client.Firewalls.Insert(projectID, &firewallInfo).Do()

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		cblogger.Error(err)
		return compute.Firewall{}, err
	}
	callogger.Info(call.String(callLogInfo))
	cblogger.Debug("create default firewall rule result : ", res)

	errWait := securityHandler.WaitUntilComplete(res.Name)
	if errWait != nil {
		cblogger.Errorf("SecurityGroup create 완료 대기 실패")
		cblogger.Error(errWait)
		return compute.Firewall{}, errWait
	}

	return firewallInfo, nil
}

// firewall 삭제
// param에 resourceID가 있으면 해당 resourceID만 제거
func (securityHandler *GCPSecurityHandler) firewallDelete(securityGroupTag string, firewallName string, firewallInfo compute.FirewallList) (bool, error) {
	projectID := securityHandler.Credential.ProjectID
	//
	resourceID := ""
	for _, item := range firewallInfo.Items {
		if !strings.EqualFold(firewallName, "") {
			if !strings.EqualFold(firewallName, item.Name) {
				continue
			}
			resourceID = firewallName
		}
		resourceID = item.Name
		cblogger.Debug("firewallDelete ", securityGroupTag, " : ", resourceID)
		callogger := call.GetLogger("HISCALL")
		callLogInfo := call.CLOUDLOGSCHEMA{
			CloudOS:      call.GCP,
			RegionZone:   securityHandler.Region.Zone,
			ResourceType: call.SECURITYGROUP,
			ResourceName: securityGroupTag,
			CloudOSAPI:   "Firewalls.Delete()",
			ElapsedTime:  "",
			ErrorMSG:     "",
		}
		callLogStart := call.Start()

		res, err := securityHandler.Client.Firewalls.Delete(projectID, resourceID).Do()
		if err != nil {
			return false, err
		}
		callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
		if err != nil {
			callLogInfo.ErrorMSG = err.Error()
			callogger.Error(call.String(callLogInfo))
			cblogger.Error(err)
			return false, err
		}
		callogger.Info(call.String(callLogInfo))
		cblogger.Debug("remove result : ", resourceID, res)

		errWait := securityHandler.WaitUntilComplete(res.Name)
		if errWait != nil {
			cblogger.Errorf("SecurityGroup Delete 완료 대기 실패")
			cblogger.Error(errWait)
			return false, errWait
		}
	}
	return true, nil
}

// 현재 프로젝트의 firewall 목록 조회
// GCP는 프로젝트 아래에 모든 firewall이 있음
// tag단위로 묶음.(tag가 있으면 tag로, 없으면 item.Name을 그대로 사용)
// tag 값이 없으면 전체 목록, 있으면 tags에서 해당 tag만 추출
func (securityHandler *GCPSecurityHandler) firewallList(tag string) ([]compute.FirewallList, error) {
	projectID := securityHandler.Credential.ProjectID

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: "",
		CloudOSAPI:   "Firewalls.List()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := securityHandler.Client.Firewalls.List(projectID).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	firewallList := extractFirewallList(*result, tag) // 그룹으로 묶기
	return firewallList, nil
}

// securityGroup(firewall) 은 global
func (securityHandler *GCPSecurityHandler) WaitUntilComplete(resourceId string) error {

	//project string, operation string
	project := securityHandler.Credential.ProjectID

	before_time := time.Now()
	max_time := 300 //최대 300초간 체크

	var opSatus *compute.Operation
	var err error

	for {
		opSatus, err = securityHandler.Client.GlobalOperations.Get(project, resourceId).Do()
		if err != nil {
			return err
		}
		cblogger.Infof("==> 상태 : 진행율 : [%d] / [%s]", opSatus.Progress, opSatus.Status)

		//PENDING, RUNNING, or DONE.
		//if (opSatus.Status == "RUNNING" || opSatus.Status == "DONE") && opSatus.Progress >= 100 {
		if opSatus.Status == "DONE" {
			cblogger.Info("요청 작업이 정상적으로 처리되어서 Wait을 종료합니다.")
			return nil
		}

		time.Sleep(time.Second * 1)
		after_time := time.Now()
		diff := after_time.Sub(before_time)
		if int(diff.Seconds()) > max_time {
			cblogger.Errorf("[%d]초 동안 리소스[%s]의 상태가 완료되지 않아서 Wait을 강제로 종료함.", max_time, resourceId)
			return errors.New("장시간 요청 작업이 완료되지 않아서 Wait을 강제로 종료함.)")
		}
	}

	return nil
}
