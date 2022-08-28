package resources

import (
	"encoding/json"
	"errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"reflect"
)

//// Alibaba API 1:1로 대응

/**
디스크 목록 조회
*/
func DescribeDisks(client *ecs.Client, regionInfo idrv.RegionInfo, instanceIID irs.IID, diskIIDs []irs.IID) ([]ecs.Disk, error) {
	regionID := regionInfo.Region

	request := ecs.CreateDescribeDisksRequest()
	request.Scheme = "https"
	request.RegionId = regionID

	if CBPageOn {
		request.PageNumber = requests.NewInteger(CBPageNumber)
		request.PageSize = requests.NewInteger(CBPageSize)
	}

	if instanceIID != (irs.IID{}) {
		request.InstanceId = instanceIID.SystemId
	}

	var diskIIDList []string
	for _, diskIID := range diskIIDs {
		diskIIDList = append(diskIIDList, diskIID.SystemId)
	}
	diskJson, err := json.Marshal(diskIIDList)
	if err != nil {

	}
	if len(diskIIDList) > 0 {
		request.DiskIds = string(diskJson)
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   regionInfo.Zone,
		ResourceType: call.DISK,
		ResourceName: "ListDisk()",
		CloudOSAPI:   "DescribeDisks()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()

	var totalCount = 0
	curPage := CBPageNumber
	var resultDiskList []ecs.Disk
	for {
		result, err := client.DescribeDisks(request)
		callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
		//spew.Dump(result) //출력 정보가 너무 많아서 생략
		if err != nil {
			callLogInfo.ErrorMSG = err.Error()
			callogger.Error(call.String(callLogInfo))

			cblogger.Errorf("Unable to get Images, %v", err)
			return resultDiskList, err
		}
		callogger.Info(call.String(callLogInfo))

		resultDiskList = append(resultDiskList, result.Disks.Disk...)
		if CBPageOn {
			totalCount = len(resultDiskList)
			cblogger.Infof("CSP 전체 이미지 갯수 : [%d] - 현재 페이지:[%d] - 누적 결과 개수:[%d]", result.TotalCount, curPage, totalCount)
			if totalCount >= result.TotalCount {
				break
			}
			curPage++
			request.PageNumber = requests.NewInteger(curPage)
		} else {
			break
		}
	}
	cblogger.Info(cblogger.Level.String())
	if cblogger.Level.String() == "debug" {
		spew.Dump(resultDiskList)
	}
	return resultDiskList, nil
}

/**
1개 Disk의 정보 조회
*/
func DescribeDiskByDiskId(client *ecs.Client, regionInfo idrv.RegionInfo, diskIID irs.IID) (ecs.Disk, error) {

	var diskIIDList []irs.IID
	diskIIDList = append(diskIIDList, diskIID)

	diskList, err := DescribeDisks(client, regionInfo, irs.IID{}, diskIIDList)
	if err != nil {
		return ecs.Disk{}, err
	}

	if len(diskList) != 1 {
		return ecs.Disk{}, errors.New("search failed")
	}

	return diskList[0], nil
}

/**
1개 Disk의 정보 조회
*/
func DescribeDisksByInstanceId(client *ecs.Client, regionInfo idrv.RegionInfo, instanceIID irs.IID) ([]ecs.Disk, error) {

	diskList, err := DescribeDisks(client, regionInfo, instanceIID, nil)
	//if err != nil {
	//	return nil, err
	//}
	//
	//if len(diskList) != 1 {
	//	return nil, errors.New("search failed")
	//}

	return diskList, err
}

/**
해당 리소스가 사용가능한지 조회
	https://help.aliyun.com/document_detail/66186.html?spm=api-workbench.Troubleshoot.0.0.43651e0folUpip#doc-api-Ecs-DescribeAvailableResource
	https://next.api.alibabacloud.com/api/Ecs/2014-05-26/DescribeAvailableResource?lang=GO&params={}


필수 parameter
	RegionId
	DestinationResource : "Zone", "IoOptimized", "InstanceType", "SystemDisk", "DataDisk", "Network", "ddh"

결과 : AvailableZone 값이 들어있음. 배열형태임.
	비정상 : requestID만 반환. ex) {"RequestId":"7F2E6252-7FF6-31AF-9067-1EECF1B6B3FA"}
	정상 : requestID 외에 Available
		ex) {"RequestId":"7F2E6252-7FF6-31AF-9067-1EECF1B6B3FA","AvailableZones":{"AvailableZone":[{"Status":"Available","StatusCategory":"WithStock","ZoneId":"ap-southeast-1b","AvailableResources":{"AvailableResource":[{"Type":"DataDisk","SupportedResources":{"SupportedResource":[{"Status":"Available","Min":20,"Max":32768,"Value":"cloud_efficiency","Unit":"GiB"}]}}]},"RegionId":"ap-southeast-1"}]}}
*/
func DescribeAvailableResource(client *ecs.Client, regionId string, zoneId string, resourceType string, destinationResource string, categoryValue string) (ecs.AvailableZonesInDescribeAvailableResource, error) {

	request := ecs.CreateDescribeAvailableResourceRequest()
	request.Scheme = "https"

	request.RegionId = regionId
	if zoneId != "" {
		request.ZoneId = zoneId
	}

	request.ResourceType = resourceType

	request.DestinationResource = destinationResource
	switch destinationResource {

	case "Zone":
		request.ZoneId = categoryValue
	case "IoOptimized":
		request.IoOptimized = categoryValue
	case "InstanceType":
		request.InstanceType = categoryValue
	case "SystemDisk":
		request.SystemDiskCategory = categoryValue
	case "DataDisk":
		request.DataDiskCategory = categoryValue
	case "Network":
		request.NetworkCategory = categoryValue
	case "ddh":
		request.DedicatedHostId = categoryValue
	}
	//request.DataDiskCategory = "cloud"
	spew.Dump(request)
	result, err := client.DescribeAvailableResource(request)
	cblogger.Info(result)
	if err != nil {
		cblogger.Errorf("DescribeAvailableResource %v.", err)
	}
	spew.Dump(result)

	metaValue := reflect.ValueOf(result).Elem()
	fieldAvailableZones := metaValue.FieldByName("AvailableZones")
	if fieldAvailableZones == (reflect.Value{}) {
		cblogger.Errorf("Field not exist")
		cblogger.Errorf("Not available in this region")
		return ecs.AvailableZonesInDescribeAvailableResource{}, errors.New("Not available in this region")
	}

	return result.AvailableZones, nil
}

/**
Instance에 Disk Attach
한번에 1개씩.
*/
func AttachDisk(client *ecs.Client, regionInfo idrv.RegionInfo, ownerVM irs.IID, diskIID irs.IID) error {

	cblogger.Infof("AttachDisk : [%s]", diskIID.SystemId)
	// Delete the Image by Id

	request := ecs.CreateAttachDiskRequest()
	request.Scheme = "https"

	request.DiskId = diskIID.SystemId
	request.InstanceId = ownerVM.SystemId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   regionInfo.Zone,
		ResourceType: call.DISK,
		ResourceName: diskIID.SystemId,
		CloudOSAPI:   "AttachDisk()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	result, err := client.AttachDisk(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to attach Disk: %s, %v.", diskIID.SystemId, err)
		return err
	}
	callogger.Info(call.String(callLogInfo))
	return nil
}

