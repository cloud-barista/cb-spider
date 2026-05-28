package resources

import (
	"context"
	"encoding/json"
	"fmt"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
)

type OracleRegionZoneHandler struct {
	Region         idrv.RegionInfo
	TenancyID      string
	Client         identity.IdentityClient
	ConfigProvider common.ConfigurationProvider
	Ctx            context.Context
}

func (handler *OracleRegionZoneHandler) ListRegionZone() ([]*irs.RegionZoneInfo, error) {
	hiscallInfo := getCallLogScheme(handler.Region, call.REGIONZONE, "RegionZone", "ListRegionZone()")
	start := call.Start()

	// ListRegions is a global public API available to all tenancies including Free Tier.
	regionsResp, err := handler.Client.ListRegions(handler.Ctx)
	if err != nil {
		wrapped := statusErr("failed to list Oracle regions", err)
		logError(hiscallInfo, wrapped)
		return nil, wrapped
	}

	// Build a set of subscribed regions (paid accounts can subscribe to multiple regions).
	// Free Tier only has the home region subscribed.
	subscribedZones := handler.listSubscribedRegionZones()

	results := make([]*irs.RegionZoneInfo, 0, len(regionsResp.Items))
	for _, region := range regionsResp.Items {
		regionName := stringValue(region.Name)
		zones, ok := subscribedZones[regionName]
		if !ok {
			zones = []irs.ZoneInfo{}
		}
		info := irs.RegionZoneInfo{
			Name:           regionName,
			DisplayName:    regionName,
			CSPDisplayName: regionName,
			ZoneList:       zones,
			KeyValueList:   []irs.KeyValue{},
		}
		results = append(results, &info)
	}

	logInfo(hiscallInfo, start)
	return results, nil
}

// listSubscribedRegionZones returns AD lists keyed by region name for all subscribed regions.
func (handler *OracleRegionZoneHandler) listSubscribedRegionZones() map[string][]irs.ZoneInfo {
	result := make(map[string][]irs.ZoneInfo)

	if handler.ConfigProvider == nil {
		// Fallback: only home region.
		zones, _ := handler.listAvailabilityDomains()
		result[handler.Region.Region] = zones
		return result
	}

	subResp, err := handler.Client.ListRegionSubscriptions(handler.Ctx, identity.ListRegionSubscriptionsRequest{
		TenancyId: common.String(handler.TenancyID),
	})
	if err != nil {
		// If permission denied, fall back to home region only.
		if cblogger != nil {
			cblogger.Warnf("Oracle: failed to list region subscriptions, using home region only: %v", err)
		}
		zones, _ := handler.listAvailabilityDomains()
		result[handler.Region.Region] = zones
		return result
	}

	for _, sub := range subResp.Items {
		if sub.Status != identity.RegionSubscriptionStatusReady {
			continue
		}
		regionName := stringValue(sub.RegionName)
		if regionName == handler.Region.Region {
			// Use the already-connected client for the home region.
			zones, _ := handler.listAvailabilityDomains()
			result[regionName] = zones
			continue
		}
		// Create a region-specific client for each additionally subscribed region.
		regionalClient, clientErr := identity.NewIdentityClientWithConfigurationProvider(handler.ConfigProvider)
		if clientErr != nil {
			if cblogger != nil {
				cblogger.Warnf("Oracle: failed to create identity client for region %s: %v", regionName, clientErr)
			}
			continue
		}
		regionalClient.SetRegion(regionName)
		adResp, adErr := regionalClient.ListAvailabilityDomains(handler.Ctx, identity.ListAvailabilityDomainsRequest{
			CompartmentId: common.String(handler.TenancyID),
		})
		if adErr != nil {
			if cblogger != nil {
				cblogger.Warnf("Oracle: failed to list ADs for region %s: %v", regionName, adErr)
			}
			result[regionName] = []irs.ZoneInfo{}
			continue
		}
		zones := make([]irs.ZoneInfo, 0, len(adResp.Items))
		for _, ad := range adResp.Items {
			zoneName := stringValue(ad.Name)
			zones = append(zones, irs.ZoneInfo{
				Name:           zoneName,
				DisplayName:    zoneName,
				CSPDisplayName: zoneName,
				Status:         irs.ZoneAvailable,
				KeyValueList: []irs.KeyValue{
					{Key: "Id", Value: stringValue(ad.Id)},
					{Key: "CompartmentId", Value: stringValue(ad.CompartmentId)},
				},
			})
		}
		result[regionName] = zones
	}
	return result
}

