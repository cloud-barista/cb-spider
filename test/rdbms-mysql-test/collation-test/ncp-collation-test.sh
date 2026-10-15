#!/bin/bash

# NCP RDBMS Collation Configuration Test Script
# Note: NCP has no native charset/collation field; CB-Spider always routes through NCP's
# documented `sys.ncp_create_db('name','charset','collation')` stored procedure over direct SQL
# whenever Charset/Collation is requested (see RDBMSManager.go's rdbmsDatabaseSQLStatementBuilder),
# which is why MASTER_USER_NAME/PASSWORD are required here.
export CSP_NAME="NCP"
export CONNECTION_NAME="ncp-korea1-config"
export RDBMS_NAME="cb-spider-mysql-test"
export MASTER_USER_NAME="myadmin"
export MASTER_USER_PASSWORD="Password123!"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_collation_results}/result_ncp.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-collation-test.sh"
