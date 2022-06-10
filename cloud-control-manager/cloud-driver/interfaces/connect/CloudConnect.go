// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Connection interfaces of Cloud Driver.
//
// by CB-Spider Team, 2019.06.

package connect

import (
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type CloudConnection interface {
	CreateImageHandler() (irs.ImageHandler, error)
	//CreateVNetworkHandler() (irs.VNetworkHandler, error)
	CreateVPCHandler() (irs.VPCHandler, error)

	CreateSecurityHandler() (irs.SecurityHandler, error)
	CreateKeyPairHandler() (irs.KeyPairHandler, error)
	//CreateVNicHandler() (irs.VNicHandler, error)
	//CreatePublicIPHandler() (irs.PublicIPHandler, error)

	CreateVMHandler() (irs.VMHandler, error)

	CreateVMSpecHandler() (irs.VMSpecHandler, error)
	CreateNLBHandler() (irs.NLBHandler, error)

	IsConnected() (bool, error)
	Close() error
}
