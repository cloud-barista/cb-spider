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
	"regexp"
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

// redactRDBMSMasterCredentials clears MasterUserName/MasterUserPassword on a response about to be
// returned from RegisterRDBMS/ListRDBMS/GetRDBMS (only a successful CreateRDBMS response still
// carries these). Drivers no longer populate either field outside of CreateRDBMS, so this is a
// safety net, not a live correction — it keeps this function the one place that guarantees no
// credential ever leaves CB-Spider via Get/List/Register, even if a future driver regresses.
func redactRDBMSMasterCredentials(info *cres.RDBMSInfo) {
	info.MasterUserName = ""
	info.MasterUserPassword = ""
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

// ListRDBMSEngine returns the RDBMS engine names (e.g., "mysql", "mariadb", "postgresql")
// that the connection's CSP driver supports, derived from the same per-engine capability
// flags exposed by GetDriverCapabilityInfo()/GET /driver/capability (RDBMSHandler,
// RDBMSMySQLHandler, RDBMSMariaDBHandler, RDBMSPostgreSQLHandler). RDBMSHandler gates the
// whole group: if it is false, the CSP driver does not support RDBMS at all, so the
// per-engine flags are not meaningful and an empty list is returned.
func ListRDBMSEngine(connectionName string) ([]string, error) {
	cblog.Info("call ListRDBMSEngine()")

	capability, err := GetDriverCapabilityInfo(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	engines := []string{}
	if capability.RDBMSHandler {
		if capability.RDBMSMySQLHandler {
			engines = append(engines, "mysql")
		}
		if capability.RDBMSMariaDBHandler {
			engines = append(engines, "mariadb")
		}
		if capability.RDBMSPostgreSQLHandler {
			engines = append(engines, "postgresql")
		}
	}

	return engines, nil
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

	// RegisterRDBMS only attaches an existing CSP DB engine; it is not a CB-Spider deployment,
	// so the master credentials are redacted like any other Get/List response.
	redactRDBMSMasterCredentials(&getInfo)

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
	//
	// NHN Cloud RDBMS does not use SecurityGroupNames/SecurityGroupIIDs at all
	// (see RDBMSInfo.NHNAutoOpenDBSecurityGroup instead): NHN Cloud RDS DB
	// Security Groups are a resource type separate from the VPC/Neutron
	// security group this loop resolves, so a name given here would never
	// correspond to a real registered SecurityGroup and resolution would
	// always fail with "not found". Skip translation entirely for NHN so any
	// value the caller sent is simply ignored, matching the driver's behavior.
	providerName, err := ccm.GetProviderNameByConnectionName(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if !strings.EqualFold(providerName, "NHN") {
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

		redactRDBMSMasterCredentials(&info)

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

	redactRDBMSMasterCredentials(&info)

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

// openRDBMSSQLConn opens a direct SQL connection to the RDBMS instance using endpoint/port from info
// and the caller-supplied masterUserName/masterUserPassword. The master username is never read off
// info/the CSP driver: CB-Spider cannot reliably identify the actual master account for several CSPs
// (see redactRDBMSMasterCredentials), so callers must supply the credentials they set at deploy time,
// same as the password. Returns the *sql.DB and the driver name ("mysql"/"postgres").
func openRDBMSSQLConn(info *cres.RDBMSInfo, masterUserName, masterUserPassword string) (*sql.DB, string, error) {
	if masterUserName == "" || masterUserPassword == "" {
		return nil, "", ErrRDBMSDatabaseMgrNotSupported
	}
	if info.Endpoint == "" {
		return nil, "", fmt.Errorf("RDBMS endpoint is empty; instance may still be provisioning")
	}

	engine := strings.ToLower(string(info.DBEngine))
	host, port := splitRDBMSEndpoint(info.Endpoint)
	user := masterUserName

	var driverName, dsn string
	switch {
	case engine == "mysql" || engine == "mariadb":
		driverName = "mysql"
		if port == "" {
			port = "3306"
		}
		// Use TLS whenever the server offers it, but don't require it: some CSPs enforce
		// require_secure_transport=ON (e.g. IBM, and AWS/others when the operator enabled it),
		// which rejects a plain connection outright; others don't enforce it at all. "preferred"
		// handles both without needing to special-case any CSP's endpoint domain.
		dsn = fmt.Sprintf("%s:%s@tcp(%s)/?tls=preferred", user, masterUserPassword, net.JoinHostPort(host, port))
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

// lookupDefaultCollation connects directly to the instance (mysql/mariadb only -- callers must
// enforce this) and asks the engine itself what collation it considers the default for charset,
// via information_schema.CHARACTER_SETS. This is deliberately a live query rather than a hardcoded
// table: a charset's default collation can differ by engine version (e.g. MySQL 5.7's default for
// utf8mb4 is utf8mb4_general_ci; MySQL 8.0+'s is utf8mb4_0900_ai_ci; MariaDB 11.4.2+'s is
// utf8mb4_uca1400_ai_ci) and CB-Spider has no reliable way to know which version a given instance
// is actually running without asking it.
//
// This requires masterUserName/masterUserPassword even on CSPs that otherwise don't need them for
// CreateDatabase (e.g. Azure, which creates the database via its own native API, not SQL) --
// there's no other way to query information_schema. Called only when the caller supplied a charset
// but no collation; passing both explicitly skips this lookup entirely.
func lookupDefaultCollation(info *cres.RDBMSInfo, masterUserName, masterUserPassword, charset string) (string, error) {
	if masterUserName == "" || masterUserPassword == "" {
		return "", fmt.Errorf("resolving the default collation for charset %q requires MasterUserName/MasterUserPassword (used once, read-only, to query information_schema.CHARACTER_SETS on the instance) -- pass Collation explicitly to skip this lookup", charset)
	}
	db, _, err := openRDBMSSQLConn(info, masterUserName, masterUserPassword)
	if err != nil {
		return "", fmt.Errorf("connecting to look up the default collation for charset %q: %w", charset, err)
	}
	defer db.Close()

	var collation string
	err = db.QueryRow(
		"SELECT DEFAULT_COLLATE_NAME FROM information_schema.CHARACTER_SETS WHERE CHARACTER_SET_NAME = ?", charset,
	).Scan(&collation)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("charset %q is not a character set this instance recognizes", charset)
	}
	if err != nil {
		return "", fmt.Errorf("querying default collation for charset %q: %w", charset, err)
	}
	return collation, nil
}

// splitRDBMSEndpoint splits an RDBMSInfo.Endpoint value into host and port. The port is
// returned empty when Endpoint has no ":port" suffix (or that suffix isn't numeric) — callers
// apply their own engine-specific default in that case.
func splitRDBMSEndpoint(endpoint string) (host, port string) {
	host = endpoint
	idx := strings.LastIndex(endpoint, ":")
	if idx <= 0 {
		return host, ""
	}
	hostPart := endpoint[:idx]
	portPart := endpoint[idx+1:]
	var p int
	if _, err := fmt.Sscanf(portPart, "%d", &p); err != nil {
		return host, ""
	}
	return hostPart, portPart
}

// RDBMSPgHbaRule is one row of PostgreSQL's pg_hba_file_rules view.
type RDBMSPgHbaRule struct {
	Type       string `json:"Type"`
	Database   string `json:"Database"`
	UserName   string `json:"UserName"`
	Address    string `json:"Address,omitempty"`
	AuthMethod string `json:"AuthMethod"`
}

// RDBMS SSL modes recommended via RDBMSSecureTransportInfo.RecommendedSSLMode, named after the
// standard MySQL ssl-mode values (PostgreSQL clients should map VERIFY_CA/VERIFY_IDENTITY to
// their own sslmode=verify-ca/verify-full).
const (
	RDBMSSSLModeDisabled       = "DISABLED"
	RDBMSSSLModeVerifyCA       = "VERIFY_CA"
	RDBMSSSLModeVerifyIdentity = "VERIFY_IDENTITY"
)

// RDBMSSecureTransportInfo reports whether an RDBMS instance enforces encrypted (TLS/SSL)
// client connections, determined via standard SQL against the engine itself (CSP-agnostic).
type RDBMSSecureTransportInfo struct {
	Engine string `json:"Engine"` // "mysql", "mariadb", or "postgres"

	// MySQL/MariaDB: raw value of the require_secure_transport system variable ("ON" or "OFF").
	RequireSecureTransport string `json:"RequireSecureTransport,omitempty"`

	// PostgreSQL: best-effort verdict derived from pg_hba_file_rules — true only if every
	// matching TCP rule requires SSL (no plain "host" rule accepts a non-rejected connection).
	Enforced *bool `json:"Enforced,omitempty"`
	// PostgreSQL: the raw pg_hba_file_rules rows the verdict above was derived from, for transparency.
	Rules []RDBMSPgHbaRule `json:"Rules,omitempty"`

	// TLSInUse is empirical, not config-derived: this diagnostic connection itself was opened
	// with tls=preferred (TLS attempted first, plaintext only as fallback), so TLSInUse=false
	// means the server doesn't offer TLS at all — independent of RequireSecureTransport/Enforced,
	// which only say whether TLS is mandatory, not whether it's available.
	TLSInUse bool `json:"TLSInUse"`
	// TLSCipher: negotiated cipher suite name when TLSInUse=true, e.g. "ECDHE-RSA-AES128-GCM-SHA256"
	// (TLS 1.2) or "TLS_AES_256_GCM_SHA384" (TLS 1.3); empty when TLSInUse=false.
	TLSCipher string `json:"TLSCipher,omitempty"`

	// CACertificate is captured via a separate, live TLS handshake against the endpoint (not
	// sourced from any CSP API/doc — see RDBMSCACertInfo). nil when TLSInUse=false, or when the
	// probe itself failed; this is best-effort and never fails the overall request.
	CACertificate *RDBMSCACertInfo `json:"CACertificate,omitempty"`
	// CACertificateError explains why CACertificate is absent despite TLSInUse=true (e.g. a
	// transient network/handshake timeout on this separate probe connection) — set only in that
	// case, so callers don't have to dig through server logs to tell "not attempted" from "failed".
	CACertificateError string `json:"CACertificateError,omitempty"`

	// RecommendedSSLMode is the strongest of RDBMSSSLModeDisabled/VerifyCA/VerifyIdentity a
	// client can actually use against this instance, derived empirically rather than from CSP
	// metadata (no CSP exposes this directly):
	//   - DISABLED: TLSInUse=false — the server offers no TLS at all (e.g. Alibaba/Tencent/NCP
	//     with TLS turned off).
	//   - VERIFY_CA: TLSInUse=true, but the server's certificate has no Subject Alternative Name
	//     at all, so hostname verification (VERIFY_IDENTITY) will always fail regardless of
	//     client config — e.g. MySQL's own auto-generated certs on OpenStack Trove/NHN Cloud
	//     (see test/rdbms-mysql-test/tls-test/README.md's Known Caveats). Chain-only
	//     verification against CACertificate above still works.
	//   - VERIFY_IDENTITY: TLSInUse=true and the certificate carries a usable SAN (e.g. AWS,
	//     Azure, GCP, IBM) — the strongest mode is safe to use.
	// Empty when TLSInUse=true but the certificate probe itself failed (see CACertificateError):
	// there's then no basis to tell VERIFY_CA and VERIFY_IDENTITY apart.
	RecommendedSSLMode string `json:"RecommendedSSLMode,omitempty"`
}

// GetRDBMSSecureTransportStatus connects to the RDBMS instance with standard SQL and reports
// whether the server enforces encrypted (TLS/SSL) client connections:
//   - MySQL/MariaDB: SHOW VARIABLES LIKE 'require_secure_transport'
//   - PostgreSQL:    pg_hba_file_rules
//
// This works uniformly across every CSP because it queries the engine itself rather than
// each CSP's own (inconsistently available) management API.
func GetRDBMSSecureTransportStatus(connectionName, rdbmsName, masterUserName, masterUserPassword string) (*RDBMSSecureTransportInfo, error) {
	cblog.Info("call GetRDBMSSecureTransportStatus()")

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

	db, driverName, err := openRDBMSSQLConn(&info, masterUserName, masterUserPassword)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var result *RDBMSSecureTransportInfo

	switch driverName {
	case "mysql":
		var varName, varValue string
		if err := db.QueryRow("SHOW VARIABLES LIKE 'require_secure_transport'").Scan(&varName, &varValue); err != nil {
			return nil, fmt.Errorf("failed to read require_secure_transport: %w", err)
		}

		// Empirical check: did THIS connection (opened with tls=preferred) actually end up
		// encrypted? A non-empty Ssl_cipher means yes; empty means the server offered no TLS
		// at all, so the preferred-mode client fell back to plaintext.
		var sslStatusName, sslCipher string
		_ = db.QueryRow("SHOW STATUS LIKE 'Ssl_cipher'").Scan(&sslStatusName, &sslCipher)

		result = &RDBMSSecureTransportInfo{
			Engine:                 string(info.DBEngine),
			RequireSecureTransport: strings.ToUpper(varValue),
			TLSInUse:               sslCipher != "",
			TLSCipher:              sslCipher,
		}

	case "postgres":
		rows, err := db.Query(`SELECT type, database::text, user_name::text, COALESCE(address, ''), auth_method FROM pg_hba_file_rules`)
		if err != nil {
			return nil, fmt.Errorf("failed to read pg_hba_file_rules: %w", err)
		}
		defer rows.Close()

		var rules []RDBMSPgHbaRule
		enforced := true
		for rows.Next() {
			var r RDBMSPgHbaRule
			if err := rows.Scan(&r.Type, &r.Database, &r.UserName, &r.Address, &r.AuthMethod); err != nil {
				return nil, fmt.Errorf("failed to scan pg_hba_file_rules row: %w", err)
			}
			rules = append(rules, r)
			// A plain "host" (non-SSL TCP) rule that doesn't reject the connection means
			// plaintext connections are still accepted for whatever it matches.
			if r.Type == "host" && strings.ToLower(r.AuthMethod) != "reject" {
				enforced = false
			}
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("error reading pg_hba_file_rules: %w", err)
		}

		// Empirical check: did THIS connection (opened with sslmode=require) actually end up
		// encrypted? pg_stat_ssl reports it per-backend; a query error (e.g. insufficient
		// privilege) just leaves tlsInUse at its zero value rather than failing the request.
		var tlsInUse bool
		var sslVersion, sslCipher string
		_ = db.QueryRow(`SELECT ssl, COALESCE(version, ''), COALESCE(cipher, '') FROM pg_stat_ssl WHERE pid = pg_backend_pid()`).
			Scan(&tlsInUse, &sslVersion, &sslCipher)

		result = &RDBMSSecureTransportInfo{
			Engine:    string(info.DBEngine),
			Enforced:  &enforced,
			Rules:     rules,
			TLSInUse:  tlsInUse,
			TLSCipher: sslCipher,
		}

	default:
		return nil, fmt.Errorf("unsupported DB engine: %s", driverName)
	}

	// Best-effort: capture the server's certificate via a separate, live TLS handshake (not
	// sourced from any CSP API/doc — see RDBMSCACertInfo). Only worth attempting when we already
	// know TLS is available; a probe failure here must never fail the overall request.
	//
	// Close the SQL connection first (rather than waiting for the deferred Close) so this probe
	// isn't competing with an already-open connection to the same instance — a second concurrent
	// connection attempt right after the first is a plausible source of the transient handshake
	// timeouts this probe can hit in practice.
	if !result.TLSInUse {
		result.RecommendedSSLMode = RDBMSSSLModeDisabled
	} else {
		db.Close()
		host, port := splitRDBMSEndpoint(info.Endpoint)
		if port == "" {
			if driverName == "mysql" {
				port = "3306"
			} else {
				port = "5432"
			}
		}
		if cert, hasSAN, err := fetchRDBMSCACertificate(string(info.DBEngine), host, port); err != nil {
			cblog.Warnf("GetRDBMSSecureTransportStatus: CA certificate probe failed for %s: %v", rdbmsName, err)
			result.CACertificateError = err.Error()
		} else {
			result.CACertificate = cert
			if hasSAN {
				result.RecommendedSSLMode = RDBMSSSLModeVerifyIdentity
			} else {
				result.RecommendedSSLMode = RDBMSSSLModeVerifyCA
			}
		}
	}

	return result, nil
}

// createDatabaseSQL creates a database via direct SQL (CREATE DATABASE).
// charset/collation are optional (empty string = engine default) and, when the target is
// mysql/mariadb, are appended as a CHARACTER SET/COLLATE clause. CB-Spider does not validate
// these against a list of known charset/collation names -- MySQL and MariaDB each add new ones
// in almost every release, and the two engines don't even share a naming scheme for their newest
// ones (MySQL's "_0900_*" vs MariaDB's "_uca1400_*") -- it passes the caller's value straight to
// the engine and lets the engine itself reject anything it doesn't recognize. The caller
// (CreateRDBMSDatabase) is responsible for restricting dbName/charset/collation to safe identifier
// characters before calling this function, since none of these three values can be passed as a
// bound query parameter in this position and are concatenated into the statement text as-is.
func createDatabaseSQL(info *cres.RDBMSInfo, masterUserName, masterUserPassword, dbName, charset, collation string) error {
	db, driverName, err := openRDBMSSQLConn(info, masterUserName, masterUserPassword)
	if err != nil {
		return err
	}
	defer db.Close()

	var stmt string
	if driverName == "postgres" {
		// Charset/collation are rejected before reaching here for postgres (see
		// CreateRDBMSDatabase) -- Postgres' CREATE DATABASE ENCODING/LC_COLLATE semantics (template
		// database matching, locale availability) are different enough from MySQL/MariaDB's
		// CHARACTER SET/COLLATE that supporting them is out of scope for now.
		stmt = `CREATE DATABASE "` + dbName + `"`
	} else {
		stmt = "CREATE DATABASE `" + dbName + "`"
		if charset != "" {
			stmt += " CHARACTER SET `" + charset + "`"
		}
		if collation != "" {
			stmt += " COLLATE `" + collation + "`"
		}
	}
	if _, err := db.Exec(stmt); err != nil {
		return fmt.Errorf("CREATE DATABASE %q: %w", dbName, err)
	}
	return nil
}

// execSQLStatement opens a direct SQL connection and executes a single statement. Used by
// CSP-specific paths that need something other than a plain CREATE DATABASE: NCP's
// rdbmsDatabaseSQLStatementBuilder (a stored-procedure call the driver itself builds) and
// Tencent's rdbmsDatabaseCharsetManager (a follow-up ALTER DATABASE to apply a collation its
// native API has no field for). Callers are responsible for ensuring stmt only contains values
// already validated by validateSQLSafeValue -- it is executed as-is, with no parameter binding.
func execSQLStatement(info *cres.RDBMSInfo, masterUserName, masterUserPassword, stmt string) error {
	db, _, err := openRDBMSSQLConn(info, masterUserName, masterUserPassword)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec(stmt); err != nil {
		return fmt.Errorf("executing %q: %w", stmt, err)
	}
	return nil
}

// listDatabasesSQL lists databases via direct SQL.
func listDatabasesSQL(info *cres.RDBMSInfo, masterUserName, masterUserPassword string) ([]string, error) {
	db, driverName, err := openRDBMSSQLConn(info, masterUserName, masterUserPassword)
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
func deleteDatabaseSQL(info *cres.RDBMSInfo, masterUserName, masterUserPassword, dbName string) error {
	db, driverName, err := openRDBMSSQLConn(info, masterUserName, masterUserPassword)
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

// rdbmsDatabaseOptionsManager is an additional, separately-checked interface for drivers whose
// CSP-native "create database" API can accept an explicit charset and/or collation (Azure, GCP,
// OpenStack, and Alibaba -- see the CB-Spider RDBMS charset/collation support survey).
// It is intentionally a distinct interface from rdbmsDatabaseManager, not an extra parameter on
// CreateDatabase, so that drivers which don't support charset/collation at all need no signature
// change. Checked first: it's the only path that can honor both charset and collation using a
// single native API call.
type rdbmsDatabaseOptionsManager interface {
	CreateDatabaseWithOptions(rdbmsSystemId, dbEngine, dbName, charset, collation string) error
}

// rdbmsDatabaseRequiresPairedCollation marks a rdbmsDatabaseOptionsManager driver whose CSP-native
// "create database" call silently drops a Charset-only request -- falling back to the server's own
// default charset entirely, with no error -- unless Collation is also given in the same call.
// Confirmed on Azure MySQL Flexible Server; confirmed NOT needed on GCP, OpenStack, or Alibaba (all
// three apply a Charset-only request correctly on their own, per the charset/collation test
// suites' very first run, before this auto-resolution existed at all).
//
// This must stay a separate, narrowly-scoped interface rather than something CreateRDBMSDatabase
// does unconditionally for every rdbmsDatabaseOptionsManager driver: Alibaba is also one, and its
// CreateDatabaseWithOptions explicitly rejects a non-empty Collation on MariaDB (see its own
// comment) -- auto-filling a collation there for a plain Charset-only request would turn a request
// that used to work (and should keep working) into a guaranteed, confusing rejection for something
// the caller never asked for. Only query/pair collation where it's actually known to be needed.
type rdbmsDatabaseRequiresPairedCollation interface {
	RequiresCollationPairedWithCharset() bool
}

// rdbmsDatabaseCharsetManager is for drivers whose CSP-native "create database" API accepts a
// charset but has no field for collation at all (Tencent's CDB CreateDatabase -- its request
// struct has exactly three fields: InstanceId, DBName, CharacterSetName). When collation is also
// requested, CreateRDBMSDatabase applies it with a follow-up ALTER DATABASE over direct SQL, using
// the caller-supplied master credentials -- see execSQLStatement.
type rdbmsDatabaseCharsetManager interface {
	CreateDatabaseWithCharset(rdbmsSystemId, dbEngine, dbName, charset string) error
}

// rdbmsDatabaseSQLStatementBuilder is for drivers whose CSP has no charset/collation support in
// its native "create database" API, and whose normal SQL privileges don't allow a plain CREATE
// DATABASE either, but does expose some other SQL-callable mechanism that can -- a stored
// procedure, concretely (NCP's `sys.ncp_create_db('name','charset','collation')`, which the NCP
// console/docs document as the sanctioned way around its master account's lack of a global CREATE
// grant). The driver only builds the SQL text (it knows the exact procedure name/signature for its
// CSP); CreateRDBMSDatabase executes it via execSQLStatement. dbName/charset/collation are
// guaranteed already validated by validateSQLSafeValue by the time this is called, so building the
// statement by string concatenation is safe.
type rdbmsDatabaseSQLStatementBuilder interface {
	BuildCreateDatabaseSQL(dbEngine, dbName, charset, collation string) (string, error)
}

// rdbmsDatabaseSQLFallbackEligible marks a rdbmsDatabaseManager driver whose CSP, unlike Tencent or
// NCP, allows a direct SQL connection sufficient for a plain CREATE DATABASE ... CHARACTER SET ...
// COLLATE ... -- but only after an instance-level opt-in CB-Spider has no way to verify or enable
// itself (NHN Cloud's "DB 스키마 & 사용자 직접 제어" / "Direct Control" toggle, set in the NHN
// console; see NHN's RDBMSHandler.go). When the driver implements this and the caller requests
// charset/collation, CreateRDBMSDatabase uses the generic SQL path (createDatabaseSQL) instead of
// returning the usual "not supported" error. If the opt-in hasn't actually been enabled on the
// instance, the SQL engine's own permission error surfaces to the caller as-is, which already
// states the problem (a privilege error) clearly enough without CB-Spider editorializing on it.
type rdbmsDatabaseSQLFallbackEligible interface {
	IsSQLFallbackEligibleForCharsetCollation() bool
}

// sqlIdentifierPattern restricts a value to letters, digits, and underscores -- enough to cover
// every official MySQL/MariaDB charset and collation name (e.g. "utf8mb4_de_pb_0900_as_cs"), and
// safe to concatenate into a backtick-quoted SQL identifier.
var sqlIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// dbNamePattern additionally allows '-', which is common in cloud resource naming conventions and
// carries no SQL-injection risk once backtick-quoted.
var dbNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// validateSQLSafeValue rejects a dbName/charset/collation value that isn't a plain run of safe
// identifier characters. CB-Spider does not maintain an enum of valid charset/collation names (see
// rdbmsDatabaseOptionsManager/createDatabaseSQL) and passes whatever the caller supplies straight
// through to the SQL engine or CSP API -- but dbName/charset/collation are concatenated directly
// into a CREATE DATABASE statement in the SQL-fallback path (they can't be bound as query
// parameters in this position), so the format must still be restricted to prevent SQL injection.
func validateSQLSafeValue(fieldName, value string, pattern *regexp.Regexp) error {
	if !pattern.MatchString(value) {
		return fmt.Errorf("%s %q contains characters that are not allowed by CB-Spider (letters, digits, and underscores only; dbName may also contain '-')", fieldName, value)
	}
	return nil
}

func getRDBMSSystemId(connectionName, rdbmsName string) (string, string, error) {
	var iidInfo RDBMSIIDInfo
	err := infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, rdbmsName)
	if err != nil {
		return "", "", fmt.Errorf("RDBMS '%s' not found in connection '%s': %w", rdbmsName, connectionName, err)
	}
	return iidInfo.SystemId, iidInfo.NameId, nil
}

// CreateRDBMSDatabase creates a database in the named RDBMS instance, optionally with an explicit
// charset and/or collation (both may be left empty to get the engine/CSP default).
//
// Routing, in priority order:
//  1. If the driver implements rdbmsDatabaseOptionsManager, it is always used (it accepts charset/
//     collation even when both are empty, so this is just the native per-database create call).
//  2. Else, if the driver implements rdbmsDatabaseManager but charset/collation were requested,
//     CB-Spider returns a clear error rather than silently creating the database with the wrong
//     charset/collation -- that driver's CSP-native API has no field for it (see the charset/
//     collation support survey; Tencent, NCP, and NHN are in this bucket today).
//  3. Else (no CSP-native API at all, e.g. AWS, IBM), a direct SQL connection is used, with charset/
//     collation appended as a CHARACTER SET/COLLATE clause (mysql/mariadb only -- see
//     createDatabaseSQL).
//
// CB-Spider does not validate charset/collation against a list of known names (see
// rdbmsDatabaseOptionsManager); it only checks that dbName/charset/collation are safe to embed in
// SQL/identifiers and otherwise passes them through unchanged.
func CreateRDBMSDatabase(connectionName, rdbmsName, dbName, masterUserName, masterUserPassword, charset, collation string) error {
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
	if err := validateSQLSafeValue("dbName", dbName, dbNamePattern); err != nil {
		return err
	}

	charset = strings.TrimSpace(charset)
	if charset != "" {
		if err := validateSQLSafeValue("charset", charset, sqlIdentifierPattern); err != nil {
			return err
		}
	}
	collation = strings.TrimSpace(collation)
	if collation != "" {
		if err := validateSQLSafeValue("collation", collation, sqlIdentifierPattern); err != nil {
			return err
		}
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

	engine := strings.ToLower(string(info.DBEngine))
	if (charset != "" || collation != "") && engine != "mysql" && engine != "mariadb" {
		return fmt.Errorf("charset/collation is only supported for mysql/mariadb RDBMS instances, not %q", info.DBEngine)
	}

	if optMgr, ok := handler.(rdbmsDatabaseOptionsManager); ok {
		// A charset given without a collation is resolved to that charset's actual default
		// collation on this specific instance (queried live, not guessed) before calling the
		// driver -- but only for drivers that actually need this (see
		// rdbmsDatabaseRequiresPairedCollation). Doing this unconditionally for every
		// rdbmsDatabaseOptionsManager driver would be wrong, not just unnecessary: Alibaba also
		// implements this interface, and its driver explicitly rejects a non-empty Collation on
		// MariaDB -- auto-filling one here for a plain Charset-only request would turn a
		// previously-working call into a confusing rejection for something the caller never asked
		// for (this was caught by test/rdbms-mariadb-test/charset-test regressing after that
		// Alibaba check was added).
		if charset != "" && collation == "" {
			if needsPaired, ok := handler.(rdbmsDatabaseRequiresPairedCollation); ok && needsPaired.RequiresCollationPairedWithCharset() {
				resolved, err := lookupDefaultCollation(&info, masterUserName, masterUserPassword, charset)
				if err != nil {
					return fmt.Errorf("CreateRDBMSDatabase: %w", err)
				}
				collation = resolved
			}
		}
		return optMgr.CreateDatabaseWithOptions(driverIId.SystemId, string(info.DBEngine), dbName, charset, collation)
	}

	if charsetMgr, ok := handler.(rdbmsDatabaseCharsetManager); ok {
		// Tencent: charset goes through the native API as usual. Collation has no field in that
		// API at all, so if requested, it's applied with a follow-up ALTER DATABASE over direct
		// SQL. If that ALTER fails, the database now exists with the wrong (default) collation --
		// worse than not existing at all, since it looks like success -- so best-effort roll it
		// back via DeleteDatabase (Tencent also implements rdbmsDatabaseManager) rather than leave
		// a silently-mis-configured database behind.
		if err := charsetMgr.CreateDatabaseWithCharset(driverIId.SystemId, string(info.DBEngine), dbName, charset); err != nil {
			return err
		}
		if collation != "" {
			stmt := "ALTER DATABASE `" + dbName + "` COLLATE `" + collation + "`"
			if err := execSQLStatement(&info, masterUserName, masterUserPassword, stmt); err != nil {
				if dbMgr, ok := handler.(rdbmsDatabaseManager); ok {
					if delErr := dbMgr.DeleteDatabase(driverIId.SystemId, string(info.DBEngine), dbName); delErr != nil {
						return fmt.Errorf("CreateRDBMSDatabase: created database but failed to apply collation: %w (and cleanup also failed: %v)", err, delErr)
					}
				}
				return fmt.Errorf("CreateRDBMSDatabase: created database but failed to apply collation (database was rolled back): %w", err)
			}
		}
		return nil
	}

	if sqlBuilder, ok := handler.(rdbmsDatabaseSQLStatementBuilder); ok {
		// NCP: only route through the driver-built SQL statement (its sys.ncp_create_db stored
		// procedure) when charset/collation is actually requested -- otherwise fall through to the
		// plain rdbmsDatabaseManager case below, preserving existing behavior (and not demanding
		// master credentials) for callers who don't need either.
		if charset != "" || collation != "" {
			stmt, err := sqlBuilder.BuildCreateDatabaseSQL(string(info.DBEngine), dbName, charset, collation)
			if err != nil {
				return fmt.Errorf("CreateRDBMSDatabase: %w", err)
			}
			return execSQLStatement(&info, masterUserName, masterUserPassword, stmt)
		}
	}

	if dbMgr, ok := handler.(rdbmsDatabaseManager); ok {
		if charset != "" || collation != "" {
			// NHN is the only current rdbmsDatabaseManager-only driver whose CSP allows a direct
			// SQL CREATE DATABASE at all (after an instance-level opt-in CB-Spider can't verify --
			// see rdbmsDatabaseSQLFallbackEligible). Anything else landing here (none today) has no
			// charset/collation path whatsoever.
			if sqlEligible, ok := handler.(rdbmsDatabaseSQLFallbackEligible); ok && sqlEligible.IsSQLFallbackEligibleForCharsetCollation() {
				return createDatabaseSQL(&info, masterUserName, masterUserPassword, dbName, charset, collation)
			}
			return fmt.Errorf("charset/collation is not supported for database creation on this CSP's driver; omit both or create the database without them")
		}
		return dbMgr.CreateDatabase(driverIId.SystemId, string(info.DBEngine), dbName)
	}

	// SQL fallback (for drivers without CSP-native DB management API, e.g. AWS, IBM)
	return createDatabaseSQL(&info, masterUserName, masterUserPassword, dbName, charset, collation)
}

// ListRDBMSDatabases lists databases in the named RDBMS instance.
// If the driver supports the CSP-native rdbmsDatabaseManager interface, it is used.
// Otherwise, if masterUserName/masterUserPassword are provided, a direct SQL connection is attempted.
func ListRDBMSDatabases(connectionName, rdbmsName, masterUserName, masterUserPassword string) ([]string, error) {
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
	return listDatabasesSQL(&info, masterUserName, masterUserPassword)
}

// DeleteRDBMSDatabase drops a database from the named RDBMS instance.
// If the driver supports the CSP-native rdbmsDatabaseManager interface, it is used.
// Otherwise, if masterUserName/masterUserPassword are provided, a direct SQL connection is attempted.
func DeleteRDBMSDatabase(connectionName, rdbmsName, dbName, masterUserName, masterUserPassword string) error {
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
	return deleteDatabaseSQL(&info, masterUserName, masterUserPassword, dbName)
}
