# cb-spider
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/cloud-barista/cb-spider?label=go.mod)](https://github.com/cloud-barista/cb-spider/blob/master/go.mod)
[![GoDoc](https://godoc.org/github.com/cloud-barista/cb-spider?status.svg)](https://pkg.go.dev/github.com/cloud-barista/cb-spider@master)&nbsp;&nbsp;&nbsp;
[![Release Version](https://img.shields.io/github/v/release/cloud-barista/cb-spider)](https://github.com/cloud-barista/cb-spider/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/cloud-barista/cb-spider/blob/master/LICENSE)

CB-Spider is a sub-framework of the Cloud-Barista Multi-Cloud Platform.<br>
CB-Spider offers a unified view and interface for multi-cloud management.


```
[NOTE]
CB-Spider is currently under development. (not v1.0 yet)
We welcome any new suggestions, issues, opinions, and contributors !
Please note that the functionalities of Cloud-Barista are not stable and secure yet.
Be careful if you plan to use the current release in production.
If you have any difficulties in using Cloud-Barista, please let us know.
(Open an issue or Join the Cloud-Barista Slack)
```
***
### ▶ **[Quick Guide](https://github.com/cloud-barista/cb-spider/wiki/Quick-Start-Guide)**
***

#### [목    차]

1. [실행 환경](#1-실행-환경)
2. [실행 방법](#2-실행-방법)
3. [제공 자원](#3-제공-자원)
4. [VM 계정](#4-VM-계정)
5. [활용 방법](#5-활용-방법)
6. [API 규격](#6-API-규격)
7. [특이 사항](#7-특이-사항)
8. [활용 정보](#8-활용-정보)
 
***

#### 1. 실행 환경

- ##### 공식환경
  - OS: Ubuntu 22.04
  - Build: Go 1.21
  - Container: Docker v19.03

- ##### 시험환경
  - OS: , Ubuntu 22.04, Ubuntu 20.04, Ubuntu 18.04, Debian 10.6, macOS Ventura 13.5, macOS Catalina 10.15, Android 8.1 등
  - Build: Go 1.21, Go 1.19, Go 1.18, Go 1.16
  - Container: Docker v19.03, Docker v20.10

#### 2. 실행 방법

- ##### 소스 기반 실행: https://github.com/cloud-barista/cb-spider/wiki/Quick-Start-Guide
- ##### 컨테이너 기반 실행: https://github.com/cloud-barista/cb-spider/wiki/Docker-based-Start-Guide
- ##### cb-operator 기반 실행: https://github.com/cloud-barista/cb-operator


#### 3. 제공 자원

| Provider      | Price<br>Info | Region/Zone<br>Info | Image<br>Info | VMSpec<br>Info | VPC<br>Subnet       | Security<br>Group | VM KeyPair      | VM             | Disk | MyImage | NLB | managed-K8S |
|:-------------:|:-------------:|:-------------------:|:-------------:|:--------------:|:-------------------:|:-----------------:|:---------------:|:--------------:|:----:|:---:|:-------:|:-----------:|
| AWS           | O             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | WIP        |
| Azure         | O<br>(Spec제외)| O                  | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | WIP        |
| GCP           | WIP           | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | WIP        |
| Alibaba       | O             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O           |
| Tencent       | O             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O           |
| IBM VPC       | O<br>(Spec제외)| O                  | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | WIP        |
| OpenStack     | NA             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | ?           |
| NCP Classic   | WIP            | O                   | O             | O              | O<br>(Type-1)       | O<br>(Note-1)     | O               | O              | O    | O   | O       | NA           |
| NCP VPC       | WIP            | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | ?           |
| NHN           | NA             | O                   | O             | O              | O<br>(Type-2)       | O                 | O               | O<br>(Note-2)  | O    | WIP| WIP    | O           |
| KT Classic    | NA             | O                   | O             | O              | O<br>(Type-1)       | O                 | O               | O              | O    | O   | O       | NA          |
| KT VPC        | NA             | O                   | O             | O              | O<br>(Type-3)       | O                 | O               | O              | O    | WIP   | O    | Wait API    |

    ※ WIP: Work In Progress, NA: Not Applicable, Wait API: CSP API 공개 대기, ?: 미정/분석필요
    
    ※ VPC 특이사항(세부 내용: 각 드라이버 Readme 참고)
        ◉ Type-1: VPC/Subnet Emulation
          - CSP: VPC 개념 제공하지 않음
          - CB-Spider: API 추상화를 위한 단일 VPC/Subnet 생성 제공 (두개 이상 VPC/Subnet 생성 불가)
          - CIDR: 제공하지 않음(설정 무의미)
          
        ◉ Type-2: Console에서 사전 생성 후 등록 활용
          - CSP(NHN) IG(Internet Gateway) 제어 API 부재(추후 제공 예정)
          - 사전 작업: Console에서 VPC 사전 생성 및 IG(Internet Gateway) 맵핑 필요(#1109 참고)
          - CB-Spider: Register/UnRegister API 활용
          - 등록 예시
              curl -sX POST http://localhost:1024/spider/regvpc -H 'Content-Type: application/json' -d \
                '{
                  "ConnectionName": "'${CONN_CONFIG}'", 
                  "ReqInfo": { "Name": "'${VPC_NAME}'", "CSPId": "'${VPC_CSPID}'"} 
                }'

        ◉ Type-3: default VPC 활용 (KT VPC)
          - CSP: 생성 제공 없이 고정된 default VPC 1개만 제공
          - CB-Spider: API 추상화를 위한 단일 VPC 생성만 제공 (이름 등록 수준)
            - 두개 이상 VPC 생성 불가, Subnet은 추가/삭제 가능

    ※ Security Group 특이사항(세부 내용: 각 드라이버 Readme 참고)
        ◉ Note-1: Console에서 사전 생성 후 등록 활용
          - CSP: Security Group Create API 부재
          - 사전 작업: Console에서 Security Group 사전 생성
          - CB-Spider: Register/UnRegister API 활용
          - 등록 예시
              curl -sX POST http://localhost:1024/spider/regsecuritygroup -H 'Content-Type: application/json' -d \
               	'{
               		"ConnectionName": "'${CONN_CONFIG}'", 
               		"ReqInfo": { "VPCName": "'${VPC_NAME}'", "Name": "'${SG_NAME}'", "CSPId": "'${SG_CSPID}'"} 
               	}'
          
    ※ VM 특이사항(세부 내용: 각 드라이버 Readme 참고)
        ◉ Note-2: Wdindows VM일 경우 SSH Key 사용한 VM 생성 후 Console에서 Key를 이용하여 PW 확인 필요


#### 4. VM 계정
- Ubuntu, Debian VM User: cb-user
- Windows VM User: Administrator


#### 5. 활용 방법
- [사용자 기능 및 활용 가이드 참고](https://github.com/cloud-barista/cb-spider/wiki/features-and-usages)


#### 6. API 규격

- [인터페이스 규격 및 예시](https://github.com/cloud-barista/cb-spider/wiki/CB-Spider-User-Interface)


#### 7. 특이 사항
- 개발상태: 주요기능 중심 개발추진 중 / 기술개발용 / 상용활용시 보완필요


#### 8. 활용 정보
- 위키: https://github.com/cloud-barista/cb-spider/wiki
