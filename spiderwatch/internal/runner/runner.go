// Package runner manages the cb-spider Docker container lifecycle
// and calls Spider REST APIs to collect resource status for each CSP.
package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/cloud-barista/cb-spider/spiderwatch/internal/config"
	"github.com/cloud-barista/cb-spider/spiderwatch/internal/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

var log = logrus.New()

// nginxDeployYAML is the Kubernetes manifest applied during the cluster nginx test.
const nginxDeployYAML = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:latest
          ports:
            - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
spec:
  selector:
    app: nginx
  ports:
    - protocol: TCP
      port: 80
      targetPort: 80
  type: LoadBalancer
`

// Runner orchestrates Spider Docker container and API calls.
type Runner struct {
	mu         sync.Mutex
	running    bool
	cancelFunc context.CancelFunc // non-nil while a run is in progress
	stopped    bool               // true if Stop() was called
}

// New returns a new Runner.
func New() *Runner {
	return &Runner{}
}

// IsRunning reports whether a test run is in progress.
func (r *Runner) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// Stop cancels the in-progress test run so results gathered so far can be
// persisted and reported. Returns true if a run was actually cancelled.
func (r *Runner) Stop() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running || r.cancelFunc == nil {
		return false
	}
	r.stopped = true
	r.cancelFunc()
	return true
}

// IsSpiderRunning reports whether the Spider server is currently running.
// When external_url is configured, it pings the /readyz endpoint instead of
// checking the Docker container.
func (r *Runner) IsSpiderRunning() bool {
	cfg := config.Get()
	if cfg.Spider.ExternalURL != "" {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get(strings.TrimRight(cfg.Spider.ExternalURL, "/") + "/readyz")
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode < 400
	}
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", "cb-spider-watch-tmp").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// Run starts a full test run. It is safe to call this from a goroutine.
// The returned RunResult is updated in place and also passed to onUpdate after each CSP finishes.
func (r *Runner) Run(cfg *config.Config, onUpdate func(*model.RunResult)) (*model.RunResult, error) {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return nil, fmt.Errorf("runner: a test run is already in progress")
	}
	r.running = true
	r.stopped = false
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.running = false
		r.cancelFunc = nil
		r.mu.Unlock()
	}()

	now := time.Now()
	runID := now.Format("2006-01-02T15-04-05")
	result := &model.RunResult{
		ID:          runID,
		StartedAt:   now,
		Status:      model.RunStatusRunning,
		CleanupOnly: strings.ToLower(strings.TrimSpace(cfg.Cleanup)) == "only",
	}
	log.Infof("runner: starting run %s", runID)

	externalMode := cfg.Spider.ExternalURL != ""

	// Overall timeout for the entire run.
	// baseCtx is cancelled by Stop(); the child ctx adds the wall-clock timeout.
	baseCtx, baseCancel := context.WithCancel(context.Background())
	defer baseCancel()
	ctx, cancel := context.WithTimeout(baseCtx, time.Duration(cfg.Spider.RunTimeoutMin)*time.Minute)
	defer cancel()
	r.mu.Lock()
	r.cancelFunc = baseCancel
	r.mu.Unlock()

	if externalMode {
		// External Spider mode: skip Docker entirely, just verify the server is reachable.
		log.Infof("runner: external Spider mode — using %s (no Docker container management)", cfg.Spider.ExternalURL)
		if err := r.waitReady(ctx, cfg); err != nil {
			result.Status = model.RunStatusFailed
			result.Error = err.Error()
			fin := time.Now()
			result.FinishedAt = &fin
			return result, err
		}
		result.SpiderImage = "external: " + cfg.Spider.ExternalURL
	} else {
		// Docker mode: preflight check, pull image, run container.
		if err := checkDocker(); err != nil {
			result.Status = model.RunStatusFailed
			result.Error = err.Error()
			fin := time.Now()
			result.FinishedAt = &fin
			log.WithError(err).Error("runner: docker preflight check failed")
			return result, err
		}
		if err := r.startContainer(ctx, cfg); err != nil {
			result.Status = model.RunStatusFailed
			result.Error = err.Error()
			fin := time.Now()
			result.FinishedAt = &fin
			log.WithError(err).Error("runner: failed to start spider container")
			return result, err
		}
		result.SpiderImage = spiderImageInfo
		if err := r.waitReady(ctx, cfg); err != nil {
			_ = stopContainer(cfg)
			result.Status = model.RunStatusFailed
			fin := time.Now()
			result.FinishedAt = &fin
			return result, err
		}
	}

	// Pre-allocate result slots for all enabled CSPs so partial results
	// are visible immediately when goroutines start.
	type cspEntry struct {
		cfg config.CSPConfig
		idx int
	}
	var enabledCSPs []cspEntry
	for _, cspCfg := range cfg.CSPs {
		if !cspCfg.Enabled {
			continue
		}
		enabledCSPs = append(enabledCSPs, cspEntry{cfg: cspCfg, idx: len(result.CSPs)})
		result.CSPs = append(result.CSPs, model.CSPResult{
			Name:       cspCfg.Name,
			Connection: cspCfg.Connection,
		})
	}
	result.Progresses = make(map[string]*model.RunProgress, len(enabledCSPs))
	if onUpdate != nil {
		onUpdate(result)
	}

	// 4-digit sequence derived from run timestamp; cycles every ~2.78 h.
	// Ensures S3 bucket names differ between consecutive runs so CSPs that
	// delay bucket-name reuse (e.g. AWS) don't cause spurious create errors.
	s3Seq := uint16(now.Unix() % 10000)

	// Test all enabled CSPs concurrently.
	var resultMu sync.Mutex
	var wg sync.WaitGroup
	for _, e := range enabledCSPs {
		e := e // capture
		wg.Add(1)
		go func() {
			defer wg.Done()
			notify := func(partial model.CSPResult, resource, op string, done []model.OperationResult) {
				resultMu.Lock()
				result.CSPs[e.idx] = partial
				if resource != "" {
					result.Progresses[e.cfg.Name] = &model.RunProgress{CSP: e.cfg.Name, Resource: resource, Operation: op, CompletedOps: done}
				} else {
					delete(result.Progresses, e.cfg.Name)
				}
				if onUpdate != nil {
					onUpdate(result)
				}
				resultMu.Unlock()
			}
			cspResult := r.testCSP(ctx, cfg, e.cfg, s3Seq, notify)
			resultMu.Lock()
			result.CSPs[e.idx] = cspResult
			delete(result.Progresses, e.cfg.Name)
			if onUpdate != nil {
				onUpdate(result)
			}
			resultMu.Unlock()
		}()
	}
	wg.Wait()

	if !externalMode {
		_ = stopContainer(cfg)
	}
	r.mu.Lock()
	wasStopped := r.stopped
	r.mu.Unlock()
	if wasStopped {
		result.Status = model.RunStatusStopped
		log.Infof("runner: run %s stopped by user", runID)
	} else {
		result.Status = model.RunStatusDone
		log.Infof("runner: run %s finished", runID)
	}
	fin := time.Now()
	result.FinishedAt = &fin
	return result, nil
}

// checkDocker verifies the Docker daemon is reachable by running 'docker info'.
func checkDocker() error {
	out, err := exec.Command("docker", "info", "--format", "{{.ServerVersion}}").CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker daemon is not reachable: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// startContainer pulls the Spider image and runs a detached container.
// spiderImageInfo holds the resolved image reference after pull (image:tag@sha256:... created=...).
var spiderImageInfo string

func (r *Runner) startContainer(ctx context.Context, cfg *config.Config) error {
	log.Infof("runner: pulling image %s", cfg.Spider.Image)
	pullArgs := []string{"pull", cfg.Spider.Image}
	if out, err := exec.CommandContext(ctx, "docker", pullArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("runner: docker pull: %w - %s", err, strings.TrimSpace(string(out)))
	}

	// Capture the image digest and creation timestamp for traceability.
	inspectOut, inspectErr := exec.CommandContext(ctx, "docker", "inspect",
		"--format", "{{index .RepoDigests 0}} created={{.Created}}",
		cfg.Spider.Image).Output()
	if inspectErr == nil {
		raw := strings.TrimSpace(string(inspectOut))
		if idx := strings.Index(raw, " created="); idx >= 0 {
			digest := raw[:idx]
			createdRaw := raw[idx+len(" created="):]
			if t, err := time.Parse(time.RFC3339Nano, createdRaw); err == nil {
				kst := time.FixedZone("KST", 9*3600)
				timeStr := t.In(kst).Format("Jan 2, 2006 at 3:04 PM")
				timeStr = strings.Replace(timeStr, " AM", " am", 1)
				timeStr = strings.Replace(timeStr, " PM", " pm", 1)
				spiderImageInfo = digest + " created=" + timeStr
			} else {
				spiderImageInfo = raw
			}
		} else {
			spiderImageInfo = raw
		}
		log.Infof("runner: spider image info: %s", spiderImageInfo)
	} else {
		spiderImageInfo = cfg.Spider.Image
	}

	// Remove leftover container if exists
	_ = exec.Command("docker", "rm", "-f", "cb-spider-watch-tmp").Run()
	log.Infof("runner: starting spider container on port %d", cfg.Spider.HostPort)
	runArgs := []string{
		"run", "--rm", "-d",
		"-p", fmt.Sprintf("%d:1024", cfg.Spider.HostPort),
		"-v", fmt.Sprintf("%s:/root/go/src/github.com/cloud-barista/cb-spider/meta_db", cfg.Spider.MetaDBDir),
		"-e", fmt.Sprintf("SERVER_ADDRESS=%s", cfg.Spider.ServerAddress),
		"-e", fmt.Sprintf("SPIDER_USERNAME=%s", cfg.Spider.Username),
		"-e", fmt.Sprintf("SPIDER_PASSWORD=%s", cfg.Spider.Password),
		"-e", fmt.Sprintf("MC_INSIGHT_API_TOKEN=%s", cfg.Spider.MCInsightAPIToken),
		"--name", "cb-spider-watch-tmp",
		cfg.Spider.Image,
	}
	if out, err := exec.CommandContext(ctx, "docker", runArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("runner: docker run: %w - %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// waitReady polls the Spider /readyz endpoint until it responds.
func (r *Runner) waitReady(ctx context.Context, cfg *config.Config) error {
	client := &http.Client{Timeout: 5 * time.Second}
	apiURL := cfg.Spider.APIURL
	if cfg.Spider.ExternalURL != "" {
		apiURL = cfg.Spider.ExternalURL
	}
	waitSec := cfg.Spider.StartupWaitSec
	if cfg.Spider.ExternalURL != "" && waitSec > 10 {
		waitSec = 10 // External server should already be running; short wait only.
	}
	deadline := time.Now().Add(time.Duration(waitSec) * time.Second)
	waitURL := apiURL + "/readyz"
	log.Infof("runner: waiting up to %ds for spider to be ready at %s", waitSec, waitURL)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return fmt.Errorf("runner: context cancelled while waiting for spider: %w", ctx.Err())
		default:
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, waitURL, nil)
		req.SetBasicAuth(cfg.Spider.Username, cfg.Spider.Password)
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode < 500 {
				log.Info("runner: spider is ready")
				return nil
			}
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("runner: spider did not become ready within %ds", cfg.Spider.StartupWaitSec)
}

// stopContainer stops and removes the Spider container.
func stopContainer(cfg *config.Config) error {
	log.Info("runner: stopping spider container")
	out, err := exec.Command("docker", "rm", "-f", "cb-spider-watch-tmp").CombinedOutput()
	if err != nil {
		log.Warnf("runner: docker rm: %s", strings.TrimSpace(string(out)))
	}
	return err
}

// StartSpider starts the Spider Docker container and waits until it is ready.
// In external Spider mode, it only verifies the server is reachable.
func (r *Runner) StartSpider(cfg *config.Config) error {
	if cfg.Spider.ExternalURL != "" {
		log.Infof("runner: external Spider mode — verifying %s is reachable", cfg.Spider.ExternalURL)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return r.waitReady(ctx, cfg)
	}
	if err := checkDocker(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Spider.StartupWaitSec+30)*time.Second)
	defer cancel()
	if err := r.startContainer(ctx, cfg); err != nil {
		return err
	}
	return r.waitReady(ctx, cfg)
}

// StopSpider stops and removes the Spider Docker container.
// In external Spider mode, this is a no-op — the external server is not managed by SpiderWatch.
func (r *Runner) StopSpider(cfg *config.Config) error {
	if cfg.Spider.ExternalURL != "" {
		log.Info("runner: external Spider mode — skipping stop (server not managed by SpiderWatch)")
		return nil
	}
	return stopContainer(cfg)
}

// cspTestState tracks resources created during a testCSP run so they can be
// shared across individual resource tests and cleaned up in a single final step.
type cspTestState struct {
	prefix      string // common name prefix: "spider-watch"
	vpcName     string
	subnetName  string
	sgName      string
	kpName      string
	vmName      string
	nlbName     string
	diskName    string
	myImgName   string
	clusterName string
	s3Name      string

	// kpPrivateKey holds the PEM private key returned at keypair creation time.
	// Spider does not expose it via LIST or GET, so it must be captured here.
	kpPrivateKey string

	// vmPublicIP and vmSshLoginOK are set by testVMCRUD after the VM is up.
	// vmSshLoginOK is true only if ssh-login succeeded; it gates nginx pre-install.
	vmPublicIP   string
	vmSshLoginOK bool

	// nlbTest holds the per-CSP NLB configuration; needed inside testVMCRUD
	// to decide whether to run the ssh-nginx-install step.
	nlbTest config.NLBTestConfig

	// Per-CSP test settings copied from config at run time.
	vmTest      config.VMTestConfig
	diskTest    config.DiskTestConfig
	clusterTest config.ClusterTestConfig

	// sgExtraInboundPorts lists additional TCP inbound ports to open on the test SG.
	sgExtraInboundPorts []string

	vpcCreated         bool
	sgCreated          bool
	kpCreated          bool
	nlbCreated         bool
	vmCreated          bool
	diskCreated        bool
	myImgCreated       bool
	clusterCreated     bool
	extraSubnetCreated bool
	s3Created          bool

	// nginxDeployed is true once kubectl apply for the nginx deployment has succeeded.
	// kubeconfigPath is the path to the temp kubeconfig file (empty until written).
	nginxDeployed  bool
	kubeconfigPath string

	// notifyOp is called by st.op() before and after each operation to update live progress.
	// Set by testCSP for each resource and cleared when the resource finishes.
	notifyOp func(opName string, done []model.OperationResult)
	// liveOps accumulates completed operations for the current resource in real time.
	liveOps []model.OperationResult
}

// op wraps the package-level runOp, calling st.notifyOp before and after so the
// live board can show which operation is running and which have already completed.
func (st *cspTestState) op(name string, fn func() error) model.OperationResult {
	if st.notifyOp != nil {
		st.notifyOp(name, st.liveOps) // before: signal this op is starting
	}
	res := runOp(name, fn)
	st.liveOps = append(st.liveOps, res)
	if st.notifyOp != nil {
		st.notifyOp("", st.liveOps) // after: refresh with the now-completed op
	}
	return res
}

// buildSGRules returns the base inbound SG rules plus any extra ports from st.sgExtraInboundPorts.
func (st *cspTestState) buildSGRules() []sgRule {
	rules := []sgRule{
		{FromPort: "22", ToPort: "22", IPProtocol: "tcp", Direction: "inbound", CIDR: "0.0.0.0/0"},
		{FromPort: "80", ToPort: "80", IPProtocol: "tcp", Direction: "inbound", CIDR: "0.0.0.0/0"},
		{FromPort: "443", ToPort: "443", IPProtocol: "tcp", Direction: "inbound", CIDR: "0.0.0.0/0"},
		{FromPort: "30000", ToPort: "32767", IPProtocol: "tcp", Direction: "inbound", CIDR: "0.0.0.0/0"},
	}
	for _, p := range st.sgExtraInboundPorts {
		rules = append(rules, sgRule{FromPort: p, ToPort: p, IPProtocol: "tcp", Direction: "inbound", CIDR: "0.0.0.0/0"})
	}
	return rules
}

// testCSP calls CRUD or list APIs for each resource type and builds a CSPResult.
// notify (may be nil) is called before each resource test starts (with the resource
// name) and after it finishes (with empty string), enabling live board updates.
func (r *Runner) testCSP(ctx context.Context, cfg *config.Config, cspCfg config.CSPConfig, s3Seq uint16, notify func(model.CSPResult, string, string, []model.OperationResult)) model.CSPResult {
	log.Infof("runner: testing CSP=%s connection=%s", cspCfg.Name, cspCfg.Connection)
	// Pre-calculate the expected total so the dashboard shows a fixed number
	// while the test is in progress, rather than incrementing dynamically.
	expectedTotal := len(cfg.Resources)
	cleanupModeForCount := strings.ToLower(strings.TrimSpace(cfg.Cleanup))
	if cleanupModeForCount == "true" || cleanupModeForCount == "only" {
		expectedTotal++
	}
	result := model.CSPResult{
		Name:          cspCfg.Name,
		Connection:    cspCfg.Connection,
		ExpectedTotal: expectedTotal,
	}
	client := &http.Client{Timeout: time.Duration(cfg.Spider.APITimeoutSec) * time.Second}

	// Shared state: all CRUD tests within one CSP run share a single name prefix
	// and accumulate which resources were created.  The final "cleanup" entry
	// deletes everything in reverse order.
	// Fixed resource names so cleanup can find them across runs.
	st := &cspTestState{prefix: "spider-watch"}
	st.vpcName = "spider-watch"
	st.subnetName = "spider-watch"
	st.sgName = "spider-watch"
	st.kpName = "spider-watch"
	st.vmName = "spider-watch"
	st.diskName = "spider-watch"
	st.myImgName = "spider-watch"
	st.nlbName = "spider-watch"
	st.clusterName = "spider-watch"
	st.s3Name = s3BucketName(cspCfg.Connection, s3Seq)
	st.vmTest = cspCfg.VMTest
	st.diskTest = cspCfg.DiskTest
	st.clusterTest = cspCfg.ClusterTest
	st.nlbTest = cspCfg.NLBTest
	st.sgExtraInboundPorts = cspCfg.SGExtraInboundPorts

	cleanupMode := strings.ToLower(strings.TrimSpace(cfg.Cleanup))

	// "only" mode: skip all resource tests, jump straight to cleanup.
	if cleanupMode != "only" {
		for _, resource := range cfg.Resources {
			res := resource // capture loop var for closure
			st.liveOps = nil
			st.notifyOp = func(opName string, done []model.OperationResult) {
				if notify != nil {
					notify(result, res, opName, done)
				}
			}
			if notify != nil {
				notify(result, resource, "", nil) // signal: this resource is now being tested
			}
			var rr model.ResourceResult
			switch resource {
			case "vpc":
				rr = r.testVPCCRUD(ctx, client, cfg, cspCfg.Connection, cspCfg.VPCTest, st)
			case "securitygroup":
				rr = r.testSecurityGroupCRUD(ctx, client, cfg, cspCfg.Connection, cspCfg.VPCTest, st)
			case "keypair":
				rr = r.testKeyPairCRUD(ctx, client, cfg, cspCfg.Connection, st)
			case "vm":
				rr = r.testVMCRUD(ctx, client, cfg, cspCfg.Connection, cspCfg.VPCTest, st)
			case "disk":
				rr = r.testDiskCRUD(ctx, client, cfg, cspCfg.Connection, st)
			case "nlb":
				rr = r.testNLBCRUD(ctx, client, cfg, cspCfg.Connection, cspCfg.NLBTest, cspCfg.VPCTest, st)
			case "myimage":
				rr = r.testMyImageCRUD(ctx, client, cfg, cspCfg.Connection, cspCfg.VPCTest, st)
			case "cluster":
				rr = r.testClusterCRUD(ctx, client, cfg, cspCfg.Connection, cspCfg.VPCTest, st)
			case "s3":
				rr = r.testS3CRUD(ctx, client, cfg, cspCfg.Connection, st)
			default:
				rr = callListAPI(ctx, client, cfg, cspCfg.Connection, resource)
			}
			result.Resources = append(result.Resources, rr)
			log.Infof("runner: CSP=%s resource=%s status=%s count=%d", cspCfg.Name, resource, rr.Status, rr.Count)
			st.notifyOp = nil
			st.liveOps = nil
			if notify != nil {
				notify(result, "", "", nil) // signal: resource done, board now shows its result
			}
			// Stop was pressed: cancel remaining resources in this CSP
			if ctx.Err() != nil {
				break
			}
		}
	} else {
		log.Infof("runner: CSP=%s cleanup-only mode, skipping all resource tests", cspCfg.Name)
	}

	// Execute cleanup according to mode: true or only → run cleanup; false → skip.
	// Cleanup is skipped when the run was cancelled (Stop button pressed).
	// Cleanup always uses a fresh context with a 30-minute deadline so quick
	// deletes (kp, sg, vpc) always complete within its own budget.
	if ctx.Err() != nil {
		log.Infof("runner: CSP=%s cleanup skipped (run was stopped)", cspCfg.Name)
	} else {
		switch cleanupMode {
		case "true", "only":
			st.liveOps = nil
			st.notifyOp = func(opName string, done []model.OperationResult) {
				if notify != nil {
					notify(result, "cleanup", opName, done)
				}
			}
			if notify != nil {
				notify(result, "cleanup", "", nil)
			}
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cleanupCancel()
			cleanupRR := r.testCleanup(cleanupCtx, client, cfg, cspCfg.Connection, st)
			result.Resources = append(result.Resources, cleanupRR)
			log.Infof("runner: CSP=%s resource=cleanup status=%s", cspCfg.Name, cleanupRR.Status)
			st.notifyOp = nil
			st.liveOps = nil
			if notify != nil {
				notify(result, "", "", nil)
			}
		default: // "false" or anything else
			log.Infof("runner: CSP=%s cleanup skipped (cleanup: false)", cspCfg.Name)
		}
	}

	return result
}

// vpcCreateBody is the request body for POST /spider/vpc.
type vpcCreateBody struct {
	ConnectionName string     `json:"ConnectionName"`
	ReqInfo        vpcReqInfo `json:"ReqInfo"`
}

type vpcReqInfo struct {
	Name           string       `json:"Name"`
	IPv4CIDR       string       `json:"IPv4_CIDR"`
	SubnetInfoList []subnetInfo `json:"SubnetInfoList"`
}

type subnetInfo struct {
	Name     string `json:"Name"`
	IPv4CIDR string `json:"IPv4_CIDR"`
}

type deleteBody struct {
	ConnectionName string `json:"ConnectionName"`
}

// skipError marks an operation as intentionally skipped (not a failure).
type skipError struct{ reason string }

func (e *skipError) Error() string { return e.reason }

// isNotImplementedErr returns true when an error message indicates the operation
// is not supported by this CSP (e.g., HTTP 4xx/5xx body containing "not implemented"
// or "does not support XxxHandler" from CB-Spider driver dispatch).
func isNotImplementedErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not implemented") ||
		strings.Contains(msg, "does not support")
}

// isDoesNotExistBody returns true when the response body indicates the resource
// does not exist — some CSPs (e.g., Azure) return HTTP 500 instead of 404 for
// missing resources. Detecting these avoids needless 30-second retry waits.
func isDoesNotExistBody(body []byte) bool {
	msg := strings.ToLower(string(body))
	return strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "resource not found")
}

// opsStatus computes the resource-level status and error string from a slice of
// OperationResults. Any FAIL → FAIL; no FAIL but at least one SKIP → SKIP; else OK.
func opsStatus(ops []model.OperationResult) (model.ResourceStatus, string) {
	hasSkip := false
	for _, op := range ops {
		if op.Status == model.ResourceStatusFail {
			return model.ResourceStatusFail, fmt.Sprintf("[%s] %s", op.Op, op.Error)
		}
		if op.Status == model.ResourceStatusSkipped {
			hasSkip = true
		}
	}
	if hasSkip {
		return model.ResourceStatusSkipped, ""
	}
	return model.ResourceStatusOK, ""
}

// runOp executes a named operation function and wraps the result in OperationResult.
// If fn returns a *skipError the operation is marked SKIPPED instead of FAIL.
func runOp(name string, fn func() error) model.OperationResult {
	op := model.OperationResult{Op: name}
	start := time.Now()
	err := fn()
	op.DurationMs = time.Since(start).Milliseconds()
	var se *skipError
	if errors.As(err, &se) {
		op.Status = model.ResourceStatusSkipped
		op.Error = se.reason
	} else if err != nil && isNotImplementedErr(err) {
		op.Status = model.ResourceStatusSkipped
		op.Error = err.Error()
	} else if err != nil {
		op.Status = model.ResourceStatusFail
		op.Error = err.Error()
	} else {
		op.Status = model.ResourceStatusOK
	}
	return op
}

// testVPCCRUD runs create / list / get for VPC.
// The VPC is NOT deleted here; it is kept in st for dependent tests and
// removed in the shared cleanup step.
func (r *Runner) testVPCCRUD(ctx context.Context, client *http.Client, cfg *config.Config, connection string, vpcCfg config.VPCTestConfig, st *cspTestState) model.ResourceResult {
	start := time.Now()
	rr := model.ResourceResult{
		Kind:     "vpc",
		TestedAt: start,
	}
	apiBase := spiderAPIBase(cfg)
	testName := st.vpcName

	doReq := func(method, url string, bodyBytes []byte) ([]byte, int, error) {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, 0, err
		}
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.SetBasicAuth(cfg.Spider.Username, cfg.Spider.Password)
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return b, resp.StatusCode, nil
	}

	// 1. CREATE
	createOp := st.op("create", func() error {
		body := vpcCreateBody{
			ConnectionName: connection,
			ReqInfo: vpcReqInfo{
				Name:     testName,
				IPv4CIDR: vpcCfg.VPCCIDR,
				SubnetInfoList: []subnetInfo{
					{Name: st.subnetName, IPv4CIDR: vpcCfg.SubnetCIDR},
				},
			},
		}
		b, _ := json.Marshal(body)
		errBody, code, err := doReq(http.MethodPost, apiBase+"/vpc", b)
		if err != nil {
			return err
		}
		if code >= 400 {
			return fmt.Errorf("HTTP %d: %s", code, string(errBody))
		}
		st.vpcCreated = true
		return nil
	})
	rr.Operations = append(rr.Operations, createOp)

	// 2. LIST
	listOp := st.op("list", func() error {
		body, code, err := doReq(http.MethodGet, apiBase+"/vpc?ConnectionName="+connection, nil)
		if err != nil {
			return err
		}
		if code >= 400 {
			return fmt.Errorf("HTTP %d: %s", code, string(body))
		}
		cnt, err := extractCount("vpc", body)
		if err != nil {
			return fmt.Errorf("parse response: %v", err)
		}
		rr.Count = cnt
		return nil
	})
	rr.Operations = append(rr.Operations, listOp)

	// 3. GET — only when create succeeded
	if createOp.Status == model.ResourceStatusOK {
		getOp := st.op("get", func() error {
			url := fmt.Sprintf("%s/vpc/%s?ConnectionName=%s", apiBase, testName, connection)
			body, code, err := doReq(http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(body))
			}
			return nil
		})
		rr.Operations = append(rr.Operations, getOp)
	}

	// Overall status
	rr.Status, rr.Error = opsStatus(rr.Operations)
	rr.DurationMs = time.Since(start).Milliseconds()
	return rr
}

// kpCreateBody is the request body for POST /spider/keypair.
type kpCreateBody struct {
	ConnectionName string    `json:"ConnectionName"`
	ReqInfo        kpReqInfo `json:"ReqInfo"`
}

type kpReqInfo struct {
	Name string `json:"Name"`
}

// testKeyPairCRUD runs create / list / get for KeyPair.
// The KP is kept in st for the VM test and removed in the cleanup step.
func (r *Runner) testKeyPairCRUD(ctx context.Context, client *http.Client, cfg *config.Config, connection string, st *cspTestState) model.ResourceResult {
	start := time.Now()
	rr := model.ResourceResult{
		Kind:     "keypair",
		TestedAt: start,
	}
	apiBase := spiderAPIBase(cfg)
	testName := st.kpName

	doReq := func(method, url string, bodyBytes []byte) ([]byte, int, error) {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, 0, err
		}
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.SetBasicAuth(cfg.Spider.Username, cfg.Spider.Password)
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return b, resp.StatusCode, nil
	}

	// 1. CREATE — capture PrivateKey here; it is not available via LIST or GET.
	createOp := st.op("create", func() error {
		body := kpCreateBody{
			ConnectionName: connection,
			ReqInfo:        kpReqInfo{Name: testName},
		}
		b, _ := json.Marshal(body)
		respBytes, code, err := doReq(http.MethodPost, apiBase+"/keypair", b)
		if err != nil {
			return err
		}
		if code >= 400 {
			return fmt.Errorf("HTTP %d: %s", code, string(respBytes))
		}
		st.kpCreated = true
		var kpResp struct {
			PrivateKey string `json:"PrivateKey"`
		}
		if jerr := json.Unmarshal(respBytes, &kpResp); jerr == nil && kpResp.PrivateKey != "" {
			st.kpPrivateKey = kpResp.PrivateKey
		}
		return nil
	})
	rr.Operations = append(rr.Operations, createOp)

	// 2. LIST
	listOp := st.op("list", func() error {
		body, code, err := doReq(http.MethodGet, apiBase+"/keypair?ConnectionName="+connection, nil)
		if err != nil {
			return err
		}
		if code >= 400 {
			return fmt.Errorf("HTTP %d: %s", code, string(body))
		}
		cnt, err := extractCount("keypair", body)
		if err != nil {
			return fmt.Errorf("parse response: %v", err)
		}
		rr.Count = cnt
		return nil
	})
	rr.Operations = append(rr.Operations, listOp)

	if st.kpCreated {
		// 3. GET
		getOp := st.op("get", func() error {
			url := fmt.Sprintf("%s/keypair/%s?ConnectionName=%s", apiBase, testName, connection)
			body, code, err := doReq(http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(body))
			}
			return nil
		})
		rr.Operations = append(rr.Operations, getOp)
	}

	// Overall status
	rr.Status, rr.Error = opsStatus(rr.Operations)
	rr.DurationMs = time.Since(start).Milliseconds()
	return rr
}

// sgCreateBody is the request body for POST /spider/securitygroup.
type sgCreateBody struct {
	ConnectionName string    `json:"ConnectionName"`
	ReqInfo        sgReqInfo `json:"ReqInfo"`
}

type sgReqInfo struct {
	Name          string   `json:"Name"`
	VPCName       string   `json:"VPCName"`
	SecurityRules []sgRule `json:"SecurityRules"`
}

type sgRule struct {
	FromPort   string `json:"FromPort"`
	ToPort     string `json:"ToPort"`
	IPProtocol string `json:"IPProtocol"`
	Direction  string `json:"Direction"`
	CIDR       string `json:"CIDR"`
}

// testSecurityGroupCRUD runs create / list / get for SecurityGroup.
// If a VPC was already created by a previous test (st.vpcCreated) it is reused;
// otherwise a temp VPC is created here and tracked in st for cleanup.
func (r *Runner) testSecurityGroupCRUD(ctx context.Context, client *http.Client, cfg *config.Config, connection string, vpcCfg config.VPCTestConfig, st *cspTestState) model.ResourceResult {
	start := time.Now()
	rr := model.ResourceResult{
		Kind:     "securitygroup",
		TestedAt: start,
	}
	apiBase := spiderAPIBase(cfg)
	testName := st.sgName

	doReq := func(method, url string, bodyBytes []byte) ([]byte, int, error) {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, 0, err
		}
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.SetBasicAuth(cfg.Spider.Username, cfg.Spider.Password)
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return b, resp.StatusCode, nil
	}

	// 0. Ensure VPC exists (create only if not already created by vpc test)
	if !st.vpcCreated {
		vpcOp := st.op("vpc-create", func() error {
			body := vpcCreateBody{
				ConnectionName: connection,
				ReqInfo: vpcReqInfo{
					Name:     st.vpcName,
					IPv4CIDR: vpcCfg.VPCCIDR,
					SubnetInfoList: []subnetInfo{
						{Name: st.subnetName, IPv4CIDR: vpcCfg.SubnetCIDR},
					},
				},
			}
			b, _ := json.Marshal(body)
			errBody, code, err := doReq(http.MethodPost, apiBase+"/vpc", b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			st.vpcCreated = true
			return nil
		})
		rr.Operations = append(rr.Operations, vpcOp)
		if vpcOp.Status != model.ResourceStatusOK {
			goto done
		}
	}

	{
		// 1. CREATE SecurityGroup
		createOp := st.op("create", func() error {
			body := sgCreateBody{
				ConnectionName: connection,
				ReqInfo: sgReqInfo{
					Name:          testName,
					VPCName:       st.vpcName,
					SecurityRules: st.buildSGRules(),
				},
			}
			b, _ := json.Marshal(body)
			errBody, code, err := doReq(http.MethodPost, apiBase+"/securitygroup", b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			st.sgCreated = true
			return nil
		})
		rr.Operations = append(rr.Operations, createOp)

		// 2. LIST
		listOp := st.op("list", func() error {
			body, code, err := doReq(http.MethodGet, apiBase+"/securitygroup?ConnectionName="+connection, nil)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(body))
			}
			cnt, err := extractCount("securitygroup", body)
			if err != nil {
				return fmt.Errorf("parse response: %v", err)
			}
			rr.Count = cnt
			return nil
		})
		rr.Operations = append(rr.Operations, listOp)

		if st.sgCreated {
			// 3. GET
			getOp := st.op("get", func() error {
				url := fmt.Sprintf("%s/securitygroup/%s?ConnectionName=%s", apiBase, testName, connection)
				body, code, err := doReq(http.MethodGet, url, nil)
				if err != nil {
					return err
				}
				if code >= 400 {
					return fmt.Errorf("HTTP %d: %s", code, string(body))
				}
				return nil
			})
			rr.Operations = append(rr.Operations, getOp)
		}
	}

done:
	rr.Status, rr.Error = opsStatus(rr.Operations)
	rr.DurationMs = time.Since(start).Milliseconds()
	return rr
}

// vmCreateBody is the request body for POST /spider/vm.
type vmCreateBody struct {
	ConnectionName string    `json:"ConnectionName"`
	ReqInfo        vmReqInfo `json:"ReqInfo"`
}

type vmReqInfo struct {
	Name               string   `json:"Name"`
	ImageName          string   `json:"ImageName"`
	VPCName            string   `json:"VPCName"`
	SubnetName         string   `json:"SubnetName"`
	SecurityGroupNames []string `json:"SecurityGroupNames"`
	VMSpecName         string   `json:"VMSpecName"`
	KeyPairName        string   `json:"KeyPairName"`
}

// resolveImageName resolves the effective image name to use for a VM.
// If imageName starts with "__prefix:", the suffix is treated as a name prefix;
// this function calls GET /vmimage to list all public images and returns the
// lexicographically latest name that starts with the prefix (Alibaba-style
// images embed a date string, so the last name is the most recent).
// For any other value, imageName is returned unchanged.
func resolveImageName(ctx context.Context, client *http.Client, cfg *config.Config, connection, imageName string) (string, error) {
	const marker = "__prefix:"
	if !strings.HasPrefix(imageName, marker) {
		return imageName, nil
	}
	prefix := strings.TrimPrefix(imageName, marker)
	if prefix == "" {
		return "", fmt.Errorf("resolveImageName: __prefix: marker used but prefix is empty")
	}

	url := fmt.Sprintf("%s/vmimage?ConnectionName=%s", cfg.Spider.APIURL, connection)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("resolveImageName: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(cfg.Spider.Username, cfg.Spider.Password)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("resolveImageName: GET vmimage: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("resolveImageName: GET vmimage HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Image []struct {
			IId struct {
				NameId string `json:"NameId"`
			} `json:"IId"`
			Name string `json:"Name"`
		} `json:"image"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("resolveImageName: parse vmimage response: %w", err)
	}

	var latest string
	for _, img := range result.Image {
		// Use Name field; fall back to IId.NameId if Name is empty
		name := img.Name
		if name == "" {
			name = img.IId.NameId
		}
		if strings.HasPrefix(name, prefix) {
			if name > latest {
				latest = name
			}
		}
	}
	if latest == "" {
		return "", fmt.Errorf("resolveImageName: no image found with prefix %q (connection=%s)", prefix, connection)
	}
	log.Infof("runner: resolved image prefix %q → %q (connection=%s)", prefix, latest, connection)
	return latest, nil
}

