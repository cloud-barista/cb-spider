// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mock Driver: CSI storage driver management of a Kubernetes Cluster.
// The Block driver is BuiltIn (always enabled).
//
// by CB-Spider Team, 2026.10.

package resources

import (
	"fmt"

	cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

func mockBlockStorageDriverInfo() irs.StorageDriverInfo {
	return irs.StorageDriverInfo{
		Kind:           irs.StorageDriverBlock,
		CSPDriverName:  "mock-block-csi",
		CSIDriverName:  "block.csi.mock.cloud-barista.org",
		ManagementMode: irs.StorageDriverBuiltIn,
		Status:         irs.StorageDriverEnabled,
	}
}

func (clusterHandler *MockClusterHandler) ListStorageDriver(clusterIID irs.IID) ([]irs.StorageDriverInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListStorageDriver()!")

	if _, err := clusterHandler.GetCluster(clusterIID); err != nil {
		return nil, err
	}
	return []irs.StorageDriverInfo{mockBlockStorageDriverInfo()}, nil
}

func (clusterHandler *MockClusterHandler) EnableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind, version string) (irs.StorageDriverInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called EnableStorageDriver()!")

	if kind != irs.StorageDriverBlock {
		return irs.StorageDriverInfo{}, fmt.Errorf("Mock Driver: storage driver kind(%s) is not supported; only Block is supported", kind)
	}
	if _, err := clusterHandler.GetCluster(clusterIID); err != nil {
		return irs.StorageDriverInfo{}, err
	}
	return mockBlockStorageDriverInfo(), nil
}

func (clusterHandler *MockClusterHandler) DisableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called DisableStorageDriver()!")

	if kind != irs.StorageDriverBlock {
		return false, fmt.Errorf("Mock Driver: storage driver kind(%s) is not supported; only Block is supported", kind)
	}
	if _, err := clusterHandler.GetCluster(clusterIID); err != nil {
		return false, err
	}
	return false, fmt.Errorf("Mock Driver: the Block storage driver is BuiltIn and cannot be disabled")
}
