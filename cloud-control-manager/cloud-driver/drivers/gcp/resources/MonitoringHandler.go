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

// ============================================================================
// Types
// ============================================================================

type GCPMonitoringHandler struct {
	Region          idrv.RegionInfo
	Ctx             context.Context
	Credential      idrv.CredentialInfo
	VMClient        *compute.Service
	ContainerClient *container.Service
}

// gcpMetricSpec describes how to query a single GCP Cloud Monitoring metric.
// ALIGN_DELTA is used for cumulative byte counters so each point is
// "bytes in the period". ValueScale=0 is treated as 1.0.
type gcpMetricSpec struct {
	MetricType  string
	Unit        string
	Aligner     monitoringpb.Aggregation_Aligner
	ExtraFilter string // AND-appended metric-label selector
	ValueScale  float64
}

// monitoringTarget is the resolved per-request selector passed to
// getMetricData. filter is the resource.type + labels clause appended after
// metric.type; name and status are used only for empty-result diagnostics.
type monitoringTarget struct {
	filter string
	name   string
	status string
}

type vmQueryContext struct {
	instanceName   string
	instanceID     string
	instanceStatus string
	intervalMinute int
	timeBeforeHour int
}

func (c vmQueryContext) gceTarget() monitoringTarget {
	return monitoringTarget{
		filter: fmt.Sprintf(`resource.type="gce_instance" AND resource.labels.instance_id="%s"`, c.instanceID),
		name:   c.instanceName,
		status: c.instanceStatus,
	}
}

type clusterNodeQueryContext struct {
	vmQueryContext
	clusterName string
	location    string
	vmSpecName  string // machine type (e.g. "e2-small")
}

// k8sTarget assumes the GKE default where the Kubernetes node name equals
// the underlying GCE instance name. The VM status is reused as the target
// status so empty-result diagnostics can still say "node VM stopped".
func (c clusterNodeQueryContext) k8sTarget() monitoringTarget {
	return monitoringTarget{
		filter: fmt.Sprintf(
			`resource.type="k8s_node" AND resource.labels.cluster_name="%s" AND resource.labels.node_name="%s" AND resource.labels.location="%s"`,
			c.clusterName, c.instanceName, c.location,
		),
		name:   c.instanceName,
		status: c.instanceStatus,
	}
}

type vmMetricHandler func(*GCPMonitoringHandler, vmQueryContext) (irs.MetricData, error)
type gkeMetricHandler func(*GCPMonitoringHandler, clusterNodeQueryContext) (irs.MetricData, error)

// ============================================================================
// Metric Specs
// ============================================================================

var (
	specCPU = gcpMetricSpec{
		MetricType: "compute.googleapis.com/instance/cpu/utilization",
		Unit:       "Percent",
		Aligner:    monitoringpb.Aggregation_ALIGN_MEAN,
		ValueScale: 100.0,
	}
	specDiskRead = gcpMetricSpec{
		MetricType: "compute.googleapis.com/instance/disk/read_bytes_count",
		Unit:       "Bytes",
		Aligner:    monitoringpb.Aggregation_ALIGN_DELTA,
		ValueScale: 1.0,
	}
	specDiskWrite = gcpMetricSpec{
		MetricType: "compute.googleapis.com/instance/disk/write_bytes_count",
		Unit:       "Bytes",
		Aligner:    monitoringpb.Aggregation_ALIGN_DELTA,
		ValueScale: 1.0,
	}
	specDiskReadOps = gcpMetricSpec{
		MetricType: "compute.googleapis.com/instance/disk/read_ops_count",
		Unit:       "CountPerSecond",
		Aligner:    monitoringpb.Aggregation_ALIGN_RATE,
		ValueScale: 1.0,
	}
	specDiskWriteOps = gcpMetricSpec{
		MetricType: "compute.googleapis.com/instance/disk/write_ops_count",
		Unit:       "CountPerSecond",
		Aligner:    monitoringpb.Aggregation_ALIGN_RATE,
		ValueScale: 1.0,
	}
	specNetworkIn = gcpMetricSpec{
		MetricType: "compute.googleapis.com/instance/network/received_bytes_count",
		Unit:       "Bytes",
		Aligner:    monitoringpb.Aggregation_ALIGN_DELTA,
		ValueScale: 1.0,
	}
	specNetworkOut = gcpMetricSpec{
		MetricType: "compute.googleapis.com/instance/network/sent_bytes_count",
		Unit:       "Bytes",
		Aligner:    monitoringpb.Aggregation_ALIGN_DELTA,
		ValueScale: 1.0,
	}

	// GKE memory percent uses used_bytes from Cloud Monitoring; the
	// denominator comes from the VMSpec, not a second metric.
	specGKEMemoryUsedBytes = gcpMetricSpec{
		MetricType:  "kubernetes.io/node/memory/used_bytes",
		Unit:        "Bytes",
		Aligner:     monitoringpb.Aggregation_ALIGN_MEAN,
		ExtraFilter: `metric.labels.memory_type="non-evictable"`,
		ValueScale:  1.0,
	}
)