// checkSSH attempts an SSH connection to host:22 using the given user and PEM private key.
// It retries up to maxAttempts times with retryInterval between attempts to allow
// cloud-init time to provision the user account after VM creation.
func checkSSH(ctx context.Context, host, user, pemKey string) error {
	signer, err := ssh.ParsePrivateKey([]byte(pemKey))
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // test-only check
		Timeout:         30 * time.Second,
	}
	addr := net.JoinHostPort(host, "22")

	const maxAttempts = 10
	const retryInterval = 30 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Infof("runner: SSH login attempt %d/%d — host=%s user=%s", attempt, maxAttempts, host, user)

		type result struct {
			client *ssh.Client
			err    error
		}
		ch := make(chan result, 1)
		go func() {
			c, e := ssh.Dial("tcp", addr, cfg)
			ch <- result{c, e}
		}()

		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled waiting for SSH: %w", ctx.Err())
		case r := <-ch:
			if r.err == nil {
				r.client.Close()
				log.Infof("runner: SSH login OK — host=%s user=%s (attempt %d)", host, user, attempt)
				return nil
			}
			log.Warnf("runner: SSH attempt %d failed: %v", attempt, r.err)
			if attempt == maxAttempts {
				return fmt.Errorf("SSH dial %s: %w", addr, r.err)
			}
		}

		// Wait before retrying, but respect context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled waiting for SSH retry: %w", ctx.Err())
		case <-time.After(retryInterval):
		}
	}
	return fmt.Errorf("SSH login failed after %d attempts", maxAttempts)
}

