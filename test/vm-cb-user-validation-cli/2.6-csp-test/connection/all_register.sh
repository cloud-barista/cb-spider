#!/bin/bash

export RESTSERVER=localhost

# List of script files to execute
scripts=(
  "1.aws.sh"
  "2.azure.sh"
  "3.gcp.sh"
  "4.alibaba.sh"
  "5.tencent.sh"
  "6.openstack.sh"
)

# Execute scripts
for script in "${scripts[@]}"; do
  if [ -x "$script" ]; then
    echo "Executing: $script"
    ./"$script"
    if [ $? -ne 0 ]; then
      echo "Error occurred while executing $script"
      exit 1
    fi
  else
    echo "$script is not executable"
    exit 1
  fi
done

echo "All scripts executed successfully"

