// AWS Monitoring end-to-end test.
// Creates VPC -> SG -> KeyPair -> VM (and optionally EKS) using the AWS driver,
// then exercises every MetricType from PR #1697 against both the VM and an EKS
// cluster node, and tears everything down.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awscreds "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"gopkg.in/yaml.v2"

	awsdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	namePrefix  = "cb-mon-test"
	region      = "ap-northeast-2"
	zoneA       = "ap-northeast-2a"
	zoneC       = "ap-northeast-2c"
	vpcCIDR     = "10.42.0.0/16"
	subnetACIDR = "10.42.1.0/24"
	subnetCCIDR = "10.42.2.0/24"
	vmSpec      = "t3.small"
	eksVersion  = "1.30"
	eksNodeSpec = "t3.medium"

	// CloudWatch standard EC2 metrics post at 5-min granularity unless detailed
	// monitoring is on. Wait long enough that GetMetricStatistics returns >=1
	// datapoint for cpu/network. Disk metrics may still be empty on idle VMs.
	metricSettleWait = 7 * time.Minute
)

type yamlConfig struct {
	AWS struct {
		AccessKey string `yaml:"aws_access_key_id"`
		SecretKey string `yaml:"aws_secret_access_key"`
		Region    string `yaml:"region"`
	} `yaml:"aws"`
}

type testCtx struct {
	cred       idrv.CredentialInfo
	regionInfo idrv.RegionInfo
	conn       icon.CloudConnection
	al2AMI     string
	eksAMI     string

	vpcIID     irs.IID
	subnetAIID irs.IID // captured from CreateVPC response (zoneA)
	subnetCIID irs.IID // captured from CreateVPC response (zoneC)
	sgIID      irs.IID
	keyIID     irs.IID
	vmIID      irs.IID

	clusterIID   irs.IID
	nodeGroupIID irs.IID
	nodeIID      irs.IID

	withCluster  bool
	reuse        bool
	skipTeardown bool
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to AWS test config yaml")
	noCluster := flag.Bool("no-cluster", false, "skip EKS cluster creation/test")
	reuse := flag.Bool("reuse", false, "reuse existing resources matching namePrefix (skip preflight cleanup, skip create where one exists)")
	skipTeardown := flag.Bool("skip-teardown", false, "leave resources behind for inspection")
	flag.Parse()

	cfg := loadConfig(*configPath)
	tc := &testCtx{
		cred: idrv.CredentialInfo{
			ClientId:     cfg.AWS.AccessKey,
			ClientSecret: cfg.AWS.SecretKey,
		},
		regionInfo: idrv.RegionInfo{
			Region: cfg.AWS.Region,
			Zone:   zoneA,
		},
		withCluster:  !*noCluster,
		reuse:        *reuse,
		skipTeardown: *skipTeardown,
	}

	log.Printf("[start] region=%s cluster=%v", tc.regionInfo.Region, tc.withCluster)

	mustResolveAMIs(tc)

	driver := &awsdrv.AwsDriver{}
	conn, err := driver.ConnectCloud(idrv.ConnectionInfo{
		CredentialInfo: tc.cred,
		RegionInfo:     tc.regionInfo,
	})
	if err != nil {
		log.Panicf("ConnectCloud: %v", err)
	}
	tc.conn = conn

	defer func() {
		panicked := false
		if r := recover(); r != nil {
			panicked = true
			log.Printf("[panic-recovered] %v", r)
		}
		if tc.skipTeardown {
			log.Printf("[teardown] SKIPPED (--skip-teardown).")
			return
		}
		if tc.reuse && panicked {
			log.Printf("[teardown] SKIPPED in --reuse mode after panic — resources may pre-exist; clean up manually or re-run.")
			return
		}
		teardown(tc)
	}()

	if tc.reuse {
		adoptExisting(tc)
	} else {
		preflightCleanup(tc)
	}
	if tc.vpcIID.SystemId == "" {
		createVPC(tc)
	}
	if tc.sgIID.SystemId == "" {
		createSG(tc)
	}
	if tc.keyIID.SystemId == "" {
		createKeyPair(tc)
	}
	if tc.vmIID.SystemId == "" {
		createVM(tc)
	}

	if tc.withCluster {
		createCluster(tc)
	}

	log.Printf("[wait] %s for CloudWatch metrics to populate...", metricSettleWait)
	time.Sleep(metricSettleWait)

	runVMMetricTests(tc)
	if tc.withCluster {
		runClusterNodeMetricTests(tc)
	}

	log.Printf("[done] tests finished. teardown will proceed via defer.")
}

