// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mock Driver.
//
// by CB-Spider Team, 2020.05.

package connect

import (
	cblog "github.com/cloud-barista/cb-log"
	mkrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	"errors"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type MockConnection struct {
	Region   idrv.RegionInfo
	MockName string
}

func (cloudConn *MockConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("Mock Driver: called CreateImageHandler()!")
	handler := mkrs.MockImageHandler{cloudConn.MockName}
	return &handler, nil
}

func (cloudConn *MockConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("Mock Driver: called CreateVMHandler()!")
	handler := mkrs.MockVMHandler{cloudConn.Region, cloudConn.MockName}
	return &handler, nil
}

func (cloudConn *MockConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("Mock Driver: called CreateVPCHandler()!")
	handler := mkrs.MockVPCHandler{cloudConn.MockName}
	return &handler, nil
}

func (cloudConn MockConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("Mock Driver: called CreateSecurityHandler()!")
	handler := mkrs.MockSecurityHandler{cloudConn.MockName}
	return &handler, nil
}

func (cloudConn *MockConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("Mock Driver: called CreateKeyPairHandler()!")
	handler := mkrs.MockKeyPairHandler{cloudConn.MockName}
	return &handler, nil
}

func (cloudConn *MockConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("Mock Driver: called CreateVMSpecHandler()!")
	handler := mkrs.MockVMSpecHandler{cloudConn.MockName}
	return &handler, nil
}

func (cloudConn *MockConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	cblogger.Info("Mock Driver: called CreateNLBHandler()!")
	handler := mkrs.MockNLBHandler{cloudConn.MockName}
	return &handler, nil
}

func (cloudConn *MockConnection) IsConnected() (bool, error) {
	cblogger.Info("Mock Driver: called IsConnected()!")
	if cloudConn == nil {
		return false, nil
	}

	return true, nil
}

func (cloudConn *MockConnection) Close() error {
	cblogger.Info("Mock Driver: called Close()!")
	return nil
}

func (cloudConn *MockConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	cblogger.Info("Mock Driver: called CreateDiskHandler()!")
	handler := mkrs.MockDiskHandler{cloudConn.MockName}
	return &handler, nil
}

func (cloudConn *MockConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
        return nil, errors.New("Mock Driver: not implemented")
}

func (cloudConn *MockConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	cblogger.Info("Mock Driver: called CreateMyImageHandler()!")
	handler := mkrs.MockMyImageHandler{cloudConn.MockName}
	return &handler, nil
}

func (cloudConn *MockConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
        cblogger.Info("Mock Driver: called CreateAnyCallHandler()!")
        handler := mkrs.MockAnyCallHandler{cloudConn.MockName}
        return &handler, nil
}

