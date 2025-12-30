// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista

package adminweb

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	"github.com/labstack/echo/v4"
)

// SystemInfo structure mapping from the API response
type SystemInfo struct {
	CPUModel        string
	ClockSpeed      string
	DiskPartitions  []DiskPartitionInfo
	Hostname        string
	KernelArch      string
	KernelVersion   string
	LogicalCores    int
	PhysicalCores   int
	Platform        string
	PlatformVersion string
	SwapMemory      string
	TotalMemory     string
	Uptime          string
}

// DiskPartitionInfo structure for disk partition information
type DiskPartitionInfo struct {
	MountPoint string
	TotalSpace string
}

// ResourceUsage structure mapping from the API response
type ResourceUsage struct {
	ProcessCPUCorePercent map[string]string
	ProcessCPUPercent     string
	ProcessDiskRead       string
	ProcessDiskWrite      string
	ProcessMemoryPercent  string
	ProcessMemoryUsed     string
	ProcessName           string
	ProcessNetReceived    string
	ProcessNetSent        string
	SystemCPUCorePercent  map[string]string
	SystemCPUPercent      string
	SystemDiskRead        string
	SystemDiskWrite       string
	SystemMemoryPercent   string
	SystemMemoryTotal     string
	SystemMemoryUsed      string
	SystemNetReceived     string
	SystemNetSent         string
}

// Function to fetch System Information
func fetchSystemInfo() (SystemInfo, error) {
	url := "http://localhost:1024/spider/sysstats/system"
	resp, err := http.Get(url)
	if err != nil {
		return SystemInfo{}, fmt.Errorf("error fetching System Info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return SystemInfo{}, fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	var sysStats SystemInfo
	if err := json.NewDecoder(resp.Body).Decode(&sysStats); err != nil {
		return SystemInfo{}, fmt.Errorf("error decoding System Info: %v", err)
	}

	return sysStats, nil
}

// Function to fetch Resource Usage
func fetchResourceUsage() (ResourceUsage, error) {
	url := "http://localhost:1024/spider/sysstats/usage"
	resp, err := http.Get(url)
	if err != nil {
		return ResourceUsage{}, fmt.Errorf("error fetching Resource Usage: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ResourceUsage{}, fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	var resourceUsage ResourceUsage
	if err := json.NewDecoder(resp.Body).Decode(&resourceUsage); err != nil {
		return ResourceUsage{}, fmt.Errorf("error decoding Resource Usage: %v", err)
	}

	return resourceUsage, nil
}

// Handler function to render the Combined System Information and Resource Usage page
func SystemStatsInfoPage(c echo.Context) error {
	// Fetch system information
	sysStats, err := fetchSystemInfo()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error fetching system info: " + err.Error()})
	}

	// Fetch resource usage
	resourceUsage, err := fetchResourceUsage()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error fetching resource usage: " + err.Error()})
	}

	// Get sorted core keys for consistent display
	var systemCoreKeys []string
	for key := range resourceUsage.SystemCPUCorePercent {
		systemCoreKeys = append(systemCoreKeys, key)
	}
	sort.Strings(systemCoreKeys)

	var processCoreKeys []string
	for key := range resourceUsage.ProcessCPUCorePercent {
		processCoreKeys = append(processCoreKeys, key)
	}
	sort.Strings(processCoreKeys)

	data := struct {
		SystemInfo      SystemInfo
		ResourceUsage   ResourceUsage
		SystemCoreKeys  []string
		ProcessCoreKeys []string
		ShortStartTime  string
	}{
		SystemInfo:      sysStats,
		ResourceUsage:   resourceUsage,
		SystemCoreKeys:  systemCoreKeys,
		ProcessCoreKeys: processCoreKeys,
		ShortStartTime:  cr.StartTime,
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/system-stats.html")
	tmpl, err := template.New("system-stats.html").Funcs(template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
		"parsePercentage": func(percentStr string) float64 {
			// Remove the % sign and parse to float
			if percentStr == "" {
				return 0
			}
			percentValue := strings.TrimSuffix(percentStr, " %")
			value, err := strconv.ParseFloat(percentValue, 64)
			if err != nil {
				return 0
			}
			return value
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
