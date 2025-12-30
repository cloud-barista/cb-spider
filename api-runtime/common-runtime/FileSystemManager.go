// Cloud Control Manager's Rest Runtime of CB-Spider.
// Common Runtime for FileSystemHandler interface
// by CB-Spider Team

package commonruntime

import (
	"fmt"
	"os"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	infostore "github.com/cloud-barista/cb-spider/info-store"
)

// -------- IID Info for FileSystem

type FileSystemIIDInfo struct {
	ConnectionName string `gorm:"primaryKey"` // ex) "aws-seoul-config"
	ZoneId         string // ex) "ap-southeast-2a"
	NameId         string `gorm:"primaryKey"` // ex) "my_filesystem"
	SystemId       string // ID in CSP, ex) "fs-12345678"
	OwnerVPCName   string // ex) "my_vpc" - NOT primaryKey
}

func (FileSystemIIDInfo) TableName() string {
	return "filesystem_iid_infos"
}

func init() {
	db, err := infostore.Open()
	if err != nil {
		cblog.Error(err)
		return
	}
	db.AutoMigrate(&FileSystemIIDInfo{})
	infostore.Close(db)
}

// -------- FileSystem Common Runtime

func CreateFileSystem(connectionName string, reqInfo cres.FileSystemInfo) (*cres.FileSystemInfo, error) {
	cblog.Info("call CreateFileSystem()")

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
	cblog.Infof("CreateFileSystem - Found VPC in infostore: NameId='%s', SystemId='%s'", vpcIIDInfo.NameId, vpcIIDInfo.SystemId)
	reqInfo.VpcIID = getDriverIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})
	//+++++++++++++++++++++++++++++++++++++++++++

	// AccessSubnetList is optional, set SystemId if provided
	if reqInfo.AccessSubnetList != nil && len(reqInfo.AccessSubnetList) > 0 {
		for idx, subnetIID := range reqInfo.AccessSubnetList {
			var subnetIIDInfo SubnetIIDInfo
			if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
				var iidInfoList []*SubnetIIDInfo
				err = getAuthIIDInfoList(connectionName, &iidInfoList)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
				castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, subnetIID.NameId)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
				subnetIIDInfo = *castedIIDInfo.(*SubnetIIDInfo)
			} else {
				err = infostore.GetByConditions(&subnetIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, subnetIID.NameId)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
			}
			reqInfo.AccessSubnetList[idx] = getDriverIID(cres.IID{NameId: subnetIIDInfo.NameId, SystemId: subnetIIDInfo.SystemId})
		}
	}
	//+++++++++++++++++++++++++++++++++++++++++++

	fsSPLock.Lock(connectionName, reqInfo.IId.NameId)
	defer fsSPLock.Unlock(connectionName, reqInfo.IId.NameId)

	exist, err := infostore.HasByConditions(&FileSystemIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, reqInfo.IId.NameId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if exist {
		return nil, fmt.Errorf("FileSystem '%s' already exists in connection '%s'", reqInfo.IId.NameId, connectionName)
	}

	cldConn, err := ccm.GetZoneLevelCloudConnection(connectionName, reqInfo.Zone)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateFileSystemHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// Log FileSystemType before calling driver
	cblog.Infof("Calling driver CreateFileSystem - Type: '%s', Zone: '%s', VPC: '%s'", reqInfo.FileSystemType, reqInfo.Zone, reqInfo.VpcIID.SystemId)

	info, err := handler.CreateFileSystem(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	spiderIID := cres.IID{NameId: reqInfo.IId.NameId, SystemId: info.IId.SystemId}
	err = infostore.Insert(&FileSystemIIDInfo{ConnectionName: connectionName, ZoneId: reqInfo.Zone, NameId: spiderIID.NameId, SystemId: spiderIID.SystemId, OwnerVPCName: vpcIIDInfo.NameId})
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteFileSystem(info.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf("Failed to delete FileSystem %s: %v", info.IId.NameId, err2)
		}
		cblog.Error(err)
		return nil, err
	}

	info.IId = cres.IID{NameId: spiderIID.NameId, SystemId: spiderIID.SystemId}

	// Set VPC NameId from vpcIIDInfo (already fetched above) - like SecurityGroup pattern
	info.VpcIID = getUserIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: info.VpcIID.SystemId})
	cblog.Infof("CreateFileSystem - Set VpcIID: NameId='%s', SystemId='%s'", info.VpcIID.NameId, info.VpcIID.SystemId)

	// Set NameId for AccessSubnetList
	accessSubnetList := []cres.IID{}
	for _, subnetIID := range info.AccessSubnetList {
		var subnetIIDInfo SubnetIIDInfo
		err := infostore.GetByConditionsAndContain(&subnetIIDInfo, CONNECTION_NAME_COLUMN, connectionName,
			OWNER_VPC_NAME_COLUMN, vpcIIDInfo.NameId, SYSTEM_ID_COLUMN, subnetIID.SystemId)
		if err != nil {
			// if not found, use SystemId only
			if checkNotFoundError(err) {
				cblog.Info(err)
				accessSubnetList = append(accessSubnetList, subnetIID)
				continue
			}
			cblog.Error(err)
			// don't fail the whole creation, just skip this subnet NameId resolution
			accessSubnetList = append(accessSubnetList, subnetIID)
			continue
		}
		if subnetIIDInfo.NameId != "" {
			subnetIID.NameId = subnetIIDInfo.NameId
		}
		accessSubnetList = append(accessSubnetList, subnetIID)
	}
	info.AccessSubnetList = accessSubnetList

	info.KeyValueList = cres.StructToKeyValueList(info)
	return &info, nil
}

