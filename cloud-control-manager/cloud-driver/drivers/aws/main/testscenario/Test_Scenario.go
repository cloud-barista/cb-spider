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

// AWS Resource Configuration - Defined as variables for easy user modification
const (
	// VPC and Network Configuration
	VPC_ID  = "vpc-0a48d45f6bc3a71da"
	ZONE_ID = "ap-northeast-2a"
)

// Subnet and Security Group information managed as struct
type SubnetConfig struct {
	SubnetID       string
	SecurityGroups []string
}

// Test subnet configuration - Defined as struct for easy user modification
var TestSubnets = map[string]SubnetConfig{
	"subnet-1": {
		SubnetID:       "subnet-04bd8bcbeb8cf7748",
		SecurityGroups: []string{"sg-xxxxxxxxx"}, // Need to modify with actual security group ID
	},
	"subnet-2": {
		SubnetID:       "subnet-08124f8bc6b14d6c9",
		SecurityGroups: []string{"sg-xxxxxxxxx"}, // Need to modify with actual security group ID
	},
}

// Define list of scenarios to execute (only scenarios in this list will be executed)
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
	// Remove or comment out scenarios that should not be executed
	// "9.2", // Expensive scenario (1024 MiB/s provisioned throughput)
}

func defineTestScenarios() []TestScenario {
	return []TestScenario{
		// 1. Basic Setup Mode
		{
			ID:          "1.1",
			Description: "Minimum Required Settings",
			Purpose:     "Test minimum required settings in basic setup mode",
			Request: irs.FileSystemInfo{
				IId:    irs.IID{NameId: "01.01-efs-basic-01"},
				VpcIID: irs.IID{SystemId: VPC_ID},
			},
			Expected: "Success - Default values applied",
		},
		{
			ID:          "1.2",
			Description: "Call without VPC",
			Purpose:     "Validate VPC requirement",
			Request: irs.FileSystemInfo{
				IId:    irs.IID{NameId: "01.02-efs-no-vpc"},
				VpcIID: irs.IID{SystemId: ""},
			},
			Expected: "Failure - VPC is required for AWS EFS file system creation",
		},
		{
			ID:          "1.3",
			Description: "Tag Processing (Name Tag not specified)",
			Purpose:     "Validate tag processing and automatic Name tag addition",
			Request: irs.FileSystemInfo{
				IId:    irs.IID{NameId: "01.03-efs-with-tags"},
				VpcIID: irs.IID{SystemId: VPC_ID},
				TagList: []irs.KeyValue{
					{Key: "Environment", Value: "Production"},
					{Key: "Project", Value: "TestProject"},
				},
			},
			Expected: "Success - User tags + Name tag automatically added",
		},
		{
			ID:          "1.4",
			Description: "When Name tag exists",
			Purpose:     "Validate user-defined Name tag priority",
			Request: irs.FileSystemInfo{
				IId:    irs.IID{NameId: "01.04-efs-name-tag-exists"},
				VpcIID: irs.IID{SystemId: VPC_ID},
				TagList: []irs.KeyValue{
					{Key: "Name", Value: "CustomName"},
					{Key: "Environment", Value: "Dev"},
				},
			},
			Expected: "Success - User Name tag used",
		},

		// 2. Advanced Setup Mode
		{
			ID:          "2.1",
			Description: "RegionType (Multi-AZ) + Basic Performance Settings",
			Purpose:     "Test basic Multi-AZ EFS creation",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "02.01-efs-region-basic"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.RegionType,
				Encryption:     true,
				NFSVersion:     "4.1",
			},
			Expected: "Success - Multi-AZ EFS created",
		},
		{
			ID:          "2.2",
			Description: "ZoneType (One Zone) + Basic Performance Settings",
			Purpose:     "Test basic One Zone EFS creation",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "02.02-efs-zone-basic"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.ZoneType,
				Zone:           ZONE_ID,
				Encryption:     true,
				NFSVersion:     "4.1",
			},
			Expected: "Success - One Zone EFS created",
		},
		{
			ID:          "2.3",
			Description: "ZoneType + Zone not specified",
			Purpose:     "Test automatic zone determination feature",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "02.03-efs-zone-auto"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.ZoneType,
				Encryption:     true,
				NFSVersion:     "4.1",
			},
			Expected: "Success - Zone automatically determined",
		},

		// 3. Performance Settings Test
		{
			ID:          "3.1",
			Description: "Elastic + GeneralPurpose (Recommended Combination)",
			Purpose:     "Test Elastic + GeneralPurpose performance combination",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "03.01-efs-elastic-gp"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.RegionType,
				PerformanceInfo: map[string]string{
					"ThroughputMode":  "Elastic",
					"PerformanceMode": "GeneralPurpose",
				},
			},
			Expected: "Success - Elastic + GeneralPurpose",
		},
		{
			ID:          "3.2",
			Description: "Bursting + MaxIO",
			Purpose:     "Test Bursting + MaxIO performance combination",
			Request: irs.FileSystemInfo{
				IId:            irs.IID{NameId: "03.02-efs-bursting-maxio"},
				VpcIID:         irs.IID{SystemId: VPC_ID},
				FileSystemType: irs.RegionType,
				PerformanceInfo: map[string]string{
					"ThroughputMode":  "Bursting",
					"PerformanceMode": "MaxIO",
				},
			},
			Expected: "Success - Bursting + MaxIO",
		},

		// 4. One Zone + MaxIO Error Test
		{
			ID:          "4.1",
			Description: "One Zone + MaxIO (Should generate error)",
			Purpose:     "Validate MaxIO limitation in One Zone",
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
			Expected: "Failure - One Zone does not support MaxIO performance mode",
		},

		// 5. Encryption Settings Test
		{
			ID:          "5.1",
			Description: "Enable Encryption",
			Purpose:     "Test encryption activation",
			Request: irs.FileSystemInfo{
				IId:        irs.IID{NameId: "05.01-efs-encrypted"},
				VpcIID:     irs.IID{SystemId: VPC_ID},
				Encryption: true,
			},
			Expected: "Success - Encrypted EFS created",
		},
		{
			ID:          "5.2",
			Description: "Disable Encryption",
			Purpose:     "Test encryption deactivation",
			Request: irs.FileSystemInfo{
				IId:        irs.IID{NameId: "05.02-efs-not-encrypted"},
				VpcIID:     irs.IID{SystemId: VPC_ID},
				Encryption: false,
			},
			Expected: "Success - Non-encrypted EFS created",
		},

		// 6. NFS Version Test
		{
			ID:          "6.1",
			Description: "NFS 4.1 Version",
			Purpose:     "Test NFS version setting",
			Request: irs.FileSystemInfo{
				IId:        irs.IID{NameId: "06.01-efs-nfs41"},
				VpcIID:     irs.IID{SystemId: VPC_ID},
				NFSVersion: "4.1",
			},
			Expected: "Success - NFS 4.1 version EFS created",
		},

		// 7. Mount Target Creation Test
		{
			ID:          "7.1",
			Description: "Using AccessSubnetList (Official Feature)",
			Purpose:     "Test mount target creation through AccessSubnetList",
			Request: irs.FileSystemInfo{
				IId:              irs.IID{NameId: "07.01-efs-access-subnets"},
				VpcIID:           irs.IID{SystemId: VPC_ID},
				FileSystemType:   irs.RegionType,
				AccessSubnetList: createAccessSubnetList("subnet-1", "subnet-2"),
			},
			Expected: "Success - 2 mount targets created, default security group used",
		},
		{
			ID:          "7.2",
			Description: "AccessSubnetList - One Zone Constraint",
			Purpose:     "Validate One Zone EFS mount target limitation",
			Request: irs.FileSystemInfo{
				IId:              irs.IID{NameId: "07.02-efs-zone-access-error"},
				VpcIID:           irs.IID{SystemId: VPC_ID},
				FileSystemType:   irs.ZoneType,
				Zone:             ZONE_ID,
				AccessSubnetList: createAccessSubnetList("subnet-1", "subnet-2"),
			},
			Expected: "Failure - One Zone EFS can only have 1 mount target, but 2 subnets were specified",
		},
		{
			ID:          "7.3",
			Description: "Using MountTargetList (Security Group Specification)",
			Purpose:     "Test security group specification through MountTargetList",
			Request: irs.FileSystemInfo{
				IId:             irs.IID{NameId: "07.03-efs-mount-targets"},
				VpcIID:          irs.IID{SystemId: VPC_ID},
				FileSystemType:  irs.RegionType,
				MountTargetList: createMountTargetList("subnet-1", "subnet-2"),
			},
			Expected: "Success - 2 mount targets created, specified security group used",
		},
		{
			ID:          "7.4",
			Description: "MountTargetList - One Zone Constraint",
			Purpose:     "Validate MountTargetList One Zone constraint",
			Request: irs.FileSystemInfo{
				IId:             irs.IID{NameId: "07.04-efs-zone-mount-error"},
				VpcIID:          irs.IID{SystemId: VPC_ID},
				FileSystemType:  irs.ZoneType,
				Zone:            ZONE_ID,
				MountTargetList: createMountTargetList("subnet-1", "subnet-2"),
			},
			Expected: "Failure - One Zone EFS can only have 1 mount target, but 2 were specified",
		},

		// 8. Complex Scenario Test
		{
			ID:          "8.1",
			Description: "Complete Advanced Settings",
			Purpose:     "Test complex advanced settings",
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
			Expected: "Success - Multi-AZ EFS + Provisioned + MaxIO + Encryption + Tags",
		},
		{
			ID:          "8.2",
			Description: "One Zone Complete Settings",
			Purpose:     "Test One Zone complex settings",
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
			Expected: "Success - One Zone EFS + Provisioned + GeneralPurpose + Encryption + Tags",
		},

		// 9. Boundary Value Test
		{
			ID:          "9.1",
			Description: "Minimum ProvisionedThroughput",
			Purpose:     "Test minimum ProvisionedThroughput boundary value",
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
			Expected: "Success - 1 MiB/s provisioned throughput",
		},
		{
			ID:          "9.2",
			Description: "Maximum ProvisionedThroughput",
			Purpose:     "Test maximum ProvisionedThroughput boundary value (expensive)",
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
			Expected: "Success - 1024 MiB/s provisioned throughput",
		},
		{
			ID:          "9.3",
			Description: "Exceed Maximum ProvisionedThroughput",
			Purpose:     "Validate exceeding maximum ProvisionedThroughput",
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
			Expected: "Failure - provisioned throughput must be between 1 and 1024 MiB/s",
		},

		// 10. Special Case Test
		{
			ID:          "10.1",
			Description: "Empty Name (Name is not required)",
			Purpose:     "Validate empty name allowance",
			Request: irs.FileSystemInfo{
				IId:    irs.IID{NameId: ""},
				VpcIID: irs.IID{SystemId: VPC_ID},
			},
			Expected: "Success - AWS EFS does not require Name",
		},
		{
			ID:          "10.2",
			Description: "Very Long Name (128 characters)",
			Purpose:     "Validate long name (128 characters) support",
			Request: irs.FileSystemInfo{
				IId:    irs.IID{NameId: createLongString(128)},
				VpcIID: irs.IID{SystemId: VPC_ID},
			},
			Expected: "Success - AWS EFS supports up to 128 character names",
		},
		{
			ID:          "10.3",
			Description: "Very Long Name (257 characters)",
			Purpose:     "Validate long name (257 characters) limitation",
			Request: irs.FileSystemInfo{
				IId:    irs.IID{NameId: createLongString(257)},
				VpcIID: irs.IID{SystemId: VPC_ID},
			},
			Expected: "Failure - AWS EFS does not support names exceeding 256 characters",
		},
	}
}

