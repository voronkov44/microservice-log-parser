container_runtime := $(shell which podman || which docker)
$(info using ${container_runtime})

ENV_FILE ?= .env
COMPOSE := ${container_runtime} compose --env-file $(ENV_FILE)
SERVICE ?=

# all services
.PHONY: up down clean logs ps run-tests test test-unit test-e2e-sh test-smoke test-e2e test-all lint proto tools

up: down
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

clean:
	$(COMPOSE) down -v

logs:
	@if [ -z "$(SERVICE)" ]; then \
		$(COMPOSE) logs -f; \
	else \
		case "$(SERVICE)" in \
			app|parser|repository|topology|postgres|pgadmin) \
				$(COMPOSE) logs -f "$(SERVICE)";; \
			*) \
				echo "Usage: make logs SERVICE=app|parser|repository|topology|postgres|pgadmin"; \
				exit 2;; \
		esac; \
	fi

ps:
	$(COMPOSE) ps

run-tests:
	${container_runtime} run --rm --network=host tests:latest

test: test-unit

test-unit:
	$(MAKE) -C log-services test-unit

test-e2e-sh:
	$(COMPOSE) up --build -d
	BASE_URL=$${BASE_URL:-http://localhost:8080} ./tests/e2e.sh

test-smoke:
	$(COMPOSE) up --build -d
	cd tests && BASE_URL=$${BASE_URL:-http://localhost:8080} go test -v ./...

test-e2e: test-smoke

test-all: lint test-unit test-smoke

lint:
	$(MAKE) -C log-services lint

proto:
	$(MAKE) -C log-services protobuf

tools:
	go install github.com/yoheimuta/protolint/cmd/protolint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.4.0
	@echo "checking protobuf compiler, if it fails follow guide at https://protobuf.dev/installation/"
	@which -s protoc && echo OK || exit 1
