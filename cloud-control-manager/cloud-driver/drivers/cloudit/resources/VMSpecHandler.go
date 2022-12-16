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

func (vmSpecHandler *ClouditVMSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VMSPEC, VMSpec, "ListVMSpec()")
	start := call.Start()

	vmSpecHandler.Client.TokenID = vmSpecHandler.CredentialInfo.AuthToken
	authHeader := vmSpecHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	list, err := specs.List(vmSpecHandler.Client, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List VMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	vmSpecList := make([]*irs.VMSpecInfo, len(*list))
	for i, spec := range *list {
		vmSpecList[i] = setterVMSpec(vmSpecHandler.CredentialInfo.IdentityEndpoint, spec)
	}
	LoggingInfo(hiscallInfo, start)
	return vmSpecList, nil
}

func (vmSpecHandler *ClouditVMSpecHandler) GetVMSpec(Name string) (irs.VMSpecInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VMSPEC, Name, "GetVMSpec()")

	start := call.Start()
	specInfo, err := vmSpecHandler.GetVMSpecByName(vmSpecHandler.CredentialInfo.IdentityEndpoint, Name)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VMSpecInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)

	return *specInfo, nil
}

func (vmSpecHandler *ClouditVMSpecHandler) ListOrgVMSpec() (string, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VMSPEC, VMSpec, "ListOrgVMSpec()")

	vmSpecHandler.Client.TokenID = vmSpecHandler.CredentialInfo.AuthToken
	authHeader := vmSpecHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	start := call.Start()
	list, err := specs.List(vmSpecHandler.Client, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List OrgVMSpec. err = %s", err.Error()))
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
		getErr := errors.New(fmt.Sprintf("Failed to List OrgVMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}

	jsonString := string(jsonBytes)

	return jsonString, nil
}

func (vmSpecHandler *ClouditVMSpecHandler) GetOrgVMSpec(Name string) (string, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VMSPEC, Name, "GetOrgVMSpec()")

	start := call.Start()
	specInfo, err := vmSpecHandler.GetVMSpecByName(vmSpecHandler.CredentialInfo.IdentityEndpoint, Name)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get OrgVMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	LoggingInfo(hiscallInfo, start)

	jsonBytes, err := json.Marshal(specInfo)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get OrgVMSpec. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
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
