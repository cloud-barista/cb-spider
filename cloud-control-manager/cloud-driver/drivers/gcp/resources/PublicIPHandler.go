package resources

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	idrv "../../../interfaces"
	irs "../../../interfaces/resources"
	compute "google.golang.org/api/compute/v1"
)

type GCPPublicIPHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

// @TODO: PublicIP 리소스 프로퍼티 정의 필요
type PublicIPInfo struct {
	Id                string
	Name              string
	Region            string // GCP
	CreationTimestamp string // GCP
	Address           string // GCP
	NetworkTier       string // GCP : PREMIUM, STANDARD
	AddressType       string // GCP : External, INTERNAL, UNSPECIFIED_TYPE
	Status            string // GCP : IN_USE, RESERVED, RESERVING
	InstanceId        string // GCP : 연결된 VM
}

func (publicIpHandler *GCPPublicIPHandler) CreatePublicIP(publicIPReqInfo irs.PublicIPReqInfo) (irs.PublicIPInfo, error) {

	return publicIPInfo, nil
}

func (publicIpHandler *GCPPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	projectID := publicIpHandler.Credential.projectID
	region := publicIpHandler.Region.region

	list, err := publicIpHandler.Client.Addresses.List(projectID, region).Do()
	if err != nil {
		log.Fatal(err)
	}
	for _, item := range list.Items {

	}
	return nil, nil
}

func (publicIpHandler *GCPPublicIPHandler) GetPublicIP(publicIPID string) (irs.PublicIPInfo, error) {
	projectID := publicIpHandler.Credential.projectID
	region := publicIpHandler.Region.region
	name := publicIPID
	info, err := publicIpHandler.Client.Addresses.Get(projectID, region, name).Do()
	if err != nil {
		log.Fatal(err)
	}
	infoByte, err := info.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}

	var publicInfo irs.PublicIPInfo

	err := json.Unmarshal(infoByte, &publicInfo)
	users := info.Users[0]
	vmArr := strings.Split(users, "/")
	&publicInfo.InstanceId = vmArr[len(vmArr)-1]
	if err != nil {
		log.Fatal(err)
	}

	return publicInfo, err
}

func (publicIpHandler *GCPPublicIPHandler) DeletePublicIP(publicIPID string) (bool, error) {

	return true, nil
}

func (*GCPPublicIPHandler) mappingPublicIpInfo(infos []byte) (irs.PublicIPInfo, error) {
	var publicInfo irs.PublicIPInfo
	err := json.Unmarshal(infos, &publicInfo)

	return publicInfo
}
