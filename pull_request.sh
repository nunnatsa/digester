#!/usr/bin/env bash

set +ex

if ! git diff --quiet --exit-code; then
  echo > a.txt
  git add txt
fi