// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2024.01.

package commonruntime

import (
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// ================ PriceInfo Handler
func ListProductFamily(connectionName string, regionName string) ([]string, error) {
	cblog.Info("call ListProductFamily()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	regionName, err = EmptyCheckAndTrim("regionName", regionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreatePriceInfoHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	listProductFamily, err := handler.ListProductFamily(regionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// Set KeyValueList to an empty array if it is nil
	if listProductFamily == nil {
		listProductFamily = []string{}
	}

	return listProductFamily, nil
}

func GetPriceInfo(connectionName string, productFamily string, regionName string, filterList []cres.KeyValue) (string, error) {
	cblog.Info("call GetPriceInfo()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	productFamily, err = EmptyCheckAndTrim("productFamily", productFamily)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	regionName, err = EmptyCheckAndTrim("regionName", regionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	handler, err := cldConn.CreatePriceInfoHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}
	priceInfo, err := handler.GetPriceInfo(productFamily, regionName, filterList)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return priceInfo, nil
}
