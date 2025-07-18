// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2025.06.

package adminweb

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
)

type S3BucketInfo struct {
	Name             string `json:"Name"`
	BucketRegion     string `json:"BucketRegion,omitempty"`
	CreationDate     string `json:"CreationDate"`
	VersioningStatus string `json:"VersioningStatus"`
	CORSStatus       string `json:"CORSStatus"`
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

	regionName, err := getRegionName(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	buckets, err := fetchS3Buckets(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	data := struct {
		ConnectionConfig string
		RegionName       string
		Buckets          []S3BucketInfo
	}{
		ConnectionConfig: connConfig,
		RegionName:       regionName,
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
	// Use root S3 API endpoint directly since /spider/ routing seems to have issues
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:1024/", nil)
	if err != nil {
		return nil, err
	}

	// Add ConnectionName as query parameter
	q := req.URL.Query()
	q.Add("ConnectionName", connConfig)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	// Debug: print response to understand what we're getting
	fmt.Printf("Response status: %d\n", resp.StatusCode)
	fmt.Printf("Response Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("Response body preview: %s\n", string(body[:min(len(body), 200)]))

	// Check if response is XML (should start with <?xml or <ListAllMyBucketsResult)
	bodyStr := string(body)
	if !strings.HasPrefix(bodyStr, "<?xml") && !strings.HasPrefix(bodyStr, "<ListAllMyBucketsResult") {
		// If we get HTML, it means the S3 API didn't recognize our request
		if strings.Contains(bodyStr, "<html>") {
			return nil, fmt.Errorf("S3 API endpoint not accessible with ConnectionName parameter. Try checking S3 routes configuration. Response: %s", bodyStr[:min(len(bodyStr), 200)])
		}
		return nil, fmt.Errorf("received non-XML response: %s", bodyStr[:min(len(bodyStr), 100)])
	}

	// Parse S3 standard XML response
	type ListAllMyBucketsResult struct {
		XMLName xml.Name `xml:"ListAllMyBucketsResult"`
		Buckets struct {
			Bucket []struct {
				Name         string `xml:"Name"`
				CreationDate string `xml:"CreationDate"`
			} `xml:"Bucket"`
		} `xml:"Buckets"`
	}

	var xmlResult ListAllMyBucketsResult
	if err := xml.Unmarshal(body, &xmlResult); err != nil {
		return nil, fmt.Errorf("failed to parse XML response: %v", err)
	}

	var result []S3BucketInfo
	for _, bucket := range xmlResult.Buckets.Bucket {
		versioningStatus := fetchVersioningStatus(connConfig, bucket.Name)
		corsStatus := fetchCORSStatus(connConfig, bucket.Name)
		result = append(result, S3BucketInfo{
			Name:             bucket.Name,
			CreationDate:     bucket.CreationDate,
			BucketRegion:     "", // Region info not available in standard S3 list buckets response
			VersioningStatus: versioningStatus,
			CORSStatus:       corsStatus,
		})
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func fetchVersioningStatus(connConfig, bucketName string) string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:1024/%s?versioning", bucketName), nil)
	if err != nil {
		return "Error"
	}

	q := req.URL.Query()
	q.Add("ConnectionName", connConfig)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return "Error"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "Suspended"
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "Error"
	}

	type VersioningConfiguration struct {
		Status string `xml:"Status"`
	}

	var versioningConfig VersioningConfiguration
	if err := xml.Unmarshal(body, &versioningConfig); err != nil {
		return "Error"
	}

	if versioningConfig.Status == "" {
		return "Suspended"
	}

	return versioningConfig.Status
}

func fetchCORSStatus(connConfig, bucketName string) string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:1024/%s?cors", bucketName), nil)
	if err != nil {
		return "Not configured"
	}

	q := req.URL.Query()
	q.Add("ConnectionName", connConfig)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return "Not configured"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "Not configured"
	}

	return "Configured"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
