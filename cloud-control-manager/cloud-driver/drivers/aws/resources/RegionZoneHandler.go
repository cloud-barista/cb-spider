package resources

import (
	"errors"
	"sync"

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

	responseRegions, err := DescribeRegions(regionZoneHandler.Client, false, "")
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	var errlist []error
	chanRegionZoneInfos := make(chan irs.RegionZoneInfo, len(responseRegions.Regions))
	var wg sync.WaitGroup
	for _, region := range responseRegions.Regions {
		wg.Add(1)
		go func(region *ec2.Region) {
			defer wg.Done()

			sess, err := session.NewSession(&aws.Config{
				Credentials: regionZoneHandler.Client.Config.Credentials,
				Region:      region.RegionName,
			})
			if err != nil {
				cblogger.Error(err)
			}
			tempclient := ec2.New(sess)

			responseZones, err := DescribeAvailabilityZones(tempclient, false)
			if err != nil {
				cblogger.Infof("error on [%s]", *region.RegionName)
				cblogger.Infof(err.Error())
				errlist = append(errlist, err)
			} else {
				var zoneInfoList []irs.ZoneInfo
				for _, zone := range responseZones.AvailabilityZones {
					zoneInfo := irs.ZoneInfo{}
					zoneInfo.Name = *zone.ZoneName
					zoneInfo.DisplayName = *zone.ZoneName
					zoneInfo.Status = GetZoneStatus(*zone.State)
					zoneInfoList = append(zoneInfoList, zoneInfo)
				}
				regionInfo := irs.RegionZoneInfo{}
				regionInfo.Name = *region.RegionName
				regionInfo.DisplayName = *region.RegionName
				regionInfo.ZoneList = zoneInfoList

				chanRegionZoneInfos <- regionInfo
			}
		}(region)

	}
	wg.Wait()
	close(chanRegionZoneInfos)

	var regionZoneInfoList []*irs.RegionZoneInfo
	for regionZoneInfo := range chanRegionZoneInfos {
		insertRegionZoneInfo := regionZoneInfo
		regionZoneInfoList = append(regionZoneInfoList, &insertRegionZoneInfo)
	}

	if len(errlist) > 0 {
		errlistjoin := errors.Join(errlist...)
		cblogger.Error("ListRegionZone() error : ", errlistjoin)
		return regionZoneInfoList, errlistjoin
	}

	return regionZoneInfoList, nil
}

func (regionZoneHandler *AwsRegionZoneHandler) GetRegionZone(Name string) (irs.RegionZoneInfo, error) {
	responseRegions, err := DescribeRegions(regionZoneHandler.Client, false, Name)
	if err != nil {
		cblogger.Error(err)
		return irs.RegionZoneInfo{}, err
	}

	var regionZoneInfo irs.RegionZoneInfo
	for _, region := range responseRegions.Regions {
		cblogger.Debug("#################### region.RegionName", region.RegionName)
		sess, err := session.NewSession(&aws.Config{
			Credentials: regionZoneHandler.Client.Config.Credentials,
			Region:      region.RegionName,
		})
		if err != nil {
			cblogger.Error(err)
		}
		tempclient := ec2.New(sess)

		responseZones, err := DescribeAvailabilityZones(tempclient, false)
		if err != nil {
			cblogger.Errorf("error on [%s]", *region.RegionName)
			cblogger.Error(err)
		} else {
			var zoneInfoList []irs.ZoneInfo
			for _, zone := range responseZones.AvailabilityZones {
				zoneInfo := irs.ZoneInfo{}
				zoneInfo.Name = *zone.ZoneName
				zoneInfo.DisplayName = *zone.ZoneName
				zoneInfo.Status = GetZoneStatus(*zone.State)

				zoneInfoList = append(zoneInfoList, zoneInfo)
			}

			regionZoneInfo.Name = *region.RegionName
			regionZoneInfo.DisplayName = *region.RegionName
			regionZoneInfo.ZoneList = zoneInfoList
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
			Credentials: regionZoneHandler.Client.Config.Credentials,
			Region:      region.RegionName,
		})
		if err != nil {
			cblogger.Errorf("NewSession err %s", *region.RegionName)
			cblogger.Error(err)
		} else {
			tempclient := ec2.New(sess)

			responseZones, err := DescribeAvailabilityZones(tempclient, false)
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
