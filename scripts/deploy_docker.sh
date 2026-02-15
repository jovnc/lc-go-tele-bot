#!/bin/bash

PROJECT_ID="lc-telegram-bot"
REGION="asia-southeast1"
REPO="leetcode-bot"
SERVICE="leetcode-telegram-bot"
TAG="v2"
IMAGE="${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO}/${SERVICE}:${TAG}"

gcloud auth configure-docker "${REGION}-docker.pkg.dev"
docker build -t "$IMAGE" .
docker push "$IMAGE"
