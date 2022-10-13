// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mini Driver.
//
// by CB-Spider Team, 2021.11.

package connect

import (
	"errors"

	cblog "github.com/cloud-barista/cb-log"
	minirs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mini/resources"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type MiniConnection struct {
	IdentityEndpoint string
	AuthToken string
	ConnectionName string
}

func (cloudConn *MiniConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("Mini Driver: called CreateImageHandler()!")
	handler := minirs.MiniImageHandler{
		MiniAddr : cloudConn.IdentityEndpoint, 
		AuthToken:	cloudConn.AuthToken, 
		ConnectionName:	cloudConn.ConnectionName,
		}


	return &handler, nil
}

func (cloudConn *MiniConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("Mini Driver: called CreateVMSpecHandler()!")
	//handler := minirs.MiniVMSpecHandler{cloudConn.MiniAddr}
	//return &handler, nil

	return nil, errors.New("not implemented")
}

func (cloudConn *MiniConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("Mini Driver: called CreateVMHandler()!")

	return nil, errors.New("not implemented")
}

func (cloudConn *MiniConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("Mini Driver: called CreateVPCHandler()!")

	return nil, errors.New("not implemented")
}

func (cloudConn MiniConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("Mini Driver: called CreateSecurityHandler()!")

	return nil, errors.New("not implemented")
}

func (cloudConn *MiniConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("Mini Driver: called CreateKeyPairHandler()!")

	return nil, errors.New("not implemented")
}

func (cloudConn *MiniConnection) IsConnected() (bool, error) {
	cblogger.Info("Mini Driver: called IsConnected()!")
	if cloudConn == nil {
		return false, nil
	}

	return true, nil
}

func (cloudConn *MiniConnection) Close() error {
	cblogger.Info("Mini Driver: called Close()!")
	return nil
}

func (cloudConn *MiniConnection) CreateNLBHandler() (irs.NLBHandler, error) {
        return nil, errors.New("Mini Driver: not implemented")
}

func (cloudConn *MiniConnection) CreateDiskHandler() (irs.DiskHandler, error) {
        return nil, errors.New("Mini Driver: not implemented")
}

func (cloudConn *MiniConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
        return nil, errors.New("Mini Driver: not implemented")
}

func (cloudConn *MiniConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
        return nil, errors.New("Mini Driver: not implemented")
}


func (cloudConn *MiniConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
        return nil, errors.New("Mini Driver: not implemented")
}

