// Alibaba Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// CSI storage driver management of a Kubernetes Cluster.
//
// On ACK, the disk (Block) CSI driver is served by two cluster add-ons, csi-plugin and
// csi-provisioner, which are installed by default on ACK managed clusters. ACK Pro clusters
// may run the control-plane-hosted variant of csi-provisioner, named managed-csiprovisioner;
// it is accepted as csi-provisioner.
//   - Block: CSIDriver diskplugin.csi.alibabacloud.com
// The same add-ons also serve the NAS and OSS CSI drivers, so the Block driver cannot be
// disabled without breaking them.
//
// by CB-Spider Team, 2026.10.

package resources

import (
	"fmt"
	"strings"

	cs2015 "github.com/alibabacloud-go/cs-20151215/v4/client"
	"github.com/alibabacloud-go/tea/tea"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	aliCSIPluginAddon             = "csi-plugin"
	aliCSIProvisionerAddon        = "csi-provisioner"
	aliManagedCSIProvisionerAddon = "managed-csiprovisioner"

	aliDiskCSIDriverName = "diskplugin.csi.alibabacloud.com"
)

// aliAddonState is the normalized state of one installed add-on.
type aliAddonState int

const (
	aliAddonMissing aliAddonState = iota
	aliAddonActive
	aliAddonInProgress
	aliAddonFailed
	aliAddonUnknown
)

type aliAddon struct {
	Name    string
	State   string
	Version string
}

