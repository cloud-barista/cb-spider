// AWS Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2026.04.

package resources

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// ============================================================================
// Types
// ============================================================================

type AwsMonitoringHandler struct {
	Region         idrv.RegionInfo
	CredentialInfo idrv.CredentialInfo
	CwClient       *cloudwatch.CloudWatch
	VMClient       *ec2.EC2
	ClusterHandler *AwsClusterHandler
}

// awsMetricSpec describes how to query a single CloudWatch metric.
// PerSecond divides the aggregated value by the period (seconds) to turn a
// Sum-of-counts into a per-second rate.
type awsMetricSpec struct {
	Namespace  string
	MetricName string
	Statistic  string // "Average", "Sum"
	PerSecond  bool
	Unit       string // informational only; the common runtime rewrites the response unit
}

// monitoringQuery is the resolved per-request input for getMetricData.
type monitoringQuery struct {
	instanceID     string
	instanceType   string // e.g. "t3.small" — drives instance-store vs EBS metric selection
	instanceState  string
	intervalMinute int
	timeBeforeHour int
}

// ============================================================================
// Metric Specs
// ============================================================================

var (
	awsSpecCPU = awsMetricSpec{
		Namespace:  "AWS/EC2",
		MetricName: "CPUUtilization",
		Statistic:  "Average",
		Unit:       "Percent",
	}
	// Disk*: instance-store volume I/O (legacy/Xen-era and i3/m4d/c5d-style
	// instances that ship with NVMe instance store). Reports 0 / not-published
	// on EBS-only instances.
	awsSpecDiskRead = awsMetricSpec{
		Namespace:  "AWS/EC2",
		MetricName: "DiskReadBytes",
		Statistic:  "Sum",
		Unit:       "Bytes",
	}
	awsSpecDiskWrite = awsMetricSpec{
		Namespace:  "AWS/EC2",
		MetricName: "DiskWriteBytes",
		Statistic:  "Sum",
		Unit:       "Bytes",
	}
	awsSpecDiskReadOps = awsMetricSpec{
		Namespace:  "AWS/EC2",
		MetricName: "DiskReadOps",
		Statistic:  "Sum",
		PerSecond:  true,
		Unit:       "CountPerSecond",
	}
	awsSpecDiskWriteOps = awsMetricSpec{
		Namespace:  "AWS/EC2",
		MetricName: "DiskWriteOps",
		Statistic:  "Sum",
		PerSecond:  true,
		Unit:       "CountPerSecond",
	}
	// EBS*: EBS volume I/O on Nitro instances (most current types: t3, t4g, m5,
	// c5, c6i, …). Same statistic semantics as the Disk* group.
	awsSpecEBSRead = awsMetricSpec{
		Namespace:  "AWS/EC2",
		MetricName: "EBSReadBytes",
		Statistic:  "Sum",
		Unit:       "Bytes",
	}
	awsSpecEBSWrite = awsMetricSpec{
		Namespace:  "AWS/EC2",
		MetricName: "EBSWriteBytes",
		Statistic:  "Sum",
		Unit:       "Bytes",
	}
	awsSpecEBSReadOps = awsMetricSpec{
		Namespace:  "AWS/EC2",
		MetricName: "EBSReadOps",
		Statistic:  "Sum",
		PerSecond:  true,
		Unit:       "CountPerSecond",
	}
	awsSpecEBSWriteOps = awsMetricSpec{
		Namespace:  "AWS/EC2",
		MetricName: "EBSWriteOps",
		Statistic:  "Sum",
		PerSecond:  true,
		Unit:       "CountPerSecond",
	}
	awsSpecNetworkIn = awsMetricSpec{
		Namespace:  "AWS/EC2",
		MetricName: "NetworkIn",
		Statistic:  "Sum",
		Unit:       "Bytes",
	}
	awsSpecNetworkOut = awsMetricSpec{
		Namespace:  "AWS/EC2",
		MetricName: "NetworkOut",
		Statistic:  "Sum",
		Unit:       "Bytes",
	}
)

// ============================================================================
// Handler Registry
// ============================================================================

// awsMetricHandler resolves a request to either a CloudWatch query or a
// dedicated unsupported-metric error, picked by the per-MetricType registry.
type awsMetricHandler func(*AwsMonitoringHandler, monitoringQuery) (irs.MetricData, error)

