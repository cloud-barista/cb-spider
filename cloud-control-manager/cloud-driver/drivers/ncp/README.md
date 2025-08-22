### NCP (Naver Cloud Platform) VPC Connection driver Build 및 CB-Spider에 적용 방법

#### # 연동 Driver 관련 기본적인 사항은 아래 link 참고

   - [Cloud Driver Developer Guide](https://github.com/cloud-barista/cb-spider/wiki/Cloud-Driver-Developer-Guide) 
<p><br>

#### # CB-Spider에 NCP VPC 연동 driver 적용 방법

​	O 위에서와 같이 CB-Spider 코드가 clone된 상태에서 CB-Spider setup 파일 적용
```
$CBSPIDER_ROOT/ source setup.env
```

​	O Dynamic plugin mode로 'CB-Spider' build 실행

```
cd $CBSPIDER_ROOT

make dyna

```
   - CB-Spider build 과정이 완료되면, $CBSPIDER_ROOT/bin/ 아래에 binary 파일로 'cb-spider-dyna' 가 생김

<p><br>

​	O CB-Spider server 구동(Dynamic plugin 방식, 1024 포트로 REST API server가 구동됨)

```
cd $CBSPIDER_ROOT/bin

./start-dyna.sh
```

   - CB-Spider server가 구동된 후, NCP VPC driver 등록 과정을 거치고 사용

<p><br>

#### # CB-Spider에 NCP VPC 연동 driver 테스트 방법

​	O NCP VPC Credential 정보(API 인증키) 발급 방법<BR>
 - [네이버클라우드 포털](https://www.ncloud.com) > 마이페이지 > 계정관리 > '인증키 관리' 메뉴로 이동<br>
   > [신규 API 인증키 생성] 버튼을 클릭하면 인증키가 생성됨.
 - 네이버클라우드 Credential 정보(API 인증키) 생성 메뉴 바로 가기 : 네이버클라우드 포털 > [인증키 관리](https://www.ncloud.com/mypage/manage/authkey) 메뉴

   (참고) 네이버클라우드 API 인증키는 계정당 2개까지 생성할 수 있음.

​	O 아래의 NCP VPC connection config script 파일에 NCP VPC Credential 정보(API 인증키) 기입 후 실행<BR>

```
$CBSPIDER_ROOT/api-runtime/rest-runtime/test/connect-config/ ./14.ncp-conn-config.sh
```
<p><br>

#### # CB-Spider REST API 이용 NCP VPC driver 모든 기능 테스트

​	O NCP VPC 각 자원 생성, 자원정보 조회, VM instance Start 및 Terminate 테스트 등 실행 가능

-   Curl command를 이용하여 CB-Spider REST API를 호출하는 테스트 script 이용
```
$CBSPIDER_ROOT/api-runtime/rest-runtime/test/full-test/14.ncp-test.sh
$CBSPIDER_ROOT/api-runtime/rest-runtime/test/full-test/ncp-full_test.sh
```
<p><br>

#### # NCP VPC 연동 driver 자체 test 파일을 이용한 기능 테스트

​	O CB-Spider 환경 파일 적용
```
$CBSPIDER_ROOT/ source setup.env
```

​	O 아래의 config 파일에 NCP VPC Credential 정보 기입
```
$CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/ncp/main/config/config.yaml
```

​	O 아래의 위치에 있는 ~.sh 파일을 실행해서 NCP VPC driver 각 handler 세부 기능 테스트 
```
$CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/ncp/main/
```
<p><br>

#### # NCP VPC 연동 driver로 생성된 VM에 로그인하는 방법

​	O Cloud-Barista NCP VPC driver를 이용해 생성된 VM에 로그인하는 방법

   - 사용자가 keyPair 생성시 생성되는 private key를 local에 저장해놓고, 그 private key를 이용해 'cb-user' 계정으로 SSH를 이용해 로그인하는 방법을 제공함.

   - Private key 파일을 이용한 VM 접속 방법 

```
ssh -i /private_key_경로/private_key_파일명(~~.pem) cb-user@VM의_public_ip
```

​	(참고) NCP VPC CSP에서 제공하는 VM 로그인 방법

   - NCP VPC console에서 사용자가 생성한 private key를 매핑하여 VM 생성 후, 그 private key를 이용해 해당 VM의 root 비밀번호를 알아내어 SSH를 이용해 root 계정과 비밀번호로 로그인함.

<p><br>

#### # NCP VPC Cloud driver 사용시 참고 및 주의 사항
  O NCP VPC 버전 driver이 지원하는 region 및 zone은 아래의 파일을 참고
```
  ./ncp/ncp/main/config/config.yaml.sample
  ./cb-spider/api-runtime/rest-runtime/test/connect-config/14.ncp-conn-config.sh
```

  ​O NCP VPC driver를 이용해 VM 생성시 VPC, Subnet, Network Interface에 대해 다음 사항을 참고
   - VPC의 private IP address 범위는, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 대역 내에서, 실제 생성시 /16~/28 범위여야함.
      - 참고 : NCP의 VPC/Subnet 가용 범위
         - https://guide.ncloud-docs.com/docs/vpc-spec-vpc
         - https://guide.ncloud-docs.com/docs/vpc-glossary-vpc
   - Subnet의 private IP address 범위는, VPC의 private address 범위 이하로만 지정이 가능하며, 생성 이후에는 Network ACL만 변경이 가능함.
   - 본 driver에서 VM의 Network Interface는 VPC 생성시 자동으로 생성되는 Default Network Interface를 사용하도록 개발되어있음.

  O Security Group에 inbound rule이나 outbound rule을 생성 혹은 추가할때 다음 사항을 참고
   - 다음과 같이 지정하면 모든 protocol에 대해, 모든 port를 open하는 rule이 생성/추가됨(CIDR : '0.0.0.0/0'로 기본 설정됨.)
     - Rule 생성/추가시 사용자 지정 값 : IPProtocol : 'All', FromPort : '-1', ToPort : '-1'

  ​O NCP VPC driver를 이용해 VM 생성시 ImageType에 대해 다음 사항을 참고
   - VM 생성시 ImageType을 'default', ''(지정하지 않음), or 'PublicImage'로 지정하면, NCP VPC에서 제공하는 Public Image를 사용하여 VM을 생성할 수 있음.
   - VM 생성시 ImageType을 'MyImage'로 지정하면, MyImage 기능으로 VM에 대해 snapshot을 하여 VM의 RootDisk와 DataDisk들을 image로 생성하여 만든 image를 사용하여 VM을 생성할 수 있음.
     - MyImage를 이용하여 VM을 생성한 경우, snapshot 대상이었던 VM의 모든 disk가 그대로 신규 VM의 disk로 생성됨.

  ​O NCP VPC driver를 이용해 VM 생성시 VMSpec ID를 지정할때 다음 사항을 주의
   - VM 생성을 위해 입력하는 VM Image와 호환되는 VM Spec을 지정해야함.
     - Driver를 통해 조회된 VMSpec 목록의 부가정보들 중 'CorrespondingImageIds'에 그 Image ID가 포함되어 있는 VMSpec을 지정해야함.

  ​O NCP VPC driver를 이용해 VM 생성시 Root disk에 대해 다음 사항을 참고
   - VM 생성시 option으로 RootDiskSize 및 RootDiskType 지정은 지원하지 않음.
   - NCP VPC는 고정된 disk size로서, Linux 계열은 50GB, Windows 계열은 100GB를 지원함.
   - Disk type으로는 HDD와 SDD를 지원하는데, VMSpec type에따라 지원하는 type이 다르니 VMSpec 선정시 disk type 확인이 필요함.
   - NCP 3세대(KVM 기반) VM을 REST API를 통해 처음 생성시 NCP 고객센터에 3세대(3G)용 쿼터 증가시켜주기를 요청해야함.
     - 처음에는 "~ Product type: [G3] CPU > CPU Creation limit: 0 ~" 이라는 오류 발생
   - NCP 3세대(KVM 기반) VM은 disk size 지정 가능
     - 참고 : https://guide.ncloud-docs.com/docs/server-create-vpc

  O NCP VPC driver를 이용해 Root disk 외의 추가 Disk Volume(Block Storage) 생성시 다음 사항을 참고
   - (주의-1) NCP VPC에서 추가로 disk를 생성하기 위해서는 해당 region에 최소 하나의 VM이 생성되어있어야함.(Suspended or Running 상태의 VM)
   - (주의-2) 추가로 생성된 disk를 특정 VM에 data disk로 attach 한 후에 다음 사항을 주의
     - Data disk가 attach된 그 VM을 바로 삭제할 수 없고, 그 disk를 detach 한 후에 VM을 삭제할 수 있음.
     - 그 data disk를 삭제할 경우에도, VM에 attach된 상태에서는 삭제할 수 없고 detach 한 후에 삭제할 수 있음.
   - Disk Voulme 생성시 option으로 DiskType과 DiskSize를 다음과 같이 지정할 수 있음.
     - DiskType : 'default', 'SSD' or 'HDD' ('default'로 지정하면 'SSD'로 생성됨.)
     - DiskSize : 'default' or 10~2000(GB)의 범위로 숫자만 기입. ('default'로 지정하면 10GB가 생성됨.)

  O NCP VPC driver를 이용해 LoadBalancer 생성시 다음 사항을 참고('Network' Type LoadBalancer 기준임)
   - (주의) NLB 생성을 위해서는 타 CSP와 다르게 LB 전용(LB type)의 subnet을 이용해야하므로, NLB 생성시 driver 자체에서 LB 전용 subnet 유무를 확인 후 없으면 자동으로 생성함.(해당 VPC의 마지막 LB 삭제시, 이 subnet도 자동 삭제됨)
     - LB 전용 subnet의 CIDR은 driver에서 VPC CIDR 내의 'X.X.X.240/28'으로 생성되니 VPC 내에서 일반 subnet 생성시 이 CIDR 범위는 사용되지 않아야함.
     - VPC, 일반 subnet, LB 전용 Subnet 생성 예
       - VPC CIDR : '10.0.0.0/16'
       - 일반 subnet CIDR : '10.0.0.0/28'
       - LB 전용 Subnet CIDR : '10.0.0.240/28'(Driver에서 자동 생성됨)
     - LB 전용 subnet name은 'ncp-subnet-for-nlb-XXXXXXXX' 과 같은 형식으로 자동 부여됨. 
   - NCP VPC의 경우, 'SGN'(Singapore), 'JPN'(Japan) region에서만 internet gateway를 지원하는 public용 NLB를 생성할 수 있음.('KOR'(Korea) region에서는 private용 NLB만 지원)
     - 'SGN'(Singapore), 'JPN'(Japan) region에서 NLB 생성시 public IP가 지정됨.
   - NLB 생성시 option으로 NLB Network Type을 다음과 같이 지정할 수 있음.
     - 'default', ''(지정하지 않음), or 'PUBLIC'으로 지정하면, NCP VPC의 'PUBLIC' network type NLB가 생성됨.
     - 'INTERNAL'으로 지정하면, NCP VPC의 'PRIVATE' network type NLB가 생성됨.
   - NCP VPC NLB에서 지원하는 protocol은 다음과 같음. 
     - Listener protocol type : 'Network' Load Balancer는 TCP/UDP만 지원
       - 단, UDP protocol은 'SGN'(Singapore), 'JPN'(Japan) region에서만 이용 가능
     - VMGroup protocol type : 'Network' Load Balancer는 TCP/UDP만 지원
       - 단, UDP protocol은 'SGN'(Singapore), 'JPN'(Japan) region에서만 이용 가능
     - HealthChecker protocol type : 'Network' Load Balancer는 TCP만 지원
   - HealthChecker Iimeout 값 : LB type이 'NETWORK' 가 아닌 경우에만 유효(NLB에서는 무의미함.)
   - HealthChecker Interval 값 범위 : 5 ~ 300 (seconds). '-1' 입력시, default 값 '30' (seconds) 입력됨.
   - HealthChecker Threshold 값 범위 : 2 ~ 10. '-1' 입력시, default 값 '2' 입력됨.

​	O NCP CSP 정책상, 생성되는 public IP 개수가 VM instance 수를 초과할 수 없으므로 다음 사항을 주의
   - 만약, NCP VPC console에서 수동으로 VM을 반납(termination) 할 경우에는 반드시 public IP도 반납하는것으로 체크 후 반납 필요
     - Public IP를 반납하지 않으면, NCP VPC driver를 통해 VM instance 신규 생성 요청시, driver에서 public IP를 추가 생성할때 public IP 수가 instance 개수보다 많게 되어 error를 return 하게됨.
     - 단, NCP driver나 CB-Spider API를 이용해 NCP VM 반납시에는 VM 반납 후 자동으로 public IP까지 반납됨.

  O NCP 정책상, VM이 생성된 후 한번 정지시킨 상태에서 연속으로 최대 90일, 12개월 누적 180일을 초과하여 정지할 수 없으며, 해당 기간을 초과할 경우 반납(Termination)하도록 안내하고 있음.
