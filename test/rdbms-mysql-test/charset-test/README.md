# CB-Spider RDBMS Charset Configuration Test

Test suite that validates CB-Spider's `Charset` parameter on the Database CRUD API (see
`api-runtime/common-runtime/RDBMSManager.go`'s `CreateRDBMSDatabase` and the
`rdbmsDatabaseOptionsManager`/`rdbmsDatabaseCharsetManager`/`rdbmsDatabaseSQLStatementBuilder`/
`rdbmsDatabaseSQLFallbackEligible` interfaces -- see Scope below for which CSP uses which). It
doesn't just check that `CreateDatabase` returns success -- it connects directly to the newly
created database and verifies the charset actually took effect, both via `information_schema`
metadata and a real INSERT/SELECT round-trip.

## Why this is a separate test from `../database-test`

`../database-test` validates the basic CreateDatabase/ListDatabases/DeleteDatabase CRUD flow with
no charset/collation involved at all. This suite exists because CB-Spider does not validate the
`Charset` value against a fixed list (MySQL/MariaDB add new charsets over time -- see the
charset/collation support survey in project history) -- it only checks the string is safe to use as
a SQL identifier and passes it straight through to the CSP API or SQL engine. That means a
`CreateDatabase` call can return `"created"` even if the CSP silently ignored the charset and used
its own default instead. The only way to catch that is to look at what the database actually does
with real data.

## Scope

All 9 RDBMS-capable CSPs, via four different mechanisms (see `RDBMSManager.go`):

| Mechanism | CSPs | How Charset is applied |
|---|---|---|
| `rdbmsDatabaseOptionsManager` | Azure, GCP, OpenStack, Alibaba | Native "create database" API field |
| `rdbmsDatabaseCharsetManager` | Tencent | Native API field (collation, tested separately in `../collation-test`, has no native field at all -- applied via a SQL `ALTER DATABASE` follow-up) |
| `rdbmsDatabaseSQLStatementBuilder` | NCP | NCP's documented `sys.ncp_create_db('name','charset','collation')` stored procedure, called over direct SQL (NCP's native API has no charset field, and a plain `CREATE DATABASE` is blocked for the master account by NCP's own design) |
| `rdbmsDatabaseSQLFallbackEligible` | NHN | Plain SQL `CREATE DATABASE ... CHARACTER SET ...` -- **only works if the instance owner has separately enabled "Direct Control" in the NHN console** (see `nhn-charset-test.sh`); CB-Spider can't check or enable this itself, so an un-opted-in instance fails here with the engine's own permission error, not a CB-Spider message |
| Plain SQL fallback | AWS, IBM | No CSP-native database API at all; `CREATE DATABASE ... CHARACTER SET ...` directly |

Collation is tested separately -- see `../collation-test`.

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### RDBMS Instance Must Already Exist

This test assumes the RDBMS instance already exists -- the same instance `../database-test` and
`../tls-test` use. First run the network prerequisites and the create test in the parent directory:

```bash
cd ..
./run-all-csp-network-prepare.sh   # VPC/Subnet/SG prerequisites (one-time)
./run-all-csp-rdbms-tests.sh
```

### Required Tools

