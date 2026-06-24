// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2025.06.

package adminweb

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

// fetchPublicIPs fetches the list of Public IPs from the Spider API.
func fetchPublicIPs(connConfig string) ([]*cres.PublicIPInfo, error) {
	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "publicip")
	if err != nil {
		return nil, fmt.Errorf("error fetching PublicIPs: %v", err)
	}

	var info struct {
		ResultList []*cres.PublicIPInfo `json:"publicip"`
	}
	if err := json.Unmarshal(resBody, &info); err != nil {
		return nil, fmt.Errorf("error decoding PublicIPs: %v", err)
	}

	sort.Slice(info.ResultList, func(i, j int) bool {
		return info.ResultList[i].IId.NameId < info.ResultList[j].IId.NameId
	})

	return info.ResultList, nil
}

// PublicIPManagement renders the PublicIP management page.
func PublicIPManagement(c echo.Context) error {
	connConfig := c.Param("ConnectConfig")
	if connConfig == "region not set" {
		htmlStr := `
			<html>
			<head>
			    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
				<style>
				th { border: 1px solid lightgray; }
				td { border: 1px solid lightgray; border-radius: 4px; }
				</style>
			</head>
			<body>
				<br><br>
				<label style="font-size:24px;color:#606262;">&nbsp;&nbsp;&nbsp;Please select a Connection Configuration! (MENU: 2.CONNECTION)</label>
			</body>
		`
		return c.HTML(http.StatusOK, htmlStr)
	}

	regionName, err := getRegionName(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	publicIPs, err := fetchPublicIPs(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	nics, _ := fetchNICs(connConfig)
	// Non-fatal: page renders without NIC dropdown if fetchNICs fails

	// Only fetch VMs as fallback when no NICs available (e.g. NCP).
	// Avoids unnecessary VM+Disk API calls for CSPs that support NICs (AWS, Azure, GCP, ...).
	var vms []*cres.VMInfo
	if len(nics) == 0 {
		vms, _ = fetchVMs(connConfig)
	}

	data := struct {
		ConnectionConfig string
		RegionName       string
		PublicIPs        []*cres.PublicIPInfo
		NICs             []*cres.NICInfo
		VMs              []*cres.VMInfo
		APIUsername      string
		APIPassword      string
	}{
		ConnectionConfig: connConfig,
		RegionName:       regionName,
		PublicIPs:        publicIPs,
		NICs:             nics,
		VMs:              vms,
		APIUsername:      os.Getenv("SPIDER_USERNAME"),
		APIPassword:      os.Getenv("SPIDER_PASSWORD"),
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/publicip.html")
	tmpl, err := template.New("publicip.html").Funcs(template.FuncMap{
		"inc": func(i int) int { return i + 1 },
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