// Helper function: Create AccessSubnetList
func createAccessSubnetList(subnetKeys ...string) []irs.IID {
	var subnets []irs.IID
	for _, key := range subnetKeys {
		if config, exists := TestSubnets[key]; exists {
			subnets = append(subnets, irs.IID{SystemId: config.SubnetID})
		}
	}
	return subnets
}

// Helper function: Create MountTargetList
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

// Open HTML file in browser (considering remote environment)
func openBrowser(filename string) {
	// In remote environments, opening browser may be difficult, so suggest starting HTTP server
	cblogger.Info("=== Test Report Generated Successfully ===")
	cblogger.Infof("File: %s", filename)
	cblogger.Info("")
	cblogger.Info("To view the report in a remote environment:")
	cblogger.Info("1. Start HTTP server: python3 -m http.server 8080")
	cblogger.Info("2. Use SSH tunnel: ssh -L 8080:localhost:8080 user@remote-server")
	cblogger.Info("3. Open browser: http://localhost:8080/Test_Scenario_Result.html")
	cblogger.Info("")

	// Try to open browser only in local environment
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
	FileSystemID    string // Actual created filesystem ID
	RequestInfo     string // Request information summary
	ResponseInfo    string // Response information summary (for successful cases)
	Validation      string // Validation result
	ScenarioSuccess bool   // Scenario success status (true when execution failure is expected)
	Skipped         bool   // True when scenario is skipped (commented out, etc.)
}

