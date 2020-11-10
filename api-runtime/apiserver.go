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

	grpcruntime "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime"
	restruntime "github.com/cloud-barista/cb-spider/api-runtime/rest-runtime"	
)

func main() {
	wg := new(sync.WaitGroup)

	wg.Add(2)

	go func() {
		restruntime.RunServer()
		wg.Done()
	}()

	go func() {
		grpcruntime.RunServer()
		wg.Done()
	}()

	wg.Wait()
}