func (handler *OracleRegionZoneHandler) GetRegionZone(name string) (irs.RegionZoneInfo, error) {
	hiscallInfo := getCallLogScheme(handler.Region, call.REGIONZONE, name, "GetRegionZone()")
	start := call.Start()

	// Only subscribed regions return AD information; use the current region for AD lookup.
	if name != "" && name != handler.Region.Region {
		// Verify region exists via ListRegions, but ADs are unavailable for unsubscribed regions.
		regionsResp, err := handler.Client.ListRegions(handler.Ctx)
		if err != nil {
			wrapped := statusErr("failed to list Oracle regions", err)
			logError(hiscallInfo, wrapped)
			return irs.RegionZoneInfo{}, wrapped
		}
		for _, r := range regionsResp.Items {
			if stringValue(r.Name) == name {
				info := irs.RegionZoneInfo{
					Name:           name,
					DisplayName:    name,
					CSPDisplayName: name,
					ZoneList:       []irs.ZoneInfo{},
					KeyValueList:   []irs.KeyValue{},
				}
				logInfo(hiscallInfo, start)
				return info, nil
			}
		}
		err = fmt.Errorf("Oracle Driver: region %s not found", name)
		logError(hiscallInfo, err)
		return irs.RegionZoneInfo{}, err
	}

	info, err := handler.regionZoneInfo()
	if err != nil {
		wrapped := statusErr("failed to get Oracle region zone", err)
		logError(hiscallInfo, wrapped)
		return irs.RegionZoneInfo{}, wrapped
	}
	logInfo(hiscallInfo, start)
	return info, nil
}

func (handler *OracleRegionZoneHandler) ListOrgRegion() (string, error) {
	regionsResp, err := handler.Client.ListRegions(handler.Ctx)
	if err != nil {
		return "", statusErr("failed to list Oracle regions", err)
	}
	data, err := json.Marshal(regionsResp.Items)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (handler *OracleRegionZoneHandler) ListOrgZone() (string, error) {
	zones, err := handler.listAvailabilityDomains()
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(zones)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (handler *OracleRegionZoneHandler) regionZoneInfo() (irs.RegionZoneInfo, error) {
	zones, err := handler.listAvailabilityDomains()
	if err != nil {
		return irs.RegionZoneInfo{}, err
	}

	return irs.RegionZoneInfo{
		Name:           handler.Region.Region,
		DisplayName:    handler.Region.Region,
		CSPDisplayName: handler.Region.Region,
		ZoneList:       zones,
		KeyValueList:   []irs.KeyValue{},
	}, nil
}

func (handler *OracleRegionZoneHandler) listAvailabilityDomains() ([]irs.ZoneInfo, error) {
	resp, err := handler.Client.ListAvailabilityDomains(handler.Ctx, identity.ListAvailabilityDomainsRequest{CompartmentId: common.String(handler.TenancyID)})
	if err != nil {
		return nil, err
	}

	zones := make([]irs.ZoneInfo, 0, len(resp.Items))
	for _, ad := range resp.Items {
		zoneName := stringValue(ad.Name)
		zoneInfo := irs.ZoneInfo{
			Name:           zoneName,
			DisplayName:    zoneName,
			CSPDisplayName: zoneName,
			Status:         irs.ZoneAvailable,
			KeyValueList: []irs.KeyValue{
				{Key: "Id", Value: stringValue(ad.Id)},
				{Key: "CompartmentId", Value: stringValue(ad.CompartmentId)},
			},
		}
		zones = append(zones, zoneInfo)
	}

	return zones, nil
}
