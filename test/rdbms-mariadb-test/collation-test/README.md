# CB-Spider RDBMS Collation Configuration Test (MariaDB)

Test suite that validates CB-Spider's `Collation` parameter on the Database CRUD API (see
`api-runtime/common-runtime/RDBMSManager.go`'s `CreateRDBMSDatabase` and
`rdbmsDatabaseOptionsManager`) against **MariaDB** instances. This is the MariaDB-engine
counterpart of `../../rdbms-mysql-test/collation-test` -- same mechanics, narrower CSP scope (see
below). It connects directly to the newly created database and verifies the collation actually
took effect, both via `information_schema` metadata and a real functional comparison.

## Why collation gets its own test, separate from `../charset-test`

Same reasoning as `../../rdbms-mysql-test/collation-test/README.md`: charset and collation are
independent, separately-settable parameters, and a CSP can support one without the other. Charset
is tested in isolation in `../charset-test`; collation is tested here with **Charset held fixed at
`utf8mb4`** so that only collation varies.

## Scope: why only 4 CSPs here, vs. 9 for MySQL

See `../charset-test/README.md`'s scope table -- the Collation parameter was wired up together
with Charset in the same change, so the scope is identical: **AWS, Alibaba, OpenStack, NHN**.

NHN requires "Direct Control" to already be enabled on the test instance (see
`nhn-collation-test.sh`) -- without it, this test fails with the engine's own permission error,
not a CB-Spider "not supported" message.

## Confirmed finding: Alibaba MariaDB does not support Collation

Running this suite against a real Alibaba MariaDB instance confirmed Alibaba's documented
restriction: `CollationName` on the RDS `CreateDatabase` API is **MySQL-only**, and a MariaDB
instance does not error when it's given -- it silently keeps its own default
(`utf8mb4_general_ci`) instead. The `utf8mb4_general_ci` scenario here happened to PASS only
because that's also the silent fallback value; the `utf8mb4_bin` scenario exposed the real
behavior (`MetaBin`/`FuncBin` both FAIL -- `DEFAULT_COLLATION_NAME` came back
`utf8mb4_general_ci`, and the case-sensitivity check matched when it should not have), the same
"happens to match the default" false-positive pattern documented for Azure charset-only requests
in `../../rdbms-mysql-test/charset-test/README.md`.

**Fix applied**: `cloud-control-manager/cloud-driver/drivers/alibaba/resources/RDBMSHandler.go`'s
`CreateDatabaseWithOptions` now returns an explicit error when `Collation` is given for a MariaDB
instance, instead of silently succeeding with the wrong collation -- consistent with this
project's "clear error over silent wrong behavior" principle (see the `CreateRDBMSDatabase`
routing in `RDBMSManager.go` for the general version of this rule). Re-running this suite after
the fix should show `CreateBin=FAIL` for ALIBABA with that error message, which is the *correct*
outcome now -- a FAIL here means CB-Spider is refusing to lie about what it did, not that
something is broken.

**OpenStack Trove + MariaDB datastore** collation support was separately confirmed to work (see
`../charset-test/README.md` for the equivalent charset finding) -- this is empirical confirmation
for this specific Trove deployment, not a guarantee for every Trove deployment.

## Known gap: this test does not cover "Collation given, Charset omitted"

Same caveat as `../../rdbms-mysql-test/collation-test/README.md` -- Charset is always sent as
`utf8mb4` alongside Collation in both scenarios here, so the reverse combination (Collation alone,
no Charset) remains unverified on every CSP, including the four in scope here.

## Test Flow

Identical to `../../rdbms-mysql-test/collation-test` -- see that README for the full description of
the two scenarios (`utf8mb4_general_ci`, `utf8mb4_bin`), why they were chosen over each engine's
own version-specific default collation, and the Create/Meta/Func check semantics. The only
difference is the target engine (MariaDB) and the narrower CSP list above.

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### RDBMS Instance Must Already Exist

This test assumes the RDBMS instance already exists -- the same instance `../database-test` and
`../charset-test` use (`cb-spider-mariadb-test`). First run the network prerequisites and the
create test in the parent directory:

```bash
cd ..
./run-all-csp-network-prepare.sh   # VPC/Subnet/SG prerequisites (one-time)
./run-all-csp-rdbms-tests.sh
```

### Required Tools

- `bash` 3.2+
- `go` (1.21+) -- to build/run `collationprobe`, the Go program that does the actual SQL-level
  verification

## Configuration

```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:****             # Basic auth (admin:<password>)
export TIMEOUT=15                         # Per-operation timeout in seconds for direct SQL
                                           # connections (dial/ping/query/exec)
export API_TIMEOUT=120                    # Per-call timeout in seconds for CB-Spider REST API
                                           # calls (GetRDBMS/CreateDatabase/DeleteDatabase)
```

## How to Run Tests

### All CSPs in Parallel

```bash
./run-all-csp-collation-test.sh
```

