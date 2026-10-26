// IBM Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// CSI storage driver management of a Kubernetes Cluster.
//
// IKS (VPC) manages its CSI drivers as cluster add-ons:
//   - GET   /v1/clusters/{idOrName}/addons  (GetClusterAddons)
//   - PATCH /v1/clusters/{idOrName}/addons  (ManageClusterAddons, enable/update)
//   - GET   /v1/addons                      (GetAddons, available add-on versions)
// Both add-ons can be enabled and disabled through the API, so both kinds are Addon mode.
//
// by CB-Spider Team, 2026.10.

package resources

import (
	"errors"
	"fmt"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibm/utils/kubernetesserviceapiv1"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/hashicorp/go-version"
)

const (
	// IKS add-on names (as shown by `ibmcloud ks cluster addon ls`)
	ibmBlockCSIAddon = "vpc-block-csi-driver" // enabled by default on VPC clusters, can be disabled
	ibmFileCSIAddon  = "vpc-file-csi-driver"  // disabled by default

	// Kubernetes CSIDriver object names
	ibmBlockCSIDriverName = "vpc.block.csi.ibm.io"
	ibmFileCSIDriverName  = "vpc.file.csi.ibm.io"
)

type ibmStorageDriverSpec struct {
	addonName     string
	csiDriverName string
}

// ibmStorageDriverSpecOf maps a StorageDriverKind to the IKS add-on.
func ibmStorageDriverSpecOf(kind irs.StorageDriverKind) (ibmStorageDriverSpec, error) {
	switch kind {
	case irs.StorageDriverBlock:
		return ibmStorageDriverSpec{addonName: ibmBlockCSIAddon, csiDriverName: ibmBlockCSIDriverName}, nil
	case irs.StorageDriverFile:
		return ibmStorageDriverSpec{addonName: ibmFileCSIAddon, csiDriverName: ibmFileCSIDriverName}, nil
	default:
		return ibmStorageDriverSpec{}, fmt.Errorf("Unsupported storage driver kind: %q (must be %q or %q)", kind, irs.StorageDriverBlock, irs.StorageDriverFile)
	}
}

// ibmStorageDriverStatusOf maps an installed add-on's healthState to a StorageDriverStatus.
// An add-on that is not installed is Disabled (handled by the caller).
func ibmStorageDriverStatusOf(healthState string) irs.StorageDriverStatus {
	switch strings.ToLower(strings.TrimSpace(healthState)) {
	case "normal":
		return irs.StorageDriverEnabled
	case "pending", "deploying", "updating", "upgrading", "enabling", "disabling":
		return irs.StorageDriverUpdating
	case "critical", "warning", "error", "failed":
		return irs.StorageDriverFailed
	default:
		return irs.StorageDriverUnknown
	}
}

// ibmStorageDriverInfoOf builds the StorageDriverInfo of a kind from the installed add-on (nil if not installed).
func ibmStorageDriverInfoOf(kind irs.StorageDriverKind, spec ibmStorageDriverSpec, addon *kubernetesserviceapiv1.ClusterAddon) irs.StorageDriverInfo {
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

	healthState := ""
	if addon.HealthState != nil {
		healthState = *addon.HealthState
	}
	info.Status = ibmStorageDriverStatusOf(healthState)
	// IKS may return only healthStatus (e.g., "Enabling") without healthState.
	if info.Status == irs.StorageDriverUnknown && addon.HealthStatus != nil {
		info.Status = ibmStorageDriverStatusOf(*addon.HealthStatus)
	}
	if addon.Version != nil {
		info.Version = *addon.Version
	}

	var kvs []irs.KeyValue
	if healthState != "" {
		kvs = append(kvs, irs.KeyValue{Key: "HealthState", Value: healthState})
	}
	if addon.HealthStatus != nil && *addon.HealthStatus != "" {
		kvs = append(kvs, irs.KeyValue{Key: "HealthStatus", Value: *addon.HealthStatus})
	}
	if addon.TargetVersion != nil && *addon.TargetVersion != "" {
		kvs = append(kvs, irs.KeyValue{Key: "TargetVersion", Value: *addon.TargetVersion})
	}
	if addon.Deprecated != nil {
		kvs = append(kvs, irs.KeyValue{Key: "Deprecated", Value: fmt.Sprintf("%t", *addon.Deprecated)})
	}
	info.KeyValueList = kvs
	return info
}

// findIbmClusterAddon returns the add-on with the given name, or nil.
func findIbmClusterAddon(addons []kubernetesserviceapiv1.ClusterAddon, name string) *kubernetesserviceapiv1.ClusterAddon {
	for i := range addons {
		if addons[i].Name != nil && *addons[i].Name == name {
			return &addons[i]
		}
	}
	return nil
}

