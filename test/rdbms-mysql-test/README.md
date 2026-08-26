# CB-Spider RDBMS API Test

Automated test suite for CB-Spider RDBMS API — creates MySQL instances across 9 CSPs in parallel, waits until each becomes available, then collects and displays a unified result table.

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### CSP Connection Configuration

Before running tests, register connection names for each CSP in CB-Spider.

| CSP | Connection Name | Region | Zone |
|-----|----------------|--------|------|
| AWS | `aws-config01` | `ap-southeast-2` | `ap-southeast-2a` |
| Azure | `azure-koreacentral-config` | `koreacentral` | `1` |
| GCP | `gcp-iowa-config` | `us-central1` | `us-central1-a` |
| Alibaba | `alibaba-beijing-config` | `cn-beijing` | `cn-beijing-f` |
| Tencent | `tencent-beijing3-config` | `ap-beijing` | `ap-beijing-3` |
| IBM | `ibm-us-east-1-config` | `us-east` | `us-east-1` |
| OpenStack | `openstack-config01` | `RegionOne` | `nova` |
| NCP | `ncp-korea1-config` | `KR` | `KR-1` |
| NHN | `nhn-korea-pangyo1-config` | `KR1` | `kr-pub-a` |

### Pre-created Network Resources

RDBMS 생성 전에 각 CSP에 VPC와 서브넷이 미리 생성되어 있어야 합니다. AWS는 Security Group도 미리 생성되어 있어야 합니다.

| CSP | VPC | Subnet | Security Group | 비고 |
|-----|-----|--------|-----------------|------|
| AWS | `vpc-01` | `subnet-01`, `subnet-02` | `sg-01` | 서로 다른 AZ의 서브넷 2개 필수 (SubnetGroup 요건) |
| Azure | `vpc-01` | `subnet-01` | | 서브넷 미사용 |
| GCP | `vpc-01` | `subnet-01` | | 서브넷 미사용 |
| Alibaba | `vpc-01` | `subnet-01` | | |
| Tencent | `vpc-01` | `subnet-01` | | |
| IBM | `vpc-01` | `subnet-01` | | 서브넷 미사용 |
| OpenStack | `vpc-01` | `subnet-01` | | 서브넷 미사용 |
| NCP | `vpc-01` | `subnet-01` | | |
| NHN | `vpc-01` | `subnet-01` | | |

아래 스크립트로 9개 CSP의 VPC/Subnet(및 AWS의 Security Group)을 한 번에 생성/삭제할 수 있습니다. 두 스크립트 모두 이미 존재하는 자원은 건너뛰므로(idempotent) 반복 실행해도 안전합니다.

```bash
# 전체 CSP 사전 자원 생성 (병렬)
./run-all-csp-network-prepare.sh

# 전체 CSP 사전 자원 삭제 (병렬)
# RDBMS 인스턴스가 먼저 삭제된 이후에 실행해야 합니다 (VPC/SG가 사용 중이면 삭제 실패)
./delete-all-csp-network.sh
```

특정 CSP만 단독 실행하려면 `<csp>-network-prepare.sh`를 직접 실행합니다 (예: `./aws-network-prepare.sh`). AWS의 AZ는 `AWS_AZ1`/`AWS_AZ2` 환경변수로 override 가능합니다 (기본값: `ap-southeast-2a`/`ap-southeast-2b`).

내부적으로는 다음과 같이 CB-Spider REST API를 호출합니다:

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