// sshRunCommand connects to host via SSH and runs a single shell command,
// returning an error that includes combined stdout+stderr on failure.
func sshRunCommand(ctx context.Context, host, user, pemKey, command string) error {
	signer, err := ssh.ParsePrivateKey([]byte(pemKey))
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // test-only
		Timeout:         30 * time.Second,
	}
	addr := net.JoinHostPort(host, "22")

	type dialResult struct {
		client *ssh.Client
		err    error
	}
	ch := make(chan dialResult, 1)
	go func() {
		c, e := ssh.Dial("tcp", addr, cfg)
		ch <- dialResult{c, e}
	}()
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled waiting for SSH: %w", ctx.Err())
	case r := <-ch:
		if r.err != nil {
			return fmt.Errorf("SSH dial %s: %w", addr, r.err)
		}
		defer r.client.Close()
		sess, err := r.client.NewSession()
		if err != nil {
			return fmt.Errorf("SSH new session: %w", err)
		}
		defer sess.Close()
		out, err := sess.CombinedOutput(command)
		if err != nil {
			return fmt.Errorf("SSH command failed: %w — %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}
}

// testVMCRUD runs create / list / get / ssh-login for VM.
// It reuses VPC/SG/KP already in st (created by earlier tests); if they are
// absent it creates them here and records them in st for cleanup.
// The VM is NOT deleted here; it is kept for the myimage test.
func (r *Runner) testVMCRUD(ctx context.Context, client *http.Client, cfg *config.Config, connection string, vpcCfg config.VPCTestConfig, st *cspTestState) model.ResourceResult {
	start := time.Now()
	rr := model.ResourceResult{
		Kind:     "vm",
		TestedAt: start,
	}
	apiBase := spiderAPIBase(cfg)

	doReq := func(method, url string, bodyBytes []byte) ([]byte, int, error) {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, 0, err
		}
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.SetBasicAuth(cfg.Spider.Username, cfg.Spider.Password)
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		return b, resp.StatusCode, nil
	}

	// Reuse the private key captured at keypair creation time (shared state).
	privateKey := st.kpPrivateKey
	var vmPublicIP string

	// 1. Ensure VPC exists
	if !st.vpcCreated {
		vpcOp := st.op("vpc-create", func() error {
			body := vpcCreateBody{
				ConnectionName: connection,
				ReqInfo: vpcReqInfo{
					Name:     st.vpcName,
					IPv4CIDR: vpcCfg.VPCCIDR,
					SubnetInfoList: []subnetInfo{
						{Name: st.subnetName, IPv4CIDR: vpcCfg.SubnetCIDR},
					},
				},
			}
			b, _ := json.Marshal(body)
			errBody, code, err := doReq(http.MethodPost, apiBase+"/vpc", b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			st.vpcCreated = true
			return nil
		})
		rr.Operations = append(rr.Operations, vpcOp)
		if vpcOp.Status != model.ResourceStatusOK {
			goto done
		}
	}

	// 2. Ensure SG exists
	if !st.sgCreated {
		sgOp := st.op("sg-create", func() error {
			body := sgCreateBody{
				ConnectionName: connection,
				ReqInfo: sgReqInfo{
					Name:          st.sgName,
					VPCName:       st.vpcName,
					SecurityRules: st.buildSGRules(),
				},
			}
			b, _ := json.Marshal(body)
			errBody, code, err := doReq(http.MethodPost, apiBase+"/securitygroup", b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			st.sgCreated = true
			return nil
		})
		rr.Operations = append(rr.Operations, sgOp)
		if sgOp.Status != model.ResourceStatusOK {
			goto done
		}
	}

	// 3. Ensure KP exists
	if !st.kpCreated {
		kpOp := st.op("kp-create", func() error {
			body := kpCreateBody{
				ConnectionName: connection,
				ReqInfo:        kpReqInfo{Name: st.kpName},
			}
			b, _ := json.Marshal(body)
			respBytes, code, err := doReq(http.MethodPost, apiBase+"/keypair", b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(respBytes))
			}
			st.kpCreated = true
			var kpResp struct {
				PrivateKey string `json:"PrivateKey"`
			}
			if jerr := json.Unmarshal(respBytes, &kpResp); jerr == nil && kpResp.PrivateKey != "" {
				st.kpPrivateKey = kpResp.PrivateKey
				privateKey = kpResp.PrivateKey
			}
			return nil
		})
		rr.Operations = append(rr.Operations, kpOp)
		if kpOp.Status != model.ResourceStatusOK {
			goto done
		}
	}

	// Sync privateKey from shared state (keypair test may have set it before vm test).
	if privateKey == "" {
		privateKey = st.kpPrivateKey
	}

	{
		imageName, err := resolveImageName(ctx, client, cfg, connection, st.vmTest.ImageName)
		if err != nil {
			rr.Operations = append(rr.Operations, model.OperationResult{
				Op:     "create",
				Status: model.ResourceStatusFail,
				Error:  err.Error(),
			})
			goto done
		}
		specName := st.vmTest.SpecName
		log.Infof("runner: vm test using image=%s spec=%s", imageName, specName)

		// 4. CREATE VM
		createOp := st.op("create", func() error {
			body := vmCreateBody{
				ConnectionName: connection,
				ReqInfo: vmReqInfo{
					Name:               st.vmName,
					ImageName:          imageName,
					VPCName:            st.vpcName,
					SubnetName:         st.subnetName,
					SecurityGroupNames: []string{st.sgName},
					VMSpecName:         specName,
					KeyPairName:        st.kpName,
				},
			}
			b, _ := json.Marshal(body)
			respBytes, code, err := doReq(http.MethodPost, apiBase+"/vm", b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(respBytes))
			}
			st.vmCreated = true
			var vmResp struct {
				PublicIP string `json:"PublicIP"`
			}
			if jerr := json.Unmarshal(respBytes, &vmResp); jerr == nil {
				vmPublicIP = vmResp.PublicIP
			}
			return nil
		})
		rr.Operations = append(rr.Operations, createOp)
		if createOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 5. LIST
		listOp := st.op("list", func() error {
			body, code, err := doReq(http.MethodGet, apiBase+"/vm?ConnectionName="+connection, nil)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(body))
			}
			cnt, err := extractCount("vm", body)
			if err != nil {
				return fmt.Errorf("parse response: %v", err)
			}
			rr.Count = cnt
			return nil
		})
		rr.Operations = append(rr.Operations, listOp)

		// 6. GET
		getOp := st.op("get", func() error {
			url := fmt.Sprintf("%s/vm/%s?ConnectionName=%s", apiBase, st.vmName, connection)
			body, code, err := doReq(http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(body))
			}
			return nil
		})
		rr.Operations = append(rr.Operations, getOp)

		// 7. SSH login check — fails when no public IP or private key is available.
		sshOp := st.op("ssh-login", func() error {
			if vmPublicIP == "" {
				return fmt.Errorf("no public IP available for SSH check")
			}
			if privateKey == "" {
				return fmt.Errorf("no private key configured")
			}
			if err := checkSSH(ctx, vmPublicIP, "cb-user", privateKey); err != nil {
				return err
			}
			st.vmPublicIP = vmPublicIP
			st.vmSshLoginOK = true
			return nil
		})
		rr.Operations = append(rr.Operations, sshOp)

		// 8. nginx pre-install — only when the CSP's NLB health-checker uses HTTP
		// and therefore requires port 80 to be served on the VM.
		// Controlled by nlb_test.nginx_pre_install: true in spiderwatch.yaml.
		if st.nlbTest.NginxPreInstall {
			nginxOp := st.op("ssh-nginx-install", func() error {
				if !st.vmSshLoginOK {
					return fmt.Errorf("ssh-login failed — skipping nginx install")
				}
				return sshRunCommand(ctx, vmPublicIP, "cb-user", privateKey,
					"sudo apt-get update -qq && sudo apt-get install -y -qq nginx && sudo systemctl enable --now nginx")
			})
			rr.Operations = append(rr.Operations, nginxOp)
		}
	}

done:
	rr.Status, rr.Error = opsStatus(rr.Operations)
	rr.DurationMs = time.Since(start).Milliseconds()
	return rr
}

// nlbCreateBody is the request body for POST /spider/nlb.
type nlbCreateBody struct {
	ConnectionName string     `json:"ConnectionName"`
	ReqInfo        nlbReqInfo `json:"ReqInfo"`
}

type nlbReqInfo struct {
	Name          string           `json:"Name"`
	VPCName       string           `json:"VPCName"`
	Type          string           `json:"Type"`
	Scope         string           `json:"Scope"`
	Listener      nlbListener      `json:"Listener"`
	VMGroup       nlbVMGroup       `json:"VMGroup,omitempty"`
	HealthChecker nlbHealthChecker `json:"HealthChecker"`
}

type nlbListener struct {
	Protocol string `json:"Protocol"`
	Port     string `json:"Port"`
}

type nlbVMGroup struct {
	Protocol string   `json:"Protocol"`
	Port     string   `json:"Port"`
	VMs      []string `json:"VMs,omitempty"`
}

type nlbHealthChecker struct {
	Protocol  string `json:"Protocol"`
	Port      string `json:"Port"`
	Interval  string `json:"Interval,omitempty"`
	Timeout   string `json:"Timeout,omitempty"`
	Threshold string `json:"Threshold,omitempty"`
}

// testNLBCRUD runs create / list / get for NLB.
// Requires a VPC (reused from st if already created); no delete here — cleanup handles it.
func (r *Runner) testNLBCRUD(ctx context.Context, client *http.Client, cfg *config.Config, connection string, nlbCfg config.NLBTestConfig, vpcCfg config.VPCTestConfig, st *cspTestState) model.ResourceResult {
	start := time.Now()
	rr := model.ResourceResult{
		Kind:     "nlb",
		TestedAt: start,
	}
	apiBase := spiderAPIBase(cfg)

	doReq := func(method, url string, bodyBytes []byte) ([]byte, int, error) {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, 0, err
		}
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.SetBasicAuth(cfg.Spider.Username, cfg.Spider.Password)
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return b, resp.StatusCode, nil
	}

	// Validate required NLB config
	if nlbCfg.Type == "" || nlbCfg.ListenerProtocol == "" || nlbCfg.ListenerPort == "" {
		rr.Status = model.ResourceStatusFail
		rr.Error = "nlb_test config missing required fields (type, listener_protocol, listener_port)"
		rr.DurationMs = time.Since(start).Milliseconds()
		return rr
	}

	// 1. Ensure VPC exists
	if !st.vpcCreated {
		vpcOp := st.op("vpc-create", func() error {
			body := vpcCreateBody{
				ConnectionName: connection,
				ReqInfo: vpcReqInfo{
					Name:     st.vpcName,
					IPv4CIDR: vpcCfg.VPCCIDR,
					SubnetInfoList: []subnetInfo{
						{Name: st.subnetName, IPv4CIDR: vpcCfg.SubnetCIDR},
					},
				},
			}
			b, _ := json.Marshal(body)
			errBody, code, err := doReq(http.MethodPost, apiBase+"/vpc", b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			st.vpcCreated = true
			return nil
		})
		rr.Operations = append(rr.Operations, vpcOp)
		if vpcOp.Status != model.ResourceStatusOK {
			goto done
		}
	}

	{
		// 2. CREATE NLB
		createOp := st.op("create", func() error {
			body := nlbCreateBody{
				ConnectionName: connection,
				ReqInfo: nlbReqInfo{
					Name:    st.nlbName,
					VPCName: st.vpcName,
					Type:    nlbCfg.Type,
					Scope:   nlbCfg.Scope,
					Listener: nlbListener{
						Protocol: nlbCfg.ListenerProtocol,
						Port:     nlbCfg.ListenerPort,
					},
					VMGroup: nlbVMGroup{
						Protocol: nlbCfg.TargetProtocol,
						Port:     nlbCfg.TargetPort,
					},
					HealthChecker: nlbHealthChecker{
						Protocol:  nlbCfg.HealthProtocol,
						Port:      nlbCfg.HealthPort,
						Interval:  nlbCfg.HealthInterval,
						Timeout:   nlbCfg.HealthTimeout,
						Threshold: nlbCfg.HealthThreshold,
					},
				},
			}
			b, _ := json.Marshal(body)
			errBody, code, err := doReq(http.MethodPost, apiBase+"/nlb", b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			st.nlbCreated = true
			return nil
		})
		rr.Operations = append(rr.Operations, createOp)
		if createOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 3. LIST
		listOp := st.op("list", func() error {
			body, code, err := doReq(http.MethodGet, apiBase+"/nlb?ConnectionName="+connection, nil)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(body))
			}
			cnt, err := extractCount("nlb", body)
			if err != nil {
				return fmt.Errorf("parse response: %v", err)
			}
			rr.Count = cnt
			return nil
		})
		rr.Operations = append(rr.Operations, listOp)

		// 4. GET
		getOp := st.op("get", func() error {
			url := fmt.Sprintf("%s/nlb/%s?ConnectionName=%s", apiBase, st.nlbName, connection)
			body, code, err := doReq(http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(body))
			}
			return nil
		})
		rr.Operations = append(rr.Operations, getOp)

		// 5. ADD-VM — add the test VM to the NLB's VMGroup.
		// Requires a VM to already exist (created by the vm test).
		addVMOp := st.op("add-vm", func() error {
			if !st.vmCreated {
				return &skipError{"no VM available to add to NLB"}
			}
			addBody := struct {
				ConnectionName string `json:"ConnectionName"`
				ReqInfo        struct {
					VMs []string `json:"VMs"`
				} `json:"ReqInfo"`
			}{
				ConnectionName: connection,
			}
			addBody.ReqInfo.VMs = []string{st.vmName}
			b, _ := json.Marshal(addBody)
			url := fmt.Sprintf("%s/nlb/%s/vms", apiBase, st.nlbName)
			errBody, code, err := doReq(http.MethodPost, url, b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			return nil
		})
		rr.Operations = append(rr.Operations, addVMOp)
		if addVMOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 6. HEALTH-CHECK — poll until the added VM appears in HealthyVMs.
		// Uses up to 10 × 30s = 5 min for the health status to settle.
		healthOp := st.op("health-check", func() error {
			healthURL := fmt.Sprintf("%s/nlb/%s/health?ConnectionName=%s", apiBase, st.nlbName, connection)
			const maxAttempts = 10
			const interval = 30 * time.Second
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				body, code, err := doReq(http.MethodGet, healthURL, nil)
				if err != nil {
					return err
				}
				// 4xx means a client-side error (bad request, not found) — fail immediately.
				// 5xx means the CSP's health data isn't ready yet (e.g. GCP Target Pool
				// returns 500 until the pool's health-check has settled) — retry.
				if code >= 500 {
					log.Warnf("runner: NLB health-check attempt %d/%d: HTTP %d (will retry) — %s",
						attempt, maxAttempts, code, string(body))
					if attempt == maxAttempts {
						return fmt.Errorf("HTTP %d: %s", code, string(body))
					}
					select {
					case <-ctx.Done():
						return fmt.Errorf("context cancelled waiting for NLB health: %w", ctx.Err())
					case <-time.After(interval):
					}
					continue
				}
				if code >= 400 {
					return fmt.Errorf("HTTP %d: %s", code, string(body))
				}
				var resp struct {
					HealthInfo struct {
						AllVMs []struct {
							NameId string `json:"NameId"`
						} `json:"AllVMs"`
						HealthyVMs []struct {
							NameId string `json:"NameId"`
						} `json:"HealthyVMs"`
						UnHealthyVMs []struct {
							NameId string `json:"NameId"`
						} `json:"UnHealthyVMs"`
					} `json:"healthinfo"`
				}
				if jerr := json.Unmarshal(body, &resp); jerr != nil {
					return fmt.Errorf("parse health response: %v", jerr)
				}
				for _, vm := range resp.HealthInfo.HealthyVMs {
					if vm.NameId == st.vmName {
						log.Infof("runner: NLB %s VM %s is Healthy (attempt %d/%d)", st.nlbName, st.vmName, attempt, maxAttempts)
						return nil
					}
				}
				log.Infof("runner: NLB %s VM %s not yet Healthy (attempt %d/%d, healthy=%d, unhealthy=%d)",
					st.nlbName, st.vmName, attempt, maxAttempts,
					len(resp.HealthInfo.HealthyVMs), len(resp.HealthInfo.UnHealthyVMs))
				if attempt == maxAttempts {
					return fmt.Errorf("VM %s did not become Healthy after %d attempts", st.vmName, maxAttempts)
				}
				select {
				case <-ctx.Done():
					return fmt.Errorf("context cancelled waiting for NLB health: %w", ctx.Err())
				case <-time.After(interval):
				}
			}
			return nil
		})
		rr.Operations = append(rr.Operations, healthOp)
		if healthOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 7. REMOVE-VM — detach the test VM from the NLB's VMGroup.
		removeVMOp := st.op("remove-vm", func() error {
			removeBody := struct {
				ConnectionName string `json:"ConnectionName"`
				ReqInfo        struct {
					VMs []string `json:"VMs"`
				} `json:"ReqInfo"`
			}{ConnectionName: connection}
			removeBody.ReqInfo.VMs = []string{st.vmName}
			b, _ := json.Marshal(removeBody)
			url := fmt.Sprintf("%s/nlb/%s/vms", apiBase, st.nlbName)
			errBody, code, err := doReq(http.MethodDelete, url, b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			return nil
		})
		rr.Operations = append(rr.Operations, removeVMOp)
		if removeVMOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 8. HEALTH-WAIT — poll until the VM is no longer visible in AllVMs.
		healthWaitOp := st.op("health-wait", func() error {
			healthURL := fmt.Sprintf("%s/nlb/%s/health?ConnectionName=%s", apiBase, st.nlbName, connection)
			const maxAttempts = 20
			const interval = 30 * time.Second
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				body, code, err := doReq(http.MethodGet, healthURL, nil)
				if err != nil {
					return err
				}
				if code >= 400 {
					return fmt.Errorf("HTTP %d: %s", code, string(body))
				}
				var resp struct {
					HealthInfo struct {
						AllVMs []struct {
							NameId string `json:"NameId"`
						} `json:"AllVMs"`
					} `json:"healthinfo"`
				}
				if jerr := json.Unmarshal(body, &resp); jerr != nil {
					return fmt.Errorf("parse health response: %v", jerr)
				}
				found := false
				for _, vm := range resp.HealthInfo.AllVMs {
					if vm.NameId == st.vmName {
						found = true
						break
					}
				}
				if !found {
					log.Infof("runner: NLB %s VM %s no longer visible in health-check (attempt %d/%d)",
						st.nlbName, st.vmName, attempt, maxAttempts)
					return nil
				}
				log.Infof("runner: NLB %s VM %s still visible, waiting (attempt %d/%d, allVMs=%d)",
					st.nlbName, st.vmName, attempt, maxAttempts, len(resp.HealthInfo.AllVMs))
				if attempt == maxAttempts {
					return fmt.Errorf("VM %s still visible in NLB health-check after %d attempts", st.vmName, maxAttempts)
				}
				select {
				case <-ctx.Done():
					return fmt.Errorf("context cancelled waiting for NLB health-wait: %w", ctx.Err())
				case <-time.After(interval):
				}
			}
			return nil
		})
		rr.Operations = append(rr.Operations, healthWaitOp)
	}

