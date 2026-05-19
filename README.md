<h1 align="center">CB-Spider : "One-Code, Multi-Cloud"</h1>

<h2 align="center">A unified framework for multi-cloud infrastructure control</h2>

<p align="center">
  <a href="https://github.com/cloud-barista/cb-spider/blob/master/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/cloud-barista/cb-spider?label=go.mod" alt="Go Version"></a>
  <a href="https://github.com/cloud-barista/cb-spider/blob/master/LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License"></a>
  <a href="https://github.com/cloud-barista/cb-spider/releases"><img src="https://img.shields.io/github/v/release/cloud-barista/cb-spider" alt="Release"></a>
  <a href="https://github.com/cloud-barista/cb-spider/wiki"><img src="https://img.shields.io/badge/docs-Wiki-green" alt="Docs"></a>
  <a href="https://cloud-barista.github.io/api/?url=https://raw.githubusercontent.com/cloud-barista/cb-spider/refs/heads/master/api/swagger.yaml"><img src="https://img.shields.io/badge/API-Swagger-blue" alt="Swagger"></a>
</p>

<p align="center">
  <img width="850" alt="CB-Spider Architecture" src="https://github.com/user-attachments/assets/c1e5328b-151d-4b24-ad62-947e8bfcbbcf">
</p>

---

