// Mini Driver Test of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

package minitest

import (
	_ "fmt"
	minidrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mini"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"testing"
)

var vmSpecHandler irs.VMSpecHandler

func init() {
        cred := idrv.CredentialInfo{
                MiniName:      "MiniDriver-01",
        }
	connInfo := idrv.ConnectionInfo {
		CredentialInfo: cred, 
		RegionInfo: idrv.RegionInfo{},
	}
	cloudConn, _ := (&minidrv.MiniDriver{}).ConnectCloud(connInfo)
	vmSpecHandler, _ = cloudConn.CreateVMSpecHandler()
}


func TestVMSpecListGet(t *testing.T) {
	regionTest(t, "common-region")
}

func regionTest(t *testing.T, miniRegion string) {
        // check the list size and values
        infoList, err := vmSpecHandler.ListVMSpec(miniRegion)
        if err != nil {
                t.Error(err.Error())
        }

        if len(infoList) != 4 {
                t.Errorf("The number of Infos is not %d. It is %d.", len(infoList), 4)
        }

        for _, info := range infoList {
                if info.Region != miniRegion {
                        t.Errorf("Region Name %s is not same %s", info.Region, miniRegion)
                }
                //fmt.Printf("\n\t%#v\n", info)
        }
}