// ---------------------------------------------------------------------------
// Config + AMI resolution
// ---------------------------------------------------------------------------

func loadConfig(path string) yamlConfig {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Panicf("readConfig %q: %v", path, err)
	}
	var c yamlConfig
	if err := yaml.Unmarshal(data, &c); err != nil {
		log.Panicf("yaml unmarshal: %v", err)
	}
	if c.AWS.AccessKey == "" || c.AWS.SecretKey == "" {
		log.Panicf("config has empty credentials")
	}
	if c.AWS.Region == "" {
		c.AWS.Region = region
	}
	return c
}

func mustResolveAMIs(tc *testCtx) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(tc.regionInfo.Region),
		Credentials: awscreds.NewStaticCredentials(tc.cred.ClientId, tc.cred.ClientSecret, ""),
	})
	if err != nil {
		log.Panicf("aws session: %v", err)
	}
	svc := ssm.New(sess)

	resolve := func(name string) string {
		out, err := svc.GetParameter(&ssm.GetParameterInput{Name: aws.String(name)})
		if err != nil {
			log.Panicf("ssm get %s: %v", name, err)
		}
		return aws.StringValue(out.Parameter.Value)
	}

	tc.al2AMI = resolve("/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-x86_64-gp2")
	tc.eksAMI = resolve(fmt.Sprintf("/aws/service/eks/optimized-ami/%s/amazon-linux-2/recommended/image_id", eksVersion))
	log.Printf("[ami] AL2=%s  EKS-AL2-%s=%s", tc.al2AMI, eksVersion, tc.eksAMI)
}

// ---------------------------------------------------------------------------
// Resource lifecycle
// ---------------------------------------------------------------------------

// adoptExisting fills tc with IIDs of any pre-existing resources whose NameId
// matches our namePrefix conventions. Used in --reuse mode so we don't re-create
// what's already there from a previous interrupted run.
func adoptExisting(tc *testCtx) {
	log.Printf("[reuse] adopting existing resources with prefix %q", namePrefix)
	if h, err := tc.conn.CreateVPCHandler(); err == nil {
		if list, _ := h.ListVPC(); list != nil {
			for _, v := range list {
				if v.IId.NameId == namePrefix+"-vpc" {
					tc.vpcIID = v.IId
					for _, s := range v.SubnetInfoList {
						switch s.Zone {
						case zoneA:
							tc.subnetAIID = s.IId
						case zoneC:
							tc.subnetCIID = s.IId
						}
					}
					log.Printf("[reuse] VPC %s subnetA=%s subnetC=%s",
						v.IId.SystemId, tc.subnetAIID.SystemId, tc.subnetCIID.SystemId)
				}
			}
		}
	}
	if h, err := tc.conn.CreateSecurityHandler(); err == nil {
		if list, _ := h.ListSecurity(); list != nil {
			for _, s := range list {
				if s.IId.NameId == namePrefix+"-sg" {
					tc.sgIID = s.IId
					log.Printf("[reuse] SG %s", s.IId.SystemId)
				}
			}
		}
	}
	if h, err := tc.conn.CreateKeyPairHandler(); err == nil {
		if list, _ := h.ListKey(); list != nil {
			for _, k := range list {
				if k.IId.NameId == namePrefix+"-key" {
					tc.keyIID = k.IId
					log.Printf("[reuse] KeyPair %s", k.IId.SystemId)
				}
			}
		}
	}
	if h, err := tc.conn.CreateVMHandler(); err == nil {
		if list, _ := h.ListVM(); list != nil {
			for _, v := range list {
				if v.IId.NameId != namePrefix+"-vm" {
					continue
				}
				// Skip terminated/shutting-down — the test needs a live VM that
				// can produce CloudWatch metrics during the settle window.
				st, _ := h.GetVMStatus(v.IId)
				stStr := strings.ToLower(string(st))
				if stStr == "terminated" || stStr == "terminating" || stStr == "shutting-down" {
					log.Printf("[reuse] skipping VM %s (status=%s) — will recreate", v.IId.SystemId, stStr)
					continue
				}
				tc.vmIID = v.IId
				log.Printf("[reuse] VM %s status=%s", v.IId.SystemId, stStr)
			}
		}
	}
	if h, err := tc.conn.CreateClusterHandler(); err == nil {
		// Direct GetCluster — ListCluster's per-item GetCluster sometimes errors
		// on CREATING clusters (no nodegroup yet), filtering them out silently.
		ci, err := h.GetCluster(irs.IID{SystemId: namePrefix + "-eks"})
		if err == nil && ci.IId.SystemId != "" {
			tc.clusterIID = ci.IId
			log.Printf("[reuse] Cluster %s status=%s", ci.IId.SystemId, ci.Status)
		}
	}
}

