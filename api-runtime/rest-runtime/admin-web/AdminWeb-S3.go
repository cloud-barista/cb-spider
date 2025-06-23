// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2025.06.

package adminweb

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/labstack/echo/v4"
)

type S3BucketInfo struct {
	Name         string `json:"Name"`
	BucketRegion string `json:"BucketRegion,omitempty"`
	CreationDate string `json:"CreationDate"`
}
type S3ObjectInfo struct {
	ETag         string `json:"ETag"`
	Key          string `json:"Key"`
	LastModified string `json:"LastModified"`
	Size         int64  `json:"Size"`
	ContentType  string `json:"ContentType"`
}

func S3Management(c echo.Context) error {
	connConfig := c.Param("ConnectConfig")
	if connConfig == "region not set" {
		htmlStr := `
			<html>
			<head>
			    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
				<style>
				th { border: 1px solid lightgray; }
				td { border: 1px solid lightgray; border-radius: 4px; }
				</style>
			    <script type="text/javascript"> alert(connConfig) </script>
			</head>
			<body>
				<br><br>
				<label style="font-size:24px;color:#606262;">&nbsp;&nbsp;&nbsp;Please select a Connection Configuration! (MENU: 2.CONNECTION)</label>
			</body>
		`
		return c.HTML(http.StatusOK, htmlStr)
	}

	buckets, err := fetchS3Buckets(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	data := struct {
		ConnectionConfig string
		Buckets          []S3BucketInfo
	}{
		ConnectionConfig: connConfig,
		Buckets:          buckets,
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/s3.html")
	tmpl, err := template.New("s3.html").Funcs(template.FuncMap{
		"inc": func(i int) int { return i + 1 },
	}).ParseFiles(templatePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error loading template: " + err.Error()})
	}
	return tmpl.Execute(c.Response().Writer, data)

}

func fetchS3Buckets(connConfig string) ([]S3BucketInfo, error) {
	resp, err := http.Get(fmt.Sprintf("http://localhost:1024/spider/s3/bucket?ConnectionName=%s", connConfig))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var result []S3BucketInfo
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}
