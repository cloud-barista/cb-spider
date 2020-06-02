# cb-spider
CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.

The CB-Spider Mission is to connect all the clouds with a single interface.


```
[NOTE]
CB-Spider is currently under development. (the latest version is 0.2 cappuccino)
So, we do not recommend using the current release in production.
Please note that the functionalities of CB-Spider are not stable and secure yet.
If you have any difficulties in using CB-Spider, please let us know.
(Open an issue or Join the cloud-barista Slack)
```

***

## [목    차]

1. [실행 환경](#실행-환경)
2. [실행 방법](#실행-방법)
3. [API 규격](#API-규격)
4. [활용 예시](#활용-예시)
5. [특이 사항](#특이-사항)
6. [소스 트리](#소스-트리)

***

## [실행 환경]

- 리눅스 (검증시험:Ubuntu 18.04, Raspbian GNU/Linux 10, Android aarch64)

## [실행 방법]

### (1) 컨테이너 기반 실행
- CB-Spider 이미지 확인 (https://hub.docker.com/r/cloudbaristaorg/cb-spider/tags)
- CB-Spider 컨테이너 실행

```
# docker run -p 1024:1024 \
-v /root/go/src/github.com/cloud-barista/cb-spider/meta_db:/root/go/src/github.com/cloud-barista/cb-spider/meta_db \
--name cb-spider \
cloudbaristaorg/cb-spider:v0.1.v
```

### (2) 소스 기반 실행

#### (a) 소스 설치

- Git 설치
- Go 설치 (1.12 이상)  

- Cloud-Barista alliance 설치 (CB-Log)
  - `go get -u -v github.com/cloud-barista/cb-log`
  - https://github.com/cloud-barista/cb-log README를 참고하여 설치 및 설정
  
- Cloud-Barista alliance 설치 (CB-Store)
  - `go get -u -v github.com/cloud-barista/cb-store`
  - https://github.com/cloud-barista/cb-store README를 참고하여 설치 및 설정

- CB-Spider 설치
    - `go get -u -v github.com/cloud-barista/cb-spider`    

- 설치 오류시 참고
    - 오류 메시지: `"panic: /debug/requests is already registered. You may have two independent copies of golang.org/x/net/trace in your binary, trying to maintain separate state. This may involve a vendored copy of golang.org/x/net/trace.”`
    
      - 해결방법: `$ rm -rf $GOPATH/src/go.etcd.io/etcd/vendor/golang.org/x/net/trace`
      
    - 오류 메시지: `"gosrc/src/go.etcd.io/etcd/vendor/google.golang.org/grpc/clientconn.go:49:2: use of internal package google.golang.org/grpc/internal/resolver/dns not allowed"`
    
      - 해결방법: `$ rm -rf $GOPATH/gosrc/src/go.etcd.io/etcd/vendor/google.golang.org/grpc`
      
#### (b) 실행 준비
- CB-Spider 실행에 필요한 환경변수 설정
  - `source setup.env` (위치: ./cb-spider)
     - Android 실행시: export PLUGIN_SW=OFF

-	driver shared library 생성 방법 (설치 시스템 당 1회 실행, driver source 변경시 실행)
    - `./build_all_driver_lib.sh` 실행
    -	결과: `cb-spider/cloud-driver-libs/xxx-driver-v1.0.so` 생성
    - 참고: 특정 CSP driver만 build하는 방법
        - `cd cb-spider/cloud-control-manager/cloud-driver/drivers/aws-plugin` # AWS Driver 경우
        - `build_driver_lib.sh` 실행

#### (c) 서버 실행
- `cd cb-spider/api-runtime/rest-runtime`
-	`go run *.go`    # 1024 포트 REST API Server 실행됨
-	참고: 메타 정보 손상시 초기화 방법
    - `cb-spider/cloud-driver-libs/.ssh-*/*` 파일 삭제
    - `cb-spider/meta_db/dat` 경로 삭제(ex: 0.dat) 후 서버 재가동

### (3) Cloud-Barista 플랫폼 통합 실행 방법 (Docker-Compose 기반)
- cb-operator 참고: https://github.com/cloud-barista/cb-operator

## [API 규격]
- 클라우드 인프라 연동 정보 관리: https://documenter.getpostman.com/view/9027676/SVzz4fb4?version=latest
  - 클라우드 드라이버 정보 관리
  - 클라우드 인프라 인증정보 관리
  - 클라우드 인프라 리젼 정보 관리
- 클라우드 인프라 공통 제어 관리: https://documenter.getpostman.com/view/9027676/SVtSXpzE (update 필요)
  - 이미지 자원 제어
  - 네트워크 자원 제어
  - Security Group 자원 제어  
  - 키페어 자원 제어
  - VM 제어 및 정보 제공
  
## [활용 예시]
- 시험 도구: `cb-spier/api-runtime/rest-runtime/test/[fulltest|eachtest|parallel-test]` (AWS 경우:aws)
- 시험 순서: 연동 정보 추가 => 자원등록 => VM 생성 및 제어 시험
- 시험 방법: 
  - (연동정보관리) `cb-spider/api-runtime/rest-runtime/test/connect-config` 참고(Credential 정보 수정 후 실행)
  - (자원관리) `cb-spider/api-runtime/rest-runtime/test/fulltest` 아래 자원 별 시험 스크립트 존재
    - (자원관리) 자원 생성 순서
    - (1) vpc, security group, keypair 생성
    - (2) vm 생성 및 제어
    - (3)	삭제는 자원 생성 역순
- CSP별 VM User: 2020.05.29.현재 

  | CSP        | user ID          | 비고 |
  |:-------------:|:-------------:|:-------------|
  | AWS      | ubuntu 또는 ec2-user 등 | Image 의존 |
  | Azure      | cb-user | Driver에서 고정 |
  | GCP      | cb-user      | Driver에서 고정  |
  | Alibaba | root      |   CSP에서 고정, PW 설정 가능 |
  | OpenStack | ubuntu 등     |    Image에 의존 |
  | Cloudit | root      | sshkey 제공 안함. PW 설정 가능  |
    - 개선예정(관련이슈:https://github.com/cloud-barista/cb-spider/issues/230)
  
## [특이 사항]
- 개발상태: 초기 주요 기능 중심 개발추진 중 / 기술개발용 / 상용활용시 보완필요
- Key관리: CSP가 제공하지 않는 경우 Key 자체 생성 및 Key 파일 내부 관리
  - 관리위치: `cb-spider/cloud-driver-libs/.ssh-CSPName/*` (임시방법)
  - 공유서버에서 운영시 보안 이슈 존재

***

## [소스 트리]
```
.
. Dockerfile: docker image build용
. setup.env: spider 운영에 필요한 환경변수 설정
. build_all_driver_lib.sh: 드라이버 build 스크립트
|-- api-runtime
|   |-- grpc-runtime: 향후 grpc runtime 들어올 자리
|   `-- rest-runtime: 현재 REST runtime
|       `-- test: REST API 활용 참조
|           |-- connect-config: 연결 설정 참조(driver등록 -> credential 등록 -> region 등록 -> connection config 등록)
|           |-- each-test: 자원별 기능 시험 참조(VPC->SecurityGroup->KeyPair->VM)
|           |-- full-test: 모든 자원 전체 기능 시험 참조(create -> list -> get -> delete)
|           |-- parallel-test: VM 동시 실행 시험 참조(VPC생성 -> SecurityGroup생성 -> KeyPair생성 -> N개 VM 동시 Start)

|-- cloud-control-manager
|   |-- cloud-driver
|   |   |-- drivers: 드라이버 구현체 위치
|   |   |   |-- alibaba
|   |   |   |-- alibaba-plugin
|   |   |   |-- aws
|   |   |   |-- aws-plugin
|   |   |   |-- azure
|   |   |   |-- azure-plugin
|   |   |   |-- cloudit
|   |   |   |-- cloudit-plugin
|   |   |   |-- gcp
|   |   |   |-- gcp-plugin
|   |   |   |-- docker
|   |   |   |-- docker-plugin
|   |   |   |-- openstack
|   |   |   |-- openstack-plugin
|   |   `-- interfaces: 멀티 클라우드 연동 인터페이스(드라어비 공통 인터페이스)
|   |       |-- connect
|   |       |-- resources

|   |-- iid-manager: Integrated ID 관리, IID 구조:{User-defined ID, System-defined ID}

|-- cloud-driver-libs: 드라이버 공유 라이브러리, SSH Key 생성 파일 관리 위치

|-- cloud-info-manager
|   |-- driver-info-manager: 드라이버 정보 관리
|   |-- credential-info-manager: 크리덴셜 정보 관리
|   |-- region-info-manager: 리젼 정보 관리
|   |-- connection-config-info-manager: 

|-- conf: Spider 운영을 위한 설정 정보(spider설정, 메타 정보 관리 설정, 로그 설정)

`-- meta_db: 메타 정보 local FS(nutsdb) 활용시 저장소 위치
    `-- dat
`-- utils
    |-- import-info: Cloud Driver 및 Region 정보 자동 등록 지원 도구
```
