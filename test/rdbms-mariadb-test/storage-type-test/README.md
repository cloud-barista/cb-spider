# CB-Spider RDBMS StorageType Test (MariaDB)

MariaDB를 지원하는 각 CSP(AWS, Alibaba, OpenStack, NHN)의 StorageType별로 RDBMS 인스턴스를 생성·검증하는 병렬 테스트 스위트입니다. 각 CSP의 `rdbmsmetainfo` API에서 지원 StorageType 목록을 동적으로 조회한 뒤, 옵션별로 인스턴스를 병렬 생성하고 반환된 StorageType이 요청과 일치하는지 검증합니다.

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

1. **FetchStorageTypeOptions** — `GET /spider/rdbmsmetainfo?DBEngine=mariadb&ConnectionName=...` — 지원하는 StorageType 목록을 동적으로 조회
2. **CreateRDBMS** (StorageType별 병렬) — `POST /spider/rdbms` — 옵션별로 `cb-mariadb-st-<type>` 인스턴스 생성
3. **Poll Available** — `GET /spider/rdbms/{Name}` — Available 상태가 될 때까지 폴링 (기본 30초 간격, 최대 3600초)
4. **VerifyStorageType** — 반환된 StorageType이 요청과 일치하는지 확인 (`Result` 필드에 `PASS`/`FAIL` 기록)
5. **(옵션) AutoDelete** — `AUTO_DELETE=true`이면 검증 직후 인스턴스 자동 삭제. 기본값(`false`)에서는 `delete-all-csp-storage-type-rdbms.sh`로 별도 정리

### StorageType 선택 가능 여부

| CSP | SupportsStorageTypeSelection | 비고 |
|-----|------------------------------|------|
| AWS | ✅ | gp2, gp3, io1, io2 |
| Alibaba | ✅ | cloud_essd, cloud_essd2, cloud_essd3 |
| OpenStack | ✅ | `__DEFAULT__`, `RBD` (MariaDB Trove datastore 구축 필요) |
| NHN | ✅ | General HDD, General SSD |

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

- MariaDB 지원 4개 CSP(AWS/Alibaba/OpenStack/NHN)에 대해 병렬로 StorageType별 검증 실행
- 완료 후 통합 결과 테이블 및 PASS/FAIL/SKIP 집계 출력

**Example output:**
```
================================================================================================================================
                           RDBMS StorageType Test Summary - All CSPs (MariaDB)
================================================================================================================================
CSP          | StorageType(Req)     | StorageType(Ret)   | Result | DB Status      | Elapsed    | Reason
--------------------------------------------------------------------------------------------------------------------------------
AWS          | gp2                  | gp2                | PASS   | Available      | 4m41s      | -
ALIBABA      | cloud_essd           | cloud_essd         | PASS   | Available      | 3m26s      | -
OPENSTACK    | __DEFAULT__          | N/A                | FAIL   | CREATE_ERROR   | 1s         | -
NHN          | General HDD          | General HDD        | PASS   | Available      | 9m22s      | -
--------------------------------------------------------------------------------------------------------------------------------

Total: 4  PASS: 3  FAIL: 1  SKIP: 0
```

### Individual CSP

특정 CSP만 단독 실행:

```bash
./aws-storage-type-test.sh
./alibaba-storage-type-test.sh
./openstack-storage-type-test.sh
./nhn-storage-type-test.sh
```

단독 실행 시 결과/로그 디렉토리는 `RESULT_DIR` / `LOG_DIR` 환경변수로 지정하거나 기본값(`/tmp/st_results_<PID>`, `/tmp/st_logs_<PID>`)이 사용됩니다.

### Delete All Instances

```bash
./delete-all-csp-storage-type-rdbms.sh
```

- StorageTypeOptions를 다시 조회해 `cb-mariadb-st-<type>` 이름의 인스턴스를 유추한 뒤 전체 CSP 병렬 삭제

## Script Structure

