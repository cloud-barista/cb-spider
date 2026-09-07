# CB-Spider RDBMS Database Management Test

Test suite that validates the Database CRUD API inside an RDBMS instance. It runs in parallel across 9 CSPs, exercising the full flow of creating, listing, and deleting a database inside an already-running RDBMS instance.

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

## Test Flow

For each CSP, Database CRUD is validated in this order:

1. **CreateDatabase** — `POST /spider/rdbms/{Name}/databases` — creates the `spidertestdb` database
2. **ListDatabases** — `GET /spider/rdbms/{Name}/databases` — confirms the database list can be retrieved
3. **FoundInList** — confirms `spidertestdb` is present in the list
4. **DeleteDatabase** — `DELETE /spider/rdbms/{Name}/databases/spidertestdb` — deletes the database
5. **VerifyDeleted** — `GET /spider/rdbms/{Name}/databases` — confirms it's gone from the list after deletion

### Implementation Approach

CB-Spider supports database management through two mechanisms:

| Mechanism | Condition | Description |
|------|------|------|
| **CSP-native API** | The driver implements the `rdbmsDatabaseManager` interface | Calls the CSP's own database management API directly |
| **Direct SQL execution** | Automatic fallback when the driver doesn't implement it | Connects using `MasterUserPassword` and runs SQL (`CREATE/DROP DATABASE`) |

`MasterUserPassword` is always included in requests, since it's required by the SQL fallback path.

## Configuration

```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:****             # Basic auth (admin:<password>)
```

To change the test database name:

```bash
export DB_NAME=mydb ./aws-database-test.sh
```

## How to Run Tests

### All CSPs in Parallel

```bash
./run-all-csp-database-tests.sh
```

- Runs Database CRUD validation on all 9 CSPs in parallel
- Prints a unified result table and a PASS/FAIL tally when done

**Example output:**
```
=======================================================================================================
                  RDBMS DATABASE MANAGEMENT TEST SUMMARY - ALL CSPs
=======================================================================================================
CSP          | CreateDB   | ListDB   | FoundInList  | DeleteDB   | VerifyDeleted  | Elapsed
-------------------------------------------------------------------------------------------------------
AWS          | PASS       | PASS     | FOUND        | PASS       | PASS           | 2s
AZURE        | PASS       | PASS     | FOUND        | PASS       | PASS           | 3s
GCP          | PASS       | PASS     | FOUND        | PASS       | PASS           | 1s
ALIBABA      | PASS       | PASS     | FOUND        | PASS       | PASS           | 2s
TENCENT      | PASS       | PASS     | FOUND        | PASS       | PASS           | 4s
IBM          | PASS       | PASS     | FOUND        | PASS       | PASS           | 3s
OPENSTACK    | PASS       | PASS     | FOUND        | PASS       | PASS           | 2s
NCP          | PASS       | PASS     | FOUND        | PASS       | PASS           | 5s
NHN          | PASS       | PASS     | FOUND        | PASS       | PASS           | 3s
-------------------------------------------------------------------------------------------------------

Total: 9 PASS, 0 FAIL
```

### Individual CSP

To run a single CSP on its own:

```bash
./aws-database-test.sh
./azure-database-test.sh
./gcp-database-test.sh
./alibaba-database-test.sh
./tencent-database-test.sh
./ibm-database-test.sh
./openstack-database-test.sh
./ncp-database-test.sh
./nhn-database-test.sh
```

When run individually, the result file goes to the `RESULT_DIR` you specify, or to the default (`/tmp/rdbms_mgmt_results`).

## Script Structure

```
database-test/
├── run-all-csp-database-tests.sh   # Orchestrator: full parallel run, PASS/FAIL tally
├── common-database-test.sh         # Common: CreateDB -> ListDB -> DeleteDB -> VerifyDeleted
├── aws-database-test.sh
├── azure-database-test.sh
├── gcp-database-test.sh
├── alibaba-database-test.sh
├── tencent-database-test.sh
├── ibm-database-test.sh
├── openstack-database-test.sh
├── ncp-database-test.sh
└── nhn-database-test.sh
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:****` | Basic auth credentials |
| `DB_NAME` | `spidertestdb` | Name of the test database to create/delete |
| `RESULT_DIR` | `/tmp/rdbms_mgmt_results` | Result file output directory (single-CSP runs) |
| `VERBOSE` | `0` | Set to `1` for a per-CSP full log dump |

```bash
# Example: verbose output
VERBOSE=1 ./run-all-csp-database-tests.sh
```

## Result Format

Each result file (`result_<csp>.txt`) has 7 pipe-separated fields:

```
CSP|CreateDB|ListDB|FoundInList|DeleteDB|VerifyDeleted|Elapsed
```

| Field | Description |
|------|------|
| `CSP` | CSP name (e.g. AWS) |
| `CreateDB` | Result of creating the database (`PASS` / `FAIL`) |
| `ListDB` | Result of listing databases |
| `FoundInList` | Whether the created database appears in the list (`FOUND` / `NOT_FOUND`) |
| `DeleteDB` | Result of deleting the database |
| `VerifyDeleted` | Result of confirming it's gone from the list after deletion |
| `Elapsed` | Elapsed time |

## API Reference

| Operation | Method | Path |
|-----------|--------|------|
| CreateDatabase | `POST` | `/spider/rdbms/{Name}/databases` |
| ListDatabases | `GET` | `/spider/rdbms/{Name}/databases` |
| DeleteDatabase | `DELETE` | `/spider/rdbms/{Name}/databases/{DBName}` |

Request body for all calls:
```json
{
  "ConnectionName": "<connection-name>",
  "DatabaseName": "<db-name>",
  "MasterUserPassword": "<password>"
}
```
(`DatabaseName` is only used for CreateDatabase; `MasterUserPassword` is required by the SQL fallback path)

## Logs & Results

```
/tmp/rdbms_mgmt_results_<PID>/result_<csp>.txt
/tmp/rdbms_mgmt_logs_<PID>/log_<csp>.txt
```

To monitor a run in progress:

```bash
tail -f /tmp/rdbms_mgmt_logs_<PID>/log_aws.txt
```

## Test Results

### 2026-08-03

```
=======================================================================================================
                RDBMS DATABASE MANAGEMENT TEST SUMMARY - ALL CSPs
=======================================================================================================
CSP          | CreateDB   | ListDB   | FoundInList  | DeleteDB   | VerifyDeleted | Elapsed
-------------------------------------------------------------------------------------------------------
AWS          | PASS       | PASS     | FOUND        | PASS       | PASS          | 9s
AZURE        | PASS       | PASS     | FOUND        | PASS       | PASS          | 59s
GCP          | PASS       | PASS     | FOUND        | PASS       | PASS          | 10s
ALIBABA      | PASS       | PASS     | FOUND        | PASS       | PASS          | 19s
TENCENT      | PASS       | PASS     | FOUND        | PASS       | PASS          | 9s
IBM          | PASS       | PASS     | FOUND        | PASS       | PASS          | 29s
OPENSTACK    | PASS       | PASS     | FOUND        | PASS       | PASS          | 27s
NCP          | PASS       | PASS     | FOUND        | PASS       | PASS          | 27s
NHN          | PASS       | PASS     | FOUND        | PASS       | PASS          | 31s
-------------------------------------------------------------------------------------------------------
Total: 9 PASS, 0 FAIL
```
