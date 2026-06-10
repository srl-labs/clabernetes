#!/usr/bin/env bash

set -euo pipefail

REGISTRY="${REGISTRY:-ghcr.io/srl-labs/clabernetes}"
ARCHES="${ARCHES:-amd64,arm64}"
COMMIT_HASH="${COMMIT_HASH:-$(git describe --always --abbrev=8)}"
IMAGE_TAGS="${IMAGE_TAGS:-dev-latest,${COMMIT_HASH}}"

images=(
  "${REGISTRY}/clabernetes-manager"
  "${REGISTRY}/clabernetes-launcher"
  "${REGISTRY}/clabernetes-ui"
  "${REGISTRY}/clabverter"
)

IFS=',' read -r -a tags <<< "${IMAGE_TAGS}"
IFS=',' read -r -a arches <<< "${ARCHES}"

for image in "${images[@]}"; do
  for tag in "${tags[@]}"; do
    sources=()
    for arch in "${arches[@]}"; do
      sources+=("${image}:${tag}-${arch}")
    done

    echo "--> CREATE-IMAGE-MANIFESTS: publishing ${image}:${tag} from ${sources[*]}"
    docker buildx imagetools create -t "${image}:${tag}" "${sources[@]}"
  done
done
