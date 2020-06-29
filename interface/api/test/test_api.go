package main

import (
	"fmt"
	"time"

	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
	"github.com/cloud-barista/cb-spider/interface/api"
)

func main() {
	SimpleCIMApiTest()
	ConfigCIMApiTest()
	DocTypeCIMApiTest()

	SimpleCCMApiTest()
}

// SimpleCIMApiTest - 간단한 CIM API 호출
func SimpleCIMApiTest() {

	fmt.Print("\n\n============= SimpleCIMApiTest() =============\n")

	logger := logger.NewLogger()

	cim := api.NewCloudInfoManager()

	err := cim.SetServerAddr("localhost:50251")
	if err != nil {
		logger.Fatal(err)
	}

	err = cim.SetTimeout(90 * time.Second)
	if err != nil {
		logger.Fatal(err)
	}

	/* 서버가 JWT 인증이 설정된 경우
	err = cim.SetJWTToken("xxxxxxxxxxxxxxxxxxx")
	if err != nil {
		logger.Fatal(err)
	}
	*/

	err = cim.Open()
	if err != nil {
		logger.Fatal(err)
	}

	result, err := cim.ListCloudOS()
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("\nresult :\n%s\n", result)

	cim.Close()
}

// ConfigCIMApiTest - 환경설정파일을 이용한 CIM API 호출
func ConfigCIMApiTest() {

	fmt.Print("\n\n============= ConfigCIMApiTest() =============\n")

	logger := logger.NewLogger()

	cim := api.NewCloudInfoManager()

	err := cim.SetConfigPath("../../grpc_conf.yaml")
	if err != nil {
		logger.Fatal(err)
	}

	err = cim.Open()
	if err != nil {
		logger.Fatal(err)
	}

	result, err := cim.ListCloudOS()
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("\nresult :\n%s\n", result)

	cim.Close()
}

// DocTypeCIMApiTest - 입력/출력 타입을 이용한 CIM API 호출
func DocTypeCIMApiTest() {

	fmt.Print("\n\n============= DocTypeCIMApiTest() =============\n")

	logger := logger.NewLogger()

	cim := api.NewCloudInfoManager()

	err := cim.SetConfigPath("../../grpc_conf.yaml")
	if err != nil {
		logger.Fatal(err)
	}

	err = cim.Open()
	if err != nil {
		logger.Fatal(err)
	}

	// 입력타입이 json 이고 출력타입이 Json 경우
	err = cim.SetInType("json")
	if err != nil {
		logger.Fatal(err)
	}
	err = cim.SetOutType("json")
	if err != nil {
		logger.Fatal(err)
	}

	doc := `{
		"DriverName":"openstack-driver01"
	}`
	result, err := cim.GetCloudDriver(doc)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("\njson result :\n%s\n", result)

	// 출력타입을 yaml 로 변경
	err = cim.SetOutType("yaml")
	if err != nil {
		logger.Fatal(err)
	}

	result, err = cim.GetCloudDriver(doc)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("\nyaml result :\n%s\n", result)

	// 입력타입을 yaml 로 변경
	err = cim.SetInType("yaml")
	if err != nil {
		logger.Fatal(err)
	}

	doc = `
DriverName: openstack-driver01
`
	result, err = cim.GetCloudDriver(doc)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("\nyaml result :\n%s\n", result)

	// 출력타입을 json 로 변경하고 파라미터로 정보 입력
	err = cim.SetOutType("json")
	if err != nil {
		logger.Fatal(err)
	}

	result, err = cim.GetCloudDriverByParam("openstack-driver01")
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("\njson result :\n%s\n", result)

	cim.Close()
}

// SimpleCCMApiTest - 간단한 CCM API 호출
func SimpleCCMApiTest() {

	fmt.Print("\n\n============= SimpleCCMApiTest() =============\n")

	logger := logger.NewLogger()

	ccm := api.NewCloudInfoResourceHandler()

	err := ccm.SetServerAddr("localhost:50251")
	if err != nil {
		logger.Fatal(err)
	}

	err = ccm.SetTimeout(90 * time.Second)
	if err != nil {
		logger.Fatal(err)
	}

	/* 서버가 JWT 인증이 설정된 경우
	err = ccm.SetJWTToken("xxxxxxxxxxxxxxxxxxx")
	if err != nil {
		logger.Fatal(err)
	}
	*/

	err = ccm.Open()
	if err != nil {
		logger.Fatal(err)
	}

	result, err := ccm.ListVMStatusByParam("openstack-driver01")
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("\nresult :\n%s\n", result)

	ccm.Close()
}
