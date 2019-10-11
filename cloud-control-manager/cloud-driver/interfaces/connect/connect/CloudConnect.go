// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Connection interfaces of Cloud Driver.
//
// by powerkim@etri.re.kr, 2019.06.

package connect

import (
	irs2 "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type CloudConnection interface {
	CreateImageHandler() (irs2.ImageHandler, error)
	//CreateImageHandler() (irs.ImageHandler, error)
	CreateVNetworkHandler() (irs.VNetworkHandler, error)
	CreateSecurityHandler() (irs.SecurityHandler, error)
	CreateKeyPairHandler() (irs.KeyPairHandler, error)
	CreateVNicHandler() (irs.VNicHandler, error)
	CreatePublicIPHandler() (irs.PublicIPHandler, error)

	CreateVMHandler() (irs.VMHandler, error)

	IsConnected() (bool, error)
	Close() error
}
