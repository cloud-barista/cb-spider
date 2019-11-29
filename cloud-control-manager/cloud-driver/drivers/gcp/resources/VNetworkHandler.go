// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// program by ysjeon@mz.co.kr, 2019.07.
// modify by devunet@mz.co.kr, 2019.11.

package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	compute "google.golang.org/api/compute/v1"

	"time"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type GCPVNetworkHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

func (vNetworkHandler *GCPVNetworkHandler) CreateVNetwork(vNetworkReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {

	projectID := vNetworkHandler.Credential.ProjectID
	region := vNetworkHandler.Region.Region
	name := GetCBDefaultVNetName()
	vNetInfo, errVnet := vNetworkHandler.Client.Networks.Get(projectID, name).Do()

	var cnt string
	spew.Dump(vNetInfo)
	if errVnet != nil {
		network := &compute.Network{
			Name: name,
			//Name:                  GetCBDefaultVNetName(),
			AutoCreateSubnetworks: true, // subnet 자동으로 생성됨

		}

		_, err := vNetworkHandler.Client.Networks.Insert(projectID, network).Do()
		if err != nil {
			cblogger.Error(err)
		}
		before_time := time.Now()
		time.Sleep(time.Second * 20)
		max_time := 120

		// loop --> 생성 check --> 생성 되었으면, break; 안됐으면 sleep 5초 -->
		// if(total sleep 120sec?) error

		for {
			newvNetInfo, errVnet := vNetworkHandler.Client.Networks.Get(projectID, name).Do()
			if errVnet != nil {
				time.Sleep(time.Second * 5)
				after_time := time.Now()
				diff := after_time.Sub(before_time)
				if int(diff.Seconds()) > max_time {
					cblogger.Error("max time 동안 vNet 정보가  조회가 안되어서 에러처리함")
					cblogger.Error(errVnet)
					return irs.VNetworkInfo{}, errVnet
				}
			} else {
				cnt = strconv.Itoa(len(newvNetInfo.Subnetworks) + 1)
				break
			}
		}

	} else {
		cnt = strconv.Itoa(len(vNetInfo.Subnetworks) + 1)
	}

	fmt.Println("CNT : ", cnt)

	subnetInfo, errSubnet := vNetworkHandler.GetVNetwork(vNetworkReqInfo.Name)

	if errSubnet == nil {
		spew.Dump(subnetInfo)
		cblogger.Error(errSubnet)
		return irs.VNetworkInfo{}, errors.New("Already Exist")
	}

	// vNetResult, _ := vNetworkHandler.ListVNetwork()

	networkUrl := "https://www.googleapis.com/compute/v1/projects/" + projectID + "/global/networks/" + name
	subnetWork := &compute.Subnetwork{
		Name:        vNetworkReqInfo.Name,
		IpCidrRange: "192.168." + cnt + ".0/24",
		Network:     networkUrl,
	}
	res, err := vNetworkHandler.Client.Subnetworks.Insert(projectID, region, subnetWork).Do()
	if err != nil {
		cblogger.Error(err)
		return irs.VNetworkInfo{}, err
	}
	cblogger.Info(res)

	//생성되는데 시간이 필요 함. 약 20초정도?

	info, err2 := vNetworkHandler.Client.Subnetworks.Get(projectID, region, vNetworkReqInfo.Name).Do()
	if err2 != nil {
		cblogger.Error(err2)
		return irs.VNetworkInfo{}, err2
	}
	networkInfo := irs.VNetworkInfo{
		Name:          info.Name,
		Id:            strconv.FormatUint(info.Id, 10),
		AddressPrefix: info.IpCidrRange,
		KeyValueList: []irs.KeyValue{
			{"SubnetId", info.Name},
			{"Region", info.Region},
			{"GatewayAddress", info.GatewayAddress},
			{"SelfLink", info.SelfLink},
		},
	}

	return networkInfo, nil
}

func (vNetworkHandler *GCPVNetworkHandler) ListVNetwork() ([]*irs.VNetworkInfo, error) {
	projectID := vNetworkHandler.Credential.ProjectID
	region := vNetworkHandler.Region.Region

	vNetworkList, err := vNetworkHandler.Client.Subnetworks.List(projectID, region).Do()
	if err != nil {

		return nil, err
	}
	var vNetworkInfo []*irs.VNetworkInfo
	for _, item := range vNetworkList.Items {
		networkInfo := irs.VNetworkInfo{
			Name:          item.Name,
			Id:            strconv.FormatUint(item.Id, 10),
			AddressPrefix: item.IpCidrRange,
			KeyValueList: []irs.KeyValue{
				{"SubnetId", item.Name},
				{"Region", item.Region},
				{"GatewayAddress", item.GatewayAddress},
				{"SelfLink", item.SelfLink},
			},
		}

		vNetworkInfo = append(vNetworkInfo, &networkInfo)

	}

	return vNetworkInfo, nil
}

func (vNetworkHandler *GCPVNetworkHandler) GetVNetwork(vNetworkID string) (irs.VNetworkInfo, error) {

	projectID := vNetworkHandler.Credential.ProjectID
	region := vNetworkHandler.Region.Region
	//name := vNetworkID
	name := GetCBDefaultVNetName()
	cblogger.Infof("Name : [%s]", name)
	info, err := vNetworkHandler.Client.Subnetworks.Get(projectID, region, vNetworkID).Do()
	if err != nil {
		cblogger.Error(err)
		return irs.VNetworkInfo{}, err
	}

	networkInfo := irs.VNetworkInfo{
		Name:          info.Name,
		Id:            strconv.FormatUint(info.Id, 10),
		AddressPrefix: info.IpCidrRange,
		KeyValueList: []irs.KeyValue{
			{"SubnetId", info.Name},
			{"Region", info.Region},
			{"GatewayAddress", info.GatewayAddress},
			{"SelfLink", info.SelfLink},
		},
	}

	return networkInfo, nil
}

func (vNetworkHandler *GCPVNetworkHandler) DeleteVNetwork(vNetworkID string) (bool, error) {
	projectID := vNetworkHandler.Credential.ProjectID
	region := vNetworkHandler.Region.Region
	//name := vNetworkID
	name := GetCBDefaultVNetName()
	cblogger.Infof("Name : [%s]", name)
	info, err := vNetworkHandler.Client.Subnetworks.Delete(projectID, region, vNetworkID).Do()
	cblogger.Info(info)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}
	//fmt.Println(info)
	return true, nil
}
