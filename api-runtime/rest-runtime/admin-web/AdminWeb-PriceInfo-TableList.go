// Cloud Price Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2024.01.

package adminweb

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

//====================================== PriceInfo Table List

// PriceInfoTableList handles the display of CloudPriceData to Table
func PriceInfoTableList(c echo.Context) error {
	cblog.Info("call PriceInfoTableList()")

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

	var req struct {
		// FilterList []cres.KeyValue
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var info struct {
		FilterList []cres.KeyValue `json:"FilterList"`
	}

	json.Unmarshal([]byte(c.QueryParam("filterlist")), &info)

	var data cres.CloudPriceData
	// err := getPriceInfoJsonString(connConfig, "priceinfo", c.Param("ProductFamily"), c.Param("RegionName"), req.FilterList, &data)
	err := getPriceInfoJsonString(connConfig, "priceinfo", c.Param("ProductFamily"), c.Param("RegionName"), info.FilterList, &data)
	if err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/priceinfo-tablelist-template.html")
	cloudPriceTemplate := getHtmlTemplate(templatePath)
	if cloudPriceTemplate == "" {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load HTML template")
	}

	tmpl, err := addTemplateFuncs(template.New("cloudPrice")).Parse(cloudPriceTemplate)
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

// getHtmlTemplate reads HTML template from a file.
func getHtmlTemplate(filepath string) string {
	file, err := os.Open(filepath)
	if err != nil {
		return ""
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return ""
	}

	return string(content)
}

func addTemplateFuncs(t *template.Template) *template.Template {
	return t.Funcs(template.FuncMap{
		"json": func(v interface{}) string {
			a, _ := json.MarshalIndent(v, "", "    ")
			return string(a)
		},
	})
}
