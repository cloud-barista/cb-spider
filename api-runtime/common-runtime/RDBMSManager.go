// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, April 2026.

package commonruntime

import (
	"fmt"
	"os"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
	infostore "github.com/cloud-barista/cb-spider/info-store"
)

// ====================================================================
// type for GORM

type RDBMSIIDInfo VPCDependentIIDInfo

func (RDBMSIIDInfo) TableName() string {
	return "rdbms_iid_infos"
}

//====================================================================

func init() {
	db, err := infostore.Open()
	if err != nil {
		cblog.Error(err)
		return
	}
	db.AutoMigrate(&RDBMSIIDInfo{})
	infostore.Close(db)
}

//================ RDBMS Handler

// GetRDBMSOwnerVPC returns the owner VPC of a given RDBMS CSP ID
func GetRDBMSOwnerVPC(connectionName string, cspID string) (ownerVPC cres.IID, err error) {
	cblog.Info("call GetRDBMSOwnerVPC()")

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

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return cres.IID{}, err
	}

	handler, err := cldConn.CreateRDBMSHandler()
	if err != nil {
		cblog.Error(err)
		return cres.IID{}, err
	}

	var iidInfoList []*RDBMSIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return cres.IID{}, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return cres.IID{}, err
		}
	}
	var isExist bool = false
	var nameId string
	for _, OneIIdInfo := range iidInfoList {
		saveSystemId := getMSShortID(getDriverSystemId(cres.IID{NameId: OneIIdInfo.NameId, SystemId: OneIIdInfo.SystemId}))
		if saveSystemId == cspID {
			nameId = OneIIdInfo.NameId
			isExist = true
			break
		}
	}
	if isExist {
		var iidInfo RDBMSIIDInfo
		err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return cres.IID{}, err
		}
		ownerVPCName := iidInfo.OwnerVPCName
		var vpcIIDInfo VPCIIDInfo
		err = infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, ownerVPCName)
		if err != nil {
			cblog.Error(err)
			return cres.IID{}, err
		}
		return getUserIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId}), nil
	}

	// if not found in metadb, get from CSP
	info, err := handler.GetRDBMS(cres.IID{NameId: getMSShortID(cspID), SystemId: cspID})
	if err != nil {
		cblog.Error(err)
		return cres.IID{}, err
	}

	// find VPC by SystemId
	var vpcIIDInfo VPCIIDInfo
	err = infostore.GetByContain(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, info.VpcIID.SystemId)
	if err != nil {
		cblog.Error(err)
		return cres.IID{}, err
	}
	return getUserIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId}), nil
}

// GetRDBMSMetaInfo returns CSP's RDBMS meta information
func GetRDBMSMetaInfo(connectionName string) (*cres.RDBMSMetaInfo, error) {
	cblog.Info("call GetRDBMSMetaInfo()")

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

	handler, err := cldConn.CreateRDBMSHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	info, err := handler.GetMetaInfo()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return &info, nil
}

