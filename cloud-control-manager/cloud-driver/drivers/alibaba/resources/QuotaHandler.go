// Alibaba Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Alibaba Driver.
//
// by CB-Spider Team, 2025.07.

package resources

import (
	"strconv"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaQuotaHandler struct {
	Region idrv.RegionInfo
	Client *ecs.Client
}

// alibabaQuotaEntry defines a mapping from Alibaba attribute names to a CB-Spider ResourceType,
// with optional pairing of a "used" attribute for current usage.
type alibabaQuotaEntry struct {
	resourceType string
	limitAttr    string
	usedAttr     string // empty if not available
	unit         string
	description  string
}

var alibabaQuotaEntries = []alibabaQuotaEntry{
	{
		resourceType: "SecurityGroup",
		limitAttr:    "max-security-groups",
		usedAttr:     "",
		unit:         "count",
		description:  "Maximum number of security groups",
	},
	{
		resourceType: "SecurityGroupRule",
		limitAttr:    "max-ip-per-vpc-security-group",
		usedAttr:     "",
		unit:         "count",
		description:  "Maximum IP entries per VPC security group",
	},
	{
		resourceType: "vCPU",
		limitAttr:    "max-postpaid-instance-vcpu-count",
		usedAttr:     "used-postpaid-instance-vcpu-count",
		unit:         "vCPU",
		description:  "Pay-as-you-go instance vCPU limit",
	},
	{
		resourceType: "vCPU-Spot",
		limitAttr:    "max-spot-instance-vcpu-count",
		usedAttr:     "used-spot-instance-vcpu-count",
		unit:         "vCPU",
		description:  "Spot instance vCPU limit",
	},
	{
		resourceType: "NIC",
		limitAttr:    "max-elastic-network-interfaces",
		usedAttr:     "",
		unit:         "count",
		description:  "Maximum elastic network interfaces",
	},
	{
		resourceType: "Disk-TB",
		limitAttr:    "max-postpaid-yundisk-capacity",
		usedAttr:     "used-postpaid-yundisk-capacity",
		unit:         "GiB",
		description:  "Total pay-as-you-go cloud disk capacity (sum across disk types)",
	},
}

func (handler *AlibabaQuotaHandler) GetQuota() (irs.QuotaInfo, error) {
	cblogger.Info("Alibaba Driver: called GetQuota()")

	quotaInfo := irs.QuotaInfo{
		CSP:    "Alibaba",
		Region: handler.Region.Region,
	}

	request := ecs.CreateDescribeAccountAttributesRequest()
	request.RegionId = handler.Region.Region

	response, err := handler.Client.DescribeAccountAttributes(request)
	if err != nil {
		return quotaInfo, err
	}

	// Collect all attribute values into a map (attribute name → first numeric value string)
	// For multi-value attributes (e.g. disk capacity per type), we sum all values.
	attrMap := make(map[string]string)
	for _, item := range response.AccountAttributeItems.AccountAttributeItem {
		vals := item.AttributeValues.ValueItem
		if len(vals) == 0 {
			continue
		}
		// Try to sum numeric values; if non-numeric, just take the first
		sum := 0.0
		allNumeric := true
		for _, v := range vals {
			f, err := strconv.ParseFloat(strings.TrimSpace(v.Value), 64)
			if err != nil {
				allNumeric = false
				break
			}
			sum += f
		}
		if allNumeric && len(vals) > 1 {
			attrMap[item.AttributeName] = strconv.FormatFloat(sum, 'f', -1, 64)
		} else {
			attrMap[item.AttributeName] = strings.TrimSpace(vals[0].Value)
		}
	}

	var resourceQuotas []irs.ResourceQuota
	for _, entry := range alibabaQuotaEntries {
		limitStr, hasLimit := attrMap[entry.limitAttr]
		if !hasLimit {
			continue
		}

		used := "NA"
		available := "NA"
		if entry.usedAttr != "" {
			if u, ok := attrMap[entry.usedAttr]; ok {
				used = u
				limitF, e1 := strconv.ParseFloat(limitStr, 64)
				usedF, e2 := strconv.ParseFloat(used, 64)
				if e1 == nil && e2 == nil {
					available = strconv.FormatFloat(limitF-usedF, 'f', -1, 64)
				}
			}
		}

		resourceQuotas = append(resourceQuotas, irs.ResourceQuota{
			ResourceType: entry.resourceType,
			Limit:        limitStr,
			Used:         used,
			Available:    available,
			Unit:         entry.unit,
			Description:  entry.description,
		})
	}

	quotaInfo.ResourceQuotas = resourceQuotas
	return quotaInfo, nil
}
