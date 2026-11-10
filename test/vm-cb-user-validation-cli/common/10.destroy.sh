#!/bin/bash
#
# Delete all test resources in the connection
#   1. Object Storage(S3): buckets that start with ${S3_PREFIX} (force: including objects)
#   2. In parallel:
#      [cluster] (type1 only) NodeGroups => wait gone => Cluster => wait gone => extra Subnet   (cf. spiderwatch cleanup)
#      [others]  NLB, MyImage => VMs (parallel) => Disk      (do not depend on the Cluster)
#   3. after both are done: Spider Destroy API for the rest (KeyPair, SG, VPC, ...),
#      retried while some resources remain
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

MAX_ATTEMPTS=120
POLL_INTERVAL=30
CONN_BODY='{"ConnectionName": "'"${CONN_CONFIG}"'"}'

# usage: delete_with_retry <resource-path> <label> [max-tries, default 10]
#   - retries every 20s on errors such as dependency/in-use
delete_with_retry() {
	local RPATH=$1 LABEL=$2 MAX=${3:-10} i
	for (( i=1; i <= MAX; i++ )); do
		spider_req DELETE "$RPATH" "$CONN_BODY"
		if is_ok; then
			echo "[OK]   delete ${LABEL}"
			return 0
		fi
		if is_not_found; then
			echo "[SKIP] ${LABEL} not found"
			return 0
		fi
		echo "  ... retry delete ${LABEL} ($i/$MAX): HTTP ${RES_CODE} $(echo "$RES_BODY" | jq -r '.message // empty' 2>/dev/null | cut -c1-150)"
		[ $i -lt $MAX ] && sleep 20
	done
	echo "[FAIL] delete ${LABEL}"
	return 1
}

#--- 1. Object Storage(S3)
delete_s3() {
	[ "$S3_SUPPORT" = "true" ] || return 0
	echo "============== delete S3 buckets: '${S3_PREFIX}*'"
	spider_req GET "s3?ConnectionName=${CONN_CONFIG}"
	local BUCKETS=$(echo "$RES_BODY" | jq -r --arg p "$S3_PREFIX" '.Buckets.Bucket[]?.IId.NameId | select(startswith($p))' 2>/dev/null)
	if [ -z "$BUCKETS" ]; then
		echo "[SKIP] no S3 bucket with prefix '${S3_PREFIX}'"
	fi
	for B in $BUCKETS; do
		spider_req DELETE "s3/${B}?force&ConnectionName=${CONN_CONFIG}"
		print_res "delete S3 bucket: ${B}"
	done
}

