## Alibaba 드라이버

- ### 클러스터 핸들러 개발 관련 (aka. PMKS)
  - #### 일반사항
    - 타 핸들러와는 다르게 구현 시점 상 최신 버전인 SDK 2.0을 기반으로 개발함.
    [관련 문서](https://api.alibabacloud.com/api-tools/sdk/CS?version=2015-12-15&language=go-tea&tab=primer-doc)
    - 노드 이미지의 설정을 지원하지 않기 때문에 사용자가 설정한 노드 이미지를 활용하지 않음.
  - #### 특이사항
    - 클러스터 생성시 VPC 내 인터넷 NAT 게이트웨이가 없는 경우 클러스터 내에서
    외부 인터넷 접근이 불가하기 때문에 이를 위해 클러스터 생성시 자동으로
    인터넷 NAT 게이트웨이를 생성하고 SNAT 규칙을 설정하도록 지원하며,
    CB-Spider에 의해 활용되는 클러스터가 더 이상 존재하지 않는 경우
    상기 자동 생성된 인터넷 NAT 게이트웨이의 삭제를 진행함.
    [#902](https://github.com/cloud-barista/cb-spider/issues/902),
    [관련 문서](https://api.alibabacloud.com/document/CS/2015-12-15/CreateCluster?spm=api-workbench-intl.api_explorer.0.0.3afc9140EUUgIb)
    - Alibaba API 제약에 따라 노드그룹 관련 API 동작시 오토스케일링이
    활성화된 경우 DesiredNodeSize 값은 반영되지 않음. 
    [관련 문서](https://api.alibabacloud.com/document/CS/2015-12-15/ModifyClusterNodePool?spm=api-workbench-intl.api_explorer.0.0.3afc9140EUUgIb)