func ListFileSystem(connectionName string) ([]*cres.FileSystemInfo, error) {
	cblog.Info("call ListFileSystem()")

	var iidInfoList []*FileSystemIIDInfo
	err := infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
	if err != nil {
		return nil, err
	}
	var infoList []*cres.FileSystemInfo
	for _, iid := range iidInfoList {
		cldConn, err := ccm.GetZoneLevelCloudConnection(connectionName, iid.ZoneId)
		if err != nil {
			return nil, err
		}
		handler, err := cldConn.CreateFileSystemHandler()
		if err != nil {
			return nil, err
		}
		info, err := handler.GetFileSystem(cres.IID{NameId: iid.NameId, SystemId: iid.SystemId})
		if err != nil {
			continue
		}
		info.IId = cres.IID{NameId: iid.NameId, SystemId: iid.SystemId}

		// Set VPC NameId from FileSystemIIDInfo's OwnerVPCName (already stored)
		if iid.OwnerVPCName != "" {
			// Fetch VPC SystemId from infostore to match with driver's SystemId
			var vpcIIDInfo VPCIIDInfo
			err := infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, iid.ConnectionName, NAME_ID_COLUMN, iid.OwnerVPCName)
			if err == nil {
				info.VpcIID.NameId = vpcIIDInfo.NameId
				cblog.Infof("VPC NameId resolved from OwnerVPCName: %s (SystemId: %s)", vpcIIDInfo.NameId, info.VpcIID.SystemId)
			} else {
				cblog.Warnf("Failed to resolve VPC info for OwnerVPCName: %s, err: %v", iid.OwnerVPCName, err)
			}
		} else if info.VpcIID.SystemId != "" {
			// Fallback: try to resolve VPC NameId from SystemId (for old data)
			var vpcIIDInfo VPCIIDInfo
			err := infostore.GetByConditionsAndContain(&vpcIIDInfo, CONNECTION_NAME_COLUMN, iid.ConnectionName,
				NAME_ID_COLUMN, "", SYSTEM_ID_COLUMN, info.VpcIID.SystemId)
			if err == nil && vpcIIDInfo.NameId != "" {
				info.VpcIID.NameId = vpcIIDInfo.NameId
				cblog.Infof("VPC NameId resolved from SystemId (fallback): %s (SystemId: %s)", vpcIIDInfo.NameId, info.VpcIID.SystemId)
			} else {
				cblog.Warnf("Failed to resolve VPC NameId for SystemId: %s, err: %v", info.VpcIID.SystemId, err)
			}
		}

		// Set NameId for AccessSubnetList (use OwnerVPCName if available)
		vpcNameForSubnet := iid.OwnerVPCName
		if vpcNameForSubnet == "" {
			vpcNameForSubnet = info.VpcIID.NameId
		}
		accessSubnetList := []cres.IID{}
		for _, subnetIID := range info.AccessSubnetList {
			if vpcNameForSubnet == "" {
				// If VPC NameId is not available, just use SystemId for subnet
				cblog.Warnf("VPC NameId not available, keeping subnet SystemId only: %s", subnetIID.SystemId)
				accessSubnetList = append(accessSubnetList, subnetIID)
				continue
			}

			var subnetIIDInfo SubnetIIDInfo
			err := infostore.GetByConditionsAndContain(&subnetIIDInfo, CONNECTION_NAME_COLUMN, iid.ConnectionName,
				OWNER_VPC_NAME_COLUMN, vpcNameForSubnet, SYSTEM_ID_COLUMN, subnetIID.SystemId)
			if err != nil {
				// if not found, use SystemId only
				if checkNotFoundError(err) {
					cblog.Info(err)
					accessSubnetList = append(accessSubnetList, subnetIID)
					continue
				}
				cblog.Error(err)
				// don't fail the whole list, just skip this subnet NameId resolution
				accessSubnetList = append(accessSubnetList, subnetIID)
				continue
			}
			if subnetIIDInfo.NameId != "" {
				subnetIID.NameId = subnetIIDInfo.NameId
			}
			accessSubnetList = append(accessSubnetList, subnetIID)
		}
		info.AccessSubnetList = accessSubnetList

		infoList = append(infoList, &info)
	}
	return infoList, nil
}

