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
	"net/http"
	"os"
	"path/filepath"
	"sort"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

// fetchNICs fetches the list of NICs from the Spider API.
func fetchNICs(connConfig string) ([]*cres.NICInfo, error) {
	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "nic")
	if err != nil {
		return nil, fmt.Errorf("error fetching NICs: %v", err)
	}

	var info struct {
		ResultList []*cres.NICInfo `json:"nic"`
	}
	if err := json.Unmarshal(resBody, &info); err != nil {
		return nil, fmt.Errorf("error decoding NICs: %v", err)
	}

	sort.Slice(info.ResultList, func(i, j int) bool {
		return info.ResultList[i].IId.NameId < info.ResultList[j].IId.NameId
	})

	return info.ResultList, nil
}

// NICManagement renders the NIC management page.
func NICManagement(c echo.Context) error {
	connConfig := c.Param("ConnectConfig")
	if connConfig == "region not set" {
		htmlStr := `
			<html><head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8" /></head>
			<body><br><br>
			<label style="font-size:24px;color:#606262;">&nbsp;&nbsp;&nbsp;Please select a Connection Configuration! (MENU: 2.CONNECTION)</label>
			</body></html>`
		return c.HTML(http.StatusOK, htmlStr)
	}

	regionName, err := getRegionName(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	nics, err := fetchNICs(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	vms, _ := fetchVMs(connConfig)

	// Build publicIPAddrToName: IP address → Spider PublicIP NameId
	// Avoids browser GET+body limitation by pre-computing the map server-side.
	publicIPs, _ := fetchPublicIPs(connConfig)
	publicIPAddrToName := make(map[string]string)
	for _, pip := range publicIPs {
		if pip.PublicIPAddress != "" && pip.IId.NameId != "" {
			publicIPAddrToName[pip.PublicIPAddress] = pip.IId.NameId
		}
	}

	providerName, _ := getProviderName(connConfig)

	// Build NIC data JSON for OS Config Guide (JavaScript)
	type nicOSData struct {
		Name         string   `json:"name"`
		MAC          string   `json:"mac"`
		PrivateIPs   []string `json:"privateIPs"`
		Status       string   `json:"status"`
		OwnerVM      string   `json:"ownerVM"`
		OwnerVMPubIP string   `json:"ownerVMPubIP"`
	}
	var nicOSDataList []nicOSData
	for _, nic := range nics {
		pubIP := ""
		if len(nic.PublicIPs) > 0 {
			pubIP = nic.PublicIPs[0]
		}
		privIPs := nic.PrivateIPs
		if privIPs == nil {
			privIPs = []string{}
		}
		nicOSDataList = append(nicOSDataList, nicOSData{
			Name:         nic.IId.NameId,
			MAC:          nic.MACAddress,
			PrivateIPs:   privIPs,
			Status:       string(nic.Status),
			OwnerVM:      nic.OwnerVM.NameId,
			OwnerVMPubIP: pubIP,
		})
	}
	nicDataJSON, _ := json.Marshal(nicOSDataList)

	data := struct {
		ConnectionConfig   string
		RegionName         string
		NICs               []*cres.NICInfo
		VMs                []*cres.VMInfo
		PublicIPAddrToName map[string]string
		ProviderName       string
		NICDataJSON        template.JS
		APIUsername        string
		APIPassword        string
	}{
		ConnectionConfig:   connConfig,
		RegionName:         regionName,
		NICs:               nics,
		VMs:                vms,
		PublicIPAddrToName: publicIPAddrToName,
		ProviderName:       providerName,
		NICDataJSON:        template.JS(nicDataJSON),
		APIUsername:        os.Getenv("SPIDER_USERNAME"),
		APIPassword:        os.Getenv("SPIDER_PASSWORD"),
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/nic.html")
	tmpl, err := template.New("nic.html").Funcs(template.FuncMap{
		"inc": func(i int) int { return i + 1 },
		"add": func(a, b int) int { return a + b },
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
