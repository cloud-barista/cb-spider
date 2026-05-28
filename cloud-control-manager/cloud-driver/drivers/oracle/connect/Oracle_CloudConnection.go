package connect

import (
	"context"
	"errors"

	ors "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/oracle/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/oracle/oci-go-sdk/v65/limits"
)

type OracleConnection struct {
	CredentialInfo       idrv.CredentialInfo
	Region               idrv.RegionInfo
	CompartmentID        string
	ComputeClient        core.ComputeClient
	VirtualNetworkClient core.VirtualNetworkClient
	IdentityClient       identity.IdentityClient
	LimitsClient         limits.LimitsClient
	BlockstorageClient   core.BlockstorageClient
	ConfigProvider       common.ConfigurationProvider
	Ctx                  context.Context
	Cancel               context.CancelFunc
}

func (cloudConn *OracleConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	return &ors.OracleVPCHandler{Region: cloudConn.Region, CompartmentID: cloudConn.CompartmentID, Client: cloudConn.VirtualNetworkClient, Ctx: cloudConn.Ctx}, nil
}

func (cloudConn *OracleConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	return &ors.OracleSecurityHandler{Region: cloudConn.Region, CompartmentID: cloudConn.CompartmentID, Client: cloudConn.VirtualNetworkClient, Ctx: cloudConn.Ctx}, nil
}

func (cloudConn *OracleConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	return &ors.OracleKeyPairHandler{CredentialInfo: cloudConn.CredentialInfo, Region: cloudConn.Region}, nil
}

func (cloudConn *OracleConnection) CreateVMHandler() (irs.VMHandler, error) {
	return &ors.OracleVMHandler{CredentialInfo: cloudConn.CredentialInfo, Region: cloudConn.Region, CompartmentID: cloudConn.CompartmentID, ComputeClient: cloudConn.ComputeClient, VirtualNetworkClient: cloudConn.VirtualNetworkClient, BlockstorageClient: cloudConn.BlockstorageClient, Ctx: cloudConn.Ctx}, nil
}

func (cloudConn *OracleConnection) IsConnected() (bool, error) { return true, nil }

func (cloudConn *OracleConnection) Close() error {
	if cloudConn.Cancel != nil {
		cloudConn.Cancel()
	}
	return nil
}

func (cloudConn *OracleConnection) CreateImageHandler() (irs.ImageHandler, error) {
	return &ors.OracleImageHandler{Region: cloudConn.Region, CompartmentID: cloudConn.CompartmentID, Client: cloudConn.ComputeClient, Ctx: cloudConn.Ctx}, nil
}
func (cloudConn *OracleConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	return &ors.OracleVMSpecHandler{Region: cloudConn.Region, CompartmentID: cloudConn.CompartmentID, Client: cloudConn.ComputeClient, Ctx: cloudConn.Ctx}, nil
}
func (cloudConn *OracleConnection) CreateMonitoringHandler() (irs.MonitoringHandler, error) {
	return nil, errors.New("Oracle Driver: MonitoringHandler not implemented")
}
func (cloudConn *OracleConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	return nil, errors.New("Oracle Driver: NLBHandler not implemented")
}
func (cloudConn *OracleConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	return &ors.OracleDiskHandler{
		Region:             cloudConn.Region,
		CompartmentID:      cloudConn.CompartmentID,
		BlockstorageClient: cloudConn.BlockstorageClient,
		ComputeClient:      cloudConn.ComputeClient,
		Ctx:                cloudConn.Ctx,
	}, nil
}
func (cloudConn *OracleConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	return &ors.OracleMyImageHandler{
		Region:        cloudConn.Region,
		CompartmentID: cloudConn.CompartmentID,
		Client:        cloudConn.ComputeClient,
		Ctx:           cloudConn.Ctx,
	}, nil
}
func (cloudConn *OracleConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
	return nil, errors.New("Oracle Driver: ClusterHandler not implemented")
}
func (cloudConn *OracleConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
	return nil, errors.New("Oracle Driver: AnyCallHandler not implemented")
}
func (cloudConn *OracleConnection) CreateRegionZoneHandler() (irs.RegionZoneHandler, error) {
	return &ors.OracleRegionZoneHandler{Region: cloudConn.Region, TenancyID: cloudConn.CredentialInfo.TenantId, Client: cloudConn.IdentityClient, ConfigProvider: cloudConn.ConfigProvider, Ctx: cloudConn.Ctx}, nil
}
func (cloudConn *OracleConnection) CreatePriceInfoHandler() (irs.PriceInfoHandler, error) {
	return nil, errors.New("Oracle Driver: PriceInfoHandler not implemented")
}
func (cloudConn *OracleConnection) CreateTagHandler() (irs.TagHandler, error) {
	return &ors.OracleTagHandler{Region: cloudConn.Region, CompartmentID: cloudConn.CompartmentID, ComputeClient: cloudConn.ComputeClient, NetworkClient: cloudConn.VirtualNetworkClient, Ctx: cloudConn.Ctx}, nil
}
func (cloudConn *OracleConnection) CreateFileSystemHandler() (irs.FileSystemHandler, error) {
	return nil, errors.New("Oracle Driver: FileSystemHandler not implemented")
}
func (cloudConn *OracleConnection) CreateQuotaInfoHandler() (irs.QuotaInfoHandler, error) {
	return &ors.OracleQuotaInfoHandler{
		Region:        cloudConn.Region,
		TenancyID:     cloudConn.CredentialInfo.TenantId,
		CompartmentID: cloudConn.CompartmentID,
		LimitsClient:  cloudConn.LimitsClient,
		Ctx:           cloudConn.Ctx,
	}, nil
}
func (cloudConn *OracleConnection) CreateRDBMSHandler() (irs.RDBMSHandler, error) {
	return nil, errors.New("Oracle Driver: RDBMSHandler not implemented")
}
