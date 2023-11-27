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

#### # 참고 사항

​	O VM 생성을 위한 VMImage ID, VMSpec ID 결정 관련
   - 해당 zone에서 지원하는 VM Image(KT Cloud의 Template) 목록중 사용하고자 하는 운영체제(OS)에 대한 Image ID 값을 찾은 뒤, VM Spec 목록에서 추가 정보로 제공하는 'SupportingImage(Template)ID'에서 그 Image ID와 같은 VM Spec을 찾아 해당 Image ID를 지원하는 VMSpec ID를 사용해야함.
   - 위와 같이 해당 VMImage를 지원하는 VMSpec ID를 사용해야하는데, 그렇지 않은 경우 KT Cloud에서는 error message로 "general error"를 return함.
<p><br>

​	O Security Group 설정시, inbound rule만 지원
   - 본 드라이버를 통해 KT Cloud에서 실제 적용시 public IP 단위로 firewall rule이 적용되는데, outbound rule은 지원하지 않으므로 outbound rule을 설정해도 적용되지 않음.
<p><br>

​	O Disk 추가 볼륨 생성 방법
   - VM Spec 조회시, Spec 이름의 맨 뒤에 붙은 disk 크기가 기본(Root) disk volume과 추가 volume을 합한 크기임.
      - 예) 97359d1d-a7b1-49d9-b435-14608543f00b#097b63d7-e725-4db7-b4dd-a893b0c76cb0_disk100GB
      - 위의 예의 경우, Linux 계열에서는 기본 volume 20GB에 80GB의 추가 볼륨이 생성되어 총 100GB가 됨.
   - VM 생성시 원하는 총 disk 크기에 따라 Spec을 결정해서 입력하면됨.
<p><br>
