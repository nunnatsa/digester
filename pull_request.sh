#!/usr/bin/env bash

set +ex

if ! git diff --quiet --exit-code; then
  echo "$(date +%Y-%m-%dT%H:%M:%S)" > a.txt
  git add txt
fi