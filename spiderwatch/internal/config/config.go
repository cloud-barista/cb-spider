// Package config handles loading and hot-reloading of SpiderWatch configuration.
package config

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// AllKnownResources lists every resource type SpiderWatch can test, in display order.
var AllKnownResources = []string{
	"vpc", "securitygroup", "keypair", "vm", "disk", "nlb", "myimage", "cluster", "s3",
}

// Config is the root configuration structure.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Spider    SpiderConfig    `yaml:"spider"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	// Cleanup controls resource deletion after each test run.
	// "true"  – run all tests then delete created resources
	// "false" – run all tests but leave resources in the CSP
	// "only"  – skip all tests; only delete previously left resources
	Cleanup     string            `yaml:"cleanup"`
	CSPs        []CSPConfig       `yaml:"csps"`
	Resources   []string          `yaml:"resources"`
	Log         LogConfig         `yaml:"log"`
	GitHub      GitHubConfig      `yaml:"github"`
	StatusBoard StatusBoardConfig `yaml:"statusboard"`
}

// StatusBoardConfig holds settings for pushing results to the public Status Board.
type StatusBoardConfig struct {
	URL   string `yaml:"url"`   // e.g. "https://spider-status.example.com"
	Token string `yaml:"token"` // shared secret (must match statusboard.yaml auth.token)
}

