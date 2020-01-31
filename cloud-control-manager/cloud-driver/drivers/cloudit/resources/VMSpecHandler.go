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
	cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/specs"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strconv"
	"strings"
)

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type ClouditVMSpecHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func setterVMSpec(vmSpec specs.VMSpecInfo) *irs.VMSpecInfo {
	vmSpecInfo := &irs.VMSpecInfo{
		Name: vmSpec.Name,
		VCpu: irs.VCpuInfo{Count: strconv.Itoa(vmSpec.Cpu)},
		Mem:  strconv.Itoa(vmSpec.Mem),
		Gpu:  []irs.GpuInfo{{Conut: strconv.Itoa(vmSpec.GPU)}},
		//KeyValueList: nil,
	}

	return vmSpecInfo
}

func (vmSpecHandler *ClouditVMSpecHandler) ListVMSpec(Region string) ([]*irs.VMSpecInfo, error) {
	vmSpecHandler.Client.TokenID = vmSpecHandler.CredentialInfo.AuthToken
	authHeader := vmSpecHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	list, err := specs.List(vmSpecHandler.Client, &requestOpts)
	if err != nil {
		cblogger.Error(fmt.Sprintf("failed to get VM spec list, err : %s", err))
		return nil, err
	}

	vmSpecList := make([]*irs.VMSpecInfo, len(*list))
	for i, spec := range *list {
		vmSpecList[i] = setterVMSpec(spec)
	}
	return vmSpecList, nil
}

func (vmSpecHandler *ClouditVMSpecHandler) GetVMSpec(Region string, Name string) (irs.VMSpecInfo, error) {

	specInfo, err := vmSpecHandler.GetVVMSpecByName(Name)
	if err != nil {
		cblogger.Error(fmt.Sprintf("failed to get VM spec, err : %s", err))
		notFoundErr := errors.New("failed to get VM spec")

		return irs.VMSpecInfo{}, notFoundErr
	}
	return *specInfo, nil
}

func (vmSpecHandler *ClouditVMSpecHandler) ListOrgVMSpec(Region string) (string, error) {
	vmSpecHandler.Client.TokenID = vmSpecHandler.CredentialInfo.AuthToken
	authHeader := vmSpecHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	list, err := specs.List(vmSpecHandler.Client, &requestOpts)
	if err != nil {
		cblogger.Error(fmt.Sprintf("failed to get VM spec list, err : %s", err))
		return "", err
	}

	vmSpecList := make([]*irs.VMSpecInfo, len(*list))
	for i, spec := range *list {
		vmSpecList[i] = setterVMSpec(spec)
	}

	jsonBytes, err := json.Marshal(vmSpecList)
	if err != nil {
		panic(err)
	}

	jsonString := string(jsonBytes)

	return jsonString, nil
}

func (vmSpecHandler *ClouditVMSpecHandler) GetOrgVMSpec(Region string, Name string) (string, error) {
	specInfo, err := vmSpecHandler.GetVVMSpecByName(Name)
	if err != nil {
		cblogger.Error(fmt.Sprintf("failed to get VM spec, err : %s", err))
		notFoundErr := errors.New("failed to get VM spec")

		return "", notFoundErr
	}

	jsonBytes, err := json.Marshal(specInfo)
	if err != nil {
		panic(err)
	}

	jsonString := string(jsonBytes)

	return jsonString, err
}

func (vmSpecHandler *ClouditVMSpecHandler) GetVVMSpecByName(specName string) (*irs.VMSpecInfo, error) {
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
			specInfo = setterVMSpec(spec)
			break
		}
	}

	if specInfo == nil {
		err := errors.New(fmt.Sprintf("failed to find vmSpec with name %s", specName))
		return nil, err
	}

	return specInfo, nil
}
