# Deployment Guide

This guide covers full deployment of the Telegram LeetCode bot to Google Cloud Run using Terraform.

## What this deployment creates

- Cloud Run service for the Go bot
- Firestore Native database (optional if not already created)
- Cloud Scheduler minute tick job for daily dispatch
- Artifact Registry Docker repository
- Secret Manager secrets and versions for bot/runtime credentials
- IAM bindings for Cloud Run service account

## Prerequisites

1. Install and authenticate tooling.

```bash
gcloud --version
terraform --version
docker --version
```

2. Ensure these are available.
- GCP project with billing enabled
- Telegram bot token from `@BotFather`
- OpenAI API key (if AI evaluation enabled)

3. Authenticate and select project.

```bash
gcloud auth login
gcloud auth application-default login
gcloud config set project "<PROJECT_ID>"
```

## Step 1: Prepare required secrets

Generate webhook/cron secrets:

```bash
openssl rand -hex 32
openssl rand -hex 32
```

You need these values:

- `TELEGRAM_BOT_TOKEN` from BotFather
- `WEBHOOK_SECRET` generated above
- `CRON_SECRET` generated above
- `OPENAI_API_KEY` from OpenAI platform (optional if `ai_enabled=false`)

## Step 2: Configure Terraform inputs

```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars` and set at least:

- `project_id`
- `region`
- `service_name`
- `artifact_repo_name`
- `container_image`
- `telegram_bot_token`
- `webhook_secret`
- `cron_secret`
- `openai_api_key`
- `ai_enabled`
- `openai_model`
- `allowed_telegram_usernames` (optional, comma-separated usernames)

Example image format:

`<region>-docker.pkg.dev/<project_id>/<artifact_repo_name>/<service_name>:v1`

## Step 3: Initialize Terraform and create bootstrap infra

```bash
terraform init
terraform apply -target=google_project_service.required -target=google_artifact_registry_repository.repo
```

This enables required APIs and creates the Docker repo needed before image push.

## Step 4: Build and push container image

Run from repository root:

```bash
PROJECT_ID="<PROJECT_ID>"
REGION="us-central1"
REPO="leetcode-bot"
SERVICE="leetcode-telegram-bot"
TAG="v1"
IMAGE="${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO}/${SERVICE}:${TAG}"

gcloud auth configure-docker "${REGION}-docker.pkg.dev"
docker build -t "$IMAGE" .
docker push "$IMAGE"
```

Update `terraform/terraform.tfvars` so `container_image` matches the pushed image.

## Step 5: Deploy full stack

```bash
cd terraform
terraform apply
```

Expected outputs include:

- `cloud_run_url`
- `webhook_url` (sensitive)
- `secret_manager_secret_ids`

## Step 6: Register Telegram webhook

If `auto_set_webhook=false`, set webhook manually:

```bash
cd ..
./scripts/set_webhook.sh "<TELEGRAM_BOT_TOKEN>" "<CLOUD_RUN_URL>" "<WEBHOOK_SECRET>"
```

If `auto_set_webhook=true`, set `bot_base_url` in Terraform and re-apply.

## Step 7: Verify deployment

1. Health check.

```bash
curl "<CLOUD_RUN_URL>/healthz"
```

2. Telegram flow check.

- Send `/help`
- Send `/lc`
- Submit an answer and confirm grade response
- Send `/skip` and confirm a different question is sent
- Send `/exit` and confirm free text is no longer graded
- Send `/answered`
- Send `/revise`

3. Scheduler check.

- In GCP Console, verify Cloud Scheduler job is successful
- Optionally test cron endpoint manually:

```bash
curl -X POST "<CLOUD_RUN_URL>/cron/daily" -H "X-Cron-Secret: <CRON_SECRET>"
```

## Step 8: Deploy updates

For repeat deployments, use the dedicated runbook: `docs/UPDATE_RUNBOOK.md`.

1. Build and push a new image tag.

```bash
TAG="v2"
IMAGE="${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO}/${SERVICE}:${TAG}"
docker build -t "$IMAGE" .
docker push "$IMAGE"
```

2. Update `container_image` in `terraform.tfvars` and apply.

```bash
cd terraform
terraform apply
```

## Step 9: Rotate secrets

1. Update secret values in `terraform.tfvars`.
2. Apply Terraform.

```bash
cd terraform
terraform apply
```

Terraform creates new Secret Manager versions and Cloud Run reads `latest`.

## Step 10: Destroy stack (optional)

```bash
./scripts/destroy_stack.sh
```

If Firestore default database already existed before this project, set `create_firestore_database=false` to avoid managing it with Terraform.

## Troubleshooting

1. Webhook not receiving updates.
- Confirm `cloud_run_url` is reachable
- Confirm webhook secret path exactly matches app config
- Re-run `set_webhook.sh`

2. `401 unauthorized` on `/cron/daily`.
- Ensure Scheduler sends correct `X-Cron-Secret`
- Ensure `CRON_SECRET` in Terraform matches app secret

3. AI evaluation not working.
- Check `AI_ENABLED=true`
- Check `OPENAI_API_KEY` is set and valid
- Check Cloud Run logs for OpenAI API errors

4. Firestore permission errors.
- Verify Cloud Run service account has `roles/datastore.user`
- Verify ADC/project alignment for local runs

5. Cloud Run apply fails with reserved env name `PORT`.
- Remove any `PORT` entry from Cloud Run container `env` in Terraform.
- Cloud Run sets `PORT` automatically; keep only `container_port = 8080`.

6. Terraform error: `Provider produced inconsistent final plan` for `google_cloud_scheduler_job.daily_tick`.
- This is a known provider edge case when Scheduler URI is derived from Cloud Run URL during first apply.
- Re-run with two phases once:
```bash
cd terraform
terraform apply -target=google_cloud_run_v2_service.bot
terraform apply
```
