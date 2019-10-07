package resources

import (
	//cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/nic"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/iam/securitygroup"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	//"github.com/sirupsen/logrus"
	"strconv"
)

type ClouditNicHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func (nicHandler *ClouditNicHandler) CreateVNic(vNicReqInfo irs.VNicReqInfo) (irs.VNicInfo, error) {
	nicHandler.Client.TokenID = nicHandler.CredentialInfo.AuthToken
	authHeader := nicHandler.Client.AuthenticatedHeaders()

	// @TODO: NIC 생성 요청 파라미터 정의 필요
	type VNicReqInfo struct {
		SubnetAddr string                             `json:"subnetAddr" required:"true"`
		VmId       string                             `json:"vmId" required:"true"`
		Type       string                             `json:"type" required:"true"`
		Secgroups  []securitygroup.SecurityGroupRules `json:"secgroups" required:"true"`
		IP         string                             `json:"ip" required:"true"`
	}
	reqInfo := VNicReqInfo{
		SubnetAddr: "10.0.8.0",
		VmId:       "025e5edc-54ad-4b98-9292-6eeca4c36a6d",
		Type:       "INTERNAL",
		Secgroups: []securitygroup.SecurityGroupRules{
			{
				ID: "b2be62e7-fd29-43ff-b008-08ae736e092a",
			},
		},
		IP: "",
	}

	createOpts := client.RequestOpts{
		MoreHeaders: authHeader,
		JSONBody:    reqInfo,
	}
	if nic, err := nic.Create(nicHandler.Client, reqInfo.VmId, &createOpts); err != nil {
		return irs.VNicInfo{}, err
	} else {
		spew.Dump(nic)
		return irs.VNicInfo{Id: nic.Mac}, nil
	}
}

func (nicHandler *ClouditNicHandler) ListVNic() ([]*irs.VNicInfo, error) {
	nicHandler.Client.TokenID = nicHandler.CredentialInfo.AuthToken
	authHeader := nicHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	serverId := "025e5edc-54ad-4b98-9292-6eeca4c36a6d"
	if vNicList, err := nic.List(nicHandler.Client, serverId, &requestOpts); err != nil {
		return nil, err
	} else {
		for i, nic := range *vNicList {
			cblogger.Info("[" + strconv.Itoa(i) + "]")
			spew.Dump(nic)
		}
		return nil, nil
	}
}

func (nicHandler *ClouditNicHandler) GetVNic(vNicID string) (irs.VNicInfo, error) {
	nicHandler.Client.TokenID = nicHandler.CredentialInfo.AuthToken
	authHeader := nicHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	serverId := "025e5edc-54ad-4b98-9292-6eeca4c36a6d"
	if vNic, err := nic.Get(nicHandler.Client, serverId, vNicID, &requestOpts); err != nil {
		return irs.VNicInfo{}, err
	} else {
		spew.Dump(vNic)
		return irs.VNicInfo{Id: vNic.Mac}, nil
	}
}
func (nicHandler *ClouditNicHandler) DeleteVNic(vNicID string) (bool, error) {
	nicHandler.Client.TokenID = nicHandler.CredentialInfo.AuthToken
	authHeader := nicHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	serverId := "025e5edc-54ad-4b98-9292-6eeca4c36a6d"
	if err := nic.Delete(nicHandler.Client, serverId, vNicID, &requestOpts); err != nil {
		return false, err
	} else {
		return true, nil
	}
}
