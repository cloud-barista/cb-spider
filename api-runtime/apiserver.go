// Start Runtime Servers of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.


package main

import (
	"sync"
	"time"
	"os"

	grpcruntime "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime"
	restruntime "github.com/cloud-barista/cb-spider/api-runtime/rest-runtime"	
	meerkatruntime "github.com/cloud-barista/cb-spider/api-runtime/meerkat-runtime"	
)

func main() {
	wg := new(sync.WaitGroup)

	wg.Add(2)

	go func() {
		restruntime.RunServer()
		wg.Done()
	}()

	time.Sleep(time.Millisecond*5)

	go func() {
		grpcruntime.RunServer()
		wg.Done()
	}()

	if os.Getenv("MEERKAT") == "ON" {
		time.Sleep(time.Millisecond*10)
		wg.Add(1)

		go func() {
			meerkatruntime.RunServer()
			wg.Done()
		}()
	}

	wg.Wait()
}
