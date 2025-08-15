# AWS EFS FileSystem Test Guide

## 개요
이 문서는 AWS EFS(Elastic File System) FileSystemHandler의 테스트 방법을 설명합니다.

## AWS EFS FileSystemHandler 구현 사항

### 지원하는 FileSystem 타입
- **RegionType**: Regional EFS (Multi-AZ) - 기본값
- **ZoneType**: One Zone EFS (단일 AZ)

### Performance Settings 지원
AWS EFS 콘솔과 동일한 Performance settings를 지원합니다:

#### Throughput Mode
- **Elastic** (권장): AWS EFS Elastic throughput mode
- **Bursting**: AWS EFS Bursting throughput mode (API 명칭)
- **Provisioned**: AWS EFS provisioned mode

#### Performance Mode
- **GeneralPurpose** (권장): General Purpose 성능 모드
- **MaxIO**: Max I/O 성능 모드 (One Zone 제외)

#### 제약사항
- **Elastic + MaxIO**: 지원하지 않음 (General Purpose만 가능)
- **One Zone + MaxIO**: 지원하지 않음 (General Purpose만 가능)
- **Provisioned + MaxIO**: 지원 (One Zone 제외)
- **Provisioned 모드**: ProvisionedThroughput 값 필수 (1-1024 MiB/s)
- **Default Lifecycle 일부만 지원**
  - Transition into **Infrequent Access** (IA):30 day(s) since last access는 지원됨
  - Transition into Archive:90 day(s) since last access는 지원 안됨.
    - Archive 옵션은 AWS V2 SDK에서만 지원됨

### Mount Target 생성 전략
1. **MountTargetList 제공**: 사용자가 지정한 서브넷과 보안 그룹으로 생성
2. **AccessSubnetList 제공**: 사용자가 지정한 서브넷으로 생성 (기본 보안 그룹 사용) - cb-spider 기본 권장 방식
3. **기본 동작**: AWS 콘솔과 동일하게 모든 가용 AZ에 Mount Target 생성

### 암호화 설정
- 사용자 선택 가능: `Encryption: true/false`
- 기본값: `true` (AWS EFS 권장사항)

### 백업 정책
- AWS EFS 자동 백업 지원


## 사전 요구사항
1. AWS 계정 및 적절한 권한
2. AWS EFS 서비스 접근 권한
3. VPC 및 Subnet 설정
4. 환경 설정 파일 (config.yaml)

## 환경 설정

### 1. 환경 변수 설정
```bash
export CBSPIDER_TEST_CONF_PATH=/path/to/your/config.yaml
```

### 2. config.yaml 예제
```yaml
aws:
  aws_access_key_id: "your-access-key"
  aws_secret_access_key: "your-secret-key"
  aws_sts_token: "your-sts-token"  # 선택사항
  region: "ap-northeast-2"
  zone: "ap-northeast-2a"
```

## 테스트 실행

### 1. 컴파일
```bash
cd cloud-control-manager/cloud-driver/drivers/aws/main
go build
```

### 2. 실행
```bash
./main
```

## 테스트 메뉴

FileSystem 테스트는 다음과 같은 메뉴를 제공합니다:

```
FileSystem Management
0. Quit
1. Get Meta Info
2. FileSystem List
3. FileSystem Create
4. FileSystem Get
5. FileSystem Delete
6. Add Access Subnet
7. Remove Access Subnet
8. List Access Subnet
9. ListIID
```

### 메뉴 설명

1. **Get Meta Info**: AWS EFS의 지원 기능 정보 조회
   - 지원하는 FileSystem 타입 (RegionType/ZoneType)
   - 지원하는 NFS 버전 (4.0, 4.1)
   - Performance Options (ThroughputMode, PerformanceMode)
   - 제약사항 및 사용 예시

2. **FileSystem List**: 생성된 모든 FileSystem 목록 조회

3. **FileSystem Create**: 새로운 EFS FileSystem 생성
   - 다양한 Performance settings 지원
   - Mount Target 생성 전략 선택 가능
   - 암호화 설정 선택 가능

4. **FileSystem Get**: 특정 FileSystem 상세 정보 조회

5. **FileSystem Delete**: FileSystem 삭제
   - Mount Target도 함께 삭제됨

6. **Add Access Subnet**: FileSystem에 접근 가능한 Subnet 추가
   - Mount Target 생성

7. **Remove Access Subnet**: FileSystem에서 Subnet 접근 권한 제거
   - Mount Target 삭제

8. **List Access Subnet**: FileSystem에 접근 가능한 Subnet 목록 조회

9. **ListIID**: FileSystem ID 목록 조회

## FileSystem 생성 예시

