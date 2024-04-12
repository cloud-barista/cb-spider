### NHN Cloud 연동 driver Build 및 CB-Spider에 적용 방법

#### # 연동 Driver 관련 기본적인 사항은 아래 link 참고

   - [Cloud Driver Developer Guide](https://github.com/cloud-barista/cb-spider/wiki/Cloud-Driver-Developer-Guide) 
<p><br>

#### # CB-Spider에 NHN Cloud 연동 driver 적용 방법

​	O CB-Spider 코드가 clone된 상태에서 setup 파일 적용
```
$CBSPIDER_ROOT/ source setup.env
```

​	O Dynamic plugin mode로 CB-Spider build 실행

```
cd $CBSPIDER_ROOT

make dyna

```
   - Build 과정이 완료되면, $CBSPIDER_ROOT/bin/에 binary 파일로 'cb-spider-dyna' 가 생김 

<p><br>

​	O CB-Spider server 구동(Dynamic plugin 방식, 1024 포트로 REST API Server 구동됨)

```
cd bin

./start-dyna.sh
```

   - CB-Spider server가 구동된 후, NHN Cloud 연동 driver 등록 과정을 거치고 사용

<p><br>

#### # CB-Spider에 NHN Cloud 연동 driver 테스트 방법

​	O NHN Cloud connection config 파일에 NHN Cloud Credential 정보(Username, TenantId 등) 기입 후 실행<BR>

```
$CBSPIDER_ROOT/api-runtime/rest-runtime/test/connect-config/ ./13.nhncloud-conn-config.sh
```
<p><br>

#### # CB-Spider REST API 이용 NHN Cloud 연동 driver 모든 기능 테스트

​	O NHN Cloud 각 자원 생성, 자원정보 조회, VM instance Start 및 Terminate 테스트 등

-   Curl command를 이용하는 테스트 script
```
$CBSPIDER_ROOT/api-runtime/rest-runtime/test/full-test/13.nhncloud-test.sh
$CBSPIDER_ROOT/api-runtime/rest-runtime/test/full-test/nhncloud-full_test.sh
```
<p><br>

#### # NHN Cloud driver 자체 test 파일을 이용한 기능 테스트

​	O CB-Spider 환경 파일 적용
```
$CBSPIDER_ROOT/ source setup.env
```

​	O 아래의 config 파일에 NHN Cloud Credential 정보 기입
```
$HOME/go/src/github.com/cloud-barista/nhncloud/nhncloud/main/conf/config.yaml
```

​	O 아래의 위치에 있는 ~.sh 파일을 실행해서 NHN Cloud driver 각 handler 세부 기능 테스트 
```
$HOME/go/src/github.com/cloud-barista/nhncloud/nhncloud/main/
```
<p><br>

#### # NHN Cloud driver로 생성된 VM에 로그인하는 방법

​	O Cloud-Barista NHN Cloud driver를 이용해 생성된 VM에 로그인하는 방법

   - KeyPair 생성시 생성되는 private key를 저장해놓고, 그 private 키를 이용해 'cb-user' 계정으로 SSH를 이용해 로그인하는 방법을 제공함.

   - Private key 파일을 이용한 VM 접속 방법 

```
ssh -i /private_key_파일_경로/private_key_파일명(~~.pem) cb-user@해당_VM의_public_ip
```
<p><br>

#### # NHN Cloud driver 사용시 참고 및 주의 사항
​	O NHN Cloud driver를 CB-Spider에 연동하여 이용할 때, Spider에서 log level을 아래와 같이 설정하면 Spider server 구동 후 NHN Cloud driver의 동작 상태에 대한 상세 log 정보를 확인할 수 있음.
   - CB-Spider log 설정 파일에서 loglevel을 'info'나 'debug' 로 설정
      - CB-Spider log 설정 파일 위치 : ./cb-spider/conf/log_conf.yaml

​	O NHN Cloud infra가 논리적 네트워크 기반이지만, NHN Cloud에서 VPC 및 Subnet 생성/삭제 API는 지원하지 않고 조회 API만 제공하므로 본 driver에서는 아래와 같은 제약 사항이 있음.
   - 현재는 사용자가 원하는 이름으로 VPC와 Subnet을 생성하면, driver 내부적으로 NHN Cloud에서 project별, region별로 기본적으로 생성되어 제공하는 'Default Network' VPC와 'Default Network' Subnet을 사용하게됨.
      - Cloud-Barista를 통해 생성시, VPC와 Subnet의 이름(NameId)은 사용자가 원하는 이름으로 사용 가능하고, driver 내부적으로 VPC와 Subnet의 SystemId는 기본적으로 생성되어 있는 'Default Network' VPC와 'Default Network' Subnet의 SystemId가 적용됨.
      - 'Default Network' VPC CIDR : 192.168.0.0/16, 'Default Network' Subnet CIDR : 192.168.0.0/24
   - (주의) 위와 같은 제약에 대한 사용상의 문제를 최소화하기 위해, 하나의 project 내의 동일한 region에서는 위의 VPC를 이용해 <u>하나의 VPC만 생성하도록 제한</u>하고있음.
   - (참고) 추후 NHN Cloud에서 VPC/Subnet 생성/삭제 API를 제공하면, driver에서 사용자가 원하는 CIDR 대역의 VPC와 Subnet이 생성 가능한 완전한 기능을 지원하도록 보완할 예정임.

   - VPC/Subnet을 API를 이용해서(본 드라이버를 이용해서) 생성할때, NHN Cloud region별로 다른 API endpoint를 사용하기 때문에 config에 설정된 region에 VPC/Subnet이 생성됨.(Region 단위로 구분되어 생성됨.)
   - 그 특정 region에 생성된 VPC/Subnet은 그 region에 속하는 zone에 공유해서 사용함.(그 region 내 모든 zone에서 조회되고 사용 가능함.)

​	O NHN Cloud driver를 통해 VM instance 생성시, VMSpec type별로 지원하는 root disk type와 volume 크기가 다름.(아래는 CB-Spider 기준으로 지정 가능한 option임.)
   - u2.~~~ type의 VMSpec은 RootDiskType으로 default인 'General_HDD'만을 지원하고, RootDiskSize는 VMSpec별로 지정된 size를 지원함.
      - 가용 RootDiskType : ""(Blank, Not specified), 'default', 'General_HDD', 'TYPE1'
      - 가용 RootDiskSize : VMSpec 조회시 spec별 지원 disk size 확인
   - u2.~~~ type 외의 VMSpec을 지정할 경우에는 RootDiskType과 RootDiskSize를 지정해야함.
      - 가용 RootDiskType : ""(Blank, Not specified), 'default', 'General_HDD', 'General_SSD', 'TYPE1', 'TYPE2'
      - 가용 RootDiskSize : 20 ~ 1000(GB)
      - 이 외의 잘못된 RootDiskType이나 RootDiskSize를 지정하면 오류 메시지로 안내함.
   - u2.~~~ type 외의 VMSpec 일때 RootDiskType과 RootDiskSize를 지정하지 않아도 정상적으로 VM이 생성되는데, 지정하지않을 경우 default 값으로써 RootDiskType은 'General_HDD' type이 지정되고, RootDiskSize로 20G가 지정됨.
   - VMSpec type별로 가용한 RootDiskType/RootDiskSize 설정 option 및 그에 대한 disk 생성 결과 정리
      - https://github.com/cloud-barista/cb-spider/issues/598#issuecomment-1097610395
      
   - (참고) NHN Cloud driver를 통해 VMSpec 조회시, u2 type의 VMSpec은 LocalDiskSize가 나타나고, u2 type을 제외한 VMSpec type은 LocalDiskSize가 '0'으로 나타남.

   - <B>반드시 참고해야할 사항(NHN Cloud에서 u2 type 선택시 명시되는 주의 사항)</B>
      - NHN Cloud에서 u2 type의 VM instance는 root volume을 local disk로 제공함.
      - 따라서, 하드웨어 장애 시 사용자 instance는 생성 당시의 상태로 제공될 수 있으며, 이때 데이터의 복구는 불가능하니 별도로 백업하거나 이중화할 것을 권장함.
      - U2 instance는 NHN Cloud 서비스 이용약관 제34조에 따른 손해배상 대상에서 제외됨.

​	O NHN Cloud driver를 통해 Security Group을 생성하거나 Security Rule을 추가시 inbound/outbound에 대해 IPProtocol : "ALL", FromPort: "-1", ToPort: "-1"을 입력하면, 모든 Protocol에 대해 모든 영역 port가 open됨.
   - Security Rule을 제거시에 동일하게 설정하면, 모든 Protocol에 대해 모든 영역 port가 open된 Rule이 제거됨.

  ​O NHN Cloud driver를 통해 MyImage 생성시, 다음 사항을 참고
   - Snapshot 대상의 VM에 RootDisk 외에 attach된 disk가 있을 경우, attach된 disk는 제외하고 RootDisk만으로 MyImage가 생성됨.

  ​O NHN Cloud driver를 통해 MyImage를 이용한 VM 생성시, 다음 사항을 참고
   - u2.~~~ type의 VMSpec으로 생성되었던 VM을 기준으로 생성된 MyImage는 그 VM의 local disk가 MyImage로 생성됨.
      - (주의) 이와 같이, u2 type의 VMSpec으로 생성된 VM의 MyImage를 이용해 신규 VM을 생성할 경우, 그 신규 VM도 u2 type의 VMSpec을 이용해야함.

  ​O NHN Cloud에서 Windows VM instance 생성을 위한 제약 조건으로 아래의 사항을 공지함.
   - VM 생성시, Linux 계열 VM과 같이 KyePair를 지정해야하며, Windows VM 생성 완료 후 콘솔에서 그 KeyPair를 이용해서 default 미밀번호를 확인할 수 있음.
   - RAM이 최소 2GB 이상인 VMSpec 사용해야함.
   - 50GB 이상의 루트 블록 스토리지 필요
   - U2 type의 VMSpec은 Windows image를 사용할 수 없음.

​	O 본 드라이버를 통해 미국 region infra는 사용 불가
   - 현재 한국 : 2개 region X 2개 zone, 일본 : 1개 region X 2개 zone 지원
   - NHN Cloud에서 미국 region은 API endpoint를 제공하지 않으므로 미국 region은 console을 통해서만 사용 가능

#### # 클러스터 핸들러 개발 관련 (aka. PMKS)
   - #### 일반사항
      - [cloud-barista/nhncloud-sdk-go](https://github.com/cloud-barista/nhncloud-sdk-go)를 기반으로 개발함
      - 한국(판교) 리전과 한국(평촌) 리전만 NKS(NHN Kubernetes Service) 제공
      - 노드 이미지 정보는 리전별로 지원하는 베이스 이미지 UUID로 설정해야 함 ([link](https://docs.nhncloud.com/ko/Container/NKS/ko/public-api/#uuid_3))
      - 클러스터 생성시 1개의 노드그룹 설정만 지원하며, 클러스터 생성 이후 노드 그룹 추가를 지원함
   - #### 특이사항
      - 노드그룹에 포함된 노드의 Security Group을 사용자가 요청한 값으로 설정하지 않고 자체적으로 생성한 Security Group으로 설정되며, NetworkInfo.SecurityGroupIIDs.NameId를 '#'+SystemId로 리턴함 ([#1065](https://github.com/cloud-barista/cb-spider/issues/1065))
   - #### CSP 제약 사항
      - 첫번째 노드그룹 이름은 default-worker로 고정 생성됨 ([#867](https://github.com/cloud-barista/cb-spider/issues/867))
      - 프로젝트당 클러스터 생성 개수 제한 존재 (기본 3개)
      - 클러스터 업그레이드시 마스터/워커 노드그룹 단위 업그레이드만 가능하며 동시 업그레이드 미지원 ([#1129](https://github.com/cloud-barista/cb-spider/issues/1129))
      - 현재 판교, 평촌만 NKS 지원 ([link](https://docs.nhncloud.com/ko/Container/NKS/ko/public-api/))
      - 인터넷 게이트웨이가 연결된 VPC에서만 Public K8s 엔드포인트 지정 가능
         - 인터넷 게이트웨어 제어 API를 미제공하므로 사전 생성 및 연결 절차 수행 필요 ([#1109](https://github.com/cloud-barista/cb-spider/issues/1109))
      - 클러스터 업그레이드시 마스터/워커 노드그룹 단위 업그레이드만 가능하며 동시 업그레이드 미지원 ([#1129](https://github.com/cloud-barista/cb-spider/issues/1129))
      - 노드그룹의 노드에 설정되는 Security Group을 자체적으로 생성하여 적용함 ([#1065](https://github.com/cloud-barista/cb-spider/issues/1065))
