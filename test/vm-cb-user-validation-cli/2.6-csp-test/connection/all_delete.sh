#!/bin/bash

export RESTSERVER=localhost

# List of delete script files to execute
delete_scripts=(
  "1.aws_delete.sh"
  "2.azure_delete.sh"
  "3.gcp_delete.sh"
  "4.alibaba_delete.sh"
  "5.tencent_delete.sh"
  "6.openstack_delete.sh"
)

# Execute delete scripts
for script in "${delete_scripts[@]}"; do
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

echo "All delete scripts executed successfully"
