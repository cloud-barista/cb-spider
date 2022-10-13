// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Docker Driver.
//
// by CB-Spider Team, 2020.05.

package connect

import (
	"context"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/docker/docker/client"
	dkrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/docker/resources"
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

type DockerCloudConnection struct {
	ConnectionInfo      idrv.ConnectionInfo
	Context		    context.Context
	Client              *client.Client
}

func (cloudConn *DockerCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("Docker Cloud Driver: called CreateImageHandler()!")
	imageHandler := dkrs.DockerImageHandler{cloudConn.ConnectionInfo.RegionInfo, cloudConn.Context, cloudConn.Client}
	return &imageHandler, nil
}


func (cloudConn *DockerCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("Docker Cloud Driver: called CreateVMHandler()!")
	vmHandler := dkrs.DockerVMHandler{
		Region:         cloudConn.ConnectionInfo.RegionInfo,
		Context:        cloudConn.Context,
		Client:         cloudConn.Client,
	}
	return &vmHandler, nil
}

func (cloudConn *DockerCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
        cblogger.Error("Docker Cloud Driver: called CreateVPCHandler(), but not supported!")
        return nil, nil
}

func (cloudConn DockerCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
        cblogger.Error("Docker Cloud Driver: called CreateSecurityHandler(), but not supported!")
        return nil, nil
}

func (cloudConn *DockerCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
        cblogger.Error("Docker Cloud Driver: called CreateKeyPairHandler(), but not supported!")
        return nil, nil
}

func (cloudConn *DockerCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
        cblogger.Error("Docker Cloud Driver: called CreateVMSpecHandler(), but not supported!")
	return nil, nil
}

func (cloudConn *DockerCloudConnection) IsConnected() (bool, error) {
        cblogger.Info("Docker Cloud Driver: called IsConnected()!")
	if cloudConn == nil {
		return false, nil
	}

	if cloudConn.Client == nil {
		return false, nil
	}

	return true, nil
}

func (cloudConn *DockerCloudConnection) Close() error {
        cblogger.Info("Docker Cloud Driver: called Close()!")
	return cloudConn.Client.Close()
}

func (cloudConn *DockerCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
        return nil, errors.New("Docker Cloud Driver NLB: not implemented")
}

func (cloudConn *DockerCloudConnection) CreateDiskHandler() (irs.DiskHandler, error) {
        return nil, errors.New("Docker Driver: not implemented")
}

func (cloudConn *DockerCloudConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
        return nil, errors.New("Docker Driver: not implemented")
}

func (cloudConn *DockerCloudConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
        return nil, errors.New("Docker Driver: not implemented")
}

func (cloudConn *DockerCloudConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
        return nil, errors.New("Docker Driver: not implemented")
}