// ============================================================================
// Handler Registry
// ============================================================================

var gcpVMMetricHandlers = map[irs.MetricType]vmMetricHandler{
	irs.CPUUsage: vmDirect(irs.CPUUsage, specCPU),
	irs.MemoryUsage: vmRejected(
		"memory_usage is not supported for GCP VMs in API-based(agentless) monitoring.",
	),
	irs.DiskRead:     vmDirect(irs.DiskRead, specDiskRead),
	irs.DiskWrite:    vmDirect(irs.DiskWrite, specDiskWrite),
	irs.DiskReadOps:  vmDirect(irs.DiskReadOps, specDiskReadOps),
	irs.DiskWriteOps: vmDirect(irs.DiskWriteOps, specDiskWriteOps),
	irs.NetworkIn:    vmDirect(irs.NetworkIn, specNetworkIn),
	irs.NetworkOut:   vmDirect(irs.NetworkOut, specNetworkOut),
}

var gcpGKEMetricHandlers = map[irs.MetricType]gkeMetricHandler{
	irs.CPUUsage:     gkeDirect(irs.CPUUsage, specCPU),
	irs.MemoryUsage:  gkeMemoryPercent(),
	irs.DiskRead:     gkeDirect(irs.DiskRead, specDiskRead),
	irs.DiskWrite:    gkeDirect(irs.DiskWrite, specDiskWrite),
	irs.DiskReadOps:  gkeDirect(irs.DiskReadOps, specDiskReadOps),
	irs.DiskWriteOps: gkeDirect(irs.DiskWriteOps, specDiskWriteOps),
	irs.NetworkIn:    gkeDirect(irs.NetworkIn, specNetworkIn),
	irs.NetworkOut:   gkeDirect(irs.NetworkOut, specNetworkOut),
}

func vmDirect(mt irs.MetricType, spec gcpMetricSpec) vmMetricHandler {
	return func(h *GCPMonitoringHandler, ctx vmQueryContext) (irs.MetricData, error) {
		return h.getMetricData(mt, spec, ctx.gceTarget(), ctx.intervalMinute, ctx.timeBeforeHour)
	}
}

// vmRejected returns a handler that fails fast with a fixed reason. Used for
// MetricTypes present in the common interface but intentionally unsupported
// on GCE VMs, so callers see why instead of "unsupported VM metric type".
func vmRejected(reason string) vmMetricHandler {
	return func(_ *GCPMonitoringHandler, _ vmQueryContext) (irs.MetricData, error) {
		err := errors.New(reason)
		cblogger.Error(err.Error())
		return irs.MetricData{}, err
	}
}

func gkeDirect(mt irs.MetricType, spec gcpMetricSpec) gkeMetricHandler {
	return func(h *GCPMonitoringHandler, ctx clusterNodeQueryContext) (irs.MetricData, error) {
		return h.getMetricData(mt, spec, ctx.gceTarget(), ctx.intervalMinute, ctx.timeBeforeHour)
	}
}

func gkeMemoryPercent() gkeMetricHandler {
	return func(h *GCPMonitoringHandler, ctx clusterNodeQueryContext) (irs.MetricData, error) {
		return h.computeGKEMemoryPercent(ctx.k8sTarget(), ctx.vmSpecName, ctx.intervalMinute, ctx.timeBeforeHour)
	}
}

// ============================================================================
// Public API
// ============================================================================