// (1) check existence(UserID)
// (2) get resource info(CSP-ID)
// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
// (4) insert spiderIID
func RegisterRDBMS(connectionName string, vpcUserID string, userIID cres.IID) (*cres.RDBMSInfo, error) {
	cblog.Info("call RegisterRDBMS()")

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

	emptyPermissionList := []string{}
	err = ValidateStruct(userIID, emptyPermissionList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	rsType := RDBMS

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateRDBMSHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcSPLock.RLock(connectionName, vpcUserID)
	defer vpcSPLock.RUnlock(connectionName, vpcUserID)
	rdbmsSPLock.Lock(connectionName, userIID.NameId)
	defer rdbmsSPLock.Unlock(connectionName, userIID.NameId)

	// (0) check VPC existence(VPC UserID)
	var bool_ret bool
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
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
		err := fmt.Errorf("%s '%s' does not exist in connection '%s'", RSTypeString(VPC), vpcUserID, connectionName)
		cblog.Error(err)
		return nil, err
	}

	// (1) check existence(UserID)
	var isExist bool
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		isExist, err = infostore.HasByCondition(&RDBMSIIDInfo{}, NAME_ID_COLUMN, userIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		isExist, err = infostore.HasByConditions(&RDBMSIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, userIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	if isExist {
		err := fmt.Errorf("%s '%s' already exists in connection '%s'", RSTypeString(rsType), userIID.NameId, connectionName)
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource info(CSP-ID)
	getInfo, err := handler.GetRDBMS(cres.IID{NameId: getMSShortID(userIID.SystemId), SystemId: userIID.SystemId})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
	systemId := getMSShortID(getInfo.IId.SystemId)
	spiderIId := cres.IID{NameId: userIID.NameId, SystemId: systemId + ":" + getInfo.IId.SystemId}

	// (4) insert spiderIID
	err = infostore.Insert(&RDBMSIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId,
		OwnerVPCName: vpcUserID})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// set up RDBMS User IID for return info
	getInfo.IId = userIID

	// set up VPC UserIID for return info
	var iidInfo VPCIIDInfo
	err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, vpcUserID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	getInfo.VpcIID = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

	return &getInfo, nil
}

// (1) check exist(NameID)
// (2) generate SP-XID and create reqIID, driverIID
// (3) create Resource
// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
// (5) insert spiderIID
// (6) create userIID
func CreateRDBMS(connectionName string, rsType string, reqInfo cres.RDBMSInfo, IDTransformMode string) (*cres.RDBMSInfo, error) {
	cblog.Info("call CreateRDBMS()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcSPLock.RLock(connectionName, reqInfo.VpcIID.NameId)
	defer vpcSPLock.RUnlock(connectionName, reqInfo.VpcIID.NameId)

	//+++++++++++++++++++++++++++++++++++++++++++
	// set VPC's SystemId
	var vpcIIDInfo VPCIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*VPCIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, reqInfo.VpcIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		vpcIIDInfo = *castedIIDInfo.(*VPCIIDInfo)
	} else {
		err = infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, reqInfo.VpcIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	reqInfo.VpcIID = getDriverIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})
	//+++++++++++++++++++++++++++++++++++++++++++

	// SubnetIIDs translation
	for idx, subnetIID := range reqInfo.SubnetIIDs {
		var subnetIIdInfo SubnetIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
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
			vpcInfo := *castedIIDInfo.(*VPCIIDInfo)
			err = infostore.GetBy3Conditions(&subnetIIdInfo, CONNECTION_NAME_COLUMN, vpcInfo.ConnectionName, NAME_ID_COLUMN, subnetIID.NameId, OWNER_VPC_NAME_COLUMN, vpcInfo.NameId)
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
		reqInfo.SubnetIIDs[idx] = getDriverIID(cres.IID{NameId: subnetIIdInfo.NameId, SystemId: subnetIIdInfo.SystemId})
	}
	//+++++++++++++++++++++++++++++++++++++++++++

	// SecurityGroupIIDs translation
	for idx, sgIID := range reqInfo.SecurityGroupIIDs {
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
		reqInfo.SecurityGroupIIDs[idx] = getDriverIID(cres.IID{NameId: sgIIdInfo.NameId, SystemId: sgIIdInfo.SystemId})
	}
	//+++++++++++++++++++++++++++++++++++++++++++

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateRDBMSHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	rdbmsSPLock.Lock(connectionName, reqInfo.IId.NameId)
	defer rdbmsSPLock.Unlock(connectionName, reqInfo.IId.NameId)

	// (1) check exist(NameID)
	var iidInfoList []*RDBMSIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, OWNER_VPC_NAME_COLUMN, vpcIIDInfo.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		err = infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, OWNER_VPC_NAME_COLUMN, vpcIIDInfo.NameId)
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
		err := fmt.Errorf("%s '%s' already exists in connection '%s'", RSTypeString(rsType), reqInfo.IId.NameId, connectionName)
		cblog.Error(err)
		return nil, err
	}

	spUUID := ""
	if GetID_MGMT(IDTransformMode) == "ON" {
		// (2) generate SP-XID and create reqIID, driverIID
		spUUID, err = iidm.New(connectionName, rsType, reqInfo.IId.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		spUUID = reqInfo.IId.NameId
	}

	// reqIID
	reqIId := cres.IID{NameId: reqInfo.IId.NameId, SystemId: spUUID}
	// driverIID
	driverIId := cres.IID{NameId: spUUID, SystemId: ""}
	reqInfo.IId = driverIId

	// (3) create Resource
	info, err := handler.CreateRDBMS(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// set VPC NameId
	info.VpcIID.NameId = vpcIIDInfo.NameId

	// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
	spiderIId := cres.IID{NameId: reqIId.NameId, SystemId: spUUID + ":" + info.IId.SystemId}

	// (5) insert spiderIID
	iidInfo := RDBMSIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId,
		OwnerVPCName: vpcIIDInfo.NameId}
	err = infostore.Insert(&iidInfo)
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteRDBMS(info.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		cblog.Error(err)
		return nil, err
	}

	// (6) create userIID: {reqNameID, driverSystemID}
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

	// set SubnetIIDs UserIID
	setRDBMSSubnetUserIID(connectionName, vpcIIDInfo, &info)
	// set SecurityGroupIIDs UserIID
	setRDBMSSGUserIID(connectionName, vpcIIDInfo, &info)

	return &info, nil
}

