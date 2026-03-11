// Azure Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Azure Driver.
//
// by CB-Spider Team, 2025.07.

package resources

import (
	"context"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzureQuotaHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Ctx            context.Context
}

// azureComputeQuotaMap maps Azure compute usage name to CB-Spider ResourceType.
var azureComputeQuotaMap = map[string]string{
	"cores":                      "vCPU",
	"virtualMachines":            "VM",
	"standardDSv3Family":         "vCPU-DSv3",
	"standardDSv4Family":         "vCPU-DSv4",
	"standardFSv2Family":         "vCPU-FSv2",
	"standardNCSFamily":          "vCPU-NCS",
	"standardNCFamily":           "vCPU-NC",
	"standardNDSFamily":          "vCPU-NDS",
	"availabilitySets":           "AvailabilitySet",
	// Disk counts
	"PremiumDiskCount":           "Disk-Premium",
	"StandardSSDDiskCount":       "Disk-SSD",
	"StandardDiskCount":          "Disk-HDD",
	// Snapshot counts
	"PremiumSnapshotCount":       "Snapshot-Premium",
	"StandardSSDSnapshotCount":   "Snapshot-SSD",
	"StandardSnapshotCount":      "Snapshot-HDD",
}

// azureNetworkQuotaMap maps Azure network usage name to CB-Spider ResourceType.
var azureNetworkQuotaMap = map[string]string{
	"VirtualNetworks":                "VPC",
	"SubnetsPerVirtualNetwork":       "Subnet",
	"NetworkSecurityGroups":          "SecurityGroup",
	"SecurityRulesPerNetworkSecurityGroup": "SecurityGroupRule",
	"PublicIPAddresses":              "PublicIP",
	"LoadBalancers":                  "NLB",
	"NetworkInterfaces":              "NIC",
}

func (handler *AzureQuotaHandler) GetQuota() (irs.QuotaInfo, error) {
	cblogger.Info("Azure Driver: called GetQuota()")

	quotaInfo := irs.QuotaInfo{
		CSP:    "Azure",
		Region: handler.Region.Region,
	}

	var resourceQuotas []irs.ResourceQuota

	subscriptionID := handler.CredentialInfo.SubscriptionId
	location := handler.Region.Region

	cred, err := azidentity.NewClientSecretCredential(
		handler.CredentialInfo.TenantId,
		handler.CredentialInfo.ClientId,
		handler.CredentialInfo.ClientSecret,
		nil,
	)
	if err != nil {
		return quotaInfo, err
	}

	// Compute Usages
	computeClient, err := armcompute.NewUsageClient(subscriptionID, cred, nil)
	if err != nil {
		return quotaInfo, err
	}
	pager := computeClient.NewListPager(location, nil)
	for pager.More() {
		page, err := pager.NextPage(handler.Ctx)
		if err != nil {
			cblogger.Warnf("Azure GetQuota: failed to list compute usage: %v", err)
			break
		}
		for _, u := range page.Value {
			if u.Name == nil || u.Name.Value == nil {
				continue
			}
			nameVal := strings.TrimSpace(*u.Name.Value)
			resourceType, ok := azureComputeQuotaMap[nameVal]
			if !ok {
				// try prefix for vCPU families
				if strings.HasSuffix(nameVal, "Family") || strings.HasSuffix(nameVal, "Promo") {
					resourceType = "vCPU-" + nameVal
				} else {
					continue
				}
			}
			limit := "NA"
			used := "NA"
			available := "NA"
			if u.Limit != nil {
				limit = strconv.FormatInt(*u.Limit, 10)
			}
			if u.CurrentValue != nil {
				used = strconv.FormatInt(int64(*u.CurrentValue), 10)
			}
			if u.Limit != nil && u.CurrentValue != nil {
				av := *u.Limit - int64(*u.CurrentValue)
				available = strconv.FormatInt(av, 10)
			}
			localName := ""
			if u.Name.LocalizedValue != nil {
				localName = *u.Name.LocalizedValue
			}
			rq := irs.ResourceQuota{
				ResourceType: resourceType,
				Limit:        limit,
				Used:         used,
				Available:    available,
				Unit:         "count",
				Description:  "[Compute] " + localName,
			}
			resourceQuotas = append(resourceQuotas, rq)
		}
	}

	// Network Usages
	networkClient, err := armnetwork.NewUsagesClient(subscriptionID, cred, nil)
	if err != nil {
		cblogger.Warnf("Azure GetQuota: failed to create network usage client: %v", err)
	} else {
		netPager := networkClient.NewListPager(location, nil)
		for netPager.More() {
			page, err := netPager.NextPage(handler.Ctx)
			if err != nil {
				cblogger.Warnf("Azure GetQuota: failed to list network usage: %v", err)
				break
			}
			for _, u := range page.Value {
				if u.Name == nil || u.Name.Value == nil {
					continue
				}
				nameVal := strings.TrimSpace(*u.Name.Value)
				resourceType, ok := azureNetworkQuotaMap[nameVal]
				if !ok {
					continue
				}
				limit := "NA"
				used := "NA"
				available := "NA"
				if u.Limit != nil {
					limit = strconv.FormatInt(*u.Limit, 10)
				}
				if u.CurrentValue != nil {
					used = strconv.FormatInt(*u.CurrentValue, 10)
				}
				if u.Limit != nil && u.CurrentValue != nil {
					av := *u.Limit - *u.CurrentValue
					available = strconv.FormatInt(av, 10)
				}
				localName := ""
				if u.Name.LocalizedValue != nil {
					localName = *u.Name.LocalizedValue
				}
				rq := irs.ResourceQuota{
					ResourceType: resourceType,
					Limit:        limit,
					Used:         used,
					Available:    available,
					Unit:         "count",
					Description:  "[Network] " + localName,
				}
				resourceQuotas = append(resourceQuotas, rq)
			}
		}
	}

	quotaInfo.ResourceQuotas = resourceQuotas
	return quotaInfo, nil
}
