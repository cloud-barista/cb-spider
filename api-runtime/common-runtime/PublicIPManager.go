// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2025.06.

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

type PublicIPIIDInfo FirstIIDInfo

func (PublicIPIIDInfo) TableName() string {
	return "publicip_iid_infos"
}

//====================================================================

func init() {
	db, err := infostore.Open()
	if err != nil {
		cblog.Error(err)
		return
	}
	db.AutoMigrate(&PublicIPIIDInfo{})
	infostore.Close(db)
}

//================ PublicIP Handler

// UserIID{UserID, CSP-ID} => SpiderIID{UserID, SP-XID:CSP-ID}
// (1) check existence(UserID)
// (2) get resource info(CSP-ID)
// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
// (4) insert spiderIID
func RegisterPublicIP(connectionName string, userIID cres.IID) (*cres.PublicIPInfo, error) {
	cblog.Info("call RegisterPublicIP()")

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

	rsType := PUBLICIP

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	publicipSPLock.Lock(connectionName, userIID.NameId)
	defer publicipSPLock.Unlock(connectionName, userIID.NameId)

	// (1) check existence(UserID)
	bool_ret := false
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		bool_ret, err = infostore.HasByCondition(&PublicIPIIDInfo{}, NAME_ID_COLUMN, userIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		bool_ret, err = infostore.HasByConditions(&PublicIPIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, userIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	if bool_ret {
		err := fmt.Errorf("%s '%s' already exists in connection '%s'", RSTypeString(rsType), userIID.NameId, connectionName)
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource info(CSP-ID)
	getInfo, err := handler.GetPublicIP(cres.IID{NameId: getMSShortID(userIID.SystemId), SystemId: userIID.SystemId})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
	systemId := getMSShortID(getInfo.IId.SystemId)
	spiderIId := cres.IID{NameId: userIID.NameId, SystemId: systemId + ":" + getInfo.IId.SystemId}

	// (4) insert spiderIID
	err = infostore.Insert(&PublicIPIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	getInfo.IId = userIID
	return &getInfo, nil
}

// (1) check exist(NameID)
// (2) generate SP-XID and create reqIID, driverIID
// (3) create Resource
// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
// (5) insert spiderIID
// (6) create userIID
func CreatePublicIP(connectionName string, rsType string, reqInfo cres.PublicIPInfo, IDTransformMode string) (*cres.PublicIPInfo, error) {
	cblog.Info("call CreatePublicIP()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if reqInfo.IId.NameId == "" {
		err := fmt.Errorf("PublicIP Name is empty!")
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	publicipSPLock.Lock(connectionName, reqInfo.IId.NameId)
	defer publicipSPLock.Unlock(connectionName, reqInfo.IId.NameId)

	// (1) check exist(NameID)
	bool_ret := false
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		bool_ret, err = infostore.HasByCondition(&PublicIPIIDInfo{}, NAME_ID_COLUMN, reqInfo.IId.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		bool_ret, err = infostore.HasByConditions(&PublicIPIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, reqInfo.IId.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	if bool_ret {
		// Idempotent check: if the resource exists in both IID store and the CSP, return it
		// (handles browser timeout + retry scenario where Spider registered but client got error)
		var existingIID PublicIPIIDInfo
		if getErr := infostore.GetByConditions(&existingIID, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, reqInfo.IId.NameId); getErr == nil {
			driverIID := getDriverIID(cres.IID{NameId: existingIID.NameId, SystemId: existingIID.SystemId})
			if info, getCSPErr := handler.GetPublicIP(driverIID); getCSPErr == nil {
				info.IId.NameId = existingIID.NameId
				cblog.Info("CreatePublicIP: returning existing PublicIP (idempotent): " + existingIID.NameId)
				return &info, nil
			}
		}
		err := fmt.Errorf("%s '%s' already exists in connection '%s'", RSTypeString(PUBLICIP), reqInfo.IId.NameId, connectionName)
		cblog.Error(err)
		return nil, err
	}

	spUUID := ""
	if GetID_MGMT(IDTransformMode) == "ON" {
		// (2) generate SP-XID
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
	info, err := handler.CreatePublicIP(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
	spiderIId := cres.IID{NameId: reqIId.NameId, SystemId: spUUID + ":" + info.IId.SystemId}

	// (5) insert spiderIID
	err = infostore.Insert(&PublicIPIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId})
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeletePublicIP(info.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		return nil, err
	}

	// (6) create userIID: {reqNameID, driverSystemID}
	info.IId = getUserIID(cres.IID{NameId: spiderIId.NameId, SystemId: spiderIId.SystemId})

	return &info, nil
}

// (1) get IID:list
// (2) get PublicIPInfo:list
func ListPublicIP(connectionName string, rsType string) ([]*cres.PublicIPInfo, error) {
	cblog.Info("call ListPublicIP()")

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

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (1) get IID:list
	var iidInfoList []*PublicIPIIDInfo
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

	var infoList []*cres.PublicIPInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.PublicIPInfo{}
		return infoList, nil
	}

	// (2) get PublicIPInfo:list
	infoList2 := []*cres.PublicIPInfo{}
	for _, iidInfo := range iidInfoList {
		publicipSPLock.RLock(connectionName, iidInfo.NameId)
		info, err := handler.GetPublicIP(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
		if err != nil {
			publicipSPLock.RUnlock(connectionName, iidInfo.NameId)
			cblog.Error(err)
			info = cres.PublicIPInfo{IId: cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}, Status: cres.PublicIPNotFound}
			infoList2 = append(infoList2, &info)
			continue
		}
		publicipSPLock.RUnlock(connectionName, iidInfo.NameId)
		info.IId.NameId = iidInfo.NameId
		// Resolve OwnedVM SystemId → Spider NameId
		if info.OwnedVM.SystemId != "" && info.OwnedVM.NameId == info.OwnedVM.SystemId {
			var vmIIdInfo VMIIDInfo
			if lookupErr := infostore.GetByContain(&vmIIdInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, info.OwnedVM.SystemId); lookupErr == nil {
				info.OwnedVM.NameId = vmIIdInfo.NameId
			}
		}
		// Resolve OwnedNIC SystemId → Spider NameId
		if info.OwnedNIC.SystemId != "" {
			var nicIIdInfo NICIIDInfo
			if lookupErr := infostore.GetByContain(&nicIIdInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, info.OwnedNIC.SystemId); lookupErr == nil {
				info.OwnedNIC.NameId = nicIIdInfo.NameId
			} else {
				info.OwnedNIC.NameId = info.OwnedNIC.SystemId
			}
		}
		infoList2 = append(infoList2, &info)
	}

	return infoList2, nil
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetPublicIP(connectionName string, rsType string, nameID string) (*cres.PublicIPInfo, error) {
	cblog.Info("call GetPublicIP()")

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

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	publicipSPLock.RLock(connectionName, nameID)
	defer publicipSPLock.RUnlock(connectionName, nameID)

	// (1) get IID(NameId)
	var iidInfo PublicIPIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*PublicIPIIDInfo
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
		iidInfo = *castedIIDInfo.(*PublicIPIIDInfo)
	} else {
		err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	// (2) get resource(SystemId)
	info, err := handler.GetPublicIP(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) set ResourceInfo(IID.NameId)
	info.IId.NameId = iidInfo.NameId
	// Resolve OwnedVM SystemId → Spider NameId
	if info.OwnedVM.SystemId != "" && info.OwnedVM.NameId == info.OwnedVM.SystemId {
		var vmIIdInfo VMIIDInfo
		if lookupErr := infostore.GetByContain(&vmIIdInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, info.OwnedVM.SystemId); lookupErr == nil {
			info.OwnedVM.NameId = vmIIdInfo.NameId
		}
	}
	// Resolve OwnedNIC SystemId → Spider NameId
	if info.OwnedNIC.SystemId != "" {
		var nicIIdInfo NICIIDInfo
		if lookupErr := infostore.GetByContain(&nicIIdInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, info.OwnedNIC.SystemId); lookupErr == nil {
			info.OwnedNIC.NameId = nicIIdInfo.NameId
		} else {
			info.OwnedNIC.NameId = info.OwnedNIC.SystemId
		}
	}

	return &info, nil
}

// (1) get spiderIID
// (2) delete Resource(SystemId)
// (3) delete IID
func DeletePublicIP(connectionName string, rsType string, nameID string, force string) (bool, error) {
	cblog.Info("call DeletePublicIP()")

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

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	publicipSPLock.Lock(connectionName, nameID)
	defer publicipSPLock.Unlock(connectionName, nameID)

	// (1) get spiderIID for creating driverIID
	var iidInfo PublicIPIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*PublicIPIIDInfo
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
		iidInfo = *castedIIDInfo.(*PublicIPIIDInfo)
	} else {
		err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	}

	// (2) delete Resource(SystemId)
	driverIId := getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
	result, err := handler.DeletePublicIP(driverIId)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}

	if force != "true" && !result {
		return false, nil
	}

	// (3) delete IID
	_, err = infostore.DeleteByConditions(&PublicIPIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	return true, nil
}

// AssociatePublicIP associates a PublicIP with a NIC (or VM for NCP).
// vmName: used for NCP (VM-level NAT). nicName+privateIP: used for other CSPs.
// GCP: nicName holds the NIC identifier (e.g. "my-vm/nic0"), privateIP is ignored.
func AssociatePublicIP(connectionName string, publicIPName string, vmName string, nicName string, privateIP string) (*cres.PublicIPInfo, error) {
	cblog.Info("call AssociatePublicIP()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil { cblog.Error(err); return nil, err }
	publicIPName, err = EmptyCheckAndTrim("publicIPName", publicIPName)
	if err != nil { cblog.Error(err); return nil, err }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil { cblog.Error(err); return nil, err }

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil { cblog.Error(err); return nil, err }

	publicipSPLock.Lock(connectionName, publicIPName)
	defer publicipSPLock.Unlock(connectionName, publicIPName)

	// Get spiderIID for the PublicIP
	var pipIIDInfo PublicIPIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*PublicIPIIDInfo
		if err = getAuthIIDInfoList(connectionName, &iidInfoList); err != nil { cblog.Error(err); return nil, err }
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, publicIPName)
		if err != nil { cblog.Error(err); return nil, err }
		pipIIDInfo = *castedIIDInfo.(*PublicIPIIDInfo)
	} else {
		if err = infostore.GetByConditions(&pipIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, publicIPName); err != nil {
			cblog.Error(err); return nil, err
		}
	}
	pipDriverIID := getDriverIID(cres.IID{NameId: pipIIDInfo.NameId, SystemId: pipIIDInfo.SystemId})

	// Resolve VM IID (used by NCP)
	vmDriverIID := cres.IID{}
	if vmName != "" {
		var vmIIDInfo VMIIDInfo
		if err = infostore.GetByConditions(&vmIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, vmName); err != nil {
			cblog.Error(err); return nil, fmt.Errorf("VM '%s' not found in connection '%s': %w", vmName, connectionName, err)
		}
		vmDriverIID = getDriverIID(cres.IID{NameId: vmIIDInfo.NameId, SystemId: vmIIDInfo.SystemId})
	}

	// Resolve NIC IID (used by non-NCP/GCP CSPs)
	nicDriverIID := cres.IID{}
	if nicName != "" {
		var nicIIDInfo NICIIDInfo
		if lookupErr := infostore.GetByConditions(&nicIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nicName); lookupErr != nil {
			// For GCP, nicName may be "vmName/nic0" which is not in the NIC IID store — use as-is
			nicDriverIID = cres.IID{NameId: nicName, SystemId: nicName}
		} else {
			nicDriverIID = getDriverIID(cres.IID{NameId: nicIIDInfo.NameId, SystemId: nicIIDInfo.SystemId})
		}
	}

	info, err := handler.AssociatePublicIP(pipDriverIID, vmDriverIID, nicDriverIID, privateIP)
	if err != nil { cblog.Error(err); return nil, err }
	info.IId.NameId = publicIPName
	return &info, nil
}

// DisassociatePublicIP removes the VM association from a PublicIP.
func DisassociatePublicIP(connectionName string, publicIPName string) (bool, error) {
	cblog.Info("call DisassociatePublicIP()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	publicIPName, err = EmptyCheckAndTrim("publicIPName", publicIPName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	publicipSPLock.Lock(connectionName, publicIPName)
	defer publicipSPLock.Unlock(connectionName, publicIPName)

	var pipIIDInfo PublicIPIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*PublicIPIIDInfo
		if err = getAuthIIDInfoList(connectionName, &iidInfoList); err != nil {
			cblog.Error(err)
			return false, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, publicIPName)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		pipIIDInfo = *castedIIDInfo.(*PublicIPIIDInfo)
	} else {
		if err = infostore.GetByConditions(&pipIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, publicIPName); err != nil {
			cblog.Error(err)
			return false, err
		}
	}

	pipDriverIID := getDriverIID(cres.IID{NameId: pipIIDInfo.NameId, SystemId: pipIIDInfo.SystemId})
	return handler.DisassociatePublicIP(pipDriverIID)
}

func CountAllPublicIPs() (int64, error) {
	var info PublicIPIIDInfo
	count, err := infostore.CountAllNameIDs(&info)
	if err != nil {
		cblog.Error(err)
		return count, err
	}
	return count, nil
}

func CountPublicIPsByConnection(connectionName string) (int64, error) {
	var info PublicIPIIDInfo
	count, err := infostore.CountNameIDsByConnection(&info, connectionName)
	if err != nil {
		cblog.Error(err)
		return count, err
	}
	return count, nil
}

// ListIID lists all PublicIP IIDs registered in Spider (no CSP call).
func ListPublicIPIID(connectionName string) ([]*cres.IID, error) {
	cblog.Info("call ListPublicIPIID()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var iidInfoList []*PublicIPIIDInfo
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

	iidList := []*cres.IID{}
	for _, iidInfo := range iidInfoList {
		iidList = append(iidList, &cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
	}
	return iidList, nil
}
