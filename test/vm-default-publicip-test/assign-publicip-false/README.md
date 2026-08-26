# CB-Spider VM AssignPublicIP=false Test

`ReqInfo.AssignPublicIP: false` 옵션으로 VM을 생성했을 때, 실제로 Public IP가
할당되지 않고 VM이 정상적으로 `Running` 상태에 도달하며 Private IP는 정상
할당되는지, 그리고 이렇게 만든 VM에 VM Manager의 default-PublicIP API
(`AssignVMDefaultPublicIP`/`UnassignVMDefaultPublicIP`)로 나중에 Public IP를
붙였다 뗄 수 있는지를 10개 CSP 전체에서 검증하는 테스트 스위트입니다.
`test/rdbms-mysql-test/storage-type-test/`와 상위 디렉터리
`test/vm-default-publicip-test/`를 참고하여 작성했습니다.

## Purpose

VM 생성 REST API에 추가된 `AssignPublicIP`(옵션, `*bool`) 필드가 `false`로
설정되었을 때, 그리고 그렇게 만든 VM에 나중에 VM Manager의 default-PublicIP
API로 Public IP를 붙였다 뗄 수 있는지를 검증합니다:

1. VM이 `Running` 상태에 정상적으로 도달하는가 (타임아웃/Failed면 실패)
2. VM에 `PrivateIP`가 정상적으로 할당되는가
3. VM에 `PublicIP`가 **할당되지 않는가** (값이 존재하면 실패)
4. `AssignVMDefaultPublicIP` 요청 (`POST /vm/{Name}/publicip`)
5. Public IP가 할당되었는가 확인
6. 할당된 Public IP로 SSH 접속 확인
7. `UnassignVMDefaultPublicIP` 요청 (`DELETE /vm/{Name}/publicip`)
8. VM에 `PublicIP`가 다시 **할당되지 않는가** (값이 존재하면 실패)

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### Pre-created Network Resources

상위 디렉터리의 네트워크 준비 스크립트로 VPC/Subnet/SecurityGroup/KeyPair가
미리 생성되어 있어야 합니다 (Connection 이름, 이미지/스펙, VPC/Subnet CIDR은
`../README.md`를 참고).

```bash
cd ..
./run-all-csp-network-prepare.sh
cd assign-publicip-false
```

### Required Tools

- `bash` 3.2+
- `curl`
- `jq`
- `ssh` client (OpenSSH) — for the SSH-login check after Assign

## Test Flow

각 CSP에 대해 다음 순서로 검증합니다:

1. **CreateVM** — `POST /spider/vm` (`ReqInfo.AssignPublicIP: false` 포함) —
   `cb-spider-nopublicip-test` 인스턴스 생성
2. **Poll Running** — `GET /spider/vmstatus/{Name}` — `Running` 상태가 될
   때까지 폴링 (기본 15초 간격, 최대 1800초). `Failed` 상태이거나 타임아웃이면
   즉시 FAIL
3. **GetVM** — `GET /spider/vm/{Name}` — `PrivateIP`/`PublicIP` 조회
4. **Verify (no PublicIP)** — `PublicIP`가 비어있지 않으면 FAIL, `PrivateIP`가
   비어있으면 FAIL
5. **AssignVMDefaultPublicIP** — `POST /vm/{Name}/publicip` — Public IP 자동
   생성+할당 요청
6. **Verify (assigned)** — `GetVM`으로 `PublicIP`가 채워졌는지 확인, 비어있으면
   FAIL
7. **SSH check** — 할당된 Public IP + 상위 디렉터리에서 준비한 KeyPair의
   PrivateKey로 SSH 접속 시도 (기본 10회 재시도, 30초 간격). 접속 실패(`FAIL`)는
   물론, PrivateKey 파일이 없어서 시도 자체를 못한 경우(`NO_KEY`)와 Public IP가
   비어 있어 시도할 수 없는 경우(`NO_IP`)도 접속 가능 여부를 확인할 수 없으므로
   모두 전체 FAIL 처리
8. **UnassignVMDefaultPublicIP** — `DELETE /vm/{Name}/publicip` — 방금 할당한
   Public IP 해제(+ 삭제) 요청
9. **Verify (unassigned)** — `GetVM`으로 `PublicIP`가 다시 비었는지 확인,
   남아있으면 FAIL
10. 인스턴스는 자동 삭제되지 않습니다 — `delete-all-csp-assign-publicip-false-vm.sh`로
    별도 정리

## How to Run Tests

### All CSPs in Parallel

