# CB-Spider RDBMS Charset Configuration Test (MariaDB)

Test suite that validates CB-Spider's `Charset` parameter on the Database CRUD API (see
`api-runtime/common-runtime/RDBMSManager.go`'s `CreateRDBMSDatabase` and
`rdbmsDatabaseOptionsManager`) against **MariaDB** instances. This is the MariaDB-engine
counterpart of `../../rdbms-mysql-test/charset-test` -- same mechanics, narrower CSP scope (see
below). It doesn't just check that `CreateDatabase` returns success -- it connects directly to the
newly created database and verifies the charset actually took effect, both via
`information_schema` metadata and a real INSERT/SELECT round-trip.

## Why this is a separate test from `../database-test`

Same reasoning as `../../rdbms-mysql-test/charset-test/README.md`: CB-Spider does not validate the
`Charset` value against a fixed list, it only checks the string is safe to use as a SQL identifier
and passes it straight through. A `CreateDatabase` call can return `"created"` even if the CSP
silently ignored the charset and used its own default instead -- this is what actually happened
during the MySQL-engine version of this test's development (on Azure). The only way to catch that
is to look at what the database actually does with real data.

## Scope: why only 4 CSPs here, vs. 9 for MySQL

| CSP | Supports MariaDB at all? | Charset wired up in CB-Spider? | In this test? |
|---|---|---|---|
| AWS | ✅ | ✅ (SQL fallback) | ✅ |
| Alibaba | ✅ | ✅ (native API) | ✅ |
| OpenStack | ✅ | ✅ (native API) | ✅ |
| NHN | ✅ | ✅ (SQL fallback, requires "Direct Control" enabled -- see `nhn-charset-test.sh`) | ✅ |
| Azure | ❌ (MySQL Flexible Server only) | -- | ❌ |
| GCP | ❌ (Cloud SQL has no MariaDB engine) | -- | ❌ |
| IBM | ❌ (Cloud Databases for MySQL only) | -- | ❌ |
| Tencent | ❌ (CDB API rejects mariadb) | -- | ❌ |
| NCP | ❌ (Cloud DB for MySQL only) | -- | ❌ |

**AWS, Alibaba, OpenStack, NHN** are the only CSPs where both conditions hold -- the same 4 CSPs as
`../database-test`. See the project's charset/collation support survey for the full reasoning
behind each row.

## Known unverified cases this test resolves empirically

Two specific gaps called out in the charset/collation support survey are exactly what this test
exists to check against a real instance, rather than leave as a documentation guess:

- **Alibaba MariaDB + `CharacterSetName`**: Alibaba's official RDS API docs describe
  `CharacterSetName` without restricting it to MySQL, but the *collation* field (`CollationName`)
  is documented as MySQL-instances-only. Charset itself was not flagged as uncertain for MariaDB --
  this test's `MetaU8`/`MetaL1`/`FuncU8`/`FuncL1` checks confirm it in practice.
- **OpenStack Trove + MariaDB datastore**: Trove's official Database/User API reference only
  explicitly guarantees `character_set`/`collate` support for the **mysql** datastore; MariaDB
  datastore support is deployment-dependent and wasn't confirmed from public docs alone. A PASS
  here is empirical confirmation for whatever Trove deployment the test instance runs on (not a
  guarantee for every Trove deployment everywhere).

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### RDBMS Instance Must Already Exist

This test assumes the RDBMS instance already exists -- the same instance `../database-test` uses
(`cb-spider-mariadb-test`). First run the network prerequisites and the create test in the parent
directory:

```bash
cd ..
./run-all-csp-network-prepare.sh   # VPC/Subnet/SG prerequisites (one-time)
./run-all-csp-rdbms-tests.sh
```

### Required Tools

- `bash` 3.2+
- `go` (1.21+) -- to build/run `charsetprobe`, the Go program that does the actual SQL-level
  verification

## Test Flow

Identical to `../../rdbms-mysql-test/charset-test` -- see that README for the full description of
the two scenarios (utf8mb4, latin1), why latin1 was chosen, and the Create/Meta/Func check
semantics. The only difference is the target engine (MariaDB) and the narrower CSP list above.

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
./run-all-csp-charset-test.sh
```

**Example output:**
```
====================================================================================================================
                    RDBMS CHARSET CONFIGURATION TEST SUMMARY (MariaDB)
