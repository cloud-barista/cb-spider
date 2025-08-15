## 테스트 시나리오 관련해서는 에러가 많으며 임시로 작업 중이니 참고만 하세요.
# AWS EFS FileSystem Create Test Scenarios

## 개요
이 문서는 AWS EFS FileSystemHandler의 `CreateFileSystem` 메서드에 대한 사용자들의 기능 이해를 위해 가능한 조합들의 생성 예시 시나리오를 정의합니다. 각 시나리오는 `irs.FileSystemInfo` 구조체의 다양한 필드 조합에 대해 AWS EFS 생성 로직이 어떤 형태로 처리되는지 설명합니다.

## 테스트 환경 정보
- **리전**: 서울 (ap-northeast-2)
- **VPC**: vpc-0a48d45f6bc3a71da
- **서브넷**:
  - ap-northeast-2a (apne2-az1): subnet-04bd8bcbeb8cf7748
  - ap-northeast-2b (apne2-az2): subnet-08124f8bc6b14d6c9
- **보안 그룹**:
  - sg-0f5fdc13eef5c83c3
  - sg-0c88474826a32fb4c

## AWS EFS CreateFileSystem 처리 로직

### 주요 검증 로직
1. **NFS 버전 검증**: 4.0, 4.1 지원 (기본값: 4.1)
2. **VPC 필수 검증**: VPC SystemId가 반드시 필요
3. **태그 처리**: 사용자 태그 + Name 태그 자동 추가
4. **기본 설정 모드**: FileSystemType이 비어있으면 기본값 적용
5. **성능 설정 검증**: PerformanceInfo의 유효성 검사
6. **암호화 설정**: 사용자 선택에 따라 적용
7. **파일 시스템 타입**: RegionType (Multi-AZ) 또는 ZoneType (One Zone)
8. **Zone 정보 처리**: One Zone EFS 생성 시 Zone 우선순위 적용
9. **마운트 타겟 생성**: 3가지 전략 (AccessSubnetList, MountTargetList, 기본 동작)

### Zone 정보 처리 로직 (One Zone EFS)
One Zone EFS 생성 시 Zone 정보는 다음 우선순위로 처리됩니다:
1. **reqInfo.Zone**: 사용자가 명시적으로 지정한 Zone
2. **클라이언트 설정 Zone**: fileSystemHandler.Region.Zone
3. **자동 선택**: 리전의 첫 번째 사용 가능한 Zone

### 지원되는 성능 옵션
- **ThroughputMode**: Elastic, Bursting, Provisioned
- **PerformanceMode**: GeneralPurpose, MaxIO
- **제약사항**:
  - Elastic + MaxIO: 지원 안됨
  - One Zone + MaxIO: 지원 안됨 (GeneralPurpose로 자동 변경)
  - Provisioned 요청 시 ProvisionedThroughput 값 설정 필수 (1-1024 MiB/s)

## 사용 시나리오

### 1. 기본 설정 모드 (Basic Setup Mode)

#### 1.1 최소 필수 설정
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-basic-01"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
}
```
**예상 결과**: 
- ✅ 성공
- FileSystemType: RegionType (기본값)
- Encryption: true (기본값)
- NFSVersion: 4.1 (기본값)
- PerformanceMode: generalPurpose (AWS 기본값)
- ThroughputMode: bursting (AWS 기본값)
- Backup: true (기본값)
- 기본 라이프사이클 정책 적용

#### 1.2 VPC 없이 호출
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-no-vpc"},
    VpcIID: irs.IID{SystemId: ""},
}
```
**예상 결과**: 
- ❌ 실패
- 에러: "VPC is required for AWS EFS file system creation"

#### 1.3 태그 처리 (Name Tag 미지정)
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-with-tags"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    TagList: []irs.KeyValue{
        {Key: "Environment", Value: "Production"},
        {Key: "Project", Value: "TestProject"},
    },
}
```
**예상 결과**: 
- ✅ 성공
- 사용자 요청 태그 추가
- Name 태그 자동 추가 (IId.NameId 사용)
- 등록된 태그: Environment=Production, Project=TestProject, Name=efs-with-tags

#### 1.4 Name 태그가 있는 경우
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-name-tag-exists"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    TagList: []irs.KeyValue{
        {Key: "Name", Value: "CustomName"},
        {Key: "Environment", Value: "Dev"},
    },
}
```
**예상 결과**: 
- ✅ 성공
- 사용자 요청 태그 추가
- Name 태그는 사용자가 요청한 태그를 사용(IId.NameId 태그 사용 안 함)
- 등록된 태그: Name=CustomName, Environment=Dev

