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
	CreateCIMApiTest()
	DocTypeCIMApiTest()

	SimpleCCMApiTest()
	CreateCCMApiTest()
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

	/* 서버가 TLS 가 설정된 경우
	err = cim.SetTLSCA(os.Getenv("CBSPIDER_ROOT") + "/certs/ca.crt")
	if err != nil {
		logger.Fatal(err)
	}
	*/

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

// CreateCIMApiTest - 파라미터를 이용한 Create CIM API 호출
func CreateCIMApiTest() {

	fmt.Print("\n\n============= CreateCIMApiTest() =============\n")

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

	reqCloudDriver := &api.CloudDriverReq{
		DriverName:        "openstack-driver01",
		ProviderName:      "OPENSTACK",
		DriverLibFileName: "openstack-driver-v1.0.so",
	}
	result, err := cim.CreateCloudDriverByParam(reqCloudDriver)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("\nresult :\n%s\n", result)

	reqCredential := &api.CredentialReq{
		CredentialName: "openstack-credential01",
		ProviderName:   "OPENSTACK",
		KeyValueInfoList: []api.KeyValue{
			api.KeyValue{Key: "IdentityEndpoint", Value: "http://192.168.201.208:5000/v3"},
			api.KeyValue{Key: "Username", Value: "demo"},
			api.KeyValue{Key: "Password", Value: "openstack"},
			api.KeyValue{Key: "DomainName", Value: "Default"},
			api.KeyValue{Key: "ProjectID", Value: "b31474c562184bcbaf3496e08f5a6a4c"},
		},
	}
	result, err = cim.CreateCredentialByParam(reqCredential)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("\nresult :\n%s\n", result)

	reqRegion := &api.RegionReq{
		RegionName:   "openstack-region01",
		ProviderName: "OPENSTACK",
		KeyValueInfoList: []api.KeyValue{
			api.KeyValue{Key: "Region", Value: "RegionOne"},
		},
	}
	result, err = cim.CreateRegionByParam(reqRegion)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("\nresult :\n%s\n", result)

	reqConnectionConfig := &api.ConnectionConfigReq{
		ConfigName:     "openstack-config01",
		ProviderName:   "OPENSTACK",
		DriverName:     "openstack-driver01",
		CredentialName: "openstack-credential01",
		RegionName:     "openstack-region01",
	}
	result, err = cim.CreateConnectionConfigByParam(reqConnectionConfig)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("\nresult :\n%s\n", result)

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

	/* 서버가 TLS 가 설정된 경우
	err = ccm.SetTLSCA(os.Getenv("CBSPIDER_ROOT") + "/certs/ca.crt")
	if err != nil {
		logger.Fatal(err)
	}
	*/

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

// CreateCCMApiTest - 파라미터를 이용한 Create CCM API 호출
func CreateCCMApiTest() {

	fmt.Print("\n\n============= CreateCCMApiTest() =============\n")

	logger := logger.NewLogger()

	ccm := api.NewCloudInfoResourceHandler()

	err := ccm.SetConfigPath("../../grpc_conf.yaml")
	if err != nil {
		logger.Fatal(err)
	}

	err = ccm.Open()
	if err != nil {
		logger.Fatal(err)
	}

	reqVPC := &api.VPCReq{
		ConnectionName: "openstack-config01",
		ReqInfo: api.VPCInfo{
			Name:      "vpc-01",
			IPv4_CIDR: "10.0.1.0/24",
			SubnetInfoList: &[]api.SubnetInfo{
				api.SubnetInfo{
					Name:      "subnet-01",
					IPv4_CIDR: "10.0.1.0/24",
				},
			},
		},
	}
	result, err := ccm.CreateVPCByParam(reqVPC)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("\nresult :\n%s\n", result)

	reqSecurity := &api.SecurityReq{
		ConnectionName: "openstack-config01",
		ReqInfo: api.SecurityInfo{
			Name:    "sg-01",
			VPCName: "vpc-01",
			SecurityRules: &[]api.SecurityRuleInfo{
				api.SecurityRuleInfo{
					FromPort:   "1",
					ToPort:     "65535",
					IPProtocol: "tcp",
					Direction:  "inbound",
				},
				api.SecurityRuleInfo{
					FromPort:   "",
					ToPort:     "",
					IPProtocol: "icmp",
					Direction:  "inbound",
				},
			},
		},
	}
	result, err = ccm.CreateSecurityByParam(reqSecurity)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("\nresult :\n%s\n", result)

	reqKey := &api.KeyReq{
		ConnectionName: "openstack-config01",
		ReqInfo: api.KeyInfo{
			Name: "keypair-01",
		},
	}
	result, err = ccm.CreateKeyByParam(reqKey)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("\nresult :\n%s\n", result)

	reqVM := &api.VMReq{
		ConnectionName: "openstack-config01",
		ReqInfo: api.VMInfo{
			Name:               "vm-01",
			ImageName:          "cirros-0.5.1",
			VPCName:            "vpc-01",
			SubnetName:         "subnet-01",
			SecurityGroupNames: []string{"sg-01"},
			VMSpecName:         "m1.tiny",
			KeyPairName:        "keypair-01",
		},
	}
	result, err = ccm.StartVMByParam(reqVM)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("\nresult :\n%s\n", result)

	ccm.Close()
}