// EKS nodes are standard EC2 instances, so the cluster-node path reuses the
// same handlers (same AWS/EC2 metrics, InstanceId dimension).
var awsMetricHandlers = map[irs.MetricType]awsMetricHandler{
	irs.CPUUsage: awsDirect(irs.CPUUsage, awsSpecCPU),
	irs.MemoryUsage: awsRejected(
		"memory_usage is not supported for AWS VMs in API-based(agentless) monitoring.",
	),
	irs.DiskRead:     awsDiskDispatch(irs.DiskRead, awsSpecDiskRead, awsSpecEBSRead),
	irs.DiskWrite:    awsDiskDispatch(irs.DiskWrite, awsSpecDiskWrite, awsSpecEBSWrite),
	irs.DiskReadOps:  awsDiskDispatch(irs.DiskReadOps, awsSpecDiskReadOps, awsSpecEBSReadOps),
	irs.DiskWriteOps: awsDiskDispatch(irs.DiskWriteOps, awsSpecDiskWriteOps, awsSpecEBSWriteOps),
	irs.NetworkIn:    awsDirect(irs.NetworkIn, awsSpecNetworkIn),
	irs.NetworkOut:   awsDirect(irs.NetworkOut, awsSpecNetworkOut),
}

func awsDirect(mt irs.MetricType, spec awsMetricSpec) awsMetricHandler {
	return func(h *AwsMonitoringHandler, q monitoringQuery) (irs.MetricData, error) {
		return h.getMetricData(mt, spec, q)
	}
}

// awsRejected returns a handler that fails fast with a fixed reason. Used for
// MetricTypes present in the common interface but intentionally unsupported on
// EC2's API-based monitoring (e.g., memory_usage requires CloudWatch Agent).
func awsRejected(reason string) awsMetricHandler {
	return func(_ *AwsMonitoringHandler, _ monitoringQuery) (irs.MetricData, error) {
		err := errors.New(reason)
		cblogger.Error(err.Error())
		return irs.MetricData{}, err
	}
}

// awsDiskDispatch picks the right CloudWatch metric for a disk request based
// on the instance type. EC2 publishes "Disk*" only for instance-store volumes
// and "EBS*" only for EBS volumes — the two are mutually exclusive per
// instance-type, so we dispatch on InstanceStorageSupported. Falls back to the
// EBS spec when the type lookup fails (Nitro EBS-only is the current default).
func awsDiskDispatch(mt irs.MetricType, instStoreSpec, ebsSpec awsMetricSpec) awsMetricHandler {
	return func(h *AwsMonitoringHandler, q monitoringQuery) (irs.MetricData, error) {
		spec := ebsSpec
		if q.instanceType != "" {
			hasIS, err := awsInstanceHasInstanceStore(h.VMClient, q.instanceType)
			if err != nil {
				cblogger.Errorf("failed to resolve instance-store support for %q (defaulting to EBS metrics): %v", q.instanceType, err)
			} else if hasIS {
				spec = instStoreSpec
			}
		}
		return h.getMetricData(mt, spec, q)
	}
}

// awsInstanceStoreCache memoizes DescribeInstanceTypes lookups. Instance type
// catalog is stable per region, so a simple per-process map is enough.
var awsInstanceStoreCache sync.Map // map[string]bool

func awsInstanceHasInstanceStore(svc *ec2.EC2, instanceType string) (bool, error) {
	if v, ok := awsInstanceStoreCache.Load(instanceType); ok {
		cblogger.Debugf("[disk-dispatch] cache HIT instance_type=%s has_instance_store=%v", instanceType, v.(bool))
		return v.(bool), nil
	}
	cblogger.Infof("[disk-dispatch] cache MISS instance_type=%s — calling DescribeInstanceTypes", instanceType)
	out, err := svc.DescribeInstanceTypes(&ec2.DescribeInstanceTypesInput{
		InstanceTypes: []*string{aws.String(instanceType)},
	})
	if err != nil {
		return false, err
	}
	if len(out.InstanceTypes) == 0 {
		return false, fmt.Errorf("instance type not found: %s", instanceType)
	}
	has := aws.BoolValue(out.InstanceTypes[0].InstanceStorageSupported)
	awsInstanceStoreCache.Store(instanceType, has)
	cblogger.Infof("[disk-dispatch] cached instance_type=%s has_instance_store=%v", instanceType, has)
	return has, nil
}

