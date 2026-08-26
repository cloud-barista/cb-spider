# CB-Spider RDBMS StorageType Test

각 CSP의 StorageType별로 RDBMS 인스턴스를 생성·검증하는 병렬 테스트 스위트입니다. 각 CSP의 `rdbmsmetainfo` API에서 지원 StorageType 목록을 동적으로 조회한 뒤, 옵션별로 인스턴스를 병렬 생성하고 반환된 StorageType이 요청과 일치하는지 검증합니다.

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### Pre-created Network Resources

RDBMS 생성 전에 각 CSP에 VPC/Subnet(및 AWS의 Security Group)이 미리 생성되어 있어야 합니다.

```bash
cd ..
./run-all-csp-network-prepare.sh
```

### Required Tools

- `bash` 3.2+
- `curl`
- `jq`

## Test Flow

각 CSP에 대해 다음 순서로 StorageType별 검증을 수행합니다:

1. **FetchStorageTypeOptions** — `GET /spider/rdbmsmetainfo?DBEngine=mysql&ConnectionName=...` — 지원하는 StorageType 목록을 동적으로 조회
2. **CreateRDBMS** (StorageType별 병렬) — `POST /spider/rdbms` — 옵션별로 `cb-mysql-st-<type>` 인스턴스 생성
3. **Poll Available** — `GET /spider/rdbms/{Name}` — Available 상태가 될 때까지 폴링 (기본 30초 간격, 최대 3600초)
4. **VerifyStorageType** — 반환된 StorageType이 요청과 일치하는지 확인 (`Result` 필드에 `PASS`/`FAIL` 기록)
5. **(옵션) AutoDelete** — `AUTO_DELETE=true`이면 검증 직후 인스턴스 자동 삭제. 기본값(`false`)에서는 `delete-all-csp-storage-type-rdbms.sh`로 별도 정리

Azure/IBM/NCP는 `SupportsStorageTypeSelection=false`로 StorageType 지정이 불가능하여 스크립트 실행 시 즉시 SKIP 처리됩니다.

### StorageType 선택 가능 여부

| CSP | SupportsStorageTypeSelection | 비고 |
|-----|------------------------------|------|
| AWS | ✅ | gp2, gp3, io1, io2 |
| GCP | ✅ | 머신 시리즈에 따라 자동 결정 (직접 선택 아님) |
| Alibaba | ✅ | cloud_auto, cloud_essd, cloud_essd2, cloud_essd3, local_ssd |
| Tencent | ✅ | local_ssd, CLOUD_HSSD, CLOUD_SSD, CLOUD_PREMIUM |
| OpenStack | ✅ | `__DEFAULT__`, `RBD` |
| NHN | ✅ | General HDD, General SSD |
| Azure | ❌ | SKIP |
| IBM | ❌ | SKIP |
| NCP | ❌ | SKIP |

## Configuration

```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:*****           # Basic auth (admin:<password>)
```

## How to Run Tests

### All CSPs in Parallel

```bash
./run-all-csp-storage-type-tests.sh
```

- 9개 CSP에 대해 병렬로 StorageType별 검증 실행 (Azure/IBM/NCP는 자동 SKIP)
- 완료 후 통합 결과 테이블 및 PASS/FAIL/SKIP 집계 출력

**Example output:**
```
================================================================================================================================
                               RDBMS StorageType Test Summary - All CSPs
================================================================================================================================
CSP          | StorageType(Req)     | StorageType(Ret)   | Result | DB Status      | Elapsed    | Reason
--------------------------------------------------------------------------------------------------------------------------------
AWS          | gp2                  | gp2                | PASS   | Available      | 5m43s      | -
AZURE        | N/A                  | N/A                | SKIP   | NOT_APPLICABLE | -          | SupportsStorageTypeSelection=false
GCP          | PD_SSD               | PD_SSD             | PASS   | Available      | 4m10s      | -
...
--------------------------------------------------------------------------------------------------------------------------------

Total: 22  PASS: 19  FAIL: 0  SKIP: 3
```

### Individual CSP

특정 CSP만 단독 실행:

```bash
./aws-storage-type-test.sh
./azure-storage-type-test.sh          # SKIP (StorageType 선택 불가)
./gcp-storage-type-test.sh
./alibaba-storage-type-test.sh
./tencent-storage-type-test.sh
./ibm-storage-type-test.sh            # SKIP (StorageType 선택 불가)
./openstack-storage-type-test.sh
./ncp-storage-type-test.sh            # SKIP (StorageType 선택 불가)
./nhn-storage-type-test.sh
```

단독 실행 시 결과/로그 디렉토리는 `RESULT_DIR` / `LOG_DIR` 환경변수로 지정하거나 기본값(`/tmp/st_results_<PID>`, `/tmp/st_logs_<PID>`)이 사용됩니다.

