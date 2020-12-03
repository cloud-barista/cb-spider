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

	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====

// NewVMSpecCmd - VM Spec 관리 기능을 수행하는 Cobra Command 생성
func NewVMSpecCmd() *cobra.Command {

	vmSpecCmd := &cobra.Command{
		Use:   "vmspec",
		Short: "This is a manageable command for vm spec",
		Long:  "This is a manageable command for vm spec",
	}

	//  Adds the commands for application.
	vmSpecCmd.AddCommand(NewVMSpecListCmd())
	vmSpecCmd.AddCommand(NewVMSpecGetCmd())
	vmSpecCmd.AddCommand(NewVMSpecListOrgCmd())
	vmSpecCmd.AddCommand(NewVMSpecGetOrgCmd())

	return vmSpecCmd
}

// NewVMSpecListCmd - VM Spec 목록 기능을 수행하는 Cobra Command 생성
func NewVMSpecListCmd() *cobra.Command {

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "This is list command for vm spec",
		Long:  "This is list command for vm spec",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if connectionName == "" {
				logger.Error("failed to validate --cname parameter")
				return
			}
			logger.Debug("--cname parameter value : ", connectionName)

			SetupAndRun(cmd, args)
		},
	}

	listCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")

	return listCmd
}

// NewVMSpecGetCmd - VM Spec 조회 기능을 수행하는 Cobra Command 생성
func NewVMSpecGetCmd() *cobra.Command {

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "This is get command for vm spec",
		Long:  "This is get command for vm spec",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if connectionName == "" {
				logger.Error("failed to validate --cname parameter")
				return
			}
			if specName == "" {
				logger.Error("failed to validate --name parameter")
				return
			}
			logger.Debug("--cname parameter value : ", connectionName)
			logger.Debug("--name parameter value : ", specName)

			SetupAndRun(cmd, args)
		},
	}

	getCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")
	getCmd.PersistentFlags().StringVarP(&specName, "name", "n", "", "spec name")

	return getCmd
}

// NewVMSpecListOrgCmd - 클라우드의 원래 VM Spec 목록 기능을 수행하는 Cobra Command 생성
func NewVMSpecListOrgCmd() *cobra.Command {

	listOrgCmd := &cobra.Command{
		Use:   "listorg",
		Short: "This is original list command for vm spec",
		Long:  "This is original list command for vm spec",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if connectionName == "" {
				logger.Error("failed to validate --cname parameter")
				return
			}
			logger.Debug("--cname parameter value : ", connectionName)

			SetupAndRun(cmd, args)
		},
	}

	listOrgCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")

	return listOrgCmd
}

// NewVMSpecGetOrgCmd - 클라우드의 원래 VM Spec 조회 기능을 수행하는 Cobra Command 생성
func NewVMSpecGetOrgCmd() *cobra.Command {

	getOrgCmd := &cobra.Command{
		Use:   "getorg",
		Short: "This is original get command for vm spec",
		Long:  "This is original get command for vm spec",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if connectionName == "" {
				logger.Error("failed to validate --cname parameter")
				return
			}
			if specName == "" {
				logger.Error("failed to validate --name parameter")
				return
			}
			logger.Debug("--cname parameter value : ", connectionName)
			logger.Debug("--name parameter value : ", specName)

			SetupAndRun(cmd, args)
		},
	}

	getOrgCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")
	getOrgCmd.PersistentFlags().StringVarP(&specName, "name", "n", "", "spec name")

	return getOrgCmd
}
