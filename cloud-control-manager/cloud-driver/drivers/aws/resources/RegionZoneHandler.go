package resources

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/aws/aws-sdk-go/service/ec2"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsRegionZoneHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

func (regionZoneHandler *AwsRegionZoneHandler) ListRegionZone() ([]*irs.RegionZoneInfo, error) {

	responseRegions, err := DescribeRegions(regionZoneHandler.Client, true, "")
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	var regionZoneInfoList []*irs.RegionZoneInfo
	for _, region := range responseRegions.Regions {
		sess, err := session.NewSession(&aws.Config{
			Region: region.RegionName,
		})
		if err != nil {
			cblogger.Error(err)
		}
		tempclient := ec2.New(sess)

		responseZones, err := DescribeAvailabilityZones(tempclient, true)
		if err != nil {
			cblogger.Errorf("AuthFailure on [%s]", *region.RegionName)
			cblogger.Error(err)
		} else {
			var zoneInfoList []irs.ZoneInfo
			for _, zone := range responseZones.AvailabilityZones {
				zoneInfo := irs.ZoneInfo{}
				zoneInfo.Name = *zone.ZoneName
				zoneInfo.DisplayName = *zone.ZoneName
				zoneInfo.Status = GetZoneStatus(*zone.State)

				// keyValueList 삭제 https://github.com/cloud-barista/cb-spider/issues/930#issuecomment-1734817828
				// zoneInfo.KeyValueList, err = ConvertKeyValueList(zone)
				// if err != nil {
				// 	cblogger.Error(err)
				// 	zoneInfo.KeyValueList = nil
				// }

				zoneInfoList = append(zoneInfoList, zoneInfo)
			}

			regionInfo := irs.RegionZoneInfo{}
			regionInfo.Name = *region.RegionName
			regionInfo.DisplayName = *region.RegionName
			regionInfo.ZoneList = zoneInfoList

			// keyValueList 삭제 https://github.com/cloud-barista/cb-spider/issues/930#issuecomment-1734817828
			// regionInfo.KeyValueList, err = ConvertKeyValueList(region)
			// if err != nil {
			// 	cblogger.Error(err)
			// 	regionInfo.KeyValueList = nil
			// }

			regionZoneInfoList = append(regionZoneInfoList, &regionInfo)
		}
	}

	return regionZoneInfoList, nil
}

func (regionZoneHandler *AwsRegionZoneHandler) GetRegionZone(Name string) (irs.RegionZoneInfo, error) {
	responseRegions, err := DescribeRegions(regionZoneHandler.Client, true, Name)
	if err != nil {
		cblogger.Error(err)
		return irs.RegionZoneInfo{}, err
	}

	var regionZoneInfo irs.RegionZoneInfo
	for _, region := range responseRegions.Regions {
		sess, err := session.NewSession(&aws.Config{
			Region: region.RegionName,
		})
		if err != nil {
			cblogger.Error(err)
		}
		tempclient := ec2.New(sess)

		responseZones, err := DescribeAvailabilityZones(tempclient, true)
		if err != nil {
			cblogger.Errorf("AuthFailure on [%s]", *region.RegionName)
			cblogger.Error(err)
		} else {
			var zoneInfoList []irs.ZoneInfo
			for _, zone := range responseZones.AvailabilityZones {
				zoneInfo := irs.ZoneInfo{}
				zoneInfo.Name = *zone.ZoneName
				zoneInfo.DisplayName = *zone.ZoneName
				zoneInfo.Status = GetZoneStatus(*zone.State)

				// keyValueList 삭제 https://github.com/cloud-barista/cb-spider/issues/930#issuecomment-1734817828
				// zoneInfo.KeyValueList, err = ConvertKeyValueList(zone)
				// if err != nil {
				// 	cblogger.Error(err)
				// 	zoneInfo.KeyValueList = nil
				// }

				zoneInfoList = append(zoneInfoList, zoneInfo)
			}

			regionZoneInfo.Name = *region.RegionName
			regionZoneInfo.DisplayName = *region.RegionName
			regionZoneInfo.ZoneList = zoneInfoList

			// // keyValueList 삭제 https://github.com/cloud-barista/cb-spider/issues/930#issuecomment-1734817828
			// regionZoneInfo.KeyValueList, err = ConvertKeyValueList(region)
			// if err != nil {
			// 	cblogger.Error(err)
			// 	regionZoneInfo.KeyValueList = nil
			// }

		}
	}

	return regionZoneInfo, nil
}

func (regionZoneHandler *AwsRegionZoneHandler) ListOrgRegion() (string, error) {

	respRegions, err := DescribeRegions(regionZoneHandler.Client, true, "")
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	jsonString, errJson := ConvertJsonString(respRegions)
	if errJson != nil {
		cblogger.Error(err)
		return "", errJson
	}

	return jsonString, errJson
}

func (regionZoneHandler *AwsRegionZoneHandler) ListOrgZone() (string, error) {

	responseRegions, err := DescribeRegions(regionZoneHandler.Client, true, "")
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	var responseZonesList []*ec2.DescribeAvailabilityZonesOutput

	for _, region := range responseRegions.Regions {

		sess, err := session.NewSession(&aws.Config{
			Region: region.RegionName,
		})
		if err != nil {
			cblogger.Errorf("NewSession err %s", *region.RegionName)
			cblogger.Error(err)
		} else {
			tempclient := ec2.New(sess)

			responseZones, err := DescribeAvailabilityZones(tempclient, true)
			if err != nil {
				cblogger.Errorf("DescribeAvailabilityZones err %s", *region.RegionName)
				cblogger.Error(err)
			} else {
				responseZonesList = append(responseZonesList, responseZones)
			}

		}
	}

	jsonString, errJson := ConvertJsonString(responseZonesList)
	if errJson != nil {
		cblogger.Error(err)
		return "", err
	}

	return jsonString, nil
}
