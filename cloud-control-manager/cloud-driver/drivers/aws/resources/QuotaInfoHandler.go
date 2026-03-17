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

type AwsQuotaInfoHandler struct {
	Region   idrv.RegionInfo
	Client   *servicequotas.ServiceQuotas
	CwClient *cloudwatch.CloudWatch
}

// ListServiceType returns the list of AWS service codes that have quota
// information available via the Service Quotas API.
func (handler *AwsQuotaInfoHandler) ListServiceType() ([]string, error) {
	cblogger.Info("AWS Driver: called ListServiceType()")

	var serviceTypes []string
	input := &servicequotas.ListServicesInput{
		MaxResults: aws.Int64(100),
	}
	for {
		result, err := handler.Client.ListServices(input)
		if err != nil {
			return nil, fmt.Errorf("AWS ListServiceType: %w", err)
		}
		for _, svc := range result.Services {
			if svc.ServiceCode != nil {
				serviceTypes = append(serviceTypes, aws.StringValue(svc.ServiceCode))
			}
		}
		if result.NextToken == nil {
			break
		}
		input.NextToken = result.NextToken
	}
	return serviceTypes, nil
}

// GetQuotaInfo retrieves ALL quota items for the given AWS service code.
// No filtering or name-mapping is performed; CSP-original values are passed through.
func (handler *AwsQuotaInfoHandler) GetQuotaInfo(serviceType string) (irs.QuotaInfo, error) {
	cblogger.Infof("AWS Driver: called GetQuotaInfo(serviceType=%s)", serviceType)

	quotaInfo := irs.QuotaInfo{
		CSP:    "AWS",
		Region: handler.Region.Region,
	}

	var quotas []irs.Quota

	input := &servicequotas.ListServiceQuotasInput{
		ServiceCode: aws.String(serviceType),
		MaxResults:  aws.Int64(100),
	}
	for {
		result, err := handler.Client.ListServiceQuotas(input)
		if err != nil {
			return quotaInfo, fmt.Errorf("AWS GetQuotaInfo(%s): %w", serviceType, err)
		}
		for _, q := range result.Quotas {
			quotaName := aws.StringValue(q.QuotaName)
			if quotaName == "" {
				continue
			}

			limit := "NA"
			if q.Value != nil {
				limit = strconv.FormatFloat(aws.Float64Value(q.Value), 'f', -1, 64)
			}
			unit := "NA"
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

			desc := fmt.Sprintf("QuotaArn=%s", aws.StringValue(q.QuotaArn))
			rq := irs.Quota{
				QuotaName:   quotaName,
				Limit:       limit,
				Used:        used,
				Available:   available,
				Unit:        unit,
				Description: desc,
			}
			quotas = append(quotas, rq)
		}
		if result.NextToken == nil {
			break
		}
		input.NextToken = result.NextToken
	}

	quotaInfo.Quotas = quotas
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
