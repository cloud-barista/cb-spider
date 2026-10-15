# CB-Spider VM Default-PublicIP Test

Automated test suite for CB-Spider's VM API — creates a VM on all 10 CSPs in
parallel, confirms it reaches `Running` with a default PublicIP and that it
is actually SSH-reachable on that IP, then suspends and resumes it to check
whether the PublicIP (and SSH reachability) is retained or changes.
Modeled on `test/rdbms-mysql-test/`.

## Purpose

Answers, per CSP, empirically:

1. Does the VM get a default PublicIP automatically when created (no
   explicit PublicIP resource requested by the caller), and can you actually
   SSH into it using the KeyPair created in step 1 (basic-resource-prepare)?
2. When the VM is **Suspended**, does it still have a PublicIP?
3. When the VM is **Resumed**, does it come back with the **same** PublicIP,
   a **different** one, or **none** — and can you SSH into it again using the
   PublicIP obtained after resume?

## Prerequisites

### CB-Spider Running

```bash
cd ./bin; ./start.sh
```

### CSP Connection Configuration

Connection names follow `test/rdbms-mysql-test/README.md` (KT added, since
rdbms-mysql-test does not cover KT — its connection name is taken from
`spiderwatch/conf/spiderwatch.yaml` instead).

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
| KT | `kt-mokdong1-config` | (per spiderwatch.yaml) | |

### VM Image / Spec Configuration

Taken from `spiderwatch/conf/spiderwatch.yaml` → `csps[].vm_test`.

| CSP | Image | Spec |
|-----|-------|------|
| AWS | `ami-0131a0fdbb6fda7e6` | `t2.micro` |
| Azure | `Canonical:ubuntu-25_04-daily:minimal:25.04.202601140` | `Standard_B1ls` |
| GCP | `.../ubuntu-2404-noble-amd64-v20240423` | `e2-standard-2` |
| Alibaba | latest public image with prefix `ubuntu_24_04_x64_20G_alibase_` (resolved at runtime via `GET /vmimage`, since Alibaba rotates this monthly) | `ecs.c9i.large` |
| Tencent | `img-pi0ii46r` | `S5.MEDIUM8` |
| IBM | `r014-1696a049-e959-493d-9a97-1655ef4c942e` | `bx2-2x8` |
| OpenStack | `78d90dae-d21d-4606-a9dd-c1268e321864` | `m1.small` |
| NCP | `104630229` | `s2-g3` |
| NHN | `5396655e-166a-4875-80d2-ed8613aa054f` | `m2.c4m8` |
| KT | `1a772df6-262e-43a7-896f-98fa23d715c7` | `4x8.itl` |

### Basic Resources Created Per CSP

Each CSP gets its own `vpc-01`/`subnet-01`, `sg-01` (inbound TCP 22 from
`0.0.0.0/0`), and `keypair-01` — created by `run-all-csp-network-prepare.sh`
(idempotent: skips and reports `EXISTS` if already present).

| CSP | VPC CIDR | Subnet CIDR |
|-----|----------|--------------|
| AWS / GCP / Alibaba / Tencent / IBM / OpenStack / NCP / NHN | `192.168.0.0/16` | `192.168.1.0/24` |
| Azure | `192.168.0.0/16` | `192.168.0.0/23` |
| KT | `10.0.0.0/16` | `10.29.102.0/24` |

### Required Tools

- `bash` 3.2+
- `curl`
- `jq`
- `ssh` client (OpenSSH) — for the SSH-login check

## Configuration

```bash
export SPIDER_URL=http://localhost:1024   # CB-Spider REST API URL
export SPIDER_AUTH=admin:*****           # Basic auth (admin:<password>)
```

Or edit the defaults directly inside `run-all-csp-*.sh` / `delete-all-csp-*.sh`.

## How to Run

### Full Suite (Prepare -> Test -> Delete -> Cleanup)

```bash
./all_test.sh
```

### Step by Step

```bash
# 1) Basic resources: VPC/Subnet -> SecurityGroup -> KeyPair
./run-all-csp-network-prepare.sh

# 2) Create VM -> wait Running -> record PublicIP
#    -> Suspend -> record PublicIP -> Resume -> record PublicIP -> compare
./run-all-csp-vm-publicip-tests.sh

# 3) Delete the VM instances
./delete-all-csp-vm.sh

# 4) Clean up basic resources
./delete-all-csp-network.sh
```

### Run Individual CSP

```bash
./aws-network-prepare.sh
./aws-vm-publicip-test.sh
```

`RESULT_DIR` defaults to `/tmp/vm_publicip_network_results` (prepare) and
`/tmp/vm_publicip_results` (test) when run standalone; the orchestrators use
a PID-suffixed temp dir instead.

## Test Flow (per CSP, `common-vm-publicip-test.sh`)

