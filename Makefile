.PHONY: test ci-validate lint plan-release plan-rerelease plan-docker plan-all ci-build ci-test ci-publish ci-summary

ACT ?= act
ACT_IMAGE ?= ghcr.io/catthehacker/ubuntu:full-latest

test:
	@go test ./...

ci-build:
	@go build ./...

ci-test:
	@go test ./...

# Example ci-publish target that uses shipkit subcommands
# The workflow will call this if it exists, allowing full control over publishing
# Note: plan.json is auto-loaded by publish commands (tag, version, image)
ci-publish:
	@shipkit publish-goreleaser --clean

# Example ci-summary target that extends the default summary
# The workflow will call this if it exists, allowing custom post-summary actions
ci-summary:
	@shipkit summary \
		-plan-file=$${SHIPKIT_PLAN_FILE} \
		-tool-ref=$${SHIPKIT_TOOL_REF} \
		-result-plan=$${SHIPKIT_RESULT_PLAN} \
		-result-build=$${SHIPKIT_RESULT_BUILD} \
		-result-tag=$${SHIPKIT_RESULT_TAG} \
		-result-update-versions=$${SHIPKIT_RESULT_UPDATE_VERSIONS} \
		-result-publish=$${SHIPKIT_RESULT_PUBLISH} \
		-use-make=false
	@echo ""
	@echo "🔥 Custom summary extension: Everything is on fire, but it's fine! 🔥"

#ci-validate: test lint plan-release plan-rerelease plan-docker
ci-validate:
	echo "Skipping ci-validate for now, as it takes too long to run. Please run individual targets instead."

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
	@$(ACT) -n workflow_call -W .github/workflows/docker-publish.yml -j publish \
		--input image=example/image \
		--input event_name=workflow_dispatch \
		--input tag=v0.1.0 \
		--input tool_ref=main \
		-P ubuntu-latest=$(ACT_IMAGE)

# Update workflow to use current HEAD commit SHA everywhere
update-workflow-ref:
	@HASH=$$(git rev-parse HEAD); \
	echo "Updating workflow references to: $$HASH"; \
	sed -i "s|uses: karloie/shipkit/.github/workflows/release.yml@[a-f0-9]*|uses: karloie/shipkit/.github/workflows/release.yml@$$HASH|g" .github/workflows/release-shipkit.yml; \
	sed -i "s|default: [a-f0-9]\{40\}|default: $$HASH|g" .github/workflows/release-shipkit.yml; \
	sed -i "s|- [a-f0-9]\{40\}|- $$HASH|g" .github/workflows/release-shipkit.yml; \
	sed -i "s||| '[a-f0-9]\{40\}'||| '$$HASH'|g" .github/workflows/release-shipkit.yml; \
	echo "✓ Updated all references to $$HASH"
