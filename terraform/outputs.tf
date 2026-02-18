output "cloud_run_url" {
  description = "Cloud Run base URL"
  value       = google_cloud_run_v2_service.bot.uri
}

output "webhook_url" {
  description = "Telegram webhook URL (set this with Bot API if AUTO_SET_WEBHOOK is false)"
  value       = "${google_cloud_run_v2_service.bot.uri}/webhook/${var.webhook_secret}"
  sensitive   = true
}

output "artifact_registry_repository" {
  description = "Artifact Registry repository path"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}"
}

output "cloud_scheduler_job" {
  description = "Cloud Scheduler job name (null when cloud_scheduler_enabled is false)"
  value       = try(google_cloud_scheduler_job.daily_tick[0].name, null)
}

output "secret_manager_secret_ids" {
  description = "Secret Manager IDs used by Cloud Run"
  value = {
    telegram_bot_token = google_secret_manager_secret.telegram_bot_token.secret_id
    webhook_secret     = google_secret_manager_secret.webhook_secret.secret_id
    cron_secret        = google_secret_manager_secret.cron_secret.secret_id
    openai_api_key     = google_secret_manager_secret.openai_api_key.secret_id
  }
}
