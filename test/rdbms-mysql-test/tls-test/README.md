# CB-Spider RDBMS TLS Connection Test

- 각 CSP의 실제 RDBMS 엔드포인트 대상, 실제 MySQL 클라이언트 연결을 4가지 TLS 클라이언트 설정으로 검증하는 테스트
- DB 인스턴스의 실제 `require_secure_transport` 설정값 **및** TLS 지원 여부를 교차 검증
- 9개 CSP에 대해 병렬 실행

## 동작 방식

- 실제 TLS 테스트는 **[`tlsprobe`](tlsprobe/main.go)**(Go 프로그램)가 수행
- CB-Spider 내부 사용과 동일한 드라이버인 [`go-sql-driver/mysql`](https://github.com/go-sql-driver/mysql)로 RDBMS 인스턴스에 직접 접속 (`api-runtime/common-runtime/RDBMSManager.go` 참고)
- CSP별 `.sh` 스크립트(`aws-tls-test.sh` 등)는 connection/instance 이름만 설정하고 `tlsprobe`를 호출하는 얇은 래퍼
- 대상 DB 인스턴스의 TLS 관련 정보(`require_secure_transport` ON/OFF, TLS 지원 여부, 라이브 캡처한 서버 CA 인증서)는 CB-Spider의 `GET /spider/rdbms/{Name}/secure-transport` API를 통해 얻음 — 자세한 내용은 아래 API 참고 섹션 참고

### 왜 `mysql` CLI를 쓰지 않는가?

- 이전 버전: `mysql` CLI(`--ssl-mode`/`--ssl-ca`)를 셸 아웃해서 사용
- 문제점: 같은 인증서인데도 `mysql` 클라이언트 빌드(`libmysqlclient`)에 따라 검증 결과가 달라지는 신뢰성 문제(OpenSSL로 직접 검증하면 정상).
- 해결: `tlsprobe`가 CB-Spider 사용 중인 `go-sql-driver/mysql`을 그대로 재사용
- 부가 발견: 연속 시도 간 지연이 없으면 일부 네트워크에서 타임아웃 발생 → `-inter-attempt-delay`(기본 3초)로 해결

### 4가지 시나리오

- 이름은 실제로 재현하는 클라이언트 측 설정을 따서 지음 — `go-sql-driver/mysql` DSN의 `tls=` 값, 또는 `mysql` CLI의 `--ssl-mode`에 해당
- 보안이 약한 것 → 강한 것 순서로 번호를 매김:

| # | 클라이언트 설정 | `require_secure_transport=ON` | `require_secure_transport=OFF` |
|---|---|---|---|
| `S1` | `tls=false` (암호화 없음) | ❌ FAIL (`ERROR 3159`) | ✅ SUCCESS (평문, 암호화 없음) |
| `S2` | `tls=skip-verify` (암호화되지만 검증 전혀 없음) | ✅ SUCCESS (MITM에는 안전하지 않음) | ✅ SUCCESS |
| `S3` | `VERIFY_CA` 상당 (체인만 검증, 호스트명 생략) | ✅ SUCCESS (체인 신뢰는 확인 — `skip-verify`보다 안전) | ✅ SUCCESS |
| `S4` | `tls=true`, with CA (`VERIFY_IDENTITY` 상당 — 체인 + 호스트명 검증) | ✅ SUCCESS (암호화 + 신원 검증 — 가장 안전한 옵션) | ✅ SUCCESS |

- `tlsprobe` 코드(`crypto/tls.Config`) 매핑:
  - `S1` → `TLS: nil` (TLS 시도 자체를 안 함)
  - `S2` → `TLS: &tls.Config{InsecureSkipVerify: true}`
  - `S3` → `TLS: &tls.Config{InsecureSkipVerify: true, VerifyPeerCertificate: <체인만 x509.Verify(), DNSName은 빈 채로 둠>}` — Go `crypto/tls`에는 "체인만 검증" 내장 모드가 없어 표준 패턴으로 직접 구현 (`tlsprobe/main.go`의 `verifyCAOnlyConfig` 참고). MySQL 자체의 `--ssl-mode=VERIFY_CA`와 동일한 보안 수준
  - `S4` → `TLS: &tls.Config{RootCAs: capturedCAPool, ServerName: verifyServerName}` (`verifyServerName`은 보통 접속 호스트이지만, 순수 IP + 인증서가 IP SAN 대신 DNS SAN을 가진 경우 — 예: GCP Cloud SQL — 인증서의 DNS SAN 사용; Known Caveats 참고)
- `S1`(`tls=false`)은 항상 실행:
  - TLS를 아예 지원하지 않는 인스턴스(예: TLS 꺼진 Alibaba/Tencent/NCP)에서도 의미 있는 유일한 시나리오
  - 최소한 Plain Text 연결 가능 여부 확인
  - 이 경우 `S2`/`S3`/`S4`는 `N/A` (테스트할 대상 없음 — 아래 `TLSAvailable` 필드 참고)
- `S3`을 별도로 두는 이유: `S4`(`VERIFY_IDENTITY`)는 체인 신뢰 + 호스트명 일치를 모두 요구하지만, `S3`(`VERIFY_CA`)는 체인 신뢰만 요구 — 인증서에 SAN/CN이 없거나 접속 호스트명과 안 맞는 경우에도(`S4`가 실패하는 바로 그 경우, 예: OpenStack/NHN) `S3`는 여전히 성공함. 신뢰할 수 없는 CA로 서명된 인증서는 `S3`도 여전히 거부하므로 `S2`(`skip-verify`, 검증을 통째로 생략)보다 실질적으로 더 안전한 대안
- CB-Spider는 사전 정보 확인용으로만 사용 (CB-Spider 자체 REST API):
  1. 인스턴스의 실제 `Endpoint`와 `MasterUserName` (`GET /spider/rdbms/{Name}`)
  2. `require_secure_transport` ON/OFF, TLS 지원 여부(`TLSInUse` → `TLSAvailable` 컬럼), 라이브 캡처한 서버 CA 인증서 (`GET /spider/rdbms/{Name}/secure-transport`)
- CA 인증서는 그대로 `x509.CertPool`로 파싱되어 `S3`/`S4`에서 사용 — 클라이언트 툴의 특이한 동작과 무관하게, CB-Spider가 반환하는 인증서가 실제로 사용 가능한지를 검증하는 엔드투엔드 검증이기도 함

## 사전 준비

### CB-Spider 실행

```bash
cd ./bin; ./start.sh
```

### RDBMS 인스턴스가 이미 존재해야 함

- 이 테스트는 RDBMS 인스턴스가 이미 존재한다고 가정
- 먼저 상위 디렉터리에서 네트워크 사전 준비와 생성 테스트 실행:

```bash
cd ..
./run-all-csp-network-prepare.sh   # VPC/Subnet/SG 사전 준비 (최초 1회)
./run-all-csp-rdbms-tests.sh
```

### 필요한 도구

- **Go 툴체인** (`go`) — 유일한 실질적 의존성. CB-Spider 자체를 빌드할 수 있다면 이미 보유. `tlsprobe`는 `go run`으로 실행되므로 별도 빌드 불필요
- `bash` — 병렬 멀티-CSP 오케스트레이터(`run-all-csp-tls-tests.sh`)와 CSP별 `.sh` 래퍼에만 필요. `tlsprobe`를 직접 호출하면 불필요
- 필요하지 **않은** 것: `mysql`/`mysql-client`, `curl`, `jq`

## 설정

```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:****             # Basic auth (admin:<password>)
```

선택 사항:

```bash
export CONNECT_TIMEOUT=10        # 시도당 연결 타임아웃(초)
export INTER_ATTEMPT_DELAY=3     # 시나리오 시도 사이의 대기 시간(초)
export LAUNCH_STAGGER_SEC=8      # 각 CSP 실행 시작 시점 사이의 대기 시간(run-all-csp-tls-tests.sh 전용)
```

## 테스트 실행 방법

### 전체 CSP 병렬 실행

```bash
./run-all-csp-tls-tests.sh
```

- 9개 CSP 전체에 대해 4-시나리오 TLS 매트릭스를 병렬 실행
- 완료 시 통합 결과 테이블과 PASS/FAIL 집계 출력

**예시 출력** (실제 실행 결과 — OpenStack/NHN의 `S4`가 왜 실패하는 것이 정상인지, `S3`는 왜 같은 인스턴스에서 성공하는지는 Known Caveats 참고):
```
======================================================================================================================
                            RDBMS TLS CONNECTION TEST SUMMARY - ALL CSPs
======================================================================================================================

CSP          | ReqSecTLS  | TLSAvailable | S1       | S2       | S3       | S4       | Elapsed
------------------------------------------------------------------------------------------------------------------
AWS          | OFF        | ON           | PASS     | PASS     | PASS     | PASS     | 15s
AZURE        | ON         | ON           | PASS     | PASS     | PASS     | PASS     | 9s
GCP          | OFF        | ON           | PASS     | PASS     | PASS     | PASS     | 15s
ALIBABA      | OFF        | OFF          | PASS     | N/A      | N/A      | N/A      | 12s
TENCENT      | OFF        | OFF          | PASS     | N/A      | N/A      | N/A      | 11s
IBM          | ON         | ON           | PASS     | PASS     | PASS     | PASS     | 22s
OPENSTACK    | OFF        | ON           | PASS     | PASS     | PASS     | FAIL*    | 14s
NCP          | OFF        | OFF          | PASS     | N/A      | N/A      | N/A      | 8s
NHN          | OFF        | ON           | PASS     | PASS     | PASS     | FAIL*    | 32s
------------------------------------------------------------------------------------------------------------------

Total: 7 PASS, 2 FAIL
* Expected: the certificate has no Subject Alternative Name at all (MySQL's own auto-generated cert) — see Known Caveats. GCP's own IP-endpoint case is auto-resolved via its certificate's DNS SAN.
(S2-S4 = N/A counts as PASS when TLSAvailable=OFF -- there's nothing to test, not a failure)
```

- `ReqSecTLS`: `RequireSecureTransport`(해당 인스턴스의 실제 `require_secure_transport` 설정값, `ON`/`OFF`)의 축약 컬럼명
- `ALIBABA`/`TENCENT`/`NCP`(TLS 꺼짐)도 실제 `PASS`: `S1`(평문)은 실제로 테스트되어 성공, `S2`/`S3`/`S4`는 미테스트/공백이 아니라 올바르게 `N/A`
- `OPENSTACK`/`NHN`은 `S4`(호스트명 검증까지 요구)만 FAIL이고 `S3`(체인만 검증)는 PASS — 같은 인스턴스, 같은 인증서인데도 검증 방식에 따라 결과가 갈린다는 걸 그대로 보여주는 사례

### 개별 CSP

```bash
./aws-tls-test.sh
./azure-tls-test.sh
./gcp-tls-test.sh
./alibaba-tls-test.sh
./tencent-tls-test.sh
./ibm-tls-test.sh
./openstack-tls-test.sh
./ncp-tls-test.sh
./nhn-tls-test.sh
```

- 개별 실행 시 결과 파일은 지정한 `RESULT_DIR`, 또는 기본값(`/tmp/rdbms_tls_results`)에 생성
- `tlsprobe -h`: 전체 플래그 목록 (`-spider-url`, `-spider-auth`, `-connection`, `-rdbms`, `-password`, `-csp-name`, `-timeout`, `-inter-attempt-delay`, `-result-file`)

## 스크립트 구조

```
tls-test/
├── run-all-csp-tls-tests.sh   # 오케스트레이터: 전체 병렬 실행, PASS/FAIL 집계
├── common-tls-test.sh         # 얇은 래퍼: 올바른 플래그로 `go run tlsprobe`를 호출
├── tlsprobe/
│   └── main.go                # 실제 테스트 로직 — 크로스플랫폼, bash 불필요
├── aws-tls-test.sh
├── azure-tls-test.sh
├── gcp-tls-test.sh
├── alibaba-tls-test.sh
├── tencent-tls-test.sh
├── ibm-tls-test.sh
├── openstack-tls-test.sh
├── ncp-tls-test.sh
└── nhn-tls-test.sh
```

## 환경 변수

| 변수 | 기본값 | 설명 |
|----------|---------|--------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:****` | Basic auth 자격 증명 |
| `CONNECT_TIMEOUT` | `10` | 시도당 연결 타임아웃(초) |
| `INTER_ATTEMPT_DELAY` | `3` | 한 CSP 내 시나리오 시도 사이의 대기 시간(초) |
| `LAUNCH_STAGGER_SEC` | `8` | `run-all-csp-tls-tests.sh`에서 각 CSP *실행을 시작하는* 시점 사이의 대기 시간(초) — Known Caveats 참고 |
| `RESULT_DIR` | `/tmp/rdbms_tls_results` | 결과 파일 출력 디렉터리 (단일 CSP 실행 시) |
| `VERBOSE` | `0` | `1`로 설정 시 CSP별 전체 로그 출력 |

```bash
# 예시: verbose 출력
VERBOSE=1 ./run-all-csp-tls-tests.sh
```

## 결과 형식

- 각 결과 파일(`result_<csp>.txt`)은 파이프(`|`)로 구분된 8개 필드:

```
CSP|RequireSecureTransport|TLSAvailable|S1|S2|S3|S4|Elapsed
```

| 필드 | 설명 |
|------|------|
| `CSP` | CSP 이름 (예: AWS) |
| `RequireSecureTransport` | CB-Spider가 보고한 `ON` / `OFF`, 또는 전체 확인 과정이 중단된 경우(예: CB-Spider나 인스턴스에 접근 못함) `N/A` |
| `TLSAvailable` | TLS를 아예 지원하면 `ON`(CB-Spider의 `TLSInUse`), 아니면 `OFF`, 확인 중단 시 `N/A` |
| `S1`–`S4` | 시나리오별 판정 — 아래 참고 |
| `Elapsed` | 소요 시간 |

시나리오별 값:

| 값 | 의미 |
|------|------|
| `PASS` | 실제 결과가 현재 `require_secure_transport` 값에 대한 기대 결과와 일치 |
| `FAIL` | 실제 결과가 기대 결과와 불일치 |
| `SKIP` | 테스트 불가 (`TLSAvailable=ON`이지만 CA 인증서 캡처 실패 — 로그의 `CACertificateError` 참고; CA 필요한 것은 `S3`, `S4`) |
| `N/A` | `TLSAvailable=OFF`라 해당 사항 없음 (이 경우 `S1`만 실행) |

## API 참고

| 동작 | 메서드 | 경로 |
|-----------|--------|------|
| GetRDBMS (엔드포인트, 마스터 사용자) | `GET` | `/spider/rdbms/{Name}?ConnectionName=<name>` |
| GetRDBMSSecureTransport (ON/OFF + TLS 지원 여부 + CA 인증서) | `GET` | `/spider/rdbms/{Name}/secure-transport?ConnectionName=<name>&MasterUserPassword=<password>` |

- `GetRDBMSSecureTransport`: 표준 SQL(`SHOW VARIABLES LIKE 'require_secure_transport'`)로 엔진에 직접 접속 + 별도 라이브 TLS 핸드셰이크로 CA 인증서 확보 — `api-runtime/common-runtime/RDBMSManager.go`, `api-runtime/common-runtime/RDBMSTLSProbe.go` 참고

## 로그 & 결과

```
/tmp/rdbms_tls_results_<PID>/result_<csp>.txt
/tmp/rdbms_tls_logs_<PID>/log_<csp>.txt
```

진행 중인 실행 모니터링:

```bash
tail -f /tmp/rdbms_tls_logs_<PID>/log_aws.txt
```

## Known Caveats (알려진 주의사항)

- **`TLSAvailable=OFF`(예: Alibaba/Tencent/NCP 테스트 인스턴스에서 관찰)** — 단순히 해당 인스턴스의 TLS 설정이 꺼져 있다는 뜻, 테스트 결함 아님
  - Alibaba/Tencent CSP 드라이버 SDK는 켤 수 있는 API 제공: Alibaba `rds.ModifyDBInstanceSSL`(`SSLEnabled`/`ForceEncryption` 쌍으로 한 번에 활성화+강제 가능), Tencent `cdb.OpenSSL`(활성화만 확인됨; `require_secure_transport=ON` 상당의 강제 적용은 별도 파라미터 수정 호출 필요할 것으로 보이나 SDK에서 미확인)
  - 이 인스턴스들의 TLS 활성화는 이 테스트 스위트와 별개의 후속 작업
- **NCP는 VPC 외부 접근에 public IP(도메인) 엔드포인트 필요**
  - 기본 사설 엔드포인트(`*.vpc-cdb.ntruss.com`)는 VPC 바깥 테스트 머신에서 `dial tcp ...: no route to host`로 실패 — TLS 문제 아니고 네트워크 도달성 문제
  - 해결: DB 생성 후 NCP 콘솔에서 public IP 설정 (엔드포인트가 `*.vpc-pub-cdb.ntruss.com`으로 해석됨)
  - 설정 직후가 아니라 **약 5분 경과 후**부터 외부 접근 가능
  - 이 인스턴스는 TLS 자체가 꺼져 있어(`TLSAvailable=OFF`) `S1=PASS`, `S2`/`S3`/`S4`는 `N/A`로 정상 종료 (위 예시 출력의 `NCP` 행 참고)
- **`GET /rdbms/{Name}/secure-transport`가 캡처하는 CA 인증서는 서버가 제시하는 체인의 최상단 인증서**
  - 일부 CSP는 궁극적인 자체 서명 루트가 아니라 중간(intermediate) CA일 수 있음 (API 응답의 `CACertificate.IsSelfSigned`로 확인 가능)
  - 어느 쪽이든 `S3`/`S4`의 *체인* 검증에는 유효 (서버가 항상 같은 체인을 제시)
- **`S4`는 체인은 유효하지만 접속 대상에 맞는 SAN(Subject Alternative Name)이 없는 인증서에서는 실패 가능** — 체인 신뢰와 별개인 호스트명/신원 검증, CB-Spider 결함 아니고 해당 CSP 인증서의 실제 속성. Go `crypto/tls`는 이를 정확히 구분해서 보고(평범한 `mysql` CLI의 래핑된 에러는 체인 신뢰 실패와 구분 못 함):
  - **GCP Cloud SQL, `Endpoint`가 순수 IP인 경우**: 인증서에 `IP Address` 타입 SAN 없음(그대로면 `x509: ... doesn't contain any IP SANs`) — 하지만 DNS SAN은 보유(`N-<uuid>.<region>.sql.goog` 형태의 인스턴스별 고정 이름). `tlsprobe`가 자동 감지: IP `Endpoint`로 접속은 그대로 하되 TLS `ServerName`(신원 검사 대상)을 DNS SAN으로 설정해 성공 — `tlsprobe/main.go`의 `verifyServerName` 참고. 동작 시 로그: `Endpoint is a bare IP but the certificate has a DNS SAN (...) -- verifying identity against that instead`
  - **OpenStack Trove와 NHN(둘 다 MySQL 자체 `--ssl` 자동 생성 인증서 사용)**: 인증서에 SAN 확장 자체가 아예 없음 — `x509: certificate is not valid for any names, but wanted to match <name>`. Go 1.15부터 SAN 없을 때 레거시 `CN` 필드로 폴백하지 않도록 의도적으로 막음(보안 강화 — Go 1.15 릴리스 노트 참고), 이름이 `CN`과 일치해도 실패. GCP와 달리 폴백할 SAN 자체가 없어 `S4`(호스트명까지 검증)로는 클라이언트 측에서 고칠 수 없음 — 근본 해결은 해당 Trove/MySQL 인스턴스 운영 측에서 제대로 된 SAN을 가진 인증서로 재발급하는 것뿐
    - 다만 `S3`(`VERIFY_CA` 상당, 체인만 검증)는 이 두 CSP에서도 정상 `PASS` — 호스트명 검증을 아예 안 하므로 SAN 부재의 영향을 받지 않음. 즉 "완전한 신원 검증(`S4`)은 못 고치지만, 체인 신뢰만 확인하는 `VERIFY_CA`/`S3`는 그대로 유효"가 더 정확한 설명
    - OpenStack Trove는 인스턴스 생성 후 `openstack database ssl enable --mode <mode> --password_ref <Barbican secret>`로 SAN 포함 커스텀 인증서(PKCS#12, Barbican 저장)를 등록하는 기능을 프로젝트 레벨에서 지원하지만, 실제 tenant에게 열려 있는지는 CSP별 배포 설정에 달려 있어 CB-Spider 테스트 대상 OpenStack 클라우드에서는 확인되지 않음. NHN Cloud(RDS for MySQL)는 사용자가 인증서를 직접 제공하거나 SAN을 지정하는 옵션 자체가 없음(공식 사용자 가이드 기준)
  - GCP 참고: Cloud SQL Admin API의 `instances.get().serverCaCert.cert` 필드를 CSP 네이티브 CA 소스로 활용 가능 — 위 SAN 기반 `ServerName` 수정과 별개로, 캡처 로직 개선 후보(가능하면 라이브 캡처보다 CSP 네이티브 소스 우선)
- **`S3`(`VERIFY_CA` 상당)는 `go-sql-driver/mysql`에 네이티브 DSN 옵션이 없어 직접 구현함** — `tls-verify=ca` 파라미터를 추가하자는 PR([go-sql-driver/mysql#1742](https://github.com/go-sql-driver/mysql/pull/1742))이 있었지만 머지되지 않고 closed됨. 대신 Go 표준 패턴(`InsecureSkipVerify: true` + `VerifyPeerCertificate` 콜백에서 `DNSName`을 비워둔 채 `x509.Certificate.Verify()`만 수행)으로 구현 — `tlsprobe/main.go`의 `verifyCAOnlyConfig` 참고. MySQL 자체의 `--ssl-mode=VERIFY_CA`가 공식적으로 권장하는, self-signed/SAN 없는 인증서에 대한 표준적인 접속 방식과 동일한 보안 수준
- `S3`/`S4`가 (`TLSAvailable=ON` 상태에서) `SKIP`이면 해당 CSP 로그의 `CACertificateError` 확인 — CB-Spider 자체 라이브 캡처 프로브가 실패했다는 의미 (프로브의 재시도/오류 노출 로직은 `RDBMSTLSProbe.go` 참고)
- **일부 네트워크는 연속 연결 시도를 스로틀링(throttle)**
  - 단일 CSP 내 연속 시도: `INTER_ATTEMPT_DELAY`(기본 3초)로 회피
  - 9개 CSP 전체에 걸친 대규모 스로틀링: `LAUNCH_STAGGER_SEC`(기본 8초 간격)로 회피
  - 증상: `context deadline exceeded` 또는 `dial tcp ...: i/o timeout` — 구체적 `x509:` 메시지로 빠르게 실패하는 위의 SAN 관련 실패와 구분됨
  - 기본값(3초/8초)에서도 여전히 발생 시 두 값을 더 올릴 것:
    ```bash
    INTER_ATTEMPT_DELAY=5 LAUNCH_STAGGER_SEC=12 ./run-all-csp-tls-tests.sh
    ```
  - CB-Spider나 CSP 문제가 아니라 로컬 머신의 네트워크 경로(VPN, 라우터, 사내 방화벽 등)에 특유한 것이며, 적절한 간격은 머신마다 다름 — 동시에 여러 CSP를 띄우는 병렬 실행 구조 자체가 원인일 수도 있으니, 다른 네트워크(예: 모바일 핫스팟)에서 재현되지 않는다면 이 네트워크 경로 특유의 문제로 봐도 됨

## Test Results

### 2026-09-28

```
======================================================================================================================
                            RDBMS TLS CONNECTION TEST SUMMARY - ALL CSPs
======================================================================================================================

CSP          | ReqSecTLS  | TLSAvailable | S1       | S2       | S3       | S4       | Elapsed
----------------------------------------------------------------------------------------------------------------------
AWS          | OFF        | ON          | PASS     | PASS     | PASS     | PASS     | 16s
AZURE        | ON         | ON          | PASS     | PASS     | PASS     | PASS     | 11s
GCP          | OFF        | ON          | PASS     | PASS     | PASS     | PASS     | 17s
ALIBABA      | OFF        | OFF         | PASS     | N/A      | N/A      | N/A      | 14s
TENCENT      | OFF        | OFF         | PASS     | N/A      | N/A      | N/A      | 12s
IBM          | ON         | ON          | PASS     | PASS     | PASS     | PASS     | 28s
OPENSTACK    | OFF        | ON          | PASS     | PASS     | PASS     | FAIL     | 20s
NCP          | OFF        | OFF         | PASS     | N/A      | N/A      | N/A      | 11s
NHN          | OFF        | ON          | PASS     | PASS     | PASS     | FAIL     | 15s
----------------------------------------------------------------------------------------------------------------------

Total: 7 PASS, 2 FAIL
(S2-S4 = N/A counts as PASS when TLSAvailable=OFF -- there's nothing to test, not a failure)
```

`OPENSTACK`/`NHN`의 `S4` FAIL은 SAN 없는 인증서로 인한 예상된 결과(Known Caveats 참고) — 그 외 모든 CSP/시나리오 정상.
