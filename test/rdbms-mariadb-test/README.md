# CB-Spider RDBMS API Test (MariaDB)

Automated test suite for CB-Spider RDBMS API — creates MariaDB instances across CSPs in parallel, waits until each becomes available, then collects and displays a unified result table.

**Target CSPs: AWS, Alibaba, OpenStack, and NHN (the CSPs that support MariaDB)**
**Unsupported: Azure / GCP / IBM / NCP / Tencent**
  - Tencent: listed as available in the CSP's own guide, but the driver rejects `mariadb` as the requested engine with a "Not Supported" error at API call time, so it is not supported in practice.

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### CSP Connection Configuration

Before running tests, register connection names for each CSP in CB-Spider.

| CSP | Connection Name | Region | Zone |
|-----|----------------|--------|------|
| AWS | `aws-config01` | `ap-southeast-2` | `ap-southeast-2a` |
| Alibaba | `alibaba-beijing-config` | `cn-beijing` | `cn-beijing-f` |
| OpenStack | `openstack-config01` | `RegionOne` | `nova` |
| NHN | `nhn-korea-pangyo1-config` | `KR1` | `kr-pub-a` |

> Azure / GCP / IBM / NCP / Tencent are excluded from this test suite since they don't support MariaDB, so no connection needs to be registered for them.

### Pre-created Network Resources

Each CSP must already have a VPC and subnet created before an RDBMS instance can be created.

```bash
./run-all-csp-network-prepare.sh
```

### Required Tools

- `bash` 3.2+
- `curl`
- `jq`

## MariaDB Engine Support by CSP

| CSP | MariaDB Support | Tag Support | Version | Notes |
|-----|------------|---------|------|------|
| AWS | Yes | Yes | `10.6` | Amazon RDS for MariaDB |
| Alibaba | Yes | Yes | `10.6` | AliCloud RDS for MariaDB |
| OpenStack | Yes | No | `10.4` | Trove for OpenStack |
| NHN | Yes | No | `MARIADB_V101118` | NHN RDS for MariaDB |


## RDBMS Instance Configuration

The instance name used throughout is `cb-spider-mariadb-test`.

| CSP | Engine | Version | Spec | Storage |
|-----|--------|---------|------|---------|
| AWS | mariadb | 10.6 | db.t3.medium | 100GB |
| Alibaba | mariadb | 10.6 | Resolved dynamically from metainfo (e.g. `mariadb.n2.medium.2c`; falls back to `rds.mariadb.s4.large` if the lookup fails) | 20GB |
| OpenStack | mariadb | 10.4 | m1.small | 20GB |
| NHN | mariadb | MARIADB_V101118 | m2.c2m4 | 20GB |

> **NHN note**: NHN Cloud RDS requires a dedicated "DB Security Group" — separate from the VPC Security Group — before external access is possible. `nhn-rdbms-test.sh` sets `"NHNAutoOpenDBSecurityGroup": true`, which auto-creates a fully open (`0.0.0.0/0`) DB Security Group and removes it automatically when the instance is deleted. This is purely a testing convenience; in production, configure a DB Security Group through the NHN console/API that allows only specific CIDRs.

## Usage

### Quick Start (Full Test Suite)

```bash
# Runs the full suite sequentially (network prepare -> StorageType validation -> RDBMS create -> DB management -> Tag -> delete -> network cleanup)
./all_test.sh

# Abort immediately on the first failure
STOP_ON_FAIL=1 ./all_test.sh
```

### Step-by-step

```bash
# 1. Create prerequisite network resources
./run-all-csp-network-prepare.sh

# 2. StorageType validation test
./storage-type-test/run-all-csp-storage-type-tests.sh
./storage-type-test/delete-all-csp-storage-type-rdbms.sh

# 3. Create and verify RDBMS instances
./run-all-csp-rdbms-tests.sh

# 4. Database management test (CreateDB / ListDB / DeleteDB)
./database-test/run-all-csp-database-tests.sh

# 5. Tag management test
./tag-test/run-all-csp-rdbms-tag-tests.sh

# 6. Delete RDBMS instances
./delete-all-csp-rdbms.sh

# 7. Delete network resources
./delete-all-csp-network.sh
```

### Per-CSP Execution

```bash
./aws-rdbms-test.sh
./alibaba-rdbms-test.sh
./openstack-rdbms-test.sh
./nhn-rdbms-test.sh
```

### Configuration

