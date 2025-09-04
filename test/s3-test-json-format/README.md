# CB-Spider S3 API JSON Format Testing

이 디렉토리에는 CB-Spider S3 API의 JSON 입출력 형식을 테스트하는 스크립트들이 포함되어 있습니다.

## 개요

CB-Spider S3 REST API는 이제 XML과 JSON 두 가지 응답 형식을 지원합니다:
- **XML 형식**: 기존 AWS S3 호환 XML 응답 (기본값)
- **JSON 형식**: 모던 웹 애플리케이션을 위한 JSON 응답 (Accept: application/json 헤더 필요)

## 응답 형식 선택 방법

### JSON 응답 요청
다음 중 하나의 방법으로 JSON 응답을 요청할 수 있습니다:

1. **Accept 헤더**: `Accept: application/json`
2. **Query 파라미터**: `?format=json`
3. **Content-Type 헤더**: `Content-Type: application/json` (POST/PUT 요청)

### XML 응답 요청 (기본값)
- 별도 헤더나 파라미터 없이 요청하면 XML 응답
- 명시적으로 `Accept: application/xml` 헤더 사용

## 테스트 스크립트

각 CSP별로 JSON 형식 테스트 스크립트가 제공됩니다:

- `aws-s3-full-api-test.sh` - AWS S3 JSON 테스트
- `alibaba-s3-full-api-test.sh` - Alibaba Cloud Object Storage JSON 테스트  
- `gcp.s3-full-api-test.sh` - Google Cloud Storage JSON 테스트
- `ibm.s3-full-api-test.sh` - IBM Cloud Object Storage JSON 테스트
- `kt.s3-full-api-test.sh` - KT Cloud Object Storage JSON 테스트
- `ncp-s3-full-api-test.sh` - Naver Cloud Platform Object Storage JSON 테스트
- `nhn-s3-full-api-test.sh` - NHN Cloud Object Storage JSON 테스트
- `simple-json-test.sh` - 간단한 JSON 형식 검증 테스트

## 실행 방법

### 1. 간단한 JSON 형식 검증
```bash
cd /home/powerkim/powerkim/cb-spider/test/s3-test-json
./simple-json-test.sh
```

### 2. 전체 API JSON 테스트 (AWS 예시)
```bash
cd /home/powerkim/powerkim/cb-spider/test/s3-test-json
./aws-s3-full-api-test.sh
```

## 테스트되는 30개 S3 API

### 1. 버킷 관리 (6개)
- List Buckets
- Create Bucket  
- Get Bucket Info
- Head Bucket (Bucket 존재 확인)
- Get Bucket Location
- Delete Bucket

### 2. 객체 관리 (6개)
- Upload Object (File)
- Upload Object (Form)
- Download Object
- Head Object (Object 정보 확인)
- Delete Object
- Delete Multiple Objects

### 3. 멀티파트 업로드 (6개)
- Initiate Multipart Upload
- Upload Part
- Complete Multipart Upload
- Abort Multipart Upload
- List Parts
- List Multipart Uploads

### 4. 버전 관리 (4개)
- Get Bucket Versioning
- Set Bucket Versioning
- List Object Versions
- Delete Versioned Object

### 5. CORS 관리 (4개)
- Get Bucket CORS
- Set Bucket CORS
- Test CORS with OPTIONS
- Delete CORS Configuration

### 6. CB-Spider 특별 기능 (4개)
- Generate PreSigned URL (Download)
- PreSigned URL Download Test
- Generate PreSigned URL (Upload)
- PreSigned URL Upload Test

## JSON vs XML 응답 예시

### JSON 응답 예시
```bash
curl -H "Accept: application/json" -X GET "http://localhost:1024/spider/s3?ConnectionName=aws-config01"
```
```json
{
  "Owner": {
    "ID": "aws-config01",
    "DisplayName": "aws-config01"
  },
  "Buckets": {
    "Bucket": [
      {
        "Name": "my-test-bucket",
        "CreationDate": "2025-09-05T00:00:00Z"
      }
    ]
  }
}
```

### XML 응답 예시
```bash
curl -X GET "http://localhost:1024/spider/s3?ConnectionName=aws-config01"
```
```xml
<?xml version="1.0" encoding="UTF-8"?>
<ListAllMyBucketsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Owner>
    <ID>aws-config01</ID>
    <DisplayName>aws-config01</DisplayName>
  </Owner>
  <Buckets>
    <Bucket>
      <Name>my-test-bucket</Name>
      <CreationDate>2025-09-05T00:00:00Z</CreationDate>
    </Bucket>
  </Buckets>
</ListAllMyBucketsResult>
```

## 주의사항

1. **연결 설정**: 각 CSP별 connection 설정이 사전에 구성되어 있어야 합니다.
2. **권한**: S3/Object Storage 서비스에 대한 적절한 권한이 필요합니다.
3. **호환성**: JSON 응답은 XML 응답과 동일한 데이터를 포함하되, JSON 형식으로 제공됩니다.
4. **에러 응답**: 에러 응답도 요청된 형식(JSON/XML)에 따라 제공됩니다.

## 관련 파일

- **기존 테스트**: `../s3-test/` - 기존 XML 형식 테스트 스크립트들
- **S3 API 구현**: `../../api-runtime/rest-runtime/S3Rest.go` - Dual format 지원 구현
