// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2022.10.

package commonruntime

import (
	"fmt"
	_ "strings"
	_ "errors"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
)


//================ Cluster Handler

func GetClusterOwnerVPC(connectionName string, cspID string) (owerVPC cres.IID, err error) {
        cblog.Info("call GetClusterOwnerVPC()")

        // check empty and trim user inputs
        connectionName, err = EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                cblog.Error(err)
                return cres.IID{}, err
        }

        cspID, err = EmptyCheckAndTrim("cspID", cspID)
        if err != nil {
                cblog.Error(err)
                return cres.IID{}, err
        }

        rsType := rsCluster

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return cres.IID{}, err
        }

        handler, err := cldConn.CreateClusterHandler()
        if err != nil {
                cblog.Error(err)
                return cres.IID{}, err
        }

// Except Management API
//clusterSPLock.RLock()
//vpcSPLock.RLock()

        // (1) check existence(cspID)
        iidInfoList, err := getAllClusterIIDInfoList(connectionName)
        if err != nil {
//vpcSPLock.RUnlock()
//clusterSPLock.RUnlock()
                cblog.Error(err)
                return cres.IID{}, err
        }
        var isExist bool=false
        var nameId string
        for _, OneIIdInfo := range iidInfoList {
                if getMSShortID(getDriverSystemId(OneIIdInfo.IId)) == cspID {
                        nameId = OneIIdInfo.IId.NameId
                        isExist = true
                        break
                }
        }
        if isExist == true {
//vpcSPLock.RUnlock()
//clusterSPLock.RUnlock()
                err :=  fmt.Errorf(rsType + "-" + cspID + " already exists with " + nameId + "!")
                cblog.Error(err)
                return cres.IID{}, err
        }

        // (2) get resource info(CSP-ID)
        // check existence and get info of this resouce in the CSP
        // Do not user NameId, because Azure driver use it like SystemId
        getInfo, err := handler.GetCluster( cres.IID{getMSShortID(cspID), cspID} )
        if err != nil {
//vpcSPLock.RUnlock()
//clusterSPLock.RUnlock()
                cblog.Error(err)
                return cres.IID{}, err
        }

        // (3) get VPC IID:list
        vpcIIDInfoList, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsVPC)
        if err != nil {
//vpcSPLock.RUnlock()
//clusterSPLock.RUnlock()
                cblog.Error(err)
                return cres.IID{}, err
        }
//vpcSPLock.RUnlock()
//clusterSPLock.RUnlock()

        //--------
        //-------- ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
        //--------
        // Do not user NameId, because Azure driver use it like SystemId
        vpcCSPID := getMSShortID(getInfo.Network.VpcIID.SystemId)
        if vpcIIDInfoList == nil || len(vpcIIDInfoList) <= 0 {
                return cres.IID{"", vpcCSPID}, nil
        }

        // (4) check existence in the MetaDB
        for _, one := range vpcIIDInfoList {
                if getMSShortID(getDriverSystemId(one.IId)) == vpcCSPID {
                        return cres.IID{one.IId.NameId, vpcCSPID}, nil
                }
        }

        return cres.IID{"", vpcCSPID}, nil
}

