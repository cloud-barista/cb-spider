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
	"fmt"
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
//Ex) sg name = sg-test 일 때 firewallname은 조합하여 만들고, tag로 sg를 묶는다                    inbound TCP/22/22/0.0.0.0/0 이면  firewall name=sg-test-basic, tag=sg-test   : 0번 포트는 무조건 추가
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
//	fmt.Println("create result : ", res)
//	time.Sleep(time.Second * 3)
//	//secInfo, _ := securityHandler.GetSecurity(securityReqInfo.IId)
//	secInfo, _ := securityHandler.GetSecurity(irs.IID{SystemId: securityReqInfo.IId.NameId})
//	return secInfo, nil
//}
// securityGroup = GCP 의 Tag
func (securityHandler *GCPSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Info(securityReqInfo)

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
	basicSecurityRuleInfo := irs.SecurityRuleInfo{
		FromPort:   "",
		IPProtocol: "TCP",
		Direction:  "INGRESS",
		CIDR:       "0.0.0.0/0",
	}
	basicFireWall := setNewFirewall(basicSecurityRuleInfo, projectID, securityReqInfo.VpcIID.SystemId, securityReqInfo.IId.NameId, "basic", 0)
	////	fireWall := &compute.Firewall{
	////		Allowed:   firewallAllowed,
	////		Direction: commonDirection, //INGRESS(inbound), EGRESS(outbound)
	////		// SourceRanges: []string{
	////		// 	// "0.0.0.0/0",
	////		// 	commonCidr,
	////		// },
	////		Name: securityReqInfo.IId.NameId,
	////		TargetTags: []string{
	////			securityReqInfo.IId.NameId,
	////		},
	////		Network: networkURL,
	////	}
	//cblogger.Info("생성할 기본 방화벽 정책")
	//cblogger.Debug(defaultFireWall)
	//spew.Dump(fireWall)
	//

	// default firewallInsert
	_, err := securityHandler.firewallInsert(basicFireWall)
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	defaultOutboundSecurityRuleInfo := irs.SecurityRuleInfo{
		FromPort:   "",
		IPProtocol: "TCP",
		Direction:  "EGRESS",
		CIDR:       "0.0.0.0/0",
	}
	defaultOutboundFireWall := setNewFirewall(defaultOutboundSecurityRuleInfo, projectID, securityReqInfo.VpcIID.SystemId, securityReqInfo.IId.NameId, "-o-", 1)
	_, err = securityHandler.firewallInsert(defaultOutboundFireWall)
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	////// logger for HisCall
	//callogger := call.GetLogger("HISCALL")
	//callLogInfo := call.CLOUDLOGSCHEMA{
	//	CloudOS:      call.GCP,
	//	RegionZone:   securityHandler.Region.Zone,
	//	ResourceType: call.SECURITYGROUP,
	//	ResourceName: securityReqInfo.IId.NameId,
	//	CloudOSAPI:   "Firewalls.Insert()",
	//	ElapsedTime:  "",
	//	ErrorMSG:     "",
	//}
	//callLogStart := call.Start()
	//
	//defaultRes, defaultErr := securityHandler.Client.Firewalls.Insert(projectID, &defaultFireWall).Do()
	//
	//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//if defaultErr != nil {
	//	callLogInfo.ErrorMSG = defaultErr.Error()
	//	callogger.Error(call.String(callLogInfo))
	//	cblogger.Error(defaultErr)
	//	return irs.SecurityInfo{}, defaultErr
	//}
	//callogger.Info(call.String(callLogInfo))
	//fmt.Println("create default firewall rule result : ", defaultRes)
	//time.Sleep(time.Second * 3) // TODO : 매 호출마다 3초씩 기다리지 말고 CH를 이용하여 req가 모두 완료될 때까지 기다린 후 처리.

	reqSecurityRules := *securityReqInfo.SecurityRules
	reqEgressCount := 1
	reqIngressCount := 1
	//var waitGroup sync.WaitGroup
	//waitGroup.Add(len(reqSecurityRules))
	//ch := make(chan string)
	for itemIndex, item := range reqSecurityRules {
		firewallDirection := item.Direction
		firewallType := ""
		var fireWall compute.Firewall
		if strings.EqualFold(firewallDirection, "INGRESS") || strings.EqualFold(firewallDirection, "inbound") {
			firewallType = "-i-"
			fireWall = setNewFirewall(item, projectID, securityReqInfo.VpcIID.SystemId, securityReqInfo.IId.NameId, firewallType, reqIngressCount)
			reqIngressCount++
		} else if strings.EqualFold(firewallDirection, "EGRESS") || strings.EqualFold(firewallDirection, "outbound") {
			firewallType = "-o-"
			fireWall = setNewFirewall(item, projectID, securityReqInfo.VpcIID.SystemId, securityReqInfo.IId.NameId, firewallType, reqEgressCount)
			reqEgressCount++
		} else {
			// direction 이 없는데.... continue
			continue
		}

		cblogger.Info("생성할 방화벽 정책 ", itemIndex, firewallDirection, reqEgressCount, reqIngressCount)
		cblogger.Debug(fireWall)
		//spew.Dump(fireWall)

		_, err := securityHandler.firewallInsert(fireWall)
		if err != nil {
			return irs.SecurityInfo{}, err
		}

		//// logger for HisCall
		//callogger := call.GetLogger("HISCALL")
		//callLogInfo := call.CLOUDLOGSCHEMA{
		//	CloudOS:      call.GCP,
		//	RegionZone:   securityHandler.Region.Zone,
		//	ResourceType: call.SECURITYGROUP,
		//	ResourceName: securityReqInfo.IId.NameId,
		//	CloudOSAPI:   "Firewalls.Insert()",
		//	ElapsedTime:  "",
		//	ErrorMSG:     "",
		//}
		//callLogStart := call.Start()
		//
		//res, err := securityHandler.Client.Firewalls.Insert(projectID, &fireWall).Do()
		//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
		//if err != nil {
		//	callLogInfo.ErrorMSG = err.Error()
		//	callogger.Error(call.String(callLogInfo))
		//	cblogger.Error(err)
		//	return irs.SecurityInfo{}, err
		//}
		//callogger.Info(call.String(callLogInfo))
		////fmt.Println("create result : ", res)
		////fmt.Println("res.Id : ", res.Id, res.Status)
		////fmt.Println("Name : ", res.Name, res.Region)
		////fmt.Println("ClientOperationId : ", res.ClientOperationId, ": ", res.TargetId)
		////operation, err := securityHandler.Client.GlobalOperations.Get(projectID, res.Name).Do() // vpc는 Region, firewall은 Global
		////fmt.Println("operation", operation)
		////time.Sleep(time.Second * 3) // TODO : 매 호출마다 3초씩 기다리지 말고 CH를 이용하여 req가 모두 완료될 때까지 기다린 후 처리.
		////operation2, err := securityHandler.Client.GlobalOperations.Get(projectID, res.Name).Do()
		////fmt.Println("operation2", operation2)
		////time.Sleep(time.Second * 3) // TODO : 매 호출마다 3초씩 기다리지 말고 CH를 이용하여 req가 모두 완료될 때까지 기다린 후 처리.
		////
		//
		//// 우선 WaitUntilComplete을 사용하고 추후 waitgroup, channel 등 사용으로 변경할 것
		//errWait := vNetworkHandler.WaitUntilComplete(res.Name, true)
		//if errWait != nil {
		//	cblogger.Errorf("SecurityGroup create 완료 대기 실패")
		//	cblogger.Error(errWait)
		//	return irs.SecurityInfo{}, errWait
		//}

		//getOperationsStatus(*securityHandler, projectID, res.Name, "global")
		//waitGroup.Add(1)
		//go func() {
		//	//defer waitGroup.Done()
		//	fmt.Println("getOperationsStatus 호출 전 ", itemIndex)
		//	getOperationsStatus(ch, *securityHandler, projectID, res.Name, "global")
		//	fmt.Println("getOperationsStatus 호출 후  ", itemIndex)
		//	//waitGroup.Done()
		//}()
		//close(ch)
	}
	//waitGroup.Wait()
	//chMessage := <-ch
	//fmt.Println(":: ", chMessage, " :: ")

	// TODO : TAG를 이용해서 해당 security를 모두 가져오도록 보완 필요.
	securityInfo, _ := securityHandler.GetSecurity(irs.IID{SystemId: securityReqInfo.IId.NameId})
	//securityInfo, _ := securityHandler.GetSecurityByTag(irs.IID{SystemId: securityReqInfo.IId.NameId})

	// tag기반으로 security를 묶어서 가져와야 함.
	return securityInfo, nil
	//return irs.SecurityInfo{}, nil
}

