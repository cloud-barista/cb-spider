// Azure Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// CSI storage driver management of a Kubernetes Cluster.
//
// AKS manages its CSI storage drivers through the managed cluster's
// properties.storageProfile, not through a separate add-on API:
//   - Block: storageProfile.diskCSIDriver (CSIDriver disk.csi.azure.com)
//   - File : storageProfile.fileCSIDriver (CSIDriver file.csi.azure.com)
//
// Both are enabled by default on AKS (Kubernetes >= 1.21) and can be toggled
// with a managed cluster PUT. When storageProfile or a driver's Enabled field
// is absent, AKS applies its default (enabled), so nil is reported as enabled.
// AKS has no per-driver version; the driver version follows the cluster version.
// blobCSIDriver and snapshotController are out of scope.
//
// by CB-Spider Team, 2026.10.

package resources

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v9"
	"github.com/Azure/go-autorest/autorest/to"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	azureDiskCSPDriverName = "diskCSIDriver"
	azureFileCSPDriverName = "fileCSIDriver"
	azureDiskCSIDriverName = "disk.csi.azure.com"
	azureFileCSIDriverName = "file.csi.azure.com"

	azureProvisioningSucceeded = "Succeeded"
)

func (ac *AzureClusterHandler) ListStorageDriver(clusterIID irs.IID) ([]irs.StorageDriverInfo, error) {
	hiscallInfo := GetCallLogScheme(ac.Region, call.CLUSTER, clusterIID.NameId, "ListStorageDriver()")
	start := call.Start()

	cluster, err := getRawCluster(clusterIID, ac.ManagedClustersClient, ac.Ctx, ac.CredentialInfo, ac.Region)
	if err != nil {
		listErr := fmt.Errorf("failed to list storage driver: %w", err)
		cblogger.Error(listErr.Error())
		LoggingError(hiscallInfo, listErr)
		return nil, listErr
	}
	LoggingInfo(hiscallInfo, start)

	return []irs.StorageDriverInfo{
		azureStorageDriverInfo(&cluster, irs.StorageDriverBlock),
		azureStorageDriverInfo(&cluster, irs.StorageDriverFile),
	}, nil
}

