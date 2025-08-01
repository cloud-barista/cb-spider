# CB-Spider : "One-Code, Multi-Cloud"
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/cloud-barista/cb-spider?label=go.mod)](https://github.com/cloud-barista/cb-spider/blob/master/go.mod)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/cloud-barista/cb-spider/blob/master/LICENSE)&nbsp;&nbsp;&nbsp;
[![Release Version](https://img.shields.io/github/v/release/cloud-barista/cb-spider)](https://github.com/cloud-barista/cb-spider/releases)
[![Latest Docs](https://img.shields.io/badge/docs-latest-green)](https://github.com/cloud-barista/cb-spider/wiki)
[![Swagger API Docs](https://img.shields.io/badge/docs-Swagger_API-blue)](https://cloud-barista.github.io/api/?url=https://raw.githubusercontent.com/cloud-barista/cb-spider/refs/heads/master/api/swagger.yaml)


- CB-Spider is a sub-framework of the Cloud-Barista Multi-Cloud Platform.<br>
- CB-Spider offers a unified view and interface for multi-cloud management.

<p align="center">
  <img width="850" alt="image" src="https://github.com/user-attachments/assets/c1e5328b-151d-4b24-ad62-947e8bfcbbcf">
</p>

```
[NOTE]
CB-Spider is currently under development and has not yet reached version 1.0.
We welcome suggestions, issues, feedback, and contributions!
Please be aware that the functionalities of Cloud-Barista are not yet stable or secure.
Exercise caution if you plan to use the current release in a production environment.
If you encounter any difficulties while using Cloud-Barista, please let us know.
(You can open an issue or join the Cloud-Barista Slack community.)
```
***
### ▶ **[Quick Guide](https://github.com/cloud-barista/cb-spider/wiki/Quick-Start-Guide)**
***

#### [목    차]

1. [권장 환경](#1-권장-환경)
2. [실행 방법](#2-실행-방법)
3. [제공 자원](#3-제공-자원)
4. [VM 계정](#4-VM-계정)
5. [활용 방법](#5-활용-방법)
6. [API 규격](#6-API-규격)
7. [특이 사항](#7-특이-사항)
8. [활용 정보](#8-활용-정보)
 
***

#### 1. 권장 환경

- OS: Ubuntu 22.04
- Build: Go 1.23, Swag v1.16.3
- Container: Docker v19.03


#### 2. 실행 방법

- ##### 소스 기반 실행: https://github.com/cloud-barista/cb-spider/wiki/Quick-Start-Guide
- ##### 컨테이너 기반 실행: https://github.com/cloud-barista/cb-spider/wiki/Docker-based-Start-Guide


#### 3. 제공 자원
- #### ※ 참고: [Tagging Guide](https://github.com/cloud-barista/cb-spider/wiki/Tag-and-Cloud-Driver-API)


| Provider      | Price<br>Info | Region/Zone<br>Info | Image<br>Info | VMSpec<br>Info | VPC<br>Subnet       | Security<br>Group | VM KeyPair      | VM             | Disk | MyImage | NLB | managed-K8S | Object<br> Storage |
|:-------------:|:-------------:|:-------------------:|:-------------:|:--------------:|:-------------------:|:-----------------:|:---------------:|:--------------:|:----:|:---:|:-------:|:-----------:|:-----------:|
| AWS           | O             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O        | O        |
| Azure         | O             | O                  | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O        | WIP        |
| GCP           | O             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O        | O        |
| Alibaba       | O             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O           | O        |
| Tencent       | O             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O           | WIP        |
| IBM VPC       | O             | O                  | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | WIP        | O        |
| OpenStack     | NA             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | ?           | WIP        |
| NCP VPC       | O            | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | WIP           | O        |
| NHN           | NA             | O                   | O             | O              | O<br>(Type1)       | O                 | O               | O<br>(Note2)   | O    | O    | O     | O           | O        |
| KT VPC        | NA             | O                   | O             | O              | O<br>(Type2)       | O                 | O               | O              | O    | O   | O<br>(Note3)| Wait API  | O        |
| KT Classic    | NA             | O                   | O             | O              | O<br>(Type3)       | O                 | O               | O              | O    | O   | O       | NA          | -        |
| NCP Classic<br> (25.9.18.부터 VM 생성 불가)   | -            | O                   | O             | O              | O<br>(Type3)       | O<br>(Note1)     | O               | O              | O    | O   | O       | NA           | -        |

    ※ WIP: Work In Progress, NA: Not Applicable, Wait API: CSP API 공개 대기, ?: 미정/분석필요, -: 연동 제외 Classic 자원
    
    ※ VPC 특이사항(세부 내용: 각 드라이버 Readme 참고)
        ◉ Type1: Console에서 사전 생성 후 등록 활용
          - CSP(NHN) IG(Internet Gateway) 제어 API 부재(추후 제공 예정)
          - 사전 작업: Console에서 VPC 사전 생성 및 IG(Internet Gateway) 맵핑 필요(#1109 참고)
          - CB-Spider: Register/UnRegister API 활용
          - 등록 예시
              curl -sX POST http://localhost:1024/spider/regvpc -H 'Content-Type: application/json' -d \
                '{
                  "ConnectionName": "'${CONN_CONFIG}'", 
                  "ReqInfo": { "Name": "'${VPC_NAME}'", "CSPId": "'${VPC_CSPID}'"} 
                }'
          
        ◉ Type2: default VPC 활용 (KT VPC)
          - CSP: 생성 제공 없이 고정된 default VPC 1개만 제공
          - CB-Spider: API 추상화를 위한 단일 VPC 생성만 제공 (이름 등록 수준)
            - 두개 이상 VPC 생성 불가, Subnet은 추가/삭제 가능

        ◉ Type3: VPC/Subnet Emulation
          - CSP: VPC 개념 제공하지 않음
          - CB-Spider: API 추상화를 위한 단일 VPC/Subnet 생성 제공 (두개 이상 VPC/Subnet 생성 불가)
          - CIDR: 제공하지 않음(설정 무의미)


    ※ Security Group 특이사항(세부 내용: 각 드라이버 Readme 참고)
        ◉ Note1: Console에서 사전 생성 후 등록 활용
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
        ◉ Note2: Wdindows VM일 경우 SSH Key 사용한 VM 생성 후 Console에서 Key를 이용하여 PW 확인 필요

    ※ NLB 특이사항(세부 내용: 각 드라이버 Readme 참고)
        ◉ Note3: NLB에 등록할 VM은 NLB와 동일 Subnet에 존재해야 함


#### 4. VM 계정
- Ubuntu, Debian VM User: cb-user
- Windows VM User: Administrator


#### 5. 활용 방법
- [사용자 기능 및 활용 가이드 참고](https://github.com/cloud-barista/cb-spider/wiki/features-and-usages)


#### 6. API 규격
 - [Swagger Documentations](https://github.com/cloud-barista/cb-spider/tree/master/api)
 - [Swagger Guide](https://github.com/cloud-barista/cb-spider/wiki/Swagger-Guide)
 


#### 7. 특이 사항
- 개발상태: 주요기능 중심 개발추진 중 / 기술개발용 / 상용활용시 보완필요


#### 8. 활용 정보
- 위키: https://github.com/cloud-barista/cb-spider/wiki