# Security Group 생성 예시 (AWS, MySQL 3306 인바운드 허용)
curl -u admin:***** -sX POST http://localhost:1024/spider/securitygroup \
  -H 'Content-Type: application/json' \
  -d '{
    "ConnectionName": "aws-config01",
    "ReqInfo": {
      "Name": "sg-01",
      "VPCName": "vpc-01",
      "SecurityRules": [
        {"Direction": "inbound", "IPProtocol": "TCP", "FromPort": "3306", "ToPort": "3306"}
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

StorageType은 지정하지 않으며, CSP 기본값으로 생성됩니다. 결과 테이블의 Storage 컬럼에 `크기|타입` 형태로 표시됩니다 (예: `100GB|gp2`).

| CSP | Engine Version | Instance Spec | Storage | Subnet Required |
|-----|---------------|---------------|---------|-----------------|
| AWS | 8.0 | db.t3.medium | 100GB | ✅ (2개, 다른 AZ) |
| Azure | 8.0.21 | Standard_B1ms | 20GB | 미사용 |
| GCP | 8.0 | db-custom-2-8192 | 20GB | 미사용 |
| Alibaba | 8.0 | mysql.n4.large.1 | 20GB | ✅ |
| Tencent | 8.0 | 8000 (MB) | 50GB | ✅ |
| IBM | 8.4 | multitenant | 30GB | 미사용 |
| OpenStack | 5.7.29 | m1.small | 20GB | 미사용 |
| NCP | 8.0.36 | SVR.VDBAS.AMD.STAND.C002.M008.NET.SSD.B050.G003 | CSP 관리 | ✅ |
| NHN | MYSQL_V8408 | m2.c2m4 | 20GB | ✅ |

> **NHN 참고**: NHN Cloud RDS는 VPC Security Group과는 별개인 전용 "DB Security Group"이 있어야 외부 접속이 가능합니다. `nhn-rdbms-test.sh`는 `"NHNAutoOpenDBSecurityGroup": true`를 설정해 전체 개방(`0.0.0.0/0`) DB Security Group을 자동 생성하고, 인스턴스 삭제 시 함께 자동 삭제되도록 합니다 — 시험 편의를 위한 옵션이며, 운영 환경에서는 NHN 콘솔/API로 특정 CIDR만 허용하는 DB Security Group을 직접 구성하는 것을 권장합니다.

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

### 0. Prepare Network Prerequisites (VPC/Subnet, AWS Security Group)

RDBMS 생성 테스트를 실행하기 전에 먼저 실행합니다 (1회만 실행하면 됨, idempotent):

```bash
./run-all-csp-network-prepare.sh
```

### Create: All CSPs in Parallel

```bash
./run-all-csp-rdbms-tests.sh
```

- 9개 CSP에 동시 RDBMS 생성 (백그라운드 병렬 실행)
- 각 CSP별 Available 상태까지 대기 (최대 60분)
- 완료 후 통합 결과 테이블 출력

**Example output:**
```
CSP          | Status      | Engine   | Version      | Spec                     | Storage                  | Endpoint                                 | PublicAccess | Elapsed
---
AWS          | Available   | mysql    | 8.0.45       | db.t3.medium             | 100GB|gp2                | xxx.rds.amazonaws.com:3306               | true         | 8m39s
AZURE        | Available   | mysql    | 8.0.21       | Standard_B1ms            | 20GB|N/A                 | xxx.mysql.database.azure.com:3306        | true         | 5m35s
GCP          | Available   | mysql    | 8.0          | db-custom-2-8192         | 20GB|PD_SSD              | xxx.cloudsql.google.com:3306             | true         | 3m58s
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
# Prepare network prerequisites
./aws-network-prepare.sh
./azure-network-prepare.sh
./gcp-network-prepare.sh
./alibaba-network-prepare.sh
./tencent-network-prepare.sh
./ibm-network-prepare.sh
./openstack-network-prepare.sh
./ncp-network-prepare.sh
./nhn-network-prepare.sh

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

단독 실행 시에는 `RESULT_DIR` 환경변수를 지정하거나 기본값(`/tmp/rdbms_results`, network prepare/cleanup은 `/tmp/rdbms_network_results`)이 사용됩니다.

## Script Structure

```
.
├── run-all-csp-network-prepare.sh  # Orchestrator: 전체 VPC/Subnet/SG 사전 생성 (병렬)
├── delete-all-csp-network.sh       # Orchestrator: 전체 VPC/Subnet/SG 사전 자원 삭제 (병렬)
├── common-network-prepare.sh       # Common: VPC/Subnet 생성 → (옵션) SG 생성
├── common-network-cleanup.sh       # Common: (옵션) SG 삭제 → VPC/Subnet 삭제
├── aws-network-prepare.sh
├── azure-network-prepare.sh
├── gcp-network-prepare.sh
├── alibaba-network-prepare.sh
├── tencent-network-prepare.sh
├── ibm-network-prepare.sh
├── openstack-network-prepare.sh
├── ncp-network-prepare.sh
├── nhn-network-prepare.sh
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
| `AWS_AZ1` / `AWS_AZ2` | `ap-southeast-2a` / `ap-southeast-2b` | AZs used for AWS `subnet-01`/`subnet-02` in `aws-network-prepare.sh` |

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
| AWS | SubnetGroup 생성을 위해 **다른 AZ의 서브넷 2개 이상** 필요. Security Group `sg-01` 사전 생성 필요 |
| Tencent | `DBSpec`은 메모리 크기(MB) 지정 (예: `8000` = 8GB) |
| IBM | StorageType 지정 불가 (SupportsStorageTypeSelection=false). |
| NCP | StorageSize/StorageType 지정 불가 (CSP 자동 관리). G3(KVM) generation만 지원. Public 도메인은 생성 후 콘솔에서 별도 신청 필요 |

## 시험 결과

### 2026-08-03

```
=================================================================================================================================================================================
                                             RDBMS CREATE & INFO TEST SUMMARY - ALL CSPs
=================================================================================================================================================================================
CSP          | Status      | Engine   | Version      | Spec                     | Storage                  | Endpoint                                 | PublicAccess | Elapsed
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
AWS          | Available   | mysql    | 8.0.46       | db.t3.medium             | 100GB|gp2                | cb-spider-mysql-test-***.ap-southeast-2.rds.amazonaws.com:3306          | true         | 5m11s
AZURE        | Available   | mysql    | 8.0.21       | Standard_B1ms            | 20GB|Premium_LRS         | cb-spider-mysql-test-***.mysql.database.azure.com:3306                  | true         | 5m38s
GCP          | Available   | mysql    | 8.0          | db-custom-2-8192         | 20GB|PD_SSD              | *.*.*.*:3306                             | true         | 4m0s
ALIBABA      | Available   | mysql    | 8.0          | mysql.n4.large.1         | 20GB|cloud_essd          | *.*.*.*:3306                             | true         | 2m32s
TENCENT      | Available   | mysql    | 8.0          | 8000                     | 50GB|local_ssd           | bj-cdb-***.sql.tencentcdb.com:20137      | true         | 7m48s
IBM          | Available   | mysql    | 8.4          | multitenant              | 30GB|NA                  | ***.databases.appdomain.cloud:32251      | true         | 5m31s
OPENSTACK    | Available   | mysql    | 5.7.29       | m1.small                 | 20GB|NA                  | *.*.*.*:3306                             | true         | 4m31s
NCP          | Available   | mysql    | MYSQL8.0.36  | SVR.VDBAS.AMD.STAND.C002.M008.NET.SSD.B050.G003 | 10GB|SSD                 | db-***.vpc-cdb.ntruss.com:3306           | N/A          | 11m24s
NHN          | Available   | mysql    | MYSQL_V8408  | m2.c2m4                  | 20GB|General SSD         | ***.external.kr1.mysql.rds.nhncloudservice.com:3306                     | true         | 11m22s
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Failed : 0
```
