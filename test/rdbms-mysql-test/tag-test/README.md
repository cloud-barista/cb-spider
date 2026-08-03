# CB-Spider RDBMS Tag Management Test

RDBMS 리소스의 Tag CRUD API를 검증하는 테스트 스위트입니다. `RDBMSMetaInfo.SupportsTag=true`인 CSP(AWS, Azure, GCP, Alibaba, Tencent, IBM)에 대해서만 실행됩니다.

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

## Supported CSPs

`RDBMSMetaInfo.SupportsTag` 값에 따라 시험 대상 CSP가 결정됩니다.

| CSP | SupportsTag | 시험 대상 |
|-----|-------------|---------|
| AWS | `true` | ✅ |
| Azure | `true` | ✅ |
| GCP | `true` | ✅ |
| Alibaba | `true` | ✅ |
| Tencent | `true` | ✅ |
| IBM | `true` | ✅ |
| OpenStack | `false` | ❌ |
| NCP | `false` | ❌ |
| NHN | `false` | ❌ |

## Test Flow

각 CSP에 대해 다음 순서로 Tag CRUD를 검증합니다:

1. **AddTag(1)** — `POST /spider/tag` — 첫 번째 태그 추가 (`spider-rdbms-tag` / `rdbms-tag-value`)
2. **ListTag** — `GET /spider/tag?...` — 목록에서 첫 번째 태그 존재 확인
3. **GetTag** — `GET /spider/tag/{Key}?...` — 태그 값 일치 확인
4. **AddTag(2)** — `POST /spider/tag` — 두 번째 태그 추가 (`spider-rdbms-tag2` / `rdbms-tag-value2`)
5. **RemoveTag** — `DELETE /spider/tag/{Key}` — 첫 번째 태그 삭제
6. **VerifyRemoved** — `GET /spider/tag?...` — 첫 번째 태그가 목록에서 제거되었는지 확인
7. **Cleanup** — 두 번째 태그 삭제 (정리)

모든 단계가 PASS여야 해당 CSP가 전체 PASS로 판정됩니다.

## Configuration

```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:*****           # Basic auth (admin:<password>)
```

## How to Run Tests

### All Supported CSPs in Parallel

```bash
./run-all-csp-rdbms-tag-tests.sh
```

- SupportsTag=true인 6개 CSP에 대해 병렬로 Tag CRUD 검증 실행
- 완료 후 통합 결과 테이블 및 PASS/FAIL 집계 출력

**Example output:**
```
===================================================================================================================
                   RDBMS TAG MANAGEMENT TEST SUMMARY (SupportsTag=true CSPs)
===================================================================================================================
CSP          | AddTag   | ListTag  | GetTag  | AddTag2  | RemoveTag  | VerifyRemoved  | Elapsed
-------------------------------------------------------------------------------------------------------------------
AWS          | PASS     | PASS     | PASS    | PASS     | PASS       | PASS           | 3s
AZURE        | PASS     | PASS     | PASS    | PASS     | PASS       | PASS           | 2s
GCP          | PASS     | PASS     | PASS    | PASS     | PASS       | PASS           | 1s
ALIBABA      | PASS     | PASS     | PASS    | PASS     | PASS       | PASS           | 4s
TENCENT      | PASS     | PASS     | PASS    | PASS     | PASS       | PASS           | 2s
IBM          | PASS     | PASS     | PASS    | PASS     | PASS       | PASS           | 3s
-------------------------------------------------------------------------------------------------------------------

Total: 6 PASS, 0 FAIL
```

### Individual CSP

특정 CSP만 단독 실행:

```bash
./aws-rdbms-tag-test.sh
./azure-rdbms-tag-test.sh
./gcp-rdbms-tag-test.sh
./alibaba-rdbms-tag-test.sh
./tencent-rdbms-tag-test.sh
./ibm-rdbms-tag-test.sh
```

단독 실행 시 결과 파일은 `RESULT_DIR` 환경변수로 지정하거나 기본값(`/tmp/rdbms_tag_results`)이 사용됩니다.

## Script Structure

```
tag-test/
├── run-all-csp-rdbms-tag-tests.sh   # Orchestrator: 6개 CSP 병렬 실행, PASS/FAIL 집계
├── common-rdbms-tag-test.sh         # Common: AddTag → ListTag → GetTag → AddTag2 → RemoveTag → VerifyRemoved
├── aws-rdbms-tag-test.sh
├── azure-rdbms-tag-test.sh
├── gcp-rdbms-tag-test.sh
├── alibaba-rdbms-tag-test.sh
├── tencent-rdbms-tag-test.sh
└── ibm-rdbms-tag-test.sh
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:*****` | Basic auth credentials |
| `RESULT_DIR` | `/tmp/rdbms_tag_results` | Result file output directory |
| `VERBOSE` | `0` | `1`로 설정 시 per-CSP 전체 로그 덤프 출력 |

```bash
# Example: verbose output
VERBOSE=1 ./run-all-csp-rdbms-tag-tests.sh
```

## Result Format

결과 파일(`result_<csp>.txt`)은 파이프(|) 구분 8개 필드:

```
CSP|AddTag|ListTag|GetTag|AddTag2|RemoveTag|VerifyRemoved|Elapsed
```

| 필드 | 설명 |
|------|------|
| `CSP` | CSP 이름 (예: AWS) |
| `AddTag` | 첫 번째 태그 추가 결과 (`PASS` / `FAIL`) |
| `ListTag` | 태그 목록 조회 및 존재 확인 결과 |
| `GetTag` | 특정 태그 조회 및 값 일치 확인 결과 |
| `AddTag2` | 두 번째 태그 추가 결과 |
| `RemoveTag` | 첫 번째 태그 삭제 결과 |
| `VerifyRemoved` | 삭제 후 목록에서 제거 확인 결과 |
| `Elapsed` | 경과 시간 |

## Logs & Results

```
/tmp/rdbms_tag_results_<PID>/result_<csp>.txt
/tmp/rdbms_tag_logs_<PID>/log_<csp>.txt
```

실행 중 모니터링:

```bash
tail -f /tmp/rdbms_tag_logs_<PID>/log_aws.txt
```

## Tag API Reference

| Operation | Method | Path |
|-----------|--------|------|
| AddTag | `POST` | `/spider/tag` |
| ListTag | `GET` | `/spider/tag?ConnectionName=&ResourceType=rdbms&ResourceName=` |
| GetTag | `GET` | `/spider/tag/{Key}?ConnectionName=&ResourceType=rdbms&ResourceName=` |
| RemoveTag | `DELETE` | `/spider/tag/{Key}` |

## 시험 결과

### 2026-08-03

```
===================================================================================================================
                 RDBMS TAG MANAGEMENT TEST SUMMARY (SupportsTag=true CSPs)
===================================================================================================================
CSP          | AddTag   | ListTag  | GetTag  | AddTag2  | RemoveTag  | VerifyRemoved | Elapsed
-------------------------------------------------------------------------------------------------------------------
AWS          | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 6s
AZURE        | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 4m14s
GCP          | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 1m45s
ALIBABA      | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 7s
TENCENT      | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 5s
IBM          | PASS     | PASS     | PASS    | PASS     | PASS       | PASS          | 22s
-------------------------------------------------------------------------------------------------------------------
Total: 6 PASS, 0 FAIL
```
