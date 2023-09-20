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

var regionZoneHandler irs.RegionZoneHandler

func init() {
	// make the log level lower to print clearly
	cblog.SetLevel("error")

	cred := idrv.CredentialInfo{
		MockName: "MockDriver-01",
	}
	connInfo := idrv.ConnectionInfo{
		CredentialInfo: cred,
		RegionInfo:     idrv.RegionInfo{Region: "neptune", Zone: "neptune-zone01"},
	}
	cloudConn, _ := (&mockdrv.MockDriver{}).ConnectCloud(connInfo)
	regionZoneHandler, _ = cloudConn.CreateRegionZoneHandler()
}

func TestRegionZoneList(t *testing.T) {
	// check the list size and values
	infoList, err := regionZoneHandler.ListRegionZone()
	if err != nil {
		t.Error(err.Error())
	}

	if len(infoList) == 0 {
		t.Errorf("The number of RegionZone is 0.")
	} else {
		jsonData, err := json.MarshalIndent(infoList, "", "  ")
		if err != nil {
			t.Error("Error while converting to JSON: ", err)
		}
		t.Logf("%s", jsonData)
	}
}

func TestGetRegionZone(t *testing.T) {
	// check the list size and values
	info, err := regionZoneHandler.GetRegionZone("uranus")
	if err != nil {
		t.Error(err.Error())
	}

	if info.Name != "uranus" {
		t.Errorf("Region Name is not uranus.")
	} else {
		jsonData, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			t.Error("Error while converting to JSON: ", err)
		}
		t.Logf("%s", jsonData)
	}
}

func TestListOrgRegion(t *testing.T) {
	orgRegionList, err := regionZoneHandler.ListOrgRegion()
	if err != nil {
		t.Error(err.Error())
	}

	if orgRegionList == "" {
		t.Errorf("The orginal Region List is empty.")
	} else {
		t.Logf("%s", orgRegionList)
	}
}

func TestListOrgZone(t *testing.T) {
	// Region should be "neptune", because this session is set to "neptune"
	orgZoneList, err := regionZoneHandler.ListOrgZone()
	if err != nil {
		t.Error(err.Error())
	}

	if orgZoneList == "" {
		t.Errorf("The original Zone List is empty.")
	} else {
		t.Logf("%s", orgZoneList)
	}
}
