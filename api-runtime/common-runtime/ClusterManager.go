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
	"strings"
	"strconv"
	_ "errors"

	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
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

        //++++++++++++++++++++++++
        // set up Cluster User IID for return info
        getInfo.IId = userIID        
        // set up NodeGroupList User IID in this Cluster
        err = registerNodeGroupList(connectionName, &getInfo)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        // set up inner Resource User IID for return info
        setResourcesNameId(connectionName, &getInfo)

        return &getInfo, nil
}

func registerNodeGroupList(connectionName string, info *cres.ClusterInfo) error {
        // insert NodeGroup's spiderIIDs to metadb and setup NodeGroup IID for return info
        for count, ngInfo := range info.NodeGroupList {
                // generate NodeGroup's UserID
                ngUserId := info.IId.NameId + "-nodegroup-" + strconv.Itoa(count)

                // insert a NodeGroup SpiderIID to metadb
                // Do not user NameId, because Azure driver use it like SystemId
                systemId := getMSShortID(ngInfo.IId.SystemId)
                ngSpiderIId := cres.IID{ngUserId, systemId + ":" + ngInfo.IId.SystemId}
                _, err := iidRWLock.CreateIID(iidm.NGGROUP, connectionName, info.IId.NameId, ngSpiderIId)
                if err != nil {
                        cblog.Error(err)
                        return err
                }
        } // end of for _, info

        return nil
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
	netReqInfo := &reqInfo.Network
	vpcSPLock.RLock(connectionName, netReqInfo.VpcIID.NameId)
	defer vpcSPLock.RUnlock(connectionName, netReqInfo.VpcIID.NameId)
	// (1) VpcIID
        var vpcIIDInfo *iidm.IIDInfo
        if netReqInfo.VpcIID.NameId != "" {
                // get spiderIID
                vpcIIDInfo, err = iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, netReqInfo.VpcIID)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                // set driverIID
                netReqInfo.VpcIID = getDriverIID(vpcIIDInfo.IId)
        }

        // (2) SubnetIIDs
        for idx, subnetIID := range netReqInfo.SubnetIIDs {
                subnetIIdInfo, err := iidRWLock.GetIID(iidm.SUBNETGROUP, connectionName, vpcIIDInfo.IId.NameId, subnetIID) // VpcIID.NameId => rsType
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
                sgIIdInfo, err := iidRWLock.GetIID(iidm.SGGROUP, connectionName, vpcIIDInfo.IId.NameId, sgIID)  // VpcIID.NameId => rsType
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
		reqInfo.NodeGroupList[idx].ImageIID.SystemId = ngInfo.ImageIID.NameId

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

        // Create SP-UUID for NodeGroup list
        ngReqIIdList := []cres.IID{}
        ngInfoList := []cres.NodeGroupInfo{}
        for _, info := range reqInfo.NodeGroupList {
                nodeGroupUUID, err := iidm.New(connectionName, rsNodeGroup, info.IId.NameId)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }

                // reqIID
                ngReqIId := cres.IID{info.IId.NameId, nodeGroupUUID}
                ngReqIIdList = append(ngReqIIdList, ngReqIId)
                // driverIID
                ngDriverIId := cres.IID{nodeGroupUUID, ""}
                info.IId = ngDriverIId
                ngInfoList = append(ngInfoList, info)
        } // end of for _, info
        reqInfo.NodeGroupList = ngInfoList

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
	iidInfo, err := iidRWLock.CreateIID(iidm.CLUSTERGROUP, connectionName, vpcIIDInfo.IId.NameId, spiderIId)  // reqIId.NameId => rsType
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

        // insert spiderIID for NodeGroup list
        for _, ngInfo := range info.NodeGroupList {
                // key-value structure: ~/{NGGROUP}/{ConnectionName}/{Cluster-NameId}/{NodeGroup-reqNameId} 
		// 			[NodeGroup-driverNameId:nodegroup-driverSystemId]  # Cluster NameId => rsType
                ngReqNameId := getReqNameId(ngReqIIdList, ngInfo.IId.NameId)
                if ngReqNameId == "" {
                        cblog.Error(ngInfo.IId.NameId + "is not a requested NodeGroup.")
                        continue;
                }
                ngSpiderIId := cres.IID{ngReqNameId, ngInfo.IId.NameId + ":" + ngInfo.IId.SystemId}
                _, err := iidRWLock.CreateIID(iidm.NGGROUP, connectionName, reqIId.NameId, ngSpiderIId)
                if err != nil {
                        cblog.Error(err)
                        // rollback
                        // (1) for resource
                        cblog.Info("<<ROLLBACK:TRY:CLUSTER-CSP>> " + info.IId.SystemId)
                        _, err2 := handler.DeleteCluster(info.IId)
                        if err2 != nil {
                                cblog.Error(err2)
                                return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
                        }
                        // (2) for Cluster IID
                        cblog.Info("<<ROLLBACK:TRY:CLUSTER-IID>> " + info.IId.NameId)
                        _, err3 := iidRWLock.DeleteIID(iidm.CLUSTERGROUP, connectionName, vpcIIDInfo.IId.NameId, iidInfo.IId)
                        if err3 != nil {
                                cblog.Error(err3)
                                return nil, fmt.Errorf(err.Error() + ", " + err3.Error())
                        }
                        // (3) for NodeGroup IID
                        tmpIIdInfoList, err := iidRWLock.ListIID(iidm.NGGROUP, connectionName, info.IId.NameId) // Cluster info.IId.NameId => rsType
                        for _, ngIIdInfo := range tmpIIdInfoList {
                                _, err := iidRWLock.DeleteIID(iidm.NGGROUP, connectionName, info.IId.NameId, ngIIdInfo.IId) // Cluster info.IId.NameId => rsType
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
	netInfo := &info.Network
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
                // (1) NodeGroup IID
                ngIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.NGGROUP, connectionName, info.IId.NameId, ngInfo.IId)
                if err != nil {
                        cblog.Error(err)
                        return err
                }
                info.NodeGroupList[idx].IId.NameId = ngIIDInfo.IId.NameId

		// (2) ImageIID
		info.NodeGroupList[idx].ImageIID.NameId = ngInfo.ImageIID.SystemId

		// (3) KeyPair
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
func ListCluster(connectionName string, nameSpace string, rsType string) ([]*cres.ClusterInfo, error) {
	cblog.Info("call ListCluster()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
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

	if nameSpace != "" {
		nameSpace += "-"
	}

	// (2) Get ClusterInfo-list with IID-list
	infoList2 := []*cres.ClusterInfo{}
	for _, iidInfo := range iidInfoList {

		if nameSpace != "" {
			if !strings.HasPrefix(iidInfo.IId.NameId, nameSpace) {
				continue;
			}
		}
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
func GetCluster(connectionName string, rsType string, clusterName string) (*cres.ClusterInfo, error) {
	cblog.Info("call GetCluster()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return nil, err
        }

        clusterName, err = EmptyCheckAndTrim("clusterName", clusterName)
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

clusterSPLock.RLock(connectionName, clusterName)
defer clusterSPLock.RUnlock(connectionName, clusterName)

	// (1) get IID(NameId)
	iidInfoList, err := getAllClusterIIDInfoList(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	var iidInfo *iidm.IIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.IId.NameId == clusterName {
			iidInfo = OneIIdInfo
			bool_ret = true
			break;
		}
	}
	if bool_ret == false {
		err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsType), clusterName)
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

// (1) check exist(NameID)
// (2) add NodeGroup
// (3) Get ClusterInfo
// (4) Set ResoureInfo
func AddNodeGroup(connectionName string, rsType string, clusterName string, reqInfo cres.NodeGroupInfo) (*cres.ClusterInfo, error) {
        cblog.Info("call AddNodeGroup()")

        // check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        clusterName, err = EmptyCheckAndTrim("clusterName", clusterName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        providerName, err := ccm.GetProviderNameByConnectionName(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        //+++++++++++++++++++++ Set NodeGroupInfo's SystemId
        // (1) ImageIID
        reqInfo.ImageIID.SystemId = reqInfo.ImageIID.NameId

        // (2) KeyPair
        keySPLock.RLock(connectionName, reqInfo.KeyPairIID.NameId)
        defer keySPLock.RUnlock(connectionName, reqInfo.KeyPairIID.NameId)
        keyIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsKey, reqInfo.KeyPairIID)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        reqInfo.KeyPairIID = getDriverIID(keyIIDInfo.IId)
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

clusterSPLock.Lock(connectionName, clusterName)
defer clusterSPLock.Unlock(connectionName, clusterName)

        // (1) check exist(clusterName)
        iidInfoList, err := getAllClusterIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        var iidInfo *iidm.IIDInfo
        var bool_ret = false
        for _, OneIIdInfo := range iidInfoList {
                if OneIIdInfo.IId.NameId == clusterName {
                        iidInfo = OneIIdInfo
                        bool_ret = true
                        break;
                }
        }
        if bool_ret == false {
                err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsCluster), clusterName)
                cblog.Error(err)
                return nil, err
        }

        // (1) check exist(NameID)
        ngIIdInfoList, err := getAllNodeGroupIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        var isExist bool=false
        for _, OneIIdInfo := range ngIIdInfoList {
                if OneIIdInfo.IId.NameId == reqInfo.IId.NameId {
                        isExist = true
                }
        }

        if isExist == true {
                err :=  fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
                cblog.Error(err)
                return nil, err
        }

        // refine RootDisk and RootDiskSize in reqInfo(NodeGroupInfo)
        translateRootDiskInfo(providerName, &reqInfo)


        nodeGroupUUID, err := iidm.New(connectionName, rsNodeGroup, reqInfo.IId.NameId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // driverIID
        nodeGroupNameId := reqInfo.IId.NameId
        driverIId := cres.IID{nodeGroupUUID, ""}
        reqInfo.IId = driverIId

        // (2) add a NodeGroup into CSP
        ngInfo, err := handler.AddNodeGroup(getDriverIID(iidInfo.IId), reqInfo) 
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        ngSpiderIId := cres.IID{nodeGroupNameId, nodeGroupUUID + ":" + ngInfo.IId.SystemId}
        _, err2 := iidRWLock.CreateIID(iidm.NGGROUP, connectionName, clusterName, ngSpiderIId) // clusterName => rsType
        if err2 != nil {
                cblog.Error(err2)
                // rollback
                // (1) for resource
                cblog.Info("<<ROLLBACK:TRY:NODEGROUP-CSP>> " + ngInfo.IId.SystemId)
                _, err3 := handler.RemoveNodeGroup(getDriverIID(iidInfo.IId),  ngInfo.IId)
                if err3 != nil {
                        cblog.Error(err3)
                        return nil, fmt.Errorf(err2.Error() + ", " + err3.Error())
                }
                return nil, err2
        }

        // (3) Get ClusterInfo
        info, err := handler.GetCluster(getDriverIID(iidInfo.IId))
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // (4) set ResourceInfo(IID.NameId)
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

func translateRootDiskInfo(providerName string, reqInfo *cres.NodeGroupInfo) error {

        // get Provider's Meta Info
        cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo(providerName)
        if err != nil {
                cblog.Error(err)
                return err
        }

        // for Root Disk Type
        switch strings.ToUpper(reqInfo.RootDiskType) {
        case "", "DEFAULT": // bypass
                reqInfo.RootDiskType = ""
        default: // TYPE1, TYPE2, TYPE3, ... or "pd-balanced", check validation, bypass
                // TYPE2, ...
                if strings.Contains(strings.ToUpper(reqInfo.RootDiskType), "TYPE") {
                        strType := strings.ToUpper(reqInfo.RootDiskType)
                        typeNum, _ := strconv.Atoi(strings.Replace(strType, "TYPE", "", -1)) // "TYPE2" => "2" => 2
                        typeMax := len(cloudOSMetaInfo.RootDiskType)
                        if typeNum > typeMax {
                                typeNum = typeMax
                        }
                        reqInfo.RootDiskType = cloudOSMetaInfo.RootDiskType[typeNum-1]
                } else if !validateRootDiskType(reqInfo.RootDiskType, cloudOSMetaInfo.RootDiskType) {
                        errMSG :=reqInfo.RootDiskType + " is not a valid Root Disk Type of " + providerName + "!"
                        cblog.Error(errMSG)
                        return fmt.Errorf(errMSG)
                }
        }


        // for Root Disk Size
        switch strings.ToUpper(reqInfo.RootDiskSize) {
        case "", "DEFAULT": // bypass
                reqInfo.RootDiskSize = ""
        default: // "100", bypass
                err := validateRootDiskSize(reqInfo.RootDiskSize)
                if err != nil {
                        errMSG :=reqInfo.RootDiskSize + " is not a valid Root Disk Size: " + err.Error() + "!"
                        cblog.Error(errMSG)
                        return fmt.Errorf(errMSG)
                }
        }
        return nil
}

func SetNodeGroupAutoScaling(connectionName string, clusterName string, nodeGroupName string, on bool) (bool, error) {
       cblog.Info("call SetNodeGroupAutoScaling()")

        // check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        clusterName, err = EmptyCheckAndTrim("clusterName", clusterName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        nodeGroupName, err = EmptyCheckAndTrim("nodeGroupName", nodeGroupName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        handler, err := cldConn.CreateClusterHandler()
        if err != nil {
                cblog.Error(err)
                return false, err
        }

clusterSPLock.Lock(connectionName, clusterName)
defer clusterSPLock.Unlock(connectionName, clusterName)

       // (1) Check the Cluster existence(clusetName) and Get the Cluster's DriverIID and the NodeGroup's DriverIID
        cluserDriverIID, nodeGroupDriverIID, err := getClusterDriverIIDNodeGroupDriverIID(connectionName, clusterName, nodeGroupName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        // (2) Set the NodeGroup AutoScaling On/Off
        boolRet, err := handler.SetNodeGroupAutoScaling(cluserDriverIID, nodeGroupDriverIID, on) 
        if err != nil {
                cblog.Error(err)
                return false, err
        }
 
	return boolRet, nil
}

func getClusterDriverIIDNodeGroupDriverIID(connectionName string, clusterName string, nodeGroupName string) (cres.IID, cres.IID, error) {

        // (1) Check the Cluster existence(clusetName) and Get the Cluster's DriverIID
        clusterDriverIID, err := getClusterDriverIID(connectionName, clusterName)
        if err != nil {
                cblog.Error(err)
                return cres.IID{}, cres.IID{}, err
        }

        // (2) Get NodeGroup's DriverIID
        ngIIdInfoList, err := getAllNodeGroupIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return cres.IID{}, cres.IID{}, err
        }
        var ngIIdInfo *iidm.IIDInfo
        var bool_ret = false
        for _, OneIIdInfo := range ngIIdInfoList {
                if OneIIdInfo.IId.NameId == nodeGroupName {
                        ngIIdInfo = OneIIdInfo
                        bool_ret = true
                        break;
                }
        }
        if bool_ret == false {
                err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsNodeGroup), nodeGroupName)
                cblog.Error(err)
                return cres.IID{}, cres.IID{}, err
        }

        return clusterDriverIID, getDriverIID(ngIIdInfo.IId), nil
}

// Check the Cluster existence(clusetName) and Get the Cluster's DriverIID
func getClusterDriverIID(connectionName string, clusterName string) (cres.IID, error) {

        // (1) Get Cluster's SpiderIID
        iidInfoList, err := getAllClusterIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return cres.IID{}, err
        }
        var iidInfo *iidm.IIDInfo
        var bool_ret = false
        for _, OneIIdInfo := range iidInfoList {
                if OneIIdInfo.IId.NameId == clusterName {
                        iidInfo = OneIIdInfo
                        bool_ret = true
                        break;
                }
        }
        if bool_ret == false {
                err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsCluster), clusterName)
                cblog.Error(err)
                return cres.IID{}, err
        }

        return getDriverIID(iidInfo.IId), nil
}

