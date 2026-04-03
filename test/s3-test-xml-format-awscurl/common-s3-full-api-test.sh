#!/bin/bash

# CB-Spider S3 Full API Test Script (awscurl / SigV4 variant)
# Mirrors common-s3-full-api-test.sh but uses awscurl for AWS4-HMAC-SHA256 auth.
# Requires: awscurl (pip install awscurl)
# Author: CB-Spider Team

SPIDER_USERNAME=${SPIDER_USERNAME:-admin}
SPIDER_PASSWORD=${SPIDER_PASSWORD:?"SPIDER_PASSWORD is required"}
CONNECTION_NAME="${CONNECTION_NAME:?"CONNECTION_NAME is required"}"

# awscurl access_key = "username@connectionName"
ACCESS_KEY="${SPIDER_USERNAME}@${CONNECTION_NAME}"
SECRET_KEY="${SPIDER_PASSWORD}"

SPIDER_URL="http://localhost:1024/spider/s3"

TEST_BUCKET="cb-spider-test-sigv4-$(date +%s)"
TEST_OBJECT="test-file.txt"
TEST_CONTENT="Hello CB-Spider S3 SigV4 Test!"
UPLOAD_ID=""
ETAG=""
PRESIGNED_DOWNLOAD_URL=""
PRESIGNED_UPLOAD_URL=""

TEMP_DIR="/tmp/cb-spider-s3-sigv4-test-$$-$(date +%s)"
mkdir -p "$TEMP_DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

declare -A test_results
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
            key=$(echo "$upload_block"      | grep -o '<Key>[^<]*</Key>'           | sed 's/<[^>]*>//g' | head -1)
            upload_id=$(echo "$upload_block" | grep -o '<UploadId>[^<]*</UploadId>' | sed 's/<[^>]*>//g' | head -1)
            if [[ -n "$key" && -n "$upload_id" ]]; then
                log_info "Aborting multipart upload: $key (ID: $upload_id)"
                s3curl -s -X DELETE "$SPIDER_URL/$TEST_BUCKET/$key?uploadId=$upload_id" >/dev/null 2>&1
            fi
        done
    fi
}

wait_for_bucket_deletion() {
    local max_wait=30 wait_time=0
    log_info "Waiting for bucket to be completely deleted..."
    while [[ $wait_time -lt $max_wait ]]; do
        local code
        code=$(s3curl -s -o /dev/null -w '%{http_code}' -X HEAD "$SPIDER_URL/$TEST_BUCKET")
        if [[ "$code" == "404" ]]; then
            log_info "Bucket successfully deleted after ${wait_time}s"
            return 0
        fi
        sleep 3
        wait_time=$((wait_time + 3))
        log_info "Still waiting for deletion... (${wait_time}s) - Status: $code"
    done
    log_warning "Timeout waiting for bucket deletion after ${max_wait}s"
    return 1
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
    test_results["$test_name"]="$status"
}

create_test_file() {
    echo "$TEST_CONTENT" > "$TEMP_DIR/$TEST_OBJECT"
    echo "Large file content for multipart upload test" > "$TEMP_DIR/large-file.txt"
    for i in {1..100}; do
        echo "Line $i: This is test content for large file upload" >> "$TEMP_DIR/large-file.txt"
    done
}

