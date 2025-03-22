// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista

package restruntime

import (
	"bytes"
	"io"
	"net/http"
	"os"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	"github.com/labstack/echo/v4"
)

// FetchSystemInfo godoc
// @ID fetch-system-info
// @Summary Fetch System Information
// @Description Retrieve system information such as hostname, platform, CPU, memory, and disk.
// @Description Use query parameter 'mode=text' to get the output in text format instead of JSON.
// @Tags [Utility]
// @Accept json
// @Produce json,text/plain
// @Param mode query string false "Output format: set to 'text' for text output (default is JSON)" Enums(text)
// @Success 200 {object} cmrt.SystemInfo "System Information in JSON or text format based on 'mode' parameter"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /sysstats/system [get]
func FetchSystemInfo(c echo.Context) error {
	cblog.Info("call FetchSystemInfo()")

	// Get system information
	result, err := cmrt.FetchSystemInfo()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Check output mode (text or json)
	mode := c.QueryParam("mode")

	// If text mode, output as text format
	if mode == "text" {
		// Create buffer for console output
		var buffer bytes.Buffer

		// Redirect output to buffer
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Output as text format
		cmrt.DisplaySystemInfo(&result)

		// Close pipe
		w.Close()
		os.Stdout = oldStdout

		// Read to buffer
		io.Copy(&buffer, r)

		// Respond with text format
		c.Response().Header().Set(echo.HeaderContentType, "text/plain; charset=UTF-8")
		return c.String(http.StatusOK, buffer.String())
	}

	// Default response in JSON format
	return c.JSON(http.StatusOK, result)
}

// FetchResourceUsage godoc
// @ID fetch-resource-usage
// @Summary Fetch Resource Usage Information
// @Description Retrieve resource usage information such as CPU, memory, disk I/O, and network I/O.
// @Description Use query parameter 'mode=text' to get the output in text format instead of JSON.
// @Tags [Utility]
// @Accept json
// @Produce json,text/plain
// @Param mode query string false "Output format: set to 'text' for text output (default is JSON)" Enums(text)
// @Success 200 {object} cmrt.ResourceUsage "Resource Usage Information in JSON or text format based on 'mode' parameter"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /sysstats/usage [get]
func FetchResourceUsage(c echo.Context) error {
	cblog.Info("call FetchResourceUsage()")

	// Get resource usage information
	resourceUsage, err := cmrt.FetchResourceUsage()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Check output mode (text or json)
	mode := c.QueryParam("mode")

	// If text mode, output as text format
	if mode == "text" {
		// Also get system info to get TotalMemory for display
		sysInfo, err := cmrt.FetchSystemInfo()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// Create buffer for console output
		var buffer bytes.Buffer

		// Redirect output to buffer
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Output as text format
		cmrt.DisplayResourceUsage(sysInfo.TotalMemory, &resourceUsage)

		// Close pipe
		w.Close()
		os.Stdout = oldStdout

		// Read to buffer
		io.Copy(&buffer, r)

		// Respond with text format
		c.Response().Header().Set(echo.HeaderContentType, "text/plain; charset=UTF-8")
		return c.String(http.StatusOK, buffer.String())
	}

	// Default response in JSON format
	return c.JSON(http.StatusOK, resourceUsage)
}
