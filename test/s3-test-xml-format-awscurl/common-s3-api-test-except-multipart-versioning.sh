#!/bin/bash

# CB-Spider S3 API Test Script (awscurl / SigV4) — Multipart & Versioning skipped
# For CSPs that do not support Multipart upload and Versioning (e.g. OpenStack Swift).
# Mirrors common-s3-api-test-except-multipart-versioning.sh but uses awscurl.
# Requires: awscurl (pip install awscurl)
# Author: CB-Spider Team

SPIDER_USERNAME=${SPIDER_USERNAME:-admin}
SPIDER_PASSWORD=${SPIDER_PASSWORD:?"SPIDER_PASSWORD is required"}
CONNECTION_NAME="${CONNECTION_NAME:?"CONNECTION_NAME is required"}"

ACCESS_KEY="${SPIDER_USERNAME}@${CONNECTION_NAME}"
SECRET_KEY="${SPIDER_PASSWORD}"

SPIDER_URL="http://localhost:1024/spider/s3"

TEST_BUCKET="cb-spider-test-sigv4-$(date +%s)"
TEST_OBJECT="test-file.txt"
TEST_CONTENT="Hello CB-Spider S3 SigV4 Test!"

TEMP_DIR="/tmp/cb-spider-s3-sigv4-test-$$-$(date +%s)"
mkdir -p "$TEMP_DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Individual tr_<test_name> variables store results (bash 3.2 compatible)
test_count=0
pass_count=0
fail_count=0

log_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_error()   { echo -e "${RED}[FAIL]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }

# Usage: s3curl [curl-compatible args] URL
# Translates curl-only flags to awscurl equivalents and injects auth.
# Handles: -s (drop), -w '%{http_code}' (via -v stderr capture), -o (drop in http-code mode),
#          --data-binary '@file' (-> --data-binary -d @file),
#          --data-binary 'string' (-> -d 'string').
# Note: awscurl -v logs "Response code: NNN" to stderr; we capture that to get the HTTP status.
s3curl() {
    # Pass 1: detect whether -w '%{http_code}' is present (changes execution path)
    local want_http_code=false
    local argv=("$@") n=${#argv[@]} i=0
    while [[ $i -lt $n ]]; do
        [[ "${argv[$i]}" == "-w" ]] && want_http_code=true
        i=$((i+1))
    done

    # Pass 2: build clean args for awscurl
    local args=()
    i=0
    while [[ $i -lt $n ]]; do
        local arg="${argv[$i]}"
        case "$arg" in
            -s)
                # drop: awscurl is quiet by default
                ;;
            -w)
                i=$((i+1))
                # drop -w and its format string ('%{http_code}')
                ;;
            -o)
                i=$((i+1))
                # keep -o <file> only when not in http-code-capture mode
                if ! $want_http_code; then
                    args+=("-o" "${argv[$i]}")
                fi
                ;;
            --data-binary)
                i=$((i+1))
                local dval="${argv[$i]}"
                if [[ "$dval" == @* ]]; then
                    # file reference: awscurl needs '--data-binary' flag + '-d @file'
                    args+=("--data-binary" "-d" "$dval")
                else
                    # inline data string: awscurl uses plain '-d'
                    args+=("-d" "$dval")
                fi
                ;;
            *)
                args+=("$arg")
                ;;
        esac
        i=$((i+1))
    done

    if $want_http_code; then
        # awscurl -v logs "Response code: NNN" to stderr when IS_VERBOSE=True.
        # Capture stderr in a temp file; stdout is the plain response body.
        local _stmp
        _stmp=$(mktemp)
        local body
        body=$(awscurl \
            --service s3 \
            --access_key "$ACCESS_KEY" \
            --secret_key "$SECRET_KEY" \
            -v "${args[@]}" 2>"$_stmp")
        local http_code
        http_code=$(grep -oE 'Response code: [0-9]+' "$_stmp" | grep -oE '[0-9]+' | head -1)
        rm -f "$_stmp"
        # Print response body then HTTP code (mirrors curl -w '%{http_code}' behaviour)
        printf '%s' "$body"
        printf '%s' "$http_code"
    else
        awscurl \
            --service s3 \
            --access_key "$ACCESS_KEY" \
            --secret_key "$SECRET_KEY" \
            "${args[@]}"
    fi
}

