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
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"

	"github.com/labstack/echo/v4"
)

//====================================== Quota

// Quota renders the quota shell page (service type selector + AJAX-driven table).
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

	type PageData struct {
		ConnConfig string
		ServerPort string
	}

	data := PageData{
		ConnConfig: connConfig,
		ServerPort: cr.ServerPort,
	}

	// Parse the HTML template
	tmplPath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/quota-info.html")
	tmpl, err := template.New("quota-info.html").Funcs(template.FuncMap{
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

// QuotaServiceTypeList returns the service type list and provider name as JSON for AJAX calls.
func QuotaServiceTypeList(c echo.Context) error {
	cblog.Info("call QuotaServiceTypeList()")

	connConfig := c.Param("ConnectConfig")

	url := "http://localhost" + cr.ServerPort + "/spider/quotaservicetype?ConnectionName=" + connConfig
	resp, err := httpGetWithAuth(url)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	// Enrich the response with ProviderName from the connection config so that
	// the UI can highlight CSP-specific service types.
	var payload map[string]interface{}
	if json.Unmarshal(body, &payload) == nil {
		ccURL := "http://localhost" + cr.ServerPort + "/spider/connectionconfig/" + connConfig
		if ccResp, ccErr := httpGetWithAuth(ccURL); ccErr == nil {
			defer ccResp.Body.Close()
			var ccInfo ConnectionConfig
			if json.NewDecoder(ccResp.Body).Decode(&ccInfo) == nil {
				payload["ProviderName"] = ccInfo.ProviderName
			}
		}
		if merged, mergeErr := json.Marshal(payload); mergeErr == nil {
			return c.Blob(resp.StatusCode, echo.MIMEApplicationJSONCharsetUTF8, merged)
		}
	}

	return c.Blob(resp.StatusCode, echo.MIMEApplicationJSONCharsetUTF8, body)
}

// QuotaList returns the quota info for a service type as JSON for AJAX calls.
func QuotaList(c echo.Context) error {
	cblog.Info("call QuotaList()")

	connConfig := c.Param("ConnectConfig")
	serviceType := c.Param("ServiceType")

	url := "http://localhost" + cr.ServerPort + "/spider/quotainfo?ConnectionName=" + connConfig + "&ServiceType=" + serviceType
	resp, err := httpGetWithAuth(url)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	return c.Blob(resp.StatusCode, echo.MIMEApplicationJSONCharsetUTF8, body)
}
