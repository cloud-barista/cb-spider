// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is interfaces of Cloud Driver.
//
// by CB-Spider Team, 2019.06.

package interfaces

import (
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
)

type DriverCapabilityInfo struct {
	NLBHandler   bool // support: true, do not support: false
	ImageHandler bool // support: true, do not support: false
	VPCHandler   bool // support: true, do not support: false
	//VNetworkHandler bool // support: true, do not support: false
	SecurityHandler bool // support: true, do not support: false
	KeyPairHandler  bool // support: true, do not support: false
	VNicHandler     bool // support: true, do not support: false
	PublicIPHandler bool // support: true, do not support: false
	VMHandler       bool // support: true, do not support: false
	VMSpecHandler   bool // support: true, do not support: false
	DiskHandler     bool // support: true, do not support: false
	MyImageHandler  bool // support: true, do not support: false
	ClusterHandler  bool // support: true, do not support: false

	FIXED_SUBNET_CIDR bool // support: true, do not support: false
	VPC_CIDR          bool // support: true, do not support: false
	SINGLE_VPC        bool // support: true, do not support: false
}

type CredentialInfo struct {
	// @todo TBD
	// key-value pairs
	ClientId         string // Azure Credential
	ClientSecret     string // Azure Credential
	TenantId         string // Azure Credential
	SubscriptionId   string // Azure Credential
	IdentityEndpoint string // OpenStack Credential
	Username         string // OpenStack Credential, Ibm
	Password         string // OpenStack Credential
	DomainName       string // OpenStack Credential
	ProjectID        string // OpenStack Credential
	AuthToken        string // Cloudit Credential
	ClientEmail      string // GCP
	PrivateKey       string // GCP
	Host             string // Docker
	APIVersion       string // Docker
	MockName         string // Mock
	ApiKey           string // Ibm
	ConnectionName   string // MINI
	ClusterId        string // Cloudit
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
