# cb-spider
CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.

The CB-Spider Mission is to connect all the clouds with a single interface.

***

## [목    차]

1. [실행 환경](#실행-환경)
2. [실행 방법](#실행-방법)
3. [API 규격](#API-규격)
4. [활용 예시](#활용-예시)
5. [특이 사항](#특이-사항)

***

## [실행 환경]

- 리눅스(검증시험:Ubuntu 18.04, Raspbian GNU/Linux 10)

## [실행 방법]

### (1) 컨테이너 기반 실행
- CB-Spider 이미지 확인(https://hub.docker.com/r/cloudbaristahub/cb-spider/tags)
- CB-Spider 컨테이너 실행
```
# docker run -p 1024:1024 \
-v /root/go/src/github.com/cloud-barista/cb-spider/meta_db:/root/go/src/github.com/cloud-barista/cb-spider/meta_db \
--name cb-spider \
cloudbaristahub/cb-spider:v0.1-yyyymmdd
```

### (2) 소스 기반 실행

#### (a) 소스 설치

- Git 설치
- Go 설치(1.12 이상)  

- Cloud-Barista alliance 설치 (CB-Log)
  - `go get -u -v github.com/cloud-barista/cb-log`
  - https://github.com/cloud-barista/cb-log README를 참고하여 설치 및 설정
  
- Cloud-Barista alliance 설치 (CB-Store)
  - `go get -u -v github.com/cloud-barista/cb-store`
  - https://github.com/cloud-barista/cb-store README를 참고하여 설치 및 설정

- CB-Spider 설치
    - `go get -u -v github.com/cloud-barista/cb-spider`    

- 설치 오류시 참고
    - 오류 메시지: "panic: /debug/requests is already registered. You may have two independent copies of golang.org/x/net/trace in your binary, trying to maintain separate state. This may involve a vendored copy of golang.org/x/net/trace.”
    
      - 해결방법: $ rm -rf $GOPATH/src/go.etcd.io/etcd/vendor/golang.org/x/net/trace
      
    - 오류 메시지: "gosrc/src/go.etcd.io/etcd/vendor/google.golang.org/grpc/clientconn.go:49:2: use of internal package google.golang.org/grpc/internal/resolver/dns not allowed"
    
      - 해결방법: $ rm -rf $GOPATH/gosrc/src/go.etcd.io/etcd/vendor/google.golang.org/grpc
      
#### (b) 실행 준비
- CB-Spider 실행에 필요한 환경변수 설정
  - `source setup.env` (위치: ./cb-spider)

-	driver shared library 생성 방법(설치 시스템 당 1회 실행, driver source 변경시 실행)
  - `./build_all_driver_lib.sh` 실행
  -	결과: cb-spider/cloud-driver-libs/xxx-driver-v1.0.so 생성
  - 참고: 특정 CSP driver만 build하는 방법
    - `cd cb-spider/cloud-control-manager/cloud-driver/drivers/aws` # AWS Driver 경우
    - `build_driver_lib.sh` 실행

#### (c) 서버 실행
- `cd cb-spider/api-runtime/rest-runtime`
-	`go run *.go`    # 1024 포트 REST API Server 실행됨
-	참고: 메타 정보 초기화 방법
    - cb-spider/meta_db/dat 아래 파일 삭제(ex: 0.dat) 후 서버 재가동

### (3) Cloud-Barista 시스템 통합 실행 참고(Docker-Compose 기반)
```
# git clone https://github.com/jihoon-seo/cb-deployer.git
# cd cb-deployer
# docker-compose up
```

## [API 규격]
- 클라우드 인프라 연동 정보 관리: https://documenter.getpostman.com/view/9027676/SVzz4fb4?version=latest
  - 클라우드 드라이버 정보 관리
  - 클라우드 인프라 인증정보 관리
  - 클라우드 인프라 리젼 정보 관리
- 클라우드 인프라 공통 제어 관리: https://documenter.getpostman.com/view/9027676/SVtSXpzE
  - 이미지 자원 제어
  - 네트워크 자원 제어
  - Security Group 자원 제어
  - Public IP 자원 제어
  - 키페어 자원 제어
  - VM 제어 및 정보 제공
  
## [활용 예시]
- 시험 도구: `cb-spier/api-runtime/rest-runtime/test/[aws|azure|gcp|openstack|cloudit]` (AWS 경우:aws)
- 시험 순서: 연동 정보 추가 => 자원등록 => VM 생성 및 제어 시험
- 시험 방법: 
  - (연동정보관리) cb-spider/api-runtime/rest-runtime/test/aws/cim-insert-test.sh 참고(Credential 정보 수정 후 실행)
  - (자원관리) cb-spider/api-runtime/rest-runtime/test/aws 아래 자원 별 디렉토리 시험 스크립트 존재
  -	(자원관리) 자원별 create/list/get/delete 관련 shell 스크립트 실행
  - (자원관리) 자원 생성 순서
    - (1) vnetwork, keypair, publicip 및 securitygroup 생성
    - (2) vm 생성 및 제어
    - (3)	삭제는 자원 생성 역순
    
## [특이 사항]
- 개발상태: 초기 기능 중심 개발추진 중 / 기술개발용 / 상용활용시 보완필요
- Alibaba: 통합 시험 전 상태
- Key관리: CSP가 제공하지 않는 경우 Key 자체 생성 및 Key 파일 내부 관리
  - 관리위치: cb-spider/cloud-driver-libs/.ssh-CSPName/* (임시방법)
  - 공유서버에서 운영시 보안 이슈 존재


