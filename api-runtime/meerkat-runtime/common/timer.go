// MeerKat Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.11.

package common

import (
        "time"
	cblog "github.com/cloud-barista/cb-log"
)

var timer *time.Timer
var ResetFlag bool

// called by MomKat:CheckChildKatAndSet()
func StartTimer() {
	cblogger := cblog.GetLogger("CB-SPIDER")

	cblogger.Info("Call StartTimer()")
	ResetFlag = false
	timer = time.NewTimer(TimerTime*time.Second)

	<-timer.C
}

func ResetTimer() {
	cblogger := cblog.GetLogger("CB-SPIDER")

	cblogger.Info("Call ResetTimer()")
	ResetFlag = true
	//timer.Stop()
	timer.Reset(TimerTime*time.Second)
}
