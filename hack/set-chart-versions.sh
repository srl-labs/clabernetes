#!/bin/bash

set -e

requestedVersion=$1

if [ -z "$requestedVersion" ]; then
  echo "version not specified, can't continue...";
  exit 1
fi

sed -i.bak -E 's|version: [0-9.]+|version: '"$requestedVersion"'|g' charts/clabernetes/Chart.yaml && rm charts/clabernetes/Chart.yaml.bak
sed -i.bak -E 's|version: [0-9.]+|version: '"$requestedVersion"'|g' charts/clicker/Chart.yaml && rm charts/clicker/Chart.yaml.bak