// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by hyokyung.kim@innogrid.co.kr, 2020.01.

package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/specs"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	VMSpec = "VMSPEC"
)

type ClouditVMSpecHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func setterVMSpec(region string, vmSpec specs.VMSpecInfo) *irs.VMSpecInfo {
	vmSpecInfo := &irs.VMSpecInfo{
		Region:       region,
		Name:         vmSpec.Name,
		VCpu:         irs.VCpuInfo{Count: strconv.Itoa(vmSpec.Cpu)},
		Gpu:          []irs.GpuInfo{{Count: strconv.Itoa(vmSpec.GPU)}},
		KeyValueList: nil,
	}
	vmSpecInfo.Mem = strconv.FormatFloat(float64(vmSpec.Mem)*1024, 'f', 0, 64)
	return vmSpecInfo
}

func (vmSpecHandler *ClouditVMSpecHandler) ListVMSpec(Region string) ([]*irs.VMSpecInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Client.IdentityEndpoint, call.VMSPEC, VMSpec, "ListVMSpec()")

	vmSpecHandler.Client.TokenID = vmSpecHandler.CredentialInfo.AuthToken
	authHeader := vmSpecHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	start := call.Start()
	list, err := specs.List(vmSpecHandler.Client, &requestOpts)
	if err != nil {
		getError := errors.New(fmt.Sprintf("failed to get VM spec list, err : %s", err.Error()))
		cblogger.Error(getError.Error())
		LoggingError(hiscallInfo, getError)
		return nil, getError
	}
	LoggingInfo(hiscallInfo, start)

	vmSpecList := make([]*irs.VMSpecInfo, len(*list))
	for i, spec := range *list {
		vmSpecList[i] = setterVMSpec(Region, spec)
	}
	return vmSpecList, nil
}

func (vmSpecHandler *ClouditVMSpecHandler) GetVMSpec(Region string, Name string) (irs.VMSpecInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Client.IdentityEndpoint, call.VMSPEC, Name, "GetVMSpec()")

	start := call.Start()
	specInfo, err := vmSpecHandler.GetVMSpecByName(Region, Name)
	if err != nil {
		notFoundErr := errors.New(fmt.Sprintf("failed to get VM spec, err : %s", err.Error()))
		cblogger.Error(notFoundErr.Error())
		LoggingError(hiscallInfo, notFoundErr)
		return irs.VMSpecInfo{}, notFoundErr
	}
	LoggingInfo(hiscallInfo, start)

	return *specInfo, nil
}

func (vmSpecHandler *ClouditVMSpecHandler) ListOrgVMSpec(Region string) (string, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Client.IdentityEndpoint, call.VMSPEC, VMSpec, "ListOrgVMSpec()")

	vmSpecHandler.Client.TokenID = vmSpecHandler.CredentialInfo.AuthToken
	authHeader := vmSpecHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	start := call.Start()
	list, err := specs.List(vmSpecHandler.Client, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("failed to get VM spec list, err : %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	LoggingInfo(hiscallInfo, start)

	var jsonResult struct {
		Result []specs.VMSpecInfo `json:"list"`
	}
	jsonResult.Result = *list
	jsonBytes, err := json.Marshal(jsonResult)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return "", err
	}

	jsonString := string(jsonBytes)

	return jsonString, nil
}

func (vmSpecHandler *ClouditVMSpecHandler) GetOrgVMSpec(Region string, Name string) (string, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vmSpecHandler.Client.IdentityEndpoint, call.VMSPEC, Name, "GetOrgVMSpec()")

	start := call.Start()
	specInfo, err := vmSpecHandler.GetVMSpecByName(Region, Name)
	if err != nil {
		notFoundErr := errors.New(fmt.Sprintf("failed to get VM spec, err : %s", err.Error()))
		cblogger.Error(notFoundErr.Error())
		LoggingError(hiscallInfo, notFoundErr)
		return "", notFoundErr
	}
	LoggingInfo(hiscallInfo, start)

	jsonBytes, err := json.Marshal(specInfo)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return "", err
	}

	jsonString := string(jsonBytes)

	return jsonString, err
}

func (vmSpecHandler *ClouditVMSpecHandler) GetVMSpecByName(region string, specName string) (*irs.VMSpecInfo, error) {
	vmSpecHandler.Client.TokenID = vmSpecHandler.CredentialInfo.AuthToken
	authHeader := vmSpecHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	specList, err := specs.List(vmSpecHandler.Client, &requestOpts)
	if err != nil {
		return nil, err
	}

	var specInfo *irs.VMSpecInfo
	for _, spec := range *specList {
		if strings.EqualFold(spec.Name, specName) {
			specInfo = setterVMSpec(region, spec)
			break
		}
	}

	if specInfo == nil {
		err := errors.New(fmt.Sprintf("failed to find vmSpec with name %s", specName))
		return nil, err
	}

	return specInfo, nil
}
