# CB-Spider RDBMS Database Management Test (MariaDB)

RDBMS 인스턴스 내부의 Database CRUD API를 검증하는 테스트 스위트입니다. MariaDB를 지원하는 4개 CSP(AWS, Alibaba, OpenStack, NHN)에 대해 병렬로 실행하며, RDBMS 인스턴스 안에서 데이터베이스를 생성/조회/삭제하는 전체 흐름을 검증합니다.

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### RDBMS 인스턴스 사전 생성

이 시험은 RDBMS 인스턴스가 이미 생성되어 있다고 가정합니다. 먼저 상위 디렉토리의 네트워크 사전 준비 및 create 시험을 실행하세요.

```bash
cd ..
./run-all-csp-network-prepare.sh   # VPC/Subnet/SG 사전 생성 (최초 1회)
./run-all-csp-rdbms-tests.sh
```

### Required Tools

- `bash` 3.2+
- `curl`
- `jq`

## Test Flow

각 CSP에 대해 다음 순서로 Database CRUD를 검증합니다:

1. **CreateDatabase** — `POST /spider/rdbms/{Name}/databases` — `spidertestdb` 데이터베이스 생성
2. **ListDatabases** — `GET /spider/rdbms/{Name}/databases` — 데이터베이스 목록 조회 성공 확인
3. **FoundInList** — 목록에서 `spidertestdb` 존재 확인
4. **DeleteDatabase** — `DELETE /spider/rdbms/{Name}/databases/spidertestdb` — 데이터베이스 삭제
5. **VerifyDeleted** — `GET /spider/rdbms/{Name}/databases` — 삭제 후 목록에서 제거 확인

### 구현 방식

CB-Spider는 다음 두 가지 방식으로 Database 관리를 지원합니다:

| 방식 | 조건 | 설명 |
|------|------|------|
| **CSP 네이티브 API** | 드라이버가 `rdbmsDatabaseManager` 인터페이스 구현 | 각 CSP의 데이터베이스 관리 API 직접 호출 |
| **SQL 직접 실행** | 드라이버 미구현 시 자동 폴백 | `MasterUserPassword`로 접속하여 SQL 실행 (`CREATE/DROP DATABASE`) |

`MasterUserPassword`는 SQL 폴백 경로에서 필요하므로 항상 포함합니다.

## Configuration

```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:*****           # Basic auth (admin:<password>)
```

테스트 DB 이름을 변경하려면:

```bash
export DB_NAME=mydb ./aws-database-test.sh
```

## How to Run Tests

### All CSPs in Parallel

```bash
./run-all-csp-database-tests.sh
```

- MariaDB 지원 4개 CSP(AWS/Alibaba/OpenStack/NHN)에 대해 병렬로 Database CRUD 검증 실행
- 완료 후 통합 결과 테이블 및 PASS/FAIL 집계 출력

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

특정 CSP만 단독 실행:

```bash
./aws-database-test.sh
./alibaba-database-test.sh
./openstack-database-test.sh
./nhn-database-test.sh
```

단독 실행 시 결과 파일은 `RESULT_DIR` 환경변수로 지정하거나 기본값(`/tmp/rdbms_mgmt_results`)이 사용됩니다.

## Script Structure

```
database-test/
├── run-all-csp-database-tests.sh   # Orchestrator: 전체 병렬 실행, PASS/FAIL 집계
├── common-database-test.sh         # Common: CreateDB → ListDB → DeleteDB → VerifyDeleted
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
| `DB_NAME` | `spidertestdb` | 생성/삭제할 테스트 데이터베이스 이름 |
| `RESULT_DIR` | `/tmp/rdbms_mgmt_results` | Result file output directory |
| `VERBOSE` | `0` | `1`로 설정 시 per-CSP 전체 로그 덤프 출력 |

```bash
# Example: verbose output
VERBOSE=1 ./run-all-csp-database-tests.sh
```

## Result Format

결과 파일(`result_<csp>.txt`)은 파이프(|) 구분 7개 필드:

```
CSP|CreateDB|ListDB|FoundInList|DeleteDB|VerifyDeleted|Elapsed
```

| 필드 | 설명 |
|------|------|
| `CSP` | CSP 이름 (예: AWS) |
| `CreateDB` | 데이터베이스 생성 결과 (`PASS` / `FAIL`) |
| `ListDB` | 데이터베이스 목록 조회 결과 |
| `FoundInList` | 생성된 DB가 목록에 존재 (`FOUND` / `NOT_FOUND`) |
| `DeleteDB` | 데이터베이스 삭제 결과 |
| `VerifyDeleted` | 삭제 후 목록에서 제거 확인 결과 |
| `Elapsed` | 경과 시간 |

## API Reference

| Operation | Method | Path |
|-----------|--------|------|
| CreateDatabase | `POST` | `/spider/rdbms/{Name}/databases` |
| ListDatabases | `GET` | `/spider/rdbms/{Name}/databases` |
| DeleteDatabase | `DELETE` | `/spider/rdbms/{Name}/databases/{DBName}` |

모든 요청의 body:
```json
{
  "ConnectionName": "<connection-name>",
  "DatabaseName": "<db-name>",
  "MasterUserPassword": "<password>"
}
```
(`DatabaseName`은 CreateDatabase 전용, `MasterUserPassword`는 SQL 폴백 경로에 필요)

## Logs & Results

```
/tmp/rdbms_mgmt_results_<PID>/result_<csp>.txt
/tmp/rdbms_mgmt_logs_<PID>/log_<csp>.txt
```

실행 중 모니터링:

```bash
tail -f /tmp/rdbms_mgmt_logs_<PID>/log_aws.txt
```

## CSP-Specific Notes

| CSP | Note |
|-----|------|
| OpenStack | 상위 create 시험(Step 4)이 실패하면 인스턴스가 없어 본 시험도 FAIL로 이어짐 |

## 시험 결과

시험 날짜: 2026-08-18

4개 CSP 전부 PASS했습니다.

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
