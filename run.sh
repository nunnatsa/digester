#!/usr/bin/env bash

set -e

source config

VARS=""

while IFS= read -r line; do
  if [[ $line != "" && $line != \#* ]]; then
    V="${line//=*/}"
    VARS="$VARS ${V}=${!V}"
  fi
done < config

$(VARS) ./digester test.csv
