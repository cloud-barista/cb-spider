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

// NewConnectionCmd - Connection Config 관리 기능을 수행하는 Cobra Command 생성
func NewConnectionCmd() *cobra.Command {

	connectionCmd := &cobra.Command{
		Use:   "connection",
		Short: "This is a manageable command for connection config",
		Long:  "This is a manageable command for connection config",
	}

	//  Adds the commands for application.
	connectionCmd.AddCommand(NewConnectionCreateCmd())
	connectionCmd.AddCommand(NewConnectionListCmd())
	connectionCmd.AddCommand(NewConnectionGetCmd())
	connectionCmd.AddCommand(NewConnectionDeleteCmd())

	return connectionCmd
}

// NewConnectionCreateCmd - Connection Config 생성 기능을 수행하는 Cobra Command 생성
func NewConnectionCreateCmd() *cobra.Command {

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "This is create command for connection config",
		Long:  "This is create command for connection config",
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

	createCmd.PersistentFlags().StringVarP(&inData, "indata", "d", "", "input string data")
	createCmd.PersistentFlags().StringVarP(&inFile, "infile", "f", "", "input file path")

	return createCmd
}

// NewConnectionListCmd - Connection Config 목록 기능을 수행하는 Cobra Command 생성
func NewConnectionListCmd() *cobra.Command {

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "This is list command for connection config",
		Long:  "This is list command for connection config",
		Run: func(cmd *cobra.Command, args []string) {
			SetupAndRun(cmd, args)
		},
	}

	return listCmd
}

// NewConnectionGetCmd - Connection Config 조회 기능을 수행하는 Cobra Command 생성
func NewConnectionGetCmd() *cobra.Command {

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "This is get command for connection config",
		Long:  "This is get command for connection config",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if configName == "" {
				logger.Error("failed to validate --name parameter")
				return
			}
			logger.Debug("--name parameter value : ", configName)

			SetupAndRun(cmd, args)
		},
	}

	getCmd.PersistentFlags().StringVarP(&configName, "name", "n", "", "config name")

	return getCmd
}

// NewConnectionDeleteCmd - Connection Config 삭제 기능을 수행하는 Cobra Command 생성
func NewConnectionDeleteCmd() *cobra.Command {

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "This is delete command for connection config",
		Long:  "This is delete command for connection config",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if configName == "" {
				logger.Error("failed to validate --name parameter")
				return
			}
			logger.Debug("--name parameter value : ", configName)

			SetupAndRun(cmd, args)
		},
	}

	deleteCmd.PersistentFlags().StringVarP(&configName, "name", "n", "", "config name")

	return deleteCmd
}
