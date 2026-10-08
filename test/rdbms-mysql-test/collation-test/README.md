# CB-Spider RDBMS Collation Configuration Test

Test suite that validates CB-Spider's `Collation` parameter on the Database CRUD API (see
`api-runtime/common-runtime/RDBMSManager.go`'s `CreateRDBMSDatabase` and the
`rdbmsDatabaseOptionsManager`/`rdbmsDatabaseCharsetManager`/`rdbmsDatabaseSQLStatementBuilder`/
`rdbmsDatabaseSQLFallbackEligible` interfaces -- see Scope below for which CSP uses which). This is
the collation counterpart of `../charset-test` -- it doesn't just check that `CreateDatabase`
returns success, it connects directly to the newly created database and verifies the collation
actually took effect, both via `information_schema` metadata and a real functional comparison.

## Why collation gets its own test, separate from `../charset-test`

Charset and collation are independent, separately-settable parameters, and a CSP can support one
without the other (see the project's charset/collation support survey -- e.g. Tencent's native API
accepts a charset but has no collation field at all). Bundling them into one test would also hide
*which* parameter a given failure belongs to. So charset is tested in isolation (letting the engine
pick a collation on its own) in `../charset-test`, and collation is tested here with **charset held
fixed at `utf8mb4`** so that only collation varies -- any difference in observed behavior between
the two scenarios below is attributable to collation alone.