func (handler *GCPMonitoringHandler) GetVMMetricData(req irs.VMMonitoringReqInfo) (irs.MetricData, error) {
	cblogger.Info("GCP Cloud Driver: called GetVMMetricData()")

	if handler.Credential.ProjectID == "" {
		getErr := errors.New("missing project ID in credentials")
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}
	if req.VMIID.NameId == "" && req.VMIID.SystemId == "" {
		getErr := errors.New("VMIID is empty")
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}

	intervalMinute, timeBeforeHour, err := parseAndValidateInterval(req.IntervalMinute, req.TimeBeforeHour)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.MetricData{}, err
	}

	instanceName := req.VMIID.NameId
	if instanceName == "" {
		instanceName = req.VMIID.SystemId
	}

	instanceID, instanceStatus, err := handler.resolveInstance(instanceName)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.MetricData{}, err
	}

	handle, ok := gcpVMMetricHandlers[req.MetricType]
	if !ok {
		getErr := fmt.Errorf("unsupported VM metric type: %s", req.MetricType)
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}

	ctx := vmQueryContext{
		instanceName:   instanceName,
		instanceID:     instanceID,
		instanceStatus: instanceStatus,
		intervalMinute: intervalMinute,
		timeBeforeHour: timeBeforeHour,
	}
	return handle(handler, ctx)
}

// GetClusterNodeMetricData fetches node-level metrics for a GKE Standard node.
// GKE Autopilot is not supported: its nodes live in a Google-managed project
// and Instances.Get fails with a permission error, which is propagated as-is.
func (handler *GCPMonitoringHandler) GetClusterNodeMetricData(req irs.ClusterNodeMonitoringReqInfo) (irs.MetricData, error) {
	cblogger.Info("GCP Cloud Driver: called GetClusterNodeMetricData()")

	if handler.Credential.ProjectID == "" {
		getErr := errors.New("missing project ID in credentials")
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}
	if req.ClusterIID.NameId == "" && req.ClusterIID.SystemId == "" {
		getErr := errors.New("ClusterIID is empty")
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}

	intervalMinute, timeBeforeHour, err := parseAndValidateInterval(req.IntervalMinute, req.TimeBeforeHour)
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
	cluster, err := clusterHandler.GetCluster(req.ClusterIID)
	if err != nil {
		getErr := fmt.Errorf("failed to get cluster info: %w", err)
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}

	instanceName, vmSpecName, err := findClusterNodeInstance(cluster, req.NodeGroupID, req.NodeIID)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.MetricData{}, err
	}

	instanceID, instanceStatus, err := handler.resolveInstance(instanceName)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.MetricData{}, err
	}

	handle, ok := gcpGKEMetricHandlers[req.MetricType]
	if !ok {
		getErr := fmt.Errorf("unsupported cluster-node metric type: %s", req.MetricType)
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}

	clusterName := req.ClusterIID.NameId
	if clusterName == "" {
		clusterName = req.ClusterIID.SystemId
	}
	ctx := clusterNodeQueryContext{
		vmQueryContext: vmQueryContext{
			instanceName:   instanceName,
			instanceID:     instanceID,
			instanceStatus: instanceStatus,
			intervalMinute: intervalMinute,
			timeBeforeHour: timeBeforeHour,
		},
		clusterName: clusterName,
		location:    handler.Region.Zone,
		vmSpecName:  vmSpecName,
	}
	return handle(handler, ctx)
}

func findClusterNodeInstance(cluster irs.ClusterInfo, nodeGroupID, nodeIID irs.IID) (instanceName, vmSpecName string, err error) {
	for _, nodeGroup := range cluster.NodeGroupList {
		if nodeGroup.IId.NameId != nodeGroupID.NameId &&
			nodeGroup.IId.SystemId != nodeGroupID.SystemId {
			continue
		}
		for _, node := range nodeGroup.Nodes {
			if node.NameId != nodeIID.NameId && node.SystemId != nodeIID.SystemId {
				continue
			}
			name := node.NameId
			if name == "" {
				name = node.SystemId
			}
			return name, nodeGroup.VMSpecName, nil
		}
	}
	return "", "", errors.New("node not found in the cluster")
}

// ============================================================================
// Metric Execution
// ============================================================================

