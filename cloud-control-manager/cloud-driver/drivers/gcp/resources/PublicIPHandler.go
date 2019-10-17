package resources

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	idrv "../../../interfaces"
	nirs "../../../interfaces/new-resources"
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

func (publicIpHandler *GCPPublicIPHandler) CreatePublicIP(publicIPReqInfo nirs.PublicIPReqInfo) (nirs.PublicIPInfo, error) {
	projectID := publicIpHandler.Credential.projectID
	region := publicIpHandler.Region.region
	address := &compute.Address{
		Name: publicIPReqInfo.Name,
	}
	publicIpHandler.Client.Addresses.Insert(projectID, region, address).Do()
	return publicIPInfo, nil
}

func (publicIpHandler *GCPPublicIPHandler) ListPublicIP() ([]*nirs.PublicIPInfo, error) {
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

func (publicIpHandler *GCPPublicIPHandler) GetPublicIP(publicIPID string) (nirs.PublicIPInfo, error) {
	projectID := publicIpHandler.Credential.projectID
	region := publicIpHandler.Region.region
	name := publicIPID

	info, err := publicIpHandler.Client.Addresses.Get(projectID, region, name).Do()
	if err != nil {
		log.Fatal(err)
	}

	//바인딩 하기위해 []byte로 변환 처리
	infoByte, err := info.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}

	var publicInfo nirs.PublicIPInfo

	// 구조체 안에 해당값을 바인딩해준다.
	err := json.Unmarshal(infoByte, &publicInfo)
	if users := info.Users; users != nil {
		vmArr := strings.Split(users, "/")
		&publicInfo.InstanceId = vmArr[len(vmArr)-1]
	}

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
