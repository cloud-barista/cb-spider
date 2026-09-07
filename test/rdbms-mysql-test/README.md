# CB-Spider RDBMS API Test

Automated test suite for CB-Spider's RDBMS API — creates MySQL instances across 9 CSPs in parallel, waits until each becomes available, then collects and displays a unified result table.

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### CSP Connection Configuration

Before running the tests, register a connection name for each CSP in CB-Spider.

| CSP | Connection Name | Region | Zone |
|-----|----------------|--------|------|
| AWS | `aws-config01` | `ap-southeast-2` | `ap-southeast-2a` |
| Azure | `azure-koreacentral-config` | `koreacentral` | `1` |
| GCP | `gcp-iowa-config` | `us-central1` | `us-central1-a` |
| Alibaba | `alibaba-beijing-config` | `cn-beijing` | `cn-beijing-f` |
| Tencent | `tencent-beijing3-config` | `ap-beijing` | `ap-beijing-3` |
| IBM | `ibm-us-east-1-config` | `us-east` | `us-east-1` |
| OpenStack | `openstack-config01` | `RegionOne` | `nova` |
| NCP | `ncp-korea1-config` | `KR` | `KR-1` |
| NHN | `nhn-korea-pangyo1-config` | `KR1` | `kr-pub-a` |

### Pre-created Network Resources

Each CSP needs a VPC and subnet created ahead of time before an RDBMS instance can be created. AWS also needs a security group created ahead of time.

| CSP | VPC | Subnet | Security Group | Notes |
|-----|-----|--------|-----------------|------|
| AWS | `vpc-01` | `subnet-01`, `subnet-02` | `sg-01` | Requires 2 subnets in different AZs (SubnetGroup requirement) |
| Azure | `vpc-01` | `subnet-01` | | Subnet not actually used |
| GCP | `vpc-01` | `subnet-01` | | Subnet not actually used |
| Alibaba | `vpc-01` | `subnet-01` | | |
| Tencent | `vpc-01` | `subnet-01` | | |
| IBM | `vpc-01` | `subnet-01` | | Subnet not actually used |
| OpenStack | `vpc-01` | `subnet-01` | | Subnet not actually used |
| NCP | `vpc-01` | `subnet-01` | | |
| NHN | `vpc-01` | `subnet-01` | | |

The scripts below create/delete the VPC/Subnet (and, for AWS, the security group) for all 9 CSPs in one shot. Both scripts are idempotent — they skip resources that already exist, so it's safe to run them repeatedly.

```bash
# Create pre-requisite resources for all CSPs (in parallel)
./run-all-csp-network-prepare.sh

# Delete pre-requisite resources for all CSPs (in parallel)
# Run this only after the RDBMS instances have been deleted
# (VPC/SG deletion fails while they're still in use)
./delete-all-csp-network.sh
```

To run a single CSP on its own, invoke `<csp>-network-prepare.sh` directly (e.g. `./aws-network-prepare.sh`). AWS's AZs can be overridden with the `AWS_AZ1`/`AWS_AZ2` environment variables (default: `ap-southeast-2a`/`ap-southeast-2b`).

Under the hood, these scripts call the CB-Spider REST API like this:

```bash
# Create VPC (AWS example)
curl -u admin:**** -sX POST http://localhost:1024/spider/vpc \
  -H 'Content-Type: application/json' \
  -d '{
    "ConnectionName": "aws-config01",
    "ReqInfo": {
      "Name": "vpc-01",
      "IPv4_CIDR": "10.0.0.0/16",
      "SubnetInfoList": [
        {"Name": "subnet-01", "IPv4_CIDR": "10.0.1.0/24", "Zone": "<AZ-1>"},
        {"Name": "subnet-02", "IPv4_CIDR": "10.0.2.0/24", "Zone": "<AZ-2>"}
      ]
    }
  }' | jq .

# Create Security Group (AWS example, allows inbound MySQL 3306)
curl -u admin:**** -sX POST http://localhost:1024/spider/securitygroup \
  -H 'Content-Type: application/json' \
  -d '{
    "ConnectionName": "aws-config01",
    "ReqInfo": {
      "Name": "sg-01",
      "VPCName": "vpc-01",
      "SecurityRules": [
        {"Direction": "inbound", "IPProtocol": "TCP", "FromPort": "3306", "ToPort": "3306"}
      ]
    }
  }' | jq .
```

