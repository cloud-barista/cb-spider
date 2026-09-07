# CB-Spider RDBMS StorageType Test

Parallel test suite that creates and verifies RDBMS instances for each CSP's supported StorageType values. For each CSP it dynamically queries the supported StorageType list from the `rdbmsmetainfo` API, creates one instance per option in parallel, and verifies that the returned StorageType matches what was requested.

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### Pre-created Network Resources

Each CSP needs a VPC/Subnet (and, for AWS, a security group) created ahead of time before an RDBMS instance can be created.

```bash
cd ..
./run-all-csp-network-prepare.sh
```

### Required Tools

- `bash` 3.2+
- `curl`
- `jq`

## Test Flow

For each CSP, StorageType validation runs in this order:

1. **FetchStorageTypeOptions** — `GET /spider/rdbmsmetainfo?DBEngine=mysql&ConnectionName=...` — dynamically fetches the list of supported StorageTypes
2. **CreateRDBMS** (one per StorageType, in parallel) — `POST /spider/rdbms` — creates a `cb-mysql-st-<type>` instance for each option
3. **Poll Available** — `GET /spider/rdbms/{Name}` — polls until the instance becomes Available (every 30s by default, up to 3600s)
4. **VerifyStorageType** — confirms the returned StorageType matches what was requested (recorded as `PASS`/`FAIL` in the `Result` field)
5. **(Optional) AutoDelete** — if `AUTO_DELETE=true`, the instance is deleted immediately after verification. With the default (`false`), clean up separately with `delete-all-csp-storage-type-rdbms.sh`

Azure/IBM/NCP have `SupportsStorageTypeSelection=false` — StorageType cannot be specified for them, so their scripts SKIP immediately on execution.

### StorageType Selectability

| CSP | SupportsStorageTypeSelection | Notes |
|-----|------------------------------|------|
| AWS | Yes | gp2, gp3, io1, io2 |
| GCP | Yes | Determined automatically by machine series (not a direct choice) |
| Alibaba | Yes | cloud_auto, cloud_essd, cloud_essd2, cloud_essd3, local_ssd |
| Tencent | Yes | local_ssd, CLOUD_HSSD, CLOUD_SSD, CLOUD_PREMIUM |
| OpenStack | Yes | `__DEFAULT__`, `RBD` |
| NHN | Yes | General HDD, General SSD |
| Azure | No | SKIP |
| IBM | No | SKIP |
| NCP | No | SKIP |

## Configuration

```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:****             # Basic auth (admin:<password>)
```

## How to Run Tests

### All CSPs in Parallel

```bash
./run-all-csp-storage-type-tests.sh
```

- Runs StorageType validation on all 9 CSPs in parallel (Azure/IBM/NCP SKIP automatically)
- Prints a unified result table and a PASS/FAIL/SKIP tally when done

**Example output:**
```
================================================================================================================================
                               RDBMS StorageType Test Summary - All CSPs
================================================================================================================================
CSP          | StorageType(Req)     | StorageType(Ret)   | Result | DB Status      | Elapsed    | Reason
--------------------------------------------------------------------------------------------------------------------------------
AWS          | gp2                  | gp2                | PASS   | Available      | 5m43s      | -
AZURE        | N/A                  | N/A                | SKIP   | NOT_APPLICABLE | -          | SupportsStorageTypeSelection=false
GCP          | PD_SSD               | PD_SSD             | PASS   | Available      | 4m10s      | -
...
--------------------------------------------------------------------------------------------------------------------------------

Total: 22  PASS: 19  FAIL: 0  SKIP: 3
```

### Individual CSP

To run a single CSP on its own:

```bash
./aws-storage-type-test.sh
./azure-storage-type-test.sh          # SKIP (StorageType selection not supported)
./gcp-storage-type-test.sh
./alibaba-storage-type-test.sh
./tencent-storage-type-test.sh
./ibm-storage-type-test.sh            # SKIP (StorageType selection not supported)
./openstack-storage-type-test.sh
./ncp-storage-type-test.sh            # SKIP (StorageType selection not supported)
./nhn-storage-type-test.sh
```

When run individually, results and logs go to the directories you specify via `RESULT_DIR` / `LOG_DIR`, or to the defaults (`/tmp/st_results_<PID>`, `/tmp/st_logs_<PID>`).

### Delete All Instances

```bash
./delete-all-csp-storage-type-rdbms.sh
```