// UserIID{UserID, CSP-ID} => SpiderIID{UserID, SP-XID:CSP-ID}
// (0) check VPC existence(VPC UserID)
// (1) check existence(UserID)
// (2) get resource info(CSP-ID)
// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
// (4) insert spiderIID
func RegisterCluster(connectionName string, vpcUserID string, userIID cres.IID) (*cres.ClusterInfo, error) {
        cblog.Info("call RegisterCluster()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return nil, err
        }

        vpcUserID, err = EmptyCheckAndTrim("vpcUserID", vpcUserID)
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

        rsType := rsCluster

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

	handler, err := cldConn.CreateClusterHandler()
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        vpcSPLock.RLock(connectionName, vpcUserID)
        defer vpcSPLock.RUnlock(connectionName, vpcUserID)
        clusterSPLock.Lock(connectionName, userIID.NameId)
        defer clusterSPLock.Unlock(connectionName, userIID.NameId)

        // (0) check VPC existence(VPC UserID)
        bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{vpcUserID, ""})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        if bool_ret == false {
		err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsVPC), vpcUserID)
		cblog.Error(err)
                return nil, err
        }

        // (1) check existence(UserID)
        iidInfoList, err := getAllClusterIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        var isExist bool=false
        for _, OneIIdInfo := range iidInfoList {
                if OneIIdInfo.IId.NameId == userIID.NameId {
                        isExist = true
			break
                }
        }

        if isExist == true {
                err :=  fmt.Errorf(rsType + "-" + userIID.NameId + " already exists!")
                cblog.Error(err)
                return nil, err
        }


        // (2) get resource info(CSP-ID)
        // check existence and get info of this resouce in the CSP
	// Do not user NameId, because Azure driver use it like SystemId
        getInfo, err := handler.GetCluster( cres.IID{getMSShortID(userIID.SystemId), userIID.SystemId} )
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
        // insert Cluster SpiderIID to metadb
	_, err = iidRWLock.CreateIID(iidm.CLUSTERGROUP, connectionName, vpcUserID, spiderIId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // set up Cluster User IID for return info
        getInfo.IId = userIID

        // set up VPC UserIID for return info
        iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{vpcUserID, ""})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        getInfo.Network.VpcIID = getUserIID(iidInfo.IId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }


        return &getInfo, nil
}

// (1) check exist(NameID)
// (2) generate SP-XID and create reqIID, driverIID
// (3) create Resource
// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
// (5) insert spiderIID
// (6) create userIID
// (7) set used Resources's userIID
func CreateCluster(connectionName string, rsType string, reqInfo cres.ClusterInfo) (*cres.ClusterInfo, error) { 
	cblog.Info("call CreateCluster()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return nil, err
        }

	// @todo
	/* Currently, Validator does not support the struct has a point of Array such as SecurityReqInfo
        emptyPermissionList := []string{
                "resources.IID:SystemId",
                "resources.SecurityReqInfo:Direction", // because can be unused in some CSP
                "resources.SecurityRuleInfo:CIDR",     // because can be set without soruce CIDR
        }

        err = ValidateStruct(reqInfo, emptyPermissionList)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
	*/

	//+++++++++++++++++++++ Set NetworkInfo's SystemId
	netReqInfo := reqInfo.Network
	vpcSPLock.RLock(connectionName, netReqInfo.VpcIID.NameId)
	defer vpcSPLock.RUnlock(connectionName, netReqInfo.VpcIID.NameId)
	// (1) VpcIID
        if netReqInfo.VpcIID.NameId != "" {
                // get spiderIID
                vpcIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, netReqInfo.VpcIID)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                // set driverIID
                netReqInfo.VpcIID = getDriverIID(vpcIIDInfo.IId)
        }

        // (2) SubnetIIDs
        for idx, subnetIID := range netReqInfo.SubnetIIDs {
                subnetIIdInfo, err := iidRWLock.GetIID(iidm.SUBNETGROUP, connectionName, netReqInfo.VpcIID.NameId, subnetIID) // VpcIID.NameId => rsType
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                // set driverIID
                netReqInfo.SubnetIIDs[idx] = getDriverIID(subnetIIdInfo.IId)
	}

        // (3) SecurityGroupIIDs
        for idx, sgIID := range netReqInfo.SecurityGroupIIDs {
        	sgSPLock.RLock(connectionName, sgIID.NameId)
		defer sgSPLock.RUnlock(connectionName, sgIID.NameId)
                sgIIdInfo, err := iidRWLock.GetIID(iidm.SGGROUP, connectionName, netReqInfo.VpcIID.NameId, sgIID)  // VpcIID.NameId => rsType
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                // set driverIID
                netReqInfo.SecurityGroupIIDs[idx] = getDriverIID(sgIIdInfo.IId)
        }


	//+++++++++++++++++++++ Set NodeGroupInfo's SystemId
	for idx, ngInfo := range reqInfo.NodeGroupList { 
		// (1) ImageIID
		ngInfo.ImageIID.SystemId = ngInfo.ImageIID.NameId

		// (2) KeyPair
		keySPLock.RLock(connectionName, ngInfo.KeyPairIID.NameId)
		defer keySPLock.RUnlock(connectionName, ngInfo.KeyPairIID.NameId)
		keyIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsKey, ngInfo.KeyPairIID)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		reqInfo.NodeGroupList[idx].KeyPairIID = getDriverIID(keyIIDInfo.IId)
	}
        //+++++++++++++++++++++++++++++++++++++++++++

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateClusterHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

