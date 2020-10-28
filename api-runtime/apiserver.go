package main

import (
	"sync"

	// cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
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