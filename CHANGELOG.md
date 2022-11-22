# v0.7.0 (Cortado, 2022.11.25.)

### API Change

- Add Disk, VM Snapshot, MyImage, AnyCall and PMKS API ([v0.7.0](https://github.com/cloud-barista/cb-spider/wiki/CB-Spider-User-Interface))


### Feature
- Add [Disk(Volume) Mangement](https://github.com/cloud-barista/cb-spider/wiki/Disk-and-Driver-API)
- Add [VM Snapshot/MyImage Management](https://github.com/cloud-barista/cb-spider/wiki/MyImage-and-Driver-API)
- Add [AnyCall API Extension](https://github.com/cloud-barista/cb-spider/wiki/AnyCall-API-Extension-Guide)
- Add [PMKS(Provider Managed Kubernetes) Management](https://github.com/cloud-barista/cb-spider/wiki/Provider-Managed-Kubernetes-and-Driver-API)
- Support [Windows GuestOS](https://github.com/cloud-barista/cb-spider/issues/805)


***

# v0.6.0 (Cafe Latte, 2022.07.08.)

### API Change

- Add AddRules and RemoveRules API to change the rules of SecurityGroup ([v0.5.5](https://github.com/cloud-barista/cb-spider/releases/tag/v0.5.5))


### Feature
- Support Security Group Rules Specs and VM Access Validation Test ([v0.5.5](https://github.com/cloud-barista/cb-spider/releases/tag/v0.5.5))
- [CB-Spider Network Load Balancer Specification and Driver API Definition](https://github.com/cloud-barista/cb-spider/wiki/Network-Load-Balancer-and-Driver-API)
  - Add initial NLB Driver of AWS, GCP, Azure, Alibaba, Tencent, IBM, OPenStack, Cloudit, Mock
  - Support REST Runtime and API of NLB
  - Add AdminWeb Pages of NLB
  - Currently in Alpha Testing
- AdminWeb Enhancements
  - [Add Log windows to show API call status](https://github.com/cloud-barista/cb-spider/wiki/%5BAdminWeb%5D-API-Call-Log-Page-Guide)
  - Download the private key after creating VM KeyPair
- Add new Locking mechanism 'sp-lock' and Concurrent Tests ([v0.5.9](https://github.com/cloud-barista/cb-spider/releases/tag/v0.5.9))
- Enhance the VM Lifecycle Control 


***

# v0.5.0 (Affogato, 2021.12.16.)

### API Change

- Add Register and Unregister API ([v0.4.12](https://github.com/cloud-barista/cb-spider/releases/tag/v0.4.12) [#502](https://github.com/cloud-barista/cb-spider/pull/502))


### Feature
- Add a common Validator and apply it to user's input arguments ([#394 (comment)](https://github.com/cloud-barista/cb-spider/issues/394#issuecomment-963167074))
- Enhance the SSH Key management and insertion method of cb-user into VM ([#480](https://github.com/cloud-barista/cb-spider/issues/480) [#508](https://github.com/cloud-barista/cb-spider/pull/508) [v0.4.14](https://github.com/cloud-barista/cb-spider/releases/tag/v0.4.14))
- Add vm control button for AdminWeb ([#483](https://github.com/cloud-barista/cb-spider/pull/483))
- Enhance IID(Integrated ID) with IID2 ([v0.4.11](https://github.com/cloud-barista/cb-spider/releases/tag/v0.4.11))
- Add `SERVER_ADDRESS` configuration to run in firewall or Kubernetes env. ([v0.4.4](https://github.com/cloud-barista/cb-spider/releases/tag/v0.4.4))
- Update the version info AdminWeb and spctl with `0.5.0`

***

# v0.4.0 (Cafe Mocha, 2021.06.30.)

### API Change

- Add AddSubnet and RemoveSubnet API ([#325](https://github.com/cloud-barista/cb-spider/pull/325) [#326](https://github.com/cloud-barista/cb-spider/pull/326) [#327](https://github.com/cloud-barista/cb-spider/pull/327))
- Add SSHAccessPoint info to VM Info ([#338](https://github.com/cloud-barista/cb-spider/pull/338) )


### Feature
- Add AddSubnet and RemoveSubnet ([#325](https://github.com/cloud-barista/cb-spider/pull/325) [#326](https://github.com/cloud-barista/cb-spider/pull/326) [#327](https://github.com/cloud-barista/cb-spider/pull/327))
- Add SSHAccessPoint info to VM Info ([#338](https://github.com/cloud-barista/cb-spider/pull/338) )
- Support single VM User with cb-user ([#230](https://github.com/cloud-barista/cb-spider/issues/230))
- Enhance the method of Call Log Elapsed time ([#359](https://github.com/cloud-barista/cb-spider/issues/359) [ref](https://github.com/cloud-barista/cb-spider/wiki/StartVM-and-TerminateVM-Main-Flow-of-Cloud-Drivers))
- Change the OpenStack Go SDK for Improvement ([#368](https://github.com/cloud-barista/cb-spider/pull/368) [#370](https://github.com/cloud-barista/cb-spider/pull/370))
  - `github.com/rackspace/gophercloud` => `github.com/gophercloud/gophercloud`
- Update the CSP Go sdk package of cloud drivers ([#328](https://github.com/cloud-barista/cb-spider/issues/328) [ref](https://github.com/cloud-barista/cb-spider/wiki/What-is-the-CSP-SDK-API-Version-of-drivers))
- Shorten the SG delimiter: `-delimiter-` => `-deli-`
- Support Server Status and Endpoint info
  - `./bin/endpoint-info.sh`
- Add SecurityGroup Source filter with CIDR ([#355](https://github.com/cloud-barista/cb-spider/issues/355))
- Integrate tencent driver with current state
- Add REST Basic Auth ([#261](https://github.com/cloud-barista/cb-spider/issues/261) [#412](https://github.com/cloud-barista/cb-spider/pull/412))
- Add cli-dist Make option to build and tar spctl pkg and cli-examples ([c0a902](https://github.com/cloud-barista/cb-spider/commit/c0a902facc468cbf0bf22bdf3182b289484571d2))
- Add Swagger pilot codes ([#418](https://github.com/cloud-barista/cb-spider/pull/418))
- Reflect Dockerfile about cb-user materials
- Update the AdminWeb API Info Page with 0.4.0

***

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

***

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
- Cloud Driver 및 Region 정보 자동 등록 지원 도구 추가 (`utils/import-info/*`)
- Docker Driver 추가(Hetero Multi-IaaS 제어)
- Android 운영 환경을 위한 plugin off mode 추가 ([3938ea0](https://github.com/cloud-barista/cb-spider/commit/3938ea0c70e69664a62eb3cee6611cfbf26ea4ea))  

### Bug Fix

***

# v0.1.0-americano (2019.12.23.)

### Feature
- 멀티 클라우드 연동 기본 기능 제공

