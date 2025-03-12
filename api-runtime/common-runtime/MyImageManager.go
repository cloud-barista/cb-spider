// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2022.09.

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

type MyImageIIDInfo FirstIIDInfo

func (MyImageIIDInfo) TableName() string {
	return "my_image_iid_infos"
}

//====================================================================

func init() {
	db, err := infostore.Open()
	if err != nil {
		cblog.Error(err)
		return
	}
	db.AutoMigrate(&MyImageIIDInfo{})
	infostore.Close(db)
}

//================ MyImage Handler

// UserIID{UserID, CSP-ID} => SpiderIID{UserID, SP-XID:CSP-ID}
// (1) check existence(UserID)
// (2) get resource info(CSP-ID)
// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
// (4) insert spiderIID
func RegisterMyImage(connectionName string, userIID cres.IID) (*cres.MyImageInfo, error) {
	cblog.Info("call RegisterMyImage()")

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

	rsType := MYIMAGE

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateMyImageHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	myImageSPLock.Lock(connectionName, userIID.NameId)
	defer myImageSPLock.Unlock(connectionName, userIID.NameId)

	// (1) check existence(UserID)
	bool_ret := false
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		bool_ret, err = infostore.HasByCondition(&MyImageIIDInfo{}, NAME_ID_COLUMN, userIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		bool_ret, err = infostore.HasByConditions(&MyImageIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, userIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	if bool_ret {
		err := fmt.Errorf(rsType + "-" + userIID.NameId + " already exists!")
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource info(CSP-ID)
	// check existence and get info of this resouce in the CSP
	// Do not user NameId, because Azure driver use it like SystemId
	getInfo, err := handler.GetMyImage(cres.IID{NameId: getMSShortID(userIID.SystemId), SystemId: userIID.SystemId})
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
	// insert MyImage SpiderIID to metadb
	err = infostore.Insert(&MyImageIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// set up MyImage User IID for return info
	getInfo.IId = userIID

	return &getInfo, nil
}

// (1) check exist(NameID)
// (2) generate SP-XID and create reqIID, driverIID
// (3) create Resource
// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
// (5) insert spiderIID
// (6) create userIID
func SnapshotVM(connectionName string, rsType string, reqInfo cres.MyImageInfo, IDTransformMode string) (*cres.MyImageInfo, error) {
	cblog.Info("call SnapshotVM()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	/*
	   emptyPermissionList := []string{
	           "resources.IID:SystemId",
	           "resources.MyImageInfo:Status",
	   }

	   err = ValidateStruct(reqInfo, emptyPermissionList)
	   if err != nil {
	           cblog.Error(err)
	           return nil, err
	   }
	*/

	myImageSPLock.Lock(connectionName, reqInfo.IId.NameId)
	defer myImageSPLock.Unlock(connectionName, reqInfo.IId.NameId)

	// (1) check exist(NameID)
	bool_ret := false
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		bool_ret, err = infostore.HasByCondition(&MyImageIIDInfo{}, NAME_ID_COLUMN, reqInfo.IId.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		bool_ret, err = infostore.HasByConditions(&MyImageIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN,
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

	// get Source VM's IID(NameId)
	var vmIIdInfo VMIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*VMIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, reqInfo.SourceVM.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		vmIIdInfo = *castedIIDInfo.(*VMIIDInfo)
	} else {
		err = infostore.GetByConditions(&vmIIdInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, reqInfo.SourceVM.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	reqInfo.SourceVM.SystemId = getDriverSystemId(cres.IID{NameId: vmIIdInfo.NameId, SystemId: vmIIdInfo.SystemId})

	cldConn, err := ccm.GetZoneLevelCloudConnection(connectionName, vmIIdInfo.ZoneId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateMyImageHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) create Resource
	info, err := handler.SnapshotVM(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
	//     ex) spiderIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	spiderIId := cres.IID{NameId: reqIId.NameId, SystemId: info.IId.NameId + ":" + info.IId.SystemId}

	// (5) insert spiderIID
	iidInfo := MyImageIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId}
	err = infostore.Insert(&iidInfo)
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteMyImage(info.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		cblog.Error(err)
		return nil, err
	}

	// (6) create userIID: {reqNameID, driverSystemID}
	//     ex) userIID {"seoul-service", "i-0bc7123b7e5cbf79d"}
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

	// Set Source VM's NameId with reqInfo.SourceVM Info
	info.SourceVM.NameId = reqInfo.SourceVM.NameId

	return &info, nil
}

// (1) get IID:list
// (2) get MyImageInfo:list
// (3) set userIID, and ...
func ListMyImage(connectionName string, rsType string) ([]*cres.MyImageInfo, error) {
	cblog.Info("call ListMyImage()")

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

	handler, err := cldConn.CreateMyImageHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (1) get IID:list
	var iidInfoList []*MyImageIIDInfo
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

	var infoList []*cres.MyImageInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.MyImageInfo{}
		return infoList, nil
	}

	// (2) Get MyImageInfo-list with IID-list
	infoList2 := []*cres.MyImageInfo{}
	for _, iidInfo := range iidInfoList {

		myImageSPLock.RLock(connectionName, iidInfo.NameId)

		// get resource(SystemId)
		info, err := handler.GetMyImage(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
		if err != nil {
			myImageSPLock.RUnlock(connectionName, iidInfo.NameId)
			if checkNotFoundError(err) {
				cblog.Error(err)
				info = cres.MyImageInfo{IId: cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}}
				infoList2 = append(infoList2, &info)
				continue
			}
			cblog.Error(err)
			return nil, err
		}
		myImageSPLock.RUnlock(connectionName, iidInfo.NameId)

		info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

		// get Source VM's IID with VM's SystemId
		var vmIIdInfo VMIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*VMIIDInfo
			err := getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, info.SourceVM.SystemId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			vmIIdInfo = *castedIIDInfo.(*VMIIDInfo)
		} else {
			err = infostore.GetByContain(&vmIIdInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, info.SourceVM.SystemId)
			if err != nil {
				cblog.Error(err)
			}
		}
		info.SourceVM.NameId = vmIIdInfo.NameId

		infoList2 = append(infoList2, &info)
	}

	return infoList2, nil
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetMyImage(connectionName string, rsType string, nameID string) (*cres.MyImageInfo, error) {
	cblog.Info("call GetMyImage()")

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

	handler, err := cldConn.CreateMyImageHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	myImageSPLock.RLock(connectionName, nameID)
	defer myImageSPLock.RUnlock(connectionName, nameID)

	// (1) get IID(NameId)
	var iidInfo MyImageIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*MyImageIIDInfo
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
		iidInfo = *castedIIDInfo.(*MyImageIIDInfo)
	} else {
		err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	// (2) get resource(SystemId)
	info, err := handler.GetMyImage(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) set ResourceInfo(IID.NameId)
	// set ResourceInfo
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

	// get Source VM's IID with VM's SystemId
	var vmIIdInfo VMIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*VMIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, info.SourceVM.SystemId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		vmIIdInfo = *castedIIDInfo.(*VMIIDInfo)
	} else {
		err = infostore.GetByContain(&vmIIdInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, info.SourceVM.SystemId)
		if err != nil {
			cblog.Error(err)
		}
	}
	info.SourceVM.NameId = vmIIdInfo.NameId

	return &info, nil
}

func DeleteMyImage(connectionName string, rsType string, nameID string, force string) (bool, error) {
	cblog.Info("call DeleteMyImage()")

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

	handler, err := cldConn.CreateMyImageHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	myImageSPLock.Lock(connectionName, nameID)
	defer myImageSPLock.Unlock(connectionName, nameID)

	// (1) get spiderIID for creating driverIID
	var iidInfo MyImageIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*MyImageIIDInfo
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
		iidInfo = *castedIIDInfo.(*MyImageIIDInfo)
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
	result, err = handler.(cres.MyImageHandler).DeleteMyImage(driverIId)
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
	_, err = infostore.DeleteByConditions(&MyImageIIDInfo{}, CONNECTION_NAME_COLUMN, iidInfo.ConnectionName, NAME_ID_COLUMN, iidInfo.NameId)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}

	return result, nil
}

func CountAllMyImages() (int64, error) {
	var info MyImageIIDInfo
	count, err := infostore.CountAllNameIDs(&info)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}

func CountMyImagesByConnection(connectionName string) (int64, error) {
	var info MyImageIIDInfo
	count, err := infostore.CountNameIDsByConnection(&info, connectionName)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}