done:
	rr.Status, rr.Error = opsStatus(rr.Operations)
	rr.DurationMs = time.Since(start).Milliseconds()
	return rr
}

// diskCreateBody is the request body for POST /spider/disk.
type diskCreateBody struct {
	ConnectionName string      `json:"ConnectionName"`
	ReqInfo        diskReqInfo `json:"ReqInfo"`
}

type diskReqInfo struct {
	Name     string `json:"Name"`
	DiskType string `json:"DiskType"`
	DiskSize string `json:"DiskSize"`
}

// testDiskCRUD runs create / list / get for Disk.
// Disk has no VPC/SG/KP dependency. It is kept in st for the cleanup step.
func (r *Runner) testDiskCRUD(ctx context.Context, client *http.Client, cfg *config.Config, connection string, st *cspTestState) model.ResourceResult {
	start := time.Now()
	rr := model.ResourceResult{
		Kind:     "disk",
		TestedAt: start,
	}
	apiBase := spiderAPIBase(cfg)
	testName := st.diskName

	doReq := func(method, url string, bodyBytes []byte) ([]byte, int, error) {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, 0, err
		}
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.SetBasicAuth(cfg.Spider.Username, cfg.Spider.Password)
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return b, resp.StatusCode, nil
	}

	diskType := st.diskTest.DiskType
	diskSize := st.diskTest.DiskSize

	// 1. CREATE
	createOp := st.op("create", func() error {
		body := diskCreateBody{
			ConnectionName: connection,
			ReqInfo: diskReqInfo{
				Name:     testName,
				DiskType: diskType,
				DiskSize: diskSize,
			},
		}
		b, _ := json.Marshal(body)
		errBody, code, err := doReq(http.MethodPost, apiBase+"/disk", b)
		if err != nil {
			return err
		}
		if code >= 400 {
			return fmt.Errorf("HTTP %d: %s", code, string(errBody))
		}
		st.diskCreated = true
		return nil
	})
	rr.Operations = append(rr.Operations, createOp)

	// 2. LIST
	listOp := st.op("list", func() error {
		body, code, err := doReq(http.MethodGet, apiBase+"/disk?ConnectionName="+connection, nil)
		if err != nil {
			return err
		}
		if code >= 400 {
			return fmt.Errorf("HTTP %d: %s", code, string(body))
		}
		cnt, err := extractCount("disk", body)
		if err != nil {
			return fmt.Errorf("parse response: %v", err)
		}
		rr.Count = cnt
		return nil
	})
	rr.Operations = append(rr.Operations, listOp)

	if st.diskCreated {
		// 3. GET
		getOp := st.op("get", func() error {
			url := fmt.Sprintf("%s/disk/%s?ConnectionName=%s", apiBase, testName, connection)
			body, code, err := doReq(http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(body))
			}
			return nil
		})
		rr.Operations = append(rr.Operations, getOp)
		if getOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 4. ATTACH-VM — attach the test disk to the test VM.
		attachOp := st.op("attach-vm", func() error {
			if !st.vmCreated {
				return &skipError{"no VM available to attach disk"}
			}
			attachBody := struct {
				ConnectionName string `json:"ConnectionName"`
				ReqInfo        struct {
					VMName string `json:"VMName"`
				} `json:"ReqInfo"`
			}{ConnectionName: connection}
			attachBody.ReqInfo.VMName = st.vmName
			b, _ := json.Marshal(attachBody)
			url := fmt.Sprintf("%s/disk/%s/attach", apiBase, testName)
			errBody, code, err := doReq(http.MethodPut, url, b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			return nil
		})
		rr.Operations = append(rr.Operations, attachOp)
		if attachOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 5. CHECK-ATTACHED — poll GET /disk/{name} until OwnerVM.NameId == vmName.
		checkAttachedOp := st.op("check-attached", func() error {
			url := fmt.Sprintf("%s/disk/%s?ConnectionName=%s", apiBase, testName, connection)
			const maxAttempts = 20
			const interval = 30 * time.Second
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				body, code, err := doReq(http.MethodGet, url, nil)
				if err != nil {
					return err
				}
				if code >= 400 {
					return fmt.Errorf("HTTP %d: %s", code, string(body))
				}
				var resp struct {
					OwnerVM struct {
						NameId string `json:"NameId"`
					} `json:"OwnerVM"`
				}
				if jerr := json.Unmarshal(body, &resp); jerr != nil {
					return fmt.Errorf("parse disk response: %v", jerr)
				}
				if resp.OwnerVM.NameId == st.vmName {
					log.Infof("runner: disk %s attached to VM %s (attempt %d/%d)", testName, st.vmName, attempt, maxAttempts)
					return nil
				}
				log.Infof("runner: disk %s not yet attached (ownerVM=%q, attempt %d/%d)",
					testName, resp.OwnerVM.NameId, attempt, maxAttempts)
				if attempt == maxAttempts {
					return fmt.Errorf("disk %s did not attach to VM %s after %d attempts", testName, st.vmName, maxAttempts)
				}
				select {
				case <-ctx.Done():
					return fmt.Errorf("context cancelled waiting for disk attach: %w", ctx.Err())
				case <-time.After(interval):
				}
			}
			return nil
		})
		rr.Operations = append(rr.Operations, checkAttachedOp)
		if checkAttachedOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 6. DETACH-VM — detach the disk from the VM.
		detachOp := st.op("detach-vm", func() error {
			detachBody := struct {
				ConnectionName string `json:"ConnectionName"`
				ReqInfo        struct {
					VMName string `json:"VMName"`
				} `json:"ReqInfo"`
			}{ConnectionName: connection}
			detachBody.ReqInfo.VMName = st.vmName
			b, _ := json.Marshal(detachBody)
			url := fmt.Sprintf("%s/disk/%s/detach", apiBase, testName)
			errBody, code, err := doReq(http.MethodPut, url, b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			return nil
		})
		rr.Operations = append(rr.Operations, detachOp)
		if detachOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 7. CHECK-DETACHED — poll GET /disk/{name} until OwnerVM.NameId is empty.
		checkDetachedOp := st.op("check-detached", func() error {
			url := fmt.Sprintf("%s/disk/%s?ConnectionName=%s", apiBase, testName, connection)
			const maxAttempts = 20
			const interval = 30 * time.Second
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				body, code, err := doReq(http.MethodGet, url, nil)
				if err != nil {
					return err
				}
				if code >= 400 {
					return fmt.Errorf("HTTP %d: %s", code, string(body))
				}
				var resp struct {
					OwnerVM struct {
						NameId string `json:"NameId"`
					} `json:"OwnerVM"`
				}
				if jerr := json.Unmarshal(body, &resp); jerr != nil {
					return fmt.Errorf("parse disk response: %v", jerr)
				}
				if resp.OwnerVM.NameId == "" {
					log.Infof("runner: disk %s detached from VM (attempt %d/%d)", testName, attempt, maxAttempts)
					return nil
				}
				log.Infof("runner: disk %s still attached to %q, waiting (attempt %d/%d)",
					testName, resp.OwnerVM.NameId, attempt, maxAttempts)
				if attempt == maxAttempts {
					return fmt.Errorf("disk %s did not detach from VM after %d attempts", testName, maxAttempts)
				}
				select {
				case <-ctx.Done():
					return fmt.Errorf("context cancelled waiting for disk detach: %w", ctx.Err())
				case <-time.After(interval):
				}
			}
			return nil
		})
		rr.Operations = append(rr.Operations, checkDetachedOp)
	}

done:
	rr.Status, rr.Error = opsStatus(rr.Operations)
	rr.DurationMs = time.Since(start).Milliseconds()
	return rr
}

// myImageCreateBody is the request body for POST /spider/myimage.
type myImageCreateBody struct {
	ConnectionName string         `json:"ConnectionName"`
	ReqInfo        myImageReqInfo `json:"ReqInfo"`
}

type myImageReqInfo struct {
	Name     string `json:"Name"`
	SourceVM string `json:"SourceVM"`
}

// testMyImageCRUD runs myimage create / list / get for MyImage.
// It reuses VPC/SG/KP/VM already in st (created by earlier tests); if they are
// absent it creates them here.  No resources are deleted here — cleanup does it.
func (r *Runner) testMyImageCRUD(ctx context.Context, client *http.Client, cfg *config.Config, connection string, vpcCfg config.VPCTestConfig, st *cspTestState) model.ResourceResult {
	start := time.Now()
	rr := model.ResourceResult{
		Kind:     "myimage",
		TestedAt: start,
	}
	apiBase := spiderAPIBase(cfg)

	doReq := func(method, url string, bodyBytes []byte) ([]byte, int, error) {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, 0, err
		}
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.SetBasicAuth(cfg.Spider.Username, cfg.Spider.Password)
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		return b, resp.StatusCode, nil
	}

	// 1. Ensure VPC exists
	if !st.vpcCreated {
		vpcOp := st.op("vpc-create", func() error {
			body := vpcCreateBody{
				ConnectionName: connection,
				ReqInfo: vpcReqInfo{
					Name:     st.vpcName,
					IPv4CIDR: vpcCfg.VPCCIDR,
					SubnetInfoList: []subnetInfo{
						{Name: st.subnetName, IPv4CIDR: vpcCfg.SubnetCIDR},
					},
				},
			}
			b, _ := json.Marshal(body)
			errBody, code, err := doReq(http.MethodPost, apiBase+"/vpc", b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			st.vpcCreated = true
			return nil
		})
		rr.Operations = append(rr.Operations, vpcOp)
		if vpcOp.Status != model.ResourceStatusOK {
			goto done
		}
	}

	// 2. Ensure SG exists
	if !st.sgCreated {
		sgOp := st.op("sg-create", func() error {
			body := sgCreateBody{
				ConnectionName: connection,
				ReqInfo: sgReqInfo{
					Name:          st.sgName,
					VPCName:       st.vpcName,
					SecurityRules: st.buildSGRules(),
				},
			}
			b, _ := json.Marshal(body)
			errBody, code, err := doReq(http.MethodPost, apiBase+"/securitygroup", b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			st.sgCreated = true
			return nil
		})
		rr.Operations = append(rr.Operations, sgOp)
		if sgOp.Status != model.ResourceStatusOK {
			goto done
		}
	}

	// 3. Ensure KP exists
	if !st.kpCreated {
		kpOp := st.op("kp-create", func() error {
			body := kpCreateBody{
				ConnectionName: connection,
				ReqInfo:        kpReqInfo{Name: st.kpName},
			}
			b, _ := json.Marshal(body)
			errBody, code, err := doReq(http.MethodPost, apiBase+"/keypair", b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			st.kpCreated = true
			return nil
		})
		rr.Operations = append(rr.Operations, kpOp)
		if kpOp.Status != model.ResourceStatusOK {
			goto done
		}
	}

	// 4. Ensure VM exists
	if !st.vmCreated {
		imageName, err := resolveImageName(ctx, client, cfg, connection, st.vmTest.ImageName)
		if err != nil {
			rr.Operations = append(rr.Operations, model.OperationResult{
				Op:     "vm-create",
				Status: model.ResourceStatusFail,
				Error:  err.Error(),
			})
			goto done
		}
		specName := st.vmTest.SpecName
		log.Infof("runner: myimage test creating vm image=%s spec=%s", imageName, specName)
		vmOp := st.op("vm-create", func() error {
			body := vmCreateBody{
				ConnectionName: connection,
				ReqInfo: vmReqInfo{
					Name:               st.vmName,
					ImageName:          imageName,
					VPCName:            st.vpcName,
					SubnetName:         st.subnetName,
					SecurityGroupNames: []string{st.sgName},
					VMSpecName:         specName,
					KeyPairName:        st.kpName,
				},
			}
			b, _ := json.Marshal(body)
			errBody, code, err := doReq(http.MethodPost, apiBase+"/vm", b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			st.vmCreated = true
			return nil
		})
		rr.Operations = append(rr.Operations, vmOp)
		if vmOp.Status != model.ResourceStatusOK {
			goto done
		}
	}

	{
		// 5. vm-wait: poll until Running via /vmstatus endpoint (up to 20 × 30s = 10 min).
		waitOp := st.op("vm-wait", func() error {
			statusURL := fmt.Sprintf("%s/vmstatus/%s?ConnectionName=%s", apiBase, st.vmName, connection)
			const maxAttempts = 20
			const interval = 30 * time.Second
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				body, code, err := doReq(http.MethodGet, statusURL, nil)
				if err != nil {
					return err
				}
				if code >= 400 {
					return fmt.Errorf("HTTP %d: %s", code, string(body))
				}
				var vmResp struct {
					Status string `json:"Status"`
				}
				status := ""
				if jerr := json.Unmarshal(body, &vmResp); jerr == nil {
					status = vmResp.Status
				}
				log.Infof("runner: vm %s status=%q (attempt %d/%d)", st.vmName, status, attempt, maxAttempts)
				if strings.EqualFold(status, "running") {
					return nil
				}
				if attempt == maxAttempts {
					return fmt.Errorf("vm %s did not reach Running state after %d attempts, last status=%q", st.vmName, maxAttempts, status)
				}
				select {
				case <-ctx.Done():
					return fmt.Errorf("context cancelled waiting for vm Running: %w", ctx.Err())
				case <-time.After(interval):
				}
			}
			return nil
		})
		rr.Operations = append(rr.Operations, waitOp)
		if waitOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 6. myimage create
		createOp := st.op("create", func() error {
			body := myImageCreateBody{
				ConnectionName: connection,
				ReqInfo: myImageReqInfo{
					Name:     st.myImgName,
					SourceVM: st.vmName,
				},
			}
			b, _ := json.Marshal(body)
			errBody, code, err := doReq(http.MethodPost, apiBase+"/myimage", b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			st.myImgCreated = true
			return nil
		})
		rr.Operations = append(rr.Operations, createOp)
		if createOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 7. LIST
		listOp := st.op("list", func() error {
			body, code, err := doReq(http.MethodGet, apiBase+"/myimage?ConnectionName="+connection, nil)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(body))
			}
			cnt, err := extractCount("myimage", body)
			if err != nil {
				return fmt.Errorf("parse response: %v", err)
			}
			rr.Count = cnt
			return nil
		})
		rr.Operations = append(rr.Operations, listOp)

		// 8. GET
		getOp := st.op("get", func() error {
			url := fmt.Sprintf("%s/myimage/%s?ConnectionName=%s", apiBase, st.myImgName, connection)
			body, code, err := doReq(http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(body))
			}
			return nil
		})
		rr.Operations = append(rr.Operations, getOp)
	}

done:
	rr.Status, rr.Error = opsStatus(rr.Operations)
	rr.DurationMs = time.Since(start).Milliseconds()
	return rr
}

