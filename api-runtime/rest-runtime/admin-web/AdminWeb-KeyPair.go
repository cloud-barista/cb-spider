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

// Function to fetch KeyPairs
func fetchKeyPairs(connConfig string) ([]*cres.KeyPairInfo, error) {
	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "keypair")
	if err != nil {
		return nil, fmt.Errorf("error fetching KeyPairs: %v", err)
	}

	var info struct {
		ResultList []*cres.KeyPairInfo `json:"keypair"`
	}
	if err := json.Unmarshal(resBody, &info); err != nil {
		return nil, fmt.Errorf("error decoding KeyPairs: %v", err)
	}

	sort.Slice(info.ResultList, func(i, j int) bool {
		return info.ResultList[i].IId.NameId < info.ResultList[j].IId.NameId
	})

	return info.ResultList, nil
}

// Handler function to render the KeyPair management page
func KeyPairManagement(c echo.Context) error {
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

	keyPairs, err := fetchKeyPairs(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	data := struct {
		ConnectionConfig string
		RegionName       string
		KeyPairs         []*cres.KeyPairInfo
		APIUsername      string
		APIPassword      string
	}{
		ConnectionConfig: connConfig,
		RegionName:       regionName,
		KeyPairs:         keyPairs,
		APIUsername:      os.Getenv("SPIDER_USERNAME"),
		APIPassword:      os.Getenv("SPIDER_PASSWORD"),
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/keypair.html")
	tmpl, err := template.New("keypair.html").Funcs(template.FuncMap{
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

// isMultiAsyncLoad reports whether a multi-connection page is requested without its resource lists
// (load=async): the page is returned at once and lists each connection by itself, in parallel.
func isMultiAsyncLoad(c echo.Context) bool {
	return c.QueryParam("load") == "async"
}

// KeyPairMultiRow pairs a KeyPair with the connection it was fetched from,
// so a single table can list KeyPairs aggregated across several connections.
type KeyPairMultiRow struct {
	ConnectionName string
	RegionName     string
	KeyPair        *cres.KeyPairInfo
}

// Handler function to render the KeyPair management page across multiple connections at once.
// Each connection is fetched independently so that one failing connection does not block the
// others from being displayed (partial success).
func KeyPairManagementMulti(c echo.Context) error {
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

	var rows []KeyPairMultiRow
	connErrors := map[string]string{}

	for _, connName := range connNames {
		if isMultiAsyncLoad(c) {
			break
		}
		regionName, err := getRegionName(connName)
		if err != nil {
			connErrors[connName] = err.Error()
			continue
		}

		keyPairs, err := fetchKeyPairs(connName)
		if err != nil {
			connErrors[connName] = err.Error()
			continue
		}

		for _, keyPair := range keyPairs {
			rows = append(rows, KeyPairMultiRow{
				ConnectionName: connName,
				RegionName:     regionName,
				KeyPair:        keyPair,
			})
		}
	}

	data := struct {
		Connections []string
		KeyPairRows []KeyPairMultiRow
		ConnErrors  map[string]string
		APIUsername string
		APIPassword string
	}{
		Connections: connNames,
		KeyPairRows: rows,
		ConnErrors:  connErrors,
		APIUsername: os.Getenv("SPIDER_USERNAME"),
		APIPassword: os.Getenv("SPIDER_PASSWORD"),
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/keypair-multi.html")
	tmpl, err := template.New("keypair-multi.html").Funcs(template.FuncMap{
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
