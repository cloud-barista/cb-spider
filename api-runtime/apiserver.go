// Start Runtime Servers of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"sync"

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	restruntime "github.com/cloud-barista/cb-spider/api-runtime/rest-runtime"
	"github.com/spf13/cobra"
)

var (
	Version   string // Populated by ldflags
	CommitSHA string // Populated by ldflags
	BuildTime string // Populated by ldflags
)

func main() {
	// Use multi-core CPUs
	runtime.GOMAXPROCS(runtime.NumCPU())

	if err := NewRootCmd().Execute(); err != nil {
		fmt.Printf("cb-spider terminated with error: %v\n", err)
		os.Exit(1)
	}
}

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "cb-spider",
		Short: "CB-Spider API Server for managing multi-cloud infrastructure",
		Run: func(cmd *cobra.Command, args []string) {
			// Get the flags
			useTLS, _ := cmd.Flags().GetBool("tls")
			certPath, _ := cmd.Flags().GetString("cert")
			keyPath, _ := cmd.Flags().GetString("key")
			caCertPath, _ := cmd.Flags().GetString("cacert")
			port, _ := cmd.Flags().GetInt("port")

			if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
				printVersion()
				return
			}

			// WaitGroup to manage both servers
			wg := new(sync.WaitGroup)

			if useTLS {
				// Run the TLS server for spiderlet
				wg.Add(1)
				go func() {
					defer wg.Done()
					restruntime.RunTLSServer(certPath, keyPath, caCertPath, port)
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()
					restruntime.RunServer() // Run the REST server
				}()

			} else {
				// Run the REST server
				wg.Add(1)
				go func() {
					defer wg.Done()
					restruntime.RunServer()
				}()
			}

			wg.Wait() // Wait for both servers to finish
		},
	}

	// Add global flags for version info
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")

	// Add flags for TLS
	rootCmd.Flags().Bool("tls", false, "Enable TLS")
	rootCmd.Flags().String("cert", "", "TLS certificate file")
	rootCmd.Flags().String("key", "", "TLS key file")
	rootCmd.Flags().String("cacert", "", "CA certificate file")

	rootCmd.Flags().Int("port", 10241, "TLS server port")

	// Add subcommands
	rootCmd.AddCommand(NewInfoCmd())

	return rootCmd
}

func NewInfoCmd() *cobra.Command {
	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "Fetch information from the CB-Spider server",
		Run: func(cmd *cobra.Command, args []string) {
			url := "http://" + cr.ServiceIPorName + cr.ServicePort + "/spider/endpointinfo"
			resp, err := http.Get(url)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading response body: %v\n", err)
				return
			}

			fmt.Printf("%s\n", body)
		},
	}

	return infoCmd
}

// Print the version information
func printVersion() {
	fmt.Printf("Version:    %s\n", Version)
	fmt.Printf("Commit SHA: %s\n", CommitSHA)
	fmt.Printf("Build Time: %s\n", BuildTime)
}
