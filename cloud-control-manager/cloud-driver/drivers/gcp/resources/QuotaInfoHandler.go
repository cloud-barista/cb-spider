// GCP Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is GCP Driver.
//
// by CB-Spider Team, 2025.07.

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	cloudquotas "cloud.google.com/go/cloudquotas/apiv1"
	cloudquotaspb "cloud.google.com/go/cloudquotas/apiv1/cloudquotaspb"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	serviceusage "google.golang.org/api/serviceusage/v1beta1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GCPQuotaInfoHandler struct {
	Region     idrv.RegionInfo
	Credential idrv.CredentialInfo
	Ctx        context.Context
}

// credentialJSON builds a minimal service-account JSON from handler credentials.
func (handler *GCPQuotaInfoHandler) credentialJSON() []byte {
	data := map[string]string{
		"type":         "service_account",
		"private_key":  handler.Credential.PrivateKey,
		"client_email": handler.Credential.ClientEmail,
	}
	b, _ := json.Marshal(data)
	return b
}

// newServiceUsageClient creates a serviceusage v1beta1 client using JSON credentials.
// Used only for ListServiceType (Cloud Quotas API has no ListServices RPC).
func (handler *GCPQuotaInfoHandler) newServiceUsageClient() (*serviceusage.APIService, error) {
	return serviceusage.NewService(handler.Ctx,
		option.WithCredentialsJSON(handler.credentialJSON()),
		option.WithScopes("https://www.googleapis.com/auth/cloud-platform.read-only"),
	)
}

// newCloudQuotasClient creates a Cloud Quotas API v1 client using JSON credentials.
func (handler *GCPQuotaInfoHandler) newCloudQuotasClient() (*cloudquotas.Client, error) {
	return cloudquotas.NewRESTClient(handler.Ctx,
		option.WithCredentialsJSON(handler.credentialJSON()),
	)
}

// newMonitoringClient creates a Cloud Monitoring API v3 client using JSON credentials.
func (handler *GCPQuotaInfoHandler) newMonitoringClient() (*monitoring.MetricClient, error) {
	return monitoring.NewMetricClient(handler.Ctx,
		option.WithCredentialsJSON(handler.credentialJSON()),
	)
}