//func getOperationsStatus(securityHandler GCPSecurityHandler, projectID string, operationName string, operationType string) {
func (securityHandler GCPSecurityHandler) getOperationsStatus(ch chan string, projectID string, operationName string, operationType string) {
	// global : firewall
	// region : vpc
	//operation2, err := Client.GlobalOperations.Get(projectID, res.Name).Do()

	errWait := securityHandler.WaitUntilComplete(operationName)
	if errWait != nil {
		cblogger.Errorf("SecurityGroup create 완료 대기 실패")
		cblogger.Error(errWait)
	}
	fmt.Println("getOperationsStatus ", operationName)
	ch <- operationName
	//waitGroup.Done()
}

// firewall rule 설정.
// direction, port 마다 1개의 firewall로.
func setNewFirewall(ruleInfo irs.SecurityRuleInfo, projectID string, vpcSystemId string, securityGroupName string, firewallType string, sequence int) compute.Firewall {

	port := setFromPortToPort(ruleInfo.FromPort, ruleInfo.ToPort)
	var firewallAllowed []*compute.FirewallAllowed
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

	cidr := ruleInfo.CIDR
	firewallDirection := ruleInfo.Direction
	prefix := "https://www.googleapis.com/compute/v1/projects/" + projectID
	networkURL := prefix + "/global/networks/" + vpcSystemId

	// default = -basic, inbound = -i-xxx, outbount = -o-xxx
	firewallName := ""

	if strings.EqualFold(firewallType, "-i-") {
		sequenceStr := lpad(strconv.Itoa(sequence), "0", 3)
		firewallName = securityGroupName + "-i-" + sequenceStr
		//firewallName = securityGroupName + "-" + sequenceStr + "-i"
		firewallDirection = "INGRESS"
	} else if strings.EqualFold(firewallType, "-o-") {
		fmt.Println("create sequence : ", sequence, strconv.Itoa(sequence))
		sequenceStr := lpad(strconv.Itoa(sequence), "0", 3)
		firewallName = securityGroupName + "-o-" + sequenceStr
		firewallDirection = "EGRESS"
	} else {
		firewallName = securityGroupName + "-basic"
		firewallDirection = "INGRESS"
	}
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

	fmt.Println("firewallName : ", firewallName)

	fireWall := compute.Firewall{
		Name:      firewallName,
		Allowed:   firewallAllowed,
		Direction: firewallDirection,
		Network:   networkURL,
		TargetTags: []string{
			securityGroupName,
		},
	}

	//CIDR 처리 : ingress=>sourceRanges, egress=>destination  둘 중 하나만 선택 가능
	if strings.EqualFold(firewallDirection, "INGRESS") || strings.EqualFold(firewallDirection, "inbound") {
		fireWall.SourceRanges = []string{cidr}
	} else {
		fireWall.DestinationRanges = []string{cidr}
	}

	fmt.Println("firewallset : ", fireWall)
	return fireWall
}