// normalizeIbmKubeVersion turns an IKS master version (e.g. "1.30.4_1530", "1.29.8_1546 --> 1.30.4_1530")
// into a plain semantic version ("1.30.4"). For an in-progress upgrade, the current (left) version is used.
func normalizeIbmKubeVersion(masterKubeVersion string) string {
	v := strings.TrimSpace(masterKubeVersion)
	if idx := strings.Index(v, "-->"); idx >= 0 {
		v = strings.TrimSpace(v[:idx])
	}
	if idx := strings.IndexAny(v, "_ "); idx >= 0 {
		v = v[:idx]
	}
	return v
}

// ibmAddonSupportsKubeVersion checks a supportedKubeRange such as ">=1.20.0 <1.31.0" against a cluster version.
// Unlike checkIfClusterIsSupported(), it never panics on an unexpected range format.
func ibmAddonSupportsKubeVersion(supportedKubeRange string, kubeVersion string) bool {
	clusterVersion, err := version.NewVersion(kubeVersion)
	if err != nil {
		return false
	}
	constraintStr := strings.Join(strings.Fields(supportedKubeRange), ",")
	if constraintStr == "" {
		return false
	}
	constraints, err := version.NewConstraint(constraintStr)
	if err != nil {
		return false
	}
	return constraints.Check(clusterVersion)
}

// pickIbmDefaultAddonVersion selects the add-on version in the same way as installAutoScalerAddon():
// the first available version of the add-on whose supportedKubeRange covers the cluster version.
// It returns "" when no candidate is found; the caller then omits the version and IKS uses its default.
func pickIbmDefaultAddonVersion(availableAddons []kubernetesserviceapiv1.AddonCommon, addonName string, kubeVersion string) string {
	for _, a := range availableAddons {
		if a.Name == nil || *a.Name != addonName || a.Version == nil || a.SupportedKubeRange == nil {
			continue
		}
		if ibmAddonSupportsKubeVersion(*a.SupportedKubeRange, kubeVersion) {
			return *a.Version
		}
	}
	return ""
}

// getStorageDriverContext resolves the cluster ID, resource group, master version and installed add-ons.
func (ic *IbmClusterHandler) getStorageDriverContext(clusterIID irs.IID) (clusterId string, resourceGroupId string, kubeVersion string, addons []kubernetesserviceapiv1.ClusterAddon, err error) {
	rawCluster, err := ic.getRawCluster(clusterIID)
	if err != nil {
		return "", "", "", nil, err
	}
	resourceGroupId, err = ic.getDefaultResourceGroupId()
	if err != nil {
		return "", "", "", nil, fmt.Errorf("Failed to get resource group: %w", err)
	}
	addons, _, err = ic.ClusterService.GetClusterAddonsWithContext(ic.Ctx, &kubernetesserviceapiv1.GetClusterAddonsOptions{
		IdOrName:           core.StringPtr(rawCluster.Id),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
	})
	if err != nil {
		return "", "", "", nil, fmt.Errorf("Failed to get cluster addons: %w", err)
	}
	return rawCluster.Id, resourceGroupId, normalizeIbmKubeVersion(rawCluster.MasterKubeVersion), addons, nil
}