### 2. 고급 설정 모드 (Advanced Setup Mode)

#### 2.1 RegionType (Multi-AZ) + 기본 성능 설정
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-region-basic"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
    Encryption: true,
    NFSVersion: "4.1",
}
```
**예상 결과**: 
- ✅ 성공
- Multi-AZ EFS 생성
- AWS 기본 성능 설정 사용
- 암호화 활성화(KMS)

#### 2.2 ZoneType (One Zone) + 기본 성능 설정
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-zone-basic"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.ZoneType,
    Zone: "ap-northeast-2a",
    Encryption: true,
    NFSVersion: "4.1",
}
```
**예상 결과**: 
- ✅ 성공
- One Zone EFS 생성 (ap-northeast-2a)
- AWS 기본 성능 설정 사용
- 암호화 활성화(KMS)

#### 2.3 ZoneType + Zone 미지정
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-zone-auto"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.ZoneType,
    Encryption: true,
    NFSVersion: "4.1",
}
```
**예상 결과**: 
- ✅ 성공
- Zone 자동 결정
  - 결정 순서 :  1) reqInfo.Zone → 2) 클라이언트 설정 Zone → 3) 리전의 첫 번째 사용 가능한 Zone
- One Zone EFS 생성

### 3. 성능 설정 테스트

#### 3.1 Elastic + GeneralPurpose (권장 조합)
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-elastic-gp"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Elastic",
        "PerformanceMode": "GeneralPurpose",
    },
}
```
**예상 결과**: 
- ✅ 성공
- Elastic throughput mode
- GeneralPurpose performance mode

#### 3.2 Bursting + MaxIO
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-bursting-maxio"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Bursting",
        "PerformanceMode": "MaxIO",
    },
}
```
**예상 결과**: 
- ✅ 성공
- Bursting throughput mode
- MaxIO performance mode

#### 3.3 Provisioned + GeneralPurpose + ProvisionedThroughput
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-provisioned-gp"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Provisioned",
        "PerformanceMode": "GeneralPurpose",
        "ProvisionedThroughput": "128",
    },
}
```
**예상 결과**: 
- ✅ 성공
- Provisioned throughput mode
- GeneralPurpose performance mode
- 128 MiB/s provisioned throughput

#### 3.4 Provisioned + MaxIO + ProvisionedThroughput
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-provisioned-maxio"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Provisioned",
        "PerformanceMode": "MaxIO",
        "ProvisionedThroughput": "256",
    },
}
```
**예상 결과**: 
- ✅ 성공
- Provisioned throughput mode
- MaxIO performance mode
- 256 MiB/s provisioned throughput

#### 3.5 One Zone + Provisioned + GeneralPurpose
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-zone-provisioned"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.ZoneType,
    Zone: "ap-northeast-2a",
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Provisioned",
        "PerformanceMode": "GeneralPurpose",
        "ProvisionedThroughput": "64",
    },
}
```
**예상 결과**: 
- ✅ 성공
- One Zone EFS
- Provisioned throughput mode
- GeneralPurpose performance mode
- 64 MiB/s provisioned throughput

### 4. 성능 설정 오류 케이스

#### 4.1 Elastic + MaxIO (지원 안됨)
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-elastic-maxio-error"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Elastic",
        "PerformanceMode": "MaxIO",
    },
}
```
**예상 결과**: 
- ❌ 실패
- 에러: "MaxIO performance mode is not supported with Elastic throughput mode"

#### 4.2 One Zone + MaxIO (지원 안됨)
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-zone-maxio-error"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.ZoneType,
    Zone: "ap-northeast-2a",
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Bursting",
        "PerformanceMode": "MaxIO",
    },
}
```
**예상 결과**: 
- ❌ 실패
- 에러: "MaxIO performance mode is not supported for One Zone EFS. Please use GeneralPurpose performance mode"

