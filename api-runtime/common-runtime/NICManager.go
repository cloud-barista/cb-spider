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

type NICIIDInfo FirstIIDInfo

func (NICIIDInfo) TableName() string {
	return "nic_iid_infos"
}

//====================================================================

func init() {
	db, err := infostore.Open()
	if err != nil {
		cblog.Error(err)
		return
	}
	db.AutoMigrate(&NICIIDInfo{})
	infostore.Close(db)
}

//================ NIC Handler

func RegisterNIC(connectionName string, userIID cres.IID) (*cres.NICInfo, error) {
	cblog.Info("call RegisterNIC()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil { cblog.Error(err); return nil, err }

	emptyPermissionList := []string{}
	if err = ValidateStruct(userIID, emptyPermissionList); err != nil { cblog.Error(err); return nil, err }

	rsType := NIC

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil { cblog.Error(err); return nil, err }

	handler, err := cldConn.CreateNICHandler()
	if err != nil { cblog.Error(err); return nil, err }

	nicSPLock.Lock(connectionName, userIID.NameId)
	defer nicSPLock.Unlock(connectionName, userIID.NameId)

	bool_ret := false
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		bool_ret, err = infostore.HasByCondition(&NICIIDInfo{}, NAME_ID_COLUMN, userIID.NameId)
	} else {
		bool_ret, err = infostore.HasByConditions(&NICIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, userIID.NameId)
	}
	if err != nil { cblog.Error(err); return nil, err }
	if bool_ret {
		err := fmt.Errorf("%s '%s' already exists in connection '%s'", RSTypeString(rsType), userIID.NameId, connectionName)
		cblog.Error(err); return nil, err
	}

	getInfo, err := handler.GetNIC(cres.IID{NameId: getMSShortID(userIID.SystemId), SystemId: userIID.SystemId})
	if err != nil { cblog.Error(err); return nil, err }

	systemId := getMSShortID(getInfo.IId.SystemId)
	spiderIId := cres.IID{NameId: userIID.NameId, SystemId: systemId + ":" + getInfo.IId.SystemId}
	if err = infostore.Insert(&NICIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId}); err != nil {
		cblog.Error(err); return nil, err
	}
	getInfo.IId = userIID
	return &getInfo, nil
}

