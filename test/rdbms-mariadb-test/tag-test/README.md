# CB-Spider RDBMS Tag Management Test (MariaDB)

A test suite that validates the Tag CRUD API for RDBMS resources. It runs only against CSPs that both support MariaDB and have `RDBMSMetaInfo.SupportsTag=true` (AWS, Alibaba). OpenStack and NHN support MariaDB but have `SupportsTag=false`, so they are excluded from this test.

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### RDBMS Instance Must Already Exist

This test assumes an RDBMS instance has already been created. First run the network prerequisite prepare and the create test from the parent directory.

```bash
cd ..
./run-all-csp-network-prepare.sh   # Create VPC/Subnet/SG prerequisites (once)
./run-all-csp-rdbms-tests.sh
```

### Required Tools

- `bash` 3.2+
- `curl`
- `jq`

## Supported CSPs

A CSP is included in this test only if it supports MariaDB **and** has `RDBMSMetaInfo.SupportsTag=true`.

| CSP | MariaDB Support | SupportsTag | Tested |
|-----|------------|-------------|---------|
| AWS | Yes | `true` | Yes |
| Alibaba | Yes | `true` | Yes |
| OpenStack | Partial | `false` | No |
| NHN | Yes | `false` | No |

## Test Flow

For each CSP, Tag CRUD is validated in this order:

1. **AddTag(1)** — `POST /spider/tag` — adds the first tag (`spider-rdbms-tag` / `rdbms-tag-value`)
2. **ListTag** — `GET /spider/tag?...` — confirms the first tag is present in the list
3. **GetTag** — `GET /spider/tag/{Key}?...` — confirms the tag value matches
4. **AddTag(2)** — `POST /spider/tag` — adds a second tag (`spider-rdbms-tag2` / `rdbms-tag-value2`)
5. **RemoveTag** — `DELETE /spider/tag/{Key}` — removes the first tag
6. **VerifyRemoved** — `GET /spider/tag?...` — confirms the first tag is gone from the list
7. **Cleanup** — removes the second tag (housekeeping)

A CSP is judged an overall PASS only if every step passes.

## Configuration

```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:*****           # Basic auth (admin:<password>)
```

## How to Run Tests

### All Supported CSPs in Parallel

```bash
./run-all-csp-rdbms-tag-tests.sh
```

- Runs Tag CRUD validation in parallel against the 2 CSPs that support MariaDB and have SupportsTag=true (AWS, Alibaba)
- Prints a unified result table and PASS/FAIL summary when complete

**Example output:**
```
===================================================================================================================
         RDBMS TAG MANAGEMENT TEST SUMMARY (SupportsTag=true CSPs) - MariaDB
===================================================================================================================
CSP          | AddTag   | ListTag  | GetTag  | AddTag2  | RemoveTag  | VerifyRemoved | Elapsed
-------------------------------------------------------------------------------------------------------------------
AWS          | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 7s
ALIBABA      | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 5s
-------------------------------------------------------------------------------------------------------------------

Total: 2 PASS, 0 FAIL
```

### Individual CSP

To run a single CSP on its own:

```bash
./aws-rdbms-tag-test.sh
./alibaba-rdbms-tag-test.sh
```

For a standalone run, the result file location is controlled by the `RESULT_DIR` environment variable, defaulting to `/tmp/rdbms_tag_results`.

## Script Structure

```
tag-test/
├── run-all-csp-rdbms-tag-tests.sh   # Orchestrator: runs AWS/Alibaba in parallel, aggregates PASS/FAIL
├── common-rdbms-tag-test.sh         # Common: AddTag -> ListTag -> GetTag -> AddTag2 -> RemoveTag -> VerifyRemoved
├── aws-rdbms-tag-test.sh
└── alibaba-rdbms-tag-test.sh
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:*****` | Basic auth credentials |
| `RESULT_DIR` | `/tmp/rdbms_tag_results` | Result file output directory |
| `VERBOSE` | `0` | If set to `1`, dumps the full per-CSP log |

```bash
# Example: verbose output
VERBOSE=1 ./run-all-csp-rdbms-tag-tests.sh
```

## Result Format

The result file (`result_<csp>.txt`) has 8 pipe(`|`)-separated fields:

```
CSP|AddTag|ListTag|GetTag|AddTag2|RemoveTag|VerifyRemoved|Elapsed
```

| Field | Description |
|------|------|
| `CSP` | CSP name (e.g. AWS) |
| `AddTag` | Result of adding the first tag (`PASS` / `FAIL`) |
| `ListTag` | Result of listing tags and confirming presence |
| `GetTag` | Result of fetching a specific tag and confirming its value |
| `AddTag2` | Result of adding the second tag |
| `RemoveTag` | Result of deleting the first tag |
| `VerifyRemoved` | Whether the tag was confirmed gone from the list after deletion |
| `Elapsed` | Elapsed time |

## Logs & Results

```
/tmp/rdbms_tag_results_<PID>/result_<csp>.txt
/tmp/rdbms_tag_logs_<PID>/log_<csp>.txt
```

To monitor a run in progress:

```bash
tail -f /tmp/rdbms_tag_logs_<PID>/log_aws.txt
```

## Tag API Reference

| Operation | Method | Path |
|-----------|--------|------|
| AddTag | `POST` | `/spider/tag` |
| ListTag | `GET` | `/spider/tag?ConnectionName=&ResourceType=rdbms&ResourceName=` |
| GetTag | `GET` | `/spider/tag/{Key}?ConnectionName=&ResourceType=rdbms&ResourceName=` |
| RemoveTag | `DELETE` | `/spider/tag/{Key}` |

## Test Results

### 2026-08-10

```
===================================================================================================================
         RDBMS TAG MANAGEMENT TEST SUMMARY (SupportsTag=true CSPs) - MariaDB
===================================================================================================================
CSP          | AddTag   | ListTag  | GetTag  | AddTag2  | RemoveTag  | VerifyRemoved | Elapsed
-------------------------------------------------------------------------------------------------------------------
AWS          | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 7s
ALIBABA      | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 5s
-------------------------------------------------------------------------------------------------------------------
Total: 2 PASS, 0 FAIL
```
