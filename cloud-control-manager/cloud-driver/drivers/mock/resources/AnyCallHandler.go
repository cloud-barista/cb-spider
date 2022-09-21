// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mock Driver.
//
// by CB-Spider Team, 2020.09.

package resources

import (
	"strconv"
	"errors"

	cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	_ "github.com/sirupsen/logrus"
)


type MockAnyCallHandler struct {
	MockName string
}


/********************************************************
        // call example
        curl -sX POST http://localhost:1024/spider/anycall -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName" : "mock-config01",
                "ReqInfo" : {
                        "FID" : "countAll",
                        "IKeyValueList" : [{"Key":"rsType", "Value":"vpc"}]
                }
        }' | json_pp
********************************************************/
func (anyCallHandler *MockAnyCallHandler) AnyCall(callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called AnyCall()!")

	switch callInfo.FID {
	case "countAll" : 
		return countAll(anyCallHandler, callInfo)

	// add more ...

	default :
		return irs.AnyCallInfo{}, errors.New("Mock Driver: " + callInfo.FID + " Function is not implemented!")
	}
}


///////////////////////////////////////////////////////////////////
// implemented by developer user, like 'countAll(rsType string) int'
///////////////////////////////////////////////////////////////////
func countAll(anyCallHandler *MockAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called AnyCall()/countAll()!")

	mockName := anyCallHandler.MockName

	// Input Arg Validation
	if callInfo.IKeyValueList == nil {
		return irs.AnyCallInfo{}, errors.New("Mock Driver: " + callInfo.FID + "'s Argument is empty!")
	}
	if callInfo.IKeyValueList[0].Key != "rsType" {
		return irs.AnyCallInfo{}, errors.New("Mock Driver: " + callInfo.FID + "'s Argument is not 'rsType'!")
	}

	// get info
	strCount := ""
	switch callInfo.IKeyValueList[0].Value {
	case "vpc": 
		infoList, ok := vpcInfoMap[mockName]
		if !ok {
			strCount = "0"	
		} else {
			strCount = strconv.Itoa(len(infoList))
		}
	case "sg": 
		infoList, ok := securityInfoMap[mockName]
		if !ok {
			strCount = "0"	
		} else {
			strCount = strconv.Itoa(len(infoList))
		}
	}

	// make results
	if callInfo.OKeyValueList == nil {
		callInfo.OKeyValueList = []irs.KeyValue{}
	}
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Count", strCount} )

        return callInfo, nil
}