- Re-queries StorageTypeOptions to infer the `cb-mysql-st-<type>` instance names, then deletes them across all CSPs in parallel

## Script Structure

```
storage-type-test/
├── run-all-csp-storage-type-tests.sh    # Orchestrator: full parallel run across CSPs
├── delete-all-csp-storage-type-rdbms.sh # Orchestrator: full parallel delete across CSPs
├── common-storage-type-test.sh          # Common: Create -> Poll Available -> Get Info -> Verify -> (optional) Delete
├── aws-storage-type-test.sh
├── gcp-storage-type-test.sh
├── alibaba-storage-type-test.sh
├── tencent-storage-type-test.sh
├── nhn-storage-type-test.sh
├── openstack-storage-type-test.sh
├── azure-storage-type-test.sh           # SKIP (StorageType selection not supported)
├── ibm-storage-type-test.sh             # SKIP (StorageType selection not supported)
└── ncp-storage-type-test.sh             # SKIP (StorageType selection not supported)
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:****` | Basic auth credentials |
| `MAX_WAIT_SEC` | `3600` (create) / `1800` (delete) | Timeout per instance (seconds) |
| `POLL_INTERVAL` | `30` (create) / `15` (delete) | Polling interval (seconds) |
| `AUTO_DELETE` | `false` | If `true`, delete the instance immediately after verification completes |
| `VERBOSE` | `0` | Set to `1` for a per-CSP full log dump |

```bash
# Example: verbose output
VERBOSE=1 ./run-all-csp-storage-type-tests.sh
```

## Result Format

Each result file (`result_<csp>_<storagetype>.txt`) has 7 pipe-separated fields:

```
CSP|StorageType(Requested)|StorageType(Returned)|Result|DB_Status|Elapsed|Reason
```

| Field | Description |
|------|------|
| `CSP` | CSP name (e.g. AWS) |
| `StorageType(Requested)` | The StorageType value that was requested |
| `StorageType(Returned)` | The StorageType read back after creation (`N/A` means creation failed) |
| `Result` | `PASS` / `FAIL` / `SKIP` |
| `DB_Status` | Instance status (`Available`, `CREATE_ERROR`, `TIMEOUT`, `NOT_APPLICABLE`, etc.) |
| `Elapsed` | Elapsed time |
| `Reason` | Notes (e.g. SupportsStorageTypeSelection=false, cloud_auto auto-selection, etc.) |

## API Reference

| Operation | Method | Path |
|-----------|--------|------|
| FetchStorageTypeOptions | `GET` | `/spider/rdbmsmetainfo?DBEngine=mysql&ConnectionName=` |
| CreateRDBMS | `POST` | `/spider/rdbms` |
| GetRDBMS | `GET` | `/spider/rdbms/{Name}?ConnectionName=` |
| DeleteRDBMS | `DELETE` | `/spider/rdbms/{Name}` |

## Logs & Results

```
/tmp/st_test_<PID>/results/result_<csp>_<storagetype>.txt
/tmp/st_test_<PID>/logs/log_<csp>.txt
```

When running the full delete script:
```
/tmp/st_del_<PID>/results/result_<csp>_<storagetype>.txt
/tmp/st_del_<PID>/logs/log_<csp>.txt
```

To monitor a run in progress:

```bash
tail -f /tmp/st_test_<PID>/logs/log_aws.txt
```

## CSP-Specific Notes

### AWS

| StorageType | DBSpec | StorageSize | Iops |
|-------------|---------------|-------------|------|
| gp2 | db.t3.medium | 100 GB | - |
| gp3 | db.t3.medium | 100 GB | - |
| io1 | db.t3.medium | 100 GB | **3000 (required)** |
| io2 | db.t3.medium | 100 GB | **3000 (required)** |

- StorageTypeOptions: fetched dynamically from the metainfo API
- Connection: `aws-config01`
- Region: `ap-southeast-2` / Zone: `ap-southeast-2a`
- SubnetNames: `subnet-01`, `subnet-02` (different AZs, required for SubnetGroup creation)
- SecurityGroupNames: `sg-01`

**Special notes**
- io1/io2: the `Iops` field is required (Iops is AWS io1/io2-only)
- StorageSize: io1/io2 require a minimum of 100 GB

---

### GCP

GCP's StorageType is fixed by machine series, so 5 combinations are explicitly specified.

