#!/bin/bash

# NHN RDBMS Collation Configuration Test Script (MariaDB)
#
# PREREQUISITE: this instance must have "DB 스키마 & 사용자 직접 제어" ("Direct Control") enabled
# in the NHN console before running this test -- see ../charset-test/nhn-charset-test.sh for the
# full explanation. Without it, this test will fail with a SQL permission error from the MariaDB
# engine itself, not a CB-Spider "not supported" message.
export CSP_NAME="NHN"
export CONNECTION_NAME="nhn-korea-pangyo1-config"
export RDBMS_NAME="cb-spider-mariadb-test"
export MASTER_USER_NAME="myadmin"
export MASTER_USER_PASSWORD="Password123!"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_collation_results}/result_nhn.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-collation-test.sh"
