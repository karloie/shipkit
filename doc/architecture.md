# shipkit Architecture

## Philosophy

**shipkit is a toolkit, not a framework.** It orchestrates release workflows while delegating build logic to standard tools (Make, just, task). All business logic is testable Go code, not untestable YAML/bash.

## Three-Layer Architecture

```
┌─────────────────────────────────────────┐
│ GitHub Actions (Compute Layer)         │
│ - Event triggers                        │
│ - Environment setup                     │
│ - Artifact storage                      │
│ - Secrets management                    │
└─────────────────┬───────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────┐
│ shipkit (Orchestration Layer)          │
│ - When to run (plan.go)                 │
│ - Release decisions (decide.go)         │
│ - Progress reporting (summary.go)       │
│ - Reusable operations (publish, tag)    │
└─────────────────┬───────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────┐
│ Makefile (Implementation Layer)        │
│ - How to build (ci-build)               │
│ - How to test (ci-test)                 │
│ - Project-specific logic                │
└─────────────────────────────────────────┘
```

### Layer Responsibilities

**GitHub Actions**: Provides compute, triggers, and infrastructure  
**shipkit**: Makes release decisions, orchestrates workflow, provides reusable tools  
**Makefile**: Implements project-specific build/test/verify logic

## Core Commands

### `shipkit plan`
- Detects project types (npm, go, maven, docker)
- Detects build orchestrator (Makefile, justfile, Taskfile)
- Computes release version
- Outputs decisions to `plan.json`
- **Testable**: All detection and decision logic in Go

### `shipkit build`
- Parses Makefile to understand dependencies
- Generates Mermaid visualization of build flow
- Executes `make ci-build` (or `make build` as fallback)
- Updates progress visualization during execution
- **Override**: Makefile defines how to build

### `shipkit publish`
- Checks for `make ci-publish` first (full control)
- Falls back to built-in logic (docker, goreleaser, npm)
- Can be called with `--type` for composition
- **Override**: Makefile can customize entirely or augment defaults

### `shipkit publish-goreleaser`
- Executes `goreleaser release` to publish Go binaries
- Creates GitHub releases with binaries and changelogs
- Publishes to Homebrew taps (if configured)
- Supports Docker image publishing (via goreleaser)
- **Auto-loads**: Reads `plan.json` for tag/version if present
- **Flags**: `--skip-docker` to publish binaries only, `--snapshot` for testing
- **Composable**: Can be called from Makefile ci-publish target

### `shipkit publish-docker`
- Builds multi-platform Docker images using buildx
- Pushes to Docker registry
- Updates Docker Hub README automatically
- **Auto-loads**: Reads `plan.json` for image/tag if present (flags override)
- **Composable**: Can be called from Makefile ci-publish target

### `shipkit decide`
- Validates build results
- Determines if release should proceed
- **Testable**: Pure Go logic, no YAML conditionals

### `shipkit summary`
- Checks for `make ci-summary` first (allows extension)
- Generates markdown summary with Mermaid diagrams
- Reports to GitHub Actions step summary
- **Override**: Makefile can wrap with custom messages
- **Testable**: All formatting logic in Go

## Target Naming Convention: ci- Prefix

**Problem**: Existing Makefiles have `build`, `test` targets for local dev  
**Solution**: CI uses `ci-build`, `ci-test`, `ci-publish` targets

```makefile
# CI targets (production-ready)
ci-build: ci-deps ci-test
    go build -ldflags="-s -w" -o bin/release

ci-deps:
    go mod download
    go mod verify

ci-test:
    go test -race -coverprofile=coverage.out ./...

# Local targets (fast iteration)
build:
    go build -o bin/dev

test:
    go test ./...
```

**Detection order**: `ci-build` → `build` → error

## Override Hierarchy

### Build Override
```
1. ci-build target exists → make ci-build
2. build target exists → make build  
3. No Makefile → Error (no conventions fallback for builds)
```

### Publish Override
```
1. ci-publish target exists → make ci-publish (full control)
2. No ci-publish → Error (user must define publish strategy)
```

**Example**: Full control with shipkit subcommands:
```makefile
# Publish Go binaries and Docker images separately
# Note: tag, version, image are auto-loaded from plan.json if present
ci-publish:
	@shipkit publish-goreleaser --skip-docker --clean
	@shipkit publish-docker
```

