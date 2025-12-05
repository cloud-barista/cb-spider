# CB-Spider S3 API Test (JSON Format)

Automated test suite for CB-Spider S3 API endpoints with JSON response format.

## Prerequisites

### CSP Connection Configuration

Before running tests, register connection names for each CSP in CB-Spider.

| CSP | Connection Name |
|-----|----------------|
| AWS | `aws-config01` |
| GCP | `gcp-iowa-config` |
| Alibaba | `alibaba-config` |
| Tencent | `tencent-config` |
| IBM | `ibm-config01` |
| OpenStack | `openstack-config01` |
| NCP | `ncp-config01` |
| NHN | `nhn-config01` |
| KT | `kt-config` |

## CSP Test Coverage

### Full Test Suite (32 Test Cases)
- **AWS, GCP, Alibaba, Tencent, IBM, KT**: All 32 test cases
  - 6 Bucket Management + 6 Object Management + 6 Multipart Upload + 4 Versioning + 4 CORS + 6 CB-Spider Special

### Partial Test Suite

> Note: Excludes tests for features that are not supported or unstable in specific CSPs.

- **OpenStack**: 22 test cases
  - 6 Bucket + 6 Object + 0 Multipart + 0 Versioning + 4 CORS + 6 CB-Spider Special
  - Excluded: Multipart Upload (6 tests), Versioning (4 tests)
  
- **NCP, NHN**: 24 test cases
  - 6 Bucket + 6 Object + 6 Multipart + 0 Versioning + 0 CORS + 6 CB-Spider Special
  - Excluded: Versioning (4 tests), CORS (4 tests)

## How to Run Tests

### Run All CSP Tests

Execute all CSP tests sequentially with summary report:

```bash
./run-all-csp-tests.sh
```

This will:
- Test all 9 CSPs (AWS, GCP, Alibaba, Tencent, IBM, OpenStack, NCP, NHN, KT)
- Display detailed results for each CSP
- Generate a comprehensive summary table

### Run Individual CSP Test

Execute test for a specific CSP:

```bash
# AWS
./aws-test.sh

# GCP
./gcp-test.sh

# Alibaba
./alibaba-test.sh

# Tencent
./tencent-test.sh

# IBM
./ibm-test.sh

# OpenStack
./openstack-test.sh

# NCP
./ncp-test.sh

# NHN
./nhn-test.sh

# KT
./kt-test.sh
```

## Test Scripts

- `common-s3-full-api-test.sh`: Full API test (32 APIs)
- `common-s3-api-test-except-multipart-versioning.sh`: Except Multipart & Versioning (21 APIs)
- `common-s3-api-test-except-versioning-cors.sh`: Except Versioning & CORS (23 APIs)

## Notes

- Tests use bucket name pattern: `cb-spider-test-json-{timestamp}`
- JSON format responses are validated using `Accept: application/json` header
- Unsupported APIs return HTTP 501 (Not Implemented)
