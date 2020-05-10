package resources

import (
	"fmt"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/nic"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/specs"
	"strings"
)

// VM Spec 정보 조회
func GetVMSpecByName(authHeader map[string]string, reqClient *client.RestClient, specName string) (*string, error) {
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	specList, err := specs.List(reqClient, &requestOpts)
	if err != nil {
		cblogger.Error(fmt.Sprintf("failed to get security group list, err : %s", err))
		return nil, err
	}

	specInfo := specs.VMSpecInfo{}
	for _, s := range *specList {
		if strings.EqualFold(specName, s.Name) {
			specInfo = s
			break
		}
	}

	// VM Spec 정보가 없을 경우 에러 리턴
	if specInfo.Id == "" {
		cblogger.Error(fmt.Sprintf("failed to get image, err : %s", err))
		return nil, err
	}
	return &specInfo.Id, nil
}

// VNic 목록 조회
func ListVNic(authHeader map[string]string, reqClient *client.RestClient, vmId string) (*[]nic.VmNicInfo, error) {
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	vNicList, err := nic.List(reqClient, vmId, &requestOpts)
	if err != nil {
		return nil, err
	}
	return vNicList, nil
}
