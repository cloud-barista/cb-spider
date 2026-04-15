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
	compute "google.golang.org/api/compute/v1"
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
	VMClient        *compute.Service
	ContainerClient *container.Service
}

// gcpMetricSpec describes how to query a single MetricType from GCP Cloud Monitoring.
type gcpMetricSpec struct {
	MetricType    string                           // GCP metric type identifier
	Unit          string                           // human-readable unit returned in MetricData.MetricUnit
	Aligner       monitoringpb.Aggregation_Aligner // per-series alignment (mean for gauges, rate for counters)
	RequiresAgent bool                             // true if metric is sourced from Ops Agent (agent.googleapis.com/*)
}

// gcpMetricMap maps CB-Spider MetricType enum to GCP Cloud Monitoring metric specs.
// Metrics under agent.googleapis.com/* require the GCP Ops Agent installed on the target VM.
var gcpMetricMap = map[irs.MetricType]gcpMetricSpec{
	irs.CPUUsage:     {"compute.googleapis.com/instance/cpu/utilization", "Percent", monitoringpb.Aggregation_ALIGN_MEAN, false},
	irs.MemoryUsage:  {"agent.googleapis.com/memory/percent_used", "Percent", monitoringpb.Aggregation_ALIGN_MEAN, true},
	irs.DiskRead:     {"compute.googleapis.com/instance/disk/read_bytes_count", "Bytes", monitoringpb.Aggregation_ALIGN_RATE, false},
	irs.DiskWrite:    {"compute.googleapis.com/instance/disk/write_bytes_count", "Bytes", monitoringpb.Aggregation_ALIGN_RATE, false},
	irs.DiskReadOps:  {"compute.googleapis.com/instance/disk/read_ops_count", "Count", monitoringpb.Aggregation_ALIGN_RATE, false},
	irs.DiskWriteOps: {"compute.googleapis.com/instance/disk/write_ops_count", "Count", monitoringpb.Aggregation_ALIGN_RATE, false},
	irs.NetworkIn:    {"compute.googleapis.com/instance/network/received_bytes_count", "Bytes", monitoringpb.Aggregation_ALIGN_RATE, false},
	irs.NetworkOut:   {"compute.googleapis.com/instance/network/sent_bytes_count", "Bytes", monitoringpb.Aggregation_ALIGN_RATE, false},
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

// resolveInstance looks up the numeric Compute Engine instance ID and current
// status for a VM identified by name. Uses the zone configured on handler.Region.
//
// Returning status here lets callers diagnose empty metric results without an
// extra API call: a stopped VM produces no time series for either platform or
// agent metrics, which would otherwise be indistinguishable from a missing
// Ops Agent on a running VM.
func (handler *GCPMonitoringHandler) resolveInstance(instanceName string) (id, status string, err error) {
	zone := handler.Region.Zone
	if zone == "" {
		return "", "", errors.New("region zone is empty")
	}
	inst, err := handler.VMClient.Instances.Get(handler.Credential.ProjectID, zone, instanceName).Do()
	if err != nil {
		return "", "", fmt.Errorf("failed to get instance %q in zone %q: %w", instanceName, zone, err)
	}
	return strconv.FormatUint(inst.Id, 10), inst.Status, nil
}

// diagnoseEmptyMetric returns a descriptive error explaining why GCP Cloud
// Monitoring returned zero data points for an instance. The status is taken
// from the prior Instances.Get call, so this adds no additional API traffic.
func diagnoseEmptyMetric(metricType irs.MetricType, spec gcpMetricSpec, instanceName, status string) error {
	if status != "" && status != "RUNNING" {
		return fmt.Errorf(
			"no %s data: instance %q is in %q state (must be RUNNING to emit metrics)",
			metricType, instanceName, status)
	}
	if spec.RequiresAgent {
		return fmt.Errorf(
			"no %s data for running instance %q: this metric requires the GCP Ops Agent. "+
				"Install: https://cloud.google.com/monitoring/agent/ops-agent/install-index",
			metricType, instanceName)
	}
	return fmt.Errorf(
		"no %s data for running instance %q in the requested window: "+
			"the VM may have been started recently or the window is shorter than the metric ingest delay; "+
			"try increasing TimeBeforeHour",
		metricType, instanceName)
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
	instanceName, instanceStatus string,
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

	if len(result.TimestampValues) == 0 {
		diagErr := diagnoseEmptyMetric(metricType, spec, instanceName, instanceStatus)
		cblogger.Error(diagErr.Error())
		return irs.MetricData{}, diagErr
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

	intervalMinute, timeBeforeHour, err := parseAndValidateInterval(
		vmMonitoringReqInfo.IntervalMinute,
		vmMonitoringReqInfo.TimeBeforeHour,
	)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.MetricData{}, err
	}

	instanceName := vmMonitoringReqInfo.VMIID.NameId
	if instanceName == "" {
		instanceName = vmMonitoringReqInfo.VMIID.SystemId
	}

	instanceID, instanceStatus, err := handler.resolveInstance(instanceName)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.MetricData{}, err
	}

	resourceFilter := fmt.Sprintf(
		`resource.type="gce_instance" AND resource.labels.instance_id="%s"`,
		instanceID,
	)

	return handler.getMetricData(vmMonitoringReqInfo.MetricType, resourceFilter, intervalMinute, timeBeforeHour, instanceName, instanceStatus)
}

