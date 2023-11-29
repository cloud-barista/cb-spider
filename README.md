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

  | Provider | Region/Zone<br>Info | Image<br>Info | VMSpec<br>Info| VPC<br>Subnet | Security<br>Group | VM KeyPair| VM   | NLB/Disk/<br>MyImage | managed-K8S |
  |:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|:-------------:|
  | AWS           | O             | O            | O               | O            | O             | O             | O               | O          |Coming Soon|
  | Azure         | O             | O            | O               | O            | O             | O             | O               | O          | O          |
  | GCP           | O             | O            | O               | O            | O             | O             | O               | O          |Coming Soon|
  | Alibaba       | O             | O            | O               | O            | O             | O             | O               | O          | O          |
  | Tencent       | O             | O            | O               | O            | O             | O             | O               | O          | O          |
  | IBM VPC       | O             | O            | O               | O            | O             | O             | O               | O          |Coming Soon|
  | OpenStack     | O             | O            | O               | O            | O             | O             | O               | O          | - |
  | NCP Classic   | O             | O            | O               | O (Type-I)   | O             | O             | O               | O          | - |
  | NCP VPC       | O             | O            | O               | O            | O             | O             | O               | O          | - |
  | NHN           | O             | O            | O               | O (Type-II)  | O             | O             | O               | O          | - |
  | KT Classic    | O             | O            | O               | O (Type-I)   | O             | O             | O               | Plan       | - |
  | KT VPC        | O             | O            | O               | O (Type-III) | O             | O             | WIP             | Plan       | - |
  

    ※ VPC 특이사항(세부 내용: 각 드라이버 Readme 참고)
        ◉ Type-I: VPC/Subnet Emulation
          - CSP: VPC 개념 제공하지 않음
          - CB-Spider: API 추상화를 위한 단일 VPC/Subnet 생성 제공 (두개 이상 VPC/Subnet 생성 불가)
          - CIDR: 제공하지 않음(설정 무의미)
        ◉ Type-II: default VPC/Subnet 활용
          - CSP: 생성 제공 없이 고정된 default VPC 및 Subnet 1개만 제공
          - CB-Spider: API 추상화를 위한 단일 VPC/Subnet 생성만 제공 (이름 등록 수준, 두개 이상 VPC 생성 불가)
        ◉ Type-III: default VPC 활용(Subnet은 제공)
          - CSP: 생성 제공 없이 고정된 default VPC 1개만 제공
          - CB-Spider: API 추상화를 위한 단일 VPC 생성만 제공 (이름 등록 수준, 두개 이상 VPC 생성 불가)


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
