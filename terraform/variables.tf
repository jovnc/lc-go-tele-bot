variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region for Cloud Run and Artifact Registry"
  type        = string
  default     = "asia-southeast1"
}

variable "service_name" {
  description = "Cloud Run service name"
  type        = string
  default     = "leetcode-telegram-bot"
}

variable "artifact_repo_name" {
  description = "Artifact Registry repository name"
  type        = string
  default     = "leetcode-bot"
}

variable "container_image" {
  description = "Container image URI used by Cloud Run"
  type        = string
}

variable "telegram_bot_token" {
  description = "Telegram bot token from BotFather (seeded into Secret Manager)"
  type        = string
  sensitive   = true
}

variable "webhook_secret" {
  description = "Secret path segment for webhook endpoint (seeded into Secret Manager)"
  type        = string
  sensitive   = true
}

variable "cron_secret" {
  description = "Shared secret header value for Cloud Scheduler to cron endpoint (seeded into Secret Manager)"
  type        = string
  sensitive   = true
}

variable "openai_api_key" {
  description = "OpenAI API key for AI coach features (seeded into Secret Manager)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "telegram_bot_token_secret_id" {
  description = "Secret Manager secret ID for TELEGRAM_BOT_TOKEN"
  type        = string
  default     = "telegram-bot-token"
}

variable "webhook_secret_secret_id" {
  description = "Secret Manager secret ID for WEBHOOK_SECRET"
  type        = string
  default     = "telegram-webhook-secret"
}

variable "cron_secret_secret_id" {
  description = "Secret Manager secret ID for CRON_SECRET"
  type        = string
  default     = "telegram-cron-secret"
}

variable "openai_api_key_secret_id" {
  description = "Secret Manager secret ID for OPENAI_API_KEY"
  type        = string
  default     = "openai-api-key"
}

variable "default_daily_time" {
  description = "Default daily question time (HH:MM 24h)"
  type        = string
  default     = "20:00"
}

variable "daily_timezone" {
  description = "Default timezone for daily questions"
  type        = string
  default     = "Asia/Singapore"
}

variable "daily_scheduling_enabled" {
  description = "Enable daily scheduling commands and cron dispatch globally"
  type        = bool
  default     = false
}

variable "auto_set_webhook" {
  description = "Whether app should call Telegram setWebhook on startup"
  type        = bool
  default     = false
}

variable "bot_base_url" {
  description = "Public base URL for webhook setup when auto_set_webhook=true"
  type        = string
  default     = ""
}

variable "question_cache_sec" {
  description = "LeetCode question list cache duration in seconds"
  type        = number
  default     = 3600
}

variable "ai_enabled" {
  description = "Enable AI answer evaluation"
  type        = bool
  default     = true
}

variable "openai_model" {
  description = "OpenAI model used for answer evaluation"
  type        = string
  default     = "gpt-4o-mini"
}

variable "ai_timeout_sec" {
  description = "OpenAI request timeout in seconds"
  type        = number
  default     = 25
}

variable "allowed_telegram_usernames" {
  description = "Comma-separated Telegram usernames allowed to use the bot. Empty allows all users."
  type        = string
  default     = ""
}

variable "max_instance_count" {
  description = "Cloud Run max instances"
  type        = number
  default     = 1
}

variable "cpu_limit" {
  description = "Cloud Run CPU limit"
  type        = string
  default     = "1"
}

variable "memory_limit" {
  description = "Cloud Run memory limit"
  type        = string
  default     = "512Mi"
}

variable "allow_unauthenticated" {
  description = "Allow unauthenticated invoke so Telegram can hit webhook"
  type        = bool
  default     = true
}

variable "scheduler_cron" {
  description = "Cloud Scheduler cron expression"
  type        = string
  default     = "* * * * *"
}

variable "scheduler_timezone" {
  description = "Cloud Scheduler timezone"
  type        = string
  default     = "Asia/Singapore"
}

variable "create_firestore_database" {
  description = "Create default Firestore database. Set false if already provisioned."
  type        = bool
  default     = true
}

variable "firestore_location" {
  description = "Firestore location ID for default database, e.g. nam5"
  type        = string
  default     = "nam5"
}