// (1) check exist
// (2) generate SP-XID
// (3) create Resource
// (4) insert spiderIID
// (5) return userIID
func CreateNIC(connectionName string, rsType string, reqInfo cres.NICReqInfo, IDTransformMode string) (*cres.NICInfo, error) {
	cblog.Info("call CreateNIC()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil { cblog.Error(err); return nil, err }

	if reqInfo.IId.NameId == "" {
		err := fmt.Errorf("NIC Name is empty!")
		cblog.Error(err); return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil { cblog.Error(err); return nil, err }

	handler, err := cldConn.CreateNICHandler()
	if err != nil { cblog.Error(err); return nil, err }

	nicSPLock.Lock(connectionName, reqInfo.IId.NameId)
	defer nicSPLock.Unlock(connectionName, reqInfo.IId.NameId)

	bool_ret := false
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		bool_ret, err = infostore.HasByCondition(&NICIIDInfo{}, NAME_ID_COLUMN, reqInfo.IId.NameId)
	} else {
		bool_ret, err = infostore.HasByConditions(&NICIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, reqInfo.IId.NameId)
	}
	if err != nil { cblog.Error(err); return nil, err }
	if bool_ret {
		err := fmt.Errorf("%s '%s' already exists in connection '%s'", RSTypeString(NIC), reqInfo.IId.NameId, connectionName)
		cblog.Error(err); return nil, err
	}

	//+++ Resolve VPC IID → SystemId
	if reqInfo.VpcIID.NameId != "" {
		var vpcIIDInfo VPCIIDInfo
		if err = infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, reqInfo.VpcIID.NameId); err != nil {
			cblog.Error(err)
			return nil, fmt.Errorf("VPC '%s' not found in connection '%s'", reqInfo.VpcIID.NameId, connectionName)
		}
		reqInfo.VpcIID.SystemId = getDriverSystemId(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})
	}

	//+++ Resolve Subnet IID → SystemId (SubnetIIDInfo has VPC name as owner)
	if reqInfo.SubnetIID.NameId != "" {
		var subnetIIDInfo SubnetIIDInfo
		if err = infostore.GetBy3Conditions(&subnetIIDInfo, CONNECTION_NAME_COLUMN, connectionName, OWNER_VPC_NAME_COLUMN, reqInfo.VpcIID.NameId, NAME_ID_COLUMN, reqInfo.SubnetIID.NameId); err != nil {
			// Fallback: search by connection+name without VPC constraint
			if err2 := infostore.GetByConditions(&subnetIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, reqInfo.SubnetIID.NameId); err2 != nil {
				cblog.Error(err)
				return nil, fmt.Errorf("Subnet '%s' not found in connection '%s'", reqInfo.SubnetIID.NameId, connectionName)
			}
		}
		reqInfo.SubnetIID.SystemId = getDriverSystemId(cres.IID{NameId: subnetIIDInfo.NameId, SystemId: subnetIIDInfo.SystemId})
	}

	//+++ Resolve SecurityGroup IIDs → SystemId
	for i, sg := range reqInfo.SecurityGroupIIDs {
		if sg.NameId != "" && sg.SystemId == "" {
			var sgIIDInfo SGIIDInfo
			if err = infostore.GetBy3Conditions(&sgIIDInfo, CONNECTION_NAME_COLUMN, connectionName, OWNER_VPC_NAME_COLUMN, reqInfo.VpcIID.NameId, NAME_ID_COLUMN, sg.NameId); err != nil {
				if err2 := infostore.GetByConditions(&sgIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, sg.NameId); err2 != nil {
					cblog.Error(err)
					return nil, fmt.Errorf("SecurityGroup '%s' not found in connection '%s'", sg.NameId, connectionName)
				}
			}
			reqInfo.SecurityGroupIIDs[i].SystemId = getDriverSystemId(cres.IID{NameId: sgIIDInfo.NameId, SystemId: sgIIDInfo.SystemId})
		}
	}

	spUUID := ""
	if GetID_MGMT(IDTransformMode) == "ON" {
		spUUID, err = iidm.New(connectionName, rsType, reqInfo.IId.NameId)
		if err != nil { cblog.Error(err); return nil, err }
	} else {
		spUUID = reqInfo.IId.NameId
	}

	reqIId := cres.IID{NameId: reqInfo.IId.NameId, SystemId: spUUID}
	reqInfo.IId = cres.IID{NameId: spUUID, SystemId: ""}

	info, err := handler.CreateNIC(reqInfo)
	if err != nil { cblog.Error(err); return nil, err }

	spiderIId := cres.IID{NameId: reqIId.NameId, SystemId: spUUID + ":" + info.IId.SystemId}
	if err = infostore.Insert(&NICIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId}); err != nil {
		cblog.Error(err)
		handler.DeleteNIC(info.IId)
		return nil, err
	}

	info.IId = getUserIID(cres.IID{NameId: spiderIId.NameId, SystemId: spiderIId.SystemId})
	return &info, nil
}

func ListNIC(connectionName string, rsType string) ([]*cres.NICInfo, error) {
	cblog.Info("call ListNIC()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil { cblog.Error(err); return nil, err }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil { cblog.Error(err); return nil, err }

	handler, err := cldConn.CreateNICHandler()
	if err != nil { cblog.Error(err); return nil, err }

	var iidInfoList []*NICIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		if err = getAuthIIDInfoList(connectionName, &iidInfoList); err != nil { cblog.Error(err); return nil, err }
	} else {
		if err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName); err != nil { cblog.Error(err); return nil, err }
	}

	var infoList []*cres.NICInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		return []*cres.NICInfo{}, nil
	}

	for _, iidInfo := range iidInfoList {
		nicSPLock.RLock(connectionName, iidInfo.NameId)
		info, err := handler.GetNIC(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
		if err != nil {
			nicSPLock.RUnlock(connectionName, iidInfo.NameId)
			cblog.Error(err)
			info = cres.NICInfo{IId: cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}, Status: cres.NICNotFound}
			infoList = append(infoList, &info); continue
		}
		nicSPLock.RUnlock(connectionName, iidInfo.NameId)
		info.IId.NameId = iidInfo.NameId
		resolveNICRelatedIIDs(connectionName, &info)
		infoList = append(infoList, &info)
	}
	return infoList, nil
}

