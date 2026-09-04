# CB-Spider VM AssignPublicIP=true Test

`ReqInfo.AssignPublicIP: true` 옵션으로 VM을 생성했을 때, 생성 직후부터 Public
IP가 정상적으로 할당되어 SSH 접속이 가능하며, 이렇게 만든 VM에서 VM Manager의
default-PublicIP API(`UnassignVMDefaultPublicIP`/`AssignVMDefaultPublicIP`)로
Public IP를 뗐다 다시 붙일 수 있는지를 10개 CSP 전체에서 검증하는 테스트
스위트입니다. `../assign-publicip-false/`와 상위 디렉터리
`test/vm-default-publicip-test/`를 참고하여 작성했습니다.

## Purpose

VM 생성 REST API의 `AssignPublicIP`(옵션, `*bool`) 필드가 `true`로 설정되었을
때, 그리고 그렇게 만든 VM에서 나중에 VM Manager의 default-PublicIP API로
Public IP를 뗐다 다시 붙일 수 있는지를 검증합니다 (`true` → `false` → `true`):

1. VM이 `Running` 상태에 정상적으로 도달하는가 (타임아웃/Failed면 실패)
2. VM에 `PrivateIP`가 정상적으로 할당되는가
3. VM에 `PublicIP`가 생성 직후부터 **할당되는가** (값이 비어있으면 실패)
4. 할당된 초기 Public IP로 SSH 접속 확인
5. `UnassignVMDefaultPublicIP` 요청 (`DELETE /vm/{Name}/publicip`)
6. Public IP가 해제되었는가 확인 (값이 존재하면 실패)
7. `AssignVMDefaultPublicIP` 요청 (`POST /vm/{Name}/publicip`)
8. Public IP가 다시 할당되었는가 확인 (값이 비어있으면 실패)
9. 재할당된 Public IP로 SSH 접속 재확인

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
cd assign-publicip-true
```

### Required Tools

- `bash` 3.2+
- `curl`
- `jq`
- `ssh` client (OpenSSH) — for the SSH-login checks (initial + after re-assign)

## Test Flow

각 CSP에 대해 다음 순서로 검증합니다:

1. **CreateVM** — `POST /spider/vm` (`ReqInfo.AssignPublicIP: true` 포함) —
   `cb-spider-truepublicip-test` 인스턴스 생성
2. **Poll Running** — `GET /spider/vmstatus/{Name}` — `Running` 상태가 될
   때까지 폴링 (기본 15초 간격, 최대 1800초). `Failed` 상태이거나 타임아웃이면
   즉시 FAIL
3. **GetVM** — `GET /spider/vm/{Name}` — `PrivateIP`/`PublicIP` 조회
4. **Verify (has PublicIP)** — `PrivateIP`가 비어있으면 FAIL, `PublicIP`가
   비어있으면 FAIL
5. **SSH check (initial)** — 초기 Public IP + 상위 디렉터리에서 준비한
   KeyPair의 PrivateKey로 SSH 접속 시도 (기본 10회 재시도, 30초 간격). 접속
   실패(`FAIL`)는 물론, PrivateKey 파일이 없어서 시도 자체를 못한 경우
   (`NO_KEY`)와 Public IP가 비어 있어 시도할 수 없는 경우(`NO_IP`)도 접속
   가능 여부를 확인할 수 없으므로 모두 전체 FAIL 처리
6. **UnassignVMDefaultPublicIP** — `DELETE /vm/{Name}/publicip` — Public IP
   해제(+ 삭제) 요청
7. **Verify (unassigned)** — `GetVM`으로 `PublicIP`가 비었는지 확인, 남아있으면
   FAIL
8. **AssignVMDefaultPublicIP** — `POST /vm/{Name}/publicip` — Public IP 자동
   생성+재할당 요청
9. **Verify (reassigned)** — `GetVM`으로 `PublicIP`가 다시 채워졌는지 확인,
   비어있으면 FAIL
10. **SSH check (reassigned)** — 재할당된 Public IP로 SSH 접속 재확인. 규칙은
    4번과 동일
11. 인스턴스는 자동 삭제되지 않습니다 —
    `delete-all-csp-assign-publicip-true-vm.sh`로 별도 정리

## How to Run Tests

### All CSPs in Parallel

```bash
./run-all-csp-assign-publicip-true-tests.sh
```

**Example output:**
```
======================================================================================================================================================================================================
                    VM AssignPublicIP=true TEST SUMMARY - ALL CSPs
