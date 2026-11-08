// Cloud Driver Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// Stateless mode connection builder: creates a CloudConnection directly from a
// Provider name and inline Credential/Region info, without registering or looking up any
// ConnectionConfig/Credential/Region from the MetaDB.
//
// by CB-Spider Team, 2026.11.

package clouddriverhandler

import (
	"strings"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"
)

// ConnectCloudByProvider builds a CloudConnection directly from the given Provider name and
// inline Credential/Region info, bypassing ConnectionConfig/Credential/Region MetaDB lookups.
// It is used by the Stateless API mode, where no ConnectionName and no IID mapping is managed:
// the caller supplies CSP-native credential and region values on every call, and the driver's
// return values (including SystemId) are passed back unmodified.
func ConnectCloudByProvider(providerName string, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) (icon.CloudConnection, error) {
	providerName = strings.ToUpper(strings.TrimSpace(providerName))

	// use an already-registered Driver of this provider (needed for dynamic-plugin builds),
	// or fall back to the default driver naming convention (sufficient for static builds).
	cldDrvInfo := dim.CloudDriverInfo{
		DriverName:        strings.ToLower(providerName) + "-driver-v1.0",
		ProviderName:      providerName,
		DriverLibFileName: strings.ToLower(providerName) + "-driver-v1.0.so",
	}
	if drvList, listErr := dim.ListCloudDriverByProvider(providerName); listErr == nil && len(drvList) > 0 {
		cldDrvInfo = *drvList[0]
	}

	cldDriver, err := getCloudDriver(cldDrvInfo)
	if err != nil {
		return nil, err
	}

	return cldDriver.ConnectCloud(idrv.ConnectionInfo{
		CredentialInfo: credentialInfo,
		RegionInfo:     regionInfo,
	})
}
