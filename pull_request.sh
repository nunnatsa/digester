#!/usr/bin/env bash

set +ex

git diff --name-only

if ! git diff --quiet --exit-code; then
  echo "$(date +%Y-%m-%dT%H:%M:%S)" > a.txt
  git add a.txt
fi