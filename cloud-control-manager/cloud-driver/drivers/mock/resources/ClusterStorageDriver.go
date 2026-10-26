// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mock Driver: CSI storage driver management of a Kubernetes Cluster.
// Block is BuiltIn (always enabled) and File is an Addon that starts disabled.
//
// by CB-Spider Team, 2026.10.

package resources

import (
	"fmt"
	"sync"

	cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// key: MockName + "/" + cluster NameId
var mockFileDriverEnabled = map[string]bool{}
var mockStorageDriverLock = new(sync.Mutex)

func (clusterHandler *MockClusterHandler) storageDriverKey(clusterIID irs.IID) (string, error) {
	if _, err := clusterHandler.GetCluster(clusterIID); err != nil {
		return "", err
	}
	return clusterHandler.MockName + "/" + clusterIID.NameId, nil
}

func mockStorageDriverInfo(kind irs.StorageDriverKind, fileEnabled bool) irs.StorageDriverInfo {
	if kind == irs.StorageDriverBlock {
		return irs.StorageDriverInfo{
			Kind:           irs.StorageDriverBlock,
			CSPDriverName:  "mock-block-csi",
			CSIDriverName:  "block.csi.mock.cloud-barista.org",
			ManagementMode: irs.StorageDriverBuiltIn,
			Status:         irs.StorageDriverEnabled,
		}
	}
	status := irs.StorageDriverDisabled
	if fileEnabled {
		status = irs.StorageDriverEnabled
	}
	return irs.StorageDriverInfo{
		Kind:           irs.StorageDriverFile,
		CSPDriverName:  "mock-file-csi",
		CSIDriverName:  "file.csi.mock.cloud-barista.org",
		ManagementMode: irs.StorageDriverAddon,
		Status:         status,
	}
}

func (clusterHandler *MockClusterHandler) ListStorageDriver(clusterIID irs.IID) ([]irs.StorageDriverInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListStorageDriver()!")

	key, err := clusterHandler.storageDriverKey(clusterIID)
	if err != nil {
		return nil, err
	}

	mockStorageDriverLock.Lock()
	defer mockStorageDriverLock.Unlock()
	fileEnabled := mockFileDriverEnabled[key]
	return []irs.StorageDriverInfo{
		mockStorageDriverInfo(irs.StorageDriverBlock, fileEnabled),
		mockStorageDriverInfo(irs.StorageDriverFile, fileEnabled),
	}, nil
}

func (clusterHandler *MockClusterHandler) EnableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind, version string) (irs.StorageDriverInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called EnableStorageDriver()!")

	key, err := clusterHandler.storageDriverKey(clusterIID)
	if err != nil {
		return irs.StorageDriverInfo{}, err
	}

	mockStorageDriverLock.Lock()
	defer mockStorageDriverLock.Unlock()
	switch kind {
	case irs.StorageDriverBlock:
		return mockStorageDriverInfo(kind, mockFileDriverEnabled[key]), nil
	case irs.StorageDriverFile:
		mockFileDriverEnabled[key] = true
		return mockStorageDriverInfo(kind, true), nil
	}
	return irs.StorageDriverInfo{}, fmt.Errorf("Mock Driver: unknown storage driver kind '%s'", kind)
}

func (clusterHandler *MockClusterHandler) DisableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called DisableStorageDriver()!")

	key, err := clusterHandler.storageDriverKey(clusterIID)
	if err != nil {
		return false, err
	}

	mockStorageDriverLock.Lock()
	defer mockStorageDriverLock.Unlock()
	switch kind {
	case irs.StorageDriverBlock:
		return false, fmt.Errorf("Mock Driver: the Block storage driver is BuiltIn and cannot be disabled")
	case irs.StorageDriverFile:
		delete(mockFileDriverEnabled, key)
		return true, nil
	}
	return false, fmt.Errorf("Mock Driver: unknown storage driver kind '%s'", kind)
}