1. `POST /spider/vm` — create the VM (no PublicIP-related field in the
   request; whatever the CSP driver does by default is what's being tested).
2. Poll `GET /spider/vmstatus/{Name}` until `Running`.
3. `GET /spider/vm/{Name}` → record `PublicIP` as **Initial**.
4. **SSH-login check** using `keypair-01`'s PrivateKey (saved by
   `common-network-prepare.sh` to `${KEY_DIR}/${ConnectionName}.pem`) against
   the Initial PublicIP, as user `cb-user` (CB-Spider's default cloud-init
   account) — retried up to `SSH_MAX_ATTEMPTS` times to allow for cloud-init
   provisioning delay.
5. `GET /spider/controlvm/{Name}?action=suspend` → poll until `Suspended`.
6. `GET /spider/vm/{Name}` → record `PublicIP` as **Suspended** (often empty).
7. `GET /spider/controlvm/{Name}?action=resume` → poll until `Running`.
8. `GET /spider/vm/{Name}` → record `PublicIP` as **Resumed**.
9. **SSH-login check** again, this time against the Resumed PublicIP.
10. Compare Initial vs Resumed PublicIP:
    - `SAME` — same PublicIP kept across suspend/resume.
    - `CHANGED` — different PublicIP after resume.
    - `CHANGED_TO_NONE` — had a PublicIP on one side, none on the other.
    - `NONE_BOTH` — never had a PublicIP (e.g. private-subnet VM).

Each SSH-login check result is one of:
- `OK` — SSH login succeeded.
- `FAIL` — login failed after `SSH_MAX_ATTEMPTS` retries.
- `NO_IP` — there was no PublicIP to connect to at that point.
- `NO_KEY` — the PrivateKey file was missing (e.g. the KeyPair already
  existed from a previous run whose `.pem` file was cleaned up — CSPs only
  return the PrivateKey once, at creation time).

## Script Structure

```
.
├── run-all-csp-network-prepare.sh   # Orchestrator: VPC/Subnet/SG/KeyPair prepare (parallel)
├── delete-all-csp-network.sh        # Orchestrator: KeyPair/SG/VPC cleanup (parallel)
├── common-network-prepare.sh        # Common: VPC/Subnet -> SG -> KeyPair
├── common-network-cleanup.sh        # Common: KeyPair -> SG -> VPC/Subnet
├── <csp>-network-prepare.sh         # x10
├── run-all-csp-vm-publicip-tests.sh # Orchestrator: Create/Suspend/Resume test (parallel)
├── delete-all-csp-vm.sh             # Orchestrator: VM delete (parallel)
├── common-vm-publicip-test.sh       # Common: Create -> Running -> SSH -> Suspend -> Resume -> SSH -> compare
├── common-vm-delete.sh              # Common: Terminate -> poll removed
├── <csp>-vm-publicip-test.sh        # x10 (alibaba resolves its image at runtime)
└── all_test.sh                      # Full suite: prepare -> test -> delete -> cleanup
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPIDER_URL` | `http://localhost:1024` | CB-Spider REST API URL |
| `SPIDER_AUTH` | `admin:*****` | Basic auth credentials |
| `MAX_WAIT_SEC` | `1800` | Timeout waiting for `Running` after create |
| `SUSPEND_MAX_WAIT_SEC` | `600` | Timeout waiting for `Suspended` |
| `RESUME_MAX_WAIT_SEC` | `600` | Timeout waiting for `Running` after resume |
| `POLL_INTERVAL` | `15` | Polling interval (seconds) |
| `KEY_DIR` | `/tmp/vm_publicip_keys` | Directory holding the saved PrivateKey files (`${KEY_DIR}/${ConnectionName}.pem`); shared between `common-network-prepare.sh` (writer), `common-vm-publicip-test.sh` (reader), and `common-network-cleanup.sh` (deletes it) |
| `SSH_USER` | `cb-user` | OS login user for the SSH-login check |
| `SSH_MAX_ATTEMPTS` | `10` | SSH connection retry count |
| `SSH_RETRY_INTERVAL` | `30` | Seconds between SSH retries |
| `VERBOSE` | `0` | Set to `1` for per-CSP full log dump |

## Logs & Results

```
/tmp/vm_publicip_results_<PID>/result_<csp>.txt   # pipe-separated result line
/tmp/vm_publicip_logs_<PID>/log_<csp>.txt         # per-CSP full output
```

```bash
tail -f /tmp/vm_publicip_logs_<PID>/log_aws.txt
```

## 시험 결과

### 2026-08-26

`./all_test.sh` 전체 실행 (10개 CSP, Prepare → Test → Delete → Cleanup) — 4단계 전부 PASS.

