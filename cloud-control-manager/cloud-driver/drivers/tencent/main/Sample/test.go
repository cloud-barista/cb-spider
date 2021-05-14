package main

import (
	"fmt"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"

	testconf "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent/main/conf"
)

func main() {
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
