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
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

func productFamilyList() []string {
	return []string{"Select a Region first"}
}

func regionList() []string {
	return []string{"No regions available"}
}

// driverCapabilityPrice is the subset of DriverCapabilityInfo needed to tell whether this
// connection's CSP driver supports Price Info at all (e.g. OpenStack/NHN/KT do not).
type driverCapabilityPrice struct {
	PriceInfoHandler bool `json:"PriceInfoHandler"`
}

// isPriceInfoSupported reports whether connConfig's CSP driver supports Price Info, so the
// Price page can say so up front instead of only failing on Fetch. Fails open (true) if the
// capability lookup itself fails, so a hiccup there doesn't hide the page's real content.
func isPriceInfoSupported(connConfig string) bool {
	resBody, err := getResourceList_JsonByte("driver/capability?ConnectionName=" + url.QueryEscape(connConfig))
	if err != nil {
		cblog.Warnf("PriceInfoRequest: failed to load driver capability for %s: %v", connConfig, err)
		return true
	}
	var capability driverCapabilityPrice
	if err := json.Unmarshal(resBody, &capability); err != nil {
		cblog.Warnf("PriceInfoRequest: failed to parse driver capability for %s: %v", connConfig, err)
		return true
	}
	return capability.PriceInfoHandler
}

//====================================== PriceInfo Request

func PriceInfoRequest(c echo.Context) error {
	cblog.Info("call PriceInfoRequest()")

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

	providerName, _ := getProviderName(connConfig)

	type PageData struct {
		LoggingUrl         template.JS
		ConnectionName     string
		ProviderName       string
		RegionList         []string
		ProductFamilyList  []string
		LoggingResult      template.JS
		APIUsername        string
		APIPassword        string
		PriceInfoSupported bool
	}

	data := PageData{
		ConnectionName: connConfig,
		ProviderName:   providerName,
		// RegionList:        regionNameList,
		ProductFamilyList:  productFamilyList(),
		APIUsername:        os.Getenv("SPIDER_USERNAME"),
		PriceInfoSupported: isPriceInfoSupported(connConfig),
		APIPassword:        os.Getenv("SPIDER_PASSWORD"),
	}

	// Parse the HTML template
	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/priceinfo-request-template.html")
	tmpl, err := template.ParseFiles(templatePath)
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
