// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2022.10.

package commonruntime

import (
	_ "errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
	infostore "github.com/cloud-barista/cb-spider/info-store"
)

// ====================================================================
// type for GORM
type ClusterIIDInfo VPCDependentIIDInfo

func (ClusterIIDInfo) TableName() string {
	return "cluster_iid_infos"
}

type NodeGroupIIDInfo ClusterDependentIIDInfo

func (NodeGroupIIDInfo) TableName() string {
	return "node_group_iid_infos"
}

//====================================================================

func init() {
	db, err := infostore.Open()
	if err != nil {
		cblog.Error(err)
		return
	}
	db.AutoMigrate(&ClusterIIDInfo{})
	db.AutoMigrate(&NodeGroupIIDInfo{})
	infostore.Close(db)
}

//================ Cluster Handler

func GetClusterOwnerVPC(connectionName string, cspID string) (owerVPC cres.IID, err error) {
	cblog.Info("call GetClusterOwnerVPC()")

	// check empty and trim user inputs
	connectionName, err = EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return cres.IID{}, err
	}

	if err := checkCapability(connectionName, CLUSTER_HANDLER); err != nil {
		return cres.IID{}, err
	}

	cspID, err = EmptyCheckAndTrim("cspID", cspID)
	if err != nil {
		cblog.Error(err)
		return cres.IID{}, err
	}

	rsType := CLUSTER

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
	var iidInfoList []*ClusterIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			//vpcSPLock.RUnlock()
			//clusterSPLock.RUnlock()
			cblog.Error(err)
			return cres.IID{}, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			//vpcSPLock.RUnlock()
			//clusterSPLock.RUnlock()
			cblog.Error(err)
			return cres.IID{}, err
		}
	}

	var isExist bool = false
	var nameId string
	for _, OneIIdInfo := range iidInfoList {
		if getMSShortID(getDriverSystemId(cres.IID{NameId: OneIIdInfo.NameId, SystemId: OneIIdInfo.SystemId})) == cspID {
			nameId = OneIIdInfo.NameId
			isExist = true
			break
		}
	}
	if isExist {
		//vpcSPLock.RUnlock()
		//clusterSPLock.RUnlock()
		err := fmt.Errorf(rsType + "-" + cspID + " already exists with " + nameId + "!")
		cblog.Error(err)
		return cres.IID{}, err
	}

	// (2) get resource info(CSP-ID)
	// check existence and get info of this resouce in the CSP
	// Do not user NameId, because Azure driver use it like SystemId
	getInfo, err := handler.GetCluster(cres.IID{NameId: getMSShortID(cspID), SystemId: cspID})
	if err != nil {
		//vpcSPLock.RUnlock()
		//clusterSPLock.RUnlock()
		cblog.Error(err)
		return cres.IID{}, err
	}

	// (3) get VPC IID:list
	var vpcIIDInfoList []*VPCIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &vpcIIDInfoList)
		if err != nil {
			//vpcSPLock.RUnlock()
			//clusterSPLock.RUnlock()
			cblog.Error(err)
			return cres.IID{}, err
		}
	} else {
		err = infostore.ListByCondition(&vpcIIDInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			//vpcSPLock.RUnlock()
			//clusterSPLock.RUnlock()
			cblog.Error(err)
			return cres.IID{}, err
		}
	}
	//vpcSPLock.RUnlock()
	//clusterSPLock.RUnlock()

	//--------
	//-------- ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	//--------
	// Do not user NameId, because Azure driver use it like SystemId
	vpcCSPID := getMSShortID(getInfo.Network.VpcIID.SystemId)
	if vpcIIDInfoList == nil || len(vpcIIDInfoList) <= 0 {
		return cres.IID{NameId: "", SystemId: vpcCSPID}, nil
	}

	// (4) check existence in the MetaDB
	for _, one := range vpcIIDInfoList {
		if getMSShortID(getDriverSystemId(cres.IID{NameId: one.NameId, SystemId: one.SystemId})) == vpcCSPID {
			return cres.IID{NameId: one.NameId, SystemId: vpcCSPID}, nil
		}
	}

	return cres.IID{NameId: "", SystemId: vpcCSPID}, nil
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

	if err := checkCapability(connectionName, CLUSTER_HANDLER); err != nil {
		return nil, err
	}

	vpcUserID, err = EmptyCheckAndTrim("vpcUserID", vpcUserID)
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

	rsType := CLUSTER

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
	var bool_ret bool
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		// check permission to vpcName
		var iidInfoList []*VPCIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		bool_ret, err = isNameIdExists(&iidInfoList, vpcUserID)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		bool_ret, err = infostore.HasByConditions(&VPCIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, vpcUserID)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	if !bool_ret {
		err := fmt.Errorf("The %s '%s' does not exist!", RSTypeString(VPC), vpcUserID)
		cblog.Error(err)
		return nil, err
	}

	// (1) check existence(UserID)
	var isExist bool
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		isExist, err = infostore.HasByCondition(&ClusterIIDInfo{}, NAME_ID_COLUMN, userIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		isExist, err = infostore.HasByConditions(&ClusterIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, userIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	if isExist {
		err := fmt.Errorf(rsType + "-" + userIID.NameId + " already exists!")
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource info(CSP-ID)
	// check existence and get info of this resouce in the CSP
	// Do not user NameId, because Azure driver use it like SystemId
	getInfo, err := handler.GetCluster(cres.IID{NameId: getMSShortID(userIID.SystemId), SystemId: userIID.SystemId})
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
	// insert Cluster SpiderIID to metadb
	err = infostore.Insert(&ClusterIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId,
		OwnerVPCName: vpcUserID})
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
		ngSpiderIId := cres.IID{NameId: ngUserId, SystemId: systemId + ":" + ngInfo.IId.SystemId}
		err := infostore.Insert(&NodeGroupIIDInfo{ConnectionName: connectionName, NameId: ngSpiderIId.NameId, SystemId: ngSpiderIId.SystemId,
			OwnerClusterName: info.IId.NameId})
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
func CreateCluster(connectionName string, rsType string, reqInfo cres.ClusterInfo, IDTransformMode string) (*cres.ClusterInfo, error) {
	cblog.Info("call CreateCluster()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if err := checkCapability(connectionName, CLUSTER_HANDLER); err != nil {
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
	var vpcIIDInfo VPCIIDInfo
	if netReqInfo.VpcIID.NameId != "" {
		// get spiderIID
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*VPCIIDInfo
			err = getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, netReqInfo.VpcIID.NameId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			vpcIIDInfo = *castedIIDInfo.(*VPCIIDInfo)
		} else {
			err = infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, netReqInfo.VpcIID.NameId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		}
		// set driverIID
		netReqInfo.VpcIID = getDriverIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})
	}

	// (2) SubnetIIDs
	for idx, subnetIID := range netReqInfo.SubnetIIDs {
		var subnetIIdInfo SubnetIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			// 1. get VPC IIDInfo
			var iidInfoList []*VPCIIDInfo
			err = getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, vpcIIDInfo.NameId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			vpcIIDInfo := *castedIIDInfo.(*VPCIIDInfo)

			// 2. get Subnet IIDInfo
			err = infostore.GetBy3Conditions(&subnetIIdInfo, CONNECTION_NAME_COLUMN, vpcIIDInfo.ConnectionName, NAME_ID_COLUMN, subnetIID.NameId, OWNER_VPC_NAME_COLUMN, vpcIIDInfo.NameId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		} else {
			err = infostore.GetBy3Conditions(&subnetIIdInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, subnetIID.NameId, OWNER_VPC_NAME_COLUMN, vpcIIDInfo.NameId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		}
		// set driverIID
		netReqInfo.SubnetIIDs[idx] = getDriverIID(cres.IID{NameId: subnetIIdInfo.NameId, SystemId: subnetIIdInfo.SystemId})
	}

	// (3) SecurityGroupIIDs
	for idx, sgIID := range netReqInfo.SecurityGroupIIDs {
		sgSPLock.RLock(connectionName, sgIID.NameId)
		defer sgSPLock.RUnlock(connectionName, sgIID.NameId)
		var sgIIdInfo SGIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*SGIIDInfo
			err := getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, sgIID.NameId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			sgIIdInfo = *castedIIDInfo.(*SGIIDInfo)
		} else {
			err = infostore.GetByConditions(&sgIIdInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, sgIID.NameId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		}
		// set driverIID
		netReqInfo.SecurityGroupIIDs[idx] = getDriverIID(cres.IID{NameId: sgIIdInfo.NameId, SystemId: sgIIdInfo.SystemId})
	}

	//+++++++++++++++++++++ Set NodeGroupInfo's SystemId
	for idx, ngInfo := range reqInfo.NodeGroupList {
		// (1) ImageIID
		reqInfo.NodeGroupList[idx].ImageIID.SystemId = ngInfo.ImageIID.NameId

		// (2) KeyPair
		keySPLock.RLock(connectionName, ngInfo.KeyPairIID.NameId)
		defer keySPLock.RUnlock(connectionName, ngInfo.KeyPairIID.NameId)

		var keyIIDInfo KeyIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*KeyIIDInfo
			err := getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, ngInfo.KeyPairIID.NameId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			keyIIDInfo = *castedIIDInfo.(*KeyIIDInfo)
		} else {
			err := infostore.GetByConditions(&keyIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, ngInfo.KeyPairIID.NameId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		}
		reqInfo.NodeGroupList[idx].KeyPairIID = getDriverIID(cres.IID{NameId: keyIIDInfo.NameId, SystemId: keyIIDInfo.SystemId})
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
	var iidInfoList []*ClusterIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = infostore.List(&iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	var isExist bool = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == reqInfo.IId.NameId {
			isExist = true
		}
	}

	if isExist {
		err := fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
		cblog.Error(err)
		return nil, err
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

	// Create SP-UUID for NodeGroup list
	ngReqIIdList := []cres.IID{}
	ngInfoList := []cres.NodeGroupInfo{}
	for idx, info := range reqInfo.NodeGroupList {
		nodeGroupUUID := ""
		if providerName == "NHN" && idx == 0 {
			nodeGroupUUID = "default-worker" // fixed name in NHN
		} else {
			if GetID_MGMT(IDTransformMode) == "ON" { // Use IID Management
				nodeGroupUUID, err = iidm.New(connectionName, NODEGROUP, info.IId.NameId)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
			} else { // No Use IID Management
				nodeGroupUUID = info.IId.NameId
			}
		}

		// reqIID
		ngReqIId := cres.IID{NameId: info.IId.NameId, SystemId: nodeGroupUUID}
		ngReqIIdList = append(ngReqIIdList, ngReqIId)
		// driverIID
		ngDriverIId := cres.IID{NameId: nodeGroupUUID, SystemId: ""}
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
	spiderIId := cres.IID{NameId: reqIId.NameId, SystemId: spUUID + ":" + info.IId.SystemId}

	// (5) insert spiderIID
	iidInfo := ClusterIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId,
		OwnerVPCName: vpcIIDInfo.NameId}
	err = infostore.Insert(&iidInfo)
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
			continue
		}
		ngSpiderIId := cres.IID{NameId: ngReqNameId, SystemId: ngInfo.IId.NameId + ":" + ngInfo.IId.SystemId}
		err := infostore.Insert(&NodeGroupIIDInfo{ConnectionName: connectionName, NameId: ngSpiderIId.NameId, SystemId: ngSpiderIId.SystemId,
			OwnerClusterName: reqIId.NameId})
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
			_, err3 := infostore.DeleteByConditions(&ClusterIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, iidInfo.NameId)
			if err3 != nil {
				cblog.Error(err3)
				return nil, fmt.Errorf(err.Error() + ", " + err3.Error())
			}
			// (3) for NodeGroup IID List
			// delete all nodegroups of target Cluster
			_, err := infostore.DeleteByConditions(&NodeGroupIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName,
				OWNER_CLUSTER_NAME_COLUMN, iidInfo.NameId)
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
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

	// (7) set used Resources's userIID
	err = setResourcesNameId(connectionName, &info)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

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

func setResourcesNameId(connectionName string, info *cres.ClusterInfo) error {
	//+++++++++++++++++++++ Set NetworkInfo's NameId
	netInfo := &info.Network
	// (1) VpcIID
	// get spiderIID
	var vpcIIDInfo VPCIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*VPCIIDInfo
		err := getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return err
		}
		castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, netInfo.VpcIID.SystemId)
		if err != nil {
			cblog.Error(err)
			return err
		}
		vpcIIDInfo = *castedIIDInfo.(*VPCIIDInfo)
	} else {
		err := infostore.GetByContain(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, netInfo.VpcIID.SystemId)
		if err != nil {
			cblog.Error(err)
			return err
		}
	}
	// set NameId
	netInfo.VpcIID.NameId = vpcIIDInfo.NameId

	// (2) SubnetIIDs
	for idx, subnetIID := range netInfo.SubnetIIDs {
		var subnetIIdInfo SubnetIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			err := infostore.GetByConditionsAndContain(&subnetIIdInfo, CONNECTION_NAME_COLUMN, vpcIIDInfo.ConnectionName,
				OWNER_VPC_NAME_COLUMN, vpcIIDInfo.NameId, SYSTEM_ID_COLUMN, subnetIID.SystemId)
			if err != nil {
				cblog.Error(err)
				return err
			}
		} else {
			err := infostore.GetByConditionsAndContain(&subnetIIdInfo, CONNECTION_NAME_COLUMN, connectionName,
				OWNER_VPC_NAME_COLUMN, vpcIIDInfo.NameId, SYSTEM_ID_COLUMN, subnetIID.SystemId)
			if err != nil {
				cblog.Error(err)
				return err
			}
		}
		// set NameId
		netInfo.SubnetIIDs[idx].NameId = subnetIIdInfo.NameId
	}

	// (3) SecurityGroupIIDs
	for idx, sgIID := range netInfo.SecurityGroupIIDs {
		var sgIIdInfo SGIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*SGIIDInfo
			err := getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return err
			}
			castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, sgIID.SystemId)
			if err != nil {
				providerName, getErr := ccm.GetProviderNameByConnectionName(connectionName)
				if getErr != nil {
					cblog.Error(getErr)
					return getErr
				}

				// exception processing to Azure or NHN, we create new SG for K8S.
				if strings.Contains(sgIID.SystemId, "/Microsoft.") && strings.Contains(err.Error(), "not exist") {
					sgIIdInfo.NameId = "#" + getMSShortID(sgIID.SystemId)
				} else if strings.EqualFold(providerName, "NHN") && strings.Contains(err.Error(), "not exist") {
					sgIIdInfo.NameId = "#" + sgIID.SystemId
				} else {
					cblog.Error(err)
					return err
				}
			}
			sgIIdInfo = *castedIIDInfo.(*SGIIDInfo)
		} else {
			err := infostore.GetByConditionsAndContain(&sgIIdInfo, CONNECTION_NAME_COLUMN, connectionName,
				OWNER_VPC_NAME_COLUMN, netInfo.VpcIID.NameId, SYSTEM_ID_COLUMN, sgIID.SystemId)
			if err != nil {
				providerName, getErr := ccm.GetProviderNameByConnectionName(connectionName)
				if getErr != nil {
					cblog.Error(getErr)
					return getErr
				}

				// exception processing to Azure or NHN, we create new SG for K8S.
				if strings.Contains(sgIID.SystemId, "/Microsoft.") && strings.Contains(err.Error(), "not exist") {
					sgIIdInfo.NameId = "#" + getMSShortID(sgIID.SystemId)
				} else if strings.EqualFold(providerName, "NHN") && strings.Contains(err.Error(), "not exist") {
					sgIIdInfo.NameId = "#" + sgIID.SystemId
				} else {
					cblog.Error(err)
					return err
				}
			}
		}
		// set NameId
		netInfo.SecurityGroupIIDs[idx].NameId = sgIIdInfo.NameId
	}

	//+++++++++++++++++++++ Set NodeGroupInfo's NameId
	for idx, ngInfo := range info.NodeGroupList {
		// when deleting, MetaDB has no IID.
		// because It deleted asynchronous before deletion.
		if ngInfo.Status == cres.NodeGroupDeleting {
			continue
		}
		// (1) NodeGroup IID
		var ngIIDInfo NodeGroupIIDInfo
		hasNodeGroup := true
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*NodeGroupIIDInfo
			err := getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return err
			}
			castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, ngInfo.IId.SystemId)
			if err != nil {
				if checkNotFoundError(err) {
					hasNodeGroup = false
				} else {
					cblog.Error(err)
					return err
				}
			}
			ngIIDInfo = *castedIIDInfo.(*NodeGroupIIDInfo)
		} else {
			err := infostore.GetByConditionsAndContain(&ngIIDInfo, CONNECTION_NAME_COLUMN, connectionName,
				OWNER_CLUSTER_NAME_COLUMN, info.IId.NameId, SYSTEM_ID_COLUMN, ngInfo.IId.SystemId)
			if err != nil {
				if checkNotFoundError(err) {
					hasNodeGroup = false
				} else {
					cblog.Error(err)
					return err
				}
			}
		}
		if hasNodeGroup {
			info.NodeGroupList[idx].IId.NameId = ngIIDInfo.NameId
		}

		// (2) ImageIID
		info.NodeGroupList[idx].ImageIID.NameId = ngInfo.ImageIID.SystemId

		// (3) KeyPair
		var keyIIDInfo KeyIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*KeyIIDInfo
			err := getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return err
			}
			// Have to use getMSShortID()
			// because Azure has different uri of keypair SystemID.
			// ex) /.../Microsoft.Network/.../ID vs /.../Microsoft.Compute/.../ID
			castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, ngInfo.KeyPairIID.SystemId)
			if err != nil {
				cblog.Error(err)
				return err
			}
			keyIIDInfo = *castedIIDInfo.(*KeyIIDInfo)
		} else {
			// Have to use getMSShortID()
			// because Azure has different uri of keypair SystemID.
			// ex) /.../Microsoft.Network/.../ID vs /.../Microsoft.Compute/.../ID
			err := infostore.GetByContain(&keyIIDInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, ngInfo.KeyPairIID.SystemId)
			if err != nil {
				cblog.Error(err)
				return err
			}
		}
		info.NodeGroupList[idx].KeyPairIID.NameId = keyIIDInfo.NameId

		// (4) Set Nodes' NameId to SystemId if NameId is empty
		for nodeIdx, nodeInfo := range ngInfo.Nodes {
			if nodeInfo.NameId == "" {
				info.NodeGroupList[idx].Nodes[nodeIdx].NameId = nodeInfo.SystemId
			}
		}
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

	if err := checkCapability(connectionName, CLUSTER_HANDLER); err != nil {
		return nil, err
	}

	// (1) get IID:list
	var iidInfoList []*ClusterIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
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

	// (2) Get ClusterInfo-list with IID-list
	infoList2 := []*cres.ClusterInfo{}
	for _, iidInfo := range iidInfoList {

		clusterSPLock.RLock(connectionName, iidInfo.NameId)

		// get resource(SystemId)
		info, err := handler.GetCluster(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
		if err != nil {
			clusterSPLock.RUnlock(connectionName, iidInfo.NameId)
			if checkNotFoundError(err) {
				cblog.Error(err)
				info = cres.ClusterInfo{IId: cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}}
				infoList2 = append(infoList2, &info)
				continue
			}
			cblog.Error(err)
			return nil, err
		}
		clusterSPLock.RUnlock(connectionName, iidInfo.NameId)

		// (3) set ResourceInfo(IID.NameId)
		// set ResourceInfo
		info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

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

	if err := checkCapability(connectionName, CLUSTER_HANDLER); err != nil {
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
	var iidInfoList []*ClusterIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	var iidInfo *ClusterIIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == clusterName {
			iidInfo = OneIIdInfo
			bool_ret = true
			break
		}
	}
	if bool_ret == false {
		err := fmt.Errorf("The %s '%s' does not exist!", RSTypeString(rsType), clusterName)
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource(SystemId)
	info, err := handler.GetCluster(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) set ResourceInfo(IID.NameId)
	// set ResourceInfo
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

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
func AddNodeGroup(connectionName string, rsType string, clusterName string, reqInfo cres.NodeGroupInfo, IDTransformMode string) (*cres.ClusterInfo, error) {
	cblog.Info("call AddNodeGroup()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if err := checkCapability(connectionName, CLUSTER_HANDLER); err != nil {
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

	var keyIIDInfo KeyIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*KeyIIDInfo
		err := getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, reqInfo.KeyPairIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		keyIIDInfo = *castedIIDInfo.(*KeyIIDInfo)
	} else {
		err = infostore.GetByConditions(&keyIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, reqInfo.KeyPairIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	reqInfo.KeyPairIID = getDriverIID(cres.IID{NameId: keyIIDInfo.NameId, SystemId: keyIIDInfo.SystemId})
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
	var iidInfoList []*ClusterIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	var iidInfo *ClusterIIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == clusterName {
			iidInfo = OneIIdInfo
			bool_ret = true
			break
		}
	}
	if !bool_ret {
		err := fmt.Errorf("The %s '%s' does not exist!", RSTypeString(CLUSTER), clusterName)
		cblog.Error(err)
		return nil, err
	}

	// (1) check exist(NameID)
	isExist, err := infostore.HasBy3Conditions(&NodeGroupIIDInfo{}, CONNECTION_NAME_COLUMN, iidInfo.ConnectionName, OWNER_CLUSTER_NAME_COLUMN, clusterName, NAME_ID_COLUMN, reqInfo.IId.NameId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if isExist {
		err := fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
		cblog.Error(err)
		return nil, err
	}

	// refine RootDisk and RootDiskSize in reqInfo(NodeGroupInfo)
	translateRootDiskInfo(providerName, &reqInfo)

	nodeGroupUUID := ""
	if GetID_MGMT(IDTransformMode) == "ON" { // Use IID Management
		nodeGroupUUID, err = iidm.New(connectionName, NODEGROUP, reqInfo.IId.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else { // No Use IID Management
		nodeGroupUUID = reqInfo.IId.NameId
	}

	// driverIID
	nodeGroupNameId := reqInfo.IId.NameId
	driverIId := cres.IID{NameId: nodeGroupUUID, SystemId: ""}
	reqInfo.IId = driverIId

	// (2) add a NodeGroup into CSP
	ngInfo, err := handler.AddNodeGroup(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}), reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	ngSpiderIId := cres.IID{NameId: nodeGroupNameId, SystemId: nodeGroupUUID + ":" + ngInfo.IId.SystemId}
	err2 := infostore.Insert(&NodeGroupIIDInfo{ConnectionName: iidInfo.ConnectionName, NameId: ngSpiderIId.NameId, SystemId: ngSpiderIId.SystemId,
		OwnerClusterName: clusterName})
	if err2 != nil {
		cblog.Error(err2)
		// rollback
		// (1) for resource
		cblog.Info("<<ROLLBACK:TRY:NODEGROUP-CSP>> " + ngInfo.IId.SystemId)
		_, err3 := handler.RemoveNodeGroup(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}), ngInfo.IId)
		if err3 != nil {
			cblog.Error(err3)
			return nil, fmt.Errorf(err2.Error() + ", " + err3.Error())
		}
		return nil, err2
	}

	// (3) Get ClusterInfo
	info, err := handler.GetCluster(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (4) set ResourceInfo(IID.NameId)
	// set ResourceInfo
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

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
			errMSG := reqInfo.RootDiskType + " is not a valid Root Disk Type of " + providerName + "!"
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
			errMSG := reqInfo.RootDiskSize + " is not a valid Root Disk Size: " + err.Error() + "!"
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

	if err := checkCapability(connectionName, CLUSTER_HANDLER); err != nil {
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
	var ngIIdInfoList []*NodeGroupIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		// 1. get Cluster IIDInfo
		var iidInfoList []*ClusterIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return cres.IID{}, cres.IID{}, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, clusterName)
		if err != nil {
			cblog.Error(err)
			return cres.IID{}, cres.IID{}, err
		}
		clusterIIDInfo := *castedIIDInfo.(*ClusterIIDInfo)

		// 2. get Nodegroup Info List in ConnectionName of Cluster IIDInfo
		err = infostore.ListByConditions(&ngIIdInfoList, CONNECTION_NAME_COLUMN, clusterIIDInfo.ConnectionName, OWNER_CLUSTER_NAME_COLUMN, clusterName)
		if err != nil {
			cblog.Error(err)
			return cres.IID{}, cres.IID{}, err
		}
	} else {
		err = infostore.ListByConditions(&ngIIdInfoList, CONNECTION_NAME_COLUMN, connectionName, OWNER_CLUSTER_NAME_COLUMN, clusterName)
		if err != nil {
			cblog.Error(err)
			return cres.IID{}, cres.IID{}, err
		}
	}
	var ngIIdInfo *NodeGroupIIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range ngIIdInfoList {
		if OneIIdInfo.NameId == nodeGroupName {
			ngIIdInfo = OneIIdInfo
			bool_ret = true
			break
		}
	}
	if !bool_ret {
		err := fmt.Errorf("The %s '%s' does not exist!", RSTypeString(NODEGROUP), nodeGroupName)
		cblog.Error(err)
		return cres.IID{}, cres.IID{}, err
	}

	return clusterDriverIID, getDriverIID(cres.IID{NameId: ngIIdInfo.NameId, SystemId: ngIIdInfo.SystemId}), nil
}

// Check the Cluster existence(clusetName) and Get the Cluster's DriverIID
func getClusterDriverIID(connectionName string, clusterName string) (cres.IID, error) {

	// (1) Get Cluster's SpiderIID
	var iidInfoList []*ClusterIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err := getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return cres.IID{}, err
		}
	} else {
		err := infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return cres.IID{}, err
		}
	}
	var iidInfo *ClusterIIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == clusterName {
			iidInfo = OneIIdInfo
			bool_ret = true
			break
		}
	}
	if !bool_ret {
		err := fmt.Errorf("The %s '%s' does not exist!", RSTypeString(CLUSTER), clusterName)
		cblog.Error(err)
		return cres.IID{}, err
	}

	return getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}), nil
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

	if err := checkCapability(connectionName, CLUSTER_HANDLER); err != nil {
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
	var ngIIDInfo NodeGroupIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*NodeGroupIIDInfo
		err := getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return cres.NodeGroupInfo{}, err
		}
		castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, ngInfo.IId.SystemId)
		if err != nil {
			cblog.Error(err)
			return cres.NodeGroupInfo{}, err
		}
		ngIIDInfo = *castedIIDInfo.(*NodeGroupIIDInfo)
	} else {
		err = infostore.GetByContain(&ngIIDInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, ngInfo.IId.SystemId)
		if err != nil {
			cblog.Error(err)
			return cres.NodeGroupInfo{}, err
		}
	}
	ngInfo.IId.NameId = ngIIDInfo.NameId

	// (2) ImageIID
	ngInfo.ImageIID.NameId = ngInfo.ImageIID.SystemId

	// (3) Get KeyPair IIDInfo with SystemId
	var keyIIDInfo KeyIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*KeyIIDInfo
		err := getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return cres.NodeGroupInfo{}, err
		}
		castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, ngInfo.KeyPairIID.SystemId)
		if err != nil {
			cblog.Error(err)
			return cres.NodeGroupInfo{}, err
		}
		keyIIDInfo = *castedIIDInfo.(*KeyIIDInfo)
	} else {
		err = infostore.GetByContain(&keyIIDInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, ngInfo.KeyPairIID.SystemId)
		if err != nil {
			cblog.Error(err)
			return cres.NodeGroupInfo{}, err
		}
	}
	ngInfo.KeyPairIID.NameId = keyIIDInfo.NameId

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

	if err := checkCapability(connectionName, CLUSTER_HANDLER); err != nil {
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
		if checkNotFoundError(err) {
			// if not found in CSP, continue
			force = "true"
		} else if force != "true" {
			return false, err
		}
	}

	if force != "true" {
		if !result {
			return result, nil
		}
	}

	// (3) delete IID
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		_, err = infostore.DeleteByConditions(&NodeGroupIIDInfo{}, NAME_ID_COLUMN, nodeGroupName, OWNER_CLUSTER_NAME_COLUMN, clusterName)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, err
			}
		}
	} else {
		_, err = infostore.DeleteBy3Conditions(&NodeGroupIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nodeGroupName,
			OWNER_CLUSTER_NAME_COLUMN, clusterName)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, err
			}
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

	if err := checkCapability(connectionName, CLUSTER_HANDLER); err != nil {
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

	iid := cres.IID{NameId: "", SystemId: systemID}

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

	if err := checkCapability(connectionName, CLUSTER_HANDLER); err != nil {
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

func DeleteCluster(connectionName string, rsType string, nameID string, force string) (bool, error) {
	cblog.Info("call DeleteCluster()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	if err := checkCapability(connectionName, CLUSTER_HANDLER); err != nil {
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

	handler, err := cldConn.CreateClusterHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	clusterSPLock.Lock(connectionName, nameID)
	defer clusterSPLock.Unlock(connectionName, nameID)

	// (1) get spiderIID for creating driverIID
	var iidInfo *ClusterIIDInfo
	var iidInfoList []*ClusterIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	}
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == nameID {
			iidInfo = OneIIdInfo
			bool_ret = true
			break
		}
	}
	if !bool_ret {
		err := fmt.Errorf("[" + connectionName + ":" + RSTypeString(rsType) + ":" + nameID + "] does not exist!")
		cblog.Error(err)
		return false, err
	}

	// (2) delete Resource(SystemId)
	driverIId := getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
	result := false

	result, err = handler.(cres.ClusterHandler).DeleteCluster(driverIId)
	if err != nil {
		cblog.Error(err)
		if checkNotFoundError(err) {
			// if not found in CSP, continue
			force = "true"
		} else if force != "true" {
			return false, err
		}
	}

	if force != "true" {
		if result == false {
			return result, nil
		}
	}

	// (3) delete IID
	_, err = infostore.DeleteByConditions(&ClusterIIDInfo{}, CONNECTION_NAME_COLUMN, iidInfo.ConnectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}

	// for NodeGroup list
	// delete all nodegroups of target Cluster
	_, err = infostore.DeleteByConditions(&NodeGroupIIDInfo{}, CONNECTION_NAME_COLUMN, iidInfo.ConnectionName,
		OWNER_CLUSTER_NAME_COLUMN, iidInfo.NameId)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	return result, nil
}

func CountAllClusters() (int64, error) {
	var info ClusterIIDInfo
	count, err := infostore.CountAllNameIDs(&info)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}

func CountClustersByConnection(connectionName string) (int64, error) {
	var info ClusterIIDInfo
	count, err := infostore.CountNameIDsByConnection(&info, connectionName)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}
