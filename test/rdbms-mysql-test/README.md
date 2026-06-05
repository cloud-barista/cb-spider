# CB-Spider RDBMS API Test

Automated test suite for CB-Spider RDBMS API — creates MySQL instances across 9 CSPs in parallel, waits until each becomes available, then collects and displays a unified result table.

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### CSP Connection Configuration

Before running tests, register connection names for each CSP in CB-Spider.

| CSP | Connection Name |
|-----|----------------|
| AWS | `aws-config01` |
| Azure | `azure-koreacentral-config` |
| GCP | `gcp-iowa-config` |
| Alibaba | `alibaba-beijing-config` |
| Tencent | `tencent-beijing3-config` |
| IBM | `ibm-us-east-1-config` |
| OpenStack | `openstack-config01` |
| NCP | `ncp-korea1-config` |
| NHN | `nhn-korea-pangyo1-config` |

### Pre-created Network Resources

RDBMS 생성 전에 각 CSP에 VPC와 서브넷이 미리 생성되어 있어야 합니다.

| CSP | VPC | Subnet | 비고 |
|-----|-----|--------|------|
| AWS | `vpc-01` | `subnet-01`, `subnet-02` | 서로 다른 AZ의 서브넷 2개 필수 (SubnetGroup 요건) |
| Azure | `vpc-01` | `subnet-01` | 서브넷 미사용 |
| GCP | `vpc-01` | `subnet-01` | 서브넷 미사용 |
| Alibaba | `vpc-01` | `subnet-01` | |
| Tencent | `vpc-01` | `subnet-01` | |
| IBM | `vpc-01` | `subnet-01` | 서브넷 미사용 |
| OpenStack | `vpc-01` | `subnet-01` | 서브넷 미사용 |
| NCP | `vpc-01` | `subnet-01` | |
| NHN | `vpc-01` | `subnet-01` | |

CB-Spider REST API로 생성하는 경우:

```bash
# VPC 생성 예시 (AWS)
curl -u admin:***** -sX POST http://localhost:1024/spider/vpc \
  -H 'Content-Type: application/json' \
  -d '{
    "ConnectionName": "aws-config01",
    "ReqInfo": {
      "Name": "vpc-01",
      "IPv4_CIDR": "10.0.0.0/16",
      "SubnetInfoList": [
        {"Name": "subnet-01", "IPv4_CIDR": "10.0.1.0/24", "Zone": "<AZ-1>"},
        {"Name": "subnet-02", "IPv4_CIDR": "10.0.2.0/24", "Zone": "<AZ-2>"}
      ]
    }
  }' | jq .
```

### Required Tools

- `bash` 3.2+
- `curl`
- `jq`

## RDBMS Instance Configuration

All CSPs create a MySQL instance named `cb-spider-mysql-test`.

| CSP | Engine Version | Instance Spec | Storage | Subnet Required |
|-----|---------------|---------------|---------|-----------------|
| AWS | 8.0 | db.t3.medium | 100GB | ✅ (2개, 다른 AZ) |
| Azure | 8.0.21 | Standard_B1ms | 20GB | 미사용 |
| GCP | 8.0 | db-custom-2-8192 | 20GB | 미사용 |
| Alibaba | 8.0 | mysql.n2e.small.1 | 20GB | ✅ |
| Tencent | 8.0 | 8000 (MB) | 50GB | ✅ |
| IBM | 8.4 | multitenant | 30GB | 미사용 |
| OpenStack | 5.7.29 | m1.small | 20GB | 미사용 |
| NCP | 8.0.36 | SVR.VDBAS.AMD.STAND.C002.M008.NET.SSD.B050.G003 | 10GB | ✅ |
| NHN | MYSQL_V8408 | m2.c2m4 | 20GB | ✅ |

## Configuration

테스트 실행 전에 CB-Spider 접속 정보를 환경변수로 설정하거나, `run-all-csp-rdbms-tests.sh` / `delete-all-csp-rdbms.sh` 파일 내부의 기본값을 직접 수정합니다.

**방법 1) 환경변수 설정**
```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:*****           # Basic auth (admin:<password>)
```

**방법 2) 스크립트 파일 직접 수정** (`run-all-csp-rdbms-tests.sh`, `delete-all-csp-rdbms.sh`)
```bash
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:*****}"   # <-- 비밀번호 변경
```

> `SPIDER_AUTH`의 비밀번호는 CB-Spider 기동 시 설정한 값으로 변경하세요.