func getAllNodeGroupIIDInfoList(connectionName string) ([]*iidm.IIDInfo, error) {

        // (1) Get VPC's Name List
        // format) /resource-info-spaces/{iidGroup}/{connectionName}/{resourceType}/{resourceName} [{resourceID}]
        clusterNameList, err := iidRWLock.ListResourceType(iidm.NGGROUP, connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        clusterNameList = uniqueNameList(clusterNameList)
        // (2) Create All NodeGroup's IIDInfo List
        iidInfoList := []*iidm.IIDInfo{}
        for _, clusterName := range clusterNameList {
                ngIIDInfoList, err := iidRWLock.ListIID(iidm.NGGROUP, connectionName, clusterName)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                iidInfoList = append(iidInfoList, ngIIDInfoList...)
        }
        return iidInfoList, nil
}

func ChangeNodeGroupScaling(connectionName string, clusterName string, nodeGroupName string, 
	DesiredNodeSize int, MinNodeSize int, MaxNodeSize int) (cres.NodeGroupInfo, error) {
        cblog.Info("call ChangeNodeGroupScaling()")

        // check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                cblog.Error(err)
                return cres.NodeGroupInfo{}, err
        }

        clusterName, err = EmptyCheckAndTrim("clusterName", clusterName)
        if err != nil {
                cblog.Error(err)
                return cres.NodeGroupInfo{}, err
        }

        nodeGroupName, err = EmptyCheckAndTrim("nodeGroupName", nodeGroupName)
        if err != nil {
                cblog.Error(err)
                return cres.NodeGroupInfo{}, err
        }

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return cres.NodeGroupInfo{}, err
        }

        handler, err := cldConn.CreateClusterHandler()
        if err != nil {
                cblog.Error(err)
                return cres.NodeGroupInfo{}, err
        }