// GitHubConfig holds settings for filing GitHub issues from SpiderWatch.
type GitHubConfig struct {
	Token  string   `yaml:"token"`
	Owner  string   `yaml:"owner"`
	Repo   string   `yaml:"repo"`
	Labels []string `yaml:"labels"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port    int    `yaml:"port"`
	Address string `yaml:"address"`
}

// SpiderConfig holds Spider Docker and API settings.
type SpiderConfig struct {
	Image             string `yaml:"image"`
	HostPort          int    `yaml:"host_port"`
	APIURL            string `yaml:"api_url"`
	Username          string `yaml:"username"`
	Password          string `yaml:"password"`
	MetaDBDir         string `yaml:"meta_db_dir"`
	MCInsightAPIToken string `yaml:"mc_insight_api_token"`
	ServerAddress     string `yaml:"server_address"`
	APITimeoutSec     int    `yaml:"api_timeout_sec"`
	StartupWaitSec    int    `yaml:"startup_wait_sec"`
	RunTimeoutMin     int    `yaml:"run_timeout_min"`
	// ExternalURL, when non-empty, instructs SpiderWatch to use an already-running
	// Spider server at this URL instead of starting/stopping a Docker container.
	// The Docker image, host_port, meta_db_dir, etc. settings are ignored.
	// The Spider server will NOT be stopped after a test run or cleanup-only run.
	ExternalURL string `yaml:"external_url"`
}

// SchedulerConfig holds cron and startup settings.
type SchedulerConfig struct {
	Cron         string `yaml:"cron"`
	RunOnStartup bool   `yaml:"run_on_startup"`
}

// CSPConfig represents a single CSP to test.
type CSPConfig struct {
	Name        string            `yaml:"name"`
	Connection  string            `yaml:"connection"`
	Enabled     bool              `yaml:"enabled"`
	VPCTest     VPCTestConfig     `yaml:"vpc_test"`
	VMTest      VMTestConfig      `yaml:"vm_test"`
	DiskTest    DiskTestConfig    `yaml:"disk_test"`
	NLBTest     NLBTestConfig     `yaml:"nlb_test"`
	ClusterTest ClusterTestConfig `yaml:"cluster_test"`
	// SGExtraInboundPorts lists additional TCP ports to open on the test security group.
	// Useful for CSPs that require extra ports (e.g. Alibaba ACK needs port 6443
	// for the Kubernetes API server).
	SGExtraInboundPorts []string `yaml:"sg_extra_inbound_ports"`
}

// VPCTestConfig holds per-CSP VPC and subnet CIDR settings.
type VPCTestConfig struct {
	VPCCIDR    string `yaml:"vpc_cidr"`    // e.g. "192.168.0.0/16"
	SubnetCIDR string `yaml:"subnet_cidr"` // e.g. "192.168.1.0/24"
}

// ClusterTestConfig holds per-CSP settings for Cluster CRUD tests.
type ClusterTestConfig struct {
	Version string `yaml:"version"` // Kubernetes version, e.g. "1.30"

	// KubeconfigType controls how kubectl authenticates to the cluster.
	//   "static"         – credentials embedded in kubeconfig (Azure, Alibaba, Tencent, IBM, NHN, …)
	//   "spider_default" – exec-plugin that calls Spider Token API (AWS, GCP, NCP)
	//   "csp_native"     – exec-plugin using CSP tool (aws-iam-authenticator, gke-gcloud-auth-plugin)
	// Defaults to "static" when empty.
	KubeconfigType string `yaml:"kubeconfig_type"`

	// NodeGroupType controls how the initial NodeGroup is handled.
	//   "type1" – NodeGroup is added separately after the cluster is Active
	//              (Type-I CSPs: AWS, ALIBABA, TENCENT)
	//   "type2" – NodeGroup is included in the initial cluster create request
	//              (Type-II CSPs: AZURE, GCP, NHN, NCP, IBM, …)
	// Defaults to "type1" when empty.
	NodeGroupType string `yaml:"node_group_type"`

	// NodeGroup resource settings (shared for both create-time and add-time).
	NodeGroupImageName       string `yaml:"node_group_image_name"`
	NodeGroupVMSpecName      string `yaml:"node_group_vm_spec_name"`
	NodeGroupRootDiskType    string `yaml:"node_group_root_disk_type"`
	NodeGroupRootDiskSize    string `yaml:"node_group_root_disk_size"`
	NodeGroupOnAutoScaling   string `yaml:"node_group_on_auto_scaling"`
	NodeGroupDesiredNodeSize string `yaml:"node_group_desired_node_size"`
	NodeGroupMinNodeSize     string `yaml:"node_group_min_node_size"`
	NodeGroupMaxNodeSize     string `yaml:"node_group_max_node_size"`

	// ExtraSubnetCIDR/ExtraSubnetZone lets you add a second subnet in a different
	// AZ before creating the cluster. Required for CSPs like AWS EKS that mandate
	// subnets in at least two availability zones.
	ExtraSubnetCIDR string `yaml:"extra_subnet_cidr"`
	ExtraSubnetZone string `yaml:"extra_subnet_zone"`
}

// NLBTestConfig holds per-CSP settings for NLB CRUD tests.
type NLBTestConfig struct {
	Type             string `yaml:"type"`              // PUBLIC / PRIVATE
	Scope            string `yaml:"scope"`             // REGION / GLOBAL
	ListenerProtocol string `yaml:"listener_protocol"` // TCP / HTTP / HTTPS
	ListenerPort     string `yaml:"listener_port"`     // e.g. "80"
	TargetProtocol   string `yaml:"target_protocol"`   // TCP / HTTP / HTTPS
	TargetPort       string `yaml:"target_port"`       // e.g. "80"
	HealthProtocol   string `yaml:"health_protocol"`   // TCP / HTTP / HTTPS
	HealthPort       string `yaml:"health_port"`       // e.g. "22"
	HealthInterval   string `yaml:"health_interval"`   // seconds or "default"
	HealthTimeout    string `yaml:"health_timeout"`    // seconds or "default"
	HealthThreshold  string `yaml:"health_threshold"`  // count or "default"
	// NginxPreInstall installs nginx on the test VM via SSH before the NLB test.
	// Required for CSPs (e.g. GCP) whose NLB health-checker uses HTTP and needs
	// port 80 to be served on the VM.
	NginxPreInstall bool `yaml:"nginx_pre_install"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// VMTestConfig holds fixed image/spec names used for VM CRUD tests.
type VMTestConfig struct {
	ImageName string `yaml:"image_name"`
	SpecName  string `yaml:"spec_name"`
}

// DiskTestConfig holds settings used for Disk CRUD tests.
type DiskTestConfig struct {
	DiskType string `yaml:"disk_type"`
	DiskSize string `yaml:"disk_size"`
}

var (
	mu      sync.RWMutex
	fileMu  sync.Mutex // serialises concurrent YAML read-modify-write operations
	current *Config
	logger  = logrus.New()
)

// Load reads the YAML config file and starts a file watcher for hot-reload.
func Load(path string, onChange func(*Config)) (*Config, error) {
	cfg, err := readFile(path)
	if err != nil {
		return nil, err
	}
	applyDefaults(cfg)

	mu.Lock()
	current = cfg
	mu.Unlock()

	go watch(path, onChange)
	return cfg, nil
}

// Get returns the current configuration (thread-safe).
func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

func readFile(path string) (*Config, error) {
	expanded := os.ExpandEnv(path)
	data, err := os.ReadFile(expanded)
	if err != nil {
		return nil, fmt.Errorf("config: read file %q: %w", expanded, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse yaml %q: %w", expanded, err)
	}
	// Expand environment variables in string fields
	cfg.Spider.MetaDBDir = os.ExpandEnv(cfg.Spider.MetaDBDir)
	cfg.Spider.APIURL = os.ExpandEnv(cfg.Spider.APIURL)
	cfg.Spider.ServerAddress = os.ExpandEnv(cfg.Spider.ServerAddress)
	cfg.Log.File = os.ExpandEnv(cfg.Log.File)
	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 2048
	}
	if cfg.Spider.HostPort == 0 {
		cfg.Spider.HostPort = 1024
	}
	if cfg.Spider.APITimeoutSec == 0 {
		cfg.Spider.APITimeoutSec = 60
	}
	if cfg.Spider.StartupWaitSec == 0 {
		cfg.Spider.StartupWaitSec = 30
	}
	if cfg.Spider.RunTimeoutMin == 0 {
		cfg.Spider.RunTimeoutMin = 120
	}
	if cfg.Scheduler.Cron == "" {
		cfg.Scheduler.Cron = "0 0 1 * * *"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
}

func watch(path string, onChange func(*Config)) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.WithError(err).Error("config: failed to create file watcher")
		return
	}
	defer watcher.Close()

	if err := watcher.Add(path); err != nil {
		logger.WithError(err).Errorf("config: failed to watch %q", path)
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Handle atomic save editors (vim, nano): re-add watch after REMOVE/RENAME
			if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
				_ = watcher.Add(path)
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				cfg, err := readFile(path)
				if err != nil {
					logger.WithError(err).Warning("config: hot-reload failed, keeping previous config")
					continue
				}
				applyDefaults(cfg)
				mu.Lock()
				current = cfg
				mu.Unlock()
				logger.Info("config: hot-reloaded from ", path)
				if onChange != nil {
					onChange(cfg)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.WithError(err).Error("config: watcher error")
		}
	}
}

