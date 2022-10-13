// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.

package commonruntime

import (
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

//================ VMSpec Handler
func ListVMSpec(connectionName string) ([]*cres.VMSpecInfo, error) {
	cblog.Info("call ListVMSpec()")

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

	handler, err := cldConn.CreateVMSpecHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	infoList, err := handler.ListVMSpec()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if infoList == nil || len(infoList) <= 0 {
		infoList = []*cres.VMSpecInfo{}
	}

	return infoList, nil
}

func GetVMSpec(connectionName string, nameID string) (*cres.VMSpecInfo, error) {
	cblog.Info("call GetVMSpec()")

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

	handler, err := cldConn.CreateVMSpecHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	info, err := handler.GetVMSpec(nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return &info, nil
}

func ListOrgVMSpec(connectionName string) (string, error) {
	cblog.Info("call ListOrgVMSpec()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return "", err
        }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	handler, err := cldConn.CreateVMSpecHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	infoList, err := handler.ListOrgVMSpec()
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return infoList, nil
}

func GetOrgVMSpec(connectionName string, nameID string) (string, error) {
	cblog.Info("call GetOrgVMSpec()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
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

	handler, err := cldConn.CreateVMSpecHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}
	info, err := handler.GetOrgVMSpec(nameID)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return info, nil
}
