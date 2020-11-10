// gRPC Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

// Package cmd - 어플리케이션 실행을 위한 Cobra 기반의 CLI Commands 기능 제공
package cmd

import (
	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/config"
	"github.com/spf13/cobra"
)

// ===== [ Constants and Variables ] =====

var (
	configFile string
	inData     string
	inFile     string
	inType     string
	outType    string

	driverName     string
	credentialName string
	regionName     string
	configName     string

	connectionName string
	imageName      string
	specName       string
	vpcName        string
	securityName   string
	keypairName    string
	vmName         string
	action         string
	cspID          string
	force          string

	parser config.Parser
)

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====

// NewRootCmd - 어플리케이션 진입점으로 사용할 Root Cobra Command 생성
func NewRootCmd() *cobra.Command {

	rootCmd := &cobra.Command{
		Use:   "spider",
		Short: "spider is an cb-spider grpc cli tool",
		Long:  "This is a lightweight cb-spider grpc cli tool for Cloud-Barista",
	}

	// 옵션 플래그 설정
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "./grpc_conf.yaml", "config file")
	rootCmd.PersistentFlags().StringVarP(&inType, "input", "i", "yaml", "input format (json/yaml)")
	rootCmd.PersistentFlags().StringVarP(&outType, "output", "o", "yaml", "output format (json/yaml)")

	// Viper 를 사용하는 설정 파서 생성
	parser = config.MakeParser()

	//  Adds the commands for application.
	rootCmd.AddCommand(NewOsCmd())
	rootCmd.AddCommand(NewDriverCmd())
	rootCmd.AddCommand(NewCredentialCmd())
	rootCmd.AddCommand(NewRegionCmd())
	rootCmd.AddCommand(NewConnectionCmd())

	rootCmd.AddCommand(NewImageCmd())
	rootCmd.AddCommand(NewVMSpecCmd())
	rootCmd.AddCommand(NewVPCCmd())
	rootCmd.AddCommand(NewSecurityCmd())
	rootCmd.AddCommand(NewKeyPairCmd())
	rootCmd.AddCommand(NewVMCmd())

	rootCmd.AddCommand(NewSSHCmd())

	return rootCmd
}