func GetFileSystem(connectionName string, nameID string) (*cres.FileSystemInfo, error) {
	cblog.Info("call GetFileSystem()")
	var iidInfo FileSystemIIDInfo
	err := infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		return nil, err
	}
	cldConn, err := ccm.GetZoneLevelCloudConnection(connectionName, iidInfo.ZoneId)
	if err != nil {
		return nil, err
	}
	handler, err := cldConn.CreateFileSystemHandler()
	if err != nil {
		return nil, err
	}
	info, err := handler.GetFileSystem(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
	if err != nil {
		return nil, err
	}
	info.IId = cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}

	// Set VPC NameId from FileSystemIIDInfo's OwnerVPCName (already stored)
	if iidInfo.OwnerVPCName != "" {
		// Fetch VPC SystemId from infostore to match with driver's SystemId
		var vpcIIDInfo VPCIIDInfo
		err := infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, iidInfo.ConnectionName, NAME_ID_COLUMN, iidInfo.OwnerVPCName)
		if err == nil {
			info.VpcIID.NameId = vpcIIDInfo.NameId
			cblog.Infof("VPC NameId resolved from OwnerVPCName: %s (SystemId: %s)", vpcIIDInfo.NameId, info.VpcIID.SystemId)
		} else {
			cblog.Warnf("Failed to resolve VPC info for OwnerVPCName: %s, err: %v", iidInfo.OwnerVPCName, err)
		}
	} else if info.VpcIID.SystemId != "" {
		// Fallback: try to resolve VPC NameId from SystemId (for old data)
		var vpcIIDInfo VPCIIDInfo
		err := infostore.GetByConditionsAndContain(&vpcIIDInfo, CONNECTION_NAME_COLUMN, iidInfo.ConnectionName,
			NAME_ID_COLUMN, "", SYSTEM_ID_COLUMN, info.VpcIID.SystemId)
		if err == nil && vpcIIDInfo.NameId != "" {
			info.VpcIID.NameId = vpcIIDInfo.NameId
			cblog.Infof("VPC NameId resolved from SystemId (fallback): %s (SystemId: %s)", vpcIIDInfo.NameId, info.VpcIID.SystemId)
		} else {
			cblog.Warnf("Failed to resolve VPC NameId for SystemId: %s, err: %v", info.VpcIID.SystemId, err)
		}
	}

	// Set NameId for AccessSubnetList (use OwnerVPCName if available)
	vpcNameForSubnet := iidInfo.OwnerVPCName
	if vpcNameForSubnet == "" {
		vpcNameForSubnet = info.VpcIID.NameId
	}
	accessSubnetList := []cres.IID{}
	for _, subnetIID := range info.AccessSubnetList {
		if vpcNameForSubnet == "" {
			// If VPC NameId is not available, just use SystemId for subnet
			cblog.Warnf("VPC NameId not available, keeping subnet SystemId only: %s", subnetIID.SystemId)
			accessSubnetList = append(accessSubnetList, subnetIID)
			continue
		}

		var subnetIIDInfo SubnetIIDInfo
		err := infostore.GetByConditionsAndContain(&subnetIIDInfo, CONNECTION_NAME_COLUMN, iidInfo.ConnectionName,
			OWNER_VPC_NAME_COLUMN, vpcNameForSubnet, SYSTEM_ID_COLUMN, subnetIID.SystemId)
		if err != nil {
			// if not found, use SystemId only
			if checkNotFoundError(err) {
				cblog.Info(err)
				accessSubnetList = append(accessSubnetList, subnetIID)
				continue
			}
			cblog.Error(err)
			return nil, err
		}
		if subnetIIDInfo.NameId != "" {
			subnetIID.NameId = subnetIIDInfo.NameId
		}
		accessSubnetList = append(accessSubnetList, subnetIID)
	}
	info.AccessSubnetList = accessSubnetList

	return &info, nil
}