// GetClusterNodeMetricData fetches VM-level metrics for a GKE Standard node.
//
// Implementation treats a GKE node as its underlying GCE instance, mirroring
// the Azure AKS handler that uses VMSS VM metrics for the same purpose. This
// keeps CSP semantics consistent and lets us reuse the VM metric mapping,
// resolveInstanceID helper, and getMetricData helper.
//
// Supported:     GKE Standard clusters (nodes are user-owned GCE instances)
// NOT supported: GKE Autopilot
//   - Nodes are owned by a Google-managed project and not accessible via
//     Compute Engine APIs. Calls for Autopilot clusters fail at
//     resolveInstanceID with a permission error, propagated as-is.
//   - Autopilot's operational model is pod-based; node-level observability
//     is not meaningful for end users since they cannot resize, ssh, or
//     pin workloads to a specific node.
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

	intervalMinute, timeBeforeHour, err := parseAndValidateInterval(
		clusterMonitoringReqInfo.IntervalMinute,
		clusterMonitoringReqInfo.TimeBeforeHour,
	)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.MetricData{}, err
	}

	clusterHandler := GCPClusterHandler{
		Region:          handler.Region,
		Ctx:             handler.Ctx,
		Client:          handler.VMClient,
		ContainerClient: handler.ContainerClient,
		Credential:      handler.Credential,
	}

	cluster, err := clusterHandler.GetCluster(clusterMonitoringReqInfo.ClusterIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get cluster info. err = %s", err))
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}

	var nodeFound bool
	var instanceName string

	for _, nodeGroup := range cluster.NodeGroupList {
		if nodeGroup.IId.NameId != clusterMonitoringReqInfo.NodeGroupID.NameId &&
			nodeGroup.IId.SystemId != clusterMonitoringReqInfo.NodeGroupID.SystemId {
			continue
		}
		for _, node := range nodeGroup.Nodes {
			if node.NameId == clusterMonitoringReqInfo.NodeIID.NameId ||
				node.SystemId == clusterMonitoringReqInfo.NodeIID.SystemId {
				nodeFound = true
				instanceName = node.NameId
				if instanceName == "" {
					instanceName = node.SystemId
				}
				break
			}
		}
		if nodeFound {
			break
		}
	}

	if !nodeFound {
		getErr := errors.New("Failed to get metric data. err = Node not found from the cluster")
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}

	instanceID, instanceStatus, err := handler.resolveInstance(instanceName)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.MetricData{}, err
	}

	resourceFilter := fmt.Sprintf(
		`resource.type="gce_instance" AND resource.labels.instance_id="%s"`,
		instanceID,
	)

	return handler.getMetricData(clusterMonitoringReqInfo.MetricType, resourceFilter, intervalMinute, timeBeforeHour, instanceName, instanceStatus)
}
