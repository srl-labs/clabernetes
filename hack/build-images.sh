#!/usr/bin/env bash

set -euo pipefail

REGISTRY="${REGISTRY:-ghcr.io/srl-labs/clabernetes}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
PUSH="${PUSH:-false}"
COMMIT_HASH="${COMMIT_HASH:-$(git describe --always --abbrev=8)}"
BUILD_VERSION="${BUILD_VERSION:-0.0.0-${COMMIT_HASH}}"
IMAGE_TAGS="${IMAGE_TAGS:-dev-latest,${COMMIT_HASH}}"
IMAGE_TAG_SUFFIX="${IMAGE_TAG_SUFFIX:-}"

common_args=(
  --platform "${PLATFORMS}"
  --build-arg "VERSION=${BUILD_VERSION}"
)

if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
  common_args+=(--provenance=false)
fi

if [[ "${PUSH}" == "true" ]]; then
  common_args+=(--push)
fi

build_image() {
  local name="$1"
  local image="$2"
  local dockerfile="$3"
  local context="$4"

  local args=("${common_args[@]}")

  if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
    local cache_scope_platforms="${PLATFORMS//\//-}"
    cache_scope_platforms="${cache_scope_platforms//,/-}"
    args+=(
      --cache-from "type=gha,scope=${name}-${cache_scope_platforms}"
      --cache-to "type=gha,mode=max,scope=${name}-${cache_scope_platforms}"
    )
  fi

  IFS=',' read -r -a tags <<< "${IMAGE_TAGS}"
  for tag in "${tags[@]}"; do
    args+=(-t "${image}:${tag}${IMAGE_TAG_SUFFIX}")
  done

  echo "--> BUILD-IMAGES: building ${image} for ${PLATFORMS}"
  docker buildx build "${args[@]}" -f "${dockerfile}" "${context}"
}

build_image "clabernetes-manager" "${REGISTRY}/clabernetes-manager" "build/manager.Dockerfile" "."
build_image "clabernetes-launcher" "${REGISTRY}/clabernetes-launcher" "build/launcher.Dockerfile" "."
build_image "clabernetes-ui" "${REGISTRY}/clabernetes-ui" "build/ui.Dockerfile" "ui"
build_image "clabverter" "${REGISTRY}/clabverter" "build/clabverter.Dockerfile" "."
