# CB-Spider S3 API Test (JSON Format)

Automated test suite for CB-Spider S3 API endpoints with JSON response format.

## Prerequisites

### Basic Auth

Tests authenticate to CB-Spider with HTTP Basic Auth. Set these before running (defaults shown):

```bash
export SPIDER_USERNAME=admin   # default: admin
export SPIDER_PASSWORD=<your-password>
```

### CSP Connection Configuration

Before running tests, register connection names for each CSP in CB-Spider.

| CSP | Connection Name |
|-----|----------------|
| AWS | `aws-config01` |
| GCP | `gcp-iowa-config` |
| Azure | `azure-northeu-config` |
| Alibaba | `alibaba-tokyo-config` |
| Tencent | `tencent-tokyo-config` |
| IBM | `ibm-us-south-1-config` |
| OpenStack | `openstack-config01` |
| NCP | `ncp-korea1-config` |
| NHN | `nhn-korea-pangyo1-config` |
| KT | `kt-mokdong1-config` |

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

- **Azure**: 18 test cases
  - 6 Bucket + 6 Object + 0 Multipart + 0 Versioning + 0 CORS + 6 CB-Spider Special
  - Excluded: Multipart Upload (6 tests), Versioning (4 tests), CORS (4 tests)
  - Azure Blob Storage's S3-compatible layer does not support Versioning, CORS, Multipart Upload, or Delete Marker
  - Uses `common-s3-api-test-except-versioning-cors.sh` with `SKIP_MULTIPART=true`, and adds the `x-ms-blob-type: BlockBlob` header required for PreSigned uploads via `PRESIGNED_UPLOAD_EXTRA_HEADER`

## How to Run Tests

### Run All CSP Tests

Execute all CSP tests sequentially with summary report:

```bash
./run-all-csp-tests.sh
```

This will:
- Test all 10 CSPs (AWS, GCP, Azure, Alibaba, Tencent, IBM, OpenStack, NCP, NHN, KT)
- Display detailed results for each CSP
- Generate a comprehensive summary table

### Run Individual CSP Test

Execute test for a specific CSP:

```bash
# AWS
./aws-test.sh

# GCP
./gcp-test.sh

# Azure
./azure-test.sh

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

- `common-s3-full-api-test.sh`: Full API test — Bucket(6) + Object(6) + Multipart(6) + Versioning(4) + CORS(4) + Special(6) = 32 tests
  - Used by: AWS, GCP, Alibaba, Tencent, IBM, KT
- `common-s3-api-test-except-multipart-versioning.sh`: Skips Multipart Upload & Versioning — Bucket(6) + Object(6) + CORS(4) + Special(6) = 22 tests
  - Used by: OpenStack
- `common-s3-api-test-except-versioning-cors.sh`: Skips Versioning & CORS — Bucket(6) + Object(6) + Multipart(6) + Special(6) = 24 tests
  - Used by: NCP, NHN
  - Supports `SKIP_MULTIPART=true` to additionally skip Multipart Upload (used by Azure, 18 tests)

### Common Environment Variables

| Variable | Purpose | Used by |
|----------|---------|---------|
| `CONNECTION_NAME` | CB-Spider connection name for the target CSP | all |
| `SPIDER_USERNAME` / `SPIDER_PASSWORD` | Basic Auth credentials for the CB-Spider API | all |
| `SKIP_MULTIPART` | Skip Multipart Upload tests within `common-s3-api-test-except-versioning-cors.sh` | Azure |
| `PRESIGNED_UPLOAD_EXTRA_HEADER` | Extra header appended to PreSigned upload `curl` calls | Azure |

## Notes

- Tests use bucket name pattern: `cb-spider-test-json-{timestamp}`
- JSON format responses are validated using `Accept: application/json` header
- Unsupported APIs return HTTP 501 (Not Implemented)
