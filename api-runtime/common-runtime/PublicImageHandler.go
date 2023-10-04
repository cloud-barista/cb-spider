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

func ListImage(connectionName string, rsType string) ([]*cres.ImageInfo, error) {
	cblog.Info("call ListImage()")

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

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	infoList, err := handler.ListImage()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if infoList == nil || len(infoList) <= 0 {
		infoList = []*cres.ImageInfo{}
	}

	return infoList, nil
}

func GetImage(connectionName string, rsType string, nameID string) (*cres.ImageInfo, error) {
	cblog.Info("call GetImage()")

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

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// now, NameID = SystemID
	info, err := handler.GetImage(cres.IID{nameID, nameID})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return &info, nil
}
