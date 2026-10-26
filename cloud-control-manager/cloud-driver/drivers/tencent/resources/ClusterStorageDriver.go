// Tencent Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// CSI storage driver management of a Kubernetes Cluster.
//
// TKE manages its CSI drivers as cluster add-ons:
//   - DescribeAddon / InstallAddon / UpdateAddon / DeleteAddon (tke v20180525)
// Both add-ons can be installed and deleted through the API, so both kinds are Addon mode.
//
// by CB-Spider Team, 2026.10.

package resources

import (
	"fmt"
	"strings"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent/utils/tencent"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
)

const (
	// TKE add-on names (AddonName of DescribeAddon/InstallAddon/DeleteAddon)
	tencentBlockCSIAddon = "cbs"
	// UNCONFIRMED: the API add-on name of the CFS CSI component ("CFS" in the TKE console).
	// Verify with DescribeAddon on a real cluster that has CFS installed.
	tencentFileCSIAddon = "cfs"

	// Kubernetes CSIDriver object names, from tencentcloud kubernetes-csi-tencentcloud.
	// UNCONFIRMED on a TKE-managed add-on; verify with `kubectl get csidriver` on a real cluster.
	tencentBlockCSIDriverName = "com.tencent.cloud.csi.cbs"
	tencentFileCSIDriverName  = "com.tencent.cloud.csi.cfs"
)

type tencentStorageDriverSpec struct {
	addonName     string
	csiDriverName string
}

// tencentStorageDriverSpecOf maps a StorageDriverKind to the TKE add-on.
func tencentStorageDriverSpecOf(kind irs.StorageDriverKind) (tencentStorageDriverSpec, error) {
	switch kind {
	case irs.StorageDriverBlock:
		return tencentStorageDriverSpec{addonName: tencentBlockCSIAddon, csiDriverName: tencentBlockCSIDriverName}, nil
	case irs.StorageDriverFile:
		return tencentStorageDriverSpec{addonName: tencentFileCSIAddon, csiDriverName: tencentFileCSIDriverName}, nil
	default:
		return tencentStorageDriverSpec{}, fmt.Errorf("unsupported storage driver kind: %q (must be %q or %q)", kind, irs.StorageDriverBlock, irs.StorageDriverFile)
	}
}

// tencentStorageDriverStatusOf maps an installed add-on's Phase to a StorageDriverStatus.
// Phases documented by the SDK: Installing, Upgrading, Terminating, Succeeded, InstallFailed, UpgradFailed.
// An add-on that is not installed is Disabled (handled by the caller).
func tencentStorageDriverStatusOf(phase string) irs.StorageDriverStatus {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "succeeded":
		return irs.StorageDriverEnabled
	case "installing", "upgrading", "terminating":
		return irs.StorageDriverUpdating
	case "installfailed", "upgradfailed", "upgradefailed", "failed":
		return irs.StorageDriverFailed
	default:
		return irs.StorageDriverUnknown
	}
}

// tencentStorageDriverInfoOf builds the StorageDriverInfo of a kind from the installed add-on (nil if not installed).
func tencentStorageDriverInfoOf(kind irs.StorageDriverKind, spec tencentStorageDriverSpec, addon *tke.Addon) irs.StorageDriverInfo {
	info := irs.StorageDriverInfo{
		Kind:           kind,
		CSPDriverName:  spec.addonName,
		CSIDriverName:  spec.csiDriverName,
		ManagementMode: irs.StorageDriverAddon,
		Status:         irs.StorageDriverDisabled,
	}
	if addon == nil {
		return info
	}

	phase := ""
	if addon.Phase != nil {
		phase = *addon.Phase
	}
	info.Status = tencentStorageDriverStatusOf(phase)
	if addon.AddonVersion != nil {
		info.Version = *addon.AddonVersion
	}

	var kvs []irs.KeyValue
	if phase != "" {
		kvs = append(kvs, irs.KeyValue{Key: "Phase", Value: phase})
	}
	if addon.Reason != nil && *addon.Reason != "" {
		kvs = append(kvs, irs.KeyValue{Key: "Reason", Value: *addon.Reason})
	}
	if addon.CreateTime != nil && *addon.CreateTime != "" {
		kvs = append(kvs, irs.KeyValue{Key: "CreateTime", Value: *addon.CreateTime})
	}
	info.KeyValueList = kvs
	return info
}

// findTencentAddon returns the add-on with the given name, or nil.
func findTencentAddon(addons []*tke.Addon, name string) *tke.Addon {
	for _, a := range addons {
		if a != nil && a.AddonName != nil && *a.AddonName == name {
			return a
		}
	}
	return nil
}

