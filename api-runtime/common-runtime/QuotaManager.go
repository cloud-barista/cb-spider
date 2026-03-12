// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2025.07.

package commonruntime

import (
	"sort"
	"strings"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// quotaSortOrder defines the canonical display order for ResourceType groups.
// Each entry is a prefix/substring that the ResourceType starts with (case-insensitive).
// Items not matching any prefix are placed at the end.
var quotaSortOrder = []string{
	"vpc",
	"subnet",
	"securitygroup",
	"keypair",
	"vcpu",
	"nic",
	"publicip",
	"vm",
	"disk",
	"snapshot",
	"myimage",
	"image",
	"nlb",
	"alb",
	"lb",
	"cluster",
	"s3",
	"objectstorage",
	"filesystem",
	"fs",
}

// quotaGroupIndex returns the sort group index for a given ResourceType.
// Lower index = earlier in output. Unknown types get the highest index.
func quotaGroupIndex(resourceType string) int {
	lower := strings.ToLower(resourceType)
	for i, prefix := range quotaSortOrder {
		if strings.HasPrefix(lower, prefix) {
			return i
		}
	}
	return len(quotaSortOrder)
}

// sortQuotaResourceTypes sorts ResourceQuotas in the canonical display order.
func sortQuotaResourceTypes(quotas []cres.ResourceQuota) {
	sort.SliceStable(quotas, func(i, j int) bool {
		gi := quotaGroupIndex(quotas[i].ResourceType)
		gj := quotaGroupIndex(quotas[j].ResourceType)
		if gi != gj {
			return gi < gj
		}
		// Within the same group, sort alphabetically for stable output
		return quotas[i].ResourceType < quotas[j].ResourceType
	})
}

// ================ Quota Handler

// GetQuota retrieves the resource quota limits and current usage for the specified connection.
func GetQuota(connectionName string) (cres.QuotaInfo, error) {
	cblog.Info("call GetQuota()")

	// check empty and trim user input
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return cres.QuotaInfo{}, err
	}

	if err := checkCapability(connectionName, QUOTA_HANDLER); err != nil {
		return cres.QuotaInfo{}, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return cres.QuotaInfo{}, err
	}

	handler, err := cldConn.CreateQuotaHandler()
	if err != nil {
		cblog.Error(err)
		return cres.QuotaInfo{}, err
	}

	quotaInfo, err := handler.GetQuota()
	if err != nil {
		cblog.Error(err)
		return cres.QuotaInfo{}, err
	}

	// Ensure the slice is never nil
	if quotaInfo.ResourceQuotas == nil {
		quotaInfo.ResourceQuotas = []cres.ResourceQuota{}
	}

	// Sort in canonical display order
	sortQuotaResourceTypes(quotaInfo.ResourceQuotas)

	return quotaInfo, nil
}