clusterSPLock.Lock(connectionName, reqInfo.IId.NameId)
defer clusterSPLock.Unlock(connectionName, reqInfo.IId.NameId)

	// (1) check exist(NameID)
	iidInfoList, err := getAllClusterIIDInfoList(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	var isExist bool=false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.IId.NameId == reqInfo.IId.NameId {
			isExist = true
		}
	}

	if isExist == true {
		err :=  fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
		cblog.Error(err)
		return nil, err
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

	// (3) create Resource
	info, err := handler.CreateCluster(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
	//     ex) spiderIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	spiderIId := cres.IID{reqIId.NameId, spUUID + ":" + info.IId.SystemId}

	// (5) insert spiderIID
	iidInfo, err := iidRWLock.CreateIID(iidm.CLUSTERGROUP, connectionName, netReqInfo.VpcIID.NameId, spiderIId)  // reqIId.NameId => rsType
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteCluster(info.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		cblog.Error(err)
		return nil, err
	}

	// (6) create userIID: {reqNameID, driverSystemID}
	//     ex) userIID {"seoul-service", "i-0bc7123b7e5cbf79d"}
	info.IId = getUserIID(iidInfo.IId)

	// (7) set used Resources's userIID
	err = setResourcesNameId(connectionName, &info)
	if err != nil {
		cblog.Error(err)
                return nil, err
	}

	return &info, nil
}

func setResourcesNameId(connectionName string, info *cres.ClusterInfo) error {
	//+++++++++++++++++++++ Set NetworkInfo's NameId
	netInfo := info.Network
	// (1) VpcIID
	// get spiderIID
	vpcIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, connectionName, rsVPC, netInfo.VpcIID)
	if err != nil {
		cblog.Error(err)
		return err
	}
	// set NameId
	netInfo.VpcIID.NameId = vpcIIDInfo.IId.NameId

        // (2) SubnetIIDs
        for idx, subnetIID := range netInfo.SubnetIIDs {
                subnetIIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.SUBNETGROUP, connectionName, netInfo.VpcIID.NameId, subnetIID) // VpcIID.NameId => rsType
                if err != nil {
                        cblog.Error(err)
                        return err
                }
                // set NameId
                netInfo.SubnetIIDs[idx].NameId = subnetIIdInfo.IId.NameId
	}

        // (3) SecurityGroupIIDs
        for idx, sgIID := range netInfo.SecurityGroupIIDs {
                sgIIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.SGGROUP, connectionName, netInfo.VpcIID.NameId, sgIID)  // VpcIID.NameId => rsType
                if err != nil {
                        cblog.Error(err)
                        return err
                }
                // set NameId
                netInfo.SecurityGroupIIDs[idx].NameId = sgIIdInfo.IId.NameId
        }


	//+++++++++++++++++++++ Set NodeGroupInfo's NameId
	for idx, ngInfo := range info.NodeGroupList { 
		// (1) ImageIID
		ngInfo.ImageIID.NameId = ngInfo.ImageIID.SystemId

		// (2) KeyPair
		keyIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, connectionName, rsKey, ngInfo.KeyPairIID)
		if err != nil {
			cblog.Error(err)
			return err
		}
		info.NodeGroupList[idx].KeyPairIID.NameId = keyIIDInfo.IId.NameId
	}

        return nil	
}

// (1) get IID:list
// (2) get ClusterInfo:list
// (3) set userIID, and ...
func ListCluster(connectionName string, rsType string) ([]*cres.ClusterInfo, error) {
	cblog.Info("call ListCluster()")

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

	handler, err := cldConn.CreateClusterHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}


	// (1) get IID:list
	iidInfoList, err := getAllClusterIIDInfoList(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var infoList []*cres.ClusterInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.ClusterInfo{}
		return infoList, nil
	}

	// (2) Get ClusterInfo-list with IID-list
	infoList2 := []*cres.ClusterInfo{}
	for _, iidInfo := range iidInfoList {

clusterSPLock.RLock(connectionName, iidInfo.IId.NameId)

		// get resource(SystemId)
		info, err := handler.GetCluster(getDriverIID(iidInfo.IId))
		if err != nil {
clusterSPLock.RUnlock(connectionName, iidInfo.IId.NameId)
			if checkNotFoundError(err) {
				cblog.Info(err)
				continue
			}
			cblog.Error(err)
			return nil, err
		}
clusterSPLock.RUnlock(connectionName, iidInfo.IId.NameId)


		// (3) set ResourceInfo(IID.NameId)
		// set ResourceInfo
		info.IId = getUserIID(iidInfo.IId)

		// set used Resources's userIID
		err = setResourcesNameId(connectionName, &info)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}

		infoList2 = append(infoList2, &info)
	}

	return infoList2, nil
}

