// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2026.05.

package adminweb

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"github.com/labstack/echo/v4"
)

//====================================== Quota

func Quota(c echo.Context) error {
	cblog.Info("call Quota()")

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

	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "quota")
	if err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var quotaInfo cres.QuotaInfo
	json.Unmarshal(resBody, &quotaInfo)

	type PageData struct {
		CSP            string
		Region         string
		ResourceQuotas []cres.ResourceQuota
		KeyValueList   []cres.KeyValue
		LoggingUrl     template.JS
		LoggingResult  template.JS
	}

	data := PageData{
		CSP:            quotaInfo.CSP,
		Region:         quotaInfo.Region,
		ResourceQuotas: quotaInfo.ResourceQuotas,
		KeyValueList:   quotaInfo.KeyValueList,
		LoggingUrl:     template.JS(genLoggingGETURL2(connConfig, "quota")),
		LoggingResult:  template.JS(genLoggingResult2(string(resBody[:len(resBody)-1]))),
	}

	// Parse the HTML template
	tmplPath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/quota.html")
	tmpl, err := template.New("quota.html").Funcs(template.FuncMap{
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