### Required Tools

- `bash` 3.2+
- `curl`
- `jq`

## RDBMS Instance Configuration

Every CSP creates a MySQL instance named `cb-spider-mysql-test`.

StorageType is not specified for this test, so each CSP falls back to its own default. The result table's Storage column shows this as `size|type` (e.g. `100GB|gp2`).

| CSP | Engine Version | Instance Spec | Storage | Subnet Required |
|-----|---------------|---------------|---------|-----------------|
| AWS | 8.0 | db.t3.medium | 100GB | Yes (2, in different AZs) |
| Azure | 8.0.21 | Standard_B1ms | 20GB | Not used |
| GCP | 8.0 | db-custom-2-8192 | 20GB | Not used |
| Alibaba | 8.0 | mysql.n4.large.1 | 20GB | Yes |
| Tencent | 8.0 | 8000 (MB) | 50GB | Yes |
| IBM | 8.4 | multitenant | 30GB | Not used |
| OpenStack | 5.7.29 | m1.small | 20GB | Not used |
| NCP | 8.0.36 | SVR.VDBAS.AMD.STAND.C002.M008.NET.SSD.B050.G003 | Managed by CSP | Yes |
| NHN | MYSQL_V8408 | m2.c2m4 | 20GB | Yes |

> **NHN note**: NHN Cloud RDS requires a dedicated "DB Security Group" — separate from the VPC security group — before external access is possible. `nhn-rdbms-test.sh` sets `"NHNAutoOpenDBSecurityGroup": true`, which auto-creates a fully open (`0.0.0.0/0`) DB Security Group and auto-deletes it along with the instance. This is purely a convenience for testing; in production, configure a DB Security Group that allows only specific CIDRs via the NHN console/API.

## Configuration

Before running the tests, set CB-Spider's connection details as environment variables, or edit the defaults directly inside `run-all-csp-rdbms-tests.sh` / `delete-all-csp-rdbms.sh`.

**Option 1) Environment variables**
```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:****             # Basic auth (admin:<password>)
```

**Option 2) Edit the script defaults directly** (`run-all-csp-rdbms-tests.sh`, `delete-all-csp-rdbms.sh`)
```bash
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"   # <-- change the password here
```

> Change the password portion of `SPIDER_AUTH` to whatever was set when CB-Spider started up.

## How to Run Tests

### 0. Prepare Network Prerequisites (VPC/Subnet, AWS Security Group)