```bash
./run-all-csp-assign-publicip-false-tests.sh
```

**Example output:**
```
====================================================================================================================================================================
                    VM AssignPublicIP=false TEST SUMMARY - ALL CSPs
====================================================================================================================================================================

CSP         | Result | VMStatus   | PrivateIP       | PubIP(init)   | PubIP(assign)   | SSH    | PubIP(unassign) | Elapsed  | Reason
--------------------------------------------------------------------------------------------------------------------------------------------------------------------
AWS         | PASS   | Running    | 192.168.1.10    | (none)        | 3.34.12.55      | OK     | (none)          | 4m30s    | -
AZURE       | PASS   | Running    | 192.168.0.5     | (none)        | 20.196.55.10    | OK     | (none)          | 6m12s    | -
...
--------------------------------------------------------------------------------------------------------------------------------------------------------------------

Total: 10  PASS: 10  FAIL: 0
```

### Individual CSP

```bash
./aws-assign-publicip-false-test.sh
./azure-assign-publicip-false-test.sh
./gcp-assign-publicip-false-test.sh
./alibaba-assign-publicip-false-test.sh
./tencent-assign-publicip-false-test.sh
./ibm-assign-publicip-false-test.sh
./openstack-assign-publicip-false-test.sh
./ncp-assign-publicip-false-test.sh
./nhn-assign-publicip-false-test.sh
./kt-assign-publicip-false-test.sh
```

단독 실행 시 결과 디렉터리는 `RESULT_DIR` 환경변수로 지정하거나 기본값
(`/tmp/vm_nopublicip_results`)이 사용됩니다.

### Delete All Test VMs

```bash
./delete-all-csp-assign-publicip-false-vm.sh
```

## Configuration

```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:*****           # Basic auth (admin:<password>)
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|--------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:****` | Basic auth credentials |
| `MAX_WAIT_SEC` | `1800` (create) / `900` (delete) | Timeout per instance (seconds) |
| `POLL_INTERVAL` | `15` | Polling interval (seconds) |
| `KEY_DIR` | `/tmp/vm_publicip_keys` | Directory holding the PrivateKey saved by `../common-network-prepare.sh` (file: `${KEY_DIR}/${CONNECTION_NAME}.pem`) |
| `SSH_USER` | `cb-user` | OS login user for the SSH-login check |
| `SSH_MAX_ATTEMPTS` | `10` | SSH connection retry count |
| `SSH_RETRY_INTERVAL` | `30` | Seconds between SSH retries |
| `VERBOSE` | `0` | `1`로 설정 시 CSP별 전체 로그 덤프 출력 |

## Result Format

결과 파일(`result_<csp>.txt`)은 파이프(|) 구분 10개 필드:

```
CSP|Result|VMStatus|PrivateIP|PublicIP(NoAssign)|PublicIP(Assigned)|SSH(Assigned)|PublicIP(Unassigned)|Elapsed|Reason
```

| 필드 | 설명 |
|------|------|
| `CSP` | CSP 이름 (예: AWS) |
| `Result` | `PASS` / `FAIL` |
| `VMStatus` | 최종 확인된 상태 (예: `Running`), 실패 시 실패 당시 상태 |
| `PrivateIP` | 생성된 VM의 PrivateIP (`-`는 생성 실패로 조회 불가) |
| `PublicIP(NoAssign)` | 생성 직후(AssignPublicIP=false)의 PublicIP — `(none)`이어야 함 |
| `PublicIP(Assigned)` | `AssignVMDefaultPublicIP` 이후의 PublicIP — 비어있지 않아야 함 |
| `SSH(Assigned)` | 할당된 PublicIP로의 SSH 접속 결과 — `OK`/`FAIL`/`NO_IP`/`NO_KEY` |
| `PublicIP(Unassigned)` | `UnassignVMDefaultPublicIP` 이후의 PublicIP — 다시 `(none)`이어야 함 |
| `Elapsed` | 경과 시간 |
| `Reason` | FAIL 사유 (예: PublicIP가 예상치 못하게 존재/부재, 타임아웃, Failed 상태, Assign/Unassign API 에러, SSH 실패 등) |

## KT 관련 주의사항

KT Cloud VPC 드라이버는 Security Group을 오직 Public IP 기반
PortForwarding/Firewall 규칙으로만 구현합니다. 따라서 `AssignPublicIP: false`
로 생성된 KT VM은 요청한 SecurityGroup이 네트워크 레벨에서 적용되지
않습니다 (VM 자체는 정상적으로 PrivateIP만으로 생성됩니다). 이 테스트는
`Running` 도달 여부와 PrivateIP/PublicIP만 검증하며, SG 적용 여부는
검증 범위가 아닙니다.