```
storage-type-test/
├── run-all-csp-storage-type-tests.sh    # Orchestrator: 전체 CSP 병렬 실행 (AWS/Alibaba/OpenStack/NHN)
├── delete-all-csp-storage-type-rdbms.sh # Orchestrator: 전체 CSP 병렬 삭제
├── common-storage-type-test.sh          # Common: Create → Poll Available → Get Info → Verify → (옵션) Delete
├── aws-storage-type-test.sh
├── alibaba-storage-type-test.sh
├── openstack-storage-type-test.sh
└── nhn-storage-type-test.sh
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
| `DB_Status` | 인스턴스 상태 (`Available`, `CREATE_ERROR`, `TIMEOUT` 등) |
| `Elapsed` | 경과 시간 |
| `Reason` | 특이사항 (예: OpenStack은 StorageType 미제공으로 Available만으로 PASS 처리) |

## API Reference

| Operation | Method | Path |
|-----------|--------|------|
| FetchStorageTypeOptions | `GET` | `/spider/rdbmsmetainfo?DBEngine=mariadb&ConnectionName=` |
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

`StorageTypeOptions`는 metainfo API에서 동적으로 조회되며, 각 타입에 대해 아래 설정으로 인스턴스를 생성합니다.

| 항목 | 값 |
|------|-----|
| DBEngine / Version | `mariadb` / `10.6` |
| DBInstanceSpec | `db.t3.medium` |
| StorageSize | `100 GB` (io1/io2 포함 전 타입 동일) |
| Connection | `aws-config01` |
| Region / Zone | `ap-southeast-2` / `ap-southeast-2a` |
| SubnetNames | `subnet-01`, `subnet-02` (서로 다른 AZ, SubnetGroup 생성 필수) |
| SecurityGroupNames | `sg-01` |

**특이사항**
- io1/io2: `Iops: "3000"` 필드 필수

---

### Alibaba

`StorageTypeOptions`와 `DBInstanceSpecOptions`를 모두 metainfo API에서 동적으로 조회하며, 지역/존에 유효한 스펙을 자동 선택합니다(하드코딩된 스펙은 StorageType과 호환되지 않을 수 있음).

| 항목 | 값 |
|------|-----|
| DBEngine / Version | `mariadb` / `10.6` |
| DBInstanceSpec | metainfo `DBInstanceSpecOptions[0]` (조회 실패 시 `rds.mariadb.s4.large`로 폴백) |
| StorageSize | 기본 `20 GB`, `cloud_essd2`는 `500 GB`, `cloud_essd3`는 `1500 GB` |
| Connection | `alibaba-beijing-config` |
| Region / Zone | `cn-beijing` / `cn-beijing-f` |
| SubnetNames | `subnet-01` |

**특이사항**
- cloud_essd2/cloud_essd3는 최소 StorageSize 요건이 있어 스크립트에서 자동으로 크기를 늘려 요청

---

### OpenStack

| 항목 | 값 |
|------|-----|
| DBEngine / Version | `mariadb` / `10.6` |
| DBInstanceSpec | `m1.small` |
| StorageSize | `20 GB` |
| Connection | `openstack-config01` |
| Region / Zone | `RegionOne` / `nova` |

**특이사항**
- MariaDB Trove datastore가 구축전 상태

---

### NHN

| 항목 | 값 |
|------|-----|
| DBEngine / Version | `mariadb` / `MARIADB_V101118` |
| DBInstanceSpec | `m2.c2m4` |
| StorageSize | `20 GB` |
| Connection | `nhn-korea-pangyo1-config` |
| Region / Zone | `KR1` / `kr-pub-a` |
| SubnetNames | `subnet-01` |


## 시험 결과

### 2026-08-10

OpenStack은 이 환경에 MariaDB Trove datastore가 아직 구축되지 않아 `DBEngineVersion '10.6' cannot be found` 오류로 CREATE_ERROR 발생 (datastore 구축 후 정상 동작 예상).

```
================================================================================================================================
                          RDBMS StorageType Test Summary - All CSPs (MariaDB)
================================================================================================================================
CSP          | StorageType(Req)     | StorageType(Ret)   | Result | DB Status      | Elapsed    | Reason
--------------------------------------------------------------------------------------------------------------------------------
AWS          | gp2                  | gp2                | PASS   | Available      | 4m41s      | -
AWS          | gp3                  | gp3                | PASS   | Available      | 4m41s      | -
AWS          | io1                  | io1                | PASS   | Available      | 4m41s      | -
AWS          | io2                  | io2                | PASS   | Available      | 4m42s      | -
ALIBABA      | cloud_essd           | cloud_essd         | PASS   | Available      | 3m26s      | -
ALIBABA      | cloud_essd2          | cloud_essd2        | PASS   | Available      | 3m6s       | -
ALIBABA      | cloud_essd3          | cloud_essd3        | PASS   | Available      | 3m36s      | -
OPENSTACK    | __DEFAULT__          | N/A                | FAIL   | CREATE_ERROR   | 1s         | -
OPENSTACK    | RBD                  | N/A                | FAIL   | CREATE_ERROR   | 1s         | -
NHN          | General HDD          | General HDD        | PASS   | Available      | 9m22s      | -
NHN          | General SSD          | General SSD        | PASS   | Available      | 10m20s     | -
--------------------------------------------------------------------------------------------------------------------------------
Total: 11  PASS: 9  FAIL: 2  SKIP: 0
```