### Delete All Instances

```bash
./delete-all-csp-storage-type-rdbms.sh
```

- StorageTypeOptions를 다시 조회해 `cb-mysql-st-<type>` 이름의 인스턴스를 유추한 뒤 전체 CSP 병렬 삭제

## Script Structure

```
storage-type-test/
├── run-all-csp-storage-type-tests.sh    # Orchestrator: 전체 CSP 병렬 실행
├── delete-all-csp-storage-type-rdbms.sh # Orchestrator: 전체 CSP 병렬 삭제
├── common-storage-type-test.sh          # Common: Create → Poll Available → Get Info → Verify → (옵션) Delete
├── aws-storage-type-test.sh
├── gcp-storage-type-test.sh
├── alibaba-storage-type-test.sh
├── tencent-storage-type-test.sh
├── nhn-storage-type-test.sh
├── openstack-storage-type-test.sh
├── azure-storage-type-test.sh           # SKIP (StorageType 선택 불가)
├── ibm-storage-type-test.sh             # SKIP (StorageType 선택 불가)
└── ncp-storage-type-test.sh             # SKIP (StorageType 선택 불가)
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:****` | Basic auth credentials |
| `MAX_WAIT_SEC` | `3600` (create) / `1800` (delete) | Timeout per instance (seconds) |
| `POLL_INTERVAL` | `30` (create) / `15` (delete) | Polling interval (seconds) |
| `AUTO_DELETE` | `false` | `true`이면 검증 완료 직후 인스턴스 자동 삭제 |
| `VERBOSE` | `0` | `1`로 설정 시 CSP별 전체 로그 덤프 출력 |

```bash
# Example: verbose output
VERBOSE=1 ./run-all-csp-storage-type-tests.sh
```

## Result Format

결과 파일(`result_<csp>_<storagetype>.txt`)은 파이프(|) 구분 7개 필드:

```
CSP|StorageType(Requested)|StorageType(Returned)|Result|DB_Status|Elapsed|Reason
```

| 필드 | 설명 |
|------|------|
| `CSP` | CSP 이름 (예: AWS) |
| `StorageType(Requested)` | 요청한 StorageType 값 |
| `StorageType(Returned)` | 생성 후 조회된 StorageType 값 (`N/A`는 생성 실패) |
| `Result` | `PASS` / `FAIL` / `SKIP` |
| `DB_Status` | 인스턴스 상태 (`Available`, `CREATE_ERROR`, `TIMEOUT`, `NOT_APPLICABLE` 등) |
| `Elapsed` | 경과 시간 |
| `Reason` | 특이사항 (예: SupportsStorageTypeSelection=false, cloud_auto 자동 선택 등) |

## API Reference

| Operation | Method | Path |
|-----------|--------|------|
| FetchStorageTypeOptions | `GET` | `/spider/rdbmsmetainfo?DBEngine=mysql&ConnectionName=` |
| CreateRDBMS | `POST` | `/spider/rdbms` |
| GetRDBMS | `GET` | `/spider/rdbms/{Name}?ConnectionName=` |
| DeleteRDBMS | `DELETE` | `/spider/rdbms/{Name}` |

## Logs & Results

```
/tmp/st_test_<PID>/results/result_<csp>_<storagetype>.txt
/tmp/st_test_<PID>/logs/log_<csp>.txt
```

전체 삭제 실행 시:
```
/tmp/st_del_<PID>/results/result_<csp>_<storagetype>.txt
/tmp/st_del_<PID>/logs/log_<csp>.txt
```

실행 중 모니터링:

```bash
tail -f /tmp/st_test_<PID>/logs/log_aws.txt
```

## CSP-Specific Notes

### AWS

| StorageType | DBSpec | StorageSize | Iops |
|-------------|---------------|-------------|------|
| gp2 | db.t3.medium | 100 GB | - |
| gp3 | db.t3.medium | 100 GB | - |
| io1 | db.t3.medium | 100 GB | **3000 (필수)** |
| io2 | db.t3.medium | 100 GB | **3000 (필수)** |

- StorageTypeOptions: metainfo API에서 동적으로 조회
- Connection: `aws-config01`
- Region: `ap-southeast-2` / Zone: `ap-southeast-2a`
- SubnetNames: `subnet-01`, `subnet-02` (서로 다른 AZ, SubnetGroup 생성 필수)
- SecurityGroupNames: `sg-01`

**특이사항**
- io1/io2: `Iops` 필드 필수(Iops:AWS io1/io2 전용)
- StorageSize: io1/io2는 최소 100 GB

---

### GCP

