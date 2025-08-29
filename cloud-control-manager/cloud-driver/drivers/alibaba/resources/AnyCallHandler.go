// Cloud Driver Interface of CB-Spider.
// AnyCallHandler for Alibaba driver
// by CB-Spider Team

package resources

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaAnyCallHandler struct {
	RegionId string
	ZoneId   string
	Client   *ecs.Client
}

/*
*******************************************************

	// CheckInstanceTypeAvailability
	curl -sX POST http://localhost:1024/spider/anycall -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "alibaba-beijing-config",
		"ReqInfo": {
			"FID": "CheckInstanceTypeAvailability",
			"IKeyValueList": [
				{"Key": "InstanceType", "Value": "ecs.c7.large"}
			]
		}
	}' | json_pp

	// GetAvailableSystemDisksByInstanceType
	curl -sX POST http://localhost:1024/spider/anycall -H 'Content-Type: application/json' -d '
	{
		"ConnectionName": "alibaba-beijing-config",
		"ReqInfo": {
			"FID": "GetAvailableSystemDisksByInstanceType",
			"IKeyValueList": [
				{"Key": "InstanceType", "Value": "ecs.c7.large"}
			]
		}
	}' | json_pp

	// GetAvailableDataDisksByInstanceType
	curl -sX POST http://localhost:1024/spider/anycall -H 'Content-Type: application/json' -d '
	{
		"ConnectionName": "alibaba-beijing-config",
		"ReqInfo": {
			"FID": "GetAvailableDataDisksByInstanceType",
			"IKeyValueList": [
				{"Key": "InstanceType", "Value": "ecs.c7.large"}
			]
		}
	}' | json_pp

	// GetInstanceTypeAvailableZones
	curl -sX POST http://localhost:1024/spider/anycall -H 'Content-Type: application/json' -d '
	{
		"ConnectionName": "alibaba-beijing-config",
		"ReqInfo": {
			"FID": "GetInstanceTypeAvailableZones",
			"IKeyValueList": [
				{"Key": "InstanceType", "Value": "ecs.c7.large"}
			]
		}
	}' | json_pp

	// GetInstanceTypeAvailableAllZones
	curl -sX POST http://localhost:1024/spider/anycall -H 'Content-Type: application/json' -d '
	{
		"ConnectionName": "alibaba-beijing-config",
		"ReqInfo": {
			"FID": "GetInstanceTypeAvailableAllZones",
			"IKeyValueList": [
				{"Key": "InstanceType", "Value": "ecs.c7.large"}
			]
		}
	}' | json_pp

*******************************************************
*/
func (anyCallHandler *AlibabaAnyCallHandler) AnyCall(callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	switch callInfo.FID {
	case "CheckInstanceTypeAvailability":
		return checkInstanceTypeAvailabilityAnyCall(anyCallHandler, callInfo)
	case "GetAvailableSystemDisksByInstanceType":
		return describeAvailableSystemDisksByInstanceTypeAnyCall(anyCallHandler, callInfo)
	case "GetAvailableDataDisksByInstanceType":
		return getAvailableDataDisksByInstanceTypeAnyCall(anyCallHandler, callInfo)
	case "GetInstanceTypeAvailableZones":
		return getInstanceTypeAvailableZonesAnyCall(anyCallHandler, callInfo)
	case "GetInstanceTypeAvailableAllZones":
		return getInstanceTypeAvailableAllZonesAnyCall(anyCallHandler, callInfo)
	default:
		return irs.AnyCallInfo{}, errors.New("Alibaba Driver: " + callInfo.FID + " Function is not implemented!")
	}
}