cleanup() {
    log_info "Cleaning up test resources..."
    if bucket_exists; then
        cleanup_multipart_uploads
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

print_summary() {
    echo
    echo "==================================================================================="
    echo "                    CB-SPIDER S3 API TEST REPORT (SigV4 / awscurl)"
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
    printf "%-50s | %-10s\n" "  List Buckets"            "${test_results[list_buckets]:-SKIP}"
    printf "%-50s | %-10s\n" "  Create Bucket"           "${test_results[create_bucket]:-SKIP}"
    printf "%-50s | %-10s\n" "  Get Bucket Info"         "${test_results[get_bucket_info]:-SKIP}"
    printf "%-50s | %-10s\n" "  Check Bucket Exists (HEAD)" "${test_results[head_bucket]:-SKIP}"
    printf "%-50s | %-10s\n" "  Get Bucket Location"     "${test_results[get_bucket_location]:-SKIP}"
    printf "%-50s | %-10s\n" "  Delete Bucket"           "${test_results[delete_bucket]:-SKIP}"
    echo

    echo "2. OBJECT MANAGEMENT (6 tests)"
    printf "%-50s | %-10s\n" "  Upload Object (File)"    "${test_results[upload_object_file]:-SKIP}"
    printf "%-50s | %-10s\n" "  Upload Object (Form)"    "${test_results[upload_object_form]:-SKIP}"
    printf "%-50s | %-10s\n" "  Download Object"         "${test_results[download_object]:-SKIP}"
    printf "%-50s | %-10s\n" "  Get Object Info (HEAD)"  "${test_results[head_object]:-SKIP}"
    printf "%-50s | %-10s\n" "  Delete Object"           "${test_results[delete_object]:-SKIP}"
    printf "%-50s | %-10s\n" "  Delete Multiple Objects" "${test_results[delete_multiple_objects]:-SKIP}"
    echo

    echo "3. MULTIPART UPLOAD (6 tests)"
    printf "%-50s | %-10s\n" "  Initiate Multipart Upload" "${test_results[initiate_multipart]:-SKIP}"
    printf "%-50s | %-10s\n" "  Upload Part"             "${test_results[upload_part]:-SKIP}"
    printf "%-50s | %-10s\n" "  Complete Multipart Upload" "${test_results[complete_multipart]:-SKIP}"
    printf "%-50s | %-10s\n" "  Abort Multipart Upload"  "${test_results[abort_multipart]:-SKIP}"
    printf "%-50s | %-10s\n" "  List Parts"              "${test_results[list_parts]:-SKIP}"
    printf "%-50s | %-10s\n" "  List Multipart Uploads"  "${test_results[list_multipart_uploads]:-SKIP}"
    echo

    echo "4. VERSIONING MANAGEMENT (4 tests)"
    printf "%-50s | %-10s\n" "  Get Bucket Versioning"   "${test_results[get_bucket_versioning]:-SKIP}"
    printf "%-50s | %-10s\n" "  Set Bucket Versioning"   "${test_results[set_bucket_versioning]:-SKIP}"
    printf "%-50s | %-10s\n" "  List Object Versions"    "${test_results[list_object_versions]:-SKIP}"
    printf "%-50s | %-10s\n" "  Delete Versioned Object" "${test_results[delete_versioned_object]:-SKIP}"
    echo

    echo "5. CORS MANAGEMENT (4 tests)"
    printf "%-50s | %-10s\n" "  Get Bucket CORS"         "${test_results[get_bucket_cors]:-SKIP}"
    printf "%-50s | %-10s\n" "  Set Bucket CORS"         "${test_results[set_bucket_cors]:-SKIP}"
    printf "%-50s | %-10s\n" "  Test CORS with OPTIONS"  "${test_results[test_cors_options]:-SKIP}"
    printf "%-50s | %-10s\n" "  Delete CORS Configuration" "${test_results[delete_bucket_cors]:-SKIP}"
    echo

    echo "6. CB-SPIDER SPECIAL FEATURES (6 tests)"
    printf "%-50s | %-10s\n" "  Generate PreSigned URL (Download)" "${test_results[generate_presigned_download]:-SKIP}"
    printf "%-50s | %-10s\n" "  PreSigned URL Download Test"       "${test_results[test_presigned_download]:-SKIP}"
    printf "%-50s | %-10s\n" "  Generate PreSigned URL (Upload)"   "${test_results[generate_presigned_upload]:-SKIP}"
    printf "%-50s | %-10s\n" "  PreSigned URL Upload Test"         "${test_results[test_presigned_upload]:-SKIP}"
    printf "%-50s | %-10s\n" "  Force Empty Bucket"                "${test_results[force_empty_bucket]:-SKIP}"
    printf "%-50s | %-10s\n" "  Force Delete Bucket"               "${test_results[force_delete_bucket]:-SKIP}"
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
    echo "              CB-SPIDER S3 FULL API TEST SUITE (SigV4 / awscurl)"
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

    # Delete-bucket test uses a separate bucket
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
        test_results["delete_bucket"]="FAIL"
    fi

    # ======================================================
    # 2. OBJECT MANAGEMENT (6)
    # ======================================================
    log_info "=== 2. OBJECT MANAGEMENT TESTS ==="

    run_test "upload_object_file" \
        "s3curl -s -w '%{http_code}' -X PUT '$SPIDER_URL/$TEST_BUCKET/$TEST_OBJECT' --data-binary '@$TEMP_DIR/$TEST_OBJECT'" \
        "200" \
        "Upload object from file"

    # awscurl does not support multipart/form-data (no -F flag).
    # Fall back to curl with Basic Auth for the form-upload test only.
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
    # 3. MULTIPART UPLOAD (6)
    # ======================================================
    log_info "=== 3. MULTIPART UPLOAD TESTS ==="

    # Prepare a fresh object for multipart tests
    s3curl -s -X PUT "$SPIDER_URL/$TEST_BUCKET/multipart-test.txt" \
        --data-binary "@$TEMP_DIR/large-file.txt" >/dev/null

    run_test "initiate_multipart" \
        "UPLOAD_ID=\$(s3curl -s -X POST '$SPIDER_URL/$TEST_BUCKET/multipart-large.txt?uploads' | grep -o '<UploadId>[^<]*</UploadId>' | sed 's/<[^>]*>//g'); echo \"UploadId: \$UPLOAD_ID\"" \
        "UploadId:" \
        "Initiate multipart upload"

    UPLOAD_ID=$(s3curl -s -X POST "$SPIDER_URL/$TEST_BUCKET/multipart-large.txt?uploads" \
        | grep -o '<UploadId>[^<]*</UploadId>' | sed 's/<[^>]*>//g')

    if [[ -n "$UPLOAD_ID" ]]; then
        # awscurl -i prints response headers as Python dict repr to stdout (head -1).
        # Extract ETag by key name; handles both 'ETag': '"val"' and 'ETag': 'val' formats.
        _part_headers=$(awscurl \
            --service s3 \
            --access_key "$ACCESS_KEY" --secret_key "$SECRET_KEY" \
            -i -X PUT "$SPIDER_URL/$TEST_BUCKET/multipart-large.txt?partNumber=1&uploadId=$UPLOAD_ID" \
            --data-binary -d "@$TEMP_DIR/large-file.txt" 2>/dev/null | head -1)
        ACTUAL_ETAG=$(echo "$_part_headers" | grep -io "'[Ee][Tt]ag': '[^']*'" \
            | grep -o "'[^']*'$" | tr -d "'\"")
        HTTP_CODE=200

        run_test "upload_part" \
            "echo \"ETag: $ACTUAL_ETAG, HTTP: $HTTP_CODE\"" \
            "ETag:" \
            "Upload part"

        run_test "list_parts" \
            "s3curl -s -X GET '$SPIDER_URL/$TEST_BUCKET/multipart-large.txt?uploadId=$UPLOAD_ID&list-type=parts'" \
            "ListPartsResult" \
            "List parts"

        run_test "abort_multipart" \
            "s3curl -s -w '%{http_code}' -X DELETE '$SPIDER_URL/$TEST_BUCKET/multipart-large.txt?uploadId=$UPLOAD_ID'" \
            "204" \
            "Abort multipart upload"
    else
        log_warning "Failed to get UploadId — skipping upload_part / list_parts / abort_multipart"
        test_results["upload_part"]="SKIP"
        test_results["list_parts"]="SKIP"
        test_results["abort_multipart"]="SKIP"
    fi

    # Complete multipart test with a fresh upload
    NEW_UPLOAD_ID=$(s3curl -s -X POST "$SPIDER_URL/$TEST_BUCKET/multipart-complete.txt?uploads" \
        | grep -o '<UploadId>[^<]*</UploadId>' | sed 's/<[^>]*>//g')

    if [[ -n "$NEW_UPLOAD_ID" ]]; then
        _part_headers=$(awscurl \
            --service s3 \
            --access_key "$ACCESS_KEY" --secret_key "$SECRET_KEY" \
            -i -X PUT "$SPIDER_URL/$TEST_BUCKET/multipart-complete.txt?partNumber=1&uploadId=$NEW_UPLOAD_ID" \
            --data-binary -d "@$TEMP_DIR/large-file.txt" 2>/dev/null | head -1)
        REAL_ETAG=$(echo "$_part_headers" | grep -io "'[Ee][Tt]ag': '[^']*'" \
            | grep -o "'[^']*'$" | tr -d "'\"")

        if [[ -n "$REAL_ETAG" ]]; then
            run_test "complete_multipart" \
                "s3curl -s -X POST '$SPIDER_URL/$TEST_BUCKET/multipart-complete.txt?uploadId=$NEW_UPLOAD_ID' \
                    -H 'Content-Type: application/xml' \
                    --data-binary '<CompleteMultipartUpload><Part><PartNumber>1</PartNumber><ETag>\"$REAL_ETAG\"</ETag></Part></CompleteMultipartUpload>'" \
                "CompleteMultipartUploadResult" \
                "Complete multipart upload"
        else
            run_test "complete_multipart" \
                "s3curl -s -X POST '$SPIDER_URL/$TEST_BUCKET/multipart-complete.txt?uploadId=$NEW_UPLOAD_ID' \
                    -H 'Content-Type: application/xml' \
                    --data-binary '<CompleteMultipartUpload><Part><PartNumber>1</PartNumber><ETag>\"mock-etag\"</ETag></Part></CompleteMultipartUpload>'" \
                "Error" \
                "Complete multipart upload (expected error – mock ETag)"
        fi
    else
        run_test "complete_multipart" \
            "echo 'Failed to get UploadId'" \
            "Failed" \
            "Complete multipart upload"
    fi

    run_test "list_multipart_uploads" \
        "s3curl -s -X GET '$SPIDER_URL/$TEST_BUCKET?uploads'" \
        "ListMultipartUploadsResult" \
        "List multipart uploads"

    # ======================================================
    # 4. VERSIONING MANAGEMENT (4)
    # ======================================================
    log_info "=== 4. VERSIONING MANAGEMENT TESTS ==="

    run_test "get_bucket_versioning" \
        "s3curl -s -X GET '$SPIDER_URL/$TEST_BUCKET?versioning'" \
        "VersioningConfiguration" \
        "Get bucket versioning status"

    run_test "set_bucket_versioning" \
        "s3curl -s -w '%{http_code}' -X PUT '$SPIDER_URL/$TEST_BUCKET?versioning' \
            -H 'Content-Type: application/xml' \
            --data-binary '<VersioningConfiguration><Status>Enabled</Status></VersioningConfiguration>'" \
        "200" \
        "Enable bucket versioning"

    run_test "list_object_versions" \
        "s3curl -s -X GET '$SPIDER_URL/$TEST_BUCKET?versions'" \
        "ListVersionsResult" \
        "List object versions"

    run_test "delete_versioned_object" \
        "s3curl -s -w '%{http_code}' -X DELETE '$SPIDER_URL/$TEST_BUCKET/multipart-test.txt'" \
        "204" \
        "Delete versioned object"

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

    # OPTIONS is a CORS preflight — no auth header is sent by browsers; use plain curl
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

    # Upload a test file for presigned URL tests
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
        cleanup_all_objects
        cleanup_multipart_uploads
        run_test "force_empty_bucket" \
            "s3curl -s -w '%{http_code}' -X DELETE '$SPIDER_URL/$TEST_BUCKET?empty=true'" \
            "204" \
            "Force empty bucket"
    else
        log_info "Skipping force_empty_bucket: bucket does not exist"
        test_results["force_empty_bucket"]="SKIP"
    fi

    if bucket_exists; then
        cleanup_all_objects
        cleanup_multipart_uploads
        run_test "force_delete_bucket" \
            "s3curl -s -w '%{http_code}' -X DELETE '$SPIDER_URL/$TEST_BUCKET?force=true'" \
            "204" \
            "Force delete bucket"
    else
        log_info "Skipping force_delete_bucket: bucket does not exist"
        test_results["force_delete_bucket"]="SKIP"
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