func newTencentTkeClient(secretId string, secretKey string, regionId string) (*tke.Client, error) {
	credential := common.NewCredential(secretId, secretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	return tke.NewClient(credential, regionId, cpf)
}

// resolveTencentClusterId returns the TKE cluster ID (SystemId, or looked up by NameId).
func (clusterHandler *TencentClusterHandler) resolveTencentClusterId(clusterIID irs.IID) (string, error) {
	if clusterIID.SystemId != "" {
		return clusterIID.SystemId, nil
	}
	if clusterIID.NameId == "" {
		return "", fmt.Errorf("invalid IID: both NameId and SystemId are empty")
	}
	res, err := tencent.GetClusters(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region)
	if err != nil {
		return "", fmt.Errorf("failed to list clusters: %w", err)
	}
	for _, c := range res.Response.Clusters {
		if c != nil && c.ClusterName != nil && *c.ClusterName == clusterIID.NameId && c.ClusterId != nil {
			return *c.ClusterId, nil
		}
	}
	return "", fmt.Errorf("cluster not found: %s", clusterIID.NameId)
}

// getStorageDriverContext returns a TKE client, the cluster ID and the installed add-ons of the cluster.
func (clusterHandler *TencentClusterHandler) getStorageDriverContext(clusterIID irs.IID) (*tke.Client, string, []*tke.Addon, error) {
	clusterId, err := clusterHandler.resolveTencentClusterId(clusterIID)
	if err != nil {
		return nil, "", nil, err
	}
	client, err := newTencentTkeClient(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to create TKE client: %w", err)
	}

	// AddonName is not set: list all add-ons of the cluster, so a missing add-on is not an API error.
	request := tke.NewDescribeAddonRequest()
	request.ClusterId = common.StringPtr(clusterId)
	response, err := client.DescribeAddon(request)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to describe addons: %w", err)
	}
	var addons []*tke.Addon
	if response != nil && response.Response != nil {
		addons = response.Response.Addons
	}
	return client, clusterId, addons, nil
}

func (clusterHandler *TencentClusterHandler) ListStorageDriver(clusterIID irs.IID) ([]irs.StorageDriverInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called ListStorageDriver()")
	callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo, call.CLUSTER, clusterIID.NameId, "ListStorageDriver()")

	start := call.Start()
	_, _, addons, err := clusterHandler.getStorageDriverContext(clusterIID)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("failed to list storage driver: %w", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return nil, err
	}
	calllogger.Info(call.String(callLogInfo))

	result := make([]irs.StorageDriverInfo, 0, 2)
	for _, kind := range []irs.StorageDriverKind{irs.StorageDriverBlock, irs.StorageDriverFile} {
		spec, _ := tencentStorageDriverSpecOf(kind)
		result = append(result, tencentStorageDriverInfoOf(kind, spec, findTencentAddon(addons, spec.addonName)))
	}
	return result, nil
}

func (clusterHandler *TencentClusterHandler) EnableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind, version string) (irs.StorageDriverInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called EnableStorageDriver()")
	callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo, call.CLUSTER, clusterIID.NameId, "EnableStorageDriver()")

	start := call.Start()
	result, err := clusterHandler.enableStorageDriver(clusterIID, kind, strings.TrimSpace(version))
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("failed to enable storage driver: %w", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return irs.StorageDriverInfo{}, err
	}
	calllogger.Info(call.String(callLogInfo))
	return result, nil
}

func (clusterHandler *TencentClusterHandler) enableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind, version string) (irs.StorageDriverInfo, error) {
	spec, err := tencentStorageDriverSpecOf(kind)
	if err != nil {
		return irs.StorageDriverInfo{}, err
	}
	client, clusterId, addons, err := clusterHandler.getStorageDriverContext(clusterIID)
	if err != nil {
		return irs.StorageDriverInfo{}, err
	}

	installed := findTencentAddon(addons, spec.addonName)
	current := tencentStorageDriverInfoOf(kind, spec, installed)

	if installed != nil {
		// idempotent: already enabled, and no other version requested
		if version == "" || version == current.Version {
			return current, nil
		}
		// already enabled with a different version: update the add-on version
		request := tke.NewUpdateAddonRequest()
		request.ClusterId = common.StringPtr(clusterId)
		request.AddonName = common.StringPtr(spec.addonName)
		request.AddonVersion = common.StringPtr(version)
		if _, err := client.UpdateAddon(request); err != nil {
			return irs.StorageDriverInfo{}, fmt.Errorf("failed to update addon %s: %w", spec.addonName, err)
		}
	} else {
		// AddonVersion omitted: TKE installs the latest version
		request := tke.NewInstallAddonRequest()
		request.ClusterId = common.StringPtr(clusterId)
		request.AddonName = common.StringPtr(spec.addonName)
		if version != "" {
			request.AddonVersion = common.StringPtr(version)
		}
		if _, err := client.InstallAddon(request); err != nil {
			return irs.StorageDriverInfo{}, fmt.Errorf("failed to install addon %s: %w", spec.addonName, err)
		}
	}

	// TKE applies add-on changes asynchronously
	result := current
	result.Status = irs.StorageDriverUpdating
	result.Version = version
	return result, nil
}

func (clusterHandler *TencentClusterHandler) DisableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind) (bool, error) {
	cblogger.Info("Tencent Cloud Driver: called DisableStorageDriver()")
	callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo, call.CLUSTER, clusterIID.NameId, "DisableStorageDriver()")

	start := call.Start()
	err := clusterHandler.disableStorageDriver(clusterIID, kind)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("failed to disable storage driver: %w", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return false, err
	}
	calllogger.Info(call.String(callLogInfo))
	return true, nil
}

func (clusterHandler *TencentClusterHandler) disableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind) error {
	spec, err := tencentStorageDriverSpecOf(kind)
	if err != nil {
		return err
	}
	client, clusterId, addons, err := clusterHandler.getStorageDriverContext(clusterIID)
	if err != nil {
		return err
	}

	// idempotent: already disabled
	if findTencentAddon(addons, spec.addonName) == nil {
		return nil
	}

	// Deleting the add-on removes the driver components only; CBS disks and CFS file systems are kept.
	request := tke.NewDeleteAddonRequest()
	request.ClusterId = common.StringPtr(clusterId)
	request.AddonName = common.StringPtr(spec.addonName)
	if _, err := client.DeleteAddon(request); err != nil {
		return fmt.Errorf("failed to delete addon %s: %w", spec.addonName, err)
	}
	return nil
}
