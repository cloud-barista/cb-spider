package profile

import (
	"net/http"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	cblogger "github.com/cloud-barista/cb-log"
)

var cblog = cblogger.GetLogger("CLOUD-BARISTA")

var azureAPICallCount int
var azureMutex sync.Mutex

func incrementCallCount() {
	azureMutex.Lock()
	defer azureMutex.Unlock()
	azureAPICallCount++
}

func ResetCallCount() {
	azureMutex.Lock()
	defer azureMutex.Unlock()
	azureAPICallCount = 0
}

func GetCallCount() int {
	azureMutex.Lock()
	defer azureMutex.Unlock()
	return azureAPICallCount
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
