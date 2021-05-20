package main

import (
	"fmt"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"

	testconf "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent/main/conf"
)

func parsePort() {
	var fromPort string
	var toPort string
	policyGroups := "1,2-3,4,5-6"

	portArr := strings.Split(policyGroups, ",")
	for _, curPolicy := range portArr {
		portRange := strings.Split(curPolicy, "-")
		fromPort = portRange[0]
		if len(portRange) > 1 {
			toPort = portRange[len(portRange)-1]
		} else {
			toPort = ""
		}

		fmt.Printf("FromPort: %s / ToPort: %s", fromPort, toPort)
	}

	/*
		if strings.EqualFold(*curPolicy.Port, "all") {
			fromPort = "ALL"
			toPort = ""
		} else if strings.Contains(*curPolicy.Port, ",") {
			cblogger.Error("Port 정보에 콤머가 섞여있음 - [" + *curPolicy.Port + "]")
			//콤머 기반에서 다시 "-"도 처리해줘야 함.
			return nil, errors.New("NotSupport Rules: The port policy with a comma is not implemented on the mcb tencent spider driver-port[" + *curPolicy.Port + "]")
		} else {
			portArr := strings.Split(*curPolicy.Port, "-")
			fromPort = portArr[0]
			if len(portArr) > 1 {
				toPort = portArr[len(portArr)-1]
			} else {
				toPort = ""
			}
		}
	*/
}

func main() {
	parsePort()
	return

	config := testconf.ReadConfigFile()
	//reqRegion := config.Tencent.Region

	// Instantiate an authentication object. The Tencent Cloud account secretId and secretKey need to be passed in as the input parameters.
	credential := common.NewCredential(
		config.Tencent.SecretId,
		config.Tencent.SecretKey,
	)

	// Instantiate a client configuration object; you can specify the timeout and other configurations.
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.ReqMethod = "GET"
	cpf.HttpProfile.ReqTimeout = 5
	cpf.SignMethod = "HmacSHA1"

	// Instantiate the client object to request the product (with CVM as an example).
	client, _ := cvm.NewClient(credential, "ap-tokyo", cpf)
	// Instantiate a request object; you can further set the request parameters according to the API called and actual conditions.
	request := cvm.NewDescribeZonesRequest()
	// Call the API you want to access through the client object; you need to pass in the request object.
	response, err := client.DescribeZones(request)
	// Handle the exception
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		fmt.Printf("An API error has returned: %s", err)
		return
	}
	// unexpected errors
	if err != nil {
		panic(err)
	}
	// Print the returned json string
	fmt.Printf("%s", response.ToJsonString())
}
