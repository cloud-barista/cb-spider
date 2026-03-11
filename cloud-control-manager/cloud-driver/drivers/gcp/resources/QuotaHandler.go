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
	"strconv"
	"strings"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	compute "google.golang.org/api/compute/v1"
)

type GCPQuotaHandler struct {
	Region     idrv.RegionInfo
	Credential idrv.CredentialInfo
	Ctx        context.Context
	Client     *compute.Service
}

// gcpRegionalQuotaMap maps GCP regional quota metric to CB-Spider ResourceType.
var gcpRegionalQuotaMap = map[string]string{
	"CPUS":              "vCPU",
	"INSTANCES":         "VM",
	"DISKS_TOTAL_GB":    "Disk-GB",
	"SSD_TOTAL_GB":      "Disk-SSD-GB",
	"IN_USE_ADDRESSES":  "PublicIP",
	"STATIC_ADDRESSES":  "StaticIP",
	"NETWORKS":          "VPC",
	"SUBNETWORKS":       "Subnet",
	"FORWARDING_RULES":  "NLB",
	"ROUTERS":           "Router",
	"SNAPSHOTS":         "Snapshot",
	// GPU (NVIDIA)
	"NVIDIA_K80_GPUS":      "GPU-K80",
	"NVIDIA_P100_GPUS":     "GPU-P100",
	"NVIDIA_P100_VWS_GPUS": "GPU-P100-VWS",
	"NVIDIA_V100_GPUS":     "GPU-V100",
	"NVIDIA_P4_GPUS":       "GPU-P4",
	"NVIDIA_P4_VWS_GPUS":   "GPU-P4-VWS",
	"NVIDIA_T4_GPUS":       "GPU-T4",
	"NVIDIA_T4_VWS_GPUS":   "GPU-T4-VWS",
	"NVIDIA_A100_GPUS":     "GPU-A100",
	"NVIDIA_A100_80GB_GPUS":"GPU-A100-80GB",
	"NVIDIA_L4_GPUS":       "GPU-L4",
	// TPU
	"TPU_LITE_DEVICE_V5":   "TPU-V5",
	"TPU_LITE_PODSLICE_V5": "TPU-V5-PodSlice",
}

// gcpGlobalQuotaMap maps GCP global quota metric to CB-Spider ResourceType.
var gcpGlobalQuotaMap = map[string]string{
	"FIREWALLS":        "SecurityGroup",
	"NETWORKS":         "VPC-Global",
	"FORWARDING_RULES": "NLB-Global",
	"IN_USE_ADDRESSES": "PublicIP-Global",
	"CPUS_ALL_REGIONS": "vCPU-AllRegions",
}

func (handler *GCPQuotaHandler) GetQuota() (irs.QuotaInfo, error) {
	cblogger.Info("GCP Driver: called GetQuota()")

	projectID := handler.Credential.ProjectID
	region := handler.Region.Region

	quotaInfo := irs.QuotaInfo{
		CSP:    "GCP",
		Region: region,
	}

	var resourceQuotas []irs.ResourceQuota

	// Regional quotas
	regionResult, err := handler.Client.Regions.Get(projectID, region).Context(handler.Ctx).Do()
	if err != nil {
		cblogger.Warnf("GCP GetQuota: failed to get regional quotas for %s: %v", region, err)
	} else {
		for _, q := range regionResult.Quotas {
			resourceType, ok := gcpRegionalQuotaMap[strings.ToUpper(q.Metric)]
			if !ok {
				continue
			}
			limit := strconv.FormatFloat(q.Limit, 'f', -1, 64)
			used := strconv.FormatFloat(q.Usage, 'f', -1, 64)
			available := "NA"
			if q.Limit >= q.Usage {
				available = strconv.FormatFloat(q.Limit-q.Usage, 'f', -1, 64)
			}
			unit := "count"
			if strings.HasSuffix(resourceType, "-GB") || strings.HasSuffix(resourceType, "GB") {
				unit = "GB"
			}
			rq := irs.ResourceQuota{
				ResourceType: resourceType,
				Limit:        limit,
				Used:         used,
				Available:    available,
				Unit:         unit,
				Description:  "[Regional] " + q.Metric,
			}
			resourceQuotas = append(resourceQuotas, rq)
		}
	}

	// Global quotas
	projectResult, err := handler.Client.Projects.Get(projectID).Context(handler.Ctx).Do()
	if err != nil {
		cblogger.Warnf("GCP GetQuota: failed to get global project quotas: %v", err)
	} else {
		for _, q := range projectResult.Quotas {
			resourceType, ok := gcpGlobalQuotaMap[strings.ToUpper(q.Metric)]
			if !ok {
				continue
			}
			limit := strconv.FormatFloat(q.Limit, 'f', -1, 64)
			used := strconv.FormatFloat(q.Usage, 'f', -1, 64)
			available := "NA"
			if q.Limit >= q.Usage {
				available = strconv.FormatFloat(q.Limit-q.Usage, 'f', -1, 64)
			}
			rq := irs.ResourceQuota{
				ResourceType: resourceType,
				Limit:        limit,
				Used:         used,
				Available:    available,
				Unit:         "count",
				Description:  "[Global] " + q.Metric,
			}
			resourceQuotas = append(resourceQuotas, rq)
		}
	}

	quotaInfo.ResourceQuotas = resourceQuotas
	return quotaInfo, nil
}