**Example output:**
```
====================================================================================================================
                   RDBMS COLLATION CONFIGURATION TEST SUMMARY (MariaDB)
====================================================================================================================

CSP        | CreateCI  | MetaCI    | FuncCI    | CreateBin | MetaBin   | FuncBin   | Elapsed
--------------------------------------------------------------------------------------------------------------------
AWS        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 15s
ALIBABA    | FAIL      | SKIP      | SKIP      | FAIL      | SKIP      | SKIP      | 9s
OPENSTACK  | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 27s
NHN        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 16s
--------------------------------------------------------------------------------------------------------------------

Total: 2 PASS, 2 FAIL
```

ALIBABA's FAIL above is expected (see "Confirmed finding" above). NHN's row assumes "Direct
Control" is already enabled on the test instance.

### Individual CSP

```bash
./aws-collation-test.sh
./alibaba-collation-test.sh
./openstack-collation-test.sh
./nhn-collation-test.sh
```

When run individually, the result file goes to the `RESULT_DIR` you specify, or to the default
(`/tmp/rdbms_collation_results`).

## Script Structure

```
collation-test/
├── collationprobe/                     # Go program: does the real create+verify work
│   └── main.go
├── run-all-csp-collation-test.sh       # Orchestrator: full parallel run, PASS/FAIL tally
├── common-collation-test.sh            # Common: thin wrapper around `go run collationprobe`
├── aws-collation-test.sh
├── alibaba-collation-test.sh
├── openstack-collation-test.sh
└── nhn-collation-test.sh
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:****` | Basic auth credentials |
| `TIMEOUT` | `15` | Per-operation timeout in seconds for direct SQL connections |
| `API_TIMEOUT` | `120` | Per-call timeout in seconds for CB-Spider REST API calls |
| `RESULT_DIR` | `/tmp/rdbms_collation_results` | Result file output directory (single-CSP runs) |
| `VERBOSE` | `0` | Set to `1` for a per-CSP full log dump |

## Result Format

Each result file (`result_<csp>.txt`) has 8 pipe-separated fields:

```
CSP|CreateCI|MetaCI|FuncCI|CreateBin|MetaBin|FuncBin|Elapsed
```

See `../../rdbms-mysql-test/collation-test/README.md`'s Result Format section for field
descriptions -- identical here.

## API Reference

Same endpoints and request body as `../../rdbms-mysql-test/collation-test` -- see
`api-runtime/rest-runtime/RDBMSRest.go`'s `RDBMSDatabaseRequest`.

## Logs & Results

```
/tmp/rdbms_collation_results_<PID>/result_<csp>.txt
/tmp/rdbms_collation_logs_<PID>/log_<csp>.txt
```

## Test Results

### 2026-10-07 (NHN Direct Control 비활성화)

```
====================================================================================================================
                   RDBMS COLLATION CONFIGURATION TEST SUMMARY (MariaDB)
====================================================================================================================

CSP        | CreateCI  | MetaCI    | FuncCI    | CreateBin | MetaBin   | FuncBin   | Elapsed
--------------------------------------------------------------------------------------------------------------------
AWS        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 16s
ALIBABA    | FAIL      | SKIP      | SKIP      | FAIL      | SKIP      | SKIP      | 9s
OPENSTACK  | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 18s
NHN        | FAIL      | SKIP      | SKIP      | FAIL      | SKIP      | SKIP      | 8s
--------------------------------------------------------------------------------------------------------------------

Total: 2 PASS, 2 FAIL
```

`ALIBABA`의 FAIL은 "Confirmed finding" 섹션에서 설명한 대로 **의도된 결과**임 -- Alibaba RDS가 MariaDB 인스턴스에서 Collation을 지원하지 않는다는 공식 문서상 제약을 CB-Spider가 명확한 에러로 거부한 것(조용히 무시하지 않음). `NHN`의 FAIL은 이 테스트 인스턴스에 아직 "Direct Control"이 활성화되지 않은 상태라서 발생한 것으로, 환경 전제조건 미충족이지 CB-Spider 버그가 아님 (`nhn-collation-test.sh` 참고). AWS/OpenStack은 모두 정상.

### 2026-10-07 (NHN Direct Control 활성화 후)

```
====================================================================================================================
                   RDBMS COLLATION CONFIGURATION TEST SUMMARY (MariaDB)
====================================================================================================================

CSP        | CreateCI  | MetaCI    | FuncCI    | CreateBin | MetaBin   | FuncBin   | Elapsed
--------------------------------------------------------------------------------------------------------------------
AWS        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 15s
ALIBABA    | FAIL      | SKIP      | SKIP      | FAIL      | SKIP      | SKIP      | 8s
OPENSTACK  | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 20s
NHN        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 13s
--------------------------------------------------------------------------------------------------------------------

Total: 3 PASS, 1 FAIL
```

NHN 테스트 인스턴스(MariaDB)에서 "Direct Control"을 활성화한 뒤 재실행한 결과, NHN은 PASS로 전환. `ALIBABA`는 Direct Control과 무관한, 앞서 설명한 별개의 의도된 제약이라 여전히 FAIL -- 이 FAIL은 정상적으로 유지되는 것이 맞음.