// ============================================================================
// Public API
// ============================================================================

func (handler *AwsMonitoringHandler) GetVMMetricData(req irs.VMMonitoringReqInfo) (irs.MetricData, error) {
	cblogger.Info("AWS Cloud Driver: called GetVMMetricData()")

	if req.VMIID.NameId == "" && req.VMIID.SystemId == "" {
		return irs.MetricData{}, errors.New("VMIID is empty")
	}

	intervalMinute, timeBeforeHour, err := awsParseAndValidateInterval(req.IntervalMinute, req.TimeBeforeHour)
	if err != nil {
		return irs.MetricData{}, err
	}

	instanceID, instanceType, instanceState, err := handler.resolveInstance(req.VMIID)
	if err != nil {
		return irs.MetricData{}, err
	}

	handle, ok := awsMetricHandlers[req.MetricType]
	if !ok {
		return irs.MetricData{}, fmt.Errorf("unsupported VM metric type: %s", req.MetricType)
	}

	q := monitoringQuery{
		instanceID:     instanceID,
		instanceType:   instanceType,
		instanceState:  instanceState,
		intervalMinute: intervalMinute,
		timeBeforeHour: timeBeforeHour,
	}
	return handle(handler, q)
}

// GetClusterNodeMetricData fetches node-level metrics for an EKS node. EKS
// nodes are standard EC2 instances, so queries reuse the same AWS/EC2 metrics
// as VMs with the node's EC2 InstanceId.
func (handler *AwsMonitoringHandler) GetClusterNodeMetricData(req irs.ClusterNodeMonitoringReqInfo) (irs.MetricData, error) {
	cblogger.Info("AWS Cloud Driver: called GetClusterNodeMetricData()")

	if req.ClusterIID.NameId == "" && req.ClusterIID.SystemId == "" {
		return irs.MetricData{}, errors.New("ClusterIID is empty")
	}

	intervalMinute, timeBeforeHour, err := awsParseAndValidateInterval(req.IntervalMinute, req.TimeBeforeHour)
	if err != nil {
		return irs.MetricData{}, err
	}

	if handler.ClusterHandler == nil {
		return irs.MetricData{}, errors.New("cluster handler is not configured")
	}
	cluster, err := handler.ClusterHandler.GetCluster(req.ClusterIID)
	if err != nil {
		return irs.MetricData{}, fmt.Errorf("failed to get cluster info: %w", err)
	}

	instanceID, err := findAwsClusterNodeInstanceID(cluster, req.NodeGroupID, req.NodeIID)
	if err != nil {
		return irs.MetricData{}, err
	}

	_, instanceType, instanceState, err := handler.resolveInstance(irs.IID{SystemId: instanceID})
	if err != nil {
		return irs.MetricData{}, err
	}

	handle, ok := awsMetricHandlers[req.MetricType]
	if !ok {
		return irs.MetricData{}, fmt.Errorf("unsupported cluster-node metric type: %s", req.MetricType)
	}

	q := monitoringQuery{
		instanceID:     instanceID,
		instanceType:   instanceType,
		instanceState:  instanceState,
		intervalMinute: intervalMinute,
		timeBeforeHour: timeBeforeHour,
	}
	return handle(handler, q)
}

func findAwsClusterNodeInstanceID(cluster irs.ClusterInfo, nodeGroupID, nodeIID irs.IID) (string, error) {
	for _, nodeGroup := range cluster.NodeGroupList {
		if nodeGroup.IId.NameId != nodeGroupID.NameId &&
			nodeGroup.IId.SystemId != nodeGroupID.SystemId {
			continue
		}
		for _, node := range nodeGroup.Nodes {
			if node.NameId != nodeIID.NameId && node.SystemId != nodeIID.SystemId {
				continue
			}
			if node.SystemId != "" {
				return node.SystemId, nil
			}
			return node.NameId, nil
		}
	}
	return "", errors.New("node not found in the cluster")
}

// ============================================================================
// Metric Execution
// ============================================================================

