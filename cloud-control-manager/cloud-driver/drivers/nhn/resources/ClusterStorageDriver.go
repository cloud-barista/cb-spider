// NHN Cloud Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// CSI storage driver management of a Kubernetes Cluster.
//
// NKS offers its CSI drivers as optional cluster add-ons (cinder-csi-plugin, nfs-csi-plugin).
// The nhncloud-sdk-go containerinfra package has no add-on API, so the add-on endpoints are
// called with the cluster ServiceClient, which adds the auth token and the
// "OpenStack-API-Version: container-infra latest" header.
//
// API reference: https://docs.nhncloud.com/en/Container/NKS/en/public-api/
//   - GET    /v1/clusters/{CLUSTER_ID_OR_NAME}/addons
//   - POST   /v1/clusters/{CLUSTER_ID_OR_NAME}/addons
//   - PATCH  /v1/clusters/{CLUSTER_ID_OR_NAME}/addons/{ADDON_UUID_OR_NAME}
//   - DELETE /v1/clusters/{CLUSTER_ID_OR_NAME}/addons/{ADDON_UUID_OR_NAME}
//   - GET    /v1/addons?k8s_version=...
//
// by CB-Spider Team, 2026.10.

package resources

import (
	"fmt"
	"net/url"
	"runtime/debug"
	"sort"
	"strings"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"golang.org/x/mod/semver"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	nhnAddonResolveConflictsOverwrite = "overwrite"
)

// nhnStorageDriverSpec describes the NKS add-on that provides a StorageDriverKind.
type nhnStorageDriverSpec struct {
	Kind               irs.StorageDriverKind
	AddonName          string
	CSIDriverName      string
	RequiresFileSystem bool
}

var nhnStorageDriverSpecs = []nhnStorageDriverSpec{
	{Kind: irs.StorageDriverBlock, AddonName: "cinder-csi-plugin", CSIDriverName: "cinder.csi.openstack.org"},
	// NHN's nfs.csi.k8s.io creates an NHN NAS volume per PVC, so no file system is needed in advance
	// (per NHN NKS user guide "NHN Cloud NAS Service Integration").
	{Kind: irs.StorageDriverFile, AddonName: "nfs-csi-plugin", CSIDriverName: "nfs.csi.k8s.io", RequiresFileSystem: false},
}

// nhnClusterAddon is an add-on installed on a cluster (GET /v1/clusters/{id}/addons).
type nhnClusterAddon struct {
	UUID         string                 `json:"uuid"`
	ClusterUUID  string                 `json:"cluster_uuid"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Options      map[string]interface{} `json:"options"`
	Status       string                 `json:"status"`
	StatusReason string                 `json:"status_reason"`
	Scope        string                 `json:"scope"`
}

type nhnClusterAddonList struct {
	Addons []nhnClusterAddon `json:"addons"`
}

// nhnAddon is an add-on offered by NHN Cloud (GET /v1/addons).
type nhnAddon struct {
	UUID          string `json:"uuid"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	K8sMinVersion string `json:"k8s_min_version"`
	K8sMaxVersion string `json:"k8s_max_version"`
}

type nhnAddonList struct {
	Addons []nhnAddon `json:"addons"`
}

type nhnAddonInstallReq struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	ResolveConflicts string `json:"resolve_conflicts"`
}

type nhnAddonUpdateReq struct {
	Version          string `json:"version"`
	ResolveConflicts string `json:"resolve_conflicts"`
}

func nhnStorageDriverSpecOf(kind irs.StorageDriverKind) (nhnStorageDriverSpec, error) {
	for _, spec := range nhnStorageDriverSpecs {
		if spec.Kind == kind {
			return spec, nil
		}
	}
	return nhnStorageDriverSpec{}, fmt.Errorf("unsupported storage driver kind: %q", kind)
}

