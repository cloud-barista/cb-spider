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

// Function to fetch VPCs
func fetchVPCs(connConfig string) ([]*cres.VPCInfo, error) {
	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "vpc")
	if err != nil {
		return nil, fmt.Errorf("error fetching VPCs: %v", err)
	}

	var info struct {
		ResultList []*cres.VPCInfo `json:"vpc"`
	}
	if err := json.Unmarshal(resBody, &info); err != nil {
		return nil, fmt.Errorf("error decoding VPCs: %v", err)
	}

	// Sort the SubnetInfoList of each VPC by NameId
	for _, vpc := range info.ResultList {
		sort.Slice(vpc.SubnetInfoList, func(i, j int) bool {
			return vpc.SubnetInfoList[i].IId.NameId < vpc.SubnetInfoList[j].IId.NameId
		})
	}

	return info.ResultList, nil
}

// Handler function to render the VPC-Subnet management page
func VPCSubnetManagement(c echo.Context) error {
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

	// Fetch VPCs
	vpcs, err := fetchVPCs(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	data := struct {
		ConnectionConfig string
		RegionName       string
		Region           string
		Zone             string
		VPCs             []*cres.VPCInfo
	}{
		ConnectionConfig: connConfig,
		RegionName:       regionName,
		Region:           region,
		Zone:             zone,
		VPCs:             vpcs,
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/vpc-subnet.html")
	tmpl, err := template.New("vpc-subnet.html").Funcs(template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	}).ParseFiles(templatePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error loading template: " + err.Error()})
	}

	return tmpl.Execute(c.Response().Writer, data)
}
