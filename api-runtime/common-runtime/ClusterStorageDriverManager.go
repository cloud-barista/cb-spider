// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// CSI storage driver management of a Kubernetes Cluster.
//
// by CB-Spider Team, 2026.10.

package commonruntime

import (
	"fmt"
	"strings"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// ParseStorageDriverKind accepts "Block"/"File" case-insensitively.
func ParseStorageDriverKind(kind string) (cres.StorageDriverKind, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "block":
		return cres.StorageDriverBlock, nil
	case "file":
		return cres.StorageDriverFile, nil
	}
	return "", fmt.Errorf("invalid storage driver kind '%s': must be one of Block, File", kind)
}

// getClusterHandlerAndDriverIID validates the inputs and resolves the Cluster's driver IID.
func getClusterHandlerAndDriverIID(connectionName string, clusterName string) (cres.ClusterHandler, cres.IID, string, string, error) {
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, cres.IID{}, "", "", err
	}

	if err := checkCapability(connectionName, CLUSTER_HANDLER); err != nil {
		return nil, cres.IID{}, "", "", err
	}

	clusterName, err = EmptyCheckAndTrim("clusterName", clusterName)
	if err != nil {
		cblog.Error(err)
		return nil, cres.IID{}, "", "", err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, cres.IID{}, "", "", err
	}

	handler, err := cldConn.CreateClusterHandler()
	if err != nil {
		cblog.Error(err)
		return nil, cres.IID{}, "", "", err
	}

	clusterDriverIID, err := getClusterDriverIID(connectionName, clusterName)
	if err != nil {
		cblog.Error(err)
		return nil, cres.IID{}, "", "", err
	}

	return handler, clusterDriverIID, connectionName, clusterName, nil
}

func ListStorageDriver(connectionName string, clusterName string) ([]cres.StorageDriverInfo, error) {
	cblog.Info("call ListStorageDriver()")

	handler, clusterDriverIID, connectionName, clusterName, err := getClusterHandlerAndDriverIID(connectionName, clusterName)
	if err != nil {
		return nil, err
	}

	clusterSPLock.RLock(connectionName, clusterName)
	defer clusterSPLock.RUnlock(connectionName, clusterName)

	drivers, err := handler.ListStorageDriver(clusterDriverIID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	return drivers, nil
}

func EnableStorageDriver(connectionName string, clusterName string, kind string, version string) (cres.StorageDriverInfo, error) {
	cblog.Info("call EnableStorageDriver()")

	driverKind, err := ParseStorageDriverKind(kind)
	if err != nil {
		cblog.Error(err)
		return cres.StorageDriverInfo{}, err
	}

	handler, clusterDriverIID, connectionName, clusterName, err := getClusterHandlerAndDriverIID(connectionName, clusterName)
	if err != nil {
		return cres.StorageDriverInfo{}, err
	}

	clusterSPLock.Lock(connectionName, clusterName)
	defer clusterSPLock.Unlock(connectionName, clusterName)

	info, err := handler.EnableStorageDriver(clusterDriverIID, driverKind, strings.TrimSpace(version))
	if err != nil {
		cblog.Error(err)
		return cres.StorageDriverInfo{}, err
	}
	return info, nil
}

func DisableStorageDriver(connectionName string, clusterName string, kind string) (bool, error) {
	cblog.Info("call DisableStorageDriver()")

	driverKind, err := ParseStorageDriverKind(kind)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	handler, clusterDriverIID, connectionName, clusterName, err := getClusterHandlerAndDriverIID(connectionName, clusterName)
	if err != nil {
		return false, err
	}

	clusterSPLock.Lock(connectionName, clusterName)
	defer clusterSPLock.Unlock(connectionName, clusterName)

	result, err := handler.DisableStorageDriver(clusterDriverIID, driverKind)
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	return result, nil
}
