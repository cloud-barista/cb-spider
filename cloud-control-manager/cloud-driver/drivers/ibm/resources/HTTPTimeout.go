package resources

import (
	"net/http"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
)

// IbmHTTPRequestTimeout bounds each single HTTP request to the IBM Cloud APIs.
//
// The SDK default client (core.DefaultHTTPClient) has no overall request timeout, so a request
// whose response never arrives blocks until the connection context expires (about 100 min),
// holding Spider's resource lock all that time.
//
// This is a per-request timeout, not a per-driver-call timeout: long driver operations
// (ex: CreateNLB, DeleteNLB, which take several minutes) are polling loops made of short
// requests, so they are not affected. IBM VPC/Kubernetes APIs are asynchronous
// (they return once the request is accepted), and the slowest single request observed
// is far below this value.
const IbmHTTPRequestTimeout = 5 * time.Minute

// NewIbmHTTPClient returns the SDK default HTTP client with IbmHTTPRequestTimeout applied.
func NewIbmHTTPClient() *http.Client {
	client := core.DefaultHTTPClient()
	client.Timeout = IbmHTTPRequestTimeout
	return client
}

// SetIbmHTTPTimeout applies IbmHTTPRequestTimeout to an IBM SDK service.
func SetIbmHTTPTimeout(service *core.BaseService) {
	if service != nil {
		service.SetHTTPClient(NewIbmHTTPClient())
	}
}
