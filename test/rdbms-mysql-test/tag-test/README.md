# CB-Spider RDBMS Tag Management Test

Test suite that validates the Tag CRUD API on RDBMS resources. It only runs against the CSPs where `RDBMSMetaInfo.SupportsTag=true` (AWS, Azure, GCP, Alibaba, Tencent, IBM).

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### RDBMS Instance Must Already Exist

This test assumes the RDBMS instance already exists. First run the network prerequisites and the create test in the parent directory:

```bash
cd ..
./run-all-csp-network-prepare.sh   # VPC/Subnet/SG prerequisites (one-time)
./run-all-csp-rdbms-tests.sh
```

### Required Tools

- `bash` 3.2+
- `curl`
- `jq`

## Supported CSPs

Which CSPs are tested is determined by the `RDBMSMetaInfo.SupportsTag` value.

| CSP | SupportsTag | Tested |
|-----|-------------|---------|
| AWS | `true` | Yes |
| Azure | `true` | Yes |
| GCP | `true` | Yes |
| Alibaba | `true` | Yes |
| Tencent | `true` | Yes |
| IBM | `true` | Yes |
| OpenStack | `false` | No |
| NCP | `false` | No |
| NHN | `false` | No |

## Test Flow

For each CSP, Tag CRUD is validated in this order:

1. **AddTag(1)** — `POST /spider/tag` — adds the first tag (`spider-rdbms-tag` / `rdbms-tag-value`)
2. **ListTag** — `GET /spider/tag?...` — confirms the first tag is present in the list
3. **GetTag** — `GET /spider/tag/{Key}?...` — confirms the tag value matches
4. **AddTag(2)** — `POST /spider/tag` — adds a second tag (`spider-rdbms-tag2` / `rdbms-tag-value2`)
5. **RemoveTag** — `DELETE /spider/tag/{Key}` — deletes the first tag
6. **VerifyRemoved** — `GET /spider/tag?...` — confirms the first tag is gone from the list
7. **Cleanup** — deletes the second tag (cleanup)

A CSP is judged an overall PASS only if every step passes.

## Configuration

```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:****             # Basic auth (admin:<password>)
```

## How to Run Tests

### All Supported CSPs in Parallel

```bash
./run-all-csp-rdbms-tag-tests.sh
```

- Runs Tag CRUD validation in parallel on the 6 CSPs where SupportsTag=true
- Prints a unified result table and a PASS/FAIL tally when done

**Example output:**
```
===================================================================================================================
                   RDBMS TAG MANAGEMENT TEST SUMMARY (SupportsTag=true CSPs)
===================================================================================================================
CSP          | AddTag   | ListTag  | GetTag  | AddTag2  | RemoveTag  | VerifyRemoved  | Elapsed
-------------------------------------------------------------------------------------------------------------------
AWS          | PASS     | PASS     | PASS    | PASS     | PASS       | PASS           | 3s
AZURE        | PASS     | PASS     | PASS    | PASS     | PASS       | PASS           | 2s
GCP          | PASS     | PASS     | PASS    | PASS     | PASS       | PASS           | 1s
ALIBABA      | PASS     | PASS     | PASS    | PASS     | PASS       | PASS           | 4s
TENCENT      | PASS     | PASS     | PASS    | PASS     | PASS       | PASS           | 2s
IBM          | PASS     | PASS     | PASS    | PASS     | PASS       | PASS           | 3s
-------------------------------------------------------------------------------------------------------------------

Total: 6 PASS, 0 FAIL
```

### Individual CSP

To run a single CSP on its own:

```bash
./aws-rdbms-tag-test.sh
./azure-rdbms-tag-test.sh
./gcp-rdbms-tag-test.sh
./alibaba-rdbms-tag-test.sh
./tencent-rdbms-tag-test.sh
./ibm-rdbms-tag-test.sh
```

When run individually, the result file goes to the `RESULT_DIR` you specify, or to the default (`/tmp/rdbms_tag_results`).

## Script Structure

```
tag-test/
├── run-all-csp-rdbms-tag-tests.sh   # Orchestrator: parallel run across the 6 CSPs, PASS/FAIL tally
├── common-rdbms-tag-test.sh         # Common: AddTag -> ListTag -> GetTag -> AddTag2 -> RemoveTag -> VerifyRemoved
├── aws-rdbms-tag-test.sh
├── azure-rdbms-tag-test.sh
├── gcp-rdbms-tag-test.sh
├── alibaba-rdbms-tag-test.sh
├── tencent-rdbms-tag-test.sh
└── ibm-rdbms-tag-test.sh
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:****` | Basic auth credentials |
| `RESULT_DIR` | `/tmp/rdbms_tag_results` | Result file output directory |
| `VERBOSE` | `0` | Set to `1` for a per-CSP full log dump |

```bash
# Example: verbose output
VERBOSE=1 ./run-all-csp-rdbms-tag-tests.sh
```

## Result Format

Each result file (`result_<csp>.txt`) has 8 pipe-separated fields:

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
| `VerifyRemoved` | Result of confirming it's gone from the list after deletion |
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

### 2026-08-03

```
===================================================================================================================
                 RDBMS TAG MANAGEMENT TEST SUMMARY (SupportsTag=true CSPs)
===================================================================================================================
CSP          | AddTag   | ListTag  | GetTag  | AddTag2  | RemoveTag  | VerifyRemoved | Elapsed
-------------------------------------------------------------------------------------------------------------------
AWS          | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 6s
AZURE        | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 4m14s
GCP          | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 1m45s
ALIBABA      | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 7s
TENCENT      | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 5s
IBM          | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 22s
-------------------------------------------------------------------------------------------------------------------
Total: 6 PASS, 0 FAIL
```
