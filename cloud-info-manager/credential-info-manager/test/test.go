// Test for Cloud Credential Info. Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.09.

package main

import (
	"fmt"

	"github.com/cloud-barista/cb-store/config"
	icbs "github.com/cloud-barista/cb-store/interfaces"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
)
func main() {

// ex)
// /cloud-info-spaces/credentials/<aws_credential01>/{aws}/{ClientId} [value1]
// /cloud-info-spaces/credentials/<aws_credential01>/{aws}/{ClientSecret} [value2]
// /cloud-info-spaces/credentials/<aws_credential01>/{aws}/{TenantId} [value3]
// /cloud-info-spaces/credentials/<aws_credential01>/{aws}/{SubscriptionId} [value4]


fmt.Println("\n============== RegisterCredential()")
	cName := "aws_credential01"
	pName := "AWS"
	keyValueList := []icbs.KeyValue{ {"ClientId", "value1"},
					 {"ClientSecret", "value2"},
					 {"TenantId", "value3"},
					 {"SubscriptionId", "value4"},
				       }

	crdInfo, err := cim.RegisterCredential(cName, pName, keyValueList)
	if err != nil {
		config.Cblogger.Error(err)
	}

	fmt.Printf(" === %#v\n", crdInfo)

fmt.Println("\n============== RegisterCredential()")
        cName = "aws_credential02"
        pName = "AWS"
	keyValueList = []icbs.KeyValue{ {"ClientId", "value101"}, 
					 {"ClientSecret", "value102"},
					 {"TenantId", "value103"},
					 {"SubscriptionId", "value104"},
				       }
	
        crdInfo, err = cim.RegisterCredential(cName, pName, keyValueList)
        if err != nil {
                config.Cblogger.Error(err)
        }

	fmt.Printf(" === %#v\n", crdInfo)

fmt.Println("\n============== RegisterCredential()")
        cName = "openstack_credential03"
        pName = "OPENSTACK"
        keyValueList = []icbs.KeyValue{ {"IdentityEndpoint", "value101"},
                                         {"Username", "value202"},
                                         {"Password", "value203"},
                                         {"DomainName", "value204"},
                                         {"ProjectID", "value205"},
                                       }

        crdInfo, err = cim.RegisterCredential(cName, pName, keyValueList)
        if err != nil {
                config.Cblogger.Error(err)
        }

        fmt.Printf(" === %#v\n", crdInfo)
	
fmt.Println("\n============== ListCredential()")
	credentialInfoList, err2 := cim.ListCredential()
	if err2 != nil {
		config.Cblogger.Error(err2)
	}

	for _, keyValue := range credentialInfoList {
                fmt.Printf(" === %#v\n", keyValue)
		cim.GetCredential(keyValue.CredentialName)
        }

fmt.Println("\n============== UnRegisterCredential()")
	cName = "aws_credential01"
        result, err3 := cim.UnRegisterCredential(cName)
        if err3 != nil {
                config.Cblogger.Error(err3)
        }

	fmt.Printf(" === cim.UnRegisterCredential %s : %#v\n", cName, result)

fmt.Println("\n============== ListCredential()")
        credentialInfoList, err2 = cim.ListCredential()
        if err2 != nil {
                config.Cblogger.Error(err2)
        }

        for _, keyValue := range credentialInfoList {
                fmt.Printf(" === %#v\n", keyValue)
        }

fmt.Println("\n============== UnRegisterCredential()")
	cName = "aws_credential02"
        result, err3 = cim.UnRegisterCredential(cName)
        if err3 != nil {
                config.Cblogger.Error(err3)
        }

        fmt.Printf(" === cim.UnRegisterCredential %s : %#v\n", cName, result)

fmt.Println("\n============== ListCredential()")
        credentialInfoList, err2 = cim.ListCredential()
        if err2 != nil {
                config.Cblogger.Error(err2)
        }

        for _, keyValue := range credentialInfoList {
                fmt.Printf(" === %#v\n", keyValue)
        }

fmt.Println("\n============== UnRegisterCredential()")
	cName = "openstack_credential03"
        result, err3 = cim.UnRegisterCredential(cName)
        if err3 != nil {
                config.Cblogger.Error(err3)
        }

        fmt.Printf(" === cim.UnRegisterCredential %s : %#v\n", cName, result)

fmt.Println("\n============== ListCredential()")
        credentialInfoList, err2 = cim.ListCredential()
        if err2 != nil {
                config.Cblogger.Error(err2)
        }

        for _, keyValue := range credentialInfoList {
                fmt.Printf(" === %#v\n", keyValue)
        }

}
