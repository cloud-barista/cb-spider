// Cloud Control Manager's Common Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, August 2026.

package commonruntime

import (
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// ================ DBSpec Handler
func ListDBSpec(connectionName string, dbEngine string) ([]*cres.DBSpecInfo, error) {
	cblog.Info("call ListDBSpec()")

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

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateDBSpecHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	infoList, err := handler.ListDBSpec(dbEngine)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if infoList == nil || len(infoList) <= 0 {
		infoList = []*cres.DBSpecInfo{}
	}

	return infoList, nil
}

func GetDBSpec(connectionName string, dbEngine string, nameID string) (*cres.DBSpecInfo, error) {
	cblog.Info("call GetDBSpec()")

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

	handler, err := cldConn.CreateDBSpecHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	info, err := handler.GetDBSpec(dbEngine, nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return &info, nil
}

func ListOrgDBSpec(connectionName string, dbEngine string) (string, error) {
	cblog.Info("call ListOrgDBSpec()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}
	dbEngine, err = EmptyCheckAndTrim("dbEngine", dbEngine)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	handler, err := cldConn.CreateDBSpecHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	infoList, err := handler.ListOrgDBSpec(dbEngine)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return infoList, nil
}

func GetOrgDBSpec(connectionName string, dbEngine string, nameID string) (string, error) {
	cblog.Info("call GetOrgDBSpec()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}
	dbEngine, err = EmptyCheckAndTrim("dbEngine", dbEngine)
	if err != nil {
		cblog.Error(err)
		return "", err
	}
	nameID, err = EmptyCheckAndTrim("nameID", nameID)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	handler, err := cldConn.CreateDBSpecHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}
	info, err := handler.GetOrgDBSpec(dbEngine, nameID)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return info, nil
}
