#!/bin/bash

# CB-Spider S3 API Test Runner for All CSPs
# This script runs S3 API tests for all supported CSPs in sequence
# Author: CB-Spider Team

# Initialize result tracking (individual variables for bash 3.2 compatibility)
# Uses tr_<CSP>_<category>, csp_pc_<CSP>, csp_tc_<CSP> variables via eval

# Function to extract test results from output
extract_results() {
    local csp_name="$1"
    local output="$2"
    
    # Extract SUMMARY section
    local summary=$(echo "$output" | sed -n '/^SUMMARY:/,/^$/p')
    
    # Extract total and passed counts
    local total=$(echo "$summary" | grep "Total Tests:" | grep -o '[0-9]*' | head -1)
    local passed=$(echo "$summary" | grep "Passed:" | grep -o '[0-9]*' | head -1)
    
    if [[ -n "$total" && -n "$passed" ]]; then
        eval "csp_pc_${csp_name}=\${passed}"
        eval "csp_tc_${csp_name}=\${total}"
        
        # Extract category results from TEST REPORT section
        # Use more precise pattern matching to avoid counting duplicate PASS entries
        local bucket_pass=$(echo "$output" | sed -n '/^1\. BUCKET MANAGEMENT/,/^2\. OBJECT MANAGEMENT/p' | grep -c "| PASS")
        local object_pass=$(echo "$output" | sed -n '/^2\. OBJECT MANAGEMENT/,/^3\. MULTIPART UPLOAD/p' | grep -c "| PASS")
        local multipart_pass=$(echo "$output" | sed -n '/^3\. MULTIPART UPLOAD/,/^4\. VERSIONING MANAGEMENT/p' | grep -c "| PASS")
        local versioning_pass=$(echo "$output" | sed -n '/^4\. VERSIONING MANAGEMENT/,/^5\. CORS MANAGEMENT/p' | grep -c "| PASS")
        local cors_pass=$(echo "$output" | sed -n '/^5\. CORS MANAGEMENT/,/^6\. CB-SPIDER SPECIAL/p' | grep -c "| PASS")
        local special_pass=$(echo "$output" | sed -n '/^6\. CB-SPIDER SPECIAL/,/^SUMMARY:/p' | grep -c "| PASS")
        
        eval "tr_${csp_name}_bucket=\${bucket_pass}/6"
        eval "tr_${csp_name}_object=\${object_pass}/6"
        eval "tr_${csp_name}_multipart=\${multipart_pass}/6"
        eval "tr_${csp_name}_versioning=\${versioning_pass}/4"
        eval "tr_${csp_name}_cors=\${cors_pass}/4"
        eval "tr_${csp_name}_special=\${special_pass}/6"
    else
        eval "csp_pc_${csp_name}=0"
        eval "csp_tc_${csp_name}=0"
        eval "tr_${csp_name}_bucket=ERR"
        eval "tr_${csp_name}_object=ERR"
        eval "tr_${csp_name}_multipart=ERR"
        eval "tr_${csp_name}_versioning=ERR"
        eval "tr_${csp_name}_cors=ERR"
        eval "tr_${csp_name}_special=ERR"
    fi
}

# Function to run test and capture output
run_test() {
    local test_script="$1"
    local csp_name="$2"
    
    echo ""
    echo "##################################################################################"
    echo "#                         ${csp_name} S3 TEST                                    "
    echo "##################################################################################"
    echo ""
    
    local output=$(./"$test_script" 2>&1)
    echo "$output"
    extract_results "$csp_name" "$output"
}

run_test "aws-test.sh" "AWS"
run_test "gcp-test.sh" "GCP"
run_test "azure-test.sh" "AZURE"
run_test "alibaba-test.sh" "ALIBABA"
run_test "tencent-test.sh" "TENCENT"
run_test "ibm-test.sh" "IBM"
run_test "openstack-test.sh" "OPENSTACK"
run_test "ncp-test.sh" "NCP"
run_test "nhn-test.sh" "NHN"
run_test "kt-test.sh" "KT"

echo ""
echo ""
echo "##################################################################################"
echo "#                       ALL CSP TESTS COMPLETED                                  #"
echo "##################################################################################"
echo ""
echo ""
echo "===================================================================================="
echo "                         S3 API TEST SUMMARY - ALL CSPs"
echo "===================================================================================="
echo ""
printf "%-12s | %-8s | %-8s | %-10s | %-10s | %-8s | %-8s | %-10s\n" \
    "CSP" "Bucket" "Object" "Multipart" "Versioning" "CORS" "Special" "Total"
echo "------------------------------------------------------------------------------------"

# Print results for each CSP
for csp in AWS GCP AZURE ALIBABA TENCENT IBM OPENSTACK NCP NHN KT; do
    eval "bucket=\${tr_${csp}_bucket:-N/A}"
    eval "object=\${tr_${csp}_object:-N/A}"
    eval "multipart=\${tr_${csp}_multipart:-N/A}"
    eval "versioning=\${tr_${csp}_versioning:-N/A}"
    eval "cors=\${tr_${csp}_cors:-N/A}"
    eval "special=\${tr_${csp}_special:-N/A}"
    eval "total_pass=\${csp_pc_${csp}:-0}"
    eval "total_count=\${csp_tc_${csp}:-0}"
    
    # Handle not supported cases
    if [[ "$csp" == "AZURE" ]]; then
        multipart="0/6 (NA)"
        versioning="0/4 (NA)"
        cors="0/4 (NA)"
    elif [[ "$csp" == "OPENSTACK" ]]; then
        multipart="0/6 (NA)"
        versioning="0/4 (NA)"
    elif [[ "$csp" == "NCP" || "$csp" == "NHN" ]]; then
        versioning="0/4 (NA)"
        cors="0/4 (NA)"
    fi
    
    printf "%-12s | %-8s | %-8s | %-10s | %-10s | %-8s | %-8s | %-10s\n" \
        "$csp" "$bucket" "$object" "$multipart" "$versioning" "$cors" "$special" "$total_pass/$total_count"
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
