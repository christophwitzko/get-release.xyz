#!/usr/bin/env bash

set -euo pipefail

if [ -z "${1:-}" ]; then
  echo "version not set"
  exit 1
fi

version=$1
image_name="gcr.io/get-release-xyz/get-release-server"
image_name_version="$image_name:$version"

echo "building image..."
docker build -t $image_name_version .

echo "pushing image..."
docker tag $image_name_version $image_name
docker push $image_name_version
docker push $image_name
