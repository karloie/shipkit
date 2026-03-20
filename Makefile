.PHONY: test ci-validate lint plan-release plan-rerelease plan-docker plan-all ci-build ci-test

ACT ?= act
ACT_IMAGE ?= ghcr.io/catthehacker/ubuntu:full-latest

test:
	@go test ./...

ci-build:
	@go build ./...

ci-test:
	@go test ./...

coverage:
	@go test -count=1 ./... -coverpkg=./... -coverprofile=coverage.out -covermode=atomic >/dev/null 2>&1 || true
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

lint:
	@command -v actionlint >/dev/null 2>&1 || { \
		echo "actionlint is required (install: go install github.com/rhysd/actionlint/cmd/actionlint@latest)"; \
		exit 1; \
	}
	actionlint .github/workflows/*.yml

plan-release:
	@$(ACT) -n workflow_call -W .github/workflows/release.yml -j plan \
		--input image=example/image \
		--input event_name=workflow_dispatch \
		--input bump=patch \
		--input tool_ref=main \
		-P ubuntu-latest=$(ACT_IMAGE)

plan-rerelease:
	@$(ACT) -n workflow_call -W .github/workflows/release.yml -j plan \
		--input image=example/image \
		--input mode=rerelease \
		--input tool_ref=main \
		-P ubuntu-latest=$(ACT_IMAGE)

plan-docker:
	@$(ACT) -n workflow_call -W .github/workflows/docker.yml -j docker \
		--input image=example/image \
		--input event_name=workflow_dispatch \
		--input tag=v0.1.0 \
		--input tool_ref=main \
		-P ubuntu-latest=$(ACT_IMAGE)

ci-validate: test lint plan-release plan-rerelease plan-docker