This suite exists for the same reason `../charset-test` does: CB-Spider does not validate the
`Collation` value against a fixed list -- it only checks the string is safe to use as a SQL
identifier and passes it straight through to the CSP API or SQL engine. A `CreateDatabase` call can
return `"created"` even if the CSP silently ignored the collation and used its own default instead
-- exactly what was found on Azure during `../charset-test` development (see
`RDBMSManager.go`'s `lookupDefaultCollation` and its call site for the fix that resulted).

## Scope

All 9 RDBMS-capable CSPs -- the same scope as `../charset-test`, see its Scope section for the
mechanism table. Two are worth calling out specifically for collation:

- **Tencent**'s native API has no collation field at all, so this test always exercises
  CB-Spider's SQL `ALTER DATABASE` follow-up (`rdbmsDatabaseCharsetManager`) -- unlike
  `../charset-test`, where Tencent's charset goes straight through the native API.
- **NHN** requires "Direct Control" to already be enabled on the instance (see
  `nhn-collation-test.sh`) -- without it, this test fails with the engine's own permission error,
  not a CB-Spider "not supported" message.

## Known gap: this test does not cover "Collation given, Charset omitted"

Charset is always sent as `utf8mb4` alongside Collation in both scenarios here (see above). The
reverse combination -- a caller supplying only `Collation` and leaving `Charset` empty, relying on
the engine to infer the charset from the collation name (standard, valid MySQL/MariaDB behavior) --
is not covered. Azure was found to silently drop a `Charset`-only request unless `Collation` was
also given (see `../charset-test/README.md` history); it is plausible the same CSP would similarly
drop a `Collation`-only request unless `Charset` is also given, but this hasn't been verified, and
`RDBMSManager.go`'s `lookupDefaultCollation` fix only resolves the `Charset`-given/`Collation`-empty
direction, not this one. Treat `Collation` without `Charset` as unverified until a similar fix (or a
test proving it already works) is in place.

## Test Flow

For each CSP, two collation scenarios are run back-to-back, with `Charset=utf8mb4` fixed in both:

| Scenario | Collation | Behavior under correct application |
|---|---|---|
| CI | `utf8mb4_general_ci` | Case-insensitive comparison (`'TestWord' = 'testword'` is TRUE) |
| Bin | `utf8mb4_bin` | Byte-exact comparison (`'TestWord' = 'testword'` is FALSE) |

These two were chosen because both have existed, under these exact names, in every
MySQL/MariaDB version that supports `utf8mb4` at all -- unlike each engine's own *default*
collation for `utf8mb4` (MySQL 8.0's is `utf8mb4_0900_ai_ci`; MariaDB 11.4.2+'s is
`utf8mb4_uca1400_ai_ci`; MySQL 5.7's is `utf8mb4_general_ci`), which would make this test's own
pass/fail expectations a moving target depending on the instance's engine version.

For each scenario:

1. **CreateDatabase** -- `POST /spider/rdbms/{Name}/databases` with `Charset=utf8mb4` and
   `Collation` set to the scenario's value. (`cbspidercolci` / `cbspidercolbin`)
2. **Metadata check** -- connects directly to the RDBMS endpoint via `go-sql-driver/mysql` (the
   same driver CB-Spider's own SQL fallback path uses) and queries
   `information_schema.SCHEMATA.DEFAULT_COLLATION_NAME` for the new database.
3. **Functional check** -- creates a table with *no* explicit column/table collation (so it
   inherits the database's default), inserts the literal string `"TestWord"`, then runs
   `SELECT COUNT(*) FROM collation_probe WHERE txt = 'testword'` (lowercase):
   - **CI**: the count must be greater than zero (case-insensitive match).
   - **Bin**: the count must be zero (case-sensitive, no match).

   Plain ASCII upper/lowercase is used deliberately -- it needs no locale-specific case-folding
   rules and is unambiguous under any collation, unlike e.g. German ß or Turkish dotless i.
4. **Cleanup** -- the test database is deleted via `DELETE /spider/rdbms/{Name}/databases/{DBName}`
   (best-effort; also pre-cleans up a leftover database from a previous aborted run before
   creating).

### Implementation Approach

Same as `../charset-test`: a standalone Go program (`collationprobe/main.go`) using
`go-sql-driver/mysql` directly, for cross-platform reliability and to reuse the exact driver
CB-Spider's own SQL fallback path uses, rather than shelling out to the `mysql` CLI/curl/jq.

## Configuration

```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:****             # Basic auth (admin:<password>)
export TIMEOUT=15                         # Per-operation timeout in seconds for direct SQL
                                           # connections (dial/ping/query/exec)
export API_TIMEOUT=120                    # Per-call timeout in seconds for CB-Spider REST API
                                           # calls (GetRDBMS/CreateDatabase/DeleteDatabase) --
                                           # kept generous because some CSPs (observed on Azure)
                                           # poll a CSP-native async operation to completion
                                           # before responding, which can take over a minute
```

## How to Run Tests

### All CSPs in Parallel

```bash
./run-all-csp-collation-test.sh
```

- Runs the collation verification on all 9 CSPs in parallel
- Prints a unified result table and a PASS/FAIL tally when done

**Example output:**
```
====================================================================================================================
                        RDBMS COLLATION CONFIGURATION TEST SUMMARY
====================================================================================================================

CSP        | CreateCI  | MetaCI    | FuncCI    | CreateBin | MetaBin   | FuncBin   | Elapsed
--------------------------------------------------------------------------------------------------------------------
AWS        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 16s
AZURE      | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 1m42s
GCP        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 20s
ALIBABA    | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 18s
TENCENT    | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 10s
IBM        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 47s
OPENSTACK  | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 28s
NCP        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 15s
NHN        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 17s
--------------------------------------------------------------------------------------------------------------------

Total: 9 PASS, 0 FAIL
```

NHN's row above assumes "Direct Control" is already enabled on the test instance.

### Individual CSP

```bash
./aws-collation-test.sh
./azure-collation-test.sh
./gcp-collation-test.sh
./alibaba-collation-test.sh
./tencent-collation-test.sh
./ibm-collation-test.sh
./openstack-collation-test.sh
./ncp-collation-test.sh
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
├── azure-collation-test.sh
├── gcp-collation-test.sh
├── alibaba-collation-test.sh
├── tencent-collation-test.sh
├── ibm-collation-test.sh
├── openstack-collation-test.sh
├── ncp-collation-test.sh
└── nhn-collation-test.sh
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:****` | Basic auth credentials |
| `TIMEOUT` | `15` | Per-operation timeout in seconds for direct SQL connections |
| `API_TIMEOUT` | `120` | Per-call timeout in seconds for CB-Spider REST API calls (kept generous -- Azure's CreateDatabase has been observed to take over a minute) |
| `RESULT_DIR` | `/tmp/rdbms_collation_results` | Result file output directory (single-CSP runs) |
| `VERBOSE` | `0` | Set to `1` for a per-CSP full log dump |

```bash
# Example: verbose output
VERBOSE=1 ./run-all-csp-collation-test.sh
```

## Result Format

Each result file (`result_<csp>.txt`) has 8 pipe-separated fields:

```
CSP|CreateCI|MetaCI|FuncCI|CreateBin|MetaBin|FuncBin|Elapsed
```

| Field | Description |
|------|------|
| `CSP` | CSP name (e.g. AWS) |
| `CreateCI` / `CreateBin` | `CreateDatabase` with that `Collation` returned `"created"` |
| `MetaCI` / `MetaBin` | `information_schema.SCHEMATA.DEFAULT_COLLATION_NAME` matches |
| `FuncCI` / `FuncBin` | functional case-comparison behaved as expected (see Test Flow) |
| `Elapsed` | Elapsed time |

A missing result file (the probe crashed or the `go` toolchain wasn't found) prints as
`NO_RESULT`/`SKIP` and counts as a FAIL.

## API Reference

CreateDatabase/DeleteDatabase are the same endpoints `../database-test` and `../charset-test` use,
with both the `Charset` and `Collation` fields set:

```json
{
  "ConnectionName": "<connection-name>",
  "DatabaseName": "<db-name>",
  "MasterUserName": "<username>",
  "MasterUserPassword": "<password>",
  "Charset": "utf8mb4",
  "Collation": "utf8mb4_bin"
}
```

See `api-runtime/rest-runtime/RDBMSRest.go`'s `RDBMSDatabaseRequest` for the full field
documentation.

## Logs & Results

```
/tmp/rdbms_collation_results_<PID>/result_<csp>.txt
/tmp/rdbms_collation_logs_<PID>/log_<csp>.txt
```

To monitor a run in progress:

```bash
tail -f /tmp/rdbms_collation_logs_<PID>/log_aws.txt
```

## Test Results

### 2026-10-07 (NHN Direct Control 비활성화)

```
====================================================================================================================
                        RDBMS COLLATION CONFIGURATION TEST SUMMARY
====================================================================================================================

CSP        | CreateCI  | MetaCI    | FuncCI    | CreateBin | MetaBin   | FuncBin   | Elapsed
--------------------------------------------------------------------------------------------------------------------
AWS        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 17s
AZURE      | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 1m40s
GCP        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 15s
ALIBABA    | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 25s
TENCENT    | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 18s
IBM        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 42s
OPENSTACK  | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 20s
NCP        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 17s
NHN        | FAIL      | SKIP      | SKIP      | FAIL      | SKIP      | SKIP      | 11s
--------------------------------------------------------------------------------------------------------------------

Total: 8 PASS, 1 FAIL
```

`NHN`의 FAIL은 이 테스트 인스턴스에 아직 "Direct Control"이 활성화되지 않은 상태라서 발생한 것으로, 환경 전제조건 미충족이지 CB-Spider 버그가 아님 (`nhn-collation-test.sh` 참고). 나머지 8개 CSP는 모두 정상, Tencent의 SQL `ALTER DATABASE` 후속 처리 경로와 NCP의 저장 프로시저 경로도 포함해서 정상 동작 확인.

### 2026-10-07 (NHN Direct Control 활성화 후)

```
====================================================================================================================
                        RDBMS COLLATION CONFIGURATION TEST SUMMARY
====================================================================================================================

CSP        | CreateCI  | MetaCI    | FuncCI    | CreateBin | MetaBin   | FuncBin   | Elapsed
--------------------------------------------------------------------------------------------------------------------
AWS        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 17s
AZURE      | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 1m41s
GCP        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 13s
ALIBABA    | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 19s
TENCENT    | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 18s
IBM        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 38s
OPENSTACK  | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 19s
NCP        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 17s
NHN        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 15s
--------------------------------------------------------------------------------------------------------------------

Total: 9 PASS, 0 FAIL
```

NHN 테스트 인스턴스에서 "Direct Control"을 활성화한 뒤 재실행한 결과 -- 9개 CSP 전체 정상 확인.
