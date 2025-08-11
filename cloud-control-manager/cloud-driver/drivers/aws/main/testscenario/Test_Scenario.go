package testscenario

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"time"

	awsdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"os/exec"

	cblog "github.com/cloud-barista/cb-log"
)

var cblogger *logrus.Logger

// AWS 리소스 설정 - 사용자가 쉽게 수정할 수 있도록 변수로 정의
const (
	// VPC 및 네트워크 설정
	VPC_ID  = "vpc-0a48d45f6bc3a71da"
	ZONE_ID = "ap-northeast-2a"
)

// 서브넷과 보안 그룹 정보를 구조체로 관리
type SubnetConfig struct {
	SubnetID       string
	SecurityGroups []string
}

// 테스트용 서브넷 설정 - 사용자가 쉽게 수정할 수 있도록 구조체로 정의
var TestSubnets = map[string]SubnetConfig{
	"subnet-1": {
		SubnetID:       "subnet-04bd8bcbeb8cf7748",
		SecurityGroups: []string{"sg-xxxxxxxxx"}, // 실제 보안 그룹 ID로 수정 필요
	},
	"subnet-2": {
		SubnetID:       "subnet-08124f8bc6b14d6c9",
		SecurityGroups: []string{"sg-xxxxxxxxx"}, // 실제 보안 그룹 ID로 수정 필요
	},
}

// 실행할 시나리오 목록 정의 (이 목록에 있는 시나리오만 실행됨)
var EXECUTE_SCENARIOS = []string{
	"1.1", "1.2", "1.3", "1.4",
	"2.1", "2.2", "2.3",
	"3.1", "3.2",
	"4.1",
	"5.1", "5.2",
	"6.1",
	"7.1", "7.2", "7.3", "7.4",
	"8.1", "8.2",
	"9.1", "9.3",
	"10.1", "10.2", "10.3",
	// 실행하지 않을 시나리오는 이 목록에서 제거하거나 주석 처리
	// "9.2", // 비용이 많이 드는 시나리오 (1024 MiB/s provisioned throughput)
}