func (handler *GCPMonitoringHandler) getMetricData(
	metricType irs.MetricType,
	spec gcpMetricSpec,
	target monitoringTarget,
	intervalMinute, timeBeforeHour int,
) (irs.MetricData, error) {
	client, err := handler.newMonitoringClient()
	if err != nil {
		getErr := fmt.Errorf("failed to create monitoring client: %w", err)
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}
	defer client.Close()

	endTime := time.Now().UTC()
	startTime := endTime.Add(-time.Duration(timeBeforeHour) * time.Hour)

	filter := fmt.Sprintf(`metric.type="%s" AND %s`, spec.MetricType, target.filter)
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
			getErr := fmt.Errorf("failed to get metric data: %w", err)
			cblogger.Error(getErr.Error())
			return irs.MetricData{}, getErr
		}
		for _, point := range ts.GetPoints() {
			result.TimestampValues = append(result.TimestampValues, irs.TimestampValue{
				Timestamp: point.GetInterval().GetEndTime().AsTime(),
				Value:     pointValueToString(point.GetValue(), spec.ValueScale),
			})
		}
	}

	if len(result.TimestampValues) == 0 {
		diagErr := diagnoseEmptyMetric(metricType, target.name, target.status)
		cblogger.Error(diagErr.Error())
		return irs.MetricData{}, diagErr
	}

	return result, nil
}

// computeGKEMemoryPercent returns used memory as a percent of the
// machine type's total RAM.
func (handler *GCPMonitoringHandler) computeGKEMemoryPercent(
	target monitoringTarget,
	vmSpecName string,
	intervalMinute, timeBeforeHour int,
) (irs.MetricData, error) {
	totalBytes, err := handler.getVMSpecMemoryBytes(vmSpecName)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.MetricData{}, err
	}
	if totalBytes <= 0 {
		getErr := fmt.Errorf("VMSpec %q reports non-positive memory (%d bytes)", vmSpecName, totalBytes)
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}

	used, err := handler.getMetricData(irs.MemoryUsage, specGKEMemoryUsedBytes, target, intervalMinute, timeBeforeHour)
	if err != nil {
		return irs.MetricData{}, err
	}

	totalBytesF := float64(totalBytes)
	result := irs.MetricData{
		MetricName:      "kubernetes.io/node/memory/used_percent",
		MetricUnit:      "Percent",
		TimestampValues: make([]irs.TimestampValue, 0, len(used.TimestampValues)),
	}
	for _, tv := range used.TimestampValues {
		usedVal, parseErr := strconv.ParseFloat(tv.Value, 64)
		if parseErr != nil {
			continue
		}
		result.TimestampValues = append(result.TimestampValues, irs.TimestampValue{
			Timestamp: tv.Timestamp,
			Value:     strconv.FormatFloat(usedVal/totalBytesF*100, 'f', -1, 64),
		})
	}
	return result, nil
}

// ============================================================================
// Low-level Helpers
// ============================================================================

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

func (handler *GCPMonitoringHandler) getVMSpecMemoryBytes(vmSpecName string) (int64, error) {
	if vmSpecName == "" {
		return 0, errors.New("vmSpecName is empty")
	}
	vmSpecHandler := &GCPVMSpecHandler{
		Region:     handler.Region,
		Ctx:        handler.Ctx,
		Credential: handler.Credential,
		Client:     handler.VMClient,
	}
	spec, err := vmSpecHandler.GetVMSpec(vmSpecName)
	if err != nil {
		return 0, fmt.Errorf("failed to get VMSpec %q: %w", vmSpecName, err)
	}
	memMiB, err := strconv.ParseInt(spec.MemSizeMiB, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse VMSpec %q MemSizeMiB %q: %w", vmSpecName, spec.MemSizeMiB, err)
	}
	return memMiB * 1024 * 1024, nil
}

// resolveInstance fetches the numeric instance ID and current status. The
// status is surfaced for empty-result diagnostics to distinguish a stopped
// VM from a metric-ingest delay without a second API call.
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

func diagnoseEmptyMetric(metricType irs.MetricType, instanceName, status string) error {
	if status != "" && status != "RUNNING" {
		return fmt.Errorf(
			"no %s data: instance %q is in %q state (must be RUNNING to emit metrics)",
			metricType, instanceName, status)
	}
	return fmt.Errorf(
		"no %s data for running instance %q in the requested window: "+
			"the VM may have been started recently or the window is shorter than the metric ingest delay; "+
			"try increasing TimeBeforeHour",
		metricType, instanceName)
}
