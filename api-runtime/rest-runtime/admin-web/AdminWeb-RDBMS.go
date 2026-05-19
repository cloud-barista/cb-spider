// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, April 2026.

package adminweb

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

// Function to fetch RDBMSs
func fetchRDBMSs(connConfig string) ([]*cres.RDBMSInfo, error) {
	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "rdbms")
	if err != nil {
		return nil, fmt.Errorf("error fetching RDBMSs: %v", err)
	}

	var info struct {
		ResultList []*cres.RDBMSInfo `json:"rdbms"`
	}
	if err := json.Unmarshal(resBody, &info); err != nil {
		return nil, fmt.Errorf("error decoding RDBMSs: %v", err)
	}

	return info.ResultList, nil
}

// Handler function to render the RDBMS management page
func RDBMSManagement(c echo.Context) error {
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
			    <script type="text/javascript">
				alert(connConfig)
			    </script>
			</head>
			<body>
				<br>
				<br>
				<label style="font-size:24px;color:#606262;">&nbsp;&nbsp;&nbsp;Please select a Connection Configuration! (MENU: 2.CONNECTION)</label>	
			</body>
		`

		return c.HTML(http.StatusOK, htmlStr)
	}

	providerName, err := getProviderName(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	regionName, err := getRegionName(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	region, zone, err := getRegionZone(regionName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Fetch RDBMSs
	rdbmss, err := fetchRDBMSs(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	data := struct {
		ConnectionConfig string
		ProviderName     string
		RegionName       string
		Region           string
		Zone             string
		RDBMSs           []*cres.RDBMSInfo
		APIUsername      string
		APIPassword      string
	}{
		ConnectionConfig: connConfig,
		ProviderName:     providerName,
		RegionName:       regionName,
		Region:           region,
		Zone:             zone,
		RDBMSs:           rdbmss,
		APIUsername:      os.Getenv("SPIDER_USERNAME"),
		APIPassword:      os.Getenv("SPIDER_PASSWORD"),
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/rdbms.html")
	tmpl, err := template.New("rdbms.html").Funcs(template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
		"trim": func(s string) string {
			return strings.TrimSpace(s)
		},
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
