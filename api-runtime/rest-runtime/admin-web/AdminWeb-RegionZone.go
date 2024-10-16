// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2024.07.

package adminweb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"github.com/labstack/echo/v4"
)

//====================================== RegionZone

func RegionZone(c echo.Context) error {
	cblog.Info("call RegionZone()")

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

	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "regionzone")
	if err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var info struct {
		ResultList []*cres.RegionZoneInfo `json:"regionzone"`
	}
	json.Unmarshal(resBody, &info)

	// struct for HTML template
	type ZoneInfo struct {
		ZoneName    string
		DisplayName string
		ZoneStatus  string
		IsDefault   bool
	}

	type RegionInfo struct {
		RegionName   string
		DisplayName  string
		InnerTableID string
		ZoneInfo     []ZoneInfo
	}

	type PageData struct {
		LoggingUrl    template.JS
		RegionInfo    []*RegionInfo
		LoggingResult template.JS
	}

	var regionInfos []*RegionInfo
	regionZoneInfos := info.ResultList
	for idx, rzInfo := range regionZoneInfos {
		rInfo := &RegionInfo{
			RegionName:   rzInfo.Name,
			DisplayName:  rzInfo.DisplayName,
			InnerTableID: fmt.Sprintf("%s-%d", rzInfo.Name, idx),
		}

		for i, zone := range rzInfo.ZoneList {
			isDefault := i == 0 // Only the first row is true, for the default zone
			rInfo.ZoneInfo = append(rInfo.ZoneInfo, ZoneInfo{
				ZoneName:    zone.Name,
				DisplayName: zone.DisplayName,
				ZoneStatus:  string(zone.Status),
				IsDefault:   isDefault,
			})
		}
		regionInfos = append(regionInfos, rInfo)
	}

	data := PageData{
		LoggingUrl:    template.JS(genLoggingGETURL2(connConfig, "regionzone")),
		RegionInfo:    regionInfos,
		LoggingResult: template.JS(genLoggingResult2(string(resBody[:len(resBody)-1]))),
	}

	// Parse the HTML template
	tmplPath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/region-zone.html")
	tmpl, err := template.New("region-zone.html").Funcs(template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	}).ParseFiles(tmplPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Execute the template with data
	var result bytes.Buffer
	err = tmpl.Execute(&result, data)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.HTML(http.StatusOK, result.String())
}

func addFuncsToTemplate(t *template.Template) *template.Template {
	return t.Funcs(template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	})
}
