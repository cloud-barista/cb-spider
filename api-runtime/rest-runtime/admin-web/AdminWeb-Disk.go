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

// Function to fetch Disks
func fetchDisks(connConfig string) ([]*cres.DiskInfo, error) {
	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "disk")
	if err != nil {
		return nil, fmt.Errorf("error fetching Disks: %v", err)
	}

	var info struct {
		ResultList []*cres.DiskInfo `json:"disk"`
	}
	if err := json.Unmarshal(resBody, &info); err != nil {
		return nil, fmt.Errorf("error decoding Disks: %v", err)
	}

	sort.Slice(info.ResultList, func(i, j int) bool {
		return info.ResultList[i].IId.NameId < info.ResultList[j].IId.NameId
	})

	return info.ResultList, nil
}

// Handler function to render the Disk management page
func DiskManagement(c echo.Context) error {
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

	disks, err := fetchDisks(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	vms, err := fetchVMs(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	data := struct {
		ConnectionConfig string
		RegionName       string
		Region           string
		Zone             string
		Disks            []*cres.DiskInfo
		VMs              []*cres.VMInfo
		APIUsername      string
		APIPassword      string
	}{
		ConnectionConfig: connConfig,
		RegionName:       regionName,
		Region:           region,
		Zone:             zone,
		Disks:            disks,
		VMs:              vms,
		APIUsername:      os.Getenv("SPIDER_USERNAME"),
		APIPassword:      os.Getenv("SPIDER_PASSWORD"),
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/disk.html")
	tmpl, err := template.New("disk.html").Funcs(template.FuncMap{
		"inc": func(i int) int {
			return i + 1
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

// DiskMultiConnection describes a connection used by the multi-connection Disk page.
// Provider selects the disk types to offer and Zone is the default zone of a new disk.
type DiskMultiConnection struct {
	Name     string
	Provider string
	Zone     string
}

// DiskMultiRow pairs a Disk with the connection it was fetched from,
// so a single table can list Disks aggregated across several connections.
type DiskMultiRow struct {
	ConnectionName string
	Disk           *cres.DiskInfo
}

// Handler function to render the Disk management page across multiple connections at once.
// Each connection is fetched independently so that one failing connection does not block the
// others from being displayed (partial success).
func DiskManagementMulti(c echo.Context) error {
	idsParam := c.QueryParam("ids")
	if strings.TrimSpace(idsParam) == "" {
		htmlStr := `
			<html>
			<body>
				<br><br>
				<label style="font-size:24px;color:#606262;">&nbsp;&nbsp;&nbsp;Please select connections for multi-connection management! (MENU: 2.CONNECTION)</label>
			</body>
			`
		return c.HTML(http.StatusOK, htmlStr)
	}

	var connNames []string
	for _, id := range strings.Split(idsParam, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			connNames = append(connNames, id)
		}
	}

	var conns []DiskMultiConnection
	var rows []DiskMultiRow
	connErrors := map[string]string{}

	for _, connName := range connNames {
		conn := DiskMultiConnection{Name: connName}
		// provider and zone are used only to create Disks, so the Disk list is still shown without them
		conn.Provider, _ = getProviderName(connName)
		if regionName, err := getRegionName(connName); err == nil {
			_, conn.Zone, _ = getRegionZone(regionName)
		}
		conns = append(conns, conn)
		if isMultiAsyncLoad(c) {
			continue
		}

		disks, err := fetchDisks(connName)
		if err != nil {
			connErrors[connName] = err.Error()
			continue
		}

		for _, disk := range disks {
			rows = append(rows, DiskMultiRow{
				ConnectionName: connName,
				Disk:           disk,
			})
		}
	}

	data := struct {
		Connections []DiskMultiConnection
		DiskRows    []DiskMultiRow
		ConnErrors  map[string]string
		APIUsername string
		APIPassword string
	}{
		Connections: conns,
		DiskRows:    rows,
		ConnErrors:  connErrors,
		APIUsername: os.Getenv("SPIDER_USERNAME"),
		APIPassword: os.Getenv("SPIDER_PASSWORD"),
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/disk-multi.html")
	tmpl, err := template.New("disk-multi.html").Funcs(template.FuncMap{
		"inc": func(i int) int {
			return i + 1
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