// testClusterCRUD runs create / wait-active / nodegroup-add-or-verify / list / get /
// remove-nodegroup / wait-ng-gone for Cluster.
//
// Type-I CSPs (AWS, ALIBABA, TENCENT): NodeGroup is added separately after the cluster
// reaches Active status (node_group_type: "type1").
// Type-II CSPs (AZURE, GCP, NHN, NCP, IBM, …): NodeGroup is included in the create request
// (node_group_type: "type2").
//
// Both cluster create/delete and NodeGroup operations use a 120 × 30 s = 1 hour timeout.
func (r *Runner) testClusterCRUD(ctx context.Context, client *http.Client, cfg *config.Config, connection string, vpcCfg config.VPCTestConfig, st *cspTestState) model.ResourceResult {
	start := time.Now()
	rr := model.ResourceResult{
		Kind:     "cluster",
		TestedAt: start,
	}
	apiBase := spiderAPIBase(cfg)

	doReq := func(method, url string, bodyBytes []byte) ([]byte, int, error) {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, 0, err
		}
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.SetBasicAuth(cfg.Spider.Username, cfg.Spider.Password)
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return b, resp.StatusCode, nil
	}

	const maxAttempts = 120 // 120 × 30 s = 1 hour
	const pollInterval = 30 * time.Second

	clusterTest := st.clusterTest
	ngName := st.clusterName // NodeGroup reuses the "spider-watch" prefix
	ngType := clusterTest.NodeGroupType
	if ngType == "" {
		ngType = "type1" // default: add nodegroup after cluster is Active
	}

	// orDef returns v if non-empty, otherwise def.
	orDef := func(v, def string) string {
		if v == "" {
			return def
		}
		return v
	}

	// nodeGroupInfo is used both in create (Type-II) and add-nodegroup (Type-I).
	type nodeGroupInfo struct {
		Name            string `json:"Name"`
		ImageName       string `json:"ImageName,omitempty"`
		VMSpecName      string `json:"VMSpecName,omitempty"`
		RootDiskType    string `json:"RootDiskType,omitempty"`
		RootDiskSize    string `json:"RootDiskSize,omitempty"`
		KeyPairName     string `json:"KeyPairName"`
		OnAutoScaling   string `json:"OnAutoScaling"`
		DesiredNodeSize string `json:"DesiredNodeSize"`
		MinNodeSize     string `json:"MinNodeSize"`
		MaxNodeSize     string `json:"MaxNodeSize"`
	}

	buildNGInfo := func() nodeGroupInfo {
		return nodeGroupInfo{
			Name:            ngName,
			ImageName:       clusterTest.NodeGroupImageName,
			VMSpecName:      clusterTest.NodeGroupVMSpecName,
			RootDiskType:    clusterTest.NodeGroupRootDiskType,
			RootDiskSize:    clusterTest.NodeGroupRootDiskSize,
			KeyPairName:     st.kpName,
			OnAutoScaling:   orDef(clusterTest.NodeGroupOnAutoScaling, "true"),
			DesiredNodeSize: orDef(clusterTest.NodeGroupDesiredNodeSize, "1"),
			MinNodeSize:     orDef(clusterTest.NodeGroupMinNodeSize, "1"),
			MaxNodeSize:     orDef(clusterTest.NodeGroupMaxNodeSize, "3"),
		}
	}

	// waitNGActive polls GET /cluster/{name} until the named NodeGroup is Active.
	waitNGActive := func() model.OperationResult {
		return st.op("wait-ng-active", func() error {
			getURL := fmt.Sprintf("%s/cluster/%s?ConnectionName=%s", apiBase, st.clusterName, connection)
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				body, code, err := doReq(http.MethodGet, getURL, nil)
				if err != nil {
					return err
				}
				if code >= 400 {
					return fmt.Errorf("HTTP %d: %s", code, string(body))
				}
				var resp struct {
					NodeGroupList []struct {
						IId struct {
							NameId string `json:"NameId"`
						} `json:"IId"`
						Status string `json:"Status"`
					} `json:"NodeGroupList"`
				}
				if jerr := json.Unmarshal(body, &resp); jerr != nil {
					return fmt.Errorf("parse cluster response: %v", jerr)
				}
				for _, ng := range resp.NodeGroupList {
					if ng.IId.NameId == ngName {
						log.Infof("runner: nodegroup %s status=%q (attempt %d/%d)", ngName, ng.Status, attempt, maxAttempts)
						if strings.EqualFold(ng.Status, "Active") {
							return nil
						}
						break
					}
				}
				if attempt == maxAttempts {
					return fmt.Errorf("nodegroup %s did not reach Active after %d attempts", ngName, maxAttempts)
				}
				select {
				case <-ctx.Done():
					return fmt.Errorf("context cancelled waiting for nodegroup Active: %w", ctx.Err())
				case <-time.After(pollInterval):
				}
			}
			return nil
		})
	}

	// waitNGGone polls GET /cluster/{name} until the named NodeGroup is absent.
	waitNGGone := func() model.OperationResult {
		return st.op("wait-ng-gone", func() error {
			getURL := fmt.Sprintf("%s/cluster/%s?ConnectionName=%s", apiBase, st.clusterName, connection)
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				body, code, err := doReq(http.MethodGet, getURL, nil)
				if err != nil {
					return err
				}
				if code >= 400 {
					return fmt.Errorf("HTTP %d: %s", code, string(body))
				}
				var resp struct {
					NodeGroupList []struct {
						IId struct {
							NameId string `json:"NameId"`
						} `json:"IId"`
					} `json:"NodeGroupList"`
				}
				if jerr := json.Unmarshal(body, &resp); jerr != nil {
					return fmt.Errorf("parse cluster response: %v", jerr)
				}
				found := false
				for _, ng := range resp.NodeGroupList {
					if ng.IId.NameId == ngName {
						found = true
						break
					}
				}
				log.Infof("runner: waiting nodegroup %s gone, found=%v (attempt %d/%d)", ngName, found, attempt, maxAttempts)
				if !found {
					return nil
				}
				if attempt == maxAttempts {
					return fmt.Errorf("nodegroup %s still present after %d attempts", ngName, maxAttempts)
				}
				select {
				case <-ctx.Done():
					return fmt.Errorf("context cancelled waiting for nodegroup gone: %w", ctx.Err())
				case <-time.After(pollInterval):
				}
			}
			return nil
		})
	}

	// extraSubnetName is the name of the optional second subnet for multi-AZ CSPs.
	extraSubnetName := st.subnetName + "-2"

	// 1. Ensure VPC exists (shared with other tests).
	if !st.vpcCreated {
		vpcOp := st.op("vpc-create", func() error {
			body := vpcCreateBody{
				ConnectionName: connection,
				ReqInfo: vpcReqInfo{
					Name:     st.vpcName,
					IPv4CIDR: vpcCfg.VPCCIDR,
					SubnetInfoList: []subnetInfo{
						{Name: st.subnetName, IPv4CIDR: vpcCfg.SubnetCIDR},
					},
				},
			}
			b, _ := json.Marshal(body)
			errBody, code, err := doReq(http.MethodPost, apiBase+"/vpc", b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			st.vpcCreated = true
			return nil
		})
		rr.Operations = append(rr.Operations, vpcOp)
		if vpcOp.Status != model.ResourceStatusOK {
			goto done
		}
	}

	// 1-b. If an extra subnet is required (e.g. AWS EKS needs ≥2 AZs), add it now.
	if clusterTest.ExtraSubnetCIDR != "" && !st.extraSubnetCreated {
		type subnetReqInfo struct {
			Name     string `json:"Name"`
			Zone     string `json:"Zone,omitempty"`
			IPv4CIDR string `json:"IPv4_CIDR"`
		}
		type subnetAddBody struct {
			ConnectionName string        `json:"ConnectionName"`
			ReqInfo        subnetReqInfo `json:"ReqInfo"`
		}
		subnetOp := st.op("subnet-create", func() error {
			body := subnetAddBody{
				ConnectionName: connection,
				ReqInfo: subnetReqInfo{
					Name:     extraSubnetName,
					Zone:     clusterTest.ExtraSubnetZone,
					IPv4CIDR: clusterTest.ExtraSubnetCIDR,
				},
			}
			b, _ := json.Marshal(body)
			url := fmt.Sprintf("%s/vpc/%s/subnet", apiBase, st.vpcName)
			errBody, code, err := doReq(http.MethodPost, url, b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			st.extraSubnetCreated = true
			return nil
		})
		rr.Operations = append(rr.Operations, subnetOp)
		if subnetOp.Status != model.ResourceStatusOK {
			goto done
		}
	}

	{
		// nginx-test state: LoadBalancer address (kubeconfig path lives in st).
		var nginxLBAddr string
		defer func() {
			if st.kubeconfigPath != "" {
				_ = os.Remove(st.kubeconfigPath)
				st.kubeconfigPath = ""
			}
		}()

		version := clusterTest.Version

		// Build the NodeGroupList for the create request.
		// Type-I (node_group_type=type1): empty list — NodeGroup is added after cluster is Active.
		// Type-II (node_group_type=type2): include NodeGroup in create.
		type createReqInfo struct {
			Name               string          `json:"Name"`
			Version            string          `json:"Version,omitempty"`
			VPCName            string          `json:"VPCName"`
			SubnetNames        []string        `json:"SubnetNames"`
			SecurityGroupNames []string        `json:"SecurityGroupNames"`
			NodeGroupList      []nodeGroupInfo `json:"NodeGroupList"`
		}
		type createBody struct {
			ConnectionName string        `json:"ConnectionName"`
			ReqInfo        createReqInfo `json:"ReqInfo"`
		}

		var ngList []nodeGroupInfo
		if ngType == "type2" {
			ngList = []nodeGroupInfo{buildNGInfo()}
		} else {
			ngList = []nodeGroupInfo{}
		}

		// 2. Resolve actual subnet names from CB-Spider's VPC registry.
		// We GET the VPC rather than hard-coding "spider-watch" because CB-Spider
		// maps CSP-specific subnet IDs internally; using the name returned by the
		// registry avoids a "At least one Subnet must be specified" validation error.
		var subnetNames []string
		vpcGetURL := fmt.Sprintf("%s/vpc/%s?ConnectionName=%s", apiBase, st.vpcName, connection)
		vpcBody, vpcCode, vpcErr := doReq(http.MethodGet, vpcGetURL, nil)
		if vpcErr != nil {
			rr.Operations = append(rr.Operations, model.OperationResult{
				Op:     "create",
				Status: model.ResourceStatusFail,
				Error:  fmt.Sprintf("GET VPC for subnet list: %v", vpcErr),
			})
			goto done
		}
		if vpcCode >= 400 {
			rr.Operations = append(rr.Operations, model.OperationResult{
				Op:     "create",
				Status: model.ResourceStatusFail,
				Error:  fmt.Sprintf("GET VPC for subnet list: HTTP %d: %s", vpcCode, string(vpcBody)),
			})
			goto done
		}
		{
			var vpcResp struct {
				SubnetInfoList []struct {
					IId struct {
						NameId string `json:"NameId"`
					} `json:"IId"`
				} `json:"SubnetInfoList"`
			}
			if err := json.Unmarshal(vpcBody, &vpcResp); err != nil {
				rr.Operations = append(rr.Operations, model.OperationResult{
					Op:     "create",
					Status: model.ResourceStatusFail,
					Error:  fmt.Sprintf("parse VPC response for subnet list: %v", err),
				})
				goto done
			}
			for _, s := range vpcResp.SubnetInfoList {
				if s.IId.NameId != "" {
					subnetNames = append(subnetNames, s.IId.NameId)
				}
			}
			if len(subnetNames) == 0 {
				rr.Operations = append(rr.Operations, model.OperationResult{
					Op:     "create",
					Status: model.ResourceStatusFail,
					Error:  "VPC has no subnets registered in CB-Spider",
				})
				goto done
			}
		}

		// 3. CREATE — POST /cluster
		createOp := st.op("create", func() error {
			body := createBody{
				ConnectionName: connection,
				ReqInfo: createReqInfo{
					Name:               st.clusterName,
					Version:            version,
					VPCName:            st.vpcName,
					SubnetNames:        subnetNames,
					SecurityGroupNames: []string{st.sgName},
					NodeGroupList:      ngList,
				},
			}
			b, _ := json.Marshal(body)
			errBody, code, err := doReq(http.MethodPost, apiBase+"/cluster", b)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(errBody))
			}
			st.clusterCreated = true
			return nil
		})
		rr.Operations = append(rr.Operations, createOp)
		if createOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 3. WAIT-ACTIVE — poll until cluster Status == "Active" (up to 1 hour).
		waitOp := st.op("wait-active", func() error {
			getURL := fmt.Sprintf("%s/cluster/%s?ConnectionName=%s", apiBase, st.clusterName, connection)
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				body, code, err := doReq(http.MethodGet, getURL, nil)
				if err != nil {
					return err
				}
				if code >= 400 {
					return fmt.Errorf("HTTP %d: %s", code, string(body))
				}
				var resp struct {
					Status string `json:"Status"`
				}
				if jerr := json.Unmarshal(body, &resp); jerr != nil {
					return fmt.Errorf("parse cluster response: %v", jerr)
				}
				log.Infof("runner: cluster %s status=%q (attempt %d/%d)", st.clusterName, resp.Status, attempt, maxAttempts)
				if strings.EqualFold(resp.Status, "Active") {
					return nil
				}
				if attempt == maxAttempts {
					return fmt.Errorf("cluster %s did not reach Active after %d attempts, last status=%q", st.clusterName, maxAttempts, resp.Status)
				}
				select {
				case <-ctx.Done():
					return fmt.Errorf("context cancelled waiting for cluster Active: %w", ctx.Err())
				case <-time.After(pollInterval):
				}
			}
			return nil
		})
		rr.Operations = append(rr.Operations, waitOp)
		if waitOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 4. Type-I only: ADD-NODEGROUP then wait for it to become Active.
		if ngType != "type2" {
			addNGOp := st.op("add-nodegroup", func() error {
				ngInfo := buildNGInfo()
				addBody := struct {
					ConnectionName string        `json:"ConnectionName"`
					ReqInfo        nodeGroupInfo `json:"ReqInfo"`
				}{
					ConnectionName: connection,
					ReqInfo:        ngInfo,
				}
				b, _ := json.Marshal(addBody)
				url := fmt.Sprintf("%s/cluster/%s/nodegroup", apiBase, st.clusterName)
				errBody, code, err := doReq(http.MethodPost, url, b)
				if err != nil {
					return err
				}
				if code >= 400 {
					return fmt.Errorf("HTTP %d: %s", code, string(errBody))
				}
				return nil
			})
			rr.Operations = append(rr.Operations, addNGOp)
			if addNGOp.Status != model.ResourceStatusOK {
				goto done
			}
		}

		// 5. WAIT-NG-ACTIVE — poll until NodeGroup is Active (both Type-I and Type-II).
		ngActiveOp := waitNGActive()
		rr.Operations = append(rr.Operations, ngActiveOp)
		if ngActiveOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 5a. DEPLOY-NGINX — GET cluster kubeconfig, write to temp file, run kubectl apply.
		//   kubeconfig_type controls the URL and any pre-flight checks:
		//     "static"         — certs embedded directly (Azure, Alibaba, Tencent, IBM, NHN…)
		//     "spider_default" — exec-plugin calling Spider Token API (AWS, GCP, NCP)
		//     "csp_native"     — exec-plugin using CSP tool; fetched with &KubeconfigType=native
		kubeconfigURL := func() string {
			base := fmt.Sprintf("%s/cluster/%s?ConnectionName=%s", apiBase, st.clusterName, connection)
			if clusterTest.KubeconfigType == "csp_native" {
				base += "&KubeconfigType=native"
			}
			return base
		}
		// Pre-flight: for Spider Default warn if credential file is absent.
		if clusterTest.KubeconfigType == "spider_default" {
			credFile := os.ExpandEnv("$HOME/.cb-spider/.spider-credential")
			if _, serr := os.Stat(credFile); serr != nil {
				log.Warnf("runner: kubeconfig_type=spider_default but credential file %s not found; kubectl token refresh will fail", credFile)
			}
		}
		deployNginxOp := st.op("deploy-nginx", func() error {
			getURL := kubeconfigURL()
			// Poll until the CSP provides a non-empty Kubeconfig.
			// Some CSPs (e.g. Alibaba ACK) take extra time to issue credentials
			// even after the cluster reaches Active status.
			const maxKubeconfigAttempts = 20
			const kubeconfigPollInterval = 30 * time.Second
			var kubeconfig string
			for attempt := 1; attempt <= maxKubeconfigAttempts; attempt++ {
				body, code, err := doReq(http.MethodGet, getURL, nil)
				if err != nil {
					return err
				}
				if code >= 400 {
					return fmt.Errorf("HTTP %d: %s", code, string(body))
				}
				var clusterResp struct {
					Kubeconfig string `json:"Kubeconfig"`
					AccessInfo struct {
						Kubeconfig string `json:"Kubeconfig"`
					} `json:"AccessInfo"`
				}
				if jerr := json.Unmarshal(body, &clusterResp); jerr != nil {
					return fmt.Errorf("parse cluster response: %v", jerr)
				}
				// CB-Spider returns Kubeconfig either at top level or under AccessInfo.
				// When the CSP has not yet issued credentials, CB-Spider may return
				// an empty string or the sentinel "Kubeconfig is not ready yet!".
				isKubeconfigReady := func(s string) bool {
					return s != "" && !strings.Contains(s, "Kubeconfig is not ready yet!")
				}
				kubeconfig = clusterResp.Kubeconfig
				if !isKubeconfigReady(kubeconfig) {
					kubeconfig = clusterResp.AccessInfo.Kubeconfig
				}
				if isKubeconfigReady(kubeconfig) {
					// Verify the server URL in the kubeconfig is DNS-resolvable.
					// Some CSPs (e.g. Tencent TKE) initially return a VPC-internal hostname
					// that is not resolvable outside the VPC. Detect this early and retry.
					if host := kubeconfigServerHost(kubeconfig); host != "" {
						if _, rerr := net.LookupHost(host); rerr != nil {
							log.Infof("runner: kubeconfig server host %q not resolvable (attempt %d/%d); retrying in %s", host, attempt, maxKubeconfigAttempts, kubeconfigPollInterval)
							goto retryKubeconfig
						}
					}
					break
				}
				log.Infof("runner: cluster GET returned unready Kubeconfig (attempt %d/%d); retrying in %s", attempt, maxKubeconfigAttempts, kubeconfigPollInterval)
				if attempt == maxKubeconfigAttempts {
					log.Warnf("runner: cluster GET body snippet: %s", string(body))
					return fmt.Errorf("cluster GET returned unready Kubeconfig after %d attempts", maxKubeconfigAttempts)
				}
			retryKubeconfig:
				select {
				case <-ctx.Done():
					return fmt.Errorf("context cancelled waiting for Kubeconfig: %w", ctx.Err())
				case <-time.After(kubeconfigPollInterval):
				}
			}
			f, ferr := os.CreateTemp("", "spiderwatch-kubeconfig-*.yaml")
			if ferr != nil {
				return fmt.Errorf("create temp kubeconfig: %v", ferr)
			}
			st.kubeconfigPath = f.Name()
			if _, werr := f.WriteString(kubeconfig); werr != nil {
				f.Close()
				return fmt.Errorf("write kubeconfig: %v", werr)
			}
			f.Close()
			// Retry kubectl apply: some CSPs (e.g. Tencent TKE) take extra time
			// to open the public API server endpoint even after kubeconfig is issued.
			const maxApplyAttempts = 10
			const applyRetryInterval = 30 * time.Second
			var lastApplyErr error
			for applyAttempt := 1; applyAttempt <= maxApplyAttempts; applyAttempt++ {
				cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", st.kubeconfigPath, "apply", "--validate=false", "-f", "-")
				cmd.Stdin = strings.NewReader(nginxDeployYAML)
				out, kerr := cmd.CombinedOutput()
				if kerr == nil {
					log.Infof("runner: kubectl apply nginx: %s", strings.TrimSpace(string(out)))
					st.nginxDeployed = true
					return nil
				}
				lastApplyErr = fmt.Errorf("kubectl apply: %v — %s", kerr, string(out))
				log.Warnf("runner: kubectl apply failed (attempt %d/%d): %v; retrying in %s", applyAttempt, maxApplyAttempts, lastApplyErr, applyRetryInterval)
				if applyAttempt == maxApplyAttempts {
					break
				}
				select {
				case <-ctx.Done():
					return fmt.Errorf("context cancelled waiting for kubectl apply: %w", ctx.Err())
				case <-time.After(applyRetryInterval):
				}
			}
			return lastApplyErr
		})
		rr.Operations = append(rr.Operations, deployNginxOp)
		if deployNginxOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 5b. NGINX-LB-READY — poll until nginx-service LoadBalancer gets an IP or hostname.
		nginxLBReadyOp := st.op("nginx-lb-ready", func() error {
			const maxAttempts = 40
			const interval = 30 * time.Second
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				for _, jsonPath := range []string{
					`{.status.loadBalancer.ingress[0].ip}`,
					`{.status.loadBalancer.ingress[0].hostname}`,
				} {
					cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", st.kubeconfigPath,
						"get", "svc", "nginx-service", "-o", "jsonpath="+jsonPath)
					out, _ := cmd.Output()
					if addr := strings.TrimSpace(string(out)); addr != "" {
						nginxLBAddr = addr
						log.Infof("runner: nginx-service LoadBalancer addr=%s (attempt %d/%d)", addr, attempt, maxAttempts)
						return nil
					}
				}
				log.Infof("runner: waiting for nginx-service LoadBalancer IP (attempt %d/%d)", attempt, maxAttempts)
				if attempt == maxAttempts {
					return fmt.Errorf("nginx-service did not receive a LoadBalancer IP/hostname after %d attempts", maxAttempts)
				}
				select {
				case <-ctx.Done():
					return fmt.Errorf("context cancelled waiting for nginx LB: %w", ctx.Err())
				case <-time.After(interval):
				}
			}
			return nil
		})
		rr.Operations = append(rr.Operations, nginxLBReadyOp)
		if nginxLBReadyOp.Status != model.ResourceStatusOK {
			goto done
		}

		// 5b-2. NGINX-PODS-READY — wait for all nginx-deployment pods to be Running/Ready.
		// nginx-lb-ready only confirms the LoadBalancer got an IP; the nginx pods may still
		// be pulling images (slow on some CSPs, e.g. Alibaba). This step is non-blocking:
		// a timeout here logs a warning but allows nginx-http-check to be the final verdict,
		// since the LB may start routing once even one pod becomes ready.
		nginxPodsReadyOp := st.op("nginx-pods-ready", func() error {
			cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", st.kubeconfigPath,
				"rollout", "status", "deployment/nginx-deployment", "--timeout=10m")
			out, err := cmd.CombinedOutput()
			if err != nil {
				// Non-fatal: log and let nginx-http-check be the final verdict.
				log.Warnf("runner: nginx rollout status did not complete (proceeding to HTTP check): %v — %s",
					err, string(out))
				return nil
			}
			log.Infof("runner: nginx rollout status: %s", strings.TrimSpace(string(out)))
			return nil
		})
		rr.Operations = append(rr.Operations, nginxPodsReadyOp)

		// 5c. NGINX-HTTP-CHECK — poll until nginx responds with HTTP 2xx.
		nginxHTTPOp := st.op("nginx-http-check", func() error {
			const maxAttempts = 40 // 40 × 30 s = 20 min (IBM LB DNS propagation can take 10–20 min)
			const interval = 30 * time.Second
			target := "http://" + nginxLBAddr + "/"
			httpCli := &http.Client{Timeout: 10 * time.Second}
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				resp, err := httpCli.Get(target)
				if err == nil {
					resp.Body.Close()
					if resp.StatusCode < 400 {
						log.Infof("runner: nginx HTTP check OK status=%d addr=%s (attempt %d/%d)",
							resp.StatusCode, nginxLBAddr, attempt, maxAttempts)
						return nil
					}
					log.Infof("runner: nginx HTTP check status=%d (attempt %d/%d)", resp.StatusCode, attempt, maxAttempts)
				} else {
					log.Infof("runner: nginx HTTP check attempt %d/%d: %v", attempt, maxAttempts, err)
				}
				if attempt == maxAttempts {
					if err != nil {
						return fmt.Errorf("nginx HTTP unreachable after %d attempts: %v", maxAttempts, err)
					}
					return fmt.Errorf("nginx HTTP returned status %d after %d attempts", resp.StatusCode, maxAttempts)
				}
				select {
				case <-ctx.Done():
					return fmt.Errorf("context cancelled waiting for nginx HTTP: %w", ctx.Err())
				case <-time.After(interval):
				}
			}
			return nil
		})
		rr.Operations = append(rr.Operations, nginxHTTPOp)

		// 6. LIST
		listOp := st.op("list", func() error {
			body, code, err := doReq(http.MethodGet, apiBase+"/cluster?ConnectionName="+connection, nil)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(body))
			}
			cnt, err := extractCount("cluster", body)
			if err != nil {
				return fmt.Errorf("parse response: %v", err)
			}
			rr.Count = cnt
			return nil
		})
		rr.Operations = append(rr.Operations, listOp)

		// 7. GET
		getOp := st.op("get", func() error {
			url := fmt.Sprintf("%s/cluster/%s?ConnectionName=%s", apiBase, st.clusterName, connection)
			body, code, err := doReq(http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(body))
			}
			return nil
		})
		rr.Operations = append(rr.Operations, getOp)

		// 8. NGINX-DELETE — remove nginx Deployment+Service while nodegroup nodes are
		//    still running so cloud-controller-manager can process the ELB deletion.
		//    Must run BEFORE remove-nodegroup.
		if st.nginxDeployed {
			nginxDeleteTestOp := st.op("nginx-delete", func() error {
				const maxDeleteAttempts = 10
				const deleteRetryInterval = 30 * time.Second
				// nginxGone checks whether both nginx resources have already been deleted.
				nginxGone := func() bool {
					getCmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", st.kubeconfigPath,
						"get", "deployment/nginx-deployment", "service/nginx-service",
						"--ignore-not-found", "-o", "name")
					getOut, _ := getCmd.Output()
					return strings.TrimSpace(string(getOut)) == ""
				}
				var lastDeleteErr error
				for deleteAttempt := 1; deleteAttempt <= maxDeleteAttempts; deleteAttempt++ {
					cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", st.kubeconfigPath,
						"delete", "-f", "-", "--ignore-not-found=true", "--wait=false")
					cmd.Stdin = strings.NewReader(nginxDeployYAML)
					out, kerr := cmd.CombinedOutput()
					outStr := strings.TrimSpace(string(out))
					if kerr != nil {
						// Connection reset may occur even when deletion succeeded — verify.
						if nginxGone() {
							log.Infof("runner: nginx-delete: resources already gone (delete err: %v)", kerr)
							st.nginxDeployed = false
							return nil
						}
						lastDeleteErr = fmt.Errorf("kubectl delete nginx: %v — %s", kerr, outStr)
						log.Warnf("runner: nginx-delete failed (attempt %d/%d): %v; retrying in %s", deleteAttempt, maxDeleteAttempts, lastDeleteErr, deleteRetryInterval)
						select {
						case <-ctx.Done():
							return fmt.Errorf("context cancelled waiting for nginx-delete: %w", ctx.Err())
						case <-time.After(deleteRetryInterval):
						}
						continue
					}
					log.Infof("runner: nginx-delete: %s", outStr)
					st.nginxDeployed = false
					return nil
				}
				return lastDeleteErr
			})
			rr.Operations = append(rr.Operations, nginxDeleteTestOp)
			if nginxDeleteTestOp.Status != model.ResourceStatusOK {
				goto done
			}

			// wait-lb-gone: poll until nginx-service is absent, then wait an extra
			// buffer for AWS ELB/ENI to be fully released before nodegroup is deleted.
			waitLBGoneTestOp := st.op("wait-lb-gone", func() error {
				const maxAttempts = 24 // 24 × 30 s = 12 min
				const interval = 30 * time.Second
				const extraBuffer = 3 * time.Minute
				for attempt := 1; attempt <= maxAttempts; attempt++ {
					cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", st.kubeconfigPath,
						"get", "svc", "nginx-service", "--ignore-not-found",
						"-o", "jsonpath={.metadata.name}")
					out, kerr := cmd.Output()
					if kerr != nil {
						// kubectl failed — do NOT treat empty stdout as confirmed-gone.
						log.Warnf("runner: wait-lb-gone kubectl error (attempt %d/%d): %v", attempt, maxAttempts, kerr)
					} else if strings.TrimSpace(string(out)) == "" {
						log.Infof("runner: nginx-service confirmed deleted; waiting %s for LB/ENI cleanup", extraBuffer)
						select {
						case <-ctx.Done():
							return fmt.Errorf("context cancelled waiting for LB cleanup: %w", ctx.Err())
						case <-time.After(extraBuffer):
						}
						return nil
					} else {
						log.Infof("runner: wait-lb-gone: nginx-service still present (attempt %d/%d)", attempt, maxAttempts)
					}
					if attempt == maxAttempts {
						return fmt.Errorf("nginx-service still present after %d attempts", maxAttempts)
					}
					select {
					case <-ctx.Done():
						return fmt.Errorf("context cancelled waiting for LB gone: %w", ctx.Err())
					case <-time.After(interval):
					}
				}
				return nil
			})
			rr.Operations = append(rr.Operations, waitLBGoneTestOp)
			if waitLBGoneTestOp.Status != model.ResourceStatusOK {
				goto done
			}
		}

		// 9. REMOVE-NODEGROUP and WAIT-NG-GONE — only for Type-I CSPs (node_group_type=type1).
		// For Type-II CSPs (Azure, GCP, NHN, NCP, IBM), the initial node group is the
		// cluster's system pool and cannot be deleted independently. It is removed
		// automatically when the cluster itself is deleted (cluster-delete in cleanup).
		if ngType != "type2" {
			removeNGOp := st.op("remove-nodegroup", func() error {
				url := fmt.Sprintf("%s/cluster/%s/nodegroup/%s", apiBase, st.clusterName, ngName)
				db := deleteBody{ConnectionName: connection}
				const maxRetries = 20
				const retryInterval = 60 * time.Second
				for attempt := 1; attempt <= maxRetries; attempt++ {
					b, _ := json.Marshal(db)
					errBody, code, err := doReq(http.MethodDelete, url, b)
					if err != nil {
						return err
					}
					if code == http.StatusNotFound || isDoesNotExistBody(errBody) {
						return nil // already gone
					}
					if code < 400 {
						return nil
					}
					log.Warnf("runner: remove-nodegroup attempt %d/%d failed HTTP %d: %s",
						attempt, maxRetries, code, string(errBody))
					if attempt == maxRetries {
						return fmt.Errorf("HTTP %d: %s", code, string(errBody))
					}
					// Before retrying, check if the nodegroup is already in Deleting state.
					// If so, the CSP accepted the request; let wait-ng-gone poll for completion.
					getURL := fmt.Sprintf("%s/cluster/%s?ConnectionName=%s", apiBase, st.clusterName, connection)
					if gBody, gCode, gErr := doReq(http.MethodGet, getURL, nil); gErr == nil && gCode < 400 {
						var clResp struct {
							NodeGroupList []struct {
								IId struct {
									NameId string `json:"NameId"`
								} `json:"IId"`
								Status string `json:"Status"`
							} `json:"NodeGroupList"`
						}
						if json.Unmarshal(gBody, &clResp) == nil {
							for _, ng := range clResp.NodeGroupList {
								if ng.IId.NameId == ngName && strings.EqualFold(ng.Status, "Deleting") {
									log.Infof("runner: remove-nodegroup: nodegroup %s is already in Deleting state; wait-ng-gone will poll for completion", ngName)
									return nil
								}
							}
						}
					}
					select {
					case <-ctx.Done():
						return fmt.Errorf("context cancelled: %w", ctx.Err())
					case <-time.After(retryInterval):
					}
				}
				return nil
			})
			rr.Operations = append(rr.Operations, removeNGOp)
			if removeNGOp.Status != model.ResourceStatusOK {
				goto done
			}

			// WAIT-NG-GONE — poll until NodeGroup is absent from the cluster.
			ngGoneOp := waitNGGone()
			rr.Operations = append(rr.Operations, ngGoneOp)
		}
	}

