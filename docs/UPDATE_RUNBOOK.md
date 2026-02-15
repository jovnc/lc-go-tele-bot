# Update Runbook (Terraform Deployment)

Use this guide after the bot is already deployed once with Terraform.

## Scope

This runbook covers:

- releasing new bot code
- updating runtime config or secrets
- applying Terraform infrastructure changes
- rolling back to a previous container image

## Prerequisites

1. Initial deployment is already completed.
2. `terraform/terraform.tfvars` is present and correct for your project.
3. You are authenticated in GCP:

```bash
gcloud auth login
gcloud auth application-default login
gcloud config set project "<PROJECT_ID>"
```

## Path A: Code Update (Most Common)

1. Pull latest code and run tests from repo root.

```bash
git pull
GOCACHE=$(pwd)/.gocache go test ./...
```

2. Build and push a new image tag.

```bash
PROJECT_ID="<PROJECT_ID>"
REGION="us-central1"
REPO="leetcode-bot"
SERVICE="leetcode-telegram-bot"
TAG="v2"
IMAGE="${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO}/${SERVICE}:${TAG}"

gcloud auth configure-docker "${REGION}-docker.pkg.dev"
docker build -t "$IMAGE" .
docker push "$IMAGE"
```

3. Update Terraform input to use the new image.

Edit `terraform/terraform.tfvars`:

```hcl
container_image = "<REGION>-docker.pkg.dev/<PROJECT_ID>/<REPO>/<SERVICE>:<TAG>"
```

4. Apply Terraform.

```bash
cd terraform
terraform plan -out=tfplan
terraform apply tfplan
```

5. Verify deployment.

```bash
CLOUD_RUN_URL=$(terraform output -raw cloud_run_url)
curl "${CLOUD_RUN_URL}/healthz"
```

Then verify in Telegram:

- `/help`
- `/lc`
- `/done`
- `/delete <slug>`
- `/skip`
- `/exit`
- `/daily_status`

## Path B: Runtime Config or Secret Update

Use this when changing values like `ai_enabled`, `openai_model`, `daily_timezone`, `cron_secret`, `allowed_telegram_usernames`, or API keys.

1. Edit `terraform/terraform.tfvars`.
2. Apply Terraform:

```bash
cd terraform
terraform plan -out=tfplan
terraform apply tfplan
```

3. If `webhook_secret` changed and `auto_set_webhook=false`, re-register webhook:

```bash
cd ..
./scripts/set_webhook.sh "<TELEGRAM_BOT_TOKEN>" "<CLOUD_RUN_URL>" "<WEBHOOK_SECRET>"
```

## Path C: Terraform Infrastructure Change

Use this when changing `.tf` files (IAM, scheduler, scaling, memory, region, etc.).

```bash
cd terraform
terraform fmt
terraform validate
terraform plan -out=tfplan
terraform apply tfplan
```

## Rollback

If a deployment is bad, roll back by pinning `container_image` to the previous known-good tag in `terraform/terraform.tfvars`, then apply again:

```bash
cd terraform
terraform apply
```

## Teardown

To destroy all managed resources, run from repo root:

```bash
./scripts/destroy_stack.sh
```

## Release Checklist

1. New image tag pushed successfully.
2. `container_image` updated in `terraform/terraform.tfvars`.
3. `terraform apply` completed without errors.
4. `/healthz` returns `ok`.
5. Telegram bot command flow works.