**Example**: Let goreleaser handle everything:
```makefile
# Publish Go binaries + Docker in one goreleaser run
# Auto-loads tag/version from plan.json
ci-publish:
	@shipkit publish-goreleaser --clean
```

**Example**: Explicit flags override plan.json:
```makefile
# Override with explicit flags (ignores plan.json values)
ci-publish:
	@shipkit publish-docker --image=myorg/custom --tag=v1.0.0
```

**Example**: Mixed with custom logic:
```makefile
ci-publish:
	@echo "Running pre-publish checks..."
	@./scripts/validate-release.sh
	@shipkit publish-goreleaser
	@shipkit publish-docker
	@echo "Notifying team..."
	@./scripts/notify-slack.sh
```

### Summary Override
```
1. ci-summary target exists → make ci-summary (can extend summary)
2. No ci-summary → shipkit built-in summary generation
3. Makefile can wrap: call shipkit summary + custom messages
```

**Example**: Extend summary with custom messages:
```makefile
ci-summary:
	@shipkit summary \
		-plan-file=$(SHIPKIT_PLAN_FILE) \
		-result-build=$(SHIPKIT_RESULT_BUILD) \
		-result-publish=$(SHIPKIT_RESULT_PUBLISH) \
		-use-make=false
	@echo "🔥 Server is on fire, all is ok! 🔥"
```

**Note**: Use `shipkit` (not `./shipkit`) to work both in CI (local binary) and locally (brew-installed). Add `-use-make=false` to prevent recursive calls.

## State Transfer: plan.json

The plan job generates `plan.json` containing computed values:

```json
{
  "mode": "release",
  "version": "1.2.3",
  "tag": "v1.2.3",
  "dockerimage": "owner/project",
  "has_npm": true,
  "has_go": true,
  "build_orchestrator": "make",
  "make_targets": ["ci-build", "ci-test", "ci-publish"]
}
```

**Auto-loading in publish commands:**
- `publish-goreleaser` and `publish-docker` automatically read `plan.json` if present
- Provides smart defaults (tag, version, image) without flag duplication
- **Explicit flags override plan.json values**
- Commands still work locally without plan.json (use flags)

**Hierarchy:**
1. Explicit command flags (highest priority)
2. plan.json values
3. Built-in defaults (lowest priority)

**Example - CI workflow:**
```yaml
- name: Plan
  run: shipkit plan  # Creates plan.json

- name: Publish
  run: make ci-publish  # Makefile calls shipkit commands, auto-loads plan.json
```

**Example - Local testing:**
```bash
# Works without plan.json using explicit flags
shipkit publish-docker --image=myorg/app --tag=test
```

**Uploaded as artifact**: Plan job produces, downstream jobs consume

## Visualization: Mermaid Diagrams

shipkit parses Makefile dependencies and generates live-updating Mermaid diagrams.

**User controls granularity** by how they structure their Makefile:

```makefile
# Fine-grained (shows 5 steps)
ci-build: deps generate compile test package

# Coarse-grained (shows 1 step)  
ci-build:
    npm ci && npm run build && npm test
```

**Diagrams show**:
- Build flow with dependency arrows
- Status colors (pending, running, success, failure)
- Emoji indicators per target type
- Updates during execution

## Design Principles

### 1. Testability First
- All business logic in Go with unit tests
- YAML reduced to thin orchestration
- Makefiles are testable (run locally)

### 2. No DSLs
- No shipkit.yml configuration language
- Use existing tools: Make, bash, Python, etc.
- shipkit commands, not framework

### 3. Progressive Enhancement
- Simple projects: just use shipkit defaults
- Complex projects: customize via Makefile
- Full escape hatch: ci-publish gives total control

### 4. Reusability
- shipkit commands work across projects
- Build once, use in kompass, bastille, shipkit itself
- Composable operations (can call `shipkit publish --type=docker` from Makefile)

### 5. Local/CI Parity
```bash
# Works identically locally and in CI
make ci-build
shipkit publish

# Full workflow (future)
shipkit release --dry-run
```

## Project Structure

```
shipkit/
├── cmd/shipkit/
│   ├── main.go           # Command dispatch
│   ├── plan.go           # Detection & decisions
│   ├── build.go          # Build orchestration  
│   ├── publish.go        # Publishing operations
│   ├── decide.go         # Release validation
│   ├── summary.go        # Report generation
│   ├── makefile.go       # Makefile parsing
│   ├── visualize.go      # Mermaid generation
│   └── *_test.go         # Comprehensive tests
└── .github/workflows/
    └── release.yml       # Thin YAML wrapper
```

