// Mock Driver Test of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.10.

package mocktest

import (
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	mockdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"testing"
	cblog "github.com/cloud-barista/cb-log"
)

var vmHandler irs.VMHandler

func init() {
        // make the log level lower to print clearly
        cblog.SetLevel("error")

	cred := idrv.CredentialInfo{
		MockName: "MockDriver-77", // *** 주의 *** : 다른 테스트의 데이터와 충돌 방지를 위해 별도 이름 지정
	}
	connInfo := idrv.ConnectionInfo{
		CredentialInfo: cred,
		RegionInfo:     idrv.RegionInfo{"default", "", ""},
	}
	cloudConn, _ := (&mockdrv.MockDriver{}).ConnectCloud(connInfo)
	vmHandler, _ = cloudConn.CreateVMHandler()

	imageHandler, _ := cloudConn.CreateImageHandler()
	vmSpecHandler, _ := cloudConn.CreateVMSpecHandler()
	vpcHandler, _ := cloudConn.CreateVPCHandler()
	securityHandler, _ := cloudConn.CreateSecurityHandler()
	keyPairHandler, _ := cloudConn.CreateKeyPairHandler()

	// image creation
	for _, info := range imgTestInfoList {
		imgReqInfo := irs.ImageReqInfo{
			IId: irs.IID{info.ImageIID, ""},
		}
		imageHandler.CreateImage(imgReqInfo)
	}

	// spec creation
	vmSpecHandler.ListVMSpec()

	// vpc creation
	for _, info := range vpcSubnetTestInfoList {
		vpcReqInfo := irs.VPCReqInfo{
			IId:            irs.IID{info.VpcIID, ""},
			IPv4_CIDR:      "10.0.1.0/24",
			SubnetInfoList: []irs.SubnetInfo{{IId: irs.IID{info.SubnetIID, ""}, IPv4_CIDR: "10.0.1.0/24"}},
		}
		vpcHandler.CreateVPC(vpcReqInfo)
	}

	// sg creation
	for _, info := range sgTestInfoList {
		for _, sgIId := range info.SecurityGroupIIDs {
			sgReqInfo := irs.SecurityReqInfo{
				IId:           irs.IID{sgIId, ""},
				VpcIID:        irs.IID{info.VpcIID, ""},
				SecurityRules: &[]irs.SecurityRuleInfo{{FromPort: "1", ToPort: "65535", IPProtocol: "tcp", Direction: "inbound"}},
			}
			securityHandler.CreateSecurity(sgReqInfo)
		}
	}

	// keypair creation
	for _, info := range keypairTestInfoList {
		keypairReqInfo := irs.KeyPairReqInfo{
			IId: irs.IID{info.KeyPairIID, ""},
		}
		keyPairHandler.CreateKey(keypairReqInfo)
	}
}

type VMTestInfo struct {
	IId               string
	ImageIID          string
	VpcIID            string
	SubnetIID         string
	SecurityGroupIIDs []string
	VMSpecName        string
	KeyPairIID        string
}

var imgTestInfoList = []VMTestInfo{
	{ImageIID: "mock-img-01"},
	{ImageIID: "mock-img-02"},
	{ImageIID: "mock-img-03"},
}

var vpcSubnetTestInfoList = []VMTestInfo{
	{VpcIID: "mock-vpc-01", SubnetIID: "mock-subnet-11"},
	{VpcIID: "mock-vpc-02", SubnetIID: "mock-subnet-21"},
}

var sgTestInfoList = []VMTestInfo{
	{VpcIID: "mock-vpc-01", SecurityGroupIIDs: []string{"mock-sg-11", "mock-sg-12"}},
	{VpcIID: "mock-vpc-02", SecurityGroupIIDs: []string{"mock-sg-21"}},
}

var keypairTestInfoList = []VMTestInfo{
	{KeyPairIID: "mock-keypair-01"},
	{KeyPairIID: "mock-keypair-02"},
	{KeyPairIID: "mock-keypair-03"},
}

