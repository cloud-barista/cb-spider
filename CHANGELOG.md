# v0.3.0-espresso (2020.12.11.)
### API Change
- 관리용 API listAllXXX(), deleteXXX(force=true), deleteCSPXXX() 추가
  - ref) https://github.com/cloud-barista/cb-spider/issues/228#issuecomment-644536669
- AWS Region 등록 정보에 Zone 정보 추가
  - ref) https://github.com/cloud-barista/cb-spider/issues/248
- Supports adding and deleting subnets in an existing VPC 
  - ref) https://github.com/cloud-barista/cb-spider/issues/277
- Supports gRPC-based GO API for all REST APIs.
- Supports Web-based AdminWeb Tool for easy management.

### Feature
- IID에 등록된 자원 ID와 CSP 자원 ID에 대한 맵핑 관계 손상시 관리 기능 추가
  - ref) https://github.com/cloud-barista/cb-spider/issues/228#issuecomment-644536669
- Add spider's 'AdminWeb Tool' for easy resouce managements and corruected IID management.
- Improved Getting all list of CSP's Image Info.
- Improved VPC management through adding/deleting subnets.
- Support HisCall Log Schema & Call-Log Logger for call logging.
- Support MockDriver.
  - ref) https://github.com/cloud-barista/cb-spider/issues/292
- Add Experimental Features about distributed Spiders PoC(MEERKAT Project)

# v0.2.0-cappuccino (2020.06.01.)
### API Change
- CloudO 목록 제공 API 추가
- VPC/Subnet API 추가 ([#9](https://github.com/cloud-barista/cb-spider/pull/9) [#226](https://github.com/cloud-barista/cb-spider/pull/226))
- VMSpec API 추가 ([#151](https://github.com/cloud-barista/cb-spider/pull/151) [#223](https://github.com/cloud-barista/cb-spider/pull/223))
- VNic API 삭제
- PublicIP API 삭제

### Feature
- 통합ID IID Manager 추가 ([#163](https://github.com/cloud-barista/cb-spider/pull/163) [#194](https://github.com/cloud-barista/cb-spider/pull/194))  
- VPC/Subnet 기능 추가  ([#9](https://github.com/cloud-barista/cb-spider/pull/9) [#226](https://github.com/cloud-barista/cb-spider/pull/226)) 
- VNic, PublicIP 자동 관리 기능으로 개선
- Cloud Driver 및 Region 정보 자동 등록 지원 도구 추가 utils/import-info/*
- Docker Driver 추가(Hetero Multi-IaaS 제어)
- Android 운영 환경을 위한 plugin off mode 추가 ([3938ea0](https://github.com/cloud-barista/cb-spider/commit/3938ea0c70e69664a62eb3cee6611cfbf26ea4ea))  

### Bug Fix

# v0.1.0-americano (2019.12.23.)

### Feature
- 멀티 클라우드 연동 기본 기능 제공

