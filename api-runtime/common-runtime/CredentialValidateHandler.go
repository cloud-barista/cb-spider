// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2026.10.

package commonruntime

import (
	"fmt"
	"time"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const credentialValidateTimeout = 60 * time.Second

// CredentialValidateResult represents the result of a credential connectivity check.
type CredentialValidateResult struct {
	Valid        bool   `json:"Valid" validate:"required" example:"true"`
	ProviderName string `json:"ProviderName" validate:"required" example:"AWS"`
	Message      string `json:"Message" validate:"required" example:"Successfully authenticated to AWS."`
	ElapsedTime  string `json:"ElapsedTime" validate:"required" example:"1.234s"`
}

// ValidateCredential checks whether the given (not yet registered) credential can connect to the cloud.
// It calls the driver's ValidateCredential(), a lightweight authentication API without region information.
// The credential is not stored.
func ValidateCredential(providerName string, kvList []cres.KeyValue) (*CredentialValidateResult, error) {
	cblog.Info("call ValidateCredential()")

	providerName, err := EmptyCheckAndTrim("providerName", providerName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if len(kvList) == 0 {
		err := fmt.Errorf("KeyValueInfoList is empty")
		cblog.Error(err)
		return nil, err
	}

	result := &CredentialValidateResult{ProviderName: providerName}
	start := time.Now()

	done := make(chan error, 1)
	go func() {
		done <- ccm.ValidateCredentialByProvider(providerName, kvList)
	}()

	select {
	case err := <-done:
		if err != nil {
			result.Message = err.Error()
		} else {
			result.Valid = true
			result.Message = fmt.Sprintf("Successfully authenticated to %s.", providerName)
		}
	case <-time.After(credentialValidateTimeout):
		result.Message = fmt.Sprintf("Timed out after %v while connecting to %s.", credentialValidateTimeout, providerName)
	}
	result.ElapsedTime = time.Since(start).Round(time.Millisecond).String()

	if !result.Valid {
		cblog.Infof("ValidateCredential(): %s credential validation failed: %s", providerName, result.Message)
	}
	return result, nil
}