// preflightCleanup deletes any stale resources from a previous failed run that
// share our namePrefix, so the fresh run does not collide on names.
func preflightCleanup(tc *testCtx) {
	log.Printf("[pre ] sweeping stale resources with prefix %q", namePrefix)

	if h, err := tc.conn.CreateClusterHandler(); err == nil {
		if list, err := h.ListCluster(); err == nil {
			for _, c := range list {
				if strings.HasPrefix(c.IId.NameId, namePrefix) {
					log.Printf("[pre ] DeleteCluster %s", c.IId.SystemId)
					_, _ = h.DeleteCluster(c.IId)
				}
			}
		}
	}
	if h, err := tc.conn.CreateVMHandler(); err == nil {
		if list, err := h.ListVM(); err == nil {
			for _, v := range list {
				if strings.HasPrefix(v.IId.NameId, namePrefix) {
					log.Printf("[pre ] TerminateVM %s", v.IId.SystemId)
					_, _ = h.TerminateVM(v.IId)
				}
			}
		}
	}
	if h, err := tc.conn.CreateKeyPairHandler(); err == nil {
		if list, err := h.ListKey(); err == nil {
			for _, k := range list {
				if strings.HasPrefix(k.IId.NameId, namePrefix) {
					log.Printf("[pre ] DeleteKey %s", k.IId.SystemId)
					_, _ = h.DeleteKey(k.IId)
				}
			}
		}
	}
	if h, err := tc.conn.CreateSecurityHandler(); err == nil {
		if list, err := h.ListSecurity(); err == nil {
			for _, s := range list {
				if strings.HasPrefix(s.IId.NameId, namePrefix) {
					log.Printf("[pre ] DeleteSecurity %s", s.IId.SystemId)
					_, _ = h.DeleteSecurity(s.IId)
				}
			}
		}
	}
	if h, err := tc.conn.CreateVPCHandler(); err == nil {
		if list, err := h.ListVPC(); err == nil {
			for _, v := range list {
				if strings.HasPrefix(v.IId.NameId, namePrefix) {
					log.Printf("[pre ] DeleteVPC %s", v.IId.SystemId)
					_, _ = h.DeleteVPC(v.IId)
				}
			}
		}
	}
}