bucket_exists() {
    local code
    code=$(s3curl -s -o /dev/null -w '%{http_code}' -X HEAD "$SPIDER_URL/$TEST_BUCKET")
    [[ "$code" == "200" ]]
}

cleanup_multipart_uploads() {
    log_info "Cleaning up incomplete multipart uploads..."
    local uploads_response
    uploads_response=$(s3curl -s -X GET "$SPIDER_URL/$TEST_BUCKET?uploads" 2>/dev/null)
    if [[ -n "$uploads_response" ]] && [[ "$uploads_response" =~ \<UploadId\> ]]; then
        echo "$uploads_response" | sed 's/<\/Upload>/\n<\/Upload>\n/g' | grep '<Upload>' | while read -r upload_block; do
            local key upload_id
            key=$(echo "$upload_block"       | grep -o '<Key>[^<]*</Key>'           | sed 's/<[^>]*>//g' | head -1)
            upload_id=$(echo "$upload_block"  | grep -o '<UploadId>[^<]*</UploadId>' | sed 's/<[^>]*>//g' | head -1)
            if [[ -n "$key" && -n "$upload_id" ]]; then
                log_info "Aborting multipart upload: $key (ID: $upload_id)"
                s3curl -s -X DELETE "$SPIDER_URL/$TEST_BUCKET/$key?uploadId=$upload_id" >/dev/null 2>&1
            fi
        done
    fi
}

cleanup_all_objects() {
    log_info "Cleaning up all objects in bucket..."
    local objects_response
    objects_response=$(s3curl -s -X GET "$SPIDER_URL/$TEST_BUCKET" 2>/dev/null)
    if [[ -n "$objects_response" ]] && [[ "$objects_response" =~ \<Key\> ]]; then
        echo "$objects_response" | grep -o '<Key>[^<]*</Key>' | sed 's/<[^>]*>//g' | while read -r key; do
            if [[ -n "$key" ]]; then
                log_info "Deleting object: $key"
                s3curl -s -X DELETE "$SPIDER_URL/$TEST_BUCKET/$key" >/dev/null 2>&1
            fi
        done
    fi
}

cleanup() {
    log_info "Cleaning up test resources..."
    # No multipart cleanup: not supported by this CSP
    if bucket_exists; then
        cleanup_all_objects
        s3curl -s -X DELETE "$SPIDER_URL/$TEST_BUCKET?force=true" >/dev/null 2>&1
    else
        log_info "Bucket $TEST_BUCKET already removed, skipping force delete in cleanup"
    fi
    rm -f "$TEMP_DIR/$TEST_OBJECT" \
          "$TEMP_DIR/large-file.txt" \
          "$TEMP_DIR/downloaded-file.txt" \
          "$TEMP_DIR/presigned-download.txt" \
          "$TEMP_DIR/presigned-upload.txt"
    log_info "Cleanup completed"
}

run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_pattern="$3"
    local description="$4"

    test_count=$((test_count + 1))
    log_info "Testing: $test_name"

    local result exit_code
    result=$(eval "$test_command" 2>&1)
    exit_code=$?

    local status="FAIL"
    if [[ $exit_code -eq 0 ]] && [[ -z "$expected_pattern" || "$result" =~ $expected_pattern ]]; then
        status="PASS"
        pass_count=$((pass_count + 1))
        log_success "$test_name - $description"
    else
        fail_count=$((fail_count + 1))
        log_error "$test_name - $description"
        echo "  Command   : $test_command"
        echo "  Exit Code : $exit_code"
        echo "  Output    : $result"
    fi
    eval "tr_${test_name}=\${status}"
}

create_test_file() {
    echo "$TEST_CONTENT" > "$TEMP_DIR/$TEST_OBJECT"
    echo "Large file content for multipart upload test" > "$TEMP_DIR/large-file.txt"
    for i in {1..100}; do
        echo "Line $i: This is test content for large file upload" >> "$TEMP_DIR/large-file.txt"
    done
}

