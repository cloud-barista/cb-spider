// Mock Driver Test of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

package mocktest

import (
	_ "fmt"
	mockdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"testing"
	cblog "github.com/cloud-barista/cb-log"
)

var vmSpecHandler irs.VMSpecHandler

func init() {
        // make the log level lower to print clearly
        cblog.SetLevel("error")

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
        }

        for _, info := range infoList {
                if info.Region != mockRegion {
                        t.Errorf("Region Name %s is not same %s", info.Region, mockRegion)
                }
                //fmt.Printf("\n\t%#v\n", info)
        }
}
