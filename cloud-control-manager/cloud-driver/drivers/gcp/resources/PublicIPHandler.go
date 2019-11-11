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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	compute "google.golang.org/api/compute/v1"
)

type GCPPublicIPHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

//GCP에서 PublicIP를 변경하려 할때 deleteaccessConfig => addAccessconfig 이때 넣어줘야 할 값은
// natIp, NetworkTier 이 2개를 추가 해 줘야 하며
// 추가 또는 삭제 시에는 networkInterface Name, zone, instananceName, projectId, accessConfig Name 등을 알아야 한다.

func (publicIpHandler *GCPPublicIPHandler) CreatePublicIP(publicIPReqInfo irs.PublicIPReqInfo) (irs.PublicIPInfo, error) {
	cblogger.Info(publicIPReqInfo)

	projectID := publicIpHandler.Credential.ProjectID
	region := publicIpHandler.Region.Region
	publicIpName := publicIPReqInfo.Name
	address := &compute.Address{
		Name: publicIpName,
	}

	result, errInsert := publicIpHandler.Client.Addresses.Insert(projectID, region, address).Do()
	if errInsert != nil {
		cblogger.Error("PublicIp 생성 실패1!")
		cblogger.Error(errInsert)
		return irs.PublicIPInfo{}, errInsert
	}
	cblogger.Info("PublicIP 생성 요청 성공 - 정보 조회를 위해 3초간 대기")
	cblogger.Info(result)
	time.Sleep(time.Second * 3)

	publicIPInfo, err := publicIpHandler.GetPublicIP(publicIpName)
	if err != nil {
		cblogger.Error("PublicIp 생성 실패!")
		cblogger.Error(err)
		return irs.PublicIPInfo{}, err
	}
	cblogger.Info(publicIPInfo)

	return publicIPInfo, nil
}

func (publicIpHandler *GCPPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	projectID := publicIpHandler.Credential.ProjectID
	region := publicIpHandler.Region.Region

	list, err := publicIpHandler.Client.Addresses.List(projectID, region).Do()
	spew.Dump(list)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	var publicIpInfoArr []*irs.PublicIPInfo
	for _, item := range list.Items {
		var publicInfo irs.PublicIPInfo
		publicInfo.Name = item.Name
		publicInfo.PublicIP = item.Address
		publicInfo.Status = item.Status
		//publicInfo.KeyValueList = GetKeyValueList()
		if users := item.Users; users != nil {
			vmArr := strings.Split(users[0], "/")
			publicInfo.OwnedVMID = vmArr[len(vmArr)-1]
		}
		keyValueList := []irs.KeyValue{
			{"id", strconv.FormatUint(item.Id, 10)},
			{"creationTimestamp", item.CreationTimestamp},
			{"region", item.Region},
			{"selfLink", item.SelfLink},
			{"networkTier", item.NetworkTier},
			{"addressType", item.AddressType},
			{"kind", item.Kind},
		}
		publicInfo.KeyValueList = keyValueList

		publicIpInfoArr = append(publicIpInfoArr, &publicInfo)

	}
	return publicIpInfoArr, nil
}

func (publicIpHandler *GCPPublicIPHandler) GetPublicIP(publicIPID string) (irs.PublicIPInfo, error) {
	cblogger.Infof("publicIPID : [%s]", publicIPID)
	projectID := publicIpHandler.Credential.ProjectID
	region := publicIpHandler.Region.Region
	name := publicIPID // name or resource ID

	info, err := publicIpHandler.Client.Addresses.Get(projectID, region, name).Do()
	//cblogger.Info(info)
	spew.Dump(info)
	if err != nil {
		cblogger.Error("PublicIP 정보 조회 실패")
		cblogger.Error(err)
		return irs.PublicIPInfo{}, err
	}
	cblogger.Infof("PublicIP[%s] 정보 조회 API 응답 수신", publicIPID)

	//바인딩 하기위해 []byte로 변환 처리
	infoByte, err2 := info.MarshalJSON()
	cblogger.Info(infoByte)
	//spew.Dump(infoByte)
	if err2 != nil {
		cblogger.Error("JSON 변환 실패")
		cblogger.Error(err2)
		return irs.PublicIPInfo{}, err2
	}

	var publicInfo irs.PublicIPInfo
	var keyValueList []irs.KeyValue

	publicInfo.Name = info.Name
	publicInfo.PublicIP = info.Address
	if users := info.Users; users != nil {
		vmArr := strings.Split(users[0], "/")
		publicInfo.OwnedVMID = vmArr[len(vmArr)-1]
	}
	publicInfo.Status = info.Status

	// 구조체 안에 해당값을 바인딩해준다.
	var result map[string]interface{}

	json.Unmarshal(infoByte, &result)
	//spew.Dump(result)
	//cblogger.Info(result)

	keyValueList = GetKeyValueList(result)
	// for key, value := range result {
	// 	keyValueList = append(keyValueList, irs.KeyValue{key, value})
	// }

	publicInfo.KeyValueList = keyValueList
	return publicInfo, nil
}

func (publicIpHandler *GCPPublicIPHandler) DeletePublicIP(publicIPID string) (bool, error) {
	projectID := publicIpHandler.Credential.ProjectID
	region := publicIpHandler.Region.Region
	name := publicIPID // name or resource ID

	info, err := publicIpHandler.Client.Addresses.Delete(projectID, region, name).Do()
	if err != nil {
		cblogger.Error(err)
		return false, err
	}
	fmt.Println(info)

	return true, nil
}

// func (*GCPPublicIPHandler) mappingPublicIpInfo(infos []byte) (irs.PublicIPInfo, error) {
// 	var publicInfo irs.PublicIPInfo
// 	err := json.Unmarshal(infos, &publicInfo)

// 	return publicInfo
// }