// setRDBMSSubnetUserIID sets SubnetIIDs to user-friendly names from metadb
func setRDBMSSubnetUserIID(connectionName string, vpcIIDInfo VPCIIDInfo, info *cres.RDBMSInfo) {
	for idx, subnetIID := range info.SubnetIIDs {
		var subnetIIdInfo SubnetIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			err := infostore.GetByConditionsAndContain(&subnetIIdInfo, CONNECTION_NAME_COLUMN, vpcIIDInfo.ConnectionName,
				OWNER_VPC_NAME_COLUMN, vpcIIDInfo.NameId, SYSTEM_ID_COLUMN, subnetIID.SystemId)
			if err != nil {
				cblog.Info(err)
				continue
			}
		} else {
			err := infostore.GetByConditionsAndContain(&subnetIIdInfo, CONNECTION_NAME_COLUMN, connectionName,
				OWNER_VPC_NAME_COLUMN, vpcIIDInfo.NameId, SYSTEM_ID_COLUMN, subnetIID.SystemId)
			if err != nil {
				cblog.Info(err)
				continue
			}
		}
		info.SubnetIIDs[idx].NameId = subnetIIdInfo.NameId
	}
}

// setRDBMSSGUserIID sets SecurityGroupIIDs to user-friendly names from metadb
func setRDBMSSGUserIID(connectionName string, vpcIIDInfo VPCIIDInfo, info *cres.RDBMSInfo) {
	for idx, sgIID := range info.SecurityGroupIIDs {
		var sgIIdInfo SGIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*SGIIDInfo
			err := getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Info(err)
				continue
			}
			castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, sgIID.SystemId)
			if err != nil {
				cblog.Info(err)
				continue
			}
			sgIIdInfo = *castedIIDInfo.(*SGIIDInfo)
		} else {
			err := infostore.GetByConditionsAndContain(&sgIIdInfo, CONNECTION_NAME_COLUMN, connectionName,
				OWNER_VPC_NAME_COLUMN, vpcIIDInfo.NameId, SYSTEM_ID_COLUMN, sgIID.SystemId)
			if err != nil {
				cblog.Info(err)
				continue
			}
		}
		info.SecurityGroupIIDs[idx].NameId = sgIIdInfo.NameId
	}
}