// TestScenario represents a test scenario
type TestScenario struct {
	ID          string
	Description string
	Purpose     string // Test purpose added
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

// TestScenarioFileSystem function - Called from Test_Resources.go
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

		// Generate request information summary
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
			requestInfo += fmt.Sprintf(", Tags: %d", len(scenario.Request.TagList))
		}
		if len(scenario.Request.AccessSubnetList) > 0 {
			requestInfo += fmt.Sprintf(", AccessSubnets: %d", len(scenario.Request.AccessSubnetList))
		}
		testResult.RequestInfo = requestInfo

		if err != nil {
			testResult.Success = false
			testResult.Actual = "Failure"
			testResult.ErrorMessage = err.Error()
			testResult.FileSystemID = ""
			testResult.ResponseInfo = ""
			testResult.Validation = ""
			// Determine scenario success status
			if strings.HasPrefix(scenario.Expected, "Failure") {
				// Extract message after 'Failure - '
				expectedMsg := strings.TrimSpace(strings.TrimPrefix(scenario.Expected, "Failure -"))
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
				testResult.Actual = "Success"
				testResult.ErrorMessage = ""
				testResult.FileSystemID = result.IId.SystemId
				cblogger.Infof("Test Scenario %s SUCCESS: %s", scenario.ID, result.IId.SystemId)

				// Validate created EFS
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
				testResult.ScenarioSuccess = strings.HasPrefix(scenario.Expected, "Success")
			} else {
				testResult.Success = false
				testResult.Actual = "Failure"
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

	// Skip processing: Add scenarios not defined in EXECUTE_SCENARIOS as skipped
	allScenarios := defineTestScenarios()
	scenarioMap := make(map[string]TestScenario)
	for _, scenario := range allScenarios {
		scenarioMap[scenario.ID] = scenario
	}

	// Add scenarios not in EXECUTE_SCENARIOS as skipped
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

// validateFileSystemCreation function - Query created EFS and compare with request values for validation
func validateFileSystemCreation(handler irs.FileSystemHandler, request irs.FileSystemInfo, fileSystemID string) (string, string, string) {
	if fileSystemID == "" {
		return "", "", "FileSystem ID is missing"
	}

	// Query created EFS
	createdFS, err := handler.GetFileSystem(irs.IID{SystemId: fileSystemID})
	if err != nil {
		return "", "", fmt.Sprintf("EFS query failed: %v", err)
	}

	// Request information summary
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
		requestInfo += fmt.Sprintf(", Tags: %d", len(request.TagList))
	}
	if len(request.AccessSubnetList) > 0 {
		requestInfo += fmt.Sprintf(", AccessSubnets: %d", len(request.AccessSubnetList))
	}

	// Response information summary
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
		responseInfo += fmt.Sprintf(", Tags: %d", len(createdFS.TagList))
	}

	// Validation result
	validation := "✅ Validation passed"

	// Basic validation
	if request.IId.NameId != "" && createdFS.IId.NameId != request.IId.NameId {
		validation = "❌ Name mismatch"
	}
	if request.VpcIID.SystemId != "" && createdFS.VpcIID.SystemId != request.VpcIID.SystemId {
		validation = "❌ VPC mismatch"
	}
	if request.FileSystemType != "" && createdFS.FileSystemType != request.FileSystemType {
		validation = "❌ FileSystemType mismatch"
	}
	if request.Zone != "" && createdFS.Zone != request.Zone {
		validation = "❌ Zone mismatch"
	}
	if request.Encryption != createdFS.Encryption {
		validation = "❌ Encryption mismatch"
	}
	if request.NFSVersion != "" && createdFS.NFSVersion != request.NFSVersion {
		validation = "❌ NFSVersion mismatch"
	}

	// PerformanceInfo validation
	if request.PerformanceInfo != nil && createdFS.PerformanceInfo != nil {
		for key, expectedValue := range request.PerformanceInfo {
			if actualValue, exists := createdFS.PerformanceInfo[key]; !exists || actualValue != expectedValue {
				validation = fmt.Sprintf("❌ PerformanceInfo[%s] mismatch (request: %s, actual: %s)", key, expectedValue, actualValue)
				break
			}
		}
	}

	// Tag validation (Name tag is automatically added, so exclude)
	if len(request.TagList) > 0 {
		requestTagCount := 0
		for _, tag := range request.TagList {
			if tag.Key != "Name" { // Name tag is automatically added, so exclude
				requestTagCount++
			}
		}
		if requestTagCount > 0 && len(createdFS.TagList) < requestTagCount {
			validation = "❌ Tag count mismatch"
		}
	}

	return requestInfo, responseInfo, validation
}

