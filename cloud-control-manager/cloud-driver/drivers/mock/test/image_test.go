// Mock Driver Test of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

package mocktest

import (
	mockdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	_ "fmt"
	"testing"
        cblog "github.com/cloud-barista/cb-log"
)

var imageHandler irs.ImageHandler

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
	imageHandler, _ = cloudConn.CreateImageHandler()
}

const BUILTIN_IMG_NUM int = 5

var imageTestInfoList = []string{
	"mock-user-image-Name01",
	"mock-user-image-Name02",
	"mock-user-image-Name03",
	"mock-user-image-Name04",
	"mock-user-image-Name05",
}

func TestImageCreateList(t *testing.T) {
	// create
	for _, info := range imageTestInfoList {
		reqInfo := irs.ImageReqInfo{
			IId: irs.IID{info, ""},
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
	if len(infoList) != len(imageTestInfoList)+BUILTIN_IMG_NUM { // BUILTIN_IMG_NUM: built-in image (see MockDriver)
		t.Errorf("The number of Infos is not %d. It is %d.", len(imageTestInfoList), len(infoList))
	}
	for _, imgName := range imageTestInfoList {
		one, err := imageHandler.GetImage(irs.IID{imgName, imgName})
		if err != nil {
			t.Error(err.Error())
		}
		if imgName != one.IId.NameId {
			t.Errorf("Image ID %s is not same %s", imgName, one.IId.NameId)
		}
		//		fmt.Printf("\n\t%#v\n", info)
	}
}

func TestImageDeleteGet(t *testing.T) {
	// Get & check the Value
	info, err := imageHandler.GetImage(irs.IID{imageTestInfoList[0], ""})
	if err != nil {
		t.Error(err.Error())
	}
	if info.IId.SystemId != imageTestInfoList[0] {
		t.Errorf("System ID %s is not same %s", info.IId.SystemId, imageTestInfoList[0])
	}

	// delete all
	infoList, err := imageHandler.ListImage()
	if err != nil {
		t.Error(err.Error())
	}
	for _, imgName := range imageTestInfoList {
		ret, err := imageHandler.DeleteImage(irs.IID{"", imgName})
		if err != nil {
			t.Error(err.Error())
		}
		if !ret {
			t.Errorf("Return is not True!! %s", imgName)
		}
	}
	// check the result of Delete Op
	infoList, err = imageHandler.ListImage()
	if err != nil {
		t.Error(err.Error())
	}
	if len(infoList) > BUILTIN_IMG_NUM { // BUILTIN_IMG_NUM: built-in image (see MockDriver)
		t.Errorf("The number of Infos is not %d. It is %d.", 0, len(infoList))
	}
}
