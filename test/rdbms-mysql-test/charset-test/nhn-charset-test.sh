#!/bin/bash

# NHN RDBMS Charset Configuration Test Script
#
# PREREQUISITE: this instance must have "DB 스키마 & 사용자 직접 제어" ("Direct Control") enabled
# in the NHN console before running this test. NHN's native API has no charset/collation field, so
# CB-Spider falls back to a plain SQL CREATE DATABASE whenever Charset/Collation is requested (see
# RDBMSManager.go's rdbmsDatabaseSQLFallbackEligible) -- but NHN blocks that for the master account
# by default. CB-Spider cannot check or enable Direct Control itself; if it isn't on, this test
# will fail with a SQL permission error (e.g. "command denied") rather than CB-Spider's own
# "not supported" message, since the error comes straight from the MySQL/MariaDB engine. See
# NHN Cloud's RDS documentation for how to enable Direct Control, and ../charset-test/README.md
# (if listed there) for the tradeoffs -- enabling it grants broad privileges to all existing users
# on this instance.
export CSP_NAME="NHN"
export CONNECTION_NAME="nhn-korea-pangyo1-config"
export RDBMS_NAME="cb-spider-mysql-test"
export MASTER_USER_NAME="myadmin"
export MASTER_USER_PASSWORD="Password123!"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_charset_results}/result_nhn.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-charset-test.sh"
