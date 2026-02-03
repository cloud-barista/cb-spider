// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// MC-Insight API Proxy for VM Image and Spec metadata
// by CB-Spider Team, 2025.12.

package restruntime

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/labstack/echo/v4"
)

const (
	MC_INSIGHT_API_BASE = "http://mc-insight.cloud-barista.org:8000"
)

func getMCInsightAPIToken() string {
	return os.Getenv("MC_INSIGHT_API_TOKEN")
}

// ================ MC-Insight Proxy Handlers

// proxyMcInsightVMImageFilters godoc
// @ID proxy-mcinsight-vmimage-filters
// @Summary Get VM Image Filters from MC-Insight
// @Description Proxy endpoint to retrieve VM Image filters from MC-Insight API
// @Tags [Cloud Metadata] MC-Insight
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]interface{} "Filter hierarchy data"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /mcinsight/vm-image/filters [get]
func ProxyMcInsightVMImageFilters(c echo.Context) error {
	cblog.Info("call ProxyMcInsightVMImageFilters()")

	token := getMCInsightAPIToken()
	if token == "" {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "MC-Insight API token is not configured. Please set MC_INSIGHT_API_TOKEN in setup.env")
	}

	apiURL := fmt.Sprintf("%s/api/v1/vm-image/filters", MC_INSIGHT_API_BASE)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to create request: %v", err))
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to fetch from MC-Insight: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to read response: %v", err))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to parse response: %v", err))
	}

	return c.JSON(http.StatusOK, result)
}

// proxyMcInsightVMImage godoc
// @ID proxy-mcinsight-vmimage
// @Summary Get VM Images from MC-Insight
// @Description Proxy endpoint to retrieve VM Image list from MC-Insight API with filters
// @Tags [Cloud Metadata] MC-Insight
// @Accept  json
// @Produce  json
// @Param csp query string true "CSP name (aws, gcp, azure, etc.)"
// @Param region query string false "Region filter"
// @Param zone query string false "Zone filter"
// @Param image_id query string false "Image ID filter (partial match)"
// @Param image_name query string false "Image name filter (partial match)"
// @Param os_architecture query string false "OS Architecture filter"
// @Param os_platform query string false "OS Platform filter"
// @Param os_distribution query string false "OS Distribution filter (partial match)"
// @Param os_disk_type query string false "OS Disk Type filter"
// @Param image_status query string false "Image Status filter"
// @Param min_os_disk_size_gb query int false "Minimum OS disk size in GB"
// @Param max_os_disk_size_gb query int false "Maximum OS disk size in GB"
// @Param key_value_json_search_text query string false "Search text in key_value_json field"
// @Param sort query string false "Sort fields (e.g., 'name' or '-name' for descending). Can be used multiple times."
// @Param skip query int false "Number of records to skip for pagination"
// @Param limit query int false "Maximum number of records to return"
// @Success 200 {object} map[string]interface{} "VM Image list data"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /mcinsight/vm-image [get]
func ProxyMcInsightVMImage(c echo.Context) error {
	cblog.Info("call ProxyMcInsightVMImage()")

	token := getMCInsightAPIToken()
	if token == "" {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "MC-Insight API token is not configured. Please set MC_INSIGHT_API_TOKEN in setup.env")
	}

	// Build query parameters
	params := url.Values{}
	queryParams := []string{"csp", "region", "zone", "image_id", "image_name", "os_architecture", "os_platform", "os_distribution", "os_disk_type", "image_status", "min_os_disk_size_gb", "max_os_disk_size_gb", "key_value_json_search_text", "skip", "limit"}

	for _, param := range queryParams {
		if value := c.QueryParam(param); value != "" {
			params.Set(param, value)
		}
	}

	// Handle multiple sort parameters
	if sortParams := c.QueryParams()["sort"]; len(sortParams) > 0 {
		for _, sortParam := range sortParams {
			if sortParam != "" {
				params.Add("sort", sortParam)
			}
		}
	}

	apiURL := fmt.Sprintf("%s/api/v1/vm-image/?%s", MC_INSIGHT_API_BASE, params.Encode())

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to create request: %v", err))
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to fetch from MC-Insight: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to read response: %v", err))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to parse response: %v", err))
	}

	return c.JSON(http.StatusOK, result)
}

