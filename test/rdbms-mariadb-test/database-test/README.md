# CB-Spider RDBMS Database Management Test (MariaDB)

A test suite that validates the Database CRUD API inside an RDBMS instance. It runs in parallel against the 4 CSPs that support MariaDB (AWS, Alibaba, OpenStack, NHN), verifying the full flow of creating, listing, and deleting a database inside an RDBMS instance.

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

## Test Flow

For each CSP, Database CRUD is validated in this order:

1. **CreateDatabase** — `POST /spider/rdbms/{Name}/databases` — creates the `spidertestdb` database
2. **ListDatabases** — `GET /spider/rdbms/{Name}/databases` — confirms the database list can be retrieved
3. **FoundInList** — confirms `spidertestdb` is present in the list
4. **DeleteDatabase** — `DELETE /spider/rdbms/{Name}/databases/spidertestdb` — deletes the database
5. **VerifyDeleted** — `GET /spider/rdbms/{Name}/databases` — confirms it is gone from the list after deletion

### Implementation

CB-Spider supports database management through two mechanisms:

| Mechanism | Condition | Description |
|------|------|------|
| **CSP native API** | The driver implements the `rdbmsDatabaseManager` interface | Calls the CSP's own database management API directly |
| **Direct SQL execution** | Automatic fallback when the driver doesn't implement it | Connects using `MasterUserPassword` and runs SQL (`CREATE/DROP DATABASE`) |

`MasterUserPassword` is required for the SQL fallback path, so it is always included in the request.

## Configuration

```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:*****           # Basic auth (admin:<password>)
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

- Runs Database CRUD validation in parallel against the 4 CSPs that support MariaDB (AWS/Alibaba/OpenStack/NHN)
- Prints a unified result table and PASS/FAIL summary when complete

**Example output:**
```
=======================================================================================================
             RDBMS DATABASE MANAGEMENT TEST SUMMARY - ALL CSPs (MariaDB)
=======================================================================================================
CSP          | CreateDB   | ListDB   | FoundInList  | DeleteDB   | VerifyDeleted | Elapsed
-------------------------------------------------------------------------------------------------------
AWS          | PASS       | PASS     | FOUND        | PASS       | PASS          | 17s
ALIBABA      | PASS       | PASS     | FOUND        | PASS       | PASS          | 21s
OPENSTACK    | PASS       | PASS     | FOUND        | PASS       | PASS          | 4s
NHN          | PASS       | PASS     | FOUND        | PASS       | PASS          | 31s
-------------------------------------------------------------------------------------------------------

Total: 4 PASS, 0 FAIL
```

### Individual CSP

To run a single CSP on its own:

```bash
./aws-database-test.sh
./alibaba-database-test.sh
./openstack-database-test.sh
./nhn-database-test.sh
```

For a standalone run, the result file location is controlled by the `RESULT_DIR` environment variable, defaulting to `/tmp/rdbms_mgmt_results`.

## Script Structure

```
database-test/
├── run-all-csp-database-tests.sh   # Orchestrator: runs all CSPs in parallel, aggregates PASS/FAIL
├── common-database-test.sh         # Common: CreateDB -> ListDB -> DeleteDB -> VerifyDeleted
├── aws-database-test.sh
├── alibaba-database-test.sh
├── openstack-database-test.sh
└── nhn-database-test.sh
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:*****` | Basic auth credentials |
| `DB_NAME` | `spidertestdb` | Name of the test database to create/delete |
| `RESULT_DIR` | `/tmp/rdbms_mgmt_results` | Result file output directory |
| `VERBOSE` | `0` | If set to `1`, dumps the full per-CSP log |

```bash
# Example: verbose output
VERBOSE=1 ./run-all-csp-database-tests.sh
```

## Result Format

The result file (`result_<csp>.txt`) has 7 pipe(`|`)-separated fields:

```
CSP|CreateDB|ListDB|FoundInList|DeleteDB|VerifyDeleted|Elapsed
```

| Field | Description |
|------|------|
| `CSP` | CSP name (e.g. AWS) |
| `CreateDB` | Database creation result (`PASS` / `FAIL`) |
| `ListDB` | Database list retrieval result |
| `FoundInList` | Whether the created database was found in the list (`FOUND` / `NOT_FOUND`) |
| `DeleteDB` | Database deletion result |
| `VerifyDeleted` | Whether the database was confirmed gone from the list after deletion |
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
(`DatabaseName` is only used for CreateDatabase; `MasterUserPassword` is needed for the SQL fallback path)

## Logs & Results

```
/tmp/rdbms_mgmt_results_<PID>/result_<csp>.txt
/tmp/rdbms_mgmt_logs_<PID>/log_<csp>.txt
```

To monitor a run in progress:

```bash
tail -f /tmp/rdbms_mgmt_logs_<PID>/log_aws.txt
```

## CSP-Specific Notes

| CSP | Note |
|-----|------|
| OpenStack | If the create test in the parent suite (Step 4) fails, no instance exists, so this test fails as well |

## Test Results

Test Date: 2026-08-18

All 4 CSPs passed.

```
=======================================================================================================
             RDBMS DATABASE MANAGEMENT TEST SUMMARY - ALL CSPs (MariaDB)
=======================================================================================================
CSP          | CreateDB   | ListDB   | FoundInList  | DeleteDB   | VerifyDeleted | Elapsed
-------------------------------------------------------------------------------------------------------
AWS          | PASS       | PASS     | FOUND        | PASS       | PASS          | 8s
ALIBABA      | PASS       | PASS     | FOUND        | PASS       | PASS          | 17s
OPENSTACK    | PASS       | PASS     | FOUND        | PASS       | PASS          | 28s
NHN          | PASS       | PASS     | FOUND        | PASS       | PASS          | 33s
-------------------------------------------------------------------------------------------------------
Total: 4 PASS, 0 FAIL
```
