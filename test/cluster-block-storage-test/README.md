# CB-Spider Cluster Block Storage Driver (Block-PV/PVC) Test

> **Status: initial draft, not yet a validated test tool**
>
> This suite is an initial version written alongside the Cluster Storage Driver API
> (PR #1892). It has not been run end to end against real CSP clusters yet.
> CSP-specific values (VM specs, images, subnets, StorageClass parameters) and the
> verification logic may not match every CSP, so expect to debug the test scripts
> themselves when you run them. Check `${RESULT_DIR}/<csp>/test.log` before concluding
> that a failure comes from CB-Spider, and send fixes back to this suite.

Automated test suite for CB-Spider's Cluster Storage Driver API (`/cluster/{Name}/storagedriver`). For each of 8 CSPs in parallel, it creates a test cluster, enables the Block CSI driver, checks it end to end through StorageClass → PVC → PV → Pod mount, disables and re-enables the driver, deletes everything, and prints a unified result table.

No CSP console check is needed: CSP disks created by the CSI driver are checked through Spider `GET /alldisk`.

> **Warning**
>
> - Each CSP run creates a real cluster with 2 nodes and a 10Gi disk, which is billed.
>   Running all 8 CSPs in parallel creates 8 clusters at once.
> - Resources named `blk-test-*` (VPC, SG, KeyPair, Cluster) are reused if they already
>   exist, and are deleted at the end of the test.
> - Ctrl+C (or SIGTERM) writes the result and runs cleanup. Press Ctrl+C again to abort cleanup.
>   A CSP disk that could not be released is reported in the notes as `CSP disk left`;
>   delete it manually.

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### CSP Connection Configuration

Before running the tests, register a connection name for each CSP in CB-Spider. The default names are the same as in `test/rdbms-mysql-test`. Override one with `CONNECTION_NAME` when running a single CSP.

| CSP | Connection Name | Zone (subnet) |
|-----|----------------|---------------|
| AWS | `aws-config01` | `ap-southeast-2a`, `ap-southeast-2b` |
| Azure | `azure-koreacentral-config` | |
| GCP | `gcp-iowa-config` | |
| Alibaba | `alibaba-beijing-config` | `cn-beijing-f` |
| Tencent | `tencent-beijing3-config` | `ap-beijing-3` |
| IBM | `ibm-us-east-1-config` | `us-east-1` |
| NCP | `ncp-korea1-config` | `KR-1` |
| NHN | `nhn-korea-pangyo1-config` | `kr-pub-a` |

### Network Resources

No pre-created network is needed. Step 0 creates `blk-test-vpc`, `blk-test-sg` and `blk-test-key`, or reuses them if they already exist. All of them are deleted in cleanup.

The default security group allows all traffic inside the VPC CIDR and TCP 443 from anywhere. Override it with `SG_RULES_JSON`.

### Required Tools and Files

- `bash` 3.2+
- `curl`
- `jq`
- `kubectl`
- `~/.cb-spider/.spider-credential`: the kubeconfig that Spider returns for AWS, GCP and NCP gets its token by calling Spider's `/cluster/{Name}/token` with this file. Format:

  ```bash
  SPIDER_USERNAME=admin
  SPIDER_PASSWORD=<password>
  ```

## Cluster Configuration

Every CSP creates a cluster named `blk-test-<csp>` with one NodeGroup `ng-01` of 2 nodes (`NODE_SIZE`). The K8s version and the node image are the CSP defaults unless `K8S_VERSION` or `NODE_IMAGE` is set.

| CSP | Type | Node VM Spec | Subnet | Disable expected |
|-----|------|--------------|--------|------------------|
| AWS | I | `t3.medium` | 2 subnets in different AZs | ok |
| Azure | II | `Standard_D2s_v3` | `10.0.0.0/22` (AKS needs a large subnet) | ok |
| GCP | II | `e2-standard-2` | 1 | ok (Autopilot: BuiltIn, rejected) |
| Alibaba | I | `ecs.g6.large` | 1 | **reject** (driver shared with File) |
| Tencent | I | **`SET_ME`** (+ `NODE_IMAGE` **`SET_ME`**) | 1 | ok |
| IBM | II | `bx2.2x8` | 1 | ok |
| NCP | II | **`SET_ME`** | 1 | ok (K8s < 1.36: BuiltIn, rejected) |
| NHN | II | **`SET_ME`** | 1 | ok |

- **Type-I**: the NodeGroup is added after the cluster becomes Active. **Type-II**: the NodeGroup is created together with the cluster.
- `SET_ME` values must be given as environment variables. Otherwise the CSP fails at step 0 with `required env ... is not set`.
- A BuiltIn driver is detected from the `ManagementMode` of the List API, so a rejected Disable is counted as expected (➖) without setting `EXPECT_DISABLE`.

## Configuration

Set CB-Spider's connection details as environment variables:

```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:****             # Basic auth (admin:<password>)

# required for NCP, NHN, Tencent
export NODE_VM_SPEC=<spec>                # set per CSP when running them one by one
export NODE_IMAGE=<TKE OS name>           # Tencent only
```

> Change the password portion of `SPIDER_AUTH` to whatever was set when CB-Spider started up.

## How to Run Tests

### All CSPs in Parallel

```bash
./run-all-csp-block-storage-tests.sh              # all 8 CSPs
./run-all-csp-block-storage-tests.sh aws gcp nhn  # selected CSPs
```

- Runs each CSP's test concurrently in the background
- Waits until every CSP finishes, including cleanup (cluster deletion can take tens of minutes)
- Prints a unified Markdown result table and saves it to `${RESULT_DIR}/results.md`

Because `NODE_VM_SPEC` is shared by all CSPs in one run, run NCP, NHN and Tencent one at a time, or edit the default in their scripts.

### Run Individual CSP Test

```bash
./aws-block-storage-test.sh
./azure-block-storage-test.sh
./gcp-block-storage-test.sh
./alibaba-block-storage-test.sh
./ibm-block-storage-test.sh
NODE_VM_SPEC=<spec> ./ncp-block-storage-test.sh
NODE_VM_SPEC=<spec> ./nhn-block-storage-test.sh
NODE_VM_SPEC=<spec> NODE_IMAGE=<os> ./tencent-block-storage-test.sh
```

When you debug, keep the cluster and run again. Step 0 reuses the existing cluster, so later runs skip cluster creation:

```bash
KEEP_CLUSTER=true ./aws-block-storage-test.sh
export KUBECONFIG=/tmp/cluster_block_storage_results/aws/kubeconfig   # inspect with kubectl
./aws-block-storage-test.sh                                            # final run with cleanup
```

## Test Steps

| # | Step | Pass Criteria |
|---|---|---|
| 0 | Test Cluster Create | VPC/SG/KeyPair → Cluster (+NodeGroup) Active → Node Ready |
| 1 | Driver Enable | `PUT /cluster/{Name}/storagedriver` → List `Status: Enabled` |
| 2 | CSI Driver Installed | CSIDriver exists and is registered on CSINode |
| 3 | StorageClass | CSP-provided SC of the driver is used, or one is created |
| 4 | PVC → PV → Bound | PV created, CSP disk exists (Spider ListAllDisk) |
| 5 | Attach/Mount, R/W | write/read, data kept after Pod re-create |
| 6 | PVC Delete | PV deleted, CSP disk deleted (Spider ListAllDisk) |
| 7 | Driver Disable | `Result: true`, `Status: Disabled`, CSI driver gone from CSINode (rejection expected for `EXPECT_DISABLE=reject` or BuiltIn → ➖) |
| 8 | Driver Re-Enable | `Status: Enabled`, CSI driver registered again |

Step dependencies: `0 → 1 → 2 → 3 → 4 → 5 → 6`, and `1 → 7 → 8`. A step whose preceding step failed is marked ⏸. Steps 7 and 8 run even if 2-6 fail. If steps 4-6 fail, the test PVC is deleted before step 7, while the driver can still release the CSP disk.

**Example output** (marks: ✅ pass / ❌ fail / ➖ not applicable / ⏸ not run):

```text
| # | 단계 | 성공 기준 | AWS | NHN |
|---|---|---|---|---|
| 0 | 테스트 Cluster 생성 | VPC/SG/KeyPair → Cluster(+NodeGroup) Active → Node Ready | ✅ | ✅ |
| 1 | Driver Enable | `PUT /cluster/{Name}/storagedriver` → List `Status: Enabled` | ✅ | ✅ |
...
| 8 | Disable 후 재 Enable | `Status: Enabled`, CSI 드라이버 재등록 | ✅ | ❌ |

### 비고
- **AWS**: before enable: Addon/Disabled; driver: aws-ebs-csi-driver v1.37.0-eksbuild.1 (Addon); ...
- **NHN**: ...; S8: Status Enabled, not Enabled within 900s (no-op re-enable?); ...
```

## Script Structure

```text
.
├── run-all-csp-block-storage-tests.sh  # Orchestrator: run CSPs in parallel, print result table
├── common-block-storage-test.sh        # Common: steps 0-8 -> write result -> cleanup
├── aws-block-storage-test.sh           # Per-CSP settings, then exec the common script
├── azure-block-storage-test.sh
├── gcp-block-storage-test.sh
├── alibaba-block-storage-test.sh
├── tencent-block-storage-test.sh
├── ibm-block-storage-test.sh
├── ncp-block-storage-test.sh
└── nhn-block-storage-test.sh
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:****` | Basic auth credentials |
| `RESULT_DIR` | `/tmp/cluster_block_storage_results` | Result and log directory |
| `CONNECTION_NAME` | per CSP | Connection name (single-CSP runs) |
| `NODE_VM_SPEC` / `NODE_IMAGE` | per CSP | Node VM spec / image (`SET_ME` must be set) |
| `NODE_ROOT_DISK_TYPE` / `NODE_ROOT_DISK_SIZE` | CSP default | Node root disk |
| `NODE_SIZE` | `2` | Nodes in the NodeGroup |
| `K8S_VERSION` | CSP default | Cluster version |
| `DRIVER_VERSION` | CSP default | Driver version for step 1 |
| `EXPECT_DISABLE` | per CSP | `reject` for Alibaba. BuiltIn drivers (GKE Autopilot, NCP < 1.36) are detected automatically |
| `SUBNET_JSON` | per CSP | Subnet list for the VPC (`AWS_AZ1`/`AWS_AZ2` for AWS AZs) |
| `SG_RULES_JSON` | VPC CIDR all + TCP 443 | Security group rules |
| `SC_PARAMETERS_JSON` | `{}` | StorageClass parameters when step 3 creates one |
| `TEST_IMAGE` | `busybox:1.36` | Pod image (use a mirror if Docker Hub is blocked) |
| `KEEP_CLUSTER` | (unset) | `true`: skip cleanup, keep cluster and network for debugging |
| `CLUSTER_TIMEOUT` / `DRIVER_TIMEOUT` / `K8S_TIMEOUT` | `3600` / `900` / `600` | Wait limits (seconds) |

## Logs & Results

```text
${RESULT_DIR}/results.md            # unified result table (orchestrator)
${RESULT_DIR}/result_<csp>.txt      # pipe-separated result line: CSP|mark0|...|mark8|notes
${RESULT_DIR}/<csp>/test.log        # per-CSP log, including every Spider request/response
${RESULT_DIR}/<csp>/kubeconfig      # kubeconfig of the test cluster
```

To monitor a run in progress:

```bash
tail -f /tmp/cluster_block_storage_results/aws/test.log
```

## CSP-Specific Notes

| CSP | Note |
|-----|------|
| AWS | EKS needs 2 or more subnets in different AZs. Default AZs are `ap-southeast-2a`/`ap-southeast-2b`; override with `AWS_AZ1`/`AWS_AZ2` to match the connection's region |
| Azure | AKS needs a large subnet (`/23` or larger recommended) |
| GCP | Standard cluster. On Autopilot the driver is BuiltIn and Disable is rejected |
| Alibaba | Disable is rejected by design: `csi-plugin`/`csi-provisioner` are shared with File storage |
| Tencent | `NODE_IMAGE` must be an OS name from the TKE list |
| NCP | Clusters below K8s 1.36 report the driver as BuiltIn. Set the subnets NKS requires in `SUBNET_JSON` if the default is not enough |
| NHN | NKS provides no StorageClass, so step 3 creates one. Step 8 checks the re-enable after `DELETE_COMPLETE` |

## Test Results

Not run on real CSP clusters yet.
