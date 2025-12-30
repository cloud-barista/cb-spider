// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2025.12.

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
	"strings"

	"github.com/labstack/echo/v4"
)

type FileSystemInfo struct {
	IId            IID        `json:"IId"`
	Name           string     `json:"Name"`
	Zone           string     `json:"Zone"`
	VpcIID         IID        `json:"VpcIID"`
	FileSystemType string     `json:"FileSystemType"`
	Status         string     `json:"Status"`
	CreatedTime    string     `json:"CreatedTime"`
	AccessSubnets  []IID      `json:"AccessSubnets,omitempty"`
	KeyValueList   []KeyValue `json:"KeyValueList,omitempty"`
}

type IID struct {
	NameId   string `json:"NameId"`
	SystemId string `json:"SystemId"`
}

type KeyValue struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

func FileSystemManagement(c echo.Context) error {
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

	fileSystems, err := fetchFileSystems(connConfig)
	var errorMessage string
	if err != nil {
		// Set error message but don't return error - show empty table with error message
		errorMessage = err.Error()
		fileSystems = []FileSystemInfo{} // Empty list
	}

	data := struct {
		ConnectionConfig string
		RegionName       string
		FileSystems      []FileSystemInfo
		ErrorMessage     string
	}{
		ConnectionConfig: connConfig,
		RegionName:       regionName,
		FileSystems:      fileSystems,
		ErrorMessage:     errorMessage,
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/filesystem.html")
	tmpl, err := template.New("filesystem.html").Funcs(template.FuncMap{
		"inc":   func(i int) int { return i + 1 },
		"lower": func(s string) string { return strings.ToLower(s) },
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

func fetchFileSystems(connConfig string) ([]FileSystemInfo, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:1024/spider/filesystem", nil)
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch FileSystems: HTTP %d - %s", resp.StatusCode, string(body))
	}

	type ListResponse struct {
		FileSystem []FileSystemInfo `json:"filesystem"`
	}

	var listResp ListResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		// Try direct array format
		var result []FileSystemInfo
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %v", err)
		}
		sort.Slice(result, func(i, j int) bool { return result[i].IId.NameId < result[j].IId.NameId })
		return result, nil
	}

	result := listResp.FileSystem
	sort.Slice(result, func(i, j int) bool { return result[i].IId.NameId < result[j].IId.NameId })
	return result, nil
}
