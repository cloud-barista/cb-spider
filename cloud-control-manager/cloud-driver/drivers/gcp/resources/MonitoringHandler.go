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
	"strings"
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

// ============================================================================
// Types & Specs
// ============================================================================

type GCPMonitoringHandler struct {
	Region          idrv.RegionInfo
	Ctx             context.Context
	Credential      idrv.CredentialInfo
	VMClient        *compute.Service
	ContainerClient *container.Service
}

// gcpMetricSpec describes how to query a single MetricType from GCP Cloud Monitoring.
type gcpMetricSpec struct {
	MetricType string
	Unit       string

	// Aligner matches Azure semantics: ALIGN_MEAN for gauges, ALIGN_DELTA for
	// cumulative byte counters (bytes-in-period, like Azure Total), ALIGN_RATE
	// for /sec counters.
	Aligner monitoringpb.Aggregation_Aligner

	// RequiresAgent marks metrics sourced from Ops Agent (agent.googleapis.com/*).
	// Used by diagnoseEmptyMetric to hint at agent installation on empty results.
	RequiresAgent bool

	// ExtraFilter appends a metric-label selector via AND
	// (e.g. `metric.labels.state="used"` on agent memory).
	ExtraFilter string

	// ValueScale post-multiplies each point. CPU uses 100 to convert GCP's
	// 0-1 fraction to Azure's 0-100 scale. Zero is treated as 1.0.
	ValueScale float64
}

// gcpVMMetricMap: specs for standalone VMs. Memory requires Ops Agent.
var gcpVMMetricMap = map[irs.MetricType]gcpMetricSpec{
	irs.CPUUsage:     {"compute.googleapis.com/instance/cpu/utilization", "Percent", monitoringpb.Aggregation_ALIGN_MEAN, false, "", 100.0},
	irs.MemoryUsage:  {"agent.googleapis.com/memory/bytes_used", "Bytes", monitoringpb.Aggregation_ALIGN_MEAN, true, `metric.labels.state="used"`, 1.0},
	irs.DiskRead:     {"compute.googleapis.com/instance/disk/read_bytes_count", "Bytes", monitoringpb.Aggregation_ALIGN_DELTA, false, "", 1.0},
	irs.DiskWrite:    {"compute.googleapis.com/instance/disk/write_bytes_count", "Bytes", monitoringpb.Aggregation_ALIGN_DELTA, false, "", 1.0},
	irs.DiskReadOps:  {"compute.googleapis.com/instance/disk/read_ops_count", "CountPerSecond", monitoringpb.Aggregation_ALIGN_RATE, false, "", 1.0},
	irs.DiskWriteOps: {"compute.googleapis.com/instance/disk/write_ops_count", "CountPerSecond", monitoringpb.Aggregation_ALIGN_RATE, false, "", 1.0},
	irs.NetworkIn:    {"compute.googleapis.com/instance/network/received_bytes_count", "Bytes", monitoringpb.Aggregation_ALIGN_DELTA, false, "", 1.0},
	irs.NetworkOut:   {"compute.googleapis.com/instance/network/sent_bytes_count", "Bytes", monitoringpb.Aggregation_ALIGN_DELTA, false, "", 1.0},
}

// gcpGKEMetricOverrides: GKE-specific specs that deviate from gcpVMMetricMap.
// Non-overridden metrics fall back to the VM map (GKE Standard nodes are GCE
// instances). Memory uses kubernetes.io/* so no Ops Agent is needed; the
// memory_type="non-evictable" filter approximates "in-use" memory.
var gcpGKEMetricOverrides = map[irs.MetricType]gcpMetricSpec{
	irs.MemoryUsage: {"kubernetes.io/node/memory/used_bytes", "Bytes", monitoringpb.Aggregation_ALIGN_MEAN, false, `metric.labels.memory_type="non-evictable"`, 1.0},
}

// monitoringTarget lets getMetricData stay agnostic of the underlying
// resource.type (gce_instance vs k8s_node). Each implementation builds its
// own filter selector and exposes a display name plus VM status for
// empty-result diagnostics.
type monitoringTarget interface {
	filter() string
	name() string
	status() string
}

type gceInstanceTarget struct {
	Name   string
	ID     string
	Status string
}

func (t gceInstanceTarget) filter() string {
	return fmt.Sprintf(`resource.type="gce_instance" AND resource.labels.instance_id="%s"`, t.ID)
}
func (t gceInstanceTarget) name() string   { return t.Name }
func (t gceInstanceTarget) status() string { return t.Status }

// k8sNodeTarget reuses the underlying VM status so diagnostics can still
// distinguish "node VM stopped" from "metric not yet reporting" without an
// extra API call.
type k8sNodeTarget struct {
	ClusterName        string
	NodeName           string
	Location           string
	UnderlyingVMStatus string
}

func (t k8sNodeTarget) filter() string {
	return fmt.Sprintf(
		`resource.type="k8s_node" AND resource.labels.cluster_name="%s" AND resource.labels.node_name="%s" AND resource.labels.location="%s"`,
		t.ClusterName, t.NodeName, t.Location,
	)
}
func (t k8sNodeTarget) name() string   { return t.NodeName }
func (t k8sNodeTarget) status() string { return t.UnderlyingVMStatus }

