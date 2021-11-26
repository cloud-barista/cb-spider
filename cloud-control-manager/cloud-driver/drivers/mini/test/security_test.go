// Mini Driver Test of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.10.

package minitest

import (
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	// icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	minidrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mini"

	"testing"
	_ "fmt"
)

var securityHandler irs.SecurityHandler

func init() {
        cred := idrv.CredentialInfo{
                MiniName:      "MiniDriver-01",
        }
	connInfo := idrv.ConnectionInfo {
		CredentialInfo: cred, 
		RegionInfo: idrv.RegionInfo{},
	}
	cloudConn, _ := (&minidrv.MiniDriver{}).ConnectCloud(connInfo)
	securityHandler, _ = cloudConn.CreateSecurityHandler()
}

type SecurityTestInfo struct {
	IId string
	VpcIID string
}

var securityTestInfoList = []SecurityTestInfo{
	{"mini-sg-name01", "mini-vpc-name01"},
	{"mini-sg-name02", "mini-vpc-name02"},
	{"mini-sg-name03", "mini-vpc-name03"},
	{"mini-sg-name04", "mini-vpc-name04"},
	{"mini-sg-name05", "mini-vpc-name05"},
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