func defineTestScenarios() []TestScenario {
	return []TestScenario{
		// 1. 기본 설정 모드 (Basic Setup Mode)
		{
			ID:          "1.1",
			Description: "최소 필수 설정",
			Purpose:     "기본 설정 모드의 최소 필수 설정 테스트",
			Request: irs.FileSystemInfo{
				IId:    irs.IID{NameId: "01.01-efs-basic-01"},
				VpcIID: irs.IID{SystemId: VPC_ID},
			},
			Expected: "성공 - 기본값 적용",
		},
		{
			ID:          "1.2",
			Description: "VPC 없이 호출",
			Purpose:     "VPC 필수 요구사항 검증",
			Request: irs.FileSystemInfo{
				IId:    irs.IID{NameId: "01.02-efs-no-vpc"},
				VpcIID: irs.IID{SystemId: ""},
			},
			Expected: "실패 - VPC is required for AWS EFS file system creation",
		},
		{
			ID:          "1.3",
			Description: "태그 처리 (Name Tag 미지정)",
			Purpose:     "태그 처리 및 Name 태그 자동 추가 검증",
			Request: irs.FileSystemInfo{
				IId:    irs.IID{NameId: "01.03-efs-with-tags"},
				VpcIID: irs.IID{SystemId: VPC_ID},
				TagList: []irs.KeyValue{
					{Key: "Environment", Value: "Production"},
					{Key: "Project", Value: "TestProject"},
				},
			},
			Expected: "성공 - 사용자 태그 + Name 태그 자동 추가",
		},
		{
			ID:          "1.4",
			Description: "Name 태그가 있는 경우",
			Purpose:     "사용자 정의 Name 태그 우선순위 검증",
			Request: irs.FileSystemInfo{
				IId:    irs.IID{NameId: "01.04-efs-name-tag-exists"},
				VpcIID: irs.IID{SystemId: VPC_ID},
				TagList: []irs.KeyValue{
					{Key: "Name", Value: "CustomName"},
					{Key: "Environment", Value: "Dev"},
				},
			},
			Expected: "성공 - 사용자 Name 태그 사용",
		},

		// 2. 고급 설정 모드 (Advanced Setup Mode)
		{
			ID:          "2.1",
			Description: "RegionType (Multi-AZ) + 기본 성능 설정",
			Purpose:     "Multi-AZ EFS 기본 생성 테스트",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "02.01-efs-region-basic"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.RegionType,
				Encryption:     true,
				NFSVersion:     "4.1",
			},
			Expected: "성공 - Multi-AZ EFS 생성",
		},
		{
			ID:          "2.2",
			Description: "ZoneType (One Zone) + 기본 성능 설정",
			Purpose:     "One Zone EFS 기본 생성 테스트",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "02.02-efs-zone-basic"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.ZoneType,
				Zone:           ZONE_ID,
				Encryption:     true,
				NFSVersion:     "4.1",
			},
			Expected: "성공 - One Zone EFS 생성",
		},
		{
			ID:          "2.3",
			Description: "ZoneType + Zone 미지정",
			Purpose:     "Zone 자동 결정 기능 테스트",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "02.03-efs-zone-auto"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.ZoneType,
				Encryption:     true,
				NFSVersion:     "4.1",
			},
			Expected: "성공 - Zone 자동 결정",
		},

		// 3. 성능 설정 테스트
		{
			ID:          "3.1",
			Description: "Elastic + GeneralPurpose (권장 조합)",
			Purpose:     "Elastic + GeneralPurpose 성능 조합 테스트",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "03.01-efs-elastic-gp"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.RegionType,
				PerformanceInfo: map[string]string{
					"ThroughputMode":  "Elastic",
					"PerformanceMode": "GeneralPurpose",
				},
			},
			Expected: "성공 - Elastic + GeneralPurpose",
		},
		{
			ID:          "3.2",
			Description: "Bursting + MaxIO",
			Purpose:     "Bursting + MaxIO 성능 조합 테스트",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "03.02-efs-bursting-maxio"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.RegionType,
				PerformanceInfo: map[string]string{
					"ThroughputMode":  "Bursting",
					"PerformanceMode": "MaxIO",
				},
			},
			Expected: "성공 - Bursting + MaxIO",
		},

		// 4. One Zone + MaxIO 에러 테스트
		{
			ID:          "4.1",
			Description: "One Zone + MaxIO (에러 발생해야 함)",
			Purpose:     "One Zone에서 MaxIO 제한 검증",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "04.01-efs-onezone-maxio-error"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.ZoneType,
				Zone:           ZONE_ID,
				PerformanceInfo: map[string]string{
					"ThroughputMode":  "Bursting",
					"PerformanceMode": "MaxIO",
				},
			},
			Expected: "실패 - One Zone에서는 MaxIO 성능 모드를 지원하지 않음",
		},

		// 5. 암호화 설정 테스트
		{
			ID:          "5.1",
			Description: "암호화 활성화",
			Purpose:     "암호화 활성화 테스트",
			Request: irs.FileSystemInfo{
				IId:        irs.IID{NameId: "05.01-efs-encrypted"},
				VpcIID:     irs.IID{SystemId: VPC_ID},
				Encryption: true,
			},
			Expected: "성공 - 암호화된 EFS 생성",
		},
		{
			ID:          "5.2",
			Description: "암호화 비활성화",
			Purpose:     "암호화 비활성화 테스트",
			Request: irs.FileSystemInfo{
				IId:        irs.IID{NameId: "05.02-efs-not-encrypted"},
				VpcIID:     irs.IID{SystemId: VPC_ID},
				Encryption: false,
			},
			Expected: "성공 - 암호화되지 않은 EFS 생성",
		},

		// 6. NFS 버전 테스트
		{
			ID:          "6.1",
			Description: "NFS 4.1 버전",
			Purpose:     "NFS 버전 설정 테스트",
			Request: irs.FileSystemInfo{
				IId:        irs.IID{NameId: "06.01-efs-nfs41"},
				VpcIID:     irs.IID{SystemId: VPC_ID},
				NFSVersion: "4.1",
			},
			Expected: "성공 - NFS 4.1 버전 EFS 생성",
		},

		// 7. 마운트 타겟 생성 테스트
		{
			ID:          "7.1",
			Description: "AccessSubnetList 사용 (공식 기능)",
			Purpose:     "AccessSubnetList를 통한 마운트 타겟 생성 테스트",
			Request: irs.FileSystemInfo{
				IId:              irs.IID{NameId: "07.01-efs-access-subnets"},
				VpcIID:           irs.IID{SystemId: VPC_ID},
				FileSystemType:   irs.RegionType,
				AccessSubnetList: createAccessSubnetList("subnet-1", "subnet-2"),
			},
			Expected: "성공 - 2개의 마운트 타겟 생성, 기본 보안 그룹 사용",
		},
		{
			ID:          "7.2",
			Description: "AccessSubnetList - One Zone 제약사항",
			Purpose:     "One Zone EFS 마운트 타겟 제한 검증",
			Request: irs.FileSystemInfo{
				IId:              irs.IID{NameId: "07.02-efs-zone-access-error"},
				VpcIID:           irs.IID{SystemId: VPC_ID},
				FileSystemType:   irs.ZoneType,
				Zone:             ZONE_ID,
				AccessSubnetList: createAccessSubnetList("subnet-1", "subnet-2"),
			},
			Expected: "실패 - One Zone EFS can only have 1 mount target, but 2 subnets were specified",
		},
		{
			ID:          "7.3",
			Description: "MountTargetList 사용 (보안 그룹 지정)",
			Purpose:     "MountTargetList를 통한 보안 그룹 지정 테스트",
			Request: irs.FileSystemInfo{
				IId:             irs.IID{NameId: "07.03-efs-mount-targets"},
				VpcIID:          irs.IID{SystemId: VPC_ID},
				FileSystemType:  irs.RegionType,
				MountTargetList: createMountTargetList("subnet-1", "subnet-2"),
			},
			Expected: "성공 - 2개의 마운트 타겟 생성, 지정된 보안 그룹 사용",
		},
		{
			ID:          "7.4",
			Description: "MountTargetList - One Zone 제약사항",
			Purpose:     "MountTargetList One Zone 제약사항 검증",
			Request: irs.FileSystemInfo{
				IId:             irs.IID{NameId: "07.04-efs-zone-mount-error"},
				VpcIID:          irs.IID{SystemId: VPC_ID},
				FileSystemType:  irs.ZoneType,
				Zone:            ZONE_ID,
				MountTargetList: createMountTargetList("subnet-1", "subnet-2"),
			},
			Expected: "실패 - One Zone EFS can only have 1 mount target, but 2 were specified",
		},

		// 8. 복합 시나리오 테스트
		{
			ID:          "8.1",
			Description: "완전한 고급 설정",
			Purpose:     "복합 고급 설정 테스트",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "08.01-efs-complete-advanced"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.RegionType,
				Zone:           ZONE_ID,
				Encryption:     true,
				NFSVersion:     "4.1",
				PerformanceInfo: map[string]string{
					"ThroughputMode":        "Provisioned",
					"PerformanceMode":       "MaxIO",
					"ProvisionedThroughput": "512",
				},
				TagList: []irs.KeyValue{
					{Key: "Environment", Value: "Production"},
					{Key: "CostCenter", Value: "IT-001"},
				},
			},
			Expected: "성공 - Multi-AZ EFS + Provisioned + MaxIO + 암호화 + 태그",
		},
		{
			ID:          "8.2",
			Description: "One Zone 완전 설정",
			Purpose:     "One Zone 복합 설정 테스트",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "08.02-efs-onezone-complete"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.ZoneType,
				Zone:           ZONE_ID,
				Encryption:     true,
				NFSVersion:     "4.1",
				PerformanceInfo: map[string]string{
					"ThroughputMode":        "Provisioned",
					"PerformanceMode":       "GeneralPurpose",
					"ProvisionedThroughput": "128",
				},
				TagList: []irs.KeyValue{
					{Key: "Environment", Value: "Development"},
					{Key: "Backup", Value: "Daily"},
				},
			},
			Expected: "성공 - One Zone EFS + Provisioned + GeneralPurpose + 암호화 + 태그",
		},

		// 9. 경계값 테스트
		{
			ID:          "9.1",
			Description: "최소 ProvisionedThroughput",
			Purpose:     "최소 ProvisionedThroughput 경계값 테스트",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "09.01-efs-min-throughput"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.RegionType,
				PerformanceInfo: map[string]string{
					"ThroughputMode":        "Provisioned",
					"PerformanceMode":       "GeneralPurpose",
					"ProvisionedThroughput": "1",
				},
			},
			Expected: "성공 - 1 MiB/s provisioned throughput",
		},
		{
			ID:          "9.2",
			Description: "최대 ProvisionedThroughput",
			Purpose:     "최대 ProvisionedThroughput 경계값 테스트 (비용이 많이 발생)",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "09.02-efs-max-throughput"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.RegionType,
				PerformanceInfo: map[string]string{
					"ThroughputMode":        "Provisioned",
					"PerformanceMode":       "GeneralPurpose",
					"ProvisionedThroughput": "1024",
				},
			},
			Expected: "성공 - 1024 MiB/s provisioned throughput",
		},
		{
			ID:          "9.3",
			Description: "최대 ProvisionedThroughput 초과",
			Purpose:     "최대 ProvisionedThroughput 초과 검증",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "09.03-efs-throughput-overflow"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.RegionType,
				PerformanceInfo: map[string]string{
					"ThroughputMode":        "Provisioned",
					"PerformanceMode":       "GeneralPurpose",
					"ProvisionedThroughput": "1025",
				},
			},
			Expected: "실패 - provisioned throughput must be between 1 and 1024 MiB/s",
		},

		// 10. 특수 케이스 테스트
		{
			ID:          "10.1",
			Description: "빈 이름 (Name이 필수가 아님)",
			Purpose:     "빈 이름 허용 검증",
			Request: irs.FileSystemInfo{
				IId:    irs.IID{NameId: ""},
				VpcIID: irs.IID{SystemId: VPC_ID},
			},
			Expected: "성공 - AWS EFS는 Name이 필수가 아님",
		},
		{
			ID:          "10.2",
			Description: "매우 긴 이름 (128자)",
			Purpose:     "긴 이름(128자) 지원 검증",
			Request: irs.FileSystemInfo{
				IId:    irs.IID{NameId: createLongString(128)},
				VpcIID: irs.IID{SystemId: VPC_ID},
			},
			Expected: "성공 - AWS EFS는 최대 128자 이름 지원",
		},
		{
			ID:          "10.3",
			Description: "매우 긴 이름 (257자)",
			Purpose:     "긴 이름(257자) 제한 검증",
			Request: irs.FileSystemInfo{
				IId:    irs.IID{NameId: createLongString(257)},
				VpcIID: irs.IID{SystemId: VPC_ID},
			},
			Expected: "실패 - AWS EFS는 256자를 초과하는 이름을 지원하지 않음",
		},
	}
}