// ============================================================================
// Public API
// ============================================================================

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

	spec, ok := gcpVMMetricMap[vmMonitoringReqInfo.MetricType]
	if !ok {
		getErr := fmt.Errorf("unsupported VM metric type: %s", vmMonitoringReqInfo.MetricType)
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}

	instanceID, instanceStatus, err := handler.resolveInstance(instanceName)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.MetricData{}, err
	}

	target := gceInstanceTarget{
		Name:   instanceName,
		ID:     instanceID,
		Status: instanceStatus,
	}

	return handler.getMetricData(vmMonitoringReqInfo.MetricType, spec, target, intervalMinute, timeBeforeHour)
}

// GetClusterNodeMetricData fetches node-level metrics for a GKE Standard node.
// Non-memory metrics query compute.googleapis.com on the underlying GCE
// instance; Memory uses kubernetes.io/node/memory/used_bytes so no Ops Agent
// install is required. GKE Autopilot is not supported: Autopilot nodes live
// in a Google-managed project and Instances.Get fails with a permission
// error, which is propagated as-is.
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

	spec, ok := resolveGKEMetricSpec(clusterMonitoringReqInfo.MetricType)
	if !ok {
		getErr := fmt.Errorf("unsupported cluster-node metric type: %s", clusterMonitoringReqInfo.MetricType)
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}

	// Always resolve: gce_instance needs the ID, k8s_node still uses the
	// VM status for empty-result diagnostics. Same single API call either way.
	instanceID, instanceStatus, err := handler.resolveInstance(instanceName)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.MetricData{}, err
	}

	var target monitoringTarget
	if strings.HasPrefix(spec.MetricType, "kubernetes.io/") {
		clusterName := clusterMonitoringReqInfo.ClusterIID.NameId
		if clusterName == "" {
			clusterName = clusterMonitoringReqInfo.ClusterIID.SystemId
		}
		// GKE default: k8s node name == GCE instance name.
		target = k8sNodeTarget{
			ClusterName:        clusterName,
			NodeName:           instanceName,
			Location:           handler.Region.Zone,
			UnderlyingVMStatus: instanceStatus,
		}
	} else {
		target = gceInstanceTarget{
			Name:   instanceName,
			ID:     instanceID,
			Status: instanceStatus,
		}
	}

	return handler.getMetricData(clusterMonitoringReqInfo.MetricType, spec, target, intervalMinute, timeBeforeHour)
}

// ============================================================================
// Helpers
// ============================================================================

func resolveGKEMetricSpec(mt irs.MetricType) (gcpMetricSpec, bool) {
	if spec, ok := gcpGKEMetricOverrides[mt]; ok {
		return spec, true
	}
	spec, ok := gcpVMMetricMap[mt]
	return spec, ok
}

func (handler *GCPMonitoringHandler) credentialJSON() []byte {
	data := map[string]string{
		"type":         "service_account",
		"private_key":  handler.Credential.PrivateKey,
		"client_email": handler.Credential.ClientEmail,
	}
	b, _ := json.Marshal(data)
	return b
}

func (handler *GCPMonitoringHandler) newMonitoringClient() (*monitoring.MetricClient, error) {
	return monitoring.NewMetricClient(handler.Ctx,
		option.WithCredentialsJSON(handler.credentialJSON()),
	)
}

// resolveInstance returns the numeric instance ID and status from a single
// Instances.Get call. Status is exposed so empty-result diagnostics can tell
// "VM stopped" from "Ops Agent missing" without an extra API call.
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

// parseAndValidateInterval enforces the same constraint as the Azure handler:
// timeBeforeHour*60 >= intervalMinute. Empty strings default to "1".
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

func (handler *GCPMonitoringHandler) getMetricData(
	metricType irs.MetricType,
	spec gcpMetricSpec,
	target monitoringTarget,
	intervalMinute, timeBeforeHour int,
) (irs.MetricData, error) {
	client, err := handler.newMonitoringClient()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to create monitoring client. err = %s", err))
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}
	defer client.Close()

	endTime := time.Now().UTC()
	startTime := endTime.Add(-time.Duration(timeBeforeHour) * time.Hour)

	filter := fmt.Sprintf(`metric.type="%s" AND %s`, spec.MetricType, target.filter())
	if spec.ExtraFilter != "" {
		filter = fmt.Sprintf(`%s AND %s`, filter, spec.ExtraFilter)
	}

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
			value := pointValueToString(point.GetValue(), spec.ValueScale)
			result.TimestampValues = append(result.TimestampValues, irs.TimestampValue{
				Timestamp: point.GetInterval().GetEndTime().AsTime(),
				Value:     value,
			})
		}
	}

	if len(result.TimestampValues) == 0 {
		diagErr := diagnoseEmptyMetric(metricType, spec, target.name(), target.status())
		cblogger.Error(diagErr.Error())
		return irs.MetricData{}, diagErr
	}

	return result, nil
}

// pointValueToString formats a TypedValue and applies ValueScale. Scale 0
// is treated as 1.0 so unset specs pass values through unchanged.
func pointValueToString(v *monitoringpb.TypedValue, scale float64) string {
	if v == nil {
		return ""
	}
	if scale == 0 {
		scale = 1.0
	}
	switch x := v.Value.(type) {
	case *monitoringpb.TypedValue_DoubleValue:
		return strconv.FormatFloat(x.DoubleValue*scale, 'f', -1, 64)
	case *monitoringpb.TypedValue_Int64Value:
		if scale == 1.0 {
			return strconv.FormatInt(x.Int64Value, 10)
		}
		return strconv.FormatFloat(float64(x.Int64Value)*scale, 'f', -1, 64)
	case *monitoringpb.TypedValue_BoolValue:
		return strconv.FormatBool(x.BoolValue)
	case *monitoringpb.TypedValue_StringValue:
		return x.StringValue
	default:
		return ""
	}
}

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
