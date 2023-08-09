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
	"strings"

	compute "google.golang.org/api/compute/v1"

	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type GCPVPCHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

// @TODO : VPC 생성 로직 변경 필요 / 서브넷이 백그라운드로 생성되기 때문에 조회 시 모두 생성될 때까지 대기하는 로직 필요(그렇지 않으면 일부 정보가 누락됨)
func (vVPCHandler *GCPVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger.Info(vpcReqInfo)

	if vpcReqInfo.IId.NameId == "" {
		cblogger.Infof("생성할 VPC 이름[%s]이 없습니다.", vpcReqInfo.IId.NameId)
		return irs.VPCInfo{}, errors.New("Invalid Request - VPC NameId is required.")
	}

	if vpcReqInfo.SubnetInfoList == nil {
		cblogger.Info("생성할 VPC의 Subnet 정보가 없습니다.")
		return irs.VPCInfo{}, errors.New("Invalid Request - Subnet information is required.")
	}

	cblogger.Infof("생성된 [%s] VPC가 있는지 체크", vpcReqInfo.IId.NameId)
	_, errChkVpc := vVPCHandler.GetVPC(irs.IID{SystemId: vpcReqInfo.IId.NameId})
	if errChkVpc == nil {
		cblogger.Infof("이미 [%s] VPCs가 존재함.", vpcReqInfo.IId.NameId)
		return irs.VPCInfo{}, errors.New("Already Exist - " + vpcReqInfo.IId.NameId)
	}

	projectID := vVPCHandler.Credential.ProjectID
	region := vVPCHandler.Region.Region
	name := vpcReqInfo.IId.NameId

	autoCreateSubnetworks := false // VPC에 서브넷을 자동으로 생성하지 않도록 함.

	network := &compute.Network{
		Name: name,
		//Name:                  GetCBDefaultVNetName(),
		AutoCreateSubnetworks: autoCreateSubnetworks, // subnet 유무에 따라서 달라짐 true, false
		ForceSendFields:       []string{"AutoCreateSubnetworks"},
	}

	cblogger.Infof("[%s] VPC 생성 시작", name)
	cblogger.Info(network)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vVPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: vpcReqInfo.IId.NameId,
		CloudOSAPI:   "Insert()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	req, err := vVPCHandler.Client.Networks.Insert(projectID, network).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		cblogger.Errorf("[%s] VPC 생성 실패", name)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()

		callogger.Info(call.String(callLogInfo))
		return irs.VPCInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Infof("[%s] VPC가 정상적으로 생성되고 있습니다 - 리소스 ID : [%d]", name, req.Id)
	errWait := vVPCHandler.WaitUntilComplete(strconv.FormatUint(req.Id, 10), true)
	if errWait != nil {
		cblogger.Errorf("[%s] VPC 생성 완료 대기 실패", name)
		cblogger.Error(errWait)
		return irs.VPCInfo{}, errWait
	}

	/*
		//VPC를 생성하면 생성된 정보가 조회되기전까지는 다른 작업을 할 수 없기 때문에 대기함.
		errChkVpcStatus := vVPCHandler.WaitForRunVpc(name, true)
		if errChkVpcStatus != nil {
			cblogger.Errorf("생성된 VPC[%s] 정보 조회 실패", name)
			return irs.VPCInfo{}, errChkVpcStatus
		}
	*/

	//서브넷 생성
	vpcNetworkUrl := "https://www.googleapis.com/compute/v1/projects/" + projectID + "/global/networks/" + vpcReqInfo.IId.NameId
	for _, item := range vpcReqInfo.SubnetInfoList {
		subnetName := item.IId.NameId
		cblogger.Infof("생성할 [%s] Subnet이 존재하는지 체크", subnetName)
		checkInfo, err := vVPCHandler.Client.Subnetworks.Get(projectID, region, subnetName).Do()
		if err == nil {
			cblogger.Errorf("이미 [%s] Subnet이 존재함", subnetName)
			return irs.VPCInfo{}, errors.New("Already Exist - " + subnetName + " Subnet is exist")
		}
		cblogger.Info(" Subnet info : ", checkInfo)

		//서브네 생성
		subnetWork := &compute.Subnetwork{
			Name:        subnetName,
			IpCidrRange: item.IPv4_CIDR,
			Network:     vpcNetworkUrl,
		}
		cblogger.Infof("[%s] Subnet 생성시작", subnetName)
		cblogger.Info(subnetWork)

		infoSubnet, errSubnet := vVPCHandler.Client.Subnetworks.Insert(projectID, region, subnetWork).Do()
		if errSubnet != nil {
			cblogger.Error(errSubnet)
			return irs.VPCInfo{}, errors.New("Making Subnet Error - " + subnetName)
		}

		spew.Dump(infoSubnet)
		//생성된 서브넷이 조회되는데 시간이 필요하기 때문에 홀딩 함.
		/*
			errChkSubnetStatus := vVPCHandler.WaitForRunSubnet(subnetName, true)
			if errChkSubnetStatus != nil {
				cblogger.Errorf("생성된 Subnet[%s] 정보 조회 실패", subnetName)
				return irs.VPCInfo{}, errChkSubnetStatus
			}
		*/
		cblogger.Infof("[%s] Subnet 생성 성공 - 리소스 ID : [%d]", subnetName, infoSubnet.Id)
		errWait := vVPCHandler.WaitUntilComplete(strconv.FormatUint(infoSubnet.Id, 10), false)
		if errWait != nil {
			cblogger.Errorf("[%s] Subnet 생성 완료 대기 실패", subnetName)
			cblogger.Error(errWait)
			return irs.VPCInfo{}, errWait
		}

		cblogger.Infof("[%s] Subnet 생성완료", subnetName)
		cblogger.Info(infoSubnet)
	}

	//최신 정보로 리턴 함.
	vpcInfo, errVPC := vVPCHandler.GetVPC(irs.IID{SystemId: vpcReqInfo.IId.NameId})
	if errVPC != nil {
		cblogger.Errorf("최종 생성된 [%s] VPC 정보 조회 실패", vpcReqInfo.IId.NameId)
		cblogger.Error(errVPC)
		return vpcInfo, errVPC
	}
	vpcInfo.IId.NameId = vpcReqInfo.IId.NameId

	return vpcInfo, nil
}

