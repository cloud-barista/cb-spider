// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2021.11.

package validatetest

import (
	cdcom "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	"testing"
	"log"
)

func TestValidatePW(t *testing.T) {

	pwList:= []string{
	      "cloudbarista",		// invalid
	      "cloudbarista123",	// invalid
	      "cloudbarista123^",	// invalid
	}

	for _, pw := range pwList {

		err := cdcom.ValidateWindowsPassword(pw)
		if err != nil {
			log.Println(pw + ": is invalid!")
			log.Println(err)
		} else {
			log.Println(pw + ": is valid!")
		}
	}
}

