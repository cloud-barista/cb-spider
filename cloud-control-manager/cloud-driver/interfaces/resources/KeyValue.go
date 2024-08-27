// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2019.10.

package resources

type KeyValue struct {
	Key   string `json:"Key" validate:"required" example:"key1"`
	Value string `json:"Value,omitempty" validate:"omitempty"  example:"value1"`
}
