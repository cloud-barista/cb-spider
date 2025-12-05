// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2024.05.

package adminweb

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	"github.com/labstack/echo/v4"
)

// ResourceCounts holds the counts for various resources
type ResourceCounts struct {
	ConnectionName       string `json:"connectionName"`
	RegionName           string `json:"regionName"`
	VPCs                 int    `json:"vpcs"`
	Subnets              int    `json:"subnets"`
	SecurityGroups       int    `json:"securityGroups"`
	VMs                  int    `json:"vms"`
	KeyPairs             int    `json:"keyPairs"`
	Disks                int    `json:"disks"`
	NetworkLoadBalancers int    `json:"nlbs"`
	Clusters             int    `json:"clusters"`
	MyImages             int    `json:"myImages"`
	S3Buckets            int    `json:"s3Buckets"`
}

// DashboardData aggregates the data for rendering the dashboard
type DashboardData struct {
	ServerIP           string
	TotalConnections   int
	ConnectionsByCloud map[string]int
	Providers          []string
	ResourceCounts     map[string][]ResourceCounts
	Regions            map[string]string
	ShowEmpty          bool
}

// Filter out empty connections
func filterEmptyConnections(resourceCounts map[string][]ResourceCounts) map[string][]ResourceCounts {
	filteredCounts := make(map[string][]ResourceCounts)
	for provider, counts := range resourceCounts {
		var nonEmptyCounts []ResourceCounts
		for _, count := range counts {
			if count.VPCs > 0 || count.Subnets > 0 || count.SecurityGroups > 0 || count.VMs > 0 ||
				count.KeyPairs > 0 || count.Disks > 0 || count.NetworkLoadBalancers > 0 ||
				count.Clusters > 0 || count.MyImages > 0 || count.S3Buckets > 0 {
				nonEmptyCounts = append(nonEmptyCounts, count)
			}
		}
		if len(nonEmptyCounts) > 0 {
			filteredCounts[provider] = nonEmptyCounts
		}
	}
	return filteredCounts
}

type CountResponse struct {
	Count int `json:"count"`
}

// Fetch resource counts using specific connection names
func fetchResourceCounts(config ConnectionConfig, provider string, wg *sync.WaitGroup, countsChan chan<- struct {
	Provider string
	Counts   ResourceCounts
}, errorChan chan<- error) {
	defer wg.Done()

	var counts ResourceCounts
	counts.ConnectionName = config.ConfigName
	counts.RegionName = config.RegionName

	baseURL := "http://localhost:1024/spider"
	resources := []string{"vpc", "subnet", "securitygroup", "vm", "keypair", "disk", "nlb", "cluster", "myimage", "s3"}

	for _, resource := range resources {
		url := fmt.Sprintf("%s/count%s/%s", baseURL, resource, config.ConfigName)
		resp, err := http.Get(url)
		if err != nil {
			errorChan <- fmt.Errorf("error fetching %s count for %s: %v", resource, config.ConfigName, err)
			return
		}
		defer resp.Body.Close()

		var response CountResponse
		if resp.StatusCode != http.StatusOK {
			errorChan <- fmt.Errorf("received non-OK status %d while fetching %s count for %s", resp.StatusCode, resource, config.ConfigName)
			return
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			errorChan <- fmt.Errorf("error decoding %s count for %s: %v", resource, config.ConfigName, err)
			return
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
		case "s3":
			counts.S3Buckets = response.Count
		}
	}
	countsChan <- struct {
		Provider string
		Counts   ResourceCounts
	}{Provider: provider, Counts: counts}
}

// Template cache
var tmplCache *template.Template

func init() {
	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/dashboard.html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		panic(fmt.Errorf("error loading template: %v", err))
	}
	tmplCache = tmpl
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
	var wg sync.WaitGroup
	countsChan := make(chan struct {
		Provider string
		Counts   ResourceCounts
	}, len(connectionConfigs))
	errorChan := make(chan error, len(connectionConfigs))

	for provider, configs := range connectionConfigs {
		for _, config := range configs {
			wg.Add(1)
			go fetchResourceCounts(config, provider, &wg, countsChan, errorChan)
		}
	}

	go func() {
		wg.Wait()
		close(countsChan)
		close(errorChan)
	}()

	for result := range countsChan {
		resourceCounts[result.Provider] = append(resourceCounts[result.Provider], result.Counts)
	}

	// Handle errors
	for err := range errorChan {
		fmt.Println("Error:", err)
	}

	if !showEmpty {
		resourceCounts = filterEmptyConnections(resourceCounts)
	}

	// Sort the resource counts for each provider by ConnectionName
	for provider := range resourceCounts {
		sort.Slice(resourceCounts[provider], func(i, j int) bool {
			return resourceCounts[provider][i].ConnectionName < resourceCounts[provider][j].ConnectionName
		})
	}

	regionMap, err := fetchRegions()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	data := DashboardData{
		ServerIP:       serverIP,
		Providers:      providers,
		ResourceCounts: resourceCounts,
		Regions:        regionMap,
		ShowEmpty:      showEmpty,
	}

	return tmplCache.Execute(c.Response().Writer, data)
}