func GetNIC(connectionName string, rsType string, nameID string) (*cres.NICInfo, error) {
	cblog.Info("call GetNIC()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil { cblog.Error(err); return nil, err }
	nameID, err = EmptyCheckAndTrim("nameID", nameID)
	if err != nil { cblog.Error(err); return nil, err }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil { cblog.Error(err); return nil, err }

	handler, err := cldConn.CreateNICHandler()
	if err != nil { cblog.Error(err); return nil, err }

	nicSPLock.RLock(connectionName, nameID)
	defer nicSPLock.RUnlock(connectionName, nameID)

	var iidInfo NICIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*NICIIDInfo
		if err = getAuthIIDInfoList(connectionName, &iidInfoList); err != nil { cblog.Error(err); return nil, err }
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, nameID)
		if err != nil { cblog.Error(err); return nil, err }
		iidInfo = *castedIIDInfo.(*NICIIDInfo)
	} else {
		if err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID); err != nil {
			cblog.Error(err); return nil, err
		}
	}

	info, err := handler.GetNIC(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil { cblog.Error(err); return nil, err }
	info.IId.NameId = iidInfo.NameId
	resolveNICRelatedIIDs(connectionName, &info)
	return &info, nil
}

func DeleteNIC(connectionName string, rsType string, nameID string, force string) (bool, error) {
	cblog.Info("call DeleteNIC()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil { cblog.Error(err); return false, err }
	nameID, err = EmptyCheckAndTrim("nameID", nameID)
	if err != nil { cblog.Error(err); return false, err }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil { cblog.Error(err); return false, err }

	handler, err := cldConn.CreateNICHandler()
	if err != nil { cblog.Error(err); return false, err }

	nicSPLock.Lock(connectionName, nameID)
	defer nicSPLock.Unlock(connectionName, nameID)

	var iidInfo NICIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*NICIIDInfo
		if err = getAuthIIDInfoList(connectionName, &iidInfoList); err != nil { cblog.Error(err); return false, err }
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, nameID)
		if err != nil { cblog.Error(err); return false, err }
		iidInfo = *castedIIDInfo.(*NICIIDInfo)
	} else {
		if err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID); err != nil {
			cblog.Error(err); return false, err
		}
	}

	driverIId := getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
	result, err := handler.DeleteNIC(driverIId)
	if err != nil { cblog.Error(err); if force != "true" { return false, err } }
	if force != "true" && !result { return false, nil }

	_, err = infostore.DeleteByConditions(&NICIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
	if err != nil { cblog.Error(err); return false, err }
	return true, nil
}