// VPC 정보가 조회될때까지 대기
// waitFound : true - 정보가 조회될때까지 대기(생성 시) / false - 정보가 조회되지 않을때까지 대기(삭제 시)
func (vVPCHandler *GCPVPCHandler) WaitForRunVpc(name string, waitFound bool) error {
	cblogger.Info("======> VPC 정보가 조회될때까지 대기함.")

	before_time := time.Now()
	max_time := 300 //최대 300초간 체크

	cblogger.Infof("VPC 정보 조회가 %v될 때까지 1초 텀으로 체크 시작", waitFound)
	for {
		cblogger.Infof("==> [%s] VPC 정보 조회", name)
		vpcInfo, errVnet := vVPCHandler.Client.Networks.Get(vVPCHandler.Credential.ProjectID, name).Do()
		//spew.Dump(vpcInfo)

		//============================
		//정보가 조회될때까지 대기
		//============================
		if waitFound {
			if errVnet != nil {
				cblogger.Errorf("==> [%s] VPC 정보 조회 실패", name)
				cblogger.Error(errVnet)

				time.Sleep(time.Second * 1)
				after_time := time.Now()
				diff := after_time.Sub(before_time)
				if int(diff.Seconds()) > max_time {
					cblogger.Errorf("[%d]초 동안 [%s] VPC 정보가 조회되지 않아서 강제로 종료함.", max_time, name)
					return errVnet
				}
			} else {
				cblogger.Infof("==> [%s] VPC 정보 조회 완료", name)
				spew.Dump(vpcInfo)
				//cblogger.Info(vpcInfo)
				return nil
			}
		} else {
			//============================
			//정보가 조회되지 않을때까지 대기
			//============================
			if errVnet == nil {
				cblogger.Errorf("==> [%s] VPC 정보 조회 성공", name)
				//cblogger.Info(vpcInfo)
				spew.Dump(vpcInfo)

				time.Sleep(time.Second * 1)
				after_time := time.Now()
				diff := after_time.Sub(before_time)
				if int(diff.Seconds()) > max_time {
					cblogger.Errorf("[%d]초를 기다려도 [%s] VPC 정보가 조회되고 있어서 Wait을 강제로 종료함.", max_time, name)
					return errors.New("300초를 기다려도 생성된 " + name + " VPC 정보가 조회되고 있어서 Wait을 강제로 종료함.")
				}
			} else {
				cblogger.Infof("==> [%s] VPC 정보가 사라졌음", name)
				return nil
			}
		} //end of if waitFound : 조회 옵션
	}

	return nil
}

