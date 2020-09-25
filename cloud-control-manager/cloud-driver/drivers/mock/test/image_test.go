// Mock Driver Test of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

package main

import (
	"github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"

	//idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	//mockdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock"
	mkcon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock/connect"

	"testing"
)

// logger for CB-Spider
var cblogger *logrus.Logger
// logger for HisCall
var callogger *logrus.Logger
var cloudConn icon.CloudConnection
var imageHandler irs.ImageHandler

func init() {
        cblogger = cblog.GetLogger("CB-SPIDER")
        callogger = call.GetLogger("HISCALL")

        cloudConn := mkcon.MockConnection{
                MockName:      "MockDriver-01",
        }
	//cloudConn = mockdrv.MockDriver.ConnectCloud()
	imageHandler, _ = cloudConn.CreateImageHandler()
}
/*
func getCloudDriver() idrv.CloudDriver {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(mocdrv.MockDriver)
	return cloudDriver
}
*/

func TestCreate(t *testing.T) {
	cblogger.Info("Create Test")

	imageReqInfo := irs.ImageReqInfo {
		IId : irs.IID{"mock-image-01", "mock-image-01"},
	}

	info := call.CLOUDLOGSCHEMA {
		CloudOS: call.MOCK,
		RegionZone: "no Region",
		ResourceType: call.VMIMAGE,
		ResourceName: "mock-image-01",
		CloudOSAPI: "CreateImage()",
		ElapsedTime: "",
		ErrorMSG: "",
	}
	start := call.Start()
	//imageInfo, err := imageHandler.CreateImage(imageReqInfo)
	_, err := imageHandler.CreateImage(imageReqInfo)
	info.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err.Error() + "|" + "CB-Spider Log Error message")
		info.ErrorMSG = err.Error()
	} 
	callogger.Info(call.String(info))


	imageInfoList, err := imageHandler.ListImage()
	for _, info := range imageInfoList {
		cblogger.Info(info)
	}
}

