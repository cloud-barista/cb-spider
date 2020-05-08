package resources

import (
	"fmt"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/specs"
	"strings"
)

// VM Spec 정보 조회
func GetVMSpecById(authHeader map[string]string, reqClient *client.RestClient, specId string) (*string, error) {
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
		if strings.EqualFold(specId, s.Id) {
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
