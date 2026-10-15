#!/bin/bash

# Tencent RDBMS Collation Configuration Test Script
# Note: Tencent's CreateDatabase API has no Collation field at all -- CB-Spider applies the
# requested Collation with a follow-up ALTER DATABASE over direct SQL (see RDBMSManager.go's
# rdbmsDatabaseCharsetManager). This test (unlike ../charset-test) always requests a Collation, so
# it always exercises that SQL path and always requires MASTER_USER_NAME/PASSWORD.
export CSP_NAME="TENCENT"
export CONNECTION_NAME="tencent-beijing3-config"
export RDBMS_NAME="cb-spider-mysql-test"
export MASTER_USER_NAME="root"
export MASTER_USER_PASSWORD="Password123!"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_collation_results}/result_tencent.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-collation-test.sh"
