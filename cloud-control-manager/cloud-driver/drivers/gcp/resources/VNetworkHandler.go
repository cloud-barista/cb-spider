package resources

import (
	"context"
	"fmt"
	"log"
	"strconv"

	compute "google.golang.org/api/compute/v1"

	"time"

	idrv "../../../interfaces"
	irs "../../../interfaces/resources"
)

//var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	//cblogger = cblog.GetLogger("AWS Connect")
}

type GCPVNetworkHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

func (vNetworkHandler *GCPVNetworkHandler) CreateVNetwork(vNetworkReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {
	// priject id
	projectID := vNetworkHandler.Credential.ProjectID
	name := vNetworkReqInfo.Name

	network := &compute.Network{
		Name:                  name,
		AutoCreateSubnetworks: true, // subnet 자동으로 생성됨
	}

	res, err := vNetworkHandler.Client.Networks.Insert(projectID, network).Do()
	if err != nil {
		log.Fatal(err)

	}
	fmt.Println(res)

	//생성되는데 시간이 필요 함. 약 20초정도?
	time.Sleep(time.Second * 20)
	info, err2 := vNetworkHandler.Client.Networks.Get(projectID, name).Do()
	if err2 != nil {
		log.Fatal(err2)
	}
	networkInfo := irs.VNetworkInfo{
		Name: info.Name,
		Id:   strconv.FormatUint(info.Id, 10),
		KeyValueList: []irs.KeyValue{
			{"SubnetId", info.Name},
		},
	}

	return networkInfo, nil
}

func (vNetworkHandler *GCPVNetworkHandler) ListVNetwork() ([]*irs.VNetworkInfo, error) {
	projectID := vNetworkHandler.Credential.ProjectID

	vNetworkList, err := vNetworkHandler.Client.Networks.List(projectID).Do()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	var vNetworkInfo []*irs.VNetworkInfo
	for _, item := range vNetworkList.Items {
		networkInfo := irs.VNetworkInfo{
			Name: item.Name,
			Id:   strconv.FormatUint(item.Id, 10),
			KeyValueList: []irs.KeyValue{
				{"SubnetId", item.Name},
			},
		}

		vNetworkInfo = append(vNetworkInfo, &networkInfo)

	}

	return vNetworkInfo, nil
}

func (vNetworkHandler *GCPVNetworkHandler) GetVNetwork(vNetworkID string) (irs.VNetworkInfo, error) {

	projectID := vNetworkHandler.Credential.ProjectID
	name := vNetworkID
	info, err := vNetworkHandler.Client.Networks.Get(projectID, name).Do()
	if err != nil {
		log.Fatal(err)
	}

	networkInfo := irs.VNetworkInfo{
		Name: info.Name,
		Id:   strconv.FormatUint(info.Id, 10),
		KeyValueList: []irs.KeyValue{
			{"SubnetId", info.Name},
		},
	}

	return networkInfo, nil
}

func (vNetworkHandler *GCPVNetworkHandler) DeleteVNetwork(vNetworkID string) (bool, error) {
	projectID := vNetworkHandler.Credential.ProjectID
	name := vNetworkID
	info, err := vNetworkHandler.Client.Networks.Delete(projectID, name).Do()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(info)
	return true, nil
}