done:
	rr.Status, rr.Error = opsStatus(rr.Operations)
	rr.DurationMs = time.Since(start).Milliseconds()
	return rr
}

// s3BucketName returns a per-run S3 bucket name with a fixed prefix and a
// 4-digit sequence so consecutive runs never reuse the same name.
// Format: "spider-watch-{seq:04d}"  (e.g. "spider-watch-3847")
// The name is well within the 63-char S3/OSS limit.
func s3BucketName(_ string, seq uint16) string {
	return fmt.Sprintf("spider-watch-%04d", seq)
}

// S3 uses query-param-only ConnectionName (no JSON body for create/delete).
// Accept: application/json makes CB-Spider return JSON instead of XML.
// The bucket is removed in the shared cleanup step.
func (r *Runner) testS3CRUD(ctx context.Context, client *http.Client, cfg *config.Config, connection string, st *cspTestState) model.ResourceResult {
	start := time.Now()
	rr := model.ResourceResult{Kind: "s3", TestedAt: start}
	apiBase := spiderAPIBase(cfg)
	bucketName := st.s3Name

	const objectKey = "test-object.txt"
	uploadContent := []byte("CB-Spider-Watch S3 object upload/download test.\n")

	doReq := func(method, url string, bodyBytes []byte) ([]byte, int, error) {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, 0, err
		}
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		// Accept: application/json forces CB-Spider to return JSON instead of raw S3 XML.
		req.Header.Set("Accept", "application/json")
		req.SetBasicAuth(cfg.Spider.Username, cfg.Spider.Password)
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return b, resp.StatusCode, nil
	}

	// 1. CREATE — PUT /s3/{bucketName}?ConnectionName=xxx (no body)
	createOp := st.op("create", func() error {
		url := fmt.Sprintf("%s/s3/%s?ConnectionName=%s", apiBase, bucketName, connection)
		errBody, code, err := doReq(http.MethodPut, url, nil)
		if err != nil {
			return err
		}
		if code >= 400 {
			return fmt.Errorf("HTTP %d: %s", code, string(errBody))
		}
		st.s3Created = true
		return nil
	})
	rr.Operations = append(rr.Operations, createOp)

	// 2. LIST — GET /s3?ConnectionName=xxx
	listOp := st.op("list", func() error {
		body, code, err := doReq(http.MethodGet, apiBase+"/s3?ConnectionName="+connection, nil)
		if err != nil {
			return err
		}
		if code >= 400 {
			return fmt.Errorf("HTTP %d: %s", code, string(body))
		}
		cnt, err := extractCount("s3", body)
		if err != nil {
			return fmt.Errorf("parse response: %v", err)
		}
		rr.Count = cnt
		return nil
	})
	rr.Operations = append(rr.Operations, listOp)

	// 3. GET — only when create succeeded
	if createOp.Status == model.ResourceStatusOK {
		getOp := st.op("get", func() error {
			url := fmt.Sprintf("%s/s3/%s?ConnectionName=%s", apiBase, bucketName, connection)
			body, code, err := doReq(http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(body))
			}
			return nil
		})
		rr.Operations = append(rr.Operations, getOp)

		// 4. UPLOAD — PUT /s3/{bucketName}/{objectKey} with plain-text body
		uploadOp := st.op("upload", func() error {
			url := fmt.Sprintf("%s/s3/%s/%s?ConnectionName=%s", apiBase, bucketName, objectKey, connection)
			req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(uploadContent))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "text/plain")
			req.Header.Set("Accept", "application/json")
			req.SetBasicAuth(cfg.Spider.Username, cfg.Spider.Password)
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			if resp.StatusCode >= 400 {
				return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
			}
			return nil
		})
		rr.Operations = append(rr.Operations, uploadOp)

		// 5. DOWNLOAD + VERIFY — GET /s3/{bucketName}/{objectKey} and compare bytes.
		// Retries up to 3 times (5 s apart) to tolerate S3 read-after-write lag.
		if uploadOp.Status == model.ResourceStatusOK {
			downloadOp := st.op("download", func() error {
				url := fmt.Sprintf("%s/s3/%s/%s?ConnectionName=%s", apiBase, bucketName, objectKey, connection)
				const maxAttempts = 3
				const retryInterval = 5 * time.Second
				for attempt := 1; attempt <= maxAttempts; attempt++ {
					req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
					if err != nil {
						return err
					}
					req.SetBasicAuth(cfg.Spider.Username, cfg.Spider.Password)
					resp, err := client.Do(req)
					if err != nil {
						return err
					}
					got, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
					resp.Body.Close()
					if resp.StatusCode >= 400 {
						return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(got))
					}
					if readErr != nil {
						return fmt.Errorf("reading download response: %w", readErr)
					}
					if bytes.Equal(got, uploadContent) {
						return nil
					}
					// 0-byte or mismatched — retry in case of S3 read-after-write lag
					log.Warnf("runner: S3 download attempt %d/%d: uploaded %d bytes, got %d bytes; retrying",
						attempt, maxAttempts, len(uploadContent), len(got))
					if attempt == maxAttempts {
						return fmt.Errorf("content mismatch after %d attempts: uploaded %d bytes but downloaded %d bytes",
							maxAttempts, len(uploadContent), len(got))
					}
					select {
					case <-ctx.Done():
						return fmt.Errorf("context cancelled waiting for S3 download: %w", ctx.Err())
					case <-time.After(retryInterval):
					}
				}
				return nil
			})
			if downloadOp.Status == model.ResourceStatusOK {
				downloadOp.Message = fmt.Sprintf("content verified (%d bytes match)", len(uploadContent))
			}
			rr.Operations = append(rr.Operations, downloadOp)
		}
	}

	rr.Status, rr.Error = opsStatus(rr.Operations)
	rr.DurationMs = time.Since(start).Milliseconds()
	return rr
}