func generateTestReport(results []TestResult) {
	// Generate report with HTML styles and JavaScript
	html := `<!DOCTYPE html>
<html lang="en">
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

	// Generate table of contents
	html += `
        <div class="toc">
            <h3>📋 Table of Contents</h3>
            <ul>
                <li><a href="#summary">📊 Overall Summary</a></li>
                <li><a href="#scenarios">📋 Scenario List</a></li>
                <li><a href="#results">📈 Overall Execution Results</a></li>
                <li><a href="#success">✅ Successful Scenarios Detail</a></li>
                <li><a href="#failure">❌ Failed Scenarios Detail</a></li>
                <li><a href="#skipped">⏭️ Skipped Scenarios</a></li>
            </ul>
        </div>`

	// Summary statistics
	successCount := 0
	failureCount := 0
	var failedScenarios []TestResult
	var successScenarios []TestResult

	for _, result := range results {
		if result.Skipped {
			continue // Exclude skipped from statistics
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
            <h2>📊 Overall Summary</h2>
            <div class="summary">
                <div class="summary-item">Total Tests: <span class="success">` + fmt.Sprintf("%d", totalCount) + `</span></div>
                <div class="summary-item">Success: <span class="success">` + fmt.Sprintf("%d", successCount) + `</span></div>
                <div class="summary-item">Failure: <span class="failure">` + fmt.Sprintf("%d", failureCount) + `</span></div>
                <div class="summary-item">Success Rate: <span class="success">` + fmt.Sprintf("%.2f%%", successRate) + `</span></div>
            </div>
        </div>`

	// Scenario list (serves as table of contents)
	html += `
        <div id="scenarios">
            <h2>📋 Scenario List</h2>
            <table>
                <tr>
                    <th>Scenario Number</th>
                    <th>Scenario Title</th>
                    <th>Expected Result</th>
                    <th>Test Purpose</th>
                </tr>`

	// Get scenario definitions again to create list
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

	// Overall execution results
	html += `
        <div id="results">
            <h2>📈 Overall Execution Results</h2>
            <table>
                <tr>
                    <th>Scenario Number</th>
                    <th>Scenario Title</th>
                    <th>Expected Result</th>
                    <th>Execution Result</th>
                    <th>Scenario Result</th>
                </tr>`

	for _, result := range results {
		// Execution result
		statusClass := "status-success"
		statusText := "✅ Success"
		if !result.Success {
			statusClass = "status-failure"
			statusText = "❌ Failure"
		}
		// Scenario result
		scenarioClass := "status-success"
		scenarioText := "✅ Success"
		if result.Skipped {
			scenarioClass = "status-warning"
			scenarioText = "⏭️ Skip"
		} else if !result.ScenarioSuccess {
			scenarioClass = "status-failure"
			scenarioText = "❌ Failure"
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

	// Successful scenarios detail
	if len(successScenarios) > 0 {
		html += `
        <div id="success">
            <h2>✅ Successful Scenarios Detail</h2>`

		for _, result := range successScenarios {
			html += fmt.Sprintf(`
            <button class="collapsible">%s - %s (Execution Time: %s)</button>
            <div class="content">
                <h4>📋 Request Information</h4>
                <p><strong>%s</strong></p>
                
                <h4>📤 Response Information</h4>
                <p><strong>%s</strong></p>
                
                <h4>🔍 Validation Result</h4>
                <p><strong>%s</strong></p>
                
                <h4>📝 Detailed Log</h4>
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

	// Failed scenarios detail
	if len(failedScenarios) > 0 {
		html += `
        <div id="failure">
            <h2>❌ Failed Scenarios Detail</h2>`

		for _, result := range failedScenarios {
			html += fmt.Sprintf(`
            <button class="collapsible">%s - %s (Execution Time: %s)</button>
            <div class="content">
                <h4>📋 Request Information</h4>
                <p><strong>%s</strong></p>
                
                <h4>❌ Error Message</h4>
                <p><strong>%s</strong></p>
                
                <h4>📝 Detailed Log</h4>
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

	// Skipped scenarios detail
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
            <h2>⏭️ Skipped Scenarios Detail</h2>`
			for _, result := range skipScenarios {
				html += fmt.Sprintf(`
            <button class="collapsible">%s - %s</button>
            <div class="content">
                <h4>📋 Scenario Information</h4>
                <p><strong>Scenario ID:</strong> %s</p>
                <p><strong>Description:</strong> %s</p>
                <p><strong>Expected Result:</strong> %s</p>
                
                <h4>⏭️ Skip Reason</h4>
                <p>This scenario was not executed due to cost, time, or other reasons.</p>
                <p>To execute, add "%s" to the <code>EXECUTE_SCENARIOS</code> list.</p>
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

		// Automatically open in browser
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
	// Use CBSPIDER_TEST_CONF_PATH environment variable
	confPath := os.Getenv("CBSPIDER_TEST_CONF_PATH")
	if confPath == "" {
		panic("CBSPIDER_TEST_CONF_PATH environment variable is not set")
	}
	cblogger.Infof("Configuration file path: [%s]", confPath)

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
