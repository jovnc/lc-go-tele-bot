locals {
  required_apis = [
    "artifactregistry.googleapis.com",
    "cloudscheduler.googleapis.com",
    "firestore.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
  ]

  # Service account IDs must be <= 30 chars and follow [a-z][a-z0-9-]+.
  service_account_id = substr("bot-${replace(lower(var.service_name), "/[^a-z0-9-]/", "-")}", 0, 30)
}

resource "google_project_service" "required" {
  for_each           = toset(local.required_apis)
  project            = var.project_id
  service            = each.key
  disable_on_destroy = false
}

resource "google_firestore_database" "default" {
  count       = var.create_firestore_database ? 1 : 0
  project     = var.project_id
  name        = "(default)"
  location_id = var.firestore_location
  type        = "FIRESTORE_NATIVE"

  depends_on = [google_project_service.required]
}

resource "google_artifact_registry_repository" "repo" {
  project       = var.project_id
  location      = var.region
  repository_id = var.artifact_repo_name
  format        = "DOCKER"

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret" "telegram_bot_token" {
  project   = var.project_id
  secret_id = var.telegram_bot_token_secret_id

  replication {
    auto {}
  }

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret" "webhook_secret" {
  project   = var.project_id
  secret_id = var.webhook_secret_secret_id

  replication {
    auto {}
  }

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret" "cron_secret" {
  project   = var.project_id
  secret_id = var.cron_secret_secret_id

  replication {
    auto {}
  }

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret" "openai_api_key" {
  project   = var.project_id
  secret_id = var.openai_api_key_secret_id

  replication {
    auto {}
  }

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret_version" "telegram_bot_token" {
  secret      = google_secret_manager_secret.telegram_bot_token.id
  secret_data = var.telegram_bot_token
}

resource "google_secret_manager_secret_version" "webhook_secret" {
  secret      = google_secret_manager_secret.webhook_secret.id
  secret_data = var.webhook_secret
}

resource "google_secret_manager_secret_version" "cron_secret" {
  secret      = google_secret_manager_secret.cron_secret.id
  secret_data = var.cron_secret
}

resource "google_secret_manager_secret_version" "openai_api_key" {
  secret      = google_secret_manager_secret.openai_api_key.id
  secret_data = var.openai_api_key
}

resource "google_service_account" "bot" {
  account_id   = local.service_account_id
  display_name = "Service account for ${var.service_name}"
}

resource "google_project_iam_member" "bot_firestore_access" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.bot.email}"
}

resource "google_secret_manager_secret_iam_member" "bot_telegram_token_access" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.telegram_bot_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.bot.email}"
}

resource "google_secret_manager_secret_iam_member" "bot_webhook_secret_access" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.webhook_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.bot.email}"
}

resource "google_secret_manager_secret_iam_member" "bot_cron_secret_access" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.cron_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.bot.email}"
}

resource "google_secret_manager_secret_iam_member" "bot_openai_api_key_access" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.openai_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.bot.email}"
}

resource "google_cloud_run_v2_service" "bot" {
  name                = var.service_name
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = false

  template {
    timeout         = "60s"
    service_account = google_service_account.bot.email

    scaling {
      min_instance_count = 0
      max_instance_count = var.max_instance_count
    }

    containers {
      image = var.container_image

      ports {
        container_port = 8080
      }

      env {
        name = "TELEGRAM_BOT_TOKEN"

        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.telegram_bot_token.secret_id
            version = "latest"
          }
        }
      }

      env {
        name = "WEBHOOK_SECRET"

        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.webhook_secret.secret_id
            version = "latest"
          }
        }
      }

      env {
        name = "CRON_SECRET"

        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.cron_secret.secret_id
            version = "latest"
          }
        }
      }

      env {
        name = "OPENAI_API_KEY"

        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.openai_api_key.secret_id
            version = "latest"
          }
        }
      }

      env {
        name  = "FIRESTORE_PROJECT_ID"
        value = var.project_id
      }

      env {
        name  = "DAILY_DEFAULT_TIME"
        value = var.default_daily_time
      }

      env {
        name  = "DAILY_TIMEZONE"
        value = var.daily_timezone
      }

      env {
        name  = "AUTO_SET_WEBHOOK"
        value = var.auto_set_webhook ? "true" : "false"
      }

      env {
        name  = "BOT_BASE_URL"
        value = var.bot_base_url
      }

      env {
        name  = "QUESTION_CACHE_SEC"
        value = tostring(var.question_cache_sec)
      }

      env {
        name  = "AI_ENABLED"
        value = var.ai_enabled ? "true" : "false"
      }

      env {
        name  = "OPENAI_MODEL"
        value = var.openai_model
      }

      env {
        name  = "AI_TIMEOUT_SEC"
        value = tostring(var.ai_timeout_sec)
      }

      env {
        name  = "ALLOWED_TELEGRAM_USERNAMES"
        value = var.allowed_telegram_usernames
      }

      resources {
        limits = {
          cpu    = var.cpu_limit
          memory = var.memory_limit
        }
      }
    }
  }

  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }

  depends_on = [
    google_project_service.required,
    google_project_iam_member.bot_firestore_access,
    google_firestore_database.default,
    google_secret_manager_secret_version.telegram_bot_token,
    google_secret_manager_secret_version.webhook_secret,
    google_secret_manager_secret_version.cron_secret,
    google_secret_manager_secret_version.openai_api_key,
    google_secret_manager_secret_iam_member.bot_telegram_token_access,
    google_secret_manager_secret_iam_member.bot_webhook_secret_access,
    google_secret_manager_secret_iam_member.bot_cron_secret_access,
    google_secret_manager_secret_iam_member.bot_openai_api_key_access,
  ]
}

resource "google_cloud_run_v2_service_iam_member" "public_invoker" {
  count    = var.allow_unauthenticated ? 1 : 0
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.bot.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

data "google_cloud_run_v2_service" "bot_live" {
  name     = google_cloud_run_v2_service.bot.name
  location = var.region

  depends_on = [
    google_cloud_run_v2_service.bot,
  ]
}

resource "google_cloud_scheduler_job" "daily_tick" {
  name        = "${var.service_name}-daily-tick"
  region      = var.region
  schedule    = var.scheduler_cron
  time_zone   = var.scheduler_timezone
  description = "Minute tick for per-chat daily LeetCode scheduling"

  http_target {
    http_method = "POST"
    uri         = "${data.google_cloud_run_v2_service.bot_live.uri}/cron/daily"

    headers = {
      "X-Cron-Secret" = var.cron_secret
    }
  }

  attempt_deadline = "30s"

  retry_config {
    max_retry_duration = "0s"
  }

  depends_on = [
    google_cloud_run_v2_service.bot,
    google_cloud_run_v2_service_iam_member.public_invoker,
  ]
}
