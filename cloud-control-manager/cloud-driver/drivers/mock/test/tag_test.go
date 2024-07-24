// Mock Driver Test of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2024.07.

package mocktest

import (
	mockdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	_ "fmt"
	"testing"

	cblog "github.com/cloud-barista/cb-log"
)

var tagHandler irs.TagHandler
var vpcTestHandler irs.VPCHandler
var vmTestHandler irs.VMHandler
var securityTestHandler irs.SecurityHandler
var keyPairTestHandler irs.KeyPairHandler

func init() {
	// make the log level lower to print clearly
	cblog.SetLevel("error")

	cred := idrv.CredentialInfo{
		MockName: "MockDriver-01",
	}
	connInfo := idrv.ConnectionInfo{
		CredentialInfo: cred,
		RegionInfo:     idrv.RegionInfo{},
	}
	cloudConn, _ := (&mockdrv.MockDriver{}).ConnectCloud(connInfo)
	tagHandler, _ = cloudConn.CreateTagHandler()
	vpcTestHandler, _ = cloudConn.CreateVPCHandler()
	vmTestHandler, _ = cloudConn.CreateVMHandler()
	securityTestHandler, _ = cloudConn.CreateSecurityHandler()
	keyPairTestHandler, _ = cloudConn.CreateKeyPairHandler()
}

type TagTestInfo struct {
	ResType irs.RSType
	ResID   string
	Key     string
	Value   string
}

var tagTestInfoList = []TagTestInfo{
	{ResType: irs.VM, ResID: "mock-vm-01", Key: "Environment", Value: "Dev"},
	{ResType: irs.VM, ResID: "mock-vm-02", Key: "Environment", Value: "Test"},
	{ResType: irs.VM, ResID: "mock-vm-03", Key: "Environment", Value: "Prod"},
	{ResType: irs.VM, ResID: "mock-vm-04", Key: "Owner", Value: "TeamA"},
	{ResType: irs.SUBNET, ResID: "mock-subnet-01", Key: "Environment", Value: "Dev"},
	{ResType: irs.SUBNET, ResID: "mock-subnet-02", Key: "Environment", Value: "Test"},
	{ResType: irs.VPC, ResID: "mock-vpc-01", Key: "Department", Value: "Finance"},
	{ResType: irs.SG, ResID: "mock-sg-01", Key: "Environment", Value: "Production"},
	{ResType: irs.KEY, ResID: "mock-keypair-01", Key: "Owner", Value: "Admin"},
}

type ResourceInfo struct {
	IId               string
	ImageIID          string
	VpcIID            string
	SubnetIID         string
	SecurityGroupIIDs []string
	VMSpecName        string
	KeyPairIID        string
}

var resourceInfoList = []ResourceInfo{
	{"mock-vm-01", "mock-img-01", "mock-vpc-01", "mock-subnet-01", []string{"mock-sg-01"}, "mock-vmspec-01", "mock-keypair-01"},
	{"mock-vm-02", "mock-img-01", "mock-vpc-01", "mock-subnet-01", []string{"mock-sg-01"}, "mock-vmspec-01", "mock-keypair-01"},
	{"mock-vm-03", "mock-img-01", "mock-vpc-01", "mock-subnet-01", []string{"mock-sg-01"}, "mock-vmspec-01", "mock-keypair-01"},
	{"mock-vm-04", "mock-img-01", "mock-vpc-01", "mock-subnet-01", []string{"mock-sg-01"}, "mock-vmspec-01", "mock-keypair-01"},
}

