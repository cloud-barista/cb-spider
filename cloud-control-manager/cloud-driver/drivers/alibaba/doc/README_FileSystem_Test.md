# Alibaba Cloud NAS FileSystem Test Guide

## 개요
이 문서는 Alibaba Cloud NAS(Network Attached Storage) FileSystemHandler의 구현 사항과 테스트 방법을 설명합니다.

## Alibaba Cloud NAS FileSystemHandler 구현 사항

### 지원하는 FileSystem 타입
- **ZoneType**: Zone-based NAS (단일 Zone) - 기본값
- **RegionType**: 지원하지 않음 (Alibaba NAS는 Zone 기반)

### Storage Type 설정
**리전별로 사용 가능한 스토리지 타입이 다르며, API 필수 설정 항목인데 API의 기본값이 없기에 StorageType은 사용자 설정이 필수입니다.**   

Alibaba Cloud NAS는 아래와 같이 **FileSystemType에 따라 다른 StorageType을 지원**합니다. 

**[주의]**: FileSystemHandler 인터페이스에는 StorageType 필드가 정의되어 있지 않으므로, PerformanceInfo 옵션을 통해 설정해야 합니다.

#### Standard FileSystemType (기본값)
- **Performance**: Performance-based storage (성능 기반 과금)
- **Capacity**: Capacity-based storage (용량 기반 과금) - 기본값
- **Premium**: Premium storage (프리미엄 성능)

#### Extreme FileSystemType
- **standard**: Standard storage for extreme NAS
- **advance**: Advanced storage for extreme NAS

#### CPFS FileSystemType
- **standard**: Standard storage for CPFS

### Capacity 설정
Extreme 타입만 CapacityGB 설정이 가능하며, 그 외의 파일 시스템 타입은 무시됩니다.

**Capacity 설정 방법**:
- **주요 방법**: `CapacityGB` 필드 사용 (권장)
- **대체 방법**: `PerformanceInfo["Capacity"]` 사용 (fallback)

- **Standard FileSystemType**: Capacity 자동 관리 (사용자 설정 불가)
- **Extreme FileSystemType**: 100GB ~ 262,144GB 범위에서 필수 설정
- **CPFS FileSystemType**: 용량 설정 지원 (범위 확인 필요)

### Protocol 지원
- **NFS**: NFS v3.0, v4.0 지원 (기본값: v4.0)

### 암호화 설정
- **기본값**: `Encryption: false`로 고정되어 있습니다.
- **향후 계획**: 암호화 기능은 향후 개발 예정입니다.

### Mount Target 생성 전략
1. **AccessSubnetList 제공**: 사용자가 지정한 서브넷으로 Mount Target 생성
2. **기본 동작**: 지정된 서브넷에 Mount Target 생성

## 사전 요구사항
1. Alibaba Cloud 계정 및 적절한 권한
2. Alibaba Cloud NAS 서비스 접근 권한
3. VPC 및 Subnet 설정
4. 환경 설정 파일 (config.yaml)

## 환경 설정

### 1. 환경 변수 설정
```bash
$ source setup.env
$ source develop.env
```

### 2. config.yaml 예제
```yaml
alibaba:
  ali_access_key_id: "your-access-key"
  ali_secret_access_key: "your-secret-key"
  region: "ap-northeast-1"
  zone: "ap-northeast-1a"
```

## 테스트 실행

### 실행
```bash
$ cdalibaba
$ cd main
$ go run Test_Resources.go
```

## 테스트 메뉴

FileSystem 테스트는 다음과 같은 메뉴를 제공합니다:

```
FileSystem Management
0. Quit
1. GetMetaInfo
2. FileSystem List
3. FileSystem Create (Basic Setup)
4. FileSystem Create (Premium Storage)
5. FileSystem Create (Extreme FileSystemType)
6. FileSystem Get
7. FileSystem Delete
8. Add Access Subnet
9. Remove Access Subnet
10. List Access Subnet
11. List IID
```

### 메뉴 설명

