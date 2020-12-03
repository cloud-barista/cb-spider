// gRPC Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

package cmd

import (
	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
	"github.com/spf13/cobra"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====

// NewSSHCmd - SSH 관리 기능을 수행하는 Cobra Command 생성
func NewSSHCmd() *cobra.Command {

	sshCmd := &cobra.Command{
		Use:   "ssh",
		Short: "This is a manageable command for ssh",
		Long:  "This is a manageable command for ssh",
	}

	//  Adds the commands for application.
	sshCmd.AddCommand(NewSSHRunCmd())

	return sshCmd
}

// NewSSHRunCmd - SSH 실행 기능을 수행하는 Cobra Command 생성
func NewSSHRunCmd() *cobra.Command {

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "This is run command for ssh",
		Long:  "This is run command for ssh",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			readInDataFromFile()
			if inData == "" {
				logger.Error("failed to validate --indata parameter")
				return
			}
			logger.Debug("--indata parameter value : \n", inData)
			logger.Debug("--infile parameter value : ", inFile)

			SetupAndRun(cmd, args)
		},
	}

	runCmd.PersistentFlags().StringVarP(&inData, "indata", "d", "", "input string data")
	runCmd.PersistentFlags().StringVarP(&inFile, "infile", "f", "", "input file path")

	return runCmd
}