func setFromPortToPort(fp string, tp string) string {
	var port string
	if fp == "-1" || tp == "-1" {
		if (fp == "-1" && tp == "-1") || (fp == "-1" && tp == "") || (fp == "" && tp == "-1") {
			port = ""
		} else if fp == "-1" {
			port = tp
		} else {
			port = fp
		}
	} else {
		//둘 다 있는 경우
		if tp != "" && fp != "" {
			port = fp + "-" + tp
			//From Port가 없는 경우
		} else if tp != "" && fp == "" {
			port = tp
			//To Port가 없는 경우
		} else if tp == "" && fp != "" {
			port = fp
		} else {
			port = ""
		}
	}
	return port
}

// string 원본, 앞에 붙일 값, 전체 길이
func lpad(sequence string, pad string, plength int) string {
	for i := len(sequence); i < plength; i++ {
		sequence = pad + sequence
	}
	return sequence
}

func (securityHandler *GCPSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	////result, err := securityHandler.Client.ListAll(securityHandler.Ctx)
	//projectID := securityHandler.Credential.ProjectID
	//// logger for HisCall
	//callogger := call.GetLogger("HISCALL")
	//callLogInfo := call.CLOUDLOGSCHEMA{
	//	CloudOS:      call.GCP,
	//	RegionZone:   securityHandler.Region.Zone,
	//	ResourceType: call.SECURITYGROUP,
	//	ResourceName: "",
	//	CloudOSAPI:   "Firewalls.List()",
	//	ElapsedTime:  "",
	//	ErrorMSG:     "",
	//}
	//callLogStart := call.Start()
	//
	//result, err := securityHandler.Client.Firewalls.List(projectID).Do()
	//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//if err != nil {
	//	callLogInfo.ErrorMSG = err.Error()
	//	callogger.Info(call.String(callLogInfo))
	//	cblogger.Error(err)
	//	return nil, err
	//}
	//callogger.Info(call.String(callLogInfo))
	//
	//firewallList := extractFirewallList(*result) // 그룹으로 묶기
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
	fmt.Println("securityInfoList = ", securityInfoList)

	//var securityInfoList []*irs.SecurityInfo
	//for key, val := range securityGroupNameMap {
	//	fmt.Println(key, val)
	//	var elementType = "tag"
	//	extractResult := extractFirewallList(*result, key, elementType) // tag로 찾기
	//
	//	firewallItems := result.Items
	//	fmt.Println("firewallItems length = ", len(firewallItems))
	//	if len(firewallItems) > 0 {
	//	} else { // tag에서 못찾았으면 이름에서 찾기
	//		elementType = "name"
	//		extractResult = extractFirewallList(*result, key, elementType) // 아름으로 찾기
	//	}
	//	securityInfo, err := convertFromFirewallToSecurityInfo(extractResult, key, elementType)
	//	if err != nil {
	//		//
	//	}
	//	securityInfoList = append(securityInfoList, &securityInfo)
	//}

	//for _, item := range result.Items {
	//	name := item.Name
	//	//systemId := strconv.FormatUint(item.Id, 10)
	//	//secInfo, _ := securityHandler.GetSecurity(irs.IID{NameId: name, SystemId: systemId})
	//	secInfo, _ := securityHandler.GetSecurity(irs.IID{SystemId: name}) // => Extract 로 바꿔야
	//
	//	securityInfo = append(securityInfo, &secInfo)
	//}

	return securityInfoList, nil
}