```
CSP         | Result | PublicIP(Initial) | PublicIP(Suspend) | PublicIP(Resume)  | Initial->Resume  | SSH(Init) | SSH(Resume) | Elapsed
------------------------------------------------------------------------------------------------------------------------------------------------------------------------
AWS         | OK     | 54.252.157.228    | (none)            | 32.236.157.83     | CHANGED          | OK        | OK          | 2m22s
AZURE       | OK     | 20.196.139.24     | 20.196.139.24     | 20.196.139.24     | SAME             | OK        | OK          | 2m30s
GCP         | OK     | 35.255.77.150     | (none)            | 35.255.77.150     | SAME             | OK        | OK          | 3m21s
ALIBABA     | OK     | 47.94.229.45      | (none)            | 47.94.229.45      | SAME             | OK        | OK          | 1m21s
TENCENT     | OK     | 192.144.167.240   | 192.144.167.240   | 192.144.167.240   | SAME             | OK        | OK          | 3m9s
IBM         | OK     | 150.239.83.41     | 150.239.83.41     | 150.239.83.41     | SAME             | OK        | OK          | 2m13s
OPENSTACK   | OK     | 183.111.177.137   | 183.111.177.137   | 183.111.177.137   | SAME             | OK        | OK          | 4m7s
NCP         | OK     | 175.45.194.155    | 175.45.194.155    | 175.45.194.155    | SAME             | OK        | OK          | 4m53s
NHN         | OK     | 133.186.159.190   | 133.186.159.190   | 133.186.159.190   | SAME             | OK        | OK          | 2m57s
KT          | OK     | 211.34.246.115    | 211.34.246.115    | 211.34.246.115    | SAME             | OK        | OK          | 2m42s
------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Failed : 0
```

- 10개 CSP 전부 VM 생성, Suspend, Resume, SSH 로그인(생성 직후 + Resume 직후) 성공.
- **AWS만 Resume 후 PublicIP가 바뀜(`CHANGED`)** — 나머지 9개 CSP(Azure/GCP/Alibaba/Tencent/IBM/OpenStack/NCP/NHN/KT)는 전부 `SAME`(동일 IP 유지).
- GCP/Alibaba는 Suspend 중 조회된 PublicIP가 `(none)`이었으나 Resume 후 Initial과 동일 IP로 복귀 — 최종 비교는 `SAME`.
- **KT 소요 시간이 6m19s(2026-08-20) → 2m42s로 크게 단축** — `GetVM()`이 매 호출마다 NIC별로 실행하던 `ktports.Get()`/`ktl3fips.List()`(KT VPC에서 불안정하거나 사실상 무의미했던 Neutron 계열 호출)를 제거하고, 공인IP 매핑은 KT VPC가 실제로 쓰는 PortForwarding 목록을 재사용하도록 정리한 결과로 보임.
- VM Delete, 기본 리소스(KeyPair/SG/VPC) Cleanup도 10개 CSP 전부 정상 완료.

### 2026-08-20

`./all_test.sh` 전체 실행 (10개 CSP, Prepare → Test → Delete → Cleanup) — 4단계 전부 PASS.

```
CSP         | Result | PublicIP(Initial) | PublicIP(Suspend) | PublicIP(Resume)  | Initial->Resume  | SSH(Init) | SSH(Resume) | Elapsed
------------------------------------------------------------------------------------------------------------------------------------------------------------------------
AWS         | OK     | 3.27.184.213      | (none)            | 3.107.68.139      | CHANGED          | OK        | OK          | 2m13s
AZURE       | OK     | 20.196.113.231    | 20.196.113.231    | 20.196.113.231    | SAME             | OK        | OK          | 3m28s
GCP         | OK     | 136.64.45.57      | (none)            | 35.253.51.181     | CHANGED          | OK        | OK          | 3m30s
ALIBABA     | OK     | 123.56.91.45      | (none)            | 123.56.91.45      | SAME             | OK        | OK          | 1m25s
TENCENT     | OK     | 81.70.99.119      | 81.70.99.119      | 81.70.99.119      | SAME             | OK        | OK          | 2m32s
IBM         | OK     | 150.239.83.195    | 150.239.83.195    | 150.239.83.195    | SAME             | OK        | OK          | 1m57s
OPENSTACK   | OK     | 183.111.177.159   | 183.111.177.159   | 183.111.177.159   | SAME             | OK        | OK          | 4m26s
NCP         | OK     | 211.188.50.212    | 211.188.50.212    | 211.188.50.212    | SAME             | OK        | OK          | 4m40s
NHN         | OK     | 133.186.210.74    | 133.186.210.74    | 133.186.210.74    | SAME             | OK        | OK          | 3m12s
KT          | OK     | 211.34.246.118    | 211.34.246.118    | 211.34.246.118    | SAME             | OK        | OK          | 6m19s
------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Failed : 0
```

- 10개 CSP 전부 VM 생성, Suspend, Resume, SSH 로그인(생성 직후 + Resume 직후) 성공.
- **AWS, GCP만 Resume 후 PublicIP가 바뀜(`CHANGED`)** — 나머지 8개 CSP(Azure/Alibaba/Tencent/IBM/OpenStack/NCP/NHN/KT)는 전부 `SAME`(동일 IP 유지).
- Alibaba는 Suspend 중 조회된 PublicIP가 `(none)`이었으나 Resume 후 Initial과 동일 IP로 복귀 — 최종 비교는 `SAME`.
- VM Delete, 기본 리소스(KeyPair/SG/VPC) Cleanup도 10개 CSP 전부 정상 완료.