clusterSPLock.Lock(connectionName, clusterName)
defer clusterSPLock.Unlock(connectionName, clusterName)

        // (1) Check the Cluster existence(clusetName) and Get the Cluster's DriverIID and the NodeGroup's DriverIID
        cluserDriverIID, nodeGroupDriverIID, err := getClusterDriverIIDNodeGroupDriverIID(connectionName, clusterName, nodeGroupName)
        if err != nil {
                cblog.Error(err)
                return cres.NodeGroupInfo{}, err
        }

        // (2) Change NodeGroup Scaling Size
        ngInfo, err := handler.ChangeNodeGroupScaling(cluserDriverIID, nodeGroupDriverIID, DesiredNodeSize, MinNodeSize, MaxNodeSize) 
        if err != nil {
                cblog.Error(err)
                return cres.NodeGroupInfo{}, err
        }

        // ++++++++++++++++++
        // (1) NodeGroup IID
        ngIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.NGGROUP, connectionName, clusterName, ngInfo.IId)
        if err != nil {
                cblog.Error(err)
                return cres.NodeGroupInfo{}, err
        }
        ngInfo.IId.NameId = ngIIDInfo.IId.NameId

        // (2) ImageIID
        ngInfo.ImageIID.NameId = ngInfo.ImageIID.SystemId

        // (3) KeyPair
        keyIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, connectionName, rsKey, ngInfo.KeyPairIID)
        if err != nil {
                cblog.Error(err)
                return cres.NodeGroupInfo{}, err
        }
        ngInfo.KeyPairIID.NameId = keyIIDInfo.IId.NameId

	return ngInfo, nil      
}