// Subnet 정보가 조회될때까지 대기
// waitFound : true - 정보가 조회될때까지 대기(생성 시) / false - 정보가 조회되지 않을때까지 대기(삭제 시)
func (vVPCHandler *GCPVPCHandler) WaitForRunSubnet(subnetName string, waitFound bool) error {
	cblogger.Info("======> Subnet 정보가 조회될때까지 대기함.")

	before_time := time.Now()
	max_time := 300 //최대 300초간 체크

	cblogger.Infof("Subnet 정보 조회가 %v될 때까지 1초 텀으로 체크 시작", waitFound)
	for {
		cblogger.Infof("--> 생성된 [%s] Subnet이 존재하는지 체크", subnetName)
		chkInfo, err := vVPCHandler.Client.Subnetworks.Get(vVPCHandler.Credential.ProjectID, vVPCHandler.Region.Region, subnetName).Do()
		//spew.Dump(chkInfo)
		//============================
		//정보가 조회될때까지 대기
		//============================
		if waitFound {
			if err != nil {
				cblogger.Errorf("==> [%s] Subnet 정보 조회 실패", subnetName)
				spew.Dump(err)

				time.Sleep(time.Second * 1)
				after_time := time.Now()
				diff := after_time.Sub(before_time)
				if int(diff.Seconds()) > max_time {
					cblogger.Errorf("[%d]초 동안 [%s] Subnet 정보가 조회되지 않아서 강제로 종료함.", max_time, subnetName)
					return errors.New("생성된 Subnet 정보가 장시간 조회되지 않아서 강제로 종료함.)")
				}
			} else {
				cblogger.Infof("==> [%s] Subnet 정보 조회 완료", subnetName)
				//cblogger.Info(chkInfo)
				spew.Dump(chkInfo)
				return nil
			}
		} else {
			//============================
			//정보가 조회되지 않을때까지 대기
			//============================
			if err == nil {
				cblogger.Errorf("==> [%s] Subnet 정보 조회 성공", subnetName)
				//cblogger.Info(chkInfo)
				spew.Dump(chkInfo)

				time.Sleep(time.Second * 1)
				after_time := time.Now()
				diff := after_time.Sub(before_time)
				if int(diff.Seconds()) > max_time {
					cblogger.Errorf("[%d]초를 기다려도 [%s] Subnet 정보가 조회되고 있어서 Wait을 강제로 종료함.", max_time, subnetName)
					return errors.New("300초를 기다려도 생성된 " + subnetName + " Subnet 정보가 조회되고 있어서 Wait을 강제로 종료함.")
				}
			} else {
				spew.Dump(err)
				cblogger.Infof("==> [%s] Subnet 정보가 사라졌음", subnetName)
				return nil
			}
		} // end of if : 정보 조회 옵션
	}

	return nil
}

//https://cloud.google.com/compute/docs/reference/rest/v1/globalOperations/list
//GET https://compute.googleapis.com/compute/v1/projects/{project}/global/operations
//https://godoc.org/google.golang.org/api/compute/v1#GlobalOperationsGetCall.Do
//https://cloud.google.com/compute/docs/reference/rest/v1/globalOperations/list