// 헬퍼 함수: AccessSubnetList 생성
func createAccessSubnetList(subnetKeys ...string) []irs.IID {
	var subnets []irs.IID
	for _, key := range subnetKeys {
		if config, exists := TestSubnets[key]; exists {
			subnets = append(subnets, irs.IID{SystemId: config.SubnetID})
		}
	}
	return subnets
}

// 헬퍼 함수: MountTargetList 생성
func createMountTargetList(subnetKeys ...string) []irs.MountTargetInfo {
	var mountTargets []irs.MountTargetInfo
	for _, key := range subnetKeys {
		if config, exists := TestSubnets[key]; exists {
			mountTarget := irs.MountTargetInfo{
				SubnetIID:      irs.IID{SystemId: config.SubnetID},
				SecurityGroups: config.SecurityGroups,
			}
			mountTargets = append(mountTargets, mountTarget)
		}
	}
	return mountTargets
}

// 브라우저에서 HTML 파일 열기 (원격 환경 고려)
func openBrowser(filename string) {
	// 원격 환경에서는 브라우저 열기가 어려울 수 있으므로 HTTP 서버 시작을 제안
	cblogger.Info("=== Test Report Generated Successfully ===")
	cblogger.Infof("File: %s", filename)
	cblogger.Info("")
	cblogger.Info("To view the report in a remote environment:")
	cblogger.Info("1. Start HTTP server: python3 -m http.server 8080")
	cblogger.Info("2. Use SSH tunnel: ssh -L 8080:localhost:8080 user@remote-server")
	cblogger.Info("3. Open browser: http://localhost:8080/Test_Scenario_Result.html")
	cblogger.Info("")

	// 로컬 환경에서만 브라우저 열기 시도
	if os.Getenv("DISPLAY") != "" || runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		var cmd *exec.Cmd

		switch runtime.GOOS {
		case "linux":
			cmd = exec.Command("xdg-open", filename)
		case "darwin":
			cmd = exec.Command("open", filename)
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", filename)
		default:
			return
		}

		err := cmd.Start()
		if err != nil {
			cblogger.Warnf("Failed to open browser: %v", err)
		} else {
			cblogger.Infof("Opened test report in browser: %s", filename)
		}
	} else {
		cblogger.Info("Running in headless environment - browser not opened automatically")
	}
}