func createVPC(tc *testCtx) {
	h, err := tc.conn.CreateVPCHandler()
	if err != nil {
		log.Panicf("VPCHandler: %v", err)
	}
	req := irs.VPCReqInfo{
		IId:       irs.IID{NameId: namePrefix + "-vpc"},
		IPv4_CIDR: vpcCIDR,
		SubnetInfoList: []irs.SubnetInfo{
			{IId: irs.IID{NameId: namePrefix + "-subnet-a"}, Zone: zoneA, IPv4_CIDR: subnetACIDR},
			{IId: irs.IID{NameId: namePrefix + "-subnet-c"}, Zone: zoneC, IPv4_CIDR: subnetCCIDR},
		},
	}
	info, err := h.CreateVPC(req)
	if err != nil {
		log.Panicf("CreateVPC: %v", err)
	}
	tc.vpcIID = info.IId
	for _, s := range info.SubnetInfoList {
		switch s.Zone {
		case zoneA:
			tc.subnetAIID = s.IId
		case zoneC:
			tc.subnetCIID = s.IId
		}
	}
	log.Printf("[vpc ] created: %s (%s)  subnetA=%s subnetC=%s",
		info.IId.NameId, info.IId.SystemId,
		tc.subnetAIID.SystemId, tc.subnetCIID.SystemId)
	if tc.subnetAIID.SystemId == "" || tc.subnetCIID.SystemId == "" {
		log.Panicf("CreateVPC returned no SubnetInfoList SystemIds: %+v", info.SubnetInfoList)
	}
}

func createSG(tc *testCtx) {
	h, err := tc.conn.CreateSecurityHandler()
	if err != nil {
		log.Panicf("SecurityHandler: %v", err)
	}
	// AWS auto-attaches an outbound allow-all rule to every new SG, so we only
	// add inbound. Adding outbound here would 409 with InvalidPermission.Duplicate.
	rules := []irs.SecurityRuleInfo{
		{Direction: "inbound", IPProtocol: "ALL", FromPort: "-1", ToPort: "-1", CIDR: "0.0.0.0/0"},
	}
	req := irs.SecurityReqInfo{
		IId:           irs.IID{NameId: namePrefix + "-sg"},
		VpcIID:        tc.vpcIID,
		SecurityRules: &rules,
	}
	info, err := h.CreateSecurity(req)
	if err != nil {
		log.Panicf("CreateSecurity: %v", err)
	}
	tc.sgIID = info.IId
	log.Printf("[sg  ] created: %s (%s)", info.IId.NameId, info.IId.SystemId)
}

func createKeyPair(tc *testCtx) {
	h, err := tc.conn.CreateKeyPairHandler()
	if err != nil {
		log.Panicf("KeyPairHandler: %v", err)
	}
	req := irs.KeyPairReqInfo{IId: irs.IID{NameId: namePrefix + "-key"}}
	info, err := h.CreateKey(req)
	if err != nil {
		log.Panicf("CreateKey: %v", err)
	}
	tc.keyIID = info.IId
	log.Printf("[key ] created: %s (%s)", info.IId.NameId, info.IId.SystemId)
}

func createVM(tc *testCtx) {
	h, err := tc.conn.CreateVMHandler()
	if err != nil {
		log.Panicf("VMHandler: %v", err)
	}
	req := irs.VMReqInfo{
		IId:               irs.IID{NameId: namePrefix + "-vm"},
		ImageType:         irs.PublicImage,
		ImageIID:          irs.IID{SystemId: tc.al2AMI},
		VpcIID:            tc.vpcIID,
		SubnetIID:         tc.subnetAIID,
		SecurityGroupIIDs: []irs.IID{tc.sgIID},
		VMSpecName:        vmSpec,
		KeyPairIID:        tc.keyIID,
		RootDiskType:      "default",
		RootDiskSize:      "default",
	}
	info, err := h.StartVM(req)
	if err != nil {
		log.Panicf("StartVM: %v", err)
	}
	tc.vmIID = info.IId
	log.Printf("[vm  ] started: %s (%s)", info.IId.NameId, info.IId.SystemId)

	// Wait until Running.
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		st, err := h.GetVMStatus(tc.vmIID)
		if err == nil {
			if strings.EqualFold(string(st), "running") {
				log.Printf("[vm  ] Running confirmed")
				return
			}
		}
		time.Sleep(15 * time.Second)
	}
	log.Printf("[vm  ] WARN: did not confirm Running within 5m (continuing)")
}