// proxyMcInsightVMSpecFilters godoc
// @ID proxy-mcinsight-vmspec-filters
// @Summary Get VM Spec Filters from MC-Insight
// @Description Proxy endpoint to retrieve VM Spec filters from MC-Insight API
// @Tags [Cloud Metadata] MC-Insight
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]interface{} "Filter hierarchy data"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /mcinsight/vm-spec/filters [get]
func ProxyMcInsightVMSpecFilters(c echo.Context) error {
	cblog.Info("call ProxyMcInsightVMSpecFilters()")

	token := getMCInsightAPIToken()
	if token == "" {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "MC-Insight API token is not configured. Please set MC_INSIGHT_API_TOKEN in setup.env")
	}

	apiURL := fmt.Sprintf("%s/api/v1/vm-spec/filters", MC_INSIGHT_API_BASE)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to create request: %v", err))
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to fetch from MC-Insight: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to read response: %v", err))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to parse response: %v", err))
	}

	return c.JSON(http.StatusOK, result)
}

// proxyMcInsightVMSpec godoc
// @ID proxy-mcinsight-vmspec
// @Summary Get VM Specs from MC-Insight
// @Description Proxy endpoint to retrieve VM Spec list from MC-Insight API with filters
// @Tags [Cloud Metadata] MC-Insight
// @Accept  json
// @Produce  json
// @Param csp query string true "CSP name (aws, gcp, azure, etc.)"
// @Param name query string false "Spec name filter (partial match)"
// @Param region query string false "Region filter"
// @Param zone query string false "Zone filter"
// @Param min_vcpu_count query int false "Minimum vCPU count"
// @Param max_vcpu_count query int false "Maximum vCPU count"
// @Param min_mem_size_mib query int false "Minimum memory size in MiB"
// @Param max_mem_size_mib query int false "Maximum memory size in MiB"
// @Param min_vcpu_clock_ghz query number false "Minimum vCPU clock speed in GHz"
// @Param max_vcpu_clock_ghz query number false "Maximum vCPU clock speed in GHz"
// @Param min_disk_size_gb query int false "Minimum disk size in GB"
// @Param max_disk_size_gb query int false "Maximum disk size in GB"
// @Param min_gpu_count query int false "Minimum GPU count"
// @Param max_gpu_count query int false "Maximum GPU count"
// @Param min_gpu_mem_size_gb query int false "Minimum GPU memory size in GB"
// @Param max_gpu_mem_size_gb query int false "Maximum GPU memory size in GB"
// @Param gpu_mfr query string false "GPU manufacturer filter"
// @Param gpu_model query string false "GPU model filter"
// @Param min_gpu_total_mem_size_gb query int false "Minimum GPU total memory size in GB"
// @Param max_gpu_total_mem_size_gb query int false "Maximum GPU total memory size in GB"
// @Param search_text query string false "Search text in key_value_json field"
// @Param sort query string false "Sort fields (e.g., 'name' or '-name' for descending). Can be used multiple times."
// @Param skip query int false "Number of records to skip for pagination"
// @Param limit query int false "Maximum number of records to return"
// @Success 200 {object} map[string]interface{} "VM Spec list data"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /mcinsight/vm-spec [get]
func ProxyMcInsightVMSpec(c echo.Context) error {
	cblog.Info("call ProxyMcInsightVMSpec()")

	token := getMCInsightAPIToken()
	if token == "" {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "MC-Insight API token is not configured. Please set MC_INSIGHT_API_TOKEN in setup.env")
	}

	// Build query parameters
	params := url.Values{}
	queryParams := []string{"csp", "name", "region", "zone", "min_vcpu_count", "max_vcpu_count", "min_mem_size_mib", "max_mem_size_mib", "min_vcpu_clock_ghz", "max_vcpu_clock_ghz", "min_disk_size_gb", "max_disk_size_gb", "min_gpu_count", "max_gpu_count", "min_gpu_mem_size_gb", "max_gpu_mem_size_gb", "gpu_mfr", "gpu_model", "min_gpu_total_mem_size_gb", "max_gpu_total_mem_size_gb", "search_text", "skip", "limit"}

	for _, param := range queryParams {
		if value := c.QueryParam(param); value != "" {
			params.Set(param, value)
		}
	}

	// Handle multiple sort parameters
	if sortParams := c.QueryParams()["sort"]; len(sortParams) > 0 {
		for _, sortParam := range sortParams {
			if sortParam != "" {
				params.Add("sort", sortParam)
			}
		}
	}

	apiURL := fmt.Sprintf("%s/api/v1/vm-spec/?%s", MC_INSIGHT_API_BASE, params.Encode())

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to create request: %v", err))
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to fetch from MC-Insight: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to read response: %v", err))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to parse response: %v", err))
	}

	return c.JSON(http.StatusOK, result)
}

