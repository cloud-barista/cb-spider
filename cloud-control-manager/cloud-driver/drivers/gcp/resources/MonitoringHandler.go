// GCP Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is GCP Driver.
//
// by CB-Spider Team, 2026.04.

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	container "google.golang.org/api/container/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GCPMonitoringHandler struct {
	Region          idrv.RegionInfo
	Ctx             context.Context
	Credential      idrv.CredentialInfo
	ContainerClient *container.Service
}

// gcpMetricSpec describes how to query a single MetricType from GCP Cloud Monitoring.
type gcpMetricSpec struct {
	MetricType string                           // GCP metric type identifier
	Unit       string                           // human-readable unit returned in MetricData.MetricUnit
	Aligner    monitoringpb.Aggregation_Aligner // per-series alignment (mean for gauges, rate for counters)
}

// gcpMetricMap maps CB-Spider MetricType enum to GCP Cloud Monitoring metric specs.
// Memory metric requires the GCP Ops Agent installed on the target VM.
var gcpMetricMap = map[irs.MetricType]gcpMetricSpec{
	irs.CPUUsage:     {"compute.googleapis.com/instance/cpu/utilization", "Percent", monitoringpb.Aggregation_ALIGN_MEAN},
	irs.MemoryUsage:  {"agent.googleapis.com/memory/percent_used", "Percent", monitoringpb.Aggregation_ALIGN_MEAN},
	irs.DiskRead:     {"compute.googleapis.com/instance/disk/read_bytes_count", "Bytes", monitoringpb.Aggregation_ALIGN_RATE},
	irs.DiskWrite:    {"compute.googleapis.com/instance/disk/write_bytes_count", "Bytes", monitoringpb.Aggregation_ALIGN_RATE},
	irs.DiskReadOps:  {"compute.googleapis.com/instance/disk/read_ops_count", "Count", monitoringpb.Aggregation_ALIGN_RATE},
	irs.DiskWriteOps: {"compute.googleapis.com/instance/disk/write_ops_count", "Count", monitoringpb.Aggregation_ALIGN_RATE},
	irs.NetworkIn:    {"compute.googleapis.com/instance/network/received_bytes_count", "Bytes", monitoringpb.Aggregation_ALIGN_RATE},
	irs.NetworkOut:   {"compute.googleapis.com/instance/network/sent_bytes_count", "Bytes", monitoringpb.Aggregation_ALIGN_RATE},
}

// credentialJSON builds a minimal service-account JSON from handler credentials.
func (handler *GCPMonitoringHandler) credentialJSON() []byte {
	data := map[string]string{
		"type":         "service_account",
		"private_key":  handler.Credential.PrivateKey,
		"client_email": handler.Credential.ClientEmail,
	}
	b, _ := json.Marshal(data)
	return b
}

// newMonitoringClient creates a Cloud Monitoring API v3 client using JSON credentials.
func (handler *GCPMonitoringHandler) newMonitoringClient() (*monitoring.MetricClient, error) {
	return monitoring.NewMetricClient(handler.Ctx,
		option.WithCredentialsJSON(handler.credentialJSON()),
	)
}

// parseAndValidateInterval parses the IntervalMinute and TimeBeforeHour strings
// and enforces the same constraint Azure uses: timeBeforeHour*60 >= intervalMinute.
// Empty strings default to "1".
func parseAndValidateInterval(intervalMinuteStr, timeBeforeHourStr string) (intervalMinute, timeBeforeHour int, err error) {
	if intervalMinuteStr == "" {
		intervalMinuteStr = "1"
	}
	if timeBeforeHourStr == "" {
		timeBeforeHourStr = "1"
	}

	intervalMinute, err = strconv.Atoi(intervalMinuteStr)
	if err != nil || intervalMinute <= 0 {
		return 0, 0, errors.New("invalid value of IntervalMinute")
	}

	timeBeforeHour, err = strconv.Atoi(timeBeforeHourStr)
	if err != nil || timeBeforeHour < 0 {
		return 0, 0, errors.New("invalid value of TimeBeforeHour")
	}

	if timeBeforeHour*60 < intervalMinute {
		return 0, 0, errors.New("IntervalMinute is too far in the past")
	}

	return intervalMinute, timeBeforeHour, nil
}