// ListServiceType dynamically discovers enabled GCP services.
// Returns service DNS names (e.g., "compute.googleapis.com", "dns.googleapis.com").
// Uses Service Usage API because Cloud Quotas API has no ListServices RPC.
func (handler *GCPQuotaInfoHandler) ListServiceType() ([]string, error) {
	cblogger.Info("GCP Driver: called ListServiceType()")

	svc, err := handler.newServiceUsageClient()
	if err != nil {
		return nil, fmt.Errorf("GCP ListServiceType: failed to create service usage client: %w", err)
	}

	projectID := handler.Credential.ProjectID
	parent := "projects/" + projectID

	var serviceTypes []string

	// List enabled services
	err = svc.Services.List(parent).Filter("state:ENABLED").Pages(handler.Ctx,
		func(resp *serviceusage.ListServicesResponse) error {
			for _, s := range resp.Services {
				if s.Config == nil || s.Config.Name == "" {
					continue
				}
				serviceTypes = append(serviceTypes, s.Config.Name)
			}
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("GCP ListServiceType: %w", err)
	}

	sort.Strings(serviceTypes)
	return serviceTypes, nil
}

// usageKey is the map key for looking up quota usage by metric and location.
type usageKey struct {
	quotaMetric string // e.g., "compute.googleapis.com/cpus"
	location    string // e.g., "asia-northeast3", "global"
}

// fetchQuotaUsage queries Cloud Monitoring API for quota allocation usage of
// the given service. Returns a map from (quota_metric, location) to usage value.
// If the monitoring query fails, it logs a warning and returns an empty map
// so that GetQuotaInfo can still return limit-only results.
func (handler *GCPQuotaInfoHandler) fetchQuotaUsage(serviceType string) map[usageKey]int64 {
	usageMap := make(map[usageKey]int64)

	mc, err := handler.newMonitoringClient()
	if err != nil {
		cblogger.Warnf("GCP QuotaInfoHandler: failed to create monitoring client: %v", err)
		return usageMap
	}
	defer mc.Close()

	projectID := handler.Credential.ProjectID
	now := time.Now()

	// Query the last 1 hour of quota allocation usage for the service.
	// Quota metrics are GAUGE and may be reported every few minutes.
	filter := fmt.Sprintf(
		`metric.type = "serviceruntime.googleapis.com/quota/allocation/usage" AND `+
			`resource.type = "consumer_quota" AND `+
			`resource.label.service = "%s"`,
		serviceType,
	)

	cblogger.Infof("GCP QuotaInfoHandler: monitoring filter: %s", filter)

	it := mc.ListTimeSeries(handler.Ctx, &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + projectID,
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(now.Add(-1 * time.Hour)),
			EndTime:   timestamppb.New(now),
		},
		View: monitoringpb.ListTimeSeriesRequest_FULL,
	})

	count := 0
	for {
		ts, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			cblogger.Warnf("GCP QuotaInfoHandler: monitoring query error for %s: %v", serviceType, err)
			break
		}
		count++

		// Extract the quota_metric label (e.g., "compute.googleapis.com/cpus")
		quotaMetric := ""
		if ts.GetMetric() != nil {
			quotaMetric = ts.GetMetric().GetLabels()["quota_metric"]
		}
		if quotaMetric == "" {
			cblogger.Debugf("GCP QuotaInfoHandler: skipping time series with empty quota_metric, metric labels: %v", ts.GetMetric().GetLabels())
			continue
		}

		// Extract location from resource labels
		location := "global"
		if ts.GetResource() != nil {
			if loc := ts.GetResource().GetLabels()["location"]; loc != "" {
				location = loc
			}
		}

		// Get the latest data point (points are returned in reverse time order)
		points := ts.GetPoints()
		if len(points) == 0 {
			continue
		}

		var value int64
		pt := points[0]
		if pt.GetValue() != nil {
			value = pt.GetValue().GetInt64Value()
		}

		key := usageKey{quotaMetric: quotaMetric, location: location}
		usageMap[key] = value
		cblogger.Debugf("GCP QuotaInfoHandler: usage entry [quota_metric=%s, location=%s] = %d", quotaMetric, location, value)
	}

	cblogger.Infof("GCP QuotaInfoHandler: fetched %d time series, %d usage entries for %s", count, len(usageMap), serviceType)
	return usageMap
}

