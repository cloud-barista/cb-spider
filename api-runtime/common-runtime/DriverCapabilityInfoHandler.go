// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2025.02.

package commonruntime

import (
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	ifs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
)

// ================ DriverCapabilityInfo Handler
func GetDriverCapabilityInfo(connectionName string) (ifs.DriverCapabilityInfo, error) {
	cblog.Info("call GetDriverCapabilityInfo()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return ifs.DriverCapabilityInfo{}, err
	}

	cldDriver, err := ccm.GetCloudDriver(connectionName)
	if err != nil {
		return ifs.DriverCapabilityInfo{}, err
	}

	drvCapabilityInfo := cldDriver.GetDriverCapability()

	return drvCapabilityInfo, nil
}