====================================================================================================================

CSP        | CreateU8  | MetaU8    | FuncU8    | CreateL1  | MetaL1    | FuncL1    | Elapsed
--------------------------------------------------------------------------------------------------------------------
AWS        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 15s
ALIBABA    | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 18s
OPENSTACK  | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 27s
NHN        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 13s
--------------------------------------------------------------------------------------------------------------------

Total: 4 PASS, 0 FAIL
```

NHN's row above assumes "Direct Control" is already enabled on the test instance.

### Individual CSP

```bash
./aws-charset-test.sh
./alibaba-charset-test.sh
./openstack-charset-test.sh
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
├── alibaba-charset-test.sh
├── openstack-charset-test.sh
└── nhn-charset-test.sh
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:****` | Basic auth credentials |
| `TIMEOUT` | `15` | Per-operation timeout in seconds for direct SQL connections |
| `API_TIMEOUT` | `120` | Per-call timeout in seconds for CB-Spider REST API calls |
| `RESULT_DIR` | `/tmp/rdbms_charset_results` | Result file output directory (single-CSP runs) |
| `VERBOSE` | `0` | Set to `1` for a per-CSP full log dump |

## Result Format

Each result file (`result_<csp>.txt`) has 8 pipe-separated fields:

```
CSP|CreateUtf8mb4|MetaUtf8mb4|FuncUtf8mb4|CreateLatin1|MetaLatin1|FuncLatin1|Elapsed
```

See `../../rdbms-mysql-test/charset-test/README.md`'s Result Format section for field
descriptions -- identical here.

## API Reference

Same endpoints and request body as `../../rdbms-mysql-test/charset-test` -- see
`api-runtime/rest-runtime/RDBMSRest.go`'s `RDBMSDatabaseRequest`.

## Logs & Results

```
/tmp/rdbms_charset_results_<PID>/result_<csp>.txt
/tmp/rdbms_charset_logs_<PID>/log_<csp>.txt
```

## Test Results

### 2026-10-07 (NHN Direct Control 비활성화)

```
====================================================================================================================
                    RDBMS CHARSET CONFIGURATION TEST SUMMARY (MariaDB)
====================================================================================================================

CSP        | CreateU8  | MetaU8    | FuncU8    | CreateL1  | MetaL1    | FuncL1    | Elapsed
--------------------------------------------------------------------------------------------------------------------
AWS        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 15s
ALIBABA    | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 23s
OPENSTACK  | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 22s
NHN        | FAIL      | SKIP      | SKIP      | FAIL      | SKIP      | SKIP      | 9s
--------------------------------------------------------------------------------------------------------------------

Total: 3 PASS, 1 FAIL
```

`NHN`의 FAIL은 이 테스트 인스턴스에 아직 "Direct Control"이 활성화되지 않은 상태라서 발생한 것으로, 환경 전제조건 미충족이지 CB-Spider 버그가 아님 (`nhn-charset-test.sh` 참고). AWS/Alibaba/OpenStack은 모두 정상.

### 2026-10-07 (NHN Direct Control 활성화 후)

```
====================================================================================================================
                    RDBMS CHARSET CONFIGURATION TEST SUMMARY (MariaDB)
====================================================================================================================

CSP        | CreateU8  | MetaU8    | FuncU8    | CreateL1  | MetaL1    | FuncL1    | Elapsed
--------------------------------------------------------------------------------------------------------------------
AWS        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 14s
ALIBABA    | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 20s
OPENSTACK  | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 20s
NHN        | PASS      | PASS      | PASS      | PASS      | PASS      | PASS      | 13s
--------------------------------------------------------------------------------------------------------------------

Total: 4 PASS, 0 FAIL
```

NHN 테스트 인스턴스(MariaDB)에서도 "Direct Control"을 활성화한 뒤 재실행한 결과, NHN도 PASS로 전환 -- 4개 CSP 전체 정상 확인.
