// Mock Driver Test of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

package main

import (
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	mockdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock"

	"testing"
	 "fmt"
)

var cloudConn icon.CloudConnection
var imageHandler irs.ImageHandler

func init() {
        cred := idrv.CredentialInfo{
                MockName:      "MockDriver-01",
        }
	connInfo := idrv.ConnectionInfo {
		CredentialInfo: cred, 
		RegionInfo: idrv.RegionInfo{},
	}
	cloudConn, _ = (&mockdrv.MockDriver{}).ConnectCloud(connInfo)
	imageHandler, _ = cloudConn.CreateImageHandler()
}

type TestInfo struct {
	ImageId string
}

var testInfoList = []TestInfo{
	{"mock-image-Name01"},
	{"mock-image-Name02"},
	{"mock-image-Name03"},
	{"mock-image-Name04"},
	{"mock-image-Name05"},
}

func TestCreateList(t *testing.T) {
	// create
	for _, info := range testInfoList {
		imageReqInfo := irs.ImageReqInfo {
			IId : irs.IID{info.ImageId, ""},
		}
		_, err := imageHandler.CreateImage(imageReqInfo)
		if err != nil {
			t.Error(err.Error())
		}
	}

	// check the list size and values
	imageInfoList, err := imageHandler.ListImage()
	if err != nil {
		t.Error(err.Error())
	}
	if len(imageInfoList) != len(testInfoList) {
		t.Errorf("The number of Images is not %d. It is %d.", len(testInfoList), len(imageInfoList))
	}
	for i, info := range imageInfoList {
		if info.IId.SystemId != testInfoList[i].ImageId {
			t.Errorf("Image System ID %s is not same %s", info.IId.SystemId, testInfoList[i].ImageId)
		}
		fmt.Printf("\n\t%#v\n", info)
	}
}
/*
func TestDeleteGet(t *testing.T) {
        // Get & check the Value
        imageInfo, err := imageHandler.GetImage(irs.IID{testInfoList[0].ImageId, ""})
        if err != nil {
                t.Error(err.Error())
        }
	if imageInfo.IId.SystemId != testInfoList[0].ImageId {
		t.Errorf("Image System ID %s is not same %s", imageInfo.IId.SystemId, testInfoList[0].ImageId)
	}

	// delete all
	imageInfoList, err := imageHandler.ListImage()
        if err != nil {
                t.Error(err.Error())
        }
        for _, info := range imageInfoList {
		ret, err := imageHandler.DeleteImage(info.IId)
		if err!=nil {
                        t.Error(err.Error())
		}
		if !ret {
                        t.Errorf("Return is not True!! %s", info.IId.NameId)
		}
        }
	// check the result of Delete Op
	imageInfoList, err = imageHandler.ListImage()
        if err != nil {
                t.Error(err.Error())
        }
	if len(imageInfoList)>0 {
		t.Errorf("The number of Images is not %d. It is %d.", 0, len(imageInfoList))
	}
}
*/