| StorageType | DBSpec | StorageSize | Edition | Machine Series |
|-------------|---------------|-------------|---------|---------------|
| PD_SSD | db-perf-optimized-N-4 | 10 GB | Enterprise Plus | N2 |
| PD_SSD | db-custom-2-8192 | 10 GB | Enterprise | Shared/Dedicated core |
| PD_HDD | db-custom-2-8192 | 10 GB | Enterprise | Shared/Dedicated core |
| HYPERDISK_BALANCED | db-c4a-highmem-4 | 20 GB | Enterprise Plus | C4A |
| HYPERDISK_BALANCED | db-custom-N4-2-4096 | 20 GB | Enterprise | N4 |

- Connection: `gcp-iowa-config`
- Region: `us-central1` / Zone: `us-central1-a`
- VPCName: `vpc-01`

**Special notes**
- A mismatch between machine series and StorageType causes the Spider driver to return an error (e.g. requesting HYPERDISK_BALANCED on an N2 machine)
- N2/C4A machine types automatically get Edition=ENTERPRISE_PLUS set (handled internally by the driver)
- For N2 (`db-perf-optimized-N-*`), C4A (`db-c4a-highmem-*`), and N4 (`db-custom-N4-*`): StorageType is not passed to the API — it's determined automatically by the machine series
- GCP's metainfo StorageTypeOptions includes HYPERDISK_BALANCED, but it's selected via machine series rather than direct specification
- HYPERDISK_BALANCED requires a minimum StorageSize of 20 GB

---

### Alibaba

| StorageType | DBSpec | StorageSize | Notes |
|-------------|---------------|-------------|------|
| cloud_auto | mysql.n4.large.1 | 20 GB | |
| cloud_essd | mysql.n4.large.1 | 20 GB | ESSD PL1 |
| cloud_essd2 | mysql.n2.small.2c | **500 GB** | ESSD PL2, minimum 500 GB |
| cloud_essd3 | mysql.n2.small.2c | **1500 GB** | ESSD PL3, minimum 1500 GB |
| local_ssd | rds.mysql.t1.small | 20 GB | Premium Local SSD, only available on the rds.mysql.* family |

- StorageTypeOptions: fetched dynamically from the metainfo API
- Connection: `alibaba-beijing-config`
- Region: `cn-beijing` / Zone: `cn-beijing-f`
- SubnetNames: `subnet-01`

**Special notes**
- cloud_essd2/3 are incompatible with the mysql.n4.* family — mysql.n2.small.2c is used instead
- local_ssd is only available on the rds.mysql.* family (using mysql.n4.* causes an InvalidInstanceLevel.DiskType error)

---

### Tencent

| StorageType | DBSpec | StorageSize |
|-------------|---------------|-------------|
| local_ssd | 8000 (MB) | 50 GB |
| CLOUD_HSSD | 8000 (MB) | 50 GB |
| CLOUD_SSD | 8000 (MB) | 50 GB |
| CLOUD_PREMIUM | 8000 (MB) | 50 GB |

- StorageTypeOptions: fetched dynamically from the metainfo API
- Connection: `tencent-beijing3-config`
- Region: `ap-beijing` / Zone: `ap-beijing-3`
- SubnetNames: `subnet-01`
- DBSpec: specifies memory size in MB (8000 = 8 GB)

**Special notes**
- Concurrent order submission can be rejected (OperationDenied.OtherOderInProcess) — the driver retries automatically

---

### IBM — SKIP

- SupportsStorageTypeSelection=false
- Connection: `ibm-us-east-1-config`
- Region: `us-east` / Zone: `us-east-1`
- DBEngineVersion: `8.4`
- IBM Cloud Databases has no storage type selection feature
- The test script SKIPs immediately on execution

---

### OpenStack

| StorageType | DBSpec | StorageSize |
|-------------|---------------|-------------|
| __DEFAULT__ | m1.small | 20 GB |
| RBD | m1.small | 20 GB |

- Connection: `openstack-config01`
- Region: `RegionOne` / Zone: `nova`
- DBEngineVersion: `5.7.29`
- StorageTypeOptions are fetched dynamically from Cinder's volume type list, so the actual values available depend on how the target OpenStack deployment is configured

---

### NHN

| StorageType | DBSpec | StorageSize |
|-------------|---------------|-------------|
| General HDD | m2.c2m4 | 20 GB |
| General SSD | m2.c2m4 | 20 GB |