// https://cloud.google.com/compute/docs/reference/rest/v1/globalOperations/get
//
// resourceId : API 호출후 받은 리소스 값
// VPC : 글로벌
// https://www.googleapis.com/compute/v1/projects/mcloud-barista2020/global/networks/cb-vpc-load-test
// Subnet : Regions
// https://www.googleapis.com/compute/v1/projects/mcloud-barista2020/regions/asia-northeast3/operations/operation-1590139586815-5a6393937274c-71aebdca-1574e4d7
// 404 에러 체크해서 global과 region 자동으로 처리 가능하니 필요하면 나중에 공통 유틸로 변경할 것
func (vVPCHandler *GCPVPCHandler) WaitUntilComplete(resourceId string, isGlobalAction bool) error {
	//compute.ZoneOperationsGetCall
	//chkInfo, err := vVPCHandler.Client.Subnetworks.Get(vVPCHandler.Credential.ProjectID, vVPCHandler.Region.Region, subnetName).Do()

	//project string, operation string
	project := vVPCHandler.Credential.ProjectID
	region := vVPCHandler.Region.Region
	//resourceId := ""

	before_time := time.Now()
	max_time := 300 //최대 300초간 체크

	var opSatus *compute.Operation
	var err error

	for {
		//opSatus, err := vVPCHandler.Client.GlobalOperations.Get(project, resourceId).Do()
		if isGlobalAction {
			opSatus, err = vVPCHandler.Client.GlobalOperations.Get(project, resourceId).Do()
		} else {
			opSatus, err = vVPCHandler.Client.RegionOperations.Get(project, region, resourceId).Do()
		}
		if err != nil {
			return err
		}
		cblogger.Infof("==> 상태 : 진행율 : [%d] / [%s]", opSatus.Progress, opSatus.Status)

		//PENDING, RUNNING, or DONE.
		//if (opSatus.Status == "RUNNING" || opSatus.Status == "DONE") && opSatus.Progress >= 100 {
		if opSatus.Status == "DONE" {
			cblogger.Info("요청 작업이 정상적으로 처리되어서 Wait을 종료합니다.")
			return nil
		}

		time.Sleep(time.Second * 1)
		after_time := time.Now()
		diff := after_time.Sub(before_time)
		if int(diff.Seconds()) > max_time {
			cblogger.Errorf("[%d]초 동안 리소스[%s]의 상태가 완료되지 않아서 Wait을 강제로 종료함.", max_time, resourceId)
			return errors.New("장시간 요청 작업이 완료되지 않아서 Wait을 강제로 종료함.)")
		}
	}

	return nil
}

func (vVPCHandler *GCPVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	projectID := vVPCHandler.Credential.ProjectID
	//region := vVPCHandler.Region.Region
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vVPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: "",
		CloudOSAPI:   "List()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	vpcList, err := vVPCHandler.Client.Networks.List(projectID).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()

		callogger.Info(call.String(callLogInfo))

		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	var vpcInfo []*irs.VPCInfo

	for _, item := range vpcList.Items {
		iId := irs.IID{
			NameId: item.Name,
			//SystemId: strconv.FormatUint(item.Id, 10),
			SystemId: item.Name,
		}
		subnetInfo, err := vVPCHandler.GetVPC(iId)
		if err != nil {
			cblogger.Error(err)
			return vpcInfo, err
		}

		vpcInfo = append(vpcInfo, &subnetInfo)

	}

	return vpcInfo, nil
}

func (vVPCHandler *GCPVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {

	projectID := vVPCHandler.Credential.ProjectID
	region := vVPCHandler.Region.Region
	//name := VPCID
	systemId := vpcIID.SystemId
	//name := vpcIID.NameId

	//cblogger.Infof("NameID : [%s] / SystemID : [%s]", name, systemId)
	cblogger.Infof("SystemID : [%s]", systemId)
	subnetInfoList := []irs.SubnetInfo{}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vVPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: vpcIID.SystemId,
		CloudOSAPI:   "Get()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	infoVPC, err := vVPCHandler.Client.Networks.Get(projectID, systemId).Do()
	//infoVPC, err := vVPCHandler.Client.Networks.Get(projectID, name).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()

		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.VPCInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))
	spew.Dump(infoVPC)
	if infoVPC.Subnetworks != nil {
		for _, item := range infoVPC.Subnetworks {
			str := strings.Split(item, "/")
			region = str[len(str)-3]
			subnet := str[len(str)-1]
			infoSubnet, err := vVPCHandler.Client.Subnetworks.Get(projectID, region, subnet).Do()
			if err != nil {
				cblogger.Error(err)
				return irs.VPCInfo{}, err
			}
			subnetInfoList = append(subnetInfoList, mappingSubnet(infoSubnet))
		}

	}

	networkInfo := irs.VPCInfo{
		IId: irs.IID{
			NameId: infoVPC.Name,
			//SystemId: strconv.FormatUint(infoVPC.Id, 10),
			SystemId: infoVPC.Name,
		},
		IPv4_CIDR:      "Not support IPv4_CIDR at GCP VPC",
		SubnetInfoList: subnetInfoList,
		KeyValueList: []irs.KeyValue{
			{"RoutingMode", infoVPC.RoutingConfig.RoutingMode},
			{"Description", infoVPC.Description},
			{"SelfLink", infoVPC.SelfLink},
		},
	}

	return networkInfo, nil
}