## Usage Patterns

### Simple Project (Defaults)
```yaml
# .github/workflows/release.yml
jobs:
  build:
    steps:
      - run: shipkit build
  publish:
    steps:
      - run: shipkit publish
```

### Custom Build (Makefile)
```makefile
# Makefile
ci-build: frontend backend
    @echo "Build complete"

frontend:
    npm ci && npm run build

backend:
    go build -ldflags="$(LDFLAGS)"
```

### Custom Publish (Mixed)
```makefile
ci-publish: verify
    shipkit publish --type=docker
    ./scripts/custom-announce.sh

verify:
    docker run myapp:latest --version
```

## YAML-First Usage (Without Makefile)

**For teams that prefer keeping logic in YAML workflows:**

```yaml
# .github/workflows/release.yml
jobs:
  build-npm:
    steps:
      - uses: actions/setup-node@v4
      - run: npm ci
      - run: npm test
      - run: npm run build
  
  build-go:
    steps:
      - uses: actions/setup-go@v5
      - run: go build -ldflags="-X main.version=${{ needs.plan.outputs.version }}"
  
  publish:
    steps:
      - run: shipkit publish --type=docker
      - run: shipkit publish --type=npm
```

**What you get:**
- ✅ shipkit orchestration (plan, decide, summary)
- ✅ Reusable publish commands
- ✅ Mermaid diagrams in summary
- ❌ No local testing (can't run workflow locally)
- ❌ No build flow visualization (no Makefile to parse)
- ❌ Untestable YAML build logic

**What's still better than pure YAML:**
- shipkit's detection logic is testable Go (not bash in YAML)
- shipkit's publish commands are reusable across projects
- shipkit's summary generation is clean (not 122-line bash script)

**Recommendation**: Start YAML-first, migrate build logic to Makefile as complexity grows.

## Gradual Migration Path

```
Stage 1: Pure YAML (Where you might be now)
├── All build logic in YAML steps
├── Complex if statements and bash
└── Hard to test, can't run locally

Stage 2: YAML + shipkit commands (Easy win)
├── Use: shipkit plan, shipkit publish, shipkit summary
├── Keep: Build logic in YAML steps
└── Better: Orchestration is testable, publish is reusable

Stage 3: YAML + Makefile for build (Recommended)
├── Makefile: ci-build (testable locally)
├── YAML: calls shipkit build
└── Win: Can test builds locally, build flow visualization

Stage 4: Makefile-first (Maximum value)
└── Makefile: ci-build, ci-test, ci-publish
└── YAML: Thin wrapper calling shipkit
└── Win: Everything testable, everything local
```

**Each stage is valuable. No need to jump to stage 4.**

### What Works in Each Stage

| Feature | YAML-only | +shipkit | +Makefile | Makefile-first |
|---------|-----------|----------|-----------|----------------|
| Build locally | ❌ | ❌ | ✅ | ✅ |
| Test locally | ❌ | ❌ | ✅ | ✅ |
| Publish locally | ❌ | ⚠️ Manual | ✅ | ✅ |
| Flow visualization | ❌ | ❌ | ✅ | ✅ |
| Testable build | ❌ | ❌ | ✅ | ✅ |
| Testable orchestration | ❌ | ✅ | ✅ | ✅ |
| Reusable publish | ❌ | ✅ | ✅ | ✅ |
| Works in CI | ✅ | ✅ | ✅ | ✅ |
| Works in CLI | ❌ | ⚠️ Partial | ✅ | ✅ |

**shipkit doesn't force you to use Makefiles. It just makes them more valuable when you do.**

## Key Benefits

✅ **Testable**: Go code with unit tests, not untestable YAML  
✅ **Flexible**: Makefile gives full control when needed  
✅ **Visual**: Mermaid diagrams show build flow and progress  
✅ **Reusable**: shipkit commands work across projects  
✅ **Simple**: Works out-of-box with sensible defaults  
✅ **No DSL**: Uses standard tools (Make, bash, Go)  
✅ **Local/CI parity**: Same commands work everywhere  
✅ **Gradual adoption**: YAML-first → Makefile-first at your pace