print_summary() {
    echo
    echo "==================================================================================="
    echo "     CB-SPIDER S3 API TEST REPORT (SigV4 / awscurl) — No Multipart/Versioning"
    echo "==================================================================================="
    echo "Test Date  : $(date '+%Y-%m-%d %H:%M:%S')"
    echo "Spider URL : $SPIDER_URL"
    echo "Access Key : $ACCESS_KEY"
    echo "Test Bucket: $TEST_BUCKET"
    echo "==================================================================================="
    echo
    printf "%-50s | %-10s\n" "TEST NAME" "STATUS"
    echo "--------------------------------------------------------------------------------"

    echo "1. BUCKET MANAGEMENT (6 tests)"
    printf "%-50s | %-10s\n" "  List Buckets"            "${tr_list_buckets:-SKIP}"
    printf "%-50s | %-10s\n" "  Create Bucket"           "${tr_create_bucket:-SKIP}"
    printf "%-50s | %-10s\n" "  Get Bucket Info"         "${tr_get_bucket_info:-SKIP}"
    printf "%-50s | %-10s\n" "  Check Bucket Exists (HEAD)" "${tr_head_bucket:-SKIP}"
    printf "%-50s | %-10s\n" "  Get Bucket Location"     "${tr_get_bucket_location:-SKIP}"
    printf "%-50s | %-10s\n" "  Delete Bucket"           "${tr_delete_bucket:-SKIP}"
    echo

    echo "2. OBJECT MANAGEMENT (6 tests)"
    printf "%-50s | %-10s\n" "  Upload Object (File)"    "${tr_upload_object_file:-SKIP}"
    printf "%-50s | %-10s\n" "  Upload Object (Form)"    "${tr_upload_object_form:-SKIP}"
    printf "%-50s | %-10s\n" "  Download Object"         "${tr_download_object:-SKIP}"
    printf "%-50s | %-10s\n" "  Get Object Info (HEAD)"  "${tr_head_object:-SKIP}"
    printf "%-50s | %-10s\n" "  Delete Object"           "${tr_delete_object:-SKIP}"
    printf "%-50s | %-10s\n" "  Delete Multiple Objects" "${tr_delete_multiple_objects:-SKIP}"
    echo

    echo "3. MULTIPART UPLOAD (6 tests) — SKIPPED (not supported by this CSP)"
    printf "%-50s | %-10s\n" "  Initiate Multipart Upload" "${tr_initiate_multipart:-SKIP}"
    printf "%-50s | %-10s\n" "  Upload Part"             "${tr_upload_part:-SKIP}"
    printf "%-50s | %-10s\n" "  Complete Multipart Upload" "${tr_complete_multipart:-SKIP}"
    printf "%-50s | %-10s\n" "  Abort Multipart Upload"  "${tr_abort_multipart:-SKIP}"
    printf "%-50s | %-10s\n" "  List Parts"              "${tr_list_parts:-SKIP}"
    printf "%-50s | %-10s\n" "  List Multipart Uploads"  "${tr_list_multipart_uploads:-SKIP}"
    echo

    echo "4. VERSIONING MANAGEMENT (4 tests) — SKIPPED (not supported by this CSP)"
    printf "%-50s | %-10s\n" "  Get Bucket Versioning"   "${tr_get_bucket_versioning:-SKIP}"
    printf "%-50s | %-10s\n" "  Set Bucket Versioning"   "${tr_set_bucket_versioning:-SKIP}"
    printf "%-50s | %-10s\n" "  List Object Versions"    "${tr_list_object_versions:-SKIP}"
    printf "%-50s | %-10s\n" "  Delete Versioned Object" "${tr_delete_versioned_object:-SKIP}"
    echo

    echo "5. CORS MANAGEMENT (4 tests)"
    printf "%-50s | %-10s\n" "  Set Bucket CORS"         "${tr_set_bucket_cors:-SKIP}"
    printf "%-50s | %-10s\n" "  Get Bucket CORS"         "${tr_get_bucket_cors:-SKIP}"
    printf "%-50s | %-10s\n" "  Test CORS with OPTIONS"  "${tr_test_cors_options:-SKIP}"
    printf "%-50s | %-10s\n" "  Delete CORS Configuration" "${tr_delete_bucket_cors:-SKIP}"
    echo

    echo "6. CB-SPIDER SPECIAL FEATURES (6 tests)"
    printf "%-50s | %-10s\n" "  Generate PreSigned URL (Download)" "${tr_generate_presigned_download:-SKIP}"
    printf "%-50s | %-10s\n" "  PreSigned URL Download Test"       "${tr_test_presigned_download:-SKIP}"
    printf "%-50s | %-10s\n" "  Generate PreSigned URL (Upload)"   "${tr_generate_presigned_upload:-SKIP}"
    printf "%-50s | %-10s\n" "  PreSigned URL Upload Test"         "${tr_test_presigned_upload:-SKIP}"
    printf "%-50s | %-10s\n" "  Force Empty Bucket"                "${tr_force_empty_bucket:-SKIP}"
    printf "%-50s | %-10s\n" "  Force Delete Bucket"               "${tr_force_delete_bucket:-SKIP}"
    echo

    echo "==================================================================================="
    echo "SUMMARY:"
    echo "  Total Tests  : $test_count"
    echo "  Passed       : $pass_count"
    echo "  Failed       : $fail_count"
    echo "  Success Rate : $(( pass_count * 100 / (test_count > 0 ? test_count : 1) ))%"
    echo "==================================================================================="
}

