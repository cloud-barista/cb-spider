# CB-SpiderWatch

**CB-SpiderWatch** is a web-based automated test and monitoring system for [CB-Spider](https://github.com/cloud-barista/cb-spider), the multi-cloud connection layer of the Cloud-Barista project.
It periodically runs tests against the latest CB-Spider and presents the results in real time.
Selective testing against a specific CB-Spider version, a subset of CSPs, or individual resource types is also supported.
Test results reveal the operational status of Spider's core functions and the availability of CSP resources managed through Spider.

CB-SpiderWatch consists of two services:

- **Spider Watch Server** — controls Spider tests across multi-cloud resources and delivers live results through an admin web UI
- **Spider Status Board** — a public, read-only web service that publishes completed test results to the open internet

> **Live Status Board:** [http://spider-statusboard.cloud-barista.org:4096](http://spider-statusboard.cloud-barista.org:4096)

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://golang.org/)

---

## Features

- Automatic scheduled tests (default: 01:00 KST, configurable cron expression)
- Covers **10 CSPs**: AWS, Azure, GCP, Alibaba, Tencent, IBM, OpenStack, NCP, NHN, KT
- Tests **9 resource types** per CSP: VPC, Security Group, Key Pair, VM, Disk, NLB, My Image, Cluster, S3
- Per-resource OK / FAIL / SKIP status with operation-level detail (create / list / get / delete)
- Dark-themed live web dashboard with real-time progress during a run
- Hot-reload of `conf/spiderwatch.yaml` without restart
- REST API for programmatic access
- Manual **Run Now** and **Cleanup Only** triggers from the web UI or API
- **Stop Run** button to abort an in-progress test
- **GitHub Issue** integration — file FAIL reports directly from the UI
- Spider lifecycle management from the UI (Start / Stop Spider container)
- Per-CSP and per-resource enable/disable from the UI (hot-reloaded)
- Run history with multi-select delete
- Support for external Spider server (`external_url`) — skips Docker management
- **Spider Status Board** — public read-only service receiving results via push after each run

---

## Port Convention

| Service | Default Port |
|---|---|
| CB-Spider | 1024 |
| SpiderWatch | 2048 |
| Spider Status Board | 4096 |

---

## Getting Started

### Prerequisites

| Requirement | Version |
|---|---|
| Go | ≥ 1.25 |
| Docker | Engine + CLI (not required when using `external_url`) |

### Build & Run (SpiderWatch)

```bash
# Clone the repository
git clone https://github.com/cloud-barista/cb-spider.git
cd cb-spider/spiderwatch

# Build
make build

# Edit the configuration
vi conf/spiderwatch.yaml

# Run (background, logs to terminal + logs/spiderwatch.log)
make run

# Stop
make stop
```

Open **http://localhost:2048** in your browser.

---

## Spider Status Board

The **Spider Status Board** is a lightweight read-only web service that displays the latest CB-Spider test results to the public. SpiderWatch pushes the completed result to the Status Board after each run via an authenticated API call.

### Architecture

```
[SpiderWatch (private)]  ──POST /api/v1/push──▶  [Spider Status Board (public)]
  localhost:2048                                     0.0.0.0:4096
  conf/spiderwatch.yaml                              conf/statusboard.yaml
```

### Building & Running (from source)

```bash
# Build
make sb-build

# Edit configuration
vi conf/statusboard.yaml

# Run (background, logs to statusboard.log)
make sb-run

# Stop
make sb-stop
```

Open **http://localhost:4096** in your browser.

### Creating Distribution Packages

`make sb-dist` cross-compiles for all supported platforms and creates self-contained archives under `dist/`:

```bash
make sb-dist
```

Output:
```
dist/statusboard-<VERSION>-linux-amd64.tar.gz
dist/statusboard-<VERSION>-darwin-amd64.tar.gz
dist/statusboard-<VERSION>-darwin-arm64.tar.gz
```

Each archive extracts to a `spider-statusboard/` directory and is fully self-contained (binary + web assets + configuration).

### Deploying on a Remote Server (e.g., Ubuntu/AWS EC2)

```bash
# On the build machine — generate the Linux package
make sb-dist

# Transfer to the target server
scp dist/statusboard-<VERSION>-linux-amd64.tar.gz ubuntu@<SERVER_IP>:~/

# On the target server
tar -xzf statusboard-<VERSION>-linux-amd64.tar.gz
cd spider-statusboard

# Edit configuration (set auth token, adjust port if needed)
vi conf/statusboard.yaml

# Run
make run

# Stop
make stop
```

### Status Board Configuration (`conf/statusboard.yaml`)

```yaml
server:
  port: 4096            # Web UI / API port
  address: "0.0.0.0"

auth:
  token: "****"  # Must match statusboard.token in spiderwatch.yaml

log:
  level: "info"
  file: "logs/statusboard.log"  # leave empty to log to stdout only
```

### Connecting SpiderWatch to the Status Board

In `conf/spiderwatch.yaml`:

```yaml
statusboard:
  url: "http://<STATUS_BOARD_HOST>:4096"   # Status Board server URL
  token: "****"    # Must match statusboard.yaml auth.token
```

SpiderWatch pushes the completed `RunResult` JSON to `POST <url>/api/v1/push` with `Authorization: Bearer <token>` after each test run. The push is non-blocking (goroutine) and does not affect the test run itself.

### Status Board API

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/summary` | Dashboard summary (latest run) |
| `GET` | `/api/v1/runs` | All stored run results |
| `GET` | `/api/v1/runs/latest` | Most recent run result |
| `GET` | `/api/v1/runs/:id` | Specific run result by ID |
| `GET` | `/api/v1/status` | Service health check |
| `POST` | `/api/v1/push` | Receive a run result from SpiderWatch (Bearer token required) |

---

## SpiderWatch Configuration

Edit `conf/spiderwatch.yaml`. All changes are hot-reloaded (no restart needed).

```yaml
# ── GitHub Issue Reporting ──────────────────────────────────────────────────
github:
  token: "****"          # GitHub personal access token (needs repo scope)
  owner: "cloud-barista"
  repo: "cb-spider"
  labels:
    - "spiderwatch"
    - "bug"

# ── Web Server ──────────────────────────────────────────────────────────────
server:
  port: 2048
  address: "0.0.0.0"

# ── Spider Connection ────────────────────────────────────────────────────────
spider:
  # (Optional) URL of an already-running Spider server.
  # When set, Docker pull/run is skipped.
  external_url: ""       # e.g. "http://localhost:1024/spider"

  image: "cloudbaristaorg/cb-spider:edge"
  host_port: 1024
  api_url: "http://localhost:1024/spider"
  username: "admin"
  password: ""
  meta_db_dir: "${HOME}/cb-spider/meta_db"
  server_address: "0.0.0.0:1024"
  api_timeout_sec: 1800
  startup_wait_sec: 60
  run_timeout_min: 120

# ── Scheduler ───────────────────────────────────────────────────────────────
scheduler:
  cron: "0 0 1 * * *"   # 6-field cron: second minute hour day month weekday
  run_on_startup: false

# ── Resources ───────────────────────────────────────────────────────────────
resources:
  - vpc
  - securitygroup
  - keypair
  - vm
  - disk
  - nlb
  - myimage
  - cluster
  - s3

# ── Cleanup ─────────────────────────────────────────────────────────────────
# true  – delete created resources after tests (default)
# false – leave resources in the CSP
resources
cleanup: true

# ── Logging ─────────────────────────────────────────────────────────────────
log:
  level: "info"
  file: "logs/spiderwatch.log"

# ── Status Board Push ────────────────────────────────────────────────────────
statusboard:
  url: "http://spider-statusboard.cloud-barista.org:4096"                # e.g. "http://spider-statusboard.example.com:4096"
  token: "****"          # Shared secret matching statusboard.yaml auth.token

# ── CSPs ─────────────────────────────────────────────────────────────────────
csps:
  - name: AWS
    connection: aws-config01
    enabled: true
    vm_test:
      image_name: ami-0131a0fdbb6fda7e6
      spec_name: t2.micro
    # ... see conf/spiderwatch.yaml for full example
```

### CSP Connection Names

| CSP | Default Connection |
|---|---|
| AWS | `aws-config01` |
| Azure | `azure-northeu-config` |
| GCP | `gcp-iowa-config` |
| Alibaba | `alibaba-tokyo-config` |
| Tencent | `tencent-beijing3-config` |
| IBM | `ibm-us-east-1-config` |
| OpenStack | `openstack-config01` |
| NCP | `ncp-korea1-config` |
| NHN | `nhn-korea-pangyo1-config` |
| KT | `kt-mokdong1-config` |

### Supported Resource Types

| Resource | Description |
|---|---|
| `vpc` | VPC + Subnet CRUD |
| `securitygroup` | Security Group CRUD |
| `keypair` | Key Pair CRUD |
| `vm` | VM CRUD (create / list / get / delete) |
| `disk` | Disk CRUD |
| `nlb` | Network Load Balancer CRUD |
| `myimage` | VM Image (My Image) CRUD |
| `cluster` | Kubernetes Cluster + NodeGroup CRUD |
| `s3` | Object Storage Bucket CRUD (bucket name rotates per run to avoid reuse delays) |

Resources not implemented for a specific CSP are automatically marked **SKIP**.

---

## SpiderWatch REST API

### Run Management

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/summary` | Dashboard summary (latest run + next scheduled time) |
| `GET` | `/api/v1/runs` | All run results |
| `GET` | `/api/v1/runs/latest` | Most recent run result |
| `GET` | `/api/v1/runs/:id` | Specific run result by ID |
| `POST` | `/api/v1/runs/trigger` | Trigger an immediate full test run |
| `POST` | `/api/v1/runs/cleanup` | Trigger a cleanup-only run (no tests) |
| `POST` | `/api/v1/runs/stop` | Stop the in-progress run |
| `DELETE` | `/api/v1/runs/:id` | Delete a stored run result |

### Spider Management

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/spider/status` | Spider container running status |
| `POST` | `/api/v1/spider/start` | Start the Spider container |
| `POST` | `/api/v1/spider/stop` | Stop the Spider container |

### Configuration (hot-reload)

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/config/resources` | Get enabled resource types |
| `PUT` | `/api/v1/config/resources` | Update enabled resource types |
| `GET` | `/api/v1/config/csps` | Get CSP enable/disable state |
| `PUT` | `/api/v1/config/csps` | Update CSP enable/disable state |
| `GET` | `/api/v1/config/cleanup` | Get cleanup setting |
| `PUT` | `/api/v1/config/cleanup` | Update cleanup setting |

### GitHub Issue

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/runs/:id/issue-draft` | Get pre-filled issue title and body for a FAIL |
| `POST` | `/api/v1/runs/:id/issue` | Create a GitHub issue and save the link to the run |

---

## Project Structure

```
cb-spider/spiderwatch/
├── cmd/
│   ├── spiderwatch/      # SpiderWatch binary entry point
│   └── statusboard/      # Spider Status Board binary entry point
├── conf/
│   ├── spiderwatch.yaml  # SpiderWatch configuration
│   └── statusboard.yaml  # Status Board configuration
├── data/results/         # Stored run results (JSON)
├── internal/
│   ├── config/           # SpiderWatch config loading + hot-reload
│   ├── model/            # Shared data types (RunResult, CSPResult, …)
│   ├── runner/           # Docker lifecycle + Spider API test runner
│   ├── statusboard/      # Status Board config loader + Echo HTTP server
│   ├── store/            # JSON file store for run results (shared)
│   └── web/              # SpiderWatch Echo server, handlers, template renderer (shared)
├── web/
│   ├── static/           # CSS, JS, images
│   └── templates/        # HTML templates (shared by both services)
├── dist/                 # Generated distribution packages (sb-dist)
└── Makefile
```

### Makefile Targets

| Target | Description |
|---|---|
| `make build` | Build SpiderWatch binary |
| `make run` / `make start` | Build and run SpiderWatch in background |
| `make stop` | Stop SpiderWatch |
| `make sb-build` | Build Status Board binary (native) |
| `make sb-build-linux` | Cross-compile Status Board for Linux/amd64 |
| `make sb-build-darwin` | Cross-compile Status Board for macOS (amd64 + arm64) |
| `make sb-run` / `make sb-start` | Build and run Status Board in background |
| `make sb-stop` | Stop Status Board |
| `make sb-dist` | Build all platforms and create distribution archives |
| `make sb-dist-linux` | Build Linux/amd64 distribution archive only |
| `make tidy` | `go mod tidy && go mod verify` |
| `make lint` | `go vet ./...` |
| `make clean` | Remove build artifacts and dist/ |

---

## Roadmap

- **Failure notifications** — automatic alerts via Slack, email, or other channels when a scheduled test run produces errors

---

## License

Apache License 2.0 — see [LICENSE](LICENSE).

