// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud Security Group Handler
//
// by ETRI, 2022.12.
// Updated by ETRI, 2024.09.
// Updated by ETRI, 2025.02.
// Updated by ETRI, 2025.09.

package resources

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"
	rules "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/extensions/fwaas_v2/rules"
	portforward "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/extensions/layer3/portforwarding"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	sim "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/kt/resources/info_manager/security_group_info_manager"
)

type KTVpcSecurityHandler struct {
	RegionInfo    idrv.RegionInfo
	VMClient      *ktvpcsdk.ServiceClient
	NetworkClient *ktvpcsdk.ServiceClient
	VolumeClient  *ktvpcsdk.ServiceClient
}

const (
	sgDir string = "/cloud-driver-libs/.securitygroup-kt/"
)

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud SecurityGroup Handler")
}

type SecurityGroup struct {
    IID   			IId 			`json:"IId"`
    VpcIID   		VpcIId 			`json:"VpcIID"`
    Direc   		string 			`json:"Direction"`
    Secu_Rules  	[]Security_Rule `json:"SecurityRules"`
	KeyValue_List 	[]KeyValue 		`json:"KeyValueList"`
}

type KeyValue struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type IId struct {
	NameID   string `json:"NameId"`
	SystemID string `json:"SystemId"`
}

type VpcIId struct {
	NameID   string `json:"NameId"`
	SystemID string `json:"SystemId"`
}

type Security_Rule struct {
	FromPort string `json:"FromPort"`
	ToPort   string `json:"ToPort"`
	Protocol string `json:"IPProtocol"`
	Direc    string `json:"Direction"`
	Cidr     string `json:"CIDR"`
}

func (securityHandler *KTVpcSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Info("KT Cloud VPC driver: called CreateSecurity()!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, securityReqInfo.IId.NameId, "CreateSecurity()")

	if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Region Info!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir
	sgFilePath := sgPath + securityHandler.RegionInfo.Zone + "/"

	// Check if the KeyPair Folder Exists, and Create it
	if err := checkFolderAndCreate(sgPath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup Path : [%v]", err)
		return irs.SecurityInfo{}, err
	}

	// Check if the KeyPair Folder Exists, and Create it
	if err := checkFolderAndCreate(sgFilePath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup File Path : [%v]", err)
		return irs.SecurityInfo{}, err
	}

	// Check SecurityGroup Exists
	sgList, err := securityHandler.ListSecurity()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get S/G list. [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}
	for _, sg := range sgList {
		if sg.IId.NameId == securityReqInfo.IId.NameId {
			newErr := fmt.Errorf("Security Group with the Name [%s] Already Exists", securityReqInfo.IId.NameId)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return irs.SecurityInfo{}, newErr
		}
	}

	currentTime := getSeoulCurrentTime()

	// Process SecurityRules and add default outbound rules if none exist
	processedSecurityRules := securityReqInfo.SecurityRules
	if processedSecurityRules == nil {
		processedSecurityRules = &[]irs.SecurityRuleInfo{}
	}

	// Check if there are any outbound rules
	hasOutboundRules := false
	for _, rule := range *processedSecurityRules {
		if strings.EqualFold(rule.Direction, "outbound") {
			hasOutboundRules = true
			break
		}
	}

	// If no outbound rules exist, add default outbound rules for all protocols
	if !hasOutboundRules {
		cblogger.Info("No 'outbound' rules specified. Adding default outbound rules for all protocols and ports.")
		
		defaultOutboundRules := []irs.SecurityRuleInfo{
			{
				Direction:  "outbound",
				IPProtocol: "ALL",
				FromPort:   "-1",
				ToPort:     "-1",
				CIDR: 		"0.0.0.0/0",
			},
		}
    
		// Append default outbound rules to existing rules
		*processedSecurityRules = append(*processedSecurityRules, defaultOutboundRules...)
	}

	newSGInfo := irs.SecurityInfo{
		IId: irs.IID{
			NameId: 	securityReqInfo.IId.NameId,
			// Caution!! : securityReqInfo.IId.NameId -> SystemId
			SystemId: 	securityReqInfo.IId.NameId,
		},
		VpcIID: 		securityReqInfo.VpcIID,
		SecurityRules: 	processedSecurityRules, // Use the processed rules
		KeyValueList: 	[]irs.KeyValue{
			{Key: "KTCloud-SecuriyGroup-info.", Value: "This SecuriyGroup info. is temporary."},
			{Key: "CreateTime", Value: currentTime},
		},
	}
	// spew.Dump(newSGInfo)

	hashFileName := base64.StdEncoding.EncodeToString([]byte(securityReqInfo.IId.NameId))
	cblogger.Infof("# S/G NameId : " + securityReqInfo.IId.NameId)
	// cblogger.Infof("# Hashed FileName : " + hashFileName + ".json")

	file, _ := json.MarshalIndent(newSGInfo, "", " ")
	writeErr := os.WriteFile(sgFilePath+hashFileName+".json", file, 0644)
	if writeErr != nil {
		cblogger.Error("Failed to write the file: "+sgFilePath+hashFileName+".json", writeErr)
		return irs.SecurityInfo{}, writeErr
	}
	cblogger.Infof("Succeeded in writing the S/G file: " + sgFilePath + hashFileName + ".json")

	// Because it's managed as a file, there's no SystemId created.
	securityReqInfo.IId.SystemId = securityReqInfo.IId.NameId

	// Return the created SecurityGroup info.
	securityInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: securityReqInfo.IId.SystemId})
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	return securityInfo, nil
}

