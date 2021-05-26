# latest
### API Change
- 기존 VPC에 Subnet add/delete API 추가 ([#325](https://github.com/cloud-barista/cb-spider/pull/325) [#326](https://github.com/cloud-barista/cb-spider/pull/326) [#327](https://github.com/cloud-barista/cb-spider/pull/327))

### Feature
- 기존 VPC에 Subnet add/delete 기능 추가 ([#325](https://github.com/cloud-barista/cb-spider/pull/325) [#326](https://github.com/cloud-barista/cb-spider/pull/326) [#327](https://github.com/cloud-barista/cb-spider/pull/327))
- Azuer Driver static public ip 생성에서 dynamic mode로 개선 ([#dd881c2](https://github.com/cloud-barista/cb-spider/commit/dd881c2642286b98c5c1eb9ac6ce63de08378c8e))
- VM 기본 사용자 cb-user 계정으로 통일
  - ref) https://github.com/cloud-barista/cb-spider/issues/230
- Cloud Connection Info 중복 등록 오류 해결 ([b69989f](https://github.com/cloud-barista/cb-spider/commit/b69989f05a73a9d42acafae238b8f2e4c21a67f2))
- OpenStack Driver 무한 loop 가능성 해결 ([#368](https://github.com/cloud-barista/cb-spider/pull/368))
- OpenStack Driver 활용 Go SDK 교체 ([#368](https://github.com/cloud-barista/cb-spider/pull/368) [#370](https://github.com/cloud-barista/cb-spider/pull/370))
  - github.com/rackspace/gophercloud => github.com/gophercloud/gophercloud
- Call Log Elapse time 측정 개선
  - ref) https://github.com/cloud-barista/cb-spider/issues/359
  - ref) https://github.com/cloud-barista/cb-spider/wiki/StartVM-and-TerminateVM-Main-Flow-of-Cloud-Drivers
- Update the CSP Go sdk package of cloud drivers
  - ref) https://github.com/cloud-barista/cb-spider/issues/328
  - ref) https://github.com/cloud-barista/cb-spider/wiki/What-is-the-CSP-SDK-API-Version-of-drivers
- SG delimiter 길이 축소: `-delimiter-` => `-deli-`
  - 사용자 SG 이름 입력시 '-deli-' 사용 불가
  - 사용자 SG 이름 '-deli-' 입력 시: http.StatusInternalServerError(500 error) error, msg: "-deli- cannot be used in Security Group name!!"
  - ref) https://github.com/cloud-barista/cb-spider/commit/80b8b2151339d7ff31e2cc58935f365160e496bd
  - ref) https://github.com/cloud-barista/cb-spider/commit/144cc274dc3232b47226025dd6e8a24605784136

# v0.3.0-espresso (2020.12.11.)
### API Change
- 관리용 API listAllXXX(), deleteXXX(force=true), deleteCSPXXX() 추가
  - ref) https://github.com/cloud-barista/cb-spider/issues/228#issuecomment-644536669
- AWS Region 등록 정보에 Zone 정보 추가
  - ref) https://github.com/cloud-barista/cb-spider/issues/248
- Supports gRPC-based GO API for all REST APIs.
- Supports Web-based AdminWeb Tool for easy management.

### Feature
- IID에 등록된 자원 ID와 CSP 자원 ID에 대한 맵핑 관계 손상시 관리 기능 추가
  - ref) https://github.com/cloud-barista/cb-spider/issues/228#issuecomment-644536669
- Supports CLI for Terminal User.
- Add spider's 'AdminWeb Tool' for easy resouce managements and corruected IID management.
- Improved Getting all list of CSP's Image Info.
- Supports HisCall Log Schema & Call-Log Logger for call logging.
- Supports MockDriver.
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

