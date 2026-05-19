#!/usr/bin/env sh

set -u

SPIDER_SERVER="${SPIDER_SERVER:-http://localhost:1024}"
API_URL="${SPIDER_SERVER}/spider/vpc"
CONNECTION_NAME="ibm-us-south-1-config"
NAME_PREFIX="powerkim-parallel"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
SETUP_ENV_FILE="${SETUP_ENV_FILE:-${SCRIPT_DIR}/../../setup.env}"

usage() {
    echo "Usage: $0 <vpc_count>"
    echo "  vpc_count: number of VPCs to delete in parallel (1-999)"
    echo "  target names: ${NAME_PREFIX}-01 .. ${NAME_PREFIX}-NN"
}

if [ "$#" -ne 1 ]; then
    usage
    exit 1
fi

VPC_COUNT="$1"
case "$VPC_COUNT" in
    ''|*[!0-9]*)
        echo "[ERROR] vpc_count must be a positive integer: $VPC_COUNT"
        exit 1
        ;;
esac

if [ "$VPC_COUNT" -lt 1 ] || [ "$VPC_COUNT" -gt 999 ]; then
    echo "[ERROR] vpc_count must be between 1 and 999: $VPC_COUNT"
    exit 1
fi

DIGITS=2
if [ "$VPC_COUNT" -ge 100 ]; then
    DIGITS=3
fi

NOW_UTC="$(date -u +%Y%m%dT%H%M%SZ)"
LOG_DIR="${PWD}/ibm-vpc-parallel-delete-logs-${NOW_UTC}"
mkdir -p "$LOG_DIR"

# Try to load auth from setup.env when one of credentials is missing.
if [ -z "${SPIDER_USERNAME:-}" ] || [ -z "${SPIDER_PASSWORD:-}" ]; then
    if [ -f "$SETUP_ENV_FILE" ]; then
        # shellcheck disable=SC1090
        . "$SETUP_ENV_FILE"
    fi
fi

if [ -z "${SPIDER_USERNAME:-}" ] || [ -z "${SPIDER_PASSWORD:-}" ]; then
    echo "[ERROR] Missing Basic Auth credentials."
    echo "        Set SPIDER_USERNAME/SPIDER_PASSWORD or provide setup.env."
    echo "        setup.env path used: $SETUP_ENV_FILE"
    exit 1
fi

echo "[INFO] Start parallel delete test"
echo "[INFO] API_URL         : $API_URL"
echo "[INFO] ConnectionName  : $CONNECTION_NAME"
echo "[INFO] Count           : $VPC_COUNT"
echo "[INFO] Logs            : $LOG_DIR"

pids=""

i=1
while [ "$i" -le "$VPC_COUNT" ]; do
    INDEX="$(printf "%0${DIGITS}d" "$i")"
    VPC_NAME="${NAME_PREFIX}-${INDEX}"

    RES_FILE="${LOG_DIR}/${VPC_NAME}.response.json"
    CODE_FILE="${LOG_DIR}/${VPC_NAME}.httpcode"

    (
                HTTP_CODE=$(curl -sS \
                    -u "${SPIDER_USERNAME}:${SPIDER_PASSWORD}" \
                    -X DELETE "${API_URL}/${VPC_NAME}?force=true" \
                    -H 'Content-Type: application/json' \
                    -d "{\"ConnectionName\":\"${CONNECTION_NAME}\"}" \
                    -o "$RES_FILE" \
                    -w '%{http_code}')

        CURL_EXIT=$?
        if [ "$CURL_EXIT" -ne 0 ]; then
            echo "curl_error_${CURL_EXIT}" > "$CODE_FILE"
            echo "[FAIL] $VPC_NAME curl exit=$CURL_EXIT"
            exit 0
        fi

        echo "$HTTP_CODE" > "$CODE_FILE"
        case "$HTTP_CODE" in
            2*)
            echo "[OK]   $VPC_NAME http=$HTTP_CODE"
            ;;
            *)
            echo "[FAIL] $VPC_NAME http=$HTTP_CODE"
            ;;
        esac
    ) &

    pids="$pids $!"
    i=$((i + 1))
done

for pid in $pids; do
    wait "$pid"
done

SUCCESS=0
FAIL=0

echo
echo "[INFO] Summary"
i=1
while [ "$i" -le "$VPC_COUNT" ]; do
    INDEX="$(printf "%0${DIGITS}d" "$i")"
    VPC_NAME="${NAME_PREFIX}-${INDEX}"
    CODE_FILE="${LOG_DIR}/${VPC_NAME}.httpcode"

    if [ ! -f "$CODE_FILE" ]; then
        echo "[FAIL] $VPC_NAME no status file"
        FAIL=$((FAIL + 1))
        continue
    fi

    HTTP_CODE="$(cat "$CODE_FILE")"
    case "$HTTP_CODE" in
        2*)
        SUCCESS=$((SUCCESS + 1))
        ;;
        *)
        FAIL=$((FAIL + 1))
        echo "[DETAIL] $VPC_NAME -> $HTTP_CODE"
        echo "         response: ${LOG_DIR}/${VPC_NAME}.response.json"
        ;;
    esac
    i=$((i + 1))
done

echo "[INFO] success=$SUCCESS fail=$FAIL total=$VPC_COUNT"

if [ "$FAIL" -gt 0 ]; then
    echo "[INFO] Check failed response files under: $LOG_DIR"
    exit 2
fi

exit 0
