package resources

import (
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/nic"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/iam/securitygroup"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type ClouditNicHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func setterNic(nic nic.VmNicInfo) *irs.VNicInfo {
	vNicInfo := &irs.VNicInfo{
		Name:       nic.VmName,
		PublicIP:   nic.AdaptiveIp,
		MacAddress: nic.Mac,
		OwnedVMID:  nic.VmId,
		Status:     nic.State,
	}
	return vNicInfo
}

func (nicHandler *ClouditNicHandler) CreateVNic(vNicReqInfo irs.VNicReqInfo) (irs.VNicInfo, error) {
	nicHandler.Client.TokenID = nicHandler.CredentialInfo.AuthToken
	authHeader := nicHandler.Client.AuthenticatedHeaders()

	reqInfo := nic.VNicReqInfo{
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
		return irs.VNicInfo{Name: nic.Mac}, nil
	}
}

func (nicHandler *ClouditNicHandler) ListVNic() ([]*irs.VNicInfo, error) {
	nicHandler.Client.TokenID = nicHandler.CredentialInfo.AuthToken
	authHeader := nicHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	// TODO: serverId 인터페이스 정의
	serverId := "025e5edc-54ad-4b98-9292-6eeca4c36a6d"
	if vNicList, err := nic.List(nicHandler.Client, serverId, &requestOpts); err != nil {
		return nil, err
	} else {
		var resultList []*irs.VNicInfo
		for _, nic := range *vNicList {
			vNicInfo := setterNic(nic)
			resultList = append(resultList, vNicInfo)
		}
		return resultList, nil
	}
}

func (nicHandler *ClouditNicHandler) GetVNic(vNicID string) (irs.VNicInfo, error) {
	nicHandler.Client.TokenID = nicHandler.CredentialInfo.AuthToken
	authHeader := nicHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	// TODO: serverId 인터페이스 정의
	serverId := "025e5edc-54ad-4b98-9292-6eeca4c36a6d"
	if vNic, err := nic.Get(nicHandler.Client, serverId, vNicID, &requestOpts); err != nil {
		return irs.VNicInfo{}, err
	} else {
		spew.Dump(vNic)
		return irs.VNicInfo{Name: vNic.Mac}, nil
	}
}

func (nicHandler *ClouditNicHandler) DeleteVNic(vNicID string) (bool, error) {
	nicHandler.Client.TokenID = nicHandler.CredentialInfo.AuthToken
	authHeader := nicHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	// TODO: serverId 인터페이스 정의
	serverId := "025e5edc-54ad-4b98-9292-6eeca4c36a6d"
	if err := nic.Delete(nicHandler.Client, serverId, vNicID, &requestOpts); err != nil {
		return false, err
	} else {
		return true, nil
	}
}