GCP는 머신 시리즈에 따라 StorageType이 고정되어 5개 케이스를 직접 지정.

| StorageType | DBSpec | StorageSize | Edition | Machine Series |
|-------------|---------------|-------------|---------|---------------|
| PD_SSD | db-perf-optimized-N-4 | 10 GB | Enterprise Plus | N2 |
| PD_SSD | db-custom-2-8192 | 10 GB | Enterprise | Shared/Dedicated core |
| PD_HDD | db-custom-2-8192 | 10 GB | Enterprise | Shared/Dedicated core |
| HYPERDISK_BALANCED | db-c4a-highmem-4 | 20 GB | Enterprise Plus | C4A |
| HYPERDISK_BALANCED | db-custom-N4-2-4096 | 20 GB | Enterprise | N4 |

- Connection: `gcp-iowa-config`
- Region: `us-central1` / Zone: `us-central1-a`
- VPCName: `vpc-01`

**특이사항**
- 머신 시리즈 ↔ StorageType 불일치 시 Spider 드라이버에서 에러 반환 (e.g. N2에 HYPERDISK_BALANCED 요청시 에러)
- N2/C4A 머신 타입은 Edition=ENTERPRISE_PLUS 자동 설정 (드라이버 내부 처리)
- N2(`db-perf-optimized-N-*`), C4A(`db-c4a-highmem-*`), N4(`db-custom-N4-*`): StorageType은 API에 전달하지 않음 (머신 시리즈에 의해 자동 결정)
- GCP metainfo StorageTypeOptions에 HYPERDISK_BALANCED가 포함되나, 직접 지정이 아닌 머신 시리즈 선택으로 결정됨
- HYPERDISK_BALANCED 최소 StorageSize: 20 GB

---

### Alibaba

| StorageType | DBSpec | StorageSize | 비고 |
|-------------|---------------|-------------|------|
| cloud_auto | mysql.n4.large.1 | 20 GB | |
| cloud_essd | mysql.n4.large.1 | 20 GB | ESSD PL1 |
| cloud_essd2 | mysql.n2.small.2c | **500 GB** | ESSD PL2, 최소 500 GB |
| cloud_essd3 | mysql.n2.small.2c | **1500 GB** | ESSD PL3, 최소 1500 GB |
| local_ssd | rds.mysql.t1.small | 20 GB | Premium Local SSD, rds.mysql.* 계열 전용 |

- StorageTypeOptions: metainfo API에서 동적으로 조회
- Connection: `alibaba-beijing-config`
- Region: `cn-beijing` / Zone: `cn-beijing-f`
- SubnetNames: `subnet-01`

**특이사항**
- cloud_essd2/3는 mysql.n4.* 계열과 호환되지 않음 → mysql.n2.small.2c 사용
- local_ssd는 rds.mysql.* 계열 전용 (mysql.n4.* 사용 시 InvalidInstanceLevel.DiskType 에러)

---

### Tencent

| StorageType | DBSpec | StorageSize |
|-------------|---------------|-------------|
| local_ssd | 8000 (MB) | 50 GB |
| CLOUD_HSSD | 8000 (MB) | 50 GB |
| CLOUD_SSD | 8000 (MB) | 50 GB |
| CLOUD_PREMIUM | 8000 (MB) | 50 GB |

- StorageTypeOptions: metainfo API에서 동적으로 조회
- Connection: `tencent-beijing3-config`
- Region: `ap-beijing` / Zone: `ap-beijing-3`
- SubnetNames: `subnet-01`
- DBSpec: 메모리 크기 MB 단위 지정 (8000 = 8 GB)

**특이사항**
- 동시 주문 거부 발생 가능 (OperationDenied.OtherOderInProcess) → 드라이버 자동 재시도 처리

---

### IBM — SKIP

- SupportsStorageTypeSelection=false
- Connection: `ibm-us-east-1-config`
- Region: `us-east` / Zone: `us-east-1`
- DBEngineVersion: `8.4`
- IBM Cloud Databases는 스토리지 타입 선택 기능 부재
- 테스트 스크립트 실행 시 즉시 SKIP 처리

---

### OpenStack

| StorageType | DBSpec | StorageSize |
|-------------|---------------|-------------|
| __DEFAULT__ | m1.small | 20 GB |
| RBD | m1.small | 20 GB |

- Connection: `openstack-config01`
- Region: `RegionOne` / Zone: `nova`
- DBEngineVersion: `5.7.29`

---

### NHN

| StorageType | DBSpec | StorageSize |
|-------------|---------------|-------------|
| General HDD | m2.c2m4 | 20 GB |
| General SSD | m2.c2m4 | 20 GB |

