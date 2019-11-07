# cb-spider
CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.

The CB-Spider Mission is to connect all the clouds with a single interface.

***

## [목차]

1. [설치 환경](#설치-환경)
2. [소스 설치](#소스-설치)
3. [실행 준비](#실행-준비)
4. [서버 실행](#서버-실행)
5. [API 규격](#API-규격)
6. [활용 예시](#활용-예시)

***

## [설치 환경]

- 리눅스(검증시험:Ubuntu 18.04)

## [소스 설치]

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

## [실행 준비]
- CB-Spider 실행에 필요한 환경변수 설정
  - `source setup.env` (위치: ./cb-spider)

-	driver shared library 생성 방법(설시 시스템 당 1회 실행, driver source 변경시 실행)
  - `cd cb-spider/cloud-control-manager/cloud-driver/drivers/aws` # AWS Driver 경우
  - `./build_driver_lib.sh` 실행
  -	결과: cb-spider/cloud-driver-libs/aws-driver-v1.0.so 생성

## [서버 실행]
- `cd cb-spider/api-runtime/rest-runtime`
-	`go run *.go`    # 1024 포트 REST API Server 실행됨
-	참고: 메타 정보 초기화 방법
  - cb-spider/meta_db/dat 아래 파일 삭제(ex: 0.dat) 후 서버 재가동
  
## [API 규격]
- 클라우드 인프라 연동 정보 관리: https://documenter.getpostman.com/view/9027676/SVzz4fb4?version=latest
  - 클라우드 드라이버 정보 관리
  - 클라우드 인프라 인증정보 관리
  - 클라우드 인프라 리젼 정보 관리
- 클라우드 인프라 공통 제어 관리(): https://documenter.getpostman.com/view/9027676/SVtSXpzE
  - 이미지 자원 제어
  - 네트워크 자원 제어
  - Security Group 자원 제어
  - Public IP 자원 제어
  - 키페어 자원 제어
  - VM 제어 및 정보 제공
  
## [활용 예시]
- 시험 도구: `cb-spier/api-runtime/rest-runtime/test/aws` (AWS 경우)
- 시험 순서: 연동 정보 추가 => 자원등록 => VM 생성 및 제어 시험
- 시험 방법: 
  - cb-spider/api-runtime/rest-runtime/test/aws 아래 자원 별 디렉토리 시험 스크립트 존재
  -	자원별 create/list/get/delete 관련 shell 스크립트 실행
  - 자원 생성 순서
    - (1) vnetwork, keypair, publicip 및 securitygroup 생성
    - (2) vm 생성 및 제어
    - (3)	delete는 자원 생성 역순
    
