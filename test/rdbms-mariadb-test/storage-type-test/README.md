# CB-Spider RDBMS StorageType Test (MariaDB)

A parallel test suite that creates and validates an RDBMS instance for each StorageType supported by each MariaDB-capable CSP (AWS, Alibaba, OpenStack, NHN). It dynamically fetches the list of supported StorageTypes from each CSP's `rdbmsmetainfo` API, creates one instance per option in parallel, and verifies that the returned StorageType matches what was requested.

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### Pre-created Network Resources

Each CSP must already have a VPC/Subnet (and, for AWS, a Security Group) created before an RDBMS instance can be created.

```bash
cd ..
./run-all-csp-network-prepare.sh
```

### Required Tools

- `bash` 3.2+
- `curl`
- `jq`

## Test Flow

For each CSP, StorageType validation proceeds in this order:

1. **FetchStorageTypeOptions** — `GET /spider/rdbmsmetainfo?DBEngine=mariadb&ConnectionName=...` — dynamically retrieves the list of supported StorageTypes
2. **CreateRDBMS** (in parallel, per StorageType) — `POST /spider/rdbms` — creates a `cb-mariadb-st-<type>` instance for each option
3. **Poll Available** — `GET /spider/rdbms/{Name}` — polls until the instance becomes Available (every 30s by default, up to 3600s)
4. **VerifyStorageType** — confirms the returned StorageType matches what was requested (recorded as `PASS`/`FAIL` in the `Result` field)
5. **(Optional) AutoDelete** — if `AUTO_DELETE=true`, the instance is deleted immediately after verification. With the default (`false`), clean up separately with `delete-all-csp-storage-type-rdbms.sh`

### StorageType Selection Support

| CSP | SupportsStorageTypeSelection | Notes |
|-----|------------------------------|------|
| AWS | Yes | gp2, gp3, io1, io2 |
| Alibaba | Yes | cloud_essd, cloud_essd2, cloud_essd3 |
| OpenStack | Yes | `__DEFAULT__`, `RBD` (requires a MariaDB Trove datastore to be configured) |
| NHN | Yes | General HDD, General SSD |

## Configuration

```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:*****           # Basic auth (admin:<password>)
```

## How to Run Tests

### All CSPs in Parallel

```bash
./run-all-csp-storage-type-tests.sh
```

- Runs StorageType validation in parallel against the 4 CSPs that support MariaDB (AWS/Alibaba/OpenStack/NHN)
- Prints a unified result table and PASS/FAIL/SKIP summary when complete

**Example output:**
```
================================================================================================================================
                           RDBMS StorageType Test Summary - All CSPs (MariaDB)
================================================================================================================================
CSP          | StorageType(Req)     | StorageType(Ret)   | Result | DB Status      | Elapsed    | Reason
--------------------------------------------------------------------------------------------------------------------------------
AWS          | gp2                  | gp2                | PASS   | Available      | 4m41s      | -
ALIBABA      | cloud_essd           | cloud_essd         | PASS   | Available      | 3m26s      | -
OPENSTACK    | __DEFAULT__          | N/A                | FAIL   | CREATE_ERROR   | 1s         | -
NHN          | General HDD          | General HDD        | PASS   | Available      | 9m22s      | -
--------------------------------------------------------------------------------------------------------------------------------

Total: 4  PASS: 3  FAIL: 1  SKIP: 0
```

### Individual CSP

To run a single CSP on its own:

```bash
./aws-storage-type-test.sh
./alibaba-storage-type-test.sh
./openstack-storage-type-test.sh
./nhn-storage-type-test.sh
```

For a standalone run, the result/log directories are controlled by the `RESULT_DIR` / `LOG_DIR` environment variables, defaulting to `/tmp/st_results_<PID>` and `/tmp/st_logs_<PID>`.

### Delete All Instances

```bash
./delete-all-csp-storage-type-rdbms.sh
```

- Re-fetches the StorageTypeOptions to derive each `cb-mariadb-st-<type>` instance name, then deletes them all in parallel across CSPs

## Script Structure

```
storage-type-test/
├── run-all-csp-storage-type-tests.sh    # Orchestrator: runs all CSPs in parallel (AWS/Alibaba/OpenStack/NHN)
├── delete-all-csp-storage-type-rdbms.sh # Orchestrator: deletes all CSPs in parallel
├── common-storage-type-test.sh          # Common: Create -> Poll Available -> Get Info -> Verify -> (optional) Delete
├── aws-storage-type-test.sh
├── alibaba-storage-type-test.sh
├── openstack-storage-type-test.sh
└── nhn-storage-type-test.sh
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:****` | Basic auth credentials |
| `MAX_WAIT_SEC` | `3600` (create) / `1800` (delete) | Timeout per instance (seconds) |
| `POLL_INTERVAL` | `30` (create) / `15` (delete) | Polling interval (seconds) |
| `AUTO_DELETE` | `false` | If `true`, the instance is deleted automatically right after verification |
| `VERBOSE` | `0` | If set to `1`, dumps the full per-CSP log |

```bash
# Example: verbose output
VERBOSE=1 ./run-all-csp-storage-type-tests.sh
```

## Result Format

The result file (`result_<csp>_<storagetype>.txt`) has 7 pipe(`|`)-separated fields:

```
CSP|StorageType(Requested)|StorageType(Returned)|Result|DB_Status|Elapsed|Reason
```

