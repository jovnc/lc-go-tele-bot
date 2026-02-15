#!/bin/bash
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-lc-telegram-bot}"
REGION="${REGION:-asia-southeast1}"
REPO="${REPO:-leetcode-bot}"
SERVICE="${SERVICE:-leetcode-telegram-bot}"
TAG="${TAG:-latest}"
IMAGE="${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO}/${SERVICE}:${TAG}"

gcloud auth configure-docker "${REGION}-docker.pkg.dev"
docker build -t "$IMAGE" .
docker push "$IMAGE"