var vmTestInfoList = []VMTestInfo{
	{"mock-vm-01", "mock-img-01", "mock-vpc-01", "mock-subnet-11", []string{"mock-sg-11"}, "mock-vmspec-01", "mock-keypair-01"},
	{"mock-vm-02", "mock-img-01", "mock-vpc-01", "mock-subnet-11", []string{"mock-sg-11"}, "mock-vmspec-01", "mock-keypair-01"},
	{"mock-vm-03", "mock-img-01", "mock-vpc-01", "mock-subnet-11", []string{"mock-sg-11"}, "mock-vmspec-01", "mock-keypair-01"},
	{"mock-vm-04", "mock-img-01", "mock-vpc-01", "mock-subnet-11", []string{"mock-sg-11"}, "mock-vmspec-01", "mock-keypair-01"},
	{"mock-vm-05", "mock-img-01", "mock-vpc-02", "mock-subnet-21", []string{"mock-sg-21"}, "mock-vmspec-01", "mock-keypair-01"},
}

func TestStartVMList(t *testing.T) {

	for _, info := range vmTestInfoList {

		sgIIDs := []irs.IID{}
		for _, sgIId := range info.SecurityGroupIIDs {
			sgIIDs = append(sgIIDs, irs.IID{sgIId, ""})
		}

		// vm creation
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
		_, err := vmHandler.StartVM(vmReqInfo)
		if err != nil {
			t.Error(err.Error())
		}
	}

	// check the list size and values
	infoList, err := vmHandler.ListVM()
	if err != nil {
		t.Error(err.Error())
	}
	if len(infoList) != len(vmTestInfoList) {
		t.Errorf("The number of Infos is not %d. It is %d.", len(vmTestInfoList), len(infoList))
	}
	for i, info := range infoList {
		if info.IId.SystemId != vmTestInfoList[i].IId {
			t.Errorf("System ID %s is not same %s", info.IId.SystemId, vmTestInfoList[i].IId)
		}
		//fmt.Printf("\n\t%#v\n", info)
	}

}

func TestVMSuspendGet(t *testing.T) {

	// Get & check the Value
	info, err := vmHandler.GetVM(irs.IID{vmTestInfoList[0].IId, ""})
	if err != nil {
		t.Error(err.Error())
	}
	if info.IId.SystemId != vmTestInfoList[0].IId {
		t.Errorf("System ID %s is not same %s", info.IId.SystemId, vmTestInfoList[0].IId)
	}

	// suspend all
	infoList, err := vmHandler.ListVM()
	if err != nil {
		t.Error(err.Error())
	}
	for _, info := range infoList {
		ret, err := vmHandler.SuspendVM(info.IId)
		if err != nil {
			t.Error(err.Error())
		}
		if ret != "Suspending" {
			t.Errorf("Return is not Suspending!! %s", info.IId.NameId)
		}

		ret, err = vmHandler.GetVMStatus(info.IId)
		if err != nil {
			t.Error(err.Error())
		}
		if ret != "Suspended" {
			t.Errorf("Return is not Suspended!! %s", info.IId.NameId)
		}
	}

	// check the result of Suspend Op
	statusList, err := vmHandler.ListVMStatus()
	if err != nil {
		t.Error(err.Error())
	}
	if len(infoList) != len(statusList) {
		t.Errorf("The number of Infos is not %d. It is %d.", len(statusList), len(infoList))
	}
	for _, info := range statusList {
		if info.VmStatus != "Suspended" {
			t.Errorf("Return is not Suspended!! %s", info.IId.NameId)
		}
	}

}