// AttachNIC attaches a NIC to a VM.
func AttachNIC(connectionName string, nicName string, vmName string) (*cres.NICInfo, error) {
	cblog.Info("call AttachNIC()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil { cblog.Error(err); return nil, err }
	nicName, err = EmptyCheckAndTrim("nicName", nicName)
	if err != nil { cblog.Error(err); return nil, err }
	vmName, err = EmptyCheckAndTrim("vmName", vmName)
	if err != nil { cblog.Error(err); return nil, err }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil { cblog.Error(err); return nil, err }

	handler, err := cldConn.CreateNICHandler()
	if err != nil { cblog.Error(err); return nil, err }

	nicSPLock.Lock(connectionName, nicName)
	defer nicSPLock.Unlock(connectionName, nicName)

	// Get NIC spiderIID
	var nicIIDInfo NICIIDInfo
	if err = infostore.GetByConditions(&nicIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nicName); err != nil {
		cblog.Error(err); return nil, fmt.Errorf("NIC '%s' not found in connection '%s': %w", nicName, connectionName, err)
	}

	// Get VM spiderIID
	var vmIIDInfo VMIIDInfo
	if err = infostore.GetByConditions(&vmIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, vmName); err != nil {
		cblog.Error(err); return nil, fmt.Errorf("VM '%s' not found in connection '%s': %w", vmName, connectionName, err)
	}

	nicDriverIID := getDriverIID(cres.IID{NameId: nicIIDInfo.NameId, SystemId: nicIIDInfo.SystemId})
	vmDriverIID := getDriverIID(cres.IID{NameId: vmIIDInfo.NameId, SystemId: vmIIDInfo.SystemId})

	info, err := handler.AttachNIC(nicDriverIID, vmDriverIID)
	if err != nil { cblog.Error(err); return nil, err }
	info.IId.NameId = nicName
	return &info, nil
}

// DetachNIC detaches a NIC from its VM.
func DetachNIC(connectionName string, nicName string) (bool, error) {
	cblog.Info("call DetachNIC()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil { cblog.Error(err); return false, err }
	nicName, err = EmptyCheckAndTrim("nicName", nicName)
	if err != nil { cblog.Error(err); return false, err }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil { cblog.Error(err); return false, err }

	handler, err := cldConn.CreateNICHandler()
	if err != nil { cblog.Error(err); return false, err }

	nicSPLock.Lock(connectionName, nicName)
	defer nicSPLock.Unlock(connectionName, nicName)

	var nicIIDInfo NICIIDInfo
	if err = infostore.GetByConditions(&nicIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nicName); err != nil {
		cblog.Error(err); return false, err
	}

	nicDriverIID := getDriverIID(cres.IID{NameId: nicIIDInfo.NameId, SystemId: nicIIDInfo.SystemId})
	return handler.DetachNIC(nicDriverIID)
}

// GetNICOSConfigScript returns the OS-level configuration script for a secondary NIC.
// AWS returns an empty string (no OS config needed). Other CSPs return a bash script
// that must be executed inside the VM after the NIC is attached via the cloud API.
func GetNICOSConfigScript(connectionName string, nicName string) (string, error) {
	cblog.Info("call GetNICOSConfigScript()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil { cblog.Error(err); return "", err }
	nicName, err = EmptyCheckAndTrim("nicName", nicName)
	if err != nil { cblog.Error(err); return "", err }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil { cblog.Error(err); return "", err }

	handler, err := cldConn.CreateNICHandler()
	if err != nil { cblog.Error(err); return "", err }

	var nicIIDInfo NICIIDInfo
	if err = infostore.GetByConditions(&nicIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nicName); err != nil {
		cblog.Error(err)
		return "", fmt.Errorf("NIC '%s' not found in connection '%s': %w", nicName, connectionName, err)
	}
	nicDriverIID := getDriverIID(cres.IID{NameId: nicIIDInfo.NameId, SystemId: nicIIDInfo.SystemId})

	script, err := handler.GetNICOSConfigScript(nicDriverIID)
	if err != nil { cblog.Error(err); return "", err }
	return script, nil
}

func CountAllNICs() (int64, error) {
	var info NICIIDInfo
	count, err := infostore.CountAllNameIDs(&info)
	if err != nil { cblog.Error(err); return count, err }
	return count, nil
}

func CountNICsByConnection(connectionName string) (int64, error) {
	var info NICIIDInfo
	count, err := infostore.CountNameIDsByConnection(&info, connectionName)
	if err != nil { cblog.Error(err); return count, err }
	return count, nil
}

// resolveNICRelatedIIDs resolves VpcIID, SubnetIID, SecurityGroupIIDs, and OwnerVM NameIds
// from CSP SystemIds to Spider NameIds using the IID store.
func resolveNICRelatedIIDs(connectionName string, info *cres.NICInfo) {
	// ---- VPC ----
	// Always look up from IID store: driver may set NameId to CSP-internal name (e.g. Azure VNet name with XID suffix).
	if info.VpcIID.SystemId != "" {
		var vpcInfo VPCIIDInfo
		if err := infostore.GetByContain(&vpcInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, info.VpcIID.SystemId); err == nil {
			info.VpcIID.NameId = vpcInfo.NameId
		}
	}

	// ---- Subnet ----
	// Always look up from IID store for the same reason.
	if info.SubnetIID.SystemId != "" {
		var subnetInfo SubnetIIDInfo
		if err := infostore.GetByContain(&subnetInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, info.SubnetIID.SystemId); err == nil {
			info.SubnetIID.NameId = subnetInfo.NameId
		}
	}

	// ---- Security Groups ----
	// Always look up from IID store: driver may set NameId to CSP internal name (e.g. Spider XID),
	// so we always prefer the Spider NameId from the store.
	for i, sg := range info.SecurityGroupIIDs {
		if sg.SystemId != "" {
			var sgInfo SGIIDInfo
			if err := infostore.GetByContain(&sgInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, sg.SystemId); err == nil {
				info.SecurityGroupIIDs[i].NameId = sgInfo.NameId
			}
		}
	}

	// ---- OwnerVM ----
	// Always look up from IID store: driver sets NameId=SystemId (instance ID), so always prefer Spider NameId.
	// Fallback to SystemId if not found in store (e.g. VM not registered or ID format mismatch).
	if info.Status == cres.NICAttached && info.OwnerVM.SystemId != "" {
		var vmInfo VMIIDInfo
		if err := infostore.GetByContain(&vmInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, info.OwnerVM.SystemId); err == nil {
			info.OwnerVM.NameId = vmInfo.NameId
		} else if info.OwnerVM.NameId == "" {
			info.OwnerVM.NameId = info.OwnerVM.SystemId
		}
	}
}

// AddNICPrivateIP adds a secondary private IP to a NIC.
// If privateIP is empty, the CSP auto-assigns one.
func AddNICPrivateIP(connectionName string, nicName string, privateIP string) (*cres.NICInfo, error) {
	cblog.Info("call AddNICPrivateIP()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil { cblog.Error(err); return nil, err }
	nicName, err = EmptyCheckAndTrim("nicName", nicName)
	if err != nil { cblog.Error(err); return nil, err }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil { cblog.Error(err); return nil, err }

	handler, err := cldConn.CreateNICHandler()
	if err != nil { cblog.Error(err); return nil, err }

	nicSPLock.Lock(connectionName, nicName)
	defer nicSPLock.Unlock(connectionName, nicName)

	var nicIIDInfo NICIIDInfo
	if err = infostore.GetByConditions(&nicIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nicName); err != nil {
		cblog.Error(err); return nil, fmt.Errorf("NIC '%s' not found: %w", nicName, err)
	}

	nicDriverIID := getDriverIID(cres.IID{NameId: nicIIDInfo.NameId, SystemId: nicIIDInfo.SystemId})
	info, err := handler.AddPrivateIP(nicDriverIID, privateIP)
	if err != nil { cblog.Error(err); return nil, err }
	info.IId.NameId = nicName
	resolveNICRelatedIIDs(connectionName, &info)
	return &info, nil
}

// RemoveNICPrivateIP removes a secondary private IP from a NIC.
func RemoveNICPrivateIP(connectionName string, nicName string, privateIP string) (bool, error) {
	cblog.Info("call RemoveNICPrivateIP()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil { cblog.Error(err); return false, err }
	nicName, err = EmptyCheckAndTrim("nicName", nicName)
	if err != nil { cblog.Error(err); return false, err }
	privateIP, err = EmptyCheckAndTrim("privateIP", privateIP)
	if err != nil { cblog.Error(err); return false, err }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil { cblog.Error(err); return false, err }

	handler, err := cldConn.CreateNICHandler()
	if err != nil { cblog.Error(err); return false, err }

	nicSPLock.Lock(connectionName, nicName)
	defer nicSPLock.Unlock(connectionName, nicName)

	var nicIIDInfo NICIIDInfo
	if err = infostore.GetByConditions(&nicIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nicName); err != nil {
		cblog.Error(err); return false, fmt.Errorf("NIC '%s' not found: %w", nicName, err)
	}

	nicDriverIID := getDriverIID(cres.IID{NameId: nicIIDInfo.NameId, SystemId: nicIIDInfo.SystemId})
	return handler.RemovePrivateIP(nicDriverIID, privateIP)
}
