// GCP Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// CSI storage driver management of a Kubernetes Cluster.
//
// GKE manages the Block CSI driver through the cluster's addonsConfig:
//   - Block: gcePersistentDiskCsiDriverConfig (pd.csi.storage.gke.io)
//
// Changes are applied with projects.locations.clusters.update (desiredAddonsConfig),
// which runs asynchronously. GKE manages the driver versions, so Version is always empty.
// On Autopilot clusters the driver is always enabled and cannot be changed.
//
// by CB-Spider Team, 2026.10.

package resources

import (
	"fmt"
	"strings"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	container "google.golang.org/api/container/v1"
)

const gkeClusterStatusRunning = "RUNNING"

// gkeCSIDriverSpec describes a CSI driver that GKE manages through addonsConfig.
type gkeCSIDriverSpec struct {
	kind          irs.StorageDriverKind
	cspDriverName string // addonsConfig field name
	csiDriverName string // name of the Kubernetes CSIDriver object

	// isEnabled reports whether the driver is enabled in the addons config.
	isEnabled func(addons *container.AddonsConfig) bool
	// desiredAddonsConfig returns an addons config that sets only this driver.
	desiredAddonsConfig func(enabled bool) *container.AddonsConfig
}

var gkePDCSIDriverSpec = gkeCSIDriverSpec{
	kind:          irs.StorageDriverBlock,
	cspDriverName: "gcePersistentDiskCsiDriverConfig",
	csiDriverName: "pd.csi.storage.gke.io",
	isEnabled: func(addons *container.AddonsConfig) bool {
		return addons != nil && addons.GcePersistentDiskCsiDriverConfig != nil && addons.GcePersistentDiskCsiDriverConfig.Enabled
	},
	desiredAddonsConfig: func(enabled bool) *container.AddonsConfig {
		return &container.AddonsConfig{
			GcePersistentDiskCsiDriverConfig: &container.GcePersistentDiskCsiDriverConfig{
				Enabled: enabled,
				// Without ForceSendFields, Enabled=false is omitted and the request changes nothing.
				ForceSendFields: []string{"Enabled"},
			},
		}
	},
}

// gkeCSIDriverSpecs lists the drivers ListStorageDriver returns. Only Block is supported.
var gkeCSIDriverSpecs = []gkeCSIDriverSpec{gkePDCSIDriverSpec}

func getGKECSIDriverSpec(kind irs.StorageDriverKind) (gkeCSIDriverSpec, error) {
	for _, spec := range gkeCSIDriverSpecs {
		if strings.EqualFold(string(spec.kind), string(kind)) {
			return spec, nil
		}
	}
	return gkeCSIDriverSpec{}, fmt.Errorf("storage driver kind(%s) is not supported; only Block is supported", kind)
}

func isGKEAutopilot(cluster *container.Cluster) bool {
	return cluster != nil && cluster.Autopilot != nil && cluster.Autopilot.Enabled
}

// convertGKEClusterToStorageDriverInfo reads the driver state from the cluster.
func convertGKEClusterToStorageDriverInfo(spec gkeCSIDriverSpec, cluster *container.Cluster) irs.StorageDriverInfo {
	info := irs.StorageDriverInfo{
		Kind:           spec.kind,
		CSPDriverName:  spec.cspDriverName,
		CSIDriverName:  spec.csiDriverName,
		ManagementMode: irs.StorageDriverAddon,
		Status:         irs.StorageDriverDisabled,
		// GKE manages the driver version.
		Version: "",
	}

	if isGKEAutopilot(cluster) {
		info.ManagementMode = irs.StorageDriverBuiltIn
		info.Status = irs.StorageDriverEnabled
		info.KeyValueList = []irs.KeyValue{{Key: "Autopilot", Value: "true"}}
		return info
	}

	if cluster != nil && spec.isEnabled(cluster.AddonsConfig) {
		info.Status = irs.StorageDriverEnabled
	}
	return info
}

func (ClusterHandler *GCPClusterHandler) getGKECluster(clusterIID irs.IID, apiName string) (*container.Cluster, error) {
	projectID := ClusterHandler.Credential.ProjectID
	zone := ClusterHandler.Region.Zone
	parent := getParentClusterAtContainer(projectID, zone, clusterIID.NameId)

	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterIID.NameId, apiName)
	start := call.Start()
	cluster, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.Get(parent).Do()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, fmt.Errorf("failed to get cluster %s: %w", clusterIID.NameId, err)
	}
	LoggingInfo(hiscallInfo, start)
	return cluster, nil
}

