package adminweb

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	"github.com/labstack/echo/v4"
)

// ResourceCounts holds the counts for various resources
type ResourceCounts struct {
	ConnectionName       string `json:"connectionName"`
	VPCs                 int    `json:"vpcs"`
	Subnets              int    `json:"subnets"`
	SecurityGroups       int    `json:"securityGroups"`
	VMs                  int    `json:"vms"`
	KeyPairs             int    `json:"keyPairs"`
	Disks                int    `json:"disks"`
	NetworkLoadBalancers int    `json:"nlbs"`
	Clusters             int    `json:"clusters"`
	MyImages             int    `json:"myImages"`
}

// DashboardData aggregates the data for rendering the dashboard
type DashboardData struct {
	ServerIP           string
	TotalConnections   int
	ConnectionsByCloud map[string]int
	Providers          []string
	ResourceCounts     map[string][]ResourceCounts
	ShowEmpty          bool
}

// Add a function to filter out empty connections
func filterEmptyConnections(resourceCounts map[string][]ResourceCounts) map[string][]ResourceCounts {
	filteredCounts := make(map[string][]ResourceCounts)
	for provider, counts := range resourceCounts {
		var nonEmptyCounts []ResourceCounts
		for _, count := range counts {
			if count.VPCs > 0 || count.Subnets > 0 || count.SecurityGroups > 0 || count.VMs > 0 ||
				count.KeyPairs > 0 || count.Disks > 0 || count.NetworkLoadBalancers > 0 ||
				count.Clusters > 0 || count.MyImages > 0 {
				nonEmptyCounts = append(nonEmptyCounts, count)
			}
		}
		if len(nonEmptyCounts) > 0 {
			filteredCounts[provider] = nonEmptyCounts
		}
	}
	return filteredCounts
}

// // Fetch all providers
// func fetchProviders() ([]string, error) {
// 	resp, err := http.Get("http://localhost:1024/spider/cloudos")
// 	if err != nil {
// 		return nil, fmt.Errorf("error fetching providers: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	var providers Providers
// 	if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
// 		return nil, fmt.Errorf("error decoding providers: %v", err)
// 	}

// 	return providers.Providers, nil
// }

type CountResponse struct {
	Count int `json:"count"`
}

// Fetch resource counts using specific connection names
func fetchResourceCounts(config ConnectionConfig) (ResourceCounts, error) {
	var counts ResourceCounts
	counts.ConnectionName = config.ConfigName

	baseURL := "http://localhost:1024/spider"
	resources := []string{"vpc", "subnet", "securitygroup", "vm", "keypair", "disk", "nlb", "cluster", "myimage"}

	for _, resource := range resources {
		url := fmt.Sprintf("%s/count%s/%s", baseURL, resource, config.ConfigName)
		resp, err := http.Get(url)
		if err != nil {
			return counts, fmt.Errorf("error fetching %s count for %s: %v", resource, config.ConfigName, err)
		}
		defer resp.Body.Close()

		var response CountResponse
		if resp.StatusCode != http.StatusOK {
			return counts, fmt.Errorf("received non-OK status %d while fetching %s count for %s", resp.StatusCode, resource, config.ConfigName)
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return counts, fmt.Errorf("error decoding %s count for %s: %v", resource, config.ConfigName, err)
		}

		switch resource {
		case "vpc":
			counts.VPCs = response.Count
		case "subnet":
			counts.Subnets = response.Count
		case "securitygroup":
			counts.SecurityGroups = response.Count
		case "vm":
			counts.VMs = response.Count
		case "keypair":
			counts.KeyPairs = response.Count
		case "disk":
			counts.Disks = response.Count
		case "nlb":
			counts.NetworkLoadBalancers = response.Count
		case "cluster":
			counts.Clusters = response.Count
		case "myimage":
			counts.MyImages = response.Count
		}
	}
	return counts, nil
}

// Dashboard renders the dashboard page.
func Dashboard(c echo.Context) error {
	serverIP := cr.ServiceIPorName + cr.ServicePort // cr.ServicePort = ":1024"
	if serverIP == "" {
		serverIP = "localhost"
	}

	showEmpty := c.QueryParam("showEmpty") == "true"

	providers, err := fetchProviders()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	connectionConfigs, err := fetchConnectionConfigs()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	resourceCounts := make(map[string][]ResourceCounts)
	for provider, configs := range connectionConfigs {
		for _, config := range configs {
			counts, err := fetchResourceCounts(config)
			if err != nil {
				continue // Optionally handle error
			}
			resourceCounts[provider] = append(resourceCounts[provider], counts)
		}
	}

	if !showEmpty {
		resourceCounts = filterEmptyConnections(resourceCounts)
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/dashboard.html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error loading template"})
	}

	data := DashboardData{
		ServerIP:       serverIP,
		Providers:      providers,
		ResourceCounts: resourceCounts,
		ShowEmpty:      showEmpty,
	}

	return tmpl.Execute(c.Response().Writer, data)
}
