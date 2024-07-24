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
	// icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	mockdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	_ "fmt"
	"testing"

	cblog "github.com/cloud-barista/cb-log"
)

var vpcHandler irs.VPCHandler

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
	vpcTestHandler, _ = cloudConn.CreateVPCHandler()
}

type VPCTestInfo struct {
	IId        string
	VpcCIDR    string
	SubnetIID  string
	SubnetCIDR string
}

var vpcTestInfoList = []VPCTestInfo{
	{"mock-vpc-name01", "10.0.1.0/24", "mock-subnet-01", "10.0.1.0/24"},
	{"mock-vpc-name02", "10.0.2.0/24", "mock-subnet-01", "10.0.1.0/24"},
	{"mock-vpc-name03", "10.0.3.0/24", "mock-subnet-01", "10.0.1.0/24"},
	{"mock-vpc-name04", "10.0.4.0/24", "mock-subnet-01", "10.0.1.0/24"},
	{"mock-vpc-name05", "10.0.5.0/24", "mock-subnet-01", "10.0.1.0/24"},
}

type SubnetTestInfo struct {
	VpcIID     string
	SubnetIID  string
	SubnetCIDR string
}

var subnetTestInfoList = []SubnetTestInfo{
	{"mock-vpc-name01", "mock-subnet-02", "10.0.2.0/24"},
	{"mock-vpc-name01", "mock-subnet-03", "10.0.3.0/24"},
}

func TestVPCCreateList(t *testing.T) {
	// create
	for _, info := range vpcTestInfoList {
		reqInfo := irs.VPCReqInfo{
			IId:            irs.IID{info.IId, ""},
			IPv4_CIDR:      info.VpcCIDR,
			SubnetInfoList: []irs.SubnetInfo{{IId: irs.IID{info.SubnetIID, ""}, IPv4_CIDR: info.SubnetCIDR}},
		}
		_, err := vpcTestHandler.CreateVPC(reqInfo)
		if err != nil {
			t.Error(err.Error())
		}
	}

	// check the list size and values
	infoList, err := vpcTestHandler.ListVPC()
	if err != nil {
		t.Error(err.Error())
	}
	if len(infoList) != len(vpcTestInfoList) {
		t.Errorf("The number of Infos is not %d. It is %d.", len(vpcTestInfoList), len(infoList))
	}
	for i, info := range infoList {
		if info.IId.SystemId != vpcTestInfoList[i].IId {
			t.Errorf("System ID %s is not same %s", info.IId.SystemId, vpcTestInfoList[i].IId)
		}
		//		fmt.Printf("\n\t%#v\n", info)
	}
}

func TestVPCAddSubnet(t *testing.T) {
	// add subnet
	for _, info := range subnetTestInfoList {
		subnetInfo := irs.SubnetInfo{
			IId:       irs.IID{info.SubnetIID, ""},
			IPv4_CIDR: info.SubnetCIDR,
		}
		_, err := vpcTestHandler.AddSubnet(irs.IID{info.VpcIID, ""}, subnetInfo)
		if err != nil {
			t.Error(err.Error())
		}
	}

	// check the result of two AddSubnet()
	info, err := vpcTestHandler.GetVPC(irs.IID{subnetTestInfoList[0].VpcIID, ""})
	if err != nil {
		t.Error(err.Error())
	}
	// check the number of subnets after adding
	subnetInfoList := info.SubnetInfoList
	true_num := 3
	if true_num != len(subnetInfoList) {
		t.Errorf("The number of subnetInfo is not %d. It is %d.", true_num, len(subnetInfoList))
	}
	// check the last value of subnet list
	if subnetInfoList[2].IPv4_CIDR != subnetTestInfoList[1].SubnetCIDR {
		t.Errorf("Subnet IPv4_CIDR %s is not same %s", subnetInfoList[2].IPv4_CIDR, subnetTestInfoList[1].SubnetCIDR)
	}
}

func TestVPCRemoveSubnet(t *testing.T) {
	// remove subnet
	for _, info := range subnetTestInfoList {
		_, err := vpcTestHandler.RemoveSubnet(irs.IID{info.VpcIID, ""}, irs.IID{"", info.SubnetIID})
		if err != nil {
			t.Error(err.Error())
		}
	}

	// check the result of two RemoveSubnet()
	info, err := vpcTestHandler.GetVPC(irs.IID{subnetTestInfoList[0].VpcIID, ""})
	if err != nil {
		t.Error(err.Error())
	}
	// check the number of subnets after removal
	subnetInfoList := info.SubnetInfoList
	true_num := 1
	if true_num != len(subnetInfoList) {
		t.Errorf("The number of subnetInfo is not %d. It is %d.", true_num, len(subnetInfoList))
	}
	// check the fist value of subnet list
	if subnetInfoList[0].IPv4_CIDR != vpcTestInfoList[0].SubnetCIDR {
		t.Errorf("Subnet IPv4_CIDR %s is not same %s", subnetInfoList[0].IPv4_CIDR, vpcTestInfoList[0].SubnetCIDR)
	}
}

func TestVPCDeleteGet(t *testing.T) {
	// Get & check the Value
	info, err := vpcTestHandler.GetVPC(irs.IID{vpcTestInfoList[0].IId, ""})
	if err != nil {
		t.Error(err.Error())
	}
	if info.IId.SystemId != vpcTestInfoList[0].IId {
		t.Errorf("System ID %s is not same %s", info.IId.SystemId, vpcTestInfoList[0].IId)
	}

	// delete all
	infoList, err := vpcTestHandler.ListVPC()
	if err != nil {
		t.Error(err.Error())
	}
	for _, info := range infoList {
		ret, err := vpcTestHandler.DeleteVPC(info.IId)
		if err != nil {
			t.Error(err.Error())
		}
		if !ret {
			t.Errorf("Return is not True!! %s", info.IId.NameId)
		}
	}
	// check the result of Delete Op
	infoList, err = vpcTestHandler.ListVPC()
	if err != nil {
		t.Error(err.Error())
	}
	if len(infoList) > 0 {
		t.Errorf("The number of Infos is not %d. It is %d.", 0, len(infoList))
	}
}