func createCluster(tc *testCtx) {
	h, err := tc.conn.CreateClusterHandler()
	if err != nil {
		log.Panicf("ClusterHandler: %v", err)
	}
	clusterName := namePrefix + "-eks"
	nodeGroupName := namePrefix + "-ng"
	if tc.clusterIID.SystemId == "" {
		tc.clusterIID = irs.IID{NameId: clusterName}
	}
	if tc.nodeGroupIID.NameId == "" {
		tc.nodeGroupIID = irs.IID{NameId: nodeGroupName}
	}

	// AWS driver's CreateCluster is async (returns once the EKS CreateCluster
	// API accepts) and does NOT add NodeGroups. We need to:
	//   1) call CreateCluster (no NodeGroupList — added separately below),
	//   2) poll GetCluster until status=Active,
	//   3) call AddNodeGroup,
	//   4) poll GetCluster until the node has a SystemId.
	t0 := time.Now()
	if tc.clusterIID.SystemId == "" {
		req := irs.ClusterInfo{
			IId:     tc.clusterIID,
			Version: eksVersion,
			Network: irs.NetworkInfo{
				VpcIID:            tc.vpcIID,
				SubnetIIDs:        []irs.IID{tc.subnetAIID, tc.subnetCIID},
				SecurityGroupIIDs: []irs.IID{tc.sgIID},
			},
		}
		log.Printf("[eks ] CreateCluster (cluster-only, async)...")
		info, err := h.CreateCluster(req)
		if err != nil {
			log.Panicf("CreateCluster: %v", err)
		}
		tc.clusterIID = info.IId
		log.Printf("[eks ] CreateCluster returned in %s, status=%s — waiting until Active", time.Since(t0), info.Status)
	} else {
		log.Printf("[eks ] reusing existing cluster %s — waiting until Active", tc.clusterIID.SystemId)
	}

	waitCluster := func(target string, timeout time.Duration) irs.ClusterInfo {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			ci, err := h.GetCluster(tc.clusterIID)
			if err == nil {
				log.Printf("[eks ] cluster status=%s elapsed=%s",
					ci.Status, time.Since(t0).Round(time.Second))
				if strings.EqualFold(string(ci.Status), target) {
					return ci
				}
			} else {
				log.Printf("[eks ] GetCluster err (will retry): %v", err)
			}
			time.Sleep(30 * time.Second)
		}
		log.Panicf("[eks ] timed out waiting for cluster status=%s", target)
		return irs.ClusterInfo{}
	}
	_ = waitCluster("Active", 25*time.Minute)
	log.Printf("[eks ] cluster Active in %s", time.Since(t0).Round(time.Second))

	// Skip AddNodeGroup if the cluster already has it (reuse case).
	hasNG := false
	if ci, err := h.GetCluster(tc.clusterIID); err == nil {
		for _, ng := range ci.NodeGroupList {
			if ng.IId.NameId == tc.nodeGroupIID.NameId || ng.IId.SystemId == tc.nodeGroupIID.NameId {
				tc.nodeGroupIID = ng.IId
				hasNG = true
				log.Printf("[eks ] nodegroup %s already exists (status=%s nodes=%d)",
					ng.IId.NameId, ng.Status, len(ng.Nodes))
				break
			}
		}
	}
	if !hasNG {
		ngReq := irs.NodeGroupInfo{
			IId:             tc.nodeGroupIID,
			ImageIID:        irs.IID{SystemId: tc.eksAMI},
			VMSpecName:      eksNodeSpec,
			RootDiskType:    "default",
			RootDiskSize:    "default",
			KeyPairIID:      tc.keyIID,
			OnAutoScaling:   true,
			DesiredNodeSize: 1,
			MinNodeSize:     1,
			MaxNodeSize:     2,
		}
		log.Printf("[eks ] AddNodeGroup %s...", nodeGroupName)
		tNG := time.Now()
		ngInfo, err := h.AddNodeGroup(tc.clusterIID, ngReq)
		if err != nil {
			log.Panicf("AddNodeGroup: %v", err)
		}
		tc.nodeGroupIID = ngInfo.IId
		log.Printf("[eks ] AddNodeGroup returned in %s status=%s — waiting for node to appear", time.Since(tNG), ngInfo.Status)
	}

	// Poll GetCluster until our node group has a Node SystemId.
	deadline := time.Now().Add(15 * time.Minute)
	for time.Now().Before(deadline) {
		ci, err := h.GetCluster(tc.clusterIID)
		if err == nil {
			for _, ng := range ci.NodeGroupList {
				if ng.IId.NameId != tc.nodeGroupIID.NameId && ng.IId.SystemId != tc.nodeGroupIID.SystemId {
					continue
				}
				if len(ng.Nodes) > 0 && ng.Nodes[0].SystemId != "" {
					tc.nodeGroupIID = ng.IId
					tc.nodeIID = ng.Nodes[0]
					log.Printf("[eks ] node ready ng=%s node=%s ngStatus=%s",
						ng.IId.NameId, ng.Nodes[0].SystemId, ng.Status)
					return
				}
				log.Printf("[eks ] ng=%s status=%s nodes=%d (waiting...)",
					ng.IId.NameId, ng.Status, len(ng.Nodes))
			}
		}
		time.Sleep(30 * time.Second)
	}
	log.Panicf("[eks ] timed out waiting for node SystemId in nodegroup %s", tc.nodeGroupIID.NameId)
}