func init() {
	fmt.Println("Test Scenario init start")
	cblogger = cblog.GetLogger("AWS EFS Test Scenario")
	cblog.SetLevel("info")
}

// TestResult represents the result of a test case
type TestResult struct {
	ScenarioID      string
	Description     string
	Expected        string
	Actual          string
	Success         bool
	ErrorMessage    string
	Duration        time.Duration
	FileSystemID    string // 실제 생성된 파일시스템 ID
	RequestInfo     string // 요청 정보 요약
	ResponseInfo    string // 응답 정보 요약 (성공한 경우)
	Validation      string // 검증 결과
	ScenarioSuccess bool   // 시나리오 성공 여부 (실행 실패가 예상된 경우 true)
	Skipped         bool   // 시나리오가 Skip(주석처리 등)된 경우 true
}

// TestScenario represents a test scenario
type TestScenario struct {
	ID          string
	Description string
	Purpose     string // 테스트 목적 추가
	Request     irs.FileSystemInfo
	Expected    string
}

// Config struct for AWS credentials
type Config struct {
	Aws struct {
		AwsAccessKeyID     string `yaml:"aws_access_key_id"`
		AwsSecretAccessKey string `yaml:"aws_secret_access_key"`
		AwsStsToken        string `yaml:"aws_sts_token"`
		Region             string `yaml:"region"`
		Zone               string `yaml:"zone"`
	} `yaml:"aws"`
}

// TestScenarioFileSystem 함수 - Test_Resources.go에서 호출됨
func TestScenarioFileSystem() {
	fmt.Println("=== AWS EFS Test Scenario Execution ===")

	// Get FileSystem handler
	handler, err := getFileSystemHandler()
	if err != nil {
		cblogger.Errorf("Failed to get FileSystem handler: %v", err)
		return
	}

	// Define test scenarios based on the documentation
	allScenarios := defineTestScenarios()

	// Filter scenarios to execute based on EXECUTE_SCENARIOS
	var testScenarios []TestScenario
	executeMap := make(map[string]bool)
	for _, id := range EXECUTE_SCENARIOS {
		executeMap[id] = true
	}

	for _, scenario := range allScenarios {
		if executeMap[scenario.ID] {
			testScenarios = append(testScenarios, scenario)
		}
	}

	cblogger.Infof("Executing %d scenarios out of %d total scenarios", len(testScenarios), len(allScenarios))

	// Execute tests
	results := executeTestScenarios(handler, testScenarios)

	// Generate test report
	generateTestReport(results)
}

func getFileSystemHandler() (irs.FileSystemHandler, error) {
	cloudDriver := new(awsdrv.AwsDriver)

	config := readConfigFile()
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientId:     config.Aws.AwsAccessKeyID,
			ClientSecret: config.Aws.AwsSecretAccessKey,
			StsToken:     config.Aws.AwsStsToken,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.Aws.Region,
			Zone:   config.Aws.Zone,
		},
	}

	cloudConnection, errCon := cloudDriver.ConnectCloud(connectionInfo)
	if errCon != nil {
		return nil, errCon
	}

	fileSystemHandler, err := cloudConnection.CreateFileSystemHandler()
	if err != nil {
		return nil, err
	}
	return fileSystemHandler, nil
}

