#!/bin/bash
# Register connection info of all 10 CSPs (in order)
# usage: RESTSERVER=<spider-host> ./all_register.sh   (default: localhost)

cd "$(dirname "$0")" || exit 1
export RESTSERVER=${RESTSERVER:-localhost}

scripts=(
  "1.aws.sh"
  "2.azure.sh"
  "3.gcp.sh"
  "4.alibaba.sh"
  "5.tencent.sh"
  "6.ibm.sh"
  "7.openstack.sh"
  "8.ncp.sh"
  "9.nhn.sh"
  "10.kt.sh"
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
echo "All scripts executed successfully"