- StorageTypeOptions: fetched dynamically from the NHN Cloud RDS API
- Connection: `nhn-korea-pangyo1-config`
- Region: `KR1` / Zone: `kr-pub-a`
- SubnetNames: `subnet-01`
- `NHNAutoOpenDBSecurityGroup: true` — NHN Cloud RDS requires a dedicated "DB Security Group" separate from the VPC security group before external access is possible, so this option auto-creates/deletes a fully open (`0.0.0.0/0`) DB Security Group for testing convenience. Not recommended for production use.

---

### Azure — SKIP

- SupportsStorageTypeSelection=false
- Connection: `azure-koreacentral-config`
- Region: `koreacentral` / Zone: `1`
- Azure MySQL Flexible Server's storageSku is read-only and set automatically by Azure
- The test script SKIPs immediately on execution

---

### NCP — SKIP

- SupportsStorageTypeSelection=false
- Connection: `ncp-korea1-config`
- Region: `KR` / Zone: `KR-1`
- NCP MySQL G3 applies SSD automatically; DataStorageTypeCode cannot be specified
- The test script SKIPs immediately on execution

## Test Results

### 2026-08-03

Note: AWS `standard` type removed (deprecated); IBM changed to `SupportsStorageTypeSelection=false`; Tencent tested with `local_ssd` only.

```
================================================================================================================================
                               RDBMS StorageType Test Summary - All CSPs
================================================================================================================================
CSP          | StorageType(Req)     | StorageType(Ret)   | Result | DB Status      | Elapsed    | Reason
--------------------------------------------------------------------------------------------------------------------------------
[*] cloud_auto: Alibaba auto-select type - CSP picks the optimal cloud storage type at provisioning time
--------------------------------------------------------------------------------------------------------------------------------
AWS          | gp2                  | gp2                | PASS   | Available      | 5m43s      | -
AWS          | gp3                  | gp3                | PASS   | Available      | 5m44s      | -
AWS          | io1                  | io1                | PASS   | Available      | 5m13s      | -
AWS          | io2                  | io2                | PASS   | Available      | 5m12s      | -
AZURE        | N/A                  | N/A                | SKIP   | NOT_APPLICABLE | -          | SupportsStorageTypeSelection=false: storageSku is read-only, set automatically by Azure
GCP          | HYPERDISK_BALANCED   | HYPERDISK_BALANCED | PASS   | Available      | 3m39s      | -
GCP          | HYPERDISK_BALANCED   | HYPERDISK_BALANCED | PASS   | Available      | 3m9s       | -
GCP          | PD_HDD               | PD_HDD             | PASS   | Available      | 3m55s      | -
GCP          | PD_SSD               | PD_SSD             | PASS   | Available      | 4m10s      | -
GCP          | PD_SSD               | PD_SSD             | PASS   | Available      | 4m56s      | -
ALIBABA      | cloud_auto[*]        | general_essd       | PASS   | Available      | 3m52s      | cloud_auto: auto-select type, CSP chose 'general_essd'
ALIBABA      | cloud_essd           | cloud_essd         | PASS   | Available      | 3m36s      | -
ALIBABA      | cloud_essd2          | cloud_essd2        | PASS   | Available      | 3m51s      | -
ALIBABA      | cloud_essd3          | cloud_essd3        | PASS   | Available      | 3m4s       | -
ALIBABA      | local_ssd            | local_ssd          | PASS   | Available      | 4m43s      | -
TENCENT      | local_ssd            | local_ssd          | PASS   | Available      | 4m38s      | -
IBM          | N/A                  | N/A                | SKIP   | NOT_APPLICABLE | -          | SupportsStorageTypeSelection=false: IBM Cloud Databases has no storage type selection
OPENSTACK    | __DEFAULT__          | NA                 | PASS   | Available      | 5m0s       | OpenStack Trove does not expose StorageType post-creation; Available=PASS
OPENSTACK    | RBD                  | NA                 | PASS   | Available      | 5m22s      | OpenStack Trove does not expose StorageType post-creation; Available=PASS
NCP          | N/A                  | N/A                | SKIP   | NOT_APPLICABLE | -          | SupportsStorageTypeSelection=false: NCP G3 applies SSD automatically, StorageType cannot be specified
NHN          | General HDD          | General HDD        | PASS   | Available      | 10m7s      | -
NHN          | General SSD          | General SSD        | PASS   | Available      | 11m7s      | -
--------------------------------------------------------------------------------------------------------------------------------
Total: 22  PASS: 19  FAIL: 0  SKIP: 3
```