func executeTestScenarios(handler irs.FileSystemHandler, scenarios []TestScenario) []TestResult {
	var results []TestResult
	var executedIDs = make(map[string]bool)

	for _, scenario := range scenarios {
		cblogger.Infof("\n\n================================================\n=== Executing Test Scenario %s: %s ===\n================================================", scenario.ID, scenario.Description)

		start := time.Now()
		result, err := handler.CreateFileSystem(scenario.Request)
		duration := time.Since(start)

		testResult := TestResult{
			ScenarioID:  scenario.ID,
			Description: scenario.Description,
			Expected:    scenario.Expected,
			Duration:    duration,
		}

		// 요청 정보 요약 생성
		requestInfo := fmt.Sprintf("Name: %s, VPC: %s", scenario.Request.IId.NameId, scenario.Request.VpcIID.SystemId)
		if scenario.Request.FileSystemType != "" {
			requestInfo += fmt.Sprintf(", Type: %s", scenario.Request.FileSystemType)
		}
		if scenario.Request.Zone != "" {
			requestInfo += fmt.Sprintf(", Zone: %s", scenario.Request.Zone)
		}
		if scenario.Request.Encryption {
			requestInfo += ", Encryption: true"
		}
		if scenario.Request.NFSVersion != "" {
			requestInfo += fmt.Sprintf(", NFS: %s", scenario.Request.NFSVersion)
		}
		if scenario.Request.PerformanceInfo != nil {
			if throughput, ok := scenario.Request.PerformanceInfo["ThroughputMode"]; ok {
				requestInfo += fmt.Sprintf(", Throughput: %s", throughput)
			}
			if performance, ok := scenario.Request.PerformanceInfo["PerformanceMode"]; ok {
				requestInfo += fmt.Sprintf(", Performance: %s", performance)
			}
			if provisioned, ok := scenario.Request.PerformanceInfo["ProvisionedThroughput"]; ok {
				requestInfo += fmt.Sprintf(", Provisioned: %s MiB/s", provisioned)
			}
		}
		if len(scenario.Request.TagList) > 0 {
			requestInfo += fmt.Sprintf(", Tags: %d개", len(scenario.Request.TagList))
		}
		if len(scenario.Request.AccessSubnetList) > 0 {
			requestInfo += fmt.Sprintf(", AccessSubnets: %d개", len(scenario.Request.AccessSubnetList))
		}
		testResult.RequestInfo = requestInfo

		if err != nil {
			testResult.Success = false
			testResult.Actual = "실패"
			testResult.ErrorMessage = err.Error()
			testResult.FileSystemID = ""
			testResult.ResponseInfo = ""
			testResult.Validation = ""
			// 시나리오 성공 여부 판단
			if strings.HasPrefix(scenario.Expected, "실패") {
				// '실패 - ' 이후의 메시지를 추출
				expectedMsg := strings.TrimSpace(strings.TrimPrefix(scenario.Expected, "실패 -"))
				if expectedMsg != "" && strings.Contains(err.Error(), expectedMsg) {
					testResult.ScenarioSuccess = true
					cblogger.Infof("Test Scenario %s SUCCESS (Expected Failure): %v", scenario.ID, err)
				} else {
					testResult.ScenarioSuccess = false
					cblogger.Errorf("Test Scenario %s FAILED (Unexpected Error): %v", scenario.ID, err)
				}
			} else {
				testResult.ScenarioSuccess = false
				cblogger.Errorf("Test Scenario %s FAILED: %v", scenario.ID, err)
			}
		} else {
			if result.IId.SystemId != "" {
				testResult.Success = true
				testResult.Actual = "성공"
				testResult.ErrorMessage = ""
				testResult.FileSystemID = result.IId.SystemId
				cblogger.Infof("Test Scenario %s SUCCESS: %s", scenario.ID, result.IId.SystemId)

				// 생성된 EFS 검증
				_, responseInfo, validation := validateFileSystemCreation(handler, scenario.Request, result.IId.SystemId)
				testResult.ResponseInfo = responseInfo
				testResult.Validation = validation

				// Clean up - delete the created file system
				go func(fsID string) {
					time.Sleep(5 * time.Second) // Wait a bit before deletion
					_, deleteErr := handler.DeleteFileSystem(irs.IID{SystemId: fsID})
					if deleteErr != nil {
						cblogger.Errorf("Failed to delete file system %s: %v", fsID, deleteErr)
					} else {
						cblogger.Infof("Successfully deleted file system %s", fsID)
					}
				}(result.IId.SystemId)
				testResult.ScenarioSuccess = strings.HasPrefix(scenario.Expected, "성공")
			} else {
				testResult.Success = false
				testResult.Actual = "실패"
				testResult.ErrorMessage = "CreateFileSystem returned empty SystemId"
				testResult.FileSystemID = ""
				testResult.ResponseInfo = ""
				testResult.Validation = ""
				testResult.ScenarioSuccess = false
				cblogger.Errorf("Test Scenario %s FAILED: CreateFileSystem returned empty SystemId", scenario.ID)
			}
		}

		results = append(results, testResult)
		executedIDs[scenario.ID] = true

		// Add delay between tests to avoid rate limiting
		time.Sleep(2 * time.Second)
	}

	// Skip 처리: EXECUTE_SCENARIOS에 정의되지 않은 시나리오들을 skip으로 추가
	allScenarios := defineTestScenarios()
	scenarioMap := make(map[string]TestScenario)
	for _, scenario := range allScenarios {
		scenarioMap[scenario.ID] = scenario
	}

	// EXECUTE_SCENARIOS에 없는 시나리오들을 skip으로 추가
	for _, scenario := range allScenarios {
		if !executedIDs[scenario.ID] {
			results = append(results, TestResult{
				ScenarioID:      scenario.ID,
				Description:     scenario.Description,
				Expected:        scenario.Expected,
				Actual:          "-",
				Success:         false,
				ErrorMessage:    "",
				Duration:        0,
				FileSystemID:    "",
				RequestInfo:     "-",
				ResponseInfo:    "-",
				Validation:      "-",
				ScenarioSuccess: false,
				Skipped:         true,
			})
		}
	}

	return results
}

// Helper function to create long strings for testing
func createLongString(length int) string {
	result := ""
	for i := 0; i < length; i++ {
		result += "a"
	}
	return result
}