/**
Instance 목록 조회
 iid를 parameter로 주면 해당 iid들만 조회.
*/

func DescribeInstances(client *ecs.Client, regionInfo idrv.RegionInfo, vmIIDs []irs.IID) ([]ecs.Instance, error) {
	request := ecs.CreateDescribeInstancesRequest()
	request.Scheme = "https"

	//request.InstanceId = &[]string{vmIID.SystemId}
	var instanceIdList []string
	for _, instanceIID := range vmIIDs {
		instanceIdList = append(instanceIdList, instanceIID.SystemId)
	}
	if len(instanceIdList) > 0 {
		vmsJson, err := json.Marshal(instanceIdList)
		if err != nil {

		}
		request.InstanceIds = string(vmsJson)

	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   regionInfo.Zone,
		ResourceType: call.VM,
		ResourceName: "ListVM()",
		CloudOSAPI:   "DescribeInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	response, err := client.DescribeInstances(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))

	return response.Instances.Instance, err
}

/**
인스턴스 1개만 조회
*/
func DescribeInstanceById(client *ecs.Client, regionInfo idrv.RegionInfo, vmIID irs.IID) (ecs.Instance, error) {

	var vmIIDs []irs.IID
	vmIIDs = append(vmIIDs, vmIID)
	response, err := DescribeInstances(client, regionInfo, vmIIDs)
	if err != nil {
		return ecs.Instance{}, err
	}
	if len(response) < 1 {
		return ecs.Instance{}, errors.New("Notfound: '" + vmIID.SystemId + "' VM Not found")
	}

	return response[0], nil
}