// GetQuotaInfo retrieves all quota information for the given GCP service using the
// Cloud Quotas API (cloudquotas.googleapis.com) for limits.
// For compute.googleapis.com, usage data is fetched from the Compute Engine API
// (Regions.Get / Projects.Get) which directly provides quota usage.
// For other services, usage is fetched from the Cloud Monitoring API (best-effort).
// The serviceType parameter should be a service DNS name
// (e.g., "compute.googleapis.com") as returned by ListServiceType.
// Results are filtered to the handler's region, its zones, or global quotas.
func (handler *GCPQuotaInfoHandler) GetQuotaInfo(serviceType string) (irs.QuotaInfo, error) {
	cblogger.Infof("GCP Driver: called GetQuotaInfo(serviceType=%s)", serviceType)

	region := handler.Region.Region
	quotaInfo := irs.QuotaInfo{
		CSP:    "GCP",
		Region: region,
	}

	client, err := handler.newCloudQuotasClient()
	if err != nil {
		return quotaInfo, fmt.Errorf("GCP GetQuotaInfo: failed to create Cloud Quotas client: %w", err)
	}
	defer client.Close()

	// Fetch usage data based on service type
	isCompute := (serviceType == "compute.googleapis.com")
	var computeRegionalUsage, computeGlobalUsage map[string]float64
	var usageMap map[usageKey]int64
	if isCompute {
		// For compute, use Compute Engine API which directly returns usage
		computeRegionalUsage, computeGlobalUsage = handler.fetchComputeUsage()
	} else {
		// For other services, try Cloud Monitoring (best-effort)
		usageMap = handler.fetchQuotaUsage(serviceType)
	}

	projectID := handler.Credential.ProjectID
	parent := fmt.Sprintf("projects/%s/locations/global/services/%s", projectID, serviceType)

	it := client.ListQuotaInfos(handler.Ctx, &cloudquotaspb.ListQuotaInfosRequest{
		Parent: parent,
	})

	var quotas []irs.Quota
	for {
		qi, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return quotaInfo, fmt.Errorf("GCP GetQuotaInfo(%s): %w", serviceType, err)
		}

		displayName := qi.GetQuotaDisplayName()
		if displayName == "" {
			displayName = qi.GetMetricDisplayName()
		}
		if displayName == "" {
			displayName = qi.GetQuotaId()
		}

		unit := qi.GetMetricUnit()
		if unit == "" {
			unit = "NA"
		}

		quotaMetric := qi.GetMetric() // e.g., "compute.googleapis.com/cpus"

		// Iterate dimension infos (each represents a dimension combination with its quota value)
		for _, di := range qi.GetDimensionsInfos() {
			// Filter: include only global or region/zone-matching entries
			if !handler.matchesLocation(di, region) {
				continue
			}

			details := di.GetDetails()
			if details == nil {
				continue
			}

			quotaValue := details.GetValue()

			// Skip unlimited (0 or negative) quotas, consistent with GCP Console behavior
			if quotaValue <= 0 {
				continue
			}

			limitVal := strconv.FormatInt(quotaValue, 10)
			locDesc := handler.locationString(di)

			// Look up usage data.
			used := "NA"
			available := "NA"
			loc := handler.locationForUsageLookup(di, region)

			if isCompute {
				// For compute: map Cloud Quotas metric to Compute API metric name
				// e.g., "compute.googleapis.com/cpus" → "cpus"
				metricSuffix := strings.TrimPrefix(quotaMetric, "compute.googleapis.com/")
				isGlobal := (loc == "global")
				var usageSource map[string]float64
				if isGlobal {
					usageSource = computeGlobalUsage
				} else {
					usageSource = computeRegionalUsage
				}
				if usage, ok := usageSource[metricSuffix]; ok {
					usedInt := int64(usage)
					used = strconv.FormatInt(usedInt, 10)
					avail := quotaValue - usedInt
					if avail < 0 {
						avail = 0
					}
					available = strconv.FormatInt(avail, 10)
				}
			} else {
				// For non-compute: use Cloud Monitoring data
				if usage, ok := usageMap[usageKey{quotaMetric: quotaMetric, location: loc}]; ok {
					used = strconv.FormatInt(usage, 10)
					avail := quotaValue - usage
					if avail < 0 {
						avail = 0
					}
					available = strconv.FormatInt(avail, 10)
				}
			}

			rq := irs.Quota{
				QuotaName:   displayName,
				Limit:       limitVal,
				Used:        used,
				Available:   available,
				Unit:        unit,
				Description: fmt.Sprintf("%s", locDesc),
			}
			quotas = append(quotas, rq)
		}
	}

	quotaInfo.Quotas = quotas
	return quotaInfo, nil
}

