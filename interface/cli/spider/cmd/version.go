// gRPC Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====

// NewVersionCmd - 버전 표시 기능을 수행하는 Cobra Command 생성
func NewVersionCmd() *cobra.Command {

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "This is a version command for spctl",
		Long:  "This is a version command for spctl",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "spctl:\n    Version: %s\n    Commit SHA: %s\n    Go version: %s\n"+
				"    OS/Arch: %s\n    Build Time: %s\n    Build User: %s\n",
				Version, CommitSHA, runtime.Version(), fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH), Time, User)
		},
	}

	return versionCmd
}
