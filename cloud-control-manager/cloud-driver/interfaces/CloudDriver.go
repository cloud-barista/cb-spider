// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is interfaces of Cloud Driver.
//
// by powerkim@etri.re.kr, 2019.06.

package interfaces

import (
	//icon "./connect"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
)

type DriverCapabilityInfo struct {
	ImageHandler    bool // support: true, do not support: false
	VNetworkHandler bool // support: true, do not support: false
	SecurityHandler bool // support: true, do not support: false
	KeyPairHandler  bool // support: true, do not support: false
	VNicHandler     bool // support: true, do not support: false
	PublicIPHandler bool // support: true, do not support: false
	VMHandler       bool // support: true, do not support: false
}

type CredentialInfo struct {
	// @todo TBD
	// key-value pairs
	ClientId         string // Azure Credential
	ClientSecret     string // Azure Credential
	TenantId         string // Azure Credential
	SubscriptionId   string // Azure Credential
	IdentityEndpoint string // OpenStack Credential
	Username         string // OpenStack Credential
	Password         string // OpenStack Credential
	DomainName       string // OpenStack Credential
	ProjectID        string // OpenStack Credential
	AuthToken        string // Cloudit Credential
	ClientEmail      string // GCP
	PrivateKey       string // GCP
}

type RegionInfo struct {
	Region        string
	Zone          string
	ResourceGroup string // Azure RegionInfo
}

type ConnectionInfo struct {
	CredentialInfo CredentialInfo
	RegionInfo     RegionInfo
}

type CloudDriver interface {
	GetDriverVersion() string
	GetDriverCapability() DriverCapabilityInfo

	ConnectCloud(connectionInfo ConnectionInfo) (icon.CloudConnection, error)
	//ConnectNetworkCloud(connectionInfo ConnectionInfo) (icon.CloudConnection, error)
}