// TAG를 이용해서 해당 security(firewall)를 모두 가와야 하기 때문에
// 해당 project의 모든 list에서 해당 하는 TAG를 추출
func (securityHandler *GCPSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	// list sequrity로 목록을 가져온 후 Tag로 다시 추출
	//projectID := securityHandler.Credential.ProjectID
	//securityGroupTag := securityIID.SystemId
	//
	//callogger := call.GetLogger("HISCALL")
	//callLogInfo := call.CLOUDLOGSCHEMA{
	//	CloudOS:      call.GCP,
	//	RegionZone:   securityHandler.Region.Zone,
	//	ResourceType: call.SECURITYGROUP,
	//	ResourceName: "",
	//	CloudOSAPI:   "Firewalls.List()",
	//	ElapsedTime:  "",
	//	ErrorMSG:     "",
	//}
	//callLogStart := call.Start()
	//
	//// TODO : filter기능으로 해당 Tag만 가져올 수 있도록 보완 필요
	////req1 := securityHandler.Client.Firewalls.List(projectID)
	////req1.Filter("TargetTags:" + securityGroupTag)//Invalid value for field 'filter': 'TargetTags:sgtestaa'. Invalid list filter expression., invalid)
	////fmt.Println("firewall TargetTags : ", securityGroupTag)
	////result, err := req1.Do()
	//result, err := securityHandler.Client.Firewalls.List(projectID).Do()
	//
	//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//if err != nil {
	//	callLogInfo.ErrorMSG = err.Error()
	//	callogger.Info(call.String(callLogInfo))
	//	cblogger.Error(err)
	//	return irs.SecurityInfo{}, err
	//}
	//

	securityGroupTag := securityIID.SystemId

	//// Inbound는 sourceTag에서 outbound는 targetTag에서 추출.
	//firewallList := extractFirewallList(*result) // tag group으로 묶은 배열 추출
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

	fmt.Println("securityInfo : ", securityInfo)
	return securityInfo, nil
}

//func (securityHandler *GCPSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
//	securityInfo, _ := GetSecurityByTag(securityIID)
//
//	// tag기반으로 security를 묶어서 가져와야 함.
//	return securityInfo, nil
//}

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

// 해당 Tag를 가진 firewall 삭제
func (securityHandler *GCPSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	//projectID := securityHandler.Credential.ProjectID
	securityGroupTag := securityIID.SystemId
	fmt.Println("Delete Security ", securityGroupTag)
	// 해당 Tag를 가진 목록 조회
	firewallList, err := securityHandler.firewallList(securityGroupTag)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}
	fmt.Println("Delete Security 삭제 대상 ", len(firewallList))
	for _, firewallInfo := range firewallList {
		fmt.Println("Delete Security 삭제 대상item ", len(firewallInfo.Items))
		securityHandler.firewallDelete(securityGroupTag, "", firewallInfo)
		if err != nil {
			//500  convert Error
			return false, err
		}
	}
	return true, nil

	//callogger := call.GetLogger("HISCALL")
	//callLogInfo := call.CLOUDLOGSCHEMA{
	//	CloudOS:      call.GCP,
	//	RegionZone:   securityHandler.Region.Zone,
	//	ResourceType: call.SECURITYGROUP,
	//	ResourceName: securityIID.SystemId,
	//	CloudOSAPI:   "DeleteSecurity()",
	//	ElapsedTime:  "",
	//	ErrorMSG:     "",
	//}
	//callLogStart := call.Start()
	//res, err := securityHandler.Client.Firewalls.Delete(projectID, securityIID.SystemId).Do()
	//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//if err != nil {
	//	callLogInfo.ErrorMSG = err.Error()
	//	callogger.Info(call.String(callLogInfo))
	//	cblogger.Error(err)
	//	return false, err
	//}
	//callogger.Info(call.String(callLogInfo))
	//fmt.Println(res)
	//return true, nil
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
//	fmt.Println(res)
//	return true, nil
//}

