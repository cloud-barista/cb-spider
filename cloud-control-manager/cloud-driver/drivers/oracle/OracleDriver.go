package oracle

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/oracle/connect"
	ors "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/oracle/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/oracle/oci-go-sdk/v65/limits"
)

type OracleDriver struct{}

const cspTimeout = 600 * time.Second

func (OracleDriver) GetDriverVersion() string {
	return "ORACLE DRIVER Version 0.1"
}

func (OracleDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	return idrv.DriverCapabilityInfo{
		ZoneBasedControl:  true,
		RegionZoneHandler: true,
		ImageHandler:      true,
		VMSpecHandler:     true,
		VPCHandler:        true,
		SecurityHandler:   true,
		KeyPairHandler:    true,
		VMHandler:         true,
		VPC_CIDR:          true,
		TagHandler:        true,
		QuotaInfoHandler:  true,
		DiskHandler:       true,
		MyImageHandler:    true,
		TagSupportResourceType: []irs.RSType{
			irs.VPC, irs.SUBNET, irs.SG, irs.VM,
		},
	}
}

func (driver *OracleDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	ors.InitLog()
	if err := checkConnectionInfo(connectionInfo); err != nil {
		return nil, err
	}
	connectionInfo.CredentialInfo.ClientSecret = normalizePrivateKeyPEM(connectionInfo.CredentialInfo.ClientSecret)

	provider := common.NewRawConfigurationProvider(
		connectionInfo.CredentialInfo.TenantId,
		connectionInfo.CredentialInfo.ClientId,
		connectionInfo.RegionInfo.Region,
		connectionInfo.CredentialInfo.StsToken,
		connectionInfo.CredentialInfo.ClientSecret,
		nil,
	)

	computeClient, err := core.NewComputeClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}
	virtualNetworkClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}
	identityClient, err := identity.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}
	limitsClient, err := limits.NewLimitsClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}
	blockstorageClient, err := core.NewBlockstorageClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), cspTimeout)
	return &connect.OracleConnection{
		CredentialInfo:       connectionInfo.CredentialInfo,
		Region:               connectionInfo.RegionInfo,
		CompartmentID:        connectionInfo.CredentialInfo.ProjectID,
		ComputeClient:        computeClient,
		VirtualNetworkClient: virtualNetworkClient,
		IdentityClient:       identityClient,
		LimitsClient:         limitsClient,
		BlockstorageClient:   blockstorageClient,
		ConfigProvider:       provider,
		Ctx:                  ctx,
		Cancel:               cancel,
	}, nil
}

func checkConnectionInfo(connectionInfo idrv.ConnectionInfo) error {
	if connectionInfo.RegionInfo.Region == "" {
		return errors.New("not exist Region")
	}
	if connectionInfo.RegionInfo.Zone == "" {
		return errors.New("not exist Zone")
	}
	if connectionInfo.CredentialInfo.TenantId == "" {
		return errors.New("not exist TenantId")
	}
	if connectionInfo.CredentialInfo.ClientId == "" {
		return errors.New("not exist ClientId")
	}
	if connectionInfo.CredentialInfo.ClientSecret == "" {
		return errors.New("not exist ClientSecret")
	}
	if connectionInfo.CredentialInfo.StsToken == "" {
		return errors.New("not exist StsToken")
	}
	if connectionInfo.CredentialInfo.ProjectID == "" {
		return errors.New("not exist ProjectID")
	}
	return nil
}

func normalizePrivateKeyPEM(privateKey string) string {
	privateKey = strings.TrimSpace(privateKey)
	return strings.ReplaceAll(privateKey, `\n`, "\n")
}
