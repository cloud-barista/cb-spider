package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

// DefaultUserAgent is the default User-Agent string set in the request header.
const (
	DefaultUserAgent = "cloudit/1.0.0"
)

// ClouditEngine is type of cloud service in cloudit
type ClouditEngine string

const (
	IAM ClouditEngine = "iam"
	ACE ClouditEngine = "ace"
	DNA ClouditEngine = "dna"
)

type RestClient struct {
	// IdentityBase is the base URL used for a particular provider's identity
	// service - it will be used when issuing authenticatation requests. It
	// should point to the root resource of the identity service, not a specific
	// identity version.
	IdentityBase string

	// IdentityEndpoint is the identity endpoint. This may be a specific version
	// of the identity service. If this is the case, this endpoint is used rather
	// than querying versions first.
	IdentityEndpoint string

	// ClouditVersion
	ClouditVersion string

	// TenantId for Cloudit User
	TenantID string

	// TokenID is the ID of the most recently issued valid token.
	TokenID string

	// HTTPClient allows users to interject arbitrary http, https, or other transit behaviors.
	HTTPClient http.Client
}

// AuthenticatedHeaders returns a map of HTTP headers that are common for all
// authenticated service requests.
func (client *RestClient) AuthenticatedHeaders() map[string]string {
	if client.TokenID == "" {
		return map[string]string{}
	}
	return map[string]string{"X-Auth-Token": client.TokenID}
}

// RequestOpts customizes the behavior of the provider.Request() method.
type RequestOpts struct {
	// JSONBody, if provided, will be encoded as JSON and used as the body of the HTTP request. The
	// content type of the request will default to "application/json" unless overridden by MoreHeaders.
	// It's an error to specify both a JSONBody and a RawBody.
	JSONBody interface{}
	// RawBody contains an io.ReadSeeker that will be consumed by the request directly. No content-type
	// will be set unless one is provided explicitly by MoreHeaders.
	RawBody io.ReadSeeker

	// JSONResponse, if provided, will be populated with the contents of the response body parsed as
	// JSON.
	JSONResponse interface{}
	// OkCodes contains a list of numeric HTTP status codes that should be interpreted as success. If
	// the response has a different code, an error will be returned.
	OkCodes []int

	// MoreHeaders specifies additional HTTP headers to be provide on the request. If a header is
	// provided with a blank value (""), that header will be *omitted* instead: use this to suppress
	// the default Accept header or an inferred Content-Type, for example.
	MoreHeaders map[string]string
}

type Result struct {
	// Body is the payload of the HTTP response from the server. In most cases,
	// this will be the deserialized JSON structure.
	Body interface{}

	// Header contains the HTTP header structure from the original response.
	Header http.Header

	// Err is an error that occurred during the operation. It's deferred until
	// extraction to make it easier to chain the Extract call.
	Err error
}

func (opts *RequestOpts) setBody(body interface{}) {
	if v, ok := (body).(io.ReadSeeker); ok {
		opts.RawBody = v
	} else if body != nil {
		opts.JSONBody = body
	}
}

// UnexpectedResponseCodeError is returned by the Request method when a response code other than
// those listed in OkCodes is encountered.
type UnexpectedResponseCodeError struct {
	URL      string
	Method   string
	Expected []int
	Actual   int
	Body     []byte
}

func (err *UnexpectedResponseCodeError) Error() string {
	return fmt.Sprintf(
		"Expected HTTP response code %v when accessing [%s %s], but got %d instead\n%s",
		err.Expected, err.Method, err.URL, err.Actual, err.Body,
	)
}

var applicationJSON = "application/json"

