// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, April 2026.

package commonruntime

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
	infostore "github.com/cloud-barista/cb-spider/info-store"
)

// ====================================================================
// type for GORM

type RDBMSIIDInfo struct {
	ConnectionName string `gorm:"primaryKey"` // ex) "ncp-korea1-config"
	NameId         string `gorm:"primaryKey"` // ex) "my-rdbms-01"
	SystemId       string // ID in CSP
	OwnerVPCName   string `gorm:"primaryKey"` // ex) "vpc-01"
}

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

// GetRDBMSMetaInfo returns CSP's RDBMS meta information for a requested DB engine.
func GetRDBMSMetaInfo(connectionName string, dbEngine string) (*cres.RDBMSMetaInfo, error) {
	cblog.Info("call GetRDBMSMetaInfo()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	dbEngine, err = EmptyCheckAndTrim("dbEngine", dbEngine)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if _, err := cres.NormalizeRDBMSEngine(dbEngine); err != nil {
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

	info, err := handler.GetMetaInfo(dbEngine)
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

// -------- RDBMS Database Management (optional CSP-native API) --------
// These functions use the optional RDBMSDatabaseManager interface when available.
// If the driver does not implement the interface, ErrRDBMSDatabaseMgrNotSupported
// is returned so that callers (e.g. AdminWeb) can fall back to direct SQL.

// ErrRDBMSDatabaseMgrNotSupported is returned when the driver does not implement
// the rdbmsDatabaseManager interface.
var ErrRDBMSDatabaseMgrNotSupported = fmt.Errorf("driver does not support CSP-native database management")

// openRDBMSSQLConn opens a direct SQL connection to the RDBMS instance using endpoint/port/user from info
// and the supplied masterUserPassword. Returns the *sql.DB and the driver name ("mysql"/"postgres").
func openRDBMSSQLConn(info *cres.RDBMSInfo, masterUserPassword string) (*sql.DB, string, error) {
	if masterUserPassword == "" {
		return nil, "", ErrRDBMSDatabaseMgrNotSupported
	}
	if info.Endpoint == "" {
		return nil, "", fmt.Errorf("RDBMS endpoint is empty; instance may still be provisioning")
	}

	engine := strings.ToLower(string(info.DBEngine))
	host := info.Endpoint
	port := ""
	user := info.MasterUserName

	// Strip port embedded in endpoint ("host:port" form)
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		hostPart := host[:idx]
		portPart := host[idx+1:]
		var p int
		if _, err := fmt.Sscanf(portPart, "%d", &p); err == nil {
			host = hostPart
			port = portPart
		}
	}

	var driverName, dsn string
	switch {
	case engine == "mysql" || engine == "mariadb":
		driverName = "mysql"
		if port == "" {
			port = "3306"
		}
		// Enable TLS for IBM Cloud MySQL (skip server cert verification)
		tlsSuffix := ""
		if strings.Contains(strings.ToLower(host), ".databases.appdomain.cloud") {
			tlsSuffix = "?tls=skip-verify"
		}
		dsn = fmt.Sprintf("%s:%s@tcp(%s)/%s", user, masterUserPassword, net.JoinHostPort(host, port), tlsSuffix)
	case engine == "postgresql" || engine == "postgres":
		driverName = "postgres"
		if port == "" {
			port = "5432"
		}
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=require connect_timeout=30",
			host, port, user, masterUserPassword)
	default:
		return nil, "", fmt.Errorf("SQL fallback: unsupported DB engine %q", engine)
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, "", fmt.Errorf("SQL fallback open: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, "", fmt.Errorf("SQL fallback connect: %w", err)
	}
	return db, driverName, nil
}

// createDatabaseSQL creates a database via direct SQL (CREATE DATABASE).
func createDatabaseSQL(info *cres.RDBMSInfo, masterUserPassword, dbName string) error {
	db, driverName, err := openRDBMSSQLConn(info, masterUserPassword)
	if err != nil {
		return err
	}
	defer db.Close()

	var stmt string
	if driverName == "postgres" {
		stmt = `CREATE DATABASE "` + dbName + `"`
	} else {
		stmt = "CREATE DATABASE `" + dbName + "`"
	}
	if _, err := db.Exec(stmt); err != nil {
		return fmt.Errorf("CREATE DATABASE %q: %w", dbName, err)
	}
	return nil
}

// listDatabasesSQL lists databases via direct SQL.
func listDatabasesSQL(info *cres.RDBMSInfo, masterUserPassword string) ([]string, error) {
	db, driverName, err := openRDBMSSQLConn(info, masterUserPassword)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var query string
	if driverName == "postgres" {
		query = "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname"
	} else {
		query = "SHOW DATABASES"
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("listDatabases SQL: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("listDatabases SQL scan: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// deleteDatabaseSQL drops a database via direct SQL (DROP DATABASE).
func deleteDatabaseSQL(info *cres.RDBMSInfo, masterUserPassword, dbName string) error {
	db, driverName, err := openRDBMSSQLConn(info, masterUserPassword)
	if err != nil {
		return err
	}
	defer db.Close()

	var stmt string
	if driverName == "postgres" {
		stmt = `DROP DATABASE "` + dbName + `"`
	} else {
		stmt = "DROP DATABASE `" + dbName + "`"
	}
	if _, err := db.Exec(stmt); err != nil {
		return fmt.Errorf("DROP DATABASE %q: %w", dbName, err)
	}
	return nil
}

// rdbmsDatabaseManager is a private interface satisfied by RDBMS drivers that provide
// CSP-native database CRUD without requiring direct SQL privileges.
// This interface is NOT part of the public RDBMSHandler contract; drivers implement it
// via Go structural (duck) typing so no existing driver needs to be changed.
type rdbmsDatabaseManager interface {
	CreateDatabase(rdbmsSystemId, dbEngine, dbName string) error
	ListDatabases(rdbmsSystemId, dbEngine string) ([]string, error)
	DeleteDatabase(rdbmsSystemId, dbEngine, dbName string) error
}

func getRDBMSSystemId(connectionName, rdbmsName string) (string, string, error) {
	var iidInfo RDBMSIIDInfo
	err := infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, rdbmsName)
	if err != nil {
		return "", "", fmt.Errorf("RDBMS '%s' not found in connection '%s': %w", rdbmsName, connectionName, err)
	}
	return iidInfo.SystemId, iidInfo.NameId, nil
}

// CreateRDBMSDatabase creates a database in the named RDBMS instance.
// If the driver supports the CSP-native rdbmsDatabaseManager interface, it is used.
// Otherwise, if masterUserPassword is provided, a direct SQL connection is attempted.
func CreateRDBMSDatabase(connectionName, rdbmsName, dbName, masterUserPassword string) error {
	cblog.Info("call CreateRDBMSDatabase()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		return err
	}
	rdbmsName, err = EmptyCheckAndTrim("rdbmsName", rdbmsName)
	if err != nil {
		return err
	}
	dbName, err = EmptyCheckAndTrim("dbName", dbName)
	if err != nil {
		return err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		return err
	}

	handler, err := cldConn.CreateRDBMSHandler()
	if err != nil {
		return err
	}

	systemId, _, err := getRDBMSSystemId(connectionName, rdbmsName)
	if err != nil {
		return err
	}

	driverIId := getDriverIID(cres.IID{NameId: rdbmsName, SystemId: systemId})

	info, err := handler.GetRDBMS(driverIId)
	if err != nil {
		return err
	}

	if dbMgr, ok := handler.(rdbmsDatabaseManager); ok {
		return dbMgr.CreateDatabase(driverIId.SystemId, string(info.DBEngine), dbName)
	}

	// SQL fallback (for drivers without CSP-native DB management API, e.g. AWS, IBM)
	return createDatabaseSQL(&info, masterUserPassword, dbName)
}

// ListRDBMSDatabases lists databases in the named RDBMS instance.
// If the driver supports the CSP-native rdbmsDatabaseManager interface, it is used.
// Otherwise, if masterUserPassword is provided, a direct SQL connection is attempted.
func ListRDBMSDatabases(connectionName, rdbmsName, masterUserPassword string) ([]string, error) {
	cblog.Info("call ListRDBMSDatabases()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		return nil, err
	}
	rdbmsName, err = EmptyCheckAndTrim("rdbmsName", rdbmsName)
	if err != nil {
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		return nil, err
	}

	handler, err := cldConn.CreateRDBMSHandler()
	if err != nil {
		return nil, err
	}

	systemId, _, err := getRDBMSSystemId(connectionName, rdbmsName)
	if err != nil {
		return nil, err
	}

	driverIId := getDriverIID(cres.IID{NameId: rdbmsName, SystemId: systemId})

	info, err := handler.GetRDBMS(driverIId)
	if err != nil {
		return nil, err
	}

	if dbMgr, ok := handler.(rdbmsDatabaseManager); ok {
		return dbMgr.ListDatabases(driverIId.SystemId, string(info.DBEngine))
	}

	// SQL fallback
	return listDatabasesSQL(&info, masterUserPassword)
}

// DeleteRDBMSDatabase drops a database from the named RDBMS instance.
// If the driver supports the CSP-native rdbmsDatabaseManager interface, it is used.
// Otherwise, if masterUserPassword is provided, a direct SQL connection is attempted.
func DeleteRDBMSDatabase(connectionName, rdbmsName, dbName, masterUserPassword string) error {
	cblog.Info("call DeleteRDBMSDatabase()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		return err
	}
	rdbmsName, err = EmptyCheckAndTrim("rdbmsName", rdbmsName)
	if err != nil {
		return err
	}
	dbName, err = EmptyCheckAndTrim("dbName", dbName)
	if err != nil {
		return err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		return err
	}

	handler, err := cldConn.CreateRDBMSHandler()
	if err != nil {
		return err
	}

	systemId, _, err := getRDBMSSystemId(connectionName, rdbmsName)
	if err != nil {
		return err
	}

	driverIId := getDriverIID(cres.IID{NameId: rdbmsName, SystemId: systemId})

	info, err := handler.GetRDBMS(driverIId)
	if err != nil {
		return err
	}

	if dbMgr, ok := handler.(rdbmsDatabaseManager); ok {
		return dbMgr.DeleteDatabase(driverIId.SystemId, string(info.DBEngine), dbName)
	}

	// SQL fallback
	return deleteDatabaseSQL(&info, masterUserPassword, dbName)
}