// testCleanup deletes all shared resources that were created during the test run,
// in reverse order: myimage → vm → disk → kp → sg → vpc.
// This is always appended as the last resource entry in testCSP.
func (r *Runner) testCleanup(ctx context.Context, client *http.Client, cfg *config.Config, connection string, st *cspTestState) model.ResourceResult {
	start := time.Now()
	rr := model.ResourceResult{
		Kind:     "cleanup",
		TestedAt: start,
	}
	apiBase := spiderAPIBase(cfg)

	doReq := func(method, url string, bodyBytes []byte) ([]byte, int, error) {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, 0, err
		}
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.SetBasicAuth(cfg.Spider.Username, cfg.Spider.Password)
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return b, resp.StatusCode, nil
	}

	// isDependencyError returns true when the error body suggests the resource is
	// still in use or has dependent resources (e.g. "DependencyViolation",
	// "in use", "has dependencies", "still in use", etc.).
	isDependencyError := func(body []byte) bool {
		lower := strings.ToLower(string(body))
		for _, kw := range []string{
			"dependencyviolation", "dependency violation",
			"has dependencies", "has dependency",
			"in use", "still in use",
			"attached", "associated",
		} {
			if strings.Contains(lower, kw) {
				return true
			}
		}
		return false
	}

	delWithRetry := func(opName, url string, maxRetries int) model.OperationResult {
		return st.op(opName, func() error {
			db := deleteBody{ConnectionName: connection}
			b, _ := json.Marshal(db)
			const retryInterval = 30 * time.Second
			// depRetryInterval / depMaxAttempts: when a dependency error is detected
			// the resource is likely being deleted by a prior cleanup step.
			// Retry every 30 s for up to 10 minutes (20 attempts) before giving up.
			const depRetryInterval = 30 * time.Second
			const depMaxAttempts = 20
			depAttempt := 0
			for attempt := 1; attempt <= maxRetries; attempt++ {
				body, code, err := doReq(http.MethodDelete, url, b)
				if err != nil {
					return err
				}
				// 404 = resource does not exist; report as skipped, no retry.
				if code == http.StatusNotFound {
					return &skipError{"not found"}
				}
				// Some CSPs return HTTP 500 with a body saying the resource does not
				// exist instead of a proper 404. Treat these as skip immediately.
				if isDoesNotExistBody(body) {
					return &skipError{"not found"}
				}
				// CSP driver does not support this resource type at all — skip cleanup.
				if strings.Contains(strings.ToLower(string(body)), "does not support") {
					return &skipError{strings.TrimSpace(string(body))}
				}
				// 2xx = deleted successfully.
				if code < 400 {
					return nil
				}
				// Dependency / in-use errors: the dependent resource is likely still
				// being deleted in a concurrent or prior cleanup step.
				// Wait up to 10 minutes for it to clear before reporting failure.
				if isDependencyError(body) {
					depAttempt++
					log.Warnf("runner: cleanup %s dependency/in-use error (dep attempt %d/%d): %s",
						opName, depAttempt, depMaxAttempts, string(body))
					if depAttempt < depMaxAttempts {
						select {
						case <-ctx.Done():
							return fmt.Errorf("context cancelled: %w", ctx.Err())
						case <-time.After(depRetryInterval):
						}
						b, _ = json.Marshal(db)
						// Don't consume a normal retry slot — retry the same attempt index.
						attempt--
						continue
					}
					// Exhausted dependency retries → fall through to normal error handling.
				}
				log.Warnf("runner: cleanup %s attempt %d/%d failed HTTP %d: %s",
					opName, attempt, maxRetries, code, string(body))
				if attempt == maxRetries {
					return fmt.Errorf("HTTP %d: %s", code, string(body))
				}
				select {
				case <-ctx.Done():
					return fmt.Errorf("context cancelled: %w", ctx.Err())
				case <-time.After(retryInterval):
				}
				b, _ = json.Marshal(db)
			}
			return nil
		})
	}

	// sweepAll is true when running in cleanup-only mode (cfg.Cleanup == "only") or
	// when no resource types are configured (cfg.Resources is empty). In both cases
	// the user wants to delete ALL leftover resources regardless of what is in the
	// resources list — so every known resource type is attempted (404 = already gone).
	sweepAll := strings.ToLower(strings.TrimSpace(cfg.Cleanup)) == "only" || len(cfg.Resources) == 0

	// inResources reports whether a resource type should be included in this cleanup pass.
	// When sweepAll is true every type is included; otherwise only types that are
	// currently enabled in cfg.Resources (catches leftovers from cleanup:false runs).
	inResources := func(kind string) bool {
		if sweepAll {
			return true
		}
		for _, r := range cfg.Resources {
			if r == kind {
				return true
			}
		}
		return false
	}

	// Reverse order: cluster → nlb → myimage → disk → vm → kp → sg → vpc.
	// Delete a resource only if it was created this run OR its type is enabled in
	// cfg.Resources (to catch leftovers from a previous cleanup:false run).
	// 404 responses are treated as success (resource already gone).
	if st.s3Created || inResources("s3") {
		op := st.op("s3-delete", func() error {
			// Use ?force to delete the bucket and all its objects in one call.
			url := fmt.Sprintf("%s/s3/%s?force&ConnectionName=%s", apiBase, st.s3Name, connection)
			body, code, err := doReq(http.MethodDelete, url, nil)
			if err != nil {
				return err
			}
			// 404 or body saying "does not exist" (some CSPs return 500 for missing S3) → skip.
			if code == http.StatusNotFound || isDoesNotExistBody(body) {
				return &skipError{"not found"}
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(body))
			}
			return nil
		})
		rr.Operations = append(rr.Operations, op)
	}
	if st.clusterCreated || inResources("cluster") {
		// Delete nginx deployment/service first if kubectl apply succeeded.
		// NOTE: st.kubeconfigPath may be "" here (cleared by defer in testClusterCRUD),
		// so always re-fetch the kubeconfig from Spider while the cluster is still alive.
		if st.nginxDeployed {
			// Track a locally-managed kubeconfig temp file for the cleanup phase.
			var cleanupKubeconfigPath string
			kubeconfigTypeForCleanup := st.clusterTest.KubeconfigType
			fetchKubeconfig := func() error {
				base := fmt.Sprintf("%s/cluster/%s?ConnectionName=%s", apiBase, st.clusterName, connection)
				if kubeconfigTypeForCleanup == "csp_native" {
					base += "&KubeconfigType=native"
				}
				body, code, err := doReq(http.MethodGet, base, nil)
				if err != nil {
					return err
				}
				if code == http.StatusNotFound {
					return fmt.Errorf("cluster not found when fetching kubeconfig for nginx cleanup")
				}
				if code >= 400 {
					return fmt.Errorf("HTTP %d: %s", code, string(body))
				}
				var resp struct {
					Kubeconfig string `json:"Kubeconfig"`
					AccessInfo struct {
						Kubeconfig string `json:"Kubeconfig"`
					} `json:"AccessInfo"`
				}
				if jerr := json.Unmarshal(body, &resp); jerr != nil {
					return fmt.Errorf("parse cluster response: %v", jerr)
				}
				kubeconfig := resp.Kubeconfig
				if kubeconfig == "" {
					kubeconfig = resp.AccessInfo.Kubeconfig
				}
				if kubeconfig == "" {
					return fmt.Errorf("cluster returned empty Kubeconfig for nginx cleanup")
				}
				f, ferr := os.CreateTemp("", "spiderwatch-cleanup-kubeconfig-*.yaml")
				if ferr != nil {
					return fmt.Errorf("create temp kubeconfig: %v", ferr)
				}
				cleanupKubeconfigPath = f.Name()
				if _, werr := f.WriteString(kubeconfig); werr != nil {
					f.Close()
					return fmt.Errorf("write kubeconfig: %v", werr)
				}
				f.Close()
				return nil
			}

			nginxDeleteOp := st.op("nginx-delete", func() error {
				if err := fetchKubeconfig(); err != nil {
					return fmt.Errorf("fetch kubeconfig: %v", err)
				}
				const maxDeleteAttempts = 10
				const deleteRetryInterval = 30 * time.Second
				// nginxGone checks whether both nginx resources have already been deleted.
				nginxGone := func() bool {
					getCmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", cleanupKubeconfigPath,
						"get", "deployment/nginx-deployment", "service/nginx-service",
						"--ignore-not-found", "-o", "name")
					getOut, _ := getCmd.Output()
					return strings.TrimSpace(string(getOut)) == ""
				}
				var lastDeleteErr error
				for deleteAttempt := 1; deleteAttempt <= maxDeleteAttempts; deleteAttempt++ {
					cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", cleanupKubeconfigPath,
						"delete", "-f", "-", "--ignore-not-found=true", "--wait=false")
					cmd.Stdin = strings.NewReader(nginxDeployYAML)
					out, kerr := cmd.CombinedOutput()
					outStr := strings.TrimSpace(string(out))
					if kerr != nil {
						// Connection reset may occur even when deletion succeeded — verify.
						if nginxGone() {
							log.Infof("runner: nginx-delete: resources already gone (delete err: %v)", kerr)
							st.nginxDeployed = false
							return nil
						}
						lastDeleteErr = fmt.Errorf("kubectl delete nginx: %v — %s", kerr, outStr)
						log.Warnf("runner: nginx-delete failed (attempt %d/%d): %v; retrying in %s", deleteAttempt, maxDeleteAttempts, lastDeleteErr, deleteRetryInterval)
						select {
						case <-ctx.Done():
							return fmt.Errorf("context cancelled waiting for nginx-delete: %w", ctx.Err())
						case <-time.After(deleteRetryInterval):
						}
						continue
					}
					log.Infof("runner: kubectl delete nginx: %s", outStr)
					st.nginxDeployed = false
					return nil
				}
				return lastDeleteErr
			})
			rr.Operations = append(rr.Operations, nginxDeleteOp)

			// wait-lb-gone: after kubectl delete, the cloud provider (e.g. AWS ELB)
			// may take several minutes to fully release the LoadBalancer and its ENIs.
			// IMPORTANT: this must complete BEFORE ng-delete so that the
			// cloud-controller-manager (running on nodegroup nodes) is still alive
			// to process the ELB deletion signal from K8s.
			// Poll until the nginx-service is confirmed gone, then wait an extra buffer
			// so VPC/subnet deletion does not hit dependency errors from residual ENIs.
			waitLBGoneOp := st.op("wait-lb-gone", func() error {
				defer func() {
					if cleanupKubeconfigPath != "" {
						_ = os.Remove(cleanupKubeconfigPath)
						cleanupKubeconfigPath = ""
					}
				}()
				const maxAttempts = 24 // 24 × 30 s = 12 min
				const interval = 30 * time.Second
				const extraBuffer = 3 * time.Minute // wait for AWS ELB to be fully released
				for attempt := 1; attempt <= maxAttempts; attempt++ {
					cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", cleanupKubeconfigPath,
						"get", "svc", "nginx-service", "--ignore-not-found",
						"-o", "jsonpath={.metadata.name}")
					out, kerr := cmd.Output()
					if kerr != nil {
						// kubectl failed — do NOT treat empty stdout as confirmed-gone.
						log.Warnf("runner: cleanup wait-lb-gone kubectl error (attempt %d/%d): %v", attempt, maxAttempts, kerr)
					} else if strings.TrimSpace(string(out)) == "" {
						// Service is gone — wait an extra buffer for ELB/ENI cleanup.
						log.Infof("runner: nginx-service confirmed deleted; waiting %s for LB/ENI cleanup", extraBuffer)
						select {
						case <-ctx.Done():
							return fmt.Errorf("context cancelled waiting for LB cleanup: %w", ctx.Err())
						case <-time.After(extraBuffer):
						}
						return nil
					} else {
						log.Infof("runner: wait-lb-gone: nginx-service still present (attempt %d/%d)", attempt, maxAttempts)
					}
					if attempt == maxAttempts {
						return fmt.Errorf("nginx-service (LoadBalancer) still present after %d attempts; VPC delete may fail", maxAttempts)
					}
					select {
					case <-ctx.Done():
						return fmt.Errorf("context cancelled waiting for LB gone: %w", ctx.Err())
					case <-time.After(interval):
					}
				}
				return nil
			})
			rr.Operations = append(rr.Operations, waitLBGoneOp)
		}

		// For Type-I CSPs (AWS EKS etc.) the cluster cannot be deleted while
		// NodeGroups are attached. Remove all NodeGroups first by polling GET
		// /cluster/{name} for the current list, then DELETE each one, then
		// wait until none remain before issuing the cluster DELETE.
		removeNGCleanupOp := st.op("ng-delete", func() error {
			getURL := fmt.Sprintf("%s/cluster/%s?ConnectionName=%s", apiBase, st.clusterName, connection)
			body, code, err := doReq(http.MethodGet, getURL, nil)
			if err != nil {
				return err
			}
			if code == http.StatusNotFound || isDoesNotExistBody(body) {
				return nil // cluster already gone
			}
			if code >= 400 {
				return fmt.Errorf("HTTP %d: %s", code, string(body))
			}
			var clusterState struct {
				NodeGroupList []struct {
					IId struct {
						NameId string `json:"NameId"`
					} `json:"IId"`
				} `json:"NodeGroupList"`
				Kubeconfig string `json:"Kubeconfig"`
				AccessInfo struct {
					Kubeconfig string `json:"Kubeconfig"`
				} `json:"AccessInfo"`
			}
			if jerr := json.Unmarshal(body, &clusterState); jerr != nil {
				return fmt.Errorf("parse cluster response: %v", jerr)
			}

			// Before touching nodegroups, ensure nginx workloads are removed.
			// This handles cleanup-only runs where st.nginxDeployed was not set.
			kubeconfig := clusterState.Kubeconfig
			if kubeconfig == "" {
				kubeconfig = clusterState.AccessInfo.Kubeconfig
			}
			if kubeconfig != "" {
				ngKubeconfigFile, ferr := os.CreateTemp("", "spiderwatch-ng-delete-kubeconfig-*.yaml")
				if ferr == nil {
					ngKubeconfigPath := ngKubeconfigFile.Name()
					defer func() {
						_ = os.Remove(ngKubeconfigPath)
					}()
					_, _ = ngKubeconfigFile.WriteString(kubeconfig)
					ngKubeconfigFile.Close()

					// Check if nginx-service is still present.
					checkCmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", ngKubeconfigPath,
						"get", "svc", "nginx-service", "--ignore-not-found",
						"-o", "jsonpath={.metadata.name}")
					checkOut, checkErr := checkCmd.Output()
					if checkErr == nil && strings.TrimSpace(string(checkOut)) != "" {
						log.Infof("runner: ng-delete: nginx-service still present — deleting nginx before nodegroup removal")
						delCmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", ngKubeconfigPath,
							"delete", "-f", "-", "--ignore-not-found=true")
						delCmd.Stdin = strings.NewReader(nginxDeployYAML)
						delOut, delErr := delCmd.CombinedOutput()
						delOutStr := strings.TrimSpace(string(delOut))
						if delErr != nil {
							log.Warnf("runner: ng-delete: kubectl delete nginx failed: %v — %s", delErr, delOutStr)
						} else {
							log.Infof("runner: ng-delete: kubectl delete nginx: %s", delOutStr)
							// Wait for nginx-service LB to be fully released before removing nodegroups.
							const lbMaxAttempts = 24
							const lbInterval = 30 * time.Second
							const lbExtraBuffer = 3 * time.Minute
							for attempt := 1; attempt <= lbMaxAttempts; attempt++ {
								pollCmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", ngKubeconfigPath,
									"get", "svc", "nginx-service", "--ignore-not-found",
									"-o", "jsonpath={.metadata.name}")
								pollOut, pollErr := pollCmd.Output()
								if pollErr != nil {
									log.Warnf("runner: ng-delete wait-lb-gone kubectl error (attempt %d/%d): %v", attempt, lbMaxAttempts, pollErr)
								} else if strings.TrimSpace(string(pollOut)) == "" {
									log.Infof("runner: ng-delete: nginx-service gone; waiting %s for LB/ENI release", lbExtraBuffer)
									select {
									case <-ctx.Done():
										log.Warnf("runner: ng-delete: context cancelled waiting for LB; proceeding with nodegroup delete")
									case <-time.After(lbExtraBuffer):
									}
									break
								} else {
									log.Infof("runner: ng-delete wait-lb-gone: nginx-service still present (attempt %d/%d)", attempt, lbMaxAttempts)
								}
								if attempt == lbMaxAttempts {
									log.Warnf("runner: ng-delete: nginx-service LB still present after %d attempts; proceeding anyway", lbMaxAttempts)
									break
								}
								select {
								case <-ctx.Done():
									log.Warnf("runner: ng-delete: context cancelled waiting for LB; proceeding with nodegroup delete")
									goto deleteNodegroups
								case <-time.After(lbInterval):
								}
							}
						}
					} else if checkErr != nil {
						log.Warnf("runner: ng-delete: could not check nginx-service presence: %v (proceeding)", checkErr)
					}
				} else {
					log.Warnf("runner: ng-delete: could not create temp kubeconfig for nginx pre-check: %v (proceeding)", ferr)
				}
			}

		deleteNodegroups:
			// Type-II CSPs (node_group_type=type2) do not require explicit nodegroup
			// deletion before cluster deletion — the cluster DELETE API removes all node
			// pools automatically (e.g. Azure AKS system pool cannot be deleted independently).
			// Skip the loop; cluster-delete handles it.
			if st.clusterTest.NodeGroupType == "type2" {
				return nil
			}
			if len(clusterState.NodeGroupList) == 0 {
				return nil // no nodegroups to remove
			}
			const ngDelMaxRetries = 20
			const ngDelRetryInterval = 60 * time.Second
			for _, ng := range clusterState.NodeGroupList {
				if ng.IId.NameId == "" {
					log.Warnf("runner: cleanup ng-delete: skipping nodegroup entry with empty NameId")
					continue
				}
				delURL := fmt.Sprintf("%s/cluster/%s/nodegroup/%s", apiBase, st.clusterName, ng.IId.NameId)
				db := deleteBody{ConnectionName: connection}
				deleted := false
				for attempt := 1; attempt <= ngDelMaxRetries; attempt++ {
					b, _ := json.Marshal(db)
					errBody, dcode, derr := doReq(http.MethodDelete, delURL, b)
					if derr != nil {
						return derr
					}
					if dcode == http.StatusNotFound || isDoesNotExistBody(errBody) {
						deleted = true
						break // already gone
					}
					if dcode < 400 {
						deleted = true
						log.Infof("runner: cleanup: deleted nodegroup %s", ng.IId.NameId)
						break
					}
					log.Warnf("runner: cleanup ng-delete %s attempt %d/%d failed HTTP %d: %s",
						ng.IId.NameId, attempt, ngDelMaxRetries, dcode, string(errBody))
					if attempt == ngDelMaxRetries {
						return fmt.Errorf("HTTP %d deleting nodegroup %s: %s", dcode, ng.IId.NameId, string(errBody))
					}
					// Before waiting, check if the nodegroup is already Deleting — if so,
					// the CSP accepted the delete; break and let wait-ng-gone finish the job.
					getURL := fmt.Sprintf("%s/cluster/%s?ConnectionName=%s", apiBase, st.clusterName, connection)
					if gBody, gCode, gErr := doReq(http.MethodGet, getURL, nil); gErr == nil && gCode < 400 {
						var clResp struct {
							NodeGroupList []struct {
								IId struct {
									NameId string `json:"NameId"`
								} `json:"IId"`
								Status string `json:"Status"`
							} `json:"NodeGroupList"`
						}
						if json.Unmarshal(gBody, &clResp) == nil {
							for _, cur := range clResp.NodeGroupList {
								if cur.IId.NameId == ng.IId.NameId && strings.EqualFold(cur.Status, "Deleting") {
									log.Infof("runner: cleanup ng-delete: nodegroup %s is already in Deleting state; letting wait-ng-gone finish", ng.IId.NameId)
									deleted = true
									goto nextNG
								}
							}
						}
					}
					select {
					case <-ctx.Done():
						return fmt.Errorf("context cancelled: %w", ctx.Err())
					case <-time.After(ngDelRetryInterval):
					}
				}
			nextNG:
				_ = deleted
			}
			return nil
		})
		rr.Operations = append(rr.Operations, removeNGCleanupOp)

		// Wait until all NodeGroups are gone before proceeding to cluster delete.
		// Only needed for Type-I CSPs (node_group_type=type1); for Type-II the cluster
		// DELETE removes node pools automatically, so ng-delete already returned nil.
		if st.clusterTest.NodeGroupType != "type2" {
			waitNGGoneCleanupOp := st.op("wait-ng-gone", func() error {
				getURL := fmt.Sprintf("%s/cluster/%s?ConnectionName=%s", apiBase, st.clusterName, connection)
				const maxAttempts = 120
				for attempt := 1; attempt <= maxAttempts; attempt++ {
					body, code, err := doReq(http.MethodGet, getURL, nil)
					if err != nil {
						return err
					}
					if code == http.StatusNotFound || isDoesNotExistBody(body) {
						return nil
					}
					if code >= 400 {
						return fmt.Errorf("HTTP %d: %s", code, string(body))
					}
					var resp struct {
						NodeGroupList []struct {
							IId struct {
								NameId string `json:"NameId"`
							} `json:"IId"`
						} `json:"NodeGroupList"`
					}
					if jerr := json.Unmarshal(body, &resp); jerr != nil {
						return fmt.Errorf("parse cluster response: %v", jerr)
					}
					remaining := len(resp.NodeGroupList)
					log.Infof("runner: cleanup wait-ng-gone: remaining nodegroups=%d (attempt %d/%d)", remaining, attempt, maxAttempts)
					if remaining == 0 {
						return nil
					}
					if attempt == maxAttempts {
						names := make([]string, 0, remaining)
						for _, ng := range resp.NodeGroupList {
							names = append(names, ng.IId.NameId)
						}
						return fmt.Errorf("%d nodegroup(s) still present after %d cleanup attempts: %v", remaining, maxAttempts, names)
					}
					select {
					case <-ctx.Done():
						return fmt.Errorf("context cancelled waiting for nodegroup gone: %w", ctx.Err())
					case <-time.After(30 * time.Second):
					}
				}
				return nil
			})
			rr.Operations = append(rr.Operations, waitNGGoneCleanupOp)
		} // end if !NodeGroupInCreate

		// Use the full 120-attempt (1 hour) polling only when the cluster was
		// actually created this run. In sweep-all mode (cleanup-only / empty resources)
		// the cluster may be a leftover from a previous run — use 30 retries (~15 min).
		// Otherwise (create failed) the resource likely doesn't exist; 3 retries suffice.
		clusterRetries := 120
		if !st.clusterCreated {
			if sweepAll {
				clusterRetries = 30
			} else {
				clusterRetries = 3
			}
		}
		op := delWithRetry("cluster-delete", apiBase+"/cluster/"+st.clusterName, clusterRetries)
		rr.Operations = append(rr.Operations, op)
	}
	if st.nlbCreated || inResources("nlb") {
		op := delWithRetry("nlb-delete", apiBase+"/nlb/"+st.nlbName, 3)
		rr.Operations = append(rr.Operations, op)
	}
	if st.myImgCreated || inResources("myimage") {
		op := delWithRetry("myimage-delete", apiBase+"/myimage/"+st.myImgName, 10)
		rr.Operations = append(rr.Operations, op)
	}
	if st.diskCreated || inResources("disk") {
		op := delWithRetry("disk-delete", apiBase+"/disk/"+st.diskName, 3)
		rr.Operations = append(rr.Operations, op)
	}
	if st.vmCreated || inResources("vm") {
		op := delWithRetry("vm-delete", apiBase+"/vm/"+st.vmName, 3)
		rr.Operations = append(rr.Operations, op)
	}
	if st.kpCreated || inResources("keypair") {
		op := delWithRetry("kp-delete", apiBase+"/keypair/"+st.kpName, 3)
		rr.Operations = append(rr.Operations, op)
	}
	if st.sgCreated || inResources("securitygroup") {
		op := delWithRetry("sg-delete", apiBase+"/securitygroup/"+st.sgName, 3)
		rr.Operations = append(rr.Operations, op)
	}
	if st.extraSubnetCreated {
		extraSubnetName := st.subnetName + "-2"
		subnetURL := fmt.Sprintf("%s/vpc/%s/subnet/%s", apiBase, st.vpcName, extraSubnetName)
		op := delWithRetry("subnet-delete", subnetURL, 3)
		rr.Operations = append(rr.Operations, op)
	}
	if st.vpcCreated || inResources("vpc") {
		op := delWithRetry("vpc-delete", apiBase+"/vpc/"+st.vpcName, 3)
		rr.Operations = append(rr.Operations, op)
	}

	rr.Status, rr.Error = opsStatus(rr.Operations)
	rr.DurationMs = time.Since(start).Milliseconds()
	return rr
}

