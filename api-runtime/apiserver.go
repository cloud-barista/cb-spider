package main

import (
	"sync"

	// cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	grpcruntime "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime"
	restruntime "github.com/cloud-barista/cb-spider/api-runtime/rest-runtime"
	_ "time"
	_ "fmt"
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

//	printAdminWebURL()
	wg.Wait()
}
/*
func printAdminWebURL() {
	for count:=0; count < 3; count++{
		if cr.ServicePort
		time.Sleep(100 * time.Millisecond)
	}
	adminWebURL := "http://" + cr.HostIPorName + cr.ServicePort + "/spider/adminweb"
	fmt.Printf("\n   AdminWeb: %s\n\n", adminWebURL)
}
*/