// ---------------------------------------------------------------------------
// Monitoring tests
// ---------------------------------------------------------------------------

var allMetrics = []irs.MetricType{
	irs.CPUUsage, irs.MemoryUsage,
	irs.DiskRead, irs.DiskWrite, irs.DiskReadOps, irs.DiskWriteOps,
	irs.NetworkIn, irs.NetworkOut,
}

type testResult struct {
	metric  irs.MetricType
	want    string // "data", "reject", or "either"
	got     string // "data", "reject", "error", "empty"
	points  int
	errMsg  string
	pass    bool
	details string
}

func runVMMetricTests(tc *testCtx) {
	h, err := tc.conn.CreateMonitoringHandler()
	if err != nil {
		log.Panicf("MonitoringHandler: %v", err)
	}

	log.Printf("============================================================")
	log.Printf(" VM monitoring tests (vmIID=%s)", tc.vmIID.SystemId)
	log.Printf("============================================================")
	results := []testResult{}
	for _, m := range allMetrics {
		want := "data"
		if m == irs.MemoryUsage {
			want = "reject"
		}
		r := testOne(m, want, func() (irs.MetricData, error) {
			return h.GetVMMetricData(irs.VMMonitoringReqInfo{
				VMIID:      tc.vmIID,
				MetricType: m,
			})
		})
		results = append(results, r)
	}
	printSummary("VM", results)
}

func runClusterNodeMetricTests(tc *testCtx) {
	if tc.nodeIID.SystemId == "" {
		log.Printf("[skip] cluster-node tests: no node resolved")
		return
	}
	h, err := tc.conn.CreateMonitoringHandler()
	if err != nil {
		log.Panicf("MonitoringHandler: %v", err)
	}

	log.Printf("============================================================")
	log.Printf(" Cluster-node monitoring tests (cluster=%s ng=%s node=%s)",
		tc.clusterIID.NameId, tc.nodeGroupIID.NameId, tc.nodeIID.SystemId)
	log.Printf("============================================================")
	results := []testResult{}
	for _, m := range allMetrics {
		want := "data"
		if m == irs.MemoryUsage {
			want = "reject"
		}
		r := testOne(m, want, func() (irs.MetricData, error) {
			return h.GetClusterNodeMetricData(irs.ClusterNodeMonitoringReqInfo{
				ClusterIID:  tc.clusterIID,
				NodeGroupID: tc.nodeGroupIID,
				NodeIID:     tc.nodeIID,
				MetricType:  m,
			})
		})
		results = append(results, r)
	}
	printSummary("ClusterNode", results)
}