#### 4.3 Provisioned + ProvisionedThroughput 누락
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-provisioned-no-throughput"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Provisioned",
        "PerformanceMode": "GeneralPurpose",
    },
}
```
**예상 결과**: 
- ❌ 실패
- 에러: "ProvisionedThroughput is required when ThroughputMode is Provisioned"

#### 4.4 ProvisionedThroughput 범위 초과
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-throughput-range-error"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Provisioned",
        "PerformanceMode": "GeneralPurpose",
        "ProvisionedThroughput": "2048", // 1024 초과
    },
}
```
**예상 결과**: 
- ❌ 실패
- 에러: "provisioned throughput must be between 1 and 1024 MiB/s"

#### 4.5 필수 필드 누락
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-missing-required"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Elastic",
        // PerformanceMode 누락
    },
}
```
**예상 결과**: 
- ❌ 실패
- 에러: "required field 'PerformanceMode' is missing in PerformanceInfo"

### 5. NFS 버전 테스트

#### 5.1 NFS 4.0 지정
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-nfs40"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    NFSVersion: "4.0",
}
```
**예상 결과**: 
- ✅ 성공
- NFS 4.0 지원 확인
- 로그: "Requested NFS version: 4.0 (AWS EFS will use 4.1 for file system, but 4.0 can be used for mounting)"

#### 5.2 지원되지 않는 NFS 버전
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-nfs30-error"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    NFSVersion: "3.0",
}
```
**예상 결과**: 
- ❌ 실패
- 에러: "AWS EFS only supports NFS versions: [4.0 4.1]"

### 6. 암호화 설정 테스트

#### 6.1 암호화 비활성화
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-no-encryption"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    Encryption: false,
}
```
**예상 결과**: 
- ✅ 성공
- 로그: "User requested no encryption - creating unencrypted file system"
- 암호화되지 않은 EFS 생성

#### 6.2 암호화 활성화
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-with-encryption"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    Encryption: true,
}
```
**예상 결과**: 
- ✅ 성공
- 로그: "User requested encryption - enabling with default AWS EFS KMS key"
- 기본 KMS 키로 암호화된 EFS 생성

### 7. 마운트 타겟 생성 테스트

**마운트 타겟 생성 전략 우선순위:**
1. **AccessSubnetList** (cb-spider 공식 기능): 서브넷만 지정하여 기본 보안 그룹으로 마운트 타겟 생성
2. **MountTargetList** (옵션 기능): 서브넷과 보안 그룹을 함께 지정하여 세밀한 제어 가능
3. **기본 동작**: 마운트 타겟 정보가 없으면 AWS 콘솔과 동일하게 모든 가용 AZ에 자동 생성

#### 7.1 AccessSubnetList 사용 (기본 보안 그룹) - cb-spider 공식 기능
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-access-subnets"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
    AccessSubnetList: []irs.IID{
        {SystemId: "subnet-04bd8bcbeb8cf7748"},
        {SystemId: "subnet-08124f8bc6b14d6c9"},
    },
}
```
**예상 결과**: 
- ✅ 성공
- 2개의 마운트 타겟 생성
- 기본 보안 그룹 사용

#### 7.2 AccessSubnetList - One Zone 제약사항
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-zone-access-error"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.ZoneType,
    Zone: "ap-northeast-2a",
    AccessSubnetList: []irs.IID{
        {SystemId: "subnet-04bd8bcbeb8cf7748"},
        {SystemId: "subnet-08124f8bc6b14d6c9"},
    },
}
```
**예상 결과**: 
- ❌ 실패
- 에러: "One Zone EFS can only have 1 mount target, but 2 subnets were specified"

#### 7.3 MountTargetList 사용 (보안 그룹 지정) - 옵션 기능
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-mount-targets"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
    MountTargetList: []irs.MountTargetInfo{
        {
            SubnetIID: irs.IID{SystemId: "subnet-04bd8bcbeb8cf7748"},
            SecurityGroups: []string{"sg-0f5fdc13eef5c83c3"},
        },
        {
            SubnetIID: irs.IID{SystemId: "subnet-08124f8bc6b14d6c9"},
            SecurityGroups: []string{"sg-0c88474826a32fb4c"},
        },
    },
}
```
**예상 결과**: 
- ✅ 성공
- 2개의 마운트 타겟 생성
- 각각 지정된 보안 그룹 적용