func mappingSubnet(subnet *compute.Subnetwork) irs.SubnetInfo {
	//str := subnet.SelfLink
	str := strings.Split(subnet.SelfLink, "/")
	subnetName := str[len(str)-1]
	regionStr := strings.Split(subnet.Region, "/")
	region := regionStr[len(regionStr)-1]
	subnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId: subnet.Name,
			//SystemId: strconv.FormatUint(subnet.Id, 10),
			SystemId: subnet.Name,
		},
		IPv4_CIDR: subnet.IpCidrRange,
		KeyValueList: []irs.KeyValue{
			{"region", region},
			{"subnet", subnetName},
		},
	}
	return subnetInfo
}

func (vVPCHandler *GCPVPCHandler) DeleteVPC(vpcID irs.IID) (bool, error) {
	projectID := vVPCHandler.Credential.ProjectID
	//region := vVPCHandler.Region.Region
	//name := VPCID
	//name := vpcID.NameId
	name := vpcID.SystemId
	cblogger.Infof("Name : [%s]", name)
	subnetInfo, subErr := vVPCHandler.GetVPC(vpcID)
	if subErr != nil {
		cblogger.Error(subErr)
		return false, subErr
	}
	if subnetInfo.SubnetInfoList != nil {
		for _, item := range subnetInfo.SubnetInfoList {
			for _, si := range item.KeyValueList {
				if si.Key == "region" {
					region := si.Value
					infoSubnet, infoSubErr := vVPCHandler.Client.Subnetworks.Delete(projectID, region, item.IId.NameId).Do()
					if infoSubErr != nil {
						//cblogger.Error(infoSubErr)
						return false, infoSubErr
					}
					cblogger.Info("Delete subnet result :", infoSubnet)
					//cblogger.Info("Subnet Deleting....wait 10seconds")
					//time.Sleep(time.Second * 10)

					//서브넷이 완전히 삭제될때까지 대기
					/*
						errChkSubnetStatus := vVPCHandler.WaitForRunSubnet(item.IId.NameId, false)
						if errChkSubnetStatus != nil {
							return false, errChkSubnetStatus
						}
					*/

					cblogger.Infof("[%s] Subnet 삭제 성공 - 리소스 ID : [%d]", item.IId.NameId, infoSubnet.Id)
					errWait := vVPCHandler.WaitUntilComplete(strconv.FormatUint(infoSubnet.Id, 10), false)
					if errWait != nil {
						cblogger.Errorf("[%s] Subnet 삭제 완료 대기 실패", item.IId.NameId)
						cblogger.Error(errWait)
						return false, errWait
					}

				}
			}
		}
	}
	//cblogger.Info("VPC Deleting....wait 15seconds")
	//time.Sleep(time.Second * 15)
	cblogger.Info("[NOW Delete VPC]")
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vVPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: vpcID.SystemId,
		CloudOSAPI:   "Delete()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	info, err := vVPCHandler.Client.Networks.Delete(projectID, name).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	//time.Sleep(time.Second * 15)
	cblogger.Info(info)
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		return false, err
	}

	//삭제될때까지 대기
	/*
		errChkVpcStatus := vVPCHandler.WaitForRunVpc(name, false)
		if errChkVpcStatus != nil {
			return false, errChkVpcStatus
		}
	*/

	cblogger.Infof("[%s] VPC가 최종 삭제될까지 대기 - 리소스 ID : [%d]", name)
	errChkVpcStatus := vVPCHandler.WaitUntilComplete(strconv.FormatUint(info.Id, 10), true)
	callogger.Info(call.String(callLogInfo))
	if errChkVpcStatus != nil {
		callLogInfo.ErrorMSG = errChkVpcStatus.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Errorf("[%s] Subnet 삭제 완료 대기 실패", name)
		cblogger.Error(errChkVpcStatus)
		return false, errChkVpcStatus
	}

	//cblog.Info(info)
	return true, nil
}

func (VPCHandler *GCPVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	cblogger.Infof("[%s] Subnet 추가 - CIDR : %s", subnetInfo.IId.NameId, subnetInfo.IPv4_CIDR)
	//resSubnet, errSubnet := VPCHandler.CreateSubnet(vpcIID.SystemId, subnetInfo)
	_, errSubnet := VPCHandler.CreateSubnet(vpcIID.SystemId, subnetInfo)
	if errSubnet != nil {
		cblogger.Error(errSubnet)
		return irs.VPCInfo{}, errSubnet
	}
	//cblogger.Debug(resSubnet)

	return VPCHandler.GetVPC(vpcIID)
	//return irs.VPCInfo{}, nil
}

