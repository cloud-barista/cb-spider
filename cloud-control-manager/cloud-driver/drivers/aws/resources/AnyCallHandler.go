// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is AWS Driver.
//
// by CB-Spider Team, 2022.09.

package resources

import (
	"fmt"
	"errors"

	_ "github.com/aws/aws-sdk-go/aws"
	_ "github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsAnyCallHandler struct {
        Region         idrv.RegionInfo
        CredentialInfo idrv.CredentialInfo
        Client         *ec2.EC2
}

/********************************************************
        // call example
        curl -sX POST http://localhost:1024/spider/anycall -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName" : "aws-ohio-config",
                "ReqInfo" : {
                        "FID" : "addTag",
                        "IKeyValueList" : [{"Key":"key1", "Value":"value1"}, {"Key":"key2", "Value":"value2"}]
                }
        }' | json_pp
********************************************************/
func (anyCallHandler *AwsAnyCallHandler) AnyCall(callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("AWS Driver: called AnyCall()!")

        switch callInfo.FID {
        case "addTag" :
                return addTag(anyCallHandler, callInfo)

        // add more ...

        default :
                return irs.AnyCallInfo{}, errors.New("AWS Driver: " + callInfo.FID + " Function is not implemented!")
        }
}

///////////////////////////////////////////////////////////////////
// implemented by developer user, like 'addTag(kv []KeyVale) bool'
///////////////////////////////////////////////////////////////////
func addTag(anyCallHandler *AwsAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
        cblogger.Info("AWS Driver: called AnyCall()/addTag()!")

	// you must delete this line
	fmt.Printf("\n\n\n * Region/Zone:%s/%s, *ClientId:%s, * ClientSecret:%s\n", 
		anyCallHandler.Region.Region, anyCallHandler.Region.Zone, 
		anyCallHandler.CredentialInfo.ClientId, anyCallHandler.CredentialInfo.ClientSecret)
	fmt.Printf("\n\n\n * Client:%#v\n\n\n", *anyCallHandler.Client)

        // Input Arg Validation
        if callInfo.IKeyValueList == nil {
                return irs.AnyCallInfo{}, errors.New("AWS Driver: " + callInfo.FID + "'s Argument is empty!")
        }

	// run
	for _, kv :=range callInfo.IKeyValueList {
		// Implement to add tags in AWS
		fmt.Printf("\n\n\n Key:%s, Value:%s\n", kv.Key, kv.Value)
		// and You can use 'anyCallHandler.Client'
	}

        // make results
        if callInfo.OKeyValueList == nil {
                callInfo.OKeyValueList = []irs.KeyValue{}
        }
	// if tagging is success
        callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "true"} )

        return callInfo, nil
}
