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
	"io/ioutil"

	"github.com/spf13/cobra"

	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
	"github.com/cloud-barista/cb-spider/interface/api"
	"github.com/cloud-barista/cb-spider/interface/cli/cbadm/proc"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// ===== [ Private Functions ] =====

func readInDataFromFile() {
	logger := logger.NewLogger()
	if inData == "" {
		if inFile != "" {
			dat, err := ioutil.ReadFile(inFile)
			if err != nil {
				logger.Error("failed to read file : ", inFile)
				return
			}
			inData = string(dat)
		}
	}
}

// ===== [ Public Functions ] =====

// SetupAndRun - cbadm GRPC CLI 구동
func SetupAndRun(cmd *cobra.Command, args []string) {
	logger := logger.NewLogger()

	var (
		result string
		err    error
	)

	// panic 처리
	defer func() {
		if r := recover(); r != nil {
			logger.Error("cbadm is stopped : ", r)
		}
	}()

	// CIM API 설정
	cim = api.NewCloudInfoManager()
	err = cim.SetConfigPath(configFile)
	if err != nil {
		logger.Error("failed to set config : ", err)
		return
	}
	err = cim.Open()
	if err != nil {
		logger.Error("cim api open failed : ", err)
		return
	}
	defer cim.Close()

	// 입력 파라미터 처리
	if outType != "json" && outType != "yaml" {
		logger.Error("failed to validate --output parameter : ", outType)
		return
	}
	if inType != "json" && inType != "yaml" {
		logger.Error("failed to validate --input parameter : ", inType)
		return
	}
	cim.SetInType(inType)
	cim.SetOutType(outType)

	logger.Debug("--input parameter value : ", inType)
	logger.Debug("--output parameter value : ", outType)

	result = ""
	err = nil

	switch cmd.Parent().Name() {
	case "driver":
		switch cmd.Name() {
		case "create":
			result, err = cim.CreateCloudDriver(inData)
		case "list":
			result, err = cim.ListCloudDriver()
		case "get":
			result, err = cim.GetCloudDriverByParam(driverName)
		case "delete":
			result, err = cim.DeleteCloudDriverByParam(driverName)
		}
	case "credential":
		switch cmd.Name() {
		case "create":
			result, err = cim.CreateCredential(inData)
		case "list":
			result, err = cim.ListCredential()
		case "get":
			result, err = cim.GetCredentialByParam(credentialName)
		case "delete":
			result, err = cim.DeleteCredentialByParam(credentialName)
		}
	case "region":
		switch cmd.Name() {
		case "create":
			result, err = cim.CreateRegion(inData)
		case "list":
			result, err = cim.ListRegion()
		case "get":
			result, err = cim.GetRegionByParam(regionName)
		case "delete":
			result, err = cim.DeleteRegionByParam(regionName)
		}
	case "connect-infos":
		switch cmd.Name() {
		case "create":
			result, err = cim.CreateConnectionConfig(inData)
		case "list":
			result, err = proc.ListConnectInfos(cim)
		case "get":
			result, err = proc.GetConnectInfos(cim, configName)
		case "delete":
			result, err = cim.DeleteConnectionConfigByParam(configName)
		}
	}

	if err != nil {
		if outType == "yaml" {
			fmt.Fprintf(cmd.OutOrStdout(), "message: %v\n", err)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "{\"message\": \"%v\"}\n", err)
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", result)
	}
}
