// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Azure Driver.
//
// by CB-Spider Team, 2026.

package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v8"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v9"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzureMonitoringHandler struct {
	CredentialInfo                  idrv.CredentialInfo
	Region                          idrv.RegionInfo
	Ctx                             context.Context
	VMClient                        *armcompute.VirtualMachinesClient
	ManagedClustersClient           *armcontainerservice.ManagedClustersClient
	SecurityGroupsClient            *armnetwork.SecurityGroupsClient
	VirtualNetworksClient           *armnetwork.VirtualNetworksClient
	AgentPoolsClient                *armcontainerservice.AgentPoolsClient
	VirtualMachineScaleSetsClient   *armcompute.VirtualMachineScaleSetsClient
	VirtualMachineScaleSetVMsClient *armcompute.VirtualMachineScaleSetVMsClient
	VMSizeClient                    *armcompute.VirtualMachineSizesClient
	MetricClient                    *azquery.MetricsClient
}

var availableIntervalMinutes = []string{
	"1",
	"5",
	"15",
	"30",
	"60",
	"360",
	"720",
	"1440",
}

func toAzureIntervalMinute(intervalMinute string) (string, error) {
	switch intervalMinute {
	case availableIntervalMinutes[0]:
		return "PT1M", nil
	case availableIntervalMinutes[1]:
		return "PT5M", nil
	case availableIntervalMinutes[2]:
		return "PT15M", nil
	case availableIntervalMinutes[3]:
		return "PT30M", nil
	case availableIntervalMinutes[4]:
		return "PT1H", nil
	case availableIntervalMinutes[5]:
		return "PT6H", nil
	case availableIntervalMinutes[6]:
		return "PT12H", nil
	case availableIntervalMinutes[7]:
		return "P1D", nil
	default:
		return "", errors.New(fmt.Sprintf("Unsupported interval minute: %s. "+
			"Available interval miniutes: %s", intervalMinute, strings.Join(availableIntervalMinutes, ",")))
	}
}

