// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.

package commonruntime

import (
	"fmt"
	"strings"
	"strconv"
	"errors"
	"sync"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
)


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

	emptyPermissionList := []string{
        }

        err = ValidateStruct(userIID, emptyPermissionList)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        rsType := rsVPC

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

        // (1) check existence(UserID)
        bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsType, userIID)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        if bool_ret == true {
		err := fmt.Errorf(rsType + "-" + userIID.NameId + " already exists!")
		cblog.Error(err)
                return nil, err
        }

        // (2) get resource info(CSP-ID)
        // check existence and get info of this resouce in the CSP
	// Do not user NameId, because Azure driver use it like SystemId
        getInfo, err := handler.GetVPC( cres.IID{userIID.SystemId, userIID.SystemId} )
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // (3) create spiderIID: {UserID, SP-XID:CSP-ID}
        //     ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	// Do not user NameId, because Azure driver use it like SystemId
	systemId := getMSShortID(getInfo.IId.SystemId)
        spiderIId := cres.IID{userIID.NameId, systemId + ":" + getInfo.IId.SystemId}

        // (4) insert spiderIID
        // insert VPC SpiderIID to metadb
        _, err = iidRWLock.CreateIID(iidm.IIDSGROUP, connectionName, rsType, spiderIId)
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
		subnetSpiderIId := cres.IID{subnetUserId, systemId + ":" + subnetInfo.IId.SystemId}
                _, err = iidRWLock.CreateIID(iidm.SUBNETGROUP, connectionName, userIID.NameId, subnetSpiderIId)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }

                // setup subnet IID for return info
                subnetInfo.IId = cres.IID{subnetUserId, subnetInfo.IId.SystemId}
                getInfo.SubnetInfoList[count] = subnetInfo
        } // end of for _, info

        // set up VPC User IID for return info
        getInfo.IId = userIID

        return &getInfo, nil
}

// (1) check exist(NameID)
// (2) generate SP-XID and create reqIID, driverIID
// (3) create Resource
// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
// (5) insert spiderIID
// (6) create userIID
func CreateVPC(connectionName string, rsType string, reqInfo cres.VPCReqInfo) (*cres.VPCInfo, error) {
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

	// (1) check exist(NameID)
	bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsType, reqInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if bool_ret == true {
		err :=  fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
                cblog.Error(err)
                return nil, err
	}

        // check the Cloud Connection has the VPC already, when the CSP supports only 1 VPC.
        drv, err := ccm.GetCloudDriver(connectionName)
	if err != nil {
                cblog.Error(err)
                return nil, err
	}
        if (drv.GetDriverCapability().SINGLE_VPC == true) {
                list_ret, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsType)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                if list_ret != nil && len(list_ret) > 0 {
                        err :=  fmt.Errorf(rsType + "-" + connectionName + " can have only 1 VPC, but already have a VPC " + list_ret[0].IId.NameId)
                        cblog.Error(err)
                        return nil, err
                }
        }

	// (2) generate SP-XID and create reqIID, driverIID
	//     ex) SP-XID {"vm-01-9m4e2mr0ui3e8a215n4g"}
	//
	//     create reqIID: {reqNameID, reqSystemID}   # reqSystemID=SP-XID
	//         ex) reqIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g"} 
	//
	//     create driverIID: {driverNameID, driverSystemID}   # driverNameID=SP-XID, driverSystemID=csp's ID
	//         ex) driverIID {"vm-01-9m4e2mr0ui3e8a215n4g", "i-0bc7123b7e5cbf79d"}
	spUUID, err := iidm.New(connectionName, rsType, reqInfo.IId.NameId)
	if err != nil {
                cblog.Error(err)
                return nil, err
        }

	// reqIID
	reqIId := cres.IID{reqInfo.IId.NameId, spUUID}
	// driverIID
	driverIId := cres.IID{spUUID, ""}
	reqInfo.IId = driverIId

	// for subnet list
	subnetReqIIdList := []cres.IID{}
	subnetInfoList := []cres.SubnetInfo{}
	for _, info := range reqInfo.SubnetInfoList {
		subnetUUID, err := iidm.New(connectionName, rsSubnet, info.IId.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}

		// reqIID
		subnetReqIId := cres.IID{info.IId.NameId, subnetUUID}
		subnetReqIIdList = append(subnetReqIIdList, subnetReqIId)
		// driverIID
		subnetDriverIId := cres.IID{subnetUUID, ""}
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
	spiderIId := cres.IID{reqIId.NameId, spUUID + ":" + info.IId.SystemId}

	// (5) insert IID
	// for VPC
	iidInfo, err := iidRWLock.CreateIID(iidm.IIDSGROUP, connectionName, rsType, spiderIId)
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
		// key-value structure: ~/{SUBNETGROUP}/{ConnectionName}/{VPC-NameId}/{Subnet-reqNameId} [subnet-driverNameId:subnet-driverSystemId]  # VPC NameId => rsType
		subnetReqNameId := getReqNameId(subnetReqIIdList, subnetInfo.IId.NameId)
		if subnetReqNameId == "" {
			cblog.Error(subnetInfo.IId.NameId + "is not requested Subnet.")
			continue;
		}
		subnetSpiderIId := cres.IID{subnetReqNameId, subnetInfo.IId.NameId + ":" + subnetInfo.IId.SystemId}
		_, err := iidRWLock.CreateIID(iidm.SUBNETGROUP, connectionName, reqIId.NameId, subnetSpiderIId)
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
			_, err3 := iidRWLock.DeleteIID(iidm.IIDSGROUP, connectionName, rsType, iidInfo.IId)
			if err3 != nil {
				cblog.Error(err3)
				return nil, fmt.Errorf(err.Error() + ", " + err3.Error())
			}
			// (3) for Subnet IID
			tmpIIdInfoList, err := iidRWLock.ListIID(iidm.SUBNETGROUP, connectionName, info.IId.NameId) // VPC info.IId.NameId => rsType
			for _, subnetIIdInfo := range tmpIIdInfoList {
				_, err := iidRWLock.DeleteIID(iidm.SUBNETGROUP, connectionName, info.IId.NameId, subnetIIdInfo.IId) // VPC info.IId.NameId => rsType
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
			}
			cblog.Error(err)
			return nil, err
		}
	}

	// (6) create userIID: {reqNameID, driverSystemID}
	//     ex) userIID {"seoul-service", "i-0bc7123b7e5cbf79d"}
	// for VPC
	userIId := cres.IID{reqIId.NameId, info.IId.SystemId}
	info.IId = userIId

	// for Subnet list
	subnetUserInfoList := []cres.SubnetInfo{}
	for _, subnetInfo := range info.SubnetInfoList {
		subnetReqNameId := getReqNameId(subnetReqIIdList, subnetInfo.IId.NameId)
		userIId := cres.IID{subnetReqNameId, subnetInfo.IId.SystemId}
		subnetInfo.IId = userIId
		subnetUserInfoList = append(subnetUserInfoList, subnetInfo)
	}
	info.SubnetInfoList = subnetUserInfoList

	return &info, nil
}