======================================================================================================================================================================================================

CSP         | Result | VMStatus   | PrivateIP       | PubIP(init)     | SSH    | PubIP(unassign) | PubIP(reassign) | SSH2   | Elapsed  | Reason
------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
AWS         | PASS   | Running    | 192.168.1.10    | 3.34.12.55      | OK     | (none)          | 15.164.10.20    | OK     | 4m30s    | -
AZURE       | PASS   | Running    | 192.168.0.5     | 20.196.55.10    | OK     | (none)          | 20.196.60.30    | OK     | 6m12s    | -
...
------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Total: 10  PASS: 10  FAIL: 0
```

### Individual CSP

```bash
./aws-assign-publicip-true-test.sh
./azure-assign-publicip-true-test.sh
./gcp-assign-publicip-true-test.sh
./alibaba-assign-publicip-true-test.sh
./tencent-assign-publicip-true-test.sh
./ibm-assign-publicip-true-test.sh
./openstack-assign-publicip-true-test.sh
./ncp-assign-publicip-true-test.sh
./nhn-assign-publicip-true-test.sh
./kt-assign-publicip-true-test.sh
```

단독 실행 시 결과 디렉터리는 `RESULT_DIR` 환경변수로 지정하거나 기본값
(`/tmp/vm_truepublicip_results`)이 사용됩니다.

### Delete All Test VMs

```bash
./delete-all-csp-assign-publicip-true-vm.sh
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
| `SSH_USER` | `cb-user` | OS login user for the SSH-login checks |
| `SSH_MAX_ATTEMPTS` | `10` | SSH connection retry count |
| `SSH_RETRY_INTERVAL` | `30` | Seconds between SSH retries |
| `VERBOSE` | `0` | `1`로 설정 시 CSP별 전체 로그 덤프 출력 |

## Result Format

결과 파일(`result_<csp>.txt`)은 파이프(|) 구분 11개 필드:

```
CSP|Result|VMStatus|PrivateIP|PublicIP(Initial)|SSH(Initial)|PublicIP(Unassigned)|PublicIP(Reassigned)|SSH(Reassigned)|Elapsed|Reason
```

| 필드 | 설명 |
|------|------|
| `CSP` | CSP 이름 (예: AWS) |
| `Result` | `PASS` / `FAIL` |
| `VMStatus` | 최종 확인된 상태 (예: `Running`), 실패 시 실패 당시 상태 |
| `PrivateIP` | 생성된 VM의 PrivateIP (`-`는 생성 실패로 조회 불가) |
| `PublicIP(Initial)` | 생성 직후(AssignPublicIP=true)의 PublicIP — 비어있지 않아야 함 |
| `SSH(Initial)` | 초기 PublicIP로의 SSH 접속 결과 — `OK`/`FAIL`/`NO_IP`/`NO_KEY` |
| `PublicIP(Unassigned)` | `UnassignVMDefaultPublicIP` 이후의 PublicIP — `(none)`이어야 함 |
| `PublicIP(Reassigned)` | `AssignVMDefaultPublicIP` 이후의 PublicIP — 비어있지 않아야 함 |
| `SSH(Reassigned)` | 재할당된 PublicIP로의 SSH 접속 결과 — `OK`/`FAIL`/`NO_IP`/`NO_KEY` |
| `Elapsed` | 경과 시간 |
| `Reason` | FAIL 사유 (예: PublicIP가 예상치 못하게 존재/부재, 타임아웃, Failed 상태, Assign/Unassign API 에러, SSH 실패 등) |

## Script Structure