// updateGKEAddonsConfig starts a cluster update that sets the addons config of one driver.
func (ClusterHandler *GCPClusterHandler) updateGKEAddonsConfig(clusterIID irs.IID, spec gkeCSIDriverSpec, enabled bool, apiName string) (*container.Operation, error) {
	projectID := ClusterHandler.Credential.ProjectID
	zone := ClusterHandler.Region.Zone
	parent := getParentClusterAtContainer(projectID, zone, clusterIID.NameId)

	rb := &container.UpdateClusterRequest{
		Update: &container.ClusterUpdate{
			DesiredAddonsConfig: spec.desiredAddonsConfig(enabled),
		},
	}

	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterIID.NameId, apiName)
	start := call.Start()
	op, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.Update(parent, rb).Do()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, fmt.Errorf("failed to update %s of cluster %s: %w", spec.cspDriverName, clusterIID.NameId, err)
	}
	LoggingInfo(hiscallInfo, start)
	return op, nil
}

func checkGKEClusterRunning(cluster *container.Cluster) error {
	if cluster.Status != gkeClusterStatusRunning {
		return fmt.Errorf("cluster %s is %s (another operation may be in progress); storage drivers can be changed only when the cluster is %s",
			cluster.Name, cluster.Status, gkeClusterStatusRunning)
	}
	return nil
}

func (ClusterHandler *GCPClusterHandler) ListStorageDriver(clusterIID irs.IID) ([]irs.StorageDriverInfo, error) {
	cblogger.Info("GCP Cloud Driver: called ListStorageDriver()")

	cluster, err := ClusterHandler.getGKECluster(clusterIID, "ListStorageDriver()")
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	driverList := make([]irs.StorageDriverInfo, 0, len(gkeCSIDriverSpecs))
	for _, spec := range gkeCSIDriverSpecs {
		driverList = append(driverList, convertGKEClusterToStorageDriverInfo(spec, cluster))
	}
	return driverList, nil
}

func (ClusterHandler *GCPClusterHandler) EnableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind, version string) (irs.StorageDriverInfo, error) {
	cblogger.Info("GCP Cloud Driver: called EnableStorageDriver()")

	spec, err := getGKECSIDriverSpec(kind)
	if err != nil {
		return irs.StorageDriverInfo{}, err
	}
	if version != "" {
		cblogger.Warnf("GKE manages the %s version; the requested version %s is ignored", spec.cspDriverName, version)
	}

	cluster, err := ClusterHandler.getGKECluster(clusterIID, "EnableStorageDriver()")
	if err != nil {
		cblogger.Error(err)
		return irs.StorageDriverInfo{}, err
	}

	info := convertGKEClusterToStorageDriverInfo(spec, cluster)
	if info.ManagementMode == irs.StorageDriverBuiltIn || info.Status == irs.StorageDriverEnabled {
		return info, nil
	}

	if err := checkGKEClusterRunning(cluster); err != nil {
		cblogger.Error(err)
		return info, err
	}

	op, err := ClusterHandler.updateGKEAddonsConfig(clusterIID, spec, true, "EnableStorageDriver()")
	if err != nil {
		cblogger.Error(err)
		return info, err
	}

	// The update usually takes a few minutes. Check once in case it finished already.
	if op != nil && op.Status == "DONE" {
		updated, err := ClusterHandler.getGKECluster(clusterIID, "EnableStorageDriver()")
		if err == nil && spec.isEnabled(updated.AddonsConfig) {
			return convertGKEClusterToStorageDriverInfo(spec, updated), nil
		}
	}

	info.Status = irs.StorageDriverUpdating
	if op != nil {
		info.KeyValueList = append(info.KeyValueList, irs.KeyValue{Key: "OperationName", Value: op.Name})
	}
	return info, nil
}

func (ClusterHandler *GCPClusterHandler) DisableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind) (bool, error) {
	cblogger.Info("GCP Cloud Driver: called DisableStorageDriver()")

	spec, err := getGKECSIDriverSpec(kind)
	if err != nil {
		return false, err
	}

	cluster, err := ClusterHandler.getGKECluster(clusterIID, "DisableStorageDriver()")
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	if isGKEAutopilot(cluster) {
		err := fmt.Errorf("%s is built into the Autopilot cluster %s and cannot be disabled", spec.cspDriverName, clusterIID.NameId)
		cblogger.Error(err)
		return false, err
	}
	if !spec.isEnabled(cluster.AddonsConfig) {
		return true, nil
	}

	if err := checkGKEClusterRunning(cluster); err != nil {
		cblogger.Error(err)
		return false, err
	}

	// Disabling removes the driver only. Persistent disks are kept.
	if _, err := ClusterHandler.updateGKEAddonsConfig(clusterIID, spec, false, "DisableStorageDriver()"); err != nil {
		cblogger.Error(err)
		return false, err
	}
	return true, nil
}
