### NCP (Naver Cloud Platform) Classic driver Build 및 CB-Spider에 적용 방법


#### # 연동 Driver 관련 기본적인 사항은 아래 link 참고

   - [Cloud Driver Developer Guide](https://github.com/cloud-barista/cb-spider/wiki/Cloud-Driver-Developer-Guide) 
<p><br>
#### # CB-Spider에서 NCP 연동 driver build 및 구동 방법

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

​	O CB-Spider server 구동(Dynamic plugin 방식, 1024 포트로 REST API server 구동됨)

```
cd $CBSPIDER_ROOT/bin

./start-dyna.sh
```

   - CB-Spider server가 구동된 후, NCP driver 등록 과정을 거치고 사용

<p><br>

#### # CB-Spider에 NCP driver 테스트 방법

​	O NCP Credential 정보(API 인증키) 발급 방법<BR>
 - [네이버클라우드 포털](https://www.ncloud.com) > 마이페이지 > 계정관리 > '인증키 관리' 메뉴로 이동<br>
   > [신규 API 인증키 생성] 버튼을 클릭하면 인증키가 생성됨.
 - 네이버클라우드 Credential 정보(API 인증키) 생성 메뉴 바로 가기 : 네이버클라우드 포털 > [인증키 관리](https://www.ncloud.com/mypage/manage/authkey) 메뉴

   (참고) 네이버클라우드 API 인증키는 계정당 2개까지 생성할 수 있음.

​	O NCP connection config 파일에 NCP Credential 정보(API 인증키) 기입 후 실행<BR>

```
$CBSPIDER_ROOT/api-runtime/rest-runtime/test/connect-config/ ./8.ncp-conn-config.sh
```
<p><br>

#### # CB-Spider REST API 이용 NCP driver 모든 기능 테스트

​	O NCP 각 자원 생성, 자원정보 조회, VM instance Start 및 Terminate 테스트 등

-   Curl command를 이용하는 테스트 script
```
$CBSPIDER_ROOT/api-runtime/rest-runtime/test/full-test/8.ncp-test.sh
$CBSPIDER_ROOT/api-runtime/rest-runtime/test/full-test/ncp-full_test.sh
```
<p><br>

#### # NCP driver 자체 test 파일을 이용한 기능 테스트

​	O CB-Spider 환경 파일 적용
```
$CBSPIDER_ROOT/ source setup.env
```

​	O 아래의 config 파일에 NCP Credential 정보 기입
```
$CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/ncp/main/config/config.yaml
```

​	O 아래의 위치에 있는 ~.sh 파일을 실행해서 NCP driver 각 handler 세부 기능 테스트 
```
$CBSPIDER_ROOT/cloud-control-manager/cloud-driver/drivers/ncp/main/
```
<p><br>

#### # NCP cloud driver로 생성된 VM에 로그인하는 방법

​	O Cloud-Barista NCP driver를 이용해 생성된 VM에 로그인하는 방법(Linux 계열 guest OS 경우)

   - 사용자가 keyPair 생성시 생성되는 private key를 local에 저장해놓고, 그 private key를 이용해 'cb-user' 계정으로 SSH를 이용해 로그인하는 방법을 제공함.

   - Private ket 파일을 이용한 VM 접속 방법 

```
ssh -i /private_key_경로/private_key_파일명(~~.pem) cb-user@VM의_public_ip
```


​	(참고) NCP CSP에서 제공하는 VM 접속 방법

   - VM 생성 후, NCP console에서 사용자가 생성한 private key를 이용해 해당 VM의 root 비밀번호를 알아내어 SSH를 이용해 root 계정과 비밀번호로 로그인함.

​	O Cloud-Barista NCP driver를 이용해 생성된 VM에 로그인하는 방법(Windows 계열 guest OS 경우)

   - 'Administrator' 계정으로 VM 생성시 지정한 사용자가 password 로그인함.

<p><br>

(주의 사항)<br>
​	O NCP Classic 버전 CSP는 물리적 네트워크 기반으로 운영되므로, NCP에서 VPC/Subnet 생성 등의 제어 기능 및 API를 지원하지 않음.
   - 따라서, 본 driver를 통해 VPC 및 Subnet 관련 제어가 진행될때 VPC 및 Subnet 정보는 driver 자체에서 임의적으로 local에서 JSON 파일 형태로 관리됨.(VPC 및 Subnet 생성시 어떤 Name이든 가능)

​	O  NCP Classic CSP는 VPC 및 Subnet 제어 기능을 제공하지 않으므로, 본 driver에서 지원하는 VPC와 Subnet 제어 기능은 CB-Spider common interface를 만족하기 위해 제공하는 임의의 VPC 및 Subnet을 위한 제어 기능임.
   - NCP Classic에서 사용하는 private IP의 IPv4_CIDR은 10.0.0.0/XX 임.

​	O NCP Classic CSP는 Security Group도 REST API를 통해서는 생성, 삭제 기능을 지원하지 않고 조회 기능만 지원함(Console에서는 모든 기능 지원). 따라서, CB-Spider 및 NCP 연동 driver를 통한 제어를 위해서는 다음 사항을 주의해야함.

   - Security Group 생성/삭제시에는 NCP console에서 생성/삭제하고, Cloud-Barista에서 생성, 조회, 삭제 요청시 console에서 생성한 그 'Security Group 이름을 그대로' 사용함.
      - Console에서 생성시, 이름은 최소 6자/최대 30자의 소문자만 가능하고, 숫자, '-' 기호 사용 가능

​	O NCP Classic NLB 이용시 다음 사항을 주의해야함.
   - NCP Classic NLB는 HTTP, HTTPS, TCP, SSL protocol만 지원함.
   - Listener, VM Group, HealthChecker protocol이 동일해야함.

  O NCP Classic 버전 driver이 지원하는 region 및 zone은 아래의 파일을 참고
```
  ./ncp/ncp/main/config/config.yaml.sample
  ./cb-spider/api-runtime/rest-runtime/test/connect-config/8.ncp-conn-config.sh
```

​	O  NCP 정책상, 생성되는 public IP 개수가 VM instance 수를 초과할 수 없으므로 다음 사항을 주의

   - 만약, NCP console에서 수동으로 VM을 반납(termination) 할 경우에는 반드시 public IP도 반납하는것으로 체크 후 반납 필요
     - Public IP를 반납하지 않으면, NCP driver를 통해 VM instance 신규 생성시 public IP를  추가 생성할때 public IP 수가 instance 개수보다 많게 되어 error return

     - 단, NCP driver나 CB-Spider API를 이용해 NCP VM 반납시에는 VM 반납 후 자동으로 public IP까지 반납됨.

​	O  NCP 정책상, VM이 생성된 후 한번 정지시킨 상태에서 연속으로 최대 90일, 12개월 누적 180일을 초과하여 정지할 수 없으며, 해당 기간을 초과할 경우 반납(Terminate)하도록 안내하고 있음.
