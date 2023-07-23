// Start Runtime Servers of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

package main

import (
	"runtime"
	"fmt"
	"sync"
	"time"

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	grpcruntime "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime"
	restruntime "github.com/cloud-barista/cb-spider/api-runtime/rest-runtime"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
)

func main() {
	// use multi-Core
        runtime.GOMAXPROCS(runtime.NumCPU())

	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("cb-spider terminated with error: %v\n", err)
	}
}

func NewRootCmd() *cobra.Command {

	rootCmd := &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {

			wg := new(sync.WaitGroup)

			wg.Add(2)

			go func() {
				restruntime.RunServer()
				wg.Done()
			}()

			time.Sleep(time.Millisecond * 5)

			go func() {
				grpcruntime.RunServer()
				wg.Done()
			}()

			wg.Wait()

		},
	}

	rootCmd.AddCommand(NewInfoCmd())

	return rootCmd
}

func NewInfoCmd() *cobra.Command {

	infoCmd := &cobra.Command{
		Use: "info",
		Run: func(cmd *cobra.Command, args []string) {
			client := resty.New()
			resp, err := client.R().Get("http://" + cr.ServiceIPorName + cr.ServicePort + "/spider/endpointinfo")
			if err != nil {
				fmt.Printf("%v\n", err)
			} else {
				fmt.Printf("%v\n", resp)
			}
		},
	}

	return infoCmd
}