- StorageTypeOptions: NHN Cloud RDS API에서 동적으로 조회
- Connection: `nhn-korea-pangyo1-config`
- Region: `KR1` / Zone: `kr-pub-a`
- SubnetNames: `subnet-01`
- `NHNAutoOpenDBSecurityGroup: true` — NHN Cloud RDS는 VPC Security Group과는 별개인 전용 "DB Security Group"이 있어야 외부 접속이 가능하므로, 시험 편의를 위해 이 옵션으로 전체 개방(`0.0.0.0/0`) DB Security Group을 자동 생성/삭제합니다. 운영 환경에서는 사용을 권장하지 않습니다.

---

### Azure — SKIP

- SupportsStorageTypeSelection=false
- Connection: `azure-koreacentral-config`
- Region: `koreacentral` / Zone: `1`
- Azure MySQL Flexible Server의 storageSku는 read-only, Azure가 자동 설정
- 테스트 스크립트 실행 시 즉시 SKIP 처리

---

### NCP — SKIP

- SupportsStorageTypeSelection=false
- Connection: `ncp-korea1-config`
- Region: `KR` / Zone: `KR-1`
- NCP MySQL G3은 SSD 자동 적용, DataStorageTypeCode 지정 불가
- 테스트 스크립트 실행 시 즉시 SKIP 처리

## 시험 결과

### 2026-08-03

Note: AWS `standard` type removed (deprecated); IBM changed to `SupportsStorageTypeSelection=false`; Tencent tested with `local_ssd` only.

```
================================================================================================================================
                               RDBMS StorageType Test Summary - All CSPs
================================================================================================================================
CSP          | StorageType(Req)     | StorageType(Ret)   | Result | DB Status      | Elapsed    | Reason
--------------------------------------------------------------------------------------------------------------------------------
[*] cloud_auto: Alibaba auto-select type - CSP picks the optimal cloud storage type at provisioning time
--------------------------------------------------------------------------------------------------------------------------------
AWS          | gp2                  | gp2                | PASS   | Available      | 5m43s      | -
AWS          | gp3                  | gp3                | PASS   | Available      | 5m44s      | -
AWS          | io1                  | io1                | PASS   | Available      | 5m13s      | -
AWS          | io2                  | io2                | PASS   | Available      | 5m12s      | -
AZURE        | N/A                  | N/A                | SKIP   | NOT_APPLICABLE | -          | SupportsStorageTypeSelection=false: storageSku is read-only, set automatically by Azure
GCP          | HYPERDISK_BALANCED   | HYPERDISK_BALANCED | PASS   | Available      | 3m39s      | -
GCP          | HYPERDISK_BALANCED   | HYPERDISK_BALANCED | PASS   | Available      | 3m9s       | -
GCP          | PD_HDD               | PD_HDD             | PASS   | Available      | 3m55s      | -
GCP          | PD_SSD               | PD_SSD             | PASS   | Available      | 4m10s      | -
GCP          | PD_SSD               | PD_SSD             | PASS   | Available      | 4m56s      | -
ALIBABA      | cloud_auto[*]        | general_essd       | PASS   | Available      | 3m52s      | cloud_auto: auto-select type, CSP chose 'general_essd'
ALIBABA      | cloud_essd           | cloud_essd         | PASS   | Available      | 3m36s      | -
ALIBABA      | cloud_essd2          | cloud_essd2        | PASS   | Available      | 3m51s      | -
ALIBABA      | cloud_essd3          | cloud_essd3        | PASS   | Available      | 3m4s       | -
ALIBABA      | local_ssd            | local_ssd          | PASS   | Available      | 4m43s      | -
TENCENT      | local_ssd            | local_ssd          | PASS   | Available      | 4m38s      | -
IBM          | N/A                  | N/A                | SKIP   | NOT_APPLICABLE | -          | SupportsStorageTypeSelection=false: IBM Cloud Databases has no storage type selection
OPENSTACK    | __DEFAULT__          | NA                 | PASS   | Available      | 5m0s       | OpenStack Trove does not expose StorageType post-creation; Available=PASS
OPENSTACK    | RBD                  | NA                 | PASS   | Available      | 5m22s      | OpenStack Trove does not expose StorageType post-creation; Available=PASS
NCP          | N/A                  | N/A                | SKIP   | NOT_APPLICABLE | -          | SupportsStorageTypeSelection=false: NCP G3 applies SSD automatically, StorageType cannot be specified
NHN          | General HDD          | General HDD        | PASS   | Available      | 10m7s      | -
NHN          | General SSD          | General SSD        | PASS   | Available      | 11m7s      | -
--------------------------------------------------------------------------------------------------------------------------------
Total: 22  PASS: 19  FAIL: 0  SKIP: 3
```
