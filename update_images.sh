#!/usr/bin/env bash

set -ex

source config

while IFS= read -r line; do
  if [[ $line != "" && $line != \#* ]]; then
    V="${line//=*/}"
    export "${V}=${!V}"
  fi
done < config

./digester test.csv