func DeleteFileSystem(connectionName string, nameID string) (bool, error) {
	cblog.Info("call DeleteFileSystem()")
	var iidInfo FileSystemIIDInfo
	err := infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		return false, err
	}
	cldConn, err := ccm.GetZoneLevelCloudConnection(connectionName, iidInfo.ZoneId)
	if err != nil {
		return false, err
	}
	handler, err := cldConn.CreateFileSystemHandler()
	if err != nil {
		return false, err
	}
	result, err := handler.DeleteFileSystem(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
	if err != nil {
		return false, err
	}
	_, err = infostore.DeleteByConditions(&FileSystemIIDInfo{}, CONNECTION_NAME_COLUMN, iidInfo.ConnectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		return false, err
	}
	return result, nil
}

func AddAccessSubnet(connectionName string, nameID string, subnetIID cres.IID) (*cres.FileSystemInfo, error) {
	cblog.Info("call AddAccessSubnet()")
	var iidInfo FileSystemIIDInfo
	err := infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		return nil, err
	}

	//+++++++++++++++++++++++++++++++++++++++++++
	// set Subnet's SystemId
	var subnetIIDInfo SubnetIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*SubnetIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, subnetIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		subnetIIDInfo = *castedIIDInfo.(*SubnetIIDInfo)
	} else {
		err = infostore.GetByConditions(&subnetIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, subnetIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	subnetIID = getDriverIID(cres.IID{NameId: subnetIIDInfo.NameId, SystemId: subnetIIDInfo.SystemId})
	//+++++++++++++++++++++++++++++++++++++++++++

	cldConn, err := ccm.GetZoneLevelCloudConnection(connectionName, iidInfo.ZoneId)
	if err != nil {
		return nil, err
	}
	handler, err := cldConn.CreateFileSystemHandler()
	if err != nil {
		return nil, err
	}
	info, err := handler.AddAccessSubnet(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}, subnetIID)
	if err != nil {
		return nil, err
	}
	info.IId = cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}
	return &info, nil
}

func RemoveAccessSubnet(connectionName string, nameID string, subnetIID cres.IID) (bool, error) {
	cblog.Info("call RemoveAccessSubnet()")
	var iidInfo FileSystemIIDInfo
	err := infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		return false, err
	}

	//+++++++++++++++++++++++++++++++++++++++++++
	// set Subnet's SystemId
	var subnetIIDInfo SubnetIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*SubnetIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, subnetIID.NameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		subnetIIDInfo = *castedIIDInfo.(*SubnetIIDInfo)
	} else {
		err = infostore.GetByConditions(&subnetIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, subnetIID.NameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	}
	subnetIID = getDriverIID(cres.IID{NameId: subnetIIDInfo.NameId, SystemId: subnetIIDInfo.SystemId})
	//+++++++++++++++++++++++++++++++++++++++++++

	cldConn, err := ccm.GetZoneLevelCloudConnection(connectionName, iidInfo.ZoneId)
	if err != nil {
		return false, err
	}
	handler, err := cldConn.CreateFileSystemHandler()
	if err != nil {
		return false, err
	}
	return handler.RemoveAccessSubnet(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}, subnetIID)
}

func ListAccessSubnet(connectionName string, nameID string) ([]cres.IID, error) {
	cblog.Info("call ListAccessSubnet()")
	var iidInfo FileSystemIIDInfo
	err := infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		return nil, err
	}
	cldConn, err := ccm.GetZoneLevelCloudConnection(connectionName, iidInfo.ZoneId)
	if err != nil {
		return nil, err
	}
	handler, err := cldConn.CreateFileSystemHandler()
	if err != nil {
		return nil, err
	}
	return handler.ListAccessSubnet(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
}
