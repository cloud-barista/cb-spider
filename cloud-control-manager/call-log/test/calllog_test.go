package main

import (
	"github.com/cloud-barista/cb-spider/cloud-control-manager/call-log"

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
		CSPName: "cspname-value",
		RegionZone: "region/zone-value",
		ResourceName: "resourceName-value",
	}

        calllog.Info(info)
}

