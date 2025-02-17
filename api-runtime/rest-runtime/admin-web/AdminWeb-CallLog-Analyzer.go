// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// By CB-Spider Team

package adminweb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

type LogEntry struct {
	Timestamp    time.Time `json:"timestamp"`
	CloudOS      string    `json:"cloud_os"`
	RegionZone   string    `json:"region_zone"`
	ResourceType string    `json:"resource_type"`
	ResourceName string    `json:"resource_name"`
	CloudOSAPI   string    `json:"cloud_os_api"`
	ElapsedTime  float64   `json:"elapsed_time"`
	ErrorMSG     string    `json:"error_msg"`
}

func readLogs(filePath string) ([]LogEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	logPattern := regexp.MustCompile(`\[HISCALL\]\.\[(.*?)\]\s([\d-]+\s[\d:]+)\s\(\w+\)\s.*?-\s"CloudOS"\s:\s"(.*?)",\s"RegionZone"\s:\s"(.*?)",\s"ResourceType"\s:\s"(.*?)",\s"ResourceName"\s:\s"(.*?)",\s"CloudOSAPI"\s:\s"(.*?)",\s"ElapsedTime"\s:\s"(.*?)",\s"ErrorMSG"\s:\s"(.*?)"`)
	var logs []LogEntry

	matches := logPattern.FindAllStringSubmatch(string(data), -1)

	startIdx := 0
	if len(matches) > 1000 {
		startIdx = len(matches) - 1000
	}

	for _, match := range matches[startIdx:] {
		elapsedTime, _ := strconv.ParseFloat(match[8], 64)
		timestamp, _ := time.Parse("2006-01-02 15:04:05", match[2])
		logs = append(logs, LogEntry{
			Timestamp:    timestamp,
			CloudOS:      match[3],
			RegionZone:   match[4],
			ResourceType: match[5],
			ResourceName: match[6],
			CloudOSAPI:   match[7],
			ElapsedTime:  elapsedTime,
			ErrorMSG:     match[9],
		})
	}

	return logs, nil
}

// CallLogAnalyzer: Render the calllog-analyzer.html template
func CallLogAnalyzer(c echo.Context) error {
	tmplPath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/calllog-analyzer.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return tmpl.Execute(c.Response().Writer, nil)
}

