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

func TestAddListGetDelete(t *testing.T) {

	privateKey, _, err := cdcom.GenKeyPair()

	strList:= []string{
	      "IdentityEndpoint-01",
	      "AuthToken-01",
	      "TenantId-01",
	}
	strHash, err := cdcom.GenHash(strList)

	keyPairNameId := "keypair-0-c6ncl9aba5o081np93og"


	// (1) insert-1
        log.Println("====================================== (1) insert-1 ======================================")
	err = cdcom.AddKey("CLOUDIT", strHash, keyPairNameId, string(privateKey))
	if err != nil {
		log.Fatal("something failed!")
	}

	// (1) insert-2
        log.Println("====================================== (1) insert-2 ======================================")
	privateKey, _, err = cdcom.GenKeyPair()
        keyPairNameId = "keypair-1-c6ncl9aba5o081np93og"
	err = cdcom.AddKey("CLOUDIT", strHash, keyPairNameId, string(privateKey))
	if err != nil {
		log.Fatal("something failed!")
	}


	// (2) list
        log.Println("====================================== (2) list ======================================")
	keyValueList, err := cdcom.ListKey("CLOUDIT", strHash)
        if err != nil {
                log.Fatal("something failed!")
        }
	log.Println(keyValueList)
	if len(keyValueList) != 2 {
                log.Fatal("The number of Key list is not 2!")
	}


	// (3) get
        log.Println("====================================== (3) get ======================================")
	keyValue, err := cdcom.GetKey("CLOUDIT", strHash, keyPairNameId)
        if err != nil {
                log.Fatal("something failed!")
        }
        log.Println(keyValue)


	// (4) delete-1
        log.Println("====================================== (4) delete-1 ======================================")
        keyPairNameId = "keypair-0-c6ncl9aba5o081np93og"
        err = cdcom.DelKey("CLOUDIT", strHash, keyPairNameId)
        if err != nil {
                log.Fatal("something failed!")
        }

	// (4) delete-2
        log.Println("====================================== (4) delete-2 ======================================")
        keyPairNameId = "keypair-1-c6ncl9aba5o081np93og"
        err = cdcom.DelKey("CLOUDIT", strHash, keyPairNameId)
        if err != nil {
                log.Fatal("something failed!")
        }

	// (5) check
        log.Println("====================================== (5) check ======================================")
        keyValueList, err = cdcom.ListKey("CLOUDIT", strHash)
        if err != nil {
                log.Fatal("something failed!")
        }
        log.Println(keyValueList)
	if len(keyValueList) != 0 {
                log.Fatal("The number of Key list is not 0!")
	}
}

