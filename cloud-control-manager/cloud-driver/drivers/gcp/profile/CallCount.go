package profile

import (
	"net/http"
	"sync"

	"golang.org/x/oauth2"
)

var GCPAPICallCount int
var GCPMutex sync.Mutex

// incrementCallCount increments the count of API calls.
func incrementCallCount() {
	GCPMutex.Lock()
	defer GCPMutex.Unlock()
	GCPAPICallCount++
}

// ResetCallCount resets the API call count.
func ResetCallCount() {
	GCPMutex.Lock()
	defer GCPMutex.Unlock()
	GCPAPICallCount = 0
}

// GetCallCount returns the current count of API calls.
func GetCallCount() int {
	GCPMutex.Lock()
	defer GCPMutex.Unlock()
	return GCPAPICallCount
}

// NewCountingClient returns an *http.Client that counts each request.
func NewCountingClient() *http.Client {
	return &http.Client{
		Transport: &countingRoundTripper{
			base: http.DefaultTransport,
		},
	}
}

type countingRoundTripper struct {
	base http.RoundTripper
}

func (c *countingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	incrementCallCount()
	return c.base.RoundTrip(req)
}

// oauth2Transport is a custom RoundTripper that injects OAuth2 tokens into requests.
type oauth2Transport struct {
	base  http.RoundTripper
	token oauth2.TokenSource
}

// NewOauth2Transport creates a new oauth2Transport with the given base transport and token source.
func NewOauth2Transport(base http.RoundTripper, token oauth2.TokenSource) *oauth2Transport {
	return &oauth2Transport{
		base:  base,
		token: token,
	}
}

func (t *oauth2Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Get the current OAuth2 token.
	token, err := t.token.Token()
	if err != nil {
		return nil, err
	}
	// Set the token in the request headers.
	token.SetAuthHeader(req)
	// Use the base RoundTripper to make the request.
	return t.base.RoundTrip(req)
}
