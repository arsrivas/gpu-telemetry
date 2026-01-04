# ===============================
# Project configuration
# ===============================
IMAGE_PREFIX := gpu-telemetry
KIND_CLUSTER := telemetry-cluster
HELM_RELEASE := telemetry-apps
HELM_CHART   := deployment/helm
NAMESPACE    := telemetry
COMPONENTS := api collector mq streamer

GO := go
COVERAGE_FILE := coverage.out
OS := $(shell go env GOOS)

ifeq ($(OS),windows)
	GREEN :=
	RED :=
	YELLOW :=
	NC :=
else
	GREEN := \033[0;32m
	RED := \033[0;31m
	YELLOW := \033[0;33m
	NC := \033[0m
endif

ifeq ($(OS),windows)
	SHELL := cmd
	.SHELLFLAGS := /C
else
	SHELL := /bin/bash
	.SHELLFLAGS := -eu -o pipefail -c
endif

# ===============================

define REQUIRE_TOOL_WINDOWS
	@where $(1) >NUL 2>&1 || (echo [ERROR] $(1) not found & exit 1)
endef
define REQUIRE_TOOL_UNIX
	@command -v $(1) >/dev/null 2>&1 || \
		(echo "✖ $(1) not found" && exit 1)
endef

ifeq ($(OS),windows)
	REQUIRE_TOOL = $(call REQUIRE_TOOL_WINDOWS,$(1))
else
	REQUIRE_TOOL = $(call REQUIRE_TOOL_UNIX,$(1))
endif


# ===============================
.PHONY: preflight
preflight: ## Validate environment and tools
	$(call REQUIRE_TOOL,go)
	$(call REQUIRE_TOOL,docker)
	$(call REQUIRE_TOOL,kubectl)
	$(call REQUIRE_TOOL,helm)
	$(call REQUIRE_TOOL,kind)
	$(call REQUIRE_TOOL,swag)
	@echo "$(GREEN)✔ All required tools are installed$(NC)"

# ===============================
# Help
# ===============================
.PHONY: help
help:
	@echo ""
	@echo "Common targets:"
	@echo "  make preflight           Validate tools & environment"
	@echo "  make test                Run unit tests"
	@echo "  make coverage            Run tests with coverage"
	@echo "  make swagger             Generate OpenAPI spec"
	@echo ""
	@echo "Build targets:"
	@echo "  make docker-build        Build all components"
	@echo "  make build-api           Build API service"
	@echo "  make build-collector     Build Collector service"
	@echo "  make build-mq            Build Message Queue service"
	@echo "  make build-streamer      Build Streamer service"
	@echo ""
	@echo "Kind targets:"
	@echo "  make load                Load all images into kind"
	@echo "  make load-api            Load API image into kind"
	@echo "  make load-collector      Load Collector image into kind"
	@echo "  make load-mq             Load MQ image into kind"
	@echo "  make load-streamer       Load Streamer image into kind"
	@echo ""
	@echo "Cleanup targets:"
	@echo "  make docker-clean        Remove all project images"
	@echo "  make clean-api           Remove API image"
	@echo "  make clean-collector     Remove Collector image"
	@echo "  make clean-mq            Remove MQ image"
	@echo "  make clean-streamer      Remove Streamer image"
	@echo ""
	@echo "Deployment targets:"
	@echo "  make deploy              Helm install / upgrade"
	@echo "  make undeploy            Helm uninstall"
	@echo "  make wipe-data           Delete PVCs (DANGEROUS)"
	@echo ""
	@echo "Pipelines:"
	@echo "  make all                 Test → build → load → deploy"
	@echo ""

# ===============================
# Go
# ===============================
.PHONY: test
test:
	$(GO) test ./...

.PHONY: coverage
coverage:
	$(GO) test ./... -coverprofile=$(COVERAGE_FILE) -coverpkg=./pkg/collector,./pkg/mq,./pkg/storage,./pkg/api,./pkg/streamer,./pkg/transport,./pkg/client,./pkg/util/logger
	$(GO) tool cover -func=$(COVERAGE_FILE)

.PHONY: fmt
fmt:
	@go fmt ./...

# ===============================
# Docker build
# ===============================
.PHONY: build-api build-collector build-mq build-streamer

build-api:
	docker build -t $(IMAGE_PREFIX)/api -f build/Dockerfile.api .

build-collector:
	docker build -t $(IMAGE_PREFIX)/collector -f build/Dockerfile.collector .

build-mq:
	docker build -t $(IMAGE_PREFIX)/mq -f build/Dockerfile.mq .

build-streamer:
	docker build -t $(IMAGE_PREFIX)/streamer -f build/Dockerfile.streamer .

.PHONY: docker-build
docker-build: build-api build-collector build-mq build-streamer

.PHONY: clean-api clean-collector clean-mq clean-streamer

clean-api:
	docker rmi -f $(IMAGE_PREFIX)/api

clean-collector:
	docker rmi -f $(IMAGE_PREFIX)/collector

clean-mq:
	docker rmi -f $(IMAGE_PREFIX)/mq

clean-streamer:
	docker rmi -f $(IMAGE_PREFIX)/streamer

.PHONY: docker-clean
docker-clean: clean-api clean-collector clean-mq clean-streamer

# ===============================
# Kind
# ===============================
.PHONY: kind-create
kind-create:
	kind get clusters | grep -q "^$(KIND_CLUSTER)$$" || \
	kind create cluster --name $(KIND_CLUSTER) --config deployment/kind-cluster.yaml

.PHONY: kind-delete
kind-delete:
	kind delete cluster --name $(KIND_CLUSTER)

.PHONY: load-api load-collector load-mq load-streamer
load-api:
	kind load docker-image $(IMAGE_PREFIX)/api --name $(KIND_CLUSTER)
load-collector:
	kind load docker-image $(IMAGE_PREFIX)/collector --name $(KIND_CLUSTER)
load-mq:
	kind load docker-image $(IMAGE_PREFIX)/mq --name $(KIND_CLUSTER)
load-streamer:
	kind load docker-image $(IMAGE_PREFIX)/streamer --name $(KIND_CLUSTER)
.PHONY: kind-load
kind-load: load-api load-collector load-mq load-streamer


# ===============================
# Helm
# ===============================
.PHONY: deploy
deploy:
	helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) -n $(NAMESPACE)

.PHONY: undeploy
undeploy:
	helm uninstall $(HELM_RELEASE) -n $(NAMESPACE)

# ===============================
# Swagger / OpenAPI
# ===============================
SWAG := $(GOPATH)/bin/swag

.PHONY: swagger
swagger: $(SWAG)
	$(SWAG) init -g cmd/api/main.go -o docs

$(SWAG):
	go install github.com/swaggo/swag/cmd/swag@latest

# ==============================
# Delete PVC
# ==============================
.PHONY: wipe-data
wipe-data: ## ⚠ Delete all PVC data (irreversible)
	@echo "$(RED)⚠ This will delete ALL PVCs in $(NAMESPACE). Continue? [y/N]$(NC)"
	@read ans && [ "$$ans" = "y" ]
	kubectl delete pvc -n $(NAMESPACE) --all

# ===============================
# Full pipeline
# ===============================
.PHONY: all
all: test build load deploy
