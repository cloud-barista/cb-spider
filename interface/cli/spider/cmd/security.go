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
	"fmt"
	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====

// NewSecurityCmd - Security 관리 기능을 수행하는 Cobra Command 생성
func NewSecurityCmd() *cobra.Command {

	securityCmd := &cobra.Command{
		Use:   "security",
		Short: "This is a manageable command for security",
		Long:  "This is a manageable command for security",
	}

	//  Adds the commands for application.
	securityCmd.AddCommand(NewSecurityCreateCmd())
	securityCmd.AddCommand(NewSecurityListCmd())
	securityCmd.AddCommand(NewSecurityGetCmd())
	securityCmd.AddCommand(NewSecurityDeleteCmd())
	securityCmd.AddCommand(NewSecurityListAllCmd())
	securityCmd.AddCommand(NewSecurityDeleteCSPCmd())
	securityCmd.AddCommand(NewSecurityRegisterCmd())
	securityCmd.AddCommand(NewSecurityUnregisterCmd())
	securityCmd.AddCommand(ExSecurityCmd())

	return securityCmd
}

// NewSecurityCreateCmd - Security 생성 기능을 수행하는 Cobra Command 생성
func NewSecurityCreateCmd() *cobra.Command {

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "This is create command for security",
		Long:  "This is create command for security",
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

// NewSecurityListCmd - Security 목록 기능을 수행하는 Cobra Command 생성
func NewSecurityListCmd() *cobra.Command {

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "This is list command for security",
		Long:  "This is list command for security",
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

// NewSecurityGetCmd - Security 조회 기능을 수행하는 Cobra Command 생성
func NewSecurityGetCmd() *cobra.Command {

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "This is get command for security",
		Long:  "This is get command for security",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if connectionName == "" {
				logger.Error("failed to validate --cname parameter")
				return
			}
			if securityName == "" {
				logger.Error("failed to validate --name parameter")
				return
			}
			logger.Debug("--cname parameter value : ", connectionName)
			logger.Debug("--name parameter value : ", securityName)

			SetupAndRun(cmd, args)
		},
	}

	getCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")
	getCmd.PersistentFlags().StringVarP(&securityName, "name", "n", "", "security name")

	return getCmd
}

// NewSecurityDeleteCmd - Security 삭제 기능을 수행하는 Cobra Command 생성
func NewSecurityDeleteCmd() *cobra.Command {

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "This is delete command for security",
		Long:  "This is delete command for security",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if connectionName == "" {
				logger.Error("failed to validate --cname parameter")
				return
			}
			if securityName == "" {
				logger.Error("failed to validate --name parameter")
				return
			}
			logger.Debug("--cname parameter value : ", connectionName)
			logger.Debug("--name parameter value : ", securityName)
			logger.Debug("--force parameter value : ", force)

			SetupAndRun(cmd, args)
		},
	}

	deleteCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")
	deleteCmd.PersistentFlags().StringVarP(&securityName, "name", "n", "", "security name")
	deleteCmd.PersistentFlags().StringVarP(&force, "force", "", "false", "force flg (true/false)")

	return deleteCmd
}

// NewSecurityListAllCmd - 관리 Security 목록 기능을 수행하는 Cobra Command 생성
func NewSecurityListAllCmd() *cobra.Command {

	listAllCmd := &cobra.Command{
		Use:   "listall",
		Short: "This is list all command for security",
		Long:  "This is list all command for security",
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

	listAllCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")

	return listAllCmd
}

// NewSecurityDeleteCSPCmd - 관리 Security 삭제 기능을 수행하는 Cobra Command 생성
func NewSecurityDeleteCSPCmd() *cobra.Command {

	deleteCSPCmd := &cobra.Command{
		Use:   "deletecsp",
		Short: "This is delete csp command for security",
		Long:  "This is delete csp command for security",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if connectionName == "" {
				logger.Error("failed to validate --cname parameter")
				return
			}
			if cspID == "" {
				logger.Error("failed to validate --id parameter")
				return
			}
			logger.Debug("--cname parameter value : ", connectionName)
			logger.Debug("--id parameter value : ", cspID)

			SetupAndRun(cmd, args)
		},
	}

	deleteCSPCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")
	deleteCSPCmd.PersistentFlags().StringVarP(&cspID, "id", "", "", "csp id")

	return deleteCSPCmd
}

// NewSecurityRegisterCmd - Security Register 등록 기능을 수행하는 Cobra Command 생성
func NewSecurityRegisterCmd() *cobra.Command {

	registerCmd := &cobra.Command{
		Use:   "register",
		Short: "This is register command for security",
		Long:  "This is register command for security",
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

	registerCmd.PersistentFlags().StringVarP(&inData, "indata", "d", "", "input string data")
	registerCmd.PersistentFlags().StringVarP(&inFile, "infile", "f", "", "input file path")

	return registerCmd
}

// NewSecurityUnregisterCmd - Security Register 제거 기능을 수행하는 Cobra Command 생성
func NewSecurityUnregisterCmd() *cobra.Command {

	unregisterCmd := &cobra.Command{
		Use:   "unregister",
		Short: "This is unregister command for security",
		Long:  "This is unregister command for security",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if connectionName == "" {
				logger.Error("failed to validate --cname parameter")
				return
			}
			if securityName == "" {
				logger.Error("failed to validate --name parameter")
				return
			}
			logger.Debug("--cname parameter value : ", connectionName)
			logger.Debug("--name parameter value : ", securityName)

			SetupAndRun(cmd, args)
		},
	}

	unregisterCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")
	unregisterCmd.PersistentFlags().StringVarP(&securityName, "name", "n", "", "security name")

	return unregisterCmd
}

func ExSecurityCmd() *cobra.Command {
        exSecurityCmd := &cobra.Command{
                Use:   "ex",
                Short: "example to create security",
                Long: "example to create securityvm",
                Run: func(cmd *cobra.Command, args []string) {
                        excmd := `
[Create SG]
spctl security create -d \
    '{
      "ConnectionName":"aws-ohio-config",
      "ReqInfo": {
        "Name": "spider-sg-01",

        "VPCName": "spider-vpc-01",

        "SecurityRules": [
          {
            "Direction" : "inbound",
            "IPProtocol" : "all",
            "FromPort": "-1",
            "ToPort" : "-1"
          }
        ]
      }
    }'

[List SG]
spctl --cname aws-ohio-config security list

[Get SG]
spctl --cname aws-ohio-config security get -n spider-sg-01

[Delete SG]
spctl --cname aws-ohio-config security delete -n spider-sg-01

`
                        fmt.Printf(excmd)
                },
        }

        return exSecurityCmd
}