// UpdateResources rewrites the resources: block in the YAML config file so that
// only the supplied resource kinds are enabled; all other known kinds are written
// as commented-out entries. The file watcher picks up the change and hot-reloads
// the configuration automatically.
func UpdateResources(path string, resources []string) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	enabled := make(map[string]bool, len(resources))
	for _, r := range resources {
		enabled[r] = true
	}

	expanded := os.ExpandEnv(path)
	data, err := os.ReadFile(expanded)
	if err != nil {
		return fmt.Errorf("config: read %q: %w", expanded, err)
	}

	lines := strings.Split(string(data), "\n")

	// Locate the resources: block. It starts at the line "resources:" and ends
	// at the first subsequent non-blank line that is NOT an indented YAML list
	// item and NOT a commented-out list item (e.g. "#  - disk").
	startIdx := -1
	endIdx := len(lines)
	for i, line := range lines {
		if startIdx == -1 {
			if line == "resources:" || strings.HasPrefix(line, "resources:") {
				startIdx = i
			}
			continue
		}
		// Blank / whitespace-only → still part of the block (trailing separator).
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Indented line → YAML list item, still in block.
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		// "# - kind" style commented-out item → still in block.
		stripped := strings.TrimLeft(line, "#")
		if strings.HasPrefix(strings.TrimSpace(stripped), "-") {
			continue
		}
		// Anything else is the next top-level section.
		endIdx = i
		break
	}
	if startIdx == -1 {
		return fmt.Errorf("config: resources: block not found in %q", expanded)
	}

	// Build the replacement block.
	var newLines []string
	newLines = append(newLines, "resources:")
	for _, kind := range AllKnownResources {
		if enabled[kind] {
			newLines = append(newLines, "  - "+kind)
		} else {
			newLines = append(newLines, "#  - "+kind)
		}
	}
	// Preserve trailing blank line that acted as section separator, if present.
	if endIdx > startIdx+1 && strings.TrimSpace(lines[endIdx-1]) == "" {
		newLines = append(newLines, "")
	}

	var result []string
	result = append(result, lines[:startIdx]...)
	result = append(result, newLines...)
	result = append(result, lines[endIdx:]...)

	return os.WriteFile(expanded, []byte(strings.Join(result, "\n")), 0o644)
}