// AnyCall function for CheckInstanceTypeAvailability
func checkInstanceTypeAvailabilityAnyCall(anyCallHandler *AlibabaAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	if anyCallHandler.Client == nil {
		return irs.AnyCallInfo{}, errors.New("Alibaba Driver: " + callInfo.FID + " has no session")
	}

	var instanceType string

	for _, kv := range callInfo.IKeyValueList {
		if kv.Key == "InstanceType" {
			instanceType = kv.Value
		}
	}
	if instanceType == "" {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", "Missing required parameter: InstanceType"})
		return callInfo, nil
	}
	available, err := checkInstanceTypeAvailability(anyCallHandler.Client, anyCallHandler.RegionId, anyCallHandler.ZoneId, instanceType)
	if err != nil {
		cblogger.Errorf("checkInstanceTypeAvailability error: %v", err)
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", err.Error()})
		return callInfo, nil
	}
	if available {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "true"})
	} else {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", "InstanceType not available"})
	}
	return callInfo, nil
}

// AnyCall function for DescribeAvailableSystemDisksByInstanceType
func describeAvailableSystemDisksByInstanceTypeAnyCall(anyCallHandler *AlibabaAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	var instanceChargeType, destinationResource, instanceType string
	instanceChargeType = "PostPaid"
	destinationResource = "SystemDisk"

	for _, kv := range callInfo.IKeyValueList {
		switch kv.Key {
		case "InstanceType":
			instanceType = kv.Value
		}
	}
	if instanceChargeType == "" || destinationResource == "" || instanceType == "" {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", "Missing required parameters: InstanceChargeType, DestinationResource, InstanceType"})
		return callInfo, nil
	}
	availableZones, err := DescribeAvailableSystemDisksByInstanceType(
		anyCallHandler.Client,
		anyCallHandler.RegionId,
		anyCallHandler.ZoneId,
		instanceChargeType,
		destinationResource,
		instanceType,
	)
	if err != nil {
		cblogger.Errorf("DescribeAvailableSystemDisksByInstanceType error: %v", err)
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", err.Error()})
		return callInfo, nil
	}

	var systemDiskList []interface{}
	for _, zone := range availableZones.AvailableZone {
		for _, resource := range zone.AvailableResources.AvailableResource {
			if resource.Type == "SystemDisk" {
				for _, disk := range resource.SupportedResources.SupportedResource {
					systemDiskList = append(systemDiskList, disk)
				}
			}
		}
	}
	if err != nil {
		cblogger.Errorf("JSON marshal error: %v", err)
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", err.Error()})
		return callInfo, nil
	}
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "true"})
	var diskNames []string
	if len(systemDiskList) == 0 {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{Key: "AvailableSystemDisks", Value: "none"})
	} else {
		// systemDiskList is marshaled to JSON string, so unmarshal and extract names
		jsonBytes, err := json.Marshal(systemDiskList)
		if err == nil {
			var diskArr []map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &diskArr); err == nil {
				for _, disk := range diskArr {
					if name, ok := disk["Value"].(string); ok {
						diskNames = append(diskNames, name)
					}
				}
			}
		}
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{Key: "AvailableSystemDisks", Value: strings.Join(diskNames, ",")})
	}
	return callInfo, nil
}

// AnyCall function for GetAvailableDataDisksByInstanceType
func getAvailableDataDisksByInstanceTypeAnyCall(anyCallHandler *AlibabaAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	var instanceChargeType, destinationResource, instanceType string
	instanceChargeType = "PostPaid"
	destinationResource = "DataDisk"

	for _, kv := range callInfo.IKeyValueList {
		switch kv.Key {
		case "InstanceType":
			instanceType = kv.Value
		}
	}
	if instanceChargeType == "" || destinationResource == "" || instanceType == "" {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", "Missing required parameters: InstanceChargeType, DestinationResource, InstanceType"})
		return callInfo, nil
	}
	availableZones, err := DescribeAvailableSystemDisksByInstanceType(
		anyCallHandler.Client,
		anyCallHandler.RegionId,
		anyCallHandler.ZoneId,
		instanceChargeType,
		destinationResource,
		instanceType,
	)
	if err != nil {
		cblogger.Errorf("DescribeAvailableSystemDisksByInstanceType error: %v", err)
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", err.Error()})
		return callInfo, nil
	}

	var dataDiskList []interface{}
	for _, zone := range availableZones.AvailableZone {
		for _, resource := range zone.AvailableResources.AvailableResource {
			if resource.Type == "DataDisk" {
				for _, disk := range resource.SupportedResources.SupportedResource {
					dataDiskList = append(dataDiskList, disk)
				}
			}
		}
	}
	var diskNames []string
	if len(dataDiskList) == 0 {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{Key: "AvailableDataDisks", Value: "none"})
	} else {
		jsonBytes, err := json.Marshal(dataDiskList)
		if err == nil {
			var diskArr []map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &diskArr); err == nil {
				for _, disk := range diskArr {
					if name, ok := disk["Value"].(string); ok {
						diskNames = append(diskNames, name)
					}
				}
			}
		}
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{Key: "AvailableDataDisks", Value: strings.Join(diskNames, ",")})
	}
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "true"})
	return callInfo, nil
}

