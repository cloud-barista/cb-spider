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
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	mockdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock"

	"testing"
	_ "fmt"
)

var imageHandler irs.ImageHandler

func init() {
        cred := idrv.CredentialInfo{
                MockName:      "MockDriver-01",
        }
	connInfo := idrv.ConnectionInfo {
		CredentialInfo: cred, 
		RegionInfo: idrv.RegionInfo{},
	}
	cloudConn, _ := (&mockdrv.MockDriver{}).ConnectCloud(connInfo)
	imageHandler, _ = cloudConn.CreateImageHandler()
}

type ImageTestInfo struct {
	ImageId string
}

var imageTestInfoList = []ImageTestInfo{
	{"mock-image-Name01"},
	{"mock-image-Name02"},
	{"mock-image-Name03"},
	{"mock-image-Name04"},
	{"mock-image-Name05"},
}

func TestImageCreateList(t *testing.T) {
	// create
	for _, info := range imageTestInfoList {
		reqInfo := irs.ImageReqInfo {
			IId : irs.IID{info.ImageId, ""},
		}
		_, err := imageHandler.CreateImage(reqInfo)
		if err != nil {
			t.Error(err.Error())
		}
	}

	// check the list size and values
	infoList, err := imageHandler.ListImage()
	if err != nil {
		t.Error(err.Error())
	}
	if len(infoList) != len(imageTestInfoList) {
		t.Errorf("The number of Infos is not %d. It is %d.", len(imageTestInfoList), len(infoList))
	}
	for i, info := range infoList {
		if info.IId.SystemId != imageTestInfoList[i].ImageId {
			t.Errorf("System ID %s is not same %s", info.IId.SystemId, imageTestInfoList[i].ImageId)
		}
//		fmt.Printf("\n\t%#v\n", info)
	}
}

func TestImageDeleteGet(t *testing.T) {
        // Get & check the Value
        info, err := imageHandler.GetImage(irs.IID{imageTestInfoList[0].ImageId, ""})
        if err != nil {
                t.Error(err.Error())
        }
	if info.IId.SystemId != imageTestInfoList[0].ImageId {
		t.Errorf("System ID %s is not same %s", info.IId.SystemId, imageTestInfoList[0].ImageId)
	}

	// delete all
	infoList, err := imageHandler.ListImage()
        if err != nil {
                t.Error(err.Error())
        }
        for _, info := range infoList {
		ret, err := imageHandler.DeleteImage(info.IId)
		if err!=nil {
                        t.Error(err.Error())
		}
		if !ret {
                        t.Errorf("Return is not True!! %s", info.IId.NameId)
		}
        }
	// check the result of Delete Op
	infoList, err = imageHandler.ListImage()
        if err != nil {
                t.Error(err.Error())
        }
	if len(infoList)>0 {
		t.Errorf("The number of Infos is not %d. It is %d.", 0, len(infoList))
	}
}