// getMetricData queries GCP Cloud Monitoring for a single metric and resource filter.
// `resourceFilter` is the per-resource selector appended to the metric.type filter
// (e.g. `resource.label.instance_name="my-vm"`).
func (handler *GCPMonitoringHandler) getMetricData(
	metricType irs.MetricType,
	resourceFilter string,
	intervalMinute, timeBeforeHour int,
) (irs.MetricData, error) {
	spec, ok := gcpMetricMap[metricType]
	if !ok {
		getErr := fmt.Errorf("unsupported metric type: %s", metricType)
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}

	client, err := handler.newMonitoringClient()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to create monitoring client. err = %s", err))
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}
	defer client.Close()

	endTime := time.Now().UTC()
	startTime := endTime.Add(-time.Duration(timeBeforeHour) * time.Hour)

	filter := fmt.Sprintf(`metric.type="%s" AND %s`, spec.MetricType, resourceFilter)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + handler.Credential.ProjectID,
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(startTime),
			EndTime:   timestamppb.New(endTime),
		},
		Aggregation: &monitoringpb.Aggregation{
			AlignmentPeriod:  durationpb.New(time.Duration(intervalMinute) * time.Minute),
			PerSeriesAligner: spec.Aligner,
		},
		View: monitoringpb.ListTimeSeriesRequest_FULL,
	}

	result := irs.MetricData{
		MetricName:      spec.MetricType,
		MetricUnit:      spec.Unit,
		TimestampValues: []irs.TimestampValue{},
	}

	it := client.ListTimeSeries(handler.Ctx, req)
	for {
		ts, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to get metric data. err = %s", err))
			cblogger.Error(getErr.Error())
			return irs.MetricData{}, getErr
		}

		for _, point := range ts.GetPoints() {
			value := pointValueToString(point.GetValue())
			result.TimestampValues = append(result.TimestampValues, irs.TimestampValue{
				Timestamp: point.GetInterval().GetEndTime().AsTime(),
				Value:     value,
			})
		}
	}

	return result, nil
}

// pointValueToString formats a TypedValue from GCP Monitoring into a string.
func pointValueToString(v *monitoringpb.TypedValue) string {
	if v == nil {
		return ""
	}
	switch x := v.Value.(type) {
	case *monitoringpb.TypedValue_DoubleValue:
		return strconv.FormatFloat(x.DoubleValue, 'f', -1, 64)
	case *monitoringpb.TypedValue_Int64Value:
		return strconv.FormatInt(x.Int64Value, 10)
	case *monitoringpb.TypedValue_BoolValue:
		return strconv.FormatBool(x.BoolValue)
	case *monitoringpb.TypedValue_StringValue:
		return x.StringValue
	default:
		return ""
	}
}

func (handler *GCPMonitoringHandler) GetVMMetricData(vmMonitoringReqInfo irs.VMMonitoringReqInfo) (irs.MetricData, error) {
	cblogger.Info("GCP Cloud Driver: called GetVMMetricData()")

	if handler.Credential.ProjectID == "" {
		getErr := errors.New("missing project ID in credentials")
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}
	if vmMonitoringReqInfo.VMIID.NameId == "" && vmMonitoringReqInfo.VMIID.SystemId == "" {
		getErr := errors.New("VMIID is empty")
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}

	getErr := errors.New("GCP MonitoringHandler: not implemented yet")
	cblogger.Error(getErr.Error())
	return irs.MetricData{}, getErr
}

func (handler *GCPMonitoringHandler) GetClusterNodeMetricData(clusterMonitoringReqInfo irs.ClusterNodeMonitoringReqInfo) (irs.MetricData, error) {
	cblogger.Info("GCP Cloud Driver: called GetClusterNodeMetricData()")

	if handler.Credential.ProjectID == "" {
		getErr := errors.New("missing project ID in credentials")
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}
	if clusterMonitoringReqInfo.ClusterIID.NameId == "" && clusterMonitoringReqInfo.ClusterIID.SystemId == "" {
		getErr := errors.New("ClusterIID is empty")
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}

	getErr := errors.New("GCP MonitoringHandler: not implemented yet")
	cblogger.Error(getErr.Error())
	return irs.MetricData{}, getErr
}
