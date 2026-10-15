#!/bin/bash

# Tencent RDBMS Charset Configuration Test Script
# Note: Tencent's CreateDatabase API has no Collation field at all -- CB-Spider applies a
# requested Collation with a follow-up ALTER DATABASE over direct SQL (see RDBMSManager.go's
# rdbmsDatabaseCharsetManager), which is why MASTER_USER_NAME/PASSWORD are required here even
# though they aren't for a plain (charset/collation-less) CreateDatabase on Tencent.
export CSP_NAME="TENCENT"
export CONNECTION_NAME="tencent-beijing3-config"
export RDBMS_NAME="cb-spider-mysql-test"
export MASTER_USER_NAME="root"
export MASTER_USER_PASSWORD="Password123!"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_charset_results}/result_tencent.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-charset-test.sh"
