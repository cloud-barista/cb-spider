#!/bin/bash
# Delete connection info of all 10 CSPs (in order)
# usage: RESTSERVER=<spider-host> ./all_delete.sh   (default: localhost)

cd "$(dirname "$0")" || exit 1
export RESTSERVER=${RESTSERVER:-localhost}

scripts=(
  "1.aws_delete.sh"
  "2.azure_delete.sh"
  "3.gcp_delete.sh"
  "4.alibaba_delete.sh"
  "5.tencent_delete.sh"
  "6.ibm_delete.sh"
  "7.openstack_delete.sh"
  "8.ncp_delete.sh"
  "9.nhn_delete.sh"
  "10.kt_delete.sh"
)

failed=()
for script in "${scripts[@]}"; do
  if [ -x "$script" ]; then
    echo "Executing: $script"
    ./"$script" || failed+=("$script")
  else
    echo "$script is not executable"
    failed+=("$script")
  fi
done

if [ ${#failed[@]} -ne 0 ]; then
  echo "Failed: ${failed[*]}"
  exit 1
fi
echo "All delete scripts executed successfully"
