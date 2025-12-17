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
	"time"

	"github.com/labstack/echo/v4"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

var PRICEINFO_CACHE_PATH string = os.Getenv("CBSPIDER_ROOT") + "/cache/priceinfo"

// TemplateData structure to render the HTML template
type TemplateData struct {
	Data           cres.CloudPrice
	CachedFileName string
	TotalItems     int
	SimpleMode     string
}

func init() {
	// Create cache directory for PriceInfo if not exists
	_, err := os.Stat(PRICEINFO_CACHE_PATH)
	if os.IsNotExist(err) {
		err := os.MkdirAll(PRICEINFO_CACHE_PATH, 0755)
		if err != nil {
			cblog.Fatal(err)
			return
		}
	}
}

//====================================== PriceInfo Table List

// Sometimes, priceInfo is huge to be displayed in a browser.
// In such cases, it is better to save the data to a file and provide a link to download the file.
func saveJSONToFile(data interface{}) (string, error) {
	// Delete old JSON files
	deleteOldJSONFiles()

	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return "", err
	}

	cloudName := data.(cres.CloudPrice).CloudName
	fileName := cloudName + "_price_info_" + time.Now().Format("2006-01-02_15-04-05") + ".json"
	cacheFilePath := filepath.Join(PRICEINFO_CACHE_PATH, fileName)
	if err := ioutil.WriteFile(cacheFilePath, jsonData, 0644); err != nil {
		return "", err
	}

	cblog.Info("PriceInfo JSON data saved to: " + cacheFilePath)
	return fileName, nil
}

func deleteOldJSONFiles() {
	files, err := ioutil.ReadDir(PRICEINFO_CACHE_PATH)
	if err != nil {
		cblog.Error("Failed to list files in directory:", PRICEINFO_CACHE_PATH, err)
		return
	}

	oneDayAgo := time.Now().Add(-24 * time.Hour)

	for _, file := range files {
		if file.IsDir() {
			continue // ignore directories
		}

		if filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join(PRICEINFO_CACHE_PATH, file.Name())
			if file.ModTime().Before(oneDayAgo) {
				err := os.Remove(filePath)
				if err != nil {
					cblog.Error("Failed to delete old JSON file:", filePath, err)
				} else {
					cblog.Info("Deleted old JSON file:", filePath)
				}
			}
		}
	}
}

// PriceInfoTableList handles the display of CloudPrice to Table
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

	// Handle simplemode parameter
	simpleMode := c.QueryParam("simplemode")
	simpleVMSpecInfo := false
	if simpleMode == "ON" {
		simpleVMSpecInfo = true
	}

	var data cres.CloudPrice
	err := getPriceInfoJsonString(connConfig, "priceinfo", c.Param("ProductFamily"), c.Param("RegionName"), nil, simpleVMSpecInfo, &data)
	if err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var cachedFileName string
	if cachedFileName, err = saveJSONToFile(data); err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save JSON file: "+err.Error())
	}

	// Set SimpleMode based on query parameter
	currentSimpleMode := "OFF" // default value
	if simpleVMSpecInfo {
		currentSimpleMode = "ON"
	}

	tmplData := TemplateData{
		CachedFileName: cachedFileName,
		TotalItems:     len(data.PriceList),
		SimpleMode:     currentSimpleMode,
	}

	// Debug logging
	cblog.Infof("TemplateData created: SimpleMode=%s, TotalItems=%d", tmplData.SimpleMode, tmplData.TotalItems)

	// Limit PriceList items to 200
	if len(data.PriceList) > 200 {
		data.PriceList = data.PriceList[:200]
	}

	tmplData.Data = data

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/priceinfo-tablelist-template.html")
	cloudPriceTemplate := getHtmlTemplate(templatePath)
	if cloudPriceTemplate == "" {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load HTML template")
	}

	tmpl, err := addTemplateFuncs(template.New("cloudPrice"), currentSimpleMode).Parse(cloudPriceTemplate)
	if err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Execute the template with tmplData
	var result bytes.Buffer
	err = tmpl.Execute(&result, tmplData)
	if err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.HTML(http.StatusOK, result.String())
}

// getHtmlTemplate reads HTML template from a file.
func getHtmlTemplate(filepath string) string {
	file, err := os.Open(filepath)
	if err != nil {
		cblog.Error(err)
		return ""
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		cblog.Error(err)
		return ""
	}

	return string(content)
}

func addTemplateFuncs(t *template.Template, simpleMode string) *template.Template {
	return t.Funcs(template.FuncMap{
		"json": func(v interface{}) string {
			a, _ := json.MarshalIndent(v, "", "    ")
			return string(a)
		},

		"inc": func(i int) int {
			return i + 1
		},

		"simpleMode": func() string {
			return simpleMode
		},
	})
}

func DownloadPriceInfo(c echo.Context) error {
	fileName := c.Param("FileName")
	filePath := filepath.Join(PRICEINFO_CACHE_PATH, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		cblog.Error(err)
		return c.NoContent(http.StatusNotFound)
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename="+fileName)
	c.Response().Header().Set("Content-Type", "application/octet-stream")

	return c.File(filePath)
}
