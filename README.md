# CB-Spider : "One-Code, Multi-Cloud"
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/cloud-barista/cb-spider?label=go.mod)](https://github.com/cloud-barista/cb-spider/blob/master/go.mod)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/cloud-barista/cb-spider/blob/master/LICENSE)&nbsp;&nbsp;&nbsp;
[![Release Version](https://img.shields.io/github/v/release/cloud-barista/cb-spider)](https://github.com/cloud-barista/cb-spider/releases)
[![Latest Docs](https://img.shields.io/badge/docs-latest-green)](https://github.com/cloud-barista/cb-spider/wiki)
[![Swagger API Docs](https://img.shields.io/badge/docs-Swagger_API-blue)](https://cloud-barista.github.io/api/?url=https://raw.githubusercontent.com/cloud-barista/cb-spider/refs/heads/master/api/swagger.yaml)


- CB-Spider is a sub-framework of the Cloud-Barista Multi-Cloud Platform.<br>
- CB-Spider implements multi-cloud infrastructure abstraction and integration technology.<br>
- CB-Spider provides a unified interface and view for efficient multi-cloud management.

<p align="center">
  <img width="850" alt="image" src="https://github.com/user-attachments/assets/c1e5328b-151d-4b24-ad62-947e8bfcbbcf">
</p>


```
[NOTE]
CB-Spider is currently under development and has not yet reached version 1.0.
We welcome suggestions, issues, feedback, and contributions!
Please be aware that the functionalities of Cloud-Barista are not yet stable or secure.
Exercise caution if you plan to use the current release in a production environment.
If you encounter any difficulties while using Cloud-Barista, please let us know.
(You can open an issue or join the Cloud-Barista Slack community.)
```

***

### ▶ **[Quick Guide](https://github.com/cloud-barista/cb-spider/wiki/Quick-Start-Guide)**
***

#### Table of Contents

1. [Recommended Environment](#1-recommended-environment)  
2. [How to Run](#2-how-to-run)  
3. [Supported Resources](#3-supported-resources)  
4. [VM Accounts](#4-vm-accounts)  
5. [Usage](#5-usage)  
6. [API Specifications](#6-api-specifications)  
7. [Notes](#7-notes)  
8. [References](#8-references)  

***

#### 1. Recommended Environment

- OS: Ubuntu 24.04  
- Build: Go 1.25, Swag v1.16.3  
- Container: Docker v28.0.0  

---

#### 2. How to Run

- ##### Source-based: https://github.com/cloud-barista/cb-spider/wiki/Quick-Start-Guide  
- ##### Container-based: https://github.com/cloud-barista/cb-spider/wiki/Docker-based-Start-Guide  

---

#### 3. Supported Resources
- #### Reference: [Tagging Guide](https://github.com/cloud-barista/cb-spider/wiki/Tag-and-Cloud-Driver-API)  

| Provider      | VM Price<br>Info | Region/Zone<br>Info | Image<br>Info | VMSpec<br>Info | VPC<br>Subnet       | Security<br>Group | VM KeyPair      | VM             | Disk | MyImage | NLB | K8S | Object<br> Storage |
|:-------------:|:-------------:|:-------------------:|:-------------:|:--------------:|:-------------------:|:-----------------:|:---------------:|:--------------:|:----:|:---:|:-------:|:-----------:|:-----------:|
| AWS           | O             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O        | O        |
| Azure         | O             | O                  | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O        | WIP        |
| GCP           | O             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O        | O        |
| Alibaba       | O             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O           | O        |
| Tencent       | O             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O           | WIP        |
| IBM           | O             | O                  | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | O        | O        |
| OpenStack     | NA             | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | ?           | WIP        |
| NCP           | O            | O                   | O             | O              | O                   | O                 | O               | O              | O    | O   | O       | WIP           | O        |
| NHN           | NA             | O                   | O             | O              | O                 | O                 | O               | O<br>(Note1)   | O    | O    | O     | O           | O        |
| KT            | NA             | O                   | O             | O              | O<br>(Type1)       | O                 | O               | O              | O    | O   | O<br>(Note2)| Wait API  | O        |
| KT Classic    | NA             | O                   | O             | O              | O<br>(Type2)       | O                 | O               | O              | O    | O   | O       | NA          | -        |

※ WIP: Work In Progress, NA: Not Applicable, Wait API: Pending CSP API Release, ?: TBD/Analysis Needed, -: Excluded (Classic Resource)  

**VPC Notes (see each driver’s README for details):**  
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

---

#### 4. VM Accounts
- Ubuntu, Debian: `cb-user`  
- Windows: `Administrator`  

---

#### 5. Usage
- [Feature and Usage Guide](https://github.com/cloud-barista/cb-spider/wiki/features-and-usages)  

---

#### 6. API Specifications
- [Swagger Documentation](https://github.com/cloud-barista/cb-spider/tree/master/api)  
- [Swagger Guide](https://github.com/cloud-barista/cb-spider/wiki/Swagger-Guide)  

---

#### 7. Notes
- Development status: Focused on core features / For R&D use / Needs reinforcement for production use  

---

#### 8. References
- Wiki: https://github.com/cloud-barista/cb-spider/wiki  
