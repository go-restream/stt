UNAME := $(shell uname)

ifeq ($(OS),Windows_NT)
    EXE := .exe
else
    EXE := 
endif

ifeq ($(OS),Windows_NT)
    SEP := \\
else
    SEP := /
endif

TARGET := streamASR$(EXE)

BUILD_DIR := build
DIST_DIR := dist

CONFIG_DIR := config
STATIC_DIR := static
VAD_MODEL_DIR := vad
DENOISER_MODEL_DIR := enhance
SAMPLE_DIR := samples
VERSION := $(shell cat VERSION 2>/dev/null || echo "v0.1.1")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

all: build

build:
	@echo "Building StreamASR $(VERSION) for $(UNAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-X github.com/go-restream/stt/internal/version.Version=$(VERSION) -X github.com/go-restream/stt/internal/version.BuildTime=$(BUILD_TIME) -X github.com/go-restream/stt/internal/version.GitCommit=$(GIT_COMMIT)" -o $(BUILD_DIR)$(SEP)$(TARGET) .
	@echo "Copying configuration files..."
	@cp -r $(CONFIG_DIR)$(SEP)config.yaml $(BUILD_DIR)
	@cp -r $(STATIC_DIR) $(BUILD_DIR)
	@cp -r $(SAMPLE_DIR) $(BUILD_DIR)
	@mkdir -p $(BUILD_DIR)$(SEP)model
	@cp $(VAD_MODEL_DIR)$(SEP)model$(SEP)*.onnx $(BUILD_DIR)$(SEP)model
	@cp $(DENOISER_MODEL_DIR)$(SEP)model$(SEP)*.onnx $(BUILD_DIR)$(SEP)model
	@echo "Build completed: $(BUILD_DIR)$(SEP)$(TARGET) ($(VERSION))"

run: build
	@echo "Running application..."
	@cd $(BUILD_DIR) && ./$(TARGET)


restart: 
	@echo "Restarting application..."
	@cd $(BUILD_DIR) && ./$(TARGET)

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)$(SEP)*
	@rm -rf $(DIST_DIR)
	@echo "Clean completed"

test:
	@echo "Running tests..."
	go test ./...

test-local:
	@echo "Running local CI tests..."
	@./scripts/test-local.sh test

build-local:
	@echo "Running local build..."
	@./scripts/test-local.sh build

security-local:
	@echo "Running local security scan..."
	@./scripts/test-local.sh security

docker-local:
	@echo "Running local Docker build..."
	@./scripts/test-local.sh docker

ci-local:
	@echo "Running full local CI pipeline..."
	@./scripts/test-local.sh all

act-test:
	@echo "Running act test workflow..."
	@if [ -f .act-secrets ]; then rm .act-secrets; fi
	@echo "GITHUB_TOKEN=mock-token-for-local-testing" > .act-secrets
	@act -j test --platform ubuntu-latest=nektos/act-environments-ubuntu:18.04 --secret-file .act-secrets --container-architecture linux/amd64 --bind || echo "Act workflow completed (some issues are expected in local mode)"
	@rm -f .act-secrets

act-build:
	@echo "Running act build workflow..."
	@if [ -f .act-secrets ]; then rm .act-secrets; fi
	@echo "GITHUB_TOKEN=mock-token-for-local-testing" > .act-secrets
	@act -j build --platform ubuntu-latest=nektos/act-environments-ubuntu:18.04 --secret-file .act-secrets --container-architecture linux/amd64 --bind || echo "Act workflow completed (some issues are expected in local mode)"
	@rm -f .act-secrets

install:
	@echo "Installing dependencies..."
	go mod download

package: build
	@echo "Packaging for $(UNAME)..."
	@mkdir -p $(DIST_DIR)
	@cp -r $(BUILD_DIR) $(DIST_DIR)
	@echo "Package created in $(DIST_DIR)$(SEP)$(BUILD_DIR)"

version:
	@echo "Current version: $(VERSION)"
	@echo "Build time: $(BUILD_TIME)"
	@echo "Git commit: $(GIT_COMMIT)"
	@echo "Git branch: $(GIT_BRANCH)"

version-show:
	@echo "$(VERSION)"

version-bump-patch:
	@echo "Bumping patch version..."
	$(eval NEW_VERSION := $(shell python3 -c "import sys; v='$(VERSION)'.replace('v','').split('.'); v[2]=str(int(v[2])+1); print('v'+'.'.join(v))"))
	@echo "$(NEW_VERSION)" > VERSION
	@echo "Version bumped to: $(NEW_VERSION)"

