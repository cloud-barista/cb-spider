// Proxy for AdminWeb(Multi-Connection VPC/Subnet) of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista

package adminweb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
)

//====================================== Multi-Connection VPC/Subnet: zones for the Add VPC/Subnet dialog
//
// The Add VPC/Subnet dialog creates a second subnet in another zone of each connection's region, so that
// the VPC can host an AWS EKS cluster (subnets in at least 2 zones). The zones of a region are listed by
// the CSP, so they are fetched on the server side and streamed per connection (see streamPerConnection).

// vpcZoneInfo is one streamed line: the zones of the region of one connection.
type vpcZoneInfo struct {
	Connection string   `json:"connection"`
	Zone       string   `json:"zone"`            // the default zone of the connection
	Zones      []string `json:"zones"`           // zones of the region, except the ones the CSP reports unavailable
	Error      string   `json:"error,omitempty"` // the zones could not be listed: only the default zone is used
}

// fetchVPCZoneInfo lists the zones of the region of one connection.
func fetchVPCZoneInfo(connName string) vpcZoneInfo {
	info := vpcZoneInfo{Connection: connName, Zones: []string{}}
	regionName, err := getRegionName(connName)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	region, zone, err := getRegionZone(regionName)
	if err != nil || region == "" {
		info.Error = fmt.Sprintf("no region configured for %s", connName)
		return info
	}
	info.Zone = zone

	resp, err := httpGetWithAuth(fmt.Sprintf("%s/spider/regionzone/%s?ConnectionName=%s", spiderBaseURL(), url.PathEscape(region), url.QueryEscape(connName)))
	if err != nil {
		info.Error = err.Error()
		return info
	}
	defer resp.Body.Close()
	var regionZone struct {
		ZoneList []struct {
			Name   string `json:"Name"`
			Status string `json:"Status"`
		} `json:"ZoneList"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&regionZone); err != nil || resp.StatusCode != http.StatusOK {
		info.Error = fmt.Sprintf("failed to list the zones (HTTP %d) %s", resp.StatusCode, regionZone.Message)
		return info
	}
	for _, z := range regionZone.ZoneList {
		// Some CSPs do not report a zone status (StatusNotSupported): those zones are kept.
		if z.Name != "" && z.Status != "Unavailable" {
			info.Zones = append(info.Zones, z.Name)
		}
	}
	return info
}

// VPCZoneInfo streams the zones of the regions of the given connections, one JSON line each,
// in the order the connections finish.
// GET /spider/adminweb/vpc-zoneinfo?ids=conn1,conn2
func VPCZoneInfo(c echo.Context) error {
	return streamPerConnection(c, connectionIDs(c), func(connName string) any { return fetchVPCZoneInfo(connName) })
}