// AnyCall function for GetInstanceTypeAvailableZones
func getInstanceTypeAvailableZonesAnyCall(anyCallHandler *AlibabaAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	if anyCallHandler.Client == nil {
		return irs.AnyCallInfo{}, errors.New("Alibaba Driver: " + callInfo.FID + " has no session")
	}
	var instanceType string
	for _, kv := range callInfo.IKeyValueList {
		if kv.Key == "InstanceType" {
			instanceType = kv.Value
		}
	}
	if instanceType == "" {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", "Missing required parameter: InstanceType"})
		return callInfo, nil
	}

	availableZones, err := DescribeAvailableResource(anyCallHandler.Client, anyCallHandler.RegionId, "", "instance", "InstanceType", instanceType)
	if err != nil {
		cblogger.Errorf("DescribeAvailableResource error: %v", err)
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", err.Error()})
		return callInfo, nil
	}
	var zoneList []string
	for _, zone := range availableZones.AvailableZone {
		for _, resource := range zone.AvailableResources.AvailableResource {
			if resource.Type == "InstanceType" {
				for _, value := range resource.SupportedResources.SupportedResource {
					if value.Value == instanceType {
						zoneList = append(zoneList, zone.ZoneId)
					}
				}
			}
		}
	}
	if len(zoneList) == 0 {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{Key: "AvailableZones", Value: "none"})
	} else {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{Key: "AvailableZones", Value: strings.Join(zoneList, ",")})
	}
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "true"})
	return callInfo, nil
}

// AnyCall function for GetInstanceTypeAvailableAllZones
func getInstanceTypeAvailableAllZonesAnyCall(anyCallHandler *AlibabaAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	if anyCallHandler.Client == nil {
		return irs.AnyCallInfo{}, errors.New("Alibaba Driver: " + callInfo.FID + " has no session")
	}
	var instanceType string
	for _, kv := range callInfo.IKeyValueList {
		if kv.Key == "InstanceType" {
			instanceType = kv.Value
		}
	}
	if instanceType == "" {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", "Missing required parameter: InstanceType"})
		return callInfo, nil
	}
	regionsResp, err := DescribeRegions(anyCallHandler.Client)
	if err != nil {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", err.Error()})
		return callInfo, nil
	}
	var zoneList []string
	for _, region := range regionsResp.Regions.Region {
		regionId := region.RegionId
		zones, err := DescribeAvailableResource(anyCallHandler.Client, regionId, "", "instance", "InstanceType", instanceType)
		if err != nil {
			continue // skip error region
		}
		for _, zone := range zones.AvailableZone {
			for _, resource := range zone.AvailableResources.AvailableResource {
				if resource.Type == "InstanceType" {
					for _, value := range resource.SupportedResources.SupportedResource {
						if value.Value == instanceType {
							zoneList = append(zoneList, regionId+":"+zone.ZoneId)
						}
					}
				}
			}
		}
	}
	if len(zoneList) == 0 {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{Key: "AvailableAllZones", Value: "none"})
	} else {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{Key: "AvailableAllZones", Value: strings.Join(zoneList, ",")})
	}
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "true"})
	return callInfo, nil
}
