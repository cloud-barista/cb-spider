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
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type MockConnection struct {
	MockName	string
}

func (cloudConn *MockConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("Mock Driver: called CreateImageHandler()!")
	imageHandler := mkrs.MockImageHandler{cloudConn.MockName}
	return &imageHandler, nil
}


func (cloudConn *MockConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("Mock Driver: called CreateVMHandler()!")
	/*vmHandler := dkrs.DockerVMHandler{
		Region:         cloudConn.ConnectionInfo.RegionInfo,
		Context:        cloudConn.Context,
		Client:         cloudConn.Client,
	}
	return &vmHandler, nil
	*/
	return nil, nil
}

func (cloudConn *MockConnection) CreateVPCHandler() (irs.VPCHandler, error) {
        cblogger.Error("Mock Driver: called CreateVPCHandler(), but not supported!")
        return nil, nil
}

func (cloudConn MockConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
        cblogger.Error("Mock Driver: called CreateSecurityHandler(), but not supported!")
        return nil, nil
}

func (cloudConn *MockConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
        cblogger.Error("Mock Driver: called CreateKeyPairHandler(), but not supported!")
        return nil, nil
}

func (cloudConn *MockConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
        cblogger.Error("Mock Driver: called CreateVMSpecHandler(), but not supported!")
	return nil, nil
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
