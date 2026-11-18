// Proxy for AdminWeb(Multi-Connection Add dialogs) of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista

package adminweb

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

//====================================== Multi-Connection Add dialogs: streamed per-connection data
//
// The Add dialogs of the multi-connection pages (ex: NLB, VM) need several resource lists of every
// connection (VPCs, VMs, SecurityGroups, ...). Fetching them from the browser takes a few requests per
// connection: with 10 connections that is 30-40 requests, and the browser opens at most 6 connections
// to one server, so the requests queued behind slow ones (ex: about 10s in total for the NLB dialog).
//
// streamPerConnection fetches all connections at the same time on the server side and streams one JSON
// line per connection (NDJSON) as soon as it is ready, so the dialog fills each row as it arrives, the
// browser sends a single request, and one slow connection does not hold back the others.

// connectionIDs returns the connection names of the "ids" query parameter (comma separated).
func connectionIDs(c echo.Context) []string {
	var connNames []string
	for _, id := range strings.Split(c.QueryParam("ids"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			connNames = append(connNames, id)
		}
	}
	return connNames
}

// streamPerConnection runs fetch for every connection at the same time and writes each result as one
// JSON line, in the order the connections finish. It stops when the browser goes away (ex: the dialog
// was closed); the fetches still running finish in the background and are discarded.
func streamPerConnection(c echo.Context, connNames []string, fetch func(connName string) any) error {
	resp := c.Response()
	resp.Header().Set(echo.HeaderContentType, "application/x-ndjson")
	resp.Header().Set("Cache-Control", "no-store")
	resp.WriteHeader(http.StatusOK)
	resp.Flush()

	results := make(chan any, len(connNames)) // buffered: fetches never block after the browser left
	for _, connName := range connNames {
		go func(connName string) { results <- fetch(connName) }(connName)
	}

	encoder := json.NewEncoder(resp) // Encode writes one JSON value and a newline
	for range connNames {
		select {
		case result := <-results:
			if err := encoder.Encode(result); err != nil {
				return nil // the browser went away
			}
			resp.Flush()
		case <-c.Request().Context().Done():
			return nil
		}
	}
	return nil
}
