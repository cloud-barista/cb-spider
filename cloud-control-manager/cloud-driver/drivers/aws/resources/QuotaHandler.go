// AWS Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is AWS Driver.
//
// by CB-Spider Team, 2025.07.

package resources

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/servicequotas"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsQuotaHandler struct {
	Region   idrv.RegionInfo
	Client   *servicequotas.ServiceQuotas
	CwClient *cloudwatch.CloudWatch
}

// awsQuotaServiceCodes lists the AWS service codes whose quotas we retrieve.
var awsQuotaServiceCodes = []string{
	"ec2",
	"vpc",
	"elasticloadbalancing",
	"ebs",
	"eks",
	"s3",
}

// awsQuotaResourceTypeMap maps AWS quota name prefixes to CB-Spider ResourceType.
// Keys must match the beginning of the AWS QuotaName string.
var awsQuotaResourceTypeMap = map[string]string{
	// EC2 - vCPU quotas
	"Running On-Demand Standard":                   "vCPU",
	"Running On-Demand F":                          "vCPU-F",
	"Running On-Demand G":                          "vCPU-G",
	"Running On-Demand P":                          "vCPU-P",
	// EC2 - other
	"EC2-VPC Elastic IPs": "PublicIP",
	"Network interfaces per Region": "NIC",
	// VPC
	"VPCs per Region":                              "VPC",
	"Subnets per VPC":                              "Subnet",
	"Security groups per VPC":                      "SecurityGroup",
	"VPC security groups per Region":               "SecurityGroup-Region",
	"Inbound or outbound rules per security group": "SecurityGroupRule",
	// ELB
	"Network Load Balancers per Region":             "NLB",
	"Application Load Balancers per Region":         "ALB",
	// EBS
	"Storage for General Purpose SSD (gp2) volumes": "Disk-GP2-TB",
	"Storage for General Purpose SSD (gp3) volumes": "Disk-GP3-TB",
	"Number of EBS volumes":                         "Disk",
	"Snapshots per Region":                          "Snapshot",
	// KeyPair
	"Key pairs per Region":                          "KeyPair",
	// EKS
	"Clusters":                                      "Cluster",
	// S3
	"Buckets":                                       "S3",
}

// awsQuotaDescriptionExtra provides additional GPU/accelerator info appended to Description.
var awsQuotaDescriptionExtra = map[string]string{
	"vCPU-G": "GPU instances (e.g. g4dn=T4, g5=A10G)",
	"vCPU-P": "GPU instances (e.g. p4d=A100, p5=H100)",
	"vCPU-F": "FPGA instances (e.g. f1)",
}

func (handler *AwsQuotaHandler) GetQuota() (irs.QuotaInfo, error) {
	cblogger.Info("AWS Driver: called GetQuota()")

	quotaInfo := irs.QuotaInfo{
		CSP:    "AWS",
		Region: handler.Region.Region,
	}

	var resourceQuotas []irs.ResourceQuota

	for _, serviceCode := range awsQuotaServiceCodes {
		input := &servicequotas.ListServiceQuotasInput{
			ServiceCode: aws.String(serviceCode),
			MaxResults:  aws.Int64(100),
		}
		for {
			result, err := handler.Client.ListServiceQuotas(input)
			if err != nil {
				cblogger.Warnf("AWS GetQuota: failed to list quotas for service %s: %v", serviceCode, err)
				break
			}
			for _, q := range result.Quotas {
				resourceType := matchAwsQuotaResourceType(aws.StringValue(q.QuotaName))
				if resourceType == "" {
					continue
				}
				limit := "NA"
				if q.Value != nil {
					limit = strconv.FormatFloat(aws.Float64Value(q.Value), 'f', -1, 64)
				}
				unit := "count"
				if q.Unit != nil {
					unit = aws.StringValue(q.Unit)
				}

				used := "NA"
				available := "NA"
				if handler.CwClient != nil && q.UsageMetric != nil {
					if u, err := getQuotaUsageFromCloudWatch(handler.CwClient, q.UsageMetric); err == nil && u != "NA" {
						used = u
						if limit != "NA" {
							limitF, e1 := strconv.ParseFloat(limit, 64)
							usedF, e2 := strconv.ParseFloat(used, 64)
							if e1 == nil && e2 == nil {
								available = strconv.FormatFloat(limitF-usedF, 'f', -1, 64)
							}
						}
					}
				}

				desc := fmt.Sprintf("[%s] %s", serviceCode, aws.StringValue(q.QuotaName))
				if extra, ok := awsQuotaDescriptionExtra[resourceType]; ok {
					desc = fmt.Sprintf("%s | %s", desc, extra)
				}
				rq := irs.ResourceQuota{
					ResourceType: resourceType,
					Limit:        limit,
					Used:         used,
					Available:    available,
					Unit:         unit,
					Description:  desc,
				}
				resourceQuotas = append(resourceQuotas, rq)
			}
			if result.NextToken == nil {
				break
			}
			input.NextToken = result.NextToken
		}
	}

	quotaInfo.ResourceQuotas = resourceQuotas
	return quotaInfo, nil
}

// getQuotaUsageFromCloudWatch retrieves the current usage value from the
// AWS/Usage CloudWatch namespace using the UsageMetric embedded in the quota.
func getQuotaUsageFromCloudWatch(cwClient *cloudwatch.CloudWatch, usageMetric *servicequotas.MetricInfo) (string, error) {
	if usageMetric == nil {
		return "NA", nil
	}

	now := time.Now()
	startTime := now.Add(-3 * time.Hour)

	var dimensions []*cloudwatch.Dimension
	for k, v := range usageMetric.MetricDimensions {
		dimensions = append(dimensions, &cloudwatch.Dimension{
			Name:  aws.String(k),
			Value: v,
		})
	}

	stat := "Maximum"
	if usageMetric.MetricStatisticRecommendation != nil {
		stat = aws.StringValue(usageMetric.MetricStatisticRecommendation)
	}

	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  usageMetric.MetricNamespace,
		MetricName: usageMetric.MetricName,
		Dimensions: dimensions,
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(now),
		Period:     aws.Int64(3600),
		Statistics: []*string{aws.String(stat)},
	}

	result, err := cwClient.GetMetricStatistics(input)
	if err != nil {
		return "NA", err
	}
	if len(result.Datapoints) == 0 {
		return "NA", nil
	}

	// Pick the most recent datapoint
	var latest *cloudwatch.Datapoint
	for _, dp := range result.Datapoints {
		if latest == nil || dp.Timestamp.After(*latest.Timestamp) {
			latest = dp
		}
	}

	var value float64
	switch stat {
	case "Maximum":
		if latest.Maximum != nil {
			value = aws.Float64Value(latest.Maximum)
		}
	case "Average":
		if latest.Average != nil {
			value = aws.Float64Value(latest.Average)
		}
	case "Sum":
		if latest.Sum != nil {
			value = aws.Float64Value(latest.Sum)
		}
	default:
		if latest.Maximum != nil {
			value = aws.Float64Value(latest.Maximum)
		}
	}

	return strconv.FormatFloat(value, 'f', -1, 64), nil
}

func matchAwsQuotaResourceType(quotaName string) string {
	for pattern, resourceType := range awsQuotaResourceTypeMap {
		if len(quotaName) >= len(pattern) && quotaName[:len(pattern)] == pattern {
			return resourceType
		}
	}
	return ""
}
