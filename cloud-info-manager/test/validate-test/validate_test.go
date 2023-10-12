// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2021.11.

package validatetest

import (
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"

	"log"
	"testing"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

func TestValid(t *testing.T) {
	inKeyValueList := []idrv.KeyValue{
		{"location", "kr"},
		{"ResourceGroup", "barista"},
	}
	wantedKeyList := []string{
		"location",
		"ResourceGroup",
	}
	err := cim.ValidateKeyValueList(inKeyValueList, wantedKeyList)
	if err != nil {
		log.Fatal("something failed!")
	}
}

func TestInvalid(t *testing.T) {
	inKeyValueList := []idrv.KeyValue{
		{"Location", "kr"},
		{"ResourceGroup", "barista"},
	}
	wantedKeyList := []string{
		"location",
		"ResourceGroup",
	}
	err := cim.ValidateKeyValueList(inKeyValueList, wantedKeyList)
	if err != nil {
		log.Fatal(err)
	}
}
