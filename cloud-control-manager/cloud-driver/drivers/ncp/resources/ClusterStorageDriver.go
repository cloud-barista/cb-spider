// NCP Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// CSI storage driver management of a Kubernetes Cluster.
//
// NKS manages its Block CSI driver as a cluster add-on (nks-csi) since K8s 1.36.
// Older clusters ship the same driver preinstalled and outside the add-on API, so it is
// reported as BuiltIn. The pinned ncloud-sdk-go-v2 (services/vnks) has no add-on methods,
// so the add-on endpoints are called directly with the SDK's base path and HMAC signer.
//
// API references:
//   - https://api.ncloud-docs.com/docs/en/nks-getclusteraddonlist
//   - https://api.ncloud-docs.com/docs/en/nks-installaddon
//   - https://api.ncloud-docs.com/docs/en/nks-updateaddon
//   - https://api.ncloud-docs.com/docs/en/nks-deleteaddon
//   - https://api.ncloud-docs.com/docs/en/nks-getaddonconfig
//
// by CB-Spider Team, 2026.10.

package resources

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/hmac"
	ncloud "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	vnks "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vnks"
	"golang.org/x/mod/semver"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	ncpAddonResolveConflictsOverwrite = "Overwrite"
	ncpAddonHTTPTimeout               = 60 * time.Second

	// NKS manages CSI drivers as add-ons from this K8s minor version on.
	ncpAddonManagedCSIMajor = 1
	ncpAddonManagedCSIMinor = 36
)

// ncpStorageDriverSpec describes the NKS add-on that provides a StorageDriverKind.
type ncpStorageDriverSpec struct {
	Kind          irs.StorageDriverKind
	AddonName     string
	CSIDriverName string
}

var ncpStorageDriverSpecs = []ncpStorageDriverSpec{
	{Kind: irs.StorageDriverBlock, AddonName: "nks-csi", CSIDriverName: "blk.csi.ncloud.com"},
}

// ncpClusterAddon is an add-on item of GET /clusters/{uuid}/addons.
type ncpClusterAddon struct {
	Uuid                string `json:"uuid"`
	ClusterUuid         string `json:"clusterUuid"`
	AddonName           string `json:"addonName"`
	Version             string `json:"version"`
	ConfigurationValues string `json:"configurationValues"`
	DisplayName         string `json:"displayName"`
	Category            string `json:"category"`
	Status              string `json:"status"`
	Message             string `json:"message"`
	K8sVersion          string `json:"k8sVersion"`
	LatestVersion       string `json:"latestVersion"`
}

type ncpClusterAddonList struct {
	Addons     []ncpClusterAddon `json:"addons"`
	TotalCount int               `json:"totalCount"`
}

type ncpAddonInstallReq struct {
	AddonName           string `json:"addonName"`
	Version             string `json:"version"`
	ConfigurationValues string `json:"configurationValues,omitempty"`
	ResolveConflicts    string `json:"resolveConflicts,omitempty"`
}

type ncpAddonUpdateReq struct {
	Version          string `json:"version,omitempty"`
	ResolveConflicts string `json:"resolveConflicts,omitempty"`
}

// ncpAddonConfig is the response of GET /addon-configs/{addonName}.
type ncpAddonConfig struct {
	Name     string   `json:"name"`
	Versions []string `json:"versions"`
}

func ncpStorageDriverSpecOf(kind irs.StorageDriverKind) (ncpStorageDriverSpec, error) {
	for _, spec := range ncpStorageDriverSpecs {
		if spec.Kind == kind {
			return spec, nil
		}
	}
	return ncpStorageDriverSpec{}, fmt.Errorf("storage driver kind(%s) is not supported; only Block is supported", kind)
}