```
assign-publicip-true/
├── run-all-csp-assign-publicip-true-tests.sh   # Orchestrator: 전체 CSP 병렬 실행
├── delete-all-csp-assign-publicip-true-vm.sh   # Orchestrator: 전체 CSP 병렬 삭제 (../common-vm-delete.sh 재사용)
├── common-assign-publicip-true-test.sh         # Common: Create -> Poll Running -> Verify -> SSH -> Unassign -> Verify -> Assign -> Verify -> SSH
├── aws-assign-publicip-true-test.sh
├── azure-assign-publicip-true-test.sh
├── gcp-assign-publicip-true-test.sh
├── alibaba-assign-publicip-true-test.sh
├── tencent-assign-publicip-true-test.sh
├── ibm-assign-publicip-true-test.sh
├── openstack-assign-publicip-true-test.sh
├── ncp-assign-publicip-true-test.sh
├── nhn-assign-publicip-true-test.sh
└── kt-assign-publicip-true-test.sh
```

## Logs & Results

```
/tmp/vm_truepublicip_test_<PID>/results/result_<csp>.txt
/tmp/vm_truepublicip_test_<PID>/logs/log_<csp>.txt
```

전체 삭제 실행 시:
```
/tmp/vm_truepublicip_delete_results_<PID>/result_<csp>.txt
/tmp/vm_truepublicip_delete_logs_<PID>/log_<csp>.txt
```

## API Reference

| Operation | Method | Path |
|-----------|--------|------|
| CreateVM | `POST` | `/spider/vm` |
| GetVMStatus | `GET` | `/spider/vmstatus/{Name}?ConnectionName=` |
| GetVM | `GET` | `/spider/vm/{Name}?ConnectionName=` |
| UnassignVMDefaultPublicIP | `DELETE` | `/spider/vm/{Name}/publicip` |
| AssignVMDefaultPublicIP | `POST` | `/spider/vm/{Name}/publicip` |
| DeleteVM | `DELETE` | `/spider/vm/{Name}` |

## 시험 결과

### 2026-09-04

`./run-all-csp-assign-publicip-true-tests.sh` 전체 실행 (10개 CSP 병렬) — **10/10 PASS**.

```
CSP         | Result | VMStatus   | PrivateIP       | PubIP(init)     | SSH    | PubIP(unassign) | PubIP(reassign) | SSH2   | Elapsed  | Reason
--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
AWS         | PASS   | Running    | 192.168.1.154   | 3.25.115.56     | OK     | (none)          | 3.24.171.164    | OK     | 1m55s    | -
AZURE       | PASS   | Running    | 192.168.0.4     | 40.82.134.16    | OK     | (none)          | 52.141.56.178   | OK     | 1m36s    | -
GCP         | PASS   | Running    | 192.168.1.9     | 136.113.32.98   | OK     | (none)          | 34.136.155.127  | OK     | 2m17s    | -
ALIBABA     | PASS   | Running    | 192.168.1.127   | 123.56.91.87    | OK     | (none)          | 39.96.59.58     | OK     | 1m4s     | -
TENCENT     | PASS   | Running    | 192.168.1.9     | 101.43.140.61   | OK     | (none)          | 82.157.148.206  | OK     | 1m37s    | -
IBM         | PASS   | Running    | 192.168.1.6     | 150.239.80.67   | OK     | (none)          | 169.63.102.8    | OK     | 2m10s    | -
OPENSTACK   | PASS   | Running    | 192.168.1.39    | 183.111.177.156 | OK     | (none)          | 183.111.177.137 | OK     | 3m26s    | -
NCP         | PASS   | Running    | 192.168.1.6     | 49.50.143.235   | OK     | (none)          | 49.50.143.236   | OK     | 7m29s    | -
NHN         | PASS   | Running    | 192.168.1.101   | 103.218.159.89  | OK     | (none)          | 133.186.228.111 | OK     | 3m22s    | -
KT          | PASS   | Running    | 10.29.102.141   | 210.104.76.98   | OK     | (none)          | 210.104.76.98   | OK     | 4m53s    | -
--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Total: 10  PASS: 10  FAIL: 0
```

- 10개 CSP 전부 `AssignPublicIP=true`로 생성 → 생성 직후 PublicIP 할당 확인 → SSH 접속 확인 → `UnassignVMDefaultPublicIP` → PublicIP 해제 확인 → `AssignVMDefaultPublicIP` → PublicIP 재할당 확인 → SSH 재접속 확인까지 전 단계 통과.