// ================ VM Price Info Handlers

// proxyMcInsightVMPriceFilters godoc
// @ID proxy-mcinsight-vmprice-filters
// @Summary Get VM Price Filters from MC-Insight
// @Description Proxy endpoint to retrieve VM Price filters from MC-Insight API
// @Tags [Cloud Metadata] MC-Insight
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]interface{} "Filter hierarchy data"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /mcinsight/price-info/filters [get]
func ProxyMcInsightVMPriceFilters(c echo.Context) error {
	cblog.Info("call ProxyMcInsightVMPriceFilters()")

	token := getMCInsightAPIToken()
	if token == "" {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "MC-Insight API token is not configured. Please set MC_INSIGHT_API_TOKEN in setup.env")
	}

	apiURL := fmt.Sprintf("%s/api/v1/price-info/filters", MC_INSIGHT_API_BASE)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to create request: %v", err))
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to fetch from MC-Insight: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to read response: %v", err))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to parse response: %v", err))
	}

	return c.JSON(http.StatusOK, result)
}

// proxyMcInsightVMPrice godoc
// @ID proxy-mcinsight-vmprice
// @Summary Get VM Price Info from MC-Insight
// @Description Proxy endpoint to retrieve VM Price information from MC-Insight API with filters
// @Tags [Cloud Metadata] MC-Insight
// @Accept  json
// @Produce  json
// @Param csp query string true "CSP name (aws, gcp, azure, etc.)"
// @Param region query string false "Region filter"
// @Param zone query string false "Zone filter"
// @Param instance_type query string false "Instance type filter"
// @Param vcpu_count query int false "vCPU count filter"
// @Param mem_size_mib query int false "Memory size in MiB filter"
// @Param os_type query string false "OS type filter (linux, windows, etc.)"
// @Param pricing_type query string false "Pricing type filter (on-demand, spot, reserved, etc.)"
// @Param skip query int false "Number of records to skip for pagination"
// @Param limit query int false "Maximum number of records to return"
// @Success 200 {object} map[string]interface{} "VM Price list data"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /mcinsight/price-info [get]
func ProxyMcInsightVMPrice(c echo.Context) error {
	cblog.Info("call ProxyMcInsightVMPrice()")

	token := getMCInsightAPIToken()
	if token == "" {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "MC-Insight API token is not configured. Please set MC_INSIGHT_API_TOKEN in setup.env")
	}

	// Build query parameters
	params := url.Values{}
	queryParams := []string{"csp", "region", "zone", "name", "product_id", "instance_type", "vcpu_count", "mem_size_mib", "os_type", "pricing_type", "skip", "limit"}

	for _, param := range queryParams {
		if value := c.QueryParam(param); value != "" {
			params.Set(param, value)
		}
	}

	apiURL := fmt.Sprintf("%s/api/v1/price-info/?%s", MC_INSIGHT_API_BASE, params.Encode())

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to create request: %v", err))
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to fetch from MC-Insight: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to read response: %v", err))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to parse response: %v", err))
	}

	return c.JSON(http.StatusOK, result)
}
