// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2024.08.

package adminweb

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

// Function to fetch VMs
func fetchVMs(connConfig string) ([]*cres.VMInfo, error) {
	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "vm")
	if err != nil {
		return nil, fmt.Errorf("error fetching VMs: %v", err)
	}

	var info struct {
		ResultList []*cres.VMInfo `json:"vm"`
	}
	if err := json.Unmarshal(resBody, &info); err != nil {
		return nil, fmt.Errorf("error decoding VMs: %v", err)
	}

	sort.Slice(info.ResultList, func(i, j int) bool {
		return info.ResultList[i].IId.NameId < info.ResultList[j].IId.NameId
	})

	return info.ResultList, nil
}

func fetchAllVMStatuses(connConfig string) (map[string]string, error) {
	url := fmt.Sprintf("http://%s%s/spider/vmstatus", cr.ServiceIPorName, cr.ServicePort)
	reqBody := fmt.Sprintf(`{"ConnectionName": "%s"}`, connConfig)
	req, err := http.NewRequest("GET", url, strings.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	setBasicAuthIfConfigured(req)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching VM statuses: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get VM statuses")
	}

	var statusInfo struct {
		VMStatusList []cres.VMStatusInfo `json:"vmstatus"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&statusInfo); err != nil {
		return nil, fmt.Errorf("error decoding VM statuses: %v", err)
	}

	statusMap := make(map[string]string)
	for _, vmStatus := range statusInfo.VMStatusList {
		statusMap[vmStatus.IId.NameId] = string(vmStatus.VmStatus)
	}

	return statusMap, nil
}

type VMStatusMap map[string]string

func VMManagement(c echo.Context) error {
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

	regionName, err := getRegionName(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Get Region info to extract Region ID
	resBody, err := getResource_JsonByte("region", regionName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get region info: " + err.Error()})
	}
	var regionInfo map[string]interface{}
	if err := json.Unmarshal(resBody, &regionInfo); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse region info: " + err.Error()})
	}

	// Extract Region field from KeyValueInfoList
	region := ""
	if kvList, ok := regionInfo["KeyValueInfoList"].([]interface{}); ok {
		for _, kv := range kvList {
			if kvMap, ok := kv.(map[string]interface{}); ok {
				if key, keyOk := kvMap["Key"].(string); keyOk && (key == "Region" || key == "region") {
					if val, valOk := kvMap["Value"].(string); valOk {
						region = val
						break
					}
				}
			}
		}
	}

	vms, err := fetchVMs(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	vmStatuses, err := fetchAllVMStatuses(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	statusMap := make(VMStatusMap)
	for _, vm := range vms {
		statusMap[vm.IId.NameId] = vmStatuses[vm.IId.NameId]
	}

	data := struct {
		ConnectionConfig string
		RegionName       string
		Region           string
		VMs              []*cres.VMInfo
		VMStatusMap      VMStatusMap
		APIUsername      string
		APIPassword      string
	}{
		ConnectionConfig: connConfig,
		RegionName:       regionName,
		Region:           region,
		VMs:              vms,
		VMStatusMap:      statusMap,
		APIUsername:      os.Getenv("SPIDER_USERNAME"),
		APIPassword:      os.Getenv("SPIDER_PASSWORD"),
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/vm.html")
	tmpl, err := template.New("vm.html").Funcs(template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
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