#--- 2-a. Cluster (NodeGroups => Cluster => extra Subnet)
delete_cluster() {
	if [ "$CLUSTER_SUPPORT" = "true" ]; then
		echo "============== delete Cluster: '${CLUSTER_NAME}'"
		spider_req GET "cluster/${CLUSTER_NAME}?ConnectionName=${CONN_CONFIG}"
		if ! is_ok; then
			echo "[SKIP] Cluster not found: ${CLUSTER_NAME}"
		else
			# type2 (Azure, GCP, IBM, NCP, NHN): the initial NodeGroup is the cluster's system pool and
			#   cannot be deleted alone (ex: NHN "Deleting the last nodegroup is not supported").
			#   => skip NodeGroup deletion; the Cluster DELETE removes it. (same as spiderwatch cleanup)
			# type1 (AWS, Alibaba, Tencent): Cluster cannot be deleted while NodeGroups are attached.
			if [ "$NODEGROUP_TYPE" != "type2" ]; then
				NG_ACCEPTED=0 NG_TOTAL=0
				for NG in $(echo "$RES_BODY" | jq -r '.NodeGroupList[]?.IId.NameId'); do
					NG_TOTAL=$((NG_TOTAL+1))
					delete_with_retry "cluster/${CLUSTER_NAME}/nodegroup/${NG}" "NodeGroup: ${NG}" 10 && NG_ACCEPTED=$((NG_ACCEPTED+1))
				done

				# - Deleting / Updating(Alibaba, Tencent report 'removing' as Updating): in progress => keep waiting (max 1 hour)
				# - Active / Inactive for 10 min in a row: deletion is not progressing => stop waiting
				NG_STALL=0
				check_ng_gone() {
					spider_req GET "cluster/${CLUSTER_NAME}?ConnectionName=${CONN_CONFIG}"
					is_not_found && return 0
					is_ok || return 1
					local CNT=$(echo "$RES_BODY" | jq '[.NodeGroupList[]?] | length')
					echo "  remaining NodeGroups: ${CNT} ($(echo "$RES_BODY" | jq -r '[.NodeGroupList[]? | .Status] | join(",")'))"
					[ "$CNT" = "0" ] && return 0
					if [ "$(echo "$RES_BODY" | jq '[.NodeGroupList[]? | select(.Status == "Deleting" or .Status == "Updating")] | length')" = "0" ]; then
						NG_STALL=$((NG_STALL+1))
						[ $NG_STALL -ge $((600 / (POLL_INTERVAL > 0 ? POLL_INTERVAL : 1))) ] && return 2
					else
						NG_STALL=0
					fi
					return 1
				}
				if [ $NG_TOTAL -gt 0 ] && [ $NG_ACCEPTED -eq 0 ]; then
					# every NodeGroup DELETE request failed (after retries): nothing will disappear, do not wait
					echo "[WARN] NodeGroup delete requests all failed; try Cluster delete anyway"
				else
					wait_until $MAX_ATTEMPTS $POLL_INTERVAL "NodeGroups gone" check_ng_gone || \
						echo "[WARN] NodeGroups still remain; try Cluster delete anyway"
				fi
			fi

			# retry: Cluster delete can be refused while NodeGroups are still being removed
			delete_with_retry "cluster/${CLUSTER_NAME}" "Cluster: ${CLUSTER_NAME}" 10 && CLUSTER_ACCEPTED=1 || CLUSTER_ACCEPTED=0

			check_cluster_gone() {
				spider_req GET "cluster/${CLUSTER_NAME}?ConnectionName=${CONN_CONFIG}"
				is_not_found && return 0
				echo "  Cluster '${CLUSTER_NAME}' status: $(echo "$RES_BODY" | jq -r '.Status // empty' 2>/dev/null)"
				return 1
			}
			if [ $CLUSTER_ACCEPTED -eq 1 ]; then
				wait_until $MAX_ATTEMPTS $POLL_INTERVAL "Cluster gone" check_cluster_gone && \
					echo "[OK]   Cluster deleted: ${CLUSTER_NAME}"
			fi
		fi
	fi

	# extra Subnet for Cluster (after the Cluster is gone)
	if [ -n "$EXTRA_SUBNET_CIDR" ]; then
		echo "============== delete extra Subnet: '${EXTRA_SUBNET_NAME}'"
		delete_with_retry "vpc/${VPC_NAME}/subnet/${EXTRA_SUBNET_NAME}" "extra Subnet: ${EXTRA_SUBNET_NAME}" 10
	fi
}

#--- 2-b. resources independent of the Cluster: NLB, MyImage => VMs => Disk
delete_others() {
	echo "============== delete NLB / MyImage: '${NLB_NAME}', '${MYIMAGE_NAME}'"
	delete_with_retry "nlb/${NLB_NAME}" "NLB: ${NLB_NAME}" 10 &
	delete_with_retry "myimage/${MYIMAGE_NAME}" "MyImage: ${MYIMAGE_NAME}" 10 &
	wait

	echo "============== delete VMs: '${VM_NAME}-*'"
	spider_req GET "vm?ConnectionName=${CONN_CONFIG}"
	local VMS=$(echo "$RES_BODY" | jq -r --arg p "${VM_NAME}-" '.vm[]?.IId.NameId | select(startswith($p))' 2>/dev/null)
	if [ -z "$VMS" ]; then
		echo "[SKIP] no VM with prefix '${VM_NAME}-'"
	fi
	for V in $VMS; do
		delete_with_retry "vm/${V}" "VM: ${V}" 10 &
	done
	wait

	echo "============== delete Disk: '${DISK_NAME}'"
	delete_with_retry "disk/${DISK_NAME}" "Disk: ${DISK_NAME}" 10
}

delete_s3
echo -e "\n"

# run Cluster deletion and the other deletions at the same time
delete_cluster 2>&1 | sed -u 's/^/[cluster] /' &
CLUSTER_JOB=$!
delete_others 2>&1 | sed -u 's/^/[vm,etc ] /'
echo "============== waiting for Cluster deletion to finish ..."
wait $CLUSTER_JOB
echo -e "\n"

#--- 3. the rest (KeyPair, SG, VPC, ...): Spider Destroy API (retry while some resources remain)
for (( i=1; i <= 3; i++ )); do
	echo "============== Destroy all resources in '${CONN_CONFIG}' (try $i/3)"
	spider_req DELETE "destroy" "$CONN_BODY"
	echo "$RES_BODY" | jq . 2>/dev/null || echo "$RES_BODY"
	if is_ok && [ "$(echo "$RES_BODY" | jq -r '.IsAllDestroyed')" = "true" ]; then
		echo "[OK]   all resources destroyed: ${CONN_CONFIG}"
		break
	fi
	[ $i -lt 3 ] && { echo "  some resources remained; retry after 60s"; sleep 60; }
done

echo -e "\n\n"
