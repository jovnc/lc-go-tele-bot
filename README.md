# Telegram LeetCode Bot (Go + Cloud Run + Terraform)

A Telegram bot that delivers LeetCode practice with daily scheduling, AI-assisted evaluation, revision history, and username allow-list gating.

## Core Features

- Random unique LeetCode questions from live LeetCode catalog (`/lc`)
- Fetches and sends full question statement in Telegram Markdown
- AI evaluation for every attempt after `/lc` with heuristic fallback
- Practice controls (`/skip`, `/exit`) for active question mode
- Daily question scheduling in SGT (`/daily_on`, `/daily_time`, `/daily_off`, `/daily_status`)
- Answer history and revision workflow (`/answered`, `/history`, `/revise`)
- Username allow-list gating via env (`ALLOWED_TELEGRAM_USERNAMES`)
- Cloud Run webhook deployment with Firestore state and Terraform IaC

## Commands

- `/lc` get a random question
- `/skip` skip active question (does not keep skipped question in seen set)
- `/exit` leave active `/lc` practice mode
- `/answered [limit]` list answered questions
- `/history [limit]` alias of `/answered`
- `/revise [slug]` revisit an answered question
- `/daily_on [HH:MM]` enable daily question
- `/daily_off` disable daily question
- `/daily_time HH:MM` set daily time and enable
- `/daily_status` show daily schedule
- `/help` list commands

After `/lc`, send your approach in plain text and the bot evaluates it (AI-first, heuristic fallback).

## Local Development

1. Configure environment.

```bash
cp .env.example .env
```

2. Authenticate ADC for Firestore.

```bash
gcloud auth application-default login
```

3. Run bot.

```bash
set -a
source .env
set +a
# Primary entrypoint (root)
go run .
# Alternative entrypoint (cmd layout)
go run ./cmd/bot
```

## Testing

```bash
go test ./internal/bot -v
```