## Script Structure

```
assign-publicip-false/
├── run-all-csp-assign-publicip-false-tests.sh   # Orchestrator: 전체 CSP 병렬 실행
├── delete-all-csp-assign-publicip-false-vm.sh   # Orchestrator: 전체 CSP 병렬 삭제 (../common-vm-delete.sh 재사용)
├── common-assign-publicip-false-test.sh         # Common: Create -> Poll Running -> Verify -> Assign -> Verify -> SSH -> Unassign -> Verify
├── aws-assign-publicip-false-test.sh
├── azure-assign-publicip-false-test.sh
├── gcp-assign-publicip-false-test.sh
├── alibaba-assign-publicip-false-test.sh
├── tencent-assign-publicip-false-test.sh
├── ibm-assign-publicip-false-test.sh
├── openstack-assign-publicip-false-test.sh
├── ncp-assign-publicip-false-test.sh
├── nhn-assign-publicip-false-test.sh
└── kt-assign-publicip-false-test.sh
```

## Logs & Results

```
/tmp/vm_nopublicip_test_<PID>/results/result_<csp>.txt
/tmp/vm_nopublicip_test_<PID>/logs/log_<csp>.txt
```

전체 삭제 실행 시:
```
/tmp/vm_nopublicip_delete_results_<PID>/result_<csp>.txt
/tmp/vm_nopublicip_delete_logs_<PID>/log_<csp>.txt
```

## API Reference

| Operation | Method | Path |
|-----------|--------|------|
| CreateVM | `POST` | `/spider/vm` |
| GetVMStatus | `GET` | `/spider/vmstatus/{Name}?ConnectionName=` |
| GetVM | `GET` | `/spider/vm/{Name}?ConnectionName=` |
| AssignVMDefaultPublicIP | `POST` | `/spider/vm/{Name}/publicip` |
| UnassignVMDefaultPublicIP | `DELETE` | `/spider/vm/{Name}/publicip` |
| DeleteVM | `DELETE` | `/spider/vm/{Name}` |

## 시험 결과

### 2026-08-26

`./run-all-csp-assign-publicip-false-tests.sh` 전체 실행 (10개 CSP 병렬) — **10/10 PASS**.

```
CSP         | Result | VMStatus   | PrivateIP       | PubIP(init)   | PubIP(assign)   | SSH    | PubIP(unassign) | Elapsed  | Reason
------------------------------------------------------------------------------------------------------------------------------------------------------------------------
AWS         | PASS   | Running    | 192.168.1.68    | (none)        | 15.134.75.151   | OK     | (none)          | 1m20s    | -
AZURE       | PASS   | Running    | 192.168.0.4     | (none)        | 20.194.108.182  | OK     | (none)          | 1m44s    | -
GCP         | PASS   | Running    | 192.168.1.2     | (none)        | 34.121.181.229  | OK     | (none)          | 1m51s    | -
ALIBABA     | PASS   | Running    | 192.168.1.235   | (none)        | 39.105.135.173  | OK     | (none)          | 46s      | -
TENCENT     | PASS   | Running    | 192.168.1.8     | (none)        | 43.138.15.142   | OK     | (none)          | 1m50s    | -
IBM         | PASS   | Running    | 192.168.1.4     | (none)        | 150.239.80.216  | OK     | (none)          | 1m15s    | -
OPENSTACK   | PASS   | Running    | 192.168.1.128   | (none)        | 183.111.177.144 | OK     | (none)          | 2m36s    | -
NCP         | PASS   | Running    | 192.168.1.6     | (none)        | 101.79.28.3     | OK     | (none)          | 4m10s    | -
NHN         | PASS   | Running    | 192.168.1.74    | (none)        | 133.186.247.207 | OK     | (none)          | 2m43s    | -
KT          | PASS   | Running    | 10.29.102.133   | (none)        | 211.34.246.113  | OK     | (none)          | 4m31s    | -
------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Total: 10  PASS: 10  FAIL: 0
```

- 10개 CSP 전부 `AssignPublicIP=false`로 생성 → PublicIP 없이 Running → `AssignVMDefaultPublicIP` → PublicIP 할당 확인 → SSH 접속 확인 → `UnassignVMDefaultPublicIP` → PublicIP 해제 확인까지 전 단계 통과.
