// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2022.09.

package resources

// AnyCallInfo represents the structure for performing AnyCall API requests and responses.
// @description This structure holds both the input and output parameters for the AnyCall API.
type AnyCallInfo struct {
	FID           string     `json:"FID" validate:"required" example:"countAll"` // Function ID, ex: countAll
	IKeyValueList []KeyValue `json:"IKeyValueList" validate:"required"`          // Input Arguments List, ex:[{"Key": "rsType", "Value": "vpc"}]
	OKeyValueList []KeyValue `json:"OKeyValueList" validate:"required"`          // Output Results List, ex:[{"Key": "Count", "Value": "10"}]"
}

type AnyCallHandler interface {
	AnyCall(callInfo AnyCallInfo) (AnyCallInfo, error)
}
