# cb-spider
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/cloud-barista/cb-spider?label=go.mod)](https://github.com/cloud-barista/cb-spider/blob/master/go.mod)
[![GoDoc](https://godoc.org/github.com/cloud-barista/cb-spider?status.svg)](https://pkg.go.dev/github.com/cloud-barista/cb-spider@master)&nbsp;&nbsp;&nbsp;
[![Release Version](https://img.shields.io/github/v/release/cloud-barista/cb-spider)](https://github.com/cloud-barista/cb-spider/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/cloud-barista/cb-spider/blob/master/LICENSE)

CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.<br>
The CB-Spider Mission is to connect all the clouds with a single interface.


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
  - Build: Go 1.19
  - Container: Docker v19.03

- ##### 시험환경
  - OS: Ubuntu 18.04, Ubuntu 20.04, Ubuntu 22.04, Debian 10.6, macOS Catalina 10.15, Android 8.1 등
  - Build: Go 1.16, Go 1.18, Go 1.19
  - Container: Docker v19.03, Docker v20.10

#### 2. 실행 방법

- ##### 소스 기반 실행: https://github.com/cloud-barista/cb-spider/wiki/Quick-Start-Guide
- ##### 컨테이너 기반 실행: https://github.com/cloud-barista/cb-spider/wiki/Docker-based-Start-Guide
- ##### cb-operator 기반 실행: https://github.com/cloud-barista/cb-operator


#### 3. 제공 자원

  | Provider | Image Info | VMSpec Info| VPC/Subnet | SecurityGroup | VM KeyPair| VM   | NLB/Disk<br>MyImage | K8S |
  |:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|
  | AWS           | O          | O          | O          | O          | O          | O          | O          |Coming Soon|
  | Azure         | O          | O          | O          | O          | O          | O          | O          | O          |
  | GCP           | O          | O          | O          | O          | O          | O          | O          |Coming Soon|
  | Alibaba       | O          | O          | O          | O          | O          | O          | O          | O          |
  | Tencent       | O          | O          | O          | O          | O          | O          | O          | O          |
  | IBM           | O          | O          | O          | O          | O          | O          | O          |Coming Soon|
  | OpenStack     | O          | O          | O          | O          | O          | O          | O          | - |
  | Cloudit       | O          | O          | O(💬)      | O          | O          | O          | O          | - |
  | Docker (PoC)  | O          | -          | -          | -          | -          | O          | -          | - |

    💬 특이사항: 
        - VPC: 단일 VPC 생성 제공 (두개 이상 VPC 생성 불가)
          - VPC CIDR: 제공하지 않음(설정 무의미)
        - Subnet: 단일 VPC에 제한된 CIDR 대역의 Subnet 추가/삭제 가능
          - Subnet CIDR 가능 대역: 10.0.8.0/22, 10.0.12.0/22, 10.0.16.0/22, ... 등
            - 이미 사용 중인 CIDR 요청시 오류 메시지에 사용 가능한 CIDR 목록 반환

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