func testOne(m irs.MetricType, want string, call func() (irs.MetricData, error)) testResult {
	r := testResult{metric: m, want: want}
	data, err := call()
	if err != nil {
		r.got = "error"
		r.errMsg = err.Error()
		if want == "reject" && strings.Contains(err.Error(), "is not supported") {
			r.got = "reject"
			r.pass = true
		} else {
			r.pass = false
		}
		return r
	}
	if len(data.TimestampValues) == 0 {
		r.got = "empty"
		r.pass = (want == "either")
		return r
	}
	r.got = "data"
	r.points = len(data.TimestampValues)
	r.details = fmt.Sprintf("first=%v last=%v",
		data.TimestampValues[0].Value, data.TimestampValues[len(data.TimestampValues)-1].Value)
	r.pass = (want == "data" || want == "either")
	return r
}

func printSummary(label string, results []testResult) {
	pass, fail := 0, 0
	for _, r := range results {
		mark := "FAIL"
		if r.pass {
			mark = "PASS"
			pass++
		} else {
			fail++
		}
		switch r.got {
		case "data":
			log.Printf(" %s  %-15s  want=%s got=data points=%d %s", mark, r.metric, r.want, r.points, r.details)
		case "reject":
			log.Printf(" %s  %-15s  want=%s got=reject  %q", mark, r.metric, r.want, r.errMsg)
		case "error":
			log.Printf(" %s  %-15s  want=%s got=error   %q", mark, r.metric, r.want, r.errMsg)
		case "empty":
			log.Printf(" %s  %-15s  want=%s got=empty (no datapoints)", mark, r.metric, r.want)
		}
	}
	log.Printf("[%s summary] pass=%d fail=%d", label, pass, fail)
}

// ---------------------------------------------------------------------------
// Teardown
// ---------------------------------------------------------------------------

func teardown(tc *testCtx) {
	log.Printf("============================================================")
	log.Printf(" Teardown")
	log.Printf("============================================================")
	if tc.withCluster && tc.clusterIID.SystemId != "" {
		if h, err := tc.conn.CreateClusterHandler(); err == nil {
			log.Printf("[eks ] DeleteCluster %s", tc.clusterIID.SystemId)
			if _, err := h.DeleteCluster(tc.clusterIID); err != nil {
				log.Printf("[eks ] DeleteCluster err: %v", err)
			}
		}
	}
	if tc.vmIID.SystemId != "" {
		if h, err := tc.conn.CreateVMHandler(); err == nil {
			log.Printf("[vm  ] TerminateVM %s", tc.vmIID.SystemId)
			if _, err := h.TerminateVM(tc.vmIID); err != nil {
				log.Printf("[vm  ] Terminate err: %v", err)
			}
			// Wait for full termination so SG/KeyPair deletion can succeed.
			deadline := time.Now().Add(5 * time.Minute)
			for time.Now().Before(deadline) {
				st, err := h.GetVMStatus(tc.vmIID)
				if err != nil || string(st) == "Terminated" || string(st) == "" {
					break
				}
				time.Sleep(10 * time.Second)
			}
		}
	}
	if tc.keyIID.SystemId != "" {
		if h, err := tc.conn.CreateKeyPairHandler(); err == nil {
			log.Printf("[key ] DeleteKey %s", tc.keyIID.SystemId)
			if _, err := h.DeleteKey(tc.keyIID); err != nil {
				log.Printf("[key ] Delete err: %v", err)
			}
		}
	}
	if tc.sgIID.SystemId != "" {
		if h, err := tc.conn.CreateSecurityHandler(); err == nil {
			log.Printf("[sg  ] DeleteSecurity %s", tc.sgIID.SystemId)
			if _, err := h.DeleteSecurity(tc.sgIID); err != nil {
				log.Printf("[sg  ] Delete err: %v", err)
			}
		}
	}
	if tc.vpcIID.SystemId != "" {
		if h, err := tc.conn.CreateVPCHandler(); err == nil {
			log.Printf("[vpc ] DeleteVPC %s", tc.vpcIID.SystemId)
			if _, err := h.DeleteVPC(tc.vpcIID); err != nil {
				log.Printf("[vpc ] Delete err: %v", err)
			}
		}
	}
	log.Printf("[teardown] done")
	_ = os.Stderr
}
