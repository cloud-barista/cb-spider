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
	"sync"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

// Function to fetch MyImages
func fetchMyImages(connConfig string) ([]*cres.MyImageInfo, error) {
	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "myimage")
	if err != nil {
		return nil, fmt.Errorf("error fetching MyImages: %v", err)
	}

	var info struct {
		ResultList []*cres.MyImageInfo `json:"myImage"`
	}
	if err := json.Unmarshal(resBody, &info); err != nil {
		return nil, fmt.Errorf("error decoding MyImages: %v", err)
	}

	sort.Slice(info.ResultList, func(i, j int) bool {
		return info.ResultList[i].IId.NameId < info.ResultList[j].IId.NameId
	})

	return info.ResultList, nil
}

// MyImage creation is much slower in some regions: an image is Available only when the snapshot of the
// system disk is complete. Comparison test (2026-11-18, ubuntu_24_04 alibase image, ecs.c9i.large,
// 40GB cloud_essd PL1, new VM): Alibaba ap-northeast-1 (Tokyo) 62s; cn-beijing 5% after 5m30s, about
// 40 minutes in total. The Alibaba driver waits up to 60 minutes in cn-beijing (see CommonHandler.go).
var slowMyImageRegions = map[string]map[string]string{
	"ALIBABA": {"cn-beijing": "about 40 minutes for a 40GB system disk (Tokyo: about 1 minute); Spider waits up to 60 minutes"},
}

// slowMyImageNotice returns the notice to show for a connection whose region creates MyImages slowly, or "".
func slowMyImageNotice(connName string) string {
	provider, err := getProviderName(connName)
	if err != nil {
		return ""
	}
	regions := slowMyImageRegions[strings.ToUpper(provider)]
	if regions == nil {
		return ""
	}
	regionName, err := getRegionName(connName)
	if err != nil {
		return ""
	}
	region, _, err := getRegionZone(regionName)
	if err != nil {
		return ""
	}
	if text, ok := regions[region]; ok {
		return fmt.Sprintf("%s (%s %s): MyImage creation is slow, %s.", connName, provider, region, text)
	}
	return ""
}

// Handler function to render the MyImage management page
func MyImageManagement(c echo.Context) error {
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

	myImages, err := fetchMyImages(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	data := struct {
		ConnectionConfig string
		RegionName       string
		MyImages         []*cres.MyImageInfo
		SlowNotices      []string
		APIUsername      string
		APIPassword      string
	}{
		ConnectionConfig: connConfig,
		RegionName:       regionName,
		MyImages:         myImages,
		APIUsername:      os.Getenv("SPIDER_USERNAME"),
		APIPassword:      os.Getenv("SPIDER_PASSWORD"),
	}
	if notice := slowMyImageNotice(connConfig); notice != "" {
		data.SlowNotices = []string{notice}
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/myimage.html")
	tmpl, err := template.New("myimage.html").Funcs(template.FuncMap{
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

// MyImageMultiRow pairs a MyImage with the connection it was fetched from,
// so a single table can list MyImages aggregated across several connections.
type MyImageMultiRow struct {
	ConnectionName string
	MyImage        *cres.MyImageInfo
}

// Handler function to render the MyImage management page across multiple connections at once.
// Each connection is fetched independently so that one failing connection does not block the
// others from being displayed (partial success).
func MyImageManagementMulti(c echo.Context) error {
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

	var rows []MyImageMultiRow
	connErrors := map[string]string{}

	for _, connName := range connNames {
		if isMultiAsyncLoad(c) {
			break
		}
		myImages, err := fetchMyImages(connName)
		if err != nil {
			connErrors[connName] = err.Error()
			continue
		}

		for _, myImage := range myImages {
			rows = append(rows, MyImageMultiRow{
				ConnectionName: connName,
				MyImage:        myImage,
			})
		}
	}

	// the slow-region notices (a few REST calls per connection, at the same time)
	slowNotices := make([]string, len(connNames))
	var wg sync.WaitGroup
	for i, connName := range connNames {
		wg.Add(1)
		go func(i int, connName string) {
			defer wg.Done()
			slowNotices[i] = slowMyImageNotice(connName)
		}(i, connName)
	}
	wg.Wait()
	var notices []string
	for _, notice := range slowNotices {
		if notice != "" {
			notices = append(notices, notice)
		}
	}

	data := struct {
		Connections []string
		MyImageRows []MyImageMultiRow
		ConnErrors  map[string]string
		SlowNotices []string
		APIUsername string
		APIPassword string
	}{
		Connections: connNames,
		MyImageRows: rows,
		ConnErrors:  connErrors,
		SlowNotices: notices,
		APIUsername: os.Getenv("SPIDER_USERNAME"),
		APIPassword: os.Getenv("SPIDER_PASSWORD"),
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/myimage-multi.html")
	tmpl, err := template.New("myimage-multi.html").Funcs(template.FuncMap{
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
