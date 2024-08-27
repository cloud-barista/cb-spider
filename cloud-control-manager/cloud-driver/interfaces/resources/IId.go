// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2020.03.

package resources

// Integrated-ID consisting of User's ID and CloudOS's ID
type IID struct {
	NameId   string `json:"NameId" validate:"required" example:"user-defined-name"`
	SystemId string `json:"SystemId" validate:"required" example:"csp-defined-id"`
}
