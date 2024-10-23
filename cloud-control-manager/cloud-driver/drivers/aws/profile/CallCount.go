package profile

import (
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
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

// creates a new AWS session that counts the number of API calls made
func NewCountingSession(config *aws.Config) *session.Session {
	sess, err := session.NewSession(config)
	if err != nil {
		cblog.Error("Could not create AWS session", err)
		return nil
	}

	// increment the call count for each API call with callback
	sess.Handlers.Send.PushFront(func(r *request.Request) {
		incrementCallCount()
	})

	return sess
}