// 리턴 값은 구현하지 않고 nil을 리턴함. - 현재 사용되는 곳이 없어서 시간상 누락 시킴.
func (vVPCHandler *GCPVPCHandler) CreateSubnet(vpcId string, reqSubnetInfo irs.SubnetInfo) (irs.SubnetInfo, error) {
	cblogger.Info(reqSubnetInfo)

	projectID := vVPCHandler.Credential.ProjectID
	region := vVPCHandler.Region.Region

	//서브넷 생성
	vpcNetworkUrl := "https://www.googleapis.com/compute/v1/projects/" + projectID + "/global/networks/" + vpcId
	subnetName := reqSubnetInfo.IId.NameId
	cblogger.Infof("생성할 [%s] Subnet이 존재하는지 체크", subnetName)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vVPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: reqSubnetInfo.IId.NameId,
		CloudOSAPI:   "CreateSubnet()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	checkInfo, err := vVPCHandler.Client.Subnetworks.Get(projectID, region, subnetName).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err == nil {
		callLogInfo.ErrorMSG = err.Error()
		cblogger.Errorf("이미 [%s] Subnet이 존재함", subnetName)
		callogger.Info(call.String(callLogInfo))
		return irs.SubnetInfo{}, errors.New("Already Exist - " + subnetName + " Subnet is exist")
	}
	callogger.Info(call.String(callLogInfo))
	cblogger.Info("Subnet info : ", checkInfo)

	//서브넷 생성
	subnetWork := &compute.Subnetwork{
		Name:        subnetName,
		IpCidrRange: reqSubnetInfo.IPv4_CIDR,
		Network:     vpcNetworkUrl,
	}
	cblogger.Infof("[%s] Subnet 생성시작", subnetName)
	cblogger.Info(subnetWork)

	infoSubnet, errSubnet := vVPCHandler.Client.Subnetworks.Insert(projectID, region, subnetWork).Do()
	if errSubnet != nil {
		cblogger.Error(errSubnet)
		return irs.SubnetInfo{}, errors.New("Making Subnet Error - " + subnetName)
	}

	spew.Dump(infoSubnet)
	//생성된 서브넷이 조회되는데 시간이 필요하기 때문에 홀딩 함.
	cblogger.Infof("[%s] Subnet 생성 성공 - 리소스 ID : [%d]", subnetName, infoSubnet.Id)
	errWait := vVPCHandler.WaitUntilComplete(strconv.FormatUint(infoSubnet.Id, 10), false)
	if errWait != nil {
		cblogger.Errorf("[%s] Subnet 생성 완료 대기 실패", subnetName)
		cblogger.Error(errWait)
		return irs.SubnetInfo{}, errWait
	}

	cblogger.Infof("[%s] Subnet 생성완료", subnetName)
	cblogger.Info(infoSubnet)

	//생성된 정보 조회
	//mappingSubnet() 이용하면 되지만 수정해야 함.

	return irs.SubnetInfo{}, nil
}

func (vVPCHandler *GCPVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Infof("[%s] VPC의 [%s] Subnet 삭제", vpcIID.SystemId, subnetIID.SystemId)

	projectID := vVPCHandler.Credential.ProjectID
	region := vVPCHandler.Region.Region

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vVPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: subnetIID.SystemId,
		CloudOSAPI:   "CreateVpc()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	infoSubnet, infoSubErr := vVPCHandler.Client.Subnetworks.Delete(projectID, region, subnetIID.SystemId).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if infoSubErr != nil {
		callLogInfo.ErrorMSG = infoSubErr.Error()

		callogger.Info(call.String(callLogInfo))
		cblogger.Error(infoSubErr)
		return false, infoSubErr
	}
	callogger.Info(call.String(callLogInfo))
	cblogger.Info("Delete subnet result :", infoSubnet)

	//서브넷이 완전히 삭제될때까지 대기
	cblogger.Infof("[%s] Subnet 삭제 성공 - 리소스 ID : [%d]", subnetIID.SystemId, infoSubnet.Id)
	errWait := vVPCHandler.WaitUntilComplete(strconv.FormatUint(infoSubnet.Id, 10), false)
	if errWait != nil {
		cblogger.Errorf("[%s] Subnet 삭제 완료 대기 실패", subnetIID.SystemId)
		cblogger.Error(errWait)
		return false, errWait
	}

	return true, nil
}
