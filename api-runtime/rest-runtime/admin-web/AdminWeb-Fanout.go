// Proxy for AdminWeb(Multi-Connection fan-out) of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista

package adminweb

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

//====================================== Multi-Connection fan-out of Spider API requests
//
// The multi-connection pages send one Spider API request per connection (ex: create an NLB or a VM in
// every connection). The browser opens at most 6 connections to one server, so with more requests the
// rest queue: when creating NLBs in 10 connections, 4 requests started 4-30s late, and a request that
// takes long (ex: an IBM NLB takes about 7 minutes to provision) holds its slot the whole time.
//
// Fanout receives all the requests in one call, sends them to the Spider API at the same time, and
// streams each response as one JSON line (NDJSON) as soon as it arrives. The page turns each line back
// into a Response of the request it belongs to (see multiFanout in static/js/multi-task.js).

// fanoutRequest is one Spider API request of a fan-out.
type fanoutRequest struct {
	ID      string            `json:"id"`                // given by the page to match the response
	Method  string            `json:"method"`            // GET, POST, PUT or DELETE
	Path    string            `json:"path"`              // Spider API path with query, ex: /spider/nlb
	Headers map[string]string `json:"headers,omitempty"` // extra request headers, ex: X-Connection-Name
	Body    json.RawMessage   `json:"body,omitempty"`
}

// fanoutBlockedHeaders are headers a page cannot set: the server sets authentication itself.
var fanoutBlockedHeaders = map[string]bool{"authorization": true, "cookie": true, "host": true, "content-length": true}

// fanoutResponse is one streamed line: the response of one request.
type fanoutResponse struct {
	ID     string `json:"id"`
	Status int    `json:"status"`          // HTTP status of the Spider API, 0 when the request could not be sent
	Body   string `json:"body"`            // response body as text
	Error  string `json:"error,omitempty"` // why the request could not be sent
}

var fanoutMethods = map[string]bool{http.MethodGet: true, http.MethodPost: true, http.MethodPut: true, http.MethodDelete: true}

// validFanoutPath accepts Spider API paths only, not AdminWeb pages (no fan-out of fan-outs).
func validFanoutPath(path string) bool {
	return strings.HasPrefix(path, "/spider/") && !strings.HasPrefix(path, "/spider/adminweb") && !strings.Contains(path, "..")
}

// sendFanoutRequest sends one request to the Spider API with the server's API credentials,
// as the AdminWeb pages do with their own Spider API calls.
func sendFanoutRequest(r fanoutRequest) fanoutResponse {
	res := fanoutResponse{ID: r.ID}
	method := strings.ToUpper(r.Method)
	if !fanoutMethods[method] || !validFanoutPath(r.Path) {
		res.Error = "not allowed: " + r.Method + " " + r.Path
		return res
	}

	var body io.Reader
	if len(r.Body) > 0 {
		body = bytes.NewReader(r.Body)
	}
	req, err := http.NewRequest(method, spiderBaseURL()+r.Path, body)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range r.Headers {
		if !fanoutBlockedHeaders[strings.ToLower(key)] {
			req.Header.Set(key, value)
		}
	}
	setBasicAuthIfConfigured(req)

	// no client timeout: a request takes as long as the CSP needs (ex: IBM NLB provisioning)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Status = resp.StatusCode
	res.Body = string(respBody)
	return res
}

// Fanout sends the given Spider API requests at the same time and streams each response as one
// JSON line, in the order they finish. The requests keep running when the browser goes away
// (ex: the page was reloaded), as requests sent by the browser itself do once they reached the server.
// POST /spider/adminweb/fanout  body: {"requests": [{"id", "method", "path", "body"}, ...]}
func Fanout(c echo.Context) error {
	var reqBody struct {
		Requests []fanoutRequest `json:"requests"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&reqBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid fan-out request: "+err.Error())
	}

	resp := c.Response()
	resp.Header().Set(echo.HeaderContentType, "application/x-ndjson")
	resp.Header().Set("Cache-Control", "no-store")
	resp.WriteHeader(http.StatusOK)
	resp.Flush()

	results := make(chan fanoutResponse, len(reqBody.Requests)) // buffered: requests never block
	for _, r := range reqBody.Requests {
		go func(r fanoutRequest) { results <- sendFanoutRequest(r) }(r)
	}

	encoder := json.NewEncoder(resp)
	browserGone := c.Request().Context().Done()
	for range reqBody.Requests {
		select {
		case result := <-results:
			if err := encoder.Encode(result); err != nil {
				return nil
			}
			resp.Flush()
		case <-browserGone:
			return nil
		}
	}
	return nil
}
