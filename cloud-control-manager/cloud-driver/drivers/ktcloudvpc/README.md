### KT Cloud VPC(D1 platform) Connection driver Build 및 CB-Spider에 적용 방법

#### # 연동 Driver 관련 기본적인 사항은 아래 link 참고

   - [Cloud Driver Developer Guide](https://github.com/cloud-barista/cb-spider/wiki/Cloud-Driver-Developer-Guide) 
<p><br>

#### # CB-Spider에 KT Cloud VPC 연동 driver 적용 방법

​	O 위에서와 같이 CB-Spider 코드가 clone된 상태에서 CB-Spider setup 파일 적용
```
$CBSPIDER_ROOT/ source setup.env
```

​	O Dynamic plugin mode로 'CB-Spider' 및 driver build 실행

```
cd $CBSPIDER_ROOT

make dyna
```
   - CB-Spider build 과정이 완료되면, $CBSPIDER_ROOT/bin/ 아래에 binary 파일로 'cb-spider-dyna' 가 생김
   - 각 driver는 $CBSPIDER_ROOT/cloud-driver-libs/ 아래에 binary 파일로 'ktcloudvpc-driver-v1.0.so' 와 같이 생성됨.

<p><br>

​	O CB-Spider server 구동(Dynamic plugin 방식, 1024 포트로 REST API server가 구동됨)

```
cd $CBSPIDER_ROOT/bin

./start-dyna.sh
```

   - CB-Spider server가 구동된 후, KT Cloud VPC driver 등록 과정을 거치고 사용

<p><br>

#### # CB-Spider에 KT Cloud VPC 연동 driver 테스트 방법

​	O 아래의 KT Cloud VPC connection config script 파일에 KT Cloud VPC Credential 정보 기입 후 실행<BR>

```
$CBSPIDER_ROOT/api-runtime/rest-runtime/test/connect-config/15.ktcloudvpc-conn-config.sh
```
<p><br>

#### # CB-Spider REST API 이용 KT Cloud VPC driver 모든 기능 테스트

​	O KT Cloud VPC 각 자원 생성, 자원정보 조회, VM instance Start 및 Terminate 테스트 등 실행 가능

-   Curl command를 이용하여 CB-Spider REST API를 호출하는 테스트 script 이용
```
$CBSPIDER_ROOT/api-runtime/rest-runtime/test/full-test/14.ktcloudvpc-test.sh
$CBSPIDER_ROOT/api-runtime/rest-runtime/test/full-test/ktcloudvpc-full_test.sh
```
<p><br>

#### # KT Cloud VPC 연동 driver 자체 test 파일을 이용한 기능 테스트

​	O CB-Spider 환경 파일 적용
```
$CBSPIDER_ROOT/ source setup.env
```

​	O 아래의 config 파일에 KT Cloud VPC Credential 정보 기입
```
$CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/ktcloudvpc/main/conf/config.yaml
```

​	O 아래의 위치에 있는 ~.sh 파일을 실행해서 KT Cloud VPC driver 각 handler 세부 기능 테스트 
```
$CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/ktcloudvpc/main/
```
<p><br>

#### # KT Cloud VPC 연동 driver를 이용해 생성한 VM에 로그인하는 방법

​	O Cloud-Barista KT Cloud VPC driver를 이용해 생성된 VM에 로그인하는 방법

   - Linux 계열 guest OS의 경우, 사용자가 keyPair 생성시 생성되는 private key를 local에 저장해놓고, 그 private key를 이용해 'cb-user' 계정으로 SSH를 이용해 로그인함.

   - Private key 파일을 이용한 Linux 계열 VM 접속 방법 

```
ssh -i /private_key_경로/private_key_파일명(~~.pem) cb-user@VM의_public_ip
```

<p><br>

#### # KTCloud VPC (KT Cloud D1 플랫폼) driver 사용시 참고 및 주의 사항

​  O VPC, Subnet 관리시 아래의 사항을 참고 

   - KTCloud VPC에서는 단일(default) VPC를 제공하며, 단일 VPC를 제공하지만 driver를 기준으로는 어떤 이름으로든 생성 가능하고, 삭제도 제공함.
     - Default CIDR : 172.25.0.0/12
     - 본 Drvier 기준으로는 이 defualt VPC에 사용자가 지정하는 이름이 붙여짐.

   - KTCloud VPC 서비스에서는 Subnet과 같은 개념으로 'Tier'라는 개념을 사용함.
     - VPC는 상기와 같이 단일 VPC를 제공하지만, subnet은 추가, 삭제 등의 제어가 가능함.(AddSubnet, RemoveSubnet 기능 지원함.)
     - (주의) Default subnet(tier)인 'DMZ'과 'Private' subnet은 삭제 할 수 없으며, 기본적으로 아래의 CIDR을 사용하고 있으니 subnet 생성시 아래의 대역 외의 CIDR을 지정해야함.
       - DMZ subnet : 172.25.0.1/24, Private	subnet : 172.25.1.1/24

     - VPC의 CIDR은 위의 대역이지만, subnet (Tier) 생성시 172.25.X.X 외에도 Custom Tier로서 10.10.X.X, 192.168.X.X 등의 대역을 자유롭게 지정해서 생성할 수 있음.(VM에 그 대역의 private IP가 할당됨.)

   - (참고) Create/Add Subnet시 log에 Error가 발생해도 무시하면됨. KT Cloud D1 Platform의 버그임.

​  O Security Group 생성시 아래의 사항을 참고

   - KTCloud VPC에서는 Security Group(S/G) 개념을 지원하지 않지만, 본 드라이버에서 S/G을 생성한 후 VM 생성시 그 S/G을 적용 가능함.
      - 사용자가 S/G을 정의하면, 드라이버 내부적으로, VM 생성시 public IP가 생성된 후 그 S/G rule들이 Port forwarding rule, Firewall rule로 전환되어 반영됨.
   - KTCloud VPC에서는 outbound rule도 지원하기 때문에 VM에서 outbound 통신이 필요시 반드시 S/G의 outbound 로서 해당 protocol 및 port를 열여줘야함.