// (1) get IID:list
// (2) get RDBMSInfo:list
// (3) set userIID, and ...
func ListRDBMS(connectionName string, rsType string) ([]*cres.RDBMSInfo, error) {
	cblog.Info("call ListRDBMS()")

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

	handler, err := cldConn.CreateRDBMSHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (1) get IID:list
	var iidInfoList []*RDBMSIIDInfo
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

	var infoList []*cres.RDBMSInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.RDBMSInfo{}
		return infoList, nil
	}

	// (2) Get RDBMSInfo-list with IID-list
	infoList2 := []*cres.RDBMSInfo{}
	for _, iidInfo := range iidInfoList {

		rdbmsSPLock.RLock(connectionName, iidInfo.NameId)

		// get resource(SystemId)
		info, err := handler.GetRDBMS(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
		if err != nil {
			rdbmsSPLock.RUnlock(connectionName, iidInfo.NameId)
			if checkNotFoundError(err) {
				cblog.Error(err)
				info = cres.RDBMSInfo{IId: cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}}
				infoList2 = append(infoList2, &info)
				continue
			}
			cblog.Error(err)
			return nil, err
		}
		rdbmsSPLock.RUnlock(connectionName, iidInfo.NameId)

		// (3) set ResourceInfo(IID.NameId)
		info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

		// set VPC UserIID
		var vpcIIDInfo VPCIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*VPCIIDInfo
			err = getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, iidInfo.OwnerVPCName)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			vpcIIDInfo = *castedIIDInfo.(*VPCIIDInfo)
		} else {
			err = infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, iidInfo.OwnerVPCName)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		}
		info.VpcIID = getUserIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})

		// set SubnetIIDs UserIID
		setRDBMSSubnetUserIID(connectionName, vpcIIDInfo, &info)
		// set SecurityGroupIIDs UserIID
		setRDBMSSGUserIID(connectionName, vpcIIDInfo, &info)

		infoList2 = append(infoList2, &info)
	}

	return infoList2, nil
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetRDBMS(connectionName string, rsType string, nameID string) (*cres.RDBMSInfo, error) {
	cblog.Info("call GetRDBMS()")

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

	handler, err := cldConn.CreateRDBMSHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	rdbmsSPLock.RLock(connectionName, nameID)
	defer rdbmsSPLock.RUnlock(connectionName, nameID)

	// (1) get IID(NameId)
	var iidInfoList []*RDBMSIIDInfo
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

	var iidInfo *RDBMSIIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == nameID {
			iidInfo = OneIIdInfo
			bool_ret = true
			break
		}
	}
	if !bool_ret {
		err := fmt.Errorf("%s '%s' does not exist in connection '%s'", RSTypeString(rsType), nameID, connectionName)
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource(SystemId)
	info, err := handler.GetRDBMS(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) set ResourceInfo(IID.NameId)
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

	// set VPC UserIID
	var vpcIIDInfo VPCIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*VPCIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, iidInfo.OwnerVPCName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		vpcIIDInfo = *castedIIDInfo.(*VPCIIDInfo)
	} else {
		err = infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, iidInfo.OwnerVPCName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	info.VpcIID = getUserIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})

	// set SubnetIIDs UserIID
	setRDBMSSubnetUserIID(connectionName, vpcIIDInfo, &info)
	// set SecurityGroupIIDs UserIID
	setRDBMSSGUserIID(connectionName, vpcIIDInfo, &info)

	return &info, nil
}

func DeleteRDBMS(connectionName string, rsType string, nameID string, force string) (bool, error) {
	cblog.Info("call DeleteRDBMS()")

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

	handler, err := cldConn.CreateRDBMSHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	rdbmsSPLock.Lock(connectionName, nameID)
	defer rdbmsSPLock.Unlock(connectionName, nameID)

	// (1) get spiderIID for creating driverIID
	var iidInfoList []*RDBMSIIDInfo
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

	var iidInfo *RDBMSIIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == nameID {
			iidInfo = OneIIdInfo
			bool_ret = true
			break
		}
	}
	if !bool_ret {
		err := fmt.Errorf("%s '%s' does not exist in connection '%s'", RSTypeString(rsType), nameID, connectionName)
		cblog.Error(err)
		return false, err
	}

	// (2) delete Resource(SystemId)
	driverIId := getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
	result := false

	result, err = handler.(cres.RDBMSHandler).DeleteRDBMS(driverIId)
	if err != nil {
		cblog.Error(err)
		if checkNotFoundError(err) {
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
	_, err = infostore.DeleteByConditions(&RDBMSIIDInfo{}, CONNECTION_NAME_COLUMN, iidInfo.ConnectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}

	return result, nil
}

func CountAllRDBMS() (int64, error) {
	var info RDBMSIIDInfo
	count, err := infostore.CountAllNameIDs(&info)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}

func CountRDBMSByConnection(connectionName string) (int64, error) {
	var info RDBMSIIDInfo
	count, err := infostore.CountNameIDsByConnection(&info, connectionName)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}
