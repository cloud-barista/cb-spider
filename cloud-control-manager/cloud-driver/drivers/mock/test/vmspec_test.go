// Mock Driver Test of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

package mocktest

import (
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	// icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	mockdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock"

	"testing"
	"fmt"
)

var vmSpecHandler irs.VMSpecHandler

func init() {
        cred := idrv.CredentialInfo{
                MockName:      "MockDriver-01",
        }
	connInfo := idrv.ConnectionInfo {
		CredentialInfo: cred, 
		RegionInfo: idrv.RegionInfo{},
	}
	cloudConn, _ := (&mockdrv.MockDriver{}).ConnectCloud(connInfo)
	vmSpecHandler, _ = cloudConn.CreateVMSpecHandler()
}


func TestVMSpecListGet(t *testing.T) {
	// check the list size and values
	infoList, err := vmSpecHandler.ListVMSpec("mock-region01")
	if err != nil {
		t.Error(err.Error())
	}
/*
	if len(infoList) != len(keyPairTestInfoList) {
		t.Errorf("The number of Infos is not %d. It is %d.", len(keyPairTestInfoList), len(infoList))
	}
*/
	for _, info := range infoList {
/*
		if info.IId.SystemId != keyPairTestInfoList[i].Id {
			t.Errorf("System ID %s is not same %s", info.IId.SystemId, keyPairTestInfoList[i].Id)
		}
*/
		fmt.Printf("\n\t%#v\n", info)
	}
}