| Environment Variable | Default | Description |
|---------|--------|------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:****` | Basic auth credentials |
| `MAX_WAIT_SEC` | `3600` | Max seconds to wait for an instance to become Available |
| `POLL_INTERVAL` | `30` | Status polling interval (seconds) |
| `STOP_ON_FAIL` | `0` | If `1`, abort on the first failure |
| `VERBOSE` | `0` | If `1`, print detailed per-CSP logs |

## Script Structure

```
rdbms-mariadb-test/
├── all_test.sh                          # Full test suite
├── common-rdbms-test.sh                 # Common: Create -> Poll -> Get Info
├── common-rdbms-delete.sh               # Common: Delete -> Poll until gone
├── common-network-prepare.sh            # Common: VPC/Subnet/SG create
├── common-network-cleanup.sh            # Common: VPC/Subnet/SG delete
├── run-all-csp-rdbms-tests.sh           # Create RDBMS on all CSPs in parallel
├── delete-all-csp-rdbms.sh              # Delete RDBMS on all CSPs in parallel
├── run-all-csp-network-prepare.sh       # Prepare network prerequisites on all CSPs
├── delete-all-csp-network.sh            # Clean up network resources on all CSPs
├── aws-rdbms-test.sh / alibaba-rdbms-test.sh / openstack-rdbms-test.sh / nhn-rdbms-test.sh   # Only the 4 CSPs that support MariaDB
├── database-test/
│   ├── common-database-test.sh
│   ├── run-all-csp-database-tests.sh          # Targets: AWS/Alibaba/OpenStack/NHN
│   └── aws-database-test.sh / alibaba-database-test.sh / openstack-database-test.sh / nhn-database-test.sh
├── storage-type-test/
│   ├── common-storage-type-test.sh
│   ├── run-all-csp-storage-type-tests.sh      # Targets: AWS/Alibaba/OpenStack/NHN
│   ├── delete-all-csp-storage-type-rdbms.sh
│   └── aws-storage-type-test.sh / alibaba-storage-type-test.sh / openstack-storage-type-test.sh / nhn-storage-type-test.sh
└── tag-test/
    ├── common-rdbms-tag-test.sh
    ├── run-all-csp-rdbms-tag-tests.sh          # Targets: AWS/Alibaba (SupportsTag=true AND MariaDB supported)
    └── aws-rdbms-tag-test.sh / alibaba-rdbms-tag-test.sh
```

For the detailed scenario and individual results of each test suite (StorageType validation, database management, tag management), see the README in the corresponding subdirectory:

- [storage-type-test/README.md](storage-type-test/README.md)
- [database-test/README.md](database-test/README.md)
- [tag-test/README.md](tag-test/README.md)

## Test Results

Test Date: 2026-08-18

A full run of `./all_test.sh` passed every step for all 4 CSPs (AWS/Alibaba/OpenStack/NHN).

```
 PASS | Step 1: Network Prepare (VPC/Subnet/SG)
 PASS | Step 2: StorageType Validation Test
 PASS | Step 3: Delete StorageType RDBMS Instances
 PASS | Step 4: RDBMS Instance Create (all CSPs)
 PASS | Step 5: Database Management Test (CRUD inside instance)
 PASS | Step 6: Tag Management Test (SupportsTag=true CSPs)
 PASS | Step 7: RDBMS Instance Delete (all CSPs)
 PASS | Step 8: Network Cleanup (VPC/Subnet/SG)

 Total: 8 PASS, 0 FAIL
```

**RDBMS Instance Create (Step 4) detail:**

```
=================================================================================================================================================================================
                                          RDBMS CREATE & INFO TEST SUMMARY - ALL CSPs (MariaDB)
=================================================================================================================================================================================
CSP          | Status      | Engine   | Version      | Spec                     | Storage                  | Endpoint                                 | PublicAccess | Elapsed
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
AWS          | Available   | mariadb  | 10.6.27      | db.t3.medium             | 100GB|gp2                | cb-spider-mariadb-test-***.ap-southeast-2.rds.amazonaws.com:3306        | true         | 4m40s
ALIBABA      | Available   | mariadb  | 10.6         | mariadb.n2.medium.2c     | 20GB|cloud_essd          | *.*.*.*:3306                             | true         | 3m28s
OPENSTACK    | Available   | mariadb  | 10.4         | m1.small                 | 20GB|NA                  | *.*.*.*:3306                              | true         | 3m58s
NHN          | Available   | mariadb  | MARIADB_V101118 | m2.c2m4                  | 20GB|General SSD         | ***.external.kr1.mariadb.rds.nhncloudservice.com:3306                   | true         | 9m53s
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Failed : 0
```

**RDBMS Instance Delete (Step 7) detail:**

```
============================================================
      RDBMS DELETE SUMMARY - ALL CSPs (MariaDB)
============================================================
CSP          | Result         | Detail               | Elapsed
------------------------------------------------------------
AWS          | DELETED        | ok                   | 3m49s
ALIBABA      | DELETED        | ok                   | 23s
OPENSTACK    | DELETED        | ok                   | 20s
NHN          | DELETED        | ok                   | 18s
------------------------------------------------------------
Failed : 0
```