* CB-Spider is a sub-framework of the [Cloud-Barista](https://github.com/cloud-barista) Multi-Cloud Platform. 
* CB-Spider provides a **single unified API** for multi-cloud infrastructure control.
* CB-Spider enables write once, run on any cloud.

<br>

```
[NOTE]
CB-Spider is currently under development and has not yet reached version 1.0.
We welcome suggestions, issues, feedback, and contributions!
Please be aware that the functionalities of Cloud-Barista are not yet stable or secure.
Exercise caution if you plan to use the current release in a production environment.
If you encounter any difficulties while using Cloud-Barista, please let us know.
(You can open an issue or join the Cloud-Barista Slack community.)
```

---

## Key Features

- **Unified API** — One consistent REST API for all supported CSPs
- **Multi-Cloud Abstraction** — VPC, VM, Disk, NLB, Kubernetes, Object Storage and more
- **Dynamic Plugin Drivers** — Extensible cloud driver architecture with hot-plugin
- **AdminWeb & CLI** — Built-in web console and `spctl` CLI tool
- **Swagger API Docs** — Auto-generated, always up-to-date API documentation

---

## Supported Cloud Providers

| Provider      | VM Price<br>Info | Region/Zone<br>Info | Image<br>Info | VMSpec<br>Info | VPC<br>Subnet       | Security<br>Group | VM KeyPair      | VM             | Disk | MyImage | NLB | K8S | Object<br> Storage |
|:-------------:|:-------------:|:-------------------:|:-------------:|:--------------:|:-------------------:|:-----------------:|:---------------:|:--------------:|:----:|:---:|:-------:|:-----------:|:-----------:|
| AWS           | O             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O        | O        |
| Azure         | O             | O                  | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O        | WIP        |
| GCP           | O             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O        | O        |
| Alibaba       | O             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O           | O        |
| Tencent       | O             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O           | O        |
| IBM           | O             | O                  | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O        | O        |
| OpenStack     | NA             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | TBD           | O        |
| NCP           | O            | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | https://github.com/cloud-barista/cb-spider/issues/1607           | O        |
| NHN           | NA             | O                   | O             | O              | O                 | O                 | O               | O<br>(Note1)   | O    | O    | O     | O           | O        |
| KT            | NA             | O                   | O             | O              | O<br>(Type1)       | O                 | O               | O              | O    | O   | O<br>(Note2)| TBD  | O        |
| KT Classic    | NA             | O                   | O             | O              | O<br>(Type2)       | O                 | O               | O              | O    | O   | O       | NA          | -        |

※ WIP: Work In Progress,  NA: Not Applicable,  -: Excluded (Classic Resource)  

<details>
<summary><b>Provider-specific Notes</b></summary>
<br>

**VPC Notes (see each driver's README for details):**  
- **Type1:** Default VPC (KT VPC)  
  - CSP: Provides one fixed default VPC only  
  - CB-Spider: Only one VPC can be created (for abstraction)  

- **Type2:** VPC/Subnet Emulation  
  - CSP: No VPC concept provided  
  - CB-Spider: Provides single emulated VPC/Subnet  

**VM Notes:**  
- **Note1:** For Windows VMs, after creating a VM with SSH key, the password must be checked in Console.  

**NLB Notes:**  
- **Note2:** VMs registered to NLB must be in the same Subnet as the NLB.  

</details>

---

## Quick Start

The [**Quick Start Guide**](https://github.com/cloud-barista/cb-spider/wiki/Quick-Start-Guide) walks you through the following steps:

1. **Start CB-Spider server** — Run with a single Docker command
2. **Set cloud credentials** — Register your AWS / GCP access keys
3. **Register cloud connections** — Configure driver, credential, region, and connection
4. **Create infrastructure** — Create VPC, Security Group, KeyPair, and VM with unified API calls
5. **Multi-cloud verification** — Query both clouds with the same API, only `ConnectionName` changes
6. **Cleanup** — Terminate VMs and delete resources

> **Start methods:** [Docker Guide](https://github.com/cloud-barista/cb-spider/wiki/Docker-based-Start-Guide) | [Source Guide](https://github.com/cloud-barista/cb-spider/wiki/Source-based-Start-Guide) | [Authentication Guide](https://github.com/cloud-barista/cb-spider/wiki/Authentication-Guide)

---

## Documentation

### Getting Started
| Guide | Description |
|:------|:------------|
| [Quick Start Guide](https://github.com/cloud-barista/cb-spider/wiki/Quick-Start-Guide) | Start server & create VMs on AWS + GCP in minutes |
| [Docker-based Start Guide](https://github.com/cloud-barista/cb-spider/wiki/Docker-based-Start-Guide) | Run CB-Spider using Docker |
| [Source-based Start Guide](https://github.com/cloud-barista/cb-spider/wiki/Source-based-Start-Guide) | Build and run from source |
| [How to get CSP Credentials](https://github.com/cloud-barista/cb-spider/wiki/How-to-get-CSP-Credentials) | Obtain credentials for each cloud provider |

### API & Tools
| Resource | Link |
|:---------|:-----|
| Swagger API Docs | [Swagger UI](https://cloud-barista.github.io/api/?url=https://raw.githubusercontent.com/cloud-barista/cb-spider/refs/heads/master/api/swagger.yaml) · [Swagger Guide](https://github.com/cloud-barista/cb-spider/wiki/Swagger-Guide) |
| AdminWeb | [AdminWeb Guide](https://github.com/cloud-barista/cb-spider/wiki/CB-Spider-AdminWeb-Guide) |
| CLI (`spctl`) | [CLI Tool Guide](https://github.com/cloud-barista/cb-spider/wiki/CLI-Tool-Install-Guide) |
| Driver Capability Info | [Capability Info Guide](https://github.com/cloud-barista/cb-spider/wiki/Cloud-Driver-Capability-Info-Guide) |
| Function Menu | [CB-Spider Menu](https://github.com/cloud-barista/cb-spider/wiki/CB-Spider-Menu) |
| MetaDB Auto Backup | [Backup Guide](https://github.com/cloud-barista/cb-spider/wiki/Meta-DB-Backup-Guide) |

### Resource Management Guides
| Resource | Guide |
|:---------|:------|
| Register Connection | [Register Connection Guide](https://github.com/cloud-barista/cb-spider/wiki/Register-Connection-Guide) |
| Region/Zone Info | [Region/Zone Info Guide](https://github.com/cloud-barista/cb-spider/wiki/REST-API-Region-Zone-Information-Guide) |
| Quota Info | [Quota Info Guide](https://github.com/cloud-barista/cb-spider/wiki/Quota-Info-Guide) |
| VM Price Info | [VM Price Info Guide](https://github.com/cloud-barista/cb-spider/wiki/VM-Price-Info-Guide) |
| VM Image Info | [Public Image Info Guide](https://github.com/cloud-barista/cb-spider/wiki/Public-Image-Info-Guide) |
| VM Spec Info | [VM Spec Info Guide](https://github.com/cloud-barista/cb-spider/wiki/VM-Spec-Info-Guide) |
| VPC/Subnet | [VPC/Subnet Management Guide](https://github.com/cloud-barista/cb-spider/wiki/VPC-Subnet-Management-Guide) |
| Security Group | [SecurityGroup Management Guide](https://github.com/cloud-barista/cb-spider/wiki/SecurityGroup-Management-Guide) |
| KeyPair | [KeyPair Management Guide](https://github.com/cloud-barista/cb-spider/wiki/KeyPair-Management-Guide) |
| VM | [VM Management Guide](https://github.com/cloud-barista/cb-spider/wiki/VM-Management-Guide) |
| Disk | [Disk Management Guide](https://github.com/cloud-barista/cb-spider/wiki/Disk-Management-Guide) |
| NLB | [Network Load Balancer Guide](https://github.com/cloud-barista/cb-spider/wiki/Network-Load-Balancer(NLB)-Guide) |
| Kubernetes Cluster | [K8S Cluster Management Guide](https://github.com/cloud-barista/cb-spider/wiki/Kubernetes-Cluster-Management-Guide) |
| Object Storage (S3) | [Object Storage and S3 API Guide](https://github.com/cloud-barista/cb-spider/wiki/Object-Storage-and-S3-API-Guide) |
| Tag Management | [Tag Management Guide](https://github.com/cloud-barista/cb-spider/wiki/Tag-Management-Guide) |

> **VM default accounts:** Ubuntu/Debian → `cb-user` · Windows → `Administrator`

> 📖 **Full documentation:** [CB-Spider Wiki](https://github.com/cloud-barista/cb-spider/wiki)

---

## Contributing

We welcome contributions! Please read [CONTRIBUTING.md](https://github.com/cloud-barista/cb-spider/blob/master/CONTRIBUTING.md) before submitting a pull request.

- **Issues:** [GitHub Issues](https://github.com/cloud-barista/cb-spider/issues)
- **Pull Requests:** [GitHub Pull Requests](https://github.com/cloud-barista/cb-spider/pulls)

---

## License

CB-Spider is licensed under the [Apache License 2.0](./LICENSE).