// UpdateCSPsEnabled sets the enabled: field for each CSP in the YAML config
// file. CSPs whose names appear in enabledNames get enabled: true; all others
// get enabled: false. The file watcher picks up the write and hot-reloads the
// configuration automatically.
func UpdateCSPsEnabled(path string, enabledNames []string) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	enabled := make(map[string]bool, len(enabledNames))
	for _, n := range enabledNames {
		enabled[n] = true
	}

	expanded := os.ExpandEnv(path)
	data, err := os.ReadFile(expanded)
	if err != nil {
		return fmt.Errorf("config: read %q: %w", expanded, err)
	}

	lines := strings.Split(string(data), "\n")

	inCSPs := false
	currentCSP := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inCSPs {
			if trimmed == "csps:" {
				inCSPs = true
			}
			continue
		}

		// Exit the csps: block when we reach a non-blank top-level key.
		if trimmed != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break
		}

		// Detect new CSP list entry: "  - name: <NAME>"
		if strings.HasPrefix(line, "  - name:") {
			currentCSP = strings.TrimSpace(strings.TrimPrefix(trimmed, "- name:"))
			continue
		}

		// Rewrite the enabled: line for the current CSP (4-space indent).
		if currentCSP != "" && strings.HasPrefix(line, "    enabled:") {
			val := "false"
			if enabled[currentCSP] {
				val = "true"
			}
			lines[i] = "    enabled: " + val
		}
	}

	return os.WriteFile(expanded, []byte(strings.Join(lines, "\n")), 0o644)
}

// UpdateCleanup rewrites the cleanup: line in the YAML config file.
// value must be "true" or "false"; "only" is reserved for runtime use only.
func UpdateCleanup(path, value string) error {
	if value != "true" && value != "false" {
		return fmt.Errorf("config: cleanup value must be \"true\" or \"false\", got %q", value)
	}
	fileMu.Lock()
	defer fileMu.Unlock()

	expanded := os.ExpandEnv(path)
	data, err := os.ReadFile(expanded)
	if err != nil {
		return fmt.Errorf("config: read %q: %w", expanded, err)
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "cleanup:") {
			lines[i] = "cleanup: " + value
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("config: cleanup: line not found in %q", expanded)
	}
	return os.WriteFile(expanded, []byte(strings.Join(lines, "\n")), 0o644)
}