func (ach *AlibabaClusterHandler) ListStorageDriver(clusterIID irs.IID) ([]irs.StorageDriverInfo, error) {
	cblogger.Debug("Alibaba Cloud Driver: called ListStorageDriver()")
	hiscallInfo := GetCallLogScheme(ach.RegionInfo, call.CLUSTER, clusterIID.NameId, "ListStorageDriver()")
	start := call.Start()

	addons, err := aliListClusterAddons(ach.CsClient, clusterIID.SystemId)
	if err != nil {
		err := fmt.Errorf("failed to list storage driver: %w", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	return []irs.StorageDriverInfo{
		aliStorageDriverInfo(addons, irs.StorageDriverBlock),
	}, nil
}

// EnableStorageDriver installs whichever of csi-plugin/csi-provisioner is missing.
// It is idempotent: when both are installed (or being installed) no CSP call is made.
// A non-empty version is applied to every add-on being installed.
func (ach *AlibabaClusterHandler) EnableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind, version string) (irs.StorageDriverInfo, error) {
	cblogger.Debug("Alibaba Cloud Driver: called EnableStorageDriver()")
	hiscallInfo := GetCallLogScheme(ach.RegionInfo, call.CLUSTER, clusterIID.NameId, "EnableStorageDriver()")
	start := call.Start()

	info, err := ach.enableStorageDriver(clusterIID, kind, version)
	if err != nil {
		err := fmt.Errorf("failed to enable storage driver: %w", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.StorageDriverInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (ach *AlibabaClusterHandler) enableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind, version string) (irs.StorageDriverInfo, error) {
	if !isValidAliStorageDriverKind(kind) {
		return irs.StorageDriverInfo{}, fmt.Errorf("storage driver kind(%s) is not supported; only Block is supported", kind)
	}

	clusterId := clusterIID.SystemId
	addons, err := aliListClusterAddons(ach.CsClient, clusterId)
	if err != nil {
		return irs.StorageDriverInfo{}, err
	}

	info := aliStorageDriverInfo(addons, kind)
	switch info.Status {
	case irs.StorageDriverEnabled, irs.StorageDriverUpdating:
		return info, nil
	}

	missing := aliMissingCSIAddons(addons)
	if len(missing) == 0 {
		return irs.StorageDriverInfo{}, fmt.Errorf("csi add-ons are installed but not active (status %s); check them in the ACK console: %s",
			info.Status, aliKeyValueString(info.KeyValueList))
	}

	if err := aliInstallClusterAddons(ach.CsClient, clusterId, missing, version); err != nil {
		return irs.StorageDriverInfo{}, err
	}
	cblogger.Infof("Installing add-ons %v in Cluster(%s)", missing, clusterIID.NameId)

	info.Status = irs.StorageDriverUpdating
	if version != "" {
		info.Version = version
	}
	return info, nil
}

// DisableStorageDriver always fails on ACK: the Block (disk) driver is served by the
// csi-plugin/csi-provisioner add-ons that the NAS and OSS CSI drivers also use, so
// uninstalling them would break those drivers too.
func (ach *AlibabaClusterHandler) DisableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind) (bool, error) {
	cblogger.Debug("Alibaba Cloud Driver: called DisableStorageDriver()")
	hiscallInfo := GetCallLogScheme(ach.RegionInfo, call.CLUSTER, clusterIID.NameId, "DisableStorageDriver()")

	err := fmt.Errorf("failed to disable storage driver: %w", aliDisableStorageDriverError(kind))
	cblogger.Error(err)
	LoggingError(hiscallInfo, err)
	return false, err
}

// aliDisableStorageDriverError is the error DisableStorageDriver returns for every kind.
func aliDisableStorageDriverError(kind irs.StorageDriverKind) error {
	if !isValidAliStorageDriverKind(kind) {
		return fmt.Errorf("storage driver kind(%s) is not supported; only Block is supported", kind)
	}
	return fmt.Errorf("on Alibaba ACK the %s storage driver cannot be disabled: its %s and %s add-ons are shared with the NAS and OSS CSI drivers",
		kind, aliCSIPluginAddon, aliCSIProvisionerAddon)
}

func isValidAliStorageDriverKind(kind irs.StorageDriverKind) bool {
	return kind == irs.StorageDriverBlock
}

// aliListClusterAddons returns the installed add-ons keyed by name.
func aliListClusterAddons(csClient *cs2015.Client, clusterId string) (map[string]aliAddon, error) {
	if clusterId == "" {
		return nil, fmt.Errorf("cluster SystemId is empty")
	}
	resp, err := csClient.ListClusterAddonInstances(tea.String(clusterId))
	if err != nil {
		return nil, err
	}
	addons := map[string]aliAddon{}
	if resp == nil || resp.Body == nil {
		return addons, nil
	}
	for _, a := range resp.Body.Addons {
		if a == nil || a.Name == nil {
			continue
		}
		name := tea.StringValue(a.Name)
		addons[name] = aliAddon{Name: name, State: tea.StringValue(a.State), Version: tea.StringValue(a.Version)}
	}
	return addons, nil
}

func aliInstallClusterAddons(csClient *cs2015.Client, clusterId string, names []string, version string) error {
	body := make([]*cs2015.InstallClusterAddonsRequestBody, 0, len(names))
	for _, name := range names {
		b := &cs2015.InstallClusterAddonsRequestBody{Name: tea.String(name)}
		if version != "" {
			b.Version = tea.String(version)
		}
		body = append(body, b)
	}
	_, err := csClient.InstallClusterAddons(tea.String(clusterId), &cs2015.InstallClusterAddonsRequest{Body: body})
	return err
}

// aliAddonStateOf normalizes the add-on state string of ListClusterAddonInstances.
func aliAddonStateOf(addon aliAddon, found bool) aliAddonState {
	if !found {
		return aliAddonMissing
	}
	s := strings.ToLower(strings.TrimSpace(addon.State))
	switch {
	case s == "active" || s == "running" || s == "installed" || s == "success" || s == "succeeded":
		return aliAddonActive
	case strings.Contains(s, "fail") || s == "error":
		return aliAddonFailed
	case strings.Contains(s, "install") || strings.Contains(s, "updat") || strings.Contains(s, "upgrad") ||
		strings.Contains(s, "delet") || strings.Contains(s, "pending") || strings.Contains(s, "rollback"):
		return aliAddonInProgress
	default:
		return aliAddonUnknown
	}
}

// aliCSIAddons returns csi-plugin and csi-provisioner (or managed-csiprovisioner).
func aliCSIAddons(addons map[string]aliAddon) (plugin aliAddon, pluginFound bool, provisioner aliAddon, provisionerFound bool) {
	plugin, pluginFound = addons[aliCSIPluginAddon]
	provisioner, provisionerFound = addons[aliCSIProvisionerAddon]
	if !provisionerFound {
		provisioner, provisionerFound = addons[aliManagedCSIProvisionerAddon]
	}
	return
}

// aliMissingCSIAddons returns the add-on names to install, in install order.
func aliMissingCSIAddons(addons map[string]aliAddon) []string {
	_, pluginFound, _, provisionerFound := aliCSIAddons(addons)
	missing := []string{}
	if !pluginFound {
		missing = append(missing, aliCSIPluginAddon)
	}
	if !provisionerFound {
		missing = append(missing, aliCSIProvisionerAddon)
	}
	return missing
}

// aliCombinedStatus combines the two add-on states into one driver status.
// Failed > Updating > Disabled (any missing) > Enabled (both active) > Unknown.
func aliCombinedStatus(plugin, provisioner aliAddonState) irs.StorageDriverStatus {
	states := []aliAddonState{plugin, provisioner}
	has := func(target aliAddonState) bool {
		for _, s := range states {
			if s == target {
				return true
			}
		}
		return false
	}
	switch {
	case has(aliAddonFailed):
		return irs.StorageDriverFailed
	case has(aliAddonInProgress):
		return irs.StorageDriverUpdating
	case has(aliAddonMissing):
		return irs.StorageDriverDisabled
	case plugin == aliAddonActive && provisioner == aliAddonActive:
		return irs.StorageDriverEnabled
	default:
		return irs.StorageDriverUnknown
	}
}

func aliStorageDriverInfo(addons map[string]aliAddon, kind irs.StorageDriverKind) irs.StorageDriverInfo {
	plugin, pluginFound, provisioner, provisionerFound := aliCSIAddons(addons)

	provisionerName := aliCSIProvisionerAddon
	if provisionerFound {
		provisionerName = provisioner.Name
	}

	info := irs.StorageDriverInfo{
		Kind:           kind,
		CSPDriverName:  aliCSIPluginAddon + "," + provisionerName,
		ManagementMode: irs.StorageDriverAddon,
		Status:         aliCombinedStatus(aliAddonStateOf(plugin, pluginFound), aliAddonStateOf(provisioner, provisionerFound)),
		Version:        plugin.Version,
		KeyValueList: []irs.KeyValue{
			{Key: aliCSIPluginAddon + ".state", Value: aliAddonStateText(plugin, pluginFound)},
			{Key: aliCSIPluginAddon + ".version", Value: plugin.Version},
			{Key: provisionerName + ".state", Value: aliAddonStateText(provisioner, provisionerFound)},
			{Key: provisionerName + ".version", Value: provisioner.Version},
		},
	}
	if kind == irs.StorageDriverBlock {
		info.CSIDriverName = aliDiskCSIDriverName
	}
	return info
}

func aliAddonStateText(addon aliAddon, found bool) string {
	if !found {
		return "not installed"
	}
	return addon.State
}

func aliKeyValueString(kvs []irs.KeyValue) string {
	parts := make([]string, 0, len(kvs))
	for _, kv := range kvs {
		parts = append(parts, kv.Key+"="+kv.Value)
	}
	return strings.Join(parts, ", ")
}
