// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This covers only the Public Authoritative DNS baseline (Zone + Record CRUD).
//
// by CB-Spider Team, 2026.09.

package adminweb

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

// Function to fetch DNS Zones
func fetchDNSZones(connConfig string) ([]*cres.DNSZoneInfo, error) {
	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "dnszone")
	if err != nil {
		return nil, fmt.Errorf("error fetching DNS Zones: %v", err)
	}

	var info struct {
		ResultList []*cres.DNSZoneInfo `json:"dnszone"`
	}
	if err := json.Unmarshal(resBody, &info); err != nil {
		return nil, fmt.Errorf("error decoding DNS Zones: %v", err)
	}

	// Sort the RecordInfoList of each Zone by NameId
	for _, zone := range info.ResultList {
		sort.Slice(zone.RecordInfoList, func(i, j int) bool {
			return zone.RecordInfoList[i].IId.NameId < zone.RecordInfoList[j].IId.NameId
		})
	}

	return info.ResultList, nil
}

// Handler function to render the DNS Zone management page
func DNSZoneManagement(c echo.Context) error {
	connConfig := c.Param("ConnectConfig")
	if connConfig == "region not set" {
		htmlStr := `
			<html>
			<head>
			    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
				<style>
				th {
				  border: 1px solid lightgray;
				}
				td {
				  border: 1px solid lightgray;
				  border-radius: 4px;
				}
				</style>
			</head>
			<body>
				<br>
				<br>
				<label style="font-size:24px;color:#606262;">&nbsp;&nbsp;&nbsp;Please select a Connection Configuration! (MENU: 2.CONNECTION)</label>
			</body>
		`

		return c.HTML(http.StatusOK, htmlStr)
	}

	// Fetch DNS Zones
	dnsZones, err := fetchDNSZones(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	data := struct {
		ConnectionConfig string
		DNSZones         []*cres.DNSZoneInfo
		APIUsername      string
		APIPassword      string
	}{
		ConnectionConfig: connConfig,
		DNSZones:         dnsZones,
		APIUsername:      os.Getenv("SPIDER_USERNAME"),
		APIPassword:      os.Getenv("SPIDER_PASSWORD"),
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/dns-zone.html")
	tmpl, err := template.New("dns-zone.html").Funcs(template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
		"join": strings.Join,
	}).ParseFiles(templatePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error loading template: " + err.Error()})
	}

	c.Response().WriteHeader(http.StatusOK)
	if err := tmpl.Execute(c.Response().Writer, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	return nil
}