func (nch *NhnCloudClusterHandler) ListStorageDriver(clusterIID irs.IID) ([]irs.StorageDriverInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NHN Cloud Driver: called ListStorageDriver()")
	hiscallInfo := getCallLogScheme(nch.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "ListStorageDriver()")
	start := call.Start()

	cluster, err := nhnGetRawCluster(nch.ClusterClient, clusterIID)
	if err != nil {
		err = fmt.Errorf("failed to list storage drivers: failed to get cluster: %w", err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	addons, err := nhnGetClusterAddonList(nch.ClusterClient, cluster.UUID)
	if err != nil {
		err = fmt.Errorf("failed to list storage drivers: %w", err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	infoList := make([]irs.StorageDriverInfo, 0, len(nhnStorageDriverSpecs))
	for _, spec := range nhnStorageDriverSpecs {
		infoList = append(infoList, nhnBuildStorageDriverInfo(spec, nhnFindClusterAddon(addons, spec.AddonName)))
	}

	LoggingInfo(hiscallInfo, start)
	return infoList, nil
}

func (nch *NhnCloudClusterHandler) EnableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind, version string) (irs.StorageDriverInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NHN Cloud Driver: called EnableStorageDriver()")
	hiscallInfo := getCallLogScheme(nch.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "EnableStorageDriver()")
	start := call.Start()

	var enableErr error
	defer func() {
		if enableErr != nil {
			LoggingError(hiscallInfo, enableErr)
		}
	}()

	spec, err := nhnStorageDriverSpecOf(kind)
	if err != nil {
		enableErr = err
		return irs.StorageDriverInfo{}, enableErr
	}

	cluster, err := nhnGetRawCluster(nch.ClusterClient, clusterIID)
	if err != nil {
		enableErr = fmt.Errorf("failed to enable storage driver(%s): failed to get cluster: %w", kind, err)
		return irs.StorageDriverInfo{}, enableErr
	}

	addons, err := nhnGetClusterAddonList(nch.ClusterClient, cluster.UUID)
	if err != nil {
		enableErr = fmt.Errorf("failed to enable storage driver(%s): %w", kind, err)
		return irs.StorageDriverInfo{}, enableErr
	}

	addon := nhnFindClusterAddon(addons, spec.AddonName)
	if addon != nil {
		current := nhnBuildStorageDriverInfo(spec, addon)
		if (version == "" || version == addon.Version) && current.Status != irs.StorageDriverFailed {
			LoggingInfo(hiscallInfo, start)
			return current, nil
		}

		// Installed with another version, or failed: update in place.
		reqBody := nhnAddonUpdateReq{Version: version, ResolveConflicts: nhnAddonResolveConflictsOverwrite}
		if reqBody.Version == "" {
			reqBody.Version = addon.Version
		}
		_, err := nch.ClusterClient.Patch(nhnClusterAddonURL(nch.ClusterClient, cluster.UUID, addon.Name), reqBody, nil,
			&nhnsdk.RequestOpts{OkCodes: []int{200, 201, 202, 204}})
		if err != nil {
			enableErr = fmt.Errorf("failed to update NKS add-on %s: %w", spec.AddonName, err)
			return irs.StorageDriverInfo{}, enableErr
		}

		updated := *addon
		updated.Version = reqBody.Version
		info := nhnBuildStorageDriverInfo(spec, &updated)
		info.Status = irs.StorageDriverUpdating
		LoggingInfo(hiscallInfo, start)
		return info, nil
	}

	// A NOT_INSTALLED entry of the cluster add-on list may already carry the installable version.
	if version == "" {
		for _, a := range addons {
			if nhnSameAddonName(a.Name, spec.AddonName) && a.Version != "" {
				version = a.Version
				break
			}
		}
	}

	if version == "" {
		k8sVersion, err := getClusterVersion(cluster)
		if err != nil {
			enableErr = fmt.Errorf("failed to enable storage driver(%s): %w", kind, err)
			return irs.StorageDriverInfo{}, enableErr
		}
		version, err = nhnGetDefaultAddonVersion(nch.ClusterClient, spec.AddonName, k8sVersion)
		if err != nil {
			enableErr = fmt.Errorf("failed to enable storage driver(%s): %w", kind, err)
			return irs.StorageDriverInfo{}, enableErr
		}
	}

	// The cluster add-on list spells names with hyphens (cinder-csi-plugin) but the install API
	// takes the add-on catalog name with underscores (cinder_csi_plugin); a hyphenated name fails
	// with "Addon config ... is not found".
	reqBody := nhnAddonInstallReq{Name: nhnCatalogAddonName(spec.AddonName), Version: version, ResolveConflicts: nhnAddonResolveConflictsOverwrite}
	_, err = nch.ClusterClient.Post(nch.ClusterClient.ServiceURL("clusters", cluster.UUID, "addons"), reqBody, nil,
		&nhnsdk.RequestOpts{OkCodes: []int{200, 201, 202}})
	if err != nil {
		enableErr = fmt.Errorf("failed to install NKS add-on %s: %w", spec.AddonName, err)
		return irs.StorageDriverInfo{}, enableErr
	}

	info := nhnBuildStorageDriverInfo(spec, &nhnClusterAddon{Name: spec.AddonName, Version: version})
	info.Status = irs.StorageDriverUpdating

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (nch *NhnCloudClusterHandler) DisableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind) (bool, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NHN Cloud Driver: called DisableStorageDriver()")
	hiscallInfo := getCallLogScheme(nch.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "DisableStorageDriver()")
	start := call.Start()

	var disableErr error
	defer func() {
		if disableErr != nil {
			LoggingError(hiscallInfo, disableErr)
		}
	}()

	spec, err := nhnStorageDriverSpecOf(kind)
	if err != nil {
		disableErr = err
		return false, disableErr
	}

	cluster, err := nhnGetRawCluster(nch.ClusterClient, clusterIID)
	if err != nil {
		disableErr = fmt.Errorf("failed to disable storage driver(%s): failed to get cluster: %w", kind, err)
		return false, disableErr
	}

	addons, err := nhnGetClusterAddonList(nch.ClusterClient, cluster.UUID)
	if err != nil {
		disableErr = fmt.Errorf("failed to disable storage driver(%s): %w", kind, err)
		return false, disableErr
	}

	addon := nhnFindClusterAddon(addons, spec.AddonName)
	if addon == nil {
		// Already disabled.
		LoggingInfo(hiscallInfo, start)
		return true, nil
	}

	// Address the add-on by the name the cluster add-on list reports.
	_, err = nch.ClusterClient.Delete(nhnClusterAddonURL(nch.ClusterClient, cluster.UUID, addon.Name),
		&nhnsdk.RequestOpts{OkCodes: []int{200, 202, 204}})
	if err != nil {
		disableErr = fmt.Errorf("failed to delete NKS add-on %s: %w", spec.AddonName, err)
		return false, disableErr
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func nhnClusterAddonURL(scCluster *nhnsdk.ServiceClient, clusterId, addonName string) string {
	return scCluster.ServiceURL("clusters", clusterId, "addons", url.PathEscape(addonName))
}

func nhnGetClusterAddonList(scCluster *nhnsdk.ServiceClient, clusterId string) ([]nhnClusterAddon, error) {
	var addonList nhnClusterAddonList
	_, err := scCluster.Get(scCluster.ServiceURL("clusters", clusterId, "addons"), &addonList,
		&nhnsdk.RequestOpts{OkCodes: []int{200}})
	if err != nil {
		return nil, fmt.Errorf("failed to list add-ons of cluster(id=%s): %w", clusterId, err)
	}
	return addonList.Addons, nil
}

// nhnGetDefaultAddonVersion returns the latest version of the add-on offered for the K8s version.
func nhnGetDefaultAddonVersion(scCluster *nhnsdk.ServiceClient, addonName, k8sVersion string) (string, error) {
	// The accepted k8s_version format is not documented, so try "v1.x.y", "1.x.y" and no filter.
	var filters []string
	if k8sVersion != "" {
		filters = append(filters, "v"+strings.TrimPrefix(k8sVersion, "v"), strings.TrimPrefix(k8sVersion, "v"))
	}
	filters = append(filters, "")

	seen := map[string]bool{}
	for _, filter := range filters {
		var addonList nhnAddonList
		reqURL := scCluster.ServiceURL("addons")
		if filter != "" {
			reqURL += "?" + url.Values{"k8s_version": {filter}}.Encode()
		}
		_, err := scCluster.Get(reqURL, &addonList, &nhnsdk.RequestOpts{OkCodes: []int{200}})
		if err != nil {
			return "", fmt.Errorf("failed to list NKS add-ons for K8s %s: %w", k8sVersion, err)
		}
		for _, a := range addonList.Addons {
			seen[a.Name] = true
		}
		if latest := nhnLatestAddonVersion(addonList.Addons, addonName); latest != "" {
			return latest, nil
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return "", fmt.Errorf("no installable version of NKS add-on %s for K8s %s (catalog add-ons: %s)", addonName, k8sVersion, strings.Join(names, ", "))
}

// nhnCatalogAddonName returns the add-on catalog spelling of an add-on name (underscores).
func nhnCatalogAddonName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

// nhnSameAddonName compares NKS add-on names, treating "-" and "_" as the same character.
func nhnSameAddonName(a, b string) bool {
	return strings.ReplaceAll(a, "_", "-") == strings.ReplaceAll(b, "_", "-")
}

// NKS lists add-ons that are not installed with status NOT_INSTALLED; they are treated as
// absent so Enable installs them with POST.
func nhnFindClusterAddon(addons []nhnClusterAddon, addonName string) *nhnClusterAddon {
	for i := range addons {
		if nhnSameAddonName(addons[i].Name, addonName) && !strings.EqualFold(strings.TrimSpace(addons[i].Status), "NOT_INSTALLED") {
			return &addons[i]
		}
	}
	return nil
}

// nhnBuildStorageDriverInfo maps an NKS cluster add-on (nil when not installed) to StorageDriverInfo.
func nhnBuildStorageDriverInfo(spec nhnStorageDriverSpec, addon *nhnClusterAddon) irs.StorageDriverInfo {
	info := irs.StorageDriverInfo{
		Kind:               spec.Kind,
		CSPDriverName:      spec.AddonName,
		CSIDriverName:      spec.CSIDriverName,
		ManagementMode:     irs.StorageDriverAddon,
		Status:             irs.StorageDriverDisabled,
		RequiresFileSystem: spec.RequiresFileSystem,
	}
	if addon == nil {
		return info
	}

	info.Status = nhnConvertAddonStatus(addon.Status)
	info.Version = addon.Version

	keyValues := []irs.KeyValue{}
	if addon.UUID != "" {
		keyValues = append(keyValues, irs.KeyValue{Key: "AddonUUID", Value: addon.UUID})
	}
	if addon.Status != "" {
		keyValues = append(keyValues, irs.KeyValue{Key: "AddonStatus", Value: addon.Status})
	}
	if addon.StatusReason != "" {
		keyValues = append(keyValues, irs.KeyValue{Key: "StatusReason", Value: addon.StatusReason})
	}
	if len(keyValues) > 0 {
		info.KeyValueList = keyValues
	}
	return info
}

// nhnConvertAddonStatus maps an NKS add-on status (e.g., CREATE_COMPLETE, UPDATE_IN_PROGRESS,
// UPDATE_FAILED) to StorageDriverStatus.
func nhnConvertAddonStatus(status string) irs.StorageDriverStatus {
	s := strings.ToUpper(strings.TrimSpace(status))
	switch {
	case s == "":
		return irs.StorageDriverUnknown
	case strings.HasPrefix(s, "DELETE_COMPLETE"):
		return irs.StorageDriverDisabled
	case strings.HasSuffix(s, "_FAILED"), strings.Contains(s, "FAIL"):
		return irs.StorageDriverFailed
	case strings.HasSuffix(s, "_IN_PROGRESS"):
		return irs.StorageDriverUpdating
	case strings.HasSuffix(s, "_COMPLETE"):
		return irs.StorageDriverEnabled
	default:
		return irs.StorageDriverUnknown
	}
}

// nhnLatestAddonVersion returns the highest semantic version of the named add-on in the catalog,
// or the first catalog version if none parse as a semantic version.
func nhnLatestAddonVersion(addons []nhnAddon, addonName string) string {
	first, latest, latestCanon := "", "", ""
	for _, addon := range addons {
		if !nhnSameAddonName(addon.Name, addonName) || addon.Version == "" {
			continue
		}
		if first == "" {
			first = addon.Version
		}
		canon := addon.Version
		if !strings.HasPrefix(canon, "v") {
			canon = "v" + canon
		}
		if !semver.IsValid(canon) {
			continue
		}
		if latestCanon == "" || semver.Compare(canon, latestCanon) > 0 {
			latest, latestCanon = addon.Version, canon
		}
	}
	if latest == "" {
		return first
	}
	return latest
}
