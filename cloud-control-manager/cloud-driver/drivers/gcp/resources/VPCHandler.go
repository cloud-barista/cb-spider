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
	"strconv"

	compute "google.golang.org/api/compute/v1"

	"time"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type GCPVHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

func (vVPCHandler *GCPVPCHandler) CreateVNetwork(vNetworkReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {
	cblogger.Info(vNetworkReqInfo)

	var cnt string
	isFirst := false

	projectID := vVPCHandler.Credential.ProjectID
	region := vVPCHandler.Region.Region
	name := GetCBDefaultVNetName()

	cblogger.Infof("생성된 [%s] VNetwork가 있는지 체크", name)
	vNetInfo, errVnet := vVPCHandler.Client.Networks.Get(projectID, name).Do()
	spew.Dump(vNetInfo)
	if errVnet != nil {
		isFirst = true
		cblogger.Error(errVnet)

		cblogger.Infof("존재하는 [%s] VNetwork가 없으므로 새로 생성해야 함", name)
		network := &compute.Network{
			Name: name,
			//Name:                  GetCBDefaultVNetName(),
			AutoCreateSubnetworks: true, // subnet 자동으로 생성됨
		}

		cblogger.Infof("[%s] VNetwork 생성 시작", name)
		cblogger.Info(network)
		_, err := vVPCHandler.Client.Networks.Insert(projectID, network).Do()
		if err != nil {
			cblogger.Errorf("[%s] VNetwork 생성 실패", name)
			cblogger.Error(err)
			return irs.VNetworkInfo{}, errVnet
		}

		cblogger.Infof("[%s] VNetwork 정상적으로 생성되고 있습니다.", name)
		before_time := time.Now()
		max_time := 120

		// loop --> 생성 check --> 생성 되었으면, break; 안됐으면 sleep 5초 -->
		// if(total sleep 120sec?) error

		cblogger.Info("VNetwork가 모두 생성될 때까지 5초 텀으로 체크 시작")
		for {
			cblogger.Infof("==> [%s] VNetwork 정보 조회", name)
			_, errVnet := vVPCHandler.Client.Networks.Get(projectID, name).Do()
			if errVnet != nil {
				cblogger.Errorf("==> [%s] VNetwork 정보 조회 실패", name)
				cblogger.Error(errVnet)

				time.Sleep(time.Second * 5)
				after_time := time.Now()
				diff := after_time.Sub(before_time)
				if int(diff.Seconds()) > max_time {
					cblogger.Errorf("[%d]초 동안 [%s] VNetwork 정보가 조회되지 않아서 강제로 종료함.", max_time, name)
					return irs.VNetworkInfo{}, errVnet
				}
			} else {
				//생성된 VPC와 서브넷 이름이 동일하지 않으면 VPC의 기본 서브넷이 모두 생성될 때까지 20초 정도 대기
				//if name != vNetworkReqInfo.Name {
				cblogger.Info("생성된 VNetwork정보가 조회되어도 리전에서는 계속 생성되고 있기 때문에 20초 대기")
				time.Sleep(time.Second * 20)
				//}

				cblogger.Infof("==> [%s] VNetwork 정보 생성 완료", name)
				//서브넷이 비동기로 생성되고 있기 때문에 다시 체크해야 함.
				newvNetInfo, _ := vVPCHandler.Client.Networks.Get(projectID, name).Do()
				cnt = strconv.Itoa(len(newvNetInfo.Subnetworks) + 1)
				break
			}
		}
	} else {
		cblogger.Infof("이미 [%s] VNetworks가 존재함.", name)
		cnt = strconv.Itoa(len(vNetInfo.Subnetworks) + 1)
	}

	cblogger.Info("현재 생성된 서브넷 수 : ", cnt)
	cblogger.Infof("생성할 [%s] Subnet이 존재하는지 체크", vNetworkReqInfo.Name)

	subnetInfo, errSubnet := vVPCHandler.GetVNetwork(vNetworkReqInfo.Name)
	if errSubnet == nil {
		spew.Dump(subnetInfo)
		//최초 생성인 경우 VNetwork와 Subnet 이름이 동일하면 이미 생성되었으므로 추가로 생성하지 않고 리턴 함.
		if isFirst {
			cblogger.Error("최초 VNetwork 생성이므로 에러 없이 조회된 서브넷 정보를 리턴 함.")
			return subnetInfo, nil
		} else {
			cblogger.Error(errSubnet)
			return irs.VNetworkInfo{}, errors.New("Already Exist - " + vNetworkReqInfo.Name)
		}
	}

	// vNetResult, _ := vVPCHandler.ListVNetwork()

	networkUrl := "https://www.googleapis.com/compute/v1/projects/" + projectID + "/global/networks/" + name
	subnetWork := &compute.Subnetwork{
		Name:        vNetworkReqInfo.Name,
		IpCidrRange: "192.168." + cnt + ".0/24",
		Network:     networkUrl,
	}
	cblogger.Infof("[%s] Subnet 생성시작", vNetworkReqInfo.Name)
	cblogger.Info(subnetWork)
	res, err := vVPCHandler.Client.Subnetworks.Insert(projectID, region, subnetWork).Do()
	if err != nil {
		cblogger.Error("Subnet 생성 실패")
		cblogger.Error(err)
		return irs.VNetworkInfo{}, err
	}
	cblogger.Infof("[%s] Subnet 생성완료", vNetworkReqInfo.Name)
	cblogger.Info(res)

	//생성되는데 시간이 필요 함. 약 20초정도?
	//time.Sleep(time.Second * 20)

	info, err2 := vVPCHandler.Client.Subnetworks.Get(projectID, region, vNetworkReqInfo.Name).Do()
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

func (vVPCHandler *GCPVPCHandler) ListVNetwork() ([]*irs.VNetworkInfo, error) {
	projectID := vVPCHandler.Credential.ProjectID
	region := vVPCHandler.Region.Region

	vNetworkList, err := vVPCHandler.Client.Subnetworks.List(projectID, region).Do()
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

func (vVPCHandler *GCPVPCHandler) GetVNetwork(vNetworkID string) (irs.VNetworkInfo, error) {

	projectID := vVPCHandler.Credential.ProjectID
	region := vVPCHandler.Region.Region
	//name := vNetworkID
	name := GetCBDefaultVNetName()
	cblogger.Infof("Name : [%s] / Subnet : [%s]", name, vNetworkID)
	info, err := vVPCHandler.Client.Subnetworks.Get(projectID, region, vNetworkID).Do()
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

func (vVPCHandler *GCPVPCHandler) DeleteVNetwork(vNetworkID string) (bool, error) {
	projectID := vVPCHandler.Credential.ProjectID
	region := vVPCHandler.Region.Region
	//name := vNetworkID
	name := GetCBDefaultVNetName()
	cblogger.Infof("Name : [%s]", name)
	info, err := vVPCHandler.Client.Subnetworks.Delete(projectID, region, vNetworkID).Do()
	cblogger.Info(info)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}
	//fmt.Println(info)
	return true, nil
}