#### 7.4 MountTargetList - One Zone 제약사항
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-zone-mount-error"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.ZoneType,
    Zone: "ap-northeast-2a",
    MountTargetList: []irs.MountTargetInfo{
        {
            SubnetIID: irs.IID{SystemId: "subnet-04bd8bcbeb8cf7748"},
            SecurityGroups: []string{"sg-0f5fdc13eef5c83c3"},
        },
        {
            SubnetIID: irs.IID{SystemId: "subnet-08124f8bc6b14d6c9"},
            SecurityGroups: []string{"sg-0c88474826a32fb4c"},
        },
    },
}
```
**예상 결과**: 
- ❌ 실패
- 에러: "One Zone EFS can only have 1 mount target, but 2 were specified"

#### 7.5 MountTargetList - 잘못된 Zone의 서브넷
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-wrong-zone-subnet"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.ZoneType,
    Zone: "ap-northeast-2a",
    MountTargetList: []irs.MountTargetInfo{
        {
            SubnetIID: irs.IID{SystemId: "subnet-08124f8bc6b14d6c9"}, // ap-northeast-2b
            SecurityGroups: []string{"sg-0f5fdc13eef5c83c3"},
        },
    },
}
```
**예상 결과**: 
- ❌ 실패
- 에러: "mount target subnet subnet-08124f8bc6b14d6c9 is not in the correct zone for One Zone EFS"

#### 7.6 마운트 타겟 정보 없음 (기본 동작)
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-default-mount"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
}
```
**예상 결과**: 
- ✅ 성공
- AWS 콘솔 기본 동작으로 마운트 타겟 생성
- VPC의 각 AZ에 자동으로 마운트 타겟 생성

### 8. 복합 시나리오 테스트

#### 8.1 완전한 고급 설정
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-complete-advanced"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
    Zone: "ap-northeast-2a",
    Encryption: true,
    NFSVersion: "4.1",
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Provisioned",
        "PerformanceMode": "MaxIO",
        "ProvisionedThroughput": "512",
    },
    TagList: []irs.KeyValue{
        {Key: "Environment", Value: "Production"},
        {Key: "CostCenter", Value: "IT-001"},
    },
    MountTargetList: []irs.MountTargetInfo{ // 보안 그룹 지정을 위한 옵션 기능
        {
            SubnetIID: irs.IID{SystemId: "subnet-04bd8bcbeb8cf7748"},
            SecurityGroups: []string{"sg-0f5fdc13eef5c83c3", "sg-0c88474826a32fb4c"},
        },
        {
            SubnetIID: irs.IID{SystemId: "subnet-08124f8bc6b14d6c9"},
            SecurityGroups: []string{"sg-0f5fdc13eef5c83c3"},
        },
    },
}
```
**예상 결과**: 
- ✅ 성공
- Multi-AZ EFS 생성
- Provisioned throughput (512 MiB/s)
- MaxIO performance mode
- 암호화 활성화
- 사용자 태그 + Name 태그
- 2개의 마운트 타겟 (각각 다른 보안 그룹 설정)

#### 8.2 One Zone 완전 설정
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-onezone-complete"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.ZoneType,
    Zone: "ap-northeast-2a",
    Encryption: true,
    NFSVersion: "4.1",
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Provisioned",
        "PerformanceMode": "GeneralPurpose", // One Zone에서는 MaxIO 사용 불가
        "ProvisionedThroughput": "128",
    },
    TagList: []irs.KeyValue{
        {Key: "Environment", Value: "Development"},
        {Key: "Backup", Value: "Daily"},
    },
    MountTargetList: []irs.MountTargetInfo{ // 보안 그룹 지정을 위한 옵션 기능
        {
            SubnetIID: irs.IID{SystemId: "subnet-04bd8bcbeb8cf7748"},
            SecurityGroups: []string{"sg-0f5fdc13eef5c83c3"},
        },
    },
}
```
**예상 결과**: 
- ✅ 성공
- One Zone EFS 생성 (ap-northeast-2a)
- Provisioned throughput (128 MiB/s)
- GeneralPurpose performance mode
- 암호화 활성화
- 사용자 태그 + Name 태그
- 1개의 마운트 타겟

### 9. 경계값 테스트

#### 9.1 최소 ProvisionedThroughput
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-min-throughput"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Provisioned",
        "PerformanceMode": "GeneralPurpose",
        "ProvisionedThroughput": "1",
    },
}
```
**예상 결과**: 
- ✅ 성공
- 1 MiB/s provisioned throughput

