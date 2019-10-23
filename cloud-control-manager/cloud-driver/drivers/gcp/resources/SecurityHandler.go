package resources

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	idrv "../../../interfaces"
	irs "../../../interfaces/resources"
	compute "google.golang.org/api/compute/v1"
)

type GCPSecurityHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

func (securityHandler *GCPSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	projectID := securityHandler.Credential.ProjectID
	// @TODO: SecurityGroup 생성 요청 파라미터 정의 필요
	ports := *securityReqInfo.SecurityRules
	var firewallAllowed []*compute.FirewallAllowed

	for _, item := range ports {
		var port string
		fp := item.FromPort
		tp := item.ToPort

		if tp != "" && fp != "" {
			port = fp + "-" + tp
		}
		if tp != "" && fp == "" {
			port = tp
		}
		if tp == "" && fp != "" {
			port = fp
		}

		firewallAllowed = append(firewallAllowed, &compute.FirewallAllowed{
			IPProtocol: item.IPProtocol,
			Ports: []string{
				port,
			},
		})
	}
	fireWall := &compute.Firewall{
		Allowed:   firewallAllowed,
		Direction: securityReqInfo.Direction, //INGRESS(inbound), EGRESS(outbound)
		SourceRanges: []string{
			"0.0.0.0/0",
		},
		Name: securityReqInfo.Name,
	}

	res, err := securityHandler.Client.Firewalls.Insert(projectID, fireWall).Do()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("create result : ", res)
	time.Sleep(time.Second * 3)
	secInfo, err := securityHandler.GetSecurity(securityReqInfo.Name)

	return secInfo, err

}

func (securityHandler *GCPSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	//result, err := securityHandler.Client.ListAll(securityHandler.Ctx)
	projectID := securityHandler.Credential.ProjectID
	result, err := securityHandler.Client.Firewalls.List(projectID).Do()
	var securityInfo []*irs.SecurityInfo
	for _, item := range result.Items {
		name := item.Name
		secInfo, err := securityHandler.GetSecurity(name)
		if err != nil {
			log.Fatal(err)
		}
		securityInfo = append(securityInfo, &secInfo)
	}

	return securityInfo, err
}

func (securityHandler *GCPSecurityHandler) GetSecurity(securityID string) (irs.SecurityInfo, error) {
	projectID := securityHandler.Credential.ProjectID

	security, err := securityHandler.Client.Firewalls.Get(projectID, securityID).Do()
	if err != nil {
		log.Fatal(err)

	}
	var securityRules []irs.SecurityRuleInfo
	for _, item := range security.Allowed {
		portArr := strings.Split(item.Ports[0], "-")
		securityRules = append(securityRules, irs.SecurityRuleInfo{
			FromPort:   portArr[0],
			ToPort:     portArr[len(portArr)-1],
			IPProtocol: item.IPProtocol,
		})
	}

	securityInfo := irs.SecurityInfo{
		Id:        strconv.FormatUint(security.Id, 10),
		Name:      security.Name,
		Direction: security.Direction,
		KeyValueList: []irs.KeyValue{
			{"Priority", strconv.FormatInt(security.Priority, 10)},
			{"SourceRanges", security.SourceRanges[0]},
		},
		SecurityRules: &securityRules,
	}

	return securityInfo, nil
}

func (securityHandler *GCPSecurityHandler) DeleteSecurity(securityID string) (bool, error) {
	projectID := securityHandler.Credential.ProjectID

	res, err := securityHandler.Client.Firewalls.Delete(projectID, securityID).Do()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)
	return true, err
}
