# CB-Spider S3 API Test (SigV4 / awscurl)

Automated test suite for CB-Spider S3 API endpoints using **AWS Signature Version 4 (SigV4)** authentication via `awscurl`.  
This is the SigV4 variant of the `../s3-test-xml-format/` Basic-Auth suite.

## Prerequisites

### 1. Install awscurl

```bash
pip install awscurl
```

> `awscurl` is a drop-in replacement for `curl` that computes and injects AWS SigV4 signatures.  
> See: https://github.com/okigan/awscurl

### 2. Set environment variables

| Variable | Description | Example |
|---|---|---|
| `SPIDER_USERNAME` | CB-Spider login username | `admin` |
| `SPIDER_PASSWORD` | CB-Spider login password | `your-password` |
| `CONNECTION_NAME` | CB-Spider connection config name | `aws-config01` |

```bash
export SPIDER_USERNAME="admin"
export SPIDER_PASSWORD="your-password"
# CONNECTION_NAME is set per-CSP by each wrapper script
```

> **Access Key format**: `awscurl` is called with  
> `--access_key "${SPIDER_USERNAME}@${CONNECTION_NAME}" --secret_key "${SPIDER_PASSWORD}"`  
> CB-Spider's S3 authentication layer decodes this format to identify both the user and the target connection.

### 3. Register connection configurations in CB-Spider

Register connection names for each CSP before running tests.

| CSP | Connection Name |
|-----|----------------|
| AWS | `aws-config01` |
| GCP | `gcp-iowa-config` |
| Alibaba | `alibaba-tokyo-config` |
| Tencent | `tencent-tokyo-config` |
| IBM | `ibm-us-south-1-config` |
| OpenStack | `openstack-config01` |
| NCP | `ncp-korea1-config` |
| NHN | `nhn-korea-pangyo1-config` |
| KT | `kt-mokdong1-config` |

### 4. Start the CB-Spider server

```bash
cd /path/to/cb-spider
./bin/start.sh
```

---

## CSP Test Coverage

### Full Test Suite (30 tests)

Used for: **AWS, GCP, Alibaba, Tencent, IBM, KT**

| Category | Tests | Description |
|---|---|---|
| Bucket Management | 6 | List, Create, Get, HEAD, Location, Delete |
| Object Management | 6 | Upload (file), Upload (form), Download, HEAD, Delete, DeleteMultiple |
| Multipart Upload | 6 | Initiate, UploadPart, ListParts, Abort, Complete, ListUploads |
| Versioning | 4 | Get, Set, ListVersions, DeleteVersioned |
| CORS | 4 | Set, Get, OPTIONS, Delete |
| CB-Spider Special | 6 | PreSigned Download/Upload + Force Empty/Delete |

### Partial Test Suites

**OpenStack** (`common-s3-api-test-except-multipart-versioning.sh`) — 20 tests  
- Skipped: Multipart Upload (6) + Versioning (4)

**NCP, NHN** (`common-s3-api-test-except-versioning-cors.sh`) — 22 tests  
- Skipped: Versioning (4) + CORS (4)

---

## Auth Comparison: SigV4 vs Basic Auth

| Feature | Basic Auth (`s3-test-xml-format/`) | SigV4 awscurl (`s3-test-xml-format-awscurl/`) |
|---|---|---|
| Credential transport | Username:Password in every request | HMAC signature only; secret never sent |
| Replay protection | None | 15-minute request window |
| Body integrity | None | `x-amz-content-sha256` hash |
| `ConnectionName` binding | `?ConnectionName=` query parameter | Encoded in access key: `user@conn` |
| Multipart form upload | Native `-F` flag | Falls back to `curl -u` Basic Auth |
| CORS preflight (OPTIONS) | Plain `curl` (no auth) | Plain `curl` (no auth) |

---

## How to Run

### Run all CSPs sequentially

```bash
chmod +x run-all-csp-tests.sh
./run-all-csp-tests.sh
```

Runs all 9 CSPs and prints a consolidated pass/fail summary table.

### Run a single CSP

```bash
./aws-test.sh
./gcp-test.sh
./alibaba-test.sh
./tencent-test.sh
./ibm-test.sh
./openstack-test.sh
./ncp-test.sh
./nhn-test.sh
./kt-test.sh
```

### Override connection name at runtime

```bash
CONNECTION_NAME="my-custom-config" ./common-s3-full-api-test.sh
```

---

## Script Overview

| Script | Purpose |
|---|---|
| `common-s3-full-api-test.sh` | Full 30-test suite |
| `common-s3-api-test-except-multipart-versioning.sh` | 20-test suite (no Multipart / Versioning) |
| `common-s3-api-test-except-versioning-cors.sh` | 22-test suite (no Versioning / CORS) |
| `aws-test.sh` | AWS wrapper → full suite |
| `gcp-test.sh` | GCP wrapper → full suite |
| `alibaba-test.sh` | Alibaba wrapper → full suite |
| `tencent-test.sh` | Tencent wrapper → full suite |
| `ibm-test.sh` | IBM wrapper → full suite |
| `openstack-test.sh` | OpenStack wrapper → no Multipart/Versioning |
| `ncp-test.sh` | NCP wrapper → no Versioning/CORS |
| `nhn-test.sh` | NHN wrapper → no Versioning/CORS |
| `kt-test.sh` | KT wrapper → full suite |
| `run-all-csp-tests.sh` | Runs all CSPs and prints aggregate summary |

---

## Notes

- Test bucket name pattern: `cb-spider-test-sigv4-{timestamp}`
- XML response format is validated via pattern matching
- `awscurl` does not require `--region`; CB-Spider ignores region in SigV4 signing and routes via `ConnectionName`
- Multipart form upload (`-F`) falls back to `curl -u Basic` because `awscurl` does not support `multipart/form-data`
- CORS preflight (`OPTIONS`) uses plain `curl` — standard CORS preflight requires no authentication
- If a test fails unexpectedly, verify `SPIDER_USERNAME`, `SPIDER_PASSWORD`, and `CONNECTION_NAME` are correct