func TestVMResumeGet(t *testing.T) {

	// Get & check the Value
	info, err := vmHandler.GetVM(irs.IID{vmTestInfoList[0].IId, ""})
	if err != nil {
		t.Error(err.Error())
	}
	if info.IId.SystemId != vmTestInfoList[0].IId {
		t.Errorf("System ID %s is not same %s", info.IId.SystemId, vmTestInfoList[0].IId)
	}

	// resume all
	infoList, err := vmHandler.ListVM()
	if err != nil {
		t.Error(err.Error())
	}
	for _, info := range infoList {
		ret, err := vmHandler.ResumeVM(info.IId)
		if err != nil {
			t.Error(err.Error())
		}
		if ret != "Resuming" {
			t.Errorf("Return is not Resuming!! %s", info.IId.NameId)
		}

		ret, err = vmHandler.GetVMStatus(info.IId)
		if err != nil {
			t.Error(err.Error())
		}
		if ret != "Running" {
			t.Errorf("Return is not Running!! %s", info.IId.NameId)
		}
	}

	// check the result of Resume Op
	statusList, err := vmHandler.ListVMStatus()
	if err != nil {
		t.Error(err.Error())
	}
	if len(infoList) != len(statusList) {
		t.Errorf("The number of Infos is not %d. It is %d.", len(statusList), len(infoList))
	}
	for _, info := range statusList {
		if info.VmStatus != "Running" {
			t.Errorf("Return is not Running!! %s", info.IId.NameId)
		}
	}

}

func TestVMRebootGet(t *testing.T) {

	// Get & check the Value
	info, err := vmHandler.GetVM(irs.IID{vmTestInfoList[0].IId, ""})
	if err != nil {
		t.Error(err.Error())
	}
	if info.IId.SystemId != vmTestInfoList[0].IId {
		t.Errorf("System ID %s is not same %s", info.IId.SystemId, vmTestInfoList[0].IId)
	}

	// reboot all
	infoList, err := vmHandler.ListVM()
	if err != nil {
		t.Error(err.Error())
	}
	for _, info := range infoList {
		ret, err := vmHandler.RebootVM(info.IId)
		if err != nil {
			t.Error(err.Error())
		}
		if ret != "Rebooting" {
			t.Errorf("Return is not Rebooting!! %s", info.IId.NameId)
		}

		ret, err = vmHandler.GetVMStatus(info.IId)
		if err != nil {
			t.Error(err.Error())
		}
		if ret != "Running" {
			t.Errorf("Return is not Running!! %s", info.IId.NameId)
		}
	}

	// check the result of Reboot Op
	statusList, err := vmHandler.ListVMStatus()
	if err != nil {
		t.Error(err.Error())
	}
	if len(infoList) != len(statusList) {
		t.Errorf("The number of Infos is not %d. It is %d.", len(statusList), len(infoList))
	}
	for _, info := range statusList {
		if info.VmStatus != "Running" {
			t.Errorf("Return is not Running!! %s", info.IId.NameId)
		}
	}

}

func TestVMTerminateGet(t *testing.T) {

	// Get & check the Value
	info, err := vmHandler.GetVM(irs.IID{vmTestInfoList[0].IId, ""})
	if err != nil {
		t.Error(err.Error())
	}
	if info.IId.SystemId != vmTestInfoList[0].IId {
		t.Errorf("System ID %s is not same %s", info.IId.SystemId, vmTestInfoList[0].IId)
	}

	// terminate all
	infoList, err := vmHandler.ListVM()
	if err != nil {
		t.Error(err.Error())
	}
	for _, info := range infoList {
		ret, err := vmHandler.TerminateVM(info.IId)
		if err != nil {
			t.Error(err.Error())
		}
		if ret != "Terminating" {
			t.Errorf("Return is not Terminating!! %s", info.IId.NameId)
		}
	}
	// check the result of Terminate Op
	infoList, err = vmHandler.ListVM()
	if err != nil {
		t.Error(err.Error())
	}
	if len(infoList) > 0 {
		t.Errorf("The number of Infos is not %d. It is %d.", 0, len(infoList))
	}

}
