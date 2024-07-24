// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2024.07.

package commonruntime

import (
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

//================ Tag Handler

// AddTag adds a tag to a resource.
func AddTag(connectionName string, resType cres.RSType, resIID cres.IID, tag cres.KeyValue) (cres.KeyValue, error) {
	cblog.Info("call AddTag()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
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

	return handler.AddTag(resType, resIID, tag)
}

// ListTag lists all tags of a resource.
func ListTag(connectionName string, resType cres.RSType, resIID cres.IID) ([]cres.KeyValue, error) {
	cblog.Info("call ListTag()")

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

	return handler.ListTag(resType, resIID)
}

// GetTag gets a specific tag of a resource.
func GetTag(connectionName string, resType cres.RSType, resIID cres.IID, key string) (cres.KeyValue, error) {
	cblog.Info("call GetTag()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
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

	return handler.GetTag(resType, resIID, key)
}

// RemoveTag removes a specific tag from a resource.
func RemoveTag(connectionName string, resType cres.RSType, resIID cres.IID, key string) (bool, error) {
	cblog.Info("call RemoveTag()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
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

	return handler.RemoveTag(resType, resIID, key)
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
