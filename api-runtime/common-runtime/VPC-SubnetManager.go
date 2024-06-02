// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.

package commonruntime

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
	infostore "github.com/cloud-barista/cb-spider/info-store"
)

// ====================================================================
// type for GORM
type VPCIIDInfo FirstIIDInfo

func (VPCIIDInfo) TableName() string {
	return "vpc_iid_infos"
}

type SubnetIIDInfo ZoneLevelVPCDependentIIDInfo

func (SubnetIIDInfo) TableName() string {
	return "subnet_iid_infos"
}

//====================================================================

func init() {
	db, err := infostore.Open()
	if err != nil {
		cblog.Error(err)
		return
	}
	db.AutoMigrate(&VPCIIDInfo{})
	db.AutoMigrate(&SubnetIIDInfo{})
	infostore.Close(db)
}

//================ VPC Handler

// UserIID{UserID, CSP-ID} => SpiderIID{UserID, SP-XID:CSP-ID}
// (1) check existence(UserID)
// (2) get resource info(CSP-ID)
// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
// (4) insert spiderIID
func RegisterVPC(connectionName string, userIID cres.IID) (*cres.VPCInfo, error) {
	cblog.Info("call RegisterVPC()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	emptyPermissionList := []string{}

	err = ValidateStruct(userIID, emptyPermissionList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcSPLock.Lock(connectionName, userIID.NameId)
	defer vpcSPLock.Unlock(connectionName, userIID.NameId)

	// (1) check existence with NameId
	bool_ret, err := infostore.HasByConditions(&VPCIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, userIID.NameId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	rsType := rsVPC
	if bool_ret {
		err := fmt.Errorf(rsType + "-" + userIID.NameId + " already exists!")
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource info(CSP-ID)
	// check existence and get info of this resouce in the CSP
	// Do not use NameId, because Azure driver use it like SystemId
	getInfo, err := handler.GetVPC(cres.IID{NameId: userIID.SystemId, SystemId: userIID.SystemId})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
	//     ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	// Do not user NameId, because Azure driver use it like SystemId
	systemId := getMSShortID(getInfo.IId.SystemId)
	spiderIId := cres.IID{NameId: userIID.NameId, SystemId: systemId + ":" + getInfo.IId.SystemId}

	// (4) insert spiderIID
	// insert VPC SpiderIID to metadb
	err = infostore.Insert(&VPCIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// insert subnet's spiderIIDs to metadb and setup subnet IID for return info
	for count, subnetInfo := range getInfo.SubnetInfoList {
		// generate subnet's UserID
		subnetUserId := userIID.NameId + "-subnet-" + strconv.Itoa(count)

		// insert a subnet SpiderIID to metadb
		// Do not user NameId, because Azure driver use it like SystemId
		systemId := getMSShortID(subnetInfo.IId.SystemId)
		subnetSpiderIId := cres.IID{NameId: subnetUserId, SystemId: systemId + ":" + subnetInfo.IId.SystemId}
		err = infostore.Insert(&SubnetIIDInfo{ConnectionName: connectionName, NameId: subnetSpiderIId.NameId, SystemId: subnetSpiderIId.SystemId,
			OwnerVPCName: userIID.NameId})
		if err != nil {
			cblog.Error(err)
			return nil, err
		}

		// setup subnet IID for return info
		subnetInfo.IId = cres.IID{NameId: subnetUserId, SystemId: subnetInfo.IId.SystemId}
		getInfo.SubnetInfoList[count] = subnetInfo
	} // end of for _, info

	// set up VPC User IID for return info
	getInfo.IId = userIID

	return &getInfo, nil
}

// UserIID{UserID, CSP-ID} => SpiderIID{UserID, SP-XID:CSP-ID}
// (1) check existence(UserID)
// (2) get resource info(CSP-ID)
// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
// (4) insert spiderIID
func RegisterSubnet(connectionName string, zoneId string, vpcName string, userIID cres.IID) (*cres.VPCInfo, error) {
	cblog.Info("call RegisterSubnet()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcName, err = EmptyCheckAndTrim("vpcName", vpcName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	emptyPermissionList := []string{}

	err = ValidateStruct(userIID, emptyPermissionList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetZoneLevelCloudConnection(connectionName, zoneId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcSPLock.Lock(connectionName, userIID.NameId)
	defer vpcSPLock.Unlock(connectionName, userIID.NameId)

	// (1) check existence with NameId
	bool_ret, err := infostore.HasBy3Conditions(&SubnetIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, OWNER_VPC_NAME_COLUMN, vpcName, NAME_ID_COLUMN, userIID.NameId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	rsType := rsSubnet
	if bool_ret {
		err := fmt.Errorf(rsType + "-" + userIID.NameId + " already exists!")
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource info(CSP-ID)
	var iidInfo VPCIIDInfo
	err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, vpcName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource(driverIID)
	getInfo, err := handler.GetVPC(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
	//     ex) spiderIID {"subnet-01", "subnet-01-ck9s7jvds1k750hi2kkg:subnet-0daad7e3daa3a30f3"}
	// insert subnet's spiderIIDs to metadb and setup subnet IID for return info
	for count, subnetInfo := range getInfo.SubnetInfoList {
		// generate subnet's UserID
		subnetUserId := userIID.NameId
		// Do not use NameId, because Azure driver use it like SystemId
		systemId := getMSShortID(subnetInfo.IId.SystemId)
		if subnetInfo.IId.SystemId == userIID.SystemId {
			// insert a subnet SpiderIID to metadb
			subnetSpiderIId := cres.IID{NameId: subnetUserId, SystemId: systemId + ":" + subnetInfo.IId.SystemId}
			err = infostore.Insert(&SubnetIIDInfo{ConnectionName: connectionName, ZoneId: zoneId, NameId: subnetSpiderIId.NameId, SystemId: subnetSpiderIId.SystemId,
				OwnerVPCName: vpcName})
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			if subnetInfo.Zone == "" { // GCP has no Zone info
				var iidInfo SubnetIIDInfo
				err = infostore.GetBy3Conditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, subnetInfo.IId.NameId, OWNER_VPC_NAME_COLUMN, vpcName)
				if err != nil {
					cblog.Info(err)
				} else {
					subnetInfo.Zone = iidInfo.ZoneId
				}
			}

			// setup subnet IID for return info
			subnetInfo.IId = cres.IID{NameId: subnetUserId, SystemId: subnetInfo.IId.SystemId}
			getInfo.SubnetInfoList[count] = subnetInfo
		} // end of if subnetInfo.IId.SystemId == userIID.SystemId
	} // end of for _, info

	// set up VPC User IID for return info
	getInfo.IId = makeUserIID(iidInfo.NameId, iidInfo.SystemId)

	return &getInfo, nil
}

func UnregisterSubnet(connectionName string, vpcName string, nameId string) (bool, error) {
	cblog.Info("call UnregisterSubnet()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	vpcName, err = EmptyCheckAndTrim("vpcName", vpcName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	nameId, err = EmptyCheckAndTrim("nameId", nameId)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	vpcSPLock.Lock(connectionName, nameId)
	defer vpcSPLock.Unlock(connectionName, nameId)

	// (1) check existence with NameId
	bool_ret, err := infostore.HasBy3Conditions(&SubnetIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, OWNER_VPC_NAME_COLUMN, vpcName, NAME_ID_COLUMN, nameId)
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	rsType := rsSubnet
	if !bool_ret {
		err := fmt.Errorf("The " + rsType + "-" + nameId + " in " + vpcName + " does not exist!")
		cblog.Error(err)
		return false, err
	}

	// (2) delete subnet's spiderIIDs from metadb
	_, err = infostore.DeleteBy3Conditions(&SubnetIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, OWNER_VPC_NAME_COLUMN, vpcName, NAME_ID_COLUMN, nameId)
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	return true, nil
}

// (1) check exist(NameID)
// (2) generate SP-XID and create reqIID, driverIID
// (3) create Resource
// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
// (5) insert spiderIID
// (6) create userIID

type SubnetReqZoneInfo struct {
	IId  cres.IID
	Zone string
}

func CreateVPC(connectionName string, rsType string, reqInfo cres.VPCReqInfo, IDTransformMode string) (*cres.VPCInfo, error) {
	cblog.Info("call CreateVPC()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	emptyPermissionList := []string{
		"resources.IID:SystemId",
		"resources.VPCReqInfo:IPv4_CIDR", // because can be unused in some VPC
		"resources.SubnetInfo:Zone",      // because can be unused in some Zone
		"resources.KeyValue:Key",         // because unusing key-value list
		"resources.KeyValue:Value",       // because unusing key-value list
	}

	err = ValidateStruct(reqInfo, emptyPermissionList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcSPLock.Lock(connectionName, reqInfo.IId.NameId)
	defer vpcSPLock.Unlock(connectionName, reqInfo.IId.NameId)

	// (1) check existence with NameId
	bool_ret, err := infostore.HasByConditions(&VPCIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, reqInfo.IId.NameId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if bool_ret {
		err := fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
		cblog.Error(err)
		return nil, err
	}

	// check the Cloud Connection has the VPC already, when the CSP supports only 1 VPC.
	drv, err := ccm.GetCloudDriver(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if drv.GetDriverCapability().SINGLE_VPC {
		var vpcIIDInfoList []*VPCIIDInfo
		err := infostore.ListByCondition(&vpcIIDInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		if len(vpcIIDInfoList) > 0 {
			err := fmt.Errorf(rsType + "-" + connectionName + " can have only 1 VPC, but already have a VPC " + vpcIIDInfoList[0].NameId)
			cblog.Error(err)
			return nil, err
		}
	}

	spUUID := ""
	if GetID_MGMT(IDTransformMode) == "ON" { // Use IID Management
		// (2) generate SP-XID and create reqIID, driverIID
		//     ex) SP-XID {"vm-01-9m4e2mr0ui3e8a215n4g"}
		//
		//     create reqIID: {reqNameID, reqSystemID}   # reqSystemID=SP-XID
		//         ex) reqIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g"}
		//
		//     create driverIID: {driverNameID, driverSystemID}   # driverNameID=SP-XID, driverSystemID=csp's ID
		//         ex) driverIID {"vm-01-9m4e2mr0ui3e8a215n4g", "i-0bc7123b7e5cbf79d"}
		spUUID, err = iidm.New(connectionName, rsType, reqInfo.IId.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else { // No Use IID Management
		spUUID = reqInfo.IId.NameId
	}

	// reqIID
	reqIId := cres.IID{NameId: reqInfo.IId.NameId, SystemId: spUUID}
	// driverIID
	driverIId := cres.IID{NameId: spUUID, SystemId: ""}
	reqInfo.IId = driverIId

	providerName, err := ccm.GetProviderNameByConnectionName(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// for subnet list
	subnetReqIIdZoneList := []SubnetReqZoneInfo{}
	subnetInfoList := []cres.SubnetInfo{}
	for _, info := range reqInfo.SubnetInfoList {
		subnetUUID := ""
		if GetID_MGMT(IDTransformMode) == "ON" { // Use IID Management
			subnetUUID, err = iidm.New(connectionName, rsSubnet, info.IId.NameId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		} else { // No Use IID Management
			subnetUUID = info.IId.NameId
		}

		// special code for KT CLOUD VPC
		// related Issue: #1105
		//   [KT Cloud VPC] To use NLB, needs to support the subnet management features with a fixed name.
		if providerName == "KTCLOUDVPC" {
			if info.IId.NameId == "NLB-SUBNET" {
				subnetUUID = "NLB-SUBNET"
			}
		}

		// reqIID
		subnetReqIId := cres.IID{NameId: info.IId.NameId, SystemId: subnetUUID}
		subnetReqInfo := SubnetReqZoneInfo{IId: subnetReqIId, Zone: info.Zone}
		subnetReqIIdZoneList = append(subnetReqIIdZoneList, subnetReqInfo)
		// driverIID
		subnetDriverIId := cres.IID{NameId: subnetUUID, SystemId: ""}
		info.IId = subnetDriverIId
		subnetInfoList = append(subnetInfoList, info)
	} // end of for _, info

	reqInfo.SubnetInfoList = subnetInfoList

	// (3) create Resource
	// VPC: driverIId, Subnet: driverIId List
	info, err := handler.CreateVPC(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (4) create spiderIID: {reqNameID, driverNameID:driverSystemID}
	//     ex) spiderIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	spiderIId := cres.IID{NameId: reqIId.NameId, SystemId: spUUID + ":" + info.IId.SystemId}

	// (5) insert IID
	// for VPC
	err = infostore.Insert(&VPCIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId})
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteVPC(info.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		cblog.Error(err)
		return nil, err
	}
	// for Subnet list
	for _, subnetInfo := range info.SubnetInfoList {
		subnetReqNameId := getSubnetReqNameId(subnetReqIIdZoneList, subnetInfo.IId.NameId)
		if subnetReqNameId == "" {
			cblog.Error(subnetInfo.IId.NameId + "is not requested Subnet.")
			continue
		}
		if subnetInfo.Zone == "" { // GCP has no Zone info
			subnetInfo.Zone = getSubnetReqZoneId(subnetReqIIdZoneList, subnetInfo.IId.NameId)
		}

		subnetSpiderIId := cres.IID{NameId: subnetReqNameId, SystemId: subnetInfo.IId.NameId + ":" + subnetInfo.IId.SystemId}
		err = infostore.Insert(&SubnetIIDInfo{ConnectionName: connectionName, ZoneId: subnetInfo.Zone, NameId: subnetSpiderIId.NameId, SystemId: subnetSpiderIId.SystemId,
			OwnerVPCName: reqIId.NameId})
		if err != nil {
			cblog.Error(err)
			// rollback
			// (1) for resource
			cblog.Info("<<ROLLBACK:TRY:VPC-CSP>> " + info.IId.SystemId)
			_, err2 := handler.DeleteVPC(info.IId)
			if err2 != nil {
				cblog.Error(err2)
				return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
			}
			// (2) for VPC IID
			cblog.Info("<<ROLLBACK:TRY:VPC-IID>> " + info.IId.NameId)
			_, err3 := infostore.DeleteByConditions(&VPCIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, spiderIId.NameId)
			if err3 != nil {
				cblog.Error(err3)
				return nil, fmt.Errorf(err.Error() + ", " + err3.Error())
			}
			// (3) for Subnet IID
			// delete all subnets of target VPC
			_, err := infostore.DeleteByConditions(&SubnetIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, OWNER_VPC_NAME_COLUMN, info.IId.NameId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			cblog.Error(err)
			return nil, err
		}
	}

	// (6) create userIID: {reqNameID, driverSystemID}
	//     ex) userIID {"seoul-service", "i-0bc7123b7e5cbf79d"}
	// for VPC
	userIId := cres.IID{NameId: reqIId.NameId, SystemId: info.IId.SystemId}
	info.IId = userIId

	// for Subnet list
	subnetUserInfoList := []cres.SubnetInfo{}
	for _, subnetInfo := range info.SubnetInfoList {
		subnetReqNameId := getSubnetReqNameId(subnetReqIIdZoneList, subnetInfo.IId.NameId)
		userIId := cres.IID{NameId: subnetReqNameId, SystemId: subnetInfo.IId.SystemId}
		subnetInfo.IId = userIId
		if subnetInfo.Zone == "" { // GCP has no Zone info
			subnetInfo.Zone = getSubnetReqZoneId(subnetReqIIdZoneList, subnetInfo.IId.NameId)
		}
		subnetUserInfoList = append(subnetUserInfoList, subnetInfo)
	}
	info.SubnetInfoList = subnetUserInfoList

	return &info, nil
}

// Get reqNameId from reqIIdZoneList whith driver NameId
func getSubnetReqNameId(reqIIdZoneList []SubnetReqZoneInfo, driverNameId string) string {
	for _, reqInfo := range reqIIdZoneList {
		if reqInfo.IId.SystemId == driverNameId {
			return reqInfo.IId.NameId
		}
	}
	return ""
}

// Get reqZoneId from reqIIdZoneList whith driver NameId
func getSubnetReqZoneId(reqIIdZoneList []SubnetReqZoneInfo, driverNameId string) string {
	for _, reqInfo := range reqIIdZoneList {
		if reqInfo.IId.SystemId == driverNameId {
			return reqInfo.Zone
		}
	}
	return ""
}

type ResultVPCInfo struct {
	vpcInfo cres.VPCInfo
	err     error
}

// (1) get IID:list
// (2) get VPCInfo:list
// (3) set userIID, and...
func ListVPC(connectionName string, rsType string) ([]*cres.VPCInfo, error) {
	cblog.Info("call ListVPC()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (1) get IID:list
	var iidInfoList []*VPCIIDInfo
	err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var infoList []*cres.VPCInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.VPCInfo{}
		return infoList, nil
	}

	// (2) Get VPCInfo-list with IID-list
	wg := new(sync.WaitGroup)
	resultInfoList := []*cres.VPCInfo{}
	var retChanInfos []chan ResultVPCInfo
	for i := 0; i < len(iidInfoList); i++ {
		retChanInfos = append(retChanInfos, make(chan ResultVPCInfo))
	}

	for idx, iidInfo := range iidInfoList {

		wg.Add(1)

		go getVPCInfo(connectionName, handler, cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}, retChanInfos[idx])

		wg.Done()

	}
	wg.Wait()

	var errList []string
	for idx, retChanInfo := range retChanInfos {
		chanInfo := <-retChanInfo

		if chanInfo.err != nil {
			if checkNotFoundError(chanInfo.err) {
				cblog.Info(chanInfo.err)
			} else {
				errList = append(errList, connectionName+":VPC:"+iidInfoList[idx].NameId+" # "+chanInfo.err.Error())
			}
		} else {
			resultInfoList = append(resultInfoList, &chanInfo.vpcInfo)
		}

		close(retChanInfo)
	}

	if len(errList) > 0 {
		cblog.Error(strings.Join(errList, "\n"))
		return nil, errors.New(strings.Join(errList, "\n"))
	}

	return resultInfoList, nil
}

func getVPCInfo(connectionName string, handler cres.VPCHandler, iid cres.IID, retInfo chan ResultVPCInfo) {

	vpcSPLock.RLock(connectionName, iid.NameId)
	// get resource(SystemId)
	info, err := handler.GetVPC(getDriverIID(iid))
	if err != nil {
		vpcSPLock.RUnlock(connectionName, iid.NameId)
		cblog.Error(err)
		retInfo <- ResultVPCInfo{cres.VPCInfo{}, err}
		return
	}

	// set ResourceInfo(IID.NameId)
	info.IId = getUserIID(iid)

	// set NameId for SubnetInfo List
	// create new SubnetInfo List
	subnetInfoList := []cres.SubnetInfo{}
	for _, subnetInfo := range info.SubnetInfoList {
		var subnetIIDInfo SubnetIIDInfo
		err := infostore.GetByConditionsAndContain(&subnetIIDInfo, CONNECTION_NAME_COLUMN, connectionName,
			OWNER_VPC_NAME_COLUMN, iid.NameId, SYSTEM_ID_COLUMN, getMSShortID(subnetInfo.IId.SystemId))
		if err != nil {
			// if not found, continue
			if checkNotFoundError(err) {
				cblog.Info(err)
				continue
			}
			vpcSPLock.RUnlock(connectionName, iid.NameId)
			cblog.Error(err)
			retInfo <- ResultVPCInfo{cres.VPCInfo{}, err}
			return
		}
		if subnetIIDInfo.NameId != "" { // insert only this user created.
			subnetInfo.IId = getUserIID(cres.IID{NameId: subnetIIDInfo.NameId, SystemId: subnetIIDInfo.SystemId})
			if subnetInfo.Zone == "" { // GCP has no Zone info
				subnetInfo.Zone = subnetIIDInfo.ZoneId
			}
			subnetInfoList = append(subnetInfoList, subnetInfo)
		}
	}
	vpcSPLock.RUnlock(connectionName, iid.NameId)

	info.SubnetInfoList = subnetInfoList

	retInfo <- ResultVPCInfo{info, nil}
}

// (1) get spiderIID(NameId)
// (2) get resource(driverIID)
// (3) set ResourceInfo(userIID)
func GetVPC(connectionName string, rsType string, nameID string) (*cres.VPCInfo, error) {
	cblog.Info("call GetVPC()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	nameID, err = EmptyCheckAndTrim("nameID", nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcSPLock.RLock(connectionName, nameID)
	defer vpcSPLock.RUnlock(connectionName, nameID)
	// (1) get spiderIID(NameId)
	var iidInfo VPCIIDInfo
	err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource(driverIID)
	info, err := handler.GetVPC(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	// (3) set ResourceInfo(userIID)
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

	// set NameId for SubnetInfo List
	// create new SubnetInfo List
	subnetInfoList := []cres.SubnetInfo{}
	for _, subnetInfo := range info.SubnetInfoList {
		var subnetIIDInfo SubnetIIDInfo
		err := infostore.GetByConditionsAndContain(&subnetIIDInfo, CONNECTION_NAME_COLUMN, connectionName,
			OWNER_VPC_NAME_COLUMN, info.IId.NameId, SYSTEM_ID_COLUMN, getMSShortID(subnetInfo.IId.SystemId))
		if err != nil {
			// if not found, continue
			if checkNotFoundError(err) {
				cblog.Info(err)
				continue
			}
			cblog.Error(err)
			return nil, err
		}
		if subnetIIDInfo.NameId != "" { // insert only this user created.
			subnetInfo.IId = getUserIID(cres.IID{NameId: subnetIIDInfo.NameId, SystemId: subnetIIDInfo.SystemId})
			if subnetInfo.Zone == "" { // GCP has no Zone info
				subnetInfo.Zone = subnetIIDInfo.ZoneId
			}
			subnetInfoList = append(subnetInfoList, subnetInfo)
		}
	}
	info.SubnetInfoList = subnetInfoList

	return &info, nil
}

// (1) check exist(NameID)
// (2) create Resource
// (3) insert IID
func AddSubnet(connectionName string, rsType string, vpcName string, reqInfo cres.SubnetInfo, IDTransformMode string) (*cres.VPCInfo, error) {
	cblog.Info("call AddSubnet()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcName, err = EmptyCheckAndTrim("vpcName", vpcName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcSPLock.Lock(connectionName, vpcName)
	defer vpcSPLock.Unlock(connectionName, vpcName)
	// (1) check exist(NameID)
	bool_ret, err := infostore.HasBy3Conditions(&SubnetIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName,
		NAME_ID_COLUMN, reqInfo.IId.NameId, OWNER_VPC_NAME_COLUMN, vpcName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if bool_ret {
		err := fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
		cblog.Error(err)
		return nil, err
	}
	// (2) create Resource
	var iidVPCInfo VPCIIDInfo
	err = infostore.GetByConditions(&iidVPCInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, vpcName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	subnetUUID := ""
	if GetID_MGMT(IDTransformMode) == "ON" { // Use IID Management
		subnetUUID, err = iidm.New(connectionName, rsType, reqInfo.IId.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else { // No Use IID Management
		subnetUUID = reqInfo.IId.NameId
	}

	// special code for KT CLOUD VPC
	// related Issue:
	//   #1105 [KT Cloud VPC] To use NLB, needs to support the subnet management features with a fixed name.
	providerName, err := ccm.GetProviderNameByConnectionName(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if providerName == "KTCLOUDVPC" {
		if reqInfo.IId.NameId == "NLB-SUBNET" {
			subnetUUID = "NLB-SUBNET"
		}
	}

	// driverIID for driver
	subnetReqNameId := reqInfo.IId.NameId
	reqInfo.IId = cres.IID{NameId: subnetUUID, SystemId: ""}
	info, err := handler.AddSubnet(getDriverIID(cres.IID{NameId: iidVPCInfo.NameId, SystemId: iidVPCInfo.SystemId}), reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) insert IID
	// for Subnet list
	for _, subnetInfo := range info.SubnetInfoList {
		if subnetInfo.IId.NameId == reqInfo.IId.NameId { // NameId => SS-UUID
			subnetSpiderIId := cres.IID{NameId: subnetReqNameId, SystemId: subnetInfo.IId.NameId + ":" + subnetInfo.IId.SystemId}
			err = infostore.Insert(&SubnetIIDInfo{ConnectionName: connectionName, ZoneId: reqInfo.Zone, NameId: subnetSpiderIId.NameId, SystemId: subnetSpiderIId.SystemId,
				OwnerVPCName: vpcName})
			if err != nil {
				cblog.Error(err)
				// rollback
				// (1) for resource
				cblog.Info("<<ROLLBACK:TRY:VPC-SUBNET-CSP>> " + subnetInfo.IId.SystemId)
				_, err2 := handler.RemoveSubnet(getDriverIID(cres.IID{NameId: iidVPCInfo.NameId, SystemId: iidVPCInfo.SystemId}), subnetInfo.IId)
				if err2 != nil {
					cblog.Error(err2)
					return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
				}
				// (2) for Subnet IID
				cblog.Info("<<ROLLBACK:TRY:VPC-SUBNET-IID>> " + subnetInfo.IId.NameId)
				_, err3 := infostore.DeleteBy3Conditions(&SubnetIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, subnetSpiderIId.NameId,
					OWNER_VPC_NAME_COLUMN, vpcName)
				if err3 != nil {
					cblog.Error(err3)
					return nil, fmt.Errorf(err.Error() + ", " + err3.Error())
				}
				cblog.Error(err)
				return nil, err
			}
		}
	}

	// (3) set ResourceInfo(userIID)
	info.IId = getUserIID(cres.IID{NameId: iidVPCInfo.NameId, SystemId: iidVPCInfo.SystemId})

	// set NameId for SubnetInfo List
	// create new SubnetInfo List
	subnetInfoList := []cres.SubnetInfo{}
	for _, subnetInfo := range info.SubnetInfoList {
		var subnetIIDInfo SubnetIIDInfo
		err := infostore.GetByConditionsAndContain(&subnetIIDInfo, CONNECTION_NAME_COLUMN, connectionName,
			OWNER_VPC_NAME_COLUMN, vpcName, SYSTEM_ID_COLUMN, getMSShortID(subnetInfo.IId.SystemId))
		if err != nil {
			// if not found, continue
			if checkNotFoundError(err) {
				cblog.Info(err)
				continue
			}
			cblog.Error(err)
			return nil, err
		}
		if subnetIIDInfo.NameId != "" { // insert only this user created.
			subnetInfo.IId = getUserIID(cres.IID{NameId: subnetIIDInfo.NameId, SystemId: subnetInfo.IId.SystemId})
			if subnetInfo.Zone == "" { // GCP has no Zone info
				subnetInfo.Zone = subnetIIDInfo.ZoneId
			}
			subnetInfoList = append(subnetInfoList, subnetInfo)
		}
	}
	info.SubnetInfoList = subnetInfoList

	return &info, nil
}

// (1) get spiderIID
// (2) delete Resource(SystemId)
// (3) delete IID
func RemoveSubnet(connectionName string, vpcName string, nameID string, force string) (bool, error) {
	cblog.Info("call RemoveSubnet()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	vpcName, err = EmptyCheckAndTrim("vpcName", vpcName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	nameID, err = EmptyCheckAndTrim("nameID", nameID)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	vpcSPLock.Lock(connectionName, vpcName)
	defer vpcSPLock.Unlock(connectionName, vpcName)

	// (1) get spiderIID for creating driverIID
	var iidInfo SubnetIIDInfo
	err = infostore.GetBy3Conditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID, OWNER_VPC_NAME_COLUMN, vpcName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	// (2) delete Resource(SystemId)
	driverIId := getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
	result := false

	cldConn, err := ccm.GetZoneLevelCloudConnection(connectionName, iidInfo.ZoneId)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	var iidVPCInfo VPCIIDInfo
	err = infostore.GetByConditions(&iidVPCInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, vpcName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	result, err = handler.(cres.VPCHandler).RemoveSubnet(getDriverIID(cres.IID{NameId: iidVPCInfo.NameId, SystemId: iidVPCInfo.SystemId}), driverIId)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}

	if force != "true" {
		if !result {
			return result, nil
		}
	}

	// (3) delete IID
	_, err = infostore.DeleteBy3Conditions(&SubnetIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID,
		OWNER_VPC_NAME_COLUMN, vpcName)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}

	return result, nil
}

// remove CSP's Subnet(SystemId)
func RemoveCSPSubnet(connectionName string, vpcName string, systemID string) (bool, error) {
	cblog.Info("call DeleteCSPSubnet()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	vpcName, err = EmptyCheckAndTrim("vpcName", vpcName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	systemID, err = EmptyCheckAndTrim("systemID", systemID)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	iid := cres.IID{NameId: "", SystemId: systemID}

	// delete Resource(SystemId)
	result := false
	// get owner vpc IIDInfo
	var iidVPCInfo VPCIIDInfo
	err = infostore.GetByConditions(&iidVPCInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, vpcName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	result, err = handler.(cres.VPCHandler).RemoveSubnet(getDriverIID(cres.IID{NameId: iidVPCInfo.NameId, SystemId: iidVPCInfo.SystemId}), iid)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	return result, nil
}

// (1) get spiderIID
// (2) delete Resource(SystemId)
// (3) delete IID
func DeleteVPC(connectionName string, rsType string, nameID string, force string) (bool, error) {
	cblog.Info("call DeleteeVPC()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	nameID, err = EmptyCheckAndTrim("nameID", nameID)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	vpcSPLock.Lock(connectionName, nameID)
	defer vpcSPLock.Unlock(connectionName, nameID)

	// (1) get spiderIID for creating driverIID
	var iidInfo VPCIIDInfo
	err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	// (2) delete Resource(SystemId)
	driverIId := getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
	result := false
	result, err = handler.(cres.VPCHandler).DeleteVPC(driverIId)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}
	if force != "true" {
		if !result {
			return result, nil
		}
	}

	// (3) delete IID
	// for vPC
	_, err = infostore.DeleteByConditions(&VPCIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, iidInfo.NameId)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}
	// for Subnet list
	_, err2 := infostore.DeleteByConditions(&SubnetIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, OWNER_VPC_NAME_COLUMN, iidInfo.NameId)
	if err2 != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}
	return result, nil
}

func CountAllVPCs() (int64, error) {
	var info VPCIIDInfo
	count, err := infostore.CountAllNameIDs(&info)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}

func CountVPCsByConnection(connectionName string) (int64, error) {
	var info VPCIIDInfo
	count, err := infostore.CountNameIDsByConnection(&info, connectionName)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}

func CountAllSubnets() (int64, error) {
	var info SubnetIIDInfo
	count, err := infostore.CountAllNameIDs(&info)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}

func CountSubnetsByConnection(connectionName string) (int64, error) {
	var info SubnetIIDInfo
	count, err := infostore.CountNameIDsByConnection(&info, connectionName)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}
