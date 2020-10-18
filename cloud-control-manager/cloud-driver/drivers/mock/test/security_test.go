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
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	mockdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock"

	"testing"
	_ "fmt"
)

var securityHandler irs.SecurityHandler

func init() {
        cred := idrv.CredentialInfo{
                MockName:      "MockDriver-01",
        }
	connInfo := idrv.ConnectionInfo {
		CredentialInfo: cred, 
		RegionInfo: idrv.RegionInfo{},
	}
	cloudConn, _ := (&mockdrv.MockDriver{}).ConnectCloud(connInfo)
	securityHandler, _ = cloudConn.CreateSecurityHandler()
}

type SecurityTestInfo struct {
	IId string
	VpcIID string
}

var securityTestInfoList = []SecurityTestInfo{
	{"mock-sg-name01", "mock-vpc-name01"},
	{"mock-sg-name02", "mock-vpc-name02"},
	{"mock-sg-name03", "mock-vpc-name03"},
	{"mock-sg-name04", "mock-vpc-name04"},
	{"mock-sg-name05", "mock-vpc-name05"},
}

func TestSecurityCreateList(t *testing.T) {
	// create
	for _, info := range securityTestInfoList {
		reqInfo := irs.SecurityReqInfo {
			IId : irs.IID{info.IId, ""},
			VpcIID : irs.IID{info.VpcIID, ""},
			SecurityRules : &[]irs.SecurityRuleInfo{ {FromPort: "1", ToPort : "65535", IPProtocol : "tcp", Direction : "inbound"}, },
		}
		_, err := securityHandler.CreateSecurity(reqInfo)
		if err != nil {
			t.Error(err.Error())
		}
	}

	// check the list size and values
	infoList, err := securityHandler.ListSecurity()
	if err != nil {
		t.Error(err.Error())
	}
	if len(infoList) != len(securityTestInfoList) {
		t.Errorf("The number of Infos is not %d. It is %d.", len(securityTestInfoList), len(infoList))
	}
	for i, info := range infoList {
		if info.IId.SystemId != securityTestInfoList[i].IId {
			t.Errorf("System ID %s is not same %s", info.IId.SystemId, securityTestInfoList[i].IId)
		}
//		fmt.Printf("\n\t%#v\n", info)
	}
}

func TestSecurityDeleteGet(t *testing.T) {
        // Get & check the Value
        info, err := securityHandler.GetSecurity(irs.IID{securityTestInfoList[0].IId, ""})
        if err != nil {
                t.Error(err.Error())
        }
	if info.IId.SystemId != securityTestInfoList[0].IId {
		t.Errorf("System ID %s is not same %s", info.IId.SystemId, securityTestInfoList[0].IId)
	}

	// delete all
	infoList, err := securityHandler.ListSecurity()
        if err != nil {
                t.Error(err.Error())
        }
        for _, info := range infoList {
		ret, err := securityHandler.DeleteSecurity(info.IId)
		if err!=nil {
                        t.Error(err.Error())
		}
		if !ret {
                        t.Errorf("Return is not True!! %s", info.IId.NameId)
		}
        }
	// check the result of Delete Op
	infoList, err = securityHandler.ListSecurity()
        if err != nil {
                t.Error(err.Error())
        }
	if len(infoList)>0 {
		t.Errorf("The number of Infos is not %d. It is %d.", 0, len(infoList))
	}
}

