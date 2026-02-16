SHELL := /bin/bash

APP_NAME ?= lc-go-tele-bot
BUILD_DIR ?= bin
GO_MAIN ?= ./cmd/bot
GOCACHE ?= $(CURDIR)/.gocache

PROJECT_ID ?= lc-telegram-bot
REGION ?= asia-southeast1
REPO ?= leetcode-bot
SERVICE ?= leetcode-telegram-bot
TAG ?= latest
IMAGE_REPO ?= $(REGION)-docker.pkg.dev/$(PROJECT_ID)/$(REPO)/$(SERVICE)
IMAGE ?= $(IMAGE_REPO):$(TAG)

TF_DIR ?= terraform
TF_PLAN ?= tfplan
TFVARS_FILE ?= $(TF_DIR)/terraform.tfvars

.PHONY: help build dev test deploy-docker deploy tf-init tf-plan tf-set-image cloud-deploy terraform-apply

help:
	@echo "Common targets:"
	@echo "  make build                         Build binary into $(BUILD_DIR)/$(APP_NAME)"
	@echo "  make dev                           Run the bot locally (loads .env if present)"
	@echo "  make test                          Run all Go tests"
	@echo "  make deploy-docker PROJECT_ID=...  Build and push Docker image to Artifact Registry"
	@echo "  make cloud-deploy PROJECT_ID=...   Push image and apply Terraform in $(TF_DIR)"

build:
	@mkdir -p "$(BUILD_DIR)"
	go build -o "$(BUILD_DIR)/$(APP_NAME)" "$(GO_MAIN)"

dev:
	set -a; [ -f .env ] && . ./.env; set +a; go run "$(GO_MAIN)"

test:
	GOCACHE="$(GOCACHE)" go test ./...

deploy-docker:
	@[ -n "$(PROJECT_ID)" ] || (echo "PROJECT_ID is required. Example: make deploy-docker PROJECT_ID=my-gcp-project" && exit 1)
	gcloud auth configure-docker "$(REGION)-docker.pkg.dev"
	docker build -t "$(IMAGE)" .
	docker push "$(IMAGE)"
	@echo "Pushed $(IMAGE)"

deploy: deploy-docker

tf-init:
	terraform -chdir="$(TF_DIR)" init

tf-plan: tf-init
	terraform -chdir="$(TF_DIR)" plan -out="$(TF_PLAN)"

tf-set-image:
	@[ -f "$(TFVARS_FILE)" ] || (echo "$(TFVARS_FILE) not found. Create it from terraform/terraform.tfvars.example first." && exit 1)
	@DIGEST=""; \
	for i in 1 2 3 4 5; do \
		DIGEST="$$(gcloud artifacts docker images describe "$(IMAGE)" --format='get(image_summary.digest)' 2>/dev/null || true)"; \
		[ -n "$$DIGEST" ] && break; \
		sleep 2; \
	done; \
	[ -n "$$DIGEST" ] || (echo "Could not resolve digest for $(IMAGE). Run make deploy-docker first." && exit 1); \
	DEPLOY_IMAGE="$(IMAGE_REPO)@$$DIGEST"; \
	if grep -Eq '^[[:space:]]*container_image[[:space:]]*=' "$(TFVARS_FILE)"; then \
		sed -i.bak -E "s|^[[:space:]]*container_image[[:space:]]*=.*|container_image = \"$$DEPLOY_IMAGE\"|" "$(TFVARS_FILE)"; \
	else \
		printf '\ncontainer_image = "%s"\n' "$$DEPLOY_IMAGE" >> "$(TFVARS_FILE)"; \
	fi; \
	echo "Resolved deploy image: $$DEPLOY_IMAGE"
	@rm -f "$(TFVARS_FILE).bak"
	@echo "Updated container_image in $(TFVARS_FILE)"

cloud-deploy: deploy-docker tf-set-image tf-plan
	terraform -chdir="$(TF_DIR)" apply "$(TF_PLAN)"

terraform-apply: cloud-deploy
