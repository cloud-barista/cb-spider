# cb-spider
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/cloud-barista/cb-spider?label=go.mod)](https://github.com/cloud-barista/cb-spider/blob/master/go.mod)
[![GoDoc](https://godoc.org/github.com/cloud-barista/cb-spider?status.svg)](https://pkg.go.dev/github.com/cloud-barista/cb-spider@master)&nbsp;&nbsp;&nbsp;
[![Release Version](https://img.shields.io/github/v/release/cloud-barista/cb-spider)](https://github.com/cloud-barista/cb-spider/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/cloud-barista/cb-spider/blob/master/LICENSE)

CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.<br>
The CB-Spider Mission is to connect all the clouds with a single interface.


```
[NOTE]
CB-Spider is currently under development. (the latest version is 0.3.0 espresso)
So, we do not recommend using the current release in production.
Please note that the functionalities of CB-Spider are not stable and secure yet.
If you have any difficulties in using CB-Spider, please let us know.
(Open an issue or Join the cloud-barista Slack)
```
***
### ▶ **[Quick Guide](https://github.com/cloud-barista/cb-spider/wiki/Quick-Start-Guide)**
***

#### [목    차]

1. [실행 환경](#실행-환경)
2. [실행 방법](#실행-방법)
3. [API 규격](#API-규격)
4. [제공 자원](#제공-자원)
5. [활용 예시](#활용-예시)
6. [특이 사항](#특이-사항)
7. [관련 정보](#관련-정보)
 
***

#### [실행 환경]

- #### 공식환경
  - OS: Ubuntu 20.04
  - Container: Docker 19.03
  - Build: Go 1.15
- #### 시험환경
  - OS: Ubuntu 18.04, Ubuntu 20.04, Debian 10.6, macOS Catalina 10.15, Android 8.1 등
  - Container: latest Docker
  - Build: latest Go


#### [실행 방법]

- ##### 소스 기반 실행: https://github.com/cloud-barista/cb-spider/wiki/Quick-Start-Guide
- ##### 컨테이너 기반 실행: https://github.com/cloud-barista/cb-spider/wiki/Docker-based-Start-Guide
- ##### cb-operator 기반 실행: https://github.com/cloud-barista/cb-operator

#### [API 규격]
- 클라우드 인프라 연동 정보 관리: https://cloud-barista.github.io/rest-api/v0.3.0/spider/ccim/
  - 관리대상: Cloud Driver / Credential / Region:Zone
- 클라우드 인프라 공통 제어 관리: https://cloud-barista.github.io/rest-api/v0.3.0/spider/cctm/
  - 제어대상: Image / Spec / VPC/Subnet / SecurityGroup / KeyPair / VM

#### [제공 자원] 

  | Provider(CloudOS) | VM Image List/Get | VM Spec List/Get| VPC/Subnet | Security Group | VM KeyPair| VM   |
  |:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|
  | AWS           | O          | O          | O          | O          | O          | O          |
  | Azure         | O          | O          | O          | O          | O          | O          |
  | GCP           | O          | O          | O          | O          | O          | O          |
  | Alibaba       | O          | O          | O          | O          | O          | O          |
  | OpenStack     | O          | O          | O          | O          | O          | O          |
  | Cloudit       | O          | O          | O(💬)          | O          | -          | O          |
  | Docker        | O          | -          | -          | -          | -          | O          |

    💬 특이사항: 
        - VPC: 단일 VPC 제공 
        - CIDR: 사용자 설정과 무관하게, CSP 내부에서 유휴 CIDR 할당 후 반납
    
- #### 시험 결과: https://github.com/cloud-barista/cb-spider/wiki/Test-Reports-of-v0.3.0-espresso

#### [활용 예시]
- 시험 도구: `cb-spider/api-runtime/rest-runtime/test/[fulltest|image-test|spec-test|eachtest|parallel-test]` (AWS 경우:aws)
- 시험 순서: 연동 정보 추가 => 자원등록 => VM 생성 및 제어 시험
- 시험 방법: 
  - (연동정보관리) `cb-spider/api-runtime/rest-runtime/test/connect-config` 참고(Credential 정보 수정 후 실행)
  - (자원관리) `cb-spider/api-runtime/rest-runtime/test/fulltest` 아래 자원 별 시험 스크립트 존재
    - (자원관리) 자원 생성 순서
    - (1) vpc, security group, keypair 생성
    - (2) vm 생성 및 제어
    - (3)	삭제는 자원 생성 역순
- CSP별 VM User 

  | CSP        | user ID          | 비고 |
  |:-------------:|:-------------:|:-------------|
  | AWS      | ubuntu 또는 ec2-user 등 | Image에 의존적 |
  | Azure      | cb-user | Spider에서 고정 |
  | GCP      | cb-user      | Spider에서 고정  |
  | Alibaba | root      |   CSP에서 고정, PW 설정 가능 |
  | OpenStack | ubuntu 등     |    Image에 의존적 |
  | Cloudit | root      | sshkey 제공 안함. PW 설정 가능  |
    - 개선예정(관련이슈:https://github.com/cloud-barista/cb-spider/issues/230)
  
#### [특이 사항]
- 개발상태: 초기 주요 기능 중심 개발추진 중 / 기술개발용 / 상용활용시 보완필요
- Key관리: CSP가 제공하지 않는 경우 Key 자체 생성 및 Key 파일 내부 관리
  - 관리위치: `cb-spider/cloud-driver-libs/.ssh-CSPName/*` (임시방법)
  - 공유서버에서 상시 운영시 보안 이슈 존재

***

#### [관련 정보]
- 위키: https://github.com/cloud-barista/cb-spider/wiki
<details>
<summary> [소스 트리] </summary>

```
.
. go.mod:  imported Go module definition
. Dockerfile: docker image build용
. setup.env: spider 운영에 필요한 환경변수 설정
. develop.env: 개발자 편의위한 alias 설정 등
. build_grpc_idl.sh: gRPC IDL build 스크립트
. build_all_driver_lib.sh: 드라이버 build 스크립트
|-- api-runtime
|   |-- common-runtime: REST 및 gRPC runtime 공통 모듈
|   |-- grpc-runtime: gRPC runtime
|   |   |-- idl: gRPC Interface Definition
|   `-- rest-runtime: REST runtime
|       |-- admin-web: AdminWeb GUI 도구
|       `-- test: REST API 활용 참조 및 시험 도구
|           |-- connect-config: 연결 설정 참조(driver등록 -> credential 등록 -> region 등록 -> connection config 등록)
|           |-- each-test: 자원별 기능 시험 참조(VPC->SecurityGroup->KeyPair->VM)
|           |-- full-test: 모든 자원 전체 기능 시험 참조(create -> list -> get -> delete)
|           |-- 0.full-liststatus-test: 모든 VM 상태 정보 제공 스크립트
|           |-- 1.full-create-test: 모든 자원 생성까지 시험 참조(VPC->SecurityGroup->KeyPair->VM)
|           |-- 2.full-suspend-test: 모든 VM suspend 상태 시험 스크립트
|           |-- 3.full-resume-test: 모든 VM suspend 상태 시험 스크립트
|           |-- 4.full-delete-test
|           |-- docker: Docker Driver 개발 시험 스크립트
|           |-- parallel-test: 동시 실행 시험 스크립트
|           |-- pocketman: Americano 오픈 행사 시현용, Raspberry 환경 운영
|           `-- vm-ssh: 생성된 VM에 대한 ssh/scp REST API 시험 스크립트

|-- cloud-info-manager
|   |-- driver-info-manager: 드라이버 정보 관리
|   |-- credential-info-manager: 크리덴셜 정보 관리
|   |-- region-info-manager: 리젼 정보 관리
|   |-- connection-config-info-manager: 연결 설정 정보 관리(연결설정=드라이버+크리덴셜+리젼)

|-- cloud-control-manager
|   |-- cloud-driver
|   |   |-- call-log: CSP API 호출 이력 정보 수집을 위한 로거, 드라이버 내부에서 활용 
|   |   |   |-- gen4test: HisCall 서버 운영 시험을 위한 CallLog 자동 발생기 
|   |   |-- drivers: 드라이버 구현체 위치(*-plugin: dynamic plugin, shared-library)
|   |   |   |-- alibaba | alibaba-plugin: Alibaba 드라이버 
|   |   |   |-- aws | aws-plugin: AWS 드라이버
|   |   |   |-- azure | azure-plugin: Azure 드라이버 
|   |   |   |-- cloudit | cloudit-plugin: Cloudit 드라이버
|   |   |   |-- gcp | gcp-plugin: GCP 드라이버 
|   |   |   |-- docker | docker-plugin: Docker 드라이버
|   |   |   |-- openstack | openstack-plugin: OpenStack 드라이버 
|   |   |   |-- mock: 서버 기능 시험 및 CI 시험 환경 구성을 위한 Mock Driver
|   |   `-- interfaces: 멀티 클라우드 연동 드라이버 인터페이스(드라이버 공통 인터페이스)
|   |       |-- connect
|   |       `-- resources
|   |-- iid-manager: Integrated ID 관리, IID 구조:{User-defined ID, System-defined ID(CSP ID)}
|   `-- vm-ssh: VM에 대한 SSH/SCP 기능 제공
|-- cloud-driver-libs: 드라이버 공유 라이브러리, SSH Key 생성 파일 관리 위치
|-- conf: Spider 서버 운영을 위한 설정 정보(spider 서버설정, 메타정보 설정, 로거 설정)

|-- interface
|   |-- api: Go API 기반 응용 개발을 위한 Client Package
|   |-- cli: CLI 기반 운용을 위한 Client Package
|   |   |-- cbadm: cloud-barista 대상 사용자 cli
|   |   `-- spider: spider 대상 사용자 cli
|-- log
|   `-- calllog: CallLog 출력 로그 파일 
|-- meta_db: 메타 정보 local FS(nutsdb) 활용시 저장소 위치
`-- utils
    |-- docker: gRPC API runtime 개발 지원 도구(prometheus, grafana 등) 설정 정보
    |   `-- data
    |       |-- grafana-grpc
    |       `-- prometheus
    `-- import-info: Cloud Driver 및 Region 정보 자동 등록 지원 도구

```
</details>

- 소스 트리 상세 설명 : https://han.gl/3IOVD
