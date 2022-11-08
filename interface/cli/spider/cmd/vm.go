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

// NewVMCmd - VM 관리 기능을 수행하는 Cobra Command 생성
func NewVMCmd() *cobra.Command {

	vmCmd := &cobra.Command{
		Use:   "vm",
		Short: "This is a manageable command for vm",
		Long:  "This is a manageable command for vm",
	}

	//  Adds the commands for application.
	vmCmd.AddCommand(NewVMStartCmd())
	vmCmd.AddCommand(NewVMSuspendCmd())
	vmCmd.AddCommand(NewVMResumeCmd())
	vmCmd.AddCommand(NewVMRebootCmd())
	vmCmd.AddCommand(NewVMListStatusCmd())
	vmCmd.AddCommand(NewVMGetStatusCmd())
	vmCmd.AddCommand(NewVMListCmd())
	vmCmd.AddCommand(NewVMGetCmd())
	vmCmd.AddCommand(NewVMTerminateCmd())
	vmCmd.AddCommand(NewVMListAllCmd())
	vmCmd.AddCommand(NewVMTerminateCSPCmd())
	vmCmd.AddCommand(NewVMRegisterCmd())
	vmCmd.AddCommand(NewVMUnregisterCmd())

	vmCmd.AddCommand(ExVMCmd())

	return vmCmd
}

// NewVMStartCmd - VM 시작 기능을 수행하는 Cobra Command 생성
func NewVMStartCmd() *cobra.Command {

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "This is start command for vm",
		Long:  "This is start command for vm",
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

	startCmd.PersistentFlags().StringVarP(&inData, "indata", "d", "", "input string data")
	startCmd.PersistentFlags().StringVarP(&inFile, "infile", "f", "", "input file path")

	return startCmd
}

// NewVMSuspendCmd - VM Suspend 제어 기능을 수행하는 Cobra Command 생성
func NewVMSuspendCmd() *cobra.Command {

	controlCmd := &cobra.Command{
		Use:   "suspend",
		Short: "This is suspend control command for vm",
		Long:  "This is suspend control command for vm",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if connectionName == "" {
				logger.Error("failed to validate --cname parameter")
				return
			}
			if vmName == "" {
				logger.Error("failed to validate --name parameter")
				return
			}
			logger.Debug("--cname parameter value : ", connectionName)
			logger.Debug("--name parameter value : ", vmName)

			SetupAndRun(cmd, args)
		},
	}

	controlCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")
	controlCmd.PersistentFlags().StringVarP(&vmName, "name", "n", "", "vm name")

	return controlCmd
}

// NewVMResumeCmd - VM Resume 제어 기능을 수행하는 Cobra Command 생성
func NewVMResumeCmd() *cobra.Command {

	controlCmd := &cobra.Command{
		Use:   "resume",
		Short: "This is resume control command for vm",
		Long:  "This is resume control command for vm",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if connectionName == "" {
				logger.Error("failed to validate --cname parameter")
				return
			}
			if vmName == "" {
				logger.Error("failed to validate --name parameter")
				return
			}
			logger.Debug("--cname parameter value : ", connectionName)
			logger.Debug("--name parameter value : ", vmName)

			SetupAndRun(cmd, args)
		},
	}

	controlCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")
	controlCmd.PersistentFlags().StringVarP(&vmName, "name", "n", "", "vm name")

	return controlCmd
}

// NewVMRebootCmd - VM Reboot 제어 기능을 수행하는 Cobra Command 생성
func NewVMRebootCmd() *cobra.Command {

	controlCmd := &cobra.Command{
		Use:   "reboot",
		Short: "This is reboot control command for vm",
		Long:  "This is reboot control command for vm",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if connectionName == "" {
				logger.Error("failed to validate --cname parameter")
				return
			}
			if vmName == "" {
				logger.Error("failed to validate --name parameter")
				return
			}
			logger.Debug("--cname parameter value : ", connectionName)
			logger.Debug("--name parameter value : ", vmName)

			SetupAndRun(cmd, args)
		},
	}

	controlCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")
	controlCmd.PersistentFlags().StringVarP(&vmName, "name", "n", "", "vm name")

	return controlCmd
}

// NewVMListStatusCmd - VM 상태 목록 기능을 수행하는 Cobra Command 생성
func NewVMListStatusCmd() *cobra.Command {

	listStatusCmd := &cobra.Command{
		Use:   "liststatus",
		Short: "This is list status command for vm",
		Long:  "This is list status command for vm",
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

	listStatusCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")

	return listStatusCmd
}

// NewVMGetStatusCmd - VM 상태 조회 기능을 수행하는 Cobra Command 생성
func NewVMGetStatusCmd() *cobra.Command {

	getStatusCmd := &cobra.Command{
		Use:   "getstatus",
		Short: "This is get status command for vm",
		Long:  "This is get status command for vm",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if connectionName == "" {
				logger.Error("failed to validate --cname parameter")
				return
			}
			if vmName == "" {
				logger.Error("failed to validate --name parameter")
				return
			}
			logger.Debug("--cname parameter value : ", connectionName)
			logger.Debug("--name parameter value : ", vmName)

			SetupAndRun(cmd, args)
		},
	}

	getStatusCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")
	getStatusCmd.PersistentFlags().StringVarP(&vmName, "name", "n", "", "vm name")

	return getStatusCmd
}

