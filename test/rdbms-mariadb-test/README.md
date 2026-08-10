# CB-Spider RDBMS API Test (MariaDB)

Automated test suite for CB-Spider RDBMS API — creates MariaDB instances across CSPs in parallel, waits until each becomes available, then collects and displays a unified result table.

**대상 CSP: MariaDB를 지원하는 AWS · Alibaba · OpenStack · NHN **
**미지원 CSP: Azure / GCP / IBM / NCP / Tencent **
  - Tencent: 가이드에는 제공, 실제 API 호출시 "Not Supported" 관련 에러 반환으로 현재는 지원하지 않음

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
| Alibaba | `alibaba-beijing-config` | `cn-beijing` | `cn-beijing-f` |
| OpenStack | `openstack-config01` | `RegionOne` | `nova` |
| NHN | `nhn-korea-pangyo1-config` | `KR1` | `kr-pub-a` |

> Azure / GCP / IBM / NCP / Tencent는 MariaDB 미지원으로 이 테스트 스위트에서 제외되어 커넥션 등록이 필요 없습니다.

### Pre-created Network Resources

RDBMS 생성 전에 각 CSP에 VPC와 서브넷이 미리 생성되어 있어야 합니다.

```bash
./run-all-csp-network-prepare.sh
```

### Required Tools

- `bash` 3.2+
- `curl`
- `jq`

## MariaDB Engine Support by CSP

| CSP | MariaDB 지원 | Tag 지원 | 버전 | 비고 |
|-----|------------|---------|------|------|
| AWS | ✅ | ✅ | `10.6` | Amazon RDS for MariaDB |
| Alibaba | ✅ | ✅ | `10.6` | AliCloud RDS for MariaDB |
| OpenStack | ⚠️ | ❌ | `10.6` | **현재 이 환경은 MariaDB Trove datastore 구축 전 상태** (설치되면 정상 동작 예상) — 시험 스크립트는 그대로 유지 |
| NHN | ✅ | ❌ | `MARIADB_V10622` | NHN RDS for MariaDB |


## RDBMS Instance Configuration

인스턴스 이름은 `cb-spider-mariadb-test`입니다.

| CSP | Engine | Version | Spec | Storage |
|-----|--------|---------|------|---------|
| AWS | mariadb | 10.6 | db.t3.medium | 100GB |
| Alibaba | mariadb | 10.6 | rds.mariadb.s4.large | 20GB |
| OpenStack | mariadb | 10.6 | m1.small | 20GB (Trove 구축 후 정상 동작 예상) |
| NHN | mariadb | MARIADB_V10622 | m2.c2m4 | 20GB |

## Usage

### Quick Start (Full Test Suite)

```bash
# 전체 테스트 순차 실행 (네트워크 준비 → StorageType 검증 → RDBMS 생성 → DB 관리 → Tag → 삭제 → 네트워크 정리)
./all_test.sh

# 실패 시 즉시 중단
STOP_ON_FAIL=1 ./all_test.sh
```

### Step-by-step

```bash
# 1. 네트워크 사전 자원 생성
./run-all-csp-network-prepare.sh

# 2. StorageType 검증 테스트
./storage-type-test/run-all-csp-storage-type-tests.sh
./storage-type-test/delete-all-csp-storage-type-rdbms.sh

# 3. RDBMS 인스턴스 생성 및 검증
./run-all-csp-rdbms-tests.sh

# 4. DB 관리 테스트 (CreateDB / ListDB / DeleteDB)
./database-test/run-all-csp-database-tests.sh

# 5. Tag 관리 테스트
./tag-test/run-all-csp-rdbms-tag-tests.sh

# 6. RDBMS 인스턴스 삭제
./delete-all-csp-rdbms.sh

# 7. 네트워크 자원 삭제
./delete-all-csp-network.sh
```

### Per-CSP Execution

```bash
./aws-rdbms-test.sh
./alibaba-rdbms-test.sh
./openstack-rdbms-test.sh
./nhn-rdbms-test.sh
```

### Configuration

| 환경변수 | 기본값 | 설명 |
|---------|--------|------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:****` | Basic auth credentials |
| `MAX_WAIT_SEC` | `3600` | 인스턴스 Available 대기 최대 시간 (초) |
| `POLL_INTERVAL` | `30` | 상태 폴링 간격 (초) |
| `STOP_ON_FAIL` | `0` | `1`이면 첫 번째 실패 시 중단 |
| `VERBOSE` | `0` | `1`이면 CSP별 상세 로그 출력 |

## Script Structure

```
rdbms-mariadb-test/
├── all_test.sh                          # 전체 테스트 수트
├── common-rdbms-test.sh                 # 공통: Create → Poll → Get Info
├── common-rdbms-delete.sh               # 공통: Delete → Poll until gone
├── common-network-prepare.sh            # 공통: VPC/Subnet/SG 생성
├── common-network-cleanup.sh            # 공통: VPC/Subnet/SG 삭제
├── run-all-csp-rdbms-tests.sh           # 전체 CSP 병렬 RDBMS 생성
├── delete-all-csp-rdbms.sh              # 전체 CSP 병렬 RDBMS 삭제
├── run-all-csp-network-prepare.sh       # 전체 CSP 네트워크 사전 준비
├── delete-all-csp-network.sh            # 전체 CSP 네트워크 정리
├── aws-rdbms-test.sh / alibaba-rdbms-test.sh / openstack-rdbms-test.sh / nhn-rdbms-test.sh   # MariaDB 지원 4개 CSP만 존재
├── database-test/
│   ├── common-database-test.sh
│   ├── run-all-csp-database-tests.sh          # 실행 대상: AWS/Alibaba/OpenStack/NHN
│   └── aws-database-test.sh / alibaba-database-test.sh / openstack-database-test.sh / nhn-database-test.sh
├── storage-type-test/
│   ├── common-storage-type-test.sh
│   ├── run-all-csp-storage-type-tests.sh      # 실행 대상: AWS/Alibaba/OpenStack/NHN
│   ├── delete-all-csp-storage-type-rdbms.sh
│   └── aws-storage-type-test.sh / alibaba-storage-type-test.sh / openstack-storage-type-test.sh / nhn-storage-type-test.sh
└── tag-test/
    ├── common-rdbms-tag-test.sh
    ├── run-all-csp-rdbms-tag-tests.sh          # 실행 대상: AWS/Alibaba (SupportsTag=true AND MariaDB 지원)
    └── aws-rdbms-tag-test.sh / alibaba-rdbms-tag-test.sh
```
