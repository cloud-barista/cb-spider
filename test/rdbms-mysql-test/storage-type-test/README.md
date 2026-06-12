# CB-Spider RDBMS StorageType Test

각 CSP의 StorageType별로 RDBMS 인스턴스를 생성·검증하는 병렬 테스트 스위트.

## 실행

```bash
# 전체 CSP 병렬 실행
./run-all-csp-storage-type-tests.sh

# CSP별 단독 실행
./aws-storage-type-test.sh
./gcp-storage-type-test.sh
# ...

# 전체 삭제
./delete-all-csp-storage-type-rdbms.sh
```

## 스크립트 구조

```
.
├── run-all-csp-storage-type-tests.sh   # 전체 CSP 병렬 실행
├── delete-all-csp-storage-type-rdbms.sh # 전체 CSP 병렬 삭제
├── common-storage-type-test.sh          # 공통: Create → Poll Available → Get Info
├── aws-storage-type-test.sh
├── gcp-storage-type-test.sh
├── alibaba-storage-type-test.sh
├── tencent-storage-type-test.sh
├── ibm-storage-type-test.sh
├── nhn-storage-type-test.sh
├── openstack-storage-type-test.sh
├── azure-storage-type-test.sh           # SKIP (StorageType 선택 불가)
└── ncp-storage-type-test.sh             # SKIP (StorageType 선택 불가)
```

---

## CSP별 시험 설정

### AWS

| StorageType | DBInstanceSpec | StorageSize | Iops |
|-------------|---------------|-------------|------|
| gp2 | db.t3.medium | 100 GB | - |
| gp3 | db.t3.medium | 100 GB | - |
| io1 | db.t3.medium | 100 GB | **3000 (필수)** |
| io2 | db.t3.medium | 100 GB | **3000 (필수)** |
| standard | db.t3.medium | 100 GB | - |

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

| StorageType | DBInstanceSpec | StorageSize | Edition | Machine Series |
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

| StorageType | DBInstanceSpec | StorageSize | 비고 |
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

| StorageType | DBInstanceSpec | StorageSize |
|-------------|---------------|-------------|
| local_ssd | 8000 (MB) | 50 GB |
| CLOUD_HSSD | 8000 (MB) | 50 GB |
| CLOUD_SSD | 8000 (MB) | 50 GB |
| CLOUD_PREMIUM | 8000 (MB) | 50 GB |

- StorageTypeOptions: metainfo API에서 동적으로 조회
- Connection: `tencent-beijing6-config`
- Region: `ap-beijing` / Zone: `ap-beijing-6`
- SubnetNames: `subnet-01`
- DBInstanceSpec: 메모리 크기 MB 단위 지정 (8000 = 8 GB)

**특이사항**
- 동시 주문 거부 발생 가능 (OperationDenied.OtherOderInProcess) → 드라이버 자동 재시도 처리

---

### IBM

| StorageType | DBInstanceSpec | StorageSize |
|-------------|---------------|-------------|
| standard | multitenant | 30 GB |

- Connection: `ibm-us-east-1-config`
- Region: `us-east` / Zone: `us-east-1`
- DBEngineVersion: `8.4`

---

### OpenStack

| StorageType | DBInstanceSpec | StorageSize |
|-------------|---------------|-------------|
| __DEFAULT__ | m1.small | 20 GB |
| RBD | m1.small | 20 GB |

- Connection: `openstack-config01`
- Region: `RegionOne` / Zone: `nova`
- DBEngineVersion: `5.7.29`

---

### NHN

| StorageType | DBInstanceSpec | StorageSize |
|-------------|---------------|-------------|
| General HDD | m2.c2m4 | 20 GB |
| General SSD | m2.c2m4 | 20 GB |

- StorageTypeOptions: NHN Cloud RDS API에서 동적으로 조회
- Connection: `nhn-korea-pangyo1-config`
- Region: `KR1` / Zone: `kr-pub-a`
- SubnetNames: `subnet-01`

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

---

## StorageType 선택 가능 여부 요약