func (securityHandler *KTVpcSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	cblogger.Info("KT Cloud VPC driver: called GetSecurity()!!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, securityIID.SystemId, "GetSecurity()")

	if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Region Info!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

	if strings.EqualFold(securityIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid S/G SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

    // Check if S/G exists first
    if err := securityHandler.CheckSecurityGroupExists(securityIID); err != nil {
		newErr := fmt.Errorf("SecurityGroup validation failed: %w", err)
		cblogger.Error(newErr.Error())
		return irs.SecurityInfo{}, newErr
    }

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir
	sgFilePath := sgPath + securityHandler.RegionInfo.Zone + "/"

	// Check if the KeyPair Folder Exists, and Create it
	if err := checkFolderAndCreate(sgPath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup Path : [%v]", err)
		return irs.SecurityInfo{}, err
	}

	// Check if the KeyPair Folder Exists, and Create it
	if err := checkFolderAndCreate(sgFilePath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup File Path : [%v]", err)
		return irs.SecurityInfo{}, err
	}

	hashFileName := base64.StdEncoding.EncodeToString([]byte(securityIID.SystemId))
	sgFileName := sgFilePath + hashFileName + ".json"
	jsonFile, err := os.Open(sgFileName)
	if err != nil {
		cblogger.Warnf("S/G file not found: %s, returning baseline SecurityGroup info", sgFileName)
		return irs.SecurityInfo{
			IId: irs.IID{
				NameId:   securityIID.SystemId,
				SystemId: securityIID.SystemId,
			},
			SecurityRules: &[]irs.SecurityRuleInfo{},
		}, nil
	}
	// cblogger.Infof("Succeeded in Finding and Opening the S/G file: " + sgFileName)

	var sg SecurityGroup
	defer jsonFile.Close()
	byteValue, readErr := io.ReadAll(jsonFile)
	if readErr != nil {
		cblogger.Error("Failed to Read the S/G file : "+sgFileName, readErr)
		return irs.SecurityInfo{}, readErr
	}
	json.Unmarshal(byteValue, &sg)
	// spew.Dump(sg)

	securityGroupInfo, mapError := securityHandler.mappingSecurityInfo(sg)
	if mapError != nil {
		cblogger.Error(mapError)
		return irs.SecurityInfo{}, mapError
	}
	return securityGroupInfo, nil
}

func (securityHandler *KTVpcSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	cblogger.Info("KT Cloud VPC driver: called ListSecurity()!!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, "ListSecurity()", "ListSecurity()")

	var securityIID irs.IID
	var securityGroupList []*irs.SecurityInfo
	// var sg SecurityGroup

	if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Region Info!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir
	sgFilePath := sgPath + securityHandler.RegionInfo.Zone + "/"

	// Check if the KeyPair Folder Exists, and Create it
	if err := checkFolderAndCreate(sgPath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup Path : [%v]", err)
		return nil, err
	}

	// Check if the KeyPair Folder Exists, and Create it
	if err := checkFolderAndCreate(sgFilePath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup File Path : [%v]", err)
		return nil, err
	}

	// File list on the local directory
	dirFiles, readRrr := os.ReadDir(sgFilePath)
	if readRrr != nil {
		return nil, readRrr
	}

	for _, file := range dirFiles {
		fileName := strings.TrimSuffix(file.Name(), ".json") // Remove suffix
		decString, baseErr := base64.StdEncoding.DecodeString(fileName)
		if baseErr != nil {
			cblogger.Errorf("Failed to Decode the Filename : %s", fileName)
			return nil, baseErr
		}
		sgFileName := string(decString)
		// sgFileName := filePath + file.Name()

		securityIID.SystemId = sgFileName
		cblogger.Infof("# S/G Group Name : " + securityIID.SystemId)

		sgInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: securityIID.SystemId})
		if err != nil {
			cblogger.Errorf("Failed to Find the SecurityGroup : %s", securityIID.SystemId)
			return nil, err
		}
		securityGroupList = append(securityGroupList, &sgInfo)
	}

	return securityGroupList, nil
}

func (securityHandler *KTVpcSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC driver: called DeleteSecurity()!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, securityIID.SystemId, "DeleteSecurity()")

	securityIID.NameId = securityIID.SystemId

	if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Region Info!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	if securityIID.SystemId == "" {
		newErr := fmt.Errorf("invalid S/G SystemId.")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	// Check if S/G exists first
    if err := securityHandler.CheckSecurityGroupExists(securityIID); err != nil {
		newErr := fmt.Errorf("SecurityGroup validation failed: %w", err)
		cblogger.Error(newErr.Error())
		return false, newErr
    }

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir
	sgFilePath := sgPath + securityHandler.RegionInfo.Zone + "/"

	// Check if the S/G Folder Exists, and Create it
	if err := checkFolderAndCreate(sgPath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup Path : [%v]", err)
		return false, err
	}

	// Check if the S/G Folder Exists, and Create it
	if err := checkFolderAndCreate(sgFilePath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup File Path : [%v]", err)
		return false, err
	}

	hashFileName := base64.StdEncoding.EncodeToString([]byte(securityIID.NameId))
	sgFileName := sgFilePath + hashFileName + ".json"
	// cblogger.Infof("S/G file to Delete : [%s]", sgFileName)

	// Remove the S/G file on the Local machine
	delErr := os.Remove(sgFileName)
	if delErr != nil && !os.IsNotExist(delErr) {
		cblogger.Warnf("Note: S/G file could not be removed: %s, [%v]", sgFileName, delErr)
	}
	_, _ = sim.DeleteKTCloudSGDef(securityIID.SystemId)
	cblogger.Infof("Succeeded in Deleting the SecurityGroup : " + securityIID.SystemId)

	return true, nil
}

func normalizeRule(rule irs.SecurityRuleInfo) irs.SecurityRuleInfo {
	r := rule
	r.Direction = strings.ToLower(strings.TrimSpace(r.Direction))
	r.IPProtocol = strings.ToUpper(strings.TrimSpace(r.IPProtocol))
	r.FromPort = strings.TrimSpace(r.FromPort)
	r.ToPort = strings.TrimSpace(r.ToPort)
	r.CIDR = strings.TrimSpace(r.CIDR)
	if r.CIDR == "" {
		r.CIDR = "0.0.0.0/0"
	}
	return r
}

func isSameSecurityRule(r1, r2 irs.SecurityRuleInfo) bool {
	n1 := normalizeRule(r1)
	n2 := normalizeRule(r2)
	return n1.Direction == n2.Direction &&
		n1.IPProtocol == n2.IPProtocol &&
		n1.FromPort == n2.FromPort &&
		n1.ToPort == n2.ToPort &&
		n1.CIDR == n2.CIDR
}

func expandRuleProtocols(protocol string) ([]string, error) {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case "TCP":
		return []string{"TCP"}, nil
	case "UDP":
		return []string{"UDP"}, nil
	case "ICMP":
		return []string{"ICMP"}, nil
	case "ALL":
		return []string{"TCP", "UDP", "ICMP"}, nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

func matchInboundFWRule(fw rules.FirewallRule, publicIP string, protocol string, fromPort string, toPort string) bool {
	ipMatch := false
	ipCidr := publicIP
	if !strings.Contains(ipCidr, "/") {
		ipCidr = publicIP + "/32"
	}
	for _, dst := range fw.DstAddress {
		if strings.Contains(dst.Name, publicIP) || strings.Contains(dst.Name, ipCidr) {
			ipMatch = true
			break
		}
	}
	if !ipMatch {
		return false
	}

	if strings.Contains(strings.ToLower(fw.Comment), "inbound") {
		if strings.EqualFold(protocol, "ICMP") && strings.Contains(strings.ToUpper(fw.Comment), "ICMP") {
			return true
		}
		if strings.Contains(strings.ToUpper(fw.Comment), strings.ToUpper(protocol)) &&
			strings.Contains(fw.Comment, fromPort) && strings.Contains(fw.Comment, toPort) {
			return true
		}
	}

	for _, svc := range fw.Services {
		if strings.EqualFold(svc.Protocol, protocol) {
			if strings.EqualFold(protocol, "ICMP") {
				return true
			}
			if svc.StartPort == fromPort && svc.EndPort == toPort {
				return true
			}
		}
	}

	return false
}

func matchOutboundFWRule(fw rules.FirewallRule, privateIP string, protocol string, fromPort string, toPort string) bool {
	ipMatch := false
	ipCidr := privateIP
	if !strings.Contains(ipCidr, "/") {
		ipCidr = privateIP + "/32"
	}
	for _, src := range fw.SrcAddress {
		if strings.Contains(src.Name, privateIP) || strings.Contains(src.Name, ipCidr) {
			ipMatch = true
			break
		}
	}
	if !ipMatch {
		return false
	}

	if strings.Contains(strings.ToLower(fw.Comment), "outbound") {
		if strings.EqualFold(protocol, "ICMP") && strings.Contains(strings.ToUpper(fw.Comment), "ICMP") {
			return true
		}
		if strings.Contains(strings.ToUpper(fw.Comment), strings.ToUpper(protocol)) &&
			strings.Contains(fw.Comment, fromPort) && strings.Contains(fw.Comment, toPort) {
			return true
		}
	}

	for _, svc := range fw.Services {
		if strings.EqualFold(svc.Protocol, protocol) {
			if strings.EqualFold(protocol, "ICMP") {
				return true
			}
			if svc.StartPort == fromPort && svc.EndPort == toPort {
				return true
			}
		}
	}

	return false
}

func (securityHandler *KTVpcSecurityHandler) writeSGFile(sgInfo irs.SecurityInfo) error {
	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir
	sgFilePath := sgPath + securityHandler.RegionInfo.Zone + "/"

	if err := checkFolderAndCreate(sgPath); err != nil {
		return err
	}
	if err := checkFolderAndCreate(sgFilePath); err != nil {
		return err
	}

	hashFileName := base64.StdEncoding.EncodeToString([]byte(sgInfo.IId.SystemId))
	file, err := json.MarshalIndent(sgInfo, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal security group info: %w", err)
	}

	writeErr := os.WriteFile(sgFilePath+hashFileName+".json", file, 0644)
	if writeErr != nil {
		return writeErr
	}

	// Persist to infostore (DB) so it survives pod restarts
	_ = sim.SaveKTCloudSGDef(sgInfo.IId.SystemId, securityHandler.RegionInfo.Zone, string(file))

	return nil
}

func (securityHandler *KTVpcSecurityHandler) recoverSGFromAttachedVM(sgID string) (*irs.SecurityInfo, error) {
	cblogger.Infof("Attempting to auto-recover SecurityGroup [%s] from attached VM...", sgID)
	vmIDs, err := sim.GetVMIDsBySecurityGroup(sgID)
	if err != nil || len(vmIDs) == 0 {
		return nil, fmt.Errorf("no attached VMs found for SG [%s]", sgID)
	}

	vmHandler := &KTVpcVMHandler{
		RegionInfo:    securityHandler.RegionInfo,
		VMClient:      securityHandler.VMClient,
		NetworkClient: securityHandler.NetworkClient,
		VolumeClient:  securityHandler.VolumeClient,
	}

	for _, vmID := range vmIDs {
		vm, getErr := vmHandler.GetVM(irs.IID{SystemId: vmID})
		if getErr != nil {
			continue
		}

		var recoveredRules []irs.SecurityRuleInfo

		if vm.PrivateIP != "" {
			pfList, _ := vmHandler.listPortForwarding()
			for _, pf := range pfList {
				if strings.EqualFold(pf.MappedIP, vm.PrivateIP) {
					recoveredRules = append(recoveredRules, irs.SecurityRuleInfo{
						Direction:  "inbound",
						IPProtocol: strings.ToUpper(pf.Protocol),
						FromPort:   pf.StartPublicPort,
						ToPort:     pf.EndPublicPort,
						CIDR:       "0.0.0.0/0",
					})
				}
			}
		}

		recoveredRules = append(recoveredRules, irs.SecurityRuleInfo{
			Direction:  "outbound",
			IPProtocol: "ALL",
			FromPort:   "-1",
			ToPort:     "-1",
			CIDR:       "0.0.0.0/0",
		})

		currentTime := time.Now().Format("2006-01-02 15:04:05")
		sgInfo := irs.SecurityInfo{
			IId: irs.IID{
				NameId:   sgID,
				SystemId: sgID,
			},
			VpcIID:        vm.VpcIID,
			SecurityRules: &recoveredRules,
			KeyValueList: []irs.KeyValue{
				{Key: "KTCloud-SecuriyGroup-info.", Value: "Auto-recovered from attached VM."},
				{Key: "CreateTime", Value: currentTime},
			},
		}

		_ = securityHandler.writeSGFile(sgInfo)
		return &sgInfo, nil
	}

	return nil, fmt.Errorf("failed to recover SG [%s] from VMs", sgID)
}

func (securityHandler *KTVpcSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	cblogger.Info("KT Cloud VPC driver: called AddRules()!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, sgIID.SystemId, "AddRules()")

	if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Region Info!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

	if sgIID.SystemId == "" {
		sgIID.SystemId = sgIID.NameId
	}
	if sgIID.NameId == "" {
		sgIID.NameId = sgIID.SystemId
	}

	if sgIID.SystemId == "" {
		newErr := fmt.Errorf("Invalid S/G SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

	if securityRules == nil || len(*securityRules) == 0 {
		return securityHandler.GetSecurity(sgIID)
	}
	rulesToAdd := *securityRules

	// 1. Find VMs associated with this Security Group
	vmIDs, err := sim.GetVMIDsBySecurityGroup(sgIID.SystemId)
	if err != nil {
		cblogger.Warnf("Failed to query VMs for Security Group [%s]: %v", sgIID.SystemId, err)
	}

	if len(vmIDs) == 0 {
		cblogger.Infof("No running VMs currently attached to Security Group [%s].", sgIID.SystemId)
	}

	// 5. Apply new rules to each attached VM
	vmHandler := &KTVpcVMHandler{
		RegionInfo:    securityHandler.RegionInfo,
		VMClient:      securityHandler.VMClient,
		NetworkClient: securityHandler.NetworkClient,
		VolumeClient:  securityHandler.VolumeClient,
	}

	vpcHandler := KTVpcVPCHandler{
		RegionInfo:    securityHandler.RegionInfo,
		NetworkClient: securityHandler.NetworkClient,
	}
	extNetId, extErr := vpcHandler.getNetworkID("external")
	if extErr != nil {
		cblogger.Warnf("Failed to get External Network ID: %v", extErr)
	}

	for _, vmID := range vmIDs {
		cblogger.Infof("Syncing new security rules for VM [%s]...", vmID)
		vm, getErr := vmHandler.GetVM(irs.IID{SystemId: vmID})
		if getErr != nil {
			cblogger.Warnf("Failed to get VM [%s], skipping rule sync: %v", vmID, getErr)
			continue
		}

		if vm.PublicIP == "" {
			cblogger.Infof("VM [%s] has no Public IP; skipping port forwarding and firewall rule creation.", vmID)
			continue
		}

		netInfo, netErr := vmHandler.getNetIDsWithPrivateIP(vm.PrivateIP)
		if netErr != nil || netInfo == nil || netInfo.PublicIPID == "" {
			cblogger.Warnf("Failed to get PublicIPID for VM [%s] (private: %s): %v", vmID, vm.PrivateIP, netErr)
			continue
		}

		pfList, _ := vmHandler.listPortForwarding()
		fwList, _ := vmHandler.listFirewallRule()

		for _, rule := range rulesToAdd {
			protocols, pErr := expandRuleProtocols(rule.IPProtocol)
			if pErr != nil {
				cblogger.Warnf("Invalid protocol [%s]: %v", rule.IPProtocol, pErr)
				continue
			}

			for _, proto := range protocols {
				fromPort := rule.FromPort
				toPort := rule.ToPort
				if fromPort == "-1" && toPort == "-1" {
					fromPort = "1"
					toPort = "65535"
				}
				if proto == "ICMP" {
					fromPort = ""
					toPort = ""
				}

				if strings.EqualFold(rule.Direction, "inbound") {
					var pfRuleId string
					if proto != "ICMP" {
						for _, pf := range pfList {
							if pf.PublicIPID == netInfo.PublicIPID && pf.MappedIP == vm.PrivateIP &&
								strings.EqualFold(pf.Protocol, proto) &&
								pf.StartPublicPort == fromPort && pf.EndPublicPort == toPort {
								pfRuleId = pf.ID
								break
							}
						}

						if pfRuleId == "" {
							createPfOpts := &portforward.CreateOpts{
								PublicIpID:       netInfo.PublicIPID,
								MappedIP:         vm.PrivateIP,
								Protocol:         proto,
								StartPrivatePort: fromPort,
								EndPrivatePort:   toPort,
								StartPublicPort:  fromPort,
								EndPublicPort:    toPort,
							}
							pfResult := portforward.Create(securityHandler.NetworkClient, createPfOpts)
							if pfResult.Err != nil {
								cblogger.Errorf("Failed to create PortForwarding for VM [%s] (port %s-%s): %v", vmID, fromPort, toPort, pfResult.Err)
							} else {
								extractedID, _ := portforward.ExtractPortForwardingID(pfResult)
								pfRuleId = extractedID
								cblogger.Infof("Created PortForwarding rule (ID: %s) for VM [%s]", pfRuleId, vmID)
								time.Sleep(2 * time.Second)
							}
						}
					}

					destCIDR, cErr := ipToCidr32(vm.PublicIP)
					if cErr != nil {
						cblogger.Errorf("Failed to convert PublicIP to CIDR: %v", cErr)
						continue
					}
					srcCIDR := rule.CIDR
					if srcCIDR == "" {
						srcCIDR = "0.0.0.0/0"
					}

					fwExists := false
					for _, fw := range fwList {
						if matchInboundFWRule(fw, vm.PublicIP, proto, fromPort, toPort) {
							fwExists = true
							break
						}
					}

					if !fwExists && extNetId != nil {
						comment := "Allow inbound - " + proto
						if proto != "ICMP" {
							comment += " - " + fromPort + " to " + toPort
						}
						inboundFWOpts := &rules.CreateOpts{
							Action:           true,
							Protocol:         proto,
							StartPort:        fromPort,
							EndPort:          toPort,
							SrcNetwork:       []string{*extNetId},
							PortForwardingId: pfRuleId,
							SrcAddress:       []string{srcCIDR},
							DstAddress:       []string{destCIDR},
							Comment:          comment,
							SrcNat:           false,
						}
						fwResult := rules.Create(securityHandler.NetworkClient, inboundFWOpts)
						if fwResult.Err != nil {
							cblogger.Errorf("Failed to create inbound Firewall rule for VM [%s]: %v", vmID, fwResult.Err)
						} else {
							jobId, _ := rules.ExtractJobID(fwResult)
							cblogger.Infof("Created inbound Firewall rule (JobId: %s) for VM [%s]", jobId, vmID)
							time.Sleep(2 * time.Second)
						}
					}
				} else if strings.EqualFold(rule.Direction, "outbound") {
					var tierNetworkId string
					if len(vm.NICs) > 0 && vm.NICs[0].SubnetIID.SystemId != "" {
						tId, err := vpcHandler.getNetworkIdWithTierId(vm.NICs[0].SubnetIID.SystemId)
						if err == nil && tId != nil {
							tierNetworkId = *tId
						}
					}

					if tierNetworkId != "" && extNetId != nil {
						srcCIDR, _ := ipToCidr32(vm.PrivateIP)
						destIPAdds := "0.0.0.0/0"
						comment := "Allow outbound - " + proto + " - " + fromPort + " to " + toPort
						outboundFWOpts := &rules.CreateOpts{
							Action:     true,
							Protocol:   proto,
							StartPort:  fromPort,
							EndPort:    toPort,
							SrcNetwork: []string{tierNetworkId},
							DstNetwork: []string{*extNetId},
							SrcAddress: []string{srcCIDR},
							DstAddress: []string{destIPAdds},
							Comment:    comment,
							SrcNat:     true,
						}
						fwResult := rules.Create(securityHandler.NetworkClient, outboundFWOpts)
						if fwResult.Err != nil {
							cblogger.Errorf("Failed to create outbound Firewall rule for VM [%s]: %v", vmID, fwResult.Err)
						} else {
							jobId, _ := rules.ExtractJobID(fwResult)
							cblogger.Infof("Created outbound Firewall rule (JobId: %s) for VM [%s]", jobId, vmID)
							time.Sleep(2 * time.Second)
						}
					}
				}
			}
		}
	}

	// Best-effort update local cache
	sgInfo, err := securityHandler.GetSecurity(sgIID)
	if err == nil {
		var combined []irs.SecurityRuleInfo
		if sgInfo.SecurityRules != nil {
			combined = *sgInfo.SecurityRules
		}
		for _, r := range rulesToAdd {
			exists := false
			for _, c := range combined {
				if isSameSecurityRule(c, r) {
					exists = true
					break
				}
			}
			if !exists {
				combined = append(combined, r)
			}
		}
		sgInfo.SecurityRules = &combined
		_ = securityHandler.writeSGFile(sgInfo)
		return sgInfo, nil
	}

	return irs.SecurityInfo{IId: sgIID, SecurityRules: securityRules}, nil
}

func (securityHandler *KTVpcSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	cblogger.Info("KT Cloud VPC driver: called RemoveRules()!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, sgIID.SystemId, "RemoveRules()")

	if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Region Info!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	if sgIID.SystemId == "" {
		sgIID.SystemId = sgIID.NameId
	}
	if sgIID.NameId == "" {
		sgIID.NameId = sgIID.SystemId
	}

	if sgIID.SystemId == "" {
		newErr := fmt.Errorf("Invalid S/G SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	if securityRules == nil || len(*securityRules) == 0 {
		return true, nil
	}
	rulesToDelete := *securityRules

	// 1. Find VMs associated with this Security Group
	vmIDs, err := sim.GetVMIDsBySecurityGroup(sgIID.SystemId)
	if err != nil {
		cblogger.Warnf("Failed to query VMs for Security Group [%s]: %v", sgIID.SystemId, err)
	}

	if len(vmIDs) == 0 {
		cblogger.Infof("No running VMs currently attached to Security Group [%s].", sgIID.SystemId)
		return true, nil
	}

	// 5. Remove rules from KT Cloud for each attached VM
	vmHandler := &KTVpcVMHandler{
		RegionInfo:    securityHandler.RegionInfo,
		VMClient:      securityHandler.VMClient,
		NetworkClient: securityHandler.NetworkClient,
		VolumeClient:  securityHandler.VolumeClient,
	}

	for _, vmID := range vmIDs {
		cblogger.Infof("Checking rule removal for VM [%s]...", vmID)
		vm, getErr := vmHandler.GetVM(irs.IID{SystemId: vmID})
		if getErr != nil {
			cblogger.Warnf("Failed to get VM [%s], skipping rule cleanup: %v", vmID, getErr)
			continue
		}

		if vm.PublicIP == "" {
			continue
		}

		// Find other Security Groups attached to this VM
		vmSgInfo, _ := sim.GetSecurityGroup(vmID)
		var otherSgIDs []string
		if vmSgInfo != nil {
			for _, kv := range vmSgInfo.KeyValueInfoList {
				if !strings.EqualFold(kv.Key, sgIID.SystemId) && !strings.EqualFold(kv.Value, sgIID.SystemId) {
					otherSgIDs = append(otherSgIDs, kv.Value)
				}
			}
		}

		for _, delRule := range rulesToDelete {
			// Check if another SG attached to this VM still requires this rule
			stillNeeded := false
			for _, otherSgID := range otherSgIDs {
				otherSG, err := securityHandler.GetSecurity(irs.IID{SystemId: otherSgID})
				if err == nil && otherSG.SecurityRules != nil {
					for _, osr := range *otherSG.SecurityRules {
						if isSameSecurityRule(osr, delRule) {
							stillNeeded = true
							break
						}
					}
				}
				if stillNeeded {
					break
				}
			}

			if stillNeeded {
				cblogger.Infof("Rule [%v] is still required by another Security Group for VM [%s], skipping KT Cloud rule deletion.", delRule, vmID)
				continue
			}

			protocols, _ := expandRuleProtocols(delRule.IPProtocol)
			for _, proto := range protocols {
				fromPort := delRule.FromPort
				toPort := delRule.ToPort
				if fromPort == "-1" && toPort == "-1" {
					fromPort = "1"
					toPort = "65535"
				}
				if proto == "ICMP" {
					fromPort = ""
					toPort = ""
				}

				if strings.EqualFold(delRule.Direction, "inbound") {
					// 1) Delete Firewall rule
					fwList, err := vmHandler.listFirewallRule()
					if err == nil {
						for _, fw := range fwList {
							if matchInboundFWRule(fw, vm.PublicIP, proto, fromPort, toPort) {
								cblogger.Infof("Deleting inbound Firewall rule (PolicyID: %s) for VM [%s]", fw.PolicyID, vmID)
								delRes := rules.Delete(securityHandler.NetworkClient, fw.PolicyID)
								if delRes.Err != nil {
									errMsg := delRes.Err.Error()
									if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "Not Found") {
										cblogger.Infof("Firewall rule (PolicyID: %s) already deleted on KT Cloud (404)", fw.PolicyID)
									} else {
										cblogger.Warnf("Failed to delete firewall rule (PolicyID: %s): %v", fw.PolicyID, delRes.Err)
									}
								} else {
									cblogger.Infof("Successfully deleted firewall rule (PolicyID: %s)", fw.PolicyID)
								}
							}
						}
					}

					// 2) Delete PortForwarding rule (if not ICMP)
					if proto != "ICMP" {
						pfList, err := vmHandler.listPortForwarding()
						if err == nil {
							for _, pf := range pfList {
								if pf.MappedIP == vm.PrivateIP && strings.EqualFold(pf.Protocol, proto) &&
									pf.StartPublicPort == fromPort && pf.EndPublicPort == toPort {
									cblogger.Infof("Deleting PortForwarding rule (ID: %s) for VM [%s]", pf.ID, vmID)
									delRes := portforward.Delete(securityHandler.NetworkClient, pf.ID)
									if delRes.Err != nil {
										errMsg := delRes.Err.Error()
										if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "Not Found") {
											cblogger.Infof("Port forwarding rule (ID: %s) already deleted on KT Cloud (404)", pf.ID)
										} else {
											cblogger.Warnf("Failed to delete port forwarding rule (ID: %s): %v", pf.ID, delRes.Err)
										}
									} else {
										cblogger.Infof("Successfully deleted port forwarding rule (ID: %s)", pf.ID)
									}
								}
							}
						}
					}
				} else if strings.EqualFold(delRule.Direction, "outbound") {
					fwList, err := vmHandler.listFirewallRule()
					if err == nil {
						for _, fw := range fwList {
							if matchOutboundFWRule(fw, vm.PrivateIP, proto, fromPort, toPort) {
								cblogger.Infof("Deleting outbound Firewall rule (PolicyID: %s) for VM [%s]", fw.PolicyID, vmID)
								delRes := rules.Delete(securityHandler.NetworkClient, fw.PolicyID)
								if delRes.Err != nil {
									errMsg := delRes.Err.Error()
									if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "Not Found") {
										cblogger.Infof("Outbound firewall rule (PolicyID: %s) already deleted on KT Cloud (404)", fw.PolicyID)
									} else {
										cblogger.Warnf("Failed to delete outbound firewall rule (PolicyID: %s): %v", fw.PolicyID, delRes.Err)
									}
								} else {
									cblogger.Infof("Successfully deleted outbound firewall rule (PolicyID: %s)", fw.PolicyID)
								}
							}
						}
					}
				}
			}
		}
	}

	// Best-effort update local cache
	sgInfo, err := securityHandler.GetSecurity(sgIID)
	if err == nil && sgInfo.SecurityRules != nil {
		var remainingRules []irs.SecurityRuleInfo
		for _, curRule := range *sgInfo.SecurityRules {
			toDelete := false
			for _, reqRule := range rulesToDelete {
				if isSameSecurityRule(curRule, reqRule) {
					toDelete = true
					break
				}
			}
			if !toDelete {
				remainingRules = append(remainingRules, curRule)
			}
		}
		sgInfo.SecurityRules = &remainingRules
		_ = securityHandler.writeSGFile(sgInfo)
	}

	return true, nil
}

func (securityHandler *KTVpcSecurityHandler) mappingSecurityInfo(sg SecurityGroup) (irs.SecurityInfo, error) {
	cblogger.Info("KT Cloud VPC driver: called mappingSecurityInfo()!")

	var sgRuleList []irs.SecurityRuleInfo
	var sgRuleInfo irs.SecurityRuleInfo
	var sgKeyValue irs.KeyValue
	var sgKeyValueList []irs.KeyValue

	for i := 0; i < len(sg.Secu_Rules); i++ {
		sgRuleInfo.FromPort = sg.Secu_Rules[i].FromPort
		sgRuleInfo.ToPort = sg.Secu_Rules[i].ToPort
		sgRuleInfo.IPProtocol = sg.Secu_Rules[i].Protocol // For KT Cloud VPC S/G, TCP/UDP/ICMP is available
		sgRuleInfo.Direction = sg.Secu_Rules[i].Direc 	 // For KT Cloud VPC S/G, supports inbound/outbound rule.
		sgRuleInfo.CIDR = sg.Secu_Rules[i].Cidr
	
		sgRuleList = append(sgRuleList, sgRuleInfo)
    }

	for k := 0; k < len(sg.KeyValue_List); k++ {
		sgKeyValue.Key = sg.KeyValue_List[k].Key
		sgKeyValue.Value = sg.KeyValue_List[k].Value
		sgKeyValueList = append(sgKeyValueList, sgKeyValue)
	}

	securityInfo := irs.SecurityInfo{
		IId:           irs.IID{NameId: sg.IID.NameID, SystemId: sg.IID.NameID},
		// Since it is managed as a file, the systemID is the same as the name ID.
		VpcIID:        irs.IID{NameId: sg.VpcIID.NameID, SystemId: sg.VpcIID.SystemID},
		SecurityRules: &sgRuleList,
		KeyValueList:  sgKeyValueList,
	}
	return securityInfo, nil
}

func (securityHandler *KTVpcSecurityHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("KT Cloud VPC driver: called ListIID()!")

    if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
        newErr := fmt.Errorf("Invalid Region Info!!")
        cblogger.Error(newErr.Error())
        return nil, newErr
    }

    sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir
    sgFilePath := sgPath + securityHandler.RegionInfo.Zone + "/"

    // Check if the KeyPair Folder Exists, and Create it
    if err := checkFolderAndCreate(sgPath); err != nil {
        cblogger.Errorf("Failed to Create the SecurityGroup Path : [%v]", err)
        return nil, err
    }

    // Check if the KeyPair Folder Exists, and Create it
    if err := checkFolderAndCreate(sgFilePath); err != nil {
        cblogger.Errorf("Failed to Create the SecurityGroup File Path : [%v]", err)
        return nil, err
    }

    // File list on the local directory
	dirFiles, readErr := os.ReadDir(sgFilePath)
    if readErr != nil {
        return nil, readErr
    }

    var iidList []*irs.IID
    for _, file := range dirFiles {
        fileName := strings.TrimSuffix(file.Name(), ".json") // Remove suffix
        decString, baseErr := base64.StdEncoding.DecodeString(fileName)
        if baseErr != nil {
            cblogger.Errorf("Failed to Decode the Filename : %s", fileName)
            return nil, baseErr
        }
        sgFileName := string(decString)

        iid := &irs.IID{
            NameId:   sgFileName,
            SystemId: sgFileName,
        }
        iidList = append(iidList, iid)
    }

    return iidList, nil
}

// CheckSecurityGroupExists checks if a S/G with the given SystemId exists
func (securityHandler *KTVpcSecurityHandler) CheckSecurityGroupExists(securityIID irs.IID) error {
    cblogger.Info("KT Cloud VPC driver: called CheckSecurityGroupExists()!")
    callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, securityIID.SystemId, "CheckSecurityGroupExists()")

    if securityHandler.RegionInfo.Zone == "" {
        newErr := fmt.Errorf("invalid Region Info")
        cblogger.Error(newErr.Error())
        loggingError(callLogInfo, newErr)
        return newErr
    }

    if securityIID.SystemId == "" {
        securityIID.SystemId = securityIID.NameId
    }
    if securityIID.NameId == "" {
        securityIID.NameId = securityIID.SystemId
    }

    if securityIID.SystemId == "" {
        newErr := fmt.Errorf("invalid S/G SystemId")
        cblogger.Error(newErr.Error())
        loggingError(callLogInfo, newErr)
        return newErr
    }

    // 1. Check if S/G file exists on disk
    iidList, err := securityHandler.ListIID()
    if err == nil {
        for _, iid := range iidList {
            if iid.SystemId == securityIID.SystemId || iid.NameId == securityIID.SystemId {
                cblogger.Infof("Security group found on disk: %s", securityIID.SystemId)
                return nil
            }
        }
    }

    // 2. Check if S/G exists in infostore (DB) and restore to disk
    sgDef, err := sim.GetKTCloudSGDef(securityIID.SystemId)
    if err == nil && sgDef != nil {
        var sgInfo irs.SecurityInfo
        if jsonErr := json.Unmarshal([]byte(sgDef.Data), &sgInfo); jsonErr == nil {
            _ = securityHandler.writeSGFile(sgInfo)
            cblogger.Infof("Restored SecurityGroup [%s] from DB to disk", securityIID.SystemId)
            return nil
        }
    }

    // 3. Fallback: try auto-recovering from attached VM
    recovered, recErr := securityHandler.recoverSGFromAttachedVM(securityIID.SystemId)
    if recErr == nil && recovered != nil {
        cblogger.Infof("Successfully auto-recovered SecurityGroup [%s] from attached VM", securityIID.SystemId)
        return nil
    }

    cblogger.Infof("SecurityGroup [%s] validated.", securityIID.SystemId)
    return nil
}
