### KT Cloud connection driver Build 및 CB-Spider에 적용 방법

#### # 연동 Driver 관련 기본적인 사항은 아래 link 참고

   - [Cloud Driver Developer Guide](https://github.com/cloud-barista/cb-spider/wiki/Cloud-Driver-Developer-Guide) 
<p><br>

#### # CB-Spider에 KT Cloud 연동 driver 적용 방법

​	O CB-Spider 코드가 clone된 상태에서 setup 파일 적용
```
$CBSPIDER_ROOT/ source setup.env
```

​	O Dynamic plugin mode로 CB-Spider build 실행

```
cd $CBSPIDER_ROOT

make dyna

```
   - CB-Spider Build 과정이 완료되면, $CBSPIDER_ROOT/bin/에 binary 파일로 'cb-spider-dyna' 가 생김 

<p><br>

​	O CB-Spider server 구동(Dynamic plugin 방식, 1024 포트로 REST API Server 구동됨)

```
cd bin

./start-dyna.sh
```

   - CB-Spider server가 구동된 후, KT Cloud driver 등록 과정을 마친 후 사용<BR>
     (아래의 11.ktcloud-conn-config.sh 파일을 실행해서 등록 가능)

<p><br>

#### # CB-Spider에 KT Cloud driver 테스트 방법

​	O 위와 같은 방법으로 CB-Spider 서버가 구동된 상태에서, 아래 위치의 KT Cloud connection config 파일에 KT Cloud Credential 정보 기입 후 실행<BR>
```
$CBSPIDER_ROOT/api-runtime/rest-runtime/test/connect-config/ ./11.ktcloud-conn-config.sh
```

<p><br>

​	O AdminWeb 기능을 이용한 테스트

   - 위와 같이 connection config 정보가 기입된 상태에서 http://localhost:1024/spider/adminweb 로 접속하여 테스트

<p><br>

#### # KT Cloud driver 자체 test 파일을 이용한 기능 테스트

​	O CB-Spider 환경 파일 적용
```
$CBSPIDER_ROOT/ source setup.env
```

​	O 아래 위치의 config 파일에 KT Cloud Credential 정보 기입
```
$GOPATH/src/github.com/cloud-barista/ktcloud/ktcloud/main/config/config.yaml
```

​	O 아래의 위치에 있는 ~.sh 파일을 실행해서 KT Cloud driver 각 handler 세부 기능 테스트 
```
$GOPATH/src/github.com/cloud-barista/ktcloud/ktcloud/main/
```
<p><br>

#### # KT Cloud Classic (G1/G2) 드라이버 이용시 참고 사항
​	O KT Cloud Classic 버전(G1/G2) CSP 인프라 서비스는 물리적 네트워크 기반으로 운영되므로, KT에서 VPC/Subnet 생성 등의 제어 기능 및 관련 API를 지원하지 않음.
   - 따라서, 본 driver를 통해 VPC 및 Subnet 관련 제어가 실행될때, VPC 및 Subnet 정보는 driver 자체에서 임의적으로 local에서 JSON 파일 형태로 관리됨.(VPC 및 Subnet 생성시 어떤 Name이든 가능)
<p><br>

​	O  KT Cloud Classic 인프라 서비스는 VPC 및 Subnet 제어 기능을 제공하지 않으므로, 본 driver에서 지원하는 VPC와 Subnet 제어 기능은 CB-Spider common interface를 만족하기 위해 제공하는 임의의 VPC 및 Subnet을 위한 제어 기능임.
   - KT Cloud Classic 서비스 기반으로 VM 생성시 적용되는 private IP의 IPv4_CIDR은 KT Cloud Zone마다 다른 대역을 지원함.
<p><br>

​	O VM 생성을 위한 VMImage ID, VMSpec ID 결정 관련
   - 해당 zone에서 지원하는 VM Image(KT Cloud의 Template) 목록중 사용하고자 하는 운영체제(OS)에 대한 Image ID 값을 찾은 뒤, VM Spec 목록에서 추가 정보로 제공하는 'SupportingImage(Template)ID'에서 그 Image ID와 같은 VM Spec을 찾아 해당 Image ID를 지원하는 VMSpec ID를 사용해야함.
   - 위와 같이 해당 VMImage를 지원하는 VMSpec ID를 사용해야하는데, 그렇지 않은 경우 KT Cloud에서는 error message로 "general error"를 return함.
<p><br>

​	O Security Group 설정시 주의해야할 사항으로, 본 드라이버는 inbound rule만 지원하고, protocol별 rule이 중복되지 않아야함.
   - KT Cloud Classic(G1/G2) 인프라 서비스는 port forwarding rule, firewall rule 적용시 inbound rule만 지원함.
   - Security Group을 생성하고 VM 생성시 본 드라이버 내에서 그 Security Group의 rule을 VM에 매핑된 public IP 기준으로 port forwarding rule, firewall rule이 적용됨.
   - 이때 KT Cloud Classic(G1/G2)에서 inbound rule만 지원하므로, Security Group에 outbound rule을 설정해도 inbound rule만 적용됨.
   - 추가로 주의해야할 사항으로, 본 드라이버로 Security Group 설정시 protocol별 rule이 중복되지 않아야함.
     - 예를들어, 현재 버전의 드라이버를 기준으로, VM 생성 후 드라이버 내부적으로 Security Group 적용시에 TCP 22번 port를 open하는 rule을 적용하고, TCP 모든 port를 open하는 rule을 적용할 수 없음.
<p><br>

  O 생성되는 VM의 root disk(volume) type 및 size (KT Cloud Classic(G1/G2) 기준)
   - VM 생성시, root disk type으로 'Seoul-M2' zone은 SSD type만 지원하고, 나머지 zone은 HDD type만 지원함.​
   - Root disk(volume) size는 default로 Linux 20G, Windows 50G로 지원됨.(향후 변경될 수 있음)
<p><br>

  O 'VM 생성시', Data disk (추가 volume) 생성 방법
   - VM Spec 조회시, Spec 이름의 맨 뒤에 붙은 disk 크기가 기본(Root) disk volume과 추가 volume을 합한 크기임.
      - 본 드라이버를 통해 조회되는 VM Spec 예) 97359d1d-a7b1-49d9-b435-14608543f00b#097b63d7-e725-4db7-b4dd-a893b0c76cb0_disk100GB
      - 위의 예의 경우, Linux 계열에서는 기본 volume 20GB에 80GB의 추가 볼륨이 생성되어 총 100GB가 됨.
   - VM 생성시 원하는 총 disk 크기에 따라 Spec을 결정해서 입력하면됨.