main() {
    echo "==================================================================================="
    echo "   CB-SPIDER S3 API TEST SUITE (SigV4 / awscurl) — No Multipart/Versioning"
    echo "==================================================================================="
    echo "Test Date  : $(date '+%Y-%m-%d %H:%M:%S')"
    echo "Spider URL : $SPIDER_URL"
    echo "Access Key : $ACCESS_KEY"
    echo "Test Bucket: $TEST_BUCKET"
    echo "==================================================================================="
    echo

    create_test_file
    trap cleanup EXIT

    # ======================================================
    # 1. BUCKET MANAGEMENT (6)
    # ======================================================
    log_info "=== 1. BUCKET MANAGEMENT TESTS ==="

    run_test "list_buckets" \
        "s3curl -s -X GET '$SPIDER_URL/'" \
        "ListAllMyBucketsResult" \
        "List all buckets"

    run_test "create_bucket" \
        "s3curl -s -w '%{http_code}' -X PUT '$SPIDER_URL/$TEST_BUCKET'" \
        "200" \
        "Create test bucket"

    sleep 2

    run_test "get_bucket_info" \
        "s3curl -s -X GET '$SPIDER_URL/$TEST_BUCKET'" \
        "ListBucketResult" \
        "Get bucket information"

    run_test "head_bucket" \
        "s3curl -s -w '%{http_code}' -X HEAD '$SPIDER_URL/$TEST_BUCKET'" \
        "200" \
        "Check bucket exists (HEAD)"

    run_test "get_bucket_location" \
        "s3curl -s -X GET '$SPIDER_URL/$TEST_BUCKET?location'" \
        "LocationConstraint" \
        "Get bucket location"

    DELETE_BUCKET="${TEST_BUCKET}-del"
    log_info "Creating separate bucket for deletion test: $DELETE_BUCKET"
    DELETE_CREATE_CODE=$(s3curl -s -o /dev/null -w '%{http_code}' -X PUT "$SPIDER_URL/$DELETE_BUCKET")
    if [[ "$DELETE_CREATE_CODE" == "200" || "$DELETE_CREATE_CODE" == "201" ]]; then
        sleep 2
        run_test "delete_bucket" \
            "s3curl -s -w '%{http_code}' -X DELETE '$SPIDER_URL/$DELETE_BUCKET'" \
            "204" \
            "Delete bucket"
    else
        log_warning "Failed to create separate bucket for deletion test (HTTP $DELETE_CREATE_CODE)"
        tr_delete_bucket="FAIL"
    fi

    # ======================================================
    # 2. OBJECT MANAGEMENT (6)
    # ======================================================
    log_info "=== 2. OBJECT MANAGEMENT TESTS ==="

    run_test "upload_object_file" \
        "s3curl -s -w '%{http_code}' -X PUT '$SPIDER_URL/$TEST_BUCKET/$TEST_OBJECT' --data-binary '@$TEMP_DIR/$TEST_OBJECT'" \
        "200" \
        "Upload object from file"

    run_test "upload_object_form" \
        "curl -u '$SPIDER_USERNAME:$SPIDER_PASSWORD' -s -w '%{http_code}' -X POST '$SPIDER_URL/$TEST_BUCKET?ConnectionName=$CONNECTION_NAME' -F 'key=form-upload.txt' -F 'file=@$TEMP_DIR/$TEST_OBJECT'" \
        "200" \
        "Upload object via form (curl Basic Auth – awscurl lacks multipart/form-data support)"

    run_test "download_object" \
        "s3curl -s -X GET '$SPIDER_URL/$TEST_BUCKET/$TEST_OBJECT' -o '$TEMP_DIR/downloaded-file.txt' && cat '$TEMP_DIR/downloaded-file.txt'" \
        "$TEST_CONTENT" \
        "Download object"

    run_test "head_object" \
        "s3curl -s -w '%{http_code}' -X HEAD '$SPIDER_URL/$TEST_BUCKET/$TEST_OBJECT'" \
        "200" \
        "Get object info (HEAD)"

    run_test "delete_object" \
        "s3curl -s -w '%{http_code}' -X DELETE '$SPIDER_URL/$TEST_BUCKET/form-upload.txt'" \
        "204" \
        "Delete single object"

    run_test "delete_multiple_objects" \
        "s3curl -s -X POST '$SPIDER_URL/$TEST_BUCKET?delete' \
            -H 'Content-Type: application/xml' \
            --data-binary '<Delete><Object><Key>$TEST_OBJECT</Key></Object></Delete>'" \
        "DeleteResult" \
        "Delete multiple objects"

    # ======================================================
    # 3. MULTIPART UPLOAD — SKIPPED
    # ======================================================
    log_info "=== 3. MULTIPART UPLOAD TESTS (SKIPPED) ==="
    tr_initiate_multipart="SKIP"
    tr_upload_part="SKIP"
    tr_list_parts="SKIP"
    tr_abort_multipart="SKIP"
    tr_complete_multipart="SKIP"
    tr_list_multipart_uploads="SKIP"
    log_warning "Multipart upload tests skipped (6 tests)"

    # ======================================================
    # 4. VERSIONING MANAGEMENT — SKIPPED
    # ======================================================
    log_info "=== 4. VERSIONING MANAGEMENT TESTS (SKIPPED) ==="
    tr_get_bucket_versioning="SKIP"
    tr_set_bucket_versioning="SKIP"
    tr_list_object_versions="SKIP"
    tr_delete_versioned_object="SKIP"
    log_warning "Versioning management tests skipped (4 tests)"

    # ======================================================
    # 5. CORS MANAGEMENT (4)
    # ======================================================
    log_info "=== 5. CORS MANAGEMENT TESTS ==="

    run_test "set_bucket_cors" \
        "s3curl -s -w '%{http_code}' -X PUT '$SPIDER_URL/$TEST_BUCKET?cors' \
            -H 'Content-Type: application/xml' \
            --data-binary '<CORSConfiguration><CORSRule><AllowedOrigin>*</AllowedOrigin><AllowedMethod>GET</AllowedMethod><AllowedMethod>PUT</AllowedMethod><AllowedHeader>*</AllowedHeader></CORSRule></CORSConfiguration>'" \
        "200" \
        "Set bucket CORS configuration"

    run_test "get_bucket_cors" \
        "s3curl -s -X GET '$SPIDER_URL/$TEST_BUCKET?cors'" \
        "CORSRule" \
        "Get bucket CORS configuration"

    run_test "test_cors_options" \
        "curl -s -w '%{http_code}' -X OPTIONS '$SPIDER_URL/$TEST_BUCKET' \
            -H 'Origin: http://example.com' \
            -H 'Access-Control-Request-Method: GET'" \
        "204" \
        "Test CORS with OPTIONS (no auth – standard CORS preflight)"

    run_test "delete_bucket_cors" \
        "s3curl -s -w '%{http_code}' -X DELETE '$SPIDER_URL/$TEST_BUCKET?cors'" \
        "204" \
        "Delete CORS configuration"

    # ======================================================
    # 6. CB-SPIDER SPECIAL FEATURES (6)
    # ======================================================
    log_info "=== 6. CB-SPIDER SPECIAL FEATURES ==="

    s3curl -s -X PUT "$SPIDER_URL/$TEST_BUCKET/presigned-test.txt" \
        --data-binary "@$TEMP_DIR/$TEST_OBJECT" >/dev/null

    run_test "generate_presigned_download" \
        "PRESIGNED_DOWNLOAD_URL=\$(s3curl -s -X GET '$SPIDER_URL/$TEST_BUCKET/presigned-test.txt?presigned&duration=3600' | grep -o '<PresignedURL>[^<]*</PresignedURL>' | sed 's/<[^>]*>//g'); echo \"Generated URL: \${PRESIGNED_DOWNLOAD_URL:0:50}...\"" \
        "Generated URL:" \
        "Generate presigned download URL"

    PRESIGNED_DOWNLOAD_URL=$(s3curl -s -X GET \
        "$SPIDER_URL/$TEST_BUCKET/presigned-test.txt?presigned&duration=3600" \
        | grep -o '<PresignedURL>[^<]*</PresignedURL>' | sed 's/<[^>]*>//g')

    if [[ -n "$PRESIGNED_DOWNLOAD_URL" ]]; then
        run_test "test_presigned_download" \
            "curl -s '$PRESIGNED_DOWNLOAD_URL' -o '$TEMP_DIR/presigned-download.txt' && cat '$TEMP_DIR/presigned-download.txt'" \
            "$TEST_CONTENT" \
            "Test presigned URL download"
    else
        run_test "test_presigned_download" \
            "echo 'Failed to extract presigned download URL'" \
            "Failed" \
            "Test presigned URL download"
    fi

    run_test "generate_presigned_upload" \
        "PRESIGNED_UPLOAD_URL=\$(s3curl -s -X GET '$SPIDER_URL/$TEST_BUCKET/presigned-upload-test.txt?presigned&upload&duration=3600' | grep -o '<PresignedURL>[^<]*</PresignedURL>' | sed 's/<[^>]*>//g'); echo \"Generated URL: \${PRESIGNED_UPLOAD_URL:0:50}...\"" \
        "Generated URL:" \
        "Generate presigned upload URL"

    PRESIGNED_UPLOAD_URL=$(s3curl -s -X GET \
        "$SPIDER_URL/$TEST_BUCKET/presigned-upload-test.txt?presigned&upload&duration=3600" \
        | grep -o '<PresignedURL>[^<]*</PresignedURL>' | sed 's/<[^>]*>//g')

    if [[ -n "$PRESIGNED_UPLOAD_URL" ]]; then
        run_test "test_presigned_upload" \
            "echo 'Presigned upload test content' > '$TEMP_DIR/presigned-upload.txt' && curl -s -w '%{http_code}' -X PUT '$PRESIGNED_UPLOAD_URL' --data-binary '@$TEMP_DIR/presigned-upload.txt'" \
            "200" \
            "Test presigned URL upload"
    else
        run_test "test_presigned_upload" \
            "echo 'Failed to extract presigned upload URL'" \
            "Failed" \
            "Test presigned URL upload"
    fi

    if bucket_exists; then
        # No multipart cleanup: not supported by this CSP
        cleanup_all_objects
        run_test "force_empty_bucket" \
            "s3curl -s -w '%{http_code}' -X DELETE '$SPIDER_URL/$TEST_BUCKET?empty=true'" \
            "204" \
            "Force empty bucket"
    else
        log_info "Skipping force_empty_bucket: bucket does not exist"
        tr_force_empty_bucket="SKIP"
    fi

    if bucket_exists; then
        # No multipart cleanup: not supported by this CSP
        cleanup_all_objects
        run_test "force_delete_bucket" \
            "s3curl -s -w '%{http_code}' -X DELETE '$SPIDER_URL/$TEST_BUCKET?force=true'" \
            "204" \
            "Force delete bucket"
    else
        log_info "Skipping force_delete_bucket: bucket does not exist"
        tr_force_delete_bucket="SKIP"
    fi

    log_info "=== CLEANING UP ==="
    print_summary

    echo
    if [[ $fail_count -eq 0 ]]; then
        log_success "All tests completed successfully!"
        exit 0
    else
        log_error "$fail_count test(s) failed. Please check the results above."
        exit 1
    fi
}

check_server() {
    if ! curl -s "http://localhost:1024/spider/readyz" | grep -q "ready"; then
        log_error "CB-Spider server is not running at http://localhost:1024"
        log_info "Please start the server with: ./bin/start.sh"
        exit 1
    fi
}

check_awscurl() {
    if ! command -v awscurl &>/dev/null; then
        log_error "'awscurl' is not installed. Install with: pip install awscurl"
        exit 1
    fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    check_awscurl
    check_server
    main "$@"
fi