// AnalyzeLogs handles the log analysis request
func AnalyzeLogs(c echo.Context) error {
	var userInput struct {
		Query string `json:"query"`
	}

	if err := c.Bind(&userInput); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}

	logFilePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/log/calllog/calllogs.log")
	logs, err := readLogs(logFilePath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	apiKey, err := getClaudeAPIKey()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get API key")
	}

	var logMessages []ClaudeContent
	logMessages = append(logMessages, ClaudeContent{
		Type: "text",
		Text: fmt.Sprintf("Analyzing %d most recent logs", len(logs)),
	})

	for _, log := range logs {
		logMessages = append(logMessages, ClaudeContent{
			Type: "text",
			Text: fmt.Sprintf("%s - CloudOS: %s, API: %s, ElapsedTime: %.4f",
				log.Timestamp, log.CloudOS, log.CloudOSAPI, log.ElapsedTime),
		})
	}

	requestBody := ClaudeRequestBody{
		Model:       "claude-3-5-sonnet-20241022",
		MaxTokens:   8192,
		Temperature: 0.5,
		System: `I am a specialized analyst and reporter for CB-Spider API Call logs.
			My primary role is to analyze API call logs to provide meaningful statistical information and insights.
			I excel at extracting valuable patterns and trends from API usage data to deliver comprehensive analytical reports.

			You are a cloud API log analyzer that creates comprehensive HTML-based analysis reports. Your analysis should include:
				
				1. Summary Information (Always shown at the top):
				- Start directly with the HTML content without explanation
				- Overview of included providers/CSPs
				- Types of resources found in logs
				- Time range of log data
				- Total number of log entries analyzed
				
				2. Detailed Statistical Analysis:
				- Total request counts and unique providers
				- Per-provider statistics (request counts, response times, success rates)
				- Most frequent operations and their performance
				- Error analysis and patterns
				- Time-based trends
				
				3. Visualization and Presentation Requirements:
				- For analyzable data, present information in the following order:
					* Analysis title/header
					* Data table with relevant statistics
					* Corresponding chart visualization
				- Create SVG charts using vanilla JavaScript (no external libraries)
				- Include only relevant charts based on available data
				- All charts must include:
					* Properly title labeled X and Y axes
					* Legend when applicable
					* Interactive tooltips

				4. Chart Generation and Validation:
				- Before creating any chart:
					* Verify data existence and validity
					* Check for minimum required data points (at least 2 points for line charts)
					* Ensure all required values are non-null
				- When generating SVG charts:
					* Set explicit width and height attributes (e.g., width="800" height="400")
					* Include viewBox attribute for proper scaling
					* Define chart area with proper margins (e.g., 60px margin for axes)
					* Add error handlers in JavaScript for data processing
				- After chart creation:
					* Verify SVG elements are properly nested
					* Confirm all data points are within viewBox
					* Test tooltip functionality
				- Implement fallback display:
					* Show placeholder message if chart cannot be rendered
					* Display data in table format as backup
				
				5. Color and Style Requirements:
				- Use pastel color palette for all charts and graphs:
					* Primary colors: #FFB3BA (pastel pink), #BAFFC9 (pastel green), #BAE1FF (pastel blue)
					* Secondary colors: #FFE4B5 (pastel orange), #E6E6FA (pastel purple), #FFFACD (pastel yellow)
				- Ensure sufficient contrast for readability
				- Apply subtle gradients or opacity variations for depth
				
				6. Chart JavaScript Template:
				
					function createChart(data, containerId) {
						// Validate input data
						if (!data || data.length < 2) {
							const container = document.getElementById(containerId);
							container.innerHTML = 'Insufficient data for chart visualization';
							return;
						}

						// Set up chart dimensions
						const width = 800;
						const height = 400;
						const margin = { top: 40, right: 40, bottom: 60, left: 60 };
						const innerWidth = width - margin.left - margin.right;
						const innerHeight = height - margin.top - margin.bottom;

						// Create SVG element with proper attributes
						const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
						svg.setAttribute("width", width);
						svg.setAttribute("height", height);
						svg.setAttribute("viewBox", "0 0 " + width + " " + height);
						
						try {
							// Chart creation logic here
							// Include proper error handling
						} catch (error) {
							console.error("Chart creation error:", error);
							const container = document.getElementById(containerId);
							container.innerHTML = 'Error creating chart: ' + error.message;
						}
					}

				
				7. Report Structure:
				- Statistical analysis in a readable format
				- All necessary styling (use clean, modern design)
				- JavaScript for chart creation with error handling
				- Data directly embedded in the page
				- Include insights section at the bottom when meaningful patterns or trends are identified

				8. Quality Assurance:
				- Each chart must be tested for:
					* Proper rendering
					* Data accuracy
					* Responsive behavior
					* Error handling
				- Provide fallback displays when charts cannot be rendered
				- Log any chart creation errors for debugging

			Every visualization must include proper validation and error handling to prevent blank or broken charts.
			When in doubt about data validity, favor displaying data in table format over potentially problematic charts.`,
		Messages: []ClaudeMessage{
			{
				Role: "user",
				Content: append(logMessages, ClaudeContent{
					Type: "text",
					Text: userInput.Query,
				}),
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal JSON")
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create API request")
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to call API")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to read response")
	}

	var anthropicResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse response")
	}

	var analysisResult string
	for _, content := range anthropicResp.Content {
		analysisResult += content.Text
	}

	return c.HTML(http.StatusOK, analysisResult)
}