<p><br>

  O 일반적인 Data disk (추가 volume) 생성 방법
   - 본 드라이버에서 data disk 생성시, disk type은 'HDD'와 'SSD-Provisioned'를 지원함.
     - 참고) Zone별로 가용한 disk type이 다르므로, 본 드라이버에서는 현재 모든 zone에서 가용한 위의 두가지만 지원함.(향후 변경 가능)
   - Disk size 지정시, type별로 아래의 기준으로 지정해야함.
     - HHD : 10 ~ 300G(10G 단위 지정) (단, Seoul-M2 존은 400G 및 500G 지정 가능)
     - SSD-Provisioned : 100 ~ 800G(100G 단위 지정)
     - 아래의 link에서 'Volume : 생성' 부분 > 'diskofferingid' 표 참고
       - https://cloud.kt.com/docs/open-api-guide/g/computing/disk-volume
   - Disk 생성시 다른 해당 connection 외의 다른 zone을 지정하여 disk를 생성할 수 없음.
       - 다른 zone을 지정하여 생성할 경우 CSP(KT Cloud) API로부터 오류 발생
<p><br>

  O VM을 대상으로 MyImage 생성 및 삭제시 주의할 점
   - KT Cloud Classic 클라우드 서비스에서는 VM이 중지된(Suspended) 상태에서만 MyImage(KT Cloud Template) 생성이 가능함.
   - 생성된지 1시간 이내의 MyImage(KT Cloud Template)는 삭제할 수 없음.
<p><br>

#### # KT Cloud Classic (G1/G2) 드라이버 개발시 참고 사항
​	O 생성되는 VM의 root disk(volume) type 정보
   - KT Cloud Volume 정보에서 root disk의 type 정보는 제공하지 않음.
     - 아래의 기준을 드라이버에 반영함.
       - Seoul M2 zone은 SSD type으로 root volume이 생성되고, 나머지 zone은 HDD type으로 생성됨.​
<p><br>