// EnableStorageDriver sets storageProfile.<driver>.enabled=true with a managed cluster PUT.
// The PUT is long-running; the poller is not awaited and Status Updating is returned.
// AKS has no per-driver version, so a non-empty version is ignored.
func (ac *AzureClusterHandler) EnableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind, version string) (irs.StorageDriverInfo, error) {
	hiscallInfo := GetCallLogScheme(ac.Region, call.CLUSTER, clusterIID.NameId, "EnableStorageDriver()")
	start := call.Start()

	if version != "" {
		cblogger.Warnf("Azure AKS does not support selecting a storage driver version; ignoring version(%s)", version)
	}

	info, err := ac.setStorageDriverEnabled(clusterIID, kind, true)
	if err != nil {
		enableErr := fmt.Errorf("failed to enable storage driver: %w", err)
		cblogger.Error(enableErr.Error())
		LoggingError(hiscallInfo, enableErr)
		return irs.StorageDriverInfo{}, enableErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}

// DisableStorageDriver sets storageProfile.<driver>.enabled=false with a managed cluster PUT.
// AKS removes the driver components only; disks, file shares and PVs are kept.
func (ac *AzureClusterHandler) DisableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind) (bool, error) {
	hiscallInfo := GetCallLogScheme(ac.Region, call.CLUSTER, clusterIID.NameId, "DisableStorageDriver()")
	start := call.Start()

	_, err := ac.setStorageDriverEnabled(clusterIID, kind, false)
	if err != nil {
		disableErr := fmt.Errorf("failed to disable storage driver: %w", err)
		cblogger.Error(disableErr.Error())
		LoggingError(hiscallInfo, disableErr)
		return false, disableErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// setStorageDriverEnabled toggles one storageProfile flag. It is idempotent: when the
// flag already has the requested value, no CSP call is made and the current info is returned.
func (ac *AzureClusterHandler) setStorageDriverEnabled(clusterIID irs.IID, kind irs.StorageDriverKind, enable bool) (irs.StorageDriverInfo, error) {
	if !isValidAzureStorageDriverKind(kind) {
		return irs.StorageDriverInfo{}, fmt.Errorf("invalid StorageDriverKind(%s): must be %s or %s", kind, irs.StorageDriverBlock, irs.StorageDriverFile)
	}

	cluster, err := getRawCluster(clusterIID, ac.ManagedClustersClient, ac.Ctx, ac.CredentialInfo, ac.Region)
	if err != nil {
		return irs.StorageDriverInfo{}, err
	}
	if cluster.Properties == nil {
		return irs.StorageDriverInfo{}, fmt.Errorf("cluster(%s) has no properties", to.String(cluster.Name))
	}

	if azureStorageDriverEnabled(cluster.Properties.StorageProfile, kind) == enable {
		return azureStorageDriverInfo(&cluster, kind), nil
	}

	provisioningState := to.String(cluster.Properties.ProvisioningState)
	if provisioningState != azureProvisioningSucceeded {
		return irs.StorageDriverInfo{}, fmt.Errorf("cluster(%s) provisioningState is %q (another operation is in progress or the cluster is not ready); retry after it becomes %q",
			to.String(cluster.Name), provisioningState, azureProvisioningSucceeded)
	}

	setAzureStorageDriverEnabled(cluster.Properties, kind, enable)

	// Same PUT pattern as UpgradeCluster (current cluster object, one field changed),
	// but the long-running poller is not awaited; callers poll ListStorageDriver.
	_, err = ac.ManagedClustersClient.BeginCreateOrUpdate(ac.Ctx, ac.Region.Region, to.String(cluster.Name), cluster, nil)
	if err != nil {
		return irs.StorageDriverInfo{}, err
	}

	info := azureStorageDriverInfo(&cluster, kind)
	info.Status = irs.StorageDriverUpdating
	return info, nil
}

func isValidAzureStorageDriverKind(kind irs.StorageDriverKind) bool {
	return kind == irs.StorageDriverBlock || kind == irs.StorageDriverFile
}

// azureStorageDriverEnabled reports the configured flag. A nil profile, nil driver
// struct or nil Enabled means the AKS default, which is enabled for disk and file
// (Kubernetes >= 1.21).
func azureStorageDriverEnabled(profile *armcontainerservice.ManagedClusterStorageProfile, kind irs.StorageDriverKind) bool {
	if profile == nil {
		return true
	}
	switch kind {
	case irs.StorageDriverBlock:
		if profile.DiskCSIDriver == nil || profile.DiskCSIDriver.Enabled == nil {
			return true
		}
		return *profile.DiskCSIDriver.Enabled
	case irs.StorageDriverFile:
		if profile.FileCSIDriver == nil || profile.FileCSIDriver.Enabled == nil {
			return true
		}
		return *profile.FileCSIDriver.Enabled
	}
	return false
}

func setAzureStorageDriverEnabled(props *armcontainerservice.ManagedClusterProperties, kind irs.StorageDriverKind, enable bool) {
	if props.StorageProfile == nil {
		props.StorageProfile = &armcontainerservice.ManagedClusterStorageProfile{}
	}
	switch kind {
	case irs.StorageDriverBlock:
		props.StorageProfile.DiskCSIDriver = &armcontainerservice.ManagedClusterStorageProfileDiskCSIDriver{Enabled: toBoolPtr(enable)}
	case irs.StorageDriverFile:
		props.StorageProfile.FileCSIDriver = &armcontainerservice.ManagedClusterStorageProfileFileCSIDriver{Enabled: toBoolPtr(enable)}
	}
}

// azureStorageDriverStatus maps the configured flag and the cluster provisioningState.
// While the cluster is in a non-terminal state (Creating, Updating, Upgrading, ...),
// the storageProfile may not be applied yet, so Updating is reported.
func azureStorageDriverStatus(enabled bool, provisioningState string) irs.StorageDriverStatus {
	switch strings.ToLower(provisioningState) {
	case "succeeded", "failed", "canceled":
		if enabled {
			return irs.StorageDriverEnabled
		}
		return irs.StorageDriverDisabled
	case "":
		return irs.StorageDriverUnknown
	default:
		return irs.StorageDriverUpdating
	}
}

func azureStorageDriverInfo(cluster *armcontainerservice.ManagedCluster, kind irs.StorageDriverKind) irs.StorageDriverInfo {
	info := irs.StorageDriverInfo{
		Kind:           kind,
		ManagementMode: irs.StorageDriverAddon,
		Status:         irs.StorageDriverUnknown,
	}
	switch kind {
	case irs.StorageDriverBlock:
		info.CSPDriverName = azureDiskCSPDriverName
		info.CSIDriverName = azureDiskCSIDriverName
	case irs.StorageDriverFile:
		info.CSPDriverName = azureFileCSPDriverName
		info.CSIDriverName = azureFileCSIDriverName
		info.RequiresFileSystem = false // the driver creates storage accounts and file shares dynamically
	}
	if cluster == nil || cluster.Properties == nil {
		return info
	}

	provisioningState := to.String(cluster.Properties.ProvisioningState)
	info.Status = azureStorageDriverStatus(azureStorageDriverEnabled(cluster.Properties.StorageProfile, kind), provisioningState)
	info.KeyValueList = []irs.KeyValue{
		{Key: "ClusterProvisioningState", Value: provisioningState},
		{Key: "ClusterKubernetesVersion", Value: to.String(cluster.Properties.CurrentKubernetesVersion)},
	}
	return info
}
