// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2021.10.

package commonruntime

import (
	"time"
)

//============================================
type WAITER struct {
        start 	 time.Time
	Sleep 	 int  // sec, default = 1
	Timeout  int  // sec, default = 120
	
}
//============================================

func NewWaiter(sleep int, timeout int) *WAITER {
	var waiter = new(WAITER)
	waiter.start = time.Now()
	waiter.Sleep = 1
	waiter.Timeout = 120

	if sleep > 1 {
		waiter.Sleep = sleep
	}

	if timeout > waiter.Sleep {
		waiter.Timeout = timeout
	}

	return waiter
}

func (waiter *WAITER)Wait() bool {
	elapsed := time.Since(waiter.start)

	if int(elapsed.Seconds()) < waiter.Timeout {
		time.Sleep(time.Duration(waiter.Sleep) * time.Second)
		return true // more waiting
	}
	return false // stop waiting
}