func (securityHandler *GCPSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	cblogger.Info(*securityRules)

	projectID := securityHandler.Credential.ProjectID
	securityGroupTag := sgIID.SystemId

	// 기존에 존재하는지
	//searchSecurityInfo, err := securityHandler.GetSecurity(sgIID)
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
		searchSecurityInfo.VpcIID = tempSecurityInfo.VpcIID
	}
	searchSecurityInfo.SecurityRules = &tempSecurityRules

	// 동일한 rule이 존재하면 존재하는 목록 return
	sameRuleList := sameRuleCheck(searchSecurityInfo.SecurityRules, securityRules, Const_Add)
	if len(*sameRuleList) > 0 {
		return irs.SecurityInfo{}, errors.New("Same SecurityRule is exists")
	}

	// TODO : 존재하는 item의 max Sequence 찾아와야 함
	//reqEgressCount := 1
	//reqIngressCount := 1
	reqEgressCount := maxFirewallSequence(firewallList, "outbound")
	reqIngressCount := maxFirewallSequence(firewallList, "outbound")
	reqEgressCount++
	reqIngressCount++
	for _, item := range *securityRules {
		firewallDirection := item.Direction
		firewallType := ""
		var fireWall compute.Firewall
		if strings.EqualFold(firewallDirection, "INGRESS") || strings.EqualFold(firewallDirection, "inbound") {
			firewallType = "-i-"
			fireWall = setNewFirewall(item, projectID, searchSecurityInfo.VpcIID.SystemId, securityGroupTag, firewallType, reqIngressCount)
			reqIngressCount++
		} else if strings.EqualFold(firewallDirection, "EGRESS") || strings.EqualFold(firewallDirection, "outbound") {
			firewallType = "-o-"
			fireWall = setNewFirewall(item, projectID, searchSecurityInfo.VpcIID.SystemId, securityGroupTag, firewallType, reqEgressCount)
			reqEgressCount++
		} else {
			// direction 이 없는데.... continue
			fmt.Println("no direction : ", firewallDirection)
			continue
		}

		_, err := securityHandler.firewallInsert(fireWall)
		if err != nil {
			return irs.SecurityInfo{}, err
		}

		//cblogger.Info("생성할 방화벽 정책 ", firewallDirection, reqEgressCount, reqIngressCount)
		//cblogger.Debug(fireWall)
		////spew.Dump(fireWall)
		//
		//// logger for HisCall
		//callogger := call.GetLogger("HISCALL")
		//callLogInfo := call.CLOUDLOGSCHEMA{
		//	CloudOS:      call.GCP,
		//	RegionZone:   securityHandler.Region.Zone,
		//	ResourceType: call.SECURITYGROUP,
		//	ResourceName: securityGroupTag,
		//	CloudOSAPI:   "Firewalls.Insert()",
		//	ElapsedTime:  "",
		//	ErrorMSG:     "",
		//}
		//callLogStart := call.Start()
		//
		//res, err := securityHandler.Client.Firewalls.Insert(projectID, &fireWall).Do()
		//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
		//if err != nil {
		//	callLogInfo.ErrorMSG = err.Error()
		//	callogger.Error(call.String(callLogInfo))
		//	cblogger.Error(err)
		//	return irs.SecurityInfo{}, err
		//}
		//callogger.Info(call.String(callLogInfo))
		//fmt.Println("create result : ", res)
		//operation := securityHandler.Client.GlobalOperations.Get(projectID, res.ClientOperationId)
		//fmt.Println("operation", operation)
		//time.Sleep(time.Second * 3) // TODO : 매 호출마다 3초씩 기다리지 말고 CH를 이용하여 req가 모두 완료될 때까지 기다린 후 처리.
		//operation2 := securityHandler.Client.GlobalOperations.Get(projectID, res.ClientOperationId)
		//fmt.Println("operation", operation2)
	}
	return securityHandler.GetSecurity(sgIID)

	//// @TODO: SecurityGroup 생성 요청 파라미터 정의 필요
	//ports := *securityRules
	//existedRules := security.Allowed
	//var firewallAllowed []*compute.FirewallAllowed
	//
	//for _, rule := range existedRules {
	//	firewallAllowed = append(firewallAllowed, rule)
	//}
	//
	////다른 드라이버와의 통일을 위해 All은 -1로 처리함.
	////GCP는 포트 번호를 적지 않으면 All임.
	////GCP 방화벽 정책
	////https://cloud.google.com/vpc/docs/firewalls?hl=ko&_ga=2.238147008.-1577666838.1589162755#protocols_and_ports
	//for _, item := range ports {
	//	var port string
	//	fp := item.FromPort
	//	tp := item.ToPort
	//
	//	//GCP는 1개의 정책에 1가지 Direction만 지정 가능하기 때문에 Inbound와 Outbound 모두 지정되었을 경우 에러 처리함.
	//	if !strings.EqualFold(item.Direction, commonDirection) {
	//		return irs.SecurityInfo{}, errors.New("invalid value - GCP can only use one Direction for one security policy")
	//	}
	//
	//	// CB Rule에 의해 Port 번호에 -1이 기입된 경우 GCP Rule에 맞게 치환함.
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
	//
	//	if port == "" {
	//		firewallAllowed = append(firewallAllowed, &compute.FirewallAllowed{
	//			IPProtocol: item.IPProtocol,
	//		})
	//	} else {
	//		firewallAllowed = append(firewallAllowed, &compute.FirewallAllowed{
	//			IPProtocol: item.IPProtocol,
	//			Ports: []string{
	//				port,
	//			},
	//		})
	//	}
	//}
	//
	//if strings.EqualFold(commonDirection, "inbound") || strings.EqualFold(commonDirection, "INGRESS") {
	//	commonDirection = "INGRESS"
	//} else if strings.EqualFold(commonDirection, "outbound") || strings.EqualFold(commonDirection, "EGRESS") {
	//	commonDirection = "EGRESS"
	//} else {
	//	return irs.SecurityInfo{}, errors.New("invalid value - The direction[" + commonDirection + "] information is unknown")
	//}
	//
	//if !strings.EqualFold(security.Direction, commonDirection) {
	//	return irs.SecurityInfo{}, errors.New("invalid value - GCP can only use one Direction for one security policy")
	//}

	//prefix := "https://www.googleapis.com/compute/v1/projects/" + projectID
	////networkURL := prefix + "/global/networks/" + securityReqInfo.VpcIID.NameId
	//networkURL := prefix + "/global/networks/" + vpcName
	//
	//fireWall := &compute.Firewall{
	//	Allowed:   firewallAllowed,
	//	Direction: commonDirection, //INGRESS(inbound), EGRESS(outbound)
	//	// SourceRanges: []string{
	//	// 	// "0.0.0.0/0",
	//	// 	commonCidr,
	//	// },
	//	//Name: security.Name,
	//	//TargetTags: []string{
	//	//security.Name,
	//	//},
	//	Network: networkURL,
	//}
	//
	////CIDR 처리
	//if strings.EqualFold(commonDirection, "INGRESS") {
	//	//fireWall.SourceRanges = []string{commonCidr}
	//	fireWall.SourceRanges = commonCidr
	//} else {
	//	//fireWall.DestinationRanges = []string{commonCidr}
	//	fireWall.DestinationRanges = commonCidr
	//}
	//
	//cblogger.Info("생성할 방화벽 정책")
	//cblogger.Debug(fireWall)
	////spew.Dump(fireWall)
	//
	//// logger for HisCall
	//callogger := call.GetLogger("HISCALL")
	//callLogInfo := call.CLOUDLOGSCHEMA{
	//	CloudOS:      call.GCP,
	//	RegionZone:   securityHandler.Region.Zone,
	//	ResourceType: call.SECURITYGROUP,
	//	ResourceName: security.Name,
	//	CloudOSAPI:   "Firewalls.Update()",
	//	ElapsedTime:  "",
	//	ErrorMSG:     "",
	//}
	//callLogStart := call.Start()
	//
	//res, err := securityHandler.Client.Firewalls.Update(projectID, sgIID.SystemId, fireWall).Do()
	//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//if err != nil {
	//	callLogInfo.ErrorMSG = err.Error()
	//	callogger.Error(call.String(callLogInfo))
	//	cblogger.Error(err)
	//	return irs.SecurityInfo{}, err
	//}
	//callogger.Info(call.String(callLogInfo))
	//fmt.Println("create result : ", res)
	//time.Sleep(time.Second * 3)
	////secInfo, _ := securityHandler.GetSecurity(securityReqInfo.IId)
	//secInfo, _ := securityHandler.GetSecurity(irs.IID{SystemId: sgIID.SystemId})
	//return secInfo, nil
	//return irs.SecurityInfo{}, fmt.Errorf("Coming Soon!")
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
//	fmt.Println("create result : ", res)
//	time.Sleep(time.Second * 3)
//	//secInfo, _ := securityHandler.GetSecurity(securityReqInfo.IId)
//	secInfo, _ := securityHandler.GetSecurity(irs.IID{SystemId: sgIID.SystemId})
//	return secInfo, nil
//	//return irs.SecurityInfo{}, fmt.Errorf("Coming Soon!")
//}

