#!/bin/bash

files=($(find . -mindepth 2 -maxdepth 2 -type f | sort))

for file in "${files[@]}"; do
    dir=$(basename "$(dirname "$file")")
    echo "[$dir]"
    while IFS= read -r line; do
        echo "  $line"
    done < "$file"
    echo
done

