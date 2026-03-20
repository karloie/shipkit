# 🚢 Shipkit

[![CI](https://github.com/karloie/shipkit/actions/workflows/ci.yml/badge.svg)](https://github.com/karloie/shipkit/actions/workflows/ci.yml)
[![Release](https://github.com/karloie/shipkit/actions/workflows/release.yml/badge.svg)](https://github.com/karloie/shipkit/actions/workflows/release.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/karloie/shipkit.svg)](https://pkg.go.dev/github.com/karloie/shipkit)
[![Go Report Card](https://goreportcard.com/badge/github.com/karloie/shipkit)](https://goreportcard.com/report/github.com/karloie/shipkit)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/karloie/shipkit)](go.mod)

<img src="https://raw.githubusercontent.com/karloie/shipkit/main/doc/vibecoded.png" width="200" alt="Vibe Coded Badge" align="right">

Reusable GitHub workflow tooling for my GitHub projects.

This repository ships a single CLI:

- `shipkit` (`cmd/shipkit`)

The tool provides subcommands:

**Primary commands:**
- `plan` - Compute release versions and validate publishing requirements
- `version` - Display version information
- `goreleaser` - Generate GoReleaser configuration from template
- `docker-hub-readme` - Upload README.md to Docker Hub repository
- `assets-delete` - Delete GitHub release assets by tag

**Internal commands (used by workflows):**
- `docker-hub-status` - Check Docker login status  
- `git-config` - Configure git user
- `git-tag` - Create annotated git tags
- `git-tag-cleanup` - Delete git tags on failures

## Use In Another Repository

Run directly from GitHub using a tagged version:

```bash
go run github.com/karloie/shipkit/cmd/shipkit@v0.1.0 plan -mode=release -bump=patch -image=owner/repo
go run github.com/karloie/shipkit/cmd/shipkit@v0.1.0 goreleaser -project=myapp -owner=myorg -description="My application"
```

Pin to a stable tag (recommended) rather than branch names.

### Plan Command (Recommended)

The `plan` subcommand computes release versions and validates publishing requirements:

```bash
# Automatic bump detection (from commits/PR labels)
shipkit plan -mode=release -image=karloie/kompass -sha=${{ github.sha }}

# Manual bump specification
shipkit plan -mode=release -bump=patch -image=karloie/kompass -sha=${{ github.sha }}
```

The plan command automatically detects your project type by checking for common build files and logs what it finds:
- **Go**: `go.mod`
- **Node**: `package.json`
- **Docker**: `Containerfile`, `Dockerfile`
- **GoReleaser**: `.goreleaser.yml`, `.goreleaser.yaml`

**Modes and Required Secrets:**

- `release` - Docker + binary release (requires: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`)
- `rerelease` - Re-publish existing tag (requires: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`, `HOMEBREW_TAP_GITHUB_TOKEN`)
- `goreleaser` - Binary-only release (requires: `HOMEBREW_TAP_GITHUB_TOKEN`)
- `docker` - Docker-only release (requires: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`)

Secrets are auto-detected by mode, or specify with `-required-secrets`.

**In GitHub Actions (release mode):**
```yaml
- name: Plan release
  id: plan
  run: >-
    go run github.com/karloie/shipkit/cmd/shipkit@v0.1.0 plan
    -mode=release
    -bump=${{ inputs.bump }}
    -image=karloie/kompass
    -sha=${{ github.sha }}
    -owner=${{ github.repository_owner }}
    -repo=${{ github.event.repository.name }}
  env:
    GITHUB_TOKEN: ${{ github.token }}
    DOCKERHUB_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}
    DOCKERHUB_TOKEN: ${{ secrets.DOCKERHUB_TOKEN }}
    HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

**For goreleaser mode:**
```yaml
- name: Plan release
  id: plan
  run: >-
    go run github.com/karloie/shipkit/cmd/shipkit@v0.1.0 plan
    -mode=goreleaser
    -bump=${{ inputs.bump }}
    -owner=${{ github.repository_owner }}
    -repo=${{ github.event.repository.name }}
  env:
    GITHUB_TOKEN: ${{ github.token }}
    HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

**Outputs:** `latest_tag`, `next_tag`, `publish`, `dryrun`, `release_tag`, `version`, `version_major_minor`, `dockerfile`, `summary_message`, `goreleaser_yml_current`, `goreleaser_yml_new`, `goreleaser_docker`

**Using plan with a known tag (docker mode):**

When the version is already known (e.g., from a parent workflow), pass `-next-tag` to skip version computation:

```yaml
- name: Docker publish
  run: >-
    go run github.com/karloie/shipkit/cmd/shipkit@v0.1.0 plan
    -mode=docker
    -next-tag=${{ needs.release.outputs.next_tag }}
    -image=karloie/kompass
    -sha=${{ github.sha }}
  env:
    GITHUB_EVENT_NAME: push
    DOCKERHUB_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}
    DOCKERHUB_TOKEN: ${{ secrets.DOCKERHUB_TOKEN }}
```

### Other Commands

- **`version`** - Computes version bumps from commits or manual input
- **`assets-delete`** - Removes release assets (useful for re-releases)

### GoReleaser Config Generation

The `goreleaser` subcommand creates a `.goreleaser.yml` file with sensible defaults:

```bash
# Basic usage
shipkit goreleaser \
  -project=myapp \
  -owner=myorg \
  -description="My application"

# Full options
shipkit goreleaser \
  -project=kompass \
  -binary=kompass \
  -owner=karloie \
  -repo=kompass \
  -description="Kubernetes resource relationship visualizer" \
  -output=.goreleaser.yml
```

Features:
- Detects `package.json` and includes npm build steps (runs before Go build for embedding)
- Detects `Containerfile.goreleaser` or `Dockerfile.goreleaser` for Docker publishing (when present, GoReleaser handles Docker builds and `docker.yml` workflow is skipped)
- Detects `CHANGELOG.md` to use manual changelog or auto-generate from GitHub
- Parameterized project name, binary name, repository details, license
- Generates config for multi-platform builds (Linux, macOS, Windows on amd64 and arm64)
- Includes Homebrew tap, deb/rpm/apk packages, Docker images, and changelog formatting

### Docker Hub README Upload

The `docker-hub-readme` subcommand uploads your README.md to Docker Hub:

```bash
# Basic usage (reads README.md from current directory)
shipkit docker-hub-readme -repo=owner/image

# Custom README path
shipkit docker-hub-readme -repo=owner/image -readme=docs/DOCKER.md

# Environment variables (recommended for CI)
export DOCKERHUB_USERNAME=myuser
export DOCKERHUB_TOKEN=mytoken
shipkit docker-hub-readme -repo=owner/image
```

**In GitHub Actions:**
```yaml
- name: Update Docker Hub README
  run: |
    go run github.com/karloie/shipkit/cmd/shipkit@main docker-hub-readme \
      -repo=${{ github.repository_owner }}/${{ github.event.repository.name }}
  env:
    DOCKERHUB_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}
    DOCKERHUB_TOKEN: ${{ secrets.DOCKERHUB_TOKEN }}
```

## Local Development

```bash
make ci-validate  # runs tests, lint, and plan dry-runs
make test
make lint
```

## Reusable Workflows

This repository also provides parameterized reusable workflows for callers:

- `.github/workflows/ci.yml` - CI validation and build/test
- `.github/workflows/release.yml` - Automated releases with GoReleaser + Docker (supports release and rerelease modes)

### Consumer Project Contract

Projects using shipkit's CI workflow must expose two Makefile targets:

| Target | Purpose |
|---|---|
| `make ci-build` | Build the project for CI (may differ from local `make build`) |
| `make ci-test` | Run tests for CI |

These are kept separate from `make build` / `make test` intentionally — local dev workflows often have different flags, side-effects, or assumptions (e.g. requiring a running database). `ci-build` and `ci-test` are the clean, repeatable CI contract.

**Minimal example:**
```makefile
ci-build:
	go build -o myapp ./cmd/myapp

ci-test:
	go test ./...
```

**Example with Node frontend (like kompass):**
```makefile
ci-build:
	npm run build
	go build -tags release -o myapp ./cmd/myapp

ci-test:
	go test -count=1 ./...
```
- `.github/workflows/docker.yml` - Docker-only publishing (called by release.yml)

### CI Workflow

`ci.yml` supports two modes:
- `validation` - Runs Make targets (default: `ci-validate`)
- `build-test` - Builds and tests with `make ci-build` and `make ci-test`

**Inputs:**
- `mode` - `validation` or `build-test` (default: validation on push/PR)
- `make_targets` - Make targets for validation mode (default: `ci-validate`)
- `go_version` - Go version override (default: auto-detect)
- `node_version` - Node version for build-test mode (default: none)
- `node_cache` - npm/yarn/pnpm cache type (default: npm)

**Example validation mode:**
```yaml
jobs:
  validate:
    uses: karloie/shipkit/.github/workflows/ci.yml@v0.1.0
    with:
      mode: validation
      make_targets: test lint
```

**Example build-test with Node.js:**
```yaml
jobs:
  build:
    uses: karloie/shipkit/.github/workflows/ci.yml@v0.1.0
    with:
      mode: build-test
      go_version: '1.25'
      node_version: '22'
```

**Note:** Build-test mode runs `npm ci` + `npm run build` when `node_version` is set, then `make ci-build` and `make ci-test`. Consumer projects must expose these targets — see [Consumer Project Contract](#consumer-project-contract) below.

### Release Workflow

`release.yml` performs automated releases using GoReleaser and optionally Docker publishing. Supports both new releases and re-releases via the `mode` parameter.

**Inputs:**
- `image` (required) - Docker image name
- `mode` (optional) - `release` (default) or `rerelease`
- `event_name` (optional) - GitHub event name (required for release mode)
- `bump` (optional) - Manual version bump for release mode
- `tool_ref` (optional) - Shipkit version (default: main)
- `go_version` (optional) - Go version (default: auto-detect)- `node_version` (optional) - Node version for frontend build in goreleaser job (default: none)
**Secrets:**
- `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN` - For Docker publishing
- `HOMEBREW_TAP_GITHUB_TOKEN` - For Homebrew tap

**Release mode example:**

```yaml
name: Release

on:
  push:
    branches: [main]
  workflow_dispatch:
    inputs:
      bump:
        type: choice
        options: [patch, minor, major]

jobs:
  release:
    uses: karloie/shipkit/.github/workflows/release.yml@v0.1.0
    with:
      image: karloie/kompass
      mode: release  # default, can be omitted
      event_name: ${{ github.event_name }}
      bump: ${{ inputs.bump }}
      tool_ref: v0.1.0
    secrets:
      DOCKERHUB_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}
      DOCKERHUB_TOKEN: ${{ secrets.DOCKERHUB_TOKEN }}
      HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

**Re-release mode example:**

```yaml
name: Re-release

on:
  workflow_dispatch:

jobs:
  rerelease:
    uses: karloie/shipkit/.github/workflows/release.yml@v0.1.0
    with:
      image: karloie/bastille
      mode: rerelease
      tool_ref: v0.1.0
    secrets:
      DOCKERHUB_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}
      DOCKERHUB_TOKEN: ${{ secrets.DOCKERHUB_TOKEN }}
      HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

**Available workflow inputs:**
- `image` (required): Docker image name (e.g., `karloie/kompass` or `karloie/bastille`)
- `mode` (optional): `release` (default) or `rerelease`
- `event_name` (required for release mode): GitHub event name (`push` or `workflow_dispatch`)
- `bump` (optional, release mode only): Manual version bump (`patch`, `minor`, `major`)
- `tool_ref` (optional): Specific shipkit version/tag (default: `main`)
- `go_version` (optional): Go version override (default: auto-detect from go.mod)
- `node_version` (optional): Node version for frontend build in goreleaser job (e.g. `22`)

### Workflow Notes

- **Docker publishing:** If `Containerfile.goreleaser` or `Dockerfile.goreleaser` exists, GoReleaser handles Docker builds and `docker.yml` is skipped. Otherwise, `release.yml` calls `docker.yml` after GoReleaser completes.
- **Node.js projects:** If `package.json` exists, GoReleaser automatically runs `npm ci` and `npm run build` in its `before.hooks` section.
- **Docker build args:** Use uppercase `BUILD_VERSION`, `BUILD_COMMIT`, `BUILD_DATE`
- **Go ldflags:** Use camelCase `buildVersion`, `buildCommit`, `buildDate`
