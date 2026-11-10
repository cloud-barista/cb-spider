#!/bin/bash
#
# Common REST helpers for CB-Spider test scripts (requires: curl, jq)
#   - source after common/setup.env (uses SPIDER_URL, SPIDER_USERNAME, SPIDER_PASSWORD, CONN_CONFIG)
#

# usage: spider_req <METHOD> <path-with-query> [json-body]
#   => sets RES_CODE (HTTP status) and RES_BODY (response body)
spider_req() {
	local METHOD=$1 RPATH=$2 BODY=$3 RESP
	if [ -n "$BODY" ]; then
		RESP=$(curl -u "$SPIDER_USERNAME:$SPIDER_PASSWORD" -s -w '\n%{http_code}' -X "$METHOD" \
			"$SPIDER_URL/$RPATH" -H 'Content-Type: application/json' -H 'Accept: application/json' -d "$BODY")
	else
		RESP=$(curl -u "$SPIDER_USERNAME:$SPIDER_PASSWORD" -s -w '\n%{http_code}' -X "$METHOD" \
			"$SPIDER_URL/$RPATH" -H 'Accept: application/json')
	fi
	RES_CODE=$(echo "$RESP" | tail -n1)
	RES_BODY=$(echo "$RESP" | sed '$d')
}

# usage: is_ok  => true when RES_CODE is 2xx
is_ok() { [[ "$RES_CODE" == 2* ]]; }

# usage: is_not_found  => true when the last response means "resource does not exist"
is_not_found() {
	[[ "$RES_CODE" == 404 ]] && return 0
	echo "$RES_BODY" | grep -qiE 'not exist|not found|does not exist|NotFound' && return 0
	return 1
}

# usage: print_res <label>
print_res() {
	if is_ok; then
		echo "[OK]   $1 (HTTP $RES_CODE)"
	else
		echo "[FAIL] $1 (HTTP $RES_CODE) $(echo "$RES_BODY" | jq -r '.message // .Message // empty' 2>/dev/null)"
	fi
	echo "$RES_BODY" | jq . 2>/dev/null || echo "$RES_BODY"
}

# usage: resolve_image_name <image-name>
#   "__prefix:<prefix>" => the lexicographically latest image whose name starts with <prefix>
#   (cf. spiderwatch resolveImageName: e.g. Alibaba rotates ubuntu images monthly)
resolve_image_name() {
	local NAME=$1 PREFIX
	if [[ "$NAME" != __prefix:* ]]; then
		echo "$NAME"
		return 0
	fi
	PREFIX=${NAME#__prefix:}
	spider_req GET "vmimage?ConnectionName=${CONN_CONFIG}"
	if ! is_ok; then
		echo "resolve_image_name: GET vmimage failed (HTTP $RES_CODE)" >&2
		return 1
	fi
	echo "$RES_BODY" | jq -r --arg p "$PREFIX" \
		'[.image[]? | (if (.Name // "") != "" then .Name else .IId.NameId end) | select(startswith($p))] | sort | last // empty'
}

# usage: wait_until <max-attempts> <interval-sec> <label> <check-function>
#   check-function returns 0 when done, 1 to keep waiting, 2 to abort (optional)
wait_until() {
	local MAX=$1 INTERVAL=$2 LABEL=$3 FN=$4 i rc
	for (( i=1; i <= MAX; i++ )); do
		$FN; rc=$?
		[ $rc -eq 0 ] && return 0
		[ $rc -eq 2 ] && return 1
		echo "  ... waiting $LABEL ($i/$MAX)"
		sleep "$INTERVAL"
	done
	echo "[FAIL] timeout waiting $LABEL"
	return 1
}
