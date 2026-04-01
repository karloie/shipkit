.PHONY: test build build-release coverage dev snapshot snapshot-real mock real tui service help docker-dev ci-build ci-test

GIT_VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_DATE       := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION_LDFLAGS := -X main.buildVersion=$(GIT_VERSION) -X main.buildCommit=$(GIT_COMMIT) -X main.buildDate=$(GIT_DATE)
RELEASE_LDFLAGS := -s -w $(VERSION_LDFLAGS)
DOCKER ?= docker
DOCKER_IMAGE ?= karloie/kompass
DOCKER_DEV_TAG ?= dev
LDFLAGS ?= $(VERSION_LDFLAGS)
SHIPKIT ?= go run ./cmd/shipkit
ARGS    ?=
COVERPKG ?= ./...
GO_RUN   = go run $(if $(strip $(LDFLAGS)),-ldflags "$(LDFLAGS)") ./cmd/kompass
GOW     ?= gow
ACT     ?= act
ACT_IMAGE ?= ghcr.io/catthehacker/ubuntu:full-latest

SNAP_DIR            ?= testdata/fixtures
SNAP_MOCK_NAMESPACE ?= petshop
SNAP_REAL_CONTEXT   ?= tool-test-01
SNAP_REAL_NAMESPACE ?= applikasjonsplattform

build: test
	go build $(if $(strip $(LDFLAGS)),-ldflags "$(LDFLAGS)") -o kompass ./cmd/kompass
	@OUT_SIZE=$$(du -hs kompass | cut -f1); OUT_PATH=$$(realpath kompass); \
	echo "\n$$OUT_PATH $(GIT_VERSION) # $(GIT_COMMIT) ~ $$OUT_SIZE"

build-release: LDFLAGS := $(RELEASE_LDFLAGS)
build-release: test
	npm run build
	go build -tags release $(if $(strip $(LDFLAGS)),-ldflags "$(LDFLAGS)") -o kompass ./cmd/kompass
	@OUT_SIZE=$$(du -hs kompass | cut -f1); OUT_PATH=$$(realpath kompass); \
	echo "\n$$OUT_PATH $(GIT_VERSION) # $(GIT_COMMIT) ~ $$OUT_SIZE"

docker-build:
	$(DOCKER) build -f Containerfile \
		--build-arg VERSION=$(GIT_VERSION) \
		--build-arg COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(GIT_DATE) \
		-t $(DOCKER_IMAGE):$(DOCKER_DEV_TAG) .

docker-run: docker-build
	$(DOCKER) run --rm --entrypoint=/bin/ash -ti $(DOCKER_IMAGE):$(DOCKER_DEV_TAG)

docker-push: docker-build
	$(DOCKER) push $(DOCKER_IMAGE):$(DOCKER_DEV_TAG)

test:
	go test -count=1 ./...

coverage: build
	@go test -count=1 ./... -coverpkg=$(COVERPKG) -coverprofile=coverage.out -covermode=atomic >/dev/null 2>&1 || true
	@echo "┌─────────────────────────────────────────────────────────┬──────────┬──────────┐"
	@echo "│ Package                                                 │  LOCAL   │  CROSS   │"
	@echo "├─────────────────────────────────────────────────────────┼──────────┼──────────┤"
	@for pkg in $$(go list ./...); do \
		local_cov=$$(go test $$pkg -cover 2>&1 | grep -o 'coverage: [0-9.]*%' | cut -d' ' -f2); \
		cross_cov=$$(awk -v p="$$pkg" 'NR>1 { split($$1, a, ":"); file=a[1]; pkg=file; sub("/[^/]+$$", "", pkg); if (pkg==p) { total += $$2; if ($$3 > 0) covered += $$2 } } END { if (total > 0) printf "%.1f%%", (covered/total)*100 }' coverage.out); \
		if [ -n "$$local_cov" ] || [ -n "$$cross_cov" ]; then \
			if [ -z "$$local_cov" ]; then local_cov="-"; fi; \
			if [ -z "$$cross_cov" ]; then cross_cov="-"; fi; \
			printf "│ %-55s │ %7s  │ %7s  │\n" $$pkg $$local_cov $$cross_cov; \
		fi; \
	done
	@echo "├─────────────────────────────────────────────────────────┼──────────┼──────────┤"
	@go tool cover -func=coverage.out | grep 'total:' | awk '{printf "│ %-55s │ %7s  │ %7s  │\n", "TOTAL", "-", $$3}'
	@echo "└─────────────────────────────────────────────────────────┴──────────┴──────────┘"

coverage-func: build
	@go test ./... -coverpkg=$(COVERPKG) -coverprofile=coverage.out -covermode=atomic >/dev/null 2>&1 || true
	@echo "┌───────────────────────────────────────────────────────────────────────────────┐"
	@echo "│ Function Coverage                                                             │"
	@echo "├────────────────────────────────────────────────────────────────────┬──────────┤"
	@go tool cover -func=coverage.out | grep -v 'total:' | awk '{printf "│ %-66s │ %7s  │\n", substr($$1":"$$2, 1, 66), $$3}'
	@echo "└────────────────────────────────────────────────────────────────────┴──────────┘"

dev:
	@echo "Starting dev server ($(GIT_VERSION) # $(GIT_COMMIT))"
	@set -e; \
	trap 'kill $$gow_pid $$vite_pid 2>/dev/null || true' INT TERM EXIT; \
	$(GOW) -e=go -e=mod -e=sum -e=tmpl -e=html -e=js -e=css run -ldflags "$(VERSION_LDFLAGS)" ./cmd/kompass --debug --mock --service $(ARGS) & gow_pid=$$!; \
	npm run dev & vite_pid=$$!; \
	wait $$gow_pid $$vite_pid

help:    ; @$(GO_RUN) --help
mock:    ; @$(GO_RUN) --mock $(ARGS)
real:    ; @$(GO_RUN) $(ARGS)
service: ; @$(GOW) -e=go -e=mod -e=sum -e=tmpl -e=html -e=js -e=css run -ldflags "$(VERSION_LDFLAGS)" ./cmd/kompass --mock --service $(ARGS)

snapshot:
	$(GO_RUN) --json --mock -n $(SNAP_MOCK_NAMESPACE) > $(SNAP_DIR)/mock.json
	$(GO_RUN)        --mock -n $(SNAP_MOCK_NAMESPACE) > $(SNAP_DIR)/mock.txt

snapshot-real:
	$(GO_RUN) --json  -c $(SNAP_REAL_CONTEXT) -n $(SNAP_REAL_NAMESPACE) > $(SNAP_DIR)/real.json
	$(GO_RUN)         -c $(SNAP_REAL_CONTEXT) -n $(SNAP_REAL_NAMESPACE) > $(SNAP_DIR)/real.txt

#
# CI targets for shipkit
#

ci-generate:
	@echo "📦 No code generation needed for shipkit"

ci-build:
	$(SHIPKIT) go-build --output=kompass --main=./cmd/kompass

ci-test:
	go test -count=1 ./...

ci-integration-test:
	@echo "🧪 No integration tests defined for shipkit"

ci-release:
	echo "📦 Building release artifacts..."
	$(SHIPKIT) install --force goreleaser
	$(SHIPKIT) docker --release
	$(SHIPKIT) goreleaser --generate --homebrew