1. **GetMetaInfo**: Alibaba Cloud NAS의 지원 기능 정보 조회
   - 지원하는 FileSystem 타입 (ZoneType)
   - 지원하는 NFS 버전 (3.0, 4.0)
   - StorageType 옵션 (FileSystemType별)
   - Capacity 규칙 및 제약사항

2. **FileSystem List**: 생성된 모든 FileSystem 목록 조회

3. **FileSystem Create (Basic Setup)**: 기본 설정으로 NAS FileSystem 생성
   - FileSystemType: standard (기본값)
   - StorageType: Capacity (기본값)
   - ProtocolType: NFS (기본값)

4. **FileSystem Create (Premium Storage)**: Premium Storage로 FileSystem 생성
   - FileSystemType: standard
   - StorageType: Premium

5. **FileSystem Create (Extreme FileSystemType)**: Extreme FileSystemType으로 생성
   - FileSystemType: extreme
   - StorageType: standard
   - Capacity: 100GB (필수)

6. **FileSystem Get**: 특정 FileSystem 상세 정보 조회

7. **FileSystem Delete**: FileSystem 삭제
   - Mount Target도 함께 삭제됨

8. **Add Access Subnet**: FileSystem에 접근 가능한 Subnet 추가
   - Mount Target 생성

9. **Remove Access Subnet**: FileSystem에서 Subnet 접근 권한 제거
   - Mount Target 삭제

10. **List Access Subnet**: FileSystem에 접근 가능한 Subnet 목록 조회

11. **ListIID**: FileSystem ID 목록 조회

## FileSystem 생성 예시

### 1. 기본 설정으로 생성 (권장)
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{
        NameId: "my-nas-basic",
    },
    VpcIID: irs.IID{
        SystemId: "vpc-xxxxxxxxx",
    },
    Zone: "ap-northeast-1a",
    AccessSubnetList: []irs.IID{
        {SystemId: "vsw-xxxxxxxxx"},
    },
    PerformanceInfo: map[string]string{
        "StorageType": "Capacity", // 기본값
    },
}
```

### 2. Standard FileSystemType + Premium Storage
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{
        NameId: "my-nas-premium",
    },
    VpcIID: irs.IID{
        SystemId: "vpc-xxxxxxxxx",
    },
    Zone: "ap-northeast-1a",
    AccessSubnetList: []irs.IID{
        {SystemId: "vsw-xxxxxxxxx"},
    },
    PerformanceInfo: map[string]string{
        "StorageType": "Premium",
    },
}
```

### 3. Standard FileSystemType + Performance Storage
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{
        NameId: "my-nas-performance",
    },
    VpcIID: irs.IID{
        SystemId: "vpc-xxxxxxxxx",
    },
    Zone: "ap-northeast-1a",
    AccessSubnetList: []irs.IID{
        {SystemId: "vsw-xxxxxxxxx"},
    },
    PerformanceInfo: map[string]string{
        "StorageType": "Performance",
    },
}
```

### 4. Extreme FileSystemType + Standard Storage
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{
        NameId: "my-nas-extreme",
    },
    VpcIID: irs.IID{
        SystemId: "vpc-xxxxxxxxx",
    },
    Zone: "ap-northeast-1a",
    AccessSubnetList: []irs.IID{
        {SystemId: "vsw-xxxxxxxxx"},
    },
    CapacityGB: 1024, // 100GB ~ 262,144GB 범위에서 필수
    PerformanceInfo: map[string]string{
        "FileSystemType": "extreme",
        "StorageType": "standard",
    },
}
```

