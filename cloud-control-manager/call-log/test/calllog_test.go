package main

import (
	"github.com/cloud-barista/cb-spider/cloud-control-manager/call-log"

	"time"
	"testing"
)

func init() {
        // calllog is a global variable.
        calllog.GetLogger("HISCALL")
}


/*
func TestCallLog(t *testing.T) {

		calllogger.Info("start.........")

		err := createUser("newUser")
		calllogger.Info("msg for debugging msg!!")
		if err != nil {
			t.Error(err)
		}

		calllogger.Info("end.........")

		time.Sleep(time.Second*1)
		fmt.Print("\n")
	}
}
*/


func TestCallLog(t *testing.T) {

	info := calllog.CLOUDLOGSCHEMA {
		CSPName: "AWS",
		RegionZone: "us-east1/us-east1-c",
		ResourceType: "VPC/SUBNET",

		ResourceName: "aws-vpc-01",
		ElapsedTime: "",

		ErrorNumber: "",
		ErrorMSG: "",
	}

start := time.Now()
	err := CallFunc()
info.ElapsedTime = time.Since(start).String()
	if err != nil {
		calllog.Error(info, err.Error())
	} 
	calllog.Info(info)
}

func CallFunc() error {
        time.Sleep(time.Second*1)
	return nil
}
