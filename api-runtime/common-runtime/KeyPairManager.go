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
	"os"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
	infostore "github.com/cloud-barista/cb-spider/info-store"
)

// ====================================================================
// type for GORM

type KeyIIDInfo FirstIIDInfo

func (KeyIIDInfo) TableName() string {
	return "key_iid_infos"
}

//====================================================================

func init() {
	db, err := infostore.Open()
	if err != nil {
		cblog.Error(err)
		return
	}
	db.AutoMigrate(&KeyIIDInfo{})
	infostore.Close(db)
}

//================ KeyPair Handler

// UserIID{UserID, CSP-ID} => SpiderIID{UserID, SP-XID:CSP-ID}
// (1) check existence(UserID)
// (2) get resource info(CSP-ID)
// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
// (4) insert spiderIID
func RegisterKey(connectionName string, userIID cres.IID) (*cres.KeyPairInfo, error) {
	cblog.Info("call RegisterKey()")

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

	rsType := KEY

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	keySPLock.Lock(connectionName, userIID.NameId)
	defer keySPLock.Unlock(connectionName, userIID.NameId)

	// (1) check existence(UserID)
	bool_ret, err := infostore.HasByConditions(&KeyIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, userIID.NameId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if bool_ret {
		err := fmt.Errorf(rsType + "-" + userIID.NameId + " already exists!")
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource info(CSP-ID)
	// check existence and get info of this resouce in the CSP
	// Do not user NameId, because Azure driver use it like SystemId
	getInfo, err := handler.GetKey(cres.IID{NameId: userIID.SystemId, SystemId: userIID.SystemId})
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
	// insert KeyPair SpiderIID to metadb
	err = infostore.Insert(&KeyIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// set up KeyPair User IID for return info
	getInfo.IId = userIID
	hideSecretInfo(&getInfo)

	return &getInfo, nil
}

// (1) check exist(NameID)
// (2) generate SP-XID and create reqIID, driverIID
// (3) create Resource
// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
// (5) insert spiderIID
// (6) create userIID
func CreateKey(connectionName string, rsType string, reqInfo cres.KeyPairReqInfo, IDTransformMode string) (*cres.KeyPairInfo, error) {
	cblog.Info("call CreateKey()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	emptyPermissionList := []string{
		"resources.IID:SystemId",
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

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	keySPLock.Lock(connectionName, reqInfo.IId.NameId)
	defer keySPLock.Unlock(connectionName, reqInfo.IId.NameId)

	// (1) check exist(NameID)
	bool_ret := false
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*KeyIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		bool_ret, err = isNameIdExists(&iidInfoList, reqInfo.IId.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		bool_ret, err = infostore.HasByConditions(&KeyIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN,
			reqInfo.IId.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	if bool_ret {
		err := fmt.Errorf(reqInfo.IId.NameId + " already exists!")
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

	// (3) create Resource
	info, err := handler.CreateKey(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
	//     ex) spiderIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	spiderIId := cres.IID{NameId: reqIId.NameId, SystemId: spUUID + ":" + info.IId.SystemId}

	// (5) insert spiderIID
	err = infostore.Insert(&KeyIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId})
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteKey(info.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		cblog.Error(err)
		return nil, err
	}

	// (6) create userIID: {reqNameID, driverSystemID}
	//     ex) userIID {"seoul-service", "i-0bc7123b7e5cbf79d"}
	info.IId = getUserIID(cres.IID{NameId: spiderIId.NameId, SystemId: spiderIId.SystemId})

	return &info, nil
}

// (1) get IID:list
// (2) get KeyInfo:list
func ListKey(connectionName string, rsType string) ([]*cres.KeyPairInfo, error) {
	cblog.Info("call ListKey()")

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

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (1) get IID:list
	var iidInfoList []*KeyIIDInfo
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

	var infoList []*cres.KeyPairInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.KeyPairInfo{}
		return infoList, nil
	}

	// (2) get KeyInfo:list
	infoList2 := []*cres.KeyPairInfo{}
	for _, iidInfo := range iidInfoList {

		keySPLock.RLock(connectionName, iidInfo.NameId)

		// (2) get resource(SystemId)
		info, err := handler.GetKey(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
		if err != nil {
			keySPLock.RUnlock(connectionName, iidInfo.NameId)
			if checkNotFoundError(err) {
				cblog.Info(err)
				continue
			}
			cblog.Error(err)
			return nil, err
		}
		keySPLock.RUnlock(connectionName, iidInfo.NameId)

		info.IId.NameId = iidInfo.NameId
		hideSecretInfo(&info)

		infoList2 = append(infoList2, &info)
	}

	return infoList2, nil
}

func hideSecretInfo(info *cres.KeyPairInfo) {
	info.PublicKey = "Hidden for security."
	info.PrivateKey = "Hidden for security."
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetKey(connectionName string, rsType string, nameID string) (*cres.KeyPairInfo, error) {
	cblog.Info("call GetKey()")

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

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	keySPLock.RLock(connectionName, nameID)
	defer keySPLock.RUnlock(connectionName, nameID)

	// (1) get IID(NameId)
	var iidInfo KeyIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*KeyIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, nameID)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		iidInfo = *castedIIDInfo.(*KeyIIDInfo)
	} else {
		err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	// (2) get resource(SystemId)
	info, err := handler.GetKey(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) set ResourceInfo(IID.NameId)
	info.IId.NameId = iidInfo.NameId
	hideSecretInfo(&info)

	return &info, nil
}

// (1) get spiderIID
// (2) delete Resource(SystemId)
// (3) delete IID
func DeleteKey(connectionName string, rsType string, nameID string, force string) (bool, error) {
	cblog.Info("call DeleteKey()")

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

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	keySPLock.Lock(connectionName, nameID)
	defer keySPLock.Unlock(connectionName, nameID)

	// (1) get spiderIID for creating driverIID
	var iidInfo KeyIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*KeyIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, nameID)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		iidInfo = *castedIIDInfo.(*KeyIIDInfo)
	} else {
		err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	}

	// (2) delete Resource(SystemId)
	driverIId := getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
	result := false
	result, err = handler.(cres.KeyPairHandler).DeleteKey(driverIId)
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
	_, err = infostore.DeleteByConditions(&KeyIIDInfo{}, CONNECTION_NAME_COLUMN, iidInfo.ConnectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}

	return result, nil
}

func CountAllKeys() (int64, error) {
	var info KeyIIDInfo
	count, err := infostore.CountAllNameIDs(&info)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}

func CountKeysByConnection(connectionName string) (int64, error) {
	var info KeyIIDInfo
	count, err := infostore.CountNameIDsByConnection(&info, connectionName)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}
