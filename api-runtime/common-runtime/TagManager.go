// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2024.07.

package commonruntime

import (
	"fmt"
	"strings"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	infostore "github.com/cloud-barista/cb-spider/info-store"
)

//================ Tag Handler

// AddTag adds a tag to a resource.
func AddTag(connectionName string, resType cres.RSType, resName string, tag cres.KeyValue) (cres.KeyValue, error) {
	cblog.Info("call AddTag()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return cres.KeyValue{}, err
	}

	// locking by resource type
	if err := rLockResource(connectionName, resType, resName); err != nil {
		cblog.Error(err)
		return cres.KeyValue{}, err
	}
	defer rUnlockResource(connectionName, resType, resName)

	// get NameId and SystemId of the target resource
	nameId, systemId, err := getIIDInfoByResourceType(connectionName, resType, resName)
	if err != nil {
		cblog.Error(err)
		return cres.KeyValue{}, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return cres.KeyValue{}, err
	}

	handler, err := cldConn.CreateTagHandler()
	if err != nil {
		cblog.Error(err)
		return cres.KeyValue{}, err
	}

	return handler.AddTag(resType, getDriverIID(cres.IID{NameId: nameId, SystemId: systemId}), tag)
}

// ListTag lists all tags of a resource.
func ListTag(connectionName string, resType cres.RSType, resName string) ([]cres.KeyValue, error) {
	cblog.Info("call ListTag()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// locking by resource type
	if err := rLockResource(connectionName, resType, resName); err != nil {
		cblog.Error(err)
		return nil, err
	}
	defer rUnlockResource(connectionName, resType, resName)

	// get NameId and SystemId of the target resource
	nameId, systemId, err := getIIDInfoByResourceType(connectionName, resType, resName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateTagHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return handler.ListTag(resType, getDriverIID(cres.IID{NameId: nameId, SystemId: systemId}))
}

// GetTag gets a specific tag of a resource.
func GetTag(connectionName string, resType cres.RSType, resName string, key string) (cres.KeyValue, error) {
	cblog.Info("call GetTag()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return cres.KeyValue{}, err
	}

	// // locking by resource type
	// if err := rLockResource(connectionName, resType, resName); err != nil {
	// 	cblog.Error(err)
	// 	return cres.KeyValue{}, err
	// }
	// defer rUnlockResource(connectionName, resType, resName)

	// get NameId and SystemId of the target resource
	nameId, systemId, err := getIIDInfoByResourceType(connectionName, resType, resName)
	if err != nil {
		cblog.Error(err)
		return cres.KeyValue{}, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return cres.KeyValue{}, err
	}

	handler, err := cldConn.CreateTagHandler()
	if err != nil {
		cblog.Error(err)
		return cres.KeyValue{}, err
	}

	return handler.GetTag(resType, getDriverIID(cres.IID{NameId: nameId, SystemId: systemId}), key)
}

// RemoveTag removes a specific tag from a resource.
func RemoveTag(connectionName string, resType cres.RSType, resName string, key string) (bool, error) {
	cblog.Info("call RemoveTag()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	// locking by resource type
	if err := rLockResource(connectionName, resType, resName); err != nil {
		cblog.Error(err)
		return false, err
	}
	defer rUnlockResource(connectionName, resType, resName)

	// get NameId and SystemId of the target resource
	nameId, systemId, err := getIIDInfoByResourceType(connectionName, resType, resName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	handler, err := cldConn.CreateTagHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	return handler.RemoveTag(resType, getDriverIID(cres.IID{NameId: nameId, SystemId: systemId}), key)
}

// FindTag finds tags by key or value.
func FindTag(connectionName string, resType cres.RSType, keyword string) ([]*cres.TagInfo, error) {
	cblog.Info("call FindTag()")

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

	handler, err := cldConn.CreateTagHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return handler.FindTag(resType, keyword)
}

// rLockResource locks the resource based on its type.
func rLockResource(connectionName string, resType cres.RSType, resName string) error {
	resType = cres.RSType(strings.ToLower(string(resType)))

	switch resType {
	case cres.VPC, cres.SUBNET:
		vpcSPLock.RLock(connectionName, resName)
	case cres.SG:
		sgSPLock.RLock(connectionName, resName)
	case cres.KEY:
		keySPLock.RLock(connectionName, resName)
	case cres.VM:
		vmSPLock.RLock(connectionName, resName)
	case cres.NLB:
		nlbSPLock.RLock(connectionName, resName)
	case cres.DISK:
		diskSPLock.RLock(connectionName, resName)
	case cres.MYIMAGE:
		myImageSPLock.RLock(connectionName, resName)
	case cres.CLUSTER:
		clusterSPLock.RLock(connectionName, resName)
	default:
		return fmt.Errorf(string(resType) + " is not supported Resource!!")
	}
	return nil
}

// unlockResource unlocks the resource based on its type.
func rUnlockResource(connectionName string, resType cres.RSType, resName string) {
	resType = cres.RSType(strings.ToLower(string(resType)))

	switch resType {
	case cres.VPC, cres.SUBNET:
		vpcSPLock.RUnlock(connectionName, resName)
	case cres.SG:
		sgSPLock.RUnlock(connectionName, resName)
	case cres.KEY:
		keySPLock.RUnlock(connectionName, resName)
	case cres.VM:
		vmSPLock.RUnlock(connectionName, resName)
	case cres.NLB:
		nlbSPLock.RUnlock(connectionName, resName)
	case cres.DISK:
		diskSPLock.RUnlock(connectionName, resName)
	case cres.MYIMAGE:
		myImageSPLock.RUnlock(connectionName, resName)
	case cres.CLUSTER:
		clusterSPLock.RUnlock(connectionName, resName)
	}
}

// getIIDInfoByResourceType gets the IID info for a given resource type and resource name.
func getIIDInfoByResourceType(connectionName string, resType cres.RSType, resName string) (string, string, error) {
	resType = cres.RSType(strings.ToLower(string(resType)))
	switch resType {
	case cres.VPC:
		var info VPCIIDInfo
		err := infostore.GetByConditions(&info, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, resName)
		if err != nil {
			return "", "", err
		}
		return info.NameId, info.SystemId, nil
	case cres.SUBNET:
		var info SubnetIIDInfo
		err := infostore.GetByConditions(&info, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, resName)
		if err != nil {
			return "", "", err
		}
		return info.NameId, info.SystemId, nil
	case cres.SG:
		var info SGIIDInfo
		err := infostore.GetByConditions(&info, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, resName)
		if err != nil {
			return "", "", err
		}
		return info.NameId, info.SystemId, nil
	case cres.KEY:
		var info KeyIIDInfo
		err := infostore.GetByConditions(&info, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, resName)
		if err != nil {
			return "", "", err
		}
		return info.NameId, info.SystemId, nil
	case cres.VM:
		var info VMIIDInfo
		err := infostore.GetByConditions(&info, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, resName)
		if err != nil {
			return "", "", err
		}
		return info.NameId, info.SystemId, nil
	case cres.NLB:
		var info NLBIIDInfo
		err := infostore.GetByConditions(&info, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, resName)
		if err != nil {
			return "", "", err
		}
		return info.NameId, info.SystemId, nil
	case cres.DISK:
		var info DiskIIDInfo
		err := infostore.GetByConditions(&info, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, resName)
		if err != nil {
			return "", "", err
		}
		return info.NameId, info.SystemId, nil
	case cres.MYIMAGE:
		var info MyImageIIDInfo
		err := infostore.GetByConditions(&info, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, resName)
		if err != nil {
			return "", "", err
		}
		return info.NameId, info.SystemId, nil
	case cres.CLUSTER:
		var info ClusterIIDInfo
		err := infostore.GetByConditions(&info, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, resName)
		if err != nil {
			return "", "", err
		}
		return info.NameId, info.SystemId, nil
	default:
		return "", "", fmt.Errorf("unsupported resource type: %s", resType)
	}
}
