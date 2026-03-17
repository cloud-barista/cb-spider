// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2025.07.

package commonruntime

import (
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// ================ Quota Info Handler

// ListQuotaServiceType returns the list of service type names available
// for quota queries on the given connection.
func ListQuotaServiceType(connectionName string) ([]string, error) {
	cblog.Info("call ListQuotaServiceType()")

	// check empty and trim user input
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if err := checkCapability(connectionName, QUOTA_INFO_HANDLER); err != nil {
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateQuotaInfoHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	serviceTypes, err := handler.ListServiceType()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if serviceTypes == nil {
		serviceTypes = []string{}
	}

	return serviceTypes, nil
}

// GetQuotaInfo retrieves the quota limits and current usage for the
// specified service type on the given connection.
// No filtering or sorting is applied; CSP-original values are passed through.
func GetQuotaInfo(connectionName string, serviceType string) (cres.QuotaInfo, error) {
	cblog.Info("call GetQuotaInfo()")

	// check empty and trim user input
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return cres.QuotaInfo{}, err
	}

	serviceType, err = EmptyCheckAndTrim("serviceType", serviceType)
	if err != nil {
		cblog.Error(err)
		return cres.QuotaInfo{}, err
	}

	if err := checkCapability(connectionName, QUOTA_INFO_HANDLER); err != nil {
		return cres.QuotaInfo{}, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return cres.QuotaInfo{}, err
	}

	handler, err := cldConn.CreateQuotaInfoHandler()
	if err != nil {
		cblog.Error(err)
		return cres.QuotaInfo{}, err
	}

	quotaInfo, err := handler.GetQuotaInfo(serviceType)
	if err != nil {
		cblog.Error(err)
		return cres.QuotaInfo{}, err
	}

	// Ensure the slice is never nil
	if quotaInfo.Quotas == nil {
		quotaInfo.Quotas = []cres.Quota{}
	}

	return quotaInfo, nil
}