#### 9.2 최대 ProvisionedThroughput
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-max-throughput"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Provisioned",
        "PerformanceMode": "GeneralPurpose",
        "ProvisionedThroughput": "1024",
    },
}
```
**예상 결과**: 
- ✅ 성공
- 1024 MiB/s provisioned throughput

#### 9.3 최대 ProvisionedThroughput 초과
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "efs-throughput-overflow"},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
    FileSystemType: irs.RegionType,
    PerformanceInfo: map[string]string{
        "ThroughputMode": "Provisioned",
        "PerformanceMode": "GeneralPurpose",
        "ProvisionedThroughput": "1025",
    },
}
```
**예상 결과**: 
- ❌ 실패
- 에러: "provisioned throughput must be between 1 and 1024 MiB/s"

### 10. 특수 케이스 테스트

#### 10.1 빈 이름
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: ""},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
}
```
**예상 결과**: 
- ❌ 실패 (AWS API 레벨에서)
- AWS EFS는 CreationToken이 비어있으면 오류 발생

#### 10.2 매우 긴 이름 (128자)
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "a".repeat(128)},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
}
```
**예상 결과**: 
- ✅ 성공
- AWS EFS는 최대 128자 이름 지원

#### 10.3 매우 긴 이름 (129자)
```go
reqInfo := irs.FileSystemInfo{
    IId: irs.IID{NameId: "a".repeat(129)},
    VpcIID: irs.IID{SystemId: "vpc-0a48d45f6bc3a71da"},
}
```
**예상 결과**: 
- ❌ 실패 (AWS API 레벨에서)
- AWS EFS는 128자를 초과하는 이름을 지원하지 않음

## 테스트 실행 방법

### 1. 테스트 환경 준비
```bash
# AWS 자격 증명 설정
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_DEFAULT_REGION="ap-northeast-2"

# cb-spider 실행
cd cloud-control-manager/cloud-driver/drivers/aws/main
go run Test_Resources.go
```

### 2. 테스트 실행 순서
1. **기본 설정 모드 테스트** (1.1-1.4)
2. **고급 설정 모드 테스트** (2.1-2.3)
3. **성능 설정 테스트** (3.1-3.5)
4. **성능 설정 오류 케이스** (4.1-4.5)
5. **NFS 버전 테스트** (5.1-5.2)
6. **암호화 설정 테스트** (6.1-6.2)
7. **마운트 타겟 생성 테스트** (7.1-7.6)
   - 7.1: AccessSubnetList 사용 (공식 기능)
   - 7.2: AccessSubnetList One Zone 제약사항
   - 7.3: MountTargetList 사용 (보안 그룹 지정 옵션)
   - 7.4: MountTargetList One Zone 제약사항
   - 7.5: MountTargetList 잘못된 Zone 서브넷
   - 7.6: 기본 동작 (마운트 타겟 정보 없음)
8. **복합 시나리오 테스트** (8.1-8.2)
9. **경계값 테스트** (9.1-9.3)
10. **특수 케이스 테스트** (10.1-10.3)

### 3. 테스트 결과 검증
각 테스트 후 다음을 확인:
- **성공 케이스**: EFS가 정상적으로 생성되고 예상된 설정이 적용되었는지 확인
- **실패 케이스**: 예상된 에러 메시지가 정확히 반환되는지 확인
- **로그 확인**: 콘솔 로그에서 예상된 메시지가 출력되는지 확인

## 예상 결과 요약

| 시나리오 카테고리 | 성공 | 실패 | 비고 |
|------------------|------|------|------|
| 기본 설정 모드 | 3 | 1 | VPC 필수 검증 |
| 고급 설정 모드 | 3 | 0 | - |
| 성능 설정 | 5 | 0 | - |
| 성능 설정 오류 | 0 | 6 | 제약사항 검증 |
| NFS 버전 | 1 | 1 | 지원 버전 검증 |
| 암호화 설정 | 2 | 0 | - |
| 마운트 타겟 | 3 | 3 | One Zone 제약사항 |
| 복합 시나리오 | 2 | 0 | - |
| 경계값 | 2 | 1 | 범위 검증 |
| 특수 케이스 | 1 | 2 | AWS 제한사항 |

**총계**: 21개 성공, 14개 실패 (예상된 실패 포함)

이 테스트 시나리오를 통해 AWS EFS FileSystemHandler의 CreateFileSystem 메서드가 모든 경우에 대해 정상적으로 동작하는지 검증할 수 있습니다. 