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
	_ "fmt"
	cblog "github.com/cloud-barista/cb-log"
)

var keyPairHandler irs.KeyPairHandler

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
	keyPairHandler, _ = cloudConn.CreateKeyPairHandler()
}

type KeyPairTestInfo struct {
	Id string
}

var keyPairTestInfoList = []KeyPairTestInfo{
	{"mock-key-Name01"},
	{"mock-key-Name02"},
	{"mock-key-Name03"},
	{"mock-key-Name04"},
	{"mock-key-Name05"},
}

func TestKeyPairCreateList(t *testing.T) {
	// create
	for _, info := range keyPairTestInfoList {
		reqInfo := irs.KeyPairReqInfo {
			IId : irs.IID{info.Id, ""},
		}
		_, err := keyPairHandler.CreateKey(reqInfo)
		if err != nil {
			t.Error(err.Error())
		}
	}

	// check the list size and values
	infoList, err := keyPairHandler.ListKey()
	if err != nil {
		t.Error(err.Error())
	}
	if len(infoList) != len(keyPairTestInfoList) {
		t.Errorf("The number of Infos is not %d. It is %d.", len(keyPairTestInfoList), len(infoList))
	}
	for i, info := range infoList {
		if info.IId.SystemId != keyPairTestInfoList[i].Id {
			t.Errorf("System ID %s is not same %s", info.IId.SystemId, keyPairTestInfoList[i].Id)
		}
//		fmt.Printf("\n\t%#v\n", info)
	}
}

func TestKeyPairDeleteGet(t *testing.T) {
        // Get & check the Value
        info, err := keyPairHandler.GetKey(irs.IID{keyPairTestInfoList[0].Id, ""})
        if err != nil {
                t.Error(err.Error())
        }
	if info.IId.SystemId != keyPairTestInfoList[0].Id {
		t.Errorf("System ID %s is not same %s", info.IId.SystemId, keyPairTestInfoList[0].Id)
	}

	// delete all
	infoList, err := keyPairHandler.ListKey()
        if err != nil {
                t.Error(err.Error())
        }
        for _, info := range infoList {
		ret, err := keyPairHandler.DeleteKey(info.IId)
		if err!=nil {
                        t.Error(err.Error())
		}
		if !ret {
                        t.Errorf("Return is not True!! %s", info.IId.NameId)
		}
        }
	// check the result of Delete Op
	infoList, err = keyPairHandler.ListKey()
        if err != nil {
                t.Error(err.Error())
        }
	if len(infoList)>0 {
		t.Errorf("The number of Infos is not %d. It is %d.", 0, len(infoList))
	}
}

