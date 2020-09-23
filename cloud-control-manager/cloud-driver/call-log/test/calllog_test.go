// Call-Log: calling logger of Cloud & VM in CB-Spider
//           Referred to cb-log
//
//      * Cloud-Barista: https://github.com/cloud-barista
//      * CB-Spider: https://github.com/cloud-barista/cb-spider
//      * cb-log: https://github.com/cloud-barista/cb-log
//
// load and set config file
//
// ref) https://github.com/go-yaml/yaml/tree/v3
//      https://godoc.org/gopkg.in/yaml.v3
//
// by CB-Spider Team, 2020.09.

package main

import (
	"github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"

	"time"
	"testing"
)

func TestCallLog(t *testing.T) {
	// logger for CB-Spider
        cblogger := cblog.GetLogger("CB-SPIDER")
        // logger for HisCall
        callogger := call.GetLogger("HISCALL")

	cblogger.Info("CB-Spider Log Info message")

	info := call.CLOUDLOGSCHEMA {
		CloudOS: call.AWS,
		RegionZone: "us-east1/us-east1-c",
		ResourceType: call.VPCSUBNET,

		ResourceName: "aws-vpc-01",
		ElapsedTime: "",

		ErrorMSG: "",
	}
	start := call.Start()
	err := ListVPC()
	info.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err.Error() + "|" + "CB-Spider Log Error message")
		info.ErrorMSG = err.Error()
	} 
	callogger.Info(call.String(info))
}

func ListVPC() error {
        //time.Sleep(time.Second*1)
        time.Sleep(time.Millisecond*10)
	return nil
}