func (monitoringHandler *AzureMonitoringHandler) getMetricData(metricType irs.MetricType, interval string, timeBeforeHour int, resourceID string) (irs.MetricData, error) {
	endTime := time.Now().UTC()
	startTime := endTime.Add(time.Duration(-timeBeforeHour) * time.Hour)
	timespan := azquery.TimeInterval(fmt.Sprintf("%s/%s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339)))

	var metricName = "Percentage CPU" // irs.CPUUsage
	var aggregation = azquery.AggregationTypeAverage

	switch metricType {
	case irs.MemoryUsage:
		metricName = "Available Memory Bytes"
	case irs.DiskRead:
		metricName = "Disk Read Bytes"
		aggregation = azquery.AggregationTypeTotal
	case irs.DiskWrite:
		metricName = "Disk Write Bytes"
		aggregation = azquery.AggregationTypeTotal
	case irs.DiskReadOps:
		metricName = "Disk Read Operations/Sec"
	case irs.DiskWriteOps:
		metricName = "Disk Write Operations/Sec"
	case irs.NetworkIn:
		metricName = "Network In"
		aggregation = azquery.AggregationTypeTotal
	case irs.NetworkOut:
		metricName = "Network Out"
		aggregation = azquery.AggregationTypeTotal
	}

	metrics := make([]string, 0)
	metrics = append(metrics, metricName)
	metricNames := strings.Join(metrics, ",")
	resultType := azquery.ResultTypeData

	resp, err := monitoringHandler.MetricClient.QueryResource(context.Background(), resourceID, &azquery.MetricsClientQueryResourceOptions{
		Aggregation:     []*azquery.AggregationType{&aggregation},
		Filter:          nil,
		Interval:        toStrPtr(interval),
		MetricNames:     &metricNames,
		MetricNamespace: nil,
		OrderBy:         nil,
		ResultType:      &resultType,
		Timespan:        &timespan,
		Top:             nil,
	})
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get metric data. err = %s", err))
		cblogger.Error(getErr.Error())
		return irs.MetricData{}, getErr
	}

	var vmMonitoringInfo irs.MetricData
	var timestampValues []irs.TimestampValue

	for i, metric := range resp.Value {
		if i == 0 && metric.Unit != nil {
			if metric.Name != nil && metric.Name.Value != nil {
				vmMonitoringInfo.MetricName = *metric.Name.Value
			}
			vmMonitoringInfo.MetricUnit = string(*metric.Unit)
		}

		for _, timeseries := range metric.TimeSeries {
			if timeseries.Data == nil {
				continue
			}
			for _, data := range timeseries.Data {
				timestamp := data.TimeStamp
				if timestamp == nil {
					continue
				}

				var value *float64
				if aggregation == azquery.AggregationTypeTotal {
					value = data.Total
				} else {
					value = data.Average
				}

				if value == nil {
					continue
				}

				timestampValues = append(timestampValues, irs.TimestampValue{
					Timestamp: *timestamp,
					Value:     strconv.FormatFloat(*value, 'f', -1, 64),
				})
			}
		}
	}

	vmMonitoringInfo.TimestampValues = timestampValues

	return vmMonitoringInfo, nil
}

// getVMMemoryPercent converts Azure's "Available Memory Bytes" into a
// used-percent series. Azure does not expose a memory-percent counter for
// the base VM platform, so we derive percent = (total - available) / total * 100
// using the VM size's declared memory as the denominator.
func (monitoringHandler *AzureMonitoringHandler) getVMMemoryPercent(interval string, timeBeforeHour int, resourceID string, vmSizeName string) (irs.MetricData, error) {
	totalBytes := monitoringHandler.getVMTotalMemoryBytes(vmSizeName)
	if totalBytes <= 0 {
		return irs.MetricData{}, fmt.Errorf("failed to resolve total memory for VM size %q", vmSizeName)
	}

	available, err := monitoringHandler.getMetricData(irs.MemoryUsage, interval, timeBeforeHour, resourceID)
	if err != nil {
		return irs.MetricData{}, err
	}

	totalF := float64(totalBytes)
	result := irs.MetricData{
		TimestampValues: make([]irs.TimestampValue, 0, len(available.TimestampValues)),
	}
	for _, tv := range available.TimestampValues {
		availF, parseErr := strconv.ParseFloat(tv.Value, 64)
		if parseErr != nil {
			continue
		}
		usedPct := (totalF - availF) / totalF * 100
		if usedPct < 0 {
			usedPct = 0
		}
		result.TimestampValues = append(result.TimestampValues, irs.TimestampValue{
			Timestamp: tv.Timestamp,
			Value:     strconv.FormatFloat(usedPct, 'f', -1, 64),
		})
	}
	return result, nil
}

func (monitoringHandler *AzureMonitoringHandler) GetVMMetricData(vmMonitoringReqInfo irs.VMMonitoringReqInfo) (irs.MetricData, error) {
	intervalMinute, err := strconv.Atoi(vmMonitoringReqInfo.IntervalMinute)
	if err != nil {
		if vmMonitoringReqInfo.IntervalMinute == "" {
			vmMonitoringReqInfo.IntervalMinute = "1"
			intervalMinute = 1
		} else {
			return irs.MetricData{}, errors.New("invalid value of IntervalMinute")
		}
	}

	interval, err := toAzureIntervalMinute(vmMonitoringReqInfo.IntervalMinute)
	if err != nil {
		return irs.MetricData{}, err
	}

	timeBeforeHour, err := strconv.Atoi(vmMonitoringReqInfo.TimeBeforeHour)
	if err != nil {
		if vmMonitoringReqInfo.TimeBeforeHour == "" {
			vmMonitoringReqInfo.TimeBeforeHour = "1"
			timeBeforeHour = 1
		} else {
			return irs.MetricData{}, errors.New("invalid value of TimeBeforeHour")
		}
	}
	if timeBeforeHour < 0 {
		return irs.MetricData{}, errors.New("invalid value of TimeBeforeHour")
	}

	if timeBeforeHour*60 < intervalMinute {
		return irs.MetricData{}, errors.New("IntervalMinute is too far in the past")
	}

	// log HisCall
	hiscallInfo := GetCallLogScheme(monitoringHandler.Region, call.MONITORING, vmMonitoringReqInfo.VMIID.NameId, "GetVMMetricData()")
	start := call.Start()

	convertedIID, err := ConvertVMIID(vmMonitoringReqInfo.VMIID, monitoringHandler.CredentialInfo, monitoringHandler.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get metric data. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.MetricData{}, getErr
	}

	vm, err := GetRawVM(convertedIID, monitoringHandler.Region.Region, monitoringHandler.VMClient, monitoringHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get metric data. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.MetricData{}, getErr
	}

	var vmMonitoringInfo irs.MetricData
	if vmMonitoringReqInfo.MetricType == irs.MemoryUsage {
		vmSizeName := ""
		if vm.Properties != nil && vm.Properties.HardwareProfile != nil && vm.Properties.HardwareProfile.VMSize != nil {
			vmSizeName = string(*vm.Properties.HardwareProfile.VMSize)
		}
		vmMonitoringInfo, err = monitoringHandler.getVMMemoryPercent(interval, timeBeforeHour, *vm.ID, vmSizeName)
	} else {
		vmMonitoringInfo, err = monitoringHandler.getMetricData(vmMonitoringReqInfo.MetricType, interval, timeBeforeHour, *vm.ID)
	}
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get metric data. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.MetricData{}, getErr
	}

	LoggingInfo(hiscallInfo, start)

	return vmMonitoringInfo, nil
}