// Request performs an HTTP request using the RestClient's current HTTPClient. An authentication
// header will automatically be provided.
func (client *RestClient) Request(method, url string, options RequestOpts) (*http.Response, error) {
	var body io.ReadSeeker
	var contentType *string

	// Derive the content body by either encoding an arbitrary object as JSON, or by taking a provided
	// io.ReadSeeker as-is. Default the content-type to application/json.
	if options.JSONBody != nil {
		if options.RawBody != nil {
			cblogger.Error("Please provide only one of JSONBody or RawBody to cloudit.Request().")
		}

		rendered, err := json.Marshal(options.JSONBody)
		if err != nil {
			return nil, err
		}

		body = bytes.NewReader(rendered)
		contentType = &applicationJSON
	}

	if options.RawBody != nil {
		body = options.RawBody
	}

	// Construct the http.Request.
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Populate the request headers. Apply options.MoreHeaders last, to give the caller the chance to
	// modify or omit any header.
	if contentType != nil {
		req.Header.Set("Content-Type", *contentType)
	}
	req.Header.Set("Accept", applicationJSON)

	for k, v := range client.AuthenticatedHeaders() {
		req.Header.Add(k, v)
	}

	if options.MoreHeaders != nil {
		for k, v := range options.MoreHeaders {
			if v != "" {
				req.Header.Set(k, v)
			} else {
				req.Header.Del(k)
			}
		}
	}

	// Set connection parameter to close the connection immediately when we've got the response
	req.Close = true

	// Issue the request.
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Allow default OkCodes if none explicitly set
	if options.OkCodes == nil {
		options.OkCodes = defaultOkCodes(method)
	}

	// Validate the HTTP response status.
	var ok bool
	for _, code := range options.OkCodes {
		if resp.StatusCode == code {
			ok = true
			break
		}
	}
	if !ok {
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		return resp, &UnexpectedResponseCodeError{
			URL:      url,
			Method:   method,
			Expected: options.OkCodes,
			Actual:   resp.StatusCode,
			Body:     body,
		}
	}

	// Parse the response body as JSON, if requested to do so.
	if options.JSONResponse != nil {
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(options.JSONResponse); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

func defaultOkCodes(method string) []int {
	switch {
	case method == "GET":
		return []int{200}
	case method == "POST":
		return []int{201, 202}
	case method == "PUT":
		return []int{201, 202}
	case method == "PATCH":
		return []int{200, 204}
	case method == "DELETE":
		return []int{202, 204}
	}

	return []int{}
}

func (client *RestClient) Get(url string, JSONResponse *interface{}, opts *RequestOpts) (*http.Response, error) {
	if opts == nil {
		opts = &RequestOpts{}
	}
	if JSONResponse != nil {
		opts.JSONResponse = JSONResponse
	}
	return client.Request("GET", url, *opts)
}

func (client *RestClient) Post(url string, body interface{}, JSONResponse *interface{}, opts *RequestOpts) (*http.Response, error) {
	if opts == nil {
		opts = &RequestOpts{}
	}

	opts.setBody(body)

	if JSONResponse != nil {
		opts.JSONResponse = JSONResponse
	}

	return client.Request("POST", url, *opts)
}

func (client *RestClient) Put(url string, body interface{}, JSONResponse *interface{}, opts *RequestOpts) (*http.Response, error) {
	if opts == nil {
		opts = &RequestOpts{}
	}

	opts.setBody(body)

	if JSONResponse != nil {
		opts.JSONResponse = JSONResponse
	}

	return client.Request("PUT", url, *opts)
}

func (client *RestClient) Patch(url string, JSONBody interface{}, JSONResponse *interface{}, opts *RequestOpts) (*http.Response, error) {
	if opts == nil {
		opts = &RequestOpts{}
	}

	if v, ok := (JSONBody).(io.ReadSeeker); ok {
		opts.RawBody = v
	} else if JSONBody != nil {
		opts.JSONBody = JSONBody
	}

	if JSONResponse != nil {
		opts.JSONResponse = JSONResponse
	}

	return client.Request("PATCH", url, *opts)
}

func (client *RestClient) Delete(url string, opts *RequestOpts) (*http.Response, error) {
	if opts == nil {
		opts = &RequestOpts{}
	}

	return client.Request("DELETE", url, *opts)
}

func (r Result) ExtractInto(to interface{}) error {
	if reader, ok := r.Body.(io.Reader); ok {
		if readCloser, ok := reader.(io.Closer); ok {
			defer readCloser.Close()
		}
		return json.NewDecoder(reader).Decode(to)
	}

	b, err := json.Marshal(r.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, to)

	return err
}

func (client *RestClient) CreateRequestBaseURL(engine ClouditEngine, parts ...string) string {
	engineName := fmt.Sprint(engine)
	baseURL := strings.Join([]string{client.IdentityBase, "cloudit", client.ClouditVersion, engineName, "v1.0", client.TenantID}, "/")
	customURI := strings.Join(parts, "/")
	return baseURL + "/" + customURI
}