func TestSetup(t *testing.T) {
	// Create VPC
	vpcReqInfo := irs.VPCReqInfo{
		IId:       irs.IID{NameId: "mock-vpc-01"},
		IPv4_CIDR: "192.168.0.0/16",
		SubnetInfoList: []irs.SubnetInfo{
			{IId: irs.IID{NameId: "mock-subnet-01"}, IPv4_CIDR: "192.168.1.0/24"},
			{IId: irs.IID{NameId: "mock-subnet-02"}, IPv4_CIDR: "192.168.2.0/24"},
		},
	}
	_, err := vpcTestHandler.CreateVPC(vpcReqInfo)
	if err != nil {
		t.Error(err.Error())
	}

	// Create Security Group
	sgReqInfo := irs.SecurityReqInfo{
		IId:           irs.IID{NameId: "mock-sg-01"},
		VpcIID:        irs.IID{NameId: "mock-vpc-01"},
		SecurityRules: &[]irs.SecurityRuleInfo{{FromPort: "1", ToPort: "65535", IPProtocol: "tcp", Direction: "inbound"}},
	}
	_, err = securityTestHandler.CreateSecurity(sgReqInfo)
	if err != nil {
		t.Error(err.Error())
	}

	// Create KeyPair
	keypairReqInfo := irs.KeyPairReqInfo{
		IId: irs.IID{NameId: "mock-keypair-01"},
	}
	_, err = keyPairTestHandler.CreateKey(keypairReqInfo)
	if err != nil {
		t.Error(err.Error())
	}

	// Create VM
	for _, info := range resourceInfoList {
		sgIIDs := []irs.IID{}
		for _, sgIId := range info.SecurityGroupIIDs {
			sgIIDs = append(sgIIDs, irs.IID{sgIId, ""})
		}

		vmReqInfo := irs.VMReqInfo{
			IId: irs.IID{info.IId, ""},

			ImageIID:          irs.IID{info.ImageIID, ""},
			VpcIID:            irs.IID{info.VpcIID, ""},
			SubnetIID:         irs.IID{info.SubnetIID, ""},
			SecurityGroupIIDs: sgIIDs,

			VMSpecName: info.VMSpecName,
			KeyPairIID: irs.IID{info.KeyPairIID, ""},

			VMUserId:     "user01",
			VMUserPasswd: "pass01",
		}
		_, err := vmTestHandler.StartVM(vmReqInfo)
		if err != nil {
			t.Error(err.Error())
		}
	}

	// Check VM List
	infoList, err := vmTestHandler.ListVM()
	if err != nil {
		t.Error(err.Error())
	}
	if len(infoList) != len(resourceInfoList) {
		t.Errorf("The number of Infos is not %d. It is %d.", len(resourceInfoList), len(infoList))
	}
	for i, info := range infoList {
		if info.IId.SystemId != resourceInfoList[i].IId {
			t.Errorf("System ID %s is not same %s", info.IId.SystemId, resourceInfoList[i].IId)
		}
	}
}

func TestTagCreateList(t *testing.T) {
	// Create tags
	for _, info := range tagTestInfoList {
		tag := irs.KeyValue{Key: info.Key, Value: info.Value}
		_, err := tagHandler.AddTag(info.ResType, irs.IID{NameId: info.ResID}, tag)
		if err != nil {
			t.Error(err.Error())
		}
	}

	// Check the list size and values
	for _, info := range tagTestInfoList {
		tagList, err := tagHandler.ListTag(info.ResType, irs.IID{NameId: info.ResID})
		if err != nil {
			t.Error(err.Error())
		}
		found := false
		for _, tag := range tagList {
			if tag.Key == info.Key && tag.Value == info.Value {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Tag %s=%s not found for %s %s", info.Key, info.Value, info.ResType, info.ResID)
		}
	}
}

func TestTagDeleteGet(t *testing.T) {
	// Get & check the Value
	info := tagTestInfoList[0]
	tag, err := tagHandler.GetTag(info.ResType, irs.IID{NameId: info.ResID}, info.Key)
	if err != nil {
		t.Error(err.Error())
	}
	if tag.Key != info.Key || tag.Value != info.Value {
		t.Errorf("Tag %s=%s not found for %s %s", info.Key, info.Value, info.ResType, info.ResID)
	}

	// Delete all
	for _, info := range tagTestInfoList {
		_, err := tagHandler.RemoveTag(info.ResType, irs.IID{NameId: info.ResID}, info.Key)
		if err != nil {
			t.Error(err.Error())
		}
	}

	// Check the result of Delete Op
	for _, info := range tagTestInfoList {
		tagList, err := tagHandler.ListTag(info.ResType, irs.IID{NameId: info.ResID})
		if err != nil {
			t.Error(err.Error())
		}
		if len(tagList) > 0 {
			t.Errorf("Tags are not deleted for %s %s", info.ResType, info.ResID)
		}
	}
}

func TestCleanup(t *testing.T) {
	// Terminate VMs
	infoList, err := vmTestHandler.ListVM()
	if err != nil {
		t.Error(err.Error())
	}
	for _, info := range infoList {
		_, err := vmTestHandler.TerminateVM(info.IId)
		if err != nil {
			t.Error(err.Error())
		}
	}

	// Delete Security Groups
	sgList, err := securityTestHandler.ListSecurity()
	if err != nil {
		t.Error(err.Error())
	}
	for _, info := range sgList {
		_, err := securityTestHandler.DeleteSecurity(info.IId)
		if err != nil {
			t.Error(err.Error())
		}
	}

	// Delete KeyPairs
	keyList, err := keyPairTestHandler.ListKey()
	if err != nil {
		t.Error(err.Error())
	}
	for _, info := range keyList {
		_, err := keyPairTestHandler.DeleteKey(info.IId)
		if err != nil {
			t.Error(err.Error())
		}
	}

	// Delete VPCs
	vpcInfoList, err := vpcTestHandler.ListVPC()
	if err != nil {
		t.Error(err.Error())
	}
	for _, info := range vpcInfoList {
		_, err := vpcTestHandler.DeleteVPC(info.IId)
		if err != nil {
			t.Error(err.Error())
		}
	}
}