### 1. 기본 설정으로 생성 (권장)
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{
        NameId: "my-efs-basic",
    },
    VpcIID: irs.IID{
        SystemId: "vpc-xxxxxxxxx",
    },
    // 기본값: RegionType, Elastic, GeneralPurpose, Encryption: true
}
```

### 2. Regional EFS + Elastic + General Purpose
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{
        NameId: "my-efs-elastic",
    },
    VpcIID: irs.IID{
        SystemId: "vpc-xxxxxxxxx",
    },
    FileSystemType: irs.RegionType,
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Elastic",
        "PerformanceMode": "GeneralPurpose",
    },
    Encryption: true,
}
```

### 3. Regional EFS + Provisioned + Max I/O
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{
        NameId: "my-efs-provisioned-maxio",
    },
    VpcIID: irs.IID{
        SystemId: "vpc-xxxxxxxxx",
    },
    FileSystemType: irs.RegionType,
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Provisioned",
        "PerformanceMode": "MaxIO",
        "ProvisionedThroughput": "128",
    },
    Encryption: true,
}
```

### 4. One Zone EFS + Provisioned + General Purpose
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{
        NameId: "my-efs-onezone",
    },
    VpcIID: irs.IID{
        SystemId: "vpc-xxxxxxxxx",
    },
    FileSystemType: irs.ZoneType,
    Zone: "ap-northeast-2a",
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Provisioned",
        "PerformanceMode": "GeneralPurpose", // Max I/O는 One Zone에서 지원 안함
        "ProvisionedThroughput": "64",
    },
    Encryption: true,
}
```

### 5. Mount Target 지정으로 생성
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{
        NameId: "my-efs-with-mount-targets",
    },
    VpcIID: irs.IID{
        SystemId: "vpc-xxxxxxxxx",
    },
    MountTargetList: []irs.MountTargetInfo{
        {
            SubnetIID: irs.IID{
                SystemId: "subnet-xxxxxxxxx",
            },
            SecurityGroups: []string{"sg-xxxxxxxxx"},
        },
    },
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Bursting",
        "PerformanceMode": "MaxIO",
    },
    Encryption: false,
}
```

### 6. Access Subnet만 지정으로 생성
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{
        NameId: "my-efs-with-subnets",
    },
    VpcIID: irs.IID{
        SystemId: "vpc-xxxxxxxxx",
    },
    AccessSubnetList: []irs.IID{
        {SystemId: "subnet-xxxxxxxxx"},
        {SystemId: "subnet-yyyyyyyyy"},
    },
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Elastic",
        "PerformanceMode": "GeneralPurpose",
    },
    Encryption: true,
}
```

### 7. 태그와 함께 생성
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{
        NameId: "my-efs-with-tags",
    },
    VpcIID: irs.IID{
        SystemId: "vpc-xxxxxxxxx",
    },
    TagList: []irs.KeyValue{
        {Key: "Environment", Value: "Production"},
        {Key: "Project", Value: "CB-Spider"},
        {Key: "CostCenter", Value: "IT-001"},
    },
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Provisioned",
        "PerformanceMode": "GeneralPurpose",
        "ProvisionedThroughput": "256",
    },
    Encryption: true,
}
```

## 주의사항

1. **VPC 설정**: FileSystem은 VPC 내에서만 생성 가능
2. **Subnet 설정**: Mount Target 생성을 위해 적절한 Subnet 필요
3. **보안 그룹**: NFS 포트(2049) 접근 허용 필요
4. **비용**: EFS 사용 시 비용 발생
5. **삭제**: FileSystem 삭제 시 모든 데이터 손실
6. **One Zone 제약**: One Zone EFS는 Max I/O 성능 모드 지원 안함
7. **Elastic 제약**: Elastic Throughput Mode는 General Purpose 성능 모드만 지원

## 지원되지 않는 기능

- 스냅샷/백업 기능 (향후 개발 예정)
- NFS 4.0, 4.1 이외의 버전
- 용량 지정 (자동 확장)


## 문제 해결

### 일반적인 오류

1. **권한 오류**: AWS IAM 권한 확인
2. **VPC 오류**: VPC ID 및 Subnet ID 확인
3. **네트워크 오류**: 보안 그룹 설정 확인
4. **리소스 제한**: AWS 계정의 EFS 제한 확인
5. **Performance 설정 오류**: GetMetaInfo()로 지원되는 옵션 확인

### 로그 확인

테스트 실행 시 상세한 로그가 출력됩니다:
- API 호출 정보
- 오류 메시지
- 응답 데이터
- Performance settings 검증 결과

## 추가 정보

- [AWS EFS 공식 문서](https://docs.aws.amazon.com/efs/)
- [AWS EFS 성능 가이드](https://docs.aws.amazon.com/ko_kr/efs/latest/ug/performance.html)
- [CB-Spider File Storage Guide](https://github.com/cloud-barista/cb-spider/wiki/File-Storage-Guide) 