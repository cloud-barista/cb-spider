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


type AnyCallInfo struct {
	FID   string		// Function ID

	IKeyValueList []KeyValue // Input Arguments List
	OKeyValueList []KeyValue // Output Results List
}

type AnyCallHandler interface {
	AnyCall(callInfo AnyCallInfo) (AnyCallInfo, error)
}
