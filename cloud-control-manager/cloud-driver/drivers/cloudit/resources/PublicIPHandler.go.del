package resources

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/dna/adaptiveip"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type ClouditPublicIPHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func setterIP(adaptiveip adaptiveip.AdaptiveIPInfo) *irs.PublicIPInfo {
	publicIP := &irs.PublicIPInfo{
		Name:         adaptiveip.IP,
		PublicIP:     adaptiveip.IP,
		OwnedVMID:    adaptiveip.VmName,
		Status:       adaptiveip.State,
		KeyValueList: []irs.KeyValue{{Key: "Name", Value: adaptiveip.Name}},
	}
	return publicIP
}

func (publicIPHandler *ClouditPublicIPHandler) CreatePublicIP(publicIPReqInfo irs.PublicIPReqInfo) (irs.PublicIPInfo, error) {
	publicIPHandler.Client.TokenID = publicIPHandler.CredentialInfo.AuthToken
	authHeader := publicIPHandler.Client.AuthenticatedHeaders()

	var availableIP adaptiveip.IPInfo

	// 1. 사용 가능한 PublicIP 목록 가져오기
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	if availableIPList, err := adaptiveip.ListAvailableIP(publicIPHandler.Client, &requestOpts); err != nil {
		return irs.PublicIPInfo{}, err
	} else {
		if len(*availableIPList) == 0 {
			allocateErr := errors.New(fmt.Sprintf("There is no PublicIPs to allocate"))
			return irs.PublicIPInfo{}, allocateErr
		} else {
			availableIP = (*availableIPList)[0]
		}
	}

	// 2. PublicIP 생성 및 할당
	reqInfo := adaptiveip.PublicIPReqInfo{
		IP:   availableIP.IP,
		Name: publicIPReqInfo.Name,
	}
	// VM PrivateIP 값 설정
	for _, meta := range publicIPReqInfo.KeyValueList {
		if meta.Key == "PrivateIP" {
			reqInfo.PrivateIP = meta.Value
		}
	}

	createOpts := client.RequestOpts{
		JSONBody:    reqInfo,
		MoreHeaders: authHeader,
	}
	publicIP, err := adaptiveip.Create(publicIPHandler.Client, &createOpts)
	if err != nil {
		cblogger.Error(err)
		return irs.PublicIPInfo{}, err
	} else {
		publicIPInfo := setterIP(*publicIP)
		return *publicIPInfo, nil
	}
}

func (publicIPHandler *ClouditPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	publicIPHandler.Client.TokenID = publicIPHandler.CredentialInfo.AuthToken
	authHeader := publicIPHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	publicIPList, err := adaptiveip.List(publicIPHandler.Client, &requestOpts)
	if err != nil {
		return nil, err
	} else {
		var resultList []*irs.PublicIPInfo

		for _, publicIP := range *publicIPList {
			publicIPInfo := setterIP(publicIP)
			resultList = append(resultList, publicIPInfo)
		}
		return resultList, nil
	}
}

func (publicIPHandler *ClouditPublicIPHandler) GetPublicIP(publicIPID string) (irs.PublicIPInfo, error) {
	publicIPHandler.Client.TokenID = publicIPHandler.CredentialInfo.AuthToken
	authHeader := publicIPHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if publicIP, err := adaptiveip.Get(publicIPHandler.Client, publicIPID, &requestOpts); err != nil {
		return irs.PublicIPInfo{}, err
	} else {
		publicIPInfo := setterIP(*publicIP)
		return *publicIPInfo, nil
	}
}

func (publicIPHandler *ClouditPublicIPHandler) DeletePublicIP(publicIPID string) (bool, error) {
	publicIPHandler.Client.TokenID = publicIPHandler.CredentialInfo.AuthToken
	authHeader := publicIPHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := adaptiveip.Delete(publicIPHandler.Client, publicIPID, &requestOpts); err != nil {
		return false, err
	} else {
		return true, nil
	}
}