- `bash` 3.2+
- `go` (1.21+) -- to build/run `charsetprobe`, the Go program that does the actual SQL-level
  verification (see below for why this isn't a `curl`/`mysql` CLI script)

## Test Flow

For each CSP, two charset scenarios are run back-to-back:

| Scenario | Charset | Why this charset |
|---|---|---|
| U8 | `utf8mb4` | CB-Spider/most CSPs' effective default -- confirms the happy path still works |
| L1 | `latin1` | A legacy, single-byte Western European charset that cannot represent Korean or emoji characters -- chosen specifically so its behavior is observably different from utf8mb4, which is what makes this a real functional test rather than a check that would pass even if the CSP ignored the request |

For each scenario:

1. **CreateDatabase** -- `POST /spider/rdbms/{Name}/databases` with `Charset` set to the scenario's
   value. (`cbspidercsutf8mb4` / `cbspidercslatin1`)
2. **Metadata check** -- connects directly to the RDBMS endpoint via `go-sql-driver/mysql` (the
   same driver CB-Spider's own SQL fallback path uses) and queries
   `information_schema.SCHEMATA.DEFAULT_CHARACTER_SET_NAME` for the new database.
3. **Functional check** -- creates a table with *no* explicit charset (so it inherits the
   database's default), then inserts a string containing both a 3-byte Korean character and a
   4-byte emoji -- neither representable in latin1 -- and reads it back:
   - **utf8mb4**: the insert must succeed and the value must round-trip byte-for-byte.
   - **latin1**: the insert must either fail outright (the common case under strict `sql_mode`) or,
     if it succeeds, the round-tripped value must **not** match the original (non-strict
     `sql_mode` silently replaces unrepresentable characters). An *exact* round-trip under the
     latin1 scenario is a FAIL -- it means the CSP silently used utf8mb4 (or some other Unicode
     charset) despite the explicit `latin1` request.
4. **Cleanup** -- the test database is deleted via `DELETE /spider/rdbms/{Name}/databases/{DBName}`
   (best-effort; also pre-cleans up a leftover database from a previous aborted run before
   creating).

### Implementation Approach: why a Go program instead of `curl`/`jq`/the `mysql` CLI

`../database-test` and `../tls-test`'s own READMEs cover the general reasoning (cross-platform
reliability, reuse of CB-Spider's own driver). It applies even more strongly here: this test needs
to send and verify exact UTF-8 byte sequences (Korean + emoji) through a live SQL connection and
compare them byte-for-byte on read-back -- something that's fragile and easy to get subtly wrong
through shell string interpolation, a particular `mysql` CLI build's own default charset handling,
or `jq`'s JSON string escaping. `charsetprobe/main.go` uses `go-sql-driver/mysql` directly, the same
driver `api-runtime/common-runtime/RDBMSManager.go`'s own SQL fallback path uses, so the comparison
is apples-to-apples with how CB-Spider itself talks to these databases.

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
./run-all-csp-charset-test.sh
```

- Runs the charset verification on all 9 CSPs in parallel
- Prints a unified result table and a PASS/FAIL tally when done

**Example output:**
```
====================================================================================================================
                         RDBMS CHARSET CONFIGURATION TEST SUMMARY
====================================================================================================================

CSP        | CreateU8  | MetaU8    | FuncU8    | CreateL1  | MetaL1    | FuncL1    | Elapsed
--------------------------------------------------------------------------------------------------------------------
AWS        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 4s
AZURE      | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 11s
GCP        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 5s
ALIBABA    | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 8s
TENCENT    | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 7s
IBM        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 9s
OPENSTACK  | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 6s
NCP        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 12s
NHN        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 14s
--------------------------------------------------------------------------------------------------------------------

Total: 9 PASS, 0 FAIL
```

NHN's row above assumes "Direct Control" is already enabled on the test instance -- see the
mechanism table above and `nhn-charset-test.sh`.

### Individual CSP

```bash
./aws-charset-test.sh
./azure-charset-test.sh
./gcp-charset-test.sh
./alibaba-charset-test.sh
./tencent-charset-test.sh
./ibm-charset-test.sh
./openstack-charset-test.sh
./ncp-charset-test.sh
./nhn-charset-test.sh
```

When run individually, the result file goes to the `RESULT_DIR` you specify, or to the default
(`/tmp/rdbms_charset_results`).

## Script Structure

```
charset-test/
├── charsetprobe/                     # Go program: does the real create+verify work
│   └── main.go
├── run-all-csp-charset-test.sh       # Orchestrator: full parallel run, PASS/FAIL tally
├── common-charset-test.sh            # Common: thin wrapper around `go run charsetprobe`
├── aws-charset-test.sh
├── azure-charset-test.sh
├── gcp-charset-test.sh
├── alibaba-charset-test.sh
├── tencent-charset-test.sh
├── ibm-charset-test.sh
├── openstack-charset-test.sh
├── ncp-charset-test.sh
└── nhn-charset-test.sh
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:****` | Basic auth credentials |
| `TIMEOUT` | `15` | Per-operation timeout in seconds for direct SQL connections |
| `API_TIMEOUT` | `120` | Per-call timeout in seconds for CB-Spider REST API calls (kept generous -- Azure's CreateDatabase has been observed to take over a minute) |
| `RESULT_DIR` | `/tmp/rdbms_charset_results` | Result file output directory (single-CSP runs) |
| `VERBOSE` | `0` | Set to `1` for a per-CSP full log dump |

```bash
# Example: verbose output
VERBOSE=1 ./run-all-csp-charset-test.sh
```

## Result Format

Each result file (`result_<csp>.txt`) has 8 pipe-separated fields:

```
CSP|CreateUtf8mb4|MetaUtf8mb4|FuncUtf8mb4|CreateLatin1|MetaLatin1|FuncLatin1|Elapsed
```

| Field | Description |
|------|------|
| `CSP` | CSP name (e.g. AWS) |
| `CreateUtf8mb4` / `CreateLatin1` | `CreateDatabase` with that `Charset` returned `"created"` |
| `MetaUtf8mb4` / `MetaLatin1` | `information_schema.SCHEMATA.DEFAULT_CHARACTER_SET_NAME` matches |
| `FuncUtf8mb4` / `FuncLatin1` | functional INSERT/SELECT round-trip behaved as expected (see Test Flow) |
| `Elapsed` | Elapsed time |

A missing result file (the probe crashed or the `go` toolchain wasn't found) prints as
`NO_RESULT`/`SKIP` and counts as a FAIL.

## API Reference

CreateDatabase/DeleteDatabase are the same endpoints `../database-test` uses, with the addition of
the optional `Charset` field:

```json
{
  "ConnectionName": "<connection-name>",
  "DatabaseName": "<db-name>",
  "MasterUserName": "<username>",
  "MasterUserPassword": "<password>",
  "Charset": "utf8mb4"
}
```

See `api-runtime/rest-runtime/RDBMSRest.go`'s `RDBMSDatabaseRequest` for the full field
documentation, including why CB-Spider doesn't validate `Charset`'s value against a fixed list.

## Logs & Results

```
/tmp/rdbms_charset_results_<PID>/result_<csp>.txt
/tmp/rdbms_charset_logs_<PID>/log_<csp>.txt
```

To monitor a run in progress:

```bash
tail -f /tmp/rdbms_charset_logs_<PID>/log_aws.txt
```

## Test Results

### 2026-10-07 (NHN Direct Control 비활성화)

```
====================================================================================================================
                         RDBMS CHARSET CONFIGURATION TEST SUMMARY
====================================================================================================================

CSP        | CreateU8  | MetaU8    | FuncU8    | CreateL1  | MetaL1    | FuncL1    | Elapsed
--------------------------------------------------------------------------------------------------------------------
AWS        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 17s
AZURE      | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 1m42s
GCP        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 15s
ALIBABA    | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 17s
TENCENT    | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 9s
IBM        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 1m36s
OPENSTACK  | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 21s
NCP        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 16s
NHN        | FAIL      | SKIP      | SKIP      | FAIL      | SKIP      | SKIP      | 11s
--------------------------------------------------------------------------------------------------------------------

Total: 8 PASS, 1 FAIL
```

`NHN`의 FAIL은 이 테스트 인스턴스에 아직 "Direct Control"이 활성화되지 않은 상태라서 발생한 것으로, 환경 전제조건 미충족이지 CB-Spider 버그가 아님 (`nhn-charset-test.sh` 참고). 나머지 8개 CSP는 모두 정상.

### 2026-10-07 (NHN Direct Control 활성화 후)

```
====================================================================================================================
                         RDBMS CHARSET CONFIGURATION TEST SUMMARY
====================================================================================================================

CSP        | CreateU8  | MetaU8    | FuncU8    | CreateL1  | MetaL1    | FuncL1    | Elapsed
--------------------------------------------------------------------------------------------------------------------
AWS        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 19s
AZURE      | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 1m40s
GCP        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 15s
ALIBABA    | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 19s
TENCENT    | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 9s
IBM        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 41s
OPENSTACK  | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 28s
NCP        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 16s
NHN        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 15s
--------------------------------------------------------------------------------------------------------------------

Total: 9 PASS, 0 FAIL
```

NHN 테스트 인스턴스에서 "Direct Control"을 활성화한 뒤 재실행한 결과 -- 9개 CSP 전체 정상 확인.