func (nvch *NcpVpcClusterHandler) ListStorageDriver(clusterIID irs.IID) ([]irs.StorageDriverInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NCP Cloud Driver: called ListStorageDriver()")
	hiscallInfo := GetCallLogScheme(nvch.RegionInfo.Zone, call.CLUSTER, clusterIID.NameId, "ListStorageDriver()")
	start := call.Start()

	cluster, err := nvch.getStorageDriverCluster(clusterIID)
	if err != nil {
		err = fmt.Errorf("failed to list storage drivers: %w", err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	addons, err := nvch.getStorageDriverAddons(cluster)
	if err != nil {
		err = fmt.Errorf("failed to list storage drivers: %w", err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	addonManaged := ncpIsAddonManagedCSI(ncloud.StringValue(cluster.K8sVersion))
	infoList := make([]irs.StorageDriverInfo, 0, len(ncpStorageDriverSpecs))
	for _, spec := range ncpStorageDriverSpecs {
		infoList = append(infoList, ncpBuildStorageDriverInfo(spec, ncpFindAddon(addons, spec.AddonName), addonManaged))
	}

	LoggingInfo(hiscallInfo, start)
	return infoList, nil
}

func (nvch *NcpVpcClusterHandler) EnableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind, version string) (irs.StorageDriverInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NCP Cloud Driver: called EnableStorageDriver()")
	hiscallInfo := GetCallLogScheme(nvch.RegionInfo.Zone, call.CLUSTER, clusterIID.NameId, "EnableStorageDriver()")
	start := call.Start()

	var enableErr error
	defer func() {
		if enableErr != nil {
			LoggingError(hiscallInfo, enableErr)
		}
	}()

	spec, err := ncpStorageDriverSpecOf(kind)
	if err != nil {
		enableErr = err
		return irs.StorageDriverInfo{}, enableErr
	}

	cluster, err := nvch.getStorageDriverCluster(clusterIID)
	if err != nil {
		enableErr = fmt.Errorf("failed to enable storage driver(%s): %w", kind, err)
		return irs.StorageDriverInfo{}, enableErr
	}
	clusterUUID := ncloud.StringValue(cluster.Uuid)
	k8sVersion := ncloud.StringValue(cluster.K8sVersion)

	addons, err := nvch.getStorageDriverAddons(cluster)
	if err != nil {
		enableErr = fmt.Errorf("failed to enable storage driver(%s): %w", kind, err)
		return irs.StorageDriverInfo{}, enableErr
	}

	addon := ncpFindAddon(addons, spec.AddonName)
	current := ncpBuildStorageDriverInfo(spec, addon, ncpIsAddonManagedCSI(k8sVersion))

	// Preinstalled outside the add-on API: always enabled, nothing to call.
	if current.ManagementMode == irs.StorageDriverBuiltIn {
		LoggingInfo(hiscallInfo, start)
		return current, nil
	}

	if addon != nil {
		if (version == "" || version == addon.Version) && current.Status != irs.StorageDriverFailed {
			LoggingInfo(hiscallInfo, start)
			return current, nil
		}

		// Installed with another version, or failed: update in place.
		reqBody := ncpAddonUpdateReq{Version: version, ResolveConflicts: ncpAddonResolveConflictsOverwrite}
		if reqBody.Version == "" {
			reqBody.Version = addon.Version
		}
		var updated ncpClusterAddon
		path := "/clusters/" + url.PathEscape(clusterUUID) + "/addons/" + url.PathEscape(spec.AddonName)
		if err := nvch.ncpAddonAPIRequest(http.MethodPatch, path, nil, reqBody, &updated); err != nil {
			enableErr = fmt.Errorf("failed to update NKS add-on %s: %w", spec.AddonName, err)
			return irs.StorageDriverInfo{}, enableErr
		}
		if updated.AddonName == "" {
			updated = *addon
			updated.Version = reqBody.Version
		}
		info := ncpBuildStorageDriverInfo(spec, &updated, true)
		info.Status = irs.StorageDriverUpdating
		LoggingInfo(hiscallInfo, start)
		return info, nil
	}

	if version == "" {
		version, err = nvch.getDefaultAddonVersion(spec.AddonName, k8sVersion)
		if err != nil {
			enableErr = fmt.Errorf("failed to enable storage driver(%s): %w", kind, err)
			return irs.StorageDriverInfo{}, enableErr
		}
	}

	reqBody := []ncpAddonInstallReq{{
		AddonName:           spec.AddonName,
		Version:             version,
		ConfigurationValues: "{}",
		ResolveConflicts:    ncpAddonResolveConflictsOverwrite,
	}}
	var installed ncpClusterAddonList
	path := "/clusters/" + url.PathEscape(clusterUUID) + "/addons"
	if err := nvch.ncpAddonAPIRequest(http.MethodPost, path, nil, reqBody, &installed); err != nil {
		enableErr = fmt.Errorf("failed to install NKS add-on %s: %w", spec.AddonName, err)
		return irs.StorageDriverInfo{}, enableErr
	}

	newAddon := ncpFindAddon(installed.Addons, spec.AddonName)
	if newAddon == nil {
		newAddon = &ncpClusterAddon{AddonName: spec.AddonName, Version: version}
	}
	info := ncpBuildStorageDriverInfo(spec, newAddon, true)
	info.Status = irs.StorageDriverUpdating

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (nvch *NcpVpcClusterHandler) DisableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind) (bool, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NCP Cloud Driver: called DisableStorageDriver()")
	hiscallInfo := GetCallLogScheme(nvch.RegionInfo.Zone, call.CLUSTER, clusterIID.NameId, "DisableStorageDriver()")
	start := call.Start()

	var disableErr error
	defer func() {
		if disableErr != nil {
			LoggingError(hiscallInfo, disableErr)
		}
	}()

	spec, err := ncpStorageDriverSpecOf(kind)
	if err != nil {
		disableErr = err
		return false, disableErr
	}

	cluster, err := nvch.getStorageDriverCluster(clusterIID)
	if err != nil {
		disableErr = fmt.Errorf("failed to disable storage driver(%s): %w", kind, err)
		return false, disableErr
	}
	clusterUUID := ncloud.StringValue(cluster.Uuid)
	k8sVersion := ncloud.StringValue(cluster.K8sVersion)

	addons, err := nvch.getStorageDriverAddons(cluster)
	if err != nil {
		disableErr = fmt.Errorf("failed to disable storage driver(%s): %w", kind, err)
		return false, disableErr
	}

	addon := ncpFindAddon(addons, spec.AddonName)
	current := ncpBuildStorageDriverInfo(spec, addon, ncpIsAddonManagedCSI(k8sVersion))
	if current.ManagementMode == irs.StorageDriverBuiltIn {
		disableErr = fmt.Errorf("storage driver(%s, %s) is built into the NKS cluster (K8s %s) and cannot be disabled through the NCP API",
			kind, spec.AddonName, k8sVersion)
		return false, disableErr
	}

	if addon == nil {
		// Already disabled.
		LoggingInfo(hiscallInfo, start)
		return true, nil
	}

	path := "/clusters/" + url.PathEscape(clusterUUID) + "/addons/" + url.PathEscape(spec.AddonName)
	if err := nvch.ncpAddonAPIRequest(http.MethodDelete, path, nil, nil, nil); err != nil {
		disableErr = fmt.Errorf("failed to delete NKS add-on %s: %w", spec.AddonName, err)
		return false, disableErr
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// getStorageDriverCluster finds the NKS cluster by SystemId (UUID), or by NameId when SystemId is empty.
func (nvch *NcpVpcClusterHandler) getStorageDriverCluster(clusterIID irs.IID) (*vnks.Cluster, error) {
	clusterList, err := ncpClustersGet(nvch.ClusterClient, nvch.Ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster list: %w", err)
	}
	for _, cluster := range clusterList {
		if cluster == nil || cluster.Uuid == nil {
			continue
		}
		if clusterIID.SystemId != "" {
			if ncloud.StringValue(cluster.Uuid) == clusterIID.SystemId {
				return cluster, nil
			}
		} else if clusterIID.NameId != "" && ncloud.StringValue(cluster.Name) == clusterIID.NameId {
			return cluster, nil
		}
	}
	return nil, fmt.Errorf("cluster(name=%s, id=%s) not found", clusterIID.NameId, clusterIID.SystemId)
}

// getStorageDriverAddons lists the cluster add-ons. Clusters older than K8s 1.36 do not
// manage CSI drivers as add-ons, so a failing add-on API is tolerated for them.
func (nvch *NcpVpcClusterHandler) getStorageDriverAddons(cluster *vnks.Cluster) ([]ncpClusterAddon, error) {
	var addonList ncpClusterAddonList
	path := "/clusters/" + url.PathEscape(ncloud.StringValue(cluster.Uuid)) + "/addons"
	query := url.Values{}
	query.Set("page", "1")
	query.Set("size", "100")
	err := nvch.ncpAddonAPIRequest(http.MethodGet, path, query, nil, &addonList)
	if err != nil {
		if !ncpIsAddonManagedCSI(ncloud.StringValue(cluster.K8sVersion)) {
			cblogger.Warnf("NKS add-on list API failed for cluster %s (K8s %s); treating CSI drivers as built-in: %v",
				ncloud.StringValue(cluster.Uuid), ncloud.StringValue(cluster.K8sVersion), err)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list NKS cluster add-ons: %w", err)
	}
	return addonList.Addons, nil
}

// getDefaultAddonVersion returns the latest installable add-on version for the cluster K8s version.
func (nvch *NcpVpcClusterHandler) getDefaultAddonVersion(addonName, k8sVersion string) (string, error) {
	var addonConfig ncpAddonConfig
	query := url.Values{}
	query.Set("k8sVersion", ncpAddonK8sVersion(k8sVersion))
	path := "/addon-configs/" + url.PathEscape(addonName)
	if err := nvch.ncpAddonAPIRequest(http.MethodGet, path, query, nil, &addonConfig); err != nil {
		return "", fmt.Errorf("failed to get NKS add-on config %s: %w", addonName, err)
	}
	latest := ncpLatestVersion(addonConfig.Versions)
	if latest == "" {
		return "", fmt.Errorf("no installable version of NKS add-on %s for K8s %s", addonName, k8sVersion)
	}
	return latest, nil
}

// ncpAddonAPIRequest calls an NKS (vnks/v2) endpoint that the pinned SDK does not cover.
// The base path, region header and HMAC signature follow vnks.APIClient.prepareRequest.
func (nvch *NcpVpcClusterHandler) ncpAddonAPIRequest(method, path string, query url.Values, body interface{}, out interface{}) error {
	accessKey := nvch.CredentialInfo.ClientId
	secretKey := nvch.CredentialInfo.ClientSecret
	cfg := vnks.NewConfiguration(nvch.RegionInfo.Region, &ncloud.APIKey{AccessKey: accessKey, SecretKey: secretKey})

	reqURL := cfg.BasePath + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	ctx := nvch.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	signature, err := hmac.NewSigner(secretKey, crypto.SHA256).Sign(method, reqURL, accessKey, timestamp)
	if err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("x-ncp-apigw-timestamp", timestamp)
	req.Header.Set("x-ncp-iam-access-key", accessKey)
	req.Header.Set("x-ncp-apigw-signature-v1", signature)
	for header, value := range cfg.DefaultHeader {
		req.Header.Set(header, value)
	}

	resp, err := (&http.Client{Timeout: ncpAddonHTTPTimeout}).Do(req)
	if err != nil {
		return fmt.Errorf("%s %s failed: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response of %s %s: %w", method, path, err)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s failed: Status: %s, Body: %s", method, path, resp.Status, string(respBody))
	}

	trimmed := bytes.TrimSpace(respBody)
	if out != nil && len(trimmed) > 0 {
		if err := json.Unmarshal(trimmed, out); err != nil {
			return fmt.Errorf("failed to parse response of %s %s: %w", method, path, err)
		}
	}
	return nil
}

func ncpFindAddon(addons []ncpClusterAddon, addonName string) *ncpClusterAddon {
	for i := range addons {
		if addons[i].AddonName == addonName {
			return &addons[i]
		}
	}
	return nil
}

// ncpBuildStorageDriverInfo maps an NKS add-on (nil when not installed) to StorageDriverInfo.
// A missing add-on on a cluster that does not manage CSI as add-ons means the driver is preinstalled.
func ncpBuildStorageDriverInfo(spec ncpStorageDriverSpec, addon *ncpClusterAddon, addonManaged bool) irs.StorageDriverInfo {
	info := irs.StorageDriverInfo{
		Kind:          spec.Kind,
		CSPDriverName: spec.AddonName,
		CSIDriverName: spec.CSIDriverName,
	}

	if addon == nil {
		if addonManaged {
			info.ManagementMode = irs.StorageDriverAddon
			info.Status = irs.StorageDriverDisabled
		} else {
			info.ManagementMode = irs.StorageDriverBuiltIn
			info.Status = irs.StorageDriverEnabled
		}
		return info
	}

	info.ManagementMode = irs.StorageDriverAddon
	info.Status = ncpConvertAddonStatus(addon.Status)
	info.Version = addon.Version

	keyValues := []irs.KeyValue{}
	if addon.Uuid != "" {
		keyValues = append(keyValues, irs.KeyValue{Key: "AddonUuid", Value: addon.Uuid})
	}
	if addon.Status != "" {
		keyValues = append(keyValues, irs.KeyValue{Key: "AddonStatus", Value: addon.Status})
	}
	if addon.Message != "" {
		keyValues = append(keyValues, irs.KeyValue{Key: "Message", Value: addon.Message})
	}
	if addon.LatestVersion != "" {
		keyValues = append(keyValues, irs.KeyValue{Key: "LatestVersion", Value: addon.LatestVersion})
	}
	if len(keyValues) > 0 {
		info.KeyValueList = keyValues
	}
	return info
}

// ncpConvertAddonStatus maps an NKS add-on status (e.g., running, install_in_progress,
// update_in_progress, delete_in_progress) to StorageDriverStatus.
func ncpConvertAddonStatus(status string) irs.StorageDriverStatus {
	s := strings.ToLower(strings.TrimSpace(status))
	switch {
	case s == "running", s == "active", s == "installed":
		return irs.StorageDriverEnabled
	case strings.Contains(s, "fail"), strings.Contains(s, "error"):
		return irs.StorageDriverFailed
	case strings.Contains(s, "progress"), strings.Contains(s, "pending"),
		strings.Contains(s, "installing"), strings.Contains(s, "updating"), strings.Contains(s, "deleting"):
		return irs.StorageDriverUpdating
	default:
		return irs.StorageDriverUnknown
	}
}

// ncpIsAddonManagedCSI reports whether the NKS cluster version (e.g., "1.36.2-nks.1") manages
// CSI drivers as add-ons. Unparsable versions are treated as not add-on managed.
func ncpIsAddonManagedCSI(k8sVersion string) bool {
	major, minor, ok := ncpParseMajorMinor(k8sVersion)
	if !ok {
		return false
	}
	if major != ncpAddonManagedCSIMajor {
		return major > ncpAddonManagedCSIMajor
	}
	return minor >= ncpAddonManagedCSIMinor
}

func ncpParseMajorMinor(k8sVersion string) (int, int, bool) {
	parts := strings.Split(strings.TrimPrefix(ncpAddonK8sVersion(k8sVersion), "v"), ".")
	if len(parts) < 2 {
		return 0, 0, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}
	return major, minor, true
}

// ncpAddonK8sVersion strips the NKS suffix: "1.36.2-nks.1" -> "1.36.2".
func ncpAddonK8sVersion(k8sVersion string) string {
	v := strings.TrimSpace(k8sVersion)
	if idx := strings.Index(v, "-"); idx >= 0 {
		v = v[:idx]
	}
	return v
}

// ncpLatestVersion returns the highest semantic version, or the first entry if none parse.
func ncpLatestVersion(versions []string) string {
	latest, latestCanon := "", ""
	for _, v := range versions {
		canon := v
		if !strings.HasPrefix(canon, "v") {
			canon = "v" + canon
		}
		if !semver.IsValid(canon) {
			continue
		}
		if latestCanon == "" || semver.Compare(canon, latestCanon) > 0 {
			latest, latestCanon = v, canon
		}
	}
	if latest == "" && len(versions) > 0 {
		return versions[0]
	}
	return latest
}