// Get All IID:list of Cluster
// (1) Get VPC's Name List
// (2) Create All Cluster's IIDInfo List
func getAllClusterIIDInfoList(connectionName string) ([]*iidm.IIDInfo, error) {

        // (1) Get VPC's Name List
        // format) /resource-info-spaces/{iidGroup}/{connectionName}/{resourceType}/{resourceName} [{resourceID}]
        vpcNameList, err := iidRWLock.ListResourceType(iidm.CLUSTERGROUP, connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
	vpcNameList = uniqueNameList(vpcNameList)
        // (2) Create All Cluster's IIDInfo List
        iidInfoList := []*iidm.IIDInfo{}
        for _, vpcName := range vpcNameList {
                iidInfoListForOneVPC, err := iidRWLock.ListIID(iidm.CLUSTERGROUP, connectionName, vpcName)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                iidInfoList = append(iidInfoList, iidInfoListForOneVPC...)
        }
        return iidInfoList, nil
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetCluster(connectionName string, rsType string, nameID string) (*cres.ClusterInfo, error) {
	cblog.Info("call GetCluster()")

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

	handler, err := cldConn.CreateClusterHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

clusterSPLock.RLock(connectionName, nameID)
defer clusterSPLock.RUnlock(connectionName, nameID)

	// (1) get IID(NameId)
	iidInfoList, err := getAllClusterIIDInfoList(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	var iidInfo *iidm.IIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.IId.NameId == nameID {
			iidInfo = OneIIdInfo
			bool_ret = true
			break;
		}
	}
	if bool_ret == false {
		err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsType), nameID)
		cblog.Error(err)
                return nil, err
        }

	// (2) get resource(SystemId)
	info, err := handler.GetCluster(getDriverIID(iidInfo.IId))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) set ResourceInfo(IID.NameId)
	// set ResourceInfo
	info.IId = getUserIID(iidInfo.IId)

        // set used Resources's userIID
        err = setResourcesNameId(connectionName, &info)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

	return &info, nil
}

// (1) check exist(NameID) and Nodes
// (2) add Nodes
// (3) Get ClusterInfo
// (4) Set ResoureInfo
func AddNodeGroup(connectionName string, clusterName string, reqInfo cres.NodeGroupInfo) (*cres.NodeGroupInfo, error) {
        cblog.Info("call AddClusterNodes()")

	return nil, nil      
}

func ListNodeGroup(connectionName string, clusterName string) ([]*cres.NodeGroupInfo, error) {
	return nil, nil
}

func GetNodeGroup(connectionName string, clusterNameId string, nodeGroupNameId string) (cres.NodeGroupInfo, error) {
	return cres.NodeGroupInfo{}, nil
}

func SetNodeGroupAutoScaling(connectionName string, clusterNameId string, nodeGroupNameId string, on bool) (bool, error) {
	return true, nil
}

// (1) check exist(NameID)
// (2) change NodeGroup
// (3) Get ClusterInfo
// (4) Set ResoureInfo
func ChangeNodeGroupScaling(connectionName string, clusterName string, nodeGroupNameId string, 
	DesiredNodeSize int, MinNodeSize int, MaxNodeSize int) (cres.NodeGroupInfo, error) {
        cblog.Info("call ChangeNodeGroupScaling()")

	return cres.NodeGroupInfo{}, nil      
}

// (1) check exist(NameID)
// (2) remove Nodes
func RemoveNodeGroup(connectionName string, clusterName string, vmNames []string) (bool, error) {
        cblog.Info("call RemoveClusterNodes()")

        return true, nil  
}
