#!/bin/bash

# Runs the Block-PV/PVC test of the given CSPs in parallel (all 8 when no argument)
# and prints the result as a Markdown table (also saved to ${RESULT_DIR}/results.md).
#   ./run-all-csp-block-storage-tests.sh                 # all CSPs
#   ./run-all-csp-block-storage-tests.sh aws gcp nhn     # selected CSPs
# Author: CB-Spider Team

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export RESULT_DIR="${RESULT_DIR:-/tmp/cluster_block_storage_results}"
ALL_CSPS=(aws gcp azure alibaba ibm ncp nhn tencent)
CSPS=("$@")
[[ $# -eq 0 ]] && CSPS=("${ALL_CSPS[@]}")

for c in "${CSPS[@]}"; do
    if [[ ! -x "${SCRIPT_DIR}/${c}-block-storage-test.sh" ]]; then
        echo "Unknown CSP: ${c} (available: ${ALL_CSPS[*]})" >&2
        exit 1
    fi
done

mkdir -p "${RESULT_DIR}"
for c in "${CSPS[@]}"; do rm -f "${RESULT_DIR}/result_${c}.txt"; done

# Background jobs ignore SIGINT, so Ctrl+C is forwarded as SIGTERM: each test then
# writes its result and deletes its cluster. A second Ctrl+C aborts that cleanup.
PIDS=()
forward_signal() {
    echo "Interrupted: cleaning up test clusters (Ctrl+C again to abort cleanup)"
    trap 'kill -TERM "${PIDS[@]}" 2>/dev/null; exit 130' INT TERM
    kill -TERM "${PIDS[@]}" 2>/dev/null
    wait
}
trap forward_signal INT TERM

echo "Running: ${CSPS[*]} (logs: ${RESULT_DIR}/<csp>/test.log)"
for c in "${CSPS[@]}"; do
    "${SCRIPT_DIR}/${c}-block-storage-test.sh" > /dev/null 2>&1 &
    PIDS+=($!)
done
wait

STEPS=(
  "0|테스트 Cluster 생성|VPC/SG/KeyPair → Cluster(+NodeGroup) Active → Node Ready"
  "1|Driver Enable|\`PUT /cluster/{Name}/storagedriver\` → List \`Status: Enabled\`"
  "2|CSI 드라이버 설치|CSIDriver 존재, CSINode에 등록"
  "3|StorageClass 준비|CSP 제공 SC 사용 또는 생성 (비고에 auto/created)"
  "4|PVC → PV → Bound|PV 생성, CSP 디스크 존재 (Spider ListAllDisk)"
  "5|Attach/Mount, 읽기·쓰기|쓰기·읽기 성공, Pod 재생성 후 데이터 유지"
  "6|PVC 삭제|PV 삭제, CSP 디스크 삭제 (Spider ListAllDisk)"
  "7|Driver Disable|\`Result: true\`, \`Status: Disabled\`, CSINode에서 CSI 드라이버 제거 (Alibaba·BuiltIn은 거부 시 ➖)"
  "8|Disable 후 재 Enable|\`Status: Enabled\`, CSI 드라이버 재등록"
)

{
  echo "## Block-PV/PVC 시험 결과"
  echo
  echo "표기: ✅ 성공 / ❌ 실패 / ➖ 해당 없음(거부 기대 충족) / ⏸ 미실행(선행 단계 실패)"
  echo
  header="| # | 단계 | 성공 기준 |"; sep="|---|---|---|"
  for c in "${CSPS[@]}"; do
      name=$(cut -d'|' -f1 "${RESULT_DIR}/result_${c}.txt" 2>/dev/null)
      header+=" ${name:-$c} |"; sep+="---|"
  done
  echo "${header}"; echo "${sep}"
  for s in "${STEPS[@]}"; do
      IFS='|' read -r idx title crit <<< "${s}"
      row="| ${idx} | ${title} | ${crit} |"
      for c in "${CSPS[@]}"; do
          mark=$(cut -d'|' -f$(( idx + 2 )) "${RESULT_DIR}/result_${c}.txt" 2>/dev/null)
          row+=" ${mark:-⏸} |"
      done
      echo "${row}"
  done
  echo
  echo "### 비고"
  for c in "${CSPS[@]}"; do
      line=$(cat "${RESULT_DIR}/result_${c}.txt" 2>/dev/null)
      echo "- **$(echo "${line}" | cut -d'|' -f1)**: $(echo "${line}" | cut -d'|' -f11-)"
  done
} | tee "${RESULT_DIR}/results.md"