func RemoveNodeGroup(connectionName string, clusterName string, nodeGroupName string, force string) (bool, error) {
        cblog.Info("call RemoveNodeGroup()")

        // check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        clusterName, err = EmptyCheckAndTrim("clusterName", clusterName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        nodeGroupName, err = EmptyCheckAndTrim("nodeGroupName", nodeGroupName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        handler, err := cldConn.CreateClusterHandler()
        if err != nil {
                cblog.Error(err)
                return false, err
        }

clusterSPLock.Lock(connectionName, clusterName)
defer clusterSPLock.Unlock(connectionName, clusterName)

       // (1) Check the Cluster existence(clusetName) and Get the Cluster's DriverIID and the NodeGroup's DriverIID
        cluserDriverIID, nodeGroupDriverIID, err := getClusterDriverIIDNodeGroupDriverIID(connectionName, clusterName, nodeGroupName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        // (2) Remove the NodeGroup from the Cluster
        result, err := handler.RemoveNodeGroup(cluserDriverIID, nodeGroupDriverIID) 
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
        _, err = iidRWLock.DeleteIID(iidm.NGGROUP, connectionName, clusterName, cres.IID{nodeGroupName, ""})
        if err != nil {
                cblog.Error(err)
                if force != "true" {
                        return false, err
                }
        }

        return result, nil
}

func RemoveCSPNodeGroup(connectionName string, clusterName string, systemID string) (bool, error) {
        cblog.Info("call RemoveNodeGroup()")

        // check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        clusterName, err = EmptyCheckAndTrim("clusterName", clusterName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        systemID, err = EmptyCheckAndTrim("systemID", systemID)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        // Check the Cluster existence(clusetName) and Get the Cluster's DriverIID
        clusterDriverIID, err := getClusterDriverIID(connectionName, clusterName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        handler, err := cldConn.CreateClusterHandler()
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        iid := cres.IID{"", systemID}

        // delete Resource(SystemId)
        result := false
        result, err = handler.(cres.ClusterHandler).RemoveNodeGroup(clusterDriverIID, iid)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        return result, nil
}

func UpgradeCluster(connectionName string, clusterName string, newVersion string) (cres.ClusterInfo, error) {
        cblog.Info("call UpgradeCluster()")

        // check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                cblog.Error(err)
                return cres.ClusterInfo{}, err
        }

        clusterName, err = EmptyCheckAndTrim("clusterName", clusterName)
        if err != nil {
                cblog.Error(err)
                return cres.ClusterInfo{}, err
        }

        newVersion, err = EmptyCheckAndTrim("newVersion", newVersion)
        if err != nil {
                cblog.Error(err)
                return cres.ClusterInfo{}, err
        }

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return cres.ClusterInfo{}, err
        }

        handler, err := cldConn.CreateClusterHandler()
        if err != nil {
                cblog.Error(err)
                return cres.ClusterInfo{}, err
        }

clusterSPLock.Lock(connectionName, clusterName)
defer clusterSPLock.Unlock(connectionName, clusterName)

       // (1) Check the Cluster existence(clusetName) and Get the Cluster's DriverIID
        cluserDriverIID, err := getClusterDriverIID(connectionName, clusterName)
        if err != nil {
                cblog.Error(err)
                return cres.ClusterInfo{}, err
        }

        // (2) Upgrade the Cluster
        clusterInfo, err := handler.UpgradeCluster(cluserDriverIID, newVersion) 
        if err != nil {
                cblog.Error(err)
                return cres.ClusterInfo{}, err
        }

        // ++++++++++++++++++
        // set ClusterIID
        clusterInfo.IId.NameId = clusterName

        // set used Resources's userIID
        err = setResourcesNameId(connectionName, &clusterInfo)
        if err != nil {
                cblog.Error(err)
                return cres.ClusterInfo{}, err
        }

        return clusterInfo, nil
}
