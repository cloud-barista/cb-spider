// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2024.07.

package adminweb

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"github.com/labstack/echo/v4"
)

//====================================== VMImage

func VMImage(c echo.Context) error {
	cblog.Info("call VMImage()")

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

	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "vmimage")
	if err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var info struct {
		ResultList []*cres.ImageInfo `json:"image"`
	}
	json.Unmarshal(resBody, &info)

	data := struct {
		ConnConfig string
		Images     []*cres.ImageInfo
	}{
		ConnConfig: connConfig,
		Images:     info.ResultList,
	}

	tmplPath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/vm-image.html")
	tmpl, err := template.New("vm-image.html").Funcs(template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	}).ParseFiles(tmplPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var result strings.Builder
	err = tmpl.Execute(&result, data)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.HTML(http.StatusOK, result.String())
}
