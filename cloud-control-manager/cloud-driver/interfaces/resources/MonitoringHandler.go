// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2020.04.
// by CB-Spider Team, 2019.06.

package resources

import (
	"time"
)

type VMMonitoringReqInfo struct {
	VMIID          IID
	MetricType     MetricType
	IntervalMinute string
	TimeBeforeHour string
}

type ClusterNodeMonitoringReqInfo struct {
	ClusterIID     IID
	NodeGroupID    IID
	NodeIID        IID
	MetricType     MetricType
	IntervalMinute string
	TimeBeforeHour string
}

type TimestampValue struct {
	Timestamp time.Time `json:"timestamp"`
	Value     string    `json:"value"`
}

type MetricData struct {
	MetricName      string           `json:"metricName"`
	MetricUnit      string           `json:"metricUnit"`
	TimestampValues []TimestampValue `json:"timestampValues"`
}

type MetricType string

const (
	CPUUsage     MetricType = "cpu_usage"
	MemoryUsage  MetricType = "memory_usage"
	DiskRead     MetricType = "disk_read"
	DiskWrite    MetricType = "disk_write"
	DiskReadOps  MetricType = "disk_read_ops"
	DiskWriteOps MetricType = "disk_write_ops"
	NetworkIn    MetricType = "network_in"
	NetworkOut   MetricType = "network_out"
	Unknown      MetricType = "unknown"
)

func StringMetricType(input string) MetricType {
	switch input {
	case "cpu_usage":
		return CPUUsage
	case "memory_usage":
		return MemoryUsage
	case "disk_read":
		return DiskRead
	case "disk_write":
		return DiskWrite
	case "disk_read_ops":
		return DiskReadOps
	case "disk_write_ops":
		return DiskWriteOps
	case "network_in":
		return NetworkIn
	case "network_out":
		return NetworkOut
	default:
		return Unknown
	}
}

type MonitoringHandler interface {
	GetVMMetricData(vmMonitoringReqInfo VMMonitoringReqInfo) (MetricData, error)
	GetClusterNodeMetricData(clusterMonitoringReqInfo ClusterNodeMonitoringReqInfo) (MetricData, error)
}