func (handler *AwsMonitoringHandler) getMetricData(
	metricType irs.MetricType,
	spec awsMetricSpec,
	q monitoringQuery,
) (irs.MetricData, error) {
	endTime := time.Now().UTC()
	startTime := endTime.Add(-time.Duration(q.timeBeforeHour) * time.Hour)
	periodSec := int64(q.intervalMinute * 60)

	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(spec.Namespace),
		MetricName: aws.String(spec.MetricName),
		Dimensions: []*cloudwatch.Dimension{
			{Name: aws.String("InstanceId"), Value: aws.String(q.instanceID)},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int64(periodSec),
		Statistics: []*string{aws.String(spec.Statistic)},
	}

	resp, err := handler.CwClient.GetMetricStatistics(input)
	if err != nil {
		return irs.MetricData{}, fmt.Errorf("failed to get metric data: %w", err)
	}

	result := irs.MetricData{
		MetricName:      spec.MetricName,
		MetricUnit:      spec.Unit,
		TimestampValues: []irs.TimestampValue{},
	}

	points := resp.Datapoints
	sortDatapointsAsc(points)

	for _, dp := range points {
		v := pickDatapointValue(dp, spec.Statistic)
		if v == nil {
			continue
		}
		val := *v
		if spec.PerSecond && periodSec > 0 {
			val = val / float64(periodSec)
		}
		result.TimestampValues = append(result.TimestampValues, irs.TimestampValue{
			Timestamp: dp.Timestamp.UTC(),
			Value:     strconv.FormatFloat(val, 'f', -1, 64),
		})
	}

	if len(result.TimestampValues) == 0 {
		return irs.MetricData{}, awsDiagnoseEmptyMetric(metricType, q.instanceID, q.instanceState)
	}
	return result, nil
}

// ============================================================================
// Helpers
// ============================================================================

// resolveInstance returns the EC2 instance id, type, and state. instanceType
// is exposed so disk-metric dispatch can pick instance-store vs EBS metrics
// without a second API call. NameId is not supported as an EC2 lookup key on
// AWS; callers must provide SystemId.
func (handler *AwsMonitoringHandler) resolveInstance(vmIID irs.IID) (id, instanceType, state string, err error) {
	instanceID := vmIID.SystemId
	if instanceID == "" {
		instanceID = vmIID.NameId
	}
	if instanceID == "" {
		return "", "", "", errors.New("instance id is empty")
	}

	out, err := handler.VMClient.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	})
	if err != nil {
		return "", "", "", fmt.Errorf("failed to describe instance %q: %w", instanceID, err)
	}
	for _, res := range out.Reservations {
		for _, inst := range res.Instances {
			st := ""
			if inst.State != nil && inst.State.Name != nil {
				st = *inst.State.Name
			}
			return aws.StringValue(inst.InstanceId),
				aws.StringValue(inst.InstanceType),
				st,
				nil
		}
	}
	return "", "", "", fmt.Errorf("instance %q not found", instanceID)
}

// awsParseAndValidateInterval mirrors the Azure handler: TimeBeforeHour*60
// must be >= IntervalMinute. Empty strings default to "1". CloudWatch accepts
// any positive period so we do not restrict the interval values themselves.
func awsParseAndValidateInterval(intervalMinuteStr, timeBeforeHourStr string) (intervalMinute, timeBeforeHour int, err error) {
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

func pickDatapointValue(dp *cloudwatch.Datapoint, stat string) *float64 {
	if dp == nil {
		return nil
	}
	switch stat {
	case "Sum":
		return dp.Sum
	case "Average":
		return dp.Average
	case "Minimum":
		return dp.Minimum
	case "Maximum":
		return dp.Maximum
	case "SampleCount":
		return dp.SampleCount
	default:
		return dp.Average
	}
}

func sortDatapointsAsc(points []*cloudwatch.Datapoint) {
	for i := 1; i < len(points); i++ {
		for j := i; j > 0; j-- {
			if points[j-1].Timestamp == nil || points[j].Timestamp == nil {
				break
			}
			if points[j-1].Timestamp.After(*points[j].Timestamp) {
				points[j-1], points[j] = points[j], points[j-1]
			} else {
				break
			}
		}
	}
}

func awsDiagnoseEmptyMetric(metricType irs.MetricType, instanceID, state string) error {
	if state != "" && state != "running" {
		return fmt.Errorf(
			"no %s data: instance %q is in %q state (must be running to emit metrics)",
			metricType, instanceID, state)
	}
	return fmt.Errorf(
		"no %s data for running instance %q in the requested window: "+
			"the instance may have been started recently or detailed monitoring is off; "+
			"try increasing TimeBeforeHour",
		metricType, instanceID)
}