// callListAPI calls the Spider list endpoint for a resource type and connection.
func callListAPI(ctx context.Context, client *http.Client, cfg *config.Config, connection, resource string) model.ResourceResult {
	rr := model.ResourceResult{
		Kind:     resource,
		TestedAt: time.Now(),
	}
	url := buildListURL(cfg.Spider.APIURL, resource, connection)
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		rr.Status = model.ResourceStatusFail
		rr.Error = fmt.Sprintf("build request: %v", err)
		rr.DurationMs = time.Since(start).Milliseconds()
		return rr
	}
	req.SetBasicAuth(cfg.Spider.Username, cfg.Spider.Password)
	req.Header.Set("X-Connection-Name", connection)
	resp, err := client.Do(req)
	rr.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		rr.Status = model.ResourceStatusFail
		rr.Error = err.Error()
		return rr
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		rr.Status = model.ResourceStatusFail
		rr.Error = fmt.Sprintf("read body: %v", err)
		return rr
	}
	// 501 NotImplemented or "does not support XxxHandler": skip, not a failure.
	if resp.StatusCode == http.StatusNotImplemented {
		rr.Status = model.ResourceStatusSkipped
		rr.Error = fmt.Sprintf("HTTP 501: resource not implemented for this CSP")
		return rr
	}
	if resp.StatusCode >= 400 {
		bodyStr := string(body)
		if strings.Contains(strings.ToLower(bodyStr), "does not support") {
			rr.Status = model.ResourceStatusSkipped
			rr.Error = bodyStr
			return rr
		}
		rr.Status = model.ResourceStatusFail
		rr.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, bodyStr)
		return rr
	}
	count, err := extractCount(resource, body)
	if err != nil {
		rr.Status = model.ResourceStatusFail
		rr.Error = fmt.Sprintf("parse response: %v", err)
		return rr
	}
	rr.Status = model.ResourceStatusOK
	rr.Count = count
	return rr
}

// buildListURL constructs the Spider REST API URL for a given resource and connection.
func buildListURL(apiBase, resource, connection string) string {
	// Spider list endpoints pattern: GET /spider/<resource>?ConnectionName=<conn>
	// Resource-specific overrides:
	switch resource {
	case "vmspec":
		return fmt.Sprintf("%s/vmspec?ConnectionName=%s", apiBase, connection)
	case "image":
		return fmt.Sprintf("%s/vmimage?ConnectionName=%s", apiBase, connection)
	case "myimage":
		return fmt.Sprintf("%s/myimage?ConnectionName=%s", apiBase, connection)
	case "snapshot":
		return fmt.Sprintf("%s/snapshot?ConnectionName=%s", apiBase, connection)
	case "filesystem":
		return fmt.Sprintf("%s/filesystem?ConnectionName=%s", apiBase, connection)
	case "s3":
		return fmt.Sprintf("%s/s3?ConnectionName=%s", apiBase, connection)
	default:
		return fmt.Sprintf("%s/%s?ConnectionName=%s", apiBase, resource, connection)
	}
}

// Spider responses are typically: {"<Kind>": [...]} or a flat array.
// extractCount parses the Spider list response and returns the number of items.
func extractCount(resource string, body []byte) (int, error) {
	// S3 list response (with Accept: application/json): {"Owner":{...},"Buckets":{"Bucket":[...]}}
	if resource == "s3" {
		var s3Resp struct {
			Buckets struct {
				Bucket []json.RawMessage `json:"Bucket"`
			} `json:"Buckets"`
		}
		if err := json.Unmarshal(body, &s3Resp); err != nil {
			return 0, fmt.Errorf("parse s3 response: %v", err)
		}
		return len(s3Resp.Buckets.Bucket), nil
	}

	// Try object with well-known list key
	listKeys := map[string]string{
		"vpc":           "vpc",
		"securitygroup": "securitygroup",
		"keypair":       "keypair",
		"vm":            "vm",
		"vmspec":        "vmspec",
		"image":         "vmimage",
		"nlb":           "nlb",
		"disk":          "disk",
		"myimage":       "myImage",
		"snapshot":      "snapshot",
		"cluster":       "cluster",
		"filesystem":    "filesystem",
	}
	key, ok := listKeys[resource]
	if !ok {
		key = resource
	}

	// Try {"<key>": [...]}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err == nil {
		if raw, ok := envelope[key]; ok {
			var arr []json.RawMessage
			if err := json.Unmarshal(raw, &arr); err == nil {
				return len(arr), nil
			}
			return 0, fmt.Errorf("field %q is not an array", key)
		}
		// Do NOT fall back to arbitrary fields — the expected key must be present.
		return 0, fmt.Errorf("field %q not found in response", key)
	}

	// Try plain array
	var arr []json.RawMessage
	if err := json.Unmarshal(body, &arr); err == nil {
		return len(arr), nil
	}
	return 0, fmt.Errorf("unrecognised response format")
}

// spiderAPIBase returns the effective Spider REST API base URL.
// When external_url is configured, it takes precedence over api_url.
func spiderAPIBase(cfg *config.Config) string {
	if cfg.Spider.ExternalURL != "" {
		return cfg.Spider.ExternalURL
	}
	return cfg.Spider.APIURL
}

// kubeconfigServerHost parses the first "server: https://HOST[:PORT]" line in
// a kubeconfig YAML and returns the hostname (without port). Returns "" if not found.
func kubeconfigServerHost(kubeconfig string) string {
	const prefix = "server: https://"
	for _, line := range strings.Split(kubeconfig, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		host := strings.TrimPrefix(trimmed, prefix)
		// Strip path, query, etc.
		if idx := strings.Index(host, "/"); idx >= 0 {
			host = host[:idx]
		}
		// Strip port (last colon — handles IPv6 as well via brackets).
		if idx := strings.LastIndex(host, ":"); idx > strings.LastIndex(host, "]") {
			host = host[:idx]
		}
		// Strip IPv6 brackets.
		host = strings.Trim(host, "[]")
		// If it's an IP address (v4 or v6), it never needs DNS — always "resolvable".
		if net.ParseIP(host) != nil {
			return ""
		}
		return host
	}
	return ""
}
