# CB-Spider PublicIP Test Scripts

## Overview

This directory contains curl-based REST API test scripts for CB-Spider PublicIP resource lifecycle testing across all 10 supported CSPs.

## Prerequisites

- CB-Spider server is running (default: `http://localhost:1024`)
- Connection configurations are registered for each CSP in CB-Spider
- Valid credentials are configured for each CSP

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SPIDER_URL` | CB-Spider server URL | `http://localhost:1024` |
| `SPIDER_AUTH` | Basic auth credentials (`user:password`) | `admin:****` |
| `CB_CONNECTION_NAME` | CSP connection config name (overrides per-CSP default) | CSP-specific |
| `CB_PUBLICIP_NAME` | Name for the test Public IP (overrides per-CSP default) | CSP-specific |

### SPIDER_AUTH

CB-Spider REST API uses HTTP Basic Authentication. Set `SPIDER_AUTH` to `username:password`:

```bash
# Use default (admin:****)
./aws-publicip-test.sh

# Custom credentials
SPIDER_AUTH="myuser:mypassword" ./aws-publicip-test.sh

# All CSPs with custom auth
SPIDER_AUTH="myuser:mypassword" ./run-all-csp-publicip-tests.sh
```

The default value `admin:****` matches the CB-Spider server's built-in default credentials.

## Test Files

| File | CSP | Default Connection Config |
|------|-----|---------------------------|
| `aws-publicip-test.sh` | AWS | `aws-config01` |
| `azure-publicip-test.sh` | Azure | `azure-koreacentral-config` |
| `gcp-publicip-test.sh` | GCP | `gcp-iowa-config` |
| `alibaba-publicip-test.sh` | Alibaba | `alibaba-beijing-config` |
| `tencent-publicip-test.sh` | Tencent | `tencent-beijing6-config` |
| `ibm-publicip-test.sh` | IBM Cloud | `ibm-us-east-1-config` |
| `openstack-publicip-test.sh` | OpenStack | `openstack-config01` |
| `ncp-publicip-test.sh` | NCP | `ncp-korea1-config` |
| `nhn-publicip-test.sh` | NHN Cloud | `nhn-korea-pangyo1-config` |
| `kt-publicip-test.sh` | KT Cloud | `kt-kr1-config` |


## Lifecycle Test Steps

Each test script performs the following steps in order:

1. **Create** — Allocate a new Public IP (`POST /publicip`)
2. **Get** — Get details of the created Public IP (`GET /publicip/{Name}`)
3. **List** — List all Public IPs (`GET /publicip`)
4. **Associate** — Attach to a NIC or VM (`PUT /publicip/{Name}/associate`) *(optional — runs only if `NIC_NAME` or `VM_NAME` is set)*
5. **Disassociate** — Detach from NIC/VM (`PUT /publicip/{Name}/disassociate`) *(optional — same condition)*
6. **Delete** — Release the Public IP (`DELETE /publicip/{Name}`)

Overall result is **PASS** only if Create, Get, and Delete all succeed.

## Running Tests

### Single CSP
```bash
./aws-publicip-test.sh
```

### Custom Connection or Credentials
```bash
CB_CONNECTION_NAME="my-aws-config" CB_PUBLICIP_NAME="my-test-eip" ./aws-publicip-test.sh

SPIDER_URL="http://192.168.1.100:1024" SPIDER_AUTH="admin:secret" ./aws-publicip-test.sh
```

### All CSPs (parallel, with summary table)
```bash
./run-all-csp-publicip-tests.sh
```

With verbose per-CSP logs:
```bash
VERBOSE=1 ./run-all-csp-publicip-tests.sh
```

With custom Spider endpoint and auth:
```bash
SPIDER_URL="http://192.168.1.100:1024" SPIDER_AUTH="admin:secret" ./run-all-csp-publicip-tests.sh
```

## Expected Output Summary

### Create
```json
{
  "IId": {"NameId": "spider-eip-01", "SystemId": "..."},
  "PublicIPAddress": "52.x.x.x",
  "Status": "Available",
  "CreatedTime": "...",
  "KeyValueList": [...]
}
```

### List
```json
{
  "publicip": [
    {"IId": {...}, "PublicIPAddress": "52.x.x.x", "Status": "Available", ...}
  ]
}
```

### Get
```json
{
  "IId": {"NameId": "spider-eip-01", "SystemId": "..."},
  "PublicIPAddress": "52.x.x.x",
  "Status": "Available",
  ...
}
```

### Delete
```json
{"Result": "true"}
```

### All-CSP Summary Table (run-all-csp-publicip-tests.sh)
```
=====================================================================...
                CB-Spider PublicIP Lifecycle Test Summary — All CSPs
=====================================================================...

CSP        | Overall  | IP Address         | Create  | Get   | List      | Assoc   | Dissoc   | Delete  | Elapsed
---------------------------------------------------------------------...
AWS        | PASS     | 52.x.x.x           | OK      | OK    | OK(3)     | -       | -        | OK      | 12s
AZURE      | PASS     | 20.x.x.x           | OK      | OK    | OK(1)     | -       | -        | OK      | 18s
GCP        | PASS     | 34.x.x.x           | OK      | OK    | OK(2)     | -       | -        | OK      | 15s
...
---------------------------------------------------------------------...
Total        PASS=8  FAIL=2
```
