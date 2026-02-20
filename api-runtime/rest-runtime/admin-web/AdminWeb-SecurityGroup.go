// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2024.08.

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

// Function to fetch SecurityGroups
func fetchSecurityGroups(connConfig string) ([]*cres.SecurityInfo, error) {
	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "securitygroup")
	if err != nil {
		return nil, fmt.Errorf("error fetching SecurityGroups: %v", err)
	}

	var info struct {
		ResultList []*cres.SecurityInfo `json:"securitygroup"`
	}
	if err := json.Unmarshal(resBody, &info); err != nil {
		return nil, fmt.Errorf("error decoding SecurityGroups: %v", err)
	}

	// Sort the SecurityRuleList of each SecurityGroup by FromPort
	for _, sg := range info.ResultList {
		if sg.SecurityRules != nil {
			sort.Slice(*sg.SecurityRules, func(i, j int) bool {
				return (*sg.SecurityRules)[i].FromPort < (*sg.SecurityRules)[j].FromPort
			})
		}
	}

	return info.ResultList, nil
}

// Handler function to render the SecurityGroup management page
func SecurityGroupManagement(c echo.Context) error {
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

	regionName, err := getRegionName(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	region, zone, err := getRegionZone(regionName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Fetch SecurityGroups
	sgs, err := fetchSecurityGroups(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	data := struct {
		ConnectionConfig string
		RegionName       string
		Region           string
		Zone             string
		SecurityGroups   []*cres.SecurityInfo
		APIUsername      string
		APIPassword      string
	}{
		ConnectionConfig: connConfig,
		RegionName:       regionName,
		Region:           region,
		Zone:             zone,
		SecurityGroups:   sgs,
		APIUsername:      os.Getenv("API_USERNAME"),
		APIPassword:      os.Getenv("API_PASSWORD"),
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/security-group.html")
	tmpl, err := template.New("security-group.html").Funcs(template.FuncMap{
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