## How to Run Tests

### Create: All CSPs in Parallel

```bash
./run-all-csp-rdbms-tests.sh
```

- 9개 CSP에 동시 RDBMS 생성 (백그라운드 병렬 실행)
- 각 CSP별 Available 상태까지 대기 (최대 60분)
- 완료 후 통합 결과 테이블 출력

**Example output:**
```
CSP          | Status      | Engine   | Version      | Spec                     | Storage    | Endpoint                                 | PublicAccess | Elapsed
---
AWS          | Available   | mysql    | 8.0.45       | db.t3.medium             | 100GB      | xxx.rds.amazonaws.com:3306               | true         | 8m39s
AZURE        | Available   | mysql    | 8.0.21       | Standard_B1ms            | 20GB       | xxx.mysql.database.azure.com:3306        | true         | 5m35s
GCP          | Available   | mysql    | 8.0          | db-custom-2-8192         | 20GB       | xxx.cloudsql.google.com:3306             | true         | 3m58s
...
```

### Delete: All CSPs in Parallel

```bash
./delete-all-csp-rdbms.sh
```

- 9개 CSP의 RDBMS 인스턴스 동시 삭제
- 인스턴스 완전 삭제 확인 후 결과 테이블 출력

**Example output:**
```
CSP          | Result         | Detail               | Elapsed
---
AWS          | DELETED        | ok                   | 1m48s
AZURE        | DELETED        | ok                   | 33s
GCP          | DELETED        | ok                   | 2m9s
...
```

### Run Individual CSP Test

특정 CSP만 단독 실행:

```bash
# Create
./aws-rdbms-test.sh
./azure-rdbms-test.sh
./gcp-rdbms-test.sh
./alibaba-rdbms-test.sh
./tencent-rdbms-test.sh
./ibm-rdbms-test.sh
./openstack-rdbms-test.sh
./ncp-rdbms-test.sh
./nhn-rdbms-test.sh
```

단독 실행 시에는 `RESULT_DIR` 환경변수를 지정하거나 기본값(`/tmp/rdbms_results`)이 사용됩니다.

## Script Structure

```
.
├── run-all-csp-rdbms-tests.sh   # Orchestrator: 전체 생성 테스트 (병렬)
├── delete-all-csp-rdbms.sh      # Orchestrator: 전체 삭제 (병렬)
├── common-rdbms-test.sh         # Common: Create → Poll Available → Get Info
├── common-rdbms-delete.sh       # Common: Verify → Delete → Poll Removed
├── aws-rdbms-test.sh
├── azure-rdbms-test.sh
├── gcp-rdbms-test.sh
├── alibaba-rdbms-test.sh
├── tencent-rdbms-test.sh
├── ibm-rdbms-test.sh
├── openstack-rdbms-test.sh
├── ncp-rdbms-test.sh
└── nhn-rdbms-test.sh
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:*****` | Basic auth credentials |
| `MAX_WAIT_SEC` | `3600` (create) / `1800` (delete) | Timeout per CSP (seconds) |
| `POLL_INTERVAL` | `30` (create) / `15` (delete) | Polling interval (seconds) |
| `VERBOSE` | `0` | Set to `1` for per-CSP full log dump |

```bash
# Example: custom Spider URL
SPIDER_URL=http://10.0.0.1:1024 ./run-all-csp-rdbms-tests.sh

# Example: verbose output
VERBOSE=1 ./run-all-csp-rdbms-tests.sh
```

## Logs & Results

각 실행마다 PID 기반 임시 디렉토리에 로그와 결과 파일이 저장됩니다.

```
/tmp/rdbms_results_<PID>/result_<csp>.txt   # pipe-separated result line
/tmp/rdbms_logs_<PID>/log_<csp>.txt         # per-CSP full output
```

실행 중 모니터링:

```bash
tail -f /tmp/rdbms_logs_<PID>/log_aws.txt
```

## CSP-Specific Notes

| CSP | Note |
|-----|------|
| AWS | SubnetGroup 생성을 위해 **다른 AZ의 서브넷 2개 이상** 필요 |
| Tencent | `DBInstanceSpec`은 메모리 크기(MB) 지정 (예: `8000` = 8GB) |
| NCP | StorageSize는 10GB 단위. G3(KVM) generation만 지원. Public 도메인은 생성 후 콘솔에서 별도 신청 필요 |