// validateFileSystemCreation 함수 - 생성된 EFS를 조회하여 요청값과 비교 검증
func validateFileSystemCreation(handler irs.FileSystemHandler, request irs.FileSystemInfo, fileSystemID string) (string, string, string) {
	if fileSystemID == "" {
		return "", "", "FileSystem ID가 없음"
	}

	// 생성된 EFS 조회
	createdFS, err := handler.GetFileSystem(irs.IID{SystemId: fileSystemID})
	if err != nil {
		return "", "", fmt.Sprintf("EFS 조회 실패: %v", err)
	}

	// 요청 정보 요약
	requestInfo := fmt.Sprintf("Name: %s, VPC: %s", request.IId.NameId, request.VpcIID.SystemId)
	if request.FileSystemType != "" {
		requestInfo += fmt.Sprintf(", Type: %s", request.FileSystemType)
	}
	if request.Zone != "" {
		requestInfo += fmt.Sprintf(", Zone: %s", request.Zone)
	}
	if request.Encryption {
		requestInfo += ", Encryption: true"
	}
	if request.NFSVersion != "" {
		requestInfo += fmt.Sprintf(", NFS: %s", request.NFSVersion)
	}
	if request.PerformanceInfo != nil {
		if throughput, ok := request.PerformanceInfo["ThroughputMode"]; ok {
			requestInfo += fmt.Sprintf(", Throughput: %s", throughput)
		}
		if performance, ok := request.PerformanceInfo["PerformanceMode"]; ok {
			requestInfo += fmt.Sprintf(", Performance: %s", performance)
		}
		if provisioned, ok := request.PerformanceInfo["ProvisionedThroughput"]; ok {
			requestInfo += fmt.Sprintf(", Provisioned: %s MiB/s", provisioned)
		}
	}
	if len(request.TagList) > 0 {
		requestInfo += fmt.Sprintf(", Tags: %d개", len(request.TagList))
	}
	if len(request.AccessSubnetList) > 0 {
		requestInfo += fmt.Sprintf(", AccessSubnets: %d개", len(request.AccessSubnetList))
	}

	// 응답 정보 요약
	responseInfo := fmt.Sprintf("ID: %s, Name: %s, VPC: %s",
		createdFS.IId.SystemId, createdFS.IId.NameId, createdFS.VpcIID.SystemId)
	if createdFS.FileSystemType != "" {
		responseInfo += fmt.Sprintf(", Type: %s", createdFS.FileSystemType)
	}
	if createdFS.Zone != "" {
		responseInfo += fmt.Sprintf(", Zone: %s", createdFS.Zone)
	}
	if createdFS.Encryption {
		responseInfo += ", Encryption: true"
	}
	if createdFS.NFSVersion != "" {
		responseInfo += fmt.Sprintf(", NFS: %s", createdFS.NFSVersion)
	}
	if createdFS.PerformanceInfo != nil {
		if throughput, ok := createdFS.PerformanceInfo["ThroughputMode"]; ok {
			responseInfo += fmt.Sprintf(", Throughput: %s", throughput)
		}
		if performance, ok := createdFS.PerformanceInfo["PerformanceMode"]; ok {
			responseInfo += fmt.Sprintf(", Performance: %s", performance)
		}
		if provisioned, ok := createdFS.PerformanceInfo["ProvisionedThroughput"]; ok {
			responseInfo += fmt.Sprintf(", Provisioned: %s MiB/s", provisioned)
		}
	}
	if len(createdFS.TagList) > 0 {
		responseInfo += fmt.Sprintf(", Tags: %d개", len(createdFS.TagList))
	}

	// 검증 결과
	validation := "✅ 검증 통과"

	// 기본 검증
	if request.IId.NameId != "" && createdFS.IId.NameId != request.IId.NameId {
		validation = "❌ Name 불일치"
	}
	if request.VpcIID.SystemId != "" && createdFS.VpcIID.SystemId != request.VpcIID.SystemId {
		validation = "❌ VPC 불일치"
	}
	if request.FileSystemType != "" && createdFS.FileSystemType != request.FileSystemType {
		validation = "❌ FileSystemType 불일치"
	}
	if request.Zone != "" && createdFS.Zone != request.Zone {
		validation = "❌ Zone 불일치"
	}
	if request.Encryption != createdFS.Encryption {
		validation = "❌ Encryption 불일치"
	}
	if request.NFSVersion != "" && createdFS.NFSVersion != request.NFSVersion {
		validation = "❌ NFSVersion 불일치"
	}

	// PerformanceInfo 검증
	if request.PerformanceInfo != nil && createdFS.PerformanceInfo != nil {
		for key, expectedValue := range request.PerformanceInfo {
			if actualValue, exists := createdFS.PerformanceInfo[key]; !exists || actualValue != expectedValue {
				validation = fmt.Sprintf("❌ PerformanceInfo[%s] 불일치 (요청: %s, 실제: %s)", key, expectedValue, actualValue)
				break
			}
		}
	}

	// Tag 검증 (Name 태그는 자동 추가되므로 제외)
	if len(request.TagList) > 0 {
		requestTagCount := 0
		for _, tag := range request.TagList {
			if tag.Key != "Name" { // Name 태그는 자동 추가되므로 제외
				requestTagCount++
			}
		}
		if requestTagCount > 0 && len(createdFS.TagList) < requestTagCount {
			validation = "❌ Tag 개수 불일치"
		}
	}

	return requestInfo, responseInfo, validation
}