// fetchComputeUsage retrieves quota usage from the Compute Engine API.
// Returns two maps keyed by lowercase metric name (e.g., "cpus", "disks_total_gb"):
//   - regional: usage for the handler's region (from Regions.Get)
//   - global: project-level usage (from Projects.Get)
//
// This is more reliable than Cloud Monitoring for compute quotas because the
// Compute Engine API directly includes usage in its quota response.
func (handler *GCPQuotaInfoHandler) fetchComputeUsage() (regional map[string]float64, global map[string]float64) {
	regional = make(map[string]float64)
	global = make(map[string]float64)

	svc, err := compute.NewService(handler.Ctx,
		option.WithCredentialsJSON(handler.credentialJSON()),
		option.WithScopes(compute.ComputeReadonlyScope),
	)
	if err != nil {
		cblogger.Warnf("GCP QuotaInfoHandler: failed to create compute client for usage: %v", err)
		return
	}

	projectID := handler.Credential.ProjectID
	regionName := handler.Region.Region

	// Fetch regional quotas (e.g., CPUs, DISKS_TOTAL_GB per region)
	reg, err := svc.Regions.Get(projectID, regionName).Do()
	if err != nil {
		cblogger.Warnf("GCP QuotaInfoHandler: failed to get region %s quotas: %v", regionName, err)
	} else {
		for _, q := range reg.Quotas {
			regional[strings.ToLower(q.Metric)] = q.Usage
		}
	}

	// Fetch project-level (global) quotas (e.g., SNAPSHOTS, NETWORKS)
	proj, err := svc.Projects.Get(projectID).Do()
	if err != nil {
		cblogger.Warnf("GCP QuotaInfoHandler: failed to get project quotas: %v", err)
	} else {
		for _, q := range proj.Quotas {
			global[strings.ToLower(q.Metric)] = q.Usage
		}
	}

	cblogger.Infof("GCP QuotaInfoHandler: fetched %d regional, %d global compute usage entries", len(regional), len(global))
	return
}

// locationForUsageLookup returns the location string to use when looking up
// quota usage in the monitoring data. For global quotas (ApplicableLocations
// contains only "global" or Dimensions is empty) it returns "global".
// Otherwise it returns the connection's region, since matchesLocation() has
// already filtered entries to only the target region or its zones.
func (handler *GCPQuotaInfoHandler) locationForUsageLookup(di *cloudquotaspb.DimensionsInfo, region string) string {
	locs := di.GetApplicableLocations()
	if len(locs) == 0 {
		return "global"
	}
	for _, loc := range locs {
		if !strings.EqualFold(loc, "global") {
			return strings.ToLower(region)
		}
	}
	return "global"
}

// matchesLocation returns true if the DimensionsInfo applies to the target
// region. It checks ApplicableLocations for "global", the exact region, or
// any zone belonging to the region (e.g., "asia-northeast3-a" for region
// "asia-northeast3"). If ApplicableLocations is empty, falls back to checking
// the Dimensions map.
func (handler *GCPQuotaInfoHandler) matchesLocation(di *cloudquotaspb.DimensionsInfo, region string) bool {
	locs := di.GetApplicableLocations()

	// If locations list is available, use it for filtering
	if len(locs) > 0 {
		for _, loc := range locs {
			lower := strings.ToLower(loc)
			if lower == "global" {
				return true
			}
			if strings.EqualFold(loc, region) {
				return true
			}
			// Zone belongs to region: "asia-northeast3-a" starts with "asia-northeast3-"
			if strings.HasPrefix(lower, strings.ToLower(region)+"-") {
				return true
			}
		}
		return false
	}

	// Fallback: check Dimensions map
	dims := di.GetDimensions()
	if len(dims) == 0 {
		return true // global (no dimensions)
	}
	if r, ok := dims["region"]; ok {
		return strings.EqualFold(r, region)
	}
	if z, ok := dims["zone"]; ok {
		return strings.HasPrefix(strings.ToLower(z), strings.ToLower(region)+"-")
	}
	return true
}

// locationString formats the applicable locations or dimensions for display.
func (handler *GCPQuotaInfoHandler) locationString(di *cloudquotaspb.DimensionsInfo) string {
	locs := di.GetApplicableLocations()
	if len(locs) > 0 {
		sort.Strings(locs)
		return "(" + strings.Join(locs, ", ") + ")"
	}

	dims := di.GetDimensions()
	if len(dims) == 0 {
		return "(global)"
	}
	parts := make([]string, 0, len(dims))
	for k, v := range dims {
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	return "(" + strings.Join(parts, ", ") + ")"
}