| Field | Description |
|------|------|
| `CSP` | CSP name (e.g. AWS) |
| `StorageType(Requested)` | The StorageType value requested |
| `StorageType(Returned)` | The StorageType value read back after creation (`N/A` if creation failed) |
| `Result` | `PASS` / `FAIL` / `SKIP` |
| `DB_Status` | Instance status (`Available`, `CREATE_ERROR`, `TIMEOUT`, etc.) |
| `Elapsed` | Elapsed time |
| `Reason` | Notes (e.g. OpenStack doesn't expose StorageType post-creation, so it's marked PASS based on Available status alone) |

## API Reference

| Operation | Method | Path |
|-----------|--------|------|
| FetchStorageTypeOptions | `GET` | `/spider/rdbmsmetainfo?DBEngine=mariadb&ConnectionName=` |
| CreateRDBMS | `POST` | `/spider/rdbms` |
| GetRDBMS | `GET` | `/spider/rdbms/{Name}?ConnectionName=` |
| DeleteRDBMS | `DELETE` | `/spider/rdbms/{Name}` |

## Logs & Results

```
/tmp/st_test_<PID>/results/result_<csp>_<storagetype>.txt
/tmp/st_test_<PID>/logs/log_<csp>.txt
```

When running the full delete:
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

`StorageTypeOptions` is fetched dynamically from the metainfo API, and an instance is created for each type with the settings below.

| Item | Value |
|------|-----|
| DBEngine / Version | `mariadb` / `10.6` |
| DBSpec | `db.t3.medium` |
| StorageSize | `100 GB` (same for every type, including io1/io2) |
| Connection | `aws-config01` |
| Region / Zone | `ap-southeast-2` / `ap-southeast-2a` |
| SubnetNames | `subnet-01`, `subnet-02` (different AZs; a SubnetGroup must exist) |
| SecurityGroupNames | `sg-01` |

**Special notes**
- io1/io2: requires the `Iops: "3000"` field

---

### Alibaba

Both `StorageTypeOptions` and `DBSpecOptions` are fetched dynamically from the metainfo API, automatically selecting a spec valid for the target region/zone (a hardcoded spec may not be compatible with the requested StorageType).

| Item | Value |
|------|-----|
| DBEngine / Version | `mariadb` / `10.6` |
| DBSpec | metainfo `DBSpecOptions[0]` (falls back to `rds.mariadb.s4.large` if the lookup fails) |
| StorageSize | `20 GB` by default; `500 GB` for `cloud_essd2`; `1500 GB` for `cloud_essd3` |
| Connection | `alibaba-beijing-config` |
| Region / Zone | `cn-beijing` / `cn-beijing-f` |
| SubnetNames | `subnet-01` |

**Special notes**
- `cloud_essd2`/`cloud_essd3` have a minimum StorageSize requirement, so the script automatically requests a larger size for them

---

### OpenStack

| Item | Value |
|------|-----|
| DBEngine / Version | `mariadb` / `10.4` |
| DBSpec | `m1.small` |
| StorageSize | `20 GB` |
| Connection | `openstack-config01` |
| Region / Zone | `RegionOne` / `nova` |

---

### NHN

| Item | Value |
|------|-----|
| DBEngine / Version | `mariadb` / `MARIADB_V101118` |
| DBSpec | `m2.c2m4` |
| StorageSize | `20 GB` |
| Connection | `nhn-korea-pangyo1-config` |
| Region / Zone | `KR1` / `kr-pub-a` |
| SubnetNames | `subnet-01` |

`NHNAutoOpenDBSecurityGroup: true` — NHN Cloud RDS requires a dedicated "DB Security Group", separate from the VPC Security Group, before external access is possible. For testing convenience this option auto-creates and auto-deletes a fully open (`0.0.0.0/0`) DB Security Group. This is not recommended for production use.


## Test Results

Test Date: 2026-08-18

All 11 cases passed.

```
================================================================================================================================
                           RDBMS StorageType Test Summary - All CSPs (MariaDB)
================================================================================================================================
CSP          | StorageType(Req)     | StorageType(Ret)   | Result | DB Status      | Elapsed    | Reason
--------------------------------------------------------------------------------------------------------------------------------
AWS          | gp2                  | gp2                | PASS   | Available      | 5m14s      | -
AWS          | gp3                  | gp3                | PASS   | Available      | 4m42s      | -
AWS          | io1                  | io1                | PASS   | Available      | 5m15s      | -
AWS          | io2                  | io2                | PASS   | Available      | 5m14s      | -
ALIBABA      | cloud_essd           | cloud_essd         | PASS   | Available      | 3m39s      | -
ALIBABA      | cloud_essd2          | cloud_essd2        | PASS   | Available      | 3m23s      | -
ALIBABA      | cloud_essd3          | cloud_essd3        | PASS   | Available      | 3m24s      | -
OPENSTACK    | __DEFAULT__          | NA                 | PASS   | Available      | 4m28s      | OpenStack Trove does not expose StorageType post-creation; Available=PASS
OPENSTACK    | RBD                  | NA                 | PASS   | Available      | 4m58s      | OpenStack Trove does not expose StorageType post-creation; Available=PASS
NHN          | General HDD          | General HDD        | PASS   | Available      | 10m36s     | -
NHN          | General SSD          | General SSD        | PASS   | Available      | 10m21s     | -
--------------------------------------------------------------------------------------------------------------------------------
Total: 11  PASS: 11  FAIL: 0  SKIP: 0
```
