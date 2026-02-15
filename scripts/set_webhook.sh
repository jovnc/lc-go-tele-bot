#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <telegram_bot_token> <cloud_run_base_url> <webhook_secret>"
  exit 1
fi

BOT_TOKEN="$1"
BASE_URL="${2%/}"
SECRET="$3"
WEBHOOK_URL="${BASE_URL}/webhook/${SECRET}"

curl -sS "https://api.telegram.org/bot${BOT_TOKEN}/setWebhook" \
  -H "Content-Type: application/json" \
  -d "{\"url\":\"${WEBHOOK_URL}\"}" | sed 's/.*/Webhook response: &/'
