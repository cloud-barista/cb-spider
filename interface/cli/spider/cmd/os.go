// gRPC Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

package cmd

import (
	"github.com/spf13/cobra"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====

// NewOsCmd - Cloud OS 관리 기능을 수행하는 Cobra Command 생성
func NewOsCmd() *cobra.Command {

	osCmd := &cobra.Command{
		Use:   "os",
		Short: "This is a manageable command for cloud os",
		Long:  "This is a manageable command for cloud os",
	}

	//  Adds the commands for application.
	osCmd.AddCommand(NewOsListCmd())

	return osCmd
}

// NewOsListCmd - Cloud OS 목록 기능을 수행하는 Cobra Command 생성
func NewOsListCmd() *cobra.Command {

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "This is list command for cloud os",
		Long:  "This is list command for cloud os",
		Run: func(cmd *cobra.Command, args []string) {
			SetupAndRun(cmd, args)
		},
	}

	return listCmd
}