func generateTestReport(results []TestResult) {
	// HTML 스타일과 JavaScript를 포함한 보고서 생성
	html := `<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AWS EFS Test Scenario Results</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background-color: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px; }
        h2 { color: #34495e; margin-top: 30px; }
        .summary { background-color: #ecf0f1; padding: 15px; border-radius: 5px; margin: 20px 0; }
        .summary-item { margin: 10px 0; font-weight: bold; }
        .success { color: #27ae60; }
        .failure { color: #e74c3c; }
        .warning { color: #f39c12; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background-color: #3498db; color: white; font-weight: bold; }
        tr:nth-child(even) { background-color: #f9f9f9; }
        tr:hover { background-color: #f1f1f1; }
        .success-row { background-color: #d5f4e6 !important; }
        .failure-row { background-color: #fadbd8 !important; }
        .collapsible { background-color: #f1f1f1; color: #444; cursor: pointer; padding: 18px; width: 100%; border: none; text-align: left; outline: none; font-size: 15px; margin: 5px 0; border-radius: 5px; }
        .active, .collapsible:hover { background-color: #ddd; }
        .content { padding: 0 18px; max-height: 0; overflow: hidden; transition: max-height 0.2s ease-out; background-color: #f9f9f9; border-radius: 5px; }
        .content.show { max-height: 500px; padding: 18px; }
        .log-content { background-color: #2c3e50; color: #ecf0f1; padding: 15px; border-radius: 5px; font-family: 'Courier New', monospace; font-size: 12px; white-space: pre-wrap; max-height: 300px; overflow-y: auto; }
        .status-badge { padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: bold; }
        .status-success { background-color: #27ae60; color: white; }
        .status-failure { background-color: #e74c3c; color: white; }
        .status-warning { background-color: #f39c12; color: white; }
        .toc { background-color: #ecf0f1; padding: 15px; border-radius: 5px; margin: 20px 0; }
        .toc ul { list-style-type: none; padding-left: 0; }
        .toc li { margin: 5px 0; }
        .toc a { text-decoration: none; color: #2c3e50; }
        .toc a:hover { color: #3498db; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 AWS EFS Test Scenario Results</h1>
        <p><strong>Test Execution Time:</strong> ` + time.Now().Format("2006-01-02 15:04:05") + `</p>`

	// 목차 생성
	html += `
        <div class="toc">
            <h3>📋 목차</h3>
            <ul>
                <li><a href="#summary">📊 전체 요약</a></li>
                <li><a href="#scenarios">📋 시나리오 목록</a></li>
                <li><a href="#results">📈 전체 실행 결과</a></li>
                <li><a href="#success">✅ 성공한 시나리오 상세</a></li>
                <li><a href="#failure">❌ 실패한 시나리오 상세</a></li>
                <li><a href="#skipped">⏭️ Skip된 시나리오</a></li>
            </ul>
        </div>`

	// 요약 통계
	successCount := 0
	failureCount := 0
	var failedScenarios []TestResult
	var successScenarios []TestResult

	for _, result := range results {
		if result.Skipped {
			continue // Skip는 통계에서 제외
		}
		if result.ScenarioSuccess {
			successCount++
			successScenarios = append(successScenarios, result)
		} else {
			failureCount++
			failedScenarios = append(failedScenarios, result)
		}
	}

	totalCount := successCount + failureCount
	successRate := 0.0
	if totalCount > 0 {
		successRate = float64(successCount) / float64(totalCount) * 100
	}

	html += `
        <div id="summary">
            <h2>📊 전체 요약</h2>
            <div class="summary">
                <div class="summary-item">총 테스트 수: <span class="success">` + fmt.Sprintf("%d", totalCount) + `</span></div>
                <div class="summary-item">성공: <span class="success">` + fmt.Sprintf("%d", successCount) + `</span></div>
                <div class="summary-item">실패: <span class="failure">` + fmt.Sprintf("%d", failureCount) + `</span></div>
                <div class="summary-item">성공률: <span class="success">` + fmt.Sprintf("%.2f%%", successRate) + `</span></div>
            </div>
        </div>`

	// 시나리오 목록 (목차 역할)
	html += `
        <div id="scenarios">
            <h2>📋 시나리오 목록</h2>
            <table>
                <tr>
                    <th>시나리오 번호</th>
                    <th>시나리오 제목</th>
                    <th>예상 결과</th>
                    <th>테스트 목적</th>
                </tr>`

	// 시나리오 정의를 다시 가져와서 목록 생성
	scenarios := defineTestScenarios()
	for _, scenario := range scenarios {
		html += fmt.Sprintf(`
                <tr>
                    <td><strong>%s</strong></td>
                    <td>%s</td>
                    <td>%s</td>
                    <td>%s</td>
                </tr>`, scenario.ID, scenario.Description, scenario.Expected, scenario.Purpose)
	}

	html += `
            </table>
        </div>`

	// 전체 실행 결과
	html += `
        <div id="results">
            <h2>📈 전체 실행 결과</h2>
            <table>
                <tr>
                    <th>시나리오 번호</th>
                    <th>시나리오 제목</th>
                    <th>예상 결과</th>
                    <th>실행 결과</th>
                    <th>시나리오 결과</th>
                </tr>`

	for _, result := range results {
		// 실행 결과
		statusClass := "status-success"
		statusText := "✅ 성공"
		if !result.Success {
			statusClass = "status-failure"
			statusText = "❌ 실패"
		}
		// 시나리오 결과
		scenarioClass := "status-success"
		scenarioText := "✅ 성공"
		if result.Skipped {
			scenarioClass = "status-warning"
			scenarioText = "⏭️ Skip"
		} else if !result.ScenarioSuccess {
			scenarioClass = "status-failure"
			scenarioText = "❌ 실패"
		}
		html += fmt.Sprintf(`
                <tr class="%s">
                    <td><strong>%s</strong></td>
                    <td>%s</td>
                    <td>%s</td>
                    <td><span class="status-badge %s">%s</span></td>
                    <td><span class="status-badge %s">%s</span></td>
                </tr>`,
			getRowClass(result.Success), result.ScenarioID, result.Description, result.Expected, statusClass, statusText, scenarioClass, scenarioText)
	}

	html += `
            </table>
        </div>`

	// 성공한 시나리오 상세
	if len(successScenarios) > 0 {
		html += `
        <div id="success">
            <h2>✅ 성공한 시나리오 상세</h2>`

		for _, result := range successScenarios {
			html += fmt.Sprintf(`
            <button class="collapsible">%s - %s (실행시간: %s)</button>
            <div class="content">
                <h4>📋 요청 정보</h4>
                <p><strong>%s</strong></p>
                
                <h4>📤 응답 정보</h4>
                <p><strong>%s</strong></p>
                
                <h4>🔍 검증 결과</h4>
                <p><strong>%s</strong></p>
                
                <h4>📝 상세 로그</h4>
                <div class="log-content">FileSystem ID: %s
Duration: %s
Request Info: %s
Response Info: %s
Validation: %s</div>
            </div>`,
				result.ScenarioID, result.Description, result.Duration.String(),
				result.RequestInfo, result.ResponseInfo, result.Validation,
				result.FileSystemID, result.Duration.String(), result.RequestInfo, result.ResponseInfo, result.Validation)
		}
		html += `</div>`
	}

	// 실패한 시나리오 상세
	if len(failedScenarios) > 0 {
		html += `
        <div id="failure">
            <h2>❌ 실패한 시나리오 상세</h2>`

		for _, result := range failedScenarios {
			html += fmt.Sprintf(`
            <button class="collapsible">%s - %s (실행시간: %s)</button>
            <div class="content">
                <h4>📋 요청 정보</h4>
                <p><strong>%s</strong></p>
                
                <h4>❌ 오류 메시지</h4>
                <p><strong>%s</strong></p>
                
                <h4>📝 상세 로그</h4>
                <div class="log-content">Scenario ID: %s
Description: %s
Expected: %s
Actual: %s
Duration: %s
Error Message: %s</div>
            </div>`,
				result.ScenarioID, result.Description, result.Duration.String(),
				result.RequestInfo, result.ErrorMessage,
				result.ScenarioID, result.Description, result.Expected, result.Actual, result.Duration.String(), result.ErrorMessage)
		}
		html += `</div>`
	}

	// Skip된 시나리오 상세
	if len(results) > 0 {
		skipScenarios := []TestResult{}
		for _, result := range results {
			if result.Skipped {
				skipScenarios = append(skipScenarios, result)
			}
		}
		if len(skipScenarios) > 0 {
			html += `
        <div id="skipped">
            <h2>⏭️ Skip된 시나리오 상세</h2>`
			for _, result := range skipScenarios {
				html += fmt.Sprintf(`
            <button class="collapsible">%s - %s</button>
            <div class="content">
                <h4>📋 시나리오 정보</h4>
                <p><strong>시나리오 ID:</strong> %s</p>
                <p><strong>설명:</strong> %s</p>
                <p><strong>예상 결과:</strong> %s</p>
                
                <h4>⏭️ Skip 이유</h4>
                <p>이 시나리오는 비용, 시간, 또는 기타 이유로 인해 실행하지 않았습니다.</p>
                <p>실행하려면 <code>EXECUTE_SCENARIOS</code> 목록에 "%s"를 추가하세요.</p>
            </div>`, result.ScenarioID, result.Description, result.ScenarioID, result.Description, result.Expected, result.ScenarioID)
			}
			html += `</div>`
		}
	}

	// JavaScript for collapsible functionality
	html += `
    </div>
    <script>
        var coll = document.getElementsByClassName("collapsible");
        var i;

        for (i = 0; i < coll.length; i++) {
            coll[i].addEventListener("click", function() {
                this.classList.toggle("active");
                var content = this.nextElementSibling;
                if (content.style.maxHeight) {
                    content.style.maxHeight = null;
                    content.classList.remove("show");
                } else {
                    content.style.maxHeight = content.scrollHeight + "px";
                    content.classList.add("show");
                }
            });
        }
    </script>
</body>
</html>`

	// Write report to file
	filename := "Test_Scenario_Result.html"
	err := ioutil.WriteFile(filename, []byte(html), 0644)
	if err != nil {
		cblogger.Errorf("Failed to write test report: %v", err)
	} else {
		cblogger.Info("Test report written to " + filename)

		// 브라우저에서 자동으로 열기
		cblogger.Info("Opening test report in browser...")
		openBrowser(filename)
	}
}

// Helper functions for HTML generation
func getRowClass(success bool) string {
	if success {
		return "success-row"
	}
	return "failure-row"
}

func readConfigFile() Config {
	// CBSPIDER_TEST_CONF_PATH 환경변수 사용
	confPath := os.Getenv("CBSPIDER_TEST_CONF_PATH")
	if confPath == "" {
		panic("CBSPIDER_TEST_CONF_PATH environment variable is not set")
	}
	cblogger.Infof("설정 파일 경로: [%s]", confPath)

	data, err := ioutil.ReadFile(confPath)
	if err != nil {
		panic(err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}

	return config
}