func (ic *IbmClusterHandler) ListStorageDriver(clusterIID irs.IID) ([]irs.StorageDriverInfo, error) {
	hiscallInfo := GetCallLogScheme(ic.Region, call.CLUSTER, clusterIID.NameId, "ListStorageDriver()")
	start := call.Start()

	if clusterIID.NameId == "" && clusterIID.SystemId == "" {
		return nil, errors.New("Failed to List Storage Driver. err = invalid IID")
	}

	_, _, _, addons, err := ic.getStorageDriverContext(clusterIID)
	if err != nil {
		err = fmt.Errorf("Failed to List Storage Driver: %w", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	result := make([]irs.StorageDriverInfo, 0, 2)
	for _, kind := range []irs.StorageDriverKind{irs.StorageDriverBlock, irs.StorageDriverFile} {
		spec, _ := ibmStorageDriverSpecOf(kind)
		result = append(result, ibmStorageDriverInfoOf(kind, spec, findIbmClusterAddon(addons, spec.addonName)))
	}

	LoggingInfo(hiscallInfo, start)
	return result, nil
}

func (ic *IbmClusterHandler) EnableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind, version string) (irs.StorageDriverInfo, error) {
	hiscallInfo := GetCallLogScheme(ic.Region, call.CLUSTER, clusterIID.NameId, "EnableStorageDriver()")
	start := call.Start()

	if clusterIID.NameId == "" && clusterIID.SystemId == "" {
		return irs.StorageDriverInfo{}, errors.New("Failed to Enable Storage Driver. err = invalid IID")
	}
	spec, err := ibmStorageDriverSpecOf(kind)
	if err != nil {
		err = fmt.Errorf("Failed to Enable Storage Driver: %w", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.StorageDriverInfo{}, err
	}
	version = strings.TrimSpace(version)

	clusterId, resourceGroupId, kubeVersion, addons, err := ic.getStorageDriverContext(clusterIID)
	if err != nil {
		err = fmt.Errorf("Failed to Enable Storage Driver: %w", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.StorageDriverInfo{}, err
	}

	installed := findIbmClusterAddon(addons, spec.addonName)
	current := ibmStorageDriverInfoOf(kind, spec, installed)

	options := &kubernetesserviceapiv1.ManageClusterAddonsOptions{
		IdOrName:           core.StringPtr(clusterId),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
	}
	if installed != nil {
		// idempotent: already enabled, and no other version requested
		if version == "" || version == current.Version {
			LoggingInfo(hiscallInfo, start)
			return current, nil
		}
		// already enabled with a different version: update the add-on version
		options.Addons = []kubernetesserviceapiv1.ClusterAddon{{Name: core.StringPtr(spec.addonName), Version: core.StringPtr(version)}}
		options.Update = core.BoolPtr(true)
	} else {
		if version == "" {
			// same selection as installAutoScalerAddon(); fall back to the IKS default (version omitted)
			availableAddons, _, getAvailErr := ic.ClusterService.GetAddonsWithContext(ic.Ctx, &kubernetesserviceapiv1.GetAddonsOptions{})
			if getAvailErr != nil {
				cblogger.Warn(fmt.Sprintf("failed to get available addons, the IKS default version of %s will be used: %v", spec.addonName, getAvailErr))
			} else {
				version = pickIbmDefaultAddonVersion(availableAddons, spec.addonName, kubeVersion)
			}
		}
		addon := kubernetesserviceapiv1.ClusterAddon{Name: core.StringPtr(spec.addonName)}
		if version != "" {
			addon.Version = core.StringPtr(version)
		}
		options.Addons = []kubernetesserviceapiv1.ClusterAddon{addon}
		options.Enable = core.BoolPtr(true)
	}

	_, _, err = ic.ClusterService.ManageClusterAddonsWithContext(ic.Ctx, options)
	if err != nil {
		err = fmt.Errorf("Failed to Enable Storage Driver: failed to manage cluster addon %s: %w", spec.addonName, err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.StorageDriverInfo{}, err
	}

	// IKS applies add-on changes asynchronously
	result := current
	result.Status = irs.StorageDriverUpdating
	result.Version = version
	LoggingInfo(hiscallInfo, start)
	return result, nil
}

func (ic *IbmClusterHandler) DisableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind) (bool, error) {
	hiscallInfo := GetCallLogScheme(ic.Region, call.CLUSTER, clusterIID.NameId, "DisableStorageDriver()")
	start := call.Start()

	if clusterIID.NameId == "" && clusterIID.SystemId == "" {
		return false, errors.New("Failed to Disable Storage Driver. err = invalid IID")
	}
	spec, err := ibmStorageDriverSpecOf(kind)
	if err != nil {
		err = fmt.Errorf("Failed to Disable Storage Driver: %w", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	clusterId, resourceGroupId, _, addons, err := ic.getStorageDriverContext(clusterIID)
	if err != nil {
		err = fmt.Errorf("Failed to Disable Storage Driver: %w", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	// idempotent: already disabled
	if findIbmClusterAddon(addons, spec.addonName) == nil {
		LoggingInfo(hiscallInfo, start)
		return true, nil
	}

	// Disabling the add-on removes the driver components only; VPC volumes are kept.
	_, _, err = ic.ClusterService.ManageClusterAddonsWithContext(ic.Ctx, &kubernetesserviceapiv1.ManageClusterAddonsOptions{
		IdOrName:           core.StringPtr(clusterId),
		Addons:             []kubernetesserviceapiv1.ClusterAddon{{Name: core.StringPtr(spec.addonName)}},
		Enable:             core.BoolPtr(false),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
	})
	if err != nil {
		err = fmt.Errorf("Failed to Disable Storage Driver: failed to manage cluster addon %s: %w", spec.addonName, err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}