| CSP | SupportsStorageTypeSelection | 비고 |
|-----|------------------------------|------|
| AWS | ✅ | gp2, gp3, io1, io2, standard |
| GCP | ✅ | 머신 시리즈에 따라 자동 결정 (직접 선택 아님) |
| Alibaba | ✅ | cloud_auto, cloud_essd, cloud_essd2, cloud_essd3, local_ssd |
| Tencent | ✅ | local_ssd, CLOUD_HSSD, CLOUD_SSD, CLOUD_PREMIUM |
| IBM | ✅ | standard |
| OpenStack | ✅ | __DEFAULT__, RBD  |
| NHN | ✅ | General HDD, General SSD |
| Azure | ❌ | SKIP |
| NCP | ❌ | SKIP |

---

## 시험 결과

### 2026-06-12

```
================================================================================================================================
                                RDBMS StorageType Test Summary - All CSPs
================================================================================================================================
CSP          | StorageType(Req)     | StorageType(Ret)   | Result | DB Status      | Elapsed    | Reason
--------------------------------------------------------------------------------------------------------------------------------
 [*] cloud_auto: Alibaba auto-select type - CSP picks the optimal cloud storage type at provisioning time
--------------------------------------------------------------------------------------------------------------------------------
AWS          | gp2                  | gp2                | PASS   | Available      | 4m35s      | -
AWS          | gp3                  | gp3                | PASS   | Available      | 4m36s      | -
AWS          | io1                  | io1                | PASS   | Available      | 4m36s      | -
AWS          | io2                  | io2                | PASS   | Available      | 4m36s      | -
AWS          | standard             | standard           | PASS   | Available      | 6m7s       | -
AZURE        | N/A                  | N/A                | SKIP   | NOT_APPLICABLE | -          | SupportsStorageTypeSelection=false: storageSku is read-only, set automatically by Azure
GCP          | HYPERDISK_BALANCED   | HYPERDISK_BALANCED | PASS   | Available      | 3m40s      | -
GCP          | HYPERDISK_BALANCED   | HYPERDISK_BALANCED | PASS   | Available      | 4m26s      | -
GCP          | PD_HDD               | PD_HDD             | PASS   | Available      | 4m10s      | -
GCP          | PD_SSD               | PD_SSD             | PASS   | Available      | 3m55s      | -
GCP          | PD_SSD               | PD_SSD             | PASS   | Available      | 3m40s      | -
ALIBABA      | cloud_auto[*]        | general_essd       | PASS   | Available      | 3m0s       | cloud_auto: auto-select type, CSP chose 'general_essd'
ALIBABA      | cloud_essd           | cloud_essd         | PASS   | Available      | 3m2s       | -
ALIBABA      | cloud_essd2          | cloud_essd2        | PASS   | Available      | 3m18s      | -
ALIBABA      | cloud_essd3          | cloud_essd3        | PASS   | Available      | 3m36s      | -
ALIBABA      | local_ssd            | local_ssd          | PASS   | Available      | 4m54s      | -
TENCENT      | CLOUD_HSSD           | CLOUD_HSSD         | PASS   | Available      | 9m14s      | -
TENCENT      | CLOUD_PREMIUM        | CLOUD_PREMIUM      | PASS   | Available      | 9m22s      | -
TENCENT      | CLOUD_SSD            | CLOUD_SSD          | PASS   | Available      | 9m15s      | -
TENCENT      | local_ssd            | local_ssd          | PASS   | Available      | 6m15s      | -
IBM          | standard             | standard           | PASS   | Available      | 39m6s      | -
OPENSTACK    | __DEFAULT__          | NA                 | PASS   | Available      | 4m57s      | OpenStack Trove does not expose StorageType post-creation; Available=PASS
OPENSTACK    | RBD                  | NA                 | PASS   | Available      | 4m59s      | OpenStack Trove does not expose StorageType post-creation; Available=PASS
NCP          | N/A                  | N/A                | SKIP   | NOT_APPLICABLE | -          | SupportsStorageTypeSelection=false: NCP G3 applies SSD automatically, StorageType cannot be specified
NHN          | General HDD          | General HDD        | PASS   | Available      | 12m5s      | -
NHN          | General SSD          | General SSD        | PASS   | Available      | 12m35s     | -
--------------------------------------------------------------------------------------------------------------------------------
Total: 26  PASS: 24  FAIL: 0  SKIP: 2
```
