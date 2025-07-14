#!/bin/bash
set -e

mkdir -p bin

for dir in ./cmd/*; do
  if [ -d "$dir" ]; then
    name=$(basename "$dir")
    echo "Building $name..."
    go build -o "bin/$name" "$dir"
  fi
done