// Get reqNameId from reqIIdList whith driver NameId
func getReqNameId(reqIIdList []cres.IID, driverNameId string) string {
	for _, iid := range reqIIdList {
		if iid.SystemId == driverNameId {
			return iid.NameId
		}
	}
	return ""
}

type ResultVPCInfo struct {
        vpcInfo  cres.VPCInfo
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
	iidInfoList, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsType)
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
        for i:=0 ; i<len(iidInfoList); i++ {
                retChanInfos = append(retChanInfos, make(chan ResultVPCInfo))
        }

        for idx, iidInfo := range iidInfoList {

                wg.Add(1)

                go getVPCInfo(connectionName, handler, iidInfo.IId, retChanInfos[idx])

                wg.Done()

        }
        wg.Wait()

        var errList []string
        for idx, retChanInfo := range retChanInfos {
                chanInfo := <-retChanInfo

                if chanInfo.err  != nil {
                        if checkNotFoundError(chanInfo.err) {
                                cblog.Info(chanInfo.err) } else {
                                errList = append(errList, connectionName + ":VPC:" + iidInfoList[idx].IId.NameId + " # " + chanInfo.err.Error())
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
		// VPC info.IId.NameId => rsType
		subnetIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.SUBNETGROUP, connectionName, iid.NameId, subnetInfo.IId) 
		if err != nil {
vpcSPLock.RUnlock(connectionName, iid.NameId)
			cblog.Error(err)
			retInfo <- ResultVPCInfo{cres.VPCInfo{}, err}
			return
		}
		if subnetIIDInfo.IId.NameId != "" { // insert only this user created.
			subnetInfo.IId = getUserIID(subnetIIDInfo.IId)
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
	iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource(driverIID)
	info, err := handler.GetVPC(getDriverIID(iidInfo.IId))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	// (3) set ResourceInfo(userIID)
	info.IId = getUserIID(iidInfo.IId)

	// set NameId for SubnetInfo List
	// create new SubnetInfo List
	subnetInfoList := []cres.SubnetInfo{}
	for _, subnetInfo := range info.SubnetInfoList {		
		subnetIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.SUBNETGROUP, connectionName, info.IId.NameId, subnetInfo.IId) // VPC info.IId.NameId => rsType
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		if subnetIIDInfo.IId.NameId != "" { // insert only this user created.
			subnetInfo.IId = getUserIID(subnetIIDInfo.IId)
			subnetInfoList = append(subnetInfoList, subnetInfo)
		}
	}
	info.SubnetInfoList = subnetInfoList

	return &info, nil
}

// (1) check exist(NameID)
// (2) create Resource
// (3) insert IID
func AddSubnet(connectionName string, rsType string, vpcName string, reqInfo cres.SubnetInfo) (*cres.VPCInfo, error) {
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
	bool_ret, err := iidRWLock.IsExistIID(iidm.SUBNETGROUP, connectionName, vpcName, reqInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if bool_ret == true {
		err := fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
		cblog.Error(err)
		return nil, err
	}
	// (2) create Resource
	iidVPCInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{vpcName, ""})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	subnetUUID, err := iidm.New(connectionName, rsType, reqInfo.IId.NameId)
	if err != nil {
                cblog.Error(err)
                return nil, err
        }

	// driverIID for driver
	subnetReqNameId := reqInfo.IId.NameId
	reqInfo.IId = cres.IID{subnetUUID, ""}
	info, err := handler.AddSubnet(getDriverIID(iidVPCInfo.IId), reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) insert IID
	// for Subnet list
	for _, subnetInfo := range info.SubnetInfoList {		
		if subnetInfo.IId.NameId == reqInfo.IId.NameId {  // NameId => SS-UUID
			// key-value structure: ~/{SUBNETGROUP}/{ConnectionName}/{VPC-NameId}/{Subnet-reqNameId} [subnet-driverNameId:subnet-driverSystemId]  # VPC NameId => rsType
			subnetSpiderIId := cres.IID{subnetReqNameId, subnetInfo.IId.NameId + ":" + subnetInfo.IId.SystemId}
			_, err := iidRWLock.CreateIID(iidm.SUBNETGROUP, connectionName, vpcName, subnetSpiderIId)
			if err != nil {
				cblog.Error(err)
				// rollback
				// (1) for resource
				cblog.Info("<<ROLLBACK:TRY:VPC-SUBNET-CSP>> " + subnetInfo.IId.SystemId)
				_, err2 := handler.RemoveSubnet(getDriverIID(iidVPCInfo.IId), subnetInfo.IId)
				if err2 != nil {
					cblog.Error(err2)
					return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
				}
				// (2) for Subnet IID
				cblog.Info("<<ROLLBACK:TRY:VPC-SUBNET-IID>> " + subnetInfo.IId.NameId)
				_, err3 := iidRWLock.DeleteIID(iidm.SUBNETGROUP, connectionName, vpcName, subnetSpiderIId) // vpcName => rsType
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
	info.IId = getUserIID(iidVPCInfo.IId)

	// set NameId for SubnetInfo List
	// create new SubnetInfo List
	subnetInfoList := []cres.SubnetInfo{}
	for _, subnetInfo := range info.SubnetInfoList {		
		subnetIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.SUBNETGROUP, connectionName, vpcName, subnetInfo.IId) // vpcName => rsType
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		if subnetIIDInfo.IId.NameId != "" { // insert only this user created.
			subnetInfo.IId = getUserIID(subnetIIDInfo.IId)
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
                return false, err
		cblog.Error(err)
        }

        vpcName, err = EmptyCheckAndTrim("vpcName", vpcName)
        if err != nil {
                return false, err
		cblog.Error(err)
        }

        nameID, err = EmptyCheckAndTrim("nameID", nameID)
        if err != nil {
                return false, err
		cblog.Error(err)
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

	vpcSPLock.Lock(connectionName, vpcName)
	defer vpcSPLock.Unlock(connectionName, vpcName)

	// (1) get spiderIID for creating driverIID
	iidInfo, err := iidRWLock.GetIID(iidm.SUBNETGROUP, connectionName, vpcName, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	// (2) delete Resource(SystemId)
	driverIId := getDriverIID(iidInfo.IId)
	result := false


	iidVPCInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{vpcName, ""})
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	result, err = handler.(cres.VPCHandler).RemoveSubnet(getDriverIID(iidVPCInfo.IId), driverIId)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}

	if force != "true" {
		if result == false {
			return result, nil
		}
	}

	// (3) delete IID
	_, err = iidRWLock.DeleteIID(iidm.SUBNETGROUP, connectionName, vpcName, cres.IID{nameID, ""})
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
                return false, err
		cblog.Error(err)
        }

        vpcName, err = EmptyCheckAndTrim("vpcName", vpcName)
        if err != nil {
                return false, err
		cblog.Error(err)
        }

        systemID, err = EmptyCheckAndTrim("systemID", systemID)
        if err != nil {
                return false, err
		cblog.Error(err)
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

        iid := cres.IID{"", systemID}

        // delete Resource(SystemId)
        result := false
	// get owner vpc IIDInfo
	iidVPCInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{vpcName, ""})
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	result, err = handler.(cres.VPCHandler).RemoveSubnet(getDriverIID(iidVPCInfo.IId), iid)
	if err != nil {
		cblog.Error(err)
		return false, err
	}


	return result, nil
}

