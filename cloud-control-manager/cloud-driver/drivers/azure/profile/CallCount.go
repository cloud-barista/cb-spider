package profile

import (
	"net/http"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	cblogger "github.com/cloud-barista/cb-log"
)

var cblog = cblogger.GetLogger("CLOUD-BARISTA")

var apiCallCount int
var mutex sync.Mutex

func incrementCallCount() {
	mutex.Lock()
	defer mutex.Unlock()
	apiCallCount++
}

func ResetCallCount() {
	mutex.Lock()
	defer mutex.Unlock()
	apiCallCount = 0
}

func GetCallCount() int {
	mutex.Lock()
	defer mutex.Unlock()
	return apiCallCount
}

// NewCountingPolicy creates a new custom policy to count Azure SDK API calls.
func NewCountingPolicy() policy.Policy {
	return &countingPolicy{}
}

type countingPolicy struct{}

func (p *countingPolicy) Do(req *policy.Request) (*http.Response, error) {
	incrementCallCount()
	return req.Next()
}