// 요청받은 Security 그룹안의 SecurityRule이 동일한 firewall 삭제
func (securityHandler *GCPSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	cblogger.Info(*securityRules)

	//projectID := securityHandler.Credential.ProjectID
	securityGroupTag := sgIID.SystemId

	// 기존에 존재하는지 : deprecated. 단일 securityGroup에서 tag로 바뀌면서 tag조회로 변경
	//searchSecurityInfo, err := securityHandler.GetSecurity(sgIID)
	//if err != nil {
	//	// 해당 securityGroup이 존재하지 않음.
	//	return false, errors.New("SecurityGroup is not exists")
	//}

	// 존재하는 목록에서 resourceId를 추출해야 하는데.... securityInfo에는 resourceID가 없음. 따라서 목록 조회한 후 직접추출
	//callogger := call.GetLogger("HISCALL")
	//callLogInfo := call.CLOUDLOGSCHEMA{
	//	CloudOS:      call.GCP,
	//	RegionZone:   securityHandler.Region.Zone,
	//	ResourceType: call.SECURITYGROUP,
	//	ResourceName: "",
	//	CloudOSAPI:   "Firewalls.List()",
	//	ElapsedTime:  "",
	//	ErrorMSG:     "",
	//}
	//callLogStart := call.Start()
	//
	//result, err := securityHandler.Client.Firewalls.List(projectID).Do()
	//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//if err != nil {
	//	callLogInfo.ErrorMSG = err.Error()
	//	callogger.Info(call.String(callLogInfo))
	//	cblogger.Error(err)
	//	return false, err
	//}
	//callogger.Info(call.String(callLogInfo))
	//
	//var securityInfo irs.SecurityInfo
	//var firewallInfo compute.FirewallList
	//firewallList := extractFirewallList(*result, securityGroupTag) // tag group으로 추출

	//var firewallInfo compute.FirewallList
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
	}
	searchSecurityInfo.SecurityRules = &tempSecurityRules

	// 동일한 rule이 존재하지 않으면 지울 수 없으므로 존재하는 않는 요청 목록 return
	sameRuleList := sameRuleCheck(searchSecurityInfo.SecurityRules, securityRules, Const_Remove)
	if len(*sameRuleList) == 0 {
		return false, errors.New("Same SecurityRule is not exists")
	}

	//fmt.Println("securityInfo.securityrule size- : ", len(*securityInfo.SecurityRules))
	for _, securityRule := range *securityRules {
		//for ruleIndex, securityRule := range *securityInfo.SecurityRules {
		// firewall 삭제를 위한 resource ID 추출
		resourceId := ""
		for _, firewallInfo := range firewallList {
			for _, item := range firewallInfo.Items {
				var portArr []string
				var fromPort string
				var toPort string
				var ipProtocol string
				direction := ""
				cidr := ""

				if strings.EqualFold(item.Direction, "EGRESS") {
					cidr = strings.Join(item.DestinationRanges, ", ")
					direction = "outbound"
				} else {
					cidr = strings.Join(item.SourceRanges, ", ")
					direction = "inbound"
				}

				for _, firewallRule := range item.Allowed {
					fmt.Println("firewallRule : ", firewallRule)
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

					fmt.Println("firewalrulel.ipProtocol- : ", ipProtocol)

				} // end of firewall rule

				fmt.Println("Direction : ", item.Direction, " : ", direction, " : ", securityRule.Direction)
				fmt.Println("Cidr : ", cidr, " : ", securityRule.CIDR)
				fmt.Println("portArr : ", portArr)
				fmt.Println("fromport : ", fromPort, " : ", securityRule.FromPort)
				fmt.Println("toport : ", toPort, " : ", securityRule.ToPort)
				fmt.Println("ipProtocol : ", ipProtocol, " : ", securityRule.IPProtocol)
				// 조건이 동일한 resource ID
				if strings.EqualFold(direction, securityRule.Direction) && strings.EqualFold(cidr, securityRule.CIDR) && strings.EqualFold(fromPort, securityRule.FromPort) && strings.EqualFold(toPort, securityRule.ToPort) && strings.EqualFold(ipProtocol, securityRule.IPProtocol) {
					resourceId = item.Name
					break
				}
			}

			if strings.EqualFold(resourceId, "") {
				//return false, errors.New("Cannot get a resourceID")
				fmt.Println("cannot get a resourceID : ")
				continue
			}

			// 삭제 호출
			_, err := securityHandler.firewallDelete(securityGroupTag, resourceId, firewallInfo) // TODO : group 전체를 지우고 있음. 1개만 지울 수 있게 보완 필요
			if err != nil {
				return false, err
			}
			//callLogInfo = call.CLOUDLOGSCHEMA{
			//	CloudOS:      call.GCP,
			//	RegionZone:   securityHandler.Region.Zone,
			//	ResourceType: call.SECURITYGROUP,
			//	ResourceName: securityGroupTag,
			//	CloudOSAPI:   "Firewalls.Delete()",
			//	ElapsedTime:  "",
			//	ErrorMSG:     "",
			//}
			//fmt.Println("resourceId : ", resourceId, ruleIndex)
			//// 해당 firewall의 resourceID 로 Delete 호출 : 싹지우는 것 같은데???
			//res, err := securityHandler.Client.Firewalls.Delete(projectID, resourceId).Do()
			//
			//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
			//if err != nil {
			//	callLogInfo.ErrorMSG = err.Error()
			//	callogger.Error(call.String(callLogInfo))
			//	cblogger.Error(err)
			//	return false, err
			//}
			//callogger.Info(call.String(callLogInfo))
			//fmt.Println("remove result : ", resourceId, res)
			//
			//operation := securityHandler.Client.GlobalOperations.Get(projectID, res.ClientOperationId)
			//fmt.Println("operation", operation)
			//time.Sleep(time.Second * 3) // TODO : 매 호출마다 3초씩 기다리지 말고 CH를 이용하여 req가 모두 완료될 때까지 기다린 후 처리.
			//operation2 := securityHandler.Client.GlobalOperations.Get(projectID, res.ClientOperationId)
			//fmt.Println("operation", operation2)
			//
			////if err = op.Wait(ctx); err != nil {
			////	return fmt.Errorf("unable to wait for the operation: %v", err)
			////}
			////operation := securityHandler.Client.GlobalOperations.Get(projectID, res.ClientOperationId)
			////operation.status : DONE, PENDING, RUNNING (enum)
			//
			////// chan
			////var waitGroup sync.WaitGroup
			////ch := make(chan int)
			////waitGroup.Add(ruleIndex)
			//
			//time.Sleep(time.Second * 3)
			// TODO : 호출결과 받을 떄까지 ch로 제어하도록 보완할 것.
		}
	}
	// TODO : 호출결과가 모두 성공일 때 true를 return하도록 변경
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
//	//fmt.Println("create result : ", res)
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
		fmt.Println("itemName : ", itemName, tag, strings.Index(itemName, tag))
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
func isExistsNameInTags(name string, tags []string) bool {
	//fmt.Println("isExistsNameInTags : ", name, tags)
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

	fmt.Println("********* ", securityGroupNameMap)
	var returnFirewallList []compute.FirewallList
	for _, sgKey := range securityGroupNameMap {
		var returnFirewall compute.FirewallList
		var returnFirewallItemList []*compute.Firewall
		//fmt.Println("returnFirewallItemList before length  : ", len(returnFirewallItemList))
		for _, item := range firewallList.Items {
			//fmt.Println("get security list result : ", sgKey, item)
			if isExistsNameInTags(sgKey, item.SourceTags) {
				//fmt.Println("SourceTags : ", sgKey, item.SourceTags)
				returnFirewallItemList = append(returnFirewallItemList, item)
				continue
			}
			if isExistsNameInTags(sgKey, item.TargetTags) {
				//fmt.Println("TargetTags : ", sgKey, item.TargetTags)
				returnFirewallItemList = append(returnFirewallItemList, item)
				continue
			}

			if strings.EqualFold(sgKey, item.Name) {
				//fmt.Println("Name : ", sgKey, item.Name)
				returnFirewallItemList = append(returnFirewallItemList, item)
			}
			//firewallItemList = append(firewallItemList, item)
		}
		//fmt.Println("returnFirewallItemList length  : ", len(returnFirewallItemList))
		//fmt.Println("returnFirewallItemList  : ", returnFirewallItemList)
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
		fmt.Println("get security list result : ", item)

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
		fmt.Println("SourceTags : ", item.SourceTags)
		//TargetServiceAccounts []string `json:"targetServiceAccounts,omitempty"`
		//TargetTags []string `json:"targetTags,omitempty"`
		fmt.Println("TargetTags : ", item.TargetTags)
		//googleapi.ServerResponse `json:"-"`
		//ForceSendFields []string `json:"-"`
		//
		//NullFields []string `json:"-"`

		cidr := ""
		if strings.EqualFold(item.Direction, "INGRESS") {
			cidr = strings.Join(item.SourceRanges, ", ")
		} else {
			cidr = strings.Join(item.DestinationRanges, ", ")
		}
		fmt.Println("cidr : ", cidr)

		var portArr []string
		var fromPort string
		var toPort string
		var ipProtocol string

		for _, firewallRule := range item.Allowed {
			if ports := firewallRule.Ports; ports != nil {
				portArr = strings.Split(firewallRule.Ports[0], "-")
				fromPort = portArr[0]
				if len(portArr) > 1 {
					toPort = portArr[len(portArr)-1]
				} else {
					toPort = ""
				}

			} else {
				fromPort = ""
				toPort = ""
			}

			ipProtocol = firewallRule.IPProtocol
			fmt.Println("ipProtocol : ", ipProtocol)

			ruleInfo := irs.SecurityRuleInfo{
				FromPort:   fromPort,
				ToPort:     toPort,
				IPProtocol: ipProtocol,
				Direction:  item.Direction,
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
		fmt.Println("securityRules : ", securityRules)
		fmt.Println("securityRules length: ", len(securityRules))
	} // end of result.items
	fmt.Println("securityInfo : ", securityInfo)
	return securityInfo, nil
}

// 동일한 rule이 있는지 check
// action = add 면 존재하는 rule 목록 반환 : 이미있는 rule은 추가하지 않음
// action = remove 면  존재하지 않는 rule 목록 반환 : 없는 rule은 삭제하지 않음
func sameRuleCheck(searchedSecurityRules *[]irs.SecurityRuleInfo, requestedSecurityRules *[]irs.SecurityRuleInfo, action string) *[]irs.SecurityRuleInfo {

	var checkResult []irs.SecurityRuleInfo
	for _, reqRule := range *requestedSecurityRules {
		hasFound := false
		reqRulePort := ""
		if reqRule.FromPort == "" {
			reqRulePort = reqRule.ToPort
		} else if reqRule.ToPort == "" {
			reqRulePort = reqRule.FromPort
		} else if reqRule.FromPort == reqRule.ToPort {
			reqRulePort = reqRule.FromPort
		} else {
			reqRulePort = reqRule.FromPort + "-" + reqRule.ToPort
		}

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

			if strings.EqualFold(reqRule.Direction, searchedRule.Direction) && strings.EqualFold(reqRule.IPProtocol, searchedRule.IPProtocol) && strings.EqualFold(reqRulePort, searchedRulePort) && strings.EqualFold(reqRule.CIDR, searchedRule.CIDR) {
				hasFound = true
			}

			// add일 때는 존재하는게 있으면 안됨.
			if hasFound && action == Const_Add {
				cblogger.Info("add")
				checkResult = append(checkResult, reqRule)
				hasFound = false // 초기화
				break            // 찾았으면 searchedRule에서는 더 찾지 않아도 됨.
			}
		}

		// remove일 때는 없으면 안됨(존재해야 함)
		if !hasFound && action == Const_Remove {
			cblogger.Info("remove")
			checkResult = append(checkResult, reqRule)
		}
		hasFound = false // 초기화
	}
	return &checkResult
}

// Tag로 묶인 firewall의 max sequence 추출
func maxFirewallSequence(firewallList []compute.FirewallList, direction string) int {
	maxSequence := 0

	namingRule := ""
	for _, firewallInfo := range firewallList {
		for _, item := range firewallInfo.Items {
			// naming rule

			if strings.EqualFold(direction, "INGRESS") || strings.EqualFold(direction, "inbound") {
				namingRule = "-i-"
			} else if strings.EqualFold(direction, "EGRESS") || strings.EqualFold(direction, "outbound") {
				namingRule = "-o-"
			} else {
				continue
			}
			str := item.Name[len(item.Name)-6:]
			if strings.Index(str, namingRule) == 0 {
				curSequence, _ := strconv.Atoi(str[:3])
				if curSequence > maxSequence {
					maxSequence = curSequence
				}
				fmt.Println("str : ", str)
				fmt.Println("curSequence : ", curSequence)

			}
		}
	}

	return maxSequence
}

const (
	Const_Add    = "add"
	Const_Remove = "remove"
)

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
	fmt.Println("create default firewall rule result : ", res)

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
		fmt.Println("firewallDelete ", securityGroupTag, " : ", resourceID)
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

		// 해당 firewall의 resourceID 로 Delete 호출 : 싹지우는 것 같은데???
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
		fmt.Println("remove result : ", resourceID, res)

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
