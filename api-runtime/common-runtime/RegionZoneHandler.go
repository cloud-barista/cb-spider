// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2023.09.

package commonruntime

import (
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// ================ RegionZone Handler
func ListRegionZone(connectionName string) ([]*cres.RegionZoneInfo, error) {
	cblog.Info("call ListRegionZone()")

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

	handler, err := cldConn.CreateRegionZoneHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	infoList, err := handler.ListRegionZone()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if infoList == nil || len(infoList) <= 0 {
		infoList = []*cres.RegionZoneInfo{}
	}

	// Set KeyValueList to an empty array if it is nil
	for _, region := range infoList {
		if region.KeyValueList == nil {
			region.KeyValueList = []cres.KeyValue{}
		}
	}

	return infoList, nil
}

func GetRegionZone(connectionName string, nameID string) (*cres.RegionZoneInfo, error) {
	cblog.Info("call GetRegionZone()")

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

	handler, err := cldConn.CreateRegionZoneHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	info, err := handler.GetRegionZone(nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// Set KeyValueList to an empty array if it is nil
	if info.KeyValueList == nil {
		info.KeyValueList = []cres.KeyValue{}
	}

	return &info, nil
}

func ListOrgRegion(connectionName string) (string, error) {
	cblog.Info("call ListOrgRegion()")

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

	handler, err := cldConn.CreateRegionZoneHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	infoList, err := handler.ListOrgRegion()
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return infoList, nil
}

func ListOrgZone(connectionName string) (string, error) {
	cblog.Info("call GetOrgRegionZone()")

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

	handler, err := cldConn.CreateRegionZoneHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}
	info, err := handler.ListOrgZone()
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return info, nil
}