// NewVMListCmd - VM 목록 기능을 수행하는 Cobra Command 생성
func NewVMListCmd() *cobra.Command {

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "This is list command for vm",
		Long:  "This is list command for vm",
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

// NewVMGetCmd - VM 조회 기능을 수행하는 Cobra Command 생성
func NewVMGetCmd() *cobra.Command {

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "This is get command for vm",
		Long:  "This is get command for vm",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if connectionName == "" {
				logger.Error("failed to validate --cname parameter")
				return
			}
			if vmName == "" {
				logger.Error("failed to validate --name parameter")
				return
			}
			logger.Debug("--cname parameter value : ", connectionName)
			logger.Debug("--name parameter value : ", vmName)

			SetupAndRun(cmd, args)
		},
	}

	getCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")
	getCmd.PersistentFlags().StringVarP(&vmName, "name", "n", "", "vm name")

	return getCmd
}

// NewVMTerminateCmd - VM 삭제 기능을 수행하는 Cobra Command 생성
func NewVMTerminateCmd() *cobra.Command {

	terminateCmd := &cobra.Command{
		Use:   "terminate",
		Short: "This is terminate command for vm",
		Long:  "This is terminate command for vm",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if connectionName == "" {
				logger.Error("failed to validate --cname parameter")
				return
			}
			if vmName == "" {
				logger.Error("failed to validate --name parameter")
				return
			}
			logger.Debug("--cname parameter value : ", connectionName)
			logger.Debug("--name parameter value : ", vmName)
			logger.Debug("--force parameter value : ", force)

			SetupAndRun(cmd, args)
		},
	}

	terminateCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")
	terminateCmd.PersistentFlags().StringVarP(&vmName, "name", "n", "", "vm name")
	terminateCmd.PersistentFlags().StringVarP(&force, "force", "", "false", "force flg (true/false)")

	return terminateCmd
}

// NewVMListAllCmd - 관리 VM 목록 기능을 수행하는 Cobra Command 생성
func NewVMListAllCmd() *cobra.Command {

	listAllCmd := &cobra.Command{
		Use:   "listall",
		Short: "This is list all command for vm",
		Long:  "This is list all command for vm",
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

// NewVMTerminateCSPCmd - 관리 VM 삭제 기능을 수행하는 Cobra Command 생성
func NewVMTerminateCSPCmd() *cobra.Command {

	terminateCSPCmd := &cobra.Command{
		Use:   "terminatecsp",
		Short: "This is terminate csp command for vm",
		Long:  "This is terminate csp command for vm",
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

	terminateCSPCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")
	terminateCSPCmd.PersistentFlags().StringVarP(&cspID, "id", "", "", "csp id")

	return terminateCSPCmd
}

func ExVMCmd() *cobra.Command {
	exVMCmd := &cobra.Command{
		Use:   "ex",
                Short: "example to start vm",
                Long: "example to start vm",
                Run: func(cmd *cobra.Command, args []string) {
			excmd := `
[Start VM]
spctl vm start -d \
    '{
      "ConnectionName":"aws-ohio-config",
      "ReqInfo": {
        "Name": "spider-vm-01",
        
        "ImageName": "ami-097a2df4ac947655f",
        "VMSpecName": "t2.micro",

        "VPCName": "spider-vpc-01",
        "SubnetName": "spider-subnet-01",
        "SecurityGroupNames": [ "spider-sg-01" ],
        "KeyPairName": "spider-key-01"
      }
    }'

[List VM]
spctl --cname aws-ohio-config vm list

[Get VM]
spctl --cname aws-ohio-config vm get -n spider-vm-01

[Terminate VM]
spctl --cname aws-ohio-config vm terminate -n spider-vm-01

`
                        fmt.Printf(excmd)
                },
	}

	return exVMCmd
}

// NewVMRegisterCmd - VM Register 등록 기능을 수행하는 Cobra Command 생성
func NewVMRegisterCmd() *cobra.Command {

	registerCmd := &cobra.Command{
		Use:   "register",
		Short: "This is register command for vm",
		Long:  "This is register command for vm",
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

// NewVMUnregisterCmd - VM Register 제거 기능을 수행하는 Cobra Command 생성
func NewVMUnregisterCmd() *cobra.Command {

	unregisterCmd := &cobra.Command{
		Use:   "unregister",
		Short: "This is unregister command for vm",
		Long:  "This is unregister command for vm",
		Run: func(cmd *cobra.Command, args []string) {
			logger := logger.NewLogger()
			if connectionName == "" {
				logger.Error("failed to validate --cname parameter")
				return
			}
			if vmName == "" {
				logger.Error("failed to validate --name parameter")
				return
			}
			logger.Debug("--cname parameter value : ", connectionName)
			logger.Debug("--name parameter value : ", vmName)

			SetupAndRun(cmd, args)
		},
	}

	unregisterCmd.PersistentFlags().StringVarP(&connectionName, "cname", "", "", "connection name")
	unregisterCmd.PersistentFlags().StringVarP(&vmName, "name", "n", "", "vm name")

	return unregisterCmd
}
