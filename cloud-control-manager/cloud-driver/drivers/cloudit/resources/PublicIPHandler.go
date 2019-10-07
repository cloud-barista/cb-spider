package resources

import (
	"errors"
	"fmt"
	//cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/dna/adaptiveip"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	//"github.com/sirupsen/logrus"
	"strconv"
)

/*var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}*/

type ClouditPublicIPHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
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
	// @TODO: PublicIP 생성 요청 파라미터 정의 필요
	type PublicIPReqInfo struct {
		IP         string `json:"ip" required:"true"`
		Name       string `json:"name" required:"true"`
		PrivateIP  string `json:"privateIp" required:"true"` // PublicIP가 적용되는 VM의 Private IP
		Protection int    `json:"protection" required:"false"`
	}
	reqInfo := PublicIPReqInfo{
		IP:        availableIP.IP,
		Name:      publicIPReqInfo.Name,
		PrivateIP: publicIPReqInfo.Id,
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
		spew.Dump(publicIP)
		return irs.PublicIPInfo{Id: publicIP.IP, Name: publicIP.Name}, nil
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
		for i, publicIP := range *publicIPList {
			cblogger.Info("[" + strconv.Itoa(i) + "]")
			spew.Dump(publicIP)
		}
		return nil, nil
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
		spew.Dump(publicIP)
		return irs.PublicIPInfo{Id: publicIP.ID, Name: publicIP.Name}, nil
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
