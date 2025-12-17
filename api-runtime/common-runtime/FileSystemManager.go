// Cloud Control Manager's Rest Runtime of CB-Spider.
// Common Runtime for FileSystemHandler interface
// by CB-Spider Team

package commonruntime

import (
	"fmt"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	infostore "github.com/cloud-barista/cb-spider/info-store"
)

// -------- IID Info for FileSystem

type FileSystemIIDInfo ZoneLevelIIDInfo

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

	fsSPLock.Lock(connectionName, reqInfo.IId.NameId)
	defer fsSPLock.Unlock(connectionName, reqInfo.IId.NameId)

	exist, err := infostore.HasByConditions(&FileSystemIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, reqInfo.IId.NameId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if exist {
		return nil, fmt.Errorf("FileSystem '%s' already exists", reqInfo.IId.NameId)
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

	info, err := handler.CreateFileSystem(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	spiderIID := cres.IID{NameId: reqInfo.IId.NameId, SystemId: info.IId.SystemId}
	err = infostore.Insert(&FileSystemIIDInfo{ConnectionName: connectionName, ZoneId: reqInfo.Zone, NameId: spiderIID.NameId, SystemId: spiderIID.SystemId})
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