Run this before the RDBMS creation tests (only needs to be run once — it's idempotent):

```bash
./run-all-csp-network-prepare.sh
```

### Create: All CSPs in Parallel

```bash
./run-all-csp-rdbms-tests.sh
```

- Creates an RDBMS instance on all 9 CSPs concurrently (background parallel execution)
- Waits for each CSP's instance to reach Available (up to 60 minutes)
- Prints a unified result table when done

**Example output:**
```
CSP          | Status      | Engine   | Version      | Spec                     | Storage                  | Endpoint                                 | PublicAccess | Elapsed
---
AWS          | Available   | mysql    | 8.0.45       | db.t3.medium             | 100GB|gp2                | xxx.rds.amazonaws.com:3306               | true         | 8m39s
AZURE        | Available   | mysql    | 8.0.21       | Standard_B1ms            | 20GB|N/A                 | xxx.mysql.database.azure.com:3306        | true         | 5m35s
GCP          | Available   | mysql    | 8.0          | db-custom-2-8192         | 20GB|PD_SSD              | xxx.cloudsql.google.com:3306             | true         | 3m58s
...
```

### Delete: All CSPs in Parallel

```bash
./delete-all-csp-rdbms.sh
```

- Deletes the RDBMS instance on all 9 CSPs concurrently
- Prints a result table after confirming full deletion

**Example output:**
```
CSP          | Result         | Detail               | Elapsed
---
AWS          | DELETED        | ok                   | 1m48s
AZURE        | DELETED        | ok                   | 33s
GCP          | DELETED        | ok                   | 2m9s
...
```

### Run Individual CSP Test

To run a single CSP on its own:

```bash
# Prepare network prerequisites
./aws-network-prepare.sh
./azure-network-prepare.sh
./gcp-network-prepare.sh
./alibaba-network-prepare.sh
./tencent-network-prepare.sh
./ibm-network-prepare.sh
./openstack-network-prepare.sh
./ncp-network-prepare.sh
./nhn-network-prepare.sh

# Create
./aws-rdbms-test.sh
./azure-rdbms-test.sh
./gcp-rdbms-test.sh
./alibaba-rdbms-test.sh
./tencent-rdbms-test.sh
./ibm-rdbms-test.sh
./openstack-rdbms-test.sh
./ncp-rdbms-test.sh
./nhn-rdbms-test.sh
```

When run individually, results are written to the `RESULT_DIR` you specify, or to the defaults (`/tmp/rdbms_results` for creation, `/tmp/rdbms_network_results` for network prepare/cleanup).

### Full Suite: All Steps in Sequence

```bash
./all_test.sh
```

`all_test.sh` runs the complete lifecycle end to end, in order:

1. Network Prepare — create VPC/Subnet/SG for all CSPs
2. StorageType Test — validate StorageType options per CSP (`storage-type-test/`)
3. Delete StorageType RDBMS instances created by step 2
4. RDBMS Create — create the instance on all 9 CSPs
5. Database Test — validate Database CRUD inside each instance (`database-test/`)
6. Tag Test — validate Tag CRUD for SupportsTag=true CSPs (`tag-test/`)
7. RDBMS Delete — delete the instance on all 9 CSPs
8. Network Cleanup — delete VPC/Subnet/SG for all CSPs

Each step runs as a separate script; a per-step PASS/FAIL summary is printed at the end. By default, a failed step doesn't stop the run — set `STOP_ON_FAIL=1` to abort on the first failure instead.

```bash
STOP_ON_FAIL=1 ./all_test.sh
```

## Script Structure

```
.
├── all_test.sh                     # Orchestrator: full lifecycle, all 8 steps in sequence
├── run-all-csp-network-prepare.sh  # Orchestrator: create VPC/Subnet/SG for all CSPs (parallel)
├── delete-all-csp-network.sh       # Orchestrator: delete VPC/Subnet/SG for all CSPs (parallel)
├── common-network-prepare.sh       # Common: create VPC/Subnet -> (optional) create SG
├── common-network-cleanup.sh       # Common: (optional) delete SG -> delete VPC/Subnet
├── aws-network-prepare.sh
├── azure-network-prepare.sh
├── gcp-network-prepare.sh
├── alibaba-network-prepare.sh
├── tencent-network-prepare.sh
├── ibm-network-prepare.sh
├── openstack-network-prepare.sh
├── ncp-network-prepare.sh
├── nhn-network-prepare.sh
├── run-all-csp-rdbms-tests.sh   # Orchestrator: create on all CSPs (parallel)
├── delete-all-csp-rdbms.sh      # Orchestrator: delete on all CSPs (parallel)
├── common-rdbms-test.sh         # Common: Create -> Poll Available -> Get Info
├── common-rdbms-delete.sh       # Common: Verify -> Delete -> Poll Removed
├── aws-rdbms-test.sh
├── azure-rdbms-test.sh
├── gcp-rdbms-test.sh
├── alibaba-rdbms-test.sh
├── tencent-rdbms-test.sh
├── ibm-rdbms-test.sh
├── openstack-rdbms-test.sh
├── ncp-rdbms-test.sh
├── nhn-rdbms-test.sh
├── database-test/               # Database CRUD test suite (see database-test/README.md)
├── storage-type-test/           # StorageType validation test suite (see storage-type-test/README.md)
└── tag-test/                    # Tag CRUD test suite (see tag-test/README.md)
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:****` | Basic auth credentials |
| `MAX_WAIT_SEC` | `3600` (create) / `1800` (delete) | Timeout per CSP (seconds) |
| `POLL_INTERVAL` | `30` (create) / `15` (delete) | Polling interval (seconds) |
| `VERBOSE` | `0` | Set to `1` for a per-CSP full log dump |
| `AWS_AZ1` / `AWS_AZ2` | `ap-southeast-2a` / `ap-southeast-2b` | AZs used for AWS `subnet-01`/`subnet-02` in `aws-network-prepare.sh` |
| `STOP_ON_FAIL` | `0` | (`all_test.sh` only) Set to `1` to abort the full suite on the first failed step |

```bash
# Example: custom Spider URL
SPIDER_URL=http://10.0.0.1:1024 ./run-all-csp-rdbms-tests.sh

# Example: verbose output
VERBOSE=1 ./run-all-csp-rdbms-tests.sh
```

## Logs & Results

Each run writes logs and result files to a PID-scoped temporary directory.

```
/tmp/rdbms_results_<PID>/result_<csp>.txt   # pipe-separated result line
/tmp/rdbms_logs_<PID>/log_<csp>.txt         # per-CSP full output
```

To monitor a run in progress:

```bash
tail -f /tmp/rdbms_logs_<PID>/log_aws.txt
```

## Related Test Suites

These sibling test suites assume the RDBMS instances created above already exist, and each covers a different subset of the 9 CSPs (only the CSPs where the underlying capability applies):

| Suite | Directory | CSPs Covered | What It Validates |
|-------|-----------|--------------|--------------------|
| Database Management | `database-test/` | All 9 (AWS, Azure, GCP, Alibaba, Tencent, IBM, OpenStack, NCP, NHN) | Database CRUD (Create/List/Delete) inside an instance |
| StorageType | `storage-type-test/` | All 9 (Azure, IBM, NCP auto-SKIP; the other 6 run) | Per-StorageType instance creation and verification |
| Tag Management | `tag-test/` | 6 with `SupportsTag=true` (AWS, Azure, GCP, Alibaba, Tencent, IBM) | Tag CRUD (Add/List/Get/Remove) on an RDBMS resource |

See each subdirectory's README for full details.

## CSP-Specific Notes

| CSP | Note |
|-----|------|
| AWS | SubnetGroup creation requires **2 or more subnets in different AZs**. The `sg-01` security group must be pre-created |
| Tencent | `DBSpec` specifies memory size in MB (e.g. `8000` = 8GB) |
| IBM | StorageType cannot be specified (SupportsStorageTypeSelection=false) |
| NCP | StorageSize/StorageType cannot be specified (managed automatically by the CSP). Only the G3 (KVM) generation is supported. A public domain must be requested separately via the console after creation |

## Test Results

### 2026-08-03

```
=================================================================================================================================================================================
                                             RDBMS CREATE & INFO TEST SUMMARY - ALL CSPs
=================================================================================================================================================================================
CSP          | Status      | Engine   | Version      | Spec                     | Storage                  | Endpoint                                 | PublicAccess | Elapsed
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
AWS          | Available   | mysql    | 8.0.46       | db.t3.medium             | 100GB|gp2                | cb-spider-mysql-test-***.ap-southeast-2.rds.amazonaws.com:3306          | true         | 5m11s
AZURE        | Available   | mysql    | 8.0.21       | Standard_B1ms            | 20GB|Premium_LRS         | cb-spider-mysql-test-***.mysql.database.azure.com:3306                  | true         | 5m38s
GCP          | Available   | mysql    | 8.0          | db-custom-2-8192         | 20GB|PD_SSD              | *.*.*.*:3306                             | true         | 4m0s
ALIBABA      | Available   | mysql    | 8.0          | mysql.n4.large.1         | 20GB|cloud_essd          | *.*.*.*:3306                             | true         | 2m32s
TENCENT      | Available   | mysql    | 8.0          | 8000                     | 50GB|local_ssd           | bj-cdb-***.sql.tencentcdb.com:20137      | true         | 7m48s
IBM          | Available   | mysql    | 8.4          | multitenant              | 30GB|NA                  | ***.databases.appdomain.cloud:32251      | true         | 5m31s
OPENSTACK    | Available   | mysql    | 5.7.29       | m1.small                 | 20GB|NA                  | *.*.*.*:3306                             | true         | 4m31s
NCP          | Available   | mysql    | MYSQL8.0.36  | SVR.VDBAS.AMD.STAND.C002.M008.NET.SSD.B050.G003 | 10GB|SSD                 | db-***.vpc-cdb.ntruss.com:3306           | N/A          | 11m24s
NHN          | Available   | mysql    | MYSQL_V8408  | m2.c2m4                  | 20GB|General SSD         | ***.external.kr1.mysql.rds.nhncloudservice.com:3306                     | true         | 11m22s
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Failed : 0
```
