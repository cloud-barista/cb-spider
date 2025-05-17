# CB-Spider Default Connection Helper

**`default-connection-helper`** is a utility tool for CB-Spider that helps you easily register default connections for all supported Cloud Service Providers (CSPs) or specific CSPs.

## Overview

This utility automates the process of setting up cloud connections by performing the following steps for each CSP:

1. Register driver
2. Register credential
3. Register all available regions and zones
4. Create connection configurations for all regions and zones

All registered connections follow the naming format:
```
{CSP}.{Region}.{Zone}
```

For example:
- `aws.ap-northeast-2.ap-northeast-2a`
- `azure.eastus.1`
- `gcp.asia-northeast3.asia-northeast3-a`

## Prerequisites

Before using this utility, you need to:

1. Have a running Spider server
2. Set up credential files for the CSPs you want to connect to

## Setting Up Credentials

For each CSP you want to connect to, you need to set up a credential file:

1. Find the sample credential file (e.g., `.aws-credential.sample`)
2. Create a copy with the proper name (e.g., `.aws-credential`)
3. Edit the file and add your actual cloud credentials

Example for AWS:
```bash
cp .aws-credential.sample .aws-credential
vi .aws-credential
# Add your AWS access key and secret key
```

## Usage

### Register Default Connections for All CSPs

To register default connections for all supported CSPs:

```bash
./all_register.sh
```

This will process all CSP registration scripts in parallel and report the status of each operation.

### Delete All Registered Connections

To remove all connections, credentials, and drivers that were registered using this utility:

```bash
./all_delete.sh
```

### Connection Naming Convention

All connections registered by this utility follow a standard naming convention:
```
{CSP}.{Region}.{Zone}
```

Examples:
- `aws.us-east-1.us-east-1a`
- `azure.koreacentral.1`
- `gcp.us-central1.us-central1-a`
- `alibaba.ap-southeast-1.ap-southeast-1a`

This consistent naming convention makes it easy to locate and manage connections across different cloud providers.


### Register a Specific CSP

To register just one specific CSP:

```bash
# For AWS
./aws_register.sh

# For Azure
./azure_register.sh

# For GCP
./gcp_register.sh

# And so on for other supported CSPs...
```

### Delete a Specific CSP's Connections

To delete connections for a specific CSP:

```bash
# For AWS
./aws_delete.sh

# For Azure
./azure_delete.sh

# For GCP
./gcp_delete.sh

# And so on for other supported CSPs...
```

## Excluding Specific CSPs

If you want to register all CSPs except for specific ones:

1. Remove or rename the registration and deletion scripts for the CSPs you want to exclude
2. Run the all_register.sh or all_delete.sh script

Example to exclude AWS:
```bash
# Rename or remove AWS scripts
mv aws_register.sh aws_register.sh.disabled
mv aws_delete.sh aws_delete.sh.disabled

# Then run the all scripts
./all_register.sh
```

## Supported CSPs

This utility supports the following Cloud Service Providers:

- AWS
- Azure
- GCP
- Alibaba Cloud
- Tencent Cloud
- IBM Cloud (VPC)
- OpenStack
- NCP (Classic)
- NCP (VPC)
- NHN Cloud
- KT Cloud (Classic)
- KT Cloud (VPC)
- Mock Cloud (for testing)

## Troubleshooting

- If a script hangs, it will automatically timeout after 60 seconds
- Check the status report at the end of execution for any failed operations
- Make sure your credential files are properly set up
- Verify that the Spider server is running and accessible

## Advanced Configuration

The utility uses the `RESTSERVER` environment variable to connect to the Spider server. By default, it uses `localhost`. To specify a different server:

```bash
export RESTSERVER="192.168.0.1"
./all_register.sh
```
