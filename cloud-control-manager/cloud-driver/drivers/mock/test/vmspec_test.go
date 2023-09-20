// Mock Driver Test of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

package mocktest

import (
	"encoding/json"
	_ "fmt"
	"testing"

	cblog "github.com/cloud-barista/cb-log"
	mockdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

var vmSpecHandler irs.VMSpecHandler

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
	vmSpecHandler, _ = cloudConn.CreateVMSpecHandler()
}

func TestVMSpecListGet(t *testing.T) {
	regionTest(t, "common-region")
}

func regionTest(t *testing.T, mockRegion string) {
	// check the list size and values
	infoList, err := vmSpecHandler.ListVMSpec()
	if err != nil {
		t.Error(err.Error())
	}

	if len(infoList) != 4 {
		t.Errorf("The number of Infos is not %d. It is %d.", len(infoList), 4)
	} else {
		jsonData, err := json.MarshalIndent(infoList, "", "  ")
		if err != nil {
			t.Error("Error while converting to JSON: ", err)
		}
		t.Logf("%s", jsonData)
	}

	for _, info := range infoList {
		if info.Region != mockRegion {
			t.Errorf("Region Name %s is not same %s", info.Region, mockRegion)
		}
	}
}

func TestListOrgVMSpec(t *testing.T) {
	orgVMSpecList, err := vmSpecHandler.ListOrgVMSpec()
	if err != nil {
		t.Error(err.Error())
	}

	if orgVMSpecList == "" {
		t.Errorf("The OrgVMSpec List is empty.")
	} else {
		t.Logf("%s", orgVMSpecList)
	}
}

func TestGetOrgVMSpec(t *testing.T) {
	orgSpec, err := vmSpecHandler.GetOrgVMSpec("mock-vmspec-02")
	if err != nil {
		t.Error(err.Error())
	}

	if orgSpec == "" {
		t.Errorf("The OrgVMSpec is empty.")
	} else {
		t.Logf("%s", orgSpec)
	}
}
