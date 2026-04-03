#!/bin/bash

# CB-Spider S3 API Test Runner for All CSPs (SigV4 / awscurl)
# This script runs S3 API tests for all supported CSPs in sequence.
# Uses AWS Signature Version 4 (SigV4) via awscurl for authentication.
# Requires: awscurl (pip install awscurl)
# Author: CB-Spider Team

# ─── Pre-flight checks ────────────────────────────────────────────────────────
if ! command -v awscurl &>/dev/null; then
    echo "[ERROR] 'awscurl' is not installed. Install with: pip install awscurl"
    exit 1
fi

# Ensure scripts are executable
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
chmod +x "$SCRIPT_DIR"/*.sh

# ─── Result tracking ──────────────────────────────────────────────────────────
declare -A test_results
declare -A csp_pass_count
declare -A csp_total_count

# ─── Helpers ──────────────────────────────────────────────────────────────────
extract_results() {
    local csp_name="$1"
    local output="$2"

    # Extract SUMMARY section
    local summary
    summary=$(echo "$output" | sed -n '/^SUMMARY:/,/^$/p')

    local total passed
    total=$(echo  "$summary" | grep "Total Tests"  | grep -o '[0-9]*' | head -1)
    passed=$(echo "$summary" | grep "Passed"       | grep -o '[0-9]*' | head -1)

    if [[ -n "$total" && -n "$passed" ]]; then
        csp_pass_count["$csp_name"]="$passed"
        csp_total_count["$csp_name"]="$total"

        local bucket_pass object_pass multipart_pass versioning_pass cors_pass special_pass
        bucket_pass=$(echo    "$output" | sed -n '/^1\. BUCKET MANAGEMENT/,/^2\. OBJECT MANAGEMENT/p'       | grep -c "| PASS")
        object_pass=$(echo    "$output" | sed -n '/^2\. OBJECT MANAGEMENT/,/^3\. MULTIPART UPLOAD/p'        | grep -c "| PASS")
        multipart_pass=$(echo "$output" | sed -n '/^3\. MULTIPART UPLOAD/,/^4\. VERSIONING MANAGEMENT/p'    | grep -c "| PASS")
        versioning_pass=$(echo "$output"| sed -n '/^4\. VERSIONING MANAGEMENT/,/^5\. CORS MANAGEMENT/p'     | grep -c "| PASS")
        cors_pass=$(echo      "$output" | sed -n '/^5\. CORS MANAGEMENT/,/^6\. CB-SPIDER SPECIAL/p'         | grep -c "| PASS")
        special_pass=$(echo   "$output" | sed -n '/^6\. CB-SPIDER SPECIAL/,/^=====*/p'                      | grep -c "| PASS")

        test_results["${csp_name}_bucket"]="$bucket_pass/6"
        test_results["${csp_name}_object"]="$object_pass/6"
        test_results["${csp_name}_multipart"]="$multipart_pass/6"
        test_results["${csp_name}_versioning"]="$versioning_pass/4"
        test_results["${csp_name}_cors"]="$cors_pass/4"
        test_results["${csp_name}_special"]="$special_pass/6"
    else
        csp_pass_count["$csp_name"]="0"
        csp_total_count["$csp_name"]="0"
        test_results["${csp_name}_bucket"]="ERR"
        test_results["${csp_name}_object"]="ERR"
        test_results["${csp_name}_multipart"]="ERR"
        test_results["${csp_name}_versioning"]="ERR"
        test_results["${csp_name}_cors"]="ERR"
        test_results["${csp_name}_special"]="ERR"
    fi
}

run_csp_test() {
    local test_script="$1"
    local csp_name="$2"

    echo ""
    echo "==================================================================================="
    echo "                       ${csp_name} S3 SIGV4 TEST"
    echo "==================================================================================="
    echo ""

    local output
    output=$("$SCRIPT_DIR/$test_script" 2>&1)
    echo "$output"
    extract_results "$csp_name" "$output"
}

# ─── Run all CSPs ─────────────────────────────────────────────────────────────
run_csp_test "aws-test.sh"        "AWS"
run_csp_test "gcp-test.sh"        "GCP"
run_csp_test "alibaba-test.sh"    "ALIBABA"
run_csp_test "tencent-test.sh"    "TENCENT"
run_csp_test "ibm-test.sh"        "IBM"
run_csp_test "openstack-test.sh"  "OPENSTACK"
run_csp_test "ncp-test.sh"        "NCP"
run_csp_test "nhn-test.sh"        "NHN"
run_csp_test "kt-test.sh"         "KT"

# ─── Aggregate report ─────────────────────────────────────────────────────────
echo ""
echo ""
echo "==================================================================================="
echo "#                       ALL CSP SIGV4 TESTS COMPLETED                            #"
echo "==================================================================================="
echo ""
echo "==================================================================================="
echo "                    S3 API TEST SUMMARY (SigV4) — ALL CSPs"
echo "==================================================================================="
echo ""

printf "%-12s | %-8s | %-8s | %-10s | %-10s | %-8s | %-8s | %-10s\n" \
    "CSP" "Bucket" "Object" "Multipart" "Versioning" "CORS" "Special" "Total"
echo "------------------------------------------------------------------------------------"

for csp in AWS GCP ALIBABA TENCENT IBM OPENSTACK NCP NHN KT; do
    bucket="${test_results[${csp}_bucket]:-N/A}"
    object="${test_results[${csp}_object]:-N/A}"
    multipart="${test_results[${csp}_multipart]:-N/A}"
    versioning="${test_results[${csp}_versioning]:-N/A}"
    cors="${test_results[${csp}_cors]:-N/A}"
    special="${test_results[${csp}_special]:-N/A}"
    total_pass="${csp_pass_count[$csp]:-0}"
    total_count="${csp_total_count[$csp]:-0}"

    # Annotate CSPs with known limitations
    if [[ "$csp" == "OPENSTACK" ]]; then
        multipart="0/6 (NA)"
        versioning="0/4 (NA)"
    elif [[ "$csp" == "NCP" || "$csp" == "NHN" ]]; then
        versioning="0/4 (NA)"
        cors="0/4 (NA)"
    fi

    printf "%-12s | %-8s | %-8s | %-10s | %-10s | %-8s | %-8s | %-10s\n" \
        "$csp" "$bucket" "$object" "$multipart" "$versioning" "$cors" "$special" \
        "$total_pass/$total_count"
done

echo "------------------------------------------------------------------------------------"
echo ""
echo "Legend: X/Y (Pass/Total), NA = Not Applicable (CSP does not support)"
echo ""
echo "Test Categories:"
echo "  - Bucket      : Bucket management operations (6 tests)"
echo "  - Object      : Object operations (6 tests)"
echo "  - Multipart   : Multipart upload (6 tests)"
echo "  - Versioning  : Bucket versioning (4 tests)"
echo "  - CORS        : CORS configuration (4 tests)"
echo "  - Special     : PreSigned URLs & Force operations (6 tests)"
echo ""
echo "===================================================================================="
echo ""