### 5. Extreme FileSystemType + Advance Storage
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{
        NameId: "my-nas-extreme-advance",
    },
    VpcIID: irs.IID{
        SystemId: "vpc-xxxxxxxxx",
    },
    Zone: "ap-northeast-1a",
    AccessSubnetList: []irs.IID{
        {SystemId: "vsw-xxxxxxxxx"},
    },
    CapacityGB: 2048, // 100GB ~ 262,144GB 범위에서 필수
    PerformanceInfo: map[string]string{
        "FileSystemType": "extreme",
        "StorageType": "advance",
    },
}
```

### 6. 태그와 함께 생성
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{
        NameId: "my-nas-with-tags",
    },
    VpcIID: irs.IID{
        SystemId: "vpc-xxxxxxxxx",
    },
    Zone: "ap-northeast-1a",
    AccessSubnetList: []irs.IID{
        {SystemId: "vsw-xxxxxxxxx"},
    },
    TagList: []irs.KeyValue{
        {Key: "Environment", Value: "Production"},
        {Key: "Project", Value: "CB-Spider"},
        {Key: "CostCenter", Value: "IT-001"},
    },
    PerformanceInfo: map[string]string{
        "StorageType": "Capacity",
    },
}
```

## 주의사항

1. **VPC 설정**: FileSystem은 VPC 내에서만 생성 가능
2. **Zone 설정**: Alibaba NAS는 Zone 기반이므로 Zone 지정 필수
3. **Subnet 설정**: Mount Target 생성을 위해 적절한 Subnet 필요
4. **StorageType 필수**: 모든 FileSystem 생성 시 StorageType 지정 필수
5. **Capacity 제약**: Extreme FileSystemType에서만 Capacity 설정 가능
6. **비용**: NAS 사용 시 비용 발생
7. **삭제**: FileSystem 삭제 시 모든 데이터 손실

## 제약사항

### FileSystemType별 제약
- **Standard**: Capacity 자동 관리, Performance/Capacity/Premium StorageType 지원
- **Extreme**: Capacity 필수 설정 (100GB ~ 262,144GB), standard/advance StorageType 지원
- **CPFS**: Cloud Parallel File Storage, 특정 용도로만 사용

### StorageType 제약
- **Performance**: 성능 기반 과금, 높은 IOPS 제공
- **Capacity**: 용량 기반 과금, 비용 효율적
- **Premium**: 프리미엄 성능, 높은 처리량 제공

### Protocol 제약
- **NFS**: Linux/Unix 환경에서 주로 사용

## 지원되지 않는 기능

- 스냅샷/백업 기능 (향후 개발 예정)
- NFS 3.0, 4.0 이외의 버전
- Region 기반 FileSystem (Zone 기반만 지원)
- Standard FileSystemType에서 Capacity 수동 설정
- 암호화 설정 (향후 개발 예정)
- SMB 프로토콜 (현재 NFS만 지원)

## 알려진 버그
- Extreme NAS 생성 시 비동기로 생성되며 Tag 및 마운트 타겟 등이 생성되지 않음.
    - 향후 수정 예정


## 일반 문제 해결

### 일반적인 오류

1. **StorageType 오류**: "StorageType is mandatory for this action"
   - 해결: PerformanceInfo에 StorageType 지정
   - 예시: `"StorageType": "Capacity"`

2. **FileSystemType 불일치**: "invalid StorageType for FileSystemType"
   - 해결: FileSystemType에 맞는 StorageType 사용
   - Standard: Performance/Capacity/Premium
   - Extreme: standard/advance

3. **Capacity 오류**: "capacity is required for extreme file system type"
   - 해결: Extreme FileSystemType에서 Capacity 지정
   - 예시: `"Capacity": "1024"`

4. **권한 오류**: Alibaba Cloud RAM 권한 확인
5. **VPC 오류**: VPC ID 및 Subnet ID 확인
6. **네트워크 오류**: 보안 그룹 설정 확인

### 로그 확인

테스트 실행 시 상세한 로그가 출력됩니다:
- API 호출 정보
- 오류 메시지
- 응답 데이터
- Performance settings 검증 결과

## 추가 정보

- [Alibaba Cloud NAS 공식 문서](https://www.alibabacloud.com/help/en/nas)
- [Alibaba Cloud NAS API 문서](https://www.alibabacloud.com/help/en/nas/developer-reference/api-nas-2017-06-26-createfilesystem)
- [CB-Spider File Storage Guide](https://github.com/cloud-barista/cb-spider/wiki/File-Storage-Guide)