func (monitoringHandler *AzureMonitoringHandler) getVMTotalMemoryBytes(vmSizeName string) int64 {
	pager := monitoringHandler.VMSizeClient.NewListPager(monitoringHandler.Region.Region, nil)
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			cblogger.Error(fmt.Sprintf("Failed to list VM sizes: %s", err))
			return 0
		}
		for _, size := range page.Value {
			if size.Name != nil && string(*size.Name) == vmSizeName {
				if size.MemoryInMB != nil {
					return int64(*size.MemoryInMB) * 1024 * 1024
				}
			}
		}
	}
	return 0
}

func (monitoringHandler *AzureMonitoringHandler) GetClusterNodeMetricData(clusterNodeMonitoringReqInfo irs.ClusterNodeMonitoringReqInfo) (irs.MetricData, error) {
	intervalMinute, err := strconv.Atoi(clusterNodeMonitoringReqInfo.IntervalMinute)
	if err != nil {
		if clusterNodeMonitoringReqInfo.IntervalMinute == "" {
			clusterNodeMonitoringReqInfo.IntervalMinute = "1"
			intervalMinute = 1
		} else {
			return irs.MetricData{}, errors.New("invalid value of IntervalMinute")
		}
	}

	interval, err := toAzureIntervalMinute(clusterNodeMonitoringReqInfo.IntervalMinute)
	if err != nil {
		return irs.MetricData{}, err
	}

	timeBeforeHour, err := strconv.Atoi(clusterNodeMonitoringReqInfo.TimeBeforeHour)
	if err != nil {
		if clusterNodeMonitoringReqInfo.TimeBeforeHour == "" {
			clusterNodeMonitoringReqInfo.TimeBeforeHour = "1"
			timeBeforeHour = 1
		} else {
			return irs.MetricData{}, errors.New("invalid value of TimeBeforeHour")
		}
	}
	if timeBeforeHour < 0 {
		return irs.MetricData{}, errors.New("invalid value of TimeBeforeHour")
	}

	if timeBeforeHour*60 < intervalMinute {
		return irs.MetricData{}, errors.New("IntervalMinute is too far in the past")
	}

	// log HisCall
	hiscallInfo := GetCallLogScheme(monitoringHandler.Region, call.MONITORING, clusterNodeMonitoringReqInfo.ClusterIID.NameId, "GetClusterNodeMetricData()")
	start := call.Start()

	clusterHandler := AzureClusterHandler{
		Region:                          monitoringHandler.Region,
		CredentialInfo:                  monitoringHandler.CredentialInfo,
		Ctx:                             monitoringHandler.Ctx,
		ManagedClustersClient:           monitoringHandler.ManagedClustersClient,
		SecurityGroupsClient:            monitoringHandler.SecurityGroupsClient,
		VirtualNetworksClient:           monitoringHandler.VirtualNetworksClient,
		AgentPoolsClient:                monitoringHandler.AgentPoolsClient,
		VirtualMachineScaleSetsClient:   monitoringHandler.VirtualMachineScaleSetsClient,
		VirtualMachineScaleSetVMsClient: monitoringHandler.VirtualMachineScaleSetVMsClient,
	}

	cluster, err := clusterHandler.GetCluster(clusterNodeMonitoringReqInfo.ClusterIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get cluster info. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.MetricData{}, getErr
	}

	var nodeFound bool
	var resourceID string
	var vmSizeName string

	for _, nodeGroup := range cluster.NodeGroupList {
		if nodeGroup.IId.NameId == clusterNodeMonitoringReqInfo.NodeGroupID.NameId ||
			nodeGroup.IId.SystemId == clusterNodeMonitoringReqInfo.NodeGroupID.SystemId {
			for _, node := range nodeGroup.Nodes {
				if node.NameId == clusterNodeMonitoringReqInfo.NodeIID.NameId ||
					node.SystemId == clusterNodeMonitoringReqInfo.NodeIID.SystemId {
					nodeFound = true
					resourceID = node.SystemId
					vmSizeName = nodeGroup.VMSpecName
					break
				}
			}
		}
	}

	if !nodeFound {
		getErr := errors.New(fmt.Sprintf("Failed to get metric data. err = Node not found from the cluster"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.MetricData{}, getErr
	}

	var vmMonitoringInfo irs.MetricData
	if clusterNodeMonitoringReqInfo.MetricType == irs.MemoryUsage {
		vmMonitoringInfo, err = monitoringHandler.getVMMemoryPercent(interval, timeBeforeHour, resourceID, vmSizeName)
	} else {
		vmMonitoringInfo, err = monitoringHandler.getMetricData(clusterNodeMonitoringReqInfo.MetricType, interval, timeBeforeHour, resourceID)
	}
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get metric data. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.MetricData{}, getErr
	}

	LoggingInfo(hiscallInfo, start)

	return vmMonitoringInfo, nil
}
