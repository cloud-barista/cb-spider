package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"

	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
	"github.com/cloud-barista/cb-spider/interface/api"
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

// SetupAndRun - SPIDER GRPC CLI 구동
func SetupAndRun(cmd *cobra.Command, args []string) {
	logger := logger.NewLogger()

	var (
		result string
		err    error

		cim *api.CIMApi = nil
		ccm *api.CCMApi = nil
	)

	// panic 처리
	defer func() {
		if r := recover(); r != nil {
			logger.Error("spider is stopped : ", r)
		}
	}()

	if cmd.Parent().Name() == "os" || cmd.Parent().Name() == "driver" || cmd.Parent().Name() == "credential" ||
		cmd.Parent().Name() == "region" || cmd.Parent().Name() == "connection" {

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

	} else {

		// CCM API 설정
		ccm = api.NewCloudResourceHandler()
		err = ccm.SetConfigPath(configFile)
		if err != nil {
			logger.Error("failed to set config : ", err)
			return
		}
		err = ccm.Open()
		if err != nil {
			logger.Error("cim api open failed : ", err)
			return
		}
		defer ccm.Close()

	}

	// 입력 파라미터 처리
	if outType != "json" && outType != "yaml" {
		logger.Error("failed to validate --output parameter : ", outType)
		return
	}
	if inType != "json" && inType != "yaml" {
		logger.Error("failed to validate --input parameter : ", inType)
		return
	}

	if cmd.Parent().Name() == "os" || cmd.Parent().Name() == "driver" || cmd.Parent().Name() == "credential" ||
		cmd.Parent().Name() == "region" || cmd.Parent().Name() == "connection" {
		cim.SetInType(inType)
		cim.SetOutType(outType)
	} else {
		ccm.SetInType(inType)
		ccm.SetOutType(outType)
	}

	logger.Debug("--input parameter value : ", inType)
	logger.Debug("--output parameter value : ", outType)

	result = ""
	err = nil

	switch cmd.Parent().Name() {
	case "os":
		switch cmd.Name() {
		case "list":
			result, err = cim.ListCloudOS()
		}
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
	case "connection":
		switch cmd.Name() {
		case "create":
			result, err = cim.CreateConnectionConfig(inData)
		case "list":
			result, err = cim.ListConnectionConfig()
		case "get":
			result, err = cim.GetConnectionConfigByParam(configName)
		case "delete":
			result, err = cim.DeleteConnectionConfigByParam(configName)
		}
	case "image":
		switch cmd.Name() {
		case "create":
			result, err = ccm.CreateImage(inData)
		case "list":
			result, err = ccm.ListImage(connectionName)
		case "get":
			result, err = ccm.GetImageByParam(connectionName, imageName)
		case "delete":
			result, err = ccm.DeleteImageByParam(connectionName, imageName)
		}
	case "vmspec":
		switch cmd.Name() {
		case "list":
			result, err = ccm.ListVMSpecByParam(connectionName)
		case "get":
			result, err = ccm.GetVMSpecByParam(connectionName, specName)
		case "listorg":
			result, err = ccm.ListOrgVMSpecByParam(connectionName)
		case "getorg":
			result, err = ccm.GetOrgVMSpecByParam(connectionName, specName)
		}
	case "vpc":
		switch cmd.Name() {
		case "create":
			result, err = ccm.CreateVPC(inData)
		case "list":
			result, err = ccm.ListVPCByParam(connectionName)
		case "get":
			result, err = ccm.GetVPCByParam(connectionName, vpcName)
		case "delete":
			result, err = ccm.DeleteVPCByParam(connectionName, vpcName, force)
		case "listall":
			result, err = ccm.ListAllVPCByParam(connectionName)
		case "deletecsp":
			result, err = ccm.DeleteCSPVPCByParam(connectionName, cspID)
		}
	case "security":
		switch cmd.Name() {
		case "create":
			result, err = ccm.CreateSecurity(inData)
		case "list":
			result, err = ccm.ListSecurityByParam(connectionName)
		case "get":
			result, err = ccm.GetSecurityByParam(connectionName, securityName)
		case "delete":
			result, err = ccm.DeleteSecurityByParam(connectionName, securityName, force)
		case "listall":
			result, err = ccm.ListAllSecurityByParam(connectionName)
		case "deletecsp":
			result, err = ccm.DeleteCSPSecurityByParam(connectionName, cspID)
		}
	case "keypair":
		switch cmd.Name() {
		case "create":
			result, err = ccm.CreateKey(inData)
		case "list":
			result, err = ccm.ListKeyByParam(connectionName)
		case "get":
			result, err = ccm.GetKeyByParam(connectionName, keypairName)
		case "delete":
			result, err = ccm.DeleteKeyByParam(connectionName, keypairName, force)
		case "listall":
			result, err = ccm.ListAllKeyByParam(connectionName)
		case "deletecsp":
			result, err = ccm.DeleteCSPKeyByParam(connectionName, cspID)
		}
	case "vm":
		switch cmd.Name() {
		case "start":
			result, err = ccm.StartVM(inData)
		case "control":
			result, err = ccm.ControlVMByParam(connectionName, vmName, action)
		case "liststatus":
			result, err = ccm.ListVMStatusByParam(connectionName)
		case "getstatus":
			result, err = ccm.GetVMStatusByParam(connectionName, vmName)
		case "list":
			result, err = ccm.ListVMByParam(connectionName)
		case "get":
			result, err = ccm.GetVMByParam(connectionName, vmName)
		case "terminate":
			result, err = ccm.TerminateVMByParam(connectionName, vmName, force)
		case "listall":
			result, err = ccm.ListAllVMByParam(connectionName)
		case "terminatecsp":
			result, err = ccm.TerminateCSPVMByParam(connectionName, cspID)
		}
	case "ssh":
		switch cmd.Name() {
		case "run":
			result, err = ccm.SSHRun(inData)
		}
	}

	if err != nil {
		logger.Error("failed to run command: ", err)
	}

	fmt.Printf("%s\n", result)
}