version-bump-minor:
	@echo "Bumping minor version..."
	$(eval NEW_VERSION := $(shell python3 -c "import sys; v='$(VERSION)'.replace('v','').split('.'); v[1]=str(int(v[1])+1); v[2]='0'; print('v'+'.'.join(v))"))
	@echo "$(NEW_VERSION)" > VERSION
	@echo "Version bumped to: $(NEW_VERSION)"

version-bump-major:
	@echo "Bumping major version..."
	$(eval NEW_VERSION := $(shell python3 -c "import sys; v='$(VERSION)'.replace('v','').split('.'); v[0]=str(int(v[0])+1); v[1]='0'; v[2]='0'; print('v'+'.'.join(v))"))
	@echo "$(NEW_VERSION)" > VERSION
	@echo "Version bumped to: $(NEW_VERSION)"

version-set:
	@if [ -z "$(NEW_VERSION)" ]; then echo "Usage: make version-set NEW_VERSION=v0.1.2"; exit 1; fi
	@echo "$(NEW_VERSION)" > VERSION
	@echo "Version set to: $(NEW_VERSION)"

tag:
	@echo "Creating git tag for $(VERSION)..."
	git tag -a "$(VERSION)" -m "Release $(VERSION)"
	git push origin "$(VERSION)"
	@echo "Tag $(VERSION) created and pushed"

tag-list:
	@echo "Available tags:"
	git tag --sort=-version:refname | head -10

docker-build:
	@echo "Building Docker image for $(VERSION)..."
	docker build --build-arg VERSION=$(VERSION) --build-arg BUILD_TIME=$(BUILD_TIME) --build-arg GIT_COMMIT=$(GIT_COMMIT) -t streamasr:latest -t streamasr:$(VERSION) .
	@echo "Docker image built: streamasr:latest and streamasr:$(VERSION)"

docker-build-dev:
	@echo "Building Docker development image for $(VERSION)..."
	docker build --build-arg VERSION=$(VERSION)-dev --build-arg BUILD_TIME=$(BUILD_TIME) --build-arg GIT_COMMIT=$(GIT_COMMIT) -t streamasr:dev .

docker-run:
	@echo "Running streamASR container..."
	docker run -d --name streamasr-container \
		-p 8088:8088 \
		-v $(PWD)$(SEP)$(CONFIG_DIR)$(SEP)config.yaml:/app/config/config.yaml:ro \
		-v $(PWD)$(SEP)$(BUILD_DIR)$(SEP)model:/app/model:ro \
		-v $(PWD)$(SEP)audio:/app/audio \
		-v $(PWD)$(SEP)logs:/app/logs \
		streamasr:latest

docker-stop:
	@echo "Stopping streamASR container..."
	docker stop streamasr-container || true
	docker rm streamasr-container || true

docker-logs:
	@echo "Showing container logs..."
	docker logs -f streamasr-container || docker logs -f streamasr || echo "No running container found"

docker-exec:
	@echo "Executing shell in container..."
	docker exec -it streamasr-container /bin/bash || docker exec -it streamasr /bin/bash || echo "No running container found"

docker-compose-up:
	@echo "Starting services with docker-compose..."
	docker-compose up -d

docker-compose-down:
	@echo "Stopping services with docker-compose..."
	docker-compose down

docker-compose-logs:
	@echo "Showing docker-compose logs..."
	docker-compose logs -f

docker-compose-build:
	@echo "Building with docker-compose..."
	docker-compose build

docker-clean:
	@echo "Cleaning Docker resources..."
	docker-compose down --volumes --remove-orphans || true
	docker stop streamasr-container || true
	docker rm streamasr-container || true
	docker rmi streamasr:latest streamasr:dev || true
	docker system prune -f

docker-dev: docker-build-dev docker-run

docker-deploy: docker-compose-build docker-compose-up
	@echo "Deployment completed. Service is running on http://localhost:8088"

docker-ps:
	@echo "Docker containers status:"
	docker ps -a --filter name=streamasr || docker ps -a

docker-debug:
	@echo "Starting container in debug mode..."
	docker run -it --rm --name streamasr-debug \
		-p 8088:8088 \
		-v $(PWD)$(SEP)$(CONFIG_DIR)$(SEP)config.yaml:/app/config/config.yaml:ro \
		-v $(PWD)$(SEP)$(BUILD_DIR)$(SEP)model:/app/model:ro \
		-v $(PWD)$(SEP)audio:/app/audio \
		-v $(PWD)$(SEP)logs:/app/logs \
		streamasr:dev /bin/bash

.PHONY: all build run clean test install package version version-show version-bump-patch version-bump-minor version-bump-major version-set tag tag-list docker-build docker-build-dev docker-run docker-stop docker-logs docker-exec docker-compose-up docker-compose-down docker-compose-logs docker-compose-build docker-clean docker-dev docker-deploy docker-ps docker-debug test-local build-local security-local docker-local ci-local act-test act-build