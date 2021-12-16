# cb-spider
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/cloud-barista/cb-spider?label=go.mod)](https://github.com/cloud-barista/cb-spider/blob/master/go.mod)
[![GoDoc](https://godoc.org/github.com/cloud-barista/cb-spider?status.svg)](https://pkg.go.dev/github.com/cloud-barista/cb-spider@master)&nbsp;&nbsp;&nbsp;
[![Release Version](https://img.shields.io/github/v/release/cloud-barista/cb-spider)](https://github.com/cloud-barista/cb-spider/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/cloud-barista/cb-spider/blob/master/LICENSE)

CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.<br>
The CB-Spider Mission is to connect all the clouds with a single interface.


```
[NOTE]
CB-Spider is currently under development. (The latest version is v0.5.0 (Affogato))
So, we do not recommend using the current release in production.
Please note that the functionalities of CB-Spider are not stable and secure yet.
If you have any difficulties in using CB-Spider, please let us know.
(Open an issue or Join the cloud-barista Slack)
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
  - OS: Ubuntu 18.04
  - Container: Docker 19.03
  - Build: Go 1.16
- ##### 시험환경
  - OS: Ubuntu 18.04, Ubuntu 20.04, Debian 10.6, macOS Catalina 10.15, Android 8.1 등
  - Container: latest Docker
  - Build: latest Go


#### 2. 실행 방법

- ##### 소스 기반 실행: https://github.com/cloud-barista/cb-spider/wiki/Quick-Start-Guide
- ##### 컨테이너 기반 실행: https://github.com/cloud-barista/cb-spider/wiki/Docker-based-Start-Guide
- ##### cb-operator 기반 실행: https://github.com/cloud-barista/cb-operator


#### 3. 제공 자원

  | Provider(CloudOS) | VM Image List/Get | VM Spec List/Get| VPC/Subnet | Security Group | VM KeyPair| VM   |
  |:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|
  | AWS           | O          | O          | O          | O          | O          | O          |
  | Azure         | O          | O          | O          | O          | O          | O          |
  | GCP           | O          | O          | O          | O          | O          | O          |
  | Alibaba       | O          | O          | O          | O          | O          | O          |
  | Tencent       | O          | O          | O          | O          | O          | O          |
  | IBM           | O          | O          | O          | O          | O          | O          |
  | OpenStack     | O          | O          | O          | O          | O          | O          |
  | Cloudit       | O          | O          | O(💬)          | O          | O          | O          |
  | Docker        | O          | -          | -          | -          | -          | O          |

    💬 특이사항: 
        - VPC: 단일 VPC 생성 제공 (두개 이상 VPC 생성 요청시 동작을 보장할 수 없음)
        - Subnet: 단일 VPC에 Subnet 추가/삭제 가능
        - VPC 및 Subnet CIDR: 사용자의 설정값과 무관하게, CSP 내부에서 유휴 CIDR 할당 후 반납
    

#### 4. VM 계정
- CB Spider VM User: cb-user


#### 5. 활용 방법
- [사용자 기능 및 활용 가이드 참고](https://github.com/cloud-barista/cb-spider/wiki/features-and-usages)


#### 6. API 규격

- [인터페이스 규격 및 예시](https://github.com/cloud-barista/cb-spider/wiki/CB-Spider-User-Interface)


#### 7. 특이 사항
- 개발상태: 초기 주요기능 중심 개발추진 중 / 기술개발용 / 상용활용시 보완필요


#### 8. 활용 정보
- 위키: https://github.com/cloud-barista/cb-spider/wiki
