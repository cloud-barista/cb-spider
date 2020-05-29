#### API Change
- CloudO 목록 제공 API 추가
- VPC/Subnet API 추가
  - 참고: https://github.com/cloud-barista/cb-spider/pulls?q=is%3Apr+is%3Aclosed+vpc
- VMSpec API 추가
  - 참고: https://github.com/cloud-barista/cb-spider/pulls?q=is%3Apr+is%3Aclosed+vmspec
- VNic API 삭제
- PublicIP API 삭제

#### Features
- 통합ID IID Manager 추가
  - 참고: https://github.com/cloud-barista/cb-spider/pulls?q=is%3Apr+is%3Aclosed+iid
- VPC/Subnet 기능 추가
  - 참고: https://github.com/cloud-barista/cb-spider/pulls?q=is%3Apr+is%3Aclosed+vpc
- VNic, PublicIP 자동 관리 기능으로 개선
- Cloud Driver 및 Region 정보 자동 등록 지원 도구 추가 utils/import-info/*
- Docker Driver 추가(Hetero Multi-IaaS 제어)
- Android 운영 환경을 위한 plugin off mode 추가
  - 참고: https://github.com/cloud-barista/cb-spider/commit/3938ea0c70e69664a62eb3cee6611cfbf26ea4ea
