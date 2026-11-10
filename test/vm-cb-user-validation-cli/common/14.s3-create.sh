#!/bin/bash
#
# Create Object Storage(S3) bucket (deployment only; object upload/download is not tested)
#   - bucket name: "${S3_PREFIX}<yymmddHHMMSS>" (globally unique)
#   - API: PUT /s3/{bucket}?ConnectionName=xxx (no body)   cf) spiderwatch
#

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn|kt'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)   # this common/ directory (independent of CBSPIDER_ROOT)
source $SETUP_PATH/setup.env $1
source $SETUP_PATH/rest_func.sh

if [ "$S3_SUPPORT" != "true" ]; then
	echo "[SKIP] Object Storage(S3) is not supported for $1"
	exit 0
fi

# skip if a test bucket already exists
spider_req GET "s3?ConnectionName=${CONN_CONFIG}"
EXIST=$(echo "$RES_BODY" | jq -r --arg p "$S3_PREFIX" \
	'[.Buckets.Bucket[]?.IId.NameId | select(startswith($p))] | first // empty' 2>/dev/null)
if [ -n "$EXIST" ]; then
	echo "[SKIP] S3 bucket already exists: ${EXIST}"
	exit 0
fi

S3_NAME="${S3_PREFIX}$(date +%y%m%d%H%M%S)"

echo "============== before create S3 bucket: '${S3_NAME}'"
spider_req PUT "s3/${S3_NAME}?ConnectionName=${CONN_CONFIG}"
print_res "create S3 bucket: ${S3_NAME}"

# verify with the bucket list (GET /s3/{bucket} lists objects, which is a data-plane call
# and may be rejected by some CSPs, e.g. Alibaba OSS: "Not allowed using the OSS public endpoint")
spider_req GET "s3?ConnectionName=${CONN_CONFIG}"
if echo "$RES_BODY" | jq -e --arg n "$S3_NAME" '[.Buckets.Bucket[]?.IId.NameId] | index($n) != null' >/dev/null 2>&1; then
	echo "[OK]   S3 bucket listed: ${S3_NAME}"
else
	echo "[FAIL] S3 bucket not listed: ${S3_NAME}"
fi
echo "============== after create S3 bucket: '${S3_NAME}'"

echo -e "\n\n"
